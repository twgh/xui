package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Empty 几何常量(对齐 shadcn empty.tsx 的 radix-nova 样式, 单位逻辑像素):
//   - 容器: p-6(24) + gap-6(24), 内容水平/垂直居中; 上游 md:p-12(48) 是响应式断点,
//     Go 侧没有容器查询, 统一用基础档 24;
//   - EmptyHeader: gap-2(8), 最大宽 384(见下方说明);
//   - EmptyMedia icon 变体: size-10(40) + rounded-lg + bg-muted, 图标 24;
//   - EmptyTitle: text-lg(18) / 行高 28, font-medium;
//   - EmptyDescription: text-sm(14) / 行高 20, muted;
//   - EmptyContent: gap-4(16), 居中列。
const (
	emptyPad        int32 = 24
	emptyGap        int32 = 24
	emptyHeaderGap  int32 = 8
	emptyMediaSize  int32 = 40
	emptyIconSize   int32 = 24
	emptyTitleH     int32 = 28
	emptyDescH      int32 = 20
	emptyContentGap int32 = 16
)

// EmptyMediaVariant 空状态媒体变体, 对应 shadcn empty.tsx 的 EmptyMedia variant。
type EmptyMediaVariant int

const (
	EmptyMediaDefault EmptyMediaVariant = iota // 无底板, 只画图标(默认)
	EmptyMediaIcon                             // bg-muted 圆角底板 + 居中图标(size-10)
)

// EmptyMediaOption 媒体配置项。
type EmptyMediaOption func(*EmptyMedia)

// WithEmptyMediaVariant 设置媒体变体(默认 EmptyMediaDefault)。
func WithEmptyMediaVariant(v EmptyMediaVariant) EmptyMediaOption {
	return func(m *EmptyMedia) { m.variant = v }
}

// EmptyMedia 空状态媒体: 一个固定尺寸的自绘图标块。
//
// 与 shadcn empty.tsx 的 EmptyMedia 对齐:
//   - default: 透明底, 图标 24(上游是"内容随内容", Go 侧以 24 图标为默认形态);
//   - icon: size-10(40) 圆角 bg-muted 底板 + 居中 24 图标。
//
// 需要放图片/Avatar 这类非图标内容时, 直接往 Empty.Header() 里加自己的元素即可。
type EmptyMedia struct {
	*widget.Element
	variant EmptyMediaVariant
	icon    string
}

// NewEmptyMedia 创建媒体块。icon 是 lucide path 或整段 SVG(见 xui.Painter.Icon); 传空串 = 只画底板。
func NewEmptyMedia(hParent int, icon string, opts ...EmptyMediaOption) *EmptyMedia {
	m := &EmptyMedia{icon: icon}
	for _, o := range opts {
		o(m)
	}
	size := emptyIconSize
	if m.variant == EmptyMediaIcon {
		size = emptyMediaSize
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

func (m *EmptyMedia) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	rc := p.Rc
	if m.variant == EmptyMediaIcon {
		p.FillRound(&rc, t.Muted, t.RadiusLg())
	}
	if m.icon == "" {
		return 0
	}
	if m.variant == EmptyMediaIcon {
		inset := (emptyMediaSize - emptyIconSize) / 2
		rc = xc.RECT{
			Left:   rc.Left + inset,
			Top:    rc.Top + inset,
			Right:  rc.Right - inset,
			Bottom: rc.Bottom - inset,
		}
	}
	p.Icon(m.icon, &rc, t.Foreground, 2)
	return 0
}

// Empty 空状态容器: 居中的图标/标题/描述/动作区。
//
// 与 shadcn empty.tsx 对齐: 圆角卡片 + 虚线的观感由调用方决定 —— 上游默认只有
// `border-dashed` 而没有 `border` 宽度, 实际不画边框, 这里同样**不画边框**;
// 需要边框时把 Empty 放进 Card, 或自行给 Body 加背景/描边。
//
// Empty 本身是**竖向流式布局容器**, 宽度默认铺满、高度按内容自撑。用作整页/整卡空态时,
// 调用方再对 Empty 声明 `xui.ItemFillHeight(empty)`(或 Fill)让 Body 在剩余高度里垂直居中。
//
// 用法:
//
//	e := ui.NewEmpty(0, 0, 0, 0, stack.Handle)
//	xui.ItemFill(e)
//	h := e.Header()
//	ui.NewEmptyMedia(h.Handle, icons.IconFolderCode, ui.WithEmptyMediaVariant(ui.EmptyMediaIcon))
//	ui.NewEmptyTitle(h.Handle, "No Projects Yet")
//	ui.NewEmptyDescription(h.Handle, "You haven't created any projects yet.")
//	row := xui.HStack(e.Content().Handle, 36, 8)
//	ui.NewButton(0, 0, 0, 0, "Create Project", row.Handle)
type Empty struct {
	*widget.LayoutEle
	body    *widget.LayoutEle
	header  *widget.LayoutEle
	content *widget.LayoutEle
}

// NewEmpty 创建空状态容器。cy 仅作初始高度, 实际高度按内容自撑(布局项声明可覆盖)。
func NewEmpty(x, y, cx, cy int32, hParent int) *Empty {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	e := &Empty{LayoutEle: ele}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetAlignH(xcc.Layout_Align_Center)
	ele.SetAlignV(xcc.Layout_Align_Center)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.AttachWindow(ele.Handle)

	// Body 承担内边距(p-6)与 Header/Content 之间的 gap-6。左右内边距用元素 padding
	// (约束子项 Fill 宽); 上下用布局项外边距, 计入 Empty 的 Auto 高度 —— 四边都用
	// 外边距时, Empty 被 Fill 宽排过后内容区会向右溢出(同 card.ContentStack,
	// 见 DESIGN 第八节 129)。
	e.body = widget.NewLayoutEle(0, 0, 0, 0, ele.Handle)
	e.body.EnableHorizon(false)
	e.body.EnableLayout(true)
	e.body.EnableBkTransparent(true)
	e.body.EnableAutoWrap(false)
	e.body.SetSpace(emptyGap)
	e.body.SetAlignH(xcc.Layout_Align_Center)
	e.body.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	e.body.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	e.body.LayoutItem_SetMargin(0, emptyPad, 0, emptyPad)
	e.body.SetPadding(emptyPad, 0, emptyPad, 0)

	// Header/Content 预建并隐藏: 保证调用顺序不影响纵向顺序, 未使用的槽位也不产生 gap。
	e.header = e.newSlot(emptyHeaderGap)
	e.header.Show(false)
	e.content = e.newSlot(emptyContentGap)
	e.content.Show(false)
	return e
}

func (e *Empty) newSlot(gap int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, e.body.Handle)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(gap)
	l.SetAlignH(xcc.Layout_Align_Center)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}

// Body 返回带内边距的内容列(Header/Content 的父级)。要往空状态里放自定义元素时用它,
// 直接挂在 Empty 本身上会绕过内边距。
func (e *Empty) Body() *widget.LayoutEle {
	if e == nil {
		return nil
	}
	return e.body
}

// Header 返回标题区(竖向, gap-2, 居中)。首次调用时显示。
func (e *Empty) Header() *widget.LayoutEle {
	if e == nil {
		return nil
	}
	e.header.Show(true)
	return e.header
}

// Content 返回动作区(竖向, gap-4, 居中)。首次调用时显示。
func (e *Empty) Content() *widget.LayoutEle {
	if e == nil {
		return nil
	}
	e.content.Show(true)
	return e.content
}

// NewEmptyTitle 创建空状态标题: text-lg(18) + font-medium, 水平居中、宽度铺满标题区。
func NewEmptyTitle(hParent int, text string) *Label {
	l := NewLabel(0, 0, 0, emptyTitleH, text, hParent).SetFontSize(18).SetMedium().SetAlignCenter()
	xui.ItemFill(l)
	return l
}

// NewEmptyDescription 创建空状态描述: text-sm(14) + muted, 水平居中、宽度铺满。
// 说明文字较长时用 SetWrap 并自行给足高度(Label 不会随折行自撑)。
func NewEmptyDescription(hParent int, text string) *Label {
	l := NewLabel(0, 0, 0, emptyDescH, text, hParent).SetMuted().SetWrap().SetAlignCenter()
	xui.ItemFill(l)
	return l
}
