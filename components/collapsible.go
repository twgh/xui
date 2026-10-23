package ui

import (
	"math"
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// collapsibleDuration 是展开/收起的高度过渡时长。
//
// shadcn 的 collapsible.tsx 本身不带样式, 但 Radix 给内容暴露了
// --radix-collapsible-content-height, 官方动画示例(animate-collapsible-down/up)
// 就是 200ms ease-out; 这里用同一节奏。传 0 关闭过渡(见 WithCollapsibleDuration)。
const collapsibleDuration = 200 * time.Millisecond

// Collapsible 可折叠面板: 根容器 + 触发按钮 + 内容区。
//
// 与 shadcn / radix 的 Collapsible 对齐:
//   - 状态只有展开/收起两种, `open` / `onOpenChange` 对应 Open()/SetOpen()/OnOpenChange();
//   - 触发按钮由调用方创建, 用 BindTrigger 绑定: 点击切换、展开时带 aria-expanded 底色,
//     禁用时按钮一起禁用;
//   - 内容放进 Content()/ContentStack(), 收起时内容隐藏(不可见、不占布局), 展开时按内容自撑高度。
//
// 与 radix 的差异:
//   - 根容器是**竖向流式布局**(触发区在上、内容在下)。radix 的根是普通 div, 任意布局都行;
//     Go 侧若要横向排布, 把触发区放进内容上方的自定义行即可。
//   - 内容高度切换带 200ms 过渡(见 collapsibleDuration); 收起动画结束后才隐藏内容。
//   - 内容区的外边距不参与动画: 收起时元素隐藏、高度归零, 但外边距在动画期间仍会占位。
//     需要内边距请加在内容区里的子容器上(见 Content 的说明)。
type Collapsible struct {
	*widget.LayoutEle

	content  *widget.LayoutEle
	open     bool
	disabled bool

	triggers []*Button
	onChange []func(open bool)

	duration time.Duration
	presKey  int32 // 每个实例独立的 presence 键(见 xui.NextPresenceKey)
	targetH  int32 // 本次过渡的自然高度(动画进度乘它得到当前高度)
}

// CollapsibleOption 可折叠面板配置项。
type CollapsibleOption func(*Collapsible)

// WithCollapsibleOpen 设置初始为展开状态(不播放动画)。
func WithCollapsibleOpen(open bool) CollapsibleOption {
	return func(c *Collapsible) { c.open = open }
}

// WithCollapsibleDisabled 禁用触发按钮(内容状态不变, 仍可由 SetOpen 程序化切换)。
func WithCollapsibleDisabled(disabled bool) CollapsibleOption {
	return func(c *Collapsible) { c.disabled = disabled }
}

// WithCollapsibleDuration 覆盖高度过渡时长; 传 0 关闭过渡(状态立即生效)。
func WithCollapsibleDuration(d time.Duration) CollapsibleOption {
	return func(c *Collapsible) {
		if d < 0 {
			d = 0
		}
		c.duration = d
	}
}

// NewCollapsible 创建可折叠面板的根容器(竖向流式, 宽铺满、高按内容自撑)。
//
// 用法(触发区 + 内容区都是根容器的子项):
//
//	c := ui.NewCollapsible(0, 0, 0, 0, parent.Handle)
//	xui.ItemFill(c)
//	btn := ui.NewButton(0, 0, 0, 0, "产品详情", c.Handle,
//		ui.WithVariant(ui.ButtonGhost),
//		ui.WithButtonAlign(ui.ButtonAlignBetween),
//		ui.WithButtonTrailingIcon(icons.IconChevronDown))
//	xui.ItemFill(btn)
//	c.BindTrigger(btn)
//	body := c.ContentStack(8)
//	ui.NewLabel(0, 0, 0, 20, "收起/展开的内容", body.Handle)
func NewCollapsible(x, y, cx, cy int32, hParent int, opts ...CollapsibleOption) *Collapsible {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	c := &Collapsible{
		LayoutEle: ele,
		duration:  collapsibleDuration,
		presKey:   xui.NextPresenceKey(),
	}
	for _, o := range opts {
		o(c)
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false) // 竖向溢出换列会把内容甩到根容器右边(见 DESIGN 41)
	ele.EnableBkTransparent(true)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.AttachWindow(ele.Handle)
	// 内容区不能在这里建: 它是根容器的子项, 必须等调用方先放好触发区/其它子项,
	// 否则 Content 会排在它们前面(子元素按创建顺序参与布局)。
	return c
}

// Content 返回内容容器(竖向流式, 宽铺满)。第一次调用时创建, 因此要在放完
// 触发区之后调用它, 内容才会排在触发区下面。
//
// 内容区高度由组件管理(收起 0 / 展开 Auto), 不要自己给它声明高度; 需要内边距时
// 在内容区里放一层子容器并给它设外边距 —— 直接给内容区设外边距会在收起动画期间
// 留下一条空白。
func (c *Collapsible) Content() *widget.LayoutEle {
	if c == nil {
		return nil
	}
	c.ensureContent()
	return c.content
}

// ContentStack 设置内容区子项间距并返回它(与 Card.ContentStack 同名同义)。
func (c *Collapsible) ContentStack(space int32) *widget.LayoutEle {
	if c == nil {
		return nil
	}
	c.ensureContent()
	c.content.SetSpace(space)
	return c.content
}

// BindTrigger 把按钮绑定为触发区: 点击切换展开状态, 展开时按钮进入 aria-expanded
// 视觉态(幽灵/描边变体换 bg-muted), 禁用状态自动同步。
//
// 按钮必须是根容器的子项(布局位置由调用方决定); 可以绑定多个触发区。
func (c *Collapsible) BindTrigger(btn *Button) *Collapsible {
	if c == nil || btn == nil {
		return c
	}
	for _, b := range c.triggers {
		if b == btn || (b != nil && b.Handle == btn.Handle) {
			return c
		}
	}
	c.triggers = append(c.triggers, btn)
	btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if !c.disabled {
			c.Toggle()
		}
		return 0
	})
	btn.SetExpanded(c.open)
	btn.Enable(!c.disabled)
	return c
}

// Open 返回当前是否展开。
func (c *Collapsible) Open() bool {
	if c == nil {
		return false
	}
	return c.open
}

// SetOpen 设置展开状态并播放高度过渡。程序化调用不受 Disabled 限制(同 radix:
// disabled 只禁用触发区, open 仍可受控)。
func (c *Collapsible) SetOpen(open bool) *Collapsible {
	if c == nil || c.open == open {
		return c
	}
	c.open = open
	c.ensureContent()
	c.syncTriggers()
	if c.content != nil {
		if open {
			c.expand()
		} else {
			c.collapse()
		}
	}
	for _, fn := range c.onChange {
		fn(open)
	}
	return c
}

// Toggle 反转展开状态(Disabled 时无效)。
func (c *Collapsible) Toggle() *Collapsible {
	if c == nil || c.disabled {
		return c
	}
	return c.SetOpen(!c.open)
}

// SetDisabled 启用/禁用全部触发区。已展开的内容不会因此收起(同 radix)。
func (c *Collapsible) SetDisabled(disabled bool) *Collapsible {
	if c == nil {
		return c
	}
	c.disabled = disabled
	c.syncTriggers()
	return c
}

// Disabled 返回触发区是否被禁用。
func (c *Collapsible) Disabled() bool {
	if c == nil {
		return false
	}
	return c.disabled
}

// OnOpenChange 注册展开状态变化回调; 可注册多个。
func (c *Collapsible) OnOpenChange(fn func(open bool)) *Collapsible {
	if c == nil || fn == nil {
		return c
	}
	c.onChange = append(c.onChange, fn)
	return c
}

func (c *Collapsible) ensureContent() {
	if c == nil || c.content != nil {
		return
	}
	l := widget.NewLayoutEle(0, 0, 0, 0, c.Handle)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableAutoWrap(false)
	l.EnableBkTransparent(true)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	if !c.open {
		l.Show(false)
		l.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 0)
	}
	c.content = l
}

func (c *Collapsible) syncTriggers() {
	for _, b := range c.triggers {
		// 触发按钮可能先于根容器被销毁(句柄失效后调用 xcgui 会弹错误框)。
		if b == nil || !xc.XC_IsHXCGUI(b.Handle, xcc.XC_ELE) {
			continue
		}
		b.SetExpanded(c.open)
		b.Enable(!c.disabled)
	}
}

// expand 展开: 先量出内容自然高度, 再从 0 动画到它。量高和动画都只在 UI 线程里做。
func (c *Collapsible) expand() {
	c.content.Show(true)
	c.content.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	c.relayout()
	target := c.content.GetHeight()
	if target <= 0 || c.duration <= 0 {
		// 还没拿到宽度(隐藏页/首帧)或关闭了过渡: 直接展开, 交给布局自然长高。
		c.relayout()
		c.redrawNow()
		return
	}
	c.targetH = target
	c.setContentHeight(0)
	c.relayout()
	c.startPresence(true)
}

// collapse 收起: 重新量一次自然高度(展开期间内容可能变过), 锁住高度后逐帧缩到 0。
func (c *Collapsible) collapse() {
	if !c.content.IsShow() {
		c.applyClosed()
		return
	}
	c.content.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	c.relayout()
	target := c.content.GetHeight()
	if target <= 0 || c.duration <= 0 {
		c.applyClosed()
		c.redrawNow()
		return
	}
	c.targetH = target
	c.setContentHeight(target)
	c.relayout()
	c.startPresence(false)
}

func (c *Collapsible) startPresence(open bool) {
	xui.Presence(c.Handle, c.presKey, open, c.duration, c.applyHeight, func(v float64) {
		if open {
			// 动画结束恢复 Auto: 之后内容再变高/变矮都跟随布局。
			c.content.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
			c.relayout()
		} else {
			c.applyClosed()
		}
		c.redrawNow()
	})
}

// applyHeight 把过渡进度落到内容区高度上。
func (c *Collapsible) applyHeight(v float64) {
	if c == nil || c.content == nil {
		return
	}
	h := int32(math.Round(float64(c.targetH) * v))
	if h < 0 {
		h = 0
	}
	if h == c.content.GetHeight() {
		return
	}
	c.setContentHeight(h)
	c.relayout()
	// 内容区高度变化会让下面的兄弟项整体上移/下移, 必须整窗重绘; Presence 只负责
	// 重绘 owner(根元素), 盖不到被推走的兄弟项。
	c.redrawNow()
}

// applyClosed 是收起的终态: 隐藏内容并清除占位。
func (c *Collapsible) applyClosed() {
	if c == nil || c.content == nil {
		return
	}
	c.content.Show(false)
	c.content.SetHeight(0)
	c.content.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 0)
	c.relayout()
}

func (c *Collapsible) setContentHeight(h int32) {
	c.content.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	if c.content.GetHeight() != h {
		c.content.SetHeight(h)
	}
}

// relayout 从自己开始逐级向上重排: 根容器是 Auto 高, 内容高度变化后必须让父级重新
// 量一遍, 否则卡片/行/滚动范围仍按旧高度排版(同 alert.go 的 reflow 路线)。
// 遇到 ScrollArea 的 LayoutFrame 交给框架排 —— 直接排它的内容区会把子项宽度打成 0。
func (c *Collapsible) relayout() {
	if c == nil || c.LayoutEle == nil || !xc.XC_IsHXCGUI(c.Handle, xcc.XC_ELE) {
		return
	}
	xc.XEle_AdjustLayout(c.Handle)
	for p := xc.XWidget_GetParentEle(c.Handle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(p)
			return
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(pp)
			return
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_ELE) {
			xc.XEle_AdjustLayout(p)
			continue
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
		}
		return
	}
}

// redrawNow 立即重绘宿主窗口: 高度变化会让下面的兄弟项整体移动, 只重绘根元素会留下
// 旧位置的内容(过渡逐帧跑, 延迟重绘会被合并掉, 见 DESIGN 45/46)。
func (c *Collapsible) redrawNow() {
	if c == nil || c.LayoutEle == nil {
		return
	}
	hwnd := xc.XWidget_GetHWND(c.Handle)
	if hwnd == 0 {
		return
	}
	if hw := xc.XC_hWindowFromHWnd(hwnd); hw != 0 {
		xc.XWnd_Redraw(hw, true)
	}
}
