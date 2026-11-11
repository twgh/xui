package main

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 10: 导航1(命令面板 / 下拉菜单 / 右键菜单) =====

// buildCommandCard 命令面板演示; 放导航1页顶部, 保证输入框/条目的命中区无需滚动即可用。
func buildCommandCard(body *widget.LayoutEle) {
	_, vcmd := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "命令面板", vcmd.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Command: 输入即过滤, ↑/↓ 跳过禁用, Enter 选中, ESC 清空; 列表超高自动滚动", vcmd.Handle))

	cmdRow := xui.HStack(vcmd.Handle, 360, 24)
	cmdStatus := ui.NewLabel(0, 0, 0, 20, "选中: (无)", vcmd.Handle).SetMuted().SetFontSize(14)
	xui.ItemFill(cmdStatus)
	cmd := ui.NewCommand(0, 0, 560, 360, cmdRow.Handle,
		ui.WithCommandOnSelect(func(v string) {
			cmdStatus.SetText("选中: " + v)
			cmdStatus.Redraw(false)
		}))
	sug := cmd.AddGroup("建议")
	sug.AddItem("calendar", "日历", ui.WithCommandItemIcon(icons.IconCalendar), ui.WithCommandItemShortcut("Ctrl+C"))
	sug.AddItem("emoji", "搜索表情", ui.WithCommandItemIcon(icons.IconSmile))
	sug.AddItem("calculator", "计算器", ui.WithCommandItemIcon(icons.IconCalculator), ui.WithCommandItemShortcut("Ctrl+Shift+C"))
	sug.AddItem("user", "用户资料", ui.WithCommandItemIcon(icons.IconUser))
	cmd.AddSeparator()
	set := cmd.AddGroup("设置")
	set.AddItem("profile", "个人资料", ui.WithCommandItemIcon(icons.IconUser), ui.WithCommandItemShortcut("Ctrl+P"))
	set.AddItem("settings", "设置", ui.WithCommandItemIcon(icons.IconSettings), ui.WithCommandItemShortcut("Ctrl+S"))
	set.AddItem("keyboard", "键盘快捷键", ui.WithCommandItemIcon(icons.IconKeyboard), ui.WithCommandItemKeywords("快捷键 hotkey"))
	set.AddItem("help", "帮助与反馈", ui.WithCommandItemIcon(icons.IconCircleHelp))
	set.AddItem("disabled", "禁用条目", ui.WithCommandItemIcon(icons.IconX), ui.WithCommandItemDisabled())

	cmdInfo := xui.VStackWeight(cmdRow.Handle, 1, 10)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "键盘优先", cmdInfo.Handle).SetMedium().SetFontSize(14))
	for _, text := range []string{
		"输入即过滤(不区分大小写, 支持 keywords)",
		"↑/↓ 循环移动, 自动跳过禁用项",
		"Home / End 跳到首尾",
		"Enter 选中并回写状态",
		"ESC 清空搜索词",
	} {
		xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "· "+text, cmdInfo.Handle).SetMuted().SetFontSize(14))
	}
	focusCmd := ui.NewButton(0, 0, 0, 0, "聚焦搜索框", cmdInfo.Handle, ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm))
	focusCmd.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		cmd.Focus()
		return 0
	})
	xui.ItemFill(focusCmd)
}

func buildNav1Page(body *widget.LayoutEle) {
	buildCommandCard(body)

	r := row(body.Handle, 0, gap)

	// ---- 列 1: 下拉菜单 ----
	_, v2 := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "下拉菜单", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "原生菜单承载行为 + 窗口级自绘皮肤", v2.Handle))
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0,
		"条目 / 勾选 / 单选组 / 子菜单 / 快捷键 / 分组标签 / 禁用 / 破坏性", v2.Handle).SetMuted())
	rm := xui.HStack(v2.Handle, 36, 8)
	menuBtn := ui.NewButton(0, 0, 0, 0, "打开菜单", rm.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithButtonHasPopup(),
		ui.WithButtonTrailingIcon(icons.IconChevronDown))
	xui.ItemWidth(menuBtn, 120)
	menuBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		// PopupAtElement 阻塞到菜单关闭; 期间保持 aria-expanded 的底色, 且不整体下沉。
		menuBtn.SetExpanded(true)
		showDemoMenu(hEle)
		menuBtn.SetExpanded(false)
		return 0
	})
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "条目 hover 走 bg-accent, 与表格行 / 树行同一口径", v2.Handle).SetMuted())

	// ---- 列 2: 右键菜单 ----
	_, v3 := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "右键菜单", v3.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"右键触发区: 含子元素, 每次弹出按状态重建", v3.Handle))
	contextTrigger(v3.Handle)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "前置图标 / inset / 子菜单 / 勾选 / 单选 / 破坏性", v3.Handle).SetMuted())
}
