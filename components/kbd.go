package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Kbd 的几何常量(逻辑像素), 对齐 shadcn kbd.tsx 的
// `h-5 min-w-5 rounded-sm px-1 text-xs font-medium`。
const (
	kbdHeight   int32 = 20
	kbdPadX     int32 = 4
	kbdMinW     int32 = 20
	kbdGroupGap int32 = 4
)

// Kbd 键帽: bg-muted + rounded-sm + text-xs, 用于展示快捷键。
//
// 高度固定 20, 宽度按文字量测(最小 20), 换主题/改文字时重新量宽。
// 文字颜色 muted-foreground; 纯展示, 不接收焦点与鼠标。
//
// 多个键帽成组用 NewKbdGroup:
//
//	g := ui.NewKbdGroup(row.Handle)
//	ui.NewKbd(g.Handle, "Ctrl")
//	ui.NewKbd(g.Handle, "K")
type Kbd struct {
	*widget.Element
	text string
}

// kbdFont 返回键帽字体: 主题字号 -2(与 Badge/tooltip 的 text-xs 同口径), medium。
func kbdFont(t *xui.Theme) int {
	spec := t.FontDefault
	spec.Size -= 2
	spec.Bold = true
	return xui.FontHandle(spec)
}

func kbdWidth(t *xui.Theme, text string) int32 {
	w := xui.TextWidth(text, kbdFont(t)) + kbdPadX*2
	if w < kbdMinW {
		w = kbdMinW
	}
	return w
}

// NewKbd 创建键帽并挂到 hParent(作为流式布局项)。
func NewKbd(hParent int, text string) *Kbd {
	k := &Kbd{text: text}
	w := kbdWidth(xui.Current(), text)
	ele := widget.NewElement(0, 0, w, kbdHeight, hParent)
	if ele == nil {
		return nil
	}
	k.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, kbdHeight)
	ele.AddEvent_Paint(k.onPaint)
	xui.AttachWindow(ele.Handle)
	xui.OnThemeChangeFor(ele.Handle, func(t *xui.Theme) {
		k.syncWidth(t)
		k.Redraw(false)
	})
	return k
}

// SetText 更新键帽文字并重新量宽。
func (k *Kbd) SetText(text string) *Kbd {
	if k == nil {
		return k
	}
	k.text = text
	k.syncWidth(xui.Current())
	k.Redraw(false)
	return k
}

func (k *Kbd) syncWidth(t *xui.Theme) {
	if k == nil || k.Element == nil {
		return
	}
	w := kbdWidth(t, k.text)
	k.SetWidth(w)
	k.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
}

func (k *Kbd) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	p.FillRound(&p.Rc, t.Muted, t.RadiusSm())
	if f := kbdFont(t); f != 0 {
		p.Draw.SetFont(f)
	}
	p.TextCenter(k.text, t.MutedForeground)
	return 0
}

// NewKbdGroup 创建键帽组: 水平排列、间距 4, 宽高按内容自撑。
// 返回普通 LayoutEle, 往里放 NewKbd 即可。
func NewKbdGroup(hParent int) *widget.LayoutEle {
	g := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if g == nil {
		return nil
	}
	g.EnableHorizon(true)
	g.EnableLayout(true)
	g.EnableBkTransparent(true)
	g.EnableAutoWrap(false)
	g.SetSpace(kbdGroupGap)
	g.SetAlignV(xcc.Layout_Align_Center)
	g.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	g.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return g
}
