package main

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 6: 表单2(选择控件 / 下拉框 / 独立 Select) =====

func buildForm2Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)
	colL := xui.VStack(r.Handle, 1, gap)
	colR := xui.VStack(r.Handle, 1, gap)

	// ---- 左列: 选择控件 ----
	_, v2 := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "选择控件", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "复选框 / 单选框 / 开关, 尺寸与状态语义逐条对齐 shadcn", v2.Handle))

	xui.ItemFill(hint(v2.Handle, "复选框"))
	c1 := ui.NewCheckbox(0, 0, 0, 0, "接收邮件通知", v2.Handle)
	c1.SetChecked(true)
	xui.ItemFill(c1)
	xui.ItemFill(ui.NewCheckbox(0, 0, 0, 0, "公开展示个人主页", v2.Handle))
	c3 := ui.NewCheckbox(0, 0, 0, 0, "不可勾选", v2.Handle)
	xui.ItemFill(c3)
	c3.Enable(false)

	xui.ItemFill(ui.NewSeparator(0, 0, 0, v2.Handle))

	xui.ItemFill(hint(v2.Handle, "单选框"))
	ra := ui.NewRadio(0, 0, 0, 0, "仅错误消息", v2.Handle)
	xui.ItemFill(ra)
	xui.ItemFill(ui.NewRadio(0, 0, 0, 0, "所有消息", v2.Handle))
	ra.SetCheck(true)
	rd := ui.NewRadio(0, 0, 0, 0, "不可选", v2.Handle)
	xui.ItemFill(rd)
	rd.Enable(false)

	xui.ItemFill(ui.NewSeparator(0, 0, 0, v2.Handle))

	xui.ItemFill(hint(v2.Handle, "开关(default / sm, 禁用)"))
	rs := xui.HStack(v2.Handle, 24, 12)
	ui.NewSwitch(0, 0, rs.Handle, ui.WithSwitchChecked(true))
	ui.NewSwitch(0, 0, rs.Handle)
	ui.NewSwitch(0, 0, rs.Handle, ui.WithSwitchSize(ui.SwitchSm), ui.WithSwitchChecked(true))
	ui.NewSwitch(0, 0, rs.Handle, ui.WithSwitchSize(ui.SwitchSm), ui.WithSwitchDisabled())

	xui.ItemFill(hint(v2.Handle, "校验失败(aria-invalid; 红边 + 常驻红环) / 空格键切换"))
	rsInv := xui.HStack(v2.Handle, 24, 12)
	ui.NewSwitch(0, 0, rsInv.Handle, ui.WithSwitchInvalid())
	ui.NewSwitch(0, 0, rsInv.Handle, ui.WithSwitchInvalid(), ui.WithSwitchChecked(true))
	ui.NewSwitch(0, 0, rsInv.Handle, ui.WithSwitchChecked(true))

	xui.ItemFill(ui.NewSeparator(0, 0, 0, v2.Handle))

	const (
		demoHeartFull = `<svg viewBox="0 0 24 24"><path fill="currentColor" d="M2 9.5A5.5 5.5 0 0 1 11.591 5.824A.56 .56 0 0 0 12.409 5.824A5.49 5.49 0 0 1 22 9.5C22 11.79 20.5 13.5 19 15L13.508 20.313A2 2 0 0 1 10.508 20.332L5 15C3.5 13.5 2 11.8 2 9.5 Z"/></svg>`
		demoHeartHalf = `<svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 20.8 L10.508 20.332 L5 15 C3.5 13.5 2 11.8 2 9.5 A5.5 5.5 0 0 1 11.591 5.824 L12 5.82 Z"/></svg>`
	)

	xui.ItemFill(hint(v2.Handle, "评分(半星 / 文案 / 再点当前星清零; 点击与 ←/→/Home/End)"))
	rt := ui.NewRating(0, 0, 0, 0, v2.Handle, ui.WithRatingHalf(), ui.WithRatingValue(3.5),
		ui.WithRatingLabels([]string{"很差", "差", "一般", "好", "很好"}))
	xui.ItemFill(rt)
	rtState := ui.NewLabel(0, 0, 0, 0, "当前评分: 3.5 / 5", v2.Handle).SetMuted()
	xui.ItemFill(rtState)
	rtVal, rtPreview, rtFocus := 3.5, "", ""
	refreshRt := func() {
		rtState.SetText(fmt.Sprintf("当前评分: %g / 5%s%s", rtVal, rtPreview, rtFocus)).Redraw()
	}
	rt.OnChange(func(v float64) { rtVal = v; refreshRt() })
	rt.OnHoverChange(func(v float64) {
		if v < 0 {
			rtPreview = ""
		} else {
			rtPreview = fmt.Sprintf(" · 预览: %g", v)
		}
		refreshRt()
	})
	rt.OnFocus(func() { rtFocus = " · 焦点进入"; refreshRt() })
	rt.OnBlur(func() { rtFocus = " · 焦点离开"; refreshRt() })
	rt.OnKeyDown(func(vk uintptr) { rtFocus = fmt.Sprintf(" · 键 0x%X", vk); refreshRt() })

	xui.ItemFill(hint(v2.Handle, "只读(4 星) / 禁用(3 星) / 不允许清除"))
	rtRow := xui.HStack(v2.Handle, 24, 12)
	ui.NewRating(0, 0, 0, 0, rtRow.Handle, ui.WithRatingReadonly(), ui.WithRatingValue(4))
	ui.NewRating(0, 0, 0, 0, rtRow.Handle, ui.WithRatingDisabled(), ui.WithRatingValue(3))
	ui.NewRating(0, 0, 0, 0, rtRow.Handle, ui.WithRatingAllowClear(false), ui.WithRatingValue(3))

	xui.ItemFill(hint(v2.Handle, "尺寸 small / medium / large"))
	rtSizes := xui.HStack(v2.Handle, 34, 12)
	ui.NewRating(0, 0, 0, 0, rtSizes.Handle, ui.WithRatingSize(ui.RatingSmall), ui.WithRatingValue(2))
	ui.NewRating(0, 0, 0, 0, rtSizes.Handle, ui.WithRatingValue(3))
	ui.NewRating(0, 0, 0, 0, rtSizes.Handle, ui.WithRatingSize(ui.RatingLarge), ui.WithRatingValue(4))

	xui.ItemFill(hint(v2.Handle, "自定义图标(心形三态) / 关闭键盘(方向键交回焦点切换)"))
	rtIcons := xui.HStack(v2.Handle, 30, 12)
	ui.NewRating(0, 0, 0, 0, rtIcons.Handle, ui.WithRatingHalf(), ui.WithRatingValue(2.5),
		ui.WithRatingIcons(icons.IconHeart, demoHeartFull, demoHeartHalf))
	ui.NewRating(0, 0, 0, 0, rtIcons.Handle, ui.WithRatingKeyboard(false), ui.WithRatingValue(3))

	xui.ItemFill(hint(v2.Handle, "焦点 / 按键回调 + Blur()(聚焦后失焦按钮依次触发两个事件)"))
	rtFocusRow := xui.HStack(v2.Handle, 28, 8)
	focusBtn := ui.NewButton(0, 0, 0, 0, "聚焦", rtFocusRow.Handle,
		ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline))
	focusBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		rt.SetFocus()
		return 0
	})
	blurBtn := ui.NewButton(0, 0, 0, 0, "聚焦后失焦", rtFocusRow.Handle,
		ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline))
	blurBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		rt.SetFocus()
		rt.Blur()
		return 0
	})

	// ---- 右列 1: 下拉框 ----
	_, v4 := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "下拉框", v4.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "原生 ComboBox 行为 + 自绘弹层(item-aligned 定位, 盖住触发区)", v4.Handle))

	xui.ItemFill(hint(v4.Handle, "默认尺寸"))
	combo := ui.NewNativeComboBox(0, 0, 0, 0, v4.Handle,
		[]string{"开发环境", "测试环境", "预发环境", "灰度环境", "生产环境", "本地调试"},
		ui.WithNativeComboBoxPlaceholder("选择环境"))
	xui.ItemFill(combo)
	combo.SetSelected(1)
	tip.Bind(combo.Handle, "切换运行时环境")

	xui.ItemFill(hint(v4.Handle, "sm 尺寸"))
	xui.ItemFill(ui.NewNativeComboBox(0, 0, 0, 0, v4.Handle,
		[]string{"开发环境", "测试环境", "生产环境"},
		ui.WithNativeComboBoxSize(ui.NativeComboBoxSizeSm), ui.WithNativeComboBoxPlaceholder("sm 尺寸")))

	xui.ItemFill(hint(v4.Handle, "NativeSelect(shadcn 原生 <select> 对应物, value 字符串语义)"))
	ns := ui.NewNativeSelect(0, 0, 0, 0, v4.Handle,
		[]string{"开发环境", "测试环境", "生产环境"},
		ui.WithNativeSelectPlaceholder("选择环境"), ui.WithNativeSelectValue("测试环境"))
	xui.ItemFill(ns)
	ns.OnValueChange(func(v string) { println("NativeSelect:", v) })

	// ---- 右列 2: 独立 Select(自绘弹层) ----
	_, v5 := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "独立 Select", v5.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"radix Select(item-aligned 定位): 富条目——图标 / 禁用 / 分组 / 分隔线; ↑↓ 移动高亮、Enter 选中、ESC 关闭", v5.Handle))

	xui.ItemFill(hint(v5.Handle, "分组 + 图标 + 禁用项"))
	sel := ui.NewSelect(0, 0, 0, 0, v5.Handle,
		ui.WithSelectPlaceholder("选择主题"),
		ui.WithSelectGroups([]ui.SelectGroup{
			{Label: "外观", Items: []ui.SelectItem{
				{Value: "light", Text: "浅色", Icon: icons.IconSun},
				{Value: "dark", Text: "深色", Icon: icons.IconMoon},
				{Value: "system", Text: "跟随系统", Icon: icons.IconMonitor},
			}},
			{Label: "更多", Items: []ui.SelectItem{
				{Value: "custom", Text: "自定义(不可选)", Icon: icons.IconPalette, Disabled: true},
			}},
		}))
	xui.ItemFill(sel)
	sel.SetValue("dark")
	sel.OnChange(func(item ui.SelectItem) { println("Select:", item.Value) })

	xui.ItemFill(hint(v5.Handle, "sm 尺寸 + 平铺条目"))
	xui.ItemFill(ui.NewSelect(0, 0, 0, 0, v5.Handle,
		ui.WithSelectSize(ui.SelectSizeSm), ui.WithSelectPlaceholder("选择框架"),
		ui.WithSelectItems([]ui.SelectItem{
			{Value: "react", Text: "React"},
			{Value: "vue", Text: "Vue"},
			{Value: "svelte", Text: "Svelte"},
		})))

	// ---- 右列 3: 自动补全(Combobox) ----
	_, vc := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "自动补全", vc.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"shadcn 文档的 Combobox: 触发按钮 + 弹层搜索框 + 建议列表; 输入即过滤(不区分大小写), ↑↓ 移动、Enter 选中、ESC 关闭", vc.Handle))

	xui.ItemFill(hint(vc.Handle, "单选(分组 + 图标 + 禁用项)"))
	cb := ui.NewCombobox(0, 0, 0, 0, vc.Handle,
		ui.WithComboboxPlaceholder("Select framework..."),
		ui.WithComboboxSearchPlaceholder("Search framework..."),
		ui.WithComboboxGroups([]ui.SelectGroup{
			{Label: "前端", Items: []ui.SelectItem{
				{Value: "next", Text: "Next.js", Icon: icons.IconGlobe},
				{Value: "svelte", Text: "SvelteKit", Icon: icons.IconGlobe},
				{Value: "nuxt", Text: "Nuxt.js", Icon: icons.IconGlobe},
				{Value: "remix", Text: "Remix(不可选)", Icon: icons.IconGlobe, Disabled: true},
			}},
			{Label: "后端", Items: []ui.SelectItem{
				{Value: "go", Text: "Golang", Icon: icons.IconServer},
				{Value: "rust", Text: "Rust", Icon: icons.IconServer},
			}},
		}))
	xui.ItemFill(cb)
	cb.SetValue("next")
	cb.OnChange(func(items []ui.SelectItem) {
		if len(items) > 0 {
			println("Combobox:", items[0].Value)
		}
	})

	xui.ItemFill(hint(vc.Handle, "多选(点击切换, 弹层保持打开; 触发区显示 首个值 +N)"))
	cbMulti := ui.NewCombobox(0, 0, 0, 0, vc.Handle,
		ui.WithComboboxMultiple(), ui.WithComboboxPlaceholder("Select frameworks..."),
		ui.WithComboboxItems([]ui.SelectItem{
			{Value: "ts", Text: "TypeScript"}, {Value: "go", Text: "Golang"},
			{Value: "rs", Text: "Rust"}, {Value: "py", Text: "Python"},
		}))
	xui.ItemFill(cbMulti)
	cbMulti.SetValues("ts")

	xui.ItemFill(hint(vc.Handle, "多选 chips(每个可点 × 移除, 退格删末尾, 放不下折叠 +N)"))
	cbChips := ui.NewCombobox(0, 0, 0, 0, vc.Handle,
		ui.WithComboboxMultiple(), ui.WithComboboxChips(), ui.WithComboboxPlaceholder("Select frameworks..."),
		ui.WithComboboxItems([]ui.SelectItem{
			{Value: "ts", Text: "TypeScript"}, {Value: "go", Text: "Golang"},
			{Value: "rs", Text: "Rust"}, {Value: "py", Text: "Python"}, {Value: "c", Text: "C"}, {Value: "html", Text: "HTML"},
		}))
	xui.ItemFill(cbChips)
	cbChips.SetValues("ts", "go", "rs", "py")

	xui.ItemFill(hint(vc.Handle, "清除钮(有值时替代箭头, 点击清空) + autoHighlight"))
	xui.ItemFill(ui.NewCombobox(0, 0, 0, 0, vc.Handle,
		ui.WithComboboxClear(), ui.WithComboboxAutoHighlight(true),
		ui.WithComboboxPlaceholder("Select framework..."),
		ui.WithComboboxItems([]ui.SelectItem{
			{Value: "next", Text: "Next.js"}, {Value: "svelte", Text: "SvelteKit"},
			{Value: "nuxt", Text: "Nuxt.js"},
		})).SetValue("svelte"))

	xui.ItemFill(hint(vc.Handle, "autoHighlight 关闭(打字不自动高亮, ↓ 或悬停才点亮)"))
	xui.ItemFill(ui.NewCombobox(0, 0, 0, 0, vc.Handle,
		ui.WithComboboxAutoHighlight(false), ui.WithComboboxPlaceholder("Select framework..."),
		ui.WithComboboxItems([]ui.SelectItem{
			{Value: "next", Text: "Next.js"}, {Value: "svelte", Text: "SvelteKit"},
			{Value: "nuxt", Text: "Nuxt.js"},
		})))

	xui.ItemFill(hint(vc.Handle, "校验失败 / 禁用"))
	bad := ui.NewCombobox(0, 0, 0, 0, vc.Handle,
		ui.WithComboboxInvalid(), ui.WithComboboxPlaceholder("必选项未选择"))
	xui.ItemFill(bad)
	xui.ItemFill(ui.NewCombobox(0, 0, 0, 0, vc.Handle,
		ui.WithComboboxDisabled(), ui.WithComboboxPlaceholder("禁用"),
		ui.WithComboboxItems([]ui.SelectItem{{Value: "a", Text: "A"}})))
}
