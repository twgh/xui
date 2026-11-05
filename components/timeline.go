package ui

import (
	"math"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 时间轴几何常量(逻辑像素)。
//
// shadcn 无官方 timeline, 语义按社区通行做法(并参照 Ant Design Timeline)定死
// (见 DESIGN 第四节末):
//   - 轴线颜色取 border, 节点默认取 primary; 首条不画上游线段、末条不画下游线段;
//   - 竖向 start: 轴线在左侧, 内容在右; end: 轴线在右侧, 内容在左(右对齐);
//     alternate: 轴线居中, 内容左右交替(第一条在左, 与 AntD 一致);
//   - 水平 start: 轴线在顶部、内容在下; end: 轴线在底部、内容在上;
//     alternate: 轴线居中、内容上下交替(第一条在上); 各条目等宽。
const (
	timelineNodeCol   int32 = 32 // 竖向 start/end 的内容区让位宽度
	timelineNodeX     int32 = 12 // 轴线到所在边的内缩(也是清除条半宽)
	timelineAxisGap   int32 = 20 // 节点圆心到内容边缘的间距
	timelineLineW     int32 = 2  // 轴线宽度
	timelineDotHalo   int32 = 7  // 圆点外的底色光环半径(盖住穿过圆点的轴线)
	timelineDotR      int32 = 5  // 实心圆点半径
	timelineRingR     int32 = 6  // 空心节点外半径
	timelineRingHole  int32 = 4  // 空心节点内半径(差值 = 环宽)
	timelineGlyphR    int32 = 8  // 自定义图标/加载指示器半径(16px, 约 1.5 倍圆点)
	timelineTimeH     int32 = 16 // text-xs 行高
	timelineTitleH    int32 = 20 // text-sm 行高
	timelineDescH     int32 = 20 // text-sm 行高
	timelineContentG  int32 = 4  // 条目内各行之间的间距
	timelineItemGap   int32 = 24 // 竖向条目之间的默认间距
	timelineFallbackY int32 = 10 // 竖向没量到首个子项时圆点中心的兜底位置
)

// TimelineOrientation 时间轴取向。
type TimelineOrientation int

const (
	TimelineVertical   TimelineOrientation = iota // 竖向(默认): 条目上下排列
	TimelineHorizontal                            // 水平: 条目左右等宽排列, 轴线横贯
)

// TimelineMode 内容相对轴线出现的位置。
type TimelineMode int

const (
	TimelineStart     TimelineMode = iota // 竖向在轴线右侧 / 水平在轴线下侧
	TimelineEnd                           // 竖向在轴线左侧 / 水平在轴线上侧
	TimelineAlternate                     // 两侧交替(竖向左右、水平上下)
)

// TimelineVariant 节点形态。
type TimelineVariant int

const (
	TimelineFilled   TimelineVariant = iota // 实心圆点(默认)
	TimelineOutlined                        // 空心圆环(底色填充中心, 轴线不会穿过)
)

// TimelineColor 节点语义色。
type TimelineColor int

const (
	TimelineColorPrimary     TimelineColor = iota // 主色(默认)
	TimelineColorDestructive                      // 破坏/错误
	TimelineColorMuted                            // 弱化的灰点
)

// timelineSide 条目内容在轴线哪一侧(内部用)。
type timelineSide uint8

const (
	timelineRight timelineSide = iota
	timelineLeft
	timelineBelow
	timelineAbove
)

// Timeline 时间轴容器: 竖向或水平排列的条目, 由组件绘制贯穿的轴线与节点。
//
// 内容槽就是 TimelineItem 本身(它是一个布局容器): 内置的 时间/标题/说明 只是往里面
// 放三个 Label, 之后仍可继续加任意 Label / Item / Card —— 条目尺寸随内容自撑,
// 竖向轴线按条目实际高度一路接下去。
type Timeline struct {
	*widget.LayoutEle

	items       []*TimelineItem
	gap         int32
	backdrop    func(*xui.Theme) xui.Color
	orientation TimelineOrientation
	mode        TimelineMode
	variant     TimelineVariant

	queued      bool
	reflowing   bool
	forceAdjust bool // 条目增删后即使内边距没变也要把新高度推给外层
	lastW       int32
	lastH       int32
	itemW       int32 // 水平取向已写入的条目宽(0 = 还没锁过)
}

// TimelineOption 时间轴配置项。
type TimelineOption func(*Timeline)

// WithTimelineOrientation 设置取向(默认竖向)。
func WithTimelineOrientation(o TimelineOrientation) TimelineOption {
	return func(t *Timeline) {
		if o == TimelineVertical || o == TimelineHorizontal {
			t.orientation = o
		}
	}
}

// WithTimelineMode 设置内容位置: start(默认) / end / alternate。
// 竖向交替时第一条在左, 水平交替时第一条在上(同 Ant Design)。
func WithTimelineMode(m TimelineMode) TimelineOption {
	return func(t *Timeline) {
		if m == TimelineStart || m == TimelineEnd || m == TimelineAlternate {
			t.mode = m
		}
	}
}

// WithTimelineVariant 设置节点形态(默认实心圆点)。
func WithTimelineVariant(v TimelineVariant) TimelineOption {
	return func(t *Timeline) {
		if v == TimelineFilled || v == TimelineOutlined {
			t.variant = v
		}
	}
}

// WithTimelineGap 设置竖向条目之间的间距(默认 24)。水平取向不使用该间距(条目等宽相邻)。
func WithTimelineGap(px int32) TimelineOption {
	return func(t *Timeline) {
		if px >= 0 {
			t.gap = px
		}
	}
}

// WithTimelineBackdrop 指定轴线/节点底下铺的宿主底色(默认为 t.Card)。
// 轴线与节点每帧先用宿主底色擦掉上一帧; 放进非 Card 底色的容器时用它跟随宿主。
func WithTimelineBackdrop(fn func(t *xui.Theme) xui.Color) TimelineOption {
	return func(t *Timeline) { t.backdrop = fn }
}

// NewTimeline 创建时间轴容器。宽铺满父级、高随条目自撑(cx/cy 只作初始尺寸)。
// 水平取向时条目按容器宽度等分。
func NewTimeline(x, y, cx, cy int32, hParent int, opts ...TimelineOption) *Timeline {
	if cx <= 0 {
		cx = 320
	}
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	t := &Timeline{LayoutEle: ele, gap: timelineItemGap}
	for _, o := range opts {
		o(t)
	}
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(0)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	t.applyOrientation()
	ele.AddEvent_Size(t.onSize)
	xui.AttachWindow(ele.Handle)
	return t
}

// applyOrientation 把取向写到布局上。
func (t *Timeline) applyOrientation() {
	h := t.orientation == TimelineHorizontal
	t.EnableHorizon(h)
	if h {
		t.SetAlignV(xcc.Layout_Align_Top)
	} else {
		t.SetAlignH(xcc.Layout_Align_Left)
	}
}

// SetBackdrop 指定轴线/节点底下铺的宿主底色(默认按"放在 Card 上"用 t.Card)。
func (t *Timeline) SetBackdrop(fn func(*xui.Theme) xui.Color) *Timeline {
	if t == nil {
		return t
	}
	t.backdrop = fn
	for _, it := range t.items {
		it.Redraw(false)
	}
	return t
}

// AddItem 追加一个时间轴条目, 返回条目容器(继续往里加内容)。
func (t *Timeline) AddItem(opts ...TimelineItemOption) *TimelineItem {
	if t == nil {
		return nil
	}
	ele := widget.NewLayoutEle(0, 0, 0, 0, t.Handle)
	if ele == nil {
		return nil
	}
	it := &TimelineItem{LayoutEle: ele, timeline: t, last: true}
	it.side = timelineSideAt(t.orientation, t.mode, len(t.items))
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(timelineContentG)
	if t.orientation == TimelineHorizontal {
		// 先按 1fr 参与首轮布局, 尺寸确定后由 reflow 锁成等宽 Fixed
		// (横向 Weight 列不会把宽度下发给 Fill 子项, 见 DESIGN 第八节 101)。
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	} else {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	}
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ele.AddEvent_Paint(it.onPaint)
	for _, o := range opts {
		o(it)
	}
	it.buildText()
	// 上一条从"末条"变成"中间条": 补上底部间距(轴线要画满这段距离)。
	if n := len(t.items); n > 0 {
		prev := t.items[n-1]
		prev.last = false
		if t.orientation == TimelineVertical {
			prev.applyVerticalPadding(t.GetWidth())
		}
		prev.Redraw(false)
	}
	t.items = append(t.items, it)
	if t.orientation == TimelineVertical {
		it.applyVerticalPadding(t.GetWidth())
	}
	t.AdjustLayout()
	// 构建期只排自己: 此时窗口往往还没开始第一轮布局, 同步上推祖先会触发重入。
	// 新高度交给下一轮消息里的 reflow 推给外层(needsReflow 为假时也要推, 故用
	// forceAdjust 而不是只按取向/模式判断)。
	t.forceAdjust = true
	t.scheduleReflow()
	return it
}

// TimelineItem 是时间轴里的一个条目, 本身是内容布局容器。
// 内置的 时间/标题/说明 依次创建为子 Label; 需要更丰富的内容时继续往里加元素:
//
//	it := tl.AddItem(ui.WithTimelineItemTitle("已发货"))
//	track := ui.NewItem(0, 0, 0, 0, it.Handle, ui.WithItemVariant(ui.ItemOutline))
//	xui.ItemFill(track)
type TimelineItem struct {
	*widget.LayoutEle
	timeline *Timeline
	last     bool
	side     timelineSide

	timeText  string
	titleText string
	descText  string
	color     TimelineColor
	colorFn   func(*xui.Theme) xui.Color
	icon      string
	loading   bool

	contentH  int32 // 水平取向实测的内容高(不含上下内边距)
	padTop    int32
	padLeft   int32
	padRight  int32
	padBottom int32
	axisY     int32 // 水平取向的轴线相对条目顶部的偏移
}

// TimelineItemOption 时间轴条目配置项。
type TimelineItemOption func(*TimelineItem)

// WithTimelineItemTime 在条目顶部加一行弱化的小字时间(text-xs, muted)。
func WithTimelineItemTime(text string) TimelineItemOption {
	return func(it *TimelineItem) { it.timeText = text }
}

// WithTimelineItemTitle 加一行标题(text-sm, medium)。
func WithTimelineItemTitle(text string) TimelineItemOption {
	return func(it *TimelineItem) { it.titleText = text }
}

// WithTimelineItemDescription 加一行弱化说明(text-sm, muted, 单行省略)。
func WithTimelineItemDescription(text string) TimelineItemOption {
	return func(it *TimelineItem) { it.descText = text }
}

// WithTimelineItemColor 设置节点语义色(默认主色)。文字颜色不受影响。
func WithTimelineItemColor(c TimelineColor) TimelineItemOption {
	return func(it *TimelineItem) { it.color = c }
}

// WithTimelineItemColorFunc 用主题令牌自定义节点色(优先级高于 WithTimelineItemColor),
// 例如让节点跟随图表色板。每次绘制实时求值, 不要在 fn 里缓存颜色。
func WithTimelineItemColorFunc(fn func(*xui.Theme) xui.Color) TimelineItemOption {
	return func(it *TimelineItem) { it.colorFn = fn }
}

// WithTimelineItemIcon 用自定义图标(lucide 路径或整段 SVG)代替圆点, 图标颜色跟随节点色。
func WithTimelineItemIcon(data string) TimelineItemOption {
	return func(it *TimelineItem) { it.icon = data }
}

// WithTimelineItemLoading 把节点画成旋转的加载指示器(优先于图标与圆点)。
// 运行中可再用 SetLoading 切换。
func WithTimelineItemLoading(loading bool) TimelineItemOption {
	return func(it *TimelineItem) { it.loading = loading }
}

// SetLoading 切换加载态节点。关闭时显式停掉运行时旋转定时器。
func (it *TimelineItem) SetLoading(loading bool) *TimelineItem {
	if it == nil || it.loading == loading {
		return it
	}
	it.loading = loading
	if !loading {
		xui.Spin(it.Handle, false)
	}
	it.Redraw(false)
	return it
}

// Loading 报告条目节点是否处于加载态。
func (it *TimelineItem) Loading() bool {
	return it != nil && it.loading
}

// buildText 按选项依次创建 时间/标题/说明 三个 Label(没给就不建, 不占布局)。
// 竖向 end / alternate 的左侧内容靠右对齐, 文字朝向轴线。
func (it *TimelineItem) buildText() {
	alignRight := it.timeline != nil && it.timeline.orientation == TimelineVertical && it.side == timelineLeft
	add := func(h int32, text string, setup func(*Label)) {
		l := NewLabel(0, 0, 0, h, text, it.Handle)
		setup(l)
		if alignRight {
			l.SetTextAlign((l.GetTextAlign() &^ xcc.TextAlignFlag_Left) | xcc.TextAlignFlag_Right)
		}
		xui.ItemFill(l)
	}
	if it.timeText != "" {
		add(timelineTimeH, it.timeText, func(l *Label) { l.SetMuted().SetFontSize(12) })
	}
	if it.titleText != "" {
		add(timelineTitleH, it.titleText, func(l *Label) { l.SetMedium() })
	}
	if it.descText != "" {
		add(timelineDescH, it.descText, func(l *Label) { l.SetMuted() })
	}
}

// setPadding 写入内边距, 返回是否真的变了(没变时不触发外层重排, 避免自激)。
func (it *TimelineItem) setPadding(l, tp, r, b int32) bool {
	if it.padLeft == l && it.padTop == tp && it.padRight == r && it.padBottom == b {
		return false
	}
	it.padLeft, it.padTop, it.padRight, it.padBottom = l, tp, r, b
	xc.XEle_SetPadding(it.Handle, l, tp, r, b)
	return true
}

// applyVerticalPadding 竖向: 内容边让位节点列, 底部内边距承载条目间距(末条为 0)。
// 返回是否改变了内边距。
func (it *TimelineItem) applyVerticalPadding(width int32) bool {
	if it.timeline == nil {
		return false
	}
	l, tp, r, b := timelineVerticalPadding(width, it.timeline.mode, it.side, it.last, it.timeline.gap)
	return it.setPadding(l, tp, r, b)
}

// backdropColor 返回轴线/节点底下铺的宿主底色(默认 Card)。
func (it *TimelineItem) backdropColor(t *xui.Theme) xui.Color {
	if it.timeline != nil && it.timeline.backdrop != nil {
		return it.timeline.backdrop(t)
	}
	return t.Card
}

// nodeColor 解析节点色: 自定义函数 > 语义色 > 主色。
func (it *TimelineItem) nodeColor(t *xui.Theme) xui.Color {
	if it.colorFn != nil {
		return it.colorFn(t)
	}
	switch it.color {
	case TimelineColorDestructive:
		return t.Destructive
	case TimelineColorMuted:
		return t.MutedForeground
	default:
		return t.Primary
	}
}

// nodeOffset 返回圆点中心相对本条顶部的偏移: 对齐**第一个可见子项**的垂直中心。
// 量不到(还没排版/没有子项)时用兜底值, 避免圆点贴在最顶上。
func (it *TimelineItem) nodeOffset() int32 {
	var own xc.RECT
	xc.XEle_GetWndClientRect(it.Handle, &own)
	n := xc.XEle_GetChildCount(it.Handle)
	for i := int32(0); i < n; i++ {
		h := xc.XEle_GetChildByIndex(it.Handle, i)
		if h == 0 || !xc.XWidget_IsShow(h) {
			continue
		}
		var rc xc.RECT
		xc.XEle_GetWndClientRect(h, &rc)
		if rc.Bottom > rc.Top {
			return (rc.Top+rc.Bottom)/2 - own.Top
		}
		break
	}
	return timelineFallbackY
}

func (it *TimelineItem) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	bg := it.backdropColor(t)

	if it.timeline != nil && it.timeline.orientation == TimelineHorizontal {
		h := p.Rc.Bottom - p.Rc.Top
		y := it.axisY
		if y <= 0 {
			// reflow 尚未跑过: 按模式给一个合理位置, 下一次排完就稳定。
			switch it.timeline.mode {
			case TimelineEnd:
				y = h - timelineNodeX
			case TimelineAlternate:
				y = h / 2
			default:
				y = timelineNodeX
			}
		}
		y += p.Rc.Top
		x := p.Rc.Left + timelineNodeX
		// 擦掉轴线带上的上一帧线头; 内容距轴线 20, 不会盖到文字。
		band := xc.RECT{
			Left: p.Rc.Left, Top: y - timelineNodeX,
			Right: p.Rc.Right, Bottom: y + timelineNodeX,
		}
		if band.Bottom > band.Top {
			p.FillRound(&band, bg, 0)
		}
		if !it.isFirst() {
			p.Line(p.Rc.Left, y, x, y, t.Border, timelineLineW)
		}
		if !it.last {
			p.Line(x, y, p.Rc.Right, y, t.Border, timelineLineW)
		}
		it.drawNode(p, x, y, t, bg)
		return 0
	}

	// 竖向: 轴线位置由模式决定(start 左 / end 右 / alternate 居中)。
	var x int32
	switch it.timeline.mode {
	case TimelineEnd:
		x = p.Rc.Right - timelineNodeX
	case TimelineAlternate:
		x = p.Rc.Left + (p.Rc.Right-p.Rc.Left)/2
	default:
		x = p.Rc.Left + timelineNodeX
	}
	// 先铺节点列底色: 线条长度会随条目高度变化, 不擦底会留下上一帧的线头。
	strip := xc.RECT{
		Left: x - timelineNodeX, Top: p.Rc.Top,
		Right: x + timelineNodeX, Bottom: p.Rc.Bottom,
	}
	if strip.Right > strip.Left {
		p.FillRound(&strip, bg, 0)
	}
	y := p.Rc.Top + it.nodeOffset()
	// 首条不画上方线, 末条不画下方线; 其余各段在相邻条目间自然接上
	// (条目间距是底部内边距, 属于条目矩形, 线会画满)。
	if !it.isFirst() {
		p.Line(x, p.Rc.Top, x, y, t.Border, timelineLineW)
	}
	if !it.last {
		p.Line(x, y, x, p.Rc.Bottom, t.Border, timelineLineW)
	}
	it.drawNode(p, x, y, t, bg)
	return 0
}

// drawNode 画节点: 加载 > 自定义图标 > 空心环 > 实心圆点。
// 每层都先用宿主底色盖住穿过节点的轴线。
func (it *TimelineItem) drawNode(p *xui.Painter, x, y int32, t *xui.Theme, bg xui.Color) {
	color := it.nodeColor(t)
	circle := func(r int32) xc.RECT {
		return xc.RECT{Left: x - r, Top: y - r, Right: x + r, Bottom: y + r}
	}
	fill := func(r int32, c xui.Color) {
		rc := circle(r)
		p.Draw.SetBrushColor(uint32(c))
		p.Draw.FillEllipse(&rc)
	}
	switch {
	case it.loading:
		// 相位由运行时按墙钟算并驱动重绘, 不要自己起定时器(同 Spinner/Button)。
		fill(timelineGlyphR, bg)
		rc := circle(timelineGlyphR)
		p.IconRotated(icons.IconLoader2, &rc, color, 2, xui.Spin(it.Handle, true)*2*math.Pi)
	case it.icon != "":
		fill(timelineGlyphR, bg)
		rc := circle(timelineGlyphR)
		p.Icon(it.icon, &rc, color, 2)
	case it.timeline != nil && it.timeline.variant == TimelineOutlined:
		fill(timelineDotHalo, bg)
		fill(timelineRingR, color)
		fill(timelineRingHole, bg)
	default:
		fill(timelineDotHalo, bg)
		fill(timelineDotR, color)
	}
}

// isFirst 报告是否时间轴里的第一条。
func (it *TimelineItem) isFirst() bool {
	if it.timeline == nil || len(it.timeline.items) == 0 {
		return true
	}
	return it.timeline.items[0] == it
}

// needsReflow 报告尺寸变化时是否要重算内边距:
// 竖向 start 是静态布局; end/alternate 与水平取向依赖容器宽度或实测内容高。
func (t *Timeline) needsReflow() bool {
	return t.orientation == TimelineHorizontal || t.mode != TimelineStart
}

// onSize 尺寸变化后排队重排(改内边距会再触发尺寸事件, 由 reflowing 去重)。
func (t *Timeline) onSize(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
	if t == nil || t.reflowing || !t.needsReflow() {
		return 0
	}
	w, h := t.GetWidth(), t.GetHeight()
	if w == t.lastW && h == t.lastH {
		return 0
	}
	t.lastW, t.lastH = w, h
	if w > 0 {
		t.scheduleReflow()
	}
	return 0
}

// scheduleReflow 把重排推迟到下一轮消息: 在布局趟里改子项内边距/宽度会重入。
func (t *Timeline) scheduleReflow() {
	if t == nil || t.queued {
		return
	}
	t.queued = true
	app.AfterFunc(0, func() {
		t.queued = false
		if t == nil || t.LayoutEle == nil || !xc.XC_IsHXCGUI(t.Handle, xcc.XC_ELE) {
			return
		}
		t.reflow()
	})
}

func (t *Timeline) reflow() {
	if t == nil || t.LayoutEle == nil || t.reflowing {
		return
	}
	w := t.GetWidth()
	if w <= 0 {
		return
	}
	t.reflowing = true
	defer func() { t.reflowing = false }()
	changed := false
	if t.orientation == TimelineHorizontal {
		changed = t.reflowHorizontal(w)
	} else {
		for _, it := range t.items {
			if it.applyVerticalPadding(w) {
				changed = true
			}
		}
	}
	t.lastW, t.lastH = t.GetWidth(), t.GetHeight()
	// 计算值没变且没有增删条目就什么都不做。布局过渡期会额外触发一轮尺寸事件,
	// 每轮都上推祖先会让 ScrollArea 的滚动条显隐与内容高互相驱动, 变成无限重排
	// (同 AspectRatio)。
	if !changed && !t.forceAdjust {
		return
	}
	t.forceAdjust = false
	t.adjustAncestors()
	for _, it := range t.items {
		it.Redraw(false)
	}
	t.lastW, t.lastH = t.GetWidth(), t.GetHeight()
}

// reflowHorizontal 等分条目宽度, 量出各条内容高, 再按共享轴线写上下内边距。
// 条目高度 = 内容高 + 内边距, 全部等于同一条目总高, 轴线才能在一条水平线上。
// 返回是否改变了宽度或内边距(只改 axisY 时只重绘, 不重排)。
func (t *Timeline) reflowHorizontal(w int32) bool {
	n := int32(len(t.items))
	if n == 0 {
		return false
	}
	iw := w / n
	if iw < 1 {
		iw = 1
	}
	// 先锁宽度, 内容(Fill 子项)才拿得到宽度, 下面的高度量得准。
	changed := false
	if iw != t.itemW {
		for _, it := range t.items {
			it.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, iw)
		}
		t.itemW = iw
		t.AdjustLayout()
		changed = true
	}

	var maxAbove, maxBelow int32
	for _, it := range t.items {
		h := it.GetHeight() - it.padTop - it.padBottom
		if h < 0 {
			h = 0
		}
		it.contentH = h
		if it.side == timelineAbove {
			if h > maxAbove {
				maxAbove = h
			}
		} else if h > maxBelow {
			maxBelow = h
		}
	}
	axisY, _ := timelineHorizontalGeometry(maxAbove, maxBelow, t.mode)
	axisChanged := false
	for _, it := range t.items {
		top, bottom := timelineHorizontalPadding(it.contentH, maxAbove, maxBelow, t.mode, it.side)
		if it.setPadding(0, top, 0, bottom) {
			changed = true
		}
		if it.axisY != axisY {
			it.axisY = axisY
			axisChanged = true
		}
	}
	if axisChanged && !changed {
		for _, it := range t.items {
			it.Redraw(false)
		}
	}
	return changed
}

// adjustAncestors 把最近的 ScrollArea 框架(或窗口)重排一次, 让 Auto 高的
// 卡片/行拿到新的时间轴高度。不要在父链上逐级 AdjustLayout: 中间层的 Weight 布局
// 在量算趟里会反复改动宽度, 尺寸事件因此自激(同 AspectRatio)。
func (t *Timeline) adjustAncestors() {
	if t == nil || t.LayoutEle == nil {
		return
	}
	xc.XEle_AdjustLayout(t.Handle)
	for p := xc.XWidget_GetParentEle(t.Handle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(p)
			return
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			// p 是框架的内容区: 直接对它 AdjustLayout 会把子项宽度打成 0(实测),
			// 必须交给框架排。
			xc.XEle_AdjustLayout(pp)
			return
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
			return
		}
	}
}

// timelineSideAt 按取向与模式给出第 idx 条内容所在侧。
func timelineSideAt(o TimelineOrientation, m TimelineMode, idx int) timelineSide {
	if o == TimelineHorizontal {
		switch m {
		case TimelineStart:
			return timelineBelow
		case TimelineEnd:
			return timelineAbove
		default:
			if idx%2 == 0 {
				return timelineAbove
			}
			return timelineBelow
		}
	}
	switch m {
	case TimelineStart:
		return timelineRight
	case TimelineEnd:
		return timelineLeft
	default:
		if idx%2 == 0 {
			return timelineLeft
		}
		return timelineRight
	}
}

// timelineVerticalPadding 计算竖向条目的内边距:
// start 左侧让位节点列; end 右侧让位; alternate 轴线居中, 左右各占一半再退一个间距。
// 底部内边距承载条目间距(末条为 0), 这样轴线能画满间隙与下一条接上。
func timelineVerticalPadding(w int32, mode TimelineMode, side timelineSide, last bool, gap int32) (l, tp, r, b int32) {
	if !last {
		b = gap
	}
	switch mode {
	case TimelineStart:
		l = timelineNodeCol
	case TimelineEnd:
		r = timelineNodeCol
	case TimelineAlternate:
		if w <= 0 {
			// 还没排版: 先按左侧布局, reflow 后会按真实宽度改到居中。
			w = timelineNodeCol * 2
		}
		half := w / 2
		if side == timelineLeft {
			r = w - half + timelineAxisGap
		} else {
			l = half + timelineAxisGap
		}
	}
	return
}

// timelineHorizontalGeometry 由两侧最大内容高推出共享轴线位置与条目总高。
func timelineHorizontalGeometry(maxAbove, maxBelow int32, mode TimelineMode) (axisY, total int32) {
	switch mode {
	case TimelineEnd:
		total = maxAbove + timelineAxisGap + timelineNodeX
		axisY = total - timelineNodeX
	case TimelineAlternate:
		total = maxAbove + timelineAxisGap + timelineAxisGap + maxBelow
		axisY = maxAbove + timelineAxisGap
	default:
		total = timelineNodeX + timelineAxisGap + maxBelow
		axisY = timelineNodeX
	}
	return
}

// timelineHorizontalPadding 计算水平条目的上下内边距, 使内容边缘距轴线一个
// timelineAxisGap, 且上下内边距 + 内容高 = 条目总高(所有条目同高)。
func timelineHorizontalPadding(contentH, maxAbove, maxBelow int32, mode TimelineMode, side timelineSide) (top, bottom int32) {
	axisY, total := timelineHorizontalGeometry(maxAbove, maxBelow, mode)
	if side == timelineAbove {
		top = axisY - timelineAxisGap - contentH
		bottom = total - (axisY - timelineAxisGap)
	} else {
		top = axisY + timelineAxisGap
		bottom = total - top - contentH
	}
	if top < 0 {
		top = 0
	}
	if bottom < 0 {
		bottom = 0
	}
	return
}
