# xui — shadcn 风格的 xcgui 自绘组件库: 自定义指南

> 这份文档随 `xui init` / `xui init -n` 安装到你的组件目录(默认 `ui/`, 也可用别的目录名), 与组件源码放在一起。源码注释里的
> "见 DESIGN 第 N 节"指的就是这里, **"见 DESIGN N / 第八节 N 条"指同目录的 `DESIGN-PITFALLS.md` 第八节**(它是本文第八节的完整正文,
> 编号一一对应; 本文第八节只留索引) —— 章节与条目编号稳定, 只增不改。
>
> **按需查**:
> - 改主题 / 换配色 → **第三节**(令牌表 + "绘制时取令牌"的规则)
> - 改组件外观 → **第四节**(以 Button 为例的解剖 + 每个组件的技术路线)
> - 改交互语义(悬停/焦点/禁用/校验失败) → 先看**第四节末尾的语义汇总表**
> - 碰到 xcgui 的怪行为(不刷新 / 透明失效 / 布局塌陷 / 事件不触发 / 文字量错) → **第八节索引** + 同目录 `DESIGN-PITFALLS.md` 的第八节, 按关键词找
> - 要调某个 xcgui API → **第九节**, 那里逐条标注了实测结论与坑
> - 安装更新 / 目录结构 / 已知限制 → **第五~七节**
>
> 全部结论基于 xcgui **1.4.3** 的真实窗口验证(含 150% DPI、亮/暗主题与多显示器场景), 不是凭文档或经验猜的;
> 组件实现以组件目录下的 `*.go` 为准。交互语义的原始参考是 shadcn-ui 的 React 源码, 文中"对齐"均指与它一致。

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
| `components/ui/` 拷贝源码 | 用户项目组件目录拷贝 .go 源码 |
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
用户项目 <组件目录>/*.go          组件源码(CLI 拷贝,用户所有)
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
> 两处与上游的差异要知道:① `DestructiveForeground` 是本组件库的**扩展令牌** —— nova 预设已删除 `--destructive-foreground`(破坏性按钮直接写死白字), 我们保留它以便组件取"破坏色上的文字"; **当前没有任何组件消费它**(库里破坏性一律软色调 `bg-destructive/10 + text-destructive`, 没有实心红底), 改它不会改变外观。② `--chart-1..5` 与 `--sidebar-*` 已建模(`Chart1..5` / `Sidebar*`): **chart 色板只提供给用户应用**(组件库不画图表), **sidebar 令牌由 `ui.Sidebar` 使用**(分组/条目/描边/焦点环取自 `Sidebar` / `SidebarForeground` / `SidebarAccent*` / `SidebarBorder` / `SidebarRing`);其中 `SidebarPrimary` / `SidebarPrimaryForeground` **当前没有消费点** —— 上游只用它们画"团队/项目 logo 方块", 本库侧栏没有那个区块, 保留字段是为了机械转译社区主题时不丢值。机械转译社区主题时这两组都要填上(内置主题已按它们换算好)。
> 基准圆角与 shadcn 一致: `--radius` 在 nova 预设下 = 0.625rem = 10px, 对应 `Theme.Radius = 10`。

Color 封装 uint32,提供 `xui.Hex("#fafafa")`(底层 `xc.HexRGB2RGBA`)。主题文件长得和 CSS 一样,任何 shadcn 社区主题都能机械转译成 Go(`init` 生成的 `theme.go` 默认取内置主题的**副本** —— `light = *xui.Light`):

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

**API 形状决策**:内置主题用**包级函数** `xui.Light / xui.Dark`,**不提供 `xui.Themes` 命名空间对象**。Go 里"包本身就是命名空间",用一个全局单例对象当模块不合惯例(收益只是省一个点号,代价是多一个可变全局 + godoc 层级更深)。将来做第七节 P4 的"主题画廊"(几十上百个主题, 包级函数会污染根包名)时,正确解法是新增 **`xui/themes` 子包**(`themes.Get(name)` / `themes.All`, 与本组件库 `icons` 子包同构),根包保留这两个函数作默认兜底;届时 `xui.Light` 迁到 `themes.Light` 保留别名即可, 非破坏。

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

| 组件 | 载体 | 方案 | 参考 |
|---|---|---|---|
| AppWindow | 原生 window.Window 封装 | 无系统标题栏(样式不含 Icon/Btn_*): 阴影圆角 + 自绘 `[图标][标题 Weight][自定义按钮区][最小化/最大化/关闭]`; 双击标题栏最大化/还原; 内容区 = 纵向流式 Weight 1(第八节 76) | 窗口 |
| Button | embed Button | paint 壳, 原生文字(WithIcon 时自绘图标) | BeautifyButton 示例 |
| Card | LayoutEle(布局容器) | 圆角面板 + ring 描边; 内容区是 Fill 子项、左右内边距用元素 padding / 上下用外边距(随卡片缩放, 见第八节 129), 高度可按内容自撑(Auto), 见第八节 56; 页脚是横向布局容器(内边距 + items-center), 见第八节 95 | — |
| Collapsible | LayoutEle(根 + 内容) + Button(触发区) | 竖向流容器: 触发区由调用方创建 Button 后 `BindTrigger` 绑定(展开态同步 Button 的 aria-expanded 视觉); 内容高度收起 0 / 展开 Auto, 切换时 200ms 逐帧重排祖先并整窗重绘(第八节 97) | collapsible.tsx + radix collapsible(高度 CSS 变量动画示例) |
| AspectRatio | LayoutEle(布局容器) | 宽度驱动高度: 尺寸事件按新宽度算高写回, 延迟到下一轮消息并逐级重排祖先(同第八节 93 的三段式); 内容区随比例框同步尺寸, 见第八节 94 | shadcn/radix aspect-ratio |
| Alert | LayoutEle + Label/Element | 卡片底 + 1px border; 图标/文本列横向流式, 标题/描述按宽度量高后写回, 动作槽按内容自撑; 见第八节 90 | alert.tsx(radix-nova) |
| Empty | LayoutEle + Label/Element | 竖向居中的空状态: Body(p-6/gap-6) + Header(gap-2) + Content(gap-4); 媒体是自绘图标块(icon 变体 size-10 圆角 bg-muted); 不画边框(上游 `border-dashed` 无宽度), 需要边框放进 Card | empty.tsx(radix-nova) |
| Field | LayoutEle + Label/Element | 表单字段组合: Field/Set/Group 全是流式布局容器; Label/Description 走 Label, Separator 自绘线 + 背景压字, Error 多条 Label 去重; 横向 Content 固定 240(第八节 101) | field.tsx(radix-nova) |
| Checkbox/Radio | embed Button | 画方框/圆 + 勾线, 文字原生 | BeautifyCheckBox/RadioButton 示例 |
| Input | embed Edit | 只画边框+焦点环, 原生光标/IME | BeautifyEdit 示例 |
| Textarea | embed Edit | Input 的多行预设 + **随内容自动长高**(第八节 166): 上下内边距 8(py-2)、最小高 64(min-h-16); 行数取原生 Edit 的显示行数(含折行), 内容装得下, 不挂滚动条 | textarea.tsx(radix-nova) |
| InputGroup | LayoutEle(外框 + 三行) + 裸 Input | 输入框与 addon 共用一层边框/焦点环: 内层是 `WithInputBare` 裸输入, 环/禁用/校验由容器画; addon 四向(inline-start/end、block-start/end), 内容可 Text/Icon/Button/IconButton/Kbd/Spinner/MenuButton/自定义; 复合 addon(多控件、下拉菜单、按钮尺寸档)见第八节 165 | input-group.tsx |
| Switch | 纯自绘 Element | 轨道+滑块, 尺寸预设; hover 过渡走运行时 `anim.go`(第八节 45); aria-invalid 红边 + 常驻红环, 空格切换, `SetDisabled` 关鼠标与键盘(第八节 146) | — |
| Label/Separator/Badge | 纯 Element | paint;`Label.SetAutoWrap()` 按当前宽度量折行高度并自撑,宽度/文本/字体变化时重排(量高口径见第八节 90) | — |
| Progress | 纯自绘 Element | h-1 圆角轨道 + 指示器(宽度 = value/max); max 可配, 不确定态短条往返, 指示器颜色可覆盖; `ProgressField` = 标题+百分比+条(第八节 145) | BeautifyProgressBar 配方移植成 token + progress 文档 |
| Slider | 纯自绘 Element | **弃用原生 SliderBar**(滑块按钮 BkInfo 残留压不住, 见第八节 25), 自实现拖拽/点击跳转/键盘; 值数组(单值/范围/多滑块)、垂直、禁用、最小间距、inverted、onValueCommit(第八节 144) | BeautifySliderBar 示例 |
| Rating | 纯自绘单元素 | 5 星(可配)、整星/半星、只读/禁用; lucide 描边星 + 同色实心 SVG 多边形叠画(半星沿中轴线闭合); 悬停预览 + 点击提交; 焦点环只围星星区; 键盘挂窗口级(同第八节 152) | 社区 rating 通行语义(shadcn 无官方组件) |
| ComboBox | embed ComboBox | 画输入框; 下拉 ListView 弹层在 `AddEvent_Combobox_Popup_List` 里半接管(第八节 30) | 库文档 FAQ |
| Tabs | LayoutEle(列表底) + Button × N(全自绘) | paint(default/line 两变体, 行内图标 + 单标签禁用); 整块列表一个 Tab 站点 + 内部方向键(键盘挂窗口级, 见第八节 152/155); `BindPanel` 按选中项显隐面板; 换行/纵向/`activationMode`/value 语义 | tabs.tsx(radix-nova) |
| Dialog | 纯 Element 覆盖层 | 全窗遮罩元素(自绘黑 10% + backdrop-blur) + 面板子元素;ESC/点遮罩关闭, Tab 焦点循环(第八节 33);可滚动内容用内嵌 ScrollArea(第八节 91) | — |
| AlertDialog | 复用 Dialog 覆盖层 | 同一套遮罩/模糊/焦点陷阱/模态栈;面板按 default/sm 排头部与页脚, media 自绘;点遮罩不关、焦点强制 Cancel(第八节 91) | alert-dialog.tsx(radix-nova) |
| Menu | 原生菜单 | `XMenu_EnableDrawBackground/EnableDrawItem` + `AddEvent_Menu_Draw_Background/DrawItem`(第八节 36) | xcgui 文档 |
| Tooltip | FloatWindow | paint, 异型透明窗口 + 进出场 fade/slide(第八节 24/47); 上下左右四方向 + align/sideOffset + 自动翻转夹紧(第八节 156), 延迟/skipDelay、键盘焦点触发、快捷键块 | tooltip.tsx(radix-nova) |
| Table | 原生 List + 项模板 | 表头/行底/边框/滚动条全自绘(绘制回调按**单元格**触发, 见第八节 40); 原生 Table(Shape 系)与 shadcn 语义不符, 不单独做; 另有 `TableColumn.Align`(原生文本对齐)、`Table.SetColumns` 换列结构(重建适配器, 见 169)、内部 `onCellExtra` 单元格扩展钩子(见 167/168) | Basic/List 示例 |
| DataTable | LayoutEle(工具栏 + Table + 页脚) | 在 Table 之上做排序/筛选/分页/双击编辑: `view` = 过滤 + 稳定多键排序后的原始行下标, 排序与分页全部在 Go 侧做(不启用原生 `SetSort`, 见第八节 162); 排序图标由 Table 的内部表头扩展钩子绘制(另注册 DrawItem 会被底色盖掉); 表头悬停 / 列拖拽订阅表头元素鼠标事件(见 176), 显示顺序存在 `order`、隐藏列位置保留; 筛选是带清除钮的 Input(清除按钮要自己广播变更, 见 163); 页脚每页行数用 Select、翻页用 outline 图标钮; 空状态画在 List 的 `PAINT_END`(见 162); cy=0 时根高度按工具栏/每页行数/页脚配好(见 164); 勾选列 / 列头菜单 / 分面筛选 / 富单元格分别见 167~175 | shadcn data-table 文档(tasks 示例 + TanStack Table 的 autoResetPageIndex / enableSortingRemoval / multiSort 语义) |
| Tree | 原生 Tree + 项模板 | `XE_TREE_DRAWITEM` 自绘行底/参考线/chevron, 模板子元素保留(第八节 42); 勾选模式在模板里加 16px 占位、复选框由 DrawItem 自绘, 状态按项 ID 存并由直接子项聚合(第八节 170) | 库文档 |
| Pagination | 纯自绘单元素 | 条目行 `[上一页][页码…][下一页]`, 逐项 hover 过渡; 恒定槽位折叠(第八节 53); 简洁 / 仅导航 / 只留图标三种形态; 一个 Tab 站点 + 内部方向键(键盘必须挂窗口级, 第八节 152/153) | — |
| Steps | 纯自绘单元素 | 横向等宽列(圆点在上、标题/说明居中)或竖向固定行高(圆点在左、文字在右); 状态 = 已完成(主色实心 + 对勾) / 进行中(主色环 + 主色编号) / 未达(边框环 + 弱化编号) / 出错(破坏色实心 + X, 指向它的连接线也转破坏色), 按 `current` 推导、`StepItem.Status` 可单步覆盖; 已完成段连接线取主色, `StepItem.Percent` 在进行中圆点上画进度弧; `StepItem.Icon` 非空时圆点内改画该图标(代替序号/对勾/X, 状态由圆点/连接线颜色区分); 默认只有已完成步可点回, `WithStepsClickable` 放开任意非禁用步、`StepItem.Disabled` 单步禁用; `WithStepsSize(StepsSmall)` 走独立几何/字号表、`WithStepsInitial` 只偏移编号; 一个 Tab 站点 + 内部方向键(键盘挂窗口级, 同第八节 152), 方向键在所有步骤间移动焦点、Enter/Space 激活可点步, 按任意键即切键盘焦点态; 焦点环外扩 3px, 元素四周预留 pad; `SetBackdrop` 跟随宿主底色 | shadcn 无官方 stepper, 社区通行语义 + Ant Design Steps |
| Timeline | 竖向/水平 LayoutEle + 每项一个 LayoutEle | 轴线与节点由条目自绘, 节点列先铺宿主底色擦除; 竖向: 圆点对齐**第一个可见子项**的垂直中心、条目间距用底部内边距表达, 轴线连续; 水平: 条目等分容器宽, 尺寸变化时实测各条内容高并按共享轴线写上下内边距; 内置 Label 时间/标题/说明, 内容槽可继续加任意元素; 节点支持实心/空心/语义色/自定义图标/加载态; 首条无上游线、末条无下游线 | 社区 timeline 通行语义 + Ant Design Timeline(shadcn 无官方组件) |
| Icon | 直通 D2D 几何绘制 | lucide path → 描边几何 + `DrawGeometry`(圆角样式), 色点走 `FillGeometry`(第八节 43); 回落折线描边 / `svg.Svg` | svg 包 |
| NativeSelect | ComboBox(value 封装) | 载体就是 NewComboBox: 触发区自绘 + 原生弹层(第八节 30), 在其上提供 `<option value>` 字符串语义(Value/SetValue/OnValueChange) | native-select.tsx(radix-nova) |
| Select | Button(触发区, 全自绘) + popoverPanel + ScrollArea | 触发区与 ComboBox 同一套 paint 规则; 弹层是自建阴影窗(popover.go 骨架, item-aligned 定位见第八节 71/74), 条目是自绘 Button(悬停/焦点 = bg-accent, 选中右侧 check, 禁用置灰), 分组标签/分隔线是自绘 Element; 滚动用 ScrollArea(条目超出高度才出), 键盘 ↑↓/Home/End/Enter 走弹窗 KeyDown 钩子 | select.tsx(radix-nova) |
| ColorPicker | Button(触发区, 全自绘) + popoverPanel(LayoutEle 内容) | 触发区同 Select 的 paint(左侧 16px 色块 + hex + chevron); 弹层内容自上而下: SV 方块 / 色相条 / 预设色 / Hex 输入。**SV 方块 = 纯色相底 + 两层 alpha 渐变**(横向 白(255)→透明, 纵向 透明→黑(255))—— 不要用"白→纯色""纯色→黑"两组实色渐变, 中段会出现灰调偏移, 取到的 s/v 与显示对不上; 色相条 = 六段 60° `GradientFill2` 且每段右边界 +1 盖住抗锯齿缝, 指示器是**白色实心圆 + 深色外圆**(元素比条高 8px, 圆才不会被上下裁掉); **所有圆/环都用"两层实心 `FillEllipse`"而不是 `StrokeRound`** —— 1~2px 描边落在半像素上, 圆边会毛糙, 拖动时 6px 宽胶囊的黑边还会一会儿 1px 一会儿 2px(看着像"深色底左右不均"), 实心圆叠加则天然对称且平滑, 环的"挖洞"用面板底色(`t.Popover`)盖出来。**选中环与圆点之间必须留 1px 缝**: 环色(`Ring`)和选中的色块经常同色(选中蓝 / 主色也是蓝), 去掉缝以后环、缝里的 border 淡边、圆点挤在一起, 看着就是"双圈甜甜圈"; 预设色每行 5 个圆点、行上下留 5px 给选中环(否则环的上下被元素矩形裁掉, 只剩左右两段弧), 命中测试按圆内判定(空白区不响应); Hex 用 Input(`#` 前缀, 满 6 位才应用, 回写时用 syncing 标志防止 Edit_Changed 把半成品解析成颜色)。SV/色相条按下即取色并 `SetCapture`, 指针拖出元素仍可调, 但**色相位置要先夹进 [0, w-1]**: 拖出右边界会算出 h>360, `math.Mod` 把它折回 0, 滑块瞬移到最左继续拖(左边同理折到最右); 预设色列宽用浮点算再四舍五入, 整数截断会让圆心列列左移; 触发区用 **BnClick**(原生 Button 不向外部派发 LButtonUp, 用 LButtonUp 会"点了没反应"); 过渡键一律一次性(第八节 183) | 生态 color-picker 约定 + shadcn Slider 的 thumb 观感 |
| FileDialog | 纯门面(不建元素) | 系统原生对话框: 打开/多选走 `wutil.OpenFileEx`, 保存走 `SaveFileEx`(默认覆盖提示), 选择文件夹自己拼 `BrowseInfoW` + `SHBrowseForFolderW`(`wutil.OpenDir` 的标题写死, 这里要支持 `WithFileDialogTitle`); `FileFilter{Name, Patterns}` 自动格式化成 "名称 (通配符)", 多选默认最多返回 10 个; 取消 = 空串/nil; 原生模态会自己跑消息循环, 必须在 UI 线程(事件回调)里调用 | 桌面应用约定; 不套 xui 主题 |
| InputOTP | 每格一个原生 Edit(32×32) + 自绘壳 | 文字居中用原生 `SetTextAlign(Center\|Center_V)`(单行 Edit 程序 SetText 同样生效; 不要用 border 挤位置, 会切字, 见第八节 110); 字符一律在 `AddEvent_Char` 拦截后自己落格(替换已填格 + 非法字符不入格 + 自动前进), 粘贴在 KeyDown 拦 Ctrl+V 读剪贴板分发; 相邻格共享边 = 每格画 top/bottom/right、首格加 left、中段格的 left 用底色擦掉(防 2px 叠边); aria-invalid 的组外圈画在容器 PAINT_END(子元素之后, 普通 paint 会被格子盖掉) | input-otp.tsx(input-otp 包) |
| Accordion | LayoutEle + Button(check, 全自绘) + Collapsible | 每项 = 触发钮(Toggle 式全自绘: 文字 hover 下划线**手绘 1px 线**, 箭头随开合换 ChevronUp/Down, 高度固定 40) + 一个 Collapsible 做高度过渡; 条目间 1px 分隔线(末项没有); single 互斥在 item.setOpen 里关其它项, 只通知一次; 内容区 pb-2.5 用 10px 透明占位(不能用外边距, 见第八节 111) | accordion.tsx(radix-nova) |
| Popover | popoverPanel(阴影窗) + LayoutEle | 复用 datepicker/select 的弹层底座, 内容区改成布局容器: 离屏显示一次量出 Auto 高(第八节 115), 再按 side/align 摆放并在工作区内翻转/夹取; 点击触发, 失焦/ESC 关闭; 公开 `NewPopover` 与 Header/Title/Description 助手。内容里嵌了会自己弹窗的控件(ColorPicker / Select / DatePicker)时要 `WithPopoverKeepOnBlur(true)`: 那些控件一点就会让本卡片失焦关掉, 取色/下拉面板却还开着, 用户看到"卡片闪没" | popover.tsx(radix-nova) |
| HoverCard | popoverPanel(非激活模式) + LayoutEle | 同 Popover, 但窗口带 `WS_EX_NOACTIVATE`、不调用 SetForegroundWindow、不靠 KillFocus 关闭(第八节 116); 700ms 悬停打开 / 300ms 离开关闭, 指针仍在卡片上时关闭检查自动续期 | hover-card.tsx(radix-nova) |
| Avatar | 纯自绘 Element | 圆形图片用 CPU 居中裁剪 + 抗锯齿圆形蒙版生成 RGBA, `XImage_LoadFromDataRGBA` + `XDraw_ImageEx` 绘制(第八节 117); 无图回落首字母, 右下角徽标; AvatarGroup 用横向布局负间距叠放 | avatar.tsx(radix-nova) |
| Command | LayoutEle + 原生 Edit + ScrollArea | 搜索行(自绘图标 + 透明 Edit + 下边框) + ScrollArea 列表; 分组/条目是自绘 Element, 输入即过滤(跳过禁用), ↑↓/Home/End/Enter 在 Edit 的 KeyDown 里接管; 选中不自动关闭(交给 Dialog/Popover 组合) | command.tsx(cmdk) |
| Sheet | 复用 Dialog 覆盖层(模态栈/焦点陷阱/遮罩) | 面板贴边: 左右占满高、上下按内容撑开; 面板与其直接子元素整体平移做滑入(第八节 120), 遮罩 bg-black/50; 页脚按钮纵向铺满; 只画贴边侧 1px 边框 | sheet.tsx(radix-nova) |
| Drawer | 复用 Sheet | 底部 max-h-80vh + 顶部抓手 + 上方圆角; 抓手按住捕获鼠标, 拖过面板高 1/3 关闭、否则回弹(第八节 121); 方向可改 top/left/right | drawer.tsx(vaul) |
| Menubar | 自绘根条 + 每个菜单一个 popoverPanel + 自绘条目 Element | 原生 `XMenu_Popup` 阻塞、期间主窗口收不到鼠标移动, 无法做悬停切换, 因此弹层自绘条目(与 Menu 共用 menu_model.go 的条目模型); 打开后鼠标移到其它触发区即切换; ↑↓/Home/End/Enter/←→/ESC/Tab; 子菜单是第二个激活弹层, 父层 `keepOnBlur` 防止失焦关闭(第八节 122~125) | menubar.tsx(radix-nova) |
| NavigationMenu | 自绘根列表 + **一个共享 popoverPanel(非激活)** | 所有菜单共用一个视口: 贴列表左下方、宽高随当前菜单内容变化, 切换时不重开窗口; 触发区/顶级链接是全自绘 Button; 内容页是 LayoutEle(链接行助手 `Link`/`LinkIn` + 任意控件); 悬停 200ms 打开、离开 150ms 收起、关闭后 300ms 内切其它触发区立即打开(第八节 137 的轮询同时负责链接 hover 与收起判定) | navigation-menu.tsx(radix-nova) |
| Carousel | 根 Element + 裁切视口 + 每页一个 LayoutEle | 每页宽 = 视口宽 − gap(16), 过渡 300ms 用 `xui.Presence` 每帧重写各页矩形; 循环用"模总长"折回, 首尾相接、不需要克隆页(第八节 141); 拖拽走窗口级起手 + `SetCaptureEle` 跟手(第八节 140); 上/下张是内容外侧 48px 的圆形 Button | carousel.tsx(radix-nova, embla) |
| Resizable | 分组 = LayoutEle(横向/纵向) + 面板 = LayoutEle + 分隔条 = 自绘 Element | 面板按百分比分配主轴空间, 像素宽按累计边界取整(每个手柄固定 4px); 分隔条画 1px `bg-border` 线 + 可选 4×24 抓手; 拖拽把鼠标捕获到**分组**(手柄会随布局移动, 不能当坐标原点), 键盘挂窗口级(同 152): 方向键 5%、Home/End 到边界、双击复位(第八节 157) | resizable.tsx(radix-nova) + react-resizable-panels v4 |
| StatusBar | LayoutEle(整条 + 三槽) + 自绘 Element(条目/抓手) | 整条横向流式: 左 Weight1 / 中 Auto / 右 Weight1(开头一段 Weight 撑子把条目顶到行尾) / 抓手 Fixed16; 顶边 1px 按设备像素对齐(同 Separator), 条目自绘 + `xui.Hover` 过渡, 抓手 `SetCapture` + `GetCursorPos` 屏幕物理增量 ÷ 窗口 DPI 调 `Window.SetSize`; 尺寸读数挂宿主 `WM_SIZE`; 提示走 `Tooltip.BindTip(TipTop)` | 桌面状态栏通行形态(shadcn 无官方组件, 语义由本库定死) |

非 Button 载体的组件(hover 态自管)用运行时提供的 `HoverTracker`(基于 `AddEvent_MouseStay/MouseLeave`,已确认存在)。

尺寸预设对标 shadcn(`h-9 px-4` 之类),经 DPI 助手换算。

### 与 shadcn 对齐的语义(权威汇总)

> **这是交互语义的唯一汇总表**;改任何组件的交互前先查这里,再回到 shadcn 官方源码确认。每条的根因/实测见第八节对应条目。

| 语义 | shadcn 源码依据 |
|---|---|
| **hover 过渡 = 150ms cubic-bezier(0.4,0,0.2,1)**(tailwind `transition-*` 默认值); 组件侧 `xui.Hover(ele, 悬停中)` / `xui.HoverItem(hOwner, idx, 悬停中)` + `xui.Fade(静止色, 悬停色, hv)`, 叠加层用 `ScaleAlpha(hv)` | button/tabs/switch/badge 的 `transition-all`; table/select/checkbox 的 `transition-colors`; 见第八节 45 |
| hover:bg-primary/80(实心)/80 透明度 | button.tsx variant.default |
| focus-visible 键盘才显示焦点框,ring-3 ring-ring/50,边框变 Ring 色;destructive 变体**覆写**为 border-destructive/40 + ring-destructive/20;焦点内芯**先铺宿主底色再叠 bg** —— bg 可能是半透明色(destructive/10), 直接叠在 halo 上会变成 30% 红、内芯比光环还深, 浏览器里两者是各自独立合成在页面底色上的;幽灵/链接(bg=None)铺 Card 即"透明底透出面板色"(halo 画在元素矩形内, 口径同 FocusFrame) | button/input/checkbox 的 focus-visible:* |
| 文本输入框点击聚焦也显示环(浏览器对文本控件恒匹配 focus-visible);禁用 = `bg-input/50`(暗色另有 `dark:bg-input/80`)再按组级乘 50%, 字色预混变淡 | input.tsx + 第八节 61 |
| 文本域 = 输入框同款壳体/焦点环/校验环/禁用态 + **高度随内容自动增长**(min-h 64、上下内边距 8, 内容装得下所以没有滚动条);选中/输入法/右键菜单全部是原生多行 Edit 行为 | textarea.tsx(`field-sizing-content` / `min-h-16` / `py-2`) + 第八节 166 |
| disabled:opacity-50 原色半透明 | 各组件 disabled:* |
| **禁用/未选态的半透明令牌要"乘"不要"覆盖", 且先铺宿主底色**(暗色 input/border 令牌自带 alpha) | checkbox/radio/switch/input.tsx + 第八节 61 |
| active:translate-y-px 按下整体下沉 1px(含文字→文字手动绘制);**aria-haspopup 触发区(菜单/下拉)不下沉**;aria-expanded 展开时 outline/ghost 换 bg-muted、secondary 保持 secondary 底 | button.tsx 基类 + aria-expanded 变体 |
| **button group**: 首尾保留外圆角、相邻边收直角;xs/sm/icon-xs/icon-sm 在组内圆角统一成 rounded-lg;直系子项相邻边框自然并成 1px;组内嵌组时直接子项间距 8(gap-2);文本块/分隔条拉伸到按钮视觉高 | button-group.tsx + button.tsx 的 `in-data-[slot=button-group]` 规则 + 第八节 86 |
| **button 加载态**: 行内前置图标位置换成 lucide loader-circle(纯图标按钮替换图标), 1s/圈 linear 持续旋转, 常配 `disabled`;只负责视觉, 是否禁用由调用方决定 | shadcn button 文档的 spinner 示例(`LoaderCircleIcon animate-spin` + disabled)+ 第八节 87 |
| destructive 软色调(bg-destructive/10 + 红字) | button.tsx / badge.tsx |
| radio 选中 = bg-primary 实心圆 + size-2 白点(bg-primary-foreground); **RadioGroup = 组值 + 竖向 gap-2(8)**: 方向键在可用项间移动并选中(↑↓←→ 循环)、Home/End 到首末、空格选中当前项; 同父兄弟交给原生互斥, 放进 Field 的跨父级条目由组手动取消; Tab 按桌面惯例遍历各项(非 radix 的 roving tabindex) | radio-group.tsx + 第八节 143 |
| checkbox size-4 rounded-[4px] border-input;未选**亮色无底色 / 暗色 dark:bg-input/30**;选中 bg-primary + 白勾;不确定态(`checked="indeterminate"`)画减号、点击落到选中;空格切换;禁用 = 上面每个图层各乘 50% | checkbox.tsx + Radix Checkbox indeterminate + 第八节 61 |
| card 描边 = ring-foreground/10(非 Border 令牌), rounded-xl;卡片头 = 一行两列(左列标题+说明 `1fr`, 右列动作 `auto`), 动作区贴右上、跨标题与说明两行顶端对齐(gap-1 = 4);页脚 = 全宽 `border-t + bg-muted/50`、内容垂直居中、四边内边距跟随卡片尺寸(default 16 / sm 12)、只有下两角随圆角 | card.tsx + `card.go` 的 `CardHeader` / `CardFooter` |
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
| rating 星级评分 = **纯自绘**(普通 Element): lucide 描边星 + 同色**实心多边形**叠画(星形左右对称, 半星沿 x=12 中轴线闭合), 空星描边 = `muted-foreground/50`、实心/半星 = `primary`; 每颗星左半 = i+0.5(半星模式)或 i+1、右半 = i+1, 星间空隙算前一颗星右端, 星星区左右留白与 Fill 拉宽后的空区不响应; 悬停预览、点击提交, **再次点击当前值归 0(allowClear 默认开, `WithRatingAllowClear(false)` 关)**, 清零后指针没动不恢复悬停预览; 只读仅展示、禁用 opacity-50 且不可聚焦; 整块一个焦点元素, ←/↓ 减一档、→/↑ 加一档、Home/End 两端(到端点不循环), 键盘改值立即触发 OnChange 并画 focus-visible 环(环带只围星星区, 不用 `p.FocusFrame` 的元素整矩形), `WithRatingKeyboard(false)` 把方向键交回 xcgui 的焦点切换; 键盘处理挂窗口级(同第八节 152), 嵌套布局里的 Tab 边界见第八节 157; **尺寸档** `RatingSmall/Medium/Large`(星边长 15/20/25, 自然宽高按它算, 只在构造时给); **文案展现** `WithRatingLabels([]string)` 在星星区右侧画 ceil(当前值或悬停预览)-1 对应的文案, 半星按 ceil 取上一档, 值为 0 不画; `OnHoverChange(fn)` 在预览变化时回调(-1 = 离开/清除); **自定义图标** `WithRatingIcons(outline, full, half)` 三态成套(半星须沿 x=12 闭合), 悬停那颗星放大 1.1 倍(走运行时 `xui.HoverItem`, 命中矩形不变); `OnFocus` / `OnBlur` / `OnKeyDown(vk)` 透出焦点与按键事件(键盘关闭时按键仍转发), `Blur()` 把焦点交还宿主窗口 | 社区 rating 通行语义(shadcn 无官方组件) + Ant Design Rate |
| slider 元素沿轨道方向比可见轨道多 18(两端各 9 = 滑块半径 6 + ring 3, 见第八节 26); 水平 `cx` / 垂直 `cy` 是**可见轨道长度**, 滑块中心在轨道两端之间移动; 垂直默认轨道 160(radix `min-h-40`), 值从下往上 | slider.tsx(radix 滑块中心贴轨道端点) + 第八节 144 |
| slider 交互全自实现:点击跳转(按在滑块上不跳值、按轨道把最近滑块跳过去)+ LButtonDown 起 `XEle_SetCapture` 拖拽 + MouseStay/MouseLeave 自跟踪 hover(拖完移开光环即消失, 与 shadcn 一致)+ 方向键(水平 ←→ / 垂直 ↑↓; 连续 ±5%、有 step 时一步)/PageUp·PageDown(10 步)/Home/End; `inverted` 反转取值方向; 禁用 = `SetDisabled`(鼠标与键盘一起关); `OnValuesChange` / `OnValuesCommit` 对应 onValueChange / onValueCommit | — + 第八节 144 |
| tabs 列表 rounded-lg p-[3px] h-8 bg-muted,标签等宽 rounded-md,未选 text-foreground/60、选中 bg-background(暗色 `bg-input/30 + border-input`, 否则白色卡片在暗色列表上变成一块更黑的方块);line 变体透明列表 + **gap-1** + 底边 h-0.5 下划线(换行时贴**当前行**底边);标签 text-sm **font-medium(500)≈中文常规粗细,勿用 Bold(700)**(实测对比);行内图标 size-4 + gap-1.5 + `has-data-[icon=inline-start]:pl-1`;单标签禁用 = 文字/图标 50% 预混 + 不响应点击/方向键;focus-visible 环画在内部焦点标签上(整块列表一个 Tab 站点, ←/→/↑/↓ + Home/End + Enter/Space, 见第八节 155) | tabs.tsx + 第八节 155 |
| tooltip = bg-foreground rounded-md px-3 py-1.5 text-xs text-background + size-2.5 旋转方块箭头(**只画露在气泡外的三角**, 见第八节 89/156),延迟 0、移出后 300ms 内跳过延迟;四方向(side)+ align + sideOffset, 放不下自动翻对侧并夹进工作区;内容可带右侧快捷键块(Kbd, 此时 pr 收到 6);键盘焦点进入也显示, 鼠标点击产生的焦点不显示;窗口为异型透明,气泡外真透明 | tooltip.tsx + 第八节 156 |
| select 触发区 h-8 rounded-lg border-input **bg-transparent(放卡上 = Card 色)** + 右侧 chevron-down 用 **text-muted-foreground 全色**(非前景半透明);触发区对滚轮无响应(`EnableEvent_XE_MOUSEWHEEL(false)`);弹层 = **item-aligned 定位**(盖住触发元素, 选中项正对触发位置, select.tsx `position="item-aligned"`);弹层条目 = p-1 容器 + 条目 h-7(28) rounded-md hover:bg-accent + 选中项右侧 Check 图标 + 打开时自动高亮选中项;ring-1 描边画在 `XE_PAINT_END`(内缩 1px, 见第八节 30);内容高度自适应(≤10 项不出滚动条);**弹层不做进场淡入**(上游是 fade-in-0 100ms, 整窗 alpha 在这个异型窗上会概率性冻结 → 弹层看不见还吞点击, 见第八节 30/69);**aria-invalid** = 红边框 + ring-3 destructive/20(暗色 /40), 见上表 | select.tsx + combobox.go(弹层接管路线见第八节 30) |
| menu 弹层 = **阴影弹窗**(rounded-lg 由阴影窗 nAngeleSize 逐像素画出 + shadow-md 投影, 官方 DrawMenu 同款, 见第八节 36/64) + bg-popover + ring-1 ring-foreground/10(内缩 1px); **p-1 = 菜单 SetBorderSize(4)**(默认 3); 进场**不做动画**(与原生菜单一致, 弹出即全内容 —— 整窗 alpha 会透出背景文字、只淡内容会先出一帧空白面板, 两种都闪, 见 99; 位移/缩放会崩, 见 64); 条目 h-7(28) rounded-md px-1.5, hover/focus bg-accent + 文字变 accent-foreground; destructive 红字 + hover bg-destructive/10(dark /20); disabled 文字 50% alpha 不响应; **前置图标 `MenuIcon`**(16px, left-6, 文字从 28 起, 同 shadcn 条目内联 SVG); 勾选/单选选中项右侧 Check 图标(right-2 size-4), **未选中也保留右侧 24(pr-8)**, 单选组内互斥; 分组标签 text-xs muted; 分隔栏 1px bg-border **全宽**(-mx-1; 原生槽高仅 4px, shadcn my-1 的 9px 不可达, 已知差异); 快捷键右对齐 text-xs muted; `MenuInset` 对普通条目 / `Label` / `Sub` 都可用(pl-7, 文字左缘 28); 弹层宽 = `Width(w)` 显式指定, 否则 max(触发区宽, 128)(shadcn `w-(--radix-dropdown-menu-trigger-width) min-w-32`); 无触发区的右键菜单 = 内容自适应 + 128 下限; 子菜单 = 内容自适应 + min-w-96; **所有条目宽度按自绘字体显式算**(原生菜单字体与自绘 14px 不一致, 留自然宽会裁长文本, 见第八节 99); **右键触发区 `NewContextMenu`** = 窗口级右键事件 + 触发区矩形命中(含子元素), `SetEnabled` / `OnOpenChange` | dropdown-menu.tsx + context-menu.tsx + menu.go(原生菜单皮肤, 见第八节 36/64/99) |
| table 表头 = h-10(40) px-2, 文字 foreground, 只有一条 1px 底边(border);数据行 = p-2 + text-sm(36), 行 `border-b`, **最后一行不画**;行悬浮 `hover:bg-muted/50`, 选中 `data-[state=selected]:bg-muted`;表体无外框/无圆角/无纵向网格线(外框交给 Card), 容器 `overflow-x-auto`, `caption-bottom text-sm`;自适应列宽按**元素当前宽度**算, 由组件订阅元素尺寸事件自愈(见第八节 60) | table.tsx + table.go(原生 List 接管路线见第八节 40) |
| **data table 列头**: 可排序(或可隐藏)列的表头**点击弹菜单**(升序 / 降序 / 隐藏), 不平铺直接排序;菜单**锚定在该列表头正下方**(按下后拖到别处松开也不会跟鼠标跑, 见 174);升序降序是 `Menu.Check` 勾选项, **再点当前项 = 取消排序**(保留"升→降→取消"的能力), 平局保持原顺序;图标随状态 `arrow-up-down`(muted)/`arrow-up`/`arrow-down`;排序或筛选变化都**回到第一页**(autoResetPageIndex);**Shift+点击表头** = 直接把该列加入/循环多列排序(升 → 降 → 取消), 不弹菜单, 次键箭头后有次序数字(2、3…);菜单本身固定单列排序;可交互表头有**悬停 / 按下底色**, 且可**拖拽表头列换位**(4px 起判, 目标边界画主色竖线; 顺序在列显隐之间保留, 见 176) | shadcn data-table 的 `DataTableColumnHeader`(ArrowUpDown/ArrowUp/ArrowDown + EyeOff)+ TanStack 默认 `enableSortingRemoval` / `autoResetPageIndex`;多列排序与列拖拽是 TanStack sorting/columnOrder 的桌面交互形态 |
| **data table 分面筛选**: 配了 `DataColumn.FilterOptions` 的列在工具栏多一个触发区(`⊕ 列名`), 点开是 **Popover + Command 面板** —— 顶部可搜索选项、条目左侧勾选框右侧**当前行数**、**点一项不关闭**(可连续多选)、列表底部"分割线 + 清除筛选"(点它只清选择, 面板不关);触发区右侧用**胶囊**列出已选(最多两个, 3 个起收敛成"N 个已选"), 有选中时壳铺 `secondary`、无选中是描边;列内多选是 **OR**、列间是 **AND**, 与关键字筛选叠加, 值比较是单元格**原始文本**相等;关键字或分面生效时工具栏出现"重置"(文字 + X)。工具栏顺序 = 定宽筛选框 → 分面触发区 → 重置 → 剩余空间 → 视图(输入框定宽, 触发区左缘固定, 胶囊只向右长) | tasks 示例的 `DataTableFacetedFilter`(Popover + Command 多选 + 已选 badge + 计数 + Clear filters 行) |
| **data table 筛选/分页**: 顶部输入即过滤(不区分大小写、子串匹配; 默认匹配所有列, 可用 `WithDataTableFilterColumns` 限定); 页脚 = 总行数(筛选中显示"筛选出 x / y 行") + 每页行数下拉(默认 10) + "第 x / y 页" + 首页/上一页/下一页/末页(到端点禁用); 行点击在无勾选模式下沿用 Table 的原生单选, `OnRowClick` 回传**原始数据**下标(勾选模式下改为组件自己命中并吞掉按下消息, 见 8.170) | tasks 示例的 DataTablePagination / DataTableToolbar; 行点击回调是本组件库扩展 |
| **data table 行勾选**: 最左勾选列 `w-8` 居中复选框; 表头复选框**普通点击 = 整页全选**(全选 / 不确定两态), **Shift+点击 = 整个筛选结果跨页全选/清空**(`SelectAllFiltered` / `AllFilteredSelected`); 勾选状态按原始行下标保存, 翻页 / 排序 / 筛选都不丢; 勾选行整行铺 `bg-muted`, 与 hover(`bg-muted/50`)叠色; **双击行内复选框 = 两次点击**(勾→取消 / 取消→勾, 第二次由 `LButtonDBClick` 补切并吞掉, 不进入编辑); 有勾选时页脚把总行数换成"已选 x / y 行"(y = 筛选后行数, x 统计全部筛选结果而不只是当前页) | tasks 示例的 DataTableRowSelection / TanStack 的 getIsAllPageRowsSelected + getFilteredSelectedRowModel; 跨页全选对应 selectAllRows 口径; 双击语义同 Tree 勾选框(见第八节 170) |
| **data table 虚拟滚动**(`WithDataTableVirtualScroll`): 一次展示全部筛选结果(不翻页); 表头复选框普通点击与 Shift+点击都 = 全部筛选结果(虚拟滚动没有"当前页"); 页脚只留总行数 / 已选统计, `SetPage` / `PageCount` / `OnPageChange` 不生效; 排序 / 筛选 / 换数据回到顶部(autoResetPageIndex 的等价语义), 勾选 / 编辑 / 列显隐 / 分面筛选与分页一致; 大数据量的排序 / 筛选仍在 Go 侧按全部行计算 | 本库扩展(shadcn 的虚拟滚动由 TanStack 组合, 无对应组件); 底层是原生 List 虚表 |
| **data table 列显隐**: 工具栏右侧"视图"钮(图标 + 文字、outline 8 高)弹 `Menu.Check` 列清单, 逐列勾选即显隐; 至少保留一列; 隐藏列的内容仍参与排序与关键字筛选 | tasks 示例的 DataTableViewOptions |
| **data table 富单元格**: `DataColumn.Render(p, DataCell)` 自绘徽章 / 状态图标 / 优先级 / 行操作钮, 该列**不写原生文本**; 行操作钮点击走 `OnCellClick(row, col)`(col = 可见列下标) | tasks 示例 columns.tsx 的 cell 渲染函数 |
| **data table 固定选项编辑**: `DataColumn.EditOptions` 非空时**单击**弹选择菜单(当前值带勾), 选中立即提交, **不出现文本输入框**; 空 = 普通文本编辑, `ReadOnly` 优先; 菜单**等左键松开后**再打开(避开这次点击的松开 — 底部行菜单向上翻时会直接点掉第一项), 锚定在单元格下方; 双击固定选项列不再进入文本编辑 | shadcn tasks 的状态/优先级是 Badge + 行操作菜单; 本库把枚举列的内联编辑收敛成选择菜单 |
| tree 行高 32(h-8), 视觉矩形左右内缩 4(mx-1), 圆角 rounded-md(6);行悬浮/选中都 bg-accent(同 select/menu 条目);每级缩进 16(size-4), 祖先层级画 1px 参考线(border 色);chevron size-4 text-muted-foreground, 收起朝右/展开朝下(= rotate-90 两态), 叶节点不画但留空槽位(文字全树对齐);条目文字 text-sm foreground;**勾选模式**(`WithTreeCheckable`, 构造选项): chevron 后 16px 复选框(与文字间距 4); 父节点全选 = primary 底 + primary-foreground 勾, 部分 = primary 底 + 减号半选, 未选 = border-input 空心(暗色 input/30 底); 点复选框只切勾选、不改变原生单选行(同 8.170 的吞消息), 双击复选框 = 两次切换(先勾后取消或相反)且吞掉原生展开/收起, 点父节点级联全部后代、祖先按直接子项聚合; `Checked/SetChecked/CheckedKeys/OnCheckChange`, `CheckedKeys` 为前序快照且包含全选的父节点; 半选父节点点一下落到全选, 再点取消整棵;shadcn 无官方 tree, 语义为社区通行实现 + 本组件库令牌 | shadcn 无官方 tree, 语义为社区通行实现; 原生 Tree 接管路线见第八节 42 |
| pagination 入口居中(justify-center), 页码 32 方形(ghost / 当前页 outline), 上一页/下一页 = chevron(size-4)+文字, 到边界按禁用画(50% 透明且不响应鼠标), 页数 >7 折叠成 首/末页 + 当前页±1 + 省略号(不可点);本实现用**恒定 7 槽位**(见第八节 53);三种形态 = 完整 / 简洁(只有页码) / 仅导航(只有上一页下一页, 配"每页行数"选择器), 另有只留 chevron(对应文字上的 `hidden sm:block`);**键盘**: 整块一个 Tab 站点, 进入后 ←/→/↑/↓ 在条目间移动(跳过省略号与到边界的导航项)、Home/End 首末、Enter/空格激活, 到端点不循环, 焦点环只画在**当前条目**上且仅键盘获焦时出现(见第八节 152/153) | pagination.tsx + 第八节 53/152/153 |
| steps 步骤条 = 横向等宽列(圆点居中、标题/说明在下方)或竖向固定行高(圆点在左、文字在右), 四态(已完成 = 主色实心圆 + 对勾、进行中 = 主色环 + 主色编号、未达 = 边框环 + 弱化编号、出错 = 破坏色实心 + X), 已完成段连接线取主色、指向出错步的连接线转破坏色; 状态按 `current` 推导, `StepItem.Status` 可单步覆盖; `StepItem.Percent`(1..100) 在进行中圆点上画主色进度弧(连接线不重复表达进度); `StepItem.Icon` 非空时圆点里画该图标(见 `xui.Painter.Icon`, 支持 `icons.IconXxx` / path / 整段 SVG), 四态都画它、代替序号/对勾/X, 状态只由圆点与连接线颜色区分(同 Ant Design / Element Plus 的 step icon 口径); **默认只有已完成步可点回**, 当前/未达步不响应鼠标(不给"点了没反应"的悬停暗示), `WithStepsClickable` 放开任意非禁用非当前步、`StepItem.Disabled` 单步禁用但保留显示与键盘浏览; `WithStepsSize(StepsSmall)` 用独立几何/字号表(圆点 20、标题 18/说明 14), `WithStepsInitial(n)` 只偏移显示编号; 整块一个 Tab 站点, ←/→/↑/↓ 在**所有步骤**间移动焦点(未达步可只读浏览, 光标可见)、Home/End 首末、Enter/空格激活可点的聚焦步; **按任意键即切键盘焦点态**, 鼠标点过后直接按方向键也能立刻看到焦点环(否则内部焦点在动、画面不动, 用户会认为键盘失效); 焦点环画在圆点外侧(元素四周预留 3px); 环/圆点用两层实心 `FillEllipse`(同 ColorPicker), 挖洞用 `SetBackdrop` 指定宿主底色(默认 `Card`) | 社区 stepper 通行语义 + Ant Design Steps(shadcn 无官方组件) |
| timeline 时间轴 = 每项一个 LayoutEle(内容槽) + 条目自绘轴线/节点: 轴线 2px `border`; 竖向 start 轴线在左、内容在右, end 轴线在右、内容右对齐, alternate 轴线居中、左右交替(第一条在左); 水平 start 轴线在顶、内容在下, end 轴线在底、内容在上, alternate 轴线居中、上下交替(第一条在上), 各条等分容器宽度; 水平条目高 = 内容高 + 上下内边距, 全部等于共享轴线推出的一条总高, 尺寸变化时实测重排; 竖向圆点对齐条目**第一个可见子项**的垂直中心, 条目间距 = 非末条的底部内边距(24, `WithTimelineGap`), 轴线画满内边距跨过间距; 节点默认实心圆点 10px `primary` + 7px 宿主底色光环, `TimelineOutlined` 改空心环, icon = 16px 自定义图标, loading = 16px 运行时旋转指示器; 节点色支持 primary / destructive / muted 与自定义主题色函数, 图标与加载态同样取节点色; 内置时间(text-xs muted)/标题(text-sm medium)/说明(text-sm muted 单行省略)三个 Label, 也可继续往条目里加 Label / Item / Card; **首条不画上游线、末条不画下游线**; 轴线/节点每帧用宿主底色擦除(默认 `Card`, `WithTimelineBackdrop`/`SetBackdrop` 跟随宿主) | 社区 timeline 通行语义 + Ant Design Timeline(shadcn 无官方组件) |
| collapsible: 根容器竖向流(触发区 + 内容区); 触发区是普通按钮(可绑定任意 Button), 点击切换, 展开态走 Button 的 aria-expanded 视觉(bg-muted); 内容收起时隐藏且不占布局, 展开时高度按内容自撑; **本实现**展开/收起带 200ms 高度过渡, 收起动画结束后才隐藏内容(radix presence 语义) | collapsible.tsx + radix CollapsibleContent 的 `--radix-collapsible-content-height` 动画示例 + 第八节 97 |
| tooltip 绑定元素自身有弹层(如 ComboBox)时, 弹层打开要主动调 `HideTooltip(hEle)`——弹层窗口接管鼠标后触发元素收不到 MouseLeave | tooltip.go / combobox.go |
| scroll area: 页面/长内容滚动用 `ui.ScrollArea`(原生 LayoutFrame 载体, 见第八节 66): **只做纵向**(内容区恒铺满视口宽); 滚动条**鼠标移入容器才显示、离开淡出**(同 radix 的 `type=hover`, 150ms; 用元素透明度实现, 不改变布局宽度)、10px 轨道 + 8px 圆头滑块(`rounded-full bg-border`)、轨道透明; 滑块悬停**不换色**; 滚轮悬停任意子元素都滚动(原生控件如 List 自吃 —— 嵌套滚动常态); 该规格统一在 `scrollbar.go` 的 `styleScrollBar` + `showScrollBarOnHover`, Table / Tree / ComboBox 弹层 / 多行 Input 同用(**滑块配色与容器不同**: 这四个载体在原版里是浏览器原生滚动条, 用"前景色淡化 + 悬停加深", 见第八节 21) | scroll-area.tsx(w-2.5 p-px `rounded-full bg-border`; radix `type` 默认 hover 淡入 —— 本实现的差异见"已知差异"表) |
| **disabled:opacity-50 是组级透明**:组件自绘的**形状**(壳体/轨道/勾/滑块/圆点)**逐层各乘 50%**(`ScaleAlpha`, 见第八节 61);**按钮文字/图标用"前景色×卡片色对半混"预混后绘制**(等于整组 50% 合成; 亮色页上禁用实心钮白字仍=255, 暗色下才会正确变暗);不要直接给文字加 alpha —— 文字叠在已半透明的壳体上会二次透出按钮底色。复选框/单选框的标签保持原色, 输入框禁用文字同样预混后交给原生 Edit。radio 的 Label 不跟随禁用 | button/checkbox/radio/tabs/switch/slider/input.tsx |
| **input 内嵌附加件**(input-group): 前置图标 size-4(16) / 前后缀文字(muted, 与输入同字号) / 清除钮(×)与密码显隐钮(眼睛) size-6(24) 幽灵钮 + 圆角 radius-5 + hover:bg-accent / 计数 `n/max` text-xs(12), 超上限变 destructive; 全部**自绘在壳体内部**, 不建子元素(子元素会被原生 Edit 的绘制盖住), 文字区左右内边距按附加件实际宽度算(`syncPadding`), 内容为空时清除钮不画但**留位**(打字过程中文字不左右跳) | input-group 示例 + 第八节 78/79 |
| **input group 复合 addon**: addon 内容可以是多个控件(图标 + 文字 + 按钮 + Kbd); 含按钮的 inline addon 整体向容器边缘外扩 5(-0.3rem), 只含 Kbd 外扩 2(-0.15rem), **每个 addon 只外扩一次**(不按按钮个数累加); InputGroupButton 的 xs/sm/icon-xs/icon-sm 只改高度(24/28), 字号固定 text-sm; 下拉菜单按钮默认 icon-xs, 点击弹出菜单(原生模态) | input-group.tsx 的 addon cva(has-[>button]:ml/-mr、has-[>kbd])+ InputGroupButton size + 文档 "下拉菜单" 示例; 见第八节 165 |
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
| 多层模态(对话框里再开 AlertDialog):同一宿主窗口维护打开顺序的模态栈,**只有栈顶**响应 ESC / Tab / 遮罩点击;后打开的弹层 z 序更高,关闭上层后焦点还给下层最近聚焦的控件 | 本组件库实现(radix 用嵌套 portal + 各层 DismissableLayer,语义一致) |
| **中文 500 字重 ≈ 常规**: Label 默认常规、按钮文字常规(用粗体字形实现 500 会明显偏粗); 标签比描述"看着粗"是**颜色**差(foreground 近黑 vs muted 灰), 实测笔画同宽 | label.tsx 500 + 实测像素 |
| 令牌表 nova 预设 neutral 色板(oklch 已换算 hex) | index.css :root/.dark |
| **字体单位**: xcgui 字体 size 是 pt 不是 px, FontHandle 统一按 px×0.75 换算(14px→11pt); 按 px 直接当 pt 会让全局文字放大 1.33× | font.go(见第八节 31) |
| calendar 卡片 = bg-popover rounded-lg + ring-1 ring-foreground/10(popover 皮肤; 弹层里用 `WithCalendarFlat` 免双层边框); 格子 = `--cell-size` 方格(设计值 2rem=32) + rounded-md(6); **默认尺寸 278×327 = 设计基准 ×1.12**(格子 36 / 字号 16), 且所有几何(内边距/格子/标题/星期行/页脚/字号)都按**元素宽度等比缩放**, 高度决定 6 行之间的行距(多余空间均摊) —— 元素拉大时内容一起变大, 不会只剩空白; 标题行高 = 格高, 年月**两个热区**居中(font-medium + chevron-down, 悬停/展开 bg-accent), 左右各一个 ghost 翻页钮(size-8, hover bg-accent, chevron 16); 星期行 0.8rem muted; **固定 6 行**(月份切换卡片高度不跳, 弹层尺寸恒定); 选中 bg-primary + text-primary-foreground, 今天 bg-accent, 悬停 bg-accent(150ms 过渡), 非本月 text-muted-foreground; 周起始日默认周一(中文习惯), `WithCalendarWeekStart` 可调 | calendar.tsx(react-day-picker v9 的 classNames) + calendar.go |
| calendar 年月下拉 = 点标题热区弹出自绘列表(月 12 项 / 年 = 视图年±50 可配), 面板 bg-popover rounded-md + ring, 行高按"12 行塞满可用高度"自适应(16..26), 当前值右侧 check, 超出可视高度时滚轮滚动 + 2px 细滚动条, 面板内点击=取值、面板外点击=关闭、点另一个标题热区=直接切换; 只改显示月份不动选中日期(react-day-picker `captionLayout="dropdown"` 语义) | react-day-picker 的 dropdown 形态 + calendar.go |
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
| **tag input**(shadcn 无官方组件, 语义由本库定死): 外壳 = Input 同款 h-8(32) 圆角壳 / 焦点环 / 校验环 / 禁用态;内部是 h-5(20) 的 Badge 胶囊(text-xs、secondary 底 + 右侧 ×)与等高的裸 Edit;胶囊排满一行换行, 外壳按行数自撑(行高 22 / 行距 4 / 上下内边距 5), 输入区不足 48 就整行挪到下一行;回车提交当前段, 输入中遇到分隔符(默认 `,` `，` 换行)立即把前面各段切成标签、最后一段留在输入框;trim 首尾空白、空串忽略、默认精确匹配去重(重复静默忽略), 到上限 / 校验器拒绝时输入文字保留不清空;输入框为空时 Backspace 删末个标签, 非空按原生删字符;点 × 删除对应标签(双击也要命中), 点空白 / 删完都把焦点交回输入框;禁用 = 整组 50% + Edit 关焦点与键盘;`OnChange([]string)` 只由交互触发, 程序化改值不广播;载体是"自绘外壳 + 一个 `SetRectEx` 定位的裸 Edit 子控件", 量高必须延迟到下一轮消息(见第八节 187) | 社区 tag input 通行语义(Enter / 分隔符提交、Backspace 删末个、× 删除) + 本库自定(去重 / 上限 / 校验 / 换行 / 回调边界) + 第八节 187 |
| **status bar**(shadcn 无官方组件, 语义由本库定死): 窗口底部整宽横条 h-7(28) `bg-card` + 顶边 1px `bg-border`, 左右内边距 8、槽间距 2; **左/中/右三槽都返回 LayoutEle, 可放任意控件**, 左右槽各 Weight 1、中槽 Auto(右槽开头一段 Weight 撑子把条目顶到行尾) ⇒ 中槽按**整条**中心对齐(右下抓手占 16 会让它左偏半个抓手宽), 不是"左右剩余空间"的中心; 内置条目 h-5(20) rounded-sm text-xs, 行内图标 12 + gap 4 + 左右内边距 6, 静止透明底 + muted-foreground 字, hover 150ms 过渡到 bg-accent + foreground 字(`xui.Hover`/`Fade`)、按下叠一层 accent/50, 禁用 50% 字色且不响应 hover/点击, active = primary 实心底 + primary-foreground; 点击在 `LButtonUp` 触发, 程序化 `SetText`/`SetIcon` 不触发; 条目与抓手默认绑**朝上**的 Tooltip(可用 `WithStatusBarTooltip` 注入外部管理器), 提示文字为空即解绑; 槽间可用 `NewStatusBarSeparator` 画 12 高 1px 竖线; **尺寸条 = 右下角 16×16 的 NWSE 抓手 + 可选实时尺寸读数**: 抓手是贴窗口右下角的方块(外层套一个整条高的 `LayoutEle`, 用 `SetAlignV(Bottom)` 把它压到底; 条的右侧内边距在抓手启用时收到 0, 否则抓手会被 8 内边距推离窗口角), 线条在方块内再内缩 3; 按住抓手拖改**宿主窗口**大小(不做内容缩放/zoom), 拖动以屏幕**物理**坐标增量 ÷ 窗口 DPI 换算成逻辑像素、抓手上 `SetCapture` 跟手(抓手会随窗口缩放移动, 元素坐标不能当基准), **拖动期间抓手自己的气泡被压住、收手时立即收起**(被捕获的元素收不到 MouseLeave, 不压的话气泡会在缩放中反复冒出并停在旧位置, 见 DESIGN-PITFALLS 188), 最大化态与缩到下限(默认 320×240, 可配)都不动作; 读数是 `W × H` **逻辑像素**, 跟宿主 `WM_SIZE` 实时刷新; 抓手可 `WithStatusBarGrip(false)` / `SetGrip` 关掉, 读数可 `WithStatusBarSizeReadout` / `SetSizeReadout` 开关; 条目是自绘 Element, 不进 Tab 焦点链(同第八节 157) | 桌面状态栏通行形态; 三槽权重与中槽对齐口径、抓手 DPI 换算/下限、尺寸读数、点击与提示边界均由本库定死(无官方 tsx) + 第八节 157/185/186 |

### 变体/尺寸与上游的映射、已知差异

对照 shadcn 官方源码。**令牌名**要求与 CSS 变量逐字符对应(见 3.1),**组件变体名**是 Go API、不参与批量转译, 允许更直白, 但差异必须记在这里:

| 组件 | 映射 / 实现 | 说明 |
|---|---|---|
| Button variant | `ButtonPrimary` = shadcn 的 `default`(用 primary 色), 另有 `ButtonDefault` 别名 | 上游把这个变体叫 `default`; 其余 secondary/outline/ghost/destructive/link 同名 |
| Button size | `ButtonSizeXs/Sm/Default/Lg` = h-6/h-7/h-8/h-9(24/28/32/36); `ButtonSizeIcon/IconXs/IconSm/IconLg` = size-6/7/8/9 方形 | 与上游 8 档一一对应(实测吻合); 方形档 cx 传 0 即用预设边长; xs/sm/icon-xs/icon-sm 在按钮组内统一成 rounded-lg(见 86) |
| Button 行内图标/触发区态/加载 | `WithButtonLeadingIcon` / `WithButtonTrailingIcon`(对应 `data-icon=inline-start/end`, 间距 4/6、图标侧内边距收窄为 6/8); `WithButtonAlign(ButtonAlignStart/Between)` 对应示例里加在 Button 上的 justify-start / justify-between(整宽触发区文字靠左、尾随图标贴右); `WithButtonRounded` = rounded-full; `WithButtonExpanded` = aria-expanded; `WithButtonHasPopup` = aria-haspopup(按下不下沉); `WithButtonLoading` = LoaderCircleIcon + animate-spin(1s linear) | 上游的 `has-data-[icon=...]` / `aria-expanded` / `active:not-aria-[haspopup]` / spinner 示例;Go 侧无 asChild/Slot |
| ButtonGroup | `NewButtonGroup(hParent, WithButtonGroupOrientation(...))` + `NewButtonGroupText` + `NewButtonGroupSeparator`; 组宽高按内容自撑, 首尾外角、嵌套组 8px 间距 | 对应 button-group.tsx 的 `ButtonGroup/ButtonGroupText/ButtonGroupSeparator`(role=group);不适用部分同下表的 Slot/data-* 钩子 |
| Badge variant | default/secondary/outline/destructive/ghost/link 六档齐全 | 悬停样式恒生效(**shadcn 写作 `[a]:hover:*`, 只有包在链接里才生效**; Go 侧没有链接概念) |
| Badge 悬停 | `ghost` = hover 出 bg-muted, `link` = hover 加下划线 | 下划线手绘(无下划线标志位, 见下方"下划线") |
| Card | `WithCardSize(CardSizeSm)` = `--card-spacing` 4→3(16→12);`card.Footer(0)` 创建内边距跟随尺寸、内容已垂直居中的页脚;`card.TitleSize()` 返回标题字号(16 / sm 14) | 标题字号上游是 `text-base` + `font-medium`, 中文下 500 回落常规 → `NewCardTitle` 用 16 常规(不是粗体); sm 卡片标题要自己 `SetFontSize(card.TitleSize())`(构造器拿不到所属卡片); 页脚内容要放进 `footer.Content()`(那一层才有内边距) |
| Collapsible | `NewCollapsible(x, y, cx, cy, parent, WithCollapsibleOpen/Disabled/Duration)` + `Content()` / `ContentStack(space)` + `BindTrigger(*Button)`; 状态 `Open/SetOpen/Toggle/OnOpenChange` | 对应 collapsible.tsx 的 `Collapsible/CollapsibleTrigger/CollapsibleContent`; 根容器固定竖向流(上游根是任意 div); 内容高度带 200ms 过渡(上游只暴露 CSS 变量, 动画要自己加类); 示例根节点上的 `data-[state=open]:bg-muted` 未内置, 需要时在外面包一层带底色的容器 |
| AspectRatio | `NewAspectRatio(x, y, cx, cy, ratio, parent)` + `SetRatio` / `Ratio` / `Content()` / `ContentStack(space)` | ratio = 宽 / 高; `cy > 0` 或布局项高度声明成 Fill/Weight/Fixed 时由调用方管高度, 比例不生效(同 CSS 宽高都确定时 aspect-ratio 失效); 上游的 `className`(圆角/底色)映射为内容区自绘或元素背景, 组件本身不画任何东西 |
| Alert | `NewAlert(..., WithAlertVariant/WithAlertIcon/WithAlertTitle/WithAlertDescription)` + `Action()`; 标题/描述由组件内部持有 | 对应 alert.tsx 的 `Alert` + `AlertTitle/AlertDescription/AlertAction`; Go 侧是整体自撑高度的组件; 需要自定义文字样式时用 `Title()` / `Description()` 改字体后 `Refresh()`; 动作槽按内容宽(上游固定预留 pr-18=72, 本实现由动作实际宽度缩短文本列) |
| Empty | `NewEmpty(x,y,cx,cy,parent)` + `Header()` / `Content()` / `Body()`;`NewEmptyMedia(parent, icon, WithEmptyMediaVariant(...))`、`NewEmptyTitle`、`NewEmptyDescription` | 对应 empty.tsx 的 `Empty/EmptyHeader/EmptyMedia/EmptyTitle/EmptyDescription/EmptyContent`; media 变体 default/icon, default 用 24 图标、icon 用 40 圆角 bg-muted + 24 图标; 不画边框(上游默认 `border-dashed` 无宽度, 没有可见边框), 需要边框放进 Card; 上游 md:p-12 响应式内边距未做(统一 p-6) |
| Field | `NewField(..., WithFieldOrientation/WithFieldInvalid)` + `Label/Title/Description/Content/SetInvalid`;`NewFieldSet/FieldGroup/FieldLegend/FieldSeparator/FieldError` | 对应 field.tsx 的 `Field/FieldSet/FieldGroup/FieldLegend/FieldLabel/FieldTitle/FieldDescription/FieldSeparator/FieldError`; 无 responsive 取向与 DOM 级 label htmlFor; 横向 `Content` 默认 240 宽(嵌套布局只认 Fixed 宽, 见第八节 101), 用 `xui.ItemWidth` 覆盖; `FieldError` 单条错误默认单行(对 `Labels()` 的 Label 调 `SetAutoWrap()` 可折行), 多条去重并用 "•" 列表 |
| Dialog | `NewDialog(hWnd, WithDialogWidth/WithDialogCloseButton/WithDialogScroll)` + `SetTitle/SetDescription/AddFooter/BindTrigger/BindClose/Content/ScrollArea/Refresh` | 对应 dialog.tsx 的 `DialogContent` + `DialogHeader/Title/Description/Footer/Close/Trigger`; 无 compound 子组件拆分, 头部/页脚由组件内部持有; `WithDialogScroll` 对应文档的 Scrollable Content / With Sticky Footer; 缩放动画不做(见第八节 47) |
| AlertDialog | `NewAlertDialog(hWnd, WithAlertDialogSize/WithAlertDialogTitle/WithAlertDialogDescription/WithAlertDialogMedia/WithAlertDialogMediaTone)` + `AddCancel/AddAction/BindTrigger` + `CancelButton/ActionButton` | 对应 alert-dialog.tsx 的 `AlertDialogContent` + `Header/Title/Description/Footer/Media/Action/Cancel/Trigger`; `size` 是枚举 `AlertDialogSizeDefault/Small`; 媒体块由组件自绘(不建子控件); `AddCancel` 固定 outline、`AddAction` 的变体由调用方给(可用 `CancelButton().SetVariant(ButtonGhost)` 改变体) |
| Separator | `WithSeparatorVertical` 支持垂直 | 上游 `data-vertical:w-px` |
| Tabs | `NewTabsItems` / `TabsItem{Text,Icon,Disabled}`(行内图标与单标签禁用) + `WithTabsDisabledAt` / `SetDisabled` / `SetIcon` / `SetText`; `WithTabsVertical` 垂直(标签堆叠、左对齐、指示条在右侧); `WithTabsWrap` 横向换行(上游没有: 每行均衡、高度随行数增加、line 下划线贴当前行底边); `WithTabsActivation`/`WithTabsManual` 对应 `activationMode`; `WithTabsValues` + `ActiveValue`/`SetActiveValue`/`OnValueChange` 对应受控 value; `BindPanel` 按选中项显隐面板(对应 TabsContent 的显隐); 键盘: 整块一个 Tab 站点, ←/→/↑/↓ + Home/End + Enter·空格(第八节 155) | 上游 `data-[variant=line]:*` / `data-[orientation=vertical]:*` / `activationMode`; 换行与 `BindPanel` 是本组件库扩展; 未做 `forceMount` 与拖动滑块式指示器 |
| Tooltip | `Bind` / `BindTip(hEle, text, opts...)` + `WithTipSide`(上/下/左/右, 默认自动) / `WithTipAlign`(start/center/end) / `WithTipOffset`(sideOffset) / `WithTipDelay` / `WithTipShortcut`(内容右侧 Kbd) / `WithTipMaxWidth`; 全局 `SetDelay` / `SetSkipDelay`(默认 0 / 300ms); 受控 `Show` / `Hide` / `IsOpen` | 上游 `side` / `align` / `sideOffset` / `delayDuration` / `skipDelayDuration`; 上游一个 Tooltip 是"一个 trigger + 一个 content", 本组件库是**一个管理器绑定任意多个元素**(每个元素可各自配置方向/延迟), 因为桌面端控件是原生子元素; `TooltipProvider` 的共享延迟语义收进 `SetDelay`/`SetSkipDelay`; 未做 `disableHoverableContent` 与箭头 `arrowPadding` 单独配置 |
| Toggle | `ToggleVariantDefault/Outline`、`ToggleSizeDefault/Sm/Lg` = h-9/h-8/h-10(36/32/40)、`WithTogglePressed/Icon/Invalid` + `SetPressed/SetIcon/SetInvalid/OnChange` | 对应 toggle.tsx 的 variant/size; 组内条目按 px-3 + min-w-0 量宽(ToggleGroupItem 的覆盖); 原生 Check 按钮不响应空格/回车, 键盘切换由组件在 KeyDown 里补(与鼠标共用 OnChange); shadow-xs 不绘制(同其他自绘壳体) |
| ToggleGroup | `NewToggleGroup(hParent, WithToggleGroupVariant/Size/Spacing/Multiple)` + `AddItem(value, text, opts...)` + `Value/SetValue/OnChange/SetSpacing` | 对应 toggle-group.tsx; 单选(type=single)允许全部取消(radix 语义), 组级 variant/size 覆盖条目自己的同名选项(context.variant || variant); spacing 用逻辑像素(上游 spacing={2}=8px); 上游当前源码没有 orientation, 只做横向 |
| Slider | `WithSliderRange(min,max)` / `WithSliderStep(step)` | 对应 radix 的 min/max/step; 默认仍是 0~100 连续 |
| ComboBox 触发区 | `WithComboSize(ComboSizeSm)` = h-7 + `rounded-[min(--radius-md,10px)]` | 对应 select.tsx 的 `data-[size=sm]` |
| Calendar | `WithCalendarSelected/WeekStart/OnSelect` + `WithCalendarFlat`(弹层内嵌不画卡片); 尺寸 `CalWidth/CalHeight`(单月 278×325) + `CalFooterHeight`, 自定义宽度时用 `CalHeightFor(width)` / `CalFooterHeightFor(width)` 配高度(或 cy 传 0 自动); 年月下拉 `WithCalendarYearRange(from,to)`; 禁用 `WithCalendarDisabled(fn)` + `SetDisabled` + helper `CalendarDisableBefore/After/Weekend/Dates/Any`; 划线 `WithCalendarStrike(fn)` + `SetStrike` + `CalendarStrikeDates`; 页脚 `WithCalendarFooter(items...)` / `WithCalendarFooterToday` / `WithCalendarPresets(CalPreset(label,days,to)...)` / `AddFooterItem/Button/Text` / `ClearFooter` / `FooterRect`; **范围/多月/周数**: `WithCalendarRange` + `WithCalendarSelectedRange(from,to)` + `WithCalendarOnRangeSelect` + `SelectedRange/SetRange/RangeMode`, `WithCalendarMonths(n)`(1~4), `WithCalendarWeekNumbers`; 尺寸 helper `CalWidthFor(months)` / `CalWidthForWeekNumbers(months)` / `CalHeightForMonths(width,months)` / `CalFooterHeightForMonths` | react-day-picker 的 props(mode/numberOfMonths/showWeekNumber);固定 6 行 + 等比缩放 + 页脚换行是本实现决策 |
| DatePicker / TimePicker | `WithDatePickerPlaceholder/Value/Invalid` / `WithTimePickerPlaceholder/Value/Invalid` + 运行时 `SetInvalid`; 程序化开关 `Open/Close/IsOpen`; DatePicker 另有 `Clear/SetValue/Text`; **范围**: `WithDatePickerRange` + `WithDatePickerRangeValue` + `Range/SetRange/OnRangeChange/RangeMode`; `WithDatePickerPresets` / `WithDatePickerYearRange` / `WithDatePickerDisabled` / `WithDatePickerEditable`(可输入触发区, 配 `ParseDateText(s)` 解析) | 选项名带组件前缀(组件同在 package ui); 弹层骨架 `popup.go` 是两个组件的内部实现, 随组件一起拷贝 |
| 校验失败态(aria-invalid) | `WithButtonInvalid` / `WithInputInvalid` / `WithComboInvalid` + 运行时 `SetInvalid(bool)` | 选项名带组件前缀, 因为组件同在 `package ui`(Button/Input/ComboBox 三处语义相同); 详见语义表 |
| Input 内嵌附加件 | `WithLeadingIcon` / `WithPrefixText` / `WithSuffixText` / `WithClearButton` / `WithPasswordToggle`(配合 `SetPasswordVisible(bool)` / `PasswordVisible()`) / `WithCounter(max)`; 步进器 `WithStepper(min,max,step)` + `SetNumber/Number/OnStep`(`WithStepperArrows` 换成上下三角 = 原生 number 输入框形态, `WithStepperAutoHide` 让按钮只在悬停/聚焦时出现; 悬停底色/按下保持底色 + 箭头下沉 1px, 与 −/+ 一致) | shadcn 的 input-group 是"输入框 + 兄弟 addon"的复合结构(`InputGroupAddon` 里还能放按钮), 我们是**壳体内部自绘**的简化形态(不建子元素): 前置图标 + 前后缀文字 + 两个 ghost 钮(清除/眼睛) + 计数; 计数放在壳内右侧(上游示例是输入框**下方**右对齐的一行); stepper 上游没有官方组件, 语义取社区通行形态; 显隐切换需重排(见第八节 84) |
| Textarea | `NewTextarea(x, y, cx, cy, parent, opts...)`(`opts` 与 Input 相同) + `SetValue` / `Refresh`; `cy` = 最小高度(0 = 64) | 对应 textarea.tsx: Input 的多行预设, 上下内边距 8(py-2)、min-h-16 + **field-sizing-content 自动长高**(见第八节 166); 行数取原生 Edit 的显示行数, 所以换行/缩放/换字体都对得上; 内容永远装得下 → 不挂滚动条; 需要固定高度或溢出滚动条时用 `Input` + `WithMultiline`(高度写在创建参数里) |
| Input Group | `NewInputGroup(..., WithInputGroupInvalid/Disabled)` + `Addon(InputGroupInlineStart/InlineEnd/BlockStart/BlockEnd)` + `Input/Textarea/Search`; addon 内容 `Text` / `Icon` / `Button` / `IconButton` / `Kbd` / `Spinner` / `MenuButton(text, icon, build)` / `Content()`; `SetInvalid/SetDisabled/Focus` | 对应 input-group.tsx 的 `InputGroup/InputGroupAddon/InputGroupButton/InputGroupInput/InputGroupTextarea`: 按钮默认 ghost + xs、字号跟正文、圆角按 InputGroupButton 语义(147); 一个 addon 可放多个控件, 外扩量按 addon 整体算(165); `MenuButton` 用原生菜单(模态)承载下拉, 上游是 radix 非模态菜单 |
| Card 头 | `ui.NewCardHeader(parent, space)` + `TitleStack` / `ActionStack` | 对应 `CardHeader` + `CardAction`(上游是 grid 1fr_auto); 动作区跨两行贴右上 |
| ScrollArea | **本组件库形态**: 原生 LayoutFrame 载体 + 主题化滚动条(10px 轨道 + 8px 圆头滑块, 与 Table/Tree/ComboBox/多行 Input 共用 `scrollbar.go` 的统一收敛); 显隐同 radix 的 hover 淡入(150ms 淡入淡出); 只做纵向(上游 ScrollBar 有 horizontal 档); 隐藏页恢复可见后要延迟重排一次页面, 否则子控件命中区停在旧滚动位置 | 页面滚动容器; 语义与踩坑见第八节 66 |
| Menu 条目缩进 / 图标 / 右键触发 | `MenuInset` = shadcn 的 `inset` prop(`data-inset:pl-7`, 文字左缘 28; 条目/Label/Sub 都支持); `MenuIcon(path)` = 条目内联前置图标(16px, left-6); `NewContextMenu(hEle, build)` = `ContextMenu` + `ContextMenuTrigger`(右键触发区, 每次右键重建菜单, `SetEnabled`/`OnOpenChange`) | inset 用于和带前置图标的条目文字对齐; 高亮底仍是整行; 右键触发走窗口事件做区域命中(元素右键不冒泡), 见第八节 99 |
| NativeSelect | `NewNativeSelect(x,y,cx,cy,hParent, values, WithNativeSelectPlaceholder/Size/Value/Invalid)` + `Value/SetValue/OnValueChange` | 对应 native-select.tsx(浏览器原生 `<select>` + 外壳): 弹层完全是原生 ComboBox 弹窗(与 NewComboBox 一致), 组件只补 `value` 字符串语义(`<option value>` / `onChange(e.target.value)`); 未选中时 Value() 返回空串, SetValue 匹配不上会取消选中 |
| Select | `NewSelect(..., WithSelectPlaceholder/Size/Items/Groups/Value/Invalid/Disabled)` + `Value/SetValue/Selected/SetItems/SetGroups/OnChange/Open/Close/IsOpen`; 条目 `SelectItem{Value,Text,Icon,Disabled}`、分组 `SelectGroup{Label,Items}` | 对应 select.tsx(radix, item-aligned): 弹层盖住触发区、选中项正对触发区(两种盒子高度按**中心**对齐, 同 ComboBox 弹层的推导); 宽 = max(触发区宽, min-w-36); 键盘 ↑↓(跳过禁用)/Home/End 移动高亮(焦点即高亮 bg-accent), Enter/空格 选中, ESC 关闭并把焦点还给触发区; 打开即把焦点放到选中项(radix 语义); 点击选中同值也通知(同 radix onValueChange); 与 ComboBox 的取舍: 弹层自持(可富条目), 原生弹窗行为不复刻 |
| InputOTP | `NewInputOTP(..., WithInputOTPLength/Pattern/SeparatorAfter/Value/Invalid/Disabled)` + `Value/SetValue/Clear/Focus/SetInvalid/SetDisabled/OnChange/OnComplete`; 预置过滤 `InputOTPPatternDigit/Alphanumeric/Any` | 对应 input-otp.tsx(底层 input-otp 包): 键入打进已填格 = **替换**并前进一格(同上游), 非法字符不入格; ←/→/Home/End 移动, Backspace 清当前格、空格时退格清前一格, Delete 清当前格; Ctrl+V 从当前格起分发粘贴(非法字符跳过, 焦点停最后写入格); 全满触发 OnComplete; 文字居中由原生 `SetTextAlign(Center|Center_V)` 负责(不要用 border 挤位置, 见第八节 110), 只有空格子用对称 border 把原生 caret 顶到格心; 上游的"跨格拖动选择"没有对应物(每格独立 Edit) |
| Accordion | `NewAccordion(..., WithAccordionType(AccordionSingle/AccordionMultiple))` + `AddItem(value, title, WithAccordionItemDisabled)` + 条目 `Open/SetOpen/Toggle/SetDisabled/Content/ContentStack/Trigger`; 根 `Value/SetValue/OnChange` | 对应 accordion.tsx(radix): 触发区文字左对齐(上游无水平内边距; 我们留 4px 给内嵌环带, 见第八节 112)、箭头贴右、触发区高 40(py-2.5 + text-sm), hover:underline 手绘 1px 线(无过渡), focus-visible 环 rounded-lg; 箭头是 ChevronUp/Down 两枚图标互换(上游也是互换, 无旋转动画); 条目间 1px 分隔线(Item 的 not-last:border-b, 末项没有); 内容底部 pb-2.5 用末尾的透明占位表达(挂在开合时、高度扣掉行间距, 见第八节 111); single = 单开互斥且允许全部收起(radix type="single" collapsible), multiple 各自独立; disabled 只禁触发区, 已展开内容保留 |
| Popover | `NewPopover(hTrigger, WithPopoverContent(build)/Width/Height/Padding/Gap/Side/Align/KeepOnBlur/OnOpenChange)` + `Open/Close/SetOpen/IsOpen/Content/Refresh`; `PopoverHeader/Title/Description` 助手 | 对应 popover.tsx 的 `Popover/PopoverTrigger/PopoverContent`(Header/Title/Description); 内容宽铺满、高自撑(空内容保留 p-4); 触发区可以是任意元素(Button 走 BnClick, 普通元素走 LButtonUp); 弹层窗口是自建阴影窗, 无 zoom/slide 动画; 内容里嵌会自己弹窗的控件(ColorPicker / Select / DatePicker)时 `WithPopoverKeepOnBlur(true)`, 关闭改由宿主窗口的 `WM_LBUTTONDOWN` 负责(见第八节 184) |
| HoverCard | `NewHoverCard(hTrigger, WithHoverCardContent/Width/Padding/Side/Align/Delay/OnOpenChange)` + `Open/Close/SetOpen/IsOpen/Content/Refresh` | 对应 hover-card.tsx; 默认 `openDelay=700ms` / `closeDelay=300ms`; 非激活窗口, 不抢焦点; 内容里的子元素都会挂 MouseLeave, 离开后按指针位置决定是否真正关闭 |
| Avatar | `NewAvatar(x,y,cx,cy,parent,fallback,WithAvatarSize/Image/ImageData/Ring/Badge)` + `SetImage/SetImageData/SetFallback/SetBadge`; `NewAvatarGroup(parent, WithAvatarGroupSize)` + `Add(fallback, opts...)` / `AddCount(n)` | 对应 avatar.tsx 的 `Avatar/AvatarImage/AvatarFallback/AvatarBadge/AvatarGroup/AvatarGroupCount`; 图片走本地文件或 `image.Image`(png/jpeg/gif), 圆形蒙版在 CPU 侧生成; 徽标画在头像内右下角(圆环内缩 2px, 避免被父元素裁剪) |
| Command | `NewCommand(x,y,cx,cy,parent, WithCommandPlaceholder/EmptyText/OnSelect/OnChange/OnEscape)` + `AddGroup(label)` / `AddSeparator` + `CommandGroup.AddItem(value,text, WithCommandItemIcon/Shortcut/Keywords/Disabled/OnSelect)` + `Focus/Query/SetQuery/Clear/SetPlaceholder` | 对应 command.tsx(cmdk) 的基础形态; 过滤为不区分大小写的包含匹配(可配 keywords), 不是 cmdk 的 command-score; 列表高度 = cy - 36(cy 传 0 = 336); `CommandDialog` 用 `ui.NewDialog` + `Command` 组合(无单独的 CommandDialog 类型) |
| Toast | `NewToaster(hWnd, ...)` + `Show(ToastOptions)`(标题/描述/类型/图标覆盖/动作/关闭按钮/时长/`OnClose/OnAutoClose/OnDismiss`) + `Success/Error/Warning/Info/Loading` 快捷方法 + `Promise(fn, ToastPromise{Loading, Success, Error})`; `WithToastPosition/Width/Offset/Gap/Duration/MaxVisible/RichColors/CloseButton`; 实例 `Update/Close/Handle` | sonner 的 `toast` API 的 Go 形态; 位置六种(四角 + `ToastTopCenter`/`ToastBottomCenter`), `richColors` 通过 `WithToastRichColors` / `ToastOptions.RichColors` 开启; `Loading`/`Promise` 的转圈图标由 `xui.Spin` 驱动(退出加载态要显式停表, 见第八节 154), `Promise` 的 fn 在 goroutine 跑、结果经 `xc.Auto` 回 UI 线程 |
| ScreenToast | `NewScreenToaster(hAnchor, ...)` + 与 Toast 相同的 `Show/Success/Error/Warning/Info/Loading/Promise/Update/Close`; `WithScreenToastWidth/Offset/Gap/Duration/MaxVisible/RichColors/CloseButton` | 屏幕右下角的独立置顶通知(不是窗口内 Toast): 每条一个 `Window_Transparent_Shaped` 小窗, 不占任务栏、不抢焦点; hAnchor 只用来选显示器、驱动过渡与随宿主销毁清理; 卡片排版/绘制共用 `toast.go`; 难点见第八节 177 |
| ScreenPopup | `NewScreenPopup(hAnchor, WithScreenPopupWidth/Height/Offset/Padding/Gap/Duration/Focusable/CloseButton/Header/OnOpenChange/Content)` + `Show/Hide/Toggle/IsOpen/FitContent/Content/Handle/Destroy` | 屏幕右下角的常驻**面板**(不是通知): 单个 `popoverPanel`(阴影窗)承载任意 xui 内容, 手动显隐、不堆叠, 默认不自动关闭(`WithScreenPopupDuration` 设显示后自动关闭, 不因悬停暂停); `WithScreenPopupHeader(标题)` 在内容上方自带标题 + 右上角 X, `WithScreenPopupCloseButton` 只要 X; **默认不抢焦点**(显示/点击都不激活, 内容拿不到键盘焦点), `WithScreenPopupFocusable(true)` 改为可点击激活: 显示仍不抢前台, 用户点弹窗后才被系统激活、内容可输入(失焦不自动关, ESC 收起); 开关状态(含 X / 超时)走 `WithScreenPopupOnOpenChange`; 显隐都走 100ms 淡入淡出(元素 alpha 与窗口阴影浓度一起变), 关闭淡出中重新 `Show` 会从当前透明度反向淡入; 自动高按内容量, 内容增删后调 `FitContent`; 随锚点窗口销毁; 难点见第八节 181/182 |
| Tray | `NewTray(hWindow, iconPath, tips, WithTrayMenu/WithTrayIconHandle/WithTrayLeftClick)` + `SetIcon/SetIconHandle/SetTips/Show/Hide/Balloon/Destroy/Visible` | 系统托盘图标 + themed 右键菜单; 默认左键还原并前台化绑定窗口; 托盘事件走 `AddEvent_TrayIcon`; 监听 `TaskbarCreated` 在 Explorer 重启后恢复; xcgui 托盘 API 是进程级全局状态, 同一进程只建一个; 细节见第八节 178 |
| Pagination | **本组件库扩展**: shadcn 的 Pagination 是纯展示组件, 省略号要手拼; 我们做了自动折叠(第八节 53) | 见 53 |
| Pagination(形态/键盘) | `WithPaginationSimple()`(只有页码) / `WithPaginationNavOnly()`(只有上一页下一页) / `WithPaginationHideText()`(只留 chevron); 键盘 ←/→/↑/↓、Home/End、Enter/空格 | 见 152/153 |
| DataTable | `NewDataTable(x,y,cx,cy,parent, []DataColumn{Title,Width,Sortable,Compare,Align,Render,Hidden,NoHide,FilterOptions,EditOptions,ReadOnly}, WithDataTableSearch/NoSearch/PageSize/PageSizes/FilterColumns/Selection/ColumnToggle/Editing/VirtualScroll)` + `SetRows/AddRow/Refresh/SetColumns`、`SetFilter/Filter`、`SetSort/Sort`、`SetSortKeys/SortKeys`、`SetPage/Page/PageCount`、`SetPageSize/PageSize`、`TotalRows/FilteredRows`、`Selected/SetSelected/ClearSelection/IsSelected/SelectedCount`、`SelectAllFiltered/AllFilteredSelected`、`SetColumnVisible/ColumnVisible`、`SetFacet/FacetValues/ResetFilters`、`OnSortChange/OnSortKeysChange/OnPageChange/OnFilterChange/OnFacetChange/OnRowClick/OnCellClick/OnSelectionChange/OnCellEdit` | 对应 shadcn data-table 文档(tasks 示例, TanStack Table): 排序键是 **DataSortKey 数组**(主→次; 列头菜单仍单列, Shift+点击表头建多列), 关键字筛选是工具栏输入框的子串匹配, 分面筛选是 `DataColumn.FilterOptions`(工具栏多选钮), 分页在页脚(可选虚拟滚动, 见下一条); 列头 = `Asc/Desc/Hide` 菜单 + 悬停底色 + 拖拽换位, 勾选 = `WithDataTableSelection`(普通点表头 = 整页, Shift+点 = 跨页全选筛选结果), 列显隐 = `WithDataTableColumnToggle`, 富单元格 = `DataColumn.Render`, 双击原地编辑 = `WithDataTableEditing`(Enter/失焦提交、Esc 取消; `OnCellEdit` 返回 false 拒绝提交; `DataColumn.ReadOnly` 排除该列, 自绘列也能编辑; `DataColumn.EditOptions` 非空时**单击**弹固定选项菜单 —— 状态/优先级这类枚举列不出现文本输入框); 未做行拖拽(见第八节 162~176 与差异表) |
| DataTable | **本组件库形态**: `NewDataTable` 自带工具栏/表格/页脚三段; shadcn 的 data-table 是"用户用 TanStack 组合"的模式, 没有对应组件 —— 我们把它收敛成一个可复用控件 | 见 162~176;底层用到 `Table.HeaderCellRect`(表头单元格矩形缓存)、`Table.RowHeight`(数据行高, 内联编辑反推单元格矩形)、`Menu.PopupAtElementRect`(按元素内子矩形锚定弹出, 见 174)与表头元素鼠标事件(见 176) |
| DataTable(未做) | 行拖拽 | 原生 List 没有行拖拽; 排序/筛选/分页后"显示顺序"与原始行顺序是两回事, 拖拽要回写业务数据并定义跨页/筛选下的落点语义, 不是组件能单方面决定的 |
| DataTable(模式) | 分页(默认) / 虚拟滚动(`WithDataTableVirtualScroll`) | 两种模式共用排序/筛选/勾选/编辑/列显隐/空状态. 分页: 表格体只装当前页(默认 10 行, 推荐 20~50), 页脚有页码/每页行数/翻页; 虚拟滚动: 一次展示全部筛选结果, 原生 List 只给可见行填文本, 页脚只留总行数/勾选统计, `SetPage`/`PageCount`/`OnPageChange` 不生效(排序/筛选/换数据回到顶部). 原生 List 的滚动绘制只跟可见行数有关, 两种模式帧时间相同; 1M 行时把全部行装进普通 List 会额外复制约 1GB 到原生适配器, 分页/虚拟滚动都不复制. 真正随总行数增长的是 Go 侧排序/筛选与业务数据本身 —— 50 万行以上请用 `DataColumn.Compare` 提供快速比较, 或把排序/筛选移出 UI 线程 |
| DataTable(排序口径) | 数字(允许 `¥$€£,` 与 `%`)按数值、英文与数字段按自然顺序(同 `localeCompare numeric:true`); **中文按 UTF-8 码位**, 不按拼音 | Go 侧没有浏览器中文拼音排序; 需要业务语义时用 `DataColumn.Compare` 自定义 |
| Resizable | `NewResizableGroup(x,y,cx,cy,parent, WithResizableVertical/Disabled/OnResize)` + `AddPanel(WithResizableSize/Min/Max)` + `AddHandle(WithResizableGrip)`; `Sizes/SetSizes/OnResize/SetDisabled/Disabled`; 面板是流式容器, 内容直接放进去 | 对应 resizable.tsx 的 `PanelGroup/Panel/Handle`; 未做 `collapsible/collapsedSize`(Enter 折叠)、`preserve-pixel-size`、F6 在分隔条之间跳焦点、面板/分隔条级 disabled; 尺寸只支持百分比(上游还能写 px/rem); 面板是容器而非独立元素, 没有 `data-panel` 之类的 DOM 钩子 |
| StatusBar | `NewStatusBar(x,y,cx,cy,parent, WithStatusBarHeight/Grip/SizeReadout/MinWindowSize/Tooltip)` + `Left/Center/Right/Bar/Grip/Tooltip/Height/GripEnabled` + `SetGrip/SetSizeReadout` + `AddLeft/AddCenter/AddRight`; `NewStatusBarItem(hParent,text, WithStatusBarItemIcon/Tooltip/OnClick/Disabled/Active/Width)` + `Text/SetText/SetIcon/SetTooltip/SetDisabled/SetActive/OnClick`、`NewStatusBarSeparator` | 本组件库自定扩展, 上游没有 status-bar; 抓手只改宿主窗口尺寸(浏览器式 zoom / 内容缩放不在组件边界内), 尺寸读数是逻辑像素; 三槽是流式容器而非 DOM children, 中槽按整条中心、右槽条目靠尾; 条目不参与 Tab 焦点链(见第八节 157) |
| 不适用 | `asChild`(Slot)、`data-slot`/`data-state`/`data-variant` 钩子、compound 子组件拆分、props 透传 | React/DOM 形态, Go 侧没有对应物 |
| 令牌 | `--chart-1..5` / `--sidebar-*` 已进 `xui.Theme`(Chart1..5 / Sidebar*) | chart 色板供你的应用取用(组件库不画图表); sidebar 令牌由 `ui.Sidebar` 使用; 机械转译新主题时两组一并填(运行时的主题测试有守卫) |

**下划线(Button link / Badge link / Accordion 触发区)**: xcgui 没有下划线文本标志位(`TextFormatFlag_` 里没有), 字体侧 `FontStyle_Underline` 在 D2D 文本路径下也不渲染(见第八节 110), 只能手绘: 宽度用 `xui.TextWidth` 量, y = 行盒中心 + 0.3em + offset(`underline-offset-4` 用 4px; Accordion 的 hover:underline 没有 offset, 用行盒底边 - 2px), 1px 线(`button.go` / `separator.go` / `accordion.go`)。实测(150% DPI): "Link" 下方 1 条 43 物理像素宽的横线, 紧贴文字下沿。

---

## 五、安装与更新

命令与开关的清单只有一份: **`xui --help`** —— 它由你手上那个二进制印出来, 不会超前也不会滞后(本文随数据源下发, 而 CLI 可能是上个 tag 的, 照本文敲一个新开关结果报"未知选项"时, 以 `--help` 为准)。日常只有三条入口: 新项目 `xui init -n <名字>`、追加组件 `xui add <组件...>`、刷新已装 `xui update`(省略组件名 = 已装的全部 + 已在的文档)。下面只讲清单之外的事: 每条约定为什么这样定、边界在哪。

- **文档也会跟着升级**:组件目录里的 `DESIGN.md`、`DESIGN-PITFALLS.md` 与(普通 `init` 写的)`AGENTS.md` 和组件源码**共用同一套基线判定** —— 没改过就随 CLI 刷新(措辞"更新: 版本升级"), 改过就保留(要 `-o` 才丢弃)。以前它们是"存在就永远跳过", 等于第一次 `init` 那一刻定版, CLI 后来修了文档里的错也传不到已有项目。`theme.go` 例外: 它就是给你改主题用的, 存在就一个字都不动, 加 `-o` 也不覆写。
- **文档可以单独点名**:`xui add DESIGN.md` 只拿文档(没跑过 `init` 的项目补齐它, 并连带配套的 `DESIGN-PITFALLS.md` —— 后者是本文第八节的完整正文),`xui update DESIGN.md` 只刷文档(那一份 + 配套文件)、不碰组件, 不存在的文档不创建(那是 `add` 的活)。**裸 `xui update`(不带组件名)= "把这个目录里 CLI 给过的都刷新"**: 已装组件 + 已经在的 `DESIGN.md` / `DESIGN-PITFALLS.md` / `AGENTS.md`(缺的文档不补也不提醒 —— 补齐是 `add`/`init` 的活)。唯一的例外: 老项目里 `DESIGN.md` 已在、拆分后才有的 `DESIGN-PITFALLS.md` 还没有时, `update` 会补上它(两份本来就是一份说明书, 不补的话第八节的索引全指向空处; 这不算"新增它没交付过的东西"), `--dry-run` 只预告。文档内容随**数据源**下发(与组件同源, 见下面的「下发的文档与组件同源」), 所以 `XUI_REGISTRY` 钉在哪个 tag, 拿到的说明就是那个 tag 的那一份。
- **默认不覆盖你改过的文件**:逐个比对内容 —— 本地没有则创建、内容一致则跳过、被改过则警告跳过;确认要丢掉本地改动时加 `-o/--overwrite`。判定看"安装基线"(组件目录 `.xui-manifest.json` 里记的上次写入哈希):基线对得上才叫"只是版本升级"(直接替换), 对不上叫"你改过"(默认保留), 该文件没有基线只说"无法判断来源"、不断言你改过。这是 copy-not-install 的前提:代码归你。
- **组件目录**默认 `ui`, 可用 `init -n -p/--path <目录>` 在新项目里就把名字定下来(`main.go` 的 import、README、AGENTS 与安装清单都跟着这个名字;只许项目内的相对目录, 绝对路径或 `..` 跳出会被拒 —— 清单落在组件目录里, 跳出去之后就再也找不到自己的基线), 之后用 `add/update/info -p/--path <目录>` 指过去(`info` 也认它, 否则组件装在 `components/` 时它会答"ui/: 未初始化" —— 那是错的);全局 `-c/--cwd` 先切换工作目录。**包名始终是 `ui`**, 与目录名无关, import 时给别名(`ui "myapp/components"`)。
- **带值的选项两种写法都不接受空值**(`-p ""` 与 `-p=` 一样当场报错): 空目录会把组件直接拷进当前目录根, 空的 `--view=` 会把"只看一个文件"静默变成"全部打印"。值被 shell 变量吃掉时该失败就失败, 不要退化成"少了一个过滤器"。
- **`list` / `search` 的搜索词可以写成位置参数**(`xui search table` 与 `-q table` 等价);给了两个不同的词会报错而不是猜, 同一个词重复给不算矛盾。
- **`-v` 是全局的版本号**(等价 `xui version`), 与上游 shadcn 的 `add -v, --view` 撞名 —— 取舍是把 `-v` 留给版本, 看源码用 `--view`;打 `xui add -v card` 会报错并指你去 `--view`, 不会让你以为这个开关不存在。
- **静默 `-s/--silent` 只压进度输出**:它压掉"创建了哪个文件"这类过程信息, **命令本身的结果照打**(`list` 的组件表、`view`/`--diff` 的源码与差异、`info` 的报告、`version`、`self-update --check` 的对比), 警告与错误也照打到 stderr。所以 `xui list -s` 给你一份没有表头的干净清单, 可以直接 `|` 给脚本; 升级提示走 stderr, 不会混进被 `>` 重定向的源码里。
- **只展示的参数不吞写文件的参数**:`--view` / `--diff` 与 `--dry-run`、`-o/--overwrite`、`--no-install`、`--strict` 同时给会直接报错(它们本来就不写文件, 静默忽略等于让你以为预览生效了)。**过滤值谁都没命中 = 命令失败(退出码 1)**, 不是"没有差异/没有内容": 前者是"你要的东西不存在", 后者是"存在且一样", 混成一句会让脚本以为拿到了正确的空结果(`-j` 里同时给 `matched=false`, 但退出码也照此)。
- **`--dry-run` 没有任何副作用**(init / add / update 都认): 不建目录、不写文件、不写安装清单、不联网动依赖、不调 `xc` 生成 `.syso`, 措辞一律用"将创建 / 将更新 / 将覆盖 / 将跳过"。以前 `init` 根本没有预览 —— 在别人仓库里生成项目只能赌一把。
- **`-j/--json` 给脚本与 AI 助手读**: `list` / `search` / `info` / `view`、`add` / `update` 的 `--view` 与 `--diff`、`self-update --check` 都支持。字段形状见 README 的表(空的 `files` / `deps` 一律 `[]` 不是 `null`; 过滤值没命中时靠 `matched=false` 区分"没命中"与"没差异")。别的命令不给: 写文件那条路径的输出是过程日志, 收下 `-j` 又不生效是骗人。
- **退出码**: `0` 跑通(有跳过时 stderr 汇总一句"共 N 个文件被跳过")、`1` 命令失败(含 `--view`/`--diff` 的过滤值没命中)、`2` 只在 `--strict` 且有文件被跳过时出现。默认 `0` 是故意的 —— 保守跳过本地改动是正确行为, 不该让人类手里的 `xui update` 报错; CI/agent 要"没更新干净就失败"就跑 `xui update --strict`(可以配 `--dry-run` 先检查一遍)。
- **`update` 不接 `--all`**:`xui update` 省略组件名就已经是"更新组件目录里已装的全部"; 把没装过的组件也一起拷进来是 `xui add --all` 的事, 两个命令不该混在一个开关里。点名一个**一个文件都不在本地**的组件同样只提示 `xui add <组件>`、不替你装(名字叫 update、行为是 add 的情况在显式点名时也一样不许发生); 已装组件缺的个别文件(跨版本升级新增的共享文件)则照常补齐 —— 那不算装新组件, 不补直接编译不过。
- **新建项目的两个开关分开**:`-f/--force` 只放开"目标目录已存在且非空"这道闸门;`-o/--overwrite` 才决定能不能丢弃本地改动, 而且与组件文件共用上面那套基线判定(`go.mod` / `main.go` / `README.md` / `AGENTS.md` / `.gitignore` 的基线记在同一份清单的 `project_files` 里)。以前一个 `-f` 两件事都干, 于是一次 `xui init -n app --force` 能把你改过的 `main.go` 无声覆掉。普通 `xui init [ui目录]` 没有"非空目录"这道闸门(`-f` 会直接报错), 它按同一套基线刷新: 没改过的文件随 CLI 升级, 改过的保留(要丢弃加 `-o`)。重跑 `init -n` 时若 `-p` 与上次不同(或忘了带, 默认 `ui`), 会**另起一套**组件目录与安装清单(骨架文件的基线在旧清单里, 新目录找不到) —— CLI 会提醒一句但不阻断, 之后每条命令的 `-p` 别混用。
- **依赖安装**:`xui init -n` 生成后默认在项目目录执行 `go mod tidy`(生成 `go.sum`), 并在缺 `xc` 时 `go install` 后生成 winres 的 `.syso`;离线或想手动控制时加 `--no-install`(`add` / `update` / `init` 都认): 组件与文档照写, 只是不动 `go.mod`; `init -n` 还跳过 tidy 与 xc(`winres/` 照写, 之后自己跑一次 `xc res make --arch amd64,386`)。
- **升级自己**:`xui self-update [版本]` 跑 `go install github.com/twgh/xui/cmd/xui@<目标>`(默认 `@latest`, 也可 `vX.Y.Z` / `main` / `HEAD` / commit 短哈希)。成没成以"跑新二进制拿到的版本"为准;正在运行的 exe 不许就地覆写, 失败时会把它改名成 `.old` 让路重试一次。`--check` 报"当前版本 / 最新 tag / CLI 最后改动的 tag"三行再给结论, 因此**不接受版本参数**(收下又丢掉会让人以为查的是那个版本)。
- **升级提示**:依据是 `registry.json` 的 `cliChange` —— **CLI 最后一次改动在哪个 tag**(命令行为与内嵌模板/资源;组件批次与设计文档更新不抬它, 机械抬配套版本也不算)。装的版本低于它, 命令正常跑完后才向 stderr 提示一句;数据源没这个字段(旧镜像)才退回比最新 tag, 并说明这是退路。`-s/--silent` 与 `XUI_NO_UPDATE_CHECK=1` 关闭;本地/`go run` 构建不提示;`version` 与 `info` 不联网拉数据源, 所以也拿不到这个依据。
- **配套运行时版本**:组件源码跟 `main`(永远最新), 运行时 `github.com/twgh/xui` 只能按 tag 取, 两者的差值就是编译错误(`undefined: xui.Xxx`,因为组件用了新加的运行时助手而项目钉着旧 tag)。`registry.json` 的 `runtime` 声明与这批组件**配套**的运行时版本 —— 每次发布都抬成刚发的 tag(只改 CLI 也抬), 所以 `xui update` 会顺带把项目 `go.mod` 的该依赖升到最新发布版(等价 `go get github.com/twgh/xui@<tag>` **再 `go mod tidy`** —— 只 `go get` 的话 `go.sum` 里会留着上个版本的条目, 看着像没升干净), 运行时的修复也一并带上;`--dry-run` 与 `--no-install` 只打印命令, `go.mod` 里 `replace` 了该模块时完全不动你的依赖; 向上找到的那份 `go.mod` 若**既在当前目录之外、又没要求 `github.com/twgh/xui`**, 也不动它(在临时目录或别人的 monorepo 上层跑 CLI 时, 不该去改一个不相干模块的依赖)。
- **数据源**:默认从仓库 `main` 分支拉组件(永远最新),自动在直连 / Gitee 镜像 / GitCode 镜像 / 加速代理之间选路;拷到一半某个文件拉不到(镜像限速、临时挂起)会自动换下一个源继续,不会留下半套组件源码。`XUI_REGISTRY` 可指向本地目录或任意 `http(s)://` 基址(离线开发 / 锁定组件版本),显式设置时只走你写的地址、失败直接报错(不静默降级)。
- **下发的文档与组件同源**:`DESIGN.md` / `DESIGN-PITFALLS.md` 也从数据源的 `docs/` 取,不是从 CLI 自带的版本读 —— 组件批次同时会改这两份说明(第八节的编号就是组件行为的依据), 只从 CLI 本地读会把"最新组件 + 上个 tag 的说明"装进同一个项目。拉不通(源里没有 `docs/`、代理限流回了个页面)才退回 CLI 自带的那份, 并在 stderr 说明用的是哪一份。`AGENTS.md` 由 CLI 内嵌模板渲染(它要写 `-p <目录>` 这类路径), 那部分才跟 CLI 版本走。
- **registry.json** 是组件清单:每个组件登记 `files`(要拷的文件,含跨文件助手)与 `deps`(依赖的其它组件)。CLI 只拷这些 `.go` 文件;运行时(`github.com/twgh/xui`)与图标子包(`github.com/twgh/xui/icons`)是 Go 模块依赖,不进组件目录。

---

## 六、组件目录与组件依赖

`xui init` / `xui init -n` 之后你的项目里是(目录名默认为 `ui`, 可用 `xui init <目录>` 或
`xui add -p <目录>` 指定别的名字;本文里不带路径前缀的 `xxx.go` 都指这个目录下的文件):

```
<组件目录>/
  theme.go              主题令牌 —— 你的文件, 随意改
  DESIGN.md             本文
  DESIGN-PITFALLS.md    第八节的完整正文(与本文一起下发, 编号一一对应)
  AGENTS.md             给 AI 助手的目录说明(普通 init 才放; init -n 放在项目根)
  button.go card.go …   组件源码(init 只带基础组件, 其余按需 add)
```

- `AGENTS.md` 随 `init` 下发: `init -n` 放在项目根, `init [目录]` 放在组件目录; 它给 AI 助手
  说明运行方式、组件用法与修改规则。和组件源码一样走安装基线: 重复 init 时没改过就随 CLI 刷新,
  改过则保留(要丢弃加 `-o`); `-f` 只管"目标目录已存在且非空"这道闸门, 不决定覆不覆盖。
- 所有组件是**扁平的单包** `package ui`(一个组件一个文件),互相直接调用,没有子包。
- 几个**跨文件助手**会随依赖一起拷进来,不要删:

  | 文件 | 作用 | 被谁依赖 |
  |---|---|---|
  | `scrollbar.go` | 细滚动条皮肤 + 悬停显隐 | table / tree / combobox / input(多行) / scrollarea / dialog(可滚动内容) |
  | `popup.go` | 弹层骨架(定位、阴影、进出场、打开态重落位/光标判定) | datepicker / timepicker / select / popover / hovercard / menubar / navigationmenu |
  | `backdrop.go` | 弹层背景模糊(抓窗快照) | dialog |
  | `sheet.go` | 侧边面板(复用 Dialog 的模态基座) | dialog 的文件清单(模态分支) / drawer |
  | `menu_model.go` | 菜单条目模型与 `MenuOpt` 构造器 | menu / menubar |

- 组件依赖(registry 里的 `deps`):`card → label`、`dialog → button + label`、`alertdialog → dialog`、`combobox → tooltip`、`menu → tooltip`、`empty/field → label`、`item → label + separator`、`sheet → dialog`、`drawer → sheet`、`menubar → button`、`navigationmenu → button`、`carousel → button`、`data-table → table + input + button + label + select`、`input-group → menu`(复合 addon 的 MenuButton)、`status-bar → tooltip`(条目/抓手的悬停气泡);`add` 会自动补齐,不用手抄。`dialog` 的 `files` 还包含 `scrollarea.go` / `scrollbar.go`(可滚动内容用),`alertdialog` 共用 `dialog.go` 的面板/遮罩/模态栈。
- 图标不在组件目录里:用 `github.com/twgh/xui/icons` 的常量(只是一条 SVG path 字符串,只把用到的打进二进制,体积增量 0)。
- 升级:`xui update`(已装组件 + 已下发的文档)或 `xui update button`(只更新该组件及其依赖)。

---

## 七、组件覆盖与已知限制

组件与 shadcn 的对应关系见第四节。以下差异是 **xcgui 能力边界**导致的,改组件代码也补不上:

| 差异 | 原因 |
|---|---|
| Input 的文字/光标/选区/IME 由原生 Edit 绘制 | 都在 DLL 内;只开放字体/字色/占位符色/插入符色/文字内边距。要完全控制就得自绘文字并接管光标闪烁、选区命中、IME 组合窗 |
| 英文长文本折行按字符断行(可能在单词中间换行) | Label 载体是原生 Button, 折行算法在 DLL 内, 不按单词边界; 中文不受影响(第八节 90/91) |
| Dialog 没有 zoom-in-95 / AlertDialog 没有 text-balance | 元素级缩放对自绘元素无效(第八节 47); 自绘文字没有平衡排版能力 |
| 没有 `asChild` / `data-slot` / `data-state` | React/DOM 概念, Go 侧没有对应物 |
| Card 的 `overflow-hidden` / 首末图片自动圆角、AspectRatio 的圆角裁剪 | xcgui **会按父元素矩形裁剪子元素**(实测, 见第八节 97), 但没有圆角裁剪/overflow-hidden 语义: 超出圆角的四角区域仍会画出来; 需要圆角裁剪的图片/内容请自己按路径裁 |
| `--chart-*` 色板只有 5 色, 且中性主题下是灰阶 | 上游同样是 5 个槽位循环; Chart 组件按分类序号取色, 超过 5 类会重复, 要区分请自己给足色板。灰阶下饼图百分比标签按片色亮度在正文色/背景色之间取对比更高的那个(第八节 159) |
| Chart 没有 HTML 的响应式容器/Tooltip 组件联动 | 尺寸完全由布局项决定(建议 `ItemFill` 或给定宽高); 悬停提示卡是元素内自绘的小卡片, 不是 `ui.Tooltip` 弹窗(不抢焦点、不新建小窗) |
| 横向柱用一个选项而不是一个类型 | 上游 recharts 是 `<BarChart layout="vertical">` 的形态开关, 这里同样是 `ChartBar + WithChartHorizontal()`: 几何整套旋转 90°(见第八节 161), 不新增 `ChartKind` |
| 径向图与雷达图的刻度语义 | 径向图每层弧长 = 该分类 / 该类最大值(不是占总量, 那与环形图重复); 雷达图半径 = 该点 / 各轴峰值, **不套数值轴的整齐刻度** —— 套了 305 会被抬到 400, 多边形整体缩一圈, 看着像数据没到顶 |
| 雷达图维度 < 3 时不画 | 两点连不成面, 直接留白(调用方应保证 `len(labels) >= 3`) |
| Calendar 不做 RTL(dir=rtl)/波斯·Jalali 历/timeZone/自然语言日期(chrono-node) | 文字渲染只有 LTR; 历法换算与时区数据是另一套领域库, Go 侧没有 chrono-node 的等价物 —— 绝对/相对日期的键入用 DatePicker 的可输入形态(`ParseDateText`)覆盖最常用场景; 范围色带跨月时在两个月块之间不连续(上游同样是每月一张独立表格) |
| Field 横向 `Content` 的嵌套列只认 Fixed 宽 | xcgui 不会把 Weight/Percent 宽度传给嵌套布局的 Fill 子项(实测子项宽恒为 0, 内容不显示); 组件默认 Fixed 240, 用 `xui.ItemWidth` 覆盖(第八节 101) |
| EmptyDescription / FieldDescription / FieldError 默认单行 | 标签本身默认单行截断; 需要多行时对返回的 Label 调用 `SetAutoWrap()` 自动折行并长高(也可用 `SetWrap` + `WrappedHeight` 手动量高) |
| Item 的 Header/Footer 没有自动 space-between, 动作之间用 Item 的行间距 | Go 布局没有 justify-between: 用 `xui.ItemWeight` 给靠左的第一个子项撑开; 行内动作也不再单独套 gap-2 容器(嵌套 Auto 宽不量测原生按钮, 见第八节 104) |
| Breadcrumb 链接/当前页是单行自绘文本, 且不参与 Tab 焦点 | Label 的原生 Button 载体按 `TextWidth` 精确设宽会被内部文本内边距截断(第八节 105); 自绘后不折行、不支持富文本, 键盘焦点需要每链接一个可聚焦元素(当前为鼠标驱动) |
| 嵌套在布局容器里的控件(卡片里的按钮 / 分页 / 标签栏 / 分隔条等)无法用 Tab 聚焦 | xcgui 的 Tab 焦点链只覆盖窗口内容区的**直接子元素**: 直接挂在内容区的可聚焦 Element 可以 Tab 到, 套一层 LayoutEle/Element 容器后就被跳过(Button 也一样, `EnableFocus/EnableSwitchFocus/EnableKeyTab` 都补不上, 见第八节 157); 组件内的键盘操作需先用鼠标点击获得焦点 |
| Toggle / ToggleGroup 的 outline 不画 shadow-xs | 上游有 1px 浅投影; 本组件库自绘壳体统一不画阴影(Button/Input 同), 在 Card 底上几乎不可见; `border-input` 与悬停/选中底色完整保留 |
| InputOTP 没有"跨格拖动选择"/Ctrl+A 全选 | 每格是独立的原生 Edit, 选择语义无法跨格; 复制/剪切整段验证码也未做(粘贴方向完整) |
| Select 弹层没有 radix 的 scroll-up/down 悬停自动滚按钮与 typeahead(键入跳转) | 自建弹层用 ScrollArea 承载, 滚轮与 ↑↓/Home/End 都可用; typeahead 需要按键聚合计时, 后续有需求再加 |
| Popover / HoverCard 进场只淡入, 没有 zoom/slide | 自建阴影窗能稳定动画的只有元素 alpha; 整窗位移/缩放会闪或崩(第八节 47/64), 离屏测量与淡入都在第八节 115/116 |
| HoverCard 不支持键盘聚焦打开 | 非激活窗口不参与 Tab 焦点链; 需要键盘可访问时用 Popover 包一个可聚焦触发区 |
| HoverCard 内容可鼠标交互, 但非激活窗口不接收键盘焦点 | WS_EX_NOACTIVATE 不抢前台, 这是"悬停预览不打断输入"的前提; 需要键盘操作的内容用 Popover |
| ScreenPopup 默认拿不到键盘焦点 | 与 HoverCard / ScreenToast 同为非激活窗口: 鼠标悬停与点击正常, 但输入框/文本域无法输入, ESC 也不会被面板收到。需要输入时传 `WithScreenPopupFocusable(true)`: 显示仍不抢前台, 用户点击弹窗后才激活, 输入/ESC 可用, 且失焦不自动关; 或改用 Popover / Dialog。它是单个常驻面板, 默认不自动关闭(可设时长, 计时不因悬停暂停)、不堆叠 —— 需要通知流用 ScreenToaster |
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
| DataTable 没有行拖拽 | 原生 List 没有行拖拽; 排序 / 筛选 / 分页之后"看到的位置"与原始行下标不是一回事, 拖拽落点要由业务定义(是否改原始顺序、跨页怎么放、筛选下隐藏行怎么算), 组件不替调用方决定 |
| DataTable 的表头拖拽只能换显示顺序, 不改 `DataColumn` 下标 | `Render` / `FilterOptions` / 排序键都用稳定的列名下标记; 拖拽只重排 `order`(隐藏列的位置也保留), 避免换位后 Facet / 自定义渲染回调错位 |
| DataTable 的中文按 UTF-8 码位排序 | Go 侧没有中文拼音排序; 数字/英文自然排序已对齐, 业务口径用 `DataColumn.Compare` 覆盖 |
| DataTable 的表头排序只能鼠标点击 | 表头是原生 List 的子元素, 不参与 Tab 焦点链; 页脚按钮/下拉嵌在布局容器里, 同样到不了 Tab(见上一条); 表头没有可聚焦元素, 也无法补按键处理 |

其余与上游的细小差异(逐项语义、变体映射)已在第四节的两张表里列出。

---

## 八、xcgui 踩坑全集(索引;完整条目见同目录 DESIGN-PITFALLS.md 第八节)

> 编号稳定, 只增不改; 源码注释里的"见第八节 N"按编号查。
> 每条的现象 / 根因 / 修法 / 实测都在 **DESIGN-PITFALLS.md** 的第八节, 下面只列编号与标题。

- **1** 渲染模式二选一
- **2** 2000 回调上限
- **3** 背景管理器累积上限 20
- **4** 主题切换必须 Redraw
- **5** 模态窗口销毁
- **6** DPI(实测 DPI=144 修正)
- **7** Edit 自绘
- **8** 字体
- **9** 拷贝模式的代价
- **10** 元素绘制裁剪
- **11** 实心按钮 hover 用透明度
- **12** cy=0 的控件高度就是 0
- **13** 窗口背景必须用 Leave 状态注册
- **14** 标题文字默认白色
- **15** 窗口样式按标志位组合
- **16** focus-visible 需自行跟踪焦点来源
- **17** 元素 paint 回调里能画位图、也能画 SVG;图标走专用管线
- **18** `XWnd_SetWindowPos` 吃物理像素
- **19** 原生 ComboBox 必须先 `CreateAdapter`
- **20** SliderBar 的滑块几何
- **21** 原生滚动条要手动收敛
- **22** SliderBar 的滑块按钮在 hover 时会冒方块
- **23** 原生绘制的文字不要自己再画一遍
- **24** 异型透明窗口可以做到真透明,且背景对象支持圆角
- **25** 别跟原生 SliderBar 的滑块按钮(原生 Button)纠缠, 要么全套压制要么完全弃用
- **26** Slider 几何要对齐 radix
- **27** `XDraw_FillRoundRect`/`XDraw_DrawRoundRect` 的 nWidth/nHeight 就是圆角半径
- **28** 不整体填充的透明元素必须在 onPaint 开头先铺宿主底色, 否则上一帧像素残留
- **29** LayoutEle 布局语义与文档直觉不符, 全部经实测
- **30** ComboBox 原生弹层的样式接管路线
- **31** xcgui 字体的 size 单位是 pt(磅), 不是像素
- **32** 本文的 API/行为结论对应 xcgui 1.4.3
- **33** Dialog 用"窗口内覆盖元素"实现
- **34** Dialog/文字/量测的一批实测结论
- **35** 暗色下的"半透明令牌 + /NN 修饰"必须做 alpha 乘法
- **36** 原生菜单皮肤的一批实测结论
- **64** 菜单弹窗转阴影窗 + 进场动画的一批实测结论
- **37** 页面容器 Show(false)→Show(true) 后内容空白
- **38** 自绘标题栏的无框窗口组合与鼠标穿透
- **39** 窗口布局(EnableLayout) + 阴影窗口 + 自绘标题栏的一批实测结论
- **40** 原生 List 皮肤接管路线
- **41** LayoutEle 纵向(列)布局也会"溢出换列"
- **42** 原生 Tree 自绘皮肤(Tree 组件实测, 载体路线同 40)
- **43** 图标抗锯齿: 直通 D2D 几何填充(`d2dicon.go` + `icons.go`)
- **44** lucide 图标资源预处理(生成 `icons/` 子包的常量表)
- **45** hover 过渡动画(运行时 `anim.go`)
- **46** 重绘节流 `XC_SetPaintFrequency`(实测; 顺带记一条空操作 API)
- **47** 弹层 / 对话框进出场动画(运行时 `presence.go`)
- **48** `XC_GetTextShowRect` 的 width=0 是"按一个字符宽折行", 不是"不限宽"
- **49** Tooltip 被"迟到的 MouseLeave"秒关: 密集元素上会表现成"提示根本不显示"
- **50** Tooltip 是深色气泡, 压在深色背景上肉眼容易漏看
- **51** 隐藏容器里建的 Tree, 首次显示"只有行底和 chevron, 文字全空"
- **52** ComboBox 弹层高度: 内容高 = n×(项高+1) + 2×pad + 5, 取该值才无滚动且留白对称
- **53** Pagination: 折叠必须"恒定槽位", 且组件在 OnChange 前不要用立即重绘
- **54** 元素销毁事件会为子元素逐个触发(含嵌套), 顺序在窗口 `XE_DESTROY` 之后
- **55** 布局尺寸语义实测全集: `Fill` ≠ "剩余空间", `Weight` 只在父级尺寸"确定"时生效
- **56** Card 是布局容器: 内容区随卡片尺寸自动联动
- **57** 异型透明窗(Shaped)的"整窗透明度"只认第一次调用, 之后永远冻结
- **58** 独立置顶小窗不跟随主题切换重绘, 也不在"已注册窗口"清单里
- **59** "Auto 高的元素"挂在错的父级里时, 它内部 Fill 子项的宽度会按
- **60** "自适应列宽"依赖元素宽度, 而宽度是布局算出来的 —— 必须订阅元素尺寸事件自愈
- **61** 自绘组件的"宿主底色 + 组级半透明"合成规则
- **62** hover 过渡的"最后一帧"不再触发重绘 —— 二值可见物(下划线)淡出会永久残留
- **63** 开着对话框最小化再还原 = 整窗元素绘制/命中全灭, 只能"最小化前挂起、还原后重显"规避
- **65** COM 直通的参数传递(386 / amd64) + "浮点形参"的完整解法
- **66** 页面滚动容器 = 原生 `LayoutFrame`(布局框架)的实测语义
- **67** 自绘标题栏的"最大化/还原"按钮:用 `Button_Type_Max`, 别自己调 `MaxWindow`
- **68** 原生 ComboBox 默认带中文字间距, 弹层 ShapeText 没有 → 打开下拉时文字会"变紧"
- **69** ComboBox 弹层: 原生弹层能转异型透明窗, 但整窗 alpha 淡入不可靠 —— 已取消进场淡入
- **70** Presence 的键必须全局唯一: 撞键 = 新弹层"全透明但仍在", 表现为点击没反应
- **71** 异型/阴影弹窗里, 元素的布局坐标原点是窗口矩形原点(含阴影边距), 不是可见区(卡片)原点
- **72** 手动创建原生 ListBox 做自绘列表(时间框弹层)的实测语义
- **73** 复用的 Shadow 弹窗做淡入: 整窗 alpha 冻结 / 窗口白底 / apply 互搏 / 过渡挂错窗口 / 首帧残留 / DLL 无效句柄 —— 五个坑叠一起才是"闪烁、抖动、闪退"
- **74** 弹层夹边界必须用"显示器工作区", 不是屏幕高度
- **75** 通知(Toast)用窗口内单元素自绘, 悬停暂停走窗口级 GetStayEle, 位置动画必须整窗重绘
- **76** 自绘标题栏窗口的封装: `EnableLayout` 必须调, 双击判定别看 `GetStayEle != 0`, 最大化不用手动去阴影
- **77** 只隐藏不销毁的辅助小窗会让进程挂在 `Run()` 里
- **78** 输入框的内嵌按钮: 动作要放在 `LButtonUp`, 不能放 `LButtonDown`
- **79** 密码框遮挡符默认是 `*`, 要显式设成实心圆点
- **80** 原生 `Edit` 的内边距必须自己给(多行尤其)
- **81** 内嵌按钮在任何状态都要吞掉点击, 否则连点会选中文字
- **82** 原生 `Edit` 不继承窗口字体, 必须显式 `SetFont`
- **83** paint 回调里换过的字体必须还原, 否则会带偏原生文字
- **84** `EnablePassword` 切换不会重排字符步进, 切换后必须手动触发一次重排
- **85** 焦点环/校验环画在元素内部, 内嵌控件要按环宽内缩
- **86** 按钮组(ButtonGroup)的布局/绘制语义
- **87** 持续旋转动画(spinner)与图标旋转
- **88** 高 DPI 下"文字/图标/壳体按下位移不一致"
- **89** 元素半透明时, 同色图元重叠会二次叠色
- **90** 折行文本的高度与 Auto 布局重排
- **91** AlertDialog / 多层模态 / 折行量测的一批结论
- **92** 过渡"到位的那一帧"必须重绘, 否则屏幕停在倒数第二帧
- **93** "宽度被布局拉伸的 Auto 高组件"要自己重算高度, 且改动要逐级通知祖先
- **94** AspectRatio(比例容器): xcgui 没有 aspect-ratio, 只能"宽度驱动高度 + 批量重算"
- **95** CardFooter 的内边距用"内层内容区的元素 padding + 上下外边距"表达, 子项别直接挂在页脚上
- **96** 窗口级 `AdjustLayout` 会重排窗口直属的绝对定位覆盖层(Toast), 不能拿它当"全局重绘"
- **97** 父元素按矩形裁剪子元素; Collapsible 的高度动画可以直接靠它遮罩
- **98** 滚动条的 alpha 是"悬停显隐"状态, 不能跟着弹窗整组淡入一起设置
- **99** 右键触发区与菜单自适应宽度的一批实测结论
- **100** `XDraw_SetFont` 会重置文本对齐状态
- **101** 横向布局里的嵌套布局只认 Fixed 宽(Weight/Percent 不会传给 Fill 子项)
- **102** 元素重排后必须整窗重绘, 否则旧位置文字留下重影
- **103** `XEle_SetPadding` 可用于布局元素; 用带外边距的子容器当 padding 会让 Fill 行超宽
- **104** 横向 Item 的 Content 不能靠 Weight 吃剩余宽度, 要算成 Fixed
- **105** Label 按 `TextWidth` 精确设宽会被省略号截断
- **106** 连续动画: 共享宿主窗口定时器 + 局部矩形重绘 + 只在看不见时停表
- **107** 不透明的圆角控件不要先铺方底擦色
- **108** 原生 Check 按钮不处理键盘激活
- **109** Label 载体的默认上下 border 会裁掉 g/p 的下伸部
- **110** `XEdit_SetTextAlign` 是生效的; 用它给文字"顶居中"的反而是 border 技巧
- **111** 同一窗口里多个高度过渡, 每帧会画出"半新半旧"的中间态
- **112** 真彩/暗色里"整块被填成灰色"的焦点环, 十有八九是漏了"内芯重新铺底"
- **113** 分段拼接的边框(多个格子共享边)必须按"组"算圆角, 且擦边只擦中段
- **114** `Enable(false)` 挡不住鼠标聚焦原生 Edit
- **115** 隐藏窗口里量不出布局的 Auto 高, 要"离屏显示一次"再量
- **116** 不抢焦点的悬停弹层不能走 KillFocus 关闭
- **117** 圆形头像图片: RGBA 蒙版 + `XDraw_ImageEx` 在 D2D 下可用
- **118** 长页面滚到底后, 深层嵌套卡片的命中区与绘制位置可能不同步
- **119** `XWnd_SetWindowPos` 的位置是物理像素, 尺寸是逻辑像素
- **120** 窗口内的侧边面板滑入 = 平移面板及其直接子元素
- **121** 拖拽关闭不能清掉主 presence 过渡
- **122** Menubar 不能用原生菜单做弹层
- **123** 子菜单必须是激活弹层, 父层要在焦点转移期间"免疫失焦"
- **124** 普通 Element 在布局里的 Auto 宽会解析成 0
- **125** `XWnd_GetRect` 返回逻辑像素, `wapi.GetWindowRect` 才是物理像素
- **126** 原生 Edit 的滚动条子元素是"延迟创建"的, 整树 Show(true) 会在第二次显示时把它们点亮
- **127** 相邻弹层之间不能留缝, 光标归属要按"窗口矩形"而不是卡片矩形判定
- **128** 窗口直属的绝对定位浮层要用"布局项置位置"钉住, 才扛得住一切窗口级布局趟
- **129** 容器的内边距不能用子项外边距表达: Fill 子项会按父级全宽解析并溢出
- **130** 横向行里 Weight 宽的嵌套列, 其 Fill 子项要由容器按实测宽锁 Fixed(并在尺寸变化时重新锁定)
- **131** 宿主置顶(SetTop)时, 自建弹层窗口必须用 HWND_TOPMOST; 原生菜单/下拉不用改
- **132** 原生 `XMenu_Popup` 在副屏(显示器原点非 0,0)上把"显示器矩形 × DPI"当边界钳制, 弹层严重错位; 落位要在 Popup_Wnd 回调里用 `SetRect` 自己算
- **134** 原生子菜单的 Y 会把它的底边夹进"父弹窗矩形底边"的缓存值 —— 父弹窗被我们挪过就整体错位一整行
- **135** ComboBox 自绘下拉的"屏幕边界"不能写死主屏
- **133** "上方放不下就翻到下方"的判断不能拿虚拟屏原点 (0,0) 当边界
- **136** `Presence` 幂等: 值已经等于目标值时不起过渡、也不会回调 done —— 靠 done 落终态的组件会"布局变了但屏幕没重绘"
- **137** 非激活弹层里的普通元素没有原生 hover 状态、也没有 MouseMove —— 链接悬停与"指针进入内容"都要自己轮询
- **138** 在 KeyDown 回调里同步 `XWnd_SetFocusEle` 会把同一次按键重新派发给新焦点, ←→ 一次移动两项
- **139** 面板 `resize` 之后还有一次布局趟, 会把手动写的滑入位移原地复位
- **140** 元素鼠标事件不冒泡; 拖拽要"窗口级起手 + 捕获元素跟手"
- **141** 循环轮播的折回要用"正模"
- **142** Button 的元素高 = 视觉高 + `pressShift`(按压位移预留), 容器要按"视觉高"居中, 不要按元素高
- **143** Radio 的原生互斥不要逐项手动取消, 键盘搬焦点必须推迟到消息循环
- **144** Slider 的"值数组 / 垂直方向 / 滑块热区"三条几何结论
- **145** Progress 的 max / 不确定态 / 带标签形态
- **146** Switch 的空格键、禁用与 aria-invalid
- **147** Input Group: 内层输入框必须是"裸输入", 焦点环由容器画
- **148** 布局容器放进自建弹层时: 位置参数会被布局覆盖, 内边距只能用元素 padding + 子项锁 Fixed 宽
- **149** xcgui 会把 9pt 及以下的常规字重画成宋体, 从字体文件创建可绕开
- **150** 按钮类文字的 Vcenter 不需要上移 1px —— 早期回补会让小字偏高
- **151** 中文文本的 Vcenter 要上移半个逻辑像素(只调中文)
- **152** 可聚焦元素(EnableSwitchFocus)的 ←/→ 会被框架当成"切换焦点", 键盘处理必须挂在窗口级
- **153** 分页的三个显示形态与"焦点环画在条目上"
- **154** 转圈图标切出加载态时要显式 `xui.Spin(hEle, false)`
- **155** 标签栏整块是一个 Tab 站点(roving focus), 单标签不能再各自可聚焦
- **156** 提示气泡的四方向定位与"每实例一个过渡键"
- **157** 可调分栏: 百分比换算像素要按累计边界取整, 拖动以分组为坐标原点
- **158** 侧边栏: 原生 Button 会吞掉子元素的点击, 动作钮只能做内部命中区
- **159** 图表: 曲线要自己采样, 悬停要走窗口级命中, 片内文字要按对比挑色
- **160** 平滑的扇形: `XDraw_FillPolygonF` 在 D2D 下没有抗锯齿, 而直通 D2D 拿到的渲染目标没有元素变换
- **161** 横向柱 / 径向 / 雷达: 直角坐标几何可以整体旋转, 但极坐标的命中与量程各有一套语义
- **162** 数据表格: 表头排序图标要挂在 Table 的表头绘制回调里, 点击用 Header_Click
- **163** 输入框的清除按钮: 程序化 SetText 不发变更事件, 内嵌件命中区又是 paint 时建的
- **164** DataTable 的高度: Auto 父级里的 Weight 子项不会撑高, 连 `ItemMinSize` 也不行
- **165** Input Group 复合 addon: 外扩量要加在 addon 上, 按钮高度按尺寸档算
- **166** 文本域自动长高: 行数取原生 Edit 的显示行数, 重排只交给 ScrollArea 框架
- **167** 单元格扩展绘制要走 Table 的内部钩子, 列索引在 `pItem.ISubItem`
- **168** 自绘单元格必须把原生文本清空
- **169** 删列重建列结构: `DeleteColumnAll` 之后必须重建行适配器
- **170** 勾选列与原生单选解耦: 在 `LButtonDown` 里命中测试并吞掉消息
- **171** 分面筛选比的是单元格原始文本, 不是画出来的内容
- **172** 布局项"收起"要同时把宽度和边距归零
- **173** 原生菜单是模态的: 打开期间的重绘会被它关闭时的"还原屏幕"盖回去
- **174** 列头菜单的锚点用绘制缓存里的表头矩形, 不要 `PopupAtCursor`
- **175** 分面筛选面板用 Popover + Command, 触发区是自绘胶囊
- **176** 表头悬停底色与列拖拽: 原生表头只改项状态、不重绘, 必须订阅表头元素的鼠标事件
- **177** 屏幕通知用"每条一个异型置顶小窗", 动画要挂宿主窗口驱动
- **178** 托盘图标: xcgui 托盘 API 是进程级全局状态, 还要自己处理 Explorer 重启
- **179** 承载组件的窗口都会被登记主题重绘: 自建小窗销毁时必须反注册
- **180** 窗口图标要单独设: 任务栏 / Alt+Tab / 任务栏预览窗是三个位置
- **181** 非激活窗口只设 WS_EX_NOACTIVATE 不够: 点击仍会被 xcgui 顶到前台
- **182** 屏幕弹窗复用弹层底座: 尺寸、定位与生命周期要自己算
- **183** 过渡表按"每次交互一个新键"登记会只增不减: 内存与每帧遍历成本随交互次数线性增长
- **184** 标题栏按钮上双击会把窗口顶成最大化; 弹层里再开弹层会让弹层自己关掉
- **185** 布局元素是"纯排版壳": 默认鼠标穿透, 要当命中区必须显式关掉
- **186** 小图形折线用 `DrawLine`, 坐标会被截断到逻辑整数, 图形整体偏左/偏上
- **187** 自绘 chips 容器随内容长高: 输入事件里同步改高度 + 逐级 `AdjustLayout` 会让兄弟项停在旧位置
- **188** 被 `SetCapture` 的元素收不到 `MouseLeave`: 拖动结束后提示气泡不消失
- **189** `XEle_GetWndClientRect` 给的是未裁剪矩形: 滚出视口的控件仍在视口外"命中"鼠标
- **190** 滚轮只发 `WM_MOUSEWHEEL`: 提示气泡不会随页面滚动收起
- **191** 阴影窗口实时拖边框改大小会闪 1~2 帧整窗透明: 在 Size 事件里同步重绘
- **192** List 滚到非整行位置后双击编辑, 输入框会整体上移: 行顶要用真实滚动偏移算, 不能对点击坐标取模

## 九、本设计依赖的已验证 API 清单(xcgui 1.4.3)

- `Element.AddEvent_Paint / AddEvent_MouseStay / AddEvent_MouseLeave / AddEvent_KeyDown`(widget/element.go:1192, 1288, 1336, 1744);键盘激活见第八节 108
- 焦点: `Element.EnableFocus / EnableSwitchFocus`(后者 = 接受方向键与 Tab 切换焦点)、`xc.XEle_EnableDrawFocus(hEle, false)`(关原生焦点框)、`xc.XWnd_GetFocusEle / XWnd_SetFocusEle`、`xui.StateOf(hEle).Focused` + `xui.FocusedByKeyboard()`(focus-visible 判定)、`window.NewByHandle(hWnd).AddEvent_KeyDown`(窗口级拦截, 语义见第八节 152)
- `Element.AddEvent_Menu_Draw_Background / AddEvent_Menu_DrawItem`(widget/element.go:2056, 2080)
- Menu / ContextMenu(见第八节 36/64/99):`Menu.AddItem / SetItemWidth / SetItemHeight / SetLeftWidth / SetBorderSize / EnableDrawBackground / EnableDrawItem / AddEvent_Menu_Select / AddEvent_Menu_Exit / AddEvent_Menu_PopupWnd`;弹出方式 `PopupAtElement`(元素下方) / `PopupAtElementRect`(元素内子矩形下方, 表头/单元格用, 见 174) / `PopupAtCursor`(光标处);
- Command(见 175):`NewCommand(x,y,cx,cy,parent, WithCommandPlaceholder/EmptyText/OnSelect/OnChange/OnEscape)` + `AddGroup/AddSeparator/Focus/Query/SetQuery/Clear`;条目 `AddItem(value,text, WithCommandItemIcon/Shortcut/Keywords/Disabled/OnSelect/Checked/Count)` + `SetChecked/SetCount`(`Checked`/`Count` 是给分面筛选这类"多选 + 计数"面板加的, 点条目不会自动关闭面板, 关不关由调用方决定);右键触发用 `Window.AddEvent_RButtonUp`(pPt = 逻辑像素, 原点 = 窗口矩形原点)+ `xc.XEle_GetWndClientRect` + `XC_IsHXCGUI / XEle_IsEnable / XWidget_IsShow`;`xui.TextWidth` 按自绘字体自行给菜单条目定宽(原生字体与自绘不一致)
- `drawx.NewByHandle / EnableSmoothingMode / SetBrushColor / FillRoundRect(Ex) / DrawRoundRect(Ex) / GradientFill2/4 / DrawText / TextOut* / Image* / DrawSvg* / SetClipRect / SetLineWidth`
- 持续旋转 / 脉冲 / 高 DPI 位移(见第八节 87/88/177):`xui.Spin(hOwner, active) float64`(相位 0..1, 1s/圈)、`xui.Pulse(hOwner, active) float64`(相位 0..1, 2s/来回, 与 Spin 共用动画定时器、键空间独立;`SetAnimationScale(0)` 冻结)、`xui.SpinDetached(hEle, hOwner, active)`(元素在独立小窗里时, 定时器挂 hOwner、每帧补重绘元素所在小窗)、`Painter.IconRotated(data, rc, color, strokeWidth, angleRadians, scale...)`、`Painter.IconRotatedOffset(..., dx, dy int32)`(按画布物理像素微调)、`Painter.TextF(str, *xc.RECTF, color, align)`(浮点矩形文字, 用于亚逻辑像素位移;整数矩形下与 `Painter.Text` 一致)
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
- 文件对话框(FileDialog): `wutil.OpenFileEx / SaveFileEx`(`OpenFileOption{Title, DefDir, DefFileName, DefExt, Filters, MaxOpenFiles, Flags}`)、`wapi.OFN_ALLOWMULTISELECT / OFN_EXPLORER / OFN_PATHMUTEXIST / OFN_OVERWRITEPROMPT`、`wapi.BrowseInfoW` + `SHBrowseForFolderW` + `SHGetPathFromIDListW`(`BIF_USENEWUI`)、`xc.XWnd_GetHWND`
- D2D 直通:`xc.XDraw_GetD2dRenderTarget / XDraw_ConvRect / XC_GetD2dFactory`(裸指针;直通绘制必须先用 `ConvRect` 换到画布对齐坐标,见第八节 43)+ **`github.com/twgh/xcgui/wapi/d2d`**(按 edge 的 COM 接口虚表风格封装,ABI 三条硬规则写在包注释里):借用入口 `NewRenderTargetByHandle`/`NewFactoryByHandle`;几何 `NewGeometry`/`NewGeometryFigures`(逐条可指定开/闭)/`Create{Rectangle,RoundedRectangle,Ellipse}Geometry` + `ID2D1Geometry.Usable/GetBounds` + `ID2D1PathGeometry.Open/Base` + `ID2D1GeometrySink.{SetFillMode,BeginFigure,AddLines,EndFigure,Close}`;绘制 `FillGeometry(Color)`/`DrawGeometry(Color)`(任意几何) + 图元 `DrawLine`/`DrawRectangle`/`FillRectangle`/`DrawEllipse`/`FillEllipse`(各带 `*Color` 便利版) + `CreateStrokeStyle`/`RoundStrokeStyleProperties`;其余 `SetTransform`/`GetTransform`/`GeometryCache[K]`(按工厂作废的几何缓存)
- backdrop-blur(见第八节 47):`xc.XImage_LoadFromGoImage`、`xc.XBkM_AddImage(hBkM, state, hImage, id)`、`xc.XBkM_GetObject`、`xc.XEle_GetBkManager`、`xc.XEle_ClearBkInfo`、`xcc.Element_State_Flag_Leave`(元素态, ≠ Window_State_Flag_Leave)、`XWnd_GetChildCount/GetChildByIndex`、`XEle_GetAlpha/GetBkInfoCount`、`wapi.GetWindowRect` + `PrintWindow(PW_RENDERFULLCONTENT)` 抓窗、`XWnd_SetTimer/KillTimer + AddEvent_Timer`
- 排查"窗口可见但屏幕上看不见"(见 57):`user32!EnumWindows` + `GetWindowThreadProcessId` 枚举进程窗口、`IsWindowVisible`、`user32!GetLayeredWindowAttributes`(读 layered alpha/颜色键)、`PrintWindow` 与 `CopyFromScreen` 对照(画了 vs 屏幕上没有)
- `app.GetTextShowRect`(文本测量:限宽换行与绘制共用限宽;返回的 CY **已含行距** —— 13px 字体单行 18 / 两行 36, 见第八节 75)
- 折行文本(见第八节 90/91):`Label.WrappedHeight(width)`(当前字体下按宽度折行的高度;读的是 Label **当前文本**,先 SetText 再量)、`Label.SetWrap()`(只换行,调用方管高度)、`Label.SetAutoWrap()`(按当前宽度自动量高 + 自撑,尺寸/文本/字体变化时重排)、`Label.SetAlignCenter()/SetAlignLeft()`、`Label.SetColorFunc(fn)`(语义字色按当前主题实时解析)、`xui.TextWidth(text, hFont)`(单行真实宽度, dialog 页脚按它定宽)、`xc.XC_IsHXCGUI(h, xcc.XC_LAYOUT_FRAME)`(识别滚动容器框架)、`xc.XWnd_AdjustLayout`(窗口级重排;`XEle_AdjustLayout` 不能直接作用于 LayoutFrame 的内容区)
- `Element.SetBorderSize(左, 上, 右, 下)`:原生 Edit/ComboBox 的**文字内边距**, 也是原生 Button 载体的文字区内缩(默认四边 1); 传**逻辑像素**、由框架随 DPI 自动缩放(实测 150% DPI: Input 传 10, 文字起点距边框 20 物理 ≈ 13.3 逻辑 = 10 + 字形左侧留白) —— 别再乘 `dpi/96`, 会重复放大; Label 清上下内缩的用法见第八节 109
- 窗口真透明:`Window.SetTransparentType(xcc.Window_Transparent_Shaped)` + `Window.SetTransparentAlpha(255)`、`xcc.Window_Transparent_Shaped`(xcconst.go:55);**对原生自建弹窗**(ComboBox 下拉, 只有窗口句柄)同样可用: `xc.XWnd_SetTransparentType / XWnd_SetTransparentAlpha`, 在弹出回调里(显示前)设置, 见第八节 69 —— **单次设置可信, 逐帧改会概率性冻结**(57/69), 原生弹层不做整窗 alpha 动画
- 背景对象美化:`BkManager.AddFill/AddBorder(state, color, id)`、`BkManager.GetObjectObj(id) *bkobj.BkObj`、`BkObj.SetRectRoundAngle/SetMargin/SetAlign/SetRotate/SetFillColor/SetBorderColor`、`xcc.BkObject_Align_Flag_*`(bkmanager.go:69/185, bkobj.go:104/42/50/66/74/90)
- Dialog:`XEle_SetZOrderEx(h, 0, xcc.Zorder_Top)`、`XEle_AdjustLayout`、`XEle_Redraw`、`XEle_GetChildCount/GetChildByIndex`、`XObj_GetType`(→ `xcc.XC_*`)、`XWidget_GetParentEle`、`XWnd_GetRect/Redraw`、`XWnd_SetFocusEle/GetFocusEle`、`xcc.AdjustLayout_No`、`wapi.GetAsyncKeyState`
- 输入组 / 自动补全(见第八节 147/148):`Element.AddEvent_SetFocus / AddEvent_KillFocus`(裸输入框聚焦进出时让外层容器重绘焦点环)、`xui.StateOf(h).Focused`(在容器 Paint 里判焦点)、`Element.AddEvent_LButtonUp(hEle, nFlags, *xc.POINT, *bool) int`(徽章/补全条目的点击回调)、`xui.Hex("#RRGGBB")`(自定义配色, 徽章 `WithBadgeColors`)
- 整组透明度:`xui.SetSubtreeAlpha(v, roots...)`(逐个子元素设 alpha, 不级联)、`xui.SetSubtreeAlphaSkipScrollBars(v, roots...)`(跳过滚动条子树, 弹窗打开时的淡入用, 见 98)、`Element.AddEvent_Show(fn(hEle, bShow, pbHandled))`(元素显示/隐藏事件; 滚动条在宿主隐藏时归零自己的显隐状态)
- Toast(见第八节 75): `XWnd_GetBodyRect`(Body 矩形, 阴影窗口有 band 偏移)、`XWnd_GetStayEle`(窗口级悬停判定)、`XEle_SetXCTimer/KillXCTimer + AddEvent_XC_Timer`(每条通知自己的倒计时)、`XEle_Destroy`、`Element.AddEvent_Destroy_End`、`Window.AddEvent_Size/MouseMove/MouseLeave`、`XEle_SetZOrderEx`
- 屏幕通知(见第八节 177): 独立 `Window_Transparent_Shaped + WS_EX_TOOLWINDOW|WS_EX_NOACTIVATE` 小窗(建窗/显示同弹层)、`XWnd_GetDPI` + `MonitorFromWindow/GetMonitorInfo`(物理工作区换逻辑)、`Window.AddEvent_WindProc`(宿主 `WM_DESTROY` 清理)、`Window.AddEvent_MouseMove/MouseLeave`(整组悬停暂停)、`xui.OnThemeChangeFor`(独立小窗重排)、`xui.Presence`/`xui.SpinDetached`(动画挂宿主)
- 托盘(见第八节 178): `window.NewTrayIcon / TrayIcon.Show/SetIcon/SetTips/SetPopupBalloon/Modify/Reset`、`Window.AddEvent_TrayIcon`(wParam=托盘 ID, lParam=WM_LBUTTONDOWN/WM_RBUTTONDOWN)、`wapi.RegisterWindowMessageW("TaskbarCreated")`(Explorer 重启恢复)、`wutil.HIcon`(只接受文件路径)、`wapi.LoadImageW + wapi.GetModuleHandleW`(从 exe 的 `RT_GROUP_ICON` 资源取 HICON, 免临时文件; `LR_SHARED` 不释放)、`wapi.DestroyIcon`(自有 HICON 释放)、`Window.IsMaxWindow`(左键还原时保持最大化)
- 窗口图标(见第八节 180): `window.Window.SetSmallIcon`(任务栏预览窗口左上角, 也影响任务管理器) / `SetBigIcon`(任务栏 + Alt+Tab); 参数都是 HICON, 可与托盘共用 `wapi.LoadImageW` 从 exe 资源取的 `LR_SHARED` 图标(不释放)
- Table(原生 List 载体):`CreateAdapterHeader/CreateAdapter/AddColumnText/SetHeaderHeight/SetRowHeightDefault/SetRowSpace/SetColumnWidth/SetSplitLineColor/SetDrawRowBkFlags/AddRowText/SetItemText/GetCount_AD/SetSelectRow/GetSelectRow/GetVisibleRowRange/RefreshRow/RefreshData/AddEvent_List_DrawItem/AddEvent_List_Header_DrawItem/AddEvent_List_Temp_Create_End/AddEvent_List_Header_Temp_Create_End/AddEvent_List_Select/GetTemplateObject/GetHeaderTemplateObject/GetHeaderHELE/GetScrollBarVObj`
- ⚠️ `TabBar.SetPadding` 在 xcgui.dll 4.0.0.0 中不存在(调用 panic)
- 直通 D2D(图标抗锯齿, 见第八节 43):`xc.XDraw_GetD2dRenderTarget(hDraw) → *ID2D1RenderTarget`、`xc.XC_GetD2dFactory → *ID2D1Factory`、`xc.XDraw_ConvRect(hDraw, *RECT) / XDraw_ConvXY`(客户区坐标 → 画布对齐物理坐标, 第三方绘图必须先换算); 用到的 D2D 方法: `ID2D1Factory::CreatePathGeometry`、`ID2D1PathGeometry::Open / GetBounds`、`ID2D1GeometrySink::{SetFillMode, BeginFigure, AddLines, EndFigure, Close}`、`ID2D1RenderTarget::{CreateSolidColorBrush, FillGeometry}`
- 官方自绘示例:BeautifyButton / CheckBox / RadioButton / Edit / ProgressBar / SliderBar
- 剪贴板读(InputOTP 粘贴, 见第八节 110 所在组件): `wapi.IsClipboardFormatAvailable(CF_UNICODETEXT) / OpenClipboard(0) / GetClipboardData / CloseClipboard` + `wapi.GlobalLock/GlobalUnlock`; GlobalLock 返回的 uintptr 别直接转 `unsafe.Pointer`(vet 报"可能的误用"), 用同布局的自定义 slice 头读 UTF-16; `wapi.GetAsyncKeyState(VK_CONTROL)` 判 Ctrl(VK 状态位按 `uint16` 与 `0x8000` 比, int16 会溢出)
- 弹层内容测量/非激活显示(Popover/HoverCard, 见第八节 115/116/181): `xc.XWnd_GetDPI`、`SetWindowPos(..., SWP_NOACTIVATE|SWP_SHOWWINDOW)`、`wapi.ShowWindow(hwnd, SW_SHOWNOACTIVATE=4)`、`xcc.WS_EX_NOACTIVATE`、`xc.XEle_SetRect`、`xc.XEle_AdjustLayout`、`Window.AddEvent_WindProc` + 对 `WM_MOUSEACTIVATE` 返回 `MA_NOACTIVATE`(3, 标记 handled); 关闭检查用 `wapi.GetCursorPos` + `wapi.GetWindowRect` 与 `XEle_GetWndClientRect` 的物理换算
- 屏幕弹窗(见第八节 182): `MonitorWorkArea` + `XWnd_GetDPI` 换算工作区、`popoverPanel.MeasureContent` 离屏量高(可点击激活模式下, 量高与显示仍都用 `SW_SHOWNOACTIVATE`, 不主动抢前台)、`content.AdjustLayout` + `bodyLayout.AdjustLayout` 同步新增内容高度、`popoverPanel.HideWithFade` 退场淡出(同一 `presenceKey` 反向实现淡出中重新显示; 屏幕弹窗用 `SetShadowInfo` 的 depth 与元素 alpha 同步降阴影浓度)、`app.AfterFunc` 定时自动关闭(自增代次丢弃已取消的迟到回调)、`xc.WndEventBus.RemoveCallback(hWindow, xcc.XWM_WINDPROC, id)` 反注册宿主销毁监听
- 圆形头像图片(见第八节 117): `xc.XImage_LoadFromDataRGBA(data, w, h)`(非预乘 RGBA, 内部转 BGRA)、`xc.XImage_Destroy`、`drawx.Draw.XDraw_ImageEx(hImage, x, y, w, h)`; `image.Decode`(png/jpeg/gif) 与 CPU 侧蒙版在 `avatar.go`
- 命令面板: 原生 `widget.NewEdit` 透明化(EnableBkTransparent/EnableDrawBorder(false)/EnableDrawFocus(false)/SetBorderSize(0,0,0,0)/EnableAutoSelAll(false)) + `SetDefaultText/SetDefaultTextColor/SetCaretColor/SetTextAlign(Left|Center_V)`; `AddEvent_Edit_Changed` / `AddEvent_KeyDown` 接管 ↑↓/Home/End/Enter/ESC(原生箭头键会先到这里, 标记 handled 即可); 列表滚动用 `ScrollArea` 的 `GetViewHeight/GetViewPosV/ScrollPosYV`
- 字体:`xui.FontSpec{Family, Size, Bold, Italic}`。**没有下划线字段**: `xcc.FontStyle_Underline` 在本框架的 D2D 文本路径下不渲染(实测: 下划线字体与普通字体像素一致), hover:underline 一律手绘 1px 线 —— 宽度取 `xui.TextWidth`, 落点见 Button 的 link 变体与 Accordion 触发区。9pt 以下的**常规**字重由 `FontHandle` 从字体文件创建(GDI+ 取英文族名 + Fonts 注册表找文件), 绕开 xcgui 的小字号宋体替换(见 149); 组件一律用 `FontHandle`, 不要直接 `font.NewEX`
- 弹层键盘钩子(`popoverPanel`, Select 的 ↑↓/Enter 用): `Window.AddEvent_KeyDown`(窗口级, 焦点在子元素上也先于元素分发)、`popoverPanel.OnKeyDown/OnEsc`(popup.go 预留的钩子) + `Element.SetFocus`(焦点即条目高亮)
- 轮播(见第八节 140/141): 纯自绘根 Element + 视口 + 每页一个 LayoutEle; 位移过渡复用 `xui.Presence`(`DropTransitions` 后每帧写各页矩形); 拖拽用窗口级 `AddEvent_LButtonDown/MouseMove/LButtonUp` + `Window.SetCaptureEle` + `AddEvent_KeyDown`(焦点在轮播内才响应); 上/下张用 `Button`(outline + icon-sm + rounded-full); 循环位置取正模
- 可调分栏(见第八节 157): 分组/面板都是 LayoutEle(主轴 Fixed 由百分比算出、交叉轴 Fill); `xc.XEle_GetWndClientRectDPI`(物理客户区矩形 —— 与逻辑坐标的 `XEle_GetWndClientRect` 区分)、`wapi.ClientToScreen`、`wapi.SetCursorPos`、`wapi.LoadImageW(0, wapi.IDC_SIZEWE/IDC_SIZENS/IDC_ARROW, wapi.IMAGE_CURSOR, 0, 0, wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)` + `Element.SetCursor`(col/row-resize 光标)、`Element.AddEvent_LButtonDBClick`(双击复位)
- 侧边面板 / 抽屉 / 菜单栏(见第八节 120~125): `XWnd_GetBodyRect`(遮罩铺 Body)、`XEle_SetAlpha`(遮罩淡入, 不级联子元素)、`XEle_GetRect/SetRect`(面板整体平移, 逻辑像素)、`XEle_SetCapture`(抓手拖拽捕获鼠标)、`wapi.GetCursorPos`(拖拽用屏幕坐标算增量)、`wapi.GetForegroundWindow/SetForegroundWindow`(子菜单与父弹层的焦点归属判定)、`app.AfterFunc`(弹层关闭后延迟把焦点还给触发区, 否则被随后的窗口激活覆盖)
- 官方透明/异型窗口示例:`Advanced/WindowPartiallyTransparent`、`webview/WebViewPartiallyTransparent`
- 图表(见第八节 159): 纯自绘根 Element, 绘制全走 paint 回调里的 `*xui.Painter`(`Line`/`Text`/`FillRound`/`StrokeRound`/`FillRoundCorners`)与 `p.Draw` 的浮点几何(`FillPolygonF` 画扇形与面积带、`FillEllipseF`/`DrawEllipse` 画点与圆环); 悬停靠宿主窗口级 `AddEvent_MouseMove` + `xc.XEle_GetWndClientRect` 命中(ScrollArea 里的元素收不到元素级 MouseMove, 同 156); 隐藏页判定沿 `XWidget_GetParentEle` 链查 `XWidget_IsShow`; 曲线**自己采样**成分段折线(D2D 下曲线绘制是空实现); 提示卡与图例都在同一个 Paint 回调里画, 不建小窗
- 侧边栏(见第八节 158): `wapi.GetCursorPos` + `wapi.ClientToScreen(hwnd, &pt)` + `xc.XEle_GetWndClientRectDPI`(物理客户区矩形) + `xc.XWnd_GetDPI` 换算动作钮命中; `Element.SetZOrder(0)`(构造期抬升 Z 序, `SetZOrderEx` 在窗口未显示时会崩); `Window.AddEvent_MouseMove/AddEvent_MouseLeave`(普通 Element 的悬停兜底); `XWidget_IsShow`; `Painter.TextF` + `xcc.TextFormatFlag_NoWrap|xcc.TextTrimming_EllipsisCharacter`(菜单项长文本省略); `xui.Hover` + `Presence`(宽度 200ms 过渡, 挂宿主窗口)
- 自绘 chips 容器 + 裸 Edit 子控件(TagInput, 见第八节 187): `Element.SetRectEx(x,y,cx,cy,false,xcc.AdjustLayout_No)`(普通 Element 未开布局时给子控件定坐标, 布局趟不会覆盖)、`Edit.AddEvent_Edit_Changed/AddEvent_KeyDown/SetDefaultText/SetDefaultTextColor/SetCaretColor/MoveEnd`、`Element.AddEvent_MouseMove/AddEvent_MouseLeave`(显式 `EnableMouseThrough(false)` 后普通 Element 每次移动都触发, pPt 是元素客户区坐标)、`app.AfterFunc(0, ...)`(把量高推出当前输入/尺寸事件并合并同轮触发)
- 状态栏(StatusBar, 语义见第四节): `xc.XEle_SetPadding`(左右内边距) + `xcc.Layout_Size_Weight/Auto` 三槽 + `EnableMouseThrough`(布局壳保持穿透, 见 185)、`xc.XEle_SetCapture` + `wapi.GetCursorPos`(尺寸抓手拖拽, 屏幕物理增量 ÷ `xc.XWnd_GetDPI`)、`xc.XWnd_GetRect/XWnd_GetClientRect`、`window.Window.SetSize`(底层 `XWnd_SetRect` 吃逻辑像素)、`Window.IsMaxWindow`、`Window.AddEvent_WindProc` + `wapi.WM_SIZE` + `xc.XC_CallUT`、`wapi.LoadImageW(0, wapi.IDC_SIZENWSE, wapi.IMAGE_CURSOR, ...)` + `Element.SetCursor`、`drawx.Draw.DrawLineF/SetLineWidthF`(1px 斜线避免整数截断, 见 186)、`Tooltip.BindTip(..., WithTipSide(TipTop))`、`app.AfterFunc(0, ...)`
