package ui

import (
	"math"
	"time"

	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Sidebar 的几何常量(逻辑像素), 对齐 shadcn sidebar.tsx(radix-nova):
//   - 展开 16rem = 256, 折叠成图标栏 3rem = 48; floating/inset 变体外面还有 8px 的内边距,
//     折叠宽度按上游的 calc(48px + 16px + 2px) = 66(面板 50, 比图标多 2px 防止描边被裁);
//   - Header / Footer: p-2 + gap-2; Group: p-2; Content 里组与组之间 gap-2;
//   - 菜单按钮 h-8(32) / sm h-7(28) / lg h-12(48), 图标 size-4(16), 文字 text-sm;
//   - 子菜单: mx-3.5(14) + border-l + px-2.5(10), 子项 h-7(28);
//   - 组标签 h-8(32) px-2 text-xs(12); 菜单动作/组动作 20×20; 徽章 h-5 min-w-5 px-1。
const (
	sidebarWidth    int32 = 256
	sidebarIconW    int32 = 48
	sidebarPad      int32 = 8
	sidebarGap      int32 = 8
	sidebarItemIcon int32 = 16
	sidebarIconGap  int32 = 8
	sidebarChevron  int32 = 16 // 子菜单展开/收起 chevron(size-4)
	sidebarChevronX int32 = 8  // 距条目右缘 8(ml-auto + p-2)
	sidebarSubMarX  int32 = 14
	sidebarSubPadL  int32 = 11
	sidebarSubPadR  int32 = 10
	sidebarRate     int32 = 20 // 动作钮边长
	sidebarBadgeH   int32 = 20
	sidebarBadgePad int32 = 4
	sidebarLabelH   int32 = 32
	sidebarGroupGap int32 = 4
	sidebarAnimMs         = 200 * time.Millisecond
	sidebarRingW    int32 = 2 // focus-visible:ring-2
)

// SidebarSide 侧栏贴哪一边, 对应 shadcn sidebar.tsx 的 side。
type SidebarSide int

const (
	SidebarLeft  SidebarSide = iota // 左侧(默认)
	SidebarRight                    // 右侧
)

// SidebarVariant 侧栏外观, 对应 shadcn sidebar.tsx 的 variant。
type SidebarVariant int

const (
	// SidebarVariantSidebar 默认: 整条面板贴边、内缘 1px 分隔线。
	SidebarVariantSidebar SidebarVariant = iota
	// SidebarVariantFloating 浮动: 面板四周留 8px, 圆角 + 描边。
	SidebarVariantFloating
	// SidebarVariantInset 内嵌: 外层铺 sidebar 底色, 主内容区像卡片一样浮在里面(圆角 + 8px 边距)。
	SidebarVariantInset
)

// SidebarCollapsible 收起方式, 对应 shadcn sidebar.tsx 的 collapsible。
type SidebarCollapsible int

const (
	// SidebarCollapsibleOffcanvas 整条侧栏滑出视野(默认)。
	SidebarCollapsibleOffcanvas SidebarCollapsible = iota
	// SidebarCollapsibleIcon 收成 3rem 图标栏: 文字裁掉、分组标签/子菜单/徽章/动作隐藏,
	// 菜单项悬停时在右侧弹 tooltip(上游 TooltipProvider delayDuration=0)。
	SidebarCollapsibleIcon
	// SidebarCollapsibleNone 不收起(宽度恒定)。
	SidebarCollapsibleNone
)

// SidebarSize 菜单按钮尺寸档, 对应 sidebarMenuButtonVariants 的 size。
type SidebarSize int

const (
	SidebarSizeDefault SidebarSize = iota // h-8 text-sm
	SidebarSizeSm                         // h-7 text-xs
	SidebarSizeLg                         // h-12 text-sm
)

// SidebarItemVariant 菜单按钮变体, 对应 sidebarMenuButtonVariants 的 variant。
type SidebarItemVariant int

const (
	SidebarItemDefault SidebarItemVariant = iota // 透明底, 悬停/选中 accent
	SidebarItemOutline                           // 背景色 + 1px sidebar-border, 悬停边框转 accent
)

// SidebarOption 侧栏配置项。
type SidebarOption func(*Sidebar)

// WithSidebarSide 设置贴边方向(默认左侧)。
func WithSidebarSide(side SidebarSide) SidebarOption {
	return func(s *Sidebar) { s.side = side }
}

// WithSidebarVariant 设置外观变体(默认 SidebarVariantSidebar)。
func WithSidebarVariant(v SidebarVariant) SidebarOption {
	return func(s *Sidebar) { s.variant = v }
}

// WithSidebarCollapsible 设置收起方式(默认 SidebarCollapsibleOffcanvas)。
func WithSidebarCollapsible(c SidebarCollapsible) SidebarOption {
	return func(s *Sidebar) { s.collapsible = c }
}

// WithSidebarWidth 展开宽度(逻辑像素, 默认 256 = 16rem)。
func WithSidebarWidth(px int32) SidebarOption {
	return func(s *Sidebar) {
		if px > 0 {
			s.width = px
		}
	}
}

// WithSidebarIconWidth 折叠成图标栏时的宽度(默认 48 = 3rem)。
func WithSidebarIconWidth(px int32) SidebarOption {
	return func(s *Sidebar) {
		if px > 0 {
			s.iconWidth = px
		}
	}
}

// WithSidebarOpen 初始展开状态(默认展开)。
func WithSidebarOpen(open bool) SidebarOption {
	return func(s *Sidebar) { s.open = open }
}

// WithSidebarRail 在面板内缘加一条 16px 宽的可点击细条: 悬停显出 2px 分隔线,
// 点击开合(对应上游的 SidebarRail)。
func WithSidebarRail() SidebarOption {
	return func(s *Sidebar) { s.rail = true }
}

// WithSidebarNoShortcut 关闭 Ctrl+B 快捷键(默认开启, 同上游 SIDEBAR_KEYBOARD_SHORTCUT)。
func WithSidebarNoShortcut() SidebarOption {
	return func(s *Sidebar) { s.shortcut = false }
}

// WithSidebarOnOpenChange 注册展开状态变化回调。
func WithSidebarOnOpenChange(fn func(open bool)) SidebarOption {
	return func(s *Sidebar) {
		if fn != nil {
			s.changes = append(s.changes, fn)
		}
	}
}

// Sidebar 侧边栏外壳: 左侧(或右侧)是可折叠面板, 其余空间是主内容区(SidebarInset)。
//
// 与 shadcn sidebar.tsx 对齐的部分: 展开/收起(offcanvas 滑出 / icon 图标栏 / none)、
// 左右贴边、sidebar / floating / inset 三种外观、Header / Content / Footer 三段、
// 分组标签与组动作、菜单按钮(选中/徽章/尾随动作/子菜单)、折叠后悬停出右侧 tooltip、
// Ctrl+B 开合、200ms 线性宽度过渡、全部使用 --sidebar-* 令牌。
//
// 用法(必须在确定高度的容器里, 侧栏内容区按剩余高度滚动):
//
//	sb := ui.NewSidebar(0, 0, 0, 560, stack.Handle,
//		ui.WithSidebarVariant(ui.SidebarVariantInset),
//		ui.WithSidebarCollapsible(ui.SidebarCollapsibleIcon))
//	xui.ItemFill(sb)
//	xui.ItemHeight(sb, 560)
//
//	sb.AddHeaderItem("Acme Inc.", icons.IconGalleryVerticalEnd, ui.WithSidebarItemSize(ui.SidebarSizeLg))
//	g := sb.AddGroup("平台")
//	g.AddItem("模型", icons.IconMonitor, ui.WithSidebarItemActive(true), ui.WithSidebarItemTooltip("模型"))
//	...
//	main := sb.Main() // 主内容区: 自己往里放标题栏/页面内容
//
// 与上游的差异:
//   - 没有 isMobile / Sheet 分支: 桌面窗口没有"移动端"这一层, 窗口窄时用 offcanvas 收起即可;
//   - 折叠动画只过渡宽度(上游还过渡 gap/内边距, 观感一致); 子菜单/徽章/动作在状态切换时立即隐藏(同上游);
//   - group 动作、菜单动作是独立子元素(上游是绝对定位的按钮), 悬停底色与点击语义一致。
type Sidebar struct {
	*widget.Element

	side        SidebarSide
	variant     SidebarVariant
	collapsible SidebarCollapsible
	width       int32
	iconWidth   int32
	open        bool
	progress    float64 // 0 = 收起, 1 = 展开(宽度过渡逐帧写入)
	rail        bool
	shortcut    bool

	panel   *widget.LayoutEle
	header  *widget.LayoutEle
	content *ScrollArea
	footer  *widget.LayoutEle
	main    *widget.LayoutEle

	railEle *widget.Element
	// 普通 Element 收不到可靠的 MouseStay(ScrollArea 里的元素尤其如此, 见 DESIGN 21),
	// 细条的悬停改由宿主窗口的 MouseMove 命中测试驱动。
	railHover bool
	actions   []*sidebarIconAction
	mouseWin  int
	mouseID   int
	leaveID   int
	groups    []*SidebarGroup
	items     []*SidebarMenuButton
	subs      []*SidebarMenuSub

	animKey int32
	changes []func(open bool)

	keyWin int
	keyID  int

	tip *Tooltip
}

// NewSidebar 创建侧边栏外壳。
//
// cx 传 0 时宽度铺满父级(通常是整行, 主内容区占剩余宽度); cy 是侧栏高度,
// 传 0 用默认 480 —— 内容区(ScrollArea)按剩余高度滚动, 所以高度必须是确定值,
// 后续也可以用 xui.ItemFillHeight / ItemHeight 覆盖布局项。
func NewSidebar(x, y, cx, cy int32, hParent int, opts ...SidebarOption) *Sidebar {
	s := &Sidebar{
		side:        SidebarLeft,
		variant:     SidebarVariantSidebar,
		collapsible: SidebarCollapsibleOffcanvas,
		width:       sidebarWidth,
		iconWidth:   sidebarIconW,
		open:        true,
		shortcut:    true,
		progress:    1,
		animKey:     xui.NextPresenceKey(),
	}
	for _, o := range opts {
		o(s)
	}
	if cy <= 0 {
		cy = 480
	}
	if !s.open {
		// 初始收起: 直接落在收起宽度上, 不播过渡(SetOpen 才动画)。
		if s.collapsible == SidebarCollapsibleNone {
			s.open = true
			s.progress = 1
		} else {
			s.progress = 0
		}
	}
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	s.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.SetPadding(0, 0, 0, 0)
	if cx <= 0 {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	} else {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, cx)
	}
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, cy)
	ele.AddEvent_Paint(s.onPaintShell)
	ele.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		s.layout()
		return 0
	})
	xui.AttachWindow(ele.Handle)

	s.buildPanel()
	s.buildMain()
	if s.rail {
		s.buildRail()
	}
	if !s.open {
		s.applyState()
	}
	s.layout()
	// 预热过渡值: Presence 的新键默认从 0 起。展开态起始时键必须已经是 1,
	// 否则第一次收起会撞上"目标 0 == 当前 0", 过渡不跑、宽度瞬跳(探针实测)。
	// 反向也一样: 初始收起时键保持 0, 第一次展开才会从 0 长到 1。
	if s.collapsible != SidebarCollapsibleNone {
		xui.Presence(s.presenceOwner(), s.animKey, s.open, 0, nil, nil)
	}
	s.hookKeys()
	s.hookRailMouse()
	s.syncTooltips()
	ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		xui.DropTransitions(s.Handle)
		return 0
	})
	return s
}

// hookRailMouse 用宿主窗口的鼠标移动给 rail 做悬停命中(元素级 MouseStay 不可靠):
// 光标进入 16px 细条范围就重绘, 细线由 hover 过渡淡入。
func (s *Sidebar) hookRailMouse() {
	if s == nil {
		return
	}
	hw := xui.HostWindowOf(s.Handle)
	if hw == 0 {
		return
	}
	w := window.NewByHandle(hw)
	if w == nil {
		return
	}
	s.mouseWin = hw
	s.mouseID = w.AddEvent_MouseMove(func(hWin int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		if pPt == nil {
			return 0
		}
		var rc xc.RECT
		if s.railEle != nil && xc.XC_IsHXCGUI(s.railEle.Handle, xcc.XC_ELE) {
			xc.XEle_GetWndClientRect(s.railEle.Handle, &rc)
		}
		hover := s.railEle != nil && xc.XWidget_IsShow(s.railEle.Handle) &&
			pPt.X >= rc.Left && pPt.X < rc.Right && pPt.Y >= rc.Top && pPt.Y < rc.Bottom
		if hover != s.railHover {
			s.railHover = hover
			s.railEle.Redraw(true)
		}
		for _, a := range s.actions {
			if a == nil || !xc.XC_IsHXCGUI(a.Handle, xcc.XC_ELE) {
				continue
			}
			var ar xc.RECT
			xc.XEle_GetWndClientRect(a.Handle, &ar)
			h := xc.XWidget_IsShow(a.Handle) &&
				pPt.X >= ar.Left && pPt.X < ar.Right && pPt.Y >= ar.Top && pPt.Y < ar.Bottom
			if h != a.hover {
				a.hover = h
				a.Redraw(true)
			}
		}
		return 0
	})
	leave := func(hWin int, pbHandled *bool) int {
		if s.railHover && s.railEle != nil && xc.XC_IsHXCGUI(s.railEle.Handle, xcc.XC_ELE) {
			s.railHover = false
			s.railEle.Redraw(true)
		}
		for _, a := range s.actions {
			if a != nil && a.hover {
				a.hover = false
				if xc.XC_IsHXCGUI(a.Handle, xcc.XC_ELE) {
					a.Redraw(true)
				}
			}
		}
		return 0
	}
	if id := w.AddEvent_MouseLeave(leave); id > 0 {
		s.leaveID = id
	}
	s.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if xc.XC_IsHXCGUI(s.mouseWin, xcc.XC_WINDOW) {
			if w := window.NewByHandle(s.mouseWin); w != nil {
				if s.mouseID > 0 {
					w.RemoveEvent(xcc.WM_MOUSEMOVE, s.mouseID)
				}
				if s.leaveID > 0 {
					w.RemoveEvent(xcc.WM_MOUSELEAVE, s.leaveID)
				}
			}
		}
		s.mouseID, s.leaveID = 0, 0
		return 0
	})
}

// ===== 结构 =====

func (s *Sidebar) buildPanel() {
	p := widget.NewLayoutEle(0, 0, 0, 0, s.Handle)
	if p == nil {
		return
	}
	s.panel = p
	p.EnableHorizon(false)
	p.EnableLayout(true)
	p.EnableAutoWrap(false)
	p.EnableBkTransparent(true)
	p.EnableDrawBorder(false)
	p.EnableFocus(false)
	p.SetSpace(0)
	p.AddEvent_Paint(s.onPaintPanel)

	// Header / Footer = p-2 + gap-2: 左右用元素 padding(约束子项 Fill 宽),
	// 上下的 8px 用布局项外边距(计入 Auto 高, 见 DESIGN 8.129)。
	s.header = newSidebarBlock(p.Handle)
	s.header.SetSpace(sidebarGap)
	s.header.SetPadding(sidebarPad, 0, sidebarPad, 0)
	s.header.LayoutItem_SetMargin(0, sidebarPad, 0, sidebarPad)

	// Content = flex-1 + overflow-auto: 高度吃面板剩余空间(面板高度确定), 内容自撑。
	s.content = NewScrollArea(0, 0, 0, 0, p.Handle)
	s.content.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	xui.ItemFillHeight(s.content)
	s.content.Content().SetSpace(sidebarGap)

	s.footer = newSidebarBlock(p.Handle)
	s.footer.SetSpace(sidebarGap)
	s.footer.SetPadding(sidebarPad, 0, sidebarPad, 0)
	s.footer.LayoutItem_SetMargin(0, sidebarPad, 0, sidebarPad)
}

func (s *Sidebar) buildMain() {
	m := widget.NewLayoutEle(0, 0, 0, 0, s.Handle)
	if m == nil {
		return
	}
	s.main = m
	m.EnableHorizon(false)
	m.EnableLayout(true)
	m.EnableAutoWrap(false)
	m.EnableBkTransparent(true)
	m.EnableDrawBorder(false)
	m.EnableFocus(false)
	m.SetSpace(0)
	m.AddEvent_Paint(s.onPaintMain)
}

func (s *Sidebar) buildRail() {
	r := widget.NewElement(0, 0, 16, 0, s.Handle)
	if r == nil {
		return
	}
	s.railEle = r
	// 细条要压在 panel / main 之上。SetZOrderEx 在窗口尚未创建时会访问违例(实测),
	// 索引入口的 SetZOrder(0) 是构造期唯一可用的抬升方式(InputGroup 同款)。
	_ = r.SetZOrder(0)
	r.EnableBkTransparent(true)
	r.EnableDrawBorder(false)
	r.EnableFocus(false)
	r.EnableSwitchFocus(false)
	r.SetCursor(sidebarResizeCursor())
	r.AddEvent_Paint(s.onPaintRail)
	r.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		s.Toggle()
		*pbHandled = true
		return 0
	})
}

// newSidebarBlock 建一个竖向流容器: 宽铺满、高自撑、透明。
func newSidebarBlock(hParent int) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableAutoWrap(false)
	l.EnableBkTransparent(true)
	l.EnableDrawBorder(false)
	l.EnableFocus(false)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}

// Header 返回侧栏头部容器(竖向流, 内边距 8、子项间距 8), 可放任意控件。
func (s *Sidebar) Header() *widget.LayoutEle { return s.header }

// Content 返回侧栏内容区(滚动容器)。内容里的分组一般用 AddGroup 创建。
func (s *Sidebar) Content() *ScrollArea { return s.content }

// Footer 返回侧栏脚部容器(竖向流, 内边距 8、子项间距 8)。
func (s *Sidebar) Footer() *widget.LayoutEle { return s.footer }

// Main 返回主内容区(竖向流容器, 背景 = 主题 Background; inset 变体下是圆角卡片)。
// 它按外壳尺寸自动定位/缩放, 调用方往里放标题栏、页面内容即可。
func (s *Sidebar) Main() *widget.LayoutEle { return s.main }

// AddHeaderItem 在头部追加一个菜单按钮(随折叠态自动变成图标按钮并绑定 tooltip)。
func (s *Sidebar) AddHeaderItem(text, icon string, opts ...SidebarItemOption) *SidebarMenuButton {
	return s.addItemIn(s.header.Handle, text, icon, opts...)
}

// AddFooterItem 在脚部追加一个菜单按钮。
func (s *Sidebar) AddFooterItem(text, icon string, opts ...SidebarItemOption) *SidebarMenuButton {
	return s.addItemIn(s.footer.Handle, text, icon, opts...)
}

// AddGroup 在内容区追加一个分组(label 为空 = 无标题分组), 返回分组容器。
func (s *Sidebar) AddGroup(label string) *SidebarGroup {
	if s == nil || s.content == nil {
		return nil
	}
	g := &SidebarGroup{sidebar: s}
	outer := widget.NewLayoutEle(0, 0, 0, 0, s.content.Content().Handle)
	if outer == nil {
		return nil
	}
	g.LayoutEle = outer
	outer.EnableHorizon(false)
	outer.EnableLayout(true)
	outer.EnableAutoWrap(false)
	outer.EnableBkTransparent(true)
	outer.EnableDrawBorder(false)
	outer.EnableFocus(false)
	outer.SetSpace(0)
	outer.SetPadding(sidebarPad, 0, sidebarPad, 0)
	outer.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	outer.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	outer.LayoutItem_SetMargin(0, sidebarPad, 0, sidebarPad)

	if label != "" {
		g.label = newSidebarGroupLabel(g, label)
	}
	g.menu = newSidebarBlock(outer.Handle)
	g.menu.SetSpace(sidebarGroupGap)

	s.groups = append(s.groups, g)
	if s.iconMode() {
		g.applyIconMode(true)
	}
	s.panel.AdjustLayout()
	s.content.AdjustLayout()
	return g
}

// AddSeparator 在内容区追加一条 sidebar-border 分隔线(上游 SidebarSeparator: mx-2)。
func (s *Sidebar) AddSeparator() *widget.Element {
	if s == nil || s.content == nil {
		return nil
	}
	ele := widget.NewElement(0, 0, 0, 1, s.content.Content().Handle)
	if ele == nil {
		return nil
	}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 1)
	ele.LayoutItem_SetMargin(sidebarPad, 0, sidebarPad, 0)
	ele.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		sidebarFillHPx(p, p.Rc.Left, p.Rc.Right, (p.Rc.Top+p.Rc.Bottom)/2, xui.Current().SidebarBorder)
		return 0
	})
	xui.AttachWindow(ele.Handle)
	s.content.AdjustLayout()
	return ele
}

// AddTrigger 在指定父容器里建一个 ghost 图标按钮(28×28, PanelLeft), 点击开合侧栏。
func (s *Sidebar) AddTrigger(hParent int) *Button {
	if s == nil {
		return nil
	}
	b := NewButton(0, 0, 0, 0, "", hParent,
		WithVariant(ButtonGhost), WithSize(ButtonSizeIconSm), WithIcon(icons.IconPanelLeft))
	if b == nil {
		return nil
	}
	b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		s.Toggle()
		return 0
	})
	return b
}

// ===== 状态 / 动画 =====

// Open 报告侧栏是否处于展开状态。
func (s *Sidebar) Open() bool { return s != nil && s.open }

// Toggle 切换展开/收起。
func (s *Sidebar) Toggle() *Sidebar {
	if s == nil {
		return s
	}
	return s.SetOpen(!s.open)
}

// SetOpen 设置展开/收起(200ms 宽度过渡, 同上游 duration-200 ease-linear)。
func (s *Sidebar) SetOpen(open bool) *Sidebar {
	if s == nil || s.collapsible == SidebarCollapsibleNone || s.open == open {
		// collapsible=none 的上游实现是一块恒定宽度的面板, open 状态不影响渲染。
		return s
	}
	s.open = open
	s.applyState()
	s.animateWidth()
	for _, fn := range s.changes {
		fn(open)
	}
	return s
}

// OnOpenChange 注册展开状态变化回调(可多次注册)。
func (s *Sidebar) OnOpenChange(fn func(open bool)) *Sidebar {
	if s == nil || fn == nil {
		return s
	}
	s.changes = append(s.changes, fn)
	return s
}

// iconMode 报告当前是否处于"图标栏"形态(只有 collapsible=icon 才有结构变化;
// offcanvas 收起时面板只是滑出视野, 内容保持原样, 与上游 data-collapsible 一致)。
func (s *Sidebar) iconMode() bool {
	return s != nil && s.collapsible == SidebarCollapsibleIcon && !s.open
}

func (s *Sidebar) animateWidth() {
	if s == nil || s.panel == nil {
		return
	}
	if s.collapsible == SidebarCollapsibleNone {
		s.progress = 1
		s.layout()
		s.redraw()
		return
	}
	apply := func(v float64) {
		s.progress = v
		s.layout()
	}
	settle := func(v float64) {
		s.progress = v
		s.layout()
		s.redraw()
	}
	v := xui.Presence(s.presenceOwner(), s.animKey, s.open, sidebarAnimMs, apply, settle)
	if (s.open && v >= 1) || (!s.open && v <= 0) {
		settle(v)
	}
}

// presenceOwner 过渡挂宿主窗口: 同一窗口里的过渡共用一条定时器, 每帧只整窗重绘一次
// (同 Collapsible, 见 DESIGN 111)。
func (s *Sidebar) presenceOwner() int {
	if hw := xui.HostWindowOf(s.Handle); hw != 0 {
		return hw
	}
	return s.Handle
}

// applyState 把"图标栏"相关的结构变化落到子元素上(收起瞬间生效, 同上游的 hidden 类):
// 分组标签/子菜单/徽章/动作隐藏, 菜单项压成 32×32, 折叠 tooltip 绑上。
func (s *Sidebar) applyState() {
	icon := s.iconMode()
	for _, g := range s.groups {
		g.applyIconMode(icon)
	}
	for _, m := range s.items {
		m.applyIconMode(icon)
	}
	for _, sub := range s.subs {
		sub.applyIconMode(icon)
	}
	s.syncTooltips()
	// 结构变了, 内容高度/滚动范围都要重算。
	if s.panel != nil {
		s.panel.AdjustLayout()
	}
	if s.content != nil {
		s.content.AdjustLayout()
	}
}

// syncTooltips 只给"折叠成图标栏"的菜单项绑提示(上游 <TooltipContent hidden={state!=='collapsed'}>):
// 展开时解绑, 收起时绑定; 提示在右侧居中、无延迟(TooltipProvider delayDuration=0)。
func (s *Sidebar) syncTooltips() {
	if s == nil {
		return
	}
	icon := s.iconMode()
	for _, m := range s.items {
		text := m.tooltipText()
		if text == "" {
			continue
		}
		if icon {
			s.ensureTip().BindTip(m.Handle, text, WithTipSide(TipRight), WithTipAlign(TipAlignCenter), WithTipDelay(0))
		} else if s.tip != nil {
			s.tip.Unbind(m.Handle)
		}
	}
}

func (s *Sidebar) ensureTip() *Tooltip {
	if s.tip == nil {
		s.tip = NewTooltip()
	}
	return s.tip
}

// ===== 布局 =====

// gapWidth 返回侧栏占用的横向空间(含 floating/inset 的 8px 外边距)。
func (s *Sidebar) gapWidth() float64 {
	if s.collapsible == SidebarCollapsibleNone {
		return float64(s.width)
	}
	var collapsed float64
	switch s.collapsible {
	case SidebarCollapsibleOffcanvas:
		collapsed = 0
	default:
		collapsed = float64(s.iconWidth)
		if s.variant != SidebarVariantSidebar {
			collapsed += float64(sidebarPad*2 + 2) // calc(3rem + spacing(4) + 2px)
		}
	}
	return collapsed + (float64(s.width)-collapsed)*s.progress
}

// layout 按当前外壳尺寸与过渡进度摆放面板 / 主内容区 / 可拖细条。
func (s *Sidebar) layout() {
	if s == nil || s.panel == nil || s.main == nil {
		return
	}
	if !xc.XC_IsHXCGUI(s.Handle, xcc.XC_ELE) {
		return
	}
	w, h := s.GetWidth(), s.GetHeight()
	if w < 0 || h < 0 {
		return
	}
	gap := int32(math.Round(s.gapWidth()))
	if gap < 0 {
		gap = 0
	}
	if gap > s.width {
		gap = s.width
	}
	padOuter := int32(0)
	if s.variant != SidebarVariantSidebar {
		padOuter = sidebarPad
	}

	// 面板: offcanvas 收起时整块面板固定展开宽度、随 gap 移出视野(内容被父级裁剪),
	// icon 收起时面板本身就在变窄。
	outerW := gap
	if s.collapsible == SidebarCollapsibleOffcanvas {
		outerW = s.width
	}
	var outerX int32
	if s.side == SidebarLeft {
		outerX = 0
		if s.collapsible == SidebarCollapsibleOffcanvas {
			outerX = gap - s.width
		}
	} else {
		outerX = w - gap
	}
	panelW := outerW - padOuter*2
	if panelW < 0 {
		panelW = 0
	}
	panelH := h - padOuter*2
	if panelH < 0 {
		panelH = 0
	}
	s.panel.SetRectEx(outerX+padOuter, padOuter, panelW, panelH, false, xcc.AdjustLayout_No)
	s.panel.AdjustLayout()

	// 主内容区: floating/sidebar 从 gap 起铺满; inset 四周 8px, 收起后靠侧栏一侧再加 8px
	// (上游的 m-2 ml-0 / 收起态 ml-2)。
	mx, my, mw, mh := int32(0), int32(0), w, h
	if s.variant == SidebarVariantInset {
		my, mh = sidebarPad, h-sidebarPad*2
		if mh < 0 {
			mh = 0
		}
		if s.side == SidebarLeft {
			mx = gap
			if !s.open {
				mx += sidebarPad
			}
			mw = w - mx - sidebarPad
		} else {
			mx = sidebarPad
			right := w - gap
			if !s.open {
				right -= sidebarPad
			}
			mw = right - mx
		}
		if mw < 0 {
			mw = 0
		}
	} else if s.side == SidebarLeft {
		mx, mw = gap, w-gap
	} else {
		mw = w - gap
	}
	s.main.SetRectEx(mx, my, mw, mh, false, xcc.AdjustLayout_No)
	s.main.AdjustLayout()

	if s.railEle != nil {
		rx := gap - 8
		if s.side == SidebarRight {
			rx = w - gap - 8
		}
		s.railEle.SetRectEx(rx, 0, 16, h, false, xcc.AdjustLayout_No)
	}
}

func (s *Sidebar) redraw() {
	if s == nil || !xc.XC_IsHXCGUI(s.Handle, xcc.XC_ELE) {
		return
	}
	if hw := xui.HostWindowOf(s.Handle); hw != 0 {
		xc.XWnd_Redraw(hw, true)
	}
}

func (s *Sidebar) hookKeys() {
	if s == nil || !s.shortcut {
		return
	}
	hw := xui.HostWindowOf(s.Handle)
	if hw == 0 {
		return
	}
	w := window.NewByHandle(hw)
	if w == nil {
		return
	}
	s.keyWin = hw
	s.keyID = w.AddEvent_KeyDown(func(hWin int, wParam, lParam uintptr, pbHandled *bool) int {
		const vkB = 0x42
		if wParam != vkB {
			return 0
		}
		if wapi.GetAsyncKeyState(0x11) >= 0 { // VK_CONTROL
			return 0
		}
		s.Toggle()
		*pbHandled = true
		return 0
	})
	// 回调挂在窗口上, 元素先销毁时不会自动摘掉(同 Pagination, 见 DESIGN 152)。
	s.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if s.keyID > 0 && xc.XC_IsHXCGUI(s.keyWin, xcc.XC_WINDOW) {
			if w := window.NewByHandle(s.keyWin); w != nil {
				w.RemoveEvent(xcc.WM_KEYDOWN, s.keyID)
			}
		}
		s.keyID = 0
		xui.DropTransitions(s.Handle)
		return 0
	})
}

// ===== 绘制 =====

func (s *Sidebar) onPaintShell(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	// 外层只在 inset 变体里可见: 铺 sidebar 底色, 主内容区像卡片一样浮在上面。
	if s.variant == SidebarVariantInset {
		p.FillRound(&p.Rc, t.Sidebar, 0)
	}
	return 0
}

func (s *Sidebar) onPaintPanel(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	switch s.variant {
	case SidebarVariantFloating:
		p.FillRound(&p.Rc, t.Sidebar, t.RadiusLg())
		p.StrokeRound(&p.Rc, t.SidebarBorder, t.RadiusLg(), 1)
	case SidebarVariantInset:
		p.FillRound(&p.Rc, t.Sidebar, 0)
	default:
		p.FillRound(&p.Rc, t.Sidebar, 0)
		// border-r / border-l: 1 设备像素竖线贴内缘(否则 150% 下 1.5 物理像素会时粗时细)。
		x := p.Rc.Right - 1
		if s.side == SidebarRight {
			x = p.Rc.Left
		}
		sidebarFillVPx(p, x, 1, p.Rc.Top, p.Rc.Bottom, t.SidebarBorder)
	}
	return 0
}

func (s *Sidebar) onPaintMain(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	p.FillRound(&p.Rc, t.Background, 0)
	if s.variant == SidebarVariantInset {
		r := t.RadiusXl()
		p.FillRound(&p.Rc, t.Background, r)
		p.StrokeRound(&p.Rc, t.Border.ScaleAlpha(0.7), r, 1)
	}
	return 0
}

func (s *Sidebar) onPaintRail(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	st := xui.StateOf(hEle)
	hv := xui.Hover(hEle, s.railHover || st.Hover)
	if hv <= 0 {
		return 0
	}
	c := xui.Current()
	col := xui.Fade(xui.None, c.SidebarBorder, hv)
	// after:w-[2px]: 16px 细条正中 2px 宽的线。
	sidebarFillVPx(p, p.Rc.Left+7, 2, p.Rc.Top, p.Rc.Bottom, col)
	return 0
}

// ===== 分组 / 标签 =====

// SidebarGroup 侧栏分组: 可选标签 + 菜单列表(上游 SidebarGroup + SidebarGroupLabel + SidebarMenu)。
type SidebarGroup struct {
	*widget.LayoutEle
	sidebar *Sidebar
	label   *sidebarGroupLabel
	menu    *widget.LayoutEle
}

// Menu 返回分组里的菜单容器(竖向流, 子项间距 4), 可直接塞自定义控件。
func (g *SidebarGroup) Menu() *widget.LayoutEle {
	if g == nil {
		return nil
	}
	return g.menu
}

// SetLabel 设置分组标签文字(label 为空则隐藏标签)。
func (g *SidebarGroup) SetLabel(text string) *SidebarGroup {
	if g == nil {
		return g
	}
	if g.label == nil {
		if text == "" {
			return g
		}
		g.label = newSidebarGroupLabel(g, text)
		// 后建的标签默认排在菜单之后, 用布局序索引挪回第一位(InputGroup 同款)。
		g.label.SetZOrder(0)
		g.AdjustLayout()
		return g
	}
	g.label.setText(text)
	return g
}

// AddItem 往分组里追加菜单按钮。
func (g *SidebarGroup) AddItem(text, icon string, opts ...SidebarItemOption) *SidebarMenuButton {
	if g == nil || g.menu == nil {
		return nil
	}
	return g.sidebar.addItemIn(g.menu.Handle, text, icon, opts...)
}

// SetAction 给分组标签右侧加一个动作钮(20×20, 图标替换), 点击执行 fn。
// 只有分组带标签时才有效(上游的 SidebarGroupAction 是绝对定位在标签行上的)。
func (g *SidebarGroup) SetAction(icon string, fn func()) *SidebarGroup {
	if g == nil || g.label == nil {
		return g
	}
	g.label.setAction(icon, fn)
	return g
}

func (g *SidebarGroup) applyIconMode(icon bool) {
	if g == nil {
		return
	}
	if g.label != nil {
		g.label.Show(!icon)
	}
}

type sidebarGroupLabel struct {
	*widget.Element
	group  *SidebarGroup
	text   string
	action *sidebarIconAction
}

func newSidebarGroupLabel(g *SidebarGroup, text string) *sidebarGroupLabel {
	ele := widget.NewElement(0, 0, 0, sidebarLabelH, g.Handle)
	if ele == nil {
		return nil
	}
	l := &sidebarGroupLabel{Element: ele, group: g, text: text}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, sidebarLabelH)
	ele.AddEvent_Paint(l.onPaint)
	ele.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		l.layoutAction()
		return 0
	})
	return l
}

func (l *sidebarGroupLabel) setText(text string) {
	l.text = text
	l.Redraw(false)
}

func (l *sidebarGroupLabel) setAction(icon string, fn func()) {
	if l.action == nil {
		l.action = newSidebarIconAction(l.Handle, icon, fn)
		if l.group != nil && l.group.sidebar != nil {
			l.group.sidebar.actions = append(l.group.sidebar.actions, l.action)
		}
		l.action.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
			l.action = nil
			return 0
		})
	} else {
		l.action.icon = icon
		l.action.onClick = fn
		l.action.Redraw(false)
	}
	l.layoutAction()
}

func (l *sidebarGroupLabel) layoutAction() {
	if l.action == nil {
		return
	}
	a := l.action
	h := l.GetHeight()
	y := (h - sidebarRate) / 2
	// SidebarGroupAction: right-3 相对 group(border box), group 有 p-2 → 动作钮右缘 =
	// 标签右缘 - 4(与菜单动作的 right-1 同一条视觉线)。
	a.SetRectEx(l.GetWidth()-4-sidebarRate, y, sidebarRate, sidebarRate, false, xcc.AdjustLayout_No)
}

func (l *sidebarGroupLabel) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	p.FillRound(&p.Rc, t.Sidebar, 0)
	if l.text == "" {
		return 0
	}
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 12})
	if f != 0 {
		p.Draw.SetFont(f)
	}
	rc := xc.RECT{Left: p.Rc.Left + sidebarPad, Top: p.Rc.Top, Right: p.Rc.Right - sidebarPad - 28, Bottom: p.Rc.Bottom}
	sidebarText(p, l.text, rc, t.SidebarForeground.ScaleAlpha(0.7), f)
	return 0
}

// sidebarIconAction 是分组标签右侧的 20×20 动作钮(自绘 hover + 键盘激活)。
// 菜单项的动作钮不建子元素 —— 原生 Button 会截走子元素事件(实测), 见 SidebarMenuButton.actionHit。
type sidebarIconAction struct {
	*widget.Element
	icon    string
	onClick func()
	// hover 由宿主窗口的鼠标移动命中维护(普通 Element 的 MouseStay 在滚动区里不可靠, 见 DESIGN 21/158)。
	hover bool
}

func newSidebarIconAction(hParent int, icon string, fn func()) *sidebarIconAction {
	ele := widget.NewElement(0, 0, sidebarRate, sidebarRate, hParent)
	if ele == nil {
		return nil
	}
	a := &sidebarIconAction{Element: ele, icon: icon, onClick: fn}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.SetPadding(0, 0, 0, 0)
	ele.AddEvent_Paint(a.onPaint)
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		a.fire()
		*pbHandled = true
		return 0
	})
	ele.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		const vkReturn, vkSpace = 0x0D, 0x20
		if wParam != vkReturn && wParam != vkSpace {
			return 0
		}
		a.fire()
		*pbHandled = true
		return 0
	})
	ele.EnableFocus(true)
	ele.EnableSwitchFocus(true)
	return a
}

func (a *sidebarIconAction) fire() {
	if a != nil && a.onClick != nil {
		a.onClick()
	}
}

func (a *sidebarIconAction) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)
	hv := xui.Hover(hEle, a.hover || st.Hover)
	bg := xui.Fade(xui.None, t.SidebarAccent, hv)
	p.FillRound(&p.Rc, t.Sidebar, 0)
	if !bg.IsNone() {
		p.FillRound(&p.Rc, bg, t.RadiusMd())
	}
	if st.Focused && xui.FocusedByKeyboard() {
		// focus-visible:ring-2 ring-sidebar-ring(20×20 动作钮上环带 2px)。
		p.FillRound(&p.Rc, t.SidebarRing, t.RadiusMd())
		const inset = int32(2)
		inner := xc.RECT{Left: p.Rc.Left + inset, Top: p.Rc.Top + inset,
			Right: p.Rc.Right - inset, Bottom: p.Rc.Bottom - inset}
		if inner.Right-inner.Left > 2 && inner.Bottom-inner.Top > 2 {
			innerCol := t.Sidebar
			if !bg.IsNone() {
				innerCol = bg
			}
			p.FillRound(&inner, innerCol, t.RadiusMd()-inset)
		}
	}
	if a.icon != "" {
		rc := xc.RECT{
			Left:   p.Rc.Left + (p.W()-sidebarItemIcon)/2,
			Top:    p.Rc.Top + (p.H()-sidebarItemIcon)/2,
			Right:  p.Rc.Left + (p.W()-sidebarItemIcon)/2 + sidebarItemIcon,
			Bottom: p.Rc.Top + (p.H()-sidebarItemIcon)/2 + sidebarItemIcon,
		}
		p.Icon(a.icon, &rc, t.SidebarForeground, 2)
	}
	return 0
}

// ===== 菜单按钮 =====

type sidebarItemCfg struct {
	active     bool
	variant    SidebarItemVariant
	size       SidebarSize
	badge      string
	tooltip    string
	disabled   bool
	onClick    func()
	action     string
	actionFn   func()
	actionShow bool // showOnHover
}

// SidebarItemOption 菜单按钮配置项。
type SidebarItemOption func(*sidebarItemCfg)

// WithSidebarItemActive 初始选中态(视觉同上游 data-[active=true]: bg-sidebar-accent + 加粗)。
func WithSidebarItemActive(active bool) SidebarItemOption {
	return func(c *sidebarItemCfg) { c.active = active }
}

// WithSidebarItemVariant 设置菜单按钮变体(默认 SidebarItemDefault)。
func WithSidebarItemVariant(v SidebarItemVariant) SidebarItemOption {
	return func(c *sidebarItemCfg) { c.variant = v }
}

// WithSidebarItemSize 设置尺寸档(默认 SidebarSizeDefault; Header/Footer 常用 Lg)。
func WithSidebarItemSize(size SidebarSize) SidebarItemOption {
	return func(c *sidebarItemCfg) { c.size = size }
}

// WithSidebarItemBadge 在菜单项右侧显示徽章文字(如未读数)。
func WithSidebarItemBadge(text string) SidebarItemOption {
	return func(c *sidebarItemCfg) { c.badge = text }
}

// WithSidebarItemTooltip 折叠成图标栏时, 悬停该菜单项显示的提示文字。
func WithSidebarItemTooltip(text string) SidebarItemOption {
	return func(c *sidebarItemCfg) { c.tooltip = text }
}

// WithSidebarItemDisabled 禁用菜单项(不响应点击, 整体 50%)。
func WithSidebarItemDisabled() SidebarItemOption {
	return func(c *sidebarItemCfg) { c.disabled = true }
}

// WithSidebarItemOnClick 点击回调。
func WithSidebarItemOnClick(fn func()) SidebarItemOption {
	return func(c *sidebarItemCfg) { c.onClick = fn }
}

// WithSidebarItemAction 在菜单项右侧加一个 20×20 动作钮(如更多操作), 点击只触发 fn。
func WithSidebarItemAction(icon string, fn func()) SidebarItemOption {
	return func(c *sidebarItemCfg) { c.action, c.actionFn = icon, fn }
}

// WithSidebarItemActionOnHover 动作钮只在悬停/选中菜单项时显示(上游 showOnHover)。
func WithSidebarItemActionOnHover() SidebarItemOption {
	return func(c *sidebarItemCfg) { c.actionShow = true }
}

// SidebarMenuButton 侧栏菜单按钮: 原生 Button 载体(点击/键盘焦点直接用原生语义),
// 外观全部自绘: 16px 图标 + 文字(超宽省略号) + 右侧徽章 / 动作钮。
type SidebarMenuButton struct {
	*widget.Button
	sidebar *Sidebar
	cfg     sidebarItemCfg
	icon    string
	sub     bool
	// 有子菜单时, 条目右侧画 chevron(展开转 90°), 点击整行开合子菜单。
	subMenu *SidebarMenuSub
	subOpen bool
	subKey  int32
}

// NewSidebarMenuButton 创建独立菜单按钮(也可直接放进 Header/Footer, 不绑定侧栏折叠态)。
func NewSidebarMenuButton(x, y, cx, cy int32, text, icon string, hParent int, opts ...SidebarItemOption) *SidebarMenuButton {
	cfg := sidebarItemCfg{size: SidebarSizeDefault}
	for _, o := range opts {
		o(&cfg)
	}
	h := sidebarItemHeight(cfg.size)
	btn := widget.NewButton(x, y, cx, h, text, hParent)
	if btn == nil {
		return nil
	}
	m := &SidebarMenuButton{Button: btn, cfg: cfg, icon: icon}
	m.buildHeight(h)
	btn.SetTextColor(0)
	btn.EnableBkTransparent(true)
	btn.EnableDrawBorder(false)
	btn.SetBorderSize(0, 0, 0, 0)
	btn.SetPadding(0, 0, 0, 0)
	if cfg.disabled {
		btn.Enable(false)
	}
	btn.AddEvent_Paint(m.onPaint)
	btn.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if m.cfg.disabled {
			return 0
		}
		// 动作钮与整行共用一个原生 Button(子元素按钮收不到点击, 实测): 鼠标点击时按
		// 光标是否落在右侧动作区来分流; 键盘激活(Tab/Enter)一律算整行。
		if !xui.FocusedByKeyboard() && m.cursorInAction() {
			if m.cfg.actionFn != nil {
				m.cfg.actionFn()
			}
			return 0
		}
		if m.subMenu != nil && !(m.sidebar != nil && m.sidebar.iconMode()) {
			m.ToggleSub()
		}
		if m.cfg.onClick != nil {
			m.cfg.onClick()
		}
		return 0
	})
	xui.AttachWindow(btn.Handle)
	return m
}

// addItemIn 在指定容器里建一个绑定侧栏状态的菜单按钮。
func (s *Sidebar) addItemIn(hParent int, text, icon string, opts ...SidebarItemOption) *SidebarMenuButton {
	m := NewSidebarMenuButton(0, 0, 0, 0, text, icon, hParent, opts...)
	if m == nil {
		return nil
	}
	m.sidebar = s
	s.items = append(s.items, m)
	if s.iconMode() {
		m.applyIconMode(true)
	}
	s.syncTooltips()
	s.panel.AdjustLayout()
	s.content.AdjustLayout()
	return m
}

func (m *SidebarMenuButton) buildHeight(h int32) {
	m.SetHeight(h)
	m.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	m.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
}

// itemFontSize 返回菜单项字号(text-sm=14 / text-xs=12)。
func (m *SidebarMenuButton) itemFontSize() int32 {
	if m != nil && (m.cfg.size == SidebarSizeSm) {
		return 12
	}
	return 14
}

func (m *SidebarMenuButton) tooltipText() string {
	if m == nil {
		return ""
	}
	return m.cfg.tooltip
}

// SetActive 设置选中态(不触发点击回调)。
func (m *SidebarMenuButton) SetActive(active bool) *SidebarMenuButton {
	if m == nil {
		return m
	}
	m.cfg.active = active
	m.Redraw(false)
	return m
}

// Active 报告是否选中。
func (m *SidebarMenuButton) Active() bool { return m != nil && m.cfg.active }

// SetIcon 设置/替换前置图标(空串移除)。
func (m *SidebarMenuButton) SetIcon(icon string) *SidebarMenuButton {
	if m == nil {
		return m
	}
	m.icon = icon
	m.Redraw(false)
	return m
}

// SetBadge 设置/清空徽章文字。
func (m *SidebarMenuButton) SetBadge(text string) *SidebarMenuButton {
	if m == nil {
		return m
	}
	m.cfg.badge = text
	m.Redraw(false)
	return m
}

// SetTooltip 设置折叠态提示文字(空串移除)。
func (m *SidebarMenuButton) SetTooltip(text string) *SidebarMenuButton {
	if m == nil {
		return m
	}
	m.cfg.tooltip = text
	if m.sidebar != nil {
		m.sidebar.syncTooltips()
	}
	return m
}

// SetAction 设置/替换右侧动作钮; icon 传空串移除。
func (m *SidebarMenuButton) SetAction(icon string, fn func()) *SidebarMenuButton {
	if m == nil {
		return m
	}
	m.cfg.action, m.cfg.actionFn = icon, fn
	m.Redraw(false)
	return m
}

// SetDisabled 设置禁用态。
func (m *SidebarMenuButton) SetDisabled(disabled bool) *SidebarMenuButton {
	if m == nil {
		return m
	}
	m.cfg.disabled = disabled
	m.Enable(!disabled)
	m.Redraw(false)
	return m
}

// AddSub 在菜单项后面追加子菜单(展开态可见, 图标栏形态隐藏), 返回子菜单容器。
// 必须**紧跟所属 AddItem 调用**(子菜单按父容器的子项顺序排, 之后再 AddItem 会排到子菜单前面)。
// 默认展开; 条目右侧会自动出现 chevron(展开时转 90°), 点击整行开合。
// 需要默认收起时在 AddSub() 后调用 SetSubOpen(false)。
func (m *SidebarMenuButton) AddSub() *SidebarMenuSub {
	if m == nil {
		return nil
	}
	sub := newSidebarMenuSub(m)
	if sub == nil {
		return nil
	}
	sub.open = true
	m.subMenu = sub
	m.subOpen = true
	m.subKey = xui.NextPresenceKey()
	// 预热过渡值(同侧栏宽度): 展开态起始时键必须是 1, 第一次收起才会动画。
	owner := m.Handle
	if m.sidebar != nil {
		owner = m.sidebar.presenceOwner()
	}
	xui.Presence(owner, m.subKey, true, 0, nil, nil)
	if m.sidebar != nil {
		m.sidebar.subs = append(m.sidebar.subs, sub)
		if m.sidebar.iconMode() {
			sub.applyIconMode(true)
		}
		m.sidebar.panel.AdjustLayout()
		m.sidebar.content.AdjustLayout()
	}
	m.Redraw(false)
	return sub
}

// SubOpen 报告子菜单当前是否展开。
func (m *SidebarMenuButton) SubOpen() bool { return m != nil && m.subMenu != nil && m.subOpen }

// SetSubOpen 展开/收起子菜单(200ms 高度过渡, 同上游 Collapsible); 图标栏形态下只记状态。
func (m *SidebarMenuButton) SetSubOpen(open bool) *SidebarMenuButton {
	if m == nil || m.subMenu == nil || m.subOpen == open {
		return m
	}
	m.subOpen = open
	m.subMenu.open = open
	if m.sidebar != nil && m.sidebar.iconMode() {
		m.subMenu.applyIconMode(true)
		m.Redraw(false)
		return m
	}
	m.animateSub(open)
	m.Redraw(false)
	return m
}

// ToggleSub 切换子菜单开合。
func (m *SidebarMenuButton) ToggleSub() *SidebarMenuButton {
	if m == nil || m.subMenu == nil {
		return m
	}
	return m.SetSubOpen(!m.subOpen)
}

// animateSub 用 Presence 对子菜单高度做 200ms 过渡。
func (m *SidebarMenuButton) animateSub(open bool) {
	sub := m.subMenu
	if sub == nil {
		return
	}
	owner := m.Handle
	if m.sidebar != nil {
		owner = m.sidebar.presenceOwner()
	}
	// 先按内容量一次自然高度(Auto)。
	sub.Show(true)
	sub.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	m.reflowSub()
	target := sub.GetHeight()
	anim := sidebarAnimMs
	if target <= 0 || anim <= 0 {
		// 隐藏页/宽度未定: 直接落终态, 并把键抬到目标值(避免下次首帧跳)。
		xui.Presence(owner, m.subKey, open, 0, nil, nil)
		m.applySubFinal(open)
		return
	}
	if open {
		sub.SetHeight(0)
		sub.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 0)
	} else {
		sub.SetHeight(target)
		sub.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, target)
	}
	m.reflowSub()
	apply := func(v float64) {
		h := int32(math.Round(float64(target) * v))
		if sub.GetHeight() == h {
			return
		}
		sub.SetHeight(h)
		sub.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
		m.reflowSub()
	}
	settle := func(float64) { m.applySubFinal(open) }
	v := xui.Presence(owner, m.subKey, open, anim, apply, settle)
	if (open && v >= 1) || (!open && v <= 0) {
		settle(v)
	}
}

// applySubFinal 落子菜单终态: 展开恢复 Auto 高, 收起隐藏并清高。
func (m *SidebarMenuButton) applySubFinal(open bool) {
	sub := m.subMenu
	if sub == nil {
		return
	}
	if open {
		sub.Show(true)
		sub.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	} else {
		sub.Show(false)
		sub.SetHeight(0)
		sub.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 0)
	}
	m.reflowSub()
	if m.sidebar != nil {
		m.sidebar.redraw()
	}
}

// reflowSub 子菜单高度变化后重排所在的菜单/分组/滚动内容。
func (m *SidebarMenuButton) reflowSub() {
	if m == nil || m.subMenu == nil {
		return
	}
	if p := xc.XWidget_GetParentEle(m.subMenu.Handle); p != 0 {
		xc.XEle_AdjustLayout(p)
	}
	if m.sidebar != nil {
		m.sidebar.panel.AdjustLayout()
		m.sidebar.content.AdjustLayout()
	}
}

// actionRightInset 动作钮距右缘的间距; 有子菜单时让开 chevron(right 8 + 16 + 4)。
func (m *SidebarMenuButton) actionRightInset() int32 {
	if m != nil && m.subMenu != nil {
		return sidebarChevronX + sidebarChevron + 4
	}
	return 4
}

// actionShown 报告动作钮此刻是否可见(showOnHover 时随菜单项悬停/选中显隐)。
// 上游 showOnHover 只在 hover / focus-within / 打开态显示; 选中态只换图标颜色, 不显示动作钮。
func (m *SidebarMenuButton) actionShown() bool {
	if m == nil || m.cfg.action == "" {
		return false
	}
	if m.sidebar != nil && m.sidebar.iconMode() {
		return false
	}
	if !m.cfg.actionShow {
		return true
	}
	st := xui.StateOf(m.Handle)
	return st.Hover || st.Pressed
}

// actionHit 判断元素客户区坐标是否落在右侧动作钮上(20×20, right-1, 垂直居中)。
func (m *SidebarMenuButton) actionHit(x, y int32) bool {
	if !m.actionShown() {
		return false
	}
	ax := m.GetWidth() - m.actionRightInset() - sidebarRate
	ay := (m.GetHeight() - sidebarRate) / 2
	return x >= ax && x < ax+sidebarRate && y >= ay && y < ay+sidebarRate
}

// cursorInAction 用物理屏幕坐标判断这次点击是否落在动作钮上。
//
// 不用事件里的 pPt: Button 的鼠标事件坐标口径与元素客户区不一致(实测按元素坐标算
// 永远命中不了右侧动作区)。这里按 DESIGN 157 的做法换算: 光标(物理屏幕) − 窗口客户区
// 原点(物理) − 元素物理矩形, 再除以 DPI 得到元素内逻辑坐标。
func (m *SidebarMenuButton) cursorInAction() bool {
	if m == nil || !m.actionShown() {
		return false
	}
	hwnd := xc.XWidget_GetHWND(m.Handle)
	if hwnd == 0 {
		return false
	}
	var cursor wapi.POINT
	if !wapi.GetCursorPos(&cursor) {
		return false
	}
	var origin wapi.POINT
	if !wapi.ClientToScreen(hwnd, &origin) {
		return false
	}
	var rc xc.RECT
	xc.XEle_GetWndClientRectDPI(m.Handle, &rc)
	scale := 1.0
	if hw := xui.HostWindowOf(m.Handle); hw != 0 {
		if dpi := xc.XWnd_GetDPI(hw); dpi > 0 {
			scale = float64(dpi) / 96
		}
	}
	x := float64(cursor.X-origin.X-rc.Left) / scale
	y := float64(cursor.Y-origin.Y-rc.Top) / scale
	return m.actionHit(int32(x), int32(y))
}

// applyIconMode 把菜单项压成 32×32 图标格(上游 group-data-[collapsible=icon]:size-8! p-2!)。
func (m *SidebarMenuButton) applyIconMode(icon bool) {
	if m == nil {
		return
	}
	h := sidebarItemHeight(m.cfg.size)
	if icon {
		h = 32
	}
	if m.GetHeight() != h {
		m.SetHeight(h)
		m.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	}
	m.Redraw(false)
}

// ===== 子菜单 =====

// SidebarMenuSub 子菜单容器: 左侧 1px 竖线 + 缩进(上游 mx-3.5 border-l px-2.5 py-0.5)。
type SidebarMenuSub struct {
	*widget.LayoutEle
	sidebar *Sidebar
	open    bool
}

func newSidebarMenuSub(m *SidebarMenuButton) *SidebarMenuSub {
	parent := xc.XWidget_GetParentEle(m.Handle)
	if parent == 0 {
		return nil
	}
	ele := widget.NewLayoutEle(0, 0, 0, 0, parent)
	if ele == nil {
		return nil
	}
	sub := &SidebarMenuSub{LayoutEle: ele, sidebar: m.sidebar}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false)
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.SetSpace(sidebarGroupGap)
	ele.SetPadding(sidebarSubPadL, 0, sidebarSubPadR, 0)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ele.LayoutItem_SetMargin(sidebarSubMarX, 2, sidebarSubMarX, 2)
	ele.AddEvent_Paint(sub.onPaint)
	return sub
}

// AddItem 追加子菜单条目(高度固定 28, 默认 text-sm; 传 WithSidebarItemSize(SidebarSizeSm) 用 text-xs)。
func (sub *SidebarMenuSub) AddItem(text, icon string, opts ...SidebarItemOption) *SidebarMenuButton {
	if sub == nil {
		return nil
	}
	cfg := append([]SidebarItemOption{func(c *sidebarItemCfg) {
		c.size = SidebarSizeDefault
	}}, opts...)
	m := NewSidebarMenuButton(0, 0, 0, 0, text, icon, sub.Handle, cfg...)
	if m == nil {
		return nil
	}
	m.sub = true
	m.sidebar = sub.sidebar
	m.SetHeight(28)
	m.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 28)
	if sub.sidebar != nil && sub.sidebar.iconMode() {
		sub.applyIconMode(true)
	}
	// 子项不参与折叠 tooltip(整个子菜单在图标栏里是隐藏的)。
	return m
}

func (sub *SidebarMenuSub) applyIconMode(icon bool) {
	if sub == nil {
		return
	}
	if icon {
		sub.Show(false)
		return
	}
	if sub.open {
		sub.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
		sub.Show(true)
	} else {
		sub.Show(false)
	}
}

func (sub *SidebarMenuSub) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	// border-l: 1 设备像素竖线, 从元素左缘画起(元素已经缩进 14)。
	sidebarFillVPx(p, p.Rc.Left, 1, p.Rc.Top, p.Rc.Bottom, t.SidebarBorder)
	return 0
}

// ===== 菜单按钮绘制 =====

func (m *SidebarMenuButton) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)
	iconMode := m.sidebar != nil && m.sidebar.iconMode()

	bg, fore, border := m.resolveColors(t, st)

	// 透明元素重绘不擦底(同 Toggle): 先铺面板底色, 再叠自身状态色。
	p.FillRound(&p.Rc, t.Sidebar, 0)
	focusRing := st.Focused && !st.Disabled && xui.FocusedByKeyboard()
	if focusRing {
		// focus-visible:ring-2 ring-sidebar-ring: 环带铺满元素, 内芯内缩 2px 后重新铺底色与状态色。
		p.FillRound(&p.Rc, t.SidebarRing, t.RadiusMd())
		inner := xc.RECT{
			Left: p.Rc.Left + sidebarRingW, Top: p.Rc.Top + sidebarRingW,
			Right: p.Rc.Right - sidebarRingW, Bottom: p.Rc.Bottom - sidebarRingW,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			r := t.RadiusMd() - sidebarRingW
			if r < 0 {
				r = 0
			}
			if bg.IsNone() {
				p.FillRound(&inner, t.Sidebar, r)
			} else {
				p.FillRound(&inner, bg, r)
			}
		}
	} else {
		if !bg.IsNone() {
			p.FillRound(&p.Rc, bg, t.RadiusMd())
		}
		if !border.IsNone() {
			p.StrokeRound(&p.Rc, border, t.RadiusMd(), 1)
		}
	}

	if iconMode {
		// 图标栏: 只画居中的 16px 图标。
		if m.iconPath() != "" {
			cx := (p.Rc.Left + p.Rc.Right) / 2
			cy := (p.Rc.Top + p.Rc.Bottom) / 2
			rc := xc.RECT{Left: cx - sidebarItemIcon/2, Top: cy - sidebarItemIcon/2,
				Right: cx - sidebarItemIcon/2 + sidebarItemIcon, Bottom: cy - sidebarItemIcon/2 + sidebarItemIcon}
			p.Icon(m.iconPath(), &rc, fore, 2)
		}
		return 0
	}

	fontPx := m.itemFontSize()
	if m.sub {
		fontPx = 14
		if m.cfg.size == SidebarSizeSm {
			fontPx = 12
		}
	}
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: fontPx, Bold: m.cfg.active})
	if f != 0 {
		p.Draw.SetFont(f)
	}
	cy := (p.Rc.Top + p.Rc.Bottom) / 2

	iconColor := fore
	if m.sub {
		// SidebarMenuSubButton: [&>svg]:text-sidebar-accent-foreground —— 子项图标恒 accent 前景。
		iconColor = t.SidebarAccentForeground
	}

	// 图标 + 文字(上游 p-2 gap-2)。
	textLeft := p.Rc.Left + sidebarPad
	if ip := m.iconPath(); ip != "" {
		rc := xc.RECT{Left: textLeft, Top: cy - sidebarItemIcon/2,
			Right: textLeft + sidebarItemIcon, Bottom: cy - sidebarItemIcon/2 + sidebarItemIcon}
		p.Icon(ip, &rc, iconColor, 2)
		textLeft += sidebarItemIcon + sidebarIconGap
	}
	badgeFont := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 12, Bold: true})
	// 右侧预留: 动作钮 / 子菜单 chevron / 徽章(绝对定位, 这里也给它让位, 长文本不压它们)。
	textRight := p.Rc.Right - sidebarPad
	badgeRight := int32(4)
	if m.subMenu != nil {
		badgeRight = sidebarChevronX + sidebarChevron + 4
	}
	if m.cfg.action != "" {
		textRight = p.Rc.Right - m.actionRightInset() - sidebarRate - 4
	} else if badge := m.cfg.badge; badge != "" {
		textRight = p.Rc.Right - badgeRight - sidebarBadgeWidth(badge, badgeFont) - 4
	} else if m.subMenu != nil {
		textRight = p.Rc.Right - sidebarChevronX - sidebarChevron - 4
	}
	if textRight > textLeft {
		rc := xc.RECT{Left: textLeft, Top: p.Rc.Top, Right: textRight, Bottom: p.Rc.Bottom}
		sidebarText(p, m.GetText(), rc, fore, f)
	}

	if badge := m.cfg.badge; badge != "" && m.cfg.action == "" {
		bw := sidebarBadgeWidth(badge, badgeFont)
		x := p.Rc.Right - badgeRight - bw
		rc := xc.RECT{Left: x, Top: cy - sidebarBadgeH/2, Right: x + bw, Bottom: cy - sidebarBadgeH/2 + sidebarBadgeH}
		sidebarTextAlign(p, badge, rc, fore, badgeFont, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
	}

	// 动作钮: 20×20(right-1, 垂直居中), 图标 16。底色与整行相同, 不单独铺块。
	if m.cfg.action != "" && m.actionShown() {
		ax := p.Rc.Right - m.actionRightInset() - sidebarRate
		ay := cy - sidebarRate/2
		ir := xc.RECT{
			Left:   ax + (sidebarRate-sidebarItemIcon)/2,
			Top:    ay + (sidebarRate-sidebarItemIcon)/2,
			Right:  ax + (sidebarRate-sidebarItemIcon)/2 + sidebarItemIcon,
			Bottom: ay + (sidebarRate-sidebarItemIcon)/2 + sidebarItemIcon,
		}
		p.Icon(m.cfg.action, &ir, fore, 2)
	}

	// 子菜单 chevron: ml-auto, 展开时转 90°(ChevronRight → 朝下)。
	if m.subMenu != nil {
		left := p.Rc.Right - sidebarChevronX - sidebarChevron
		rc := xc.RECT{Left: left, Top: cy - sidebarChevron/2, Right: left + sidebarChevron, Bottom: cy - sidebarChevron/2 + sidebarChevron}
		angle := 0.0
		if m.subOpen {
			angle = math.Pi / 2
		}
		p.IconRotated(icons.IconChevronRight, &rc, fore, 2, angle)
	}
	return 0
}

func (m *SidebarMenuButton) iconPath() string {
	return m.icon
}

// resolveColors 解析壳体/文字颜色, 对应 sidebarMenuButtonVariants:
// default 悬停 accent, outline 白底 + 1px sidebar-border(悬停边框转 accent), 选中 accent。
func (m *SidebarMenuButton) resolveColors(t *xui.Theme, st xui.State) (bg, fore, border xui.Color) {
	bg, bgHV := xui.None, t.SidebarAccent
	fore, foreHV := t.SidebarForeground, t.SidebarAccentForeground

	if m.sub {
		// SidebarMenuSubButton: 悬停/选中都是 accent, 图标恒 accent 前景。
		if m.cfg.active {
			bg, bgHV = t.SidebarAccent, t.SidebarAccent
			fore, foreHV = t.SidebarAccentForeground, t.SidebarAccentForeground
		}
	} else if m.cfg.variant == SidebarItemOutline {
		bg, bgHV = t.Background, t.SidebarAccent
		fore, foreHV = t.SidebarForeground, t.SidebarAccentForeground
		border = t.SidebarBorder
	}
	if m.cfg.active {
		bg, bgHV = t.SidebarAccent, t.SidebarAccent
		fore, foreHV = t.SidebarAccentForeground, t.SidebarAccentForeground
	}
	hovered := (st.Hover || st.Pressed) && !st.Disabled
	hv := xui.Hover(m.Handle, hovered)
	bg = xui.Fade(bg, bgHV, hv)
	fore = xui.Fade(fore, foreHV, hv)
	if !border.IsNone() {
		border = xui.Fade(border, t.SidebarAccent, hv)
	}
	if st.Disabled {
		if !bg.IsNone() {
			bg = bg.ScaleAlpha(0.5)
		}
		fore = fore.Mix(t.Sidebar, 0.5)
		if !border.IsNone() {
			border = border.ScaleAlpha(0.5)
		}
	}
	return bg, fore, border
}

// icon 字段由 NewSidebarMenuButton 保存(原生 Button 没有"图标 + 文字"常驻形态)。

func sidebarItemHeight(size SidebarSize) int32 {
	switch size {
	case SidebarSizeSm:
		return 28
	case SidebarSizeLg:
		return 48
	default:
		return 32
	}
}

func sidebarBadgeWidth(text string, font int) int32 {
	w := xui.TextWidth(text, font) + sidebarBadgePad*2
	if w < sidebarBadgeH {
		w = sidebarBadgeH
	}
	return w
}

// sidebarText 在矩形里画单行左对齐文字(垂直居中 + NoWrap + 省略号), 中文上移半像素。
func sidebarText(p *xui.Painter, text string, rc xc.RECT, color xui.Color, font int) {
	sidebarTextAlign(p, text, rc, color, font, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap|xcc.TextTrimming_EllipsisCharacter)
}

func sidebarTextAlign(p *xui.Painter, text string, rc xc.RECT, color xui.Color, font int, align xcc.TextFormatFlag_) {
	if text == "" || color.IsNone() {
		return
	}
	if font != 0 {
		p.Draw.SetFont(font)
	}
	dy := textVDy(text)
	rcf := xc.RECTF{
		Left:   float32(rc.Left),
		Top:    float32(rc.Top) - dy,
		Right:  float32(rc.Right),
		Bottom: float32(rc.Bottom) - dy,
	}
	p.TextF(text, &rcf, color, align)
}

// sidebarFillVPx 在元素客户区里画一条 w 逻辑像素宽、对齐整设备像素的竖线(从 x 向右)。
func sidebarFillVPx(p *xui.Painter, x, w, top, bottom int32, color xui.Color) {
	if color.IsNone() || w <= 0 {
		return
	}
	sc := p.Scale
	if sc <= 0 {
		sc = 1
	}
	c := math.Floor(float64(x) * sc)
	c2 := c + math.Max(1, math.Round(float64(w)*sc))
	rc := xc.RECTF{
		Left:   float32(c / sc),
		Top:    float32(top),
		Right:  float32(c2 / sc),
		Bottom: float32(bottom),
	}
	p.Draw.FillRectColorF(&rc, uint32(color))
}

// sidebarFillHPx 在元素客户区里画一条 1 设备像素高、以 centerY 为中心的横线。
func sidebarFillHPx(p *xui.Painter, left, right, centerY int32, color xui.Color) {
	if color.IsNone() {
		return
	}
	sc := p.Scale
	if sc <= 0 {
		sc = 1
	}
	c := math.Floor(float64(centerY) * sc)
	rc := xc.RECTF{
		Left:   float32(left),
		Top:    float32(c / sc),
		Right:  float32(right),
		Bottom: float32((c + 1) / sc),
	}
	p.Draw.FillRectColorF(&rc, uint32(color))
}

func sidebarResizeCursor() uintptr {
	return wapi.LoadImageW(0, uintptr(wapi.IDC_SIZEWE), wapi.IMAGE_CURSOR, 0, 0,
		wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)
}
