# xui — shadcn 风格的 xcgui 自绘组件库: 自定义指南

> 这份文档随 `xui init` 安装到你的 `ui/` 目录, 是组件源码的配套说明书。源码注释里的
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
> 全部结论基于 xcgui **1.4.2** 实测; 组件实现以 `ui/*.go` 为准。交互语义的原始参考是
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

> **字段命名约定**: 字段名 = shadcn CSS 变量的**直接 CamelCase, 不做缩写** —— `--accent-foreground` → `AccentForeground`。早前实现曾缩写成 `AccentFore`, 已统一(否则批量转译社区主题要带一张特例表)。
> 两处与上游的差异要知道:① `DestructiveForeground` 是本库的**扩展令牌** —— nova 预设已删除 `--destructive-foreground`(破坏性按钮直接写死白字), 我们保留它以便组件取"破坏色上的文字"; ② `--chart-1..5` 与 `--sidebar-*` 已建模(`Chart1..5` / `Sidebar*`), **组件库自己不用、是给用户应用取用的** —— 机械转译社区主题时一并填上即可(内置主题的换算值与 `manager_test.go` 的漏填守卫都在)。
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

**API 形状决策(早前设计稿写的 `xui.Themes.Dark` 已废弃)**:内置主题用**包级函数** `xui.Light / xui.Dark`,**不提供 `xui.Themes` 命名空间对象**。Go 里"包本身就是命名空间",用一个全局单例对象当模块不合惯例(收益只是省一个点号,代价是多一个可变全局 + godoc 层级更深)。将来做第七节 P4 的"主题画廊"(几十上百个主题, 包级函数会污染根包名)时,正确解法是新增 **`xui/themes` 子包**(`themes.Get(name)` / `themes.All`, 与本库 `icons` 子包同构),根包保留这两个函数作默认兜底;届时 `xui.Light` 迁到 `themes.Light` 保留别名即可, 非破坏。

> 相关实现约定:`OnThemeChange` 返回的反注册函数**必须真的生效**(早期实现用 `&f == &fn` 比函数地址, 恒为假 → 是空操作, 已改为自增 id 句柄);窗口底色绑定(`BindWindowBackground`)由管理器持有, `UnregisterWindow` 会一并解绑, 避免主题切换时对失效句柄调 `XWnd_*` 弹错误框;**组件构造函数里的主题订阅一律用 `xui.OnThemeChangeFor(元素句柄, fn)`**(元素销毁时自动反注册, 见第八节 54), 裸 `OnThemeChange` 只用于非元素的长生命周期订阅。

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
| AppWindow | 原生 window.Window 封装 | 无系统标题栏(样式不含 Icon/Btn_*): 阴影圆角 + 自绘 `[图标][标题 Weight][自定义按钮区][最小化/最大化/关闭]`; 双击标题栏最大化/还原; 内容区 = 纵向流式 Weight 1(第八节 76) | demo 窗口(已迁移) |
| Button | embed Button | paint 壳, 原生文字(WithIcon 时自绘图标) | BeautifyButton 示例 |
| Card | LayoutEle(布局容器) | 圆角面板 + ring 描边; 内容区是 Fill 子项、内边距用外边距表达(随卡片缩放), 高度可按内容自撑(Auto), 见第八节 56 | — |
| Checkbox/Radio | embed Button | 画方框/圆 + 勾线, 文字原生 | BeautifyCheckBox/RadioButton 示例 |
| Input | embed Edit | 只画边框+焦点环, 原生光标/IME | BeautifyEdit 示例 |
| Switch | 纯自绘 Element | 轨道+滑块, 尺寸预设; hover 过渡走运行时 `anim.go`(第八节 45) | — |
| Label/Separator/Badge | 纯 Element | paint | — |
| Progress | 纯自绘 Element | h-1 圆角轨道 + 指示器(宽度按百分比) | BeautifyProgressBar 配方移植成 token |
| Slider | 纯自绘 Element | **弃用原生 SliderBar**(滑块按钮 BkInfo 残留压不住, 见第八节 25), 自实现拖拽/点击跳转/键盘 | BeautifySliderBar 示例 + 探针 |
| ComboBox | embed ComboBox | 画输入框; 下拉 ListView 弹层在 `AddEvent_Combobox_Popup_List` 里半接管(第八节 30) | 库文档 FAQ |
| Tabs | embed TabBar | paint(default/line 两变体) | — |
| Dialog | 纯 Element 覆盖层 | 全窗遮罩元素(自绘黑 10% + backdrop-blur) + 面板子元素;ESC/点遮罩关闭, Tab 焦点循环(第八节 33) | — |
| Menu | 原生菜单 | `XMenu_EnableDrawBackground/EnableDrawItem` + `AddEvent_Menu_Draw_Background/DrawItem`(第八节 36) | element.go 已确认事件存在 |
| Tooltip | FloatWindow | paint, 异型透明窗口 + 进出场 fade/slide(第八节 24/47) | — |
| Table | 原生 List + 项模板 | 表头/行底/边框/滚动条全自绘(绘制回调按**单元格**触发, 见第八节 40); 原生 Table(Shape 系)与 shadcn 语义不符, 不单独做 | Basic/List 示例 + 本库探针 |
| Tree | 原生 Tree + 项模板 | `XE_TREE_DRAWITEM` 自绘行底/参考线/chevron, 模板子元素保留(第八节 42) | 库文档 + 探针 |
| Pagination | 纯自绘单元素 | 条目行 `[上一页][页码…][下一页]`, 逐项 hover 过渡; 恒定槽位折叠(第八节 53) | — |
| Icon | 直通 D2D 几何绘制 | lucide path → 描边几何 + `DrawGeometry`(圆角样式), 色点走 `FillGeometry`(第八节 43); 回落折线描边 / `svg.Svg` | svg 包 + D2D 探针 |

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
| active:translate-y-px 按下整体下沉 1px(含文字→文字手动绘制) | button.tsx 基类 |
| destructive 软色调(bg-destructive/10 + 红字) | button.tsx / badge.tsx |
| radio 选中 = bg-primary 实心圆 + size-2 白点(bg-primary-foreground) | radio-group.tsx |
| checkbox size-4 rounded-[4px] border-input;未选**亮色无底色 / 暗色 dark:bg-input/30**;选中 bg-primary + 白勾;禁用 = 上面每个图层各乘 50% | checkbox.tsx + 第八节 61 |
| card 描边 = ring-foreground/10(非 Border 令牌), rounded-xl;卡片头 = 一行两列(左列标题+说明 `1fr`, 右列动作 `auto`), 动作区贴右上、跨标题与说明两行顶端对齐(gap-1 = 4) | card.tsx + `ui/card.go` 的 `CardHeader` |
| **aria-invalid**(表单校验失败): 边框变 destructive + ring-3 destructive/20(暗色 border /50、ring /40), **常驻不要求焦点**;与 focus-visible 同时成立时红色优先(变体顺序里 aria-invalid 在后) | button.tsx / input.tsx / select.tsx + 第八节 69 |
| badge h-5 胶囊(rounded-4xl) | badge.tsx |
| switch 轨道 32x18(选中 bg-primary/未选 bg-input, **暗色未选是 dark:bg-input/80**),滑块 16 圆、两侧各留 1px;dark 下选中滑块用 primary-foreground、未选用 foreground;**禁用时轨道与滑块一起乘 50%**(滑块不跟班会剩一颗全亮白圆) | switch.tsx(radix-nova) + 第八节 61 |
| switch 元素比轨道每边大 3px:给 focus-visible 的 ring-3 留绘制空间,同时外扩点击热区(≈ after:-inset-*) | switch.tsx |
| progress = h-1 rounded-full bg-muted 轨道 + bg-primary 指示器(宽度按百分比) | progress.tsx |
| slider = **纯自绘**(普通 Element, 不用原生 SliderBar——其滑块按钮 BkInfo 残留绘制压不住, 见第八节 25):轨道 h-1 bg-muted + 已完成区间 bg-primary + 滑块 size-3 border-ring bg-white(字面量 white),hover/拖动中/focus-visible 显示 ring-3 | slider.tsx |
| slider 元素比可见轨道左右各宽 9px(滑块半径 6 + ring 3, 给两端 ring 留绘制空间, 见第八节 26);滑块中心在轨道两端之间移动, 位置由 pos 自算 | slider.tsx(radix 滑块中心贴轨道端点) |
| slider 交互全自实现:点击跳转 + LButtonDown 起 `XEle_SetCapture` 拖拽 + MouseStay/MouseLeave 自跟踪 hover(拖完移开光环即消失, 与 shadcn 一致)+ KeyDown 左右 ±5%/Home/End | — |
| tabs 列表 rounded-lg p-[3px] h-8 bg-muted,标签等宽 rounded-md,未选 text-foreground/60、选中 bg-background;line 变体透明列表 + 底边 h-0.5 下划线;标签 text-sm **font-medium(500)≈中文常规粗细,勿用 Bold(700)**(实测对比) | tabs.tsx |
| tooltip = bg-foreground rounded-md px-3 py-1.5 text-xs text-background + size-2.5 旋转方块箭头,延迟 0;窗口为异型透明,气泡外真透明 | tooltip.tsx |
| select 触发区 h-8 rounded-lg border-input **bg-transparent(放卡上 = Card 色)** + 右侧 chevron-down 用 **text-muted-foreground 全色**(非前景半透明);触发区对滚轮无响应(`EnableEvent_XE_MOUSEWHEEL(false)`);弹层 = **item-aligned 定位**(盖住触发元素, 选中项正对触发位置, select.tsx `position="item-aligned"`);弹层条目 = p-1 容器 + 条目 h-7(28) rounded-md hover:bg-accent + 选中项右侧 Check 图标 + 打开时自动高亮选中项;ring-1 描边画在 `XE_PAINT_END`(内缩 1px, 见第八节 30);内容高度自适应(≤10 项不出滚动条);**弹层进场 fade-in 100ms**(整窗 alpha: 弹出回调里转 `Window_Transparent_Shaped`, 见第八节 30/69);**aria-invalid** = 红边框 + ring-3 destructive/20(暗色 /40), 见上表 | select.tsx + combobox.go(弹层接管路线见第八节 30) |
| menu 弹层 = **阴影弹窗**(rounded-lg 由阴影窗 nAngeleSize 逐像素画出 + shadow-md 投影, 官方 DrawMenu 同款, 见第八节 36/64) + bg-popover + ring-1 ring-foreground/10(内缩 1px); **p-1 = 菜单 SetBorderSize(4)**(默认 3); 进场 fade-in 100ms(整窗 alpha, 无位移 —— 位移会崩, 见 64); 条目 h-7(28) rounded-md px-1.5, hover/focus bg-accent + 文字变 accent-foreground; destructive 红字 + hover bg-destructive/10(dark /20); disabled 文字 50% alpha 不响应; 勾选/单选选中项右侧 Check 图标(right-2 size-4), 单选组内互斥; 分组标签 text-xs muted; 分隔栏 1px bg-border **全宽**(-mx-1; 原生槽高仅 4px, shadcn my-1 的 9px 不可达, 已知差异); 快捷键右对齐 text-xs muted; `MenuInset` 条目文字左缩进到 28(pl-7); 弹层宽 = `Width(w)` 显式指定, 否则 max(触发区宽, 128)(shadcn 的 `w-(--radix-dropdown-menu-trigger-width) min-w-32`, 经 `XMenu_SetItemWidth` 逐条目设"目标宽 − 开销"实现, 长文本硬裁剪, 见第八节 64); 右键菜单(PopupAtCursor)无触发区, 退化为内容自适应 + 128 下限; 子菜单条目不跟宽(shadcn SubContent 内容自适应 + min-w-96) | dropdown-menu.tsx + menu.go(原生菜单皮肤, 见第八节 36/64) |
| table 表头 = h-10(40) px-2, 文字 foreground, 只有一条 1px 底边(border);数据行 = p-2 + text-sm(36), 行 `border-b`, **最后一行不画**;行悬浮 `hover:bg-muted/50`, 选中 `data-[state=selected]:bg-muted`;表体无外框/无圆角/无纵向网格线(外框交给 Card), 容器 `overflow-x-auto`, `caption-bottom text-sm`;自适应列宽按**元素当前宽度**算, 由组件订阅元素尺寸事件自愈(见第八节 60) | table.tsx + table.go(原生 List 接管路线见第八节 40) |
| tree 行高 32(h-8), 视觉矩形左右内缩 4(mx-1), 圆角 rounded-md(6);行悬浮/选中都 bg-accent(同 select/menu 条目);每级缩进 16(size-4), 祖先层级画 1px 参考线(border 色);chevron size-4 text-muted-foreground, 收起朝右/展开朝下(= rotate-90 两态), 叶节点不画但留空槽位(文字全树对齐);条目文字 text-sm foreground;shadcn 无官方 tree, 语义为社区通行实现 + 本库令牌 | App.tsx 手写 TreeRow 对照卡(无官方 tree.tsx) + tree.go(原生 Tree 接管路线见第八节 42) |
| pagination 入口居中(justify-center), 页码 32 方形(ghost / 当前页 outline), 上一页/下一页 = chevron(size-4)+文字, 到边界按禁用画(50% 透明且不响应鼠标), 页数 >7 折叠成 首/末页 + 当前页±1 + 省略号(不可点);本实现用**恒定 7 槽位**(见第八节 53) | pagination.tsx + 第八节 53 |
| tooltip 绑定元素自身有弹层(如 ComboBox)时, 弹层打开要主动调 `HideTooltip(hEle)`——弹层窗口接管鼠标后触发元素收不到 MouseLeave | tooltip.go / combobox.go |
| scroll area: 页面/长内容滚动用 `ui.ScrollArea`(原生 LayoutFrame 载体, 见第八节 66): **只做纵向**(内容区恒铺满视口宽); 滚动条**鼠标移入容器才显示、离开淡出**(同 radix 的 `type=hover`, 150ms; 用元素透明度实现, 不改变布局宽度)、10px 轨道 + 8px 圆头滑块(`rounded-full bg-border`)、轨道透明; 滑块悬停**不换色**; 滚轮悬停任意子元素都滚动(原生控件如 List 自吃 —— 嵌套滚动常态); 该规格统一在 `ui/scrollbar.go` 的 `styleScrollBar` + `showScrollBarOnHover`, Table / Tree / ComboBox 弹层 / 多行 Input 同用(**滑块配色与容器不同**: 这四个载体在原版里是浏览器原生滚动条, 用"前景色淡化 + 悬停加深", 见第八节 21) | scroll-area.tsx(w-2.5 p-px `rounded-full bg-border`; radix `type` 默认 hover 淡入 —— 本实现的差异见"已知差异"表) |
| **disabled:opacity-50 是组级透明**:组件自绘的**形状**(壳体/轨道/勾/滑块/圆点)**逐层各乘 50%**(`ScaleAlpha`, 见第八节 61);**原生文字**不吃 alpha —— 按钮文字/复选框与单选框的标签保持原色(实测浏览器禁用实心钮白字核心=255,给文字加 50% alpha 只有 194),输入框禁用文字用"前景色×卡片色对半混"预混后发给原生 Edit。radio 的 Label 不跟随禁用 | button/checkbox/radio/tabs/switch/slider/input.tsx |
| dialog 面板 = w-full sm:max-w-sm(384) rounded-xl bg-popover p-4 gap-4 + ring-1 ring-foreground/10(ring 画在面板矩形**外侧** 1px, 直接描面板矩形会被遮罩灰吞掉, 见第八节 33);标题 text-base(16)/描述 text-sm(14) muted,header gap-2 | dialog.tsx |
| dialog 关闭钮 = 幽灵图标钮 size-7(28×28) top-2 right-2(X 图标 size-4);页脚 -mx-4 -mb-4 贴面板左右与底边、border-t、bg-muted/50、p-4、gap-2、右对齐(只有下两角随圆角) | dialog.tsx |
| dialog 遮罩 bg-black/10(固定黑色,不随主题;**只铺 Body 区, 不铺阴影窗口的阴影带**——铺上去会把窗口自身投影再压暗 10%, 四角最明显;**backdrop-blur-xs 已实现**——物理像素抓窗快照(PrintWindow 不缩放, DIB 必须给物理尺寸)+融合降采样+半分辨率盒式模糊+成对平均放大+BkM 图片填充,见第八节 47) | dialog.tsx(语义权威) + 第八节 47 条 6/10 |
| dialog 打开时焦点进第一个可聚焦控件,Tab/Shift+Tab 在对话框内循环(焦点陷阱),ESC / 点击遮罩 / 关闭钮关闭 | radix dialog 语义 |
| **中文 500 字重 ≈ 常规**: Label 默认常规、按钮文字常规(用粗体字形实现 500 会明显偏粗); 标签比描述"看着粗"是**颜色**差(foreground 近黑 vs muted 灰), 实测笔画同宽 | label.tsx 500 + 实测像素 |
| 令牌表 nova 预设 neutral 色板(oklch 已换算 hex) | index.css :root/.dark |
| **字体单位**: xcgui 字体 size 是 pt 不是 px, FontHandle 统一按 px×0.75 换算(14px→11pt); 之前全局文字放大 1.33× | font.go(见第八节 31) |
| calendar 卡片 = bg-popover rounded-lg + ring-1 ring-foreground/10(popover 皮肤; 弹层里用 `WithCalendarFlat` 免双层边框); 格子 = `--cell-size` 方格(设计值 2rem=32) + rounded-md(6); **默认尺寸 278×327 = 设计基准 ×1.12**(格子 36 / 字号 16), 且所有几何(内边距/格子/标题/星期行/页脚/字号)都按**元素宽度等比缩放**, 高度决定 6 行之间的行距(多余空间均摊) —— 元素拉大时内容一起变大, 不会只剩空白; 标题行高 = 格高, 年月**两个热区**居中(font-medium + chevron-down, 悬停/展开 bg-accent), 左右各一个 ghost 翻页钮(size-8, hover bg-accent, chevron 16); 星期行 0.8rem muted; **固定 6 行**(月份切换卡片高度不跳, 弹层尺寸恒定); 选中 bg-primary + text-primary-foreground, 今天 bg-accent, 悬停 bg-accent(150ms 过渡), 非本月 text-muted-foreground; 周起始日默认周一(中文习惯), `WithCalendarWeekStart` 可调 | calendar.tsx(react-day-picker v9 的 classNames) + calendar.go |
| calendar 年月下拉 = 点标题热区弹出自绘列表(月 12 项 / 年 = 视图年±50 可配), 面板 bg-popover rounded-md + ring, 行高按"12 行塞满可用高度"自适应(16..26), 当前值右侧 check, 超出可视高度时滚轮滚动 + 2px 细滚动条, 面板内点击=取值、面板外点击=关闭、点另一个标题热区=直接切换; 只改显示月份不动选中日期(react-day-picker `captionLayout="dropdown"` 语义) | 参考截图里的 select 弹层形态 + calendar.go |
| calendar 禁用日期 = `disabled` matcher(内置禁用过去/周末/日期数组 helper), 禁用格 text-muted-foreground opacity-50, **不可点、无悬停** | react-day-picker 的 `disabled` + `.rdp-disabled` |
| calendar 页脚 = 卡片全宽 1px border-t + 上下 6px + 24px 按钮行, 项水平居中、间距 8; 按钮 ghost 化(muted-foreground → 悬停 bg-accent + accent-foreground), 纯文本项只有文字; 元素未指定高度时启用页脚自动长高 `CalFooterHeight`(37) | shadcn calendar 的 footer 示例 + 参考截图"今天" |
| date picker 触发区 = **Button variant="outline"**(不是 Input): h-8 rounded-lg border-input bg-background, 左侧 size-4 calendar 图标 + 文字, hover bg-accent / 文字 accent-foreground; 未选择显示占位符(muted); 弹层 = 自建阴影窗 + Calendar(sideOffset=4, 下方放不下翻上方, 夹在**显示器工作区**内 —— 扣掉任务栏, 见第八节 74); 进场 fade-in 100ms; 选中即回写+关闭+回调, 点外部(弹层失焦)/ESC 关闭; **点触发区关闭后 250ms 内不再重开**(弹层 KillFocus 先于触发区 LButtonUp, 否则表现为"点两次才关掉"); aria-invalid 同 Button | date-picker 文档示例(Button outline + Popover + Calendar) + datepicker.go / popup.go |
| time picker = 触发区同 date picker(clock 图标 + "09:30") + 弹层两列滚动列表(时 0-23 / 分 0-59): 每列 64 宽、8 项可见(28×8), p-1 内边距; 载体是**原生 ListBox**(滚动/滚轮/点击都是原生), 条目背景与文字自绘 —— 选中 bg-primary + text-primary-foreground, 悬停 bg-accent(150ms 过渡); **选择即生效但弹层保持打开**(继续改另一列), 点外部/ESC 关闭; 打开时选中项滚到列表中间(行步进 = 项高+1, 见第八节 72) | shadcn 无官方 time-picker, 语义取社区通行形态(Popover + 滚动列表) + timepicker.go |
| toast(sonner) = 右下角堆叠(新的在最下)、卡宽 356、与内容区边缘 24、卡间距 10; 卡片 bg-popover + border + rounded-lg + shadow-md, 内边距 16, 图标 20(与文字间距 10), 标题 13px font-medium / 描述 13px 可折行, 动作按钮 h-26 bg-muted hover:bg-accent, **关闭按钮默认隐藏**(同 shadcn 的 sonner), 显示时在右上角; 类型(success/error/warning/info)只改图标与图标色, 不给卡面上底色(未开 richColors); 自动关闭 **4s**、**悬停任一条暂停全部倒计时**、超出可见条数(默认 3)移除最旧; 进场淡入 + 从堆叠方向滑入 12px(200ms), 退场淡出 + 向侧边滑出 24px(180ms); 类型快捷入口 `Success/Error/Warning/Info`, `Duration < 0` = 常驻 | sonner 默认值(宽度/offset/gap/4s/pauseTimers/visibleToasts=3) + shadcn 的 sonner.tsx(theme 交给令牌; 不传 closeButton) + 第八节 75 |

### 变体/尺寸与上游的映射、已知差异

对照 `shadcn/src/components/ui/*.tsx` 逐个核过一轮。**令牌名**要求与 CSS 变量逐字符对应(见 3.1),**组件变体名**是 Go API、不参与批量转译, 允许更直白, 但差异必须记在这里:

| 组件 | 映射 / 实现 | 说明 |
|---|---|---|
| Button variant | `ButtonPrimary` = shadcn 的 `default`(用 primary 色), 另有 `ButtonDefault` 别名 | 上游把这个变体叫 `default`; 其余 secondary/outline/ghost/destructive/link 同名 |
| Button size | `ButtonSizeXs/Sm/Default/Lg` = h-6/h-7/h-8/h-9(24/28/32/36); `ButtonSizeIcon/IconXs/IconSm/IconLg` = size-6/7/8/9 方形 | 与上游 8 档一一对应(像素实测吻合); 方形档 cx 传 0 即用预设边长 |
| Badge variant | default/secondary/outline/destructive/ghost/link 六档齐全 | 悬停样式恒生效(**shadcn 写作 `[a]:hover:*`, 只有包在链接里才生效**; Go 侧没有链接概念) |
| Badge 悬停 | `ghost` = hover 出 bg-muted, `link` = hover 加下划线 | 下划线手绘(无下划线标志位, 见下方"下划线") |
| Card | `WithCardSize(CardSizeSm)` = `--card-spacing` 4→3(16→12) | 标题字号上游是 `text-base` + `font-medium`, 中文下 500 回落常规 → `NewCardTitle` 用 16 常规(不是粗体); sm 卡片标题自己 `SetFontSize(14)` |
| Separator | `WithSeparatorVertical` 支持垂直 | 上游 `data-vertical:w-px` |
| Tabs | `WithTabsVertical` 支持垂直(标签堆叠、左对齐、指示条在右侧) | 上游 `group-data-vertical/tabs:*` |
| Slider | `WithSliderRange(min,max)` / `WithSliderStep(step)` | 对应 radix 的 min/max/step; 默认仍是 0~100 连续 |
| ComboBox 触发区 | `WithComboSize(ComboSizeSm)` = h-7 + `rounded-[min(--radius-md,10px)]` | 对应 select.tsx 的 `data-[size=sm]` |
| Calendar | `WithCalendarSelected/WeekStart/OnSelect` + `WithCalendarFlat`(弹层内嵌不画卡片); 尺寸 `CalWidth/CalHeight`(278×327) + `CalFooterHeight`(41), 自定义宽度时用 `CalHeightFor(width)` / `CalFooterHeightFor(width)` 配高度(或 cy 传 0 自动); 年月下拉 `WithCalendarYearRange(from,to)`; 禁用 `WithCalendarDisabled(fn)` + `SetDisabled` + helper `CalendarDisableBefore/Weekend/Dates/Any`; 页脚 `WithCalendarFooter(items...)` / `WithCalendarFooterToday` / `AddFooterItem/Button/Text` / `ClearFooter` / `FooterRect`(自定义原生控件的插槽) | react-day-picker 的 props;固定 6 行 + 等比缩放是本实现决策 |
| DatePicker / TimePicker | `WithDatePickerPlaceholder/Value/Invalid` / `WithTimePickerPlaceholder/Value/Invalid` + 运行时 `SetInvalid`; 程序化开关 `Open/Close/IsOpen`; DatePicker 另有 `Clear/SetValue` | 选项名带组件前缀(组件同在 package ui); 弹层骨架 `popup.go` 是两个组件的内部实现, 随组件一起拷贝 |
| 校验失败态(aria-invalid) | `WithButtonInvalid` / `WithInputInvalid` / `WithComboInvalid` + 运行时 `SetInvalid(bool)` | 选项名带组件前缀, 因为组件同在 `package ui`(Button/Input/ComboBox 三处语义相同); 详见语义表 |
| Card 头 | `ui.NewCardHeader(parent, space)` + `TitleStack` / `ActionStack` | 对应 `CardHeader` + `CardAction`(上游是 grid 1fr_auto); 动作区跨两行贴右上 |
| ScrollArea | **本库形态**: 原生 LayoutFrame 载体 + 主题化滚动条(10px 轨道 + 8px 圆头滑块, 与 Table/Tree/ComboBox/多行 Input 共用 `scrollbar.go` 的统一收敛); 显隐同 radix 的 hover 淡入(150ms 淡入淡出); 只做纵向(上游 ScrollBar 有 horizontal 档) | 页面滚动容器; 语义与踩坑见第八节 66 |
| Menu 条目缩进 | `MenuInset` = shadcn 的 `inset` prop(`data-inset:pl-7`, 文字左缘 28) | 用于让普通条目与带前置图标/勾选的条目文字对齐;高亮底仍是整行 |
| Toast | `NewToaster(hWnd, ...)` + `Show(ToastOptions)`(标题/描述/类型/图标覆盖/动作/关闭按钮/时长) + `Success/Error/Warning/Info` 快捷方法; `WithToastPosition/Width/Offset/Gap/Duration/MaxVisible`; 实例 `Update/Close/Handle` | sonner 的 `toast` API 的 Go 形态; 位置只做四角(上游另有 top-center/bottom-center), `richColors` 不做 |
| Pagination | **本库扩展**: shadcn 的 Pagination 是纯展示组件, 省略号要手拼; 我们做了自动折叠(第八节 53) | 见 53 |
| 不适用 | `asChild`(Slot)、`data-slot`/`data-state`/`data-variant` 钩子、compound 子组件拆分、props 透传 | React/DOM 形态, Go 侧没有对应物 |
| 令牌 | `--chart-1..5` / `--sidebar-*` 已进 `xui.Theme`(Chart1..5 / Sidebar*) | 组件库自己不用, 供你的应用取用; 机械转译新主题时一并填(运行时的主题测试有守卫) |

**下划线(Button link / Badge link)**: xcgui 没有下划线文本标志位(`TextFormatFlag_` 里没有), 只能手绘: 宽度用 `xui.TextWidth` 量, y = 行盒中心 + 0.3em + 4px(`underline-offset-4`), 1px 线(`button.go` / `separator.go`)。实测(150% DPI): "Link" 下方 1 条 43 物理像素宽的横线, 紧贴文字下沿。

---

## 五、安装与更新

```bash
xui init                 # 创建 ui/: theme.go + 基础组件 + 这份说明
xui add button dialog    # 追加组件(自动带上依赖的组件)
xui add --all            # 拷贝全部组件
xui update               # 更新已装组件(省略组件名 = ui/ 里已装的全部)
xui list                 # 列出可用组件
```

- **`update` 不覆盖你改过的文件**:逐个比对内容 —— 本地没有则创建、内容一致则跳过、被改过则警告跳过;确认要丢掉本地改动时加 `--force`。这是 copy-not-install 的前提:代码归你。
- **数据源**:默认从仓库 `main` 分支拉组件(永远最新),自动在直连 / Gitee 镜像 / 加速代理之间选路;`XUI_REGISTRY` 可指向本地目录或任意 `http(s)://` 基址(离线开发 / 锁定组件版本),显式设置时只走你写的地址、失败直接报错(不静默降级)。
- **registry.json** 是组件清单:每个组件登记 `files`(要拷的文件,含跨文件助手)与 `deps`(依赖的其它组件)。CLI 只拷这些 `.go` 文件;运行时(`github.com/twgh/xui`)与图标子包(`github.com/twgh/xui/icons`)是 Go 模块依赖,不进 `ui/`。

---

## 六、ui/ 目录与组件依赖

`xui init` 之后你的项目里是:

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
  | `scrollbar.go` | 细滚动条皮肤 + 悬停显隐 | table / tree / combobox / input(多行) / scrollarea |
  | `popup.go` | 弹层骨架(定位、阴影、进出场) | datepicker / timepicker |
  | `backdrop.go` | 弹层背景模糊(抓窗快照) | dialog |

- 组件依赖(registry 里的 `deps`):`card → label`、`dialog → button + label`、`combobox → tooltip`、`menu → tooltip`;`add` 会自动补齐,不用手抄。
- 图标不在 `ui/` 里:用 `github.com/twgh/xui/icons` 的常量(只是一条 SVG path 字符串,只把用到的打进二进制,体积增量 0)。
- 升级:`xui update` 或 `xui update button`(只更新该组件及其依赖)。

---

## 七、组件覆盖与已知限制

组件与 shadcn 的对应关系见第四节。以下差异是 **xcgui 能力边界**导致的,改组件代码也补不上:

| 差异 | 原因 |
|---|---|
| Input 的文字/光标/选区/IME 由原生 Edit 绘制 | 都在 DLL 内;只开放字体/字色/占位符色/插入符色/文字内边距。要完全控制就得自绘文字并接管光标闪烁、选区命中、IME 组合窗 |
| Toast 只支持四角堆叠, 无 `richColors` | 顶部/底部居中与"按类型给卡面上底色"未做(第八节 75) |
| 没有 `asChild` / `data-slot` / `data-state` | React/DOM 概念, Go 侧没有对应物 |
| `--chart-*` / `--sidebar-*` 令牌只提供取值 | 组件库自身不用, 供你的应用取用 |

其余与上游的细小差异(逐项语义、变体映射)已在第四节的两张表里列出。

---

## 八、xcgui 踩坑全集(基于 1.4.2 源码逐条实测;编号稳定, 只增不改)

1. **渲染模式二选一**:`app.New(true)` = Direct2D,`false` = GDI+。`GDI_*` 系列函数仅 GDI+ 模式可用,圆角抗锯齿 D2D 更好。**运行时统一按 D2D 设计**,示例程序固定 D2D。
2. **2000 回调上限**:一律 `AddEvent*` 系列注册(复用回调,无上限风险)。
3. **背景管理器累积上限 20**:组件全部走 paint 事件,完全不碰 bkmanager。
4. **主题切换必须 Redraw**:主题管理器统一触发 `XWnd_Redraw`,组件自己不管。
5. **模态窗口销毁**:Dialog API 设计为"先取值再 EndModal"。
6. **DPI(实测 DPI=144 修正)**:元素创建坐标会被缩放渲染(24 逻辑 → 36 物理),但 `XEle_GetClientRect` 返回的是**未缩放的逻辑矩形**,且绘制上下文按逻辑坐标变换。因此自绘时**不要**按 `XWnd_GetDPI` 预缩放尺寸——二次缩放会让指示器撑满客户区被上下裁剪(圆变八边形)。这也是库中存在 `*DPI` 后缀函数的原因:普通版返回逻辑坐标。
7. **Edit 自绘**:原生背景需 `EnableBkTransparent` 处理,BeautifyEdit 已验证。
8. **字体**:主题携带字体方案(微软雅黑等中文环境默认字体),font 包管理。
9. **拷贝模式的代价**:组件 bug 不会随依赖自动修复 → 用 `xui update` + "组件文件写薄"缓解。
10. **元素绘制裁剪**:绘制被裁剪在元素自身客户区内,描边沿路径居中(贴边描边外半侧被裁),外扩式焦点环完全不可见 → 焦点框画在边框内侧(填充式,见 FocusFrame),需完整可见的描边请内缩。`XDraw_SetClipRect` 放宽裁剪区**无法突破**这层元素裁剪(实测:按下位移把底边推出 3px 后仍被裁掉,壳体高度少 3px)——要做"整体位移"这类越界绘制,应给元素预留像素(见 `button.go` 的 `pressShift`)。**1px 细条元素(分隔线)不要用画线**:线宽沿路径居中,画在元素边缘会有一半落在客户区外被裁掉 —— 150% DPI 下实测只剩 1 个物理像素、且整条比 Border 令牌淡一档;正确做法是直接 `p.FillRound(&p.Rc, t.Border, 0)` 填满整条(`separator.go`)。
11. **实心按钮 hover 用透明度**:shadcn 的 `hover:bg-primary/80` 语义用 `WithAlpha` 实现,比 lighten/darken 在近黑/近白主题色上有可感知变化,且明暗主题行为一致。
12. **cy=0 的控件高度就是 0**:xcgui 不会按内容自适应,高度为 0 的控件直接不渲染(探针实证)→ 组件构造函数对 cy<=0 一律补默认高度。
13. **窗口背景必须用 Leave 状态注册**:xcgui 的 `XBkM_AddFill` 按"当前组合状态"选填充,窗口绘制时处于 `Window_State_Flag_Leave`,用 `Element_State_Flag_Nothing(0)` 注册的填充永远不被选中;且需先 `XBkM_Clear` 清默认填充(paint 事件 / BkManagerEx / SetBkInfo XML 均无法改变窗口背景,探针实证)。已封装为 `xui.BindWindowBackground`。
14. **标题文字默认白色**:xcgui 标题栏文字颜色为白色(为默认蓝色标题栏设计),主题化窗口(白/黑底)需 `w.SetTitleColor(theme.Foreground)` 并挂 OnThemeChange。
15. **窗口样式按标志位组合**:去掉系统图标和三键 = `Caption|Border|Center|Drag_Border|Allow_MaxWindow|Title`(不含 Icon/Btn_*);去掉关闭按钮后需自补 ESC 关闭(`XWnd_CloseWindow`)。
16. **focus-visible 需自行跟踪焦点来源**:xcgui 只有单一 Focus 状态,不区分鼠标/键盘来源 → 全局跟踪鼠标按下/Tab 键(运行时 focus.go),文本输入类控件例外(始终显示,同浏览器语义)。
17. **元素 paint 回调里能画位图、也能画 SVG;图标走专用管线**(复核修正 —— 此前本条曾记成"paint 画不出图片", 有误导):
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
24. **异型透明窗口可以做到真透明,且背景对象支持圆角**(旧结论"背景管理器没有圆角矩形对象,开了 Window_Transparent_Shaped 整窗什么都不画"是错的,已由探针推翻):
    - 形状来源:`Window_Transparent_Shaped` + `SetTransparentAlpha(255)` 后,窗口形状 = **自身绘制内容中不透明像素的并集**,并不是只认背景对象。因此**仅靠 paint 自绘**就能拿到带真实 alpha 边缘的形状——气泡圆角外、箭头两侧完全透明,压在其它应用画面上也不露方块。
    - **前提**:不能给窗口再挂背景填充(如 `xui.BindWindowBackground`),那会把整窗铺成不透明矩形,透明随即失效。Tooltip 原来的"四周同色方块"就是这个原因,不是形状机制做不到。
    - 对照实测(探针,窗口矩形内与基线逐像素比较):仅 paint 自绘 + 异型 → 88% 像素变化,气泡下方一行 0 像素变化(真透明);整窗铺主题底色 → 99% 像素变化,气泡下方 453/480 像素变化(不透明)。同一份 paint 代码,只差是否挂窗口填充。
    - 想用背景对象做圆角也行:`bkm.AddFill(state, color, id)` 传入**非 0 的 id**,再 `bkm.GetObjectObj(id).SetRectRoundAngle(左上,左下,右上,右下)`(配 `SetMargin` 定位置尺寸)即可得到圆角矩形;该对象还支持 `SetRotate`、`SetBorderColor` 等更多美化。官方示例 `Advanced/WindowPartiallyTransparent`(窗口级 bk 管理器)与 `webview/WebViewPartiallyTransparent`(布局元素级 bk 管理器)都用了这一手法。
    - 生命周期无坑:`Window_Style_Nothing` 的窗口按"先建 10x10 → 首次显示前改尺寸/位置 → 复用再改尺寸"的懒创建流程走,异型透明一切正常(探针实证)。
    - ⚠️ 附带发现:`Window_Style_Nothing` / `Window_Style_Center` 的窗口**自身不绘制任何东西**(只画子元素),拿它当普通宿主窗口会"看不见窗口";要正常显示请用 `Window_Style_Default|Drag_Window`。
25. **别跟原生 SliderBar 的滑块按钮(原生 Button)纠缠, 要么全套压制要么完全弃用**(实测两轮踩坑):它的默认样式通过 **BkInfo** 给 Stay/Down/Focus 态注册了填充+边框——悬浮冒灰色方块、拖动后冒白色方块、点击聚焦后又冒一种方块, 形态还各不相同。`ClearBkInfo`(实测 bkinfo=0 仍漏)+ `EnableBkTransparent` + `EnableDrawBorder/DrawFocus(false)` + 空 paint 回调(`pbHandled=true`) 全部叠加也压不住 Focus 态;而 `EnableFocus(false)` 虽然能断掉焦点方块, 却会让按钮**不再上报 Stay 状态**(悬浮检测失效, hover 环画不出来)。结论:Slider 最终**弃用原生 SliderBar 载体**, 改用普通 `widget.NewElement` + 自实现交互(LButtonDown 里 `XEle_SetCapture(true)` 捕获鼠标, MouseMove 持续更新, LButtonUp 释放; MouseStay/MouseLeave 自跟踪 hover; KeyDown 左右 ±5%/Home/End), 绘制 100% 自绘后一切干净。**教训:当原生控件的残留绘制出现在多个互不相干的状态路径上时, 逐个压制是无底洞, 直接换纯自绘。**
26. **Slider 几何要对齐 radix**:滑块**中心**在轨道两端之间移动(滑块会向轨道端点外伸出半个身位), hover/active/focus-visible 的 ring-3 再向外扩 3px;而元素自绘被裁剪在自身客户区内(见第 10 条)。因此:元素左右各比**可见轨道**宽出 9px(滑块半径 6 + ring 3), 轨道画在内缩 9px 处, **入参 cx 仍是可见轨道宽度**;滑块中心由 pos 自算, 两端 ring 刚好落在元素边缘, 完整可见。
27. **`XDraw_FillRoundRect`/`XDraw_DrawRoundRect` 的 nWidth/nHeight 就是圆角半径**(曾误判为"直径"并把 Painter 里 x2, 导致按钮/开关/下拉框全体过圆成胶囊形, 已回退):判断依据是对照实验——按钮等组件按半径直传时视觉与 shadcn 一致,x2 后过圆。`xui.Painter.FillRound/StrokeRound` 直传半径即可, 不要再自行换算。(Slider 上最初看到的"方块光环"其实是原生按钮 BkInfo 残留, 见第 25 条, 与 FillRound 无关。)
28. **不整体填充的透明元素必须在 onPaint 开头先铺宿主底色, 否则上一帧像素残留**(实测踩坑):Slider 的轨道/滑块/ring 都是局部形状, 元素背景透明, 状态变化触发重绘时框架**不会擦除旧像素**——hover 离开/点击聚焦后灰环一直留在画布上(`Redraw(true)` 立即重绘也无效, 因为重绘本身不擦底)。修法:onPaint 第一步 `p.FillRound(&p.Rc, t.Card, 0)` 铺满宿主底色(组件放在卡片上用 Card; 放在窗口背景上需对应调整)。Switch/Button 不留残影正是因为它们整体填充了自己的矩形。
29. **LayoutEle 布局语义与文档直觉不符, 全部经探针实测**(探针结论,示例程序布局化依赖):LayoutEle 实为"横向 flex 流 + 自动换行"盒子——
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
      并在高度传 `GetHeight`"就是抄元素尺寸, 抄错即出 bug: 标题栏按钮曾矮 1px)。实测数据见上表:
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
    - **进场淡入**: 弹层是**普通窗口**, `SetTransparentAlpha` 对它无效(探针实测: 不换类型设 128, 抓图仍全白)。`XE_COMBOBOX_POPUP_LIST` 回调里 `XWnd_SetTransparentType(hWindow, Window_Transparent_Shaped)` + 把 alpha 压到下限即可 —— 该回调触发时窗口**还没显示**(`IsWindowVisible=false`), 首帧不会闪白。与 57 的 tooltip 异型窗不同, **这个弹层上连续改 alpha 是生效的**(`shaped-seq` 探针依次设 64/160/255, 三帧抓图 #40A0FF / #A0D0FF / #FFFFFF), 所以直接用整窗 alpha 播 100ms 淡入, 不需要(也无法)走元素 alpha —— 内容全是原生 ListBox。过渡挂**宿主窗口**驱动、apply 每帧 `XC_IsHXCGUI` 守卫(弹层可能被立刻关闭销毁), 细节见 69;
    - **描边不能画在 ListBox 的 paint 事件——该事件在这个弹层窗口上不渲染输出**(探针实证: 内缩 10px 的 6px 红色粗环都画不出来); 这个窗口上能出图的只有 BkM 背景对象、条目 `DrawItem` 回调和 `XE_PAINT_END`;
    - **BkM 的 border 对象贴边画, 会被 SetWindowRgn 的窗口区域裁得只剩左边一条**(上/右/底全消失, 只留左缘; 区域与内容为何错位未深究)。ring-1 描边的出路: 先 `XEle_EnableEvent_XE_PAINT_END` 再 `AddEvent_Paint_End`(该元素及子元素绘制完成事件, 在条目之后执行 = 边框盖内容边缘, 同 shadcn), 描边整体**内缩 1px** 落在窗口区域内, 四边完整, 圆角弹层上 1px 白边距不可感知;
    - **条目高亮/勾用 `AddEvent_ListBox_DrawItem`**(回调带 Index/NState/RcItem): 原生在弹层打开时**不给选中项置 Select 态**, 要自己拿 `XComboBox_GetSelItem` 比对求"选中项"(只用来决定右侧 check), 但**底色只画给"指针所在项"**(`NState == Stay || Select`)——shadcn 的 item 只有 `focus:bg-accent`, 选中态由右侧 check 表达, 不带常驻底色; **指针不在任何条目上时(弹层留白、弹层上下左右边缘外)没有任何底色**, 对齐 radix: 条目的 pointerleave 把焦点交回 content(`onItemLeave = content.focus`), 高亮随之消失。两个被实测否掉的版本: ① "选中项永远高亮"(鼠标移到别项后选中项仍带灰底); ② "指针离开条目后高亮留在最后悬停项"(radix 的 `highlightedItem` 在 item 失焦后并不保留, 用 `XListBox_GetItemMouseStay` 闩住最后悬停索引会画出一个不该有的灰底)。灰底的像素验收: 指针在项上时该行背景均值 ≈ 241-246, 一离开条目(移到弹层外)同一批行(弹层 y 436..458)均值回到 249-255(= popover 白底 + 文字旁 JPEG 振铃), 即灰底消失;文字是项模板的 shapeText 子元素, 在项背景之后绘制不受影响;
    - **`SetDropHeight` 必须在弹层打开前设置才生效**(构造时), 弹出事件里设置无效(实测 set/get 均为新值但窗口不变);固定模式(`EnableDropHeightFixed(true)`)才使用该值;
    - **触发区滚轮**: 原生会滚动 ComboBox 内部文本(文字上移), `EnableEvent_XE_MOUSEWHEEL(false)` 屏蔽(事件交给父元素), 弹层 ListBox 独立不受影响;
    - **弹层退出后窗口句柄即失效**: 关闭后用旧句柄调任何 XWnd_* 会弹"句柄无效"错误框(实测 `XWnd_GetDPI`), 必须挂 `AddEvent_Combobox_Exit_List` 置空缓存句柄;
    - **弹层窗口/ListBox 每次弹出都是新建的**(退出即销毁, Exit_List 后窗口句柄即失效), 事件在每次 Popup_List 里重新注册即可, 不会重复;全局的 OnThemeChange 回调才需要 once 守卫。
    - **滚动条残影**: 原生的滚动条显隐判定在打开时用旧条目高算好, 调矮条目后不会自己收起; 用滚动视级开关 `ShowSBarV/ShowSBarH` 显式控制(条目级 `ScrollBar.Show` 会被原生布局覆盖)。弹层高度按 `comboDropHeight` 取**内容高**(见本节 52), 不再需要早期"内容高 +4 余量"的兜底。
    - **弹层默认定位与 item-aligned 重定位**(探针实测): 原生把弹层放在触发区**正下方**(顶=触发区底、左=触发区左、宽=触发区宽)。radix Select 的默认定位是 item-aligned(shadcn select.tsx 的 SelectContent), 即**弹层盖住触发元素、选中项正对触发位置**——用 `wapi.SetWindowPos`(物理像素, `SWP_NOSIZE|NOZORDER|NOACTIVATE`)在 Popup_List 事件里重定位: 弹层顶 = 触发区顶 − (选中项在内容里的顶 − (触发区高−条目高)/2), 其中选中项在内容里的顶 = p-1 边距 4 + ListBox 固有 3px 内缩 + 选中索引×(条目高+1)= 7+29i。**两种盒子高度不同(触发区 32、条目 28), 必须按"文字中心"对齐**(选中项顶 = 触发区顶 + 2), 早期用"p-1 + 选中索引×条目高"少算 5px 且按底边对齐, 实测选中项文字比触发区文字低 3 物理像素(点开瞬间字"往下跳"), 修正后 1 物理像素内重合; **横向也必须按"文字左缘"对齐**(radix 源码 `SelectItemAlignedPosition`: `left = 触发区文字 left − (条目文字 left − 内容 left)`, 对齐的是文字而不是盒子): 弹层左缘 = 触发区左缘, 而条目文字相对弹层左缘的内缩 = p-1(4)+ListBox 固有 2+条目文字左边距, 该边距按 shadcn 的 px-2=8 时实测条目文字比触发区文字**右移 2px**(触发区文字 813..862 / 条目文字 815..862), 点开瞬间看着像"文字往内收缩"; 边距改 5 后两者墨迹起点完全重合(同为 x 813, 条目 813..859)。裸宽度仍差 ~3px(50 vs 47)——触发区文字走原生控件(Direct2D)、条目文字走 shapeText 两条不同渲染路径, 同为 4 个全角字时字形推进取整不同, 非位置问题, 视觉不可感知。注意别用"整体左移弹层窗口"来对齐: 弹层宽度 = 触发区宽度, 窗口左移会让右侧 check 与弹层右缘的距离一起变; 贴屏幕边时向下压并用 `ScrollView.ScrollPosYV` 滚动让选中项仍可见。坐标链路: `XEle_GetWndClientRect`(逻辑, **原点=所属窗口 rect 原点, 含阴影边距**——实测含阴影窗口也成立) + 所属窗口 `XWidget_GetHWINDOW` → `wapi.GetWindowRect`(物理) → 逻辑差 ×DPI/96;
31. **xcgui 字体的 size 单位是 pt(磅), 不是像素**(重大全局坑, 实测确认): `font.NewEX(name, size, style)` 文档写明"size: 字体大小, 单位(pt, 磅)", 渲染尺寸 = size×DPI/72 物理像素。此前 FontHandle 把主题的 14(逻辑 px) 直接当 pt 传入, 144 DPI 下渲染出 28 物理像素 em = **18.7 逻辑像素**, 全组件文字放大了 1.33×(墨迹实测 26-27 物理像素, 与 14pt 完全吻合)。修法: **pt = px×72/96(= ×0.75, 与 DPI 无关)**, 在 FontHandle 里统一换算(14px→11pt=14.67px, pt 取整导致 14 这类 ×0.75=10.5 的值有 ±0.67px 误差; 12/13/16px 恰好或接近取整)。排查入口: 同一截图里对比组件文字与已知 14px 参考的墨迹高度。
32. 用 pnpm 添加 shadcn 组件会报错, 应该用npx。
33. **Dialog 用"窗口内覆盖元素"实现**(探针 + 示例程序实测, dialog.go 依赖):
    - **窗口子元素坐标原点 = 窗口左上角(含标题栏), 不是客户区原点**:`XWnd_GetClientRect` 返回的 `{8,8,624,412}` 只是窗口边框内缩(与标题栏无关), 用它给元素定位会整体偏移;遮罩尺寸直接取 `XWnd_GetRect` 的宽高放 (0,0) 即可铺满整窗(实测遮罩把标题栏也压暗, 窗口自绘的标题文字仍在上层保持清晰);
    - **自绘 Element 覆盖全窗天然是模态**:它挡住下层所有鼠标事件(实测覆盖后下层按钮既收不到点击也收不到 hover), 配合 `XEle_SetZOrderEx(h, 0, xcc.Zorder_Top)` 保证在最顶层(每次 Show 都抬一次, 因为示例程序里元素创建顺序可能比对话框晚);
    - **半透明合成精确实测**:遮罩填 `RGBA(0,0,0,26)`(bg-black/10)后, 白底 255→229、深色钮 14→59(悬浮态), 与 0.9 系数完全吻合, 无累积变暗;
    - **隐藏遮罩必须整窗重绘**:`XWnd_Redraw(hWnd, true)`;局部重绘会留下变暗残影(半透明像素已经"烙"进帧缓冲);
    - **元素收不到键盘事件, 键盘统一走窗口 `AddEvent_KeyDown`**:探针里遮罩元素注册的 KeyDown 从不触发;窗口级回调在"编辑框聚焦"时也照常收到 ESC(27)/Tab(9) → 对话框的 ESC 关闭与 Tab 循环都在窗口回调里做, 用 `wapi.GetAsyncKeyState(VK_SHIFT)<0` 判 Shift+Tab;
    - **打开时先 Show 再 layout**:内容高度量的是"可见子元素"的占位(`XWidget_IsShow` 过滤), 隐藏状态下量出来是 0(示例程序第一次跑面板只有标题+页脚就是这个原因);控件自建后默认可见, 所以 NewDialog 末尾要用递归 `XWidget_Show(false)` 隐藏整棵子树(父元素隐藏不会连带隐藏子元素);
    - **`XWnd_SetFocusEle` 搬焦点后不会重绘**:按钮上的焦点框不出现(输入框因为要重画光标/IME 才碰巧刷新), 必须手动 `XEle_Redraw(旧)` + `XEle_Redraw(新)`;且顺序要在搬焦点**之前**调 `xui.NoteKeyboard`——AttachWindow 的 Tab 钩子注册得晚(回调后执行), 等它标记"键盘焦点"时元素早已重绘完;
    - **焦点反查要沿父链**:`XWnd_GetFocusEle` 拿到的是复合控件(Edit)的内部元素, 与创建时的句柄不同 → 用 `XWidget_GetParentEle` 往上找焦点序列里的成员, 找不到再用自维护的下标兜底;
    - **Tab 序列只收交互控件**:`XObj_GetType` 白名单(Button/Edit/ComboBox/SliderBar/Editor/RichEdit/ListBox/TabBar)+ `IsEnableFocus && IsSwitchFocus`;Label 载体是原生 Button, 必须显式 `EnableSwitchFocus(false)` 才不会挤进 Tab 序列(顺带 `EnableDrawBorder(false)`, 否则它能画出原生边框);
    - **backdrop-blur-xs 后来补上了**(见第八节 47 的 backdrop-blur 小节): 当时尚以为画不出, 实际走"抓窗快照 + BkM 图片填充"能精确实现。
    - **ring-1 要画在面板矩形外侧**:tailwind 的 ring 是 box-shadow, 画在盒子**外面**压着遮罩灰底, 才形成可见的深色描边;xcgui 圆角矩形描边落在路径**内侧**, 直接描面板矩形会把 10% 黑压在面板白底边上(230), 与遮罩灰(229)几乎同值而完全不可见(像素实测)。修法 = 描边矩形外扩 1px(dialog.go onPaint)。

34. **Dialog/文字/量测的一批实测结论**(视觉对齐期间, 探针+像素测量):
    - **自绘文字的字号就是逻辑像素, 不要按 DPI 反除**: 曾据"按钮文字顶到边缘"推断绘制路径二次乘 DPI 并做了字号反除, 实测反证(反除后全局文字明显偏小), 已回退并记录于此。真正的原因是量测口径;
    - **`app.GetTextShowRect` 在 144 DPI 下返回值约为实际绘制宽度的一半**: 对话框页脚用 "保存"(2 字)量出 14.7 逻辑宽, 实际绘制 29.3 → 按它算按钮宽会让文字顶到按钮边。现在量宽处 ×2 校正(改量测 API 或换 API 亦可, 但要重新验证);
    - **量宽与绘制必须同字体同限宽**: Tooltip 曾用 14px 主题字体量宽、按 12px(`text-xs`)绘制, 气泡两侧留白偏宽 17%;
    - **句柄有两级, 混用会弹模态错误框**: `XWidget_GetHWND(元素)` → Win32 HWND; `XWnd_*` 系列要的是 xcgui 窗口句柄 hWindow(用 `XC_hWindowFromHWnd` 转)。窗口句柄喂 `XWidget_GetHWND`、HWND 直接喂 `XWnd_GetDPI` 都会弹"输入句柄可能无效"错误框, 且**绘制回调里调这类校验型 API 风险高**(实测复现两次);
    - **容器元素被点击会聚焦并画原生框**: 普通 Element/内容容器点一下就获得焦点, 原生焦点框是**蓝色**(#58B1FC), 而它的**边框**在聚焦态是深色 —— 两个都要灭: `EnableFocus(false)` + `EnableDrawBorder(false)`。蓝框还曾压住输入框左右描边, 造成"输入框边框缺失/圆角怪"的错觉;
    - **元素内文字被元素矩形裁剪**: 按 CSS `leading-none` 把行盒压到字号高度时, 中文行高(≈1.32em)放不下, 字被切头 —— 文本元素要上下各外扩几个像素(布局高度仍按行盒算), 见 dialog.go `dlgTextOver`;
    - **页脚 `-mb-4` 的高度账**: 页脚贴面板底边时它自身高度要**全量**计入面板高(此前少算 16 逻辑像素, 面板整体矮一截);
    - **中文下 `font-medium`(500) ≈ 常规字重**: 微软雅黑只有 Regular/Bold 两档, 浏览器 500 会回落到 Regular。用粗体字形实现 500 会明显偏粗(实测两轮翻工)。标签看着比描述粗, 主因是**颜色**(`--foreground` 近黑 vs `--muted-foreground` 灰), 实测两者笔画宽度相同;
    - **Tooltip 定位不能叠加客户区偏移**: `XEle_GetWndClientRect` 返回的坐标原点已是"窗口 rect 原点(含阴影边距)", 再叠加 `XWnd_GetClientRect` 的内缩会让气泡整体下移/右移一个阴影宽度(实测 8 逻辑像素, 表现为箭头尖端扎进目标元素)。

35. **暗色下的"半透明令牌 + /NN 修饰"必须做 alpha 乘法**(暗色验收, tabs/switch/radio 三处同源):
    - 暗色主题里 `input` 等令牌**自带 alpha**(`oklch(1 0 0 / 15%)`, 内置暗色主题写作 `RGBA(255,255,255,38)`); 此时 CSS 的 `bg-input/30` 语义是 15% × 30% = **4.5%**, 而 `disabled:opacity-50` 是 15% × 50% = 7.5%;
    - 我们原先用 `WithAlpha(v)` 直接**覆盖**了这层 alpha, 于是: tabs 选中态比参考亮一倍(#686868 vs #2f2f2f)、未选开关的轨道变成不透明白饼、禁用开关反而比正常态更亮(50% > 15%);
    - 修法: 新增 `Color.ScaleAlpha(frac)`(在现有 alpha 上乘), 全库凡是"半透明令牌再叠透明度"的地方都用它, 不要用 `WithAlpha`;
    - 相关规则(逐条对源码): tabs 暗色选中 = `bg-input/30` + `border-input`(**不是** `bg-background`, 否则暗色下比列表更黑); radio 未选 = `border-input` + 暗色额外 `bg-input/30` 淡填充, 禁用 = 指示器整体半透明(标签文字不跟随) —— 原先禁用画成 `Background@50%`, 暗色下就是一块黑盘; switch 未选轨道 `bg-input`/暗色 `bg-input/80`, 选中滑块 `primary-foreground`、未选 `foreground`。

36. **原生菜单皮肤的一批实测结论**(ui/menu.go;  按第八节 64 升级为阴影弹窗):
    - 弹窗圆角+阴影: **已改走 `Window_Transparent_Shadow` 路线**(官方 DrawMenu 示例同款, 详见 64) —— `Menu_PopupWnd` 事件里 `SetTransparentType(Shadow)` + `SetShadowInfo(nSize, depth, RadiusLg, false, 黑/128)` + `SetRect` 四周外扩 nSize; 圆角由阴影窗按逐像素 alpha 画出, 不再用 `wnd.SetWindowRound` 区域裁剪; shaped 透明对菜单弹窗无效, **Shadow 有效**(整窗 alpha 也可用);
    - 项几何(`SetBorderSize(4,4,4,4)` 后探针实测): `Menu_DrawItem` 的 `RcItem` 四周各留 **4px**(= shadcn p-1; 默认边距是 3, 差的那 1px 就是"高亮距弹层左右边缘与参考不一致"的根因), 高度 = `SetItemHeight`(28); **分隔栏槽高固定 4px**, 1px 线画在 `Top+2`、左右越过边距铺满弹层(shadcn `-mx-1`);
    - `"	"` 快捷键: 原生把 `	` 后的文本算进自适应宽度并预留 `NShortcutKeyWidth`(实测 "Ctrl+C"=37) → 把 "名字	快捷键" 整串传给 AddItem, 自绘时拆开分别渲染就得到右对齐快捷键, 不用自己量宽;
    - `SetLeftWidth(0)` 去掉左侧图标条; Label 项借 `Menu_Item_Flag_Disable` 实现"不可高亮不可点"(自绘时对 kind==label 单独配色);
    - **`XMenu_SetItemWidth` 逐项设置条目文本区宽度, 有效**(官方 DrawMenu 示例同款用法: AddItem 后对单个 nID 调用, 返回 true 表示受理; 探针实测: 3 条短条目弹窗宽 104, 全部条目 `SetItemWidth(300)` → 348, 只设其中一条 → 同样 348, 全部设 60 → 108): 弹窗宽 = **max(各条目有效宽) + 固定开销**(未设条目按自然文本宽计入; 开销 = 边框 + 文本间隔等, 条目高 28 / BorderSize(4) / LeftWidth(0) 配置下实测 48); **设定值完全覆盖自然宽** —— 自然文本宽 229 的长条目设 60 后弹窗仍 108, 长条目不会撑宽弹层。shadcn 的"弹层宽 = 触发区宽 + min-w-32 下限"(dropdown-menu.tsx `w-(--radix-dropdown-menu-trigger-width) min-w-32`)由此实现: 逐条目设 `max(触发区宽, 128) − 开销`(menu.go 已接入) —— 长文本由自绘 `XDraw_SetClipRect` 硬裁剪, 与 radix 的 overflow-x-hidden 一致(原生条目文本同样裁在弹层边缘, 截图验证); 显式覆盖 = `Menu.Width(w)`(对应 shadcn 在 Content 上加 `className="w-56"` 这类宽度类, demo 参考卡就是 w-56=224, 实测精确命中); 右键菜单(PopupAtCursor)无触发区, 退化为"内容自适应 + 128 下限"; 子菜单条目不设宽(shadcn SubContent 是内容自适应 + min-w-96, 不跟触发区);
    - 事件拓扑: 窗口级 `XWM_MENU_DRAW_BACKGROUND/DRAWITEM` 对该窗口弹出的**所有**菜单生效(每次新建的菜单对象、各自的子菜单窗口都触发) → 皮肤每窗口只注册一次(map 守卫), 绘制实时取令牌, 换肤零成本; `Menu_PopupWnd` 对每个弹出窗口(含子菜单)各触发一次且顶级先于子菜单 → 阴影处理放这里全命中;
    - 弹窗定位: `Popup` 收**物理像素**; 元素锚点 = `XEle_GetWndClientRect`(逻辑, 原点=窗口矩形含阴影边距) + `wapi.GetWindowRect`(物理) 换算; **翻转/夹紧/子菜单定位全部原生自带, 任何手动位置校正都不做** —— 详见下一条;
    - ⚠️ **验收坑: 全屏缩略截图里暗色弹层窗口会显示成亮底**(截图管线对弹层窗口的合成/缩放问题), 区域放大看才是真实效果 —— 弹层类组件的暗色验收一律以 zoom/像素测量为准, 不要凭全屏缩略图下结论。
    - ⚠️ **别给弹出菜单手动校正位置**(探针实测, 踩了两轮): `XMenu_Popup` 是**阻塞**的(菜单关闭后才返回), 且 `Menu_PopupWnd` 事件对每个弹窗触发一次。两条死路: ① 在 PopupWnd 回调里 `SetWindowPos` 移动顶级弹窗 → 子菜单锚点失效, 二级菜单弹到父菜单**原点**盖住父菜单; ② 等 `Popup` 返回后再校正 → 弹窗已销毁, `XWnd_GetHWND` 报「输入句柄可能无效」。结论: 初始锚点给对(元素矩形换算), 其余(顶部放不下翻到上方、左右夹紧、子菜单贴父菜单右侧行对齐)原生全包, **不要动**。(补充: PopupWnd 里的 `SetRect` 四周对称外扩**不影响**子菜单锚点, 阴影路线依赖它; 但单独的 `SetWindowPos` 位移会直接崩进程, 见 64。)

64. **菜单弹窗转阴影窗 + 进场淡入的一批实测结论**(ui/menu.go,示例程序+探针双向验证):
    - **阴影弹窗 = 官方 DrawMenu 示例同款三步**: `SetTransparentType(Window_Transparent_Shadow)` → `SetShadowInfo(nSize, nDepth, nAngeleSize=RadiusLg, false, 黑/128)` → `SetRect` 窗口矩形四周外扩 nSize(逻辑像素, 客户区会收缩出同尺寸阴影带, 必须外扩补偿, Body 才回到原生给定的菜单位置)。nAngeleSize 一次把圆角(逐像素 alpha, 无区域裁剪锯齿)和投影都做了。**参数对齐 shadcn shadow-md**(0 4px 6px -1px black/10%): nSize=5(官方示例的 8 比shadow-md 大一圈)、nDepth=50(阴影引擎边缘浓度≈depth/1000, 50 时白底上边缘约 5%, 与 shadow-md 侧缘观感一致; DLL 阴影是均匀环带, 做不出 shadow-md 上 2px/下 10px 的不对称);
    - **子菜单锚点不受外扩影响**(探针实测: 子菜单仍贴父条目右侧、行对齐); 子菜单窗口走同样的 PopupWnd 事件, 阴影自动全命中;
    - **整窗 alpha 淡入可用**: 阴影窗是透明类型, `XWnd_SetTransparentAlpha` 有效 —— 第八节 47 条 5 "native 弹层做不了淡入"的结论只适用于**非透明类型**弹窗(menu/combobox 的下拉窗默认不是 WS_EX_LAYERED, `GetLayeredWindowAttributes` 直接失败), **转 Shadow 后可淡入**; 依然遵守"alpha 永远不能精确 = 0"(47), 进场前压到 `xui.AlphaFloor`;
    - ⚠️ **过渡闭包绝对不能捕获事件结构体指针**: `Menu_PopupWnd_` 是原生栈对象, 回调返回后被复用/清空 —— Presence 的 apply 帧里再解引用, 读到 0(淡入永远不动, 弹窗 alpha 卡在下限"隐形")或垃圾值(拿无效句柄调 X API → **整个进程静默崩溃**,示例程序实测崩了两轮)。必须先把 `hw := pInfo.HWindow` **拷贝值**再进闭包;
    - ⚠️ **apply 每帧要 `XC_IsHXCGUI(hw, XC_WINDOW)` 守卫**: 菜单可能在过渡途中被关闭(弹窗随之销毁), 官方 DrawMenu 示例在 KillFocus 里同款用法;
    - ⚠️ **对菜单弹窗做 `SetWindowPos` 位移会直接崩进程**(`SWP_NOSIZE|NOZORDER|NOACTIVATE` 也不行, 进场那一瞬移 8px 就崩,示例程序+探针双向验证; `SetRect` 对称外扩没事) → radix 的 slide-in-from-top-2 **不可实现**, 进场只做 fade-in(100ms, 与 radix fade-in-0 观感一致);
    - 像素验收(144 DPI): 条目高亮距弹层左右边缘各 6 物理像素 = 4 逻辑(= p-1, 左右对称); 分隔栏全宽;体色暗色 (23,23,23); 阴影带梯度柔和无黑边;
    - **菜单弹窗的绘制坐标系**(色块标定实验实测): `Menu_DrawBackground/DrawItem` 的绘制上下文**原点 = 弹窗窗口左上角(逻辑像素)**; `XWnd_GetClientRect` 返回的 {8,8,168,276} 就是阴影窗 Body(可见白区)在窗口内的矩形(= 窗口矩形 ± 阴影带) —— fill/ring 都画在它上面即可; **画进 Body 之外(阴影带里)的任何像素都会被阴影 alpha 蒙版压暗至不可见**(opaque 色块画进带里直接"消失"); ring 画在 Body 矩形边界上(线宽外半被蒙版裁掉), 肉眼就是一条贴着窗口边缘的线, 圆角用 RadiusLg 与 Body 一致, 四角才不脱开 —— ring 若内缩或半径不匹配, 视觉上就是"矩形线条浮在弹层里不贴边"。
    - **子菜单悬停延迟 ~330ms 是 xcgui DLL 固化的**(纯净菜单探针实测同值 324ms): 与系统 MenuShowDelay 无关(SPI 置 0 复测仍 343ms), 无配置 API, VK_RIGHT / SendInput 真实点击 / PostMessage 鼠标消息都无法提前触发。xui 侧唯一能省的是自己的 100ms 进场淡入 —— 子菜单(NParentID != 0)直接全 alpha 出现, 不做动画。要达到 shadcn 的即时展开只能改 DLL 或等作者提供 API。

37. **页面容器 Show(false)→Show(true) 后内容空白**(示例程序顶层 Tabs 分页):
    - 现象: 隐藏期间创建好的布局子树(LayoutEle 流式布局)在页面恢复显示后只有壳在渲染, 内部控件全空白;
    - 根因: xcgui 布局元素在父容器不可见期间不排版(与第八节 29 "布局项变更需手动 AdjustLayout" 同源), Show(true) 只改显隐属性, 不会触发子树重排;
    - 修法: 切页时对**新显示的页面容器**调 `AdjustLayout`, 再 `Redraw`; 初始就可见的页面不需要。

38. **自绘标题栏的无框窗口组合与鼠标穿透**(demo/main.go):
    - 不要 Default(自带图标/标题/三键): 样式用 `Caption|Border|Drag_Window`(不含 Icon/Title/Btn_*/Allow_MaxWindow/Drag_Border), `SetBorderSize(0,30,0,0)` 保留 30px 标题栏高度, 阴影照旧 `Window_Transparent_Shadow + SetShadowInfo`; BkM 窗口背景绑定在此组合下正常工作(Window_Style_Nothing 才是"自身不绘制任何东西"的那个);
    - 图标/标题等**纯展示元素必须 `EnableMouseThrough(true)`**, 否则元素挡住 Drag_Window/Caption 的拖动命中(实测标题文字区域拖不动窗口, 穿透后整条标题栏可拖); 子元素不受穿透影响;
    - 控制按钮用普通自绘按钮(幽灵变体 + 图标)即可, 最小化 `w.ShowWindow(xcc.SW_MINIMIZE)`(窗口包没有 Minimize 包装), 关闭 `w.CloseWindow`; 不需要 CaptionButton 示例那套"读原生最小化按钮坐标 + Paint_End 里重定位"的逻辑(那是给 Default 样式加按钮用的)。

39. **窗口布局(EnableLayout) + 阴影窗口 + 自绘标题栏的一批实测结论**(示例程序布局化重构):
    - **客户区已避开阴影带**: rect 800x590、阴影 8px 时 `XWnd_GetClientRect` = {8,8,792,582}(784x574), 子元素坐标就是可见区坐标, **不需要再设 padding 避让**; 但窗口 rect 比可见区大一圈, 按 rect 宽算坐标会超出可见区(实测按钮右缘超 12px);
    - **布局区 = 客户区 − border size − padding**: border 归零后布局区=整个客户区;
    - **布局项必须是 LayoutEle**: 普通 Element 设置 `LayoutItem_*` 也不被排版(探针实证 red 块不占位、尺寸保持 0 → 连带裁掉全部子元素, 表现为整块空白);
    - **主轴剩余空间用 Weight 不是 Fill**: 纵向布局里 Fill 高度 = 独占整列(与前面 Fixed 项重叠, 实测 content 574=全高盖住 cap 30); 横向行里 Fill 宽度会把后面的 Fixed 兄弟挤到下一行(按钮消失)。DESIGN 第八节 29 的"Fill 独占一行"就是指主轴; 交叉轴用 Fill 铺满没问题;
    - **绝对定位容器要 `EnableLayout(false)`**: LayoutEle 默认排版开着, 内容区里绝对定位的 tabs/pages 会被当布局项排走(实测 tabs 跑到窗口底部);
    - **双击最大化自己补**: border 归零后原生 caption 区域没了, 窗口级 `AddEvent_LButtonDBClick` 里 `IsMaxWindow ? ShowWindow(SW_RESTORE) : MaxWindow`(MaxWindow 只管最大化, 不会切换); 最大化/还原时窗口布局自动跟随, 按钮组(Weight 顶右)始终贴右上角。**双击命中带的 y 阈值 = `captionH + shadowSize`(最大化时不计阴影)**: 子元素坐标原点是含阴影带的窗口左上角, 阴影大小抽成 `demo/main.go` 的 `const shadowSize = 12`(别把 12/30 裸数字散在阈值里); 已用真实鼠标逐带验证(标题栏上半段/下半段都最大化、标题栏之下不最大化);
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

41. **LayoutEle 纵向(列)布局也会"溢出换列"**(示例程序数据页踩坑): 列布局里主轴上剩余高度放不下下一个项时, 该项**不会被裁剪也不会重叠, 而是被甩到"下一列"** —— `x += 容器宽度, y = 0`, 用户看到的就是"控件整块消失"(与第八节 29 的"横向 Fill 独占整行把兄弟挤到下一行"是同一套换行机制的两个方向)。修法: 布局级 `EnableAutoWrap(false)`; 同时要保证容器高度够(本例 card 高 320 才能容下 24+14+24+14+180=256 的列)。`AdjustLayout` 之前读到的 `XEle_GetRect` 是旧值(排版前), 排查这类问题要在排版之后取坐标。列容器现在都走 `xui.VStackWeight` / `card.ContentStack`(`EnableAutoWrap(false)` 已内置), 手搓 `NewLayoutEle` 时别忘了这一条。

42. **原生 Tree 自绘皮肤(Tree 组件探针实测, 载体路线同 40)**:
    - **自定义项模板要在 `CreateAdapter` 之前 `SetItemTemplateXMLFromString`**, `SetItemHeightDefault(n, n)` 也要在模板/适配器/数据**之前**调用——之后设置不生效(模板 `layout.height="fill"` 会吃掉模板里的 heightT, 行高实际由 SetItemHeightDefault 决定, 探针默认 20);
    - **展开按钮(itemID=1, expandButton=true)的 "+/-" 原生图形**: `ClearBkInfo` 压不掉——背景清空会**回落到"系统默认内容"**(同窗口背景那条例)。必须 Clear 后再 `AddBkFill` 四态全透明色(透明填充算"有内容"就不会回落), 图形消失且**点击展开的原生行为保留**;
    - **展开按钮的点击热区矩形 = `RcItem.Left + 1 + 16×深度` 起、16×16**(开启 `app.ShowLayoutFrame` 后 PIL 实测布局边框红框标定, 每级步进恰好 16)。自绘 chevron 必须居中于这个矩形——画偏了(比如整体再右移半槽)图标就在热区右缘外, 点图标点不中展开; 模板实例的 `XEle_GetRect` 返回的是模板 rect 属性(相对容器), 参照系混乱, 标定要以布局边框为准;
    - 模板子元素绘制在 `XE_TREE_DRAWITEM` **之后**: 自绘回调里画行底/参考线/chevron 会被文字正常覆盖, 展开按钮透明填充不遮挡自绘 chevron;
    - `Tree_Item_` 有 NDepth/BExpand/NState(Leave/Stay/Select)/RcItem(全行宽, 左右各 1px 内缩); 项是 **NID 寻址**(非行号), 没有按行号枚举可视项的 API → 换肤同步文字色用 `RefreshData` 重走模板创建事件, 不能像 List 那样按可视行范围逐个刷;
    - **模板里的 ShapeText(itemID=2)是 Shape 对象不是 Element**, 不能调 `XEle_GetClientRect` 等元素 API(弹"输入句柄可能无效"错误框), 只能 `XShapeText_*`;
    - 叶节点同样保留展开按钮槽位(每项都有 itemID=0/1 的模板实例), 文字全树自然对齐——正好满足 shadcn 树"叶节点留空槽位"的语义, chevron 只需对"有子项"的项绘制;
    - **隐藏页上创建的树, 启动时的 `SetSelectItem` 选中态会丢**(展开态 `ExpandItem` 却保留); 页面真正显示后再补一次 Select 即可(示例程序在页签 OnChange 里做)。

43. **图标抗锯齿: 直通 D2D 几何填充(`d2dicon.go` + `icons.go`, 探针 + 像素实测)**:

    - **炫彩把真的 COM 接口指针交出来了**: `XDraw_GetD2dRenderTarget(hDraw) → *ID2D1RenderTarget`、`XC_GetD2dFactory → *ID2D1Factory`(另有 WIC/DWrite 工厂与 `XDraw_GetD2dBitmap`)。paint 回调里可以纯 Go 直通调用, **不需要 cgo、不需要汇编**, D2D 的 AA 也对直通调用生效(实测 AA 模式 = PER_PRIMITIVE)。
    - **ABI 三规则**(破任一条: 轻则整帧被毁、窗口隐形, 重则崩溃):① 形参里出现 `float32/float64` 的方法一律不能这样调 —— 浮点参数走 XMM 寄存器, `syscall.SyscallN` 传不进去; ② 按值传递的 8 字节结构体要打包成一个 64 位整数(`D2D1_POINT_2F` → `f32(x) | f32(y)<<32`); ③ 其余全用指针/整数传是安全的。
    - **vtable 索引**(实测交叉验证过, 错一位就是调到邻居方法; 曾把 30 当 GetAntialiasMode 实为 SetTransform, 用垃圾指针设了变换 → 整窗不再呈现):`ID2D1RenderTarget` IUnknown 0-2 + `ID2D1Resource::GetFactory` 3, 方法从 4 起 —— `CreateSolidColorBrush=8`、`FillGeometry=23`(参数 (geometry, brush, layer=NULL), 是本方案的唯一出图口)、`SetTransform=30`、`GetAntialiasMode=33`; `ID2D1Factory` 直接继承 IUnknown —— `CreateEllipseGeometry=7`、`CreatePathGeometry=10`; `ID2D1PathGeometry` 继承 `ID2D1Geometry` 的 13 个方法 —— `GetBounds=4`、`Open=17`; `ID2D1GeometrySink` 的 `ID2D1SimplifiedGeometrySink` 部分自有方法只有 3~9 —— `SetFillMode=3`、`BeginFigure=5`、`AddLines=6`、`EndFigure=8`、`Close=9`。
    - **`ID2D1SimplifiedGeometrySink` 里只有 `Close` 返回 HRESULT, `SetFillMode/BeginFigure/AddLines/EndFigure` 全是 `STDMETHOD_(void, ...)`**(同理 `ID2D1RenderTarget` 的 Draw/Fill 全家也是 void):它们的"返回值"是寄存器残留垃圾。曾照着"每个 hr 都校验"的直觉检查 `EndFigure`, 它返回 0x1f80(非 0)被当成失败 → **每次都静默回落折线渲染**:症状是"代码改了, 画面一模一样", 单看输出根本发现不了 —— 探针里必须把每一步的 hr 打出来看。
    - **直通调用拿不到"当前元素"的变换与裁剪**(本库最大的坑):炫彩是在**刷新它自己的绘制命令**(`XDraw_*`)时才施加元素偏移/裁剪的, 直通 D2D 调用发生在回调里、命令队列之前, 因此拿到的是**父级(窗口)坐标系**。实测:元素内 (50,50)-(150,150) 的图标被画到窗口左上角; 反过来"先 XDraw 填充、后画图标"的按钮**看起来正常**(先有填充说明队列已推进), 而幽灵态图标按钮(背景/边框都是 None, 前面没有任何 XDraw 指令)的图标全跑到窗口左上角、正好被自绘的窗口图标盖住 → 表现为"标题栏按钮的图标消失"。
      **正解: 用官方提供的 `XDraw_ConvRect(hDraw, *RECT)` / `XDraw_ConvXY`** —— "从当前 UI 对象客户区坐标转换到画布对齐坐标, 自动适应 DPI", 本来就是给第三方绘图接口用的。换算结果就是**物理像素**(实测 150% DPI:元素在窗口 (60,60), 客户端 (50,50,150,150) → (165,165,315,315)), 直接喂给 D2D 即可, 尺寸与线宽随之自动变成物理像素, 不要再乘一次 DPI。
    - **WINDING 填充模式下所有子路径必须同向**:描边展开是"每段一个四边形 + 每个顶点一个圆帽", 而段四边形与圆帽的天然绕向相反, 重叠处 winding 归零被挖空 → 描边变成一排"梳齿"。修法:鞋带公式定号, 负号就反转点序(`sameOrientation`)。
    - **几何合法性自检**:索引错位/未 Close 的几何喂给 `FillGeometry` 会让 xcgui 整帧不呈现 —— **窗口彻底隐形(连边框都没有)、进程不崩、日志正常**, 只信截图;因此每次出图前用 `ID2D1Geometry::GetBounds`(idx 4;未 Close 的几何返回 `D2DERR_WRONG_STATE` 0x88990001)确认 hr=0 且 bounds 有限。
    - **当时为什么没用 `RT::DrawGeometry` 真描边**:它的线宽是 float 形参, 撞 ABI 规则① —— 于是改在 CPU 侧把描边展开成填充多边形(段四边形 + 圆帽正多边形, 圆帽 12 边在 r=2px 时与真圆偏差 < 0.07px), 一次 `FillGeometry` 出图。展开是"内接近似"(每个形状都落在真·圆角描边内部), 不会比真描边胖。**这条限制在  被 `wapi/com` 的汇编转发器解开, 已改用真描边, 见本节末尾。**
    - **实测对比**(200px 图标同屏对照):新管线墨迹面积 10972 物理像素² vs 理论 10575(误差 4%), 旧折线 16064(胖 50%, 因为线宽被 `ceil` 取整、重叠段在 AA 下互相加深);X 图标边缘灰度层数 5 → 16。旧实现真正难看的原因是**折线点 `round32` 取整到逻辑像素**(18px 图标上 ±1.5 物理像素)+ 8 段离散的多边形棱角, 而不是"XDraw 没有 AA"——实测 XDraw 线条在 D2D 模式下也有 AA(两者像素统计几乎相同), 但几何是精确的。
    - **缓存**:几何是**设备无关**资源(归属 D2D 工厂而非渲染目标), 因此缓存按**工厂指针**作废(工厂变了=设备重建, 整体 Release 重建), 多个窗口(主窗/菜单/弹层各有各的 RT)共用一份几何;画刷是**设备相关**资源(必须由该 RT 创建、只能给该 RT 用), 所以每帧建一个、画完立刻 Release。绘制回调每帧都会新建 Painter(帧内对象), 缓存必须放**包级**。缓存 key = **(path 原文, size, strokeW)** —— **位置(ox/oy)不进 key**: 几何以自身原点构造, 绘制时用 `SetTransform(平移矩阵)` 移过去(见下一条)。
    - **位置外置: 用 RT 变换代替"把 ox/oy 烘进几何"**: 早先 key 含位置, 于是窗口一 resize(位置全变、尺寸不变)时每个图标都是新 key → 45 个图标几何**全部重建**(CPU 描边展开 + CreatePathGeometry), 缓存还会随位置无上限累积; 现在几何只按 (path, size, strokeW) 缓存, 绘制前后成对 `GetTransform` → `SetTransform(平移)` → 恢复。
      实测(示例程序图标页, 窗口尺寸在两档间来回切 10 次): CPU **78 / 93ms → 62 / 62ms**(省 20~33%); 像素验收三组(图标页亮/暗 + 基础页)两组 **0 差**、图标页 **6/2322000 像素**差(都在 AA 边缘, 最大灰度差 16 —— 坐标从"一次加法 `ox + p.x*s`"变成"`p.x*s` 再乘平移矩阵", 浮点加法顺序不同而已)。
      **前提是必须成对恢复**: 实测炫彩自己的绘制(`XDraw_*`)用的是**绘制当时的 RT 变换** —— "设了平移不恢复"会让紧随其后的 XDraw 矩形整体右移 150 逻辑像素(探针实测 的绿色块), 恢复后的蓝色块位置正确 ✓。故 `d2dStroke` 里严格 `defer rt.SetTransform(&prev)`(详见 §65)。
    - **回落路径**:`app.New(false)`(GDI 渲染)时 `XDraw_GetD2dRenderTarget` 返回 0 → 自动回落到旧的折线描边(图标仍可见, 只是没有新管线干净);GDI 模式实测不崩、布局正常。
    - **COM 直通底座已上移到上游 `github.com/twgh/xcgui/wapi/d2d`**: 上面这些"通用但必须有"的东西 —— ABI 三条硬规则、vtable 索引、几何/画刷的创建与释放、几何合法性自检(`GetBounds`)、一步到位的 `NewGeometry`/`FillGeometryColor` —— 正是 xcgui "交了裸 D2D 指针却没给封装"留下的空白, 现在收进 xcgui 的 `d2d` 子包; **COM 基础件(`ComProc`/`IUnknown`/`IUnknownVtbl`/`InterfacePtr`)另抽到 `wapi/com` 子包共用**(d2d 用类型别名引用, 虚表字段仍写 `ComProc`; edge 包自带一份同源实现, 迁移面较大暂未动)。封装形式照 **edge 包(WebView2)** 那套(每接口一文件、虚表按 Windows SDK 声明顺序**完整列出**、调用一律 `ComProc.Call` 带 `//go:uintptrescapes`、中文注释 + MSDN 链接), 接口结构体再按 COM 继承链**值嵌入**串起来:
      `IUnknown{ Vtbl }` → `ID2D1Resource` → { `ID2D1RenderTarget`, `ID2D1Geometry`→`ID2D1PathGeometry`, `ID2D1Brush`→`ID2D1SolidColorBrush` }、`ID2D1SimplifiedGeometrySink`→`ID2D1GeometrySink`、`ID2D1Factory`。于是 `AddRef/Release/QueryInterface` 只在 `IUnknown` 上实现一次、`GetFactory` 只在 `ID2D1Resource` 上实现一次; 各接口自己的方法用未导出的 `vtbl` 把同一个对象指针按**自己的**虚表类型重解释。
      三个必须记住的点(都有测试守着):
      ① **`vtbl` 必须取 `i.Vtbl`(对象首字段里存的虚表指针), 绝不能把对象地址当虚表**: 写成 `(*ID2D1FactoryVtbl)(unsafe.Pointer(i))` 时会从"对象 + 80 字节"处读垃圾当函数指针 → **跳到非法地址**(实测 PC=0x989680,示例程序启动即 panic、退出码 2; `-H windowsgui` 构建看不到任何堆栈, 必须用带控制台的构建 `go build -o dbg.exe ./demo` + 重定向 stderr 才抓到)。`TestVtblResolution` 用"假虚表 + 假对象"把这条钉死。
      ② **虚表字段顺序就是 ABI**: `TestVtblIndexLayout` 用 `unsafe.Offsetof` 断言 `CreateSolidColorBrush=8`、`FillGeometry=23`、`IsSupported=50`、`CreatePathGeometry=10`、`Open=17`、`SetFillMode=3`、`Close=9`、`SetColor=8` 等槽位; `TestStructLayout` + 包内编译期断言保证每个接口结构体**只有一个指针大小**(否则与 COM 对象布局错位)。
      ③ **nil 语义**: 构造函数在句柄为 0(未开启 D2D)时返回 nil, 调用方必须判空; nil 上只有 `Release`(每个接口都覆盖了判空版)与"自有方法"安全 —— 值嵌入下**被提升的基类方法**在 nil 外层指针上会先解引用嵌入字段而 panic(基类里判空也救不了), 这是 Go 的语义而非实现缺陷。
      `XDraw_GetD2dRenderTarget` / `XDraw_ConvRect` / `XC_GetD2dFactory` 的注释也补齐了"直通拿的是父级坐标系, 必须先 ConvRect"这条坑。本库的 `d2dicon.go` 因此只剩图标语义(几何构造 + 盒计算 + 缓存 + 元素 alpha), 少了约 200 行重复底座。

    - **换用 `RT::DrawGeometry` 真描边(取代 CPU 展开)**: `wapi/com` 的汇编转发器把"浮点形参"这条 ABI 限制解开之后(见 §65 的 `CallWithFloat32Stack`), 描边不再需要在 CPU 侧展开 —— 折线保持**开口**直接交给 D2D(`DrawGeometry` + `RoundStrokeStyleProperties` 的圆角连接/圆头), 少数 fill 色点仍走 `FillGeometry`。
      实测(探针 , 同几何同线宽各画 2 万次): **DrawGeometry 约 20ms, 填充"已展开的环"约 80ms(4 倍)**。但**真实场景看不出差异**: 图标页 45 图标/帧, 省下约 0.14ms/帧(相对 ~4ms/帧), 鼠标扫动的 CPU 两轮 828ms(旧) vs 836ms(新)、resize 来回切 62ms vs 78ms —— 全在噪声带里。**所以这次换路线的主要收益是"质量 + 代码", 不是帧率**:
      ① 圆角连接/圆帽由 D2D 精确保证, 与 lucide 的 `stroke-linejoin/linecap="round"` 语义一致(旧版是斜接近似; 放大对比里三角形、八边形图标的"角变圆"); ② `d2dicon.go` 删掉 `strokeOutline/addArc/miterPoint` 约 300 行; ③ 几何既不含位置也不含线宽, 缓存 key 只剩 `(path, size)`。
      像素对照(改动前后各编一份示例程序): 图标页 light **6958/2322000(0.30%)**、dark **7644(0.33%)**、基础页 **1051(0.045%)**, 差异全部集中在图标边缘(形状差异而非渲染错误); 图标墨迹量 142960 → 143083(+0.09%, 圆角比斜接略多一点面积)。
      **探针里的两个坑**: ① `ID2D1RenderTarget::Flush` 不能在有炫彩绘制回调里调 —— 会让进程静默退出(预热那步就没了下文), 所以性能只能按"提交路径耗时/大批量反压"来测; ② 想拿真实吞吐就得把次数堆到 2 万级, 60 次时四组全是 0.00ms。
      **验收**(改动前后各编一份示例程序二进制同机对照): 图标页 + 基础页 × 亮/暗共三组 PrintWindow 抓图**逐像素 0 差**(1800×1290 = 2322000 像素全等, 2798 色); 图标页真实鼠标扫动 3.62s 的 CPU 时间交替多轮无可辨差异(±10% 噪声); d2d 包 8 个单测覆盖颜色换算 / 点打包 / 虚表槽位 / **虚表解析** / 结构体布局 / 继承复用 / nil 安全 / 空句柄。

    **填充几何的 figure 数必须少(图标总览页卡顿的根因, 重要)**:

    - D2D 填充**自重叠**路径走的是 stencil-and-cover:代价随**子路径(figure)数**近似平方增长, 与顶点数关系不大。探针实测(45 个图标同框、每图标同规模):9 个 figure/图标 → 4.1ms/帧;39 个 → 39ms;99 个 → 350ms;**199 个 → 1590ms**。
    - 最初的"描边展开"写法是**每段一个四边形 + 每个顶点一个圆帽**(一个图标上百个 figure), 于是 45 个图标一帧要 **719ms** —— 表现为示例程序图标总览页每动一下鼠标就掉帧("非常的卡"), 切页/换肤这种整窗重绘直接卡住近 1 秒。
    - 修法: 展开成**每条子路径一条闭合轮廓**(`strokeOutline`)。构造:两侧各按 w/2 偏移, 顶点处**外侧补圆弧**(round join)、**内侧取两条偏移线的交点**(精确, 不是直连); 两端各补一个半圆帽。改完实测 **2130 个图标平均 4.11 条轮廓/图标(最多 20 条, `virus` 尖刺多)**, 45 个图标一帧 **4.14ms**(= 空窗口基线, 图标开销归零)。`icons_test.go` 的 `TestIconGeometryFigureCount` 守住这条红线(>32 条就报错)。
    - **轮廓绕向必须一致**:所有轮廓的绕向由构造方式天然决定(与折线自身绕向无关), 因此 WINDING 填充下并集即笔画形状; 跨子路径的重叠(X 的两笔、太阳的辐条)也会正确并起来。填充子路径(lucide 的色点)则用 `sameOrientation` 归一。绕向不一致会让重叠区 winding 归零, 描边裂成一片片(`TestStrokeOutline` 守住)。
    - **此路不通的记录**: 本可以用 `ID2D1Geometry::CombineWithGeometry(UNION)` 一次性把图元汤合并 —— 索引按 Windows SDK `d2d1.h` 的声明顺序是 11(`GetBounds=4`、`CompareWithGeometry=8` 都已逐个验证), 参数全是枚举/指针(可 syscall), 但**实测调用直接崩(访问违例)**, 传不传单位矩阵都一样。顺带发现 `ComputeArea(13)`/`ComputeLength(14)` 的首参 `worldTransform` 是 `_In_`(**不能传 NULL**, 传了就崩), 而 `GetBounds(4)` 的 `transform` 是 `_In_opt_`(可 NULL)—— 本文件因此只用 `GetBounds` 一个几何方法。
    - **示例程序侧同样要省重绘**: 格子元素的 `MouseStay` 在"鼠标在元素内移动"时**每帧都会来一发**, 不判状态就会变成每个鼠标移动事件触发一次整窗重绘; 只在 hover 状态**变化**时 `Redraw` 即可(图标页 45 个格子扫一遍 CPU 从 0.5s/3s 降到 0.17s/3s)。

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
      (`parseIconPathEx` 的 complete 标志, 正是上面那类静默截断的探针)、点落在 24 网格内、填充标记存在。
      预处理回归时立刻炸, 不用靠肉眼比对整本图标。
    - **对照验收**:同屏对照浏览器渲染的原始 SVG(headless Chrome + `--force-device-scale-factor=1.5`
      对齐 150% DPI), 32 个覆盖全部图元类型的图标逐格像素差 ≤ 4.2/255, 墨量比 0.95~0.97
      (残差来自我们 `#0a0a0a` 与参考纯黑的色差);1.52.0 新增的 9 个图标单独复验 ≤ 4.7/255。
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
      **验收**(探针实测): heroicons 官方 SVG 全文画出的星形与 lucide 内置形状一致;
      手写 16 网格 SVG(`<g fill="none" stroke="currentColor">` 两条线 + 一个 `fill="currentColor"` 色块)的
      "十字 + 实心方块"正确 —— 缩放与填充判定都对。**不支持**: `<circle>/<rect>` 等形状元素、transform
      属性、CSS class 里的样式(这些元素/属性被忽略, 不报错)。顺带修了 `parseIconPathEx` 对**尾部空白**的误判
      (会被当成"重复上一指令"从而判数据不完整; 生成资源都以 `Z` 结尾, 一直没暴露)。
    - **解析层已上移到上游 `github.com/twgh/xcgui/svgp`**: path 语法解析(全命令/相对/紧凑写法/
      弧转贝塞尔)、SVG 文档遍历(`<g>` 继承 / viewBox)、子路径分组与绕向统一、d2d 几何组装都在那边
      (纯 Go, 不经过 DLL; 14 个单测), xui 只留图标语义: 24 网格归一化、`!` 填充前缀(现在也是 svgp 的
      扩展语法)、SVG 原文缓存、GDI 折线回落。**迁移后像素实质等价**: 图标页 148/2322000(0.0064%)、
      dark 185、基础页 **4** 像素, 全是 `Z` 闭合语义(补重复点+OPEN → CLOSED)的 AA 级微差,
      墨迹量 143083 → 143082; 本库再薄约 500 行。选 svgp 而不是改 xcgui 自带 `svg` 包的原因:
      后者是 DLL 里的 C++ 实现, 而且无法灵活更改 —— 纯 Go 是唯一能继续改进的路径。

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
    - **用法**:应用初始化 `a.SetPaintFrequency(10)`(示例程序已加)。**为什么是 10 不是 16**: 节流按"两次绘制的间隔"算,
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
      不存在"没抗锯齿/要换灰度 AA"这个可调点 —— 想关彩色边缘得另找路径(1.4.2 未见暴露)。

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
      (12 倍时外部截图脚本才能抓到中间帧做像素验收)。
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
         可用 —— menu 弹层的进场淡入已按此实现(见 64); 位移滑动不可行(会崩, 见 64)。**
      6. 首帧闪现: `SWP_SHOWWINDOW` 后的首次绘制带着上一次的 alpha(255) → 进场前先
         `SetWindowAlpha(1/255)`(见 3, 不能是 0)。
    - **验证**(探针 + `SetAnimationScale(12)` 放慢 + 外部逐帧像素实测):气泡淡入
      255→254→245→232→201→175→141→117→91→74→60→45→37→27→22→18→14→12→**10**
      (17 帧, 150ms×12)且顶部 y 36→**33**(8px 逻辑滑入); 1× 常速 77→24→**10** 正常完成;
      对话框遮罩点 255→251→247→241→237→235→232→231→**229**(淡入)与反向(淡出)平滑往返,
      淡出到位后内容树隐藏(`窗口最暗` 回到 255)。

    - **backdrop-blur-xs 完成**(`ui/backdrop.go`, 前一晚被误判卡死):
      思路 = 打开时 GDI `PrintWindow(PW_RENDERFULLCONTENT)` 抓窗 → Go 盒式模糊 ×4 趟
      (backdrop-blur-xs=4px 逻辑)→ **元素 BkM 图片填充**显示在遮罩之下(元素透明度能跟着
      遮罩一起淡入。注: 当年选 BkM 是误信"元素 paint 画不出位图", 实际能画 —— 见 17 条
       修正; BkM 路线已验证且淡入一致故保留)。验收 = blurprobe 程序化
      Show/Close + 像素逐点比对(模糊+10%变暗复算期望 mean=0.23/255, 不模糊假设 mean=4.64
      被显著排除; 关闭还原全图 max=0)+示例程序 computer-use 真实鼠标双主题复核。
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
         在示例程序(布局盒子嵌套)能渲染、在平铺窗口的探针里不渲染, 原因未查明 —— 统一用
         带 id 的路线绕开。注意重复注册前先 `XEle_ClearBkInfo`(累积 >20 个炫彩报错)。
      4. **没有消息循环就没有一切**: 探针若只 Sleep 不跑 `a.Run`, Presence 淡入淡出
         (窗口定时器)全不工作, 关闭也不会真的隐藏 —— 无泵跑出来的"验证结论"全部作废。
         程序化验收要用 `w.SetTimer + AddEvent_Timer` 驱动阶段, 再 `a.Run`。
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
           左上角、还被放大好几倍"(踩过这个坑; 当时用 System.Drawing 做的对照实验给出了
           "它会缩放"的错误结论, 那次实验作废, 以本库这份 Go 探针为准)。
         - 想要小图只能自己降采样; 而这一趟本来就要读遍全部源像素(全屏物理 3.9M), 顺手把
           通道交换与盒式平均一起做掉 —— 实测降采样比"只抓屏"几乎不增加耗时(常态 26→27ms)。
         - 原实现 = "整分辨率模糊 ×4 趟(半径按 DPI 放大) + 事后缩放", 模糊代价随面积增长
           而全屏物理面积是逻辑的 DPI² 倍 → 2560×1528 @150% 全屏 **160ms**、普通窗口 **39ms**,
           这就是"全屏时点按钮弹窗比正常窗口慢"的根因。
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
         (示例程序是 12 逻辑像素), `PrintWindow` 把半透明阴影抓成**纯黑**(alpha 丢失),
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
         于是 dialog.go 里做了 `CX*2` 校正 —— 那只对**恰好 2 个字**的按钮文案凑巧对
         (`保存` 15→30, 真实 30), 换成 3 个字的 `上一页` 就偏小(30 vs 真实 45) →
         分页组件的"上一页/下一页"文字被矩形裁掉半截(实测只剩 "上-" / "-页")。
         - 正确用法: **给足 width, 并加 `TextFormatFlag_NoWrap`**,
         `GetTextShowRect(str, -1, hFont, TextAlignFlag_Left|TextFormatFlag_NoWrap, 1000, &sz)`
         返回的 `CX` 就是单行真实逻辑宽度(14px 字号的汉字恰好 15/字, 与绘制结果一致),
         不需要任何倍率校正。量宽必须与绘制同字体、同 DPI。
         - 顺带两条组件侧的实测结论(分页组件踩出来的): ① 自绘文字矩形**右/左边界取到壳体边缘**、
           只在一侧留 padding, 量宽有 ±1px 误差也不会截字; ② 组件在 `OnChange` 之前的立即重绘
           (`XEle_Redraw(ele, true)`)会连带清掉窗口的待重绘区域, 调用方在回调里改的
           **原生控件文字(Label 等)会滞后到下一次无关重绘才出现** —— 组件改状态后一律用
           `Redraw(false)`, 原生控件自己改完文字也要显式 `Redraw(false)`(示例程序里 `SetIcon` 后
           本来就带 `w.Redraw(false)`, 同理)。

         49. **Tooltip 被"迟到的 MouseLeave"秒关: 密集元素上会表现成"提示根本不显示"**(图标网格实测 + 日志定位): 同一个 Tooltip 绑定到一整片密集元素时(9×4 个 77×60 的格子),
         鼠标从一个格子划到相邻格子, 实测事件顺序是
         `leave(A) → show(B) → leave(A) 又来一次`(xcgui 的 stay 元素切换与鼠标移动是两条到达
         路径), 第二条 leave 会把刚进场的气泡立刻关掉 —— 日志里能直接看到
         `show hEle=616 → fresh 进场 → Hide → show hEle=612 → Hide → 退场到 0`, 气泡只亮了
         十几毫秒, 肉眼就是"tooltip 全都不显示了"; 绑到稀疏元素上(标题栏按钮)时不会撞上,
         所以之前一直没暴露。
         修法 = 气泡记住自己当前"描述"的元素 `shownEle`, `MouseLeave` 里只有
         `hEle == shownEle` 才真隐藏, 迟到的 leave 忽略(需要强制关掉的场景走 `HideTooltip`)。
         排查手法值得记一笔: 在 `show/Hide` 与 presence 的 apply/done 里追加写文件日志,
         跑一遍鼠标轨迹就能把"窗口没建/没显示/被谁关掉"三类原因区分开 —— 这次是**被自己关掉**。

         50. **Tooltip 是深色气泡, 压在深色背景上肉眼容易漏看**: 亮色主题下气泡是
         `bg-foreground`(近黑)+ 白字, 悬浮在标题栏/暗色面板上方时几乎和背景融为一体,
         截图验收时别急着判"没显示" —— 先按元素几何算出气泡应该出现的矩形, 或者干脆用
         日志/pixel_scan 确认, 否则很容易把正常渲染误判成 bug。

         51. **隐藏容器里建的 Tree, 首次显示"只有行底和 chevron, 文字全空"**(结构页实测 +
         真实鼠标复现; 该问题由**当天早些时候的布局重构引入**, 见下面"回归来源"): 启动在别的页签、点进
         "结构"页时, 行底/缩进线/chevron/选中态全对, **每一行都没有文字**; 再切一次页就好了, 且**等多久都
         不会自己恢复**(8 秒后截图仍空白)。
         - 根因: 原生 Tree 的项模板是在**布局排版**(元素处于 show 状态时的那次排版)里实例化的。树建在
           隐藏的祖先容器(页面容器 `Show(false)`)里、并且尺寸只由创建参数给定时, 全程没有任何一次
           "可见时的排版"落在它身上 → 模板从没实例化; 首次显示时 `XE_TREE_DRAWITEM`(行底/chevron/参考线
           都是我们自绘的)照常跑, 但模板里的 ShapeText 没有内容。**绘制不负责补模板实例**。
         - **回归来源(布局重构)**: 重构前示例程序是 `NewTree(..., cy=0, ...)` + 布局项声明
           `itemFixed(t, cw, 322)` —— 页面首次显示时那次排版把元素从 0 拉到 322, 这次"可见时的排版"正好
           把模板实例化了; 重构按新约定"尺寸写创建参数、布局只声明要改变的轴"改成 `NewTree(..., 322, ...)`
           且不再声明, 于是那次排版不再落到树上 → 首次显示空白。已用 `git worktree` 拉 24fcb6b 重新构建
           对比确认(旧版正常、新版空白)。
         - 探针数据(关掉下面的修复后逐个试): 创建 `cy=0` + 布局项 Fixed 尺寸 → **正常**; 创建 `cy=322` +
           同样数值的布局项声明 → **空白**(数值相同, 排版本不产生变化); 创建 `cy=322` 后在隐藏期间
           `SetHeight(321/322)` 手动制造尺寸变化 → **空白**(隐藏期间的排版不算); 用 `XUI_HOME_PAGE=4`
           让结构页**创建时就是 show** 的 → **正常**。四组对照把条件钉死成"必须有可见时的排版作用于它"。
         - 修法(`ui/tree.go`): `NewTree` 挂 `AddEvent_Paint` 记下"已首次绘制", 再用炫彩定时器
           (`treeWakeTimerID`, 1ms ≈ 下一帧)排一次 `RefreshData + Redraw(false)`, 让刷新发生在绘制回调
           之外。刷新不依赖调用方, 所以任何隐藏容器/多页签/对话框里建的树都自愈; 代价是首帧后有一次
           刷新(一次性、只刷可视项), 树本来就建在可见处时等于多做一次无害刷新 —— 比"要求调用方声明
           布局尺寸"更稳(那种写法一旦尺寸与创建参数相同就失效)。
         - 探针手法(值得复用): 给组件挂上 `AddEvent_Paint / AddEvent_Show / AddEvent_Size` 写文件日志后
           一眼看出"**只来了一次 Paint, SHOW 一次都不来**" —— 父容器 Show 不会给子元素发 `XE_SHOW`, 所以
           "我可见了"不能用 SHOW 事件判断, 只能拿首次绘制当信号; 另外"在页签 OnChange 里手动
           `RefreshData` 后截图立即正常"这一步把原因锁死在"模板没实例化", 而不是几何/主题/颜色。

         52. **ComboBox 弹层高度: 内容高 = n×(项高+1) + 2×pad + 5, 取该值才无滚动且留白对称**
         (ListBox 探针参数矩阵 +示例程序截图/像素剖面实测): 症状是 6 项弹层悬浮最后一项时
         hover 背景底边正好贴住弹层底边、滚轮还能再滚一下。
         - ListBox 内容有**固有内缩**: 首项 top = pad + 3(上下各固有 3px、左右固有 2px), 行步进 =
           项高 + 1px(相邻项 top 差 29 @ 项高 28); 内容高 = n×(项高+1) + 2×pad + 5
           (顶固有 3 + 底固有 3 − 最后 1px 间隙)。公式在 pad=0/4/8 与 n=3/5/6/10 上均验证过。
         - 弹层高度 < 内容高 → 出滚动条且"差多少滚多少"(实测 6 项 H=180 内容 187 → range=10, 能再滚
           一下); > 内容高 → 底部多留白。**高度取内容高**时: 无滚动、顶部/底部留白对称(各 pad+3)。
         - 固定模式必须在**弹层打开前**`EnableDropHeightFixed(true) + SetDropHeight(h)`(组件在构造期
           调, 见 `ui/combobox.go` 的 `comboDropHeight`)。
         - 探针手法: ListBox 弹层内 `AddEvent_ListBox_DrawItem` 打印 `pItem.RcItem` 可拿到每项真实
           绘制矩形(项坐标与滚动量都能看出); 弹层打开后用 `GetScrollBarVObj.GetRange/XSBar_GetCurPos`
           判有无滚动。
         - 验收(截图 + 像素剖面): 悬浮最后一项时 hover 背景底边与弹层底边框间应有 ~pad+3 留白;
           滚轮不产生位移 —— 对比滚动前后同一竖线(x=item 右侧空白列)的灰/白分段, 各条 hover/选中带
           的 y 区间应完全不变(变化只说明鼠标 hover 换了行)。

53. **Pagination: 折叠必须"恒定槽位", 且组件在 OnChange 前不要用立即重绘**(`ui/pagination.go` +示例程序图标页, 亮/暗双主题 + 真实鼠标验收):
    - **组件路线 = 纯自绘单元素**(同 slider, 不拼 Button 子元素): 页码数量随数据变化, 拼子元素要反复增删/重排; 条目 = `[上一页][1][2]…[n][下一页]`, 内容居中(justify-center), 页码 32 方形(ghost / 当前页 outline), 上一页/下一页 = `chevron(size-4)` + 文字; 悬停 `xui.HoverItem` 逐项过渡、按下 `active:translate-y-px` 下沉 1px; 到边界时上一页/下一页按禁用画(50% 透明且不响应鼠标)。API `NewPagination / SetTotal / SetPageSize / SetPage / Page / PageCount / OnChange`(`SetPage` 不回调, 供调用方同步数据游标)。
    - **折叠策略必须"恒定 7 个槽位"** —— 首段 `1 2 3 4 5 … n` / 中段 `1 … p-1 p p+1 … n` / 末段 `1 … n-4 … n`。shadcn 文档那版三段分别占 6/7/7 个槽位, 翻页跨段时整行**宽出一个页码位**, 居中排布下"下一页"跟着右移 —— 实测连点 10 次同一坐标点不到(用户反馈"点着点着就点不到了"); 恒定槽位后同坐标连点 10 次稳定到第 11 页。
    - **组件在 `OnChange` 之前用 `Redraw(true)` 会吞掉调用方的待重绘区域** → 调用方在回调里改的原生控件文字(Label 等)滞后到下一次无关重绘才出现; 组件改状态后一律 `Redraw(false)`, 原生控件自己改完文字也要显式 `Redraw(false)`(同第 48 条)。
54. **元素销毁事件会为子元素逐个触发(含嵌套), 顺序在窗口 `XE_DESTROY` 之后**(探针实测):
    - **三种销毁路径都触发** `XE_DESTROY`(33) + `XE_DESTROY_END`(42): ① 元素显式 `Destroy`; ② 直接挂在窗口上的元素随 `CloseWindow` 连带销毁; ③ **嵌套在子容器(LayoutEle)里的元素同样触发**。
    - **顺序**: 窗口的 `XE_DESTROY` 先到, 再按层级逐个销毁子元素(实测日志 `WINDOW → C(嵌套) → A(直挂)`)。
    - **意义**: 这是"把订阅/缓存绑到元素生命周期"的依据 —— 运行时的 `xui.OnThemeChangeFor(hEle, fn)`(主题订阅随元素销毁自动反注册)与 `anim.go` 里 `AddEvent_Destroy_End` 清理 hover 状态, 都挂在这上面。对裸句柄注册要用 `widget.NewElementByHandle(hEle)`, 且**必须走 `AddEvent*` 事件总线**(不能用 `XEle_RegEventC` 那套有 2000 上限的注册)。

55. **布局尺寸语义实测全集: `Fill` ≠ "剩余空间", `Weight` 只在父级尺寸"确定"时生效**(探针六组对照 +示例程序布局重构;示例程序之前那套"卡片用绝对坐标摆"的做法就是被这些语义逼出来的):
    - **`Fill` = 父级整尺寸(减自身四边外边距)**, 不是"剩下的都给它": `Fill/Fill + margin 24` 在 800×600 的父级里得到 752×552 ✓; 但**前面还有兄弟项时它不会让位** —— 竖向 `[Fixed 60][Fill]` 里 Fill 拿到的仍是整高 600(而非 540), 直接溢出容器。竖向堆叠想要"剩下的高度"不能用 Fill。
    - **`Weight` 才是"按比例分剩余空间"**: 竖向 `[Fixed 60][Weight 1]`(父 400)→ 340 ✓; `Percent` 同源(Percent 50 在同一个父级里 = 170, 即剩余空间的百分比)。
    - **但 `Weight`/`Percent` 只在父级尺寸"确定"时才成立**: 父级是 `Fixed`、或是被 `Weight` 解析出来的尺寸、或是**窗口本身** → 正常; 父级是 `Fill` 或 `Auto` → **恒为 0**(实测: `root(Fill)` 里的 Weight 高度 = 0, 换成窗口直属后立刻正确)。**宽度方向不受影响** —— 横排里 `Weight` 列在 Fill 宽的行里一直正常。
    - 推论(已踩): **竖向主链中间不要垫 `Fill` 容器**。要从窗口开始一路 `Weight`, 每级都是确定尺寸, 下一级的 Weight 才算得出来 ——示例程序的窗口骨架因此删掉了原来那层 `root(Fill)`, 改成 `[标题栏行 Fixed][内容区 Weight 1]` 直接挂在窗口上。
    - **`Auto` = 按内容自撑**: 两张 Auto 高度卡片竖排各自高度准确(120/184, 与内容理论值一致)、互不挤压 —— 这是"卡片不用手算高度"的正解。Auto 卡片里**不要放 Weight 子项**(Auto 父级 → 0), 里面用 Auto / Fixed 就好。
    - **要让"面板随窗口长高", 整条竖链必须全 Weight**: 窗口 → 内容(Weight) → 页(Weight) → 内容区(Weight + 外边距) → 行(横排, 高 Weight) → 列(宽 Weight + **高 Weight**) → 卡片(高 Weight) → 内容区(Weight) → 面板(Weight)。实测面板 700 高窗口下 628、900 高下 828, 逐级跟随 ✓。注意"列"的高度: `xui.VStackWeight` 用的是 `Fill`(= 父级整高), 列内再用 Weight 就废了 ——示例程序因此加了 `colStretch`(高度 Weight, 见 `demo/main.go`)。
    - **普通 `widget.Element`(非 LayoutEle)可以当布局项**: 被父布局排版、跟随缩放都正常(探针对照 A/B 一致)。而 `EnableLayout / EnableHorizon / EnableAutoWrap` 这些"**当容器排子项**"的开关只有 LayoutEle(与窗口)有 —— 所以卡片能否排自己的内容取决于它是不是 LayoutEle(见 56)。
    - **`Show(false)` 的子项会被布局跳过**(排成 0x0): 页签式"同位置叠放多页"因此不需要绝对定位 —— 竖向堆叠里隐藏的页不占位。改可见性后**必须重排宿主**(`AdjustLayout`), 否则新显示的那页拿不到尺寸(实测只排那一页时仍是 0x0)。
    - **窗口必须 `Show` 过之后布局才有非零尺寸**: `Show` 之前即使调 `AdjustLayout`, root 恒 0x0(探针里读不到数值就是这个原因, 不是探针写错了)。
    - 另: `xc.XEle_SetPadding(hEle, l, t, r, b)` 存在(元素级内边距); 本库暂用"子项外边距"达到同样效果, 未实测。

56. **Card 改成布局容器: 内容区随卡片尺寸自动联动**(`ui/card.go`):
    - 卡片底层由 `widget.Element` 换成 `widget.LayoutEle`(竖直流式 + `EnableAutoWrap(false)`), 内容区 `ContentStack(space)` 是它的 **Fill 宽 + 外边距 = 内边距** 的子项 —— 卡片被父布局拉伸时内容区与其中的控件自动跟随。旧写法要调用方拿 `ContentWidth` 去算, 那个值取的是**创建那一刻**的宽度, 卡片一缩放就错(且卡片本来就不该知道自己的最终宽度)。
    - 内容区高度是 **Auto**(按内容自撑), 于是卡片高度可以 Auto(自撑)也可以固定/Weight, 内容一律从上往下排。内容要撑满卡片高度(表格/树)时: 卡片高度给 Weight, 再对内容区与目标子项声明 `xui.ItemFillHeight`(Weight 1, 受 55 的"父级确定尺寸"约束)。
    - 一个卡片只有一个内容区: 重复调用 `ContentStack` 返回同一个(只改 space)。页脚 `CardFooter` 以**卡片**为父句柄创建、声明宽度铺满 —— 内容区高度给了 Fill/Weight 时会自动让出页脚那一段。

57. **异型透明窗(Shaped)的"整窗透明度"只认第一次调用, 之后永远冻结**(探针 + 抓屏/像素实测; Tooltip 气泡"窗口明明显示了却看不见"的根因):
    - **症状**: 悬停后提示气泡**确实被显示出来**(枚举窗口可见、矩形正确、位置还随进场动画滑动), 但屏幕上什么都没有 —— 用户侧的各种描述(如"点几次切换主题后 ToolTip 就不显示了")都源于此。
    - **定位过程(手法值得复用)**: ① ⚠️ `Process.MainWindowHandle` 在 tooltip 小窗(懒创建, 带 `WS_EX_TOOLWINDOW`)存在后**返回的是小窗而不是主窗口**, 用它算坐标会让后续所有悬停/抓图全错(踩过; 改成"枚举进程顶层窗口, 取可见且面积最大"); ② 用 `PrintWindow` 抓小窗自己的内容 vs `CopyFromScreen` 抓同一块屏幕, 二分出"**画是画了的**, 只是屏幕上没有" ⇒ 合成/透明度问题; ③ `user32!GetLayeredWindowAttributes` 读出 layered alpha = **3/255**(窗口可见但几乎全透明)。
    - **根因**(探针复现): `XWnd_SetTransparentAlpha` 对 `Window_Transparent_Shaped` 窗口**只在第一次调用时生效** —— 那一发把窗口切成 layered 并写进 `SetLayeredWindowAttributes`, 之后无论设什么值、`XWnd_Redraw(本窗, 立即)`、`XEle_Redraw`、`SetWindowPos` 移动、重绘宿主窗口、`ShowWindow`, **layered alpha 都停在首次那个值不再变化**(实测 3 → 200/120/60/90/30 全部无效)。于是"进场动画第一帧"的极小值被永久钉住 ⇒ `FadeWindowAlpha` / `SlideWindowAlpha` 这类**整窗 alpha 动画在异型窗上等于没做**。(`Window_Transparent_Simple` 则是从建窗起就不是 layered 模式、该 API 同样不生效。)
    - **修法**(`ui/tooltip.go`): 进出场**改用元素 alpha**(`xui.SetElementsAlpha` / `XEle_SetAlpha`, 由框架在合成阶段逐帧乘上去, 每帧都生效, 对话框整组淡入就是这么做的, 见 47); 整窗 alpha 只在**窗口显示出来之后**设一次 `1`(不透明)钉住; 位置滑动仍走 `SetWindowPos`; 驱动计时器仍挂宿主窗口(异型小窗自己的定时器会掉到约 100ms/帧)。
    - 首次进场不再需要"把窗口 alpha 压到底"防闪: 元素 alpha 直接以 0 起步, 首帧什么都不画即可(**元素** alpha 允许为 0; 只有**窗口** alpha 不能为 0, 否则 xcgui 会停止合成该窗口, 见 `SetWindowAlpha`)。
    - **验收**(真实鼠标 + 抓屏像素, 亮/暗双主题各 3 轮): 亮色下悬停处"深色像素"由 7996 升到 9022(气泡是深色块); 暗色下"亮色像素"由 0 升到 2979(气泡是浅色块) —— 连续 6 次切换主题后每轮增量完全一致, 即"切主题把气泡弄坏"的现象消失。
    - **通用判据**: `IsWindowVisible=true` + 矩形正确 + **屏幕像素为空** ⇒ 先怀疑整窗 alpha / 窗口形状, 不要只凭"窗口可见"下结论。

58. **独立置顶小窗不跟随主题切换重绘, 也不在"已注册窗口"清单里**(`ui/tooltip.go`):
    - **现象**: 气泡已经挂在屏幕上时点标题栏"切换主题", 页面与标题栏都换了配色, **气泡还是原来的配色**, 直到鼠标移开重新进场才更新。
    - **原因**: `SetTheme` 只做三件事 —— 重刷 `BindWindowBackground` 的底色、跑 `OnThemeChange` 订阅回调、重绘 `RegisterWindow` 登记过的窗口(见 manager.go)。Tooltip 的气泡是组件自建的**独立置顶小窗**, 既不在登记表里、也不是"组件窗口", 于是没人重绘它; 宿主窗口重绘只带动**合成**(透明度), 不动它的**内容**。
    - **修法**: 组件自建窗口要自己订阅主题 —— `xui.OnThemeChangeFor(元素句柄, fn)`(元素随窗口销毁时自动反注册, 见 54), 回调里**重跑一遍 show**: 重新量文字、重算气泡几何、重绘。重跑而不是单纯 `Redraw` 是因为主题可能换字体/字号, 尺寸得按新度量重算(目标元素已被 `Unbind` 时退化为纯重绘)。
    - **验收**: 悬停标题栏主题按钮(气泡弹出, 亮色下是深色块) → 光标不离开按钮直接点它切主题 → 同一区域抓图, 气泡立刻变成浅色块(浅色像素 4432, 抓图对比见 探针实测 的 `cap_hover/cap_click`); 页面元素的"Icon"气泡在改动后依旧正常(深色像素 +1026, 与改动前完全一致)。
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
    - **修法(本库现在采用的骨架)**: 每个"栅格"用 **一行**(行高 Weight) + **`Weight` 宽的列, 列高声明 `Auto`**(`示例程序` 的 `col` 已改成 Auto), 卡片按内容顺序**竖着排在列里**; 整宽卡片直接挂 `body`(实测正确)。竖排还顺带修掉现象②: 同一列里两张卡是"先后关系", 不再有行与行之间的重叠。
    - **验收**(真实截图像素, 1600% 缩放阈值): 输入框右端 x=863 = 卡片内容区右缘(863), 卡片边框在 887; 反馈页进度条 x 78..1721 = 内容区 78..1721; 基础页分隔线 x 1103..1721 = 内容区 1103..1721。探针矩阵见 探针实测(A/B/C/D/E/P1/P2 候选)。
    - **代价**: 页面内容比窗口高时会被页面自身的 `EnableOverflowHide` 裁掉(不再重叠, 但也看不到)。要"看得到"得给页面加滚动容器 —— **已解**: 页面换成 `ui.ScrollArea` 滚动容器, 见第八节 66。

60. **"自适应列宽"依赖元素宽度, 而宽度是布局算出来的 —— 必须订阅元素尺寸事件自愈**(`ui/table.go`):
    - **现象(实测)**: 刚进入数据页时表格**最后一列(自适应列)被挤成 20px**, 表头文字裁成"日"、单元格文字全是"..."(`...` 是形状文本放不下时的截断标记); 拉一下窗口尺寸(或换页回来)就正常。
    - **根因**: `NewTable(x,y,0,...)` 的 `cx=0`(尺寸交给布局), 于是建表时 `GetWidth==0`, `colWidth` 算出 `0 - 表头内缩*2 - 固定列之和` → 负数 → 兜底 20。修复前的策略是"调用方自己在窗口尺寸事件/换页时调 `FitColumns`", 但**首次显示**这条路上没人调: `w.Show` 之前元素尺寸全是 0, `show(home)` 里的 `fitTable` 也量到 0(见 55), 之后窗口尺寸事件又不一定发。
    - **修法**: 组件自己订阅 `List.AddEvent_Size`(元素尺寸变化事件) → 回调里 `relayout(true)`; `relayout` 增加 `lastW` 记录"上次算列宽时的元素宽度", 宽度与滚动条需求都没变就直接返回, 因此不会因 `SetColumnWidth` 引发的重复尺寸事件而震荡。组件自愈后, `FitColumns` 退化成"特殊时序下的强制重算", 调用方不必记得调。
    - **验收**: 修复前首屏"日期"列 = 20px(文本全 "..."), 修复后首屏列宽 = 剩余宽度(表头"日期"与 `2026-10-01` 完整); 拉窗/换页/切主题三种路径像素一致。

61. **自绘组件的"宿主底色 + 组级半透明"合成规则(暗色禁用/未选态踩过的坑)**(checkbox/radio/switch/input):
    - **现象(实测)**: 暗色下 ①未勾选的复选框是**一块比卡片更黑**的方块(亮色下看不出来); ②复选框**禁用态**的框线比参考实现亮一档; ③开关**禁用态**的滑块还是一颗扎眼的全亮白圆; ④禁用输入框是一整块**发亮的灰饼**。
    - **根因**: ①未选态在 shadcn 是"亮色无底色 / 暗色 `dark:bg-input/30`", 我们却铺了 `Background`(#0a0a0a), 比卡片(#171717)还深; ②③④ 禁用态错用 `WithAlpha(128)` **覆盖**透明度 —— 暗色 `input`/`border` 令牌本身就是"白 15% 半透明", 覆盖成 50% 等于画了一条亮白线/一坨亮白饼; 滑块/勾号则干脆没跟着淡。
    - **修法(三条一起立规矩)**:
      1. **先铺宿主底色**: 组件按"放在 `Card` 上"设计, `onPaint` 第一步 `p.FillRound(&p.Rc, t.Card, 0)` —— 既擦掉上一帧(透明元素重绘不擦底, 见 28:焦点环消失后的灰环、上一个状态的实心形状都会留在画布上), 又给半透明令牌一个正确的底。
      2. **未选/未勾的底**: 亮色不铺(露卡片), 暗色才叠 `t.Input.ScaleAlpha(0.30)`(= `dark:bg-input/30`);开关轨道暗色未选是 `t.Input.ScaleAlpha(0.8)`(= `dark:data-unchecked:bg-input/80`)。
      3. **禁用 = 每个图层各乘 50%**(`ScaleAlpha(0.5)`, 不是 `WithAlpha`): 不透明色乘完再叠到宿主上就等价于"整组半透明合成", 半透明令牌乘完也不会被拉亮。**形状**全部跟随(框/勾/轨道/滑块/圆点);**原生文字**不吃 alpha, 保持原色(见第四节末汇总表), 输入框禁用文字则改成"前景色 × 卡片色对半混"预混后下发(`Input.syncTextColor`)。
    - **实测(暗色, 与 shadcn 逐像素核对; 基底 `card=#171717`)**: 未选框底 `#212121`(白 4.5% 叠卡)/框线 `#424242`(白 15% 叠在框底上);禁用未选底 `#1c1c1c`/线 `#2d2d2d`(各再乘 50%);禁用已勾框 `#7e7e7e`(白 90% 实色乘 50%)+ 勾 `#4a4a4a`;开关禁用滑块 `#8f8f8f`(白 98% 乘 50% 叠轨道)。全部与"整组 50% 合成"的理论值一致。
    - **回归验收手段(重要)**: 程序化改状态**不会自动重绘**(原生 `SetCheck` 不失效重绘, 所以要 `Checkbox.SetChecked` 显式 `Redraw`); 环境里的合成点击对原生 Button 不可靠, 因此"切换前 vs 切换后"用**探针**验证 —— 探针实测: `XUI_STATE=before|after|target` 三态(`after` = 建好 before 后 1.5s 用元素定时器切成 target), 抓图后与 `target` **逐像素比对必须 0 差异**(底盘内部区域), 差异为 0 即无残留。亮/暗两轮都跑过。

62. **hover 过渡的"最后一帧"不再触发重绘 —— 二值可见物(下划线)淡出会永久残留**(`ui/separator.go` Badge Link):
    - **现象(实测)**: Link 徽章鼠标悬浮后移开, 下划线留在画布上不消失; Button Link 无此问题。
    - **根因(两条叠加)**: ①徽章 link/ghost 变体静止态 `bg=None`,`FillRound` 跳过 —— 透明元素重绘不擦底(见 28), 上一帧像素留在画布; ②过渡内核 `hoverTick` 里当某一帧的值**恰好到达目标**时该帧不重绘(`v == target` 时 `active=false`), 颜色类过渡无所谓(倒数第二帧 v≈0.02 与静止色视觉相同), 但"下划线"是 `hv>0` 就全色画的**二值可见物** —— 倒数第二帧(v≈0.02)的全色下划线被画上去后再也没人擦。
    - **修法**: ①`onPaint` 开头铺宿主底色 `t.Card`(同 61);②下划线透明度随过渡值缩放 `fore.ScaleAlpha(hv)` —— 倒数第二帧 alpha≈2, 肉眼不可见; 顺带淡入淡出也更平滑(shadcn 的 underline 本身不参与 transition, 但 GUI 侧缩放更接近"整体过渡"的观感)。**推论: 凡以过渡值为开关条件的可见物, 透明度一律跟 `hv` 走, 不要只判 `hv>0`。**
63. **开着对话框最小化再还原 = 整窗元素绘制/命中全灭, 只能"最小化前挂起、还原后重显"规避**(`ui/dialog.go`):
    - **现象(实测)**: 弹窗打开后点任务栏最小化, 再还原 —— 弹窗消失(只剩 backdrop 的模糊快照), 点遮罩/页面全无反应。
    - **诊断(探针 +示例程序逐层排查)**: 还原后元素树状态**全部正常**(`IsShow=true`、alpha=255、矩形正确、z 序 overlay>backdrop>页面), 绘制回调**照常触发**且 `p.Rc` 正确 —— 但绘制输出不上屏、鼠标事件不到达任何元素(含页面原生控件); 无弹窗可见时还原完全正常。结论是 xcgui 异型阴影窗口(`Window_Transparent_Shadow`)还原后合成/命中管线对"对话框遮罩子树可见期间"的状态损坏, **引擎级, 组件层无法事后根治**。
    - **试过无效的事后修复**(全部实测): 还原时重抬 z 序 + backdrop 压回 overlay 之下、`XWnd_Redraw(true)` 整窗重绘、重新显隐整棵子树、延迟到消息循环里整套重物化(重拍快照 + 重新 Show) —— 全部无效; 在 WM_ACTIVATE 里直接操作元素更会把整窗合成打断(全窗白屏)。
    - **修法 = 挂起/恢复**(对话框状态保留, 不触发 onClose): `AddEvent_WindProc` 拦 `WM_SIZE` —— `wParam==SIZE_MINIMIZED(1)` 时**同步**隐藏整组(清焦点 + `showEleTree(false)` + backdrop 隐藏; 不能走 Hide, 其淡出完成回调是异步的)置 suspended; `wParam==SIZE_RESTORED(0)/SIZE_MAXIMIZED(2)` 且 suspended 时用 `app.AfterFunc(0, ...)`(= `time.AfterFunc` + `XC_CallUT`, 回调被投递到 UI 线程执行)做 `materialize` + `SetSubtreeAlpha(1)` 无动画直接显示。引擎全程只见"隐藏状态的还原", 与已验证可靠的路径一致。
    - **三个时序/机制坑**: ①`XE_SIZE` 在还原时不保证触发, 程序化最小化**不发 WM_ACTIVATE(WA_INACTIVE)** —— 抓"最小化"只能拦 `WM_SIZE/SIZE_MINIMIZED`; ②WM_ACTIVATE/WM_SIZE 里直接操作元素会把还原流程的合成打断(实测全窗白屏), 恢复动作必须延迟出消息处理(用 `XC_CallUT` 类机制投递到消息循环); ③**炫彩定时器是独立命名空间**(排查过程中踩到): `AddEvent_XC_Timer` 只接收 `XWnd_SetXCTimer` 启动的定时器, 用 Win32 的 `XWnd_SetTimer` 启动的话回调**永远不会被派发**(与 anim.go 的 animTimerID 机制同源, 自用 ID 避开 0x7875) —— 最终修法用 AfterFunc 避开了这套机制; ④**还原后立即启动的 Presence 过渡帧不会被驱动**(整组 alpha 卡 0, SetSubtreeAlpha(0) 之后全组隐身) —— 恢复显示必须无动画, 之后用户再打开仍走正常淡入。

65. **COM 直通的参数传递(386 / amd64) + "浮点形参"的完整解法**(`wapi/com` 与 `d2d`探针实测):

    - **386**(stdcall, 参数全部走栈): 结构体按值传要**拆成 32 位槽**逐个压栈; float64 用 `common.Float64ToUint32Pair` 拆 low/high 两个槽 —— **符合 ABI** 的写法(edge 的 `edge_386.go` 正是如此)。
    - **amd64**(Microsoft x64): 8 字节以上的结构体**按引用传**(要传指针); 浮点走 XMM0~XMM3, 与通用寄存器是**两套独立**的 —— **SyscallN 直接传(哪怕位打包)无效**: `ID2D1RenderTarget::SetDpi(float,float)` 位打包**直接崩**(`Exception 0xc0000005`, 现场 `rax=r9=0x42C00000` 正是 96.0f 的位模式)。
    - **解法 = `wapi/com` 的汇编转发器**: `CallWithFloat32` / `CallWithFloat64` / `CallWithFloat32RetFloat`(读"按值返回 float"的方法)。三个坑都实测定死:
      ① **必须由 `syscall.SyscallN` 调用转发器** —— 借它的 `runtime.cgocall` 切到**系统栈**: Windows 的栈探测看 TEB 里的栈范围, 而 Go 的 goroutine 栈不在其中。直接在 goroutine 栈上 CALL: `GetTickCount` 能过, `user32.GetSystemMetrics` 崩(栈探测落到非法地址)。
      ② **转发器要自己摆正栈**: 调用点 RSP 16 字节对齐 + 32 字节 shadow space。实测这条路径入口 RSP ≡ 0 (mod 16) —— `SUBQ $32` 过、`SUBQ $40` 崩(用到 SSE 对齐存取的 API 立刻暴露)。
      ③ **浮点槽号要按接口实测**: 教科书说"整数/浮点各自独立计数"(`SetOpacity(FLOAT)` 的浮点应在 XMM0、this 走 RCX), 实测 `ID2D1SolidColorBrush::SetOpacity` 读的是 **XMM1** —— 逐个槽单独填 0.5, **只有 XMM1 生效**; 这类接口按"参数位置"编号(this 占位置 0)。
      ④ **第 5 个及以后的参数要压栈**(x64 约定: 超出 RCX/RDX/R8/R9 的放 `[RSP+32]` 起, 前 32 字节是 shadow space), **栈上的 float 占 8 字节槽、值放低 4 字节**: `wapi/com.CallWithFloat32Stack` 覆盖"4 个整数/指针 + 4 个浮点 + 最多 4 个栈参数"这个通用形态, 单测拿 `ReadProcessMemory` 的第 5 个参数(实际读到的字节数)验证它真的落在了栈上。实测用例: `DrawGeometry(geometry, brush, FLOAT w, style)` → 线宽 XMM3 / style 栈[0]; `DrawLine(p0, p1, brush, FLOAT w, style)`(5 个参数) → **线宽在栈[0]**(位置 4, 前 4 个 XMM 槽已被位置 0~3 占掉)/ style 栈[1]。若按"独立计数"喂(XMM0 + R9), 调用本身不崩(void 方法)但**整帧渲染被搞坏**(窗口全黑)。这条打通后, 图标描边从"CPU 展开 + 填充"换成了真描边(见 §43 末尾), 基本图元(直线/矩形/椭圆)与形状几何(矩形/圆角矩形/椭圆)也一并封装(探针一次抓图全对: 线宽/圆帽/位置/圆角都对)。
    - **验证数据**(探针): 回声桩(原样返回 XMM0)0.5/0.125 精确往返 ✓; 画刷 `SetColor/GetColor` 往返 0.25 ✓(对象正确); `SetOpacity`: 只填 XMM1=0.5 → **读回 0.500** ✓(填 XMM0/2/3 均无效); 反例 —— 该 RT 的 DPI 是 0/0(非法状态), 在它上面 `SetDpi` 怎么传都崩, **与传参无关**(验证浮点要用纯属性 setter, 别拿有状态的 RT 方法试)。
    - **两处认知纠正**: ①"COM 直通只支持 x64"是错的 —— 386 同样能跑, 只是参数传递写法不同(edge 两个架构各一套); ②`ID2D1Brush::GetOpacity` 是**按值返回 FLOAT**(在 XMM0), 不是 `(float *out)` 出参 —— 要读它得用 `CallWithFloat32RetFloat`。
    - **别把"按值 float"与"浮点结构指针"搞混**(本条最初就写错过): 受限的只有**按值传的** float64/float32 形参; 指向浮点结构的**指针**不受限, 照常 syscall 调用即可 —— `ID2D1RenderTarget::SetTransform` 的矩阵、`CreateSolidColorBrush` 的颜色都是这种, `d2d` 已封装 `SetTransform`/`GetTransform`(探针往返一致: 单位矩阵 → 平移 12,34 → 设回单位矩阵)。
    - **`SetTransform` 与炫彩绘制的相互作用(实测, 用它的前提)**: 炫彩的 `XDraw_*` 走的是**绘制当时的 RT 变换** —— 设了平移不恢复的话, 紧随其后的 XDraw 内容会一起被平移(探针 : 变换下画的绿色矩形右移了 150 逻辑像素, 恢复单位矩阵后画的蓝色矩形位置正确)。所以它可以**用来做"直通绘制的局部平移"**(图标位置外置即靠它, 见 43), 但**必须成对设置/恢复**。
    - 与 43(直通拿不到元素变换/裁剪)、47(整窗 alpha 与生命周期)同源: 都是 **SyscallN 这条通道的能力边界**, 而不是 xcgui 的缺陷。

66. **页面滚动容器 = 原生 `LayoutFrame`(布局框架)的实测语义**(`ui/scrollarea.go` +示例程序六页;探针 / 探针实测):
   - **布局框架 = 布局盒子 + 滚动视图**: 和其他容器一样流式排子项, 但**内容总大小按子项自动撑开**、溢出时自动出现纵向滚动条 —— 不需要 ScrollView 那套手动 `SetTotalSize`(示例 `Basic/ScrollView` 的用法), 页面滚动容器直接用它。
   - **子项尺寸语义**(内容轴是"按内容自撑", 与 55 的通用语义不同):
     - `Fixed` / `Auto` 拿到自然值(6×100 + Auto 高容器 866 对 视口 532 → 出滚动条, 内容不被压扁);
     - **`Fill` = 视口高**, 且它**内部**子项的溢出**不算进滚动范围**(8×100 的 Fixed 卡片挂在一个 Fill 高子项里, 662 高的框架 total 仍是 662 —— 滚不动), 只适合"铺满视口 + 内层 Weight 吃剩余"的形态(Fill 子项自己的高度是确定值, 内层 `[Fixed100][Weight1][Fixed100]` 实测拿到 442 ✓);
     - **`Weight` 直接挂在框架上恒为 0**(内容轴不确定); 内层再写 Weight 也是 0;
     - **`EnableOverflowHide` 不是"裁剪显示"、是"布局时把超出视口的子项高度裁掉"**: 400 高的子项被压成 10(实测 400→10)。滚动页面要的是前者, 而**视口裁剪 ScrollView 自带**(滚出视口的子项本来就不画出来), 所以这个开关**不要开** —— 开了卡片会被切短。
   - **首帧(以及滚动条显隐翻转的那一帧)内容宽度会少一个滚动条宽**: 视口 772 时 Fill 子项只有 760(右侧多 12px 空白), 原因疑似该趟排版先按"无滚动条"排子项、随后视口才被滚动条占去; **再排一次就正确**(`AdjustLayout`)。`ui.ScrollArea` 用元素尺寸事件 + (frame宽, 视口宽) 闸门自愈一次, 调用方不必管(窗口 resize 导致滚动条显隐翻转时会再自愈一次)。
   - **滚轮**: 悬停在**子元素上**也会冒泡到框架并滚动(子元素的 `XE_MOUSEWHEEL` 默认禁用, "禁用则事件传给父元素"); 悬停原生控件(如 List)会被它自己吃掉 —— 嵌套滚动的正常行为。注入真实滚轮时**窗口必须在后台有焦点**(工具进程还要 per-monitor DPI-aware, 否则坐标虚拟化)。
   - **滚动条滑块拖动: 合成输入下未复现**(探针实测: 裸 LayoutFrame 与 `ui.ScrollArea` 的滑块都拖不动, 而同一脚本拖**表格 List 的**(同款皮肤)能滚 —— 不是组件引入的差异, 疑与 ScrollView 自带滚动条的命中/拖拽实现有关, 未深究); 滚轮、代码滚动、点选/输入等常用路径都正常。真实鼠标下是否可拖未验证。
   - **代码滚动**: `ScrollPosYV(像素)` 是像素语义(滚到 100 就是 100); **`ScrollPosV(pos)` 的参数是"行"单位**(默认 20px/行, `GetScrollBarVObj.GetRange` 也按行算) —— 拿像素喂它会直接滚到底(实测 100 → range 17 全用掉)。
   - **滚动条几何/配色/显隐**(统一规格见 `ui/scrollbar.go`): 元素占宽 = `SetScrollBarSize` 的值(10 就是 10), `SetSliderPadding(1)` 把**滑块**在轨道里缩成 8px 宽(`w-2.5` + `p-px` 的 shadcn 规格); 轨道要真透明得 `ClearBkInfo` 后垫一层 alpha=0 的填充(否则回退系统默认内容, 同 30); 圆角 / 两级配色 / 悬停显隐的完整机制见第 21 条。Table 的自适应列按 14 让宽(`tableSBW`), 多行 Input 在 `applyTheme` 里刷这条皮肤。⚠️ **"自动显隐"模式下滚动条的 `XEle_GetRect` 返回的是隐藏位置**(实测: 表格滚动条 raw x = -12..-2, 而渲染/命中在内容区右缘)—— 别拿它做命中或定位计算; 判定鼠标是否在滚动条上要用 `XWnd_GetStayEle`(渲染与命中的坐标系一致)。
   - **容器被点一下会聚焦并画原生蓝框**(#58B1FC, 同 34) → `EnableFocus(false)` + `EnableDrawBorder(false)`。
   - **内容区的外边距计入滚动范围**: 内容区四边各 24 外边距时 total = 24 + 内容高 + 24 —— 页面底部留白因此天然有效(滚到底正好留 24, 不必额外垫占位元素)。
   - **Auto 内容里的栅格写法**(探针实测): 行高必须 **Auto**(行高写 Weight/Fill 在 Auto 内容里塌成 0 ——示例程序首次接入时两列卡片整段消失就是这个), 行里 **`Weight` 宽的列仍然算得出宽度**(352 宽的行 → 201/135 = 3:2 ✓), 列高 Auto、卡片 Auto —— 就是 CSS flex 的直觉写法。**铺满型页面(数据页)反过来**: 内容区 `SetContentFill`(Fill) + 行 `Weight` + 列/卡片 Weight 整链 Weight, 表格/树才会跟随窗口长高, 代价是页面不再滚动。
   - 与 55/56/59(尺寸语义、Card、Auto 高的坑)同源; 该条同时解释了"页面内容比窗口高被 `EnableOverflowHide` 裁掉"那个问题的解法。

67. **自绘标题栏的"最大化/还原"按钮:用 `Button_Type_Max`, 别自己调 `MaxWindow`**(demo/main.go):

    - `SetTypeEx(xcc.Button_Type_Max)` 的按钮**会自己切换**最大化/还原(实测: 默认尺寸点一次 → `IsZoomed=true`; 最大化状态再点同位置 → `IsZoomed=false`, 窗口回到原矩形), 与 39 里 `MaxWindow` API\"只管最大化、不切换\"不同 —— 双击/按钮两条路径各用各的, 不要互相替代。
    - 按钮顺序仍按视觉排: 最小化 → 最大化 → 关闭, 尺寸与 Ghost 样式同其他控制按钮(`ButtonSizeIconSm` 那套)。
    - **同一排图标要按"墨迹占比"配平, 而不是统一画布**: `IconEye`(2..22)、`IconMoon`/`IconMaximize2`/`IconMinimize2`(3..21)在 18px 画布下墨迹 23~25 物理像素; `IconMinus`(5..19)、`IconX`(6..18)同画布只有 17/15, 看起来"最小化/关闭小了一圈"。**保持最大化/主题图标默认 18, 把 minus/× 的画布放大到 22**(`WithIconSize(22)`): 实测 eye 25×17、moon 23×23、maximize2 23×23、minus 21×3、× 19×19, 一排视觉齐平。24 也齐(24×2/20×20)但描边同比偏粗, 取 22。反向"缩小 maximize 到 14"同样齐(17/17/15), 但最大化会比旁边的主题按钮小, 不取。
    - **自绘 `WithIcon` 的按钮不会随窗口状态换图标**(组件每帧只画 `b.icon`), 最大化后仍是原图标, 需要自己换: 窗口没有 MAXIMIZE/RESTORE 事件, 拦 `AddEvent_WindProc` 的 `WM_SIZE`(`wParam==SIZE_MAXIMIZED/SIZE_RESTORED`), 用 `a.UI` 推迟到消息处理完之后按 `w.IsMaxWindow` 调 `SetIcon(icons.IconMinimize2/icons.IconMaximize2)` —— 这条路径同时覆盖按钮、双击标题栏、Win+↑ 和系统菜单; `iconSize` 不随图标变, 两个图的墨迹比例相近, 不用再调。
    - 最大化后标题栏按钮整体贴屏幕右上角(窗口矩形 = 屏幕), 元素坐标随布局重排 —— 用固定坐标做 UI 自动化时, 两种情况要分别取按钮中心。

68. **原生 ComboBox 默认带中文字间距, 弹层 ShapeText 没有 → 打开下拉时文字会"变紧"**(`ui/combobox.go`):

    - 实测(150% DPI, 14px 主题字体, 文本"测试环境"): 原生 ComboBox 墨迹宽 **91 物理像素**; 同一字体句柄下 ShapeText 只有 **87**。逐字对比单字墨迹(测 20/20、国 19/18)证明**不是字号/字形缩放, 而是每个字多出约 1px 的 advance** —— xcgui 给原生 ComboBox 的中文默认加了字间距。
    - 触发区用 ShapeText 弹层、原生控件又带字间距, 结果就是"点开下拉瞬间文字突然变紧"(用户反馈的"下拉列表文字间距小")。调条目左边距无法解决, 只会让左缘对齐而宽度仍不一致。
    - **`xc.XEdit_SetCharSpaceSize(comboHandle, 0, 0)` 对 ComboBox 句柄同样有效**(`EnableEdit(false)` 也生效): 触发区 91→**87**, 与条目/ShapeText 完全一致; 传 `-1,-1` 则收紧到 82(负间距)。
    - 修法写在 `NewComboBox` 里(设完字体之后调用一次); 修后同一弹层内"触发区文字"与每个条目的左缘、墨迹宽度三个数全部相同(1336..1422)。

69. **ComboBox 弹层进场淡入: 原生弹层能转异型透明窗, 且 alpha 可以连续改**(`ui/combobox.go`探针 + 探针实测):
    - **时机**: `XE_COMBOBOX_POPUP_LIST` 触发时弹层窗口**尚未显示**(探针实测 `IsWindowVisible=false`, 窗口矩形/客户区已就绪), 透明类型与 alpha 都能在首帧前设好 —— 不会闪白。
    - **普通窗口上 alpha 无效**: 不换类型直接 `XWnd_SetTransparentAlpha(hWindow, 128)`, 抓图仍全白(纯白 popover 一点没透)。必须先 `XWnd_SetTransparentType(hWindow, Window_Transparent_Shaped)` —— 它在弹出后(未显示时)切换也生效。
    - **与 57 的 tooltip 结论不同, 别一概而论**: 这个弹层转 Shaped 之后**连续改 alpha 每帧都生效**(`shaped-seq`: 设 64 → 抓图 #40A0FF、160 → #A0D0FF、255 → #FFFFFF, 背景是 #0080FF 的父窗)。57 的"整窗 alpha 只认第一次调用"是 tooltip 那个窗口的具体问题(它自己还要靠元素 alpha 做动画)。
    - **转类型不影响区域裁剪的圆角**: `SetWindowRound` 先设、再转 Shaped, 圆角外仍是背景色(角落采样 = 父窗蓝); ListBox 的 BkM 底色、条目 DrawItem、`XE_PAINT_END` 的 ring 全部照常输出。
    - **动画实测**(`probefadecomp` 每 16ms 抓弹层中心): `#0180FF`(下限) → `#8CC6FF` → `#C6E3FF` → `#F6FBFF` → `#FFFFFF`, 约 5 帧到全白, 即 100ms `ease` 的淡入曲线; 角落全程是背景蓝。
    - **实现要点**: 过渡挂**宿主窗口**(`XWidget_GetHWINDOW`)而不是弹层小窗(见 47); `apply` 每帧先 `XC_IsHXCGUI(hw, xcc.XC_WINDOW)` 守卫 —— 弹层会在点外部/选中条目时立刻销毁, 用失效句柄调 X API 会静默崩进程(与 64 的菜单淡入同一个坑); 每次弹出用 **`xui.NextPresenceKey`** 取键(Presence 对目标不变的调用幂等, 键复用会让过渡根本不启动, 见 70)。

70. **Presence 的键必须全局唯一: 撞键 = 新弹层"全透明但仍在", 表现为点击没反应**(`presence.go` + `menu.go` / `combobox.go` / `dialog.go`; 实测复现并修复):
    - **现象**: 表单页选过一次 sm 下拉框 → 切到导航页点"打开菜单" → **第一次点击毫无反应**, 第二次才弹出; 反过来先开菜单、再点 sm 下拉框, 下拉框也"点了没反应"。点同页其它控件(分页)同样失效一次。
    - **不是点击被吞**: 给按钮的 `AddEvent_BnClick` 加日志, 日志**照常触发** —— 菜单确实弹了, 只是窗口 alpha 停在 `AlphaFloor`(看不见), 而原生菜单一弹出就是模态的, 于是下一次点击被这个"透明的菜单"吃掉(选中/点外部才关闭)。
    - **根因(键撞车)**: `Presence` 按 `(hOwner, key)` 记过渡值且**幂等** —— 目标值没变就不重启过渡、**也不再调 apply**; 而 `target=1` 的条目**不回收**(只有淡出到 0 才删除)。menu 与 combobox 之前各自维护 `menuPresenceKey++` / `comboPresenceKey++`, **都从 1 开始、都挂在宿主窗口上**: 先开下拉(key=1, 走到 1 后条目留在表里) → 再开菜单(key=1) → 撞上已完成的条目, 动画不启动、apply 不执行 → 菜单窗停在 popup 回调里刚设的 alpha 下限。反向同理。
    - **修法**: ① 新增 `xui.NextPresenceKey`(全局原子递增), menu / combobox 改用它, 对话框也从固定键 0 改成按实例分配(`Dialog.fadeKey`) —— **任何新调用方都不许自起计数器**; ② `Presence` 在返回前**立即用当前进度调一次 apply**: 键一旦撞上已完成的过渡, 最坏也只是"没有动画", 目标状态一定落地, 不会留下透明窗口(防呆)。
    - **排查要点**: 这类"点了没反应"先确认事件到没到(BN_CLICK 日志); 到了就怀疑"弹窗存在但不可见"。`SetAnimationScale` 放慢也看不到 —— 过渡根本没启动, 不是太快。
    - **同源**: 47(弹层透明度/整窗 alpha 的约束)、64(菜单淡入)、69(下拉弹层淡入)。

71. **异型/阴影弹窗里, 元素的布局坐标原点是窗口矩形原点(含阴影边距), 不是可见区(卡片)原点**(`ui/popup.go`探针实测):
    - `Window_Transparent_Shadow` 的窗口, `XWnd_GetClientRect` 返回的是含偏移的 Body(实测 menu: `{8,8,168,276}`, 即卡片在窗口坐标 (8,8) 起 —— 与 `XEle_GetWndClientRect` 的注释"原点 = 所属窗口矩形原点(含阴影边距)"同口径)。但**元素布局不吃这个收缩**: 元素放 (0,0) 会贴在窗口矩形左上(阴影带内), 卡片右侧与底部露出 band 宽的窗口底色 —— **浅色主题下完全看不出来**(窗口默认白底、卡片也是白), 深色主题下是一条白边(实测截图)。
    - 修法: 弹层内容元素建在 `(band, band)`(band = 阴影带宽), 尺寸 = 卡片尺寸; 窗口尺寸 = 卡片 + 2×band。定位同理: 窗口矩形左上 = 卡片目标位置 − band(物理像素要乘 DPI, 见 66 的坐标系说明)。
    - 与 36(菜单阴影窗)/47/64 同一套窗口, 菜单自己画背景用的是窗口绘制事件(rc = GetClientRect, 原点在窗口原点), 所以它看起来"没问题" —— 两条绘制路径的坐标系**差一个 band**, 混用时最容易在这翻车。

72. **手动创建原生 ListBox 做自绘列表(时间框弹层)的实测语义**(`ui/timepicker.go`探针实测):
    - **必须先 `CreateAdapter` 再 `AddItemText`**, 否则条目装不进去(同 ComboBox 的适配器语义)。
    - **几何**: `SetPadding` 生效(内容区四边内缩); **行步进 = 项高 + 1**(项与项之间固定 1px 缝, 与 combobox 弹层实测一致); **首项 top = padding + 3**(固有内缩, ScrollPosYV 定位要把它算上, 少 7px 目标项会露不全)。`SetItemHeightDefault(h, h)` 设默认行高。
    - **`ScrollPosYV` 是像素**(同 66); `SetSelectItem` 只改选中态**不滚动视口** —— 打开弹层时要手动滚到当前项。
    - **项模板的文字是"绘制前静态"的**: 在 `XE_LISTBOX_DRAWITEM` 里对模板 ShapeText `SetTextColor` 改选中项文字色**来不及**(文字已在 DrawItem 之前画完, 实测黑底上黑字)。要按状态变文字色只能**自绘文字**: 把模板文字设成全透明(`XShapeText_SetTextColor(h, 0)`), DrawItem 里 `p.Text(GetItemText(i, 0), &pItem.RcItem, col, Center|Vcenter)` 自己画。
    - **两套索引**: `GetItemText(i, col)` 的列从 **0**(适配器列)起; `GetTemplateObject(i, itemID)` 的参数是模板 XML 里的 **itemID**(默认模板 shapeText = 1)。混用会拿到空串/空句柄。
    - **选中态在 `pItem.NState`**(`List_Item_State_Leave/Stay/Select`), 悬停过渡用 `xui.HoverItem(hEle, pItem.Index, hot)`; 项背景画在 `pItem.RcItem`(已扣 padding, 不含滚动条宽度)。
    - `SetDrawItemBkFlags(0)` 关原生项背景 + `EnableBkTransparent(true)` 关容器背景; 滚动条走 `scrollbar.go` 的统一皮肤(`SetScrollBarSize` + `styleScrollBar`), `EnableAutoShowScrollBar(true)` 原生自动显隐。
    - `XWidget_LayoutItem_SetWidth(模板文字, Layout_Size_Fill)` 会把项模板文字宽算成 0(实测文字整体消失) —— 项模板里的 `layout.width="auto"` 别动, 居中交给自绘。
    - **元素鼠标事件的 pPt 是"元素客户区"坐标**(探针 : 元素在 (100,100) 时点元素内 (99,47) 的 raw = 99,47; `XEle_PointWndClientToEleClient` 是窗口客户区 → 元素客户区的**再次**换算, 对事件坐标再用会多减一次)。自绘组件的命中测试直接用 pPt 即可。

73. **复用的 Shadow 弹窗做淡入: 整窗 alpha 冻结 / 窗口白底 / apply 互搏 / 过渡挂错窗口 / 首帧残留 / DLL 无效句柄 —— 五个坑叠一起才是"闪烁、抖动、闪退"**(`ui/popup.go` + `anim.go` + `presence.go`, 实测四轮递进 + 60fps 录屏逐帧确认;探针 / 探针实测 / 弹窗内逐帧抓图):
    - **现象演进**: ①"弹出动画闪烁" → ②"第一次不闪, 第二次起闪" → ③修复后仍闪 → ④"点击后弹窗直接显示, 然后消失, 然后开始淡入"(用户逐帧) + "时间框点击闪退"。帧级抓图(弹窗进程内每 8ms BitBlt 整个弹窗矩形存 PNG)实锤: 第二次显示的第 1-2 帧 mean=228(**白帧**)。
    - **根因 C2(快速点击 UI 卡顿, "抖动")**: 过渡挂宿主主窗口 → hoverTick 每 tick 全量重绘整个表单页 ×100ms, 快速点击时重绘排队、UI 线程卡到出系统忙碌光标(用户 60fps 录屏可见蓝圈)。**修法**: 过渡挂**弹窗自己**(p.win) —— 元素 alpha 由弹窗自身重绘落地, 400×500 物理像素每 tick 无压力; 整窗 alpha 时代挂宿主窗是被迫的(DWM 合成要宿主带动, 见 47/73 根因 B), 元素 alpha 不需要。
    - **根因 A(主因, 白闪)**: 弹窗窗口不设置背景时, xcgui 画**白色默认底**。淡入(元素 alpha)期间卡片半透明, 白底从卡片下面透出来 —— 深色主题下就是刺眼的白闪。**修法**: 弹窗创建时给窗口 BkM 垫一层 `alpha=0` 的填充(`XBkM_Clear + XBkM_AddFill(..., 0, 0)`) —— 留空会回退"系统默认内容", 透明必须显式表达, 同第八节 30 的轨道透明。
    - **根因 B(淡入路线错误)**: 整窗 alpha(`XWnd_SetTransparentAlpha`)对透明窗**只认第一次调用**(57) —— 创建时的 255 就是那次, 之后全部冻结。menu/combobox 的弹窗淡入能用是因为**每次弹出都重建窗口**(每次都是"第一次调用"), 而组件弹层是复用的 → 整窗 alpha 路线整段失效(逐帧抓图证实两次显示都是瞬间全亮)。**修法**: 淡入改**元素 alpha**(`SetSubtreeAlpha(body)` + 每帧 `XWnd_Redraw(弹窗)`, tooltip 同款, 见 57); 显示前压 `SetSubtreeAlpha(0)`, 首帧只有阴影。附带修正: `SetWindowPos` 不带 `SWP_SHOWWINDOW`(提前显示会多一帧)。
    - **根因 C(第二次起 apply 互搏)**: `hoverTick` 对 owner 上所有 presence 条目调 apply, 已到位旧条目(不回收)每帧 `apply(v=1.0)` 与新条目打架 —— 实测二次弹出时亮度 12↔85 逐帧交替。**修法**(anim.go): 到位条目(settled)不再调 apply("到位后条目静默"补上实现; 反向过渡 settled 会重置, 淡出不受影响)。menu/combobox/dialog 共用这张表, 同样受益。
    - **排查方法论**: ①亮度采样在"浅色弹层弹在浅色页面/弹层内容与页面内容相同"时完全失效, 必须**帧级抓图看画面**; ②外部工具(BitBlt 轮询)有 10ms+ 延迟且可能错过首帧, 首帧问题要用**实测逐帧**或进程内自抓(Show 里起协程); ③录屏分辨率是缩放的(2560→1728), 且 IDE 全屏会挡住采样区域 —— 坐标要先按视频实际内容校准; ④apply 日志显示 v 正常推进 ≠ 画面正常(白帧期间 v 也在走), 一定要把"值序列"与"帧序列"分开对齐; ⑤**DLL 错误弹窗 ≠ Go panic** —— 用户看到 DLL 报错弹窗(如"SetAlpha 输入句柄无效")时, 先定位是哪个 X API 收了无效句柄, 再查该句柄的生命周期(所属窗口显示前/销毁后); 被自动关闭的错误弹窗背后可能跟着 access violation 闪退。
    - **根因 D(时间框点击闪退 + "SetAlpha 句柄无效"弹窗)**: `GetChildByIndex` 对原生 ListBox 会返回**非 0 但无效**的子句柄(实测 type=62 的延迟创建内部对象, 弹窗已显示后依然如此, 每个子 ListBox 约 8-16 个) —— 对它调 `XEle_SetAlpha` 触发 xcgui 的 **DLL 错误弹窗("SetAlpha 输入句柄无效")**。⚠️ DLL 内部错误**不产生 Go panic** —— 它走 xcgui 自己的错误处理(弹窗/日志/返回),示例程序里被"自动关错误弹窗"逻辑吞掉后, 后续无效句柄调用可能直接 access violation 闪退(无堆栈)。**修法三层**: ①遍历组件(`popup.go` 的 `fadeSubtreeAlpha` 与 `presence.go` 的 `setAlphaTree`)对每个句柄加 `XC_IsHXCGUI(h, XC_ELE)` 守卫; ②`fadeSubtreeAlpha` 同时跳过滚动条子树(滚动条显隐由 showScrollBarOnHover 自理, 跟随淡入会被每帧拉亮); ③Show 时序改**屏幕外预显示**(见根因 E)。
    - **根因 E(首帧残留, 用户逐帧确认"直接显示 → 消失 → 再淡入")**: 复用窗口再次 ShowWindow 时, 首帧合成的是**窗口表面残留的上一次内容**(关闭前 alpha=1 的全亮卡片); 元素 alpha=0 要等第一次重绘才落地, 于是先闪一帧全亮。**修法**: **屏幕外预显示** —— 先 `SetWindowPos(-30000,-30000, SWP_SHOWWINDOW)` + `Redraw(true)` 在不可见位置画好"透明首帧", 再 `SetWindowPos` 挪到目标位置(挪动只改 DWM 合成位置, 表面内容已是透明帧; Win32 窗口坐标范围 ±32768, -30000 安全)。60fps 录屏逐帧验证: 弹层出现 = 19→24→27→31 纯渐显, 无全亮相。
    - **交互补充**: 关闭后 300ms 内重开(`popoverReopenFadeSkip`)**跳过淡入直接完成态** —— 快速连点时弹层反复从透明渐显会"眨眼", 对齐 CSS 过渡的中断语义(不重播动画)。
    - **同源**: 57(tooltip: 异型窗整窗 alpha 只认第一次 → 元素 alpha)、64/69(menu/combobox 每次重建弹窗所以整窗 alpha 可用)、70(presence 键唯一 + 根因 C 的条目静默)、30(BkM 留空回退白底)、47(整窗 alpha 时代挂宿主窗的原因)。

74. **弹层夹边界必须用"显示器工作区", 不是屏幕高度**(`ui/popup.go`; 现象: "窗口最大化 + 页面滚到末尾, 点日期框后月历卡片底部被任务栏盖住"):
    - **现象**: 触发区贴着任务栏时, 弹层按"屏幕高度"判断下方放得下(y+ch <= SM_CYSCREEN), 于是照常向下弹, 卡片下半截被任务栏盖住(任务栏永远在顶层, 弹层是 topmost 也没用)。
    - **修法**: `wapi.MonitorFromWindow(hWnd, MONITOR_DEFAULTTONEAREST)` + `wapi.GetMonitorInfo` 取 `RcWork`(已扣任务栏/停靠栏), 用逻辑坐标夹边界: 下方放不下 → 翻上方; 上方也放不下 → 贴 `RcWork.Bottom`; 水平夹在 `RcWork.Left/Right`。
    - **坑**: ①`rcWork` 是**物理像素**, 而 xcgui 的窗口/元素矩形是逻辑像素 —— 除以 DPI 比例再夹(本库 `XWnd_GetDPI`); ②`SM_CYSCREEN` 只是主屏高度, 副屏/多屏应取"窗口所在显示器"的工作区(本条实现), 同 `SM_CXVIRTUALSCREEN` 与 `SM_CYSCREEN` 混用也是错的; ③`MONITORINFO` 的 `cbSize` 由 `wapi.GetMonitorInfo` 内部填好, 调用方不用管。
    - **坑(第二次修复的直接原因)**: `MonitorFromWindow` 要的是 **Win32 HWND**, 而 `xc.XWidget_GetHWINDOW(hAnchor)` 返回的是 **xcgui 窗口句柄** —— 喂进去 `MonitorFromWindow` 返回 0, 代码静默走"整屏高度"兜底分支, 于是任务栏照样盖住弹层(第一次修完看起来好了, 只是因为那一轮探针的窗口恰好离底部足够近, 靠整屏高度也能触发翻上)。**修法**: `xc.XWnd_GetHWND(hWnd)` 转换后再查显示器; `monitorWorkArea` 里再兜一层"句柄无效 → 用 `GetDesktopWindow` 查", 保证无论如何拿到的都是工作区而不是整屏。
    - **上游化**: 这些 Win32 声明原先散落在 xui(`ui/backdrop.go` 的 `modUser32` + `ui/popup.go` 的 `MonitorFromWindow/GetMonitorInfoW`), 现已全部收进 xcgui 的 `wapi`: `GetWindowDC / ReleaseDC / PrintWindow / MonitorFromWindow / GetMonitorInfo` 与 `PW_` / `MONITOR_` / `MONITORINFO`(`wapi/types`); gdi 侧本来就有 `gdi.CreateCompatibleDC / CreateDIBSection / SelectObject / DeleteObject / DeleteDC`。xui 里**不再有任何裸 `syscall.NewLazyDLL`**。

75. **通知(Toast)用窗口内单元素自绘, 悬停暂停走窗口级 GetStayEle, 位置动画必须整窗重绘**(`ui/toast.go`,示例程序反馈页逐项点击 + 亮/暗抓图):
    - **载体**: 通知要贴着主窗口右下角、跟随窗口移动/缩放/最小化 —— 用**窗口内的覆盖元素**(每条通知一个 `widget.NewElement`)而不是独立置顶小窗, 这些跟随都由框架负责。窗口开着布局(`EnableLayout(true)`)时, 未声明 `LayoutItem_*` 的子元素**不会被 AdjustLayout 重排**, 手动 `XEle_SetRect(…, AdjustLayout_No)` 的位置会保留(与 Dialog 的全窗遮罩同一条路径, 实测)。
    - **一条通知 = 一个元素整体自绘**(图标/标题/描述/动作按钮/关闭按钮全画在它里面, 不建子元素): 悬停不会被内部控件打断, 命中判定只是"卡片局部坐标 + 一次矩形比较"; 若用原生 Button 做关闭钮, 鼠标移到子元素上会让父元素收到 MouseLeave(见 21), 悬停暂停会莫名其妙地恢复。
    - **悬停暂停必须用窗口级 `MouseMove` + `XWnd_GetStayEle`**: 元素级 MouseStay/MouseLeave 不可靠(21), 而且这里要的是"鼠标在这组通知的任意一条上"的**整组语义**；单条元素判定在两条卡片之间移动时会反复 leave/enter。窗口尺寸事件同理走 `AddEvent_Size` 重排。
    - **倒计时**: 每条通知在自己的元素上挂炫彩定时器(ID 用 `0x7876`, 避开 anim 的 `0x7875`、Tree 的 `0x7874`), 100ms 一帧; 悬停时只是不递减(`paused`)。为什么不用窗口级定时器: 同一窗口可能有多个 Toaster, `XWnd_SetXCTimer` 同一 ID 只能有一个定时器, 会互相覆盖。
    - **定位**: 用 `XWnd_GetBodyRect` 取内容区右下角 —— 阴影窗口(`Window_Transparent_Shadow`)的 Body 比整窗小一圈 band(71), 而**元素坐标原点是窗口原点(含阴影带)**, 不要再叠加. 元素比卡片四周外扩 6px(投影留白), 因为元素绘制被裁剪在自身客户区内。
    - **布局坐标一律用"卡片矩形", 元素矩形由设置函数外扩**: 元素 = 卡片 + 四周 6px 投影留白。若把 `it.h`(按 padding+内容算出的卡片高)直接当元素高, 卡片可见高度就少 2×6 = 12 —— 上下内边距各被吃掉 6px, 表现为"文字顶到卡片底、上留白大下留白小"(最初的报障现象)。修法: 排版/布局全按卡片坐标算, `toastSetRect(h, x, y, w, h)` 内部统一减 6 并加 12; 实测修复后两行卡片 topGap=botGap=28 物理像素(@150% DPI), 单行卡片 26/27。
    - **`GetTextShowRect` 的 CY 已含行距**(探针实测: 微软雅黑 13px 单行 18 / 两行 36 / 三行 53), 与 `p.Text` 实际排版一致 —— 不要按行数再乘系数补偿, 否则底部多出一整行留白。
    - **位置/透明度动画挂窗口**(`xui.Presence(hWnd, key, …)`, hOwner = 窗口): 透明元素的局部重绘**不擦底**, 卡片一移动就在旧位置留下残影; 挂窗口后每 tick 整窗重绘, 滑入与重排都是干净的(代价是动画期间整窗重绘一次/帧, 通知进出场只有 200ms)。动画期间 `layout` 只更新 `baseX/baseY`、不直接落位, 否则会和 apply 的位置打架。
    - 进场前先 `SetElementsAlpha(0, ele)` + 整窗重绘一帧, 再启动 Presence —— 元素 alpha 要画一遍才落地(同 57/73 的首帧问题), 否则第一帧是上一次的 255。
    - 主题切换: 每条通知 `OnThemeChangeFor` 重新量高再重排 —— 字号/字体变了, 描述折行高度也会变, 只重绘不重量会裁掉末行。

76. **自绘标题栏窗口的封装: `EnableLayout` 必须调, 双击判定别看 `GetStayEle != 0`, 最大化不用手动去阴影**(`ui/window.go`, 亮/暗 + 最大化/还原/拖动/双击逐项实测):
    - **`XWnd_EnableLayout(true)` 漏调 = 子项"存在但不排版"**: 窗口上 `LayoutItem_SetWidth/Height` 全都不生效, 子元素停在创建位置(实测内容区 rect `{12 0 628 0}` 高 0, 界面只剩标题栏)。封装里要在创建子项**之前**开窗口布局; 用户的子项是之后加进内容容器的, 不用再手动 `AdjustLayout`(LayoutEle 增删子项会自己重排)。
    - **双击标题栏判定不能用 `GetStayEle() != 0`**: 标题/图标元素为了不挡窗口拖动都开了 `EnableMouseThrough(true)`, 鼠标停在它们上面时 `GetStayEle()` **仍返回该元素**(非 0) —— 按"非 0 就跳过"写, 双击标题栏整段失效(实测: 事件到了, 但被判断挡掉)。正确做法是只拦按钮: `XC_GetObjectType(stay) == xcc.XC_BUTTON`。
    - **y 判定要算阴影带**: 事件坐标原点是整窗左上角, 非最大化时标题栏落在 `[shadowSize, shadowSize+captionH]`, 最大化时 `[0, captionH]` —— 只写 captionH 会让非最大化时的下半段双击失效。
    - **最大化不需要手动去阴影**: 阴影窗最大化后窗口矩形 = 工作区(实测 1707×1018 逻辑, 屏幕 1707×1067, 差的就是任务栏), 阴影带由系统处理; 别在 WM_SIZE 里去 `SetShadowInfo(0, …)`(会闪)。
    - **三键**: `SetTypeEx(xcc.Button_Type_Min/Max/Close)` 直接拿到原生行为(最小化/最大化/关闭窗口); 自绘图标不会跟状态走, 要在 `WM_SIZE(SIZE_MAXIMIZED/SIZE_RESTORED)` 里换 —— 经 `XC_CallUT` 延后到消息处理之后再读 `IsMaxWindow`, 否则拿到旧状态。
    - **窗口销毁时 `xui.UnregisterWindow`**: 句柄会被系统复用, 主题注册表里留着会让切主题操作到别的窗口(Win32 `WM_DESTROY` 里反注册)。

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
- LayoutEle:`EnableLayout / IsEnableLayout / EnableHorizon / EnableAutoWrap / EnableOverflowHide / SetAlignH(V) / SetSpace / SetSpaceRow / ShowLayoutFrame / GetWidthIn(HeightIn)`、`xc.XWidget_LayoutItem_SetWidth/Height/SetMargin/SetMinSize`、尺寸类型 `xcc.Layout_Size_{Fixed,Fill,Auto,Weight,Percent,Disable}` 与对齐 `xcc.Layout_Align_*`(语义与"Fill≠剩余空间 / Weight 需父级确定尺寸"见第八节 29/41/55;Go 包装无 AdjustLayout, 用 `XEle_AdjustLayout`);`EnableLayout/EnableHorizon/EnableAutoWrap` **只有 LayoutEle 与窗口有**(普通 Element 只能当布局项)
- 布局读取/定位:`xc.XEle_GetRectLogic(hEle, *RECT)`(逻辑像素, 坐标相对父元素)、`xc.XEle_GetRect/GetWidth/GetHeight/GetPosition`、`window.AddEvent_Size(xc.WM_SIZE1)`(字号/尺寸变化后重算节流列宽等);`xc.XEle_SetPadding(hEle, l, t, r, b)` 存在(元素级内边距), 本库暂用子项外边距替代, 未实测
- 滚动容器(`LayoutFrame`, 见第八节 66):`widget.NewLayoutFrame`(布局盒子 + 滚动视图, 内容总大小自动撑开) + `XLayoutBox_*`(与 LayoutEle 同一套 EnableHorizon/AutoWrap/OverflowHide/Align/Space) + `XSView_SetTotalSize/GetTotalSize/SetScrollBarSize/EnableAutoShowScrollBar/ShowSBarH(V)/GetViewWidth/GetViewHeight/GetViewRect/GetViewPosH(V)/ScrollPosYV(像素)/ScrollPosV(行)/ScrollTop(Bottom)/GetScrollBarH(V)Obj`、`ScrollBar.ShowButton/SetSliderPadding/GetButtonSliderObj` + `AddEvent_Size(xc.XE_SIZE1)`(自愈补排)
- `SliderBar.SetRange/SetPos/SetButtonWidth/SetButtonHeight/EnableHorizon/GetButtonObj`、`ProgressBar.SetRange/SetPos/EnableShowText/EnableStretch`、`TabBar.AddLabel/SetSelect/SetLabelWidth`、`ComboBox.CreateAdapter/AddItemText/SetSelItem/EnableEdit/EnableDrawButton/SetDefaultText/SetDropHeight/GetButtonRect`
- 手动建列表(时间框弹层, 见第八节 72):`ListBox.CreateAdapter/AddItemText/SetItemHeightDefault/SetDrawItemBkFlags/GetItemText(i, 适配器列)/GetTemplateObject(i, 模板itemID)/SetSelectItem(不滚动)/GetCount_AD/AddEvent_ListBox_DrawItem/Temp_Create_End/Select/GetScrollBarVObj`;`XShapeText_SetTextColor(h, 0)` 让模板文字全透明、`XWidget_LayoutItem_SetMargin(h, l, t, r, b)`、`xc.XWidget_LayoutItem_SetWidth(…Fill)` 对项模板文字会算出 0 宽(勿用)
- `XWnd_SetWindowPos(物理像素)`、`XC_hWindowFromHWnd`、`XEle_GetWndClientRect`、`XWnd_GetClientRect`、`wapi.ShowWindow/SetForegroundWindow/GetSystemMetrics/GetWindowLongPtrW`
- D2D 直通:`xc.XDraw_GetD2dRenderTarget / XDraw_ConvRect / XC_GetD2dFactory`(裸指针;直通绘制必须先用 `ConvRect` 换到画布对齐坐标,见第八节 43)+ **`github.com/twgh/xcgui/wapi/d2d`**(按 edge 的 COM 接口虚表风格封装,ABI 三条硬规则写在包注释里):借用入口 `NewRenderTargetByHandle`/`NewFactoryByHandle`;几何 `NewGeometry`/`NewGeometryFigures`(逐条可指定开/闭)/`Create{Rectangle,RoundedRectangle,Ellipse}Geometry` + `ID2D1Geometry.Usable/GetBounds` + `ID2D1PathGeometry.Open/Base` + `ID2D1GeometrySink.{SetFillMode,BeginFigure,AddLines,EndFigure,Close}`;绘制 `FillGeometry(Color)`/`DrawGeometry(Color)`(任意几何) + 图元 `DrawLine`/`DrawRectangle`/`FillRectangle`/`DrawEllipse`/`FillEllipse`(各带 `*Color` 便利版) + `CreateStrokeStyle`/`RoundStrokeStyleProperties`;其余 `SetTransform`/`GetTransform`/`GeometryCache[K]`(按工厂作废的几何缓存)
- backdrop-blur(见第八节 47):`xc.XImage_LoadFromGoImage`、`xc.XBkM_AddImage(hBkM, state, hImage, id)`、`xc.XBkM_GetObject`、`xc.XEle_GetBkManager`、`xc.XEle_ClearBkInfo`、`xcc.Element_State_Flag_Leave`(元素态, ≠ Window_State_Flag_Leave)、`XWnd_GetChildCount/GetChildByIndex`、`XEle_GetAlpha/GetBkInfoCount`、`wapi.GetWindowRect` + `PrintWindow(PW_RENDERFULLCONTENT)` 抓窗、`XWnd_SetTimer/KillTimer + AddEvent_Timer`(程序化验收)
- 排查"窗口可见但屏幕上看不见"(见 57):`user32!EnumWindows` + `GetWindowThreadProcessId` 枚举进程窗口(⚠️ 别用 `Process.MainWindowHandle`, 弹层小窗存在时它会返回小窗)、`IsWindowVisible`、`user32!GetLayeredWindowAttributes`(读 layered alpha/颜色键)、`PrintWindow` 与 `CopyFromScreen` 对照(画了 vs 屏幕上没有)
- `app.GetTextShowRect`(文本测量:限宽换行与绘制共用限宽;返回的 CY **已含行距** —— 13px 字体单行 18 / 两行 36, 见第八节 75)
- `Element.SetBorderSize(左, 上, 右, 下)`:原生 Edit/ComboBox 的**文字内边距**, 传**逻辑像素**、由框架随 DPI 自动缩放(实测 150% DPI: Input 传 10, 文字起点距边框 20 物理 ≈ 13.3 逻辑 = 10 + 字形左侧留白) —— 别再乘 `dpi/96`, 会重复放大
- 窗口真透明:`Window.SetTransparentType(xcc.Window_Transparent_Shaped)` + `Window.SetTransparentAlpha(255)`、`xcc.Window_Transparent_Shaped`(xcconst.go:55);**对原生自建弹窗**(ComboBox 下拉, 只有窗口句柄)同样可用: `xc.XWnd_SetTransparentType / XWnd_SetTransparentAlpha`, 在弹出回调里(显示前)设置, 见第八节 69
- 背景对象美化:`BkManager.AddFill/AddBorder(state, color, id)`、`BkManager.GetObjectObj(id) *bkobj.BkObj`、`BkObj.SetRectRoundAngle/SetMargin/SetAlign/SetRotate/SetFillColor/SetBorderColor`、`xcc.BkObject_Align_Flag_*`(bkmanager.go:69/185, bkobj.go:104/42/50/66/74/90)
- Dialog:`XEle_SetZOrderEx(h, 0, xcc.Zorder_Top)`、`XEle_AdjustLayout`、`XEle_Redraw`、`XEle_GetChildCount/GetChildByIndex`、`XObj_GetType`(→ `xcc.XC_*`)、`XWidget_GetParentEle`、`XWnd_GetRect/Redraw`、`XWnd_SetFocusEle/GetFocusEle`、`xcc.AdjustLayout_No`、`wapi.GetAsyncKeyState`
- Toast(见第八节 75): `XWnd_GetBodyRect`(Body 矩形, 阴影窗口有 band 偏移)、`XWnd_GetStayEle`(窗口级悬停判定)、`XEle_SetXCTimer/KillXCTimer + AddEvent_XC_Timer`(每条通知自己的倒计时)、`XEle_Destroy`、`Element.AddEvent_Destroy_End`、`Window.AddEvent_Size/MouseMove/MouseLeave`、`XEle_SetZOrderEx`
- Table(原生 List 载体):`CreateAdapterHeader/CreateAdapter/AddColumnText/SetHeaderHeight/SetRowHeightDefault/SetRowSpace/SetColumnWidth/SetSplitLineColor/SetDrawRowBkFlags/AddRowText/SetItemText/GetCount_AD/SetSelectRow/GetSelectRow/GetVisibleRowRange/RefreshRow/RefreshData/AddEvent_List_DrawItem/AddEvent_List_Header_DrawItem/AddEvent_List_Temp_Create_End/AddEvent_List_Header_Temp_Create_End/AddEvent_List_Select/GetTemplateObject/GetHeaderTemplateObject/GetHeaderHELE/GetScrollBarVObj`
- ⚠️ `TabBar.SetPadding` 在 xcgui.dll 4.0.0.0 中不存在(调用 panic)
- 直通 D2D(图标抗锯齿, 见第八节 43):`xc.XDraw_GetD2dRenderTarget(hDraw) → *ID2D1RenderTarget`、`xc.XC_GetD2dFactory → *ID2D1Factory`、`xc.XDraw_ConvRect(hDraw, *RECT) / XDraw_ConvXY`(客户区坐标 → 画布对齐物理坐标, 第三方绘图必须先换算); 用到的 D2D 方法: `ID2D1Factory::CreatePathGeometry`、`ID2D1PathGeometry::Open / GetBounds`、`ID2D1GeometrySink::{SetFillMode, BeginFigure, AddLines, EndFigure, Close}`、`ID2D1RenderTarget::{CreateSolidColorBrush, FillGeometry}`
- 官方自绘示例:BeautifyButton / CheckBox / RadioButton / Edit / ProgressBar / SliderBar
- 官方透明/异型窗口示例:`Advanced/WindowPartiallyTransparent`、`webview/WebViewPartiallyTransparent`
