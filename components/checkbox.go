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
	backdrop      func(t *xui.Theme) xui.Color
	changes       []func(checked bool)
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

// NewCheckbox 创建复选框。cy 传 0 时使用默认行高 24。
func NewCheckbox(x, y, cx, cy int32, text string, hParent int, opts ...CheckboxOption) *Checkbox {
	if cy <= 0 {
		cy = 24
	}
	btn := widget.NewButton(x, y, cx, cy, text, hParent)
	if btn == nil {
		return nil
	}
	c := &Checkbox{Button: btn, BoxSize: 16}
	for _, o := range opts {
		o(c)
	}
	btn.SetTypeEx(xcc.Button_Type_Check)
	btn.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter)
	btn.EnableBkTransparent(true)
	btn.AddEvent_Paint(c.onPaint)
	xui.AttachWindow(c.Handle)
	c.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	// 原生勾选不会自动失效重绘; 鼠标点击经 XE_BUTTON_CHECK 补重绘并通知监听者。
	// 不确定态只是显示态: 用户点击后按 Radix 语义落到"选中"。
	c.AddEvent_Button_Check(func(hEle int, bCheck bool, pbHandled *bool) int {
		if bCheck {
			c.indeterminate = false
		}
		c.Redraw(false)
		c.notifyChange(bCheck)
		return 0
	})
	// 原生 Check 按钮不处理空格(实测 KeyDown 能收到、状态不变), Radix 的
	// role=checkbox 用空格切换, 键盘激活自己补。
	c.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		const vkSpace = 0x20
		if wParam == vkSpace && c.IsEnable() {
			c.toggleNotify()
			*pbHandled = true
		}
		return 0
	})
	return c
}

// Checked 返回选中状态(不确定态返回 false)。
func (c *Checkbox) Checked() bool { return c.IsCheck() }

// Indeterminate 返回是否处于不确定态。
func (c *Checkbox) Indeterminate() bool { return c.indeterminate }

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

// Toggle 切换选中状态并触发 OnChange(与鼠标点击/空格等价), 供"整块区域可点"的
// 字段卡片使用; 禁用时为空操作。不确定态点击后落到选中(同鼠标路径)。
func (c *Checkbox) Toggle() *Checkbox {
	if c == nil || !c.IsEnable() {
		return c
	}
	c.toggleNotify()
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

// toggleNotify 供键盘路径使用: 写状态并触发与鼠标点击相同的回调。
func (c *Checkbox) toggleNotify() {
	if c.indeterminate {
		c.indeterminate = false
		c.SetCheck(true)
	} else {
		c.SetCheck(!c.IsCheck())
	}
	c.Redraw(false)
	c.notifyChange(c.IsCheck())
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

	// 指示器(逻辑像素, 框架按 DPI 变换渲染)
	size := c.BoxSize
	gap := int32(8)
	box := xc.RECT{
		Left:   4,
		Top:    (p.H() - size) / 2,
		Right:  4 + size,
		Bottom: (p.H()-size)/2 + size,
	}

	// aria-invalid 的 ring 常驻(不要求焦点); focus-visible:ring-3 ring-ring/50 只在键盘焦点时画。
	// 两者同时成立时红色优先(变体顺序里 aria-invalid 在 focus-visible 之后)。
	invalid := c.invalid && !st.Disabled
	focusRing := st.Focused && !st.Disabled && xui.FocusedByKeyboard()
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
	const disabledAlpha = 0.5
	var fill, border, glyph xui.Color
	switch {
	case active:
		fill, border, glyph = t.Primary, t.Primary, t.PrimaryForeground
	default:
		border = t.Input
		if t.IsDark() {
			fill = t.Input.ScaleAlpha(0.30)
		}
		// focus-visible:border-ring; 选中态由 data-checked 的 border-primary 覆盖。
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
	if st.Disabled {
		fill = fill.ScaleAlpha(disabledAlpha) // None 乘完仍是 None(亮色未选态即无填充)
		border = border.ScaleAlpha(disabledAlpha)
		glyph = glyph.ScaleAlpha(disabledAlpha)
	}

	if !fill.IsNone() {
		p.FillRound(&box, fill, 4)
	}
	p.StrokeRound(&box, border, 4, 1)
	if active {
		if c.indeterminate {
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
	if st.Disabled {
		fore = t.MutedForeground
	}
	p.Text(c.GetText(), &textRC, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	return 0
}

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
