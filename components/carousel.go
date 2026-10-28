package ui

import (
	"math"
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Carousel 几何常量(逻辑像素), 对齐 shadcn carousel.tsx(radix-nova):
//
//   - CarouselPrevious/Next 是 outline + icon-sm + rounded-full, 绝对定位在内容两侧
//     `-left-12` / `-right-12` —— 即按钮与内容之间留 48 的槽位, 组件把这 48 做进根元素
//     (相当于 shadcn 示例外层的 `px-12`), 内容视口因此比根元素窄 96;
//   - CarouselContent 的 `-ml-4` + CarouselItem 的 `pl-4` 合成"每页之间 16 的间隙",
//     这里等价为"幻灯片宽 = 视口宽 − 16, 步进 = 视口宽"。
const (
	carouselGutter          int32 = 48 // 按钮槽位宽(-left-12 / -right-12)
	carouselGap             int32 = 16 // 幻灯片之间的间隙(pl-4 / -ml-4)
	carouselButtonSize      int32 = 28 // icon-sm
	carouselDragSlop        int32 = 6  // 超过这个位移才算拖拽, 否则当普通点击
	carouselAutoTimerID           = uint32(0x7873)
	carouselDurationDefault       = 300 * time.Millisecond
)

// CarouselOption 轮播配置项。
type CarouselOption func(*Carousel)

// WithCarouselLoop 循环播放(默认关, 同 embla 的 loop=false); 开启后首尾按钮始终可用。
func WithCarouselLoop(loop bool) CarouselOption {
	return func(c *Carousel) { c.loop = loop }
}

// WithCarouselVertical 纵向轮播(默认横向)。按钮变为内容上/下方的圆形钮, 图标换成上下箭头
// (等价于 shadcn 的 `rotate-90`); 键盘仍响应 ←/→(与上游 handleKeyDown 一致)。
func WithCarouselVertical() CarouselOption {
	return func(c *Carousel) { c.vertical = true }
}

// WithCarouselControls 是否创建上一张/下一张按钮(默认 true)。关掉后内容占满整个根元素。
func WithCarouselControls(show bool) CarouselOption {
	return func(c *Carousel) { c.controls = show }
}

// WithCarouselGap 幻灯片之间的间隙(默认 16); 传 0 为无缝整页切换。
func WithCarouselGap(gap int32) CarouselOption {
	return func(c *Carousel) {
		if gap >= 0 {
			c.gap = gap
		}
	}
}

// WithCarouselDuration 切换页的过渡时长(默认 300ms)。
func WithCarouselDuration(d time.Duration) CarouselOption {
	return func(c *Carousel) {
		if d > 0 {
			c.duration = d
		}
	}
}

// WithCarouselInitialIndex 初始页(默认 0)。
func WithCarouselInitialIndex(i int) CarouselOption {
	return func(c *Carousel) { c.initial = i }
}

// WithCarouselOnChange 页变化(过渡落定后)回调。
func WithCarouselOnChange(fn func(index int)) CarouselOption {
	return func(c *Carousel) {
		if fn != nil {
			c.changes = append(c.changes, fn)
		}
	}
}

// WithCarouselAutoplay 自动翻页间隔(默认关闭)。自动翻页不是 shadcn 弹层核心里的一部分
// (embla 的 autoplay 插件), 这里按插件默认语义实现: 用户拖动/点箭头/按键后停止,
// 循环模式下一直转, 非循环模式到末页停住; 页面不可见(尺寸为 0)时不跑。
func WithCarouselAutoplay(d time.Duration) CarouselOption {
	return func(c *Carousel) {
		if d > 0 {
			c.autoplay = d
		}
	}
}

// CarouselSlide 是轮播中的一页: 一个纵向流式容器, 宽高由 Carousel 管理。
//
// 幻灯片之间有 gap 的间隙; 内容超出会被幻灯片(及视口)按矩形裁剪, 需要滚动时自己在
// 里面放 ui.ScrollArea。内边距/子项间距可以像普通容器一样用 SetPadding / SetSpace。
type CarouselSlide struct {
	*widget.LayoutEle
}

// Carousel 整页位移的轮播容器, 对应 shadcn carousel.tsx。
//
// 与 shadcn 对齐:
//   - 每页与视口等宽(减 gap), 整页位移; 上一张/下一张按钮是圆角描边的 icon-sm,
//     位于内容两侧 48 的槽位里, 到达边界时禁用(loop 时始终可用);
//   - ←/→ 在轮播内任意控件获得焦点时翻页(上游 onKeyDownCapture 的语义);
//   - 支持鼠标拖拽, 松手后吸附到最近一页(拖动阈值 6, 与点击区分);
//   - 循环模式首尾相接无回卷跳帧: 位移用"模总长"折回, 最后一张的下一页紧跟着第一张。
//
// 与 shadcn 的差异:
//   - 高度显式给出(构造参数 cy): 网页里轮播高度由最高的一页撑开, 自绘容器没有可靠的
//     "离屏量高"通道; 需要随内容长高时把高度放到卡片外的布局项里自己控制;
//   - 按钮槽位固定占 96(左右各 48), 与 shadcn 示例外层 px-12 的布局等价; 不要按钮时
//     传 WithCarouselControls(false), 内容会占满整个根元素;
//   - 触摸/惯性按桌面鼠标实现: 拖动跟手, 松手按"最近一页"吸附, 不做 velocity 惯性。
//
// 用法:
//
//	car := ui.NewCarousel(0, 0, 0, 240, parent, ui.WithCarouselLoop(true))
//	xui.ItemFill(car) // 宽按布局走, 高由构造参数给出
//	car.AddSlide(func(s *ui.CarouselSlide) {
//	    ui.NewCard(...).LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
//	})
//	car.OnChange(func(i int) { ... })
type Carousel struct {
	*widget.Element

	viewport *widget.Element
	slides   []*CarouselSlide
	prev     *Button
	next     *Button

	loop     bool
	vertical bool
	controls bool
	gap      int32
	duration time.Duration
	autoplay time.Duration
	initial  int
	changes  []func(int)

	index   int
	pending int     // 正在过渡到的页(连点 Next 时以它为准, 否则第二次点击会被旧的 index 吃掉)
	curPx   float64 // 当前位移(逻辑像素), 静止时 = index × step
	step    float64 // 每页步进 = 视口在主轴上 的长度
	moving  bool

	indexInit bool

	hWindow int
	win     *window.Window

	dragArmed  bool
	dragging   bool
	dragStart  int32   // 按下点(窗口客户区坐标, 主轴上)
	dragStartP float64 // 按下时的 curPx
	vpOrigin   int32   // 视口原点在窗口客户区里的主轴坐标(捕获后坐标要换回来)

	autoOn      bool
	autoStopped bool
}

// NewCarousel 创建轮播容器。尺寸/布局项语义与普通容器相同(默认不接管尺寸, 用 LayoutItem_* 声明);
// cy 是内容区高度(横向)或内容区宽度(纵向), 按钮槽位已经包含在根元素尺寸里。
func NewCarousel(x, y, cx, cy int32, hParent int, opts ...CarouselOption) *Carousel {
	c := &Carousel{
		gap:      carouselGap,
		controls: true,
		duration: carouselDurationDefault,
	}
	for _, o := range opts {
		o(c)
	}

	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	c.Element = ele

	c.viewport = widget.NewElement(0, 0, 0, 0, ele.Handle)
	c.viewport.EnableBkTransparent(true)
	c.viewport.EnableDrawBorder(false)
	c.viewport.EnableFocus(false)

	if c.controls {
		prevIcon, nextIcon := icons.IconChevronLeft, icons.IconChevronRight
		if c.vertical {
			prevIcon, nextIcon = icons.IconChevronUp, icons.IconChevronDown
		}
		c.prev = NewButton(0, 0, carouselButtonSize, carouselButtonSize, "", ele.Handle,
			WithVariant(ButtonOutline), WithSize(ButtonSizeIconSm),
			WithButtonRounded(), WithIconSize(16), WithIcon(prevIcon))
		c.next = NewButton(0, 0, carouselButtonSize, carouselButtonSize, "", ele.Handle,
			WithVariant(ButtonOutline), WithSize(ButtonSizeIconSm),
			WithButtonRounded(), WithIconSize(16), WithIcon(nextIcon))
		if c.prev != nil {
			c.prev.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
				c.Prev()
				return 0
			})
		}
		if c.next != nil {
			c.next.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
				c.Next()
				return 0
			})
		}
	}

	ele.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		c.applyInitial(true)
		c.layoutGeometry()
		return 0
	})
	ele.AddEvent_XC_Timer(func(hEle, nID int, pbHandled *bool) int {
		if uint32(nID) != carouselAutoTimerID || !c.autoOn {
			return 0
		}
		// 隐藏页的元素会被布局排成 0 尺寸: 不可见时不跑, 切回来再继续。
		if c.GetWidth() <= 0 || len(c.slides) < 2 {
			return 0
		}
		c.advanceAuto()
		return 0
	})
	ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		c.stopAutoplay(false)
		return 0
	})
	c.bindInput()
	c.layoutGeometry()
	c.syncButtons()

	return c
}

// AddSlide 追加一页并返回它; build 里往幻灯片(纵向流容器)加内容。
func (c *Carousel) AddSlide(build func(s *CarouselSlide)) *CarouselSlide {
	ele := widget.NewLayoutEle(0, 0, 0, 0, c.viewport.Handle)
	if ele == nil {
		return nil
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false)
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	s := &CarouselSlide{LayoutEle: ele}
	c.slides = append(c.slides, s)
	if build != nil {
		build(s)
	}
	c.applyInitial(false)
	c.layoutGeometry()
	c.syncButtons()
	c.startAutoplay()
	return s
}

// SlideCount 返回页数。
func (c *Carousel) SlideCount() int { return len(c.slides) }

// Index 返回当前页(过渡中返回起始页)。
func (c *Carousel) Index() int { return c.index }

// SetIndex 跳到指定页(带动画, 越界会夹取/取模)。程序化跳转不停止自动翻页。
func (c *Carousel) SetIndex(i int) *Carousel {
	n := len(c.slides)
	if n == 0 {
		return c
	}
	if c.loop {
		i = ((i % n) + n) % n
	} else if i < 0 {
		i = 0
	} else if i >= n {
		i = n - 1
	}
	c.goTo(i, false)
	return c
}

// Next 下一页(到末页后: loop 回第一页, 否则不动); 属于用户交互, 会停止自动翻页。
func (c *Carousel) Next() *Carousel {
	n := len(c.slides)
	if n == 0 {
		return c
	}
	if c.pending >= n-1 && !c.loop {
		return c
	}
	c.goTo((c.pending+1)%n, true)
	return c
}

// Prev 上一页; 属于用户交互, 会停止自动翻页。
func (c *Carousel) Prev() *Carousel {
	n := len(c.slides)
	if n == 0 {
		return c
	}
	if c.pending <= 0 && !c.loop {
		return c
	}
	c.goTo((c.pending-1+n)%n, true)
	return c
}

// SetLoop 运行时切换循环模式。
func (c *Carousel) SetLoop(loop bool) *Carousel {
	c.loop = loop
	c.syncButtons()
	return c
}

// Loop 返回是否循环。
func (c *Carousel) Loop() bool { return c.loop }

// SetAutoplay 运行时设置自动翻页间隔(传 0 关闭); 会解除"用户交互后停止"的状态。
func (c *Carousel) SetAutoplay(d time.Duration) *Carousel {
	c.stopAutoplay(false)
	c.autoplay = d
	c.autoStopped = false
	c.startAutoplay()
	return c
}

// OnChange 追加页变化回调(过渡落定后才触发)。
func (c *Carousel) OnChange(fn func(index int)) *Carousel {
	if fn != nil {
		c.changes = append(c.changes, fn)
	}
	return c
}

// PrevButton 返回上一张按钮(WithCarouselControls(false) 时为 nil), 可改变体/加 Tooltip。
func (c *Carousel) PrevButton() *Button { return c.prev }

// NextButton 返回下一张按钮(WithCarouselControls(false) 时为 nil)。
func (c *Carousel) NextButton() *Button { return c.next }

// goTo 从当前视觉位置过渡到 target; from 用 curPx 而不是 index*step —— 过渡被中途打断时
// index 还是起始页, 直接按 index 算目标会多走/少走半页。
func (c *Carousel) goTo(target int, user bool) {
	n := len(c.slides)
	if n == 0 {
		return
	}
	if user {
		c.stopAutoplay(true)
	}
	c.indexInit = true
	c.pending = target
	if c.step <= 0 {
		old := c.index
		c.index = target
		c.curPx = 0
		c.syncButtons()
		if old != c.index {
			c.emit(target)
		}
		return
	}
	// 目标位置按 target×step 取"离当前视觉位置最近的等价点"(loop 时相差整轮的位置等价)。
	// 不能写成 curPx + delta×step: 过渡被中途打断时 curPx 停在两页之间, 那样会吸附到
	// 目标再跳一下(落定时 curPx 归位到 index×step)。
	to := float64(target) * c.step
	if c.loop && n > 1 {
		total := float64(n) * c.step
		for to-c.curPx > total/2 {
			to -= total
		}
		for c.curPx-to > total/2 {
			to += total
		}
	}
	if math.Abs(to-c.curPx) < 0.5 || c.duration <= 0 || xui.AnimationScale() == 0 {
		c.moving = false
		c.curPx = to
		c.settle(target)
		return
	}

	c.moving = true
	from := c.curPx
	// 同一视口上的旧过渡必须先丢掉: Presence 的键各自推进, 两条同时 apply 会互相覆盖
	// (与弹层快速重开的竞态同一类, 见 DESIGN 第八节 92)。
	xui.DropTransitions(c.viewport.Handle)
	key := xui.NextPresenceKey()
	xui.Presence(c.viewport.Handle, key, true, c.duration,
		func(v float64) {
			c.curPx = from + (to-from)*v
			c.positionSlides()
		},
		func(v float64) {
			c.moving = false
			c.settle(target)
		})
}

// settle 落到目标页: 归一化位移(loop 时折回 [0,(n−1)]×step 的等价位置)、同步按钮与回调。
func (c *Carousel) settle(target int) {
	n := len(c.slides)
	old := c.index
	c.index = target
	c.pending = target
	if n > 0 {
		c.curPx = float64(c.index) * c.step
	}
	c.positionSlides()
	c.syncButtons()
	if old != c.index {
		c.emit(c.index)
	}
}

// advanceAuto 是自动翻页的一步: 不停止自动翻页, 到末页(非 loop)停住。
func (c *Carousel) advanceAuto() {
	n := len(c.slides)
	if n < 2 || c.dragging {
		return
	}
	if c.pending >= n-1 && !c.loop {
		c.stopAutoplay(true)
		return
	}
	c.goTo((c.pending+1)%n, false)
}

// startAutoplay 按需启动自动翻页定时器(幂等; 定时器回调在构造时只注册一次)。
func (c *Carousel) startAutoplay() {
	if c.autoplay <= 0 || c.autoStopped || c.autoOn || len(c.slides) < 2 {
		return
	}
	c.autoOn = true
	xc.XEle_SetXCTimer(c.Handle, carouselAutoTimerID, uint32(c.autoplay/time.Millisecond))
}

// stopAutoplay 停止定时器; user=true 表示"用户交互导致停止", 之后不再自动恢复
// (与 embla autoplay 的 stopOnInteraction 默认值一致)。
func (c *Carousel) stopAutoplay(user bool) {
	if !c.autoOn && !user {
		return
	}
	if c.autoOn {
		xc.XEle_KillXCTimer(c.Handle, carouselAutoTimerID)
		c.autoOn = false
	}
	if user {
		c.autoStopped = true
	}
}

func (c *Carousel) emit(i int) {
	for _, fn := range c.changes {
		fn(i)
	}
}

// applyInitial 落初始页。AddSlide 每加一页都会调用一次(不锁定), 首次真实布局(Size)
// 再按最终页数落一次并锁定; 任何导航(goTo/拖拽)也会锁定, 之后改写 initial 不再生效。
// 不能只在第一页时落: 第一页时 initial 会被夹成 0, 后加的页就回不来了(探针踩过)。
func (c *Carousel) applyInitial(final bool) {
	if c.indexInit || len(c.slides) == 0 {
		return
	}
	n := len(c.slides)
	i := c.initial
	if c.loop {
		i = ((i % n) + n) % n
	} else if i < 0 {
		i = 0
	} else if i > n-1 {
		i = n - 1
	}
	c.index = i
	c.pending = i
	if final {
		c.indexInit = true
	}
}

// layoutGeometry 按根元素当前尺寸重排视口/按钮/幻灯片尺寸与静止位移。
func (c *Carousel) layoutGeometry() {
	if c.Element == nil || c.viewport == nil {
		return
	}
	if c.moving {
		// 尺寸变了, 旧的位移过渡按旧步进在跑: 直接落定到当前页, 避免停在两页之间。
		c.moving = false
		xui.DropTransitions(c.viewport.Handle)
	}
	w, h := c.GetWidth(), c.GetHeight()
	gutter := int32(0)
	if c.prev != nil || c.next != nil {
		gutter = carouselGutter
	}
	vx, vy, vw, vh := int32(0), int32(0), w, h
	if c.vertical {
		vy, vh = gutter, h-2*gutter
	} else {
		vx, vw = gutter, w-2*gutter
	}
	if vw < 0 {
		vw = 0
	}
	if vh < 0 {
		vh = 0
	}
	c.viewport.SetRectEx(vx, vy, vw, vh, false, xcc.AdjustLayout_No)
	c.step = float64(vw)
	if c.vertical {
		c.step = float64(vh)
	}

	for _, s := range c.slides {
		sw, sh := vw, vh
		if c.vertical {
			sh -= c.gap
		} else {
			sw -= c.gap
		}
		if sw < 0 {
			sw = 0
		}
		if sh < 0 {
			sh = 0
		}
		s.SetRectEx(0, 0, sw, sh, false, xcc.AdjustLayout_No)
		s.AdjustLayout()
	}

	if c.index >= len(c.slides) {
		c.index = len(c.slides) - 1
	}
	if c.index < 0 {
		c.index = 0
	}
	if c.pending >= len(c.slides) {
		c.pending = len(c.slides) - 1
	}
	if c.pending < 0 {
		c.pending = 0
	}
	c.curPx = float64(c.index) * c.step
	c.positionSlides()
	c.layoutButtons(w, h)
	c.syncButtons()
}

// positionSlides 只挪位置不改尺寸(过渡/拖拽每帧调用, 不能触发幻灯片内部重排)。
func (c *Carousel) positionSlides() {
	n := len(c.slides)
	if n == 0 {
		return
	}
	for i, s := range c.slides {
		x := carouselSlideOffset(i, c.curPx, c.step, n, c.loop)
		if c.vertical {
			s.SetRectEx(0, int32(math.Round(x)), s.GetWidth(), s.GetHeight(), false, xcc.AdjustLayout_No)
		} else {
			s.SetRectEx(int32(math.Round(x)), 0, s.GetWidth(), s.GetHeight(), false, xcc.AdjustLayout_No)
		}
	}
}

// carouselSlideOffset 返回第 i 页相对视口左/上缘的偏移。循环时折回 (-step, (n−1)step],
// 左侧留一格给"上一张从左边进来"; Go 的 math.Mod 符号跟被除数, 负数会留在负区间
// (循环时第一张永远进不来, 探针踩过), 所以先加一轮再取模。
func carouselSlideOffset(i int, curPx, step float64, n int, loop bool) float64 {
	x := float64(i)*step - curPx
	if loop && n > 0 {
		total := float64(n) * step
		x = math.Mod(x+step, total)
		if x < 0 {
			x += total
		}
		x -= step
	}
	return x
}

// layoutButtons 把两个圆形钮放进槽位: 横向在左右、垂直在上下, 主轴上居中。
func (c *Carousel) layoutButtons(w, h int32) {
	if c.prev == nil || c.next == nil {
		return
	}
	bw, bh := c.prev.GetWidth(), c.prev.GetHeight()
	if bw <= 0 {
		bw = carouselButtonSize
	}
	if bh <= 0 {
		bh = carouselButtonSize
	}
	if c.vertical {
		x := (w - bw) / 2
		c.prev.SetRectEx(x, 0, bw, bh, false, xcc.AdjustLayout_No)
		c.next.SetRectEx(x, h-bh, bw, bh, false, xcc.AdjustLayout_No)
		return
	}
	y := (h - bh) / 2
	c.prev.SetRectEx(0, y, bw, bh, false, xcc.AdjustLayout_No)
	c.next.SetRectEx(w-bw, y, bw, bh, false, xcc.AdjustLayout_No)
}

// syncButtons 按当前位置更新两个按钮的可用态(与 embla 的 canScrollPrev/Next 一致)。
func (c *Carousel) syncButtons() {
	if c.prev == nil || c.next == nil {
		return
	}
	n := len(c.slides)
	canPrev, canNext := false, false
	if n > 1 {
		canPrev = c.loop || c.pending > 0
		canNext = c.loop || c.pending < n-1
	}
	if c.prev.IsEnable() != canPrev {
		c.prev.Enable(canPrev)
		c.prev.Redraw(true)
	}
	if c.next.IsEnable() != canNext {
		c.next.Enable(canNext)
		c.next.Redraw(true)
	}
}

// bindInput 绑定窗口级的按下/移动/抬起与键盘, 以及捕获阶段的元素级移动。
//
// 元素事件**不冒泡**(探针实测: 指针在子元素上时父元素收不到 MouseMove/LButtonDown),
// 而幻灯片内容往往铺满整页, 所以拖拽必须从窗口级事件起步; 越过阈值后把鼠标捕获设到
// 视口上, 之后的移动/抬起由视口独占(窗口不再收到移动) —— 拖到窗口外也能继续跟手。
func (c *Carousel) bindInput() {
	hWnd := xc.XWidget_GetHWND(c.Handle)
	if hWnd == 0 {
		return
	}
	hWindow := xc.XC_hWindowFromHWnd(hWnd)
	if hWindow == 0 {
		return
	}
	c.hWindow = hWindow
	w := window.NewByHandle(hWindow)
	if w == nil {
		return
	}
	c.win = w

	w.AddEvent_LButtonDown(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		if hWindow != c.hWindow || len(c.slides) < 2 {
			return 0
		}
		if !c.hitViewport(pPt) {
			return 0
		}
		c.dragArmed = true
		c.dragging = false
		c.dragStart = c.axisOf(pPt)
		c.dragStartP = c.curPx
		vr := xc.RECT{}
		xc.XEle_GetWndClientRect(c.viewport.Handle, &vr)
		if c.vertical {
			c.vpOrigin = vr.Top
		} else {
			c.vpOrigin = vr.Left
		}
		return 0
	})
	w.AddEvent_MouseMove(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		if hWindow != c.hWindow || !c.dragArmed {
			return 0
		}
		if !c.dragging {
			if abs32(c.axisOf(pPt)-c.dragStart) < carouselDragSlop {
				return 0
			}
			c.beginDrag()
		}
		c.moveDrag(c.axisOf(pPt), nFlags)
		return 0
	})
	w.AddEvent_LButtonUp(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		if hWindow != c.hWindow {
			return 0
		}
		c.dragArmed = false
		if c.dragging {
			c.endDrag()
		}
		return 0
	})
	w.AddEvent_KeyDown(func(hWindow int, wParam, lParam uintptr, pbHandled *bool) int {
		if hWindow != c.hWindow || len(c.slides) < 2 {
			return 0
		}
		focus := xc.XWnd_GetFocusEle(hWindow)
		if focus == 0 || (focus != c.Handle && !xc.XEle_IsChildEle(c.Handle, focus)) {
			return 0
		}
		switch wParam {
		case 0x25: // VK_LEFT
			c.Prev()
			*pbHandled = true
		case 0x27: // VK_RIGHT
			c.Next()
			*pbHandled = true
		}
		return 0
	})

	// 捕获之后移动/抬起只送到视口: pPt 是视口客户区坐标, 加上按下时记下的原点换回窗口坐标。
	c.viewport.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if !c.dragging {
			return 0
		}
		if nFlags&1 == 0 { // MK_LBUTTON 已松开(在窗口外松手时没有 UP 事件)
			c.endDrag()
			return 0
		}
		c.moveDrag(c.axisOf(pPt)+c.vpOrigin, uint(nFlags))
		return 0
	})
	c.viewport.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		c.dragArmed = false
		if c.dragging {
			c.endDrag()
		}
		return 0
	})
	c.viewport.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		// 捕获期间的新一次按下: 先结束可能残留的拖拽(在窗口外松手后没收到 UP 的情形)。
		// 只处理"正在拖"的情况: 普通按下时窗口级处理器刚把 dragArmed 置上, 这里不能清掉
		// (窗口事件与元素事件会先后各来一次, 清掉就永远拖不动 —— 探针踩过)。
		if c.dragging {
			c.endDrag()
			c.dragArmed = false
		}
		return 0
	})
}

// hitViewport 判断窗口客户区坐标点是否落在视口内。
func (c *Carousel) hitViewport(pPt *xc.POINT) bool {
	var r xc.RECT
	xc.XEle_GetWndClientRect(c.viewport.Handle, &r)
	return pPt.X >= r.Left && pPt.X < r.Right && pPt.Y >= r.Top && pPt.Y < r.Bottom
}

// axisOf 取主轴坐标。
func (c *Carousel) axisOf(pPt *xc.POINT) int32 {
	if c.vertical {
		return pPt.Y
	}
	return pPt.X
}

func (c *Carousel) beginDrag() {
	c.indexInit = true
	c.moving = false
	c.dragging = true
	c.stopAutoplay(true)
	xui.DropTransitions(c.viewport.Handle)
	if c.win != nil {
		c.win.SetCaptureEle(c.viewport.Handle)
	}
}

// moveDrag 跟手更新位移; 非循环时两端加阻尼, 松手回弹。
func (c *Carousel) moveDrag(pos int32, nFlags uint) {
	if nFlags&1 == 0 {
		c.endDrag()
		return
	}
	px := c.dragStartP - float64(pos-c.dragStart)
	n := len(c.slides)
	if !c.loop && n > 1 && c.step > 0 {
		minPx, maxPx := 0.0, float64(n-1)*c.step
		if px < minPx {
			px = minPx + (px-minPx)*0.35
		} else if px > maxPx {
			px = maxPx + (px-maxPx)*0.35
		}
	}
	c.curPx = px
	c.positionSlides()
	c.viewport.Redraw(true)
}

// endDrag 松手: 吸附到最近一页(按当前视觉位置算, 循环模式下取模)。
func (c *Carousel) endDrag() {
	c.dragging = false
	c.dragArmed = false
	if c.win != nil {
		c.win.SetCaptureEle(0)
	}
	n := len(c.slides)
	if n == 0 || c.step <= 0 {
		return
	}
	target := int(math.Round(c.curPx / c.step))
	if c.loop {
		target = ((target % n) + n) % n
	} else if target < 0 {
		target = 0
	} else if target > n-1 {
		target = n - 1
	}
	to := float64(target) * c.step
	if c.loop && n > 1 {
		total := float64(n) * c.step
		for to-c.curPx > total/2 {
			to -= total
		}
		for c.curPx-to > total/2 {
			to += total
		}
	}
	if math.Abs(to-c.curPx) < 0.5 {
		c.curPx = to
		c.settle(target)
		return
	}
	c.pending = target
	from := c.curPx
	c.moving = true
	xui.DropTransitions(c.viewport.Handle)
	key := xui.NextPresenceKey()
	xui.Presence(c.viewport.Handle, key, true, c.duration,
		func(v float64) {
			c.curPx = from + (to-from)*v
			c.positionSlides()
		},
		func(v float64) {
			c.moving = false
			c.settle(target)
		})
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
