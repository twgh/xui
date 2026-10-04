package ui

import (
	"github.com/twgh/xcgui/drawx"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Checkbox 自绘复选框: 载体为 Button_Type_Check 类型的原生按钮,
// 左侧绘制圆角方框与勾号, 文本由本组件绘制(需要与指示器对齐排版)。
//
// 与 shadcn checkbox.tsx 对齐: 指示器 size-4 rounded-[4px] border-input,
// 选中 bg-primary border-primary 白勾, focus-visible 环画在指示器外圈
// (行内有边距, 不受元素裁剪), 键盘焦点才显示。
type Checkbox struct {
	*widget.Button
	BoxSize int32 // 指示器边长(逻辑像素), shadcn 为 size-4
}

// NewCheckbox 创建复选框。cy 传 0 时使用默认行高 24。
func NewCheckbox(x, y, cx, cy int32, text string, hParent int) *Checkbox {
	if cy <= 0 {
		cy = 24
	}
	btn := widget.NewButton(x, y, cx, cy, text, hParent)
	if btn == nil {
		return nil
	}
	c := &Checkbox{Button: btn, BoxSize: 16}
	btn.SetTypeEx(xcc.Button_Type_Check)
	btn.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter)
	btn.EnableBkTransparent(true)
	btn.AddEvent_Paint(c.onPaint)
	xui.AttachWindow(c.Handle)
	c.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	return c
}

// Checked 返回选中状态。
func (c *Checkbox) Checked() bool { return c.IsCheck() }

// SetChecked 设置选中状态。
func (c *Checkbox) SetChecked(v bool) *Checkbox {
	c.SetCheck(v)
	return c
}

// OnCheck 注册选中状态变化回调。
func (c *Checkbox) OnChange(fn func(checked bool)) *Checkbox {
	c.AddEvent_Button_Check(func(hEle int, bCheck bool, pbHandled *bool) int {
		fn(bCheck)
		return 0
	})
	return c
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

	// 指示器(逻辑像素, 框架按 DPI 变换渲染)
	size := c.BoxSize
	gap := int32(8)
	box := xc.RECT{
		Left:   4,
		Top:    (p.H() - size) / 2,
		Right:  4 + size,
		Bottom: (p.H()-size)/2 + size,
	}

	// focus-visible:ring-3 ring-ring/50 画在指示器外圈(键盘焦点才显示)
	if st.Focused && !st.Disabled && xui.FocusedByKeyboard() {
		halo := box
		halo.Left -= 3
		halo.Top -= 3
		halo.Right += 3
		halo.Bottom += 3
		p.FillRound(&halo, t.Ring.WithAlpha(128), 4+3)
	}

	var fill, border, check xui.Color
	switch {
	case st.Disabled: // disabled:opacity-50 —— 组级透明, 壳体变淡, 勾保持原色(与按钮文字同理)
		a := byte(128)
		if checked {
			fill, border, check = t.Primary.WithAlpha(a), t.Primary.WithAlpha(a), t.PrimaryFore
		} else {
			fill, border, check = t.Background.WithAlpha(a), t.Input.WithAlpha(a), xui.None
		}
	case checked: // data-checked:bg-primary border-primary
		fill, border, check = t.Primary, t.Primary, t.PrimaryFore
	default: // border-input
		fill, border = t.Background, t.Input
	}

	p.FillRound(&box, fill, 4)
	p.StrokeRound(&box, border, 4, 1)
	if checked {
		// 勾号内缩 1px, 避免线帽被指示器边缘裁掉
		drawCheck(p.Draw, box, check, 2)
	}

	// 文本
	textRC := xc.RECT{
		Left:   box.Right + gap,
		Top:    p.Rc.Top,
		Right:  p.Rc.Right,
		Bottom: p.Rc.Bottom,
	}
	if f := xui.FontHandle(xui.FontSpec{
		Family: t.FontDefault.Family,
		Size:   xui.FontPx(hEle, t.FontDefault.Size),
	}); f != 0 {
		p.Draw.SetFont(f)
	}
	fore := t.Foreground
	if st.Disabled {
		fore = t.MutedFore
	}
	p.Text(c.GetText(), &textRC, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	return 0
}

// drawCheck 在指示器内绘制勾号, 采用 shadcn 勾号的经典比例。
func drawCheck(d *drawx.Draw, box xc.RECT, color xui.Color, stroke int32) {
	w, h := box.Right-box.Left, box.Bottom-box.Top
	d.SetLineWidth(stroke)
	d.SetBrushColor(uint32(color))
	d.DrawLine(box.Left+w*22/100, box.Top+h*52/100, box.Left+w*42/100, box.Top+h*74/100)
	d.DrawLine(box.Left+w*42/100, box.Top+h*74/100, box.Left+w*80/100, box.Top+h*28/100)
}
