package ui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// TabsOrientation 标签栏取向, 对应 shadcn tabs.tsx 的 orientation。
type TabsOrientation int

const (
	TabsHorizontal TabsOrientation = iota // 标签横向排列, 列表 h-8
	TabsVertical                          // 标签纵向堆叠, 列表 h-fit
)

// TabsVariant 是标签栏的视觉变体, 对应 shadcn tabs.tsx 的 TabsList variant。
type TabsVariant int

const (
	TabsDefault TabsVariant = iota // bg-muted 的胶囊列表, 选中项是 bg-background 的白色小卡片
	TabsLine                       // 透明列表, 选中项只加下划线
)

// 标签栏尺寸(逻辑像素, 逐条对应 tabs.tsx 的类)。
const (
	tabsH     int32 = 32 // 横向列表高: h-8
	tabsPad   int32 = 3  // 列表内边距: p-[3px](同时给焦点环留位)
	tabsVTabH int32 = 24 // 纵向单个标签高: text-sm 行高 20 + py-0.5×2
	tabsTextX int32 = 6  // 纵向标签的文字左边距: px-1.5

	// 换行模式: 单标签的舒适最小宽(3 个中文字 + 左右留白)与行间距。
	// 可用宽装不下时按"行数最少、每行数量均衡"换行, 不去压窄标签。
	tabsWrapMinW int32 = 72
	tabsRowGap   int32 = 4
)

// Tabs 自绘标签栏(等价 shadcn 的 Tabs + TabsList + TabsTrigger)。
// 载体是一个原生 Element(只画列表底), 每个标签是一个原生 Button(行为层),
// 外观全部由本组件绘制。
//
// 与 shadcn tabs.tsx(radix-nova) 对齐:
//   - 列表: rounded-lg p-[3px] h-8(纵向为 h-fit + 标签堆叠), default 变体 bg-muted,
//     line 变体 bg-transparent;
//   - 标签: 横向等宽(flex-1)居中, 纵向 w-full 左对齐(justify-start), rounded-md,
//     未选 text-foreground/60 hover 到 text-foreground, 选中 bg-background + text-foreground;
//   - line 变体选中项: 横向在列表底边画 h-0.5 下划线, 纵向在标签右侧画 w-0.5 指示条;
//   - focus-visible: 标签边框变 Ring 色 + ring-3 ring-ring/50(环画在列表的绘制回调里,
//     因为标签自身客户区只有标签大小, 画不下外扩的环)。
//
// 注意: 标签内容是面板, 与本组件无关——用 OnChange 自行切换面板可见性。
type Tabs struct {
	*widget.Element
	list     []*widget.Button
	labels   []string
	active   int32
	variant  TabsVariant
	vertical bool
	wrap     bool
	cb       func(index int32)

	// 换行模式的状态: 上一次宽度(尺寸事件节流)与延迟重排标记。
	lastWrapW  int32
	wrapQueued bool
}

// TabsOption 标签栏配置项。
type TabsOption func(*Tabs)

// WithTabsVariant 设置视觉变体。
func WithTabsVariant(v TabsVariant) TabsOption { return func(t *Tabs) { t.variant = v } }

// WithTabsActive 设置初始选中项。
func WithTabsActive(i int32) TabsOption { return func(t *Tabs) { t.active = i } }

// WithTabsOrientation 设置取向(默认横向)。
func WithTabsOrientation(o TabsOrientation) TabsOption {
	return func(t *Tabs) { t.vertical = o == TabsVertical }
}

// WithTabsVertical 改为纵向堆叠(等价 WithTabsOrientation(TabsVertical))。
func WithTabsVertical() TabsOption { return func(t *Tabs) { t.vertical = true } }

// WithTabsWrap 让横向标签在宽度装不下时自动换行(每行均衡分配, 标签高度不变)。
// 开启后列表宽度应交给布局(如 xui.ItemFill): 组件按实际宽度重排, 高度随之增加;
// 纵向标签忽略该选项。
func WithTabsWrap() TabsOption { return func(t *Tabs) { t.wrap = true } }

// NewTabs 创建标签栏。cx 是列表宽度: 横向时标签等宽平分, 纵向时是标签宽度(高度自适应为 n 行)。
// 开启 WithTabsWrap 后, 横向列表按布局给的实际宽度换行, cx 只作为首帧宽度。
func NewTabs(x, y, cx int32, hParent int, labels []string, opts ...TabsOption) *Tabs {
	tb := &Tabs{labels: labels}
	for _, o := range opts { // 先应用配置: 取向决定列表尺寸
		o(tb)
	}
	count := int32(len(labels))
	if count <= 0 {
		count = 1
	}
	cy := tabsH
	if tb.vertical {
		cy = count*tabsVTabH + tabsPad*2
	}
	if cx <= 0 {
		cx = tabsVTabH * 8
	}

	el := widget.NewElement(x, y, cx, cy, hParent)
	if el == nil {
		return nil
	}
	tb.Element = el
	el.EnableBkTransparent(true)
	el.AddEvent_Paint(tb.onPaintList)

	tabW := (cx - tabsPad*2) / count
	for i, text := range labels {
		var bx, by, bw, bh int32
		if tb.vertical {
			bx, by = tabsPad, tabsPad+int32(i)*tabsVTabH
			bw, bh = cx-tabsPad*2, tabsVTabH
		} else {
			bx, by = tabsPad+int32(i)*tabW, tabsPad
			bw, bh = tabW, tabsH-tabsPad*2
		}
		// 标签是列表的子元素, 坐标相对列表
		btn := widget.NewButton(bx, by, bw, bh, text, el.Handle)
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
	if tb.wrap && !tb.vertical {
		el.AddEvent_Size(tb.onSize)
		tb.scheduleWrap()
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

// onSize 尺寸事件: 延迟到下一轮 UI 循环再换行, 避免在布局趟里改自己的高度。
func (t *Tabs) onSize(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
	t.scheduleWrap()
	return 0
}

func (t *Tabs) scheduleWrap() {
	if t == nil || t.wrapQueued {
		return
	}
	t.wrapQueued = true
	app.AfterFunc(0, func() {
		t.wrapQueued = false
		t.relayoutWrap()
	})
}

// relayoutWrap 按当前宽度重排换行标签:
//   - 行数取最少(maxPerRow = 可用宽 / 最小标签宽), 再把标签数平均到各行;
//   - 每行标签等宽、左对齐, 最后一行不足时右侧留空;
//   - 行数变化时同步元素高度与布局项高度, 并逐级通知祖先重排(Auto 高的卡片/页面才会让位)。
func (t *Tabs) relayoutWrap() {
	if t == nil || !t.wrap || t.vertical || t.Element == nil || len(t.list) == 0 {
		return
	}
	w := t.GetWidth()
	if w <= 0 || w == t.lastWrapW {
		return
	}
	inner := w - tabsPad*2
	if inner <= 0 {
		return
	}
	rows, perRow, tabW := tabsWrapGrid(inner, int32(len(t.list)))
	if tabW <= 0 {
		return
	}
	t.lastWrapW = w

	for i, btn := range t.list {
		if btn == nil {
			continue
		}
		row := int32(i) / perRow
		col := int32(i) % perRow
		btn.SetRectEx(tabsPad+col*tabW, tabsPad+row*(tabsH+tabsRowGap), tabW, tabsH-tabsPad*2,
			false, xcc.AdjustLayout_No)
	}

	h := rows*tabsH + (rows-1)*tabsRowGap
	heightChanged := t.GetHeight() != h
	if heightChanged {
		t.SetHeight(h)
		t.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	}
	// 位置与高度都改完再整窗重绘: 只重绘列表会留下旧位置的标签残影。
	if hwnd := xc.XWidget_GetHWND(t.Handle); hwnd != 0 {
		if hw := xc.XC_hWindowFromHWnd(hwnd); hw != 0 {
			xc.XWnd_Redraw(hw, false)
		}
	}
	if heightChanged {
		t.reflowAncestors()
	}
}

// tabsWrapGrid 计算换行网格: 行数取最少(每行最多 inner/tabsWrapMinW 个),
// 再把标签数平均到各行; 返回每行个数与等宽标签宽。标签少/宽不足时返回的 tabW 可能为 0。
func tabsWrapGrid(inner, n int32) (rows, perRow, tabW int32) {
	if inner <= 0 || n <= 0 {
		return 0, 0, 0
	}
	perRowMax := inner / tabsWrapMinW
	if perRowMax < 1 {
		perRowMax = 1
	}
	rows = (n + perRowMax - 1) / perRowMax
	perRow = (n + rows - 1) / rows
	tabW = inner / perRow
	return rows, perRow, tabW
}

// reflowAncestors 逐级通知祖先重排(同 Alert 的做法): 直接调父级对 ScrollArea 内容区
// 会把宽度打成 0, 遇到 LayoutFrame 必须交给框架自己排。
func (t *Tabs) reflowAncestors() {
	for p := xc.XWidget_GetParentEle(t.Handle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(p)
			break
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(pp)
			break
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_ELE) {
			xc.XEle_AdjustLayout(p)
			continue
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
		}
		break
	}
}

// onPaintList 绘制列表底与"画在标签外侧"的两个元素:
// 焦点环(ring-3 需要 3px 外扩空间)与 line 变体的指示条(横向下划线 / 纵向右侧竖条)。
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
			if t.vertical {
				// after:inset-y-0 after:-right-1 after:w-0.5 —— 贴在标签右侧
				line := xc.RECT{Left: rc.Right + 1, Top: rc.Top, Right: rc.Right + 3, Bottom: rc.Bottom}
				p.FillRound(&line, th.Foreground, 1)
			} else {
				// after:inset-x-0 after:bottom-[-5px] after:h-0.5 —— 贴在**当前行**的底边
				// (换行时不能再用列表底边, 否则下划线会跑到最后一行下面)
				line := xc.RECT{Left: rc.Left, Top: rc.Bottom + tabsPad - 2, Right: rc.Right, Bottom: rc.Bottom + tabsPad}
				p.FillRound(&line, th.Foreground, 1)
			}
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
		// data-active:bg-background; 暗色下是 data-active:bg-input/30 + border-input
		// (tabs.tsx) —— 暗色用 bg-background 会变成一块比列表更黑的方块(实测纠正)
		if th.IsDark() {
			// bg-input/30: 暗色 input 本身就是"白 15%", 再 /30 是乘不是覆盖
			p.FillRound(&p.Rc, th.Input.ScaleAlpha(0.30), radius) // bg-input/30
			p.StrokeRound(&p.Rc, th.Input, radius, 1)             // border-input(白 15%)
		} else {
			p.FillRound(&p.Rc, th.Background, radius)
		}
	}
	if st.Focused && xui.FocusedByKeyboard() {
		// focus-visible:border-ring
		p.StrokeRound(&p.Rc, th.Ring, radius, 1)
	}

	// 文本: 未选 text-foreground/60, 悬停或选中 text-foreground。
	// 悬停带过渡(tabs.tsx 的 transition-all); 选中态立即到位 —— 切页时其它标签
	// 未必重绘, 走过渡会留下颜色残影。
	color := xui.Fade(th.Foreground.WithAlpha(153), th.Foreground, xui.Hover(hEle, st.Hover))
	if active {
		color = th.Foreground
	}
	if st.Disabled {
		// disabled:opacity-50 组级透明 —— 文字核心保持原色, 只有底色变淡
		// (实测浏览器禁用按钮的白字核心 = 255, 深字核心接近纯黑)
		color = th.Foreground
	}
	if f := xui.FontHandle(xui.FontSpec{
		Family: th.FontDefault.Family,
		Size:   th.FontDefault.Size,
		// text-sm font-medium(500): 中文回退字体没有 Medium 字重, 浏览器按常规渲染,
		// 这里若用 Bold(700) 会明显比 shadcn 粗(实测对比)。
	}); f != 0 {
		p.Draw.SetFont(f)
	}
	// 横向 justify-center, 纵向 justify-start(左对齐 + px-1.5)
	rc := p.Rc
	align := xcc.TextAlignFlag_Center | xcc.TextAlignFlag_Vcenter
	if t.vertical {
		rc.Left += tabsTextX
		align = xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter
	}
	p.Text(t.labels[idx], &rc, color, align)
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
