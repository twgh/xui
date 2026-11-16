package main

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
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

	// 可移入的气泡(WithTipHoverable): 光标从按钮移进气泡时保持显示, 离开气泡才收。
	rt3 := xui.HStack(v2.Handle, 36, 8)
	xui.ItemFill(rt3)
	hoverableBtn := mkTipBtn("可移入", rt3)
	tip.BindTip(hoverableBtn.Handle, "把光标移进这段文字试试(不会消失)",
		ui.WithTipHoverable(), ui.WithTipArrowPadding(12))

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

	// ---- 列 3: 头部 / 页脚内容槽 ----
	_, vs := autoCard(xui.VStack(rDlg.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "内容槽", vs.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Header() / Footer() 返回布局容器: 徽章、复选框等任意控件与标题/描述/按钮共存", vs.Handle))
	rs := xui.HStack(vs.Handle, 36, 8)
	slotBtn := ui.NewButton(0, 0, 0, 0, "打开内容槽", rs.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(slotBtn, 104)
	buildSlotDialog(hWindow, slotBtn)
	xui.ItemFill(desc(vs.Handle,
		"页脚自定义内容在按钮左侧(Sheet 是纵向堆叠); 追加控件后用 ItemFill / ItemWidth 声明宽度。"))
}

// buildDialogs 三个对话框: 表单型(编辑资料)、确认型(删除项目) 与可滚动内容。
func buildDialogs(hWindow int, editTrigger, delTrigger, scrollTrigger *ui.Button) {
	// 表单对话框: 标题 + 描述 + 两个带标签的输入框 + 页脚
	edit := ui.NewDialog(hWindow)
	edit.SetTitle("编辑资料")
	edit.SetDescription("修改你的公开资料, 完成后点击保存。")
	edit.BindTrigger(editTrigger)

	// 内容容器是普通元素, 子元素按显式坐标摆放(对话框宽度固定, 高度由对话框自动量出)。
	// 对齐 shadcn App.tsx: 两个 space-y-2 组(标签 14 + 8 + 输入框 32), 组间 gap-4
	cw := edit.ContentWidth()
	ui.NewLabel(0, -2, cw, 20, "显示名称", edit.Content())
	name := ui.NewInput(0, 22, cw, 0, edit.Content(), ui.WithPlaceholder("显示名称"))
	name.SetValue("twgh")
	ui.NewLabel(0, 68, cw, 20, "个人主页", edit.Content())
	ui.NewInput(0, 92, cw, 0, edit.Content(), ui.WithPlaceholder("https://"))

	edit.AddFooter("取消", ui.ButtonOutline, func() { edit.Close() })
	edit.AddFooter("保存", ui.ButtonPrimary, func() { edit.Close() })

	// 确认对话框: 只有标题 + 描述 + 页脚
	del := ui.NewDialog(hWindow)
	del.SetTitle("确认删除?")
	del.SetDescription("删除后无法恢复, 确定要删除该项目吗?")
	del.BindTrigger(delTrigger)
	del.AddFooter("取消", ui.ButtonOutline, func() { del.Close() })
	del.AddFooter("删除", ui.ButtonDestructive, func() { del.Close() })

	// 可滚动内容: 对应 shadcn 文档的 Scrollable Content / With Sticky Footer ——
	// 头部与页脚固定, 中间内容超过 70vh(实际再夹到面板可用高度)后出滚动条。
	scroll := ui.NewDialog(hWindow, ui.WithDialogScroll(0))
	scroll.SetTitle("可滚动内容")
	scroll.SetDescription("头部与页脚固定, 中间内容区超过可用高度后滚动。")
	scroll.BindTrigger(scrollTrigger)
	body := scroll.ScrollArea().ContentStack(12)
	for i := 0; i < 10; i++ {
		l := ui.NewLabel(0, 0, 0, 100, fmt.Sprintf(
			"%d. 这是一段用于演示滚动的长文本。对话框的面板高度由头部、内容与页脚共同决定, "+
				"内容超过 max-h-[70vh] 时保持头部和页脚不动, 只让中间区域滚动。", i+1),
			body.Handle).SetMuted().SetWrap()
		xui.ItemFill(l)
	}
	scroll.AddFooter("关闭", ui.ButtonOutline, func() { scroll.Close() })
}

// buildSlotDialog 演示头部/页脚内容槽: Header() 里追加徽章, Footer() 里放复选框,
// 与 AddFooter 的按钮共用同一行(普通对话框自定义内容在左、按钮在右)。
func buildSlotDialog(hWindow int, trigger *ui.Button) {
	d := ui.NewDialog(hWindow)
	d.SetTitle("发布版本")
	d.SetDescription("头部与页脚都能放任意控件: 徽章、复选框、说明文字。")
	d.BindTrigger(trigger)

	badge := ui.NewBadge(0, 0, 0, 0, "Beta", d.Header().Handle,
		ui.WithBadgeVariant(ui.BadgeSecondary))
	xui.ItemWidth(badge, 46)

	ui.NewLabel(0, 0, d.ContentWidth(), 24,
		"发布后所有用户会收到更新提示。", d.Content()).SetFontSize(14).SetMuted()

	cb := ui.NewCheckbox(0, 0, 0, 0, "同时部署到测试环境", d.Footer().Handle)
	xui.ItemWidth(cb, 168)
	// 复选框先铺宿主底色再画指示器; 页脚带是 bg-muted/50 叠在面板底色上,
	// 默认的 t.Card 会在灰带上露出白块, 按页脚实际颜色铺底。
	cb.SetBackdrop(func(t *xui.Theme) xui.Color {
		return t.Popover.Mix(t.Muted, 0.5)
	})

	d.AddFooter("取消", ui.ButtonOutline, func() { d.Close() })
	d.AddFooter("发布", ui.ButtonPrimary, func() { d.Close() })
}

// buildAlertDialogs 四个 AlertDialog 示例: 默认 / 小尺寸 / 带媒体块 / 破坏性。
func buildAlertDialogs(hWindow int, defaultTrigger, smallTrigger, mediaTrigger, destructiveTrigger *ui.Button) {
	basic := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogTitle("Are you absolutely sure?"),
		ui.WithAlertDialogDescription("This action cannot be undone. This will permanently delete your account and remove your data from our servers."))
	basic.AddCancel("Cancel", nil)
	basic.AddAction("Continue", ui.ButtonPrimary, nil)
	basic.BindTrigger(defaultTrigger)
	// 页脚内容槽: 自定义说明在左、按钮靠右(AlertDialogFooter 里按钮之外的任意内容)
	note := ui.NewLabel(0, 0, 0, 20, "This action is permanent.", basic.Footer().Handle).SetMuted()
	xui.ItemWidth(note, 190)
	basic.Refresh()

	small := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogSize(ui.AlertDialogSizeSmall),
		ui.WithAlertDialogTitle("Allow accessory to connect?"),
		ui.WithAlertDialogDescription("Do you want to allow the USB accessory to connect to this device?"))
	small.AddCancel("Don't allow", nil)
	small.AddAction("Allow", ui.ButtonPrimary, nil)
	small.BindTrigger(smallTrigger)

	media := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogMedia(icons.IconBluetooth),
		ui.WithAlertDialogTitle("Are you absolutely sure?"),
		ui.WithAlertDialogDescription("This will permanently delete your account and remove your data from our servers."))
	media.AddCancel("Cancel", nil)
	media.AddAction("Continue", ui.ButtonPrimary, nil)
	media.BindTrigger(mediaTrigger)

	destructive := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogSize(ui.AlertDialogSizeSmall),
		ui.WithAlertDialogMedia(icons.IconTrash2),
		ui.WithAlertDialogMediaTone(ui.AlertDialogMediaDestructive),
		ui.WithAlertDialogTitle("Delete chat?"),
		ui.WithAlertDialogDescription("This will permanently delete this chat conversation. You can delete any memories saved during this chat in Settings."))
	destructive.AddCancel("Cancel", nil) // AlertDialogCancel 默认就是 outline(有边框)样式
	destructive.AddAction("Delete", ui.ButtonDestructive, nil)
	destructive.BindTrigger(destructiveTrigger)
}

// buildDialogVariants 无右上角关闭钮的对话框 + 嵌套模态示例(对话框里再开 AlertDialog)。
func buildDialogVariants(hWindow int, noCloseTrigger, nestedTrigger *ui.Button) {
	noClose := ui.NewDialog(hWindow, ui.WithDialogCloseButton(false))
	noClose.SetTitle("No Close Button")
	noClose.SetDescription("This dialog doesn't have a close button in the top-right corner.")
	noClose.BindTrigger(noCloseTrigger)
	noClose.AddFooter("Close", ui.ButtonOutline, func() { noClose.Close() })

	// 嵌套: 下层 Dialog 开着一层 AlertDialog。模态栈保证只有栈顶响应 ESC/Tab,
	// 关闭上层后焦点回到下层对话框。
	host := ui.NewDialog(hWindow)
	host.SetTitle("Alert Dialog Example")
	host.SetDescription("Click the button below to open an alert dialog.")
	host.BindTrigger(nestedTrigger)
	nested := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogSize(ui.AlertDialogSizeSmall),
		ui.WithAlertDialogTitle("Are you absolutely sure?"),
		ui.WithAlertDialogDescription("This action cannot be undone. This will permanently delete your account and remove your data from our servers."))
	nested.AddCancel("Cancel", nil)
	nested.AddAction("Continue", ui.ButtonPrimary, nil)
	openNested := ui.NewButton(0, 8, host.ContentWidth(), 0, "Open Alert Dialog", host.Content())
	openNested.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		nested.Show()
		return 0
	})
	host.AddFooter("Close", ui.ButtonOutline, func() { host.Close() })
}
