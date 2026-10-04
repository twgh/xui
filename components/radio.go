package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Radio 自绘单选按钮: 载体为 Button_Type_Radio 类型的原生按钮。
// 与 shadcn radio-group.tsx 对齐: 选中 = bg-primary 实心圆(含 border-primary)
// + 中心 size-2 的白色(bg-primary-foreground)圆点; focus-visible 环画在
// 指示器外圈, 键盘焦点才显示。
type Radio struct {
	*widget.Button
	BoxSize int32 // 圆形指示器直径(逻辑像素)
}

// NewRadio 创建单选按钮。cy 传 0 时使用默认行高 24。同组互斥由 xcgui 按父容器自动管理。
func NewRadio(x, y, cx, cy int32, text string, hParent int) *Radio {
	if cy <= 0 {
		cy = 24
	}
	btn := widget.NewButton(x, y, cx, cy, text, hParent)
	if btn == nil {
		return nil
	}
	r := &Radio{Button: btn, BoxSize: 16}
	btn.SetTypeEx(xcc.Button_Type_Radio)
	btn.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter)
	btn.EnableBkTransparent(true)
	btn.AddEvent_Paint(r.onPaint)
	xui.AttachWindow(r.Handle)
	r.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	return r
}

// Selected 返回选中状态。
func (r *Radio) Selected() bool { return r.IsCheck() }

// OnChange 注册选中状态变化回调。
func (r *Radio) OnChange(fn func(selected bool)) *Radio {
	r.AddEvent_Button_Check(func(hEle int, bCheck bool, pbHandled *bool) int {
		fn(bCheck)
		return 0
	})
	return r
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

	// 圆形指示器(逻辑像素, 框架按 DPI 变换渲染)
	size := r.BoxSize
	gap := int32(8)
	circle := xc.RECT{
		Left:   4,
		Top:    (p.H() - size) / 2,
		Right:  4 + size,
		Bottom: (p.H()-size)/2 + size,
	}

	// focus-visible:ring-3 ring-ring/50 画在指示器外圈(键盘焦点才显示)
	if st.Focused && !st.Disabled && xui.FocusedByKeyboard() {
		halo := circle
		halo.Left -= 3
		halo.Top -= 3
		halo.Right += 3
		halo.Bottom += 3
		p.Draw.SetBrushColor(uint32(t.Ring.WithAlpha(128)))
		p.Draw.FillEllipse(&halo)
	}

	// shadcn 源码(data-checked): 选中 = bg-primary 圆 + border-primary,
	// 指示器为 size-2 的 bg-primary-foreground 白色圆点(直径 50%)。
	var fill, border, dot xui.Color
	switch {
	case st.Disabled: // disabled:opacity-50
		a := byte(128)
		fill, border, dot = t.Background.WithAlpha(a), t.Input.WithAlpha(a), t.Input
	case checked:
		fill, border, dot = t.Primary, t.Primary, t.PrimaryFore
	default:
		fill, border, dot = t.Background, t.Input, t.Input
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
	// 文字保持原色: shadcn 里禁用只作用于圆点(RadioGroupItem 的 disabled:opacity-50),
	// Label 不跟随禁用(实测参考页禁用项文字核心仍接近纯黑), 只有圆点变淡。
	p.Text(r.GetText(), &textRC, t.Foreground, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	return 0
}
