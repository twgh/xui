package ui

import (
	"fmt"
	"math"

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
//   - value/max: 百分比 = value / max(radix 的 max 默认 100), 也支持 value=null 的
//     不确定态(indeterminate): 指示器按"短条左右往返"呈现, 不报告百分比;
//   - 指示器颜色可覆盖(radix 的 Indicator 可加自定义 className)。
type Progress struct {
	*widget.ProgressBar
	value         float64
	max           float64
	indeterminate bool
	indicator     func(*xui.Theme) xui.Color // 指示器颜色解析器(nil = Primary)
}

// progressRange 是内部进度范围: 用千分比保留 0.1% 精度, 对外 API 是实际值域。
const progressRange int32 = 1000

// progressIndeterminatePercent 是不确定态指示器占轨道的百分比(短条往返的视觉长度)。
const progressIndeterminatePercent int32 = 30

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

// NewProgress 创建进度条。cy 传 0 时使用 shadcn 默认的 4px 高度。
func NewProgress(x, y, cx, cy int32, hParent int, opts ...ProgressOption) *Progress {
	if cy <= 0 {
		cy = 4
	}
	p := &Progress{max: 100}
	for _, o := range opts {
		o(p)
	}
	if p.max <= 0 {
		p.max = 100
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
	return p
}

// Value 返回当前进度(0 ~ Max)。
func (p *Progress) Value() float64 { return p.value }

// Max 返回最大值(默认 100)。
func (p *Progress) Max() float64 { return p.max }

// Percent 返回百分比(0~100), 不确定态时返回 0。
func (p *Progress) Percent() float64 {
	if p.indeterminate || p.max <= 0 {
		return 0
	}
	return p.value / p.max * 100
}

// SetValue 设置进度, v 会被夹取到 0 ~ Max。
func (p *Progress) SetValue(v float64) *Progress {
	if v < 0 {
		v = 0
	} else if v > p.max {
		v = p.max
	}
	if v == p.value {
		return p
	}
	p.value = v
	p.syncPos()
	p.Redraw(false)
	return p
}

// SetMax 设置最大值(对应 radix 的 max): **value 保持绝对值不变**, 百分比按新上限重算。
func (p *Progress) SetMax(max float64) *Progress {
	if max <= 0 {
		max = 1
	}
	if max == p.max {
		return p
	}
	p.max = max
	if p.value > max {
		p.value = max
	}
	p.syncPos()
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
	p.Redraw(false)
	return p
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
	pos := int32(p.value/p.max*float64(progressRange) + 0.5)
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

	// 轨道: bg-muted
	pt.FillRound(&pt.Rc, t.Muted, radius)

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
		barW = int32(float64(trackW) * p.value / p.max)
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
	max           float64
	indeterminate bool
	indicator     func(*xui.Theme) xui.Color
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

// NewProgressField 创建带标题与百分比文本的进度条。cy 传 0 时用 32(标题 20 + 间距 8 + 条 4)。
func NewProgressField(x, y, cx, cy int32, hParent int, opts ...ProgressFieldOption) *ProgressField {
	pf := &ProgressField{max: 100, titleText: "Progress"}
	for _, o := range opts {
		o(pf)
	}
	if pf.max <= 0 {
		pf.max = 100
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
		text = fmt.Sprintf("%.0f%%", v/max*100)
	}
	if pf.value.GetText() != text {
		pf.value.SetText(text).Redraw()
	}
}
