package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
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
//
// 与 shadcn card.tsx 对齐: rounded-xl, bg-card, ring-1 ring-foreground/10
// (卡片描边是 10% 前景色, 不是 Border 令牌), 内边距 16(或 size=sm 的 12)。
//
// **卡片本身是布局容器**(底层 LayoutEle, 竖直流式): 内容区
// (ContentStack 返回的那个)是它的子项, 随卡片尺寸自动跟随 —— 卡片被父布局拉伸/切成
// Auto 高度时, 内容区与其内的控件一起跟随, 不需要调用方重算宽度。
// 卡片若要按内容自撑高度, 把布局项高度声明成 `xcc.Layout_Size_Auto` 即可。
type Card struct {
	*widget.LayoutEle
	Padding int32 // 内容内边距(逻辑像素)
	size    CardSize
	stack   *widget.LayoutEle // 内容区(懒建, 只建一个)
}

// NewCard 创建卡片。cx/cy 是卡片尺寸(放进布局时由布局项声明覆盖); opts 见 WithCardSize。
func NewCard(x, y, cx, cy int32, hParent int, opts ...CardOption) *Card {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	c := &Card{LayoutEle: ele, Padding: 16}
	for _, o := range opts {
		o(c)
	}
	if c.size == CardSizeSm {
		c.Padding = 12
	}
	// 竖直流式 + 关掉"纵向溢出换列": LayoutEle 在主轴放不下下一项时会把它甩到下一列
	// (x += 容器宽度, y = 0), 表现为控件整块消失(实测, 见 DESIGN 第八节 41)。
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false)
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(c.onPaint)
	xui.AttachWindow(ele.Handle)
	return c
}

// ContentLeft 内容区左边界(相对卡片) —— 即内边距。
func (c *Card) ContentLeft() int32 { return c.Padding }

// ContentTop 内容区上边界(相对卡片) —— 即内边距。
func (c *Card) ContentTop() int32 { return c.Padding }

// Size 返回卡片尺寸预设(sm 卡片的内边距/页脚内边距都是 12)。
func (c *Card) Size() CardSize {
	if c == nil {
		return CardSizeDefault
	}
	return c.size
}

// TitleSize 返回卡片标题字号(对应上游 `text-base` / sm 卡片的 `text-sm`):
// default 16, sm 14。标题创建后用它设置字号, 与卡片尺寸保持一致。
func (c *Card) TitleSize() int32 {
	if c != nil && c.size == CardSizeSm {
		return 14
	}
	return 16
}

// Footer 创建与卡片尺寸匹配的页脚(内边距跟随卡片: default 16 / sm 12),
// 并以 Fill 宽作为卡片的布局项 —— 通常直接用它, 不必再算 NewCardFooter 的参数。
func (c *Card) Footer(cy int32) *CardFooter {
	if c == nil {
		return nil
	}
	f := NewCardFooter(0, 0, 0, cy, c.Handle, WithCardFooterPadding(c.Padding))
	if f == nil {
		return nil
	}
	xui.ItemFill(f)
	return f
}

// ContentWidth 内容区当前宽度(卡片宽 - 两侧内边距)。
func (c *Card) ContentWidth() int32 { return c.GetWidth() - c.Padding*2 }

// ContentStack 取卡片的内容区 —— 一个纵向流式布局容器, 卡片里那串内容都挂在它下面。
// 子项间距 space; 内边距由内容区的四边外边距表达, 所以它会随卡片尺寸自动伸缩
// (卡片被父布局拉伸时, 内容区宽度跟着变, 不需要调用方重算)。
//
// 高度按**内容自撑**(Auto): 卡片高度可以是 Auto(自撑)也可以是固定值, 内容都贴着上边排。
// 内容要撑满卡片高度(表格/树/图标网格这类"越高越好"的内容)时, 卡片高度给固定值(或 Fill),
// 再对内容区与目标子项声明 `xui.ItemFillHeight`。
//
// 同一个卡片只建一个内容区: 重复调用返回同一个, 只是把 space 改掉。
func (c *Card) ContentStack(space int32) *widget.LayoutEle {
	if c.stack == nil {
		l := widget.NewLayoutEle(0, 0, 0, 0, c.Handle)
		l.EnableHorizon(false)
		l.EnableLayout(true)
		l.EnableBkTransparent(true)
		l.EnableAutoWrap(false)
		l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
		l.LayoutItem_SetMargin(c.Padding, c.Padding, c.Padding, c.Padding)
		c.stack = l
	}
	c.stack.SetSpace(space)
	return c.stack
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
// 用 `card.TitleSize()` 取字号后 `SetFontSize`(构造器拿不到所属卡片, 无法自动判断)。
func NewCardTitle(x, y, cx int32, text string, hParent int) *Label {
	return NewLabel(x, y, cx, 24, text, hParent).SetFontSize(16)
}

// NewCardDescription 卡片说明文字。
// 与 shadcn card.tsx 对齐: `text-sm text-muted-foreground`(弱化色)。
func NewCardDescription(x, y, cx int32, text string, hParent int) *Label {
	return NewLabel(x, y, cx, 20, text, hParent).SetMuted()
}

// CardHeader 卡片头: 一行两列 —— 左列放标题/说明(竖排), 右列放动作(贴右上)。
//
// 对齐 shadcn card.tsx:
//   - CardHeader = grid auto-rows-min items-start gap-1, 有 CardAction 时
//     grid-cols-[1fr_auto](左列占满剩余宽, 动作列按内容宽);
//   - CardAction = col-start-2 row-span-2 row-start-1 self-start justify-self-end
//     —— **跨越标题与说明两行、顶端对齐**贴在右上角。所以动作列是整行高的一块,
//     不是"标题行里的一格": 标题再长(换行)动作也不会被挤下去。
//
// 用法(parent = 卡片内容区 ContentStack 的返回值, space = shadcn 的 gap-1 = 4):
//
//	h := ui.NewCardHeader(stack.Handle, 4)
//	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "标题", h.TitleStack().Handle))
//	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "说明", h.TitleStack().Handle))
//	xui.ItemFill(ui.NewButton(0, 0, 0, 0, "", h.ActionStack().Handle,
//		ui.WithIcon(icons.IconEllipsis), ui.WithSize(ui.ButtonSizeIconSm)))
type CardHeader struct {
	*widget.LayoutEle
	title  *widget.LayoutEle
	action *widget.LayoutEle
	space  int32
}

// NewCardHeader 创建卡片头。parent 传卡片内容区(ContentStack 的返回值)。
// 头的宽度已声明为 Fill(卡片内容区宽), 高度 Auto 自撑, 调用方不需要再声明尺寸。
func NewCardHeader(x, y, cx int32, hParent int, space int32) *CardHeader {
	row := widget.NewLayoutEle(x, y, cx, 0, hParent)
	if row == nil {
		return nil
	}
	h := &CardHeader{LayoutEle: row, space: space}
	row.EnableHorizon(true)
	row.EnableLayout(true)
	row.EnableBkTransparent(true)
	row.EnableAutoWrap(false) // 横向行放不下时别把动作列甩到下一行(见 DESIGN 41)
	row.SetSpace(space)
	row.SetAlignV(xcc.Layout_Align_Top) // items-start: 动作列顶端对齐
	row.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	row.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.AttachWindow(row.Handle)
	// 左列先建出来(哪怕没人往里放标题): 它是 `1fr` 的占位列, 只有它存在,
	// 只放动作、不写标题的卡片头才会把动作顶到右边(同 shadcn 的 grid-cols-[1fr_auto])。
	h.TitleStack()
	return h
}

// TitleStack 取左侧标题/说明的竖排容器(构造时已建, 同时充当 1fr 占位列)。
// 标题与说明按 space 间距堆叠。
func (h *CardHeader) TitleStack() *widget.LayoutEle {
	if h.title == nil {
		l := widget.NewLayoutEle(0, 0, 0, 0, h.Handle)
		l.EnableHorizon(false)
		l.EnableLayout(true)
		l.EnableBkTransparent(true)
		l.EnableAutoWrap(false)
		l.SetSpace(h.space)
		l.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1) // 1fr
		l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
		h.title = l
	}
	return h.title
}

// ActionStack 取右侧动作区(随需创建), 横排、宽高都按内容自撑, 贴右上角。
func (h *CardHeader) ActionStack() *widget.LayoutEle {
	if h.action == nil {
		l := widget.NewLayoutEle(0, 0, 0, 0, h.Handle)
		l.EnableHorizon(true)
		l.EnableLayout(true)
		l.EnableBkTransparent(true)
		l.EnableAutoWrap(false)
		l.SetSpace(h.space)
		l.SetAlignV(xcc.Layout_Align_Center)
		l.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
		l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
		h.action = l
	}
	return h.action
}

// CardFooterOption 页脚配置项。
type CardFooterOption func(*CardFooter)

// WithCardFooterPadding 设置页脚四边内边距(对应 shadcn 的 p-(--card-spacing))。
// 默认 16; size=sm 的卡片传 12(用 Card.Footer 时自动跟随)。
func WithCardFooterPadding(p int32) CardFooterOption {
	return func(f *CardFooter) { f.Padding = p }
}

// CardFooter 卡片页脚: 顶边一条 border-t + bg-muted/50, 只有**下两角**随卡片圆角
// (shadcn card.tsx 的 CardFooter: `flex items-center rounded-b-xl border-t bg-muted/50 p-4`)。
//
// 页脚自己是横向布局容器(子项垂直居中、高度按内容自撑), 内容放在 Content() 返回的
// 内层容器里 —— 那一层带四边内边距, 所以按钮/文字不会贴边。直接挂到页脚上的子项
// 会跳过内边距, 只有确实要贴边的内容(自定义背景/分隔线)才那样用。
//
// 用法: `footer := card.Footer(0)`(推荐, 内边距跟随卡片尺寸), 或
// `ui.NewCardFooter(0, 0, 0, 0, card.Handle)` 后声明 `xui.ItemFill(footer)`;
// 再往 `footer.Content()` 里放控件。
type CardFooter struct {
	*widget.LayoutEle
	Padding int32             // 四边内边距(逻辑像素)
	content *widget.LayoutEle // 内层内容区(带内边距外边距)
}

// NewCardFooter 创建页脚容器。cy 传 0 = 高度按内容自撑(含上下内边距);
// 传固定值则锁高, 内容在页脚内垂直居中。
func NewCardFooter(x, y, cx, cy int32, hParent int, opts ...CardFooterOption) *CardFooter {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	f := &CardFooter{LayoutEle: ele, Padding: 16}
	for _, o := range opts {
		o(f)
	}
	ele.EnableHorizon(true)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false) // 页脚一行放不下时别把子项甩到下一行(见 DESIGN 41)
	ele.EnableBkTransparent(true)
	ele.SetAlignV(xcc.Layout_Align_Center) // items-center
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	if cy > 0 {
		ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, cy)
	} else {
		ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	}

	// 内层内容区: 宽铺满, 高按内容自撑, 四边外边距 = 页脚内边距。
	// 用子项外边距而不是元素内边距: xcgui 的 XEle_SetPadding 对布局子项的影响未实测,
	// 外边距是与卡片内容区一致的既有做法(见 card.ContentStack)。
	inner := widget.NewLayoutEle(0, 0, 0, 0, ele.Handle)
	inner.EnableHorizon(true)
	inner.EnableLayout(true)
	inner.EnableAutoWrap(false)
	inner.EnableBkTransparent(true)
	inner.SetAlignV(xcc.Layout_Align_Center)
	inner.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	inner.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	inner.LayoutItem_SetMargin(f.Padding, f.Padding, f.Padding, f.Padding)
	f.content = inner

	ele.AddEvent_Paint(f.onPaint)
	xui.AttachWindow(ele.Handle)
	return f
}

// Content 返回页脚的内层内容区(横向流式、子项垂直居中、四边已留出内边距)。
// 页脚要竖排(如登录卡的两个整宽按钮)时对返回值调 `EnableHorizon(false)`。
func (f *CardFooter) Content() *widget.LayoutEle {
	if f == nil {
		return nil
	}
	return f.content
}

// ContentStack 设置内容区子项间距并返回它(与 Card.ContentStack 同名同义)。
func (f *CardFooter) ContentStack(space int32) *widget.LayoutEle {
	if f == nil || f.content == nil {
		return nil
	}
	f.content.SetSpace(space)
	return f.content
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
