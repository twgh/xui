package ui

import (
	"math"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// ScreenPopup 是贴在显示器工作区右下角的常驻弹窗: 任意 xui 内容、手动显隐、
// 默认显示与点击都不抢焦点(WS_EX_NOACTIVATE + SW_SHOWNOACTIVATE, 点击再拦
// WM_MOUSEACTIVATE 返回 MA_NOACTIVATE, 见 DESIGN 181)。需要键盘输入时传
// WithScreenPopupFocusable(true): 窗口可被用户点击激活, 但 Show 仍不抢前台,
// 用户点弹窗里的输入框才把焦点切过来, 之后可输入、ESC 收起。
//
// 与 ScreenToaster 的分工: ScreenToast 是"通知" —— 每条一个卡片、自动关闭、堆叠;
// ScreenPopup 是"面板" —— 整个弹窗只有一个, 默认不自动关闭, 内容由调用方决定;
// 需要自动关闭时用 WithScreenPopupDuration。
//
// 默认不抢焦点意味着它拿不到键盘焦点: 内容里的输入框/文本域无法输入、ESC 不会被它收到。
// 鼠标悬停与点击正常; 需要键盘操作的内容用 WithScreenPopupFocusable(true), 或改用 Popover / Dialog。
//
// 用法:
//
//	sp := ui.NewScreenPopup(w.Handle,
//		ui.WithScreenPopupDuration(8*time.Second),
//		ui.WithScreenPopupHeader("同步完成"), // 标题 + 右上角 X
//		ui.WithScreenPopupFocusable(false), // true = 用户点击后可输入(显示仍不抢前台)
//		ui.WithScreenPopupContent(func(c int) {
//			ui.NewLabel(0, 0, 0, 24, "文件已写入磁盘", c)
//		}))
//	sp.Show()
//
// 必须是 UI 线程调用。
type ScreenPopup struct {
	p            *popoverPanel
	hAnchor      int
	width        int32
	height       int32 // > 0 固定高度; 0 = 随内容
	offset       int32
	padding      int32
	duration     time.Duration // > 0 显示后自动关闭; 0 = 常驻
	autoTimer    *time.Timer
	autoGen      int // 计时代次: 取消后迟到的旧回调不能关掉重新显示的弹窗
	onOpenChange func(open bool)
	anchorHook   int // 宿主 WM_DESTROY 监听 ID(销毁时反注册, 避免反复建弹窗时累积)
	anchorGone   bool
	destroyed    bool
}

// ScreenPopupOption 是 ScreenPopup 的配置项。
type ScreenPopupOption func(*screenPopupConfig)

type screenPopupConfig struct {
	width        int32
	height       int32
	offset       int32
	padding      int32
	gap          int32
	duration     time.Duration
	focusable    bool
	headerTitle  string
	closeButton  bool
	onOpenChange func(open bool)
	build        func(content int)
}

const (
	screenPopupDefaultWidth   int32 = 360
	screenPopupDefaultOffset  int32 = 24
	screenPopupDefaultPadding int32 = 16
	screenPopupDefaultGap     int32 = 8
)

// WithScreenPopupWidth 覆盖卡片宽度(默认 360)。
func WithScreenPopupWidth(w int32) ScreenPopupOption {
	return func(c *screenPopupConfig) {
		if w > 0 {
			c.width = w
		}
	}
}

// WithScreenPopupHeight 固定卡片高度(默认 0 = 按内容自动量高)。
// 内容比固定高度高时会被裁掉 —— 需要滚动请在内容里放 ScrollArea。
func WithScreenPopupHeight(h int32) ScreenPopupOption {
	return func(c *screenPopupConfig) {
		if h >= 0 {
			c.height = h
		}
	}
}

// WithScreenPopupOffset 设置卡片与工作区右下角的距离(默认 24)。
func WithScreenPopupOffset(d int32) ScreenPopupOption {
	return func(c *screenPopupConfig) {
		if d >= 0 {
			c.offset = d
		}
	}
}

// WithScreenPopupPadding 覆盖内容区四周内边距(默认 16)。传 0 表示内容自己贴边。
func WithScreenPopupPadding(p int32) ScreenPopupOption {
	return func(c *screenPopupConfig) {
		if p >= 0 {
			c.padding = p
		}
	}
}

// WithScreenPopupGap 设置内容区子项间距(默认 8)。
func WithScreenPopupGap(g int32) ScreenPopupOption {
	return func(c *screenPopupConfig) {
		if g >= 0 {
			c.gap = g
		}
	}
}

// WithScreenPopupDuration 设置显示后自动关闭的时长(默认 0 / 负数 = 不自动关闭)。
// 每次 Show 重新计时; Hide / Destroy 会取消计时。计时期间不因悬停暂停。
func WithScreenPopupDuration(d time.Duration) ScreenPopupOption {
	return func(c *screenPopupConfig) {
		if d > 0 {
			c.duration = d
		} else {
			c.duration = 0
		}
	}
}

// WithScreenPopupFocusable 设置窗口是否可接收焦点(默认 false = 完全不激活)。
// true 时 Show 仍不抢前台(与默认模式一样, 用户正在用的窗口/光标不受影响);
// 区别在于用户点击弹窗时会被系统正常激活, 输入框/文本域可用、ESC 收起。
// 失焦不会自动关闭(关不关仍由 X / Duration / 调用方决定), 保持常驻面板语义。
// false 时点击也只走鼠标消息, 窗口永远不会被激活。
func WithScreenPopupFocusable(focusable bool) ScreenPopupOption {
	return func(c *screenPopupConfig) { c.focusable = focusable }
}

// WithScreenPopupCloseButton 在卡片右上角加一个 X 关闭按钮(点击调用 Hide)。
// 需要标题时用 WithScreenPopupHeader(标题会自动排在同一行左侧)。
func WithScreenPopupCloseButton() ScreenPopupOption {
	return func(c *screenPopupConfig) { c.closeButton = true }
}

// WithScreenPopupHeader 在内容顶部加一行标题栏: 标题(Weight 宽) + 右上角 X。
// 标题永远与关闭按钮一起出现; 只要 X 不要标题就传空字符串(或直接用 CloseButton)。
func WithScreenPopupHeader(title string) ScreenPopupOption {
	return func(c *screenPopupConfig) {
		c.headerTitle = title
		c.closeButton = true
	}
}

// WithScreenPopupOnOpenChange 注册开关状态回调(Show/Hide 都会通知, 含 X 与自动关闭)。
func WithScreenPopupOnOpenChange(fn func(open bool)) ScreenPopupOption {
	return func(c *screenPopupConfig) { c.onOpenChange = fn }
}

// WithScreenPopupContent 在构造时填充内容(content 是纵向流式容器, 宽铺满、高自撑)。
func WithScreenPopupContent(build func(content int)) ScreenPopupOption {
	return func(c *screenPopupConfig) { c.build = build }
}

// NewScreenPopup 创建屏幕弹窗。hAnchor 是宿主窗口句柄, 只用来:
//   - 选择显示器(弹窗出现在该窗口所在显示器的右下角);
//   - 宿主销毁时自动销毁弹窗, 不挡住消息循环退出。
//
// 返回 nil 表示句柄无效或创建失败。
func NewScreenPopup(hAnchor int, opts ...ScreenPopupOption) *ScreenPopup {
	if hAnchor <= 0 || !xc.XC_IsHXCGUI(hAnchor, xcc.XC_WINDOW) {
		return nil
	}
	cfg := screenPopupConfig{
		width:   screenPopupDefaultWidth,
		offset:  screenPopupDefaultOffset,
		padding: screenPopupDefaultPadding,
		gap:     screenPopupDefaultGap,
	}
	for _, o := range opts {
		o(&cfg)
	}

	initH := cfg.height
	if initH <= 0 {
		initH = 1 // 自动高: 先建一个最小卡片, Show 时再按内容改
	}
	p := newPopoverPanelForWindow(cfg.width, initH, hAnchor, true, cfg.focusable)
	if p == nil {
		return nil
	}
	// 显隐都走淡入淡出: 退场要用进场那条 presence 过渡反向, 所以不能跳过进场。
	// 阴影是窗口级绘制, 不在元素 alpha 里 —— 内容与阴影浓度要一起淡, 否则结尾
	// 会剩下一圈明显的阴影。
	p.alwaysAnimate = true
	p.fadeShadow = true
	if cfg.focusable {
		// 可输入面板仍是"常驻面板": 点一下别的窗口就自动收起不是它的语义。
		p.keepOnBlur = true
		// 窗口可激活(不加 WS_EX_NOACTIVATE / 不拦 WM_MOUSEACTIVATE), 但 Show 与
		// 离屏量高都不主动拿前台 —— 只有用户点弹窗时, 系统才像普通窗口一样激活它。
		p.showWithoutActivate = true
	}
	content := p.LayoutContent(cfg.padding, cfg.gap)
	if content == nil {
		p.Destroy()
		return nil
	}
	s := &ScreenPopup{
		p:            p,
		hAnchor:      hAnchor,
		width:        cfg.width,
		height:       cfg.height,
		offset:       cfg.offset,
		padding:      cfg.padding,
		duration:     cfg.duration,
		onOpenChange: cfg.onOpenChange,
		anchorHook:   -1,
	}
	if cfg.closeButton || cfg.headerTitle != "" {
		s.buildHeader(content.Handle, cfg.headerTitle)
	}
	if cfg.build != nil {
		cfg.build(content.Handle)
	}
	s.watch()
	// 主题切换可能改变字体/折行, 高要重新量(只重绘会裁掉末行, 同 ScreenToast)。
	xui.OnThemeChangeFor(content.Handle, func(*xui.Theme) {
		s.fitContent()
	})
	return s
}

// buildHeader 在内容顶部建一行标题 + 右上角关闭按钮(标题可为空, 只留 X)。
func (s *ScreenPopup) buildHeader(parent int, title string) {
	row := xui.HStack(parent, 28, 8)
	lab := NewLabel(0, 0, 0, 28, title, row.Handle)
	lab.SetMedium().SetFontSize(14)
	xui.ItemWeight(lab, 1)
	b := NewButton(0, 0, 0, 28, "", row.Handle,
		WithVariant(ButtonGhost), WithSize(ButtonSizeIconSm), WithIcon(icons.IconX))
	xui.ItemWidth(b, 28)
	b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		s.Hide()
		return 0
	})
}

// watch 宿主窗口销毁时拆掉弹窗; 弹窗自己被外部销毁时把对象标记为失效。
func (s *ScreenPopup) watch() {
	if s == nil || s.p == nil {
		return
	}
	if w := window.NewByHandle(s.hAnchor); w != nil {
		s.anchorHook = w.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
			if message == wapi.WM_DESTROY {
				s.anchorGone = true
				s.destroyNow()
			}
			return 0
		})
	}
	s.p.win.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
		if message == wapi.WM_DESTROY {
			s.stopAutoHide()
			s.destroyed = true
			s.p = nil
		}
		return 0
	})
}

// scheduleAutoHide 按当前 duration 开始计时(Show 时调用)。
func (s *ScreenPopup) scheduleAutoHide() {
	s.stopAutoHide()
	if s == nil || s.destroyed || s.duration <= 0 {
		return
	}
	s.autoGen++
	gen := s.autoGen
	// app.AfterFunc 的回调会切回 UI 线程(与组件里其它延时同一条)。已经排队但随后
	// 被 Hide/重新 Show 取消的旧回调, 靠 gen 比对丢弃 —— 否则它会把新弹窗提前关掉。
	s.autoTimer = app.AfterFunc(s.duration, func() {
		if s.autoGen != gen || s.autoTimer == nil {
			return
		}
		s.autoTimer = nil
		s.Hide()
	})
}

// stopAutoHide 取消自动关闭计时(幂等)。
func (s *ScreenPopup) stopAutoHide() {
	if s == nil {
		return
	}
	s.autoGen++
	if s.autoTimer != nil {
		s.autoTimer.Stop()
		s.autoTimer = nil
	}
}

// notifyOpen 通知开关状态回调。
func (s *ScreenPopup) notifyOpen(open bool) {
	if s != nil && s.onOpenChange != nil {
		s.onOpenChange(open)
	}
}

// Content 返回内容根元素句柄(逃生口)。
func (s *ScreenPopup) Content() int {
	if s == nil || s.destroyed || s.p == nil || s.p.content == nil {
		return 0
	}
	return s.p.content.Handle
}

// Handle 返回弹窗 xcgui 窗口句柄(逃生口)。
func (s *ScreenPopup) Handle() int {
	if s == nil || s.destroyed || s.p == nil || s.p.win == nil {
		return 0
	}
	return s.p.win.Handle
}

// IsOpen 报告弹窗是否正在显示。
func (s *ScreenPopup) IsOpen() bool {
	return s != nil && !s.destroyed && s.p != nil && s.p.IsOpen()
}

// Show 按当前尺寸贴到工作区右下角并显示(不抢焦点)。
// 自动高度模式会在显示前按内容量一次高。
func (s *ScreenPopup) Show() *ScreenPopup {
	if s == nil || s.destroyed || s.p == nil || !s.p.alive() || s.p.IsOpen() {
		return s
	}
	s.applySize()

	waL, waT, waR, waB, ok := s.workArea()
	if !ok {
		return s
	}
	// 工作区放不下时收窄/收矮, 不让弹窗越出屏幕(内容用 ScrollArea 才能滚动)。
	s.clampTo(waL, waT, waR, waB)

	x := waR - s.offset - s.p.cw
	y := waB - s.offset - s.p.ch
	s.p.showAtHWND(s.hAnchor, x, y)
	if s.p.IsOpen() {
		s.scheduleAutoHide()
		s.notifyOpen(true)
	}
	return s
}

// workArea 返回锚点窗口所在显示器的工作区(逻辑像素)。
// 工作区是物理像素, 锚点窗口 DPI 换成逻辑(SetWindowPos 的 x/y 是物理, 宽高是逻辑, 见 DESIGN 119)。
func (s *ScreenPopup) workArea() (waL, waT, waR, waB int32, ok bool) {
	hWnd := xc.XWnd_GetHWND(s.hAnchor)
	if hWnd == 0 {
		return 0, 0, 0, 0, false
	}
	wa, has := xui.MonitorWorkArea(hWnd)
	if !has {
		return 0, 0, 0, 0, false
	}
	sc := s.scale()
	return int32(math.Round(float64(wa.Left) / sc)),
		int32(math.Round(float64(wa.Top) / sc)),
		int32(math.Round(float64(wa.Right) / sc)),
		int32(math.Round(float64(wa.Bottom) / sc)), true
}

// clampTo 把卡片尺寸夹进工作区(扣除 offset)。
func (s *ScreenPopup) clampTo(waL, waT, waR, waB int32) {
	if maxW := waR - waL - s.offset*2; maxW > 0 && s.p.cw > maxW {
		s.p.resize(maxW, s.p.ch)
	}
	if maxH := waB - waT - s.offset*2; maxH > 0 && s.p.ch > maxH {
		s.p.resize(s.p.cw, maxH)
	}
}

// Hide 隐藏弹窗(播放 100ms 淡出, 窗口保留复用; 逻辑状态立即变为关闭)。
func (s *ScreenPopup) Hide() *ScreenPopup {
	if s == nil || s.destroyed || s.p == nil {
		return s
	}
	wasOpen := s.p.IsOpen()
	s.stopAutoHide()
	s.p.HideWithFade(xui.PopupDuration)
	if wasOpen {
		s.notifyOpen(false)
	}
	return s
}

// Toggle 在显示/隐藏之间切换。
func (s *ScreenPopup) Toggle() *ScreenPopup {
	if s.IsOpen() {
		return s.Hide()
	}
	return s.Show()
}

// FitContent 重新按内容量高(自动高度模式)并贴回右下角; 打开中也立即生效, 不闪。
// 内容增删后调用; 固定高度模式不做事。
func (s *ScreenPopup) FitContent() *ScreenPopup {
	if s == nil || s.destroyed || s.p == nil {
		return s
	}
	s.fitContent()
	return s
}

// fitContent 是 FitContent 与主题切换的公共实现。
func (s *ScreenPopup) fitContent() {
	if s == nil || s.destroyed || s.p == nil || !s.p.alive() {
		return
	}
	if s.height > 0 { // 固定高度不随内容
		if s.p.IsOpen() {
			s.reposition()
		}
		return
	}
	if !s.p.IsOpen() {
		s.applySize() // 隐藏时离屏量高, 下次 Show 直接用
		return
	}
	// 打开中只重排内容, 不做离屏测量(测量会先把窗口挪到 -30000 再隐藏)。
	// ⚠️ 新增子项后只调 content.AdjustLayout 读到的还是旧高(实测 93), 要让父级
	// bodyLayout 也排一次才变成实际高度(121) —— 父级那次布局才会重新解析子项的
	// Auto 高。少了这一步 FitContent 改不动窗口高度。
	s.p.content.AdjustLayout()
	s.p.bodyLayout.AdjustLayout()
	h := s.p.content.GetHeight()
	if h < 1 {
		h = 1
	}
	s.p.resize(s.width, h+s.padding*2)
	if waL, waT, waR, waB, ok := s.workArea(); ok {
		s.clampTo(waL, waT, waR, waB)
	}
	s.reposition()
	s.p.win.Redraw(true)
}

// applySize 应用固定高度或按内容量高(只在隐藏时调用, 走离屏测量)。
func (s *ScreenPopup) applySize() {
	if s == nil || s.p == nil || !s.p.alive() {
		return
	}
	if s.height > 0 {
		s.p.resize(s.width, s.height)
		return
	}
	h := s.p.MeasureContent(s.padding)
	if h <= 0 { // 空内容也保留上下内边距
		s.p.resize(s.width, s.padding*2)
		return
	}
	s.p.resize(s.width, h+s.padding*2)
}

// reposition 把已显示的弹窗按当前尺寸钉回工作区右下角。
func (s *ScreenPopup) reposition() {
	if s == nil || s.destroyed || s.p == nil || !s.p.alive() || !s.p.IsOpen() {
		return
	}
	_, _, waR, waB, ok := s.workArea()
	if !ok {
		return
	}
	sc := s.scale()
	x := waR - s.offset - s.p.cw
	y := waB - s.offset - s.p.ch
	band := float64(popoverShadow)
	wx := int32((float64(x)-band)*sc + 0.5)
	wy := int32((float64(y)-band)*sc + 0.5)
	s.p.win.SetWindowPos(xcc.HWND_TOPMOST, wx, wy, 0, 0,
		xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE)
}

// scale 返回锚点窗口的 DPI 比例。
func (s *ScreenPopup) scale() float64 {
	dpi := xc.XWnd_GetDPI(s.hAnchor)
	if dpi <= 0 {
		dpi = 96
	}
	return float64(dpi) / 96
}

// Destroy 立即销毁弹窗(不播退场动画)。
func (s *ScreenPopup) Destroy() {
	s.destroyNow()
}

// destroyNow 拆掉弹窗窗口并清掉挂在自己窗口上的过渡; 幂等。
func (s *ScreenPopup) destroyNow() {
	if s == nil || s.destroyed {
		return
	}
	s.stopAutoHide()
	s.destroyed = true
	// 宿主还在时反注册监听: 反复 New/Destroy 弹窗不能在宿主事件表里越积越多。
	// 宿主自己正在销毁时跳过 —— 它随后会由事件总线的 WM_NCDESTROY 统一清理。
	if !s.anchorGone && s.anchorHook >= 0 && xc.XC_IsHXCGUI(s.hAnchor, xcc.XC_WINDOW) {
		xc.WndEventBus.RemoveCallback(s.hAnchor, xcc.XWM_WINDPROC, s.anchorHook)
	}
	s.anchorHook = -1
	if p := s.p; p != nil && p.alive() {
		xui.DropTransitions(p.win.Handle)
		p.win.DestroyWindow()
	}
	s.p = nil
}
