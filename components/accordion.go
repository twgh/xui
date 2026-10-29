package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// accordion 几何(逻辑像素), 对齐 radix-nova 的 accordion.tsx:
// 触发区 py-2.5(text-sm 行高 20 + 上下各 10 = 40)、内容 pb-2.5;
// 上游没有水平内边距 —— 文字与所属卡片内容左缘对齐, 箭头贴触发区右缘。
const (
	accordionPadY    int32 = 10 // py-2.5 / 内容 pb-2.5
	accordionTrigger int32 = 40 // 触发区高
	accordionIconSz  int32 = 18 // size-4(16) + 2px 描边补偿, 同 Button/Toggle
	// accordionPadX 触发区左右内距。上游是 `border border-transparent` + 环画在**框外**,
	// 文字因此离环 1px 且不会被环压住; 我们的绘制被裁剪在元素内, 环只能画在里面, 所以
	// 用 4 的内距(环带 3 + 1)把文字/箭头让出来 —— 否则聚焦时环带会盖到字和图标上。
	accordionPadX int32 = 4
)

// AccordionType 手风琴的开合模式, 对应 radix Accordion.Root 的 type:
// single = 同一时间至多一项展开(且允许全部收起, 同 collapsible); multiple = 各自独立。
type AccordionType int

const (
	AccordionSingle   AccordionType = iota // 单开(默认, 同 shadcn 文档示例 type="single" collapsible)
	AccordionMultiple                      // 多开
)

// Accordion 手风琴: 竖向排列的若干折叠项, 每项 = 触发区 + 高度过渡的内容区。
//
// 与 shadcn accordion.tsx(radix-nova) 对齐:
//   - 触发区: 透明底、文字左对齐 text-sm、箭头贴右(展开 ChevronUp / 收起 ChevronDown,
//     上游就是两枚图标互换, 不做旋转动画); hover:underline; focus-visible 环
//     (border-ring + ring-3 ring-ring/50, rounded-lg); disabled:opacity-50;
//   - 条目间 1px 分隔线(Item 的 not-last:border-b, 末项没有);
//   - 内容: 展开高度过渡 200ms(复用 Collapsible), 内容底部 pb-2.5。
//
// 根容器宽必须确定(Fill): 放进卡片/行时用 xui.ItemFill 给它宽度(同 Collapsible)。
type Accordion struct {
	*widget.LayoutEle
	mode    AccordionType
	items   []*AccordionItem
	changes []func(values []string)
}

// AccordionOption 手风琴配置项。
type AccordionOption func(*Accordion)

// WithAccordionType 设置开合模式(默认 AccordionSingle)。
func WithAccordionType(t AccordionType) AccordionOption {
	return func(a *Accordion) { a.mode = t }
}

// WithAccordionMultiple 等价 WithAccordionType(AccordionMultiple)。
func WithAccordionMultiple() AccordionOption {
	return func(a *Accordion) { a.mode = AccordionMultiple }
}

// NewAccordion 创建手风琴根容器(竖向流式, 子项间距 0 —— 分隔线自带节奏)。
func NewAccordion(x, y, cx, cy int32, hParent int, opts ...AccordionOption) *Accordion {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	a := &Accordion{LayoutEle: ele}
	for _, o := range opts {
		o(a)
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false)
	ele.EnableBkTransparent(true)
	ele.SetSpace(0)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.AttachWindow(ele.Handle)
	return a
}

// AddItem 追加一个折叠项(触发区 + 内容容器 + 与上一项之间的分隔线), 返回条目。
//
//	value 是条目的标识(同 radix AccordionItem 的 value, 用于 Value/SetValue);
//	title 是触发区文字。要往内容区放东西用返回条目的 ContentStack:
//
//	it := acc.AddItem("item-1", "产品详情")
//	body := it.ContentStack(6)
//	xui.ItemFill(ui.NewLabel(0, 0, 0, 24, "内容", body.Handle))
//
// 注意内容区里的每个子项都要自己声明宽度(ItemFill / ItemWeight): 布局容器不会替
// 子项铺满, Label 的 cx 传 0 就是 0 宽 —— 忘了写会得到"展开后有高度但一片空白"。
func (a *Accordion) AddItem(value, title string, opts ...AccordionItemOption) *AccordionItem {
	if len(a.items) > 0 {
		// Item 的 not-last:border-b: 分隔线画在两项之间, 末项后面没有
		sep := NewSeparator(0, 0, 0, a.Handle)
		if sep != nil {
			xui.ItemFill(sep)
		}
	}
	it := &AccordionItem{acc: a, value: value}
	for _, o := range opts {
		o(it)
	}

	btn := NewButton(0, 0, 0, 0, title, a.Handle)
	if btn == nil {
		return nil
	}
	it.trigger = btn
	xui.ItemFill(btn)
	// 触发区高 40(py-2.5 + text-sm 行高 20): Button 的默认预设是 32+1(pressShift),
	// 这里按触发区语义固定高度, 不再走按钮预设(见 accordionTrigger)。
	btn.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, accordionTrigger)
	btn.SetHeight(accordionTrigger)
	btn.SetTypeEx(xcc.Button_Type_Check)
	btn.EnableBkTransparent(true)
	// 原生 Button 自带 1px 边框(Label 踩过, 见 DESIGN 109); 这里整条边都关掉,
	// 否则触发区底缘会多出一条 1.5 设备像素的线, 和下面的分隔线叠成"双线"。
	btn.SetBorderSize(0, 0, 0, 0)
	btn.EnableDrawBorder(false)
	btn.SetTextColor(0) // 文字自绘, 颜色随状态
	btn.AddEvent_Paint(it.onPaintTrigger)
	btn.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	// Check 按钮的点击以 Button_Check 事件到达(状态已翻好), 在这里同步开合。
	btn.AddEvent_Button_Check(func(hEle int, bCheck bool, pbHandled *bool) int {
		it.setOpen(bCheck)
		return 0
	})
	// 原生 Check 按钮不响应空格/回车(KeyDown 能收到但状态不变), 键盘激活自己补
	// (同 Toggle, 见 DESIGN 第八节 108)。
	btn.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		const vkReturn, vkSpace = 0x0D, 0x20
		if (wParam == vkReturn || wParam == vkSpace) && xc.XEle_IsEnable(hEle) {
			it.setOpen(!it.open)
			*pbHandled = true
		}
		return 0
	})

	coll := NewCollapsible(0, 0, 0, 0, a.Handle)
	if coll == nil {
		return nil
	}
	xui.ItemFill(coll)
	it.collapsible = coll

	a.items = append(a.items, it)
	if it.disabled {
		it.SetDisabled(true)
	}
	return it
}

// Items 返回全部折叠项。
func (a *Accordion) Items() []*AccordionItem { return a.items }

// Value 返回处于展开态的条目 value 列表(同 radix type=multiple 的 value 数组;
// single 模式下至多一个)。
func (a *Accordion) Value() []string {
	vals := make([]string, 0, len(a.items))
	for _, it := range a.items {
		if it.open {
			vals = append(vals, it.value)
		}
	}
	return vals
}

// SetValue 展开指定 value 的条目并收起其余(同 radix 的受控 value)。
// single 模式只认第一个匹配项; 没有匹配 = 全部收起。
func (a *Accordion) SetValue(values ...string) *Accordion {
	for _, it := range a.items {
		open := false
		for _, v := range values {
			if it.value == v {
				open = true
				break
			}
		}
		if a.mode == AccordionSingle && open {
			// single 模式只展开第一个匹配项, 后续匹配项保持收起
			values = nil
		}
		it.SetOpen(open)
	}
	return a
}

// OnChange 注册展开集合变化回调(用户点击、键盘或 SetValue 都会触发)。
func (a *Accordion) OnChange(fn func(values []string)) *Accordion {
	a.changes = append(a.changes, fn)
	return a
}

func (a *Accordion) notify() {
	vals := a.Value()
	for _, fn := range a.changes {
		fn(vals)
	}
}

// AccordionItem 手风琴的折叠项: 触发按钮 + 复用 Collapsible 的内容区。
type AccordionItem struct {
	acc         *Accordion
	value       string
	trigger     *Button
	collapsible *Collapsible
	open        bool
	disabled    bool
	body        *widget.LayoutEle // 内容区容器(ContentStack 返回的那个)
	space       int32             // 内容区子项间距(ContentStack 的参数)
	pad         *widget.Element   // 内容区末尾的 pb-2.5 占位(见 ensureBottomPad)
}

// AccordionItemOption 折叠项配置项。
type AccordionItemOption func(*AccordionItem)

// WithAccordionItemDisabled 禁用触发区(内容保持当前开合状态, 同 radix item disabled)。
func WithAccordionItemDisabled() AccordionItemOption {
	return func(it *AccordionItem) { it.disabled = true }
}

// Value 返回条目 value。
func (it *AccordionItem) Value() string { return it.value }

// Trigger 返回触发按钮(可以改文字: SetText)。
func (it *AccordionItem) Trigger() *Button { return it.trigger }

// Open 返回是否展开。
func (it *AccordionItem) Open() bool { return it != nil && it.open }

// SetOpen 设置展开状态(会触发根容器的 OnChange)。
func (it *AccordionItem) SetOpen(v bool) *AccordionItem {
	if it == nil {
		return it
	}
	it.setOpen(v)
	return it
}

// Toggle 反转展开状态(disabled 时无效)。
func (it *AccordionItem) Toggle() *AccordionItem {
	if it == nil || it.disabled {
		return it
	}
	it.setOpen(!it.open)
	return it
}

// SetDisabled 禁用/启用触发区。
func (it *AccordionItem) SetDisabled(d bool) *AccordionItem {
	if it == nil {
		return it
	}
	it.disabled = d
	if it.trigger != nil {
		it.trigger.Enable(!d)
		it.trigger.Redraw(false)
	}
	return it
}

// Disabled 返回触发区是否禁用。
func (it *AccordionItem) Disabled() bool { return it != nil && it.disabled }

// Content 返回内容容器(竖向流式, 宽铺满; 底部已带 pb-2.5 的 10px 间距)。
// 第一次调用时创建, 内容高度由 Collapsible 管理, 不要自己给它声明高度。
func (it *AccordionItem) Content() *widget.LayoutEle {
	if it == nil || it.collapsible == nil {
		return nil
	}
	return it.ContentStack(0)
}

// ContentStack 设置内容区子项间距并返回它(同 Card.ContentStack)。
func (it *AccordionItem) ContentStack(space int32) *widget.LayoutEle {
	if it == nil || it.collapsible == nil {
		return nil
	}
	body := it.collapsible.ContentStack(space)
	it.body, it.space = body, space
	// 又往内容里加了子项: 把 pb-2.5 占位挪回末尾(见 ensureBottomPad)。
	if it.pad != nil {
		it.pad.Destroy()
		it.pad = nil
		it.ensureBottomPad()
	}
	return body
}

// ensureBottomPad 在内容区**末尾**补一个透明占位, 补足 shadcn AccordionContent 内层
// div 的 pb-2.5(= 内容最后一行到分隔线之间 10px)。
//
// 三点都是实测踩出来的(见 DESIGN 第八节 111):
//
//	① 不能用外边距: 布局对 `Auto` 高度的父级**不计子项外边距**, 而动画把内容区锁成
//	   `Fixed(h)` 时那段外边距又确实占位 —— "动画目标 34 / 终态 Auto 24", 展开结束
//	   那一帧整条列表向上跳 10px。
//	② 必须等用户把内容加完再挂: 在 ContentStack 里就挂会排到**第一个**子项之前,
//	   于是触发区和描述之间空出一大截, 而描述下面紧贴分隔线。所以挂载点在开合时
//	   (只创建一次, 之后幂等)。
//	③ 高度要扣掉内容区的行间距(布局在相邻子项之间也插 spacer), 否则底部会比顶部宽
//	   出"间距"那么多。
func (it *AccordionItem) ensureBottomPad() {
	if it == nil || it.body == nil || it.pad != nil {
		return
	}
	h := accordionPadY - it.space
	if h <= 0 {
		return
	}
	// 追加到 body 末尾
	sp := widget.NewElement(0, 0, 0, h, it.body.Handle)
	if sp == nil {
		return
	}
	// EnableBkTransparent 只清背景, 原生 1px 边框还在 —— 必须一起关掉,
	// 否则展开后内容区会出现一个整宽的细矩形(看起来像多了一条分隔线)。
	sp.EnableBkTransparent(true)
	sp.EnableDrawBorder(false)
	xui.ItemFill(sp)
	sp.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	it.pad = sp
}

// setOpen 同步开合状态: 触发钮勾选态、内容高度过渡与 single 模式的互斥。
func (it *AccordionItem) setOpen(v bool) {
	if it == nil || it.open == v {
		return
	}
	it.applyOpen(v)
	// single: 展开一项就收起其它项。互斥收起不各自通知, 由这里统一通知一次
	// (同 radix 的 onValueChange: 一次交互只报一次最终集合)。
	if v && it.acc != nil && it.acc.mode == AccordionSingle {
		for _, other := range it.acc.items {
			if other != it && other.open {
				other.applyOpen(false)
			}
		}
	}
	if it.acc != nil {
		it.acc.notify()
	}
}

// applyOpen 只落状态与视觉(勾选态 + 内容高度过渡), 不通知根容器。
func (it *AccordionItem) applyOpen(v bool) {
	it.open = v
	if it.trigger != nil {
		it.trigger.SetCheck(v)
		it.trigger.Redraw(false)
	}
	// 占位必须在量测之前挂好: 展开量"自然高度"、收起量"当前自然高度"都用它。
	it.ensureBottomPad()
	if it.collapsible != nil {
		it.collapsible.SetOpen(v)
	}
}

// onPaintTrigger 绘制触发区: 透明底(擦宿主卡底)、文字 hover 下划线、
// 右侧箭头随开合切换、focus-visible 环、禁用半透明。坐标为逻辑像素。
func (it *AccordionItem) onPaintTrigger(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()
	st := xui.StateOf(hEle)

	// 透明元素重绘不擦底, 先铺宿主底色(同 Toggle)
	p.FillRound(&p.Rc, th.Card, 0)
	corners := xui.UniformCorners(th.RadiusLg()) // rounded-lg

	// focus-visible: ring-3 ring-ring/50 + border-ring(禁用不显示)。
	// 画法与 Button 一致: 整块铺环色 → 内芯**重新铺回宿主底色** → 内缘描 1px。
	// 少了"内芯铺底"这一步, 环色会把整条触发区填成灰条(看起来像选中态, 实测)。
	focusRing := st.Focused && !st.Disabled && xui.FocusedByKeyboard()
	if focusRing {
		// ring-3 ring-ring/50 的内嵌等价物: 3px 环带 + 内芯重铺宿主底色(同 Button)。
		// 不再额外描内缘 —— 多一道内描边会让文字离环更近。
		p.FocusFrameCorners(th.Ring, th.Card, corners)
	}

	fore := th.Foreground
	if st.Disabled {
		fore = fore.Mix(th.Card, 0.5) // disabled:opacity-50: 文字按宿主底色预混
	}
	f := xui.FontHandle(xui.FontSpec{Family: th.FontDefault.Family, Size: 14})
	it.drawText(p, f, fore)
	// hover:underline(上游没有过渡, 进入即出现): 字体侧的下划线(FontStyle_Underline)
	// 在本框架的 D2D 文本路径下**不会渲染**(实测, 见 DESIGN 第八节 110), 只能在文字
	// 墨迹下方自己补一条 1px 线 —— 宽度按实测文字宽度, 与文字同色。
	if (st.Hover || st.Pressed) && !st.Disabled {
		it.drawUnderline(p, f, fore)
	}

	// 右侧箭头: 收起 ChevronDown / 展开 ChevronUp(上游是两枚图标互换)
	icon := icons.IconChevronDown
	if it.open {
		icon = icons.IconChevronUp
	}
	// 箭头绘制块贴右缘(不再额外内距): chevron 的墨迹在 24 网格里横向只占 6..18,
	// 加上描边补偿, 绘制块贴边时**墨迹**离环带内缘天然还有 ~2px, 与左侧文字的观感
	// 一致(实测探针: 左右间隙都 ≈2 设备像素)。再往里让右侧就会空出一截。
	rc := xc.RECT{
		Right:  p.Rc.Right,
		Left:   p.Rc.Right - accordionIconSz,
		Top:    (p.Rc.Top + p.Rc.Bottom - accordionIconSz) / 2,
		Bottom: (p.Rc.Top + p.Rc.Bottom + accordionIconSz) / 2,
	}
	p.Icon(icon, &rc, th.MutedForeground, 2)
	return 0
}

// drawText 左对齐画触发区文字, 左内距 accordionPadX(给聚焦环带让位)。
func (it *AccordionItem) drawText(p *xui.Painter, font int, fore xui.Color) {
	text := it.trigger.GetText()
	if text == "" || font == 0 {
		return
	}
	p.Draw.SetFont(font)
	dy := textVDy(text)
	rc := xc.RECTF{
		Left:   float32(p.Rc.Left + accordionPadX),
		Top:    float32(p.Rc.Top) - dy,
		Right:  float32(p.Rc.Right - accordionPadX),
		Bottom: float32(p.Rc.Bottom) - dy,
	}
	p.TextF(text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
}

// drawUnderline 在文字墨迹下方补一条 1px 线(hover:underline 的自绘等价物)。
//
// 位置: Vcenter 下文字行盒以触发区中线为轴居中, 行盒底边比字形基线低一个 descent
// (中文约 4 逻辑像素); 上游的 hover:underline 没有 underline-offset, 所以落点是
// "行盒底边再上 2px" —— 实测(150% DPI)这样线在墨迹下方 3 物理像素, 不压字脚。
func (it *AccordionItem) drawUnderline(p *xui.Painter, font int, fore xui.Color) {
	text := it.trigger.GetText()
	if text == "" || font == 0 {
		return
	}
	w := xui.TextWidth(text, font)
	if w <= 0 {
		return
	}
	y := (p.Rc.Top+p.Rc.Bottom)/2 + xui.TextHeight(text, font)/2 - 2
	x := p.Rc.Left + accordionPadX
	p.Line(x, y, x+w, y, fore, 1)
}
