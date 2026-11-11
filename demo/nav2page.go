package main

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 11: 导航2(标签页 / 面包屑 / 分页) =====

func buildNav2Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)

	// ---- 列 1: 标签页 ----
	_, v := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "标签页", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "default / line 两个变体 + 纵向取向; 行内图标、单标签禁用、键盘导航", v.Handle))

	xui.ItemFill(hint(v.Handle, "default · 行内图标 / 单项禁用"))
	tabs := ui.NewTabsItems(0, 0, 240, v.Handle, []ui.TabsItem{
		{Text: "账户", Icon: icons.Path("user")},
		{Text: "密码", Icon: icons.Path("lock"), Disabled: true},
		{Text: "通知", Icon: icons.Path("bell")},
	}, ui.WithTabsValues("account", "password", "notice"))
	xui.ItemWidth(tabs, 240)
	panel := ui.NewLabel(0, 0, 0, 0, "账户设置面板", v.Handle).SetMuted()
	xui.ItemFill(panel)
	tabs.OnChange(func(i int32) {
		panel.SetText([]string{"账户设置面板", "密码设置面板", "通知设置面板"}[i]).Redraw()
	})

	xui.ItemFill(hint(v.Handle, "line"))
	// Tabs 的标签宽度按创建时给的 cx 等分, 不能被布局拉伸, 所以用固定宽度
	lineTabs := ui.NewTabs(0, 0, 200, v.Handle, []string{"概览", "详情"},
		ui.WithTabsVariant(ui.TabsLine), ui.WithTabsActive(1))
	xui.ItemWidth(lineTabs, 200)

	xui.ItemFill(hint(v.Handle, "纵向 · 图标"))
	vTabs := ui.NewTabsItems(0, 0, 112, v.Handle, []ui.TabsItem{
		{Text: "账户", Icon: icons.Path("user")},
		{Text: "密码", Icon: icons.Path("lock")},
		{Text: "通知", Icon: icons.Path("bell")},
	}, ui.WithTabsVertical())
	xui.ItemWidth(vTabs, 112)

	xui.ItemFill(desc(v.Handle, "键盘: 整块列表是一个 Tab 站点, ←/→ 或 ↑/↓ 移动焦点, Home/End 首末; 默认(automatic)移动即切换, WithTabsManual 则 Enter/Space 才切换。"))

	// ---- 分页 ----
	_, v5 := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "分页", v5.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"折叠采用恒定 7 槽位: 首段/中段/末段占位宽度一致, 连点同一坐标不会点空", v5.Handle))

	pagInfo := ui.NewLabel(0, 0, 0, 0, "", v5.Handle).SetMuted()
	xui.ItemFill(pagInfo)
	pag := ui.NewPagination(0, 0, 0, 0, v5.Handle,
		ui.WithPrevText("上一页"), ui.WithNextText("下一页"))
	xui.ItemFill(pag)
	pagInfo.SetText("共 120 条 · 每页 10 条 · 当前第 5 页")
	pag.SetTotal(120)
	pag.SetPageSize(10)
	pag.SetPage(5)
	pag.OnChange(func(p int) {
		pagInfo.SetText(fmt.Sprintf("共 120 条 · 每页 10 条 · 当前第 %d 页", p))
		pagInfo.Redraw(false)
	})

	// 简洁(只有页码) / 仅上一页下一页 / 只留 chevron 三种形态, 对应文档的三个示例
	xui.ItemFill(hint(v5.Handle, "简洁: 只有页码(WithPaginationSimple)"))
	pagSimple := ui.NewPagination(0, 0, 0, 0, v5.Handle, ui.WithPaginationSimple())
	xui.ItemFill(pagSimple)
	pagSimple.SetTotal(50)

	xui.ItemFill(hint(v5.Handle, "仅导航: 上一页 / 下一页(WithPaginationNavOnly, 配合每页行数)"))
	pagNav := ui.NewPagination(0, 0, 0, 0, v5.Handle,
		ui.WithPaginationNavOnly(), ui.WithPrevText("上一页"), ui.WithNextText("下一页"))
	xui.ItemFill(pagNav)
	pagNav.SetTotal(120)

	xui.ItemFill(hint(v5.Handle, "只留图标(WithPaginationHideText)"))
	pagIcon := ui.NewPagination(0, 0, 0, 0, v5.Handle, ui.WithPaginationHideText())
	xui.ItemFill(pagIcon)
	pagIcon.SetTotal(120)
	xui.ItemFill(desc(v5.Handle, "Tab 进入后用 ←/→ 移动、Home/End 首末、Enter/空格翻页; 焦点环只在键盘操作时出现。"))

	// ---- 列 2: 面包屑 ----
	_, vbc := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "面包屑", vbc.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Breadcrumb: 链接 muted → 悬停过渡到前景色; 分隔符 14, 条目间距 6, 列表间距 10 并自动换行", vbc.Handle))

	bcInfo := ui.NewLabel(0, 0, 0, 20, "点击任意链接查看回调", vbc.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(bcInfo)

	xui.ItemFill(hint(vbc.Handle, "完整路径"))
	bc := ui.NewBreadcrumb(0, 0, 0, 0, vbc.Handle)
	xui.ItemFill(bc)
	addBCLink := func(text string) {
		it := ui.NewBreadcrumbItem(bc.Handle)
		ui.NewBreadcrumbLink(it.Handle, text).OnClick(func() {
			bcInfo.SetText("点击了「" + text + "」").Redraw(false)
		})
	}
	addBCLink("首页")
	ui.NewBreadcrumbSeparator(bc.Handle)
	addBCLink("组件")
	ui.NewBreadcrumbSeparator(bc.Handle)
	itCur := ui.NewBreadcrumbItem(bc.Handle)
	ui.NewBreadcrumbPage(itCur.Handle, "面包屑")

	xui.ItemFill(hint(vbc.Handle, "折叠省略号(36×36, 可自行挂菜单事件)"))
	bc2 := ui.NewBreadcrumb(0, 0, 0, 0, vbc.Handle)
	xui.ItemFill(bc2)
	itHome := ui.NewBreadcrumbItem(bc2.Handle)
	ui.NewBreadcrumbLink(itHome.Handle, "首页")
	ui.NewBreadcrumbSeparator(bc2.Handle)
	ui.NewBreadcrumbEllipsis(bc2.Handle)
	ui.NewBreadcrumbSeparator(bc2.Handle)
	itSet := ui.NewBreadcrumbItem(bc2.Handle)
	ui.NewBreadcrumbLink(itSet.Handle, "设置")
	ui.NewBreadcrumbSeparator(bc2.Handle)
	itMe := ui.NewBreadcrumbItem(bc2.Handle)
	ui.NewBreadcrumbPage(itMe.Handle, "个人资料")
}
