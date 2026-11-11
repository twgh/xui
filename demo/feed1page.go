package main

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

// ===== 页 16: 反馈1(对话框 / 提示气泡 / 警示对话框 / 对话框变体) =====

func buildFeed1Page(body *widget.LayoutEle, hWindow int) {
	r := row(body.Handle, 0, gap)

	// ---- 列 1: 对话框 ----
	_, v := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "对话框", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "窗口内覆盖元素: 全窗遮罩(backdrop-blur) + 居中面板 + Tab 焦点陷阱", v.Handle))
	rd := xui.HStack(v.Handle, 36, 8)
	editBtn := ui.NewButton(0, 0, 0, 0, "编辑资料", rd.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(editBtn, 104)
	delBtn := ui.NewButton(0, 0, 0, 0, "删除项目", rd.Handle, ui.WithVariant(ui.ButtonDestructive))
	xui.ItemWidth(delBtn, 104)
	scrollBtn := ui.NewButton(0, 0, 0, 0, "滚动内容", rd.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(scrollBtn, 104)
	buildDialogs(hWindow, editBtn, delBtn, scrollBtn)
	xui.ItemFill(desc(v.Handle, "ESC / 遮罩 / X 关闭, Tab 焦点循环;「滚动内容」头部/页脚固定。"))

	// ---- 列 2: 提示气泡 ----
	_, v2 := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "提示气泡", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "独立透明弹窗承载, 不抢焦点; 四个方向 + 对齐/间距 + 快捷键内容, 延迟可配", v2.Handle))
	// 卡片内容区放不下 4 个 88 宽的按钮, 分两行(每行 2 个)
	mkTipBtn := func(text string, rt *widget.LayoutEle) *ui.Button {
		b := ui.NewButton(0, 0, 0, 0, text, rt.Handle, ui.WithVariant(ui.ButtonOutline))
		xui.ItemWidth(b, 88)
		return b
	}
	rt := xui.HStack(v2.Handle, 36, 8)
	xui.ItemFill(rt)
	hoverBtn := mkTipBtn("悬浮我", rt)
	tip.Bind(hoverBtn.Handle, "上方(默认; 上方放不下翻到下方)")
	tipBottom := mkTipBtn("下方", rt)
	tip.BindTip(tipBottom.Handle, "提示在下方", ui.WithTipSide(ui.TipBottom))

	rt2 := xui.HStack(v2.Handle, 36, 8)
	xui.ItemFill(rt2)
	tipRight := mkTipBtn("右侧", rt2)
	tip.BindTip(tipRight.Handle, "右对齐 + 6px 间距", ui.WithTipSide(ui.TipRight),
		ui.WithTipAlign(ui.TipAlignCenter), ui.WithTipOffset(6))
	tipKbd := mkTipBtn("快捷键", rt2)
	tip.BindTip(tipKbd.Handle, "打开命令面板", ui.WithTipShortcut("Ctrl+K"))

	xui.ItemFill(desc(v2.Handle, "键盘焦点进入也显示(鼠标点击产生的焦点不算); WithTipDelay 设延迟、移出后 300ms 内移到下一个元素跳过延迟。绑定的元素自身有弹层时(如 NativeComboBox), 弹层打开要主动 HideTooltip。"))

	// ---- 第二行: 警示对话框 + 对话框变体 ----
	rDlg := row(body.Handle, 0, gap)

	_, vd := autoCard(xui.VStack(rDlg.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "警示对话框", vd.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"无 X、点遮罩不关、ESC 关闭; sm 居中 + 按钮等宽双列", vd.Handle))
	rad := xui.HStack(vd.Handle, 36, 8)
	dfltBtn := ui.NewButton(0, 0, 0, 0, "默认", rad.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(dfltBtn, 78)
	smBtn := ui.NewButton(0, 0, 0, 0, "小尺寸", rad.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(smBtn, 78)
	mediaBtn := ui.NewButton(0, 0, 0, 0, "带图标", rad.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(mediaBtn, 78)
	destrBtn := ui.NewButton(0, 0, 0, 0, "破坏性", rad.Handle, ui.WithVariant(ui.ButtonDestructive))
	xui.ItemWidth(destrBtn, 78)
	buildAlertDialogs(hWindow, dfltBtn, smBtn, mediaBtn, destrBtn)
	xui.ItemFill(desc(vd.Handle, "打开时焦点落在 Cancel; Tab 只在当前对话框内循环; Action / Cancel 点击后关闭。"))

	_, vdv := autoCard(xui.VStack(rDlg.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "对话框变体", vdv.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"无关闭钮; 嵌套时只有栈顶响应 ESC, 关闭上层后焦点回到下层", vdv.Handle))
	rv := xui.HStack(vdv.Handle, 36, 8)
	noCloseBtn := ui.NewButton(0, 0, 0, 0, "无关闭钮", rv.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(noCloseBtn, 104)
	nestedBtn := ui.NewButton(0, 0, 0, 0, "嵌套警示", rv.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(nestedBtn, 104)
	buildDialogVariants(hWindow, noCloseBtn, nestedBtn)
	xui.ItemFill(desc(vdv.Handle, "嵌套示例: 点「打开警示」在对话框上再叠一层; 连按 ESC 会先关上层。"))
}
