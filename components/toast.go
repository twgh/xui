package ui

import (
	"fmt"
	"math"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 通知卡的几何常量(对齐 sonner 的默认值, 单位逻辑像素):
//   - 卡片宽 356(--width)、右下角距窗口内容区边缘 24、卡片间距 10;
//   - 内边距 16、图标 20x20(与文字间距 10);
//   - 标题 13px / 行高 18, 描述 13px(可折行, 行距 2);
//   - 动作按钮高 26、左右内边距 10; 关闭按钮 22x22。
const (
	toastPad        int32 = 16
	toastIconSize   int32 = 20
	toastIconGap    int32 = 10
	toastTitleH     int32 = 18
	toastDescGap    int32 = 2
	toastActionH    int32 = 26
	toastActionPadX int32 = 10
	toastActionGap  int32 = 10
	toastCloseSize  int32 = 22
	// toastShadow 是卡片外侧的投影留白: 元素比卡片四周各大一圈, 投影画在这圈里。
	// 元素绘制被裁剪在自身客户区, 不留白的话投影会被切掉。
	toastShadow int32 = 6
	// toastSlide 是进场时从下方(或上方)滑入的距离; 退场向侧边滑出。
	toastSlide    int32 = 12
	toastSlideOut int32 = 24

	toastDefaultWidth   int32 = 356
	toastDefaultOffset  int32 = 24
	toastDefaultGap     int32 = 10
	toastMinWidth       int32 = 160
	toastDefaultVisible       = 3
)

// 时间参数(sonner: 默认 4000ms 自动关闭, 进出场 200ms / 400ms)。
const (
	toastDefaultDuration = 4 * time.Second
	toastEnterDuration   = 200 * time.Millisecond
	toastExitDuration    = 180 * time.Millisecond
	// toastTickID 是倒计时占用的炫彩定时器 ID(与运行时过渡的 0x7875、Tree 的 0x7874 避开)。
	toastTickID       uint32 = 0x7876
	toastTickInterval        = 100 * time.Millisecond
)

// ToastVariant 是通知的语义类型(对应 sonner 的 toast / toast.success / toast.error ...)。
// 默认只影响图标与图标颜色, 卡片底色仍是 bg-popover —— 与 shadcn 的 sonner 一致
// (不传 richColors 时 sonner 也不给类型上底色); 开启 richColors 后按类型给整卡上色。
type ToastVariant int

const (
	ToastDefault ToastVariant = iota // 无图标
	ToastSuccess                     // 绿色圆勾
	ToastError                       // 红色圆叉
	ToastWarning                     // 橙色三角叹号
	ToastInfo                        // 蓝色信息
	// ToastLoading 是转圈图标(sonner 的 toast.loading / toast.promise 的进行中态)。
	// 相位由运行时 xui.Spin 驱动, 不需要调用方起定时器; 用 Toaster.Loading /
	// Toaster.Promise 创建时会自动设为常驻(不自动关闭)。
	ToastLoading
)

// ToastPosition 是通知的堆叠位置(sonner 的 position)。
type ToastPosition int

const (
	ToastBottomRight ToastPosition = iota // 默认: 右下角, 新的在最下
	ToastBottomLeft
	ToastTopRight
	ToastTopLeft
	ToastTopCenter    // 顶部居中(sonner position="top-center")
	ToastBottomCenter // 底部居中
)

// ToastOptions 描述一条通知。
type ToastOptions struct {
	Title       string
	Description string
	Variant     ToastVariant

	// Icon 覆盖按 Variant 取的内置图标(path 数据或整段 SVG, 见 xui.Painter.Icon)。
	// 传空串 = 用变体图标(ToastDefault 没有图标)。
	Icon string
	// IconColor 覆盖图标颜色; xui.None = 按变体取语义色(默认)。
	IconColor xui.Color

	// Duration 是自动关闭时长: 0 = Toaster 的默认值(4s); 负数 = 常驻, 不自动关闭。
	Duration time.Duration

	// Action 非空时在右侧显示一个动作按钮(如"撤销"), 点击调用 OnAction,
	// 默认点击后关闭通知; KeepOpen 可让通知继续留着。
	Action   string
	OnAction func()
	KeepOpen bool

	// CloseButton 显示右上角关闭按钮(默认不显示, 同 shadcn 的 sonner)。
	CloseButton bool
	// RichColors 本条通知按类型给整卡上底色/描边/文字色(sonner 的 richColors)。
	// Toaster 用 WithToastRichColors 全局开启后本条也生效; 默认 false = 只有图标带语义色。
	RichColors bool
	// OnClose 在任何方式关闭时回调一次(最宽的口子, 相当于 sonner 的 onDismiss + onAutoClose)。
	OnClose func()
	// OnAutoClose 只在倒计时到点自动关闭时回调(sonner 的 onAutoClose)。
	OnAutoClose func()
	// OnDismiss 在非自动关闭时回调: 关闭按钮、动作按钮、Toast.Close()、Toaster.Clear()
	// 与超出可见条数被挤掉(sonner 的 onDismiss)。
	OnDismiss func()
}

// ToasterOption 是 Toaster 的配置项。
type ToasterOption func(*Toaster)

// WithToastPosition 设置堆叠位置(默认右下角)。
func WithToastPosition(p ToastPosition) ToasterOption {
	return func(t *Toaster) { t.position = p }
}

// WithToastWidth 覆盖卡片宽度(默认 356, 会被窗口宽度夹小)。
func WithToastWidth(w int32) ToasterOption {
	return func(t *Toaster) {
		if w > 0 {
			t.width = w
		}
	}
}

// WithToastOffset 设置卡片与窗口内容区边缘的距离(默认 24)。
func WithToastOffset(d int32) ToasterOption {
	return func(t *Toaster) {
		if d >= 0 {
			t.offset = d
		}
	}
}

// WithToastGap 设置卡片之间的间距(默认 10)。
func WithToastGap(g int32) ToasterOption {
	return func(t *Toaster) {
		if g >= 0 {
			t.gap = g
		}
	}
}

// WithToastDuration 覆盖默认自动关闭时长(默认 4s)。
func WithToastDuration(d time.Duration) ToasterOption {
	return func(t *Toaster) { t.duration = d }
}

// WithToastMaxVisible 设置同屏最多显示几条(默认 3), 超出时最旧的立即移除。
// 传 <= 0 表示不限制。
func WithToastMaxVisible(n int) ToasterOption {
	return func(t *Toaster) { t.maxVisible = n }
}

// WithToastRichColors 按类型给整卡上底色/描边/文字色(sonner 的 richColors)。
// 默认关闭 = 只有图标带语义色、卡面统一 bg-popover; 也可以只给单条开(ToastOptions.RichColors)。
func WithToastRichColors() ToasterOption {
	return func(t *Toaster) { t.richColors = true }
}

// WithToastCloseButton 让所有通知都显示右上角关闭按钮(sonner 的 closeButton 属性)。
// 单条的 ToastOptions.CloseButton 仍可单独打开; 开了这个开关后无法单条关掉(与 sonner 一致)。
func WithToastCloseButton() ToasterOption {
	return func(t *Toaster) { t.closeButton = true }
}

// ToastPromise 描述一条 Promise 通知在三个阶段显示的内容(对应 sonner 的
// toast.promise(fn, {loading, success, error}))。
type ToastPromise struct {
	// Loading 是进行中的文案; 未指定 Variant 时用 ToastLoading(转圈图标),
	// 未指定 Duration 时常驻。
	Loading ToastOptions
	// Success 按 fn 的返回值生成成功文案; 传 nil 时显示"操作成功"。
	Success func(result string) ToastOptions
	// Error 按 fn 的错误生成失败文案; 传 nil 时显示错误文本本身。
	Error func(err error) ToastOptions
}

// Toaster 是挂在窗口上的通知管理器: 每条通知是一个铺在窗口之上的自绘元素,
// 缺省堆叠在内容区右下角(可设六种位置, 见 ToastPosition)。
//
// 为什么不做成独立置顶小窗(像 Tooltip): 通知要贴着主窗口的右下角, 跟随窗口移动 /
// 缩放 / 最小化; 用窗口内覆盖元素由框架负责这些, 也不用处理"点了别的窗口要不要消失"。
// 卡片是**单个元素整体自绘**(图标/标题/描述/动作/关闭按钮都画在它里面), 没有子元素
// —— 悬停事件不会被内部控件打断, 命中判定也就是一次矩形比较。
//
// 交互语义对齐 sonner:
//   - 默认右下角堆叠, 新的在最下, 旧的被顶上去; 超出 maxVisible 的最旧一条被移除;
//   - 悬停任意一条通知时**暂停全部倒计时**, 鼠标离开后继续(sonner 的 pauseTimers);
//   - 自动关闭默认 4s; 关闭按钮默认隐藏; 动作按钮点击后关闭;
//   - 进场: 淡入 + 从堆叠方向滑入 12px(200ms); 退场: 淡出 + 向侧边滑出 24px(180ms)。
type Toaster struct {
	hWnd int
	win  *window.Window

	position   ToastPosition
	width      int32
	offset     int32
	gap        int32
	duration    time.Duration
	maxVisible  int
	richColors  bool // 整组按类型上色(sonner 的 richColors); 单条可用 ToastOptions.RichColors 覆盖开启
	closeButton bool // 整组显示右上角关闭按钮(sonner 的 closeButton 属性)

	paused bool // 鼠标停在本组任一条通知上: 全部倒计时暂停
	items  []*Toast
}

// Toast 是一条已显示的通知(句柄用于 Update / Close)。
type Toast struct {
	t   *Toaster
	ele *widget.Element

	opts ToastOptions
	key  int32 // 本条的 presence 键(窗口级唯一, 见 xui.NextPresenceKey)

	// 卡片尺寸(不含投影留白)与在窗口中的静止位置(元素左上角不含留白)。
	w, h         int32
	baseX, baseY int32
	geom         toastGeom

	closed     bool // 已开始退场(或已销毁)
	entered    bool // 进场动画已完成
	destroyed  bool
	autoClosed bool // 关闭原因是倒计时到点(区分 OnAutoClose / OnDismiss)

	paused   bool // 鼠标悬停中: 暂停倒计时
	remain   time.Duration
	hAction  bool // 动作按钮悬停
	hClose   bool // 关闭按钮悬停
	onThemeC func(*xui.Theme)
}

// toastGeom 是卡片内部的排版结果(卡片局部坐标, 不含投影留白)。
type toastGeom struct {
	icon   xc.RECT
	title  xc.RECT
	desc   xc.RECT
	action xc.RECT
	close  xc.RECT
}

// NewToaster 创建通知管理器。hWnd 为宿主窗口句柄(通知覆盖整个窗口内容区)。
func NewToaster(hWnd int, opts ...ToasterOption) *Toaster {
	win := app.NewWindowByHandle(hWnd)
	if win == nil {
		return nil
	}
	t := &Toaster{
		hWnd:       hWnd,
		win:        win,
		width:      toastDefaultWidth,
		offset:     toastDefaultOffset,
		gap:        toastDefaultGap,
		duration:   toastDefaultDuration,
		maxVisible: toastDefaultVisible,
	}
	for _, o := range opts {
		o(t)
	}
	if t.duration == 0 {
		t.duration = toastDefaultDuration
	}
	// 窗口尺寸变化时重排(卡片宽度与堆叠位置都随内容区走); layout() 里同时按新
	// Body 矩形重钉布局项位置 —— 否则缩放后的窗口级布局趟会拿旧 inset 把元素拉伸
	// 变形(布局项置位置是四边 inset, 见 pin 注释)。
	win.AddEvent_Size(func(hWindow int, nFlags uint, pPt *xc.SIZE, pbHandled *bool) int {
		t.layout()
		return 0
	})
	// 悬停暂停: 走**窗口级** MouseMove + GetStayEle, 不用元素自己的 MouseStay/MouseLeave
	// —— 元素级 Stay/Leave 在 xcgui 里不可靠(多行 Edit 收不到 leave、原生 List 不把悬停
	// 路由给滚动条, 见 DESIGN 第八节 21), 而且这里要的是"鼠标在这组通知的任意一条上"
	// 这个整组语义, 窗口级判定一次就够。
	win.AddEvent_MouseMove(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		t.syncPaused(win.GetStayEle())
		return 0
	})
	win.AddEvent_MouseLeave(func(hWindow int, pbHandled *bool) int {
		t.setPaused(false)
		return 0
	})
	return t
}

// syncPaused 按窗口当前停留元素更新整组暂停态: 停在任一条通知上就暂停全部倒计时。
func (t *Toaster) syncPaused(hStay int) {
	on := false
	if hStay != 0 {
		for _, it := range t.items {
			if it.ele != nil && it.ele.Handle == hStay {
				on = true
				break
			}
		}
	}
	t.setPaused(on)
}

// setPaused 切换整组倒计时暂停。
func (t *Toaster) setPaused(p bool) {
	if t.paused == p {
		return
	}
	t.paused = p
	for _, it := range t.items {
		it.paused = p
	}
}

// Show 显示一条通知并返回它的句柄(可用于 Update / Close)。
func (t *Toaster) Show(o ToastOptions) *Toast {
	if t == nil || t.win == nil {
		return nil
	}
	if o.Duration == 0 {
		o.Duration = t.duration
	}
	if o.Duration < 0 {
		o.Duration = -1
	}
	if t.closeButton {
		o.CloseButton = true
	}
	// 超出可见数量: 立即移除最旧的那条(无退场动画, 避免堆积)。
	for t.maxVisible > 0 && len(t.items) >= t.maxVisible {
		old := t.items[0]
		t.items = t.items[1:]
		old.destroy()
	}

	ele := widget.NewElement(0, 0, 10, 10, t.hWnd)
	if ele == nil {
		return nil
	}
	ele.EnableBkTransparent(true)
	ele.EnableFocus(false)       // 点卡片不产生原生焦点框
	ele.EnableDrawBorder(false)  // 原生边框灭掉(卡片外观全部自绘)
	ele.EnableSwitchFocus(false) // 通知不进 Tab 遍历
	it := &Toast{t: t, ele: ele, opts: o, key: xui.NextPresenceKey(), paused: t.paused}
	it.remain = o.Duration
	ele.AddEvent_Paint(it.onPaint)
	ele.AddEvent_LButtonUp(it.onLButtonUp)
	ele.AddEvent_MouseMove(it.onMouseMove)
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		if it.hAction || it.hClose {
			it.hAction, it.hClose = false, false
			it.ele.Redraw(false)
		}
		return 0
	})
	ele.AddEvent_XC_Timer(func(hEle, nID int, pbHandled *bool) int {
		if uint32(nID) == toastTickID {
			it.tick()
		}
		return 0
	})
	// 元素随窗口销毁时把条目摘掉: 句柄可能被复用, 留着会让 layout 往无效句柄上设矩形。
	ele.AddEvent_Destroy_End(func(hEle int, pbHandled *bool) int {
		it.destroyed = true
		t.removeItem(it)
		return 0
	})
	// 主题切换: 字体/圆角/配色都可能变, 重新量尺寸后再重排(卡片宽高与文字折行都受影响)。
	it.onThemeC = func(*xui.Theme) {
		if it.destroyed || it.closed {
			return
		}
		t.layout()
	}
	xui.OnThemeChangeFor(ele.Handle, it.onThemeC)
	xui.AttachWindow(ele.Handle)

	t.items = append(t.items, it)
	t.layout()
	it.enter()
	if o.Duration > 0 {
		it.startTick()
	}
	return it
}

// Default / Success / Error / Warning / Info 是常用类型的快捷入口。
func (t *Toaster) Default(title, description string) *Toast {
	return t.Show(ToastOptions{Title: title, Description: description})
}

func (t *Toaster) Success(title, description string) *Toast {
	return t.Show(ToastOptions{Title: title, Description: description, Variant: ToastSuccess})
}

func (t *Toaster) Error(title, description string) *Toast {
	return t.Show(ToastOptions{Title: title, Description: description, Variant: ToastError})
}

func (t *Toaster) Warning(title, description string) *Toast {
	return t.Show(ToastOptions{Title: title, Description: description, Variant: ToastWarning})
}

func (t *Toaster) Info(title, description string) *Toast {
	return t.Show(ToastOptions{Title: title, Description: description, Variant: ToastInfo})
}

// Loading 显示一条转圈的加载通知(sonner 的 toast.loading), 默认常驻不自动关闭,
// 完成后用 Update 改成结果态(或直接改用 Promise)。
func (t *Toaster) Loading(title, description string) *Toast {
	return t.Show(ToastOptions{
		Title:       title,
		Description: description,
		Variant:     ToastLoading,
		Duration:    -1,
	})
}

// Promise 执行 fn 并在期间显示一条 loading 通知, 结束后把**同一条**通知更新成
// success / error 并重新开始倒计时(对应 sonner 的 toast.promise)。
//
// fn 在独立 goroutine 里执行, 结果经 xc.Auto 回到 UI 线程再改界面 —— 不要自己
// 在 goroutine 里调 Toaster/Toast 的方法(不是线程安全的)。
//
// 常驻形态: loading 阶段 Duration 为 -1; 成功/失败文案沿用 ToastOptions.Duration
// (0 = Toaster 默认 4s), 想让结果也常驻就在 Success/Error 里传负数。
func (t *Toaster) Promise(fn func() (string, error), p ToastPromise) *Toast {
	if t == nil || t.win == nil || fn == nil {
		return nil
	}
	loading := p.Loading
	loading.Variant = ToastLoading
	loading.Duration = -1
	it := t.Show(loading)
	if it == nil {
		return nil
	}
	go func() {
		result, err := promiseCall(fn)
		// 通知已被用户关掉 / 被挤掉: 不再更新(句柄可能已销毁)。
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

// promiseCall 执行用户函数并把 panic 转成错误(否则 goroutine 里的 panic 会直接
// 崩掉进程, 而通知会永远停在转圈上)。
func promiseCall(fn func() (string, error)) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return fn()
}

// Count 返回当前显示中的通知条数(含正在退场的)。
func (t *Toaster) Count() int {
	if t == nil {
		return 0
	}
	return len(t.items)
}

// Clear 立即移除全部通知(不播放退场动画)。
func (t *Toaster) Clear() {
	if t == nil {
		return
	}
	items := t.items
	t.items = nil
	for _, it := range items {
		it.destroy()
	}
}

// Update 更新通知内容(文本/类型/动作)并重排。传入的 Duration 会重置倒计时
// (0 = 用 Toaster 默认值, 负数 = 转为常驻)。
func (it *Toast) Update(o ToastOptions) *Toast {
	if it == nil || it.closed || it.destroyed {
		return it
	}
	if o.Duration == 0 {
		o.Duration = it.t.duration
	}
	if o.Duration < 0 {
		o.Duration = -1
	}
	it.opts = o
	it.remain = o.Duration
	if o.Duration > 0 {
		it.startTick()
	} else {
		it.stopTick()
	}
	// 从转圈态切到结果态时必须显式关掉旋转: xui.Spin 的登记是"一次登记持续运行",
	// 只在绘制时不再调用它并不会停表 —— 定时器会一直空转重绘这张卡片。
	if !it.loading() {
		xui.Spin(it.ele.Handle, false)
	}
	it.t.layout()
	it.ele.Redraw(false)
	return it
}

// Close 关闭通知: 淡出 + 向侧边滑出, 完成后销毁元素。算作"主动关闭"(触发 OnDismiss)。
func (it *Toast) Close() { it.close(false) }

// close 是关闭的统一入口: auto=true 表示倒计时到点(触发 OnAutoClose),
// 否则是用户/调用方主动关闭(触发 OnDismiss)。两者都走退场动画, 回调在销毁时发出。
func (it *Toast) close(auto bool) {
	if it == nil || it.closed || it.destroyed {
		return
	}
	it.autoClosed = auto
	it.closed = true
	it.stopTick()
	it.t.removeItem(it)
	// 先把其余卡片归位(它们立即上移/下移), 再让本条滑出去。
	it.t.layout()

	baseX, baseY := it.baseX, it.baseY
	hEle := it.ele.Handle
	hWnd := it.t.hWnd
	dx := int32(toastSlideOut)
	dy := int32(0)
	switch it.t.position {
	case ToastBottomLeft, ToastTopLeft:
		dx = -dx
	case ToastTopCenter:
		dx, dy = 0, -toastSlideOut // 顶部居中: 向上滑出
	case ToastBottomCenter:
		dx, dy = 0, toastSlideOut // 底部居中: 向下滑出
	}
	key := it.key
	xui.Presence(hWnd, key, false, toastExitDuration, func(v float64) {
		if !xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
			return
		}
		xui.SetElementsAlpha(v, hEle)
		toastSetRect(hEle, baseX+int32(float64(dx)*(1-v)+0.5), baseY+int32(float64(dy)*(1-v)+0.5), it.w, it.h)
	}, func(float64) {
		it.destroy()
		xc.XWnd_Redraw(hWnd, true)
	})
}

// Handle 返回通知元素句柄(需要自行绘制/挂事件时可用的逃生口)。
func (it *Toast) Handle() int {
	if it == nil || it.ele == nil {
		return 0
	}
	return it.ele.Handle
}

// destroy 立即销毁通知元素(不播动画)并按关闭原因回调 OnAutoClose / OnDismiss / OnClose。
func (it *Toast) destroy() {
	if it == nil || it.destroyed {
		return
	}
	it.destroyed = true
	it.stopTick()
	hEle := 0
	if it.ele != nil {
		hEle = it.ele.Handle
	}
	if hEle != 0 && xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
		xui.Spin(hEle, false) // 转圈态可能正被销毁(见 Update 里的说明)
		xc.XEle_Destroy(hEle)
	}
	it.t.removeItem(it)
	if it.autoClosed {
		if it.opts.OnAutoClose != nil {
			it.opts.OnAutoClose()
		}
	} else if it.opts.OnDismiss != nil {
		it.opts.OnDismiss()
	}
	if it.opts.OnClose != nil {
		it.opts.OnClose()
	}
}

// enter 播放进场: 元素 alpha 0→1, 位置从堆叠方向外侧滑入。
func (it *Toast) enter() {
	hEle := it.ele.Handle
	hWnd := it.t.hWnd
	it.entered = false
	// 起始帧: 全透明 + 偏移(先设好属性再让整窗画一遍, 避免首帧闪出全亮的卡片)。
	xui.SetElementsAlpha(0, hEle)
	offset := int32(toastSlide)
	if it.t.position == ToastBottomRight || it.t.position == ToastBottomLeft ||
		it.t.position == ToastBottomCenter {
		offset = -offset // 底部堆叠(含底部居中): 从更下方滑上来
	}
	toastSetRect(hEle, it.baseX, it.baseY-offset, it.w, it.h)
	xc.XWnd_Redraw(hWnd, true)

	xui.Presence(hWnd, it.key, true, toastEnterDuration, func(v float64) {
		if it.closed || it.destroyed || !xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
			return
		}
		xui.SetElementsAlpha(v, hEle)
		toastSetRect(hEle, it.baseX, it.baseY-int32(float64(offset)*(1-v)), it.w, it.h)
	}, func(float64) {
		it.entered = true
	})
}

// tick 是倒计时的一帧(元素上的炫彩定时器每 100ms 调一次)。
func (it *Toast) tick() {
	if it.closed || it.destroyed || it.paused || it.remain <= 0 {
		return
	}
	it.remain -= toastTickInterval
	if it.remain <= 0 {
		it.close(true) // 倒计时到点: 触发 OnAutoClose
	}
}

func (it *Toast) startTick() {
	if it.ele == nil || it.destroyed || it.closed {
		return
	}
	xc.XEle_SetXCTimer(it.ele.Handle, toastTickID, uint32(toastTickInterval/time.Millisecond))
}

func (it *Toast) stopTick() {
	if it.ele == nil || it.destroyed {
		return
	}
	if xc.XC_IsHXCGUI(it.ele.Handle, xcc.XC_ELE) {
		xc.XEle_KillXCTimer(it.ele.Handle, toastTickID)
	}
}

// removeItem 从管理器的列表里摘掉一条(幂等)。
func (t *Toaster) removeItem(it *Toast) {
	for i, v := range t.items {
		if v == it {
			t.items = append(t.items[:i], t.items[i+1:]...)
			return
		}
	}
}

// layout 重新量宽/量高并把所有通知摆到静止位置。正在进场/退场的条目由各自的
// apply 回调控制位置, 这里只更新它的 baseX/baseY(下一帧自动跟上)。
func (t *Toaster) layout() {
	if t == nil || t.win == nil || len(t.items) == 0 {
		return
	}
	br := t.bodyRect()
	w := t.width
	if max := br.Right - br.Left - 2*t.offset; w > max {
		w = max
	}
	if w < toastMinWidth {
		w = toastMinWidth
	}
	left := t.position == ToastBottomLeft || t.position == ToastTopLeft
	center := t.position == ToastTopCenter || t.position == ToastBottomCenter
	top := t.position == ToastTopRight || t.position == ToastTopLeft || t.position == ToastTopCenter

	y := br.Bottom - t.offset
	if top {
		y = br.Top + t.offset
	}
	for i := len(t.items) - 1; i >= 0; i-- {
		it := t.items[i]
		it.measure(w)
		if top {
			it.baseY = y
			y += it.h + t.gap
		} else {
			it.baseY = y - it.h
			y = it.baseY - t.gap
		}
		if left {
			it.baseX = br.Left + t.offset
		} else if center {
			it.baseX = br.Left + (br.Right-br.Left-it.w)/2
		} else {
			it.baseX = br.Right - t.offset - it.w
		}
		// 动画中的条目不要直接落位(会被下一帧 apply 覆盖, 但旧位置会留残影)。
		if it.entered {
			toastSetRect(it.ele.Handle, it.baseX, it.baseY, it.w, it.h)
		}
		it.pin(br)
		it.ele.SetZOrderEx(0, xcc.Zorder_Top)
	}
	xc.XWnd_Redraw(t.hWnd, true)
}

// pin 把通知元素"钉"成相对窗口 Body 的四边 inset(布局项置位置)。
//
// 窗口级布局趟(拖拽缩放、页签换行重排等组件逐级重排到 XWnd_AdjustLayout)会把
// 窗口直属的绝对定位元素当布局项排到流式位置 —— 左侧(见 DESIGN 96)。元素无法
// 豁免布局(没有"不参与布局"开关, XE_ADJUSTLAYOUT_END 事件注册即崩), 但布局项
// 置位置是四边 inset 钉住: 钉住后布局趟把元素摆到钉的位置而不是流式位置, 且
// 钉在 XEle_SetRect(进出场动画逐帧挪动)之后仍然存活 —— 实测: 探针钉住后
// SetRect 挪走再 AdjustLayout, 元素回到钉位。所以每次 layout() 都按当前静止
// 位置重钉, 一切窗口级重排反而会帮我们把通知摆正。
//
// 注意 insets 必须与元素矩形一致: 四边都钉住的项在布局趟里被拉伸到四边之间,
// 缩放后旧 inset 会把元素拉变形 —— 所以窗口 Size 事件里的 layout() 必须用新
// Body 矩形重钉(元素矩形 = 卡片外扩 toastShadow, 与 toastSetRect 同一套)。
func (it *Toast) pin(br xc.RECT) {
	if it.ele == nil {
		return
	}
	x := it.baseX - toastShadow
	y := it.baseY - toastShadow
	cx := it.w + 2*toastShadow
	cy := it.h + 2*toastShadow
	it.ele.LayoutItem_SetPosition(x-br.Left, y-br.Top, br.Right-(x+cx), br.Bottom-(y+cy))
}

// bodyRect 返回窗口内容区(Body)的矩形。阴影窗口的 Body 比整窗小一圈,
// 卡片要贴着 Body 边缘而不是阴影带边缘。
func (t *Toaster) bodyRect() xc.RECT {
	var br xc.RECT
	xc.XWnd_GetBodyRect(t.hWnd, &br)
	if br.Right <= br.Left || br.Bottom <= br.Top {
		xc.XWnd_GetRect(t.hWnd, &br)
	}
	return br
}

// measure 按给定卡片宽度量出本条通知的高度与内部排版。
func (it *Toast) measure(w int32) {
	th := xui.Current()
	it.w = w

	// 图标列
	x := toastPad
	if it.iconData() != "" {
		it.geom.icon = xc.RECT{Left: x, Top: 0, Right: x + toastIconSize, Bottom: toastIconSize}
		x += toastIconSize + toastIconGap
	} else {
		it.geom.icon = xc.RECT{}
	}
	right := w - toastPad

	// 关闭按钮: 右上角(内缩 8), 标题需要让位
	it.geom.close = xc.RECT{}
	if it.opts.CloseButton {
		it.geom.close = xc.RECT{
			Left:   w - toastPad - 6 - toastCloseSize,
			Top:    toastPad - 6,
			Right:  w - toastPad - 6,
			Bottom: toastPad - 6 + toastCloseSize,
		}
		right = it.geom.close.Left - 4
	}

	// 动作按钮: 右侧, 宽度按文字量出
	it.geom.action = xc.RECT{}
	if it.opts.Action != "" {
		aw := xui.TextWidth(it.opts.Action, toastFont(th, false)) + 2*toastActionPadX
		it.geom.action = xc.RECT{
			Left:   right - aw,
			Top:    0,
			Right:  right,
			Bottom: toastActionH,
		}
		right -= aw + toastActionGap
	}

	// 文本列
	textW := right - x
	if textW < 40 {
		textW = 40
	}
	titleH := int32(0)
	if it.opts.Title != "" {
		titleH = toastTitleH
	}
	descH := int32(0)
	if it.opts.Description != "" {
		descH = toastTextHeight(it.opts.Description, toastFont(th, false), textW)
	}
	textH := titleH
	if descH > 0 {
		if textH > 0 {
			textH += toastDescGap
		}
		textH += descH
	}

	// 内容区高度 = 文本列 / 图标 / 动作按钮三者的最大高度
	bodyH := textH
	if it.geom.icon.Bottom-it.geom.icon.Top > bodyH {
		bodyH = it.geom.icon.Bottom - it.geom.icon.Top
	}
	if it.geom.action.Bottom-it.geom.action.Top > bodyH {
		bodyH = it.geom.action.Bottom - it.geom.action.Top
	}
	if bodyH <= 0 {
		bodyH = toastTitleH
	}
	it.h = toastPad*2 + bodyH

	// 垂直排布: 文本列、图标、动作按钮都以主体区中心为基准
	mid := toastPad + bodyH/2
	if titleH > 0 {
		it.geom.title = xc.RECT{Left: x, Top: mid - textH/2, Right: x + textW, Bottom: mid - textH/2 + titleH}
	} else {
		it.geom.title = xc.RECT{}
	}
	if descH > 0 {
		descTop := mid - textH/2
		if titleH > 0 {
			descTop += titleH + toastDescGap
		}
		it.geom.desc = xc.RECT{Left: x, Top: descTop, Right: x + textW, Bottom: descTop + descH}
	} else {
		it.geom.desc = xc.RECT{}
	}
	if it.geom.icon.Bottom > it.geom.icon.Top {
		it.geom.icon.Top = mid - toastIconSize/2
		it.geom.icon.Bottom = it.geom.icon.Top + toastIconSize
	}
	if it.geom.action.Bottom > it.geom.action.Top {
		it.geom.action.Top = mid - toastActionH/2
		it.geom.action.Bottom = it.geom.action.Top + toastActionH
	}
}

// iconData 返回本条实际使用的图标(path 数据 / 整段 SVG; 空 = 无图标)。
func (it *Toast) iconData() string {
	if it.opts.Icon != "" {
		return it.opts.Icon
	}
	switch it.opts.Variant {
	case ToastSuccess:
		return icons.IconCircleCheck
	case ToastError:
		return icons.IconCircleX
	case ToastWarning:
		return icons.IconTriangleAlert
	case ToastInfo:
		return icons.IconInfo
	case ToastLoading:
		return icons.IconLoaderCircle
	}
	return ""
}

// loading 报告本条是否是转圈态(自定义图标时按调用方给的图静态绘制)。
func (it *Toast) loading() bool {
	return it.opts.Variant == ToastLoading && it.opts.Icon == ""
}

// iconColor 返回图标颜色: 未显式指定时用类型的语义色(sonner 的 --success-text 等)。
// 这两套色值是 sonner 亮/暗主题下的状态色, 不属于 shadcn 令牌。
func (it *Toast) iconColor(th *xui.Theme) xui.Color {
	if it.opts.IconColor != xui.None {
		return it.opts.IconColor
	}
	dark := th.IsDark()
	switch it.opts.Variant {
	case ToastSuccess:
		if dark {
			return xui.RGBA(134, 239, 172, 255) // hsl(141 79% 85%)
		}
		return xui.RGBA(0, 138, 46, 255) // hsl(140 100% 27%)
	case ToastError:
		if dark {
			return xui.RGBA(255, 162, 162, 255) // hsl(0 100% 85%)
		}
		return xui.RGBA(231, 0, 11, 255) // hsl(360 100% 45%)
	case ToastWarning:
		if dark {
			return xui.RGBA(255, 212, 102, 255) // hsl(46 100% 70%)
		}
		return xui.RGBA(221, 122, 0, 255) // hsl(31 92% 45%)
	case ToastInfo:
		if dark {
			return xui.RGBA(153, 204, 255, 255) // hsl(210 100% 80%)
		}
		return xui.RGBA(0, 112, 224, 255) // hsl(210 100% 44%)
	case ToastLoading:
		// sonner 的 loading 转圈用默认前景色(不加类型色), 与后续成功/失败态对比明显
		return th.MutedForeground
	}
	return th.PopoverForeground
}

// toastRichPalette 是 richColors 开启时一张通知卡的三色(卡面/描边/文字)。
type toastRichPalette struct {
	bg, border, text xui.Color
}

// toastRichColorsFor 返回类型在亮/暗主题下的富色; 无类型的 default 返回 false
// (卡面维持 bg-popover)。色值逐条取自 sonner 的 styles.css(data-rich-colors 变量),
// 与 iconColor 一样不属于 shadcn 令牌。
func toastRichColorsFor(v ToastVariant, dark bool) (toastRichPalette, bool) {
	switch v {
	case ToastSuccess:
		if dark {
			return toastRichPalette{xui.RGBA(0, 31, 15, 255), xui.RGBA(0, 61, 28, 255), xui.RGBA(89, 243, 166, 255)}, true
		}
		return toastRichPalette{xui.RGBA(236, 253, 243, 255), xui.RGBA(191, 252, 217, 255), xui.RGBA(0, 138, 46, 255)}, true
	case ToastError:
		if dark {
			return toastRichPalette{xui.RGBA(45, 6, 7, 255), xui.RGBA(77, 4, 8, 255), xui.RGBA(255, 158, 161, 255)}, true
		}
		return toastRichPalette{xui.RGBA(255, 240, 240, 255), xui.RGBA(255, 224, 225, 255), xui.RGBA(230, 0, 0, 255)}, true
	case ToastWarning:
		if dark {
			return toastRichPalette{xui.RGBA(29, 31, 0, 255), xui.RGBA(46, 46, 0, 255), xui.RGBA(243, 207, 88, 255)}, true
		}
		return toastRichPalette{xui.RGBA(255, 252, 240, 255), xui.RGBA(251, 238, 177, 255), xui.RGBA(220, 118, 9, 255)}, true
	case ToastInfo:
		if dark {
			return toastRichPalette{xui.RGBA(0, 13, 31, 255), xui.RGBA(25, 35, 62, 255), xui.RGBA(88, 150, 243, 255)}, true
		}
		return toastRichPalette{xui.RGBA(240, 248, 255, 255), xui.RGBA(221, 231, 253, 255), xui.RGBA(9, 115, 220, 255)}, true
	}
	return toastRichPalette{}, false
}

// richPalette 返回本条的富色(整组或单条开启时); default 类型没有富色。
func (it *Toast) richPalette(th *xui.Theme) (toastRichPalette, bool) {
	if it == nil || it.t == nil || (!it.t.richColors && !it.opts.RichColors) {
		return toastRichPalette{}, false
	}
	return toastRichColorsFor(it.opts.Variant, th.IsDark())
}

// onPaint 绘制整张通知卡: 投影 / 底色 / 边框 / 图标 / 标题 / 描述 / 动作 / 关闭按钮。
func (it *Toast) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()

	// 卡片外扩投影留白(toastShadow); 卡片矩形是元素客户区内缩后的区域。
	card := xc.RECT{
		Left:   p.Rc.Left + toastShadow,
		Top:    p.Rc.Top + toastShadow,
		Right:  p.Rc.Right - toastShadow,
		Bottom: p.Rc.Bottom - toastShadow,
	}
	if card.Right <= card.Left || card.Bottom <= card.Top {
		return 0
	}
	radius := th.RadiusLg()

	// richColors: 按类型换整卡底色/描边/文字; 未开启或 default 类型保持 bg-popover。
	cardBg, cardBorder := th.Popover, th.Border
	titleC, descC, closeC := th.PopoverForeground, th.MutedForeground, th.MutedForeground
	if pal, ok := it.richPalette(th); ok {
		cardBg, cardBorder = pal.bg, pal.border
		titleC, descC, closeC = pal.text, pal.text, pal.text
	}

	// shadow-md 的近似: 向下偏移 2px 的一层柔和投影(颜色固定半透明黑, 不随主题变)
	sh := card
	sh.Top += 2
	sh.Bottom += 2
	p.FillRound(&sh, xui.RGBA(0, 0, 0, 18), radius)

	p.FillRound(&card, cardBg, radius)
	p.StrokeRound(&card, cardBorder, radius, 1)

	// 所有内部矩形都是卡片局部坐标, 绘制时整体平移 shadow
	off := func(rc xc.RECT) xc.RECT {
		rc.Left += toastShadow
		rc.Right += toastShadow
		rc.Top += toastShadow
		rc.Bottom += toastShadow
		return rc
	}

	// 图标: 转圈态走 IconRotated, 相位由运行时按墙钟算并驱动元素重绘(同 Spinner)
	if data := it.iconData(); data != "" {
		rc := off(it.geom.icon)
		if it.loading() {
			p.IconRotated(data, &rc, it.iconColor(th), 2, xui.Spin(hEle, true)*2*math.Pi)
		} else {
			p.Icon(data, &rc, it.iconColor(th), 2)
		}
	}

	// 标题(text-[13px] font-medium, 单行截断)
	if it.opts.Title != "" {
		f := toastFont(th, true)
		if f != 0 {
			p.Draw.SetFont(f)
		}
		rc := off(it.geom.title)
		p.Text(it.opts.Title, &rc, titleC,
			xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|
				xcc.TextFormatFlag_NoWrap|xcc.TextTrimming_EllipsisCharacter)
	}

	// 描述(text-[13px], 可折行; 行高按量测结果给足, 避免末行被裁)
	if it.opts.Description != "" {
		f := toastFont(th, false)
		if f != 0 {
			p.Draw.SetFont(f)
		}
		rc := off(it.geom.desc)
		p.Text(it.opts.Description, &rc, descC,
			xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	}

	// 动作按钮: 悬停 bg-accent, 静止 bg-muted
	if it.opts.Action != "" {
		hv := xui.HoverItem(hEle, 0, it.hAction)
		rc := off(it.geom.action)
		bg := th.Muted
		if hv > 0 {
			bg = xui.Fade(th.Muted, th.Accent, hv)
		}
		p.FillRound(&rc, bg, th.RadiusSm())
		if f := toastFont(th, false); f != 0 {
			p.Draw.SetFont(f)
		}
		p.Text(it.opts.Action, &rc, th.PopoverForeground,
			xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
	}

	// 关闭按钮: 悬停底色 + X 图标
	if it.opts.CloseButton {
		hv := xui.HoverItem(hEle, 1, it.hClose)
		rc := off(it.geom.close)
		if hv > 0 {
			p.FillRound(&rc, th.Accent.ScaleAlpha(hv), th.RadiusSm())
		}
		n := int32(14)
		icon := xc.RECT{
			Left:   (rc.Left+rc.Right)/2 - n/2,
			Top:    (rc.Top+rc.Bottom)/2 - n/2,
			Right:  (rc.Left+rc.Right)/2 + n/2,
			Bottom: (rc.Top+rc.Bottom)/2 + n/2,
		}
		p.Icon(icons.IconX, &icon, closeC, 2)
	}
	return 0
}

// onLButtonUp 处理动作按钮与关闭按钮的点击(卡片其余区域不响应)。
func (it *Toast) onLButtonUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pPt == nil || it.closed || it.destroyed {
		return 0
	}
	x, y := pPt.X-toastShadow, pPt.Y-toastShadow
	if it.opts.CloseButton && toastPointIn(it.geom.close, x, y) {
		*pbHandled = true
		it.Close()
		return 0
	}
	if it.opts.Action != "" && toastPointIn(it.geom.action, x, y) {
		*pbHandled = true
		if it.opts.OnAction != nil {
			it.opts.OnAction()
		}
		if !it.opts.KeepOpen {
			it.Close()
		}
	}
	return 0
}

// onMouseMove 更新两个按钮的悬停态(整张卡片是一个元素, 需要自己判命中)。
func (it *Toast) onMouseMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pPt == nil {
		return 0
	}
	x, y := pPt.X-toastShadow, pPt.Y-toastShadow
	ha := it.opts.Action != "" && toastPointIn(it.geom.action, x, y)
	hc := it.opts.CloseButton && toastPointIn(it.geom.close, x, y)
	if ha != it.hAction || hc != it.hClose {
		it.hAction, it.hClose = ha, hc
		it.ele.Redraw(false)
	}
	return 0
}

// toastFont 返回通知文字用的字体句柄: 标题用粗体字形(13px 下的中等字重), 描述用常规。
func toastFont(th *xui.Theme, bold bool) int {
	spec := th.FontDefault
	if spec.Size > 1 {
		spec.Size-- // text-[13px] 相对正文 14px
	}
	spec.Bold = bold
	return xui.FontHandle(spec)
}

// toastTextHeight 量出按宽度折行后的文本高度(绘制与量测必须同一字体、同一限宽)。
//
// 实测(微软雅黑 13px): GetTextShowRect 返回的 CY **已含行距**
// —— 单行 18、两行 36、三行 53, 与 p.Text 的实际排版一致。别自作聪明再按行数补偿,
// 那样底部会多出一行留白(第一版踩过)。
func toastTextHeight(text string, hFont int, w int32) int32 {
	if text == "" || hFont == 0 {
		return 0
	}
	var sz xc.SIZE
	app.GetTextShowRect(text, -1, hFont, xcc.TextAlignFlag_Left, w, &sz)
	if sz.CY <= 0 {
		return toastTitleH
	}
	return sz.CY
}

func toastPointIn(rc xc.RECT, x, y int32) bool {
	return rc.Right > rc.Left && rc.Bottom > rc.Top &&
		x >= rc.Left && x < rc.Right && y >= rc.Top && y < rc.Bottom
}

// toastSetRect 按**卡片**矩形设置元素矩形: 元素四周各外扩 toastShadow 画投影。
// 对外(布局/baseX/baseY)一律用卡片坐标, 别把投影留白算进排版 —— 否则卡片实际
// 可见高度比预留的少 2*toastShadow, 内边距上下各被吃掉 6px(文字顶到卡片底)。
// 组件各自带一份: 本文件必须能独立编译(copy-not-install 下不保证 dialog.go 也在)。
func toastSetRect(hEle int, x, y, cx, cy int32) {
	x -= toastShadow
	y -= toastShadow
	cx += 2 * toastShadow
	cy += 2 * toastShadow
	xc.XEle_SetRect(hEle, &xc.RECT{Left: x, Top: y, Right: x + cx, Bottom: y + cy},
		false, xcc.AdjustLayout_No)
}
