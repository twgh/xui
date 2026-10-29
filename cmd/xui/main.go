// xui CLI: 把组件源码拷贝进用户项目(shadcn 的 copy-not-install 模式)。
//
// 用法:
//
//	go run github.com/twgh/xui/cmd/xui@latest init -n myapp   # 新建完整项目
//	go run github.com/twgh/xui/cmd/xui@latest init [ui目录]   # 已有项目装组件(默认 ui)
//	go run github.com/twgh/xui/cmd/xui@latest add button card ...
//	go run github.com/twgh/xui/cmd/xui@latest add --all
//	go run github.com/twgh/xui/cmd/xui@latest update [组件...]
//	go run github.com/twgh/xui/cmd/xui@latest list [-q 词] [-l n] [-o n]
//	go run github.com/twgh/xui/cmd/xui@latest view button
//	go run github.com/twgh/xui/cmd/xui@latest info
//
// 通用选项: -c/--cwd <目录> 先切目录, -s/--silent 只打印警告与错误。
//
// 数据源默认是仓库 main 分支(组件永远最新), 并按"直连 → Gitee 镜像 → GitCode 镜像 →
// 加速代理"的顺序自动选路(直连最快但有墙, 国内镜像稳定, 代理兜底); 环境变量 XUI_REGISTRY 可指向
// 本地目录或任意 URL(离线开发 / 锁版本); 远程全不通时降级到 CLI 内嵌源码树。
// init 在用户项目创建 ui/ 目录并写入 theme.go 模板; add 按 registry.json
// 把组件源码(及其依赖组件)拷入 ui/, 已存在且被用户改过的文件会提示跳过。
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	rdebug "runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"
)

// defaultRegistryBase 默认远程数据源。跟 main 分支 = 组件永远最新; 要锁组件版本
// 把 XUI_REGISTRY 指到某个 tag 的 raw 基址即可(GitHub raw 对 tag 同样有效)。
const defaultRegistryBase = "https://raw.githubusercontent.com/twgh/xui/main"

// giteeRegistryBase Gitee 镜像(与 GitHub raw 布局同构: <base>/registry.json,
// <base>/components/xxx.go), 国内访问稳定。前提是仓库也同步到 Gitee 的 twgh/xui。
const giteeRegistryBase = "https://gitee.com/twgh/xui/raw/main"

// gitcodeRegistryBase GitCode 镜像。GitCode 的 raw 直链(raw.gitcode.com/.../files/...)
// 对匿名访问的 .go 文件只回 403 "暂不支持预览", 所以走 v5 contents API(JSON + base64),
// 只有 JSON/README 这类预览白名单里的文件才能从 raw 直链拿到。匿名 API 也有限速,
// 所以排在 Gitee 之后当备用镜像。
const gitcodeRegistryBase = "https://api.gitcode.com/api/v5/repos/twgh/xui/contents"

// proxyPrefixes GitHub 加速前缀, 仅作兜底(排在直连与国内镜像之后)。
var proxyPrefixes = []struct{ name, prefix string }{
	{"ghfast", "https://ghfast.top/"},
	{"gh-proxy", "https://gh-proxy.org/"},
}

// 生成新项目 go.mod 的回退依赖版本。正式发布版(go run/install ...@vX)从 build info
// 读到自身版本; 只有本地 devel 构建会用到这里的值, 取最近一个已发布版本保证可拉取。
const (
	fallbackXuiVersion   = "v0.2.0"
	fallbackXcguiVersion = "v1.4.3"
)

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

// candidate 一个远程候选基址。直连失败也会打日志(便于看清卡在哪一环), 但直连成功
// 与默认路径无异, 不打日志。
type candidate struct {
	label  string // 日志名: "直连" / "Gitee 镜像" / "GitCode 镜像" / "ghfast 代理"
	direct bool   // 直连(GitHub 原址)
	base   string // 基址; 组件文件 = <base>/components/<file>
	api    bool   // GitCode v5 contents API: 返回 JSON+base64, 不是文件原文
}

// read 拉候选源里的相对路径。GitCode 的 contents API 响应形如
// {"encoding":"base64","content":"..."}, 要解码后才是源码; 默认分支固定 main。
func (c candidate) read(rel string, timeout time.Duration) ([]byte, error) {
	u := c.base + "/" + rel
	if !c.api {
		return fetchWithTimeout(u, timeout)
	}
	data, err := fetchWithTimeout(u+"?ref=main", timeout)
	if err != nil {
		return nil, err
	}
	var body struct {
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("%s: 解析 GitCode API 响应失败: %w", u, err)
	}
	if body.Encoding != "base64" {
		return nil, fmt.Errorf("%s: GitCode API 编码 = %q, 期望 base64", u, body.Encoding)
	}
	out, err := base64.StdEncoding.DecodeString(strings.NewReplacer("\r", "", "\n", "").Replace(body.Content))
	if err != nil {
		return nil, fmt.Errorf("%s: 解码 GitCode API 内容失败: %w", u, err)
	}
	return out, nil
}

// candidates 生成候选序列: 直连 → Gitee 镜像 → GitCode 镜像 → 加速代理。
func candidates(base string) []candidate {
	out := []candidate{{label: "直连", direct: true, base: base}} // 直连优先
	if base == defaultRegistryBase {
		out = append(out,
			candidate{label: "Gitee 镜像", base: giteeRegistryBase},
			candidate{label: "GitCode 镜像", base: gitcodeRegistryBase, api: true},
		)
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
	sel   candidate   // 已选定的远程候选(loadRegistry 之后才有效)
	rest  []candidate // 选定源之后还没试过的候选, 供单文件拉取失败时换源
	env   string      // XUI_REGISTRY 原值; 非空 = 用户显式指定, 失败时直接报错不降级
}

func (s *source) desc() string {
	if s.local != "" {
		return s.local
	}
	return s.sel.base
}

// read 读相对路径(registry.json / components/xxx.go)。本地直接读文件; 远程先走已选定的源,
// 失败就按候选顺序换源并把新源粘住。镜像各有各的限制(GitCode 匿名 API 50 次/分钟),
// 一次 add --all 要 28 个请求, 撞上限速只报"某个文件读失败"会留下半套 ui/。
func (s *source) read(rel string) ([]byte, error) {
	if s.local != "" {
		return os.ReadFile(filepath.Join(s.local, filepath.FromSlash(rel)))
	}
	data, err := s.sel.read(rel, fetchTimeout)
	if err == nil {
		return data, nil
	}
	for len(s.rest) > 0 {
		c := s.rest[0]
		s.rest = s.rest[1:]
		if data, err = c.read(rel, fetchTimeout); err == nil {
			warn("%s 拉取 %s 失败(%v), 数据源切到 %s (%s)", s.sel.base, rel, err, c.base, c.label)
			s.sel = c
			return data, nil
		}
	}
	return nil, err
}

// registrySource 按 XUI_REGISTRY 解析数据源(空 = 默认远程源, 直连优先)。
func registrySource() *source {
	env := strings.TrimSpace(os.Getenv("XUI_REGISTRY"))
	switch {
	case env == "":
		return &source{cands: candidates(defaultRegistryBase)}
	case strings.HasPrefix(env, "http://"), strings.HasPrefix(env, "https://"):
		// 显式指定的基址原样使用: 想强制直连就把默认地址显式写上, 想用别的代理就写代理地址。
		// GitCode 的 contents API 返回 JSON+base64, 显式指定时也要走解码路径。
		base := strings.TrimRight(env, "/")
		api := strings.Contains(base, "gitcode.com/api/v5/repos/") && strings.HasSuffix(base, "/contents")
		return &source{cands: []candidate{{base: base, api: api}}, env: env}
	default:
		abs, err := filepath.Abs(env)
		if err != nil {
			abs = env
		}
		return &source{local: abs, env: env}
	}
}

// loadRegistry 读 registry.json, 并返回实际生效的数据源。
// 默认远程源: 逐个候选(直连优先)拉 registry.json, 第一个拉通且解析成功的胜出;
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
	for i, c := range src.cands {
		data, err := c.read("registry.json", probeTimeout)
		if err == nil {
			reg, perr := parseRegistry(c.base, data)
			if perr == nil {
				src.sel = c
				src.rest = src.cands[i+1:] // 后续单文件失败时按序换源
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
	cmd := os.Args[1]
	args, err := extractGlobal(os.Args[2:])
	if err != nil {
		fatal("%v", err)
	}
	switch cmd {
	case "init":
		opts, err := parseInitArgs(args)
		if err != nil {
			fatal("%v", err)
		}
		if opts.project != "" {
			if err := initNewProject(opts.project, opts.force, opts.noInstall); err != nil {
				fatal("%v", err)
			}
			return
		}
		initProject(opts.uiDir, opts.force)
	case "add", "update":
		opts, err := parseComponentArgs(args)
		if err != nil {
			fatal("%v", err)
		}
		if cmd == "add" {
			add(opts)
		} else {
			update(opts)
		}
	case "list", "search": // search 是 shadcn 的写法, list 是历史别名
		opts, err := parseListArgs(args)
		if err != nil {
			fatal("%v", err)
		}
		list(opts)
	case "view":
		names, err := parseViewArgs(args)
		if err != nil {
			fatal("%v", err)
		}
		view(names)
	case "info":
		if len(args) > 0 {
			fatal("用法: xui info (不接受参数)")
		}
		infoCommand()
	default:
		usage()
	}
}

// quiet 由全局 -s/--silent 打开: 只抑制普通进度输出, 警告与错误照常打印。
var quiet bool

func info(format string, a ...any) {
	if !quiet {
		fmt.Printf(format+"\n", a...)
	}
}

func infoRaw(s string) {
	if !quiet {
		fmt.Print(s)
	}
}

// extractGlobal 抽出所有命令通用的 -c/--cwd 与 -s/--silent, 其余原样返回。
// cwd 在解析阶段就生效, 后续相对路径(项目目录 / ui 目录 / 数据源本地路径)都以它为基准。
func extractGlobal(args []string) ([]string, error) {
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c" || a == "--cwd":
			if i+1 >= len(args) || args[i+1] == "" {
				return nil, fmt.Errorf("%s 需要一个目录", a)
			}
			i++
			if err := os.Chdir(args[i]); err != nil {
				return nil, fmt.Errorf("切换工作目录失败: %v", err)
			}
		case strings.HasPrefix(a, "--cwd="):
			dir := strings.TrimPrefix(a, "--cwd=")
			if dir == "" {
				return nil, fmt.Errorf("--cwd 需要一个目录")
			}
			if err := os.Chdir(dir); err != nil {
				return nil, fmt.Errorf("切换工作目录失败: %v", err)
			}
		case a == "-s" || a == "--silent":
			quiet = true
		default:
			rest = append(rest, a)
		}
	}
	return rest, nil
}

// initOptions 是 init 的解析结果: project 非空 = 新建完整项目(xui init -n)，
// 否则 = 在已有项目里初始化组件目录(兼容原来的 xui init [ui目录])。
type initOptions struct {
	uiDir     string
	project   string
	force     bool
	noInstall bool
}

func parseInitArgs(args []string) (initOptions, error) {
	o := initOptions{}
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-n" || a == "--name":
			if i+1 >= len(args) || args[i+1] == "" {
				return o, fmt.Errorf("%s 需要一个项目名", a)
			}
			i++
			o.project = args[i]
		case strings.HasPrefix(a, "-n="):
			if o.project = strings.TrimPrefix(a, "-n="); o.project == "" {
				return o, fmt.Errorf("-n 需要一个项目名")
			}
		case strings.HasPrefix(a, "--name="):
			if o.project = strings.TrimPrefix(a, "--name="); o.project == "" {
				return o, fmt.Errorf("--name 需要一个项目名")
			}
		case a == "-f" || a == "--force":
			o.force = true
		case a == "--no-install":
			o.noInstall = true
		case strings.HasPrefix(a, "-"):
			return o, fmt.Errorf("未知选项 %s", a)
		default:
			pos = append(pos, a)
		}
	}
	if o.project != "" && len(pos) > 0 {
		return o, fmt.Errorf("init -n 不能同时指定 ui 目录")
	}
	if len(pos) > 1 {
		return o, fmt.Errorf("用法: xui init [ui目录] | xui init -n <项目名>")
	}
	if o.project == "" {
		if o.noInstall {
			return o, fmt.Errorf("--no-install 只用于 init -n")
		}
		o.uiDir = "ui"
		if len(pos) == 1 {
			o.uiDir = pos[0]
		}
	}
	return o, nil
}

// componentOptions 是 add / update 共用的参数集(对齐 shadcn add 的命名)。
type componentOptions struct {
	names     []string
	all       bool
	overwrite bool
	uiDir     string
	dryRun    bool
	view      bool
	viewFile  string
	diff      bool
	diffFile  string
}

func parseComponentArgs(args []string) (componentOptions, error) {
	o := componentOptions{uiDir: "ui"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-o" || a == "--overwrite":
			o.overwrite = true
		case a == "-a" || a == "--all":
			o.all = true
		case a == "-p" || a == "--path":
			if i+1 >= len(args) || args[i+1] == "" {
				return o, fmt.Errorf("%s 需要一个目录", a)
			}
			i++
			o.uiDir = args[i]
		case strings.HasPrefix(a, "--path="):
			if o.uiDir = strings.TrimPrefix(a, "--path="); o.uiDir == "" {
				return o, fmt.Errorf("--path 需要一个目录")
			}
		case a == "--dry-run":
			o.dryRun = true
		case a == "--view":
			o.view = true
		case strings.HasPrefix(a, "--view="):
			o.view = true
			o.viewFile = strings.TrimPrefix(a, "--view=")
		case a == "--diff":
			o.diff = true
		case strings.HasPrefix(a, "--diff="):
			o.diff = true
			o.diffFile = strings.TrimPrefix(a, "--diff=")
		case strings.HasPrefix(a, "-"):
			return o, fmt.Errorf("未知选项 %s", a)
		default:
			o.names = append(o.names, a)
		}
	}
	if o.view && o.diff {
		return o, fmt.Errorf("--view 与 --diff 不能同时使用")
	}
	if o.all && len(o.names) > 0 {
		return o, fmt.Errorf("--all 不能与组件名同时使用")
	}
	return o, nil
}

// listOptions 是 list / search 的参数集(对齐 shadcn search: -q / -l / -o)。
type listOptions struct {
	query  string
	limit  int
	offset int
}

func parseListArgs(args []string) (listOptions, error) {
	o := listOptions{}
	intValue := func(flag, s string) (int, error) {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("%s 需要一个整数: %q", flag, s)
		}
		return n, nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-q" || a == "--query":
			if i+1 >= len(args) {
				return o, fmt.Errorf("%s 需要一个搜索词", a)
			}
			i++
			o.query = args[i]
		case strings.HasPrefix(a, "--query="):
			o.query = strings.TrimPrefix(a, "--query=")
		case a == "-l" || a == "--limit":
			if i+1 >= len(args) {
				return o, fmt.Errorf("%s 需要一个数字", a)
			}
			i++
			n, err := intValue(a, args[i])
			if err != nil || n <= 0 {
				return o, fmt.Errorf("%s 需要正整数: %q", a, args[i])
			}
			o.limit = n
		case strings.HasPrefix(a, "--limit="):
			n, err := intValue("--limit", strings.TrimPrefix(a, "--limit="))
			if err != nil || n <= 0 {
				return o, fmt.Errorf("--limit 需要正整数")
			}
			o.limit = n
		case a == "-o" || a == "--offset":
			if i+1 >= len(args) {
				return o, fmt.Errorf("%s 需要一个数字", a)
			}
			i++
			n, err := intValue(a, args[i])
			if err != nil || n < 0 {
				return o, fmt.Errorf("%s 需要非负整数: %q", a, args[i])
			}
			o.offset = n
		case strings.HasPrefix(a, "--offset="):
			n, err := intValue("--offset", strings.TrimPrefix(a, "--offset="))
			if err != nil || n < 0 {
				return o, fmt.Errorf("--offset 需要非负整数")
			}
			o.offset = n
		default:
			return o, fmt.Errorf("未知参数 %s", a)
		}
	}
	return o, nil
}

func parseViewArgs(args []string) ([]string, error) {
	var names []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			return nil, fmt.Errorf("未知选项 %s", a)
		}
		names = append(names, a)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("用法: xui view <组件...>")
	}
	return names, nil
}

func usage() {
	fmt.Println(`xui — shadcn 风格的 xcgui 自绘组件

  xui init -n <项目名>       新建完整 Go 项目(go.mod + main.go + README + ui/ 组件)
  xui init -n .              在当前空目录直接生成项目(项目名取目录名)
  xui init [ui目录]          已有项目: 只初始化组件目录(默认 ui)
  xui add <组件...>          拷贝组件(自动带上依赖组件)
  xui add --all              拷贝全部组件
  xui add <组件...> --dry-run
                             只显示将创建/覆盖/跳过的文件
  xui add <组件...> --view[=文件]
                             打印组件源码, 不写本地
  xui add <组件...> --diff[=文件]
                             与本地文件做逐行 diff, 不写本地
  xui update [组件...]       更新已装组件(省略组件名 = ui 目录里已装的全部)
  xui list | xui search      列出/搜索组件
  xui view <组件...>         打印组件源码
  xui info                   显示 CLI / 项目环境信息

选项:
  -n, --name <项目名>   init: 新建项目目录
  -f, --force           init: 允许写入非空项目目录(覆盖模板文件)
  --no-install          init -n: 跳过自动 go mod tidy
  -o, --overwrite       add/update: 覆盖本地已修改的文件
  -p, --path <目录>     add/update: 组件目录(默认 ui)
  -a, --all             add/update: 全部组件
  -q, --query <词>      list/search: 按组件名过滤
  -l, --limit <n>       list/search: 最多显示 n 个
  -o, --offset <n>      list/search: 跳过前 n 个
  -c, --cwd <目录>      先切换到该目录, 后续相对路径都以它为准
  -s, --silent          只打印警告与错误

数据源(环境变量 XUI_REGISTRY, 不设 = 远程 main 分支):
  不设                  自动选路: 直连 → Gitee 镜像 → GitCode 镜像 →
                        加速代理(ghfast / gh-proxy); 全不通则用 CLI 内嵌源码
  本地路径              直接读该目录(须含 registry.json 与 components/), 完全离线
  http(s):// 基址        以该地址为前缀原样拉取, 不套镜像/代理(失败直接报错, 不降级)

组件源码拷贝进你的项目后归你所有, 可随意修改。`)
	os.Exit(0)
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "xui: "+format+"\n", a...)
	os.Exit(1)
}

// warn 打印警告: 不受 -s/--silent 影响(静默模式只压普通进度输出)。
func warn(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "xui: "+format+"\n", a...)
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
	if err := scaffoldUI(dir, force); err != nil {
		fatal("%v", err)
	}
	info("\n完成。接下来:")
	info("  1. go get github.com/twgh/xui github.com/twgh/xcgui")
	info("  2. 在代码中 import 目录 %s 下的组件 (包名 ui, 与目录名无关, 可自行起别名)", dir)
	info("  3. 主题在 %s 中, 随意修改", filepath.Join(dir, "theme.go"))
	info("  4. 改组件/主题前先看 %s (xcgui 的坑、令牌表、每个组件的画法都在里面)", filepath.Join(dir, "DESIGN.md"))
}

// scaffoldUI 写 ui 目录: theme.go + DESIGN.md + 基础组件。普通 init 与 init -n 共用。
func scaffoldUI(dir string, force bool) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}
	themePath := filepath.Join(dir, "theme.go")
	if _, err := os.Stat(themePath); err == nil {
		info("跳过 %s (已存在)", themePath)
	} else if err := os.WriteFile(themePath, []byte(themeTemplate), 0o644); err != nil {
		return fmt.Errorf("写入 theme.go 失败: %v", err)
	} else {
		info("创建 %s", themePath)
	}

	// 配套文档: 组件源码的注释里大量 "见 DESIGN 第 N 节", 拷给用户才是活的。
	// 从模块根读(go run pkg@version 时位于模块缓存, 源码文件同样在), 读不到不阻断 init。
	docPath := filepath.Join(dir, "DESIGN.md")
	if _, err := os.Stat(docPath); err == nil {
		info("跳过 %s (已存在)", docPath)
	} else if data, err := os.ReadFile(filepath.Join(moduleRoot(), "docs", "DESIGN.md")); err != nil {
		warn("读取 DESIGN.md 失败, 跳过: %v", err)
	} else if err := os.WriteFile(docPath, data, 0o644); err != nil {
		warn("写入 DESIGN.md 失败: %v", err)
	} else {
		info("创建 %s", docPath)
	}

	// 拷入全部基础组件, 开箱即用(window 是快速开始示例用的主窗口封装)
	add(componentOptions{names: baseComponents, uiDir: dir, overwrite: force})
	return nil
}

// initNewProject 创建完整项目目录: go.mod + main.go + README + .gitignore + ui/。
// 目标目录已存在且非空时默认拒绝, force 允许覆盖模板文件(不动无关文件)。
// noInstall = 跳过自动 go mod tidy(--no-install)。
//
// 目录规则:
//   - `-n .` 直接把骨架铺在当前目录(项目名取当前目录名);
//   - 当前目录本身是空的、且目录名 == -n 的项目名时, 同样铺在当前目录 ——
//     否则在 `D:\...\sitless` 里执行 `xui init -n sitless` 会生成 sitless/sitless。
func initNewProject(name string, force, noInstall bool) error {
	dir := name
	if name == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("读取当前目录失败: %v", err)
		}
		if name = projectNameFromDir(cwd); name == "" {
			return fmt.Errorf("无法从当前目录推断项目名, 请用 xui init -n <项目名>")
		}
	} else if cwdIsEmptyDirNamed(name) {
		dir = "."
	}
	if err := validateProjectName(name); err != nil {
		return err
	}
	if entries, err := os.ReadDir(dir); err == nil {
		if len(entries) > 0 && !force {
			return fmt.Errorf("目录 %s 已存在且非空 (确认后加 --force 覆盖模板文件)", dir)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取目录 %s 失败: %v", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建项目目录失败: %v", err)
	}

	module := moduleNameFromProject(name)
	xuiVer, xcguiVer := templateVersions()
	files := []struct{ name, body string }{
		{"go.mod", fmt.Sprintf(goModTemplate, module, xuiVer, xcguiVer)},
		{"main.go", fmt.Sprintf(mainGoTemplate, module, strconv.Quote(name))},
		{"README.md", fmt.Sprintf(readmeTemplate, name)},
		{".gitignore", gitignoreTemplate},
	}
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		action := "创建"
		if _, err := os.Stat(path); err == nil {
			action = "覆盖"
		}
		if err := os.WriteFile(path, []byte(f.body), 0o644); err != nil {
			return fmt.Errorf("写入 %s 失败: %v", path, err)
		}
		info("%s %s", action, path)
	}
	if err := scaffoldUI(filepath.Join(dir, "ui"), force); err != nil {
		return err
	}

	depsReady := false
	if !noInstall {
		info("\n正在安装依赖 (go mod tidy)...")
		if err := installGoModules(dir); err != nil {
			warn("依赖安装失败: %v", err)
		} else {
			depsReady = true
			info("依赖安装完成")
		}
	}
	next := "接下来:\n"
	if dir != "." {
		next += fmt.Sprintf("  cd %s\n", dir)
	}
	next += "  go mod tidy\n  go run ."
	if !depsReady {
		info("\n已创建项目 %s (module %s)\n\n%s", name, module, next)
		return nil
	}
	next = strings.Replace(next, "  go mod tidy\n", "", 1)
	info("\n已创建项目 %s (module %s)\n\n%s", name, module, next)
	return nil
}

// projectNameFromDir 取目录名做项目名(根目录无名字时返回空)。
func projectNameFromDir(dir string) string {
	base := filepath.Base(filepath.Clean(dir))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return ""
	}
	return base
}

// cwdIsEmptyDirNamed 报告当前工作目录是否为空、且目录名就是 name。
func cwdIsEmptyDirNamed(name string) bool {
	cwd, err := os.Getwd()
	if err != nil || projectNameFromDir(cwd) != name {
		return false
	}
	entries, err := os.ReadDir(cwd)
	return err == nil && len(entries) == 0
}

// installGoModules 在项目目录执行 go mod tidy, 生成 go.sum 并拉取依赖。
// GOWORK=off: 避免新项目被父目录的 go.work 接管, 依赖解析只按它自己的 go.mod。
func installGoModules(dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if msg := strings.TrimSpace(string(out)); msg != "" {
		return fmt.Errorf("%v\n%s", err, msg)
	}
	return err
}

// validateProjectName 只接受单级目录名(项目在当前目录下创建); 想换模块路径改 go.mod。
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("项目名不能为空")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("项目名只能是单级目录名: %q", name)
	}
	return nil
}

// moduleNameFromProject 把项目目录名转成合法的本地 Go module 路径:
// 只保留 ASCII 字母/数字/._-, 空格转 -, 全被过滤掉时回退 app。
func moduleNameFromProject(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), ".-_")
	if s == "" {
		return "app"
	}
	return s
}

// templateVersions 解析生成的 go.mod 用的依赖版本: 正式构建从 build info 取 CLI
// 自身版本与 xcgui 依赖版本, 本地 devel 构建回退到最近发布版。
func templateVersions() (xuiVer, xcguiVer string) {
	xuiVer, xcguiVer = fallbackXuiVersion, fallbackXcguiVersion
	bi, ok := rdebug.ReadBuildInfo()
	if !ok {
		return
	}
	if bi.Main.Path == "github.com/twgh/xui" && isReleaseVersion(bi.Main.Version) {
		xuiVer = bi.Main.Version
	}
	for _, d := range bi.Deps {
		switch d.Path {
		case "github.com/twgh/xui":
			if isReleaseVersion(d.Version) {
				xuiVer = d.Version
			}
		case "github.com/twgh/xcgui":
			if isReleaseVersion(d.Version) {
				xcguiVer = d.Version
			}
		}
	}
	return
}

// isReleaseVersion 判断 build info 里的版本能否直接写进 go.mod:
// 排除 (devel)、+dirty(本地修改)与伪版本(vX.Y.Z-0.<时间戳>-<commit>, 该 commit
// 不一定已推送/可拉取)。本地开发构建因此回退到 fallbackXuiVersion。
func isReleaseVersion(v string) bool {
	if !strings.HasPrefix(v, "v") || strings.Contains(v, "+") || strings.Contains(v, "-0.") {
		return false
	}
	return true
}

// baseComponents 是 init 默认装好的基础组件(window 是模板主窗口用的封装)。
var baseComponents = []string{"button", "checkbox", "radio", "input", "label", "separator", "badge", "card", "window"}

// add 拷贝组件(含依赖闭包); --view/--diff 只展示不写文件。
func add(opts componentOptions) {
	src, reg := loadRegistry()
	names := opts.names
	if opts.all {
		names = allComponents(reg)
	}
	if len(names) == 0 {
		fatal("用法: xui add <组件...> | xui add --all  (组件见 xui list)")
	}
	order := resolveNames(reg, names)
	switch {
	case opts.view:
		viewFiles(src, reg, order, opts.viewFile)
	case opts.diff:
		diffFiles(src, reg, order, opts.uiDir, opts.diffFile)
	default:
		install(src, reg, order, opts.uiDir, opts.overwrite, opts.dryRun)
	}
}

// update 更新已装组件: 省略组件名时 = uiDir 里已装的全部(按 registry 依赖闭包)。
func update(opts componentOptions) {
	src, reg := loadRegistry()
	names := opts.names
	switch {
	case opts.all:
		names = allComponents(reg)
	case len(names) == 0:
		if names = installedComponents(reg, opts.uiDir); len(names) == 0 {
			info("%s 下没有已安装的组件 (先用 xui add 装)", opts.uiDir)
			return
		}
	}
	order := resolveNames(reg, names)
	switch {
	case opts.view:
		viewFiles(src, reg, order, opts.viewFile)
	case opts.diff:
		diffFiles(src, reg, order, opts.uiDir, opts.diffFile)
	default:
		install(src, reg, order, opts.uiDir, opts.overwrite, opts.dryRun)
	}
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
		name = lookupComponent(reg, name)
		if seen[name] {
			return
		}
		seen[name] = true
		for _, d := range reg.Components[name].Deps {
			resolve(d)
		}
		order = append(order, name)
	}
	for _, n := range names {
		resolve(n)
	}
	return order
}

// lookupComponent 把用户输入解析成 registry 里的组件名, 允许省掉连字符、大小写随意。
//
// 为什么需要: registry 的 key 统一是 shadcn 文档的 kebab-case(navigation-menu /
// date-picker / scroll-area), 但用户常按文件名形状敲(navigationmenu / datepicker),
// 也常混大小写(实测反馈"为啥有 -")。这里先精确匹配(老命令一字不变), 失败后按
// "小写 + 去掉非字母数字"再比一次, 且**只有唯一命中**才采用 —— 不做前缀匹配,
// `xui add nav` 这类输入仍然报错, 不会被瞎猜成 navigation-menu。
func lookupComponent(reg *registry, name string) string {
	if _, ok := reg.Components[name]; ok {
		return name
	}
	want := normalizeName(name)
	if want != "" {
		hit, dup := "", false
		for k := range reg.Components {
			if normalizeName(k) != want {
				continue
			}
			if hit != "" {
				dup = true // 归一化后撞名: 不猜, 走报错
				break
			}
			hit = k
		}
		if hit != "" && !dup {
			return hit
		}
	}
	fatal("未知组件 %q%s (xui list 查看可用组件)", name, suggestComponent(reg, want))
	return "" // 不可达, fatal 会退出
}

// normalizeName 小写并去掉非字母数字: RadioGroup / radio_group / radio-group 都归一成 radiogroup。
func normalizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// suggestComponent 拼出"是否想找 xxx?"的提示, 按编辑距离挑最近的候选(同分取名字短的)。
//
// 两条例外: ① 输入是候选的**前缀**(只打了 nav)时算很近 —— 否则距 navigation-menu 有 8 个
// 编辑, 永远不会被推荐; ② 反过来(候选是输入的前缀, 即用户多打了字)不加分, 否则
// `radio-grup` 会优先推荐 `radio`(它是 radiogrup 的前缀)而不是 `radio-group`。
// 距离超过长度 1/3(至少 2) 的候选一律不推荐: 每个错别字都硬凑一个候选反而误导。
func suggestComponent(reg *registry, want string) string {
	if want == "" {
		return ""
	}
	best, bestD := "", 0
	for k := range reg.Components {
		nk := normalizeName(k)
		if nk == "" {
			continue
		}
		limit := len(nk) / 3
		if limit < 2 {
			limit = 2
		}
		d := editDistance(want, nk)
		if d > limit && len(want) >= 3 && strings.HasPrefix(nk, want) {
			d = 1 // 只写了前缀
		}
		if d > limit {
			continue
		}
		if best == "" || d < bestD || (d == bestD && len(k) < len(best)) {
			best, bestD = k, d
		}
	}
	if best == "" {
		return ""
	}
	return fmt.Sprintf(", 是否想找 %q?", best)
}

// editDistance 是 Levenshtein 距离(滚动一维数组, 只用来给"是否想找"挑候选)。
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// installedComponents 扫描 uiDir, 返回**全部文件都在本地**的组件名(排序)。
//
// 不能用"任意一个文件存在"判定: 组件之间共享助手文件(scrollbar.go / popup.go /
// scrollarea.go / group.go …), 只装了 select 时 popup.go 等已存在, 会把 date-picker /
// popover / dialog / table / tree / combobox / toggle 全判成"已安装", update 无参数
// 时把它们整套拷进 ui/。要求 files 全部存在才能把共享文件归到真正装过的组件上。
func installedComponents(reg *registry, uiDir string) []string {
	var names []string
	for name, e := range reg.Components {
		installed := len(e.Files) > 0
		for _, f := range e.Files {
			if _, err := os.Stat(filepath.Join(uiDir, f)); err != nil {
				installed = false
				break
			}
		}
		if installed {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// install 逐文件写入: 本地没有则创建, 内容一致则跳过, 用户改过则警告跳过
// (-o/--overwrite 覆盖)。dryRun 只打印将要发生的动作, 不创建任何文件/目录。
// 多个组件共享同一文件(separator/badge)时只处理一次。
func install(src *source, reg *registry, order []string, uiDir string, overwrite, dryRun bool) {
	if !dryRun {
		if err := os.MkdirAll(uiDir, 0o755); err != nil {
			fatal("创建目录失败: %v", err)
		}
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
			missing := os.IsNotExist(err)
			if err != nil && !missing {
				fatal("读取 %s 失败: %v", dst, err)
			}
			switch {
			case missing:
				if dryRun {
					info("将创建 %s (%s)", dst, name)
				} else {
					if err := os.WriteFile(dst, data, 0o644); err != nil {
						fatal("写入失败: %v", err)
					}
					info("创建 %s (%s)", dst, name)
				}
			case string(old) == string(data):
				info("已一致 %s", dst)
			case overwrite:
				if dryRun {
					info("将覆盖 %s (本地修改已丢弃)", dst)
				} else {
					if err := os.WriteFile(dst, data, 0o644); err != nil {
						fatal("写入失败: %v", err)
					}
					info("覆盖 %s (本地修改已丢弃)", dst)
				}
			default:
				warn("%s 本地已修改, 跳过 (确认后加 -o/--overwrite 覆盖)", dst)
			}
		}
	}
}

// viewFiles 打印组件的文件内容(不写本地); filter 非空时按文件名过滤。
func viewFiles(src *source, reg *registry, order []string, filter string) {
	done := map[string]bool{}
	found := false
	for _, name := range order {
		for _, f := range reg.Components[name].Files {
			if done[f] {
				continue
			}
			done[f] = true
			if filter != "" && filepath.Base(f) != filepath.Base(filter) {
				continue
			}
			found = true
			info("\n===== %s (%s) =====", f, name)
			infoRaw(string(readComponent(src, f)))
		}
	}
	if !found {
		info("没有匹配的文件: %s", filter)
	}
}

// view 打印指定组件的全部源码(依赖闭包内的共享文件去重)。
func view(names []string) {
	src, reg := loadRegistry()
	viewFiles(src, reg, resolveNames(reg, names), "")
}

// infoCommand 打印本机环境与项目状态(不拉 registry, 离线可用)。
func infoCommand() {
	ver := "(devel)"
	if bi, ok := rdebug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		ver = bi.Main.Version
	}
	wd, _ := os.Getwd()
	info("xui: %s", ver)
	info("工作目录: %s", wd)
	info("Go: %s", runtime.Version())
	if entries, err := os.ReadDir("ui"); err == nil {
		n := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				n++
			}
		}
		info("ui/: %d 个 Go 文件", n)
	} else {
		info("ui/: 未初始化")
	}
	src := strings.TrimSpace(os.Getenv("XUI_REGISTRY"))
	if src == "" {
		src = "默认(远程 main 分支)"
	}
	info("数据源: %s", src)
}

// diffFiles 把本地文件与 registry 内容做逐行 diff(不写本地); filter 非空时按文件名过滤。
func diffFiles(src *source, reg *registry, order []string, uiDir, filter string) {
	done := map[string]bool{}
	found := false
	for _, name := range order {
		for _, f := range reg.Components[name].Files {
			if done[f] {
				continue
			}
			done[f] = true
			if filter != "" && filepath.Base(f) != filepath.Base(filter) {
				continue
			}
			remote := string(readComponent(src, f))
			dst := filepath.Join(uiDir, f)
			local, err := os.ReadFile(dst)
			if err != nil && !os.IsNotExist(err) {
				fatal("读取 %s 失败: %v", dst, err)
			}
			if err == nil && string(local) == remote {
				continue
			}
			found = true
			infoRaw(unifiedDiff(dst, "components/"+f, string(local), remote))
		}
	}
	if !found {
		info("没有差异")
	}
}

// unifiedDiff 生成整文件级的 unified diff(一个 hunk), 用于 add/update --diff。
// 不做 3 行上下文裁剪: 组件文件通常只有几百行, 整文件输出更直观。
func unifiedDiff(localName, remoteName, local, remote string) string {
	a, b := splitDiffLines(local), splitDiffLines(remote)
	if len(a) == 0 && len(b) == 0 {
		return ""
	}
	// LCS 动态规划; 组件源码规模下内存/时间都可接受。
	n, m := len(a), len(b)
	dp := make([][]int32, n+1)
	for i := range dp {
		dp[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n@@ -1,%d +1,%d @@\n", localName, remoteName, len(a), len(b))
	for i, j := 0, 0; i < n || j < m; {
		switch {
		case i < n && j < m && a[i] == b[j]:
			out.WriteString(" " + a[i] + "\n")
			i++
			j++
		case j >= m || (i < n && dp[i+1][j] >= dp[i][j+1]):
			out.WriteString("-" + a[i] + "\n")
			i++
		default:
			out.WriteString("+" + b[j] + "\n")
			j++
		}
	}
	return out.String()
}

func splitDiffLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

// list 列出/搜索组件; -l/--limit 与 -o/--offset 按过滤后的顺序切片。
func list(opts listOptions) {
	src, reg := loadRegistry()
	matched := filterComponents(allComponents(reg), opts.query)
	start := opts.offset
	if start > len(matched) {
		start = len(matched)
	}
	end := len(matched)
	if opts.limit > 0 && start+opts.limit < end {
		end = start + opts.limit
	}
	info("可用组件 (数据源: %s):", src.desc())
	if start == end {
		if len(matched) == 0 {
			info("  (没有匹配的组件)")
		} else {
			info("  (没有更多组件)")
		}
	}
	for _, name := range matched[start:end] {
		e := reg.Components[name]
		extra := ""
		if len(e.Deps) > 0 {
			extra = "  (依赖: " + strings.Join(e.Deps, ", ") + ")"
		}
		info("  %-12s %s%s", name, strings.Join(e.Files, ", "), extra)
	}
	if opts.query != "" || opts.limit > 0 || opts.offset > 0 {
		info("\n共 %d 个匹配, 显示 %d 个", len(matched), end-start)
	}
}

// filterComponents 按组件名做大小写不敏感的子串过滤(空 query = 全部)。
func filterComponents(names []string, query string) []string {
	q := strings.ToLower(query)
	if q == "" {
		return names
	}
	var out []string
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), q) {
			out = append(out, name)
		}
	}
	return out
}

const themeTemplate = `package ui

import "github.com/twgh/xui"

// 这是你的主题文件 —— 改这里即可全局换肤, 组件在绘制时实时读取令牌。
// 默认取内置主题(nova 预设 neutral)的**副本**, 逐字段改即可自定义;
// 想从零写一份也行 —— 令牌与 shadcn-ui 的 CSS 变量一一对应, 字段表见 DESIGN.md 3.1。

// Active 当前使用的主题。想要暗色: Active = dark
var Active = light

func init() {
	xui.SetTheme(&Active)
}

// light 亮色主题(内置 xui.Light() 的副本)。
var light = *xui.Light()

// dark 暗色主题(内置 xui.Dark() 的副本)。
var dark = *xui.Dark()

// UseDark 切换亮色/暗色主题, 全局生效(内部会通知所有已注册窗口重绘)。
// 切换的是本文件里的 light/dark 副本, 逐字段定制过的颜色也会跟着走。
func UseDark(v bool) {
	if v {
		Active = dark
	} else {
		Active = light
	}
	xui.SetTheme(&Active)
}

// 自定义示例(改字段即可, 不必重写整个结构体):
//	func init() {
//		light.Primary = xui.Hex("#2563eb")
//		light.PrimaryForeground = xui.Hex("#ffffff")
//		light.Radius = 8
//	}
`

// goModTemplate 生成的 go.mod: 模块名取项目名, 依赖版本见 templateVersions。
const goModTemplate = `module %s

go 1.18

require (
	github.com/twgh/xui %s
	github.com/twgh/xcgui %s
)
`

// mainGoTemplate 生成的项目入口: AppWindow + 一张示例卡片(只用 init 装好的基础组件)。
const mainGoTemplate = `package main

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "%s/ui"
)

func main() {
	app.InitOrExit()
	a := app.New(true) // xui 固定使用 Direct2D 渲染
	a.EnableAutoDPI(true).EnableDPI(true)
	a.SetPaintFrequency(10) // 重绘节流 10ms, hover 过渡约 60fps

	w := ui.NewAppWindow(%s, 960, 640, ui.WithAppWindowMinSize(640, 480))
	content := w.Content()
	content.SetPadding(24, 24, 24, 24)
	content.SetSpace(16)

	_, stack := newCard(content.Handle, 12)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "欢迎使用 xui", stack.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "ui/ 目录是组件源码, 归你所有, 可随意修改。", stack.Handle))
	xui.ItemFill(ui.NewInput(0, 0, 0, 0, stack.Handle, ui.WithPlaceholder("输入点什么...")))
	xui.ItemFill(ui.NewCheckbox(0, 0, 0, 0, "记住我", stack.Handle))

	row := xui.HStack(stack.Handle, 36, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "开始使用", row.Handle), 104)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "了解更多", row.Handle, ui.WithVariant(ui.ButtonOutline)), 104)

	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "打开 ui/theme.go 改主题, 或看 ui/DESIGN.md 了解组件。", content.Handle).SetMuted())

	w.Show(true)
	a.Run()
	a.Exit()
}

// newCard 建一张宽度铺满、高度按内容自撑的卡片, 返回卡片与内容区。
func newCard(parent int, space int32) (*ui.Card, *widget.LayoutEle) {
	c := ui.NewCard(0, 0, 0, 0, parent)
	c.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	c.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return c, c.ContentStack(space)
}
`

const readmeTemplate = `# %s

由 xui 生成的项目骨架: Go + xcgui + xui(shadcn 风格自绘组件)。

## 运行

~~~bash
go run .
~~~

生成项目时已自动执行 go mod tidy; 如果初始化时用了 --no-install, 先跑一次 go mod tidy。

## 目录

- main.go — 应用入口(AppWindow + 示例卡片)
- ui/ — 组件源码拷贝, 归你所有, 可随意修改
- ui/theme.go — 主题令牌, 改这里全局换肤
- ui/DESIGN.md — 组件自定义指南与 xcgui 踩坑全集

## 添加组件

~~~bash
go run github.com/twgh/xui/cmd/xui@latest add table tree
~~~

组件名见 xui list 的输出。

## 依赖

- github.com/twgh/xui — 组件运行时 + CLI
- github.com/twgh/xcgui — 界面库(Windows, Direct2D)
`

const gitignoreTemplate = `# 编译产物
*.exe
*.test
*.out

# Go 工作区
go.work
go.work.sum

# 编辑器
.idea/
.vscode/
`
