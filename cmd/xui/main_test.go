package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
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

// stubWinresMake 把 winresMake 换成桩(写出假的 .syso), 避免单测依赖本机 xc;
// 返回的切片记录每次调用的 allowInstall。
func stubWinresMake(t *testing.T) *[]bool {
	t.Helper()
	var calls []bool
	orig := winresMake
	winresMake = func(dir string, allowInstall bool) error {
		calls = append(calls, allowInstall)
		for _, arch := range []string{"386", "amd64"} {
			if err := os.WriteFile(filepath.Join(dir, "rsrc_windows_"+arch+".syso"), []byte("stub"), 0o644); err != nil {
				return err
			}
		}
		return nil
	}
	t.Cleanup(func() { winresMake = orig })
	return &calls
}

func TestInitNewProjectIntoEmptyCwd(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot()) // 本地源: 不访问网络
	stubWinresMake(t)
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
	calls := stubWinresMake(t)
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
		"go.mod", "main.go", "README.md", "AGENTS.md", ".gitignore",
		"ui/theme.go", "ui/DESIGN.md", "ui/button.go", "ui/window.go",
		"winres/icon.ico", "winres/winres.json",
		"rsrc_windows_386.syso", "rsrc_windows_amd64.syso",
	} {
		if _, err := os.Stat(filepath.Join(dir, "myapp", f)); err != nil {
			t.Errorf("缺少 %s: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "myapp", "rsrc_windows_arm64.syso")); err == nil {
		t.Error("不应生成 arm64 syso: xcgui 不支持 ARM")
	}
	icon, err := os.ReadFile(filepath.Join(dir, "myapp", "winres", "icon.ico"))
	if err != nil {
		t.Fatal(err)
	}
	if string(icon) != string(projectIconICO) {
		t.Error("winres/icon.ico 应写入内嵌的默认图标")
	}
	winres, err := os.ReadFile(filepath.Join(dir, "myapp", "winres", "winres.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"RT_GROUP_ICON"`, `"APP"`, `"RT_MANIFEST"`, `"RT_VERSION"`,
		`"dpi-awareness": "per monitor v2"`,
		`"OriginalFilename": "myapp.exe"`, `"ProductName": "myapp"`,
	} {
		if !strings.Contains(string(winres), want) {
			t.Errorf("winres.json 缺少 %q:\n%s", want, winres)
		}
	}
	if got := *calls; len(got) == 0 || got[0] {
		t.Errorf("--no-install 应传 allowInstall=false, calls=%v", got)
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
	for _, want := range []string{"setWindowIcons(w)", "SetBigIcon", "SetSmallIcon"} {
		if !strings.Contains(string(mainGo), want) {
			t.Errorf("生成的 main.go 缺少窗口图标设置 %q:\n%s", want, mainGo)
		}
	}

	agents, err := os.ReadFile(filepath.Join(dir, "myapp", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "SetPaintFrequency(10)") {
		t.Errorf("项目根 AGENTS.md 缺少运行约定:\n%s", agents)
	}
	if !strings.Contains(string(agents), `ui "myapp/ui"`) {
		t.Errorf("项目根 AGENTS.md 的导入路径没替换成模块名:\n%s", agents)
	}
	if !strings.Contains(string(agents), "xc res make --arch amd64,386") {
		t.Errorf("项目根 AGENTS.md 缺少 winres 生成说明:\n%s", agents)
	}
	for _, want := range []string{"SetBigIcon", "SetSmallIcon", "任务栏预览窗口左上角"} {
		if !strings.Contains(string(agents), want) {
			t.Errorf("项目根 AGENTS.md 缺少窗口图标说明 %q:\n%s", want, agents)
		}
	}
	if !strings.Contains(string(agents), "go install github.com/twgh/xui/cmd/xui@latest") {
		t.Errorf("项目根 AGENTS.md 缺少 CLI 安装说明:\n%s", agents)
	}
	readme, err := os.ReadFile(filepath.Join(dir, "myapp", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "winres/") || !strings.Contains(string(readme), "xc res make") {
		t.Errorf("生成的 README 缺少 winres 说明:\n%s", readme)
	}
	for _, want := range []string{"SetBigIcon", "SetSmallIcon", "任务栏预览窗口左上角"} {
		if !strings.Contains(string(readme), want) {
			t.Errorf("生成的 README 缺少窗口图标说明 %q:\n%s", want, readme)
		}
	}
	for _, want := range []string{"go install github.com/twgh/xui/cmd/xui@latest", "xui add"} {
		if !strings.Contains(string(readme), want) {
			t.Errorf("生成的 README 缺少 CLI 使用说明 %q:\n%s", want, readme)
		}
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

// winres 模板必须是 `xc res init` 的完整三段(图标/清单/版本), 且项目名按 JSON 转义。
func TestProjectWinresJSON(t *testing.T) {
	body := projectWinresJSON(`my"app`)
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("winres.json 不是合法 JSON: %v", err)
	}
	for _, key := range []string{"RT_GROUP_ICON", "RT_MANIFEST", "RT_VERSION"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("缺少 %s 段", key)
		}
	}
	if !strings.Contains(body, `"OriginalFilename": "my\"app.exe"`) {
		t.Errorf("项目名未按 JSON 转义:\n%s", body)
	}
	if !strings.Contains(body, `"APP"`) {
		t.Error("图标资源名应为 APP")
	}
}

// --no-install 时 xc 不存在不能触发 go install, 只报错。
func TestFindXcNoInstall(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if p, err := exec.LookPath("xc"); err == nil {
		t.Skipf("测试环境仍能解析到 xc (%s), 跳过", p)
	}
	if _, err := findXc(false); err == nil {
		t.Error("xc 缺失且不允许安装时应返回错误")
	}
}

// 组件目录可能不叫 ui: DESIGN.md 的文件引用必须是同目录的裸文件名,
// AGENTS.md 要落在组件目录里, 且重复 init 不覆盖用户改过的版本。
func TestInitProjectDocsAreDirAgnostic(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
	dir := filepath.Join(t.TempDir(), "widgets")
	initProject(dir, false)

	doc, err := os.ReadFile(filepath.Join(dir, "DESIGN.md"))
	if err != nil {
		t.Fatalf("缺少 DESIGN.md: %v", err)
	}
	if m := regexp.MustCompile(`(?:^|[^A-Za-z0-9_/])ui/[a-z0-9]+\.go`).FindString(string(doc)); m != "" {
		t.Errorf("DESIGN.md 写死了 ui/ 路径: %q", m)
	}
	if !strings.Contains(string(doc), "组件实现以组件目录下的 `*.go` 为准") {
		t.Error("DESIGN.md 缺少“组件目录下的 *.go”口径")
	}

	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("缺少 AGENTS.md: %v", err)
	}
	for _, want := range []string{"DESIGN.md", "xui.Current()", "ui.ScrollArea", "go install github.com/twgh/xui/cmd/xui@latest"} {
		if !strings.Contains(string(agents), want) {
			t.Errorf("AGENTS.md 缺少 %q", want)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}
	initProject(dir, false)
	if got, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md")); string(got) != "custom" {
		t.Error("重复 init 覆盖了已有的 AGENTS.md")
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
	// 短选项也接受 = 形式(与 --path= 一致)
	if got, err := parseComponentArgs([]string{"-p=src", "button"}); err != nil || got.uiDir != "src" {
		t.Errorf("-p=src 解析失败: %+v, %v", got, err)
	}
	// 值位置紧跟另一个选项时要报缺值, 不能把 --all 当成目录名
	if _, err := parseComponentArgs([]string{"-p", "--all"}); err == nil {
		t.Error("-p 后紧跟选项应报缺值")
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
	// 短选项 = 形式
	if got, err := parseListArgs([]string{"-q=card", "-l=2", "-o=1"}); err != nil ||
		got.query != "card" || got.limit != 2 || got.offset != 1 {
		t.Errorf("短选项 = 形式解析失败: %+v, %v", got, err)
	}
	// -q 后面紧跟选项: 以前会把 "-l" 当搜索词, 静默吞掉后续参数
	if _, err := parseListArgs([]string{"-q", "-l", "3"}); err == nil {
		t.Error("-q 后紧跟选项应报缺值")
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
	// 短选项 = 形式与长选项一致
	if _, err := extractGlobal([]string{"-c=" + dir}); err != nil {
		t.Errorf("-c=dir 应可用: %v", err)
	}
	// 值位置紧跟另一个选项 → 报缺值(不能把 -s 当目录)
	if _, err := extractGlobal([]string{"-c", "-s"}); err == nil {
		t.Error("-c 后紧跟选项应报缺值")
	}
}

// 未知命令必须以非零码退出 —— 脚本里 `xui foo || exit 1` 要能判错(以前打 usage 后返回 0)。
func TestUnknownCommandExitsNonZero(t *testing.T) {
	if os.Getenv("XUI_TEST_UNKNOWN_CMD") == "1" {
		os.Args = []string{"xui", "definitely-not-a-command"}
		main()
		os.Exit(0) // 走到这里说明没报错
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=TestUnknownCommandExitsNonZero")
	cmd.Env = append(os.Environ(), "XUI_TEST_UNKNOWN_CMD=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("未知命令应非零退出, 输出: %s", out)
	}
	if !strings.Contains(string(out), "未知命令") {
		t.Errorf("报错文案不对: %s", out)
	}
}

// -h/--help 在任何位置都打帮助(包括子命令参数位置), 且退出码为 0。
func TestHelpFlagAnywhere(t *testing.T) {
	if os.Getenv("XUI_TEST_HELP") == "1" {
		os.Args = []string{"xui", "add", "--help"}
		main()
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=TestHelpFlagAnywhere")
	cmd.Env = append(os.Environ(), "XUI_TEST_HELP=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help 应正常退出(码 0): %v, %s", err, out)
	}
	if !strings.Contains(string(out), "xui — shadcn") {
		t.Errorf("应打印帮助: %s", out)
	}
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

// captureOutput 抓 fn 期间的标准输出与警告(info → stdout, warn → stderr)。
// 动作文案本身就是行为约定, 只断言文件内容抓不到"说的是什么话"。
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = w, w
	reads := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		reads <- buf.String()
	}()
	fn()
	_ = w.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	out := <-reads
	_ = r.Close()
	return out
}

// -o 的文案要分清"版本升级"与"真的丢了本地改动": 干净项目加 -o 也报"本地修改已丢弃",
// 会让人以为自己有改动没备份; 没有安装基线时同样不能断言用户改过。
func TestOverwriteWordingTellsUpgradeFromLostEdit(t *testing.T) {
	const v1 = "package ui\n// button v1\n"
	const v2 = "package ui\n// button v2\n"
	const v3 = "package ui\n// button v3\n"
	const mine = "package ui\n// mine\n"

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
	dst := filepath.Join("ui", "button.go")

	// 装完没动过 + -o → 纯版本差异, 说"版本升级"
	remote = v2
	out := captureOutput(t, func() {
		update(componentOptions{names: []string{"button"}, uiDir: "ui", overwrite: true})
	})
	if data, _ := os.ReadFile(dst); string(data) != v2 {
		t.Errorf("-o 应替换未改动的文件, 实为 %q", data)
	}
	if !strings.Contains(out, "版本升级") || strings.Contains(out, "本地修改已丢弃") {
		t.Errorf("未改动的文件在 -o 下应报版本升级, 实为 %q", out)
	}

	// 真改过 + -o → 这才叫"本地修改已丢弃"
	if err := os.WriteFile(dst, []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}
	remote = v3
	out = captureOutput(t, func() {
		update(componentOptions{names: []string{"button"}, uiDir: "ui", overwrite: true})
	})
	if data, _ := os.ReadFile(dst); string(data) != v3 {
		t.Errorf("-o 应覆盖改过的文件, 实为 %q", data)
	}
	if !strings.Contains(out, "本地修改已丢弃") {
		t.Errorf("改过的文件在 -o 下应报本地修改已丢弃, 实为 %q", out)
	}

	// 没有基线(老项目): 默认跳过与 -o 都只说"判断不了", 不断言用户改过
	if err := os.Remove(filepath.Join("ui", manifestName)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(v2), 0o644); err != nil {
		t.Fatal(err)
	}
	out = captureOutput(t, func() {
		update(componentOptions{names: []string{"button"}, uiDir: "ui"})
	})
	if strings.Contains(out, "本地已修改") || !strings.Contains(out, "无法判断来源") {
		t.Errorf("无基线时默认路径只该说判断不了, 实为 %q", out)
	}
	if data, _ := os.ReadFile(dst); string(data) != v2 {
		t.Errorf("无基线时不应覆盖, 实为 %q", data)
	}
	out = captureOutput(t, func() {
		update(componentOptions{names: []string{"button"}, uiDir: "ui", overwrite: true})
	})
	if strings.Contains(out, "本地修改已丢弃") || !strings.Contains(out, "无安装清单基线") {
		t.Errorf("无基线时 -o 不该断言丢了本地改动, 实为 %q", out)
	}
	if data, _ := os.ReadFile(dst); string(data) != v3 {
		t.Errorf("-o 仍应覆盖, 实为 %q", data)
	}
}

// TestMain 默认把 go get 换成桩: 单测绝不联网改依赖。
func TestMain(m *testing.M) {
	goGetModule = func(string, string) error { return nil }
	os.Exit(m.Run())
}

// goGetCall 记录一次被桩掉的 go get 调用。
type goGetCall struct{ dir, spec string }

// stubGoGet 换成记录用的桩, 返回调用清单。
func stubGoGet(t *testing.T) *[]goGetCall {
	t.Helper()
	orig := goGetModule
	var calls []goGetCall
	goGetModule = func(dir, spec string) error {
		calls = append(calls, goGetCall{dir, spec})
		return nil
	}
	t.Cleanup(func() { goGetModule = orig })
	return &calls
}

// chdirProject 进临时目录, 铺一个"项目": go.mod + ui/ 组件目录。
func chdirProject(t *testing.T, goMod string) string {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "ui"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// 组件从 main 现拉、运行时只能按 tag 取: 装完组件必须把项目的 xui 依赖抬到 registry
// 声明的下限, 否则组件用了新运行时助手而项目钉着旧 tag, 装成功却编不过。
func TestAddRaisesRuntimeDependencyToRegistryFloor(t *testing.T) {
	calls := stubGoGet(t)
	const goMod = "module myapp\n\ngo 1.18\n\nrequire (\n\tgithub.com/twgh/xcgui v1.4.3\n\tgithub.com/twgh/xui v0.7.0\n)\n"
	remote := "package ui\n// button v1\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registry.json":
			w.Write([]byte(`{"runtime":"v0.8.0","components":{"button":{"files":["button.go"],"deps":[]}}}`))
		case "/components/button.go":
			w.Write([]byte(remote))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("XUI_REGISTRY", srv.URL)
	dir := chdirProject(t, goMod)

	add(componentOptions{names: []string{"button"}, uiDir: "ui"})
	if len(*calls) != 1 || (*calls)[0].spec != "github.com/twgh/xui@v0.8.0" {
		t.Fatalf("应恰好调用一次 go get 把运行时抬到 registry 下限, 实为 %+v", *calls)
	}
	if (*calls)[0].dir != dir {
		t.Errorf("go get 的工作目录应为项目根 %q, 实为 %q", dir, (*calls)[0].dir)
	}

	// 只预览的路径不许动依赖
	*calls = nil
	add(componentOptions{names: []string{"button"}, uiDir: "ui", dryRun: true})
	if len(*calls) != 0 {
		t.Errorf("--dry-run 不该动依赖, 实为 %+v", *calls)
	}
}

// 下限不该被滥动: 够新 / 更高 / replace 到本地 / --no-install / 找到 xui 自己的
// go.mod, 一律不起 go get。
func TestEnsureRuntimeLeavesAloneWhenItShould(t *testing.T) {
	cases := []struct {
		name, body       string
		dryRun, autoInst bool
		wantCall         bool
		wantOut          string
	}{
		{"已经够新", "module myapp\nrequire github.com/twgh/xui v0.8.0\n", false, true, false, ""},
		{"比下限更高", "module myapp\nrequire github.com/twgh/xui v0.9.0\n", false, true, false, ""},
		{"块式且够新", "module myapp\nrequire (\n\tgithub.com/twgh/xui v0.8.0\n)\n", false, true, false, ""},
		{"replace 到工作副本", "module myapp\nrequire github.com/twgh/xui v0.0.0\nreplace github.com/twgh/xui => ../xui\n", false, true, false, "被 replace"},
		{"xui 仓库自己的 go.mod", "module github.com/twgh/xui\n", false, true, false, ""},
		{"需要抬(dry-run)", "module myapp\nrequire github.com/twgh/xui v0.7.0\n", true, true, false, "将升级运行时"},
		{"需要抬(--no-install)", "module myapp\nrequire github.com/twgh/xui v0.7.0\n", false, false, false, "go get github.com/twgh/xui@v0.8.0"},
		{"需要抬(没写 require)", "module myapp\n\ngo 1.18\n", false, true, true, "升级运行时"},
		{"需要抬(旧版)", "module myapp\nrequire github.com/twgh/xui v0.7.0\n", false, true, true, "运行时已升到 v0.8.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calls := stubGoGet(t)
			dir := chdirProject(t, c.body)
			out := captureOutput(t, func() {
				ensureRuntime("ui", "v0.8.0", c.dryRun, c.autoInst)
			})
			if got := len(*calls) != 0; got != c.wantCall {
				t.Errorf("go get 调用 = %v, 期望 %v (输出 %q)", got, c.wantCall, out)
			}
			if len(*calls) > 0 && (*calls)[0].spec != "github.com/twgh/xui@v0.8.0" {
				t.Errorf("spec = %q", (*calls)[0].spec)
			}
			if len(*calls) > 0 && (*calls)[0].dir != dir {
				t.Errorf("工作目录 = %q, 期望项目根 %q", (*calls)[0].dir, dir)
			}
			switch {
			case c.wantOut != "" && !strings.Contains(out, c.wantOut):
				t.Errorf("输出缺少 %q, 实为 %q", c.wantOut, out)
			case c.wantOut == "" && strings.Contains(out, "升级"):
				t.Errorf("不该有升级动作/文案, 实为 %q", out)
			}
		})
	}
}

// go.mod 的 require/replace 解析要认常见写法(块式、单行式、// indirect、伪版本)。
func TestGoModRuntimePin(t *testing.T) {
	cases := []struct {
		name, body, wantVer string
		wantReplace         bool
	}{
		{"块式 require", "module myapp\n\ngo 1.18\n\nrequire (\n\tgithub.com/twgh/xcgui v1.4.3\n\tgithub.com/twgh/xui v0.7.0\n)\n", "v0.7.0", false},
		{"单行 require", "module myapp\nrequire github.com/twgh/xui v0.8.0\n", "v0.8.0", false},
		{"带 indirect 注释", "module myapp\nrequire github.com/twgh/xui v0.7.1 // indirect\n", "v0.7.1", false},
		{"伪版本", "module myapp\nrequire github.com/twgh/xui v0.7.1-0.20261102040404-f7072ad01234\n", "v0.7.1-0.20261102040404-f7072ad01234", false},
		{"块式 replace", "module myapp\nrequire (\n\tgithub.com/twgh/xui v0.0.0\n)\nreplace (\n\tgithub.com/twgh/xui => ../xui\n)\n", "v0.0.0", true},
		{"单行 replace 带版本", "module myapp\nrequire github.com/twgh/xui v0.7.0\nreplace github.com/twgh/xui v0.7.0 => ./local\n", "v0.7.0", true},
		{"没要求该模块", "module myapp\n\ngo 1.18\n", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "go.mod")
			if err := os.WriteFile(path, []byte(c.body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, ver, replaced, err := goModRuntimePin(path)
			if err != nil {
				t.Fatal(err)
			}
			if ver != c.wantVer || replaced != c.wantReplace {
				t.Errorf("ver=%q replaced=%v, 期望 ver=%q replaced=%v", ver, replaced, c.wantVer, c.wantReplace)
			}
		})
	}
}

func TestVersionBeforeAndReleaseTag(t *testing.T) {
	for _, c := range [][2]string{
		{"v0.7.0", "v0.8.0"}, {"v0.8.0", "v0.10.0"}, {"v1.2.3", "v2.0.0"},
		{"v0.0.0", "v0.8.0"}, {"", "v0.8.0"}, {"latest", "v0.8.0"},
		{"v0.7.1-0.20261102040404-f7072ad01234", "v0.8.0"},
	} {
		if !versionBefore(c[0], c[1]) {
			t.Errorf("%q 应低于 %q", c[0], c[1])
		}
	}
	for _, c := range [][2]string{
		{"v0.8.0", "v0.8.0"}, {"v0.8.1", "v0.8.0"}, {"v0.9.0", "v0.8.0"},
		{"v0.9.0-0.20261102040404-f7072ad01234", "v0.8.0"}, {"v1.0.0", "v0.8.0"},
	} {
		if versionBefore(c[0], c[1]) {
			t.Errorf("%q 不应低于 %q", c[0], c[1])
		}
	}
	for _, v := range []string{"v0.8.0", "v10.20.30"} {
		if !isReleaseTag(v) {
			t.Errorf("%q 应是发布 tag", v)
		}
	}
	for _, v := range []string{"", "0.8.0", "v0.8", "v0.8.0-dirty", "v0.8.0-0.20261102-f7072ad", "main"} {
		if isReleaseTag(v) {
			t.Errorf("%q 不该被当成发布 tag", v)
		}
	}
}

// registry.json 的 runtime 字段: 缺失(老镜像)要容忍, 格式不对要忽略并警告。
func TestParseRegistryRuntime(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"正常", `{"runtime":"v0.8.0","components":{"button":{"files":["button.go"],"deps":[]}}}`, "v0.8.0"},
		{"缺字段", `{"components":{"button":{"files":["button.go"],"deps":[]}}}`, ""},
		{"格式不对", `{"runtime":"main","components":{"button":{"files":["button.go"],"deps":[]}}}`, ""},
		{"伪版本不算", `{"runtime":"v0.8.1-0.20261102","components":{"button":{"files":["button.go"],"deps":[]}}}`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reg, err := parseRegistry("test", []byte(c.body))
			if err != nil {
				t.Fatal(err)
			}
			if reg.Runtime != c.want {
				t.Errorf("runtime=%q 期望 %q", reg.Runtime, c.want)
			}
		})
	}
}

// self-update 的目标串: 空/latest 归一成 @latest, 分支与 commit 原样,
// 裸版本号补 v, 认不出的直接报错(别跑一条注定失败的 go install)。
func TestCliInstallSpec(t *testing.T) {
	cases := []struct {
		target  string
		want    string
		wantErr bool
	}{
		{"", cliModule + "@latest", false},
		{"latest", cliModule + "@latest", false},
		{"@latest", cliModule + "@latest", false},
		{"LATEST", cliModule + "@latest", false},
		{"main", cliModule + "@main", false},
		{"HEAD", cliModule + "@HEAD", false},
		{"v0.8.1", cliModule + "@v0.8.1", false},
		{"0.8.1", cliModule + "@v0.8.1", false},
		{"v0.8.1-0.20261102101010-f7072ad01234", cliModule + "@v0.8.1-0.20261102101010-f7072ad01234", false},
		{"f7072ad", cliModule + "@f7072ad", false},
		{"v0.8", "", true},
		{"v1.2.3.4", "", true},
		{"not a version", "", true},
		{"../elsewhere", "", true},
		{"zzzzzzz", "", true},
	}
	for _, c := range cases {
		got, err := cliInstallSpec(c.target)
		if c.wantErr {
			if err == nil {
				t.Errorf("目标 %q 应报错, 实为 %q", c.target, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("目标 %q 不该报错: %v", c.target, err)
			continue
		}
		if got != c.want {
			t.Errorf("目标 %q => %q, 期望 %q", c.target, got, c.want)
		}
	}
}

// 临时构建(go run / go test 的产物)认得出来, 装在自定义目录里的正常二进制不误判。
func TestIsTempBuildPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/tmp/go-build123456/b001/exe/xui", true},
		{"C:/Users/u/AppData/Local/Temp/go-build7/b001/exe/xui.exe", true},
		{"/home/u/go/bin/xui", false},
		{"C:/Users/u/go/bin/xui.exe", false},
		// 单纯"在临时目录下"不足以判定是临时构建: 有人就把二进制放 Temp 里试。
		{filepath.ToSlash(filepath.Join(os.TempDir(), "xui-check", "xui.exe")), false},
	}
	for _, c := range cases {
		if got := isTempBuildPath(c.path); got != c.want {
			t.Errorf("isTempBuildPath(%q) = %v, 期望 %v", c.path, got, c.want)
		}
	}
}

// stubSelfUpdate 把自我升级依赖的外部世界(go install / 查最新 tag / GOBIN / 读新版本)
// 全换成桩; installErrs 依次决定前几次 go install 的成败。
func stubSelfUpdate(t *testing.T, binDir, newest, nextVer string, installErrs ...error) *[]string {
	t.Helper()
	origInstall, origList, origBin, origVer := goInstall, goListLatest, goBinDir, readBinaryVersion
	t.Cleanup(func() {
		goInstall, goListLatest, goBinDir, readBinaryVersion = origInstall, origList, origBin, origVer
	})
	var specs []string
	step := 0
	goInstall = func(spec string) error {
		specs = append(specs, spec)
		if step < len(installErrs) {
			err := installErrs[step]
			step++
			if err != nil {
				return err
			}
		}
		if binDir == "" {
			return nil
		}
		return os.WriteFile(filepath.Join(binDir, "xui"+exeExt()), []byte("binary"), 0o755)
	}
	goListLatest = func() (string, error) { return newest, nil }
	if binDir == "" {
		goBinDir = func() (string, error) { return "", errors.New("没有 GOBIN") }
	} else {
		goBinDir = func() (string, error) { return binDir, nil }
	}
	readBinaryVersion = func(string) (string, error) { return nextVer, nil }
	return &specs
}

// --check 只比较版本, 一次 go install 都不跑。
func TestSelfUpdateCheckOnly(t *testing.T) {
	cases := []struct {
		name, current, newest, wantOut string
	}{
		{"可以升级", "v0.8.0", "v0.9.9", "可以升级: xui self-update v0.9.9"},
		{"已是最新", "v0.9.9", "v0.9.9", "已是最新"},
		{"本地构建", "(devel)", "v0.9.9", "不算发布版"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			specs := stubSelfUpdate(t, filepath.Join(t.TempDir(), "bin"), c.newest, c.newest)
			out := captureOutput(t, func() {
				if err := selfUpdateAt("/home/u/go/bin/xui", c.current, "", true); err != nil {
					t.Errorf("不该失败: %v", err)
				}
			})
			if len(*specs) != 0 {
				t.Errorf("--check 不该跑 go install, 实为 %v", *specs)
			}
			if !strings.Contains(out, c.wantOut) {
				t.Errorf("输出缺少 %q, 实为 %q", c.wantOut, out)
			}
		})
	}
}

// 升级成功: 跑 go install@目标, 然后以"新二进制的版本"为证据报结果;
// 运行副本不在 GOBIN 时要额外提醒 PATH。
func TestSelfUpdateUpgradesAndReportsNewVersion(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	specs := stubSelfUpdate(t, bin, "v0.9.9", "v0.9.9")
	running := filepath.Join(t.TempDir(), "elsewhere", "xui"+exeExt())
	out := captureOutput(t, func() {
		if err := selfUpdateAt(running, "v0.8.0", "v0.9.9", false); err != nil {
			t.Errorf("不该失败: %v", err)
		}
	})
	if len(*specs) != 1 || (*specs)[0] != cliModule+"@v0.9.9" {
		t.Fatalf("go install 目标不对: %v", *specs)
	}
	for _, want := range []string{"CLI 已升级 v0.8.0 → v0.9.9", "下次调用生效", "不在 GOBIN"} {
		if !strings.Contains(out, want) {
			t.Errorf("输出缺少 %q, 实为 %q", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(bin, "xui"+exeExt())); err != nil {
		t.Errorf("GOBIN 里应有新二进制: %v", err)
	}
}

// @latest 常被模块代理拖旧: 装完版本没变要说清原因, 不能报"已升级"。
func TestSelfUpdateReportsUnchangedVersion(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	stubSelfUpdate(t, bin, "v0.8.0", "v0.8.0")
	out := captureOutput(t, func() {
		if err := selfUpdateAt(filepath.Join(bin, "xui"+exeExt()), "v0.8.0", "", false); err != nil {
			t.Errorf("不该失败: %v", err)
		}
	})
	if !strings.Contains(out, "版本没变(v0.8.0)") || !strings.Contains(out, "代理") {
		t.Errorf("应说明 @latest 滞后, 实为 %q", out)
	}
	if strings.Contains(out, "已升级") {
		t.Errorf("版本没变不许说已升级: %q", out)
	}
}

// 正在运行的 exe 不许就地覆写: 第一次 go install 失败时改名让路重试一次。
func TestSelfUpdateRetiresLockedBinaryAndRetries(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(bin, "xui"+exeExt())
	if err := os.WriteFile(installed, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	specs := stubSelfUpdate(t, bin, "v0.9.9", "v0.9.9", errors.New("access denied"))
	out := captureOutput(t, func() {
		if err := selfUpdateAt(installed, "v0.8.0", "", false); err != nil {
			t.Errorf("改名让路后应重试成功: %v", err)
		}
	})
	if len(*specs) != 2 {
		t.Fatalf("应重试一次 go install, 实为 %v", *specs)
	}
	if !strings.Contains(out, "改名让路") || !strings.Contains(out, "CLI 已升级") {
		t.Errorf("输出不对: %q", out)
	}
	if _, err := os.Stat(installed + ".old"); err != nil {
		t.Errorf("旧文件应被改名成 .old: %v", err)
	}
}

// 失败路径: 报清楚的手动命令; 临时构建直接拒绝。
func TestSelfUpdateFailures(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	stubSelfUpdate(t, bin, "v0.9.9", "v0.9.9", errors.New("no network"), errors.New("no network"))
	running := filepath.Join(t.TempDir(), "elsewhere", "xui"+exeExt())
	err := selfUpdateAt(running, "v0.8.0", "v0.9.9", false)
	if err == nil || !strings.Contains(err.Error(), "手动跑: go install "+cliModule+"@v0.9.9") {
		t.Errorf("应给出手动升级命令, 实为 %v", err)
	}

	err = selfUpdateAt("/tmp/go-build7/b001/exe/xui", "v0.8.0", "", false)
	if err == nil || !strings.Contains(err.Error(), "临时构建") {
		t.Errorf("go run 的临时二进制应被拒绝, 实为 %v", err)
	}

	err = selfUpdateAt(running, "v0.8.0", "not-a-version", false)
	if err == nil || !strings.Contains(err.Error(), "认不出版本") {
		t.Errorf("非法目标应报错, 实为 %v", err)
	}
}

// 取不到 GOBIN 时: go install 照跑, 但不谎报"已升级"。
func TestSelfUpdateWithoutGoBin(t *testing.T) {
	specs := stubSelfUpdate(t, "", "v0.9.9", "v0.9.9")
	out := captureOutput(t, func() {
		if err := selfUpdateAt("/home/u/go/bin/xui", "v0.8.0", "", false); err != nil {
			t.Errorf("不该失败: %v", err)
		}
	})
	if len(*specs) != 1 {
		t.Fatalf("仍应跑一次 go install, 实为 %v", *specs)
	}
	if !strings.Contains(out, "取不到 GOBIN") || strings.Contains(out, "已升级") {
		t.Errorf("输出不对: %q", out)
	}
}

// 升级提示只看缓存: 有新版才提示, devel/损坏/已是最新/查不到都不唠叨。
func TestUpdateNotice(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		current string
		cache   updateCheck
		want    string // "" = 不该提示
	}{
		{"有新版", "v0.8.0", updateCheck{CheckedAt: now, Latest: "v0.8.1"}, "xui 有新版 v0.8.0 → v0.8.1"},
		{"已是最新", "v0.8.1", updateCheck{CheckedAt: now, Latest: "v0.8.1"}, ""},
		{"本地比缓存新", "v0.9.0", updateCheck{CheckedAt: now, Latest: "v0.8.1"}, ""},
		{"devel 不提示", "(devel) f7072ad+dirty", updateCheck{CheckedAt: now, Latest: "v0.8.1"}, ""},
		{"没查到版本", "v0.8.0", updateCheck{CheckedAt: now}, ""},
		{"缓存里是垃圾", "v0.8.0", updateCheck{CheckedAt: now, Latest: "latest"}, ""},
	}
	for _, c := range cases {
		got := updateNotice(c.current, c.cache, now)
		if c.want == "" {
			if got != "" {
				t.Errorf("%s: 不该提示, 实为 %q", c.name, got)
			}
			continue
		}
		if !strings.HasPrefix(got, c.want) {
			t.Errorf("%s: %q, 期望以 %q 开头", c.name, got, c.want)
		}
		// 提示里要带上自查、升级与关闭的入口
		for _, want := range []string{"self-update --check", "xui self-update", "XUI_NO_UPDATE_CHECK"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: 提示缺少 %q: %q", c.name, want, got)
			}
		}
	}
}

func TestShouldProbeUpdate(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		cache updateCheck
		want  bool
	}{
		{"从没查过", updateCheck{}, true},
		{"23 小时前", updateCheck{CheckedAt: now.Add(-23 * time.Hour)}, false},
		{"25 小时前", updateCheck{CheckedAt: now.Add(-25 * time.Hour)}, true},
		{"时间戳在未来(时钟回跳)", updateCheck{CheckedAt: now.Add(time.Hour)}, true},
	}
	for _, c := range cases {
		if got := shouldProbeUpdate(c.cache, now); got != c.want {
			t.Errorf("%s: %v, 期望 %v", c.name, got, c.want)
		}
	}
}

// 缓存坏了要能退回"没查过", 而不是让每条命令崩在半路上。
func TestUpdateCheckCacheRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "update-check.json")
	saveUpdateCheck(path, updateCheck{CheckedAt: time.Now(), Latest: "v0.8.1"})
	got := loadUpdateCheck(path)
	if !strings.Contains(string(mustJSON(t, path)), `"latest":"v0.8.1"`) {
		t.Errorf("缓存内容不对: %s", mustJSON(t, path))
	}
	if !strings.HasPrefix(got.Latest, "v0.8.1") || got.CheckedAt.IsZero() {
		t.Errorf("读回不对: %+v", got)
	}
	if err := os.WriteFile(path, []byte("{ broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if c := loadUpdateCheck(path); !c.CheckedAt.IsZero() || c.Latest != "" {
		t.Errorf("坏缓存应退回空, 实为 %+v", c)
	}
	if c := loadUpdateCheck(filepath.Join(t.TempDir(), "missing.json")); !c.CheckedAt.IsZero() {
		t.Errorf("缺失缓存应退回空, 实为 %+v", c)
	}
}

func mustJSON(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// 缓存路径落在用户缓存目录下的 xui/ 里(拿不到就退 Temp), 不能污染项目目录。
func TestUpdateCheckPath(t *testing.T) {
	p := updateCheckPath()
	if filepath.Base(p) != "update-check.json" || filepath.Base(filepath.Dir(p)) != "xui" {
		t.Errorf("路径不对: %s", p)
	}
}
