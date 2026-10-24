package ui

import (
	"math"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
)

// SkeletonOption 骨架屏配置项。
type SkeletonOption func(*Skeleton)

// WithSkeletonRadius 覆盖圆角(逻辑像素), 默认按主题的 rounded-md(RadiusMd)。
// 传 0 = 直角。
func WithSkeletonRadius(radius int32) SkeletonOption {
	return func(s *Skeleton) { s.radius = radius }
}

// WithSkeletonCircle 画成圆形(头像占位), 圆角取短边的一半。
func WithSkeletonCircle() SkeletonOption {
	return func(s *Skeleton) { s.circle = true }
}

// WithSkeletonOnBackground 按放在窗口底色(Background)上计算脉冲, 默认按 Card 计算。
func WithSkeletonOnBackground() SkeletonOption {
	return func(s *Skeleton) { s.onBackground = true }
}

// WithSkeletonStatic 关闭脉冲动画(静态占位)。
func WithSkeletonStatic() SkeletonOption {
	return func(s *Skeleton) { s.pulse = false }
}

// Skeleton 骨架屏占位块: 圆角矩形 + bg-accent + 2s 脉冲(animate-pulse)。
//
// 尺寸完全由调用方给(宽高是设计稿常量), 常用组合是头像用 WithSkeletonCircle,
// 文本行走布局里固定宽高。脉冲走运行时 xui.Pulse, 不要自己起定时器;
// SetAnimationScale(0) 会冻结脉冲, 便于截图对比。
//
// 颜色按"放在 Card 上"设计: 脉冲是**不透明**的 Accent 与 Card 混色(元素重绘
// 不会擦掉上一帧, 直接用半透明色叠加会越叠越亮, 见 DESIGN 第八节 28)。
// 放在窗口底色上时加 WithSkeletonOnBackground。
type Skeleton struct {
	*widget.Element
	radius       int32 // -1 = 跟随主题 RadiusMd
	circle       bool
	onBackground bool
	pulse        bool
}

// NewSkeleton 创建骨架屏块。宽高按设计稿传入(放进布局时可用布局项声明覆盖)。
func NewSkeleton(x, y, cx, cy int32, hParent int, opts ...SkeletonOption) *Skeleton {
	s := &Skeleton{radius: -1, pulse: true}
	for _, o := range opts {
		o(s)
	}
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	s.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.AddEvent_Paint(s.onPaint)
	xui.AttachWindow(ele.Handle)
	return s
}

// SetPulse 打开/关闭脉冲。关闭后立刻恢复实色。
func (s *Skeleton) SetPulse(pulse bool) *Skeleton {
	if s == nil || s.pulse == pulse {
		return s
	}
	s.pulse = pulse
	s.Redraw(false)
	return s
}

func (s *Skeleton) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()

	// opacity: 1 → 0.5 → 1, 与 animate-pulse 的关键帧一致。
	opacity := 1.0
	if s.pulse {
		phase := xui.Pulse(hEle, true)
		opacity = 0.75 + 0.25*math.Cos(2*math.Pi*phase)
	}
	base := t.Card
	if s.onBackground {
		base = t.Background
	}

	radius := s.radius
	if s.circle {
		side := p.W()
		if p.H() < side {
			side = p.H()
		}
		radius = side / 2
	} else if radius < 0 {
		radius = t.RadiusMd()
	}
	p.FillRound(&p.Rc, base.Mix(t.Accent, opacity), radius)
	return 0
}
