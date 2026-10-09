package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

// CardSize 卡片尺寸预设, 对应 shadcn card.tsx 的 size。
// shadcn 用 `--card-spacing` 表达内边距: default = spacing(4) = 16px, sm = spacing(3) = 12px。
type CardSize int

const (
	CardSizeDefault CardSize = iota // 内边距 16
	CardSizeSm                      // 内边距 12(卡片标题也应变小, 见下)
)

// CardOption 卡片配置项。
type CardOption func(*Card)

// WithCardSize 设置卡片尺寸预设(sm = 内边距 12)。
func WithCardSize(s CardSize) CardOption { return func(c *Card) { c.size = s } }

// Card 自绘卡片容器: 圆角面板 + 边框, 自带标题/说明/页脚类型的静态排版助手。
// 子元素创建时以 Card 为父句柄, 用 ContentLeft/ContentTop 计算坐标。
//
// 与 shadcn card.tsx 对齐: rounded-xl, bg-card, ring-1 ring-foreground/10
// (卡片描边是 10% 前景色, 不是 Border 令牌), 内边距 16(或 size=sm 的 12)。
type Card struct {
	*widget.Element
	Padding int32 // 内容内边距(逻辑像素)
	size    CardSize
}

// NewCard 创建卡片。cx/cy 是卡片尺寸; opts 见 WithCardSize。
func NewCard(x, y, cx, cy int32, hParent int, opts ...CardOption) *Card {
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	c := &Card{Element: ele, Padding: 16}
	for _, o := range opts {
		o(c)
	}
	if c.size == CardSizeSm {
		c.Padding = 12
	}
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(c.onPaint)
	xui.AttachWindow(ele.Handle)
	return c
}

// ContentLeft 内容区左边界(相对卡片)。
func (c *Card) ContentLeft() int32 { return c.Padding }

// ContentTop 内容区上边界(相对卡片)。
func (c *Card) ContentTop() int32 { return c.Padding }

// ContentWidth 内容区宽度。
func (c *Card) ContentWidth() int32 { return c.GetWidth() - c.Padding*2 }

// ContentStack 在内容区建一个纵向流式布局容器(卡片里那串内容都挂在它下面): 绝对定位在内容区
// (ContentLeft/ContentTop), 宽度 = 内容区宽, 子项间距 space。
//
// 卡片不是布局容器, 所以它是绝对定位的; 它自己的子项才走布局(见 xui/layout.go 的布局约定)。
// 关掉"纵向溢出换列": LayoutEle 在主轴放不下下一项时会把它甩到下一列(x += 容器宽度, y = 0),
// 表现为控件整块消失(实测, 见 DESIGN 第八节 41)。
func (c *Card) ContentStack(space int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(c.ContentLeft(), c.ContentTop(),
		c.ContentWidth(), c.GetHeight()-c.Padding*2, c.Handle)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.SetSpace(space)
	l.EnableAutoWrap(false)
	return l
}

func (c *Card) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	p.FillRound(&p.Rc, t.Card, t.RadiusXl())
	p.StrokeRound(&p.Rc, t.Foreground.WithAlpha(26), t.RadiusXl(), 1)
	return 0
}

// NewCardTitle 卡片标题。
// 与 shadcn card.tsx 对齐: `text-base`(16) + `font-medium`(中文下 500 回落到常规字重,
// 所以不开粗体 —— 用粗体字形会明显偏粗)。size=sm 的卡片标题应为 text-sm(14),
// 传 `SetFontSize(14)` 即可。
func NewCardTitle(x, y, cx int32, text string, hParent int) *Label {
	return NewLabel(x, y, cx, 24, text, hParent).SetFontSize(16)
}

// NewCardDescription 卡片说明文字。
// 与 shadcn card.tsx 对齐: `text-sm text-muted-foreground`(弱化色)。
func NewCardDescription(x, y, cx int32, text string, hParent int) *Label {
	return NewLabel(x, y, cx, 20, text, hParent).SetMuted()
}

// CardFooter 卡片页脚容器: 顶边一条 border-t + bg-muted/50, 只有**下两角**随卡片圆角
// (shadcn card.tsx 的 CardFooter)。放进内容后由调用方把它贴到卡片底边。
type CardFooter struct {
	*widget.Element
}

// NewCardFooter 创建页脚容器。cy 传 0 时用 44(与 dialog 页脚同量级)。
func NewCardFooter(x, y, cx, cy int32, hParent int) *CardFooter {
	if cy <= 0 {
		cy = 44
	}
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	f := &CardFooter{Element: ele}
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(f.onPaint)
	xui.AttachWindow(ele.Handle)
	return f
}

func (f *CardFooter) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	// bg-muted/50(半透明底叠在卡片上), 底两角随卡片 rounded-xl
	bg := t.Muted.ScaleAlpha(0.5)
	r := t.RadiusXl()
	p.FillRound(&p.Rc, bg, r)
	// 上两角补方: shadcn 的页脚只有下两角圆角(整块圆角矩形会把上两角也削掉)
	if h := p.Rc.Bottom - p.Rc.Top; h > r {
		top := xc.RECT{Left: p.Rc.Left, Top: p.Rc.Top, Right: p.Rc.Right, Bottom: p.Rc.Top + r}
		p.FillRound(&top, bg, 0)
	}
	// border-t
	p.Line(p.Rc.Left, p.Rc.Top, p.Rc.Right, p.Rc.Top, t.Border, 1)
	return 0
}
