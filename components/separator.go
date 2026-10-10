package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
)

// SeparatorOrientation 分隔线取向, 对应 shadcn separator.tsx 的 orientation。
type SeparatorOrientation int

const (
	SeparatorHorizontal SeparatorOrientation = iota // 水平: 高 1px, 横向铺开
	SeparatorVertical                               // 垂直: 宽 1px, 纵向铺开
)

// Separator 分隔线: 载体为原生 Element, 画一条 1 逻辑像素的主题色线。
// 与 shadcn separator.tsx 对齐: bg-border; 水平 `h-px w-full`, 垂直 `w-px self-stretch`。
type Separator struct {
	*widget.Element
	vertical bool
}

// SeparatorOption 分隔线配置项。
type SeparatorOption func(*Separator)

// WithSeparatorVertical 改为垂直分隔线。此时 NewSeparator 的 cx 表示**长度(高度)**。
func WithSeparatorVertical() SeparatorOption {
	return func(s *Separator) { s.vertical = true }
}

// WithSeparatorOrientation 显式指定取向。
func WithSeparatorOrientation(o SeparatorOrientation) SeparatorOption {
	return func(s *Separator) { s.vertical = o == SeparatorVertical }
}

// NewSeparator 创建分隔线。cx 是长度(水平 = 宽, 垂直 = 高), 线粗固定 1 逻辑像素。
func NewSeparator(x, y, cx int32, hParent int, opts ...SeparatorOption) *Separator {
	s := &Separator{}
	for _, o := range opts {
		o(s)
	}
	var ele *widget.Element
	if s.vertical {
		ele = widget.NewElement(x, y, 1, cx, hParent)
	} else {
		ele = widget.NewElement(x, y, cx, 1, hParent)
	}
	if ele == nil {
		return nil
	}
	s.Element = ele
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
	// 直接填充整个元素矩形(元素本身就是 1px 粗的细条): 用画线的话线宽沿路径居中,
	// 贴在元素边缘的那一半会被元素裁剪掉 —— 实测 150% DPI 下只剩 1 个物理像素、
	// 比 Border 令牌的观感淡一半(见 DESIGN 第八节 10)。
	p.FillRound(&p.Rc, t.Border, 0)
	return 0
}

// BadgeVariant 徽章变体, 对应 shadcn badge.tsx 的 variant。
type BadgeVariant int

const (
	BadgeDefault     BadgeVariant = iota // 实心主色 —— shadcn 的 `default`
	BadgeSecondary                       // 弱化填充
	BadgeOutline                         // 边框
	BadgeDestructive                     // 破坏性(软色调)
	BadgeGhost                           // 无底无框, 悬停才出底色
	BadgeLink                            // 主色文字, 悬停加下划线
)

// Badge 自绘徽章: 载体为原生 Element + 手绘文字。
// 与 shadcn badge.tsx 对齐: h-5(20), rounded-4xl(胶囊), text-xs(主题字号 -2);
// destructive 为软色调(bg-destructive/10 text-destructive)。
//
// 关于悬停: shadcn 的 badge 悬停样式写作 `[a]:hover:*`(只有包在链接里才生效)。
// Go 侧没有"链接"概念, 这里**恒按悬停生效**, 行为与 Button 一致(见 docs/DESIGN.md 第四节的差异清单)。
type Badge struct {
	*widget.Element
	variant BadgeVariant
	text    string
	hover   bool
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
	// 悬停: ghost / link / outline 的变体语义要靠它体现
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		b.hover = true
		ele.Redraw(true)
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		b.hover = false
		ele.Redraw(true)
		return 0
	})
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

// resolve 按变体与悬停进度解析底色/边框/文字色, 逐条对应 badge.tsx 的类名。
func (b *Badge) resolve(t *xui.Theme, hv float64) (bg, border, fore xui.Color) {
	var bgHV, foreHV xui.Color
	switch b.variant {
	case BadgeSecondary: // bg-secondary text-secondary-foreground; hover:bg-secondary/80
		bg, fore = t.Secondary, t.SecondaryForeground
		bgHV = t.Secondary.WithAlpha(204)
	case BadgeOutline: // border-border text-foreground; hover:bg-muted hover:text-muted-foreground
		border, fore = t.Border, t.Foreground
		bgHV, foreHV = t.Muted, t.MutedForeground
	case BadgeDestructive: // bg-destructive/10 text-destructive; hover:bg-destructive/20
		bg, fore = t.Destructive.WithAlpha(26), t.Destructive
		bgHV = t.Destructive.WithAlpha(51)
	case BadgeGhost: // hover:bg-muted hover:text-muted-foreground (dark:hover:bg-muted/50)
		fore = t.Foreground
		bgHV, foreHV = t.Muted, t.MutedForeground
		if t.IsDark() {
			bgHV = t.Muted.ScaleAlpha(0.5)
		}
	case BadgeLink: // text-primary hover:underline(下划线在 onPaint 画)
		fore = t.Primary
	default: // bg-primary text-primary-foreground; hover:bg-primary/80
		bg, fore = t.Primary, t.PrimaryForeground
		bgHV = t.Primary.WithAlpha(204)
	}
	bg = xui.Fade(bg, bgHV, hv)
	if !foreHV.IsNone() {
		fore = xui.Fade(fore, foreHV, hv)
	}
	return bg, border, fore
}

func (b *Badge) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	hv := xui.Hover(hEle, b.hover)
	bg, border, fore := b.resolve(t, hv)
	radius := t.Pill(p.H())
	// 先铺宿主底色: link / ghost 变体静止态 bg = None(FillRound 对 None 跳过),
	// 透明元素重绘不擦底(见 DESIGN 第八节 28), 不铺的话上一帧的悬停痕迹会残留。
	// 组件按"放在 Card 上"设计(同 Checkbox/Slider/Pagination)。
	p.FillRound(&p.Rc, t.Card, 0)
	p.FillRound(&p.Rc, bg, radius)
	p.StrokeRound(&p.Rc, border, radius, 1)

	// text-xs: 主题字号 -2(与 tooltip 同一口径)
	fontSize := t.FontDefault.Size - 2
	f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: fontSize})
	if f != 0 {
		p.Draw.SetFont(f)
	}
	p.TextCenter(b.text, fore)

	// link 变体: hover:underline + underline-offset-4(手绘, 写法同 Button link)。
	// 透明度随过渡值缩放 —— 过渡最后一帧(值恰好到位)不再触发重绘, 若按全色画,
	// 淡出倒数第二帧(v≈0.02)的全色下划线会永久留在画布上(实测残留)。
	if b.variant == BadgeLink && hv > 0 {
		if w := xui.TextWidth(b.text, f); w > 0 {
			cy := (p.Rc.Top + p.Rc.Bottom) / 2
			y := cy + int32(float32(fontSize)*0.3) + 4
			x0 := (p.Rc.Left+p.Rc.Right)/2 - w/2
			p.Line(x0, y, x0+w, y, fore.ScaleAlpha(hv), 1)
		}
	}
	return 0
}
