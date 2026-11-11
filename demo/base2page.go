package main

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 2: 基础2(徽章 / 分隔线 / 标签 / 头像 / 快捷键 / 加载 / 骨架) =====

func buildBase2Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)

	// ---- 徽章 / 分隔线 / 标签 ----
	_, v2 := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "徽章 · 分隔线 · 标签", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "徽章 6 个变体; 分隔线横/竖; 标签含弱化与禁用态", v2.Handle))

	xui.ItemFill(hint(v2.Handle, "徽章"))
	rg := xui.HStack(v2.Handle, 24, 8)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Default", rg.Handle), 68)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Secondary", rg.Handle, ui.WithBadgeVariant(ui.BadgeSecondary)), 84)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Outline", rg.Handle, ui.WithBadgeVariant(ui.BadgeOutline)), 76)
	rg = xui.HStack(v2.Handle, 24, 8)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Destructive", rg.Handle, ui.WithBadgeVariant(ui.BadgeDestructive)), 92)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Ghost", rg.Handle, ui.WithBadgeVariant(ui.BadgeGhost)), 64)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Link", rg.Handle, ui.WithBadgeVariant(ui.BadgeLink)), 56)

	xui.ItemFill(hint(v2.Handle, "行内图标(inline-start/end) / 加载指示器 / 自定义配色"))
	ri := xui.HStack(v2.Handle, 24, 8)
	ui.NewBadge(0, 0, 0, 0, "Verified", ri.Handle, ui.WithBadgeLeadingIcon(icons.IconBadgeCheck))
	ui.NewBadge(0, 0, 0, 0, "Bookmark", ri.Handle, ui.WithBadgeTrailingIcon(icons.IconBookmark))
	ui.NewBadge(0, 0, 0, 0, "Generating", ri.Handle,
		ui.WithBadgeLoading(), ui.WithBadgeVariant(ui.BadgeSecondary))
	ui.NewBadge(0, 0, 0, 0, "Custom", ri.Handle,
		ui.WithBadgeColors(xui.Hex("#E6F4EA"), xui.Hex("#1E7E34"), xui.Hex("#B7E1CD")))

	xui.ItemFill(hint(v2.Handle, "可点击(链接形态): 回调 + 悬停下划线"))
	rc := xui.HStack(v2.Handle, 24, 8)
	clickBadge := ui.NewBadge(0, 0, 0, 0, "打开文档", rc.Handle,
		ui.WithBadgeVariant(ui.BadgeLink), ui.WithBadgeTrailingIcon(icons.IconArrowUpRight))
	clickTip := ui.NewLabel(0, 0, 0, 20, "点一下试试", rc.Handle).SetMuted().SetFontSize(14)
	xui.ItemWidth(clickTip, 120)
	clickBadge.OnClick(func() { clickTip.SetText("已点击 ✓").Redraw() })

	xui.ItemFill(ui.NewSeparator(0, 0, 0, v2.Handle))

	xui.ItemFill(hint(v2.Handle, "分隔线与标签"))
	rl := xui.HStack(v2.Handle, 24, 12)
	xui.ItemWidth(ui.NewLabel(0, 0, 0, 0, "常规文本", rl.Handle), 96)
	ui.NewSeparator(0, 0, 20, rl.Handle, ui.WithSeparatorVertical())
	xui.ItemWidth(ui.NewLabel(0, 0, 0, 0, "弱化文本", rl.Handle).SetMuted(), 96)
	ui.NewSeparator(0, 0, 20, rl.Handle, ui.WithSeparatorVertical())
	disLabel := ui.NewLabel(0, 0, 0, 0, "禁用文本", rl.Handle)
	xui.ItemWidth(disLabel, 96)
	disLabel.Enable(false)

	// ---- 头像 ----
	buildAvatarCard(xui.VStack(r.Handle, 1, gap).Handle)

	// ---- 第二行: 快捷键 / 加载指示器 / 骨架屏 ----
	r3 := row(body.Handle, 0, gap)

	// 快捷键键帽
	_, vkbd := autoCard(xui.VStack(r3.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "快捷键", vkbd.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Kbd 键帽: 高 20、宽度按文字量测; KbdGroup 负责组合键间距 4", vkbd.Handle))

	xui.ItemFill(hint(vkbd.Handle, "单键 / 组合键"))
	rk := xui.HStack(vkbd.Handle, 24, 12)
	ui.NewKbd(rk.Handle, "Esc")
	gk := ui.NewKbdGroup(rk.Handle)
	ui.NewKbd(gk.Handle, "Ctrl")
	ui.NewKbd(gk.Handle, "K")
	gk = ui.NewKbdGroup(rk.Handle)
	ui.NewKbd(gk.Handle, "⌘")
	ui.NewKbd(gk.Handle, "⇧")
	ui.NewKbd(gk.Handle, "P")
	xui.ItemFill(hint(vkbd.Handle, "快捷键紧跟说明文字"))
	rk2 := xui.HStack(vkbd.Handle, 24, 8)
	gk2 := ui.NewKbdGroup(rk2.Handle)
	ui.NewKbd(gk2.Handle, "Ctrl")
	ui.NewKbd(gk2.Handle, "S")
	xui.ItemWidth(ui.NewLabel(0, 0, 0, 20, "保存", rk2.Handle).SetFontSize(14), 48)

	// 加载指示器
	_, vspin := autoCard(xui.VStack(r3.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "加载指示器", vspin.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Spinner: Loader2 图标持续旋转, 1s/圈; 相位由运行时驱动, 不占组件定时器", vspin.Handle))

	xui.ItemFill(hint(vspin.Handle, "与文字同行"))
	spRow := xui.HStack(vspin.Handle, 24, 8)
	ui.NewSpinner(spRow.Handle)
	spText := ui.NewLabel(0, 0, 0, 20, "正在加载…", spRow.Handle).SetFontSize(14).SetMuted()
	xui.ItemWeight(spText, 1)

	xui.ItemFill(hint(vspin.Handle, "尺寸 16 / 20 / 24"))
	spSizes := xui.HStack(vspin.Handle, 32, 16)
	ui.NewSpinner(spSizes.Handle)
	ui.NewSpinner(spSizes.Handle, ui.WithSpinnerSize(20))
	ui.NewSpinner(spSizes.Handle, ui.WithSpinnerSize(24))

	xui.ItemFill(hint(vspin.Handle, "颜色: muted / primary"))
	spColors := xui.HStack(vspin.Handle, 24, 16)
	ui.NewSpinner(spColors.Handle, ui.WithSpinnerMuted())
	ui.NewSpinner(spColors.Handle, ui.WithSpinnerColorFunc(func(t *xui.Theme) xui.Color {
		return t.Primary
	}))

	// 骨架屏
	_, vsk := autoCard(xui.VStack(r3.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "骨架屏", vsk.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Skeleton: bg-accent + rounded-md, 2s 脉冲(1 → 0.5 → 1)", vsk.Handle))

	xui.ItemFill(hint(vsk.Handle, "头像 + 两行文本"))
	skRow := xui.HStack(vsk.Handle, 40, 12)
	ui.NewSkeleton(0, 0, 40, 40, skRow.Handle, ui.WithSkeletonCircle())
	skCol := widget.NewLayoutEle(0, 0, 0, 0, skRow.Handle)
	skCol.EnableHorizon(false)
	skCol.EnableLayout(true)
	skCol.EnableBkTransparent(true)
	skCol.EnableAutoWrap(false)
	skCol.SetSpace(8)
	skCol.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	skCol.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ui.NewSkeleton(0, 0, 180, 12, skCol.Handle)
	ui.NewSkeleton(0, 0, 120, 12, skCol.Handle)

	xui.ItemFill(hint(vsk.Handle, "卡片占位"))
	xui.ItemFill(ui.NewSkeleton(0, 0, 0, 72, vsk.Handle))
	xui.ItemFill(ui.NewSkeleton(0, 0, 0, 12, vsk.Handle))
	lastSk := ui.NewSkeleton(0, 0, 0, 12, vsk.Handle)
	xui.ItemFill(lastSk)
	xui.ItemMarginRight(lastSk, 80)
}

// buildPopoverCard 演示弹出层 / 悬停卡片。
// 放在基础3页顶部, 避免长页面滚到底后深层卡片命中区与绘制位置的累计偏移。

func buildAvatarCard(parent int) {
	_, vav := autoCard(parent, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "头像", vav.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Avatar: sm 24 / default 32 / lg 40; 圆形裁图 / 首字母回落 / 徽标 / 叠放组", vav.Handle))

	xui.ItemFill(hint(vav.Handle, "尺寸 / 图片 / 徽标"))
	avRow := xui.HStack(vav.Handle, 40, 14)
	ui.NewAvatar(0, 0, 0, 0, avRow.Handle, "JD")
	ui.NewAvatar(0, 0, 0, 0, avRow.Handle, "X",
		ui.WithAvatarImageData(demoAvatar()), ui.WithAvatarBadge(icons.IconCheck))
	ui.NewAvatar(0, 0, 0, 0, avRow.Handle, "AB", ui.WithAvatarSize(ui.AvatarSizeSm), ui.WithAvatarRing())
	ui.NewAvatar(0, 0, 0, 0, avRow.Handle, "L", ui.WithAvatarSize(ui.AvatarSizeLg), ui.WithAvatarBadge(""))

	xui.ItemFill(hint(vav.Handle, "叠放组(-space-x-2) + 计数"))
	grp := ui.NewAvatarGroup(vav.Handle)
	grp.Add("A")
	grp.Add("B")
	grp.Add("C")
	grp.AddCount(3)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20,
		"AvatarGroup 用横向布局的负间距叠放; 组内头像默认带 2px 背景色环", vav.Handle).SetMuted().SetFontSize(14))
}
