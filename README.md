# xui — shadcn 风格的 xcgui 自绘组件

<p>
    <a href="https://github.com/twgh/xcgui"><img src="https://img.shields.io/badge/xcgui≥1.4.3-blue" alt="release"></a>
   <a href="https://golang.org"> <img src="https://img.shields.io/badge/golang-≥1.18-blue" alt="golang"></a>
    <a href="https://pkg.go.dev/github.com/twgh/xui"><img src="https://img.shields.io/badge/go.dev-reference-brightgreen" alt="GoDoc"></a>
    <a href="https://github.com/twgh/xui/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-brightgreen" alt="License"></a>
</p>

## 介绍

为 [github.com/twgh/xcgui](https://github.com/twgh/xcgui)(炫彩界面库)提供一套 **shadcn-ui 风格**的自绘组件:主题令牌、变体系统、copy-not-install 分发。

```
┌─────────────────────────────────────────────┐
│  你的应用                                    │
│  ┌───────────────────────────────────────┐  │
│  │ ui/  ← 组件源码归你所有, 随意修改       │  │
│  │   button.go  checkbox.go  input.go …  │  │
│  │   theme.go   ← 你的主题, 改这里换肤    │  │
│  └───────────────────────────────────────┘  │
│              ↑ 依赖              ↑ 拷贝      │
│  github.com/twgh/xui (运行时)   xui CLI     │
│              ↑                              │
│  github.com/twgh/xcgui (行为层)             │
└─────────────────────────────────────────────┘
```

### 示例图

![xui](https://s41.ax1x.com/2026/10/19/pn0t240.png)

## 快速开始

### 直接运行示例(想先查看效果)

不想先建项目时, 可以直接运行仓库里的 demo(全部组件 + 亮/暗主题切换):

```bash
go run github.com/twgh/xui/demo@latest
```

### 新建项目(推荐)

```bash
go run github.com/twgh/xui/cmd/xui@latest init -n myapp
cd myapp
go run .
```

`-n` 会创建一个完整项目目录:`go.mod` + `main.go`(AppWindow + 示例卡片) + `README.md` +
`.gitignore` + `ui/`(theme.go + 基础组件 + DESIGN.md), 并**自动执行 `go mod tidy`** 装好
依赖(生成 `go.sum`)。目录已存在且非空时需加 `-f/--force`; 想跳过依赖安装(如离线)加
`--no-install`, 之后手动跑一次 `go mod tidy`。

### 已有项目

```bash
# 先在你的项目根目录执行(需要 Go 1.18+;项目要先有 go.mod,没有就先 go mod init)
go run github.com/twgh/xui/cmd/xui@latest init
```

此时 `init` 只创建 `ui/` 目录:写入 `theme.go`(你的主题文件)、拷入全部基础组件,外加一份
`DESIGN.md` —— 组件源码的注释里到处是"见 DESIGN 第 N 节",这份**自定义指南**随源码一起安装在你的项目里:
令牌表、组件解剖、xcgui 踩坑全集、已验证 API 清单都在里面。改组件前先翻它。

```go
package main

import (
    "github.com/twgh/xcgui/app"
    ui "yourmodule/ui" // 换成你的模块路径
)

func main() {
    app.InitOrExit()
    a := app.New(true)                    // Direct2D 渲染
    a.EnableAutoDPI(true).EnableDPI(true) // 跟随系统 DPI
    a.SetPaintFrequency(10)               // hover 过渡 ~60fps

    // 窗口: 圆角阴影 + 自绘标题栏(最小化/最大化/关闭) + 主题化底色, 全部默认配好
    w := ui.NewAppWindow("我的应用", 640, 480)
    content := w.Content()
    content.SetPadding(24, 20, 24, 24) // 内容区留白

    ui.NewButton(0, 0, 0, 0, "确定", content.Handle)
    ui.NewCheckbox(0, 0, 0, 0, "记住我", content.Handle)
    ui.NewInput(0, 0, 0, 0, content.Handle, ui.WithPlaceholder("用户名"))

    w.Show(true)
    a.Run()
    a.Exit()
}
```

标题栏要加自己的按钮(如切换主题)用 `w.AddCaptionButton(icons.IconMoon, func() { ... })`,
排在最小化按钮左边; 内容组件纵向排列, 间距由内容容器的 `SetSpace` 控制。

## 换肤

组件在**绘制时**实时读取主题令牌(等价于 CSS 变量),切换主题零成本:

```go
xui.SetTheme(xui.Dark())          // 一行全局换肤
```

主题文件 `ui/theme.go` 默认是**内置主题的副本**(逐字段改即可换肤),也可以照 shadcn 的
`:root` / `.dark` 自己写一份 —— 令牌与 CSS 变量一一对应(字段表见 DESIGN.md 3.1)。

## 组件

| 组件 | 说明 |
|---|---|
| Button | 6 变体(Primary/Secondary/Outline/Ghost/Destructive/Link)× 8 尺寸档(xs/sm/default/lg + 4 个方形图标档),hover/按下/禁用/焦点环/校验失败(`WithButtonInvalid`);`WithIcon` 做图标按钮,`WithButtonLeadingIcon` / `WithButtonTrailingIcon` 做"图标+文字",`WithButtonAlign(Start/Between)` 控制整宽按钮的内容对齐(示例里的 justify-start/justify-between)、`WithButtonRounded` 胶囊、`WithButtonExpanded` / `WithButtonHasPopup` 对应弹层触发区、`WithButtonLoading` 显示 1s/圈旋转的加载图标;`cx` 传 0 时按文字自动量宽 |
| ButtonGroup | 按钮组(`components/buttongroup.go`):横向/纵向(`WithButtonGroupOrientation`)、首尾外角 + 相邻边直角、宽高按内容自撑;可放 Button / Input / ComboBox;`NewButtonGroupText` 文本块、`NewButtonGroupSeparator` 分隔条跟随组高;组里嵌组自动 8px 间距 |
| Alert | 提示条:图标 + 标题 + 描述 + 右上动作槽;`AlertDefault` / `AlertDestructive` 两变体,标题/描述随宽度自动折行、高度按内容自撑;`Action()` 里放 Button / Badge |
| Empty | 空状态容器(`components/empty.go`):图标底板(`EmptyMediaIcon`, 40px bg-muted 圆角)/无底板媒体 + 标题 + 描述 + 动作区;`NewEmpty` 的 `Header()` / `Content()` / `Body()` 三个槽位,内边距 24、间距 24、整块居中;不画边框(与上游默认一致),高度自撑,整页空态可 `xui.ItemFillHeight` 让内容垂直居中 |
| Field | 表单字段布局(`components/field.go`):`NewField` + `Label` / `Title` / `Description` / `Content` / `SetInvalid`;`FieldSet`(gap 24)/ `FieldGroup`(gap 28)/ `FieldLegend` / `FieldSeparator`(可带居中文字)/ `FieldError`(多条去重);垂直/水平取向,校验失败时标签转 destructive;横向 `Content` 默认 240 宽(嵌套布局只认 Fixed 宽,用 `xui.ItemWidth` 覆盖) |
| Item | 列表项(`components/item.go`):`NewItem` + `Media()` / `Content()` / `Actions()` / `Header()` / `Footer()` 槽位;`ItemDefault` / `ItemOutline` / `ItemMuted` 三变体,`ItemSizeDefault` / `ItemSizeSm` 两档;`ItemMedia`(icon 变体 32 底板)/ `NewItemTitle` / `NewItemDescription` / `NewItemGroup` / `NewItemSeparator`;Content 宽度按媒体/动作实测自动锁成 Fixed |
| Breadcrumb | 面包屑(`components/breadcrumb.go`):`NewBreadcrumb` 横向换行列表 + `NewBreadcrumbItem` / `NewBreadcrumbLink`(hover 过渡到前景色、`OnClick`)/ `NewBreadcrumbPage` / `NewBreadcrumbSeparator`(可换图标)/ `NewBreadcrumbEllipsis`;文本自绘、按字体精确量宽 |
| Kbd | 键帽(`components/kbd.go`):高 20、bg-muted、rounded-sm、text-xs,宽度按文字自适应;`NewKbdGroup` 组合键间距 4 |
| Skeleton | 骨架屏(`components/skeleton.go`):bg-accent + 主题圆角 + 2s 脉冲;`WithSkeletonCircle` 头像、`WithSkeletonRadius` 自定义圆角、`WithSkeletonStatic` 关闭动画 |
| Spinner | 加载指示器(`components/spinner.go`):Loader2 图标 1s/圈旋转,`WithSpinnerSize` / `WithSpinnerMuted` / `WithSpinnerColorFunc`,`SetSpinning` 开关 |
| Toggle | 双状态按钮(`components/toggle.go`):`ToggleVariantDefault`(透明底)/ `ToggleVariantOutline`(border-input)、`ToggleSizeDefault/Sm/Lg`(h-9/h-8/h-10);`WithTogglePressed` 初始选中、`WithToggleIcon` 行内图标、`WithToggleInvalid` 校验失败;鼠标/空格/回车切换,支持焦点环与 `OnChange` |
| ToggleGroup | 切换按钮组(`components/toggle.go`):`NewToggleGroup(parent, WithToggleGroupVariant/Size/Spacing/Multiple)` + `AddItem(value, text, ...)`;单选默认同组互斥且可全部取消,`Value` / `SetValue` / `OnChange`,spacing=0 首尾外角(分段控件),spacing 为逻辑像素 |
| NativeSelect | 原生下拉框(`components/native_select.go`):ComboBox 载体 + `<option value>` 字符串语义(`Value` / `SetValue` / `OnValueChange`);`WithNativeSelectPlaceholder` / `WithNativeSelectSize` / `WithNativeSelectInvalid`;富条目(图标/禁用/分组)用 Select |
| Select | 独立下拉选择框(`components/select.go`,radix item-aligned 弹层):条目 `SelectItem{Value, Text, Icon, Disabled}`、分组 `SelectGroup{Label, Items}`(`WithSelectGroups` / `WithSelectItems`);弹层盖住触发区、选中项正对触发区,↑↓(跳过禁用)/Home/End 移动高亮、Enter/空格 选中、ESC 关闭;`Open` / `Close` / `IsOpen` / `SetValue` / `OnChange` |
| InputOTP | 验证码输入框(`components/inputotp.go`):逐格输入,打进已填格 = 替换并前进;←→/Home/End 移动、Backspace 空格退格、Ctrl+V 分发粘贴(非法字符跳过);`WithInputOTPLength` / `WithInputOTPPattern(InputOTPPatternDigit/Alphanumeric/Any)` / `WithInputOTPSeparatorAfter(n)` / `WithInputOTPValue` / `WithInputOTPInvalid` / `WithInputOTPDisabled`;全满触发 `OnComplete` |
| Accordion | 手风琴(`components/accordion.go`):`NewAccordion` + `AddItem(value, title)`(返回条目,内容放 `ContentStack()`,**内容里的子项记得 `xui.ItemFill` 声明宽度**);single 单开互斥可全收 / `WithAccordionMultiple` 各自独立;触发区 hover 下划线 + 箭头随开合切换,条目间 1px 分隔线,展开 200ms 高度过渡;根 `Value` / `SetValue` / `OnChange` |
| Popover | 弹出卡片(`components/popover.go`):`NewPopover(trigger.Handle, WithPopoverContent(func(c int){...}))`,内容放进纵向流容器;点击触发、点外部/ESC 关闭;默认宽 288、内边距 16、内容高自撑,`WithPopoverWidth/Height/Padding/Gap/Side/Align`;`PopoverHeader/Title/Description` 做标题区;`Open/Close/IsOpen/Refresh` |
| HoverCard | 悬停卡片(`components/hovercard.go`):`NewHoverCard(trigger.Handle, WithHoverCardContent(...))`;悬停 700ms 打开、移出 300ms 关闭,卡片可交互且不抢焦点;`WithHoverCardWidth/Padding/Side/Align/Delay`;适合用户资料预览 |
| Avatar | 圆形头像(`components/avatar.go`):`NewAvatar(..., fallback, WithAvatarSize/Image/ImageData/Ring/Badge)`;图片居中裁圆,无图回落首字母;右下角徽标(可带图标);`NewAvatarGroup` 负间距叠放 + `Add` / `AddCount`;尺寸 sm 24 / default 32 / lg 40 |
| Sheet | 侧边面板(`components/sheet.go`):`NewSheet(w.Handle, WithSheetSide(SheetRight/Left/Top/Bottom))`,标题/描述/内容/页脚与 Dialog 同一套(`SetTitle` / `SetDescription` / `Content()` / `AddFooter`);`WithSheetWidth` / `WithSheetCloseButton` / `WithSheetDuration`;ESC / 点遮罩 / X 关闭,Tab 焦点陷阱,从贴边侧滑入 |
| Drawer | 底部抽屉(`components/drawer.go`):`NewDrawer(w.Handle)` + 同款内容 API;顶部抓手按住下拖过面板高 1/3 关闭、否则回弹;`WithDrawerDirection` 可改 top/left/right;`BindTrigger` / `BindClose` / `OnClose` |
| Menubar | 菜单栏(`components/menubar.go`):`NewMenubar(card.Content())` + `AddMenu("文件")`,条目 API 与 Menu 共用(`Item` / `Check` / `Radio` / `Label` / `Separator` / `Sub`,选项同为 `MenuShortcut` / `MenuOnSelect` / `MenuOnCheck` / `MenuDisabled` / `MenuDestructive` / `MenuGroup` / `MenuIcon` / `MenuInset`);打开任一菜单后鼠标移到其它触发区即切换,↑↓/Home/End/Enter/←→/ESC/Tab 键盘导航,子菜单悬停展开 |
| NavigationMenu | 导航菜单(`components/navigationmenu.go`):`NewNavigationMenu(parent)` + `AddMenu(title, func(c *ui.NavigationMenuContent){ c.Link(标题, 说明, fn) }, WithNavigationMenuWidth(w))` + `AddLink`;所有菜单共用一个贴在列表下方的视口,悬停 200ms 打开、离开 150ms 收起,切换菜单时视口随内容收放、新页面从切换方向滑入;内容里可用 `LinkIn` 放进 `xui.HStack` 做多列网格,`c.Handle()` 放任意自定义控件;内容不抢焦点,键盘 ←→/↓/Enter/ESC 与 `Value` / `SetValue` / `OnValueChange` |
| Carousel | 轮播(`components/carousel.go`):`NewCarousel(x,y,cx,cy,parent, WithCarouselLoop/Controls/Vertical/Gap/Duration/InitialIndex/Autoplay/OnChange)` + `AddSlide(func(s *ui.CarouselSlide){...})`;整页滑动(300ms,循环首尾相接)、←/→ 键盘翻页、鼠标拖拽松手吸附;上/下张是内容外侧 48px 的圆形描边按钮,到边界禁用;幻灯片是纵向流容器,内容超出由视口裁剪;`SetIndex` / `Next` / `Prev` / `Index` / `SetLoop` / `SetAutoplay` / `PrevButton` / `NextButton` |
| Command | 命令面板(`components/command.go`):`NewCommand` + `AddGroup(label).AddItem(value,text, WithCommandItemIcon/Shortcut/Keywords/Disabled)` + `AddSeparator`;输入即过滤、↑↓ 循环跳过禁用、Home/End/Enter/ESC,鼠标悬停高亮;`WithCommandOnSelect/Placeholder/EmptyText`;可放进 `ui.NewDialog` 组合成命令面板弹窗 |
| Checkbox / Radio | 自绘指示器 + 原生状态管理;Checkbox 三态(`WithCheckboxIndeterminate` / `SetIndeterminate`,不确定态画减号、点击落到选中)、校验态(`WithCheckboxInvalid` / `SetInvalid`);Radio 同款校验态;两者都支持空格切换/选中与键盘焦点环 |
| RadioGroup | 单选组(`components/radiogroup.go`):`NewRadioGroup(parent, WithRadioGroupValue/Disabled/Invalid/Spacing)` + `AddItem(value, text)`;另用 `Attach(radio, value)` 收编放在 Field 等子容器里的条目(跨父级互斥);`Value` / `SetValue` / `OnChange` / `SetDisabled` / `SetInvalid`;↑↓←→ 循环移动并选中、Home/End 首末、空格选中,组级禁用不影响各自原有禁用项 |
| Input | 原生 Edit(光标/输入法/选择)+ 自绘壳体与焦点环,占位符随主题;校验失败态 `WithInputInvalid`;内嵌附加件:`WithLeadingIcon` / `WithPrefixText` / `WithSuffixText` / `WithClearButton` / `WithPasswordToggle`(配合 `SetPasswordVisible` / `PasswordVisible`) / `WithCounter`;步进器 `WithStepper(min,max,step)` + `SetNumber` / `Number` / `OnStep`(支持 −/+ 按钮与 ↑/↓ 键,失焦自动夹取吸附;`WithStepperArrows` 换成上下三角 = 原生 number 输入框形态,`WithStepperAutoHide` 让按钮只在悬停/聚焦时出现) |
| Switch / Progress / Slider | 开关 / 进度条 / 滑块;滑块支持自定义值域与步进(`WithSliderRange` / `WithSliderStep`) |
| Tabs | 标签页(下划线指示条随活动页滑动),支持纵向堆叠(`WithTabsVertical`);内容超出宽度时可用 `WithTabsWrap` 自动换行 |
| Collapsible | 可折叠面板(触发区 + 内容区):`NewCollapsible` 建根容器,`BindTrigger` 绑定任意 Button 作触发区,内容放进 `Content()`/`ContentStack()`;展开/收起带 200ms 高度过渡(收起后不占布局),支持初始展开(`WithCollapsibleOpen`)、禁用与 `OnOpenChange` |
| ComboBox | 下拉框(列表由独立透明弹出窗口承载, 打开时 100ms 淡入);触发区支持 sm 尺寸(`WithComboSize`)与校验失败态(`WithComboInvalid`) |
| Calendar | 月历卡片(全自绘):6×7 固定网格、周起始日可调(`WithCalendarWeekStart`)、选中实心/今天浅底/悬停过渡;**范围选择**(`WithCalendarRange` + `WithCalendarOnRangeSelect`, 两次点击落区间、端点交换/中段色带, 同 react-day-picker)、**多月并排**(`WithCalendarMonths(1~4)`)、**ISO 周数列**(`WithCalendarWeekNumbers`)、禁用(`WithCalendarDisabled` + Before/After/Weekend/Dates helper)与预订划线(`WithCalendarStrike`)、年月可下拉(`WithCalendarYearRange`)、页脚/预设(`WithCalendarFooter` / `WithCalendarPresets`, 自动换行);可独立摆放, 弹层内用 `WithCalendarFlat` 免双层边框; 尺寸 helper `CalWidthFor` / `CalWidthForWeekNumbers` / `CalHeightForMonths` |
| DatePicker | 日期框:outline 触发区 + 自建阴影弹层(内含 Calendar),下方放不下自动翻到上方;点外部/ESC 关闭,`OnChange` 回调;**范围模式**(`WithDatePickerRange`, 弹层并排两个月, 起点未落终点时保持打开, `Range/SetRange/OnRangeChange`)、预设按钮(`WithDatePickerPresets`)、年份范围与禁用日期、**可输入形态**(`WithDatePickerEditable`: 可键入 2026-10-22 / 今天 / 3天后, Enter/失焦回写, 右侧日历钮打开弹层) |
| TimePicker | 时间框:outline 触发区 + 弹层内"时/分"两列滚动列表(原生 ListBox 行为,条目自绘);选择即生效,弹层保持打开 |
| Menu | 下拉菜单 / 右键菜单(原生菜单承载行为 + 自绘皮肤):条目 / 勾选 / 单选组 / 子菜单 / 快捷键 / 禁用 / 破坏性 / 文字缩进(`MenuInset`, 条目/Label/Sub 都可用) / 前置图标(`MenuIcon`);`NewContextMenu(触发区元素, build)` 右键触发(每次右键重建菜单, 支持 `SetEnabled` / `OnOpenChange`);弹层宽支持 `Width` 固定、触发区对齐、内容自适应 + 128 下限 |
| Tooltip | 提示气泡(独立透明弹出窗口,不抢焦点) |
| Toast | 通知(窗口内覆盖元素,默认右下角堆叠,支持四角与上下居中):4 种语义类型(success/error/warning/info)+ 动作按钮 + 可选关闭按钮;`WithToastRichColors` 可让类型给整卡上色;自动关闭(默认 4s)、悬停暂停、进出场滑入淡入;`NewToaster(w.Handle).Success("标题","描述")` |
| AppWindow | 主窗口封装:圆角阴影 + 自绘标题栏(最小化/最大化/关闭,双击标题栏最大化/还原,拖动,最大化图标自动切换) + 主题化底色;`NewAppWindow("标题", 640, 480)` 后往 `Content()` 里放内容,`AddCaptionButton` 加标题栏按钮(如切换主题),`WithAppWindowNoMinMax` 只留关闭 |
| Dialog | 模态对话框:全窗遮罩 + 居中面板,标题/描述(按宽度折行)+ 任意内容 + 页脚按钮;ESC、点击遮罩、右上角 X 关闭,Tab 在对话框内循环;`WithDialogScroll` 让内容区滚动、头部/页脚固定,`BindClose` 可把任意控件变成关闭钮 |
| AlertDialog | 警示对话框:必须明确选择(无右上角 X、点遮罩不关闭、ESC 关闭,焦点落在 Cancel);`AlertDialogSizeDefault` / `Small`(居中头 + 等宽双列页脚)、`WithAlertDialogMedia` 图标块(含破坏性配色);`AddCancel` / `AddAction` 点击后关闭;与 Dialog 共用遮罩/模糊/焦点陷阱,支持对话框内再叠一层 |
| Table | 数据表格(原生 `List` 载体 + 全自绘皮肤):表头/行边框/悬浮/选中/主题化细滚动条;列宽支持"占满剩余宽度" |
| Tree | 树形视图(原生 `Tree` 载体 + 全自绘皮肤):缩进参考线/旋转 chevron/行悬浮选中;叶节点留空槽位文字全树对齐,点击 chevron 展开收起 |
| Pagination | 分页(纯自绘单元素,鼠标驱动):上一页/下一页 + 页码 + 省略号折叠(>7 页),内容居中、当前页为描边态;`SetTotal/SetPageSize/SetPage/OnChange` |
| ScrollArea | 滚动容器(原生 `LayoutFrame` 载体):内容比容器高时纵向滚动,内容总大小按子项自动撑开、滚动条自动显隐;主题化细滚动条,内容区 `ContentStack()` 随便塞;`SetContentFill()` 切"铺满视口 + 面板吃剩余高度"形态 |
| AspectRatio | 固定宽高比容器: 宽度由布局/布局项决定,高度 = 宽度 / ratio(16:9、1:1、9:16…),宽度变化时自动重算并逐级重排;内容放 `Content()` 里铺满比例框,`SetRatio` 运行时切换比例 |
| Label / Card / Badge / Separator | 静态展示组件:Badge 6 变体(含 ghost/link,悬停生效)、Card 支持 `sm` 尺寸与说明/页脚/卡片头(`NewCardHeader` 的标题+说明+右上动作区;`Card.Footer` 页脚内边距随尺寸、内容垂直居中)、Separator 支持垂直 |

交互类组件默认带齐 4 态:正常 / hover / 按下 / 禁用,外加键盘焦点环(静态展示与容器类组件没有按下态)。

hover 带过渡动画(默认 **150ms + cubic-bezier(0.4,0,0.2,1)**,与 tailwind / shadcn 的 `transition-*`
默认值一致;表格行/树行/菜单条目这类逐项高亮同样有过渡)。过渡由运行时管理(临时开一个 16ms 的
定时器,跑完自动关,静止时零开销;实际帧率受下面的 `SetPaintFrequency` 节流),组件侧只写一行:
`hv := xui.Hover(hEle, st.Hover)` + `xui.Fade(静止色, 悬停色, hv)`。整体时长/开关可调:

```go
xui.SetHoverDuration(200 * time.Millisecond) // 改时长
xui.SetHoverDuration(0)                      // 关掉过渡(立即到位)
```

想让过渡更顺(约 60fps)只需在 app 初始化里放开 xcgui 默认的 30ms 重绘节流:

```go
a := app.New(true)
a.EnableAutoDPI(true).EnableDPI(true)
a.SetPaintFrequency(10)   // 实测帧率 30.9 → 59.0 fps(见 DESIGN.md 第八节 46)
```

对话框与提示气泡自带进出场动画(dialog 淡入淡出 100ms、tooltip 淡入+8px 滑入 150ms, 对齐
shadcn/tw-animate-css 的默认值)。自己的弹层要用同款动画时:

```go
// 整组元素淡入/淡出(进度 0→1 进场, 1→0 退场; apply 每帧收到当前进度)
xui.Presence(hWnd, 0, visible, xui.PopupDuration,
    func(v float64) { xui.SetElementsAlpha(v, panel, btn1, btn2) },
    func(v float64) { /* 退场到位后在这里隐藏 */ })

// 异型/阴影弹层小窗: 淡入 + 从 (x, y+dy) 滑到 (x, y)
xui.Presence(hWnd, 0, true, xui.PopupDuration,
    xui.SlideWindowAlpha(hPopup, x, y, 0, dy, 255), nil)
```

## 图标

运行时自带一份预处理好的 lucide 图标资源(2130 个,当前版本见 `icons.Version`),
描边用直通 D2D 几何填充出图(自带抗锯齿)。**每个图标一个常量**,名字是 lucide 名的驼峰形式:

```go
import "github.com/twgh/xui/icons"

p.Icon(icons.IconEye, &rc, xui.Current().Foreground, 2)   // 常量: 编译期内联, 零体积成本
p.Icon(icons.IconChartScatter, &rc, color, 2)             // 含"填充点"的图标也照渲染

// 名字来自用户输入/配置时按名字取 —— 会把整张资源表链进二进制(实测 +426KB):
p.Icon(icons.Path("eye"), &rc, color, 2)
icons.Has("star"); icons.Names(); icons.Count()           // 查询
```

也可以直接把 path 数据贴进 `Painter.Icon`(24 网格,`strokeWidth` 用 24 网格单位,lucide 默认 2);
**手里是整段 SVG**(从 heroicons / feather / iconfont 复制来的 `<svg viewBox="…"><path d="…"/></svg>`)
也交给**同一个 `Painter.Icon`** —— 它会自动识别(看首字符是不是 `<`),提取 path、按 viewBox 归一到
24 网格、按 fill/stroke 判定填充:

```go
p.Icon(svgText, &rc, color, 0)                // 整段 SVG: 0 = 用 SVG 自己的 stroke-width(heroicons 1.5)
paths, sw, err := xui.ParseIconSVG(svgText)   // 想先校验 / 自己缓存归一化结果
```

组件内部用到的图标同样取 `icons.IconXxx`:实测"只用常量"的程序体积增量 **0 字节**(2130 条数据里
只进用到的那条),所以组件、业务代码都推荐用常量。资源与常量由仓库脚本从 lucide 生成(用户无需关心),
转换规则与踩坑见 DESIGN.md 第八节 44。

仓库 demo 的「图标」页可以搜索/翻页浏览整份资源:格子里只画图标(不显示名字),
**悬浮出提示气泡显示图标名, 点击把图标名复制到剪贴板**。

## CLI

```bash
xui init -n myapp       # 新建完整项目(自动 go mod tidy; go.mod + main.go + README + ui/)
xui init -n .           # 在当前空目录直接生成项目(项目名取目录名)
xui init [ui目录]       # 已有项目: 只写 ui/(theme.go + 基础组件 + DESIGN.md; 已存在则跳过)
xui add button card     # 追加拷贝指定组件(自动带上依赖)
xui add --all           # 拷贝全部组件
xui add card --dry-run  # 只显示将创建/覆盖/跳过的文件, 不写本地
xui add card --view     # 打印组件源码(可加 =card.go 只看某个文件), 不写本地
xui add card --diff     # 与本地文件做逐行 diff, 不写本地
xui update              # 更新已装组件(省略组件名 = ui/ 里已装的全部)
xui update button       # 只更新指定组件(含依赖)
xui list                # 列出可用组件
xui search -q table     # 按组件名搜索(等价于 list -q table)
xui view button         # 打印组件源码
xui info                # 显示 CLI / 项目环境信息
```

选项:

- `init -n, --name <项目名>`: 新建项目目录(`-n myapp` 等价于在当前目录创建 `myapp/`);
- `init -f, --force`: 允许写入非空项目目录(覆盖模板文件, 不动无关文件);
- `init --no-install`: 跳过生成后的自动 `go mod tidy`(离线时用);
- `add/update -o, --overwrite`: 覆盖本地已修改的组件文件(默认警告并跳过);
- `add/update -p, --path <目录>`: 组件目录, 默认 `ui`;
- `add/update -a, --all`: 全部组件;
- `add --dry-run`: 预览动作, 不写文件;
- `add --view[=文件]` / `--diff[=文件]`: 查看源码 / 与本地对比, 不写文件;
- `list/search -q, --query <词>` / `-l, --limit <n>` / `-o, --offset <n>`: 过滤、限制数量、跳过前 n 个;
- 通用 `-c, --cwd <目录>` 先切工作目录, `-s, --silent` 只打印警告与错误。

组件源码拷进项目后归你所有;`add` / `update` 都不会覆盖你本地改过的文件,确认要丢弃本地改动时加 `-o/--overwrite`。

**数据源**:默认从仓库 `main` 分支现拉组件,组件永远最新。想锁 CLI 与组件的版本时,把 `@latest`
换成具体版本号(如 `@v0.1.0`)。默认源**自动选路**:

```
直连 https://raw.githubusercontent.com/twgh/xui/main
  → Gitee 镜像 https://gitee.com/twgh/xui/raw/main        (国内稳定)
  → GitCode 镜像 https://api.gitcode.com/api/v5/repos/twgh/xui/contents (国内备用)
  → 加速代理 ghfast / gh-proxy                             (兜底)
```

每个候选的探测就是拉一次 `registry.json`(6s 超时),第一个通的用它跑完本次命令;全都不通才降级到
CLI 内嵌源码。选路过程逐行打在 stderr 上, 想看清卡在哪一环直接看输出。

拷到一半某个文件拉不到(镜像限速、临时挂起)时, CLI 会按候选顺序换下一个源接着拷并把新源粘住,
不会留下半套 `ui/`。

环境变量 `XUI_REGISTRY` 覆盖数据源(**写什么就用什么,不再套代理**):

```bash
XUI_REGISTRY=D:\path\to\xui xui add dialog                    # 本地目录(离线)
XUI_REGISTRY=https://raw.githubusercontent.com/twgh/xui/v0.1.0 xui add dialog   # 锁组件版本(换成实际 tag)
XUI_REGISTRY=https://raw.githubusercontent.com/twgh/xui/main xui add dialog     # 只走直连(跳过镜像与代理)
XUI_REGISTRY=https://gitee.com/twgh/xui/raw/main xui add dialog                # 只走 Gitee 镜像
XUI_REGISTRY=https://api.gitcode.com/api/v5/repos/twgh/xui/contents xui add dialog  # 只走 GitCode 镜像
```

显式指定 `XUI_REGISTRY` 时拉取失败会直接报错(便于排查),不静默降级。

## 设计要点(为什么这样做)

- **copy-not-install**:组件源码进你的仓库,可改可删,不受上游版本绑架——组件是起点,不是黑盒。
- **行为/视觉分层**:xcgui 原生控件当作"无头"层(事件/状态/焦点/输入法),`AddEvent_Paint` + `drawx` 只负责外观。
- **绘制时取令牌**:组件永不缓存颜色,这是零成本换肤的前提;需要同步给原生 API 的颜色(Edit 文字色、窗口底色)通过 `xui.OnThemeChangeFor` 订阅(元素销毁时自动反注册)。
- **组件文件薄**:绘制细节下沉到运行时助手(`xui.Painter`);连 hover 过渡这种"要定时器/要生命周期"的脏活也归运行时(`xui.Hover`/`HoverItem`/`Fade`),拷贝文件主要是"变体→令牌映射 + 几笔绘制",降低升级冲突面。
- **交互语义对齐 shadcn 源码**:hover 透明度、focus-visible(键盘才显示焦点框)、disabled opacity-50、按压下沉等均逐条对照 shadcn-ui 官方源码实现。

## 已知约束

- 仅 Windows;渲染模式固定按 Direct2D(`app.New(true)`)设计。
- 组件坐标使用逻辑像素(`XEle_GetClientRect` 返回未缩放矩形),框架渲染时统一按窗口 DPI 变换;
  自绘**不要**再乘 DPI(开 `EnableAutoDPI` + `EnableDPI` 即可)。
- **依赖 `github.com/twgh/xcgui` 的新版本**:用到 `xcgui/svgp`(图标 path 解析)与 `xcgui/wapi/d2d`
  (直通 D2D),这两个包从 **v1.4.3** 起提供 —— v1.4.2 及更早没有,`go get` 到旧版会编译失败。
- 主题切换须在 UI 线程调用 `xui.SetTheme`。
- **本模块的 `go` 指令是 1.18**:即使你用 Go 1.22+ 的工具链,`for range` 的循环变量也按**旧语义共享** ——
  在循环里给控件注册回调(或起 goroutine)时,循环变量必须先拷一份,否则所有回调都取到最后一次的值:

  ```go
  for _, item := range items {
      item := item // ← 必须有(本模块 go 指令 < 1.22)
      btn.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int { use(item); return 0 })
  }
  ```

详见 [DESIGN.md](docs/DESIGN.md)。

## 参考实现(shadcn-ui)

交互语义与视觉以 [shadcn-ui](https://github.com/shadcn-ui/ui) 官方源码为准:

- 组件源码:`packages/shadcn/src/registry/**/ui/*.tsx`(语义的唯一权威来源)
- 主题令牌:`app/globals.css` 的 `:root` / `.dark`

本仓库的 `demo/` 是按它逐页对照做的视觉验证程序(全组件 + 亮/暗主题),可对照运行。

## 项目文档

| 文档 | 内容 |
|---|---|
| README.md | 本文:安装 / 用法 / 组件一览 |
| [docs/DESIGN.md](docs/DESIGN.md) | **自定义指南(随 `xui init` 安装为 `ui/DESIGN.md`)**:架构原理、主题系统、组件解剖、**xcgui 踩坑全集(第八节)**、已验证 API 清单 |
| [docs/HANDOFF.md](docs/HANDOFF.md) | 仓库维护者用:当前进度 / 待办 / 仓库结构 / 常用命令 / 视觉验证工作流 |
| [AGENTS.md](AGENTS.md) | 仓库维护者用:AI 会话入口硬规则与文档导航 |

> 改任何组件的交互前,先看 DESIGN.md 第四节末「与 shadcn 对齐的语义(权威汇总)」,再对照 shadcn-ui 官方源码。

## License

[MIT](LICENSE)
