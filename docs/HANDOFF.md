# HANDOFF — 交接文档

> **本文只回答"现在到哪了"**:当前状态、待办、命令、仓库结构、验证工作流、去哪查。
> 设计原理 / xcgui 踩坑细节 / 交互语义**一律不在这里重复** —— 见 **DESIGN.md**(第八节 = 踩坑全集按编号;第四节末 = 语义权威汇总表)。agent 会话硬规则见 **AGENTS.md**,面向用户的用法见 **README.md**。

## 项目一句话

`github.com/twgh/xui`:为 xcgui(炫彩界面库)提供 **shadcn-ui 风格**的自绘组件——主题令牌、变体系统、源码拷贝分发(copy-not-install)。交互语义以本地 shadcn 参考实现(`D:\GOProject\src\ui\shadcn`)的源码为唯一权威。

## 当前状态(截至 2026-10-10)

**阶段**:P0 运行时 + P1/P2/P3 组件全部完成并视觉验收(P3 仅剩"CLI 远程 registry"与"native 弹层进场动画"两项未定);P4(文档站、主题画廊)未开始。路线图见 DESIGN 第七节。

**已完成(括号里是根因/踩坑所在的 DESIGN 第八节条目号)**:

- ✅ **P0 运行时**:Color/Theme 令牌、主题管理器(SetTheme 零成本换肤 + OnThemeChange)、Painter、字体缓存(单位坑见 31)、focus-visible 焦点来源跟踪(16)
- ✅ **P1 组件**:Button(6 变体×8 尺寸档)、Checkbox、Radio、Input、Label、Separator+Badge、Card
- ✅ **P2 组件**:Switch(2 尺寸)、Progress、Slider、Tabs(default/line)、Tooltip、ComboBox、图标系统
- ✅ **Slider 渲染缺陷彻底修复**:悬浮/拖动冒方块、两端 ring 被裁、灰环残留(根因与结论见 25/26/27/28)
- ✅ **CLI**:`init / add / list`(registry.json 驱动,组件源码拷入用户项目);`update / --all` **未实现**
- ✅ **demo + 用户项目 E2E**:拷贝出的组件在独立用户模块编译/运行/渲染全部正常
- ✅ **demo 布局化 + 布局 API 收口**(2026-10-04~08):改用 `widget.NewLayoutEle` 流式布局;布局项统一到 `xui.Item*`(只声明要改变的那条轴, 哨兵 `xui.ItemKeep=-1`), 容器骨架统一到 `xui.HStack` / `xui.VStackWeight` / `ui.Card.ContentStack`;语义与"纵向溢出换列"坑见 29/41
- ✅ **参考实现补齐**(本地 shadcn 对照页):switch/progress/slider/tabs/tooltip/select/table/dropdown-menu 源码落盘 + `App.tsx` 对照卡(含新第 6 页「图标」);`pnpm build` 通过
- ✅ **ComboBox 弹层对齐 shadcn**:ring 四边完整(画在 `XE_PAINT_END` 内缩 1px)、item-aligned 定位 + 屏幕边界截断滚动(30)
- ✅ **Dialog**(2026-10-04):路线 = 窗口内覆盖元素(全窗遮罩 + 居中面板 + 页脚 + 幽灵图标关闭钮), ESC / 点遮罩 / 关闭钮关闭 + Tab 焦点陷阱;backdrop-blur-xs(33/34/47)
- ✅ **Menu**(2026-10-05):原生菜单承载行为 + 窗口级 `XWM_MENU_DRAW_*` 自绘皮肤(36)
- ✅ **Table**(2026-10-05):原生 List 载体 + 外观全自绘(表头/行底/边框/滚动条)(40/41)
- ✅ **Tree**(2026-10-06):原生 Tree 载体 + `XE_TREE_DRAWITEM` 自绘;"隐藏容器首显空白"回归已改为组件内自愈(42/51)
- ✅ **hover 过渡动画**(2026-10-06):150ms cubic-bezier(0.4,0,0.2,1), 运行时 `anim.go` 驱动, 组件侧只用 `xui.Hover / HoverItem + Fade`(45)
- ✅ **重绘节流调优**:应用初始化 `a.SetPaintFrequency(10)`, 实测 30.9 → 59.0 fps;`XC_SetTextRenderingHint` 在 D2D 下是空操作(46)
- ✅ **弹层/对话框进出场动画**(2026-10-06):`xui.Presence(...)` + `SetElementsAlpha / FadeWindowAlpha / SlideWindowAlpha`;窗口 alpha 永远不精确 = 0(47)。⚠️ **异型透明窗上不要用整窗 alpha 做动画**(只认第一次调用, 见 57), 淡入淡出改成元素 alpha
- ✅ **backdrop-blur-xs**(2026-10-07):抓窗快照 + Go 盒式模糊 + BkM 图片填充;改物理像素管线后全屏弹窗 160ms → 61ms(47)
- ✅ **Pagination + demo 图标页改造**(2026-10-08):恒定 7 槽位折叠, 搜索 / 翻页 / 悬浮 Tooltip / 点击复制图标名(53)
- ✅ **图标系统**:抗锯齿改**直通 D2D 几何填充**(`d2dicon.go`);**lucide 资源预处理**(2130 个图标, `icons/` 子包)—— 原理 / 坑 / 性能见 43/44,速览见下方「图标系统」
- ✅ **与 shadcn 逐条核对变体/尺寸并补齐**(2026-10-09):核对本地 18 个 tsx 的 cva variants/props 后补齐 —— Button 补 Link 变体 + 8 档尺寸(xs/sm/default/lg + 4 个方形图标档)、Badge 补 ghost/link、Card 补 sm 尺寸与说明/页脚、ComboBox 补 sm 触发区、Slider 补 min/max/step、Separator/Tabs 补纵向取向;像素实测按钮高度 24/28/32/36 与 shadcn h-6/7/8/9 完全吻合。映射与**仍然存在的差异**(aria-invalid / CardAction / Menu inset / chart·sidebar 令牌)记在 DESIGN 第四节末「变体/尺寸与上游的映射、已知差异」;1px 细条的绘制结论见 10
- ✅ **demo 全面布局化 + 分类重规划 + `Card` 改布局容器**(2026-10-09):窗口 800×590 → **1200×860**(最小 960×660), 页面/卡片去掉全部"算出来的绝对坐标", 改为流式布局(窗口最大化即整页自适应, 像素实测: 卡片左边距 36 = 阴影 12 + 页边距 24 ✓, 最大化时 24 ✓, 数据页表格跟随长到 ~1000 逻辑高)。页签重规划为 **基础 / 表单 / 导航 / 反馈 / 数据 / 图标**(原"控件/结构/新增变体"散布的内容各归其位, 不再挤在一页)。`Card` 底层换 `LayoutEle`、内容区随卡片尺寸联动(见 DESIGN 第八节 56);布局尺寸语义(尤其"`Fill` ≠ 剩余空间"、"`Weight` 需父级确定尺寸")见第八节 55
- ✅ **页面宿主的"叠放换页"不需要绝对定位**(2026-10-09):布局会跳过 `Show(false)` 的子项(排成 0x0), 所以 6 个页是宿主里的 6 个 Fill/Weight 兄弟, 切页只 `Show` + `AdjustLayout`(实测, 见 55)
- ✅ **Tooltip 气泡"显示着却看不见"根因修复**(2026-10-09):症状是悬停后提示不出现(用户侧描述"点几次切换主题后 ToolTip 就不显示了"), 实际窗口**被显示出来了**(可见、矩形正确、位置还随动画滑动), 但 layered alpha 被永久钉在 3/255 —— 异型透明窗的整窗透明度**只认第一次调用**(探针复现, 见 57)。修法: 进出场改淡**元素** alpha, 整窗 alpha 只在显示后设一次 1; 验收=亮/暗双主题各 3 轮、连续 6 次切主题后每轮气泡像素增量完全一致
- ✅ **Tooltip 跟随主题切换**(2026-10-09):气泡挂在屏幕上时点"切换主题", 它过去会保持旧配色 —— 自建的独立置顶小窗不在 `RegisterWindow` 重绘清单里, 得自己 `OnThemeChangeFor` 订阅并重算+重绘(见 58); 顺带把窗口扩展样式改成 `window.NewEx` 一次传入。验收=光标停在主题按钮上直接点它, 气泡当场从深色块变浅色块(`.verify/hovercheck.ps1`)

- ✅ **表格"首屏自适应列宽塌成最小值"修复**(2026-10-10):数据页刚进来时最后一列只有 20px(表头裁成"日"、单元格全显示 `...`), 拉一下窗口才正常 —— 根因是"列宽按元素宽度算, 而 `Show()` 之前元素宽度是 0"; 现在组件自己订阅 `List.AddEvent_Size` 自愈(`lastW` 防震荡), 调用方不必再记得调 `FitColumns`(见 60)
- ✅ **暗色禁用/未选态配色按 shadcn 逐像素对齐**(2026-10-10):复选框未选态(暗色 `bg-input/30`)、复选框/单选框/开关的禁用态、禁用输入框过去都错用 `WithAlpha` 覆盖半透明令牌(暗色下画出发亮灰块 / 亮白框线 / 全亮白滑块)。现在统一为"**先铺宿主底色(Card) + 每个图层各乘 50%**":形状全部跟随, 原生文字保持原色, 输入框禁用文字预混变淡(见 61); 顺带修掉透明元素重绘**不擦底**留下的旧像素(焦点环 / 上一个状态)。验收=探针 `.verify/statecheck` 让"切换后的渲染"与"直接建成目标状态"逐像素比, 亮/暗各一轮都是 **0 像素差**(色值与 shadcn 的整组 50% 合成理论值逐条吻合)
- ✅ **本地 shadcn 参照页(界面展示)与新 demo 同构**(2026-10-10):`shadcn/src/App.tsx` 重整为与 demo 一致的六页(**基础 / 表单 / 导航 / 反馈 / 数据 / 图标**), 内容逐条对应(6 变体 × 4 档尺寸 + 4 方形图标档、徽章 6 变体、CardFooter、多行与禁用输入、滑动条两档、开关 default/sm、Tabs default/line/纵向、恒 7 槽分页、表格 12 行 + 树); 新增 `?tab=form` 这类初始页签参数(对应 demo 的 `XUI_HOME_PAGE`), 便于分页截图对照; `pnpm build` 通过

## 待办

- **页面滚动容器**: 页面内容比窗口高时, 现在被页面自身的 `EnableOverflowHide` 裁掉(表单页在默认窗口下底部约裁掉一行)。要么给页面套滚动容器(`widget.ScrollView` 一类的自绘滚动条 + 内容高度撑开), 要么让内容自适应(见 DESIGN 8.59)

- ⬜ **P3 剩余**:CLI 远程 registry;menu/combobox 弹层的进场动画(native 弹窗淡入不可行, 只能做位置滑动, 待定)
- ⬜ **发布 GitHub**(发布后 `go run github.com/twgh/xui/cmd/xui@latest init` 即可用;CLI 已按此设计)
- ⬜ **与 shadcn 仍未对齐的边角**(完整清单见 DESIGN 第四节末):Button 的 `aria-invalid` 态、Card 的 `CardAction`、Menu 的 `inset`、`--chart-1..5`/`--sidebar-*` 令牌未建模
- ⬜ text 手绘后 Input 仍是原生文字
- ⬜ CLI `update / --all` 未实现(见上)

## 仓库结构

```
D:\GOProject\src\ui\xui\          ← 本仓库(模块 github.com/twgh/xui)
├── *.go                          运行时: color / theme / themes_builtin / manager
│                                 draw(Painter) / icons(图标几何+D2D 直通出图) / focus / font / state
│                                 anim(hover 过渡) / presence(进出场) / container·layout(布局骨架)
├── components\                   组件源码 = registry 内容(扁平单包 package ui)
│                                 button checkbox radio input label separator(含 Badge) card
│                                 switch progress slider tabs tooltip combobox dialog
│                                 menu table tree pagination
├── icons\                        图标资源子包: lucide 预处理数据(2130 个) + 每图标一个
│                                 IconXxx 常量 + Path/Has/Names/Count/Version
├── scripts\gen_icons.py          lucide-static SVG → icons\lucide_gen.go(名字表)
│                                 + icons\lucide_icons_*.go(图标常量, 每 400 个一个文件)
├── cmd\xui\                      CLI(init/add/list)
├── icons_test.go                 解析器语法 + 资源完整性测试(go test ./...)
├── demo\                         演示应用(也是全组件编译测试目标)
├── registry.json                 组件清单(files + deps)
├── docs\DESIGN.md                知识库: 架构设计 + xcgui 踩坑全集(第八节必读)
├── docs\HANDOFF.md               本文
├── README.md                     用法
└── AGENTS.md                     agent 会话自动加载的入口规则

D:\GOProject\src\ui\shadcn\       ← 本地部署的 shadcn-ui 参考实现(语义权威)
├── src\components\ui\*.tsx       组件语义的唯一来源(hover/focus-visible/disabled 等)
├── src\index.css                 主题令牌(:root/.dark, nova 预设 neutral 色板)
└── src\App.tsx                   与 demo 同构的对照页

D:\GOProject\src\tools\llama-launcher\  ← 同技术栈的成品项目(可参考)
├── internal\ui\caption.go        菜单美化, 标题上自定义按钮
└── internal\ui\builder.go        List 美化, LayoutEle 布局盒子的成熟用法
                                  其布局语义结论与本项目 DESIGN.md 第八节 29 一致
```

## 常用命令

```bash
# xui demo(全部组件的编译+视觉验证目标)
cd D:\GOProject\src\ui\xui
go build -ldflags="-s -w -H windowsgui" -trimpath -o demo/demo.exe ./demo && demo\demo.exe

# shadcn 参考实现 dev server(可能已在运行: netstat -ano | findstr 5173)
cd D:\GOProject\src\ui\shadcn && pnpm dev    # http://localhost:5173

# CLI 本地测试(仓库未发布, 远程 go run 会失败)
go run ./cmd/xui list / init <目录> / add <组件...>

# 暗色对照验收: 环境变量决定启动主题, 不用改代码
#     XUI_THEME=dark  启动 demo 即暗色(cmd: set XUI_THEME=dark && demo\demo.exe)
#     运行中仍可用右上角按钮切换(参考页 shadcn 同样支持按 d 键切换)
# 直接启动到指定页签(验收某一页时省得点): XUI_HOME_PAGE=0..5 = 基础/表单/导航/反馈/数据/图标

# 全部测试(图标解析器语法 + 资源完整性 + 缓动曲线)
go test ./...
```

## 关键设计决策(一句话版, 详见 DESIGN.md)

1. **copy-not-install**:组件源码由 CLI 拷进用户 `ui/` 目录,归用户;运行时(theme/icons/helpers)走 go.mod
2. **绘制时取令牌**:组件永不缓存颜色,paint 回调里 `xui.Current()` 现取 → 切主题 = 换指针 + Redraw
3. **行为/视觉分层**:xcgui 原生控件 = 无头行为层,`AddEvent_Paint` + `drawx` 只管视觉
4. **交互语义逐条对齐 shadcn 源码**:改交互前先在 DESIGN 第四节末的语义汇总表确认,再读 `shadcn/src/components/ui/*.tsx`
5. **组件文件写薄**:绘制细节下沉运行时助手,降低拷贝模式下的升级冲突面
6. **界面用流式布局表达,不写绝对坐标**:窗口是唯一"天然确定尺寸"的父级,竖向主链从窗口开始一路 `Weight`(每级确定尺寸,下一级的 `Weight` 才算得出剩余空间);卡片高度用 `Auto` 自撑,要长高的面板整条链全 `Weight`。**别在中间垫 `Fill` 容器,也别用 `Fill` 当"剩余空间"**(尺寸语义见 DESIGN 第八节 55,样板在 `demo/main.go` 的 `row`/`colStretch`/`autoCard`)

## 文档导航(去哪查什么)

| 想了解 | 去哪 |
|---|---|
| 当前进度 / 待办 / 命令 / 仓库结构 / 验证工作流 | **本文** |
| 架构原理、分层边界、主题令牌、组件解剖 | DESIGN.md 一~四节 |
| **交互语义(改交互前必看)** | DESIGN.md 第四节末「与 shadcn 对齐的语义(权威汇总)」+ 本地 shadcn 源码 |
| **我们与 shadcn 的差异 / 变体尺寸映射** | DESIGN.md 第四节末「变体/尺寸与上游的映射、已知差异」 |
| 某个 xcgui 行为 / 为什么这么写(踩坑) | DESIGN.md 第八节(按编号, 1~58) |
| 界面布局怎么写(容器/尺寸/随窗口自适应) | DESIGN.md 第八节 55(尺寸语义) + 56(Card), 样板在 `demo/main.go` |
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
- 布局类问题**用 `xc.XEle_GetRectLogic` 打印各元素矩形**判定, 比截图量像素快且准(探针里 `AdjustLayout()` 之后读;注意窗口必须 `Show` 过才有非零尺寸, 见 DESIGN 8.55)
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
- demo 第 6 页「图标」是浏览入口(`XUI_HOME_PAGE=5`):9×4 格 + 搜索 + 翻页, 悬浮出 Tooltip 显示图标名、点击复制
- **原理、坑、性能结论见 DESIGN 第八节 43(抗锯齿 / figure 数)与 44(资源预处理 / 名字单射 / arc 标志位)**
