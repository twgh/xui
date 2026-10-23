package ui

import (
	"strings"

	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/drawx"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Menu 下拉菜单: 载体为原生 Menu(弹出/键盘导航/子菜单/边缘翻转全部原生),
// 皮肤通过窗口级 Menu_DrawBackground / Menu_DrawItem 事件接管 —— 所有颜色
// 绘制时实时取令牌, 换主题零成本(同一窗口弹出的所有菜单共用这套皮肤)。
//
// 与 shadcn dropdown-menu.tsx(radix-nova) 对齐:
//   - 弹层: bg-popover rounded-lg p-1 ring-1 ring-foreground/10 shadow-md。
//     载体是阴影弹窗(Window_Transparent_Shadow, 官方 DrawMenu 示例同款路线):
//     圆角(ShadowInfo 的 nAngeleSize)与投影一次解决, 不再用 Win32 区域裁剪;
//     p-1 用菜单 SetBorderSize(4) 落实(默认 3);
//   - 进场动画: data-open:fade-in-0(duration-100) = 100ms **内容淡入**(文字/
//     图标/分隔线由 xui.Presence 挂宿主窗口驱动); 面板与阴影从第一帧起不透明 ——
//     不用整窗 alpha, 否则 popover 底色一起半透明, 后面的页面文字会透上来;
//     无位移(slide-in 会让 demo 崩溃, 见 menuOnPopupWnd 注释);
//     退场原生即关即销毁, 无退场动画;
//   - 条目: h-7(28) rounded-md px-1.5, hover/focus 高亮 bg-accent + 文字变
//     accent-foreground; destructive 文字红、高亮 bg-destructive/10(dark /20);
//   - 禁用: opacity-50(组级透明 → 文字用 50% alpha 合成, 且不响应);
//   - 勾选/单选条目: 选中项右侧 Check 图标(right-2 size-4), 单选组内互斥;
//   - 分组标签: text-xs muted; 分隔栏: 1px bg-border 全宽(-mx-1; 原生槽高仅
//     4px, shadcn my-1 的 9px 不可达, 已知差异);
//   - 快捷键: 右对齐 text-xs muted(文本里保留 "\t" 供原生菜单解析, 自绘时拆开渲染);
//   - 宽度: 弹层固定时对齐 shadcn `w-(--radix-dropdown-menu-trigger-width)
//     min-w-32`(Width(w) 显式覆盖, 否则 max(触发区宽, 128)); 右键菜单无触发区,
//     内容自适应 + 128 下限; 子菜单内容自适应 + min-w-96。所有条目都用
//     XMenu_SetItemWidth 按自绘字体算宽(原生字体会把长中文算窄, 见 DESIGN 99);
//     固定宽下长文本仍硬裁剪, 与 radix 的 overflow-x-hidden 一致(见 DESIGN 64);
//   - 定位: 触发元素下方左对齐 sideOffset=4(radix Content 默认), 下方放不下
//     翻到上方, 屏幕边缘夹紧;
//   - 前置图标: MenuIcon(条目/子菜单触发项, 图标 16 画在 left-1.5, 文字从 28 起,
//     与 shadcn 条目内联 SVG 的口径一致);
//   - 右键触发区: NewContextMenu(元素, build) 对应 shadcn ContextMenuTrigger ——
//     右键触发区矩形内(含子元素)弹出 build 现场构造的菜单。
//
// 用法(一次性对象, 每次弹出新建):
//
//	m := ui.NewMenu()
//	m.Item("复制", ui.MenuShortcut("Ctrl+C"), ui.MenuIcon(icons.IconCopy), ui.MenuOnSelect(func() {...}))
//	m.Check("显示行号", checked, ui.MenuOnCheck(func(on bool) {...}))
//	m.PopupAtElement(btn.Handle) // 或 m.PopupAtCursor(w.Handle)
//
// 右键触发区(每次右键重建菜单, 状态由调用方持有):
//
//	ui.NewContextMenu(area.Handle, func(m *ui.Menu) {
//	    m.Item("复制", ui.MenuIcon(icons.IconCopy), ui.MenuOnSelect(func() {...}))
//	})
type Menu struct {
	m      *widget.Menu
	items  []*menuItem
	byID   map[int32]*menuItem
	nextID int32
	parent int32 // Sub() 构建期间的父项 ID

	// 弹出锚点(物理屏幕像素); 翻转/夹紧由原生菜单自带, 不需也不应手动校正
	ax, yBelow int32
	// 触发元素宽(逻辑像素, PopupAtElement 记录; 0 = 无触发元素, 如右键菜单),
	// 用于弹层宽对齐触发区(shadcn 的 w-(--trigger-width) min-w-32)。
	triggerW int32
	// 显式弹层宽(Width 设置, 对应 shadcn 在 Content 上加 className="w-56" 这类
	// 覆盖); 0 = 走组件默认(max(触发区宽, 128))。
	width int32
}

type menuKind int8

const (
	menuKindNormal menuKind = iota
	menuKindCheck
	menuKindRadio
	menuKindLabel
	menuKindSeparator
	menuKindSub
)

type menuItem struct {
	id          int32
	text        string
	shortcut    string
	kind        menuKind
	parent      int32
	checked     bool
	group       int32
	disabled    bool
	destructive bool
	inset       bool   // 文字左缩进到与图标/勾选条目对齐(shadcn 的 data-inset:pl-7)
	icon        string // 前置图标(path/SVG, 与 Button.WithIcon 同口径)
	onSelect    func()
	onCheck     func(checked bool)
}

// MenuOpt 菜单项配置。
type MenuOpt func(*menuItem)

// MenuShortcut 菜单项右侧的快捷键文本。
func MenuShortcut(s string) MenuOpt { return func(it *menuItem) { it.shortcut = s } }

// MenuDisabled 禁用该项(灰显且不可点)。
func MenuDisabled() MenuOpt { return func(it *menuItem) { it.disabled = true } }

// MenuDestructive 破坏性操作(红字, 高亮为红色淡底)。
func MenuDestructive() MenuOpt { return func(it *menuItem) { it.destructive = true } }

// MenuGroup 单选组号: 同组 Radio 条目选中互斥(不设组号时单选条目独立切换)。
func MenuGroup(group int32) MenuOpt { return func(it *menuItem) { it.group = group } }

// MenuInset 让条目文字左缩进到 28(shadcn 的 `inset` prop = `data-inset:pl-7`):
// 与带前置图标(MenuIcon)/前置指示器的条目文字对齐。普通条目、Label、Sub 都可加;
// 绘制只挪文字与右侧图标, 高亮底仍是整行。
func MenuInset() MenuOpt { return func(it *menuItem) { it.inset = true } }

// MenuIcon 给条目(普通/勾选/单选/子菜单)画前置图标, 图标 16px 贴着条目左缘
// left-1.5, 文字从 28 起(shadcn 条目内联 SVG 的 gap-1.5 口径)。
// 图标与文字/勾选/子菜单箭头可同时存在; path 与 Button.WithIcon 同口径
// (lucide path 或整段 SVG, 见 icons 子包)。
func MenuIcon(path string) MenuOpt { return func(it *menuItem) { it.icon = path } }

// MenuOnSelect 普通条目的点击回调。
func MenuOnSelect(fn func()) MenuOpt { return func(it *menuItem) { it.onSelect = fn } }

// MenuOnCheck 勾选/单选条目的状态回调, 参数为点击后的新状态。
func MenuOnCheck(fn func(checked bool)) MenuOpt { return func(it *menuItem) { it.onCheck = fn } }

// NewMenu 创建下拉菜单。菜单是一次性对象: 弹出关闭后原生菜单即销毁,
// 状态(如勾选)由调用方持有并在下次弹出时传入。
func NewMenu() *Menu {
	return &Menu{byID: map[int32]*menuItem{}, nextID: 1}
}

func (m *Menu) add(it *menuItem, opts []MenuOpt) *Menu {
	for _, o := range opts {
		o(it)
	}
	it.id = m.nextID
	m.nextID++
	it.parent = m.parent
	m.items = append(m.items, it)
	m.byID[it.id] = it
	return m
}

// Item 添加普通菜单项。
func (m *Menu) Item(text string, opts ...MenuOpt) *Menu {
	return m.add(&menuItem{kind: menuKindNormal, text: text}, opts)
}

// Check 添加勾选菜单项: 点击后状态翻转, 新状态通过 MenuOnCheck 回调告知。
func (m *Menu) Check(text string, checked bool, opts ...MenuOpt) *Menu {
	return m.add(&menuItem{kind: menuKindCheck, text: text, checked: checked}, opts)
}

// Radio 添加单选菜单项: 与同组(MenuGroup)条目互斥, 行为同 shadcn RadioItem
// (视觉上同样是右侧 Check 图标)。组号不同或未设组号时独立切换。
func (m *Menu) Radio(text string, checked bool, opts ...MenuOpt) *Menu {
	return m.add(&menuItem{kind: menuKindRadio, text: text, checked: checked}, opts)
}

// Label 添加分组标签(不可聚焦、不可点)。可加 MenuInset 对齐带图标的条目
// (shadcn context-menu 的 ContextMenuLabel inset)。
func (m *Menu) Label(text string, opts ...MenuOpt) *Menu {
	return m.add(&menuItem{kind: menuKindLabel, text: text, disabled: true}, opts)
}

// Separator 添加分隔栏。
func (m *Menu) Separator() *Menu {
	return m.add(&menuItem{kind: menuKindSeparator}, nil)
}

// Sub 添加子菜单, build 里添加的条目都挂在该子菜单下(可嵌套)。
// 子菜单触发项可加 MenuInset / MenuIcon(shadcn ContextMenuSubTrigger 的 inset 与图标)。
func (m *Menu) Sub(text string, build func(s *Menu), opts ...MenuOpt) *Menu {
	p := &menuItem{kind: menuKindSub, text: text}
	m.add(p, opts)
	saved := m.parent
	m.parent = p.id
	build(m)
	m.parent = saved
	return m
}

// PopupAtElement 在触发元素下方弹出(radix DropdownMenuContent 默认定位:
// align=start, sideOffset=4), 下方放不下翻到上方, 屏幕边缘夹紧。
func (m *Menu) PopupAtElement(hEle int) {
	HideTooltip(hEle) // 弹层接管鼠标后触发元素收不到 MouseLeave(见 tooltip.go)
	var trig xc.RECT
	xc.XEle_GetWndClientRect(hEle, &trig)
	hw := xc.XWidget_GetHWINDOW(hEle)
	if hw <= 0 {
		return
	}
	dpi := xc.XWnd_GetDPI(hw)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96
	var mrc xc.RECT
	if !wapi.GetWindowRect(xc.XWnd_GetHWND(hw), &mrc) {
		return
	}
	const sideOffset = 4
	m.ax = mrc.Left + int32(float64(trig.Left)*sc)
	m.yBelow = mrc.Top + int32(float64(trig.Bottom)*sc) + int32(sideOffset*sc)
	m.triggerW = trig.Right - trig.Left // 逻辑像素(XEle_GetWndClientRect 未缩放)
	m.popup(hw)
}

// PopupAtCursor 在鼠标位置弹出(右键菜单用法)。hWindow 是承载菜单事件的
// 炫彩窗口句柄 —— 皮肤绘制事件注册在它上面。
func (m *Menu) PopupAtCursor(hWindow int) {
	if hWindow <= 0 {
		return
	}
	var pt wapi.POINT
	wapi.GetCursorPos(&pt)
	m.ax, m.yBelow = pt.X, pt.Y
	m.triggerW = 0 // 右键菜单无触发区, 弹层宽退化为内容自适应 + min-w-32 下限
	m.popup(hWindow)
}

// menuMinWidth 是 shadcn DropdownMenuContent 的 min-w-32(8rem = 128 逻辑像素)。
const menuMinWidth int32 = 128

// menuSubMinWidth 是 shadcn DropdownMenuSubContent 的 min-w-[96px]。
const menuSubMinWidth int32 = 96

// menuLeadSpace 是 inset / 前置图标条目的额外左内缩(文字左缘 28 vs 普通 6)。
const menuLeadSpace int32 = 22

// menuWidthOverhead 是弹窗宽与最大条目文本区宽之间的固定开销(逻辑像素):
// 由左右边框(SetBorderSize(4) ×2)与原生文本间隔等构成, 探针实测 48;
// 随 BorderSize / 文本间隔设置变化, 条目高 28 / LeftWidth(0) 配置下成立。
const menuWidthOverhead int32 = 48

// menuTargetWidth 弹层目标宽: 显式 Width 优先(shadcn 的 className="w-56" 覆盖);
// 默认 = shadcn 组件默认类 `w-(--radix-dropdown-menu-trigger-width) min-w-32`
// = max(触发区宽, 128)(radix 确实提供该变量, = popper-anchor-width); 无触发
// 元素(右键菜单)时取 128 下限 —— 只把第一个条目垫到 128−开销, 其余按内容算宽,
// 即"内容自适应 + 下限"。
func (m *Menu) menuTargetWidth() int32 {
	if m.width > 0 {
		return m.width
	}
	if m.triggerW > menuMinWidth {
		return m.triggerW
	}
	return menuMinWidth
}

// Width 显式指定弹层宽度(逻辑像素), 对应 shadcn 在 DropdownMenuContent 上加
// `className="w-56"` 这类宽度覆盖(w-56 = 224)。
// 不调用时走组件默认: 弹层宽 = max(触发区宽, 128)。
func (m *Menu) Width(w int32) *Menu {
	m.width = w
	return m
}

// ===== 右键触发区 =====

// ContextMenu 把"右键触发区"绑定到元素: 在触发区矩形内任意位置右键(含子元素),
// 都用 build 现场创建一个 Menu 并弹出 —— 对应 shadcn 的 ContextMenu +
// ContextMenuTrigger。菜单是一次性对象, 每次右键都会得到全新的 Menu(勾选等状态
// 由调用方持有, 在 build 里按当前状态重建)。
//
// 命中测试放在**窗口**右键事件上: 元素右键事件不会冒泡到父元素(探针实测, 点在
// 触发区里的子控件上时只有子控件和窗口收到), 直接绑触发区元素会漏掉子元素区域。
// 窗口右键坐标与 XEle_GetWndClientRect 同口径(逻辑像素, 原点 = 窗口矩形原点),
// 直接做矩形包含判断即可; 这与 radix "整个触发区都可右键"的语义一致。
// 代价: 触发区里的原生控件(Edit/List 等)如果吃掉了自己的右键消息, 窗口收不到,
// 这类控件上不会弹菜单 —— 右键触发区里别放需要自带右键行为的原生控件。
//
// 弹出是阻塞的(原生菜单): OnOpenChange(true) 在弹出前触发(可在这里给触发区置
// 展开态并 Redraw), 菜单关闭后触发 OnOpenChange(false)。
//
//	ui.NewContextMenu(area.Handle, func(m *ui.Menu) {
//	    m.Item("复制", ui.MenuIcon(icons.IconCopy), ui.MenuOnSelect(func() {...}))
//	}).OnOpenChange(func(open bool) {
//	    // 触发区是 Button 时可在这里 SetExpanded(open) + Redraw()
//	})
type ContextMenu struct {
	hEle         int
	hWindow      int
	build        func(*Menu)
	onOpenChange func(open bool)
	enabled      bool
}

// NewContextMenu 给 hEle 绑定右键触发区, build 负责填充每次弹出的菜单。
// hEle 必须在窗口里(能取到所属窗口)才有意义; 菜单内容不跟随触发区宽度,
// 走 Menu 的右键菜单默认宽(内容自适应 + 128 下限), 需要固定宽在 build 里 Width(w)。
func NewContextMenu(hEle int, build func(m *Menu)) *ContextMenu {
	c := &ContextMenu{hEle: hEle, build: build, enabled: true}
	if hEle <= 0 || !xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
		return c
	}
	c.hWindow = xc.XWidget_GetHWINDOW(hEle)
	if c.hWindow <= 0 {
		return c
	}
	w := window.NewByHandle(c.hWindow)
	if w == nil {
		return c
	}
	w.AddEvent_RButtonUp(func(hWindow int, _ uint, pPt *xc.POINT, pbHandled *bool) int {
		c.handleRightUp(hWindow, pPt, pbHandled)
		return 0
	})
	return c
}

// SetEnabled 启用/停用触发区(false = 右键不再弹菜单, shadcn Trigger 的 disabled)。
func (c *ContextMenu) SetEnabled(on bool) *ContextMenu {
	c.enabled = on
	return c
}

// OnOpenChange 菜单开/关回调: true 在菜单弹出前(阻塞前)触发, false 在关闭后触发。
func (c *ContextMenu) OnOpenChange(fn func(open bool)) *ContextMenu {
	c.onOpenChange = fn
	return c
}

// handleRightUp 在窗口右键事件里做触发区命中测试。
// 多个触发区重叠时由后创建者先处理(窗口事件按注册逆序执行), 处理后置 pbHandled
// 阻止其它触发区继续弹菜单。
func (c *ContextMenu) handleRightUp(hWindow int, pPt *xc.POINT, pbHandled *bool) {
	if !c.enabled || c.build == nil || pPt == nil || (pbHandled != nil && *pbHandled) {
		return
	}
	if !xc.XC_IsHXCGUI(c.hEle, xcc.XC_ELE) || !xc.XEle_IsEnable(c.hEle) || !xc.XWidget_IsShow(c.hEle) {
		return
	}
	var rc xc.RECT
	xc.XEle_GetWndClientRect(c.hEle, &rc)
	if pPt.X < rc.Left || pPt.X >= rc.Right || pPt.Y < rc.Top || pPt.Y >= rc.Bottom {
		return
	}
	if pbHandled != nil {
		*pbHandled = true
	}
	HideTooltip(c.hEle)
	if c.onOpenChange != nil {
		c.onOpenChange(true)
	}
	m := NewMenu()
	c.build(m)
	m.PopupAtCursor(hWindow)
	if c.onOpenChange != nil {
		c.onOpenChange(false)
	}
}

func (m *Menu) popup(hWindow int) {
	m.m = m.build(hWindow)
	// 初始位置即锚点(元素矩形是精确的)。
	// ⚠️ 不要在这里做任何位置校正: XMenu_Popup 是阻塞的(菜单关闭后才返回),
	// 校正会拿到已销毁的弹窗句柄(报「输入句柄可能无效」)。
	// Menu_PopupWnd 回调里的窗口矩形外扩/进场动画位移是官方 DrawMenu 示例
	// 同款模式, 探针实测不影响子菜单锚点; 但"把弹窗挪去别的位置"这类校正
	// 仍然不要做(历史踩坑: 二级菜单弹到父菜单原点盖住它)。
	// 翻转/夹紧/子菜单定位原生自带, 锚点给对即可。
	m.m.Popup(xc.XWnd_GetHWND(hWindow), m.ax, m.yBelow, 0, xcc.Menu_Popup_Position_Left_Top)
}

// build 创建原生菜单并同步条目。
// 快捷键拼进文本("\t" 分隔): 原生按整串计算自适应宽度并为快捷键预留区域
// (探针实测 NShortcutKeyWidth=37), 绘制时再拆开分别渲染。
func (m *Menu) build(hWindow int) *widget.Menu {
	menu := widget.NewMenu()
	menu.SetAutoDestroy(true)
	menu.SetItemHeight(28) // 条目 = text-sm(20) + py-1(4×2)
	menu.SetLeftWidth(0)   // 不留原生左侧图标条: 前置图标与右侧勾/箭头都由自绘负责
	// p-1(4px): 默认边距是 3, 条目高亮距弹层左右边缘原差 1px; 设 4 后条目盒
	// 与 shadcn 完全一致(内容内缩 4, 文字再 px-1.5)。
	menu.SetBorderSize(4, 4, 4, 4)
	menu.EnableDrawBackground(true)
	menu.EnableDrawItem(true)
	for _, it := range m.items {
		text := it.text
		if it.shortcut != "" {
			text += "\t" + it.shortcut
		}
		flags := xcc.Menu_Item_Flag_Normal
		switch {
		case it.kind == menuKindSeparator:
			flags = xcc.Menu_Item_Flag_Separator
		case it.disabled: // Label 借用禁用态: 不高亮、不可点
			flags = xcc.Menu_Item_Flag_Disable
		case (it.kind == menuKindCheck || it.kind == menuKindRadio) && it.checked:
			flags = xcc.Menu_Item_Flag_Check
		}
		menu.AddItem(it.id, text, it.parent, flags)
	}
	// 弹层宽: 依据探针实测(见 DESIGN 64), 弹窗宽 = max(各条目文本区宽) + 固定开销。
	// 所有条目都显式设宽(不能交给原生自然宽: 原生字体与自绘不一致会裁长文本):
	//
	//   - 顶级菜单有显式 Width / 触发区宽(≥128): 每个顶级条目都设"目标宽 − 开销"
	//     (shadcn dropdown-menu.tsx 的 `w-(--trigger-width) min-w-32`);
	//   - 顶级菜单无触发区(右键菜单): 内容自适应 + min-w-32(第一个条目垫到 80);
	//   - 子菜单不跟触发区宽(shadcn SubContent 内容自适应 + min-w-96, 第一项垫到 48);
	//   - inset/图标 +22、Check/Radio/Sub 右侧恒 +24、快捷键 +12+小字宽, 逐项算。
	m.applyItemWidths(menu)
	ensureMenuSkin(hWindow)
	menuSkin.active = m
	menu.AddEvent_Menu_Select(hWindow, func(_ int, nID int32, _ *bool) int {
		m.dispatch(nID)
		return 0
	}, false)
	menu.AddEvent_Menu_Exit(hWindow, func(_ int, _ *bool) int {
		if menuSkin.active == m {
			menuSkin.active = nil
		}
		return 0
	}, false)
	return menu
}

// applyItemWidths 见 build 里的宽度说明。fixed = 顶级弹层走定宽(显式 Width 或
// 触发区宽 ≥ 128); 其余条目按分组(0 = 顶级, 其余 = 子菜单)按自绘字体逐项算宽。
//
// ⚠️ 不能留自然宽: 原生菜单按它自己的字体量文本, 与自绘的 14px 字体不一致 ——
// 长文本会按较窄的原生宽度撑弹层, 自绘时尾部被裁掉(探针截图实测)。
// 这里统一用 xui.TextWidth(自绘同字体)算"文本区宽":
//   宽 = 前置保留(inset/图标 22) + 文字 + 快捷键(12 间隔 + 小字宽)
//        + 右侧保留(Check/Radio/Sub 恒 24, 对齐 shadcn pr-8)。
// 弹窗宽 = max(条目宽)+48, 条目盒内的文本区比该值再多 28px 余量(见 48 的构成),
// 因此按此公式算出的宽足以容纳自绘内容。
func (m *Menu) applyItemWidths(menu *widget.Menu) {
	t := xui.Current()
	font := xui.FontHandle(t.FontDefault)
	smallFont := xui.FontHandle(xui.FontSpec{
		Family: t.FontDefault.Family,
		Size:   t.FontDefault.Size - 2, // text-xs
	})
	fixed := m.width > 0 || m.triggerW > menuMinWidth
	fixedW := m.menuTargetWidth() - menuWidthOverhead
	seen := map[int32]bool{}
	for _, it := range m.items {
		if it.kind == menuKindSeparator {
			continue
		}
		first := !seen[it.parent]
		seen[it.parent] = true
		var w int32
		if it.parent == 0 && fixed {
			w = fixedW
		} else {
			textFont := font
			if it.kind == menuKindLabel {
				textFont = smallFont
			}
			w = xui.TextWidth(it.text, textFont)
			if it.inset || it.icon != "" {
				w += menuLeadSpace
			}
			if it.shortcut != "" {
				w += 12 + xui.TextWidth(it.shortcut, smallFont)
			}
			if it.kind == menuKindCheck || it.kind == menuKindRadio || it.kind == menuKindSub {
				w += 24
			}
			minW := menuMinWidth - menuWidthOverhead
			if it.parent != 0 {
				minW = menuSubMinWidth - menuWidthOverhead
			}
			if first && w < minW {
				w = minW
			}
		}
		if w < 0 { // Width 小于固定开销时别把负宽喂给原生
			w = 0
		}
		menu.SetItemWidth(it.id, w)
	}
}

// dispatch 分发选择事件: Check 翻转, Radio 置真并清同组, 其余走 onSelect。
// 菜单关闭后原生对象销毁, 状态只保留在 Go 侧条目上, 由调用方回调承接。
func (m *Menu) dispatch(nID int32) {
	it := m.byID[nID]
	if it == nil {
		return
	}
	switch it.kind {
	case menuKindCheck:
		it.checked = !it.checked
		if it.onCheck != nil {
			it.onCheck(it.checked)
		}
	case menuKindRadio:
		it.checked = true
		for _, other := range m.items {
			if other != it && other.kind == menuKindRadio && other.group == it.group {
				other.checked = false
			}
		}
		if it.onCheck != nil {
			it.onCheck(true)
		}
	default:
		if it.onSelect != nil {
			it.onSelect()
		}
	}
}

// ===== 皮肤(窗口级, 同一窗口弹出的所有菜单共用) =====

var menuSkin = struct {
	windows map[int]bool
	active  *Menu
	// 顶级弹层的内容淡入进度(0..1)与所属弹窗。面板/阴影始终不透明, 只淡文字/图标,
	// 避免整窗 alpha 把底色也变透明(后面的页面文字会透上来, 看起来像闪烁)。
	fade    float64
	fadeWin int
	// fadeToken 防止"上一次弹层的淡入 apply"在窗口句柄被复用后继续写新弹层的值;
	// 每次弹出取新 token, 旧过渡的 apply/done 发现 token 不匹配就直接退出。
	fadeToken uint64
	fadeSeq   uint64
}{windows: map[int]bool{}, fade: 1}

// ensureMenuSkin 在承载窗口上注册菜单绘制事件(每窗口一次)。
// 绘制回调全部实时取令牌, 换主题后下一次弹出自动是新皮肤。
func ensureMenuSkin(hWindow int) {
	if menuSkin.windows[hWindow] {
		return
	}
	w := window.NewByHandle(hWindow)
	if w == nil {
		return
	}
	menuSkin.windows[hWindow] = true
	w.AddEvent_Menu_PopupWnd(menuOnPopupWnd)
	w.AddEvent_Menu_DrawBackground(menuDrawBackground)
	w.AddEvent_Menu_DrawItem(menuDrawItem)
	w.AddEvent_Menu_Exit(func(_ int, _ *bool) int {
		menuSkin.active = nil
		return 0
	})
}

// menuShadowSize 是顶级菜单弹窗阴影带的宽度(逻辑像素) —— Window_Transparent_Shadow
// 会把客户区四周各收缩这么多, 窗口矩形要同步外扩让 Body(菜单内容区)回到原位。
// 取 5 对齐 shadcn shadow-md(0 4px 6px -1px black/10%: 侧向可见约 5px)。
const menuShadowSize int32 = 5

// menuShadowSizeSub 是子菜单弹窗阴影带宽: shadcn 子菜单用 shadow-lg
// (0 10px 15px -3px black/10%, 侧向可见约 12px), 均匀环带引擎下取 8 折中。
const menuShadowSizeSub int32 = 8

// menuShadowDepth 是阴影深度(0-255), 与 shadow-md/lg 的 10% 峰值黑度对齐
// (阴影引擎在白底上的边缘浓度约 depth/1000)。
const menuShadowDepth int32 = 50

// menuOnPopupWnd 弹窗出现: 转阴影弹窗(圆角+投影), 顶级弹窗播内容淡入,
// 子菜单即时出现(见下方说明)。
// 窗口矩形外扩在本回调里做(官方 DrawMenu 示例同款), 探针实测子菜单锚点不受影响。
//
// ⚠️ 只有淡入, 没有 radix 的 slide-in 位移: 对菜单弹窗做 SetWindowPos 移动
// (哪怕只在进场那一瞬、SWP_NOSIZE|NOZORDER|NOACTIVATE)会让 demo 直接崩溃
// (DLL 内部访问违例, 静默退出; 官方示例的 SetRect 外扩没事, 单独的"移动"不行,
// 探针+demo 双向验证)。100ms 的内容淡入与 radix fade-in-0 观感一致, 位移省略。
func menuOnPopupWnd(hWindow, hMenu int, pInfo *xc.Menu_PopupWnd_, pbHandled *bool) int {
	if pInfo == nil || pInfo.HWindow <= 0 {
		return 0
	}
	mw := window.NewByHandle(pInfo.HWindow)
	if mw == nil {
		return 0
	}
	t := xui.Current()

	// 阴影弹窗: nAngeleSize = rounded-lg, 圆角由阴影窗口自身按逐像素 alpha 画出
	// (背景/条目照常填充, 框架自动裁到圆角 Body), 不再需要 SetWindowRound 区域裁剪。
	// 阴影带宽: 顶级 = shadow-md(5), 子菜单 = shadow-lg(8), 对齐 shadcn。
	band := menuShadowSize
	if pInfo.NParentID != 0 {
		band = menuShadowSizeSub
	}
	mw.SetTransparentType(xcc.Window_Transparent_Shadow)
	mw.SetShadowInfo(band, menuShadowDepth, t.RadiusLg(), false, xc.RGBA(0, 0, 0, 128))
	mw.SetTransparentAlpha(255)
	// 客户区已收缩出阴影带: 窗口矩形四周外扩同尺寸, Body 对齐原生给定的菜单位置。
	rc := mw.GetRectEx()
	mw.SetRect(&xc.RECT{
		Left:   rc.Left - band,
		Top:    rc.Top - band,
		Right:  rc.Right + band,
		Bottom: rc.Bottom + band,
	})

	// 进场(100ms): 只淡**内容**(文字/图标/快捷键/分隔线), 面板与阴影从第一帧起
	// 就不透明。
	//
	// ⚠️ 不要用整窗 alpha 做菜单进场: 整窗 alpha 会把 popover 底色一起变半透明,
	// 底下的页面文字在淡入过程中会透到菜单上, 到 alpha=255 才被"突然"盖住 ——
	// 观感是菜单上闪过后面的文字。内容淡入只是自绘颜色的 alpha, 不经过窗口合成层,
	// 没有这种穿透, 也没有整窗重合成的帧抖动。
	//
	// 过渡挂宿主窗口驱动 —— 弹层小窗自己做重绘目标时定时器掉帧(DESIGN 47);
	// ⚠️ apply 每帧必须先验证弹窗还活着(菜单可能在过渡途中被关闭、弹窗随之销毁),
	// 拿失效句柄调 X API 会让整个进程静默崩溃(官方 DrawMenu 示例在 KillFocus 里
	// 同样用 XC_IsHXCGUI 做守卫)。
	//
	// 子菜单(NParentID != 0)不做淡入: 悬停到展开本身有 xcgui 固化的 ~330ms
	// 内部延迟(纯净菜单探针实测同值, 无配置 API, 输入注入也无法提前触发),
	// 再叠 100ms 动画会明显比 shadcn 慢一截 —— 子菜单直接全内容出现。
	if pInfo.NParentID == 0 {
		// ⚠️ 必须拷贝句柄值而不是捕获 pInfo 指针: 该结构体是原生栈对象, 回调返回后
		// 会被复用/清空, 过渡帧里再解引用拿到的是 0 或垃圾(0 → 淡入永远不动,
		// 垃圾 → 拿无效句柄调 X API 直接崩进程, demo 实测)。
		hw := pInfo.HWindow
		// 首帧先让内容透明(面板已经是不透明底), 避免"全亮一帧再变暗"。
		menuSkin.fadeSeq++
		token := menuSkin.fadeSeq
		menuSkin.fadeToken = token
		menuSkin.fade = 0
		menuSkin.fadeWin = hw
		// 键取全局唯一(不要用本组件自己的计数器: 各组件都从 1 开始会撞车, 见
		// xui.NextPresenceKey 的说明)。
		key := xui.NextPresenceKey()
		xui.Presence(hWindow, key, true, xui.PopupDuration, func(v float64) {
			if menuSkin.fadeToken != token || !xc.XC_IsHXCGUI(hw, xcc.XC_WINDOW) {
				return
			}
			menuSkin.fade = v
			menuSkin.fadeWin = hw
			xc.XWnd_Redraw(hw, true)
		}, func(float64) {
			// 到位后恢复"不参与淡入"的默认值; 下次弹出会重新置 0。
			if menuSkin.fadeToken == token {
				menuSkin.fade = 1
				menuSkin.fadeWin = 0
			}
		})
	}
	return 0
}

// menuDrawBackground 弹层底色 + ring-1 ring-foreground/10。
//
// 坐标系(标定实验实测): 绘制上下文原点 = 弹窗窗口左上角(逻辑像素),
// XWnd_GetClientRect 返回的 {8,8,168,276} 就是阴影窗的 Body(可见白区)
// 在窗口内的矩形 —— 它与窗口矩形差一圈阴影带, 但 fill/ring 都画在它上面
// 就是正确的(探针色块标定验证)。ring 直接画在 Body 边界上: 线宽的一半
// 落在体外被阴影 alpha 蒙版裁掉, 肉眼是一条贴着窗口边缘的线(shadcn 边框
// 贴边), 圆角与 Body 的 nAngeleSize(RadiusLg) 一致, 四角不脱开。
// ⚠️ 画在 Body 矩形之外(阴影带里)的任何像素都会被蒙版压暗至不可见。
func menuDrawBackground(_ int, hDraw int, pInfo *xc.Menu_DrawBackground_, pbHandled *bool) int {
	*pbHandled = true
	d := drawx.NewByHandle(hDraw)
	if d == nil {
		return 0
	}
	d.EnableSmoothingMode(true)
	t := xui.Current()
	var rc xc.RECT
	xc.XWnd_GetClientRect(pInfo.HWindow, &rc)
	d.SetBrushColor(uint32(t.Popover))
	d.FillRect(&rc)
	d.SetLineWidth(1)
	d.SetBrushColor(uint32(t.Foreground.WithAlpha(26)))
	d.DrawRoundRect(&rc, t.RadiusLg(), t.RadiusLg())
	return 0
}

// menuDrawItem 自绘菜单项。
// 项矩形几何(SetBorderSize(4) 后探针实测): 四周各 4px 边距(= shadcn p-1),
// 高度即 SetItemHeight(28), 分隔栏槽高固定 4px。
func menuDrawItem(_ int, hDraw int, pInfo *xc.Menu_DrawItem_, pbHandled *bool) int {
	*pbHandled = true
	d := drawx.NewByHandle(hDraw)
	if d == nil {
		return 0
	}
	d.EnableSmoothingMode(true)
	p := &xui.Painter{Draw: d}
	t := xui.Current()
	st := pInfo.NState
	rc := pInfo.RcItem

	var it *menuItem
	if am := menuSkin.active; am != nil {
		it = am.byID[pInfo.NID]
	}
	// 进场内容淡入: 只有顶级弹窗参与(面板/阴影始终不透明, 见 menuOnPopupWnd)。
	fv := 1.0
	if pInfo.HWindow == menuSkin.fadeWin {
		fv = menuSkin.fade
		if fv < 0 {
			fv = 0
		} else if fv > 1 {
			fv = 1
		}
	}

	// 分隔栏: 1px bg-border 全宽(shadcn -mx-1: 越过 p-1 边距铺满弹层),
	// 居中画在原生固定的 4px 槽内
	if st&xcc.MenuItem_State_Flag_Separator > 0 {
		d.SetLineWidth(1)
		d.SetBrushColor(uint32(t.Border.ScaleAlpha(fv)))
		d.DrawLine(rc.Left-4, rc.Top+2, rc.Right+4, rc.Top+2)
		return 0
	}

	disabled := st&xcc.MenuItem_State_Flag_Disable > 0
	// ⚠️ 不要用 MenuItem_State_Flag_Popup 判断"子菜单已展开": 它是"该项有子菜单"
	// 的静态标记(所有 Sub 项都带), 并进来会让每个子菜单触发行常亮(截图实测)。
	hot := st&xcc.MenuItem_State_Flag_Stay > 0

	// 文字与高亮配色(dropdown-menu.tsx):
	//   普通: text-popover-foreground, hot: bg-accent + text-accent-foreground
	//   destructive: text-destructive, hot: bg-destructive/10(dark /20), 文字不变
	//   disabled: opacity-50(组级透明), 不参与高亮
	//   Label: text-xs muted, 永不高亮(原生已置禁用)
	fore := t.PopoverForeground
	hotBg := t.Accent
	isDestructive := it != nil && it.destructive
	if isDestructive {
		fore = t.Destructive
		hotBg = t.Destructive.WithAlpha(26)
		if t.IsDark() {
			hotBg = t.Destructive.WithAlpha(51)
		}
	}
	if disabled {
		fore = fore.WithAlpha(128)
		hot = false
	}
	isLabel := it != nil && it.kind == menuKindLabel
	// 悬停底与字色按过渡值淡入淡出。载体是原生菜单弹窗(pInfo.HWindow): 鼠标在条目
	// 间移动时原生自己会重绘, 过渡值由 xui 按"窗口 + 条目 ID"保管并驱动补帧。
	hv := xui.HoverItem(pInfo.HWindow, pInfo.NID, hot && !isLabel)
	if hv > 0 {
		p.FillRound(&rc, hotBg.ScaleAlpha(hv*fv), t.RadiusSm())
		if !isDestructive {
			fore = xui.Fade(fore, t.AccentForeground, hv)
		}
	}
	fore = fore.ScaleAlpha(fv)

	name, shortcut := menuItemText(it, pInfo)
	// Checkbox/Radio 无论是否勾选都按 pr-8 给右侧指示器留位(shadcn 的固定 pr-8),
	// 否则同一组条目在勾选/未勾选之间文字右缘会跳。
	reserveTrailing := it != nil && (it.kind == menuKindCheck || it.kind == menuKindRadio ||
		it.kind == menuKindSub)
	drawTrailing := it != nil && ((it.kind == menuKindCheck || it.kind == menuKindRadio) && it.checked ||
		it.kind == menuKindSub)
	hasLead := it != nil && it.icon != ""

	// 前置图标: left-1.5(6) 起画 16px, 文字从 28 起(6+16+6, shadcn gap-1.5)。
	// 颜色: muted-foreground, hover 跟随 accent-foreground, destructive 红,
	// disabled 按组级 50% 透明(同文字口径)。
	if hasLead {
		ic := t.MutedForeground
		if isDestructive {
			ic = t.Destructive
		}
		if hv > 0 && !isDestructive {
			ic = xui.Fade(ic, t.AccentForeground, hv)
		}
		if disabled && !isLabel { // Label 借用禁用态只是不可点, 视觉仍是 muted 全色
			ic = ic.WithAlpha(128)
		}
		ic = ic.ScaleAlpha(fv)
		iconRc := xc.RECT{
			Left:   rc.Left + 6,
			Top:    rc.Top + (rc.Bottom-rc.Top-16)/2,
			Right:  rc.Left + 22,
			Bottom: rc.Top + (rc.Bottom-rc.Top+16)/2,
		}
		p.Icon(it.icon, &iconRc, ic, 2)
	}

	// 文本: px-1.5; 右侧有图标(勾/子菜单箭头)时按 pr-8 让位;
	// inset 或带前置图标的条目按 shadcn 的 `data-inset:pl-7` 左缘 = 条目矩形 + 28。
	textRc := rc
	textRc.Left += 6
	if it != nil && (it.inset || hasLead) {
		textRc.Left = rc.Left + 28
	}
	textRc.Right -= 6
	if reserveTrailing {
		textRc.Right -= 24
	}
	if isLabel {
		d.SetFont(xui.FontHandle(xui.FontSpec{
			Family: t.FontDefault.Family,
			Size:   t.FontDefault.Size - 2, // text-xs
		}))
		d.SetBrushColor(uint32(t.MutedForeground.ScaleAlpha(fv)))
	} else {
		d.SetFont(xui.FontHandle(t.FontDefault))
		d.SetBrushColor(uint32(fore))
	}
	d.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter | xcc.TextFormatFlag_NoWrap)
	// 长文本硬裁剪(shadcn overflow-x-hidden): 宽度对齐触发区后弹层不再按内容
	// 撑宽, 超出文本区的墨迹裁掉, 与 radix 行为一致(原生条目文本同样裁在弹层
	// 边缘, 探针截图验证)。
	d.SetClipRect(&textRc)
	d.DrawText(name, &textRc)
	d.ClearClip()

	// 快捷键: 右对齐 text-xs muted(hot 时跟随 accent-foreground)
	if shortcut != "" {
		sc := t.MutedForeground
		if hv > 0 {
			sc = xui.Fade(sc, t.AccentForeground, hv)
		}
		if disabled {
			sc = sc.WithAlpha(128)
		}
		sc = sc.ScaleAlpha(fv)
		sr := rc
		sr.Right -= 6
		if reserveTrailing {
			sr.Right -= 24
		}
		d.SetFont(xui.FontHandle(xui.FontSpec{
			Family: t.FontDefault.Family,
			Size:   t.FontDefault.Size - 2, // text-xs
		}))
		d.SetBrushColor(uint32(sc))
		d.SetTextAlign(xcc.TextAlignFlag_Right | xcc.TextAlignFlag_Vcenter | xcc.TextFormatFlag_NoWrap)
		d.DrawText(shortcut, &sr)
	}

	// 右侧图标: 勾选标记 / 子菜单箭头(right-2, size-4)
	if drawTrailing {
		iconRc := xc.RECT{
			Left:   rc.Right - 24,
			Top:    rc.Top + (rc.Bottom-rc.Top-16)/2,
			Right:  rc.Right - 8,
			Bottom: rc.Top + (rc.Bottom-rc.Top+16)/2,
		}
		if it.kind == menuKindSub {
			p.Icon(icons.IconChevronRight, &iconRc, fore, 2)
		} else {
			p.Icon(icons.IconCheck, &iconRc, fore, 2)
		}
	}
	return 0
}

// menuItemText 取条目文本: 优先用 Go 侧条目定义(名字/快捷键分开);
// 未知条目(如外部代码弹的原生菜单)从 PText 按 "\t" 拆分兜底。
func menuItemText(it *menuItem, pInfo *xc.Menu_DrawItem_) (name, shortcut string) {
	if it != nil {
		return it.text, it.shortcut
	}
	s := ""
	if pInfo.PText != 0 {
		s = common.UintPtrToString(pInfo.PText)
	}
	if i := strings.IndexByte(s, '\t'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
