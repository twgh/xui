package main

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 13: 导航4(时间轴) =====

func buildNav4Page(body *widget.LayoutEle) {
	// ---- 时间轴 ----
	_, vtl := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "时间轴", vtl.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"竖向 start: 左侧轴线 + 节点 + 内容槽; 内置时间/标题/说明, 也能直接加 Label / Item / Card", vtl.Handle))
	tl := ui.NewTimeline(0, 0, 0, 0, vtl.Handle)
	xui.ItemFill(tl)
	tl.AddItem(ui.WithTimelineItemTime("09:24"), ui.WithTimelineItemTitle("订单已创建"),
		ui.WithTimelineItemDescription("订单 #4189 · 共 3 件商品"))
	tl.AddItem(ui.WithTimelineItemTime("10:05"), ui.WithTimelineItemTitle("已付款"),
		ui.WithTimelineItemDescription("微信支付 ¥ 268.00"))
	ship := tl.AddItem(ui.WithTimelineItemTime("14:32"), ui.WithTimelineItemTitle("已发货"),
		ui.WithTimelineItemDescription("顺丰速运 SF1234567890"))
	track := ui.NewItem(0, 0, 0, 0, ship.Handle, ui.WithItemVariant(ui.ItemOutline), ui.WithItemSize(ui.ItemSizeSm))
	xui.ItemFill(track)
	ui.NewItemTitle(track.Content().Handle, "运输中")
	ui.NewItemDescription(track.Content().Handle, "快件已到达【杭州转运中心】")
	tl.AddItem(ui.WithTimelineItemTime("预计明天"), ui.WithTimelineItemTitle("待收货"),
		ui.WithTimelineItemDescription("预计 11 月 6 日送达"))

	// ---- 时间轴 · 两侧交替 ----
	_, vtl2 := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "时间轴 · 两侧交替", vtl2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"alternate: 轴线居中、内容左右交替(第一条在左); 节点支持语义色 / 自定义图标 / 加载态", vtl2.Handle))
	altl := ui.NewTimeline(0, 0, 0, 0, vtl2.Handle, ui.WithTimelineMode(ui.TimelineAlternate))
	xui.ItemFill(altl)
	altl.AddItem(ui.WithTimelineItemTime("09:24"), ui.WithTimelineItemTitle("订单已创建"),
		ui.WithTimelineItemDescription("订单 #4189 · 共 3 件商品"), ui.WithTimelineItemIcon(icons.IconPackage))
	altl.AddItem(ui.WithTimelineItemTime("10:05"), ui.WithTimelineItemTitle("支付失败过"),
		ui.WithTimelineItemDescription("已原路退回, 重新发起支付"),
		ui.WithTimelineItemColor(ui.TimelineColorDestructive), ui.WithTimelineItemIcon(icons.IconCreditCard))
	altl.AddItem(ui.WithTimelineItemTime("14:32"), ui.WithTimelineItemTitle("已发货"),
		ui.WithTimelineItemDescription("顺丰速运 SF1234567890"), ui.WithTimelineItemIcon(icons.IconTruck))
	altl.AddItem(ui.WithTimelineItemTime("昨天"), ui.WithTimelineItemTitle("运输中"),
		ui.WithTimelineItemDescription("快件已到达【杭州转运中心】"),
		ui.WithTimelineItemColor(ui.TimelineColorMuted))
	altl.AddItem(ui.WithTimelineItemTime("预计明天"), ui.WithTimelineItemTitle("待收货"),
		ui.WithTimelineItemDescription("预计 11 月 6 日送达"), ui.WithTimelineItemLoading(true))

	xui.ItemFill(hint(vtl2.Handle, "end: 轴线在右侧, 内容整体右对齐"))
	etl := ui.NewTimeline(0, 0, 0, 0, vtl2.Handle, ui.WithTimelineMode(ui.TimelineEnd))
	xui.ItemFill(etl)
	etl.AddItem(ui.WithTimelineItemTime("09:24"), ui.WithTimelineItemTitle("订单已创建"))
	etl.AddItem(ui.WithTimelineItemTime("14:32"), ui.WithTimelineItemTitle("已发货"),
		ui.WithTimelineItemDescription("顺丰速运 SF1234567890"))
	etl.AddItem(ui.WithTimelineItemTime("预计明天"), ui.WithTimelineItemTitle("待收货"))

	// ---- 时间轴 · 水平 ----
	_, vtl3 := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "时间轴 · 水平", vtl3.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"WithTimelineOrientation(ui.TimelineHorizontal): 条目等宽, 轴线横贯; start / end / alternate 决定内容在轴线上侧还是下侧", vtl3.Handle))
	xui.ItemFill(hint(vtl3.Handle, "start: 轴线在顶部, 内容在下"))
	htl := ui.NewTimeline(0, 0, 0, 0, vtl3.Handle, ui.WithTimelineOrientation(ui.TimelineHorizontal))
	xui.ItemFill(htl)
	for _, s := range [][2]string{
		{"09:24", "订单已创建"}, {"10:05", "已付款"}, {"14:32", "已发货"}, {"明天", "待收货"},
	} {
		htl.AddItem(ui.WithTimelineItemTime(s[0]), ui.WithTimelineItemTitle(s[1]))
	}
	xui.ItemFill(hint(vtl3.Handle, "end: 轴线在底部, 内容在上"))
	htl2 := ui.NewTimeline(0, 0, 0, 0, vtl3.Handle,
		ui.WithTimelineOrientation(ui.TimelineHorizontal), ui.WithTimelineMode(ui.TimelineEnd))
	xui.ItemFill(htl2)
	for _, s := range [][2]string{
		{"v1.0", "需求评审"}, {"v1.1", "开发联调"}, {"v1.2", "灰度发布"}, {"v1.3", "全量上线"},
	} {
		htl2.AddItem(ui.WithTimelineItemTitle(s[1]), ui.WithTimelineItemDescription(s[0]))
	}
	xui.ItemFill(hint(vtl3.Handle, "alternate + 空心节点: 上下交替, 共享一条居中轴线"))
	htl3 := ui.NewTimeline(0, 0, 0, 0, vtl3.Handle,
		ui.WithTimelineOrientation(ui.TimelineHorizontal),
		ui.WithTimelineMode(ui.TimelineAlternate),
		ui.WithTimelineVariant(ui.TimelineOutlined))
	xui.ItemFill(htl3)
	htl3.AddItem(ui.WithTimelineItemTitle("设计"), ui.WithTimelineItemDescription("9 月 1 日"),
		ui.WithTimelineItemIcon(icons.IconPackage))
	htl3.AddItem(ui.WithTimelineItemTitle("开发"), ui.WithTimelineItemDescription("9 月 8 日"),
		ui.WithTimelineItemColor(ui.TimelineColorDestructive))
	htl3.AddItem(ui.WithTimelineItemTitle("测试"), ui.WithTimelineItemDescription("9 月 20 日"),
		ui.WithTimelineItemColor(ui.TimelineColorMuted))
	htl3.AddItem(ui.WithTimelineItemTitle("上线"), ui.WithTimelineItemDescription("10 月 1 日"))
	xui.ItemFill(desc(vtl3.Handle, "水平条目的内容高会实时量测: 上下交替时按两侧最高内容对齐同一条轴线; 窗口缩放后自动重排。"))
}

// buildCollapsibleTextTrigger 演示整宽文字触发区(justify-between: 文字靠左、箭头贴右)。
