package main

import (
	"encoding/base64"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 候选顺序是行为约定: 直连 → Gitee 镜像 → GitCode 镜像 → 加速代理。
func TestCandidates(t *testing.T) {
	cs := candidates(defaultRegistryBase)
	if len(cs) != len(proxyPrefixes)+3 {
		t.Fatalf("候选数 = %d, 期望 %d", len(cs), len(proxyPrefixes)+3)
	}
	if cs[0].base != defaultRegistryBase || cs[0].label != "直连" || !cs[0].direct {
		t.Errorf("第一个候选应为直连 %q, 实为 %q (%q)", defaultRegistryBase, cs[0].base, cs[0].label)
	}
	if cs[1].base != giteeRegistryBase || cs[1].label != "Gitee 镜像" {
		t.Errorf("第二个候选应为 Gitee 镜像 %q, 实为 %q (%q)", giteeRegistryBase, cs[1].base, cs[1].label)
	}
	if cs[2].base != gitcodeRegistryBase || cs[2].label != "GitCode 镜像" || !cs[2].api {
		t.Errorf("第三个候选应为 GitCode 镜像 %q(API 模式), 实为 %q (%q, api=%v)",
			gitcodeRegistryBase, cs[2].base, cs[2].label, cs[2].api)
	}
	for i, p := range proxyPrefixes {
		c := cs[i+3]
		if c.label != p.name+" 代理" {
			t.Errorf("候选 %d label = %q, 期望 %q", i+3, c.label, p.name+" 代理")
		}
		want := p.prefix + defaultRegistryBase
		if c.base != want {
			t.Errorf("候选 %d 基址 = %q, 期望 %q", i+3, c.base, want)
		}
	}
	// 国内镜像只在默认基址上有意义, 其它基址不该被镜像化
	if cs := candidates("https://example.com/raw/main"); len(cs) != len(proxyPrefixes)+1 {
		t.Errorf("非默认基址的候选数 = %d, 期望 %d", len(cs), len(proxyPrefixes)+1)
	}
}

func TestRegistrySource(t *testing.T) {
	t.Run("默认源自动选路", func(t *testing.T) {
		t.Setenv("XUI_REGISTRY", "")
		src := registrySource()
		if src.local != "" || len(src.cands) != len(proxyPrefixes)+3 {
			t.Fatalf("默认源应给出 %d 个候选, 实为 %d (local=%q)", len(proxyPrefixes)+3, len(src.cands), src.local)
		}
		if src.env != "" {
			t.Errorf("默认源的 env 应为空, 实为 %q", src.env)
		}
	})

	t.Run("显式基址不套镜像与代理", func(t *testing.T) {
		t.Setenv("XUI_REGISTRY", "https://raw.githubusercontent.com/twgh/xui/v1.2.3/")
		src := registrySource()
		if len(src.cands) != 1 {
			t.Fatalf("显式基址应只有 1 个候选, 实为 %d", len(src.cands))
		}
		if src.cands[0].base != "https://raw.githubusercontent.com/twgh/xui/v1.2.3" {
			t.Errorf("尾斜杠未去掉: %q", src.cands[0].base)
		}
		if src.cands[0].label != "" {
			t.Errorf("显式基址不该被标成镜像/代理: %q", src.cands[0].label)
		}
		if src.env == "" {
			t.Error("显式指定时 env 必须非空(失败不降级的判据)")
		}
	})

	t.Run("显式 GitCode API 基址", func(t *testing.T) {
		t.Setenv("XUI_REGISTRY", gitcodeRegistryBase)
		src := registrySource()
		if len(src.cands) != 1 || !src.cands[0].api {
			t.Fatalf("GitCode contents 基址应识别成 API 模式: %+v", src.cands)
		}
	})

	t.Run("本地路径", func(t *testing.T) {
		t.Setenv("XUI_REGISTRY", "testdata-local")
		src := registrySource()
		if src.local == "" || !filepath.IsAbs(src.local) {
			t.Fatalf("本地路径应转成绝对路径, 实为 %q", src.local)
		}
		if !strings.HasSuffix(src.local, "testdata-local") {
			t.Errorf("本地路径不对: %q", src.local)
		}
	})
}

// GitCode 的 contents API 返回 JSON+base64, candidate.read 要带 ?ref=main 并解码。
func TestGitCodeAPIRead(t *testing.T) {
	const content = "package ui\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/registry.json" || r.URL.Query().Get("ref") != "main" {
			t.Errorf("请求 = %s, 期望 /registry.json?ref=main", r.URL.String())
		}
		w.Write([]byte(`{"encoding":"base64","content":"` + base64.StdEncoding.EncodeToString([]byte(content)) + `"}`))
	}))
	defer srv.Close()

	got, err := (candidate{base: srv.URL, api: true}).read("registry.json", fetchTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("解码结果 = %q, 期望 %q", got, content)
	}
}

// 选定源中途失败(GitCode 匿名限速这类)时按候选顺序换源, 成功后粘住新源。
func TestSourceReadFallsBackToNextCandidate(t *testing.T) {
	const body = "package ui\n// button\n"
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/components/button.go" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(body))
	}))
	defer good.Close()

	src := &source{sel: candidate{base: bad.URL}, rest: []candidate{{label: "备用", base: good.URL}}}
	got, err := src.read("components/button.go")
	if err != nil {
		t.Fatalf("换源后应成功: %v", err)
	}
	if string(got) != body {
		t.Errorf("内容 = %q, 期望 %q", got, body)
	}
	if src.sel.base != good.URL || len(src.rest) != 0 {
		t.Errorf("应粘住备用源, 实为 sel=%q rest=%v", src.sel.base, src.rest)
	}

	// 全都不通才报错(显式源只有一个候选, 直接走这条路)
	dead := &source{sel: candidate{base: bad.URL}}
	if _, err := dead.read("components/button.go"); err == nil {
		t.Error("没有备用候选时应报错")
	}
}

func TestParseRegistry(t *testing.T) {
	if _, err := parseRegistry("x", []byte(`{"components":{"button":{"files":["button.go"]}}}`)); err != nil {
		t.Errorf("合法清单解析失败: %v", err)
	}
	if _, err := parseRegistry("x", []byte(`<html>代理劫持页</html>`)); err == nil {
		t.Error("非 JSON 应报错")
	}
	if _, err := parseRegistry("x", []byte(`{"components":{}}`)); err == nil {
		t.Error("空组件表应报错(视为不可用候选)")
	}
}

// 共享助手文件不能把没装过的组件判成"已安装": 只装 select 时,
// popup.go / scrollarea.go / scrollbar.go 都在, 但它们分别也是 date-picker /
// dialog / table / tree / combobox 的文件 —— update 无参数不应因此全量同步。
func TestInstalledComponentsRequiresAllFiles(t *testing.T) {
	reg := &registry{Components: map[string]entry{
		"select":      {Files: []string{"select.go", "popup.go", "scrollarea.go", "scrollbar.go"}},
		"popover":     {Files: []string{"popover.go", "popup.go"}},
		"hover-card":  {Files: []string{"hovercard.go", "popup.go"}},
		"dialog":      {Files: []string{"dialog.go", "scrollarea.go", "scrollbar.go"}},
		"table":       {Files: []string{"table.go", "scrollbar.go"}},
		"calendar":    {Files: []string{"calendar.go"}},
		"scroll-area": {Files: []string{"scrollarea.go", "scrollbar.go"}},
	}}
	dir := t.TempDir()
	for _, f := range []string{"select.go", "popup.go", "scrollarea.go", "scrollbar.go"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("package ui\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := installedComponents(reg, dir)
	want := "scroll-area,select"
	if strings.Join(got, ",") != want {
		t.Errorf("installedComponents = %v, 期望 %s", got, want)
	}
}

func TestInitNewProjectIntoEmptyCwd(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot()) // 本地源: 不访问网络
	top := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	// 场景 1: 在空的 sitless 目录里执行 xui init -n sitless, 不应生成 sitless/sitless。
	named := filepath.Join(top, "sitless")
	if err := os.MkdirAll(named, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(named); err != nil {
		t.Fatal(err)
	}
	if err := initNewProject("sitless", false, true); err != nil {
		t.Fatalf("initNewProject 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(named, "go.mod")); err != nil {
		t.Errorf("骨架应铺在当前目录: %v", err)
	}
	if _, err := os.Stat(filepath.Join(named, "sitless")); err == nil {
		t.Error("不应再建一层 sitless/ 目录")
	}

	// 场景 2: xui init -n . 直接铺在当前空目录, 项目名取目录名。
	dot := filepath.Join(top, "myapp")
	if err := os.MkdirAll(dot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dot); err != nil {
		t.Fatal(err)
	}
	if err := initNewProject(".", false, true); err != nil {
		t.Fatalf("initNewProject(.) 失败: %v", err)
	}
	gomod, err := os.ReadFile(filepath.Join(dot, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gomod), "module myapp") {
		t.Errorf("go.mod 模块名应为目录名 myapp:\n%s", gomod)
	}
}

// 端到端: 用 httptest 冒充远程源, 走完"拉 registry → 解析依赖 → 拷文件"。
func TestAddViaRemoteSource(t *testing.T) {
	files := map[string]string{
		"/registry.json": `{"components":{
			"button":{"files":["button.go"],"deps":[]},
			"dialog":{"files":["dialog.go","backdrop.go"],"deps":["button"]}}}`,
		"/components/button.go":   "package ui\n// button\n",
		"/components/dialog.go":   "package ui\n// dialog\n",
		"/components/backdrop.go": "package ui\n// backdrop\n",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(body))
	}))
	defer srv.Close()

	t.Setenv("XUI_REGISTRY", srv.URL)
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	add(componentOptions{names: []string{"button"}, uiDir: "ui"}) // button 单独装
	add(componentOptions{names: []string{"dialog"}, uiDir: "ui"}) // dialog + 依赖 + 共享的 backdrop.go

	for name, want := range map[string]string{
		"button.go":   "package ui\n// button\n",
		"dialog.go":   "package ui\n// dialog\n",
		"backdrop.go": "package ui\n// backdrop\n",
	} {
		data, err := os.ReadFile(filepath.Join(dir, "ui", name))
		if err != nil {
			t.Fatalf("缺少 %s: %v", name, err)
		}
		if string(data) != want {
			t.Errorf("%s 内容 = %q, 期望 %q", name, data, want)
		}
	}

	// update: 用户改过的文件默认不覆盖, --force 才覆盖
	edited := filepath.Join(dir, "ui", "button.go")
	if err := os.WriteFile(edited, []byte("package ui\n// mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	update(componentOptions{names: []string{"button"}, uiDir: "ui"})
	if data, _ := os.ReadFile(edited); string(data) != "package ui\n// mine\n" {
		t.Errorf("update 默认不该覆盖本地修改, 实为 %q", data)
	}
	update(componentOptions{names: []string{"button"}, uiDir: "ui", overwrite: true})
	if data, _ := os.ReadFile(edited); string(data) != "package ui\n// button\n" {
		t.Errorf("update --force 应覆盖, 实为 %q", data)
	}
}

// 显式源不可达时必须报错退出(不降级)。用子进程跑, 因为报错走的是 fatal/os.Exit。
func TestExplicitSourceFailureExits(t *testing.T) {
	if os.Getenv("XUI_TEST_SUBPROCESS") == "1" {
		os.Setenv("XUI_REGISTRY", "http://127.0.0.1:1") // 必然连不上
		loadRegistry()
		os.Exit(0) // 走到这里说明没有报错退出 = 失败
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=TestExplicitSourceFailureExits", "-test.v")
	cmd.Env = append(os.Environ(), "XUI_TEST_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("显式源不可达应非零退出, 输出: %s", out)
	}
	if !strings.Contains(string(out), "拉取 registry 失败") {
		t.Errorf("报错文案不对: %s", out)
	}
}

func TestParseInitArgs(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    initOptions
		wantErr bool
	}{
		{"默认 ui", nil, initOptions{uiDir: "ui"}, false},
		{"旧位置参数", []string{"components"}, initOptions{uiDir: "components"}, false},
		{"-n", []string{"-n", "myapp"}, initOptions{project: "myapp"}, false},
		{"--name=", []string{"--name=myapp"}, initOptions{project: "myapp"}, false},
		{"-n 加 force", []string{"-f", "-n", "myapp"}, initOptions{project: "myapp", force: true}, false},
		{"--no-install", []string{"--no-install", "-n", "myapp"}, initOptions{project: "myapp", noInstall: true}, false},
		{"--no-install 缺 -n", []string{"--no-install"}, initOptions{}, true},
		{"-n 缺值", []string{"-n"}, initOptions{}, true},
		{"-n= 空值", []string{"-n="}, initOptions{}, true},
		{"-n 与位置参数冲突", []string{"-n", "myapp", "ui"}, initOptions{}, true},
		{"多个位置参数", []string{"a", "b"}, initOptions{}, true},
		{"未知选项", []string{"--wat"}, initOptions{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseInitArgs(c.args)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if !c.wantErr && got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestModuleNameFromProject(t *testing.T) {
	for name, want := range map[string]string{
		"myapp":   "myapp",
		"my-app":  "my-app",
		"my app":  "my-app",
		"我的应用":    "app",
		".hidden": "hidden",
		"app_1":   "app_1",
		"--":      "app",
		"a.b-c_d": "a.b-c_d",
	} {
		if got := moduleNameFromProject(name); got != want {
			t.Errorf("moduleNameFromProject(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestInitNewProject(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot()) // 本地源: 不访问网络
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	if err := initNewProject("myapp", false, true); err != nil {
		t.Fatalf("initNewProject 失败: %v", err)
	}

	for _, f := range []string{
		"go.mod", "main.go", "README.md", ".gitignore",
		"ui/theme.go", "ui/DESIGN.md", "ui/button.go", "ui/window.go",
	} {
		if _, err := os.Stat(filepath.Join(dir, "myapp", f)); err != nil {
			t.Errorf("缺少 %s: %v", f, err)
		}
	}

	gomod, err := os.ReadFile(filepath.Join(dir, "myapp", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"module myapp", "github.com/twgh/xui v", "github.com/twgh/xcgui v"} {
		if !strings.Contains(string(gomod), want) {
			t.Errorf("go.mod 缺少 %q:\n%s", want, gomod)
		}
	}

	theme, err := os.ReadFile(filepath.Join(dir, "myapp", "ui", "theme.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(theme), "func UseDark(") {
		t.Errorf("theme.go 应生成 UseDark 切换助手:\n%s", theme)
	}

	mainGo, err := os.ReadFile(filepath.Join(dir, "myapp", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainGo), `"myapp/ui"`) {
		t.Errorf("main.go 未导入 myapp/ui:\n%s", mainGo)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "main.go", mainGo, 0); err != nil {
		t.Errorf("生成的 main.go 语法错误: %v", err)
	}

	// 非空目录默认拒绝, --force 放行且不删除无关文件
	if err := initNewProject("myapp", false, true); err == nil {
		t.Error("非空目录未加 --force 应报错")
	}
	keep := filepath.Join(dir, "myapp", "notes.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := initNewProject("myapp", true, true); err != nil {
		t.Fatalf("--force 应允许写入非空目录: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Error("--force 不应删除无关文件")
	}
}

func TestParseComponentArgs(t *testing.T) {
	got, err := parseComponentArgs([]string{"-o", "-p", "components", "button", "card"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.overwrite || got.uiDir != "components" || len(got.names) != 2 {
		t.Errorf("解析结果不对: %+v", got)
	}
	if got, err := parseComponentArgs([]string{"--all"}); err != nil || !got.all {
		t.Errorf("--all 解析失败: %+v, %v", got, err)
	}
	if got, err := parseComponentArgs([]string{"--dry-run", "--view=button.go", "button"}); err != nil ||
		!got.dryRun || !got.view || got.viewFile != "button.go" {
		t.Errorf("dry-run/view 解析失败: %+v, %v", got, err)
	}
	if _, err := parseComponentArgs([]string{"--view", "--diff", "button"}); err == nil {
		t.Error("--view 与 --diff 应互斥")
	}
	if _, err := parseComponentArgs([]string{"--all", "button"}); err == nil {
		t.Error("--all 与组件名应互斥")
	}
	if _, err := parseComponentArgs([]string{"--wat"}); err == nil {
		t.Error("未知选项应报错")
	}
}

func TestParseListArgs(t *testing.T) {
	got, err := parseListArgs([]string{"-q", "table", "-l", "5", "-o", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "table" || got.limit != 5 || got.offset != 2 {
		t.Errorf("解析结果不对: %+v", got)
	}
	if got, err := parseListArgs([]string{"--query=card", "--limit=3", "--offset=1"}); err != nil ||
		got.query != "card" || got.limit != 3 || got.offset != 1 {
		t.Errorf("= 形式解析失败: %+v, %v", got, err)
	}
	for _, args := range [][]string{{"-l", "0"}, {"-l", "x"}, {"-o", "-1"}, {"button"}} {
		if _, err := parseListArgs(args); err == nil {
			t.Errorf("%v 应报错", args)
		}
	}
}

func TestParseViewArgs(t *testing.T) {
	names, err := parseViewArgs([]string{"button", "card"})
	if err != nil || len(names) != 2 {
		t.Errorf("view 参数解析失败: %v, %v", names, err)
	}
	if _, err := parseViewArgs(nil); err == nil {
		t.Error("view 缺少组件名应报错")
	}
	if _, err := parseViewArgs([]string{"--wat"}); err == nil {
		t.Error("未知选项应报错")
	}
}

func TestExtractGlobal(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	quiet = false
	defer func() { quiet = false }()

	dir := t.TempDir()
	rest, err := extractGlobal([]string{"-s", "-c", dir, "add", "button"})
	if err != nil {
		t.Fatal(err)
	}
	if !quiet {
		t.Error("-s 未打开 silent")
	}
	if strings.Join(rest, ",") != "add,button" {
		t.Errorf("剩余参数 = %v", rest)
	}
	wd, _ := os.Getwd()
	if !samePath(wd, dir) {
		t.Errorf("cwd = %q, want %q", wd, dir)
	}
	if _, err := extractGlobal([]string{"--cwd"}); err == nil {
		t.Error("--cwd 缺值应报错")
	}
}

// samePath 规避 Windows 下大小写/8.3 短路径差异, 只用于测试断言。
func samePath(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	return err1 == nil && err2 == nil && strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
}

func TestUnifiedDiff(t *testing.T) {
	d := unifiedDiff("ui/a.go", "components/a.go", "a\nb\n", "a\nc\n")
	for _, want := range []string{"--- ui/a.go", "+++ components/a.go", "-b\n", "+c\n", " a\n"} {
		if !strings.Contains(d, want) {
			t.Errorf("diff 缺少 %q:\n%s", want, d)
		}
	}
	if d := unifiedDiff("a", "b", "", "x\n"); !strings.Contains(d, "+x\n") {
		t.Errorf("新增文件 diff 不对:\n%s", d)
	}
}

func TestFilterComponents(t *testing.T) {
	all := []string{"button", "card", "table", "tabs"}
	got := filterComponents(all, "ta")
	if strings.Join(got, ",") != "table,tabs" {
		t.Errorf("过滤结果 = %v", got)
	}
	if got := filterComponents(all, "TABLE"); len(got) != 1 || got[0] != "table" {
		t.Errorf("大小写不敏感过滤失败: %v", got)
	}
	if got := filterComponents(all, ""); len(got) != 4 {
		t.Errorf("空查询应返回全部: %v", got)
	}
}

// 用户输错形状(radiogroup / radio_group / RadioGroup / navigationmenu)也要能装对;
// 精确名必须优先, 不能被同族抢走。
func TestLookupComponent(t *testing.T) {
	reg := &registry{Components: map[string]entry{
		"radio": {}, "radio-group": {}, "input": {}, "input-group": {},
		"navigation-menu": {}, "combobox-search": {}, "carousel": {},
	}}
	cases := []struct{ in, want string }{
		{"radio-group", "radio-group"},
		{"radiogroup", "radio-group"},
		{"radio_group", "radio-group"},
		{"RadioGroup", "radio-group"},
		{"radio", "radio"},
		{"inputgroup", "input-group"},
		{"input", "input"},
		{"navigation-menu", "navigation-menu"},
		{"navigationmenu", "navigation-menu"},
		{"comboboxsearch", "combobox-search"},
		{"carousel", "carousel"},
	}
	for _, c := range cases {
		if got := lookupComponent(reg, c.in); got != c.want {
			t.Errorf("lookupComponent(%q) = %q, 期望 %q", c.in, got, c.want)
		}
	}
}

// 打错字/只写前缀时给候选, 太远的输入不硬凑。
func TestSuggestComponent(t *testing.T) {
	reg := &registry{Components: map[string]entry{
		"radio-group": {}, "navigation-menu": {}, "carousel": {},
	}}
	if s := suggestComponent(reg, normalizeName("radio-grup")); !strings.Contains(s, "radio-group") {
		t.Errorf("拼写错误应提示 radio-group, 实得 %q", s)
	}
	if s := suggestComponent(reg, normalizeName("nav")); !strings.Contains(s, "navigation-menu") {
		t.Errorf("前缀应提示 navigation-menu, 实得 %q", s)
	}
	if s := suggestComponent(reg, normalizeName("zzzzzzzz")); s != "" {
		t.Errorf("完全无关的输入不该给候选, 实得 %q", s)
	}
}

func TestAddDryRunDoesNotWrite(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	add(componentOptions{names: []string{"button"}, uiDir: "ui", dryRun: true})
	if _, err := os.Stat("ui/button.go"); !os.IsNotExist(err) {
		t.Errorf("--dry-run 不应写文件: %v", err)
	}
	add(componentOptions{names: []string{"button"}, uiDir: "ui"})
	if _, err := os.Stat("ui/button.go"); err != nil {
		t.Errorf("正常 add 应写文件: %v", err)
	}
}

// 安装清单让 update 能区分"用户改过"和"版本升级": 本地未动过的文件随版本替换,
// 改过的仍然跳过, 没有清单的老项目退回保守跳过。
func TestUpdateUsesManifestToTellUpgradeFromLocalEdit(t *testing.T) {
	const v1 = "package ui\n// button v1\n"
	const v2 = "package ui\n// button v2\n"
	const v3 = "package ui\n// button v3\n"

	remote := v1
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registry.json":
			w.Write([]byte(`{"components":{"button":{"files":["button.go"],"deps":[]}}}`))
		case "/components/button.go":
			w.Write([]byte(remote))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	t.Setenv("XUI_REGISTRY", srv.URL)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	add(componentOptions{names: []string{"button"}, uiDir: "ui"})
	if m := loadManifest(filepath.Join(dir, "ui")); m.Files["button.go"] != sha256Hex([]byte(v1)) {
		t.Fatalf("add 后清单应记录文件哈希, 实为 %+v", m)
	}

	// 远端升到 v2、本地没动过 → 直接替换
	remote = v2
	update(componentOptions{names: []string{"button"}, uiDir: "ui"})
	if data, _ := os.ReadFile(filepath.Join(dir, "ui", "button.go")); string(data) != v2 {
		t.Errorf("本地未改动的文件应随版本升级替换, 实为 %q", data)
	}

	// 用户改过 → 远端再到 v3 也跳过, 且清单保持"上次安装"的哈希不变
	dst := filepath.Join(dir, "ui", "button.go")
	if err := os.WriteFile(dst, []byte("package ui\n// mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	remote = v3
	update(componentOptions{names: []string{"button"}, uiDir: "ui"})
	if data, _ := os.ReadFile(dst); string(data) != "package ui\n// mine\n" {
		t.Errorf("改过的文件不该被覆盖, 实为 %q", data)
	}

	// 删掉清单(老项目) → 内容不同只能保守跳过
	if err := os.Remove(filepath.Join(dir, "ui", manifestName)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(v2), 0o644); err != nil {
		t.Fatal(err)
	}
	update(componentOptions{names: []string{"button"}, uiDir: "ui"})
	if data, _ := os.ReadFile(dst); string(data) != v2 {
		t.Errorf("无清单时应保守跳过, 实为 %q", data)
	}
}
