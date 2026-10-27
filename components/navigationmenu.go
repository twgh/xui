package ui

import (
	"math"
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

// NavigationMenu 的几何常量(逻辑像素), 对齐 shadcn navigation-menu.tsx(radix-nova):
// 触发区 h-9(36) / px-4(16), 列表 gap-1(4), 视口 mt-1.5(6) + p-2(8);
// 内容链接行是文档示例的 ListItem(p-3 + 标题 20 + 间距 4 + 一行说明 20)。
const (
	navH         int32 = 36 // 触发区 / 列表高
	navGap       int32 = 4  // 列表 gap-1
	navPadX      int32 = 16 // 触发区 px-4
	navLinkPadX  int32 = 8  // 顶级普通链接 p-2
	navIcon      int32 = 12 // chevron size-3
	navIconGap   int32 = 4  // chevron ml-1
	navPanelGap  int32 = 6  // 视口 mt-1.5
	navContentPd int32 = 8  // 视口 p-2
	navDefaultW  int32 = 400
	navLinkPd    int32 = 12 // ListItem p-3
	navLinkTitle int32 = 20 // text-sm 行高
	navLinkGap   int32 = 4  // 标题与说明的间距
	navLinkH     int32 = navLinkPd*2 + navLinkTitle + navLinkGap + navLinkTitle
	navLinkOnlyH int32 = navLinkPd*2 + navLinkTitle
	navSlide     int32 = 208 // 内容切换的横向滑入距离(slide-in-from-*-52 = 208)
)

// NavigationMenu 的过渡时长, 对齐 radix NavigationMenu.Root 的默认值:
// delayDuration 200 / 离开视口后 150ms 收起 / skipDelayDuration 300;
// chevron rotate-180 是 duration-300, 内容切换动画 duration-200。
const (
	navOpenDelay  = 200 * time.Millisecond
	navCloseDelay = 150 * time.Millisecond
	navSkipDelay  = 300 * time.Millisecond
	navChevronDur = 300 * time.Millisecond
	navContentDur = 200 * time.Millisecond
)

const (
	navVKLeft  uintptr = 0x25
	navVKUp    uintptr = 0x26
	navVKRight uintptr = 0x27
	navVKDown  uintptr = 0x28
	navVKHome  uintptr = 0x24
	navVKEnd   uintptr = 0x23
	navVKEnter uintptr = 0x0D
	navVKSpace uintptr = 0x20
	navVKTab   uintptr = 0x09
	navVKEsc   uintptr = 0x1B
)

// navFont 返回触发区/链接的字体(text-sm = 14)。
func navFont() int {
	t := xui.Current()
	return xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 14})
}

// NavigationMenu 导航菜单: 一排触发区 + 一个**共享**的自绘视口, 对应 shadcn
// navigation-menu.tsx(radix-nova, viewport 模式)。
//
// 与 shadcn 对齐:
//   - 触发区 h-9 px-4 rounded-md text-sm; 悬停 bg-accent, 展开态 bg-accent/50
//     (悬停时回到全亮); chevron 12 随展开旋转 180°(300ms); 普通链接是 p-2 的菜单项;
//   - 视口是一块共享弹层: 贴在列表左下方(mt-1.5), 宽高随当前菜单的内容变化,
//     不会为每个触发区各开一块; 内容切换时新页面从切换方向滑入 208px 并淡入(200ms);
//   - 交互: 悬停触发区 200ms 后打开; 离开触发区/视口 150ms 后收起(进入另一者即取消);
//     打开后悬停其它触发区立即切换; 关闭后 300ms 内切到其它触发区立即打开;
//     点击触发区开合; 点击内容里的链接后关闭; ESC / Tab / 宿主窗口失焦、移动、缩放关闭。
//   - 键盘: ←→ 在顶级项之间移动焦点, ↓/Enter/Space 打开, ↓ 再进入内容;
//     内容里 ↑↓←→/Home/End 移动高亮, Enter 激活, ESC 关闭。
//
// 视口是独立弹层窗口(与 Popover 共用 popoverPanel), 所以**不会抢焦点** ——
// 悬停打开时用户正在输入的控件不会丢光标。内容里的链接由本组件自己维护键盘高亮,
// 不依赖原生控件焦点; 自定义放进内容的其它控件只支持鼠标操作。
//
// 用法:
//
//	nav := ui.NewNavigationMenu(card.Content())
//	nav.AddMenu("开始使用", func(c *ui.NavigationMenuContent) {
//	    c.Link("组件介绍", "可复用的自绘组件", func() { ... })
//	    c.Link("安装", "把组件源码拷进你的项目", func() { ... })
//	}, ui.WithNavigationMenuWidth(400))
//	nav.AddLink("文档", func() { ... })
type NavigationMenu struct {
	hWnd  int
	root  *widget.LayoutEle
	items []*navItem
	open  int // 当前打开的菜单下标; -1 = 全关

	panel   *popoverPanel
	content *widget.LayoutEle

	// isOpenDelayed 对齐 radix 的 isOpenDelayed: 刚关掉后 300ms 内为 true,
	// 此时悬停要等 200ms 才打开; 打开过一次后为 false, 悬停其它触发区立即切换。
	isOpenDelayed bool
	openTimer     *time.Timer
	closeTimer    *time.Timer
	skipTimer     *time.Timer

	kbLink      *NavigationMenuLink // 内容里键盘高亮的链接(nil = 无)
	hoverLink   *NavigationMenuLink // 指针悬停的链接(非激活弹层里原生 hover 状态不更新, 自己轮询)
	watchTimer  *time.Timer
	everInside  bool // 指针在本次打开期间是否进过导航(程序化打开且指针一直在外面时不要自动收起)
	lastEmitted string
	changes     []func(value string)
}

// navKind 区分顶级项的类型。
type navKind int

const (
	navKindMenu navKind = iota // 带内容的菜单(chevron)
	navKindLink                // 顶级普通链接(无弹层)
)

// navItem 是一个顶级项: 触发按钮 + 它自己的内容页(仅菜单)。
type navItem struct {
	nav   *NavigationMenu
	kind  navKind
	title string
	value string

	btn    *Button
	width  int32
	titleW int32
	fontH  int

	panelW int32 // 视口宽(该菜单的内容宽)
	build  func(*NavigationMenuContent)
	entry  *NavigationMenuContent
	links  []*NavigationMenuLink

	open         bool
	disabled     bool
	suppressOpen bool // ESC/Tab/点击关掉后, 指针还停在触发区上时不要被 MouseMove 立刻重开
	chevron      float64
	chevKey      int32

	onClick func()
}

// NavigationMenuItem 是 AddMenu / AddLink 返回的顶级项句柄。
type NavigationMenuItem struct {
	it *navItem
}

// NavigationMenuContent 是内容页的构造上下文: 用 Link 添加键盘可导航的链接行,
// Handle 取布局容器句柄以放入任意自定义控件。
type NavigationMenuContent struct {
	nav  *NavigationMenu
	item *navItem
	ele  *widget.LayoutEle
}

// NavigationMenuLink 是内容页里的一行"标题 + 说明"链接。
type NavigationMenuLink struct {
	*widget.Element
	nav   *NavigationMenu
	title string
	desc  string
	click func()
}

// NavigationMenuItemOption 配置顶级菜单项。
type NavigationMenuItemOption func(*navItem)

// WithNavigationMenuValue 指定该项的值(默认用标题); Value/SetValue/OnValueChange 用它。
func WithNavigationMenuValue(v string) NavigationMenuItemOption {
	return func(it *navItem) {
		if v != "" {
			it.value = v
		}
	}
}

// WithNavigationMenuWidth 指定该菜单的内容宽(默认 400); 视口宽度随打开的菜单变化。
func WithNavigationMenuWidth(w int32) NavigationMenuItemOption {
	return func(it *navItem) {
		if w > 0 {
			it.panelW = w
		}
	}
}

// NewNavigationMenu 创建导航菜单。根容器宽高按触发区自撑(Fixed), 放在任意流式布局里即可;
// 需要居中时在外面套一行容器加弹性项。
func NewNavigationMenu(hParent int) *NavigationMenu {
	root := widget.NewLayoutEle(0, 0, 10, navH, hParent)
	if root == nil {
		return nil
	}
	root.EnableHorizon(true)
	root.EnableLayout(true)
	root.EnableBkTransparent(true)
	root.EnableAutoWrap(false)
	root.SetSpace(navGap)
	root.SetAlignV(xcc.Layout_Align_Center)
	root.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 10)
	root.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, navH)

	n := &NavigationMenu{
		hWnd:          xc.XWidget_GetHWINDOW(root.Handle),
		root:          root,
		open:          -1,
		isOpenDelayed: true,
	}
	xui.AttachWindow(root.Handle)
	// 弹层窗口是懒创建、只隐藏不销毁的: 主窗口销毁时必须一起销毁, 否则消息循环
	// 永远等不到"所有窗口销毁", 关窗后进程不退(同 Tooltip/Menubar, 见 DESIGN 77)。
	root.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		n.cleanup()
		return 0
	})
	if win := window.NewByHandle(n.hWnd); win != nil {
		win.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
			switch message {
			case wapi.WM_DESTROY:
				n.cleanup()
			case wapi.WM_MOVE:
				// 拖动窗口时视口是独立顶层窗口, 不跟随宿主移动; 收起比留在原地更合理。
				n.closeAll(false)
			}
			return 0
		})
		// 独立弹层不会因为失去焦点而收起(非激活窗口), 由宿主窗口兜底:
		// 切到别的程序 / 缩放窗口时导航菜单不该继续挂在屏幕上。
		win.AddEvent_KillFocus(func(hWindow int, pbHandled *bool) int {
			n.closeAll(false)
			return 0
		})
		win.AddEvent_Size(func(hWindow int, nFlags uint, pPt *xc.SIZE, pbHandled *bool) int {
			n.closeAll(false)
			return 0
		})
		// 鼠标悬停打开时焦点还在页面里(可能是输入框), ESC 由窗口兜底关闭。
		win.AddEvent_KeyDown(func(hWindow int, wParam, lParam uintptr, pbHandled *bool) int {
			if wParam == navVKEsc && n.open >= 0 {
				n.suppressHoverOnOpenItem()
				n.closeAll(false)
				*pbHandled = true
			}
			return 0
		})
	}
	xui.OnThemeChangeFor(root.Handle, func(*xui.Theme) {
		n.relayout()
		// 换主题可能改变字体族 → 触发区宽/根宽变化, 挂着的那层要跟着挪回列表左下方。
		if n.open >= 0 && n.panel != nil && n.panel.IsOpen() {
			x, y := n.panelPos()
			n.panel.Relocate(n.root.Handle, x, y)
		}
	})
	return n
}

// Root 返回根容器句柄。
func (n *NavigationMenu) Root() int {
	if n == nil || n.root == nil {
		return 0
	}
	return n.root.Handle
}

// AddMenu 追加一个带内容页的菜单。build 在首次打开时执行(不需要预先构建),
// c.Link 添加的链接参与键盘导航; 也可以用 c.Handle 放任意控件。
func (n *NavigationMenu) AddMenu(title string, build func(*NavigationMenuContent),
	opts ...NavigationMenuItemOption) *NavigationMenuItem {
	if n == nil || n.root == nil {
		return nil
	}
	it := &navItem{
		kind:   navKindMenu,
		title:  title,
		value:  title,
		panelW: navDefaultW,
		build:  build,
	}
	for _, o := range opts {
		o(it)
	}
	if it.panelW < 120 {
		it.panelW = 120
	}
	return n.addItem(it)
}

// AddLink 追加一个不带弹层的普通链接(顶级项)。
func (n *NavigationMenu) AddLink(title string, onClick func(), opts ...NavigationMenuItemOption) *NavigationMenuItem {
	if n == nil || n.root == nil {
		return nil
	}
	it := &navItem{kind: navKindLink, title: title, value: title, onClick: onClick}
	for _, o := range opts {
		o(it)
	}
	return n.addItem(it)
}

func (n *NavigationMenu) addItem(it *navItem) *NavigationMenuItem {
	it.nav = n
	it.chevKey = xui.NextPresenceKey()
	f := navFont()
	it.fontH = f
	it.titleW = xui.TextWidth(it.title, f)
	it.width = n.widthOf(it)

	// 原生 Button 只借行为(焦点/点击/键盘), 外观全部自绘: 自定义 paint 注册在
	// Button.onPaint 之后, 先执行并置 pbHandled, 原生绘制不再跑(同 Accordion)。
	btn := NewButton(0, 0, it.width, navH, "", n.root.Handle,
		WithVariant(ButtonGhost), WithButtonHasPopup())
	if btn == nil {
		return nil
	}
	it.btn = btn
	btn.SetBorderSize(0, 0, 0, 0)
	btn.EnableDrawBorder(false)
	btn.SetTextColor(0)
	btn.SetWidth(it.width)
	btn.SetHeight(navH)
	btn.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, it.width)
	btn.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, navH)
	btn.AddEvent_Paint(n.paintItem(it))

	idx := len(n.items)
	n.items = append(n.items, it)

	btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		*pbHandled = true
		if it.disabled {
			return 0
		}
		if it.kind == navKindLink {
			n.closeAll(false)
			if it.onClick != nil {
				it.onClick()
			}
			return 0
		}
		if n.open == idx {
			it.suppressOpen = true
			n.closeAll(false)
			return 0
		}
		n.openItem(idx, false)
		return 0
	})
	btn.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if it.disabled {
			return 0
		}
		// 只有菜单触发区参与"悬停打开/切换"; 顶级普通链接不取消关闭计时器 ——
		// 从菜单移到普通链接上时, radix 已经开始 150ms 收起, 不该被链接续命。
		if it.kind != navKindMenu {
			return 0
		}
		n.cancelClose()
		if n.open == idx || it.suppressOpen {
			return 0
		}
		if n.open >= 0 {
			n.openItem(idx, false)
			return 0
		}
		if n.isOpenDelayed {
			n.scheduleOpen(idx)
		} else {
			n.openItem(idx, false)
		}
		return 0
	})
	btn.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		it.suppressOpen = false
		return 0
	})
	btn.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		it.suppressOpen = false
		n.stopOpenTimer()
		if n.open >= 0 {
			n.scheduleClose()
		}
		return 0
	})
	btn.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		*pbHandled = n.onItemKey(idx, wParam)
		return 0
	})

	n.relayout()
	return &NavigationMenuItem{it: it}
}

// Button 返回该项的触发按钮(可挂 Tooltip / 改布局项)。
func (i *NavigationMenuItem) Button() *Button {
	if i == nil || i.it == nil {
		return nil
	}
	return i.it.btn
}

// Value 返回该项的值。
func (i *NavigationMenuItem) Value() string {
	if i == nil || i.it == nil {
		return ""
	}
	return i.it.value
}

// SetDisabled 设置/清除禁用态; 禁用中的菜单会被收起。
func (i *NavigationMenuItem) SetDisabled(v bool) *NavigationMenuItem {
	if i == nil || i.it == nil {
		return i
	}
	it := i.it
	if it.disabled == v {
		return i
	}
	it.disabled = v
	if it.btn != nil && xc.XC_IsHXCGUI(it.btn.Handle, xcc.XC_ELE) {
		it.btn.Enable(!v)
		it.btn.Redraw(false)
	}
	if v && it.nav != nil && it.nav.open >= 0 && it.nav.items[it.nav.open] == it {
		it.nav.closeAll(false)
	}
	return i
}

// Value 返回当前打开菜单的值("" = 全关)。
func (n *NavigationMenu) Value() string {
	if n == nil || n.open < 0 || n.open >= len(n.items) {
		return ""
	}
	return n.items[n.open].value
}

// SetValue 按值打开/关闭菜单("" = 关闭)。值对应 WithNavigationMenuValue 或标题。
func (n *NavigationMenu) SetValue(value string) {
	if n == nil {
		return
	}
	if value == "" {
		n.closeAll(false)
		return
	}
	for i, it := range n.items {
		if it.kind == navKindMenu && it.value == value {
			if n.open == i {
				return
			}
			n.openItem(i, false)
			return
		}
	}
	n.closeAll(false)
}

// IsOpen 报告是否有菜单打开。
func (n *NavigationMenu) IsOpen() bool {
	return n != nil && n.open >= 0
}

// Close 收起菜单(不改焦点)。
func (n *NavigationMenu) Close() {
	if n != nil {
		n.closeAll(false)
	}
}

// OnValueChange 注册值变化回调(打开菜单 = 菜单值, 关闭 = "").
func (n *NavigationMenu) OnValueChange(fn func(value string)) {
	if n != nil && fn != nil {
		n.changes = append(n.changes, fn)
	}
}

// Handle 返回内容布局容器的句柄(放自定义控件时用)。
func (c *NavigationMenuContent) Handle() int {
	if c == nil || c.ele == nil {
		return 0
	}
	return c.ele.Handle
}

// Link 在内容页里追加一行链接(标题 + 一行说明, 说明传空串时只显示标题)。
// 链接参与键盘高亮/Enter 激活; 点击后先执行 fn 再关闭菜单。
func (c *NavigationMenuContent) Link(title, description string, fn func()) *NavigationMenuLink {
	if c == nil || c.ele == nil {
		return nil
	}
	return c.LinkIn(c.ele.Handle, title, description, fn)
}

// LinkIn 把链接行建在指定父容器里(做多列网格时把链接放进 xui.HStack 的列,
// 再 xui.ItemWeight 分配宽度)。键盘顺序按创建顺序(行优先)。
func (c *NavigationMenuContent) LinkIn(hParent int, title, description string, fn func()) *NavigationMenuLink {
	if c == nil || hParent == 0 {
		return nil
	}
	h := navLinkH
	if description == "" {
		h = navLinkOnlyH
	}
	ele := widget.NewElement(0, 0, 0, h, hParent)
	if ele == nil {
		return nil
	}
	l := &NavigationMenuLink{Element: ele, nav: c.nav, title: title, desc: description, click: fn}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	ele.SetHeight(h)
	ele.AddEvent_Paint(l.onPaint)
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		*pbHandled = true
		l.activate()
		return 0
	})
	ele.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		c.nav.cancelClose()
		c.nav.setHoverLink(l)
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		c.nav.scheduleClose()
		c.nav.setHoverLink(nil)
		return 0
	})
	xui.AttachWindow(ele.Handle)
	c.item.links = append(c.item.links, l)
	return l
}

func (l *NavigationMenuLink) activate() {
	if l == nil {
		return
	}
	if l.click != nil {
		l.click()
	}
	if l.nav != nil {
		l.nav.closeAll(false)
	}
}

// ===== 展开 / 收起 =====

// openItem 打开第 idx 个菜单。keyboard=true 时打开后进入内容键盘导航。
func (n *NavigationMenu) openItem(idx int, keyboard bool) {
	if n == nil || idx < 0 || idx >= len(n.items) {
		return
	}
	it := n.items[idx]
	if it.kind != navKindMenu || it.disabled {
		return
	}
	if n.open == idx {
		if keyboard {
			n.enterContent()
		}
		return
	}
	if !n.ensurePanel() || !n.ensureContent(it) {
		return
	}
	n.stopOpenTimer()
	n.cancelClose()

	prev := n.open
	var prevIt *navItem
	if prev >= 0 && prev < len(n.items) {
		prevIt = n.items[prev]
		prevIt.open = false
		if prevIt.btn != nil && xc.XC_IsHXCGUI(prevIt.btn.Handle, xcc.XC_ELE) {
			prevIt.btn.SetExpanded(false)
		}
		n.animateChevron(prevIt, false)
	}
	if prevIt != nil {
		n.setHoverLink(nil)
		n.setKBLink(nil)
	}
	n.showOnly(it)
	panelH := n.panelHeightFor(it)
	it.open = true
	if n.panel.IsOpen() {
		n.panel.resize(it.panelW, panelH)
		x, y := n.panelPos()
		n.panel.Relocate(n.root.Handle, x, y)
		dir := int32(0)
		if prev >= 0 {
			if idx > prev {
				dir = 1
			} else {
				dir = -1
			}
		}
		n.animateContentIn(it, dir)
	} else {
		x, y := n.panelPos()
		n.panel.ShowAt(n.root.Handle, x, y)
	}
	n.open = idx
	n.isOpenDelayed = false
	if n.skipTimer != nil {
		n.skipTimer.Stop()
		n.skipTimer = nil
	}
	n.everInside = n.pointerOver()
	n.startWatch()
	it.btn.SetExpanded(true)
	n.animateChevron(it, true)
	n.emit()
	if keyboard {
		n.enterContent()
	}
}

// closeAll 收起当前菜单。focus=true 时把焦点还给触发区(ESC 语义)。
func (n *NavigationMenu) closeAll(focus bool) {
	if n == nil || n.open < 0 || n.open >= len(n.items) {
		return
	}
	idx := n.open
	it := n.items[idx]
	n.stopOpenTimer()
	n.cancelClose()
	n.stopWatch()
	n.setHoverLink(nil)
	n.everInside = false
	if n.panel != nil && n.panel.alive() {
		xui.DropTransitions(n.panel.win.Handle)
		// 内容切换动画可能被中途打断: 手动 SetRect 的横移不会自己复位,
		// 收起前重排一次内容页, 让下次打开从布局位置开始。
		if it.entry != nil && it.entry.ele != nil && n.content != nil &&
			xc.XC_IsHXCGUI(it.entry.ele.Handle, xcc.XC_ELE) {
			n.content.AdjustLayout()
		}
		n.panel.Hide()
	}
	if it.entry != nil && it.entry.ele != nil && xc.XC_IsHXCGUI(it.entry.ele.Handle, xcc.XC_ELE) {
		xui.SetSubtreeAlpha(1, it.entry.ele.Handle)
	}
	it.open = false
	if it.btn != nil && xc.XC_IsHXCGUI(it.btn.Handle, xcc.XC_ELE) {
		it.btn.SetExpanded(false)
	}
	n.animateChevron(it, false)
	n.setKBLink(nil)
	n.open = -1
	n.startSkipTimer()
	n.emit()
	if focus {
		n.focusLater(idx)
	}
}

// panelHeightFor 让视口按第 it 个菜单的内容高度收放。打开中只能就地重排
// (不能再走离屏测量, 那会把窗口隐藏); 未打开时用弹层的离屏测量。
func (n *NavigationMenu) panelHeightFor(it *navItem) int32 {
	if n.panel.IsOpen() {
		n.panel.resize(it.panelW, n.panel.ch)
		h := n.content.GetHeight()
		if h <= 0 {
			h = 1
		}
		return h + navContentPd*2
	}
	n.panel.resize(it.panelW, navContentPd*2+1)
	h := n.panel.MeasureContent(navContentPd)
	if h <= 0 {
		h = 1
	}
	return h + navContentPd*2
}

// animateChevron 用 Presence 驱动 chevron 的 0..180° 旋转(稳定键, 中断可反向)。
func (n *NavigationMenu) animateChevron(it *navItem, open bool) {
	if it == nil || it.btn == nil || !xc.XC_IsHXCGUI(it.btn.Handle, xcc.XC_ELE) {
		return
	}
	xui.Presence(it.btn.Handle, it.chevKey, open, navChevronDur, func(v float64) {
		it.chevron = v * math.Pi
	}, nil)
}

// animateContentIn 让切换后的内容从切换方向滑入并淡入。首次打开不播 ——
// 视口自己的淡入已经在跑, 两套 alpha 会互相覆盖(每帧后写的赢)。
//
// 位移必须等**布局趟落定**再开始: 切页前刚调过 panel.resize, 窗口收到 WM_SIZE
// 后还会再排一次内容, 会把手动设的横移原地复位(实测: 内容直接出现在终点,
// 只有 alpha 在动)。先压透明藏住, AfterFunc(0) 时再取基准矩形并起过渡;
// 万一之后仍有布局趟, 每帧 apply 会重新写回偏移。
func (n *NavigationMenu) animateContentIn(it *navItem, dir int32) {
	if dir == 0 || it == nil || it.entry == nil || it.entry.ele == nil || n.panel == nil {
		return
	}
	c := it.entry.ele.Handle
	if !xc.XC_IsHXCGUI(c, xcc.XC_ELE) {
		return
	}
	xui.SetSubtreeAlpha(0, c)
	app.AfterFunc(0, func() {
		if !xc.XC_IsHXCGUI(c, xcc.XC_ELE) || n.panel == nil || !n.panel.alive() {
			return
		}
		if n.open < 0 || n.open >= len(n.items) || n.items[n.open] != it {
			// 这一页已经被关掉/切走: 恢复不透明, 免得下次打开是透明的
			xui.SetSubtreeAlpha(1, c)
			return
		}
		var base xc.RECT
		xc.XEle_GetRect(c, &base)
		dx := float64(dir) * float64(navSlide)
		rc := base
		rc.Left += int32(dx)
		rc.Right += int32(dx)
		xc.XEle_SetRect(c, &rc, false, xcc.AdjustLayout_No)
		xui.Presence(n.panel.win.Handle, xui.NextPresenceKey(), true, navContentDur, func(v float64) {
			if !xc.XC_IsHXCGUI(c, xcc.XC_ELE) {
				return
			}
			xui.SetSubtreeAlpha(v, c)
			off := int32(math.Round(dx * (1 - v)))
			r := base
			r.Left += off
			r.Right += off
			xc.XEle_SetRect(c, &r, false, xcc.AdjustLayout_No)
		}, func(v float64) {
			if !xc.XC_IsHXCGUI(c, xcc.XC_ELE) {
				return
			}
			xui.SetSubtreeAlpha(1, c)
			xc.XEle_SetRect(c, &base, false, xcc.AdjustLayout_No)
		})
	})
}

// showOnly 只显示第 it 个菜单的内容页(其余隐藏, 布局会跳过隐藏项)。
func (n *NavigationMenu) showOnly(it *navItem) {
	for _, other := range n.items {
		if other == it || other.entry == nil || other.entry.ele == nil {
			continue
		}
		if other.entry.ele.IsShow() {
			other.entry.ele.Show(false)
		}
	}
	if it.entry != nil && it.entry.ele != nil && !it.entry.ele.IsShow() {
		it.entry.ele.Show(true)
	}
}

// ensurePanel 懒创建共享视口(非激活: 不抢焦点, 不因失焦关闭 —— 收起完全由
// 悬停计时器决定, 与 HoverCard 同一套机制, 见 DESIGN 116)。
func (n *NavigationMenu) ensurePanel() bool {
	if n.panel != nil && n.panel.alive() {
		return true
	}
	p := newPopoverPanelEx(navDefaultW, navContentPd*2+1, n.root.Handle, true, false)
	if p == nil {
		return false
	}
	t := xui.Current()
	p.paintRadius = t.RadiusMd()
	p.paintBorderSolid = true
	content := p.LayoutContent(navContentPd, 0)
	if content == nil {
		p.Destroy()
		return false
	}
	n.panel, n.content = p, content
	// 续期/收起必须挂**窗口级**事件: 内容里的链接行是子元素, 指针在链接上时
	// body 收不到 MouseMove(否则从触发区移进内容会被 150ms 计时器关掉, 实测)。
	p.win.AddEvent_MouseMove(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		n.cancelClose()
		return 0
	})
	p.win.AddEvent_MouseLeave(func(hWindow int, pbHandled *bool) int {
		n.scheduleClose()
		return 0
	})
	body := p.body
	body.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		n.cancelClose()
		return 0
	})
	body.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		n.scheduleClose()
		return 0
	})
	return true
}

// ensureContent 首次打开时构建该菜单的内容页。
func (n *NavigationMenu) ensureContent(it *navItem) bool {
	if it.entry != nil {
		return true
	}
	if !n.ensurePanel() {
		return false
	}
	ele := widget.NewLayoutEle(0, 0, 10, 10, n.content.Handle)
	if ele == nil {
		return false
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetAlignV(xcc.Layout_Align_Top)
	ele.SetSpace(8)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ele.Show(false)
	ctx := &NavigationMenuContent{nav: n, item: it, ele: ele}
	it.entry = ctx
	if it.build != nil {
		it.build(ctx)
	}
	return true
}

// panelPos 返回视口卡片左上角的屏幕逻辑坐标: 贴列表左缘, 列表底边 + 6。
func (n *NavigationMenu) panelPos() (int32, int32) {
	var rc, wr xc.RECT
	xc.XEle_GetWndClientRect(n.root.Handle, &rc)
	xc.XWnd_GetRect(n.hWnd, &wr)
	return wr.Left + rc.Left, wr.Top + rc.Bottom + navPanelGap
}

// ===== 悬停计时器 =====

func (n *NavigationMenu) scheduleOpen(idx int) {
	if n.openTimer != nil {
		return
	}
	n.openTimer = app.AfterFunc(navOpenDelay, func() {
		n.openTimer = nil
		if n.open >= 0 || !n.alive() {
			return
		}
		n.openItem(idx, false)
	})
}

func (n *NavigationMenu) stopOpenTimer() {
	if n.openTimer != nil {
		n.openTimer.Stop()
		n.openTimer = nil
	}
}

func (n *NavigationMenu) scheduleClose() {
	if n == nil || n.open < 0 {
		return
	}
	n.stopCloseTimer()
	n.closeTimer = app.AfterFunc(navCloseDelay, func() {
		n.closeTimer = nil
		n.closeIfOutside()
	})
}

// closeIfOutside 计时到点时按**光标位置**决定关不关: 指针还在视口或菜单触发区上
// 就每 120ms 再看一次。不能只靠 MouseMove/MouseLeave —— 指针落在内容里的子元素
// 上时, 弹层 body 收不到 MouseMove(实测: 从触发区移进链接行会被 150ms 计时器
// 直接关掉); 而"从子元素移到留白"又不会再有 MouseLeave。轮询到真正离开为止,
// 与 HoverCard 的关闭判定同一套语义。
func (n *NavigationMenu) closeIfOutside() {
	if n == nil || n.open < 0 || !n.alive() {
		return
	}
	if n.pointerOver() {
		n.stopCloseTimer()
		n.closeTimer = app.AfterFunc(120*time.Millisecond, func() {
			n.closeTimer = nil
			n.closeIfOutside()
		})
		return
	}
	n.closeAll(false)
}

// pointerOver 报告光标是否还在共享视口(含阴影带)或任一**菜单触发区**上。
// 顶级普通链接不算: radix 里从内容移到普通链接会正常收起。
func (n *NavigationMenu) pointerOver() bool {
	if n.panel != nil && n.panel.alive() && cursorInPanel(n.panel) {
		return true
	}
	var pt wapi.POINT
	if !wapi.GetCursorPos(&pt) {
		return false
	}
	dpi := xc.XWnd_GetDPI(n.hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96
	var wr xc.RECT
	xc.XWnd_GetRect(n.hWnd, &wr)
	cx := int32(math.Round(float64(pt.X)/sc)) - wr.Left
	cy := int32(math.Round(float64(pt.Y)/sc)) - wr.Top
	for _, it := range n.items {
		if it.kind != navKindMenu || it.btn == nil || !xc.XC_IsHXCGUI(it.btn.Handle, xcc.XC_ELE) {
			continue
		}
		var rc xc.RECT
		xc.XEle_GetWndClientRect(it.btn.Handle, &rc)
		if cx >= rc.Left && cx < rc.Right && cy >= rc.Top && cy < rc.Bottom {
			return true
		}
	}
	return false
}

// startWatch 在菜单打开期间每 30ms 看一次光标:
//   - 视口是**非激活**窗口, 里面的普通元素拿不到原生 hover 状态(实测 Stay 位
//     恒为 Leave), 也没有 MouseMove 事件 —— 链接的悬停高亮只能自己判定;
//   - 同时它也是收起的兜底: 指针一旦离开视口/触发区就启动 150ms 计时,
//     不依赖子元素的 MouseLeave。
//
// GetStayEle 拿到的是光标下的元素(窗口未激活也有效, 滚动条显隐用的是同一条)。
func (n *NavigationMenu) startWatch() {
	n.stopWatch()
	n.watchTimer = app.AfterFunc(30*time.Millisecond, n.watch)
}

func (n *NavigationMenu) stopWatch() {
	if n != nil && n.watchTimer != nil {
		n.watchTimer.Stop()
		n.watchTimer = nil
	}
}

func (n *NavigationMenu) watch() {
	n.watchTimer = nil
	if n == nil || n.open < 0 || !n.alive() {
		return
	}
	n.watchTick()
	if n.open >= 0 && n.alive() {
		n.watchTimer = app.AfterFunc(30*time.Millisecond, n.watch)
	}
}

func (n *NavigationMenu) watchTick() {
	if n.panel == nil || !n.panel.alive() {
		return
	}
	if cursorInPanel(n.panel) {
		n.setHoverLink(n.linkFromStay(xc.XWnd_GetStayEle(n.panel.win.Handle)))
	} else {
		n.setHoverLink(nil)
	}
	if n.pointerOver() {
		n.everInside = true
		n.cancelClose()
	} else if n.everInside && n.closeTimer == nil {
		n.scheduleClose()
	}
}

// linkFromStay 把 GetStayEle 的句柄归到当前内容页的链接行(向上找父级)。
func (n *NavigationMenu) linkFromStay(h int) *NavigationMenuLink {
	if h == 0 {
		return nil
	}
	it := n.openItemValue()
	if it == nil {
		return nil
	}
	for x := h; x != 0; x = xc.XWidget_GetParentEle(x) {
		for _, l := range it.links {
			if l.Handle == x {
				return l
			}
		}
		if n.content != nil && x == n.content.Handle {
			break
		}
	}
	return nil
}

// syncSwitchFocus 在"内容键盘模式"期间关掉**全部**顶级触发区的原生方向键焦点切换。
//
// DLL 的方向键换焦点是默认处理, 不看事件回调的 pbHandled: 内容高亮和触发区
// 焦点会被同一次按键各推一步, 屏幕上就是两个焦点环一起切换(实测)。进入内容
// 时禁用、退出(ESC/点击/切换菜单)时恢复; Tab 遍历不受影响(恢复发生在同一次
// 按键的回调里, 原生 Tab 处理在其后)。只关"打开那一项"不够 —— 原生遍历从
// 焦点所在元素出发, 焦点可能停在别的触发区上。
func (n *NavigationMenu) syncSwitchFocus() {
	for _, it := range n.items {
		if it.btn == nil || !xc.XC_IsHXCGUI(it.btn.Handle, xcc.XC_ELE) {
			continue
		}
		it.btn.EnableSwitchFocus(n.kbLink == nil)
	}
}

// setHoverLink 更新指针悬停的链接并重绘(纯 Element 不会自动因 hover 重绘)。
func (n *NavigationMenu) setHoverLink(l *NavigationMenuLink) {
	if n.hoverLink == l {
		return
	}
	old := n.hoverLink
	n.hoverLink = l
	if old != nil && xc.XC_IsHXCGUI(old.Handle, xcc.XC_ELE) {
		old.Redraw(false)
	}
	if l != nil && xc.XC_IsHXCGUI(l.Handle, xcc.XC_ELE) {
		l.Redraw(false)
	}
	// 指针移到某条链接上时让位给鼠标(只在新进入一条链接时清键盘高亮,
	// 否则指针停在原地时方向键永远选不中)。
	if l != nil && n.kbLink != nil {
		n.setKBLink(nil)
	}
}

func (n *NavigationMenu) cancelClose() {
	n.stopCloseTimer()
}

func (n *NavigationMenu) stopCloseTimer() {
	if n.closeTimer != nil {
		n.closeTimer.Stop()
		n.closeTimer = nil
	}
}

// startSkipTimer 对齐 radix 的 skipDelayDuration: 关掉后 300ms 内仍按
// isOpenDelayed=false 处理(悬停其它触发区立即打开), 300ms 之后才恢复
// 200ms 的打开延迟。方向别写反 —— 刚关就恢复延迟, 会让用户在不同菜单之间
// 移动时每次都要再等 200ms。
func (n *NavigationMenu) startSkipTimer() {
	if n.skipTimer != nil {
		n.skipTimer.Stop()
	}
	n.isOpenDelayed = false
	n.skipTimer = app.AfterFunc(navSkipDelay, func() {
		n.skipTimer = nil
		n.isOpenDelayed = true
	})
}

// ===== 键盘 =====

func (n *NavigationMenu) onItemKey(idx int, wParam uintptr) bool {
	if n == nil || idx < 0 || idx >= len(n.items) {
		return false
	}
	it := n.items[idx]
	if n.kbLink != nil {
		return n.onContentKey(wParam)
	}
	switch wParam {
	case navVKLeft:
		n.focusNeighbor(idx, -1)
		return true
	case navVKRight:
		n.focusNeighbor(idx, 1)
		return true
	case navVKDown:
		if it.kind != navKindMenu || it.disabled {
			return false
		}
		if n.open == idx {
			n.enterContent()
		} else {
			n.openItem(idx, true)
		}
		return true
	case navVKEnter, navVKSpace:
		if it.disabled {
			return true
		}
		if it.kind == navKindLink {
			if it.onClick != nil {
				it.onClick()
			}
			n.closeAll(false)
			return true
		}
		if n.open == idx {
			n.closeAll(true)
		} else {
			n.openItem(idx, true)
		}
		return true
	case navVKEsc:
		if n.open >= 0 {
			n.suppressHoverOnOpenItem()
			n.closeAll(true)
			return true
		}
	case navVKTab:
		if n.open >= 0 {
			n.suppressHoverOnOpenItem()
			n.closeAll(false)
		}
	}
	return false
}

func (n *NavigationMenu) onContentKey(wParam uintptr) bool {
	switch wParam {
	case navVKUp, navVKLeft:
		n.moveKB(-1)
		return true
	case navVKDown, navVKRight:
		n.moveKB(1)
		return true
	case navVKHome:
		if it := n.openItemValue(); it != nil && len(it.links) > 0 {
			n.setKBLink(it.links[0])
		}
		return true
	case navVKEnd:
		if it := n.openItemValue(); it != nil && len(it.links) > 0 {
			n.setKBLink(it.links[len(it.links)-1])
		}
		return true
	case navVKEnter, navVKSpace:
		if n.kbLink != nil {
			n.kbLink.activate()
		}
		return true
	case navVKEsc:
		n.closeAll(true)
		return true
	case navVKTab:
		n.closeAll(false)
		return true
	}
	return false
}

func (n *NavigationMenu) openItemValue() *navItem {
	if n.open < 0 || n.open >= len(n.items) {
		return nil
	}
	return n.items[n.open]
}

// enterContent 把键盘高亮放进内容页的第一条链接。
func (n *NavigationMenu) enterContent() {
	it := n.openItemValue()
	if it == nil || len(it.links) == 0 {
		return
	}
	n.setKBLink(it.links[0])
}

func (n *NavigationMenu) moveKB(dir int) {
	it := n.openItemValue()
	if it == nil || len(it.links) == 0 {
		return
	}
	cur := -1
	for i, l := range it.links {
		if l == n.kbLink {
			cur = i
			break
		}
	}
	next := 0
	if cur >= 0 {
		next = ((cur+dir)%len(it.links) + len(it.links)) % len(it.links)
	} else if dir < 0 {
		next = len(it.links) - 1
	}
	n.setKBLink(it.links[next])
}

func (n *NavigationMenu) setKBLink(l *NavigationMenuLink) {
	if n.kbLink == l {
		return
	}
	old := n.kbLink
	n.kbLink = l
	if old != nil && xc.XC_IsHXCGUI(old.Handle, xcc.XC_ELE) {
		old.Redraw(false)
	}
	if l != nil && xc.XC_IsHXCGUI(l.Handle, xcc.XC_ELE) {
		l.Redraw(false)
	}
	n.syncSwitchFocus()
}

// suppressHoverOnOpenItem 让"指针还停在触发区上"的那一项不要在关闭后
// 因为一次轻微移动就重新打开(radix 的 wasEscapeCloseRef 语义); 指针离开
// 再进入(MouseStay)时清除。
func (n *NavigationMenu) suppressHoverOnOpenItem() {
	if it := n.openItemValue(); it != nil {
		it.suppressOpen = true
	}
}

// focusNeighbor 在顶级项之间移动焦点(←→); 用 NoteKeyboard 让焦点环出现。
//
// 搬焦点必须推迟到消息循环(焦点会"晚到"): 在 KeyDown 回调里同步
// XWnd_SetFocusEle, 同一次按键会被重新派发给刚拿到焦点的按钮, 于是 ←→
// 一次移动两项(实测)。推迟后按键已经处理完, 新按钮只收到下一次按键。
func (n *NavigationMenu) focusNeighbor(idx, dir int) {
	if len(n.items) == 0 {
		return
	}
	next := ((idx+dir)%len(n.items) + len(n.items)) % len(n.items)
	xui.NoteKeyboard()
	n.focusLater(next)
}

// focusLater 把焦点还给第 idx 个触发区。弹层隐藏后主窗口可能还没拿回激活,
// 立刻 SetFocusEle 会被激活流程覆盖(同 Menubar)。
func (n *NavigationMenu) focusLater(idx int) {
	if idx < 0 || idx >= len(n.items) {
		return
	}
	h := n.items[idx].btn.Handle
	app.AfterFunc(0, func() {
		if !xc.XC_IsHXCGUI(n.hWnd, xcc.XC_WINDOW) || !xc.XC_IsHXCGUI(h, xcc.XC_ELE) {
			return
		}
		xc.XWnd_SetFocusEle(n.hWnd, h)
	})
}

// ===== 尺寸 / 主题 =====

// widthOf 量出触发区的宽度(文字 + 左右内边距 + chevron)。
func (n *NavigationMenu) widthOf(it *navItem) int32 {
	w := it.titleW
	if it.kind == navKindMenu {
		w += navPadX*2 + navIconGap + navIcon
	} else {
		w += navLinkPadX * 2
	}
	return w
}

// relayout 按当前主题字体重量触发区宽度并同步根容器宽(换主题时字体族变化)。
func (n *NavigationMenu) relayout() {
	if n == nil || n.root == nil || !xc.XC_IsHXCGUI(n.root.Handle, xcc.XC_ELE) {
		return
	}
	f := navFont()
	total := int32(0)
	for i, it := range n.items {
		if it.fontH != f || it.titleW <= 0 {
			it.fontH = f
			it.titleW = xui.TextWidth(it.title, f)
			it.width = n.widthOf(it)
		}
		if it.btn != nil && xc.XC_IsHXCGUI(it.btn.Handle, xcc.XC_ELE) {
			it.btn.SetWidth(it.width)
			it.btn.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, it.width)
		}
		if i > 0 {
			total += navGap
		}
		total += it.width
	}
	if total < 1 {
		total = 1
	}
	n.root.SetWidth(total)
	n.root.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, total)
	n.root.AdjustLayout()
	n.root.Redraw(false)
	if p := xc.XWidget_GetParentEle(n.root.Handle); p != 0 {
		xc.XEle_AdjustLayout(p)
	}
}

// ===== 绘制 =====

// paintItem 绘制顶级项: hover/open 底色、focus-visible 环、14px 文字与旋转 chevron。
func (n *NavigationMenu) paintItem(it *navItem) func(hEle int, hDraw int, pbHandled *bool) int {
	return func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		t := xui.Current()
		st := xui.StateOf(hEle)
		host := t.Card
		radius := t.RadiusMd()
		// 透明元素重绘不擦底: 先铺宿主底色(同 Accordion/Toggle)
		p.FillRound(&p.Rc, host, 0)

		disabled := st.Disabled || it.disabled
		hv := xui.Hover(hEle, st.Hover && !disabled)
		var bg xui.Color
		if it.open {
			// data-[state=open]:bg-accent/50, 悬停时回到全亮 bg-accent
			bg = t.Accent.ScaleAlpha(0.5 + 0.5*hv)
		} else if hv > 0 {
			bg = t.Accent.ScaleAlpha(hv)
		}
		if st.Focused && !disabled && xui.FocusedByKeyboard() {
			p.FocusFrame(t.Ring, host, radius)
			if !bg.IsNone() {
				const inset = int32(3)
				inner := xc.RECT{
					Left:   p.Rc.Left + inset,
					Top:    p.Rc.Top + inset,
					Right:  p.Rc.Right - inset,
					Bottom: p.Rc.Bottom - inset,
				}
				if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
					p.FillRound(&inner, bg, radius-inset)
				}
			}
		} else if !bg.IsNone() {
			p.FillRound(&p.Rc, bg, radius)
		}

		fore := t.Foreground
		if a := hv; it.open || a > 0 {
			if it.open && a < 1 {
				a = 1 // 展开态文字就是 accent-foreground
			}
			fore = xui.Fade(fore, t.AccentForeground, a)
		}
		if disabled {
			fore = fore.Mix(host, 0.5)
		}

		f := navFont()
		if f != 0 && f != it.fontH {
			it.fontH = f
			it.titleW = xui.TextWidth(it.title, f)
		}
		if f != 0 {
			p.Draw.SetFont(f)
		}
		iconSpace := int32(0)
		if it.kind == navKindMenu {
			iconSpace = navIconGap + navIcon
		}
		content := it.titleW + iconSpace
		x := p.Rc.Left + (p.W()-content)/2
		textRc := xc.RECT{
			Left:   x,
			Top:    p.Rc.Top - 1,
			Right:  x + it.titleW,
			Bottom: p.Rc.Bottom - 1,
		}
		p.Text(it.title, &textRc, fore,
			xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		if it.kind == navKindMenu {
			iconRc := xc.RECT{
				Left:   x + it.titleW + navIconGap,
				Top:    (p.Rc.Top+p.Rc.Bottom-navIcon)/2 + 1,
				Right:  x + it.titleW + navIconGap + navIcon,
				Bottom: (p.Rc.Top+p.Rc.Bottom+navIcon)/2 + 1,
			}
			p.IconRotated(icons.IconChevronDown, &iconRc, fore, 2, it.chevron)
		}
		return 0
	}
}

// onPaint 绘制内容链接行: 悬停/键盘高亮铺 accent, 键盘高亮另画 focus-visible 环;
// 标题 14px, 说明一行 14px muted(超宽省略号)。
func (l *NavigationMenuLink) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)
	radius := t.RadiusSm()
	// 弹层底色是 Popover, 不是 Card
	p.FillRound(&p.Rc, t.Popover, 0)

	kb := l.nav != nil && l.nav.kbLink == l
	hovered := st.Hover || (l.nav != nil && l.nav.hoverLink == l)
	hv := xui.Hover(hEle, hovered)
	if kb {
		p.FocusFrame(t.Ring, t.Accent, radius)
	} else if hv > 0 {
		p.FillRound(&p.Rc, t.Accent.ScaleAlpha(hv), radius)
	}

	f := navFont()
	if f != 0 {
		p.Draw.SetFont(f)
	}
	titleRc := xc.RECT{
		Left:   p.Rc.Left + navLinkPd,
		Top:    p.Rc.Top + navLinkPd - 1,
		Right:  p.Rc.Right - navLinkPd,
		Bottom: p.Rc.Top + navLinkPd + navLinkTitle - 1,
	}
	p.Text(l.title, &titleRc, t.Foreground,
		xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap|
			xcc.TextTrimming_EllipsisCharacter)
	if l.desc != "" {
		descRc := xc.RECT{
			Left:   p.Rc.Left + navLinkPd,
			Top:    titleRc.Bottom + navLinkGap - 1,
			Right:  p.Rc.Right - navLinkPd,
			Bottom: titleRc.Bottom + navLinkGap + navLinkTitle - 1,
		}
		p.Text(l.desc, &descRc, t.MutedForeground,
			xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap|
				xcc.TextTrimming_EllipsisCharacter)
	}
	return 0
}

// ===== 生命周期 =====

func (n *NavigationMenu) alive() bool {
	return n != nil && n.root != nil && xc.XC_IsHXCGUI(n.root.Handle, xcc.XC_ELE) &&
		xc.XC_IsHXCGUI(n.hWnd, xcc.XC_WINDOW)
}

func (n *NavigationMenu) stopTimers() {
	n.stopOpenTimer()
	n.stopCloseTimer()
	n.stopWatch()
	if n.skipTimer != nil {
		n.skipTimer.Stop()
		n.skipTimer = nil
	}
}

func (n *NavigationMenu) cleanup() {
	if n == nil {
		return
	}
	n.stopTimers()
	if n.panel != nil {
		n.panel.Destroy()
		n.panel, n.content = nil, nil
	}
	n.open = -1
	n.hoverLink = nil
	n.kbLink = nil
	n.items = nil
}

// emit 派发值变化(只在值真的变了时回调)。
func (n *NavigationMenu) emit() {
	v := n.Value()
	if v == n.lastEmitted {
		return
	}
	n.lastEmitted = v
	for _, fn := range n.changes {
		fn(v)
	}
}
