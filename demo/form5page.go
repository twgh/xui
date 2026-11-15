package main

import (
	"strings"

	"github.com/twgh/xcgui/widget"
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

	xui.ItemFill(hint(vGrp.Handle, "横向排列(WithRadioGroupHorizontal, 条目宽按文字量固定)"))
	grpH := ui.NewRadioGroup(vGrp.Handle, ui.WithRadioGroupHorizontal(), ui.WithRadioGroupValue("m"))
	xui.ItemFill(grpH)
	grpH.AddItem("s", "小")
	grpH.AddItem("m", "中")
	grpH.AddItem("l", "大")
	grpH.AddItem("xl", "特大")

	xui.ItemFill(hint(vGrp.Handle, "禁用整组"))
	grpDis := ui.NewRadioGroup(vGrp.Handle, ui.WithRadioGroupDisabled(), ui.WithRadioGroupValue("a"))
	xui.ItemFill(grpDis)
	grpDis.AddItem("a", "选项 A")
	grpDis.AddItem("b", "选项 B")

	xui.ItemFill(hint(vGrp.Handle, "整组只读 / 否决换选(未保存内容禁止切换)"))
	grpRO := ui.NewRadioGroup(vGrp.Handle, ui.WithRadioGroupValue("a"), ui.WithRadioGroupReadOnly())
	xui.ItemFill(grpRO)
	grpRO.AddItem("a", "只读 A(点 B 不会换)")
	grpRO.AddItem("b", "只读 B")
	grpVeto := ui.NewRadioGroup(vGrp.Handle, ui.WithRadioGroupValue("keep"))
	xui.ItemFill(grpVeto)
	grpVeto.AddItem("keep", "保持此项(换选被否决)")
	grpVeto.AddItem("other", "换不过去")
	grpVeto.OnValueChangeVeto(func(v string) bool { return v == "keep" })

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

	xui.ItemFill(hint(vCB.Handle, "复选组(父项全选 / 半选联动 + 组值)"))
	cbAll := ui.NewCheckbox(0, 0, 0, 0, "全选", vCB.Handle)
	xui.ItemFill(cbAll)
	cbGrp := ui.NewCheckboxGroup(vCB.Handle,
		ui.WithCheckboxGroupValues("disks", "servers"), ui.WithCheckboxGroupSpacing(8))
	xui.ItemFill(cbGrp)
	cbGrp.SetParent(cbAll)
	cbGrp.Add("disks", "硬盘")
	cbGrp.Add("external", "外置磁盘")
	cbGrp.Add("cds", "CD / DVD")
	cbGrp.Add("servers", "连接的服务器")
	cbGrpState := ui.NewLabel(0, 0, 0, 0, "选中: disks, servers", vCB.Handle).SetMuted()
	xui.ItemFill(cbGrpState)
	cbGrp.OnChange(func(values []string) {
		if len(values) == 0 {
			cbGrpState.SetText("选中: (空)")
		} else {
			cbGrpState.SetText("选中: " + strings.Join(values, ", "))
		}
		cbGrpState.Redraw()
	})

	xui.ItemFill(hint(vCB.Handle, "纯指示器热区(默认 40×32) / 只读 / 可否决"))
	hotRow := xui.HStack(vCB.Handle, 32, 16)
	ui.NewCheckbox(0, 0, 0, 0, "", hotRow.Handle, ui.WithCheckboxChecked(true))
	ui.NewCheckbox(0, 0, 0, 0, "只读(点击不切换)", hotRow.Handle,
		ui.WithCheckboxReadOnly(), ui.WithCheckboxChecked(true))
	vetoCB := ui.NewCheckbox(0, 0, 0, 0, "否决取消(试试点掉)", vCB.Handle,
		ui.WithCheckboxChecked(true))
	xui.ItemFill(vetoCB)
	vetoCB.OnCheckedChange(func(checked bool) bool { return checked }) // 取消操作一律否决

	xui.ItemFill(hint(vCB.Handle, "标签关联(<label for>): 点文字聚焦并切换"))
	labRow := xui.HStack(vCB.Handle, 24, 8)
	cbLab := ui.NewCheckbox(0, 0, 0, 0, "", labRow.Handle)
	labText := ui.NewLabel(0, 0, 0, 24, "点这段文字也能切换复选框", labRow.Handle)
	xui.ItemWidth(labText, 200)
	labText.BindToggle(cbLab.Handle, func() { cbLab.Toggle() })

	xui.ItemFill(hint(vCB.Handle, "组级禁用(父项与子项一起变淡)"))
	cbGrpDis := ui.NewCheckboxGroup(vCB.Handle,
		ui.WithCheckboxGroupDisabled(), ui.WithCheckboxGroupValues("a"))
	xui.ItemFill(cbGrpDis)
	cbGrpDis.Add("a", "选项 A")
	cbGrpDis.Add("b", "选项 B")

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
	cards := make([]*ui.FieldChoiceCard, 0, len(plans))
	for _, plan := range plans {
		card := ui.NewFieldChoiceCard(0, 0, 0, 0, grp.Handle)
		if card == nil {
			continue
		}
		cards = append(cards, card)
		xui.ItemFill(card)
		// 先建内容列 → 指示器排在右端
		card.Label(plan.title)
		card.Description(plan.desc)
		radio := ui.NewRadio(0, 0, 0, 0, "", card.Handle)
		card.Attach(radio) // 穿透 / 焦点环代理 / 宿主底色 / 点卡片换选都由卡片处理
		grp.Attach(radio, plan.value)
	}
	// 选中变化时同步所有卡片: 条目分属不同父级(各自卡片), 原生互斥管不到别的卡片,
	// 被取消选中的卡片不会收到 OnChange(false), 只能在这里统一刷。
	grp.OnChange(func(v string) {
		for i, plan := range plans {
			if i < len(cards) {
				cards[i].SetActive(plan.value == v)
			}
		}
	})
	for i, plan := range plans {
		if i < len(cards) && plan.value == grp.Value() {
			cards[i].SetActive(true)
		}
	}
}

// buildCheckboxChoiceCard 是 shadcn checkbox「描述」示例: 指示器在左, 标题 + 说明在右,
// 点卡片任意位置切换勾选(整卡一个命中区, 与 shadcn 的 Field label 包裹一致)。
func buildCheckboxChoiceCard(parent int) {
	card := ui.NewFieldChoiceCard(0, 0, 0, 0, parent)
	if card == nil {
		return
	}
	xui.ItemFill(card)
	// 指示器先建 → 内容列排在右端
	cb := ui.NewCheckbox(0, 0, 24, 24, "", card.Handle)
	cb.SetChecked(true)
	card.Attach(cb)
	card.Label("启用通知")
	card.Description("你可以随时开启或关闭通知。")
}
