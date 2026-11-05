package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 时间轴几何常量(逻辑像素)。
//
// shadcn 无官方 timeline, 语义按社区通行做法定死(见 DESIGN 第四节末):
//   - 竖向轴线在左侧, 每个条目一个实心主色圆点, 圆点与首行内容垂直居中对齐;
//   - 条目内容是一个竖向布局, 可放内置的 时间/标题/说明, 也可直接再加 Label / Item / Card;
//   - 轴线颜色取 border, 圆点取 primary; 首条不画上方线、末条不画下方线;
//   - 条目之间的间距用元素底部内边距表达(线要画满这段距离才能与下一条接上)。
const (
	timelineNodeCol   int32 = 32 // 左侧节点列宽(内容从这条线之后开始)
	timelineNodeX     int32 = 12 // 轴线/圆点的圆心 x
	timelineLineW     int32 = 2  // 轴线宽度
	timelineDotHalo   int32 = 7  // 圆点外的底色光环半径(盖住穿过圆点的轴线)
	timelineDotR      int32 = 5  // 实心圆点半径
	timelineTimeH     int32 = 16 // text-xs 行高
	timelineTitleH    int32 = 20 // text-sm 行高
	timelineDescH     int32 = 20 // text-sm 行高
	timelineContentG  int32 = 4  // 条目内各行之间的间距
	timelineItemGap   int32 = 24 // 条目之间的默认间距
	timelineFallbackY int32 = 10 // 没量到首个子项时圆点中心的兜底位置
)

// Timeline 竖向时间轴容器: 一个竖向布局, 每个条目是一个 TimelineItem。
//
// 内容槽就是 TimelineItem 本身(它是竖向 LayoutEle): 内置的 时间/标题/说明 只是往里面
// 放三个 Label, 之后仍可继续加任意 Label / Item / Card —— 条目高度随内容自撑,
// 轴线按条目实际高度一路接下去。
type Timeline struct {
	*widget.LayoutEle

	items    []*TimelineItem
	gap      int32
	backdrop func(*xui.Theme) xui.Color
}

// TimelineOption 时间轴配置项。
type TimelineOption func(*Timeline)

// WithTimelineGap 设置条目之间的间距(默认 24)。会给所有非末条目的底部内边距。
func WithTimelineGap(px int32) TimelineOption {
	return func(t *Timeline) {
		if px >= 0 {
			t.gap = px
		}
	}
}

// WithTimelineBackdrop 指定节点列底下铺的宿主底色(默认为 t.Card)。
// 轴线/圆点只画局部形状, 每帧先用宿主底色擦掉上一帧; 放进非 Card 底色的容器时用它跟随宿主。
func WithTimelineBackdrop(fn func(t *xui.Theme) xui.Color) TimelineOption {
	return func(t *Timeline) { t.backdrop = fn }
}

// NewTimeline 创建时间轴容器。宽铺满父级、高随条目自撑(cx/cy 只作初始尺寸)。
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
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(0)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.AttachWindow(ele.Handle)
	return t
}

// SetBackdrop 指定节点列底下铺的宿主底色(默认按"放在 Card 上"用 t.Card)。
func (t *Timeline) SetBackdrop(fn func(t *xui.Theme) xui.Color) *Timeline {
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
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(timelineContentG)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
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
		prev.applyPadding()
		prev.Redraw(false)
	}
	it.applyPadding()
	t.items = append(t.items, it)
	t.AdjustLayout()
	return it
}

// TimelineItem 是时间轴里的一个条目, 本身是竖向布局的内容槽。
// 内置的 时间/标题/说明 依次创建为子 Label; 需要更丰富的内容时继续往里加元素:
//
//	it := tl.AddItem(ui.WithTimelineItemTitle("已发货"))
//	track := ui.NewItem(0, 0, 0, 0, it.Handle, ui.WithItemVariant(ui.ItemOutline))
//	xui.ItemFill(track)
type TimelineItem struct {
	*widget.LayoutEle
	timeline *Timeline
	last     bool

	timeText  string
	titleText string
	descText  string
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

// applyPadding 左内边距给节点列留位; 底部内边距是条目间距(末条为 0)。
func (it *TimelineItem) applyPadding() {
	bottom := it.timeline.gap
	if it.last {
		bottom = 0
	}
	xc.XEle_SetPadding(it.Handle, timelineNodeCol, 0, 0, bottom)
}

// buildText 按选项依次创建 时间/标题/说明 三个 Label(没给就不建, 不占布局)。
func (it *TimelineItem) buildText() {
	if it.timeText != "" {
		l := NewLabel(0, 0, 0, timelineTimeH, it.timeText, it.Handle)
		l.SetMuted().SetFontSize(12)
		xui.ItemFill(l)
	}
	if it.titleText != "" {
		l := NewLabel(0, 0, 0, timelineTitleH, it.titleText, it.Handle)
		l.SetMedium()
		xui.ItemFill(l)
	}
	if it.descText != "" {
		l := NewLabel(0, 0, 0, timelineDescH, it.descText, it.Handle)
		l.SetMuted()
		xui.ItemFill(l)
	}
}

// backdropColor 返回节点列底色(默认 Card)。
func (it *TimelineItem) backdropColor(t *xui.Theme) xui.Color {
	if it.timeline != nil && it.timeline.backdrop != nil {
		return it.timeline.backdrop(t)
	}
	return t.Card
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

	// 先铺节点列底色: 线条长度会随条目高度变化, 不擦底会留下上一帧的线头。
	strip := p.Rc
	strip.Right = strip.Left + timelineNodeCol
	if strip.Right > strip.Left {
		p.FillRound(&strip, it.backdropColor(t), 0)
	}

	x := p.Rc.Left + timelineNodeX
	y := p.Rc.Top + it.nodeOffset()
	// 首条不画上方线, 末条不画下方线; 其余各段在相邻条目间自然接上
	// (条目间距是底部内边距, 属于条目矩形, 线会画满)。
	if !it.isFirst() {
		p.Line(x, p.Rc.Top, x, y, t.Border, timelineLineW)
	}
	if !it.last {
		p.Line(x, y, x, p.Rc.Bottom, t.Border, timelineLineW)
	}

	// 圆点: 先画宿主底色的光环盖住穿过圆点的轴线, 再画主色实心圆。
	halo := xc.RECT{
		Left: x - timelineDotHalo, Top: y - timelineDotHalo,
		Right: x + timelineDotHalo, Bottom: y + timelineDotHalo,
	}
	p.Draw.SetBrushColor(uint32(it.backdropColor(t)))
	p.Draw.FillEllipse(&halo)
	dot := xc.RECT{
		Left: x - timelineDotR, Top: y - timelineDotR,
		Right: x + timelineDotR, Bottom: y + timelineDotR,
	}
	p.Draw.SetBrushColor(uint32(t.Primary))
	p.Draw.FillEllipse(&dot)
	return 0
}

// isFirst 报告是否时间轴里的第一条。
func (it *TimelineItem) isFirst() bool {
	if it.timeline == nil || len(it.timeline.items) == 0 {
		return true
	}
	return it.timeline.items[0] == it
}
