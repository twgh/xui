package ui

import (
	"math"
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
//   - 进场动画: **不做**(radix 的 fade-in-0 + zoom-in-95 在桌面分层窗口上两种做法
//     都会闪烁 —— 整窗 alpha 会透出背景文字, 只淡内容会先出一帧空白面板; 位移/缩放
//     会崩进程, 见 DESIGN 64/99)。菜单与原生菜单一样即时全内容出现, 衔接交给触发区
//     的 hover / 展开态; 退场原生即关即销毁;
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

	// 弹出锚点(物理屏幕像素): 触发元素矩形(右键菜单 = 光标点)。
	// 弹层位置以它为准, 在 Popup_Wnd 回调里统一校正(原生定位在副屏上会错,
	// 见 menuPopupBody)。
	anchor xc.RECT
	// 每个分组弹层 Body 的顶(逻辑像素): 键 = 分组 ID(0 = 顶级, 其余 = Sub 触发项 id)。
	// 子菜单的 Y 必须按"父弹层顶 + 该行偏移"自己算(见 menuOnPopupWnd)。
	groupTop map[int32]int32
	// 触发元素宽(逻辑像素, PopupAtElement 记录; 0 = 无触发元素, 如右键菜单),
	// 用于弹层宽对齐触发区(shadcn 的 w-(--trigger-width) min-w-32)。
	triggerW int32
	// 显式弹层宽(Width 设置, 对应 shadcn 在 Content 上加 className="w-56" 这类
	// 覆盖); 0 = 走组件默认(max(触发区宽, 128))。
	width int32
}

// NewMenu 创建下拉菜单。菜单是一次性对象: 弹出关闭后原生菜单即销毁,
// 状态(如勾选)由调用方持有并在下次弹出时传入。
func NewMenu() *Menu {
	return &Menu{byID: map[int32]*menuItem{}, groupTop: map[int32]int32{}, nextID: 1}
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
	m.anchor = xc.RECT{
		Left:   mrc.Left + int32(float64(trig.Left)*sc),
		Top:    mrc.Top + int32(float64(trig.Top)*sc),
		Right:  mrc.Left + int32(float64(trig.Right)*sc),
		Bottom: mrc.Top + int32(float64(trig.Bottom)*sc),
	}
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
	m.anchor = xc.RECT{Left: pt.X, Top: pt.Y, Right: pt.X, Bottom: pt.Y}
	m.triggerW = 0 // 右键菜单无触发区, 弹层宽退化为内容自适应 + min-w-32 下限
	m.popup(hWindow)
}

// menuSideOffset 是弹层与锚点的间距(radix DropdownMenuContent 的 sideOffset=4,
// 逻辑像素)。
const menuSideOffset int32 = 4

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
	// 传给原生的锚点只决定它的初始落位; 最终位置在 Menu_PopupWnd 回调里统一校正
	// (原生在副屏上会错位, 见 menuPopupBody)。这里不要做任何事后校正: XMenu_Popup
	// 是阻塞的、返回时弹窗已销毁, 那时再碰句柄会报「输入句柄可能无效」。
	dpi := xc.XWnd_GetDPI(hWindow)
	if dpi <= 0 {
		dpi = 96
	}
	gap := int32(math.Round(float64(menuSideOffset) * float64(dpi) / 96))
	m.m.Popup(xc.XWnd_GetHWND(hWindow), m.anchor.Left, m.anchor.Bottom+gap, 0,
		xcc.Menu_Popup_Position_Left_Top)
}

// build 创建原生菜单并同步条目。
// 快捷键拼进文本("\t" 分隔): 原生按整串计算自适应宽度并为快捷键预留区域
// (探针实测 NShortcutKeyWidth=37), 绘制时再拆开分别渲染。
func (m *Menu) build(hWindow int) *widget.Menu {
	menu := widget.NewMenu()
	menu.SetAutoDestroy(true)
	menu.SetItemHeight(menuRowH) // 条目 = text-sm(20) + py-1(4×2); 子菜单行偏移也用它(见 rowBodyTop)
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
//
//	宽 = 前置保留(inset/图标 22) + 文字 + 快捷键(12 间隔 + 小字宽)
//	     + 右侧保留(Check/Radio/Sub 恒 24, 对齐 shadcn pr-8)。
//
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
}{windows: map[int]bool{}}

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

// menuPopupBody 计算顶级菜单弹窗 Body(可见卡片)的左上角(逻辑像素):
// 以菜单锚点(物理)为基准 —— 默认贴锚点下边、左缘对齐(radix align=start、
// sideOffset=4), 下方放不下翻到锚点上方, 最后夹进**弹窗自己所在显示器**的工作区。
//
// 为什么必须自己算: 原生 XMenu_Popup 在副屏(显示器原点非 0,0)上会拿
// "显示器矩形 × DPI"当边界做钳制 —— 150% 缩放下显示器左缘 2570 被当成 3855,
// 锚点落在钳制线左侧的弹窗被强行推到 3855(实测 1237px 级偏移, 位置输入怎么给
// 都绕不过: 给多少都钳到同值); 反过来在主屏上边界被放大到 1.5 倍, 贴屏幕
// 右/下边的弹层会溢出屏幕也不夹。翻到上方用 radix 的 side 翻转语义
// (弹层下边 = 锚点上边 − sideOffset)。
//
// 工作区按**弹窗自己的句柄**取, 不按宿主窗口: 宿主窗口横跨两块显示器时
// MonitorFromWindow 返回"占多数"的那块, 而锚点(触发区)可能在另一块上 ——
// 实测窗口压在两屏交界时, 按宿主取会把弹层夹到另一块屏的顶边(y=857)。
// 原生已经把弹窗放在锚点附近, 它落在哪块屏就是哪块屏。
func menuPopupBody(mw *window.Window, m *Menu, rc xc.RECT) (int32, int32) {
	wa, ok := xui.MonitorWorkArea(xc.XWnd_GetHWND(mw.Handle))
	if !ok {
		return rc.Left, rc.Top // 取不到显示器信息: 保持原生落位
	}
	dpi := xc.XWnd_GetDPI(mw.Handle)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96
	bw := int32(math.Round(float64(rc.Right-rc.Left) * sc)) // Body 物理宽(GetRectEx 是逻辑尺寸)
	bh := int32(math.Round(float64(rc.Bottom-rc.Top) * sc))
	gap := int32(math.Round(float64(menuSideOffset) * sc))
	x, y := m.anchor.Left, m.anchor.Bottom+gap
	if y+bh > wa.Bottom && m.anchor.Top-gap-bh >= wa.Top {
		y = m.anchor.Top - gap - bh
	}
	if y+bh > wa.Bottom {
		y = wa.Bottom - bh
	}
	if y < wa.Top {
		y = wa.Top
	}
	if x+bw > wa.Right {
		x = wa.Right - bw
	}
	if x < wa.Left {
		x = wa.Left
	}
	left := int32(math.Round(float64(x) / sc))
	top := int32(math.Round(float64(y) / sc))
	return left, top
}

// menuRowH / menuSepH 与 build 里的原生设置一一对应: 条目高 SetItemHeight(28)、
// 分隔栏槽高固定 4(逻辑像素)。
const (
	menuRowH = 28
	menuSepH = 4
)

// rowBodyTop 算"挂在条目 id 上的子菜单"该落在的 Body 顶(逻辑像素):
// 父弹层顶 + 该条目在同组内的行偏移(= 原生 align=start + alignOffset=-4,
// 子菜单首行与触发行齐平)。返回的 id2 是该条目自己的 id = 子菜单分组的键。
func (m *Menu) rowBodyTop(id int32) (top int32, id2 int32, ok bool) {
	var target *menuItem
	for _, it := range m.items {
		if it.id == id {
			target = it
			break
		}
	}
	if target == nil {
		return 0, 0, false
	}
	base, ok := m.groupTop[target.parent]
	if !ok {
		return 0, 0, false
	}
	var off int32
	for _, it := range m.items {
		if it == target {
			break
		}
		if it.parent != target.parent {
			continue
		}
		if it.kind == menuKindSeparator {
			off += menuSepH
		} else {
			off += menuRowH
		}
	}
	// 行顶 = Body 顶 + 内边距 4, 子菜单自身也有 4 的内边距 → 相抵后 Body 顶
	// = 父 Body 顶 + 行偏移, 首行正好压在触发行上(原生实测同值)。
	return base + off, target.id, true
}

// menuClampSubTop 把子菜单夹进它自己所在显示器的工作区(radix avoidCollisions:
// 触发行贴近屏幕底边时整体上移, 不裁条目)。
func menuClampSubTop(mw *window.Window, top, h int32) int32 {
	wa, ok := xui.MonitorWorkArea(xc.XWnd_GetHWND(mw.Handle))
	if !ok {
		return top
	}
	dpi := xc.XWnd_GetDPI(mw.Handle)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96
	bot := int32(math.Round(float64(wa.Bottom) / sc))
	t0 := int32(math.Round(float64(wa.Top) / sc))
	if top+h > bot {
		top = bot - h
	}
	if top < t0 {
		top = t0
	}
	return top
}

// menuOnPopupWnd 弹窗出现: 转阴影弹窗(圆角 + 投影)、把窗口矩形四周外扩补回阴影带,
// 并**重新落位**: 顶级弹窗以锚点为准重算位置(见 menuPopupBody); 子菜单以"父弹窗
// 实际顶 + 触发行偏移"重算 Y, X 保留原生(原生把它锚在父弹窗的实际矩形右缘, 会跟随)。
// 菜单与子菜单都即时全内容出现, 这里不做进场动画(原因见函数末尾说明与 DESIGN 99)。
//
// ⚠️ 挪窗只能用 SetRect(本回调里做, 官方 DrawMenu 示例的阴影带外扩同款路径,
// 大位移安全、命中判定跟随 —— 探针验证); 对菜单弹窗做 SetWindowPos 移动
// (哪怕只在进场那一瞬、SWP_NOSIZE|NOZORDER|NOACTIVATE)会让 demo 直接崩溃
// (DLL 内部访问违例, 静默退出; 探针+demo 双向验证)。radix 的
// slide-in-from-* 与 zoom-in-95 因此都不可实现。
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
	// 客户区已收缩出阴影带: 窗口矩形四周外扩同尺寸, Body 对齐菜单内容区位置。
	// 顶级弹窗按锚点重算落位(native 给的落位在副屏上不可信, 见 menuPopupBody);
	// 子菜单的 Y 也自己算, X 保留原生(探针实测原生把子菜单 X 锚在父弹窗的**实际
	// 矩形**右缘, 会跟随)。Y 不能只补位移: 原生算子菜单位置时把它的底边夹进
	// "父弹窗底边"的**缓存值**(= 我们挪动之前的位置), 父级被翻到上方过就整体
	// 下移一整行(见 DESIGN 134)。
	rc := mw.GetRectEx()
	left, top := rc.Left, rc.Top
	m := menuSkin.active
	if m != nil {
		if pInfo.NParentID == 0 {
			left, top = menuPopupBody(mw, m, rc)
			m.groupTop[0] = top
		} else if bt, gid, ok := m.rowBodyTop(pInfo.NParentID); ok {
			top = menuClampSubTop(mw, bt, rc.Bottom-rc.Top)
			m.groupTop[gid] = top
		}
	}
	mw.SetRect(&xc.RECT{
		Left:   left - band,
		Top:    top - band,
		Right:  left + (rc.Right - rc.Left) + band,
		Bottom: top + (rc.Bottom - rc.Top) + band,
	})

	// 进场**不做任何淡入**(录屏实测后的结论):
	//   - 整窗 alpha: popover 底色一起半透明, 淡入过程中后面的页面文字透到菜单上,
	//     alpha 到 255 才被盖住 —— 观感是"菜单上闪过背景字";
	//   - 只淡内容(文字/图标, 面板保持不透明): 第一帧就是一块**空白面板**(或极淡
	//     的文字), 录屏里清楚可见, 同样是闪烁;
	//   - 位移/缩放做不到: 对菜单弹窗 SetWindowPos 会直接崩进程(见 DESIGN 64)。
	// 原生菜单本来就是即时出现, 这里保持一致; 交互上的"衔接"交给触发区的
	// hover / 展开态(见 ContextMenu 与 demo 的 contextTrigger)。
	//
	// 子菜单同样即时出现: 悬停到展开本身有 xcgui 固化的 ~330ms 内部延迟(纯净菜单
	// 探针实测同值, 无配置 API, 输入注入也无法提前触发), 再叠动画会明显比 shadcn 慢。
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

	// 分隔栏: 1px bg-border 全宽(shadcn -mx-1: 越过 p-1 边距铺满弹层),
	// 居中画在原生固定的 4px 槽内
	if st&xcc.MenuItem_State_Flag_Separator > 0 {
		d.SetLineWidth(1)
		d.SetBrushColor(uint32(t.Border))
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
		p.FillRound(&rc, hotBg.ScaleAlpha(hv), t.RadiusSm())
		if !isDestructive {
			fore = xui.Fade(fore, t.AccentForeground, hv)
		}
	}

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
		d.SetBrushColor(uint32(t.MutedForeground))
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
