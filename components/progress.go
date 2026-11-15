package ui

import (
	"fmt"
	"math"
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Progress 自绘进度条: 载体为原生 ProgressBar(提供 range/pos 与进度变化事件),
// 外观全部自绘。
//
// 与 shadcn progress.tsx(radix-nova) 对齐:
//   - 轨道: h-1(4px) w-full rounded-full bg-muted;
//   - 指示器: bg-primary, 宽度 = 轨道宽 x 百分比(overflow-hidden 由轨道圆角天然形成);
//     值变化时宽度按 150ms 过渡(对应 Indicator 的 transition-all, 运行时过渡表驱动);
//   - value/max: 百分比 = value / max(radix 的 max 默认 100), 也支持 value=null 的
//     不确定态(indeterminate): 指示器按"短条左右往返"呈现, 不报告百分比;
//   - 指示器颜色可覆盖(radix 的 Indicator 可加自定义 className)。
type Progress struct {
	*widget.ProgressBar
	value         float64
	min           float64
	max           float64
	indeterminate bool
	indicator     func(*xui.Theme) xui.Color // 指示器颜色解析器(nil = Primary)
	track         func(*xui.Theme) xui.Color // 轨道颜色解析器(nil = Muted)
	radius        int32                      // 圆角覆盖(0 = 高度的一半, 胶囊)
	displayPct    float64                    // 当前显示的百分比(值过渡的插值结果)
	animSeq       uint64                     // 过渡代次: 新值到来后旧过渡不再写值
}

// progressRange 是内部进度范围: 用千分比保留 0.1% 精度, 对外 API 是实际值域。
const progressRange int32 = 1000

// progressIndeterminatePercent 是不确定态指示器占轨道的百分比(短条往返的视觉长度)。
const progressIndeterminatePercent int32 = 30

// progressTransition 是值变化的宽度过渡时长(上游 Indicator 的 transition-all,
// tailwind 默认 150ms)。过渡由运行时 PresenceOnce 驱动, SetAnimationScale 同样生效。
const progressTransition = 150 * time.Millisecond

// ProgressOption 进度条配置项。
type ProgressOption func(*Progress)

// WithProgressMax 设置最大值(对应 radix 的 max), 默认 100。
func WithProgressMax(max float64) ProgressOption {
	return func(p *Progress) {
		if max > 0 {
			p.max = max
		}
	}
}

// WithProgressMin 设置下限(对应 Base UI Progress Root 的 min, 默认 0):
// 百分比按 (value-min)/(max-min) 算, 值域可以是负数(如 -50 ~ 50)。
func WithProgressMin(min float64) ProgressOption {
	return func(p *Progress) { p.min = min }
}

// WithProgressIndeterminate 初始为不确定态(radix 的 value=null): 不显示具体百分比,
// 指示器按短条沿轨道往返。
func WithProgressIndeterminate() ProgressOption {
	return func(p *Progress) { p.indeterminate = true }
}

// WithProgressIndicatorColor 覆盖指示器颜色(对应 radix Indicator 的 className)。
// 固定色直接传, 需要跟随主题时用 WithProgressIndicatorColorFunc。
func WithProgressIndicatorColor(c xui.Color) ProgressOption {
	return func(p *Progress) {
		p.indicator = func(*xui.Theme) xui.Color { return c }
	}
}

// WithProgressIndicatorColorFunc 用主题解析器取指示器颜色(每次绘制实时求值)。
func WithProgressIndicatorColorFunc(fn func(*xui.Theme) xui.Color) ProgressOption {
	return func(p *Progress) {
		if fn != nil {
			p.indicator = fn
		}
	}
}

// WithProgressTrackColor 覆盖轨道颜色(上游 Track 是独立部件, 可传 className 改底色)。
func WithProgressTrackColor(c xui.Color) ProgressOption {
	return func(p *Progress) { p.track = func(*xui.Theme) xui.Color { return c } }
}

// WithProgressTrackColorFunc 用主题解析器取轨道颜色(每次绘制实时求值)。
func WithProgressTrackColorFunc(fn func(*xui.Theme) xui.Color) ProgressOption {
	return func(p *Progress) {
		if fn != nil {
			p.track = fn
		}
	}
}

// WithProgressRadius 覆盖轨道与指示器的圆角(默认 = 高度的一半, 即胶囊; 传 0 恢复胶囊)。
// 上游通过 Track / Indicator 的 className 改 rounded-*, Go 侧收敛成这一个数。
func WithProgressRadius(r int32) ProgressOption {
	return func(p *Progress) {
		if r > 0 {
			p.radius = r
		}
	}
}

// NewProgress 创建进度条。cy 传 0 时使用 shadcn 默认的 4px 高度。
func NewProgress(x, y, cx, cy int32, hParent int, opts ...ProgressOption) *Progress {
	if cy <= 0 {
		cy = 4
	}
	p := &Progress{max: 100}
	for _, o := range opts {
		o(p)
	}
	if p.max <= p.min {
		p.max = p.min + 100
	}
	pb := widget.NewProgressBar(x, y, cx, cy, hParent)
	if pb == nil {
		return nil
	}
	p.ProgressBar = pb
	pb.EnableBkTransparent(true) // 灭掉原生外观
	pb.EnableShowText(false)
	pb.SetRange(progressRange)
	pb.AddEvent_Paint(p.onPaint)
	pb.AddEvent_ProgressBar_Change(func(hEle int, pos int32, pbHandled *bool) int {
		pb.Redraw(false)
		return 0
	})
	xui.AttachWindow(pb.Handle)
	p.syncPos()
	p.displayPct = p.clampPct(p.percentOf(p.value))
	return p
}

// Value 返回当前进度(Min ~ Max)。
func (p *Progress) Value() float64 { return p.value }

// Min 返回下限(默认 0)。
func (p *Progress) Min() float64 { return p.min }

// Max 返回最大值(默认 100)。
func (p *Progress) Max() float64 { return p.max }

// Percent 返回百分比(0~100), 不确定态时返回 0。
func (p *Progress) Percent() float64 {
	if p.indeterminate {
		return 0
	}
	return p.percentOf(p.value)
}

// percentOf 把绝对值换算成 0~100 的百分比(值域 min..max)。
func (p *Progress) percentOf(v float64) float64 {
	span := p.max - p.min
	if span <= 0 {
		return 0
	}
	return (v - p.min) / span * 100
}

// SetValue 设置进度, v 会被夹取到 Min ~ Max。
func (p *Progress) SetValue(v float64) *Progress {
	if v < p.min {
		v = p.min
	} else if v > p.max {
		v = p.max
	}
	if v == p.value {
		return p
	}
	p.value = v
	p.syncPos()
	p.retarget()
	p.Redraw(false)
	return p
}

// SetMax 设置最大值(对应 radix 的 max): **value 保持绝对值不变**, 百分比按新上限重算。
func (p *Progress) SetMax(max float64) *Progress {
	if max <= p.min {
		max = p.min + 1
	}
	if max == p.max {
		return p
	}
	p.max = max
	if p.value > max {
		p.value = max
	}
	p.syncPos()
	p.retarget()
	p.Redraw(false)
	return p
}

// SetMin 设置下限(对应 Base UI 的 min): value 保持绝对值不变, 低于新下限时抬到下限,
// 百分比按新值域重算。
func (p *Progress) SetMin(min float64) *Progress {
	if min == p.min {
		return p
	}
	p.min = min
	if p.max <= min {
		p.max = min + 1
	}
	if p.value < min {
		p.value = min
	}
	p.syncPos()
	p.retarget()
	p.Redraw(false)
	return p
}

// Indeterminate 返回是否处于不确定态。
func (p *Progress) Indeterminate() bool { return p.indeterminate }

// SetIndeterminate 设置不确定态(radix 的 value=null): 不显示百分比, 指示器按短条往返。
// 为 false 时恢复按 Value/Max 绘制。
func (p *Progress) SetIndeterminate(v bool) *Progress {
	if p.indeterminate == v {
		return p
	}
	p.indeterminate = v
	p.animSeq++ // 停掉未完成的值过渡
	if v {
		p.displayPct = p.clampPct(p.percentOf(p.value))
	}
	p.Redraw(false)
	return p
}

// clampPct 把百分比夹到 0~100。
func (p *Progress) clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// retarget 让显示百分比从当前位置过渡到当前值(对应 Indicator 的 transition-all)。
//
// 用运行时 PresenceOnce 驱动: 每次值变化开始一段带新键的过渡, 到位条目自动回收;
// 旧过渡还没跑完时由 animSeq 让它停止写值 —— 否则新旧两段会同时改 displayPct,
// 宽度来回跳。过渡与"弹层进场"共用运行时过渡表, 不自起定时器。
func (p *Progress) retarget() {
	to := p.clampPct(p.percentOf(p.value))
	from := p.displayPct
	p.animSeq++
	seq := p.animSeq
	if p.indeterminate || from == to {
		p.displayPct = to
		return
	}
	xui.PresenceOnce(p.Handle, true, progressTransition, func(v float64) {
		if seq != p.animSeq {
			return
		}
		p.displayPct = from + (to-from)*v
	}, func(float64) {
		if seq != p.animSeq {
			return
		}
		p.displayPct = to
		// 到位帧的重绘发生在 done 之前, 终值要再补一次(同 anim.go 的 settledNow 语义)。
		p.Redraw(false)
	})
}

// SetIndicatorColor 覆盖指示器颜色(固定色; 与主题无关)。
func (p *Progress) SetIndicatorColor(c xui.Color) *Progress {
	p.indicator = func(*xui.Theme) xui.Color { return c }
	p.Redraw(false)
	return p
}

// SetIndicatorColorFunc 用主题解析器取指示器颜色; 传 nil 恢复 Primary。
func (p *Progress) SetIndicatorColorFunc(fn func(*xui.Theme) xui.Color) *Progress {
	p.indicator = fn
	p.Redraw(false)
	return p
}

// SetTrackColorFunc 用主题解析器取轨道颜色; 传 nil 恢复 Muted。
func (p *Progress) SetTrackColorFunc(fn func(*xui.Theme) xui.Color) *Progress {
	if p == nil {
		return p
	}
	p.track = fn
	p.Redraw(false)
	return p
}

// SetRadius 覆盖圆角(<=0 恢复胶囊)。
func (p *Progress) SetRadius(r int32) *Progress {
	if p == nil {
		return p
	}
	if r < 0 {
		r = 0
	}
	if p.radius == r {
		return p
	}
	p.radius = r
	p.Redraw(false)
	return p
}

// OnChange 注册进度变化回调(v 为 0 ~ Max 的实际值)。
func (p *Progress) OnChange(fn func(v float64)) *Progress {
	if fn == nil {
		return p
	}
	p.AddEvent_ProgressBar_Change(func(hEle int, pos int32, pbHandled *bool) int {
		fn(p.value)
		return 0
	})
	return p
}

func (p *Progress) syncPos() {
	pos := int32(p.percentOf(p.value)/100*float64(progressRange) + 0.5)
	if pos < 0 {
		pos = 0
	} else if pos > progressRange {
		pos = progressRange
	}
	p.SetPos(pos)
}

func (p *Progress) color(t *xui.Theme) xui.Color {
	if p.indicator != nil {
		return p.indicator(t)
	}
	return t.Primary
}

func (p *Progress) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	pt := xui.BeginPaint(hEle, hDraw)
	if pt == nil {
		return 0
	}
	*pbHandled = true // 轨道与指示器全部自绘
	t := xui.Current()

	h := pt.H()
	radius := h / 2
	if p.radius > 0 && p.radius < radius {
		radius = p.radius
	}

	// 轨道: bg-muted(可覆盖)
	trackColor := t.Muted
	if p.track != nil {
		trackColor = p.track(t)
	}
	pt.FillRound(&pt.Rc, trackColor, radius)

	trackW := pt.W()
	if trackW <= 0 {
		return 0
	}

	var barW int32
	var left int32
	if p.indeterminate {
		// 不确定态(radix 的 value=null): 短条沿轨道左右往返, 由运行时 Spin 驱动。
		// 三角波(0→1→0)比正弦更像原生的 indeterminate 进度条, 且两端有停顿感。
		barW = trackW * progressIndeterminatePercent / 100
		if barW < 8 {
			barW = 8
		}
		if barW > trackW {
			barW = trackW
		}
		phase := xui.Spin(hEle, true)
		f := 1 - math.Abs(2*phase-1)
		left = pt.Rc.Left + int32(math.Round(f*float64(trackW-barW)))
	} else {
		// 值变化时宽度按过渡插值(上游 Indicator 的 transition-all): displayPct
		// 由 retarget 启动的运行时过渡逐帧更新, 这里只读当前显示值。
		barW = int32(float64(trackW) * p.displayPct / 100)
		if barW <= 0 {
			return 0
		}
		left = pt.Rc.Left
	}

	// 指示器: 宽度按进度; 圆角随宽度收缩, 保证窄进度也是胶囊形
	if r := barW / 2; r < radius {
		radius = r
	}
	bar := xc.RECT{
		Left:   left,
		Top:    pt.Rc.Top,
		Right:  left + barW,
		Bottom: pt.Rc.Bottom,
	}
	pt.FillRound(&bar, p.color(t), radius)
	return 0
}

// ===== 独立摆放的标题 / 数值(对应 ProgressLabel / ProgressValue 部件) =====
//
// ProgressField 是"标题 + 数值 + 条"的固定组合; 上游把 ProgressLabel / ProgressValue 列成
// 可自由摆放的子部件(标签在条上方、数值在条右侧、数值单独放一行等)。这两个构造函数给出
// 同样的自由度: 尺寸与位置交给调用方的布局, 数值文本自己跟着 Progress 刷新。

// NewProgressLabel 创建独立的进度标题(对应 ProgressLabel 部件): medium 字重的普通 Label,
// 摆放位置由调用方决定(通常配 xui.ItemWeight 或 xui.ItemFill)。
func NewProgressLabel(x, y, cx, cy int32, text string, hParent int) *Label {
	if cy <= 0 {
		cy = 20
	}
	l := NewLabel(x, y, cx, cy, text, hParent)
	if l == nil {
		return nil
	}
	return l.SetMedium()
}

// ProgressValue 是独立摆放的进度数值文本(对应 ProgressValue 部件): 绑定一个 Progress,
// 值变化时自动刷新, 文本格式可覆盖(默认四舍五入到整数百分比)。
//
// 它只是"订阅者", 不参与 Progress 的布局: 放在条的右侧/下方都行。
type ProgressValue struct {
	*Label
	bar    *Progress
	format func(v, max float64) string
}

// NewProgressValue 创建并绑定到 bar 的进度数值文本。cy 传 0 用 20。
// bar 为 nil 时退化成普通 Label(始终显示空文本)。
func NewProgressValue(bar *Progress, x, y, cx, cy int32, hParent int) *ProgressValue {
	if cy <= 0 {
		cy = 20
	}
	l := NewLabel(x, y, cx, cy, "", hParent)
	if l == nil {
		return nil
	}
	l.SetMuted()
	pv := &ProgressValue{Label: l, bar: bar}
	if bar != nil {
		bar.OnChange(func(float64) { pv.Refresh() })
	}
	pv.Refresh()
	return pv
}

// SetFormat 自定义数值文本(参数是当前值与最大值; 与 ProgressField 的
// WithProgressFieldFormat 同一口径)。传 nil 恢复默认百分比。
func (pv *ProgressValue) SetFormat(fn func(v, max float64) string) *ProgressValue {
	if pv == nil {
		return pv
	}
	pv.format = fn
	pv.Refresh()
	return pv
}

// Refresh 按当前进度重算文本(程序化改过值/格式后调用; 值变化事件已自动处理)。
func (pv *ProgressValue) Refresh() *ProgressValue {
	if pv == nil || pv.bar == nil {
		return pv
	}
	text := ""
	switch {
	case pv.bar.Indeterminate():
		text = "…" // 不确定态不报告百分比(同 ProgressField)
	case pv.format != nil:
		text = pv.format(pv.bar.Value(), pv.bar.Max())
	default:
		text = fmt.Sprintf("%.0f%%", pv.bar.Percent())
	}
	if pv.GetText() != text {
		pv.SetText(text).Redraw()
	}
	return pv
}

// ===== ProgressField: 标题 + 百分比 + 进度条(对应 shadcn 文档的 Labels 示例) =====

// ProgressField 是"标题行(左标题 / 右百分比) + 进度条"的复合形态, 对应 shadcn
// progress 文档的 Labels 示例(Field + ProgressLabel + ProgressValue 的组合)。
//
// 高度默认 = 标题行 20 + 间距 8 + 进度条 4 = 32; 宽度由调用方的布局项决定
// (通常 `xui.ItemFill(pf)`)。内部是纵向布局容器, 标题/数值/进度条自动跟随宽度。
type ProgressField struct {
	*widget.LayoutEle
	title         *Label
	value         *Label
	bar           *Progress
	format        func(v, max float64) string
	titleText     string
	initial       float64
	min           float64
	max           float64
	indeterminate bool
	indicator     func(*xui.Theme) xui.Color
	track         func(*xui.Theme) xui.Color
	radius        int32
}

// ProgressFieldOption 复合进度条配置项。
type ProgressFieldOption func(*ProgressField)

// WithProgressFieldTitle 设置标题文本(默认 "Progress")。
func WithProgressFieldTitle(text string) ProgressFieldOption {
	return func(pf *ProgressField) { pf.titleText = text }
}

// WithProgressFieldValue 设置初始值。
func WithProgressFieldValue(v float64) ProgressFieldOption {
	return func(pf *ProgressField) { pf.initial = v }
}

// WithProgressFieldMax 设置最大值(默认 100)。
func WithProgressFieldMax(max float64) ProgressFieldOption {
	return func(pf *ProgressField) { pf.max = max }
}

// WithProgressFieldMin 设置下限(默认 0), 语义同 WithProgressMin。
func WithProgressFieldMin(min float64) ProgressFieldOption {
	return func(pf *ProgressField) { pf.min = min }
}

// WithProgressFieldFormat 自定义右侧数值文本(默认四舍五入到整数百分比, 如 "66%")。
func WithProgressFieldFormat(fn func(v, max float64) string) ProgressFieldOption {
	return func(pf *ProgressField) {
		if fn != nil {
			pf.format = fn
		}
	}
}

// WithProgressFieldIndeterminate 复合进度条初始为不确定态。
func WithProgressFieldIndeterminate() ProgressFieldOption {
	return func(pf *ProgressField) { pf.indeterminate = true }
}

// WithProgressFieldIndicatorColor 覆盖指示器颜色(固定色)。
func WithProgressFieldIndicatorColor(c xui.Color) ProgressFieldOption {
	return func(pf *ProgressField) { pf.indicator = func(*xui.Theme) xui.Color { return c } }
}

// WithProgressFieldTrackColor 覆盖轨道颜色。
func WithProgressFieldTrackColor(c xui.Color) ProgressFieldOption {
	return func(pf *ProgressField) { pf.track = func(*xui.Theme) xui.Color { return c } }
}

// WithProgressFieldRadius 覆盖圆角(默认胶囊)。
func WithProgressFieldRadius(r int32) ProgressFieldOption {
	return func(pf *ProgressField) {
		if r > 0 {
			pf.radius = r
		}
	}
}

// NewProgressField 创建带标题与百分比文本的进度条。cy 传 0 时用 32(标题 20 + 间距 8 + 条 4)。
func NewProgressField(x, y, cx, cy int32, hParent int, opts ...ProgressFieldOption) *ProgressField {
	pf := &ProgressField{max: 100, titleText: "Progress"}
	for _, o := range opts {
		o(pf)
	}
	if pf.max <= pf.min {
		pf.max = pf.min + 100
	}
	if pf.titleText == "" {
		pf.titleText = "Progress"
	}
	if cy <= 0 {
		cy = 32
	}
	root := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if root == nil {
		return nil
	}
	pf.LayoutEle = root
	root.EnableHorizon(false)
	root.EnableLayout(true)
	root.EnableBkTransparent(true)
	root.EnableAutoWrap(false) // 关掉"纵向溢出换列"(见 DESIGN 第八节 41)
	root.SetSpace(8)

	// 标题行: 标题占剩余宽, 百分比固定 48 宽右对齐
	row := xui.HStack(root.Handle, 20, 8)
	pf.title = NewLabel(0, 0, 0, 20, pf.titleText, row.Handle)
	if pf.title == nil {
		return nil
	}
	pf.title.SetMedium() // 对应上游 FieldLabel 的 font-medium
	xui.ItemWeight(pf.title, 1)
	pf.value = NewLabel(0, 0, 48, 20, "", row.Handle).SetMuted()
	xui.ItemWidth(pf.value, 48)
	// 右对齐: 直接改原生 Button 的对齐旗标(保留 Label 构造时设的 NoWrap/省略号)
	pf.value.SetTextAlign(xcc.TextAlignFlag_Right | xcc.TextAlignFlag_Vcenter |
		xcc.TextFormatFlag_NoWrap | xcc.TextTrimming_EllipsisCharacter)

	barOpts := []ProgressOption{WithProgressMax(pf.max)}
	if pf.min != 0 {
		barOpts = append(barOpts, WithProgressMin(pf.min))
	}
	if pf.indeterminate {
		barOpts = append(barOpts, WithProgressIndeterminate())
	}
	pf.bar = NewProgress(0, 0, 0, 4, root.Handle, barOpts...)
	if pf.bar == nil {
		return nil
	}
	xui.ItemFill(pf.bar)
	if pf.indicator != nil {
		pf.bar.SetIndicatorColorFunc(pf.indicator)
	}
	if pf.track != nil {
		pf.bar.SetTrackColorFunc(pf.track)
	}
	if pf.radius > 0 {
		pf.bar.SetRadius(pf.radius)
	}
	pf.bar.SetValue(pf.initial)
	pf.syncValueText(pf.bar.Value(), pf.bar.Max())
	pf.bar.OnChange(func(v float64) { pf.syncValueText(v, pf.bar.Max()) })
	return pf
}

// Bar 返回内部的 Progress, 需要更细的控制时用它。
func (pf *ProgressField) Bar() *Progress { return pf.bar }

// TitleLabel 返回标题 Label。
func (pf *ProgressField) TitleLabel() *Label { return pf.title }

// ValueLabel 返回右侧百分比 Label。
func (pf *ProgressField) ValueLabel() *Label { return pf.value }

// SetTitle 设置标题文本。
func (pf *ProgressField) SetTitle(text string) *ProgressField {
	pf.title.SetText(text).Redraw()
	return pf
}

// SetValue 设置进度值。
func (pf *ProgressField) SetValue(v float64) *ProgressField {
	pf.bar.SetValue(v)
	pf.syncValueText(pf.bar.Value(), pf.bar.Max())
	return pf
}

// Value 返回当前值。
func (pf *ProgressField) Value() float64 { return pf.bar.Value() }

// SetMax 设置最大值(value 保持绝对值)。
func (pf *ProgressField) SetMax(max float64) *ProgressField {
	pf.bar.SetMax(max)
	pf.syncValueText(pf.bar.Value(), pf.bar.Max())
	return pf
}

// SetIndeterminate 设置不确定态。
func (pf *ProgressField) SetIndeterminate(v bool) *ProgressField {
	pf.bar.SetIndeterminate(v)
	pf.syncValueText(pf.bar.Value(), pf.bar.Max())
	return pf
}

// OnChange 注册进度变化回调(转发内部 Progress)。
func (pf *ProgressField) OnChange(fn func(v float64)) *ProgressField {
	pf.bar.OnChange(fn)
	return pf
}

// syncValueText 刷新右侧数值文本; 不确定态显示 "…"(不报告百分比)。
func (pf *ProgressField) syncValueText(v, max float64) {
	text := ""
	if pf.bar.Indeterminate() {
		text = "…"
	} else if pf.format != nil {
		text = pf.format(v, max)
	} else {
		text = fmt.Sprintf("%.0f%%", pf.bar.Percent())
	}
	if pf.value.GetText() != text {
		pf.value.SetText(text).Redraw()
	}
}
