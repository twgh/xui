package ui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 消息行几何常量(逻辑像素), 对齐 shadcn message.tsx:
//   - MessageGroup: flex-col gap-2(8);
//   - Message: 行内 gap-2(8), align=end 时整行镜像(头像在右);
//   - MessageContent: 竖排 gap-2.5(10);
//   - MessageHeader/Footer: px-3(12), text-xs(12) medium, muted-foreground;
//   - MessageAvatar: min-w-8(32), 底对齐。
const (
	messageRowGap     int32 = 8
	messageContentGap int32 = 10
	messageMetaPadX   int32 = 12
)

// MessageAlign 消息对齐方向, 对应 shadcn message.tsx 的 align。
type MessageAlign int

const (
	MessageAlignStart MessageAlign = iota // 头像在左、内容左对齐(对方)
	MessageAlignEnd                       // 头像在右、内容右对齐(自己)
)

// MessageOption 消息配置项。
type MessageOption func(*Message)

// WithMessageAlign 设置对齐方向(默认 MessageAlignStart)。
func WithMessageAlign(a MessageAlign) MessageOption {
	return func(m *Message) { m.align = a }
}

// MessageGroup 消息组: 同一发送者的连续消息, 竖向 gap-2。
func NewMessageGroup(x, y, cx, cy int32, hParent int) *widget.LayoutEle {
	l := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if l == nil {
		return nil
	}
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(messageRowGap)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}

// Message 一条会话消息: 可选页眉 + [头像 + 内容列] + 可选页脚。
//
// 与 shadcn message.tsx 对齐:
//   - 根容器竖向排列, 页眉/正文/页脚间距 10(MessageContent 的 gap-2.5);
//   - 正文行 = 头像槽 + 内容列; align=end 时头像在右、内容列子项右对齐;
//   - 头像底对齐(bottom), 页脚在正文行**下方** —— 等价于上游
//     "头像 self-end + 有 footer 时上移 32px" 的观感; 页眉/页脚按头像槽缩进,
//     文字与内容列对齐(上游它们就在 MessageContent 里面), 缩放时随行宽更新。
//
// 槽位按需创建/显示: 没用的页眉/页脚/头像不占布局。内容列里放 Bubble:
//
//	msg := ui.NewMessage(0, 0, 0, 0, group.Handle, ui.WithMessageAlign(ui.MessageAlignEnd))
//	xui.ItemFill(msg)
//	ui.NewBubble(msg.Content().Handle, "Deploying to prod real quick.")
type Message struct {
	*widget.LayoutEle
	align   MessageAlign
	header  *widget.LayoutEle
	body    *widget.LayoutEle
	content *widget.LayoutEle
	avatar  *widget.LayoutEle
	footer  *widget.LayoutEle
	flush   bool

	contentW    int32
	metaL       int32
	metaR       int32
	bodyPending bool
	bodyHealing bool
}

// NewMessage 创建消息行。宽铺满、高自撑; cx/cy 仅作初始尺寸。
func NewMessage(x, y, cx, cy int32, hParent int, opts ...MessageOption) *Message {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	m := &Message{LayoutEle: ele}
	m.metaL, m.metaR = -1, -1
	for _, o := range opts {
		o(m)
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(messageContentGap)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)

	// 三个行预先建好(创建顺序 = 布局顺序), 未使用的不显示。
	m.header = newMessageMetaRow(ele.Handle, messageMetaPadX, m.align == MessageAlignEnd)
	m.header.Show(false)

	m.body = widget.NewLayoutEle(0, 0, 0, 0, ele.Handle)
	m.body.EnableHorizon(true)
	m.body.EnableLayout(true)
	m.body.EnableBkTransparent(true)
	m.body.EnableAutoWrap(false)
	m.body.SetSpace(messageRowGap)
	m.body.SetAlignV(xcc.Layout_Align_Bottom)
	m.body.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	m.body.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)

	// 头像在行的哪一侧由对齐方向决定: 子项顺序就是布局顺序。
	if m.align == MessageAlignEnd {
		m.content = newMessageContent(m.body.Handle, m.align)
		m.avatar = newMessageAvatarSlot(m.body.Handle)
	} else {
		m.avatar = newMessageAvatarSlot(m.body.Handle)
		m.content = newMessageContent(m.body.Handle, m.align)
	}

	m.footer = newMessageMetaRow(ele.Handle, messageMetaPadX, m.align == MessageAlignEnd)
	m.footer.Show(false)
	// 内容列在横向行里用 Weight 拿宽: 有些容器里 Weight 解析不稳定, 尺寸落定后
	// 再按"行宽 - 头像 - 间距"锁成 Fixed, 内容里的气泡/整宽控件才有准确宽度。
	m.body.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		m.scheduleReflowBody()
		return 0
	})
	m.scheduleReflowBody()
	xui.AttachWindow(ele.Handle)
	return m
}

// scheduleReflowBody 把内容列宽度重算推迟到下一轮消息(布局趟里改尺寸会重入)。
func (m *Message) scheduleReflowBody() {
	if m == nil || m.bodyPending {
		return
	}
	m.bodyPending = true
	app.AfterFunc(0, func() {
		m.bodyPending = false
		m.reflowBody()
	})
}

// reflowBody 把内容列宽度锁成"行宽 - 其它可见子项 - 间距"。
func (m *Message) reflowBody() {
	if m == nil || m.body == nil || m.content == nil || m.bodyHealing {
		return
	}
	bw := m.body.GetWidth()
	if bw <= 0 {
		return
	}
	used, others := int32(0), int32(0)
	n := xc.XEle_GetChildCount(m.body.Handle)
	for i := int32(0); i < n; i++ {
		h := xc.XEle_GetChildByIndex(m.body.Handle, i)
		if h == 0 || h == m.content.Handle || !xc.XWidget_IsShow(h) {
			continue
		}
		used += xc.XEle_GetWidth(h)
		others++
	}
	w := bw - used - others*messageRowGap
	if w < 0 {
		w = 0
	}
	if w != m.contentW {
		m.contentW = w
		m.bodyHealing = true
		m.content.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
		xc.XEle_AdjustLayout(m.body.Handle)
		m.bodyHealing = false
	}
	m.syncMetaIndent()
}

// syncMetaIndent 让页眉/页脚的文字与内容列对齐: 头像显示时按"头像宽 + 行间距"
// 把内侧边距让出来(上游这两个节点就在 MessageContent 里, 天然对齐)。
func (m *Message) syncMetaIndent() {
	if m == nil || m.header == nil || m.footer == nil {
		return
	}
	pad := messageMetaPadX
	if m.flush {
		pad = 0
	}
	var indent int32
	if m.avatar != nil && m.avatar.IsShow() {
		if w := m.avatar.GetWidth(); w > 0 {
			indent = w + messageRowGap
		}
	}
	l, r := pad, pad
	if m.align == MessageAlignEnd {
		r += indent
	} else {
		l += indent
	}
	if l == m.metaL && r == m.metaR {
		return
	}
	m.metaL, m.metaR = l, r
	xc.XEle_SetPadding(m.header.Handle, l, 0, r, 0)
	xc.XEle_SetPadding(m.footer.Handle, l, 0, r, 0)
	m.header.AdjustLayout()
	m.footer.AdjustLayout()
}

// newMessageAvatarSlot 建头像槽位(默认隐藏, 贴底对齐)。
func newMessageAvatarSlot(hParent int) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	l.EnableHorizon(true)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetAlignV(xcc.Layout_Align_Bottom)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	l.Show(false)
	return l
}

// newMessageContent 建内容列: 竖向 gap-10、吃剩余宽度; end 时子项右对齐。
//
// Weight 宽不会把宽度下发给 Fill 子项(见 DESIGN 第八节 101): Trace / Marker 这类
// 整宽内容直接放进内容列会塌成 0 宽。列宽确定后把它们锁成 Fixed, 缩放时继续更新。
func newMessageContent(hParent int, align MessageAlign) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(messageContentGap)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	if align == MessageAlignEnd {
		l.SetAlignH(xcc.Layout_Align_Right)
	}
	var lastW int32 = -1
	var pending bool
	synced := map[int]bool{}
	l.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		if pending {
			return 0
		}
		pending = true
		app.AfterFunc(0, func() {
			pending = false
			w := l.GetWidth()
			if w <= 0 || w == lastW {
				return
			}
			lastW = w
			next := make(map[int]bool, len(synced))
			for i := int32(0); i < xc.XEle_GetChildCount(l.Handle); i++ {
				child := xc.XEle_GetChildByIndex(l.Handle, i)
				var typ xcc.Layout_Size_
				var cw int32
				xc.XWidget_LayoutItem_GetWidth(child, &typ, &cw)
				if typ == xcc.Layout_Size_Fill || synced[child] {
					xc.XWidget_LayoutItem_SetWidth(child, xcc.Layout_Size_Fixed, w)
					next[child] = true
				}
			}
			synced = next
			l.AdjustLayout()
		})
		return 0
	})
	return l
}

// newMessageMetaRow 建页眉/页脚行: 横向 gap-8、宽铺满、高自撑、左右内边距 12。
func newMessageMetaRow(hParent int, padX int32, alignEnd bool) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	l.EnableHorizon(true)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(messageRowGap)
	l.SetAlignV(xcc.Layout_Align_Center)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	if alignEnd {
		l.SetAlignH(xcc.Layout_Align_Right)
	}
	xc.XEle_SetPadding(l.Handle, padX, 0, padX, 0)
	return l
}

// Header 返回页眉行(首次访问时显示)。往里放 NewMessageMeta 或自定义控件。
// 页眉随消息对齐(start 左 / end 右), 并按头像槽自动缩进到内容列。
func (m *Message) Header() *widget.LayoutEle {
	if m == nil {
		return nil
	}
	if !m.header.IsShow() {
		m.header.Show(true)
		chatReflow(m.Handle)
	}
	return m.header
}

// Content 返回内容列: 竖向、子项间距 10、吃剩余宽度; align=end 时子项右对齐。
// Bubble / BubbleGroup 放这里。
func (m *Message) Content() *widget.LayoutEle {
	if m == nil {
		return nil
	}
	return m.content
}

// AvatarSlot 返回头像槽位(首次访问时显示), 放在正文行、贴底对齐。
// 里面放 ui.NewAvatar 或自己的头像元素; 不调用则不占布局、不留间距。
func (m *Message) AvatarSlot() *widget.LayoutEle {
	if m == nil {
		return nil
	}
	if !m.avatar.IsShow() {
		m.avatar.Show(true)
		chatReflow(m.Handle)
		m.scheduleReflowBody()
	}
	return m.avatar
}

// Footer 返回页脚行(首次访问时显示), 位于正文下方;
// align=end 的消息页脚整体右对齐。
func (m *Message) Footer() *widget.LayoutEle {
	if m == nil {
		return nil
	}
	if !m.footer.IsShow() {
		m.footer.Show(true)
		chatReflow(m.Handle)
	}
	return m.footer
}

// SetMetaFlush 去掉页眉/页脚的左右内边距(对应上游 ghost 气泡时的 px-0):
// 内容用全宽透明气泡(BubbleGhost)时调用, 页眉/页脚文字才会与正文左缘对齐。
func (m *Message) SetMetaFlush(flush bool) *Message {
	if m == nil {
		return m
	}
	m.flush = flush
	m.metaL, m.metaR = -1, -1
	m.syncMetaIndent()
	return m
}

// NewMessageMeta 创建页眉/页脚里的弱化小字(text-xs medium muted-foreground)。
// 宽度按文字量测; 需要靠右时给前一个子项 ItemWeight(x, 1) 或把它放进自己的右对齐行。
func NewMessageMeta(hParent int, text string) *chatText {
	c := newChatFitText(hParent, text, chatMetaSize)
	if c == nil {
		return nil
	}
	c.SetMedium().SetMuted()
	return c
}
