// xui CLI: 把组件源码拷贝进用户项目(shadcn 的 copy-not-install 模式)。
//
// 安装(装一次, 之后直接用 xui):
//
//	go install github.com/twgh/xui/cmd/xui@latest
//
// 免安装用法:
//
//	go run github.com/twgh/xui/cmd/xui@latest init -n myapp   # 新建完整项目
//	go run github.com/twgh/xui/cmd/xui@latest init [ui目录]   # 已有项目装组件(默认 ui); 空目录则生成完整项目
//	go run github.com/twgh/xui/cmd/xui@latest add button card ...
//	go run github.com/twgh/xui/cmd/xui@latest add --all
//	go run github.com/twgh/xui/cmd/xui@latest update [组件...]
//	go run github.com/twgh/xui/cmd/xui@latest list [-q 词] [-l n] [-o n]
//	go run github.com/twgh/xui/cmd/xui@latest view button
//	go run github.com/twgh/xui/cmd/xui@latest info
//
// 装好以后升级 CLI 自己: xui self-update(或 xui self-update --check 只看)。
//
// 通用选项: -c/--cwd <目录> 先切目录, -s/--silent 只压进度输出(结果与警告/错误照常)。
//
// 数据源默认是仓库 main 分支(组件永远最新), 并按"直连 → Gitee 镜像 → GitCode 镜像 →
// 加速代理"的顺序自动选路(直连最快但有墙, 国内镜像稳定, 代理兜底); 环境变量 XUI_REGISTRY 可指向
// 本地目录或任意 URL(离线开发 / 锁版本); 远程全不通时降级到 CLI 内嵌源码树。
// init 在用户项目创建 ui/ 目录并写入 theme.go 模板; add 按 registry.json
// 把组件源码(及其依赖组件)拷入 ui/, 已存在且被用户改过的文件会提示跳过。
package main

import (
	"context"
	"crypto/sha256"
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
	fallbackXuiVersion   = "v0.10.0"
	fallbackXcguiVersion = "v1.4.3"
)

type entry struct {
	Files []string `json:"files"`
	Deps  []string `json:"deps"`
}

type registry struct {
	// Runtime 是这批组件源码要求的最低运行时版本(github.com/twgh/xui 的已发布 tag)。
	// 老数据源没这个字段, 所以是可选的。
	Runtime string `json:"runtime,omitempty"`
	// CLIChange 是最近一次改动 CLI 的 tag(命令行为 + 内嵌模板与资源; 机械抬 fallback 版本不算)。
	// 组件与 DESIGN 文档随数据源下发, 不需要换 CLI 就能拿到, 所以"有没有更新 tag"不能拿来当
	// 升级提示的依据 —— 只有这个字段高于用户装的版本才值得提示。老数据源没这个字段, 是可选的。
	CLIChange  string           `json:"cliChange,omitempty"`
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
				// 选路是进度信息(不是警告), 走 stderr 以免污染 `xui view > x.go` 的正文,
				// 但要受 -s/--silent 约束 —— 它与"创建了哪个文件"同属过程日志。
				if !c.direct && !quiet {
					// XUI_REGISTRY 显式指定时这个候选没有 label(不打镜像名), 别印出空的 ()
					if c.label == "" {
						fmt.Fprintf(os.Stderr, "xui: 数据源 %s\n", c.base)
					} else {
						fmt.Fprintf(os.Stderr, "xui: 数据源 %s (%s)\n", c.base, c.label)
					}
				}
				return src, reg
			}
			err = perr
		}
		lastErr = err
		if len(src.cands) > 1 && !quiet { // 显式源只有一个候选, 不必打"换下一个"
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
	if r.Runtime != "" && !isReleaseTag(r.Runtime) {
		// 认不出的配套版本就当没有: 宁可不动用户依赖, 也不要 go get @一个不存在的版本。
		warn("registry.json 的 runtime %q 不是 vX.Y.Z, 忽略 (%s)", r.Runtime, desc)
		r.Runtime = ""
	}
	if r.CLIChange != "" && !isReleaseTag(r.CLIChange) {
		warn("registry.json 的 cliChange %q 不是 vX.Y.Z, 忽略 (%s)", r.CLIChange, desc)
		r.CLIChange = ""
	}
	// 升级提示在命令跑完后才决定(那时拿不到 reg), 所以在这里把数据源声明的值记下来。
	// 一次进程只服务一个数据源, 后面候选换源/降级覆盖也是应该的。
	cliChangeTag = r.CLIChange
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
	// 先剥全局选项再认命令: `xui -s list` 与 `xui list -s` 等价(以前只认后者,
	// 前者会被当成"未知命令 -s")。
	rest, err := extractGlobal(os.Args[1:])
	if err != nil {
		fatal("%v", err)
	}
	if len(rest) == 0 {
		usage()
	}
	// -h/--help 优先于一切: 任何位置出现都直接打帮助。否则 `xui add --help` 会走到
	// 参数解析里报"未知选项 --help"。
	for _, a := range rest {
		if a == "-h" || a == "--help" {
			usage()
		}
	}
	cmd := rest[0]
	args := rest[1:]
	// 命令正常跑完后提示"CLI 有更新"(依据来自这次已经拉取的数据源, 不额外联网);
	// fatal 走 os.Exit 不会触发这里, 所以失败的命令不唠叨。
	defer notifyAvailableUpdate(cmd)
	// 版本: -v / --version / version 三种写法等价。放在命令分派之前, 且不碰数据源
	// (离线可用) —— 排查问题时第一句话就是"你装的哪个版本"。
	if cmd == "-v" || cmd == "--version" || cmd == "version" {
		if len(args) > 0 {
			fatal("用法: xui version")
		}
		out("xui %s", displayVersion())
		return
	}
	switch cmd {
	case "init":
		opts, err := parseInitArgs(args)
		if err != nil {
			fatal("%v", err)
		}
		initCommand(opts)
	case "add", "update":
		opts, err := parseComponentArgs(cmd, args)
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
		names, jsonMode, err := parseViewArgs(args)
		if err != nil {
			fatal("%v", err)
		}
		view(names, jsonMode)
	case "info":
		uiDir, jsonMode, err := parseInfoArgs(args)
		if err != nil {
			fatal("%v", err)
		}
		infoCommand(uiDir, jsonMode)
	case "self-update":
		target, checkOnly, jsonMode, err := parseSelfUpdateArgs(args)
		if err != nil {
			fatal("%v", err)
		}
		selfUpdate(target, checkOnly, jsonMode)
	default:
		fatal("未知命令 %q (见 xui --help)", cmd)
	}
}

// quiet 由全局 -s/--silent 打开: 只压进度输出(info/infoRaw), 命令结果(out/outRaw)、
// 警告与错误一律照常。
var quiet bool

func info(format string, a ...any) {
	if !quiet {
		fmt.Printf(format+"\n", a...)
	}
}

// infoRaw 与 info 同属进度输出(不补换行), 受 -s/--silent 影响。
func infoRaw(s string) {
	if !quiet {
		fmt.Print(s)
	}
}

// out / outRaw 打印**命令本身的结果** —— list 的组件表、view/diff 的源码与差异、
// info 与 self-update --check 的报告。它们和"创建了哪个文件"这类进度不是一回事:
// 结果是用户要读、要 `>` 重定向走的东西, 加 -s 把它一起吞掉等于命令没跑却退出码 0。
func out(format string, a ...any) {
	fmt.Printf(format+"\n", a...)
}

func outRaw(s string) {
	fmt.Print(s)
}

// ---------------------------------------------------------------- 结构化输出

// printJSON 打印命令结果的机器可读形式(-j / --json)。它与 out 同属"结果", 所以也不受
// -s 影响。中文表格给人看, 脚本与 AI 助手要稳定的字段 —— 项目会往用户项目下发 AGENTS.md,
// 那些会话里读 CLI 输出的多半是助手, 不该让它去解析对齐空格。
func printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal("生成 JSON 失败: %v", err)
	}
	outRaw(string(data) + "\n")
}

// componentJSON 是 list 的一条组件(files/deps 一律给数组, 空就是 [] 而不是 null,
// 省得调用方两种都要防)。
type componentJSON struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
	Deps  []string `json:"deps"`
}

// listJSON 是 list / search 的结果: 切片后的条目 + 让调用方知道自己看了多少。
type listJSON struct {
	Source     string          `json:"source"`
	CLI        string          `json:"cli"`
	Runtime    string          `json:"runtime,omitempty"`
	CLIChange  string          `json:"cliChange,omitempty"`
	Query      string          `json:"query,omitempty"`
	Offset     int             `json:"offset"`
	Limit      int             `json:"limit,omitempty"`
	Total      int             `json:"total"`
	Shown      int             `json:"shown"`
	Components []componentJSON `json:"components"`
}

// infoJSON 是 xui info 的环境报告(componentsDir 就是 -p 指过去的目录)。
type infoJSON struct {
	CLI                   string `json:"cli"`
	CWD                   string `json:"cwd"`
	Go                    string `json:"go"`
	ComponentsDir         string `json:"componentsDir"`
	ComponentsInitialized bool   `json:"componentsInitialized"`
	GoFiles               int    `json:"goFiles"`
	Source                string `json:"source"`
	NoticeDisabled        bool   `json:"noticeDisabled"`
}

// sourceFileJSON 是一个组件文件(content 是全文)。
type sourceFileJSON struct {
	Name      string `json:"name"`
	Component string `json:"component"`
	Content   string `json:"content"`
}

// viewJSON 是 view / --view 的结果。matched=false 表示过滤值谁都没挨着 ——
// 这时 files 为空, 调用方不该把它当成"这个组件没有文件"。
type viewJSON struct {
	Source  string           `json:"source"`
	Filter  string           `json:"filter,omitempty"`
	Matched bool             `json:"matched"`
	Files   []sourceFileJSON `json:"files"`
}

// diffFileJSON 是一个有差异的文件;state 与人类可读输出用的是同一套判定:
// missing(本地没有) / upgrade(未改动, 纯版本差异) / edited(本地改过) / unknown(没基线, 判断不了)。
type diffFileJSON struct {
	Path  string `json:"path"`
	File  string `json:"file"`
	State string `json:"state"`
	Diff  string `json:"diff"`
}

// diffJSON 是 --diff 的结果。matched=false 同上(过滤器没命中, 不等于"没有差异")。
type diffJSON struct {
	Dir     string         `json:"dir"`
	Filter  string         `json:"filter,omitempty"`
	Matched bool           `json:"matched"`
	Files   []diffFileJSON `json:"files"`
}

// checkJSON 是 self-update --check 的结果;command 是"下一步该跑什么", 已最新时为空串。
type checkJSON struct {
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	CLIFloor   string `json:"cliChange,omitempty"`
	LocalBuild bool   `json:"localBuild"`
	Upgradable bool   `json:"upgradable"`
	Command    string `json:"command,omitempty"`
}

// checkCommand 把 --check 的三种结论压成一条可直接执行的命令(给脚本用)。
func checkCommand(newest string, localBuild, upgradable bool) string {
	switch {
	case localBuild:
		return "xui self-update"
	case upgradable:
		return "xui self-update " + newest
	default:
		return ""
	}
}

// nextValue 取选项后面的值。三种情况报错:
//   - 到末尾(`xui add -p`)或值是空串(`xui add -p ""` 会把组件直接拷进当前目录根);
//   - 后面紧跟另一个选项(`xui add -p --all` 以前会把 --all 当目录名, 最后报出莫名其妙的
//     "用法: xui add <组件...>");
//   - 负数不算选项, 让 `-l -1` 落到"需要正整数"那条分支。
//
// 空值一律拒: 与 `--flag=` 的写法同一规则, 不能让"值被 shell 变量吃掉"变成"少个过滤器"
// 那种静默翻转(以前 --view= 会把"只看一个文件"打成"全部打印")。
func nextValue(args []string, i int, flag string) (string, error) {
	if i+1 >= len(args) {
		return "", fmt.Errorf("%s 需要一个值", flag)
	}
	v := args[i+1]
	if v == "" {
		return "", fmt.Errorf("%s 的值不能为空", flag)
	}
	if _, err := strconv.Atoi(v); err != nil && strings.HasPrefix(v, "-") {
		return "", fmt.Errorf("%s 需要一个值 (后面紧跟的是选项 %q)", flag, v)
	}
	return v, nil
}

// inlineValue 取 `-x=值` / `--flag=值` 写法里的值。空值报错, 与 nextValue 同一条规则 ——
// 值被 shell 变量吃掉时, 绝不能退化成"少了一个过滤器"。
func inlineValue(arg, prefix string) (string, error) {
	v := strings.TrimPrefix(arg, prefix)
	if v == "" {
		return "", fmt.Errorf("%s 的值不能为空", strings.TrimSuffix(prefix, "="))
	}
	return v, nil
}

// extractGlobal 抽出所有命令通用的 -c/--cwd 与 -s/--silent, 其余原样返回。
// cwd 在解析阶段就生效, 后续相对路径(项目目录 / ui 目录 / 数据源本地路径)都以它为基准。
func extractGlobal(args []string) ([]string, error) {
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c" || a == "--cwd":
			v, err := nextValue(args, i, a)
			if err != nil {
				return nil, err
			}
			i++
			if err := os.Chdir(v); err != nil {
				return nil, fmt.Errorf("切换工作目录失败: %v", err)
			}
		case strings.HasPrefix(a, "-c="):
			dir, err := inlineValue(a, "-c=")
			if err != nil {
				return nil, err
			}
			if err := os.Chdir(dir); err != nil {
				return nil, fmt.Errorf("切换工作目录失败: %v", err)
			}
		case strings.HasPrefix(a, "--cwd="):
			dir, err := inlineValue(a, "--cwd=")
			if err != nil {
				return nil, err
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
// 否则 = 在当前目录初始化组件目录(兼容原来的 xui init [ui目录]; 空目录里由 initCommand
// 改走完整项目, 见那里的 isBlankProjectDir)。
// force 与 overwrite 是两件事(以前挤在 -f 一个旗上):
//   - force    = 允许往已存在且非空的项目目录里写(只是开闸门, 不隐含"改动可以丢");
//   - overwrite = 允许丢弃本地改动(与 add/update 的 -o 同一个意思、同一套判定)。
type initOptions struct {
	uiDir     string
	project   string
	force     bool
	overwrite bool
	noInstall bool
	dryRun    bool
}

// writeOpts 是"往某个目录里写 xui 下发的文件"共用的三个开关(init 的两种形态与 add 都走它)。
// 收成 struct 是因为散装布尔一路加到 5 个, 调用点 initNewProject("app", true, true, false)
// 已经读不出第二个 true 是什么意思了。
type writeOpts struct {
	overwrite bool
	noInstall bool
	dryRun    bool
}

func (o initOptions) writes() writeOpts {
	return writeOpts{overwrite: o.overwrite, noInstall: o.noInstall, dryRun: o.dryRun}
}

func parseInitArgs(args []string) (initOptions, error) {
	o := initOptions{}
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-n" || a == "--name":
			v, err := nextValue(args, i, a)
			if err != nil {
				return o, err
			}
			i++
			o.project = v
		case strings.HasPrefix(a, "-n="):
			v, err := inlineValue(a, "-n=")
			if err != nil {
				return o, err
			}
			o.project = v
		case strings.HasPrefix(a, "--name="):
			v, err := inlineValue(a, "--name=")
			if err != nil {
				return o, err
			}
			o.project = v
		case a == "-f" || a == "--force":
			o.force = true
		case a == "-o" || a == "--overwrite":
			o.overwrite = true
		case a == "-p" || a == "--path":
			v, err := nextValue(args, i, a)
			if err != nil {
				return o, err
			}
			i++
			o.uiDir = v
		case strings.HasPrefix(a, "-p="):
			v, err := inlineValue(a, "-p=")
			if err != nil {
				return o, err
			}
			o.uiDir = v
		case strings.HasPrefix(a, "--path="):
			v, err := inlineValue(a, "--path=")
			if err != nil {
				return o, err
			}
			o.uiDir = v
		case a == "--no-install":
			o.noInstall = true
		case a == "--dry-run":
			o.dryRun = true
		case strings.HasPrefix(a, "-"):
			return o, fmt.Errorf("未知选项 %s", a)
		default:
			pos = append(pos, a)
		}
	}
	if len(pos) > 1 {
		return o, fmt.Errorf("用法: xui init [ui目录] | xui init -n <项目名>")
	}
	if o.project != "" {
		// init -n: -p 定的是"新项目里的组件目录", 必须落在项目内 —— 绝对路径或 .. 出去
		// 会把组件写到别的仓库里, 而清单跟着组件走, 之后就找不到自己的基线了。
		if len(pos) > 0 {
			return o, fmt.Errorf("xui init -n 的位置参数是项目名的一部分, 组件目录请用 -p <目录>")
		}
		if o.uiDir == "" {
			o.uiDir = "ui"
		}
		if err := checkUIWithinProject(o.uiDir); err != nil {
			return o, err
		}
		// 归一化后再落盘: `-p .\components` / `-p components/` 会原样进 main.go 的 import
		// 与下发文档的路径, 而 Go 的 import 路径不认点段与尾斜杠(`app/./components` 直接编译不过)。
		o.uiDir = filepath.ToSlash(filepath.Clean(o.uiDir))
		if o.uiDir == "." {
			return o, fmt.Errorf("xui init -n 的 -p 不能是项目根: 组件是 package ui, 与 main 放同一目录编译不过")
		}
		return o, nil
	}
	// 普通 init: 位置参数与 -p 是同一个东西(组件目录), 两个都给且不同不猜。
	if o.force {
		// 普通 init 没有"非空目录"这道闸门, 已有文件一律保留 —— 收下 -f 又不生效
		// 就是骗人; 真要丢弃本地改动说的是 -o。
		return o, fmt.Errorf("xui init [ui目录] 没有非空目录闸门, 不需要 --force; 要丢弃本地改动加 -o/--overwrite")
	}
	if o.uiDir == "" {
		o.uiDir = "ui"
		if len(pos) == 1 {
			o.uiDir = pos[0]
		}
	} else if len(pos) == 1 && pos[0] != o.uiDir {
		return o, fmt.Errorf("组件目录给了两个: -p %q 与 %q", o.uiDir, pos[0])
	}
	if o.uiDir == "" {
		// `xui init ""`: 位置参数是空串。不挡的话 Clean 会把它变成 "."(组件直接铺进当前目录),
		// 与"空值一律拒"的其他参数不是一套规矩。
		return o, fmt.Errorf("组件目录不能为空")
	}
	// 同 init -n: `.\ui` 与 `ui/` 是同一个目录, 统一成干净写法给后续的路径拼接与文档渲染。
	o.uiDir = filepath.ToSlash(filepath.Clean(o.uiDir))
	return o, nil
}

// checkUIWithinProject 保证 init -n 的组件目录还在项目内: 不许绝对路径、不许有 ".." 这一段。
// 判 segment 而不是判 Clean 的结果: filepath.Join 会把 ".." 直接消化掉("../ui" 变 "ui"),
// 拿它来检测跳出等于永远不触发。
func checkUIWithinProject(uiDir string) error {
	if filepath.IsAbs(uiDir) || strings.HasPrefix(uiDir, "/") || strings.HasPrefix(uiDir, `\`) {
		return fmt.Errorf("xui init -n 的 -p 要给项目内的相对目录, 实为 %q", uiDir)
	}
	for _, seg := range strings.Split(filepath.ToSlash(uiDir), "/") {
		if seg == ".." {
			return fmt.Errorf("xui init -n 的 -p 不能跳出项目目录: %q(安装清单落在组件目录里, 跳出去之后就找不到自己的基线了)", uiDir)
		}
	}
	return nil
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
	// json = --view/--diff 的结果改用结构化输出(-j / --json); 写文件那条路径没有结构化形式。
	json bool
	// strict = 有文件因本地改动被跳过时, 用退出码 2 结束(默认只汇总一句就正常退出)。
	strict bool
	// noInstall = 写组件文件但不联网动 go.mod 的配套运行时(与 init --no-install 同一个旗)。
	noInstall bool
}

// parseComponentArgs 解析 add / update。cmd 参与校验: 两个命令共用参数集,
// 但 --all 在 update 里不是"更新全部已装"(省略组件名本来就是这个意思), 它会把整套
// registry 组件拷进来 = add; 静默接受它等于让 update 悄悄装新东西。
func parseComponentArgs(cmd string, args []string) (componentOptions, error) {
	o := componentOptions{uiDir: "ui"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-o" || a == "--overwrite":
			o.overwrite = true
		case a == "-a" || a == "--all":
			o.all = true
		case a == "-p" || a == "--path":
			v, err := nextValue(args, i, a)
			if err != nil {
				return o, err
			}
			i++
			o.uiDir = v
		case strings.HasPrefix(a, "-p="):
			v, err := inlineValue(a, "-p=")
			if err != nil {
				return o, err
			}
			o.uiDir = v
		case strings.HasPrefix(a, "--path="):
			v, err := inlineValue(a, "--path=")
			if err != nil {
				return o, err
			}
			o.uiDir = v
		case a == "--dry-run":
			o.dryRun = true
		case a == "--no-install":
			o.noInstall = true
		case a == "--strict":
			o.strict = true
		case a == "--view":
			o.view = true
		case strings.HasPrefix(a, "--view="):
			v, err := inlineValue(a, "--view=")
			if err != nil {
				return o, err
			}
			o.view, o.viewFile = true, v
		case a == "--diff":
			o.diff = true
		case strings.HasPrefix(a, "--diff="):
			v, err := inlineValue(a, "--diff=")
			if err != nil {
				return o, err
			}
			o.diff, o.diffFile = true, v
		case a == "-j" || a == "--json":
			o.json = true
		case a == "-v":
			// 上游 shadcn 是 `add -v, --view`, 这里的 -v 归了全局版本号 —— 报错要说清去向,
			// 否则照文档敲的人会以为 --view 不存在。
			return o, fmt.Errorf("-v 是显示版本(xui version); 打印组件源码用 --view[=组件或文件]")
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
	if cmd == "update" && o.all {
		return o, fmt.Errorf("xui update 不带组件名就是更新 %s 里已装的全部; 想把没装的组件也一起装上请用 xui add --all", o.uiDir)
	}
	if o.json && !(o.view || o.diff) {
		// 只有"展示"这两条有结构化输出; 写文件那条没有, 收下 -j 又不生效是骗人。
		return o, fmt.Errorf("--json 只用于 --view / --diff (预览写入动作用 --dry-run)")
	}
	if o.view || o.diff {
		// --view/--diff 只展示、不写文件: 这些"怎么写"的开关跟着给会被静默忽略, 直接拒。
		if o.dryRun {
			return o, fmt.Errorf("--dry-run 与 --view/--diff 不能同时使用 (--view/--diff 本来就不写文件)")
		}
		if o.overwrite {
			return o, fmt.Errorf("-o/--overwrite 与 --view/--diff 不能同时使用 (--view/--diff 本来就不写文件)")
		}
		if o.noInstall {
			return o, fmt.Errorf("--no-install 与 --view/--diff 不能同时使用 (--view/--diff 本来就不写文件)")
		}
		if o.strict {
			return o, fmt.Errorf("--strict 与 --view/--diff 不能同时使用 (它管的是有文件被跳过, 而 --view/--diff 不写文件)")
		}
	}
	return o, nil
}

// parseSelfUpdateArgs 只认一个可选版本参数、--check 与 --json; 别在这里发明新语法。
func parseSelfUpdateArgs(args []string) (target string, checkOnly, jsonMode bool, err error) {
	for _, a := range args {
		switch {
		case a == "--check":
			checkOnly = true
		case a == "-j" || a == "--json":
			jsonMode = true
		case strings.HasPrefix(a, "-"):
			return "", false, false, fmt.Errorf("未知选项 %s (用法: xui self-update [版本|--check])", a)
		case target != "":
			return "", false, false, fmt.Errorf("用法: xui self-update [版本|--check] (只能给一个版本)")
		default:
			target = a
		}
	}
	if checkOnly && target != "" {
		// --check 只回答"我相对 @latest 新不新"。把版本参数收下又丢掉, 会让人以为
		// 它检查的是自己刚指定的那个版本。
		return "", false, false, fmt.Errorf("xui self-update --check 不接受版本 (要升到指定版本就去掉 --check: xui self-update %s)", target)
	}
	if jsonMode && !checkOnly {
		// 真正的升级动作没有结构化输出(它是一串过程日志), 收下 -j 又不生效是骗人。
		return "", false, false, fmt.Errorf("--json 只用于 xui self-update --check")
	}
	return target, checkOnly, jsonMode, nil
}

// listOptions 是 list / search 的参数集(对齐 shadcn search: -q / -l / -o)。
type listOptions struct {
	query  string
	limit  int
	offset int
	// json = 结果改用结构化输出给脚本与 AI 助手读(-j / --json)。
	json bool
}

// parseListArgs 解析 list / search。位置参数就是搜索词(`xui search button` 是最自然的
// 写法, 以前它报"未知参数"), 与 -q/--query 等价;两个都给且不同则报错, 不猜用哪个。
func parseListArgs(args []string) (listOptions, error) {
	o := listOptions{}
	var pos []string
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
			v, err := nextValue(args, i, a)
			if err != nil {
				return o, err
			}
			i++
			o.query = v
		case strings.HasPrefix(a, "-q="):
			v, err := inlineValue(a, "-q=")
			if err != nil {
				return o, err
			}
			o.query = v
		case strings.HasPrefix(a, "--query="):
			v, err := inlineValue(a, "--query=")
			if err != nil {
				return o, err
			}
			o.query = v
		case a == "-l" || a == "--limit":
			v, err := nextValue(args, i, a)
			if err != nil {
				return o, err
			}
			i++
			n, err := intValue(a, v)
			if err != nil || n <= 0 {
				return o, fmt.Errorf("%s 需要正整数: %q", a, v)
			}
			o.limit = n
		case strings.HasPrefix(a, "-l="):
			v, err := inlineValue(a, "-l=")
			if err != nil {
				return o, err
			}
			n, err := intValue("-l", v)
			if err != nil || n <= 0 {
				return o, fmt.Errorf("-l 需要正整数")
			}
			o.limit = n
		case strings.HasPrefix(a, "--limit="):
			v, err := inlineValue(a, "--limit=")
			if err != nil {
				return o, err
			}
			n, err := intValue("--limit", v)
			if err != nil || n <= 0 {
				return o, fmt.Errorf("--limit 需要正整数")
			}
			o.limit = n
		case a == "-o" || a == "--offset":
			v, err := nextValue(args, i, a)
			if err != nil {
				return o, err
			}
			i++
			n, err := intValue(a, v)
			if err != nil || n < 0 {
				return o, fmt.Errorf("%s 需要非负整数: %q", a, v)
			}
			o.offset = n
		case strings.HasPrefix(a, "-o="):
			v, err := inlineValue(a, "-o=")
			if err != nil {
				return o, err
			}
			n, err := intValue("-o", v)
			if err != nil || n < 0 {
				return o, fmt.Errorf("-o 需要非负整数")
			}
			o.offset = n
		case strings.HasPrefix(a, "--offset="):
			v, err := inlineValue(a, "--offset=")
			if err != nil {
				return o, err
			}
			n, err := intValue("--offset", v)
			if err != nil || n < 0 {
				return o, fmt.Errorf("--offset 需要非负整数")
			}
			o.offset = n
		case a == "-j" || a == "--json":
			o.json = true
		case strings.HasPrefix(a, "-"):
			return o, fmt.Errorf("未知选项 %s", a)
		default:
			pos = append(pos, a)
		}
	}
	if len(pos) > 1 {
		return o, fmt.Errorf("搜索词只能给一个, 实为 %v (想要全部就都别给, -q 只接一个词)", pos)
	}
	if len(pos) == 1 {
		// 同一个词用 -q 和位置各给一次不算矛盾, 不必报错; 给成两个不同的词才要停。
		if o.query != "" && o.query != pos[0] {
			return o, fmt.Errorf("搜索词给了两个: -q %q 与 %q", o.query, pos[0])
		}
		o.query = pos[0]
	}
	return o, nil
}

// parseInfoArgs 给 info 解析组件目录: 它报的是"我的项目里 xui 装成什么样", 目录不是 ui
// 时必须能指过去(以前 info 写死读 ui/, 组件装在 components/ 的项目里回答"ui/: 未初始化",
// 是错的)。
func parseInfoArgs(args []string) (uiDir string, jsonMode bool, err error) {
	uiDir = "ui"
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-p" || a == "--path":
			v, err := nextValue(args, i, a)
			if err != nil {
				return "", false, err
			}
			i++
			uiDir = v
		case strings.HasPrefix(a, "-p="):
			v, err := inlineValue(a, "-p=")
			if err != nil {
				return "", false, err
			}
			uiDir = v
		case strings.HasPrefix(a, "--path="):
			v, err := inlineValue(a, "--path=")
			if err != nil {
				return "", false, err
			}
			uiDir = v
		case a == "-j" || a == "--json":
			jsonMode = true
		case strings.HasPrefix(a, "-"):
			return "", false, fmt.Errorf("未知选项 %s (用法: xui info [-p 组件目录] [-j])", a)
		default:
			return "", false, fmt.Errorf("xui info 不接受参数 %s (组件目录用 -p <目录>)", a)
		}
	}
	return uiDir, jsonMode, nil
}

// parseViewArgs 解析 `xui view <组件...> [-j]`。
func parseViewArgs(args []string) (names []string, jsonMode bool, err error) {
	for _, a := range args {
		switch {
		case a == "-j" || a == "--json":
			jsonMode = true
		case strings.HasPrefix(a, "-"):
			return nil, false, fmt.Errorf("未知选项 %s", a)
		default:
			names = append(names, a)
		}
	}
	if len(names) == 0 {
		return nil, false, fmt.Errorf("用法: xui view <组件...>")
	}
	return names, jsonMode, nil
}

func usage() {
	fmt.Println(`xui — shadcn 风格的 xcgui 自绘组件

  xui init -n <项目名>       新建完整 Go 项目(go.mod + main.go + README + AGENTS.md + 组件目录 + winres/ 图标资源)
  xui init -n <项目名> -p <目录>
                             同上, 但组件目录改名(默认 ui; 必须落在项目内)
  xui init -n .              在当前空目录直接生成项目(项目名取目录名)
  xui init [组件目录]        已有项目: 只初始化组件目录(默认 ui)
                             当前目录是空的(且不在别的 Go 模块里)= 上一条, 在这里生成完整项目
  xui add <组件...>          拷贝组件(自动带上依赖组件)
  xui add --all              拷贝全部组件
  xui add <组件...> --dry-run
                             只显示将创建/覆盖/跳过的文件(init 也支持)
  xui add <组件...> --view[=组件或文件]
                             打印组件源码, 不写本地
  xui add <组件...> --diff[=组件或文件]
                             与本地文件做逐行 diff, 不写本地
                             (过滤值谁都没命中 = 命令失败, 退出码 1; 不是"没有差异")
  xui update [组件...]       升级已装组件(省略组件名 = ui 目录里已装的全部, 外加已在的
                             DESIGN.md / DESIGN-PITFALLS.md / AGENTS.md);
                             安装后没改过的文件直接替换, 改过的跳过(见 -o);
                             连没装过的组件一起装是 xui add 的事: 点名没装过的组件只提示,
                             update 不接 --all
  xui add DESIGN.md          只要下发的文档(DESIGN.md / DESIGN-PITFALLS.md / AGENTS.md
  xui update DESIGN.md       不是组件、不在 registry 里, 但前两份与组件同源, 从数据源取):
                             add 可以单独拿 / update 只刷已在目录里的那份;
                             DESIGN.md 会连配套的 DESIGN-PITFALLS.md 一起给
  xui list | xui search [词] 列出/搜索组件(位置参数就是搜索词, 等价于 -q 词)
  xui view <组件...>         打印组件源码
  xui info [-p 目录]         显示 CLI / 项目环境信息(组件目录不是 ui 时用 -p 指过去)
  xui version                只打印版本(同 -v / --version; 离线可用, 不拉数据源)
  xui self-update [版本]     升级 CLI 自己(默认 @latest; 也可给 tag / main / commit)
                             --check 报"当前版本 / 最新 tag / CLI 最后改动的 tag", 不安装
                             (不接版本参数)

选项:
  -n, --name <项目名>   init: 新建项目目录
  -f, --force           init -n: 允许写入已存在且非空的项目目录(只开这道闸门,
                        不隐含"你的改动可以丢"; 普通 init 用不上它)
  --no-install          init/add/update: 写文件但不联网动 go.mod 的依赖
                        (init -n 还跳过自动 go mod tidy 与 xc 安装, 仍写 winres/)
  --dry-run             init/add/update: 只预览, 不建目录、不写文件、不联网
  --strict              add/update: 有文件因本地改动被跳过时用退出码 2 失败(给 CI)
  -o, --overwrite       init/add/update: 丢弃本地改动(默认保留; 判定见下)
  -p, --path <目录>     init/add/update/info: 组件目录(默认 ui)。init -n 用它在新项目
                        里定目录名(必须是项目内的相对目录, 不能绝对也不能 .. 跳出);
                        普通 init 它与位置参数是同一个东西
  -a, --all             add: 全部组件
  -q, --query <词>      list/search: 按组件名过滤(直接写位置参数等价)
  -l, --limit <n>       list/search: 最多显示 n 个
  -o, --offset <n>      list/search: 跳过前 n 个
  -j, --json            list/search/info/view 与 add/update 的 --view/--diff、
                        self-update --check: 结果改出结构化 JSON(给脚本与 AI 助手读)
  -c, --cwd <目录>      先切换到该目录, 后续相对路径都以它为准
  -s, --silent          只压进度输出; 命令结果(list/view/diff/info/version)、
                        警告与错误照常
  -v, --version         显示版本(本地构建显示 (devel) + 短 commit)
  -h, --help            显示本帮助(任何位置都能用)

短选项也接受 -x=值 的写法(与 --long=值 一致), 例如 -p=src -l=5。
带值的选项两种写法都不许给空值(-p "" 与 -p= 同样报错): 空目录会把组件直接拷进当前目录根,
空的 --view= 会把"只看一个文件"静默变成"全部打印"。
--view/--diff 是"只展示"的开关, 与 --dry-run / -o / --no-install 同时给会直接报错。

改动判定(add/update 的组件文件与 init 的模板文件同一套): 装完之后没动过(本地哈希等于
                        安装清单里的基线)→ 随版本升级直接替换; 基线对不上 → 你改过,
                        默认保留, 要 -o 才覆盖; 该文件没有基线 → 只说"无法判断来源"。

退出码: 0 = 跑通了(该装的都装了, 跳过的已在 stderr 汇总一句说明);1 = 命令本身失败
                        (源拉不到/权限/参数错/过滤值没命中);2 = 只在看 --strict 且有
                        文件被跳过时出现。
                        CI 里想"有文件没更新干净就失败"就跑 --strict。

运行时版本: 组件源码跟 main 分支, 而运行时 github.com/twgh/xui 只能按 tag 取 ——
                        registry.json 的 runtime 字段是与这批组件配套的运行时版本,
                        add/update/init 装完组件会把项目 go.mod 的该依赖一起抬上去,
                        再跑 go mod tidy 清掉 go.sum 里的旧版本条目
                        (--dry-run 或 --no-install 时只打印这两条命令, 不联网改依赖)

升级提示: 依据是 registry.json 的 cliChange = CLI 最后一次改动的 tag(命令行为与内嵌模板/资源,
                        光抬配套版本不算)。装的版本低于它, 命令跑完才在 stderr 提示一句
                        "CLI 有更新"(不污染被重定向的结果);数据源没这个字段就只比 tag。
                        组件与设计文档(DESIGN / DESIGN-PITFALLS)都随数据源下发, 换 tag 不
                        必升级 CLI —— 所以只有真改了 CLI 才会唠叨。
                        -s/--silent 或 XUI_NO_UPDATE_CHECK=1 关闭。

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

// initCommand 分派 xui init 的三种走法:
//   - -n <项目名>: 在(通常是新建的)目录里生成完整项目;
//   - 空目录: 同一个脚手架铺在当前目录(项目名取目录名), 等同 xui init -n . —— 空目录里
//     没有"已有项目"可言, 只写组件目录会让人拿到一份既不能 go run、也不能 go get 的骨架;
//   - 其余: 已有项目, 只初始化组件目录。
func initCommand(opts initOptions) {
	if opts.project != "" {
		if err := initNewProject(opts.project, opts.uiDir, opts.force, opts.writes()); err != nil {
			fatal("%v", err)
		}
		return
	}
	cwd, err := os.Getwd()
	if err == nil && isBlankProjectDir(cwd, opts.uiDir) {
		// force=true 不是"允许写非空目录": 空/只有 .git 是上面自己判的, 这里的闸门没有内容可放。
		info("当前目录是空的: 在这里生成完整项目 (项目名 %s, 同 xui init -n .)", projectNameFromDir(cwd))
		if err := initNewProject(".", opts.uiDir, true, opts.writes()); err != nil {
			fatal("%v", err)
		}
		return
	}
	initProject(opts.uiDir, opts.writes())
}

// isBlankProjectDir 报告 cwd 能不能直接当一个新项目根: 目录里没有内容(.git 这类版本控制
// 元数据不算 —— git init 之后建仓是常见顺序)、不在别人的 Go 模块里(否则组件该并进那个
// 模块, 而不是就地再起一个 go.mod)、目录名能当项目名, 且组件目录落在它内部。
func isBlankProjectDir(cwd, uiDir string) bool {
	if projectNameFromDir(cwd) == "" || findGoMod(cwd) != "" {
		return false
	}
	// 组件目录跳出去(绝对路径 / .. / 当前目录根)时, 用户要的是"组件放别处/根上",
	// 不是在这里建项目 —— 保持旧行为, 不猜。
	if uiDir == "." || checkUIWithinProject(uiDir) != nil {
		return false
	}
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			return false
		}
	}
	return true
}

func initProject(dir string, w writeOpts) {
	// 组件目录里那份给 AI 助手的 AGENTS.md 由 scaffoldUI 写(与其它下发文档同一套基线判定)。
	if err := scaffoldUI(dir, w, true); err != nil {
		fatal("%v", err)
	}
	if w.dryRun {
		info("\n以上是预览: 没有创建任何文件/目录, 也没有联网动依赖")
		return
	}
	agentsPath := filepath.Join(dir, "AGENTS.md")
	info("\n完成。接下来:")
	info("  1. go get github.com/twgh/xui github.com/twgh/xcgui")
	info("  2. 在代码中 import 目录 %s 下的组件 (包名 ui, 与目录名无关, 可自行起别名)", dir)
	info("  3. 主题在 %s 中, 随意修改", filepath.Join(dir, "theme.go"))
	info("  4. 改组件/主题前先看 %s (令牌表、每个组件的画法; xcgui 的坑在配套的 %s)",
		filepath.Join(dir, "DESIGN.md"), filepath.Join(dir, "DESIGN-PITFALLS.md"))
	info("  5. AI 助手先读 %s (组件用法/添加更新/改组件规则)", agentsPath)
	if findGoMod(dir) == "" {
		info("\n当前目录不在任何 Go 模块里: 第 1 步的 go get 要先有 go.mod —— 自己 go mod init <模块路径>, 或直接要一套带 main.go 的骨架: xui init -n <项目名>(空目录里 xui init 就是这条)。")
	}
}

// placeFile 按 classifyPlace 的判定写一个"由 CLI 下发"的文件, 并更新清单里的基线。
// 组件源码与 CLI 下发的文档走的都是这一条, 所以"你改过就保留、没改过跟着版本走"
// 在项目里只有一套行为、一套措辞。返回是否被跳过(给汇总用)。
func placeFile(dst, rel string, data []byte, inst *installation, overwrite, dryRun bool) (skipped bool) {
	local, err := os.ReadFile(dst)
	missing := os.IsNotExist(err)
	if err != nil && !missing {
		fatal("读取 %s 失败: %v", dst, err)
	}
	baseline, hasBaseline := inst.Files[rel]
	switch classifyPlace(!missing, local, data, baseline, hasBaseline) {
	case placeCreate:
		if dryRun {
			info("将创建 %s", dst)
			return false
		}
		writeFileOrFatal(dst, data)
		inst.Files[rel] = sha256Hex(data)
		info("创建 %s", dst)
	case placeSame:
		if !dryRun {
			inst.Files[rel] = sha256Hex(data)
		}
		info("已一致 %s", dst)
	case placeUpgrade:
		if dryRun {
			info("将更新 %s (版本升级)", dst)
			return false
		}
		writeFileOrFatal(dst, data)
		inst.Files[rel] = sha256Hex(data)
		info("更新 %s (版本升级)", dst)
	case placeEdited, placeUnknown:
		if !overwrite {
			verb := "跳过"
			if dryRun {
				verb = "将跳过"
			}
			if !hasBaseline {
				warn("%s 内容与 CLI 下发的不同, 且没有安装清单基线, 无法判断来源, %s (确认本地无改动后加 -o/--overwrite 覆盖)", dst, verb)
				return true
			}
			warn("%s 本地已修改, %s (确认后加 -o/--overwrite 覆盖)", dst, verb)
			return true
		}
		lost := " (本地修改已丢弃)"
		if !hasBaseline {
			lost = " (无安装清单基线, 无法判断是否改过)"
		}
		if dryRun {
			info("将覆盖 %s%s", dst, lost)
			return false
		}
		writeFileOrFatal(dst, data)
		inst.Files[rel] = sha256Hex(data)
		info("覆盖 %s%s", dst, lost)
	}
	return false
}

// scaffoldUI 写 ui 目录: theme.go + DESIGN.md + DESIGN-PITFALLS.md + AGENTS.md + 基础组件。
// 普通 init 与 init -n 共用。
//
//   - overwrite = 允许丢弃本地改动(add 的 -o);
//   - dryRun = 只说将做什么, 不建目录、不写文件;
//   - dirAgents = 组件目录里也放一份给 AI 助手的 AGENTS.md(普通 init 才放: init -n 写的是
//     项目根那份, 已在模板文件的清单里)。
//
// 下发的文档与组件源码共用同一套基线判定: 以前它们"存在就永远跳过", 于是
// 第一次 init 那一刻文档就定版了, 之后 CLI 修了文档里的错也传不到已有项目。
// theme.go 是明确给用户改的(改主题就在那里), 所以不纳入 —— 它存在就跳过, 一次都不覆写。
func scaffoldUI(dir string, w writeOpts, dirAgents bool) error {
	// 数据源在这里拉一次: 文档与基础组件同源, 不能再让下游各自拉一遍。
	src, reg := loadRegistry()
	// dry-run 与 install 同一条纪律: 不建目录、不写文件, 只说将做什么。
	if !w.dryRun {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建目录失败: %v", err)
		}
	}
	themePath := filepath.Join(dir, "theme.go")
	switch {
	case fileExists(themePath):
		info("跳过 %s (已存在)", themePath)
	case w.dryRun:
		info("将创建 %s", themePath)
	default:
		if err := os.WriteFile(themePath, []byte(themeTemplate), 0o644); err != nil {
			return fmt.Errorf("写入 theme.go 失败: %v", err)
		}
		info("创建 %s", themePath)
	}

	docs := []string{"DESIGN.md", "DESIGN-PITFALLS.md"}
	if dirAgents {
		// 目录级 AGENTS.md 只有普通 init 要; init -n 写的是项目根那份(在 project_files 里)。
		docs = append(docs, "AGENTS.md")
	}
	skipped := placeDocs(src, docs, dir, w, true)

	// 拷入全部基础组件, 开箱即用(window 是快速开始示例用的主窗口封装)
	addWith(src, reg, componentOptions{names: baseComponents, uiDir: dir, overwrite: w.overwrite, noInstall: w.noInstall, dryRun: w.dryRun})
	reportSkipped(skipped, w.dryRun, false)
	return nil
}

// fileExists 只问"路径在不在"(读一次 Stat)。
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// initNewProject 创建完整项目目录: go.mod + main.go + README + .gitignore + 组件目录。
//
// uiSub 是组件目录相对项目根的名字(`-p` 指定, 默认 ui) —— 以前写死 ui/, 想要 components/
// 的人只能 init 完再手工删掉 ui/, 而 README/AGENTS 与 main.go 的 import 都说的是 ui。
// 目录名在这一步定下: 安装清单落在组件目录里, 之后每条命令靠 `-p` 指过去。
//
// 两个开关分开(以前挤在 -f 一个旗上, 于是"往非空目录里装"必然连带"丢掉你的改动"):
//   - force     = 目标目录已存在且非空时才需要, 只负责开这道闸门;
//   - overwrite = 允许丢弃本地改动, 判定与 add/update 的 -o 完全同一套(见 classifyPlace):
//     与上次生成一致 → 随模板升级直接替换; 基线对不上 → 你改过, 默认保留; 没有基线 →
//     只说"判断不了来源", 不断言你改过。go.mod / main.go / README.md 都是会被改的文件,
//     光凭"存在"就覆写等于把用户代码抹了。
//
// noInstall = 跳过自动 go mod tidy(--no-install); dryRun = 只预览, 不建目录、不写文件、
// 不联网(与 add --dry-run 同一套纪律: 预览不该有副作用)。
//
// 目录规则:
//   - `-n .` 直接把骨架铺在当前目录(项目名取当前目录名);
//   - 当前目录本身是空的、且目录名 == -n 的项目名时, 同样铺在当前目录 ——
//     否则在 `D:\...\sitless` 里执行 `xui init -n sitless` 会生成 sitless/sitless。
func initNewProject(name, uiSub string, force bool, w writeOpts) error {
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
			return fmt.Errorf("目录 %s 已存在且非空 (确认后加 --force 写入; 已有的模板文件默认保留, 要丢弃本地改动再加 -o/--overwrite)", dir)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取目录 %s 失败: %v", dir, err)
	} else if w.dryRun {
		info("将创建项目目录 %s", dir)
	}
	uiDir := filepath.Join(dir, uiSub)
	if !w.dryRun {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建项目目录失败: %v", err)
		}
		// 模板文件的基线记在同一份安装清单里(清单落在组件目录), 所以先确保目录存在。
		if err := os.MkdirAll(uiDir, 0o755); err != nil {
			return fmt.Errorf("创建目录失败: %v", err)
		}
	}
	// 清单读得到就读(预览时目录还不存在也不影响: 拿空基线, 全部走"将创建")。
	inst := loadManifest(uiDir)
	// 重跑 init -n 时忘了带 -p(或换了目录名), 会在项目里另起一套组件与清单, 而旧的
	// main.go / README 基线在新清单里找不到 —— 之后每条命令都报"无法判断来源"。
	// 这里只提醒, 不阻断: 一个仓库里放两套组件是罕见但合法的。
	if len(inst.Project) == 0 {
		if other := otherProjectComponentDir(dir, uiSub); other != "" {
			warn("项目根已经有组件目录 %s (它的安装清单里存着本项目的骨架基线), 这次的组件目录是 %s —— 两个目录各存一份清单, 之后每条命令的 -p 别混用", other, uiSub)
		}
	}

	module := moduleNameFromProject(name)
	xuiVer, xcguiVer := templateVersions()
	files := []struct{ name, body string }{
		{"go.mod", fmt.Sprintf(goModTemplate, module, xuiVer, xcguiVer)},
		{"main.go", renderUI(fmt.Sprintf(mainGoTemplate, module, strconv.Quote(name)), uiSub)},
		{"README.md", renderUI(fmt.Sprintf(readmeTemplate, name), uiSub)},
		{"AGENTS.md", renderUI(fmt.Sprintf(agentsProjectTemplate, module), uiSub)},
		{".gitignore", gitignoreTemplate},
	}
	skipped := 0
	// delivered 记下"这次真的交付了"哪些模板 —— 末尾要按盘上的内容重算它们的基线(见下面)。
	delivered := map[string]bool{}
	// 预览用的动词: 与实际写入同一套判定, 只是前面加"将"、不落盘。
	verb := func(created string) string {
		if w.dryRun {
			return "将" + created
		}
		return created
	}
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		data := []byte(f.body)
		local, err := os.ReadFile(path)
		missing := os.IsNotExist(err)
		if err != nil && !missing {
			return fmt.Errorf("读取 %s 失败: %v", path, err)
		}
		baseline, hasBaseline := inst.Project[f.name]
		switch classifyPlace(!missing, local, data, baseline, hasBaseline) {
		case placeCreate:
			if !w.dryRun {
				if err := os.WriteFile(path, data, 0o644); err != nil {
					return fmt.Errorf("写入 %s 失败: %v", path, err)
				}
				inst.Project[f.name] = sha256Hex(data)
				delivered[f.name] = true
			}
			info("%s %s", verb("创建"), path)
		case placeSame:
			if !w.dryRun {
				inst.Project[f.name] = sha256Hex(data)
				delivered[f.name] = true
			}
			info("已一致 %s", path)
		case placeUpgrade:
			// 与上次生成的一字不差 ⇒ 差异只可能来自 CLI 模板升级, 安全替换。
			if !w.dryRun {
				if err := os.WriteFile(path, data, 0o644); err != nil {
					return fmt.Errorf("写入 %s 失败: %v", path, err)
				}
				inst.Project[f.name] = sha256Hex(data)
				delivered[f.name] = true
			}
			info("%s %s (模板升级)", verb("更新"), path)
		case placeEdited, placeUnknown:
			if !w.overwrite {
				// 闸门已经开了(--force), 但丢弃改动是另一个决定 —— 默认保留你的文件。
				skipped++
				if !hasBaseline {
					warn("%s 与模板不同, 且没有生成时的基线, 无法判断来源, %s (确认本地无改动后加 -o/--overwrite 覆盖)", path, verb("跳过"))
					continue
				}
				warn("%s 本地已修改, %s (确认后加 -o/--overwrite 覆盖)", path, verb("跳过"))
				continue
			}
			lost := " (本地修改已丢弃)"
			if !hasBaseline {
				lost = " (无生成基线, 无法判断是否改过)"
			}
			if !w.dryRun {
				if err := os.WriteFile(path, data, 0o644); err != nil {
					return fmt.Errorf("写入 %s 失败: %v", path, err)
				}
				inst.Project[f.name] = sha256Hex(data)
				delivered[f.name] = true
			}
			info("%s %s%s", verb("覆盖"), path, lost)
		}
	}
	if !w.dryRun {
		// 预览不落清单: 写清单本身就是副作用(还会覆盖旧条目)。
		if err := inst.save(uiDir); err != nil {
			warn("写入安装清单失败: %v", err)
		}
	}
	// init 没有 --strict(它是建仓的一步, 跳过模板该由用户决定), 只要那一句汇总。
	reportSkipped(skipped, w.dryRun, false)
	// init -n 不写组件目录那份 AGENTS.md: 它的项目根 AGENTS.md 已经在上面的模板清单里。
	if err := scaffoldUI(uiDir, w, false); err != nil {
		return err
	}
	// 图标/版本资源: winres/ 写好即生效, .syso 由 xc 生成; --no-install 不联网装 xc。
	scaffoldProjectWinres(dir, name, !w.noInstall, w.dryRun)

	depsReady := false
	if w.dryRun {
		info("\n(预览) 将执行: 在 %s 跑 go mod tidy", dir)
		info("以上是预览: 没有创建任何文件/目录, 也没动依赖; 去掉 --dry-run 才会真的生成。")
		return nil
	}
	if !w.noInstall {
		info("\n正在安装依赖 (go mod tidy)...")
		if err := installGoModules(dir); err != nil {
			warn("依赖安装失败: %v", err)
		} else {
			depsReady = true
			info("依赖安装完成")
		}
	}
	// 基线按"交付完成那一刻盘上的内容"记, 而不是"我们打算写的字节": 同一次 init -n 后面
	// 自己就会改写 go.mod(ensureRuntime 的 go get、以及 go mod tidy 都会动它), 按原样记
	// 就必然与盘上一致不上 —— 下次 `init -n --force` 会说"本地已修改, 跳过", 可用户根本没
	// 动过。只重算这次真的交付了的那些: 被跳过的那份(用户改过)不许洗成基线, 否则下次模板
	// 升级会连用户的改动一起替换掉。清单要重新加载 —— scaffoldUI 里的 add 已经存过一份。
	if len(delivered) > 0 {
		man := loadManifest(uiDir)
		for f := range delivered {
			data, err := os.ReadFile(filepath.Join(dir, f))
			if err != nil {
				continue
			}
			man.Project[f] = sha256Hex(data)
		}
		if err := man.save(uiDir); err != nil {
			warn("写入安装清单失败: %v", err)
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

// otherProjectComponentDir 找 root 下另一个"CLI 交付过项目骨架"的组件目录(清单里有
// project_files), 相对 root 返回; 没有返回 ""。只扫三层: 组件目录通常是根下一层
// (ui / components), 深了既慢又容易误报。
func otherProjectComponentDir(root, uiSub string) string {
	return findComponentDir(root, root, uiSub, 0)
}

func findComponentDir(root, dir, uiSub string, depth int) string {
	if depth > 2 {
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		switch e.Name() {
		case ".git", "node_modules", "vendor", "winres":
			continue
		}
		p := filepath.Join(dir, e.Name())
		rel, err := filepath.Rel(root, p)
		if err != nil {
			continue
		}
		relSlash := filepath.ToSlash(rel)
		// 目标目录自己(及其子树)不参与比较 —— 它就是这次要写的那套。
		if relSlash == uiSub || strings.HasPrefix(relSlash, uiSub+"/") {
			continue
		}
		if len(loadManifest(p).Project) > 0 {
			return relSlash
		}
		if got := findComponentDir(root, p, uiSub, depth+1); got != "" {
			return got
		}
	}
	return ""
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
func installGoModules(dir string) error {
	return runGo(dir, "mod", "tidy")
}

// runGoOutput 跑 go 子命令并取标准输出(给 `go env` / `go list` 这类查询用)。
func runGoOutput(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	return string(out), err
}

// runGo 在 dir 里跑 go 子命令。GOWORK=off: 避免项目被父目录的 go.work 接管,
// 依赖解析只按它自己的 go.mod(否则拿到的版本是工作区里的那份, 结果不可信)。
func runGo(dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", args...)
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

// ---------------------------------------------------------------- 配套运行时版本

// runtimeModule 是组件源码 import 的运行时模块。组件从 main 现拉、运行时只能按 tag 取:
// main 上的组件用了新加的运行时助手而项目还钉着旧 tag 时, 装完直接 undefined 编译不过。
// 配套版本写在 registry.json 的 runtime 字段 —— 它和组件是同一次下发的, 所以一定对得上。
const runtimeModule = "github.com/twgh/xui"

// goGetModule / tidyModules 是测试缝: 单测换成桩, 不依赖网络也不动用户的 go.sum。
var (
	goGetModule = func(dir, spec string) error { return runGo(dir, "get", spec) }
	tidyModules = func(dir string) error { return runGo(dir, "mod", "tidy") }
)

// findGoMod 从 dir 向上找最近的 go.mod(组件目录一般就在项目里); 找不到返回 ""。
func findGoMod(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	for {
		p := filepath.Join(abs, "go.mod")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}

// goModRuntimePin 读 go.mod 的模块路径、runtimeModule 的 require 版本与是否被 replace。
// 只认常见写法: require/replace 的单行式与括号块式(块里一行一个模块路径)。
// ver 为空 = 这个项目还没要求该模块。
func goModRuntimePin(path string) (module, ver string, replaced bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false, err
	}
	block := ""
	for _, raw := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(raw)
		if i := strings.Index(s, "//"); i >= 0 {
			s = strings.TrimSpace(s[:i])
		}
		if s == "" {
			continue
		}
		if s == ")" {
			block = ""
			continue
		}
		fields := strings.Fields(s)
		if len(fields) == 2 && (fields[0] == "require" || fields[0] == "replace") && fields[1] == "(" {
			block = fields[0]
			continue
		}
		if block != "" {
			switch {
			case block == "require" && fields[0] == runtimeModule && len(fields) >= 2:
				ver = fields[1]
			case block == "replace" && fields[0] == runtimeModule:
				replaced = true
			}
			continue
		}
		switch {
		case fields[0] == "module" && len(fields) >= 2:
			module = fields[1]
		case fields[0] == "require" && len(fields) >= 3 && fields[1] == runtimeModule:
			ver = fields[2]
		case fields[0] == "replace" && len(fields) >= 2 && fields[1] == runtimeModule:
			replaced = true
		}
	}
	return module, ver, replaced, nil
}

// semverParts 取 vX.Y.Z 的三段数字, 剥掉 -伪版本 与 +dirty 后缀; 认不出来返回 ok=false。
func semverParts(v string) (maj, min, patch int, ok bool) {
	body := strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(body, "-+"); i >= 0 {
		body = body[:i]
	}
	fields := strings.Split(body, ".")
	if len(fields) != 3 {
		return 0, 0, 0, false
	}
	nums := []int{0, 0, 0}
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil || f == "" {
			return 0, 0, 0, false
		}
		nums[i] = n
	}
	return nums[0], nums[1], nums[2], true
}

// isReleaseTag 认严格的 vX.Y.Z —— go get @vX.Y.Z 要的形态, 伪版本与 +dirty 都不收。
func isReleaseTag(v string) bool {
	maj, min, patch, ok := semverParts(v)
	return ok && v == fmt.Sprintf("v%d.%d.%d", maj, min, patch)
}

// versionBefore 报告 a 是否低于 b。a 认不出数字时当作"低于" —— 宁可多抬一次,
// 也不要放着一个大概率编不过的旧运行时不管(b 由 registry 给出, 格式是校验过的)。
func versionBefore(a, b string) bool {
	am, amin, aparse, aok := semverParts(a)
	bm, bmin, bpatch, bok := semverParts(b)
	if !aok || !bok {
		return true
	}
	switch {
	case am != bm:
		return am < bm
	case amin != bmin:
		return amin < bmin
	default:
		return aparse < bpatch
	}
}

// insideCwd 报告 dir 是否在当前目录或其下(取的是"用户站着的那个项目"这条边界:
// `xui -c D:\proj ...` 已经把 cwd 定在项目根, 所以正常安装时组件目录的最近 go.mod 必然在其中)。
func insideCwd(dir string) bool {
	wd, err := os.Getwd()
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(wd, dir)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ensureRuntime 把项目 go.mod 的运行时依赖抬到 registry 声明的配套版本。
// 只在真正写文件的路径上调用(add/update/init 的写入分支); dryRun 只预览,
// autoInstall=false(init --no-install)时只打印要跑的那条命令。
//
// 找 go.mod 是"从组件目录向上"的, 所以要挡住误伤: 在临时目录里跑 `xui add btn` 时,
// 向上可能命中某个**无关**的父模块并改它的依赖。判据两条, 满足其一才动手 ——
// 那个 go.mod 在当前目录之下(就是用户站着的项目), 或者它本来就 require 了本模块。
func ensureRuntime(uiDir, floor string, dryRun, autoInstall bool) {
	if floor == "" {
		return // 数据源没这个字段(老 tag / 镜像还没同步), 不动用户依赖
	}
	gm := findGoMod(uiDir)
	if gm == "" {
		if dryRun {
			// 预览 init -n 时 go.mod 本来就还不存在(它就是这次"将创建"的东西之一),
			// 这时说"没找到 go.mod, 跳过"会让人以为依赖不会被抬。
			info("(预览) 项目里还没有 go.mod; 真跑时会把配套运行时抬到 %s %s", runtimeModule, floor)
			return
		}
		info("没找到 go.mod, 跳过运行时版本检查 (配套运行时是 %s %s)", runtimeModule, floor)
		return
	}
	module, ver, replaced, err := goModRuntimePin(gm)
	if err != nil {
		warn("读取 %s 失败, 跳过运行时版本检查: %v", gm, err)
		return
	}
	gmDir := filepath.Dir(gm)
	switch {
	case module == runtimeModule:
		// 向上找到了 xui 仓库自己的 go.mod(在仓库里跑 CLI), 不改仓库的依赖。
		return
	case !insideCwd(gmDir) && ver == "":
		// 既不在当前目录之下(不是用户站着的那个项目), 又没声明本模块 —— 它跟这次安装的
		// 组件没关系(在临时目录 / 别人 monorepo 的上层跑 CLI), 动它的依赖是越界。
		info("%s 在当前目录之外且没有要求 %s, 不动它的依赖 (配套运行时是 %s)", gmDir, runtimeModule, floor)
		return
	case replaced:
		// 本地联调(replace 到工作副本)时版本由开发者自己管, 抬 require 只会误导。
		info("%s 在 go.mod 里被 replace, 运行时版本由你自己管 (配套运行时是 %s)", runtimeModule, floor)
		return
	case !versionBefore(ver, floor):
		return
	}
	from := "未声明"
	if ver != "" {
		from = ver
	}
	if dryRun {
		info("将升级运行时 %s %s → %s (go get %s@%s, 再 go mod tidy 清掉 go.sum 里的旧条目)",
			runtimeModule, from, floor, runtimeModule, floor)
		return
	}
	if !autoInstall {
		info("与这批组件配套的运行时是 %s %s (当前 %s); 本次不联网改依赖, 自己跑: go get %s@%s, 再 go mod tidy 清掉 go.sum 里的旧条目",
			runtimeModule, floor, from, runtimeModule, floor)
		return
	}
	info("升级运行时 %s %s → %s ...", runtimeModule, from, floor)
	if err := goGetModule(gmDir, runtimeModule+"@"+floor); err != nil {
		warn("自动升级运行时失败: %v (手动跑: 在 %s 执行 go get %s@%s)", err, gmDir, runtimeModule, floor)
		return
	}
	// go get 只往 go.sum 里**追加**新版本的 h1/go.mod, 上个版本的条目会原样留着 ——
	// 用户看到"go.sum 里还印着旧版本", 会以为这次升级没升干净。go.sum 的过期条目
	// 只有 go mod tidy 会清, 所以升级后补一步。tidy 失败不算升级失败(go.mod 已经抬上去了),
	// 只提示手动跑: 大项目 tidy 可能超过命令超时, 不该因此把整次安装判失败。
	info("整理依赖 (go mod tidy) ...")
	if err := tidyModules(gmDir); err != nil {
		warn("go mod tidy 失败: %v (运行时已升到 %s; go.sum 里可能还留着 %s 的条目, 手动在 %s 跑一次 go mod tidy)",
			err, floor, from, gmDir)
		info("运行时已升到 %s (go.mod 已更新)", floor)
		noteVersionSplit(displayVersion(), floor)
		return
	}
	info("运行时已升到 %s (go.mod 与 go.sum 一起更新, 旧版本条目已清)", floor)
	noteVersionSplit(displayVersion(), floor)
}

// noteVersionSplit 给"运行时刚升到一个比 CLI 更新的 tag"这一句配一句消歧说明: 同一个 import
// 路径上挂着两个版本号(go.mod 里的运行时库 vs CLI 二进制), 用户很容易读成"CLI 没装新、
// 这次没升干净"。两个号的关系不需要解释时(相等 / CLI 更新 / 本地构建)一句都不打。
func noteVersionSplit(current, floor string) {
	if !isReleaseTag(current) || !isReleaseTag(floor) || !versionBefore(current, floor) {
		return
	}
	info("顺带说明: 刚升的是运行时库 %s(项目 import 的那份), 与 CLI 二进制是两回事(xui --version = %s) —— "+
		"组件与设计文档随数据源下发, 不必跟着每个 tag 升级 CLI, 只有 CLI 自己改了才会提示你 xui self-update",
		runtimeModule, current)
}

// ---------------------------------------------------------------- CLI 自我升级

// cliModule 是 CLI 自己的包路径(go install 的目标)。
const cliModule = runtimeModule + "/cmd/xui"

// goInstall / goListLatest / goBinDir / readBinaryVersion 是测试缝: 单测换成桩,
// 既不碰网络也不动本机 GOBIN。
var (
	goInstall         = func(spec string) error { return runGo(".", "install", spec) }
	goListLatest      = newestTagViaGoList
	goBinDir          = queryGoBinDir
	readBinaryVersion = binaryVersion
	// registryCLIFloor 取数据源声明的"CLI 最后一次改动在哪个 tag"(registry.json 的 cliChange):
	// --check 要回答"最新 tag 里到底有没有 CLI 的改动", 光比版本号答不了。单测换成桩, 不联网。
	registryCLIFloor = func() string { _, reg := loadRegistry(); return reg.CLIChange }
)

// semverOK 只问"能不能解析出三段数字"(伪版本/裸版本号都算)。
func semverOK(v string) bool { _, _, _, ok := semverParts(v); return ok }

// isHexCommit 认 7~40 位十六进制的 commit 哈希。
func isHexCommit(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// cliInstallSpec 把用户给的版本规范成 `go install <pkg>@<版本>` 的目标:
// 空/latest 走 @latest, main/HEAD 走分支或引用, vX.Y.Z 与伪版本原样, 裸的 0.8.0 补 v,
// 7~40 位十六进制当 commit; 其余直接报错 —— 别跑一条注定失败的命令。
func cliInstallSpec(target string) (string, error) {
	t := strings.TrimPrefix(strings.TrimSpace(target), "@")
	switch {
	case t == "" || strings.EqualFold(t, "latest"):
		t = "latest"
	case strings.ContainsAny(t, " \t/\\"):
		return "", fmt.Errorf("版本 %q 里有空格或路径分隔符", target)
	case strings.EqualFold(t, "main") || strings.EqualFold(t, "master") || strings.EqualFold(t, "HEAD"):
	case isReleaseTag(t):
	case strings.HasPrefix(t, "v") && semverOK(t): // 伪版本 vX.Y.Z-0.<日期>-<hash>
	case semverOK(t): // 0.8.0 -> v0.8.0
		t = "v" + t
	case isHexCommit(t):
	default:
		return "", fmt.Errorf("认不出版本 %q (支持 latest / main / HEAD / vX.Y.Z / commit 短哈希)", target)
	}
	return cliModule + "@" + t, nil
}

// isTempBuildPath 认出 go run / go test 的一次性构建产物(路径里有 go-buildXXXX 段)。
// 那种二进制本来就没"安装"过, 自我升级没有意义(升级完 GOBIN 里那份, 下次还是 go run)。
// 只认这个标记, 不认"在临时目录下": 有人把 xui 装在自定义的 Temp 里, 不能把他挡掉。
func isTempBuildPath(exe string) bool {
	return strings.Contains(filepath.ToSlash(exe), "/go-build")
}

// samePath 判断两个路径是不是同一个文件(Windows 不区分大小写)。
func samePath(a, b string) bool {
	normalize := func(p string) string {
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = filepath.Clean(p)
		}
		if runtime.GOOS == "windows" {
			return strings.ToLower(abs)
		}
		return abs
	}
	return normalize(a) == normalize(b)
}

// goBinDir 返回 go install 的落地目录: GOBIN, 没有就 GOPATH 第一条 + /bin。
func queryGoBinDir() (string, error) {
	if out, err := runGoOutput("env", "GOBIN"); err == nil {
		if v := strings.TrimSpace(out); v != "" {
			return v, nil
		}
	}
	out, err := runGoOutput("env", "GOPATH")
	if err != nil {
		return "", err
	}
	list := filepath.SplitList(strings.TrimSpace(out))
	if len(list) == 0 || list[0] == "" {
		return "", fmt.Errorf("`go env` 没给出 GOBIN 或 GOPATH")
	}
	return filepath.Join(list[0], "bin"), nil
}

// newestTagViaGoList 问模块代理 "@latest 现在解析到哪个版本"。查的是**模块路径**
// (github.com/twgh/xui)而不是 CLI 的包路径 —— cmd/xui 不是独立模块, 查它会一无所获。
// 用 go list 而不是抓 tag 接口: 走的就是用户机器上的 GOPROXY, 结果与他 go install @latest 一致
// (包括代理缓存滞后这一点)。
func newestTagViaGoList() (string, error) {
	out, err := runGoOutput("list", "-m", runtimeModule+"@latest")
	if err != nil {
		return "", err
	}
	fields := strings.Fields(out)
	if len(fields) < 2 {
		return "", fmt.Errorf("没查到 %s 的 @latest 版本", runtimeModule)
	}
	return fields[len(fields)-1], nil
}

// retireForReplace 把目标 exe 改名成 .old, 腾出路径让 go install 写新文件。
// Windows 允许改名正在运行的映像但不允许覆写它, Linux 能改名不能覆写(ETXTBSY),
// 所以"改名让路"是两边通用的办法; 改名失败就回 false 让调用方报错。
func retireForReplace(path string) bool {
	old := path + ".old"
	if err := os.Remove(old); err != nil && !os.IsNotExist(err) {
		return false
	}
	return os.Rename(path, old) == nil
}

// binaryVersion 跑一次指定二进制取版本串(升级后核对新文件是否真到位)。
func binaryVersion(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(out))
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	if line == "" {
		return "", fmt.Errorf("%s version 没有输出", path)
	}
	return strings.TrimPrefix(line, "xui "), nil
}

// selfUpdate 升级 CLI 本体: 先弄清"我是谁"(可执行文件路径、当前版本),
// 再把流程交给可测内核 selfUpdateAt。
func selfUpdate(target string, checkOnly, jsonMode bool) {
	exe, err := os.Executable()
	if err != nil {
		fatal("取不到当前可执行文件, 没法自我升级: %v (手动跑: go install %s@latest)", err, cliModule)
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	if err := selfUpdateAt(exe, displayVersion(), target, checkOnly, jsonMode); err != nil {
		fatal("%v", err)
	}
}

// selfUpdateAt 是自我升级的流程。默认目标 latest, 也可以指定 tag / main / commit。
// 两个坑: ① 正在运行的 exe 不许就地覆写(Windows 拒绝、Linux 是 ETXTBSY) —— go install
// 失败时把旧文件改名让路再试一次; ② 成没成以"跑新二进制拿到的版本"为准,
// 不拿"命令没报错"当证据(go install 失败也可能只留下半截状态)。
func selfUpdateAt(exe, current, target string, checkOnly, jsonMode bool) error {
	if isTempBuildPath(exe) {
		return fmt.Errorf("现在跑的是临时构建(%s), 它不是安装出来的文件, 不能自我升级; 装一次以后就能直接敲 xui: go install %s@latest", exe, cliModule)
	}
	dir, err := goBinDir()
	if err != nil {
		warn("取不到 GOBIN/GOPATH: %v", err)
	}
	installed := ""
	if dir != "" {
		installed = filepath.Join(dir, "xui"+exeExt())
	}
	if checkOnly {
		newest, err := goListLatest()
		if err != nil {
			return fmt.Errorf("查询最新版本失败: %w", err)
		}
		// 这几行就是 --check 的全部结果, 加 -s 也得打(否则命令等于什么都没查)。
		floor := registryCLIFloor()
		declared := isReleaseTag(floor)
		if !declared {
			// 旧数据源/镜像没声明 CLI 最后改动版本: 退化成"比 tag", 并把这一步说出来,
			// 免得用户以为最新 tag 真的改了 CLI。
			floor = newest
		}
		localBuild := !isReleaseTag(current)
		upgradable := !localBuild && isReleaseTag(floor) && versionBefore(current, floor)
		if jsonMode {
			printJSON(checkJSON{
				Current:    current,
				Latest:     newest,
				CLIFloor:   floor,
				LocalBuild: localBuild,
				Upgradable: upgradable,
				Command:    checkCommand(newest, localBuild, upgradable),
			})
			return nil
		}
		out("当前 CLI: %s", current)
		out("最新 tag: %s", newest)
		if declared {
			out("CLI 最后改动: %s", floor)
		} else {
			out("CLI 最后改动: 数据源没声明(只能按最新 tag 判断)")
		}
		switch {
		case localBuild:
			out("本地构建(%s)不算发布版; 想装正式版: xui self-update", current)
		case upgradable:
			out("可以升级: xui self-update %s", newest)
		case current == newest:
			out("已是最新")
		case declared && versionBefore(current, newest):
			// 落后于最新 tag, 但那个差值里没有 CLI 的改动: 组件与设计文档随数据源下发,
			// 不换 CLI 一样拿得到, 所以这里明确说"无需升级", 别让人白跑一次 go install。
			out("CLI 代码自 %s 起没变过, 无需升级 (最新 tag %s 只动组件与设计文档)", floor, newest)
		default:
			// 本地版本比 @latest 高(还没发布 / 代理滞后): 既不是最新, 也没什么可升的,
			// 不能什么都不说 —— 用户看到的只是两行版本号, 得自己比。
			out("本地版本 %s 不低于 @latest(%s), 无需升级", current, newest)
		}
		return nil
	}
	spec, err := cliInstallSpec(target)
	if err != nil {
		return err
	}
	info("升级 CLI: go install %s ...", spec)
	if err := goInstall(spec); err != nil {
		if installed == "" || !samePath(exe, installed) || !retireForReplace(installed) {
			return fmt.Errorf("升级失败: %w (手动跑: go install %s)", err, spec)
		}
		info("旧文件正被占用, 已改名让路, 重试 ...")
		if err := goInstall(spec); err != nil {
			return fmt.Errorf("升级失败: %w (手动跑: go install %s)", err, spec)
		}
	}
	if installed == "" {
		info("go install 已执行, 但取不到 GOBIN, 没法核对新版本")
		return nil
	}
	if _, err := os.Stat(installed); err != nil {
		return fmt.Errorf("go install 跑完了, 但 %s 不存在(检查 PATH / GOBIN): %w", installed, err)
	}
	next, err := readBinaryVersion(installed)
	if err != nil {
		warn("升级命令跑完了, 但核对新版本失败: %v", err)
		return nil
	}
	switch {
	case next == current:
		info("版本没变(%s): 模块代理的 @latest 可能滞后, 稍后再试或直接指定 tag (xui self-update <tag>)", current)
	default:
		info("CLI 已升级 %s → %s (%s)", current, next, installed)
		info("本次运行用的还是旧进程, 下次调用生效")
	}
	if !samePath(exe, installed) {
		warn("当前运行的是 %s, 不在 GOBIN(%s): 刚升级的是 GOBIN 那份, 确认 shell 里敲的 xui 指向它(where xui / echo $PATH)", exe, dir)
	}
	return nil
}

// exeExt 返回当前平台的可执行文件后缀。
func exeExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// ---------------------------------------------------------------- 升级提示

// updateCheckDisableEnv 非空就彻底关掉升级提示(CI、离线机、隐私偏好)。
const updateCheckDisableEnv = "XUI_NO_UPDATE_CHECK"

// cliChangeTag 记下本次数据源声明的"CLI 最后一次改动在哪个 tag"(registry.json 的 cliChange),
// 由 parseRegistry 写入, 命令跑完后的提示只读它。
//
// 这里以前是一套"后台子进程 + 用户缓存目录里的 update-check.json + 24h TTL": 依据(@latest
// 解析到哪个 tag)不在手头的数据里, 只能另开一次网络、还不能让前台等。现在依据跟着 registry.json
// 一起来, 而每条内容命令本来就拉了它 —— 于是既不用额外联网, 也不会再"每次发版都唠叨一遍":
// tag 变了不等于 CLI 变了, 组件与 DESIGN 文档都随数据源下发, 不换 CLI 照样拿得到最新的。
var cliChangeTag string

// updateNotice 只靠数据源声明的 CLI 改动版本决定提示语, 空串表示不提示。
// 本地/devel 构建不提示: 那是开发者在仓库里跑的, 比 tag 落后是常态, nag 也没法 self-update。
func updateNotice(current, floor string) string {
	if !isReleaseTag(current) || !isReleaseTag(floor) || !versionBefore(current, floor) {
		return ""
	}
	return fmt.Sprintf("xui CLI 有更新 (装的是 %s, 最后一次改动在 %s): xui self-update; 组件与设计文档随数据源下发, 不升级也拿得到 (关掉提示: 设 %s=1)",
		current, floor, updateCheckDisableEnv)
}

// notifyAvailableUpdate 在命令正常结束后, 按这次加载的数据源声明的 CLI 改动版本提示。
// 没拉过数据源的命令(version / info)没有依据, 自然不提示; self-update 刚跑完不再提示
// (否则升级完立刻被唠叨一遍); -s 与禁用变量都闭嘴。
func notifyAvailableUpdate(cmd string) {
	if quiet || cmd == "self-update" {
		return
	}
	if os.Getenv(updateCheckDisableEnv) != "" {
		return
	}
	if msg := updateNotice(displayVersion(), cliChangeTag); msg != "" {
		// 走 stderr: `xui view button.go > button.go` 这类重定向里, 提示语混进正文就是坏代码。
		warn("%s", msg)
	}
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

// cliDocs 是 CLI 下发的文档。它们**不在 registry.json 里**(不是组件), 但 DESIGN.md 与
// DESIGN-PITFALLS.md 的内容随数据源走 —— 与组件同源才不会把最新组件配上旧说明(见 docContent);
// AGENTS.md 由 CLI 内嵌模板渲染, 那部分才跟 CLI 版本走。名字可以直接写在 add/update 后面:
//
//	xui add DESIGN.md      # 只要文档(项目里没跑过 init 时补齐; 连带配套的 DESIGN-PITFALLS.md)
//	xui update AGENTS.md   # 只刷这一份, 不动任何组件
//
// 分工与 add/update 一贯的边界一致: add 可以创建, update 只刷已存在的。
var cliDocs = []string{"DESIGN.md", "DESIGN-PITFALLS.md", "AGENTS.md"}

// docCompanions 是必须成对交付的文档: DESIGN.md 第八节只剩编号索引, 完整条目(现象/根因/修法)
// 都在 DESIGN-PITFALLS.md 里 —— 少了配套文件, 说明书里的"见第八节 N"就是指向空处。
// 点名主文档时把配套一起带上(见 expandDocs)。
var docCompanions = map[string]string{"DESIGN.md": "DESIGN-PITFALLS.md"}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// expandDocs 把点名的文档展开成实际要交付的文件(主文档带上配套文件), 保持顺序并去重。
func expandDocs(docs []string) []string {
	seen := make(map[string]bool, len(docs))
	out := make([]string, 0, len(docs))
	addOne := func(name string) {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for _, name := range docs {
		addOne(name)
		if c := docCompanions[name]; c != "" {
			addOne(c)
		}
	}
	return out
}

// splitForWrite 只在写入模式下把名字拆成"文档目标 + 组件目标"; --view/--diff 是展示模式,
// 文档名在那儿当未知组件报错(它们不参与 diff)。基名比对, 所以 ui/DESIGN.md 也认。
func splitForWrite(opts componentOptions) (docs, comps []string) {
	if opts.view || opts.diff {
		return nil, opts.names
	}
	for _, n := range opts.names {
		if d := docTargetName(n); d != "" {
			docs = append(docs, d) // 用规范名, 后面按名字取内容/记基线才对得上(输入可以是小写或带路径)
			continue
		}
		comps = append(comps, n)
	}
	return docs, comps
}

func isDocTarget(name string) bool { return docTargetName(name) != "" }

// docTargetName 返回这个输入指的是哪份文档(拿**规范文件名**回, 因为报错里的建议命令要能直接照抄);
// 不是文档则返回 ""。基名比对, 所以 ui/DESIGN.md 也认。
func docTargetName(name string) string {
	base := filepath.Base(filepath.ToSlash(name))
	for _, d := range cliDocs {
		if strings.EqualFold(base, d) {
			return d
		}
	}
	return ""
}

// docContent 取要下发的文档内容。DESIGN.md / DESIGN-PITFALLS.md 与组件源码同源(数据源的 docs/
// 下), 拉不通才用 CLI 自带的那份:
//
// 组件跟 main 分支下发, 每个批次都往这两份文档里加条目, 而 CLI 是按 tag 锁的 —— 只从本地读会把
// "最新的组件 + 上个 tag 的说明"装进同一个项目, 而说明里的编号正是组件行为的依据(第八节索引)。
//
// 这两份不需要按目录名渲染: 它们的行文按"同目录裸文件名"写(见 AGENTS 的规矩), 本来就与
// 目录叫什么无关;需要渲染的只有 AGENTS.md(它会写"-p <目录>"这类路径)。
func docContent(src *source, name, uiDir string) ([]byte, error) {
	switch name {
	case "DESIGN.md", "DESIGN-PITFALLS.md":
		// 首行是 markdown 标题才认: 镜像限流时代理会回 200 + 一个 HTML/JSON 页面, 那玩意
		// 的 sha256 会被当成本地基线记进清单, 下次 update 就成了"用户改过、无法判断来源"。
		if data, err := src.read("docs/" + name); err == nil && docIsMarkdown(data) {
			return data, nil
		}
		warn("%s 没能从数据源拉到 (%s), 这次用 CLI 自带的版本", name, src.desc())
		data, err := os.ReadFile(filepath.Join(moduleRoot(), "docs", name))
		if err != nil {
			return nil, fmt.Errorf("读取 %s 失败: %w", name, err)
		}
		return data, nil
	case "AGENTS.md":
		return []byte(renderUI(agentsDirTemplate, docUIDir(uiDir))), nil
	}
	return nil, fmt.Errorf("不是可指名的文档: %s", name)
}

// docIsMarkdown 粗检文档正文: 这两份都以一级标题开头。
func docIsMarkdown(data []byte) bool {
	return strings.HasPrefix(strings.TrimLeft(string(data), " \t\r\n"), "#")
}

// docUIDir 把 -p 传来的组件目录规范成 AGENTS.md 里该写的那个路径。
// AGENTS.md 会把目录名写进正文("本目录叫 X" 与 "-p X"), 所以同一个目录必须只有一种写法:
// 不归一化时 `-p ui` / `-p .\ui` / `-p D:\proj\ui` 会生成三份不同内容 —— 同一份文件下次
// update 时被当成版本升级静默重写, 用户改过的那份则被误报"本地已修改"。
// 绝对路径优先折成相对当前目录(能照抄进命令); 落在当前目录之外就保留绝对路径。
func docUIDir(uiDir string) string {
	dir := strings.TrimSpace(uiDir)
	if dir == "" {
		return "ui"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return filepath.ToSlash(dir)
	}
	wd, err := os.Getwd()
	if err != nil {
		return filepath.ToSlash(dir)
	}
	rel, err := filepath.Rel(wd, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

// placeDocs 把点名的文档写进组件目录(create=false 时只刷已存在的, 不新增)。返回被跳过的条数。
// src 就是组件那份数据源(文档同源, 见 docContent)。
func placeDocs(src *source, docs []string, uiDir string, w writeOpts, create bool) int {
	inst := loadManifest(uiDir)
	skipped, changed := 0, false
	for _, name := range docs {
		dst := filepath.Join(uiDir, name)
		data, err := docContent(src, name, uiDir)
		if err != nil {
			warn("%v", err)
			continue
		}
		if !create && !fileExists(dst) {
			// update 不往目录里新增它没装过的东西(与它不装新组件是同一条纪律)。
			info("没有 %s (它由 xui init 写入; 要单独拿它: xui add %s)", dst, name)
			continue
		}
		if !w.dryRun {
			// 与 install 同一纪律: 要动手了才建目录(update 空跑不该留下空目录)。
			if err := os.MkdirAll(uiDir, 0o755); err != nil {
				warn("创建目录失败: %v", err)
				continue
			}
		}
		if placeFile(dst, name, data, &inst, w.overwrite, w.dryRun) {
			skipped++
		}
		changed = true
	}
	if !w.dryRun && changed {
		if err := inst.save(uiDir); err != nil {
			warn("写入安装清单失败: %v", err)
		}
	}
	return skipped
}

// existingDocs 返回组件目录里**已有**的那些文档 —— 裸 `xui update` 连它们一起刷新。
// 它们不是"新增": 是 init 交付过、也记在清单里的东西, 更新本来就该管。
// 缺的一份都不补也不吭声(补齐是 add/init 的活), 否则只用 add 装组件的项目每次 update 都被提醒。
func existingDocs(uiDir string) []string {
	var out []string
	for _, d := range cliDocs {
		if fileExists(filepath.Join(uiDir, d)) {
			out = append(out, d)
		}
	}
	return out
}

// add 拷贝组件(含依赖闭包); --view/--diff 只展示不写文件; 点名文档时只给文档
// (DESIGN.md 会连配套的 DESIGN-PITFALLS.md 一起交付, 见 expandDocs)。
func add(opts componentOptions) {
	src, reg := loadRegistry()
	addWith(src, reg, opts)
}

// addWith 是 add 的内核, 数据源由调用方给(init 的 scaffoldUI 已经拉过一次, 不该再来一次)。
func addWith(src *source, reg *registry, opts componentOptions) {
	docs, comps := splitForWrite(opts)
	names := comps
	if opts.all {
		names = allComponents(reg)
	}
	if len(names) == 0 && len(docs) == 0 {
		fatal("用法: xui add <组件...> | xui add --all  (组件见 xui list; 文档可点名 DESIGN.md / DESIGN-PITFALLS.md / AGENTS.md)")
	}
	order := resolveNames(reg, names)
	switch {
	case opts.view:
		requireMatched(viewFiles(src, reg, order, opts.viewFile, opts.json), opts.viewFile)
	case opts.diff:
		requireMatched(diffFiles(src, reg, order, opts.uiDir, opts.diffFile, opts.json), opts.diffFile)
	default:
		skipped := placeDocs(src, expandDocs(docs), opts.uiDir, writeOpts{overwrite: opts.overwrite, dryRun: opts.dryRun}, true)
		if len(order) > 0 {
			skipped += install(src, reg, order, opts.uiDir, opts.overwrite, opts.dryRun)
			// 组件是从 main 拉的, 运行时按 tag 走 —— 顺手把项目的依赖抬到与它配套的版本。
			ensureRuntime(opts.uiDir, reg.Runtime, opts.dryRun, !opts.noInstall)
		}
		reportSkipped(skipped, opts.dryRun, opts.strict)
	}
}

// update 更新已装组件: 省略组件名时 = uiDir 里已装的全部(按 registry 依赖闭包)
// **加上目录里已有的 DESIGN.md / DESIGN-PITFALLS.md / AGENTS.md**。
// --all 在这里会被解析器拒掉(update 不该顺手装新组件, 那是 add 的事);
// 点名文档则只刷指定的那几份(已存在的才刷; DESIGN.md 的配套文件缺失时补上, 见函数内)。
func update(opts componentOptions) {
	src, reg := loadRegistry()
	updateWith(src, reg, opts)
}

// updateWith 是 update 的内核(数据源由调用方给, 与 addWith 同理)。
func updateWith(src *source, reg *registry, opts componentOptions) {
	docs, comps := splitForWrite(opts)
	names := comps
	if len(names) == 0 && len(docs) == 0 {
		// 裸 update = "把这个目录里 CLI 给过的东西都刷新": 已装组件 + 已交付的文档。
		names = installedComponents(reg, opts.uiDir)
		docs = existingDocs(opts.uiDir)
		if len(names) == 0 && len(docs) == 0 {
			info("%s 下没有已安装的组件 (先用 xui add 装)", opts.uiDir)
			return
		}
	}
	order := resolveNames(reg, names)
	switch {
	case opts.view:
		requireMatched(viewFiles(src, reg, order, opts.viewFile, opts.json), opts.viewFile)
	case opts.diff:
		requireMatched(diffFiles(src, reg, order, opts.uiDir, opts.diffFile, opts.json), opts.diffFile)
	default:
		w := writeOpts{overwrite: opts.overwrite, dryRun: opts.dryRun}
		// create=false: update 只刷已经在目录里的东西, 不新增。
		skipped := placeDocs(src, docs, opts.uiDir, w, false)
		// 配套文件是 DESIGN.md 的一部分(第八节正文), 不算"update 顺手装新东西": 老项目里
		// DESIGN.md 已在、DESIGN-PITFALLS.md 还没有时补上 —— 刷新后的 DESIGN.md 第八节是
		// 索引, 缺配套文件就成了指向空处的说明书。
		if companion := docCompanions["DESIGN.md"]; containsString(docs, "DESIGN.md") &&
			!containsString(docs, companion) && fileExists(filepath.Join(opts.uiDir, "DESIGN.md")) {
			skipped += placeDocs(src, []string{companion}, opts.uiDir, w, true)
		}
		// 裸 update 已按 installedComponents 筛过, 但显式点名绕过了它 —— 名字叫 update、
		// 行为是 add(在空目录里 `xui update button` 会把 button 全套创建出来)。这里统一挡掉:
		// 一个文件都不在本地的组件算"没装过", 只指路不给装。
		order = dropUninstalled(reg, order, opts.uiDir)
		if len(order) > 0 {
			skipped += install(src, reg, order, opts.uiDir, opts.overwrite, opts.dryRun)
			// 组件是从 main 拉的, 运行时按 tag 走 —— 顺手把项目的依赖抬到与它配套的版本。
			ensureRuntime(opts.uiDir, reg.Runtime, opts.dryRun, !opts.noInstall)
		}
		reportSkipped(skipped, opts.dryRun, opts.strict)
	}
}

// dropUninstalled 去掉"组件目录里一个文件都没有"的组件: update 不装新组件(那是 add 的事)。
// 只挡全空的: 部分文件缺失仍交给 install 补齐 —— 跨版本升级常在组件里新增共享文件,
// 那些文件本地本来就不存在, 不补会直接编译不过(与 installedComponents 的"全都有才算已装"互补)。
func dropUninstalled(reg *registry, order []string, uiDir string) []string {
	kept := make([]string, 0, len(order))
	for _, name := range order {
		files := reg.Components[name].Files
		if len(files) > 0 && !anyFileLocal(uiDir, files) {
			warn("%s 没装过, update 只更新已装组件 —— 要装它: xui add %s", name, name)
			continue
		}
		kept = append(kept, name)
	}
	return kept
}

// anyFileLocal 报告这组文件里有没有一个已经在本地 —— "这个组件装过"的最低证据。
// 组件之间共享助手文件, 所以只能逐文件 Stat; 不要求全都在(缺的那部分正是 update 要补的)。
func anyFileLocal(uiDir string, files []string) bool {
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(uiDir, f)); err == nil {
			return true
		}
	}
	return false
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
	if d := docTargetName(name); d != "" {
		// 下发的文档不是组件, 是 add/update 可点名的目标; 在 --view/--diff 与
		// xui view 里它们没有内容可比(那些读的是 registry), 所以说清去哪儿能指名。
		// 建议里用规范文件名(回显用户输入的归一化形式会给出不能照抄的命令)。
		fatal("%q 不是组件, 是 CLI 下发的文档 (指名它请用 xui add %s 或 xui update %s; --view/--diff 只作用于组件)", name, d, d)
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

// manifestName 是安装清单的文件名(放在组件目录里, 点开头 —— go 工具链忽略)。
//
// 为什么需要它: 过去 install 只比较"本地内容 vs 远端内容", 分不清"用户改过"和"版本升级
// 导致源码变了" —— 跨版本 update 时每个文件都算"有差异", 默认全跳过, update 等于空转。
// 有清单后: 本地哈希 == 清单哈希 ⇒ 这份文件自安装后没人动过 ⇒ 可以安全升级(直接替换);
// 不相等 ⇒ 用户改过 ⇒ 保持原来的"警告 + 跳过", 要 -o 才覆盖。
const manifestName = ".xui-manifest.json"

// installation 是 .xui-manifest.json 的内容: 安装时的 CLI 版本 + 文件名 → sha256。
type installation struct {
	Version string            `json:"version,omitempty"`
	Files   map[string]string `json:"files"`
	// Project 是 init -n 生成的项目骨架文件(go.mod / main.go / README.md / AGENTS.md /
	// .gitignore, 键相对项目根)→ 写入时的 sha256。与组件共用这一份清单: 模板文件同样要
	// 分得清"本地改过"和"模板随 CLI 升级", 否则覆写只能靠猜。
	Project map[string]string `json:"project_files,omitempty"`
}

// loadManifest 读清单; 缺失或损坏都当作"没有清单"(行为退回保守的"警告跳过")。
func loadManifest(uiDir string) installation {
	m := installation{Files: map[string]string{}, Project: map[string]string{}}
	data, err := os.ReadFile(filepath.Join(uiDir, manifestName))
	if err != nil {
		return m
	}
	var got installation
	if json.Unmarshal(data, &got) != nil || got.Files == nil {
		return m
	}
	m.Files = got.Files
	m.Project = got.Project
	if m.Project == nil {
		m.Project = map[string]string{}
	}
	m.Version = got.Version
	return m
}

func (m installation) save(uiDir string) error {
	if m.Files == nil {
		m.Files = map[string]string{}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(uiDir, manifestName), append(data, '\n'), 0o644)
}

// sha256Hex 是清单里的文件指纹(记录"安装那一刻"的内容)。
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum[:])
}

// cliVersion 当前 CLI 的模块版本; go run 本地源码 / devel 构建时为空。
func cliVersion() string {
	bi, ok := rdebug.ReadBuildInfo()
	if !ok || bi.Main.Version == "" || !isReleaseVersion(bi.Main.Version) {
		return ""
	}
	return bi.Main.Version
}

// displayVersion 给用户看的版本串: 发布版 = vX.Y.Z; 本地/源码构建 = (devel),
// 有 VCS 信息时带短 commit, 便于确认"装的是不是刚提交的那版"。
func displayVersion() string {
	if v := cliVersion(); v != "" {
		return v
	}
	bi, ok := rdebug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	rev, dirty := "", false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				rev = s.Value[:7]
			}
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "(devel)"
	}
	if dirty {
		rev += "+dirty"
	}
	return "(devel) " + rev
}

func writeFileOrFatal(path string, data []byte) {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fatal("写入失败: %v", err)
	}
}

// placeAction 是"这个由 CLI 下发的文件该怎么处理"的判定结果。
type placeAction int

const (
	placeCreate  placeAction = iota // 本地没有 → 创建
	placeSame                       // 内容与新源码一致 → 不动
	placeUpgrade                    // 本地 == 安装基线 → 差异只来自版本升级, 可直接替换
	placeEdited                     // 基线对不上 → 本地改过, 默认跳过
	placeUnknown                    // 该文件没有基线 → 判断不了来源, 默认跳过
)

// classifyPlace 是 add/update 的组件文件与 init 的模板文件**共用**的差异判定。
//
// 差异来源只看"安装基线"(清单里记的上次写入哈希): 只有基线对不上才叫本地改过; 没有基线
// 时只能说"判断不了", 不能断言用户改过。判定必须在 overwrite 之前取 —— 否则 -o 会把纯
// 版本升级也说成"本地修改已丢弃", 让人以为自己在干净项目上丢了改动。
func classifyPlace(exists bool, local, remote []byte, baseline string, hasBaseline bool) placeAction {
	switch {
	case !exists:
		return placeCreate
	case string(local) == string(remote):
		return placeSame
	case hasBaseline && baseline == sha256Hex(local):
		return placeUpgrade
	case !hasBaseline:
		return placeUnknown
	default:
		return placeEdited
	}
}

// install 逐文件写入, 按 classifyPlace 的判定分五种情况:
//   - 本地没有 → 创建;
//   - 内容一致 → 跳过("已一致");
//   - 内容不同, 但本地哈希等于基线(安装后没人动过) → 版本升级, 直接替换(-o 也是这句);
//   - 内容不同, 基线对不上 → 本地改过, 警告跳过(-o 覆盖并报"本地修改已丢弃");
//   - 该文件没有基线(老项目 / 清单里没登记) → 同样跳过, 但只说"无法判断来源"。
//
// dryRun 只打印将要发生的动作, 不创建任何文件/目录、不写清单。
// 多个组件共享同一文件(separator/badge)时只处理一次。
func install(src *source, reg *registry, order []string, uiDir string, overwrite, dryRun bool) (skipped int) {
	if !dryRun {
		if err := os.MkdirAll(uiDir, 0o755); err != nil {
			fatal("创建目录失败: %v", err)
		}
	}
	inst := loadManifest(uiDir)
	prevVersion := inst.Version
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
			// 差异来源只看"安装基线"(清单里记的上次安装哈希), 判定与 init 的模板文件共用。
			baseline, hasBaseline := inst.Files[f]
			switch classifyPlace(!missing, old, data, baseline, hasBaseline) {
			case placeCreate:
				if dryRun {
					info("将创建 %s (%s)", dst, name)
					continue
				}
				writeFileOrFatal(dst, data)
				inst.Files[f] = sha256Hex(data)
				info("创建 %s (%s)", dst, name)
			case placeSame:
				if !dryRun {
					inst.Files[f] = sha256Hex(data)
				}
				info("已一致 %s", dst)
			case placeUpgrade:
				// 本地与"上次安装"逐字节一致 ⇒ 差异只可能来自版本升级, 安全替换。
				if dryRun {
					info("将更新 %s (版本升级)", dst)
					continue
				}
				writeFileOrFatal(dst, data)
				inst.Files[f] = sha256Hex(data)
				info("更新 %s (版本升级)", dst)
			case placeEdited, placeUnknown:
				if !overwrite {
					skipped++
					// dry-run 只是预览, 文案统一用"将…", 别让用户以为已经动过文件。
					verb := "跳过"
					if dryRun {
						verb = "将跳过"
					}
					if !hasBaseline {
						warn("%s 内容与远端不同, 且没有该文件的安装清单基线, 无法判断来源, %s (确认本地无改动后加 -o/--overwrite 覆盖)", dst, verb)
						continue
					}
					warn("%s 本地已修改, %s (确认后加 -o/--overwrite 覆盖)", dst, verb)
					continue
				}
				lost := " (本地修改已丢弃)"
				if !hasBaseline {
					// 老项目 / 该文件从没进过清单: 没有基线, 不能断言用户改过。
					lost = " (无安装清单基线, 无法判断是否改过)"
				}
				if dryRun {
					info("将覆盖 %s%s", dst, lost)
					continue
				}
				writeFileOrFatal(dst, data)
				inst.Files[f] = sha256Hex(data)
				info("覆盖 %s%s", dst, lost)
			}
		}
	}
	if dryRun {
		return skipped
	}
	if v := cliVersion(); v != "" {
		if prevVersion != "" && prevVersion != v {
			info("组件版本 %s → %s", prevVersion, v)
		}
		inst.Version = v
	}
	if err := inst.save(uiDir); err != nil {
		warn("写入安装清单失败: %v", err)
	}
	return skipped
}

// exitPartial 是 --strict 下"有文件被跳过"的退出码。与 fatal 的 1 分开, CI 才能区分
// "命令跑挂了(网络/权限)"与"跑通了但没更新干净(本地有改动)"。
const exitPartial = 2

// reportSkipped 汇总跳过: 默认只向 stderr 说一句就正常退出 —— 保守跳过是这套设计的
// 正确行为, 不是故障, 不该让 `xui update` 在人类手里报错。要它因此失败就加 --strict
// (CI/agent 用法: 跑一次 update --strict, 非 0 就说明有文件需要人工确认)。
func reportSkipped(skipped int, dryRun, strict bool) {
	if skipped == 0 {
		return
	}
	verb := "被跳过"
	if dryRun {
		verb = "会被跳过"
	}
	// 不写"本地有改动": 无基线的那类是"判断不了来源", 断言用户改过是错的(逐行已分开说)。
	warn("共 %d 个文件%s (原因见上面各行; 本地改动确认后加 -o/--overwrite 覆盖)", skipped, verb)
	if strict {
		os.Exit(exitPartial)
	}
}

// filterHit 判断 --view=/--diff= 后面那个值是否指向这个文件: 文件基名(button.go,
// 带路径的 ui/button.go 也认)或组件名(button / date-picker)都算命中。
// 用户刚在命令里打过组件名, 到这一步却必须改写成文件基名是不可用的门槛 ——
// 而 `--diff=button` 匹配不到文件时旧版会回一句"没有差异", 直接把"没匹配上"说成"已经是最新"。
func filterHit(name, file, filter string) bool {
	if filter == "" {
		return true
	}
	if filepath.Base(filter) == filepath.Base(file) {
		return true
	}
	return normalizeName(filter) == normalizeName(name)
}

// requireMatched 把"过滤值谁都没挨着"变成失败: 用户请求的那份源码/差异不存在, 与"未知组件"
// 同类, 不能以 0 退出(脚本会以为拿到了东西)。JSON 模式也照打结果再失败 —— 结构里
// matched=false 已经写清了原因, 退出码只是给 shell 判断。
func requireMatched(matched bool, filter string) {
	if !matched {
		fatal("没有匹配的文件: %s (给组件名或文件基名, 组件见 xui list)", filter)
	}
}

// viewFiles 打印组件的文件内容(不写本地); filter 非空时只打匹配的文件, json 出结构化。
// 正文走 out/outRaw(命令结果, 不受 -s 影响)。
// 返回是否命中过滤值 —— "一个都没匹配上"由调用方按失败处理(退出码 1), 与"未知组件"同类:
// 用户请求的那份源码不存在, 不能静悄悄地退出 0(脚本会以为拿到了东西)。
func viewFiles(src *source, reg *registry, order []string, filter string, jsonMode bool) bool {
	done := map[string]bool{}
	matched := false
	var files []sourceFileJSON
	for _, name := range order {
		for _, f := range reg.Components[name].Files {
			if done[f] {
				continue
			}
			done[f] = true
			if !filterHit(name, f, filter) {
				continue
			}
			matched = true
			content := string(readComponent(src, f))
			if jsonMode {
				files = append(files, sourceFileJSON{Name: f, Component: name, Content: content})
				continue
			}
			out("\n===== %s (%s) =====", f, name)
			outRaw(content)
		}
	}
	if jsonMode {
		if files == nil {
			files = []sourceFileJSON{}
		}
		printJSON(viewJSON{Source: src.desc(), Filter: filter, Matched: matched, Files: files})
		return matched
	}
	return matched
}

// view 打印指定组件的全部源码(依赖闭包内的共享文件去重)。
func view(names []string, jsonMode bool) {
	src, reg := loadRegistry()
	if !viewFiles(src, reg, resolveNames(reg, names), "", jsonMode) {
		fatal("没有匹配的文件: %s", strings.Join(names, ", "))
	}
}

// infoCommand 打印本机环境与项目状态(不拉 registry, 离线可用)。整段都是命令结果。
func infoCommand(uiDir string, jsonMode bool) {
	ver := displayVersion()
	wd, _ := os.Getwd()
	src := strings.TrimSpace(os.Getenv("XUI_REGISTRY"))
	if src == "" {
		src = "默认(远程 main 分支)"
	}
	// 升级提示的依据是数据源声明的 cliChange, 那要联网才拿得到 —— info 保持离线, 所以这里只报
	// "提示被环境变量关了"这条本地就知道的状态。
	disabled := os.Getenv(updateCheckDisableEnv) != ""
	// 组件目录里有多少 .go: 目录读不到 = 还没 init。
	n := 0
	initialized := false
	if entries, err := os.ReadDir(uiDir); err == nil {
		initialized = true
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				n++
			}
		}
	}
	if jsonMode {
		printJSON(infoJSON{
			CLI: ver, CWD: wd, Go: runtime.Version(),
			ComponentsDir: uiDir, ComponentsInitialized: initialized, GoFiles: n,
			Source: src, NoticeDisabled: disabled,
		})
		return
	}
	out("xui: %s", ver)
	out("工作目录: %s", wd)
	out("Go: %s", runtime.Version())
	if initialized {
		out("%s: %d 个 Go 文件", uiDir, n)
	} else {
		out("%s: 未初始化", uiDir)
	}
	out("数据源: %s", src)
	if disabled {
		out("升级提示: 已由 %s=1 关闭", updateCheckDisableEnv)
	}
}

// diffFiles 把本地文件与 registry 内容做逐行 diff(不写本地); filter 非空时只比匹配的
// 文件(组件名或文件基名, 见 filterHit); json 出结构化。
// "一个都没匹配上"与"匹配上了但确实没差异"是两件事, 必须分开说: 都答"没有差异"会让用户
// 以为已装的组件是最新的。前者返回 false, 由调用方按失败处理(过滤值写错 = 命令没达成目标)。
func diffFiles(src *source, reg *registry, order []string, uiDir, filter string, jsonMode bool) bool {
	inst := loadManifest(uiDir)
	done := map[string]bool{}
	matched := false
	differs := false
	var files []diffFileJSON
	for _, name := range order {
		for _, f := range reg.Components[name].Files {
			if done[f] {
				continue
			}
			done[f] = true
			if !filterHit(name, f, filter) {
				continue
			}
			matched = true
			remote := string(readComponent(src, f))
			dst := filepath.Join(uiDir, f)
			local, err := os.ReadFile(dst)
			if err != nil && !os.IsNotExist(err) {
				fatal("读取 %s 失败: %v", dst, err)
			}
			if err == nil && string(local) == remote {
				continue
			}
			differs = true
			// 给差异标出来源, 否则光看 diff 分不清"是我改的"还是"版本升级" ——
			// 这正是 update 要不要 -o 的分界。没有基线时只能说"判断不了", 别断言用户改过。
			baseline, hasBaseline := inst.Files[f]
			note, state := "", ""
			switch {
			case os.IsNotExist(err):
				// 说清"谁来创建": 整个组件没装过时 update 只提示, 不会替你装(见 dropUninstalled)。
				if anyFileLocal(uiDir, reg.Components[name].Files) {
					note = " (本地没有, update 会补上)"
				} else {
					note = " (本地没有, 要装它请用 xui add " + name + ")"
				}
				state = "missing"
			case hasBaseline && baseline == sha256Hex(local):
				note, state = " (本地未改动, 纯版本差异 —— update 会直接替换)", "upgrade"
			case !hasBaseline:
				note, state = " (没有该文件的安装清单基线, 无法判断来源 —— update 默认跳过, 需 -o 覆盖)", "unknown"
			default:
				note, state = " (本地已修改 —— update 会跳过, 需 -o 覆盖)", "edited"
			}
			text := unifiedDiff(dst, "components/"+f, string(local), remote)
			if jsonMode {
				files = append(files, diffFileJSON{Path: dst, File: f, State: state, Diff: text})
				continue
			}
			out("\n=== %s%s ===", dst, note)
			outRaw(text)
		}
	}
	if jsonMode {
		if files == nil {
			files = []diffFileJSON{}
		}
		printJSON(diffJSON{Dir: uiDir, Filter: filter, Matched: matched, Files: files})
		return matched
	}
	if matched && !differs {
		out("没有差异")
	}
	return matched
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
	// 空文件的 hunk 从第 0 行起(标准写法): 本地没有这个文件时是 `@@ -0,0 +1,N @@`,
	// 写 `-1,0` 虽然多数工具能读, 但 patch 会把它当成"删了第 1 行"那种边界情况。
	startA, startB := 1, 1
	if len(a) == 0 {
		startA = 0
	}
	if len(b) == 0 {
		startB = 0
	}
	fmt.Fprintf(&out, "--- %s\n+++ %s\n@@ -%d,%d +%d,%d @@\n", localName, remoteName, startA, len(a), startB, len(b))
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

// list 列出/搜索组件; -l/--limit 与 -o/--offset 按过滤后的顺序切片; -j 出结构化。
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
	if opts.json {
		items := make([]componentJSON, 0, end-start)
		for _, name := range matched[start:end] {
			e := reg.Components[name]
			// 空的一律给 [] 而不是 null, 调用方不必两种都防。
			files, deps := e.Files, e.Deps
			if files == nil {
				files = []string{}
			}
			if deps == nil {
				deps = []string{}
			}
			items = append(items, componentJSON{Name: name, Files: files, Deps: deps})
		}
		printJSON(listJSON{
			Source: src.desc(), CLI: displayVersion(), Runtime: reg.Runtime, CLIChange: reg.CLIChange,
			Query: opts.query, Offset: opts.offset, Limit: opts.limit,
			Total: len(matched), Shown: end - start, Components: items,
		})
		return
	}
	info("可用组件 (数据源: %s):", src.desc())
	if start == end {
		if len(matched) == 0 {
			out("  (没有匹配的组件)")
		} else {
			out("  (没有更多组件)")
		}
	}
	for _, name := range matched[start:end] {
		e := reg.Components[name]
		extra := ""
		if len(e.Deps) > 0 {
			extra = "  (依赖: " + strings.Join(e.Deps, ", ") + ")"
		}
		out("  %-12s %s%s", name, strings.Join(e.Files, ", "), extra)
	}
	if opts.query != "" || opts.limit > 0 || opts.offset > 0 {
		out("\n共 %d 个匹配, 显示 %d 个", len(matched), end-start)
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

// uiDirToken 是下发模板里代表"组件目录"的占位符。用显式 token 而不是再加一个 %s:
// 这些模板已经有 %s 在填模块路径/项目名, 再加动词会让 fmt.Sprintf 的参数顺序一眼看不出来,
// 填错一位就是一份能编译但说假话的 README。
const uiDirToken = "{{UIDIR}}"

// renderUI 把模板里的占位符换成真正的组件目录名(目录名可以是 components/ 以外的任何合法名)。
func renderUI(body, uiSub string) string {
	return strings.ReplaceAll(body, uiDirToken, filepath.ToSlash(uiSub))
}

// mainGoTemplate 生成的项目入口: AppWindow + 一张示例卡片(只用 init 装好的基础组件)。
const mainGoTemplate = `package main

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "%s/{{UIDIR}}"
)

func main() {
	app.InitOrExit()
	a := app.New(true) // xui 固定使用 Direct2D 渲染
	a.EnableAutoDPI(true).EnableDPI(true)
	a.SetPaintFrequency(10) // 重绘节流 10ms, hover 过渡约 60fps

	w := ui.NewAppWindow(%s, 960, 640, ui.WithAppWindowMinSize(640, 480))
	setWindowIcons(w)
	content := w.Content()
	content.SetPadding(24, 24, 24, 24)
	content.SetSpace(16)

	_, stack := newCard(content.Handle, 12)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "欢迎使用 xui", stack.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "{{UIDIR}}/ 目录是组件源码, 归你所有, 可随意修改。", stack.Handle))
	xui.ItemFill(ui.NewInput(0, 0, 0, 0, stack.Handle, ui.WithPlaceholder("输入点什么...")))
	xui.ItemFill(ui.NewCheckbox(0, 0, 0, 0, "记住我", stack.Handle))

	row := xui.HStack(stack.Handle, 36, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "开始使用", row.Handle), 104)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "了解更多", row.Handle, ui.WithVariant(ui.ButtonOutline)), 104)

	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "打开 {{UIDIR}}/theme.go 改主题, 或看 {{UIDIR}}/DESIGN.md 了解组件。", content.Handle).SetMuted())

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

// setWindowIcons 从 exe 的 APP 图标资源设置窗口图标(资源见 winres/, 由 xc res make 编成 .syso):
// 大图标管任务栏 + Alt+Tab, 小图标(24x24)管任务栏预览窗口左上角, 也影响任务管理器。
// LR_SHARED 的 HICON 归系统共享管理, 不要 DestroyIcon。
func setWindowIcons(w *ui.AppWindow) {
	hMod := wapi.GetModuleHandleW("")
	if hIcon := wapi.LoadImageW(hMod, common.StrPtr("APP"), wapi.IMAGE_ICON, 0, 0, wapi.LR_SHARED|wapi.LR_DEFAULTSIZE); hIcon != 0 {
		w.SetBigIcon(hIcon)
	}
	if hIcon := wapi.LoadImageW(hMod, common.StrPtr("APP"), wapi.IMAGE_ICON, 24, 24, wapi.LR_SHARED); hIcon != 0 {
		w.SetSmallIcon(hIcon)
	}
}
`

// agentsRules 是两份下发的 AGENTS.md(项目根 / 组件目录)共用的改组件规则。
// 用 ~~~ 而不是 ``` 围栏: Go 原始字符串不能包含反引号。
const agentsRules = `## 修改/自定义组件

- 先读 DESIGN.md: 交互语义看第四节末的汇总表;第九节是**本库依赖的 xcgui API 清单**, 不是
  组件选项表 —— 每个组件的选项与方法看本目录的源码注释(或 pkg.go.dev)。
- xcgui 的坑(现象/根因/修法)在同目录的 DESIGN-PITFALLS.md(第八节, 按编号);DESIGN.md 第八节
  只有编号索引。源码注释里的 "见第八节 N" 按编号去那份查。
- 颜色、圆角、字体一律从主题令牌取(xui.Current()), 禁止硬编码颜色; 不要缓存颜色,
  切换主题用 xui.OnThemeChangeFor(元素句柄, fn) 订阅重绘。
- 事件一律用 AddEvent* 系列; hover 过渡用 xui.Hover / xui.Fade,
  弹层进出场用 xui.Presence, 不要自己起定时器或缓存过渡值。
- 布局禁止绝对坐标: Fill = 父级整尺寸(不是剩余空间), Weight = 剩余空间(仅父级尺寸确定时
  有效), Auto = 随内容自撑; 页面/长内容用 ui.ScrollArea(见 DESIGN 8.55/8.56/8.66)。
- 组件源码里的 "见 DESIGN 第 N 节" 指 DESIGN.md, "见第八节 N 条" 指 DESIGN-PITFALLS.md;
  章节与条目编号稳定, 只增不改。
`

// agentsProjectTemplate 是 `xui init -n` 生成到项目根的 AGENTS.md。
// README 归项目作者、可能被改写, 所以运行方式与组件用法要在这里留一份给 AI 助手。
const agentsProjectTemplate = `# AGENTS.md — 给 AI 助手的项目说明

本项目界面使用 xui(shadcn 风格的 xcgui 自绘组件库)。组件源码已经拷贝进 {{UIDIR}}/(包名 ui),
归项目所有, 可以随意修改。README.md 由项目作者维护、可能被改写; 用法与约定以本文件
和 {{UIDIR}}/DESIGN.md、{{UIDIR}}/DESIGN-PITFALLS.md 为准。

## 运行

~~~bash
go run .
~~~

- 仅支持 Windows(xcgui + Direct2D)。
- 入口在 main.go: app.InitOrExit() → app.New(true) → 建窗口 → a.Run() → a.Exit()。
- a.SetPaintFrequency(10) 不能省: 默认 30ms 重绘节流会让悬停与动画发涩。

## 目录

- main.go — 应用入口与示例
- {{UIDIR}}/ — 组件源码(package ui, 扁平单包; 跨文件助手也在这里)
- {{UIDIR}}/theme.go — 主题令牌, 改这里全局换肤
- {{UIDIR}}/DESIGN.md — 说明书: 架构、主题、组件解剖、依赖的 xcgui API 清单(第九节)
- {{UIDIR}}/DESIGN-PITFALLS.md — xcgui 踩坑全集(现象/根因/修法, 按编号; DESIGN.md 第八节是索引)
- winres/ — exe 图标与版本信息(icon.ico + winres.json), 由 xc 生成 .syso

## 图标与版本信息

winres/ 里的图标与版本资源已编进程序。换图标替换 winres/icon.ico, 改版本信息编辑
winres/winres.json, 然后重新生成 syso:

~~~bash
xc res make --arch amd64,386
~~~

- 没装 xc 先 go install github.com/twgh/xc@latest。
- 只生成 386/amd64: xcgui 不支持 ARM。
- 图标资源名是 APP; 窗口与托盘图标都从这里取(HICON 是 LR_SHARED, 不要 DestroyIcon):

~~~go
hMod := wapi.GetModuleHandleW("")
hBig := wapi.LoadImageW(hMod, common.StrPtr("APP"), wapi.IMAGE_ICON, 0, 0, wapi.LR_SHARED|wapi.LR_DEFAULTSIZE)
hSmall := wapi.LoadImageW(hMod, common.StrPtr("APP"), wapi.IMAGE_ICON, 24, 24, wapi.LR_SHARED)
w.SetBigIcon(hBig)     // 任务栏图标 + Alt+Tab
w.SetSmallIcon(hSmall) // 任务栏预览窗口左上角, 也影响任务管理器
~~~

生成的 main.go 已用 setWindowIcons 设好这两处(换图标只需替换 winres/icon.ico 再重跑
xc res make)。ui.AppWindow 内嵌 *window.Window, 需要时可直接调用 SetBigIcon/SetSmallIcon;
只设托盘不会同步任务栏与预览窗。托盘组件不在默认安装里, 需要时先 xui add tray, 再把
同一个 hBig 传给 ui.WithTrayIconHandle。

## 添加与更新组件

CLI 装一次即可(没装就把 xui 换成 go run github.com/twgh/xui/cmd/xui@latest); 在项目根目录执行
(组件名用 xui list 查):

~~~bash
go install github.com/twgh/xui/cmd/xui@latest
xui add table tree   # 自动带上依赖组件
xui update           # 更新 {{UIDIR}}/ 里已装组件
xui list             # 列出可用组件
~~~

CLI 逐个比对内容: 本地没有则创建、一致则跳过、被改过则警告跳过; 确认丢弃本地改动时加
-o/--overwrite。本项目的组件目录是 {{UIDIR}}/(相对项目根), 下面所有命令都按它写; 目录名在
xui init -n <项目名> -p <目录> 这一步就定下来了, 装好以后每条命令用 -p 指过去。
装完组件, CLI 会把 go.mod 里的 github.com/twgh/xui 抬到与这批组件配套的运行时版本
(组件源码跟仓库最新, 运行时按 tag 取), 并跑一次 go mod tidy 清掉 go.sum 里的旧版本条目;
离线或 --no-install 时只打印那两条命令, 不动你的 go.mod。

## 最小用法

~~~go
import (
    "github.com/twgh/xcgui/app"
    "github.com/twgh/xui"
    ui "%s/{{UIDIR}}"
)

func main() {
    app.InitOrExit()
    a := app.New(true) // 固定 Direct2D
    a.EnableAutoDPI(true).EnableDPI(true)
    a.SetPaintFrequency(10)

    w := ui.NewAppWindow("标题", 960, 640)
    content := w.Content()
    content.SetPadding(24, 24, 24, 24)
    content.SetSpace(16)
    xui.ItemFill(ui.NewButton(0, 0, 0, 0, "确定", content.Handle))

    w.Show(true)
    a.Run()
    a.Exit()
}
~~~

组件的构造函数参数顺序是 x, y, cx, cy, parent(cx/cy 传 0 表示交给布局), 之后是选项
(ui.WithXxx); 布局用 xui.ItemFill / xui.ItemWidth / xui.HStack / xui.VStack。

` + agentsRules

// agentsDirTemplate 是普通 `xui init [目录]` 生成到组件目录的 AGENTS.md。
// 目录名可能不叫 ui, 所以全文用"本目录"表述, 不写死路径。
const agentsDirTemplate = `# AGENTS.md — 给 AI 助手的目录说明

本目录是 xui(shadcn 风格的 xcgui 自绘组件库)的组件源码拷贝(package ui), 归项目所有,
可以随意修改。README.md 由项目作者维护、可能被改写; 组件用法与修改规则以本文件为准。

- DESIGN.md — 说明书: 架构、主题系统、组件解剖、依赖的 xcgui API 清单(第九节)
- DESIGN-PITFALLS.md — xcgui 踩坑全集(现象/根因/修法, 按编号; DESIGN.md 第八节只有索引)
- theme.go — 主题令牌, 改这里全局换肤
- 其余 *.go — 组件源码(扁平单包, 互相直接调用)

## 添加与更新组件

CLI 装一次即可(没装就把 xui 换成 go run github.com/twgh/xui/cmd/xui@latest); 在项目根目录执行,
本目录叫 {{UIDIR}}: 在项目根执行 xui 时给 add/update/info 带上 -p {{UIDIR}}(目录就叫 ui 时可省):

~~~bash
go install github.com/twgh/xui/cmd/xui@latest
xui add table tree -p <目录>
xui update -p <目录>
xui list
~~~

CLI 不会覆盖被改过的文件; 确认丢弃本地改动时加 -o/--overwrite。
装完组件, CLI 会把项目 go.mod 里的 github.com/twgh/xui 抬到与这批组件配套的运行时版本
(组件源码跟仓库最新、运行时按 tag 取), 并跑一次 go mod tidy 清掉 go.sum 里的旧版本条目;
离线时只打印那两条命令, 不动 go.mod。

## 组件速用

~~~go
import ui "<模块路径>/<本目录>"

xui.ItemFill(ui.NewButton(0, 0, 0, 0, "确定", parent.Handle))
xui.ItemWidth(ui.NewInput(0, 0, 0, 0, parent.Handle, ui.WithPlaceholder("搜索")), 240)
~~~

所有组件的构造函数都是 x, y, cx, cy, parent(cx/cy 传 0 交给布局), 之后是选项(ui.WithXxx)。

` + agentsRules

const readmeTemplate = `# %s

由 xui 生成的项目骨架: Go + xcgui + xui(shadcn 风格自绘组件)。

## 运行

~~~bash
go run .
~~~

生成项目时已自动执行 go mod tidy; 如果初始化时用了 --no-install, 先跑一次 go mod tidy。

## 目录

- main.go — 应用入口(AppWindow + 示例卡片)
- AGENTS.md — 给 AI 助手的使用说明与修改规则(README 改写后仍然有效)
- {{UIDIR}}/ — 组件源码拷贝, 归你所有, 可随意修改
- {{UIDIR}}/theme.go — 主题令牌, 改这里全局换肤
- {{UIDIR}}/DESIGN.md — 组件自定义指南
- {{UIDIR}}/DESIGN-PITFALLS.md — xcgui 踩坑全集(现象/根因/修法, 按编号)
- winres/ — exe 图标与版本信息(已生成 syso, 开箱即用)

## 图标与版本信息

winres/icon.ico 是程序图标, winres/winres.json 是资源完整配置(RT_GROUP_ICON /
RT_MANIFEST / RT_VERSION)。改完重新生成:

~~~bash
xc res make --arch amd64,386
~~~

没装 xc 时先 go install github.com/twgh/xc@latest。只生成 386/amd64 —— xcgui 不支持 ARM。
图标资源名是 APP; 窗口与托盘图标都从这里取(HICON 是 LR_SHARED, 不要 DestroyIcon):

~~~go
hMod := wapi.GetModuleHandleW("")
hBig := wapi.LoadImageW(hMod, common.StrPtr("APP"), wapi.IMAGE_ICON, 0, 0, wapi.LR_SHARED|wapi.LR_DEFAULTSIZE)
hSmall := wapi.LoadImageW(hMod, common.StrPtr("APP"), wapi.IMAGE_ICON, 24, 24, wapi.LR_SHARED)
w.SetBigIcon(hBig)     // 任务栏图标 + Alt+Tab
w.SetSmallIcon(hSmall) // 任务栏预览窗口左上角, 也影响任务管理器
~~~

生成的 main.go 已用 setWindowIcons 设好窗口图标, 换图标只需替换 winres/icon.ico 再重跑
xc res make。ui.AppWindow 内嵌 *window.Window, 需要时可直接调用 SetBigIcon/SetSmallIcon;
只设托盘不会同步任务栏与预览窗。需要托盘就先 xui add tray, 再把同一个 hBig 传给
ui.WithTrayIconHandle。

## 添加组件

CLI 装一次即可(没装就把 xui 换成 go run github.com/twgh/xui/cmd/xui@latest):

~~~bash
go install github.com/twgh/xui/cmd/xui@latest
xui add table tree
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
