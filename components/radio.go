package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Radio 自绘单选按钮: 载体为 Button_Type_Radio 类型的原生按钮。
// 与 shadcn radio-group.tsx 对齐: 选中 = bg-primary 实心圆(含 border-primary)
// + 中心 size-2 的白色(bg-primary-foreground)圆点; focus-visible:border-ring +
// ring-3 ring-ring/50, aria-invalid 红边框 + 常驻红环; 键盘空格选中。
// 同组互斥: 独立使用时由 xcgui 按父容器管理, 组容器 RadioGroup 另做跨层互斥。
type Radio struct {
	*widget.Button
	BoxSize  int32 // 圆形指示器直径(逻辑像素)
	invalid  bool
	readOnly bool   // 用户不可选中, 程序化 SetSelected 仍生效(对应 Base UI 的 readOnly)
	value    string // 所属 RadioGroup 的值(Attach 写入; 独立使用时为空)
	backdrop func(t *xui.Theme) xui.Color
	changes  []func(selected bool)
	// focusRingProxy: 由外层标签/选择卡片代理键盘焦点环(同 Checkbox)。
	focusRingProxy bool
}

// RadioOption 单选按钮配置项。
type RadioOption func(*Radio)

// WithRadioInvalid 初始即为校验失败态(对应 shadcn 的 aria-invalid):
// border-destructive + ring-3 destructive/20(暗色 /40), 常驻不要求焦点。
func WithRadioInvalid() RadioOption {
	return func(r *Radio) { r.invalid = true }
}

// WithRadioReadOnly 用户不可选中该项, 但保留焦点与键盘可达性
// (对应 Base UI Radio Root 的 readOnly; 程序化 SetSelected 仍然生效)。
// 组级只读用 RadioGroup.SetReadOnly, 不用给每项设这个。
func WithRadioReadOnly() RadioOption {
	return func(r *Radio) { r.readOnly = true }
}

// NewRadio 创建单选按钮。cy 传 0 时使用默认行高 24。同组互斥由 xcgui 按父容器自动管理。
func NewRadio(x, y, cx, cy int32, text string, hParent int, opts ...RadioOption) *Radio {
	if cy <= 0 {
		cy = 24
	}
	btn := widget.NewButton(x, y, cx, cy, text, hParent)
	if btn == nil {
		return nil
	}
	r := &Radio{Button: btn, BoxSize: 16}
	for _, o := range opts {
		o(r)
	}
	btn.SetTypeEx(xcc.Button_Type_Radio)
	btn.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter)
	btn.EnableBkTransparent(true)
	btn.SetCursor(controlCursor(!btn.IsEnable()))
	btn.AddEvent_Paint(r.onPaint)
	xui.AttachWindow(r.Handle)
	r.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	// 原生勾选不会自动失效重绘; 鼠标点击经 XE_BUTTON_CHECK 补重绘并通知监听者。
	// 原生勾选在通知到达时已经发生: 条目只读要在这里把勾选位回滚(同 Checkbox,
	// 见 DESIGN 第八节 198), 组级只读由 RadioGroup 统一还原(它要顾及同父项互斥)。
	r.AddEvent_Button_Check(func(hEle int, bCheck bool, pbHandled *bool) int {
		if r.readOnly && bCheck {
			r.SetCheck(false)
			r.Redraw(false)
			return 0
		}
		r.Redraw(false)
		r.notifyChange(bCheck)
		return 0
	})
	// 原生 Check 按钮不处理空格; Radix 的 role=radio 用空格选中当前项。
	r.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		const vkSpace = 0x20
		if wParam == vkSpace && r.IsEnable() {
			if !r.readOnly {
				r.setSelectedNotify(true)
			}
			*pbHandled = true
		}
		return 0
	})
	return r
}

// Selected 返回选中状态。
func (r *Radio) Selected() bool { return r.IsCheck() }

// naturalWidth 返回条目按当前文字需要的宽度(指示器 + 间距 + 文字 + 两侧余量)。
//
// 只给"横向排列"的 RadioGroup 用: 横向布局的主轴是宽度, 条目铺满会把每个条目挤成
// 一整行, 不声明宽度又会被排成 0 宽 —— 所以按文字量一个固定值。几何与 onPaint
// 一致(左 4 + 指示器 + 8 + 文字 + 右 4), 换主题后由 RadioGroup 重新调用。
func (r *Radio) naturalWidth() int32 {
	if r == nil {
		return 0
	}
	const (
		left  int32 = 4
		gap   int32 = 8
		right int32 = 4
	)
	box := r.BoxSize
	if box <= 0 {
		box = 16
	}
	text := r.GetText()
	if text == "" {
		return left + box + right
	}
	t := xui.Current()
	return left + box + gap + xui.TextWidth(text, xui.FontHandle(t.FontDefault)) + right
}

// FieldValue 实现 FieldValueProvider: 选中为 "1", 未选中为空串。
func (r *Radio) FieldValue() string {
	if r == nil || !r.Selected() {
		return ""
	}
	return "1"
}

// Select 选中该项并触发 OnChange(与鼠标点击/空格等价), 已选中时为空操作。
// 用于"整块区域可点"的场景(如选择卡片的卡片本体), 让外部点击走与点击指示器相同的路径。
func (r *Radio) Select() *Radio {
	if r == nil || r.IsCheck() || r.readOnly {
		return r
	}
	r.setSelectedNotify(true)
	return r
}

// SetReadOnly 设置条目只读: 用户点击/空格不再选中, 焦点与 Tab 可达性保留;
// 程序化 SetSelected 仍然生效(同 Base UI 的 readOnly)。
func (r *Radio) SetReadOnly(v bool) *Radio {
	if r == nil || r.readOnly == v {
		return r
	}
	r.readOnly = v
	r.Redraw(false)
	return r
}

// ReadOnly 报告条目是否只读。
func (r *Radio) ReadOnly() bool { return r != nil && r.readOnly }

// SetBackdrop 指定绘制开头铺的宿主底色(默认按"放在 Card 上"用 t.Card)。
// 组件是自绘的局部形状, 重绘不擦底, 必须先铺宿主底色; 放进有底色的容器
// (例如选中态 bg-accent 的选择卡片)时必须跟随, 否则元素矩形会留下一个异色块。
func (r *Radio) SetBackdrop(fn func(t *xui.Theme) xui.Color) *Radio {
	if r == nil {
		return r
	}
	r.backdrop = fn
	r.Redraw(false)
	return r
}

// backdropColor 返回当前宿主底色。
func (r *Radio) backdropColor(t *xui.Theme) xui.Color {
	if r.backdrop != nil {
		return r.backdrop(t)
	}
	return t.Card
}

// SetSelected 设置选中状态(立即重绘), 不触发 OnChange。
func (r *Radio) SetSelected(v bool) *Radio {
	if r.IsCheck() == v {
		return r
	}
	r.SetCheck(v)
	r.Redraw(false)
	return r
}

// setSelectedNotify 供鼠标/键盘路径使用: 写状态并触发与点击相同的回调。
func (r *Radio) setSelectedNotify(v bool) {
	if r.IsCheck() == v {
		return
	}
	r.SetCheck(v)
	r.Redraw(false)
	r.notifyChange(v)
}

// SetInvalid 设置/清除校验失败态(见 WithRadioInvalid)。
func (r *Radio) SetInvalid(v bool) *Radio {
	if r.invalid == v {
		return r
	}
	r.invalid = v
	r.Redraw(false)
	return r
}

// Invalid 返回当前是否处于校验失败态。
func (r *Radio) Invalid() bool { return r.invalid }

// SetDisabled 设置禁用态: 不可点击/聚焦, 指针变 not-allowed, 绘制整体 50%。
func (r *Radio) SetDisabled(v bool) *Radio {
	if r == nil {
		return r
	}
	r.Enable(!v)
	r.EnableFocus(!v)
	r.EnableSwitchFocus(!v)
	r.SetCursor(controlCursor(v))
	r.Redraw(false)
	return r
}

// Disabled 返回是否禁用。
func (r *Radio) Disabled() bool { return r == nil || !r.IsEnable() }

// SetFieldState 实现 FieldControl: 由 Field 级联校验失败/禁用态。
func (r *Radio) SetFieldState(invalid, disabled bool) {
	r.SetInvalid(invalid)
	r.SetDisabled(disabled)
}

// FieldDisabled 实现 FieldControl。
func (r *Radio) FieldDisabled() bool { return r.Disabled() }

// SetFocusRingProxy 由外层包裹元素代理键盘焦点环, 本控件不再画自己的环。
func (r *Radio) SetFocusRingProxy(v bool) *Radio {
	if r == nil || r.focusRingProxy == v {
		return r
	}
	r.focusRingProxy = v
	r.Redraw(false)
	return r
}

// FocusRingProxy 返回是否由外层代理键盘焦点环。
func (r *Radio) FocusRingProxy() bool { return r != nil && r.focusRingProxy }

// OnChange 注册选中状态变化回调(鼠标点击/空格/方向键选择时触发)。
func (r *Radio) OnChange(fn func(selected bool)) *Radio {
	r.addChangeListener(fn)
	return r
}

func (r *Radio) addChangeListener(fn func(selected bool)) {
	r.changes = append(r.changes, fn)
}

func (r *Radio) notifyChange(selected bool) {
	for _, fn := range r.changes {
		fn(selected)
	}
}

func (r *Radio) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)
	checked := r.IsCheck()

	// 先铺宿主底色(整元素): 透明元素重绘**不擦底**(见 DESIGN 第八节 28) —— 未选态在暗色
	// 下是半透明令牌(dark:bg-input/30), 不铺底会透出上一帧的实心圆盘; 单调焦点环也会残留。
	// 组件按"放在 Card 上"设计(同 Table/Checkbox/Switch), 放在异色容器上时用 SetBackdrop 跟随。
	host := r.backdropColor(t)
	p.FillRound(&p.Rc, host, 0)

	// 圆形指示器(逻辑像素, 框架按 DPI 变换渲染)
	size := r.BoxSize
	gap := int32(8)
	circle := xc.RECT{
		Left:   4,
		Top:    (p.H() - size) / 2,
		Right:  4 + size,
		Bottom: (p.H()-size)/2 + size,
	}

	// aria-invalid 的 ring 常驻; focus-visible:ring-3 ring-ring/50 只在键盘焦点时画。
	// 两者同时成立时红色优先(aria-invalid 在变体顺序里靠后)。
	disabled := st.Disabled || !r.IsEnable()
	invalid := r.invalid && !disabled
	focusRing := st.Focused && !disabled && !r.focusRingProxy && xui.FocusedByKeyboard()
	if invalid || focusRing {
		halo := circle
		halo.Left -= 3
		halo.Top -= 3
		halo.Right += 3
		halo.Bottom += 3
		ring := t.Ring.WithAlpha(128)
		if invalid {
			ring = t.Destructive.WithAlpha(51)
			if t.IsDark() {
				ring = t.Destructive.WithAlpha(102)
			}
		}
		p.Draw.SetBrushColor(uint32(ring))
		p.Draw.FillEllipse(&halo)
	}

	// 环带是实心填充, 未选中亮色没有自己的底色(fill=None), 不先盖一层宿主色的话
	// 红色/灰色环会从指示器内部透出来。整元素已铺过宿主底色, 这里只需盖住圆盘。
	p.Draw.SetBrushColor(uint32(host))
	p.Draw.FillEllipse(&circle)

	// shadcn 源码(data-checked): 选中 = bg-primary 圆 + border-primary,
	// 指示器为 size-2 的 bg-primary-foreground 白色圆点(直径 50%)。
	// 未选: border-input 描边 + 暗色额外 dark:bg-input/30 的淡填充(radio-group.tsx);
	// 禁用是 disabled:opacity-50 —— 指示器整体半透明(标签文字不跟随)
	var idleFill xui.Color // 亮色未选态没有自己的底色, 露出宿主(Card)即可
	if t.IsDark() {
		idleFill = t.Input.ScaleAlpha(0.30)
	}
	var fill, border, dot xui.Color
	switch {
	case disabled && checked:
		fill, border, dot = t.Primary.ScaleAlpha(0.5), t.Primary.ScaleAlpha(0.5), t.PrimaryForeground.ScaleAlpha(0.5)
	case disabled:
		// 禁用 = 上述颜色再 ×50%: 注意是"乘"不是覆盖(暗色 input 自带 15% alpha,
		// 覆盖成 50% 会画出一个亮的实心灰盘)
		fill, border, dot = idleFill.ScaleAlpha(0.5), t.Input.ScaleAlpha(0.5), t.Input.ScaleAlpha(0.5)
	case checked:
		fill, border, dot = t.Primary, t.Primary, t.PrimaryForeground
	default:
		fill, border, dot = idleFill, t.Input, t.Input
		// focus-visible:border-ring; aria-invalid 的 border-destructive 覆盖它
		// (aria-invalid:aria-checked:border-primary 对选中态生效, 选中态不描边)。
		if focusRing && !invalid {
			border = t.Ring
		}
		if invalid {
			border = t.Destructive
			if t.IsDark() {
				border = t.Destructive.WithAlpha(128) // dark:aria-invalid:border-destructive/50
			}
		}
	}

	if !fill.IsNone() {
		p.Draw.SetBrushColor(uint32(fill))
		p.Draw.FillEllipse(&circle)
	}
	// 选中时 border 与 fill 同色, 描边只会让圆盘变大、白点占比变小, 跳过
	if !checked {
		p.Draw.SetLineWidth(1)
		p.Draw.SetBrushColor(uint32(border))
		p.Draw.DrawEllipse(&circle)
	}

	// 选中时中心画实心圆点
	if checked {
		dotSize := size * 50 / 100
		center := xc.RECT{
			Left:   circle.Left + (size-dotSize)/2,
			Top:    circle.Top + (size-dotSize)/2,
			Right:  circle.Left + (size+dotSize)/2,
			Bottom: circle.Top + (size+dotSize)/2,
		}
		p.Draw.SetBrushColor(uint32(dot))
		p.Draw.FillEllipse(&center)
	}

	textRC := xc.RECT{
		Left:   circle.Right + gap,
		Top:    p.Rc.Top,
		Right:  p.Rc.Right,
		Bottom: p.Rc.Bottom,
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	// 禁用时文字同样变淡: 参考页暗色下禁用项文字 ≈ #8a8a8a(实测), 亮色下是黑字 50%
	textColor := t.Foreground
	if disabled {
		textColor = t.Foreground.ScaleAlpha(0.5)
	}
	p.Text(r.GetText(), &textRC, textColor, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	return 0
}

// ===== FieldChoiceCard 适配(接口方法不导出, 只有内置控件能当选择卡片指示器) =====

func (r *Radio) choiceCardHandle() int                            { return r.Handle }
func (r *Radio) choiceCardToggle()                                { r.Select() }
func (r *Radio) choiceCardActive() bool                           { return r.Selected() }
func (r *Radio) choiceCardOnChange(fn func(bool))                 { r.OnChange(fn) }
func (r *Radio) choiceCardFocusProxy()                            { r.SetFocusRingProxy(true) }
func (r *Radio) choiceCardBackdrop(fn func(*xui.Theme) xui.Color) { r.SetBackdrop(fn) }
func (r *Radio) choiceCardSize() (int32, int32)                   { return 24, 24 }
