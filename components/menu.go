package ui

import (
	"strings"

	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/drawx"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/wapi/wnd"
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
//   - 弹层: bg-popover rounded-lg p-1 ring-1 ring-foreground/10(无阴影, xcgui
//     菜单弹窗没有阴影入口, 与本项目 Tooltip/Select 弹层口径一致);
//   - 条目: h-7(28) rounded-md px-1.5, hover/focus 高亮 bg-accent + 文字变
//     accent-foreground; destructive 文字红、高亮 bg-destructive/10(dark /20);
//   - 禁用: opacity-50(组级透明 → 文字用 50% alpha 合成, 且不响应);
//   - 勾选/单选条目: 选中项右侧 Check 图标(right-2 size-4), 单选组内互斥;
//   - 分组标签: text-xs muted; 分隔栏: 1px bg-border(原生槽高仅 4px);
//   - 快捷键: 右对齐 text-xs muted(借助原生 "\t" 解析预留宽度);
//   - 定位: 触发元素下方左对齐 sideOffset=4(radix Content 默认), 下方放不下
//     翻到上方, 屏幕边缘夹紧。
//
// 用法(一次性对象, 每次弹出新建):
//
//	m := ui.NewMenu()
//	m.Item("复制", ui.MenuShortcut("Ctrl+C"), ui.MenuOnSelect(func() {...}))
//	m.Check("显示行号", checked, ui.MenuOnCheck(func(on bool) {...}))
//	m.PopupAtElement(btn.Handle) // 或 m.PopupAtCursor(w.Handle)
type Menu struct {
	m      *widget.Menu
	items  []*menuItem
	byID   map[int32]*menuItem
	nextID int32
	parent int32 // Sub() 构建期间的父项 ID

	// 弹出锚点(物理屏幕像素); 翻转/夹紧由原生菜单自带, 不需也不应手动校正
	ax, yBelow int32
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

// Label 添加分组标签(不可聚焦、不可点)。
func (m *Menu) Label(text string) *Menu {
	return m.add(&menuItem{kind: menuKindLabel, text: text, disabled: true}, nil)
}

// Separator 添加分隔栏。
func (m *Menu) Separator() *Menu {
	return m.add(&menuItem{kind: menuKindSeparator}, nil)
}

// Sub 添加子菜单, build 里添加的条目都挂在该子菜单下(可嵌套)。
func (m *Menu) Sub(text string, build func(s *Menu)) *Menu {
	p := &menuItem{kind: menuKindSub, text: text}
	m.add(p, nil)
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
	m.popup(hWindow)
}

func (m *Menu) popup(hWindow int) {
	m.m = m.build(hWindow)
	// 初始位置即锚点(元素矩形是精确的)。
	// ⚠️ 不要在这里或 Menu_PopupWnd 回调里做任何位置校正: XMenu_Popup 是阻塞的
	// (菜单关闭后才返回), 校正要么拿到已销毁的弹窗句柄(报「输入句柄可能无效」),
	// 要么在事件内 SetWindowPos 让子菜单锚点失效(二级菜单弹到父菜单原点盖住它,
	// 探针实测)。翻转/夹紧/子菜单定位原生自带, 锚点给对即可。
	m.m.Popup(xc.XWnd_GetHWND(hWindow), m.ax, m.yBelow, 0, xcc.Menu_Popup_Position_Left_Top)
}

// build 创建原生菜单并同步条目。
// 快捷键拼进文本("\t" 分隔): 原生按整串计算自适应宽度并为快捷键预留区域
// (探针实测 NShortcutKeyWidth=37), 绘制时再拆开分别渲染。
func (m *Menu) build(hWindow int) *widget.Menu {
	menu := widget.NewMenu()
	menu.SetAutoDestroy(true)
	menu.SetItemHeight(28) // 条目 = text-sm(20) + py-1(4×2)
	menu.SetLeftWidth(0)   // shadcn 条目没有左侧图标条, 图标(勾)画在右侧
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

// menuOnPopupWnd 弹窗出现: 圆角裁剪(Win32 区域, 同 ComboBox 弹层 —— shaped
// 透明对原生弹层无效)。只做圆角, 不动位置 —— 本回调里 SetWindowPos 会让子菜单
// 锚点失效(二级菜单弹到父菜单原点盖住它, 探针实测); 位置翻转/夹紧由原生菜单
// 自己处理。
func menuOnPopupWnd(_ int, hMenu int, pInfo *xc.Menu_PopupWnd_, pbHandled *bool) int {
	if pInfo == nil || pInfo.HWindow <= 0 {
		return 0
	}
	mw := window.NewByHandle(pInfo.HWindow)
	if mw == nil {
		return 0
	}
	t := xui.Current()
	dpi := xc.XWnd_GetDPI(pInfo.HWindow)
	if dpi <= 0 {
		dpi = 96
	}
	wnd.SetWindowRound(mw.GetHWND(), int32(float64(t.RadiusLg())*float64(dpi)/96))
	// 子菜单锚点由原生管理, 顶级弹窗的位置/翻转原生自己处理(屏幕边缘会自动
	// 夹紧/翻转), 这里不需要也不应该动位置 —— 探针实测对顶级弹窗做任何
	// SetWindowPos 都会让子菜单锚点失效(盖到父菜单上)。
	return 0
}

// menuDrawBackground 弹层底色 + ring-1 ring-foreground/10(内缩 1px, 四边完整,
// 同 ComboBox 弹层的结论)。
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
	rc.Left++
	rc.Top++
	rc.Right--
	rc.Bottom--
	d.SetLineWidth(1)
	d.SetBrushColor(uint32(t.Foreground.WithAlpha(26)))
	d.DrawRoundRect(&rc, t.RadiusLg()-1, t.RadiusLg()-1)
	return 0
}

// menuDrawItem 自绘菜单项。
// 项矩形几何(探针实测): 左右各留原生 3px 边距, 高度即 SetItemHeight(28),
// 分隔栏槽高固定 4px。
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

	// 分隔栏: 1px bg-border, 居中画在 4px 槽内
	if st&xcc.MenuItem_State_Flag_Separator > 0 {
		d.SetLineWidth(1)
		d.SetBrushColor(uint32(t.Border))
		d.DrawLine(rc.Left+6, rc.Top+2, rc.Right-6, rc.Top+2)
		return 0
	}

	disabled := st&xcc.MenuItem_State_Flag_Disable > 0
	hot := st&xcc.MenuItem_State_Flag_Stay > 0

	// 文字与高亮配色(dropdown-menu.tsx):
	//   普通: text-popover-foreground, hot: bg-accent + text-accent-foreground
	//   destructive: text-destructive, hot: bg-destructive/10(dark /20), 文字不变
	//   disabled: opacity-50(组级透明), 不参与高亮
	//   Label: text-xs muted, 永不高亮(原生已置禁用)
	fore := t.PopoverFore
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
			fore = xui.Fade(fore, t.AccentFore, hv)
		}
	}

	name, shortcut := menuItemText(it, pInfo)
	hasIcon := it != nil && ((it.kind == menuKindCheck || it.kind == menuKindRadio) && it.checked ||
		it.kind == menuKindSub)

	// 文本: px-1.5; 右侧有图标(勾/子菜单箭头)时按 pr-8 让位
	textRc := rc
	textRc.Left += 6
	textRc.Right -= 6
	if hasIcon {
		textRc.Right -= 24
	}
	if isLabel {
		d.SetFont(xui.FontHandle(xui.FontSpec{
			Family: t.FontDefault.Family,
			Size:   t.FontDefault.Size - 2, // text-xs
		}))
		d.SetBrushColor(uint32(t.MutedFore))
	} else {
		d.SetFont(xui.FontHandle(t.FontDefault))
		d.SetBrushColor(uint32(fore))
	}
	d.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter | xcc.TextFormatFlag_NoWrap)
	d.DrawText(name, &textRc)

	// 快捷键: 右对齐 text-xs muted(hot 时跟随 accent-foreground)
	if shortcut != "" {
		sc := t.MutedFore
		if hv > 0 {
			sc = xui.Fade(sc, t.AccentFore, hv)
		}
		if disabled {
			sc = sc.WithAlpha(128)
		}
		sr := rc
		sr.Right -= 6
		if hasIcon {
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
	if hasIcon {
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
