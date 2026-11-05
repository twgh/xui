package ui

import (
	"fmt"
	"math"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// StatusBar 几何(逻辑像素)。shadcn 没有官方 status-bar, 尺寸是本库按桌面状态栏
// 的通行形态定的: h-7(28) 整宽横条 + 顶边 1px, 条目 h-5(20) / text-xs(主题字号 -2)。
const (
	statusBarHeight   int32 = 28
	statusBarPadX     int32 = 8
	statusBarGap      int32 = 2
	statusBarItemH    int32 = 20
	statusBarItemPadX int32 = 6
	statusBarItemGap  int32 = 4
	statusBarItemIcon int32 = 12
	statusBarGripW    int32 = 16
	statusBarGripH    int32 = 16
	statusBarMinWinW  int32 = 320
	statusBarMinWinH  int32 = 240
)

// StatusBarOption 是状态栏配置项。
type StatusBarOption func(*StatusBar)

// WithStatusBarHeight 覆盖状态栏高度(默认 28); 构造参数 cy > 0 时以 cy 为准。
func WithStatusBarHeight(h int32) StatusBarOption {
	return func(b *StatusBar) {
		if h >= 20 {
			b.height = h
		}
	}
}

// WithStatusBarGrip 开/关右下角尺寸抓手(默认开)。抓手拖动改的是**宿主窗口大小**。
func WithStatusBarGrip(v bool) StatusBarOption {
	return func(b *StatusBar) { b.gripEnabled = v }
}

// WithStatusBarSizeReadout 在右槽内置一条实时窗口尺寸读数("W × H", 逻辑像素)。
func WithStatusBarSizeReadout() StatusBarOption {
	return func(b *StatusBar) { b.showSize = true }
}

// WithStatusBarMinWindowSize 设置抓手拖动时的窗口尺寸下限(默认 320×240, 逻辑像素)。
func WithStatusBarMinWindowSize(w, h int32) StatusBarOption {
	return func(b *StatusBar) {
		if w > 0 {
			b.minW = w
		}
		if h > 0 {
			b.minH = h
		}
	}
}

// WithStatusBarTooltip 注入外部提示管理器; 不给时组件自己建一个(条目/抓手气泡默认在上方)。
func WithStatusBarTooltip(t *Tooltip) StatusBarOption {
	return func(b *StatusBar) { b.tip = t }
}

// StatusBar 窗口底部状态栏: 左/中/右三个槽位 + 右下角尺寸抓手 + 条目悬停提示。
//
// shadcn 没有官方 tsx, 交互语义由本库定死(见 DESIGN.md 第四节末的语义表):
//   - 整条是横向流式容器(宽 Fill / 高 Fixed), 底色 bg-card + 顶边 1px bg-border;
//     左右槽各 Weight 1、中槽 Auto, 因此中槽按**整条**中心对齐(右侧抓手占 16 会让它
//     左偏半个抓手宽), 不是"左右剩余空间"的中心; 右槽开头是一段 Weight 撑子,
//     把右槽条目顶到行尾。
//   - 槽位就是普通 LayoutEle, 可以放任意 xui 控件; 条目助手 NewStatusBarItem 只是
//     自绘文字/图标 + hover 过渡 + 点击回调 + 提示气泡的便捷形态。
//   - 尺寸条 = 右下角 16 宽 NWSE 抓手(按住拖动改宿主窗口大小) + 可选尺寸读数。
//     拖动以**屏幕物理坐标增量 / 窗口 DPI** 换算成逻辑像素: 抓手自己会随窗口缩放
//     移动, 拿元素坐标当基准会漂。最大化态 / 缩到下限都不动作。
//
// 用法(挂在窗口的流式布局末尾, 内容区用 Weight 高, 状态栏自然落到窗口底部):
//
//	sb := ui.NewStatusBar(0, 0, 0, 0, w.Handle, ui.WithStatusBarSizeReadout())
//	sb.AddLeft("就绪", ui.WithStatusBarItemTooltip("构建完成"))
//	sb.AddCenter("main.go")
//	sb.AddRight("UTF-8", ui.WithStatusBarItemTooltip("文件编码"))
//	w.AdjustLayout()
type StatusBar struct {
	*widget.LayoutEle
	left   *widget.LayoutEle
	center *widget.LayoutEle
	right  *widget.LayoutEle
	grip   *statusBarGrip
	tip    *Tooltip

	hWindow     int
	height      int32
	minW, minH  int32
	gripEnabled bool
	showSize    bool
	sizeItem    *StatusBarItem
	sizeHooked  bool
	destroyed   bool
	reflowing   bool
}

// NewStatusBar 创建状态栏。宽恒铺满父级(Fill), 高 = cy(<=0 用默认 28); cx 只作初始宽。
func NewStatusBar(x, y, cx, cy int32, hParent int, opts ...StatusBarOption) *StatusBar {
	b := &StatusBar{
		height:      statusBarHeight,
		minW:        statusBarMinWinW,
		minH:        statusBarMinWinH,
		gripEnabled: true,
	}
	for _, o := range opts {
		o(b)
	}
	if cy > 0 {
		b.height = cy
	}
	ele := widget.NewLayoutEle(x, y, cx, b.height, hParent)
	if ele == nil {
		return nil
	}
	b.LayoutEle = ele
	ele.EnableHorizon(true)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(0)
	ele.SetAlignV(xcc.Layout_Align_Center)
	// 布局元素是纯排版壳: 保持鼠标穿透, 让槽里的条目/抓手自己收事件(见 DESIGN 185)。
	ele.EnableMouseThrough(true)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, b.height)
	// 左右内边距用元素 padding(不能用子项外边距, 见 DESIGN 129)。
	// 右侧不留内边距: 抓手是最后一项, 要让它贴到窗口右下角(线条自己再内缩 3);
	// 关掉抓手时恢复常规内边距, 否则条目会顶到边上。
	xc.XEle_SetPadding(ele.Handle, statusBarPadX, 0, b.gripPadRight(), 0)
	ele.AddEvent_Paint(b.onPaint)
	ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		b.destroyed = true
		return 0
	})
	xui.AttachWindow(ele.Handle)
	b.hWindow = xui.HostWindowOf(ele.Handle)
	if b.tip == nil {
		b.tip = NewTooltip()
	}

	// 创建顺序 = 布局顺序: 左 Weight / 中 Auto / 右 Weight / 抓手 Fixed。
	b.left = b.newSlot(xcc.Layout_Size_Weight, 1)
	b.center = b.newSlot(xcc.Layout_Size_Auto, 0)
	b.right = b.newSlot(xcc.Layout_Size_Weight, 1)
	b.newRightSpacer()
	b.grip = b.newGrip()
	if !b.gripEnabled && b.grip != nil {
		b.grip.setShown(false)
	}
	if b.showSize {
		b.sizeItem = b.AddRight("— × —", WithStatusBarItemTooltip("窗口尺寸(逻辑像素)"))
		b.watchWindowSize()
	}
	ele.AdjustLayout()
	return b
}

func (b *StatusBar) newSlot(wType xcc.Layout_Size_, w int32) *widget.LayoutEle {
	s := widget.NewLayoutEle(0, 0, 0, b.height, b.Handle)
	if s == nil {
		return nil
	}
	s.EnableHorizon(true)
	s.EnableLayout(true)
	s.EnableBkTransparent(true)
	s.EnableAutoWrap(false)
	s.SetSpace(statusBarGap)
	s.SetAlignV(xcc.Layout_Align_Center)
	s.EnableMouseThrough(true)
	s.LayoutItem_SetWidth(wType, w)
	s.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, b.height)
	return s
}

// newRightSpacer 在右槽最前面放一段 Weight 撑子, 把后续条目顶到右边缘。
// 不依赖 SetAlignH(Right): 布局的主轴对齐在不同容器里语义不稳, Weight 撑子是
// 本项目已实测的"靠尾"写法(与 AppWindow 标题栏的 Weight 标题同款)。
func (b *StatusBar) newRightSpacer() {
	if b.right == nil {
		return
	}
	sp := widget.NewLayoutEle(0, 0, 0, b.height, b.right.Handle)
	if sp == nil {
		return
	}
	sp.EnableHorizon(true)
	sp.EnableLayout(true)
	sp.EnableBkTransparent(true)
	sp.EnableMouseThrough(true)
	sp.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	sp.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, b.height)
}

// Left / Center / Right 返回三个槽位, 可以直接往里面放任意控件(放原生控件时
// 记得自己声明布局项宽, 同 KbdGroup 的用法)。
func (b *StatusBar) Left() *widget.LayoutEle {
	if b == nil {
		return nil
	}
	return b.left
}

func (b *StatusBar) Center() *widget.LayoutEle {
	if b == nil {
		return nil
	}
	return b.center
}

func (b *StatusBar) Right() *widget.LayoutEle {
	if b == nil {
		return nil
	}
	return b.right
}

// Bar 返回根容器句柄(可挂 tooltip 或改布局项)。
func (b *StatusBar) Bar() int {
	if b == nil || b.LayoutEle == nil {
		return 0
	}
	return b.Handle
}

// Grip 返回尺寸抓手的句柄(0 = 没有抓手)。
func (b *StatusBar) Grip() int {
	if b == nil || b.grip == nil {
		return 0
	}
	return b.grip.Handle
}

// Tooltip 返回组件使用的提示管理器, 便于继续给自定义槽位内容绑气泡。
func (b *StatusBar) Tooltip() *Tooltip {
	if b == nil {
		return nil
	}
	return b.tip
}

// suppressTip 压住/放开某个元素的气泡(尺寸抓手拖动期间用, 见 Tooltip.suppress)。
// 没有注入提示管理器(WithStatusBarTooltip)时无操作。
func (b *StatusBar) suppressTip(hEle int, on bool) {
	if b == nil || b.tip == nil {
		return
	}
	b.tip.suppress(hEle, on)
}

// Height 返回状态栏高度(逻辑像素)。
func (b *StatusBar) Height() int32 {
	if b == nil {
		return 0
	}
	return b.height
}

// GripEnabled 报告尺寸抓手是否启用。
func (b *StatusBar) GripEnabled() bool { return b != nil && b.gripEnabled }

// gripPadRight 是条的右侧内边距: 抓手启用时不留(抓手是最后一项, 要贴住右下角),
// 关掉时恢复常规内边距, 免得条目顶到窗口边上。
func (b *StatusBar) gripPadRight() int32 {
	if b != nil && b.gripEnabled {
		return 0
	}
	return statusBarPadX
}

// SetGrip 运行时开/关尺寸抓手(隐藏的项不参与布局, 中槽会回到整条中心)。
func (b *StatusBar) SetGrip(v bool) *StatusBar {
	if b == nil || b.grip == nil || b.gripEnabled == v {
		return b
	}
	b.gripEnabled = v
	if !v {
		b.grip.endDrag()
	}
	b.grip.setShown(v)
	xc.XEle_SetPadding(b.Handle, statusBarPadX, 0, b.gripPadRight(), 0)
	b.reflow()
	return b
}

// SetSizeReadout 运行时开/关内置尺寸读数。
func (b *StatusBar) SetSizeReadout(v bool) *StatusBar {
	if b == nil || b.destroyed {
		return b
	}
	if v {
		if b.sizeItem == nil {
			b.sizeItem = b.AddRight("— × —", WithStatusBarItemTooltip("窗口尺寸(逻辑像素)"))
		} else {
			b.sizeItem.Show(true)
		}
		b.watchWindowSize()
	} else if b.sizeItem != nil {
		b.sizeItem.Show(false)
	}
	b.reflow()
	return b
}

// AddLeft / AddCenter / AddRight 在对应槽位追加一个自绘条目。
func (b *StatusBar) AddLeft(text string, opts ...StatusBarItemOption) *StatusBarItem {
	return b.addItem(b.left, text, opts)
}

func (b *StatusBar) AddCenter(text string, opts ...StatusBarItemOption) *StatusBarItem {
	return b.addItem(b.center, text, opts)
}

func (b *StatusBar) AddRight(text string, opts ...StatusBarItemOption) *StatusBarItem {
	return b.addItem(b.right, text, opts)
}

func (b *StatusBar) addItem(slot *widget.LayoutEle, text string, opts []StatusBarItemOption) *StatusBarItem {
	if b == nil || slot == nil {
		return nil
	}
	it := NewStatusBarItem(slot.Handle, text, opts...)
	if it == nil {
		return nil
	}
	it.bar = b
	it.tip = b.tip
	it.syncTip()
	return it
}

// reflow 条目宽度变化后重排三个槽位与整条。reflowing 挡住"重排 → 尺寸事件 →
// 再重量宽"的递归(同 Item.reflowBody 的做法)。
func (b *StatusBar) reflow() {
	if b == nil || b.LayoutEle == nil || b.destroyed || b.reflowing {
		return
	}
	b.reflowing = true
	for _, s := range []*widget.LayoutEle{b.left, b.center, b.right} {
		if s != nil {
			s.AdjustLayout()
		}
	}
	b.AdjustLayout()
	b.Redraw(false)
	b.reflowing = false
}

// watchWindowSize 让尺寸读数跟随宿主窗口的 WM_SIZE。读数必须延后到消息处理之后
// 再取客户区矩形, 同 AppWindow 最大化图标的时机(见 window.go)。
func (b *StatusBar) watchWindowSize() {
	if b == nil || b.hWindow == 0 || b.sizeHooked {
		return
	}
	w := window.NewByHandle(b.hWindow)
	if w == nil {
		return
	}
	b.sizeHooked = true
	w.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
		if message == wapi.WM_SIZE {
			xc.XC_CallUT(func() { b.syncSizeText() })
		}
		return 0
	})
	app.AfterFunc(0, func() { b.syncSizeText() })
}

func (b *StatusBar) syncSizeText() {
	if b == nil || b.destroyed || b.sizeItem == nil || !b.sizeItem.IsShow() || b.hWindow == 0 {
		return
	}
	if !xc.XC_IsHXCGUI(b.hWindow, xcc.XC_WINDOW) {
		return
	}
	var rc xc.RECT
	if !xc.XWnd_GetClientRect(b.hWindow, &rc) {
		return
	}
	w, h := rc.Right-rc.Left, rc.Bottom-rc.Top
	if w <= 0 || h <= 0 {
		return
	}
	b.sizeItem.SetText(statusBarSizeText(w, h))
}

func statusBarSizeText(w, h int32) string {
	return fmt.Sprintf("%d × %d", w, h)
}

func statusBarClamp(v, floor int32) int32 {
	if v < floor {
		return floor
	}
	return v
}

func (b *StatusBar) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	// 整条不透明打底: 透明元素重绘不擦底, 半透明 hover 痕迹会残留(见 DESIGN 28)。
	p.Draw.FillRectColor(&p.Rc, uint32(t.Card))
	paintStatusBarTopLine(p, t)
	return 0
}

// paintStatusBarTopLine 画 1 个**物理像素**的顶边: 1 逻辑像素在 150% DPI 下是
// 1.5 物理像素, 交给 D2D 按相位取整会一条 1 行一条 2 行(同 separator.go)。
func paintStatusBarTopLine(p *xui.Painter, t *xui.Theme) {
	sc := p.Scale
	if sc <= 0 {
		sc = 1
	}
	c := math.Floor(float64(p.Rc.Top) * sc)
	rc := xc.RECTF{
		Left:   float32(p.Rc.Left),
		Top:    float32(c / sc),
		Right:  float32(p.Rc.Right),
		Bottom: float32((c + 1) / sc),
	}
	p.Draw.FillRectColorF(&rc, uint32(t.Border))
}

// ===== 条目 =====

// StatusBarItemOption 是条目配置项。
type StatusBarItemOption func(*StatusBarItem)

// WithStatusBarItemIcon 在文字前放一枚 lucide 图标(12px path / SVG, 见 xui.Painter.Icon)。
func WithStatusBarItemIcon(path string) StatusBarItemOption {
	return func(i *StatusBarItem) { i.icon = path }
}

// WithStatusBarItemTooltip 给条目绑悬停气泡(状态栏在窗口底部, 气泡固定朝上)。
func WithStatusBarItemTooltip(text string) StatusBarItemOption {
	return func(i *StatusBarItem) { i.tipText = text }
}

// WithStatusBarItemOnClick 注册点击回调(在 LButtonUp 触发, 禁用态不触发)。
func WithStatusBarItemOnClick(fn func()) StatusBarItemOption {
	return func(i *StatusBarItem) { i.onClick = fn }
}

// WithStatusBarItemDisabled 建一个禁用条目(50% 字色, 不响应 hover / 点击)。
func WithStatusBarItemDisabled() StatusBarItemOption {
	return func(i *StatusBarItem) { i.disabled = true }
}

// WithStatusBarItemActive 用 primary 实心底 + primary-foreground 字(如"只读/语言模式"这类当前态)。
func WithStatusBarItemActive() StatusBarItemOption {
	return func(i *StatusBarItem) { i.active = true }
}

// WithStatusBarItemWidth 锁固定宽(默认按文字/图标量宽); 超长文字画省略号。
func WithStatusBarItemWidth(w int32) StatusBarItemOption {
	return func(i *StatusBarItem) { i.fixedW = w }
}

// StatusBarItem 状态栏条目: 自绘文字/图标 + 150ms hover 过渡 + 点击回调 + 提示气泡。
//
// 条目不是原生 Button, 不进 Tab 焦点链(与本库其它嵌套布局里的控件同一限制,
// 见 DESIGN 157); 悬停/按下/禁用/激活四态都由自己的 paint 解析。
type StatusBarItem struct {
	*widget.Element
	bar      *StatusBar
	tip      *Tooltip
	text     string
	icon     string
	tipText  string
	onClick  func()
	disabled bool
	active   bool
	hover    bool
	pressed  bool
	fixedW   int32
}

// NewStatusBarItem 创建独立条目; 放进 StatusBar.Left()/Center()/Right() 槽位即可。
// 通过 StatusBar.AddXxx 创建的条目会自动接上状态栏的提示管理器。
func NewStatusBarItem(hParent int, text string, opts ...StatusBarItemOption) *StatusBarItem {
	i := &StatusBarItem{text: text}
	for _, o := range opts {
		o(i)
	}
	t := xui.Current()
	w := i.fixedW
	if w <= 0 {
		w = i.naturalWidth(t)
	}
	ele := widget.NewElement(0, 0, w, statusBarItemH, hParent)
	if ele == nil {
		return nil
	}
	i.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, statusBarItemH)
	ele.AddEvent_Paint(i.onPaint)
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		i.setHover(true)
		return 0
	})
	ele.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		i.setHover(true)
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		i.setHover(false)
		i.pressed = false
		ele.Redraw(true)
		return 0
	})
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if !i.disabled {
			i.pressed = true
			ele.Redraw(true)
		}
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		wasPressed := i.pressed
		i.pressed = false
		if !i.disabled && wasPressed && i.onClick != nil {
			xui.NotePointerPress()
			i.onClick()
		}
		ele.Redraw(true)
		return 0
	})
	xui.AttachWindow(ele.Handle)
	// 主题切换可能换字族/字号, 自动宽要跟着重量(同 Kbd)。
	xui.OnThemeChangeFor(ele.Handle, func(th *xui.Theme) {
		i.syncWidth(th)
		i.Redraw(false)
	})
	i.syncTip()
	return i
}

func (i *StatusBarItem) setHover(v bool) {
	if i == nil || i.Element == nil || i.hover == v {
		return
	}
	i.hover = v
	// hover 反向过渡靠 paint 每帧推进; 离开后没有 MouseMove, 必须主动补这一帧。
	i.Redraw(true)
}

func statusBarFont(t *xui.Theme) int {
	if t == nil {
		return 0
	}
	spec := t.FontDefault
	spec.Size -= 2
	return xui.FontHandle(spec)
}

func (i *StatusBarItem) naturalWidth(t *xui.Theme) int32 {
	if i == nil || t == nil {
		return statusBarItemH
	}
	w := 2*statusBarItemPadX + xui.TextWidth(i.text, statusBarFont(t))
	if i.icon != "" {
		w += statusBarItemIcon + statusBarItemGap
	}
	if w < statusBarItemH {
		w = statusBarItemH
	}
	return w
}

func (i *StatusBarItem) syncWidth(t *xui.Theme) {
	if i == nil || i.Element == nil || i.fixedW > 0 {
		return
	}
	w := i.naturalWidth(t)
	if w <= 0 {
		return
	}
	i.SetWidth(w)
	i.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	if i.bar != nil {
		i.bar.reflow()
	}
}

// SetText 改文字并按内容重量宽; 程序化改值不触发点击回调。
func (i *StatusBarItem) SetText(text string) *StatusBarItem {
	if i == nil || i.text == text {
		return i
	}
	i.text = text
	i.syncWidth(xui.Current())
	i.Redraw(false)
	return i
}

// Text 返回当前文字。
func (i *StatusBarItem) Text() string {
	if i == nil {
		return ""
	}
	return i.text
}

// SetIcon 改行内图标(空串 = 去掉图标)。
func (i *StatusBarItem) SetIcon(path string) *StatusBarItem {
	if i == nil || i.icon == path {
		return i
	}
	i.icon = path
	i.syncWidth(xui.Current())
	i.Redraw(false)
	return i
}

// SetTooltip 改悬停提示(空串 = 解绑)。
func (i *StatusBarItem) SetTooltip(text string) *StatusBarItem {
	if i == nil {
		return i
	}
	i.tipText = text
	i.syncTip()
	return i
}

func (i *StatusBarItem) syncTip() {
	if i == nil || i.Element == nil || i.tip == nil {
		return
	}
	if i.tipText == "" {
		i.tip.Unbind(i.Handle)
		return
	}
	i.tip.BindTip(i.Handle, i.tipText, WithTipSide(TipTop))
}

// SetDisabled 开/关禁用态(50% 字色, 不响应 hover / 点击)。
func (i *StatusBarItem) SetDisabled(v bool) *StatusBarItem {
	if i == nil || i.disabled == v {
		return i
	}
	i.disabled = v
	if v {
		i.hover, i.pressed = false, false
		xui.Hover(i.Handle, false)
	}
	i.Redraw(true)
	return i
}

// Disabled 报告禁用态。
func (i *StatusBarItem) Disabled() bool { return i != nil && i.disabled }

// SetActive 开/关 primary 实心当前态。
func (i *StatusBarItem) SetActive(v bool) *StatusBarItem {
	if i == nil || i.active == v {
		return i
	}
	i.active = v
	i.Redraw(false)
	return i
}

// Active 报告当前态。
func (i *StatusBarItem) Active() bool { return i != nil && i.active }

// OnClick 追加/替换点击回调(与 WithStatusBarItemOnClick 等价)。
func (i *StatusBarItem) OnClick(fn func()) *StatusBarItem {
	if i != nil {
		i.onClick = fn
	}
	return i
}

func (i *StatusBarItem) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	hv := xui.Hover(hEle, i.hover && !i.disabled)
	r := t.RadiusSm()
	// 静止态底色是透明的, 每帧先铺宿主底色擦掉上一帧的半透明 hover 痕迹。
	p.FillRound(&p.Rc, t.Card, r)
	if i.active {
		p.FillRound(&p.Rc, t.Primary, r)
		if hv > 0 {
			p.FillRound(&p.Rc, t.PrimaryForeground.ScaleAlpha(0.12*hv), r)
		}
	} else if hv > 0 {
		p.FillRound(&p.Rc, t.Accent.ScaleAlpha(hv), r)
		if i.pressed && !i.disabled {
			p.FillRound(&p.Rc, t.Accent.ScaleAlpha(0.5), r)
		}
	}

	fore := xui.Fade(t.MutedForeground, t.Foreground, hv)
	if i.active {
		fore = t.PrimaryForeground
	}
	if i.disabled {
		fore = t.MutedForeground.ScaleAlpha(0.5)
	}
	if f := statusBarFont(t); f != 0 {
		p.Draw.SetFont(f)
	}
	x := p.Rc.Left + statusBarItemPadX
	cy := (p.Rc.Top + p.Rc.Bottom) / 2
	if i.icon != "" {
		rc := xc.RECT{
			Left:   x,
			Top:    cy - statusBarItemIcon/2,
			Right:  x + statusBarItemIcon,
			Bottom: cy + statusBarItemIcon/2,
		}
		p.Icon(i.icon, &rc, fore, 2)
		x += statusBarItemIcon + statusBarItemGap
	}
	textRc := xc.RECT{
		Left:   x,
		Top:    p.Rc.Top,
		Right:  p.Rc.Right - statusBarItemPadX,
		Bottom: p.Rc.Bottom,
	}
	p.Text(i.text, &textRc, fore,
		xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|
			xcc.TextFormatFlag_NoWrap|xcc.TextTrimming_EllipsisCharacter)
	return 0
}

// NewStatusBarSeparator 建一条槽内的 1px 竖分隔线(高 12)。
func NewStatusBarSeparator(hParent int) *widget.Element {
	ele := widget.NewElement(0, 0, 1, 12, hParent)
	if ele == nil {
		return nil
	}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableMouseThrough(true)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 1)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 12)
	ele.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		sc := p.Scale
		if sc <= 0 {
			sc = 1
		}
		c := math.Floor(float64(p.Rc.Left+p.Rc.Right) / 2 * sc)
		rc := xc.RECTF{
			Left:   float32(c / sc),
			Top:    float32(p.Rc.Top),
			Right:  float32((c + 1) / sc),
			Bottom: float32(p.Rc.Bottom),
		}
		p.Draw.FillRectColorF(&rc, uint32(xui.Current().Border))
		return 0
	})
	xui.AttachWindow(ele.Handle)
	return ele
}

// ===== 尺寸抓手 =====

type statusBarGrip struct {
	*widget.Element
	holder   *widget.LayoutEle // 整条高的容器, 只负责把抓手贴到条的底部
	bar      *StatusBar
	hover    bool
	dragging bool
	startX   int32 // 屏幕物理像素
	startY   int32
	startW   int32 // 窗口逻辑像素
	startH   int32
	scale    float64
}

var statusBarGripCursorHandle uintptr

func statusBarGripCursor() uintptr {
	if statusBarGripCursorHandle == 0 {
		statusBarGripCursorHandle = wapi.LoadImageW(0, uintptr(wapi.IDC_SIZENWSE),
			wapi.IMAGE_CURSOR, 0, 0, wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)
	}
	return statusBarGripCursorHandle
}

func (b *StatusBar) newGrip() *statusBarGrip {
	// 抓手是贴窗口右下角的 16×16 方块: 外面套一个整条高的容器(交叉轴对齐到底部),
	// 抓手自己只占方块那块 —— 否则它跟整条一样高, hover 高亮会在右端竖出一道 28 高的
	// 色块, 而且线条离角还有一大截(实测: 线条离右边 ~9、离底 ~6)。
	holder := widget.NewLayoutEle(0, 0, statusBarGripW, b.height, b.Handle)
	if holder == nil {
		return nil
	}
	holder.EnableMouseThrough(true)
	holder.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, statusBarGripW)
	holder.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, b.height)
	holder.SetAlignV(xcc.Layout_Align_Bottom)

	ele := widget.NewElement(0, 0, statusBarGripW, statusBarGripH, holder.Handle)
	if ele == nil {
		return nil
	}
	g := &statusBarGrip{Element: ele, holder: holder, bar: b}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, statusBarGripW)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, statusBarGripH)
	ele.SetCursor(statusBarGripCursor())
	ele.AddEvent_Paint(g.onPaint)
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		g.setHover(true)
		return 0
	})
	ele.AddEvent_MouseMove(g.onMove)
	ele.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		g.setHover(false)
		return 0
	})
	ele.AddEvent_LButtonDown(g.onDown)
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		g.endDrag()
		return 0
	})
	ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		g.dragging = false
		return 0
	})
	xui.AttachWindow(ele.Handle)
	if b.tip != nil {
		b.tip.BindTip(ele.Handle, "拖动调整窗口大小", WithTipSide(TipTop))
	}
	return g
}

// setShown 连抓手外面那个容器一起显隐: 只显隐抓手本身的话容器仍占着 16 宽,
// 中槽会一直往左偏半个抓手宽。
func (g *statusBarGrip) setShown(v bool) {
	if g == nil || g.Element == nil {
		return
	}
	if g.holder != nil {
		g.holder.Show(v)
		return
	}
	g.Show(v)
}

func (g *statusBarGrip) setHover(v bool) {
	if g == nil || g.Element == nil || g.hover == v {
		return
	}
	g.hover = v
	g.Redraw(true)
}

func (g *statusBarGrip) onDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if g == nil || g.bar == nil || g.bar.destroyed || !g.bar.gripEnabled {
		return 0
	}
	xui.NotePointerPress()
	HideTooltip(hEle)
	b := g.bar
	if b.hWindow == 0 || !xc.XC_IsHXCGUI(b.hWindow, xcc.XC_WINDOW) {
		return 0
	}
	w := window.NewByHandle(b.hWindow)
	if w == nil || w.IsMaxWindow() {
		return 0
	}
	var rc xc.RECT
	xc.XWnd_GetRect(b.hWindow, &rc)
	var pt wapi.POINT
	if !wapi.GetCursorPos(&pt) {
		return 0
	}
	sc := 1.0
	if dpi := xc.XWnd_GetDPI(b.hWindow); dpi > 0 {
		sc = float64(dpi) / 96
	}
	if sc <= 0 {
		sc = 1
	}
	g.startX, g.startY = pt.X, pt.Y
	g.startW, g.startH = rc.Right-rc.Left, rc.Bottom-rc.Top
	g.scale = sc
	g.dragging = true
	// 拖动期间压住抓手的气泡: 被捕获的元素仍会持续收到 MouseMove, 提示会在缩放过程中
	// 反复冒出并停在拖动开始的位置(松手时光标常在窗口外, 没有 MouseLeave 来关它)。
	g.bar.suppressTip(hEle, true)
	// 捕获到抓手: 拖出状态栏(窗口变小时指针可能落在栏外)也能继续跟手(见 DESIGN 140)。
	xc.XEle_SetCapture(hEle, true)
	return 0
}

func (g *statusBarGrip) onMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if g == nil || g.bar == nil {
		return 0
	}
	if g.dragging {
		// 在窗口外松手时收不到 LButtonUp, 用 MK_LBUTTON 收尾(同 ResizableGroup)。
		if nFlags&1 == 0 {
			g.endDrag()
			return 0
		}
		var pt wapi.POINT
		if !wapi.GetCursorPos(&pt) {
			return 0
		}
		dw := int32(math.Round(float64(pt.X-g.startX) / g.scale))
		dh := int32(math.Round(float64(pt.Y-g.startY) / g.scale))
		w := statusBarClamp(g.startW+dw, g.bar.minW)
		h := statusBarClamp(g.startH+dh, g.bar.minH)
		if win := window.NewByHandle(g.bar.hWindow); win != nil {
			win.SetSize(w, h) // XWnd_SetRect 走逻辑像素, 不用再乘 DPI
		}
		return 0
	}
	g.setHover(true)
	return 0
}

func (g *statusBarGrip) endDrag() {
	if g == nil || g.Element == nil {
		return
	}
	if g.dragging {
		g.dragging = false
		xc.XEle_SetCapture(g.Handle, false)
		g.Redraw(false)
	}
	// 收手: 放开拖动期间压住的气泡, 并确保它此刻是收起的 —— 松手时指针往往停在新窗口
	// 之外(窗口被拖小了), 既收不到 MouseLeave 也没有窗口级命中测试来关它, 不主动关掉
	// 就会一直挂在屏幕上(用户报障: 拖完窗口大小后提示气泡不消失)。
	if g.bar != nil {
		g.bar.suppressTip(g.Handle, false)
	}
	HideTooltip(g.Handle)
}

func (g *statusBarGrip) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	hv := xui.Hover(hEle, g.hover)
	p.Draw.FillRectColor(&p.Rc, uint32(t.Card))
	if hv > 0 {
		p.Draw.FillRectColor(&p.Rc, uint32(t.Accent.ScaleAlpha(hv)))
	}
	// 抓手是 16×16 的角落方块(不再顶到条的上沿), 不需要再补画顶边。
	// 三条 45° 斜线(右下角尺寸把手)。用 DrawLineF: 整数版 DrawLine 会把小图形
	// 的坐标截断到逻辑整数, 整体偏左/偏上(见 DESIGN 186)。
	fore := xui.Fade(t.MutedForeground, t.Foreground, hv)
	p.Draw.SetLineWidthF(1)
	p.Draw.SetBrushColor(uint32(fore))
	right := float32(p.Rc.Right - 3)
	bottom := float32(p.Rc.Bottom - 3)
	for i := 0; i < 3; i++ {
		off := float32(i) * 3
		p.Draw.DrawLineF(right-6-off, bottom-off, right-off, bottom-6-off)
	}
	return 0
}
