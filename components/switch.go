package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// SwitchSize 开关尺寸预设, 对应 shadcn switch.tsx 的 size 属性。
type SwitchSize int

const (
	SwitchDefault SwitchSize = iota // 轨道 32x18, 滑块 16
	SwitchSm                        // 轨道 24x14, 滑块 12
)

var switchSpecs = map[SwitchSize]struct{ w, h, thumb int32 }{
	SwitchDefault: {w: 32, h: 18, thumb: 16}, // data-[size=default]:h-[18.4px] w-[32px], thumb size-4
	SwitchSm:      {w: 24, h: 14, thumb: 12}, // data-[size=sm]:h-[14px] w-[24px], thumb size-3
}

// switchRingPad 是元素比轨道大出来的量(每边):
//   - 元素绘制被裁剪在自身客户区内, focus-visible 的 ring-3 需要 3px 绘制空间;
//   - 顺带把点击热区外扩, 与 shadcn 的 after:-inset-x-3 after:-inset-y-2 等效。
const switchRingPad int32 = 3

// Switch 自绘开关: 载体为 Button_Type_Check 类型的原生按钮(与 Checkbox 相同的行为层),
// 轨道与滑块全部由本组件绘制。
//
// 与 shadcn switch.tsx(radix-nova) 对齐:
//   - 轨道 rounded-full, 选中 bg-primary, 未选 bg-input, 禁用 opacity-50;
//   - 滑块 rounded-full bg-background, 选中时右移(两侧各留 1px);
//   - focus-visible: 边框变 Ring 色 + ring-3 ring-ring/50(鼠标点击不显示)。
type Switch struct {
	*widget.Button
	size SwitchSize
}

// SwitchOption 开关配置项。
type SwitchOption func(*Switch)

// WithSwitchSize 设置尺寸预设。
func WithSwitchSize(s SwitchSize) SwitchOption { return func(sw *Switch) { sw.size = s } }

// WithSwitchChecked 设置初始选中状态。
func WithSwitchChecked(v bool) SwitchOption { return func(sw *Switch) { sw.SetCheck(v) } }

// NewSwitch 创建开关。元素尺寸 = 轨道 + 每边 switchRingPad。
func NewSwitch(x, y int32, hParent int, opts ...SwitchOption) *Switch {
	sw := &Switch{Button: widget.NewButton(x, y, 0, 0, "", hParent)}
	if sw.Button == nil {
		return nil
	}
	for _, o := range opts {
		o(sw)
	}
	spec := switchSpecs[sw.size]
	sw.SetRect(&xc.RECT{
		Left:   x,
		Top:    y,
		Right:  x + spec.w + switchRingPad*2,
		Bottom: y + spec.h + switchRingPad*2,
	}, false, xcc.AdjustLayout_No)

	sw.SetTypeEx(xcc.Button_Type_Check)
	sw.EnableBkTransparent(true) // 灭掉原生外观, 交给 onPaint
	sw.AddEvent_Paint(sw.onPaint)
	xui.AttachWindow(sw.Handle)
	sw.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	return sw
}

// Checked 返回选中状态。
func (s *Switch) Checked() bool { return s.IsCheck() }

// SetChecked 设置选中状态。
func (s *Switch) SetChecked(v bool) *Switch {
	s.SetCheck(v)
	s.Redraw(false)
	return s
}

// OnChange 注册选中状态变化回调。
func (s *Switch) OnChange(fn func(checked bool)) *Switch {
	s.AddEvent_Button_Check(func(hEle int, bCheck bool, pbHandled *bool) int {
		fn(bCheck)
		return 0
	})
	return s
}

func (s *Switch) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true // 轨道与滑块全部自绘
	t := xui.Current()
	st := xui.StateOf(hEle)
	checked := s.IsCheck()

	// 先铺宿主底色(整元素): 透明元素重绘**不擦底**(见 DESIGN 第八节 28) —— 暗色下
	// 未选轨道是半透明令牌(dark:data-unchecked:bg-input/80), 不铺底会透出上一帧的实心
	// 轨道/滑块, 键盘焦点环消失后也会把灰环留在画布上。组件按"放在 Card 上"设计。
	p.FillRound(&p.Rc, t.Card, 0)

	// 轨道在元素内居中(四周留出 switchRingPad 给焦点环)
	spec := switchSpecs[s.size]
	track := xc.RECT{
		Left:   p.Rc.Left + (p.W()-spec.w)/2,
		Top:    p.Rc.Top + (p.H()-spec.h)/2,
		Right:  p.Rc.Left + (p.W()-spec.w)/2 + spec.w,
		Bottom: p.Rc.Top + (p.H()-spec.h)/2 + spec.h,
	}
	radius := spec.h / 2

	alpha := 1.0
	if st.Disabled {
		alpha = 0.5 // disabled:opacity-50
	}
	focusRing := st.Focused && !st.Disabled && xui.FocusedByKeyboard()

	// focus-visible:ring-3 ring-ring/50
	if focusRing {
		halo := xc.RECT{
			Left:   track.Left - switchRingPad,
			Top:    track.Top - switchRingPad,
			Right:  track.Right + switchRingPad,
			Bottom: track.Bottom + switchRingPad,
		}
		p.FillRound(&halo, t.Ring.WithAlpha(128), radius+switchRingPad)
	}

	// 轨道: data-checked:bg-primary / data-unchecked:bg-input(dark 下 input/80)
	trackColor := t.Input
	if t.IsDark() {
		trackColor = t.Input.ScaleAlpha(0.8)
	}
	if checked {
		trackColor = t.Primary
	}
	// 暗色 input 令牌自带 alpha(白 15%), 禁用是"再乘 50%"而不是覆盖(实测覆盖会变成
	// 不透明白色, 未选开关看着像一块白饼)
	p.FillRound(&track, trackColor.ScaleAlpha(alpha), radius)
	if focusRing {
		// focus-visible:border-ring
		p.StrokeRound(&track, t.Ring, radius, 1)
	}

	// 滑块: 选中右移, 两侧各留 1px
	const inset = int32(1)
	thumb := spec.thumb
	x := track.Left + inset
	if checked {
		x = track.Right - thumb - inset
	}
	y := track.Top + (spec.h-thumb)/2
	trc := xc.RECT{Left: x, Top: y, Right: x + thumb, Bottom: y + thumb}
	// 滑块跟随组级透明: disabled:opacity-50 是作用在整个开关上的, 滑块同样要变淡
	// (只淡轨道的话, 暗色下禁用开关会剩一颗扎眼的全亮白圆)
	thumbColor := s.thumbColor(t, checked)
	if st.Disabled {
		thumbColor = thumbColor.ScaleAlpha(alpha)
	}
	p.Draw.SetBrushColor(uint32(thumbColor))
	p.Draw.FillEllipse(&trc)
	return 0
}

// thumbColor 按 shadcn 的滑块配色规则取色(switch.tsx 的 thumb 类):
// 亮色主题固定 bg-background; 暗色主题选中 primary-foreground、未选 foreground。
func (s *Switch) thumbColor(t *xui.Theme, checked bool) xui.Color {
	if !t.IsDark() {
		return t.Background
	}
	if checked {
		return t.PrimaryForeground
	}
	return t.Foreground
}
