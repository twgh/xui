package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
)

// Card 自绘卡片容器: 圆角面板 + 边框, 自带标题/内容区域的静态排版助手。
// 子元素创建时以 Card 为父句柄, 用 ContentLeft/ContentTop 计算坐标。
//
// 与 shadcn card.tsx 对齐: rounded-xl, bg-card, ring-1 ring-foreground/10
// (卡片描边是 10% 前景色, 不是 Border 令牌), 内边距 16。
type Card struct {
	*widget.Element
	Padding int32 // 内容内边距(逻辑像素)
}

// NewCard 创建卡片。
func NewCard(x, y, cx, cy int32, hParent int) *Card {
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	c := &Card{Element: ele, Padding: 16}
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

// NewCardTitle 卡片标题(主题默认字体的粗体 Label)。
func NewCardTitle(x, y, cx int32, text string, hParent int) *Label {
	return NewLabel(x, y, cx, 24, text, hParent).SetBold()
}
