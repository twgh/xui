# HANDOFF — 交接文档

> **本文只回答"现在到哪了"**:当前状态、待办、命令、仓库结构、验证工作流、去哪查。
> 设计原理 / xcgui 踩坑细节 / 交互语义**一律不在这里重复** —— 见 **DESIGN.md**(第八节 = 踩坑索引;第四节末 = 语义权威汇总表)与 **DESIGN-PITFALLS.md**(第八节完整条目)。agent 会话硬规则见 **AGENTS.md**,面向用户的用法见 **README.md**。
> ⚠️ **DESIGN.md + DESIGN-PITFALLS.md 会随 `xui init` 一起拷进用户项目的组件目录(默认 `ui/`, 目录名可改)**, 按使用者视角写:不写日期、报障时间线、待办、发布流程、仓库结构 —— 日期与报障时间线放 **CHANGELOG.md**, 其余放本文。

## 项目一句话

`github.com/twgh/xui`:为 xcgui(炫彩界面库)提供 **shadcn-ui 风格**的自绘组件——主题令牌、变体系统、源码拷贝分发(copy-not-install)。交互语义以本地 shadcn 参考实现(`D:\GOProject\src\ui\shadcn`)的源码为唯一权威。

## 当前状态(截至 2026-11-07)

- **阶段**: P0 运行时 + P1/P2/P3 组件已完成并视觉验收; P4(文档站、主题画廊)未开始。
- **发布**: v0.11.0(2026-11-07, GitHub Release 已建)。组件跟 `main` 下发, 运行时按 tag 取。
- **未推送的破坏性改动(两项, 同一次推送)**: ① registry 键名交换 —— `combobox` 现在指自动补全(= shadcn 文档的 Combobox), 原生载体改叫 `native-combobox`, `native-select.deps` 跟着改; ② Go 类型 `ComboBox` 一族改名 `NativeComboBox`(`NewNativeComboBox` / `NativeComboBoxSize` / `NativeComboBoxSizeDefault·Sm` / `WithNativeComboBoxPlaceholder·Size·Invalid`), 与 `NativeSelect` 同族, 顺带消掉 `WithCombo*` 与 `WithCombobox*` 只差一个 box 的三对撞名。⚠️ **数据源是 main, 推上去即刻生效, 不等 tag**: ①让 `xui add combobox` 装到另一个组件; ②更硬 —— 类型定义在拷给用户的组件源码里, `xui update` 一拉, 用户写的 `ui.NewComboBox(...)` **直接编译不过**(实测报 `undefined: ui.NewComboBox (but have NewCombobox)`, 告知里要写明"改成 `NewNativeComboBox`, 别顺着编译器提示改成 `NewCombobox`, 那是自动补全")。文件名未动, 老项目的安装/改动判定实测不受影响; 下游 sitless / myapp 应用代码零调用。配套 tag 发布时 `runtime` 照抬、`cliChange` 不抬(CLI 逻辑未改)。证据见 CHANGELOG。
- **CLI 版本语义(2026-11-06 改)**: 设计文档与组件同源(都从数据源取), 所以 tag 只代表"CLI 代码 + 内嵌模板/资源"的变化; 升级提示与 `--check` 的依据是 `registry.json` 的 `cliChange`, 不再是 `@latest` tag。后台探测子进程与 `update-check.json` 缓存已删除。
- **组件覆盖**: 70 个可安装名(`registry.json` / `xui list`, 本地与镜像一致)= 56 个 shadcn 官方组件(上游 Components 侧栏 66 个名字, 其中 65 个可安装)+ 14 个桌面扩展(color-picker / file-dialog / time-picker / tree / tray / window / screen-popup / screen-toast / radio / rating / steps / timeline / tag-input / status-bar); context-menu 在 `menu`, form 在 `field`。完整清单见 README / DESIGN 第七节, 上游缺口见下面待办。
- **DataTable 双模式(2026-11-07 探针定案)**: 分页是默认, 虚拟滚动用 `WithDataTableVirtualScroll` 显式开启 —— 两种模式共用排序/筛选/勾选/编辑/列显隐; 分页只装当前页(默认 10, 推荐 20~50), 虚拟滚动一次展示全部筛选结果且页脚只留总行数(分页 API 不生效). 原生 List 的滚动绘制只跟可见行数有关, 两种模式同为 ~4ms/帧(1M 行也一样); 把全部行装进普通 List 时 1M 行会多占约 1GB 原生内存, 分页/虚拟滚动都不复制. 真正随行数增长的是 Go 侧排序/筛选(1M 行 sortID 约 157ms、数字自然比较约 1.6s、筛选约 85ms), 与表格载体无关; 50 万行以上建议 `DataColumn.Compare` 或后台排序/筛选. demo 新增「数据2」页(10 万行虚拟滚动表, `XUI_HOME_PAGE=20`, **首次进入才生成数据**)与「数据」页的分页表对照; 依据与原始数据在 `.verify/probe-longlist/`(临时, 未入库).
- **最近一批**: 修复阴影窗口拖动改大小时整窗闪 1~2 帧透明(`NewAppWindow` 改在 `AddEvent_Size` 里同步重绘, 见 DESIGN 191); Tree 支持勾选模式(`WithTreeCheckable`: chevron 后自绘复选框, 父节点半选, 点父节点级联全部后代; 勾选与原生单选解耦, `Checked/SetChecked/CheckedKeys/OnCheckChange`; demo 数据页树已改勾选模式), 自动折行抽成 `Label.SetAutoWrap()`(数据页两条长说明改用); Steps 补齐状态/进度/可点/尺寸(四态 + `StepItem.Status/Disabled/Percent`、`WithStepsClickable/Size/Initial`、`SetItemPercent`); demo 导航2 拆为导航2(标签页/面包屑/分页)、导航3(步骤条/可折叠面板)、导航4(时间轴), 原导航菜单/侧边栏顺延为导航5/6(`XUI_HOME_PAGE` 当前 0..21); Timeline 支持竖向/水平、start/end/alternate 与节点色/图标/加载/空心样式、Rating 星级评分、Steps 自定义图标(`StepItem.Icon`)、FileDialog 文件对话框、Steps 步骤条; 文档体积优化(2026-11-04)—— HANDOFF 的历史记录拆到 CHANGELOG, DESIGN 第八节正文拆到 DESIGN-PITFALLS.md、DESIGN 只留索引; CLI 的 `init` / `add DESIGN.md` 一起下发配套文件, 老项目 `update` 会补装缺失的配套文件; 新增 `tag-input`(shadcn 无官方组件, 交互语义由本库定死并写进 DESIGN 第四节末): 自绘 chips 外壳 + 裸 Edit, 回车/分隔符成标签、退格删末个、点 × 删除、换行自撑, 亮暗 DPI 截图与逐组件安装编译均已验收; 新增 `status-bar`(窗口底部状态栏: 左中右槽 + 尺寸抓手/读数 + 悬停提示, 自定语义已写进 DESIGN 第四节末), demo 基础4 加卡并在窗口底部挂真实状态栏。
- **历史记录**: 按时间记录的完成 / 修复条目(含现象、根因、验收)全部在 [CHANGELOG.md](CHANGELOG.md)。

## 已知限制(不是待办, 是 xcgui 的能力边界)


- ℹ️ Input 的文字/光标/选区/IME 由原生 Edit 绘制(只开放字体/字色/占位符色/插入符色/内边距, 见 DESIGN 三 第 4 条 / 23 / 61)。
- 完整清单见 DESIGN 第七节。


## 待办

- ⬜ **上游新增的消息/聊天组(推荐的下一步)**: 缺口数字见上面「组件覆盖」。本地参考实现没装这批 tsx, 上游源码可从 registry JSON 取; 语义对照后补进 DESIGN 第四节末的表, 每条走 kebab-case key + 逐组件安装编译回归。
  - 先做 **Marker / Message / Bubble / Trace / Attachment**: 都是 `registry:ui`, npm 侧只依赖 `cn` / radix Slot, registry 依赖(Trace → collapsible、Attachment → button)已有 —— 零新依赖类, 纯布局 + 自绘; 一批做完加一页 demo 一次验收。
  - 之后 **Questionnaire**(分步状态机, 上游靠 `@shadcn/react` 无头原语)、**Message Scroller**(滚动锚定 + 只在底部跟随; 上游明确不含虚拟化, 可用原生 List 虚拟滚动替代)。
  - **Code Block** 只做降级版或不做: 全量对齐要 Go 侧 tokenizer(上游 Shiki 13 语法 + ANSI + Streamdown + 选区感知复制), 做了必须在 DESIGN 第七节声明部分对齐。**Direction** 不做(文字渲染只有 LTR)、**Typography** 不做(上游是纯样式文档页, 无可安装组件)。
  - 侧栏 **AI Elements** 21 个(Chain of Thought / Reasoning / Tool 等)是 Vercel 自己的项目(elements.ai-sdk.dev), 不属 shadcn registry, **不在覆盖目标内**。
- ⬜ **P4(发布后, 非阻塞)**: 文档站、主题画廊(把 shadcn 社区主题批量转译成 Go 令牌)。
- ⬜ **可选(不阻塞)**: 补齐本地 shadcn 参考实现中旧批次组件的 tsx 与 App.tsx 对照卡(alert / alert-dialog / aspect-ratio / breadcrumb / button-group / calendar / chart / context-menu / empty / field / item / sonner / toggle / toggle-group); 对应 Go 组件均已实现并验收, 仅用于逐像素视觉对照。calendar / chart / sonner 需要额外 npm 依赖。Data Table / Combobox 是组合模式, 无独立 tsx。

## 仓库结构

```
D:\GOProject\src\ui\xui\          ← 本仓库(模块 github.com/twgh/xui)
├── *.go                          运行时: color / theme / themes_builtin / manager
│                                 draw(Painter) / icons(图标几何+D2D 直通出图) / focus / font / state
│                                 anim(hover 过渡) / presence(进出场) / container·layout(布局骨架)
├── components\                   组件源码 = registry 内容(扁平单包 package ui)
│                                 button checkbox radio radiogroup input label separator(含 Badge) card alert
│                                 switch progress slider tabs collapsible tooltip toast window combobox dialog alertdialog
│                                 menu table datatable tree pagination scrollarea
│                                 calendar datepicker timepicker(弹层骨架 popup.go)
│                                 empty field item breadcrumb kbd skeleton spinner toggle
│                                 native_select select inputotp accordion
                                 popover hovercard avatar command
                                 sheet drawer menubar navigationmenu carousel resizable
                                 sidebar screentoast tray statusbar
├── icons\                        图标资源子包: lucide 预处理数据(2130 个) + 每图标一个
│                                 IconXxx 常量 + Path/Has/Names/Count/Version
├── scripts\gen_icons.py          lucide-static SVG → icons\lucide_gen.go(名字表)
│                                 + icons\lucide_icons_*.go(图标常量, 每 400 个一个文件)
├── cmd\xui\                      CLI(init / add / add --all / update / list; 数据源默认
│                                 远程 main 分支, XUI_REGISTRY 可覆盖, 失败降级内嵌)
│                                 assets\app.ico = init -n 生成项目用的默认图标
│                                 winres.go = 生成 winres/ + `xc res make` (缺 xc 自动 go install)
├── icons_test.go                 解析器语法 + 资源完整性测试(go test ./...)
├── demo\                         演示应用(也是全组件编译测试目标)
│                                 winres\ = 图标资源(winres.json + icon.ico) + xc res 生成的 rsrc_windows_*.syso
├── registry.json                 组件清单(files + deps) + 两个版本声明: runtime(配套运行时) /
│                                 cliChange(CLI 最后改动, 升级提示的唯一依据)
├── registry.schema.json          registry.json 的 JSON Schema(编辑器补全/校验)
├── .github\workflows\ci.yml      CI: vet/test/build(amd64+386) + 逐组件安装编译(Windows)
├── docs\DESIGN.md                知识库: 架构设计 + xcgui 踩坑索引(第八节)
├── docs\DESIGN-PITFALLS.md       xcgui 踩坑全集(第八节完整条目, DESIGN 的配套文件)
├── docs\HANDOFF.md               本文
├── docs\CHANGELOG.md             历史记录(按时间; 现象 / 根因 / 修法 / 验收)
├── README.md                     用法
└── AGENTS.md                     agent 会话自动加载的入口规则

D:\GOProject\src\ui\shadcn\       ← 本地部署的 shadcn-ui 参考实现(语义权威)
├── src\components\ui\*.tsx       组件语义的唯一来源(hover/focus-visible/disabled 等)
├── src\index.css                 主题令牌(:root/.dark, nova 预设 neutral 色板)
└── src\App.tsx                   与 demo 同构的对照页
```

## 常用命令

```bash
# xui demo(全部组件的编译+视觉验证目标)
cd D:\GOProject\src\ui\xui
go build -ldflags="-s -w -H windowsgui" -trimpath -o demo/demo.exe ./demo && demo\demo.exe

# 改 demo 图标后重新生成 exe 资源(需要 xc 命令行工具; 没有就 go install github.com/twgh/xc@latest)
# 只出 386/amd64: xcgui 不支持 ARM
cd demo && xc res make --arch amd64,386

# shadcn 参考实现 dev server(可能已在运行: netstat -ano | findstr 5173)
cd D:\GOProject\src\ui\shadcn && pnpm dev    # http://localhost:5173

# CLI 本地测试(默认源:直连→Gitee 镜像→GitCode 镜像→代理, 全不通才降级内嵌源码)
go run ./cmd/xui list / init <目录> / add <组件...> / add --all / update [组件...]
go run ./cmd/xui self-update --check   # 看 @latest 解析到哪个 tag + CLI 最后改动的 tag(不安装)
#   注意: 本机 PATH 首选 D:\GOProject\src\ui\xui\xui.exe, GOBIN 是 D:\GOProject\bin\xui.exe
#         self-update 升的是 GOBIN 那份, 会警告 PATH 里另有一份
go test ./cmd/xui    # 候选顺序 / 数据源解析 / httptest 端到端 / 显式源失败退出
# 想验远程链路(无需 GitHub): 仓库根起 python -m http.server 8123,
# 再 XUI_REGISTRY=http://127.0.0.1:8123 go run ./cmd/xui add dialog
# 数据源也可指本地目录(完全离线): XUI_REGISTRY=D:\GOProject\src\ui\xui
# 依赖回归(每加组件/跨文件助手必跑): 逐组件单独装到临时项目再 go build ./ui(步骤见下面「依赖回归」小节)

# 暗色对照验收: 环境变量决定启动主题, 不用改代码
#     XUI_THEME=dark  启动 demo 即暗色(cmd: set XUI_THEME=dark && demo\demo.exe)
#     运行中仍可用右上角按钮切换(参考页 shadcn 同样支持按 d 键切换)
# 直接启动到指定页签(验收某一页时省得点): XUI_HOME_PAGE=0..21 = 基础1/基础2/基础3/基础4/表单1/表单2/表单3/表单4/表单5/导航1/导航2/导航3/导航4/导航5/导航6/反馈1/反馈2/反馈3/图表/数据/数据2/图标
#     XUI_ANIM_SCALE=5 把 hover / 弹层 / Collapsible 的过渡慢放 5 倍(抓中间帧用)

# 全部测试(图标解析器语法 + 资源完整性 + 缓动曲线 + CLI 选路/数据源/端到端)
go test ./...

# CI(.github/workflows/ci.yml): windows-latest 跑 vet/test/build(amd64+386)
# 与逐组件安装编译; 视觉/DPI/交互验收仍需本地人工
```

### 依赖回归(逐组件安装 + 编译)

**每加组件/跨文件助手就跑一遍「逐组件单独安装 + 编译」**——这不是发布前检查, 而是每次改动后的固定流程。临时项目**必须先有 go.mod**, 否则 `go run` 起不来。

```bash
# 1. 构建本地 CLI, 并让临时项目指向本仓库(离线验收; 联网时可改用 go mod tidy)
go build -o "$TEMP/xui-cli.exe" ./cmd/xui
rm -rf "$TEMP/xui-check" && mkdir -p "$TEMP/xui-check" && cd "$TEMP/xui-check"
go mod init check
go mod edit -require=github.com/twgh/xui@v0.0.0
go mod edit -replace=github.com/twgh/xui=D:/GOProject/src/ui/xui
go mod edit -require=github.com/twgh/xcgui@v1.4.3
go mod edit -replace=github.com/twgh/xcgui=D:/GOProject/src/github.com/twgh/xcgui

# 2. 单组件闭包 + 全量(改 registry 时两者都跑)
XUI_REGISTRY=D:/GOProject/src/ui/xui "$TEMP/xui-cli.exe" add alert -p ui
go build ./ui
rm -rf ui
XUI_REGISTRY=D:/GOProject/src/ui/xui "$TEMP/xui-cli.exe" add --all -p ui
go build ./ui
```

- 本地 xcgui 工作副本路径按本机调整; 没接工作副本时删掉 xcgui 的 replace, 走模块代理。
- 重跑前清空 `ui/`(或换临时目录), 避免上次拷贝的文件掩盖依赖漏登记(如跨文件助手)。
- ⚠️ **上面这套 `replace` 到工作区, 只验"registry 登记完整", 验不出运行时下限** —— 工作区的运行时永远是最新的。改运行时导出符号 / 改 `registry.runtime` 时, 再跑一遍**不带 replace** 的版本(CI 用的就是这个):

```bash
rm -rf "$TEMP/xui-floor" && mkdir -p "$TEMP/xui-floor" && cd "$TEMP/xui-floor"
go mod init check
FLOOR=$(grep -m1 '"runtime"' D:/GOProject/src/ui/xui/registry.json | sed 's/.*: *"\(v[^"]*\)".*/\1/')
go mod edit "-require=github.com/twgh/xui@$FLOOR"   # 运行时按已发布 tag 取
XUI_REGISTRY=D:/GOProject/src/ui/xui "$TEMP/xui-cli.exe" add --all -p ui
go mod tidy && go build -mod=mod ./ui
```

### 发布流程(打 tag 之后同一批)

- **`registry.json` 的 `runtime` 抬成刚发的 tag** —— 它的语义是「与 main 上这批组件**配套**的运行时版本」, 不只是"组件编得过"的最低要求:落后就等于让用户少拿运行时的修复(过渡表泄漏这类只改运行时的修复, 组件不引用新符号就不会触发下限抬升, 但用户照样该拿到)。**只改 CLI、没动组件也照样抬**(v0.8.1 起)。
  - 顺序: 先打 tag 再改 `runtime`(写早了 `go get` 拉不到不存在的 tag, CI 直接红, 是刻意的)。
  - CI 的 `registry-closure` 用这个 tag 的运行时逐组件编译(不 replace 到工作区), 所以 `runtime` 落后 / 组件引用未发布符号都会红。
  - 抬完跑一次下限回归: `go mod edit -require=github.com/twgh/xui@<新 tag>` + `add --all` + `go build ./ui`(代理可能还没收录新 tag, 必要时 `GOPROXY=direct`)。
- **`registry.json` 的 `cliChange` 只在真的改了 CLI 时才抬成刚发的 tag** —— 它是升级提示的唯一依据(「CLI 最后一次改动在哪个 tag」)。**不算改动**: 只动组件、只动设计文档、机械抬 `fallbackXuiVersion` 或 `runtime`。顺序与 `runtime` 一样: 先打 tag, 同一批再改它(必须是个已发布的 `vX.Y.Z`, 否则 CLI 警告后当没有)。忘抬的后果只是少提示一次升级, 装组件与拿文档都不受影响 —— 不像 `runtime` 抬晚会编译红。
  - **还没打 tag 就留着不动, 哪怕 CLI 已经改完了**:`cliChange` 只参与版本比较(不像 `runtime` 会被喂给 `go get`), 写一个不存在的 tag 不会让命令报错, 但会让提示变成空头支票 —— 用户 `xui self-update` 只装得到 `@latest`(上一个真 tag), 永远达不到那个版本号, 于是每条命令都被唠叨且无法解脱。提示是"装的版本低于 cliChange"这种**状态**比较而不是事件: 等 tag 真发出来再抬, 所有旧版本用户照样在下一条命令收到提示, 一个都不漏 —— 晚抬零代价, 早抬有死循环。
- 只改组件源码或设计文档 → **不用打 tag**(两者都跟 `main` 下发, 文档同源见下条); 改 CLI 行为/内嵌模板/图标才需要 tag(`go install ...@latest` 只认 tag)。版本号: **组件/运行时/API 变更走 minor, 只有 CLI 变更走 patch**(v0.8.1 就是这么定的)。
- **设计文档随数据源下发**: `xui init` / `add` 写进用户项目的 `DESIGN.md` / `DESIGN-PITFALLS.md` 取自数据源的 `docs/`(与组件同一个基址), 不再是 CLI 自带的那份 —— 组件批次改了文档里的编号, 用户不升级 CLI 也拿得到配套说明。**镜像同步要带上 `docs/`**(缺了就退回 CLI 自带版本并在 stderr 提示, 用户会拿到旧文档)。
- 版本示例(`README` 里的 `@vX.Y.Z` 与 `XUI_REGISTRY=.../vX.Y.Z`)与 `fallbackXuiVersion` 随 tag 一起更新; notes 草稿写进 `.verify/` 给用户过目后才推送。
- **GitHub Release**: 本机 `gh` 不可用(token / TLS), 用 PowerShell `Invoke-RestMethod` POST/PATCH; name 与 body 一律用 `[IO.File]::ReadAllText(path, [Text.Encoding]::UTF8)` 读文件后传 —— PS 5.1 把无 BOM 的 `.ps1` 当 ANSI 读, 脚本里直接写中文会乱码; 正文也别用 `Get-Content -Raw` 交给 `ConvertTo-Json`(会被当成对象写进发布页)。token 从 `git credential fill` 取, 用完立刻删临时文件。

### CLI registry 维护注意

- **默认源顺序与单文件换源由 `cmd/xui/main_test.go` 固化**:直连 → Gitee → GitCode(v5 API, JSON+base64) → ghfast → gh-proxy;改选路先改测试。
- 显式 `XUI_REGISTRY` **一律原样使用**(不套镜像、失败直接报错);只有未设置且全部候选不通时才降级 `moduleRoot()` 内嵌源 —— 别动 `moduleRoot`,它是离线回退的根。
- `registry.json` 的 `files` 是 `components/` 下的**扁平文件名**;`theme.go` 永不进清单;用户的 ui 目录名是 `init`/`-p` 参数,不要硬编码 `ui`。
- `registry.json` 的 `$schema` 指向仓库根的 `registry.schema.json`(编辑器补全/校验用, CLI 不读);改条目格式时两者同步, 否则 raw 链接会 404。
- CLI 本体按 Go 模块版本锁(`go run ...@vX.Y.Z`),组件跟 `main`;`@latest` 可能被模块代理缓存拖旧,组件更新不受影响。
- **升级提示没有后台进程也没有缓存文件**(以前的 `__update-check` 子进程 + `UserCacheDir/xui/update-check.json` 已删): 依据 `cliChange` 随 `registry.json` 一起到手, 所以 `version` / `info` 这类不拉数据源的命令不提示。改这套逻辑前想清楚"提示的依据必须在本次已经拿到的数据里", 别再把网络塞回前台。
- 本地把数据源指到仓库根(`XUI_REGISTRY=D:\GOProject\src\ui\xui`)时文档也从这个目录的 `docs/` 读 —— 改完 DESIGN 立刻能在临时项目里验到, 不再需要重新构建 CLI。

## 关键设计决策(一句话版, 详见 DESIGN.md)

1. **copy-not-install**:组件源码由 CLI 拷进用户 `ui/` 目录,归用户;运行时(theme/icons/helpers)走 go.mod
2. **绘制时取令牌**:组件永不缓存颜色,paint 回调里 `xui.Current()` 现取 → 切主题 = 换指针 + Redraw
3. **行为/视觉分层**:xcgui 原生控件 = 无头行为层,`AddEvent_Paint` + `drawx` 只管视觉
4. **交互语义逐条对齐 shadcn 源码**:改交互前先在 DESIGN 第四节末的语义汇总表确认,再读 `shadcn/src/components/ui/*.tsx`
5. **组件文件写薄**:绘制细节下沉运行时助手,降低拷贝模式下的升级冲突面
6. **界面用流式布局表达,不写绝对坐标**:窗口是唯一"天然确定尺寸"的父级,竖向主链从窗口开始一路 `Weight`(每级确定尺寸,下一级的 `Weight` 才算得出剩余空间);卡片高度用 `Auto` 自撑,要长高的面板整条链全 `Weight`。**别在中间垫 `Fill` 容器,也别用 `Fill` 当"剩余空间"**(尺寸语义见 DESIGN-PITFALLS.md 第八节 55,样板在 `demo/main.go` 的 `row`/`colStretch`/`autoCard`)。页面本身是 `ui.ScrollArea` 滚动容器: 自撑型页面(内容 Auto 高,行高也必须 Auto)窗口小了就滚, 铺满型页面(数据页 `SetContentFill`)整链 Weight、面板跟随窗口长高(见第八节 66)

## 文档导航(去哪查什么)

| 想了解 | 去哪 |
|---|---|
| 当前进度 / 待办 / 命令 / 仓库结构 / 验证工作流 | **本文** |
| 按时间记录的完成 / 修复条目(含报障现象、根因、验收) | [CHANGELOG.md](CHANGELOG.md) |
| 架构原理、分层边界、主题令牌、组件解剖 | DESIGN.md 一~四节 |
| **交互语义(改交互前必看)** | DESIGN.md 第四节末「与 shadcn 对齐的语义(权威汇总)」+ 本地 shadcn 源码 |
| **我们与 shadcn 的差异 / 变体尺寸映射** | DESIGN.md 第四节末「变体/尺寸与上游的映射、已知差异」 |
| 某个 xcgui 行为 / 为什么这么写(踩坑) | DESIGN-PITFALLS.md 第八节(按编号; DESIGN.md 第八节只列索引) |
| 界面布局怎么写(容器/尺寸/随窗口自适应/滚动) | DESIGN-PITFALLS.md 第八节 55(尺寸语义) + 56(Card) + 66(滚动容器), 样板在 `demo/main.go` |
| 用到的 xcgui API 清单 | DESIGN.md 第九节 |
| agent 会话硬规则 | AGENTS.md |
| 面向用户的安装与用法 | README.md |

## 视觉验证工作流(本项目惯例)

按模型能力分两条路径,像素级事实(颜色值/位置/尺寸/占比)在两条路径里都**用脚本测量、不靠目测**——"看起来的颜色"经常是错的,本项目多次靠像素测量纠偏:

**主模型有视觉**:

1. 构建 + 运行 demo → computer-use 截图(返回图片直接看)
2. `crop`(vision-skills 工具)或 computer-use zoom 放大细部目检
3. 精确数值用像素扫描脚本 `python scripts/pixel_scan.py <png> v|h|box|mix ...` 测量(如圆盘/白点直径占比、hover 前后色值)

**无视觉能力的模型**:

- vision CLI 在 `C:\Users\twgh\AppData\Local\agent-vision-toolkit\bin`:glance 看图/ground 定位/crop 放大
- 或主模型自己看完,把坐标与结论转述给子智能体

**通用**:

- 疑难渲染问题写探针程序(`.verify/probeN` 独立 main 包, `go build -o .verify/probeN/probeN.exe ./.verify/probeN` 后运行;`.verify/` 已被 go 工具链忽略),完事删掉
- 布局类问题**用 `xc.XEle_GetRectLogic` 打印各元素矩形**判定, 比截图量像素快且准(探针里 `AdjustLayout()` 之后读;注意窗口必须 `Show` 过才有非零尺寸, 见 DESIGN-PITFALLS.md 8.55)
- 要按**物理/逻辑像素精确量测**时, 自己用 Win32 抓窗口(GetWindowRect + CopyFromScreen)比 MCP 截图可靠: MCP 截图会被缩放到模型尺寸。**抓之前必须把抓图进程设为 per-monitor DPI-aware**(`SetProcessDpiAwarenessContext(-4)`), 否则 GetWindowRect 给虚拟化坐标、CopyFromScreen 抓物理像素, 两者差 150% —— 抓出来的图左边会多出一条桌面(踩过)
- 需要"真实鼠标"验证 hover/点击时, 探针里用 `wapi.SetCursorPos` 比外部脚本可靠;从 PowerShell 移动鼠标要注意它是 DPI-unaware 进程, 坐标会被虚拟化/偏移, 别拿截图坐标直接当屏幕坐标用(踩过坑)。脚本里出现中文时**必须存成 UTF-8 **带 BOM**** —— PowerShell 5.1 按 ANSI 读 `.ps1`, 无 BOM 的中文串会把语法读坏(报一堆莫名其妙的 "Unexpected token")
- ⚠️ **定位窗口一律用"枚举进程顶层窗口 + 取可见且面积最大"**, 不要用 `Process.MainWindowHandle`: 弹层/提示这类小窗(带 `WS_EX_TOOLWINDOW`)一旦创建, 它就会返回**小窗**, 之后用它算出来的坐标、抓出来的图全是错的(踩过: 整个复现流程因此偏了半天)。诊断脚本见 DESIGN 第九节的 Win32 诊断 API 一行
- **"窗口可见 ≠ 屏幕上有东西"**: 判断弹层/提示这类异型透明窗有没有真的显示, 不能只看 `IsWindowVisible`/矩形 —— 用 `PrintWindow` 抓它自己的内容(证明"画了")与 `CopyFromScreen` 抓同一块屏幕(证明"屏幕上有没有")对照, 再读 `GetLayeredWindowAttributes` 的 alpha;`可见 + 矩形正确 + 像素为空` ⇒ 整窗 alpha/形状问题(见 57)
- **验收探针必须跑 `a.Run()` 消息循环**(无泵时 Presence/定时器全不工作, 结论作废)
- 浏览器对照:shadcn 参考页 + xui demo 同屏对比;同一张截图里量两边(ours vs ref)是最可靠的对照法
- **弹层类组件的暗色验收一律 zoom/像素测量**:全屏缩略截图里暗色弹层会假显亮底(36)

## 环境 / 注意事项

- shadcn dev server 由会话后台启动,机器重启后需按上文命令重启
- hover 过渡用运行时 `anim.go` 的炫彩定时器(不是 ani 包), 时长可用 `xui.SetHoverDuration` 覆盖(传 0 = 关掉过渡);**新增 hover 反馈只用 `xui.Hover`/`HoverItem` + `Fade`, 别自己起定时器**
- 用户项目须 `app.New(true)`(D2D)+ `EnableAutoDPI`,demo 是标准样板
- 弹层/对话框过渡挂**宿主窗口**,别挂弹层小窗;要 Win32 HWND 用窗口 wrapper 的 `GetHWND()`,别拿 xcgui 窗口句柄喂 `XWidget_GetHWND`(弹模态错误框)

## 图标系统(速览)

- `icons/` 子包内置 **2130 个** lucide 图标(lucide-static **1.52.0**), **每图标一个 `IconXxx` 常量**;按名字取用 `icons.Path("eye")`(给"名字来自用户输入/配置"的场合)。数据由 `scripts/gen_icons.py` 生成(默认取最新版, `--version` 可钉版)
- 出图走**直通 D2D 几何填充**(`d2dicon.go`), GDI 模式(`app.New(false)`)自动回落折线描边
- 体积(实测, 同基准 `-s -w`): 只用常量 **+0 字节**;调 `Path/Names` 引用整张表 **+426KB**
- demo 第 22 页「图标」是浏览入口(`XUI_HOME_PAGE=21`):9×4 格 + 搜索 + 翻页, 悬浮出 Tooltip 显示图标名、点击复制
- **原理、坑、性能结论见 DESIGN-PITFALLS.md 第八节 43(抗锯齿 / figure 数)与 44(资源预处理 / 名字单射 / arc 标志位)**
