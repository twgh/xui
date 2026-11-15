package ui

import (
	"github.com/twgh/xcgui/drawx"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Checkbox 自绘复选框: 载体为 Button_Type_Check 类型的原生按钮,
// 左侧绘制圆角方框与勾号/减号, 文本由本组件绘制(需要与指示器对齐排版)。
//
// 与 shadcn checkbox.tsx 对齐: 指示器 size-4 rounded-[4px] border-input,
// 选中 bg-primary border-primary 白勾, focus-visible:border-ring + ring-3 ring-ring/50,
// aria-invalid 红边框 + 常驻红环; 支持不确定态(checked="indeterminate" 的减号)。
// 键盘用空格切换(与 Radix 的 role=checkbox 一致)。
type Checkbox struct {
	*widget.Button
	BoxSize       int32 // 指示器边长(逻辑像素), shadcn 为 size-4
	invalid       bool
	indeterminate bool
	readOnly      bool // 用户不可改, 程序化 SetChecked 仍生效(对应 Base UI 的 readOnly)
	// autoWidth: cx 传 0 创建的复选框, 换文字后用 FitWidth 重新量宽。
	autoWidth bool
	// focusRingProxy: 由外层标签/选择卡片代理 focus-visible 环时置 true,
	// 本控件不再画自己的环(对应 checkbox.tsx 的 group-has-[:focus-visible]/field-label:ring-0)。
	focusRingProxy bool
	backdrop       func(t *xui.Theme) xui.Color
	changes        []func(checked bool)
	vetoes         []func(checked bool) bool
}

// CheckboxOption 复选框配置项。
type CheckboxOption func(*Checkbox)

// WithCheckboxInvalid 初始即为校验失败态(对应 shadcn 的 aria-invalid):
// border-destructive + ring-3 destructive/20(暗色 /40), 常驻不要求焦点。
func WithCheckboxInvalid() CheckboxOption {
	return func(c *Checkbox) { c.invalid = true }
}

// WithCheckboxIndeterminate 初始为不确定态(指示器画减号, 选中色)。
// 运行时用 SetIndeterminate 切换。
func WithCheckboxIndeterminate() CheckboxOption {
	return func(c *Checkbox) { c.indeterminate = true }
}

// WithCheckboxChecked 初始选中的非受控写法(对应 Base UI 的 defaultChecked)。
// 受控写法用 SetChecked + OnChange; 两者等价, 这个选项只是构造时少写一行。
func WithCheckboxChecked(v bool) CheckboxOption {
	return func(c *Checkbox) {
		if v {
			c.SetCheck(true)
		}
	}
}

// WithCheckboxReadOnly 用户不可勾选/取消, 但保留焦点与键盘可达性
// (对应 Base UI 的 readOnly; 程序化 SetChecked 仍然生效)。
func WithCheckboxReadOnly() CheckboxOption {
	return func(c *Checkbox) { c.readOnly = true }
}

// WithCheckboxDisabled 初始即禁用(不可点击/聚焦, 整体 50% 透明)。
func WithCheckboxDisabled() CheckboxOption {
	return func(c *Checkbox) { c.SetDisabled(true) }
}

// checkboxIndicatorW / checkboxIndicatorH 是纯指示器(无文字)复选框的默认元素尺寸:
// 16 的方框外再留 after:-inset-x-3 after:-inset-y-2 的点击热区, 约 40×32。
const (
	checkboxIndicatorW int32 = 40
	checkboxIndicatorH int32 = 32
)

// NewCheckbox 创建复选框。cy 传 0 时: 有文字用行高 24, 纯指示器用 32(点击热区);
// cx 传 0 时按文字量宽, 纯指示器用 40。
func NewCheckbox(x, y, cx, cy int32, text string, hParent int, opts ...CheckboxOption) *Checkbox {
	autoW := cx <= 0
	if cy <= 0 {
		if text == "" {
			cy = checkboxIndicatorH
		} else {
			cy = 24
		}
	}
	if autoW {
		if text == "" {
			cx = checkboxIndicatorW
		} else {
			cx = checkboxNaturalWidth(xui.Current(), text, 16)
		}
	}
	btn := widget.NewButton(x, y, cx, cy, text, hParent)
	if btn == nil {
		return nil
	}
	c := &Checkbox{Button: btn, BoxSize: 16, autoWidth: autoW}
	for _, o := range opts {
		o(c)
	}
	btn.SetTypeEx(xcc.Button_Type_Check)
	btn.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter)
	btn.EnableBkTransparent(true)
	btn.SetCursor(controlCursor(!btn.IsEnable()))
	btn.AddEvent_Paint(c.onPaint)
	xui.AttachWindow(c.Handle)
	c.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	// 原生勾选不会自动失效重绘; 鼠标点击经 XE_BUTTON_CHECK 补重绘并通知监听者。
	// 不确定态只是显示态: 用户点击后按 Radix 语义落到"选中"。
	//
	// 原生勾选在通知到达时**已经发生**, 事件没有"否决"能力: readOnly 与
	// OnCheckedChange 返回 false 都要在这里把状态回滚(见 DESIGN 第八节 198)。
	c.AddEvent_Button_Check(func(hEle int, bCheck bool, pbHandled *bool) int {
		if c.readOnly || !c.allowChange(bCheck) {
			c.revertUserChange(bCheck)
			return 0
		}
		if bCheck {
			c.indeterminate = false
		}
		c.Redraw(false)
		c.notifyChange(bCheck)
		return 0
	})
	// 原生 Check 按钮不处理空格(实测 KeyDown 能收到、状态不变), Radix 的
	// role=checkbox 用空格切换, 键盘激活自己补。readOnly 时吞掉空格但不改状态。
	c.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		const vkSpace = 0x20
		if wParam == vkSpace && c.IsEnable() {
			if !c.readOnly {
				c.toggleNotify()
			}
			*pbHandled = true
		}
		return 0
	})
	return c
}

// checkboxNaturalWidth 量出复选框的自适应宽度: 左内边距 + 指示器 + 间距 + 文字宽。
func checkboxNaturalWidth(t *xui.Theme, text string, box int32) int32 {
	if text == "" {
		return checkboxIndicatorW
	}
	const (
		left  int32 = 4
		gap   int32 = 8
		right int32 = 4
	)
	return left + box + gap + xui.TextWidth(text, xui.FontHandle(t.FontDefault)) + right
}

// Checked 返回选中状态(不确定态返回 false)。
func (c *Checkbox) Checked() bool { return c.IsCheck() }

// FieldValue 实现 FieldValueProvider: 选中为 "1", 未选(含不确定态)为空串。
func (c *Checkbox) FieldValue() string {
	if c == nil || !c.Checked() {
		return ""
	}
	return "1"
}

// Indeterminate 返回是否处于不确定态。
func (c *Checkbox) Indeterminate() bool { return c.indeterminate }

// ReadOnly 返回用户是否被禁止改选。
func (c *Checkbox) ReadOnly() bool { return c.readOnly }

// SetReadOnly 设置只读: 用户点击/空格不再切换, 但焦点与 Tab 可达性保留;
// 程序化 SetChecked / SetIndeterminate 仍然生效(同 Base UI 的 readOnly)。
func (c *Checkbox) SetReadOnly(v bool) *Checkbox {
	if c.readOnly == v {
		return c
	}
	c.readOnly = v
	c.Redraw(false)
	return c
}

// SetDisabled 设置禁用态: 不可点击、不可聚焦, 鼠标指针变 not-allowed,
// 绘制按 disabled:opacity-50 各图层乘 50%。
func (c *Checkbox) SetDisabled(v bool) *Checkbox {
	if c == nil {
		return c
	}
	c.Enable(!v)
	c.EnableFocus(!v)
	c.EnableSwitchFocus(!v)
	c.SetCursor(controlCursor(v))
	c.Redraw(false)
	return c
}

// Disabled 返回是否禁用。
func (c *Checkbox) Disabled() bool { return c == nil || !c.IsEnable() }

// SetFieldState 实现 FieldControl: 由 Field 级联校验失败/禁用态。
func (c *Checkbox) SetFieldState(invalid, disabled bool) {
	c.SetInvalid(invalid)
	c.SetDisabled(disabled)
}

// FieldDisabled 实现 FieldControl: 供 Field 在重新启用时恢复控件自己的禁用态。
func (c *Checkbox) FieldDisabled() bool { return c.Disabled() }

// SetFocusRingProxy 声明键盘焦点环由外层包裹元素(选择卡片 / FieldLabel)代理,
// 本控件不再画自己的环。对应 checkbox.tsx 的 group-has-[:focus-visible]/field-label:ring-0。
func (c *Checkbox) SetFocusRingProxy(v bool) *Checkbox {
	if c.focusRingProxy == v {
		return c
	}
	c.focusRingProxy = v
	c.Redraw(false)
	return c
}

// FocusRingProxy 返回是否由外层代理键盘焦点环。
func (c *Checkbox) FocusRingProxy() bool { return c.focusRingProxy }

// SetChecked 设置选中状态(立即重绘; 原生勾选不会自动失效重绘)。
// 同时清除不确定态。
func (c *Checkbox) SetChecked(v bool) *Checkbox {
	if c.IsCheck() == v && !c.indeterminate {
		return c
	}
	c.indeterminate = false
	c.SetCheck(v)
	c.Redraw(false)
	return c
}

// SetIndeterminate 设置不确定态。为 true 时清除选中(与 Radix 的
// checked="indeterminate" 等价), 指示器画减号; 为 false 时保持当前选中态。
func (c *Checkbox) SetIndeterminate(v bool) *Checkbox {
	if c.indeterminate == v {
		return c
	}
	c.indeterminate = v
	if v {
		c.SetCheck(false)
	}
	c.Redraw(false)
	return c
}

// SetInvalid 设置/清除校验失败态(见 WithCheckboxInvalid)。
func (c *Checkbox) SetInvalid(v bool) *Checkbox {
	if c.invalid == v {
		return c
	}
	c.invalid = v
	c.Redraw(false)
	return c
}

// Invalid 返回当前是否处于校验失败态。
func (c *Checkbox) Invalid() bool { return c.invalid }

// OnChange 注册选中状态变化回调(鼠标点击/空格切换时触发)。
func (c *Checkbox) OnChange(fn func(checked bool)) *Checkbox {
	c.changes = append(c.changes, fn)
	return c
}

// OnCheckedChange 注册"可否决"的选中变化回调(对应 Base UI 的 onCheckedChange):
// 在状态落地**之前**调用, 任一回调返回 false 即取消本次切换; 全部放行后状态
// 照常写入并继续触发 OnChange。鼠标、空格与 Toggle 走同一条路径。
// 程序化 SetChecked 不触发否决(同受控组件的 setState 语义)。
func (c *Checkbox) OnCheckedChange(fn func(checked bool) bool) *Checkbox {
	if fn != nil {
		c.vetoes = append(c.vetoes, fn)
	}
	return c
}

func (c *Checkbox) allowChange(next bool) bool {
	for _, fn := range c.vetoes {
		if !fn(next) {
			return false
		}
	}
	return true
}

// revertUserChange 回滚原生 Check 按钮已经写下的状态(否决/只读路径)。
// 不确定态在点击后会丢失, 回滚时要一并恢复。
func (c *Checkbox) revertUserChange(nativeChecked bool) {
	prevInd := c.indeterminate
	c.SetCheck(!nativeChecked)
	c.indeterminate = prevInd
	c.Redraw(false)
}

// Toggle 切换选中状态并触发 OnChange(与鼠标点击/空格等价), 供"整块区域可点"的
// 字段卡片使用; 禁用或只读时为空操作。不确定态点击后落到选中(同鼠标路径)。
func (c *Checkbox) Toggle() *Checkbox {
	if c == nil || !c.IsEnable() || c.readOnly {
		return c
	}
	c.toggleNotify()
	return c
}

// FitWidth 按当前文字重新量宽(仅 cx 传 0 创建的复选框)。
func (c *Checkbox) FitWidth() *Checkbox {
	if c == nil || !c.autoWidth {
		return c
	}
	c.SetWidth(checkboxNaturalWidth(xui.Current(), c.GetText(), c.BoxSize))
	c.Redraw(false)
	return c
}

// SetBackdrop 指定绘制开头铺的宿主底色(默认按"放在 Card 上"用 t.Card)。
// 组件是自绘的局部形状, 重绘不擦底, 必须先铺宿主底色; 放进有底色的容器
// (例如选中态 bg-accent 的卡片)时必须跟随, 否则元素矩形会留下一个异色块。
func (c *Checkbox) SetBackdrop(fn func(t *xui.Theme) xui.Color) *Checkbox {
	if c == nil {
		return c
	}
	c.backdrop = fn
	c.Redraw(false)
	return c
}

func (c *Checkbox) backdropColor(t *xui.Theme) xui.Color {
	if c.backdrop != nil {
		return c.backdrop(t)
	}
	return t.Card
}

func (c *Checkbox) notifyChange(checked bool) {
	for _, fn := range c.changes {
		fn(checked)
	}
}

// toggleNotify 供键盘路径使用: 先问否决回调, 放行后写状态并触发与鼠标点击相同的回调。
func (c *Checkbox) toggleNotify() {
	next := !c.IsCheck()
	if c.indeterminate {
		next = true
	}
	if !c.allowChange(next) {
		return
	}
	if c.indeterminate {
		c.indeterminate = false
	}
	c.SetCheck(next)
	c.Redraw(false)
	c.notifyChange(next)
}

func (c *Checkbox) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true // 文本要和指示器对齐, 全部自己画
	t := xui.Current()
	st := xui.StateOf(hEle)
	checked := c.IsCheck()
	active := checked || c.indeterminate

	// 先铺宿主底色(整元素): 透明元素重绘**不擦底**(见 DESIGN 第八节 28), 不铺的话
	// ①键盘焦点环消失后灰环留在画布上;②未选态在暗色下是半透明令牌(dark:bg-input/30),
	// 会透出上一个状态/比卡片更深的一块底色。组件按"放在 Card 上"设计(同 Table/NativeComboBox),
	// 放在异色容器上时用 SetBackdrop 跟随。
	host := c.backdropColor(t)
	p.FillRound(&p.Rc, host, 0)

	// 指示器(逻辑像素, 框架按 DPI 变换渲染)。有文字时贴左(4), 纯指示器在元素
	// 内居中 —— 后者默认元素 40×32, 正好给 after:-inset-x-3/-inset-y-2 的点击热区留白。
	size := c.BoxSize
	gap := int32(8)
	boxLeft := int32(4)
	if c.GetText() == "" {
		boxLeft = (p.W() - size) / 2
	}
	box := xc.RECT{
		Left:   boxLeft,
		Top:    (p.H() - size) / 2,
		Right:  boxLeft + size,
		Bottom: (p.H()-size)/2 + size,
	}

	disabled := st.Disabled || !c.IsEnable()
	// aria-invalid 的 ring 常驻(不要求焦点); focus-visible:ring-3 ring-ring/50 只在键盘焦点时画。
	// 两者同时成立时红色优先(变体顺序里 aria-invalid 在 focus-visible 之后)。
	invalid := c.invalid && !disabled
	focusRing := st.Focused && !disabled && !c.focusRingProxy && xui.FocusedByKeyboard()
	if invalid || focusRing {
		halo := box
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
		p.FillRound(&halo, ring, 4+3)
	}

	// 环带是实心填充, 未选中亮色没有自己的底色(fill=None), 不先盖一层宿主色的话
	// 红色/灰色环会从指示器内部透出来(看起来像整块被染色)。整元素已铺过宿主底色,
	// 这里只需盖住指示器矩形。
	p.FillRound(&box, host, 4)

	// 状态配色逐条对齐 checkbox.tsx:
	//   未选 = 无底色(亮色) / dark:bg-input/30(暗色) + border-input;
	//   选中/不确定 = bg-primary + border-primary, 勾/减号 = text-primary-foreground;
	//   禁用 = disabled:opacity-50 —— 组级透明, 每个图层各自再乘 50%(给颜色加 alpha 走
	//   画家混合, 与"整组半透明合成"等价; 见语义汇总表)。
	//
	// transition-colors: 选中/取消用运行时过渡值插值(键 1 = 选中态), 不要在组件里
	// 起定时器。纯指示器的首帧会从 0 淡入一次, 与浏览器首次挂载不过渡略有出入,
	// 但交互后的表现与 tailwind 一致。
	stateT := xui.HoverItem(hEle, 1, active)
	uncheckedFill := xui.None
	if t.IsDark() {
		uncheckedFill = t.Input.ScaleAlpha(0.30)
	}
	uncheckedBorder := t.Input
	if focusRing && !invalid {
		uncheckedBorder = t.Ring // focus-visible:border-ring
	}
	if invalid {
		uncheckedBorder = t.Destructive
		if t.IsDark() {
			uncheckedBorder = t.Destructive.WithAlpha(128) // dark:aria-invalid:border-destructive/50
		}
	}
	const disabledAlpha = 0.5
	fill := xui.Fade(uncheckedFill, t.Primary, stateT)
	border := xui.Fade(uncheckedBorder, t.Primary, stateT)
	if disabled {
		fill = fill.ScaleAlpha(disabledAlpha) // None 乘完仍是 None(亮色未选态即无填充)
		border = border.ScaleAlpha(disabledAlpha)
	}

	if !fill.IsNone() {
		p.FillRound(&box, fill, 4)
	}
	p.StrokeRound(&box, border, 4, 1)
	if stateT > 0.001 {
		alpha := byte(stateT * 255)
		if disabled {
			alpha = byte(float64(alpha) * disabledAlpha)
		}
		glyph := t.PrimaryForeground.WithAlpha(alpha)
		if c.indeterminate && active {
			// 勾号/减号内缩 1px, 避免线帽被指示器边缘裁掉
			drawMinus(p.Draw, box, glyph, 2)
		} else {
			drawCheck(p.Draw, box, glyph, 2)
		}
	}

	// 文本
	textRC := xc.RECT{
		Left:   box.Right + gap,
		Top:    p.Rc.Top,
		Right:  p.Rc.Right,
		Bottom: p.Rc.Bottom,
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	fore := t.Foreground
	if disabled {
		fore = t.MutedForeground
	}
	p.Text(c.GetText(), &textRC, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	return 0
}

// ===== FieldChoiceCard 适配(接口方法不导出, 只有内置控件能当选择卡片指示器) =====

func (c *Checkbox) choiceCardHandle() int                            { return c.Handle }
func (c *Checkbox) choiceCardToggle()                                { c.Toggle() }
func (c *Checkbox) choiceCardActive() bool                           { return c.Checked() }
func (c *Checkbox) choiceCardOnChange(fn func(bool))                 { c.OnChange(fn) }
func (c *Checkbox) choiceCardFocusProxy()                            { c.SetFocusRingProxy(true) }
func (c *Checkbox) choiceCardBackdrop(fn func(*xui.Theme) xui.Color) { c.SetBackdrop(fn) }
func (c *Checkbox) choiceCardSize() (int32, int32)                   { return 24, 24 }

// drawCheck 在指示器内绘制勾号。几何取 lucide Check 的 24 网格点
// (20,6)-(9,17)-(4,12): shadcn 的指示器是 16px 边框盒, 1px 边框内 14px 内容区
// 放 size-3.5(14px) 图标, 所以 24 网格映射到内缩 1px 的区域。
//
// 必须走 DrawLineF: DrawLine(int32) 会把 20/24*14 这类点位截断到左/上,
// 三条折线各丢 0~1 逻辑像素, 勾号整体偏左, 右臂也短一截。
func drawCheck(d *drawx.Draw, box xc.RECT, color xui.Color, stroke int32) {
	w, h := float32(box.Right-box.Left-2), float32(box.Bottom-box.Top-2)
	x, y := float32(box.Left+1), float32(box.Top+1)
	d.SetLineWidth(stroke)
	d.SetBrushColor(uint32(color))
	d.DrawLineF(x+w*20/24, y+h*6/24, x+w*9/24, y+h*17/24)
	d.DrawLineF(x+w*9/24, y+h*17/24, x+w*4/24, y+h*12/24)
}

// drawMinus 在指示器内绘制减号(不确定态), 线宽与勾号一致。
// 两端对称于指示器中心, 同样用 F 版, 否则截断会让减号偏左。
func drawMinus(d *drawx.Draw, box xc.RECT, color xui.Color, stroke int32) {
	w := float32(box.Right - box.Left)
	y := float32(box.Top+box.Bottom) / 2
	d.SetLineWidth(stroke)
	d.SetBrushColor(uint32(color))
	d.DrawLineF(float32(box.Left)+w*20/100, y, float32(box.Left)+w*80/100, y)
}
