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
	if err := initNewProject("sitless", "ui", false, writeOpts{noInstall: true}); err != nil {
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
	if err := initNewProject(".", "ui", false, writeOpts{noInstall: true}); err != nil {
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
		// init -n 也带组件目录名(默认 ui): 以前这条是空的, 目录名被写死在 initNewProject 里
		{"-n", []string{"-n", "myapp"}, initOptions{uiDir: "ui", project: "myapp"}, false},
		{"--name=", []string{"--name=myapp"}, initOptions{uiDir: "ui", project: "myapp"}, false},
		{"-n 加 force", []string{"-f", "-n", "myapp"}, initOptions{uiDir: "ui", project: "myapp", force: true}, false},
		// force 与 overwrite 分开: -f 只开"非空目录"这道闸门, -o 才允许丢弃本地改动
		{"-n 加 -o", []string{"-n", "myapp", "-o"}, initOptions{uiDir: "ui", project: "myapp", overwrite: true}, false},
		{"-o 与 -f 各自独立", []string{"--force", "--overwrite", "--name=myapp"}, initOptions{uiDir: "ui", project: "myapp", force: true, overwrite: true}, false},
		// init -n 的 -p: 组件目录名在这一步就能定, 不用再 init 完手工删 ui/
		{"-n 加 -p", []string{"-n", "myapp", "-p", "components"}, initOptions{uiDir: "components", project: "myapp"}, false},
		{"-n 加 --path=", []string{"-n", "myapp", "--path=internal/ui"}, initOptions{uiDir: "internal/ui", project: "myapp"}, false},
		{"-n 的 -p 不许绝对路径", []string{"-n", "myapp", "-p", "D:/other/ui"}, initOptions{}, true},
		{"-n 的 -p 不许跳出项目", []string{"-n", "myapp", "-p", "../ui"}, initOptions{}, true},
		{"普通 init 不需要 -f", []string{"-f"}, initOptions{}, true},
		{"普通 init 的 -o", []string{"-o"}, initOptions{uiDir: "ui", overwrite: true}, false},
		// 普通 init 的位置参数与 -p 是同一个东西: 同词不矛盾, 不同词不猜
		{"普通 init 的 -p", []string{"-p", "widgets"}, initOptions{uiDir: "widgets"}, false},
		{"普通 init -p 与位置同值", []string{"-p", "widgets", "widgets"}, initOptions{uiDir: "widgets"}, false},
		{"普通 init -p 与位置冲突", []string{"-p", "widgets", "ui"}, initOptions{}, true},
		{"--no-install", []string{"--no-install", "-n", "myapp"}, initOptions{uiDir: "ui", project: "myapp", noInstall: true}, false},
		// 普通 init 也会抬配套运行时依赖(add 路径), 所以 --no-install 对它同样成立
		{"--no-install 无 -n", []string{"--no-install"}, initOptions{uiDir: "ui", noInstall: true}, false},
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

	if err := initNewProject("myapp", "ui", false, writeOpts{noInstall: true}); err != nil {
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
	if err := initNewProject("myapp", "ui", false, writeOpts{noInstall: true}); err == nil {
		t.Error("非空目录未加 --force 应报错")
	}
	keep := filepath.Join(dir, "myapp", "notes.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := initNewProject("myapp", "ui", true, writeOpts{noInstall: true}); err != nil {
		t.Fatalf("--force 应允许写入非空目录: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Error("--force 不应删除无关文件")
	}
}

// --force 只负责开"目录已存在且非空"这道闸门, 丢弃本地改动是 -o 的决定。
// 以前一个 -f 两件事都干: 组件文件有安装基线所以措辞正确, 而 go.mod/main.go/README.md
// 这五个模板是无条件覆写 —— 实测能无声抹掉用户改过的入口文件。
func TestInitForceKeepsEditedTemplates(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
	stubWinresMake(t)
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	if err := initNewProject("app", "ui", false, writeOpts{noInstall: true}); err != nil {
		t.Fatalf("首次 init 失败: %v", err)
	}
	mainGo := filepath.Join("app", "main.go")
	if err := os.WriteFile(mainGo, []byte("package main\n// 我的入口\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 删掉清单 = 老项目 / 手删: 没有基线时只能说"判断不了来源", 不能断言用户改过
	if err := os.Remove(filepath.Join("app", "ui", manifestName)); err != nil {
		t.Fatal(err)
	}

	// 只加 --force: 闸门开了, 但你的文件一个都不动
	if err := initNewProject("app", "ui", true, writeOpts{noInstall: true}); err != nil {
		t.Fatalf("--force 应允许写入非空目录: %v", err)
	}
	if data, _ := os.ReadFile(mainGo); string(data) != "package main\n// 我的入口\n" {
		t.Errorf("--force 不该丢弃本地改动, 实为 %q", data)
	}

	// --force + -o: 这才是"确认丢弃改动"
	if err := initNewProject("app", "ui", true, writeOpts{overwrite: true, noInstall: true}); err != nil {
		t.Fatalf("--force -o 应覆盖: %v", err)
	}
	if data, _ := os.ReadFile(mainGo); string(data) == "package main\n// 我的入口\n" {
		t.Error("-o 应覆盖本地改动")
	}
	// 覆盖之后基线重建: 再跑一次是"模板没变", 不该继续说本地修改已丢弃
	out := captureOutput(t, func() {
		if err := initNewProject("app", "ui", true, writeOpts{overwrite: true, noInstall: true}); err != nil {
			t.Errorf("重复 init 失败: %v", err)
		}
	})
	if strings.Contains(out, "本地修改已丢弃") {
		t.Errorf("干净文件在 -o 下不该说丢弃了改动: %q", out)
	}

	// 有基线时措辞要指名"本地已修改"(上面删过清单, 这里补一条正常路径的断言)
	if err := os.WriteFile(filepath.Join("app", "README.md"), []byte("# 我的说明\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out = captureOutput(t, func() {
		if err := initNewProject("app", "ui", true, writeOpts{noInstall: true}); err != nil {
			t.Errorf("--force 失败: %v", err)
		}
	})
	if !strings.Contains(out, "README.md 本地已修改") {
		t.Errorf("有基线时该报本地已修改并跳过, 实为 %q", out)
	}
	if data, _ := os.ReadFile(filepath.Join("app", "README.md")); string(data) != "# 我的说明\n" {
		t.Error("README.md 不该被覆写")
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
	initProject(dir, writeOpts{})

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
	initProject(dir, writeOpts{})
	if got, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md")); string(got) != "custom" {
		t.Error("重复 init 覆盖了已有的 AGENTS.md")
	}
}

func TestParseComponentArgs(t *testing.T) {
	got, err := parseComponentArgs("add", []string{"-o", "-p", "components", "button", "card"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.overwrite || got.uiDir != "components" || len(got.names) != 2 {
		t.Errorf("解析结果不对: %+v", got)
	}
	if got, err := parseComponentArgs("add", []string{"--all"}); err != nil || !got.all {
		t.Errorf("--all 解析失败: %+v, %v", got, err)
	}
	// --no-install 对 add/update 也成立: 写组件但不动 go.mod
	if got, err := parseComponentArgs("update", []string{"--no-install", "button"}); err != nil ||
		!got.noInstall || got.dryRun {
		t.Errorf("--no-install 解析失败: %+v, %v", got, err)
	}
	if _, err := parseComponentArgs("add", []string{"--wat"}); err == nil {
		t.Error("未知选项应报错")
	}
	// 旧用例: --dry-run 与 --view 同时给时, switch 里 view 先赢、dry-run 静默失效
	for _, args := range [][]string{
		{"--dry-run", "--view"},
		{"-o", "--diff"},
		{"--no-install", "--view=button.go"},
	} {
		if _, err := parseComponentArgs("add", args); err == nil {
			t.Errorf("%v: --view/--diff 不写文件, 与写文件的开关同时给应报错", args)
		}
	}
	if got, err := parseComponentArgs("add", []string{"--dry-run", "button"}); err != nil || !got.dryRun {
		t.Errorf("--dry-run 单独给应可用: %+v, %v", got, err)
	}
	if got, err := parseComponentArgs("add", []string{"--view=button.go", "button"}); err != nil ||
		!got.view || got.viewFile != "button.go" {
		t.Errorf("--view=文件 解析失败: %+v, %v", got, err)
	}
	// update --all 实际含义是"把整套 registry 拷进来"(= add), 不该由 update 悄悄做
	if _, err := parseComponentArgs("update", []string{"--all"}); err == nil {
		t.Error("update 不该接受 --all")
	}
	if _, err := parseComponentArgs("update", []string{"button"}); err != nil {
		t.Errorf("update 带组件名应可用: %v", err)
	}
	if _, err := parseComponentArgs("add", []string{"--view", "--diff", "button"}); err == nil {
		t.Error("--view 与 --diff 应互斥")
	}
	if _, err := parseComponentArgs("add", []string{"--all", "button"}); err == nil {
		t.Error("--all 与组件名应互斥")
	}
	// 短选项也接受 = 形式(与 --path= 一致)
	if got, err := parseComponentArgs("add", []string{"-p=src", "button"}); err != nil || got.uiDir != "src" {
		t.Errorf("-p=src 解析失败: %+v, %v", got, err)
	}
	// 值位置紧跟另一个选项时要报缺值, 不能把 --all 当成目录名
	if _, err := parseComponentArgs("add", []string{"-p", "--all"}); err == nil {
		t.Error("-p 后紧跟选项应报缺值")
	}
	// 空值一律报错(两种写法同规则): 值被 shell 变量吃掉时不许静默退化成"少个过滤器"
	for _, args := range [][]string{
		{"-p=", "button"}, {"--path=", "button"}, {"--view=", "button"}, {"--diff=", "button"},
		{"-p", "", "button"},
	} {
		if _, err := parseComponentArgs("add", args); err == nil {
			t.Errorf("%v 空值应报错", args)
		}
	}
	// 上游 shadcn 是 add -v, --view; 这里 -v 归了全局版本号, 报错要说清去向
	if _, err := parseComponentArgs("add", []string{"-v", "button"}); err == nil ||
		!strings.Contains(err.Error(), "--view") {
		t.Errorf("-v 的报错该指向 --view, 实为 %v", err)
	}
}

func TestParseListArgsPositionalQuery(t *testing.T) {
	// `xui search button` 是最自然的写法(以前它报"未知参数 button")
	got, err := parseListArgs([]string{"button"})
	if err != nil || got.query != "button" {
		t.Errorf("位置参数应当搜索词: %+v, %v", got, err)
	}
	if got, err := parseListArgs([]string{"table", "-l", "5"}); err != nil || got.query != "table" || got.limit != 5 {
		t.Errorf("位置参数与选项混用失败: %+v, %v", got, err)
	}
	// -q 与位置参数一致, 两个都给且不同不猜
	if got, err := parseListArgs([]string{"-q", "table", "table"}); err != nil || got.query != "table" {
		t.Errorf("同词给两次不该报错: %+v, %v", got, err)
	}
	if _, err := parseListArgs([]string{"-q", "table", "card"}); err == nil {
		t.Error("搜索词给了两次应报错")
	}
	if _, err := parseListArgs([]string{"button", "card"}); err == nil {
		t.Error("给两个搜索词应报错")
	}
	// 空值两种写法都拒
	for _, args := range [][]string{{"-q="}, {"--query="}, {"-l="}, {"-o="}, {"-q", ""}} {
		if _, err := parseListArgs(args); err == nil {
			t.Errorf("%v 空值应报错", args)
		}
	}
}

func TestParseInfoArgs(t *testing.T) {
	// info 报的是"我的项目里 xui 装成什么样", 组件目录不是 ui 时得能指过去
	if dir, j, err := parseInfoArgs(nil); err != nil || dir != "ui" || j {
		t.Errorf("默认组件目录 = ui: %q, %v", dir, err)
	}
	for _, args := range [][]string{{"-p", "components"}, {"-p=components"}, {"--path=components"}} {
		if dir, j, err := parseInfoArgs(args); err != nil || dir != "components" || j {
			t.Errorf("%v 解析失败: %q, %v", args, dir, err)
		}
	}
	for _, args := range [][]string{{"--wat"}, {"button"}, {"-p"}, {"-p="}, {"-p", ""}} {
		if _, _, err := parseInfoArgs(args); err == nil {
			t.Errorf("%v 应报错", args)
		}
	}
}

func TestParseSelfUpdateArgs(t *testing.T) {
	if target, check, j, err := parseSelfUpdateArgs(nil); err != nil || target != "" || check || j {
		t.Errorf("无参数 = 升到 latest: %q %v %v", target, check, err)
	}
	if target, check, j, err := parseSelfUpdateArgs([]string{"v0.8.1"}); err != nil || target != "v0.8.1" || check || j {
		t.Errorf("版本参数解析失败: %q %v %v", target, check, err)
	}
	if target, check, j, err := parseSelfUpdateArgs([]string{"--check"}); err != nil || target != "" || !check || j {
		t.Errorf("--check 解析失败: %q %v %v", target, check, err)
	}
	// --check 只回答"我相对 @latest 新不新"; 收下版本又丢掉会让人以为它查的是那个版本
	if _, _, _, err := parseSelfUpdateArgs([]string{"v9.9.9", "--check"}); err == nil {
		t.Error("--check 与版本同时给应报错")
	}
	if _, _, _, err := parseSelfUpdateArgs([]string{"--wat"}); err == nil {
		t.Error("未知选项应报错")
	}
	if _, _, _, err := parseSelfUpdateArgs([]string{"v0.8.0", "v0.8.1"}); err == nil {
		t.Error("给两个版本应报错")
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
	// 裸位置参数现在是搜索词(见 TestParseListArgsPositionalQuery), 这里只留真正的坏输入
	for _, args := range [][]string{{"-l", "0"}, {"-l", "x"}, {"-o", "-1"}} {
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
	names, j, err := parseViewArgs([]string{"button", "card"})
	if err != nil || len(names) != 2 || j {
		t.Errorf("view 参数解析失败: %v json=%v, %v", names, j, err)
	}
	// -j 是"结果改结构化", 与组件名并列接受(两种写法)
	if names, j, err := parseViewArgs([]string{"button", "-j"}); err != nil || len(names) != 1 || !j {
		t.Errorf("-j 解析失败: %v json=%v, %v", names, j, err)
	}
	if _, j, err := parseViewArgs([]string{"button", "--json"}); err != nil || !j {
		t.Errorf("--json 解析失败: json=%v, %v", j, err)
	}
	if _, _, err := parseViewArgs(nil); err == nil {
		t.Error("view 缺少组件名应报错")
	}
	if _, _, err := parseViewArgs([]string{"--wat"}); err == nil {
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

// -s/--silent 该压的是进度, 不是命令本身的结果: list / view / info / self-update --check
// 的正文以前全走 info, 加 -s 就是 0 字节 + 退出码 0, 脚本与 AI 助手读到的是"什么都没发生"。
func TestSilentKeepsCommandResults(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	quiet = true
	defer func() { quiet = false }()

	if got := captureOutput(t, func() { list(listOptions{limit: 5}) }); !strings.Contains(got, "button") {
		t.Errorf("-s 下 list 的组件表被吞了: %q", got)
	}
	if got := captureOutput(t, func() { infoCommand("ui", false) }); !strings.Contains(got, "工作目录") {
		t.Errorf("-s 下 info 的报告被吞了: %q", got)
	}
	if got := captureOutput(t, func() { view([]string{"button"}, false) }); !strings.Contains(got, "package ui") {
		t.Errorf("-s 下 view 的源码被吞了: %q", got)
	}
	stubSelfUpdate(t, filepath.Join(t.TempDir(), "bin"), "v0.9.9", "v0.9.9")
	if got := captureOutput(t, func() { _ = selfUpdateAt("/home/u/go/bin/xui", "v0.8.0", "", true, false) }); !strings.Contains(got, "最新 tag") {
		t.Errorf("-s 下 --check 的结果被吞了: %q", got)
	}
	// 进度输出仍然闭嘴
	if got := captureOutput(t, func() { info("创建 %s", "ui/button.go") }); got != "" {
		t.Errorf("进度输出不该在 -s 下出现: %q", got)
	}
}

// --view=/--diff= 的过滤值既接受文件基名也接受组件名(用户刚在命令里打过它);
// 一个都没匹配上时必须说"没有匹配的文件" —— 答"没有差异"等于骗人说已经是最新的。
func TestDiffAndViewFilterTakeComponentName(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
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

	if got := captureOutput(t, func() {
		add(componentOptions{names: []string{"button"}, uiDir: "ui", diff: true, diffFile: "button"})
	}); !strings.Contains(got, "没有差异") {
		t.Errorf("--diff=button 应按组件名命中并报告没有差异: %q", got)
	}
	if got := captureOutput(t, func() {
		add(componentOptions{names: []string{"button"}, uiDir: "ui", diff: true, diffFile: "card.go"})
	}); strings.Contains(got, "没有差异") || !strings.Contains(got, "没有匹配的文件") {
		t.Errorf("--diff 没命中任何文件, 不该说没有差异: %q", got)
	}
	if got := captureOutput(t, func() {
		add(componentOptions{names: []string{"button"}, uiDir: "ui", view: true, viewFile: "button"})
	}); !strings.Contains(got, "package ui") || strings.Contains(got, "没有匹配的文件") {
		t.Errorf("--view=button 应打出源码: %q", got)
	}
}

// -j/--json 是给脚本与 AI 助手读的结果: 字段稳定、空的是 [] 不是 null、
// "过滤没命中"用 matched=false 表达(不能拿空 files 冒充"这个组件没有文件/没有差异")。
func TestJSONOutputShapes(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	out := captureOutput(t, func() { list(listOptions{json: true, query: "button", limit: 2}) })
	var lj listJSON
	if err := json.Unmarshal([]byte(out), &lj); err != nil {
		t.Fatalf("list -j 不是合法 JSON: %v\n%s", err, out)
	}
	// 只认 limit 的语义(命中够 2 条就必须恰好显示 2 条), 不写死注册表里有几个 button
	if lj.Total < 2 || lj.Shown != 2 || len(lj.Components) != lj.Shown || lj.Query != "button" {
		t.Errorf("list -j 切片字段不对: total=%d shown=%d 条目=%d", lj.Total, lj.Shown, len(lj.Components))
	}
	for _, c := range lj.Components {
		if c.Files == nil || c.Deps == nil {
			t.Errorf("%s 的 files/deps 不该是 null", c.Name)
		}
	}
	if lj.Runtime == "" {
		t.Error("list -j 该带上数据源声明的配套运行时版本")
	}

	// info -j: 组件目录跟着 -p 走, 没 init 时是 false 而不是报错
	out = captureOutput(t, func() { infoCommand("components", true) })
	var ij infoJSON
	if err := json.Unmarshal([]byte(out), &ij); err != nil {
		t.Fatalf("info -j 不是合法 JSON: %v\n%s", err, out)
	}
	if ij.ComponentsDir != "components" || ij.ComponentsInitialized || ij.GoFiles != 0 {
		t.Errorf("info -j 的目录字段不对: %+v", ij)
	}

	add(componentOptions{names: []string{"button"}, uiDir: "ui"})
	diffJSONAt := func(filter string) diffJSON {
		var got diffJSON
		out := captureOutput(t, func() {
			add(componentOptions{names: []string{"button"}, uiDir: "ui", diff: true, diffFile: filter, json: true})
		})
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("--diff=%s -j 不是合法 JSON: %v\n%s", filter, err, out)
		}
		return got
	}
	// 装完之后内容与远端一致 → 没有差异, 但 matched 必须是 true(命中了文件)
	if got := diffJSONAt("button"); !got.Matched || len(got.Files) != 0 {
		t.Errorf("一致时该 matched=true 且 files 为空: %+v", got)
	}
	// 本地改过 → state=edited, 带 diff 正文
	data, err := os.ReadFile(filepath.Join("ui", "button.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("ui", "button.go"), append(data, []byte("\n// mine\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	got := diffJSONAt("button")
	if !got.Matched || len(got.Files) != 1 || got.Files[0].State != "edited" || got.Files[0].Diff == "" {
		t.Fatalf("改过的文件该报 edited 并带 diff: %+v", got)
	}
	// 删掉清单 → 没有基线, 只能说判断不了(与人类可读输出同一套判定)
	if err := os.Remove(filepath.Join("ui", manifestName)); err != nil {
		t.Fatal(err)
	}
	if got := diffJSONAt("button"); len(got.Files) != 1 || got.Files[0].State != "unknown" {
		t.Errorf("无基线时 state 该是 unknown: %+v", got)
	}
	// 过滤值谁都没挨着 → matched=false
	if got := diffJSONAt("nomatch.go"); got.Matched || len(got.Files) != 0 {
		t.Errorf("没命中该报 matched=false: %+v", got)
	}

	// view 同理
	var vj viewJSON
	out = captureOutput(t, func() {
		add(componentOptions{names: []string{"button"}, uiDir: "ui", view: true, viewFile: "button.go", json: true})
	})
	if err := json.Unmarshal([]byte(out), &vj); err != nil {
		t.Fatalf("view -j 不是合法 JSON: %v\n%s", err, out)
	}
	if !vj.Matched || len(vj.Files) != 1 || !strings.Contains(vj.Files[0].Content, "package ui") || vj.Files[0].Component != "button" {
		t.Errorf("view -j 内容不对: matched=%v files=%d", vj.Matched, len(vj.Files))
	}
}

// --json 只在"结果本身有结构化意义"的命令上接受; 收下又不生效就是骗人。
func TestJSONFlagOnlyWhereItDoesSomething(t *testing.T) {
	if got, err := parseListArgs([]string{"-j"}); err != nil || !got.json {
		t.Errorf("list -j 应可用: %+v, %v", got, err)
	}
	if _, j, err := parseInfoArgs([]string{"--json"}); err != nil || !j {
		t.Errorf("info --json 应可用: %v", err)
	}
	if got, err := parseComponentArgs("add", []string{"--view", "-j", "button"}); err != nil || !got.json {
		t.Errorf("--view -j 应可用: %+v, %v", got, err)
	}
	if _, err := parseComponentArgs("add", []string{"--json", "button"}); err == nil {
		t.Error("add 不带 --view/--diff 时 --json 应报错(写文件那条路径没有结构化输出)")
	}
	if _, _, j, err := parseSelfUpdateArgs([]string{"--check", "--json"}); err != nil || !j {
		t.Errorf("--check --json 应可用: %v", err)
	}
	if _, _, _, err := parseSelfUpdateArgs([]string{"--json"}); err == nil {
		t.Error("self-update --json 不带 --check 应报错")
	}
}

// --check 的 JSON 报告: command 是"下一步该跑什么", 已最新时为空串。
func TestSelfUpdateCheckJSON(t *testing.T) {
	cases := []struct {
		current, newest  string
		localBuild, upgr bool
		command          string
	}{
		{"v0.8.0", "v0.9.9", false, true, "xui self-update v0.9.9"},
		{"v0.9.9", "v0.9.9", false, false, ""},
		{"v0.9.9", "v0.8.0", false, false, ""}, // 本地比 tag 新(开发者常态)
		{"(devel) abc123", "v0.9.9", true, false, "xui self-update"},
	}
	for _, c := range cases {
		stubSelfUpdate(t, filepath.Join(t.TempDir(), "bin"), c.newest, c.newest)
		var got checkJSON
		out := captureOutput(t, func() {
			if err := selfUpdateAt("/home/u/go/bin/xui", c.current, "", true, true); err != nil {
				t.Errorf("不该失败: %v", err)
			}
		})
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("current=%s 时不是合法 JSON: %v\n%s", c.current, err, out)
		}
		if got.Current != c.current || got.Latest != c.newest || got.LocalBuild != c.localBuild ||
			got.Upgradable != c.upgr || got.Command != c.command {
			t.Errorf("%s vs %s: %+v", c.current, c.newest, got)
		}
	}
}

// 有文件因本地改动被跳过时的输出契约: 默认只向 stderr 汇总一句就正常退出(保守跳过不是
// 故障), --strict 才以退出码 2 失败 —— 与 fatal 的 1 分开, CI 才能区分"跑挂了"和
// "跑通了但没更新干净"。汇总与退出码都走 os.Exit, 只能起子进程测。
func TestSkippedSummaryAndStrictExitCode(t *testing.T) {
	if dir := os.Getenv("XUI_TEST_SKIP_DIR"); dir != "" {
		if err := os.Chdir(dir); err != nil {
			os.Exit(99)
		}
		os.Setenv("XUI_REGISTRY", moduleRoot())
		os.Setenv("XUI_NO_UPDATE_CHECK", "1")
		add(componentOptions{names: []string{"button"}, uiDir: "ui"})
		p := filepath.Join(dir, "ui", "button.go")
		data, err := os.ReadFile(p)
		if err != nil {
			os.Exit(99)
		}
		if err := os.WriteFile(p, append(data, []byte("\n// mine\n")...), 0o644); err != nil {
			os.Exit(99)
		}
		add(componentOptions{names: []string{"button"}, uiDir: "ui", strict: os.Getenv("XUI_TEST_STRICT") == "1"})
		os.Exit(0) // 非 strict: 汇总完就该正常结束
	}

	dir := t.TempDir() // 由父进程建目录, 好让断言读到子进程留下的文件
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	run := func(strict bool) (string, int) {
		cmd := exec.Command(exe, "-test.run=TestSkippedSummaryAndStrictExitCode")
		env := append(os.Environ(), "XUI_TEST_SKIP_DIR="+dir)
		if strict {
			env = append(env, "XUI_TEST_STRICT=1")
		}
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("子进程没跑起来: %v\n%s", err, out)
		}
		return string(out), code
	}

	out, code := run(false)
	if code != 0 {
		t.Fatalf("默认不该失败, 退出码 = %d\n%s", code, out)
	}
	if !strings.Contains(out, "共 1 个文件被跳过") || !strings.Contains(out, "-o/--overwrite") {
		t.Errorf("默认该给一句可操作的汇总: %s", out)
	}
	// 逐文件原因要说"本地已修改"(而不是断言用户改错了), 且内容真的没被覆写
	if !strings.Contains(out, "button.go 本地已修改, 跳过") {
		t.Errorf("缺少逐文件的跳过原因: %s", out)
	}
	data, err := os.ReadFile(filepath.Join(dir, "ui", "button.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "// mine") {
		t.Error("跳过就该保住本地改动, 但文件被覆写了")
	}

	out, code = run(true)
	if code != exitPartial {
		t.Fatalf("--strict 该用退出码 %d, 实为 %d\n%s", exitPartial, code, out)
	}
	if !strings.Contains(out, "共 1 个文件被跳过") {
		t.Errorf("--strict 也要先给汇总: %s", out)
	}
}

// init --dry-run 必须真的没有副作用: 不建目录、不写文件、不调 xc、不动依赖 ——
// 预览如果有副作用, 就没人敢在别人的项目里跑它。
func TestInitDryRunWritesNothing(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
	calls := stubWinresMake(t)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	out := captureOutput(t, func() {
		if err := initNewProject("app", "ui", false, writeOpts{dryRun: true}); err != nil {
			t.Errorf("预览失败: %v", err)
		}
	})
	if _, err := os.Stat(filepath.Join(dir, "app")); !os.IsNotExist(err) {
		t.Error("--dry-run 不该创建项目目录")
	}
	for _, want := range []string{"将创建", "go mod tidy", "预览"} {
		if !strings.Contains(out, want) {
			t.Errorf("预览缺少 %q:\n%s", want, out)
		}
	}
	// 预览里不许出现"已经动过文件"的措辞(已一致 是描述当前状态, 两种模式都成立, 允许)
	if regexp.MustCompile(`(?m)^(创建|覆盖|更新) `).MatchString(out) {
		t.Errorf("预览不该用完成时态:\n%s", out)
	}
	if len(*calls) != 0 {
		t.Error("--dry-run 不该调 xc 生成 syso")
	}

	// 已建好的项目里预览: 改过的模板要说"将跳过", 而且内容不许被动
	// (setup 用 noInstall, 单测绝不联网跑 go mod tidy)
	if err := initNewProject("app", "ui", false, writeOpts{noInstall: true}); err != nil {
		t.Fatal(err)
	}
	mainGo := filepath.Join("app", "main.go")
	if err := os.WriteFile(mainGo, []byte("package main\n// 我的入口\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out = captureOutput(t, func() {
		if err := initNewProject("app", "ui", true, writeOpts{dryRun: true}); err != nil {
			t.Errorf("预览失败: %v", err)
		}
	})
	if !strings.Contains(out, "main.go 本地已修改, 将跳过") {
		t.Errorf("预览该把跳过说成将跳过:\n%s", out)
	}
	if data, _ := os.ReadFile(mainGo); string(data) != "package main\n// 我的入口\n" {
		t.Errorf("--dry-run 覆写了本地改动: %q", data)
	}

	// 普通 init 同样要能预览(它以前完全没有 dry-run)
	out = captureOutput(t, func() {
		initProject("widgets", writeOpts{dryRun: true})
	})
	if _, err := os.Stat(filepath.Join(dir, "widgets")); !os.IsNotExist(err) {
		t.Error("普通 init 的 --dry-run 也不该建目录")
	}
	if !strings.Contains(out, "将创建") {
		t.Errorf("普通 init 预览缺少将创建:\n%s", out)
	}
}

// init -n 的 -p 必须贯通到下发的每一份文件: 目录、清单、main.go 的 import、README/AGENTS
// 里说的路径。以前组件目录写死 ui/, 想要 components/ 只能 init 完手工删一个空 ui/ 再改文档。
func TestInitNewProjectCustomComponentsDir(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
	stubWinresMake(t)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	if err := initNewProject("app", "components", false, writeOpts{noInstall: true}); err != nil {
		t.Fatalf("init -n -p components 失败: %v", err)
	}
	root := filepath.Join(dir, "app")
	// 组件与清单都落在 components/, 而且不该再有 ui/
	for _, f := range []string{"components/theme.go", "components/DESIGN.md", "components/button.go", "components/" + manifestName} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(f))); err != nil {
			t.Errorf("缺少 %s: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "ui")); !os.IsNotExist(err) {
		t.Error("自定义目录名时不该再生成 ui/")
	}
	// 模板文件基线记在组件目录里那份清单上
	if data, err := os.ReadFile(filepath.Join(root, "components", manifestName)); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(string(data), `"project_files"`) {
		t.Error("清单里该有模板文件的基线")
	}
	// import 路径跟着目录名走(包名仍是 ui, 靠别名)
	mainGo, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainGo), `ui "app/components"`) {
		t.Errorf("main.go 的 import 没跟着改:\n%s", mainGo)
	}
	// 下发文档说的路径也得是真的(占位符不许漏在最后成品里)
	for _, f := range []string{"README.md", "AGENTS.md", "main.go"} {
		body, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), uiDirToken) {
			t.Errorf("%s 里漏了未替换的占位符", f)
		}
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "- components/ — 组件源码拷贝") {
		t.Errorf("README 的目录清单没跟着改:\n%s", readme)
	}
	// 之后每条命令用 -p 指得回去
	addAt := captureOutput(t, func() {
		add(componentOptions{names: []string{"card"}, uiDir: filepath.Join("app", "components")})
	})
	if !strings.Contains(addAt, "card.go") {
		t.Errorf("往自定义目录 add 失败: %s", addAt)
	}
	if _, err := os.Stat(filepath.Join(root, "components", "card.go")); err != nil {
		t.Errorf("card.go 没落到 components/: %v", err)
	}
}

// findGoMod 是"从组件目录向上找最近的 go.mod", 所以在临时目录里跑 CLI 会命中某个**无关**的
// 父模块并去改它的依赖。判据: 那份 go.mod 在当前目录之下(用户站着的项目), 或者它本来就
// require 了 github.com/twgh/xui —— 两条都不满足就只提示不动。
func TestEnsureRuntimeDoesNotTouchForeignGoMod(t *testing.T) {
	cases := []struct {
		name    string
		parent  string // 父目录(当前目录之外)的 go.mod
		wantGet bool
	}{
		{
			"无关父模块: 没要求 xui",
			"module someone/else\n\ngo 1.18\n",
			false,
		},
		{
			"父模块本来就用 xui: 该抬(在项目子目录里跑的形态)",
			"module someone/else\n\ngo 1.18\n\nrequire github.com/twgh/xui v0.7.0\n",
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calls := stubGoGet(t)
			old, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(c.parent), 0o644); err != nil {
				t.Fatal(err)
			}
			// cwd 是 root 的子目录, 组件目录在 cwd 下 —— 唯一的 go.mod 在 cwd **之外**
			sub := filepath.Join(root, "scratch")
			if err := os.MkdirAll(filepath.Join(sub, "ui"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(sub); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(old)

			out := captureOutput(t, func() { ensureRuntime(filepath.Join(sub, "ui"), "v0.8.0", false, true) })
			if c.wantGet {
				if len(*calls) != 1 || (*calls)[0].spec != "github.com/twgh/xui@v0.8.0" {
					t.Errorf("该抬但没抬, calls=%+v\n%s", *calls, out)
				}
				return
			}
			if len(*calls) != 0 {
				t.Fatalf("不该动无关模块, calls=%+v", *calls)
			}
			if !strings.Contains(out, "在当前目录之外") || !strings.Contains(out, "不动它的依赖") {
				t.Errorf("该说清为什么不动, 实为 %q", out)
			}
		})
	}
}

// 正常项目里(go.mod 就是当前目录)必须照旧抬起来 —— 上面那条守卫不能过头。
func TestEnsureRuntimeStillBumpsProjectRoot(t *testing.T) {
	calls := stubGoGet(t)
	chdirProject(t, "module myapp\n\ngo 1.18\n\nrequire github.com/twgh/xui v0.7.0\n")
	ensureRuntime("ui", "v0.8.0", false, true)
	if len(*calls) != 1 || (*calls)[0].spec != "github.com/twgh/xui@v0.8.0" {
		t.Fatalf("项目根的依赖该抬上去, calls=%+v", *calls)
	}
}

// 组件目录里的 DESIGN.md / AGENTS.md 以前是"存在就永远跳过": 第一次 init 那一刻定版,
// 之后 CLI 修了文档里的错也传不到已有项目。现在与组件源码共用同一套基线判定。
// theme.go 故意不纳入(它就是给用户改的), 所以它仍是"存在就跳过"。
func TestInitRefreshesTrackedDocs(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	initProject("ui", writeOpts{})
	agents := filepath.Join("ui", "AGENTS.md")
	design := filepath.Join("ui", "DESIGN.md")
	for _, f := range []string{agents, design} {
		if !fileExists(f) {
			t.Fatalf("缺少 %s", f)
		}
	}
	// 清单里得登记这两份(有基线才分得清"你改过"与"文档随 CLI 升级")
	man := loadManifest("ui")
	for _, k := range []string{"AGENTS.md", "DESIGN.md"} {
		if _, ok := man.Files[k]; !ok {
			t.Errorf("清单没登记 %s", k)
		}
	}

	// ① 手改过的保留
	data, err := os.ReadFile(agents)
	if err != nil {
		t.Fatal(err)
	}
	mine := append(append([]byte{}, data...), []byte("\n我自己加的一条规则\n")...)
	if err := os.WriteFile(agents, mine, 0o644); err != nil {
		t.Fatal(err)
	}
	out := captureOutput(t, func() { initProject("ui", writeOpts{}) })
	if !strings.Contains(out, "AGENTS.md 本地已修改, 跳过") {
		t.Errorf("改过的文档该报本地已修改并跳过: %s", out)
	}
	if got, _ := os.ReadFile(agents); string(got) != string(mine) {
		t.Error("改过的 AGENTS.md 被覆写了")
	}

	// ② 模拟"CLI 的文档更新了而你没动过": 盘上内容与远端不同(远端=仓库 docs/DESIGN.md),
	// 而基线等于盘上内容 ⇒ 判定该走 placeUpgrade, 直接刷新并说"版本升级"。
	designData, err := os.ReadFile(design)
	if err != nil {
		t.Fatal(err)
	}
	updated := append(append([]byte{}, designData...), []byte("\n模拟新版文档\n")...)
	if err := os.WriteFile(design, updated, 0o644); err != nil {
		t.Fatal(err)
	}
	man = loadManifest("ui")
	man.Files["DESIGN.md"] = sha256Hex(updated)
	if err := man.save("ui"); err != nil {
		t.Fatal(err)
	}
	out = captureOutput(t, func() { initProject("ui", writeOpts{}) })
	if !strings.Contains(out, "DESIGN.md (版本升级)") {
		t.Errorf("没改过的文档该随版本刷新: %s", out)
	}
	// 刷新后的内容 = 远端下发的文档(远端本来就是 designData, 所以判"模拟标记被擦掉")
	gotDesign, _ := os.ReadFile(design)
	if bytes.Contains(gotDesign, []byte("模拟新版文档")) {
		t.Error("DESIGN.md 没被刷新(还留着本地那份模拟内容)")
	}

	// ③ -o 才丢弃改动
	if err := os.WriteFile(agents, mine, 0o644); err != nil {
		t.Fatal(err)
	}
	out = captureOutput(t, func() { initProject("ui", writeOpts{overwrite: true}) })
	if !strings.Contains(out, "AGENTS.md 覆盖") && !strings.Contains(out, "覆盖 ui\\AGENTS.md (本地修改已丢弃)") {
		t.Errorf("加 -o 时该报覆盖并说明丢弃: %s", out)
	}
	if got, _ := os.ReadFile(agents); string(got) == string(mine) {
		t.Error("-o 之后 AGENTS.md 仍是被改过的内容")
	}

	// ④ theme.go 不纳入判定: 存在就一个字都不动
	theme := filepath.Join("ui", "theme.go")
	if err := os.WriteFile(theme, []byte("package ui\n// 我改过主题\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out = captureOutput(t, func() { initProject("ui", writeOpts{overwrite: true}) })
	if !strings.Contains(out, "theme.go (已存在)") {
		t.Errorf("theme.go 该永远保留(即使加了 -o): %s", out)
	}
	if got, _ := os.ReadFile(theme); !strings.Contains(string(got), "我改过主题") {
		t.Error("theme.go 被覆写了 —— 它是给用户改主题的, 不该纳入基线")
	}

	// ⑤ 预览不动文件
	before := sha256Hex(mustRead(t, design))
	out = captureOutput(t, func() { initProject("ui", writeOpts{dryRun: true}) })
	if strings.Contains(out, "\n创建 ") || !strings.Contains(out, "预览") {
		t.Errorf("--dry-run 措辞或预览总结不对: %s", out)
	}
	if sha256Hex(mustRead(t, design)) != before {
		t.Error("--dry-run 改了 DESIGN.md")
	}
}

// mustRead 读文件, 失败就终止当前测试。
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// DESIGN.md / AGENTS.md 可以点名: add 能单独拿到它们(没跑过 init 的项目也补得上),
// update 只刷已存在的、不新增 —— 与 update 不顺手装新组件是同一条纪律。
func TestNamedDocTargets(t *testing.T) {
	t.Setenv("XUI_REGISTRY", moduleRoot())
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	out := captureOutput(t, func() { add(componentOptions{names: []string{"DESIGN.md"}, uiDir: "ui"}) })
	if !strings.Contains(out, "创建 ui\\DESIGN.md") && !strings.Contains(out, "创建 ui/DESIGN.md") {
		t.Errorf("add 点名文档该创建它: %s", out)
	}
	if fileExists(filepath.Join("ui", "AGENTS.md")) {
		t.Error("只点名 DESIGN.md 却把 AGENTS.md 也写了")
	}
	if got, _ := os.ReadFile(filepath.Join("ui", "DESIGN.md")); !strings.Contains(string(got), "xcgui") {
		t.Error("DESIGN.md 内容不对")
	}
	// 文档也进清单(有基线才谈得上"你改过"与"随 CLI 升级")
	if _, ok := loadManifest("ui").Files["DESIGN.md"]; !ok {
		t.Error("DESIGN.md 没登记基线")
	}

	// update 已存在的那份: 没改过 = 已一致
	out = captureOutput(t, func() { update(componentOptions{names: []string{"DESIGN.md"}, uiDir: "ui"}) })
	if !strings.Contains(out, "已一致") {
		t.Errorf("未改动的文档该报已一致: %s", out)
	}
	// update 不新增: AGENTS.md 不在目录里 → 只说明怎么办, 不写文件
	out = captureOutput(t, func() { update(componentOptions{names: []string{"AGENTS.md"}, uiDir: "ui"}) })
	if !strings.Contains(out, "xui add AGENTS.md") {
		t.Errorf("该指明补齐的写法: %s", out)
	}
	if fileExists(filepath.Join("ui", "AGENTS.md")) {
		t.Error("update 不该新增它没装过的文件")
	}
	// add 补上第二份, 且能同时处理组件目标与文档目标
	out = captureOutput(t, func() {
		add(componentOptions{names: []string{"AGENTS.md", "card"}, uiDir: "ui"})
	})
	for _, want := range []string{"card.go", "AGENTS.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("混合目标少了 %s: %s", want, out)
		}
	}
	// 手改过的文档: update 指名也只跳过, 不覆写
	p := filepath.Join("ui", "DESIGN.md")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(append([]byte{}, data...), []byte("\n我加的段\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	out = captureOutput(t, func() { update(componentOptions{names: []string{"DESIGN.md"}, uiDir: "ui"}) })
	if !strings.Contains(out, "DESIGN.md 本地已修改, 跳过") {
		t.Errorf("改过的文档该保留: %s", out)
	}
	if got, _ := os.ReadFile(p); !strings.Contains(string(got), "我加的段") {
		t.Error("改过的 DESIGN.md 被覆写了")
	}
	// 路径写法也认(基名比对), 且拆分结果用规范名 —— 小写输入不能把基线记成两个键
	for _, in := range []string{"ui/DESIGN.md", "design.md", "DESIGN.md"} {
		d, c := splitForWrite(componentOptions{names: []string{in, "button"}})
		if len(d) != 1 || d[0] != "DESIGN.md" || len(c) != 1 {
			t.Errorf("%q 拆分不对: docs=%v comps=%v", in, d, c)
		}
	}
	// --view/--diff 是展示模式, 文档名不参与拆分(留给 registry 的解析去报错并指路)
	if d, _ := splitForWrite(componentOptions{names: []string{"DESIGN.md"}, view: true}); len(d) != 0 {
		t.Errorf("--view 模式下不该把文档名当写入目标")
	}
}

// 基线要按"交付完成那一刻盘上的内容"记。以前按"打算写的字节"记, 而同一次 init -n 后面
// 自己就会改写 go.mod(抬配套运行时的 go get、以及 go mod tidy) —— 于是文件一交付就和自己
// 的基线不一致, 下次 init -n --force 会说"本地已修改, 跳过", 可用户根本没动过。
func TestInitNewProjectBaselinesMatchDeliveredState(t *testing.T) {
	// 造一份本地源: registry.runtime 比 CLI 自身版本高 ⇒ init 会去抬依赖(触发 go.mod 被改写)
	src := t.TempDir()
	reg := `{"runtime":"v9.9.9","components":{`
	for _, c := range baseComponents {
		reg += `"` + c + `":{"files":["` + c + `.go"],"deps":[]},`
	}
	reg = strings.TrimSuffix(reg, ",") + `}}`
	if err := os.WriteFile(filepath.Join(src, "registry.json"), []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "components"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range baseComponents {
		if err := os.WriteFile(filepath.Join(src, "components", c+".go"), []byte("package ui\n// "+c+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XUI_REGISTRY", filepath.ToSlash(src))
	stubWinresMake(t)

	// go get 的桩要真的改写 go.mod —— 只记录调用就复现不出"我们自己动了这个文件"
	origGet := goGetModule
	t.Cleanup(func() { goGetModule = origGet })
	goGetModule = func(dir, spec string) error {
		p := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out := regexp.MustCompile(`(github\.com/twgh/xui v[^\s]+)`).ReplaceAllString(string(data), "github.com/twgh/xui v9.9.9")
		return os.WriteFile(p, []byte(out), 0o644)
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	wd := t.TempDir()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	if err := initNewProject("app", "ui", false, writeOpts{}); err != nil {
		t.Fatalf("init -n 失败: %v", err)
	}
	// 抬依赖确实发生了(否则这个测试没测到东西)
	onDisk, err := os.ReadFile(filepath.Join(wd, "app", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "v9.9.9") {
		t.Fatalf("桩没生效, go.mod 还是:\n%s", onDisk)
	}
	// 基线必须等于盘上内容
	man := loadManifest(filepath.Join(wd, "app", "ui"))
	if man.Project["go.mod"] != sha256Hex(onDisk) {
		t.Errorf("go.mod 基线不是交付后的内容: 清单 %s ≠ 盘上 %s", man.Project["go.mod"], sha256Hex(onDisk))
	}
	// 重跑 --force: 我们动过但不算用户改 → 该"已一致"(以前会说"本地已修改, 跳过")
	out := captureOutput(t, func() {
		if err := initNewProject("app", "ui", true, writeOpts{noInstall: true}); err != nil {
			t.Errorf("重跑失败: %v", err)
		}
	})
	if !strings.Contains(out, "go.mod") || strings.Contains(out, "go.mod 本地已修改") {
		t.Errorf("自己抬过依赖的 go.mod 不该说本地已修改:\n%s", out)
	}
	// 反过来: 用户真改过时不能被"洗成基线"蒙过去
	p := filepath.Join(wd, "app", "README.md")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(append([]byte{}, data...), []byte("\n我的项目说明\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	out = captureOutput(t, func() {
		if err := initNewProject("app", "ui", true, writeOpts{noInstall: true}); err != nil {
			t.Errorf("重跑失败: %v", err)
		}
	})
	if !strings.Contains(out, "README.md 本地已修改, 跳过") {
		t.Errorf("用户改过的该报本地已修改:\n%s", out)
	}
	// 被跳过的那份不能被顺手洗成基线(否则下次模板升级会连用户改动一起替换)
	man = loadManifest(filepath.Join(wd, "app", "ui"))
	if man.Project["README.md"] == sha256Hex(mustRead(t, p)) {
		t.Error("跳过的文件被洗成了基线 —— 下次模板升级会覆掉用户的改动")
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
				if err := selfUpdateAt("/home/u/go/bin/xui", c.current, "", true, false); err != nil {
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
		if err := selfUpdateAt(running, "v0.8.0", "v0.9.9", false, false); err != nil {
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
		if err := selfUpdateAt(filepath.Join(bin, "xui"+exeExt()), "v0.8.0", "", false, false); err != nil {
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
		if err := selfUpdateAt(installed, "v0.8.0", "", false, false); err != nil {
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
	err := selfUpdateAt(running, "v0.8.0", "v0.9.9", false, false)
	if err == nil || !strings.Contains(err.Error(), "手动跑: go install "+cliModule+"@v0.9.9") {
		t.Errorf("应给出手动升级命令, 实为 %v", err)
	}

	err = selfUpdateAt("/tmp/go-build7/b001/exe/xui", "v0.8.0", "", false, false)
	if err == nil || !strings.Contains(err.Error(), "临时构建") {
		t.Errorf("go run 的临时二进制应被拒绝, 实为 %v", err)
	}

	err = selfUpdateAt(running, "v0.8.0", "not-a-version", false, false)
	if err == nil || !strings.Contains(err.Error(), "认不出版本") {
		t.Errorf("非法目标应报错, 实为 %v", err)
	}
}

// 取不到 GOBIN 时: go install 照跑, 但不谎报"已升级"。
func TestSelfUpdateWithoutGoBin(t *testing.T) {
	specs := stubSelfUpdate(t, "", "v0.9.9", "v0.9.9")
	out := captureOutput(t, func() {
		if err := selfUpdateAt("/home/u/go/bin/xui", "v0.8.0", "", false, false); err != nil {
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
