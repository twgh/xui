package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Item 几何常量(逻辑像素), 对齐 shadcn item.tsx:
//   - 容器 rounded-md + border-transparent, text-sm;
//   - size default: gap-4(16) + p-4(16); sm: gap-2.5(10) + px-4(16) / py-3(12);
//   - Content: 竖向 flex-1, gap-1(4);
//   - Header/Footer: basis-full, gap-2(8);
//   - Media icon: size-8(32) + rounded-sm + border + bg-muted, 图标 16。
const (
	itemGapDefault    int32 = 16
	itemGapSm         int32 = 10
	itemPadDefault    int32 = 16
	itemPadSmX        int32 = 16
	itemPadSmY        int32 = 12
	itemContentGap    int32 = 4
	itemHeaderGap     int32 = 8
	itemMediaGlyph    int32 = 16
	itemMediaIconSize int32 = 32
	itemTextH         int32 = 20
	itemTextSize      int32 = 14
)

// ItemVariant 列表项变体, 对应 shadcn item.tsx 的 variant。
type ItemVariant int

const (
	ItemDefault ItemVariant = iota // 透明底
	ItemOutline                    // 1px 边框
	ItemMuted                      // bg-muted/50
)

// ItemSize 列表项尺寸档, 对应 shadcn item.tsx 的 size。
type ItemSize int

const (
	ItemSizeDefault ItemSize = iota // gap-4 p-4
	ItemSizeSm                      // gap-2.5 px-4 py-3
)

// ItemOption 列表项配置项。
type ItemOption func(*Item)

// WithItemVariant 设置变体(默认 ItemDefault)。
func WithItemVariant(v ItemVariant) ItemOption {
	return func(i *Item) { i.variant = v }
}

// WithItemSize 设置尺寸档(默认 ItemSizeDefault)。
func WithItemSize(s ItemSize) ItemOption {
	return func(i *Item) { i.size = s }
}

// Item 列表项容器: 媒体 / 内容 / 动作 + 可选页眉页脚。
//
// 与 shadcn item.tsx 对齐: rounded-md, text-sm, 默认透明底; outline 描边、
// muted 用 bg-muted/50。中间一行是 Media + Content(吃剩余宽度) + Actions,
// Header/Footer 占整宽, 位于这一行的上方/下方。
//
// 槽位在首次访问时显示, 未使用的槽位不占布局、也不产生间距:
//
//	it := ui.NewItem(0, 0, 0, 0, group.Handle, ui.WithItemVariant(ui.ItemOutline))
//	xui.ItemFill(it)
//	ui.NewItemMedia(it.Media().Handle, icons.IconShieldCheck,
//		ui.WithItemMediaVariant(ui.ItemMediaIcon))
//	c := it.Content()
//	ui.NewItemTitle(c.Handle, "Security alert")
//	ui.NewItemDescription(c.Handle, "A new login from an unrecognized device.")
//	ui.NewButton(0, 0, 0, 0, "Review", it.Actions().Handle, ui.WithSize(ui.ButtonSizeSm))
//
// 上游 Item 是 flex-wrap 单容器; Go 侧拆成"页眉行 + 中间行 + 页脚行"三段,
// 避免横向 Weight 列在换行布局里失效。Header/Footer 的 justify-between 用
// 权重撑开表达: 给靠左的第一个子项 `xui.ItemWeight(x, 1)`。
type Item struct {
	*widget.LayoutEle
	variant ItemVariant
	size    ItemSize
	header  *widget.LayoutEle
	body    *widget.LayoutEle
	footer  *widget.LayoutEle
	media   *widget.LayoutEle
	content *widget.LayoutEle
	// contentW 是最近一次算出的内容区宽度; bodyHealing 防止补排递归。
	contentW    int32
	bodyHealing bool
}

// NewItem 创建列表项。宽铺满、高自撑; cx/cy 仅作初始尺寸。
func NewItem(x, y, cx, cy int32, hParent int, opts ...ItemOption) *Item {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	it := &Item{LayoutEle: ele}
	for _, o := range opts {
		o(it)
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(it.gap())
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ele.AddEvent_Paint(it.onPaint)
	xui.AttachWindow(ele.Handle)
	// 内边距用元素级 padding(实测有效): 再用一层带外边距的 pad 包住时,
	// pad 自身的宽度会减掉外边距, 但里面的 Fill 行仍按外层宽度排版 —— 行比 pad
	// 宽一个右内边距, 行尾动作区被推到卡片外裁掉(见 DESIGN 第八节 103)。
	xc.XEle_SetPadding(ele.Handle, it.padX(), it.padY(), it.padX(), it.padY())

	it.header = newItemRow(ele.Handle, itemHeaderGap)
	it.header.Show(false)
	it.body = newItemRow(ele.Handle, it.gap())
	it.body.Show(false)
	it.footer = newItemRow(ele.Handle, itemHeaderGap)
	it.footer.Show(false)
	return it
}

func (i *Item) gap() int32 {
	if i.size == ItemSizeSm {
		return itemGapSm
	}
	return itemGapDefault
}

func (i *Item) padX() int32 {
	if i.size == ItemSizeSm {
		return itemPadSmX
	}
	return itemPadDefault
}

func (i *Item) padY() int32 {
	if i.size == ItemSizeSm {
		return itemPadSmY
	}
	return itemPadDefault
}

// newItemRow 建一条横向行(整宽、高自撑、垂直居中)。
func newItemRow(hParent int, gap int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	l.EnableHorizon(true)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(gap)
	l.SetAlignV(xcc.Layout_Align_Center)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}

// showBody 懒显示中间行, 并在首次访问时补齐它的三个槽位。
func (i *Item) showBody() {
	if i == nil {
		return
	}
	if i.content == nil {
		// 创建顺序 = 布局顺序: Media / Content / Actions。
		i.media = widget.NewLayoutEle(0, 0, 0, 0, i.body.Handle)
		i.media.EnableHorizon(true)
		i.media.EnableLayout(true)
		i.media.EnableBkTransparent(true)
		i.media.EnableAutoWrap(false)
		i.media.SetAlignV(xcc.Layout_Align_Center)
		i.media.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
		i.media.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
		i.media.Show(false)

		i.content = widget.NewLayoutEle(0, 0, 0, 0, i.body.Handle)
		i.content.EnableHorizon(false)
		i.content.EnableLayout(true)
		i.content.EnableBkTransparent(true)
		i.content.EnableAutoWrap(false)
		i.content.SetSpace(itemContentGap)
		i.content.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
		i.content.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
		i.contentW = -1
		i.body.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
			i.reflowBody()
			return 0
		})
	}
	i.body.Show(true)
}

// Media 返回媒体槽位(首次访问时显示中间行)。往里面放 NewItemMedia, 或自己的头像/图片元素。
func (i *Item) Media() *widget.LayoutEle {
	if i == nil {
		return nil
	}
	i.showBody()
	i.media.Show(true)
	return i.media
}

// Content 返回内容列: 竖向、gap-4、吃剩余宽度。往里放 NewItemTitle / NewItemDescription 或自定义控件。
func (i *Item) Content() *widget.LayoutEle {
	if i == nil {
		return nil
	}
	i.showBody()
	return i.content
}

// Actions 返回中间行本身: 动作元素直接追加在 Content 之后(行内间距 = Item 的 gap)。
//
// 不另建动作子容器: xcgui 的嵌套布局不会把内部原生按钮的宽度量进 Auto 宽, 子容器会
// 塌成 0 宽、按钮被裁到卡片外(实测)。直接挂在行上时, 按钮自己的元素宽度参与 Weight
// 计算, Content 会正确让出位置。
func (i *Item) Actions() *widget.LayoutEle {
	if i == nil {
		return nil
	}
	i.showBody()
	return i.body
}

// Header 返回页眉行(整宽、横向 gap-8、垂直居中)。首次访问时显示。
func (i *Item) Header() *widget.LayoutEle {
	if i == nil {
		return nil
	}
	i.header.Show(true)
	return i.header
}

// Footer 返回页脚行(整宽、横向 gap-8、垂直居中)。首次访问时显示。
func (i *Item) Footer() *widget.LayoutEle {
	if i == nil {
		return nil
	}
	i.footer.Show(true)
	return i.footer
}

// reflowBody 按行内实际可见的媒体/动作宽度把内容区宽度锁成 Fixed。
//
// 为什么不让 Content 用 Weight: xcgui 的 Weight 在嵌套列里不会把宽度下发给 Fill
// 子项(标题/说明宽度恒为 0), 而 Fixed 宽会正常下发(见 DESIGN 第八节 101)。行上
// 直接挂动作元素时, 用元素实测宽度算剩余空间最可靠。
func (i *Item) reflowBody() {
	if i == nil || i.body == nil || i.content == nil || i.bodyHealing {
		return
	}
	bw := i.body.GetWidth()
	if bw <= 0 {
		return
	}
	used, visibleOthers := int32(0), int32(0)
	n := xc.XEle_GetChildCount(i.body.Handle)
	for idx := int32(0); idx < n; idx++ {
		h := xc.XEle_GetChildByIndex(i.body.Handle, idx)
		if h == 0 || h == i.content.Handle || !xc.XWidget_IsShow(h) {
			continue
		}
		used += xc.XEle_GetWidth(h)
		visibleOthers++
	}
	// 可见项(含 Content 自身)之间的间距数量 = 其它可见项数量。
	w := bw - used - visibleOthers*i.gap()
	if w < 0 {
		w = 0
	}
	if w == i.contentW {
		return
	}
	i.contentW = w
	i.bodyHealing = true
	i.content.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	xc.XEle_AdjustLayout(i.body.Handle)
	i.content.AdjustLayout()
	i.bodyHealing = false
}

func (i *Item) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	radius := t.RadiusMd()
	switch i.variant {
	case ItemOutline:
		p.StrokeRound(&p.Rc, t.Border, radius, 1)
	case ItemMuted:
		p.FillRound(&p.Rc, t.Muted.ScaleAlpha(0.5), radius)
	}
	return 0
}

// NewItemGroup 创建列表分组: 竖向、间距 0、宽铺满、高自撑。
// 条目之间通常用 NewItemSeparator 分隔。
func NewItemGroup(x, y, cx, cy int32, hParent int) *widget.LayoutEle {
	l := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if l == nil {
		return nil
	}
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}

// NewItemSeparator 创建列表项之间的分隔线(整宽 1px)。
func NewItemSeparator(hParent int) *Separator {
	s := NewSeparator(0, 0, 0, hParent)
	if s == nil {
		return nil
	}
	xui.ItemFill(s)
	return s
}

// NewItemTitle 创建条目标题: text-sm(14) + font-medium。
func NewItemTitle(hParent int, text string) *Label {
	l := NewLabel(0, 0, 0, itemTextH, text, hParent).SetFontSize(itemTextSize).SetMedium()
	if l == nil {
		return nil
	}
	xui.ItemFill(l)
	return l
}

// NewItemDescription 创建条目说明: text-sm + muted, 整宽。
// Label 不会随折行自撑; 默认单行截断, 需要多行时自行 SetWrap 并给足高度。
func NewItemDescription(hParent int, text string) *Label {
	l := NewLabel(0, 0, 0, itemTextH, text, hParent).SetFontSize(itemTextSize).SetMuted()
	if l == nil {
		return nil
	}
	xui.ItemFill(l)
	return l
}

// ItemMediaVariant 媒体变体, 对应 shadcn item.tsx 的 ItemMedia variant。
type ItemMediaVariant int

const (
	ItemMediaDefault ItemMediaVariant = iota // 透明底, 16 图标
	ItemMediaIcon                            // 32 方形底板(bg-muted + border), 16 图标
)

// ItemMediaOption 媒体配置项。
type ItemMediaOption func(*ItemMedia)

// WithItemMediaVariant 设置媒体变体(默认 ItemMediaDefault)。
func WithItemMediaVariant(v ItemMediaVariant) ItemMediaOption {
	return func(m *ItemMedia) { m.variant = v }
}

// ItemMedia 列表项媒体块: 一个固定尺寸的自绘图标。
//
// 上游 default 变体本身不设尺寸(内容随子元素); Go 侧以 16 图标为默认形态,
// 需要头像/图片这类自定义内容时直接往 Item.Media() 槽位里加自己的元素。
type ItemMedia struct {
	*widget.Element
	variant ItemMediaVariant
	icon    string
}

// NewItemMedia 创建媒体块。icon 是 lucide path 或整段 SVG(见 xui.Painter.Icon); 传空串 = 只画底板。
func NewItemMedia(hParent int, icon string, opts ...ItemMediaOption) *ItemMedia {
	m := &ItemMedia{icon: icon}
	for _, o := range opts {
		o(m)
	}
	size := itemMediaGlyph
	if m.variant == ItemMediaIcon {
		size = itemMediaIconSize
	}
	ele := widget.NewElement(0, 0, size, size, hParent)
	if ele == nil {
		return nil
	}
	m.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, size)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, size)
	ele.AddEvent_Paint(m.onPaint)
	xui.AttachWindow(ele.Handle)
	return m
}

func (m *ItemMedia) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	rc := p.Rc
	if m.variant == ItemMediaIcon {
		p.FillRound(&rc, t.Muted, t.RadiusSm())
		p.StrokeRound(&rc, t.Border, t.RadiusSm(), 1)
		inset := (itemMediaIconSize - itemMediaGlyph) / 2
		rc.Left += inset
		rc.Top += inset
		rc.Right -= inset
		rc.Bottom -= inset
	}
	if m.icon != "" {
		p.Icon(m.icon, &rc, t.Foreground, 2)
	}
	return 0
}
