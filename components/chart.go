package ui

import (
	"math"
	"strconv"
	"strings"

	"github.com/twgh/xcgui/wapi/d2d"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Chart 自绘图表: 单个 Element 承担全部几何 + 绘制 + 命中, 不建子元素。
//
// 与 shadcn chart(recharts 的包装)对应的是"数据形状 + 配色 + 图例 + 悬停提示"这套语义,
// 不是 DOM/React 的 `ChartContainer` —— 真正干活的 recharts 在 Go 侧没有等价物, 所以这里
// 按桌面场景重写了一套精简引擎:
//   - 四种图: 柱状(ChartBar)、折线(ChartLine)、面积(ChartArea)、饼/环(ChartPie / ChartDonut);
//   - 柱状/面积支持堆叠(`WithChartStacked`), 折线/面积支持平滑(`WithChartCurve`);
//   - 数值轴自动取"整齐刻度"(1/2/2.5/5 × 10^n), 也可显式给值域(`WithChartYDomain`);
//   - 图例默认在底部居中, 点击可临时隐藏/恢复一项(饼图隐藏的是一块扇区);
//   - 悬停出提示卡(分类名 + 每路数值), 直角坐标系还画一条竖向指示线;
//   - 颜色默认取主题 `Chart1..5` 轮转, 每路数据也可显式指定(ChartSeries.Color)。
//
// 绘制坐标全部是逻辑像素(见 DESIGN 第六节): 元素被布局拉伸时只按新宽高重算几何,
// 不做任何 DPI 预缩放。
//
// 已知边界: 不支持横向柱状图 / 雷达图 / 径向图, 也没有进场动画 —— recharts 的
// `isAnimationActive` 依赖 CSS 过渡, 而这里是逐帧改几何 + 整窗重绘, 动画期间会把
// 整个窗口的重绘排队(同 DESIGN 第八节 73 根因 D 的教训)。
type Chart struct {
	*widget.Element

	kind    ChartKind
	labels  []string
	series  []ChartSeries
	hidden  []bool // 直角坐标系 = 每路 series; 饼图 = 每个分类
	yFormat func(float64) string

	stacked    bool
	curved     bool
	dots       bool
	showGrid   bool
	showTip    bool
	legend     bool
	legendSet  bool
	barLabels  bool
	yMin, yMax float64
	innerRatio float64

	centerTitle string
	centerSub   string

	hover  int // 悬停的分类(饼图 = 扇区), -1 = 无
	hoverX int32
	hoverY int32

	onClick func(cat, series int, value float64)

	host  int // 宿主 xcgui 窗口
	watch map[int]bool
}

// ChartKind 是图表的形态。
type ChartKind int

const (
	// ChartBar 柱状图: 每个分类一组柱子(多路并排), `WithChartStacked` 后堆叠。
	ChartBar ChartKind = iota
	// ChartLine 折线图: 每个分类一个点, 多路数据各一条线。
	ChartLine
	// ChartArea 面积图: 折线 + 半透明填充, 可堆叠。
	ChartArea
	// ChartPie 饼图: 只用第一路数据, 每个分类一块扇区。
	ChartPie
	// ChartDonut 环形图: 同饼图, 中心镂空并可放两行文字(`WithChartCenter`)。
	ChartDonut
)

// ChartSeries 是一路数据。
//
// 直角坐标系里一个 ChartSeries = 图例里的一项、一条线/一组柱子; 饼图只用第一路,
// 它的 Values[i] 是第 i 个分类的数值(饼图的图例项是分类本身)。
type ChartSeries struct {
	Label  string    // 图例文字; 空则用 "系列 N"
	Values []float64 // 与 Chart 的 labels 一一对应
	Color  xui.Color // 零值 = 按主题 Chart1..5 轮转
}

// ChartOption 图表配置项。
type ChartOption func(*Chart)

// chartDims 是图表的内部间距(逻辑像素), 取自 shadcn 示例的默认留白。
const (
	chartPad       int32 = 8  // 四周内边距
	chartAxisGap   int32 = 6  // 数值轴文字与绘图区之间
	chartXLabelH   int32 = 18 // 分类文字占用的高度
	chartLegendH   int32 = 18 // 图例行高度
	chartLegendGap int32 = 8  // 绘图区与图例之间
	chartLineW     int32 = 2  // 折线/面积描边宽度
	chartDotR      int32 = 4  // 点半径(shadcn r=4)
	chartBarRadius int32 = 4  // 柱子数据端圆角(shadcn radius={4})
	chartTipPad    int32 = 8  // 提示卡内边距
	chartTipRowH   int32 = 18 // 提示卡行高
	chartLabelPx   int32 = 12 // 轴/图例/提示卡字号(px)
)

// WithChartKind 指定图表形态(默认柱状图)。
func WithChartKind(k ChartKind) ChartOption {
	return func(c *Chart) { c.kind = k }
}

// WithChartData 设置分类与数据。series 的顺序就是图例顺序; 颜色留空时按主题色板轮转。
func WithChartData(labels []string, series ...ChartSeries) ChartOption {
	return func(c *Chart) { c.setData(labels, series) }
}

// WithChartStacked 柱状/面积图堆叠(折线与饼图忽略)。
func WithChartStacked() ChartOption {
	return func(c *Chart) { c.stacked = true }
}

// WithChartCurve 折线/面积用平滑曲线(默认直线段)。
func WithChartCurve() ChartOption {
	return func(c *Chart) { c.curved = true }
}

// WithChartDots 折线/面积在数据点上画圆点(默认不画, 与 shadcn 的 "Dots" 示例一致)。
func WithChartDots() ChartOption {
	return func(c *Chart) { c.dots = true }
}

// WithChartGrid 是否画横向网格线(默认画)。
func WithChartGrid(on bool) ChartOption {
	return func(c *Chart) { c.showGrid = on }
}

// WithChartTooltip 是否画悬停提示卡(默认画)。
func WithChartTooltip(on bool) ChartOption {
	return func(c *Chart) { c.showTip = on }
}

// WithChartLegend 是否画图例。未显式设置时: 多路数据画、单路不画; 饼图按分类数判断。
func WithChartLegend(on bool) ChartOption {
	return func(c *Chart) {
		c.legend = on
		c.legendSet = true
	}
}

// WithChartBarLabels 柱顶显示数值(shadcn 的 "Label" 示例)。
func WithChartBarLabels() ChartOption {
	return func(c *Chart) { c.barLabels = true }
}

// WithChartYFormat 自定义数值轴/提示卡/柱顶的数值文本(默认千分位 + 最多两位小数)。
func WithChartYFormat(fn func(float64) string) ChartOption {
	return func(c *Chart) {
		if fn != nil {
			c.yFormat = fn
		}
	}
}

// WithChartYDomain 显式指定数值轴范围(lo >= hi 时回到自动)。
// 自动模式下 0 一定在范围内(全正数据的基线就是 0, 与 recharts 的 domain=[0,'auto'] 一致)。
func WithChartYDomain(lo, hi float64) ChartOption {
	return func(c *Chart) {
		if lo < hi {
			c.yMin, c.yMax = lo, hi
		}
	}
}

// WithChartInnerRatio 环形图镂空比例(0~0.9, 默认 0.6)。饼图忽略。
func WithChartInnerRatio(r float64) ChartOption {
	return func(c *Chart) { c.innerRatio = chartClampRatio(r) }
}

// WithChartCenter 环形图中心的两行文字(标题 + 说明), 传空串清除。
func WithChartCenter(title, sub string) ChartOption {
	return func(c *Chart) {
		c.centerTitle, c.centerSub = title, sub
	}
}

// NewChart 创建图表。cx/cy 传 0 时用 480×240。
func NewChart(x, y, cx, cy int32, hParent int, opts ...ChartOption) *Chart {
	c := &Chart{
		kind:       ChartBar,
		showGrid:   true,
		showTip:    true,
		innerRatio: 0.6,
		hover:      -1,
		watch:      map[int]bool{},
	}
	for _, o := range opts {
		o(c)
	}
	if !c.legendSet {
		if c.isPie() {
			c.legend = len(c.labels) > 1
		} else {
			c.legend = len(c.series) > 1
		}
	}
	if cx <= 0 {
		cx = 480
	}
	if cy <= 0 {
		cy = 240
	}
	e := widget.NewElement(x, y, cx, cy, hParent)
	if e == nil {
		return nil
	}
	c.Element = e
	e.EnableBkTransparent(true)
	e.AddEvent_Paint(c.onPaint)
	e.AddEvent_MouseMove(c.onMouseMove)
	e.AddEvent_MouseLeave(c.onMouseLeave)
	e.AddEvent_LButtonUp(c.onLButtonUp)
	e.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		delete(chartInstances, hEle)
		return 0
	})
	chartInstances[e.Handle] = c
	xui.AttachWindow(e.Handle)
	c.watchHost()
	return c
}

// chartInstances 是"元素句柄 → 图表"的登记表, 供窗口级鼠标移动回调反查。
// 为什么需要窗口级回调: ScrollArea(LayoutFrame) 里的元素收不到元素级 MouseMove
// (见 DESIGN 第八节 156 第 ③ 条), 页面里的图表否则永远不弹提示卡。
var chartInstances = map[int]*Chart{}

// chartWatchers 记录已经挂过窗口级回调的宿主窗口(每个窗口只挂一次)。
var chartWatchers = map[int]bool{}

// watchHost 给宿主窗口挂一次窗口级 MouseMove。
func (c *Chart) watchHost() {
	hWnd := xui.HostWindowOf(c.Handle)
	if hWnd == 0 {
		return
	}
	c.host = hWnd
	if c.watch[hWnd] || chartWatchers[hWnd] {
		c.watch[hWnd] = true
		return
	}
	w := window.NewByHandle(hWnd)
	if w == nil {
		return
	}
	c.watch[hWnd] = true
	chartWatchers[hWnd] = true
	w.AddEvent_MouseMove(func(hWin int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		chartWindowMouseMove(hWnd, pPt)
		return 0
	})
}

// chartWindowMouseMove 把窗口客户区坐标分发给同一窗口里的所有图表。
// 命中口径: `XEle_GetWndClientRect` 给的是窗口客户区里的逻辑矩形(与窗口鼠标事件同一
// 坐标系), 减掉矩形原点就是元素局部坐标。
func chartWindowMouseMove(hWnd int, pt *xc.POINT) {
	if pt == nil {
		return
	}
	for _, c := range chartInstances {
		if c.host != hWnd || !xc.XC_IsHXCGUI(c.Handle, xcc.XC_ELE) {
			continue
		}
		var rc xc.RECT
		xc.XEle_GetWndClientRect(c.Handle, &rc)
		inside := pt.X >= rc.Left && pt.X < rc.Right && pt.Y >= rc.Top && pt.Y < rc.Bottom
		if !inside || !chartVisible(c.Handle) {
			c.setHover(-1)
			continue
		}
		c.hoverX, c.hoverY = pt.X-rc.Left, pt.Y-rc.Top
		c.setHover(c.hitTest(c.hoverX, c.hoverY))
	}
}

// chartVisible 沿父元素链判断元素是否整体可见。隐藏页面里的元素矩形仍停在原位,
// 不判祖先可见性会命中到别的页(与 tooltip 的同名判定同因, 见 DESIGN 第八节 156)。
func chartVisible(hEle int) bool {
	for h := hEle; h != 0; h = xc.XWidget_GetParentEle(h) {
		if !xc.XWidget_IsShow(h) {
			return false
		}
	}
	return true
}

// ===== 公开 API =====

// Kind 返回图表形态。
func (c *Chart) Kind() ChartKind { return c.kind }

// SetKind 切换图表形态并重绘(隐藏状态语义不同, 一并清空)。
func (c *Chart) SetKind(k ChartKind) *Chart {
	if c.kind == k {
		return c
	}
	c.kind = k
	c.hidden = nil
	c.hover = -1
	c.Redraw(false)
	return c
}

// SetData 替换分类与数据。Values 短了补 0、长了截断; 分类数量变化会清空隐藏状态。
func (c *Chart) SetData(labels []string, series ...ChartSeries) *Chart {
	c.setData(labels, series)
	c.Redraw(false)
	return c
}

// Data 返回当前分类与数据的副本。
func (c *Chart) Data() (labels []string, series []ChartSeries) {
	labels = append([]string(nil), c.labels...)
	series = make([]ChartSeries, len(c.series))
	for i, s := range c.series {
		s.Values = append([]float64(nil), s.Values...)
		series[i] = s
	}
	return
}

// SetStacked 切换堆叠(柱状/面积)。
func (c *Chart) SetStacked(v bool) *Chart {
	c.stacked = v
	c.Redraw(false)
	return c
}

// SetCurve 切换平滑曲线(折线/面积)。
func (c *Chart) SetCurve(v bool) *Chart {
	c.curved = v
	c.Redraw(false)
	return c
}

// SetDots 切换数据点圆点(折线/面积)。
func (c *Chart) SetDots(v bool) *Chart {
	c.dots = v
	c.Redraw(false)
	return c
}

// SetGrid 切换横向网格线。
func (c *Chart) SetGrid(v bool) *Chart {
	c.showGrid = v
	c.Redraw(false)
	return c
}

// SetTooltip 切换悬停提示卡。
func (c *Chart) SetTooltip(v bool) *Chart {
	c.showTip = v
	c.Redraw(false)
	return c
}

// SetLegend 切换图例(同时视为显式设置, 不再按数据路数自动判断)。
func (c *Chart) SetLegend(v bool) *Chart {
	c.legend, c.legendSet = v, true
	c.Redraw(false)
	return c
}

// SetBarLabels 切换柱顶数值。
func (c *Chart) SetBarLabels(v bool) *Chart {
	c.barLabels = v
	c.Redraw(false)
	return c
}

// SetYFormat 自定义数值文本; 传 nil 恢复默认。
func (c *Chart) SetYFormat(fn func(float64) string) *Chart {
	c.yFormat = fn
	c.Redraw(false)
	return c
}

// SetYDomain 显式指定数值轴范围; lo >= hi 时恢复自动。
func (c *Chart) SetYDomain(lo, hi float64) *Chart {
	if lo < hi {
		c.yMin, c.yMax = lo, hi
	} else {
		c.yMin, c.yMax = 0, 0
	}
	c.Redraw(false)
	return c
}

// SetCenter 设置环形图中心文字。
func (c *Chart) SetCenter(title, sub string) *Chart {
	c.centerTitle, c.centerSub = title, sub
	c.Redraw(false)
	return c
}

// SetInnerRatio 设置环形图镂空比例(0~0.9)。
func (c *Chart) SetInnerRatio(r float64) *Chart {
	c.innerRatio = chartClampRatio(r)
	c.Redraw(false)
	return c
}

// OnPointClick 注册点击回调。直角坐标系传分类与数据路下标(series = -1 表示整组),
// 饼图传扇区(分类)下标。
func (c *Chart) OnPointClick(fn func(cat, series int, value float64)) *Chart {
	c.onClick = fn
	return c
}

// ===== 数据与状态 =====

func (c *Chart) isPie() bool { return c.kind == ChartPie || c.kind == ChartDonut }

func chartClampRatio(r float64) float64 {
	if r < 0 {
		return 0
	}
	if r > 0.9 {
		return 0.9
	}
	return r
}

func (c *Chart) setData(labels []string, series []ChartSeries) {
	if len(labels) != len(c.labels) {
		c.hidden = nil
	}
	c.labels = append([]string(nil), labels...)
	n := len(c.labels)
	c.series = make([]ChartSeries, len(series))
	for i, s := range series {
		v := make([]float64, n)
		copy(v, s.Values)
		s.Values = v
		if s.Label == "" {
			s.Label = "系列 " + strconv.Itoa(i+1)
		}
		c.series[i] = s
	}
	c.hover = -1
}

func (c *Chart) value(cat, s int) float64 {
	if s < 0 || s >= len(c.series) {
		return 0
	}
	v := c.series[s].Values
	if cat < 0 || cat >= len(v) {
		return 0
	}
	return v[cat]
}

func (c *Chart) isHidden(key int) bool {
	return key >= 0 && key < len(c.hidden) && c.hidden[key]
}

// visibleIdx 返回未被图例隐藏的数据路下标(饼图不使用)。
func (c *Chart) visibleIdx() []int {
	out := make([]int, 0, len(c.series))
	for i := range c.series {
		if c.isHidden(i) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// visibleCat 返回未被图例隐藏的分类下标(饼图用)。
func (c *Chart) visibleCat() []int {
	out := make([]int, 0, len(c.labels))
	for i := range c.labels {
		if c.isHidden(i) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// seriesColor 取一路数据的颜色: 显式色优先, 否则按主题色板轮转。
func (c *Chart) seriesColor(t *xui.Theme, idx int) xui.Color {
	if idx >= 0 && idx < len(c.series) && c.series[idx].Color != 0 {
		return c.series[idx].Color
	}
	return chartPalette(t)[((idx%5)+5)%5]
}

// catColor 取饼图里某个分类的颜色(饼图的"数据路"就是分类)。
func (c *Chart) catColor(t *xui.Theme, cat int) xui.Color {
	return chartPalette(t)[((cat%5)+5)%5]
}

func chartPalette(t *xui.Theme) []xui.Color {
	return []xui.Color{t.Chart1, t.Chart2, t.Chart3, t.Chart4, t.Chart5}
}

// chartSum 返回某个分类参与数值轴计算的值(堆叠时是一条柱的总高)。
func (c *Chart) chartSum(cat int, vis []int) float64 {
	if c.stacked {
		var sum float64
		for _, s := range vis {
			sum += c.value(cat, s)
		}
		return sum
	}
	var peak float64
	for _, s := range vis {
		if v := c.value(cat, s); math.Abs(v) > math.Abs(peak) {
			peak = v
		}
	}
	return peak
}

// domain 返回数值轴范围与刻度。
func (c *Chart) domain() (lo, hi float64, ticks []float64) {
	if c.yMin < c.yMax {
		lo, hi = c.yMin, c.yMax
		ticks = chartLinearTicks(lo, hi, 5)
		return lo, hi, ticks
	}
	vis := c.visibleIdx()
	lo, hi = math.Inf(1), math.Inf(-1)
	for i := range c.labels {
		v := c.chartSum(i, vis)
		lo = math.Min(lo, v)
		hi = math.Max(hi, v)
	}
	if math.IsInf(lo, 1) {
		lo, hi = 0, 1
	}
	if lo > 0 {
		lo = 0 // 全正数据的基线必须是 0(recharts domain=[0,'auto'])
	}
	if hi < 0 {
		hi = 0
	}
	if hi-lo < 1e-9 {
		hi = lo + 1
	}
	ticks = chartNiceTicks(lo, hi, 5)
	return ticks[0], ticks[len(ticks)-1], ticks
}

// pieSlices 返回饼图各扇区的分类下标、起始角与扫过角(角度制, 12 点方向为 0、顺时针为正)。
// 负数按 0 处理(扇区无法表达负值); 总量为 0 时返回空。
func (c *Chart) pieSlices() (cats []int, start []float64, sweep []float64) {
	cats = c.visibleCat()
	total := 0.0
	for _, i := range cats {
		if v := c.value(i, 0); v > 0 {
			total += v
		}
	}
	if total <= 0 {
		return nil, nil, nil
	}
	var acc float64
	for _, i := range cats {
		v := c.value(i, 0)
		if v < 0 {
			v = 0
		}
		sw := v / total * 360
		start = append(start, acc)
		sweep = append(sweep, sw)
		acc += sw
	}
	return cats, start, sweep
}

// ===== 几何 =====

type chartXTick struct {
	x    int32
	w    int32
	text string
}

type chartYTick struct {
	y    int32
	text string
}

type chartLegendSlot struct {
	rc  xc.RECT
	key int // 直角坐标系 = 数据路下标; 饼图 = 分类下标
}

// chartLayout 是绘制与命中共享的几何(逻辑像素, 元素客户区坐标 + 原点偏移)。
type chartLayout struct {
	ok     bool
	plot   xc.RECT
	bandW  int32
	xTicks []chartXTick
	yTicks []chartYTick
	legend []chartLegendSlot

	pieC  xc.POINT
	pieR  int32
	pieR2 int32

	lo, hi float64
	yTickX int32 // 数值轴文字左缘
}

// bandCenter 返回第 i 个分类的中心 x。
func (l chartLayout) bandCenter(i int) int32 {
	return l.plot.Left + l.bandW*int32(i) + l.bandW/2
}

// valueY 把数值映射到 y 坐标。
func (l chartLayout) valueY(v float64) float32 {
	if l.hi-l.lo <= 1e-9 {
		return float32(l.plot.Bottom)
	}
	f := (v - l.lo) / (l.hi - l.lo)
	return float32(l.plot.Bottom) - float32(f)*float32(l.plot.Bottom-l.plot.Top)
}

// layout 计算整套几何。ox/oy 是元素客户区原点(绘制时取 p.Rc.Left/Top, 命中测试时取 0)。
func (c *Chart) layout(ox, oy, w, h int32, t *xui.Theme) chartLayout {
	var lay chartLayout
	if w <= 0 || h <= 0 || len(c.labels) == 0 {
		return lay
	}
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: chartLabelPx})

	// 图例: 底部居中一行, 宽度不够时裁掉末尾项(宁可少显示也不换行)。
	legendH := int32(0)
	if c.legend {
		slots, _ := c.layoutLegend(ox+chartPad, oy+h-chartPad-chartLegendH, ox+w-chartPad, f)
		if len(slots) > 0 {
			lay.legend = slots
			legendH = chartLegendH + chartLegendGap
		}
	}

	if c.isPie() {
		box := xc.RECT{
			Left:   ox + chartPad,
			Top:    oy + chartPad,
			Right:  ox + w - chartPad,
			Bottom: oy + h - chartPad - legendH,
		}
		bw, bh := box.Right-box.Left, box.Bottom-box.Top
		r := bw
		if bh < r {
			r = bh
		}
		r /= 2
		if r < 8 {
			return lay
		}
		lay.pieC = xc.POINT{X: (box.Left + box.Right) / 2, Y: (box.Top + box.Bottom) / 2}
		lay.pieR = r
		lay.pieR2 = 0
		if c.kind == ChartDonut {
			lay.pieR2 = int32(float64(r) * c.innerRatio)
		}
		lay.ok = true
		return lay
	}

	lo, hi, ticks := c.domain()
	lay.lo, lay.hi = lo, hi

	maxLabelW := int32(0)
	texts := make([]string, len(ticks))
	for i, v := range ticks {
		texts[i] = c.tickText(v, ticks)
		if tw := xui.TextWidth(texts[i], f); tw > maxLabelW {
			maxLabelW = tw
		}
	}
	lay.yTickX = ox + chartPad

	bottom := oy + h - chartPad - legendH
	if len(c.labels) > 1 || c.hasXLabels() {
		bottom -= chartXLabelH
	}
	plot := xc.RECT{
		Left:   ox + chartPad + maxLabelW + chartAxisGap,
		Top:    oy + chartPad,
		Right:  ox + w - chartPad,
		Bottom: bottom,
	}
	if plot.Right-plot.Left < 24 || plot.Bottom-plot.Top < 24 {
		return lay
	}
	lay.plot = plot
	lay.bandW = (plot.Right - plot.Left) / int32(len(c.labels))
	if lay.bandW < 1 {
		return lay
	}
	for i, v := range ticks {
		y := int32(math.Round(float64(lay.valueY(v))))
		lay.yTicks = append(lay.yTicks, chartYTick{y: y, text: texts[i]})
	}

	// 分类文字: 放不下的跳过(贪心从左往右), 宁可少显示也不重叠成黑块。
	if c.hasXLabels() {
		lastRight := int32(-1 << 30)
		for i, s := range c.labels {
			tw := xui.TextWidth(s, f)
			if tw <= 0 {
				continue
			}
			if tw > lay.bandW-4 && len([]rune(s)) > 1 {
				s = chartEllipsis(s, lay.bandW-4, f)
				tw = xui.TextWidth(s, f)
			}
			cx := lay.bandCenter(i)
			left, right := cx-tw/2, cx+tw/2
			if left < lastRight+4 || right > ox+w-chartPad+4 {
				continue
			}
			lay.xTicks = append(lay.xTicks, chartXTick{x: cx, w: tw, text: s})
			lastRight = right
		}
	}
	lay.ok = true
	return lay
}

// hasXLabels 报告是否画分类文字(饼图的分类在图例里)。
func (c *Chart) hasXLabels() bool {
	for _, s := range c.labels {
		if s != "" {
			return true
		}
	}
	return false
}

// layoutLegend 排一行图例, 返回每项命中矩形与实际占用宽度。
func (c *Chart) layoutLegend(left, top, right int32, f int) ([]chartLegendSlot, int32) {
	type item struct {
		key  int
		text string
	}
	var items []item
	if c.isPie() {
		for i, s := range c.labels {
			items = append(items, item{key: i, text: s})
		}
	} else {
		for i, s := range c.series {
			items = append(items, item{key: i, text: s.Label})
		}
	}
	if len(items) == 0 {
		return nil, 0
	}

	const dotW, dotGap, itemGap = int32(8), int32(6), int32(16)
	widths := make([]int32, len(items))
	total := int32(0)
	for i, it := range items {
		widths[i] = dotW + dotGap + xui.TextWidth(it.text, f)
		total += widths[i]
		if i > 0 {
			total += itemGap
		}
	}
	x := left
	if avail := right - left; total < avail {
		x = left + (avail-total)/2
	}
	var slots []chartLegendSlot
	for i, it := range items {
		if x+widths[i] > right {
			break
		}
		slots = append(slots, chartLegendSlot{
			rc:  xc.RECT{Left: x, Top: top, Right: x + widths[i], Bottom: top + chartLegendH},
			key: it.key,
		})
		x += widths[i] + itemGap
	}
	return slots, total
}

// ===== 绘制 =====

func (c *Chart) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	lay := c.layout(p.Rc.Left, p.Rc.Top, p.W(), p.H(), t)
	if !lay.ok {
		return 0
	}
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: chartLabelPx})
	if f != 0 {
		p.Draw.SetFont(f)
	}

	if c.isPie() {
		c.paintPie(p, t, lay)
	} else {
		c.paintAxes(p, t, lay)
		c.paintSeries(p, t, lay)
	}
	c.paintLegend(p, t, lay, f)
	c.paintTooltip(p, t, lay, f)
	return 0
}

// paintAxes 画网格线、数值轴、分类文字与悬停竖向指示线。
func (c *Chart) paintAxes(p *xui.Painter, t *xui.Theme, lay chartLayout) {
	if c.showGrid {
		for _, tk := range lay.yTicks {
			p.Line(lay.plot.Left, tk.y, lay.plot.Right, tk.y, t.Border, 1)
		}
	}
	for _, tk := range lay.yTicks {
		rc := xc.RECT{
			Left:   lay.yTickX,
			Top:    tk.y - 9,
			Right:  lay.plot.Left - chartAxisGap,
			Bottom: tk.y + 9,
		}
		p.Text(tk.text, &rc, t.MutedForeground, xcc.TextAlignFlag_Right|xcc.TextAlignFlag_Vcenter)
	}
	for _, tk := range lay.xTicks {
		rc := xc.RECT{
			Left:   tk.x - tk.w/2,
			Top:    lay.plot.Bottom + 4,
			Right:  tk.x + tk.w/2,
			Bottom: lay.plot.Bottom + 4 + chartXLabelH,
		}
		p.Text(tk.text, &rc, t.MutedForeground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	}
	if c.hover >= 0 && c.hover < len(c.labels) {
		x := lay.bandCenter(c.hover)
		p.Line(x, lay.plot.Top, x, lay.plot.Bottom, t.MutedForeground.ScaleAlpha(0.35), 1)
	}
}

func (c *Chart) paintSeries(p *xui.Painter, t *xui.Theme, lay chartLayout) {
	vis := c.visibleIdx()
	if len(vis) == 0 {
		return
	}
	switch c.kind {
	case ChartBar:
		c.paintBars(p, t, lay, vis)
	case ChartLine, ChartArea:
		c.paintLines(p, t, lay, vis)
	}
}

func (c *Chart) paintBars(p *xui.Painter, t *xui.Theme, lay chartLayout, vis []int) {
	n := int32(len(c.labels))
	groupW := int32(float64(lay.bandW) * 0.72)
	if groupW < 2 {
		groupW = 2
	}
	segN := int32(len(vis))
	if c.stacked {
		segN = 1
	}
	if segN < 1 {
		segN = 1
	}
	const innerGap int32 = 2
	segW := groupW
	if segN > 1 {
		segW = (groupW - innerGap*(segN-1)) / segN
		if segW < 1 {
			segW = 1
		}
	}
	base := lay.valueY(0)
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: chartLabelPx})

	for cat := int32(0); cat < n; cat++ {
		bandLeft := lay.plot.Left + lay.bandW*cat
		x0 := bandLeft + (lay.bandW-groupW)/2
		if c.stacked {
			// 只有"最上面那段"圆角(其余段的顶端要接下一段, 圆角会露出缝)。
			last := -1
			for _, s := range vis {
				if c.value(int(cat), s) != 0 {
					last = s
				}
			}
			acc := 0.0
			for _, s := range vis {
				v := c.value(int(cat), s)
				if v == 0 {
					continue
				}
				from, to := lay.valueY(acc), lay.valueY(acc+v)
				acc += v
				top, bottom := from, to
				if top > bottom {
					top, bottom = bottom, top
				}
				rc := xc.RECT{
					Left:   x0,
					Top:    int32(math.Round(float64(top))),
					Right:  x0 + segW,
					Bottom: int32(math.Round(float64(bottom))),
				}
				c.fillBar(p, t, s, rc, v >= 0, s == last)
			}
			continue
		}
		for i, s := range vis {
			v := c.value(int(cat), s)
			x := x0 + int32(i)*(segW+innerGap)
			y := lay.valueY(v)
			top, bottom := y, base
			if top > bottom {
				top, bottom = bottom, top
			}
			rc := xc.RECT{
				Left:   x,
				Top:    int32(math.Round(float64(top))),
				Right:  x + segW,
				Bottom: int32(math.Round(float64(bottom))),
			}
			if rc.Bottom-rc.Top < 1 {
				rc.Bottom = rc.Top + 1 // 值为 0 也留一条 1px 基线, 与 shadcn 的空柱观感一致
			}
			c.fillBar(p, t, s, rc, v >= 0, true)
			if c.barLabels && f != 0 {
				txt := c.format(v)
				tw := xui.TextWidth(txt, f)
				lrc := xc.RECT{
					Left:   x + (segW-tw)/2,
					Top:    rc.Top - 18,
					Right:  x + (segW+tw)/2,
					Bottom: rc.Top - 2,
				}
				p.Text(txt, &lrc, t.Foreground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
			}
		}
	}
}

// fillBar 画一根柱子: 只圆"数据端"的两个角, 基线端保持方角。
func (c *Chart) fillBar(p *xui.Painter, t *xui.Theme, series int, rc xc.RECT, up bool, rounded bool) {
	col := c.seriesColor(t, series)
	r := int32(0)
	if rounded {
		r = chartBarRadius
		if half := (rc.Right - rc.Left) / 2; r > half {
			r = half
		}
		if h := rc.Bottom - rc.Top; r > h {
			r = h
		}
	}
	corners := xui.Corners{TopLeft: r, TopRight: r}
	if !up {
		corners = xui.Corners{BottomLeft: r, BottomRight: r}
	}
	p.FillRoundCorners(&rc, col, corners)
}

func (c *Chart) paintLines(p *xui.Painter, t *xui.Theme, lay chartLayout, vis []int) {
	n := len(c.labels)
	base := lay.valueY(0)

	// 堆叠面积: 逐层累加, 每层填充"本层曲线与下一层曲线"之间的区域。
	acc := make([]float64, n)
	for _, s := range vis {
		pts := make([]xc.POINTF, n)
		var lower []xc.POINTF
		if c.kind == ChartArea && c.stacked {
			lower = make([]xc.POINTF, n)
		}
		for i := 0; i < n; i++ {
			v := c.value(i, s)
			if c.kind == ChartArea && c.stacked {
				pts[i] = xc.POINTF{X: float32(lay.bandCenter(i)), Y: lay.valueY(acc[i] + v)}
				lower[i] = xc.POINTF{X: float32(lay.bandCenter(i)), Y: lay.valueY(acc[i])}
			} else {
				pts[i] = xc.POINTF{X: float32(lay.bandCenter(i)), Y: lay.valueY(v)}
			}
		}
		line := pts
		if c.curved {
			line = chartSmooth(pts, 12)
		}
		if c.kind == ChartArea {
			poly := make([]xc.POINTF, 0, len(line)*2+2)
			poly = append(poly, line...)
			if c.stacked {
				under := lower
				if c.curved {
					under = chartSmooth(lower, 12)
				}
				for i := len(under) - 1; i >= 0; i-- {
					poly = append(poly, under[i])
				}
			} else {
				poly = append(poly,
					xc.POINTF{X: line[len(line)-1].X, Y: base},
					xc.POINTF{X: line[0].X, Y: base})
			}
			// 面积带仍是半透明色(0.22), 不跟着改 D2D 直通: D2D 渲染目标的 alphaMode
			// 由 xcgui 建 target 时决定, 未公开 —— 若为 PREMULTIPLIED, 非预乘色会明显偏亮。
			// 扇区是不透明色, 预乘与否没有差异, 所以只有饼/环走直通。
			p.Draw.SetBrushColor(uint32(c.seriesColor(t, s).ScaleAlpha(0.22)))
			p.Draw.FillPolygonF(poly)
		}

		p.Draw.SetLineWidth(chartLineW)
		p.Draw.SetBrushColor(uint32(c.seriesColor(t, s)))
		for i := 1; i < len(line); i++ {
			p.Draw.DrawLineF(line[i-1].X, line[i-1].Y, line[i].X, line[i].Y)
		}
		if c.dots {
			for i := 0; i < n; i++ {
				cx, cy := float32(lay.bandCenter(i)), pts[i].Y
				rr := float32(chartDotR)
				rc := xc.RECTF{Left: cx - rr, Top: cy - rr, Right: cx + rr, Bottom: cy + rr}
				// 点填充底色(与 shadcn 的 fill=background 一致), 再描一圈数据色。
				p.Draw.SetBrushColor(uint32(t.Card))
				p.Draw.FillEllipseF(&rc)
				p.Draw.SetLineWidth(chartLineW)
				p.Draw.SetBrushColor(uint32(c.seriesColor(t, s)))
				ir := xc.RECT{
					Left:   int32(math.Round(float64(rc.Left))),
					Top:    int32(math.Round(float64(rc.Top))),
					Right:  int32(math.Round(float64(rc.Right))),
					Bottom: int32(math.Round(float64(rc.Bottom))),
				}
				p.Draw.DrawEllipse(&ir)
			}
		}
		if c.kind == ChartArea && c.stacked {
			for i := 0; i < n; i++ {
				acc[i] += c.value(i, s)
			}
		}
	}
}

// fillShape 用 D2D 路径几何填充一块闭合多边形(当前只给扇区用), 失败时返回 false 由调用方回落 FillPolygonF。
// 只用于不透明色: 直通绘制绕过框架的 alpha 合成, 半透明色还要处理 D2D 的预乘 alpha。
//
// 为什么不用 XDraw_FillPolygonF: 它在 D2D 渲染下是无抗锯齿的硬填充, 饼图/环形图的
// 外弧与内弧会出现台阶(半径越大越明显; 折线采样再密也治不了, 因为问题在光栅化不在几何)。
// 这里走与图标相同的 D2D 直通(见 d2dicon.go): 建闭环路径 → FillGeometry, 边缘由
// Direct2D 做抗锯齿。回落只在 GDI 渲染(app.New(false))或 D2D 资源创建失败时发生。
func (c *Chart) fillShape(p *xui.Painter, col xui.Color, pts []xc.POINTF) bool {
	if p == nil || p.Draw == nil || p.Draw.Handle <= 0 || len(pts) < 3 {
		return false
	}
	rt := d2d.NewRenderTargetByHandle(xc.XDraw_GetD2dRenderTarget(p.Draw.Handle))
	factory := d2d.NewFactoryByHandle(xc.XC_GetD2dFactory())
	if rt == nil || factory == nil {
		return false
	}
	// 直通 D2D 拿到的渲染目标上**没有**当前元素的变换(炫彩是刷新自己的绘制命令时才施加),
	// 必须用 XDraw_ConvRect 把客户区矩形换算成画布坐标(含元素偏移 + DPI 缩放), 再按这个
	// 比值设置变换。否则高 DPI 下扇区会按"逻辑像素当物理像素"画出来 —— 图形整体缩小
	// 1/1.5 并偏向左上(实测 150% DPI 下圆心落在逻辑坐标处)。
	w0 := float64(p.Rc.Right - p.Rc.Left)
	h0 := float64(p.Rc.Bottom - p.Rc.Top)
	if w0 <= 0 || h0 <= 0 {
		return false
	}
	canvas := p.Rc
	xc.XDraw_ConvRect(p.Draw.Handle, &canvas)
	sx := float64(canvas.Right-canvas.Left) / w0
	sy := float64(canvas.Bottom-canvas.Top) / h0
	if sx <= 0 || sy <= 0 { // 换算失败(canvas 与客户区不像同一块区域)就别硬画
		return false
	}
	prev, err := rt.GetTransform()
	if err != nil {
		return false
	}
	tm := d2d.D2D1_MATRIX_3X2_F{
		M11: float32(sx),
		M22: float32(sy),
		DX:  float32(float64(canvas.Left) - float64(p.Rc.Left)*sx),
		DY:  float32(float64(canvas.Top) - float64(p.Rc.Top)*sy),
	}
	rt.SetTransform(&tm)
	defer rt.SetTransform(&prev) // 必须成对恢复: 残留矩阵会把后续元素一起平移/缩放

	flat := make([]float32, 0, len(pts)*2)
	for _, q := range pts {
		flat = append(flat, q.X, q.Y)
	}
	g, err := factory.NewGeometryFigures([]d2d.GeometryFigure{{Points: flat, Closed: true}})
	if err != nil || g == nil {
		return false
	}
	defer g.Release()

	r, gg, b, a := col.RGBA()
	// 直通绘制绕过了框架的元素 alpha 合成(同图标): 把元素当前 alpha 乘进颜色,
	// 否则该元素做淡入淡出时扇区还是全不透明。
	if ea := xc.XEle_GetAlpha(c.Handle); ea != 255 {
		a = byte(uint32(a) * uint32(ea) / 255)
	}
	return rt.FillGeometryColor(g.Base(), d2d.NewColorF(r, gg, b, a)) == nil
}

func (c *Chart) paintPie(p *xui.Painter, t *xui.Theme, lay chartLayout) {
	cats, start, sweep := c.pieSlices()
	if len(cats) == 0 {
		return
	}
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: chartLabelPx})
	for i, cat := range cats {
		r := lay.pieR
		// 悬停扇区外扩 4px: 比改 alpha 更容易被看见, 也不需要额外的过渡态。
		if c.hover >= 0 && cat == c.hover {
			r += 4
		}
		pts := chartArcPoints(lay.pieC.X, lay.pieC.Y, r, lay.pieR2, start[i], start[i]+sweep[i])
		if len(pts) < 3 {
			continue
		}
		if !c.fillShape(p, c.catColor(t, cat), pts) {
			p.Draw.SetBrushColor(uint32(c.catColor(t, cat)))
			p.Draw.FillPolygonF(pts)
		}

		if sweep[i] >= 18 && f != 0 {
			mid := start[i] + sweep[i]/2
			rr := float64(lay.pieR+lay.pieR2) / 2
			lx := float64(lay.pieC.X) + rr*math.Sin(mid*math.Pi/180)
			ly := float64(lay.pieC.Y) - rr*math.Cos(mid*math.Pi/180)
			txt := strconv.Itoa(int(math.Round(sweep[i]/3.6))) + "%"
			tw := xui.TextWidth(txt, f)
			// 左右各留 4px: DrawText 在矩形里会自动换行, 而 TextWidth 量出来的宽度
			// 与绘制时的排版宽度并不严格相等 —— 卡着 tw 给框会把 "48%" 折成 "48" + "%" 两行。
			rc := xc.RECT{
				Left:   int32(lx) - tw/2 - 4,
				Top:    int32(ly) - 10,
				Right:  int32(lx) + tw/2 + 4,
				Bottom: int32(ly) + 10,
			}
			// 上游饼图标签写死 fill="var(--background)" —— 那是给彩色片定的;
			// 本库的 chart 色板是灰阶(neutral 预设), 浅灰片上再写浅色正文就看不见了,
			// 所以按片色亮度在正文色/背景色之间取对比更高的那个。
			p.Text(txt, &rc, chartContrastText(t, c.catColor(t, cat)), xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		}
	}
	if lay.pieR2 <= 0 || (c.centerTitle == "" && c.centerSub == "") {
		return
	}
	cx, cy := lay.pieC.X, lay.pieC.Y
	half := lay.pieR2 - 8
	if c.centerTitle != "" {
		fb := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 20, Bold: true})
		if fb != 0 {
			p.Draw.SetFont(fb)
		}
		bottom := cy + 4
		if c.centerSub == "" {
			bottom = cy + 10
		}
		rc := xc.RECT{Left: cx - half, Top: bottom - 24, Right: cx + half, Bottom: bottom}
		p.Text(c.centerTitle, &rc, t.Foreground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		if f != 0 {
			p.Draw.SetFont(f)
		}
	}
	if c.centerSub != "" {
		top := cy + 4
		if c.centerTitle == "" {
			top = cy - 10
		}
		rc := xc.RECT{Left: cx - half, Top: top, Right: cx + half, Bottom: top + 18}
		p.Text(c.centerSub, &rc, t.MutedForeground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	}
}

func (c *Chart) paintLegend(p *xui.Painter, t *xui.Theme, lay chartLayout, f int) {
	for _, slot := range lay.legend {
		col, text := c.legendColorAndText(t, slot.key)
		hidden := c.isHidden(slot.key)
		dotCol, textCol := col, t.Foreground
		if hidden {
			dotCol = col.ScaleAlpha(0.3)
			textCol = t.MutedForeground.ScaleAlpha(0.7)
		}
		const dotW int32 = 8
		cy := (slot.rc.Top + slot.rc.Bottom) / 2
		dot := xc.RECTF{
			Left:   float32(slot.rc.Left),
			Top:    float32(cy - dotW/2),
			Right:  float32(slot.rc.Left + dotW),
			Bottom: float32(cy + dotW/2),
		}
		p.Draw.SetBrushColor(uint32(dotCol))
		p.Draw.FillEllipseF(&dot)
		trc := xc.RECT{
			Left:   slot.rc.Left + dotW + 6,
			Top:    slot.rc.Top,
			Right:  slot.rc.Right,
			Bottom: slot.rc.Bottom,
		}
		p.Text(text, &trc, textCol, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	}
}

// legendColorAndText 按图例键取颜色与文字(直角坐标系 = 数据路, 饼图 = 分类)。
func (c *Chart) legendColorAndText(t *xui.Theme, key int) (xui.Color, string) {
	if c.isPie() {
		text := ""
		if key >= 0 && key < len(c.labels) {
			text = c.labels[key]
		}
		return c.catColor(t, key), text
	}
	text := ""
	if key >= 0 && key < len(c.series) {
		text = c.series[key].Label
	}
	return c.seriesColor(t, key), text
}

func (c *Chart) paintTooltip(p *xui.Painter, t *xui.Theme, lay chartLayout, f int) {
	if !c.showTip || c.hover < 0 || c.hover >= len(c.labels) {
		return
	}
	type row struct {
		col   xui.Color
		label string
		value string
	}
	title := c.labels[c.hover]
	var rows []row
	if c.isPie() {
		val := c.value(c.hover, 0)
		total := 0.0
		for i := range c.labels {
			if v := c.value(i, 0); v > 0 && !c.isHidden(i) {
				total += v
			}
		}
		pct := 0.0
		if total > 0 {
			pct = val / total * 100
		}
		name := ""
		if len(c.series) > 0 {
			name = c.series[0].Label
		}
		rows = append(rows, row{
			col:   c.catColor(t, c.hover),
			label: name,
			value: c.format(val) + " (" + strconv.Itoa(int(math.Round(pct))) + "%)",
		})
	} else {
		for _, s := range c.visibleIdx() {
			rows = append(rows, row{
				col:   c.seriesColor(t, s),
				label: c.series[s].Label,
				value: c.format(c.value(c.hover, s)),
			})
		}
	}

	// 量宽: 标题与每行(圆点 + 名称 + 间距 + 值)取最大者。
	bw := xui.TextWidth(title, f)
	for _, r := range rows {
		w := 8 + 6 + xui.TextWidth(r.label, f) + 12 + xui.TextWidth(r.value, f)
		if w > bw {
			bw = w
		}
	}
	bw += chartTipPad * 2
	bh := chartTipPad*2 + 16 + 4 + int32(len(rows))*chartTipRowH

	// 默认放在光标右下, 放不下就翻到左上, 再夹进元素矩形。
	x := c.hoverX + 14
	if x+bw > int32(p.Rc.Left)+p.W()-chartPad {
		x = c.hoverX - 14 - bw
	}
	if x < p.Rc.Left+chartPad {
		x = p.Rc.Left + chartPad
	}
	y := c.hoverY + 12
	if y+bh > int32(p.Rc.Top)+p.H()-chartPad {
		y = c.hoverY - 12 - bh
	}
	if y < p.Rc.Top+chartPad {
		y = p.Rc.Top + chartPad
	}
	rc := xc.RECT{Left: x, Top: y, Right: x + bw, Bottom: y + bh}
	p.FillRound(&rc, t.Popover, t.RadiusMd())
	p.StrokeRound(&rc, t.Border, t.RadiusMd(), 1)

	trc := xc.RECT{
		Left:   rc.Left + chartTipPad,
		Top:    rc.Top + chartTipPad,
		Right:  rc.Right - chartTipPad,
		Bottom: rc.Top + chartTipPad + 16,
	}
	p.Text(title, &trc, t.Foreground, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)

	ry := rc.Top + chartTipPad + 16 + 4
	for _, r := range rows {
		cy := ry + chartTipRowH/2
		dot := xc.RECTF{
			Left:   float32(rc.Left + chartTipPad),
			Top:    float32(cy - 4),
			Right:  float32(rc.Left + chartTipPad + 8),
			Bottom: float32(cy + 4),
		}
		p.Draw.SetBrushColor(uint32(r.col))
		p.Draw.FillEllipseF(&dot)
		lrc := xc.RECT{Left: rc.Left + chartTipPad + 14, Top: ry, Right: rc.Right - chartTipPad, Bottom: ry + chartTipRowH}
		p.Text(r.label, &lrc, t.MutedForeground, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
		p.Text(r.value, &lrc, t.Foreground, xcc.TextAlignFlag_Right|xcc.TextAlignFlag_Vcenter)
		ry += chartTipRowH
	}
}

// ===== 命中 =====

// hitTest 把元素局部坐标换算成悬停下标(分类或扇区), -1 = 不在可交互区域。
func (c *Chart) hitTest(x, y int32) int {
	lay := c.localLayout()
	if !lay.ok {
		return -1
	}
	if c.isPie() {
		dx := float64(x - lay.pieC.X)
		dy := float64(y - lay.pieC.Y)
		d := math.Hypot(dx, dy)
		if d > float64(lay.pieR+4) || d < float64(lay.pieR2) {
			return -1
		}
		cats, start, sweep := c.pieSlices()
		if len(cats) == 0 {
			return -1
		}
		// atan2 以 3 点方向为 0、逆时针为正; 转成"12 点方向为 0、顺时针为正"。
		ang := math.Atan2(dx, -dy) * 180 / math.Pi
		if ang < 0 {
			ang += 360
		}
		for i := range cats {
			if ang >= start[i] && ang <= start[i]+sweep[i] {
				return cats[i]
			}
		}
		return -1
	}
	if x < lay.plot.Left || x >= lay.plot.Right || y < lay.plot.Top || y > lay.plot.Bottom+chartXLabelH {
		return -1
	}
	i := int((x - lay.plot.Left) / lay.bandW)
	if i < 0 {
		i = 0
	}
	if i >= len(c.labels) {
		i = len(c.labels) - 1
	}
	return i
}

// localLayout 按元素自身尺寸算一遍几何(命中测试用, 原点取 0)。
func (c *Chart) localLayout() chartLayout {
	var rc xc.RECT
	xc.XEle_GetClientRect(c.Handle, &rc)
	return c.layout(0, 0, rc.Right-rc.Left, rc.Bottom-rc.Top, xui.Current())
}

// setHover 更新悬停分类; 变化时立即重绘(图表没有过渡态, 悬停反馈是同步的)。
func (c *Chart) setHover(idx int) {
	if idx >= len(c.labels) {
		idx = -1
	}
	if c.hover == idx {
		return
	}
	c.hover = idx
	c.Redraw(false)
}

func (c *Chart) onMouseMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pPt == nil {
		return 0
	}
	c.watchHost()
	c.hoverX, c.hoverY = pPt.X, pPt.Y
	c.setHover(c.hitTest(pPt.X, pPt.Y))
	return 0
}

func (c *Chart) onMouseLeave(hEle int, hEleStay int, pbHandled *bool) int {
	c.setHover(-1)
	return 0
}

func (c *Chart) onLButtonUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pPt == nil {
		return 0
	}
	lay := c.localLayout()
	if !lay.ok {
		return 0
	}
	// 图例点击 = 临时隐藏/恢复一项(上游图例本身不可点, 这是桌面端的常用增益)。
	for _, slot := range lay.legend {
		if pPt.X >= slot.rc.Left && pPt.X < slot.rc.Right && pPt.Y >= slot.rc.Top && pPt.Y < slot.rc.Bottom {
			for len(c.hidden) < c.legendKeyCount() {
				c.hidden = append(c.hidden, false)
			}
			if slot.key >= 0 && slot.key < len(c.hidden) {
				c.hidden[slot.key] = !c.hidden[slot.key]
			}
			*pbHandled = true
			c.hover = -1
			c.Redraw(false)
			return 0
		}
	}
	if c.onClick == nil {
		return 0
	}
	if idx := c.hitTest(pPt.X, pPt.Y); idx >= 0 {
		if c.isPie() {
			c.onClick(idx, 0, c.value(idx, 0))
		} else {
			c.onClick(idx, -1, c.chartSum(idx, c.visibleIdx()))
		}
	}
	return 0
}

// legendKeyCount 返回图例项的键空间大小(直角坐标系 = 数据路数, 饼图 = 分类数)。
func (c *Chart) legendKeyCount() int {
	if c.isPie() {
		return len(c.labels)
	}
	return len(c.series)
}

// ===== 数值文本 =====

func (c *Chart) format(v float64) string {
	if c.yFormat != nil {
		return c.yFormat(v)
	}
	return chartFormatNumber(v)
}

// tickText 生成数值轴刻度文字: 有自定义格式器时优先, 否则按刻度步长决定小数位。
func (c *Chart) tickText(v float64, ticks []float64) string {
	if c.yFormat != nil {
		return c.yFormat(v)
	}
	step := 0.0
	if len(ticks) > 1 {
		step = math.Abs(ticks[1] - ticks[0])
	}
	return chartTickText(v, step)
}

// chartFormatNumber 是默认数值文本(千分位 + 最多两位小数)。
func chartFormatNumber(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var s string
	if math.Abs(v-math.Round(v)) < 1e-9 {
		s = strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	} else {
		s = strconv.FormatFloat(v, 'f', 2, 64)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	s = chartThousands(s)
	if neg {
		s = "-" + s
	}
	return s
}

// chartTickText 按步长补小数位(步长 0.5 时显示 0.5/1.0, 步长 100 时不带小数)。
func chartTickText(v, step float64) string {
	if step <= 0 {
		return chartFormatNumber(v)
	}
	return chartThousands(strconv.FormatFloat(v, 'f', chartStepDecimals(step), 64))
}

// chartStepDecimals 返回"刚好能表示该步长"的小数位数:
// 0.25 → 2(不能只留 1 位, 否则刻度被舍成 0.2/0.5/0.8), 2.5 → 1, 100 → 0。
func chartStepDecimals(step float64) int {
	if step <= 0 {
		return 0
	}
	for dec := 0; dec <= 8; dec++ {
		p := math.Pow(10, float64(dec))
		if math.Abs(step*p-math.Round(step*p)) < 1e-6 {
			return dec
		}
	}
	return 8
}

// chartThousands 给整数部分加千分位逗号("12000.5" → "12,000.5")。
func chartThousands(s string) string {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	if len(intPart) > 3 {
		var b strings.Builder
		pre := len(intPart) % 3
		if pre > 0 {
			b.WriteString(intPart[:pre])
		}
		for i := pre; i < len(intPart); i += 3 {
			if b.Len() > 0 {
				b.WriteByte(',')
			}
			b.WriteString(intPart[i : i+3])
		}
		intPart = b.String()
	}
	out := intPart + frac
	if neg {
		out = "-" + out
	}
	return out
}

// chartContrastText 在 Foreground / Background 之间挑一个与片色对比更高的文字色。
//
// 为什么不直接抄上游的 fill="var(--background)": 上游示例跑在彩色色板上, 背景色
// 在任意片色之上都能读; 而中性主题的 Chart1..5 是灰阶, 浅灰片 + 白字 = 看不见。
// 也不能简单按"片色亮 → 用 Foreground": 暗色主题的 Foreground 本身就接近白色,
// 浅灰片上照样糊成一片。只比两个候选与片色的亮度差, 与主题无关。
func chartContrastText(t *xui.Theme, bg xui.Color) xui.Color {
	lb := chartLum(bg)
	if math.Abs(chartLum(t.Foreground)-lb) >= math.Abs(chartLum(t.Background)-lb) {
		return t.Foreground
	}
	return t.Background
}

// chartLum 返回颜色的 0..1 感知亮度(加权近似, 只用于挑对比色)。
func chartLum(c xui.Color) float64 {
	r, g, b, _ := c.RGBA()
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
}

// chartEllipsis 把文字裁到指定宽度(按字符累加, 末尾补省略号)。
func chartEllipsis(s string, maxW int32, hFont int) string {
	if maxW <= 0 || hFont == 0 {
		return ""
	}
	if xui.TextWidth(s, hFont) <= maxW {
		return s
	}
	runes := []rune(s)
	for n := len(runes) - 1; n > 0; n-- {
		cand := string(runes[:n]) + "…"
		if xui.TextWidth(cand, hFont) <= maxW {
			return cand
		}
	}
	return "…"
}

// ===== 纯计算(几何 / 刻度 / 曲线), 与 GUI 无关, 由 chart_test.go 覆盖 =====

// chartLinearTicks 把 [lo,hi] 均分成 n 个刻度(显式值域用)。
func chartLinearTicks(lo, hi float64, n int) []float64 {
	if n < 2 {
		n = 2
	}
	out := make([]float64, n)
	step := (hi - lo) / float64(n-1)
	for i := range out {
		out[i] = lo + float64(i)*step
	}
	out[n-1] = hi
	return out
}

// chartNiceTicks 返回覆盖 [lo,hi] 的"整齐"刻度(1/2/2.5/5 × 10^n 的步长)。
// 返回值的首尾一般会略超出输入范围 —— 这正是数值轴要的留白。
func chartNiceTicks(lo, hi float64, count int) []float64 {
	if count < 2 {
		count = 2
	}
	if !(hi > lo) {
		hi = lo + 1
	}
	step := chartNiceNum((hi-lo)/float64(count-1))
	nlo := math.Floor(lo/step) * step
	nhi := math.Ceil(hi/step) * step
	n := int(math.Round((nhi - nlo) / step))
	if n < 1 || n > 60 {
		return chartLinearTicks(lo, hi, count)
	}
	out := make([]float64, 0, n+1)
	for i := 0; i <= n; i++ {
		out = append(out, chartRoundTo(nlo+float64(i)*step, step))
	}
	return out
}

// chartNiceNum 把步长吸附到 1/2/2.5/5/10 这一组(与 d3 的 nice() 同一套取值)。
func chartNiceNum(x float64) float64 {
	if x <= 0 {
		return 1
	}
	exp := math.Floor(math.Log10(x))
	pow := math.Pow(10, exp)
	f := x / pow
	var nf float64
	switch {
	case f <= 1:
		nf = 1
	case f <= 2:
		nf = 2
	case f <= 2.5:
		nf = 2.5
	case f <= 5:
		nf = 5
	default:
		nf = 10
	}
	return nf * pow
}

// chartRoundTo 把刻度值吸附到步长的小数位, 消掉 0.30000000000000004 这类浮点噪声。
func chartRoundTo(v, step float64) float64 {
	if step <= 0 {
		return v
	}
	p := math.Pow(10, float64(chartStepDecimals(step)))
	return math.Round(v*p) / p
}

// chartSmooth 用 Catmull-Rom 把折线采样成平滑曲线。
//
// 为什么不直接用 `Draw.DrawCurve`: drawx 的注释写明它在 D2D 下是空实现(见
// xcgui/drawx 的 DrawCurve), 画出来什么都看不到。这里自己按 Catmull-Rom 采样成
// 折线段, 平滑后的折线仍走 DrawLineF / FillPolygonF, 与直线路径共用同一套绘制。
func chartSmooth(pts []xc.POINTF, samples int) []xc.POINTF {
	if len(pts) < 3 || samples < 2 {
		return pts
	}
	out := make([]xc.POINTF, 0, (len(pts)-1)*samples+1)
	for i := 0; i < len(pts)-1; i++ {
		p0 := pts[i]
		if i > 0 {
			p0 = pts[i-1]
		}
		p1, p2 := pts[i], pts[i+1]
		p3 := p2
		if i+2 < len(pts) {
			p3 = pts[i+2]
		}
		for s := 0; s < samples; s++ {
			out = append(out, chartCatmullPoint(p0, p1, p2, p3, float64(s)/float64(samples)))
		}
	}
	return append(out, pts[len(pts)-1])
}

func chartCatmullPoint(p0, p1, p2, p3 xc.POINTF, t float64) xc.POINTF {
	t2, t3 := t*t, t*t*t
	f := func(a, b, c, d float64) float64 {
		return 0.5 * (2*b + (-a+c)*t + (2*a-5*b+4*c-d)*t2 + (-a+3*b-3*c+d)*t3)
	}
	return xc.POINTF{
		X: float32(f(float64(p0.X), float64(p1.X), float64(p2.X), float64(p3.X))),
		Y: float32(f(float64(p0.Y), float64(p1.Y), float64(p2.Y), float64(p3.Y))),
	}
}

// chartArcPoints 生成一块扇区(或扇环)的多边形顶点。
// 角度制, 12 点方向为 0、顺时针为正; r2 <= 0 时从圆心起(饼图), 否则是扇环(环形图)。
func chartArcPoints(cx, cy, r1, r2 int32, a0, a1 float64) []xc.POINTF {
	span := a1 - a0
	steps := int(math.Ceil(math.Abs(span) * 2)) // 每度 2 个采样点, 360° 圆周也足够平滑
	if steps < 2 {
		steps = 2
	}
	pt := func(r int32, a float64) xc.POINTF {
		rad := a * math.Pi / 180
		return xc.POINTF{
			X: float32(cx) + float32(float64(r)*math.Sin(rad)),
			Y: float32(cy) - float32(float64(r)*math.Cos(rad)),
		}
	}
	out := make([]xc.POINTF, 0, steps*2+2)
	if r2 <= 0 {
		out = append(out, xc.POINTF{X: float32(cx), Y: float32(cy)})
		for i := 0; i <= steps; i++ {
			out = append(out, pt(r1, a0+span*float64(i)/float64(steps)))
		}
		return out
	}
	for i := 0; i <= steps; i++ {
		out = append(out, pt(r1, a0+span*float64(i)/float64(steps)))
	}
	for i := steps; i >= 0; i-- {
		out = append(out, pt(r2, a0+span*float64(i)/float64(steps)))
	}
	return out
}
