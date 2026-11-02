package ui

import (
	"math"
	"time"

	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 屏幕通知是贴在**显示器工作区**右下角的独立置顶小窗(Windows 通知的形态):
// 每条通知一个异型透明窗, 不占任务栏、不抢焦点; 卡片内容与 Toaster 共用同一套
// 排版与绘制(toast.go), 所以主题/richColors/动作/关闭/转圈的表现一致。
//
// 与 Toaster 的区别只在"载体": Toaster 是主窗口内的覆盖元素, 通知跟着窗口走;
// 这里要贴在屏幕角落, 应用最小化/隐藏时也能看到, 也只能用独立窗口。
const (
	// screenToastPad 是小窗在卡片四周留的空白: 进退场滑动的 12/24 与投影 6
	// 都要落在这块空间里。取 24 与默认 offset 相同, 窗口正好贴住工作区边缘。
	screenToastPad int32 = 24
	// screenToastMinWidth 是最小卡片宽(与 Toaster 的 toastMinWidth 一致)。
	screenToastMinWidth = toastMinWidth
)

// ScreenToaster 在屏幕右下角显示独立置顶通知。
//
// 与 Toaster 的 API 基本一致: Show(ToastOptions) / Update / Close / Promise,
// 但通知窗口的生命周期必须跟一个"宿主窗口"绑定 —— hAnchor 用来:
//   - 选择显示器(通知出现在该窗口所在显示器的右下角);
//   - 驱动进出场动画(异型小窗自己的定时器只有约 10fps, 见 DESIGN 47);
//   - 宿主销毁时自动销毁所有通知小窗, 不挡住消息循环退出(见 DESIGN 77)。
//
// 用法:
//
//	sc := ui.NewScreenToaster(w.Handle)
//	sc.Success("已保存", "文件已写入磁盘")
//
// 必须是 UI 线程调用。
type ScreenToaster struct {
	hAnchor     int
	width       int32
	offset      int32
	gap         int32
	duration    time.Duration
	maxVisible  int
	richColors  bool
	closeButton bool

	paused    bool
	items     []*ScreenToast
	live      map[*ScreenToast]bool // 含正在退场、已不在 items 里的条目
	destroyed bool
}

// ScreenToastOption 是 ScreenToaster 的配置项。
type ScreenToastOption func(*ScreenToaster)

// WithScreenToastWidth 覆盖卡片宽度(默认 356)。
func WithScreenToastWidth(w int32) ScreenToastOption {
	return func(s *ScreenToaster) {
		if w >= screenToastMinWidth {
			s.width = w
		}
	}
}

// WithScreenToastOffset 设置卡片与工作区右下角的距离(默认 24)。
func WithScreenToastOffset(d int32) ScreenToastOption {
	return func(s *ScreenToaster) {
		if d >= 0 {
			s.offset = d
		}
	}
}

// WithScreenToastGap 设置卡片之间的间距(默认 10)。
func WithScreenToastGap(g int32) ScreenToastOption {
	return func(s *ScreenToaster) {
		if g >= 0 {
			s.gap = g
		}
	}
}

// WithScreenToastDuration 覆盖默认自动关闭时长(默认 4s)。
func WithScreenToastDuration(d time.Duration) ScreenToastOption {
	return func(s *ScreenToaster) { s.duration = d }
}

// WithScreenToastMaxVisible 设置同屏最多显示几条(默认 3); <= 0 表示不限制。
func WithScreenToastMaxVisible(n int) ScreenToastOption {
	return func(s *ScreenToaster) { s.maxVisible = n }
}

// WithScreenToastRichColors 按类型给整卡上色(同 Toaster 的 richColors)。
func WithScreenToastRichColors() ScreenToastOption {
	return func(s *ScreenToaster) { s.richColors = true }
}

// WithScreenToastCloseButton 让所有通知都显示右上角关闭按钮。
func WithScreenToastCloseButton() ScreenToastOption {
	return func(s *ScreenToaster) { s.closeButton = true }
}

// NewScreenToaster 创建屏幕通知管理器。hAnchor 是宿主窗口句柄(取它所在显示器,
// 并在其销毁时清理通知小窗)。返回 nil 表示句柄无效。
func NewScreenToaster(hAnchor int, opts ...ScreenToastOption) *ScreenToaster {
	if hAnchor <= 0 || !xc.XC_IsHXCGUI(hAnchor, xcc.XC_WINDOW) {
		return nil
	}
	s := &ScreenToaster{
		hAnchor:    hAnchor,
		width:      toastDefaultWidth,
		offset:     toastDefaultOffset,
		gap:        toastDefaultGap,
		duration:   toastDefaultDuration,
		maxVisible: toastDefaultVisible,
		live:       map[*ScreenToast]bool{},
	}
	for _, o := range opts {
		o(s)
	}
	s.watchAnchor()
	return s
}

// watchAnchor 在宿主窗口销毁时清掉全部通知小窗。通知窗是独立顶层窗,
// 不清理会让消息循环等它(进程关不掉, 见 DESIGN 77)。
func (s *ScreenToaster) watchAnchor() {
	w := window.NewByHandle(s.hAnchor)
	if w == nil {
		return
	}
	w.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
		if message == wapi.WM_DESTROY {
			s.destroyAll()
		}
		return 0
	})
}

// Show 显示一条屏幕通知并返回句柄(用于 Update / Close)。
func (s *ScreenToaster) Show(o ToastOptions) *ScreenToast {
	if s == nil || s.destroyed || !xc.XC_IsHXCGUI(s.hAnchor, xcc.XC_WINDOW) {
		return nil
	}
	o = s.prepare(o)

	// 超出可见条数: 立即移除最旧的一条(与 Toaster 一致, 不播退场)。
	for s.maxVisible > 0 && len(s.items) >= s.maxVisible {
		old := s.items[0]
		s.items = s.items[1:]
		old.drop()
	}

	// 先按同一套 measure 量出卡片高, 窗口一次建到正确尺寸, 避免显示后改高度闪一下。
	probe := &Toast{opts: o}
	probe.measure(s.width)
	cardW := s.width
	cardH := probe.h
	if cardH <= 0 {
		cardH = toastTitleH
	}
	winW := cardW + 2*screenToastPad
	winH := cardH + 2*screenToastPad

	win := window.NewEx(xcc.WS_EX_TOOLWINDOW|xcc.WS_EX_NOACTIVATE, 0, "", 0, 0,
		winW, winH, "", 0, xcc.Window_Style_Nothing)
	if win == nil {
		return nil
	}
	win.SetBorderSize(0, 0, 0, 0)
	win.SetTransparentType(xcc.Window_Transparent_Shaped)
	win.SetTransparentAlpha(255)
	// 小窗没有主题底色: 元素 alpha 从 0 进场时, 白色默认底会从卡片下面透出来。
	if bkm := xc.XWnd_GetBkManager(win.Handle); bkm != 0 {
		xc.XBkM_Clear(bkm)
		xc.XBkM_AddFill(bkm, xcc.CombinedState_Window_State_Flag_Leave, 0, 0)
	}
	// 屏幕外预显示: 与弹层同一条(首帧不能带旧 surface; 这里主要是让元素 alpha=0 的首帧落盘)。
	win.SetWindowPos(xcc.HWND_TOPMOST, -30000, -30000, 0, 0,
		xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE|xcc.SWP_SHOWWINDOW)
	wapi.ShowWindow(win.GetHWND(), xcc.SW_SHOWNOACTIVATE)

	rec := &ScreenToast{
		s:     s,
		win:   win,
		cardW: cardW,
		cardH: cardH,
		winW:  winW,
		winH:  winH,
	}

	opts := []ToasterOption{
		WithToastWidth(cardW),
		WithToastOffset(screenToastPad),
		WithToastGap(0),
		WithToastDuration(s.duration),
		WithToastMaxVisible(0),
		withToastAnimOwner(s.hAnchor),
	}
	if s.richColors {
		opts = append(opts, WithToastRichColors())
	}
	t := NewToaster(win.Handle, opts...)
	if t == nil {
		win.DestroyWindow()
		return nil
	}
	rec.t = t

	// 悬停暂停是整组语义: 指针落在任意一条卡片上时, 全部倒计时暂停。
	win.AddEvent_MouseMove(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		if win.GetStayEle() != 0 {
			s.setPaused(true)
		}
		return 0
	})
	win.AddEvent_MouseLeave(func(hWindow int, pbHandled *bool) int {
		s.setPaused(false)
		return 0
	})
	// 窗口被外部销毁(或随进程清理)时把记录摘掉, 防止堆叠里留悬空窗口。
	win.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
		if message == wapi.WM_DESTROY {
			rec.onWindowDestroy()
		}
		return 0
	})

	it := t.Show(o)
	if it == nil {
		win.DestroyWindow()
		return nil
	}
	rec.it = it
	it.immediateHover = true
	it.spinOwner = s.hAnchor
	// 首帧绘制时 spinOwner 还没设上, 可能在异型小窗上登记了 Spin; 这里把它停掉,
	// 之后绘制改走 SpinDetached(定时器挂宿主窗口)。
	xui.Spin(it.Handle(), false)
	it.onClosing = func() {
		// 开始退场就从堆叠里摘掉: 其余卡片立即归位(同 Toaster 的 removeItem)。
		rec.closed = true
		if !s.destroyed {
			s.remove(rec)
		}
	}
	it.onClosed = rec.cleanup

	s.live[rec] = true
	s.items = append(s.items, rec)
	s.fit(rec)
	s.reposition()

	// 主题切换: Toast 自己的回调已经重算过高度, 这里按新高度改小窗并重排。
	xui.OnThemeChangeFor(it.Handle(), func(*xui.Theme) {
		if rec.closed || rec.destroyed {
			return
		}
		s.fit(rec)
		s.reposition()
	})
	return rec
}

// prepare 补齐默认时长与整组关闭按钮设置。
func (s *ScreenToaster) prepare(o ToastOptions) ToastOptions {
	if o.Duration == 0 {
		o.Duration = s.duration
	}
	if o.Duration < 0 {
		o.Duration = -1
	}
	if s.closeButton {
		o.CloseButton = true
	}
	return o
}

// fit 按当前通知的实际高度调整小窗高度, 让 Toaster 的 Body 与卡片一致。
func (s *ScreenToaster) fit(rec *ScreenToast) {
	if s == nil || rec == nil || rec.destroyed || rec.closed || rec.win == nil || rec.it == nil {
		return
	}
	h := rec.it.h
	if h <= 0 || h == rec.cardH {
		return
	}
	rec.cardH = h
	rec.winH = h + 2*screenToastPad
	rec.win.SetWindowPos(xcc.HWND_TOPMOST, 0, 0, rec.winW, rec.winH,
		xcc.SWP_NOMOVE|xcc.SWP_NOACTIVATE|xcc.SWP_NOZORDER)
	if rec.t != nil {
		rec.t.layout()
	}
}

// reposition 把所有活动通知按"新的在最下"钉到工作区右下角。
//
// 小窗尺寸是逻辑像素(SetWindowPos 的宽高参数由 xcgui 做 DPI 缩放), 位置是
// 物理像素(见 DESIGN 119), 所以窗口位置要自己乘 DPI 比例。
func (s *ScreenToaster) reposition() {
	if s == nil || s.destroyed || len(s.items) == 0 {
		return
	}
	wa, ok := s.workArea()
	if !ok {
		return
	}
	sc := s.scale()
	waR := int32(math.Round(float64(wa.Right) / sc))
	waB := int32(math.Round(float64(wa.Bottom) / sc))

	y := waB - s.offset
	for i := len(s.items) - 1; i >= 0; i-- {
		rec := s.items[i]
		if rec.destroyed || rec.win == nil {
			continue
		}
		cardLeft := waR - s.offset - rec.cardW
		cardTop := y - rec.cardH
		winX := cardLeft - screenToastPad
		winY := cardTop - screenToastPad
		px := int32(math.Round(float64(winX) * sc))
		py := int32(math.Round(float64(winY) * sc))
		rec.win.SetWindowPos(xcc.HWND_TOPMOST, px, py, 0, 0,
			xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE)
		y = cardTop - s.gap
	}
}

// workArea 返回宿主窗口所在显示器的工作区(物理像素), 取不到时退回主显示器。
func (s *ScreenToaster) workArea() (xc.RECT, bool) {
	var hMon uintptr
	if s.hAnchor != 0 && xc.XC_IsHXCGUI(s.hAnchor, xcc.XC_WINDOW) {
		if hwnd := xc.XWnd_GetHWND(s.hAnchor); hwnd != 0 {
			hMon = wapi.MonitorFromWindow(hwnd, wapi.MONITOR_DEFAULTTONEAREST)
		}
	}
	if hMon == 0 {
		hMon = wapi.MonitorFromWindow(wapi.GetDesktopWindow(), wapi.MONITOR_DEFAULTTOPRIMARY)
	}
	if hMon == 0 {
		return xc.RECT{}, false
	}
	var mi wapi.MONITORINFO
	if !wapi.GetMonitorInfo(hMon, &mi) {
		return xc.RECT{}, false
	}
	return xc.RECT{
		Left:   mi.RcWork.Left,
		Top:    mi.RcWork.Top,
		Right:  mi.RcWork.Right,
		Bottom: mi.RcWork.Bottom,
	}, true
}

// scale 返回宿主窗口的 DPI 比例; 通知与宿主在同一块显示器上。
func (s *ScreenToaster) scale() float64 {
	dpi := xc.XWnd_GetDPI(s.hAnchor)
	if dpi <= 0 {
		dpi = 96
	}
	return float64(dpi) / 96
}

// setPaused 切换整组倒计时暂停(悬停任意一条时全部暂停, 同 sonner)。
func (s *ScreenToaster) setPaused(p bool) {
	if s == nil || s.paused == p {
		return
	}
	s.paused = p
	for _, rec := range s.items {
		if rec.t != nil {
			rec.t.setPaused(p)
		}
	}
}

// remove 从堆叠里摘掉一条并让其余卡片归位(幂等)。
func (s *ScreenToaster) remove(rec *ScreenToast) {
	for i, v := range s.items {
		if v == rec {
			s.items = append(s.items[:i], s.items[i+1:]...)
			break
		}
	}
	if !s.destroyed {
		s.reposition()
	}
}

// destroyAll 销毁全部通知小窗(宿主窗口销毁时调用, 不触发用户回调)。
func (s *ScreenToaster) destroyAll() {
	if s == nil || s.destroyed {
		return
	}
	s.destroyed = true
	xui.DropTransitions(s.hAnchor)
	for rec := range s.live {
		rec.destroyWindowNow()
	}
	s.live = map[*ScreenToast]bool{}
	s.items = nil
}

// Count 返回当前堆叠中的通知条数(不含正在退场的)。
func (s *ScreenToaster) Count() int {
	if s == nil {
		return 0
	}
	return len(s.items)
}

// Clear 立即移除全部通知(不播退场动画)。
func (s *ScreenToaster) Clear() {
	if s == nil {
		return
	}
	items := s.items
	s.items = nil
	for _, rec := range items {
		rec.drop()
	}
}

// Default / Success / Error / Warning / Info 是常用类型的快捷入口。
func (s *ScreenToaster) Default(title, description string) *ScreenToast {
	return s.Show(ToastOptions{Title: title, Description: description})
}

func (s *ScreenToaster) Success(title, description string) *ScreenToast {
	return s.Show(ToastOptions{Title: title, Description: description, Variant: ToastSuccess})
}

func (s *ScreenToaster) Error(title, description string) *ScreenToast {
	return s.Show(ToastOptions{Title: title, Description: description, Variant: ToastError})
}

func (s *ScreenToaster) Warning(title, description string) *ScreenToast {
	return s.Show(ToastOptions{Title: title, Description: description, Variant: ToastWarning})
}

func (s *ScreenToaster) Info(title, description string) *ScreenToast {
	return s.Show(ToastOptions{Title: title, Description: description, Variant: ToastInfo})
}

// Loading 显示一条转圈的加载通知(默认常驻, 完成后用 Update 改成结果态)。
func (s *ScreenToaster) Loading(title, description string) *ScreenToast {
	return s.Show(ToastOptions{
		Title:       title,
		Description: description,
		Variant:     ToastLoading,
		Duration:    -1,
	})
}

// Promise 执行 fn 并在期间显示一条 loading 通知, 结束后把同一条改成 success/error。
// fn 在独立 goroutine 里执行, 结果经 xc.Auto 回 UI 线程更新。
func (s *ScreenToaster) Promise(fn func() (string, error), p ToastPromise) *ScreenToast {
	if s == nil || fn == nil {
		return nil
	}
	loading := p.Loading
	loading.Variant = ToastLoading
	loading.Duration = -1
	it := s.Show(loading)
	if it == nil {
		return nil
	}
	go func() {
		result, err := promiseCall(fn)
		xc.Auto(func() {
			if it.closed || it.destroyed {
				return
			}
			if err != nil {
				o := ToastOptions{Title: err.Error(), Variant: ToastError}
				if p.Error != nil {
					o = p.Error(err)
					if o.Variant == ToastDefault {
						o.Variant = ToastError
					}
				}
				it.Update(o)
				return
			}
			o := ToastOptions{Title: result, Variant: ToastSuccess}
			if p.Success != nil {
				o = p.Success(result)
				if o.Variant == ToastDefault {
					o.Variant = ToastSuccess
				}
			}
			it.Update(o)
		})
	}()
	return it
}

// ScreenToast 是一条已显示的屏幕通知(句柄用于 Update / Close)。
type ScreenToast struct {
	s     *ScreenToaster
	win   *window.Window
	t     *Toaster
	it    *Toast
	cardW int32
	cardH int32
	winW  int32
	winH  int32

	closed    bool
	destroyed bool
}

// Update 更新通知内容并重排(高度可能变化)。
func (st *ScreenToast) Update(o ToastOptions) *ScreenToast {
	if st == nil || st.closed || st.destroyed || st.it == nil || st.s == nil {
		return st
	}
	o = st.s.prepare(o)
	st.it.Update(o)
	st.s.fit(st)
	st.s.reposition()
	return st
}

// Close 关闭通知(淡出 + 侧滑, 完成后销毁小窗)。
func (st *ScreenToast) Close() {
	if st == nil || st.closed || st.destroyed || st.it == nil {
		return
	}
	st.it.Close()
}

// Handle 返回通知元素句柄(逃生口)。
func (st *ScreenToast) Handle() int {
	if st == nil || st.it == nil {
		return 0
	}
	return st.it.Handle()
}

// drop 立即销毁(不播退场动画), 用于超量挤出 / Clear。
func (st *ScreenToast) drop() {
	if st == nil || st.destroyed {
		return
	}
	if st.it != nil {
		st.it.destroy()
		return
	}
	st.cleanup()
}

// cleanup 是 Toast 销毁钩子: 无论用户怎么关(倒计时/关闭钮/动作), 都回收小窗。
func (st *ScreenToast) cleanup() {
	if st == nil || st.destroyed {
		return
	}
	st.destroyed = true
	st.closed = true
	if st.s != nil {
		delete(st.s.live, st)
	}
	if st.win != nil {
		w := st.win
		st.win = nil
		if xc.XC_IsHXCGUI(w.Handle, xcc.XC_WINDOW) {
			w.DestroyWindow()
		}
	}
	st.t = nil
	st.it = nil
	if st.s != nil && !st.s.destroyed {
		st.s.remove(st)
	}
}

// onWindowDestroy 处理小窗被外部销毁(不是 cleanup 主动销毁): 只摘记录, 不触发回调。
func (st *ScreenToast) onWindowDestroy() {
	if st == nil || st.destroyed {
		return
	}
	st.destroyed = true
	st.closed = true
	// 走的是"小窗被外部销毁"而不是 Toast.destroy: 那条 presence 条目还挂在宿主
	// 窗口上(闭包捕获本条通知), 宿主长命就只增不减, 这里补一次清场。
	if st.it != nil && st.t != nil {
		hOwner := st.t.hWnd
		if st.t.animOwner != 0 {
			hOwner = st.t.animOwner
		}
		xui.DropPresence(hOwner, st.it.key)
	}
	if st.s != nil {
		delete(st.s.live, st)
	}
	st.win = nil
	st.t = nil
	st.it = nil
	if st.s != nil && !st.s.destroyed {
		st.s.remove(st)
	}
}

// destroyWindowNow 在宿主窗口销毁时直接拆小窗, 不走 Toast 回调。
func (st *ScreenToast) destroyWindowNow() {
	if st == nil || st.destroyed {
		return
	}
	st.destroyed = true
	st.closed = true
	if st.win != nil {
		w := st.win
		st.win = nil
		if xc.XC_IsHXCGUI(w.Handle, xcc.XC_WINDOW) {
			w.DestroyWindow()
		}
	}
	st.t = nil
	st.it = nil
}
