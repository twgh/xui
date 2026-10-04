# xui — shadcn 风格的 xcgui 自绘控件库设计

> 基于 xcgui release-1.4.2 源码验证。目标:像 shadcn-ui 一样,把自绘组件的**源码**交到用户手里,由用户掌控主题与样式。
>
> **状态(2026-10-03)**:P0 运行时 + P1 组件 + CLI 已实现并通过视觉验收,交互语义已与本地 shadcn 参考实现(`D:\GOProject\src\ui\shadcn`)逐条对齐。本文是设计时的架构文档;**当前进度与接手指南见 HANDOFF.md**,实现细节以 `components/*.go` 与 `AGENTS.md` 为准。

---

## 一、先拆解 shadcn 到底做了什么

shadcn-ui 不是传统意义的"组件库依赖",它由四个机制构成:

| 机制 | 说明 |
|---|---|
| **copy-not-install** | 组件不是 npm 包。`npx shadcn add button` 把 `button.tsx` **源码复制**进你项目的 `components/ui/` 目录。代码归你所有,可随意改,git 跟踪,CLI 可更新 |
| **行为/视觉分层** | Radix UI 提供"无头"原语(状态机、键盘导航、无障碍),零视觉;Tailwind 提供视觉。shadcn = 二者的意见组合,以源码形式暴露 |
| **设计令牌 = CSS 变量** | 主题是一组 CSS 自定义属性(`--background` `--primary` `--radius`…),组件代码里**只引用变量名,绝不写死颜色**。暗色模式 = 同名变量换值。因此上千个社区主题可以即插即用 |
| **变体系统** | 每个组件有命名变体(button: default/secondary/destructive/outline/ghost/link)和尺寸(sm/default/lg/icon),由 cva 声明式映射"变体→样式" |

**关键认知:xcgui 的原生控件就是"无头"层。** 它丑,但事件、状态、焦点、输入法、滚动全都齐全。官方 BeautifyButton 等示例已验证:嵌入原生控件 + `AddEvent_Paint` 自绘 + `EnableBkTransparent(true)` 隐藏原生外观,即可完整复刻 shadcn 的分层。

### 概念映射表

| shadcn | xui 对应物 |
|---|---|
| Radix 无头原语 | xcgui `widget` 原生控件(行为层,现成) |
| Tailwind 视觉 | `drawx.Draw` + paint 事件自绘(视觉层) |
| CSS 变量 | Go `Theme` 结构体 + **绘制时**取 token |
| `components/ui/` 拷贝源码 | 用户项目 `ui/` 目录拷贝 .go 源码 |
| `npx shadcn add button` | `xui add button` CLI |
| registry.json | 组件清单(内嵌 CLI,后续可远程) |
| lucide 图标 | 内嵌 SVG 字符串 + `svg.SetUserFillColor` 按主题着色 |
| cva 变体 | 枚举 variant/size + 纯函数 `(variant, state) → token` |

---

## 二、总体架构:哪些是依赖,哪些拷贝

```
github.com/twgh/xcgui            行为层(原生控件,现成)
        ↑
github.com/twgh/xui               运行时模块(go.mod 依赖,不拷贝)
  ├─ theme.go                      主题管理器 + 令牌定义
  ├─ color.go                      Color 类型(Hex 解析,底层 xc.HexRGB2RGBA)
  ├─ themes/                       内置 light/dark + 移植的 shadcn 主题
  ├─ icons/                        内嵌 lucide SVG(上千个,MB 级)
  ├─ draw.go                       绘制助手(圆角卡片/焦点环/文字度量/阴影)
  └─ state.go                      HoverTracker 等状态跟踪
        ↑
用户项目 ui/*.go                  组件源码(CLI 拷贝,用户所有)
  ├─ theme.go                      用户自己的主题(init 时拷贝)
  ├─ button.go
  └─ dialog.go
        ↑
用户应用
```

**边界原则:基础设施走依赖,用户会改的走拷贝。**

- 运行时模块走 go.mod:主题管理器需要全局注册/订阅;图标库是 MB 级内嵌资源;绘制助手的 bug 修复靠版本升级。这些是"Radix + Tailwind"的角色。
- 组件源码拷贝:这是用户定制的发生地(shadcn 的核心卖点——"组件是起点,不是黑盒")。
- **组件文件要写得"薄"**:paint 细节尽量调用运行时助手,拷贝文件里主要剩下"变体→token 映射 + 几笔绘制调用"。越薄,拷贝模式的升级风险越小。shadcn 正是这样(cn()、Slot、Radix 全在依赖里,拷贝的文件基本是 JSX + 类名)。

---

## 三、主题系统(核心中的核心)

### 3.1 令牌集:与 shadcn 一一对应

```go
type Theme struct {
    Background, Foreground         Color
    Card, CardForeground           Color
    Primary, PrimaryForeground     Color
    Secondary, SecondaryForeground Color
    Muted, MutedForeground         Color
    Accent, AccentForeground       Color
    Destructive, DestructiveForeground Color
    Border, Input, Ring            Color
    Radius                         int32 // 基准圆角 px
    FontDefault, FontMono          FontSpec
}
```

Color 封装 uint32,提供 `xui.Hex("#fafafa")`(底层 `xc.HexRGB2RGBA`)。主题文件长得和 CSS 一样,任何 shadcn 社区主题都能机械转译成 Go:

```go
var Dark = xui.Theme{
    Background: xui.Hex("#09090b"),
    Foreground: xui.Hex("#fafafa"),
    Primary:    xui.Hex("#18181b"),
    ...
    Radius:     8,
}
```

### 3.2 关键决策:绘制时解析 token,组件永不缓存颜色

官方 BeautifyButton 示例把颜色缓存在结构体字段里,再用 SetThemeColor 重算——**这是让主题系统难做的反模式**。xui 组件一律:

```go
func (b *Button) onPaint(hEle, hDraw int, pbHandled *bool) int {
    t := xui.Current()          // 每帧取当前主题
    bg := t.Primary             // 绘制时解析
    ...
}
```

推论:
- **主题切换 = 换全局指针 + Redraw**,所有组件下一帧自动用新色。这正是 CSS 变量的工作方式,而且是零成本的。
- 变体逻辑是纯函数 `(variant, state) → token`,不开窗口也能单元测试。

### 3.3 主题管理器

```go
xui.SetTheme(&myTheme)   // 切换 + 通知已注册窗口重绘(内部 XWnd_Redraw)
xui.OnThemeChange(fn)    // 需要同步原生 API 颜色的地方订阅(如 SetTextColor)
xui.Themes.Dark()        // 内置主题
```

组件若只在 paint 里取色,**无需订阅**;只有同步给原生 API 的颜色(文字色、窗口底色)才需要回调。

---

## 四、组件解剖:以 Button 为例

分层套路(四步,所有组件通用):

1. **嵌入原生控件** `*widget.Button` —— 行为(点击/状态/焦点)白拿
2. **EnableBkTransparent(true)** —— 灭掉原生外观
3. **paint 事件画壳** —— 背景(变体×状态)、边框、焦点环
4. **文字交给原生绘制** —— 不置 `pbHandled`,原生文字/图标接着画,清晰且 DPI 正确;只有 per-state 文字色、图标按钮、复选框勾这类需求才手工 `DrawText`

```go
package ui

type ButtonVariant int

const (
    BtnPrimary ButtonVariant = iota
    BtnSecondary
    BtnOutline    // 透明底 + Border token
    BtnGhost      // 透明底, hover 用 Accent
    BtnDestructive
)

type ButtonSize int

const (BtnSm, BtnMd, BtnLg, BtnIcon)

type Button struct {
    *widget.Button
    variant ButtonVariant
    size    ButtonSize
}

type ButtonConfig func(*Button) // 函数选项模式, 等价 cva 的 variants 参数

func Variant(v ButtonVariant) ButtonConfig { return func(b *Button) { b.variant = v } }
func Size(s ButtonSize) ButtonConfig       { return func(b *Button) { b.size = s } }

func NewButton(x, y, cx, cy int32, text string, hParent int, cfgs ...ButtonConfig) *Button {
    b := &Button{Button: widget.NewButton(x, y, cx, cy, text, hParent)}
    for _, c := range cfgs {
        c(b)
    }
    b.EnableBkTransparent(true)
    b.AddEvent_Paint(b.onPaint)
    return b
}

func (b *Button) onPaint(hEle, hDraw int, pbHandled *bool) int {
    t := xui.Current()
    d := drawx.NewByHandle(hDraw)
    d.EnableSmoothingMode(true)
    var rc xc.RECT
    xc.XEle_GetClientRect(hEle, &rc)

    state := xc.XBtn_GetStateEx(hEle)        // Stay/Down/Disable
    bg, border := resolve(t, b.variant, state) // 纯函数: (变体,状态)→token
    radius := t.ScaledRadius(hEle)           // 内部按 XWnd_GetDPI 缩放

    if bg != xui.None {
        d.SetBrushColor(bg)
        d.FillRoundRect(&rc, radius, radius)
    }
    if border != xui.None {
        d.SetLineWidth(1)
        d.SetBrushColor(border)
        d.DrawRoundRect(&rc, radius, radius)
    }
    return 0 // 原生文字继续绘制
}
```

### 各组件技术路线(每个 = 一份 paint 配方)

| 组件 | 载体 | 方案 | 验证来源 |
|---|---|---|---|
| Button | embed Button | paint 壳, 原生文字 | BeautifyButton 示例 |
| Checkbox/Radio | embed Button | 画方框/圆 + 勾线, 文字原生 | BeautifyCheckBox/RadioButton 示例 |
| Input | embed Edit | 只画边框+焦点环, 原生光标/IME | BeautifyEdit 示例 |
| Switch | embed Button | 轨道+滑块, ani 做过渡(可选) | — |
| Progress/Slider | embed 原生 | 官方配方移植成 token | BeautifyProgressBar/SliderBar 示例 |
| ComboBox | embed ComboBox | 画输入框; 下拉 ListView 在 `AddEvent_Combobox_Popup_List` 里皮肤化 | 库文档 FAQ |
| Tabs | embed TabBar | paint | — |
| Dialog | ModalWindow | 半透明遮罩 + Card; **DoModal 返回后子元素已销毁, API 必须先取值再 EndModal** | 库文档 |
| Menu | 原生菜单 | `XMenu_EnableDrawBackground/EnableDrawItem` + `AddEvent_Menu_Draw_Background/DrawItem` | element.go 已确认事件存在 |
| Card/Badge/Separator | 纯 Element | paint | — |
| Tooltip | FloatWindow | paint | — |
| Table/List/Tree | 原生 + 项模板 | 模板创建事件里皮肤化 + EnableTemplateReuse | 库文档 |
| Icon | svg.Svg | 内嵌 lucide(MIT), `SetUserFillColor` 主题着色, `d.DrawSvgSize` 绘制 | svg 包已确认 |

非 Button 载体的组件(hover 态自管)用运行时提供的 `HoverTracker`(基于 `AddEvent_MouseStay/MouseLeave`,已确认存在)。

尺寸预设对标 shadcn(`h-9 px-4` 之类),经 DPI 助手换算。

---

## 五、CLI 与 registry

```
xui init                    # 生成 xui.json{ui目录, 基础主题, 模块名}, 拷贝 theme.go, go get 运行时
xui add button checkbox     # 按清单拷贝组件源码 + 其依赖组件, go mod tidy
xui add --all
xui update                  # 重新拷贝; 本地已改动的文件(哈希对比)警告而非覆盖
```

registry.json 条目(shadcn 同构格式):

```json
{ "name": "dialog", "files": ["dialog.go"], "deps": ["button", "card"], "registryDependencies": [] }
```

- v1:registry + 组件源码内嵌 CLI 二进制,单文件离线可用
- v2:同格式改为远程拉取(GitHub 仓库),CLI 不发版也能更新组件

组件源码放在仓库 `components/` 目录,demo 应用直接 import 它们编译——**demo 编译通过 = 全部组件可用**,CI 即测试。

---

## 六、仓库结构

```
xui/
├── xui/            # 运行时模块(theme/color/icons/draw helpers/state)
├── components/     # 组件源码 = registry 内容(每组件一目录)
├── cmd/xui/        # CLI
├── demo/           # 演示应用(CI 编译目标 + 截图生成)
├── registry.json
└── docs/
```

用户项目执行 `xui init && xui add button` 后:

```
myapp/
├── go.mod      # + require github.com/twgh/xui
└── ui/
    ├── theme.go    # 用户自己的主题(init 拷贝, 想怎么改怎么改)
    └── button.go   # 用户所有, 可改
```

---

## 七、路线图

| 阶段 | 内容 | 验收标准 | 状态 |
|---|---|---|---|
| **P0 运行时** | Color、Theme 令牌、主题管理器、内置主题、绘制助手、HoverTracker、DPI 助手 | 单测通过(变体纯函数不开窗可测) | ✅ |
| **P1 MVP** | Button、Checkbox、Radio、Input、Label、Separator、Badge、Card + demo 页 | light/dark × hover/focus/disabled × DPI 截图 | ✅ |
| **P2** | Switch、Slider、Progress、Tabs、ComboBox、Tooltip、图标系统 + 暗色切换演示 | 同上 | ⬜ |
| **P3** | Dialog、Menu 皮肤、Table/List/Tree 皮肤、hover 过渡动画、CLI 完整化(update) | 同上 | ⬜ |
| **P4** | 文档站、主题画廊(批量转译 shadcn 社区主题为 Go) | — | ⬜ |

---

## 八、风险与已规避项(基于 1.4.2 源码逐条验证)

1. **渲染模式二选一**:`app.New(true)` = Direct2D,`false` = GDI+。`GDI_*` 系列函数仅 GDI+ 模式可用,圆角抗锯齿 D2D 更好。**运行时统一按 D2D 设计**,demo 固定 D2D。
2. **2000 回调上限**:一律 `AddEvent*` 系列注册(复用回调,无上限风险)。
3. **背景管理器累积上限 20**:组件全部走 paint 事件,完全不碰 bkmanager。
4. **主题切换必须 Redraw**:主题管理器统一触发 `XWnd_Redraw`,组件自己不管。
5. **模态窗口销毁**:Dialog API 设计为"先取值再 EndModal"。
6. **DPI(实测 DPI=144 修正)**:元素创建坐标会被缩放渲染(24 逻辑 → 36 物理),但 `XEle_GetClientRect` 返回的是**未缩放的逻辑矩形**,且绘制上下文按逻辑坐标变换。因此自绘时**不要**按 `XWnd_GetDPI` 预缩放尺寸——二次缩放会让指示器撑满客户区被上下裁剪(圆变八边形)。这也是库中存在 `*DPI` 后缀函数的原因:普通版返回逻辑坐标。
7. **Edit 自绘**:原生背景需 `EnableBkTransparent` 处理,BeautifyEdit 已验证。
8. **字体**:主题携带字体方案(微软雅黑等中文环境默认字体),font 包管理。
9. **拷贝模式的代价**:组件 bug 不会随依赖自动修复 → 用 `xui update` + "组件文件写薄"缓解。
10. **元素绘制裁剪**:绘制被裁剪在元素自身客户区内,描边沿路径居中(贴边描边外半侧被裁),外扩式焦点环完全不可见 → 焦点框画在边框内侧(填充式,见 FocusFrame),需完整可见的描边请内缩。`XDraw_SetClipRect` 放宽裁剪区**无法突破**这层元素裁剪(实测:按下位移把底边推出 3px 后仍被裁掉,壳体高度少 3px)——要做"整体位移"这类越界绘制,应给元素预留像素(见 `button.go` 的 `pressShift`)。
11. **实心按钮 hover 用透明度**:shadcn 的 `hover:bg-primary/80` 语义用 `WithAlpha` 实现,比 lighten/darken 在近黑/近白主题色上有可感知变化,且明暗主题行为一致。
12. **cy=0 的控件高度就是 0**:xcgui 不会按内容自适应,高度为 0 的控件直接不渲染(探针实证)→ 组件构造函数对 cy<=0 一律补默认高度。
13. **窗口背景必须用 Leave 状态注册**:xcgui 的 `XBkM_AddFill` 按"当前组合状态"选填充,窗口绘制时处于 `Window_State_Flag_Leave`,用 `Element_State_Flag_Nothing(0)` 注册的填充永远不被选中;且需先 `XBkM_Clear` 清默认填充(paint 事件 / BkManagerEx / SetBkInfo XML 均无法改变窗口背景,探针实证)。已封装为 `xui.BindWindowBackground`。
14. **标题文字默认白色**:xcgui 标题栏文字颜色为白色(为默认蓝色标题栏设计),主题化窗口(白/黑底)需 `w.SetTitleColor(theme.Foreground)` 并挂 OnThemeChange。
15. **窗口样式按标志位组合**:去掉系统图标和三键 = `Caption|Border|Center|Drag_Border|Allow_MaxWindow|Title`(不含 Icon/Btn_*);去掉关闭按钮后需自补 ESC 关闭(`XWnd_CloseWindow`)。
16. **focus-visible 需自行跟踪焦点来源**:xcgui 只有单一 Focus 状态,不区分鼠标/键盘来源 → 全局跟踪鼠标按下/Tab 键(运行时 focus.go),文本输入类控件例外(始终显示,同浏览器语义)。
17. **元素自绘里画不出某些 Lucide 图片/SVG, 因为不支持某些写法**:`XDraw_DrawSvgEx` / `XDraw_Image` / `XDraw_ImageEx` 在元素 paint 回调里**无输出**(探针实证:句柄有效、坐标与裁剪都正确,就是画不出来;图片用 `imagex.NewBySvgString` 加载同样无效)。需要图标时用线段图元复刻:`icons.go` 把 lucide 的 path 解析成折线再用 `DrawLine` 描边(支持 M/L/H/V/C/S/Q/T/Z,圆弧用专门实现);或把图片交给原生控件(`Button.SetIcon` 等)。
18. **`XWnd_SetWindowPos` 吃物理像素**:框架内部坐标是逻辑像素(窗口创建时自动按 DPI 缩放),但窗口定位/尺寸透传给 Win32 → 必须自己乘 `dpi/96`,否则窗口会跑到 1/scale 的位置(实测 144 DPI 下 449 变成 299)。自绘提示窗(Tooltip)踩过这个坑。
19. **原生 ComboBox 必须先 `CreateAdapter`**:否则 `AddItemText` / `GetCount` 会让炫彩弹"XAdTable_GetCountRow 中,输入句柄可能无效[0x0]"的错误框,并阻塞后续调用。
20. **SliderBar 的滑块几何**:pos→滑块按钮是 `[0, W - buttonW]` 线性映射(实测 pos=0/25/50/100 → 0/47/94/188, 宽 200 时)。自绘时直接读 `XEle_GetRect(thumb)` 拿滑块矩形,与拖动行为严格一致;滑块外扩 3px 的焦点环要画在**滑动条**的 paint 里(滑块自身客户区只有 12x12,画在自己回调里会被裁)。
21. **原生滚动条要手动收敛**:弹层 ListBox 的滚动条默认是系统配色(与主题完全不搭)。做法:`lb.SetScrollBarSize(8)` + `lb.EnableAutoShowScrollBar(true)` + `sb.ShowButton(false)`(去掉上下箭头)+ `sb.SetSliderPadding(2)` + 轨道透明,滑块用 `AddBkFill(Leave/Stay, Border/MutedFore)` 上色;这些背景对象在主题切换时要 `ClearBkInfo()` 后再重新添加(否则累积超过 20 个炫彩会报错)。
22. **SliderBar 的滑块按钮在 hover 时会冒方块**:原生 Button 的四态会画边框/焦点矩形,`EnableBkTransparent(true)` + `EnableDrawBorder(false)` + `EnableDrawFocus(false)` 三件套只能压住一部分——Focus 态的方块三件套+`ClearBkInfo()`+空 paint 回调都压不住, 最终结论见第 25 条(弃用原生按钮, 纯自绘)。
23. **原生绘制的文字不要自己再画一遍**:ComboBox 这类由原生绘制文本的控件,在 paint 里再画一次文字会出现重影(错位的一层) → 文本交给原生,只在 paint 里同步颜色(`SetTextColor` / `SetDefaultTextColor`),`*pbHandled = true` 只能拦住"绘制回调"本身,拦不住原生文本。
24. **异型透明窗口可以做到真透明,且背景对象支持圆角**(旧结论"背景管理器没有圆角矩形对象,开了 Window_Transparent_Shaped 整窗什么都不画"是错的,已由探针推翻):
    - 形状来源:`Window_Transparent_Shaped` + `SetTransparentAlpha(255)` 后,窗口形状 = **自身绘制内容中不透明像素的并集**,并不是只认背景对象。因此**仅靠 paint 自绘**就能拿到带真实 alpha 边缘的形状——气泡圆角外、箭头两侧完全透明,压在其它应用画面上也不露方块。
    - **前提**:不能给窗口再挂背景填充(如 `xui.BindWindowBackground`),那会把整窗铺成不透明矩形,透明随即失效。Tooltip 原来的"四周同色方块"就是这个原因,不是形状机制做不到。
    - 对照实测(探针,窗口矩形内与基线逐像素比较):仅 paint 自绘 + 异型 → 88% 像素变化,气泡下方一行 0 像素变化(真透明);整窗铺主题底色 → 99% 像素变化,气泡下方 453/480 像素变化(不透明)。同一份 paint 代码,只差是否挂窗口填充。
    - 想用背景对象做圆角也行:`bkm.AddFill(state, color, id)` 传入**非 0 的 id**,再 `bkm.GetObjectObj(id).SetRectRoundAngle(左上,左下,右上,右下)`(配 `SetMargin` 定位置尺寸)即可得到圆角矩形;该对象还支持 `SetRotate`、`SetBorderColor` 等更多美化。官方示例 `Advanced/WindowPartiallyTransparent`(窗口级 bk 管理器)与 `webview/WebViewPartiallyTransparent`(布局元素级 bk 管理器)都用了这一手法。
    - 生命周期无坑:`Window_Style_Nothing` 的窗口按"先建 10x10 → 首次显示前改尺寸/位置 → 复用再改尺寸"的懒创建流程走,异型透明一切正常(探针实证)。
    - ⚠️ 附带发现:`Window_Style_Nothing` / `Window_Style_Center` 的窗口**自身不绘制任何东西**(只画子元素),拿它当普通宿主窗口会"看不见窗口";要正常显示请用 `Window_Style_Default|Drag_Window`。
25. **别跟原生 SliderBar 的滑块按钮(原生 Button)纠缠, 要么全套压制要么完全弃用**(实测两轮踩坑):它的默认样式通过 **BkInfo** 给 Stay/Down/Focus 态注册了填充+边框——悬浮冒灰色方块、拖动后冒白色方块、点击聚焦后又冒一种方块, 形态还各不相同。`ClearBkInfo()`(实测 bkinfo=0 仍漏)+ `EnableBkTransparent` + `EnableDrawBorder/DrawFocus(false)` + 空 paint 回调(`pbHandled=true`) 全部叠加也压不住 Focus 态;而 `EnableFocus(false)` 虽然能断掉焦点方块, 却会让按钮**不再上报 Stay 状态**(悬浮检测失效, hover 环画不出来)。结论:Slider 最终**弃用原生 SliderBar 载体**, 改用普通 `widget.NewElement` + 自实现交互(LButtonDown 里 `XEle_SetCapture(true)` 捕获鼠标, MouseMove 持续更新, LButtonUp 释放; MouseStay/MouseLeave 自跟踪 hover; KeyDown 左右 ±5%/Home/End), 绘制 100% 自绘后一切干净。**教训:当原生控件的残留绘制出现在多个互不相干的状态路径上时, 逐个压制是无底洞, 直接换纯自绘。**
26. **Slider 几何要对齐 radix**:滑块**中心**在轨道两端之间移动(滑块会向轨道端点外伸出半个身位), hover/active/focus-visible 的 ring-3 再向外扩 3px;而元素自绘被裁剪在自身客户区内(见第 10 条)。因此:元素左右各比**可见轨道**宽出 9px(滑块半径 6 + ring 3), 轨道画在内缩 9px 处, **入参 cx 仍是可见轨道宽度**;滑块中心由 pos 自算, 两端 ring 刚好落在元素边缘, 完整可见。
27. **`XDraw_FillRoundRect`/`XDraw_DrawRoundRect` 的 nWidth/nHeight 就是圆角半径**(曾误判为"直径"并把 Painter 里 x2, 导致按钮/开关/下拉框全体过圆成胶囊形, 已回退):判断依据是对照实验——按钮等组件按半径直传时视觉与 shadcn 一致,x2 后过圆。`xui.Painter.FillRound/StrokeRound` 直传半径即可, 不要再自行换算。(Slider 上最初看到的"方块光环"其实是原生按钮 BkInfo 残留, 见第 25 条, 与 FillRound 无关。)
28. **不整体填充的透明元素必须在 onPaint 开头先铺宿主底色, 否则上一帧像素残留**(实测踩坑):Slider 的轨道/滑块/ring 都是局部形状, 元素背景透明, 状态变化触发重绘时框架**不会擦除旧像素**——hover 离开/点击聚焦后灰环一直留在画布上(`Redraw(true)` 立即重绘也无效, 因为重绘本身不擦底)。修法:onPaint 第一步 `p.FillRound(&p.Rc, t.Card, 0)` 铺满宿主底色(组件放在卡片上用 Card; 放在窗口背景上需对应调整)。Switch/Button 不留残影正是因为它们整体填充了自己的矩形。
29. **LayoutEle 布局语义与文档直觉不符, 全部经探针实测**(探针结论, demo 布局化依赖):LayoutEle 实为"横向 flex 流 + 自动换行"盒子——
    - **默认就是横向流**, 纵向堆叠必须 `EnableHorizon(false)`(列布局);不设间距时多个 Fixed 项会并排塞满一行再换行, 绝不是想象中的竖排;
    - 列布局的主轴(纵向)间距 = `SetSpace`, 交叉轴(水平)对齐 = `SetAlignH`;行布局的行内间距 = `SetSpace`, 换行后的行距 = `SetSpaceRow`(只设 SetSpace 纵向间隙恒为 0), 交叉轴(垂直)对齐 = `SetAlignV`;
    - **`Layout_Size_Fill` = 独占一整行/列**, 兄弟项被挤到下一行(甚至挤出布局而不换行), 不能与 Fixed 项同行;等分剩余空间用 `Layout_Size_Weight`(已验证多列均分);
    - 项级 `XWidget_LayoutItem_SetAlign` 实测无效, 交叉轴对齐只能用布局级 `SetAlignH/V`(对等高项无副作用, 混合高度行居中靠它);
    - `XWidget_LayoutItem_SetMargin` 各方向均生效;布局项默认尺寸类型不可靠, 每个项都应显式 SetWidth/Height(Fill/Fixed);
    - 内部标签按创建时 cx 等分的自绘组件(如 Tabs)不能被布局拉伸, 用 Fixed 宽;Slider 这类元素比"可见宽度"外扩的组件, Fixed 宽 = 传入 cx + 2×外扩量。
30. **ComboBox 原生弹层的样式接管路线**(实测踩坑, combobox.go 依赖):弹层 = 独立弹出窗口 + ListBox(本质 ListView), 只能拿到句柄做"半接管"——
    - **窗口 BkM 对这个弹层无效**: `XWnd_GetBkManager` 拿到的管理器加填充/描边完全不绘制(绿色探针实证), 弹层白底另有来源;
    - **`ClearBkInfo` 后元素回退"系统默认内容"(白底方角)**: 想要透明必须显式垫一层 alpha=0 的填充(`AddFill(state, 0, 0)`), 算"有内容"才不会回退;
    - **圆角用 Win32 区域裁剪**: `wapi/wnd.SetWindowRound(hwnd, 物理像素半径)`(SetWindowRgn)稳;xcgui 的 `SetTransparentType(Shaped)` 对原生弹层实测无效;元素背景对象 `SetRectRoundAngle` 写入成功但圆角外仍是窗口白底, 两者要配合用;
    - **描边画在 ListBox 的 paint 事件**(bk 之后、条目之前): 元素背景的 border 对象是方角, 圆角处会被窗口区域裁掉;
    - **条目高亮/勾用 `AddEvent_ListBox_DrawItem`**(回调带 Index/NState/RcItem): 原生在弹层打开时**不给选中项置 Select 态**, 要自己拿 `XComboBox_GetSelItem` 比对(复刻 shadcn 打开时选中项自动高亮);文字是项模板的 shapeText 子元素, 在项背景之后绘制不受影响;
    - **`SetDropHeight` 必须在弹层打开前设置才生效**(构造时), 弹出事件里设置无效(实测 set/get 均为新值但窗口不变);固定模式(`EnableDropHeightFixed(true)`)才使用该值;
    - **触发区滚轮**: 原生会滚动 ComboBox 内部文本(文字上移), `EnableEvent_XE_MOUSEWHEEL(false)` 屏蔽(事件交给父元素), 弹层 ListBox 独立不受影响;
    - **弹层退出后窗口句柄即失效**: 关闭后用旧句柄调任何 XWnd_* 会弹"句柄无效"错误框(实测 `XWnd_GetDPI`), 必须挂 `AddEvent_Combobox_Exit_List` 置空缓存句柄;
    - **弹层窗口/ListBox 每次弹出都是新建的**(退出即销毁, Exit_List 后窗口句柄即失效), 事件在每次 Popup_List 里重新注册即可, 不会重复;全局的 OnThemeChange 回调才需要 once 守卫。

---

## 九、本设计依赖的已验证 API 清单(xcgui 1.4.2)

- `Element.AddEvent_Paint / AddEvent_MouseStay / AddEvent_MouseLeave`(widget/element.go:1192, 1288, 1336)
- `Element.AddEvent_Menu_Draw_Background / AddEvent_Menu_DrawItem`(widget/element.go:2056, 2080)
- `drawx.NewByHandle / EnableSmoothingMode / SetBrushColor / FillRoundRect(Ex) / DrawRoundRect(Ex) / GradientFill2/4 / DrawText / TextOut* / Image* / DrawSvg* / SetClipRect / SetLineWidth`
- `xc.XBtn_GetStateEx`、`xcc.Button_State_*` 状态
- `xc.RGBA / xc.ParseRGBA / xc.HexRGB2RGBA`(Web 十六进制色直转)
- `svg.NewByString / SetUserFillColor / SetUserStrokeColor / SetSize`
- `ani.NewAnima / AnimaGroup`、`ease` 缓动
- `XWnd_Redraw / XWnd_GetDPI`、`XC_EnableAutoDPI / XC_EnableDPI`
- LayoutEle:`EnableLayout / EnableHorizon / SetAlignH(V) / SetSpace / SetSpaceRow / EnableAutoWrap / SetPadding`、`xc.XWidget_LayoutItem_SetWidth/Height/SetMargin`(语义见第八节 29;Go 包装无 AdjustLayout)
- `SliderBar.SetRange/SetPos/SetButtonWidth/SetButtonHeight/EnableHorizon/GetButtonObj`、`ProgressBar.SetRange/SetPos/EnableShowText/EnableStretch`、`TabBar.AddLabel/SetSelect/SetLabelWidth`、`ComboBox.CreateAdapter/AddItemText/SetSelItem/EnableEdit/EnableDrawButton/SetDefaultText/SetDropHeight/GetButtonRect`
- `XWnd_SetWindowPos(物理像素)`、`XC_hWindowFromHWnd`、`XEle_GetWndClientRect`、`XWnd_GetClientRect`、`wapi.ShowWindow/SetForegroundWindow/GetSystemMetrics/GetWindowLongPtrW`
- `app.GetTextShowRect`(文本测量:限宽换行与绘制共用限宽)
- 窗口真透明:`Window.SetTransparentType(xcc.Window_Transparent_Shaped)` + `Window.SetTransparentAlpha(255)`、`xcc.Window_Transparent_Shaped`(xcconst.go:55)
- 背景对象美化:`BkManager.AddFill/AddBorder(state, color, id)`、`BkManager.GetObjectObj(id) *bkobj.BkObj`、`BkObj.SetRectRoundAngle/SetMargin/SetAlign/SetRotate/SetFillColor/SetBorderColor`、`xcc.BkObject_Align_Flag_*`(bkmanager.go:69/185, bkobj.go:104/42/50/66/74/90)
- ⚠️ `TabBar.SetPadding` 在 xcgui.dll 4.0.0.0 中不存在(调用 panic)
- 官方自绘示例:BeautifyButton / CheckBox / RadioButton / Edit / ProgressBar / SliderBar
- 官方透明/异型窗口示例:`Advanced/WindowPartiallyTransparent`、`webview/WebViewPartiallyTransparent`
