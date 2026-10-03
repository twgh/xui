package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

// Progress 自绘进度条: 载体为原生 ProgressBar(提供 range/pos 与进度变化事件),
// 外观全部自绘。
//
// 与 shadcn progress.tsx(radix-nova) 对齐:
//   - 轨道: h-1(4px) w-full rounded-full bg-muted;
//   - 指示器: bg-primary, 宽度 = 轨道宽 x 百分比(overflow-hidden 由轨道圆角天然形成)。
type Progress struct {
	*widget.ProgressBar
}

// progressRange 是内部进度范围: 用千分比保留 0.1% 精度, 对外 API 仍是 0~100。
const progressRange int32 = 1000

// NewProgress 创建进度条。cy 传 0 时使用 shadcn 默认的 4px 高度。
func NewProgress(x, y, cx, cy int32, hParent int) *Progress {
	if cy <= 0 {
		cy = 4
	}
	pb := widget.NewProgressBar(x, y, cx, cy, hParent)
	if pb == nil {
		return nil
	}
	p := &Progress{ProgressBar: pb}
	pb.EnableBkTransparent(true) // 灭掉原生外观
	pb.EnableShowText(false)
	pb.SetRange(progressRange)
	pb.AddEvent_Paint(p.onPaint)
	pb.AddEvent_ProgressBar_Change(func(hEle int, pos int32, pbHandled *bool) int {
		pb.Redraw(false)
		return 0
	})
	xui.AttachWindow(pb.Handle)
	return p
}

// Value 返回当前进度(0~100)。
func (p *Progress) Value() float64 {
	return float64(p.GetPos()) / float64(progressRange) * 100
}

// SetValue 设置进度, v 会被夹取到 0~100。
func (p *Progress) SetValue(v float64) *Progress {
	if v < 0 {
		v = 0
	} else if v > 100 {
		v = 100
	}
	p.SetPos(int32(v/100*float64(progressRange) + 0.5))
	p.Redraw(false)
	return p
}

// OnChange 注册进度变化回调(v 为 0~100)。
func (p *Progress) OnChange(fn func(v float64)) *Progress {
	p.AddEvent_ProgressBar_Change(func(hEle int, pos int32, pbHandled *bool) int {
		fn(float64(pos) / float64(progressRange) * 100)
		return 0
	})
	return p
}

func (p *Progress) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	pt := xui.BeginPaint(hEle, hDraw)
	if pt == nil {
		return 0
	}
	*pbHandled = true // 轨道与指示器全部自绘
	t := xui.Current()

	h := pt.H()
	radius := h / 2

	// 轨道: bg-muted
	pt.FillRound(&pt.Rc, t.Muted, radius)

	// 指示器: bg-primary, 宽度按进度; 圆角随宽度收缩, 保证窄进度也是胶囊形
	w := int32(float64(pt.W()) * float64(p.GetPos()) / float64(progressRange))
	if w <= 0 {
		return 0
	}
	if r := w / 2; r < radius {
		radius = r
	}
	bar := xc.RECT{
		Left:   pt.Rc.Left,
		Top:    pt.Rc.Top,
		Right:  pt.Rc.Left + w,
		Bottom: pt.Rc.Bottom,
	}
	pt.FillRound(&bar, t.Primary, radius)
	return 0
}
