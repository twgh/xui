package ui

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// DatePicker 日期框: 自绘触发区 + 自建弹层(内含自绘 Calendar)。
//
// 与 shadcn date-picker 示例对齐(Button variant="outline" + Popover + Calendar):
//   - 触发区 = outline 按钮: h-8 rounded-lg border-input bg-background, 左侧
//     size-4 日历图标 + 文字, 悬停 bg-accent / 文字 accent-foreground;
//   - 未选择时显示占位符(muted-foreground);
//   - 点击弹出月历卡片, 弹层与触发区水平左对齐、垂直相距 4px(sideOffset=4),
//     下方放不下时翻到上方, 并夹在显示器工作区内;
//   - 选中日期即回写 + 关闭 + 回调; 点击外部(弹层失焦)或 ESC 关闭。
//   - **范围模式**(WithDatePickerRange): 弹层 = 两个并排月份 + 范围选择;
//     第一次点击落起点(弹层保持打开), 第二次补终点后关闭;
//     触发区显示 "2026年1月20日 - 2026年2月9日"。
//   - **预设**(WithDatePickerPresets): 弹层底部一排预设按钮(今天/明天/一周后…),
//     点击即选中并关闭, 与 shadcn 的 Calendar presets 一致。
//   - **可输入形态**(WithDatePickerEditable): 触发区变成 InputGroup —— 文字区是原生
//     Edit(可键入 2026-10-22 / 2026年10月22日 / 今天/明天/后天/N天后…), 右侧日历
//     图标按钮打开弹层; ↑↓ 等编辑行为保持原生, Enter/失焦时解析回写。
//
// 为什么不用原生 DateTime 元素: 它的日期文本是 DLL 内绘制(格式固定为 yyyy/MM/dd、
// 右侧带两个无法移除的上下调节按钮), 也拿不到"选择日期"占位符与图标布局;
// 月历弹层则是原生 MonthCal, 格子直角、比例固定(探针实测: BkM 的圆角背景无效,
// 选中格只能是直角大色块)。这里触发区与月历全部自绘, 视觉与 shadcn 一致;
// 弹层骨架见 popup.go。
type DatePicker struct {
	*widget.Element

	value       time.Time
	placeholder string
	onChange    func(time.Time)
	invalid     bool

	rangeMode bool
	rangeFrom time.Time
	rangeTo   time.Time
	onRange   func(from, to time.Time)
	yearFrom  int
	yearTo    int
	disabled  func(time.Time) bool
	presets   []CalendarPreset

	editable      bool
	edit          *widget.Edit
	editEnabled   bool
	hoverIcon     bool
	editTextColor uint32

	panel *popoverPanel // 懒创建, 复用
	cal   *Calendar
}

// datePickerGap 是弹层与触发区的垂直间距(shadcn Popover sideOffset=4)。
const datePickerGap int32 = 4

// datePickerIconArea 是可输入形态右侧日历按钮的命中宽度(逻辑像素)。
const datePickerIconArea int32 = 32

// datePickerHoverIcon 是右侧图标钮的 hover 过渡键(日期框只有一个内嵌钮)。
const datePickerHoverIcon int32 = 900

// DatePickerOption 日期框配置项。
type DatePickerOption func(*DatePicker)

// WithDatePickerPlaceholder 设置未选择时的占位文本。
func WithDatePickerPlaceholder(s string) DatePickerOption {
	return func(d *DatePicker) { d.placeholder = s }
}

// WithDatePickerValue 设置初始值(范围模式下作为起点)。
func WithDatePickerValue(t time.Time) DatePickerOption {
	return func(d *DatePicker) { d.value = t }
}

// WithDatePickerInvalid 设置校验失败态(边框/ring 同 Button 的 aria-invalid)。
func WithDatePickerInvalid() DatePickerOption {
	return func(d *DatePicker) { d.invalid = true }
}

// WithDatePickerRange 启用范围选择模式(弹层为两个并排月份)。
func WithDatePickerRange() DatePickerOption {
	return func(d *DatePicker) { d.rangeMode = true }
}

// WithDatePickerRangeValue 启用范围模式并设置初始区间(只传 from 表示区间未完成)。
func WithDatePickerRangeValue(from, to time.Time) DatePickerOption {
	return func(d *DatePicker) {
		d.rangeMode = true
		d.rangeFrom, d.rangeTo = calNormalizeRange(from, to)
		d.value = d.rangeFrom
	}
}

// WithDatePickerOnRangeChange 注册范围变化回调(第一次点击起点也会回调, to 为零)。
func WithDatePickerOnRangeChange(fn func(from, to time.Time)) DatePickerOption {
	return func(d *DatePicker) { d.onRange = fn }
}

// WithDatePickerYearRange 限定年月下拉里的年份范围(出生日期这类场景),
// 不设置时是"视图年 ±50"。
func WithDatePickerYearRange(from, to int) DatePickerOption {
	return func(d *DatePicker) {
		if from > to {
			from, to = to, from
		}
		d.yearFrom, d.yearTo = from, to
	}
}

// WithDatePickerDisabled 设置禁用日期判定(禁用格不可点、不可选)。
func WithDatePickerDisabled(fn func(time.Time) bool) DatePickerOption {
	return func(d *DatePicker) { d.disabled = fn }
}

// WithDatePickerPresets 在弹层底部放一组预设日期按钮。范围模式下预设会一次给出
// 完整区间; 单选模式点击预设即选中并关闭弹层。
func WithDatePickerPresets(presets ...CalendarPreset) DatePickerOption {
	return func(d *DatePicker) { d.presets = append(d.presets, presets...) }
}

// WithDatePickerEditable 把触发区换成可输入形态(InputGroup): 文字区是原生 Edit,
// 右侧日历图标打开弹层; Enter/失焦时解析输入(支持 2026-10-22、2026年10月22日、
// 今天/明天/后天/大后天、N天后、N周后)。范围模式暂不支持可输入形态。
func WithDatePickerEditable() DatePickerOption {
	return func(d *DatePicker) { d.editable = true }
}

// NewDatePicker 创建日期框。cy 传 0 用默认高度 32, cx 传 0 用默认宽度 200。
func NewDatePicker(x, y, cx, cy int32, hParent int, opts ...DatePickerOption) *DatePicker {
	if cy <= 0 {
		cy = 32
	}
	if cx <= 0 {
		cx = 200
	}
	d := &DatePicker{placeholder: "选择日期", editEnabled: true}
	for _, o := range opts {
		o(d)
	}
	d.rangeFrom, d.rangeTo = calNormalizeRange(d.rangeFrom, d.rangeTo)
	if d.rangeMode && d.rangeFrom.IsZero() && !d.value.IsZero() {
		d.rangeFrom = d.value
	}
	if d.rangeMode {
		d.editable = false // 范围文本无法用单个原生 Edit 表达, 退回普通触发区
	}
	e := widget.NewElement(x, y, cx, cy, hParent)
	if e == nil {
		return nil
	}
	d.Element = e
	e.EnableBkTransparent(true)
	e.AddEvent_Paint(d.onPaint)
	e.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if !d.IsEnable() {
			return 0
		}
		// 可输入形态: 文字区归原生 Edit, 只有右侧日历图标打开弹层
		if d.editable && !d.iconHit(pPt.X) {
			return 0
		}
		xui.NotePointerPress()
		// 弹层开着时点触发区: 弹层先因失焦关闭(更新了 lastClose), 这里不再重开 ——
		// 否则会出现"点两次才关掉"。
		if d.panel != nil && (d.panel.IsOpen() || d.panel.RecentlyClosed()) {
			return 0
		}
		d.Open()
		return 0
	})
	e.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if !d.editable {
			return 0
		}
		hit := d.iconHit(pPt.X)
		if hit != d.hoverIcon {
			d.hoverIcon = hit
			d.Redraw(false)
		}
		return 0
	})
	e.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		if d.hoverIcon {
			d.hoverIcon = false
			d.Redraw(false)
		}
		return 0
	})
	e.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		d.syncEditRect()
		return 0
	})
	e.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if d.panel != nil {
			d.panel.Destroy()
		}
		d.panel, d.cal, d.edit = nil, nil, nil
		return 0
	})
	if d.editable {
		d.createEdit(cx, cy)
	}
	xui.AttachWindow(e.Handle)
	return d
}

// createEdit 建可输入形态的原生 Edit(壳体/图标仍由 DatePicker 自绘)。
func (d *DatePicker) createEdit(cx, cy int32) {
	t := xui.Current()
	e := widget.NewEdit(0, 0, cx-datePickerIconArea, cy, d.Handle)
	if e == nil {
		d.editable = false
		return
	}
	d.edit = e
	e.EnableBkTransparent(true)
	e.EnableDrawBorder(false).EnableDrawFocus(false)
	e.EnableAutoSelAll(false)
	// 与 Input 同口径: 左 12 / 上 7 / 右 4 / 下 5(原生 Edit 行盒居中, 下侧多留白会显上浮)
	e.SetBorderSize(12, 7, 4, 5)
	d.applyEditTheme(t)
	e.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		switch wParam {
		case xcc.VK_Enter:
			d.commitEdit()
			*pbHandled = true
		case xcc.VK_Down:
			if d.panel == nil || !d.panel.IsOpen() {
				d.Open()
			}
			*pbHandled = true
		}
		return 0
	})
	e.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		d.commitEdit()
		return 0
	})
	e.AddEvent_Edit_Changed(func(hEle int, pbHandled *bool) int {
		d.Redraw(false)
		return 0
	})
	xui.OnThemeChangeFor(e.Handle, func(t *xui.Theme) {
		if d.edit != nil && xc.XC_IsHXCGUI(d.edit.Handle, xcc.XC_ELE) {
			d.applyEditTheme(t)
		}
	})
	d.syncEditText()
	d.syncEditRect()
}

// syncEditRect 让原生 Edit 始终占满"图标区之前"的宽度。构造时的宽度只是初始值,
// 元素被布局拉伸(Fill / Weight)后必须跟着改, 否则右侧一大段点不进输入区。
func (d *DatePicker) syncEditRect() {
	if d.edit == nil || d.Element == nil {
		return
	}
	w := d.GetWidth() - datePickerIconArea
	h := d.GetHeight()
	if w < 0 {
		w = 0
	}
	d.edit.SetWidth(w)
	d.edit.SetHeight(h)
}

// applyEditTheme 把字体/字色/占位符色同步给原生 Edit(它不继承窗口字体, 见 DESIGN 3.2)。
func (d *DatePicker) applyEditTheme(t *xui.Theme) {
	if d.edit == nil {
		return
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		d.edit.SetFont(f)
	}
	c := uint32(t.Foreground)
	if d.editTextColor != c {
		d.editTextColor = c
		d.edit.SetTextColor(c)
	}
	d.edit.SetDefaultText(d.placeholder)
	d.edit.SetDefaultTextColor(uint32(t.MutedForeground))
	d.edit.SetCaretColor(uint32(t.Primary))
}

// iconHit 判断 x 是否落在可输入形态右侧的日历图标区。
func (d *DatePicker) iconHit(x int32) bool {
	if d.Element == nil {
		return false
	}
	return x >= d.GetWidth()-datePickerIconArea
}

// syncEditText 把当前值同步到可输入形态的文字区。
func (d *DatePicker) syncEditText() {
	if d.edit == nil {
		return
	}
	d.edit.SetText(d.Text())
}

// commitEdit 解析输入框文字并回写; 解析失败时恢复成当前值。
func (d *DatePicker) commitEdit() {
	if d.edit == nil {
		return
	}
	s := strings.TrimSpace(d.edit.GetText())
	if s == "" {
		if !d.value.IsZero() {
			d.SetValue(time.Time{})
			if d.onChange != nil {
				d.onChange(time.Time{})
			}
		}
		d.syncEditText()
		return
	}
	t, ok := ParseDateText(s)
	if !ok {
		d.syncEditText()
		return
	}
	if calSameDay(t, d.value) {
		d.syncEditText()
		return
	}
	d.SetValue(t)
	d.syncEditText()
	if d.onChange != nil {
		d.onChange(d.value)
	}
}

var (
	dateAfterDaysRe  = regexp.MustCompile(`^(\d+)\s*天(?:以)?后$`)
	dateAfterWeeksRe = regexp.MustCompile(`^(\d+)\s*(?:周|个星期)(?:以)?后$`)
)

// dateInputLayouts 是可输入形态接受的具体日期格式。
var dateInputLayouts = []string{
	"2006年1月2日", "2006年01月02日",
	"2006-1-2", "2006-01-02",
	"2006/1/2", "2006/01/02",
	"2006.1.2", "2006.01.02",
}

// ParseDateText 解析日期输入: 具体日期(2026-10-22 / 2026年10月22日 / 2026/10/22 等)
// 与常用相对日期(今天/明天/后天/大后天、N天后、N周后)。解析结果按本地时区归一。
func ParseDateText(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	now := calDay(time.Now())
	switch strings.ToLower(s) {
	case "今天", "今日", "today":
		return now, true
	case "明天", "明日", "tomorrow":
		return now.AddDate(0, 0, 1), true
	case "后天":
		return now.AddDate(0, 0, 2), true
	case "大后天":
		return now.AddDate(0, 0, 3), true
	}
	if m := dateAfterDaysRe.FindStringSubmatch(s); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return now.AddDate(0, 0, n), true
		}
	}
	if m := dateAfterWeeksRe.FindStringSubmatch(s); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return now.AddDate(0, 0, n*7), true
		}
	}
	for _, layout := range dateInputLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return calDay(t), true
		}
	}
	return time.Time{}, false
}

// Value 返回当前值, 未选择时为零值。范围模式返回起点(终点用 Range)。
func (d *DatePicker) Value() time.Time { return d.value }

// Range 返回当前范围(单选模式时 to 为零值)。
func (d *DatePicker) Range() (from, to time.Time) { return d.rangeFrom, d.rangeTo }

// RangeMode 报告是否处于范围选择模式。
func (d *DatePicker) RangeMode() bool { return d.rangeMode }

// Text 返回触发区显示的日期文本(未选择时为空串)。
func (d *DatePicker) Text() string {
	if d.rangeMode {
		if d.rangeFrom.IsZero() {
			return ""
		}
		if d.rangeTo.IsZero() {
			return d.rangeFrom.Format("2006年1月2日")
		}
		return d.rangeFrom.Format("2006年1月2日") + " - " + d.rangeTo.Format("2006年1月2日")
	}
	if d.value.IsZero() {
		return ""
	}
	return d.value.Format("2006年1月2日")
}

// SetValue 设置日期, 传零值 = 清空选择。范围模式下会把起点设为该日期并清空终点。
func (d *DatePicker) SetValue(t time.Time) *DatePicker {
	d.value = t
	if d.rangeMode {
		d.rangeFrom, d.rangeTo = t, time.Time{}
	}
	if d.cal != nil {
		if d.rangeMode {
			d.cal.SetRange(t, time.Time{})
		} else {
			d.cal.SetSelected(t)
		}
	}
	d.syncEditText()
	d.Redraw(false)
	return d
}

// SetRange 设置范围(同时切到范围模式; 传零值 = 清除)。to 早于 from 时自动交换。
func (d *DatePicker) SetRange(from, to time.Time) *DatePicker {
	d.rangeMode = true
	d.rangeFrom, d.rangeTo = calNormalizeRange(from, to)
	d.value = d.rangeFrom
	if d.cal != nil {
		d.cal.SetRange(d.rangeFrom, d.rangeTo)
	}
	d.Redraw(false)
	return d
}

// Clear 清空选择。
func (d *DatePicker) Clear() *DatePicker { return d.SetValue(time.Time{}) }

// OnChange 注册选择变化回调。
func (d *DatePicker) OnChange(fn func(time.Time)) *DatePicker {
	d.onChange = fn
	return d
}

// OnRangeChange 注册范围变化回调。
func (d *DatePicker) OnRangeChange(fn func(from, to time.Time)) *DatePicker {
	d.onRange = fn
	return d
}

// SetInvalid 设置/清除校验失败态。
func (d *DatePicker) SetInvalid(v bool) *DatePicker {
	if d.invalid == v {
		return d
	}
	d.invalid = v
	d.Redraw(false)
	return d
}

// Open 以编程方式打开弹层(效果同点击触发区)。
func (d *DatePicker) Open() *DatePicker {
	if d.panel == nil && !d.ensurePopup() {
		return d
	}
	if d.panel.IsOpen() {
		return d
	}
	if d.cal != nil {
		if d.rangeMode {
			d.cal.SetRange(d.rangeFrom, d.rangeTo)
		} else if !d.value.IsZero() {
			d.cal.SetSelected(d.value) // 顺带把显示月份切到已选日期
		}
	}
	d.panel.Show(d.Handle, datePickerGap)
	return d
}

// Close 关闭弹层。
func (d *DatePicker) Close() *DatePicker {
	if d.panel != nil {
		d.panel.Hide()
	}
	return d
}

// IsOpen 报告弹层是否处于打开状态。
func (d *DatePicker) IsOpen() bool { return d.panel != nil && d.panel.IsOpen() }

// ensurePopup 懒创建弹层与月历(范围模式两个并排月份)。
func (d *DatePicker) ensurePopup() bool {
	if d.panel != nil {
		return true
	}
	months := 1
	if d.rangeMode {
		months = 2
	}
	cw := CalWidthFor(months)
	ch := CalHeightForMonths(cw, months)
	// 预设按钮在月历页脚里绘制, 弹层高度要把换行后的页脚算进来
	if len(d.presets) > 0 {
		items := make([]CalendarFooterItem, len(d.presets))
		for i, p := range d.presets {
			items[i] = CalendarFooterItem{Text: p.Label, OnClick: func() {}}
		}
		ch += calLayoutFooter(cw, months, false, items).totalH
	}
	panel := newPopoverPanel(cw, ch, d.Handle)
	if panel == nil {
		return false
	}
	opts := []CalendarOption{WithCalendarFlat(), WithCalendarMonths(months)}
	if d.rangeMode {
		opts = append(opts,
			WithCalendarRange(),
			WithCalendarSelectedRange(d.rangeFrom, d.rangeTo),
			WithCalendarOnRangeSelect(func(from, to time.Time) {
				d.rangeFrom, d.rangeTo = from, to
				d.value = from
				d.Redraw(false)
				if d.onRange != nil {
					d.onRange(from, to)
				}
				if !to.IsZero() {
					panel.Hide()
				}
			}))
	} else {
		opts = append(opts,
			WithCalendarSelected(d.value),
			WithCalendarOnSelect(func(v time.Time) {
				d.value = v
				d.syncEditText()
				d.Redraw(false)
				panel.Hide()
				if d.onChange != nil {
					d.onChange(v)
				}
			}))
	}
	if d.yearFrom != 0 || d.yearTo != 0 {
		opts = append(opts, WithCalendarYearRange(d.yearFrom, d.yearTo))
	}
	if d.disabled != nil {
		opts = append(opts, WithCalendarDisabled(d.disabled))
	}
	if len(d.presets) > 0 {
		opts = append(opts, WithCalendarPresets(d.presets...))
	}
	cal := NewCalendar(0, 0, cw, ch, panel.Content(), opts...)
	if cal == nil {
		panel.Destroy()
		return false
	}
	d.panel, d.cal = panel, cal
	return true
}

func (d *DatePicker) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)
	if d.edit != nil {
		enabled := !st.Disabled
		if d.editEnabled != enabled {
			d.editEnabled = enabled
			d.edit.Enable(enabled)
		}
	}
	activeHover := st.Hover || st.Pressed || d.IsOpen() || (d.editable && d.hoverIcon)
	hv := xui.Hover(hEle, activeHover && !st.Disabled)

	// outline 按钮: bg-background + border-input, hover:bg-accent / text-accent-foreground
	bg := xui.Fade(t.Background, t.Accent, hv)
	border := t.Input
	fore := xui.Fade(t.Foreground, t.AccentForeground, hv)
	if st.Disabled {
		bg = t.Input.ScaleAlpha(0.5).ScaleAlpha(0.5)
		border = t.Input.ScaleAlpha(0.5)
		fore = t.Foreground.Mix(t.Background, 0.5)
	}

	const inset = int32(3)
	switch {
	case d.invalid && !st.Disabled:
		halo, borderColor := t.Destructive.WithAlpha(51), t.Destructive
		if t.IsDark() {
			halo, borderColor = t.Destructive.WithAlpha(102), t.Destructive.WithAlpha(128)
		}
		p.RingFrame(halo, bg, t.RadiusLg())
		inner := xc.RECT{
			Left:   p.Rc.Left + inset,
			Top:    p.Rc.Top + inset,
			Right:  p.Rc.Right - inset,
			Bottom: p.Rc.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.StrokeRound(&inner, borderColor, t.RadiusLg()-inset, 1)
		}
	default:
		p.FillRound(&p.Rc, bg, t.RadiusLg())
		p.StrokeRound(&p.Rc, border, t.RadiusLg(), 1)
	}

	iconSize := int32(16)
	if d.editable {
		// InputGroup 形态: 右侧幽灵图标钮(24×24), 文字区由原生 Edit 画
		btnH := int32(24)
		btn := xc.RECT{
			Left:   p.Rc.Right - 4 - btnH,
			Top:    p.Rc.Top + (p.H()-btnH)/2,
			Right:  p.Rc.Right - 4,
			Bottom: p.Rc.Top + (p.H()+btnH)/2,
		}
		ihv := xui.HoverItem(hEle, datePickerHoverIcon, (d.hoverIcon || d.IsOpen()) && !st.Disabled)
		if ihv > 0 {
			p.FillRound(&btn, t.Accent.ScaleAlpha(ihv), t.RadiusSm())
		}
		iconColor := xui.Fade(t.MutedForeground, t.AccentForeground, ihv)
		if st.Disabled {
			iconColor = t.MutedForeground.ScaleAlpha(0.5)
		}
		icon := xc.RECT{
			Left:   btn.Left + (btnH-iconSize)/2,
			Top:    btn.Top + (btnH-iconSize)/2,
			Right:  btn.Left + (btnH+iconSize)/2,
			Bottom: btn.Top + (btnH+iconSize)/2,
		}
		p.Icon(icons.IconCalendar, &icon, iconColor, 2)
		return 0
	}

	// 左侧日历图标(size-4)
	icon := xc.RECT{
		Left:   p.Rc.Left + 10,
		Top:    p.Rc.Top + (p.H()-iconSize)/2,
		Right:  p.Rc.Left + 10 + iconSize,
		Bottom: p.Rc.Top + (p.H()+iconSize)/2,
	}
	p.Icon(icons.IconCalendar, &icon, fore, 2)

	text := d.Text()
	col := fore
	if text == "" {
		text = d.placeholder
		col = t.MutedForeground
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	trc := xc.RECT{
		Left:   icon.Right + 8,
		Top:    p.Rc.Top,
		Right:  p.Rc.Right - 10,
		Bottom: p.Rc.Bottom,
	}
	p.Text(text, &trc, col, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	return 0
}
