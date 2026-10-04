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
	// sliderRingPad 是滑块 hover/active/focus-visible 时 ring-3 的宽度。
	sliderRingPad int32 = 3
	// sliderHalo 是滑块中心到 ring 外缘的距离(滑块半径 + ring 宽度)。
	// 元素左右各留出这么多: 元素自绘被裁剪在自身客户区内, 滑块位于轨道两端时
	// 需要这段余量 ring 才不会被切掉(见 docs/DESIGN.md 第八节 10、20)。
	sliderHalo int32 = sliderThumb/2 + sliderRingPad // 9
)

// Slider 纯自绘滑动条(不使用原生 SliderBar)。
//
// 之所以不用原生 SliderBar 做载体: 其滑块是原生 Button, 默认样式在 Stay/Down/
// Focus 态会画出方块背景(BkInfo), ClearBkInfo + EnableBkTransparent +
// EnableDrawBorder/DrawFocus + 空 paint 回调都压不住(实测悬浮出灰方块、
// 拖动后出白方块、点击聚焦后又一种), 且 EnableFocus(false) 会让按钮不再上报
// Stay 状态。改用普通元素 + 自实现交互后, 组件 100% 掌控绘制, 无原生残留。
//
// 与 shadcn slider.tsx(radix-nova) 对齐:
//   - 轨道 h-1 rounded-full bg-muted; 已完成区间 bg-primary;
//   - 滑块 size-3 rounded-full border-ring bg-white;
//   - 滑块 hover / 拖动中 / focus-visible 时显示 ring-3 ring-ring/50;
//   - 整体禁用 opacity-50; 支持点击跳转、拖动与左右方向键(±5%)。
//
// 几何: 与 radix 一致, 滑块**中心**在轨道两端之间移动(0 → 轨道左端, 100 → 轨道右端),
// 因此滑块会向轨道端点外沿伸出半个身位, ring 再向外扩 3px。
// 由于元素自绘无法越出自身客户区, 元素实际比轨道(cx)左右各宽出 sliderHalo(9px),
// 轨道绘制在元素内缩 sliderHalo 的位置——**视觉轨道宽度仍等于入参 cx**。
type Slider struct {
	*widget.Element
	cb    func(v float64)
	pos   int32 // 千分比 0..1000
	drag  bool
	hover bool
}

// NewSlider 创建滑动条。cx 是**可见轨道**宽度; cy 传 0 时使用滑块 + 焦点环所需的最小高度(18)。
func NewSlider(x, y, cx, cy int32, hParent int) *Slider {
	if cy <= 0 {
		cy = sliderThumb + sliderRingPad*2
	}
	if cx <= 0 {
		cx = 100
	}
	// 元素外扩 sliderHalo, 给两端的 ring 留出绘制空间(超出元素客户区的绘制会被裁掉)
	e := widget.NewElement(x-sliderHalo, y, cx+sliderHalo*2, cy, hParent)
	if e == nil {
		return nil
	}
	s := &Slider{Element: e}
	e.EnableFocus(true) // 可聚焦, 支持键盘方向键

	e.AddEvent_Paint(s.onPaint)
	e.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		s.hover = true
		e.Redraw(true)
		return 0
	})
	e.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		s.hover = false
		e.Redraw(true)
		return 0
	})
	e.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress() // 鼠标操作标记焦点来源, 保持 focus-visible 语义
		s.drag = true
		xc.XEle_SetCapture(hEle, true) // 捕获鼠标, 拖出元素范围也能继续
		s.setPosFromX(pPt.X)
		return 0
	})
	e.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if s.drag {
			s.setPosFromX(pPt.X)
		}
		return 0
	})
	e.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if s.drag {
			s.drag = false
			xc.XEle_SetCapture(hEle, false)
			e.Redraw(true)
		}
		return 0
	})
	// 键盘: 左右 ±5%, Home/End 到两端(与浏览器原生 range 一致)
	e.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		const (
			vkHome  = 0x24
			vkEnd   = 0x23
			vkLeft  = 0x25
			vkRight = 0x27
		)
		step := sliderRange / 20
		old := s.pos
		switch wParam {
		case vkLeft:
			s.pos -= step
		case vkRight:
			s.pos += step
		case vkHome:
			s.pos = 0
		case vkEnd:
			s.pos = sliderRange
		default:
			return 0
		}
		if s.pos < 0 {
			s.pos = 0
		} else if s.pos > sliderRange {
			s.pos = sliderRange
		}
		if s.pos != old {
			e.Redraw(true)
			if s.cb != nil {
				s.cb(s.Value())
			}
		}
		return 0
	})
	xui.AttachWindow(e.Handle)
	return s
}

// setPosFromX 把元素客户区内 x 坐标换算为 pos 并刷新。
func (s *Slider) setPosFromX(px int32) {
	trackW := s.GetWidth() - sliderHalo*2
	if trackW <= 0 {
		return
	}
	v := (px - sliderHalo) * sliderRange / trackW
	if v < 0 {
		v = 0
	} else if v > sliderRange {
		v = sliderRange
	}
	if v != s.pos {
		s.pos = v
		s.Redraw(true)
		if s.cb != nil {
			s.cb(s.Value())
		}
	}
}

// Value 返回当前值(0~100)。
func (s *Slider) Value() float64 {
	return float64(s.pos) / float64(sliderRange) * 100
}

// SetValue 设置当前值, v 会被夹取到 0~100。
func (s *Slider) SetValue(v float64) *Slider {
	if v < 0 {
		v = 0
	} else if v > 100 {
		v = 100
	}
	s.pos = int32(v/100*float64(sliderRange) + 0.5)
	s.Redraw(true)
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
	*pbHandled = true // 全部自绘
	t := xui.Current()
	st := xui.StateOf(hEle)

	// 先铺宿主底色: 本组件不整体填充(轨道/滑块都是局部形状), 若不铺底,
	// 上一帧的 ring/滑块像素会残留在画布上(实测 hover 离开/点击聚焦后灰环不消失)。
	// 默认用 Card(组件通常放在卡片上); 放在其他底色上请改窗口主题或自行调整。
	p.FillRound(&p.Rc, t.Card, 0)

	alpha := byte(255)
	if st.Disabled {
		alpha = 128 // data-disabled:opacity-50
	}

	// 视觉轨道: 元素内缩 sliderHalo(与滑块中心的活动范围一致)
	trackLeft := p.Rc.Left + sliderHalo
	trackRight := p.Rc.Right - sliderHalo
	trackW := trackRight - trackLeft
	if trackW < 0 {
		trackW = 0
	}
	cy := p.Rc.Top + p.H()/2
	trackY := cy - sliderTrackH/2

	// 轨道: h-1 rounded-full bg-muted
	p.FillRound(&xc.RECT{Left: trackLeft, Top: trackY, Right: trackRight, Bottom: trackY + sliderTrackH},
		t.Muted.WithAlpha(alpha), sliderTrackH/2)

	// 滑块中心 x: 0 → 轨道左端, range → 轨道右端
	center := trackLeft
	if sliderRange > 0 {
		center += int32(float64(trackW) * float64(s.pos) / float64(sliderRange))
	}

	// 已完成区间: 从轨道左端到滑块中心
	if center > trackLeft {
		p.FillRound(&xc.RECT{Left: trackLeft, Top: trackY, Right: center, Bottom: trackY + sliderTrackH},
			t.Primary.WithAlpha(alpha), sliderTrackH/2)
	}

	// 滑块 hover / 拖动中 / focus-visible → ring-3 ring-ring/50
	// (ring 半径 = sliderHalo, 两端刚好落在元素客户区边缘, 不再被裁剪)
	if !st.Disabled && (s.hover || s.drag || (st.Focused && xui.FocusedByKeyboard())) {
		p.FillRound(&xc.RECT{
			Left:   center - sliderHalo,
			Top:    cy - sliderHalo,
			Right:  center + sliderHalo,
			Bottom: cy + sliderHalo,
		}, t.Ring.WithAlpha(128), sliderHalo)
	}

	// 滑块本体: border-ring bg-white(shadcn 里滑块底色是字面量 white, 两个主题都一样)
	thumb := xc.RECT{
		Left:   center - sliderThumb/2,
		Top:    cy - sliderThumb/2,
		Right:  center + sliderThumb/2,
		Bottom: cy + sliderThumb/2,
	}
	p.Draw.SetLineWidth(1)
	// 滑块是"内容", 禁用(组级透明)下核心保持原色, 只有轨道/完成区间变淡
	p.Draw.SetBrushColor(uint32(xui.RGBA(255, 255, 255, 255)))
	p.Draw.FillEllipse(&thumb)
	p.Draw.SetBrushColor(uint32(t.Ring))
	p.Draw.DrawEllipse(&thumb)
	return 0
}
