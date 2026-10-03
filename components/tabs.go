package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// TabsVariant 是标签栏的视觉变体, 对应 shadcn tabs.tsx 的 TabsList variant。
type TabsVariant int

const (
	TabsDefault TabsVariant = iota // bg-muted 的胶囊列表, 选中项是 bg-background 的白色小卡片
	TabsLine                       // 透明列表, 选中项只加下划线
)

// tabsH 是列表高度(shadcn 为 h-8), tabsPad 是列表内边距(p-[3px], 同时给焦点环留位)。
const (
	tabsH   int32 = 32
	tabsPad int32 = 3
)

// Tabs 自绘标签栏(等价 shadcn 的 Tabs + TabsList + TabsTrigger)。
// 载体是一个原生 Element(只画列表底), 每个标签是一个原生 Button(行为层),
// 外观全部由本组件绘制。
//
// 与 shadcn tabs.tsx(radix-nova) 对齐:
//   - 列表: rounded-lg p-[3px] h-8, default 变体 bg-muted, line 变体 bg-transparent;
//   - 标签: 等宽(flex-1) rounded-md, 未选 text-foreground/60 hover 到 text-foreground,
//     选中 bg-background + text-foreground;
//   - line 变体选中项在列表底边画 h-0.5 的 bg-foreground 下划线;
//   - focus-visible: 标签边框变 Ring 色 + ring-3 ring-ring/50(环画在列表的绘制回调里,
//     因为标签自身客户区只有标签大小, 画不下外扩的环)。
//
// 注意: 标签内容是面板, 与本组件无关——用 OnChange 自行切换面板可见性。
type Tabs struct {
	*widget.Element
	list    []*widget.Button
	labels  []string
	active  int32
	variant TabsVariant
	cb      func(index int32)
}

// TabsOption 标签栏配置项。
type TabsOption func(*Tabs)

// WithTabsVariant 设置视觉变体。
func WithTabsVariant(v TabsVariant) TabsOption { return func(t *Tabs) { t.variant = v } }

// WithTabsActive 设置初始选中项。
func WithTabsActive(i int32) TabsOption { return func(t *Tabs) { t.active = i } }

// NewTabs 创建标签栏。cx 是列表宽度, 标签等宽平分(cx - p-[3px]x2)。
func NewTabs(x, y, cx int32, hParent int, labels []string, opts ...TabsOption) *Tabs {
	el := widget.NewElement(x, y, cx, tabsH, hParent)
	if el == nil {
		return nil
	}
	tb := &Tabs{Element: el, labels: labels}
	for _, o := range opts {
		o(tb)
	}
	el.EnableBkTransparent(true)
	el.AddEvent_Paint(tb.onPaintList)

	count := int32(len(labels))
	w := (cx - tabsPad*2) / count
	for i, text := range labels {
		// 标签是列表的子元素, 坐标相对列表
		off := int32(i) * w
		btn := widget.NewButton(tabsPad+off, tabsPad, w, tabsH-tabsPad*2, text, el.Handle)
		if btn == nil {
			continue
		}
		btn.EnableBkTransparent(true) // 灭掉原生外观, 外观在 onPaintTab
		btn.SetText("")               // 文本自绘(要与背景同一图层, 便于选中态换色)
		idx := int32(i)
		btn.AddEvent_Paint(tb.onPaintTab)
		btn.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			xui.NotePointerPress()
			return 0
		})
		btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			tb.SetActive(idx)
			return 0
		})
		tb.list = append(tb.list, btn)
	}
	if tb.active < 0 || tb.active >= int32(len(tb.list)) {
		tb.active = 0
	}
	xui.AttachWindow(el.Handle)
	return tb
}

// Active 返回当前选中项索引。
func (t *Tabs) Active() int32 { return t.active }

// SetActive 设置选中项并重绘。
func (t *Tabs) SetActive(i int32) *Tabs {
	if i < 0 || i >= int32(len(t.labels)) || i == t.active {
		return t
	}
	t.active = i
	t.Redraw(false)
	if t.cb != nil {
		t.cb(i)
	}
	return t
}

// OnChange 注册选中项变化回调。
func (t *Tabs) OnChange(fn func(index int32)) *Tabs {
	t.cb = fn
	return t
}

// onPaintList 绘制列表底与"画在标签外侧"的两个元素:
// 焦点环(ring-3 需要 3px 外扩空间)与 line 变体的下划线(位于列表底边)。
func (t *Tabs) onPaintList(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()

	radius := th.RadiusLg() // rounded-lg
	if t.variant == TabsDefault {
		p.FillRound(&p.Rc, th.Muted, radius)
	}

	for i, btn := range t.list {
		if btn == nil {
			continue
		}
		rc := btn.GetRectEx()
		st := xui.StateOf(btn.Handle)
		if int32(i) == t.active && t.variant == TabsLine {
			// 选中下划线: h-0.5 bg-foreground, 贴在列表底边
			line := xc.RECT{Left: rc.Left, Top: p.Rc.Bottom - 2, Right: rc.Right, Bottom: p.Rc.Bottom}
			p.FillRound(&line, th.Foreground, 1)
		}
		// focus-visible:ring-3 ring-ring/50
		if st.Focused && xui.FocusedByKeyboard() {
			halo := xc.RECT{
				Left:   rc.Left - tabsPad,
				Top:    rc.Top - tabsPad,
				Right:  rc.Right + tabsPad,
				Bottom: rc.Bottom + tabsPad,
			}
			p.FillRound(&halo, th.Ring.WithAlpha(128), th.RadiusMd()+tabsPad)
		}
	}
	return 0
}

// onPaintTab 绘制单个标签: 选中态底色 / 文本 / focus-visible 边框。
func (t *Tabs) onPaintTab(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()
	st := xui.StateOf(hEle)

	idx := t.indexOf(hEle)
	active := idx == t.active
	radius := th.RadiusMd() // rounded-md

	if active && t.variant == TabsDefault {
		p.FillRound(&p.Rc, th.Background, radius)
	}
	if st.Focused && xui.FocusedByKeyboard() {
		// focus-visible:border-ring
		p.StrokeRound(&p.Rc, th.Ring, radius, 1)
	}

	// 文本: 未选 text-foreground/60, 悬停或选中 text-foreground
	color := th.Foreground.WithAlpha(153)
	if active || st.Hover {
		color = th.Foreground
	}
	if st.Disabled {
		color = th.Foreground.WithAlpha(128) // disabled:opacity-50
	}
	if f := xui.FontHandle(xui.FontSpec{
		Family: th.FontDefault.Family,
		Size:   th.FontDefault.Size,
		Bold:   true, // text-sm font-medium
	}); f != 0 {
		p.Draw.SetFont(f)
	}
	p.Text(t.labels[idx], &p.Rc, color, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	return 0
}

func (t *Tabs) indexOf(hEle int) int32 {
	for i, b := range t.list {
		if b != nil && b.Handle == hEle {
			return int32(i)
		}
	}
	return 0
}
