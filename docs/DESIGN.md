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
| Dialog | 纯 Element 覆盖层 | 全窗遮罩元素(自绘黑 10% + 面板) + 面板子元素;ESC/点遮罩关闭, Tab 焦点循环(见第八节 33) | — |
| Menu | 原生菜单 | `XMenu_EnableDrawBackground/EnableDrawItem` + `AddEvent_Menu_Draw_Background/DrawItem` | element.go 已确认事件存在 |
| Card/Badge/Separator | 纯 Element | paint | — |
| Tooltip | FloatWindow | paint | — |
| Table/List | 原生 List + 项模板 | 表头/行底/边框/滚动条全自绘(绘制回调按**单元格**触发, 见第八节 40); 原生 Table(Shape 系)与 shadcn 语义不符, 不单独做 | Basic/List 示例 + 本仓库探针 |
| Tree | 原生 + 项模板 | 模板创建事件里皮肤化 + EnableTemplateReuse | 库文档 |
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
| **P2** | Switch、Slider、Progress、Tabs、ComboBox、Tooltip、图标系统 + 暗色切换演示 | 同上 | ✅ |
| **P3** | Dialog ✅、Menu 皮肤 ✅、Table/List 皮肤 ✅、Tree 皮肤、hover 过渡动画、CLI 完整化(update) | 同上 | 🟡 |
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
17. **元素 paint 回调里能画位图, 也能画 SVG;当年"画不出"的根因是 Lucide 的 path 写法 xcgui 的 SVG 解析器不支持**(2026-10-07 复核修正, 此前本条结论有误导):
    - **位图完全能画**:`XDraw_Image`(1:1)/ `XDraw_ImageEx`(拉伸)在元素 paint 回调里输出正常(复核探针实证;当年"句柄有效、坐标与裁剪都正确,就是画不出来"的说法不成立);
    - **SVG 也能画, 但渲染结果不一定对**:`svg.NewByString` 加载合法 SVG 后,`DrawSvgEx / DrawSvg / DrawSvgSize` 三条 API 在元素 paint 里都正常出图(建议先 `SetSize/SetPosition` 再画);但**能画 ≠ 画得对** —— lucide 的归一化 path(绝对坐标、空格分隔)虽能解析成功, 渲染却走样: 圆弧变形、`fill="none"` 的子路径被糊成填充块(svgprobe 截图实证, 2026-10-07);
    - **Lucide 原始 path 是唯一画不出的**:`svg.NewByString(lucide path)` 直接返回 **nil**(解析失败, 句柄都拿不到), 喂给绘制 API 自然无输出;`imagex.NewBySvgString` 同理。当年把"SVG 解析失败"误记成了"paint 通道画不出图片"。
    - 因此需要图标时,用线段图元复刻:`icons.go` 把 lucide 的 path 解析成折线再用 `DrawLine` 描边(支持 M/L/H/V/C/S/Q/T/Z,圆弧用专门实现);或把图片交给原生控件(`Button.SetIcon` 等)。**2026-10-06 起图标的出图改为直通 D2D 几何填充(解析出的折线仍作为回落路径保留), 抗锯齿与几何精度见第 43 条。**
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
    - **描边不能画在 ListBox 的 paint 事件——该事件在这个弹层窗口上不渲染输出**(探针实证: 内缩 10px 的 6px 红色粗环都画不出来); 这个窗口上能出图的只有 BkM 背景对象、条目 `DrawItem` 回调和 `XE_PAINT_END`;
    - **BkM 的 border 对象贴边画, 会被 SetWindowRgn 的窗口区域裁得只剩左边一条**(上/右/底全消失, 只留左缘; 区域与内容为何错位未深究)。ring-1 描边的出路: 先 `XEle_EnableEvent_XE_PAINT_END` 再 `AddEvent_Paint_End`(该元素及子元素绘制完成事件, 在条目之后执行 = 边框盖内容边缘, 同 shadcn), 描边整体**内缩 1px** 落在窗口区域内, 四边完整, 圆角弹层上 1px 白边距不可感知;
    - **条目高亮/勾用 `AddEvent_ListBox_DrawItem`**(回调带 Index/NState/RcItem): 原生在弹层打开时**不给选中项置 Select 态**, 要自己拿 `XComboBox_GetSelItem` 比对(复刻 shadcn 打开时选中项自动高亮);文字是项模板的 shapeText 子元素, 在项背景之后绘制不受影响;
    - **`SetDropHeight` 必须在弹层打开前设置才生效**(构造时), 弹出事件里设置无效(实测 set/get 均为新值但窗口不变);固定模式(`EnableDropHeightFixed(true)`)才使用该值;
    - **触发区滚轮**: 原生会滚动 ComboBox 内部文本(文字上移), `EnableEvent_XE_MOUSEWHEEL(false)` 屏蔽(事件交给父元素), 弹层 ListBox 独立不受影响;
    - **弹层退出后窗口句柄即失效**: 关闭后用旧句柄调任何 XWnd_* 会弹"句柄无效"错误框(实测 `XWnd_GetDPI`), 必须挂 `AddEvent_Combobox_Exit_List` 置空缓存句柄;
    - **弹层窗口/ListBox 每次弹出都是新建的**(退出即销毁, Exit_List 后窗口句柄即失效), 事件在每次 Popup_List 里重新注册即可, 不会重复;全局的 OnThemeChange 回调才需要 once 守卫。
    - **滚动条残影**: 原生的滚动条显隐判定在打开时用旧条目高算好, 调矮条目后不会自己收起; 用滚动视级开关 `ShowSBarV/ShowSBarH` 显式控制(条目级 `ScrollBar.Show` 会被原生布局覆盖)。另: 弹层高度恰好等于内容高度时视口差 1-2px 也会出滚动条, 留 +4 余量。
    - **弹层默认定位与 item-aligned 重定位**(探针实测): 原生把弹层放在触发区**正下方**(顶=触发区底、左=触发区左、宽=触发区宽)。radix Select 的默认定位是 item-aligned(shadcn select.tsx 的 SelectContent), 即**弹层盖住触发元素、选中项正对触发位置**——用 `wapi.SetWindowPos`(物理像素, `SWP_NOSIZE|NOZORDER|NOACTIVATE`)在 Popup_List 事件里重定位: 弹层顶 = 触发区顶 − (p-1 边距 4 + 选中索引×条目高), 贴屏幕边时向下压并用 `ScrollView.ScrollPosYV` 滚动让选中项仍可见。坐标链路: `XEle_GetWndClientRect`(逻辑, **原点=所属窗口 rect 原点, 含阴影边距**——实测含阴影窗口也成立) + 所属窗口 `XWidget_GetHWINDOW` → `wapi.GetWindowRect`(物理) → 逻辑差 ×DPI/96;
31. **xcgui 字体的 size 单位是 pt(磅), 不是像素**(重大全局坑, 实测确认): `font.NewEX(name, size, style)` 文档写明"size: 字体大小, 单位(pt, 磅)", 渲染尺寸 = size×DPI/72 物理像素。此前 FontHandle 把主题的 14(逻辑 px) 直接当 pt 传入, 144 DPI 下渲染出 28 物理像素 em = **18.7 逻辑像素**, 全组件文字放大了 1.33×(墨迹实测 26-27 物理像素, 与 14pt 完全吻合)。修法: **pt = px×72/96(= ×0.75, 与 DPI 无关)**, 在 FontHandle 里统一换算(14px→11pt=14.67px, pt 取整导致 14 这类 ×0.75=10.5 的值有 ±0.67px 误差; 12/13/16px 恰好或接近取整)。排查入口: 同一截图里对比组件文字与已知 14px 参考的墨迹高度。
32. 用 pnpm 添加 shadcn 组件会报错, 应该用npx。
33. **Dialog 用"窗口内覆盖元素"实现**(探针 .verify/probe_dialog + demo 实测, dialog.go 依赖):
    - **窗口子元素坐标原点 = 窗口左上角(含标题栏), 不是客户区原点**:`XWnd_GetClientRect` 返回的 `{8,8,624,412}` 只是窗口边框内缩(与标题栏无关), 用它给元素定位会整体偏移;遮罩尺寸直接取 `XWnd_GetRect` 的宽高放 (0,0) 即可铺满整窗(实测遮罩把标题栏也压暗, 窗口自绘的标题文字仍在上层保持清晰);
    - **自绘 Element 覆盖全窗天然是模态**:它挡住下层所有鼠标事件(实测覆盖后下层按钮既收不到点击也收不到 hover), 配合 `XEle_SetZOrderEx(h, 0, xcc.Zorder_Top)` 保证在最顶层(每次 Show 都抬一次, 因为 demo 里元素创建顺序可能比对话框晚);
    - **半透明合成精确实测**:遮罩填 `RGBA(0,0,0,26)`(bg-black/10)后, 白底 255→229、深色钮 14→59(悬浮态), 与 0.9 系数完全吻合, 无累积变暗;
    - **隐藏遮罩必须整窗重绘**:`XWnd_Redraw(hWnd, true)`;局部重绘会留下变暗残影(半透明像素已经"烙"进帧缓冲);
    - **元素收不到键盘事件, 键盘统一走窗口 `AddEvent_KeyDown`**:探针里遮罩元素注册的 KeyDown 从不触发;窗口级回调在"编辑框聚焦"时也照常收到 ESC(27)/Tab(9) → 对话框的 ESC 关闭与 Tab 循环都在窗口回调里做, 用 `wapi.GetAsyncKeyState(VK_SHIFT)<0` 判 Shift+Tab;
    - **打开时先 Show 再 layout**:内容高度量的是"可见子元素"的占位(`XWidget_IsShow` 过滤), 隐藏状态下量出来是 0(demo 第一次跑面板只有标题+页脚就是这个原因);控件自建后默认可见, 所以 NewDialog 末尾要用递归 `XWidget_Show(false)` 隐藏整棵子树(父元素隐藏不会连带隐藏子元素);
    - **`XWnd_SetFocusEle` 搬焦点后不会重绘**:按钮上的焦点框不出现(输入框因为要重画光标/IME 才碰巧刷新), 必须手动 `XEle_Redraw(旧)` + `XEle_Redraw(新)`;且顺序要在搬焦点**之前**调 `xui.NoteKeyboard()`——AttachWindow 的 Tab 钩子注册得晚(回调后执行), 等它标记"键盘焦点"时元素早已重绘完;
    - **焦点反查要沿父链**:`XWnd_GetFocusEle` 拿到的是复合控件(Edit)的内部元素, 与创建时的句柄不同 → 用 `XWidget_GetParentEle` 往上找焦点序列里的成员, 找不到再用自维护的下标兜底;
    - **Tab 序列只收交互控件**:`XObj_GetType` 白名单(Button/Edit/ComboBox/SliderBar/Editor/RichEdit/ListBox/TabBar)+ `IsEnableFocus && IsSwitchFocus`;Label 载体是原生 Button, 必须显式 `EnableSwitchFocus(false)` 才不会挤进 Tab 序列(顺带 `EnableDrawBorder(false)`, 否则它能画出原生边框);
    - **backdrop-blur-xs 后来补上了**(2026-10-07, 见第八节 47 的 backdrop-blur 小节): 当时尚以为画不出, 实际走"抓窗快照 + BkM 图片填充"能精确实现。
    - **ring-1 要画在面板矩形外侧**(2026-10-07):tailwind 的 ring 是 box-shadow, 画在盒子**外面**压着遮罩灰底, 才形成可见的深色描边;xcgui 圆角矩形描边落在路径**内侧**, 直接描面板矩形会把 10% 黑压在面板白底边上(230), 与遮罩灰(229)几乎同值而完全不可见(像素实测)。修法 = 描边矩形外扩 1px(dialog.go onPaint)。

34. **Dialog/文字/量测的一批实测结论**(2026-10-04 视觉对齐期间, 探针+像素测量):
    - **自绘文字的字号就是逻辑像素, 不要按 DPI 反除**: 曾据"按钮文字顶到边缘"推断绘制路径二次乘 DPI 并做了字号反除, 实测反证(反除后全局文字明显偏小), 已回退并记录于此。真正的原因是量测口径;
    - **`app.GetTextShowRect` 在 144 DPI 下返回值约为实际绘制宽度的一半**: 对话框页脚用 "保存"(2 字)量出 14.7 逻辑宽, 实际绘制 29.3 → 按它算按钮宽会让文字顶到按钮边。现在量宽处 ×2 校正(改量测 API 或换 API 亦可, 但要重新验证);
    - **量宽与绘制必须同字体同限宽**: Tooltip 曾用 14px 主题字体量宽、按 12px(`text-xs`)绘制, 气泡两侧留白偏宽 17%;
    - **句柄有两级, 混用会弹模态错误框**: `XWidget_GetHWND(元素)` → Win32 HWND; `XWnd_*` 系列要的是 xcgui 窗口句柄 hWindow(用 `XC_hWindowFromHWnd` 转)。窗口句柄喂 `XWidget_GetHWND`、HWND 直接喂 `XWnd_GetDPI` 都会弹"输入句柄可能无效"错误框, 且**绘制回调里调这类校验型 API 风险高**(实测复现两次);
    - **容器元素被点击会聚焦并画原生框**: 普通 Element/内容容器点一下就获得焦点, 原生焦点框是**蓝色**(#58B1FC), 而它的**边框**在聚焦态是深色 —— 两个都要灭: `EnableFocus(false)` + `EnableDrawBorder(false)`。蓝框还曾压住输入框左右描边, 造成"输入框边框缺失/圆角怪"的错觉;
    - **元素内文字被元素矩形裁剪**: 按 CSS `leading-none` 把行盒压到字号高度时, 中文行高(≈1.32em)放不下, 字被切头 —— 文本元素要上下各外扩几个像素(布局高度仍按行盒算), 见 dialog.go `dlgTextOver`;
    - **页脚 `-mb-4` 的高度账**: 页脚贴面板底边时它自身高度要**全量**计入面板高(此前少算 16 逻辑像素, 面板整体矮一截);
    - **中文下 `font-medium`(500) ≈ 常规字重**: 微软雅黑只有 Regular/Bold 两档, 浏览器 500 会回落到 Regular。用粗体字形实现 500 会明显偏粗(实测两轮翻工)。标签看着比描述粗, 主因是**颜色**(`--foreground` 近黑 vs `--muted-foreground` 灰), 实测两者笔画宽度相同;
    - **Tooltip 定位不能叠加客户区偏移**: `XEle_GetWndClientRect` 返回的坐标原点已是"窗口 rect 原点(含阴影边距)", 再叠加 `XWnd_GetClientRect` 的内缩会让气泡整体下移/右移一个阴影宽度(实测 8 逻辑像素, 表现为箭头尖端扎进目标元素)。

35. **暗色下的"半透明令牌 + /NN 修饰"必须做 alpha 乘法**(2026-10-04 暗色验收, tabs/switch/radio 三处同源):
    - 暗色主题里 `input` 等令牌**自带 alpha**(`oklch(1 0 0 / 15%)`, 内置暗色主题写作 `RGBA(255,255,255,38)`); 此时 CSS 的 `bg-input/30` 语义是 15% × 30% = **4.5%**, 而 `disabled:opacity-50` 是 15% × 50% = 7.5%;
    - 我们原先用 `WithAlpha(v)` 直接**覆盖**了这层 alpha, 于是: tabs 选中态比参考亮一倍(#686868 vs #2f2f2f)、未选开关的轨道变成不透明白饼、禁用开关反而比正常态更亮(50% > 15%);
    - 修法: 新增 `Color.ScaleAlpha(frac)`(在现有 alpha 上乘), 全库凡是"半透明令牌再叠透明度"的地方都用它, 不要用 `WithAlpha`;
    - 相关规则(逐条对源码): tabs 暗色选中 = `bg-input/30` + `border-input`(**不是** `bg-background`, 否则暗色下比列表更黑); radio 未选 = `border-input` + 暗色额外 `bg-input/30` 淡填充, 禁用 = 指示器整体半透明(标签文字不跟随) —— 原先禁用画成 `Background@50%`, 暗色下就是一块黑盘; switch 未选轨道 `bg-input`/暗色 `bg-input/80`, 选中滑块 `primary-foreground`、未选 `foreground`。

36. **原生菜单皮肤的一批实测结论**(2026-10-05, components/menu.go):
    - 弹窗圆角: 与 ComboBox 弹层同路线 —— `Menu_PopupWnd` 事件里 `window.NewByHandle` + `wnd.SetWindowRound`(Win32 区域裁剪, 物理像素 = RadiusLg×DPI/96); shaped 透明对菜单弹窗同样无效;
    - 项几何(探针实测): `Menu_DrawItem` 的 `RcItem` 左右原生各留 3px 边距(贴着画条目高亮即可, 与 shadcn 的 p-1 4px 差 1px 不可感知), 高度 = `SetItemHeight`(28); **分隔栏槽高固定 4px**, 1px 线画在 `Top+2`;
    - `"	"` 快捷键: 原生把 `	` 后的文本算进自适应宽度并预留 `NShortcutKeyWidth`(实测 "Ctrl+C"=37) → 把 "名字	快捷键" 整串传给 AddItem, 自绘时拆开分别渲染就得到右对齐快捷键, 不用自己量宽;
    - `SetLeftWidth(0)` 去掉左侧图标条; Label 项借 `Menu_Item_Flag_Disable` 实现"不可高亮不可点"(自绘时对 kind==label 单独配色);
    - 事件拓扑: 窗口级 `XWM_MENU_DRAW_BACKGROUND/DRAWITEM` 对该窗口弹出的**所有**菜单生效(每次新建的菜单对象、各自的子菜单窗口都触发) → 皮肤每窗口只注册一次(map 守卫), 绘制实时取令牌, 换肤零成本; `Menu_PopupWnd` 对每个弹出窗口(含子菜单)各触发一次且顶级先于子菜单 → 圆角处理放这里全命中, 位置校正用 `HMenu == 自己的句柄` 只认顶级;
    - 弹窗定位: `Popup` 收**物理像素**; 元素锚点 = `XEle_GetWndClientRect`(逻辑, 原点=窗口矩形含阴影边距) + `wapi.GetWindowRect`(物理) 换算; **翻转/夹紧/子菜单定位全部原生自带, 任何手动位置校正都不做** —— 详见下一条;
    - ⚠️ **验收坑: 全屏缩略截图里暗色弹层窗口会显示成亮底**(截图管线对弹层窗口的合成/缩放问题), 区域放大看才是真实效果 —— 弹层类组件的暗色验收一律以 zoom/像素测量为准, 不要凭全屏缩略图下结论。
    - ⚠️ **别给弹出菜单手动校正位置**(2026-10-05 探针实测, 踩了两轮): `XMenu_Popup` 是**阻塞**的(菜单关闭后才返回), 且 `Menu_PopupWnd` 事件对每个弹窗触发一次。两条死路: ① 在 PopupWnd 回调里 `SetWindowPos` 移动顶级弹窗 → 子菜单锚点失效, 二级菜单弹到父菜单**原点**盖住父菜单; ② 等 `Popup()` 返回后再校正 → 弹窗已销毁, `XWnd_GetHWND` 报「输入句柄可能无效」。结论: 初始锚点给对(元素矩形换算), 其余(顶部放不下翻到上方、左右夹紧、子菜单贴父菜单右侧行对齐)原生全包, **不要动**。

37. **页面容器 Show(false)→Show(true) 后内容空白**(2026-10-05, demo 顶层 Tabs 分页):
    - 现象: 隐藏期间创建好的布局子树(LayoutEle 流式布局)在页面恢复显示后只有壳在渲染, 内部控件全空白;
    - 根因: xcgui 布局元素在父容器不可见期间不排版(与第八节 29 "布局项变更需手动 AdjustLayout" 同源), Show(true) 只改显隐属性, 不会触发子树重排;
    - 修法: 切页时对**新显示的页面容器**调 `AdjustLayout()`, 再 `Redraw`; 初始就可见的页面不需要。

38. **自绘标题栏的无框窗口组合与鼠标穿透**(2026-10-05, demo main):
    - 不要 Default(自带图标/标题/三键): 样式用 `Caption|Border|Drag_Window`(不含 Icon/Title/Btn_*/Allow_MaxWindow/Drag_Border), `SetBorderSize(0,30,0,0)` 保留 30px 标题栏高度, 阴影照旧 `Window_Transparent_Shadow + SetShadowInfo`; BkM 窗口背景绑定在此组合下正常工作(Window_Style_Nothing 才是"自身不绘制任何东西"的那个);
    - 图标/标题等**纯展示元素必须 `EnableMouseThrough(true)`**, 否则元素挡住 Drag_Window/Caption 的拖动命中(实测标题文字区域拖不动窗口, 穿透后整条标题栏可拖); 子元素不受穿透影响;
    - 控制按钮用普通自绘按钮(幽灵变体 + 图标)即可, 最小化 `w.ShowWindow(xcc.SW_MINIMIZE)`(窗口包没有 Minimize 包装), 关闭 `w.CloseWindow()`; 不需要 CaptionButton 示例那套"读原生最小化按钮坐标 + Paint_End 里重定位"的逻辑(那是给 Default 样式加按钮用的)。

39. **窗口布局(EnableLayout) + 阴影窗口 + 自绘标题栏的一批实测结论**(2026-10-05, demo 布局化重构):
    - **客户区已避开阴影带**: rect 800x590、阴影 8px 时 `XWnd_GetClientRect` = {8,8,792,582}(784x574), 子元素坐标就是可见区坐标, **不需要再设 padding 避让**; 但窗口 rect 比可见区大一圈, 按 rect 宽算坐标会超出可见区(实测按钮右缘超 12px);
    - **布局区 = 客户区 − border size − padding**: border 归零后布局区=整个客户区;
    - **布局项必须是 LayoutEle**: 普通 Element 设置 `LayoutItem_*` 也不被排版(探针实证 red 块不占位、尺寸保持 0 → 连带裁掉全部子元素, 表现为整块空白);
    - **主轴剩余空间用 Weight 不是 Fill**: 纵向布局里 Fill 高度 = 独占整列(与前面 Fixed 项重叠, 实测 content 574=全高盖住 cap 30); 横向行里 Fill 宽度会把后面的 Fixed 兄弟挤到下一行(按钮消失)。DESIGN 第八节 29 的"Fill 独占一行"就是指主轴; 交叉轴用 Fill 铺满没问题;
    - **绝对定位容器要 `EnableLayout(false)`**: LayoutEle 默认排版开着, 内容区里绝对定位的 tabs/pages 会被当布局项排走(实测 tabs 跑到窗口底部);
    - **双击最大化自己补**: border 归零后原生 caption 区域没了, 窗口级 `AddEvent_LButtonDBClick` 里 `IsMaxWindow() ? ShowWindow(SW_RESTORE) : MaxWindow()`(MaxWindow 只管最大化, 不会切换); 最大化/还原时窗口布局自动跟随, 按钮组(Weight 顶右)始终贴右上角;
    - 纯容器(cap 行/内容区/页面)一律 `EnableMouseThrough(true)` 不挡 Drag_Window 拖动, 子元素(按钮)不受影响。

40. **原生 List 皮肤接管路线**(2026-10-05, components/table.go): 载体是原生 `List`(表头/纵向滚动/行选中/项模板复用全部原生), 外观全自绘。逐条实测:
    - `List_Item_.RcItem` 的文档注释写着"未使用", **实际有效**(就是该单元格的逻辑矩形); 表头项 `List_Header_Item_.RcItem` 同样有效(表头项的 `HLayout` 恒为 0, 只能用 `RcItem`);
    - `XE_LIST_DRAWITEM`(AddEvent_List_DrawItem)按**单元格**触发(不是按行), 同一行各列的 `NState` 相同, 且各单元格矩形**首尾相接** → 逐格填充拼出整行底色、逐格画线拼出整行底边, 不需要自己算行矩形;
    - `RcItem` 是列表客户区的**逻辑坐标**(与 `XEle_GetRect(HLayout)` 一致), 行 0 的上边 = 表头高 + 2;
    - 原生 List 的内容区在**左上各内缩 2 逻辑像素**, 且**表头单元格从 x=0 起、数据单元格从 x=2 起**(表头整体比数据行靠左 2px); 行间距默认 **1px**。因此: 列宽按 `cx - 2*2 - (滚动条 12)` 分配, `SetRowSpace(0)` 去掉行间距, 表头形状文本缩进比单元格少 2 才能左对齐(实测列内文字与表头文字 x 完全一致);
    - `SetDrawRowBkFlags(Nothing)` 只关掉原生行背景/分割线**绘制**, **不影响**行悬浮 `List_Item_State_Stay` 的状态跟踪与重绘触发(实测鼠标移入行即触发重绘并上报 state=1);
    - 表体透明用 `EnableBkTransparent(true)`: 元素未绘制区域**真的透出宿主底色**(与 ComboBox 弹层的"白底回退"不同); 表头是独立子元素, 要另外 `GetHeaderHELE()` → `EnableBkTransparent` + `EnableDrawBorder/Focus(false)`, 否则表头保留默认灰底;
    - 行底色/行边框/表头底**全部在绘制回调里实时取令牌** → 换肤零成本(不需要 OnThemeChange); 只有"必须下发给原生对象"的东西才要重刷 —— 形状文本(单元格/表头)的字色字体、滚动条的背景对象 → 用一个 `applyTheme` 统一处理, 并挂 `xui.OnThemeChange`;
    - 滚动条只能走背景管理器(同第 21 条): `GetScrollBarVObj()` + `ShowButton(false)` + `SetSliderPadding(2)` + `EnableBkTransparent(true)` + ClearBkInfo/AddBkFill(Leave/Stay 用表面色, 滑块用 Border/MutedFore); 换肤时先 `ClearBkInfo` 再重加不会累积;
    - 纵向滚动条会占掉内容宽度: 自适应列(占满剩余宽度)要让出 ~12px, 否则最后一列被滚动条盖住; 且滚动条显隐切换时要重算列宽 → 按"是否需要滚动"翻转时 `SetColumnWidth` 重排;
    - 列宽: 原生没有权重/百分比列, 自适应列要自己算 `元素宽度 − 2×内缩 − (需要滚动条?12:0) − 固定列宽之和`;
    - 滚动条尺寸 `SetScrollBarSize(8)` + `SetSliderPadding(2)` → 滑块实际只有 4 逻辑像素宽, 观感与 shadcn 的细滚动条一致;
    - **竖排表格不能用原生 `widget.Table`**: 那是 Shape 系的"单元格画布"(合并单元格、逐格画线、无滚动/无虚拟行), 与 shadcn 的可滚动行列表语义不符 → 本组件的 `Table` 一律以原生 `List` 为载体。

41. **LayoutEle 纵向(列)布局也会"溢出换列"**(2026-10-05, demo 数据页踩坑): 列布局里主轴上剩余高度放不下下一个项时, 该项**不会被裁剪也不会重叠, 而是被甩到"下一列"** —— `x += 容器宽度, y = 0`, 用户看到的就是"控件整块消失"(与第八节 29 的"横向 Fill 独占整行把兄弟挤到下一行"是同一套换行机制的两个方向)。修法: 布局级 `EnableAutoWrap(false)`; 同时要保证容器高度够(本例 card 高 320 才能容下 24+14+24+14+180=256 的列)。`AdjustLayout()` 之前读到的 `XEle_GetRect` 是旧值(排版前), 排查这类问题要在排版之后取坐标。

42. **原生 Tree 自绘皮肤(2026-10-06, Tree 组件探针实测, 载体路线同 40)**:
    - **自定义项模板要在 `CreateAdapter()` 之前 `SetItemTemplateXMLFromString`**, `SetItemHeightDefault(n, n)` 也要在模板/适配器/数据**之前**调用——之后设置不生效(模板 `layout.height="fill"` 会吃掉模板里的 heightT, 行高实际由 SetItemHeightDefault 决定, 探针默认 20);
    - **展开按钮(itemID=1, expandButton=true)的 "+/-" 原生图形**: `ClearBkInfo()` 压不掉——背景清空会**回落到"系统默认内容"**(同窗口背景那条例)。必须 Clear 后再 `AddBkFill` 四态全透明色(透明填充算"有内容"就不会回落), 图形消失且**点击展开的原生行为保留**;
    - **展开按钮的点击热区矩形 = `RcItem.Left + 1 + 16×深度` 起、16×16**(开启 `app.ShowLayoutFrame` 后 PIL 实测布局边框红框标定, 每级步进恰好 16)。自绘 chevron 必须居中于这个矩形——画偏了(比如整体再右移半槽)图标就在热区右缘外, 点图标点不中展开; 模板实例的 `XEle_GetRect` 返回的是模板 rect 属性(相对容器), 参照系混乱, 标定要以布局边框为准;
    - 模板子元素绘制在 `XE_TREE_DRAWITEM` **之后**: 自绘回调里画行底/参考线/chevron 会被文字正常覆盖, 展开按钮透明填充不遮挡自绘 chevron;
    - `Tree_Item_` 有 NDepth/BExpand/NState(Leave/Stay/Select)/RcItem(全行宽, 左右各 1px 内缩); 项是 **NID 寻址**(非行号), 没有按行号枚举可视项的 API → 换肤同步文字色用 `RefreshData()` 重走模板创建事件, 不能像 List 那样按可视行范围逐个刷;
    - **模板里的 ShapeText(itemID=2)是 Shape 对象不是 Element**, 不能调 `XEle_GetClientRect` 等元素 API(弹"输入句柄可能无效"错误框), 只能 `XShapeText_*`;
    - 叶节点同样保留展开按钮槽位(每项都有 itemID=0/1 的模板实例), 文字全树自然对齐——正好满足 shadcn 树"叶节点留空槽位"的语义, chevron 只需对"有子项"的项绘制;
    - **隐藏页上创建的树, 启动时的 `SetSelectItem` 选中态会丢**(展开态 `ExpandItem` 却保留); 页面真正显示后再补一次 Select 即可(demo 在页签 OnChange 里做)。

43. **图标抗锯齿: 直通 D2D 几何填充(2026-10-06, `d2dicon.go` + `icons.go`, 探针 + 像素实测)**:

    - **炫彩把真的 COM 接口指针交出来了**: `XDraw_GetD2dRenderTarget(hDraw) → *ID2D1RenderTarget`、`XC_GetD2dFactory() → *ID2D1Factory`(另有 WIC/DWrite 工厂与 `XDraw_GetD2dBitmap`)。paint 回调里可以纯 Go 直通调用, **不需要 cgo、不需要汇编**, D2D 的 AA 也对直通调用生效(实测 AA 模式 = PER_PRIMITIVE)。
    - **ABI 三规则**(破任一条: 轻则整帧被毁、窗口隐形, 重则崩溃):① 形参里出现 `float32/float64` 的方法一律不能这样调 —— 浮点参数走 XMM 寄存器, `syscall.SyscallN` 传不进去; ② 按值传递的 8 字节结构体要打包成一个 64 位整数(`D2D1_POINT_2F` → `f32(x) | f32(y)<<32`); ③ 其余全用指针/整数传是安全的。
    - **vtable 索引**(实测交叉验证过, 错一位就是调到邻居方法; 曾把 30 当 GetAntialiasMode 实为 SetTransform, 用垃圾指针设了变换 → 整窗不再呈现):`ID2D1RenderTarget` IUnknown 0-2 + `ID2D1Resource::GetFactory` 3, 方法从 4 起 —— `CreateSolidColorBrush=8`、`FillGeometry=23`(参数 (geometry, brush, layer=NULL), 是本方案的唯一出图口)、`SetTransform=30`、`GetAntialiasMode=33`; `ID2D1Factory` 直接继承 IUnknown —— `CreateEllipseGeometry=7`、`CreatePathGeometry=10`; `ID2D1PathGeometry` 继承 `ID2D1Geometry` 的 13 个方法 —— `GetBounds=4`、`Open=17`; `ID2D1GeometrySink` 的 `ID2D1SimplifiedGeometrySink` 部分自有方法只有 3~9 —— `SetFillMode=3`、`BeginFigure=5`、`AddLines=6`、`EndFigure=8`、`Close=9`。
    - **`ID2D1SimplifiedGeometrySink` 里只有 `Close` 返回 HRESULT, `SetFillMode/BeginFigure/AddLines/EndFigure` 全是 `STDMETHOD_(void, ...)`**(同理 `ID2D1RenderTarget` 的 Draw/Fill 全家也是 void):它们的"返回值"是寄存器残留垃圾。曾照着"每个 hr 都校验"的直觉检查 `EndFigure`, 它返回 0x1f80(非 0)被当成失败 → **每次都静默回落折线渲染**:症状是"代码改了, 画面一模一样", 单看输出根本发现不了 —— 探针里必须把每一步的 hr 打出来看。
    - **直通调用拿不到"当前元素"的变换与裁剪**(本项目最大的坑):炫彩是在**刷新它自己的绘制命令**(`XDraw_*`)时才施加元素偏移/裁剪的, 直通 D2D 调用发生在回调里、命令队列之前, 因此拿到的是**父级(窗口)坐标系**。实测:元素内 (50,50)-(150,150) 的图标被画到窗口左上角; 反过来"先 XDraw 填充、后画图标"的按钮**看起来正常**(先有填充说明队列已推进), 而幽灵态图标按钮(背景/边框都是 None, 前面没有任何 XDraw 指令)的图标全跑到窗口左上角、正好被自绘的窗口图标盖住 → 表现为"标题栏按钮的图标消失"。
      **正解: 用官方提供的 `XDraw_ConvRect(hDraw, *RECT)` / `XDraw_ConvXY`** —— "从当前 UI 对象客户区坐标转换到画布对齐坐标, 自动适应 DPI", 本来就是给第三方绘图接口用的。换算结果就是**物理像素**(实测 150% DPI:元素在窗口 (60,60), 客户端 (50,50,150,150) → (165,165,315,315)), 直接喂给 D2D 即可, 尺寸与线宽随之自动变成物理像素, 不要再乘一次 DPI。
    - **WINDING 填充模式下所有子路径必须同向**:描边展开是"每段一个四边形 + 每个顶点一个圆帽", 而段四边形与圆帽的天然绕向相反, 重叠处 winding 归零被挖空 → 描边变成一排"梳齿"。修法:鞋带公式定号, 负号就反转点序(`sameOrientation`)。
    - **几何合法性自检**:索引错位/未 Close 的几何喂给 `FillGeometry` 会让 xcgui 整帧不呈现 —— **窗口彻底隐形(连边框都没有)、进程不崩、日志正常**, 只信截图;因此每次出图前用 `ID2D1Geometry::GetBounds`(idx 4;未 Close 的几何返回 `D2DERR_WRONG_STATE` 0x88990001)确认 hr=0 且 bounds 有限。
    - **为什么不用 `RT::DrawGeometry` 真描边**:它的线宽是 float 形参, 撞 ABI 规则①; 于是改在 CPU 侧把描边展开成填充多边形(段四边形 + 圆帽正多边形, 圆帽 12 边在 r=2px 时与真圆偏差 < 0.07px), 一次 `FillGeometry` 出图。展开是"内接近似"(每个形状都落在真·圆角描边内部), 不会比真描边胖。
    - **实测对比**(200px 图标同屏对照):新管线墨迹面积 10972 物理像素² vs 理论 10575(误差 4%), 旧折线 16064(胖 50%, 因为线宽被 `ceil` 取整、重叠段在 AA 下互相加深);X 图标边缘灰度层数 5 → 16。旧实现真正难看的原因是**折线点 `round32` 取整到逻辑像素**(18px 图标上 ±1.5 物理像素)+ 8 段离散的多边形棱角, 而不是"XDraw 没有 AA"——实测 XDraw 线条在 D2D 模式下也有 AA(两者像素统计几乎相同), 但几何是精确的。
    - **缓存**:几何是**设备无关**资源(归属 D2D 工厂而非渲染目标), 因此缓存按**工厂指针**作废(工厂变了=设备重建, 整体 Release 重建), 多个窗口(主窗/菜单/弹层各有各的 RT)共用一份几何;画刷是**设备相关**资源(必须由该 RT 创建、只能给该 RT 用), 所以每帧建一个、画完立刻 Release。绘制回调每帧都会新建 Painter(帧内对象), 缓存必须放**包级**。缓存 key 必须含 (path 原文, ox, oy, size, strokeW) —— 换算后的坐标是物理像素且随元素位置变化, 漏掉位置就会把图标画到别处。
    - **回落路径**:`app.New(false)`(GDI 渲染)时 `XDraw_GetD2dRenderTarget` 返回 0 → 自动回落到旧的折线描边(图标仍可见, 只是没有新管线干净);GDI 模式实测不崩、布局正常。

    **填充几何的 figure 数必须少(2026-10-06 图标总览页卡顿的根因, 重要)**:

    - D2D 填充**自重叠**路径走的是 stencil-and-cover:代价随**子路径(figure)数**近似平方增长, 与顶点数关系不大。探针实测(45 个图标同框、每图标同规模):9 个 figure/图标 → 4.1ms/帧;39 个 → 39ms;99 个 → 350ms;**199 个 → 1590ms**。
    - 最初的"描边展开"写法是**每段一个四边形 + 每个顶点一个圆帽**(一个图标上百个 figure), 于是 45 个图标一帧要 **719ms** —— 表现为 demo 图标总览页每动一下鼠标就掉帧("非常的卡"), 切页/换肤这种整窗重绘直接卡住近 1 秒。
    - 修法: 展开成**每条子路径一条闭合轮廓**(`strokeOutline`)。构造:两侧各按 w/2 偏移, 顶点处**外侧补圆弧**(round join)、**内侧取两条偏移线的交点**(精确, 不是直连); 两端各补一个半圆帽。改完实测 **2130 个图标平均 4.11 条轮廓/图标(最多 20 条, `virus` 尖刺多)**, 45 个图标一帧 **4.14ms**(= 空窗口基线, 图标开销归零)。`icons_test.go` 的 `TestIconGeometryFigureCount` 守住这条红线(>32 条就报错)。
    - **轮廓绕向必须一致**:所有轮廓的绕向由构造方式天然决定(与折线自身绕向无关), 因此 WINDING 填充下并集即笔画形状; 跨子路径的重叠(X 的两笔、太阳的辐条)也会正确并起来。填充子路径(lucide 的色点)则用 `sameOrientation` 归一。绕向不一致会让重叠区 winding 归零, 描边裂成一片片(`TestStrokeOutline` 守住)。
    - **此路不通的记录**: 本可以用 `ID2D1Geometry::CombineWithGeometry(UNION)` 一次性把图元汤合并 —— 索引按 Windows SDK `d2d1.h` 的声明顺序是 11(`GetBounds=4`、`CompareWithGeometry=8` 都已逐个验证), 参数全是枚举/指针(可 syscall), 但**实测调用直接崩(访问违例)**, 传不传单位矩阵都一样。顺带发现 `ComputeArea(13)`/`ComputeLength(14)` 的首参 `worldTransform` 是 `_In_`(**不能传 NULL**, 传了就崩), 而 `GetBounds(4)` 的 `transform` 是 `_In_opt_`(可 NULL)—— 本文件因此只用 `GetBounds` 一个几何方法。
    - **demo 侧同样要省重绘**: 格子元素的 `MouseStay` 在"鼠标在元素内移动"时**每帧都会来一发**, 不判状态就会变成每个鼠标移动事件触发一次整窗重绘; 只在 hover 状态**变化**时 `Redraw` 即可(图标页 45 个格子扫一遍 CPU 从 0.5s/3s 降到 0.17s/3s)。

44. **lucide 图标资源预处理(2026-10-06, `scripts/gen_icons.py` → `icons/` 子包)**:

    - **数据源**:`npm pack lucide-static@<ver>`, 版本与参考实现的 lucide-react 对齐(当前资源是 **1.52.0**)。
      `package/icons/*.svg` 共 **2130** 个(正式图标 + lucide 保留的废弃别名),
      全部统一 `viewBox="0 0 24 24"`、`stroke-width=2`、round cap/join ✓ —— 这是本渲染管线的前提。
      生成器**默认取最新版**(`npm view lucide-static version`), 要跟参考实现严格对齐时用
      `--version 1.49.0` 钉版; 版本号写进生成物(`icons.Version`)与文件头。
    - **为什么要预处理**:运行时只吃一条 path 数据, 而 SVG 里还有 `circle/ellipse/rect/line/polyline/polygon`
      六种图元(合计约 1100 个元素), 必须先把它们转成等价 path 指令;数字也要归一, 免得运行时反复解析长文本。
    - **相对指令必须整条绝对化**:同一个 SVG 里**每个图元都是独立路径**(各自从 (0,0) 起), 直接拼成一条数据后,
      后续图元开头的 `m`/`l` 会以前一个图元的终点为基准 → 原点错乱(实测 sun 的八条辐条整片跑偏)。
      生成器把每个图元改写成"全绝对 + 显式命令 + 隐式重复组展开", 拼顺序/拼位置就都无影响了。
    - **数字归一必须保住分隔符**:只在"命令字母后"与"负数前"省空格;非负数字相邻必须留空格 ——
      **arc 的 large-arc/sweep 是单字符标志**, lucide 写成 `a.25.25 0 00.407.195`
      (= flag 0 / flag 0 / x .407 / y .195)。按"数字"贪婪切会读成 `00.407` 一个数 → 参数不足 → 后面整段图形
      被静默丢弃(album/settings 等一大批图标命中, 症状是"图标只画出一半")。运行时解析器同步修了两处:
      `num()` 遇到第二个小数点就停, 新增 `flag()` 逐字符取标志位(现在贴 lucide 原始 path 也不会再断)。
    - **`fill="currentColor"` 图元要输出"填充 + 描边"两条子路径**:lucide 的这类元素(11 个图标里的色点)
      **同时继承根节点的描边**, 浏览器里是两者的并集 —— `circle r=.5` 实际画成半径 1.5 的实心点。
      像素实测(64px 图标):只填充 = 4x4 物理像素/16 像素墨量, 参考 = 12x12/120 像素。生成器用 `!` 前缀标填充,
      并对这类图元额外输出一条普通描边子路径, 与 lucide-react 渲染一致。
    - **放独立子包**(`github.com/twgh/xui/icons`):数据 ~360KB, 且**每图标一个 `IconXxx` 常量**。
      实测(2026-10-06, 同一基准比 `go build -ldflags="-s -w"` 的产物):只用常量 `icons.IconEye`
      是 **+0 字节**(二进制里 2130 条数据只进了 1 条), 调用 `icons.Path(name)`/`Has`/`Names` 会引用
      整张表 **+426KB**(1999360 → 2425856)。所以组件与业务代码都直接用常量, `Path` 留给
      "名字来自用户输入/配置"的场合。API: `Path/Has/Names/Count/Version` + 2130 个常量。
    - **常量必须由生成器产出, 不能手抄**:上一版把手写的 11 个图标常量放进运行时包(编译期内联,
      零成本), 代价是"手抄会漂移" —— 上线当天那条几何一致性测试就抓到 `eye`/`moon` 是旧版图形。
      现在常量与名字表由 `gen_icons.py` 一次产出(表里存的就是常量本身, `{"chevron-down", IconChevronDown}`),
      漂移在结构上不可能发生, 那条守卫测试与 11 个手抄常量一起删掉了。
    - **名字映射必须单射**:lucide 里存在成对的"只差一个连字符"的名字 —— `arrow-down-0-1` 与
      `arrow-down-01`、`axis-3-d` 与 `axis-3d`、`grid-2-x-2` 与 `grid-2x2`(前者是后者的废弃别名,
      数据完全相同), 直接驼峰化会撞成同一个常量名。规则:逐段首字母大写, **"数字段紧跟数字段"
      处补下划线**(`IconArrowDown0_1` / `IconArrowDown01`), 其余边界靠大小写区分(`IconAxis3D` /
      `IconAxis3d`);生成器另有冲突检查(撞了就报错退出, 不静默覆盖), 抽查用例钉在 `SPOT` 里。
    - **资源完整性用测试守**:`icons_test.go` 遍历全部 2130 个图标, 断言解析器能**完整**吃下每条数据
      (`parseIconPathEx` 的 complete 标志, 正是上面那类静默截断的探针)、点落在 24 网格内、填充标记存在。
      预处理回归时立刻炸, 不用靠肉眼比对整本图标。
    - **对照验收**:同屏对照浏览器渲染的原始 SVG(headless Chrome + `--force-device-scale-factor=1.5`
      对齐 150% DPI), 32 个覆盖全部图元类型的图标逐格像素差 ≤ 4.2/255, 墨量比 0.95~0.97
      (残差来自我们 `#0a0a0a` 与参考纯黑的色差);1.52.0 新增的 9 个图标单独复验 ≤ 4.7/255。
    - **demo 的「图标」页签**是这份资源的浏览入口(9x5 格 + 翻页), 也是"图标怎么用"的活文档:
      格子是普通自绘元素(各自 hover bg-accent), 翻页只换格子里的名字 + 重绘, 不销毁元素;
      **格子必须自己铺宿主底色**(透明子元素重绘不擦底, 见第八节 28), 容器元素要
      `EnableBkTransparent(true)` —— 否则暗色主题下容器是一整块白底、图标(浅色)直接看不见。
      45 个图标同框是**性能压力测试**: 它的卡顿逼出了第八节 43 那条"填充 figure 数必须少"
      (719ms/帧 → 4.14ms/帧); 另注意 hover 只在状态变化时重绘。

45. **hover 过渡动画(2026-10-06, 运行时 `anim.go`)**:

    语义对齐 tailwind / shadcn 的 `transition-*` 默认值: **150ms + cubic-bezier(0.4, 0, 0.2, 1)**
    (button/tabs/switch/badge 是 `transition-all`, table 行 / select / checkbox 是 `transition-colors`)。

    - **驱动用炫彩定时器, 不用 ani 包**:`XEle_SetXCTimer`/`XWnd_SetXCTimer` + `AddEvent_XC_Timer`
      (走事件总线, 不占 2000 回调上限)。它挂在**重绘目标**(元素或窗口)上, 只有还有没走完的过渡时才开,
      全部到位立刻 `KillXCTimer` —— **静止时零开销**。定时器 ID 保留 `0x7875`(用户自用炫彩定时器时避开)。
    - **实测帧率(探针)**:请求 16ms / 8ms / 4ms / 1ms 的真实回调都是 **约 30ms/帧**(xcgui 内部有帧间隔),
      所以请求 16ms 即可;150ms 过渡约 **5 帧**, 颜色单帧跨度小, 观感够平滑。
    - **必须是立即重绘**:`XEle_Redraw(hEle, true)`。走延迟重绘(`false`)时重绘会被合并, 实测掉到
      **约 65ms/帧**(同样 150ms 只剩 2~3 帧, 明显阶梯)。
    - **状态是"观察式"的**:组件不订阅鼠标事件, 而是在绘制回调里把**已经读到**的状态喂进来 ——
      `hv := xui.Hover(hEle, st.Hover)`(元素级, 原生 `Element_State_Flag_Stay` 或自跟踪旗标)、
      `hv := xui.HoverItem(hOwner, idx, stay)`(逐项, 取原生 List/Tree/ListBox 的 DrawItem `NState`,
      或菜单的 `MenuItem_State_Flag_Stay`)。逐项键是 **索引 + 1**(键 0 留给元素级)。
      好处: 组件不需要额外事件/字段, 载体换实现也不影响; 大表格只对"被悬停过的行"建状态,
      淡出到位即回收条目, 不随滚动累积。
    - **中断语义同 CSS**:目标反向时从**当前插值位置**续跑, 时长仍是完整的 150ms(不是剩余时间)。
    - **取色用 `Fade`(预乘 alpha 插值)**:`None → Muted` 这种"从全透明淡入"若按普通线性插值,
      中途会经过"半透明的黑灰"(None 的 RGB 是 0)而显得脏; 预乘后是干净的透明度渐变。
      只是在**不透明底色上叠一层**(表格行 `bg-muted/50`、菜单 hot 底、树行 accent)时,
      直接 `色.ScaleAlpha(hv)` 即可 —— 浏览器也是这么叠的。
    - **坑(务必记住)**:过渡值在悬停后的**第一帧恰好是 0**, 高亮完全靠驱动定时器补帧 ——
      也就是说**载体必须能被定时器驱动**(元素或窗口句柄可用)。若某个弹层的窗口不接受
      `XWnd_Redraw`, 症状就是"高亮永远不出现"或"停在半透明", 而不是"跳变"。
    - **验证(探针 + 真实鼠标 + PIL 像素实测, 放慢到 4s 便于抓中间帧)**:primary 按钮
      `#171717` → 28 → 49 → 62 → 68 → **69**(= `primary/80` 叠白底, 与改动前的悬停色完全一致);
      表格行 255 → 254 → 251 → **250**(= `muted/50` 叠白); 树行 255 → 246 → **245**(= accent);
      菜单条目 255 → **245**(悬停态, 弹层窗口的定时器 + `XWnd_Redraw` 这条路也验过);
      淡出方向同样连续、无残留。缓动曲线由 `anim_test.go` 对照独立实现钉住。

46. **重绘节流 `XC_SetPaintFrequency`(2026-10-06 实测; 顺带记一条空操作 API)**:

    - **语义**:`source/xcgui/xc/globalapi.go:313` "设置 UI 的最小重绘频率, nMilliseconds = 重绘最小时间间隔(ms)"
      —— 它是**节流窗口**(限制两次重绘之间至少隔多久), 不是"强制刷新"。xcgui 自带动画示例全都在动画场景里放开它:
      `xcgui-example/Advanced/Animation/Animation.go:93`、`Ease_All.go:33`、`Ease_Easy.go:20` 都是 `SetPaintFrequency(10)`;
      `HUMUI/HouTai017/HouTai017.go:108-109` 是 `SetPaintFrequency(10)` + `defer SetPaintFrequency(30)` → 反推**默认 30ms**。
    - **实测**(探针: 元素挂 1ms 炫彩定时器 + `XEle_Redraw(hEle, true)`, 与运行时 hover 过渡同一条路; 负载 = 300x40 元素满速重绘):

      | 设置 | 帧间隔均值 | 帧率 | UI 线程占用 |
      |---|---|---|---|
      | 默认(不调用) | 32.40ms | 30.9 fps | 5.5% |
      | **10ms** | **16.96ms** | **59.0 fps** | 12.7% |
      | 4ms | 16.57ms | 60.4 fps | 10.8% |
      | 1ms | 16.95ms | 59.0 fps | 10.9% |
      | 30ms | 32.54ms | 30.7 fps | 6.2% |

      结论: **默认就是 30ms**(第八节 45 说的"定时器请求 1/4/16ms 都只有约 30ms/帧"就是它造成的, 不是炫彩定时器的锅);
      设 10ms 帧率翻倍到 ~60fps; 再往下(4/1ms)毫无提升, 说明有 **~16.6ms 的硬地板**(60Hz/VSync)。
    - **用法**:应用初始化 `a.SetPaintFrequency(10)`(demo 已加)。**为什么是 10 不是 16**: 节流按"两次绘制的间隔"算,
      16 与运行时 16ms 的过渡定时器会周期性 beat(某次提前 0.1ms 就被推到下一帧 → 偶发 2 倍帧间隔), 10 留了裕量;
      16 以下也不再涨, 所以 10 是"够用且稳"的最低值。
    - **代价**:节流只"允许更早重绘", 不会凭空多画(静止时零影响); 单帧绘制成本不变(≈2ms), 所以帧率翻倍 ≈ 占用翻倍,
      且只发生在动画/过渡那 150ms 内。唯一要留意的是**整窗重绘很重的页面**(图标页 45 图标 ≈4.1ms/帧, 见第八节 43)
      在鼠标快速扫过时会多画几帧。
    - **顺带一条空操作 API(别用)**:`XC_SetTextRenderingHint`(`globalapi.go:320`) 与逐 Painter 的
      `XDraw_SetTextRenderingHint`(`xc/draw.go:1304`; 我们的 `Painter` 直接暴露 `Draw`, `draw.go:17`, 想逐帧调也行)
      在 **Direct2D 模式(`app.New(true)`)下完全无效**:全局设 `TextRenderingHintAntiAliasGridFit`(=3, 唯一常量,
      `xcc/xcconst.go:1078`)前后, 自绘文字块 / 真 Button 标签 / 原生 Edit 文字**逐像素完全相同**(`ImageChops` 差异包围盒 = None);
      逐 Painter 设 3 与默认同帧也逐像素相同(彩色边缘像素 2265 = 2265)。原因: 它们是 GDI+ 的 TextRenderingHint,
      D2D 的文字不走那条管线。**我们的文字本来就已经是 ClearType 级抗锯齿**(彩色边缘像素 2154/1474, 最大色偏 180/114),
      不存在"没抗锯齿/要换灰度 AA"这个可调点 —— 想关彩色边缘得另找路径(1.4.2 未见暴露)。

47. **弹层 / 对话框进出场动画(2026-10-06, 运行时 `presence.go`)**:

    语义逐条对齐本地 shadcn 参考实现(`shadcn/src/components/ui/*.tsx`):
    **dialog** = 遮罩 fade + 内容 fade+zoom-in-95(`dialog.tsx:42,64`, `duration-100`);
    **dropdown-menu / select** = fade+zoom+从锚点侧滑入 8px(`dropdown-menu.tsx:48`, `duration-100`);
    **tooltip** = 同上但没写时长 → tw-animate-css 的 `animate-in` 默认 **150ms**(`tooltip.tsx:44`)。
    缓动取 CSS 默认的 `ease` = cubic-bezier(0.25,0.1,0.25,1) —— `animate-in/out` 是 keyframes,
    用的是 animation 的默认缓动, **与 transition-\* 的 tailwind 默认 cubic-bezier(0.4,0,0.2,1) 不是同一条**。

    - **两个"不吃元素 alpha"的漏网之鱼**(2026-10-07, 都是淡入淡出时发现的):
      ① **D2D 直通绘制的图标**绕过炫彩的 alpha 合成(直通调用直接落在最终渲染目标上,
      元素 alpha 只作用于走 XDraw_* 命令队列的输出)→ 对话框淡入时关闭图标全黑不淡。
      修法 = d2dicon.go 出图前把元素当前 alpha(`XEle_GetAlpha`)乘进画刷颜色
      (窗口级 Painter 跳过);
      ② **输入框插入符是原生 caret**, 同样不随元素 alpha 淡。最终方案 = **焦点晚给**:
      淡入到位(Presence done)后才让第一个可聚焦控件拿焦点, 淡出开始先
      `XWnd_SetFocusEle(hWnd, 0)` 清焦点 —— 焦点没了插入符自然消失。曾试过进出场期间
      `XWnd_ShowCaret(false)` 藏、done 里恢复, 实测与编辑框自身的 blink 定时器打架
      (连续闪烁), 弃用; 晚 100ms 聚焦对"打开时焦点进第一个控件"的 radix 语义无感知
      影响, 还顺带避免了淡出 100ms 里键盘输入打进正在消失的输入框。
    - **运行时 API**(`presence.go`, 与 hover 共用 anim.go 的驱动器, 该处扩展了条目级
      时长/缓动/apply/done):`Presence(hOwner, key, visible, dur, apply, done)` 返回 0..1 进度,
      `apply` 每帧把进度落到元素透明度 / 窗口位置上, `done` 到位回调一次; 助手
      `SetElementsAlpha`(一组元素一起淡)、`SetSubtreeAlpha`(整棵**子树**逐个设 alpha ——
      元素透明度**不级联到子元素**, 只设 overlay 时标题/输入框等子元素会全不透明压在
      还没模糊的背景上, 实测踩坑)、`SetWindowAlpha` / `FadeWindowAlpha` / `SlideWindowAlpha`
      (窗口透明度 + 位移 + 自重绘)。调试用 `SetAnimationScale(f)` 把**全部**过渡放慢
      (12 倍时外部截图脚本才能抓到中间帧做像素验收)。
    - **已接入**:Dialog(遮罩+面板+全部子元素整组淡入淡出, `d.open` 立刻置 false 所以淡出期间
      键盘/点击已失效, 退场到位才 `showEleTree(false)`);Tooltip(气泡小窗 fade + 8px 滑入,
      退场 fade 完再 `SW_HIDE`; 已在显示中时只挪位置不重播动画)。
    - **shadcn 的 zoom-in-95 做不到**(留白):xcgui 的元素级 `ani` 动画(Alpha/MoveEx/Scale)对
      **自绘元素完全无效**(实测 `SetAlpha(0)`+动画后画面纹丝不动 —— ani 打在原生属性上,
      被 `AddEvent_Paint` 的自绘绕过), 而 Draw 只有 `SetOffset`(整数偏移)没有变换/缩放 API;
      所以 zoom 只能靠"改面板矩形+字号"模拟, 视觉会错位, 不做。Dialog 只做整组淡入,
      Tooltip/弹层做 fade+slide(slide 是 dropddown/tooltip 里最显眼的位移项)。
    - **踩坑(本次全踩过, 务必记住)**:
      1. **窗口透明度在绘制时才落地**:`XWnd_SetTransparentAlpha` 改完不重绘**该窗口**,
         画面会一直停在上一次绘制时的透明度上(实测: 过渡值从 0.009 一路推进到 0.25,
         像素纹丝不动)。apply 里必须 `XWnd_Redraw(该窗口, true)`。
      2. 更邪门的是: 过渡挂宿主窗口、每帧重绘**宿主**窗口时, 气泡小窗的透明度跟着动了;
         只重绘气泡小窗自己反而不动 —— 机理未知, 但结论明确: **把进出场过渡挂在宿主窗口上**
         (tooltip 就挂触发元素所在的宿主窗口), 让 tick 的整窗重绘带动一切。
      3. **alpha 精确 = 0 会让 xcgui 停止合成该窗口**(后续怎么调都不显示, 画面冻结在
         停止前那一帧; 调试日志显示 apply 已跑到 v=1.0 而像素不动)。两条触发路径:
         显示前预设 0, 以及**退场淡出结束在 0**(表现: 气泡第二次显示永远停在约 10%
         透明度)。修法 = `SetWindowAlpha` 统一抬底 `xui.AlphaFloor()`(1/255), "消失"
         一律由退场 done 里 SW_HIDE 完成; 进场首帧闪现也用预置 1/255 压掉。
      4. 异型置顶小窗(`Window_Style_Nothing` + `Shaped` + NOACTIVATE)自己的窗口定时器
         不可靠(约 100ms/帧), 别把过渡挂它身上(同 2, 挂宿主窗口)。
      5. **普通窗口(非透明类型)的 `XWnd_SetTransparentAlpha` 完全无效**(exstyle 里没有
         WS_EX_LAYERED, xcgui 也不会自动加) → native 弹层(menu / combobox 的下拉窗)做不了
         淡入; 要淡入只能自己给 HWND 加 WS_EX_LAYERED + SetLayeredWindowAttributes
         (代价: ClearType 变灰度 AA + 合成开销), 未做。这类弹层目前只能拿**位置**做动画。
      6. 首帧闪现: `SWP_SHOWWINDOW` 后的首次绘制带着上一次的 alpha(255) → 进场前先
         `SetWindowAlpha(1/255)`(见 3, 不能是 0)。
    - **验证**(探针 + `SetAnimationScale(12)` 放慢 + 外部逐帧像素实测):气泡淡入
      255→254→245→232→201→175→141→117→91→74→60→45→37→27→22→18→14→12→**10**
      (17 帧, 150ms×12)且顶部 y 36→**33**(8px 逻辑滑入); 1× 常速 77→24→**10** 正常完成;
      对话框遮罩点 255→251→247→241→237→235→232→231→**229**(淡入)与反向(淡出)平滑往返,
      淡出到位后内容树隐藏(`窗口最暗` 回到 255)。

    - **backdrop-blur-xs 完成**(2026-10-07, `components/backdrop.go`, 前一晚被误判卡死):
      思路 = 打开时 GDI `PrintWindow(PW_RENDERFULLCONTENT)` 抓窗 → Go 盒式模糊 ×4 趟
      (backdrop-blur-xs=4px 逻辑)→ **元素 BkM 图片填充**显示在遮罩之下(元素透明度能跟着
      遮罩一起淡入。注: 当年选 BkM 是误信"元素 paint 画不出位图", 实际能画 —— 见 17 条
      2026-10-07 修正; BkM 路线已验证且淡入一致故保留)。验收 = blurprobe 程序化
      Show/Close + 像素逐点比对(模糊+10%变暗复算期望 mean=0.23/255, 不模糊假设 mean=4.64
      被显著排除; 关闭还原全图 max=0)+ demo computer-use 真实鼠标双主题复核。
      **昨夜"XImage_LoadFromData 无导出"是误判**: 探测证实实际加载的 dll
      (`%TEMP%\xcgui4.0.0.0_amd64_<CRC>\xcgui.dll`)有该导出, `XImage_LoadFromGoImage`
      功能实测成功 —— 当时是 recover 兜底把真正的失败(下两条坑)静默成了"只有变暗"。
      踩坑(全是实测):
      1. **BkM 图片的状态用错就永不渲染**: 元素的"离开"态是 `Element_State_Flag_Leave`
         (0x0010), 不是窗口的 `Window_State_Flag_Leave`(0x0001)。用窗口常量注册到元素
         BkM 上, 元素 show/alpha/矩形全对也不出图(症状极具迷惑性, 状态转储看不出错)。
      2. **BkM 图片按 1:1 逻辑像素、居中、不缩放绘制**: 物理尺寸(1200x780)的快照喂给
         800x520 逻辑的元素会错位+被裁(网格采样定位到"图片落在元素中央")。修法 = 加载前
         先把模糊图面积平均缩放到窗口逻辑尺寸(`scaleRGBA`), 居中 1:1 即正好铺满。
         `BkObj.SetAlign(Left|Top|Right|Bottom)` 实测**对图片对象无拉伸效果**(别试了)。
      3. **注册走 `XBkM_AddImage(XEle_GetBkManager(ele), state, hImg, id)`**: 带 id 的
         管理器级注册在"窗口子元素直接挂"的环境里实测可靠; `Element.AddBkImage`(无 id)
         在 demo(布局盒子嵌套)能渲染、在平铺窗口的探针里不渲染, 原因未查明 —— 统一用
         带 id 的路线绕开。注意重复注册前先 `XEle_ClearBkInfo`(累积 >20 个炫彩报错)。
      4. **没有消息循环就没有一切**: 探针若只 Sleep 不跑 `a.Run()`, Presence 淡入淡出
         (窗口定时器)全不工作, 关闭也不会真的隐藏 —— 无泵跑出来的"验证结论"全部作废。
         程序化验收要用 `w.SetTimer + AddEvent_Timer` 驱动阶段, 再 `a.Run()`。
      5. 枚举窗口子元素要用 `XWnd_GetChildCount/GetChildByIndex`(喂 `XEle_GetChildCount`
         恒为 0); `XEle_GetAlpha/GetBkInfoCount/XObj_GetType` 是排查这类渲染问题的
         状态转储三件套。
      6. 模糊在物理像素上做(半径 = 逻辑 4px × DPI/96), 之后缩到逻辑尺寸再喂;
         `PrintWindow` 抓的是 BGRA, 且强制 A=255(GDI+ 认 alpha, 0 会整块透明)。

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
- backdrop-blur(见第八节 47):`xc.XImage_LoadFromGoImage`、`xc.XBkM_AddImage(hBkM, state, hImage, id)`、`xc.XBkM_GetObject`、`xc.XEle_GetBkManager`、`xc.XEle_ClearBkInfo`、`xcc.Element_State_Flag_Leave`(元素态, ≠ Window_State_Flag_Leave)、`XWnd_GetChildCount/GetChildByIndex`、`XEle_GetAlpha/GetBkInfoCount`、`wapi.GetWindowRect` + `PrintWindow(PW_RENDERFULLCONTENT)` 抓窗、`XWnd_SetTimer/KillTimer + AddEvent_Timer`(程序化验收)
- `app.GetTextShowRect`(文本测量:限宽换行与绘制共用限宽)
- 窗口真透明:`Window.SetTransparentType(xcc.Window_Transparent_Shaped)` + `Window.SetTransparentAlpha(255)`、`xcc.Window_Transparent_Shaped`(xcconst.go:55)
- 背景对象美化:`BkManager.AddFill/AddBorder(state, color, id)`、`BkManager.GetObjectObj(id) *bkobj.BkObj`、`BkObj.SetRectRoundAngle/SetMargin/SetAlign/SetRotate/SetFillColor/SetBorderColor`、`xcc.BkObject_Align_Flag_*`(bkmanager.go:69/185, bkobj.go:104/42/50/66/74/90)
- Dialog:`XEle_SetZOrderEx(h, 0, xcc.Zorder_Top)`、`XEle_AdjustLayout`、`XEle_Redraw`、`XEle_GetChildCount/GetChildByIndex`、`XObj_GetType`(→ `xcc.XC_*`)、`XWidget_GetParentEle`、`XWnd_GetRect/Redraw`、`XWnd_SetFocusEle/GetFocusEle`、`xcc.AdjustLayout_No`、`wapi.GetAsyncKeyState`
- Table(原生 List 载体):`CreateAdapterHeader/CreateAdapter/AddColumnText/SetHeaderHeight/SetRowHeightDefault/SetRowSpace/SetColumnWidth/SetSplitLineColor/SetDrawRowBkFlags/AddRowText/SetItemText/GetCount_AD/SetSelectRow/GetSelectRow/GetVisibleRowRange/RefreshRow/RefreshData/AddEvent_List_DrawItem/AddEvent_List_Header_DrawItem/AddEvent_List_Temp_Create_End/AddEvent_List_Header_Temp_Create_End/AddEvent_List_Select/GetTemplateObject/GetHeaderTemplateObject/GetHeaderHELE/GetScrollBarVObj`
- ⚠️ `TabBar.SetPadding` 在 xcgui.dll 4.0.0.0 中不存在(调用 panic)
- 直通 D2D(图标抗锯齿, 见第八节 43):`xc.XDraw_GetD2dRenderTarget(hDraw) → *ID2D1RenderTarget`、`xc.XC_GetD2dFactory() → *ID2D1Factory`、`xc.XDraw_ConvRect(hDraw, *RECT) / XDraw_ConvXY`(客户区坐标 → 画布对齐物理坐标, 第三方绘图必须先换算); 用到的 D2D 方法: `ID2D1Factory::CreatePathGeometry`、`ID2D1PathGeometry::Open / GetBounds`、`ID2D1GeometrySink::{SetFillMode, BeginFigure, AddLines, EndFigure, Close}`、`ID2D1RenderTarget::{CreateSolidColorBrush, FillGeometry}`
- 官方自绘示例:BeautifyButton / CheckBox / RadioButton / Edit / ProgressBar / SliderBar
- 官方透明/异型窗口示例:`Advanced/WindowPartiallyTransparent`、`webview/WebViewPartiallyTransparent`
