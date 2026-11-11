package main

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

// ===== 页 18: 反馈3(侧边 Sheet / 底部 Drawer) =====

// toggleDemoTray 启用/停用演示托盘图标。图标从 exe 的 APP 资源加载
// (demo/winres 由 `xc res make` 生成 .syso), 不需要运行时写临时文件。

func buildFeed3Page(body *widget.LayoutEle, hWindow int) {
	r := row(body.Handle, 0, gap)

	// ---- 列 1: Sheet(四向侧边面板) ----
	_, v := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "侧边面板", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Sheet: 从窗口边缘滑入, bg-black/50 遮罩; 左右占满高, 上/下按内容自撑", v.Handle))

	sheetStatus := ui.NewLabel(0, 0, 0, 20, "点击按钮打开面板", v.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(sheetStatus)
	rs := xui.HStack(v.Handle, 36, 8)
	rightBtn := ui.NewButton(0, 0, 0, 0, "右侧面板", rs.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(rightBtn, 100)
	leftBtn := ui.NewButton(0, 0, 0, 0, "左侧导航", rs.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(leftBtn, 100)
	bottomBtn := ui.NewButton(0, 0, 0, 0, "底部面板", rs.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(bottomBtn, 100)
	buildSheetDemos(hWindow, rightBtn, leftBtn, bottomBtn, sheetStatus)
	xui.ItemFill(desc(v.Handle, "ESC / 遮罩 / X 关闭; Tab 焦点陷阱与 Dialog 一致。"))

	// ---- 列 2: Drawer(底部抽屉 + 抓手拖拽) ----
	_, v2 := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "底部抽屉", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Drawer: 顶部抓手, 向下拖过面板高 1/3 关闭, 未过阈值回弹; max-h-[80vh]", v2.Handle))

	drawerStatus := ui.NewLabel(0, 0, 0, 20, "尚未打开抽屉", v2.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(drawerStatus)
	rd := xui.HStack(v2.Handle, 36, 8)
	drawerBtn := ui.NewButton(0, 0, 0, 0, "打开抽屉", rd.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(drawerBtn, 100)
	buildDrawerDemo(hWindow, drawerBtn, drawerStatus)
	xui.ItemFill(desc(v2.Handle,
		"点遮罩 / ESC 也可关闭; 页脚按钮纵向铺满(与 Sheet 同一套排版)。"))
}

// buildSheetDemos 建右侧表单 / 左侧导航 / 底部说明三个 Sheet。
func buildSheetDemos(hWindow int, rightBtn, leftBtn, bottomBtn *ui.Button, status *ui.Label) {
	set := func(s string) { status.SetText(s).Redraw(false) }

	right := ui.NewSheet(hWindow, ui.WithSheetSide(ui.SheetRight))
	right.SetTitle("编辑资料")
	right.SetDescription("修改你的公开资料, 完成后点击保存。")
	cw := right.ContentWidth()
	ui.NewLabel(0, 0, cw, 20, "显示名称", right.Content()).SetFontSize(14).SetMedium()
	ui.NewInput(0, 24, cw, 32, right.Content()).SetValue("twgh")
	ui.NewLabel(0, 72, cw, 20, "个人主页", right.Content()).SetFontSize(14).SetMedium()
	ui.NewInput(0, 96, cw, 32, right.Content()).SetValue("https://")
	right.AddFooter("保存更改", ui.ButtonPrimary, func() {
		set("右侧面板: 已保存更改")
		right.Close()
	})
	right.AddFooter("取消", ui.ButtonOutline, func() { right.Close() })
	right.OnClose(func() { set("右侧面板已关闭") })
	right.BindTrigger(rightBtn)

	left := ui.NewSheet(hWindow, ui.WithSheetSide(ui.SheetLeft))
	left.SetTitle("导航")
	left.SetDescription("左侧面板适合放导航 / 筛选。")
	lw := left.ContentWidth()
	for i, text := range []string{"账户设置", "通知", "外观", "快捷键", "关于"} {
		ui.NewLabel(0, int32(i*32), lw, 24, text, left.Content()).SetFontSize(14)
	}
	left.AddFooter("关闭", ui.ButtonOutline, func() { left.Close() })
	left.OnClose(func() { set("左侧面板已关闭") })
	left.BindTrigger(leftBtn)

	bottom := ui.NewSheet(hWindow, ui.WithSheetSide(ui.SheetBottom))
	bottom.SetTitle("快捷操作")
	bottom.SetDescription("上 / 下面板宽度铺满、高度按内容自撑。")
	ui.NewLabel(0, 0, bottom.ContentWidth(), 24,
		"这里可以放操作说明或一段短表单。", bottom.Content()).SetFontSize(14).SetMuted()
	bottom.AddFooter("确认", ui.ButtonPrimary, func() {
		set("底部面板: 已确认")
		bottom.Close()
	})
	bottom.AddFooter("取消", ui.ButtonOutline, func() { bottom.Close() })
	bottom.OnClose(func() { set("底部面板已关闭") })
	bottom.BindTrigger(bottomBtn)
}

// buildDrawerDemo 建底部抽屉示例。
func buildDrawerDemo(hWindow int, openBtn *ui.Button, status *ui.Label) {
	dr := ui.NewDrawer(hWindow)
	dr.SetTitle("确认离开?")
	dr.SetDescription("未保存的更改会丢失; 向下拖动顶部抓手也可以关闭。")
	ui.NewLabel(0, 0, dr.ContentWidth(), 24, "上次保存于 2 分钟前", dr.Content()).SetFontSize(14).SetMuted()
	dr.AddFooter("保存并离开", ui.ButtonPrimary, func() {
		dr.Close()
		status.SetText("已保存并离开").Redraw(false)
	})
	dr.AddFooter("取消", ui.ButtonOutline, func() { dr.Close() })
	dr.OnClose(func() { status.SetText("抽屉已关闭").Redraw(false) })
	dr.BindTrigger(openBtn)
}
