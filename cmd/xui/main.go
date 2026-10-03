// xui CLI: 把组件源码拷贝进用户项目(shadcn 的 copy-not-install 模式)。
//
// 用法:
//
//	go run github.com/twgh/xui/cmd/xui@latest init [ui目录]   # 默认 ui
//	go run github.com/twgh/xui/cmd/xui@latest add button card ...
//	go run github.com/twgh/xui/cmd/xui@latest list
//
// init 在用户项目创建 ui/ 目录并写入 theme.go 模板; add 按 registry.json
// 把组件源码(及其依赖组件)拷入 ui/, 已存在且被用户改过的文件会提示跳过。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type entry struct {
	Files []string `json:"files"`
	Deps  []string `json:"deps"`
}

type registry struct {
	Components map[string]entry `json:"components"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]
	switch cmd {
	case "init":
		dir := "ui"
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		initProject(dir)
	case "add":
		if len(os.Args) < 3 {
			fatal("用法: xui add <组件名...>  (组件见 xui list)")
		}
		add(os.Args[2:], "ui")
	case "list":
		list()
	default:
		usage()
	}
}

func usage() {
	fmt.Println(`xui — shadcn 风格的 xcgui 自绘组件

  xui init [ui目录]    初始化用户项目(默认目录 ui)
  xui add <组件...>    拷贝组件源码到 ui 目录
  xui list             列出可用组件

组件源码拷贝进你的项目后归你所有, 可随意修改。`)
	os.Exit(0)
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "xui: "+format+"\n", a...)
	os.Exit(1)
}

// moduleRoot 返回 xui 模块源码根目录(go run / go install 均可用):
// runtime.Caller 取本文件路径, 上溯两级(cmd/xui -> 模块根)。
// go run pkg@version 时该路径位于模块缓存中, 源码同样可用。
func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fatal("无法定位 xui 源码目录")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func loadRegistry() *registry {
	data, err := os.ReadFile(filepath.Join(moduleRoot(), "registry.json"))
	if err != nil {
		fatal("读取 registry.json 失败: %v", err)
	}
	var r registry
	if err := json.Unmarshal(data, &r); err != nil {
		fatal("解析 registry.json 失败: %v", err)
	}
	return &r
}

func initProject(dir string) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatal("创建目录失败: %v", err)
	}
	themePath := filepath.Join(dir, "theme.go")
	if _, err := os.Stat(themePath); err == nil {
		fmt.Printf("跳过 %s (已存在)\n", themePath)
	} else if err := os.WriteFile(themePath, []byte(themeTemplate), 0o644); err != nil {
		fatal("写入 theme.go 失败: %v", err)
	} else {
		fmt.Printf("创建 %s\n", themePath)
	}

	// 拷入全部基础组件, 开箱即用
	add([]string{"button", "checkbox", "radio", "input", "label", "separator", "badge", "card"}, dir)
	fmt.Println("\n完成。接下来:")
	fmt.Println("  1. go get github.com/twgh/xui github.com/twgh/xcgui")
	fmt.Printf("  2. 在代码中 import 目录 %s 下的组件 (包名 %s)\n", dir, filepath.Base(dir))
	fmt.Println("  3. 主题在 " + themePath + " 中, 随意修改")
}

func add(names []string, uiDir string) {
	reg := loadRegistry()
	seen := map[string]bool{}
	var order []string
	var resolve func(name string)
	resolve = func(name string) {
		if seen[name] {
			return
		}
		e, ok := reg.Components[name]
		if !ok {
			fatal("未知组件 %q (xui list 查看可用组件)", name)
		}
		seen[name] = true
		for _, d := range e.Deps {
			resolve(d)
		}
		order = append(order, name)
	}
	for _, n := range names {
		resolve(n)
	}

	root := moduleRoot()
	for _, name := range order {
		for _, f := range reg.Components[name].Files {
			src := filepath.Join(root, "components", f)
			dst := filepath.Join(uiDir, f)
			data, err := os.ReadFile(src)
			if err != nil {
				fatal("读取组件源码失败: %v", err)
			}
			if old, err := os.ReadFile(dst); err == nil {
				if string(old) == string(data) {
					fmt.Printf("已存在 %s (内容一致)\n", dst)
					continue
				}
				fmt.Printf("跳过 %s (本地已修改; 删除后重跑可覆盖)\n", dst)
				continue
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				fatal("写入失败: %v", err)
			}
			fmt.Printf("创建 %s (%s)\n", dst, name)
		}
	}
}

func list() {
	reg := loadRegistry()
	fmt.Println("可用组件:")
	for name, e := range reg.Components {
		extra := ""
		if len(e.Deps) > 0 {
			extra = "  (依赖: " + strings.Join(e.Deps, ", ") + ")"
		}
		fmt.Printf("  %-12s %s%s\n", name, strings.Join(e.Files, ", "), extra)
	}
}

const themeTemplate = `package ui

import "github.com/twgh/xui"

// 这是你的主题文件 — 改这里即可全局换肤, 组件在绘制时实时读取。
// 令牌与 shadcn-ui 的 CSS 变量一一对应, 任何 shadcn 主题都能照抄过来。

// Active 当前使用的主题。想要暗色: Active = uiDark
var Active = light

func init() {
	xui.SetTheme(Active)
}

// light 亮色主题(shadcn zinc 转译)。
var light = &xui.Theme{
	Name:       "light",
	Background: xui.Hex("#ffffff"),
	Foreground: xui.Hex("#09090b"),

	Card:        xui.Hex("#ffffff"),
	CardFore:    xui.Hex("#09090b"),
	Popover:     xui.Hex("#ffffff"),
	PopoverFore: xui.Hex("#09090b"),

	Primary:         xui.Hex("#18181b"),
	PrimaryFore:     xui.Hex("#fafafa"),
	Secondary:       xui.Hex("#f4f4f5"),
	SecondaryFore:   xui.Hex("#18181b"),
	Muted:           xui.Hex("#f4f4f5"),
	MutedFore:       xui.Hex("#71717a"),
	Accent:          xui.Hex("#f4f4f5"),
	AccentFore:      xui.Hex("#18181b"),
	Destructive:     xui.Hex("#dc2626"),
	DestructiveFore: xui.Hex("#fafafa"),

	Border: xui.Hex("#e4e4e7"),
	Input:  xui.Hex("#e4e4e7"),
	Ring:   xui.Hex("#18181b"),
	Radius: 8,

	FontDefault: xui.FontSpec{Family: "微软雅黑", Size: 14},
	FontMono:    xui.FontSpec{Family: "Consolas", Size: 14},
}

// dark 暗色主题(shadcn zinc 转译)。
var dark = &xui.Theme{
	Name:       "dark",
	Background: xui.Hex("#09090b"),
	Foreground: xui.Hex("#fafafa"),

	Card:        xui.Hex("#09090b"),
	CardFore:    xui.Hex("#fafafa"),
	Popover:     xui.Hex("#09090b"),
	PopoverFore: xui.Hex("#fafafa"),

	Primary:         xui.Hex("#fafafa"),
	PrimaryFore:     xui.Hex("#18181b"),
	Secondary:       xui.Hex("#27272a"),
	SecondaryFore:   xui.Hex("#fafafa"),
	Muted:           xui.Hex("#27272a"),
	MutedFore:       xui.Hex("#a1a1aa"),
	Accent:          xui.Hex("#27272a"),
	AccentFore:      xui.Hex("#fafafa"),
	Destructive:     xui.Hex("#7f1d1d"),
	DestructiveFore: xui.Hex("#fafafa"),

	Border: xui.Hex("#27272a"),
	Input:  xui.Hex("#27272a"),
	Ring:   xui.Hex("#d4d4d8"),
	Radius: 8,

	FontDefault: xui.FontSpec{Family: "微软雅黑", Size: 14},
	FontMono:    xui.FontSpec{Family: "Consolas", Size: 14},
}
`
