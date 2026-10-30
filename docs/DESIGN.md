# xui — shadcn 风格的 xcgui 自绘组件库: 自定义指南

> 这份文档随 `xui init` / `xui init -n` 安装到你的 `ui/` 目录, 是组件源码的配套说明书。源码注释里的
> "见 DESIGN 第 N 节 / 第八节 N 条"指的就是这里 —— 章节与条目编号稳定, 只增不改。
>
> **按需查**:
> - 改主题 / 换配色 → **第三节**(令牌表 + "绘制时取令牌"的规则)
> - 改组件外观 → **第四节**(以 Button 为例的解剖 + 每个组件的技术路线)
> - 改交互语义(悬停/焦点/禁用/校验失败) → 先看**第四节末尾的语义汇总表**
> - 碰到 xcgui 的怪行为(不刷新 / 透明失效 / 布局塌陷 / 事件不触发 / 文字量错) → **第八节**, 按关键词找
> - 要调某个 xcgui API → **第九节**, 那里逐条标注了实测结论与坑
> - 安装更新 / 目录结构 / 已知限制 → **第五~七节**
>
> 全部结论基于 xcgui **1.4.3** 实测; 组件实现以 `ui/*.go` 为准。交互语义的原始参考是
> shadcn-ui 的 React 源码, 文中"对齐"均指与它一致。
>
> 文中提到的"示例程序"指 xui 仓库自带的 `demo/`(全部组件的可视化示例, 可对照运行);
> "探针"指为验证某条 xcgui 行为临时写的小程序 —— 这类结论都写过小程序逐像素/逐消息确认过,
> 不是凭文档或经验猜的。

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
| registry.json | 组件清单(CLI 按它拷贝源码) |
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
- **组件文件要写得"薄"**:paint 细节尽量调用运行时助手,拷贝文件里主要剩下"变体→token 映射 + 几笔绘制调用"。越薄,拷贝模式的升级风险越小。shadcn 正是这样(cn、Slot、Radix 全在依赖里,拷贝的文件基本是 JSX + 类名)。

---

## 三、主题系统(核心中的核心)

### 3.1 令牌集:与 shadcn 一一对应

```go
type Theme struct {
    Name string // 主题名(调试 / 主题切换菜单用)

    Background, Foreground Color // 窗口底色 / 默认正文文字

    Card, CardForeground       Color
    Popover, PopoverForeground Color // 弹出层(菜单、下拉)底色

    Primary, PrimaryForeground         Color // 主操作色 / 其上的文字
    Secondary, SecondaryForeground     Color // 次要填充
    Muted, MutedForeground             Color // 弱化填充(输入框底、禁用底) / 弱化文字(说明、占位符)
    Accent, AccentForeground           Color // 悬停高亮
    Destructive, DestructiveForeground Color // 破坏性操作

    Border, Input, Ring Color // Input 通常与 Border 相同; Ring = 焦点环

    Chart1, Chart2, Chart3, Chart4, Chart5 Color // --chart-1..5(组件库不画图表, 供用户应用取用)
    Sidebar, SidebarForeground, SidebarPrimary, SidebarPrimaryForeground,
    SidebarAccent, SidebarAccentForeground, SidebarBorder, SidebarRing Color

    Radius              int32 // 基准圆角 px
    FontDefault, FontMono FontSpec
}
```

> **字段命名约定**: 字段名 = shadcn CSS 变量的**直接 CamelCase, 不做缩写** —— `--accent-foreground` → `AccentForeground`。缩写会迫使批量转译社区主题时带一张特例表。
> 两处与上游的差异要知道:① `DestructiveForeground` 是本库的**扩展令牌** —— nova 预设已删除 `--destructive-foreground`(破坏性按钮直接写死白字), 我们保留它以便组件取"破坏色上的文字"; ② `--chart-1..5` 与 `--sidebar-*` 已建模(`Chart1..5` / `Sidebar*`): **chart 色板只提供给用户应用**(组件库不画图表), **sidebar 令牌由 `ui.Sidebar` 使用**(分组/条目/描边/焦点环全部取自这组色) —— 机械转译社区主题时两组都要填上(内置主题的换算值与 `manager_test.go` 的漏填守卫都在)。
> 基准圆角与 shadcn 一致: `--radius` 在 nova 预设下 = 0.625rem = 10px, 对应 `Theme.Radius = 10`。

Color 封装 uint32,提供 `xui.Hex("#fafafa")`(底层 `xc.HexRGB2RGBA`)。主题文件长得和 CSS 一样,任何 shadcn 社区主题都能机械转译成 Go(`init` 生成的 `ui/theme.go` 默认取内置主题的**副本** —— `light = *xui.Light`, 见 `cmd/xui/main.go` 的 `themeTemplate`):

```go
var dark = &xui.Theme{
    Name:       "dark",
    Background: xui.Hex("#09090b"),
    Foreground: xui.Hex("#fafafa"),
    Primary:    xui.Hex("#fafafa"),
    ...
    Radius:     8,
}
```

### 3.2 关键决策:绘制时解析 token,组件永不缓存颜色

官方 BeautifyButton 示例把颜色缓存在结构体字段里,再用 SetThemeColor 重算——**这是让主题系统难做的反模式**。xui 组件一律:

```go
func (b *Button) onPaint(hEle, hDraw int, pbHandled *bool) int {
    t := xui.Current          // 每帧取当前主题
    bg := t.Primary             // 绘制时解析
    ...
}
```

推论:
- **主题切换 = 换全局指针 + Redraw**,所有组件下一帧自动用新色。这正是 CSS 变量的工作方式,而且是零成本的。
- 变体逻辑是纯函数 `(variant, state) → token`,不开窗口也能单元测试。

### 3.3 主题管理器

```go
xui.SetTheme(&myTheme)      // 切换 + 通知已注册窗口重绘(内部 XWnd_Redraw)
xui.OnThemeChange(fn)       // 需要同步原生 API 颜色的地方订阅(如 SetTextColor), 返回反注册函数
xui.Dark / xui.Light    // 内置主题(*Theme)
```

组件若只在 paint 里取色,**无需订阅**;只有同步给原生 API 的颜色(文字色、窗口底色)才需要回调。

**API 形状决策**:内置主题用**包级函数** `xui.Light / xui.Dark`,**不提供 `xui.Themes` 命名空间对象**。Go 里"包本身就是命名空间",用一个全局单例对象当模块不合惯例(收益只是省一个点号,代价是多一个可变全局 + godoc 层级更深)。将来做第七节 P4 的"主题画廊"(几十上百个主题, 包级函数会污染根包名)时,正确解法是新增 **`xui/themes` 子包**(`themes.Get(name)` / `themes.All`, 与本库 `icons` 子包同构),根包保留这两个函数作默认兜底;届时 `xui.Light` 迁到 `themes.Light` 保留别名即可, 非破坏。

> 相关实现约定:`OnThemeChange` 返回的反注册函数**必须真的生效**(Go 函数值不可比较, 用 `&f == &fn` 比函数地址恒为假 → 反注册是空操作; 用自增 id 做句柄);窗口底色绑定(`BindWindowBackground`)由管理器持有, `UnregisterWindow` 会一并解绑, 避免主题切换时对失效句柄调 `XWnd_*` 弹错误框;**组件构造函数里的主题订阅一律用 `xui.OnThemeChangeFor(元素句柄, fn)`**(元素销毁时自动反注册, 见第八节 54), 裸 `OnThemeChange` 只用于非元素的长生命周期订阅。

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

const (ButtonPrimary ButtonVariant = iota // 实心主按钮
    ButtonSecondary                     // 弱化填充
    ButtonOutline                       // 边框按钮
    ButtonGhost                         // 幽灵按钮, hover 才有底色
    ButtonDestructive                   // 破坏性(软色调))

type ButtonSize int

const (ButtonSizeSm      ButtonSize = iota // 高 28
    ButtonSizeDefault                   // 高 32
    ButtonSizeLg                        // 高 36)

type Button struct {
    *widget.Button
    variant  ButtonVariant
    size     ButtonSize
    icon     string // 非空时绘制图标替代文字(WithIcon)
    iconSize int32  // 图标绘制边长(WithIconSize); 0 = 默认 18
}

type ButtonOption func(*Button) // 函数选项模式, 等价 cva 的 variants 参数

func WithVariant(v ButtonVariant) ButtonOption  { return func(b *Button) { b.variant = v } }
func WithSize(s ButtonSize) ButtonOption        { return func(b *Button) { b.size = s } }
func WithIcon(path string) ButtonOption         { return func(b *Button) { b.icon = path } }
func WithIconSize(size int32) ButtonOption      { return func(b *Button) { b.iconSize = size } }

func NewButton(x, y, cx, cy int32, text string, hParent int, opts ...ButtonOption) *Button {
    b := &Button{Button: widget.NewButton(x, y, cx, cy, text, hParent)}
    for _, o := range opts {
        o(b)
    }
    b.EnableBkTransparent(true)
    b.AddEvent_Paint(b.onPaint)
    return b
}

func (b *Button) onPaint(hEle, hDraw int, pbHandled *bool) int {
    t := xui.Current
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
| AppWindow | 原生 window.Window 封装 | 无系统标题栏(样式不含 Icon/Btn_*): 阴影圆角 + 自绘 `[图标][标题 Weight][自定义按钮区][最小化/最大化/关闭]`; 双击标题栏最大化/还原; 内容区 = 纵向流式 Weight 1(第八节 76) | demo 窗口 |
| Button | embed Button | paint 壳, 原生文字(WithIcon 时自绘图标) | BeautifyButton 示例 |
| Card | LayoutEle(布局容器) | 圆角面板 + ring 描边; 内容区是 Fill 子项、左右内边距用元素 padding / 上下用外边距(随卡片缩放, 见第八节 129), 高度可按内容自撑(Auto), 见第八节 56; 页脚是横向布局容器(内边距 + items-center), 见第八节 95 | — |
| Collapsible | LayoutEle(根 + 内容) + Button(触发区) | 竖向流容器: 触发区由调用方创建 Button 后 `BindTrigger` 绑定(展开态同步 Button 的 aria-expanded 视觉); 内容高度收起 0 / 展开 Auto, 切换时 200ms 逐帧重排祖先并整窗重绘(第八节 97) | collapsible.tsx + radix collapsible(高度 CSS 变量动画示例) |
| AspectRatio | LayoutEle(布局容器) | 宽度驱动高度: 尺寸事件按新宽度算高写回, 延迟到下一轮消息并逐级重排祖先(同第八节 93 的三段式); 内容区随比例框同步尺寸, 见第八节 94 | shadcn/radix aspect-ratio |
| Alert | LayoutEle + Label/Element | 卡片底 + 1px border; 图标/文本列横向流式, 标题/描述按宽度量高后写回, 动作槽按内容自撑; 见第八节 90 | alert.tsx(radix-nova) |
| Empty | LayoutEle + Label/Element | 竖向居中的空状态: Body(p-6/gap-6) + Header(gap-2) + Content(gap-4); 媒体是自绘图标块(icon 变体 size-10 圆角 bg-muted); 不画边框(上游 `border-dashed` 无宽度), 需要边框放进 Card | empty.tsx(radix-nova) |
| Field | LayoutEle + Label/Element | 表单字段组合: Field/Set/Group 全是流式布局容器; Label/Description 走 Label, Separator 自绘线 + 背景压字, Error 多条 Label 去重; 横向 Content 固定 240(第八节 101) | field.tsx(radix-nova) |
| Checkbox/Radio | embed Button | 画方框/圆 + 勾线, 文字原生 | BeautifyCheckBox/RadioButton 示例 |
| Input | embed Edit | 只画边框+焦点环, 原生光标/IME | BeautifyEdit 示例 |
| Switch | 纯自绘 Element | 轨道+滑块, 尺寸预设; hover 过渡走运行时 `anim.go`(第八节 45); aria-invalid 红边 + 常驻红环, 空格切换, `SetDisabled` 关鼠标与键盘(第八节 146) | — |
| Label/Separator/Badge | 纯 Element | paint | — |
| Progress | 纯自绘 Element | h-1 圆角轨道 + 指示器(宽度 = value/max); max 可配, 不确定态短条往返, 指示器颜色可覆盖; `ProgressField` = 标题+百分比+条(第八节 145) | BeautifyProgressBar 配方移植成 token + progress 文档 |
| Slider | 纯自绘 Element | **弃用原生 SliderBar**(滑块按钮 BkInfo 残留压不住, 见第八节 25), 自实现拖拽/点击跳转/键盘; 值数组(单值/范围/多滑块)、垂直、禁用、最小间距、inverted、onValueCommit(第八节 144) | BeautifySliderBar 示例 + 探针 |
| ComboBox | embed ComboBox | 画输入框; 下拉 ListView 弹层在 `AddEvent_Combobox_Popup_List` 里半接管(第八节 30) | 库文档 FAQ |
| Tabs | LayoutEle(列表底) + Button × N(全自绘) | paint(default/line 两变体, 行内图标 + 单标签禁用); 整块列表一个 Tab 站点 + 内部方向键(键盘挂窗口级, 见第八节 152/155); `BindPanel` 按选中项显隐面板; 换行/纵向/`activationMode`/value 语义 | tabs.tsx(radix-nova) |
| Dialog | 纯 Element 覆盖层 | 全窗遮罩元素(自绘黑 10% + backdrop-blur) + 面板子元素;ESC/点遮罩关闭, Tab 焦点循环(第八节 33);可滚动内容用内嵌 ScrollArea(第八节 91) | — |
| AlertDialog | 复用 Dialog 覆盖层 | 同一套遮罩/模糊/焦点陷阱/模态栈;面板按 default/sm 排头部与页脚, media 自绘;点遮罩不关、焦点强制 Cancel(第八节 91) | alert-dialog.tsx(radix-nova) |
| Menu | 原生菜单 | `XMenu_EnableDrawBackground/EnableDrawItem` + `AddEvent_Menu_Draw_Background/DrawItem`(第八节 36) | element.go 已确认事件存在 |
| Tooltip | FloatWindow | paint, 异型透明窗口 + 进出场 fade/slide(第八节 24/47); 上下左右四方向 + align/sideOffset + 自动翻转夹紧(第八节 156), 延迟/skipDelay、键盘焦点触发、快捷键块 | tooltip.tsx(radix-nova) |
| Table | 原生 List + 项模板 | 表头/行底/边框/滚动条全自绘(绘制回调按**单元格**触发, 见第八节 40); 原生 Table(Shape 系)与 shadcn 语义不符, 不单独做 | Basic/List 示例 + 本库探针 |
| Tree | 原生 Tree + 项模板 | `XE_TREE_DRAWITEM` 自绘行底/参考线/chevron, 模板子元素保留(第八节 42) | 库文档 + 探针 |
| Pagination | 纯自绘单元素 | 条目行 `[上一页][页码…][下一页]`, 逐项 hover 过渡; 恒定槽位折叠(第八节 53); 简洁 / 仅导航 / 只留图标三种形态; 一个 Tab 站点 + 内部方向键(键盘必须挂窗口级, 第八节 152/153) | — |
| Icon | 直通 D2D 几何绘制 | lucide path → 描边几何 + `DrawGeometry`(圆角样式), 色点走 `FillGeometry`(第八节 43); 回落折线描边 / `svg.Svg` | svg 包 + D2D 探针 |
| NativeSelect | ComboBox(value 封装) | 载体就是 NewComboBox: 触发区自绘 + 原生弹层(第八节 30), 在其上提供 `<option value>` 字符串语义(Value/SetValue/OnValueChange) | native-select.tsx(radix-nova) |
| Select | Button(触发区, 全自绘) + popoverPanel + ScrollArea | 触发区与 ComboBox 同一套 paint 规则; 弹层是自建阴影窗(popover.go 骨架, item-aligned 定位见第八节 71/74), 条目是自绘 Button(悬停/焦点 = bg-accent, 选中右侧 check, 禁用置灰), 分组标签/分隔线是自绘 Element; 滚动用 ScrollArea(条目超出高度才出), 键盘 ↑↓/Home/End/Enter 走弹窗 KeyDown 钩子 | select.tsx(radix-nova) |
| InputOTP | 每格一个原生 Edit(32×32) + 自绘壳 | 文字居中用原生 `SetTextAlign(Center\|Center_V)`(单行 Edit 程序 SetText 同样生效; 不要用 border 挤位置, 会切字, 见第八节 110); 字符一律在 `AddEvent_Char` 拦截后自己落格(替换已填格 + 非法字符不入格 + 自动前进), 粘贴在 KeyDown 拦 Ctrl+V 读剪贴板分发; 相邻格共享边 = 每格画 top/bottom/right、首格加 left、中段格的 left 用底色擦掉(防 2px 叠边); aria-invalid 的组外圈画在容器 PAINT_END(子元素之后, 普通 paint 会被格子盖掉) | input-otp.tsx(input-otp 包) |
| Accordion | LayoutEle + Button(check, 全自绘) + Collapsible | 每项 = 触发钮(Toggle 式全自绘: 文字 hover 下划线**手绘 1px 线**, 箭头随开合换 ChevronUp/Down, 高度固定 40) + 一个 Collapsible 做高度过渡; 条目间 1px 分隔线(末项没有); single 互斥在 item.setOpen 里关其它项, 只通知一次; 内容区 pb-2.5 用 10px 透明占位(不能用外边距, 见第八节 111) | accordion.tsx(radix-nova) |
| Popover | popoverPanel(阴影窗) + LayoutEle | 复用 datepicker/select 的弹层底座, 内容区改成布局容器: 离屏显示一次量出 Auto 高(第八节 115), 再按 side/align 摆放并在工作区内翻转/夹取; 点击触发, 失焦/ESC 关闭; 公开 `NewPopover` 与 Header/Title/Description 助手 | popover.tsx(radix-nova) |
| HoverCard | popoverPanel(非激活模式) + LayoutEle | 同 Popover, 但窗口带 `WS_EX_NOACTIVATE`、不调用 SetForegroundWindow、不靠 KillFocus 关闭(第八节 116); 700ms 悬停打开 / 300ms 离开关闭, 指针仍在卡片上时关闭检查自动续期 | hover-card.tsx(radix-nova) |
| Avatar | 纯自绘 Element | 圆形图片用 CPU 居中裁剪 + 抗锯齿圆形蒙版生成 RGBA, `XImage_LoadFromDataRGBA` + `XDraw_ImageEx` 绘制(第八节 117); 无图回落首字母, 右下角徽标; AvatarGroup 用横向布局负间距叠放 | avatar.tsx(radix-nova) |
| Command | LayoutEle + 原生 Edit + ScrollArea | 搜索行(自绘图标 + 透明 Edit + 下边框) + ScrollArea 列表; 分组/条目是自绘 Element, 输入即过滤(跳过禁用), ↑↓/Home/End/Enter 在 Edit 的 KeyDown 里接管; 选中不自动关闭(交给 Dialog/Popover 组合) | command.tsx(cmdk) |
| Sheet | 复用 Dialog 覆盖层(模态栈/焦点陷阱/遮罩) | 面板贴边: 左右占满高、上下按内容撑开; 面板与其直接子元素整体平移做滑入(第八节 120), 遮罩 bg-black/50; 页脚按钮纵向铺满; 只画贴边侧 1px 边框 | sheet.tsx(radix-nova) |
| Drawer | 复用 Sheet | 底部 max-h-80vh + 顶部抓手 + 上方圆角; 抓手按住捕获鼠标, 拖过面板高 1/3 关闭、否则回弹(第八节 121); 方向可改 top/left/right | drawer.tsx(vaul) |
| Menubar | 自绘根条 + 每个菜单一个 popoverPanel + 自绘条目 Element | 原生 `XMenu_Popup` 阻塞、期间主窗口收不到鼠标移动, 无法做悬停切换, 因此弹层自绘条目(与 Menu 共用 menu_model.go 的条目模型); 打开后鼠标移到其它触发区即切换; ↑↓/Home/End/Enter/←→/ESC/Tab; 子菜单是第二个激活弹层, 父层 `keepOnBlur` 防止失焦关闭(第八节 122~125) | menubar.tsx(radix-nova) |
| NavigationMenu | 自绘根列表 + **一个共享 popoverPanel(非激活)** | 所有菜单共用一个视口: 贴列表左下方、宽高随当前菜单内容变化, 切换时不重开窗口; 触发区/顶级链接是全自绘 Button; 内容页是 LayoutEle(链接行助手 `Link`/`LinkIn` + 任意控件); 悬停 200ms 打开、离开 150ms 收起、关闭后 300ms 内切其它触发区立即打开(第八节 137 的轮询同时负责链接 hover 与收起判定) | navigation-menu.tsx(radix-nova) |
| Carousel | 根 Element + 裁切视口 + 每页一个 LayoutEle | 每页宽 = 视口宽 − gap(16), 过渡 300ms 用 `xui.Presence` 每帧重写各页矩形; 循环用"模总长"折回, 首尾相接、不需要克隆页(第八节 141); 拖拽走窗口级起手 + `SetCaptureEle` 跟手(第八节 140); 上/下张是内容外侧 48px 的圆形 Button | carousel.tsx(radix-nova, embla) |
| Resizable | 分组 = LayoutEle(横向/纵向) + 面板 = LayoutEle + 分隔条 = 自绘 Element | 面板按百分比分配主轴空间, 像素宽按累计边界取整(每个手柄固定 4px); 分隔条画 1px `bg-border` 线 + 可选 4×24 抓手; 拖拽把鼠标捕获到**分组**(手柄会随布局移动, 不能当坐标原点), 键盘挂窗口级(同 152): 方向键 5%、Home/End 到边界、双击复位(第八节 157) | resizable.tsx(radix-nova) + react-resizable-panels v4 |

非 Button 载体的组件(hover 态自管)用运行时提供的 `HoverTracker`(基于 `AddEvent_MouseStay/MouseLeave`,已确认存在)。

尺寸预设对标 shadcn(`h-9 px-4` 之类),经 DPI 助手换算。

### 与 shadcn 对齐的语义(权威汇总)

> **这是交互语义的唯一汇总表**;改任何组件的交互前先查这里,再回到本地 shadcn 参考实现(`shadcn/src/components/ui/*.tsx`)确认。每条的根因/实测见第八节对应条目。

| 语义 | shadcn 源码依据 |
|---|---|
| **hover 过渡 = 150ms cubic-bezier(0.4,0,0.2,1)**(tailwind `transition-*` 默认值); 组件侧 `xui.Hover(ele, 悬停中)` / `xui.HoverItem(hOwner, idx, 悬停中)` + `xui.Fade(静止色, 悬停色, hv)`, 叠加层用 `ScaleAlpha(hv)` | button/tabs/switch/badge 的 `transition-all`; table/select/checkbox 的 `transition-colors`; 见第八节 45 |
| hover:bg-primary/80(实心)/80 透明度 | button.tsx variant.default |
| focus-visible 键盘才显示焦点框,ring-3 ring-ring/50,边框变 Ring 色;destructive 变体**覆写**为 border-destructive/40 + ring-destructive/20;焦点内芯**先铺宿主底色再叠 bg** —— bg 可能是半透明色(destructive/10), 直接叠在 halo 上会变成 30% 红、内芯比光环还深, 浏览器里两者是各自独立合成在页面底色上的;幽灵/链接(bg=None)铺 Card 即"透明底透出面板色"(halo 画在元素矩形内, 口径同 FocusFrame) | button/input/checkbox 的 focus-visible:* |
| 文本输入框点击聚焦也显示环(浏览器对文本控件恒匹配 focus-visible);禁用 = `bg-input/50`(暗色另有 `dark:bg-input/80`)再按组级乘 50%, 字色预混变淡 | input.tsx + 第八节 61 |
| disabled:opacity-50 原色半透明 | 各组件 disabled:* |
| **禁用/未选态的半透明令牌要"乘"不要"覆盖", 且先铺宿主底色**(暗色 input/border 令牌自带 alpha) | checkbox/radio/switch/input.tsx + 第八节 61 |
| active:translate-y-px 按下整体下沉 1px(含文字→文字手动绘制);**aria-haspopup 触发区(菜单/下拉)不下沉**;aria-expanded 展开时 outline/ghost 换 bg-muted、secondary 保持 secondary 底 | button.tsx 基类 + aria-expanded 变体 |
| **button group**: 首尾保留外圆角、相邻边收直角;xs/sm/icon-xs/icon-sm 在组内圆角统一成 rounded-lg;直系子项相邻边框自然并成 1px;组内嵌组时直接子项间距 8(gap-2);文本块/分隔条拉伸到按钮视觉高 | button-group.tsx + button.tsx 的 `in-data-[slot=button-group]` 规则 + 第八节 86 |
| **button 加载态**: 行内前置图标位置换成 lucide loader-circle(纯图标按钮替换图标), 1s/圈 linear 持续旋转, 常配 `disabled`;只负责视觉, 是否禁用由调用方决定 | shadcn button 文档的 spinner 示例(`LoaderCircleIcon animate-spin` + disabled)+ 第八节 87 |
| destructive 软色调(bg-destructive/10 + 红字) | button.tsx / badge.tsx |
| radio 选中 = bg-primary 实心圆 + size-2 白点(bg-primary-foreground); **RadioGroup = 组值 + 竖向 gap-2(8)**: 方向键在可用项间移动并选中(↑↓←→ 循环)、Home/End 到首末、空格选中当前项; 同父兄弟交给原生互斥, 放进 Field 的跨父级条目由组手动取消; Tab 按桌面惯例遍历各项(非 radix 的 roving tabindex) | radio-group.tsx + 第八节 143 |
| checkbox size-4 rounded-[4px] border-input;未选**亮色无底色 / 暗色 dark:bg-input/30**;选中 bg-primary + 白勾;不确定态(`checked="indeterminate"`)画减号、点击落到选中;空格切换;禁用 = 上面每个图层各乘 50% | checkbox.tsx + Radix Checkbox indeterminate + 第八节 61 |
| card 描边 = ring-foreground/10(非 Border 令牌), rounded-xl;卡片头 = 一行两列(左列标题+说明 `1fr`, 右列动作 `auto`), 动作区贴右上、跨标题与说明两行顶端对齐(gap-1 = 4);页脚 = 全宽 `border-t + bg-muted/50`、内容垂直居中、四边内边距跟随卡片尺寸(default 16 / sm 12)、只有下两角随圆角 | card.tsx + `ui/card.go` 的 `CardHeader` / `CardFooter` |
| **aspect ratio** = 只有比例一个参数, **宽度驱动高度**(高度 = 宽度 / ratio);宽度变化(窗口缩放、布局分配)时自动重算;内容区铺满比例框, 不裁剪溢出内容 | radix aspect-ratio(`@radix-ui/react-aspect-ratio`, CSS aspect-ratio 语义) + 第八节 94 |
| alert = bg-card + 1px border + rounded-lg(内边距 10/8); 可选图标 size-4 下移 2、与文本列间距 8; 标题 text-sm 中等字重、描述 text-sm muted, 行间 2; destructive = 标题/图标 destructive、描述 destructive/90; 动作槽贴右上, **无图标/无动作时对应列不占位**; 标题/描述按宽度折行、高度随内容自撑 | alert.tsx(radix-nova) + 第八节 90 |
| **aria-invalid**(表单校验失败): 边框变 destructive + ring-3 destructive/20(暗色 border /50、ring /40), **常驻不要求焦点**;与 focus-visible 同时成立时红色优先(变体顺序里 aria-invalid 在后);指示器未选中时没有自己的填充, 红环必须先用宿主色盖住指示器矩形, 否则环会从内部透出来 | button.tsx / input.tsx / select.tsx / checkbox.tsx / radio-group.tsx + 第八节 69 |
| badge h-5 胶囊(rounded-4xl) | badge.tsx |
| switch 轨道 32x18(选中 bg-primary/未选 bg-input, **暗色未选是 dark:bg-input/80**),滑块 16 圆、两侧各留 1px;dark 下选中滑块用 primary-foreground、未选用 foreground;**禁用时轨道与滑块一起乘 50%**(滑块不跟班会剩一颗全亮白圆);`SetDisabled` 同时关鼠标与键盘(原生 `Enable(false)` 不禁鼠标聚焦, 见第八节 146);**aria-invalid** = 轨道描边 destructive + ring-3 destructive/20(暗色 /50、/40), 常驻不要求焦点;空格键切换(原生 Check 按钮不响应空格, 组件自补) | switch.tsx(radix-nova) + 第八节 61/146 |
| switch 元素比轨道每边大 3px:给 focus-visible / aria-invalid 的 ring-3 留绘制空间,同时外扩点击热区(≈ after:-inset-*) | switch.tsx |
| progress = h-1 rounded-full bg-muted 轨道 + bg-primary 指示器; 宽度 = value/max, max 可配(默认 100), 指示器颜色可覆盖; **value=null 的不确定态**用 30% 短条沿轨道三角波往返 | progress.tsx + 第八节 145 |
| **ProgressField** = 标题行(左标题 / 右百分比)+ 进度条, 高 32(标题 20 + 间距 8 + 条 4), 对应文档 Labels 示例(Field + ProgressLabel + ProgressValue); 数值格式可覆盖 | progress 文档 + 第八节 145 |
| **sheet**: 遮罩 bg-black/50; 面板 bg-background, 左右 h-full w-3/4 max-w-sm、上下 w-full h-auto, 贴边侧 1px border; 头部 p-4(gap-1.5)、页脚 p-4(gap-2) 按钮纵向铺满; 右上角关闭钮; 点遮罩/ESC/X 关闭, Tab 焦点陷阱同 Dialog; 进出场从贴边侧滑入/滑出 | sheet.tsx(radix-nova) + 第八节 120 |
| **drawer**: 底部面板 max-h-[80vh]、上方两角 rounded-lg、border-t; 顶部抓手 w-[100px] h-2 mt-4 rounded-full bg-muted; 抓手按住下拖过 1/3 关闭、未过阈值回弹; 头部/页脚同 Sheet | drawer.tsx(vaul) + 第八节 121 |
| **menubar**: 根条 h-9 rounded-md border p-1 gap-1; 触发区 rounded-sm px-2 py-1 text-sm, 悬停/展开 bg-accent; 弹层 min-w-[12rem] rounded-md border p-1, sideOffset 8 / alignOffset -4; 条目 px-2 py-1.5 text-sm, 悬停 bg-accent, 勾选/单选指示器在左(pl-8), 快捷键右对齐; **打开后悬停其它触发区即切换**; ←→ 切换顶级、↑↓/Home/End 移动、Enter 选中、ESC 关闭还焦点、Tab 关闭; 子菜单悬停/Enter 向右展开 | menubar.tsx(radix-nova) + 第八节 122~125 |
| **navigation menu**: 根列表 gap-1; 触发区 h-9 px-4 rounded-md text-sm, 悬停 bg-accent、展开 bg-accent/50(悬停回全亮), chevron 12 随展开旋转 180°(300ms); 顶级普通链接 p-2; **所有菜单共用一个视口**, 贴列表左缘下方 mt-1.5、p-2、rounded-md border bg-popover, 宽高随当前菜单内容变化; 内容链接行 p-3(标题 20 + 间距 4 + 说明 20), 悬停/键盘高亮 bg-accent, 键盘高亮另画 focus-visible 环; 悬停 200ms 打开、离开 150ms 收起、关闭后 300ms 内切其它触发区立即打开; ←→ 移动顶级焦点、↓ 进入内容、↑↓/Home/End 选链接、Enter 激活、ESC/Tab 关闭; 内容页面切换时从切换方向滑入 208px + 淡入(200ms) | navigation-menu.tsx(radix-nova) + 第八节 137~139 |
| **carousel**: 每页 basis-full 整页位移, 页间 16 间隙(`-ml-4` + `pl-4`); 上/下张 = outline + icon-sm + rounded-full 按钮, 在内容左右(纵向时上下)48px 处, 到边界禁用、loop 时始终可用; ←/→ 在轮播内控件获焦时翻页; 循环首尾相接(最后一张的下一页紧跟第一张); 拖拽跟手、松手吸附最近一页(阈值 6 区分点击) | carousel.tsx + embla 默认 loop=false/align=start + 第八节 140/141 |
| **resizable**: 分组内面板按**百分比**分主轴空间, 手柄各占固定 4px, 窗口缩放时保持相对比例; 分隔条 = 1px `bg-border` 视线 + 4px 热区, `withHandle` 时中间一根 4×24 圆角抓手; 拖动只伸缩两侧面板(相邻到 min 后继续挤压同侧更远的), 增长端受 max; ←/→(纵向 ↑/↓)每次 **5 个百分点**、Home/End 到边界、双击分隔条把"有显式默认值的那一侧"复位; 点击分隔条获得焦点(嵌套元素不在 xcgui 的 Tab 链上, 见第八节 157), focus-visible 画一条更粗的 ring 色单线; 一侧面板收到 0 时分隔线贴到分组边缘(与外层边框重合)且抓手隐藏; 光标 col-resize / row-resize | resizable.tsx + react-resizable-panels v4(方向键 5%、双击复位) + 第八节 157 |
| **sidebar**: 展开 16rem(256)/ 图标栏 3rem(48), floating/inset 收起宽度 = 48+16+2 = 66(面板 50); Header/Footer = p-2 + gap-2, Group = p-2, 组间 gap-2; 菜单按钮 h-8 text-sm(sm h-7 text-xs / lg h-12), 选中 = bg-sidebar-accent + 加粗; outline 变体 = bg-background + 1px sidebar-border(悬停边框转 accent); 徽章/动作钮贴右(right-1), 动作钮 20×20、`showOnHover` 只在悬停时显示(选中只换图标色, 不显示动作); 子菜单 mx-3.5 + 1px 竖线 + px-2.5, 子项 h-7; **有子菜单的条目右侧有 chevron(展开转 90°), 点击整行开合(200ms 高度过渡)**; **icon 折叠**: 菜单项压成 32×32 只留居中图标, 分组标签/子菜单/徽章/动作/chevron 隐藏(点图标不切换子菜单), 悬停菜单项弹右侧 tooltip(delay 0); **offcanvas 折叠**: 面板整体滑出视野(gap 256→0), 内容不缩窄; 宽度过渡 200ms(Ctrl+B 或触发钮/rail); inset 变体 = 外层 sidebar 底色 + 主内容区 m-2(收起态靠侧栏一侧再加 8)圆角卡片; rail = 内缘 16px 热区, 悬停出 2px sidebar-border 线, 点击开合 | sidebar.tsx(radix-nova) + 第八节 158 |
| slider = **纯自绘**(普通 Element, 不用原生 SliderBar——其滑块按钮 BkInfo 残留绘制压不住, 见第八节 25):轨道 h-1 bg-muted + 已完成区间 bg-primary + 滑块 size-3 border-ring bg-white(字面量 white),滑块热区 hover / 拖动中 / focus-visible 显示 ring-3; **值用数组表达**(长度 = 滑块数量: 1 个普通、2 个范围、3 个以上多滑块), 单滑块区间从值域下限的几何位置画到滑块、多滑块画首尾之间; 相邻滑块间距受 `minStepsBetweenThumbs` 约束 | slider.tsx + 第八节 144 |
| slider 元素沿轨道方向比可见轨道多 18(两端各 9 = 滑块半径 6 + ring 3, 见第八节 26); 水平 `cx` / 垂直 `cy` 是**可见轨道长度**, 滑块中心在轨道两端之间移动; 垂直默认轨道 160(radix `min-h-40`), 值从下往上 | slider.tsx(radix 滑块中心贴轨道端点) + 第八节 144 |
| slider 交互全自实现:点击跳转(按在滑块上不跳值、按轨道把最近滑块跳过去)+ LButtonDown 起 `XEle_SetCapture` 拖拽 + MouseStay/MouseLeave 自跟踪 hover(拖完移开光环即消失, 与 shadcn 一致)+ 方向键(水平 ←→ / 垂直 ↑↓; 连续 ±5%、有 step 时一步)/PageUp·PageDown(10 步)/Home/End; `inverted` 反转取值方向; 禁用 = `SetDisabled`(鼠标与键盘一起关); `OnValuesChange` / `OnValuesCommit` 对应 onValueChange / onValueCommit | — + 第八节 144 |
| tabs 列表 rounded-lg p-[3px] h-8 bg-muted,标签等宽 rounded-md,未选 text-foreground/60、选中 bg-background(暗色 `bg-input/30 + border-input`, 否则白色卡片在暗色列表上变成一块更黑的方块);line 变体透明列表 + **gap-1** + 底边 h-0.5 下划线(换行时贴**当前行**底边);标签 text-sm **font-medium(500)≈中文常规粗细,勿用 Bold(700)**(实测对比);行内图标 size-4 + gap-1.5 + `has-data-[icon=inline-start]:pl-1`;单标签禁用 = 文字/图标 50% 预混 + 不响应点击/方向键;focus-visible 环画在内部焦点标签上(整块列表一个 Tab 站点, ←/→/↑/↓ + Home/End + Enter/Space, 见第八节 155) | tabs.tsx + 第八节 155 |
| tooltip = bg-foreground rounded-md px-3 py-1.5 text-xs text-background + size-2.5 旋转方块箭头(**只画露在气泡外的三角**, 见第八节 89/156),延迟 0、移出后 300ms 内跳过延迟;四方向(side)+ align + sideOffset, 放不下自动翻对侧并夹进工作区;内容可带右侧快捷键块(Kbd, 此时 pr 收到 6);键盘焦点进入也显示, 鼠标点击产生的焦点不显示;窗口为异型透明,气泡外真透明 | tooltip.tsx + 第八节 156 |
| select 触发区 h-8 rounded-lg border-input **bg-transparent(放卡上 = Card 色)** + 右侧 chevron-down 用 **text-muted-foreground 全色**(非前景半透明);触发区对滚轮无响应(`EnableEvent_XE_MOUSEWHEEL(false)`);弹层 = **item-aligned 定位**(盖住触发元素, 选中项正对触发位置, select.tsx `position="item-aligned"`);弹层条目 = p-1 容器 + 条目 h-7(28) rounded-md hover:bg-accent + 选中项右侧 Check 图标 + 打开时自动高亮选中项;ring-1 描边画在 `XE_PAINT_END`(内缩 1px, 见第八节 30);内容高度自适应(≤10 项不出滚动条);**弹层不做进场淡入**(上游是 fade-in-0 100ms, 整窗 alpha 在这个异型窗上会概率性冻结 → 弹层看不见还吞点击, 见第八节 30/69);**aria-invalid** = 红边框 + ring-3 destructive/20(暗色 /40), 见上表 | select.tsx + combobox.go(弹层接管路线见第八节 30) |
| menu 弹层 = **阴影弹窗**(rounded-lg 由阴影窗 nAngeleSize 逐像素画出 + shadow-md 投影, 官方 DrawMenu 同款, 见第八节 36/64) + bg-popover + ring-1 ring-foreground/10(内缩 1px); **p-1 = 菜单 SetBorderSize(4)**(默认 3); 进场**不做动画**(与原生菜单一致, 弹出即全内容 —— 整窗 alpha 会透出背景文字、只淡内容会先出一帧空白面板, 两种都闪, 见 99; 位移/缩放会崩, 见 64); 条目 h-7(28) rounded-md px-1.5, hover/focus bg-accent + 文字变 accent-foreground; destructive 红字 + hover bg-destructive/10(dark /20); disabled 文字 50% alpha 不响应; **前置图标 `MenuIcon`**(16px, left-6, 文字从 28 起, 同 shadcn 条目内联 SVG); 勾选/单选选中项右侧 Check 图标(right-2 size-4), **未选中也保留右侧 24(pr-8)**, 单选组内互斥; 分组标签 text-xs muted; 分隔栏 1px bg-border **全宽**(-mx-1; 原生槽高仅 4px, shadcn my-1 的 9px 不可达, 已知差异); 快捷键右对齐 text-xs muted; `MenuInset` 对普通条目 / `Label` / `Sub` 都可用(pl-7, 文字左缘 28); 弹层宽 = `Width(w)` 显式指定, 否则 max(触发区宽, 128)(shadcn `w-(--radix-dropdown-menu-trigger-width) min-w-32`); 无触发区的右键菜单 = 内容自适应 + 128 下限; 子菜单 = 内容自适应 + min-w-96; **所有条目宽度按自绘字体显式算**(原生菜单字体与自绘 14px 不一致, 留自然宽会裁长文本, 见第八节 99); **右键触发区 `NewContextMenu`** = 窗口级右键事件 + 触发区矩形命中(含子元素), `SetEnabled` / `OnOpenChange` | dropdown-menu.tsx + context-menu.tsx + menu.go(原生菜单皮肤, 见第八节 36/64/99) |
| table 表头 = h-10(40) px-2, 文字 foreground, 只有一条 1px 底边(border);数据行 = p-2 + text-sm(36), 行 `border-b`, **最后一行不画**;行悬浮 `hover:bg-muted/50`, 选中 `data-[state=selected]:bg-muted`;表体无外框/无圆角/无纵向网格线(外框交给 Card), 容器 `overflow-x-auto`, `caption-bottom text-sm`;自适应列宽按**元素当前宽度**算, 由组件订阅元素尺寸事件自愈(见第八节 60) | table.tsx + table.go(原生 List 接管路线见第八节 40) |
| tree 行高 32(h-8), 视觉矩形左右内缩 4(mx-1), 圆角 rounded-md(6);行悬浮/选中都 bg-accent(同 select/menu 条目);每级缩进 16(size-4), 祖先层级画 1px 参考线(border 色);chevron size-4 text-muted-foreground, 收起朝右/展开朝下(= rotate-90 两态), 叶节点不画但留空槽位(文字全树对齐);条目文字 text-sm foreground;shadcn 无官方 tree, 语义为社区通行实现 + 本库令牌 | App.tsx 手写 TreeRow 对照卡(无官方 tree.tsx) + tree.go(原生 Tree 接管路线见第八节 42) |
| pagination 入口居中(justify-center), 页码 32 方形(ghost / 当前页 outline), 上一页/下一页 = chevron(size-4)+文字, 到边界按禁用画(50% 透明且不响应鼠标), 页数 >7 折叠成 首/末页 + 当前页±1 + 省略号(不可点);本实现用**恒定 7 槽位**(见第八节 53);三种形态 = 完整 / 简洁(只有页码) / 仅导航(只有上一页下一页, 配"每页行数"选择器), 另有只留 chevron(对应文字上的 `hidden sm:block`);**键盘**: 整块一个 Tab 站点, 进入后 ←/→/↑/↓ 在条目间移动(跳过省略号与到边界的导航项)、Home/End 首末、Enter/空格激活, 到端点不循环, 焦点环只画在**当前条目**上且仅键盘获焦时出现(见第八节 152/153) | pagination.tsx + 第八节 53/152/153 |
| collapsible: 根容器竖向流(触发区 + 内容区); 触发区是普通按钮(可绑定任意 Button), 点击切换, 展开态走 Button 的 aria-expanded 视觉(bg-muted); 内容收起时隐藏且不占布局, 展开时高度按内容自撑; **本实现**展开/收起带 200ms 高度过渡, 收起动画结束后才隐藏内容(radix presence 语义) | collapsible.tsx + radix CollapsibleContent 的 `--radix-collapsible-content-height` 动画示例 + 第八节 97 |
| tooltip 绑定元素自身有弹层(如 ComboBox)时, 弹层打开要主动调 `HideTooltip(hEle)`——弹层窗口接管鼠标后触发元素收不到 MouseLeave | tooltip.go / combobox.go |
| scroll area: 页面/长内容滚动用 `ui.ScrollArea`(原生 LayoutFrame 载体, 见第八节 66): **只做纵向**(内容区恒铺满视口宽); 滚动条**鼠标移入容器才显示、离开淡出**(同 radix 的 `type=hover`, 150ms; 用元素透明度实现, 不改变布局宽度)、10px 轨道 + 8px 圆头滑块(`rounded-full bg-border`)、轨道透明; 滑块悬停**不换色**; 滚轮悬停任意子元素都滚动(原生控件如 List 自吃 —— 嵌套滚动常态); 该规格统一在 `ui/scrollbar.go` 的 `styleScrollBar` + `showScrollBarOnHover`, Table / Tree / ComboBox 弹层 / 多行 Input 同用(**滑块配色与容器不同**: 这四个载体在原版里是浏览器原生滚动条, 用"前景色淡化 + 悬停加深", 见第八节 21) | scroll-area.tsx(w-2.5 p-px `rounded-full bg-border`; radix `type` 默认 hover 淡入 —— 本实现的差异见"已知差异"表) |
| **disabled:opacity-50 是组级透明**:组件自绘的**形状**(壳体/轨道/勾/滑块/圆点)**逐层各乘 50%**(`ScaleAlpha`, 见第八节 61);**按钮文字/图标用"前景色×卡片色对半混"预混后绘制**(等于整组 50% 合成; 亮色页上禁用实心钮白字仍=255, 暗色下才会正确变暗);不要直接给文字加 alpha —— 文字叠在已半透明的壳体上会二次透出按钮底色。复选框/单选框的标签保持原色, 输入框禁用文字同样预混后交给原生 Edit。radio 的 Label 不跟随禁用 | button/checkbox/radio/tabs/switch/slider/input.tsx |
| **input 内嵌附加件**(input-group): 前置图标 size-4(16) / 前后缀文字(muted, 与输入同字号) / 清除钮(×)与密码显隐钮(眼睛) size-6(24) 幽灵钮 + 圆角 radius-5 + hover:bg-accent / 计数 `n/max` text-xs(12), 超上限变 destructive; 全部**自绘在壳体内部**, 不建子元素(子元素会被原生 Edit 的绘制盖住), 文字区左右内边距按附加件实际宽度算(`syncPadding`), 内容为空时清除钮不画但**留位**(打字过程中文字不左右跳) | input-group 示例 + 第八节 78/79 |
| input 步进器: 右侧 −/+ 内嵌按钮(size-6) + ↑/↓ 键 ±1 步 + 回车提交 + 失焦校正; 输入过程中允许任意文本(含负号/小数点), 只有步进/回车/失焦才解析 —— 超出值域夹取、不在刻度上吸附到最近刻度、解析失败回退最近一次有效值; 步长小数位决定显示精度(0.1 → "1.2", 不出现浮点尾巴); 到边界时对应按钮画成不可点 | 社区通行 stepper 语义(shadcn 无官方 stepper), 见 78 |
| input 密码遮挡沿用浏览器口径: 遮挡符是**实心圆点 (U+25CF)**, 明文/密文切换后焦点与光标留在末尾 | input.tsx(type=password) + 第八节 79 |
| dialog 面板 = w-full sm:max-w-sm(384) rounded-xl bg-popover p-4 gap-4 + ring-1 ring-foreground/10(ring 画在面板矩形**外侧** 1px, 直接描面板矩形会被遮罩灰吞掉, 见第八节 33);标题 text-base(16)/描述 text-sm(14) muted,header gap-2 | dialog.tsx |
| dialog 关闭钮 = 幽灵图标钮 size-7(28×28) top-2 right-2(X 图标 size-4);页脚 -mx-4 -mb-4 贴面板左右与底边、border-t、bg-muted/50、p-4、gap-2、右对齐(只有下两角随圆角) | dialog.tsx |
| dialog 遮罩 bg-black/10(固定黑色,不随主题;**只铺 Body 区, 不铺阴影窗口的阴影带**——铺上去会把窗口自身投影再压暗 10%, 四角最明显;**backdrop-blur-xs 已实现**——物理像素抓窗快照(PrintWindow 不缩放, DIB 必须给物理尺寸)+融合降采样+半分辨率盒式模糊+成对平均放大+BkM 图片填充,见第八节 47) | dialog.tsx(语义权威) + 第八节 47 条 6/10 |
| dialog 打开时焦点进第一个可聚焦控件,Tab/Shift+Tab 在对话框内循环(焦点陷阱),ESC / 点击遮罩 / 关闭钮关闭;标题/描述按面板宽折行(量高见第八节 90),`WithDialogScroll` 让内容区滚动、头部与页脚固定 | radix dialog 语义 |
| alert-dialog 面板 = max-w-xs(320;default 在 <640 断点也是 320,桌面 384) rounded-xl bg-popover p-4 gap-4 + ring-1 ring-foreground/10;标题 text-base(行高 24)、描述 text-sm muted,header gap-1.5(6);**没有右上角关闭钮** | alert-dialog.tsx(radix-nova) |
| alert-dialog 头部:default 左对齐;sm 居中(grid + place-items-center);media = size-10(40) rounded-md bg-muted + size-6 图标,destructive 用 bg-destructive/10 text-destructive(暗色 /20);default+media 时 media 与文本并排(media 跨两行、gap-x-4),sm+media 时 media 堆在标题上方(mb-2) | alert-dialog.tsx |
| alert-dialog 页脚 = 与 dialog 相同的 -mx-4 -mb-4 帯;default 右对齐一行,sm 是 grid-cols-2 两个等宽按钮;Action 默认 primary、Cancel 默认 outline(可改变体) | alert-dialog.tsx |
| alert-dialog 交互:点遮罩**不关闭**(radix onInteractOutside preventDefault),ESC 关闭,打开时焦点强制落在 Cancel(radix onOpenAutoFocus),Cancel/Action 点击后关闭;Tab 焦点陷阱 | radix alert-dialog 语义 |
| 多层模态(对话框里再开 AlertDialog):同一宿主窗口维护打开顺序的模态栈,**只有栈顶**响应 ESC / Tab / 遮罩点击;后打开的弹层 z 序更高,关闭上层后焦点还给下层最近聚焦的控件 | 本库实现(radix 用嵌套 portal + 各层 DismissableLayer,语义一致) |
| **中文 500 字重 ≈ 常规**: Label 默认常规、按钮文字常规(用粗体字形实现 500 会明显偏粗); 标签比描述"看着粗"是**颜色**差(foreground 近黑 vs muted 灰), 实测笔画同宽 | label.tsx 500 + 实测像素 |
| 令牌表 nova 预设 neutral 色板(oklch 已换算 hex) | index.css :root/.dark |
| **字体单位**: xcgui 字体 size 是 pt 不是 px, FontHandle 统一按 px×0.75 换算(14px→11pt); 按 px 直接当 pt 会让全局文字放大 1.33× | font.go(见第八节 31) |
| calendar 卡片 = bg-popover rounded-lg + ring-1 ring-foreground/10(popover 皮肤; 弹层里用 `WithCalendarFlat` 免双层边框); 格子 = `--cell-size` 方格(设计值 2rem=32) + rounded-md(6); **默认尺寸 278×327 = 设计基准 ×1.12**(格子 36 / 字号 16), 且所有几何(内边距/格子/标题/星期行/页脚/字号)都按**元素宽度等比缩放**, 高度决定 6 行之间的行距(多余空间均摊) —— 元素拉大时内容一起变大, 不会只剩空白; 标题行高 = 格高, 年月**两个热区**居中(font-medium + chevron-down, 悬停/展开 bg-accent), 左右各一个 ghost 翻页钮(size-8, hover bg-accent, chevron 16); 星期行 0.8rem muted; **固定 6 行**(月份切换卡片高度不跳, 弹层尺寸恒定); 选中 bg-primary + text-primary-foreground, 今天 bg-accent, 悬停 bg-accent(150ms 过渡), 非本月 text-muted-foreground; 周起始日默认周一(中文习惯), `WithCalendarWeekStart` 可调 | calendar.tsx(react-day-picker v9 的 classNames) + calendar.go |
| calendar 年月下拉 = 点标题热区弹出自绘列表(月 12 项 / 年 = 视图年±50 可配), 面板 bg-popover rounded-md + ring, 行高按"12 行塞满可用高度"自适应(16..26), 当前值右侧 check, 超出可视高度时滚轮滚动 + 2px 细滚动条, 面板内点击=取值、面板外点击=关闭、点另一个标题热区=直接切换; 只改显示月份不动选中日期(react-day-picker `captionLayout="dropdown"` 语义) | 参考截图里的 select 弹层形态 + calendar.go |
| calendar 禁用日期 = `disabled` matcher(内置禁用过去/未来/周末/日期数组 helper), 禁用格 text-muted-foreground opacity-50, **不可点、无悬停** | react-day-picker 的 `disabled` + `.rdp-disabled` |
| calendar 范围选择(mode="range")= 两次点击落 from/to: 第二次早于起点时**交换两端**(react-day-picker `addToRange`), 完整选中后再点 = 重新开始; 中段铺满格子的 bg-accent 色带 + 文字 accent-foreground(两端单侧圆角), 两端实心 bg-primary + text-primary-foreground; 起止同一天只画一个实心格; 只落了 from 时未完成区间仅高亮起点; 每次点击都回调 `WithCalendarOnRangeSelect(fn(from, to))`(to 为零 = 未选完); 跨月时色带在两个月块之间不连续(块间有 gap-4, 与上游"每月一张独立表格"一致); **显示月份恒对齐起点**(同上游 `defaultMonth={from}`)—— 区间比可见月数长时终点在窗口外, 翻页可看, 是上游同款行为而非缺陷 | react-day-picker v9 的 range_start/range_middle/range_end modifiers + CalendarDayButton 的 data-range-* |
| calendar 多月并排(numberOfMonths, 1~4)= 翻页按钮在**整块**左右两端(shadcn 的 nav 横跨所有月份), 每个月各有自己的年月下拉; 下拉/翻月只作用于被操作的那个月(整块窗口一起平移); "非本月"按各自月份块判定; **选中/范围端点只画在本月格上** —— 同一天会同时出现在相邻块的 outside 区(如 9 月块的拖尾就是 10 月上旬), 按日期判定会让端点在隔壁块重复高亮(实测: 范围两端 + 隔壁块一份 = "3 个选中"), outside 重复格只画弱化文字; 宽度 = 月数×单月 + 月间距, 高度与单月相同 | calendar.tsx 的 months/month/nav classNames + numberOfMonths; outside 不重复画选中是本实现决策 |
| calendar 周数(showWeekNumber)= 每月左侧一列 ISO 周数(宽 = 一个格子, 列头留空), 行号取**该行里的周一**的 ISOWeek(周一开头的行就是行首; 周日开头的行 +1 天, 否则周日会被算进上一周); 周数列不参与命中; 默认宽度因此比单月多一格(`CalWidthForWeekNumbers`) | react-day-picker `showWeekNumber` + `WeekNumber` 组件(w-(--cell-size)) |
| calendar 预订日期(booked)= `WithCalendarStrike(fn)` 给日期画删除线(helper `CalendarStrikeDates`); **被划线的格子不再叠禁用态的 50% 变淡** —— shadcn booked 示例 = disabled + `opacity-100` 的 line-through, 若照搬"禁用整格 opacity-50", 半透明灰线压半透明灰字等于什么都没画(实测不可见); 划线不影响可点性, 需要同时禁用请再传 `WithCalendarDisabled` | shadcn calendar 的 booked dates 示例(modifiers + modifiersClassNames line-through) |
| calendar 页脚 = 卡片全宽 1px border-t + 上下 6px + 24px 按钮行, 项水平居中、间距 8, **一行放不下自动换行**(高度跟着长); 按钮 ghost 化(muted-foreground → 悬停 bg-accent + accent-foreground), 纯文本项只有文字; 元素未指定高度时启用页脚自动长高 | shadcn calendar 的 footer 示例(含 flex-wrap 的 Presets)+ `WithCalendarPresets` |
| date picker 触发区 = **Button variant="outline"**(不是 Input): h-8 rounded-lg border-input bg-background, 左侧 size-4 calendar 图标 + 文字, hover bg-accent / 文字 accent-foreground; 未选择显示占位符(muted); 弹层 = 自建阴影窗 + Calendar(sideOffset=4, 下方放不下翻上方, 夹在**显示器工作区**内 —— 扣掉任务栏, 见第八节 74); 进场 fade-in 100ms; 选中即回写+关闭+回调, 点外部(弹层失焦)/ESC 关闭; **点触发区关闭后 250ms 内不再重开**(弹层 KillFocus 先于触发区 LButtonUp, 否则表现为"点两次才关掉"); aria-invalid 同 Button | date-picker 文档示例(Button outline + Popover + Calendar) + datepicker.go / popup.go |
| date picker 范围/预设/可输入 = `WithDatePickerRange`(弹层并排两个月; 第一次点击落起点**保持打开**, 补上终点才关闭; 触发区显示 "起点 - 终点", 未完时只显示起点; `Range()/SetRange/OnRangeChange`)、`WithDatePickerPresets`(弹层底部预设按钮, 范围模式一次落两端并关闭)、`WithDatePickerYearRange/WithDatePickerDisabled`(转给内嵌 Calendar); **可输入形态** `WithDatePickerEditable` = InputGroup: 文字区是原生 Edit(宽度跟触发区走, 见第八节 93), 右侧 24px ghost 日历钮打开弹层, Enter/失焦解析回写(`ParseDateText`: 2026-10-22 / 2026年10月22日 / 今天/明天/后天/大后天 / N天后 / N周后), ↓ 键打开弹层; 范围模式不支持可输入(自动退回普通触发区) | date-picker 文档的 Range / Input 示例 + calendar 的 Presets 示例 + datepicker.go |
| time picker = 触发区同 date picker(clock 图标 + "09:30") + 弹层两列滚动列表(时 0-23 / 分 0-59): 每列 64 宽、8 项可见(28×8), p-1 内边距; 载体是**原生 ListBox**(滚动/滚轮/点击都是原生), 条目背景与文字自绘 —— 选中 bg-primary + text-primary-foreground, 悬停 bg-accent(150ms 过渡); **选择即生效但弹层保持打开**(继续改另一列), 点外部/ESC 关闭; 打开时选中项滚到列表中间(行步进 = 项高+1, 见第八节 72) | shadcn 无官方 time-picker, 语义取社区通行形态(Popover + 滚动列表) + timepicker.go |
| toast(sonner) = 默认右下角堆叠(新的在最下), 位置六种: 四角 + `ToastTopCenter`/`ToastBottomCenter`(居中时左右按 Body 居中); 卡宽 356、与内容区边缘 24、卡间距 10; 卡片 bg-popover + border + rounded-lg + shadow-md, 内边距 16, 图标 20(与文字间距 10), 标题 13px font-medium / 描述 13px 可折行, 动作按钮 h-26 bg-muted hover:bg-accent, **关闭按钮默认隐藏**(同 shadcn 的 sonner), 显示时在右上角; 类型(success/error/warning/info)默认只改图标与图标色, `WithToastRichColors()`(或单条 `ToastOptions.RichColors`)开启 sonner 的 richColors: 卡面/描边/标题描述文字按类型上色(亮/暗两套色值); 自动关闭 **4s**、**悬停任一条暂停全部倒计时**、超出可见条数(默认 3)移除最旧; 进场淡入 + 从堆叠方向滑入 12px(200ms), 退场淡出 + 向侧边(居中时向上/下)滑出 24px(180ms); 类型快捷入口 `Success/Error/Warning/Info`, `Duration < 0` = 常驻 | sonner 默认值(宽度/offset/gap/4s/pauseTimers/visibleToasts=3) + shadcn 的 sonner.tsx(theme 交给令牌; 不传 closeButton) + 第八节 75 |

| item = 容器 rounded-md + border-transparent、text-sm;default gap-4(16) + p-4(16),sm gap-2.5(10) + px-4(16)/py-3(12);variants = 透明 / border-border / bg-muted/50;中间行 = Media + Content(吃剩余宽) + Actions,Header/Footer 占整宽;Media icon = size-8(32) rounded-sm border bg-muted + 16 图标,Content gap-1(4);**Header/Footer 的 justify-between 用 `xui.ItemWeight` 撑开第一个子项**(布局没有 justify-between);动作之间用 Item 的行间距,不是上游 gap-2 | item.tsx + item.go(Content 宽度见第八节 104) |
| breadcrumb = 列表 text-sm muted-foreground、桌面间距 sm:gap-2.5(10)、flex-wrap;条目内 gap-1.5(6);当前页 foreground;分隔符 chevron size-3.5(14);省略号 size-9(36) 居中 size-4 图标;链接 hover 过渡到前景色 | breadcrumb.tsx + breadcrumb.go(文本自绘的原因见第八节 105) |
| kbd = h-5(20) min-w-5(20) rounded-sm bg-muted px-1 text-xs font-medium muted-foreground,宽度随文字;KbdGroup 间距 gap-1(4) | kbd.tsx |
| skeleton = bg-accent rounded-md + animate-pulse 2s(1 → 0.5 → 1);脉冲相位由 `xui.Pulse` 提供,不自己起定时器 | skeleton.tsx + anim.go |
| spinner = Loader2 size-4(16) animate-spin 1s/圈;相位由 `xui.Spin` 提供 | spinner.tsx + anim.go |
| toggle = inline-flex items-center gap-2(8) rounded-md text-sm;default 透明底, hover bg-muted + muted-foreground, 选中 bg-accent + accent-foreground;outline 加 border-input, hover/选中填充 accent;尺寸 h-9/h-8/h-10(36/32/40) + min-w 同高、px-2/1.5/2.5;focus-visible border-ring + ring-3 ring-ring/50;aria-invalid 红边框 + 红环;disabled opacity-50 | toggle.tsx(radix-nova) |
| toggle group = 横向 flex w-fit、间距 spacing;spacing=0 时条目 px-3 + min-w-0、首项只保留左外角、末项只保留右外角(分段控件), outline 的相邻边框合并;单选同组互斥且可全部取消, 多选各自独立 | toggle-group.tsx(radix-nova) |
| popover = bg-popover + 1px ring(foreground/10) + rounded-lg + shadow-md, 默认 w-72(288) + p-4(16), sideOffset=4; 点击 trigger 开合, 点外部(失焦)/ESC 关闭, ESC 后焦点回触发区; 下方优先, 放不下自动翻到上方/左右, 并夹在显示器工作区内; 内容高度按子项自撑, 也支持固定高; 进场只淡入 | popover.tsx(radix-nova) |
| hover card = 同 popover 的面板外观但默认 w-64(256) + p-4(16); 悬停 trigger 700ms 打开, 移出 trigger/卡片 300ms 关闭; 卡片可交互且**不抢焦点**(非激活窗口), 指针还在卡片上时关闭检查续期; 不做键盘聚焦打开 | hover-card.tsx(radix-nova) |
| avatar = 圆形 sm 24 / default 32 / lg 40; 图片居中裁剪, 无图 fallback bg-muted + muted-foreground(默认 14 / sm 12); AvatarBadge 在右下角(size 8/10/12 + 2px 背景色环 + primary 底), AvatarGroup 用 `-space-x-2`(负 8)叠放、组内头像带 2px 背景色环, 计数头像同尺寸 | avatar.tsx(radix-nova) |
| command = 搜索行 h-9(36) + 下边框 + size-4 搜索图标(opacity-50) + 透明输入框; 条目 h-8(32) px-2 rounded-sm, 选中 bg-accent + accent-foreground; 分组标签 text-xs muted + p-1; 列表 max-h-300; 空状态 py-6 居中; 输入即过滤, ↑↓ 循环并跳过禁用, Home/End, Enter 选中, ESC 清空; 选中不自动关闭(CommandDialog 由 Dialog + Command 组合) | command.tsx(cmdk) |

### 变体/尺寸与上游的映射、已知差异

对照 `shadcn/src/components/ui/*.tsx`。**令牌名**要求与 CSS 变量逐字符对应(见 3.1),**组件变体名**是 Go API、不参与批量转译, 允许更直白, 但差异必须记在这里:

| 组件 | 映射 / 实现 | 说明 |
|---|---|---|
| Button variant | `ButtonPrimary` = shadcn 的 `default`(用 primary 色), 另有 `ButtonDefault` 别名 | 上游把这个变体叫 `default`; 其余 secondary/outline/ghost/destructive/link 同名 |
| Button size | `ButtonSizeXs/Sm/Default/Lg` = h-6/h-7/h-8/h-9(24/28/32/36); `ButtonSizeIcon/IconXs/IconSm/IconLg` = size-6/7/8/9 方形 | 与上游 8 档一一对应(像素实测吻合); 方形档 cx 传 0 即用预设边长; xs/sm/icon-xs/icon-sm 在按钮组内统一成 rounded-lg(见 86) |
| Button 行内图标/触发区态/加载 | `WithButtonLeadingIcon` / `WithButtonTrailingIcon`(对应 `data-icon=inline-start/end`, 间距 4/6、图标侧内边距收窄为 6/8); `WithButtonAlign(ButtonAlignStart/Between)` 对应示例里加在 Button 上的 justify-start / justify-between(整宽触发区文字靠左、尾随图标贴右); `WithButtonRounded` = rounded-full; `WithButtonExpanded` = aria-expanded; `WithButtonHasPopup` = aria-haspopup(按下不下沉); `WithButtonLoading` = LoaderCircleIcon + animate-spin(1s linear) | 上游的 `has-data-[icon=...]` / `aria-expanded` / `active:not-aria-[haspopup]` / spinner 示例;Go 侧无 asChild/Slot |
| ButtonGroup | `NewButtonGroup(hParent, WithButtonGroupOrientation(...))` + `NewButtonGroupText` + `NewButtonGroupSeparator`; 组宽高按内容自撑, 首尾外角、嵌套组 8px 间距 | 对应 button-group.tsx 的 `ButtonGroup/ButtonGroupText/ButtonGroupSeparator`(role=group);不适用部分同下表的 Slot/data-* 钩子 |
| Badge variant | default/secondary/outline/destructive/ghost/link 六档齐全 | 悬停样式恒生效(**shadcn 写作 `[a]:hover:*`, 只有包在链接里才生效**; Go 侧没有链接概念) |
| Badge 悬停 | `ghost` = hover 出 bg-muted, `link` = hover 加下划线 | 下划线手绘(无下划线标志位, 见下方"下划线") |
| Card | `WithCardSize(CardSizeSm)` = `--card-spacing` 4→3(16→12);`card.Footer(0)` 创建内边距跟随尺寸、内容已垂直居中的页脚;`card.TitleSize()` 返回标题字号(16 / sm 14) | 标题字号上游是 `text-base` + `font-medium`, 中文下 500 回落常规 → `NewCardTitle` 用 16 常规(不是粗体); sm 卡片标题要自己 `SetFontSize(card.TitleSize())`(构造器拿不到所属卡片); 页脚内容要放进 `footer.Content()`(那一层才有内边距) |
| Collapsible | `NewCollapsible(x, y, cx, cy, parent, WithCollapsibleOpen/Disabled/Duration)` + `Content()` / `ContentStack(space)` + `BindTrigger(*Button)`; 状态 `Open/SetOpen/Toggle/OnOpenChange` | 对应 collapsible.tsx 的 `Collapsible/CollapsibleTrigger/CollapsibleContent`; 根容器固定竖向流(上游根是任意 div); 内容高度带 200ms 过渡(上游只暴露 CSS 变量, 动画要自己加类); 示例根节点上的 `data-[state=open]:bg-muted` 未内置, 需要时在外面包一层带底色的容器 |
| AspectRatio | `NewAspectRatio(x, y, cx, cy, ratio, parent)` + `SetRatio` / `Ratio` / `Content()` / `ContentStack(space)` | ratio = 宽 / 高; `cy > 0` 或布局项高度声明成 Fill/Weight/Fixed 时由调用方管高度, 比例不生效(同 CSS 宽高都确定时 aspect-ratio 失效); 上游的 `className`(圆角/底色)映射为内容区自绘或元素背景, 组件本身不画任何东西 |
| Alert | `NewAlert(..., WithAlertVariant/WithAlertIcon/WithAlertTitle/WithAlertDescription)` + `Action()`; 标题/描述由组件内部持有 | 对应 alert.tsx 的 `Alert` + `AlertTitle/AlertDescription/AlertAction`; Go 侧是整体自撑高度的组件; 需要自定义文字样式时用 `Title()` / `Description()` 改字体后 `Refresh()`; 动作槽按内容宽(上游固定预留 pr-18=72, 本实现由动作实际宽度缩短文本列) |
| Empty | `NewEmpty(x,y,cx,cy,parent)` + `Header()` / `Content()` / `Body()`;`NewEmptyMedia(parent, icon, WithEmptyMediaVariant(...))`、`NewEmptyTitle`、`NewEmptyDescription` | 对应 empty.tsx 的 `Empty/EmptyHeader/EmptyMedia/EmptyTitle/EmptyDescription/EmptyContent`; media 变体 default/icon, default 用 24 图标、icon 用 40 圆角 bg-muted + 24 图标; 不画边框(上游默认 `border-dashed` 无宽度, 没有可见边框), 需要边框放进 Card; 上游 md:p-12 响应式内边距未做(统一 p-6) |
| Field | `NewField(..., WithFieldOrientation/WithFieldInvalid)` + `Label/Title/Description/Content/SetInvalid`;`NewFieldSet/FieldGroup/FieldLegend/FieldSeparator/FieldError` | 对应 field.tsx 的 `Field/FieldSet/FieldGroup/FieldLegend/FieldLabel/FieldTitle/FieldDescription/FieldSeparator/FieldError`; 无 responsive 取向与 DOM 级 label htmlFor; 横向 `Content` 默认 240 宽(嵌套布局只认 Fixed 宽, 见第八节 101), 用 `xui.ItemWidth` 覆盖; `FieldError` 单条错误保持单行(Label 不自撑折行高), 多条去重并用 "•" 列表 |
| Dialog | `NewDialog(hWnd, WithDialogWidth/WithDialogCloseButton/WithDialogScroll)` + `SetTitle/SetDescription/AddFooter/BindTrigger/BindClose/Content/ScrollArea/Refresh` | 对应 dialog.tsx 的 `DialogContent` + `DialogHeader/Title/Description/Footer/Close/Trigger`; 无 compound 子组件拆分, 头部/页脚由组件内部持有; `WithDialogScroll` 对应文档的 Scrollable Content / With Sticky Footer; 缩放动画不做(见第八节 47) |
| AlertDialog | `NewAlertDialog(hWnd, WithAlertDialogSize/WithAlertDialogTitle/WithAlertDialogDescription/WithAlertDialogMedia/WithAlertDialogMediaTone)` + `AddCancel/AddAction/BindTrigger` + `CancelButton/ActionButton` | 对应 alert-dialog.tsx 的 `AlertDialogContent` + `Header/Title/Description/Footer/Media/Action/Cancel/Trigger`; `size` 是枚举 `AlertDialogSizeDefault/Small`; 媒体块由组件自绘(不建子控件); `AddCancel` 固定 outline、`AddAction` 的变体由调用方给(可用 `CancelButton().SetVariant(ButtonGhost)` 改变体) |
| Separator | `WithSeparatorVertical` 支持垂直 | 上游 `data-vertical:w-px` |
| Tabs | `NewTabsItems` / `TabsItem{Text,Icon,Disabled}`(行内图标与单标签禁用) + `WithTabsDisabledAt` / `SetDisabled` / `SetIcon` / `SetText`; `WithTabsVertical` 垂直(标签堆叠、左对齐、指示条在右侧); `WithTabsWrap` 横向换行(上游没有: 每行均衡、高度随行数增加、line 下划线贴当前行底边); `WithTabsActivation`/`WithTabsManual` 对应 `activationMode`; `WithTabsValues` + `ActiveValue`/`SetActiveValue`/`OnValueChange` 对应受控 value; `BindPanel` 按选中项显隐面板(对应 TabsContent 的显隐); 键盘: 整块一个 Tab 站点, ←/→/↑/↓ + Home/End + Enter·空格(第八节 155) | 上游 `data-[variant=line]:*` / `data-[orientation=vertical]:*` / `activationMode`; 换行与 `BindPanel` 是本库扩展; 未做 `forceMount` 与拖动滑块式指示器 |
| Tooltip | `Bind` / `BindTip(hEle, text, opts...)` + `WithTipSide`(上/下/左/右, 默认自动) / `WithTipAlign`(start/center/end) / `WithTipOffset`(sideOffset) / `WithTipDelay` / `WithTipShortcut`(内容右侧 Kbd) / `WithTipMaxWidth`; 全局 `SetDelay` / `SetSkipDelay`(默认 0 / 300ms); 受控 `Show` / `Hide` / `IsOpen` | 上游 `side` / `align` / `sideOffset` / `delayDuration` / `skipDelayDuration`; 上游一个 Tooltip 是"一个 trigger + 一个 content", 本库是**一个管理器绑定任意多个元素**(每个元素可各自配置方向/延迟), 因为桌面端控件是原生子元素; `TooltipProvider` 的共享延迟语义收进 `SetDelay`/`SetSkipDelay`; 未做 `disableHoverableContent` 与箭头 `arrowPadding` 单独配置 |
| Toggle | `ToggleVariantDefault/Outline`、`ToggleSizeDefault/Sm/Lg` = h-9/h-8/h-10(36/32/40)、`WithTogglePressed/Icon/Invalid` + `SetPressed/SetIcon/SetInvalid/OnChange` | 对应 toggle.tsx 的 variant/size; 组内条目按 px-3 + min-w-0 量宽(ToggleGroupItem 的覆盖); 原生 Check 按钮不响应空格/回车, 键盘切换由组件在 KeyDown 里补(与鼠标共用 OnChange); shadow-xs 不绘制(同其他自绘壳体) |
| ToggleGroup | `NewToggleGroup(hParent, WithToggleGroupVariant/Size/Spacing/Multiple)` + `AddItem(value, text, opts...)` + `Value/SetValue/OnChange/SetSpacing` | 对应 toggle-group.tsx; 单选(type=single)允许全部取消(radix 语义), 组级 variant/size 覆盖条目自己的同名选项(context.variant || variant); spacing 用逻辑像素(上游 spacing={2}=8px); 上游当前源码没有 orientation, 只做横向 |
| Slider | `WithSliderRange(min,max)` / `WithSliderStep(step)` | 对应 radix 的 min/max/step; 默认仍是 0~100 连续 |
| ComboBox 触发区 | `WithComboSize(ComboSizeSm)` = h-7 + `rounded-[min(--radius-md,10px)]` | 对应 select.tsx 的 `data-[size=sm]` |
| Calendar | `WithCalendarSelected/WeekStart/OnSelect` + `WithCalendarFlat`(弹层内嵌不画卡片); 尺寸 `CalWidth/CalHeight`(单月 278×325) + `CalFooterHeight`, 自定义宽度时用 `CalHeightFor(width)` / `CalFooterHeightFor(width)` 配高度(或 cy 传 0 自动); 年月下拉 `WithCalendarYearRange(from,to)`; 禁用 `WithCalendarDisabled(fn)` + `SetDisabled` + helper `CalendarDisableBefore/After/Weekend/Dates/Any`; 划线 `WithCalendarStrike(fn)` + `SetStrike` + `CalendarStrikeDates`; 页脚 `WithCalendarFooter(items...)` / `WithCalendarFooterToday` / `WithCalendarPresets(CalPreset(label,days,to)...)` / `AddFooterItem/Button/Text` / `ClearFooter` / `FooterRect`; **范围/多月/周数**: `WithCalendarRange` + `WithCalendarSelectedRange(from,to)` + `WithCalendarOnRangeSelect` + `SelectedRange/SetRange/RangeMode`, `WithCalendarMonths(n)`(1~4), `WithCalendarWeekNumbers`; 尺寸 helper `CalWidthFor(months)` / `CalWidthForWeekNumbers(months)` / `CalHeightForMonths(width,months)` / `CalFooterHeightForMonths` | react-day-picker 的 props(mode/numberOfMonths/showWeekNumber);固定 6 行 + 等比缩放 + 页脚换行是本实现决策 |
| DatePicker / TimePicker | `WithDatePickerPlaceholder/Value/Invalid` / `WithTimePickerPlaceholder/Value/Invalid` + 运行时 `SetInvalid`; 程序化开关 `Open/Close/IsOpen`; DatePicker 另有 `Clear/SetValue/Text`; **范围**: `WithDatePickerRange` + `WithDatePickerRangeValue` + `Range/SetRange/OnRangeChange/RangeMode`; `WithDatePickerPresets` / `WithDatePickerYearRange` / `WithDatePickerDisabled` / `WithDatePickerEditable`(可输入触发区, 配 `ParseDateText(s)` 解析) | 选项名带组件前缀(组件同在 package ui); 弹层骨架 `popup.go` 是两个组件的内部实现, 随组件一起拷贝 |
| 校验失败态(aria-invalid) | `WithButtonInvalid` / `WithInputInvalid` / `WithComboInvalid` + 运行时 `SetInvalid(bool)` | 选项名带组件前缀, 因为组件同在 `package ui`(Button/Input/ComboBox 三处语义相同); 详见语义表 |
| Input 内嵌附加件 | `WithLeadingIcon` / `WithPrefixText` / `WithSuffixText` / `WithClearButton` / `WithPasswordToggle`(配合 `SetPasswordVisible(bool)` / `PasswordVisible()`) / `WithCounter(max)`; 步进器 `WithStepper(min,max,step)` + `SetNumber/Number/OnStep`(`WithStepperArrows` 换成上下三角 = 原生 number 输入框形态, `WithStepperAutoHide` 让按钮只在悬停/聚焦时出现; 悬停底色/按下保持底色 + 箭头下沉 1px, 与 −/+ 一致) | shadcn 的 input-group 是"输入框 + 兄弟 addon"的复合结构(`InputGroupAddon` 里还能放按钮), 我们是**壳体内部自绘**的简化形态(不建子元素): 前置图标 + 前后缀文字 + 两个 ghost 钮(清除/眼睛) + 计数; 计数放在壳内右侧(上游示例是输入框**下方**右对齐的一行); stepper 上游没有官方组件, 语义取社区通行形态; 显隐切换需重排(见第八节 84) |
| Card 头 | `ui.NewCardHeader(parent, space)` + `TitleStack` / `ActionStack` | 对应 `CardHeader` + `CardAction`(上游是 grid 1fr_auto); 动作区跨两行贴右上 |
| ScrollArea | **本库形态**: 原生 LayoutFrame 载体 + 主题化滚动条(10px 轨道 + 8px 圆头滑块, 与 Table/Tree/ComboBox/多行 Input 共用 `scrollbar.go` 的统一收敛); 显隐同 radix 的 hover 淡入(150ms 淡入淡出); 只做纵向(上游 ScrollBar 有 horizontal 档); 隐藏页恢复可见后要延迟重排一次页面, 否则子控件命中区停在旧滚动位置 | 页面滚动容器; 语义与踩坑见第八节 66 |
| Menu 条目缩进 / 图标 / 右键触发 | `MenuInset` = shadcn 的 `inset` prop(`data-inset:pl-7`, 文字左缘 28; 条目/Label/Sub 都支持); `MenuIcon(path)` = 条目内联前置图标(16px, left-6); `NewContextMenu(hEle, build)` = `ContextMenu` + `ContextMenuTrigger`(右键触发区, 每次右键重建菜单, `SetEnabled`/`OnOpenChange`) | inset 用于和带前置图标的条目文字对齐; 高亮底仍是整行; 右键触发走窗口事件做区域命中(元素右键不冒泡), 见第八节 99 |
| NativeSelect | `NewNativeSelect(x,y,cx,cy,hParent, values, WithNativeSelectPlaceholder/Size/Value/Invalid)` + `Value/SetValue/OnValueChange` | 对应 native-select.tsx(浏览器原生 `<select>` + 外壳): 弹层完全是原生 ComboBox 弹窗(与 NewComboBox 一致), 组件只补 `value` 字符串语义(`<option value>` / `onChange(e.target.value)`); 未选中时 Value() 返回空串, SetValue 匹配不上会取消选中 |
| Select | `NewSelect(..., WithSelectPlaceholder/Size/Items/Groups/Value/Invalid/Disabled)` + `Value/SetValue/Selected/SetItems/SetGroups/OnChange/Open/Close/IsOpen`; 条目 `SelectItem{Value,Text,Icon,Disabled}`、分组 `SelectGroup{Label,Items}` | 对应 select.tsx(radix, item-aligned): 弹层盖住触发区、选中项正对触发区(两种盒子高度按**中心**对齐, 同 ComboBox 弹层的推导); 宽 = max(触发区宽, min-w-36); 键盘 ↑↓(跳过禁用)/Home/End 移动高亮(焦点即高亮 bg-accent), Enter/空格 选中, ESC 关闭并把焦点还给触发区; 打开即把焦点放到选中项(radix 语义); 点击选中同值也通知(同 radix onValueChange); 与 ComboBox 的取舍: 弹层自持(可富条目), 原生弹窗行为不复刻 |
| InputOTP | `NewInputOTP(..., WithInputOTPLength/Pattern/SeparatorAfter/Value/Invalid/Disabled)` + `Value/SetValue/Clear/Focus/SetInvalid/SetDisabled/OnChange/OnComplete`; 预置过滤 `InputOTPPatternDigit/Alphanumeric/Any` | 对应 input-otp.tsx(底层 input-otp 包): 键入打进已填格 = **替换**并前进一格(同上游), 非法字符不入格; ←/→/Home/End 移动, Backspace 清当前格、空格时退格清前一格, Delete 清当前格; Ctrl+V 从当前格起分发粘贴(非法字符跳过, 焦点停最后写入格); 全满触发 OnComplete; 文字居中由原生 `SetTextAlign(Center|Center_V)` 负责(不要用 border 挤位置, 见第八节 110), 只有空格子用对称 border 把原生 caret 顶到格心; 上游的"跨格拖动选择"没有对应物(每格独立 Edit) |
| Accordion | `NewAccordion(..., WithAccordionType(AccordionSingle/AccordionMultiple))` + `AddItem(value, title, WithAccordionItemDisabled)` + 条目 `Open/SetOpen/Toggle/SetDisabled/Content/ContentStack/Trigger`; 根 `Value/SetValue/OnChange` | 对应 accordion.tsx(radix): 触发区文字左对齐(上游无水平内边距; 我们留 4px 给内嵌环带, 见第八节 112)、箭头贴右、触发区高 40(py-2.5 + text-sm), hover:underline 手绘 1px 线(无过渡), focus-visible 环 rounded-lg; 箭头是 ChevronUp/Down 两枚图标互换(上游也是互换, 无旋转动画); 条目间 1px 分隔线(Item 的 not-last:border-b, 末项没有); 内容底部 pb-2.5 用末尾的透明占位表达(挂在开合时、高度扣掉行间距, 见第八节 111); single = 单开互斥且允许全部收起(radix type="single" collapsible), multiple 各自独立; disabled 只禁触发区, 已展开内容保留 |
| Popover | `NewPopover(hTrigger, WithPopoverContent(build)/Width/Height/Padding/Gap/Side/Align/OnOpenChange)` + `Open/Close/SetOpen/IsOpen/Content/Refresh`; `PopoverHeader/Title/Description` 助手 | 对应 popover.tsx 的 `Popover/PopoverTrigger/PopoverContent`(Header/Title/Description); 内容宽铺满、高自撑(空内容保留 p-4); 触发区可以是任意元素(Button 走 BnClick, 普通元素走 LButtonUp); 弹层窗口是自建阴影窗, 无 zoom/slide 动画 |
| HoverCard | `NewHoverCard(hTrigger, WithHoverCardContent/Width/Padding/Side/Align/Delay/OnOpenChange)` + `Open/Close/SetOpen/IsOpen/Content/Refresh` | 对应 hover-card.tsx; 默认 `openDelay=700ms` / `closeDelay=300ms`; 非激活窗口, 不抢焦点; 内容里的子元素都会挂 MouseLeave, 离开后按指针位置决定是否真正关闭 |
| Avatar | `NewAvatar(x,y,cx,cy,parent,fallback,WithAvatarSize/Image/ImageData/Ring/Badge)` + `SetImage/SetImageData/SetFallback/SetBadge`; `NewAvatarGroup(parent, WithAvatarGroupSize)` + `Add(fallback, opts...)` / `AddCount(n)` | 对应 avatar.tsx 的 `Avatar/AvatarImage/AvatarFallback/AvatarBadge/AvatarGroup/AvatarGroupCount`; 图片走本地文件或 `image.Image`(png/jpeg/gif), 圆形蒙版在 CPU 侧生成; 徽标画在头像内右下角(圆环内缩 2px, 避免被父元素裁剪) |
| Command | `NewCommand(x,y,cx,cy,parent, WithCommandPlaceholder/EmptyText/OnSelect/OnChange/OnEscape)` + `AddGroup(label)` / `AddSeparator` + `CommandGroup.AddItem(value,text, WithCommandItemIcon/Shortcut/Keywords/Disabled/OnSelect)` + `Focus/Query/SetQuery/Clear/SetPlaceholder` | 对应 command.tsx(cmdk) 的基础形态; 过滤为不区分大小写的包含匹配(可配 keywords), 不是 cmdk 的 command-score; 列表高度 = cy - 36(cy 传 0 = 336); `CommandDialog` 用 `ui.NewDialog` + `Command` 组合(无单独的 CommandDialog 类型) |
| Toast | `NewToaster(hWnd, ...)` + `Show(ToastOptions)`(标题/描述/类型/图标覆盖/动作/关闭按钮/时长/`OnClose/OnAutoClose/OnDismiss`) + `Success/Error/Warning/Info/Loading` 快捷方法 + `Promise(fn, ToastPromise{Loading, Success, Error})`; `WithToastPosition/Width/Offset/Gap/Duration/MaxVisible/RichColors/CloseButton`; 实例 `Update/Close/Handle` | sonner 的 `toast` API 的 Go 形态; 位置六种(四角 + `ToastTopCenter`/`ToastBottomCenter`), `richColors` 通过 `WithToastRichColors` / `ToastOptions.RichColors` 开启; `Loading`/`Promise` 的转圈图标由 `xui.Spin` 驱动(退出加载态要显式停表, 见第八节 154), `Promise` 的 fn 在 goroutine 跑、结果经 `xc.Auto` 回 UI 线程 |
| Pagination | **本库扩展**: shadcn 的 Pagination 是纯展示组件, 省略号要手拼; 我们做了自动折叠(第八节 53) | 见 53 |
| Pagination(形态/键盘) | `WithPaginationSimple()`(只有页码) / `WithPaginationNavOnly()`(只有上一页下一页) / `WithPaginationHideText()`(只留 chevron); 键盘 ←/→/↑/↓、Home/End、Enter/空格 | 见 152/153 |
| Resizable | `NewResizableGroup(x,y,cx,cy,parent, WithResizableVertical/Disabled/OnResize)` + `AddPanel(WithResizableSize/Min/Max)` + `AddHandle(WithResizableGrip)`; `Sizes/SetSizes/OnResize/SetDisabled/Disabled`; 面板是流式容器, 内容直接放进去 | 对应 resizable.tsx 的 `PanelGroup/Panel/Handle`; 未做 `collapsible/collapsedSize`(Enter 折叠)、`preserve-pixel-size`、F6 在分隔条之间跳焦点、面板/分隔条级 disabled; 尺寸只支持百分比(上游还能写 px/rem); 面板是容器而非独立元素, 没有 `data-panel` 之类的 DOM 钩子 |
| 不适用 | `asChild`(Slot)、`data-slot`/`data-state`/`data-variant` 钩子、compound 子组件拆分、props 透传 | React/DOM 形态, Go 侧没有对应物 |
| 令牌 | `--chart-1..5` / `--sidebar-*` 已进 `xui.Theme`(Chart1..5 / Sidebar*) | chart 色板供你的应用取用(组件库不画图表); sidebar 令牌由 `ui.Sidebar` 使用; 机械转译新主题时两组一并填(运行时的主题测试有守卫) |

**下划线(Button link / Badge link / Accordion 触发区)**: xcgui 没有下划线文本标志位(`TextFormatFlag_` 里没有), 字体侧 `FontStyle_Underline` 在 D2D 文本路径下也不渲染(见第八节 110), 只能手绘: 宽度用 `xui.TextWidth` 量, y = 行盒中心 + 0.3em + offset(`underline-offset-4` 用 4px; Accordion 的 hover:underline 没有 offset, 用行盒底边 - 2px), 1px 线(`button.go` / `separator.go` / `accordion.go`)。实测(150% DPI): "Link" 下方 1 条 43 物理像素宽的横线, 紧贴文字下沿。

---

## 五、安装与更新

```bash
xui init -n myapp        # 新建完整项目: go.mod + main.go + README + .gitignore + ui/,
                         # 并自动执行 go mod tidy
xui init                 # 已有项目: 只创建 ui/(theme.go + 基础组件 + 这份说明)

xui add button dialog    # 追加组件(自动带上依赖的组件)
xui add --all            # 拷贝全部组件
xui add card --dry-run   # 预览将创建/覆盖/跳过的文件, 不写本地
xui add card --view      # 打印 registry 源码(可加 =card.go 只看某个文件)
xui add card --diff      # 与本地文件做逐行 diff
xui update               # 更新已装组件(省略组件名 = ui/ 里已装的全部)
xui list                 # 列出可用组件
xui search -q table      # 按名称搜索(等价于 list -q table)
xui view button          # 打印组件源码
xui info                 # CLI 版本 / 工作目录 / ui 状态 / 数据源
```

- **默认不覆盖你改过的文件**:逐个比对内容 —— 本地没有则创建、内容一致则跳过、被改过则警告跳过;确认要丢掉本地改动时加 `-o/--overwrite`。这是 copy-not-install 的前提:代码归你。
- **组件目录**默认 `ui`, 可用 `add/update -p/--path <目录>` 指定;全局 `-c/--cwd` 先切换工作目录, `-s/--silent` 只打印警告与错误。
- **新建项目**:目标目录已存在且非空时需 `-f/--force`(只覆盖模板文件, 不动无关文件)。
- **依赖安装**:`xui init -n` 生成后默认在项目目录执行 `go mod tidy`(生成 `go.sum`);离线或想手动控制时加 `--no-install`, 之后自己跑一次。
- **数据源**:默认从仓库 `main` 分支拉组件(永远最新),自动在直连 / Gitee 镜像 / GitCode 镜像 / 加速代理之间选路;拷到一半某个文件拉不到(镜像限速、临时挂起)会自动换下一个源继续,不会留下半套 `ui/`。`XUI_REGISTRY` 可指向本地目录或任意 `http(s)://` 基址(离线开发 / 锁定组件版本),显式设置时只走你写的地址、失败直接报错(不静默降级)。
- **registry.json** 是组件清单:每个组件登记 `files`(要拷的文件,含跨文件助手)与 `deps`(依赖的其它组件)。CLI 只拷这些 `.go` 文件;运行时(`github.com/twgh/xui`)与图标子包(`github.com/twgh/xui/icons`)是 Go 模块依赖,不进 `ui/`。

---

## 六、ui/ 目录与组件依赖

`xui init` / `xui init -n` 之后你的项目里是:

```
ui/
  theme.go              主题令牌 —— 你的文件, 随意改
  DESIGN.md             本文
  button.go card.go …   组件源码(init 只带基础组件, 其余按需 add)
```

- 所有组件是**扁平的单包** `package ui`(一个组件一个文件),互相直接调用,没有子包。
- 几个**跨文件助手**会随依赖一起拷进来,不要删:

  | 文件 | 作用 | 被谁依赖 |
  |---|---|---|
  | `scrollbar.go` | 细滚动条皮肤 + 悬停显隐 | table / tree / combobox / input(多行) / scrollarea / dialog(可滚动内容) |
  | `popup.go` | 弹层骨架(定位、阴影、进出场、打开态重落位/光标判定) | datepicker / timepicker / select / popover / hovercard / menubar / navigationmenu |
  | `backdrop.go` | 弹层背景模糊(抓窗快照) | dialog |
  | `sheet.go` | 侧边面板(复用 Dialog 的模态基座) | dialog 的文件清单(模态分支) / drawer |
  | `menu_model.go` | 菜单条目模型与 `MenuOpt` 构造器 | menu / menubar |

- 组件依赖(registry 里的 `deps`):`card → label`、`dialog → button + label`、`alertdialog → dialog`、`combobox → tooltip`、`menu → tooltip`、`empty/field → label`、`item → label + separator`、`sheet → dialog`、`drawer → sheet`、`menubar → button`、`navigationmenu → button`、`carousel → button`;`add` 会自动补齐,不用手抄。`dialog` 的 `files` 还包含 `scrollarea.go` / `scrollbar.go`(可滚动内容用),`alertdialog` 共用 `dialog.go` 的面板/遮罩/模态栈。
- 图标不在 `ui/` 里:用 `github.com/twgh/xui/icons` 的常量(只是一条 SVG path 字符串,只把用到的打进二进制,体积增量 0)。
- 升级:`xui update` 或 `xui update button`(只更新该组件及其依赖)。

---

## 七、组件覆盖与已知限制

组件与 shadcn 的对应关系见第四节。以下差异是 **xcgui 能力边界**导致的,改组件代码也补不上:

| 差异 | 原因 |
|---|---|
| Input 的文字/光标/选区/IME 由原生 Edit 绘制 | 都在 DLL 内;只开放字体/字色/占位符色/插入符色/文字内边距。要完全控制就得自绘文字并接管光标闪烁、选区命中、IME 组合窗 |
| 英文长文本折行按字符断行(可能在单词中间换行) | Label 载体是原生 Button, 折行算法在 DLL 内, 不按单词边界; 中文不受影响(第八节 90/91) |
| Dialog 没有 zoom-in-95 / AlertDialog 没有 text-balance | 元素级缩放对自绘元素无效(第八节 47); 自绘文字没有平衡排版能力 |
| 没有 `asChild` / `data-slot` / `data-state` | React/DOM 概念, Go 侧没有对应物 |
| Card 的 `overflow-hidden` / 首末图片自动圆角、AspectRatio 的圆角裁剪 | xcgui **会按父元素矩形裁剪子元素**(探针实测, 见第八节 97), 但没有圆角裁剪/overflow-hidden 语义: 超出圆角的四角区域仍会画出来; 需要圆角裁剪的图片/内容请自己按路径裁 |
| `--chart-*` 色板只有 5 色, 且中性主题下是灰阶 | 上游同样是 5 个槽位循环; Chart 组件按分类序号取色, 超过 5 类会重复, 要区分请自己给足色板。灰阶下饼图百分比标签按片色亮度在正文色/背景色之间取对比更高的那个(第八节 159) |
| Chart 没有 HTML 的响应式容器/Tooltip 组件联动 | 尺寸完全由布局项决定(建议 `ItemFill` 或给定宽高); 悬停提示卡是元素内自绘的小卡片, 不是 `ui.Tooltip` 弹窗(不抢焦点、不新建小窗) |
| Calendar 不做 RTL(dir=rtl)/波斯·Jalali 历/timeZone/自然语言日期(chrono-node) | 文字渲染只有 LTR; 历法换算与时区数据是另一套领域库, Go 侧没有 chrono-node 的等价物 —— 绝对/相对日期的键入用 DatePicker 的可输入形态(`ParseDateText`)覆盖最常用场景; 范围色带跨月时在两个月块之间不连续(上游同样是每月一张独立表格) |
| Field 横向 `Content` 的嵌套列只认 Fixed 宽 | xcgui 不会把 Weight/Percent 宽度传给嵌套布局的 Fill 子项(实测子项宽恒为 0, 内容不显示); 组件默认 Fixed 240, 用 `xui.ItemWidth` 覆盖(第八节 101) |
| EmptyDescription / FieldDescription / FieldError 不自动折行 | Label 的高度不会随折行自撑; 需要多行时调用方用 `SetWrap` + `WrappedHeight` 量高后写回高度(同 Alert 的做法) |
| Item 的 Header/Footer 没有自动 space-between, 动作之间用 Item 的行间距 | Go 布局没有 justify-between: 用 `xui.ItemWeight` 给靠左的第一个子项撑开; 行内动作也不再单独套 gap-2 容器(嵌套 Auto 宽不量测原生按钮, 见第八节 104) |
| Breadcrumb 链接/当前页是单行自绘文本, 且不参与 Tab 焦点 | Label 的原生 Button 载体按 `TextWidth` 精确设宽会被内部文本内边距截断(第八节 105); 自绘后不折行、不支持富文本, 键盘焦点需要每链接一个可聚焦元素(当前为鼠标驱动) |
| 嵌套在布局容器里的控件(卡片里的按钮 / 分页 / 标签栏 / 分隔条等)无法用 Tab 聚焦 | xcgui 的 Tab 焦点链只覆盖窗口内容区的**直接子元素**: 直接挂在内容区的可聚焦 Element 可以 Tab 到, 套一层 LayoutEle/Element 容器后就被跳过(Button 也一样, `EnableFocus/EnableSwitchFocus/EnableKeyTab` 都补不上, 见第八节 157); 组件内的键盘操作需先用鼠标点击获得焦点 |
| Toggle / ToggleGroup 的 outline 不画 shadow-xs | 上游有 1px 浅投影; 本库自绘壳体统一不画阴影(Button/Input 同), 在 Card 底上几乎不可见; `border-input` 与悬停/选中底色完整保留 |
| InputOTP 没有"跨格拖动选择"/Ctrl+A 全选 | 每格是独立的原生 Edit, 选择语义无法跨格; 复制/剪切整段验证码也未做(粘贴方向完整) |
| Select 弹层没有 radix 的 scroll-up/down 悬停自动滚按钮与 typeahead(键入跳转) | 自建弹层用 ScrollArea 承载, 滚轮与 ↑↓/Home/End 都可用; typeahead 需要按键聚合计时, 后续有需求再加 |
| Popover / HoverCard 进场只淡入, 没有 zoom/slide | 自建阴影窗能稳定动画的只有元素 alpha; 整窗位移/缩放会闪或崩(第八节 47/64), 离屏测量与淡入都在第八节 115/116 |
| HoverCard 不支持键盘聚焦打开 | 非激活窗口不参与 Tab 焦点链; 需要键盘可访问时用 Popover 包一个可聚焦触发区 |
| HoverCard 内容可鼠标交互, 但非激活窗口不接收键盘焦点 | WS_EX_NOACTIVATE 不抢前台, 这是"悬停预览不打断输入"的前提; 需要键盘操作的内容用 Popover |
| Command 过滤是包含匹配, 不是 cmdk 的模糊评分; 没有 CommandDialog 类型 | 语义覆盖最常用的搜索/键盘选择; 命令面板弹窗用 `ui.NewDialog` + `Command` 组合 |
| Avatar 只接受本地图片或 `image.Image`, 不支持远程 URL/动图 | 圆形裁剪在 Go 侧生成 RGBA 蒙版; 远程下载与 GIF 播放属于应用层 |
| Sheet 面板没有 shadow-lg, 内容超出面板高时裁剪 | 窗口内的元素没有可用的圆角投影; 内容区高度是面板剩余空间, 需要滚动时在 `Content()` 里放 ScrollArea |
| Drawer 只能从抓手起拖, 且没有 vaul 的橡皮筋阻尼 | 内容区任意位置起拖会与文本选择/按钮点击冲突; 阻尼动画会增加一套每帧缓动, 当前用"跟手 + 松手回弹"覆盖核心手势 |
| Menubar 弹层超高不滚动 | 位置会夹进工作区, 但菜单本身没有滚动条; 需要滚动得自己控制菜单条目数量 |
| Menubar 没有 Alt 键访问与 typeahead | 原生 Windows 菜单的 Alt 导航挂在系统菜单栏上; 自绘触发区只覆盖 Tab/方向键这一套可访问路径 |
| NavigationMenu 内容里只有 `Link` 链接行支持键盘高亮/Enter | 视口是非激活窗口, 不接收键盘焦点: 键盘交互由触发区接管(↓ 进入内容后 ↑↓/Home/End/Enter), 自定义放进内容的控件只支持鼠标操作 |
| NavigationMenu 的链接说明只有一行(超宽省略号), 上游是 line-clamp-2 | 链接行高固定 68(标题 20 + 间距 4 + 说明 20 + 上下 12); 两行说明会让每个菜单的视口高度随文案跳变 |
| NavigationMenu 只做共享视口模式, 没有 Sub / viewport=false / Indicator | 每个触发区各开一块弹层可以用 Popover 组合; 视口下方的小箭头(Indicator)在 shadcn 默认示例里也不启用 |
| Carousel 高度必须显式给(构造参数 cy) | 网页里轮播高度由最高的一页撑开; 自绘容器没有可靠的离屏量高通道, 需要随内容长高时把高度交给外层布局控制 |
| Carousel 的按钮槽位固定占 96(左右/上下各 48) | 与 shadcn 示例外层 `px-12` 等价; 不需要按钮时用 `WithCarouselControls(false)`, 内容会占满整个根元素 |
| Carousel 拖拽按"最近一页"吸附, 没有 velocity 惯性 | 桌面鼠标够用; 触摸手势、RTL 与 wheel 翻页未做(embla 的这些能力不在 shadcn 核心弹层语义里) |
| Progress 的不确定态是"短条往返", 上游是空轨道 | shadcn 的 indeterminate 只暴露 `data-state`, 指示器平移到轨道外; 短条往返是"进行中但未知进度"的通行视觉(见第八节 145) |
| Slider 是一个焦点元素, 键盘只调整"当前操作滑块" | 桌面 Tab 惯例是整条滑动条一个焦点; radix 的每 thumb 独立 roving tabindex 需要子元素级焦点, 与自绘单元素冲突(见第八节 144) |
| Sidebar 没有移动端 Sheet 分支 | shadcn 的 `isMobile` 是浏览器媒体查询(窗口 <768px 时把侧栏换成 Sheet); 桌面窗口没有"移动端"这一层, 窄窗口用 offcanvas/icon 收起即可; 没有做断点自动切换 |
| Sidebar 的动作钮(菜单项右侧 …)只支持鼠标点击 | 原生 Button 会截走子元素的鼠标事件,BnClick 仍会发到整行 —— 动作钮只能做成按钮内部的命中区, 且键盘激活一律算整行(见第八节 158); 分组标签的 + 是普通 Element 的子元素, 鼠标/键盘都可用 |
| Sidebar 菜单项不能用 Tab 聚焦 | 与第八节 157 同一条 xcgui 限制: Tab 链只覆盖窗口内容区的直接子元素, 侧栏菜单嵌在 ScrollArea/布局容器里; 鼠标点击后可回车激活 |
| Sidebar 图标栏里内容区仍可滚动 | 上游 icon 折叠是 `overflow-hidden`(只能靠 tooltip 辨认); 我们的 ScrollArea 仍保留滚动/滚动条, 长菜单在 48px 栏里也能滚到 |

其余与上游的细小差异(逐项语义、变体映射)已在第四节的两张表里列出。

---

## 八、xcgui 踩坑全集(基于 1.4.3 源码逐条实测;编号稳定, 只增不改)

1. **渲染模式二选一**:`app.New(true)` = Direct2D,`false` = GDI+。`GDI_*` 系列函数仅 GDI+ 模式可用,圆角抗锯齿 D2D 更好。**运行时统一按 D2D 设计**,应用固定 D2D。
2. **2000 回调上限**:一律 `AddEvent*` 系列注册(复用回调,无上限风险)。
3. **背景管理器累积上限 20**:组件全部走 paint 事件,完全不碰 bkmanager。
4. **主题切换必须 Redraw**:主题管理器统一触发 `XWnd_Redraw`,组件自己不管。
5. **模态窗口销毁**:Dialog API 设计为"先取值再 EndModal"。
6. **DPI(实测 DPI=144 修正)**:元素创建坐标会被缩放渲染(24 逻辑 → 36 物理),但 `XEle_GetClientRect` 返回的是**未缩放的逻辑矩形**,且绘制上下文按逻辑坐标变换。因此自绘时**不要**按 `XWnd_GetDPI` 预缩放尺寸——二次缩放会让指示器撑满客户区被上下裁剪(圆变八边形)。这也是库中存在 `*DPI` 后缀函数的原因:普通版返回逻辑坐标。
7. **Edit 自绘**:原生背景需 `EnableBkTransparent` 处理,BeautifyEdit 已验证。
8. **字体**:主题携带字体方案(微软雅黑等中文环境默认字体),font 包管理。
9. **拷贝模式的代价**:组件 bug 不会随依赖自动修复 → 用 `xui update` + "组件文件写薄"缓解。
10. **元素绘制裁剪**:绘制被裁剪在元素自身客户区内,描边沿路径居中(贴边描边外半侧被裁),外扩式焦点环完全不可见 → 焦点框画在边框内侧(填充式,见 FocusFrame),需完整可见的描边请内缩。`XDraw_SetClipRect` 放宽裁剪区**无法突破**这层元素裁剪(实测:按下位移把底边推出 3px 后仍被裁掉,壳体高度少 3px)——要做"整体位移"这类越界绘制,应给元素预留像素(见 `button.go` 的 `pressShift`)。
    - **1px 细条元素(分隔线)有两个坑**(`separator.go`):
      ① 不要用画线(线宽沿路径居中, 一半被裁); 也**别以为关掉就完事** —— `widget.NewElement` 自带的原生 1px 边框(`EnableBkTransparent` 只清背景不清边框)在 1px 高的元素上就是那条线本身, 而框架对它按元素 y 的**相位取整**, 同一张卡片里的两条分隔线会一条 1 物理行、一条 2 物理行(实测截图: 229/229 两行 vs 229 一行), 看起来一粗一细。必须 `EnableDrawBorder(false)` 让自绘全权负责。
      ② 自己画也要**对齐整设备像素**: `Painter.Scale`(物理/逻辑比)填的是 `DPI/96`, 1 逻辑像素 = 1.5 物理像素, 框架对矩形按相位取整还是会出现 1/2 行漂移。取元素矩形中心所在的设备行, 用 `Draw.FillRectColorF(RECTF)` 只填那 1 行 —— 全 DPI 恒定 1 物理像素, 视觉略细于浏览器(浏览器 150% 下是 1.5 行)但**处处一致**。
    - `Painter.Scale` 曾一直是 0(`BeginPaint` 没赋值), 现已填 `DPI/96`; 需要设备像素对齐的绘制直接用它。
11. **实心按钮 hover 用透明度**:shadcn 的 `hover:bg-primary/80` 语义用 `WithAlpha` 实现,比 lighten/darken 在近黑/近白主题色上有可感知变化,且明暗主题行为一致。
12. **cy=0 的控件高度就是 0**:xcgui 不会按内容自适应,高度为 0 的控件直接不渲染(探针实证)→ 组件构造函数对 cy<=0 一律补默认高度。
13. **窗口背景必须用 Leave 状态注册**:xcgui 的 `XBkM_AddFill` 按"当前组合状态"选填充,窗口绘制时处于 `Window_State_Flag_Leave`,用 `Element_State_Flag_Nothing(0)` 注册的填充永远不被选中;且需先 `XBkM_Clear` 清默认填充(paint 事件 / BkManagerEx / SetBkInfo XML 均无法改变窗口背景,探针实证)。已封装为 `xui.BindWindowBackground`。
14. **标题文字默认白色**:xcgui 标题栏文字颜色为白色(为默认蓝色标题栏设计),主题化窗口(白/黑底)需 `w.SetTitleColor(theme.Foreground)` 并挂 OnThemeChange。
15. **窗口样式按标志位组合**:去掉系统图标和三键 = `Caption|Border|Center|Drag_Border|Allow_MaxWindow|Title`(不含 Icon/Btn_*);去掉关闭按钮后需自补 ESC 关闭(`XWnd_CloseWindow`)。
16. **focus-visible 需自行跟踪焦点来源**:xcgui 只有单一 Focus 状态,不区分鼠标/键盘来源 → 全局跟踪鼠标按下/Tab 键(运行时 focus.go),文本输入类控件例外(始终显示,同浏览器语义)。
17. **元素 paint 回调里能画位图、也能画 SVG;图标走专用管线**:
    - **位图能画**:`XDraw_Image`(1:1)/ `XDraw_ImageEx`(拉伸)在 paint 回调里输出正常。
    - **SVG 能画, 但不保证画对**:`svg.NewByString` + `DrawSvgEx / DrawSvg / DrawSvgSize` 出图正常(建议先 `SetSize/SetPosition`);但对 lucide 那种归一化 path 渲染走样(圆弧变形、`fill="none"` 子路径被糊成填充块), `svg.NewByString(原始 lucide path)` 更是直接返回 **nil**。
    - **所以图标不用 SVG**:`icons.go` 把 lucide path 解析成折线, 再用**直通 D2D 几何填充**出图(抗锯齿/几何精度见第 43 条), 折线描边保留为回落路径;也可以把图片交给原生控件(`Button.SetIcon` 等)。
18. **`XWnd_SetWindowPos` 吃物理像素**:框架内部坐标是逻辑像素(窗口创建时自动按 DPI 缩放),但窗口定位/尺寸透传给 Win32 → 必须自己乘 `dpi/96`,否则窗口会跑到 1/scale 的位置(实测 144 DPI 下 449 变成 299)。自绘提示窗(Tooltip)踩过这个坑。
19. **原生 ComboBox 必须先 `CreateAdapter`**:否则 `AddItemText` / `GetCount` 会让炫彩弹"XAdTable_GetCountRow 中,输入句柄可能无效[0x0]"的错误框,并阻塞后续调用。
20. **SliderBar 的滑块几何**:pos→滑块按钮是 `[0, W - buttonW]` 线性映射(实测 pos=0/25/50/100 → 0/47/94/188, 宽 200 时)。自绘时直接读 `XEle_GetRect(thumb)` 拿滑块矩形,与拖动行为严格一致;滑块外扩 3px 的焦点环要画在**滑动条**的 paint 里(滑块自身客户区只有 12x12,画在自己回调里会被裁)。
21. **原生滚动条要手动收敛**:弹层 ListBox 的滚动条默认是系统配色(与主题完全不搭)。做法已统一成 `ui/scrollbar.go` 的 `styleScrollBar` + `showScrollBarOnHover`(五个载体构造时 `SetScrollBarSize(10)`), 规格逐条对齐 shadcn scroll-area.tsx: 轨道 `w-2.5`(10) + `p-px`(滑块内缩 1) → 滑块 8 逻辑像素宽、`rounded-full` 圆角 4。四个必须知道的坑:
    - **圆角只能设在背景对象上, 且每个状态要各自的 id**: bkmanager 的 id 版 `AddFill(nState, color, id)` + `GetObjectObj(id).SetRectRoundAngle(4,4,4,4)`。三个状态**共用一个 id 时只有第一次生效** —— hover(Stay) 找不到背景对象会回退默认绘制, 表现为"悬停时滑块变宽且圆角消失"(实测)。换色用 `BkObj.SetFillColor` 就地改, 不必重建背景对象。
    - **配色分两套**(`containerScrollBarSkin` / `embedScrollBarSkin`): ScrollArea(容器型)= `bg-border`, 悬停不换色(radix 语义); 多行 Input / Table / Tree / ComboBox 弹层(嵌入式; 这几个在原版里就是浏览器原生滚动条)= 前景色淡化版, 悬停加深(对齐 Win11 Fluent 滚动条的观感)。颜色必须**展平成不透明**(`flatOn` 混到卡片色): 半透明的 Border(暗色是 10% 白)会与所在表面再合成一次 —— 同一个滑块在页面上是 #232323、卡片里是 #2f2f2f, 两个滚动条肉眼可见地不一致。
    - **显隐用元素透明度, 不用 `Show(false)`**: 原生滚动条的显隐翻转会改变视口宽度(列宽/内容宽跳变)。`showScrollBarOnHover` 初始 `SetSubtreeAlpha(0)`, 在**窗口** MouseMove 里用 `XWnd_GetStayEle` 判定(返回鼠标停留元素): 落在宿主子树内 → 淡入; 落在滚动条子树内 → 滑块换悬停色; 用 `xui.Presence` 做 150ms 淡入淡出。⚠️ **元素级 `MouseStay/MouseLeave` 在这里不可靠**(实测: 鼠标移出多行 Edit 收不到 MouseLeave, 原生 List 不把悬停路由给滚动条), 窗口级 + GetStayEle 对五个载体一致。⚠️ 淡出到位那一帧 anim 不再重绘, 要用 Presence 的 done 回调补一次 `XEle_Redraw`, 否则屏幕上留一层淡残影。
    - 主题切换/重入时先 `ClearBkInfo` 再重加(否则累积超过炫彩 20 个背景对象上限); 滚动条销毁时(弹层每次弹出都新建)从 `thumbSkinReg` 里清掉句柄。
22. **SliderBar 的滑块按钮在 hover 时会冒方块**:原生 Button 的四态会画边框/焦点矩形,`EnableBkTransparent(true)` + `EnableDrawBorder(false)` + `EnableDrawFocus(false)` 三件套只能压住一部分——Focus 态的方块三件套+`ClearBkInfo`+空 paint 回调都压不住, 最终结论见第 25 条(弃用原生按钮, 纯自绘)。
23. **原生绘制的文字不要自己再画一遍**:ComboBox 这类由原生绘制文本的控件,在 paint 里再画一次文字会出现重影(错位的一层) → 文本交给原生,只在 paint 里同步颜色(`SetTextColor` / `SetDefaultTextColor`),`*pbHandled = true` 只能拦住"绘制回调"本身,拦不住原生文本。
24. **异型透明窗口可以做到真透明,且背景对象支持圆角**:
    - 形状来源:`Window_Transparent_Shaped` + `SetTransparentAlpha(255)` 后,窗口形状 = **自身绘制内容中不透明像素的并集**,并不是只认背景对象。因此**仅靠 paint 自绘**就能拿到带真实 alpha 边缘的形状——气泡圆角外、箭头两侧完全透明,压在其它应用画面上也不露方块。
    - **前提**:不能给窗口再挂背景填充(如 `xui.BindWindowBackground`),那会把整窗铺成不透明矩形,透明随即失效。Tooltip 原来的"四周同色方块"就是这个原因,不是形状机制做不到。
    - 对照实测(探针,窗口矩形内与基线逐像素比较):仅 paint 自绘 + 异型 → 88% 像素变化,气泡下方一行 0 像素变化(真透明);整窗铺主题底色 → 99% 像素变化,气泡下方 453/480 像素变化(不透明)。同一份 paint 代码,只差是否挂窗口填充。
    - 想用背景对象做圆角也行:`bkm.AddFill(state, color, id)` 传入**非 0 的 id**,再 `bkm.GetObjectObj(id).SetRectRoundAngle(左上,左下,右上,右下)`(配 `SetMargin` 定位置尺寸)即可得到圆角矩形;该对象还支持 `SetRotate`、`SetBorderColor` 等更多美化。官方示例 `Advanced/WindowPartiallyTransparent`(窗口级 bk 管理器)与 `webview/WebViewPartiallyTransparent`(布局元素级 bk 管理器)都用了这一手法。
    - 生命周期无坑:`Window_Style_Nothing` 的窗口按"先建 10x10 → 首次显示前改尺寸/位置 → 复用再改尺寸"的懒创建流程走,异型透明一切正常(探针实证)。
    - ⚠️ 附带发现:`Window_Style_Nothing` / `Window_Style_Center` 的窗口**自身不绘制任何东西**(只画子元素),拿它当普通宿主窗口会"看不见窗口";要正常显示请用 `Window_Style_Default|Drag_Window`。
25. **别跟原生 SliderBar 的滑块按钮(原生 Button)纠缠, 要么全套压制要么完全弃用**:它的默认样式通过 **BkInfo** 给 Stay/Down/Focus 态注册了填充+边框——悬浮冒灰色方块、拖动后冒白色方块、点击聚焦后又冒一种方块, 形态还各不相同。`ClearBkInfo`(实测 bkinfo=0 仍漏)+ `EnableBkTransparent` + `EnableDrawBorder/DrawFocus(false)` + 空 paint 回调(`pbHandled=true`) 全部叠加也压不住 Focus 态;而 `EnableFocus(false)` 虽然能断掉焦点方块, 却会让按钮**不再上报 Stay 状态**(悬浮检测失效, hover 环画不出来)。结论:Slider 最终**弃用原生 SliderBar 载体**, 改用普通 `widget.NewElement` + 自实现交互(LButtonDown 里 `XEle_SetCapture(true)` 捕获鼠标, MouseMove 持续更新, LButtonUp 释放; MouseStay/MouseLeave 自跟踪 hover; KeyDown 左右 ±5%/Home/End), 绘制 100% 自绘后一切干净。**教训:当原生控件的残留绘制出现在多个互不相干的状态路径上时, 逐个压制是无底洞, 直接换纯自绘。**
26. **Slider 几何要对齐 radix**:滑块**中心**在轨道两端之间移动(滑块会向轨道端点外伸出半个身位), hover/active/focus-visible 的 ring-3 再向外扩 3px;而元素自绘被裁剪在自身客户区内(见第 10 条)。因此:元素左右各比**可见轨道**宽出 9px(滑块半径 6 + ring 3), 轨道画在内缩 9px 处, **入参 cx 仍是可见轨道宽度**;滑块中心由 pos 自算, 两端 ring 刚好落在元素边缘, 完整可见。
27. **`XDraw_FillRoundRect`/`XDraw_DrawRoundRect` 的 nWidth/nHeight 就是圆角半径**:不要把半径当直径再乘 2 —— 按钮/开关/下拉框会全体过圆成胶囊形; 按半径直传时视觉与 shadcn 一致。`xui.Painter.FillRound/StrokeRound` 直传半径即可, 不要自行换算。(Slider 上的"方块光环"是原生按钮 BkInfo 残留, 见第 25 条, 与 FillRound 无关。)
28. **不整体填充的透明元素必须在 onPaint 开头先铺宿主底色, 否则上一帧像素残留**(实测踩坑):Slider 的轨道/滑块/ring 都是局部形状, 元素背景透明, 状态变化触发重绘时框架**不会擦除旧像素**——hover 离开/点击聚焦后灰环一直留在画布上(`Redraw(true)` 立即重绘也无效, 因为重绘本身不擦底)。修法:onPaint 第一步 `p.FillRound(&p.Rc, t.Card, 0)` 铺满宿主底色(组件放在卡片上用 Card; 放在窗口背景上需对应调整)。Switch/Button 不留残影正是因为它们整体填充了自己的矩形。
29. **LayoutEle 布局语义与文档直觉不符, 全部经实测**(实测):LayoutEle 实为"横向 flex 流 + 自动换行"盒子——
    - **默认就是横向流**, 纵向堆叠必须 `EnableHorizon(false)`(列布局);不设间距时多个 Fixed 项会并排塞满一行再换行, 绝不是想象中的竖排;
    - 列布局的主轴(纵向)间距 = `SetSpace`, 交叉轴(水平)对齐 = `SetAlignH`;行布局的行内间距 = `SetSpace`, 换行后的行距 = `SetSpaceRow`(只设 SetSpace 纵向间隙恒为 0), 交叉轴(垂直)对齐 = `SetAlignV`;
    - **`Layout_Size_Fill` = 独占一整行/列**, 兄弟项被挤到下一行(甚至挤出布局而不换行), 不能与 Fixed 项同行;等分剩余空间用 `Layout_Size_Weight`(已验证多列均分);
    - 项级 `XWidget_LayoutItem_SetAlign` 实测无效, 交叉轴对齐只能用布局级 `SetAlignH/V`(对等高项无副作用, 混合高度行居中靠它);
    - `XWidget_LayoutItem_SetMargin` 各方向均生效;**布局项尺寸的默认值是 `Layout_Size_Disable`(0), 语义 = "布局不接管这一轴的尺寸, 元素保持自己当前的尺寸";一旦显式声明(Fixed/Fill/Weight), 就变成布局引擎反过来覆盖元素尺寸** —— 所以 `Fixed` 里传的值必须等于元素真正需要的尺寸, 它不是"把当前尺寸记下来"(探针实测, 同一个 `ui.NewButton(36, 28)`×4 放进同一行:
      | 写法 | 排版后元素 | 说明 |
      | --- | --- | --- |
      | 不设 | 36×29 | 保持创建尺寸(29 = 28 视觉 + button.go 的 pressShift 1px 预留) |
      | 宽高都 `Fixed(36, 28)` | **36×28** | 高度被压低 1px, 按 `p.Rc-1` 算壳体 → 视觉只剩 27 |
      | 只 `Fixed` 宽 | 36×29 | 高度那一轴仍是 Disable, 保持原样 |
      | 只 `Fixed` 高 | 36×28 | 同上, 只有被声明的那一轴被覆盖 |);
    - 由此定下**布局约定(见 `xui/layout.go`)**: 组件构造函数已经把设计稿尺寸建在元素上
      (Label 24 / Checkbox·Radio 24 / Badge 20 / Input 32 / Pagination 33 / Separator 1 / Progress 4 /
      Button 视觉高 + 1px 按压预留…), 所以**只声明你要改变的那条轴, 其余留给元素**:
      `xui.ItemFill(e)`(宽铺满)、`xui.ItemWidth(e, 240)`(锁设计常量宽)、`xui.ItemWeight(e, 1)`(按权重)、
      `xui.ItemHeight(e, 30)`(锁高)、`xui.ItemMinSize(e, 200, 0)`(缩放轴下限)、
      `xui.ItemMarginTop(e, 8)`(外边距, 四边各一个, 内部读改写所以互不清零);
      两个轴一起传用 `xui.ItemSize(e, w, h)`, 其中"不要覆盖"的那条轴传 **`xui.ItemKeep`(-1)** ——
      哨兵用 -1 而不是 0, 因为 0 是合法尺寸(用户可能真要 0 宽/高)。
      元素自身尺寸不对时(多行说明文字要 48 高)**改创建参数**, 不要在布局里补高度 —— 布局里的尺寸
      一旦写成常量, 就必须与设计稿一致、而不是与元素当前尺寸一致(旧约定"每项都显式 SetWidth/Height
      并在高度传 `GetHeight`"就是抄元素尺寸, 抄错即出 bug(例如标题栏按钮矮 1px)。实测数据见上表:
      不写 = 保持元素尺寸 36×29; 写错常量 = 被覆盖成 36×28(壳体少 1px);
    - 一行里**不声明宽度**的项会保持自身宽度, 但 LayoutEle 是"横向流 + 自动换行", 行内总宽超了会把
      后面的项顶到下一行(探针里 9 件超宽时 Input/分页被换行) —— 宽度该写还是得写;
    - 内部标签按创建时 cx 等分的自绘组件(如 Tabs)不能被布局拉伸, 用 Fixed 宽;Slider 这类元素比"可见宽度"外扩的组件, Fixed 宽 = 传入 cx + 2×外扩量。
    - **容器骨架收进运行时**(`container.go`): app 真正反复手抄的只有三种骨架, 固定成入口 ——
      卡片内容区 `card.ContentStack(space)`(卡片是布局容器, 内容区是它的 Fill 子项 + 内边距外边距, 随卡片缩放)、横向行 `xui.HStack(parent, ht, space)`
      (宽 Fill / 高 Fixed(ht) / 交叉轴居中)、等宽列 `xui.VStackWeight(parent, weight, space)`
      (宽 Weight / 高 Fill);**布局项尺寸与"纵向溢出换列"都在构造里处理好**(所以列容器一律
      `EnableAutoWrap(false)`), app 代码只往里塞控件。其它组合(如"铺满父级的纵向区")用
      `widget.NewLayoutEle` + `xui.Item*` 自己拼 —— 不要再造第四个骨架, 先把这一处的写法收进来。
30. **ComboBox 原生弹层的样式接管路线**(实测踩坑, combobox.go 依赖):弹层 = 独立弹出窗口 + ListBox(本质 ListView), 只能拿到句柄做"半接管"——
    - **窗口 BkM 对这个弹层无效**: `XWnd_GetBkManager` 拿到的管理器加填充/描边完全不绘制(绿色探针实证), 弹层白底另有来源;
    - **`ClearBkInfo` 后元素回退"系统默认内容"(白底方角)**: 想要透明必须显式垫一层 alpha=0 的填充(`AddFill(state, 0, 0)`), 算"有内容"才不会回退;
    - **圆角用 Win32 区域裁剪**: `wapi/wnd.SetWindowRound(hwnd, 物理像素半径)`(SetWindowRgn)稳;xcgui 的 `SetTransparentType(Shaped)` **不会**让圆角外变透明(弹层内容本身不透明, 转角仍是白底)—— 但它把窗口变成 layered, 这是**整窗 alpha 淡入**的前提(见本节 69 与下方进场淡入条), 两者各管一件事, 可以同时用(区域裁剪后转 Shaped, 圆角仍然生效);元素背景对象 `SetRectRoundAngle` 写入成功但圆角外仍是窗口白底, 和区域裁剪要配合用;
    - **进场淡入做不了(整窗 alpha 会概率性冻结)**: 弹层是**普通窗口**, 不换类型时 `SetTransparentAlpha` 完全无效(探针实测: 设 128 抓图仍全白); 在 `XE_COMBOBOX_POPUP_LIST` 回调里转 `Window_Transparent_Shaped` 后**单次**设 alpha 是生效的, 但**连续逐帧改会概率性停在早期某一帧**(实测: 40 轮快速"开→立刻选"里有 3 轮弹层窗口停在 layered alpha = 3/255, 而日志显示 apply 已经跑到 1.0) —— 弹层看不见却照样吞点击, 表现为"点触发器没反应"(详见本节 69)。**当前实现不做进场淡入**, 只保留转类型前的样式与定位;
    - **描边不能画在 ListBox 的 paint 事件——该事件在这个弹层窗口上不渲染输出**(探针实证: 内缩 10px 的 6px 红色粗环都画不出来); 这个窗口上能出图的只有 BkM 背景对象、条目 `DrawItem` 回调和 `XE_PAINT_END`;
    - **BkM 的 border 对象贴边画, 会被 SetWindowRgn 的窗口区域裁得只剩左边一条**(上/右/底全消失, 只留左缘; 区域与内容为何错位未深究)。ring-1 描边的出路: 先 `XEle_EnableEvent_XE_PAINT_END` 再 `AddEvent_Paint_End`(该元素及子元素绘制完成事件, 在条目之后执行 = 边框盖内容边缘, 同 shadcn), 描边整体**内缩 1px** 落在窗口区域内, 四边完整, 圆角弹层上 1px 白边距不可感知;
    - **条目高亮/勾用 `AddEvent_ListBox_DrawItem`**(回调带 Index/NState/RcItem): 原生在弹层打开时**不给选中项置 Select 态**, 要自己拿 `XComboBox_GetSelItem` 比对求"选中项"(只用来决定右侧 check), 但**底色只画给"指针所在项"**(`NState == Stay || Select`)——shadcn 的 item 只有 `focus:bg-accent`, 选中态由右侧 check 表达, 不带常驻底色; **指针不在任何条目上时(弹层留白、弹层上下左右边缘外)没有任何底色**, 对齐 radix: 条目的 pointerleave 把焦点交回 content(`onItemLeave = content.focus`), 高亮随之消失。两个错误版本(勿用): ① "选中项永远高亮"(鼠标移到别项后选中项仍带灰底); ② "指针离开条目后高亮留在最后悬停项"(radix 的 `highlightedItem` 在 item 失焦后并不保留, 用 `XListBox_GetItemMouseStay` 闩住最后悬停索引会画出一个不该有的灰底)。灰底的像素表现: 指针在项上时该行背景均值 ≈ 241-246, 一离开条目(移到弹层外)同一批行(弹层 y 436..458)均值回到 249-255(= popover 白底 + 文字旁 JPEG 振铃), 即灰底消失;文字是项模板的 shapeText 子元素, 在项背景之后绘制不受影响;
    - **`SetDropHeight` 必须在弹层打开前设置才生效**(构造时), 弹出事件里设置无效(实测 set/get 均为新值但窗口不变);固定模式(`EnableDropHeightFixed(true)`)才使用该值;
    - **触发区滚轮**: 原生会滚动 ComboBox 内部文本(文字上移), `EnableEvent_XE_MOUSEWHEEL(false)` 屏蔽(事件交给父元素), 弹层 ListBox 独立不受影响;
    - **弹层退出后窗口句柄即失效**: 关闭后用旧句柄调任何 XWnd_* 会弹"句柄无效"错误框(实测 `XWnd_GetDPI`), 必须挂 `AddEvent_Combobox_Exit_List` 置空缓存句柄;
    - **弹层窗口/ListBox 每次弹出都是新建的**(退出即销毁, Exit_List 后窗口句柄即失效), 事件在每次 Popup_List 里重新注册即可, 不会重复;全局的 OnThemeChange 回调才需要 once 守卫。
    - **滚动条残影**: 原生的滚动条显隐判定在打开时用旧条目高算好, 调矮条目后不会自己收起; 用滚动视级开关 `ShowSBarV/ShowSBarH` 显式控制(条目级 `ScrollBar.Show` 会被原生布局覆盖)。弹层高度按 `comboDropHeight` 取**内容高**(见本节 52)。
    - **弹层默认定位与 item-aligned 重定位**(探针实测): 原生把弹层放在触发区**正下方**(顶=触发区底、左=触发区左、宽=触发区宽)。radix Select 的默认定位是 item-aligned(shadcn select.tsx 的 SelectContent), 即**弹层盖住触发元素、选中项正对触发位置**——用 `wapi.SetWindowPos`(物理像素, `SWP_NOSIZE|NOZORDER|NOACTIVATE`)在 Popup_List 事件里重定位: 弹层顶 = 触发区顶 − (选中项在内容里的顶 − (触发区高−条目高)/2), 其中选中项在内容里的顶 = p-1 边距 4 + ListBox 固有 3px 内缩 + 选中索引×(条目高+1)= 7+29i。**两种盒子高度不同(触发区 32、条目 28), 必须按"文字中心"对齐**(选中项顶 = 触发区顶 + 2); 若用"p-1 + 选中索引×条目高"且按底边对齐, 会少算 5px, 选中项文字比触发区文字低 3 物理像素(点开瞬间字"往下跳")。 **横向也必须按"文字左缘"对齐**(radix 源码 `SelectItemAlignedPosition`: `left = 触发区文字 left − (条目文字 left − 内容 left)`, 对齐的是文字而不是盒子): 弹层左缘 = 触发区左缘, 而条目文字相对弹层左缘的内缩 = p-1(4)+ListBox 固有 2+条目文字左边距, 该边距按 shadcn 的 px-2=8 时实测条目文字比触发区文字**右移 2px**(触发区文字 813..862 / 条目文字 815..862), 点开瞬间看着像"文字往内收缩"; 边距改 5 后两者墨迹起点完全重合(同为 x 813, 条目 813..859)。裸宽度仍差 ~3px(50 vs 47)——触发区文字走原生控件(Direct2D)、条目文字走 shapeText 两条不同渲染路径, 同为 4 个全角字时字形推进取整不同, 非位置问题, 视觉不可感知。注意别用"整体左移弹层窗口"来对齐: 弹层宽度 = 触发区宽度, 窗口左移会让右侧 check 与弹层右缘的距离一起变; 贴屏幕边时向下压并用 `ScrollView.ScrollPosYV` 滚动让选中项仍可见。坐标链路: `XEle_GetWndClientRect`(逻辑, **原点=所属窗口 rect 原点, 含阴影边距**——实测含阴影窗口也成立) + 所属窗口 `XWidget_GetHWINDOW` → `wapi.GetWindowRect`(物理) → 逻辑差 ×DPI/96;
31. **xcgui 字体的 size 单位是 pt(磅), 不是像素**(重大全局坑, 实测确认): `font.NewEX(name, size, style)` 文档写明"size: 字体大小, 单位(pt, 磅)", 渲染尺寸 = size×DPI/72 物理像素。若把主题的 14(逻辑 px) 直接当 pt 传入, 144 DPI 下会渲染出 28 物理像素 em = **18.7 逻辑像素**, 全组件文字放大了 1.33×(墨迹实测 26-27 物理像素, 与 14pt 完全吻合)。修法: **pt = px×72/96(= ×0.75, 与 DPI 无关)**, 在 FontHandle 里统一换算(14px→11pt=14.67px, pt 取整导致 14 这类 ×0.75=10.5 的值有 ±0.67px 误差; 12/13/16px 恰好或接近取整)。排查入口: 同一截图里对比组件文字与已知 14px 参考的墨迹高度。
32. 用 pnpm 添加 shadcn 组件会报错, 应该用npx。
33. **Dialog 用"窗口内覆盖元素"实现**(`ui/dialog.go`):
    - **窗口子元素坐标原点 = 窗口左上角(含标题栏), 不是客户区原点**:`XWnd_GetClientRect` 返回的 `{8,8,624,412}` 只是窗口边框内缩(与标题栏无关), 用它给元素定位会整体偏移;遮罩尺寸直接取 `XWnd_GetRect` 的宽高放 (0,0) 即可铺满整窗(实测遮罩把标题栏也压暗, 窗口自绘的标题文字仍在上层保持清晰);
    - **自绘 Element 覆盖全窗天然是模态**:它挡住下层所有鼠标事件(实测覆盖后下层按钮既收不到点击也收不到 hover), 配合 `XEle_SetZOrderEx(h, 0, xcc.Zorder_Top)` 保证在最顶层(每次 Show 都抬一次, 因为元素创建顺序可能比对话框晚);
    - **半透明合成精确实测**:遮罩填 `RGBA(0,0,0,26)`(bg-black/10)后, 白底 255→229、深色钮 14→59(悬浮态), 与 0.9 系数完全吻合, 无累积变暗;
    - **隐藏遮罩必须整窗重绘**:`XWnd_Redraw(hWnd, true)`;局部重绘会留下变暗残影(半透明像素已经"烙"进帧缓冲);
    - **元素收不到键盘事件, 键盘统一走窗口 `AddEvent_KeyDown`**:探针里遮罩元素注册的 KeyDown 从不触发;窗口级回调在"编辑框聚焦"时也照常收到 ESC(27)/Tab(9) → 对话框的 ESC 关闭与 Tab 循环都在窗口回调里做, 用 `wapi.GetAsyncKeyState(VK_SHIFT)<0` 判 Shift+Tab;
    - **打开时先 Show 再 layout**:内容高度量的是"可见子元素"的占位(`XWidget_IsShow` 过滤), 隐藏状态下量出来是 0(否则面板只有标题+页脚);控件自建后默认可见, 所以 NewDialog 末尾要用递归 `XWidget_Show(false)` 隐藏整棵子树(父元素隐藏不会连带隐藏子元素);
    - **`XWnd_SetFocusEle` 搬焦点后不会重绘**:按钮上的焦点框不出现(输入框因为要重画光标/IME 才碰巧刷新), 必须手动 `XEle_Redraw(旧)` + `XEle_Redraw(新)`;且顺序要在搬焦点**之前**调 `xui.NoteKeyboard`——AttachWindow 的 Tab 钩子注册得晚(回调后执行), 等它标记"键盘焦点"时元素早已重绘完;
    - **焦点反查要沿父链**:`XWnd_GetFocusEle` 拿到的是复合控件(Edit)的内部元素, 与创建时的句柄不同 → 用 `XWidget_GetParentEle` 往上找焦点序列里的成员, 找不到再用自维护的下标兜底;
    - **Tab 序列只收交互控件**:`XObj_GetType` 白名单(Button/Edit/ComboBox/SliderBar/Editor/RichEdit/ListBox/TabBar)+ `IsEnableFocus && IsSwitchFocus`;Label 载体是原生 Button, 必须显式 `EnableSwitchFocus(false)` 才不会挤进 Tab 序列(顺带 `EnableDrawBorder(false)`, 否则它能画出原生边框);
    - **backdrop-blur-xs 可以精确实现**(见第八节 47 的 backdrop-blur 小节): 走"抓窗快照 + BkM 图片填充"。
    - **ring-1 要画在面板矩形外侧**:tailwind 的 ring 是 box-shadow, 画在盒子**外面**压着遮罩灰底, 才形成可见的深色描边;xcgui 圆角矩形描边落在路径**内侧**, 直接描面板矩形会把 10% 黑压在面板白底边上(230), 与遮罩灰(229)几乎同值而完全不可见(像素实测)。修法 = 描边矩形外扩 1px(dialog.go onPaint)。

34. **Dialog/文字/量测的一批实测结论**(视觉对齐期间, 探针+像素测量):
    - **自绘文字的字号就是逻辑像素, 不要按 DPI 反除**: 反除字号会让全局文字明显偏小; "文字顶到边缘"的真正原因是量测口径;
    - **`app.GetTextShowRect` 在 144 DPI 下返回值约为实际绘制宽度的一半**: 对话框页脚用 "保存"(2 字)量出 14.7 逻辑宽, 实际绘制 29.3 → 按它算按钮宽会让文字顶到按钮边。现在量宽处 ×2 校正(改量测 API 或换 API 亦可, 但要重新验证);
    - **量宽与绘制必须同字体同限宽**: 例如用 14px 主题字体量宽、却按 12px(`text-xs`)绘制, 气泡两侧留白会偏宽 17%;
    - **句柄有两级, 混用会弹模态错误框**: `XWidget_GetHWND(元素)` → Win32 HWND; `XWnd_*` 系列要的是 xcgui 窗口句柄 hWindow(用 `XC_hWindowFromHWnd` 转)。窗口句柄喂 `XWidget_GetHWND`、HWND 直接喂 `XWnd_GetDPI` 都会弹"输入句柄可能无效"错误框, 且**绘制回调里调这类校验型 API 风险高**(实测复现两次);
    - **容器元素被点击会聚焦并画原生框**: 普通 Element/内容容器点一下就获得焦点, 原生焦点框是**蓝色**(#58B1FC), 而它的**边框**在聚焦态是深色 —— 两个都要灭: `EnableFocus(false)` + `EnableDrawBorder(false)`。否则蓝框会压住输入框左右描边, 造成"输入框边框缺失/圆角怪"的错觉;
    - **元素内文字被元素矩形裁剪**: 按 CSS `leading-none` 把行盒压到字号高度时, 中文行高(≈1.32em)放不下, 字被切头 —— 文本元素要上下各外扩几个像素(布局高度仍按行盒算), 见 dialog.go `dlgTextOver`;
    - **页脚 `-mb-4` 的高度账**: 页脚贴面板底边时它自身高度要**全量**计入面板高(少算 16 逻辑像素会让面板整体矮一截);
    - **中文下 `font-medium`(500) ≈ 常规字重**: 微软雅黑只有 Regular/Bold 两档, 浏览器 500 会回落到 Regular。用粗体字形实现 500 会明显偏粗。标签看着比描述粗, 主因是**颜色**(`--foreground` 近黑 vs `--muted-foreground` 灰), 实测两者笔画宽度相同;
    - **Tooltip 定位不能叠加客户区偏移**: `XEle_GetWndClientRect` 返回的坐标原点已是"窗口 rect 原点(含阴影边距)", 再叠加 `XWnd_GetClientRect` 的内缩会让气泡整体下移/右移一个阴影宽度(实测 8 逻辑像素, 表现为箭头尖端扎进目标元素)。

35. **暗色下的"半透明令牌 + /NN 修饰"必须做 alpha 乘法**(暗色态, tabs/switch/radio 三处同源):
    - 暗色主题里 `input` 等令牌**自带 alpha**(`oklch(1 0 0 / 15%)`, 内置暗色主题写作 `RGBA(255,255,255,38)`); 此时 CSS 的 `bg-input/30` 语义是 15% × 30% = **4.5%**, 而 `disabled:opacity-50` 是 15% × 50% = 7.5%;
    - 用 `WithAlpha(v)` 直接**覆盖**这层 alpha 会: tabs 选中态比参考亮一倍(#686868 vs #2f2f2f)、未选开关的轨道变成不透明白饼、禁用开关反而比正常态更亮(50% > 15%);
    - 修法: 新增 `Color.ScaleAlpha(frac)`(在现有 alpha 上乘), 全库凡是"半透明令牌再叠透明度"的地方都用它, 不要用 `WithAlpha`;
    - 相关规则(逐条对源码): tabs 暗色选中 = `bg-input/30` + `border-input`(**不是** `bg-background`, 否则暗色下比列表更黑); radio 未选 = `border-input` + 暗色额外 `bg-input/30` 淡填充, 禁用 = 指示器整体半透明(标签文字不跟随) —— 禁用若画成 `Background@50%`, 暗色下就是一块黑盘; switch 未选轨道 `bg-input`/暗色 `bg-input/80`, 选中滑块 `primary-foreground`、未选 `foreground`。

36. **原生菜单皮肤的一批实测结论**(`ui/menu.go`; 圆角+阴影见第八节 64):
    - 弹窗圆角+阴影: **走 `Window_Transparent_Shadow` 路线**(官方 DrawMenu 示例同款, 详见 64) —— `Menu_PopupWnd` 事件里 `SetTransparentType(Shadow)` + `SetShadowInfo(nSize, depth, RadiusLg, false, 黑/128)` + `SetRect` 四周外扩 nSize; 圆角由阴影窗按逐像素 alpha 画出, 不再用 `wnd.SetWindowRound` 区域裁剪; shaped 透明对菜单弹窗无效, **Shadow 有效**(整窗 alpha 也可用);
    - 项几何(`SetBorderSize(4,4,4,4)` 后探针实测): `Menu_DrawItem` 的 `RcItem` 四周各留 **4px**(= shadcn p-1; 默认边距是 3, 差的那 1px 就是"高亮距弹层左右边缘与参考不一致"的根因), 高度 = `SetItemHeight`(28); **分隔栏槽高固定 4px**, 1px 线画在 `Top+2`、左右越过边距铺满弹层(shadcn `-mx-1`);
    - `"	"` 快捷键: 原生把 `	` 后的文本算进自适应宽度并预留 `NShortcutKeyWidth`(实测 "Ctrl+C"=37) → 把 "名字	快捷键" 整串传给 AddItem, 自绘时拆开分别渲染就得到右对齐快捷键, 不用自己量宽;
    - `SetLeftWidth(0)` 去掉左侧图标条; Label 项借 `Menu_Item_Flag_Disable` 实现"不可高亮不可点"(自绘时对 kind==label 单独配色);
    - **`XMenu_SetItemWidth` 逐项设置条目文本区宽度, 有效**(官方 DrawMenu 示例同款用法: AddItem 后对单个 nID 调用, 返回 true 表示受理; 探针实测: 3 条短条目弹窗宽 104, 全部条目 `SetItemWidth(300)` → 348, 只设其中一条 → 同样 348, 全部设 60 → 108): 弹窗宽 = **max(各条目有效宽) + 固定开销**(未设条目按自然文本宽计入; 开销 = 边框 + 文本间隔等, 条目高 28 / BorderSize(4) / LeftWidth(0) 配置下实测 48); **设定值完全覆盖自然宽** —— 自然文本宽 229 的长条目设 60 后弹窗仍 108, 长条目不会撑宽弹层。shadcn 的"弹层宽 = 触发区宽 + min-w-32 下限"(dropdown-menu.tsx `w-(--radix-dropdown-menu-trigger-width) min-w-32`)由此实现: 逐条目设 `max(触发区宽, 128) − 开销`(menu.go 已接入) —— 长文本由自绘 `XDraw_SetClipRect` 硬裁剪, 与 radix 的 overflow-x-hidden 一致(原生条目文本同样裁在弹层边缘, 截图验证); 显式覆盖 = `Menu.Width(w)`(对应 shadcn 在 Content 上加 `className="w-56"` 这类宽度类, demo 参考卡就是 w-56=224, 实测精确命中); 右键菜单(PopupAtCursor)无触发区, 退化为"内容自适应 + 128 下限"; 子菜单条目不设宽(shadcn SubContent 是内容自适应 + min-w-96, 不跟触发区);
    - 事件拓扑: 窗口级 `XWM_MENU_DRAW_BACKGROUND/DRAWITEM` 对该窗口弹出的**所有**菜单生效(每次新建的菜单对象、各自的子菜单窗口都触发) → 皮肤每窗口只注册一次(map 守卫), 绘制实时取令牌, 换肤零成本; `Menu_PopupWnd` 对每个弹出窗口(含子菜单)各触发一次且顶级先于子菜单 → 阴影处理放这里全命中;
    - 弹窗定位: `Popup` 收**物理像素**; 元素锚点 = `XEle_GetWndClientRect`(逻辑, 原点=窗口矩形含阴影边距) + `wapi.GetWindowRect`(物理) 换算; 原生给的初始落位在**副屏上不可信**(钳制基准错, 见 132) → 顶级弹窗的最终位置由 Popup_Wnd 回调按锚点重算, 子菜单仍由原生锚定父窗实际矩形;
    - ⚠️ **截图坑: 全屏缩略截图里暗色弹层窗口会显示成亮底**(截图管线对弹层窗口的合成/缩放问题), 区域放大看才是真实效果 —— 弹层类组件的暗色判断一律以 zoom/像素测量为准, 不要凭全屏缩略图下结论。
    - ⚠️ **菜单落位只能走 PopupWnd 回调里的 `SetRect`, 不能 `SetWindowPos`、也不能等 `Popup` 返回**: `XMenu_Popup` 是**阻塞**的(菜单关闭后才返回), 返回后再校正已拿到销毁的句柄(`XWnd_GetHWND` 报「输入句柄可能无效」); `SetWindowPos` 位移会直接崩进程(见 64)。历史结论"PopupWnd 里只能做对称外扩、不能动位置"已被探针修正: `SetRect` 做**任意落位**(单位逻辑像素)是安全的 —— 子菜单锚定父窗实际矩形会跟着走、命中判定跟随、大位移(上千像素)不崩, 副屏错位正是靠它修的(见 132)。

64. **菜单弹窗转阴影窗 + 进场动画的一批实测结论**(`ui/menu.go`):
    - **阴影弹窗 = 官方 DrawMenu 示例同款三步**: `SetTransparentType(Window_Transparent_Shadow)` → `SetShadowInfo(nSize, nDepth, nAngeleSize=RadiusLg, false, 黑/128)` → `SetRect` 窗口矩形四周外扩 nSize(逻辑像素, 客户区会收缩出同尺寸阴影带, 必须外扩补偿, Body 才回到原生给定的菜单位置)。nAngeleSize 一次把圆角(逐像素 alpha, 无区域裁剪锯齿)和投影都做了。**参数对齐 shadcn shadow-md**(0 4px 6px -1px black/10%): nSize=5(官方示例的 8 比shadow-md 大一圈)、nDepth=50(阴影引擎边缘浓度≈depth/1000, 50 时白底上边缘约 5%, 与 shadow-md 侧缘观感一致; DLL 阴影是均匀环带, 做不出 shadow-md 上 2px/下 10px 的不对称);
    - **子菜单锚点不受外扩影响**(探针实测: 子菜单仍贴父条目右侧、行对齐); 子菜单窗口走同样的 PopupWnd 事件, 阴影自动全命中;
    - **整窗 alpha 淡入技术上可用, 但菜单不用它做进场**(见 99): 阴影窗是透明类型, `XWnd_SetTransparentAlpha` 有效 —— 第八节 47 条 5 "native 弹层做不了淡入"的结论只适用于**非透明类型**弹窗(menu/combobox 的下拉窗默认不是 WS_EX_LAYERED, `GetLayeredWindowAttributes` 直接失败), **转 Shadow 后可淡入**; 依然遵守"alpha 永远不能精确 = 0"(47), 需要时进场前压到 `xui.AlphaFloor`。⚠️ 但菜单弹层用它会让 popover 底色一起半透明、底下的页面文字在过渡中透上来; 改成"面板不透明 + 只淡内容"又会先出一帧空白面板 —— 两条路都闪, 菜单最终**不做进场动画**(见 99);
    - ⚠️ **过渡闭包绝对不能捕获事件结构体指针**: `Menu_PopupWnd_` 是原生栈对象, 回调返回后被复用/清空 —— Presence 的 apply 帧里再解引用, 读到 0(淡入永远不动, 弹窗 alpha 卡在下限"隐形")或垃圾值(拿无效句柄调 X API → **整个进程静默崩溃**)。必须先把 `hw := pInfo.HWindow` **拷贝值**再进闭包;
    - ⚠️ **apply 每帧要 `XC_IsHXCGUI(hw, XC_WINDOW)` 守卫**: 菜单可能在过渡途中被关闭(弹窗随之销毁), 官方 DrawMenu 示例在 KillFocus 里同款用法;
    - ⚠️ **对菜单弹窗做 `SetWindowPos` 位移会直接崩进程**(`SWP_NOSIZE|NOZORDER|NOACTIVATE` 也不行, 进场那一瞬移 8px 就崩; `SetRect` 外扩或一次性落位校正没事, 见 132) → radix 的 slide-in-from-top-2 与 zoom-in-95 **都不可实现**, 进场动画整体放弃(见 99);
    - 像素表现(144 DPI): 条目高亮距弹层左右边缘各 6 物理像素 = 4 逻辑(= p-1, 左右对称); 分隔栏全宽;体色暗色 (23,23,23); 阴影带梯度柔和无黑边;
    - **菜单弹窗的绘制坐标系**(色块标定实验实测): `Menu_DrawBackground/DrawItem` 的绘制上下文**原点 = 弹窗窗口左上角(逻辑像素)**; `XWnd_GetClientRect` 返回的 {8,8,168,276} 就是阴影窗 Body(可见白区)在窗口内的矩形(= 窗口矩形 ± 阴影带) —— fill/ring 都画在它上面即可; **画进 Body 之外(阴影带里)的任何像素都会被阴影 alpha 蒙版压暗至不可见**(opaque 色块画进带里直接"消失"); ring 画在 Body 矩形边界上(线宽外半被蒙版裁掉), 肉眼就是一条贴着窗口边缘的线, 圆角用 RadiusLg 与 Body 一致, 四角才不脱开 —— ring 若内缩或半径不匹配, 视觉上就是"矩形线条浮在弹层里不贴边"。
    - **子菜单悬停延迟 ~330ms 是 xcgui DLL 固化的**(纯净菜单探针实测同值 324ms): 与系统 MenuShowDelay 无关(SPI 置 0 复测仍 343ms), 无配置 API, VK_RIGHT / SendInput 真实点击 / PostMessage 鼠标消息都无法提前触发。xui 侧已把自己那 100ms 进场动画整个去掉(顶级菜单与子菜单都即时全内容, 见 99)。要达到 shadcn 的即时展开只能改 DLL 或等作者提供 API。

37. **页面容器 Show(false)→Show(true) 后内容空白**:
    - 现象: 隐藏期间创建好的布局子树(LayoutEle 流式布局)在页面恢复显示后只有壳在渲染, 内部控件全空白;
    - 根因: xcgui 布局元素在父容器不可见期间不排版(与第八节 29 "布局项变更需手动 AdjustLayout" 同源), Show(true) 只改显隐属性, 不会触发子树重排;
    - 修法: 切页时对**新显示的页面容器**调 `AdjustLayout`, 再 `Redraw`; 初始就可见的页面不需要。

38. **自绘标题栏的无框窗口组合与鼠标穿透**:
    - 不要 Default(自带图标/标题/三键): 样式用 `Caption|Border|Drag_Window`(不含 Icon/Title/Btn_*/Allow_MaxWindow/Drag_Border), `SetBorderSize(0,30,0,0)` 保留 30px 标题栏高度, 阴影照旧 `Window_Transparent_Shadow + SetShadowInfo`; BkM 窗口背景绑定在此组合下正常工作(Window_Style_Nothing 才是"自身不绘制任何东西"的那个);
    - 图标/标题等**纯展示元素必须 `EnableMouseThrough(true)`**, 否则元素挡住 Drag_Window/Caption 的拖动命中(实测标题文字区域拖不动窗口, 穿透后整条标题栏可拖); 子元素不受穿透影响;
    - 控制按钮用普通自绘按钮(幽灵变体 + 图标)即可, 最小化 `w.ShowWindow(xcc.SW_MINIMIZE)`(窗口包没有 Minimize 包装), 关闭 `w.CloseWindow`; 不需要 CaptionButton 示例那套"读原生最小化按钮坐标 + Paint_End 里重定位"的逻辑(那是给 Default 样式加按钮用的)。

39. **窗口布局(EnableLayout) + 阴影窗口 + 自绘标题栏的一批实测结论**:
    - **客户区已避开阴影带**: rect 800x590、阴影 8px 时 `XWnd_GetClientRect` = {8,8,792,582}(784x574), 子元素坐标就是可见区坐标, **不需要再设 padding 避让**; 但窗口 rect 比可见区大一圈, 按 rect 宽算坐标会超出可见区(实测按钮右缘超 12px);
    - **布局区 = 客户区 − border size − padding**: border 归零后布局区=整个客户区;
    - **布局项必须是 LayoutEle**: 普通 Element 设置 `LayoutItem_*` 也不被排版(探针实证 red 块不占位、尺寸保持 0 → 连带裁掉全部子元素, 表现为整块空白);
    - **主轴剩余空间用 Weight 不是 Fill**: 纵向布局里 Fill 高度 = 独占整列(与前面 Fixed 项重叠, 实测 content 574=全高盖住 cap 30); 横向行里 Fill 宽度会把后面的 Fixed 兄弟挤到下一行(按钮消失)。DESIGN 第八节 29 的"Fill 独占一行"就是指主轴; 交叉轴用 Fill 铺满没问题;
    - **绝对定位容器要 `EnableLayout(false)`**: LayoutEle 默认排版开着, 内容区里绝对定位的 tabs/pages 会被当布局项排走(实测 tabs 跑到窗口底部);
    - **双击最大化自己补**: border 归零后原生 caption 区域没了, 窗口级 `AddEvent_LButtonDBClick` 里 `IsMaxWindow ? ShowWindow(SW_RESTORE) : MaxWindow`(MaxWindow 只管最大化, 不会切换); 最大化/还原时窗口布局自动跟随, 按钮组(Weight 顶右)始终贴右上角。**双击命中带的 y 阈值 = `captionH + shadowSize`(最大化时不计阴影)**: 子元素坐标原点是含阴影带的窗口左上角, 阴影大小抽成常量 `shadowSize = 12`(别把 12/30 裸数字散在阈值里);
    - 纯容器(cap 行/内容区/页面)一律 `EnableMouseThrough(true)` 不挡 Drag_Window 拖动, 子元素(按钮)不受影响。

40. **原生 List 皮肤接管路线**(ui/table.go): 载体是原生 `List`(表头/纵向滚动/行选中/项模板复用全部原生), 外观全自绘。逐条实测:
    - `List_Item_.RcItem` 的文档注释写着"未使用", **实际有效**(就是该单元格的逻辑矩形); 表头项 `List_Header_Item_.RcItem` 同样有效(表头项的 `HLayout` 恒为 0, 只能用 `RcItem`);
    - `XE_LIST_DRAWITEM`(AddEvent_List_DrawItem)按**单元格**触发(不是按行), 同一行各列的 `NState` 相同, 且各单元格矩形**首尾相接** → 逐格填充拼出整行底色、逐格画线拼出整行底边, 不需要自己算行矩形;
    - `RcItem` 是列表客户区的**逻辑坐标**(与 `XEle_GetRect(HLayout)` 一致), 行 0 的上边 = 表头高 + 2;
    - 原生 List 的内容区在**左上各内缩 2 逻辑像素**, 且**表头单元格从 x=0 起、数据单元格从 x=2 起**(表头整体比数据行靠左 2px); 行间距默认 **1px**。因此: 列宽按 `cx - 2*2 - (滚动条 14)` 分配, `SetRowSpace(0)` 去掉行间距, 表头形状文本缩进比单元格少 2 才能左对齐(实测列内文字与表头文字 x 完全一致);
    - `SetDrawRowBkFlags(Nothing)` 只关掉原生行背景/分割线**绘制**, **不影响**行悬浮 `List_Item_State_Stay` 的状态跟踪与重绘触发(实测鼠标移入行即触发重绘并上报 state=1);
    - 表体透明用 `EnableBkTransparent(true)`: 元素未绘制区域**真的透出宿主底色**(与 ComboBox 弹层的"白底回退"不同); 表头是独立子元素, 要另外 `GetHeaderHELE` → `EnableBkTransparent` + `EnableDrawBorder/Focus(false)`, 否则表头保留默认灰底;
    - 行底色/行边框/表头底**全部在绘制回调里实时取令牌** → 换肤零成本(不需要 OnThemeChange); 只有"必须下发给原生对象"的东西才要重刷 —— 形状文本(单元格/表头)的字色字体、滚动条的背景对象 → 用一个 `applyTheme` 统一处理, 并挂 `xui.OnThemeChange`;
    - 滚动条只能走背景管理器(同第 21 条的 `styleScrollBar`): 轨道用表面色, 滑块三态同色 `bg-Border`(8px 圆头); 换肤时先 `ClearBkInfo` 再重加不会累积;
    - 纵向滚动条会占掉内容宽度: 自适应列(占满剩余宽度)要让出 14px(`scrollBarW` 10 + 内容左右固有内缩 2×2), 否则最后一列被滚动条盖住; 且滚动条显隐切换时要重算列宽 → 按"是否需要滚动"翻转时 `SetColumnWidth` 重排;
    - 列宽: 原生没有权重/百分比列, 自适应列要自己算 `元素宽度 − 2×内缩 − (需要滚动条?14:0) − 固定列宽之和`;
    - 滚动条尺寸 `SetScrollBarSize(10)` + `SetSliderPadding(1)` → 滑块 8 逻辑像素宽、圆角 4(实测 @150% DPI 下物理 12px), 与 shadcn 的滚动条滑块尺寸/圆角一致;
    - **竖排表格不能用原生 `widget.Table`**: 那是 Shape 系的"单元格画布"(合并单元格、逐格画线、无滚动/无虚拟行), 与 shadcn 的可滚动行列表语义不符 → 本组件的 `Table` 一律以原生 `List` 为载体。

41. **LayoutEle 纵向(列)布局也会"溢出换列"**: 列布局里主轴上剩余高度放不下下一个项时, 该项**不会被裁剪也不会重叠, 而是被甩到"下一列"** —— `x += 容器宽度, y = 0`, 表现为"控件整块消失"(与第八节 29 的"横向 Fill 独占整行把兄弟挤到下一行"是同一套换行机制的两个方向)。修法: 布局级 `EnableAutoWrap(false)`; 同时要保证容器高度够(本例 card 高 320 才能容下 24+14+24+14+180=256 的列)。`AdjustLayout` 之前读到的 `XEle_GetRect` 是旧值(排版前), 排查这类问题要在排版之后取坐标。列容器现在都走 `xui.VStackWeight` / `card.ContentStack`(`EnableAutoWrap(false)` 已内置), 手搓 `NewLayoutEle` 时别忘了这一条。

42. **原生 Tree 自绘皮肤(Tree 组件探针实测, 载体路线同 40)**:
    - **自定义项模板要在 `CreateAdapter` 之前 `SetItemTemplateXMLFromString`**, `SetItemHeightDefault(n, n)` 也要在模板/适配器/数据**之前**调用——之后设置不生效(模板 `layout.height="fill"` 会吃掉模板里的 heightT, 行高实际由 SetItemHeightDefault 决定, 探针默认 20);
    - **展开按钮(itemID=1, expandButton=true)的 "+/-" 原生图形**: `ClearBkInfo` 压不掉——背景清空会**回落到"系统默认内容"**(同窗口背景那条例)。必须 Clear 后再 `AddBkFill` 四态全透明色(透明填充算"有内容"就不会回落), 图形消失且**点击展开的原生行为保留**;
    - **展开按钮的点击热区矩形 = `RcItem.Left + 1 + 16×深度` 起、16×16**(开启 `app.ShowLayoutFrame` 后 PIL 实测布局边框红框标定, 每级步进恰好 16)。自绘 chevron 必须居中于这个矩形——画偏了(比如整体再右移半槽)图标就在热区右缘外, 点图标点不中展开; 模板实例的 `XEle_GetRect` 返回的是模板 rect 属性(相对容器), 参照系混乱, 标定要以布局边框为准;
    - 模板子元素绘制在 `XE_TREE_DRAWITEM` **之后**: 自绘回调里画行底/参考线/chevron 会被文字正常覆盖, 展开按钮透明填充不遮挡自绘 chevron;
    - `Tree_Item_` 有 NDepth/BExpand/NState(Leave/Stay/Select)/RcItem(全行宽, 左右各 1px 内缩); 项是 **NID 寻址**(非行号), 没有按行号枚举可视项的 API → 换肤同步文字色用 `RefreshData` 重走模板创建事件, 不能像 List 那样按可视行范围逐个刷;
    - **模板里的 ShapeText(itemID=2)是 Shape 对象不是 Element**, 不能调 `XEle_GetClientRect` 等元素 API(弹"输入句柄可能无效"错误框), 只能 `XShapeText_*`;
    - 叶节点同样保留展开按钮槽位(每项都有 itemID=0/1 的模板实例), 文字全树自然对齐——正好满足 shadcn 树"叶节点留空槽位"的语义, chevron 只需对"有子项"的项绘制;
    - **隐藏页上创建的树, 启动时的 `SetSelectItem` 选中态会丢**(展开态 `ExpandItem` 却保留); 页面真正显示后再补一次 Select 即可(可在页签 OnChange 里做)。

43. **图标抗锯齿: 直通 D2D 几何填充(`d2dicon.go` + `icons.go`, 探针 + 像素实测)**:

    - **炫彩把真的 COM 接口指针交出来了**: `XDraw_GetD2dRenderTarget(hDraw) → *ID2D1RenderTarget`、`XC_GetD2dFactory → *ID2D1Factory`(另有 WIC/DWrite 工厂与 `XDraw_GetD2dBitmap`)。paint 回调里可以纯 Go 直通调用, **不需要 cgo、不需要汇编**, D2D 的 AA 也对直通调用生效(实测 AA 模式 = PER_PRIMITIVE)。
    - **ABI 三规则**(破任一条: 轻则整帧被毁、窗口隐形, 重则崩溃):① 形参里出现 `float32/float64` 的方法一律不能这样调 —— 浮点参数走 XMM 寄存器, `syscall.SyscallN` 传不进去; ② 按值传递的 8 字节结构体要打包成一个 64 位整数(`D2D1_POINT_2F` → `f32(x) | f32(y)<<32`); ③ 其余全用指针/整数传是安全的。
    - **vtable 索引**(错一位就是调到邻居方法: 例如把 30 当 GetAntialiasMode 实为 SetTransform, 用垃圾指针设了变换 → 整窗不再呈现):`ID2D1RenderTarget` IUnknown 0-2 + `ID2D1Resource::GetFactory` 3, 方法从 4 起 —— `CreateSolidColorBrush=8`、`FillGeometry=23`(参数 (geometry, brush, layer=NULL), 是本方案的唯一出图口)、`SetTransform=30`、`GetAntialiasMode=33`; `ID2D1Factory` 直接继承 IUnknown —— `CreateEllipseGeometry=7`、`CreatePathGeometry=10`; `ID2D1PathGeometry` 继承 `ID2D1Geometry` 的 13 个方法 —— `GetBounds=4`、`Open=17`; `ID2D1GeometrySink` 的 `ID2D1SimplifiedGeometrySink` 部分自有方法只有 3~9 —— `SetFillMode=3`、`BeginFigure=5`、`AddLines=6`、`EndFigure=8`、`Close=9`。
    - **`ID2D1SimplifiedGeometrySink` 里只有 `Close` 返回 HRESULT, `SetFillMode/BeginFigure/AddLines/EndFigure` 全是 `STDMETHOD_(void, ...)`**(同理 `ID2D1RenderTarget` 的 Draw/Fill 全家也是 void):它们的"返回值"是寄存器残留垃圾。若照着"每个 hr 都校验"的直觉检查 `EndFigure`, 0x1f80(非 0)会被当成失败 → **静默回落折线渲染**: 症状是"代码改了, 画面一模一样", 单看输出发现不了 —— 排查时必须把每一步的 hr 打出来看。
    - **直通调用拿不到"当前元素"的变换与裁剪**(本库最大的坑):炫彩是在**刷新它自己的绘制命令**(`XDraw_*`)时才施加元素偏移/裁剪的, 直通 D2D 调用发生在回调里、命令队列之前, 因此拿到的是**父级(窗口)坐标系**。实测:元素内 (50,50)-(150,150) 的图标被画到窗口左上角; 反过来"先 XDraw 填充、后画图标"的按钮**看起来正常**(先有填充说明队列已推进), 而幽灵态图标按钮(背景/边框都是 None, 前面没有任何 XDraw 指令)的图标全跑到窗口左上角、正好被自绘的窗口图标盖住 → 表现为"标题栏按钮的图标消失"。
      **正解: 用官方提供的 `XDraw_ConvRect(hDraw, *RECT)` / `XDraw_ConvXY`** —— "从当前 UI 对象客户区坐标转换到画布对齐坐标, 自动适应 DPI", 本来就是给第三方绘图接口用的。换算结果就是**物理像素**(实测 150% DPI:元素在窗口 (60,60), 客户端 (50,50,150,150) → (165,165,315,315)), 直接喂给 D2D 即可, 尺寸与线宽随之自动变成物理像素, 不要再乘一次 DPI。
    - **WINDING 填充模式下所有子路径必须同向**:描边展开是"每段一个四边形 + 每个顶点一个圆帽", 而段四边形与圆帽的天然绕向相反, 重叠处 winding 归零被挖空 → 描边变成一排"梳齿"。修法:鞋带公式定号, 负号就反转点序(`sameOrientation`)。
    - **几何合法性自检**:索引错位/未 Close 的几何喂给 `FillGeometry` 会让 xcgui 整帧不呈现 —— **窗口彻底隐形(连边框都没有)、进程不崩、日志正常**, 只信截图;因此每次出图前用 `ID2D1Geometry::GetBounds`(idx 4;未 Close 的几何返回 `D2DERR_WRONG_STATE` 0x88990001)确认 hr=0 且 bounds 有限。
    - **实测对比**(200px 图标同屏对照):几何填充墨迹面积 10972 物理像素² vs 理论 10575(误差 4%), 折线回落 16064(胖 50%, 因为线宽被 `ceil` 取整、重叠段在 AA 下互相加深);X 图标边缘灰度层数 5 → 16。折线方案真正难看的原因是**折线点 `round32` 取整到逻辑像素**(18px 图标上 ±1.5 物理像素)+ 8 段离散的多边形棱角, 而不是"XDraw 没有 AA"——实测 XDraw 线条在 D2D 模式下也有 AA(两者像素统计几乎相同), 但几何是精确的。
    - **缓存**:几何是**设备无关**资源(归属 D2D 工厂而非渲染目标), 因此缓存按**工厂指针**作废(工厂变了=设备重建, 整体 Release 重建), 多个窗口(主窗/菜单/弹层各有各的 RT)共用一份几何;画刷是**设备相关**资源(必须由该 RT 创建、只能给该 RT 用), 所以每帧建一个、画完立刻 Release。绘制回调每帧都会新建 Painter(帧内对象), 缓存必须放**包级**。缓存 key = **(path 原文, size, strokeW)** —— **位置(ox/oy)不进 key**: 几何以自身原点构造, 绘制时用 `SetTransform(平移矩阵)` 移过去(见下一条)。
    - **位置外置: 用 RT 变换代替"把 ox/oy 烘进几何"**: 几何 key 不能含位置 —— 否则窗口一 resize(位置全变、尺寸不变)时每个图标都是新 key, 几何**全部重建**(CPU 描边展开 + CreatePathGeometry), 缓存还会随位置无上限累积。几何只按 (path, size, strokeW) 缓存, 绘制前后成对 `GetTransform` → `SetTransform(平移)` → 恢复。
      实测(图标页, 窗口尺寸在两档间来回切 10 次): CPU **78 / 93ms → 62 / 62ms**(省 20~33%); 像素对比三组(图标页亮/暗 + 基础页)两组 **0 差**、图标页 **6/2322000 像素**差(都在 AA 边缘, 最大灰度差 16 —— 坐标从"一次加法 `ox + p.x*s`"变成"`p.x*s` 再乘平移矩阵", 浮点加法顺序不同而已)。
      **前提是必须成对恢复**: 实测炫彩自己的绘制(`XDraw_*`)用的是**绘制当时的 RT 变换** —— "设了平移不恢复"会让紧随其后的 XDraw 矩形整体右移 150 逻辑像素(探针实测 的绿色块), 恢复后的蓝色块位置正确 ✓。故 `d2dStroke` 里严格 `defer rt.SetTransform(&prev)`(详见 §65)。
    - **回落路径**:`app.New(false)`(GDI 渲染)时 `XDraw_GetD2dRenderTarget` 返回 0 → 自动回落到折线描边(图标仍可见, 只是几何精度较低);GDI 模式实测不崩、布局正常。
    - **COM 直通底座在 `github.com/twgh/xcgui/wapi/d2d`**: 上面这些"通用但必须有"的东西 —— ABI 三条硬规则、vtable 索引、几何/画刷的创建与释放、几何合法性自检(`GetBounds`)、一步到位的 `NewGeometry`/`FillGeometryColor` —— 都由 `d2d` 子包提供; **COM 基础件(`ComProc`/`IUnknown`/`IUnknownVtbl`/`InterfacePtr`)在 `wapi/com` 子包**(d2d 用类型别名引用, 虚表字段仍写 `ComProc`)。封装形式照 **edge 包(WebView2)** 那套(每接口一文件、虚表按 Windows SDK 声明顺序**完整列出**、调用一律 `ComProc.Call` 带 `//go:uintptrescapes`、中文注释 + MSDN 链接), 接口结构体再按 COM 继承链**值嵌入**串起来:
      `IUnknown{ Vtbl }` → `ID2D1Resource` → { `ID2D1RenderTarget`, `ID2D1Geometry`→`ID2D1PathGeometry`, `ID2D1Brush`→`ID2D1SolidColorBrush` }、`ID2D1SimplifiedGeometrySink`→`ID2D1GeometrySink`、`ID2D1Factory`。于是 `AddRef/Release/QueryInterface` 只在 `IUnknown` 上实现一次、`GetFactory` 只在 `ID2D1Resource` 上实现一次; 各接口自己的方法用未导出的 `vtbl` 把同一个对象指针按**自己的**虚表类型重解释。
      三个必须记住的点(都有测试守着):
      ① **`vtbl` 必须取 `i.Vtbl`(对象首字段里存的虚表指针), 绝不能把对象地址当虚表**: 写成 `(*ID2D1FactoryVtbl)(unsafe.Pointer(i))` 时会从"对象 + 80 字节"处读垃圾当函数指针 → **跳到非法地址**(实测 PC=0x989680, 启动即 panic、退出码 2; `-H windowsgui` 构建看不到任何堆栈, 必须用带控制台的构建 + 重定向 stderr 才能抓到)。`TestVtblResolution` 用"假虚表 + 假对象"把这条钉死。
      ② **虚表字段顺序就是 ABI**: `TestVtblIndexLayout` 用 `unsafe.Offsetof` 断言 `CreateSolidColorBrush=8`、`FillGeometry=23`、`IsSupported=50`、`CreatePathGeometry=10`、`Open=17`、`SetFillMode=3`、`Close=9`、`SetColor=8` 等槽位; `TestStructLayout` + 包内编译期断言保证每个接口结构体**只有一个指针大小**(否则与 COM 对象布局错位)。
      ③ **nil 语义**: 构造函数在句柄为 0(未开启 D2D)时返回 nil, 调用方必须判空; nil 上只有 `Release`(每个接口都覆盖了判空版)与"自有方法"安全 —— 值嵌入下**被提升的基类方法**在 nil 外层指针上会先解引用嵌入字段而 panic(基类里判空也救不了), 这是 Go 的语义而非实现缺陷。
      `XDraw_GetD2dRenderTarget` / `XDraw_ConvRect` / `XC_GetD2dFactory` 的注释也强调"直通拿的是父级坐标系, 必须先 ConvRect"。

    - **用 `RT::DrawGeometry` 真描边**: `wapi/com` 的汇编转发器把"浮点形参"这条 ABI 限制解开之后(见 §65 的 `CallWithFloat32Stack`), 描边不需要在 CPU 侧展开 —— 折线保持**开口**直接交给 D2D(`DrawGeometry` + `RoundStrokeStyleProperties` 的圆角连接/圆头), 少数 fill 色点仍走 `FillGeometry`。
      实测(同几何同线宽各画 2 万次): **DrawGeometry 约 20ms, 填充"已展开的环"约 80ms(4 倍)**。但**真实场景看不出差异**: 图标页 45 图标/帧, 省下约 0.14ms/帧(相对 ~4ms/帧), 鼠标扫动与 resize 来回切的 CPU 都在噪声带里。**这条路线的主要收益是"质量 + 代码", 不是帧率**:
      ① 圆角连接/圆帽由 D2D 精确保证, 与 lucide 的 `stroke-linejoin/linecap="round"` 语义一致(旧版是斜接近似; 放大对比里三角形、八边形图标的"角变圆"); ② `d2dicon.go` 删掉 `strokeOutline/addArc/miterPoint` 约 300 行; ③ 几何既不含位置也不含线宽, 缓存 key 只剩 `(path, size)`。
      像素对照: 图标页 light **6958/2322000(0.30%)**、dark **7644(0.33%)**、基础页 **1051(0.045%)**, 差异全部集中在图标边缘(形状差异而非渲染错误); 图标墨迹量 142960 → 143083(+0.09%, 圆角比斜接略多一点面积)。
      **探针里的两个坑**: ① `ID2D1RenderTarget::Flush` 不能在有炫彩绘制回调里调 —— 会让进程静默退出(预热那步就没了下文), 所以性能只能按"提交路径耗时/大批量反压"来测; ② 想拿真实吞吐就得把次数堆到 2 万级, 60 次时四组全是 0.00ms。

    **填充几何的 figure 数必须少(图标总览页卡顿的根因, 重要)**:

    - D2D 填充**自重叠**路径走的是 stencil-and-cover:代价随**子路径(figure)数**近似平方增长, 与顶点数关系不大。探针实测(45 个图标同框、每图标同规模):9 个 figure/图标 → 4.1ms/帧;39 个 → 39ms;99 个 → 350ms;**199 个 → 1590ms**。
    - 若用"描边展开"写法(**每段一个四边形 + 每个顶点一个圆帽**, 一个图标上百个 figure), 45 个图标一帧要 **719ms** —— 表现为图标总览页每动一下鼠标就掉帧, 切页/换肤这种整窗重绘直接卡住近 1 秒。
    - 修法: 展开成**每条子路径一条闭合轮廓**(`strokeOutline`)。构造:两侧各按 w/2 偏移, 顶点处**外侧补圆弧**(round join)、**内侧取两条偏移线的交点**(精确, 不是直连); 两端各补一个半圆帽。改完实测 **2130 个图标平均 4.11 条轮廓/图标(最多 20 条, `virus` 尖刺多)**, 45 个图标一帧 **4.14ms**(= 空窗口基线, 图标开销归零)。`icons_test.go` 的 `TestIconGeometryFigureCount` 守住这条红线(>32 条就报错)。
    - **轮廓绕向必须一致**:所有轮廓的绕向由构造方式天然决定(与折线自身绕向无关), 因此 WINDING 填充下并集即笔画形状; 跨子路径的重叠(X 的两笔、太阳的辐条)也会正确并起来。填充子路径(lucide 的色点)则用 `sameOrientation` 归一。绕向不一致会让重叠区 winding 归零, 描边裂成一片片(`TestStrokeOutline` 守住)。
    - **此路不通的记录**: 本可以用 `ID2D1Geometry::CombineWithGeometry(UNION)` 一次性把图元汤合并 —— 索引按 Windows SDK `d2d1.h` 的声明顺序是 11(`GetBounds=4`、`CompareWithGeometry=8` 都已逐个验证), 参数全是枚举/指针(可 syscall), 但**实测调用直接崩(访问违例)**, 传不传单位矩阵都一样。顺带发现 `ComputeArea(13)`/`ComputeLength(14)` 的首参 `worldTransform` 是 `_In_`(**不能传 NULL**, 传了就崩), 而 `GetBounds(4)` 的 `transform` 是 `_In_opt_`(可 NULL)—— 本文件因此只用 `GetBounds` 一个几何方法。
    - **应用侧同样要省重绘**: 格子元素的 `MouseStay` 在"鼠标在元素内移动"时**每帧都会来一发**, 不判状态就会变成每个鼠标移动事件触发一次整窗重绘; 只在 hover 状态**变化**时 `Redraw` 即可(图标页 45 个格子扫一遍 CPU 从 0.5s/3s 降到 0.17s/3s)。

44. **lucide 图标资源预处理(生成 `icons/` 子包的常量表)**:

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
      `num` 遇到第二个小数点就停, 新增 `flag` 逐字符取标志位(现在贴 lucide 原始 path 也不会再断)。
    - **`fill="currentColor"` 图元要输出"填充 + 描边"两条子路径**:lucide 的这类元素(11 个图标里的色点)
      **同时继承根节点的描边**, 浏览器里是两者的并集 —— `circle r=.5` 实际画成半径 1.5 的实心点。
      像素实测(64px 图标):只填充 = 4x4 物理像素/16 像素墨量, 参考 = 12x12/120 像素。生成器用 `!` 前缀标填充,
      并对这类图元额外输出一条普通描边子路径, 与 lucide-react 渲染一致。
    - **放独立子包**(`github.com/twgh/xui/icons`):数据 ~360KB, 且**每图标一个 `IconXxx` 常量**。
      实测(同一基准比 `go build -ldflags="-s -w"` 的产物):只用常量 `icons.IconEye`
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
      (`parseIconPathEx` 的 complete 标志, 正是上面那类静默截断的检查点)、点落在 24 网格内、填充标记存在。
      预处理回归时立刻炸, 不用靠肉眼比对整本图标。
    - **像素对照**:同屏对照浏览器渲染的原始 SVG(headless Chrome + `--force-device-scale-factor=1.5`
      对齐 150% DPI), 32 个覆盖全部图元类型的图标逐格像素差 ≤ 4.2/255, 墨量比 0.95~0.97
      (残差来自 `#0a0a0a` 与参考纯黑的色差)。
    - **示例程序的「图标」页签**是这份资源的浏览入口(9x5 格 + 翻页), 也是"图标怎么用"的活文档:
      格子是普通自绘元素(各自 hover bg-accent), 翻页只换格子里的名字 + 重绘, 不销毁元素;
      **格子必须自己铺宿主底色**(透明子元素重绘不擦底, 见第八节 28), 容器元素要
      `EnableBkTransparent(true)` —— 否则暗色主题下容器是一整块白底、图标(浅色)直接看不见。
      45 个图标同框是**性能压力测试**: 它的卡顿逼出了第八节 43 那条"填充 figure 数必须少"
      (719ms/帧 → 4.14ms/帧); 另注意 hover 只在状态变化时重绘。
    - **任意图标集可直接导入**: 用户手里的数据通常是一整段 SVG(从 heroicons/feather/iconfont
      复制来的 `<svg viewBox="…"><path d="…"/></svg>`), 与 path 数据**共用 `Painter.Icon` 一个入口**
      (跳过前导空白看首字符是不是 `<`, 是就走下面这套归一化, 每帧一次判断成本可忽略):
      提取所有 `<path>`(支持 `<g>` 继承 fill/stroke/stroke-width)、按 **viewBox 归一到 24 网格**(任意网格:
      按较长边缩放 + 平掉 min-x/min-y)、按 fill/stroke 判定填充子路径(输出 `!` 前缀走 `FillGeometry`)、
      建议线宽取 `stroke-width` 并折算到 24 网格(调用方传 >0 时以调用方为准)。解析结果**按 SVG 原文缓存**
      (最多 512 条), 每帧只多一次 map 查找, 几何照旧走图标几何缓存; 想自己缓存或先校验就用 `ParseIconSVG(svg)`。
      **实测**: heroicons 官方 SVG 全文画出的星形与 lucide 内置形状一致;
      手写 16 网格 SVG(`<g fill="none" stroke="currentColor">` 两条线 + 一个 `fill="currentColor"` 色块)的
      "十字 + 实心方块"正确 —— 缩放与填充判定都对。**不支持**: `<circle>/<rect>` 等形状元素、transform
      属性、CSS class 里的样式(这些元素/属性被忽略, 不报错)。解析器必须处理**尾部空白**,
      否则会被当成"重复上一指令"从而判数据不完整。
    - **解析层在 `github.com/twgh/xcgui/svgp`**: path 语法解析(全命令/相对/紧凑写法/
      弧转贝塞尔)、SVG 文档遍历(`<g>` 继承 / viewBox)、子路径分组与绕向统一、d2d 几何组装都在那边
      (纯 Go, 不经过 DLL; 14 个单测), xui 只留图标语义: 24 网格归一化、`!` 填充前缀(svgp 的
      扩展语法)、SVG 原文缓存、GDI 折线回落。**像素等价**: 图标页 148/2322000(0.0064%)、
      dark 185、基础页 **4** 像素, 全是 `Z` 闭合语义(补重复点+OPEN → CLOSED)的 AA 级微差,
      墨迹量 143083 → 143082。选 svgp 而不是改 xcgui 自带 `svg` 包的原因:
      后者是 DLL 里的 C++ 实现, 无法灵活更改 —— 纯 Go 是唯一能继续改进的路径。

45. **hover 过渡动画(运行时 `anim.go`)**:

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

46. **重绘节流 `XC_SetPaintFrequency`(实测; 顺带记一条空操作 API)**:

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
    - **用法**:应用初始化 `a.SetPaintFrequency(10)`。**为什么是 10 不是 16**: 节流按"两次绘制的间隔"算,
      16 与运行时 16ms 的过渡定时器会周期性 beat(某次提前 0.1ms 就被推到下一帧 → 偶发 2 倍帧间隔), 10 留了裕量;
      16 以下也不再涨, 所以 10 是"够用且稳"的最低值。
    - **代价**:节流只"允许更早重绘", 不会凭空多画(静止时零影响); 单帧绘制成本不变(≈2ms), 所以帧率翻倍 ≈ 占用翻倍,
      且只发生在动画/过渡那 150ms 内。唯一要留意的是**整窗重绘很重的页面**(图标页 45 图标 ≈4.1ms/帧, 见第八节 43)
      在鼠标快速扫过时会多画几帧。
    - **顺带一条空操作 API(别用)**:`XC_SetTextRenderingHint`(`globalapi.go:320`) 与逐 Painter 的
      `XDraw_SetTextRenderingHint`(`xc/draw.go:1304`; `Painter` 直接暴露 `Draw`, `draw.go:17`, 想逐帧调也行)
      在 **Direct2D 模式(`app.New(true)`)下完全无效**:全局设 `TextRenderingHintAntiAliasGridFit`(=3, 唯一常量,
      `xcc/xcconst.go:1078`)前后, 自绘文字块 / 真 Button 标签 / 原生 Edit 文字**逐像素完全相同**(`ImageChops` 差异包围盒 = None);
      逐 Painter 设 3 与默认同帧也逐像素相同(彩色边缘像素 2265 = 2265)。原因: 它们是 GDI+ 的 TextRenderingHint,
      D2D 的文字不走那条管线。**组件文字本来就已经是 ClearType 级抗锯齿**(彩色边缘像素 2154/1474, 最大色偏 180/114),
      不存在"没抗锯齿/要换灰度 AA"这个可调点 —— 想关彩色边缘得另找路径(1.4.3 未见暴露)。

47. **弹层 / 对话框进出场动画(运行时 `presence.go`)**:

    语义逐条对齐本地 shadcn 参考实现(`shadcn/src/components/ui/*.tsx`):
    **dialog** = 遮罩 fade + 内容 fade+zoom-in-95(`dialog.tsx:42,64`, `duration-100`);
    **dropdown-menu / select** = fade+zoom+从锚点侧滑入 8px(`dropdown-menu.tsx:48`, `duration-100`);
    **tooltip** = 同上但没写时长 → tw-animate-css 的 `animate-in` 默认 **150ms**(`tooltip.tsx:44`)。
    缓动取 CSS 默认的 `ease` = cubic-bezier(0.25,0.1,0.25,1) —— `animate-in/out` 是 keyframes,
    用的是 animation 的默认缓动, **与 transition-\* 的 tailwind 默认 cubic-bezier(0.4,0,0.2,1) 不是同一条**。

    - **两个"不吃元素 alpha"的漏网之鱼**(都是淡入淡出时发现的):
      ① **D2D 直通绘制的图标**绕过炫彩的 alpha 合成(直通调用直接落在最终渲染目标上,
      元素 alpha 只作用于走 XDraw_* 命令队列的输出)→ 对话框淡入时关闭图标全黑不淡。
      修法 = d2dicon.go 出图前把元素当前 alpha(`XEle_GetAlpha`)乘进画刷颜色
      (窗口级 Painter 跳过);
      ② **输入框插入符**:竖条不随元素 alpha 淡(原生绘制)。最终方案 = **SetCaretWidth(0) 隐身**:
      插入符宽度只在 caret 创建(获得焦点)时生效 —— 对已显示的 caret 改宽度无效,
      清焦点再聚焦才生效(实测 w0/w1/w5 三档: 不重铸焦点全部 2px, 重铸后 5→8px、0→隐身)。
      所以 Show 里聚焦前先置 0, Presence done 恢复 1 并重铸焦点(清+聚, 一个回调内同步
      完成无中间绘制); Hide 开始先 XWnd_SetFocusEle(hWnd, 0) 清焦点, 插入符自然消失。
      焦点本身打开即给 —— 焦点环随整组淡入, 不闪。另两个弃用方案: ShowCaret 藏/显
      (与编辑框 blink 定时器打架, 连续闪烁); 焦点晚给(焦点环最后一帧突现, 闪一下)。
    - **运行时 API**(`presence.go`, 与 hover 共用 anim.go 的驱动器, 该处扩展了条目级
      时长/缓动/apply/done):`Presence(hOwner, key, visible, dur, apply, done)` 返回 0..1 进度,
      `apply` 每帧把进度落到元素透明度 / 窗口位置上, `done` 到位回调一次; 助手
      `SetElementsAlpha`(一组元素一起淡)、`SetSubtreeAlpha`(整棵**子树**逐个设 alpha ——
      元素透明度**不级联到子元素**, 只设 overlay 时标题/输入框等子元素会全不透明压在
      还没模糊的背景上, 实测踩坑)、`SetWindowAlpha` / `FadeWindowAlpha` / `SlideWindowAlpha`
      (窗口透明度 + 位移 + 自重绘)。调试用 `SetAnimationScale(f)` 把**全部**过渡放慢
      (12 倍时外部截图脚本才能抓到中间帧做像素比对)。
    - **已接入**:Dialog(遮罩+面板+全部子元素整组淡入淡出, `d.open` 立刻置 false 所以淡出期间
      键盘/点击已失效, 退场到位才 `showEleTree(false)`);Tooltip(气泡小窗 fade + 8px 滑入,
      退场 fade 完再 `SW_HIDE`; 已在显示中时只挪位置不重播动画)。
    - **shadcn 的 zoom-in-95 做不到**(留白):xcgui 的元素级 `ani` 动画(Alpha/MoveEx/Scale)对
      **自绘元素完全无效**(实测 `SetAlpha(0)`+动画后画面纹丝不动 —— ani 打在原生属性上,
      被 `AddEvent_Paint` 的自绘绕过), 而 Draw 只有 `SetOffset`(整数偏移)没有变换/缩放 API;
      所以 zoom 只能靠"改面板矩形+字号"模拟, 视觉会错位, 不做。Dialog 只做整组淡入,
      Tooltip/弹层做 fade+slide(slide 是 dropddown/tooltip 里最显眼的位移项)。
    - **踩坑(这几条都踩过, 务必记住)**:
      1. **窗口透明度在绘制时才落地**:`XWnd_SetTransparentAlpha` 改完不重绘**该窗口**,
         画面会一直停在上一次绘制时的透明度上(实测: 过渡值从 0.009 一路推进到 0.25,
         像素纹丝不动)。apply 里必须 `XWnd_Redraw(该窗口, true)`。
      2. 更邪门的是: 过渡挂宿主窗口、每帧重绘**宿主**窗口时, 气泡小窗的透明度跟着动了;
         只重绘气泡小窗自己反而不动 —— 机理未知, 但结论明确: **把进出场过渡挂在宿主窗口上**
         (tooltip 就挂触发元素所在的宿主窗口), 让 tick 的整窗重绘带动一切。
      3. **alpha 精确 = 0 会让 xcgui 停止合成该窗口**(后续怎么调都不显示, 画面冻结在
         停止前那一帧; 调试日志显示 apply 已跑到 v=1.0 而像素不动)。两条触发路径:
         显示前预设 0, 以及**退场淡出结束在 0**(表现: 气泡第二次显示永远停在约 10%
         透明度)。修法 = `SetWindowAlpha` 统一抬底 `xui.AlphaFloor`(1/255), "消失"
         一律由退场 done 里 SW_HIDE 完成; 进场首帧闪现也用预置 1/255 压掉。
      4. 异型置顶小窗(`Window_Style_Nothing` + `Shaped` + NOACTIVATE)自己的窗口定时器
         不可靠(约 100ms/帧), 别把过渡挂它身上(同 2, 挂宿主窗口)。
      5. **普通窗口(非透明类型)的 `XWnd_SetTransparentAlpha` 完全无效**(exstyle 里没有
         WS_EX_LAYERED, xcgui 也不会自动加) → native 弹层(menu / combobox 的下拉窗)默认做不了
         淡入; 要淡入只能自己给 HWND 加 WS_EX_LAYERED + SetLayeredWindowAttributes
         (代价: ClearType 变灰度 AA + 合成开销), 未做。**但把弹窗转成
         `Window_Transparent_Shadow`(官方 DrawMenu 菜单阴影同款)后就是透明类型, 整窗 alpha
         可用 —— 但菜单进场**没有**用它: 整窗 alpha 会让底色一起半透明、后面的页面文字透上来,
         改成只淡内容又会先闪一帧空白面板, 故菜单不做进场动画(见 64/99); 位移滑动不可行(会崩, 见 64)。**
      6. 首帧闪现: `SWP_SHOWWINDOW` 后的首次绘制带着上一次的 alpha(255) → 进场前先
         `SetWindowAlpha(1/255)`(见 3, 不能是 0)。
    - **验证**(探针 + `SetAnimationScale(12)` 放慢 + 外部逐帧像素实测):气泡淡入
      255→254→245→232→201→175→141→117→91→74→60→45→37→27→22→18→14→12→**10**
      (17 帧, 150ms×12)且顶部 y 36→**33**(8px 逻辑滑入); 1× 常速 77→24→**10** 正常完成;
      对话框遮罩点 255→251→247→241→237→235→232→231→**229**(淡入)与反向(淡出)平滑往返,
      淡出到位后内容树隐藏(`窗口最暗` 回到 255)。

    - **backdrop-blur-xs 实现**(`ui/backdrop.go`):
      思路 = 打开时 GDI `PrintWindow(PW_RENDERFULLCONTENT)` 抓窗 → Go 盒式模糊 ×4 趟
      (backdrop-blur-xs=4px 逻辑)→ **元素 BkM 图片填充**显示在遮罩之下(元素透明度能跟着
      遮罩一起淡入。元素 paint 本身也能画位图, 见 17 条; BkM 路线能让透明度跟随遮罩, 故保留)。
      `XImage_LoadFromData` 在实际加载的 dll 里有导出; 若外面套 recover 兜底,
      真正的失败会被静默成"只有变暗"。坑:
      1. **BkM 图片的状态用错就永不渲染**: 元素的"离开"态是 `Element_State_Flag_Leave`
         (0x0010), 不是窗口的 `Window_State_Flag_Leave`(0x0001)。用窗口常量注册到元素
         BkM 上, 元素 show/alpha/矩形全对也不出图(症状极具迷惑性, 状态转储看不出错)。
      2. **BkM 图片按 1:1 逻辑像素、居中、不缩放绘制**: 物理尺寸(1200x780)的快照喂给
         800x520 逻辑的元素会错位+被裁(网格采样定位到"图片落在元素中央")。修法 = 加载前
         先把模糊图面积平均缩放到窗口逻辑尺寸(`scaleRGBA`), 居中 1:1 即正好铺满。
         `BkObj.SetAlign(Left|Top|Right|Bottom)` 实测**对图片对象无拉伸效果**(别试了)。
      3. **注册走 `XBkM_AddImage(XEle_GetBkManager(ele), state, hImg, id)`**: 带 id 的
         管理器级注册在"窗口子元素直接挂"的环境里实测可靠; `Element.AddBkImage`(无 id)
         某些布局盒子嵌套能渲染、平铺窗口里不渲染, 原因未查明 —— 统一用
         带 id 的路线绕开。注意重复注册前先 `XEle_ClearBkInfo`(累积 >20 个炫彩报错)。
      4. **没有消息循环就没有一切**: 只 Sleep 不跑 `a.Run` 时, Presence 淡入淡出
         (窗口定时器)全不工作, 关闭也不会真的隐藏。程序化验证要用
         `w.SetTimer + AddEvent_Timer` 驱动阶段, 再 `a.Run`。
      5. 枚举窗口子元素要用 `XWnd_GetChildCount/GetChildByIndex`(喂 `XEle_GetChildCount`
         恒为 0); `XEle_GetAlpha/GetBkInfoCount/XObj_GetType` 是排查这类渲染问题的
         状态转储三件套。
      6. **快照管线**: 按窗口**物理像素**抓进 DIB → 同一次遍历里做
         BGRA→RGBA + 盒式降采样到 1/backdropBlurDown 逻辑尺寸 → 双趟盒式(半径 2) →
         成对平均放大回**精确**逻辑尺寸。坑与实测:
         - **`PrintWindow` 不缩放, 它按 1:1 物理像素把整窗画进目标 DC 并裁剪到 DIB 尺寸**:
           1200×885 的窗口抓进 800×590 的 DIB, 逐点等于"物理全幅图左上 800×590 的像素"
           (与裁剪假设均值差 **0.0**/255, 与缩放假设差 20.0/255)。所以 **DIB 必须给足物理
           尺寸** —— 按逻辑/半逻辑尺寸要图只会拿到左上角一块, 后级再放大就成了"遮罩只剩
           左上角、还被放大好几倍"。
         - 想要小图只能自己降采样; 而这一趟本来就要读遍全部源像素(全屏物理 3.9M), 顺手把
           通道交换与盒式平均一起做掉 —— 实测降采样比"只抓屏"几乎不增加耗时(常态 26→27ms)。
         - **不要按整分辨率模糊 ×4 趟(半径按 DPI 放大)再事后缩放**: 模糊代价随面积增长,
           而全屏物理面积是逻辑的 DPI² 倍 → 2560×1528 @150% 全屏 **160ms**、普通窗口 **39ms**,
           表现为"全屏时点按钮弹窗比正常窗口慢"。
         - 模糊改在 1/backdropBlurDown(2) 逻辑分辨率上做: 1707×1019 整分辨率双趟要 65ms,
           半分辨率只要 14ms; 双趟盒子在半分辨率的 σ≈2.0, 放大回逻辑后 ≈4.0, 与 CSS
           backdrop-blur-xs = blur(4px) 同量级(半径不该再乘 DPI)。
         - 实测(热态, 含 `XImage_LoadFromGoImage` ~3ms): 常态 800×590 **~36ms**(原 39)、
           最大化全屏 **~61ms**(原 160)、铺满整屏同量级。注意窗口状态刚变时第一帧
           `PrintWindow` 有 DWM 预热(~75ms), 2~3 帧后落到 ~30ms。
         - **`PrintWindow` 失败不要退 BitBlt**: 对 D2D 自绘窗口从窗口 DC BitBlt 拿到的是
           **纯黑**(实测整幅 mean=0.0/255), 糊上去就是一块黑板; 现在直接放弃模糊(调用方
           保持背景元素隐藏, 对话框功能不受影响)。
         `PrintWindow` 抓的是 BGRA, 且强制 A=255(GDI+ 认 alpha, 0 会整块透明)。
      7. **阴影窗口(Window_Transparent_Shadow)抓屏有黑边**: Win32 矩形含四周阴影边带
         (12 逻辑像素), `PrintWindow` 把半透明阴影抓成**纯黑**(alpha 丢失),
         模糊后又往画面里晕一圈 —— 表现为遮罩四边黑边。修法 = 抓屏后按
         `XWnd_GetBodyRect`(Body 矩形, 窗口相对**逻辑**坐标)裁掉边带再模糊;
         非阴影窗口 Body=整窗, 仅在 Body 严格小于窗口时裁。
      8. **对开着 backdrop 的阴影窗口调 PrintWindow 会死锁**: refreshBackdrop 在 Show 里
         抓屏没事(此时 backdrop 图片还没装上), 但对话框开着时再对它 PrintWindow
         (BkM 图片 + PW_RENDERFULLCONTENT 重入 D2D 渲染)会挂死消息循环 —— 探针
         验证这类窗口要用屏幕截图(DXGI), 别用 PrintWindow。
      9. **裁边后必须把元素摆回 Body 原点, 否则内容整体偏移**: 子元素坐标系的原点是
         Win32 整窗**左上角(含阴影带)**, `XWnd_GetBodyRect` 返回的 (12,12) 就是这个
         Body 相对于原点的偏移, 不是随便的裁剪建议。只裁快照、仍把元素画在 (0,0),
         模糊背景会整体上移/左移一个阴影带宽 —— 实测主窗标题行被削掉一半。
         修法 = 裁剪后按 `backdropPlacement` 把元素放在 (body.Left, body.Top)、
         尺寸 776×566 缩放注册; 窗口缩放重排(layout)也走同一函数, 保持对齐。
         探针实测: `XWnd_GetRect`=800×590(含阴影), Body/Client=12,12→788,578(776×566)。
      10. **遮罩要铺 Body、不能铺整窗**(修正): 遮罩矩形原来用整窗矩形, 在阴影
         窗口上会盖住四周 12px 的阴影带, 把窗口自身的投影**再压暗 10%** —— Body 是圆角的,
         四角露出的阴影带最多, 肉眼就是"遮罩四角多出黑色块"(实测: 窗口铺满整屏但未最大化
         时四角 #11111b、Body 内 #ffffff; 修后开/关对话框的四角像素**完全一致**)。浏览器里
         overlay 盖的是 viewport(= 内容区)、窗口外框仍归系统, 语义也一致。
         实现要点: 遮罩**元素本体**摆到 Body 的绝对位置, 而**它的子元素坐标是相对父元素
         原点的** —— 所以面板/页脚/标题/输入框一律用"遮罩局部坐标"(尺寸按 Body 算, 从
         (0,0) 起算), 绘制(onPaint)与"点遮罩关闭"的命中判定(onOverlayDown)也都用同一套
         局部坐标, 不需要任何偏移换算。踩过: 改完第一版忘了这一点, 子元素仍按窗口绝对坐标
         摆 → 整组内容右下偏移一个阴影带宽(12px), 表现为"按钮/输入框挤到弹窗右边、
         左右内边距 27/5 不对称"(像素实测); 现在左右内边距对称 15.3/15.3、面板在 Body
         内水平垂直都居中(实测中心 399.7 vs 400、上下留白 143.3/143.3)。

         48. **`XC_GetTextShowRect` 的 width=0 是"按一个字符宽折行", 不是"不限宽"**(探针 + 像素实测, 由 pagination 组件暴露出来):
         - `app.GetTextShowRect(str, -1, hFont, align, 0, &sz)` 对 `"上一页"` 返回 `CX=15 CY=59`
         —— CX 只是**一行的宽度**(一个汉字), CY 是"行数×行高"; 对 `"保存"` 返回 `CX=15 CY=39`
         (2 行), `"accessibility"` 返回 `CX=10 CY=252`(11 行)。看上去像"宽度只有一半",
         早期的 `CX*2` 校正只对**恰好 2 个字**的按钮文案凑巧对(`保存` 15→30, 真实 30),
         换成 3 个字的 `上一页` 就偏小(30 vs 真实 45) → 分页组件的"上一页/下一页"文字被
         矩形裁掉半截(实测只剩 "上-" / "-页")。对话框页脚同样踩过: 英文按钮 "Cancel"
         量出的宽度不足, 文字被挤成两行。**现在统一用运行时 `xui.TextWidth`(内部给足宽 +
         NoWrap), 不要再用 `CX*2` 或 width=0 的量法。**
         - 正确用法: **给足 width, 并加 `TextFormatFlag_NoWrap`**,
         `GetTextShowRect(str, -1, hFont, TextAlignFlag_Left|TextFormatFlag_NoWrap, 1000, &sz)`
         返回的 `CX` 就是单行真实逻辑宽度(14px 字号的汉字恰好 15/字, 与绘制结果一致),
         不需要任何倍率校正。量宽必须与绘制同字体、同 DPI。
         - 顺带两条组件侧的实测结论(分页组件踩出来的): ① 自绘文字矩形**右/左边界取到壳体边缘**、
           只在一侧留 padding, 量宽有 ±1px 误差也不会截字; ② 组件在 `OnChange` 之前的立即重绘
           (`XEle_Redraw(ele, true)`)会连带清掉窗口的待重绘区域, 调用方在回调里改的
           **原生控件文字(Label 等)会滞后到下一次无关重绘才出现** —— 组件改状态后一律用
           `Redraw(false)`, 原生控件自己改完文字也要显式 `Redraw(false)`(例如 `SetIcon` 后
           本来就带 `w.Redraw(false)`, 同理)。

         49. **Tooltip 被"迟到的 MouseLeave"秒关: 密集元素上会表现成"提示根本不显示"**(图标网格实测 + 日志定位): 同一个 Tooltip 绑定到一整片密集元素时(9×4 个 77×60 的格子),
         鼠标从一个格子划到相邻格子, 实测事件顺序是
         `leave(A) → show(B) → leave(A) 又来一次`(xcgui 的 stay 元素切换与鼠标移动是两条到达
         路径), 第二条 leave 会把刚进场的气泡立刻关掉 —— 日志里能直接看到
         `show hEle=616 → fresh 进场 → Hide → show hEle=612 → Hide → 退场到 0`, 气泡只亮了
         十几毫秒, 肉眼就是"tooltip 全都不显示了"; 绑到稀疏元素上(标题栏按钮)时不会撞上,
         修法 = 气泡记住自己当前"描述"的元素 `shownEle`, `MouseLeave` 里只有
         `hEle == shownEle` 才真隐藏, 迟到的 leave 忽略(需要强制关掉的场景走 `HideTooltip`)。
         排查手法值得记一笔: 在 `show/Hide` 与 presence 的 apply/done 里追加写文件日志,
         跑一遍鼠标轨迹就能把"窗口没建/没显示/被谁关掉"三类原因区分开。

         50. **Tooltip 是深色气泡, 压在深色背景上肉眼容易漏看**: 亮色主题下气泡是
         `bg-foreground`(近黑)+ 白字, 悬浮在标题栏/暗色面板上方时几乎和背景融为一体,
         截图判断时别急着判"没显示" —— 先按元素几何算出气泡应该出现的矩形, 或者干脆用
         日志/pixel_scan 确认, 否则很容易把正常渲染误判成 bug。

         51. **隐藏容器里建的 Tree, 首次显示"只有行底和 chevron, 文字全空"**(`ui/tree.go`):
         - **现象**: 树建在启动时隐藏的祖先容器/页签里, 首次显示时行底/缩进线/chevron/选中态都对,
           但**每行都没有文字**; 再切一次页就恢复, 不会自己恢复。
         - **根因**: 原生 Tree 的项模板是在**可见状态下的布局排版**里实例化的。树建在隐藏的祖先容器里、
           尺寸又只由创建参数给定时, 全程没有一次"可见时的排版"落到它身上 → 模板从未实例化;
           `XE_TREE_DRAWITEM`(行底/chevron/参考线)照常跑, 但模板里的 ShapeText 没有内容。
           **绘制不负责补模板实例**。
         - **修法**: `NewTree` 挂 `AddEvent_Paint` 记下"已首次绘制", 再用炫彩定时器
           (`treeWakeTimerID`, 1ms ≈ 下一帧)排一次 `RefreshData + Redraw(false)`, 让刷新发生在绘制回调
           之外。刷新不依赖调用方, 隐藏容器/多页签/对话框里建的树都自愈; 代价是首帧后有一次一次性
           刷新(只刷可视项)。
         - **判断依据**: `XE_SHOW` 不会从父容器级联给子元素, 所以"我可见了"不能用 SHOW 事件判断,
           只能用首次绘制当信号; 若在页签 `OnChange` 里手动 `RefreshData` 后立即正常, 说明原因是
           "模板没实例化", 而不是几何/主题/颜色。

         52. **ComboBox 弹层高度: 内容高 = n×(项高+1) + 2×pad + 5, 取该值才无滚动且留白对称**
         (实测): 症状是 6 项弹层悬浮最后一项时
         hover 背景底边正好贴住弹层底边、滚轮还能再滚一下。
         - ListBox 内容有**固有内缩**: 首项 top = pad + 3(上下各固有 3px、左右固有 2px), 行步进 =
           项高 + 1px(相邻项 top 差 29 @ 项高 28); 内容高 = n×(项高+1) + 2×pad + 5
           (顶固有 3 + 底固有 3 − 最后 1px 间隙)。公式在 pad=0/4/8 与 n=3/5/6/10 上均验证过。
         - 弹层高度 < 内容高 → 出滚动条且"差多少滚多少"(实测 6 项 H=180 内容 187 → range=10, 能再滚
           一下); > 内容高 → 底部多留白。**高度取内容高**时: 无滚动、顶部/底部留白对称(各 pad+3)。
         - 固定模式必须在**弹层打开前**`EnableDropHeightFixed(true) + SetDropHeight(h)`(组件在构造期
           调, 见 `ui/combobox.go` 的 `comboDropHeight`)。
         - **正确表现**: 悬浮最后一项时 hover 背景底边与弹层底边框间应有 ~pad+3 留白, 滚轮不产生位移。
           排查时可在 `AddEvent_ListBox_DrawItem` 打印 `pItem.RcItem` 拿每项真实绘制矩形(项坐标与滚动量
           都能看出); 用 `GetScrollBarVObj.GetRange/XSBar_GetCurPos` 判有无滚动。

53. **Pagination: 折叠必须"恒定槽位", 且组件在 OnChange 前不要用立即重绘**(`ui/pagination.go`):
    - **组件路线 = 纯自绘单元素**(同 slider, 不拼 Button 子元素): 页码数量随数据变化, 拼子元素要反复增删/重排; 条目 = `[上一页][1][2]…[n][下一页]`, 内容居中(justify-center), 页码 32 方形(ghost / 当前页 outline), 上一页/下一页 = `chevron(size-4)` + 文字; 悬停 `xui.HoverItem` 逐项过渡、按下 `active:translate-y-px` 下沉 1px; 到边界时上一页/下一页按禁用画(50% 透明且不响应鼠标)。API `NewPagination / SetTotal / SetPageSize / SetPage / Page / PageCount / OnChange`(`SetPage` 不回调, 供调用方同步数据游标)。
    - **折叠策略必须"恒定 7 个槽位"** —— 首段 `1 2 3 4 5 … n` / 中段 `1 … p-1 p p+1 … n` / 末段 `1 … n-4 … n`。shadcn 文档那版三段分别占 6/7/7 个槽位, 翻页跨段时整行**宽出一个页码位**, 居中排布下"下一页"跟着右移 —— 同一坐标连点会点不到; 恒定槽位后同坐标连点 10 次稳定到第 11 页。
    - **组件在 `OnChange` 之前用 `Redraw(true)` 会吞掉调用方的待重绘区域** → 调用方在回调里改的原生控件文字(Label 等)滞后到下一次无关重绘才出现; 组件改状态后一律 `Redraw(false)`, 原生控件自己改完文字也要显式 `Redraw(false)`(同第 48 条)。
54. **元素销毁事件会为子元素逐个触发(含嵌套), 顺序在窗口 `XE_DESTROY` 之后**(探针实测):
    - **三种销毁路径都触发** `XE_DESTROY`(33) + `XE_DESTROY_END`(42): ① 元素显式 `Destroy`; ② 直接挂在窗口上的元素随 `CloseWindow` 连带销毁; ③ **嵌套在子容器(LayoutEle)里的元素同样触发**。
    - **顺序**: 窗口的 `XE_DESTROY` 先到, 再按层级逐个销毁子元素(实测日志 `WINDOW → C(嵌套) → A(直挂)`)。
    - **意义**: 这是"把订阅/缓存绑到元素生命周期"的依据 —— 运行时的 `xui.OnThemeChangeFor(hEle, fn)`(主题订阅随元素销毁自动反注册)与 `anim.go` 里 `AddEvent_Destroy_End` 清理 hover 状态, 都挂在这上面。对裸句柄注册要用 `widget.NewElementByHandle(hEle)`, 且**必须走 `AddEvent*` 事件总线**(不能用 `XEle_RegEventC` 那套有 2000 上限的注册)。

55. **布局尺寸语义实测全集: `Fill` ≠ "剩余空间", `Weight` 只在父级尺寸"确定"时生效**(布局实测):
    - **`Fill` = 父级整尺寸(减自身四边外边距)**, 不是"剩下的都给它": `Fill/Fill + margin 24` 在 800×600 的父级里得到 752×552 ✓; 但**前面还有兄弟项时它不会让位** —— 竖向 `[Fixed 60][Fill]` 里 Fill 拿到的仍是整高 600(而非 540), 直接溢出容器。竖向堆叠想要"剩下的高度"不能用 Fill。
    - **`Weight` 才是"按比例分剩余空间"**: 竖向 `[Fixed 60][Weight 1]`(父 400)→ 340 ✓; `Percent` 同源(Percent 50 在同一个父级里 = 170, 即剩余空间的百分比)。
    - **但 `Weight`/`Percent` 只在父级尺寸"确定"时才成立**: 父级是 `Fixed`、或是被 `Weight` 解析出来的尺寸、或是**窗口本身** → 正常; 父级是 `Fill` 或 `Auto` → **恒为 0**(实测: `root(Fill)` 里的 Weight 高度 = 0, 换成窗口直属后立刻正确)。**宽度方向不受影响** —— 横排里 `Weight` 列在 Fill 宽的行里一直正常。
    - 推论(已踩): **竖向主链中间不要垫 `Fill` 容器**。要从窗口开始一路 `Weight`, 每级都是确定尺寸, 下一级的 Weight 才算得出来 —— 窗口骨架直接写成 `[标题栏行 Fixed][内容区 Weight 1]`, 中间不要垫 `root(Fill)`。
    - **`Auto` = 按内容自撑**: 两张 Auto 高度卡片竖排各自高度准确(120/184, 与内容理论值一致)、互不挤压 —— 这是"卡片不用手算高度"的正解。Auto 卡片里**不要放 Weight 子项**(Auto 父级 → 0), 里面用 Auto / Fixed 就好。
    - **要让"面板随窗口长高", 整条竖链必须全 Weight**: 窗口 → 内容(Weight) → 页(Weight) → 内容区(Weight + 外边距) → 行(横排, 高 Weight) → 列(宽 Weight + **高 Weight**) → 卡片(高 Weight) → 内容区(Weight) → 面板(Weight)。实测面板 700 高窗口下 628、900 高下 828, 逐级跟随 ✓。注意"列"的高度: `xui.VStackWeight` 用的是 `Fill`(= 父级整高), 列内再用 Weight 就废了 —— 要长高的列应自己写"高 Weight"的容器。
    - **普通 `widget.Element`(非 LayoutEle)可以当布局项**: 被父布局排版、跟随缩放都正常(探针对照 A/B 一致)。而 `EnableLayout / EnableHorizon / EnableAutoWrap` 这些"**当容器排子项**"的开关只有 LayoutEle(与窗口)有 —— 所以卡片能否排自己的内容取决于它是不是 LayoutEle(见 56)。
    - **`Show(false)` 的子项会被布局跳过**(排成 0x0): 页签式"同位置叠放多页"因此不需要绝对定位 —— 竖向堆叠里隐藏的页不占位。改可见性后**必须重排宿主**(`AdjustLayout`), 否则新显示的那页拿不到尺寸(实测只排那一页时仍是 0x0)。
    - **窗口必须 `Show` 过之后布局才有非零尺寸**: `Show` 之前即使调 `AdjustLayout`, root 恒 0x0。
    - 另: `xc.XEle_SetPadding(hEle, l, t, r, b)` 是元素级内边距, **已实测可用于 LayoutEle**(Fill 子项按扣除 padding 后的宽度排版, 见 103); 带外边距的子容器当 padding 时, 内层 Fill 不会跟着缩窄(Item 曾因此丢掉右内边距)。

56. **Card 是布局容器: 内容区随卡片尺寸自动联动**(`ui/card.go`):
    - 卡片底层是 `widget.LayoutEle`(竖直流式 + `EnableAutoWrap(false)`), 内容区 `ContentStack(space)` 是它的 **Fill 宽 + 外边距 = 内边距** 的子项 —— 卡片被父布局拉伸时内容区与其中的控件自动跟随。调用方不要拿 `ContentWidth` 去算宽度: 它取的是**创建那一刻**的宽度, 卡片一缩放就错。
    - 内容区高度是 **Auto**(按内容自撑), 于是卡片高度可以 Auto(自撑)也可以固定/Weight, 内容一律从上往下排。内容要撑满卡片高度(表格/树)时: 卡片高度给 Weight, 再对内容区与目标子项声明 `xui.ItemFillHeight`(Weight 1, 受 55 的"父级确定尺寸"约束)。
    - 一个卡片只有一个内容区: 重复调用 `ContentStack` 返回同一个(只改 space)。页脚 `CardFooter` 以**卡片**为父句柄创建、声明宽度铺满 —— 内容区高度给了 Fill/Weight 时会自动让出页脚那一段。

57. **异型透明窗(Shaped)的"整窗透明度"只认第一次调用, 之后永远冻结**(实测: Tooltip 气泡"窗口明明显示了却看不见"的根因):
    - **症状**: 提示气泡**确实被显示出来**(枚举窗口可见、矩形正确、位置还随进场动画滑动), 但屏幕上什么都没有。
    - **排查方法**: ① ⚠️ 有 tooltip 小窗(懒创建, 带 `WS_EX_TOOLWINDOW`)时, `Process.MainWindowHandle` **返回的是小窗而不是主窗口**, 用它算坐标会让后续悬停/抓图全错; 应"枚举进程顶层窗口, 取可见且面积最大"; ② 用 `PrintWindow` 抓小窗自己的内容 vs `CopyFromScreen` 抓同一块屏幕, 可二分出"**画是画了的**, 只是屏幕上没有" ⇒ 合成/透明度问题; ③ `user32!GetLayeredWindowAttributes` 读出 layered alpha = **3/255**(窗口可见但几乎全透明)。
    - **根因**: `XWnd_SetTransparentAlpha` 对 `Window_Transparent_Shaped` 窗口**只在第一次调用时生效** —— 第一次调用把窗口切成 layered 并写进 `SetLayeredWindowAttributes`, 之后无论设什么值、`XWnd_Redraw(本窗, 立即)`、`XEle_Redraw`、`SetWindowPos` 移动、重绘宿主窗口、`ShowWindow`, **layered alpha 都停在首次那个值不再变化**(实测 3 → 200/120/60/90/30 全部无效)。于是"进场动画第一帧"的极小值被永久钉住 ⇒ `FadeWindowAlpha` / `SlideWindowAlpha` 这类**整窗 alpha 动画在异型窗上等于没做**。(`Window_Transparent_Simple` 从建窗起就不是 layered 模式, 该 API 同样不生效。)
    - **修法**(`ui/tooltip.go`): 进出场**改用元素 alpha**(`xui.SetElementsAlpha` / `XEle_SetAlpha`, 由框架在合成阶段逐帧乘上去, 每帧都生效; 对话框整组淡入同理, 见 47); 整窗 alpha 只在**窗口显示出来之后**设一次 `1`(不透明)钉住; 位置滑动仍走 `SetWindowPos`; 驱动计时器仍挂宿主窗口(异型小窗自己的定时器会掉到约 100ms/帧)。
    - 首次进场不需要"把窗口 alpha 压到底"防闪: 元素 alpha 直接以 0 起步, 首帧什么都不画即可(**元素** alpha 允许为 0; 只有**窗口** alpha 不能为 0, 否则 xcgui 会停止合成该窗口, 见 `SetWindowAlpha`)。
    - **通用判据**: `IsWindowVisible=true` + 矩形正确 + **屏幕像素为空** ⇒ 先怀疑整窗 alpha / 窗口形状, 不要只凭"窗口可见"下结论。

58. **独立置顶小窗不跟随主题切换重绘, 也不在"已注册窗口"清单里**(`ui/tooltip.go`):
    - **现象**: 气泡已经挂在屏幕上时点标题栏"切换主题", 页面与标题栏都换了配色, **气泡还是原来的配色**, 直到鼠标移开重新进场才更新。
    - **原因**: `SetTheme` 只做三件事 —— 重刷 `BindWindowBackground` 的底色、跑 `OnThemeChange` 订阅回调、重绘 `RegisterWindow` 登记过的窗口(见 manager.go)。Tooltip 的气泡是组件自建的**独立置顶小窗**, 既不在登记表里、也不是"组件窗口", 于是没人重绘它; 宿主窗口重绘只带动**合成**(透明度), 不动它的**内容**。
    - **修法**: 组件自建窗口要自己订阅主题 —— `xui.OnThemeChangeFor(元素句柄, fn)`(元素随窗口销毁时自动反注册, 见 54), 回调里**重跑一遍 show**: 重新量文字、重算气泡几何、重绘。重跑而不是单纯 `Redraw` 是因为主题可能换字体/字号, 尺寸得按新度量重算(目标元素已被 `Unbind` 时退化为纯重绘)。
    - **结果**: 悬停标题栏主题按钮(气泡弹出, 亮色下是深色块) → 光标不离开按钮直接点它切主题 → 同一区域抓图, 气泡立刻变成浅色块(浅色像素 4432); 页面元素的"Icon"气泡同样正常(深色像素 +1026)。
    - 顺带: 窗口扩展样式(`TOOLWINDOW|NOACTIVATE|TRANSPARENT`)用 `window.NewEx` 在建窗时一次传进去, 不必 `New` 之后再 `GetWindowLongPtrW/SetWindowLongPtrW` 读改写(前者也不再带上 xcgui 默认给的 `WS_EX_CLIENTEDGE` 位)。

59. **"Auto 高的元素"挂在错的父级里时, 它内部 Fill 子项的宽度会按**祖先**宽度算(右溢出两侧内边距)**(探针 , 表单页实测):
    - **现象①(实测)**: 表单页卡片里的原生 `Edit`/`ComboBox` 右边缘正好压在卡片边框上, 宽度 = **卡片宽**(568) 而不是内容区宽(536), 溢出量 = 左右内边距之和(32); 同一张卡里的分隔线同样溢出。
    - **现象②(实测)**: "下方两个卡片盖住上方卡片" —— 行高只有 `Weight`(按剩余空间等分)可用, 卡片比行高时后面的行直接压在它上面(实测: 输入框卡 430 高, 行只分到 363, 第二行卡片上沿 379 → 盖住 51px)。
    - **根因**: xcgui 的布局分"量算"与"定位"两趟。链上存在**尺寸依赖父级**的容器(Fill/Weight)时, 量算趟会对 `Auto` 高的容器用**外框宽度**排它的子项, 而定位趟只摆位置、不再重设子项宽度 —— 于是子项宽度停在"外框宽"上(多了两侧内边距)。
    - **实测矩阵**(行高 Weight, 每列一张卡片; 子项是原生 Edit):
      | 列高 | 卡片高 | 子项宽 | 判定 |
      |---|---|---|---|
      | Fill / Weight / Fixed | Auto | = 卡片宽 | ✗ 溢出 32 |
      | **Auto** | Auto | = 卡片宽-32 | ✓ |
      | 任意 | Fill / Fixed | = 卡片宽-32 | ✓ |
      | 列高 Fill + 再垫一层 Auto "槽位" | Auto | = 卡片宽 | ✗ 垫一层不解决问题 |
      另外: **行高 Auto 也不行** —— 横向行里 `Weight` 宽的子项会塌成 0 宽(实测: 四个列全 0 宽)。
    - **推荐骨架**: 每个"栅格"用 **一行**(行高 Weight) + **`Weight` 宽的列, 列高声明 `Auto`**, 卡片按内容顺序**竖着排在列里**; 整宽卡片直接挂 `body`(实测正确)。竖排还顺带解决现象②: 同一列里两张卡是"先后关系", 不再有行与行之间的重叠。
    - **代价**: 页面内容比窗口高时会被页面自身的 `EnableOverflowHide` 裁掉(不再重叠, 但也看不到)。要"看得到"得给页面加滚动容器 —— **已解**: 页面换成 `ui.ScrollArea` 滚动容器, 见第八节 66。

60. **"自适应列宽"依赖元素宽度, 而宽度是布局算出来的 —— 必须订阅元素尺寸事件自愈**(`ui/table.go`):
    - **现象(实测)**: 刚进入数据页时表格**最后一列(自适应列)被挤成 20px**, 表头文字裁成"日"、单元格文字全是"..."(`...` 是形状文本放不下时的截断标记); 拉一下窗口尺寸(或换页回来)就正常。
    - **根因**: `NewTable(x,y,0,...)` 的 `cx=0`(尺寸交给布局), 于是建表时 `GetWidth==0`, `colWidth` 算出 `0 - 表头内缩*2 - 固定列之和` → 负数 → 兜底 20。**不能依赖调用方在窗口尺寸事件/换页时手动 `FitColumns`**: 首次显示这条路没人调, `w.Show` 之前元素尺寸全是 0, 尺寸事件又不一定发。
    - **修法**: 组件自己订阅 `List.AddEvent_Size`(元素尺寸变化事件) → 回调里 `relayout(true)`; `relayout` 用 `lastW` 记录"上次算列宽时的元素宽度", 宽度与滚动条需求都没变就直接返回, 因此不会因 `SetColumnWidth` 引发的重复尺寸事件而震荡。组件自愈后, `FitColumns` 只用于特殊时序下的强制重算。
    - **正确表现**: 首屏自适应列宽 = 剩余宽度(表头与数据完整), 拉窗/换页/切主题三种路径一致。

61. **自绘组件的"宿主底色 + 组级半透明"合成规则**(checkbox/radio/switch/input):
    - **现象(实测)**: 暗色下 ①未勾选的复选框是**一块比卡片更黑**的方块(亮色下看不出来); ②复选框**禁用态**的框线比参考实现亮一档; ③开关**禁用态**的滑块还是一颗扎眼的全亮白圆; ④禁用输入框是一整块**发亮的灰饼**。
    - **根因**: ①未选态在 shadcn 是"亮色无底色 / 暗色 `dark:bg-input/30`", 若铺 `Background`(#0a0a0a), 比卡片(#171717)还深; ②③④ 禁用态若用 `WithAlpha(128)` **覆盖**透明度 —— 暗色 `input`/`border` 令牌本身就是"白 15% 半透明", 覆盖成 50% 等于画了一条亮白线/一坨亮白饼; 滑块/勾号也不能漏掉淡出。
    - **修法(三条一起立规矩)**:
      1. **先铺宿主底色**: 组件按"放在 `Card` 上"设计, `onPaint` 第一步 `p.FillRound(&p.Rc, t.Card, 0)` —— 既擦掉上一帧(透明元素重绘不擦底, 见 28:焦点环消失后的灰环、上一个状态的实心形状都会留在画布上), 又给半透明令牌一个正确的底。
      2. **未选/未勾的底**: 亮色不铺(露卡片), 暗色才叠 `t.Input.ScaleAlpha(0.30)`(= `dark:bg-input/30`);开关轨道暗色未选是 `t.Input.ScaleAlpha(0.8)`(= `dark:data-unchecked:bg-input/80`)。
      3. **禁用 = 每个图层各乘 50%**(`ScaleAlpha(0.5)`, 不是 `WithAlpha`): 不透明色乘完再叠到宿主上就等价于"整组半透明合成", 半透明令牌乘完也不会被拉亮。**形状**全部跟随(框/勾/轨道/滑块/圆点);**文字**也要跟随 —— 自绘按钮的文字/图标与输入框禁用文字一样, 用"前景色 × 卡片色对半混"预混后绘制(直接加 alpha 会二次透出壳体底色);只有纯原生绘制的复选框/单选框标签保持原色(见第四节末汇总表)。
    - **实测(暗色, 与 shadcn 逐像素核对; 基底 `card=#171717`)**: 未选框底 `#212121`(白 4.5% 叠卡)/框线 `#424242`(白 15% 叠在框底上);禁用未选底 `#1c1c1c`/线 `#2d2d2d`(各再乘 50%);禁用已勾框 `#7e7e7e`(白 90% 实色乘 50%)+ 勾 `#4a4a4a`;开关禁用滑块 `#8f8f8f`(白 98% 乘 50% 叠轨道)。全部与"整组 50% 合成"的理论值一致。
    - **测试注意**: 程序化改状态**不会自动重绘**(原生 `SetCheck` 不失效重绘, 所以要 `Checkbox.SetChecked` 显式 `Redraw`); 验证状态切换时, 把"切换后"与"直接建成目标状态"的画面逐像素比对, 差异为 0 才算无残留。

62. **hover 过渡的"最后一帧"不再触发重绘 —— 二值可见物(下划线)淡出会永久残留**(`ui/separator.go` Badge Link):
    - **现象(实测)**: Link 徽章鼠标悬浮后移开, 下划线留在画布上不消失; Button Link 无此问题。
    - **根因(两条叠加)**: ①徽章 link/ghost 变体静止态 `bg=None`,`FillRound` 跳过 —— 透明元素重绘不擦底(见 28), 上一帧像素留在画布; ②过渡内核 `hoverTick` 里当某一帧的值**恰好到达目标**时该帧不重绘(`v == target` 时 `active=false`), 颜色类过渡无所谓(倒数第二帧 v≈0.02 与静止色视觉相同), 但"下划线"是 `hv>0` 就全色画的**二值可见物** —— 倒数第二帧(v≈0.02)的全色下划线被画上去后再也没人擦。
    - **修法**: ①`onPaint` 开头铺宿主底色 `t.Card`(同 61);②下划线透明度随过渡值缩放 `fore.ScaleAlpha(hv)` —— 倒数第二帧 alpha≈2, 肉眼不可见; 顺带淡入淡出也更平滑(shadcn 的 underline 本身不参与 transition, 但 GUI 侧缩放更接近"整体过渡"的观感)。**推论: 凡以过渡值为开关条件的可见物, 透明度一律跟 `hv` 走, 不要只判 `hv>0`。**
63. **开着对话框最小化再还原 = 整窗元素绘制/命中全灭, 只能"最小化前挂起、还原后重显"规避**(`ui/dialog.go`):
    - **现象(实测)**: 弹窗打开后点任务栏最小化, 再还原 —— 弹窗消失(只剩 backdrop 的模糊快照), 点遮罩/页面全无反应。
    - **诊断**: 还原后元素树状态**全部正常**(`IsShow=true`、alpha=255、矩形正确、z 序 overlay>backdrop>页面), 绘制回调**照常触发**且 `p.Rc` 正确 —— 但绘制输出不上屏、鼠标事件不到达任何元素(含页面原生控件); 无弹窗可见时还原完全正常。结论是 xcgui 异型阴影窗口(`Window_Transparent_Shadow`)还原后合成/命中管线对"对话框遮罩子树可见期间"的状态损坏, **引擎级, 组件层无法事后根治**。
    - **无效方案(勿试)**: 还原时重抬 z 序 + backdrop 压回 overlay 之下、`XWnd_Redraw(true)` 整窗重绘、重新显隐整棵子树、延迟到消息循环里整套重物化(重拍快照 + 重新 Show) —— 全部无效; 在 WM_ACTIVATE 里直接操作元素更会把整窗合成打断(全窗白屏)。
    - **修法 = 挂起/恢复**(对话框状态保留, 不触发 onClose): `AddEvent_WindProc` 拦 `WM_SIZE` —— `wParam==SIZE_MINIMIZED(1)` 时**同步**隐藏整组(清焦点 + `showEleTree(false)` + backdrop 隐藏; 不能走 Hide, 其淡出完成回调是异步的)置 suspended; `wParam==SIZE_RESTORED(0)/SIZE_MAXIMIZED(2)` 且 suspended 时用 `app.AfterFunc(0, ...)`(= `time.AfterFunc` + `XC_CallUT`, 回调被投递到 UI 线程执行)做 `materialize` + `SetSubtreeAlphaSkipScrollBars(1)` 无动画直接显示(滚动条不跟着拉亮, 见 98)。引擎全程只见"隐藏状态的还原", 与已验证可靠的路径一致。
    - **三个时序/机制坑**: ①`XE_SIZE` 在还原时不保证触发, 程序化最小化**不发 WM_ACTIVATE(WA_INACTIVE)** —— 抓"最小化"只能拦 `WM_SIZE/SIZE_MINIMIZED`; ②WM_ACTIVATE/WM_SIZE 里直接操作元素会把还原流程的合成打断(实测全窗白屏), 恢复动作必须延迟出消息处理(用 `XC_CallUT` 类机制投递到消息循环); ③**炫彩定时器是独立命名空间**: `AddEvent_XC_Timer` 只接收 `XWnd_SetXCTimer` 启动的定时器, 用 Win32 的 `XWnd_SetTimer` 启动时回调**永远不会被派发**(与 anim.go 的 animTimerID 机制同源, 自用 ID 避开 0x7875); 恢复动作用 AfterFunc 可绕开这套机制; ④**还原后立即启动的 Presence 过渡帧不会被驱动**(整组 alpha 卡 0, SetSubtreeAlpha(0) 之后全组隐身) —— 恢复显示必须无动画, 之后用户再打开仍走正常淡入。

65. **COM 直通的参数传递(386 / amd64) + "浮点形参"的完整解法**(`wapi/com` 与 `d2d`):

    - **386**(stdcall, 参数全部走栈): 结构体按值传要**拆成 32 位槽**逐个压栈; float64 用 `common.Float64ToUint32Pair` 拆 low/high 两个槽 —— **符合 ABI** 的写法(edge 的 `edge_386.go` 正是如此)。
    - **amd64**(Microsoft x64): 8 字节以上的结构体**按引用传**(要传指针); 浮点走 XMM0~XMM3, 与通用寄存器是**两套独立**的 —— **SyscallN 直接传(哪怕位打包)无效**: `ID2D1RenderTarget::SetDpi(float,float)` 位打包**直接崩**(`Exception 0xc0000005`, 现场 `rax=r9=0x42C00000` 正是 96.0f 的位模式)。
    - **解法 = `wapi/com` 的汇编转发器**: `CallWithFloat32` / `CallWithFloat64` / `CallWithFloat32RetFloat`(读"按值返回 float"的方法)。三个坑都实测定死:
      ① **必须由 `syscall.SyscallN` 调用转发器** —— 借它的 `runtime.cgocall` 切到**系统栈**: Windows 的栈探测看 TEB 里的栈范围, 而 Go 的 goroutine 栈不在其中。直接在 goroutine 栈上 CALL: `GetTickCount` 能过, `user32.GetSystemMetrics` 崩(栈探测落到非法地址)。
      ② **转发器要自己摆正栈**: 调用点 RSP 16 字节对齐 + 32 字节 shadow space。实测这条路径入口 RSP ≡ 0 (mod 16) —— `SUBQ $32` 过、`SUBQ $40` 崩(用到 SSE 对齐存取的 API 立刻暴露)。
      ③ **浮点槽号要按接口实测**: 教科书说"整数/浮点各自独立计数"(`SetOpacity(FLOAT)` 的浮点应在 XMM0、this 走 RCX), 实测 `ID2D1SolidColorBrush::SetOpacity` 读的是 **XMM1** —— 逐个槽单独填 0.5, **只有 XMM1 生效**; 这类接口按"参数位置"编号(this 占位置 0)。
      ④ **第 5 个及以后的参数要压栈**(x64 约定: 超出 RCX/RDX/R8/R9 的放 `[RSP+32]` 起, 前 32 字节是 shadow space), **栈上的 float 占 8 字节槽、值放低 4 字节**: `wapi/com.CallWithFloat32Stack` 覆盖"4 个整数/指针 + 4 个浮点 + 最多 4 个栈参数"这个通用形态, 单测拿 `ReadProcessMemory` 的第 5 个参数(实际读到的字节数)验证它真的落在了栈上。实测用例: `DrawGeometry(geometry, brush, FLOAT w, style)` → 线宽 XMM3 / style 栈[0]; `DrawLine(p0, p1, brush, FLOAT w, style)`(5 个参数) → **线宽在栈[0]**(位置 4, 前 4 个 XMM 槽已被位置 0~3 占掉)/ style 栈[1]。若按"独立计数"喂(XMM0 + R9), 调用本身不崩(void 方法)但**整帧渲染被搞坏**(窗口全黑)。这条打通后, 图标描边从"CPU 展开 + 填充"换成了真描边(见 §43 末尾), 基本图元(直线/矩形/椭圆)与形状几何(矩形/圆角矩形/椭圆)也一并封装(探针一次抓图全对: 线宽/圆帽/位置/圆角都对)。
    - **验证数据**: 回声桩(原样返回 XMM0)0.5/0.125 精确往返 ✓; 画刷 `SetColor/GetColor` 往返 0.25 ✓(对象正确); `SetOpacity`: 只填 XMM1=0.5 → **读回 0.500** ✓(填 XMM0/2/3 均无效); 反例 —— 该 RT 的 DPI 是 0/0(非法状态), 在它上面 `SetDpi` 怎么传都崩, **与传参无关**(验证浮点要用纯属性 setter, 别拿有状态的 RT 方法试)。
    - **两处注意**: ①COM 直通不止 x64 —— 386 同样能跑, 只是参数传递写法不同(edge 两个架构各一套); ②`ID2D1Brush::GetOpacity` 是**按值返回 FLOAT**(在 XMM0), 不是 `(float *out)` 出参 —— 要读它得用 `CallWithFloat32RetFloat`。
    - **别把"按值 float"与"浮点结构指针"搞混**: 受限的只有**按值传的** float64/float32 形参; 指向浮点结构的**指针**不受限, 照常 syscall 调用即可 —— `ID2D1RenderTarget::SetTransform` 的矩阵、`CreateSolidColorBrush` 的颜色都是这种, `d2d` 已封装 `SetTransform`/`GetTransform`(探针往返一致: 单位矩阵 → 平移 12,34 → 设回单位矩阵)。
    - **`SetTransform` 与炫彩绘制的相互作用(实测, 用它的前提)**: 炫彩的 `XDraw_*` 走的是**绘制当时的 RT 变换** —— 设了平移不恢复的话, 紧随其后的 XDraw 内容会一起被平移(探针 : 变换下画的绿色矩形右移了 150 逻辑像素, 恢复单位矩阵后画的蓝色矩形位置正确)。所以它可以**用来做"直通绘制的局部平移"**(图标位置外置即靠它, 见 43), 但**必须成对设置/恢复**。
    - 与 43(直通拿不到元素变换/裁剪)、47(整窗 alpha 与生命周期)同源: 都是 **SyscallN 这条通道的能力边界**, 而不是 xcgui 的缺陷。

66. **页面滚动容器 = 原生 `LayoutFrame`(布局框架)的实测语义**(`ui/scrollarea.go`):
   - **布局框架 = 布局盒子 + 滚动视图**: 和其他容器一样流式排子项, 但**内容总大小按子项自动撑开**、溢出时自动出现纵向滚动条 —— 不需要 ScrollView 那套手动 `SetTotalSize`(示例 `Basic/ScrollView` 的用法), 页面滚动容器直接用它。
   - **子项尺寸语义**(内容轴是"按内容自撑", 与 55 的通用语义不同):
     - `Fixed` / `Auto` 拿到自然值(6×100 + Auto 高容器 866 对 视口 532 → 出滚动条, 内容不被压扁);
     - **`Fill` = 视口高**, 且它**内部**子项的溢出**不算进滚动范围**(8×100 的 Fixed 卡片挂在一个 Fill 高子项里, 662 高的框架 total 仍是 662 —— 滚不动), 只适合"铺满视口 + 内层 Weight 吃剩余"的形态(Fill 子项自己的高度是确定值, 内层 `[Fixed100][Weight1][Fixed100]` 实测拿到 442 ✓);
     - **`Weight` 直接挂在框架上恒为 0**(内容轴不确定); 内层再写 Weight 也是 0;
     - **`EnableOverflowHide` 不是"裁剪显示"、是"布局时把超出视口的子项高度裁掉"**: 400 高的子项被压成 10(实测 400→10)。滚动页面要的是前者, 而**视口裁剪 ScrollView 自带**(滚出视口的子项本来就不画出来), 所以这个开关**不要开** —— 开了卡片会被切短。
   - **首帧(以及滚动条显隐翻转的那一帧)内容宽度会少一个滚动条宽**: 视口 772 时 Fill 子项只有 760(右侧多 12px 空白), 原因疑似该趟排版先按"无滚动条"排子项、随后视口才被滚动条占去; **再排一次就正确**(`AdjustLayout`)。`ui.ScrollArea` 用元素尺寸事件 + (frame宽, 视口宽) 闸门自愈一次, 调用方不必管(窗口 resize 导致滚动条显隐翻转时会再自愈一次)。
   - **滚轮**: 悬停在**子元素上**也会冒泡到框架并滚动(子元素的 `XE_MOUSEWHEEL` 默认禁用, "禁用则事件传给父元素"); 悬停原生控件(如 List)会被它自己吃掉 —— 嵌套滚动的正常行为。注入真实滚轮时**窗口必须在后台有焦点**(工具进程还要 per-monitor DPI-aware, 否则坐标虚拟化)。
   - **隐藏页恢复可见后, 子控件命中区域还停在旧滚动位置**: 视觉上布局是对的, 但点日期框/时间框/输入框没反应, 重排一次页面才正常(实测)。ScrollView 的命中映射在页面隐藏期间不刷新, 显示后再延迟约 32ms(等页面真正上屏)对页面调一次 `AdjustLayout` + `Redraw`, 才能可靠重算整棵内容树的命中区。嵌套的水平容器同样有滞后风险 —— 卡片里的日期/时间框直接挂到内容区, 不要再套一层 `HStack`。
   - **滚动条滑块拖动: 合成输入下未复现**(探针实测: 裸 LayoutFrame 与 `ui.ScrollArea` 的滑块都拖不动, 而同一脚本拖**表格 List 的**(同款皮肤)能滚 —— 不是组件引入的差异, 疑与 ScrollView 自带滚动条的命中/拖拽实现有关, 未深究); 滚轮、代码滚动、点选/输入等常用路径都正常。真实鼠标下是否可拖未验证。
   - **代码滚动**: `ScrollPosYV(像素)` 是像素语义(滚到 100 就是 100); **`ScrollPosV(pos)` 的参数是"行"单位**(默认 20px/行, `GetScrollBarVObj.GetRange` 也按行算) —— 拿像素喂它会直接滚到底(实测 100 → range 17 全用掉)。
   - **滚动条几何/配色/显隐**(统一规格见 `ui/scrollbar.go`): 元素占宽 = `SetScrollBarSize` 的值(10 就是 10), `SetSliderPadding(1)` 把**滑块**在轨道里缩成 8px 宽(`w-2.5` + `p-px` 的 shadcn 规格); 轨道要真透明得 `ClearBkInfo` 后垫一层 alpha=0 的填充(否则回退系统默认内容, 同 30); 圆角 / 两级配色 / 悬停显隐的完整机制见第 21 条。Table 的自适应列按 14 让宽(`tableSBW`), 多行 Input 在 `applyTheme` 里刷这条皮肤。⚠️ **"自动显隐"模式下滚动条的 `XEle_GetRect` 返回的是隐藏位置**(实测: 表格滚动条 raw x = -12..-2, 而渲染/命中在内容区右缘)—— 别拿它做命中或定位计算; 判定鼠标是否在滚动条上要用 `XWnd_GetStayEle`(渲染与命中的坐标系一致)。
   - **容器被点一下会聚焦并画原生蓝框**(#58B1FC, 同 34) → `EnableFocus(false)` + `EnableDrawBorder(false)`。
   - **内容区的外边距计入滚动范围**: 内容区四边各 24 外边距时 total = 24 + 内容高 + 24 —— 页面底部留白因此天然有效(滚到底正好留 24, 不必额外垫占位元素)。
   - **Auto 内容里的栅格写法**(实测): 行高必须 **Auto**(行高写 Weight/Fill 在 Auto 内容里塌成 0 —— 两列卡片会整段消失), 行里 **`Weight` 宽的列仍然算得出宽度**(352 宽的行 → 201/135 = 3:2 ✓), 列高 Auto、卡片 Auto —— 就是 CSS flex 的直觉写法。**铺满型页面(数据页)反过来**: 内容区 `SetContentFill`(Fill) + 行 `Weight` + 列/卡片 Weight 整链 Weight, 表格/树才会跟随窗口长高, 代价是页面不再滚动。
   - 与 55/56/59(尺寸语义、Card、Auto 高的坑)同源; 该条同时解释了"页面内容比窗口高被 `EnableOverflowHide` 裁掉"那个问题的解法。

67. **自绘标题栏的"最大化/还原"按钮:用 `Button_Type_Max`, 别自己调 `MaxWindow`**:

    - `SetTypeEx(xcc.Button_Type_Max)` 的按钮**会自己切换**最大化/还原(实测: 默认尺寸点一次 → `IsZoomed=true`; 最大化状态再点同位置 → `IsZoomed=false`, 窗口回到原矩形), 与 39 里 `MaxWindow` API\"只管最大化、不切换\"不同 —— 双击/按钮两条路径各用各的, 不要互相替代。
    - 按钮顺序仍按视觉排: 最小化 → 最大化 → 关闭, 尺寸与 Ghost 样式同其他控制按钮(`ButtonSizeIconSm` 那套)。
    - **同一排图标要按"墨迹占比"配平, 而不是统一画布**: `IconEye`(2..22)、`IconMoon`/`IconMaximize2`/`IconMinimize2`(3..21)在 18px 画布下墨迹 23~25 物理像素; `IconMinus`(5..19)、`IconX`(6..18)同画布只有 17/15, 看起来"最小化/关闭小了一圈"。**保持最大化/主题图标默认 18, 把 minus/× 的画布放大到 22**(`WithIconSize(22)`): 实测 eye 25×17、moon 23×23、maximize2 23×23、minus 21×3、× 19×19, 一排视觉齐平。24 也齐(24×2/20×20)但描边同比偏粗, 取 22。反向"缩小 maximize 到 14"同样齐(17/17/15), 但最大化会比旁边的主题按钮小, 不取。
    - **自绘 `WithIcon` 的按钮不会随窗口状态换图标**(组件每帧只画 `b.icon`), 最大化后仍是原图标, 需要自己换: 窗口没有 MAXIMIZE/RESTORE 事件, 拦 `AddEvent_WindProc` 的 `WM_SIZE`(`wParam==SIZE_MAXIMIZED/SIZE_RESTORED`), 用 `a.UI` 推迟到消息处理完之后按 `w.IsMaxWindow` 调 `SetIcon(icons.IconMinimize2/icons.IconMaximize2)` —— 这条路径同时覆盖按钮、双击标题栏、Win+↑ 和系统菜单; `iconSize` 不随图标变, 两个图的墨迹比例相近, 不用再调。
    - 最大化后标题栏按钮整体贴屏幕右上角(窗口矩形 = 屏幕), 元素坐标随布局重排 —— 用固定坐标做 UI 自动化时, 两种情况要分别取按钮中心。

68. **原生 ComboBox 默认带中文字间距, 弹层 ShapeText 没有 → 打开下拉时文字会"变紧"**(`ui/combobox.go`):

    - 实测(150% DPI, 14px 主题字体, 文本"测试环境"): 原生 ComboBox 墨迹宽 **91 物理像素**; 同一字体句柄下 ShapeText 只有 **87**。逐字对比单字墨迹(测 20/20、国 19/18)证明**不是字号/字形缩放, 而是每个字多出约 1px 的 advance** —— xcgui 给原生 ComboBox 的中文默认加了字间距。
    - 触发区用 ShapeText 弹层、原生控件又带字间距, 结果就是"点开下拉瞬间文字突然变紧"。调条目左边距无法解决, 只会让左缘对齐而宽度仍不一致。
    - **`xc.XEdit_SetCharSpaceSize(comboHandle, 0, 0)` 对 ComboBox 句柄同样有效**(`EnableEdit(false)` 也生效): 触发区 91→**87**, 与条目/ShapeText 完全一致; 传 `-1,-1` 则收紧到 82(负间距)。
    - 修法写在 `NewComboBox` 里(设完字体之后调用一次); 修后同一弹层内"触发区文字"与每个条目的左缘、墨迹宽度三个数全部相同(1336..1422)。

69. **ComboBox 弹层: 原生弹层能转异型透明窗, 但整窗 alpha 淡入不可靠 —— 已取消进场淡入**(`ui/combobox.go`, 两次修正的结论):
    - **时机**: `XE_COMBOBOX_POPUP_LIST` 触发时弹层窗口**尚未显示**(探针实测 `IsWindowVisible=false`, 窗口矩形/客户区已就绪), 透明类型能在首帧前设好 —— 不会闪白。
    - **普通窗口上 alpha 无效**: 不换类型直接 `XWnd_SetTransparentAlpha(hWindow, 128)`, 抓图仍全白。必须先 `XWnd_SetTransparentType(hWindow, Window_Transparent_Shaped)` —— 它在弹出后(未显示时)切换也生效。
    - **但连续改 alpha 会概率性冻结**(推翻本条早期结论"每帧都生效"): 早期探针是**孤立地**隔帧设 64/160/255 三个值, 每次都变了, 于是判定"这个弹层不同于 tooltip(57), 整窗 alpha 动画可用", 并按 100ms 六帧做了淡入。真实交互下的复测(40 轮"点开→立刻选不同项", 每轮用 `GetLayeredWindowAttributes` 读实际 layered alpha): **3 轮弹层停在 alpha = 3/255**, 而同一时刻的日志显示 apply 已经跑到 v=1.0(即 255 的写入被丢弃)。冻结发生在第 2 帧前后 ⇒ 弹层**存在、可交互、但完全看不见**, 又盖住触发区 ⇒ 用户报的"快速选择几次后点触发器没反应"。
    - **结论/修法**: 异型弹层上**不要用整窗 alpha 做逐帧动画**(与 57 同一条规律, 单次设置才可信); ComboBox 弹层因此**不做进场淡入**, 弹出回调里只做上色 + 定位(`styleComboPopup` / `alignComboPopup`)。要保留淡入观感, 只能走**元素 alpha**(`SetElementsAlpha`), 但该弹层的白底是窗口自绘、不是某个元素, 只淡 ListBox 会露出全不透明的面板底 —— 不做。
    - **转类型不影响区域裁剪的圆角**: `SetWindowRound` 先设、再转 Shaped, 圆角外仍是背景色; ListBox 的 BkM 底色、条目 DrawItem、`XE_PAINT_END` 的 ring 全部照常输出。
    - **实现要点(若将来再碰这个弹层的窗口动画)**: 过渡挂**宿主窗口**(`XWidget_GetHWINDOW`)而不是弹层小窗(见 47); `apply` 每帧先 `XC_IsHXCGUI(hw, xcc.XC_WINDOW)` 守卫 —— 弹层会在点外部/选中条目时立刻销毁, 用失效句柄调 X API 会静默崩进程(与 64 的菜单淡入同一个坑); 每次弹出用 **`xui.NextPresenceKey`** 取键(见 70)。
    - **排查要点**: 判"弹层在不在"用窗口枚举(空标题 + 尺寸的顶层窗), 判"看不看得见"用 `GetLayeredWindowAttributes` 读 layered alpha —— 两者合起来才能区分"没弹出"与"弹出了但透明"。

70. **Presence 的键必须全局唯一: 撞键 = 新弹层"全透明但仍在", 表现为点击没反应**(`presence.go` + `menu.go` / `combobox.go` / `dialog.go`; 实测复现并修复):
    - **现象**: 表单页选过一次 sm 下拉框 → 切到导航页点"打开菜单" → **第一次点击毫无反应**, 第二次才弹出; 反过来先开菜单、再点 sm 下拉框, 下拉框也"点了没反应"。点同页其它控件(分页)同样失效一次。
    - **不是点击被吞**: 给按钮的 `AddEvent_BnClick` 加日志, 日志**照常触发** —— 菜单确实弹了, 只是窗口 alpha 停在 `AlphaFloor`(看不见), 而原生菜单一弹出就是模态的, 于是下一次点击被这个"透明的菜单"吃掉(选中/点外部才关闭)。
    - **根因(键撞车)**: `Presence` 按 `(hOwner, key)` 记过渡值且**幂等** —— 目标值没变就不重启过渡、**也不再调 apply**; 而 `target=1` 的条目**不回收**(只有淡出到 0 才删除)。若 menu 与 combobox 各自维护 `menuPresenceKey++` / `comboPresenceKey++`, **都从 1 开始、都挂在宿主窗口上**: 先开下拉(key=1, 走到 1 后条目留在表里) → 再开菜单(key=1) → 撞上已完成的条目, 动画不启动、apply 不执行 → 菜单窗停在 popup 回调里刚设的 alpha 下限。反向同理。
    - **修法**: ① 新增 `xui.NextPresenceKey`(全局原子递增), menu / combobox 改用它, 对话框也从固定键 0 改成按实例分配(`Dialog.fadeKey`) —— **任何新调用方都不许自起计数器**; ② `Presence` 在返回前**立即用当前进度调一次 apply**: 键一旦撞上已完成的过渡, 最坏也只是"没有动画", 目标状态一定落地, 不会留下透明窗口(防呆)。
    - **排查要点**: 这类"点了没反应"先确认事件到没到(BN_CLICK 日志); 到了就怀疑"弹窗存在但不可见"。`SetAnimationScale` 放慢也看不到 —— 过渡根本没启动, 不是太快。
    - **同源**: 47(弹层透明度/整窗 alpha 的约束)、64(菜单淡入)、69(下拉弹层淡入)。

71. **异型/阴影弹窗里, 元素的布局坐标原点是窗口矩形原点(含阴影边距), 不是可见区(卡片)原点**(`ui/popup.go`):
    - `Window_Transparent_Shadow` 的窗口, `XWnd_GetClientRect` 返回的是含偏移的 Body(实测 menu: `{8,8,168,276}`, 即卡片在窗口坐标 (8,8) 起 —— 与 `XEle_GetWndClientRect` 的注释"原点 = 所属窗口矩形原点(含阴影边距)"同口径)。但**元素布局不吃这个收缩**: 元素放 (0,0) 会贴在窗口矩形左上(阴影带内), 卡片右侧与底部露出 band 宽的窗口底色 —— **浅色主题下完全看不出来**(窗口默认白底、卡片也是白), 深色主题下是一条白边(实测截图)。
    - 修法: 弹层内容元素建在 `(band, band)`(band = 阴影带宽), 尺寸 = 卡片尺寸; 窗口尺寸 = 卡片 + 2×band。定位同理: 窗口矩形左上 = 卡片目标位置 − band(物理像素要乘 DPI, 见 66 的坐标系说明)。
    - 与 36(菜单阴影窗)/47/64 同一套窗口, 菜单自己画背景用的是窗口绘制事件(rc = GetClientRect, 原点在窗口原点), 所以它看起来"没问题" —— 两条绘制路径的坐标系**差一个 band**, 混用时最容易在这翻车。

72. **手动创建原生 ListBox 做自绘列表(时间框弹层)的实测语义**(`ui/timepicker.go`):
    - **必须先 `CreateAdapter` 再 `AddItemText`**, 否则条目装不进去(同 ComboBox 的适配器语义)。
    - **几何**: `SetPadding` 生效(内容区四边内缩); **行步进 = 项高 + 1**(项与项之间固定 1px 缝, 与 combobox 弹层实测一致); **首项 top = padding + 3**(固有内缩, ScrollPosYV 定位要把它算上, 少 7px 目标项会露不全)。`SetItemHeightDefault(h, h)` 设默认行高。
    - **`ScrollPosYV` 是像素**(同 66); `SetSelectItem` 只改选中态**不滚动视口** —— 打开弹层时要手动滚到当前项。
    - **项模板的文字是"绘制前静态"的**: 在 `XE_LISTBOX_DRAWITEM` 里对模板 ShapeText `SetTextColor` 改选中项文字色**来不及**(文字已在 DrawItem 之前画完, 实测黑底上黑字)。要按状态变文字色只能**自绘文字**: 把模板文字设成全透明(`XShapeText_SetTextColor(h, 0)`), DrawItem 里 `p.Text(GetItemText(i, 0), &pItem.RcItem, col, Center|Vcenter)` 自己画。
    - **两套索引**: `GetItemText(i, col)` 的列从 **0**(适配器列)起; `GetTemplateObject(i, itemID)` 的参数是模板 XML 里的 **itemID**(默认模板 shapeText = 1)。混用会拿到空串/空句柄。
    - **选中态在 `pItem.NState`**(`List_Item_State_Leave/Stay/Select`), 悬停过渡用 `xui.HoverItem(hEle, pItem.Index, hot)`; 项背景画在 `pItem.RcItem`(已扣 padding, 不含滚动条宽度)。
    - `SetDrawItemBkFlags(0)` 关原生项背景 + `EnableBkTransparent(true)` 关容器背景; 滚动条走 `scrollbar.go` 的统一皮肤(`SetScrollBarSize` + `styleScrollBar`), `EnableAutoShowScrollBar(true)` 原生自动显隐。
    - `XWidget_LayoutItem_SetWidth(模板文字, Layout_Size_Fill)` 会把项模板文字宽算成 0(实测文字整体消失) —— 项模板里的 `layout.width="auto"` 别动, 居中交给自绘。
    - **元素鼠标事件的 pPt 是"元素客户区"坐标**(探针 : 元素在 (100,100) 时点元素内 (99,47) 的 raw = 99,47; `XEle_PointWndClientToEleClient` 是窗口客户区 → 元素客户区的**再次**换算, 对事件坐标再用会多减一次)。自绘组件的命中测试直接用 pPt 即可。

73. **复用的 Shadow 弹窗做淡入: 整窗 alpha 冻结 / 窗口白底 / apply 互搏 / 过渡挂错窗口 / 首帧残留 / DLL 无效句柄 —— 五个坑叠一起才是"闪烁、抖动、闪退"**(`ui/popup.go` + `anim.go` + `presence.go`, 实测):
    - **现象**: 弹层闪烁/抖动/闪退; 复用弹窗第二次显示时, 首帧是一帧白闪(帧级抓图 mean=228)。
    - **根因 C2(快速点击 UI 卡顿, "抖动")**: 过渡挂宿主主窗口 → hoverTick 每 tick 全量重绘整个表单页 ×100ms, 快速点击时重绘排队、UI 线程卡到出系统忙碌光标。**修法**: 过渡挂**弹窗自己**(p.win) —— 元素 alpha 由弹窗自身重绘落地, 400×500 物理像素每 tick 无压力; 整窗 alpha 时代挂宿主窗是被迫的(DWM 合成要宿主带动, 见 47/73 根因 B), 元素 alpha 不需要。
    - **根因 A(主因, 白闪)**: 弹窗窗口不设置背景时, xcgui 画**白色默认底**。淡入(元素 alpha)期间卡片半透明, 白底从卡片下面透出来 —— 深色主题下就是刺眼的白闪。**修法**: 弹窗创建时给窗口 BkM 垫一层 `alpha=0` 的填充(`XBkM_Clear + XBkM_AddFill(..., 0, 0)`) —— 留空会回退"系统默认内容", 透明必须显式表达, 同第八节 30 的轨道透明。
    - **根因 B(淡入路线错误)**: 整窗 alpha(`XWnd_SetTransparentAlpha`)对透明窗**只认第一次调用**(57) —— 创建时的 255 就是那次, 之后全部冻结。menu/combobox 的弹窗淡入能用是因为**每次弹出都重建窗口**(每次都是"第一次调用"), 而组件弹层是复用的 → 整窗 alpha 路线整段失效(逐帧抓图证实两次显示都是瞬间全亮)。**修法**: 淡入改**元素 alpha**(`SetSubtreeAlpha(body)` + 每帧 `XWnd_Redraw(弹窗)`, tooltip 同款, 见 57); 显示前压 `SetSubtreeAlpha(0)`, 首帧只有阴影。附带修正: `SetWindowPos` 不带 `SWP_SHOWWINDOW`(提前显示会多一帧)。
    - **根因 C(第二次起 apply 互搏)**: `hoverTick` 对 owner 上所有 presence 条目调 apply, 已到位旧条目(不回收)每帧 `apply(v=1.0)` 与新条目打架 —— 实测二次弹出时亮度 12↔85 逐帧交替。**修法**(anim.go): 到位条目(settled)不再调 apply("到位后条目静默"补上实现; 反向过渡 settled 会重置, 淡出不受影响)。menu/combobox/dialog 共用这张表, 同样受益。
    - **排查方法**: ①亮度采样在"浅色弹层弹在浅色页面/弹层内容与页面内容相同"时完全失效, 必须**帧级抓图看画面**; ②外部工具(BitBlt 轮询)有 10ms+ 延迟且可能错过首帧, 首帧问题要用**逐帧抓图**或进程内自抓(Show 里起协程); ③录屏分辨率是缩放的, 且 IDE 全屏会挡住采样区域 —— 坐标要先按实际内容校准; ④apply 日志显示 v 正常推进 ≠ 画面正常(白帧期间 v 也在走), 要把"值序列"与"帧序列"分开对齐; ⑤**DLL 错误弹窗 ≠ Go panic** —— 出现 DLL 报错弹窗(如"SetAlpha 输入句柄无效")时, 先定位是哪个 X API 收了无效句柄, 再查该句柄的生命周期(所属窗口显示前/销毁后); 被自动关闭的错误弹窗背后可能跟着 access violation 闪退。
    - **根因 D(时间框点击闪退 + "SetAlpha 句柄无效"弹窗)**: `GetChildByIndex` 对原生 ListBox 会返回**非 0 但无效**的子句柄(实测 type=62 的延迟创建内部对象, 弹窗已显示后依然如此, 每个子 ListBox 约 8-16 个) —— 对它调 `XEle_SetAlpha` 触发 xcgui 的 **DLL 错误弹窗("SetAlpha 输入句柄无效")**。⚠️ DLL 内部错误**不产生 Go panic** —— 它走 xcgui 自己的错误处理(弹窗/日志/返回); 错误弹窗被自动关掉后, 后续无效句柄调用可能直接 access violation 闪退(无堆栈)。**修法三层**: ①遍历组件(`popup.go` 的 `fadeSubtreeAlpha` 与 `presence.go` 的 `setAlphaTree`)对每个句柄加 `XC_IsHXCGUI(h, XC_ELE)` 守卫; ②`fadeSubtreeAlpha` 同时跳过滚动条子树(滚动条显隐由 showScrollBarOnHover 自理, 跟随淡入会被每帧拉亮); ③Show 时序改**屏幕外预显示**(见根因 E)。
    - **根因 E(首帧残留)**: 复用窗口再次 ShowWindow 时, 首帧合成的是**窗口表面残留的上一次内容**(关闭前 alpha=1 的全亮卡片); 元素 alpha=0 要等第一次重绘才落地, 于是先闪一帧全亮。**修法**: **屏幕外预显示** —— 先 `SetWindowPos(-30000,-30000, SWP_SHOWWINDOW)` + `Redraw(true)` 在不可见位置画好"透明首帧", 再 `SetWindowPos` 挪到目标位置(挪动只改 DWM 合成位置, 表面内容已是透明帧; Win32 窗口坐标范围 ±32768, -30000 安全)。
    - **交互补充**: 关闭后 300ms 内重开(`popoverReopenFadeSkip`)**跳过淡入直接完成态** —— 快速连点时弹层反复从透明渐显会"眨眼", 对齐 CSS 过渡的中断语义(不重播动画)。
    - **同源**: 57(tooltip: 异型窗整窗 alpha 只认第一次 → 元素 alpha)、64/69(menu/combobox 每次重建弹窗所以整窗 alpha 可用)、70(presence 键唯一 + 根因 C 的条目静默)、30(BkM 留空回退白底)、47(整窗 alpha 时代挂宿主窗的原因)。

74. **弹层夹边界必须用"显示器工作区", 不是屏幕高度**(`ui/popup.go`; 现象: "窗口最大化 + 页面滚到末尾, 点日期框后月历卡片底部被任务栏盖住"):
    - **现象**: 触发区贴着任务栏时, 弹层按"屏幕高度"判断下方放得下(y+ch <= SM_CYSCREEN), 于是照常向下弹, 卡片下半截被任务栏盖住(任务栏永远在顶层, 弹层是 topmost 也没用)。
    - **修法**: `wapi.MonitorFromWindow(hWnd, MONITOR_DEFAULTTONEAREST)` + `wapi.GetMonitorInfo` 取 `RcWork`(已扣任务栏/停靠栏), 用逻辑坐标夹边界: 下方放不下 → 翻上方; 上方也放不下 → 贴 `RcWork.Bottom`; 水平夹在 `RcWork.Left/Right`。
    - **坑**: ①`rcWork` 是**物理像素**, 而 xcgui 的窗口/元素矩形是逻辑像素 —— 除以 DPI 比例再夹(本库 `XWnd_GetDPI`); ②`SM_CYSCREEN` 只是主屏高度, 副屏/多屏应取"窗口所在显示器"的工作区(本条实现), 同 `SM_CXVIRTUALSCREEN` 与 `SM_CYSCREEN` 混用也是错的; ③`MONITORINFO` 的 `cbSize` 由 `wapi.GetMonitorInfo` 内部填好, 调用方不用管。
    - **坑**: `MonitorFromWindow` 要的是 **Win32 HWND**, 而 `xc.XWidget_GetHWINDOW(hAnchor)` 返回的是 **xcgui 窗口句柄** —— 喂进去 `MonitorFromWindow` 返回 0, 代码会静默走"整屏高度"兜底分支, 任务栏照样盖住弹层。**修法**: `xc.XWnd_GetHWND(hWnd)` 转换后再查显示器; `monitorWorkArea` 里再兜一层"句柄无效 → 用 `GetDesktopWindow` 查", 保证拿到的都是工作区而不是整屏。
    - **Win32 调用一律走 `wapi`**: `GetWindowDC / ReleaseDC / PrintWindow / MonitorFromWindow / GetMonitorInfo` 与 `PW_` / `MONITOR_` / `MONITORINFO` 在 `wapi` / `wapi/types`, gdi 侧用 `gdi.*`; 不要写裸 `syscall.NewLazyDLL`。

75. **通知(Toast)用窗口内单元素自绘, 悬停暂停走窗口级 GetStayEle, 位置动画必须整窗重绘**(`ui/toast.go`):
    - **载体**: 通知要贴着主窗口右下角、跟随窗口移动/缩放/最小化 —— 用**窗口内的覆盖元素**(每条通知一个 `widget.NewElement`)而不是独立置顶小窗, 这些跟随都由框架负责。窗口开着布局(`EnableLayout(true)`)时, 未声明 `LayoutItem_*` 的子元素**不会被 AdjustLayout 重排**, 手动 `XEle_SetRect(…, AdjustLayout_No)` 的位置会保留(与 Dialog 的全窗遮罩同一条路径, 实测)。
    - **一条通知 = 一个元素整体自绘**(图标/标题/描述/动作按钮/关闭按钮全画在它里面, 不建子元素): 悬停不会被内部控件打断, 命中判定只是"卡片局部坐标 + 一次矩形比较"; 若用原生 Button 做关闭钮, 鼠标移到子元素上会让父元素收到 MouseLeave(见 21), 悬停暂停会莫名其妙地恢复。
    - **悬停暂停必须用窗口级 `MouseMove` + `XWnd_GetStayEle`**: 元素级 MouseStay/MouseLeave 不可靠(21), 而且这里要的是"鼠标在这组通知的任意一条上"的**整组语义**；单条元素判定在两条卡片之间移动时会反复 leave/enter。窗口尺寸事件同理走 `AddEvent_Size` 重排。
    - **倒计时**: 每条通知在自己的元素上挂炫彩定时器(ID 用 `0x7876`, 避开 anim 的 `0x7875`、Tree 的 `0x7874`), 100ms 一帧; 悬停时只是不递减(`paused`)。为什么不用窗口级定时器: 同一窗口可能有多个 Toaster, `XWnd_SetXCTimer` 同一 ID 只能有一个定时器, 会互相覆盖。
    - **定位**: 用 `XWnd_GetBodyRect` 取内容区右下角 —— 阴影窗口(`Window_Transparent_Shadow`)的 Body 比整窗小一圈 band(71), 而**元素坐标原点是窗口原点(含阴影带)**, 不要再叠加. 元素比卡片四周外扩 6px(投影留白), 因为元素绘制被裁剪在自身客户区内。
    - **位置六种**: `layout()` 按 left/center/right × top/bottom 组合; 居中时 `baseX = Body.Left + (Body 宽 - 卡宽)/2`, 不再用左右 offset。进场沿堆叠方向(底部从下、顶部从上), 退场四角向左右滑出、上下居中向对应方向滑出。
    - **richColors**: `WithToastRichColors()`(整组)或 `ToastOptions.RichColors`(单条)开启后, 卡面/描边/标题描述文字按类型换成 sonner 的 `--success/--info/--warning/--error-{bg,border,text}`(亮/暗两套, 取自 sonner styles.css, 不是 shadcn 令牌); default 类型不换。动作按钮保持普通配色(与 sonner 一致)。
    - **布局坐标一律用"卡片矩形", 元素矩形由设置函数外扩**: 元素 = 卡片 + 四周 6px 投影留白。若把 `it.h`(按 padding+内容算出的卡片高)直接当元素高, 卡片可见高度就少 2×6 = 12 —— 上下内边距各被吃掉 6px, 表现为"文字顶到卡片底、上留白大下留白小"。修法: 排版/布局全按卡片坐标算, `toastSetRect(h, x, y, w, h)` 内部统一减 6 并加 12; 实测两行卡片 topGap=botGap=28 物理像素(@150% DPI), 单行卡片 26/27。
    - **`GetTextShowRect` 的 CY 已含行距**(实测: 微软雅黑 13px 单行 18 / 两行 36 / 三行 53), 与 `p.Text` 实际排版一致 —— 不要按行数再乘系数补偿, 否则底部多出一整行留白。
    - **位置/透明度动画挂窗口**(`xui.Presence(hWnd, key, …)`, hOwner = 窗口): 透明元素的局部重绘**不擦底**, 卡片一移动就在旧位置留下残影; 挂窗口后每 tick 整窗重绘, 滑入与重排都是干净的(代价是动画期间整窗重绘一次/帧, 通知进出场只有 200ms)。动画期间 `layout` 只更新 `baseX/baseY`、不直接落位, 否则会和 apply 的位置打架。
    - 进场前先 `SetElementsAlpha(0, ele)` + 整窗重绘一帧, 再启动 Presence —— 元素 alpha 要画一遍才落地(同 57/73 的首帧问题), 否则第一帧是上一次的 255。
    - 主题切换: 每条通知 `OnThemeChangeFor` 重新量高再重排 —— 字号/字体变了, 描述折行高度也会变, 只重绘不重量会裁掉末行。

76. **自绘标题栏窗口的封装: `EnableLayout` 必须调, 双击判定别看 `GetStayEle != 0`, 最大化不用手动去阴影**(`ui/window.go`):
    - **`XWnd_EnableLayout(true)` 漏调 = 子项"存在但不排版"**: 窗口上 `LayoutItem_SetWidth/Height` 全都不生效, 子元素停在创建位置(实测内容区 rect `{12 0 628 0}` 高 0, 界面只剩标题栏)。封装里要在创建子项**之前**开窗口布局; 用户的子项是之后加进内容容器的, 不用再手动 `AdjustLayout`(LayoutEle 增删子项会自己重排)。
    - **双击标题栏判定不能用 `GetStayEle() != 0`**: 标题/图标元素为了不挡窗口拖动都开了 `EnableMouseThrough(true)`, 鼠标停在它们上面时 `GetStayEle()` **仍返回该元素**(非 0) —— 按"非 0 就跳过"写, 双击标题栏整段失效(实测: 事件到了, 但被判断挡掉)。正确做法是只拦按钮: `XC_GetObjectType(stay) == xcc.XC_BUTTON`。
    - **y 判定要算阴影带**: 事件坐标原点是整窗左上角, 非最大化时标题栏落在 `[shadowSize, shadowSize+captionH]`, 最大化时 `[0, captionH]` —— 只写 captionH 会让非最大化时的下半段双击失效。
    - **最大化不需要手动去阴影**: 阴影窗最大化后窗口矩形 = 工作区(实测 1707×1018 逻辑, 屏幕 1707×1067, 差的就是任务栏), 阴影带由系统处理; 别在 WM_SIZE 里去 `SetShadowInfo(0, …)`(会闪)。
    - **三键**: `SetTypeEx(xcc.Button_Type_Min/Max/Close)` 直接拿到原生行为(最小化/最大化/关闭窗口); 自绘图标不会跟状态走, 要在 `WM_SIZE(SIZE_MAXIMIZED/SIZE_RESTORED)` 里换 —— 经 `XC_CallUT` 延后到消息处理之后再读 `IsMaxWindow`, 否则拿到旧状态。
    - **窗口销毁时 `xui.UnregisterWindow`**: 句柄会被系统复用, 主题注册表里留着会让切主题操作到别的窗口(Win32 `WM_DESTROY` 里反注册)。

77. **只隐藏不销毁的辅助小窗会让进程挂在 `Run()` 里**(`ui/tooltip.go`, 实测):
    - **现象**: 点窗口关闭钮, 主窗口消失但进程仍在(任务管理器里还有); 残留窗口是悬浮过标题栏按钮后创建的提示气泡小窗, 有时还留在屏幕上。
    - **根因**: 消息循环退出的条件是**所有用户创建的窗口都销毁**。Tooltip 的置顶小窗平时只 `ShowWindow(SW_HIDE)`, 窗口对象一直在; 宿主要销毁时, 挂在它上面的退场过渡也不再执行, 气泡连隐藏都来不及。
    - **修法**(`ui/tooltip.go`): 气泡首次在某个宿主窗口上显示时, 给宿主挂 `WM_DESTROY` —— 丢弃该宿主上的过渡(`DropTransitions`)并销毁气泡窗, 状态清零, 下次显示由 `ensure` 懒重建(气泡最后显示在其他仍存活的宿主上时保留, 等那个宿主销毁再清)。
    - **坑**: 销毁辅助窗时**先清状态再 `DestroyWindow`** —— 销毁过程可能同步派发迟到的鼠标事件, 用旧的 win/ele 句柄会弹 `XEle_SetRect()` 无效句柄错误框; 宿主从收到 `WM_DESTROY` 到销毁完成期间用 `dying` 集合直接拒绝迟到的 `show()`。**组件自建的辅助窗口都要有"宿主销毁 → 辅助窗销毁"的路径**, 否则会挡住消息循环的退出判定; date/time 弹层已由触发元素的 `Destroy` 事件销毁 `popoverPanel`。

78. **输入框的内嵌按钮: 动作要放在 `LButtonUp`, 不能放 `LButtonDown`**(`ui/input.go`, 实测):
    - **现象**: 步进器连点两次 `+` 只加一步, 而且输入框里的数字被整段选中(蓝底); 把动作(清空/切换明文/步进)写在 `AddEvent_LButtonDown` 里就会这样。
    - **根因**: 连点同一个位置时, 第二次按下在系统里是 `WM_LBUTTONDBLCLK`, xcgui 不会把它翻译成元素的 LButtonDown 事件 —— 第二下动作丢; 原生 Edit 把双击当"选中单词"处理, 于是文字被选中。**抬起消息两下都有**。
    - **修法**: `LButtonDown` 只 `*pbHandled = true` 吞掉按下(光标不会跳到按钮底下, 也避免原生开始选词), 真正动作放 `AddEvent_LButtonUp`; 命中判断仍用 `partAt(pPt)`(parts 矩形在 paint 里算)。
    - **补充**: 光有 `LButtonDown` 还不够 —— 连点的第二次**按下**是 `WM_LBUTTONDBLCLK`, 它**不走** `LButtonDown`, 只绑 Down 的话原生 Edit 照样拿到双击去"选中单词"(现象: 双击步进器按钮, 数字被选蓝)。必须再绑一个 `AddEvent_LButtonDBClick` 并在命中内嵌件时 `*pbHandled = true`(可与 Down 共用同一个处理函数)。
    - **顺带**: 动作里 `SetFocus()` 之后一定要 `MoveEnd()` —— 原生 Edit 获得焦点时会顺手全选, 不全选的话下一次输入会把整段内容替换掉(步进器表现为"点完 + 再打字, 数字全没了")。
    - **hover 高亮不受影响**: 内嵌按钮的悬停仍走窗口级 MouseMove + `partAt`(原生 Edit 的 MouseMove 只在进入时报一次, 见 21)。

79. **密码框遮挡符默认是 `*`, 要显式设成实心圆点**(`ui/input.go`, 实测): `EnablePassword(true)` 之后用 `XEdit_SetPasswordCharacter(hEle, '●')`(U+25CF)。默认 `*` 在大多数字体里只有 8px 左右的字形且贴基线, 看起来像"小一号的字", 与浏览器 `-webkit-text-security: disc` 以及 Windows 原生密码框的圆点不一致。注意**每次重新 `EnablePassword(true)` 都会把它重置回 `*`**, 所以明文→密文切回时要再设一次。

80. **原生 `Edit` 的内边距必须自己给(多行尤其)**(`ui/input.go`, 实测):
    - **现象**: 文本域里正文第一行压住圆角描边、纵向滚动条贴着右边框(单行输入框因为自己算 `SetBorderSize` 才没事)。
    - **根因**: `widget.NewEdit` 的内边距默认是 **0**, 原生 Edit 会把文字从客户区 (0,0) 开始排, 滚动条也画在客户区右缘; 我们在 paint 里画的圆角描边是 1px 内描边, 于是文字/滚动条直接压上去。
    - **修法**: 构造时按形态各给一份内边距 —— 单行 = `左(图标/前缀宽度和) / 6 / 右(按钮/计数/后缀宽度和) / 6`, 多行 = `10 / 6 / 10 / 6`。改主题时重算(字体换了前后缀宽度会变), 但**不要在 paint 里调**: `SetBorderSize` 会触发重排+重绘。
    - **顺带**: 多行的高度由调用方在创建参数里定, 加了纵向内边距后同样高度会少显示小半行 —— 演示/用法里的文本域高度要相应加 12。正文是主题字体时**一行约占 22 逻辑像素**, 所以"干净显示 4 行"要 `12 + 4×22 ≈ 100`(演示就是 100); 高度写小了最后一行会被裁掉半截。
    - **上下留白要故意不对称 1px**: 原生 Edit 把行盒居中在 text rect 里, 而数字/字母的墨迹只占行盒上半部分(下半是 descender 空白), 上下都留 6px 时实测"上方 14 物理像素 / 下方 17"(32 高输入框), 肉眼看就是"文字偏上、下面空得多"。单行改成 **上 7 / 下 5**, 实测 16/15 基本居中, 同时与自绘前后缀(按整高 Vcenter)的基线也对齐。多行(textarea)仍用 6/6 —— 正文本来就该顶对齐。

81. **内嵌按钮在任何状态都要吞掉点击, 否则连点会选中文字**(`ui/input.go`, 实测):
    - **现象**: 步进器数值到上限(如 100)后连点 `+`, 输入框里的 "100" 被选中成蓝底; 有时还会在别处留下一层没擦掉的选中文字残影。
    - **根因**: 到边界时按钮"不可点"就不登记命中矩形, 于是按下落到下面的原生 Edit —— 连点的第二下是 `WM_LBUTTONDBLCLK`, 原生 Edit 按"双击选中单词"处理(见 78)。
    - **修法**: 命中区**无条件登记**(只要元素未禁用), 到边界只是把图标画淡(40% α)并跳过 hover 高亮; 动作仍走 `activate`, 里面的 `stepBy` 会自己夹住值域, 等于空操作。
    - **同时**: 关掉原生"获得焦点自动全选"(`EnableAutoSelAll(false)`) —— 内嵌按钮点完都要 `SetFocus()` 回输入框, 全选会真的把内容选成蓝底(用户可见), 取消选中后还可能留下未擦除的选中层。

82. **原生 `Edit` 不继承窗口字体, 必须显式 `SetFont`**(`ui/input.go`, 实测):
    - **现象**: 输入框里的字比周围小一圈, 数字/中文"挤在一起"(用户描述"这个字体和其他的不同吗")。
    - **根因**: `widget.NewEdit` 建出来的编辑框用的是 xcgui 内置默认字体(探针实测 `XEle_GetFontEx` = 微软雅黑 **9pt**), 它**不吃** `w.SetFont(主题字体)` 的继承(表头/树/下拉框都靠组件自己 `SetFont`, 见 `combobox.go` / `table.go` / `tree.go`)。主题是 14px → 11pt, 于是输入框正文比标签小 18%。
    - **修法**: 构造时和换主题时都 `SetFont(xui.FontHandle(t.FontDefault))`。
    - **验收方法**: 量"同一串字符的实际绘制宽度" —— 把同一串 18 位数字同时放进输入值和 `WithPrefixText`: 修前原生 Edit 只有 126 逻辑像素(9pt), 修后 153, 与自绘前缀 152 基本同宽。只查 `XEle_GetFont` 的句柄不够: 句柄对了但 pt/px 换算错了, 两边照样对不上。

83. **paint 回调里换过的字体必须还原, 否则会带偏原生文字**(`ui/input.go`, 实测):
    - **现象**: 带字符计数的输入框里, 正文的字体和字距跟旁边输入框明显不同(用户: "字符计数里面的字体与其他不同, 文字间距也大"), 看着像没对齐。
    - **根因**: `xui.BeginPaint(hEle, hDraw)` 拿到的建图对象**在回调返回后还要继续给原生文字用**, 而计数是 12px、前缀后缀是主题字号 —— 在 `paintAddons` 里 `p.Draw.SetFont(...)` 之后不还原, 原生 Edit 的正文就会接着用最后设的那个字体画。
    - **修法**: `onPaint` 结尾统一 `p.Draw.SetFont(xui.FontHandle(t.FontDefault))` 还原。同理, 任何在 paint 里临时改画笔状态(字体/画刷/裁剪)的组件都该还原后再返回, 不要指望原生绘制自己重置。

84. **`EnablePassword` 切换不会重排字符步进, 切换后必须手动触发一次重排**(`ui/input.go`, 实测):
    - **现象**: 点眼睛切明文后, 同一串 `secret-123` 被撑成 `s e c r e t - 1 2 3`(每个字符都占密码圆点的全角宽度); 再切回密文时圆点同样被撑开。奇怪的是"点一下切主题"就恢复正常。
    - **根因**: `XEdit_EnablePassword` 只换遮挡开关, 内部缓存的字符步进还是上一个模式的(明文按圆点宽度排, 或反之)。换主题能修好, 只是因为主题回调重设了内边距/字体, 顺带触发了重排。
    - **修法**: 切换后调一次 `syncPadding()`(里面的 `SetBorderSize` 改了 text rect, 原生会按当前模式重新排版)。**不要用"重新 `SetText`"来修** —— 虽然同样能修好, 但会给用户一个假的 `OnChange` 回调(值其实没变)。同理, 想触发重排时优先用"改 text rect / 改行高"这类纯布局操作。
    - **配套**: 组件公开 `SetPasswordVisible(bool)` / `PasswordVisible()`, 与点眼睛按钮走同一条路径(见语义表)。验收: 切换后量同一串文本的墨迹宽度 —— 明文行与同窗口里的普通输入框逐像素同宽(108 逻辑像素 / 10 个字形块), 密文圆点行与切换前完全一致。

85. **焦点环/校验环画在元素内部, 内嵌控件要按环宽内缩**(`ui/input.go`, 实测):
    - **现象**: 输入框有焦点框时, 右侧眼睛/清除钮的悬停底色直接压到边框上(用户: "有焦点框时, 这种右侧的按钮悬浮态右侧是压到了些边框"), 看着像啃掉一块边。
    - **根因**: 环带是 `FocusFrame/RingFrame` 画在**元素客户区内侧**(inset 3), 而内嵌按钮的悬停底色从"距壳体内缘 4"开始 —— 4 > 3 只差 1px, 圆角处实际已经压住环的内描边。
    - **修法**: 尾部按钮的内缩取 **6**(= 环宽 3 + 余量 3, 正好也是 shadcn input-group add-on 的 `pr-1.5`), 悬停方块与环之间稳定留 3px。规律: 只要组件在"带内描边环"的容器里画控件, 内缩都别小于环宽 + 2~3。
    - **验收**: 深色主题下聚焦 + 悬停截图放大, 悬停方块右缘与环内描边之间有可见间隙。

86. **按钮组(ButtonGroup)的布局/绘制语义**(`ui/buttongroup.go`, 实测):
    - **Auto 宽高不会因为 `SetSpace` 自动重算**: `LayoutEle` 的 `Auto` 宽是按已有子项量出来的;在尺寸事件里改了间距后只调**组自己**的 `AdjustLayout` 只会重排子项, 组的外框仍是旧宽(嵌套组第二个组的右缘直接溢出外框)。修法: 改完间距/子项尺寸后调**父元素**的 `XEle_AdjustLayout(parent)`, 父级才会按新的 Auto 尺寸重新给组分配矩形。尺寸事件里调父级会再触发本组的 Size 事件, 用 `fitting` 布尔量防止递归。
    - **Auto 高不会随子项变小而缩回**: 横向 sm 组里文本块初始 32 高, 组 Auto 高先被顶到 32;之后把文本压到 28, 即使重声明 Auto + 调父级, 组高仍停在 32(实测)。修法: 在尺寸事件里按**本轮计划**的子项高(按钮取元素高、文本/分隔条取目标高、嵌套组取元素高)自己算一次组高:横向取最大值、纵向求和(含外边距与间距), 再 `SetHeight` 写回;`LayoutItem` 保持 Auto, 用户显式指定的 Fixed/Fill/Weight 高不接管。
    - **竖向组的 1px 按压预留要逐项消缝**: Button 的元素高 = 视觉高 + `pressShift(1)`, 横向并排时多出的 1px 落在底边不影响邻项;竖向堆叠时就会在每两个按钮之间留 1px 透明缝。`SetSpace(-1)` 能消缝但会作用到**所有**子项(文本块/分隔条也会被叠 1px), 正确做法是给"前一个是按钮"的子项设 `LayoutItem` 上外边距 `-pressShift`, 其余保持 0。
    - **元素没有 Go 类型信息**: 按钮组靠 `XEle_SetUserData` 标记角色(button/groupH/groupV/text/separator), 子项绘制时用 `XWidget_GetParentEle` 找到父级、读标记判断是否在组内, 再用 `XEle_GetChildByIndex` 的序号判断首/中/尾。`DrawRoundRectEx/FillRoundRectEx` 接受四个独立圆角, 正好表达"外角圆、内角方";相邻边框各画一半(描边沿路径居中、贴边被元素裁剪)会拼成 1px, 不需要额外删边。输入框与下拉框的壳体也调用同一套 `groupCorners`(焦点环/校验环用 `RingFrameCorners`), 因此可以直接作为组子项;这些组语义的公共代码在 `components/group.go`, 注册表里随 button/input/combobox/buttongroup 一起拷贝。
    - **Auto 宽要求子项有确定宽度**: 布局量不到文字宽度, 过去按钮 `cx=0` 只有图标档有宽、文字按钮会塌成 0。现在 Button 在 `cx=0` 时用 `xui.TextWidth` 量 `内边距 + 行内图标 + 文字 + 间距`(纯图标用方形预设), 组才能"按内容拼成一条"。
    - **只有文本/分隔条的组**: 文本块初始高度按 32 建, 才能把 Auto 高的组撑起来;横向组里若按钮更矮(sm/xs), 组尺寸事件再按**最高按钮的视觉高**把它压到 28/24 —— 目标高取按钮视觉高而不是组高, 否则初始文本的 32 会把目标算错一档。

87. **持续旋转动画(spinner)与图标旋转**(`anim.go` 的 `Spin` + `icons.go` 的 `IconRotated`, 实测):
    - **不要为每个 spinner 起定时器**: 复用 hover 的"每元素一个炫彩定时器"(`animTimerID`)。`xui.Spin(hEle, true)` 登记一个旋转项并返回相位(0..1, 按**墙钟**算, 1s 一圈; 掉帧不会走慢), `Spin(hEle,false)` 删除它并在没有其它过渡时立刻关定时器。组件在 paint 里加载中调用即可;停止加载时 `SetLoading(false)` 主动调 `Spin(false)`, 不用等下一帧。
    - **相位只在绘制时读取**: `hoverTick` 看到旋转项就立即重绘(约 30ms/帧), 不保存逐帧累加值;`SetAnimationScale(0)` 会冻结在相位 0, 便于截图对比。
    - **旋转不能重建几何**: 图标几何按 path+size 缓存, 把角度烘进 key 会每帧重建(23fps 以下)。`Painter.IconRotated` 把旋转放进渲染目标变换矩阵, 以绘制方块中心为基准;D2D 矩阵 `x'=x·M11+y·M21+DX, y'=x·M12+y·M22+DY`, 旋转 + 缩放合成后:
      `M11=s·cos, M12=s·sin, M21=-s·sin, M22=s·cos, DX=ox+c-s·cos·c+s·sin·c, DY=oy+c-s·sin·c-s·cos·c`(c=size/2)。矩阵必须成对恢复(见第八节 43)。
    - **GDI 回落也转**: 折线回落在点坐标上绕方块中心做同样的旋转;D2D 主路径与 GDI 兜底的观感一致。
    - **禁用不影响旋转**: 加载按钮通常 `Enable(false)`, 但 spinner 仍应转;禁用只把图标颜色按 50% 透明度预混(见 61)。

88. **高 DPI 下"文字/图标/壳体按下位移不一致"**(`button.go` 的 `shellPressShift` + `TextF` + `Painter.IconRotatedOffset`, 150% DPI 实测):
    - **现象**: 默认尺寸按钮按下/松开时, 文字和行内图标看起来不是一起下沉; sm 尺寸的按钮有时又正常。前后导、尺寸档都可能影响, 逐帧测量才能定位。
    - **根因**: `active:translate-y-px` 的 1 **逻辑**像素在 150% DPI 下是 1.5 物理像素。壳体、文字排版、图标盒分别经自己的坐标取整, 最终位移可能是 1 或 2 物理像素, 而且**取决于该元素所在的物理位置/排版相位**。例如同一次按下, 壳体与文字移 2 物理、默认图标移 1; 换一个窗口位置, 壳体/文字移 1、图标可能移 2。不要假设"文字位移 = round(1×scale)": 实测同一 DPI 下不同 y 位置会得到 1 或 2。
    - **修法**: 内容不再各自按逻辑坐标平移。先让壳体按原逻辑量移动, 用 `XDraw_ConvRect` 量**壳体自身**的未按下/按下物理 top, 得到实际位移 N(1 或 2 物理像素); 然后:
      - 文字按**未按下矩形**排版, 用 `TextF` 的浮点矩形加上 `N / DPI scale` 逻辑偏移(实测浮点矩形会精确落到 N 个物理像素);
      - 图标按**未按下位置**画, 用 `IconRotatedOffset(..., dy=N)` 直接加物理像素;
      - 壳体继续按整数逻辑位移。三者都走同一个 N, 不再有相位差。
    - **范围**: 纯图标按钮没有文字, 但图标也跟随壳体 N; 行内图标(前导/尾随/加载 spinner)同理。GDI 折线回落忽略 `IconRotatedOffset` 的物理偏移。分页(上一页/下一页/页码)的按下位移同一套: 壳体量实际物理位移 `shellPressShift`, 文字用 `TextF` 加 `N/scale`、图标用 `IconRotatedOffset(...,N)`, 三者一起下沉。
    - **验收**: 60fps 录屏 + 逐帧墨迹质心。sm/default × 前导/尾随 四种组合的壳体、文字、图标位移完全一致(本机该行均为 +2 物理); 换到壳体只取整 1 物理的位置时三者应同时为 +1。不能只验证"图标自己移动了", 必须验证与壳体、文字的**相对位移**一致。

89. **元素半透明时, 同色图元重叠会二次叠色**(Tooltip 箭头淡入/淡出时"倒三角上方多一个正三角", 实测):
    - **现象**: 提示气泡淡入/淡出经过半透明阶段时, 气泡下缘的箭头位置浮出一个更深的"正三角", 与露在外面的倒三角拼成菱形鬼影; 完全不透明时几乎看不见(只剩一条抗锯齿细边), 正常速度 150ms 一闪而过, 容易误判成"残影/窗口没擦除"。
    - **根因**: `XEle_SetAlpha` 不是"先把整个元素画到离屏层、再整体乘 alpha", 而是**逐图元**混合 —— 每个 Fill 都各自与目标缓冲做一次 `src-over`。旧画法是"先画一个完整菱形箭头, 再用气泡圆角矩形盖住内侧一半": 不透明时后画的气泡完全覆盖, 看不出接缝; 半透明时重叠区被混了两次, 颜色更深, 于是被盖住的那半个菱形又显出来了。任何"用同色图元拼一个异形轮廓"的画法(圆角矩形+箭头、两个半圆拼胶囊)都会中招。
    - **修法**: 拼轮廓的图元**不要重叠**, 只画轮廓外露的部分。Tooltip 现在只画露在气泡外的三角形, 底边与气泡边线共线, 不再画被盖住的那一半; 文字画在气泡上不受影响(颜色不同, 本来就是覆盖关系)。不要用"把底边压进气泡 1px"来防缝 —— 那会留下一条更深的 1px 线; 共线即可, 同一次绘制里相邻边的抗锯齿不会产生透底缝。
    - **验收**: 用 `xui.SetAnimationScale(20~200)` 把进出场放慢(参数是**时长倍率**, 20 = 慢 20 倍; 0 = 立即到位), 在 alpha≈0.3/0.5 的中段帧截图放大检查; 正常速度下只有约 9 帧, 截图脚本很难稳定命中, 也看不出叠加。亮/暗主题各验一轮。
    - **同源坑(对话框页脚带)**: 旧画法是先 `FillRound` 整块半透明页脚、再用方角矩形补上沿的两个角 —— 补角区被合成两次, 按钮上方出现一条比页脚其余部分深一档的条带(实测亮色 #f7f7f7 vs 页脚 #fafafa)。修法 = `Painter.FillRoundCorners` **一次**画出"上平下圆"的整块(`dialog.go` 的 onPaint)。任何半透明块都不要用"整块 + 补角块"叠出来 —— `CardFooter` 后来用"补角块"画法把同一条 #f7f7f7 带又画出来过一次, 同样改成 `FillRoundCorners`。

90. **折行文本的高度与 Auto 布局重排**(`components/alert.go` + `components/label.go`, 实测):
    - **现象**: Alert 的长标题/描述只显示一两行就被省略号截断;把 Label 的布局项高度声明成 `Auto` 也不会自撑。
    - **根因**: Label 载体是原生 Button, Layout 的 `Auto` 只取元素**现有高度**, 不会按文本折行结果量高;必须量出高度后 `SetHeight` + 布局项 `Fixed`。
    - **量高口径**: `app.GetTextShowRect` 的 CY 略小于"行数×行高"(14px 三行返回 59 而非 60), 且 Button 的文字区比元素窄/矮一圈(默认内边距) —— 直接按元素宽度量会少算一行, 只按整行取整又会裁掉最后一行。修法:按 `宽度−12` 量, 多行高度取整到 20 后 **+4**;单行保持 20(数值在 `alert.go` 的 `alertTextInsetX` / `alertTextPadY`)。
    - **重排链**: 改完文字高度只调自己/直接父级不够, Auto 高的卡片/行仍按旧高度排版, 要逐级 `XEle_AdjustLayout` 到祖先。遇到 ScrollArea 的 `LayoutFrame` 时**不能**调它的内容区(实测会把 Alert 宽度打成 0), 要调框架本身(`XEle_AdjustLayout(frame)`), 由框架重排内容区并重算滚动范围。
    - **重绘**: 高度变化后要重绘标题/描述和窗口;只重绘 Alert 背景不会触发子元素重绘。
    - **触发时机**: 布局赋予宽度时尺寸事件不一定触发;在 Alert 的 paint 里比较文本列当前宽与上次量高宽度, 变化时预约一次 `AfterFunc(0)` 重排。不要在尺寸/paint 回调里直接改布局(会重入)。
    - **验收**: demo 反馈页 1200/960/1500 三种宽度、亮/暗主题下长标题与长描述完整显示、无省略号, 动作按钮始终贴右上;亮/暗运行时切换颜色实时跟随。

91. **AlertDialog / 多层模态 / 折行量测的一批结论**(`ui/alertdialog.go` + `ui/dialog.go`, 实测):
    - **多层模态必须自己做栈**: 同一窗口开两个 Dialog(例如对话框里再开 AlertDialog)时, 每个实例都在窗口上注册了 `AddEvent_KeyDown`。不排栈的话一次 ESC 会把两层一起关掉, Tab 也会被两个实例同时搬焦点。修法 = 包级 `modalStack` 记录打开顺序; `isTopModal()` 为假时 ESC/Tab/遮罩点击全部忽略; `materialize` 后按栈顺序重抬 z 序(`raiseModalStack`, 逐个"backdrop → overlay", 后打开的压在上面); 关闭栈顶后 `restoreModalFocus` 把焦点还给下层最近聚焦的控件。最小化还原时各实例的 WM_SIZE 回调都会跑, 抢焦点也要加 `isTopModal()` 条件。
    - **AlertDialog 的语义不能照抄 Dialog**: radix 的 AlertDialog 对 `onInteractOutside` 做了 `preventDefault`(点遮罩不关), 但 **ESC 仍会关闭**(源码没有拦 Escape); 打开时 `onOpenAutoFocus` 强制聚焦 Cancel。实现里用 `dismissOutside=false` + `initialFocus=Cancel` 表达。页脚 `Cancel` 先于 `Action` 加入, 这样焦点序列的第一项就是 Cancel。
    - **Label 的文本必须先 `SetText` 再参与量高**: `Label.WrappedHeight` 量的是 **Label 当前文本**。如果只在 layout 里量完高再 `SetText`, 量到的是空串 → 返回 0 → 描述只占一行并被省略号截断(现象: 明明该折 3 行的描述只显示一行)。`SetTitle/SetDescription` 现在立即把文本写进 Label; layout 里再设一次只是兜底。
    - **整树显隐会覆盖"可选子元素"的隐藏状态**: `showEleTree(overlay, true)` 会无条件显示所有子元素, 所以判断"有没有 media"不能用 `media.IsShow()`(永远为真, 空 media 也占 40+16 的宽度)。用独立的 `mediaIcon != ""` 判断; media 在构造选项里创建时还要补一次 `showEleTree(e, d.open)` 与对话框状态同步。
    - **折行是字符级, 没有 word wrap**: xcgui 的文本排版不支持 DirectWrite 的 word-break 控制(`TextFormatFlag_` 里只有 NoWrap), 英文长词会在任意字符处断开(如 "permane / ntly")。中文排版无影响; 需要英文严格按单词换行时只能在文案里手动加换行(本库不做 `text-balance` / `text-pretty`, 与上游的差异见语义表)。
    - **可滚动内容的量高顺序**: `WithDialogScroll` 的内容区是 ScrollArea。先给框架一个临时高度并 `XEle_AdjustLayout`(让内容区子项拿到宽度), 用 `childrenBottom` 量出自然高, 再夹到 `min(70vh, 面板可用高度)`; 面板排好后用最终高度再排一次框架(重算滚动范围)。头部/页脚始终在框架之外, 所以滚动时它们固定。遮罩元素还要 `EnableEvent_XE_MOUSEWHEEL(true)` 并吞掉滚轮(`*pbHandled=true`), 否则 modal 打开时滚轮会穿透到页面的 ScrollArea(浏览器里 body 是锁定的); 对话框自己的 ScrollArea / 原生列表会在子元素上先消费滚轮。

92. **过渡"到位的那一帧"必须重绘, 否则屏幕停在倒数第二帧**(`anim.go` 的 `hoverTick`, 实测):
    - 现象: 弹层(`xui.Presence` 挂弹窗自己)淡入后**停在半透明**, 能透出后面的页面文字, 等 1s 以上才"突然变实"。抓帧序列看到弹层出现后 alpha 停在某个中间值不再变化 —— 不是过渡太慢, 是**最后一帧没有上屏**(占位截图里"透字"的半透明弹层即此)。
    - 根因: 定时器 tick 里, 最后一次 `apply(v=1.0)` 与"收摊"发生在同一次回调: 该帧所有条目都已到位, `active=false`, 于是走不到 tick 末尾的重绘 —— 属性已写成终态(alpha=255), 但窗口表面还是倒数第二帧的内容, 要等下一次无关重绘才被 DWM 合成出来。
    - 修法: tick 末尾的重绘条件从 `if active` 改成 `if active || len(applies) > 0` —— 这一帧跑过 apply 就补一次重绘。与第八节 21 "淡出到位那帧 anim 不重绘"是同一类问题(那次的修法是 done 回调补 Redraw, 这次直接修在 tick 里, 所有 Presence / hover 过渡受益)。

93. **"宽度被布局拉伸的 Auto 高组件"要自己重算高度, 且改动要逐级通知祖先**(`components/calendar.go`, 实测):
    - 现象: 月历卡片被 `xui.ItemFill` 拉宽后, 元素保持构造时的高度 —— 格子变大、6 行挤在一起; 就算高度算对了, ScrollArea 的滚动范围也可能没跟着长, 底部内容滚不出来。
    - 修法(三半缺一不可):
      ① 元素挂 `AddEvent_Size`, 宽度变化时按新宽度重算自然高度(格子/内边距/字号全随宽度等比, 页脚换行的行数也重算);
      ② **尺寸事件里不能直接 `SetHeight` + `AdjustLayout`**(重入布局), 用 `app.AfterFunc(0, ...)` 推迟到下一轮消息(同第八节 90 的 scheduleReflow);
      ③ 高度变化后从自己开始逐级对祖先 `AdjustLayout`; 遇到 ScrollArea 的 LayoutFrame(或"父级是框架内容区")交给框架排 —— 只通知直接父级时 Auto 高的卡片仍按旧高排版, 滚动范围也不刷新。(宽度驱动高度的组件不要照抄这一条: 逐级重排会让中间层的量算趟把组件临时排成 0 宽再恢复, 尺寸事件无限自激, 见第八节 94。)
    - 同族坑: **原生 Edit 子元素的宽度不会跟随父元素**。DatePicker 的可输入形态把 Edit 建成触发区的子元素, 触发区被布局拉伸后 Edit 仍是创建时的宽度 —— 右侧一大段点不进输入区; 同样要挂 Size 事件把 Edit 的宽高同步成"父宽 − 图标区"。

94. **AspectRatio(比例容器): xcgui 没有 aspect-ratio, 只能"宽度驱动高度 + 批量重算"**(`components/aspectratio.go`, 实测):
    - 布局项只有 Fixed/Fill/Auto/Weight/Percent/Disable 六种尺寸类型, 没有"另一边按比例"的类型; CSS 的 `aspect-ratio` 语义要自己维护。
    - 落地: 组件**不声明自己的布局项高度**(保持 Disable, 用元素自身高)。`AddEvent_Size` 只排队, **不在布局趟里 `SetHeight`**(同步改高会触发宽度 335↔0 的布局震荡并卡死); 下一轮把本帧所有比例框的高度/内容尺寸一次写完, 再只对最近的 ScrollArea 框架(或窗口)调一次 `AdjustLayout`, 最后整窗重绘一次。全部延迟会让首次显示逐帧展开(先两块 → 前两块变高 → 竖块出现), 所以必须**批量**而不是每块各跑一轮; 尺寸没变就完全不动(布局过渡期的额外尺寸事件不能回写, 否则自激)。
    - **高度由调用方接管时比例不生效**: 构造 `cy > 0`, 或布局项高度被声明成 Fill/Weight/Fixed 时, 布局会用外部值覆盖元素高度(同 CSS 宽高都确定时 `aspect-ratio` 失效); 传 `cy = 0` 且布局项高度不接管时才按宽度联动。
    - 内容区是 `Content()` 返回的子容器: 每次同步时把它**锁成比例框的实际宽高**(布局项 Fixed), 它内部的 Fill 子项才有正确参照 —— 直接让内容区用 Fill 参加布局时, 父级是 Weight 宽会让内层按祖先宽度算(实测比比例框宽 5~15px, 9:16 那块首帧高度还是 0, 见第八节 59 的同源问题)。圆角/底色/图片裁剪由内容自绘(元素级裁剪见第七节限制)。
    - **重排只改坐标, 不重绘被推走的兄弟**: 批量写完高度后要统一 `XWnd_Redraw`, 否则隐藏页首次显示时后面的卡片仍画在旧位置(看起来像重叠)。不要在父链上逐级 `AdjustLayout` / 重新声明 `Auto` —— 框架的量算趟会把比例框临时排成 0 宽、下一轮再恢复, 尺寸事件因此无限自激(实测 CPU 跑满、窗口卡死); 只调最近的 ScrollArea 框架即可, 框架重排会统一更新内容高、兄弟位置与滚动范围。
    - **验收(探针/逐帧)**: 800 宽窗口里 `ItemWeight` 宽的 16:9 容器 → 784×441; 窗口缩到 600 → 584×329; `SetRatio(1)` → 584×584, 外层 Auto 高的行同步长高; PrintWindow 逐帧抓页签切换, **首帧三块比例框就是最终尺寸**, 无展开; 切换后 CPU 稳定(不再跑满), 窗口缩放后比例跟随且无残影。

95. **CardFooter 的内边距用"内层内容区的元素 padding + 上下外边距"表达, 子项别直接挂在页脚上**(`components/card.go`):
    - 页脚是横向布局容器(`items-center` + 高度 Auto 自撑), 内层 `Content()` 宽 Fill、高 Auto; 左右内边距用**元素 padding**, 上下用布局项外边距(default 16 / sm 12)。与卡片内容区同一套原因: 四边都用外边距时, 页脚被 Fill 宽排过后内层子项会按页脚全宽解析而右溢(见第八节 129)。水平页脚的现有几何不变; 竖向页脚(登录卡的两个整宽按钮)也不会溢出。
    - 页脚 `cy = 0` 时高度 = 内容高 + 上下外边距; 传固定高时内容在页脚内垂直居中。直接挂到页脚上的子项会跳过内边距, 只有整宽背景/分隔线这类贴边内容才这样用。
    - 页脚做成布局容器后, 页脚里放按钮/文字用 `footer.Content()` / `ContentStack(space)`; 要竖排(登录卡的两个整宽按钮)对内容区 `EnableHorizon(false)`。

96. **窗口级 `AdjustLayout` 会重排窗口直属的绝对定位覆盖层(Toast), 不能拿它当"全局重绘"**(页签切换, 实测):
    - **现象**: 反馈页右下角的通知, 切换到其他页后跑到左下角(或左上角)。
    - **根因**: Toast 卡片是窗口直属的普通 Element, 用 `XEle_SetRect` 绝对定位; `XWnd_AdjustLayout` 会把窗口的所有子元素当布局项重排, 没有布局项声明的卡片就按默认位置排到左侧。
    - **修法**: 页面切换只调页面(ScrollArea)/滚动框架的 `AdjustLayout` + `XWnd_Redraw`(重绘只刷画面, 不动坐标); 不要调 `XWnd_AdjustLayout`。隐藏页的命中区刷新同样用页面级 `AdjustLayout` 延迟一次(见 66), 不需要窗口级布局。
    - **通用**: 任何窗口直属的绝对定位浮层(Toast / 自绘遮罩 / 自定义 HUD)都怕窗口级布局; 组件自己重排时用元素级/框架级 AdjustLayout。

97. **父元素按矩形裁剪子元素; Collapsible 的高度动画可以直接靠它遮罩**(`components/collapsible.go`, 探针 + 展开/收起中间帧):
    - **实测**: 子元素绘制被裁在**父元素客户区矩形**内(不只是"自身绘制被自身裁", 见 10)。80 高的父元素里, y=100 的子元素完全不画; x=250、宽 100 的子元素只画到父元素右缘。**圆角不参与裁剪** —— 圆角外的四角区域仍会画出来, 需要圆角裁剪时请自己按路径裁(第七节)。
    - **用途**: 收起内容只需要把内容容器的高度逐帧改小, 超出部分自动被父矩形裁掉, 不需要额外遮罩或 `XDraw_SetClipRect`; 这也意味着内容里的原生控件(Edit 等)在动画中间帧不会溢出到下方。
    - **高度动画的落地**: 先把内容布局项设 `Auto` 量出自然高度, 再锁定 `Fixed` 并从 0 开始; `xui.Presence` 每帧把 `target × v` 写回内容高度; 每帧从根容器向上 `AdjustLayout`(遇到 ScrollArea 的 LayoutFrame 交给框架排, 见 90/94), 再**整窗立即重绘**。展开到位后恢复 `Auto`(之后内容增减仍跟随布局), 收起到位后才 `Show(false)`。
    - **别把 Presence 的 owner 当唯一重绘目标**: `Presence` 的 tick 只立即重绘 owner(根元素)自己的矩形; 内容高度变化会把下面的兄弟项整体推走, 只刷根元素会留下旧位置的像素, 所以 apply 里要自己 `XWnd_Redraw(hw, true)`。

98. **滚动条的 alpha 是"悬停显隐"状态, 不能跟着弹窗整组淡入一起设置**(`components/scrollbar.go` + `components/dialog.go`, 用户截图):
    - **现象**: 可滚动对话框打开时, 有时候滚动条一开始就是亮的; 鼠标不动它就一直亮着, 移出去才淡出。
    - **根因**: 对话框的进出场用 `SetSubtreeAlpha` 给整棵子树逐个设 alpha(见 47); 滚动条也是子树元素, 被一起设成了 1。但滚动条的 alpha 本来是 `showScrollBarOnHover` 的显隐状态(初始 0, 鼠标移入宿主才 1), 被整组淡入覆盖后就"常亮"了 —— 是否看到取决于淡入后鼠标有没有移动到别处。
    - **修法**: ① 运行时新增 `xui.SetSubtreeAlphaSkipScrollBars`(遍历时跳过 `xcc.XC_SCROLLBAR` 及其子树), 对话框**打开**时的初始归零与淡入改用它; **关闭**时的淡出仍用普通 `SetSubtreeAlpha`, 让滚动条跟着面板一起淡出(否则半透明面板上会留一条亮条)。② `showScrollBarOnHover` 在宿主 `XE_SHOW(false)` 时把滚动条 alpha 归零并 `DropTransitions`, 页面切换/弹窗关闭/最小化后不会带着上次的显隐状态回来; `Dialog.Show` 再主动 `resetScrollBarHover(d.scroll.Handle)`, 覆盖"淡出中途快速重开"这种宿主没收到隐藏事件的情况。
    - **验证**: 可滚动对话框打开(鼠标移开)→ 滚动条不显示; 移入内容区 → 淡入; 在滚动条可见时关闭再打开 → 仍从隐藏开始。`resetScrollBarHover` 的闭包按宿主句柄登记, 宿主销毁时一并清理。

99. **右键触发区与菜单自适应宽度的一批实测结论**(`components/menu.go`, 探针 + 截图):
    - **元素右键事件不冒泡**: 在父元素里的子控件上右键, 只有子控件和**窗口**收到 `RButtonUp`, 父元素收不到(探针: 子按钮回调 `CHILD` + 窗口 `WIN`, 没有 `PARENT`)。所以 `NewContextMenu` 绑在**窗口**上, 再用 `XEle_GetWndClientRect` 做矩形命中 —— 触发区内的子元素、文字区域都能弹; 代价是触发区里吃右键的原生控件(Edit/List 等)不会触发(窗口收不到那条消息)。
    - **窗口 `RButtonUp` 的 pPt 是逻辑像素, 原点 = 窗口矩形原点**(含标题栏/阴影边距), 与 `XEle_GetWndClientRect` 同口径, 可直接比较, 不要乘 DPI。多个触发区重叠时后创建的先处理(窗口事件按注册逆序执行), 处理后置 `pbHandled`。
    - **菜单宽度不能依赖原生自然宽**: 原生菜单用自己的字体(与自绘 14px 不一致)量文本; 只给第一个条目垫 min 宽、其余留自然宽时, 长中文会被自绘裁剪吃掉(探针截图: 尾部少字)。修法是**逐条用 `xui.TextWidth` 按自绘字体算文本区宽**: `inset/图标 +22`、`Check/Radio/Sub 右侧恒 +24`、快捷键 `+12 + 小字宽`; 弹窗宽 = max(条目宽)+48, 条目盒内文本区另有 28px 余量, 足够容纳。
    - **未勾选的 Checkbox/Radio 也要留右侧 24**(shadcn 的固定 `pr-8`): 只在"勾选时"才让位的话, 同组条目在勾/不勾之间文字右缘会跳。
    - **`MenuItem_State_Flag_Popup` 是"该项有子菜单"的静态标记, 不是"子菜单已展开"**: 把它并进 hot 会让每个 Sub 触发行常亮(截图复现)。子菜单展开时原生会保持父项的 `Stay`(真实鼠标滑入验证), 不需要额外状态。
    - **ContextMenu 的 OnOpenChange 时序**: 菜单 `Popup` 是阻塞的; true 在弹出前触发(可在回调里给触发区置展开态并 `Redraw`), false 在关闭后触发。菜单是一次性对象, 每次右键在 build 里重建, 勾选等状态由调用方持有。
    - **菜单进场两种淡法都会闪, 最终不做动画**(用户录屏逐帧确认): ① 整窗 alpha —— popover 底色一起半透明, 淡入过程中后面的页面文字透到菜单上, alpha 到 255 才被盖住; ② 只淡内容(面板/阴影不透明)—— 第一帧是一块**空白面板**或极淡的文字, 录屏里同样明显。位移/缩放又做不了(会崩, 见 64)。现与原生菜单一致: 弹出即全内容; "衔接感"由触发区自己的 hover / 展开态提供(demo 的 `contextTrigger`: `xui.Hover` + `OnOpenChange`, 透明元素先铺 `t.Card` 再叠悬停色, 见 28)。
    - **右键触发区要有 hover / 展开反馈**: 按钮触发区先变 hover/aria-expanded 再弹菜单, 右键区域如果什么都不变, 菜单就像凭空出现(与左侧按钮菜单对比时尤其明显)。做法: 触发区 `MouseStay/Leave` 里 `Redraw(true)` 启动 `xui.Hover`, 绘制 `t.Accent` 一轮; 打开态用 `OnOpenChange` + `xui.HoverItem` 再叠一档, 虚线边框按 hover/展开插值到 `t.Ring`。透明题字区必须先铺 `t.Card` 再叠悬停色, 否则淡出会留旧像素(见 28)。

100. **`XDraw_SetFont` 会重置文本对齐状态**(探针实测):
    - **现象**: 同一段绘制里先 `SetTextAlign(中心)` 再 `SetFont(...)`, 这一行文字退回**默认左对齐**; 先 `SetFont` 再 `SetTextAlign` 才是居中的。
    - **探针**: 同一元素三行同宽矩形 —— A = 先对齐后字体(左对齐)、B = 先字体后对齐(居中)、C = 先对齐后小字体(左对齐), 截图边界清晰。字体切换(含换字号)都会重置, 不是只对第一次生效。
    - **修法**: 顺序固定为 **先 `SetFont`, 再 `SetTextAlign`, 再 `DrawText`**; 用 `xui.Painter.Text` 时它内部就是"颜色 → 对齐 → 绘制", 只要别把 `SetFont` 放在它之后即可(绘制多行不同字号的文字时, 每换一次字体都要重新调 `p.Text`, 或在 `SetFont` 后补一次 `SetTextAlign`)。

101. **横向布局里的嵌套布局只认 Fixed 宽(Weight/Percent 不会传给 Fill 子项)**(`components/field.go` 的 `Field.Content`, 截图 + 元素矩形实测):
    - **现象**: `FieldHorizontal` 的 Content 用 `Weight 1` 吃剩余宽度时, Content 里的两个 `xui.ItemFill` Label 位置/高度都对, 但宽度恒为 0, 文字完全不画; 给 Label 设固定宽(300)或给 Content 设固定宽后立即正常。
    - **根因**: xcgui 的嵌套 LayoutEle 在拿到 Weight/Percent 宽时, 不会把宽度下发给自己的 Fill 子项; 只有父布局项是 **Fixed 宽**时, 内层 Fill 才有确定宽度。直接挂在横向 Field 下的 Weight Label 不受影响(父级就是当前布局)。
    - **无效的绕法**: 显式 `AdjustLayout`、`AfterFunc` 延迟重排、给 Content 用 Percent 50 都无效(实测子项宽仍为 0); 改 Fixed 宽是唯一稳定解。
    - **修法**: 横向 `Field.Content()` 默认 `Fixed 240`, 调用方用 `xui.ItemWidth(c, w)` 覆盖; 竖排 Content 走 Fill, 链路正常。同类的"横向嵌套列"布局都按此处理。

102. **元素重排后必须整窗重绘, 否则旧位置文字留下重影**(`components/field.go` 的 `FieldError.SetErrors`, 截图):
    - **现象**: 隐藏 FieldError 后下面的字段上移, 新位置画对了, 但旧位置的"显示名称 / 团队可见 / 说明"仍留在画布上, 与新位置文字叠在一起(用户截图里同一个"团队可见"出现两次)。
    - **根因**: xcgui 移动元素时不会自动擦除旧位置像素; 只 `Redraw` 错误区/自身覆盖不到搬走元素占过的矩形(透明布局容器不擦底, 同 28), 只调 `XWnd_AdjustLayout` 也只重排不重绘。
    - **修法**: 逐级 `XEle_AdjustLayout` 结束后, 从 hEle 取 HWND → `XC_hWindowFromHWnd` → `XWnd_Redraw(hw, false)` 整窗重绘(与 Alert 的 reflow 同口径, 见 90)。显示与隐藏错误区都要走这条链; 若只做布局不做整窗重绘, 重影会一直留到下一次全窗重绘。

103. **`XEle_SetPadding` 可用于布局元素; 用带外边距的子容器当 padding 会让 Fill 行超宽**(`components/item.go`, 元素矩形实测):
    - **现象**: Item 原结构是 root → pad(四边 margin 16) → body(Fill 宽)。pad 实际宽 768(16..784), 但 body 仍按 root 宽 800 排版(相对 pad 0..800) —— 行尾动作区被推到卡片右侧外裁掉, 右内边距消失。
    - **根因**: 子元素的 Fill 宽度按外层布局宽算, 不受中间 pad 自身外边距缩窄。
    - **修法**: 用 `xc.XEle_SetPadding(hEle, l, t, r, b)` 直接给根元素内边距(实测: 500 宽的 LayoutEle 设 16 后, Fill 子项宽 468); 不再套带 margin 的 pad。此 API 之前只有"存在"记录, 现已验证可替代"子项外边距当 padding"。

104. **横向 Item 的 Content 不能靠 Weight 吃剩余宽度, 要算成 Fixed**(`components/item.go`):
    - **现象**: body(横排)里 Media(Fixed) + Content(Weight 1) + 动作按钮时, Content 里的 Fill 标题/说明宽度恒为 0; Content 改成 Fixed 后正常。若把动作放进 Auto 宽的子容器, 子容器不会量测原生 Button 宽度, 会塌成 0、按钮被裁到卡片外。
    - **根因**: 101(嵌套 Weight 不把宽度下发给 Fill 子项)+ Auto 宽容器只量测声明了布局尺寸的子项。
    - **修法**: 动作元素直接挂 body 行(不套 Auto 子容器); 在 body 的 Size 事件里按"body 宽 - 可见 Media 宽 - 可见动作宽 - 间距"算出 Content 宽, 设 Fixed 后 `body.AdjustLayout()` + `content.AdjustLayout()`。运行中增删动作后, 下一次尺寸变化会重算。

105. **Label 按 `TextWidth` 精确设宽会被省略号截断**(breadcrumb 截图 + 对照探针):
    - **现象**: `w := xui.TextWidth(text, font)` 后 `NewLabel(..., w, 20, text, ...)`, "首页"(w=30) 显示 "首.."; 宽 38 才完整, "面包屑"(w=45) 同理。
    - **根因**: Label 的载体是原生 Button, 文本区有内部左右内边距; `TextWidth` 只量字形宽度, 不含这段内边距。
    - **修法**: 需要"按文字精确宽度"的小元素不要用 Label, 用透明 Element + `Painter.Text` 自绘(Breadcrumb 即此做法); 必须用 Label 时给量测宽加余量(实测再加 8 逻辑像素即可)。

106. **连续动画: 共享宿主窗口定时器 + 局部矩形重绘 + 只在看不见时停表**(`anim.go`, Spinner/Skeleton, CPU/GPU 实测):
    - **现象**: 首页放 5 个 Spinner + 4 个 Skeleton 后卡顿、DWM 持续占 10%+ GPU; 切走页签/最小化/滚出视口后仍不回落。
    - **根因**: ① 每个动画元素各起一个 60fps 的 `XEle_SetXCTimer`, N 个动画 = N 份定时器回调; ② 逐元素 `XEle_Redraw` 每次都会触达窗口合成, 多个元素比"一次局部重绘"更贵; ③ DWM 对**每帧整窗重新合成**收费, 大窗口 60fps 的连续动画即使应用侧只花 0.5% GPU, DWM 也会占 10%+(窗口样式/阴影不是主因, 普通窗与阴影窗几乎一样); ④ 页签隐藏后元素自身的 Show 标志不一定是 false, 定时器继续空转。
    - **修法**: `Spin` / `Pulse` 传元素句柄时, 定时器挂到**宿主窗口**(`XWidget_GetHWND` + `XC_hWindowFromHWND`), 键 = 元素句柄(Spin/Pulse 用最低位区分), 同一窗口的所有动画共用一个定时器; 连续动画保持 60fps(试过 30fps: DWM 仍然 ~10%, 只是旋转变卡, 已回退); 每帧用 `XEle_GetWndClientRect`(已含 ScrollArea 滚动偏移, 实测滚 1200 后 y 1500→300)算出动画元素矩形, 外扩后相近/重叠的合并成组, 每组一次 `XWnd_RedrawRect`(超过 4 组退回总并集 —— 多次局部重绘每次都有固定合成成本, 实测 9 块 11.5%, 1 块 7.5%); 每 tick 沿祖先链 `XWidget_IsShow`、窗口 `IsWindowVisible/IsIconic` 与视口矩形相交检查可见性, 隐藏/最小化/滚出视口的动画摘掉并停表。**失焦不暂停**: 曾按"非前台只跳过重绘、保留定时器与相位"做过前台暂停(省 DWM), 但在多显示器下窗口可见只是没焦点(放在副屏、用户在主屏操作)时加载圈会冻在半空, 被当成"卡死" —— 现在只要可见就照常 60fps, 开销与前台一致(单窗口动画 app GPU ≈0.5%、DWM ≈10~14%)。
    - **验收**: 2560×1528 前台动画: app CPU ≈0.5s/3s、app GPU ≈0.5%, DWM ≈10~14%(平台合成最大化窗口的固有成本; 30fps 实测仍在 ~10%, 降帧没有意义); 后台/最小化/滚出视口时 DWM ≈0.03%、app GPU 0; 切回前台 Spinner/Skeleton 自动恢复。不要再用"降帧率"换 DWM, 那只损失流畅度。



107. **不透明的圆角控件不要先铺方底擦色**(`components/separator.go` 的 Badge, 暗色 ItemMuted 截图):
    - **现象**: 暗色主题下 ItemMuted 里的 "成功" 徽章背后露出一块比 item 底色更深的矩形。
    - **根因**: Badge 为了擦掉透明元素上一帧的悬停痕迹, 每帧先铺一层 `t.Card` **方底**; default / secondary 本身是不透明圆角胶囊, 已经覆盖了整个胶囊, 这层方底纯属多余, 而 Card 色与 ItemMuted(混合底)不一致, 四角就露出来。
    - **修法**: 只给需要擦底的变体(outline / ghost / link)铺底, 且用**圆角底**(radius)而不是方底; default / secondary / destructive 直接画不透明胶囊(destructive 的 10%/20% 底色也预混成 over-Card 的不透明色); 不透明变体的 hover 色同样预混, 否则半透明 hover 会叠在上一帧实色上偏深。透明元素不擦底的根因见 28。

108. **原生 Check 按钮不处理键盘激活**(`components/toggle.go`, 探针实测):
    - **现象**: `Button_Type_Check` 获得焦点后按空格/回车, 勾选状态不变; 元素自己的 `AddEvent_KeyDown` 能收到 `VK_SPACE`(32) / `VK_RETURN`(13), 窗口级 KeyDown 也能收到。
    - **根因**: xcgui 的 Check 类型只在鼠标点击时翻转状态, 键盘消息没有浏览器的 `<button>` 激活路径。
    - **修法**: 组件在元素 `AddEvent_KeyDown` 里自己翻转 `SetCheck` 并触发与鼠标点击相同的回调 —— 不能只 `SetCheck`, 否则组的互斥与 `OnChange` 会漏。回调由组件持有(鼠标的 `XE_BUTTON_CHECK` 与键盘路径统一走 notify), 组监听同一入口。

109. **Label 载体的默认上下 border 会裁掉 g/p 的下伸部**(`components/label.go`, CardDescription "ToggleGroup:" 截图):
    - **现象**: 卡片说明里的拉丁字母 g 尾巴被切平; 探针里 20 高标签的墨迹比 24 高的少 2 行(文字带 22 vs 24 物理像素)。
    - **根因**: Label 用原生 Button 画单行文字, 而 Button 默认四边 `border=1`, 文字区 = 元素矩形上下各内缩 1px; 20 高的单行标签扣掉后装不下 14px 字体的完整行盒(墨迹只占 ~24 物理像素, 行盒更高), 底部被裁。
    - **修法**: Label 构造时 `SetBorderSize(1, 0, 1, 0)` —— 只清上下内缩, 左右保留 1px 维持原有水平内缩; **不增加任何布局高度**。探针实测 20 高标签的墨迹行数/像素数与 24 高完全一致。
    - **复核(150% DPI, 18px "设置")**: 宽度 38/40/42 出省略号, 44 起完整; 有/无 `SetBorderSize(1,0,1,0)` 两组的截断阈值**完全相同** —— 左右 1px border 不会让省略号提前。不要因为"看起来截断"删掉这行(删了 g/p 的下伸部又会被裁); 宽度不够时按第 105 条给量测宽加余量。

110. **`XEdit_SetTextAlign` 是生效的; 用它给文字"顶居中"的反而是 border 技巧**(`components/inputotp.go`, 探针实测):
    - **现象**: OTP 单格里的数字看起来"偏左 + 右半截被切掉"。曾经的结论是"`XEdit_SetTextAlign` 对程序 SetText 不生效", 于是改成按字宽给 Edit 设左右 border 把文字挤到中间 —— 切字正是这个改法造成的。
    - **根因(两条, 分开看)**:
      ① **对齐本身没问题**: 单行 Edit 上 `SetTextAlign(Center|Center_V)` 对程序 `SetText` 同样生效, 与"先设对齐还是先设文本、是否透明底/无边框/自绘 paint"都无关(五种组合逐一探针, 数字都在格心)。只给水平位(0x2)时垂直位被清成 Top, 字会贴顶甚至被裁 —— 说明垂直位也是**在生效**。
      ② **border 技巧 = 自造裁剪**: `SetBorderSize(left, 0, right, 0)` 把文本区压到正好一个字宽, 而 `xui.TextWidth` 量的是"显示矩形", 与字形实际步进差一两个像素; 文本区比字形窄, 右半截就被 Edit 自己的裁剪切掉, 同时整体位置又被 border 推到偏左。
    - **修法**: 直接用 `SetTextAlign(Center|Center_V)`, 文本区保持 `SetBorderSize(0,0,0,0)`。唯一的 border 用法是**空格子**的 caret: 空文本时原生 caret 贴在文本区左缘(实测 x≈3), 对称内缩半格后落到格心(实测 14/14 → x≈17), 有字时必须清零(否则裁字)。要用"按位置对齐"的思路前先用探针确认量测口径, 别拿裁剪当居中。

111. **同一窗口里多个高度过渡, 每帧会画出"半新半旧"的中间态**(`components/collapsible.go`, 手风琴展开逐帧探针):
    - **现象**: 手风琴点开一项(同时收起另一项)时, 下面的条目上下抖动, 展开动画结束那一帧还整体跳一下。
    - **根因(两条)**:
      ① **动画归属**取的是各自根元素 → 每条过渡一条定时器。两条定时器每帧先后到点, 每次 apply 都 relayout + **立即整窗重绘** —— 一帧里画了两次中间态(一个已经缩了、另一个还没长), 相邻两帧的兄弟项位置来回跳。
      ② **量测口径不一致**: 布局对 `Auto` 高度的父级**不计子项外边距**, 而内容区被动画锁成 `Fixed(h)` 时那段外边距又确实占位。于是"动画目标 34 / 终态 Auto 24", 动画结束恢复 `Auto` 的那一帧整条列表向上跳 10px(手风琴内容区的 pb-2.5 就踩到了)。
    - **修法**: ① 过渡归属改成**宿主窗口**(`XWidget_GetHWINDOW`), 同一窗口的所有高度过渡共用一条定时器, 本帧所有 apply 跑完只让运行时重绘一次; `applyHeight` 里**不要**再自己 `redrawNow`(终态由 `done` 重绘)。② 内边距不要用子项外边距, 改成内容区里垫一个透明占位元素, 让"动画目标"与"终态 Auto"量到同一个数。占位有三个坑: 第一, 要同时 `EnableBkTransparent(true)` **和 `EnableDrawBorder(false)`** —— 前者只清背景, 原生 1px 边框还在, 内容区会多出一个整宽的细矩形(看着像多了一条分隔线); 第二, **挂载时机必须是"用户内容都加完"之后**(在 `ContentStack` 里挂会排到第一个子项之前 —— 触发区和正文之间空一大截、正文紧贴分隔线); 第三, 占位高度要**扣掉内容区的行间距**(布局在相邻子项之间也会插 `SetSpace` 那么宽, 不扣的话下边距比上边距宽出一个间距)。

112. **真彩/暗色里"整块被填成灰色"的焦点环, 十有八九是漏了"内芯重新铺底"**(`components/accordion.go`, 手风琴触发区):
    - **现象**: Tab 聚焦触发区时, 整条 40px 高的行变成一块灰色圆角矩形, 看起来像"选中态"而不是 focus 环。
    - **根因**: 环带画法是"整块铺环色 → 内芯再铺回去", 只做了第一步。xcgui 会把绘制裁剪在元素自身客户区里, 环带只能画在元素内部, 所以内芯必须显式用宿主底色盖一次(= Button 的 `focus-visible` 画法)。
    - **修法**: 照抄 Button 的顺序: `FillRoundCorners(整块, 环色)` → `FillRoundCorners(内缩 3px, 宿主底色)` → `StrokeRoundCorners(内缩 3px, 环色, 1)`。同理, 圆角要按"元素自身属于组的哪一端"决定(见 113), 否则首格的环是方角。
    - **环画在元素内, 内容就要让位**: 上游的 `ring-3` 画在框外, 文字天然不会被压; 我们画在元素内, 环带必然吃掉 3px 内容区 —— 所以**有环带的元素, 内容要给环带让位**(手风琴触发区: 文字左留 `accordionPadX = 4`, 上游本无水平内距, 这是内嵌环的必要代价)。**让位要按"墨迹"算, 别按绘制块算**: 箭头这类图标管线的墨迹在 24 网格内本来就有内缩(chevron 横向只占 6..18), 绘制块贴边时墨迹离环带还有 ~2px, 与文字观感一致 —— 再给绘制块加内距, 右侧就会空出一截(实测探针逐像素量过)。环带内**别再描一道内缘线**, 那会让文字离环更近。

113. **分段拼接的边框(多个格子共享边)必须按"组"算圆角, 且擦边只擦中段**(`components/inputotp.go`, 验证码格子):
    - **现象**: ① 第二组的第一格没有左边框、也没有圆角; ② 每格顶/底横线在有竖线的位置断开 1px(像被咬了一口); ③ 聚焦时最左格的环带是方角。
    - **根因**: ① `first:rounded-l-lg` 语义只按 `i==0` 判断 —— 而 shadcn 的验证码是**若干 InputOTPGroup**(分隔符切组), 组的首末格各自圆角、各自画外侧边; ② "中段格擦掉自己左边线"的实现把整列(含上下横线所在行)都擦了, 横线就在那里断了; ③ 环带用了 `UniformCorners(0)`。
    - **修法**: 用分隔符位置算出 `groupStart/groupEnd`, 圆角与外侧边都按它来; 擦左边线只擦 `Top+1 … Bottom-1`(上下两行留给本格的横线); 环带/内芯描边复用同一套 corners(`corners.Inset(3)`)。**铺底也一样要圆角**: 环带内芯、格的底色如果 `FillRound(rc, 色, 0)`(半径 0), 方角会从圆角环带里露出一个白方块 —— 凡是"盖在圆角形状上"的填充都必须带同一套 corners。

114. **`Enable(false)` 挡不住鼠标聚焦原生 Edit**(`components/inputotp.go`, 禁用态验证码):
    - **现象**: 禁用组的格子里点一下, 光标照样出现并闪烁 —— 看起来"能输入"(虽然按键确实无效), 与浏览器的 `disabled` 语义不符。
    - **根因**: xcgui 的 `XEle_Enable` 只关掉输入/交互通路, 鼠标按下仍会走焦点流程; 原生 Win32 的 disabled EDIT 是完全不可聚焦的。自绘的焦点环受 `!disabled` 保护, 所以只剩一个"裸光标"暴露了状态不对。
    - **修法**: 禁用时额外 `XEle_EnableFocus(false)`(启用时再打开), 并把原生 caret 颜色设成格底兜底; `Focus()` 这类程序接口也要在禁用时直接返回。

115. **隐藏窗口里量不出布局的 Auto 高, 要"离屏显示一次"再量**(`components/popup.go` 的 `popoverPanel.MeasureContent`, Popover/HoverCard):
    - **现象**: 弹层内容区用 LayoutEle 自动高度, 隐藏状态下 `AdjustLayout()` 后 `GetHeight()` 恒为 0, `XEle_GetContentSize` 也返回 0 —— 弹层高度只能靠手算。
    - **根因**: xcgui 的布局要等元素真正可见后才产出尺寸; 离屏隐藏的窗口里 Auto 高不会计算(探针: 3×40 + 2×10 的内容, 隐藏时 0, 显示后 140)。
    - **修法**: 把弹层窗口在 `-30000,-30000` 处 `SWP_SHOWWINDOW|SWP_NOACTIVATE` 显示, 调内容宿主的 `AdjustLayout()`, 读 `content.GetHeight()`, 再 `resize` 卡片并隐藏; 内容宿主必须是 LayoutEle 且内容根用 `LayoutItem_SetHeight(Auto)`, 普通 Element 没有布局能力。

116. **不抢焦点的悬停弹层不能走 KillFocus 关闭**(`components/hovercard.go`):
    - **现象**: HoverCard 如果按普通弹层的做法 `SetForegroundWindow` 拿焦点, 会把用户正在输入的输入框焦点抢走(光标消失); 而不拿焦点时又收不到 KillFocus, 点外部不会自动关。
    - **修法**: 窗口创建时带 `WS_EX_NOACTIVATE`, 显示用 `ShowWindow(SW_SHOWNOACTIVATE)`(原生 `SW_SHOW` 会激活), 跳过 `SetForegroundWindow`; 关闭改由触发区/内容的 `MouseLeave` + 关闭计时器判断, 计时器到点时用 `GetCursorPos` 对照触发区与弹层窗口矩形, 指针仍在里面就续期(鼠标从子元素移到卡片留白不会再有 MouseLeave)。

117. **圆形头像图片: RGBA 蒙版 + `XDraw_ImageEx` 在 D2D 下可用**(`components/avatar.go`):
    - **做法**: `XImage_LoadFromDataRGBA(data, w, h)` 接收非预乘 RGBA(内部转 BGRA), `p.Draw.XDraw_ImageEx(h, x, y, w, h)` 直接缩放绘制; 设备重建/元素销毁时用 `XImage_Destroy` 释放。
    - **圆形裁剪**: xcgui 的 D2D 层/几何裁剪没有 Go 封装, 不要指望在绘制回调里 clip 成圆; 在 CPU 侧按 4× 逻辑尺寸做居中裁剪 + 双线性缩放 + 抗锯齿圆形 alpha, 高 DPI 下边缘依然平滑。
    - **注意**: `color.Color.RGBA()` 返回的是预乘值, 半透明 PNG 直接塞进去会发黑; 先反预乘成直通 alpha 再写缓冲区。

118. **长页面滚到底后, 深层嵌套卡片的命中区与绘制位置可能不同步**(demo 基础2 页实测, 根因未完全定位):
    - **现象**: 一行 `row → col → Card(内容区) → HStack → Button` 的卡片位于长页面最底部, 滚到底后按钮画在这里、命中区却整体低约 20~30 逻辑像素 —— 鼠标悬停命中的是按钮下方那一行标签, 点击按钮无反应; 窗口最大化/还原触发一次完整重排后恢复正常, 把同一行移到页面顶部也正常。
    - **现状**: 只在"页面内容很长 + 滚到底 + 多层 Auto 布局卡片"的组合里复现; 同一页面上方的 Accordion 按钮命中正常, 最小探针(单页 ScrollArea + 同样的 row/col/Card 结构)也未复现, 因此暂未定位到具体是哪一层布局缓存没有刷新。
    - **规避**: 交互密集的新组件卡片放在页首/少滚动的位置; 必须放在长页面底部时, 优先减少 `row/col` 嵌套, 或改放窗口最大化/重排后使用。本库 demo 的弹出层/头像/命令面板卡都放在对应页首。

119. **`XWnd_SetWindowPos` 的位置是物理像素, 尺寸是逻辑像素**(`popup.go` 的 `popoverPanel.resize`, Popover/HoverCard 用户报障):
    - **现象**: 自动高度弹层的卡片只占窗口的 2/3, 右侧/下方多出一大片"阴影"; 弹层窗口实测 671×356 物理, 而卡片只有 288 逻辑宽, 预期窗口应约 447 物理。
    - **根因**: `resize` 里把窗口尺寸按 `cw * DPI/96` 算成物理值再传给 `SetWindowPos`, 而 xcgui 的 `XWnd_SetWindowPos` 对 **cx/cy** 会再做一次 DPI 适配 —— 尺寸被缩放了两次; 同一个调用里的 **x/y** 却是按物理像素直接使用的(位置没有二次缩放)。
    - **修法**: 位置仍自己乘 `DPI/96` 后传物理值; 尺寸直接传**逻辑像素**(`cw + shadowBand*2`), 不再乘缩放比。弹层窗口 = 卡片 + 两侧阴影带, 与可见卡片严格贴合。

120. **窗口内的侧边面板滑入 = 平移面板及其直接子元素**(`components/sheet.go`, Sheet/Drawer):
    - **背景**: Dialog 的面板背景画在遮罩元素的 paint 里, 标题/内容/页脚都是遮罩的兄弟元素, 没有"面板容器"可以整体移动; 元素 alpha 也不会级联(第八节 47)。
    - **做法**: 记录 `slideX/slideY` 偏移; `layoutSheet` 先按最终位置排好, 再把面板矩形与**第一层子元素**(标题/描述/内容容器/关闭钮/页脚按钮/抓手)逐个 `XEle_GetRect` + `XEle_SetRect` 平移。内容容器里的控件相对容器定位, 自动跟随, 不需要遍历。遮罩自身的元素 alpha 只负责淡入淡出, 面板保持不透明。
    - **注意**: `XWnd_GetRect` / `XEle_GetRect` / `XEle_SetRect` 都是**逻辑像素**, 与代码里其它坐标同口径; 只有 Win32 `GetWindowRect` 是物理像素(见 125)。拖拽/动画期间每帧只移动偏移量, 不要重跑 `layout`(量高依赖可见布局, 会抖)。

121. **拖拽关闭不能清掉主 presence 过渡**(`components/drawer.go`):
    - **现象**: 抓手拖拽关闭抽屉后, 抽屉消失了, 但之后整页**点不动** —— 页面还在, 所有点击都被一个看不见的元素吞掉。
    - **根因**: 起拖时调了 `xui.DropTransitions(hWnd)`(想防止滑入动画和拖拽抢偏移), 它把主 presence 条目也删了; 松手后 `Presence(..., visible=false)` 对一个"从未活跃过"的新键返回 0, 既不跑过渡也不回调 `done` —— `finishSheetHide` 永远不执行, 遮罩停在 alpha=0 但元素仍可见(透明遮罩吞点击)。
    - **修法**: 起拖不要清过渡; 滑入的 apply 里用 `d.dragging` 跳过(拖拽优先)。回弹用 `NextPresenceKey()` 分配临时键做"当前偏移 → 0"的短过渡, 不碰主键。关闭动画仍用主键的 1→0, `done` 里统一 `finishSheetHide`。

122. **Menubar 不能用原生菜单做弹层**(`components/menubar.go`):
    - **原因**: `XMenu_Popup` 是**阻塞**的(菜单关闭后才返回), 期间主窗口收不到鼠标移动 —— 而 radix menubar 的核心语义是"打开一个菜单后, 鼠标移到另一个触发区即切换"。原生菜单条(MenuBar)也不提供托管菜单的能力(示例里仍是按钮点击后自己 Popup)。
    - **做法**: 复用 `popoverPanel` 的阴影窗 + 定位/工作区夹取, 弹层内容是一块自绘 Element: 行高 32, 悬停 bg-accent 走 `xui.HoverItem`, 勾选/单选指示器在左(pl-8), 快捷键右对齐; 条目模型与原生 Menu 共用 `menu_model.go`(`MenuOpt` 构造器原样可用)。

123. **子菜单必须是激活弹层, 父层要在焦点转移期间"免疫失焦"**(`menubar.go` + `popup.go` 的 `keepOnBlur`):
    - **现象**: 点子菜单条目没有任何回调, 菜单却整体关掉了。
    - **根因**: 子菜单最初做成非激活窗口(`WS_EX_NOACTIVATE`)。点击子菜单时焦点仍归父弹层, 父弹层随后的失焦处理(或点击激活主窗口)会把父层 `Hide()` 掉 —— 子菜单窗口在 `LButtonUp` 到达前就被隐藏, 点击事件丢失。
    - **修法**: 子菜单也用激活弹层(会拿焦点); 打开前把父弹层 `keepOnBlur=true`(KillFocus 处理跳过关闭); 子菜单关闭时按**前台窗口**判断: 焦点回到父弹层 → 只清子菜单引用, 焦点去了别的窗口 → 整组菜单关闭。程序化收起用 `hiding` 标记区分, 避免误关。另外父/子弹层各挂 `MouseLeave`: 指针既不在自己也不在另一层矩形内时才收起子菜单(用 `GetCursorPos` 对窗口矩形, 同 116)。

124. **普通 Element 在布局里的 Auto 宽会解析成 0**(`menubar.go` 根条):
    - **现象**: 菜单栏的触发钮都排好了, 但整条底色/边框完全不显示, 布局里却留出了高度。
    - **根因**: 根条是普通 `widget.Element`(没有布局能力), `LayoutItem_SetWidth(Auto)` 不会按子元素算宽, 得到 0 宽; 子按钮仍能画在父矩形之外(与本库其它"布局容器才有 Auto 宽"的结论一致)。
    - **修法**: 按触发钮实际宽度求和后写入 `SetWidth` + `LayoutItem_SetWidth(Fixed)`, 再让父布局 `AdjustLayout`。

125. **`XWnd_GetRect` 返回逻辑像素, `wapi.GetWindowRect` 才是物理像素**(`menubar.go` 弹层定位):
    - **现象**: 菜单弹层跑到触发区左边约 50 逻辑像素、还压住触发区上方。
    - **根因**: 定位时用 `XWnd_GetRect` 取窗口矩形后又按 DPI 除了一次 —— 它和 `XEle_GetWndClientRect` / `XEle_SetRect` 是同一套逻辑坐标, 不需要换算。
    - **修法**: `XWnd_GetRect` 的结果直接与元素 client 矩形相加; 只有 `wapi.GetWindowRect`(Win32, 物理)才乘/除 `DPI/96`。弹层屏幕位置本身按逻辑像素交给 `ShowAt`(它内部再乘缩放, 见 119)。

126. **原生 Edit 的滚动条子元素是"延迟创建"的, 整树 Show(true) 会在第二次显示时把它们点亮**(`components/input.go` + `components/dialog.go` 的 `showEleTree`, Sheet 输入框灰条):
    - **现象**: 单行输入框放在对话框/侧边面板这类"整棵子树统一显隐"的宿主里, **第一次打开正常, 第二次打开时输入框左上角多出一条带箭头的灰色短条**。
    - **实测**: 新建的单行 `XEdit` 带着横向与纵向两个滚动条子元素, 矩形是 `(0,0,100,12)` 与 `(0,0,12,100)` —— 起点在**左上角**, 所以点亮后画出来的不是底部/右部的条; 而且这两个子元素是**控件首次真正显示时才补建**的: 第一次 `XWidget_Show` 整棵树时它们还不存在(枚举不到), 于是逃过一劫; 关闭再打开, 子元素已经挂上父链, 就被整树 `Show(true)` 一起点亮了(探针: 第一次 hShow/vShow=false, 第二次=true)。
    - **修法(两处都要)**: ① 单行输入框构造时 `EnableAutoShowScrollBar(false)` + 两个滚动条 `Show(false)`(多行文本域仍要自动显示, 只在非多行分支处理); ② `showEleTree` 递归时**跳过 `XC_SCROLLBAR` 子树** —— 滚动条的显隐本来就由控件/皮肤自己管(同第八节 98 的 `SetSubtreeAlphaSkipScrollBars` 是同一个道理, 只是这里管的是 Show 标志而不是 alpha)。只做①不做②, 第二次打开照样翻车。

127. **相邻弹层之间不能留缝, 光标归属要按"窗口矩形"而不是卡片矩形判定**(`menubar.go` 子菜单):
    - **现象**: 子菜单从父菜单右侧弹出, 两块面板之间有一条几像素的缝; 鼠标从父菜单移向子菜单, 穿过缝的一刻子菜单直接关掉, 永远点不到子菜单条目。
    - **根因(两层)**: ① 定位把"重叠 4"写成了"间隙 4"(符号错), 两块面板之间真的有空档; ② 收起检查用卡片矩形判断光标在不在面板里 —— 而弹层窗口四周还有阴影带, **阴影带虽然透明却属于弹层窗口**, 光标一进去父面板就再也收不到 MouseMove 了, 此时按卡片矩形算"两边都不在", 于是误关。
    - **修法**: 子菜单与父面板**重叠** 4 像素(radix SubContent 的 `alignOffset: -4` 就是这个用意, 不是间隙); 光标归属判定按窗口矩形(卡片矩形向四周扩出阴影带宽)来做。两块面板在水平方向交叠后, 路径上不存在"谁都不接管"的空档。

128. **窗口直属的绝对定位浮层要用"布局项置位置"钉住, 才扛得住一切窗口级布局趟**(`components/toast.go` 的 `pin`, 探针实测):
    - **现象**: 通知显示中拖拽缩放窗口(或最大化/还原), 通知跑到窗口左侧。与 96 同源: 缩放与"页签换行→组件逐级重排到 `XWnd_AdjustLayout`"都会跑窗口级布局趟, 把窗口直属的绝对定位元素当布局项排到流式位置(左侧); 96 只堵了"切页别调窗口级布局"这一个触发点, 堵不住其余来源。
    - **不能走事件**: `XE_ADJUSTLAYOUT_END`(布局完成事件)在 1.4.3 里**注册即崩**(访问违例, 事件总线与 `XEle_RegEventC` 两条注册路径都崩), 没有"布局趟结束后补救"的钩子; 事后重摆(`AfterFunc` 延迟)又与组件自己的延迟重排之间存在排队竞态。
    - **修法**: 每次落位时调 `XWidget_LayoutItem_SetPosition(hEle, l, t, r, b)` 把元素**四边钉成相对窗口 Body 的 inset**。实测语义: ① 布局趟把钉住的项摆到钉位而不是流式位(缩放/重排后按新父尺寸重算, 四边都钉的项会被拉伸到四边之间 —— 所以 insets 必须与元素矩形一致, 窗口 Size 事件里要按新 Body 重钉); ② 钉在 `XEle_SetRect` 之后仍然存活(进出场动画逐帧 SetRect 不会清掉钉); ③ 只改钉不触发布局趟(显式 AdjustLayout 对"干净"布局是空操作), 所以钉住不干扰动画, 落位仍靠 SetRect。钉住后, 任何来源的窗口级布局趟反而会帮我们把浮层摆回正确位置。
    - **通用**: 任何"窗口直属 + 绝对定位 + 必须跟随窗口缩放"的浮层(Toast / HUD / 自绘遮罩)都适用这套"落位即重钉"; 元素级 `AdjustLayout_No` 照常使用。

129. **容器的内边距不能用子项外边距表达: Fill 子项会按父级全宽解析并溢出**(`components/card.go` 等, 探针 + 元素矩形实测):
    - **现象**: Fill 宽的卡片(卡片自己也被父布局排过)里, ContentStack 的子行 / Progress / 分隔线实际宽度 = **卡片全宽**, 右侧越过内边距 16px; 贴右缘的文字和行尾按钮被卡片边界裁掉。同样写法还在 CardFooter 内层、Alert 行容器、Empty Body、Command 分组、Popover/HoverCard 的 `LayoutContent` 里。
    - **根因**: xcgui 的布局项外边距只**平移并收缩容器自身矩形**, 容器给子项排版时仍按父级布局宽解析 Fill。480 宽 Fill 卡片的探针: ContentStack(四边 margin 16)自身宽 448, 它内部的 Fill 行却是 **480**; 再调几次 `AdjustLayout` 也不会自愈。
    - **修法**: 左右内边距改用根容器的元素 padding, 上下继续用布局项外边距(计入 Auto 高度):
      ```go
      l.LayoutItem_SetMargin(0, pad, 0, pad) // 上下: 计入卡片/弹层 Auto 高
      l.SetPadding(pad, 0, pad, 0)           // 左右: 约束子项 Fill 宽
      ```
      480 卡片实测: 内容区 480 宽, Fill 行 448 宽、左右各内缩 16。元素 padding 不改变容器自身的布局项尺寸(容器仍是 Fill 宽), 只约束子项排版。
    - **连带(嵌套 Weight 列)**: 容器改用元素 padding 后, 横向行里的 Weight 宽嵌套列会恢复"嵌套 Weight 不把宽度下发给 Fill 子项"(第 101 条) —— Alert 的标题/描述、CardHeader 的标题/说明、以及卡片内容区里 `Weight` 列中的 ItemGroup / Collapsible 都会变成 0 宽整块消失。修法见第 130 条; **应用代码在卡片内容区里使用 Weight 列时, 也要按第 101 条给 Fill 子项显式宽度**(通常用列尺寸事件把子项锁 Fixed)。
    - **ScrollArea 页面留白同理**: 内容区要左右留白时用 `body.SetPadding(pad,0,pad,0)` + 上下外边距, 不要四边都用 `LayoutItem_SetMargin`。

130. **横向行里 Weight 宽的嵌套列, 其 Fill 子项要由容器按实测宽锁 Fixed(并在尺寸变化时重新锁定)**(`components/alert.go` / `components/card.go`, 元素矩形实测 + 缩放复验):
    - **现象**: Alert 改用正确的元素 padding 后, 图标还在, 标题/描述整段消失; CardHeader 的"卡片/说明"以及卡片内容区里 Weight 列中的 ItemGroup / Collapsible 同样变成 0 宽。打印矩形: Weight 列 w=506, 内部 Fill 子项 w=0。
    - **根因**: 第 101 条 —— Weight 宽不会下发给嵌套布局的 Fill 子项; 此前行容器自身溢出时, 子项用的是外层全宽, 恰好把这个问题掩盖了。
    - **修法**: 容器在自己的尺寸事件里读取实测宽 w, 把 **Fill 直接子项**锁成 `LayoutItem_SetWidth(Fixed, w)`; 写回用 `AfterFunc` 延后, 避免在布局趟里改子项尺寸触发重入。
      - Alert: 在 reflow 量高时锁标题/描述(见第 90 条的重排链)。
      - CardHeader: `syncTitleWidth()` 监听标题列尺寸变化, 锁标题/说明(卡片组件内部处理, 调用方无需关心)。
      - 应用代码的 Weight 列: 同样在列 `AddEvent_Size` 里把 Fill 子项锁 Fixed; demo 的 `col` helper 就是这一套。
    - **⚠️ 锁过之后要记住哪些子项**: 第一次锁完, 子项的布局项宽度类型已经从 Fill 变成 Fixed; 如果下一次尺寸变化只按"类型 == Fill"筛选, 它们会被当成用户显式固定宽而跳过 —— 表现是**缩放窗口后子项停在旧宽, 行尾按钮被卡片裁掉**。必须用 `synced map[int]bool` 记录本组件锁过的子项, 后续尺寸变化按 `Fill || synced[child]` 继续更新(并清掉已销毁的句柄)。
    - **探针注意**: reflow 走 `AfterFunc`, 必须跑 `a.Run()` 消息循环; 只 sleep 不泵消息会一直看到宽度 0。

131. **宿主置顶(SetTop)时, 自建弹层窗口必须用 HWND_TOPMOST; 原生菜单/下拉不用改**(探针截图 A/B):
    - **自建弹层**: `popup.go` 的面板(Select / DatePicker / TimePicker / Popover / HoverCard / Menubar 子菜单)原先用 `HWND_TOP`, 主窗口 TOPMOST 时弹层整个被盖住(打开成功但看不见)。改 `HWND_TOPMOST` 后可见; `resize` 里带 `SWP_NOZORDER` 的一处保持原样。Tooltip 一直就是 TOPMOST。
    - **原生弹层**: `XMenu_Popup` 的菜单窗与 ComboBox 的下拉窗是**宿主拥有的 popup 窗口**, 实测在 TOPMOST 宿主上仍显示在宿主之上(下拉定位用 `SWP_NOZORDER` 也不影响) —— 不要给它们强加 TOPMOST。
    - **窗口内覆盖元素**: Dialog / Sheet / Drawer / AlertDialog / Toast 都是宿主窗口的子元素, 不涉及独立窗口 z 序, 置顶宿主下不受影响。

132. **原生 `XMenu_Popup` 在副屏(显示器原点非 0,0)上把"显示器矩形 × DPI"当边界钳制, 弹层严重错位; 落位要在 Popup_Wnd 回调里用 `SetRect` 自己算**(`ui/menu.go` 的 `menuPopupBody`, 三块屏探针实测):
    - **现象**: 窗口放在主屏右上方的副屏(矩形 y 为负)时, 下拉菜单弹到触发区的右上方几百像素外; 同一触发区在主屏上正常。
    - **根因(数值实锤)**: `XMenu_Popup` 收到物理锚点后, 会把它夹进"显示器**物理**矩形再乘 DPI 比例"得到的边界里 —— 150% 缩放下副屏左缘 2570 被当成 3855、上缘 857 被当成 1286。探针(窗口 2600,-500, 触发区物理 x=2618): 期望弹窗 x=2618, 实际 3855(=2570×1.5); 换 x=3018 仍是 3855(**钳制特征**: 输入怎么变都吸到同一值); x=4618 超过钳制线 → 正确 4619。右下副屏(正 y)同理: 期望 (2618,1024) → 实际 (3840,1286) = (2560×1.5, 857×1.5)。y 在钳制线以内时不受影响。主屏原点 (0,0) 乘 1.5 仍是 0, 所以只有副屏暴露; 反过来主屏上边界被放大到 1.5 倍, 贴屏幕右/下边的弹层会溢出也不夹。
    - **修法**: 顶级菜单(含右键菜单)的最终落位不再信原生 —— `Menu_PopupWnd` 回调里按锚点(物理)重算 Body 左上角: 默认贴锚点下边+4, 放不下翻到锚点上方(弹层下边 = 锚点上边 − 4, radix 的 side 翻转), 再夹进工作区(`xui.MonitorWorkArea`); 用 `SetRect` 落地(单位是逻辑像素, 窗口矩形连同阴影带一起写)。子菜单的 X 沿用原生(它锚父弹窗**实际矩形**右缘, 会跟随), Y 必须自己算(见 134)。
    - **为什么这样安全(探针验证)**: ① `SetRect` 大位移(1237px)不崩, 与阴影带外扩同一条路径; 崩的是 `SetWindowPos`, 别混用(见 64)。② 子菜单锚定父菜单**实际矩形**: 父窗挪到 (2819,-235) 后, 子菜单跟随到 (2973,-232)(= 父右缘 +1), 与不挪时行为一致。③ 命中判定跟随窗口: 挪过的菜单里悬停条目能展开子菜单、点击条目能关闭。④ 位置/夹取/上翻数值实测(150%): 副屏工作区 (2570,-743,5130,857), 锚点 (2870,-443) → Body (2870,-435)(= 锚点 +4 + 取整); 锚点 (5070,-343) → x 夹到 4867(= 5130 − 264 − band); 锚点 (2870,827) → 上翻到 683(= 827 − 4×1.5 − 138); 主屏 (0,0,2560,1528) 三组同样精确。中段位置的落位与原生的 ±1~2px 噪声一致, 主屏观感不变。
    - **注意**: 工作区按**弹窗自己的句柄**取, 不按宿主窗口 —— 宿主窗口横跨两块屏(病态布局)时 `MonitorFromWindow` 返回"占多数"的那块, 而锚点可能在另一块上, 实测会把弹层夹到另一块屏的顶边(y=857); 原生已经把弹窗放在锚点附近, 它落在哪块屏就是哪块屏。事件结构体 `pInfo` 里的 `HWindow` 是弹窗 xcgui 句柄, 记得 `xc.XWnd_GetHWND` 换 Win32 句柄再查显示器。

134. **原生子菜单的 Y 会把它的底边夹进"父弹窗矩形底边"的缓存值 —— 父弹窗被我们挪过就整体错位一整行**(`ui/menu.go` 的 `rowBodyTop`, 五组窗口位置 × 两块屏探针实测):
    - **现象**: 副屏上触发区贴近窗口底部, 下拉菜单翻到上方后, 悬停带子菜单的条目 —— 子菜单整体低一行(首项压在触发行的**下一行**上), 且低多少随触发区在窗口里的位置变化; 主屏上触发区靠上时正常。
    - **根因(数值实锤, 150% DPI)**: 原生算子菜单位置时, X 用父弹窗的**实际**矩形(跟随, 没问题), Y 先按"父弹窗顶 + 触发行偏移"算, 再把子菜单底边夹进**父弹窗创建时那个矩形**的底边 —— 也就是我们 `SetRect` **之前**的位置。探针三组同一菜单: ① 父弹窗未被挪(位移 −3 逻辑) → 子菜单顶 = 父顶 + 203, 正常; ② 位移 −19 → +200, 正常; ③ 位移 +93(父弹窗被向下校正) → 子菜单顶 = 父顶 + 139, 少 61 逻辑 = 正好被"缓存底边"截住。所以"给子菜单补一份父级位移"这种线性补偿不成立: 截断一旦发生, 位移与结果不再同比例。
    - **修法**: 子菜单 Y 一律自己算 —— `rowBodyTop(父项 id)`: 该条目在**同组**内的行偏移 = 前面每个条目 28(`SetItemHeight`) + 每条分隔栏 4(原生槽高), 子菜单 Body 顶 = 父弹窗 Body 顶 + 该偏移(父内边距 4 与子菜单内边距 4 相抵 = radix SubContent 的 `align=start, alignOffset=-4`, 首项正好压在触发行上)。分组弹层顶记在 `Menu.groupTop`(键 = 分组 id, 0 = 顶级, 其余 = Sub 触发项 id), 支持任意层嵌套。再夹进工作区(radix `avoidCollisions`)。X 保留原生。
    - **验证**: 五组配置(副屏不翻 / 小翻 / 大翻 + 主屏不翻 / 大翻)子菜单 Body 顶都 = 触发行顶 − 6 物理(= 4 逻辑, 即 alignOffset), 之前大翻那组是 −12 且低一行; 右键菜单的子菜单(锚点 = 光标点)同样齐平。
    - **注意**: `pInfo.NParentID` 是**父项 id**(不是父窗口句柄), 正好用来定位条目; 行偏移模型依赖 `SetItemHeight(28)` 与分隔栏槽高 4 这两处设置, 改它们要同步改 `menuRowH / menuSepH`。

135. **ComboBox 自绘下拉的"屏幕边界"不能写死主屏**(`ui/combobox.go` 的 `alignComboPopup`, 副屏实测):
    - **现象**: 窗口摆在主屏右上方的副屏(y 为负)上, 表单里的下拉框一点开, 弹层跑到几百物理像素之下, 完全盖不住触发区; 主屏上 item-aligned 正常。
    - **根因**: 边界取 `screenTop = 0` + `screenBottom = SM_CYSCREEN` —— 前者把"屏幕顶"当成虚拟屏原点(副屏上元素 y 本身是负的), item-aligned 算出的期望位置恒小于 0 → 恒被压到 y=0; 后者只有主屏高度, 底边同样夹错。
    - **修法**: 两端都换成 `xui.MonitorWorkArea(xc.XWnd_GetHWND(hWindow))`(见 132/133); 取不到时保留原兜底。顶边被夹时仍走原来的"滚动列表让选中项作为首项可见"。

133. **"上方放不下就翻到下方"的判断不能拿虚拟屏原点 (0,0) 当边界**(`ui/tooltip.go`, 负坐标副屏上 ToolTip 恒朝下):
    - **现象**: 窗口放在主屏正上方的副屏(y 为负)上, 标题栏按钮明明上面还有空间, ToolTip 却翻到下方压住页面内容; 主屏上正常。
    - **根因**: 翻转条件是 `y = 元素上边 − 气泡高; if y < 0 { 翻到下方 }` —— 副屏上元素的屏幕 y 本身是负的, 这个条件恒成立。
    - **修法**: 翻转与左右夹紧一律按**窗口所在显示器的工作区**(`xui.MonitorWorkArea`, 物理 → 除以 DPI 比例换成逻辑)判断: 上方放不下且下方放得下才翻; 最后把气泡夹进工作区(顺带扣掉任务栏)。取不到显示器信息时退回原"以 0 为界"的兜底。
    - **连带**: 该助手原先只在 `popup.go`(弹层底座)里, ToolTip / Menu 引用它时, copy-not-install 的**逐文件安装闭包**会缺文件 —— 已上移到运行时 `monitor.go` 的 `xui.MonitorWorkArea`(组件都依赖模块运行时, 不用改 registry)。

136. **`Presence` 幂等: 值已经等于目标值时不起过渡、也不会回调 done —— 靠 done 落终态的组件会"布局变了但屏幕没重绘"**(`ui/collapsible.go` 的 `startPresence` / `expand`, 手风琴收起后卡片文字重影):
    - **现象**: 手风琴点第一项收起, 卡片说明文字在旧位置留一整行没擦掉(像重影); 收起也没有 200ms 高度过渡, 直接跳到底。只在"从别的页签切进手风琴那一页后第一次收起"时出现, 第二次起正常, 主屏副屏都会。
    - **根因**: 建页时该页还是隐藏页, `expand()` 量内容自然高度得到 0 → 走"直接落 Auto"的早退分支, **这个键上从没建立过 presence 值**(停在 0 = 不存在)。第一次收起请求的 target 也是 0 → `transValue` 看到"值没变"→ 不起过渡、不开定时器; `Presence` 只立即执行一次 `apply`(高度一步打成 0 + 重排), 而**终态与整窗重绘都在 done 里** —— done 永远不会来。于是: 内容没隐藏(`Show(true)` + `Fixed(0)`), 窗口一次都没重绘, 屏幕上还是展开态那张旧帧; 只有触发区自己因为 `SetCheck + Redraw` 换了箭头, 看起来就是"别处没刷新"。元素往下移(旧位置在上方)时旧像素最显眼, 往上移时新帧正好盖住旧位置, 所以只在一种移动方向上被看见。
    - **修法**: 两处都补。① `startPresence` 接住 `Presence` 的返回值, 值已经在目标上就自己调一次终态(恢复 Auto / `applyClosed`)并 `redrawNow` —— 终态幂等, 之后真回调到 done 也只是再落一次; ② `expand()` 的早退分支顺手 `Presence(owner, key, true, 0, nil, nil)` 把值抬到 1, 让后续收起回到正常过渡。
    - **验证**: 探针(`XUI_ANIM_SCALE` 慢放)对比: 修前第一次收起 120ms 内就跳到底、内容元素 `show=true`、屏幕上展开态的文字一直留在原位(实测停留 20s 不消失); 修后收起逐帧缩、`show=false`、旧帧被整窗重绘擦净。demo 两条路径(启动即在基础4 / 从基础1 切进基础4)各连点 8 次, 像素级"同一行文字出现两遍"检测全 0。
    - **注意**: 同理, 任何用 `Presence` 且**终态写在 done 里**的组件(弹层 / 抽屉 / 通知)都不能假设 done 一定会来; 值可能一开始就在目标上。

---

137. **非激活弹层里的普通元素没有原生 hover 状态、也没有 MouseMove —— 链接悬停与"指针进入内容"都要自己轮询**(`ui/navigationmenu.go`):
    - **现象**: ① 悬停导航菜单内容里的链接行, 没有任何 bg-accent; ② 指针从触发区移进视口时, 150ms 的收起计时器照常到点, 菜单在指针已经停在内容上时被关掉(移动得慢一点就必现)。
    - **根因**: 视口复用 `popoverPanel` 的非激活形态(`WS_EX_NOACTIVATE` + `SW_SHOWNOACTIVATE`)。窗口没有激活时, xcgui 不维护视口里元素的 Stay/Down 状态(探针: 光标压在链接上, 元素 StateFlags=0x80000019, Stay 位不置), 元素级与窗口级 MouseMove 都收不到; 但 **`XWnd_GetStayEle(视口窗口)` 仍能返回光标下的元素**(滚动条显隐用的就是它)。
    - **修法**: 菜单打开期间起一个 30ms 观察器: `GetStayEle` 向上归到链接 → 记为 `hoverLink` 并重绘该行(`xui.Hover` 负责 150ms 过渡); 用"光标在视口窗口矩形内 或 在菜单触发区矩形内"判定 `pointerOver` —— 在就续期, 不在才启动 150ms 收起; 收起计时到点后再按光标位置续查(同 HoverCard 的做法), 不依赖子元素 MouseLeave。`everInside` 记录本次打开期间指针是否进过导航, 程序化打开且指针一直在外面时不会自动关。
    - **验证**: 探针把光标移到链接中心后 260ms 菜单仍开着、移开 150ms 后收起; 同一时刻链接 StateFlags 仍是 Leave(证明判定走的不是原生 hover)。demo 里链接行悬停出现 bg-accent, 从触发区移进内容不再被关。

138. **在 KeyDown 回调里同步 `XWnd_SetFocusEle` 会把同一次按键重新派发给新焦点, ←→ 一次移动两项**(`ui/navigationmenu.go` 的 `focusNeighbor`):
    - **现象**: 焦点在某个触发区上按 →, 一次跳到下下个; 按 ← 也同样跳两项(从第三项按 ← 直接回到第一项)。
    - **根因**: 在按键回调里同步搬焦点时, 当前按键还会被送给刚获得焦点的按钮, 新按钮的 KeyDown 再执行一次同样的移动(按键本身只按了一次)。
    - **修法**: 先用 `xui.NoteKeyboard()` 标记焦点来源(焦点环按键盘来源绘制), 再用 `AfterFunc(0)` 把 `SetFocusEle` 推到消息循环里执行 —— 当前按键已经处理完, 新按钮只等下一次按键。
    - **验证**: 实机/探针 ←→ 每次只移动一项。

139. **面板 `resize` 之后还有一次布局趟, 会把手动写的滑入位移原地复位**(`navigationmenu.go` 的 `animateContentIn`):
    - **现象**: 切换导航菜单的内容页时, 预期"从切换方向滑入 208px + 淡入", 实际内容直接出现在终点, 只有透明度在变。
    - **根因**: 切换前刚调过 `panel.resize`(视口宽随内容变化), 窗口收到 WM_SIZE 后 xcgui 会在随后的消息里再排一次布局; 我们在布局之前用 `XEle_SetRect` 写的横移被这次布局改回布局位置。
    - **修法**: 先把新内容 `SetSubtreeAlpha(0)` 压透明藏住, `AfterFunc(0)` 等布局落定后再取基准矩形(`XEle_GetRect`)、设偏移并起 `Presence`; 每帧 apply 重新写回偏移(万一之后还有布局趟也不会停在终点), done 时恢复基准矩形并把 alpha 打回 1。首次打开不播内容动画 —— 视口自己的淡入(元素 alpha)与内容 alpha 会互相覆盖, 每帧后写的赢。
    - **验证**: 探针直接读内容元素的矩形/alpha: 起始 left=213/alpha≈3, 中途 150/81, 结束 left=8/alpha=255(8 是内容区的左内边距)。

140. **元素鼠标事件不冒泡; 拖拽要"窗口级起手 + 捕获元素跟手"**(`carousel.go`, 事件路由探针实测):
    - **实测**: 指针在子元素上时, **父元素收不到** MouseMove/LButtonDown/LButtonUp —— 只有该子元素与**窗口**各收到一份(窗口级 pPt 是窗口客户区逻辑坐标, 元素级 pPt 是该元素客户区坐标)。所以"在视口里按下、在幻灯片上拖动"这种需求不能只监听视口自己的鼠标事件, 幻灯片内容一铺满就再也起不了手。
    - **修法**: 窗口级 LButtonDown/MouseMove/Up 起手(用 `XEle_GetWndClientRect` 判点是否落在视口内), 越过 6px 阈值后 `window.SetCaptureEle(viewport)`; 捕获后移动只送给该元素(窗口收不到), 坐标要加按下时记的视口原点; 在窗口外松手没有 UP 事件, 用 MouseMove 的 `nFlags & 1`(MK_LBUTTON)判定已松开并收尾; 收尾时 `SetCaptureEle(0)`。
    - **坑**: 同一次按下窗口和视口都会收到(元素级不冒泡但会各派发一份), 视口自己的 LButtonDown 处理器**不能无条件清拖拽预备状态** —— 它会在窗口级处理器刚置上之后又清掉, 表现为"永远拖不动"(探针实锤, 去掉该清理后正常)。

141. **循环轮播的折回要用"正模"**(`carousel.go` + `carousel_test.go`):
    - **现象**: 循环模式的上一张(往前拖/按上一张)时, 第一张一直不出现, 左侧留空。
    - **根因**: Go 的 `math.Mod` 符号跟被除数, `math.Mod(-1363, 2520)` 仍是 -1363; 直接减 `step` 会把第一张留在 -2203 的负区间。折回区间按 `(-step, (n−1)step]` 设计时负数必须自己归正。
    - **修法**: 先加一轮再取模(`x := math.Mod(x+step, total); if x < 0 { x += total }`)再减 `step`; 这样左侧留一格给"上一张从左边进来", 右侧留一格给"下一张从右边进来", 首尾相接且没有克隆页。`components/carousel_test.go` 固定了 index 0 / 末页 / 拖过末页三种位置。

142. **Button 的元素高 = 视觉高 + `pressShift`(按压位移预留), 容器要按"视觉高"居中, 不要按元素高**(`menubar.go` 的 `layoutTriggers`, 150% DPI 像素实测):
    - **现象**: 菜单栏一整排触发区看着偏上 —— 悬停/展开的底色块在条内上 4 / 下 6, 文字也跟着高 1 逻辑像素。
    - **根因**: `NewButton` 把创建高度加了 `pressShift`(1px, 见 88), 预留像素落在**底边**(壳体画在元素矩形去掉底部 1px 的位置)。布局按 `(条高 − 元素高)/2` 居中, 居中的是"壳体 + 底部预留"这个整体, 壳体自然被顶上去 1px; 菜单栏条高又按旧预留量(2px)写成 38, 比上游 `h-9`(36)高 2。
    - **修法**: 条高按上游 h-9 = p-1(4) + 视觉高 28 + p-1(4) = 36; 触发区按壳体高居中(`(mbBarH-(h-pressShift))/2`); 弹层的 sideOffset 也从**视觉底边**量, 不要拿含预留的元素底边(否则弹层低 1px)。
    - **验证**: 150% DPI 像素实测(探针 + demo 导航3页): 条 36、壳体 28、上下留白各约 4 逻辑; 四段文字墨迹中心与条中心差 0.5 物理像素(0.33 逻辑, 属字体墨迹不对称); 弹层卡片顶边 = 条底边 + 4 逻辑。

143. **Radio 的原生互斥不要逐项手动取消, 键盘搬焦点必须推迟到消息循环**(`radiogroup.go`, 探针实测):
    - **现象**: ①组内点选另一项后新旧两项都亮, 或者最后两个都不亮; ②按 ↓ 后选中项正确, 但焦点环落在**下一项**上。
    - **根因**: ①`Button_Type_Radio` 的互斥由原生按**同一父容器**处理; 在选中回调里再对同父兄弟调 `SetCheck(false)`, 原生会把刚选中的项一起取消(事件序列 A:true → A:false)。②在 `XE_KEYDOWN` 回调里同步 `XWnd_SetFocusEle` 会被同一次按键的原生焦点导航覆盖, 焦点落到再下一项(同 138 条)。
    - **修法**: 同父兄弟交给原生互斥, 只对**跨父级**条目(例如放进 Field 的选项)手动 `SetSelected(false)`; 键盘导航用 `app.AfterFunc(0, func(){ XWnd_SetFocusEle(...) })` 延迟搬焦点。
    - **验证**: 探针 + demo 表单5页: 点击 A/B/C 单选互斥; ↑/↓/Home/End 选中并搬焦点(循环); Space 选中; Field 嵌套项参与组互斥与方向键; 组级 `SetValue` / `SetDisabled` 正常。

144. **Slider 的"值数组 / 垂直方向 / 滑块热区"三条几何结论**(`slider.go`, 探针实测):
    - **值必须是数组**: radix 的 `value` 是 `number[]`, 长度决定滑块数量(2 个 = 范围选择)。单滑块时"已完成区间"从**值域下限的几何位置**画到滑块, 不能写死从左端 0 起 —— `inverted` 时下限在右端(垂直时在下端), 写 0 会把整条轨道点亮。多滑块时区间画首尾两个滑块之间(radix 的 Range 语义, 中间滑块不切段)。
    - **元素沿轨道方向比可见轨道多 18**: 两端各 9(滑块半径 6 + ring 3)是绘制余量, 元素自绘被裁在客户区内, 不留余量两端的 ring 会被切掉。水平 `cx`、垂直 `cy` 都是**可见轨道长度**, 元素尺寸 = 轨道 + 18; 垂直默认轨道 160(radix `min-h-40`), 值从下往上增长。
    - **滑块热区不是整条轨道**: shadcn 的光环挂在滑块上(`thumb:hover` + `after:-inset-2`), 所以 hover 判定按"滑块中心 ± (半径 + 2)"算, 指针在轨道其它位置不亮环。点击轨道时把**最近的**滑块跳到点击位置并开始拖; 按在滑块上则**不跳值**直接拖(否则轻微点击会让滑块跳动)。
    - **相邻滑块互不越过**: 拖动/设值时把目标滑块夹在 `values[i-1] + minStepsBetweenThumbs × 一格` 与 `values[i+1] - 同值` 之间; `step=0`(连续)时一格按值域千分之一算。键盘只能调整"当前操作滑块"(点击/拖动过的那一个), 因为整个滑动条是一个焦点元素 —— 不是 radix 的每 thumb 独立 Tab 焦点。
    - **键盘焦点要自己给**: 普通 Element(非按钮/编辑框)点击**不会**自动获得键盘焦点, `LButtonDown` 里必须 `SetFocus()`, 否则方向键永远收不到(实测: 不加时点击后按 → 无反应, 加上后三下 = +3 个步进)。
    - **点击跳转后光环要续上**: 点击轨道后滑块跟到指针下, 但 xcgui 不会补发 Move/Stay 事件, `hoverThumb` 还停在按下前的值 —— 松手时 `drag` 归零、`focus-visible` 又因指针来源为 false, 光环会闪一下就没。按下与松手时都要用 `thumbAt(指针)` 重算一次 `hoverThumb`, 才能对齐 shadcn 的 `thumb:hover`(指针还停在滑块上, 环就该在)。
    - **验证**: 探针断言双滑块互不越过(左滑到 55 停在 50, 右滑到 10 停在 60)、三滑块排序、垂直滑块点击位置换算、`inverted` 下 `geoFrac` 与方向键反向; `components/slider_test.go` 固定值域换算/吸附/间距; demo 实机点击跳到 48%、按 →×3 再 +15%、拖动右滑块 80 → 53 且标签同步。

145. **Progress 的 max / 不确定态 / 带标签形态**(`progress.go`):
    - **value 是绝对值, 不是百分比**: radix 的 `max` 可配(默认 100), 百分比 = value/max。组件内部保存绝对 value, `SetMax` 时 value 不变、按新上限重算百分比(与 radix 一致); 只存百分比会让改 max 时值跟着跳。
    - **不确定态(value=null)**: 轨道照常, 指示器是 30% 宽的短条沿轨道**三角波往返**(0→1→0), 相位由 `xui.Spin(hEle, true)` 驱动 —— 元素隐藏/滚出视口/窗口最小化时运行时自动停表, 组件不要自己起定时器。百分比文本显示 "…"。上游 shadcn 的 indeterminate 只暴露 `data-state`, 指示器会平移到轨道外(视觉上是空轨道); 这里用短条往返把"进行中但不知道进度"表达出来, 属观感增强。
    - **ProgressField(标题 + 百分比 + 条)**: 纵向布局容器, 高 32 = 标题行 20 + 间距 8 + 条 4。标题行是 `xui.HStack`(标题 Weight 1、数值固定 48 右对齐); 数值 Label 的右对齐直接改原生 Button 的旗标, **必须把构造时的 `NoWrap` + 省略号旗标一起带上**, 否则超宽会折行、把 20 的行高撑破。数值格式默认 `%.0f%%`(即 radix 的 `Math.round(value/max*100)`)。
    - **指示器颜色用解析函数**: `WithProgressIndicatorColorFunc(func(t *xui.Theme) xui.Color)`, 每次绘制实时取令牌; 传固定色才用 `WithProgressIndicatorColor`(见 3.2 组件禁止缓存颜色)。
    - **验证**: 探针 + demo 反馈2页 —— 24/64/100 与 150/200(max)三档、ProgressField 66%、不确定态逐帧短条位移、Chart2 指示器在亮/暗主题下的颜色。

146. **Switch 的空格键、禁用与 aria-invalid**(`switch.go`, 同 108/114 的规律):
    - **空格切换要自己补**: 载体 `Button_Type_Check` 收得到 KeyDown 但不改变选中(原生不处理空格)。键盘路径里手动 `SetCheck(!IsCheck())` 后**必须自己调监听器** —— 手动 `SetCheck` 不会触发 `XE_BUTTON_CHECK`, 只挂该事件会表现为"空格没反应/状态变了但回调没走"。鼠标路径仍由 `XE_BUTTON_CHECK` 驱动, 两条路共用同一个 `notifyChange`。
    - **禁用要关两层**: `Enable(false)` 只挡鼠标, `XEle_Enable` 不禁鼠标聚焦(见 114), 所以 `SetDisabled(true)` 里同时 `EnableFocus(false)`; 绘制按 `s.disabled || st.Disabled` 取 50% 透明。
    - **aria-invalid 画在轨道上**: 轨道本身是 `border border-transparent` 的胶囊, 红色描边必须描**轨道矩形**(不是元素矩形), 环是轨道外扩 3px 的 halo; 暗色 `border-destructive/50`、`ring-destructive/40`, 亮色 `/20`。与 focus-visible 同时成立时红色优先。
    - **验证**: 探针 + demo 表单2页 —— 空格切换触发 OnChange、禁用后点击/空格/聚焦全部无效、未选与选中两态的红边 + 常驻红环在亮/暗主题下均可分辨。

147. **Input Group: 内层输入框必须是"裸输入", 焦点环由容器画**(`inputgroup.go` + `input.go` 的 `WithInputBare`):
    - **症状**: 直接把 `ui.NewInput` 塞进一个画了边框的容器, 会看到双框(输入框自己一层 border-input + 容器一层), 聚焦时两个焦点环叠在一起。
    - **根因**: `Input` 的壳体/焦点环/校验环都画在元素自己的 Paint 里, 它不知道"外面还有一层容器"。
    - **修法**: `WithInputBare()` 关掉壳体与环(只留原生 Edit + 可选内嵌件), 容器统一画 `FillRound/Corners + Stroke + FocusFrame`。容器的 Paint 是**父级先画、子元素后画**, 子元素都是透明底、内容从 6~8px 起, 不会盖住边缘 3px 的环带 —— 不需要 `EnableEvent_XE_PAINT_END` 那种"最后画"的写法。
    - **焦点状态**: 组容器读 `xui.StateOf(input.Handle).Focused` 决定画不画环, 并在内层输入框上挂 `AddEvent_SetFocus / AddEvent_KillFocus` 让容器 `Redraw` —— 只在容器自己的 Paint 里读状态是不够的, 输入框聚焦不会自动让容器重绘。
    - **内边距联动**: 与 addon 相邻的一侧收成 6(pl-1.5/pr-1.5), 另一侧保持 12(px-3), 由容器调 `input.setBareInsets(l, r)` 更新(同包私有方法, 不必暴露给用户)。
    - **addon 宽度不能靠 Auto**: 横向主轴上嵌套 `LayoutEle` 的 `LayoutItem_SetWidth(Auto)` 实测算成 0, 整个 addon 消失。addon 自己累加内容宽(文字按 `xui.TextWidth`, 按钮/键帽取元素宽), 再用 `Fixed` 锁宽, 并在每次追加内容后让**父级** `AdjustLayout`(布局项声明只在自己身上重排不生效)。
    - **addon 左右顺序用 z 序调**: 布局顺序 = 子元素索引 = z 序。调用方通常先 `Input()` 再 `Addon(InlineStart)`, 后建的自然落在右边; 用 `xc.XEle_SetZOrder(addon, 0)` 挪到首位即可。⚠️ 不要用 `xc.XEle_InsertChild` 挪动**已布局**的子项 —— 实测会把消息循环卡死(探针只打印到调用前一行)。
    - **addon 按钮的字号/圆角用 InputGroupButton 自己的语义, 不是 Button 的尺寸档**: 上游基类是 `text-sm`, xs/sm 只改高度(24/32)与内边距, **不降字号**; 圆角 xs=`calc(--radius-5px)`、sm=`rounded-md`。直接用 `ButtonSizeXs` 会得到 12px 字号 + 10px 圆角, 与旁边输入文字不是一套(用户一眼看出"查询/发送"粗细不同)。`applyInputGroupButton` 里统一把 `fontPx` 设为主题正文、用 `Button.radiusMode` 覆写圆角; 按钮宽由 `FitWidth` 按新字号重量。
    - **addon 按钮的布局盒必须等于视觉高 24**: Button 的元素高是 24+`pressShift`(按下预留 1px)。把布局高压成 24 会把视觉壳压成 **23**, 并且按钮文字比同一行的 Label 高约 2 物理像素(用户报"发送太靠上")。修法(`fixButtonBox`): 布局高写 25、底边距 **-1**(布局盒 24、视觉壳 24、文字与行内文字对齐); 块级行再加 1px **底部内边距**, 让按下位移落在行内 —— 否则父级把壳体底部裁掉, 按下时底边不动、看起来像在变矮。行内 addon 有输入框撑高, 不需要这条 padding。
    - **同一行的说明文字也要走中文光学修正**: 按钮文字按 151 上移半像素, addon 文字若还用整型 `p.Text`, 块级行"说明文字 + 发送"的墨迹仍差 1 物理像素(表现为按钮偏高)。`InputGroupAddon.Text` 改用 `TextF` + `textVDy`; 但**只在 `addon.hasBtn`(里有文字按钮)时启用** —— 行内 addon 旁边通常是原生 `Edit`, 原生文字没有这个修正, 跟着按钮偏移反而会与输入框文字错开 1 物理像素(实测 "example.com" 前缀 + 同串占位符: 不偏移时顶边一致)。

148. **布局容器放进自建弹层时: 位置参数会被布局覆盖, 内边距只能用元素 padding + 子项锁 Fixed 宽**(`combobox_search.go` 的弹层):
    - **症状**: 弹层里的第一个子容器按 `NewLayoutEle(4, 4, w-8, h-8, content)` 建好, 期望四周留 4px 内边距; 实际内容贴边, 列表高亮条一直顶到弹层圆角上(四角露出直角)。
    - **根因**: `LayoutEle` 一旦作为布局子项参与排版, 位置/尺寸由父级布局算, 构造时传的 x/y 被覆盖; 只有宽高"可能"被布局项声明覆盖 —— 所以 `w-8` 也没生效。
    - **修法**: 容器自身 `SetPadding(4,4,4,4)` 表达内边距, 同时**里面每个子项都声明 `Fixed` 宽**(= 弹层宽 - 8)。不能给子项用 `Fill`: 元素 padding 不扣减 `Fill` 子项的解析宽度(见 129), 列表会按弹层全宽向右溢出。
    - **高度同理**: 子项容器高度要显式 `LayoutItem_SetHeight(Fixed, ch)`, 过滤后内容变矮时同步改高度 + `panel.resize()` + `Relocate()`, 否则弹层留着大片空白。
    - **过滤 resize 必须保持弹出方向**: 触发器贴近屏幕底部、弹层翻到上方后, 输入过滤让列表变矮; 若仍按 sideBottom 重算位置, 变矮的弹层"放得下"就会跳到触发器下方(视觉上整个飞走)。打开/每次 resize 后记下弹层实际在哪一侧(弹层底边 ≤ 触发区顶边即上方), resize 用记住的方向重算。
    - **多选勾选反馈要立即重绘**: 勾选状态画在弹层的行元素上, 点击切换后只重绘触发区的话, 行上的勾选标记要等下一次无关重绘才出现。切换后对全部行 `Redraw(true)` —— 行是弹层小窗里的独立元素, 立即重绘不影响宿主窗口的待重绘区域(触发区/文本仍用延迟重绘)。

149. **xcgui 会把 9pt 及以下的常规字重画成宋体, 从字体文件创建可绕开**(`font.go` / `fontfile.go`, 探针实测):
    - **现象**: 12px 的主题文字(徽章 text-xs、"打开文档"这类链接、按钮 xs)是宋体, 与 14px 正文的雅黑不一致; 13px/14px 及以上正常。
    - **根因**: `XFont_CreateEx` 在字号 < 10pt 时把**常规字重**换成宋体 —— 请求微软雅黑、Arial 还是 Consolas, 9pt 常规的渲染结果都一样; 10pt 起才按请求字族渲染。9pt **粗体**不受影响(仍是雅黑粗体)。阈值按磅算, 与 DPI 无关; 所以 12px(9pt)必中, 13px(10pt)起正常。
    - **修法**: `FontHandle` 对 `sizePt < 10 && !Bold && !Italic` 改走 `font.NewByFile`: GDI+ 取字族的**英文族名**(配置里的 "微软雅黑" 在字体表里登记为 "Microsoft YaHei"), 再查 Fonts 注册表(HKLM + HKCU)拿常规字重的字体文件, 按原 pt 创建。字号保持不变, shadcn 的 text-xs=12px 不用改尺寸。解析不到文件时回退 `font.NewEX`, 即接受 xcgui 的替换。
    - **两条限制**: 只认注册表表项名里 " & " 分隔的**第一段**(.ttc 以文件第 0 号字形为准: msyh.ttc 第 0 个是 Microsoft YaHei, Microsoft YaHei UI 在后面); HKCU 的字体值是绝对路径, HKLM 是 Fonts 目录下的文件名, 两种都要拼。
    - **顺带**: 进程里一旦从文件加载过 9pt 的雅黑(修复路径), 之后再 `font.NewEX("微软雅黑", 9)` 也会画成雅黑 —— 9pt 的槽位被替换掉了, 替换发生在渲染期, 与创建时的句柄无关; 显式 `font.NewEX("宋体", 9)` 不受影响(它本来就是宋体)。所以修好的进程里, 连没走 FontHandle 的 9pt 雅黑文字也会跟着正常。
    - **验收**: 150% DPI 截图, 12px 的 "打开文档" 与 14px 正文字形一致; `TextWidth` 量宽用的是同一个字体句柄, 布局不用改。

150. **按钮类文字的 Vcenter 不需要上移 1px —— 早期回补会让小字偏高**(`button.go` 与同族组件, 像素实测):
    - **现象**: xs(12px)/sm(13px) 按钮文字看起来不垂直居中, 整体偏高约 1 物理像素。
    - **根因**: 早期为补偿 "DrawText 按行高居中、字形偏下" 给文字矩形减了 1px。实测 xcgui 的 Vcenter 与浏览器 flex 居中落点一致: 150% DPI 下同尺寸 HTML 按钮(Chrome)的 "Xs"/"Sm" 墨迹上下留白相等; 带 1px 回补时墨迹中心比壳体中心高 2 物理像素(1.33 逻辑), 去掉后为 0 —— 回补本身就是偏差。
    - **修法**: `Button`(纯文字与行内图标两条路径)、`ButtonGroupText`、`Toggle`、`Select` 触发区与条目、`ComboboxSearch`、`NavigationMenu`、`Pagination`、`Accordion` 触发区的文字矩形不再 `Top-1/Bottom-1`; 含中文的文本另按 151 上移半像素。
    - **验收**: 150% 实测 "Xs"/"Sm"/"Default" 的墨迹中心与壳体中心差 0.0 / -0.5 / -1.0 物理像素(浏览器同尺寸 0 / 0 / -0.5); 与 149 的字体修复叠加时, 12px 文字不再同时"字体不对 + 位置偏高"。

151. **中文文本的 Vcenter 要上移半个逻辑像素(只调中文)**(`components/textopt.go` 的 `textVDy`, 150% DPI 实测):
    - **现象**: 中文按钮/胶囊(卡片页"了解详情 / 开始使用"、导航菜单"开始使用")文字看着偏下; 同尺寸拉丁标签(Xs/Default)却是正的。
    - **根因**: 同一套 Vcenter 下, 拉丁大写字母的墨迹中心在行盒中心上方约 1 物理像素, 中文字形几乎占满字身、墨迹中心在行盒中心下方约 1 物理像素; 14px 规格又被取整成 11pt(14.67px, 见 31), 中文再低一点。两者叠加, 中文低约 1.5 物理像素。
    - **修法**: `textVDy(text)` 检测到中文/全角字符返回 **0.5 逻辑像素**, 纯拉丁返回 0; 各组件用 `TextF` 的浮点矩形上移(整数矩形做不到半步)。覆盖 Button(两条路径与 link 下划线)、ButtonGroupText、Toggle、Select、ComboboxSearch、NavigationMenu、Pagination、Accordion、InputGroupAddon.Text(仅 addon 内有文字按钮时, 原因见 147)。**不要恢复全域 -1**: 拉丁文本会整体顶高(见 150)。
    - **验收**: 150% 实测卡片页"了解详情/开始使用"墨迹中心与壳体中心差 +0.5 物理像素(修前 +1.5; Chrome 同尺寸中文 +0.5), 导航菜单 0.0; 拉丁按钮维持 0 / -0.5 / -1.0。

152. **可聚焦元素(EnableSwitchFocus)的 ←/→ 会被框架当成"切换焦点", 键盘处理必须挂在窗口级**(`components/pagination.go`, 探针实测):
    - **现象**: 给分页元素 `EnableFocus(true)` + `EnableSwitchFocus(true)`(Tab 才能进入)后, 在元素 `AddEvent_KeyDown` 里处理 ←/→ 根本改不了内部焦点: 回调里 `XWnd_GetFocusEle` 已经**不是**自己了, 焦点跑到了下一个可聚焦控件上, 连按两下就把整块分页跳过去了; 回车也会被框架当成"激活"发给新焦点控件(实测把另一个分页翻了一页)。
    - **根因**: `XEle_EnableSwitchFocus(true)` 的语义是"接受通过键盘(方向键、Tab)切换焦点", 框架在**元素事件之前**就执行了方向键的焦点切换; 元素回调里再 `*pbHandled = true` 已经晚了。
    - **修法**: 键盘处理挂**窗口级** `Window.AddEvent_KeyDown`, 先判 `xc.XWnd_GetFocusEle(hWnd) == 自己` 再分发, 处理过的键置 `*pbHandled = true`。窗口事件先于元素派发(此时焦点还正确), 实测置 handled 后焦点保持不动、元素 KeyDown 也不再触发(不会双处理)。`EnableSwitchFocus` 保持 true 以便 Tab 能进来, Tab 遍历实测可停在分页上。原生焦点框用 `xc.XEle_EnableDrawFocus(hEle, false)` 关掉, 自己画 focus-visible 环(判定 `xui.StateOf(hEle).Focused && xui.FocusedByKeyboard()`)。
    - **验收**: 探针里 Tab 从普通按钮能走到分页; 焦点进入后 ←/→ 只在条目间移动(跳过省略号与到边界的导航项), Home/End 到首末, Enter/空格翻页, 按到端点不循环。

153. **分页的三个显示形态与"焦点环画在条目上"**(`components/pagination.go`):
    - 对齐 shadcn 文档的三个示例: 完整(`[上一页][1][2]…[n][下一页]`)、简洁(`WithPaginationSimple`, 只有页码)、仅导航(`WithPaginationNavOnly`, 只有上一页/下一页, 给"每页行数 + 分页"的数据表用)。上一页/下一页还可 `WithPaginationHideText` 只留 chevron(对应 pagination.tsx 里文字上的 `hidden sm:block`), 此时图标在整个链接里居中而不是贴边, 链接宽收回 32 方形(上游 default 按钮的 pl-1.5/pr-2.5 + size-4 图标)。
    - **隐藏文字必须连绘制一起跳过**: 只把量出的文字宽置 0 仍然会调用 `drawText`, 退化成一两像素宽的文字矩形会把"上一页/下一页"裁成一条竖线/一个冒号, 贴在 chevron 旁边(像素实测修前残留 1~3px 竖条)。
    - 形态切换只改条目序列(`sequence()`), 宽度与折叠规则不变; 单测覆盖三种形态的槽位稳定性(页码数量变化时整行宽度不能变, 否则同一坐标连点会点空, 见 53)。
    - **焦点环必须画在当前条目上**, 不能沿用 `p.FocusFrame`(它恒用元素整矩形): 分页是多链接并列, 整块环看不出是哪一项。做法同 Button —— 条目矩形先铺 3px 环带, 内芯再铺回卡面色, 本体视觉(底/描边/文字/图标)画在内芯上; 直接整块铺环色会盖掉当前页的 outline 描边。

154. **转圈图标切出加载态时要显式 `xui.Spin(hEle, false)`**(`components/toast.go`):
    - **现象**: 通知从"加载中"更新成成功/失败后, 卡片仍在被高频重绘(看起来像画面在闪/CPU 占用升高)。
    - **根因**: `xui.Spin(hOwner, true)` 是**一次登记、持续运行**的动画定时器登记; 后续绘制里不再调用它并不会停表, 定时器会一直按帧重绘这个元素(只有元素销毁/不可见时才被剪掉)。
    - **修法**: `Toast.Update` 里发现新状态不是 `ToastLoading` 就主动 `xui.Spin(handle, false)`; 销毁路径同样先停表再 `XEle_Destroy`。
    - 配套: `Toaster.Promise(fn, {Loading, Success, Error})` 的 `fn` 在独立 goroutine 跑, 结果经 `xc.Auto` 回 UI 线程更新同一条通知(`Toast`/`Toaster` 的方法不是线程安全的, 不要直接在 goroutine 里调); 用户函数 panic 用 recover 转成错误, 否则通知永远停在转圈上。关闭原因分 `OnAutoClose`(倒计时到点)与 `OnDismiss`(关闭钮/动作/Close/Clear/被挤出), `OnClose` 仍然两种都发。

155. **标签栏整块是一个 Tab 站点(roving focus), 单标签不能再各自可聚焦**(`components/tabs.go`, 探针实测):
    - **现象**: 给每个标签按钮 `EnableFocus(true)`(原生 Button 默认)时, Tab 键要在列表里停 N 次, 方向键也被框架当成"切换焦点"跳到别的控件 —— 与 radix Tabs 的"整个列表一个站点、内部方向键切标签"完全对不上。
    - **根因**: 同 152 —— `EnableSwitchFocus(true)` 的元素会把方向键先吃掉再派发给元素回调; 而 N 个可聚焦按钮就是 N 个 Tab 站点。
    - **修法**: 列表元素 `EnableFocus(true) + EnableSwitchFocus(true) + EnableDrawFocus(false)`, 每个标签按钮 `EnableFocus(false) + EnableSwitchFocus(false)`; 键盘处理挂窗口级(同 152), 先判 `XWnd_GetFocusEle == 列表句柄`。方向键在标签间移动"内部焦点"(不放框架), Home/End 首末, 端点不循环(radix 默认 `loop=false`), 跳过禁用标签; `activationMode` 分 automatic(移动即切换)与 manual(Enter/Space 才切换)。焦点环自己画在内部焦点所在标签上。
    - **禁用标签**: 用组件自己的 `Disabled` 标记 + `btn.Enable(false)`(原生 Enabled=false 会连点击一起挡掉), 绘制按宿主底色 50% 预混(同 DESIGN 61), 不要用元素 alpha —— 元素 alpha 是逐图元混合, 叠在列表底上会出现两层不同透明度。
    - **焦点环画"环带"不要整块填充**: 未选中的标签是透明的, 若用 `FillRound` 铺 3px 外扩的 halo, 内圈会从透明标签底下透出来, 表现为"焦点一进列表整块发灰"。改用外扩 2px + 3px 描边的 `StrokeRound`, 内芯保持原样。
    - **点标签要把焦点交给列表**: 标签不可聚焦后, 点它不会让列表拿到键盘焦点(方向键立刻失效)。在 `BnClick` 里 `xc.XWnd_SetFocusEle(hWnd, 列表句柄)`, 并把内部焦点移到被点的标签; 鼠标来源的焦点不画环(`xui.FocusedByKeyboard()` 为 false, 见 152)。
    - **换行布局**: 行内标签数上限 = 可用宽 / 最小标签宽(72), 行数取最少后把标签数平均到各行(13 个标签在 882 宽下 = 2 行 × 7, 下限宽 126 不会把标签压窄)。换行改变高度时要 `LayoutItem_SetHeight(Fixed, h)` 再逐级 `AdjustLayout`, 否则卡片/页面还按一行的高度排版。
    - **line 变体的下划线要贴"当前行"的底边**, 不能用列表底边: 换行后最后一行之外的标签, 下划线会跑到整块列表下方(视觉上像悬空)。按标签矩形 + `tabsPad` 算。
    - **鼠标点击后要主动重绘列表清环**: 焦点环画在**列表**上(`onPaintList`), 而点已选中的标签不改 active、不触发列表重绘 —— 环会留在屏幕上, 表现成"用 Tab 显示焦点环后再点一下, 反而冒出一个矩形框"。标签与列表的 `LButtonDown` 里除 `xui.NotePointerPress()` 外都要调一次 `t.Redraw(false)`。

156. **提示气泡的四方向定位与"每实例一个过渡键"**(`components/tooltip.go`, 探针 + 抓图实测):
    - **滚动区(LayoutFrame)里的元素收不到元素级 MouseMove/MouseStay**: 同一窗口的窗口级回调能收到移动, 滚动区里按钮的计数器始终是 0 —— 而桌面端页面内容几乎全在 `ui.ScrollArea` 里, 只挂元素事件的话**整页 tooltip 一个都不会弹**。做法是**窗口级兜底**: 每个宿主窗口挂一次窗口级 MouseMove, 回调里遍历已绑定元素, 用 `XEle_GetWndClientRect`(客户区坐标, 与回调给的 pPt 同一坐标系)做矩形包含判断。元素级事件保留(直属元素响应更快), `request` 内部去重。
    - **命中测试必须判可见性**: 分页容器里隐藏页面的元素矩形**仍然留在原位**, 只判矩形会命中最底层页面同位置的元素(实测: 悬停"提示气泡"卡的按钮, 弹出的是图标页某个图标的名称)。`XWidget_IsShow` 只看自己, 要沿 `XWidget_GetParentEle` 走到窗口判一遍。
    - **取宿主窗口不能用 `XC_hWindowFromHWnd(XWidget_GetHWND(ele))` 一步到位**: ScrollArea 内的元素拿到的 HWND 不是 xcgui 窗口, 映射出来是个无效小值(实测 7), 后续 `XWnd_GetRect`/presence/`AttachWindow`(主题重绘 + Tab 钩子)全部静默失败 —— 表现为"滚动区里的提示不弹、标签栏 Tab 键进不去"。统一用根包 `xui.HostWindowOf(hEle)`: 沿 HWND 父链往上找第一个映射成功且 `XC_IsHXCGUI(_, XC_WINDOW)` 的窗口。
    - **定位**: 窗口尺寸 = 气泡 + 贴元素那一侧的箭头(箭头高 = 气泡与元素的间距, 这样菱形正好填满缝隙、尖端顶住元素边缘); 翻转与夹紧按**窗口所在显示器的工作区**算(同 47), 显式 side 放不下时先翻对侧再夹紧。
    - **箭头尺寸取 7(不是 10)**: shadcn 的箭头是 `size-2.5`(10) 旋转 45° 的方块, 露在气泡外的高度只有 10/√2 ≈ 7。按 10 画时底宽 20 的三角形在单行气泡(高 28)里占了 71%, 看着是一块糊满整条边的大三角。
    - **箭头位置的夹紧要处理"气泡太窄"**: 理想夹紧区间是 `[radius+half, span-radius-half]`, 单行气泡(28)配 radius 6 + half 7 时下界 13 > 上界 11 —— 旧的"先夹下界、再夹上界"会退化成贴着一侧, 三角形仍然占满整条边且不正。现在的夹紧只保证三角完整落在气泡内(`[half, span-half]`), 因为让三角压过圆角最多差 0.2px(半径 6 的弧在 y=4.5 处内凹 `6-sqrt(36-1.5²)`), 肉眼不可见。
    - **气泡比元素短的维度上, 箭头取气泡中心**: 单行气泡高 28、按钮高 33 时, "元素中心"必然落在气泡中心上方约 2.5px(再加上 align=end 把底边对齐), 三角形看着就是歪的(实测反馈"三角偏上")。对齐语义本身已经让气泡贴着元素, 此时居中才像"尖角对着元素"; 元素比气泡长(长文案气泡)时仍按元素中心投影并夹紧, 否则 align=start/end 下箭头会被推到元素外面。
    - **箭头必须只画露在气泡外的那一段三角形**: 画完整菱形再用气泡盖住内侧, 在元素 alpha 半透明时会二次叠色(见 89); 箭头中心对准触发元素中心并夹进圆角内, 否则 `align=start/end` 时箭头会跑到气泡角上。
    - **过渡键必须每个 Tooltip 实例一个**(`xui.NextPresenceKey()`): 用固定常量会让同一宿主窗口上的多个实例抢同一条 presence 条目 —— 后显示的那个复用已到位的键, 过渡不跑, 气泡停在元素 alpha 的初始值, 表现为"气泡是灰的"(实测 5 个方向气泡同屏, 4 个半透明)。
    - **显示中切换目标(不重播进场)时要有"过渡代际"**: 上一次进场的每帧回调捕获的是旧 `px/py`, 在进场还没跑完时接着 Show 另一个元素, 旧帧会把窗口**拉回旧位置**(实测: 200ms 内切换目标, 气泡停在上一处, 时快时慢像"随机失效")。每次 Show/Hide 自增 `fadeGen`, 回调首行比对代际; 同时切换目标的那条路径要把元素 alpha 直接钉到 1(旧动画被打断会停在半透明)。
    - **延迟与 skipDelay**: `delayDuration` 用 `app.AfterFunc` 起表(回调回 UI 线程), 待显示期间 `MouseStay` 会连续触发, 用"计时器非空即已在排队"去重; 隐藏时记录时刻, `skipDelayDuration` 内移到下一个元素跳过延迟(radix 默认 300ms)。
    - **键盘焦点触发**: 元素 `AddEvent_SetFocus` 里只在 `xui.FocusedByKeyboard()` 为真时显示 —— 否则鼠标点完按钮气泡会一直挂着(点击产生的焦点也是焦点), 直到点别处; 失焦即隐藏。
    - **内容里的快捷键块**(Kbd): 量宽时要把它算进气泡宽并把右内边距从 12 收到 6(对应 `has-data-[slot=kbd]:pr-1.5`), 否则文字会被气泡挤成两行。

157. **可调分栏: 百分比换算像素要按累计边界取整, 拖动以分组为坐标原点**(`components/resizable.go`, 探针实测):
    - **分组必须有确定的主轴尺寸**: 面板主轴是 Fixed 像素(由百分比算出), 交叉轴 Fill。分组主轴是 Auto 时量算不到可用空间, 面板会停在创建时的 1px —— 放进卡片时把分组/卡片这条链的主轴尺寸写死(Fixed / Weight), 不要用 Auto 包着它。
    - **像素宽按累计边界取整**: `avail = 主轴长 − 手柄数 × 4`, 每块宽 = `round(累计百分比 × avail) − 上一块右缘`。逐块 round 会留下 1px 缝隙或向右溢出; 手柄是固定 4px, 直接用百分比排会多出 4px/个。
    - **拖动时鼠标捕获必须落到分组上, 坐标换算到分组客户区**: 手柄只有 4px 宽、拖动过程中还会随布局移动 —— 捕获到手柄时会以它为原点产生反馈(算出的位移把自身推走), 指针移出 4px 就丢事件。分组在拖动期间位置不变, 是稳定基准: `XEle_SetCapture(group, true)` + 分组 `AddEvent_MouseMove`, `nFlags & 1 == 0` 时收尾(窗口外松手没有 UP)。
    - **抓手和焦点高亮必须以分隔线的设备中心为基准, 抓手厚度取奇数设备像素**: 分隔线只占 1 个设备像素(中心 `c+0.5`), 抓手若用 `(Left+Right)/2` 的整数中心会差半个逻辑像素; 更关键的是**厚度方向的设备宽度不能是偶数** —— 4 逻辑像素在 150% 下是 6 设备像素, 偶数宽以 1 像素的线为中心永远一边多 1px(两轮修完仍"偏左", 实测 6 像素时左 3 右 2; 改成 5 后两边各 2)。画法: 从同一个 `c` 推设备矩形, 再用 `FillRoundRectF` 的浮点矩形。焦点态直接画**一条更粗的单线**(ring 色), 不要"色带 + 细线"或内缩描边: 4px 厚的手柄上双色叠画会被看成两条平行线(用户两次报"上面还是两条线")。
    - **一侧面板收到 0 时分隔线必须贴到手柄外缘, 抓手隐藏**: 4px 热区的手柄如果始终把线画在中心, 面板收起后线会停在分组边缘内 2px —— 外面再包一层边框(demo 的 borderedBox)时就是"外框 + 分隔线"两条线(用户两次报障; 底边因为布局取整只差 1px, 看起来是一条, 顶边差 2px 就露馅)。现在 `collapsedSides` 判断两侧面板是否已收到 0(像素上只剩 1px 下限), 收起侧的线画在手柄外缘(与分组边缘重合, 自然并进外框), 抓手不画。组件默认 min 仍与上游一致是 0; 若不想让面板被拖没, 用 `WithResizableMin`。
    - **Tab 链只覆盖窗口内容区的直接子元素**: 探针里直接挂在窗口内容区的可聚焦 Element 能被 Tab 到, 套一层 LayoutEle(甚至普通 Element)容器后就被跳过, Button 同理; `EnableFocus/EnableSwitchFocus` 都在也无效, `EnableKeyTab` 是"控件自己接收 Tab 输入"(Edit 之类)不是加入焦点链。所以分隔条(以及分页/标签栏等嵌在卡片里的可聚焦元素)不能用 Tab 聚焦: 先鼠标点击获得焦点, 再用方向键/Home/End。不要在任何组件文档里承诺"Tab 可聚焦"(见第七节)。
    - **键盘同 152**: 分隔条 `EnableFocus(true) + EnableSwitchFocus(true)` 后方向键会被框架先拿去切焦点, 必须挂窗口级 KeyDown 并先判 `XWnd_GetFocusEle == 手柄`。
    - **合成输入的坐标要按 DPI 换算, 且双击不能用 PostMessage**: `XEle_GetWndClientRect` 返回**逻辑**坐标, 而 `ClientToScreen` / `SetCursorPos` / `mouse_event` 走物理坐标 —— 150% 下直接拿逻辑值会在左侧 1/3 处落下(实测窗口收到按下、元素拿不到)。用 `XEle_GetWndClientRectDPI` 取物理客户区矩形再换算; 直接 `PostMessage(WM_LBUTTONDBLCLK)` 不会派发到元素, 真实双击用两次 `mouse_event`(间隔小于系统双击时间), 且先把光标移进窗口让悬停命中缓存刷新。


158. **侧边栏: 原生 Button 会吞掉子元素的点击, 动作钮只能做内部命中区**(`components/sidebar.go`, 探针 + 真实点击实测):
    - **菜单动作钮不能建成 Button 的子元素**: 子元素 `LButtonUp` 收不到, 而原生 Button 仍会派发一次 `BnClick` —— 点"更多操作"会连带触发整行回调(实测状态文字被整行点击覆盖)。改成在菜单按钮内部按坐标命中动作区: `BnClick` 里先判断光标是否落在右侧 20×20 区域, 是则执行动作并直接返回。分组标签是普通 Element(不截事件), 它的 + 仍用真正的子元素按钮(鼠标/键盘均可)。
    - **Button 鼠标事件里的 `pPt` 与元素客户区不同口径**: 同一位置按 `pPt` 算出来落在元素右边 2.7px 外(实测 150% DPI, 元素 224 宽), 按 `XEle_GetWndClientRect` 的坐标算又偏到别处。要判断"这次点击是不是动作区", 用物理屏幕坐标反算: `wapi.GetCursorPos`(物理) − `wapi.ClientToScreen(hwnd, 0,0)` − `XEle_GetWndClientRectDPI`(物理客户区矩形), 再除以 `XWnd_GetDPI/96`; 不要用事件 `pPt` 直接比元素内的逻辑矩形。
    - **普通 Element 悬停不会自动重绘**: rail(16px 热区)与分组动作钮(+ / …)的 `StateOf` 已变成 hover, 但画面不刷新, 淡入的底色/细线永远不出现; 在 ScrollArea 里补 `AddEvent_MouseStay` 也不可靠(同第八节 21)。改成宿主窗口级 `AddEvent_MouseMove` + `XEle_GetWndClientRect` 命中: 状态翻转时 `Redraw(true)`, 之后由 `xui.Hover` 的过渡定时器逐帧重绘。
    - **构造期不能调 `SetZOrderEx`**: 窗口还没显示时它在 xcgui 内部访问违例(建页阶段直接崩, 栈在 `XEle_SetZOrderEx`)。要让 rail 压在 main 之上, 用索引入口 `SetZOrder(0)`(InputGroup 同款, 构造期安全)。
    - **面板必须有确定高度**: 面板内部是 Header(Auto) + Content(ScrollArea, Weight 1) + Footer(Auto), Weight 只有父级高度确定时才算得出剩余空间 —— 外壳 `cy` 默认固定 480, 放进布局时用 `ItemHeight`/`ItemFillHeight` 给确定高度, 不要 Auto。
    - **两种收起的几何**: icon = 面板宽度本身从 256 收到 48(内容按新宽度重排); offcanvas = 面板保持 256 宽、x 随 gap 滑出(内容不缩, 由父级裁剪)。floating/inset 的收起宽度是 66(外边距 8×2 + 2px 余量), 面板 50。
    - **图标栏必须把动作/徽章/子菜单真正隐藏**: 菜单动作是绘制在按钮里的命中区, 图标栏里若只跳过绘制, 20×20 的命中区会盖在 34px 宽图标格的中央, 把图标点击吃掉。`applyIconMode` 里要关掉命中(`actionShown` 直接返回 false), 同时把子菜单/组标签 `Show(false)`(布局立即回缩)、菜单项高度改成 32(上游 `size-8!`, 所有尺寸档都一样)。
    - **隐藏组标签留下的旧矩形不影响布局**: `Show(false)` 后 label 的 rect 还是旧值, 但父级布局会跳过它, 后面的菜单从 y=0 开始排; 不要用透明度代替 Show(false), 否则 32px 空档会一直在。
    - **初始收起要直接把 progress 落到 0**: `Presence` 只在 SetOpen 时驱动过渡; 构造时 `WithSidebarOpen(false)` 必须同时把 `progress` 置 0(或 collapsible=none 时置 1), 否则首帧是展开宽度、下一帧才跳。
    - **子菜单开合要自己量高并逐帧写回**: 子菜单是 LayoutEle, 收起时 `Show(false)` 让布局立即回缩; 做 200ms 过渡时先 `Show(true)+Auto` 量自然高, 再锁 `Fixed` 从 0/目标值逐帧写到 `target*v`, 每帧重排所在菜单/分组 + ScrollArea 内容(否则滚动范围与下方条目停在旧位置), 到位后展开恢复 `Auto`、收起再 `Show(false)`。chevron 用 `IconRotated(chevron-right, ±90°)` 表示开合, 图标栏形态下不画。初始展开的子菜单也要把它的 presence 键预热到 1, 否则第一次收起瞬跳(同侧栏宽度)。

159. **图表: 曲线要自己采样, 悬停要走窗口级命中, 片内文字要按对比挑色**(`components/chart.go`, 探针 + 截图实测):
    - **D2D 下曲线绘制是空实现**: 直接调曲线 API 画出来是空白。折线/面积的平滑曲线按固定步长采样成折线段(`p.Draw.FillPolygonF` 填面积、`p.Line` 描线), 每段 12 个采样点在桌面尺寸下肉眼看不出棱角。
    - **ScrollArea 里的元素收不到元素级 MouseMove**(同 156): 悬停柱/扇区/数据点必须挂**宿主窗口级** `AddEvent_MouseMove`, 用 `XEle_GetWndClientRect` 与事件 `pPt` 同一口径做矩形包含, 再把点换算成元素内局部坐标交给几何命中。窗口级与元素级两个回调都挂: 前者兜底(页面在滚动区里), 后者让直接挂在窗口上的图表少一次遍历。
    - **隐藏页里的图表矩形还停在原位**: 命中前沿 `XWidget_GetParentEle` 链查 `XWidget_IsShow`, 否则切页后鼠标会命中别的页同位置的图表(与 tooltip 同因)。
    - **中性主题的 `Chart1..5` 是灰阶**: 上游饼图百分比标签写死 `fill="var(--background)"`(片色之上的反色文字), 那是给彩色色板定的; 灰阶片 + 浅色正文 = 看不见。改成在 `Foreground`/`Background` 之间**比亮度差**取更对比的那个(`chartContrastText`)。注意不能按"片色亮就用 Foreground": 暗色主题的 Foreground 本身接近白色, 浅灰片照样糊掉 —— 只比两个候选谁离片色更远, 与主题无关。
    - **提示卡画在元素内部**: 跟随光标在图表区内翻转/夹紧(允许盖住图例但不出客户区), 不新建小窗 —— 小窗要额外处理 DPI、置顶、主题重绘、点击穿透, 唯一收益是"能超出卡片边界"。

160. **平滑的扇形: `XDraw_FillPolygonF` 在 D2D 下没有抗锯齿, 而直通 D2D 拿到的渲染目标没有元素变换**(`components/chart.go`, 探针 + 像素扫描实测):
    - **现象**: 饼图/环形图的扇区外弧与内弧是一圈硬台阶(半径越大越明显)。加密采样**治不了** —— 台阶来自光栅化阶段(多边形是硬填充), 不是几何精度(`FillPolygonF` 画整圆同样有台阶)。
    - **修法**: 走与图标相同的 D2D 直通 —— `d2d.NewRenderTargetByHandle(xc.XDraw_GetD2dRenderTarget(p.Draw.Handle))` 取渲染目标、`factory.NewGeometryFigures([]d2d.GeometryFigure{{Points: flat, Closed: true}})` 建闭环几何、`rt.FillGeometryColor(geom.Base(), d2d.NewColorF(...))` 出图; 失败(纯 GDI 渲染)再回落 `FillPolygonF`。
    - **必须先用 `xc.XDraw_ConvRect(hDraw, &rc)` 把客户区矩形换算成画布坐标**, 再按 `sx/sy/DX/DY` 给渲染目标设置变换。直通拿到的 RT 上没有"当前元素"的变换(炫彩是在刷新自己的绘制命令时才施加), 不换算时 150% DPI 下扇区会按"逻辑像素当物理像素"画出来 —— 整个图形缩小到 1/1.5 并偏向左上(实测圆心正好落在逻辑坐标处, 一开始误以为布局错了)。变换要 `defer` 恢复, 否则残留矩阵会把后续 `XDraw_*` 绘制一起平移/缩放。
    - **元素 alpha 要自己乘**(`xc.XEle_GetAlpha`): 直通绕过了框架的元素 alpha 合成(与图标同因), 否则元素淡入淡出时扇区仍是全不透明。
    - **验收用像素扫描而不是目测**: 背景与片色之间应出现 1px 中间灰度(实测 `255 → 207 → 64`); 若 `255 → 64` 直跳就是没 AA。相邻扇区在 `y` 向扫描上是连续色段, 说明没有白缝。
    - **同一段代码里的百分比标签会被折行**: `DrawText` 在矩形内**自动换行**, 而 `TextWidth` 量出的宽度与绘制的排版宽度不严格相等 —— 卡着 `tw` 给框时 `"48%"` 会折成 `48` + `%` 两行(用户截图报障)。矩形左右各留 4px。

---

## 九、本设计依赖的已验证 API 清单(xcgui 1.4.3)

- `Element.AddEvent_Paint / AddEvent_MouseStay / AddEvent_MouseLeave / AddEvent_KeyDown`(widget/element.go:1192, 1288, 1336, 1744);键盘激活见第八节 108
- 焦点: `Element.EnableFocus / EnableSwitchFocus`(后者 = 接受方向键与 Tab 切换焦点)、`xc.XEle_EnableDrawFocus(hEle, false)`(关原生焦点框)、`xc.XWnd_GetFocusEle / XWnd_SetFocusEle`、`xui.StateOf(hEle).Focused` + `xui.FocusedByKeyboard()`(focus-visible 判定)、`window.NewByHandle(hWnd).AddEvent_KeyDown`(窗口级拦截, 语义见第八节 152)
- `Element.AddEvent_Menu_Draw_Background / AddEvent_Menu_DrawItem`(widget/element.go:2056, 2080)
- Menu / ContextMenu(见第八节 36/64/99):`Menu.AddItem / SetItemWidth / SetItemHeight / SetLeftWidth / SetBorderSize / EnableDrawBackground / EnableDrawItem / AddEvent_Menu_Select / AddEvent_Menu_Exit / AddEvent_Menu_PopupWnd`;右键触发用 `Window.AddEvent_RButtonUp`(pPt = 逻辑像素, 原点 = 窗口矩形原点)+ `xc.XEle_GetWndClientRect` + `XC_IsHXCGUI / XEle_IsEnable / XWidget_IsShow`;`xui.TextWidth` 按自绘字体自行给菜单条目定宽(原生字体与自绘不一致)
- `drawx.NewByHandle / EnableSmoothingMode / SetBrushColor / FillRoundRect(Ex) / DrawRoundRect(Ex) / GradientFill2/4 / DrawText / TextOut* / Image* / DrawSvg* / SetClipRect / SetLineWidth`
- 持续旋转 / 脉冲 / 高 DPI 位移(见第八节 87/88):`xui.Spin(hOwner, active) float64`(相位 0..1, 1s/圈)、`xui.Pulse(hOwner, active) float64`(相位 0..1, 2s/来回, 与 Spin 共用动画定时器、键空间独立;`SetAnimationScale(0)` 冻结)、`Painter.IconRotated(data, rc, color, strokeWidth, angleRadians, scale...)`、`Painter.IconRotatedOffset(..., dx, dy int32)`(按画布物理像素微调)、`Painter.TextF(str, *xc.RECTF, color, align)`(浮点矩形文字, 用于亚逻辑像素位移;整数矩形下与 `Painter.Text` 一致)
- **自绘"非 lucide"小图形走图标管线**, 别手算多边形: svgp 的子路径前加 `!` 就是填充形状(走 FillGeometry), 配 `Painter.Icon(path, rc, color, 0)` 直接出图, D2D 几何按 path+size 缓存。步进器的上下微调箭头就是两条 24 网格 path(三角占满 24 宽、Q 曲线倒角), 每枚箭头给一个**正方形**绘制块(管线按短边铺满)。曾手写"削角 + FillPolygonF 逼近圆弧", 10px 尺寸下不是歪就是糊、底角圆弧还会把墨迹洇到底边外, 全部弃用
- `xc.XBtn_GetStateEx`、`xcc.Button_State_*` 状态
- `xc.RGBA / xc.ParseRGBA / xc.HexRGB2RGBA`(Web 十六进制色直转)
- `svg.NewByString / SetUserFillColor / SetUserStrokeColor / SetSize`
- `ani.NewAnima / AnimaGroup`、`ease` 缓动
- `XWnd_Redraw / XWnd_GetDPI`、`XC_EnableAutoDPI / XC_EnableDPI`
- 显示器工作区(弹层/提示的翻转与夹取): `xui.MonitorWorkArea(hWnd uintptr) (xc.RECT, bool)`(物理像素、已扣任务栏; 必须传 Win32 HWND, 见 74/132/133/135);`xui` 运行时里, 组件文件不单独携带
- LayoutEle:`EnableLayout / IsEnableLayout / EnableHorizon / EnableAutoWrap / EnableOverflowHide / SetAlignH(V) / SetSpace / SetSpaceRow / ShowLayoutFrame / GetWidthIn(HeightIn)`、`xc.XWidget_LayoutItem_SetWidth/Height/SetMargin/SetMinSize`、`xc.XWidget_LayoutItem_GetWidth/Height/GetMargin`、尺寸类型 `xcc.Layout_Size_{Fixed,Fill,Auto,Weight,Percent,Disable}` 与对齐 `xcc.Layout_Align_*`(语义与"Fill≠剩余空间 / Weight 需父级确定尺寸"见第八节 29/41/55;Go 包装无 AdjustLayout, 用 `XEle_AdjustLayout`);`EnableLayout/EnableHorizon/EnableAutoWrap` **只有 LayoutEle 与窗口有**(普通 Element 只能当布局项);**负外边距可用**(竖向按钮组消 1px 按压缝, 见 86)
- `xc.XWidget_LayoutItem_SetPosition(h, l, t, r, b)`(布局项置位置 = 四边 inset 钉住, 让绝对定位浮层扛住窗口级布局趟; 语义与坑见第八节 128)
- 按钮组(见第八节 86): `xc.XDraw_FillRoundRectEx/DrawRoundRectEx`(四角独立半径)、`xc.XEle_SetUserData/GetUserData`(组件角色标记)、`XWidget_GetParentEle` + `XEle_GetChildByIndex/GetChildCount`(首尾判定)、`XEle_GetHeight`(按按钮视觉高定文本/分隔条高度)
- 布局读取/定位:`xc.XEle_GetRectLogic(hEle, *RECT)`(逻辑像素, 坐标相对父元素)、`xc.XEle_GetRect/GetWidth/GetHeight/GetPosition`、`window.AddEvent_Size(xc.WM_SIZE1)`(字号/尺寸变化后重算节流列宽等);`xc.XEle_SetPadding(hEle, l, t, r, b)`(元素级内边距, 已实测: Fill 子项按扣除 padding 后的宽度排版, 见 103)
- 滚动容器(`LayoutFrame`, 见第八节 66):`widget.NewLayoutFrame`(布局盒子 + 滚动视图, 内容总大小自动撑开) + `XLayoutBox_*`(与 LayoutEle 同一套 EnableHorizon/AutoWrap/OverflowHide/Align/Space) + `XSView_SetTotalSize/GetTotalSize/SetScrollBarSize/EnableAutoShowScrollBar/ShowSBarH(V)/GetViewWidth/GetViewHeight/GetViewRect/GetViewPosH(V)/ScrollPosYV(像素)/ScrollPosV(行)/ScrollTop(Bottom)/GetScrollBarH(V)Obj`、`ScrollBar.ShowButton/SetSliderPadding/GetButtonSliderObj` + `AddEvent_Size(xc.XE_SIZE1)`(自愈补排)
- `SliderBar.SetRange/SetPos/SetButtonWidth/SetButtonHeight/EnableHorizon/GetButtonObj`、`ProgressBar.SetRange/SetPos/EnableShowText/EnableStretch`、`TabBar.AddLabel/SetSelect/SetLabelWidth`、`ComboBox.CreateAdapter/AddItemText/SetSelItem/EnableEdit/EnableDrawButton/SetDefaultText/SetDropHeight/GetButtonRect`
- 手动建列表(时间框弹层, 见第八节 72):`ListBox.CreateAdapter/AddItemText/SetItemHeightDefault/SetDrawItemBkFlags/GetItemText(i, 适配器列)/GetTemplateObject(i, 模板itemID)/SetSelectItem(不滚动)/GetCount_AD/AddEvent_ListBox_DrawItem/Temp_Create_End/Select/GetScrollBarVObj`;`XShapeText_SetTextColor(h, 0)` 让模板文字全透明、`XWidget_LayoutItem_SetMargin(h, l, t, r, b)`、`xc.XWidget_LayoutItem_SetWidth(…Fill)` 对项模板文字会算出 0 宽(勿用)
- `XWnd_SetWindowPos(物理像素)`、`XC_hWindowFromHWnd`、`XEle_GetWndClientRect`、`XWnd_GetClientRect`、`wapi.ShowWindow/SetForegroundWindow/GetSystemMetrics/GetWindowLongPtrW`
- D2D 直通:`xc.XDraw_GetD2dRenderTarget / XDraw_ConvRect / XC_GetD2dFactory`(裸指针;直通绘制必须先用 `ConvRect` 换到画布对齐坐标,见第八节 43)+ **`github.com/twgh/xcgui/wapi/d2d`**(按 edge 的 COM 接口虚表风格封装,ABI 三条硬规则写在包注释里):借用入口 `NewRenderTargetByHandle`/`NewFactoryByHandle`;几何 `NewGeometry`/`NewGeometryFigures`(逐条可指定开/闭)/`Create{Rectangle,RoundedRectangle,Ellipse}Geometry` + `ID2D1Geometry.Usable/GetBounds` + `ID2D1PathGeometry.Open/Base` + `ID2D1GeometrySink.{SetFillMode,BeginFigure,AddLines,EndFigure,Close}`;绘制 `FillGeometry(Color)`/`DrawGeometry(Color)`(任意几何) + 图元 `DrawLine`/`DrawRectangle`/`FillRectangle`/`DrawEllipse`/`FillEllipse`(各带 `*Color` 便利版) + `CreateStrokeStyle`/`RoundStrokeStyleProperties`;其余 `SetTransform`/`GetTransform`/`GeometryCache[K]`(按工厂作废的几何缓存)
- backdrop-blur(见第八节 47):`xc.XImage_LoadFromGoImage`、`xc.XBkM_AddImage(hBkM, state, hImage, id)`、`xc.XBkM_GetObject`、`xc.XEle_GetBkManager`、`xc.XEle_ClearBkInfo`、`xcc.Element_State_Flag_Leave`(元素态, ≠ Window_State_Flag_Leave)、`XWnd_GetChildCount/GetChildByIndex`、`XEle_GetAlpha/GetBkInfoCount`、`wapi.GetWindowRect` + `PrintWindow(PW_RENDERFULLCONTENT)` 抓窗、`XWnd_SetTimer/KillTimer + AddEvent_Timer`(程序化验证)
- 排查"窗口可见但屏幕上看不见"(见 57):`user32!EnumWindows` + `GetWindowThreadProcessId` 枚举进程窗口(⚠️ 别用 `Process.MainWindowHandle`, 弹层小窗存在时它会返回小窗)、`IsWindowVisible`、`user32!GetLayeredWindowAttributes`(读 layered alpha/颜色键)、`PrintWindow` 与 `CopyFromScreen` 对照(画了 vs 屏幕上没有)
- `app.GetTextShowRect`(文本测量:限宽换行与绘制共用限宽;返回的 CY **已含行距** —— 13px 字体单行 18 / 两行 36, 见第八节 75)
- 折行文本(见第八节 90/91):`Label.WrappedHeight(width)`(当前字体下按宽度折行的高度;读的是 Label **当前文本**,先 SetText 再量)、`Label.SetWrap()`、`Label.SetAlignCenter()/SetAlignLeft()`、`Label.SetColorFunc(fn)`(语义字色按当前主题实时解析)、`xui.TextWidth(text, hFont)`(单行真实宽度, dialog 页脚按它定宽)、`xc.XC_IsHXCGUI(h, xcc.XC_LAYOUT_FRAME)`(识别滚动容器框架)、`xc.XWnd_AdjustLayout`(窗口级重排;`XEle_AdjustLayout` 不能直接作用于 LayoutFrame 的内容区)
- `Element.SetBorderSize(左, 上, 右, 下)`:原生 Edit/ComboBox 的**文字内边距**, 也是原生 Button 载体的文字区内缩(默认四边 1); 传**逻辑像素**、由框架随 DPI 自动缩放(实测 150% DPI: Input 传 10, 文字起点距边框 20 物理 ≈ 13.3 逻辑 = 10 + 字形左侧留白) —— 别再乘 `dpi/96`, 会重复放大; Label 清上下内缩的用法见第八节 109
- 窗口真透明:`Window.SetTransparentType(xcc.Window_Transparent_Shaped)` + `Window.SetTransparentAlpha(255)`、`xcc.Window_Transparent_Shaped`(xcconst.go:55);**对原生自建弹窗**(ComboBox 下拉, 只有窗口句柄)同样可用: `xc.XWnd_SetTransparentType / XWnd_SetTransparentAlpha`, 在弹出回调里(显示前)设置, 见第八节 69 —— **单次设置可信, 逐帧改会概率性冻结**(57/69), 原生弹层不做整窗 alpha 动画
- 背景对象美化:`BkManager.AddFill/AddBorder(state, color, id)`、`BkManager.GetObjectObj(id) *bkobj.BkObj`、`BkObj.SetRectRoundAngle/SetMargin/SetAlign/SetRotate/SetFillColor/SetBorderColor`、`xcc.BkObject_Align_Flag_*`(bkmanager.go:69/185, bkobj.go:104/42/50/66/74/90)
- Dialog:`XEle_SetZOrderEx(h, 0, xcc.Zorder_Top)`、`XEle_AdjustLayout`、`XEle_Redraw`、`XEle_GetChildCount/GetChildByIndex`、`XObj_GetType`(→ `xcc.XC_*`)、`XWidget_GetParentEle`、`XWnd_GetRect/Redraw`、`XWnd_SetFocusEle/GetFocusEle`、`xcc.AdjustLayout_No`、`wapi.GetAsyncKeyState`
- 输入组 / 自动补全(见第八节 147/148):`Element.AddEvent_SetFocus / AddEvent_KillFocus`(裸输入框聚焦进出时让外层容器重绘焦点环)、`xui.StateOf(h).Focused`(在容器 Paint 里判焦点)、`Element.AddEvent_LButtonUp(hEle, nFlags, *xc.POINT, *bool) int`(徽章/补全条目的点击回调)、`xui.Hex("#RRGGBB")`(自定义配色, 徽章 `WithBadgeColors`)
- 整组透明度:`xui.SetSubtreeAlpha(v, roots...)`(逐个子元素设 alpha, 不级联)、`xui.SetSubtreeAlphaSkipScrollBars(v, roots...)`(跳过滚动条子树, 弹窗打开时的淡入用, 见 98)、`Element.AddEvent_Show(fn(hEle, bShow, pbHandled))`(元素显示/隐藏事件; 滚动条在宿主隐藏时归零自己的显隐状态)
- Toast(见第八节 75): `XWnd_GetBodyRect`(Body 矩形, 阴影窗口有 band 偏移)、`XWnd_GetStayEle`(窗口级悬停判定)、`XEle_SetXCTimer/KillXCTimer + AddEvent_XC_Timer`(每条通知自己的倒计时)、`XEle_Destroy`、`Element.AddEvent_Destroy_End`、`Window.AddEvent_Size/MouseMove/MouseLeave`、`XEle_SetZOrderEx`
- Table(原生 List 载体):`CreateAdapterHeader/CreateAdapter/AddColumnText/SetHeaderHeight/SetRowHeightDefault/SetRowSpace/SetColumnWidth/SetSplitLineColor/SetDrawRowBkFlags/AddRowText/SetItemText/GetCount_AD/SetSelectRow/GetSelectRow/GetVisibleRowRange/RefreshRow/RefreshData/AddEvent_List_DrawItem/AddEvent_List_Header_DrawItem/AddEvent_List_Temp_Create_End/AddEvent_List_Header_Temp_Create_End/AddEvent_List_Select/GetTemplateObject/GetHeaderTemplateObject/GetHeaderHELE/GetScrollBarVObj`
- ⚠️ `TabBar.SetPadding` 在 xcgui.dll 4.0.0.0 中不存在(调用 panic)
- 直通 D2D(图标抗锯齿, 见第八节 43):`xc.XDraw_GetD2dRenderTarget(hDraw) → *ID2D1RenderTarget`、`xc.XC_GetD2dFactory → *ID2D1Factory`、`xc.XDraw_ConvRect(hDraw, *RECT) / XDraw_ConvXY`(客户区坐标 → 画布对齐物理坐标, 第三方绘图必须先换算); 用到的 D2D 方法: `ID2D1Factory::CreatePathGeometry`、`ID2D1PathGeometry::Open / GetBounds`、`ID2D1GeometrySink::{SetFillMode, BeginFigure, AddLines, EndFigure, Close}`、`ID2D1RenderTarget::{CreateSolidColorBrush, FillGeometry}`
- 官方自绘示例:BeautifyButton / CheckBox / RadioButton / Edit / ProgressBar / SliderBar
- 剪贴板读(InputOTP 粘贴, 见第八节 110 所在组件): `wapi.IsClipboardFormatAvailable(CF_UNICODETEXT) / OpenClipboard(0) / GetClipboardData / CloseClipboard` + `wapi.GlobalLock/GlobalUnlock`; GlobalLock 返回的 uintptr 别直接转 `unsafe.Pointer`(vet 报"可能的误用"), 用同布局的自定义 slice 头读 UTF-16; `wapi.GetAsyncKeyState(VK_CONTROL)` 判 Ctrl(VK 状态位按 `uint16` 与 `0x8000` 比, int16 会溢出)
- 弹层内容测量/非激活显示(Popover/HoverCard, 见第八节 115/116): `xc.XWnd_GetDPI`、`SetWindowPos(..., SWP_NOACTIVATE|SWP_SHOWWINDOW)`、`wapi.ShowWindow(hwnd, SW_SHOWNOACTIVATE=4)`、`xcc.WS_EX_NOACTIVATE`、`xc.XEle_SetRect`、`xc.XEle_AdjustLayout`; 关闭检查用 `wapi.GetCursorPos` + `wapi.GetWindowRect` 与 `XEle_GetWndClientRect` 的物理换算
- 圆形头像图片(见第八节 117): `xc.XImage_LoadFromDataRGBA(data, w, h)`(非预乘 RGBA, 内部转 BGRA)、`xc.XImage_Destroy`、`drawx.Draw.XDraw_ImageEx(hImage, x, y, w, h)`; `image.Decode`(png/jpeg/gif) 与 CPU 侧蒙版在 `avatar.go`
- 命令面板: 原生 `widget.NewEdit` 透明化(EnableBkTransparent/EnableDrawBorder(false)/EnableDrawFocus(false)/SetBorderSize(0,0,0,0)/EnableAutoSelAll(false)) + `SetDefaultText/SetDefaultTextColor/SetCaretColor/SetTextAlign(Left|Center_V)`; `AddEvent_Edit_Changed` / `AddEvent_KeyDown` 接管 ↑↓/Home/End/Enter/ESC(原生箭头键会先到这里, 标记 handled 即可); 列表滚动用 `ScrollArea` 的 `GetViewHeight/GetViewPosV/ScrollPosYV`
- 字体:`xui.FontSpec{Family, Size, Bold, Italic}`。**没有下划线字段**: `xcc.FontStyle_Underline` 在本框架的 D2D 文本路径下不渲染(探针: 下划线字体与普通字体像素一致), hover:underline 一律手绘 1px 线 —— 宽度取 `xui.TextWidth`, 落点见 Button 的 link 变体与 Accordion 触发区。9pt 以下的**常规**字重由 `FontHandle` 从字体文件创建(GDI+ 取英文族名 + Fonts 注册表找文件), 绕开 xcgui 的小字号宋体替换(见 149); 组件一律用 `FontHandle`, 不要直接 `font.NewEX`
- 弹层键盘钩子(`popoverPanel`, Select 的 ↑↓/Enter 用): `Window.AddEvent_KeyDown`(窗口级, 焦点在子元素上也先于元素分发)、`popoverPanel.OnKeyDown/OnEsc`(popup.go 预留的钩子) + `Element.SetFocus`(焦点即条目高亮)
- 轮播(见第八节 140/141): 纯自绘根 Element + 视口 + 每页一个 LayoutEle; 位移过渡复用 `xui.Presence`(`DropTransitions` 后每帧写各页矩形); 拖拽用窗口级 `AddEvent_LButtonDown/MouseMove/LButtonUp` + `Window.SetCaptureEle` + `AddEvent_KeyDown`(焦点在轮播内才响应); 上/下张用 `Button`(outline + icon-sm + rounded-full); 循环位置取正模
- 可调分栏(见第八节 157): 分组/面板都是 LayoutEle(主轴 Fixed 由百分比算出、交叉轴 Fill); `xc.XEle_GetWndClientRectDPI`(物理客户区矩形 —— 与逻辑坐标的 `XEle_GetWndClientRect` 区分)、`wapi.ClientToScreen`、`wapi.SetCursorPos`、`wapi.LoadImageW(0, wapi.IDC_SIZEWE/IDC_SIZENS/IDC_ARROW, wapi.IMAGE_CURSOR, 0, 0, wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)` + `Element.SetCursor`(col/row-resize 光标)、`Element.AddEvent_LButtonDBClick`(双击复位)
- 侧边面板 / 抽屉 / 菜单栏(见第八节 120~125): `XWnd_GetBodyRect`(遮罩铺 Body)、`XEle_SetAlpha`(遮罩淡入, 不级联子元素)、`XEle_GetRect/SetRect`(面板整体平移, 逻辑像素)、`XEle_SetCapture`(抓手拖拽捕获鼠标)、`wapi.GetCursorPos`(拖拽用屏幕坐标算增量)、`wapi.GetForegroundWindow/SetForegroundWindow`(子菜单与父弹层的焦点归属判定)、`app.AfterFunc`(弹层关闭后延迟把焦点还给触发区, 否则被随后的窗口激活覆盖)
- 官方透明/异型窗口示例:`Advanced/WindowPartiallyTransparent`、`webview/WebViewPartiallyTransparent`
- 图表(见第八节 159): 纯自绘根 Element, 绘制全走 paint 回调里的 `*xui.Painter`(`Line`/`Text`/`FillRound`/`StrokeRound`/`FillRoundCorners`)与 `p.Draw` 的浮点几何(`FillPolygonF` 画扇形与面积带、`FillEllipseF`/`DrawEllipse` 画点与圆环); 悬停靠宿主窗口级 `AddEvent_MouseMove` + `xc.XEle_GetWndClientRect` 命中(ScrollArea 里的元素收不到元素级 MouseMove, 同 156); 隐藏页判定沿 `XWidget_GetParentEle` 链查 `XWidget_IsShow`; 曲线**自己采样**成分段折线(D2D 下曲线绘制是空实现); 提示卡与图例都在同一个 Paint 回调里画, 不建小窗
- 侧边栏(见第八节 158): `wapi.GetCursorPos` + `wapi.ClientToScreen(hwnd, &pt)` + `xc.XEle_GetWndClientRectDPI`(物理客户区矩形) + `xc.XWnd_GetDPI` 换算动作钮命中; `Element.SetZOrder(0)`(构造期抬升 Z 序, `SetZOrderEx` 在窗口未显示时会崩); `Window.AddEvent_MouseMove/AddEvent_MouseLeave`(普通 Element 的悬停兜底); `XWidget_IsShow`; `Painter.TextF` + `xcc.TextFormatFlag_NoWrap|xcc.TextTrimming_EllipsisCharacter`(菜单项长文本省略); `xui.Hover` + `Presence`(宽度 200ms 过渡, 挂宿主窗口)
