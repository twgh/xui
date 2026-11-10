package ui

import (
	"math"
	"strings"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 对话类组件(message / bubble / marker / trace / attachment)共用的自绘文字与图标。
//
// 为什么不用 Label: Label 的载体是原生 Button, 文字排版区比元素窄一圈, 按文字量宽
// 会被裁成省略号(同 breadcrumb 的实测结论)。这些组件的文字宽度要么必须等于墨迹宽度
// (marker 居中分隔、气泡按文字量宽), 要么整行截断(attachment 标题), 都要求宽度精确。

const (
	chatTextSize int32 = 14 // text-sm
	chatMetaSize int32 = 12 // text-xs
)

// chatFont 取对话组件使用的字体句柄。
func chatFont(t *xui.Theme, size int32) int {
	spec := t.FontDefault
	if size > 0 {
		spec.Size = size
	}
	return xui.FontHandle(spec)
}

// chatMediumFont 取中等字重(font-medium)的字体句柄: 用粗体字形但不动字号。
func chatMediumFont(t *xui.Theme, size int32) int {
	spec := t.FontDefault
	if size > 0 {
		spec.Size = size
	}
	spec.Bold = true
	return xui.FontHandle(spec)
}

// chatText 单行自绘文本: 宽度可"按文字量测(fit)"或"铺满父级并截断(fill)"。
//
// 高度固定为一行行高; 需要多行请用 Bubble(它自己按折行量高)。
type chatText struct {
	*widget.Element
	text    string
	size    int32
	fit     bool
	muted   bool
	medium  bool
	colorFn func(*xui.Theme) xui.Color
	shimmer bool
}

// newChatFitText 创建按文字量宽的文本(宽度随主题/文字更新)。
func newChatFitText(hParent int, text string, size int32) *chatText {
	c := newChatTextBase(hParent, text, size, true)
	if c != nil {
		c.syncWidth()
	}
	return c
}

// newChatFillText 创建铺满父级宽度的文本(超宽省略号)。
func newChatFillText(hParent int, text string, size int32) *chatText {
	return newChatTextBase(hParent, text, size, false)
}

func newChatTextBase(hParent int, text string, size int32, fit bool) *chatText {
	if size <= 0 {
		size = chatTextSize
	}
	h := xui.TextHeight(text, chatFont(xui.Current(), size))
	if h <= 0 {
		h = size + 6
	}
	w := int32(1)
	if fit {
		w = chatTextWidth(text, size)
	}
	ele := widget.NewElement(0, 0, w, h, hParent)
	if ele == nil {
		return nil
	}
	c := &chatText{Element: ele, text: text, size: size, fit: fit}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	if fit {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	} else {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	}
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	ele.AddEvent_Paint(c.onPaint)
	xui.AttachWindow(ele.Handle)
	xui.OnThemeChangeFor(ele.Handle, func(t *xui.Theme) {
		c.syncWidth()
		c.Redraw(false)
	})
	return c
}

func chatTextWidth(text string, size int32) int32 {
	w := xui.TextWidth(text, chatFont(xui.Current(), size))
	if w < 1 {
		w = 1
	}
	return w
}

// syncWidth 按当前主题字体重算 fit 文本的宽度。
func (c *chatText) syncWidth() {
	if c == nil || !c.fit || c.Handle == 0 || !xc.XC_IsHXCGUI(c.Handle, xcc.XC_ELE) {
		return
	}
	t := xui.Current()
	w := int32(1)
	if c.text != "" {
		if c.medium {
			w = xui.TextWidth(c.text, chatMediumFont(t, c.size))
		} else {
			w = xui.TextWidth(c.text, chatFont(t, c.size))
		}
		if w < 1 {
			w = 1
		}
	}
	if w != c.GetWidth() {
		c.SetWidth(w)
		c.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	}
}

// SetText 更新文本(同时重算 fit 宽度)。
func (c *chatText) SetText(text string) *chatText {
	if c == nil {
		return nil
	}
	c.text = text
	c.syncWidth()
	c.Redraw(false)
	return c
}

// SetMuted 使用弱化色(muted-foreground)。
func (c *chatText) SetMuted() *chatText {
	if c == nil {
		return nil
	}
	c.muted = true
	c.Redraw(false)
	return c
}

// SetMedium 使用中等字重(font-medium)。
func (c *chatText) SetMedium() *chatText {
	if c == nil {
		return nil
	}
	c.medium = true
	c.syncWidth()
	c.Redraw(false)
	return c
}

// SetColorFunc 指定语义色解析器(绘制时实时取令牌)。
func (c *chatText) SetColorFunc(fn func(*xui.Theme) xui.Color) *chatText {
	if c == nil {
		return nil
	}
	c.colorFn = fn
	c.Redraw(false)
	return c
}

// SetShimmer 开启"微光扫过"动画(uploading / running 这类进行中状态)。
// 底字半透明, 一条亮带循环扫过; 相位由运行时 Spin 驱动, 不占额外定时器。
func (c *chatText) SetShimmer(shimmer bool) *chatText {
	if c == nil || c.shimmer == shimmer {
		return c
	}
	c.shimmer = shimmer
	c.Redraw(false)
	return c
}

func (c *chatText) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	color := t.Foreground
	if c.colorFn != nil {
		color = c.colorFn(t)
	} else if c.muted {
		color = t.MutedForeground
	}
	var font int
	if c.medium {
		font = chatMediumFont(t, c.size)
	} else {
		font = chatFont(t, c.size)
	}
	align := xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter | xcc.TextFormatFlag_NoWrap |
		xcc.TextTrimming_EllipsisCharacter
	if c.shimmer {
		chatShimmerText(p, c.text, &p.Rc, font, color.ScaleAlpha(0.55), color,
			xui.Spin(hEle, true), align)
		return 0
	}
	p.Draw.SetFont(font)
	p.Text(c.text, &p.Rc, color, align)
	return 0
}

// chatIcon 固定尺寸的自绘图标。
type chatIcon struct {
	*widget.Element
	icon    string
	angle   float64
	colorFn func(*xui.Theme) xui.Color
}

// newChatIcon 创建图标元素(边长 size)。
func newChatIcon(hParent int, icon string, size int32) *chatIcon {
	if size <= 0 {
		size = 16
	}
	ele := widget.NewElement(0, 0, size, size, hParent)
	if ele == nil {
		return nil
	}
	c := &chatIcon{Element: ele, icon: icon}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, size)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, size)
	ele.AddEvent_Paint(c.onPaint)
	xui.AttachWindow(ele.Handle)
	return c
}

// SetAngle 设置图标旋转角(弧度), 用于可折叠触发条的 chevron。
func (c *chatIcon) SetAngle(angle float64) *chatIcon {
	if c == nil {
		return nil
	}
	c.angle = angle
	c.Redraw(false)
	return c
}

// SetColorFunc 指定语义色解析器(绘制时实时取令牌)。
func (c *chatIcon) SetColorFunc(fn func(*xui.Theme) xui.Color) *chatIcon {
	if c == nil {
		return nil
	}
	c.colorFn = fn
	return c
}

func (c *chatIcon) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	color := t.MutedForeground
	if c.colorFn != nil {
		color = c.colorFn(t)
	}
	rc := p.Rc
	if c.angle != 0 {
		p.IconRotated(c.icon, &rc, color, 2, c.angle)
	} else {
		p.Icon(c.icon, &rc, color, 2)
	}
	return 0
}

// chatReflow 在子项增删后让 Auto 高度重新落定: 先排自己, 再交给最近的布局框架
// (与 label.go 的 reflowAfterTextHeight 同一条路径; 本文件被多个组件共用, 复制一份
// 避免它们为了一个重排助手把 label.go 拉进安装闭包)。
func chatReflow(hEle int) {
	if hEle == 0 || !xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
		return
	}
	xc.XEle_AdjustLayout(hEle)
	for p := xc.XWidget_GetParentEle(hEle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(p)
			break
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(pp)
			break
		}
	}
}

// chatLineHeight 返回字体的单行行高(拿不到时按 14px 文字的 20 兜底)。
func chatLineHeight(hFont int) int32 {
	if h := xui.TextHeight("Ag", hFont); h > 0 {
		return h
	}
	return 20
}

// chatWrapLines 按词折行(shadcn 的 whitespace-pre-wrap + wrap-break-word 语义):
// 先按显式换行分段, 再按空格贪心装行; 单个词超过宽度时才按字符硬切。
// 不用 DrawText 自带的换行: 它按字符切, 英文单词会被拦腰断开。
func chatWrapLines(text string, hFont int, width int32) []string {
	if text == "" {
		return []string{""}
	}
	if width <= 0 || hFont == 0 {
		return []string{text}
	}
	var out []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Split(para, " ")
		cur := ""
		for _, w := range words {
			cand := w
			if cur != "" {
				cand = cur + " " + w
			}
			if xui.TextWidth(cand, hFont) <= width {
				cur = cand
				continue
			}
			if cur != "" {
				out = append(out, cur)
			}
			if xui.TextWidth(w, hFont) <= width {
				cur = w
				continue
			}
			cur = ""
			for _, r := range w {
				next := cur + string(r)
				if cur != "" && xui.TextWidth(next, hFont) > width {
					out = append(out, cur)
					cur = string(r)
				} else {
					cur = next
				}
			}
		}
		out = append(out, cur)
	}
	return out
}

// chatWrapHeight 量测文本在 width 下折行后的高度。
func chatWrapHeight(text string, hFont int, width int32) int32 {
	if text == "" || hFont == 0 || width <= 0 {
		return 0
	}
	var sz xc.SIZE
	app.GetTextShowRect(text, -1, hFont, xcc.TextAlignFlag_Left, width, &sz)
	if sz.CY <= 0 {
		sz.CY = xui.TextHeight(text, hFont)
	}
	return sz.CY
}

// chatShimmerText 画"微光扫过"的文本: 底字用 base 色, 一条移动亮带内的字用 hi 色。
//
// shadcn 的 shimmer 是渐变文字扫光(background-clip: text); xcgui 的文字只能用
// 单色画, 这里用裁剪区让同段文字分两次上色, 观感一致且不需要生成渐变位图。
// 相位由调用方用 xui.Spin 取(持续动画, 不要自己起定时器)。
func chatShimmerText(p *xui.Painter, text string, rc *xc.RECT, hFont int, base, hi xui.Color,
	phase float64, align xcc.TextFormatFlag_) {
	if text == "" || hFont == 0 {
		return
	}
	p.Draw.SetFont(hFont)
	p.Text(text, rc, base, align)
	w := rc.Right - rc.Left
	if w <= 0 {
		return
	}
	band := int32(72)
	if band > w {
		band = w
	}
	x := rc.Left - band + int32(phase*float64(w+band))
	clip := xc.RECT{Left: x, Top: rc.Top, Right: x + band, Bottom: rc.Bottom}
	if clip.Left < rc.Left {
		clip.Left = rc.Left
	}
	if clip.Right > rc.Right {
		clip.Right = rc.Right
	}
	if clip.Right > clip.Left {
		p.Draw.SetClipRect(&clip)
		p.Text(text, rc, hi, align)
		p.Draw.ClearClip()
	}
}

// chatDashedBorder 画 1px 短划线圆角边框: 四条直边 + 四段 1/4 圆弧,
// 圆角段沿弧线按弧长布点, 四角不会留空(同 CSS border-dashed 的观感)。
func chatDashedBorder(p *xui.Painter, rc xc.RECT, radius int32, color xui.Color) {
	if color.IsNone() || rc.Right <= rc.Left || rc.Bottom <= rc.Top {
		return
	}
	d := p.Draw
	d.SetLineWidthF(1)
	d.SetBrushColor(uint32(color))
	const dash, gap = float32(4), float32(3)
	r := float32(radius)
	if r*2 >= float32(rc.Right-rc.Left) {
		r = float32(rc.Right-rc.Left) / 3
	}
	if r*2 >= float32(rc.Bottom-rc.Top) {
		r = float32(rc.Bottom-rc.Top) / 3
	}
	l, t := float32(rc.Left), float32(rc.Top)
	rt, bt := float32(rc.Right), float32(rc.Bottom)

	dashEdge := func(x1, y1, x2, y2 float32) {
		dx, dy := x2-x1, y2-y1
		length := float32(math.Hypot(float64(dx), float64(dy)))
		for s := float32(0); s < length; s += dash + gap {
			e := s + dash
			if e > length {
				e = length
			}
			d.DrawLineF(x1+dx*s/length, y1+dy*s/length, x1+dx*e/length, y1+dy*e/length)
		}
	}
	dashArc := func(cx, cy, a0, sweep float32) {
		length := r * float32(math.Abs(float64(sweep)))
		for s := float32(0); s < length; s += dash + gap {
			e := s + dash
			if e > length {
				e = length
			}
			a1 := a0 + sweep*s/length
			a2 := a0 + sweep*e/length
			d.DrawLineF(
				cx+r*float32(math.Cos(float64(a1))), cy+r*float32(math.Sin(float64(a1))),
				cx+r*float32(math.Cos(float64(a2))), cy+r*float32(math.Sin(float64(a2))),
			)
		}
	}

	dashEdge(l+r, t, rt-r, t)
	dashArc(rt-r, t+r, -math.Pi/2, math.Pi/2)
	dashEdge(rt, t+r, rt, bt-r)
	dashArc(rt-r, bt-r, 0, math.Pi/2)
	dashEdge(rt-r, bt, l+r, bt)
	dashArc(l+r, bt-r, math.Pi/2, math.Pi/2)
	dashEdge(l, bt-r, l, t+r)
	dashArc(l+r, t+r, math.Pi, math.Pi/2)
}
