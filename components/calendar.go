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
//   - **年月可选**: 标题行拆成 "2026年 ⌄" / "10月 ⌄" 两个热区, 点开是贴着标题的自绘
//     下拉列表(月 12 项, 年 = 视图年 ±50 可配), 滚轮滚动 + 2px 滚动条, 当前值右侧打勾;
//     改动只切显示月份, 不动选中日期(react-day-picker 的 captionLayout="dropdown" 语义);
//   - **禁用日期**: WithCalendarDisabled 传判定函数(内置禁用过去 / 周末 / 日期数组 / 组合
//     四个 helper), 禁用格 text-muted-foreground opacity-50, 不可点、无悬停、不参与选中;
//   - **页脚**: WithCalendarFooter / WithCalendarFooterToday(内置"今天"), 或构造后
//     AddFooterItem / AddFooterButton / AddFooterText 追加; 页脚项自绘(幽灵按钮:
//     悬停 bg-accent), 需要放原生控件时用 FooterRect() 取内容区矩形自行摆放子元素。
type Calendar struct {
	*widget.Element

	view      time.Time    // 当前显示的月份(取 1 号)
	sel       time.Time    // 选中日期, 零值 = 未选择
	weekStart time.Weekday // 一周起始日, 默认周一(中文习惯)
	onSelect  func(time.Time)
	flat      bool                 // 不画卡片底/边框(由所在弹层负责), 供 popover 内嵌
	disabled  func(time.Time) bool // 禁用日期判定, nil = 全部可选

	// ---- 页脚 ----
	footer []CalendarFooterItem
	autoH  bool // 构造时未指定高度: 增删页脚项时元素自己长高 / 缩回

	// ---- 年月下拉 ----
	yearFrom, yearTo int     // 年份范围, 0/0 = 视图年 ±50
	dropKind         int     // calDropNone / calDropYear / calDropMonth
	dropVals         []int   // 下拉项的值(年份或 1..12 的月份)
	dropScroll       int32   // 滚动偏移(像素, 吸附到整行)
	dropRowH         int32   // 行高(按可用高度自适应, 保证 12 行放得下)
	dropViewH        int32   // 可视高度
	dropContentH     int32   // 内容总高
	dropRect         xc.RECT // 面板矩形(客户区坐标)

	// 绘制时更新的热区几何(命中判定直接复用, 客户区坐标)
	hdrYear, hdrMonth xc.RECT
	footRects         []xc.RECT

	hoverIdx  int32 // 悬停的格子索引(0..41), -1 = 无
	hoverBtn  int32 // 悬停的翻页按钮(0 上/1 下), -1 = 无
	hoverHead int32 // 悬停的标题热区(0 年/1 月), -1 = 无
	hoverDrop int32 // 悬停的下拉项索引, -1 = 无
	hoverFoot int32 // 悬停的页脚项索引, -1 = 无
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

	calFootH    int32 = 37 // 页脚: 1px 分隔线 + 上 6 + 按钮 24 + 下 6
	calFootGap  int32 = 8  // 页脚项间距
	calFootBtnH int32 = 24 // 页脚按钮高(ghost size-sm)

	calDropPad    int32 = 4  // 下拉面板内边距 p-1
	calDropGap    int32 = 2  // 标题 → 面板
	calDropRowMin int32 = 16 // 行高下限(再小就滚动)
	calDropRowMax int32 = 26 // 行高上限
	calDropMinW   int32 = 72 // 面板最小宽度
)

// 下拉面板类型。
const (
	calDropNone = iota
	calDropYear
	calDropMonth
)

// hover 过渡键: 0..41 留给日期格, 其余从这里往后取(不要互相撞车, 撞了会共用过渡值)。
const (
	calHoverPrev  int32 = 100
	calHoverNext  int32 = 101
	calHoverYear  int32 = 102
	calHoverMonth int32 = 103
	calHoverFoot  int32 = 120 // 120+i: 页脚项
	calHoverDrop  int32 = 200 // 200+i: 下拉项(年份最多 100+ 项, 不会撞到页脚)
)

// CalWidth 返回月历卡片的默认宽度(逻辑像素)。
func CalWidth() int32 { return calPad*2 + 7*calCell }

// CalHeight 返回月历卡片的默认高度(逻辑像素, 不含页脚)。
func CalHeight() int32 {
	return calPad*2 + calTitleH + calGapTitle + calWeekH + calGapWeek + int32(calRows)*calCell + int32(calRows-1)*calRowGap
}

// CalFooterHeight 返回启用页脚时卡片额外增加的高度(1px 分隔线 + 上下 6 + 24 按钮)。
// 自行给元素指定高度(cy > 0)又想用页脚时, 高度要写成 CalHeight()+CalFooterHeight()。
func CalFooterHeight() int32 { return calFootH }

// CalendarFooterItem 页脚项: Text 是显示文本; OnClick 非空时是可点按钮(悬停有底),
// 为空时是纯文本; Disabled 只对按钮生效。
type CalendarFooterItem struct {
	Text     string
	OnClick  func()
	Disabled bool
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

// WithCalendarDisabled 设置禁用日期判定: 返回 true 的日期灰显且不可选
// (react-day-picker 的 disabled matcher)。可直接用 CalendarDisable 系列 helper 组合。
func WithCalendarDisabled(fn func(time.Time) bool) CalendarOption {
	return func(c *Calendar) { c.disabled = fn }
}

// WithCalendarYearRange 限定年份下拉的范围(闭区间)。不设置时为"视图年 ±50",
// 列表打开时自动滚到当前年。
func WithCalendarYearRange(from, to int) CalendarOption {
	return func(c *Calendar) {
		if from > to {
			from, to = to, from
		}
		c.yearFrom, c.yearTo = from, to
	}
}

// WithCalendarFooter 启用页脚并追加页脚项(可传多组, 每次追加到已有项之后)。
// 页脚自带 1px 分隔线, 项在卡片底部水平居中; 元素未指定高度时会自动长高。
func WithCalendarFooter(items ...CalendarFooterItem) CalendarOption {
	return func(c *Calendar) { c.footer = append(c.footer, items...) }
}

// WithCalendarFooterToday 启用页脚并放一个内置"今天"按钮: 切到今天所在月份,
// 今天未被禁用时同时选中并回调(与 shadcn calendar 的 footer Today 示例一致)。
func WithCalendarFooterToday() CalendarOption {
	return func(c *Calendar) {
		c.footer = append(c.footer, CalendarFooterItem{Text: "今天", OnClick: c.goToday})
	}
}

// CalendarDisableBefore 返回"禁用 t 之前(按天比较)"的判定函数 —— 今天可用,
// 昨天的所有日期灰显。
func CalendarDisableBefore(t time.Time) func(time.Time) bool {
	limit := calDay(t)
	return func(d time.Time) bool { return calDay(d).Before(limit) }
}

// CalendarDisableWeekend 判定周六/周日(直接当判定函数用)。
func CalendarDisableWeekend(d time.Time) bool {
	wd := d.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// CalendarDisableDates 返回"禁用 dates 里列出的这些天(按年月日比较)"的判定函数。
func CalendarDisableDates(dates ...time.Time) func(time.Time) bool {
	set := make(map[int]bool, len(dates))
	for _, d := range dates {
		set[calDayKey(d)] = true
	}
	return func(d time.Time) bool { return set[calDayKey(d)] }
}

// CalendarDisableAny 组合多个判定函数: 任一命中即禁用。
func CalendarDisableAny(fns ...func(time.Time) bool) func(time.Time) bool {
	return func(d time.Time) bool {
		for _, fn := range fns {
			if fn != nil && fn(d) {
				return true
			}
		}
		return false
	}
}

// calDay 抹掉时分秒(保留日期的日历日语义)。
func calDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// calDayKey 把日期压成 y*10000+m*100+d, 便于做集合判定。
func calDayKey(t time.Time) int { return t.Year()*10000 + int(t.Month())*100 + t.Day() }

// NewCalendar 创建月历卡片。cx/cy 传 0 使用默认尺寸(248×292, 启用页脚时高度再加
// CalFooterHeight())。
func NewCalendar(x, y, cx, cy int32, hParent int, opts ...CalendarOption) *Calendar {
	now := time.Now()
	c := &Calendar{
		view:      time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local),
		weekStart: time.Monday,
		hoverIdx:  -1,
		hoverBtn:  -1,
		hoverHead: -1,
		hoverDrop: -1,
		hoverFoot: -1,
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
	c.autoH = cy <= 0
	if cy <= 0 {
		cy = CalHeight() + c.footExtra()
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
	e.AddEvent_MouseWheel(c.onMouseWheel)
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

// Today 切到今天所在月份; 今天未被禁用时同时选中并触发 OnSelect 回调。
func (c *Calendar) Today() *Calendar {
	now := time.Now()
	c.view = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	today := calDay(now)
	if !c.dateDisabled(today) {
		c.sel = today
		if c.onSelect != nil {
			c.onSelect(today)
		}
	}
	c.Redraw(false)
	return c
}

// goToday 是内置"今天"页脚按钮的动作。
func (c *Calendar) goToday() { c.Today() }

// ViewMonth 返回当前显示的年月。
func (c *Calendar) ViewMonth() (year int, month time.Month) {
	return c.view.Year(), c.view.Month()
}

// OnSelect 注册日期选中回调(选中的是本月还是 outside 日期都会回调;
// 禁用日期不会触发)。
func (c *Calendar) OnSelect(fn func(time.Time)) *Calendar {
	c.onSelect = fn
	return c
}

// SetDisabled 运行时替换禁用日期判定(传 nil = 全部可选)。
func (c *Calendar) SetDisabled(fn func(time.Time) bool) *Calendar {
	c.disabled = fn
	c.Redraw(false)
	return c
}

// SetYearRange 运行时修改年份下拉范围(闭区间), 传 0/0 恢复"视图年 ±50"。
func (c *Calendar) SetYearRange(from, to int) *Calendar {
	if from > to {
		from, to = to, from
	}
	c.yearFrom, c.yearTo = from, to
	if c.dropKind == calDropYear { // 列表开着就按新范围重建
		c.rebuildDropVals()
		c.scrollDropToCurrent()
	}
	c.Redraw(false)
	return c
}

// ---- 页脚 ----

// AddFooterItem 追加一个页脚项(首次追加即启用页脚)。
func (c *Calendar) AddFooterItem(it CalendarFooterItem) *Calendar {
	c.footer = append(c.footer, it)
	c.syncHeight()
	c.Redraw(false)
	return c
}

// AddFooterButton 追加一个页脚按钮。
func (c *Calendar) AddFooterButton(text string, onClick func()) *Calendar {
	return c.AddFooterItem(CalendarFooterItem{Text: text, OnClick: onClick})
}

// AddFooterText 追加一段页脚静态文本。
func (c *Calendar) AddFooterText(text string) *Calendar {
	return c.AddFooterItem(CalendarFooterItem{Text: text})
}

// ClearFooter 清空页脚(自动长高的元素会缩回原高度)。
func (c *Calendar) ClearFooter() *Calendar {
	if len(c.footer) == 0 {
		return c
	}
	c.footer = nil
	c.footRects = nil
	c.hoverFoot = -1
	c.syncHeight()
	c.Redraw(false)
	return c
}

// FooterRect 返回页脚内容区矩形(元素客户区坐标), 用于在页脚里摆**自定义原生控件**
// (作为月历的子元素创建, 盖在自绘内容之上, 点击不会穿透到月历)。
// 未启用页脚时返回零矩形。
func (c *Calendar) FooterRect() xc.RECT {
	if len(c.footer) == 0 || c.Element == nil {
		return xc.RECT{}
	}
	top := c.GetHeight() - calFootH + 1 + 6
	return xc.RECT{
		Left:   calPad,
		Top:    top,
		Right:  c.GetWidth() - calPad,
		Bottom: top + calFootBtnH,
	}
}

// footExtra 返回页脚占用的额外高度(无页脚项时 0)。
func (c *Calendar) footExtra() int32 {
	if len(c.footer) == 0 {
		return 0
	}
	return calFootH
}

// syncHeight 在页脚项增删后按需调整元素高度(仅构造时未指定高度的实例;
// 指定了高度的实例由调用方自己预留 CalFooterHeight())。
func (c *Calendar) syncHeight() {
	if c.autoH && c.Element != nil {
		c.SetHeight(CalHeight() + c.footExtra())
	}
}

// dateDisabled 报告某个日期是否被禁用(未设置判定函数时恒为 false)。
func (c *Calendar) dateDisabled(d time.Time) bool {
	return c.disabled != nil && c.disabled(d)
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

// headerHit 判断坐标是否落在标题的年 / 月热区上: 0 = 年, 1 = 月, -1 = 都不是。
func (c *Calendar) headerHit(x, y int32) int32 {
	if calInRect(c.hdrYear, x, y) {
		return 0
	}
	if calInRect(c.hdrMonth, x, y) {
		return 1
	}
	return -1
}

// footHit 判断坐标是否落在某个页脚项上(纯文本项也返回索引, 由调用方决定是否响应)。
func (c *Calendar) footHit(x, y int32) int32 {
	for i, rc := range c.footRects {
		if calInRect(rc, x, y) {
			return int32(i)
		}
	}
	return -1
}

func calInRect(rc xc.RECT, x, y int32) bool {
	return x >= rc.Left && x < rc.Right && y >= rc.Top && y < rc.Bottom
}

func (c *Calendar) onLButtonUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	x, y := pPt.X, pPt.Y

	// 下拉开着时它独占点击: 面板内 = 取值, 面板外 = 关闭(不穿透到日期/按钮)
	if c.dropKind != calDropNone {
		if idx := c.dropIndexAt(x, y); idx >= 0 {
			c.applyDrop(c.dropVals[idx])
			c.closeDrop()
			return 0
		}
		// 点另一个标题热区: 一次点击就切成另一个列表(与原生 select 一致)
		if head := c.headerHit(x, y); head >= 0 {
			kind := calDropYear
			if head == 1 {
				kind = calDropMonth
			}
			if kind != c.dropKind {
				c.toggleDrop(kind)
				return 0
			}
		}
		c.closeDrop()
		return 0
	}

	if head := c.headerHit(x, y); head >= 0 {
		if head == 0 {
			c.toggleDrop(calDropYear)
		} else {
			c.toggleDrop(calDropMonth)
		}
		return 0
	}
	if b := c.hitButton(x, y, c.GetWidth()); b >= 0 {
		if b == 0 {
			c.PrevMonth()
		} else {
			c.NextMonth()
		}
		return 0
	}
	if fi := c.footHit(x, y); fi >= 0 {
		it := c.footer[fi]
		if it.OnClick != nil && !it.Disabled {
			it.OnClick()
		}
		return 0
	}
	idx := c.indexAt(x, y)
	if idx < 0 {
		return 0
	}
	d := c.dateAt(idx)
	if c.dateDisabled(d) {
		return 0
	}
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
	x, y := pPt.X, pPt.Y

	if c.dropKind != calDropNone {
		idx := c.dropIndexAt(x, y)
		changed := idx != c.hoverDrop
		if c.hoverIdx != -1 || c.hoverBtn != -1 || c.hoverHead != -1 || c.hoverFoot != -1 {
			c.hoverIdx, c.hoverBtn, c.hoverHead, c.hoverFoot = -1, -1, -1, -1
			changed = true
		}
		if changed {
			c.hoverDrop = idx
			c.Redraw(false)
		}
		return 0
	}

	idx := c.indexAt(x, y)
	if idx >= 0 && c.dateDisabled(c.dateAt(idx)) {
		idx = -1 // 禁用格不高亮
	}
	btn := c.hitButton(x, y, c.GetWidth())
	head := c.headerHit(x, y)
	foot := c.footHit(x, y)
	if foot >= 0 {
		it := c.footer[foot]
		if it.OnClick == nil || it.Disabled {
			foot = -1
		}
	}
	if idx != c.hoverIdx || btn != c.hoverBtn || head != c.hoverHead || foot != c.hoverFoot {
		c.hoverIdx, c.hoverBtn, c.hoverHead, c.hoverFoot = idx, btn, head, foot
		c.Redraw(false)
	}
	return 0
}

func (c *Calendar) onMouseLeave(hEle int, hEleStay int, pbHandled *bool) int {
	// 鼠标离开卡片 = 点了"外部": 收起年月下拉(自绘下拉没有原生弹层的失焦机制)
	if c.dropKind != calDropNone {
		c.closeDrop()
	}
	if c.hoverIdx != -1 || c.hoverBtn != -1 || c.hoverHead != -1 || c.hoverFoot != -1 || c.hoverDrop != -1 {
		c.hoverIdx, c.hoverBtn, c.hoverHead, c.hoverFoot, c.hoverDrop = -1, -1, -1, -1, -1
		c.Redraw(false)
	}
	return 0
}

// onMouseWheel 只在下拉打开时接管滚轮(关闭时会关掉元素的滚轮事件, 让滚轮
// 继续交给外层滚动容器, 见 DESIGN 8.66 的 ScrollArea)。
func (c *Calendar) onMouseWheel(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if c.dropKind == calDropNone || c.dropContentH <= c.dropViewH {
		return 0
	}
	delta := int32(int16(uint16(nFlags >> 16)))
	if delta == 0 {
		return 0
	}
	step := c.dropRowH * 3
	if delta > 0 {
		c.dropScroll -= step
	} else {
		c.dropScroll += step
	}
	c.clampDropScroll()
	c.hoverDrop = c.dropIndexAt(pPt.X, pPt.Y)
	c.Redraw(false)
	*pbHandled = true
	return 0
}

// ---- 年月下拉 ----

// toggleDrop 打开 / 收起某一类下拉(再点同一个标题热区就是收起)。
func (c *Calendar) toggleDrop(kind int) {
	if c.dropKind == kind {
		c.closeDrop()
		return
	}
	c.dropKind = kind
	c.rebuildDropVals()
	c.hoverDrop = -1
	c.EnableEvent_XE_MOUSEWHEEL(true)
	c.layoutDrop()
	c.scrollDropToCurrent()
	c.Redraw(false)
}

// closeDrop 收起下拉。
func (c *Calendar) closeDrop() {
	if c.dropKind == calDropNone {
		return
	}
	c.dropKind = calDropNone
	c.dropVals = nil
	c.dropScroll = 0
	c.hoverDrop = -1
	c.EnableEvent_XE_MOUSEWHEEL(false)
	c.Redraw(false)
}

// rebuildDropVals 按当前类型生成下拉项: 月 1..12, 年 = 设定范围或视图年 ±50。
func (c *Calendar) rebuildDropVals() {
	if c.dropKind == calDropMonth {
		c.dropVals = c.dropVals[:0]
		for m := 1; m <= 12; m++ {
			c.dropVals = append(c.dropVals, m)
		}
		return
	}
	from, to := c.yearFrom, c.yearTo
	if c.yearFrom == 0 && c.yearTo == 0 {
		from, to = c.view.Year()-50, c.view.Year()+50
	}
	c.dropVals = c.dropVals[:0]
	for y := from; y <= to; y++ {
		c.dropVals = append(c.dropVals, y)
	}
}

// applyDrop 把下拉的选中值落到显示月份上(不动选中日期)。
func (c *Calendar) applyDrop(v int) {
	if c.dropKind == calDropYear {
		c.view = time.Date(v, c.view.Month(), 1, 0, 0, 0, 0, time.Local)
	} else {
		c.view = time.Date(c.view.Year(), time.Month(v), 1, 0, 0, 0, 0, time.Local)
	}
}

// dropText 返回下拉项显示文本。
func (c *Calendar) dropText(v int) string {
	if c.dropKind == calDropYear {
		return fmt.Sprintf("%d年", v)
	}
	return fmt.Sprintf("%d月", v)
}

// dropCurrent 报告下拉项是否为当前显示的年 / 月(右侧打勾)。
func (c *Calendar) dropCurrent(v int) bool {
	if c.dropKind == calDropYear {
		return v == c.view.Year()
	}
	return v == int(c.view.Month())
}

// layoutDrop 计算下拉面板几何。面板贴着被点的标题热区下方, 左右夹在卡片内;
// 行高按"12 行正好塞满可用高度"自适应(16..26), 放不下就滚动。
func (c *Calendar) layoutDrop() bool {
	if c.dropKind == calDropNone || c.Element == nil {
		return false
	}
	w, h := c.GetWidth(), c.GetHeight()
	anchor := c.hdrMonth
	if c.dropKind == calDropYear {
		anchor = c.hdrYear
	}
	if anchor.Bottom == 0 { // 还没绘制过: 退化成标题行下沿
		anchor = xc.RECT{Left: calPad, Top: calPad + calTitleH, Right: w - calPad, Bottom: calPad + calTitleH}
	}

	f := xui.FontHandle(xui.FontSpec{Family: xui.Current().FontDefault.Family, Size: xui.Current().FontDefault.Size})
	maxW := int32(0)
	for _, v := range c.dropVals {
		if tw := xui.TextWidth(c.dropText(v), f); tw > maxW {
			maxW = tw
		}
	}
	pw := maxW + 40 // 左 6 + 右 22(勾选图标) + 左右内边距
	if pw < calDropMinW {
		pw = calDropMinW
	}
	px := anchor.Left
	if px+pw > w-8 {
		px = w - 8 - pw
	}
	if px < 8 {
		px = 8
	}
	py := anchor.Bottom + calDropGap
	avail := h - 4 - py
	if avail < calDropRowMin+calDropPad*2+4 { // 下方实在太窄: 贴顶铺开
		py = 4
		avail = h - 8
	}

	rowH := (avail - calDropPad*2) / 12
	if rowH > calDropRowMax {
		rowH = calDropRowMax
	}
	if rowH < calDropRowMin {
		rowH = calDropRowMin
	}
	contentH := int32(len(c.dropVals))*rowH + calDropPad*2
	viewH := contentH
	if viewH > avail {
		viewH = avail
	}
	if viewH < rowH+calDropPad*2 {
		viewH = rowH + calDropPad*2
	}
	c.dropRect = xc.RECT{Left: px, Top: py, Right: px + pw, Bottom: py + viewH}
	c.dropRowH, c.dropViewH, c.dropContentH = rowH, viewH, contentH
	c.clampDropScroll()
	return true
}

// dropRowsVisible 返回可视区域里完整行数。
func (c *Calendar) dropRowsVisible() int32 {
	if c.dropRowH <= 0 {
		return 0
	}
	return (c.dropViewH - calDropPad*2) / c.dropRowH
}

// clampDropScroll 把滚动偏移吸附到整行并夹在可达范围内。
func (c *Calendar) clampDropScroll() {
	if c.dropRowH <= 0 {
		c.dropScroll = 0
		return
	}
	maxScroll := (int32(len(c.dropVals)) - c.dropRowsVisible()) * c.dropRowH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if c.dropScroll > maxScroll {
		c.dropScroll = maxScroll
	}
	if c.dropScroll < 0 {
		c.dropScroll = 0
	}
	c.dropScroll = c.dropScroll / c.dropRowH * c.dropRowH
	if c.dropScroll > maxScroll {
		c.dropScroll = maxScroll
	}
}

// scrollDropToCurrent 打开时把当前年 / 月滚到可视区中间。
func (c *Calendar) scrollDropToCurrent() {
	cur := int(c.view.Month())
	if c.dropKind == calDropYear {
		cur = c.view.Year()
	}
	idx := -1
	for i, v := range c.dropVals {
		if v == cur {
			idx = i
			break
		}
	}
	if idx < 0 || c.dropContentH <= c.dropViewH {
		c.dropScroll = 0
		return
	}
	c.dropScroll = int32(idx)*c.dropRowH - (c.dropViewH-c.dropRowH)/2
	c.clampDropScroll()
}

// dropIndexAt 把坐标换算成下拉项索引, 不在完整可见的行上时返回 -1。
func (c *Calendar) dropIndexAt(x, y int32) int32 {
	if c.dropKind == calDropNone || c.dropRowH <= 0 || !calInRect(c.dropRect, x, y) {
		return -1
	}
	top := c.dropRect.Top + calDropPad
	if y < top || y >= c.dropRect.Bottom-calDropPad {
		return -1
	}
	idx := (y - top + c.dropScroll) / c.dropRowH
	if idx < 0 || idx >= int32(len(c.dropVals)) {
		return -1
	}
	// 只认完整可见的行(面板上下可能露半行, 半行不响应)
	y0 := top + idx*c.dropRowH - c.dropScroll
	if y0 < top || y0+c.dropRowH > c.dropRect.Bottom-calDropPad {
		return -1
	}
	return idx
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

	// ---- 标题行: [上一月]  2026年⌄ 10月⌄  [下一月] ----
	titleTop := top + calPad
	prev := xc.RECT{Left: left + calPad, Top: titleTop, Right: left + calPad + calCell, Bottom: titleTop + calTitleH}
	next := xc.RECT{Left: right - calPad - calCell, Top: titleTop, Right: right - calPad, Bottom: titleTop + calTitleH}

	if hv := xui.HoverItem(hEle, calHoverPrev, c.hoverBtn == 0); hv > 0 {
		p.FillRound(&prev, t.Accent.ScaleAlpha(hv), t.RadiusSm())
	}
	if hv := xui.HoverItem(hEle, calHoverNext, c.hoverBtn == 1); hv > 0 {
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

	c.paintHeader(p, t, prev.Right, next.Left, titleTop)

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
		dis := c.dateDisabled(d)

		// 禁用格按 react-day-picker 的 `.rdp-disabled`: 整格 opacity-50, 无悬停反馈。
		// 选中/今天标识仍画, 但一起淡下去(否则一个"看起来可选"的实心块点不动)。
		half := func(c0 xui.Color) xui.Color { return c0.ScaleAlpha(0.5) }
		switch {
		case isSel && dis:
			p.FillRound(&rc, half(t.Primary), t.RadiusSm())
		case isSel:
			p.FillRound(&rc, t.Primary, t.RadiusSm())
		case isToday && dis:
			p.FillRound(&rc, half(t.Accent), t.RadiusSm())
		case isToday:
			// shadcn: today = bg-accent text-accent-foreground
			p.FillRound(&rc, t.Accent, t.RadiusSm())
		case !dis:
			if hv := xui.HoverItem(hEle, idx, c.hoverIdx == idx); hv > 0 {
				p.FillRound(&rc, t.Accent.ScaleAlpha(hv), t.RadiusSm())
			}
		}

		fore := t.Foreground
		switch {
		case dis:
			fore = t.MutedForeground.ScaleAlpha(0.5)
			if isSel {
				fore = t.PrimaryForeground.ScaleAlpha(0.5)
			}
		case isSel:
			fore = t.PrimaryForeground
		case !inMonth:
			fore = t.MutedForeground
		}
		p.Text(strconv.Itoa(d.Day()), &rc, fore, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	}

	// ---- 页脚 ----
	c.paintFooter(p, t)

	// ---- 年月下拉(最后画, 盖住日期与页脚) ----
	c.paintDrop(p, t)
	return 0
}

// paintHeader 画标题中部的年月两个热区, 并记下它们的客户区矩形供命中判定复用。
func (c *Calendar) paintHeader(p *xui.Painter, t *xui.Theme, left, right, titleTop int32) {
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size, Bold: true})
	if f != 0 {
		p.Draw.SetFont(f)
	}
	yearText := fmt.Sprintf("%d年", c.view.Year())
	monthText := fmt.Sprintf("%d月", int(c.view.Month()))
	yearW := xui.TextWidth(yearText, f) + 24
	monthW := xui.TextWidth(monthText, f) + 24

	const gap = 2
	x := left + (right-left-yearW-gap-monthW)/2
	const h = int32(24)
	y := titleTop + (calTitleH-h)/2
	c.hdrYear = xc.RECT{Left: x, Top: y, Right: x + yearW, Bottom: y + h}
	c.hdrMonth = xc.RECT{Left: c.hdrYear.Right + gap, Top: y, Right: c.hdrYear.Right + gap + monthW, Bottom: y + h}

	c.paintHeadLabel(p, t, c.hdrYear, yearText, calHoverYear, c.hoverHead == 0 || c.dropKind == calDropYear)
	c.paintHeadLabel(p, t, c.hdrMonth, monthText, calHoverMonth, c.hoverHead == 1 || c.dropKind == calDropMonth)
}

// paintHeadLabel 画一个"文字 + chevron-down"的标题按钮(打开时保持 bg-accent)。
func (c *Calendar) paintHeadLabel(p *xui.Painter, t *xui.Theme, rc xc.RECT, text string, key int32, active bool) {
	hv := xui.HoverItem(c.Handle, key, active)
	if hv > 0 {
		p.FillRound(&rc, t.Accent.ScaleAlpha(hv), t.RadiusSm())
	}
	fore := xui.Fade(t.Foreground, t.AccentForeground, hv)
	chev := xui.Fade(t.MutedForeground, t.AccentForeground, hv)

	tr := rc
	tr.Left += 6
	tr.Right -= 18
	p.Text(text, &tr, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)

	const chevSize = int32(10)
	ir := xc.RECT{
		Left:   rc.Right - 6 - chevSize,
		Top:    rc.Top + (rc.Bottom-rc.Top-chevSize)/2,
		Right:  rc.Right - 6,
		Bottom: rc.Top + (rc.Bottom-rc.Top+chevSize)/2,
	}
	p.Icon(icons.IconChevronDown, &ir, chev, 2)
}

// paintFooter 画页脚: 1px 分隔线(卡片全宽) + 水平居中、间距 8 的一组项。
func (c *Calendar) paintFooter(p *xui.Painter, t *xui.Theme) {
	if len(c.footer) == 0 {
		return
	}
	footerTop := p.Rc.Bottom - calFootH
	p.Line(p.Rc.Left, footerTop, p.Rc.Right, footerTop, t.Border, 1)

	f := xui.FontHandle(t.FontDefault)
	if f != 0 {
		p.Draw.SetFont(f)
	}
	contentTop := footerTop + 1 + 6
	widths := make([]int32, len(c.footer))
	total := int32(0)
	for i, it := range c.footer {
		w := xui.TextWidth(it.Text, f)
		if it.OnClick != nil {
			w += 16 // px-2
		}
		widths[i] = w
		total += w
	}
	total += int32(len(c.footer)-1) * calFootGap
	x := p.Rc.Left + (p.W()-total)/2

	c.footRects = c.footRects[:0]
	for i, it := range c.footer {
		rc := xc.RECT{Left: x, Top: contentTop, Right: x + widths[i], Bottom: contentTop + calFootBtnH}
		c.footRects = append(c.footRects, rc)
		if it.OnClick != nil {
			hv := xui.HoverItem(c.Handle, calHoverFoot+int32(i), c.hoverFoot == int32(i) && !it.Disabled)
			if hv > 0 {
				p.FillRound(&rc, t.Accent.ScaleAlpha(hv), t.RadiusSm())
			}
			fore := xui.Fade(t.MutedForeground, t.AccentForeground, hv)
			if it.Disabled {
				fore = t.MutedForeground.ScaleAlpha(0.5)
			}
			p.Text(it.Text, &rc, fore, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		} else {
			p.Text(it.Text, &rc, t.MutedForeground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		}
		x = rc.Right + calFootGap
	}
}

// paintDrop 画年月下拉面板: bg-popover rounded-md + ring, 悬停 bg-accent,
// 当前值右侧打勾; 内容超出可视高度时右侧画 2px 滚动条(滚轮滚动)。
func (c *Calendar) paintDrop(p *xui.Painter, t *xui.Theme) {
	if !c.layoutDrop() {
		return
	}
	rc := c.dropRect
	p.FillRound(&rc, t.Popover, t.RadiusMd())
	p.StrokeRound(&rc, t.Foreground.WithAlpha(26), t.RadiusMd(), 1)

	viewTop := rc.Top + calDropPad
	viewBottom := rc.Bottom - calDropPad
	scrollable := c.dropContentH > c.dropViewH

	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	for i, v := range c.dropVals {
		y0 := viewTop + int32(i)*c.dropRowH - c.dropScroll
		y1 := y0 + c.dropRowH
		if y0 < viewTop || y1 > viewBottom {
			continue // 半行不画: 面板自带底色, 不会露出破绽
		}
		item := xc.RECT{Left: rc.Left + calDropPad, Top: y0, Right: rc.Right - calDropPad, Bottom: y1}
		if scrollable {
			item.Right -= 4
		}
		hv := xui.HoverItem(c.Handle, calHoverDrop+int32(i), c.hoverDrop == int32(i))
		fore := t.PopoverForeground
		if hv > 0 {
			p.FillRound(&item, t.Accent.ScaleAlpha(hv), t.RadiusSm())
			fore = xui.Fade(fore, t.AccentForeground, hv)
		}
		tr := item
		tr.Left += 6
		tr.Right -= 22
		p.Text(c.dropText(v), &tr, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)

		if c.dropCurrent(v) {
			const cs = int32(14)
			ir := xc.RECT{
				Left:   item.Right - 6 - cs,
				Top:    item.Top + (item.Bottom-item.Top-cs)/2,
				Right:  item.Right - 6,
				Bottom: item.Top + (item.Bottom-item.Top+cs)/2,
			}
			p.Icon(icons.IconCheck, &ir, fore, 2)
		}
	}

	if scrollable {
		trackH := viewBottom - viewTop
		thumbH := trackH * c.dropViewH / c.dropContentH
		if thumbH < 16 {
			thumbH = 16
		}
		maxScroll := c.dropContentH - c.dropViewH
		y := viewTop
		if maxScroll > 0 {
			y += (trackH - thumbH) * c.dropScroll / maxScroll
		}
		bar := xc.RECT{Left: rc.Right - 5, Top: y, Right: rc.Right - 3, Bottom: y + thumbH}
		p.FillRound(&bar, t.Foreground.WithAlpha(38), 1)
	}
}
