// xui 组件演示: light/dark 一键换肤 + P1 全部组件。
package main

import (
	"strconv"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

func main() {
	app.InitOrExit()
	a := app.New(true)
	a.EnableAutoDPI(true).EnableDPI(true)

	w := window.New(0, 0, 800, 860, "xui · shadcn 风格自绘组件", 0, xcc.Window_Style_Default|xcc.Window_Style_Drag_Window)
	w.SetBorderSize(1, 32, 1, 1)
	w.SetCaptionMargin(4, 2, 0, 0)
	w.SetFont(xui.FontHandle(xui.Current().FontDefault))
	xui.RegisterWindow(w.Handle)
	xui.BindWindowBackground(w.Handle) // 窗口底色随主题切换

	// 标题文字颜色随主题(默认白色是为蓝色标题栏设计的, 白底上不可见)
	applyTitle := func(t *xui.Theme) { w.SetTitleColor(uint32(t.Foreground)) }
	applyTitle(xui.Current())
	xui.OnThemeChange(applyTitle)

	dark := false
	toggle := ui.NewButton(340, 6, 100, 30, "Dark", w.Handle, ui.WithVariant(ui.ButtonOutline))
	toggle.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		dark = !dark
		if dark {
			xui.SetTheme(xui.Dark())
			toggle.SetText("Light")
		} else {
			xui.SetTheme(xui.Light())
			toggle.SetText("Dark")
		}
		return 0
	})

	buildPage(w.Handle)

	w.Show(true)
	a.Run()
	a.Exit()
}

// buildPage 布局演示页: 由于主题切换会整窗重绘, 页面内容静态创建一次即可。
func buildPage(hParent int) {
	// ===== 左卡: 表单类 =====
	card1 := ui.NewCard(24, 48, 340, 470, hParent)
	x, y := card1.ContentLeft(), card1.ContentTop()
	cw := card1.ContentWidth()

	ui.NewCardTitle(x, y, cw, "Account", card1.Handle)
	ui.NewLabel(x, y+34, cw, 0, "修改你的账户信息, 保存后立即生效。", card1.Handle).SetMuted()
	y += 66

	ui.NewLabel(x, y, cw, 0, "邮箱地址", card1.Handle)
	y += 30
	email := ui.NewInput(x, y, cw, 0, card1.Handle, ui.WithPlaceholder("m@example.com"))
	y += 48

	ui.NewLabel(x, y, cw, 0, "个人简介", card1.Handle)
	y += 30
	ui.NewInput(x, y, cw, 72, card1.Handle,
		ui.WithPlaceholder("介绍一下自己..."), ui.WithMultiline())
	y += 94

	// 复选框两行
	c1 := ui.NewCheckbox(x, y, cw, 0, "接收邮件通知", card1.Handle)
	c1.SetChecked(true)
	y += 32
	ui.NewCheckbox(x, y, cw, 0, "公开展示个人主页", card1.Handle)
	y += 44

	saveBtn := ui.NewButton(x, y, 96, 0, "保存", card1.Handle)
	ui.NewButton(x+104, y, 96, 0, "取消", card1.Handle, ui.WithVariant(ui.ButtonOutline))

	_ = email

	// ===== 右卡: 展示类 =====
	card2 := ui.NewCard(392, 48, 380, 470, hParent)
	x, y = card2.ContentLeft(), card2.ContentTop()
	cw = card2.ContentWidth()

	ui.NewCardTitle(x, y, cw, "Components", card2.Handle)
	y += 36

	// 按钮全变体
	ui.NewButton(x, y, 104, 0, "Primary", card2.Handle)
	ui.NewButton(x+112, y, 112, 0, "Secondary", card2.Handle, ui.WithVariant(ui.ButtonSecondary))
	ui.NewButton(x+232, y, 104, 0, "Outline", card2.Handle, ui.WithVariant(ui.ButtonOutline))
	y += 48
	ui.NewButton(x, y, 104, 0, "Ghost", card2.Handle, ui.WithVariant(ui.ButtonGhost))
	ui.NewButton(x+112, y, 112, 0, "Destructive", card2.Handle, ui.WithVariant(ui.ButtonDestructive))
	ui.NewButton(x+232, y, 104, 0, "Disabled", card2.Handle).Enable(false)
	y += 60

	ui.NewSeparator(x, y, cw, card2.Handle)
	y += 28

	// 徽章
	ui.NewBadge(x, y, 64, 0, "Default", card2.Handle)
	ui.NewBadge(x+72, y, 78, 0, "Secondary", card2.Handle, ui.WithBadgeVariant(ui.BadgeSecondary))
	ui.NewBadge(x+158, y, 72, 0, "Outline", card2.Handle, ui.WithBadgeVariant(ui.BadgeOutline))
	ui.NewBadge(x+238, y, 88, 0, "Destructive", card2.Handle, ui.WithBadgeVariant(ui.BadgeDestructive))
	y += 52

	// 单选组
	ui.NewLabel(x, y, cw, 0, "通知偏好", card2.Handle).SetMuted()
	y += 26
	r1 := ui.NewRadio(x, y, cw, 0, "仅错误消息", card2.Handle)
	r1.SetCheck(true)
	y += 30
	ui.NewRadio(x, y, cw, 0, "所有消息", card2.Handle)
	y += 30
	ui.NewRadio(x, y, cw, 0, "不接收", card2.Handle).Enable(false)

	// ===== 下卡: P2 组件 =====
	card3 := ui.NewCard(24, 536, 728, 290, hParent)
	x, y = card3.ContentLeft(), card3.ContentTop()
	cw = card3.ContentWidth()

	// 开关
	ui.NewLabel(x, y, 120, 0, "开关", card3.Handle).SetMuted()
	ui.NewSwitch(x, y+22, card3.Handle, ui.WithSwitchChecked(true))
	ui.NewSwitch(x+56, y+22, card3.Handle)
	ui.NewSwitch(x+112, y+24, card3.Handle, ui.WithSwitchSize(ui.SwitchSm))
	ui.NewSwitch(x+168, y+24, card3.Handle, ui.WithSwitchSize(ui.SwitchSm)).Enable(false)

	// 滑动条
	ui.NewLabel(x+260, y, 120, 0, "滑动条", card3.Handle).SetMuted()
	sliderValue := ui.NewLabel(x+420, y, 60, 22, "40", card3.Handle).SetMuted()
	sl := ui.NewSlider(x+260, y+25, 160, 0, card3.Handle)
	sl.SetValue(40)
	sl.OnChange(func(v float64) { sliderValue.SetText(strconv.Itoa(int(v + 0.5))).Redraw() })
	y += 68

	// 进度条 / 标签页 / 下拉框
	ui.NewLabel(x, y, 120, 0, "进度", card3.Handle).SetMuted()
	ui.NewLabel(x+260, y, 120, 0, "标签页", card3.Handle).SetMuted()
	ui.NewLabel(x+460, y, 120, 0, "下拉框", card3.Handle).SetMuted()
	ui.NewProgress(x, y+22, 200, 0, card3.Handle).SetValue(64)
	ui.NewProgress(x, y+40, 200, 0, card3.Handle).SetValue(100)

	tabs := ui.NewTabs(x+260, y+24, 180, card3.Handle, []string{"账户", "密码", "通知"})
	panel := ui.NewLabel(x+260, y+72, cw-260, 22, "账户设置面板", card3.Handle).SetMuted()
	tabs.OnChange(func(i int32) {
		panel.SetText([]string{"账户设置面板", "密码设置面板", "通知设置面板"}[i]).Redraw()
	})

	combo := ui.NewComboBox(x+460, y+24, 180, 0, card3.Handle,
		[]string{"开发环境", "测试环境", "预发环境", "灰度环境", "生产环境", "本地调试"},
		ui.WithComboPlaceholder("选择环境"))
	combo.SetSelected(1)

	// line 变体标签页
	ui.NewTabs(x+260, y+112, 180, card3.Handle, []string{"概览", "详情"},
		ui.WithTabsVariant(ui.TabsLine), ui.WithTabsActive(1))

	// 提示气泡: 绑到"保存"按钮与下拉框上
	tip := ui.NewTooltip()
	tip.Bind(saveBtn.Handle, "保存当前修改(Ctrl+S)")
	tip.Bind(combo.Handle, "切换运行时环境")
}
