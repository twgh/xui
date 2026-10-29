package ui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
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

// TabsActivation 决定"键盘焦点移动"是否等于"切换标签", 对应 radix Tabs 的 activationMode。
type TabsActivation int

const (
	TabsAutomatic TabsActivation = iota // 焦点移到哪个标签就激活哪个(radix 默认)
	TabsManual                          // 焦点移动只挪焦点, Enter/Space 才激活
)

// 标签栏尺寸(逻辑像素, 逐条对应 tabs.tsx 的类)。
const (
	tabsH     int32 = 32 // 横向列表高: h-8
	tabsPad   int32 = 3  // 列表内边距: p-[3px](同时给焦点环留位)
	tabsVTabH int32 = 24 // 纵向单个标签高: text-sm 行高 20 + py-0.5×2
	tabsTextX int32 = 6  // 纵向标签的文字左边距: px-1.5

	// 行内前置图标: [&_svg]:size-4 + gap-1.5 + has-data-[icon=inline-start]:pl-1
	tabsIcon     int32 = 16
	tabsIconGap  int32 = 6
	tabsIconPadX int32 = 4

	// line 变体的 gap-1: 标签之间留 4px(等分宽度时先扣掉)。
	tabsLineGap int32 = 4

	// 换行模式: 单标签的舒适最小宽(3 个中文字 + 左右留白)与行间距。
	// 可用宽装不下时按"行数最少、每行数量均衡"换行, 不去压窄标签。
	tabsWrapMinW int32 = 72
	tabsRowGap   int32 = 4
)

// TabsItem 是单个标签的内容与状态, 对应 TabsTrigger 的 children 与 disabled。
type TabsItem struct {
	Text     string // 标签文字
	Icon     string // 行内前置图标(path 或整段 SVG, 用法同 xui.Painter.Icon); 空 = 纯文字
	Disabled bool   // 单个标签禁用(disabled:opacity-50; 不可点击、不被方向键选中)
}

// Tabs 自绘标签栏(等价 shadcn 的 Tabs + TabsList + TabsTrigger)。
// 载体是一个原生 Element(只画列表底), 每个标签是一个原生 Button(行为层),
// 外观全部由本组件绘制。
//
// 与 shadcn tabs.tsx(radix-nova) 对齐:
//   - 列表: rounded-lg p-[3px] h-8(纵向为 h-fit + 标签堆叠), default 变体 bg-muted,
//     line 变体 bg-transparent + gap-1;
//   - 标签: 横向等宽(flex-1)居中, 纵向 w-full 左对齐(justify-start), rounded-md,
//     未选 text-foreground/60 hover 到 text-foreground, 选中 bg-background + text-foreground;
//     可带行内前置图标(size-4 + gap-1.5, 有图标时左内边距 4);
//   - line 变体选中项: 横向在列表底边画 h-0.5 下划线, 纵向在标签右侧画 w-0.5 指示条;
//   - focus-visible: 整块列表是**一个** Tab 站点(radix 的 roving focus), 进入后
//     ←/→(纵向 ↑/↓) 移动焦点、Home/End 首末、automatic 模式移动即切换、
//     manual 模式 Enter/Space 才切换; 焦点环画在当前标签上。
//
// 注意: 标签内容是面板 —— 用 BindPanel 交给组件显隐, 或自己用 OnChange 切换。
type Tabs struct {
	*widget.Element
	list  []*widget.Button
	items []TabsItem

	active   int32
	focus    int32
	variant  TabsVariant
	manual   bool
	vertical bool
	wrap     bool

	values   []string
	panels   map[int32]int // 标签索引 -> 面板元素句柄
	disabled []int32       // 构造选项里声明的禁用标签, 等标签建好再落地

	hasFocus bool
	keyID    int
	hWindow  int

	cb  func(index int32)
	vcb func(value string)

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

// WithTabsActivation 设置键盘焦点移动是否直接切换标签(默认 TabsAutomatic)。
func WithTabsActivation(a TabsActivation) TabsOption {
	return func(t *Tabs) { t.manual = a == TabsManual }
}

// WithTabsManual 等价 WithTabsActivation(TabsManual): 方向键只挪焦点, Enter/Space 才切换。
func WithTabsManual() TabsOption { return func(t *Tabs) { t.manual = true } }

// WithTabsDisabledAt 禁用若干标签(等价在 TabsTrigger 上加 disabled)。
func WithTabsDisabledAt(indexes ...int32) TabsOption {
	return func(t *Tabs) { t.disabled = append(t.disabled, indexes...) }
}

// WithTabsValues 给每个标签配一个字符串值(radix 的 value), 配合
// SetActiveValue / ActiveValue / OnValueChange 使用。
func WithTabsValues(values ...string) TabsOption {
	return func(t *Tabs) { t.values = append([]string(nil), values...) }
}

// WithTabsWrap 让横向标签在宽度装不下时自动换行(每行均衡分配, 标签高度不变)。
// 开启后列表宽度应交给布局(如 xui.ItemFill): 组件按实际宽度重排, 高度随之增加;
// 纵向标签忽略该选项。
func WithTabsWrap() TabsOption { return func(t *Tabs) { t.wrap = true } }

// NewTabs 创建标签栏。cx 是列表宽度: 横向时标签等宽平分, 纵向时是标签宽度(高度自适应为 n 行)。
// 开启 WithTabsWrap 后, 横向列表按布局给的实际宽度换行, cx 只作为首帧宽度。
func NewTabs(x, y, cx int32, hParent int, labels []string, opts ...TabsOption) *Tabs {
	items := make([]TabsItem, len(labels))
	for i, s := range labels {
		items[i] = TabsItem{Text: s}
	}
	return NewTabsItems(x, y, cx, hParent, items, opts...)
}

// NewTabsItems 创建带图标/禁用状态的标签栏(等价 NewTabs, 但每个标签是一个 TabsItem)。
func NewTabsItems(x, y, cx int32, hParent int, items []TabsItem, opts ...TabsOption) *Tabs {
	tb := &Tabs{items: append([]TabsItem(nil), items...)}
	for _, o := range opts { // 先应用配置: 取向决定列表尺寸
		o(tb)
	}
	for _, i := range tb.disabled {
		if i >= 0 && i < int32(len(tb.items)) {
			tb.items[i].Disabled = true
		}
	}
	count := int32(len(tb.items))
	if count <= 0 {
		count = 1
	}
	cy := tabsH
	if tb.vertical {
		cy = int32(len(tb.items))*tabsVTabH + tabsPad*2
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
	el.EnableDrawBorder(false)
	el.AddEvent_Paint(tb.onPaintList)
	// 整块列表是一个 Tab 站点(radix 的 roving focus): 标签自己不再是 Tab 站点,
	// 进入列表后用方向键在标签间移动(见 installKeys)。
	el.EnableFocus(true)
	el.EnableSwitchFocus(true)
	el.EnableDrawFocus(false)
	el.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		tb.hasFocus = true
		tb.focus = tb.nearestEnabled(tb.active)
		tb.Redraw(false)
		return 0
	})
	el.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		tb.hasFocus = false
		tb.Redraw(false)
		return 0
	})
	// 点在标签之间的空隙上: 焦点不会离开列表, 但要立刻退出 focus-visible 状态
	el.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		tb.Redraw(false)
		return 0
	})

	tabW := tb.tabWidth(cx, count)
	for i := range tb.items {
		idx := int32(i)
		var bx, by, bw, bh int32
		if tb.vertical {
			bx, by = tabsPad, tabsPad+idx*tabsVTabH
			bw, bh = cx-tabsPad*2, tabsVTabH
		} else {
			bx, by = tabsPad+idx*(tabW+tb.tabGap()), tabsPad
			bw, bh = tabW, tabsH-tabsPad*2
		}
		// 标签是列表的子元素, 坐标相对列表
		btn := widget.NewButton(bx, by, bw, bh, tb.items[i].Text, el.Handle)
		if btn == nil {
			continue
		}
		// 标签不再是独立 Tab 站点: 焦点归列表, 键盘导航由组件自己搬(见 installKeys)。
		btn.EnableFocus(false)
		btn.EnableSwitchFocus(false)
		btn.EnableBkTransparent(true) // 灭掉原生外观, 外观在 onPaintTab
		btn.SetText("")               // 文本自绘(要与背景同一图层, 便于选中态换色)
		if tb.items[i].Disabled {
			btn.Enable(false)
		}
		btn.AddEvent_Paint(tb.onPaintTab)
		btn.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			// 鼠标点击后不再匹配 focus-visible: 焦点环画在**列表**上(onPaintList), 而点击
			// 已选中的标签不会改变 active、也就不会触发列表重绘 —— 不在这里主动重绘列表,
			// 环会留在屏幕上, 表现成"点完反而冒出一个矩形框"。
			xui.NotePointerPress()
			tb.Redraw(false)
			return 0
		})
		btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			tb.onClick(idx)
			return 0
		})
		tb.list = append(tb.list, btn)
	}
	if tb.active < 0 || tb.active >= int32(len(tb.list)) {
		tb.active = 0
	}
	tb.focus = tb.active
	if tb.wrap && !tb.vertical {
		el.AddEvent_Size(tb.onSize)
		tb.scheduleWrap()
	}
	tb.installKeys()
	xui.AttachWindow(el.Handle)
	return tb
}

// Active 返回当前选中项索引。
func (t *Tabs) Active() int32 { return t.active }

// SetActive 设置选中项并重绘(禁用项也可程序化选中, 同 radix 的受控 value)。
func (t *Tabs) SetActive(i int32) *Tabs {
	if t == nil || i < 0 || i >= int32(len(t.items)) || i == t.active {
		return t
	}
	t.active = i
	t.focus = i
	t.syncPanels()
	t.Redraw(false)
	t.notify(i)
	return t
}

// Activate 是 SetActive 的别名, 便于与 radix 的触发语义对齐。
func (t *Tabs) Activate(i int32) *Tabs { return t.SetActive(i) }

// OnChange 注册选中项变化回调。
func (t *Tabs) OnChange(fn func(index int32)) *Tabs {
	t.cb = fn
	return t
}

// ActiveValue 返回当前选中项的字符串值(配合 WithTabsValues; 未配置时返回空串)。
func (t *Tabs) ActiveValue() string {
	if t == nil || int(t.active) < 0 || int(t.active) >= len(t.values) {
		return ""
	}
	return t.values[t.active]
}

// SetActiveValue 按字符串值选中标签(找不到时无操作)。
func (t *Tabs) SetActiveValue(v string) *Tabs {
	if t == nil {
		return t
	}
	for i, s := range t.values {
		if s == v {
			return t.SetActive(int32(i))
		}
	}
	return t
}

// OnValueChange 注册字符串值变化回调(需配合 WithTabsValues)。
func (t *Tabs) OnValueChange(fn func(value string)) *Tabs {
	t.vcb = fn
	return t
}

// BindPanel 把面板元素与标签绑定: 切换时显示对应面板、隐藏其它面板并重排父级。
// 未绑定的索引不影响任何元素。传 hEle = 0 表示解除该索引的绑定。
func (t *Tabs) BindPanel(index int32, hEle int) *Tabs {
	if t == nil {
		return t
	}
	if t.panels == nil {
		t.panels = map[int32]int{}
	}
	if hEle == 0 {
		delete(t.panels, index)
		return t
	}
	t.panels[index] = hEle
	xc.XWidget_Show(hEle, index == t.active)
	t.reflowPanels()
	return t
}

// SetText 运行时改标签文字(会重量宽, 横向等分布局不受影响)。
func (t *Tabs) SetText(index int32, text string) *Tabs {
	if t == nil || index < 0 || index >= int32(len(t.items)) {
		return t
	}
	t.items[index].Text = text
	t.Redraw(true)
	return t
}

// SetIcon 运行时改标签的行内前置图标(传空串 = 去掉图标)。
func (t *Tabs) SetIcon(index int32, icon string) *Tabs {
	if t == nil || index < 0 || index >= int32(len(t.items)) {
		return t
	}
	t.items[index].Icon = icon
	t.Redraw(true)
	return t
}

// SetDisabled 禁用/启用单个标签: 禁用后不可点击、不被方向键选中、整体 50% 透明。
func (t *Tabs) SetDisabled(index int32, disabled bool) *Tabs {
	if t == nil || index < 0 || index >= int32(len(t.items)) {
		return t
	}
	t.items[index].Disabled = disabled
	if b := t.list[index]; b != nil {
		b.Enable(!disabled)
	}
	if disabled && t.focus == index {
		t.focus = t.nearestEnabled(index)
	}
	t.redrawAll()
	return t
}

// Disabled 报告某个标签是否禁用。
func (t *Tabs) Disabled(index int32) bool {
	return t != nil && index >= 0 && index < int32(len(t.items)) && t.items[index].Disabled
}

// onClick 处理标签点击: 禁用项不响应; 顺带把键盘焦点交给整个列表(roving focus 的进入点)。
func (t *Tabs) onClick(idx int32) {
	if t.items[idx].Disabled {
		return
	}
	t.focus = idx
	if t.hWindow != 0 && !t.hasFocus {
		xc.XWnd_SetFocusEle(t.hWindow, t.Handle)
	}
	t.SetActive(idx)
}

// tabGap 返回列表内标签间距: line 变体 gap-1, default 无间距。
func (t *Tabs) tabGap() int32 {
	if t.variant == TabsLine {
		return tabsLineGap
	}
	return 0
}

// tabWidth 按可用宽与标签数算等宽标签宽(扣掉标签间距)。
func (t *Tabs) tabWidth(cx, count int32) int32 {
	if count <= 0 {
		count = 1
	}
	w := (cx - tabsPad*2 - t.tabGap()*(count-1)) / count
	if w < 1 {
		w = 1
	}
	return w
}

// installKeys 挂键盘导航。必须挂**窗口级**: 列表开了 EnableSwitchFocus(Tab 才进得来)后,
// xcgui 会把 ←/→ 当成"切换焦点"并在元素 KeyDown 之前执行(见 DESIGN 第八节 152)。
func (t *Tabs) installKeys() {
	// 用 xui.HostWindowOf: 标签栏通常在 ScrollArea 里, 直接映射 HWND 会拿到无效值(见 DESIGN 156)
	t.hWindow = xui.HostWindowOf(t.Handle)
	if t.hWindow == 0 {
		return
	}
	w := window.NewByHandle(t.hWindow)
	if w == nil {
		return
	}
	t.keyID = w.AddEvent_KeyDown(func(hWin int, wParam, lParam uintptr, pbHandled *bool) int {
		if xc.XWnd_GetFocusEle(hWin) != t.Handle {
			return 0
		}
		return t.onKeyDown(wParam, pbHandled)
	})
	t.Element.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if t.keyID > 0 {
			if xc.XC_IsHXCGUI(t.hWindow, xcc.XC_WINDOW) {
				w.RemoveEvent(xcc.WM_KEYDOWN, t.keyID)
			}
			t.keyID = 0
		}
		return 0
	})
}

// onKeyDown 处理键盘: 方向键在可用标签间移动焦点(automatic 下同时切换), Home/End 首末,
// Enter/Space 在 manual 模式下激活, Tab 由框架移出列表。
func (t *Tabs) onKeyDown(wParam uintptr, pbHandled *bool) int {
	if len(t.list) == 0 {
		return 0
	}
	switch wParam {
	case 0x25, 0x26: // VK_LEFT / VK_UP
		t.focusMove(-1)
	case 0x27, 0x28: // VK_RIGHT / VK_DOWN
		t.focusMove(1)
	case 0x24, 0x23: // VK_HOME / VK_END
		t.focusToEdge(wParam == 0x23)
	case 0x0D, 0x20: // VK_RETURN / VK_SPACE
		if !t.items[t.focus].Disabled {
			t.SetActive(t.focus)
		}
	default:
		return 0
	}
	*pbHandled = true
	return 0
}

// focusMove 把焦点往前后挪一个可用标签; 到端点不循环(radix Tabs 默认 loop=false)。
// automatic 模式下焦点落定即切换, manual 模式只挪焦点。
func (t *Tabs) focusMove(step int32) {
	if i := tabsSeek(int32(len(t.items)), t.focus+step, step, t.enabledAt); i >= 0 {
		t.setFocusIndex(i)
	}
}

// focusToEdge 把焦点挪到第一个/最后一个可用标签(Home/End)。
func (t *Tabs) focusToEdge(last bool) {
	step, from := int32(1), int32(0)
	if last {
		step, from = -1, int32(len(t.items))-1
	}
	if i := tabsSeek(int32(len(t.items)), from, step, t.enabledAt); i >= 0 {
		t.setFocusIndex(i)
	}
}

// enabledAt 报告某个标签是否可作为键盘焦点落点(禁用项跳过, 同 radix 的 disabled trigger)。
func (t *Tabs) enabledAt(i int32) bool {
	return i >= 0 && i < int32(len(t.items)) && !t.items[i].Disabled
}

// tabsSeek 从 from 起按 step 方向(±1)找第一个满足 ok 的下标, 走完没找到返回 -1。
// 抽成纯函数是为了能单测"跳过禁用标签 / 到端点不循环"的键盘逻辑(不需要真实元素)。
func tabsSeek(n, from, step int32, ok func(int32) bool) int32 {
	if step == 0 || n <= 0 {
		return -1
	}
	for i := from; i >= 0 && i < n; i += step {
		if ok(i) {
			return i
		}
	}
	return -1
}

func (t *Tabs) setFocusIndex(i int32) {
	if t.focus == i {
		return
	}
	t.focus = i
	if t.manual {
		t.Redraw(false)
		return
	}
	// automatic: 焦点移动即激活(radix activationMode="automatic")。
	t.SetActive(i)
}

// nearestEnabled 从 i 起找一个可用标签(优先自己, 再往后, 最后往前); 全禁用时返回 i。
func (t *Tabs) nearestEnabled(i int32) int32 {
	if i < 0 || i >= int32(len(t.items)) {
		return i
	}
	if !t.items[i].Disabled {
		return i
	}
	for j := i + 1; j < int32(len(t.items)); j++ {
		if !t.items[j].Disabled {
			return j
		}
	}
	for j := i - 1; j >= 0; j-- {
		if !t.items[j].Disabled {
			return j
		}
	}
	return i
}

func (t *Tabs) notify(i int32) {
	if t.cb != nil {
		t.cb(i)
	}
	if t.vcb != nil && i >= 0 && int(i) < len(t.values) {
		t.vcb(t.values[i])
	}
}

// syncPanels 按当前选中项显隐面板并重排(面板显隐改变布局高度)。
func (t *Tabs) syncPanels() {
	if len(t.panels) == 0 {
		return
	}
	parent := 0
	for idx, h := range t.panels {
		if h == 0 || !xc.XC_IsHXCGUI(h, xcc.XC_ELE) {
			continue
		}
		xc.XWidget_Show(h, idx == t.active)
		if parent == 0 {
			parent = xc.XWidget_GetParentEle(h)
		}
	}
	if parent != 0 {
		xc.XEle_AdjustLayout(parent)
	}
	t.redrawAll()
}

// reflowPanels 在绑定面板时按当前状态重排一次父级。
func (t *Tabs) reflowPanels() {
	parent := 0
	for _, h := range t.panels {
		if h != 0 && xc.XC_IsHXCGUI(h, xcc.XC_ELE) {
			parent = xc.XWidget_GetParentEle(h)
			break
		}
	}
	if parent != 0 {
		xc.XEle_AdjustLayout(parent)
	}
	t.redrawAll()
}

// redrawAll 整窗重绘: 只重绘列表会留下旧位置/旧显隐的残影(标签换行、面板切换都踩过)。
func (t *Tabs) redrawAll() {
	if hw := xui.HostWindowOf(t.Handle); hw != 0 {
		xc.XWnd_Redraw(hw, false)
	}
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
	gap := t.tabGap()
	rows, perRow, tabW := tabsWrapGrid(inner, int32(len(t.list)), gap)
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
		btn.SetRectEx(tabsPad+col*(tabW+gap), tabsPad+row*(tabsH+tabsRowGap), tabW, tabsH-tabsPad*2,
			false, xcc.AdjustLayout_No)
	}

	h := rows*tabsH + (rows-1)*tabsRowGap
	heightChanged := t.GetHeight() != h
	if heightChanged {
		t.SetHeight(h)
		t.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	}
	t.redrawAll()
	if heightChanged {
		t.reflowAncestors()
	}
}

// tabsWrapGrid 计算换行网格: 行数取最少(每行最多 inner/tabsWrapMinW 个),
// 再把标签数平均到各行; 返回每行个数与等宽标签宽(gap 是行内标签间距)。
// 标签少/宽不足时返回的 tabW 可能为 0。
func tabsWrapGrid(inner, n, gap int32) (rows, perRow, tabW int32) {
	if inner <= 0 || n <= 0 {
		return 0, 0, 0
	}
	perRowMax := inner / tabsWrapMinW
	if perRowMax < 1 {
		perRowMax = 1
	}
	rows = (n + perRowMax - 1) / perRowMax
	perRow = (n + rows - 1) / rows
	tabW = (inner - gap*(perRow-1)) / perRow
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

// onPaintList 绘制列表底与"画在标签外侧"的三个元素:
// 焦点环(ring-3 需要 3px 外扩空间)、line 变体的指示条(横向下划线 / 纵向右侧竖条)。
func (t *Tabs) onPaintList(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()

	radius := th.RadiusLg() // rounded-lg; line 变体 data-[variant=line]:rounded-none
	if t.variant == TabsDefault {
		p.FillRound(&p.Rc, th.Muted, radius)
	}

	for i, btn := range t.list {
		if btn == nil {
			continue
		}
		rc := btn.GetRectEx()
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
		// focus-visible:ring-3 ring-ring/50 —— 只画在键盘焦点所在的那个标签上。
		// 必须画**环带**(外扩的 3px 描边)而不是整块填充: 未选中的标签是透明的,
		// 填充色的内圈会从标签底下透出来, 变成"整块发灰"。
		if t.hasFocus && int32(i) == t.focus && xui.FocusedByKeyboard() {
			halo := xc.RECT{
				Left:   rc.Left - 2,
				Top:    rc.Top - 2,
				Right:  rc.Right + 2,
				Bottom: rc.Bottom + 2,
			}
			p.StrokeRound(&halo, th.Ring.WithAlpha(128), th.RadiusMd()+2, 3)
		}
	}
	return 0
}

// onPaintTab 绘制单个标签: 选中态底色 / 图标 + 文本 / focus-visible 边框。
func (t *Tabs) onPaintTab(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()
	st := xui.StateOf(hEle)

	idx := t.indexOf(hEle)
	if idx < 0 || idx >= int32(len(t.items)) {
		return 0
	}
	it := t.items[idx]
	active := idx == t.active
	disabled := it.Disabled || st.Disabled
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
	if t.hasFocus && !disabled && idx == t.focus && xui.FocusedByKeyboard() {
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
	if disabled {
		// disabled:opacity-50 是组级透明: 文字/图标按宿主底色对半预混(同 Button, 见 DESIGN 61)
		color = color.Mix(th.Muted, 0.5)
	}

	f := xui.FontHandle(xui.FontSpec{
		Family: th.FontDefault.Family,
		Size:   th.FontDefault.Size,
		// text-sm font-medium(500): 中文回退字体没有 Medium 字重, 浏览器按常规渲染,
		// 这里若用 Bold(700) 会明显比 shadcn 粗(实测对比)。
	})
	if f != 0 {
		p.Draw.SetFont(f)
	}

	// 内容宽度: 图标(size-4)+ gap-1.5 + 文字; 横向整体居中, 纵向左对齐(px-1.5 / pl-1)。
	var tw int32
	if it.Text != "" && f != 0 {
		tw = xui.TextWidth(it.Text, f)
	}
	content := tw
	hasIcon := it.Icon != ""
	if hasIcon {
		content += tabsIcon
		if it.Text != "" {
			content += tabsIconGap
		}
	}
	cy := (p.Rc.Top + p.Rc.Bottom) / 2
	x := p.Rc.Left + (p.Rc.Right-p.Rc.Left-content)/2
	if t.vertical {
		x = p.Rc.Left + tabsTextX
		if hasIcon {
			x = p.Rc.Left + tabsIconPadX // has-data-[icon=inline-start]:pl-1
		}
	} else if hasIcon && x < p.Rc.Left+tabsIconPadX {
		x = p.Rc.Left + tabsIconPadX
	}
	if hasIcon {
		irc := xc.RECT{Left: x, Top: cy - tabsIcon/2, Right: x + tabsIcon, Bottom: cy + tabsIcon/2}
		p.Icon(it.Icon, &irc, color, 2)
		x += tabsIcon + tabsIconGap
	}
	if it.Text != "" {
		dy := textVDy(it.Text)
		rcF := xc.RECTF{
			Left:   float32(x),
			Top:    float32(p.Rc.Top) - dy,
			Right:  float32(p.Rc.Right),
			Bottom: float32(p.Rc.Bottom) - dy,
		}
		p.TextF(it.Text, &rcF, color, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	}
	return 0
}

func (t *Tabs) indexOf(hEle int) int32 {
	for i, b := range t.list {
		if b != nil && b.Handle == hEle {
			return int32(i)
		}
	}
	return -1
}
