package ui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 附件几何常量(逻辑像素), 对齐 shadcn attachment.tsx:
//   - 容器 rounded-xl border bg-card, 水平: 媒体 + 内容 + 动作, 垂直: 媒体在上;
//   - 尺寸 default/sm/xs: 媒体 40/32/28, 间距 8/10/6, 内边距 8/6/4;
//   - idle 虚线边框, error 红色边框/媒体红底, uploading/processing 标题微光。
const (
	attachmentMediaDefault int32 = 40
	attachmentMediaSm      int32 = 32
	attachmentMediaXs      int32 = 28
	attachmentPadDefault   int32 = 8
	attachmentPadSm        int32 = 6
	attachmentPadXs        int32 = 4
	attachmentGapDefault   int32 = 8
	attachmentGapSm        int32 = 10
	attachmentGapXs        int32 = 6
	attachmentVerticalW    int32 = 96
	attachmentVerticalWCon int32 = 120
	attachmentDescGap      int32 = 2
	attachmentGroupGap     int32 = 12
)

// AttachmentState 附件状态, 对应 shadcn attachment.tsx 的 state。
type AttachmentState int

const (
	AttachmentDone       AttachmentState = iota // 完成(默认)
	AttachmentIdle                              // 待上传: 虚线边框
	AttachmentUploading                         // 上传中: 标题微光
	AttachmentProcessing                        // 处理中: 标题微光
	AttachmentError                             // 失败: 红色边框/媒体/描述
)

// AttachmentSize 附件尺寸档, 对应 shadcn attachment.tsx 的 size。
type AttachmentSize int

const (
	AttachmentSizeDefault AttachmentSize = iota
	AttachmentSizeSm
	AttachmentSizeXs
)

// AttachmentOrientation 附件朝向, 对应 shadcn attachment.tsx 的 orientation。
type AttachmentOrientation int

const (
	AttachmentHorizontal AttachmentOrientation = iota // 媒体 + 内容一行
	AttachmentVertical                                // 媒体在上、内容在下
)

// AttachmentOption 附件配置项。
type AttachmentOption func(*Attachment)

// WithAttachmentState 设置状态(默认 AttachmentDone)。
func WithAttachmentState(s AttachmentState) AttachmentOption {
	return func(a *Attachment) { a.state = s }
}

// WithAttachmentSize 设置尺寸档(默认 AttachmentSizeDefault)。
func WithAttachmentSize(s AttachmentSize) AttachmentOption {
	return func(a *Attachment) { a.size = s }
}

// WithAttachmentOrientation 设置朝向(默认 AttachmentHorizontal)。
func WithAttachmentOrientation(o AttachmentOrientation) AttachmentOption {
	return func(a *Attachment) { a.orientation = o }
}

// WithAttachmentIcon 在容器左侧/上方放 16px 图标媒体块。
func WithAttachmentIcon(icon string) AttachmentOption {
	return func(a *Attachment) { a.icon = icon }
}

// WithAttachmentWidth 固定水平朝向的宽度(默认铺满父级);
// 横排/附件组里放多个水平附件时用它将每个附件收成内容宽。
func WithAttachmentWidth(w int32) AttachmentOption {
	return func(a *Attachment) { a.width = w }
}

// WithAttachmentOnClick 让整个附件可点击(对应上游 AttachmentTrigger 覆盖层):
// 悬停整块底色加深, 松开左键回调。
func WithAttachmentOnClick(fn func()) AttachmentOption {
	return func(a *Attachment) { a.onClick = fn }
}

// Attachment 文件/图片附件块: 媒体 + 标题/描述 + 动作按钮。
//
// 与 shadcn attachment.tsx 对齐:
//   - 水平: [媒体][内容(吃剩余宽)][动作], 垂直: 媒体在上、内容在下、动作在右下;
//   - 5 个状态: idle 虚线边框、uploading/processing 标题微光、error 红色、
//     done 常规;
//   - 三档尺寸(媒体 40/32/28)。
//
// 用法:
//
//	att := ui.NewAttachment(0, 0, 0, 0, body.Handle,
//		ui.WithAttachmentIcon(icons.IconFileText),
//		ui.WithAttachmentState(ui.AttachmentUploading))
//	xui.ItemFill(att)
//	att.SetText("sales-dashboard.pdf", "PDF · 2.4 MB")
//	att.Action(icons.IconX, func() { ... })
type Attachment struct {
	*widget.LayoutEle
	state       AttachmentState
	size        AttachmentSize
	orientation AttachmentOrientation
	icon        string
	width       int32
	onClick     func()
	hover       bool

	media   *AttachmentMedia
	content *widget.LayoutEle
	title   *chatText
	desc    *chatText
	actions *widget.LayoutEle

	contentW  int32
	reflowing bool
}

// AttachmentMedia 附件的方形媒体块(bg-muted + 居中图标); error 时整块转红。
type AttachmentMedia struct {
	*widget.Element
	icon     string
	iconSize int32
	state    AttachmentState
	radius   int32
	vertical bool
	attach   *Attachment
}

// NewAttachment 创建附件块。宽铺满、高自撑(cx/cy 仅作初始尺寸)。
func NewAttachment(x, y, cx, cy int32, hParent int, opts ...AttachmentOption) *Attachment {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	a := &Attachment{LayoutEle: ele, contentW: -1}
	for _, o := range opts {
		o(a)
	}
	a.applyLayout()
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	if a.orientation == AttachmentHorizontal && a.width > 0 {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, a.width)
	} else {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	}
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ele.AddEvent_Paint(a.onPaint)
	ele.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		a.scheduleReflow()
		return 0
	})
	xui.AttachWindow(ele.Handle)
	if a.icon != "" {
		a.ensureMedia()
	}
	if a.onClick != nil {
		ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
			if !a.hover {
				a.hover = true
				a.Redraw(true)
			}
			return 0
		})
		ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
			if a.hover {
				a.hover = false
				a.Redraw(true)
			}
			return 0
		})
		// 有 Down 才会派发同一次按下的 Up(同 breadcrumb)。
		ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			return 0
		})
		ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			if a.onClick != nil {
				a.onClick()
			}
			return 0
		})
	}
	a.scheduleReflow()
	return a
}

// applyLayout 按尺寸/朝向重建根容器的布局属性(朝向不可变, 只在构造时调用)。
func (a *Attachment) applyLayout() {
	ele := a.LayoutEle
	if a.orientation == AttachmentVertical {
		ele.EnableHorizon(false)
		ele.EnableLayout(true)
		ele.SetAlignH(xcc.Layout_Align_Left)
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, a.verticalWidth())
	} else {
		ele.EnableHorizon(true)
		ele.EnableLayout(true)
		ele.SetAlignV(xcc.Layout_Align_Center)
		if a.width > 0 {
			ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, a.width)
		} else {
			ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		}
	}
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(a.gap())
	xc.XEle_SetPadding(ele.Handle, a.pad(), a.pad(), a.pad(), a.pad())
}

func (a *Attachment) verticalWidth() int32 {
	if a.content != nil || a.title != nil || a.desc != nil {
		return attachmentVerticalWCon
	}
	return attachmentVerticalW
}

func (a *Attachment) gap() int32 {
	switch a.size {
	case AttachmentSizeSm:
		return attachmentGapSm
	case AttachmentSizeXs:
		return attachmentGapXs
	}
	return attachmentGapDefault
}

func (a *Attachment) pad() int32 {
	switch a.size {
	case AttachmentSizeSm:
		return attachmentPadSm
	case AttachmentSizeXs:
		return attachmentPadXs
	}
	return attachmentPadDefault
}

func (a *Attachment) mediaSize() int32 {
	switch a.size {
	case AttachmentSizeSm:
		return attachmentMediaSm
	case AttachmentSizeXs:
		return attachmentMediaXs
	}
	return attachmentMediaDefault
}

func (a *Attachment) radius(t *xui.Theme) int32 {
	if a.size == AttachmentSizeXs {
		return t.RadiusLg()
	}
	return t.RadiusXl()
}

// ensureMedia 首次使用时创建媒体块(必须排在最前)。
func (a *Attachment) ensureMedia() *AttachmentMedia {
	if a == nil {
		return nil
	}
	if a.media != nil {
		return a.media
	}
	m := &AttachmentMedia{icon: a.icon, state: a.state, vertical: a.orientation == AttachmentVertical, attach: a}
	w := a.mediaSize()
	if m.vertical {
		w = a.verticalWidth() - a.pad()*2
	}
	ele := widget.NewElement(0, 0, w, w, a.Handle)
	if ele == nil {
		return nil
	}
	m.Element = ele
	m.iconSize = 16
	if a.size == AttachmentSizeXs {
		m.iconSize = 14
	}
	if m.vertical {
		m.iconSize = 24
	}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, w)
	ele.AddEvent_Paint(m.onPaint)
	xui.AttachWindow(ele.Handle)
	a.media = m
	a.bindChild(m.Handle)
	// 媒体是第一个子项: 后建的内容/动作都在它之后, 不用调整顺序。
	return m
}

// Media 返回媒体块(没有图标时不创建, 返回 nil)。需要自定义媒体内容时往
// MediaSlot() 里放自己的元素。
func (a *Attachment) Media() *AttachmentMedia {
	if a == nil {
		return nil
	}
	return a.media
}

// MediaSlot 返回媒体元素(首次访问时创建): 水平时固定方形, 垂直时占满内容宽。
// 媒体块自带底色/图标绘制; 需要图片媒体时用 MediaSlot 自己叠加图片元素。
func (a *Attachment) MediaSlot() *widget.Element {
	if a == nil {
		return nil
	}
	m := a.ensureMedia()
	if m == nil {
		return nil
	}
	return m.Element
}

// ensureContent 首次使用时创建内容列(标题/描述都挂这里)。
func (a *Attachment) ensureContent() *widget.LayoutEle {
	if a == nil {
		return nil
	}
	if a.content != nil {
		return a.content
	}
	c := widget.NewLayoutEle(0, 0, 0, 0, a.Handle)
	c.EnableHorizon(false)
	c.EnableLayout(true)
	c.EnableBkTransparent(true)
	c.EnableAutoWrap(false)
	c.SetSpace(attachmentDescGap)
	c.SetAlignV(xcc.Layout_Align_Center)
	// 横排里主轴 Fill 会独占整行、把动作区挤出去: 先按 Weight 吃剩余宽度,
	// 根宽落定后 reflow 再锁成 Fixed(Fill 子项才有确定宽度)。
	if a.orientation == AttachmentVertical {
		c.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	} else {
		c.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	}
	c.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	if a.orientation == AttachmentVertical {
		xc.XEle_SetPadding(c.Handle, 4, 0, 4, 0)
	}
	a.content = c
	a.bindChild(c.Handle)
	// 垂直朝向的宽度由"有没有内容"决定(96 → 120), 这里补一次。
	a.applyLayout()
	return c
}

// SetText 设置标题与描述(先后顺序即显示顺序)。title/desc 传空串表示不显示。
func (a *Attachment) SetText(title, desc string) *Attachment {
	if a == nil {
		return nil
	}
	c := a.ensureContent()
	if c == nil {
		return a
	}
	if title != "" && a.title == nil {
		t := newChatFillText(c.Handle, title, a.titleSize())
		if t != nil {
			t.SetMedium()
			if a.state == AttachmentUploading || a.state == AttachmentProcessing {
				t.SetShimmer(true)
			}
			a.title = t
			a.bindChild(t.Handle)
		}
	} else if title != "" && a.title != nil {
		a.title.SetText(title)
	}
	if desc != "" && a.desc == nil {
		d := newChatFillText(c.Handle, desc, chatMetaSize)
		if d != nil {
			d.SetColorFunc(a.descColor)
			a.desc = d
			a.bindChild(d.Handle)
		}
	} else if desc != "" && a.desc != nil {
		a.desc.SetText(desc)
	}
	chatReflow(a.Handle)
	a.scheduleReflow()
	return a
}

func (a *Attachment) titleSize() int32 {
	if a.size == AttachmentSizeDefault {
		return chatTextSize
	}
	return chatMetaSize
}

// bindChild 把子元素的点击转发到附件根(对应上游 AttachmentTrigger 覆盖整卡)。
func (a *Attachment) bindChild(h int) {
	if a == nil || h == 0 || a.onClick == nil {
		return
	}
	ele := widget.NewElementByHandle(h)
	if ele == nil {
		return
	}
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if a.onClick != nil {
			a.onClick()
		}
		return 0
	})
}

// descColor 描述色: 常规 muted-foreground, error 时 destructive/80。
func (a *Attachment) descColor(t *xui.Theme) xui.Color {
	if a.state == AttachmentError {
		return t.Destructive.ScaleAlpha(0.8)
	}
	return t.MutedForeground
}

// Title 返回标题文本元素(可能为 nil)。
func (a *Attachment) Title() *chatText {
	if a == nil {
		return nil
	}
	return a.title
}

// Description 返回描述文本元素(可能为 nil)。
func (a *Attachment) Description() *chatText {
	if a == nil {
		return nil
	}
	return a.desc
}

// Actions 返回动作区行(首次访问时创建), 位于内容之后; 垂直朝向右对齐。
func (a *Attachment) Actions() *widget.LayoutEle {
	if a == nil {
		return nil
	}
	if a.actions == nil {
		r := widget.NewLayoutEle(0, 0, 0, 0, a.Handle)
		r.EnableHorizon(true)
		r.EnableLayout(true)
		r.EnableBkTransparent(true)
		r.EnableAutoWrap(false)
		r.SetSpace(4)
		r.SetAlignV(xcc.Layout_Align_Center)
		if a.orientation == AttachmentVertical {
			r.SetAlignH(xcc.Layout_Align_Right)
		}
		r.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
		r.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
		a.actions = r
	}
	return a.actions
}

// Action 追加一个幽灵图标按钮(icon-xs 档), 点击回调。
func (a *Attachment) Action(icon string, onClick func()) *Button {
	row := a.Actions()
	if row == nil {
		return nil
	}
	b := NewButton(0, 0, 0, 0, "", row.Handle,
		WithVariant(ButtonGhost), WithSize(ButtonSizeIconXs), WithIcon(icon))
	if b != nil && onClick != nil {
		b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			onClick()
			return 0
		})
	}
	return b
}

// SetState 切换状态: idle 虚线框、uploading/processing 标题微光、error 红色。
func (a *Attachment) SetState(s AttachmentState) *Attachment {
	if a == nil || a.state == s {
		return a
	}
	a.state = s
	if a.media != nil {
		a.media.state = s
	}
	if a.title != nil {
		a.title.SetShimmer(s == AttachmentUploading || s == AttachmentProcessing)
	}
	a.redrawAll()
	a.Redraw(false)
	return a
}

// State 返回当前状态。
func (a *Attachment) State() AttachmentState {
	if a == nil {
		return AttachmentDone
	}
	return a.state
}

// redrawAll 重绘根与所有子项(状态色在各自 paint 里实时算)。
func (a *Attachment) redrawAll() {
	if a == nil {
		return
	}
	a.Redraw(false)
	n := xc.XEle_GetChildCount(a.Handle)
	for i := int32(0); i < n; i++ {
		if h := xc.XEle_GetChildByIndex(a.Handle, i); h != 0 {
			xc.XEle_Redraw(h, false)
		}
	}
}

// scheduleReflow 把宽度计算推迟到下一轮消息(根宽在布局后才确定)。
func (a *Attachment) scheduleReflow() {
	if a == nil || a.reflowing || a.content == nil {
		return
	}
	a.reflowing = true
	app.AfterFunc(0, func() {
		a.reflowing = false
		a.reflow()
	})
}

// reflow 按根宽给内容列算可用宽度(横排除掉媒体/动作与间距, 竖排铺满),
// 并同步垂直朝向的媒体宽度。内容列用 Fixed 宽, 里面的 Fill 文本才会被截断
// (Weight/Fill 列里的 Fill 子项拿不到宽度, 见 DESIGN 第八节 101)。
func (a *Attachment) reflow() {
	if a == nil || a.content == nil || a.Handle == 0 || !xc.XC_IsHXCGUI(a.Handle, xcc.XC_ELE) {
		return
	}
	rw := a.GetWidth()
	if rw <= 0 {
		return
	}
	inner := rw - a.pad()*2
	if a.orientation == AttachmentVertical {
		if a.media != nil && a.media.GetWidth() != inner {
			a.media.SetWidth(inner)
			a.media.SetHeight(inner)
			a.media.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, inner)
			a.media.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, inner)
		}
		if inner != a.contentW {
			a.contentW = inner
			a.content.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, inner)
		}
		// 媒体变宽变高后根容器的 Auto 高要重新落定, 否则内容被裁在卡片外。
		chatReflow(a.Handle)
		return
	}
	used, others := int32(0), int32(0)
	n := xc.XEle_GetChildCount(a.Handle)
	for i := int32(0); i < n; i++ {
		h := xc.XEle_GetChildByIndex(a.Handle, i)
		if h == 0 || h == a.content.Handle || !xc.XWidget_IsShow(h) {
			continue
		}
		used += xc.XEle_GetWidth(h)
		others++
	}
	w := inner - used - others*a.gap()
	if w < 0 {
		w = 0
	}
	if w == a.contentW {
		return
	}
	a.contentW = w
	a.content.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	xc.XEle_AdjustLayout(a.Handle)
}

func (a *Attachment) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	radius := a.radius(t)
	bg := t.Card
	if a.hover && a.onClick != nil {
		bg = t.Muted.ScaleAlpha(0.5)
	}
	p.FillRound(&p.Rc, bg, radius)
	border := t.Border
	if a.state == AttachmentError {
		border = t.Destructive.ScaleAlpha(0.3)
	}
	if a.state == AttachmentIdle {
		rc := p.Rc
		rc.Left++
		rc.Top++
		rc.Right--
		rc.Bottom--
		chatDashedBorder(p, rc, radius, border)
	} else {
		p.StrokeRound(&p.Rc, border, radius, 1)
	}
	return 0
}

func (m *AttachmentMedia) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	bg := t.Muted
	color := t.Foreground
	if m.state == AttachmentError {
		bg = t.Destructive.ScaleAlpha(0.1)
		color = t.Destructive
	}
	radius := t.RadiusLg()
	if m.attach != nil && m.attach.size == AttachmentSizeXs {
		radius = t.RadiusMd()
	}
	p.FillRound(&p.Rc, bg, radius)
	if m.icon != "" {
		inset := (p.Rc.Right - p.Rc.Left - m.iconSize) / 2
		rc := xc.RECT{
			Left: p.Rc.Left + inset, Top: p.Rc.Top + inset,
			Right: p.Rc.Right - inset, Bottom: p.Rc.Bottom - inset,
		}
		p.Icon(m.icon, &rc, color, 2)
	}
	return 0
}

// AttachmentGroup 附件横排容器: 子项间距 12, 放不下时自动换行
// (上游是横向滚动; xcgui 没有横向滚动容器, 这里用换行表达)。
func NewAttachmentGroup(x, y, cx, cy int32, hParent int) *widget.LayoutEle {
	l := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if l == nil {
		return nil
	}
	l.EnableHorizon(true)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(true)
	l.SetSpace(attachmentGroupGap)
	l.SetAlignV(xcc.Layout_Align_Top)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}
