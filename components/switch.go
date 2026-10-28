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
//   - 元素绘制被裁剪在自身客户区内, focus-visible / aria-invalid 的 ring-3 需要 3px 绘制空间;
//   - 顺带把点击热区外扩, 与 shadcn 的 after:-inset-x-3 after:-inset-y-2 等效。
const switchRingPad int32 = 3

// Switch 自绘开关: 载体为 Button_Type_Check 类型的原生按钮(与 Checkbox 相同的行为层),
// 轨道与滑块全部由本组件绘制。
//
// 与 shadcn switch.tsx(radix-nova) 对齐:
//   - 轨道 rounded-full, 选中 bg-primary, 未选 bg-input(dark:bg-input/80), 禁用 opacity-50;
//   - 滑块 rounded-full bg-background, 选中时右移(两侧各留 1px);
//   - focus-visible: 边框变 Ring 色 + ring-3 ring-ring/50(鼠标点击不显示);
//   - aria-invalid: 边框变 destructive + ring-3 destructive/20(暗色 border /50、ring /40),
//     常驻不要求焦点; 与 focus-visible 同时成立时红色优先;
//   - 空格键切换(Radix 的 role=switch 语义, 原生 Check 按钮不响应空格, 见 DESIGN 第八节 108)。
type Switch struct {
	*widget.Button
	size    SwitchSize
	invalid bool
	disabled bool
	changes []func(checked bool)
}

// SwitchOption 开关配置项。
type SwitchOption func(*Switch)

// WithSwitchSize 设置尺寸预设。
func WithSwitchSize(s SwitchSize) SwitchOption { return func(sw *Switch) { sw.size = s } }

// WithSwitchChecked 设置初始选中状态。
func WithSwitchChecked(v bool) SwitchOption { return func(sw *Switch) { sw.SetCheck(v) } }

// WithSwitchDisabled 初始即禁用(不可点击/聚焦, 整体 opacity-50)。
func WithSwitchDisabled() SwitchOption { return func(sw *Switch) { sw.disabled = true } }

// WithSwitchInvalid 初始即为校验失败态(对应 shadcn 的 aria-invalid):
// 轨道描边 destructive + ring-3 destructive/20(暗色 /50、/40), 常驻不要求焦点。
func WithSwitchInvalid() SwitchOption { return func(sw *Switch) { sw.invalid = true } }

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
	// 原生勾选不会自动失效重绘; 鼠标点击经 XE_BUTTON_CHECK 补重绘并通知监听者。
	sw.AddEvent_Button_Check(func(hEle int, bCheck bool, pbHandled *bool) int {
		sw.Redraw(false)
		sw.notifyChange(bCheck)
		return 0
	})
	// 原生 Check 按钮不处理空格(实测 KeyDown 能收到、状态不变), Radix 的 role=switch
	// 用空格切换, 键盘激活自己补(同 Checkbox/Radio/Toggle, 见 DESIGN 第八节 108)。
	sw.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		const vkSpace = 0x20
		if wParam == vkSpace && sw.IsEnable() {
			sw.SetCheck(!sw.IsCheck())
			sw.Redraw(false)
			sw.notifyChange(sw.IsCheck())
			*pbHandled = true
		}
		return 0
	})
	if sw.disabled {
		sw.disabled = false // 交给 SetDisabled 走统一路径
		sw.SetDisabled(true)
	}
	return sw
}

// Checked 返回选中状态。
func (s *Switch) Checked() bool { return s.IsCheck() }

// SetChecked 设置选中状态(不触发 OnChange)。
func (s *Switch) SetChecked(v bool) *Switch {
	s.SetCheck(v)
	s.Redraw(false)
	return s
}

// Toggle 切换选中状态并触发 OnChange(与点击/空格同一路径)。
func (s *Switch) Toggle() *Switch {
	if s.disabled {
		return s
	}
	s.SetCheck(!s.IsCheck())
	s.Redraw(false)
	s.notifyChange(s.IsCheck())
	return s
}

// SetDisabled 设置禁用态(对应 radix 的 disabled): 不可点击、不可聚焦, 轨道与滑块一起乘 50%。
func (s *Switch) SetDisabled(v bool) *Switch {
	if s.disabled == v {
		return s
	}
	s.disabled = v
	s.Enable(!v)
	s.EnableFocus(!v)
	s.Redraw(false)
	return s
}

// Disabled 返回是否禁用。
func (s *Switch) Disabled() bool { return s.disabled }

// SetInvalid 设置/清除校验失败态(见 WithSwitchInvalid)。
func (s *Switch) SetInvalid(v bool) *Switch {
	if s.invalid == v {
		return s
	}
	s.invalid = v
	s.Redraw(false)
	return s
}

// Invalid 返回当前是否处于校验失败态。
func (s *Switch) Invalid() bool { return s.invalid }

// OnChange 注册选中状态变化回调(鼠标点击/空格切换时触发)。
func (s *Switch) OnChange(fn func(checked bool)) *Switch {
	if fn == nil {
		return s
	}
	s.changes = append(s.changes, fn)
	return s
}

func (s *Switch) notifyChange(checked bool) {
	for _, fn := range s.changes {
		fn(checked)
	}
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
	disabled := s.disabled || st.Disabled

	// 先铺宿主底色(整元素): 透明元素重绘**不擦底**(见 DESIGN 第八节 28) —— 暗色下
	// 未选轨道是半透明令牌(dark:data-unchecked:bg-input/80), 不铺底会透出上一帧的实心
	// 轨道/滑块, 键盘焦点环消失后也会把灰环留在画布上。组件按"放在 Card 上"设计。
	p.FillRound(&p.Rc, t.Card, 0)

	// 轨道在元素内居中(四周留出 switchRingPad 给焦点环/校验环)
	spec := switchSpecs[s.size]
	track := xc.RECT{
		Left:   p.Rc.Left + (p.W()-spec.w)/2,
		Top:    p.Rc.Top + (p.H()-spec.h)/2,
		Right:  p.Rc.Left + (p.W()-spec.w)/2 + spec.w,
		Bottom: p.Rc.Top + (p.H()-spec.h)/2 + spec.h,
	}
	radius := spec.h / 2

	alpha := 1.0
	if disabled {
		alpha = 0.5 // disabled:opacity-50
	}
	invalid := s.invalid && !disabled
	focusRing := st.Focused && !disabled && xui.FocusedByKeyboard()

	// aria-invalid:ring-3 ring-destructive/20(dark /40) / focus-visible:ring-3 ring-ring/50
	if invalid || focusRing {
		halo := xc.RECT{
			Left:   track.Left - switchRingPad,
			Top:    track.Top - switchRingPad,
			Right:  track.Right + switchRingPad,
			Bottom: track.Bottom + switchRingPad,
		}
		ring := t.Ring.WithAlpha(128)
		if invalid {
			ring = t.Destructive.WithAlpha(51)
			if t.IsDark() {
				ring = t.Destructive.WithAlpha(102)
			}
		}
		p.FillRound(&halo, ring, radius+switchRingPad)
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
	// aria-invalid:border-destructive / focus-visible:border-ring(坑: 轨道 border 默认 transparent)
	if invalid {
		border := t.Destructive
		if t.IsDark() {
			border = t.Destructive.WithAlpha(128) // dark:aria-invalid:border-destructive/50
		}
		p.StrokeRound(&track, border, radius, 1)
	} else if focusRing {
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
	if disabled {
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
