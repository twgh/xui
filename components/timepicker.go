package ui

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// TimePicker 时间框: 自绘触发区 + 弹层里两列滚动列表(时 / 分)。
//
// shadcn 没有官方 time-picker, 这里按社区通行形态实现(与 date-picker 同一套
// Popover 骨架):
//   - 触发区 = outline 按钮(与 DatePicker 完全同款): 时钟图标 + "09:30";
//   - 弹层 = 160×248 卡片, 左右两列各 64 宽、8 项高(28×8), 中间 8px 间隙;
//   - 列表载体是原生 ListBox(滚动、滚轮、键盘上下、点击选中都是原生的),
//     条目背景自绘: 选中 bg-primary + 文字 primary-foreground, 悬停 bg-accent(150ms 过渡);
//   - 点条目立即回写触发区, 弹层保持打开(继续选另一个); 点外部 / ESC 关闭。
//
// 为什么不做"确定按钮": shadcn 的 popover 语义是选择即生效, 时间也只有时/分两维,
// 选完继续留在弹层里改更顺手; 需要确认交底时用 OnChange 回调即可。
type TimePicker struct {
	*widget.Element

	hour, minute int
	placeholder  string
	onChange     func(hour, minute int)
	invalid      bool

	panel    *popoverPanel
	hourList *widget.ListBox
	minList  *widget.ListBox
}

// 时间弹层几何(逻辑像素): 两列 64 宽、8 项可见(28×8), p-3 内边距、两列间 8px。
const (
	tpColW   int32 = 64
	tpListH  int32 = 224
	tpPad    int32 = 12
	tpGap    int32 = 8
	tpItemH  int32 = 28
	tpGapY   int32 = 4 // 弹层与触发区的垂直间距(sideOffset)
)

// TimePickerWidth 返回弹层卡片的宽度。
func TimePickerWidth() int32 { return tpPad*2 + tpColW*2 + tpGap }

// TimePickerHeight 返回弹层卡片的高度。
func TimePickerHeight() int32 { return tpPad*2 + tpListH }

// TimePickerOption 时间框配置项。
type TimePickerOption func(*TimePicker)

// WithTimePickerPlaceholder 设置未选择时的占位文本。
func WithTimePickerPlaceholder(s string) TimePickerOption {
	return func(t *TimePicker) { t.placeholder = s }
}

// WithTimePickerValue 设置初始时间(时 0-23, 分 0-59)。
func WithTimePickerValue(hour, minute int) TimePickerOption {
	return func(t *TimePicker) { t.hour, t.minute = hour, minute }
}

// WithTimePickerInvalid 设置校验失败态(边框/ring 同 Button 的 aria-invalid)。
func WithTimePickerInvalid() TimePickerOption {
	return func(t *TimePicker) { t.invalid = true }
}

// NewTimePicker 创建时间框。cy 传 0 用默认高度 32, cx 传 0 用默认宽度 160。
func NewTimePicker(x, y, cx, cy int32, hParent int, opts ...TimePickerOption) *TimePicker {
	if cy <= 0 {
		cy = 32
	}
	if cx <= 0 {
		cx = 160
	}
	t := &TimePicker{placeholder: "选择时间", hour: 0, minute: 0}
	for _, o := range opts {
		o(t)
	}
	e := widget.NewElement(x, y, cx, cy, hParent)
	if e == nil {
		return nil
	}
	t.Element = e
	e.EnableBkTransparent(true)
	e.AddEvent_Paint(t.onPaint)
	e.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if !t.IsEnable() {
			return 0
		}
		xui.NotePointerPress()
		if t.panel != nil && (t.panel.IsOpen() || t.panel.RecentlyClosed()) {
			return 0
		}
		t.Open()
		return 0
	})
	e.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if t.panel != nil {
			t.panel.Destroy()
		}
		t.panel, t.hourList, t.minList = nil, nil, nil
		return 0
	})
	xui.AttachWindow(e.Handle)
	return t
}

// Hour 返回当前小时(0-23)。
func (t *TimePicker) Hour() int { return t.hour }

// Minute 返回当前分钟(0-59)。
func (t *TimePicker) Minute() int { return t.minute }

// SetTime 设置时间(会夹取到合法范围)。弹层打开时两列同步滚动到新值。
func (t *TimePicker) SetTime(hour, minute int) *TimePicker {
	t.hour = clampInt(hour, 0, 23)
	t.minute = clampInt(minute, 0, 59)
	if t.panel != nil && t.panel.IsOpen() {
		t.scrollTo(t.hourList, t.hour)
		t.scrollTo(t.minList, t.minute)
	}
	t.Redraw(false)
	return t
}

// Text 返回 "HH:mm" 格式的当前值。
func (t *TimePicker) Text() string {
	return fmt.Sprintf("%02d:%02d", t.hour, t.minute)
}

// OnChange 注册时间变化回调。
func (t *TimePicker) OnChange(fn func(hour, minute int)) *TimePicker {
	t.onChange = fn
	return t
}

// SetInvalid 设置/清除校验失败态。
func (t *TimePicker) SetInvalid(v bool) *TimePicker {
	if t.invalid == v {
		return t
	}
	t.invalid = v
	t.Redraw(false)
	return t
}

// Open 以编程方式打开弹层。
func (t *TimePicker) Open() *TimePicker {
	if t.panel == nil && !t.ensurePopup() {
		return t
	}
	if t.panel.IsOpen() {
		return t
	}
	t.syncLists()
	t.panel.Show(t.Handle, tpGapY)
	return t
}

// Close 关闭弹层。
func (t *TimePicker) Close() *TimePicker {
	if t.panel != nil {
		t.panel.Hide()
	}
	return t
}

// IsOpen 报告弹层是否处于打开状态。
func (t *TimePicker) IsOpen() bool { return t.panel != nil && t.panel.IsOpen() }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ensurePopup 懒创建弹层与两个列表。
func (t *TimePicker) ensurePopup() bool {
	if t.panel != nil {
		return true
	}
	panel := newPopoverPanel(TimePickerWidth(), TimePickerHeight(), t.Handle)
	if panel == nil {
		return false
	}
	body := panel.Content()
	t.hourList = t.newList(body, tpPad, tpPad, 24, func() int { return t.hour }, func(idx int) {
		t.hour = idx
		t.finishChange()
	})
	t.minList = t.newList(body, tpPad+tpColW+tpGap, tpPad, 60, func() int { return t.minute }, func(idx int) {
		t.minute = idx
		t.finishChange()
	})
	if t.hourList == nil || t.minList == nil {
		panel.Destroy()
		return false
	}
	// 首帧绘制前的初始态: 两列滚到当前值(布局已完成, 且选中态已不依赖原生 Select,
	// 没有内建异步滚动来捣乱) + 滚动条隐藏(显隐归 showScrollBarOnHover 管)。
	panel.OnFaded = func() {
		t.scrollTo(t.hourList, t.hour)
		t.scrollTo(t.minList, t.minute)
		for _, lb := range []*widget.ListBox{t.hourList, t.minList} {
			if lb == nil {
				continue
			}
			if sb := lb.GetScrollBarVObj(); sb != nil && sb.Handle > 0 {
				xui.SetSubtreeAlpha(0, sb.Handle)
			}
		}
	}
	t.panel = panel
	return true
}

// newList 建一列时间数值。
//
// ⚠️ 选中态**不用原生 Select**(SetSelectItem 会触发内建"选中项滚入视口"的异步滚动,
// 在布局完成后执行并覆盖我们的 ScrollPosYV, 两列内容高度不同被钳制的位置不同,
// 选中项错开一行) —— isSel 由 sel() 回调(组件自己的 hour/minute)在 DrawItem 里
// 判定, 滚动完全由 scrollTo 控制; 原生 NState 只用来判悬停。
func (t *TimePicker) newList(parent int, x, y int32, n int, sel func() int, onPick func(int)) *widget.ListBox {
	lb := widget.NewListBox(x, y, tpColW, tpListH, parent)
	if lb == nil {
		return nil
	}
	// 原生 ListBox 必须先建数据适配器才能装数据(同 ComboBox)
	lb.CreateAdapter()
	for i := 0; i < n; i++ {
		lb.AddItemText(fmt.Sprintf("%02d", i))
	}
	lb.EnableBkTransparent(true)
	lb.EnableDrawBorder(false)
	lb.EnableDrawFocus(false)
	lb.SetPadding(4, 4, 4, 4) // p-1
	lb.SetItemHeightDefault(tpItemH, tpItemH)
	lb.SetDrawItemBkFlags(0) // 原生项背景全弃, 交给 DrawItem
	lb.SetScrollBarSize(scrollBarW)
	lb.EnableAutoShowScrollBar(true)
	// 滚动单位 = 行步进(项高+1): ScrollPosYV 的落点会被吸附到滚动单位网格(默认
	// 20px), 两列请求值的余量不同会错开最多 19px(用户截图"两列选中项不在同一高度");
	// 单位 = 行步进后, 两列 pos(相差整行数) 吸附后仍差整行, 选中项必然同高。
	lb.SetLineSize(20, tpItemH+1)
	lb.SetFont(xui.FontHandle(xui.Current().FontDefault))
	styleScrollBar(lb.GetScrollBarVObj(), 0, embedScrollBarSkin(xui.Current()))
	showScrollBarOnHover(lb.Handle, lb.GetScrollBarVObj())

	lb.AddEvent_ListBox_DrawItem(func(hEle int, hDraw int, pItem *xc.ListBox_Item_, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		th := xui.Current()
		isSel := sel() == int(pItem.Index) // 选中态来自组件值, 不依赖原生 Select
		hover := pItem.NState == xcc.List_Item_State_Stay
		switch {
		case isSel:
			p.FillRound(&pItem.RcItem, th.Primary, th.RadiusSm())
		case hover:
			if hv := xui.HoverItem(hEle, pItem.Index, true); hv > 0 {
				p.FillRound(&pItem.RcItem, th.Accent.ScaleAlpha(hv), th.RadiusSm())
			}
		default:
			// 悬停刚移走: 透明度从 1 渐隐到 0
			if hv := xui.HoverItem(hEle, pItem.Index, false); hv > 0 {
				p.FillRound(&pItem.RcItem, th.Accent.ScaleAlpha(hv), th.RadiusSm())
			}
		}
		// 文字自己画: 项模板里的 shapeText 文字色只能整项静态设置, 选中项要反色时
		// 逐帧 SetTextColor 已经晚于原生绘制(实测黑底黑字)。项模板文字统一设成全
		// 透明(见 Temp_Create_End), 这里按状态取色居中绘制。
		col := th.Foreground
		if isSel {
			col = th.PrimaryForeground
		}
		if f := xui.FontHandle(th.FontDefault); f != 0 {
			p.Draw.SetFont(f)
		}
		p.Text(lb.GetItemText(pItem.Index, 0), &pItem.RcItem, col,
			xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		return 0
	})
	lb.AddEvent_ListBox_Temp_Create_End(func(hEle int, pItem *xc.ListBox_Item_, nFlag int32, pbHandled *bool) int {
		if nFlag != 1 || pItem == nil {
			return 0
		}
		if h := lb.GetTemplateObject(pItem.Index, 1); h != 0 {
			xc.XShapeText_SetTextColor(h, 0) // 原生文字透明, 真正文字由 DrawItem 画
		}
		return 0
	})
	lb.AddEvent_ListBox_Select(func(hEle int, iItem int32, pbHandled *bool) int {
		onPick(int(iItem))
		return 0
	})
	return lb
}

// syncLists 打开弹层前对齐两列状态。选中态由 DrawItem 按组件值绘制(不用原生
// Select, 见 newList), 滚动定位在 OnFaded(首帧绘制前)统一执行 —— 这里无需再做。
func (t *TimePicker) syncLists() {}

// scrollTo 让第 idx 项滚动到列表可见区中间。
// 首项顶部还有固有内缩 3 + 内边距 4(实测 ListBox: 首项 top = pad+3, 行步进 = 项高+1),
// 少算这 7px 会让目标项偏上、露不全; 目标位置对齐到行网格(滚动单位 = 行步进, 见
// newList), 落点精确且两列一致。
func (t *TimePicker) scrollTo(lb *widget.ListBox, idx int) {
	pos := 7 + int32(idx)*(tpItemH+1) - (tpListH-tpItemH)/2
	pos = pos / (tpItemH + 1) * (tpItemH + 1) // 对齐行网格
	if pos < 0 {
		pos = 0
	}
	lb.ScrollPosYV(pos)
}

// finishChange 回写触发区并回调(不关闭弹层)。
func (t *TimePicker) finishChange() {
	t.Redraw(false)
	if t.onChange != nil {
		t.onChange(t.hour, t.minute)
	}
}

func (t *TimePicker) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()
	st := xui.StateOf(hEle)
	hv := xui.Hover(hEle, (st.Hover || st.Pressed || t.IsOpen()) && !st.Disabled)

	bg := xui.Fade(th.Background, th.Accent, hv)
	border := th.Input
	fore := xui.Fade(th.Foreground, th.AccentForeground, hv)
	if st.Disabled {
		bg = th.Input.ScaleAlpha(0.5).ScaleAlpha(0.5)
		border = th.Input.ScaleAlpha(0.5)
		fore = th.Foreground.Mix(th.Background, 0.5)
	}

	const inset = int32(3)
	switch {
	case t.invalid && !st.Disabled:
		halo, borderColor := th.Destructive.WithAlpha(51), th.Destructive
		if th.IsDark() {
			halo, borderColor = th.Destructive.WithAlpha(102), th.Destructive.WithAlpha(128)
		}
		p.RingFrame(halo, bg, th.RadiusLg())
		inner := xc.RECT{
			Left:   p.Rc.Left + inset,
			Top:    p.Rc.Top + inset,
			Right:  p.Rc.Right - inset,
			Bottom: p.Rc.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.StrokeRound(&inner, borderColor, th.RadiusLg()-inset, 1)
		}
	default:
		p.FillRound(&p.Rc, bg, th.RadiusLg())
		p.StrokeRound(&p.Rc, border, th.RadiusLg(), 1)
	}

	iconSize := int32(16)
	icon := xc.RECT{
		Left:   p.Rc.Left + 10,
		Top:    p.Rc.Top + (p.H()-iconSize)/2,
		Right:  p.Rc.Left + 10 + iconSize,
		Bottom: p.Rc.Top + (p.H()+iconSize)/2,
	}
	p.Icon(icons.IconClock, &icon, fore, 2)

	if f := xui.FontHandle(th.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	trc := xc.RECT{
		Left:   icon.Right + 8,
		Top:    p.Rc.Top,
		Right:  p.Rc.Right - 10,
		Bottom: p.Rc.Bottom,
	}
	p.Text(t.Text(), &trc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	return 0
}
