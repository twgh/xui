package ui

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/twgh/xcgui/app"
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
//   - **范围选择**(mode="range"): 两次点击落 from/to(第二次早于第一次时自动交换),
//     区间中段 bg-accent + 文字 accent-foreground, 两端实心 primary; 完整选中后再点
//     是"重新开始"。区间两端同一天时只画一个实心格(react-day-picker 的 addToRange 语义)。
//   - **多个月**(numberOfMonths): 1~4 个月并排(默认 1), 翻页按钮在整块左右两端,
//     每个月各有自己的年月下拉; 点击只改被点的那个月。
//   - **周数**(showWeekNumber): 每月左侧多一列 ISO 周数(周一为一周起点;
//     周日开头的行取该行里的周一算周数), 宽度 = 一个格子。
//   - **年月可选**: 标题行拆成 "2026年 ⌄" / "10月 ⌄" 两个热区, 点开是贴着标题的自绘
//     下拉列表(月 12 项, 年 = 视图年 ±50 可配), 滚轮滚动 + 2px 滚动条, 当前值右侧打勾;
//     改动只切显示月份, 不动选中日期(react-day-picker 的 captionLayout="dropdown" 语义);
//   - **禁用日期**: WithCalendarDisabled 传判定函数(内置禁用过去 / 未来 / 周末 / 日期数组 /
//     组合 helper), 禁用格 text-muted-foreground opacity-50, 不可点、无悬停、不参与选中;
//     预订日期这类"仍显示但划线"用 WithCalendarStrike(不参与命中判定)。
//   - **页脚 / 预设**: WithCalendarFooter / WithCalendarFooterToday / WithCalendarPresets;
//     页脚自动换行(按钮数多时高度跟着涨), 项自绘(幽灵按钮, 悬停 bg-accent)。
type Calendar struct {
	*widget.Element

	view      time.Time    // 第一个可见月份(取 1 号)
	sel       time.Time    // 单选选中日期, 零值 = 未选择
	weekStart time.Weekday // 一周起始日, 默认周一(中文习惯)
	onSelect  func(time.Time)
	flat      bool                 // 不画卡片底/边框(由所在弹层负责), 供 popover 内嵌
	disabled  func(time.Time) bool // 禁用日期判定, nil = 全部可选

	mode      int // calModeSingle / calModeRange
	months    int // 并排显示几个月(1~4)
	weekNums  bool
	rangeFrom time.Time
	rangeTo   time.Time
	onRange   func(from, to time.Time)
	strike    func(time.Time) bool // 划线日期(预订/占用), 不影响可点性

	// ---- 页脚 ----
	footer      []CalendarFooterItem
	autoH       bool // 构造时未指定高度: 增删页脚项时元素自己长高 / 缩回
	syncPending bool // 已预约一次"按宽度重算高度"(尺寸事件里不能直接重入布局)

	// ---- 年月下拉 ----
	yearFrom, yearTo int     // 年份范围, 0/0 = 视图年 ±50
	dropKind         int     // calDropNone / calDropYear / calDropMonth
	dropMonth        int     // 下拉所属的月份块索引(多个月时只改这一块)
	dropVals         []int   // 下拉项的值(年份或 1..12 的月份)
	dropScroll       int32   // 滚动偏移(像素, 吸附到整行)
	dropRowH         int32   // 行高(按可用高度自适应, 保证 12 行放得下)
	dropViewH        int32   // 可视高度
	dropContentH     int32   // 内容总高
	dropRect         xc.RECT // 面板矩形(客户区坐标)
	dropPad          int32   // 面板内边距(随元素缩放)

	// 绘制时更新的热区几何(命中判定直接复用, 客户区坐标)
	hdrYear, hdrMonth []xc.RECT
	footRects         []xc.RECT

	hoverIdx  int32 // 悬停的格子索引(月份块*42 + 0..41), -1 = 无
	hoverBtn  int32 // 悬停的翻页按钮(0 上/1 下), -1 = 无
	hoverHead int32 // 悬停的标题热区(块*2 + 0 年/1 月), -1 = 无
	hoverDrop int32 // 悬停的下拉项索引, -1 = 无
	hoverFoot int32 // 悬停的页脚项索引, -1 = 无
}

// 选择模式。
const (
	calModeSingle = iota
	calModeRange
)

// 年月下拉面板类型。
const (
	calDropNone = iota
	calDropYear
	calDropMonth
)

// 月历几何(逻辑像素, 设计基准)。cell 取 32 = shadcn 默认 --cell-size: 2rem。
//
// 真正落地的是 calGeom: 以宽度为缩放比把这里的每个值等比放大, 所以把元素拉大时
// 格子/内边距/字号一起变大, 不会只剩空白; 默认尺寸则是基准的 calDefaultScale 倍
// (实测 1.12 倍的格子 36 / 字号 16 比例更舒展, 已定为默认)。
const (
	calCell     int32 = 32
	calPad      int32 = 12 // 卡片内边距 p-3
	calTitleH   int32 = 32
	calWeekH    int32 = 26
	calRowGap   int32 = 2 // 行间距 mt-2(元素偏高时按剩余空间均摊)
	calGapTitle int32 = 4 // 标题行 → 星期行
	calGapWeek  int32 = 4 // 星期行 → 日期区
	calRows     int   = 6

	calMonthGap  int32 = 16 // 多个月之间的 gap-4
	calMaxMonths int   = 4

	calFootH      int32 = 37 // 页脚: 1px 分隔线 + 上 6 + 按钮 24 + 下 6
	calFootGap    int32 = 8  // 页脚项间距
	calFootBtnH   int32 = 24 // 页脚按钮高(ghost size-sm)
	calFootPadTop int32 = 6
	calFootPadBot int32 = 6
	calFootRowGap int32 = 4
	calFootPadX   int32 = 8  // 页脚按钮左右内边距(px-2)
	calDropPad    int32 = 4  // 下拉面板内边距 p-1
	calDropGap    int32 = 2  // 标题 → 面板
	calDropRowMin int32 = 16 // 行高下限(再小就滚动)
	calDropRowMax int32 = 26 // 行高上限
	calDropMinW   int32 = 72 // 面板最小宽度
	calMonthInner int32 = 7 * calCell
	calBaseW      int32 = calPad*2 + calMonthInner // 单月基准宽 248
	calBaseH      int32 = calPad*2 + calTitleH + calGapTitle + calWeekH + calGapWeek +
		int32(calRows)*calCell + int32(calRows-1)*calRowGap // 292
	calDefaultScale = 1.12
)

// hover 过渡键: 日期格用 0..(months*42-1), 其余从 1000 起(不要互相撞车,
// 撞了会共用过渡值; 1000 以下留给最多 4 个月的 168 个格子)。
const (
	calHoverPrev int32 = 1000
	calHoverNext int32 = 1001
	calHoverHead int32 = 1010 // 1010 + 块*2 + 0 年/1 月
	calHoverFoot int32 = 1100 // 1100+i: 页脚项
	calHoverDrop int32 = 1200 // 1200+i: 下拉项(年份最多 100+ 项, 不会撞到页脚)
)

// calMetrics 是由宽度、月数、周数列决定的整套固定尺寸(不含高度相关的行距)。
type calMetrics struct {
	s      float64
	pad    int32
	cell   int32
	weekW  int32
	monthW int32

	titleH   int32
	weekH    int32
	rowGap   int32
	gapTitle int32
	gapWeek  int32
	monthGap int32

	footPadTop int32
	footPadX   int32
	footBtnH   int32
	footGap    int32
	footRowGap int32
	footPadBot int32
}

// calGeom 是一次布局算出的整套几何: 宽度决定缩放比(格子/内边距/字号), 高度决定
// 6 行日期的行距(多出来的空间均摊到行间, 于是整块日期区始终铺满到页脚之上)。
// 绘制与命中判定共用同一套值, 保证"看到哪就能点到哪"。
type calGeom struct {
	calMetrics

	foot       calFooterLayout
	footH      int32
	gridTop    int32
	monthsLeft int32
	monthsW    int32
	rowGap     int32
}

// calFooterLayout 是页脚换行后的布局(行号分组 + 每项宽度 + 总高)。
type calFooterLayout struct {
	widths []int32
	rows   [][]int32
	rowH   int32
	rowGap int32
	padTop int32
	padBot int32
	totalH int32
}

// fontPx 按缩放比换算字号(主题字号是 14px 档, 元素放大后文字跟着放大)。
func (m calMetrics) fontPx(base int32) int32 {
	v := int32(math.Round(float64(base) * m.s))
	if v < 6 {
		v = 6
	}
	return v
}

func calClampMonths(months int) int {
	if months < 1 {
		return 1
	}
	if months > calMaxMonths {
		return calMaxMonths
	}
	return months
}

// calBaseTotalWidth 返回指定月数/周数列的基准总宽(未缩放)。
func calBaseTotalWidth(months int, weekNumbers bool) int32 {
	months = calClampMonths(months)
	inner := calMonthInner
	if weekNumbers {
		inner += calCell
	}
	return calPad*2 + int32(months)*inner + int32(months-1)*calMonthGap
}

// calScaleFor 由元素宽度反推等比缩放比例。
func calScaleFor(width int32, months int, weekNumbers bool) float64 {
	if width <= 0 {
		return calDefaultScale
	}
	s := float64(width) / float64(calBaseTotalWidth(months, weekNumbers))
	if s <= 0 {
		return calDefaultScale
	}
	return s
}

// calMetricsFor 算固定尺寸。实际月份内容宽度可能与设想的宽度差 1~2px(取整), 绘制时居中。
func calMetricsFor(width int32, months int, weekNumbers bool) calMetrics {
	months = calClampMonths(months)
	s := calScaleFor(width, months, weekNumbers)
	px := func(v int32) int32 { return int32(math.Round(float64(v) * s)) }
	pad := px(calPad)
	monthGap := px(calMonthGap)
	cols := int32(7)
	if weekNumbers {
		cols = 8
	}
	// 格子宽按"可用宽度均分"算(与旧版单月口径一致): 先把四舍五入吃掉的零头
	// 留在两侧, 而不是从格子里扣 —— 否则默认 278 宽会从 36 掉到 35, 整卡矮 6px。
	avail := width - 2*pad - int32(months-1)*monthGap
	if avail <= 0 {
		avail = int32(months) * cols * px(calCell)
	}
	cell := avail / (int32(months) * cols)
	if cell <= 0 {
		cell = px(calCell)
	}
	monthW := cols * cell
	weekW := int32(0)
	if weekNumbers {
		weekW = cell
	}
	return calMetrics{
		s:          s,
		pad:        pad,
		cell:       cell,
		weekW:      weekW,
		monthW:     monthW,
		titleH:     px(calTitleH),
		weekH:      px(calWeekH),
		rowGap:     px(calRowGap),
		gapTitle:   px(calGapTitle),
		gapWeek:    px(calGapWeek),
		monthGap:   monthGap,
		footPadTop: px(calFootPadTop),
		footPadX:   px(calFootPadX),
		footBtnH:   px(calFootBtnH),
		footGap:    px(calFootGap),
		footRowGap: px(calFootRowGap),
		footPadBot: px(calFootPadBot),
	}
}

// calBaseHeight 返回不含页脚的自然高度(6 行固定)。
func calBaseHeight(m calMetrics) int32 {
	return 2*m.pad + m.titleH + m.gapTitle + m.weekH + m.gapWeek +
		int32(calRows)*m.cell + int32(calRows-1)*m.rowGap
}

// CalWidth 返回单月、无周数列的默认宽度(逻辑像素, 278)。
func CalWidth() int32 { return CalWidthFor(1) }

// CalWidthFor 返回指定月数的默认宽度(不含周数列)。多个月时额外加月间距。
func CalWidthFor(months int) int32 {
	return int32(math.Round(float64(calBaseTotalWidth(months, false)) * calDefaultScale))
}

// CalWidthForWeekNumbers 返回启用周数列时的默认宽度(每月多占一格)。
// 外部布局要固定宽度时用它, 否则元素被拉伸后格子会跟着放大。
func CalWidthForWeekNumbers(months int) int32 {
	return int32(math.Round(float64(calBaseTotalWidth(months, true)) * calDefaultScale))
}

// CalHeight 返回单月默认高度(逻辑像素, 不含页脚; 327)。
func CalHeight() int32 { return CalHeightFor(CalWidth()) }

// CalHeightFor 返回指定宽度下、单月的自然高度(不含页脚)。
func CalHeightFor(width int32) int32 { return CalHeightForMonths(width, 1) }

// CalHeightForMonths 返回指定宽度与月数下的自然高度(不含页脚、不含周数列)。
// 多个月是并排的, 高度只跟单月宽度有关, 所以宽度要传整块宽度、月数传并排数量。
func CalHeightForMonths(width int32, months int) int32 {
	if width <= 0 {
		width = CalWidthFor(months)
	}
	return calBaseHeight(calMetricsFor(width, months, false))
}

// CalFooterHeightFor 返回指定宽度下页脚的高度(单行基准)。页脚按钮较多换行时,
// 实际高度以元素自撑(cy 传 0)为准。
func CalFooterHeightFor(width int32) int32 { return CalFooterHeightForMonths(width, 1) }

// CalFooterHeightForMonths 返回指定宽度与月数下的页脚高度(单行基准, 用于显式指定
// 元素高度时预留; 页脚会换行时请把 cy 传 0 让组件按内容自撑)。
func CalFooterHeightForMonths(width int32, months int) int32 {
	if width <= 0 {
		width = CalWidthFor(months)
	}
	m := calMetricsFor(width, months, false)
	return int32(math.Round(float64(calFootH) * m.s))
}

// CalFooterHeight 返回默认宽度下启用页脚时卡片额外增加的高度(逻辑像素, 41)。
func CalFooterHeight() int32 {
	return int32(math.Round(float64(calFootH) * calDefaultScale))
}

// CalendarFooterItem 页脚项: Text 是显示文本; OnClick 非空时是可点按钮(悬停有底),
// 为空时是纯文本; Disabled 只对按钮生效。
type CalendarFooterItem struct {
	Text     string
	OnClick  func()
	Disabled bool
}

// CalendarPreset 是页脚预设按钮: Days/To 是相对今天的天数(可为负, 表示过去)。
// 单选模式只用 Days; 范围模式 from = 今天+Days, to = 今天+To。
type CalendarPreset struct {
	Label string
	Days  int
	To    int
}

// CalPreset 是 CalendarPreset 的简写构造。
func CalPreset(label string, days, to int) CalendarPreset {
	return CalendarPreset{Label: label, Days: days, To: to}
}

// CalendarOption 月历配置项。
type CalendarOption func(*Calendar)

// WithCalendarSelected 设置初始选中日期。
func WithCalendarSelected(t time.Time) CalendarOption {
	return func(c *Calendar) { c.sel = t }
}

// WithCalendarRange 启用范围选择模式(mode="range")。范围未选完时只高亮起点,
// 第二次点击补齐区间; 再点一次重新开始。
func WithCalendarRange() CalendarOption {
	return func(c *Calendar) { c.mode = calModeRange }
}

// WithCalendarSelectedRange 启用范围模式并设置初始区间(只传 from 时区间未完成)。
// to 早于 from 时自动交换。
func WithCalendarSelectedRange(from, to time.Time) CalendarOption {
	return func(c *Calendar) {
		c.mode = calModeRange
		c.rangeFrom, c.rangeTo = calNormalizeRange(from, to)
	}
}

// WithCalendarMonths 设置并排显示的月数(numberOfMonths, 1~4)。
func WithCalendarMonths(n int) CalendarOption {
	return func(c *Calendar) { c.months = calClampMonths(n) }
}

// WithCalendarWeekNumbers 在每月左侧显示 ISO 周数(showWeekNumber)。
func WithCalendarWeekNumbers() CalendarOption {
	return func(c *Calendar) { c.weekNums = true }
}

// WithCalendarWeekStart 设置一周起始日(默认 time.Monday)。
func WithCalendarWeekStart(w time.Weekday) CalendarOption {
	return func(c *Calendar) { c.weekStart = w }
}

// WithCalendarOnSelect 注册单选日期选中回调(范围模式请用 WithCalendarOnRangeSelect)。
func WithCalendarOnSelect(fn func(time.Time)) CalendarOption {
	return func(c *Calendar) { c.onSelect = fn }
}

// WithCalendarOnRangeSelect 注册范围回调。第一次点击(from 落定、to 为零)也会回调,
// 第二次点击补上 to; 预设按钮一次给出完整区间。只会在范围模式下触发。
func WithCalendarOnRangeSelect(fn func(from, to time.Time)) CalendarOption {
	return func(c *Calendar) { c.onRange = fn }
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

// WithCalendarStrike 设置划线日期判定: 返回 true 的日期文字上画一条删除线(预订/占用).
// 与禁用是两回事 —— 只划线不影响可点性, 需要同时禁用请再传 WithCalendarDisabled。
func WithCalendarStrike(fn func(time.Time) bool) CalendarOption {
	return func(c *Calendar) { c.strike = fn }
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
// 页脚自带 1px 分隔线, 项在卡片底部水平居中(一行放不下自动换行); 元素未指定
// 高度时会自动长高。
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

// WithCalendarPresets 启用页脚并追加一组预设日期按钮(shadcn calendar 的 Presets 示例):
// 单选模式跳到今天+N 天并选中; 范围模式一次选中 [今天+Days, 今天+To] 区间。
func WithCalendarPresets(presets ...CalendarPreset) CalendarOption {
	return func(c *Calendar) {
		for _, p := range presets {
			p := p // go 1.18: 循环变量共享, 必须拷一份
			c.footer = append(c.footer, CalendarFooterItem{
				Text:    p.Label,
				OnClick: func() { c.applyPreset(p) },
			})
		}
	}
}

// CalendarDisableBefore 返回"禁用 t 之前(按天比较)"的判定函数 —— t 当天可用,
// 之前的日期全部灰显。
func CalendarDisableBefore(t time.Time) func(time.Time) bool {
	limit := calDay(t)
	return func(d time.Time) bool { return calDay(d).Before(limit) }
}

// CalendarDisableAfter 返回"禁用 t 之后(按天比较)"的判定函数 —— t 当天可用,
// 之后的日期全部灰显(日期范围选择里限制"不能选未来"常用)。
func CalendarDisableAfter(t time.Time) func(time.Time) bool {
	limit := calDay(t)
	return func(d time.Time) bool { return calDay(d).After(limit) }
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

// CalendarStrikeDates 返回"给 dates 里的日期画删除线"的判定函数(预订日期示例)。
func CalendarStrikeDates(dates ...time.Time) func(time.Time) bool {
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

func calSameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// calNormalizeRange 处理"第二次点击早于第一次"的情况(react-day-picker 会交换两端)。
func calNormalizeRange(from, to time.Time) (time.Time, time.Time) {
	if !from.IsZero() && !to.IsZero() && calDay(to).Before(calDay(from)) {
		return to, from
	}
	return from, to
}

// calRangeAfterClick 是范围选择的单击状态机, 对齐 react-day-picker v9 的 addToRange:
//   - 没有起点, 或上一段区间已经完整 → 这次点击作为新起点;
//   - 只有起点 → 这次点击补终点(早于起点则交换)。
func calRangeAfterClick(from, to, d time.Time) (time.Time, time.Time) {
	if from.IsZero() || !to.IsZero() {
		return calDay(d), time.Time{}
	}
	return calNormalizeRange(from, d)
}

// calWeekNumber 返回一行日期对应的 ISO 周数。ISO 周以周一为起点, 所以取该行里的
// 周一来算: 周一开头的行就是行首; 周日开头的行要 +1 天, 否则会把周日算进上一周。
func calWeekNumber(rowStart time.Time, weekStart time.Weekday) int {
	offset := (8 - int(weekStart)) % 7
	_, week := rowStart.AddDate(0, 0, offset).ISOWeek()
	return week
}

// NewCalendar 创建月历卡片。cx/cy 传 0 使用默认尺寸(单月 278×327, 多个月宽度按
// CalWidthFor(months) 加宽, 启用页脚时高度再加页脚高); 只想改宽度时 cy 也传 0 ——
// 高度按宽度自动配平, 内部格子/字号/页脚都会等比跟着宽度放大。
func NewCalendar(x, y, cx, cy int32, hParent int, opts ...CalendarOption) *Calendar {
	now := time.Now()
	c := &Calendar{
		view:      time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local),
		weekStart: time.Monday,
		months:    1,
		hoverIdx:  -1,
		hoverBtn:  -1,
		hoverHead: -1,
		hoverDrop: -1,
		hoverFoot: -1,
	}
	for _, o := range opts {
		o(c)
	}
	c.months = calClampMonths(c.months)
	c.rangeFrom, c.rangeTo = calNormalizeRange(c.rangeFrom, c.rangeTo)
	switch {
	case !c.rangeFrom.IsZero():
		c.sel = c.rangeFrom
		c.view = time.Date(c.rangeFrom.Year(), c.rangeFrom.Month(), 1, 0, 0, 0, 0, time.Local)
	case !c.sel.IsZero():
		c.view = time.Date(c.sel.Year(), c.sel.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	if cx <= 0 {
		cx = calDefaultWidth(c.months, c.weekNums)
	}
	c.autoH = cy <= 0
	if cy <= 0 {
		cy = c.naturalHeightFor(cx)
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
	if c.autoH {
		// 宽度由布局拉伸时, 高度按新宽度重新配平(格子等比缩放 + 页脚可能换行)。
		// 没有这一条, Fill 宽的元素会保持构造时的固定高度, 宽高比走样。
		// 尺寸事件里不能直接改高度并重排(会重入布局), 推迟到下一轮消息。
		e.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
			c.scheduleSyncHeight()
			return 0
		})
	}
	xui.AttachWindow(e.Handle)
	return c
}

// calDefaultWidth 按配置返回默认宽度(周数列会多占一格)。
func calDefaultWidth(months int, weekNumbers bool) int32 {
	return int32(math.Round(float64(calBaseTotalWidth(months, weekNumbers)) * calDefaultScale))
}

// naturalHeightFor 返回当前配置在指定宽度下的自然高度(含页脚, 页脚多行时会加高)。
func (c *Calendar) naturalHeightFor(width int32) int32 {
	m := calMetricsFor(width, c.months, c.weekNums)
	h := calBaseHeight(m)
	if len(c.footer) > 0 {
		h += calLayoutFooter(width, c.months, c.weekNums, c.footer).totalH
	}
	return h
}

// calLayoutFooter 把页脚项按可用宽度换行, 返回每项宽度、行分组和总高。
func calLayoutFooter(width int32, months int, weekNumbers bool, items []CalendarFooterItem) calFooterLayout {
	if len(items) == 0 {
		return calFooterLayout{}
	}
	m := calMetricsFor(width, months, weekNumbers)
	lay := calFooterLayout{
		widths: make([]int32, len(items)),
		rowH:   m.footBtnH,
		rowGap: m.footRowGap,
		padTop: m.footPadTop,
		padBot: m.footPadBot,
	}
	t := xui.Current()
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: m.fontPx(t.FontDefault.Size)})
	for i, it := range items {
		w := int32(0)
		if f != 0 {
			w = xui.TextWidth(it.Text, f)
		} else {
			w = int32(len([]rune(it.Text))) * m.fontPx(t.FontDefault.Size)
		}
		if it.OnClick != nil {
			w += m.footPadX * 2
		}
		lay.widths[i] = w
	}
	avail := width - 2*m.pad
	var row []int32
	rowW := int32(0)
	for i, w := range lay.widths {
		add := w
		if len(row) > 0 {
			add += m.footGap
		}
		if len(row) > 0 && rowW+add > avail {
			lay.rows = append(lay.rows, row)
			row = nil
			rowW = 0
			add = w
		}
		row = append(row, int32(i))
		rowW += add
	}
	if len(row) > 0 {
		lay.rows = append(lay.rows, row)
	}
	lay.totalH = 1 + lay.padTop + int32(len(lay.rows))*lay.rowH +
		int32(len(lay.rows)-1)*lay.rowGap + lay.padBot
	return lay
}

// geom 按元素当前尺寸算几何(元素还没创建时退回默认尺寸)。
func (c *Calendar) geom() calGeom {
	months := calClampMonths(c.months)
	m := calMetricsFor(calDefaultWidth(months, c.weekNums), months, c.weekNums)
	w := calDefaultWidth(months, c.weekNums)
	h := c.naturalHeightFor(w)
	if c.Element != nil {
		if v := c.GetWidth(); v > 0 {
			w = v
			m = calMetricsFor(w, months, c.weekNums)
		}
		if v := c.GetHeight(); v > 0 {
			h = v
		} else {
			h = c.naturalHeightFor(w)
		}
	}
	lay := calLayoutFooter(w, months, c.weekNums, c.footer)
	g := calGeom{
		calMetrics: m,
		foot:       lay,
		footH:      lay.totalH,
	}
	g.monthsW = int32(months)*m.monthW + int32(months-1)*m.monthGap
	g.monthsLeft = m.pad + (w-2*m.pad-g.monthsW)/2
	if g.monthsLeft < m.pad {
		g.monthsLeft = m.pad
	}
	g.gridTop = m.pad + m.titleH + m.gapTitle + m.weekH + m.gapWeek
	gridBottom := h - m.pad - g.footH
	leftover := gridBottom - g.gridTop - int32(calRows)*m.cell
	if leftover >= 0 {
		g.rowGap = leftover / int32(calRows-1)
	} else {
		g.rowGap = m.rowGap
	}
	return g
}

// fontPx 保留旧的调用方式(calGeom 内嵌 calMetrics, 已有同名方法)。
func (g calGeom) px(v int32) int32 { return int32(math.Round(float64(v) * g.s)) }

// Selected 返回当前选中日期(范围模式返回起点), 未选择时为零值。
func (c *Calendar) Selected() time.Time { return c.sel }

// SelectedRange 返回当前范围(单选模式时 from 为选中日期, to 为零值)。
func (c *Calendar) SelectedRange() (from, to time.Time) { return c.rangeFrom, c.rangeTo }

// RangeMode 报告是否处于范围选择模式。
func (c *Calendar) RangeMode() bool { return c.mode == calModeRange }

// SetSelected 设置选中日期, 同时把显示月份切到该日期所在月。传零值 = 清除选择。
// 范围模式下会把起点设为 t 并清空终点。
func (c *Calendar) SetSelected(t time.Time) *Calendar {
	c.sel = t
	if c.mode == calModeRange {
		c.rangeFrom, c.rangeTo = t, time.Time{}
	}
	if !t.IsZero() {
		c.view = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	c.Redraw(false)
	return c
}

// SetRange 设置范围(传零值 = 清除)。to 早于 from 时自动交换; 只传 from 表示
// "范围尚未选完"。显示月份对齐**起点**所在月 —— 与上游 defaultMonth={from} 一致;
// 区间比可见月数长时终点在窗口外, 往左翻页可看(上游同款行为)。
func (c *Calendar) SetRange(from, to time.Time) *Calendar {
	c.mode = calModeRange
	c.rangeFrom, c.rangeTo = calNormalizeRange(from, to)
	c.sel = c.rangeFrom
	if !c.rangeFrom.IsZero() {
		c.view = time.Date(c.rangeFrom.Year(), c.rangeFrom.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	c.Redraw(false)
	return c
}

// SetMonth 切换第一个可见月份(不改选中日期)。
func (c *Calendar) SetMonth(year int, month time.Month) *Calendar {
	c.view = time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	c.Redraw(false)
	return c
}

// NextMonth 显示下一个月(多个月时整块一起前移一个月)。
func (c *Calendar) NextMonth() *Calendar {
	c.view = c.view.AddDate(0, 1, 0)
	c.Redraw(false)
	return c
}

// PrevMonth 显示上一个月(多个月时整块一起后移一个月)。
func (c *Calendar) PrevMonth() *Calendar {
	c.view = c.view.AddDate(0, -1, 0)
	c.Redraw(false)
	return c
}

// Today 切到今天所在月份; 今天未被禁用时同时选中并触发回调
// (范围模式下区间 = 今天~今天)。
func (c *Calendar) Today() *Calendar {
	now := time.Now()
	c.view = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	today := calDay(now)
	if !c.dateDisabled(today) {
		c.sel = today
		if c.mode == calModeRange {
			c.rangeFrom, c.rangeTo = today, today
			c.Redraw(false)
			if c.onRange != nil {
				c.onRange(today, today)
			}
			return c
		}
		c.Redraw(false)
		if c.onSelect != nil {
			c.onSelect(today)
		}
		return c
	}
	c.Redraw(false)
	return c
}

// goToday 是内置"今天"页脚按钮的动作。
func (c *Calendar) goToday() { c.Today() }

// applyPreset 是预设按钮的动作: 相对今天计算并选中(范围模式一次落两端)。
func (c *Calendar) applyPreset(p CalendarPreset) {
	base := calDay(time.Now())
	from := base.AddDate(0, 0, p.Days)
	if c.mode == calModeRange {
		to := base.AddDate(0, 0, p.To)
		if p.To == 0 && p.Days != 0 {
			to = from // 单日预设(Days 非 0、To 省略)按"起止同一天"处理
		}
		c.SetRange(from, to)
		c.sel = c.rangeFrom
		if c.onRange != nil {
			c.onRange(c.rangeFrom, c.rangeTo)
		}
		return
	}
	c.SetSelected(from)
	if c.onSelect != nil {
		c.onSelect(from)
	}
}

// ViewMonth 返回第一个可见月份的年月。
func (c *Calendar) ViewMonth() (year int, month time.Month) {
	return c.view.Year(), c.view.Month()
}

// OnSelect 注册日期选中回调(选中的是本月还是 outside 日期都会回调;
// 禁用日期不会触发)。
func (c *Calendar) OnSelect(fn func(time.Time)) *Calendar {
	c.onSelect = fn
	return c
}

// OnRangeSelect 注册范围回调。
func (c *Calendar) OnRangeSelect(fn func(from, to time.Time)) *Calendar {
	c.onRange = fn
	return c
}

// SetDisabled 运行时替换禁用日期判定(传 nil = 全部可选)。
func (c *Calendar) SetDisabled(fn func(time.Time) bool) *Calendar {
	c.disabled = fn
	c.Redraw(false)
	return c
}

// SetStrike 运行时替换划线日期判定(传 nil = 不划线)。
func (c *Calendar) SetStrike(fn func(time.Time) bool) *Calendar {
	c.strike = fn
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
	g := c.geom()
	top := c.GetHeight() - g.footH + 1 + g.footPadTop
	return xc.RECT{
		Left:   g.pad,
		Top:    top,
		Right:  c.GetWidth() - g.pad,
		Bottom: c.GetHeight() - g.footPadBot,
	}
}

// syncHeight 在页脚项增删后按需调整元素高度(仅构造时未指定高度的实例;
// 指定了高度的实例由调用方自己预留 CalFooterHeightFor、或把 cy 传 0)。
func (c *Calendar) syncHeight() {
	if !c.autoH || c.Element == nil {
		return
	}
	w := c.GetWidth()
	if w <= 0 {
		return
	}
	h := c.naturalHeightFor(w)
	if h == c.GetHeight() {
		return
	}
	c.SetHeight(h)
	// 高度变化后要逐级通知祖先重排: 只改自己时, Auto 高的卡片/行仍按旧高度排版,
	// ScrollArea 也不会重算滚动范围(同 Alert, 见 DESIGN 第八节 90)。
	c.adjustAncestors()
}

// scheduleSyncHeight 把尺寸事件里的高度重算推迟到下一轮 UI 消息(直接重入布局会乱)。
func (c *Calendar) scheduleSyncHeight() {
	if c == nil || c.syncPending {
		return
	}
	c.syncPending = true
	app.AfterFunc(0, func() {
		c.syncPending = false
		c.syncHeight()
	})
}

// adjustAncestors 从自己开始逐级对祖先调用 AdjustLayout。
func (c *Calendar) adjustAncestors() {
	if c.Element == nil {
		return
	}
	xc.XEle_AdjustLayout(c.Handle)
	for p := xc.XWidget_GetParentEle(c.Handle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			// ScrollArea 的框架自己会重排内容区并重算滚动范围。
			xc.XEle_AdjustLayout(p)
			break
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			// p 是框架的内容区: 直接对内容区 AdjustLayout 会把子项宽度打成 0(实测),
			// 必须交给框架排。
			xc.XEle_AdjustLayout(pp)
			break
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_ELE) {
			xc.XEle_AdjustLayout(p)
			continue
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
		}
		break
	}
}

// dateDisabled 报告某个日期是否被禁用(未设置判定函数时恒为 false)。
func (c *Calendar) dateDisabled(d time.Time) bool {
	return c.disabled != nil && c.disabled(d)
}

// monthAt 返回第 i 个可见月份(取 1 号)。
func (c *Calendar) monthAt(i int) time.Time {
	return c.view.AddDate(0, i, 0)
}

// dateAtMonth 返回第 monthIdx 个月、第 idx 个格子(0..41)对应的日期。
//
// 网格起点 = 当月 1 号回退到"所在周的第一天"(weekStart 决定), 之后顺序排 42 天。
func (c *Calendar) dateAtMonth(monthIdx int, idx int32) time.Time {
	first := c.monthAt(monthIdx)
	offset := (int(first.Weekday()) - int(c.weekStart) + 7) % 7
	start := first.AddDate(0, 0, -offset)
	return start.AddDate(0, 0, int(idx))
}

// dateAt 返回第一个月第 idx 个格子对应的日期(保留旧接口)。
func (c *Calendar) dateAt(idx int32) time.Time { return c.dateAtMonth(0, idx) }

// indexAt 把元素客户区坐标换算成格子索引(块*42 + 0..41), 不在格子上时返回 -1。
func (c *Calendar) indexAt(x, y int32) int32 {
	g := c.geom()
	if y < g.gridTop {
		return -1
	}
	row := (y - g.gridTop) / (g.cell + g.rowGap)
	if (y-g.gridTop)-row*(g.cell+g.rowGap) >= g.cell { // 落在行间隙
		return -1
	}
	if row < 0 || row >= int32(calRows) {
		return -1
	}
	for i := 0; i < c.months; i++ {
		mx := g.monthsLeft + int32(i)*(g.monthW+g.monthGap)
		if x < mx || x >= mx+g.monthW {
			continue
		}
		if c.weekNums && x < mx+g.weekW {
			return -1
		}
		col := (x - mx - g.weekW) / g.cell
		if col < 0 || col >= 7 {
			return -1
		}
		return int32(i)*int32(calRows*7) + row*7 + col
	}
	return -1
}

// hitButton 判断坐标是否落在翻页按钮上: 0 = 上一月, 1 = 下一月, -1 = 都不是。
func (c *Calendar) hitButton(x, y int32, width int32) int32 {
	g := c.geom()
	if y < g.pad || y > g.pad+g.titleH {
		return -1
	}
	if x >= g.pad && x < g.pad+g.cell {
		return 0
	}
	if x >= width-g.pad-g.cell && x < width-g.pad {
		return 1
	}
	return -1
}

// headerHit 判断坐标是否落在某个月的年 / 月热区上:
// 返回值 = 块*2 + 0(年) / 1(月); -1 = 都不是。
func (c *Calendar) headerHit(x, y int32) int32 {
	for i, rc := range c.hdrYear {
		if calInRect(rc, x, y) {
			return int32(i * 2)
		}
	}
	for i, rc := range c.hdrMonth {
		if calInRect(rc, x, y) {
			return int32(i*2 + 1)
		}
	}
	return -1
}

// calHeadKind 把 headerHit 的编码(块*2 + 0 年/1 月)换算成 calDrop* 面板类型。
// ⚠️ 不能直接把编码当 calDrop* 用: calDropNone=0 恰好是"年"的编码, 直接传给
// toggleDrop 会变成"关面板"(实测: 点年份无效、点月份弹出年份列表)。
func calHeadKind(head int32) int {
	if head%2 == 0 {
		return calDropYear
	}
	return calDropMonth
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
		// 点标题热区: 同一个 = 收起, 另一个 = 直接切过去(与原生 select 一致)
		if head := c.headerHit(x, y); head >= 0 {
			block, kind := int(head/2), calHeadKind(head)
			if block == c.dropMonth && kind == c.dropKind {
				c.closeDrop()
			} else {
				c.toggleDrop(kind, block)
			}
			return 0
		}
		c.closeDrop()
		return 0
	}

	if head := c.headerHit(x, y); head >= 0 {
		c.toggleDrop(calHeadKind(head), int(head/2))
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
	block := int(idx / int32(calRows*7))
	cell := idx % int32(calRows*7)
	d := c.dateAtMonth(block, cell)
	if c.dateDisabled(d) {
		return 0
	}

	if c.mode == calModeRange {
		c.rangeFrom, c.rangeTo = calRangeAfterClick(c.rangeFrom, c.rangeTo, d)
		c.sel = c.rangeFrom
		// 点 outside 日期时单月跟随翻页(多个月并排时不跳, 否则整块窗口会突然位移)
		if c.months == 1 && (d.Month() != c.view.Month() || d.Year() != c.view.Year()) {
			c.view = time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.Local)
		}
		c.Redraw(false)
		if c.onRange != nil {
			c.onRange(c.rangeFrom, c.rangeTo)
		}
		return 0
	}

	c.sel = d
	// 点到 outside 日期: 跟随翻到目标月(shadcn 的行为; 多个月并排时不跳)
	if c.months == 1 && (d.Month() != c.view.Month() || d.Year() != c.view.Year()) {
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
	if idx >= 0 {
		block := int(idx / int32(calRows*7))
		if c.dateDisabled(c.dateAtMonth(block, idx%int32(calRows*7))) {
			idx = -1 // 禁用格不高亮
		}
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

// toggleDrop 打开 / 收起某一类下拉(再点同一个热区就是收起)。
func (c *Calendar) toggleDrop(kind, monthIdx int) {
	if monthIdx < 0 || monthIdx >= c.months {
		monthIdx = 0
	}
	if c.dropKind == kind && c.dropMonth == monthIdx {
		c.closeDrop()
		return
	}
	c.dropKind = kind
	c.dropMonth = monthIdx
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
	c.dropMonth = 0
	c.dropVals = nil
	c.dropScroll = 0
	c.hoverDrop = -1
	c.EnableEvent_XE_MOUSEWHEEL(false)
	c.Redraw(false)
}

// dropBase 返回下拉所属月份块当前显示的年月。
func (c *Calendar) dropBase() time.Time {
	if c.dropMonth < 0 || c.dropMonth >= c.months {
		return c.view
	}
	return c.monthAt(c.dropMonth)
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
	base := c.dropBase()
	from, to := c.yearFrom, c.yearTo
	if c.yearFrom == 0 && c.yearTo == 0 {
		from, to = base.Year()-50, base.Year()+50
	}
	c.dropVals = c.dropVals[:0]
	for y := from; y <= to; y++ {
		c.dropVals = append(c.dropVals, y)
	}
}

// applyDrop 把下拉的选中值落到"被点的那一块"的显示月份上(不动选中日期)。
// 多个月时通过回推 view 让目标块正好显示选中的年月。
func (c *Calendar) applyDrop(v int) {
	base := c.dropBase()
	var target time.Time
	if c.dropKind == calDropYear {
		target = time.Date(v, base.Month(), 1, 0, 0, 0, 0, time.Local)
	} else {
		target = time.Date(base.Year(), time.Month(v), 1, 0, 0, 0, 0, time.Local)
	}
	c.view = target.AddDate(0, -c.dropMonth, 0)
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
	base := c.dropBase()
	if c.dropKind == calDropYear {
		return v == base.Year()
	}
	return v == int(base.Month())
}

// layoutDrop 计算下拉面板几何。面板贴着被点的标题热区下方, 左右夹在卡片内;
// 行高按"12 行正好塞满可用高度"自适应, 放不下就滚动。所有尺寸随元素缩放。
func (c *Calendar) layoutDrop() bool {
	if c.dropKind == calDropNone || c.Element == nil {
		return false
	}
	g := c.geom()
	px := func(v int32) int32 { return int32(math.Round(float64(v) * g.s)) }
	pad := px(calDropPad)
	rowMin, rowMax := px(calDropRowMin), px(calDropRowMax)
	minW := px(calDropMinW)

	w, h := c.GetWidth(), c.GetHeight()
	anchor := xc.RECT{}
	if c.dropMonth >= 0 && c.dropMonth < len(c.hdrMonth) {
		if c.dropKind == calDropYear {
			anchor = c.hdrYear[c.dropMonth]
		} else {
			anchor = c.hdrMonth[c.dropMonth]
		}
	}
	if anchor.Bottom == 0 { // 还没绘制过: 退化成标题行下沿
		blockX := g.monthsLeft + int32(c.dropMonth)*(g.monthW+g.monthGap)
		anchor = xc.RECT{Left: blockX, Top: g.pad + g.titleH, Right: blockX + g.monthW, Bottom: g.pad + g.titleH}
	}

	t := xui.Current()
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: g.fontPx(t.FontDefault.Size)})
	maxW := int32(0)
	for _, v := range c.dropVals {
		if tw := xui.TextWidth(c.dropText(v), f); tw > maxW {
			maxW = tw
		}
	}
	pw := maxW + px(6) + px(22) + px(12) // 左 6 + 右 22(勾选图标) + 左右内边距
	if pw < minW {
		pw = minW
	}
	left := anchor.Left
	if left+pw > w-px(8) {
		left = w - px(8) - pw
	}
	if left < px(8) {
		left = px(8)
	}
	top := anchor.Bottom + px(calDropGap)
	avail := h - px(4) - top
	if avail < rowMin+pad*2+px(4) { // 下方实在太窄: 贴顶铺开
		top = px(4)
		avail = h - px(8)
	}

	rowH := (avail - pad*2) / 12
	if rowH > rowMax {
		rowH = rowMax
	}
	if rowH < rowMin {
		rowH = rowMin
	}
	contentH := int32(len(c.dropVals))*rowH + pad*2
	viewH := contentH
	if viewH > avail {
		viewH = avail
	}
	if viewH < rowH+pad*2 {
		viewH = rowH + pad*2
	}
	c.dropRect = xc.RECT{Left: left, Top: top, Right: left + pw, Bottom: top + viewH}
	c.dropRowH, c.dropViewH, c.dropContentH = rowH, viewH, contentH
	c.dropPad = pad
	c.clampDropScroll()
	return true
}

// dropRowsVisible 返回可视区域里完整行数。
func (c *Calendar) dropRowsVisible() int32 {
	if c.dropRowH <= 0 {
		return 0
	}
	return (c.dropViewH - c.dropPad*2) / c.dropRowH
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
	base := c.dropBase()
	cur := int(base.Month())
	if c.dropKind == calDropYear {
		cur = base.Year()
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
	top := c.dropRect.Top + c.dropPad
	if y < top || y >= c.dropRect.Bottom-c.dropPad {
		return -1
	}
	idx := (y - top + c.dropScroll) / c.dropRowH
	if idx < 0 || idx >= int32(len(c.dropVals)) {
		return -1
	}
	// 只认完整可见的行(面板上下可能露半行, 半行不响应)
	y0 := top + idx*c.dropRowH - c.dropScroll
	if y0 < top || y0+c.dropRowH > c.dropRect.Bottom-c.dropPad {
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

	g := c.geom()
	titleTop := p.Rc.Top + g.pad

	// 翻页按钮位于整块日历的左右两端(shadcn 的 nav 横跨所有月份)。
	prev := xc.RECT{Left: p.Rc.Left + g.pad, Top: titleTop, Right: p.Rc.Left + g.pad + g.cell, Bottom: titleTop + g.titleH}
	next := xc.RECT{Left: p.Rc.Right - g.pad - g.cell, Top: titleTop, Right: p.Rc.Right - g.pad, Bottom: titleTop + g.titleH}
	c.paintNavButton(p, t, g, prev, calHoverPrev, c.hoverBtn == 0, icons.IconChevronLeft)
	c.paintNavButton(p, t, g, next, calHoverNext, c.hoverBtn == 1, icons.IconChevronRight)

	// 每个月块: 标题(年月下拉)+ 星期行 + 日期网格。
	c.hdrYear = c.hdrYear[:0]
	c.hdrMonth = c.hdrMonth[:0]
	for i := 0; i < c.months; i++ {
		c.paintMonth(p, t, g, i)
	}

	// ---- 页脚 ----
	c.paintFooter(p, t, g)

	// ---- 年月下拉(最后画, 盖住日期与页脚) ----
	c.paintDrop(p, t, g)
	return 0
}

// paintNavButton 画标题行左右两端的换月按钮(ghost size-8, 悬停 bg-accent)。
func (c *Calendar) paintNavButton(p *xui.Painter, t *xui.Theme, g calGeom, rc xc.RECT, key int32, active bool, icon string) {
	if hv := xui.HoverItem(c.Handle, key, active); hv > 0 {
		p.FillRound(&rc, t.Accent.ScaleAlpha(hv), t.RadiusSm())
	}
	iconSize := int32(math.Round(16 * g.s))
	ir := xc.RECT{
		Left:   rc.Left + (rc.Right-rc.Left-iconSize)/2,
		Top:    rc.Top + (rc.Bottom-rc.Top-iconSize)/2,
		Right:  rc.Left + (rc.Right-rc.Left+iconSize)/2,
		Bottom: rc.Top + (rc.Bottom-rc.Top+iconSize)/2,
	}
	p.Icon(icon, &ir, t.Foreground, 2)
}

// paintMonth 画第 i 个月: 标题行(年月热区)、星期行、6×7 日期网格与周数列。
func (c *Calendar) paintMonth(p *xui.Painter, t *xui.Theme, g calGeom, i int) {
	blockX := p.Rc.Left + g.monthsLeft + int32(i)*(g.monthW+g.monthGap)
	month := c.monthAt(i)
	titleTop := p.Rc.Top + g.pad

	// 标题中部预留两个翻页按钮的宽度(与 shadcn month_caption 的 px-(--cell-size) 一致)
	capL := blockX + g.cell
	capR := blockX + g.monthW - g.cell
	c.paintMonthHeader(p, t, g, i, month, capL, capR, titleTop)

	// ---- 星期行 ----
	weekTop := titleTop + g.titleH + g.gapTitle
	fSmall := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: g.fontPx(t.FontDefault.Size - 2)})
	if fSmall != 0 {
		p.Draw.SetFont(fSmall)
	}
	if c.weekNums {
		// 周数列表头留空(shadcn WeekNumber 的 header 也是空白)
	}
	for col := int32(0); col < 7; col++ {
		rc := xc.RECT{
			Left:   blockX + g.weekW + col*g.cell,
			Top:    weekTop,
			Right:  blockX + g.weekW + (col+1)*g.cell,
			Bottom: weekTop + g.weekH,
		}
		name := weekNames[(int(c.weekStart)+int(col))%7]
		p.Text(name, &rc, t.MutedForeground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	}

	// ---- 周数列(每月各一列, 行首日期计算 ISO 周数) ----
	if c.weekNums {
		if fSmall != 0 {
			p.Draw.SetFont(fSmall)
		}
		for row := int32(0); row < int32(calRows); row++ {
			top := g.gridTop + row*(g.cell+g.rowGap)
			wrc := xc.RECT{
				Left:   blockX,
				Top:    top + p.Rc.Top,
				Right:  blockX + g.weekW,
				Bottom: top + g.cell + p.Rc.Top,
			}
			week := calWeekNumber(c.dateAtMonth(i, row*7), c.weekStart)
			p.Text(strconv.Itoa(week), &wrc, t.MutedForeground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		}
	}

	// ---- 日期格 ----
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: g.fontPx(t.FontDefault.Size)})
	if f != 0 {
		p.Draw.SetFont(f)
	}
	now := time.Now()
	for idx := int32(0); idx < int32(calRows*7); idx++ {
		d := c.dateAtMonth(i, idx)
		row, col := idx/7, idx%7
		rc := xc.RECT{
			Left:   blockX + g.weekW + col*g.cell,
			Top:    g.gridTop + row*(g.cell+g.rowGap) + p.Rc.Top,
			Right:  blockX + g.weekW + (col+1)*g.cell,
			Bottom: g.gridTop + row*(g.cell+g.rowGap) + g.cell + p.Rc.Top,
		}
		c.paintDay(p, t, g, month, d, rc, int32(i)*int32(calRows*7)+idx, now)
	}
}

// paintMonthHeader 画一个月的 "2026年 ⌄ 10月 ⌄", 并记下热区供命中判定复用。
func (c *Calendar) paintMonthHeader(p *xui.Painter, t *xui.Theme, g calGeom, block int, month time.Time, left, right, titleTop int32) {
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: g.fontPx(t.FontDefault.Size), Bold: true})
	if f != 0 {
		p.Draw.SetFont(f)
	}
	yearText := fmt.Sprintf("%d年", month.Year())
	monthText := fmt.Sprintf("%d月", int(month.Month()))
	padX := int32(math.Round(24 * g.s)) // 左右各 12(含 chevron)
	gap := int32(math.Round(2 * g.s))
	h := int32(math.Round(24 * g.s))
	yearW := xui.TextWidth(yearText, f) + padX
	monthW := xui.TextWidth(monthText, f) + padX

	x := left + (right-left-yearW-gap-monthW)/2
	y := titleTop + (g.titleH-h)/2
	yearRc := xc.RECT{Left: x, Top: y, Right: x + yearW, Bottom: y + h}
	monthRc := xc.RECT{Left: yearRc.Right + gap, Top: y, Right: yearRc.Right + gap + monthW, Bottom: y + h}
	c.hdrYear = append(c.hdrYear, yearRc)
	c.hdrMonth = append(c.hdrMonth, monthRc)

	c.paintHeadLabel(p, t, g, yearRc, yearText, int32(block*2), c.hoverHead == int32(block*2) || (c.dropKind == calDropYear && c.dropMonth == block))
	c.paintHeadLabel(p, t, g, monthRc, monthText, int32(block*2+1), c.hoverHead == int32(block*2+1) || (c.dropKind == calDropMonth && c.dropMonth == block))
}

// paintHeadLabel 画一个"文字 + chevron-down"的标题按钮(打开时保持 bg-accent)。
func (c *Calendar) paintHeadLabel(p *xui.Painter, t *xui.Theme, g calGeom, rc xc.RECT, text string, headKey int32, active bool) {
	hv := xui.HoverItem(c.Handle, calHoverHead+headKey, active)
	if hv > 0 {
		p.FillRound(&rc, t.Accent.ScaleAlpha(hv), t.RadiusSm())
	}
	fore := xui.Fade(t.Foreground, t.AccentForeground, hv)
	chev := xui.Fade(t.MutedForeground, t.AccentForeground, hv)

	padL := int32(math.Round(6 * g.s))
	chevSize := int32(math.Round(10 * g.s))
	tr := rc
	tr.Left += padL
	tr.Right -= padL + chevSize
	p.Text(text, &tr, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)

	ir := xc.RECT{
		Left:   rc.Right - padL - chevSize,
		Top:    rc.Top + (rc.Bottom-rc.Top-chevSize)/2,
		Right:  rc.Right - padL,
		Bottom: rc.Top + (rc.Bottom-rc.Top+chevSize)/2,
	}
	p.Icon(icons.IconChevronDown, &ir, chev, 2)
}

// paintDay 画一个日期格: 范围色带 / 选中实心 / 今天 / 悬停 / 禁用 / 删除线。
// month 是格子所属的月份块(多个月时 outside 判定必须按各自的块来)。
func (c *Calendar) paintDay(p *xui.Painter, t *xui.Theme, g calGeom, month time.Time, d time.Time, rc xc.RECT, key int32, now time.Time) {
	inMonth := d.Month() == month.Month() && d.Year() == month.Year()
	dis := c.dateDisabled(d)
	struck := c.strike != nil && c.strike(d)
	isToday := calSameDay(d, now)
	// 选中/范围标识只画在"本月"格上: 多月并排时同一天会同时出现在相邻块的
	// outside 区(如 9 月块的拖尾就是 10 月上旬), 按日期判定会让端点/色带在
	// 隔壁块重复出现(实测: 范围端点同时高亮两处 + 隔壁块一份, 共 3 个"选中")。
	isSel := !c.sel.IsZero() && calSameDay(d, c.sel) && c.mode == calModeSingle && inMonth
	isStart := !c.rangeFrom.IsZero() && calSameDay(d, c.rangeFrom) && inMonth
	isEnd := !c.rangeTo.IsZero() && calSameDay(d, c.rangeTo) && inMonth
	inRange := c.mode == calModeRange && inMonth && !c.rangeFrom.IsZero() && !c.rangeTo.IsZero() &&
		!calDay(d).Before(calDay(c.rangeFrom)) && !calDay(d).After(calDay(c.rangeTo))
	endpoint := isSel || ((isStart || isEnd) && c.mode == calModeRange)
	sameEnds := inRange && calSameDay(c.rangeFrom, c.rangeTo)
	band := inRange && !sameEnds

	half := func(c0 xui.Color) xui.Color { return c0.ScaleAlpha(0.5) }

	// 范围中段: 铺满格子的 accent 色带, 两端做单侧圆角(react-day-picker 的
	// range_start / range_middle / range_end); 端点再用 primary 实心盖上去。
	if band {
		bandColor := t.Accent
		if dis {
			bandColor = half(t.Accent)
		}
		corners := xui.Corners{}
		if isStart {
			corners.TopLeft, corners.BottomLeft = t.RadiusSm(), t.RadiusSm()
		}
		if isEnd {
			corners.TopRight, corners.BottomRight = t.RadiusSm(), t.RadiusSm()
		}
		p.FillRoundCorners(&rc, bandColor, corners)
	}

	switch {
	case endpoint && dis:
		p.FillRound(&rc, half(t.Primary), t.RadiusSm())
	case endpoint:
		p.FillRound(&rc, t.Primary, t.RadiusSm())
	case isToday && !band && dis:
		p.FillRound(&rc, half(t.Accent), t.RadiusSm())
	case isToday && !band:
		p.FillRound(&rc, t.Accent, t.RadiusSm())
	case !dis && !band:
		if hv := xui.HoverItem(c.Handle, key, c.hoverIdx == key); hv > 0 {
			p.FillRound(&rc, t.Accent.ScaleAlpha(hv), t.RadiusSm())
		}
	}

	// 预订日期(shadcn booked-dates 示例)= disabled + `opacity-100` 的 line-through:
	// 划线本身就把"不可用"说清楚了, 所以被划线的格子不再叠 50% 变淡, 否则一条
	// 半透明灰线压在半透明灰字上等于什么都没画(实测不可见)。
	fore := t.Foreground
	switch {
	case dis && struck && endpoint:
		fore = t.PrimaryForeground
	case dis && struck:
		fore = t.MutedForeground
	case dis && endpoint:
		fore = half(t.PrimaryForeground)
	case dis:
		fore = half(t.MutedForeground)
	case endpoint:
		fore = t.PrimaryForeground
	case inRange:
		fore = t.AccentForeground
	case isToday:
		fore = t.AccentForeground
	case !inMonth:
		fore = t.MutedForeground
	}
	p.Text(strconv.Itoa(d.Day()), &rc, fore, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)

	// 预订日期: 文字上画一条删除线(react-day-picker 的 modifiersClassNames line-through)。
	if struck {
		padding := g.cell / 4
		if padding < 2 {
			padding = 2
		}
		midY := rc.Top + (rc.Bottom-rc.Top)/2
		p.Line(rc.Left+padding, midY, rc.Right-padding, midY, fore, 1)
	}
}

// paintFooter 画页脚: 1px 分隔线(卡片全宽) + 水平居中、可换行的一组项。
func (c *Calendar) paintFooter(p *xui.Painter, t *xui.Theme, g calGeom) {
	if len(c.footer) == 0 {
		return
	}
	footerTop := p.Rc.Bottom - g.footH
	p.Line(p.Rc.Left, footerTop, p.Rc.Right, footerTop, t.Border, 1)

	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: g.fontPx(t.FontDefault.Size)})
	if f != 0 {
		p.Draw.SetFont(f)
	}
	contentTop := footerTop + 1 + g.foot.padTop

	c.footRects = c.footRects[:0]
	for rowIdx, row := range g.foot.rows {
		total := int32(0)
		for k, idx := range row {
			if k > 0 {
				total += g.footGap
			}
			total += g.foot.widths[idx]
		}
		x := p.Rc.Left + (p.W()-total)/2
		y := contentTop + int32(rowIdx)*(g.foot.rowH+g.foot.rowGap)
		for _, idx := range row {
			it := c.footer[idx]
			rc := xc.RECT{Left: x, Top: y, Right: x + g.foot.widths[idx], Bottom: y + g.foot.rowH}
			c.footRects = append(c.footRects, rc)
			if it.OnClick != nil {
				hv := xui.HoverItem(c.Handle, calHoverFoot+idx, c.hoverFoot == idx && !it.Disabled)
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
			x = rc.Right + g.footGap
		}
	}
}

// paintDrop 画年月下拉面板: bg-popover rounded-md + ring, 悬停 bg-accent,
// 当前值右侧打勾; 内容超出可视高度时右侧画 2px 滚动条(滚轮滚动)。
func (c *Calendar) paintDrop(p *xui.Painter, t *xui.Theme, g calGeom) {
	if !c.layoutDrop() {
		return
	}
	rc := c.dropRect
	p.FillRound(&rc, t.Popover, t.RadiusMd())
	p.StrokeRound(&rc, t.Foreground.WithAlpha(26), t.RadiusMd(), 1)

	viewTop := rc.Top + c.dropPad
	viewBottom := rc.Bottom - c.dropPad
	scrollable := c.dropContentH > c.dropViewH
	s := g.s
	if s <= 0 {
		s = calDefaultScale
	}
	padX := func(v int32) int32 { return int32(math.Round(float64(v) * s)) }

	if f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: g.fontPx(t.FontDefault.Size)}); f != 0 {
		p.Draw.SetFont(f)
	}
	for i, v := range c.dropVals {
		y0 := viewTop + int32(i)*c.dropRowH - c.dropScroll
		y1 := y0 + c.dropRowH
		if y0 < viewTop || y1 > viewBottom {
			continue // 半行不画: 面板自带底色, 不会露出破绽
		}
		item := xc.RECT{Left: rc.Left + c.dropPad, Top: y0, Right: rc.Right - c.dropPad, Bottom: y1}
		if scrollable {
			item.Right -= padX(4)
		}
		hv := xui.HoverItem(c.Handle, calHoverDrop+int32(i), c.hoverDrop == int32(i))
		fore := t.PopoverForeground
		if hv > 0 {
			p.FillRound(&item, t.Accent.ScaleAlpha(hv), t.RadiusSm())
			fore = xui.Fade(fore, t.AccentForeground, hv)
		}
		tr := item
		tr.Left += padX(6)
		tr.Right -= padX(22)
		p.Text(c.dropText(v), &tr, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)

		if c.dropCurrent(v) {
			cs := padX(14)
			ir := xc.RECT{
				Left:   item.Right - padX(6) - cs,
				Top:    item.Top + (item.Bottom-item.Top-cs)/2,
				Right:  item.Right - padX(6),
				Bottom: item.Top + (item.Bottom-item.Top+cs)/2,
			}
			p.Icon(icons.IconCheck, &ir, fore, 2)
		}
	}

	if scrollable {
		trackH := viewBottom - viewTop
		thumbH := trackH * c.dropViewH / c.dropContentH
		if min := padX(16); thumbH < min {
			thumbH = min
		}
		maxScroll := c.dropContentH - c.dropViewH
		y := viewTop
		if maxScroll > 0 {
			y += (trackH - thumbH) * c.dropScroll / maxScroll
		}
		barW := padX(2)
		if barW < 2 {
			barW = 2
		}
		bar := xc.RECT{Left: rc.Right - padX(5), Top: y, Right: rc.Right - padX(5) + barW, Bottom: y + thumbH}
		p.FillRound(&bar, t.Foreground.WithAlpha(38), 1)
	}
}
