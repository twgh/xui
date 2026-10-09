package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
)

// Separator 分隔线: 载体为原生 Element, 绘制一条主题色横线。
// 与 shadcn separator.tsx 对齐: bg-border, 高 1px。
type Separator struct {
	*widget.Element
}

// NewSeparator 创建水平分隔线, h 固定为 1 逻辑像素。
func NewSeparator(x, y, cx int32, hParent int) *Separator {
	ele := widget.NewElement(x, y, cx, 1, hParent)
	if ele == nil {
		return nil
	}
	s := &Separator{Element: ele}
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(s.onPaint)
	xui.AttachWindow(ele.Handle)
	return s
}

func (s *Separator) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	p.Line(p.Rc.Left, p.Rc.Top, p.Rc.Right, p.Rc.Top, t.Border, 1)
	return 0
}

// BadgeVariant 徽章变体。
type BadgeVariant int

const (
	BadgeDefault     BadgeVariant = iota // 实心主色
	BadgeSecondary                       // 弱化填充
	BadgeOutline                         // 边框
	BadgeDestructive                     // 破坏性(软色调)
)

// Badge 自绘徽章: 载体为原生 Element + 手绘文字, 静态无交互。
// 与 shadcn badge.tsx 对齐: h-5, rounded-4xl(胶囊), text-xs;
// destructive 为软色调(bg-destructive/10 text-destructive)。
type Badge struct {
	*widget.Element
	variant BadgeVariant
	text    string
}

// NewBadge 创建徽章。cy 可传 0 使用默认高度 20。
func NewBadge(x, y, cx, cy int32, text string, hParent int, opts ...func(*Badge)) *Badge {
	if cy <= 0 {
		cy = 20
	}
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	b := &Badge{Element: ele, variant: BadgeDefault, text: text}
	for _, o := range opts {
		o(b)
	}
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(b.onPaint)
	xui.AttachWindow(ele.Handle)
	return b
}

// WithBadgeVariant 构造时设置徽章变体。
func WithBadgeVariant(v BadgeVariant) func(*Badge) {
	return func(b *Badge) { b.variant = v }
}

// WithVariant 设置徽章变体。
func (b *Badge) WithVariant(v BadgeVariant) *Badge {
	b.variant = v
	b.Redraw(false)
	return b
}

func (b *Badge) resolve(t *xui.Theme) (bg, border, fore xui.Color) {
	switch b.variant {
	case BadgeSecondary:
		return t.Secondary, xui.None, t.SecondaryForeground
	case BadgeOutline:
		return xui.None, t.Border, t.Foreground
	case BadgeDestructive:
		return t.Destructive.WithAlpha(26), xui.None, t.Destructive
	default:
		return t.Primary, xui.None, t.PrimaryForeground
	}
}

func (b *Badge) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	bg, border, fore := b.resolve(t)
	p.FillRound(&p.Rc, bg, t.Pill(p.H()))
	p.StrokeRound(&p.Rc, border, t.Pill(p.H()), 1)
	p.TextCenter(b.text, fore)
	return 0
}
