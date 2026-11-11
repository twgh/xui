package ui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 气泡几何常量(逻辑像素), 对齐 shadcn bubble.tsx:
//   - 根: w-fit, max-w-80%(ghost 为全宽), 圆角 xl;
//   - BubbleContent: rounded-xl px-3(12) py-2(8), text-sm(14) leading-relaxed;
//   - 变体: default=primary / secondary / muted / tinted(primary 淡化) / outline /
//     ghost(无底无内边距) / destructive(10~20% 红底);
//   - 反应条: 叠在气泡底/顶边, 12px 内边距起排, 胶囊高 24, 约 6px 压在气泡上。
const (
	bubblePadX            int32 = 12
	bubblePadY            int32 = 8
	bubbleMaxFrac               = 0.8
	bubbleChipH           int32 = 24
	bubbleChipGap         int32 = 4
	bubbleChipPadX        int32 = 6
	bubbleReactionOverlap int32 = 6 // 反应条压在气泡上的高度
	bubbleGroupGap        int32 = 8 // BubbleGroup 子项间距(gap-2)
)

// BubbleVariant 气泡变体, 对应 shadcn bubble.tsx 的 variant。
type BubbleVariant int

const (
	BubbleDefault     BubbleVariant = iota // primary 底 + primary-foreground
	BubbleSecondary                        // secondary 底
	BubbleMuted                            // muted 底
	BubbleTinted                           // primary 淡化底(亮暗各一档)
	BubbleOutline                          // background 底 + border
	BubbleGhost                            // 无底、无内边距、全宽(assistant 正文)
	BubbleDestructive                      // destructive 10%(暗色 20%)底 + 红字
)

// BubbleAlign 气泡内容/反应条的对齐方向。
type BubbleAlign int

const (
	BubbleAlignStart BubbleAlign = iota
	BubbleAlignEnd
)

// BubbleReactionSide 反应条的叠放边。
type BubbleReactionSide int

const (
	BubbleReactionBottom BubbleReactionSide = iota // 压在下边缘
	BubbleReactionTop                              // 压在上边缘
)

// BubbleOption 气泡配置项。
type BubbleOption func(*Bubble)

// WithBubbleVariant 设置变体(默认 BubbleDefault)。
func WithBubbleVariant(v BubbleVariant) BubbleOption {
	return func(b *Bubble) { b.variant = v }
}

// WithBubbleAlign 设置对齐方向(默认 BubbleAlignStart)。对齐本身由父容器决定,
// 这里只决定反应条的默认起始边; 放进 Message.Content() 时会跟着消息对齐。
func WithBubbleAlign(a BubbleAlign) BubbleOption {
	return func(b *Bubble) { b.align = a }
}

// WithBubbleMaxWidth 覆盖最大宽度占比(0~1, 默认 0.8; Ghost 恒为全宽)。
func WithBubbleMaxWidth(frac float64) BubbleOption {
	return func(b *Bubble) {
		if frac > 0 && frac <= 1 {
			b.maxFrac = frac
		}
	}
}

// WithBubbleOnClick 让整个气泡可点击(对应上游把 BubbleContent 渲染成 button/a):
// 悬停换变体对应的深一档底色, 松开左键回调。
func WithBubbleOnClick(fn func()) BubbleOption {
	return func(b *Bubble) { b.onClick = fn }
}

type bubbleReaction struct {
	text    string
	onClick func()
}

// Bubble 文本气泡: 宽度按文字量测、超过父级 80% 时折行, 自带变体底色。
//
// 与 shadcn bubble.tsx 对齐: 内容是一段文本(BubbleContent 的常见用法),
// 反应条画在气泡边缘(胶囊 + 卡片色描边), 可点击。需要放附件/按钮这类自定义内容时,
// 把它们作为相邻子项放进 Message.Content(), 不要塞进气泡。
//
// 宽度依赖父容器实际宽度(80% 上限), 父级尺寸变化时自动重量; 构造后首帧由
// AfterFunc 补排一次。
type Bubble struct {
	*widget.Element
	text    string
	variant BubbleVariant
	align   BubbleAlign
	maxFrac float64
	onClick func()
	hover   bool

	reactions []bubbleReaction
	side      BubbleReactionSide
	chipAlign BubbleAlign
	hoverIdx  int
	chipRects []xc.RECT

	lastW, lastH int32
	pending      bool
}

// NewBubble 创建文本气泡并挂到 hParent(通常为 Message.Content())。
func NewBubble(hParent int, text string, opts ...BubbleOption) *Bubble {
	ele := widget.NewElement(0, 0, 1, 1, hParent)
	if ele == nil {
		return nil
	}
	b := &Bubble{
		Element:   ele,
		text:      text,
		maxFrac:   bubbleMaxFrac,
		chipAlign: BubbleAlignEnd,
		hoverIdx:  -1,
	}
	for _, o := range opts {
		o(b)
	}
	if b.variant == BubbleGhost {
		b.maxFrac = 1
	}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 1)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 1)
	ele.AddEvent_Paint(b.onPaint)
	xui.AttachWindow(ele.Handle)
	xui.OnThemeChangeFor(ele.Handle, func(t *xui.Theme) {
		b.scheduleReflow()
		b.Redraw(false)
	})
	if b.onClick != nil {
		ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
			if !b.hover {
				b.hover = true
				b.Redraw(true)
			}
			return 0
		})
		ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
			if b.hover {
				b.hover = false
				b.Redraw(true)
			}
			return 0
		})
		ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			return 0
		})
		ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			if b.onClick != nil {
				b.onClick()
			}
			return 0
		})
	}
	// 父容器尺寸变化 → 重新按 80% 上限折行; 直接挂父元素, 气泡自身尺寸不会
	// 随窗口缩小而变(它已锁成 Fixed)。
	if parent := xc.XWidget_GetParentEle(ele.Handle); parent != 0 {
		if pe := widget.NewElementByHandle(parent); pe != nil {
			pe.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
				b.scheduleReflow()
				return 0
			})
		}
	}
	b.scheduleReflow()
	return b
}

// SetText 更新文本并重新折行。
func (b *Bubble) SetText(text string) *Bubble {
	if b == nil {
		return nil
	}
	b.text = text
	// 同一个折行高度内继续加字(流式回复)时尺寸不变, reflow 会提前返回,
	// 不重绘的话界面停在最后一次改高时的文字(实测: 停在 "…as soon as the")。
	b.Redraw(false)
	b.scheduleReflow()
	return b
}

// SetVariant 切换变体。
func (b *Bubble) SetVariant(v BubbleVariant) *Bubble {
	if b == nil || b.variant == v {
		return b
	}
	b.variant = v
	if v == BubbleGhost {
		b.maxFrac = 1
	}
	b.scheduleReflow()
	return b
}

// AddReaction 在气泡边缘加一枚反应胶囊(文本 + 点击回调)。反应条不参与布局,
// 画在气泡外沿: side=bottom 时下缘外露, side=top 时上缘外露。
func (b *Bubble) AddReaction(text string, onClick func()) *Bubble {
	if b == nil {
		return nil
	}
	b.reactions = append(b.reactions, bubbleReaction{text: text, onClick: onClick})
	if len(b.reactions) == 1 {
		b.hoverIdx = -1
		b.Element.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			idx := b.chipAt(pPt.X, pPt.Y)
			if idx != b.hoverIdx {
				b.hoverIdx = idx
				b.Redraw(false)
			}
			return 0
		})
		b.Element.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
			if b.hoverIdx != -1 {
				b.hoverIdx = -1
				b.Redraw(false)
			}
			return 0
		})
		b.Element.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			return 0
		})
		b.Element.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			if idx := b.chipAt(pPt.X, pPt.Y); idx >= 0 && idx < len(b.reactions) {
				if b.reactions[idx].onClick != nil {
					b.reactions[idx].onClick()
				}
			}
			return 0
		})
	}
	b.scheduleReflow()
	return b
}

// SetReactionsSide 设置反应条叠放边(默认底部)。
func (b *Bubble) SetReactionsSide(side BubbleReactionSide) *Bubble {
	if b == nil {
		return b
	}
	b.side = side
	b.scheduleReflow()
	return b
}

// SetReactionsAlign 设置反应条水平对齐(默认 end)。
func (b *Bubble) SetReactionsAlign(a BubbleAlign) *Bubble {
	if b == nil {
		return b
	}
	b.chipAlign = a
	b.scheduleReflow()
	return b
}

// reserve 是反应条占用的额外高度(压在气泡上的部分不算)。
func (b *Bubble) reserve() int32 {
	if len(b.reactions) == 0 {
		return 0
	}
	return bubbleChipH - bubbleReactionOverlap
}

// pad 返回内容内边距(ghost 为 0)。
func (b *Bubble) pad() (int32, int32) {
	if b.variant == BubbleGhost {
		return 0, 0
	}
	return bubblePadX, bubblePadY
}

// bgRect 返回气泡底色矩形(反应条那一侧让出外露高度)。
func (b *Bubble) bgRect(rc xc.RECT) xc.RECT {
	r := b.reserve()
	if r > 0 {
		if b.side == BubbleReactionTop {
			rc.Top += r
		} else {
			rc.Bottom -= r
		}
	}
	return rc
}

// scheduleReflow 把重量推迟到下一轮消息(父级宽度在本轮布局里才确定)。
func (b *Bubble) scheduleReflow() {
	if b == nil || b.pending {
		return
	}
	b.pending = true
	app.AfterFunc(0, func() {
		b.pending = false
		b.reflow()
	})
}

// reflow 按父级宽度与文字量测重算气泡尺寸:
// 宽度 = min(文字宽 + 内边距, 父宽 × 80%), 高度 = 折行高 + 内边距 + 反应条外露。
func (b *Bubble) reflow() {
	if b == nil || b.Handle == 0 || !xc.XC_IsHXCGUI(b.Handle, xcc.XC_ELE) {
		return
	}
	parent := xc.XWidget_GetParentEle(b.Handle)
	if parent == 0 {
		return
	}
	pw := xc.XEle_GetWidth(parent)
	if pw <= 0 {
		return
	}
	t := xui.Current()
	font := chatFont(t, chatTextSize)
	px, py := b.pad()
	natural := xui.TextWidth(b.text, font) + px*2
	maxW := int32(float64(pw)*b.maxFrac + 0.5)
	w := natural
	if w > maxW {
		w = maxW
	}
	if w < 1 {
		w = 1
	}
	textW := w - px*2
	if textW < 1 {
		textW = 1
	}
	lineH := chatLineHeight(font)
	h := lineH*int32(len(chatWrapLines(b.text, font, textW))) + py*2
	if h < 1 {
		h = 1
	}
	h += b.reserve()
	if w == b.lastW && h == b.lastH {
		return
	}
	b.lastW, b.lastH = w, h
	b.SetWidth(w)
	b.SetHeight(h)
	b.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	b.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	// 气泡在 Message.Content() 里是 Fixed 宽: 改完要让父级重排(对齐到 start/end)。
	// 流式 SetText 时高度会反复变化, 只排直接父级时外层 Message / 滚动条目仍是旧高度
	// (实测: 气泡从 36 长到 56, 条目停在 27), 必须一路排到滚动框架。
	b.adjustAncestors()
}

// adjustAncestors 逐级重排祖先, 直到窗口或滚动框架。
func (b *Bubble) adjustAncestors() {
	for p := b.Handle; p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			// ScrollArea 的框架自己会重排内容区并重算滚动范围。
			xc.XEle_AdjustLayout(p)
			return
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			// p 是框架的内容区: 直接排内容区会把子项宽度打成 0(实测), 交给框架排。
			xc.XEle_AdjustLayout(pp)
			return
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_ELE) {
			xc.XEle_AdjustLayout(p)
			continue
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
		}
		return
	}
}

// bubbleBg 返回当前变体的底色; hover 时换深一档(对应上游 /80、/5% 这类悬停色)。
func bubbleBg(t *xui.Theme, v BubbleVariant, hover bool) xui.Color {
	switch v {
	case BubbleDefault:
		if hover {
			return t.Primary.Mix(t.Background, 0.2)
		}
		return t.Primary
	case BubbleSecondary:
		if hover {
			return t.Secondary.Mix(t.Foreground, 0.05)
		}
		return t.Secondary
	case BubbleMuted:
		if hover {
			return t.Muted.Mix(t.Foreground, 0.05)
		}
		return t.Muted
	case BubbleTinted:
		base := t.Primary.Mix(t.Background, 0.9)
		if t.IsDark() {
			base = t.Primary.Mix(t.Background, 0.8)
		}
		if hover {
			return base.Mix(t.Foreground, 0.06)
		}
		return base
	case BubbleOutline:
		if hover {
			return t.Muted
		}
		return t.Background
	case BubbleGhost:
		if hover {
			return t.Muted.ScaleAlpha(0.5)
		}
		return xui.None
	case BubbleDestructive:
		a := 0.1
		if t.IsDark() {
			a = 0.2
		}
		if hover {
			a += 0.08
		}
		return t.Destructive.ScaleAlpha(a)
	}
	return xui.None
}

// bubbleFg 返回当前变体的文字色。
func bubbleFg(t *xui.Theme, v BubbleVariant) xui.Color {
	switch v {
	case BubbleDefault:
		return t.PrimaryForeground
	case BubbleSecondary:
		return t.SecondaryForeground
	case BubbleDestructive:
		return t.Destructive
	}
	return t.Foreground
}

func (b *Bubble) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	radius := t.RadiusXl()

	bg := b.bgRect(p.Rc)
	if bg.Bottom <= bg.Top || bg.Right <= bg.Left {
		return 0
	}
	bgColor := bubbleBg(t, b.variant, b.hover)
	if !bgColor.IsNone() {
		p.FillRound(&bg, bgColor, radius)
	}
	if b.variant == BubbleOutline {
		p.StrokeRound(&bg, t.Border, radius, 1)
	}

	// 文本
	px, py := b.pad()
	tr := bg
	tr.Left += px
	tr.Right -= px
	tr.Top += py
	tr.Bottom -= py
	if tr.Right > tr.Left && tr.Bottom > tr.Top && b.text != "" {
		font := chatFont(t, chatTextSize)
		lineH := chatLineHeight(font)
		color := bubbleFg(t, b.variant)
		lines := chatWrapLines(b.text, font, tr.Right-tr.Left)
		p.Draw.SetFont(font)
		y := tr.Top
		for _, line := range lines {
			rc := xc.RECT{Left: tr.Left, Top: y, Right: tr.Right, Bottom: y + lineH}
			p.Text(line, &rc, color,
				xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
			y += lineH
		}
	}

	b.drawReactions(p, t, bg)
	return 0
}

// drawReactions 画反应胶囊: 先铺一圈卡片色描边(ring-3 ring-card), 再铺 muted 底。
func (b *Bubble) drawReactions(p *xui.Painter, t *xui.Theme, bg xc.RECT) {
	b.chipRects = b.chipRects[:0]
	if len(b.reactions) == 0 {
		return
	}
	font := chatFont(t, chatMetaSize)
	widths := make([]int32, len(b.reactions))
	total := int32(0)
	for i, r := range b.reactions {
		w := xui.TextWidth(r.text, font) + bubbleChipPadX*2
		if w < bubbleChipH {
			w = bubbleChipH
		}
		widths[i] = w
		total += w
	}
	total += bubbleChipGap * int32(len(b.reactions)-1)
	x := bg.Left + bubblePadX
	if b.chipAlign == BubbleAlignEnd {
		x = bg.Right - bubblePadX - total
	}
	if x < bg.Left {
		x = bg.Left
	}
	y := bg.Bottom - bubbleReactionOverlap
	if b.side == BubbleReactionTop {
		y = bg.Top - bubbleChipH + bubbleReactionOverlap
	}
	for i, r := range b.reactions {
		rc := xc.RECT{Left: x, Top: y, Right: x + widths[i], Bottom: y + bubbleChipH}
		b.chipRects = append(b.chipRects, rc)
		outer := rc
		p.FillRound(&outer, t.Card, bubbleChipH/2)
		inner := rc
		inner.Left += 3
		inner.Top += 3
		inner.Right -= 3
		inner.Bottom -= 3
		hover := b.hoverIdx == i
		p.FillRound(&inner, xui.Fade(t.Muted, t.Accent, xui.HoverItem(b.Handle, int32(i), hover)), bubbleChipH/2)
		crc := inner
		p.Draw.SetFont(font)
		p.Text(r.text, &crc, t.MutedForeground,
			xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		x = rc.Right + bubbleChipGap
	}
}

// chipAt 返回坐标(x, y)命中的反应胶囊下标; 未命中返回 -1。
func (b *Bubble) chipAt(x, y int32) int {
	for i, rc := range b.chipRects {
		if x >= rc.Left && x < rc.Right && y >= rc.Top && y < rc.Bottom {
			return i
		}
	}
	return -1
}

// BubbleGroup 连续气泡组: 竖向 gap-2, 宽铺满、高自撑。
func NewBubbleGroup(x, y, cx, cy int32, hParent int) *widget.LayoutEle {
	l := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if l == nil {
		return nil
	}
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(bubbleGroupGap)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}
