package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 标记行几何常量(逻辑像素), 对齐 shadcn marker.tsx:
//   - 单行 min-h-4(16), gap-2(8), text-sm(14) muted-foreground;
//   - border: 底部 1px 边框 + pb-2(8);
//   - separator: 文字居中, 两侧各一条 1px 横线(flex-1), 距文字 4。
const (
	markerGap       int32 = 8
	markerIconSize  int32 = 16
	markerLineInset int32 = 4
	markerBorderPad int32 = 8
)

// MarkerVariant 标记行变体, 对应 shadcn marker.tsx 的 variant。
type MarkerVariant int

const (
	MarkerDefault   MarkerVariant = iota // 透明底
	MarkerBorder                         // 底部边框
	MarkerSeparator                      // 两侧横线的居中分隔标签
)

// MarkerOption 标记行配置项。
type MarkerOption func(*Marker)

// WithMarkerVariant 设置变体(默认 MarkerDefault)。
func WithMarkerVariant(v MarkerVariant) MarkerOption {
	return func(m *Marker) { m.variant = v }
}

// WithMarkerIcon 在行首放一个 16px 图标(lucide path)。
func WithMarkerIcon(icon string) MarkerOption {
	return func(m *Marker) { m.icon = icon }
}

// Marker 会话里的标记行: 状态提示、系统备注、带边框的行、居中分隔标签。
//
// 与 shadcn marker.tsx 对齐:
//
//   - 默认单行、gap-8、text-sm muted-foreground;
//
//   - border 变体在底部画 1px 边框并留 8px 下内边距;
//
//   - separator 变体把内容居中, 两侧画 1px 横线(线长自动吃到剩余宽度)。
//
//     m := ui.NewMarker(body.Handle, ui.WithMarkerIcon(icons.IconCheck))
//     xui.ItemFill(m)
//     m.Content("Explored 4 files")
type Marker struct {
	*widget.LayoutEle
	variant MarkerVariant
	icon    string
}

// NewMarker 创建标记行。宽铺满、高自撑。
func NewMarker(x, y, cx, cy int32, hParent int, opts ...MarkerOption) *Marker {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	m := &Marker{LayoutEle: ele}
	for _, o := range opts {
		o(m)
	}
	ele.EnableHorizon(true)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(markerGap)
	ele.SetAlignV(xcc.Layout_Align_Center)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	if m.variant == MarkerSeparator {
		ele.SetAlignH(xcc.Layout_Align_Center)
	}
	if m.variant == MarkerBorder {
		xc.XEle_SetPadding(ele.Handle, 0, 0, 0, markerBorderPad)
	}
	if m.icon != "" {
		NewMarkerIcon(ele.Handle, m.icon)
	}
	ele.AddEvent_Paint(m.onPaint)
	xui.AttachWindow(ele.Handle)
	return m
}

// Content 添加内容文字: separator 变体按文字量宽(两侧留出横线), 其余变体铺满并截断。
func (m *Marker) Content(text string) *chatText {
	if m == nil {
		return nil
	}
	var c *chatText
	if m.variant == MarkerSeparator {
		c = newChatFitText(m.Handle, text, chatTextSize)
	} else {
		c = newChatFillText(m.Handle, text, chatTextSize)
	}
	if c == nil {
		return nil
	}
	c.SetMuted()
	return c
}

// NewMarkerIcon 创建一个 16px 的弱化色图标(放在标记行行首)。
func NewMarkerIcon(hParent int, icon string) *chatIcon {
	c := newChatIcon(hParent, icon, markerIconSize)
	if c == nil {
		return nil
	}
	c.SetColorFunc(func(t *xui.Theme) xui.Color { return t.MutedForeground })
	return c
}

func (m *Marker) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	switch m.variant {
	case MarkerBorder:
		p.Line(p.Rc.Left, p.Rc.Bottom-1, p.Rc.Right, p.Rc.Bottom-1, t.Border, 1)
	case MarkerSeparator:
		m.paintSeparatorLines(p, t)
	}
	return 0
}

// paintSeparatorLines 在内容两侧画横线: 从行首到第一个子项、从最后一个子项到行尾,
// 各留 4px 间距, 垂直居中。
func (m *Marker) paintSeparatorLines(p *xui.Painter, t *xui.Theme) {
	// 子项矩形要从窗口坐标换算回元素局部坐标, 才能和 p.Rc 一起画:
	// 直接拿 XEle_GetWndClientRect 的值去 p.Line, 线会落在元素外被裁掉(实测)。
	var own xc.RECT
	xc.XEle_GetWndClientRect(m.Handle, &own)
	ox := own.Left
	n := xc.XEle_GetChildCount(m.Handle)
	var firstL, lastR int32
	found := false
	for i := int32(0); i < n; i++ {
		h := xc.XEle_GetChildByIndex(m.Handle, i)
		if h == 0 || !xc.XWidget_IsShow(h) {
			continue
		}
		var rc xc.RECT
		xc.XEle_GetWndClientRect(h, &rc)
		l, r := rc.Left-ox, rc.Right-ox
		if !found {
			firstL = l
			found = true
		}
		lastR = r
	}
	if !found {
		return
	}
	y := p.Rc.Top + (p.Rc.Bottom-p.Rc.Top)/2
	if firstL-markerLineInset > p.Rc.Left {
		p.Line(p.Rc.Left, y, firstL-markerLineInset, y, t.Border, 1)
	}
	if p.Rc.Right > lastR+markerLineInset {
		p.Line(lastR+markerLineInset, y, p.Rc.Right, y, t.Border, 1)
	}
}
