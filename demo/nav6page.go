package main

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 15: 导航6(侧边栏) =====

// buildNav6Page 展示 Sidebar: inset 外观 + 图标折叠 + 分组/子菜单/徽章/动作 + 折叠态 tooltip。
// 外壳固定在 560 高, 面板内容超出时在侧栏内部滚动。
func buildNav6Page(body *widget.LayoutEle) {
	_, v := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "侧边栏", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Sidebar: inset 外观 + 图标折叠; 分组/子菜单/徽章/动作钮; 子菜单点父项开合(chevron 旋转); 收起后悬停菜单项出右侧提示; Ctrl+B 开合", v.Handle))

	sb := ui.NewSidebar(0, 0, 0, 560, v.Handle,
		ui.WithSidebarVariant(ui.SidebarVariantInset),
		ui.WithSidebarCollapsible(ui.SidebarCollapsibleIcon),
		ui.WithSidebarRail())
	xui.ItemFill(sb)
	xui.ItemHeight(sb, 560)

	// ---- 主内容区: 顶部栏(触发钮 + 标题) + 内容卡片 ----
	main := sb.Main()
	top := xui.HStack(main.Handle, 48, 8)
	top.SetPadding(12, 0, 12, 0)
	trig := sb.AddTrigger(top.Handle)
	tip.Bind(trig.Handle, "切换侧栏 (Ctrl+B)")
	xui.ItemWeight(ui.NewLabel(0, 0, 0, 24, "模型工作台", top.Handle), 1)
	topSep := ui.NewSeparator(0, 0, 0, main.Handle)
	xui.ItemFill(topSep)

	_, vm := autoCard(main.Handle, 10)
	vm.LayoutItem_SetMargin(16, 8, 16, 16)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "今天的用量", vm.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"侧栏收起成图标栏后, 悬停菜单项会弹出右侧提示; 点分隔条也能开合。", vm.Handle))
	status := ui.NewLabel(0, 0, 0, 24, "最近操作: —", vm.Handle).SetMuted()
	xui.ItemFill(status)
	sep := ui.NewSeparator(0, 0, 0, vm.Handle)
	xui.ItemFill(sep)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 24, "推理用量 12.4K · 本月调用 86.2% · 平均延迟 142ms", vm.Handle))

	// ---- 侧栏内容: 头部品牌 + 两组菜单 + 脚部用户 ----
	sb.AddHeaderItem("Acme Inc.", icons.IconGalleryVerticalEnd,
		ui.WithSidebarItemSize(ui.SidebarSizeLg), ui.WithSidebarItemTooltip("Acme Inc."))

	var active *ui.SidebarMenuButton
	setActive := func(it *ui.SidebarMenuButton, name string) {
		if active != nil && active != it {
			active.SetActive(false)
		}
		active = it
		it.SetActive(true)
		status.SetText("最近操作: 打开 " + name)
	}

	g1 := sb.AddGroup("平台")
	var model, projects, settings *ui.SidebarMenuButton
	model = g1.AddItem("模型", icons.IconMonitor,
		ui.WithSidebarItemActive(true), ui.WithSidebarItemTooltip("模型"),
		ui.WithSidebarItemOnClick(func() { setActive(model, "模型") }))
	active = model
	projects = g1.AddItem("项目", icons.IconFolder,
		ui.WithSidebarItemTooltip("项目"), ui.WithSidebarItemBadge("12"),
		ui.WithSidebarItemOnClick(func() { setActive(projects, "项目") }))
	settings = g1.AddItem("设置", icons.IconSettings,
		ui.WithSidebarItemTooltip("设置"),
		ui.WithSidebarItemAction(icons.IconEllipsis, func() { status.SetText("最近操作: 设置 · 更多操作") }),
		ui.WithSidebarItemActionOnHover(),
		ui.WithSidebarItemOnClick(func() { setActive(settings, "设置") }))
	g1.SetAction(icons.IconPlus, func() { status.SetText("最近操作: 新建分组条目") })

	groupSep := sb.AddSeparator()
	_ = groupSep

	g2 := sb.AddGroup("项目")
	var design, docs *ui.SidebarMenuButton
	design = g2.AddItem("设计系统", icons.IconPalette,
		ui.WithSidebarItemTooltip("设计系统"),
		ui.WithSidebarItemOnClick(func() { setActive(design, "设计系统") }))
	sub := design.AddSub()
	var subLib, subTokens *ui.SidebarMenuButton
	subLib = sub.AddItem("组件库", "", ui.WithSidebarItemActive(true),
		ui.WithSidebarItemOnClick(func() { setActive(subLib, "组件库") }))
	subTokens = sub.AddItem("设计令牌", "",
		ui.WithSidebarItemOnClick(func() { setActive(subTokens, "设计令牌") }))
	docs = g2.AddItem("文档", icons.IconFileText,
		ui.WithSidebarItemTooltip("文档"),
		ui.WithSidebarItemOnClick(func() { setActive(docs, "文档") }))

	sb.AddFooterItem("twgh", icons.IconUser,
		ui.WithSidebarItemSize(ui.SidebarSizeLg), ui.WithSidebarItemTooltip("twgh"))

	xui.ItemFill(hint(v.Handle,
		"点触发钮或 Ctrl+B 收起; 收起后菜单项只留图标, 悬停出提示; 点子菜单父项可展开/收起(右侧 chevron 旋转); 子菜单/徽章/动作随折叠隐藏。"))
}
