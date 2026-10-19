package ui

import (
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
//     下方放不下时翻到上方, 并夹在屏幕内;
//   - 选中日期即回写 + 关闭 + 回调; 点击外部(弹层失焦)或 ESC 关闭。
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

	panel *popoverPanel // 懒创建, 复用
	cal   *Calendar
}

// datePickerGap 是弹层与触发区的垂直间距(shadcn Popover sideOffset=4)。
const datePickerGap int32 = 4

// DatePickerOption 日期框配置项。
type DatePickerOption func(*DatePicker)

// WithDatePickerPlaceholder 设置未选择时的占位文本。
func WithDatePickerPlaceholder(s string) DatePickerOption {
	return func(d *DatePicker) { d.placeholder = s }
}

// WithDatePickerValue 设置初始值。
func WithDatePickerValue(t time.Time) DatePickerOption {
	return func(d *DatePicker) { d.value = t }
}

// WithDatePickerInvalid 设置校验失败态(边框/ring 同 Button 的 aria-invalid)。
func WithDatePickerInvalid() DatePickerOption {
	return func(d *DatePicker) { d.invalid = true }
}

// NewDatePicker 创建日期框。cy 传 0 用默认高度 32, cx 传 0 用默认宽度 200。
func NewDatePicker(x, y, cx, cy int32, hParent int, opts ...DatePickerOption) *DatePicker {
	if cy <= 0 {
		cy = 32
	}
	if cx <= 0 {
		cx = 200
	}
	d := &DatePicker{placeholder: "选择日期"}
	for _, o := range opts {
		o(d)
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
		xui.NotePointerPress()
		// 弹层开着时点触发区: 弹层先因失焦关闭(更新了 lastClose), 这里不再重开 ——
		// 否则会出现"点两次才关掉"。
		if d.panel != nil && (d.panel.IsOpen() || d.panel.RecentlyClosed()) {
			return 0
		}
		d.Open()
		return 0
	})
	e.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if d.panel != nil {
			d.panel.Destroy()
		}
		d.panel, d.cal = nil, nil
		return 0
	})
	xui.AttachWindow(e.Handle)
	return d
}

// Value 返回当前值, 未选择时为零值。
func (d *DatePicker) Value() time.Time { return d.value }

// SetValue 设置日期, 传零值 = 清空选择。
func (d *DatePicker) SetValue(t time.Time) *DatePicker {
	d.value = t
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
	if !d.value.IsZero() {
		d.cal.SetSelected(d.value) // 顺带把显示月份切到已选日期
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

// ensurePopup 懒创建弹层(卡片 = 月历尺寸)与月历。
func (d *DatePicker) ensurePopup() bool {
	if d.panel != nil {
		return true
	}
	cw, ch := CalWidth(), CalHeight()
	panel := newPopoverPanel(cw, ch, d.Handle)
	if panel == nil {
		return false
	}
	cal := NewCalendar(0, 0, cw, ch, panel.Content(),
		WithCalendarFlat(),
		WithCalendarOnSelect(func(v time.Time) {
			d.value = v
			d.Redraw(false)
			panel.Hide()
			if d.onChange != nil {
				d.onChange(v)
			}
		}))
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
	hv := xui.Hover(hEle, (st.Hover || st.Pressed || d.IsOpen()) && !st.Disabled)

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

	// 左侧日历图标(size-4)
	iconSize := int32(16)
	icon := xc.RECT{
		Left:   p.Rc.Left + 10,
		Top:    p.Rc.Top + (p.H()-iconSize)/2,
		Right:  p.Rc.Left + 10 + iconSize,
		Bottom: p.Rc.Top + (p.H()+iconSize)/2,
	}
	p.Icon(icons.IconCalendar, &icon, fore, 2)

	text := d.value.Format("2006年1月2日")
	col := fore
	if d.value.IsZero() {
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
