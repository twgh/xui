package main

import (
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 1: 基础1(按钮 / 按钮组 / 切换按钮) =====

func buildBase1Page(body *widget.LayoutEle) {
	// 按钮卡较高, 右侧列放切换按钮; 按钮组单独占一行铺满。
	r := row(body.Handle, 0, gap)
	colL := xui.VStack(r.Handle, 5, gap)
	colR := xui.VStack(r.Handle, 4, gap)

	// ---- 左列: 按钮 ----
	_, v := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "按钮", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "6 个变体 · 4 档尺寸 + 4 个方形图标档, 悬停 150ms 过渡", v.Handle))

	xui.ItemFill(hint(v.Handle, "变体"))
	rb := xui.HStack(v.Handle, 36, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Default", rb.Handle), 104)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Secondary", rb.Handle, ui.WithVariant(ui.ButtonSecondary)), 112)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Outline", rb.Handle, ui.WithVariant(ui.ButtonOutline)), 104)
	rb = xui.HStack(v.Handle, 36, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Ghost", rb.Handle, ui.WithVariant(ui.ButtonGhost)), 96)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Destructive", rb.Handle, ui.WithVariant(ui.ButtonDestructive)), 112)
	linkBtn := ui.NewButton(0, 0, 0, 0, "Link", rb.Handle, ui.WithVariant(ui.ButtonLink))
	xui.ItemWidth(linkBtn, 88)
	disBtn := ui.NewButton(0, 0, 0, 0, "Disabled", rb.Handle)
	xui.ItemWidth(disBtn, 104)
	disBtn.Enable(false)

	xui.ItemFill(hint(v.Handle, "尺寸档位(Xs / Sm / Default / Lg)"))
	rs := xui.HStack(v.Handle, 40, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Xs", rs.Handle, ui.WithSize(ui.ButtonSizeXs)), 56)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Sm", rs.Handle, ui.WithSize(ui.ButtonSizeSm)), 60)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Default", rs.Handle), 88)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Lg", rs.Handle, ui.WithSize(ui.ButtonSizeLg)), 68)

	xui.ItemFill(hint(v.Handle, "图标按钮(方形档, 宽度由预设给出)"))
	ri := xui.HStack(v.Handle, 40, 8)
	iconBtn := ui.NewButton(0, 0, 0, 0, "", ri.Handle, ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconPlus))
	tip.Bind(iconBtn.Handle, "Icon")
	iconBtn = ui.NewButton(0, 0, 0, 0, "", ri.Handle, ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconCheck))
	tip.Bind(iconBtn.Handle, "Icon(另一个)")
	ui.NewButton(0, 0, 0, 0, "", ri.Handle, ui.WithSize(ui.ButtonSizeIconSm), ui.WithIcon(icons.IconSettings))
	ui.NewButton(0, 0, 0, 0, "", ri.Handle, ui.WithSize(ui.ButtonSizeIconXs), ui.WithIcon(icons.IconX))
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "", ri.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithIcon(icons.IconMoon)), 36)

	// 校验失败态(aria-invalid: 红边框 + 3px 红环, 常驻不要求焦点)
	xui.ItemFill(hint(v.Handle, "校验失败(aria-invalid)"))
	rv := xui.HStack(v.Handle, 36, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Invalid", rv.Handle, ui.WithButtonInvalid()), 104)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Outline", rv.Handle, ui.WithVariant(ui.ButtonOutline), ui.WithButtonInvalid()), 104)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Destructive", rv.Handle, ui.WithVariant(ui.ButtonDestructive), ui.WithButtonInvalid()), 120)

	// ---- 按钮组(与上面的按钮卡同宽, 同处左列) ----
	// 组是宽高自撑的 LayoutEle: 子按钮 cx 传 0 时按文字量宽, 组自动拼成一条,
	// 首尾保留外圆角、相邻边收成直角, 相邻边框自然并成 1px(见 buttongroup.go)。
	_, vbg := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "按钮组", vbg.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"ButtonGroup 横向/纵向、首尾外角、自动量宽; 文本块与分隔条跟随组高; 嵌套组自动 8px 间距", vbg.Handle))

	xui.ItemFill(hint(vbg.Handle, "横向 · 分隔条 · 图标按钮"))
	rbg1 := xui.HStack(vbg.Handle, 40, 16)
	bg1 := ui.NewButtonGroup(rbg1.Handle)
	ui.NewButton(0, 0, 0, 0, "Archive", bg1.Handle, ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "Report", bg1.Handle, ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "Snooze", bg1.Handle, ui.WithVariant(ui.ButtonOutline))
	ui.NewButtonGroupSeparator(bg1.Handle)
	moreBtn := ui.NewButton(0, 0, 0, 0, "", bg1.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconMoreHorizontal),
		ui.WithVariant(ui.ButtonOutline), ui.WithButtonHasPopup())
	tip.Bind(moreBtn.Handle, "更多操作 · 弹层触发区(按下不下沉)")
	moreBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		moreBtn.SetExpanded(true)
		moreBtn.Redraw(true) // 菜单阻塞, 先立即重绘出展开色
		defer moreBtn.SetExpanded(false)
		m := ui.NewMenu()
		m.Width(176)
		m.Item("导出", ui.MenuShortcut("Ctrl+E"))
		m.Item("归档")
		m.Separator()
		m.Item("删除", ui.MenuDestructive())
		m.PopupAtElement(hEle)
		return 0
	})

	xui.ItemFill(hint(vbg.Handle, "文本块 · 分隔条 · 行内图标"))
	rbg2 := xui.HStack(vbg.Handle, 40, 16)
	bg2 := ui.NewButtonGroup(rbg2.Handle)
	ui.NewButtonGroupText("https://", bg2.Handle)
	ui.NewButton(0, 0, 0, 0, "Copy", bg2.Handle,
		ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline))
	ui.NewButtonGroupSeparator(bg2.Handle)
	ui.NewButton(0, 0, 0, 0, "", bg2.Handle,
		ui.WithSize(ui.ButtonSizeIconSm), ui.WithIcon(icons.IconArrowUp), ui.WithVariant(ui.ButtonOutline))

	xui.ItemFill(hint(vbg.Handle, "嵌套组(组间距 8)"))
	rbg3 := xui.HStack(vbg.Handle, 40, 16)
	bg3 := ui.NewButtonGroup(rbg3.Handle)
	bg31 := ui.NewButtonGroup(bg3.Handle)
	for _, s := range []string{"1", "2", "3", "4", "5"} {
		ui.NewButton(0, 0, 0, 0, s, bg31.Handle,
			ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline))
	}
	bg32 := ui.NewButtonGroup(bg3.Handle)
	ui.NewButton(0, 0, 0, 0, "", bg32.Handle,
		ui.WithSize(ui.ButtonSizeIconSm), ui.WithIcon(icons.IconArrowLeft), ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "", bg32.Handle,
		ui.WithSize(ui.ButtonSizeIconSm), ui.WithIcon(icons.IconArrowRight), ui.WithVariant(ui.ButtonOutline))

	xui.ItemFill(hint(vbg.Handle, "纵向组 · 尺寸档(组内统一 rounded-lg)"))
	rbg4 := xui.HStack(vbg.Handle, 80, 16)
	bg4 := ui.NewButtonGroup(rbg4.Handle, ui.WithButtonGroupOrientation(ui.ButtonGroupVertical))
	ui.NewButton(0, 0, 0, 0, "", bg4.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconPlus), ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "", bg4.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconMinus), ui.WithVariant(ui.ButtonOutline))

	bg5 := ui.NewButtonGroup(rbg4.Handle)
	ui.NewButton(0, 0, 0, 0, "Xs", bg5.Handle,
		ui.WithSize(ui.ButtonSizeXs), ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "Sm", bg5.Handle,
		ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "Default", bg5.Handle, ui.WithVariant(ui.ButtonOutline))

	xui.ItemFill(hint(vbg.Handle, "行内图标 · 展开态 · 胶囊 · 加载态"))
	rbg6 := xui.HStack(vbg.Handle, 40, 12)
	ui.NewButton(0, 0, 0, 0, "New Branch", rbg6.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm), ui.WithButtonLeadingIcon(icons.IconGitBranch))
	ui.NewButton(0, 0, 0, 0, "下一步", rbg6.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithButtonTrailingIcon(icons.IconChevronRight))
	filterUnread, filterAttach := false, false
	filterBtn := ui.NewButton(0, 0, 0, 0, "筛选", rbg6.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithButtonTrailingIcon(icons.IconChevronDown),
		ui.WithButtonHasPopup())
	tip.Bind(filterBtn.Handle, "展开态: 点击打开筛选菜单")
	filterBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		filterBtn.SetExpanded(true)
		filterBtn.Redraw(true)             // 菜单是阻塞式原生弹窗, 不立即重绘就看不到展开色
		defer filterBtn.SetExpanded(false) // 菜单阻塞到关闭, 返回后恢复非展开色
		m := ui.NewMenu()
		m.Width(176)
		m.Label("筛选")
		m.Check("仅未读", filterUnread, ui.MenuOnCheck(func(on bool) { filterUnread = on }))
		m.Check("含附件", filterAttach, ui.MenuOnCheck(func(on bool) { filterAttach = on }))
		m.Separator()
		m.Item("清除筛选", ui.MenuOnSelect(func() { filterUnread, filterAttach = false, false }))
		m.PopupAtElement(hEle)
		return 0
	})
	genBtn := ui.NewButton(0, 0, 0, 0, "Generating", rbg6.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm), ui.WithButtonLoading())
	genBtn.Enable(false)
	ui.NewButton(0, 0, 0, 0, "", rbg6.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconArrowUp), ui.WithVariant(ui.ButtonOutline), ui.WithButtonRounded())

	xui.ItemFill(hint(vbg.Handle, "输入框 / 下拉框直接入组(壳体跟随首尾圆角)"))
	rbg7 := xui.HStack(vbg.Handle, 40, 16)
	bg7 := ui.NewButtonGroup(rbg7.Handle)
	search := ui.NewInput(0, 0, 0, 0, bg7.Handle, ui.WithPlaceholder("Search..."))
	xui.ItemWidth(search, 220)
	ui.NewButton(0, 0, 0, 0, "", bg7.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconSearch), ui.WithVariant(ui.ButtonOutline))

	bg8 := ui.NewButtonGroup(rbg7.Handle)
	combo := ui.NewNativeComboBox(0, 0, 0, 0, bg8.Handle,
		[]string{"开发环境", "测试环境", "预发环境"}, ui.WithNativeComboBoxPlaceholder("选择环境"))
	xui.ItemWidth(combo, 180)
	ui.NewButton(0, 0, 0, 0, "", bg8.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconSettings), ui.WithVariant(ui.ButtonOutline))

	// ---- 右列: 切换按钮 / 切换按钮组 ----
	_, vt := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "切换按钮", vt.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Toggle: 双状态按钮, 点击/空格/回车切换; 选中 bg-accent、悬停 bg-muted, outline 带 border-input", vt.Handle))

	xui.ItemFill(hint(vt.Handle, "变体与选中态(点击切换)"))
	rt := xui.HStack(vt.Handle, 40, 12)
	ui.NewToggle(0, 0, 0, 0, "Bold", rt.Handle, ui.WithTogglePressed(true))
	ui.NewToggle(0, 0, 0, 0, "Italic", rt.Handle)
	rt = xui.HStack(vt.Handle, 40, 12)
	ui.NewToggle(0, 0, 0, 0, "Underline", rt.Handle,
		ui.WithToggleVariant(ui.ToggleVariantOutline), ui.WithTogglePressed(true))
	ui.NewToggle(0, 0, 0, 0, "Strikethrough", rt.Handle, ui.WithToggleVariant(ui.ToggleVariantOutline))

	xui.ItemFill(hint(vt.Handle, "尺寸档(Sm / Default / Lg)"))
	rsz := xui.HStack(vt.Handle, 48, 12)
	ui.NewToggle(0, 0, 0, 0, "Sm", rsz.Handle,
		ui.WithToggleSize(ui.ToggleSizeSm), ui.WithToggleVariant(ui.ToggleVariantOutline))
	ui.NewToggle(0, 0, 0, 0, "Default", rsz.Handle, ui.WithToggleVariant(ui.ToggleVariantOutline))
	ui.NewToggle(0, 0, 0, 0, "Lg", rsz.Handle,
		ui.WithToggleSize(ui.ToggleSizeLg), ui.WithToggleVariant(ui.ToggleVariantOutline))

	xui.ItemFill(hint(vt.Handle, "图标 / 纯图标 / 禁用 / 校验失败"))
	ric := xui.HStack(vt.Handle, 40, 12)
	ui.NewToggle(0, 0, 0, 0, "收藏", ric.Handle,
		ui.WithToggleIcon(icons.IconHeart), ui.WithToggleVariant(ui.ToggleVariantOutline))
	ui.NewToggle(0, 0, 0, 0, "", ric.Handle,
		ui.WithToggleIcon(icons.IconBold), ui.WithTogglePressed(true))
	disToggle := ui.NewToggle(0, 0, 0, 0, "Disabled", ric.Handle, ui.WithTogglePressed(true))
	disToggle.Enable(false)
	ui.NewToggle(0, 0, 0, 0, "Invalid", ric.Handle, ui.WithToggleInvalid())

	_, vtg := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "切换按钮组", vtg.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"ToggleGroup: 单选/多选、spacing=0 首尾外角; 组级 variant/size 统一作用于条目", vtg.Handle))

	valueText := func(values []string) string {
		if len(values) == 0 {
			return "无"
		}
		return strings.Join(values, ", ")
	}

	xui.ItemFill(hint(vtg.Handle, "单选 · 无缝(spacing 0)"))
	gs := ui.NewToggleGroup(vtg.Handle)
	gs.AddItem("bold", "Bold")
	gs.AddItem("italic", "Italic")
	gs.AddItem("underline", "Underline")
	gs.SetValue("bold")
	singleLabel := ui.NewLabel(0, 0, 0, 20, "当前: bold", vtg.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(singleLabel)
	gs.OnChange(func(values []string) {
		singleLabel.SetText("当前: " + valueText(values))
		singleLabel.Redraw(false)
	})

	xui.ItemFill(hint(vtg.Handle, "多选 · 纯图标(outline)"))
	gm := ui.NewToggleGroup(vtg.Handle,
		ui.WithToggleGroupMultiple(), ui.WithToggleGroupVariant(ui.ToggleVariantOutline))
	gm.AddItem("bold", "", ui.WithToggleIcon(icons.IconBold))
	gm.AddItem("italic", "", ui.WithToggleIcon(icons.IconItalic))
	gm.AddItem("underline", "", ui.WithToggleIcon(icons.IconUnderline))
	gm.SetValue("bold", "underline")
	multiLabel := ui.NewLabel(0, 0, 0, 20, "已选: bold, underline", vtg.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(multiLabel)
	gm.OnChange(func(values []string) {
		multiLabel.SetText("已选: " + valueText(values))
		multiLabel.Redraw(false)
	})

	xui.ItemFill(hint(vtg.Handle, "有间距(spacing 8)· 组级尺寸 Sm"))
	gg := ui.NewToggleGroup(vtg.Handle,
		ui.WithToggleGroupSpacing(8),
		ui.WithToggleGroupVariant(ui.ToggleVariantOutline),
		ui.WithToggleGroupSize(ui.ToggleSizeSm))
	gg.AddItem("left", "左", ui.WithToggleIcon(icons.IconAlignLeft))
	gg.AddItem("center", "中", ui.WithToggleIcon(icons.IconAlignCenter))
	gg.AddItem("right", "右", ui.WithToggleIcon(icons.IconAlignRight))
	gg.SetValue("center")

	// ---- 主题色: 克隆当前主题替换主色组(Primary/Ring/SidebarPrimary) ----
	_, vtc := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "主题色", vtc.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"克隆当前主题并替换主色组(Primary / Ring / SidebarPrimary), 亮/暗各有一套; 标题栏的调色板按钮也能切换(带取色器)", vtc.Handle))
	xui.ItemFill(hint(vtc.Handle, "点击切换(全窗口实时生效)"))
	for rowStart := 0; rowStart < len(demoThemePresets); rowStart += 3 {
		rowEnd := rowStart + 3
		if rowEnd > len(demoThemePresets) {
			rowEnd = len(demoThemePresets)
		}
		rtc := xui.HStack(vtc.Handle, 36, 8)
		for i := rowStart; i < rowEnd; i++ {
			i := i
			b := ui.NewButton(0, 0, 0, 0, demoThemePresets[i].name, rtc.Handle, ui.WithVariant(ui.ButtonOutline))
			xui.ItemWidth(b, 72)
			b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
				demoThemeColor = i
				applyDemoTheme()
				return 0
			})
		}
	}
	demoThemeColorLabel = ui.NewLabel(0, 0, 0, 20,
		"当前主题色: "+demoThemePresets[demoThemeColor].name, vtc.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(demoThemeColorLabel)
}
