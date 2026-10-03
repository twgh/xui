package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

const (
	// sliderRange 是内部范围: 千分比, 对外 API 是 0~100(保留 1 位小数)。
	sliderRange int32 = 1000
	// sliderThumb 是滑块直径, shadcn 为 size-3。
	sliderThumb int32 = 12
	// sliderTrackH 是轨道高度, shadcn 为 h-1。
	sliderTrackH int32 = 4
	// sliderRingPad 是滑块 hover/active/focus-visible 时 ring-3 的外扩量。
	sliderRingPad int32 = 3
)

// Slider 自绘滑动条: 载体为原生 SliderBar(拖动、点击跳转、键盘行为全部原生),
// 轨道、已完成区间与滑块由本组件绘制。
//
// 与 shadcn slider.tsx(radix-nova) 对齐:
//   - 轨道 h-1 rounded-full bg-muted; 已完成区间 bg-primary;
//   - 滑块 size-3 rounded-full border-ring bg-white;
//   - 滑块 hover / active / focus-visible 时显示 ring-3 ring-ring/50;
//   - 整体禁用 opacity-50。
//
// 滑块的绘制放在滑动条自身的绘制回调里(而不是滑块按钮的), 原因:
// 焦点环要画到滑块外 3px, 而滑块按钮的客户区只有 12x12, 画在它自己回调里会被裁掉;
// 滑动条客户区高度 = 12 + 3x2, 正好容纳。
type Slider struct {
	*widget.SliderBar
	thumb *widget.Button
	cb    func(v float64)
}

// NewSlider 创建滑动条。cy 传 0 时使用滑块 + 焦点环所需的最小高度(18)。
func NewSlider(x, y, cx, cy int32, hParent int) *Slider {
	if cy <= 0 {
		cy = sliderThumb + sliderRingPad*2
	}
	sb := widget.NewSliderBar(x, y, cx, cy, hParent)
	if sb == nil {
		return nil
	}
	s := &Slider{SliderBar: sb}
	sb.SetRange(sliderRange)
	sb.SetButtonWidth(sliderThumb)
	sb.SetButtonHeight(sliderThumb)
	sb.EnableBkTransparent(true) // 灭掉原生轨道
	sb.AddEvent_Paint(s.onPaint)

	// 原生滑块按钮只保留行为(拖动/命中), 外观交给 onPaint
	s.thumb = sb.GetButtonObj()
	if s.thumb != nil {
		s.thumb.EnableBkTransparent(true)
		// 原生按钮在 hover/焦点态还会画边框与焦点矩形, 滑块上会冒出一个方块, 一并关掉
		s.thumb.EnableDrawBorder(false).EnableDrawFocus(false)
		s.thumb.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
			*pbHandled = true
			return 0
		})
	}
	sb.AddEvent_SliderBar_Change(func(hEle int, pos int32, pbHandled *bool) int {
		sb.Redraw(false)
		if s.cb != nil {
			s.cb(float64(pos) / float64(sliderRange) * 100)
		}
		return 0
	})
	xui.AttachWindow(sb.Handle)
	return s
}

// Value 返回当前值(0~100)。
func (s *Slider) Value() float64 {
	return float64(s.GetPos()) / float64(sliderRange) * 100
}

// SetValue 设置当前值, v 会被夹取到 0~100。
func (s *Slider) SetValue(v float64) *Slider {
	if v < 0 {
		v = 0
	} else if v > 100 {
		v = 100
	}
	s.SetPos(int32(v/100*float64(sliderRange) + 0.5))
	s.Redraw(false)
	return s
}

// OnChange 注册值变化回调(v 为 0~100), 拖动过程中会连续触发。
func (s *Slider) OnChange(fn func(v float64)) *Slider {
	s.cb = fn
	return s
}

func (s *Slider) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true // 轨道与滑块全部自绘
	t := xui.Current()
	st := xui.StateOf(hEle)

	alpha := byte(255)
	if st.Disabled {
		alpha = 128 // data-disabled:opacity-50
	}

	// 轨道: 垂直居中
	trackY := p.Rc.Top + (p.H()-sliderTrackH)/2
	track := xc.RECT{Left: p.Rc.Left, Top: trackY, Right: p.Rc.Right, Bottom: trackY + sliderTrackH}
	p.FillRound(&track, t.Muted.WithAlpha(alpha), sliderTrackH/2)

	// 滑块矩形直接取原生滑块按钮的矩形: 与拖动行为严格一致
	thumb := xc.RECT{Left: p.Rc.Left + sliderThumb/2}
	if s.thumb != nil {
		xc.XEle_GetRect(s.thumb.Handle, &thumb)
	}
	center := (thumb.Left + thumb.Right) / 2

	// 已完成区间: 从轨道左端到滑块中心
	if center > track.Left {
		bar := xc.RECT{Left: track.Left, Top: track.Top, Right: center, Bottom: track.Bottom}
		p.FillRound(&bar, t.Primary.WithAlpha(alpha), sliderTrackH/2)
	}

	// 滑块 hover / active / focus-visible → ring-3 ring-ring/50
	if !st.Disabled && s.thumb != nil {
		ts := xui.StateOf(s.thumb.Handle)
		if ts.Hover || ts.Pressed || (ts.Focused && xui.FocusedByKeyboard()) {
			halo := xc.RECT{
				Left:   thumb.Left - sliderRingPad,
				Top:    thumb.Top - sliderRingPad,
				Right:  thumb.Right + sliderRingPad,
				Bottom: thumb.Bottom + sliderRingPad,
			}
			p.FillRound(&halo, t.Ring.WithAlpha(128), (sliderThumb+sliderRingPad*2)/2)
		}
	}

	// 滑块本体: border-ring bg-white(shadcn 里滑块底色是字面量 white, 两个主题都一样)
	p.Draw.SetLineWidth(1)
	p.Draw.SetBrushColor(uint32(xui.RGBA(255, 255, 255, 255).WithAlpha(alpha)))
	p.Draw.FillEllipse(&thumb)
	p.Draw.SetBrushColor(uint32(t.Ring.WithAlpha(alpha)))
	p.Draw.DrawEllipse(&thumb)
	return 0
}
