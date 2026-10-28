package ui

import (
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Menubar 的几何常量(对齐 shadcn menubar.tsx, 单位逻辑像素)。
const (
	mbPad        int32 = 4   // 根容器 p-1
	mbGap        int32 = 4   // 根容器 gap-1
	mbBarH       int32 = 36  // h-9: p-1 + 触发区视觉高 28 + p-1(按钮元素另含 pressShift 的 1px 预留)
	mbPanelPad   int32 = 4   // MenubarContent p-1
	mbItemH      int32 = 32  // 条目 px-2 py-1.5, text-sm 行高 20
	mbLabelH     int32 = 32  // MenubarLabel 同一条目高
	mbSepH       int32 = 9   // 分隔线 1px + my-1
	mbMinW       int32 = 192 // MenubarContent min-w-[12rem]
	mbSubMinW    int32 = 128 // MenubarSubContent min-w-[8rem]
	mbSideOffset int32 = 8   // sideOffset
	mbAlignOff   int32 = -4  // alignOffset(对齐起点再左移 4)
	// 子菜单与父面板**重叠** 4(对齐 radix SubContent 的 alignOffset -4):
	// 两块面板之间不能留缝 —— 鼠标穿过缝隙时两个面板都不再收到 MouseMove,
	// 收起检查会把子菜单关掉(写成正号就变成 4px 间隙, 实测移不过去)。
	mbSubOffset int32 = -4
)

// 键盘键码(只用到这几个)。
const (
	mbVKLeft  uintptr = 0x25
	mbVKUp    uintptr = 0x26
	mbVKRight uintptr = 0x27
	mbVKDown  uintptr = 0x28
	mbVKHome  uintptr = 0x24
	mbVKEnd   uintptr = 0x23
	mbVKEnter uintptr = 0x0D
	mbVKSpace uintptr = 0x20
	mbVKTab   uintptr = 0x09
)

// Menubar 菜单栏: 一排横向触发钮 + 每个钮下面的自绘下拉菜单。
//
// 与 shadcn menubar.tsx(radix-nova) 对齐:
//   - 根: h-9 圆角描边条、bg-background、p-1 gap-1; 触发区 rounded-sm px-2(text-sm medium),
//     悬停 / 展开 bg-accent;
//   - 弹层: bg-popover rounded-md 1px border shadow-md, min-w-[12rem], 位于触发区下方
//     左对齐(alignOffset -4, sideOffset 8), 超出工作区由弹层自身夹取;
//   - 条目: px-2 py-1.5 text-sm, 悬停 bg-accent; 前置图标 16; 勾选/单选指示器在左
//     (pl-8), 快捷键右对齐 text-xs muted; destructive 红字; disabled 50% 且不响应;
//     Label 为 text-sm 中等字重; 分隔线 1px 全宽;
//   - 子菜单: 悬停/Enter 在右侧弹出第二个弹层(非激活, 不抢焦点), 移出即收起;
//   - 交互: **打开任一菜单后, 鼠标移到其它触发区即切换**; 触发区点击切换开合;
//     ↑↓/Home/End 在条目间移动(跳过禁用/标签/分隔线), Enter 选中, ←→ 切换顶级菜单,
//     ESC 关闭并把焦点还给触发区, Tab 关闭。
//
// 弹层为什么不用原生 Menu: `XMenu_Popup` 是阻塞的(菜单关闭后才返回), 期间主窗口
// 收不到鼠标移动, 无法实现 radix 的"悬停切换菜单"语义; 这里用自绘弹层
// (复用 popoverPanel 的阴影窗/定位/夹取) + 自绘条目。
//
// 用法:
//
//	mb := ui.NewMenubar(card.Content())
//	file := mb.AddMenu("文件")
//	file.Item("新建", ui.MenuShortcut("Ctrl+N"), ui.MenuOnSelect(func() { ... }))
//	file.Check("显示行号", true, ui.MenuOnCheck(func(on bool) { ... }))
//	file.Separator()
//	file.Sub("分享", func(s *ui.MenubarMenu) {
//	    s.Item("邮件链接", ui.MenuOnSelect(func() { ... }))
//	})
type Menubar struct {
	hWnd     int
	bar      *widget.Element
	triggers []*Button
	menus    []*menubarView
	open     int // 当前打开的顶级菜单下标; -1 = 全关

	lastClose    time.Time // 点击触发区关闭后, 250ms 内的同一次点击不要再打开
	lastCloseIdx int
}

// MenubarMenu 是菜单栏里某个触发钮下的菜单(也是子菜单的构造入口)。
type MenubarMenu struct {
	v *menubarView
}

// menubarView 是一个弹层的渲染视图: 顶级菜单与每个子菜单各一个。
type menubarView struct {
	mb         *Menubar
	items      []*menuItem
	subs       map[int32]*menubarView
	parent     *menubarView
	parentItem *menuItem
	nextID     int32

	panel  *popoverPanel
	body   *widget.Element
	rows   []mbRow
	panelW int32
	panelH int32
	hover  int // 高亮行下标(-1 = 无)
	child  *menubarView
	hiding bool // 程序化收起子菜单时为 true, 用来区分"失焦被关"
}

// mbRow 是一行的布局结果(局部坐标, 相对弹层 body)。
type mbRow struct {
	it        *menuItem
	y, h      int32
	textX     int32
	textW     int32
	shortcutW int32
}

// NewMenubar 创建菜单栏。hParent 是承载它的父元素(通常是一张卡片的内容区);
// 根容器宽高按内容自撑, 放进任意流式布局即可。
func NewMenubar(hParent int) *Menubar {
	bar := widget.NewElement(0, 0, mbBarH, mbBarH, hParent)
	if bar == nil {
		return nil
	}
	bar.EnableBkTransparent(true)
	bar.EnableDrawBorder(false)
	bar.EnableFocus(false)
	bar.EnableSwitchFocus(false)
	// 普通 Element 不参与布局计算: Auto 宽会解析成 0(实测: 触发钮都排好了但整条不可见),
	// 宽度由 layoutTriggers 按内容写成 Fixed。
	bar.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, mbBarH)
	bar.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, mbBarH)
	bar.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		t := xui.Current()
		radius := t.RadiusMd()
		p.FillRound(&p.Rc, t.Background, radius)
		p.StrokeRound(&p.Rc, t.Border, radius, 1)
		return 0
	})
	mb := &Menubar{hWnd: xc.XWidget_GetHWINDOW(bar.Handle), bar: bar, open: -1}
	xui.AttachWindow(bar.Handle)
	// 弹层窗口是懒创建、只隐藏不销毁的 —— 宿主窗口销毁时必须跟着销毁, 否则
	// 消息循环的退出条件(所有窗口对象都销毁)永远不满足, 关窗后进程不退出
	// (与 Tooltip 同一个坑, 见 DESIGN 第八节 77)。
	bar.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		mb.destroyPanels()
		return 0
	})
	if win := window.NewByHandle(mb.hWnd); win != nil {
		win.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
			if message == wapi.WM_DESTROY {
				mb.destroyPanels()
			}
			return 0
		})
	}
	return mb
}

// destroyPanels 销毁全部弹层(顶级菜单与各级子菜单)。
func (mb *Menubar) destroyPanels() {
	for _, v := range mb.menus {
		v.destroyPanels()
	}
	mb.open = -1
	mb.triggers = nil
}

func (v *menubarView) destroyPanels() {
	for _, c := range v.subs {
		c.destroyPanels()
	}
	v.child = nil
	if v.panel != nil {
		v.panel.Destroy()
		v.panel, v.body = nil, nil
	}
}

// Bar 返回根容器句柄(可挂 tooltip 或改布局项)。
func (mb *Menubar) Bar() int {
	if mb == nil || mb.bar == nil {
		return 0
	}
	return mb.bar.Handle
}

// AddMenu 追加一个顶级菜单, 返回该菜单的构造器。菜单条目状态(勾选/单选)
// 保存在 Go 侧, 弹层每次打开按当前状态绘制。
func (mb *Menubar) AddMenu(text string) *MenubarMenu {
	if mb == nil || mb.bar == nil {
		return nil
	}
	idx := len(mb.menus)
	btn := NewButton(0, 0, 0, 0, text, mb.bar.Handle,
		WithVariant(ButtonGhost), WithSize(ButtonSizeSm), WithButtonHasPopup())
	if btn == nil {
		return nil
	}
	// 上游触发区是 text-sm(14) 配 h-7(28); sm 档的字号是 13, 这里只覆盖字号,
	// 高度/圆角仍走 sm 预设, 然后按新字号重量宽。
	btn.fontPx = 14
	btn.FitWidth()
	btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		*pbHandled = true
		if mb.open == idx {
			mb.closeAll(true)
			return 0
		}
		// 点触发区关闭时, 弹层失焦(KillFocus)先于 BnClick 到达 —— 这里已经在
		// OnHide 里把菜单关掉了, 不要再"关了又开"(与 DatePicker 的重开窗口同因)。
		if mb.open < 0 && idx == mb.lastCloseIdx && time.Since(mb.lastClose) < 250*time.Millisecond {
			return 0
		}
		mb.openMenu(idx, false)
		return 0
	})
	btn.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if mb.open >= 0 && mb.open != idx {
			mb.openMenu(idx, false)
		}
		return 0
	})
	btn.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		switch wParam {
		case mbVKDown, mbVKEnter, mbVKSpace:
			mb.openMenu(idx, true)
			*pbHandled = true
		case mbVKLeft:
			mb.focusTrigger((idx - 1 + len(mb.triggers)) % len(mb.triggers))
			*pbHandled = true
		case mbVKRight:
			mb.focusTrigger((idx + 1) % len(mb.triggers))
			*pbHandled = true
		}
		return 0
	})
	v := &menubarView{mb: mb, subs: map[int32]*menubarView{}, nextID: 1, hover: -1}
	v.body = nil
	mb.menus = append(mb.menus, v)
	mb.triggers = append(mb.triggers, btn)
	mb.layoutTriggers()
	return &MenubarMenu{v: v}
}

// mbSetRect 设置元素矩形(不触发重绘)。菜单栏独立安装时没有 dialog.go 的 setEleRect。
func mbSetRect(hEle int, x, y, cx, cy int32) {
	xc.XEle_SetRect(hEle, &xc.RECT{Left: x, Top: y, Right: x + cx, Bottom: y + cy},
		false, xcc.AdjustLayout_No)
}

// layoutTriggers 按内容重排触发钮并同步根容器宽。
func (mb *Menubar) layoutTriggers() {
	x := int32(mbPad)
	for _, tr := range mb.triggers {
		w, h := tr.GetWidth(), tr.GetHeight()
		// 按**壳体**高居中, 不按元素高: Button 的元素比视觉高多 pressShift(按下位移预留,
		// 留在底边), 按元素高算会让壳体整体上移 1px(条内上 4 / 下 6)。
		mbSetRect(tr.Handle, x, (mbBarH-(h-pressShift))/2, w, h)
		x += w + mbGap
	}
	total := x - mbGap + mbPad
	if total < 2*mbPad {
		total = 2 * mbPad
	}
	mb.bar.SetWidth(total)
	mb.bar.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, total)
	mb.bar.Redraw(false)
	// 根容器宽变了, 父级若是布局容器需要重排一次(Auto 宽)
	if p := xc.XWidget_GetParentEle(mb.bar.Handle); p != 0 {
		xc.XEle_AdjustLayout(p)
	}
}

// Close 关闭当前打开的菜单(等价于 ESC, 焦点回到触发区)。
func (mb *Menubar) Close() {
	if mb != nil {
		mb.closeAll(true)
	}
}

// IsOpen 报告是否有菜单打开。
func (mb *Menubar) IsOpen() bool {
	return mb != nil && mb.open >= 0
}

// focusTrigger 把键盘焦点移到某个触发钮(←→ 在触发区之间移动)。
func (mb *Menubar) focusTrigger(idx int) {
	if idx < 0 || idx >= len(mb.triggers) {
		return
	}
	xc.XWnd_SetFocusEle(mb.hWnd, mb.triggers[idx].Handle)
}

// openMenu 打开第 idx 个菜单。keyboard=true 时高亮第一个可选项。
func (mb *Menubar) openMenu(idx int, keyboard bool) {
	if idx < 0 || idx >= len(mb.menus) {
		return
	}
	v := mb.menus[idx]
	if mb.open == idx && v.panel != nil && v.panel.IsOpen() {
		return
	}
	if mb.open >= 0 && mb.open < len(mb.menus) {
		old := mb.menus[mb.open]
		old.closeChild()
		if old.panel != nil {
			old.panel.Hide()
		}
		old.hover = -1
	}
	v.build()
	if v.panel == nil {
		return
	}
	v.hover = -1
	if keyboard {
		v.hover = v.nextRow(-1, 1)
	}
	mb.open = idx
	mb.triggers[idx].SetExpanded(true)
	v.body.Redraw(false)
	x, y := mb.popupPos(idx)
	// ShowAt 之前先按最终大小重绘一次内容, 避免首帧空白(弹层窗口在屏幕外预显示)
	v.body.Redraw(true)
	v.panel.ShowAt(mb.triggers[idx].Handle, x, y)
}

// popupPos 返回弹层左上角的屏幕逻辑坐标(alignOffset -4, sideOffset 8)。
func (mb *Menubar) popupPos(idx int) (int32, int32) {
	var rc, wr xc.RECT
	xc.XEle_GetWndClientRect(mb.triggers[idx].Handle, &rc)
	// XWnd_GetRect 返回的是**逻辑像素**(与 XEle_SetRect 同一坐标系), 不要再除 DPI;
	// 只有 Win32 的 GetWindowRect 才是物理像素(见 panelScreenPos)。
	xc.XWnd_GetRect(mb.hWnd, &wr)
	// 从触发区的**视觉底边**量 8: 元素底边还含 Button 的 1px 按压位移预留
	// (见 button.go 的 pressShift), 拿元素底边算弹层会比上游低 1px。
	return wr.Left + rc.Left + mbAlignOff, wr.Top + rc.Bottom - pressShift + mbSideOffset
}

// onPanelHidden 弹层被关掉(失焦/主动 Hide)时同步菜单栏状态。
func (mb *Menubar) onPanelHidden(v *menubarView) {
	if v.parent != nil {
		v.closeChild()
		if v.hiding {
			return
		}
		// 焦点回到父菜单(点子菜单项/父菜单条目)时父层保持打开; 焦点去了
		// 别的窗口(点击外部)才整组关闭。
		fg := wapi.GetForegroundWindow()
		if v.parent.panel != nil && v.parent.panel.alive() &&
			fg == v.parent.panel.win.GetHWND() {
			v.parent.child = nil
			return
		}
		mb.closeAll(false)
		return
	}
	v.closeChild()
	if mb.open >= 0 && mb.open < len(mb.menus) && mb.menus[mb.open] == v {
		mb.lastClose = time.Now()
		mb.lastCloseIdx = mb.open
		mb.triggers[mb.open].SetExpanded(false)
		mb.open = -1
	}
	v.hover = -1
}

// closeAll 关闭整棵菜单树。焦点还必须在弹层已经隐藏之后也能工作: ESC 是
// popoverPanel 先 Hide(触发 OnHide, open 已清零)再调 OnEsc 的, 这里要从
// lastCloseIdx 取回刚关掉的那个菜单。
func (mb *Menubar) closeAll(focusTrigger bool) {
	if mb.open < 0 {
		if focusTrigger && mb.lastCloseIdx >= 0 {
			mb.focusTriggerLater(mb.lastCloseIdx)
		}
		return
	}
	idx := mb.open
	v := mb.menus[idx]
	v.closeChild()
	if v.panel != nil {
		v.panel.Hide()
	}
	v.hover = -1
	mb.triggers[idx].SetExpanded(false)
	mb.open = -1
	mb.lastClose = time.Now()
	mb.lastCloseIdx = idx
	if focusTrigger {
		mb.focusTriggerLater(idx)
	}
}

// focusTriggerLater 把焦点还给第 idx 个触发钮。弹层窗口刚隐藏时主窗口还没拿到
// 激活, 立刻 SetFocusEle 会被随后的激活流程覆盖(实测: ESC 后按 ↓ 重新打开的是
// 上一次点过的菜单); 推迟到消息循环再还焦点。
func (mb *Menubar) focusTriggerLater(idx int) {
	if idx < 0 || idx >= len(mb.triggers) {
		return
	}
	trigger := mb.triggers[idx].Handle
	app.AfterFunc(0, func() {
		if !xc.XC_IsHXCGUI(mb.hWnd, xcc.XC_WINDOW) || !xc.XC_IsHXCGUI(trigger, xcc.XC_ELE) {
			return
		}
		if hw := xc.XWnd_GetHWND(mb.hWnd); hw != 0 {
			wapi.SetForegroundWindow(hw)
		}
		xc.XWnd_SetFocusEle(mb.hWnd, trigger)
	})
}

// ===== MenubarMenu(条目构造) =====

func (m *MenubarMenu) add(it *menuItem, opts []MenuOpt) *MenubarMenu {
	if m == nil || m.v == nil {
		return m
	}
	for _, o := range opts {
		o(it)
	}
	v := m.v
	it.id = v.nextID
	v.nextID++
	v.items = append(v.items, it)
	return m
}

// Item 添加普通条目。
func (m *MenubarMenu) Item(text string, opts ...MenuOpt) *MenubarMenu {
	return m.add(&menuItem{kind: menuKindNormal, text: text}, opts)
}

// Check 添加勾选条目(点击翻转, 新状态经 MenuOnCheck 回调)。
func (m *MenubarMenu) Check(text string, checked bool, opts ...MenuOpt) *MenubarMenu {
	return m.add(&menuItem{kind: menuKindCheck, text: text, checked: checked}, opts)
}

// Radio 添加单选条目(同 MenuGroup 互斥)。
func (m *MenubarMenu) Radio(text string, checked bool, opts ...MenuOpt) *MenubarMenu {
	return m.add(&menuItem{kind: menuKindRadio, text: text, checked: checked}, opts)
}

// Label 添加分组标签(不可聚焦、不可点)。
func (m *MenubarMenu) Label(text string, opts ...MenuOpt) *MenubarMenu {
	return m.add(&menuItem{kind: menuKindLabel, text: text, disabled: true}, opts)
}

// Separator 添加分隔线。
func (m *MenubarMenu) Separator() *MenubarMenu {
	return m.add(&menuItem{kind: menuKindSeparator}, nil)
}

// Sub 添加子菜单, build 里添加的条目挂在该子菜单下。
func (m *MenubarMenu) Sub(text string, build func(s *MenubarMenu), opts ...MenuOpt) *MenubarMenu {
	if m == nil || m.v == nil {
		return m
	}
	it := &menuItem{kind: menuKindSub, text: text}
	m.add(it, opts)
	child := &menubarView{
		mb: m.v.mb, subs: map[int32]*menubarView{},
		parent: m.v, parentItem: it, nextID: 1, hover: -1,
	}
	m.v.subs[it.id] = child
	if build != nil {
		build(&MenubarMenu{v: child})
	}
	return m
}

// ===== 弹层构建与绘制 =====

// minWidth 返回该层弹层的最小宽(顶级 12rem / 子菜单 8rem)。
func (v *menubarView) minWidth() int32 {
	if v.parent != nil {
		return mbSubMinW
	}
	return mbMinW
}

// build 量出全部行、算弹层尺寸并把内容铺满面板。每次打开都重建,
// 保证勾选状态/主题字体变化能反映到尺寸与绘制上。
func (v *menubarView) build() {
	if v.mb == nil || len(v.mb.triggers) == 0 {
		return
	}
	t := xui.Current()
	font := xui.FontHandle(t.FontDefault)
	small := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size - 2})
	medium := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size, Bold: true})

	v.rows = v.rows[:0]
	y := int32(mbPanelPad)
	maxW := v.minWidth() - mbPanelPad*2
	for _, it := range v.items {
		if it == nil {
			continue
		}
		h := int32(mbItemH)
		switch it.kind {
		case menuKindSeparator:
			h = mbSepH
		case menuKindLabel:
			h = mbLabelH
		}
		r := mbRow{it: it, y: y, h: h}
		if it.kind != menuKindSeparator {
			lead := it.icon != "" || it.kind == menuKindCheck || it.kind == menuKindRadio || it.inset
			r.textX = 8
			if lead {
				r.textX = 32 // pl-8: 指示器/图标 + gap
			}
			if it.kind == menuKindLabel {
				r.textW = xui.TextWidth(it.text, medium)
			} else {
				r.textW = xui.TextWidth(it.text, font)
			}
			rowW := r.textX + r.textW + 8
			if it.shortcut != "" {
				r.shortcutW = xui.TextWidth(it.shortcut, small)
				rowW += 8 + r.shortcutW
			}
			if it.kind == menuKindSub {
				rowW += 24 // ChevronRight
			}
			if rowW > maxW {
				maxW = rowW
			}
		}
		v.rows = append(v.rows, r)
		y += h
	}
	v.panelW = maxW + mbPanelPad*2
	v.panelH = y + mbPanelPad
	v.ensurePanel()
	if v.panel == nil {
		return
	}
	v.panel.resize(v.panelW, v.panelH)
	if v.body != nil {
		mbSetRect(v.body.Handle, 0, 0, v.panelW, v.panelH)
	}
}

// ensurePanel 懒创建弹层(顶级激活、抢焦点; 子菜单非激活, 避免抢走父弹层焦点导致它失焦关闭)。
func (v *menubarView) ensurePanel() {
	if v.panel != nil || v.mb == nil || len(v.mb.triggers) == 0 {
		return
	}
	anchor := v.mb.triggers[0].Handle
	p := newPopoverPanel(10, 10, anchor)
	if p == nil {
		return
	}
	t := xui.Current()
	p.paintRadius = t.RadiusMd()
	p.paintBorderSolid = true

	body := widget.NewElement(0, 0, 10, 10, p.Content())
	if body == nil {
		p.Destroy()
		return
	}
	body.EnableBkTransparent(true)
	body.EnableDrawBorder(false)
	body.EnableFocus(false)
	body.EnableSwitchFocus(false)
	body.AddEvent_Paint(v.onPaint)
	body.AddEvent_MouseMove(v.onMouseMove)
	body.AddEvent_LButtonUp(v.onClick)
	body.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		// 指针离开这一层: 若既不在自己也不在父子另一层上, 收起子菜单。
		// (父子面板叠了 4px, 正常移入子菜单时不会走到这里)
		if v.parent != nil {
			if v.parent.child == v && !cursorInPanel(v.parent.panel) && !cursorInPanel(v.panel) {
				v.parent.closeChild()
			}
		} else if v.child != nil && !cursorInPanel(v.panel) && !cursorInPanel(v.child.panel) {
			v.closeChild()
		}
		return 0
	})

	v.panel = p
	v.body = body
	p.OnEsc = func() { v.mb.closeAll(true) }
	p.OnKeyDown = v.onKeyDown
	p.OnHide = func() { v.mb.onPanelHidden(v) }
}

// rowAt 返回 y 坐标下可交互的行下标(跳过分隔线/标签/禁用项); 没有则 -1。
func (v *menubarView) rowAt(y int32) int {
	for i := range v.rows {
		r := &v.rows[i]
		if y < r.y || y >= r.y+r.h {
			continue
		}
		if !rowSelectable(r.it) {
			return -1
		}
		return i
	}
	return -1
}

// rowSelectable 报告条目能否高亮/选择。
func rowSelectable(it *menuItem) bool {
	return it != nil && it.kind != menuKindSeparator && it.kind != menuKindLabel && !it.disabled
}

// nextRow 从 from 起按 dir(+1/-1)找下一个可交互行(循环); 找不到返回 -1。
func (v *menubarView) nextRow(from int, dir int) int {
	n := len(v.rows)
	if n == 0 {
		return -1
	}
	for k := 1; k <= n; k++ {
		i := ((from+dir*k)%n + n) % n
		if rowSelectable(v.rows[i].it) {
			return i
		}
	}
	return -1
}

// onMouseMove 悬停高亮; 移到子菜单触发行时展开子菜单, 移到其它行时收起。
func (v *menubarView) onMouseMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pPt == nil {
		return 0
	}
	idx := v.rowAt(pPt.Y)
	if idx == v.hover {
		return 0
	}
	old := v.hover
	v.hover = idx
	if old >= 0 && old < len(v.rows) && v.rows[old].it.kind == menuKindSub {
		v.closeChild()
	}
	if idx >= 0 && v.rows[idx].it.kind == menuKindSub {
		v.openChild(idx)
	}
	if v.body != nil {
		v.body.Redraw(false)
	}
	return 0
}

// onClick 点击条目: 子菜单展开, 其余分发并关闭整组菜单。
func (v *menubarView) onClick(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pPt == nil {
		return 0
	}
	idx := v.rowAt(pPt.Y)
	if idx < 0 {
		return 0
	}
	*pbHandled = true
	v.activate(idx)
	return 0
}

// activate 执行某一行: Sub 展开子菜单, 其它条目更新状态、关闭菜单并回调。
func (v *menubarView) activate(idx int) {
	if idx < 0 || idx >= len(v.rows) {
		return
	}
	it := v.rows[idx].it
	if it.kind == menuKindSub {
		v.openChild(idx)
		return
	}
	v.dispatch(it)
	v.mb.closeAll(true)
}

// dispatch 更新条目状态并回调(在关闭菜单前执行, 让回调看到新状态)。
func (v *menubarView) dispatch(it *menuItem) {
	switch it.kind {
	case menuKindCheck:
		it.checked = !it.checked
		if it.onCheck != nil {
			it.onCheck(it.checked)
		}
	case menuKindRadio:
		it.checked = true
		for _, other := range v.items {
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

// openChild 在父面板右侧展开第 idx 行的子菜单。
func (v *menubarView) openChild(idx int) {
	if idx < 0 || idx >= len(v.rows) || v.panel == nil {
		return
	}
	it := v.rows[idx].it
	child := v.subs[it.id]
	if child == nil {
		return
	}
	if v.child == child && child.panel != nil && child.panel.IsOpen() {
		return
	}
	v.closeChild()
	if child.body == nil {
		child.build()
	}
	if child.panel == nil || !child.panel.alive() {
		return
	}
	// 父面板卡片左上角的屏幕逻辑坐标(窗口矩形含阴影带, 要补回来)。
	px, py := panelScreenPos(v.panel)
	x := px + v.panelW + mbSubOffset
	y := py + v.rows[idx].y + mbSubOffset
	child.hover = -1
	child.body.Redraw(false)
	// 子菜单会抢焦点, 父弹层在焦点转移期间不能因失焦关掉自己
	v.panel.keepOnBlur = true
	child.panel.ShowAt(v.body.Handle, x, y)
	if !child.panel.IsOpen() {
		v.panel.keepOnBlur = false
		return
	}
	v.child = child
}

// closeChild 收起当前子菜单。
func (v *menubarView) closeChild() {
	if v.child == nil {
		return
	}
	child := v.child
	v.child = nil
	child.hiding = true
	if child.panel != nil {
		child.panel.Hide()
	}
	child.hiding = false
	child.hover = -1
	if v.panel != nil && v.panel.keepOnBlur {
		v.panel.keepOnBlur = false
		// 焦点还给父菜单, 后续点击外部仍能触发父层的失焦关闭
		if v.mb != nil && v.mb.open >= 0 && v.mb.open < len(v.mb.menus) &&
			v.mb.menus[v.mb.open] == v && v.panel.IsOpen() {
			wapi.SetForegroundWindow(v.panel.win.GetHWND())
		}
	}
}

// onKeyDown 键盘导航(由 popoverPanel 的窗口级 KeyDown 调用)。
func (v *menubarView) onKeyDown(wParam uintptr) bool {
	mb := v.mb
	if v.parent != nil { // 子菜单持有焦点时, ← 回到父层, ↑↓/Enter 在子菜单内导航
		switch wParam {
		case mbVKUp:
			v.moveHover(-1)
		case mbVKDown:
			v.moveHover(1)
		case mbVKHome:
			v.moveHoverTo(1)
		case mbVKEnd:
			v.moveHoverTo(-1)
		case mbVKEnter, mbVKSpace:
			if v.hover >= 0 {
				v.activate(v.hover)
			}
		case mbVKLeft:
			v.parent.closeChild()
		case mbVKRight:
			// 子菜单已是最右层: 吞掉, 不切顶级菜单
		case mbVKTab:
			mb.closeAll(true)
		default:
			return false
		}
		return true
	}
	switch wParam {
	case mbVKDown:
		if v.child != nil {
			v.child.moveHover(1)
		} else {
			v.moveHover(1)
		}
	case mbVKUp:
		if v.child != nil {
			v.child.moveHover(-1)
		} else {
			v.moveHover(-1)
		}
	case mbVKHome:
		v.moveHoverTo(1)
	case mbVKEnd:
		v.moveHoverTo(-1)
	case mbVKEnter, mbVKSpace:
		if v.child != nil && v.child.hover >= 0 {
			v.child.activate(v.child.hover)
		} else if v.hover >= 0 {
			v.activate(v.hover)
		}
	case mbVKRight:
		if v.hover >= 0 && v.rows[v.hover].it.kind == menuKindSub {
			v.openChild(v.hover)
		} else {
			mb.openMenu((mb.open+1)%len(mb.menus), true)
		}
	case mbVKLeft:
		if v.child != nil {
			v.closeChild()
		} else {
			mb.openMenu((mb.open-1+len(mb.menus))%len(mb.menus), true)
		}
	case mbVKTab:
		mb.closeAll(true)
	default:
		return false
	}
	return true
}

// moveHover 在当前层移动高亮行。
func (v *menubarView) moveHover(dir int) {
	idx := v.nextRow(v.hover, dir)
	if idx < 0 || idx == v.hover {
		return
	}
	v.setHover(idx)
}

// moveHoverTo 定位到第一个(dir=1)或最后一个(-1)可交互行。
func (v *menubarView) moveHoverTo(dir int) {
	var idx int
	if dir > 0 {
		idx = v.nextRow(-1, 1)
	} else {
		idx = v.nextRow(0, -1)
	}
	if idx >= 0 {
		v.setHover(idx)
	}
}

// setHover 更新高亮并同步子菜单开合。
func (v *menubarView) setHover(idx int) {
	old := v.hover
	v.hover = idx
	if old >= 0 && old < len(v.rows) && v.rows[old].it.kind == menuKindSub {
		v.closeChild()
	}
	if idx >= 0 && idx < len(v.rows) && v.rows[idx].it.kind == menuKindSub {
		v.openChild(idx)
	}
	if v.body != nil {
		v.body.Redraw(false)
	}
}

// onPaint 绘制全部行。
func (v *menubarView) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	font := xui.FontHandle(t.FontDefault)
	small := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size - 2})
	medium := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size, Bold: true})
	radius := t.RadiusSm()
	rowLeft := int32(mbPanelPad)
	rowRight := v.panelW - mbPanelPad

	for i := range v.rows {
		r := &v.rows[i]
		rowRc := xc.RECT{Left: rowLeft, Top: r.y, Right: rowRight, Bottom: r.y + r.h}
		if r.it.kind == menuKindSeparator {
			y := r.y + r.h/2
			p.Line(0, y, v.panelW, y, t.Border, 1)
			continue
		}
		hv := xui.HoverItem(hEle, int32(i), i == v.hover)
		disabledVis := r.it.disabled && r.it.kind != menuKindLabel
		fore := t.Foreground
		if r.it.kind == menuKindLabel {
			fore = t.Foreground
		} else if r.it.destructive {
			fore = t.Destructive
		}
		if hv > 0 {
			if r.it.destructive {
				frac := 0.1
				if t.IsDark() {
					frac = 0.2
				}
				p.FillRound(&rowRc, t.Destructive.ScaleAlpha(frac*hv), radius)
			} else {
				p.FillRound(&rowRc, t.Accent.ScaleAlpha(hv), radius)
			}
			if !r.it.destructive {
				fore = xui.Fade(fore, t.AccentForeground, hv)
			}
		}
		if disabledVis {
			fore = fore.WithAlpha(128)
		}

		// 前置指示器/图标
		if r.it.kind == menuKindCheck || r.it.kind == menuKindRadio || r.it.icon != "" {
			icRc := xc.RECT{
				Left:   rowLeft + 8,
				Top:    r.y + (r.h-16)/2,
				Right:  rowLeft + 24,
				Bottom: r.y + (r.h+16)/2,
			}
			switch {
			case r.it.kind == menuKindCheck && r.it.checked:
				p.Icon(icons.IconCheck, &icRc, fore, 2)
			case r.it.kind == menuKindRadio && r.it.checked:
				dot := xc.RECT{
					Left:   rowLeft + 12,
					Top:    r.y + (r.h-8)/2,
					Right:  rowLeft + 20,
					Bottom: r.y + (r.h+8)/2,
				}
				p.FillRound(&dot, fore, 4)
			case r.it.icon != "":
				p.Icon(r.it.icon, &icRc, fore, 2)
			}
		}

		// 文本
		textRc := rowRc
		textRc.Left += r.textX
		textRc.Right -= 8
		if r.it.shortcut != "" {
			textRc.Right -= 8 + r.shortcutW
		}
		if r.it.kind == menuKindSub {
			textRc.Right -= 24
		}
		p.Draw.SetFont(font)
		if r.it.kind == menuKindLabel {
			p.Draw.SetFont(medium)
		}
		p.Text(r.it.text, &textRc, fore,
			xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)

		// 快捷键(右对齐 text-xs)
		if r.it.shortcut != "" {
			sc := t.MutedForeground
			if hv > 0 {
				sc = xui.Fade(sc, t.AccentForeground, hv)
			}
			if disabledVis {
				sc = sc.WithAlpha(128)
			}
			sr := rowRc
			sr.Right -= 8
			if r.it.kind == menuKindSub {
				sr.Right -= 24
			}
			p.Draw.SetFont(small)
			p.Text(r.it.shortcut, &sr, sc,
				xcc.TextAlignFlag_Right|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		}

		// 子菜单箭头
		if r.it.kind == menuKindSub {
			cr := xc.RECT{
				Left:   rowRight - 24,
				Top:    r.y + (r.h-16)/2,
				Right:  rowRight - 8,
				Bottom: r.y + (r.h+16)/2,
			}
			p.Icon(icons.IconChevronRight, &cr, fore, 2)
		}
	}
	return 0
}
