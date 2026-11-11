package main

import (
	"fmt"

	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 12: 导航3(步骤条 / 可折叠面板) =====

func buildNav3Page(body *widget.LayoutEle) {
	// ---- 步骤条 ----
	_, vst := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "步骤条", vst.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"横向 / 竖向两形态; 已完成 / 进行中 / 未达 / 出错四态; 默认点已完成步回退; Tab 聚焦后 ←/→ 移动焦点、Enter 激活", vst.Handle))

	stepItems := []ui.StepItem{
		{Title: "账户信息", Description: "填写邮箱与姓名"},
		{Title: "验证邮箱", Description: "输入收到的验证码"},
		{Title: "设置密码", Description: "至少 8 位字符"},
		{Title: "完成", Description: "开始使用"},
	}
	stepInfo := ui.NewLabel(0, 0, 0, 20, "", vst.Handle).SetMuted()
	xui.ItemFill(stepInfo)
	hSteps := ui.NewSteps(0, 0, 0, 0, vst.Handle, stepItems, ui.WithStepsCurrent(1))
	xui.ItemFill(hSteps)

	stepRow := xui.HStack(vst.Handle, 32, 8)
	prevStep := ui.NewButton(0, 0, 0, 0, "上一步", stepRow.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(prevStep, 88)
	nextStep := ui.NewButton(0, 0, 0, 0, "下一步", stepRow.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(nextStep, 88)

	xui.ItemFill(hint(vst.Handle, "竖向(WithStepsOrientation(StepsVertical))"))
	vSteps := ui.NewSteps(0, 0, 0, 0, vst.Handle, stepItems,
		ui.WithStepsOrientation(ui.StepsVertical), ui.WithStepsCurrent(1))
	xui.ItemFill(vSteps)

	xui.ItemFill(hint(vst.Handle, "自定义图标 + 任意步可点(StepItem.Icon / WithStepsClickable; 第 4 步 Disabled 只挡点击)"))
	iconSteps := ui.NewSteps(0, 0, 0, 0, vst.Handle, []ui.StepItem{
		{Title: "账户信息", Description: "填写邮箱与姓名", Icon: icons.IconUser},
		{Title: "验证邮箱", Description: "输入收到的验证码", Icon: icons.IconMail},
		{Title: "设置密码", Description: "至少 8 位字符", Icon: icons.IconLock},
		{Title: "完成", Description: "开始使用", Icon: icons.IconCircleCheckBig, Disabled: true},
	}, ui.WithStepsCurrent(1), ui.WithStepsClickable())
	xui.ItemFill(iconSteps)

	xui.ItemFill(hint(vst.Handle, "出错状态(StepItem.Status = StepStatusError; 指向它的连接线同步变红)"))
	errSteps := ui.NewSteps(0, 0, 0, 0, vst.Handle, []ui.StepItem{
		{Title: "账户信息"},
		{Title: "验证邮箱"},
		{Title: "设置密码", Status: ui.StepStatusError},
		{Title: "完成"},
	}, ui.WithStepsCurrent(2))
	xui.ItemFill(errSteps)

	xui.ItemFill(hint(vst.Handle, "小号 + 进度 + 起始编号(WithStepsSize(StepsSmall) / StepItem.Percent / WithStepsInitial(2))"))
	smallInfo := ui.NewLabel(0, 0, 0, 20, "", vst.Handle).SetMuted()
	xui.ItemFill(smallInfo)
	smallSteps := ui.NewSteps(0, 0, 0, 0, vst.Handle, []ui.StepItem{
		{Title: "解析"},
		{Title: "上传", Percent: 35},
		{Title: "校验"},
		{Title: "完成"},
	}, ui.WithStepsCurrent(1), ui.WithStepsSize(ui.StepsSmall), ui.WithStepsInitial(2))
	xui.ItemFill(smallSteps)
	progressRow := xui.HStack(vst.Handle, 32, 8)
	progressBtn := ui.NewButton(0, 0, 0, 0, "推进进度", progressRow.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(progressBtn, 88)
	progressPct := int32(35)
	showProgress := func() {
		smallInfo.SetText(fmt.Sprintf("当前步进度 %d%% · 编号从第 3 步起", progressPct)).Redraw(false)
	}
	progressBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		progressPct += 15
		if progressPct > 100 {
			progressPct = 100
		}
		smallSteps.SetItemPercent(1, progressPct)
		showProgress()
		return 0
	})
	showProgress()

	syncSteps := func(i int) {
		hSteps.SetCurrent(i)
		vSteps.SetCurrent(i)
		iconSteps.SetCurrent(i)
		stepInfo.SetText(fmt.Sprintf("第 %d 步 / 共 %d 步 · %s", i+1, len(stepItems), stepItems[i].Title)).Redraw(false)
		prevStep.Enable(i > 0)
		nextStep.Enable(i < len(stepItems)-1)
		prevStep.Redraw(false)
		nextStep.Redraw(false)
	}
	prevStep.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		syncSteps(hSteps.Current() - 1)
		return 0
	})
	nextStep.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		syncSteps(hSteps.Current() + 1)
		return 0
	})
	hSteps.OnChange(syncSteps)
	vSteps.OnChange(syncSteps)
	iconSteps.OnChange(syncSteps)
	syncSteps(1)

	// ---- 可折叠面板 ----
	_, v4 := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "可折叠面板", v4.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"radix Collapsible: 触发区 + 内容区两种组合; 内容高度 200ms 过渡, 收起后不占布局", v4.Handle))
	ex := row(v4.Handle, 0, 24)
	left := xui.VStack(ex.Handle, 1, 8)
	buildCollapsibleOrder(left.Handle)
	buildCollapsibleTextTrigger(left.Handle)
	buildCollapsibleTree(xui.VStack(ex.Handle, 1, 8).Handle)
}

func buildCollapsibleTextTrigger(parent int) {
	xui.ItemFill(hint(parent, "整宽触发区 · 文字靠左 / 箭头贴右"))
	panel := ui.NewCollapsible(0, 0, 0, 0, parent)
	panel.SetSpace(8)
	xui.ItemFill(panel)
	btn := ui.NewButton(0, 0, 0, 0, "高级设置", panel.Handle,
		ui.WithVariant(ui.ButtonGhost),
		ui.WithButtonAlign(ui.ButtonAlignBetween),
		ui.WithButtonTrailingIcon(icons.IconChevronDown))
	xui.ItemFill(btn)
	panel.BindTrigger(btn)
	panel.OnOpenChange(func(open bool) {
		btn.SetTrailingIcon(common.Choose(open, icons.IconChevronUp, icons.IconChevronDown))
	})
	body := panel.ContentStack(8)
	desc := ui.NewLabel(0, 0, 0, 20, "通知与隐私相关的开关都放在这里; 展开/收起走同一个高度过渡。",
		body.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(desc)
	xui.ItemMargin(desc, 10, 0, 10, 0)
}

// buildCollapsibleOrder 是 shadcn collapsible-demo 的订单详情: 图标触发区 + 常显状态行 + 可折叠明细。
func buildCollapsibleOrder(parent int) {
	xui.ItemFill(hint(parent, "订单详情 · 图标触发区"))

	panel := ui.NewCollapsible(0, 0, 0, 0, parent)
	panel.SetSpace(8)
	xui.ItemFill(panel)

	head := widget.NewLayoutEle(0, 0, 0, 33, panel.Handle)
	head.EnableHorizon(true)
	head.EnableLayout(true)
	head.EnableBkTransparent(true)
	head.EnableAutoWrap(false)
	head.SetSpace(8)
	head.SetAlignV(xcc.Layout_Align_Center)
	head.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	head.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 33)
	xui.ItemMargin(head, 10, 0, 10, 0)
	title := ui.NewLabel(0, 0, 0, 20, "订单 #4189", head.Handle).SetFontSize(14).SetMedium()
	xui.ItemWeight(title, 1)
	toggle := ui.NewButton(0, 0, 0, 0, "", head.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeIcon),
		ui.WithIcon(icons.IconChevronsUpDown))
	xui.ItemWidth(toggle, 32)
	panel.BindTrigger(toggle)

	status := borderedBox(panel.Handle, 36, true)
	left := ui.NewLabel(0, 0, 0, 20, "状态", status.Handle).SetFontSize(14).SetMuted()
	xui.ItemWeight(left, 1)
	xui.ItemMarginLeft(left, 16)
	right := ui.NewLabel(0, 0, 0, 20, "已发货", status.Handle).SetFontSize(14).SetMedium()
	xui.ItemWidth(right, 72)
	xui.ItemMarginRight(right, 16)

	body := panel.ContentStack(8)
	addr := borderedBox(body.Handle, 56, false)
	addrTitle := ui.NewLabel(0, 0, 0, 20, "收货地址", addr.Handle).SetFontSize(14).SetMedium()
	xui.ItemFill(addrTitle)
	xui.ItemMargin(addrTitle, 16, 8, 16, 0)
	addrText := ui.NewLabel(0, 0, 0, 20, "100 Market St, San Francisco", addr.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(addrText)
	xui.ItemMargin(addrText, 16, 0, 16, 8)
	xui.ItemMargin(addr, 10, 0, 10, 0)

	items := borderedBox(body.Handle, 56, false)
	itemsTitle := ui.NewLabel(0, 0, 0, 20, "商品", items.Handle).SetFontSize(14).SetMedium()
	xui.ItemFill(itemsTitle)
	xui.ItemMargin(itemsTitle, 16, 8, 16, 0)
	itemsText := ui.NewLabel(0, 0, 0, 20, "2x 头戴式耳机", items.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(itemsText)
	xui.ItemMargin(itemsText, 16, 0, 16, 8)
	xui.ItemMargin(items, 10, 0, 10, 10)
}

// buildCollapsibleTree 演示嵌套折叠(文件夹树)与整宽触发区(文字靠左、图标靠右)。
func buildCollapsibleTree(parent int) {
	xui.ItemFill(hint(parent, "文件树 · 嵌套折叠"))

	tree := ui.NewCollapsible(0, 0, 0, 0, parent)
	tree.SetSpace(2)
	xui.ItemFill(tree)
	compBtn := folderTrigger(tree.Handle, "components")
	tree.BindTrigger(compBtn)
	tree.OnOpenChange(func(open bool) {
		compBtn.SetTrailingIcon(common.Choose(open, icons.IconChevronDown, icons.IconChevronRight))
	})

	compBody := tree.ContentStack(1)
	xui.ItemMarginLeft(compBody, 20)

	uiRoot := ui.NewCollapsible(0, 0, 0, 0, compBody.Handle)
	uiRoot.SetSpace(2)
	xui.ItemFill(uiRoot)
	uiBtn := folderTrigger(uiRoot.Handle, "ui")
	uiRoot.BindTrigger(uiBtn)
	uiRoot.OnOpenChange(func(open bool) {
		uiBtn.SetTrailingIcon(common.Choose(open, icons.IconChevronDown, icons.IconChevronRight))
	})
	uiBody := uiRoot.ContentStack(1)
	xui.ItemMarginLeft(uiBody, 20)
	fileRow(uiBody.Handle, "button.tsx")
	fileRow(uiBody.Handle, "card.tsx")
	fileRow(uiBody.Handle, "dialog.tsx")

	fileRow(compBody.Handle, "login-form.tsx")
	fileRow(compBody.Handle, "register-form.tsx")
}

// folderTrigger 建文件树的文件夹触发区: 幽灵按钮、整宽、内容靠左, 尾随方向箭头。
func folderTrigger(parent int, name string) *ui.Button {
	b := ui.NewButton(0, 0, 0, 0, name, parent,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeSm),
		ui.WithButtonAlign(ui.ButtonAlignStart),
		ui.WithButtonLeadingIcon(icons.IconFolder),
		ui.WithButtonTrailingIcon(icons.IconChevronRight))
	xui.ItemFill(b)
	return b
}

// fileRow 建文件树里的文件行(幽灵按钮, 内容靠左 + 文件图标)。
func fileRow(parent int, name string) *ui.Button {
	b := ui.NewButton(0, 0, 0, 0, name, parent,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeSm),
		ui.WithButtonAlign(ui.ButtonAlignStart),
		ui.WithButtonLeadingIcon(icons.IconFile))
	xui.ItemFill(b)
	return b
}

// borderedBox 建一个 1px Border + rounded-md 的盒子(对应 shadcn 示例里的
// `rounded-md border px-4 py-2`)。horizontal=true 时是横排(内容垂直居中)。
