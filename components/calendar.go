package ui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Calendar 月历卡片: 完全自绘(shadcn calendar 语义), 无原生 MonthCal 载体 ——
// 原生月历的格子是 DLL 内绘制, 圆角/格子比例/标题位置全部不可控(探针实测:
// BkM 的 SetRectRoundAngle 对项背景无效, 选中格只能是直角大色块), 做不到
// rounded-md + 32×32 方格的视觉, 所以这里把日期计算、命中、hover 都自己做。
//
// 与 shadcn calendar.tsx(react-day-picker v9 的 classNames)对齐:
//   - 卡片: bg-popover rounded-lg + ring-1 ring-foreground/10(popover 弹层同款皮肤);
//   - 标题行高 = 格高(32), 月份文字居中 font-medium text-sm, 左右各一个
//     ghost 翻页按钮(size-8 rounded-md, 悬停 bg-accent, chevron 图标);
//   - 星期行 text-[0.8rem] text-muted-foreground, 列宽 = 格宽;
//   - 日期格 32×32, 圆角 rounded-md(= 本主题 RadiusSm, 6), text-sm: 选中 bg-primary +
//     text-primary-foreground, 今天 bg-accent, 悬停 bg-accent(带 150ms 过渡),
//     非本月 text-muted-foreground;
//   - 固定 6 行: 月份切换时卡片高度不跳(弹层尺寸恒定), 越界日期按 outside 显示。
type Calendar struct {
	*widget.Element

	view      time.Time     // 当前显示的月份(取 1 号)
	sel       time.Time     // 选中日期, 零值 = 未选择
	weekStart time.Weekday  // 一周起始日, 默认周一(中文习惯)
	onSelect  func(time.Time)
	flat      bool          // 不画卡片底/边框(由所在弹层负责), 供 popover 内嵌

	hoverIdx int32 // 悬停的格子索引(0..41), -1 = 无
	hoverBtn int32 // 悬停的翻页按钮(0 上/1 下), -1 = 无
}

// 月历几何(逻辑像素)。cell 取 32 = shadcn 默认 --cell-size: 2rem。
const (
	calCell     int32 = 32
	calPad      int32 = 12 // 卡片内边距 p-3
	calTitleH   int32 = 32
	calWeekH    int32 = 26
	calRowGap   int32 = 2 // 行间距 mt-2
	calGapTitle int32 = 4 // 标题行 → 星期行
	calGapWeek  int32 = 4 // 星期行 → 日期区
	calRows     int   = 6
)

// 翻页按钮的 hover 过渡键: 0..41 留给日期格, 这里取两个固定值避开。
const (
	calKeyPrev int32 = 100
	calKeyNext int32 = 101
)

// CalWidth 返回月历卡片的默认宽度(逻辑像素)。
func CalWidth() int32 { return calPad*2 + 7*calCell }

// CalHeight 返回月历卡片的默认高度(逻辑像素)。
func CalHeight() int32 {
	return calPad*2 + calTitleH + calGapTitle + calWeekH + calGapWeek + int32(calRows)*calCell + int32(calRows-1)*calRowGap
}

// CalendarOption 月历配置项。
type CalendarOption func(*Calendar)

// WithCalendarSelected 设置初始选中日期。
func WithCalendarSelected(t time.Time) CalendarOption {
	return func(c *Calendar) { c.sel = t }
}

// WithCalendarWeekStart 设置一周起始日(默认 time.Monday)。
func WithCalendarWeekStart(w time.Weekday) CalendarOption {
	return func(c *Calendar) { c.weekStart = w }
}

// WithCalendarOnSelect 注册日期选中回调。
func WithCalendarOnSelect(fn func(time.Time)) CalendarOption {
	return func(c *Calendar) { c.onSelect = fn }
}

// WithCalendarFlat 让月历不画自己的卡片底与边框 —— 内嵌在 popover 弹层里时用,
// 避免与弹层卡片画出双层边框(shadcn 的 calendar 放在 popover 里就是这样)。
func WithCalendarFlat() CalendarOption {
	return func(c *Calendar) { c.flat = true }
}

// NewCalendar 创建月历卡片。cx/cy 传 0 使用默认尺寸(248×292)。
func NewCalendar(x, y, cx, cy int32, hParent int, opts ...CalendarOption) *Calendar {
	now := time.Now()
	c := &Calendar{
		view:      time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local),
		weekStart: time.Monday,
		hoverIdx:  -1,
		hoverBtn:  -1,
	}
	for _, o := range opts {
		o(c)
	}
	if c.sel.IsZero() {
		// 未选择: 显示当月
	} else {
		c.view = time.Date(c.sel.Year(), c.sel.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	if cx <= 0 {
		cx = CalWidth()
	}
	if cy <= 0 {
		cy = CalHeight()
	}
	e := widget.NewElement(x, y, cx, cy, hParent)
	if e == nil {
		return nil
	}
	c.Element = e
	e.EnableBkTransparent(true)
	e.AddEvent_Paint(c.onPaint)
	e.AddEvent_LButtonUp(c.onLButtonUp)
	e.AddEvent_MouseMove(c.onMouseMove)
	e.AddEvent_MouseLeave(c.onMouseLeave)
	xui.AttachWindow(e.Handle)
	return c
}

// Selected 返回当前选中日期, 未选择时为零值。
func (c *Calendar) Selected() time.Time { return c.sel }

// SetSelected 设置选中日期, 同时把显示月份切到该日期所在月。传零值 = 清除选择。
func (c *Calendar) SetSelected(t time.Time) *Calendar {
	c.sel = t
	if !t.IsZero() {
		c.view = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	c.Redraw(false)
	return c
}

// SetMonth 切换显示月份(不改选中日期)。
func (c *Calendar) SetMonth(year int, month time.Month) *Calendar {
	c.view = time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	c.Redraw(false)
	return c
}

// NextMonth 显示下一个月。
func (c *Calendar) NextMonth() *Calendar {
	c.view = c.view.AddDate(0, 1, 0)
	c.Redraw(false)
	return c
}

// PrevMonth 显示上一个月。
func (c *Calendar) PrevMonth() *Calendar {
	c.view = c.view.AddDate(0, -1, 0)
	c.Redraw(false)
	return c
}

// ViewMonth 返回当前显示的年月。
func (c *Calendar) ViewMonth() (year int, month time.Month) {
	return c.view.Year(), c.view.Month()
}

// OnSelect 注册日期选中回调(选中的是本月还是 outside 日期都会回调)。
func (c *Calendar) OnSelect(fn func(time.Time)) *Calendar {
	c.onSelect = fn
	return c
}

// dateAt 返回第 idx 个格子(0..41)对应的日期。
//
// 网格起点 = 当月 1 号回退到"所在周的第一天"(weekStart 决定), 之后顺序排 42 天。
func (c *Calendar) dateAt(idx int32) time.Time {
	first := time.Date(c.view.Year(), c.view.Month(), 1, 0, 0, 0, 0, time.Local)
	offset := (int(first.Weekday()) - int(c.weekStart) + 7) % 7
	start := first.AddDate(0, 0, -offset)
	return start.AddDate(0, 0, int(idx))
}

// gridTop 返回日期区顶边(元素客户区坐标)。
func (c *Calendar) gridTop() int32 {
	return calPad + calTitleH + calGapTitle + calWeekH + calGapWeek
}

// indexAt 把元素客户区坐标换算成格子索引, 不在格子上时返回 -1。
func (c *Calendar) indexAt(x, y int32) int32 {
	top := c.gridTop()
	if y < top {
		return -1
	}
	row := (y - top) / (calCell + calRowGap)
	if (y-top)-row*(calCell+calRowGap) >= calCell { // 落在行间隙
		return -1
	}
	if x < calPad {
		return -1
	}
	col := (x - calPad) / calCell
	if row < 0 || row >= int32(calRows) || col < 0 || col >= 7 {
		return -1
	}
	return row*7 + col
}

// hitButton 判断坐标是否落在翻页按钮上: 0 = 上一月, 1 = 下一月, -1 = 都不是。
func (c *Calendar) hitButton(x, y int32, width int32) int32 {
	if y < calPad || y > calPad+calTitleH {
		return -1
	}
	if x >= calPad && x < calPad+calCell {
		return 0
	}
	if x >= width-calPad-calCell && x < width-calPad {
		return 1
	}
	return -1
}

func (c *Calendar) onLButtonUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if b := c.hitButton(pPt.X, pPt.Y, c.GetWidth()); b >= 0 {
		if b == 0 {
			c.PrevMonth()
		} else {
			c.NextMonth()
		}
		return 0
	}
	idx := c.indexAt(pPt.X, pPt.Y)
	if idx < 0 {
		return 0
	}
	d := c.dateAt(idx)
	c.sel = d
	// 点到 outside 日期: 跟随翻到目标月(shadcn 的行为)
	if d.Month() != c.view.Month() || d.Year() != c.view.Year() {
		c.view = time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	c.Redraw(false)
	if c.onSelect != nil {
		c.onSelect(d)
	}
	return 0
}

func (c *Calendar) onMouseMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	idx := c.indexAt(pPt.X, pPt.Y)
	btn := c.hitButton(pPt.X, pPt.Y, c.GetWidth())
	if idx != c.hoverIdx || btn != c.hoverBtn {
		c.hoverIdx, c.hoverBtn = idx, btn
		c.Redraw(false)
	}
	return 0
}

func (c *Calendar) onMouseLeave(hEle int, hEleStay int, pbHandled *bool) int {
	if c.hoverIdx != -1 || c.hoverBtn != -1 {
		c.hoverIdx, c.hoverBtn = -1, -1
		c.Redraw(false)
	}
	return 0
}

// weekNames 是星期缩写, 索引 0 = 周日。
var weekNames = [7]string{"日", "一", "二", "三", "四", "五", "六"}

func (c *Calendar) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()

	// 卡片: bg-popover + ring-1 ring-foreground/10(shadcn 的 popover 皮肤)。
	// flat 模式(内嵌在 popover 弹层里)由弹层负责画, 这里跳过, 免出双层边框。
	if !c.flat {
		p.FillRound(&p.Rc, t.Popover, t.RadiusLg())
		p.StrokeRound(&p.Rc, t.Foreground.WithAlpha(26), t.RadiusLg(), 1)
	}

	left, top := p.Rc.Left, p.Rc.Top
	right := p.Rc.Right

	// ---- 标题行: [上一月]  2026年10月  [下一月] ----
	titleTop := top + calPad
	prev := xc.RECT{Left: left + calPad, Top: titleTop, Right: left + calPad + calCell, Bottom: titleTop + calTitleH}
	next := xc.RECT{Left: right - calPad - calCell, Top: titleTop, Right: right - calPad, Bottom: titleTop + calTitleH}

	if hv := xui.HoverItem(hEle, calKeyPrev, c.hoverBtn == 0); hv > 0 {
		p.FillRound(&prev, t.Accent.ScaleAlpha(hv), t.RadiusSm())
	}
	if hv := xui.HoverItem(hEle, calKeyNext, c.hoverBtn == 1); hv > 0 {
		p.FillRound(&next, t.Accent.ScaleAlpha(hv), t.RadiusSm())
	}
	iconSize := int32(16)
	prevIcon := xc.RECT{
		Left:   prev.Left + (calCell-iconSize)/2,
		Top:    prev.Top + (calTitleH-iconSize)/2,
		Right:  prev.Left + (calCell+iconSize)/2,
		Bottom: prev.Top + (calTitleH+iconSize)/2,
	}
	nextIcon := xc.RECT{
		Left:   next.Left + (calCell-iconSize)/2,
		Top:    next.Top + (calTitleH-iconSize)/2,
		Right:  next.Left + (calCell+iconSize)/2,
		Bottom: next.Top + (calTitleH+iconSize)/2,
	}
	p.Icon(icons.IconChevronLeft, &prevIcon, t.Foreground, 2)
	p.Icon(icons.IconChevronRight, &nextIcon, t.Foreground, 2)

	if f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size, Bold: true}); f != 0 {
		p.Draw.SetFont(f)
	}
	title := xc.RECT{Left: prev.Right, Top: titleTop, Right: next.Left, Bottom: titleTop + calTitleH}
	p.Text(fmt.Sprintf("%d年%d月", c.view.Year(), int(c.view.Month())), &title, t.Foreground,
		xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)

	// ---- 星期行 ----
	weekTop := titleTop + calTitleH + calGapTitle
	if f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size - 2}); f != 0 {
		p.Draw.SetFont(f)
	}
	for i := int32(0); i < 7; i++ {
		rc := xc.RECT{
			Left:   left + calPad + i*calCell,
			Top:    weekTop,
			Right:  left + calPad + (i+1)*calCell,
			Bottom: weekTop + calWeekH,
		}
		name := weekNames[(int(c.weekStart)+int(i))%7]
		p.Text(name, &rc, t.MutedForeground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	}

	// ---- 日期格 ----
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	now := time.Now()
	gridTop := c.gridTop()
	for idx := int32(0); idx < int32(calRows*7); idx++ {
		d := c.dateAt(idx)
		row, col := idx/7, idx%7
		rc := xc.RECT{
			Left:   left + calPad + col*calCell,
			Top:    gridTop + row*(calCell+calRowGap),
			Right:  left + calPad + (col+1)*calCell,
			Bottom: gridTop + row*(calCell+calRowGap) + calCell,
		}
		inMonth := d.Month() == c.view.Month() && d.Year() == c.view.Year()
		isSel := !c.sel.IsZero() && d.Year() == c.sel.Year() && d.YearDay() == c.sel.YearDay()
		isToday := d.Year() == now.Year() && d.YearDay() == now.YearDay()

		switch {
		case isSel:
			p.FillRound(&rc, t.Primary, t.RadiusSm())
		case isToday:
			// shadcn: today = bg-accent text-accent-foreground
			p.FillRound(&rc, t.Accent, t.RadiusSm())
		default:
			if hv := xui.HoverItem(hEle, idx, c.hoverIdx == idx); hv > 0 {
				p.FillRound(&rc, t.Accent.ScaleAlpha(hv), t.RadiusSm())
			}
		}

		fore := t.Foreground
		switch {
		case isSel:
			fore = t.PrimaryForeground
		case !inMonth:
			fore = t.MutedForeground
		}
		p.Text(strconv.Itoa(d.Day()), &rc, fore, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	}
	return 0
}
