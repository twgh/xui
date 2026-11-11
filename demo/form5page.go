package main

import (
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

// ===== 页 9: 表单5(单选组 / 复选框三态与校验) =====

func buildForm5Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)
	colL := xui.VStack(r.Handle, 1, gap)
	colR := xui.VStack(r.Handle, 1, gap)

	// ---- 左列: 单选组 ----
	_, vGrp := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "单选组(RadioGroup)", vGrp.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"组值 + 跨层互斥; ↑↓/←→/Home/End 移动并选中(循环), 空格选中当前项", vGrp.Handle))

	xui.ItemFill(hint(vGrp.Handle, "默认(含一条 Field 说明项, Attach 跨层入组)"))
	grp := ui.NewRadioGroup(vGrp.Handle, ui.WithRadioGroupValue("mentions"))
	xui.ItemFill(grp)
	grp.AddItem("all", "所有消息")
	grp.AddItem("mentions", "仅提及")
	grp.AddItem("off", "关闭")
	grpState := ui.NewLabel(0, 0, 0, 0, "选中: mentions", vGrp.Handle).SetMuted()
	xui.ItemFill(grpState)
	grp.OnChange(func(v string) { grpState.SetText("选中: " + v).Redraw() })

	// 条目放在 Field 子容器里时不在组容器直下, xcgui 的"同父容器互斥"管不到;
	// Attach 把它接入组语义(互斥/方向键/变更回调仍然生效)。
	fCustom := ui.NewField(0, 0, 0, 0, grp.Handle)
	xui.ItemFill(fCustom)
	rCustom := ui.NewRadio(0, 0, 0, 0, "自定义(带说明)", fCustom.Handle)
	xui.ItemFill(rCustom)
	fCustom.Description("条目放在 Field 子容器里, 仍参与组互斥与键盘导航。")
	grp.Attach(rCustom, "custom")

	xui.ItemFill(hint(vGrp.Handle, "禁用整组"))
	grpDis := ui.NewRadioGroup(vGrp.Handle, ui.WithRadioGroupDisabled(), ui.WithRadioGroupValue("a"))
	xui.ItemFill(grpDis)
	grpDis.AddItem("a", "选项 A")
	grpDis.AddItem("b", "选项 B")

	xui.ItemFill(hint(vGrp.Handle, "校验失败(aria-invalid 常驻红环, 可切换)"))
	grpInv := ui.NewRadioGroup(vGrp.Handle, ui.WithRadioGroupInvalid())
	xui.ItemFill(grpInv)
	grpInv.AddItem("x", "选项 X")
	grpInv.AddItem("y", "选项 Y")
	grpInvBtn := ui.NewButton(0, 0, 0, 0, "切换校验状态", vGrp.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm))
	xui.ItemWidth(grpInvBtn, 120)
	grpInvBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		grpInv.SetInvalid(!grpInv.Invalid())
		return 0
	})

	// ---- 左列: 选择卡片(整卡可点的单选) ----
	_, vChoice := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "选择卡片(RadioGroup)", vChoice.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"点卡片任意位置与点指示器等价; 选中 = border-primary + bg-accent", vChoice.Handle))
	buildRadioChoiceCards(vChoice.Handle)

	// ---- 右列: Checkbox 三态 / 校验 ----
	_, vCB := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "Checkbox 三态与校验", vCB.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"不确定态(indeterminate)画减号, 点击/空格落到选中; aria-invalid 红环常驻", vCB.Handle))

	xui.ItemFill(hint(vCB.Handle, "不确定态(点击后变选中)"))
	indet := ui.NewCheckbox(0, 0, 0, 0, "部分选中", vCB.Handle, ui.WithCheckboxIndeterminate())
	xui.ItemFill(indet)
	indetState := ui.NewLabel(0, 0, 0, 0, "状态: 不确定", vCB.Handle).SetMuted()
	xui.ItemFill(indetState)
	indet.OnChange(func(checked bool) {
		switch {
		case indet.Indeterminate():
			indetState.SetText("状态: 不确定")
		case checked:
			indetState.SetText("状态: 已选中")
		default:
			indetState.SetText("状态: 未选中")
		}
		indetState.Redraw()
	})

	xui.ItemFill(hint(vCB.Handle, "校验失败(可切换)"))
	invCB := ui.NewCheckbox(0, 0, 0, 0, "必选条款", vCB.Handle, ui.WithCheckboxInvalid())
	xui.ItemFill(invCB)
	cbInvBtn := ui.NewButton(0, 0, 0, 0, "切换校验状态", vCB.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm))
	xui.ItemWidth(cbInvBtn, 120)
	cbInvBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		invCB.SetInvalid(!invCB.Invalid())
		return 0
	})

	xui.ItemFill(hint(vCB.Handle, "带说明的字段(整卡可点)"))
	buildCheckboxChoiceCard(vCB.Handle)

	// ---- 右列: 颜色选择器 ----
	_, vCP := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "颜色选择器(ColorPicker)", vCP.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"SV 方块调饱和度/亮度(可拖出方块继续调), 色相条, 预设色, Hex 输入满 6 位才应用", vCP.Handle))

	cp := ui.NewColorPicker(0, 0, 0, 0, vCP.Handle, ui.WithColorPickerValue("#3b82f6"))
	xui.ItemFill(cp)
	cpState := ui.NewLabel(0, 0, 0, 0, "当前颜色: #3b82f6", vCP.Handle).SetMuted()
	xui.ItemFill(cpState)
	cp.OnChange(func(hex string) {
		cpState.SetText("当前颜色: " + hex).Redraw()
	})

	xui.ItemFill(hint(vCP.Handle, "禁用态与自定义预设"))
	cpDis := ui.NewColorPicker(0, 0, 0, 0, vCP.Handle, ui.WithColorPickerValue("#22c55e"))
	xui.ItemFill(cpDis)
	cpDis.Enable(false)
}

// buildRadioChoiceCards 是 shadcn radio-group「选择卡片」示例: 三张套餐卡片, 整卡可点。
// 单选指示器放在卡片右端, 卡片本体(标题/说明)点击走 radio.Select(), 与点指示器同一条路径。
func buildRadioChoiceCards(parent int) {
	plans := []struct{ value, title, desc string }{
		{"plus", "Plus", "面向个人与小团队。"},
		{"pro", "Pro", "面向成长中的团队。"},
		{"enterprise", "Enterprise", "面向大型团队与企业。"},
	}
	grp := ui.NewRadioGroup(parent, ui.WithRadioGroupValue("pro"), ui.WithRadioGroupSpacing(12))
	xui.ItemFill(grp)
	cards := make([]*choiceCard, 0, len(plans))
	for _, plan := range plans {
		card := newChoiceCard(grp.Handle, 6)
		if card == nil {
			continue
		}
		cards = append(cards, card)
		title := ui.NewLabel(0, 0, 0, 20, plan.title, card.Content().Handle).SetFontSize(14).SetMedium()
		title.EnableMouseThrough(true) // 文本不接鼠标, 点击落到卡片
		xui.ItemFill(title)
		desc := ui.NewLabel(0, 0, 0, 20, plan.desc, card.Content().Handle).SetMuted().SetFontSize(14)
		desc.EnableMouseThrough(true)
		xui.ItemFill(desc)

		radio := ui.NewRadio(0, 0, 0, 0, "", card.Handle())
		xui.ItemSize(radio, 24, 24)
		// 点击统一交给卡片: 指示器本身不参与命中, 避免父子各处理一次
		radio.EnableMouseThrough(true)
		// 卡片选中态底色是 Accent, 指示器的宿主底色要跟着走(否则元素矩形留下 Card 色方块)
		radio.SetBackdrop(func(t *xui.Theme) xui.Color {
			if radio.Selected() {
				return t.Accent
			}
			return t.Card
		})
		radio.OnChange(func(on bool) { card.setActive(on) })
		xui.ItemMarginTop(radio, 12)
		card.onClick(func() {
			radio.Select()
			xui.FocusElement(radio.Handle)
		})
		grp.Attach(radio, plan.value)
	}
	// 选中变化时同步所有卡片: 条目分属不同父级(各自卡片), 原生互斥管不到别的卡片,
	// 被取消选中的卡片不会收到 OnChange(false), 只能在这里统一刷。
	grp.OnChange(func(v string) {
		for i, plan := range plans {
			if i < len(cards) {
				cards[i].setActive(plan.value == v)
			}
		}
	})
	for i, plan := range plans {
		if i < len(cards) && plan.value == grp.Value() {
			cards[i].setActive(true)
		}
	}
}

// buildCheckboxChoiceCard 是 shadcn checkbox「描述」示例: 指示器在左, 标题 + 说明在右,
// 点卡片任意位置切换勾选(整卡一个命中区, 与 shadcn 的 Field label 包裹一致)。
func buildCheckboxChoiceCard(parent int) {
	card := newChoiceCard(parent, 6)
	if card == nil {
		return
	}
	cb := ui.NewCheckbox(0, 0, 24, 24, "", card.Handle())
	xui.ItemSize(cb, 24, 24)
	cb.SetChecked(true)
	cb.EnableMouseThrough(true) // 同单选卡片: 命中区只留卡片一层
	cb.SetBackdrop(func(t *xui.Theme) xui.Color {
		if cb.Checked() {
			return t.Accent
		}
		return t.Card
	})
	cb.OnChange(func(checked bool) { card.setActive(checked) })
	card.setActive(cb.Checked())
	xui.ItemMarginTop(cb, 12)

	title := ui.NewLabel(0, 0, 0, 20, "启用通知", card.Content().Handle).SetFontSize(14).SetMedium()
	title.EnableMouseThrough(true) // 文本不接鼠标, 点击落到卡片
	xui.ItemFill(title)
	desc := ui.NewLabel(0, 0, 0, 20, "你可以随时开启或关闭通知。",
		card.Content().Handle).SetMuted().SetFontSize(14)
	desc.EnableMouseThrough(true)
	xui.ItemFill(desc)
	card.onClick(func() {
		cb.Toggle()
		xui.FocusElement(cb.Handle)
	})
}

// choiceCard 是一张"整卡可选"的卡片(shadcn 选择卡片): rounded-lg border p-3,
// 选中态 border-primary + bg-accent; 内部横向排 —— 内容列吃剩余宽度, 指示器放另一侧。
//
// 卡片自绘是因为选中态要同时改底色和边框; 点击区域是整张卡片(子项一律鼠标穿透),
// 指示器本身不接鼠标, 由卡片统一调 Radio.Select / Checkbox.Toggle。
type choiceCard struct {
	card    *widget.LayoutEle
	content *widget.LayoutEle
	gap     int32
	active  bool
}

// Handle 返回卡片元素句柄(子项都挂在卡片下)。
func (c *choiceCard) Handle() int {
	if c == nil || c.card == nil {
		return 0
	}
	return c.card.Handle
}

// newChoiceCard 创建卡片骨架。左右内边距用元素 padding(Fill 子项才会按内容宽解析,
// 见 DESIGN 129), 上下内边距用子项外边距(计入卡片的 Auto 高)。
func newChoiceCard(parent int, gap int32) *choiceCard {
	ele := widget.NewLayoutEle(0, 0, 0, 0, parent)
	if ele == nil {
		return nil
	}
	c := &choiceCard{card: ele, gap: gap}
	// 布局元素默认是"纯排版壳"(鼠标穿透), 要自己当命中区必须显式关掉穿透
	ele.EnableMouseThrough(false)
	ele.EnableHorizon(true)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(12)
	ele.SetAlignV(xcc.Layout_Align_Top)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ele.SetPadding(12, 0, 8, 0)
	ele.AddEvent_Paint(c.onPaint)
	xui.AttachWindow(ele.Handle)
	return c
}

// Content 返回卡片内的内容列(Weight 1, Auto 高)。首次调用时创建, 所以调用顺序决定
// 指示器在左还是在右: 先建指示器 → 内容列在其右侧; 先取 Content() → 指示器在右端。
func (c *choiceCard) Content() *widget.LayoutEle {
	if c == nil {
		return nil
	}
	if c.content == nil {
		// col 会在列宽变化时把 Fill 子项锁 Fixed(横排 Weight 列的必要处理, 见 DESIGN 101/130)
		c.content = xui.VStack(c.card.Handle, 1, c.gap)
		xui.ItemMarginTop(c.content, 12)
		xui.ItemMarginBottom(c.content, 12)
		c.content.EnableMouseThrough(true)
	}
	return c.content
}

// onClick 让整张卡片成为点击区域。子项都设了鼠标穿透, 事件在这里统一处理。
func (c *choiceCard) onClick(fn func()) {
	if c == nil {
		return
	}
	c.card.SetCursor(choiceCardCursor())
	c.card.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress() // 鼠标点出来的焦点不画键盘焦点环
		return 0
	})
	c.card.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if fn != nil {
			fn()
		}
		return 0
	})
}

func (c *choiceCard) setActive(v bool) {
	if c == nil || c.active == v {
		return
	}
	c.active = v
	c.card.Redraw(false)
}

func (c *choiceCard) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	bg, border := t.Card, t.Border
	if c.active {
		bg, border = t.Accent, t.Primary
	}
	// 卡片是透明的自绘元素, 整块铺底既擦上一帧也给出选中底色; 描边内缩 1px 免得被裁
	radius := t.RadiusLg()
	p.FillRound(&p.Rc, bg, radius)
	rc := p.Rc
	rc.Left++
	rc.Top++
	rc.Right--
	rc.Bottom--
	p.StrokeRound(&rc, border, radius-1, 1)
	return 0
}

var choiceCardHand uintptr

// choiceCardCursor 懒加载系统手型光标(shadcn 卡片是 cursor-pointer)。
func choiceCardCursor() uintptr {
	if choiceCardHand == 0 {
		choiceCardHand = wapi.LoadImageW(0, wapi.IDC_HAND, wapi.IMAGE_CURSOR, 0, 0,
			wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)
	}
	return choiceCardHand
}
