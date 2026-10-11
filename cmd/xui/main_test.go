package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 候选顺序是行为约定: 直连 → Gitee 镜像 → 加速代理。
func TestCandidates(t *testing.T) {
	cs := candidates(defaultRegistryBase)
	if len(cs) != len(proxyPrefixes)+2 {
		t.Fatalf("候选数 = %d, 期望 %d", len(cs), len(proxyPrefixes)+2)
	}
	if cs[0].base != defaultRegistryBase || cs[0].label != "直连" || !cs[0].direct {
		t.Errorf("第一个候选应为直连 %q, 实为 %q (%q)", defaultRegistryBase, cs[0].base, cs[0].label)
	}
	if cs[1].base != giteeRegistryBase || cs[1].label != "Gitee 镜像" {
		t.Errorf("第二个候选应为 Gitee 镜像 %q, 实为 %q (%q)", giteeRegistryBase, cs[1].base, cs[1].label)
	}
	for i, p := range proxyPrefixes {
		c := cs[i+2]
		if c.label != p.name+" 代理" {
			t.Errorf("候选 %d label = %q, 期望 %q", i+2, c.label, p.name+" 代理")
		}
		want := p.prefix + defaultRegistryBase
		if c.base != want {
			t.Errorf("候选 %d 基址 = %q, 期望 %q", i+2, c.base, want)
		}
	}
	// Gitee 镜像只在默认基址上有意义, 其它基址不该被镜像化
	if cs := candidates("https://example.com/raw/main"); len(cs) != len(proxyPrefixes)+1 {
		t.Errorf("非默认基址的候选数 = %d, 期望 %d", len(cs), len(proxyPrefixes)+1)
	}
}

func TestRegistrySource(t *testing.T) {
	t.Run("默认源自动选路", func(t *testing.T) {
		t.Setenv("XUI_REGISTRY", "")
		src := registrySource()
		if src.local != "" || len(src.cands) != len(proxyPrefixes)+2 {
			t.Fatalf("默认源应给出 %d 个候选, 实为 %d (local=%q)", len(proxyPrefixes)+2, len(src.cands), src.local)
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

	add([]string{"button"}, "ui", false) // button 单独装
	add([]string{"dialog"}, "ui", false) // dialog + 依赖 + 共享的 backdrop.go

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
	update([]string{"button"}, "ui", false, false)
	if data, _ := os.ReadFile(edited); string(data) != "package ui\n// mine\n" {
		t.Errorf("update 默认不该覆盖本地修改, 实为 %q", data)
	}
	update([]string{"button"}, "ui", true, false)
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
