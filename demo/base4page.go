package main

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 4: 基础4(滚动容器 / 列表项 / 手风琴) =====

func buildBase4Page(body *widget.LayoutEle) {
	// ---- 滚动容器(组件形态; 页面本身也是它, 见 newPage) ----
	_, vs := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "滚动容器", vs.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"ui.ScrollArea: 原生 LayoutFrame 载体, 内容比容器高时纵向滚动、滚动条自动显隐,"+
			"内容区高按内容自撑(Auto)", vs.Handle))
	box := ui.NewScrollArea(0, 0, 0, 180, vs.Handle)
	xui.ItemFill(box)
	for i := 1; i <= 12; i++ {
		l := ui.NewLabel(0, 0, 0, 0, fmt.Sprintf("第 %d 行 · 滚动容器里的普通内容", i), box.Content().Handle)
		xui.ItemFill(l)
	}
	box.Content().SetSpace(6)

	// ---- 列表项(Item): 媒体 / 内容 / 动作 + 页眉页脚 ----
	_, vi := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "列表项", vi.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Item: default / outline / muted 三个变体, default / sm 两档尺寸; Media + Content(吃剩余宽度) + Actions, 可选 Header/Footer", vi.Handle))

	ri := row(vi.Handle, 0, 16)
	itL := xui.VStack(ri.Handle, 1, 10)
	itR := xui.VStack(ri.Handle, 1, 10)

	group := ui.NewItemGroup(0, 0, 0, 0, itL.Handle)
	xui.ItemFill(group)

	// 1. outline + icon media + 尾部按钮
	it1 := ui.NewItem(0, 0, 0, 0, group.Handle, ui.WithItemVariant(ui.ItemOutline))
	xui.ItemFill(it1)
	ui.NewItemMedia(it1.Media().Handle, icons.IconShieldCheck,
		ui.WithItemMediaVariant(ui.ItemMediaIcon))
	c1 := it1.Content()
	ui.NewItemTitle(c1.Handle, "安全提醒")
	ui.NewItemDescription(c1.Handle, "检测到一台未识别的设备登录了你的账户。")
	ui.NewButton(0, 0, 0, 0, "查看", it1.Actions().Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm))

	ui.NewItemSeparator(group.Handle)

	// 2. muted + Header(标题吃权重, 徽章贴右) + Footer
	it2 := ui.NewItem(0, 0, 0, 0, group.Handle, ui.WithItemVariant(ui.ItemMuted))
	xui.ItemFill(it2)
	itHead := it2.Header()
	headTitle := ui.NewItemTitle(itHead.Handle, "部署完成")
	xui.ItemWeight(headTitle, 1)
	okBadge := ui.NewBadge(0, 0, 0, 0, "成功", itHead.Handle, ui.WithBadgeVariant(ui.BadgeSecondary))
	xui.ItemWidth(okBadge, 56)
	c2 := it2.Content()
	ui.NewItemDescription(c2.Handle, "prod-2026.10.24 已发布到生产环境, 共 42 个文件。")
	itFoot := it2.Footer()
	footHint := ui.NewItemDescription(itFoot.Handle, "2 分钟前")
	xui.ItemWeight(footHint, 1)
	ui.NewButton(0, 0, 0, 0, "查看日志", itFoot.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeSm))

	ui.NewItemSeparator(group.Handle)

	// 3. default + sm 尺寸, 无动作区
	it3 := ui.NewItem(0, 0, 0, 0, group.Handle, ui.WithItemSize(ui.ItemSizeSm))
	xui.ItemFill(it3)
	ui.NewItemMedia(it3.Media().Handle, icons.IconFolderCode)
	c3 := it3.Content()
	ui.NewItemTitle(c3.Handle, "design-system")
	ui.NewItemDescription(c3.Handle, "昨天更新 · 12 个组件")

	// 右列: 内容 / 动作 / 自定义媒体槽
	group2 := ui.NewItemGroup(0, 0, 0, 0, itR.Handle)
	xui.ItemFill(group2)

	it4 := ui.NewItem(0, 0, 0, 0, group2.Handle, ui.WithItemVariant(ui.ItemOutline))
	xui.ItemFill(it4)
	c4 := it4.Content()
	ui.NewItemTitle(c4.Handle, "团队成员")
	ui.NewItemDescription(c4.Handle, "邀请成员加入当前工作区, 可随时移除。")
	act4 := it4.Actions()
	countBadge := ui.NewBadge(0, 0, 0, 0, "3 人", act4.Handle)
	xui.ItemWidth(countBadge, 52)
	ui.NewButton(0, 0, 0, 0, "邀请", act4.Handle,
		ui.WithSize(ui.ButtonSizeSm), ui.WithButtonLeadingIcon(icons.IconPlus))

	ui.NewItemSeparator(group2.Handle)

	it5 := ui.NewItem(0, 0, 0, 0, group2.Handle, ui.WithItemVariant(ui.ItemMuted), ui.WithItemSize(ui.ItemSizeSm))
	xui.ItemFill(it5)
	// 自定义媒体槽: 不限于图标, 任何元素都可以放进来。
	avatar := widget.NewElement(0, 0, 32, 32, it5.Media().Handle)
	avatar.EnableBkTransparent(true)
	avatar.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 32)
	avatar.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 32)
	avatar.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		th := xui.Current()
		p.FillRound(&p.Rc, th.Primary, th.Pill(32))
		return 0
	})
	xui.AttachWindow(avatar.Handle)
	c5 := it5.Content()
	ui.NewItemTitle(c5.Handle, "twgh")
	ui.NewItemDescription(c5.Handle, "工作区管理员")

	ui.NewItemSeparator(group2.Handle)

	it6 := ui.NewItem(0, 0, 0, 0, group2.Handle)
	xui.ItemFill(it6)
	h6 := it6.Header()
	bill := ui.NewLabel(0, 0, 0, 20, "本月账单", h6.Handle).SetFontSize(14).SetMuted()
	xui.ItemWeight(bill, 1)
	price := ui.NewLabel(0, 0, 96, 20, "¥ 128.00", h6.Handle).SetFontSize(14).SetMedium()
	xui.ItemWidth(price, 96)

	// ---- 手风琴(Accordion): 复用 Collapsible 的高度过渡 ----
	_, va := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "手风琴", va.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Accordion: 触发区 hover 下划线 + 箭头随开合切换; single = 单开互斥(可全收), multiple = 各条目独立; 展开高度过渡 200ms", va.Handle))

	acc := ui.NewAccordion(0, 0, 0, 0, va.Handle)
	xui.ItemFill(acc)
	// 内容区里的标签必须自己声明宽度(ItemFill): 布局容器不会替子项铺满,
	// Label 的 cx 传 0 就是 0 宽, 文字画不出来(整段内容看起来是空的)。
	a1 := acc.AddItem("item-1", "这是什么?")
	xui.ItemFill(ui.NewLabel(0, 0, 0, 24,
		"一套 shadcn-ui 风格的自绘组件, 触发区与内容区之间有 1px 分隔线, 展开高度带 200ms 过渡。",
		a1.ContentStack(6).Handle))
	a2 := acc.AddItem("item-2", "可以折叠吗?")
	xui.ItemFill(ui.NewLabel(0, 0, 0, 24,
		"可以。single 模式下同一时间至多一项展开, 点开另一项会自动收起当前项; 全部收起也允许。",
		a2.ContentStack(6).Handle))
	a3 := acc.AddItem("item-3", "键盘操作")
	xui.ItemFill(ui.NewLabel(0, 0, 0, 24,
		"触发区获得键盘焦点时显示焦点环, 空格 / 回车切换开合(Tab 在条目间移动)。",
		a3.ContentStack(6).Handle))
	a1.SetOpen(true)

	xui.ItemFill(hint(va.Handle, "multiple(各条目独立开合)"))
	acc2 := ui.NewAccordion(0, 0, 0, 0, va.Handle, ui.WithAccordionMultiple())
	xui.ItemFill(acc2)
	m1 := acc2.AddItem("m-1", "多开示例一")
	xui.ItemFill(ui.NewLabel(0, 0, 0, 24, "开合状态互不影响, 与 single 模式的互斥行为对比。", m1.ContentStack(6).Handle))
	m2 := acc2.AddItem("m-2", "多开示例二")
	xui.ItemFill(ui.NewLabel(0, 0, 0, 24, "同时展开多项时, 每一项的内容区独立做高度过渡。", m2.ContentStack(6).Handle))

	// ---- 状态栏(StatusBar): 左中右槽 + 条目提示 + 尺寸抓手 ----
	_, vst := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "状态栏", vst.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"StatusBar: 整宽横条 + 顶边 1px; 左/中/右三槽可放任意控件; 条目带 hover 过渡与上方气泡; 抓手拖动改窗口大小",
		vst.Handle))

	statusClick := ui.NewLabel(0, 0, 0, 20, "点击条目: -", vst.Handle).SetMuted()
	xui.ItemFill(statusClick)
	// 卡片里的样例关掉抓手: 抓手改的是宿主窗口, 演示页里只留窗口底部那条可拖。
	sb := ui.NewStatusBar(0, 0, 0, 0, vst.Handle,
		ui.WithStatusBarGrip(false), ui.WithStatusBarTooltip(tip))
	xui.ItemFill(sb)
	sb.AddLeft("就绪", ui.WithStatusBarItemIcon(icons.IconCircleCheck),
		ui.WithStatusBarItemTooltip("构建完成"))
	ui.NewStatusBarSeparator(sb.Left().Handle)
	sb.AddLeft("main", ui.WithStatusBarItemIcon(icons.IconGitBranch),
		ui.WithStatusBarItemTooltip("当前分支"),
		ui.WithStatusBarItemOnClick(func() { statusClick.SetText("点击条目: main").Redraw() }))
	sb.AddCenter("xui · 组件演示", ui.WithStatusBarItemTooltip("页面标题"))
	sb.AddRight("UTF-8", ui.WithStatusBarItemTooltip("文件编码"))
	sb.AddRight("Go", ui.WithStatusBarItemActive(), ui.WithStatusBarItemTooltip("语言模式"))
	sb.AddRight("只读", ui.WithStatusBarItemDisabled(), ui.WithStatusBarItemTooltip("只读模式(禁用)"))

	xui.ItemFill(hint(vst.Handle, "同一组条目换成小高度 / 带尺寸读数 / 无抓手"))
	sb2 := ui.NewStatusBar(0, 0, 0, 24, vst.Handle,
		ui.WithStatusBarGrip(false), ui.WithStatusBarSizeReadout(), ui.WithStatusBarTooltip(tip))
	xui.ItemFill(sb2)
	sb2.AddLeft("3 个问题", ui.WithStatusBarItemIcon(icons.IconInfo))
	sb2.AddLeft("0 个警告")
	sb2.AddCenter("Ctrl+K 打开命令面板")
	sb2.AddRight("Ln 12, Col 34", ui.WithStatusBarItemTooltip("光标位置"))
}
