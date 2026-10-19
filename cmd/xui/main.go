// xui CLI: 把组件源码拷贝进用户项目(shadcn 的 copy-not-install 模式)。
//
// 用法:
//
//	go run github.com/twgh/xui/cmd/xui@latest init [ui目录]   # 默认 ui
//	go run github.com/twgh/xui/cmd/xui@latest add button card ...
//	go run github.com/twgh/xui/cmd/xui@latest add --all
//	go run github.com/twgh/xui/cmd/xui@latest update [组件...]
//	go run github.com/twgh/xui/cmd/xui@latest list
//
// 数据源默认是仓库 main 分支(组件永远最新), 并按"直连 → Gitee 镜像 → 加速代理"的顺序
// 自动选路(直连最快但有墙, Gitee 镜像国内稳定, 代理兜底); 环境变量 XUI_REGISTRY 可指向
// 本地目录或任意 URL(离线开发 / 锁版本); 远程全不通时降级到 CLI 内嵌源码树。
// init 在用户项目创建 ui/ 目录并写入 theme.go 模板; add 按 registry.json
// 把组件源码(及其依赖组件)拷入 ui/, 已存在且被用户改过的文件会提示跳过。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// defaultRegistryBase 默认远程数据源。跟 main 分支 = 组件永远最新; 要锁组件版本
// 把 XUI_REGISTRY 指到某个 tag 的 raw 基址即可(GitHub raw 对 tag 同样有效)。
const defaultRegistryBase = "https://raw.githubusercontent.com/twgh/xui/main"

// giteeRegistryBase Gitee 镜像(与 GitHub raw 布局同构: <base>/registry.json,
// <base>/components/xxx.go), 国内访问稳定。前提是仓库也同步到 Gitee 的 twgh/xui。
const giteeRegistryBase = "https://gitee.com/twgh/xui/raw/main"

// proxyPrefixes GitHub 加速前缀, 仅作兜底(排在直连与 Gitee 镜像之后)。
var proxyPrefixes = []struct{ name, prefix string }{
	{"ghfast", "https://ghfast.top/"},
	{"gh-proxy", "https://gh-proxy.org/"},
}

type entry struct {
	Files []string `json:"files"`
	Deps  []string `json:"deps"`
}

type registry struct {
	Components map[string]entry `json:"components"`
}

// ---------------------------------------------------------------- 数据源

const (
	// fetchTimeout 组件文件(几十 KB)的常规超时
	fetchTimeout = 10 * time.Second
	// probeTimeout 候选源探测超时: registry.json 只有几 KB, 正常远小于 1s;
	// 直连被墙时表现为"连接建立后无响应", 短超时保证降级/换源不用白等
	probeTimeout = 6 * time.Second
)

var httpClient = &http.Client{Timeout: fetchTimeout}

func fetchWithTimeout(url string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "xui-cli")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func fetch(url string) ([]byte, error) {
	return fetchWithTimeout(url, fetchTimeout)
}

// candidate 一个远程候选基址。直连失败也会打日志(便于看清卡在哪一环), 但直连成功
// 与默认路径无异, 不打日志。
type candidate struct {
	label  string // 日志名: "直连" / "Gitee 镜像" / "ghfast 代理"
	direct bool   // 直连(GitHub 原址)
	base   string // 基址; 组件文件 = <base>/components/<file>
}

// candidates 生成候选序列: 直连 → Gitee 镜像 → 加速代理。
func candidates(base string) []candidate {
	out := []candidate{{label: "直连", direct: true, base: base}} // 直连优先
	if base == defaultRegistryBase {
		out = append(out, candidate{label: "Gitee 镜像", base: giteeRegistryBase})
	}
	for _, p := range proxyPrefixes {
		out = append(out, candidate{label: p.name + " 代理", base: p.prefix + base})
	}
	return out
}

// source 统一数据源: 本地目录与远程候选二选一。
type source struct {
	local string      // 非空 = 本地目录(须含 registry.json 与 components/)
	cands []candidate // 远程候选, 按顺序尝试, 第一个拉通 registry 的胜出
	base  string      // 已选定的远程基址(loadRegistry 之后才有效)
	env   string      // XUI_REGISTRY 原值; 非空 = 用户显式指定, 失败时直接报错不降级
}

func (s *source) desc() string {
	if s.local != "" {
		return s.local
	}
	return s.base
}

// read 读相对路径(registry.json / components/xxx.go)。本地与远程布局一致。
func (s *source) read(rel string) ([]byte, error) {
	if s.local != "" {
		return os.ReadFile(filepath.Join(s.local, filepath.FromSlash(rel)))
	}
	return fetch(s.base + "/" + rel)
}

// registrySource 按 XUI_REGISTRY 解析数据源(空 = 默认远程源, 自动先代理后直连)。
func registrySource() *source {
	env := strings.TrimSpace(os.Getenv("XUI_REGISTRY"))
	switch {
	case env == "":
		return &source{cands: candidates(defaultRegistryBase)}
	case strings.HasPrefix(env, "http://"), strings.HasPrefix(env, "https://"):
		// 显式指定的基址原样使用: 想强制直连就把默认地址显式写上, 想用别的代理就写代理地址
		return &source{cands: []candidate{{base: strings.TrimRight(env, "/")}}, env: env}
	default:
		abs, err := filepath.Abs(env)
		if err != nil {
			abs = env
		}
		return &source{local: abs, env: env}
	}
}

// loadRegistry 读 registry.json, 并返回实际生效的数据源。
// 默认远程源: 逐个候选(先代理后直连)拉 registry.json, 第一个拉通且解析成功的胜出;
// 全都不通才降级内嵌源码树。用户显式指定 XUI_REGISTRY 时直接报错, 不降级。
func loadRegistry() (*source, *registry) {
	src := registrySource()
	if src.local != "" {
		data, err := src.read("registry.json")
		if err != nil {
			fatal("读取 registry.json 失败 (%s): %v", src.desc(), err)
		}
		return src, mustParseRegistry(src.desc(), data)
	}
	var lastErr error
	for _, c := range src.cands {
		data, err := fetchWithTimeout(c.base+"/registry.json", probeTimeout)
		if err == nil {
			reg, perr := parseRegistry(c.base, data)
			if perr == nil {
				src.base = c.base
				if !c.direct {
					fmt.Fprintf(os.Stderr, "xui: 数据源 %s (%s)\n", c.base, c.label)
				}
				return src, reg
			}
			err = perr
		}
		lastErr = err
		if len(src.cands) > 1 { // 显式源只有一个候选, 不必打"换下一个"
			fmt.Fprintf(os.Stderr, "xui: %s不可用(%v), 换下一个\n", c.label, err)
		}
	}
	if src.env != "" {
		fatal("拉取 registry 失败 (%s): %v", src.env, lastErr)
	}
	fmt.Fprintf(os.Stderr, "xui: 所有远程数据源都不可用(最后错误: %v), 降级使用内嵌源码\n", lastErr)
	src = &source{local: moduleRoot()}
	data, err := src.read("registry.json")
	if err != nil {
		fatal("读取内嵌 registry.json 失败: %v", err)
	}
	return src, mustParseRegistry(src.desc(), data)
}

// parseRegistry 解析 registry.json。内容不合法(空表 / 代理返回劫持页等)时返回错误,
// 调用方据此换下一个候选。
func parseRegistry(desc string, data []byte) (*registry, error) {
	var r registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("解析 registry.json 失败 (%s): %w", desc, err)
	}
	if len(r.Components) == 0 {
		return nil, fmt.Errorf("registry.json 没有组件条目 (%s)", desc)
	}
	return &r, nil
}

func mustParseRegistry(desc string, data []byte) *registry {
	r, err := parseRegistry(desc, data)
	if err != nil {
		fatal("%v", err)
	}
	return r
}

func readComponent(src *source, file string) []byte {
	data, err := src.read("components/" + file)
	if err != nil {
		fatal("读取组件源码 %s 失败 (%s): %v", file, src.desc(), err)
	}
	return data
}

// ---------------------------------------------------------------- 命令入口

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd, args := os.Args[1], os.Args[2:]
	names, force, all := parseArgs(args)
	switch cmd {
	case "init":
		dir := "ui"
		if len(names) > 0 {
			dir = names[0]
		}
		if len(names) > 1 {
			fatal("用法: xui init [ui目录]")
		}
		initProject(dir, force)
	case "add":
		if !all && len(names) == 0 {
			fatal("用法: xui add <组件名...> | xui add --all  (组件见 xui list)")
		}
		add(names, "ui", force)
	case "update":
		update(names, "ui", force, all)
	case "list":
		list()
	default:
		usage()
	}
}

// parseArgs 抽出 --force / --all, 其余按位置参数返回(CLI 不做通用旗标解析)。
func parseArgs(args []string) (names []string, force, all bool) {
	for _, a := range args {
		switch a {
		case "--force", "-f":
			force = true
		case "--all", "-a":
			all = true
		default:
			names = append(names, a)
		}
	}
	return
}

func usage() {
	fmt.Println(`xui — shadcn 风格的 xcgui 自绘组件

  xui init [ui目录]     初始化(默认 ui: theme.go + 基础组件 + DESIGN.md; 已存在则跳过)
  xui add <组件...>     拷贝组件源码到 ui 目录(自动带上依赖组件)
  xui add --all         拷贝全部组件
  xui update [组件...]  更新已装组件(省略组件名 = ui 目录里已装的全部)
  xui list              列出可用组件

选项:
  --force               覆盖本地已修改的文件(默认: 警告并跳过)

数据源(环境变量 XUI_REGISTRY, 不设 = 远程 main 分支):
  不设                  自动选路: 直连 → Gitee 镜像 → 加速代理(ghfast / gh-proxy);
                        全不通则用 CLI 内嵌源码
  本地路径              直接读该目录(须含 registry.json 与 components/), 完全离线
  http(s):// 基址        以该地址为前缀原样拉取, 不套镜像/代理(失败直接报错, 不降级)

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
// 它同时是"远程源不可用"时的离线回退(内嵌源)。
func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fatal("无法定位 xui 源码目录")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// ---------------------------------------------------------------- 各命令实现

func initProject(dir string, force bool) {
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

	// 配套文档: 组件源码的注释里大量 "见 DESIGN 第 N 节", 拷给用户才是活的。
	// 从模块根读(go run pkg@version 时位于模块缓存, 源码文件同样在), 读不到不阻断 init。
	docPath := filepath.Join(dir, "DESIGN.md")
	if _, err := os.Stat(docPath); err == nil {
		fmt.Printf("跳过 %s (已存在)\n", docPath)
	} else if data, err := os.ReadFile(filepath.Join(moduleRoot(), "docs", "DESIGN.md")); err != nil {
		fmt.Fprintf(os.Stderr, "xui: 读取 DESIGN.md 失败, 跳过: %v\n", err)
	} else if err := os.WriteFile(docPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "xui: 写入 DESIGN.md 失败: %v\n", err)
	} else {
		fmt.Printf("创建 %s\n", docPath)
	}

	// 拷入全部基础组件, 开箱即用
	add([]string{"button", "checkbox", "radio", "input", "label", "separator", "badge", "card"}, dir, force)
	fmt.Println("\n完成。接下来:")
	fmt.Println("  1. go get github.com/twgh/xui github.com/twgh/xcgui")
	fmt.Printf("  2. 在代码中 import 目录 %s 下的组件 (包名 ui, 与目录名无关, 可自行起别名)\n", dir)
	fmt.Println("  3. 主题在 " + themePath + " 中, 随意修改")
	fmt.Printf("  4. 改组件/主题前先看 %s (xcgui 的坑、令牌表、每个组件的画法都在里面)\n", docPath)
}

// add 拷贝组件(含依赖闭包)到 uiDir; 空清单 = 全部组件(--all)。
func add(names []string, uiDir string, force bool) {
	src, reg := loadRegistry()
	if len(names) == 0 {
		names = allComponents(reg)
	}
	install(src, reg, resolveNames(reg, names), uiDir, force)
}

// update 更新已装组件: 省略组件名时 = uiDir 里已装的全部(按 registry 依赖闭包)。
func update(names []string, uiDir string, force, all bool) {
	src, reg := loadRegistry()
	switch {
	case all:
		names = allComponents(reg)
	case len(names) == 0:
		if names = installedComponents(reg, uiDir); len(names) == 0 {
			fmt.Printf("%s 下没有已安装的组件 (先用 xui add 装)\n", uiDir)
			return
		}
	}
	install(src, reg, resolveNames(reg, names), uiDir, force)
}

// allComponents 返回全部组件名(排序保证输出稳定)。
func allComponents(reg *registry) []string {
	names := make([]string, 0, len(reg.Components))
	for name := range reg.Components {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resolveNames 依赖闭包展开(去重, 依赖先于被依赖者), 未知组件直接报错。
func resolveNames(reg *registry, names []string) []string {
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
	return order
}

// installedComponents 扫描 uiDir, 返回"至少有一个文件在本地"的组件名(排序)。
func installedComponents(reg *registry, uiDir string) []string {
	var names []string
	for name, e := range reg.Components {
		for _, f := range e.Files {
			if _, err := os.Stat(filepath.Join(uiDir, f)); err == nil {
				names = append(names, name)
				break
			}
		}
	}
	sort.Strings(names)
	return names
}

// install 逐文件写入: 本地没有则创建, 内容一致则跳过, 用户改过则警告跳过(--force 覆盖)。
// 多个组件共享同一文件(separator/badge)时只处理一次。
func install(src *source, reg *registry, order []string, uiDir string, force bool) {
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		fatal("创建目录失败: %v", err)
	}
	done := map[string]bool{}
	for _, name := range order {
		for _, f := range reg.Components[name].Files {
			dst := filepath.Join(uiDir, f)
			if done[dst] {
				continue
			}
			done[dst] = true
			data := readComponent(src, f)
			old, err := os.ReadFile(dst)
			switch {
			case err != nil:
				if err := os.WriteFile(dst, data, 0o644); err != nil {
					fatal("写入失败: %v", err)
				}
				fmt.Printf("创建 %s (%s)\n", dst, name)
			case string(old) == string(data):
				fmt.Printf("已一致 %s\n", dst)
			case force:
				if err := os.WriteFile(dst, data, 0o644); err != nil {
					fatal("写入失败: %v", err)
				}
				fmt.Printf("覆盖 %s (本地修改已丢弃)\n", dst)
			default:
				fmt.Printf("警告 %s 本地已修改, 跳过 (确认后加 --force 覆盖)\n", dst)
			}
		}
	}
}

func list() {
	src, reg := loadRegistry()
	fmt.Printf("可用组件 (数据源: %s):\n", src.desc())
	for _, name := range allComponents(reg) {
		e := reg.Components[name]
		extra := ""
		if len(e.Deps) > 0 {
			extra = "  (依赖: " + strings.Join(e.Deps, ", ") + ")"
		}
		fmt.Printf("  %-12s %s%s\n", name, strings.Join(e.Files, ", "), extra)
	}
}

const themeTemplate = `package ui

import "github.com/twgh/xui"

// 这是你的主题文件 —— 改这里即可全局换肤, 组件在绘制时实时读取令牌。
// 默认取内置主题(nova 预设 neutral)的**副本**, 逐字段改即可自定义;
// 想从零写一份也行 —— 令牌与 shadcn-ui 的 CSS 变量一一对应, 字段表见 docs/DESIGN.md 3.1。

// Active 当前使用的主题。想要暗色: Active = dark
var Active = light

func init() {
	xui.SetTheme(&Active)
}

// light 亮色主题(内置 xui.Light() 的副本)。
var light = *xui.Light()

// dark 暗色主题(内置 xui.Dark() 的副本)。
var dark = *xui.Dark()

// 自定义示例(改字段即可, 不必重写整个结构体):
//	func init() {
//		light.Primary = xui.Hex("#2563eb")
//		light.PrimaryForeground = xui.Hex("#ffffff")
//		light.Radius = 8
//	}
`
