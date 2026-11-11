package main

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

// ===== 页 7: 表单3(字段与分组 / 文件对话框) =====

func buildForm3Page(body *widget.LayoutEle, hWindow int) {
	rField := row(body.Handle, 0, gap)

	_, vf := autoCard(xui.VStack(rField.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "表单字段(Field)", vf.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"标签 / 控件 / 说明 / 错误按 12px 间距成组; 标题与说明由 Field 统一持有", vf.Handle))

	fMail := ui.NewField(0, 0, 0, 0, vf.Handle)
	xui.ItemFill(fMail)
	fMail.Label("邮箱地址")
	xui.ItemFill(ui.NewInput(0, 0, 0, 0, fMail.Handle, ui.WithPlaceholder("m@example.com")))
	fMail.Description("用于接收登录链接与安全通知。")

	fName := ui.NewField(0, 0, 0, 0, vf.Handle, ui.WithFieldInvalid())
	xui.ItemFill(fName)
	fName.Label("用户名")
	name := ui.NewInput(0, 0, 0, 0, fName.Handle, ui.WithInputInvalid())
	xui.ItemFill(name)
	name.SetValue("ab")
	fName.Description("3-20 个字符, 支持字母、数字与下划线。")
	nameErr := ui.NewFieldError(0, 0, 0, 0, fName.Handle,
		"用户名至少 3 个字符", "不能包含空格", "用户名至少 3 个字符")
	xui.ItemFill(nameErr)

	toggleInvalid := ui.NewButton(0, 0, 0, 0, "切换校验状态", vf.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm))
	xui.ItemWidth(toggleInvalid, 120)
	toggleInvalid.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		invalid := !fName.Invalid()
		fName.SetInvalid(invalid)
		name.SetInvalid(invalid)
		if invalid {
			nameErr.SetErrors("用户名至少 3 个字符", "不能包含空格")
		} else {
			nameErr.SetErrors()
		}
		return 0
	})

	fDisplay := ui.NewField(0, 0, 0, 0, vf.Handle)
	xui.ItemFill(fDisplay)
	fDisplay.Label("显示名称")
	content := fDisplay.Content()
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "团队可见", content.Handle).SetMedium())
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "其他成员能看到这个名字。", content.Handle).SetMuted())
	displayName := ui.NewInput(0, 0, 0, 0, fDisplay.Handle)
	xui.ItemFill(displayName)
	displayName.SetValue("twgh")

	_, vf2 := autoCard(xui.VStack(rField.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "分组与横排(FieldSet / FieldGroup)", vf2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"FieldSet 24 / FieldGroup 28; 横排标签吃剩余宽度、控件靠右", vf2.Handle))

	set := ui.NewFieldSet(0, 0, 0, 0, vf2.Handle)
	xui.ItemFill(set)
	ui.NewFieldLegend(0, 0, 0, 0, "通知设置", set.Handle)
	grp := ui.NewFieldGroup(0, 0, 0, 0, set.Handle)
	xui.ItemFill(grp)

	fMailNotify := ui.NewField(0, 0, 0, 0, grp.Handle, ui.WithFieldOrientation(ui.FieldHorizontal))
	xui.ItemFill(fMailNotify)
	fMailNotify.Label("邮件通知")
	ui.NewSwitch(0, 0, fMailNotify.Handle, ui.WithSwitchChecked(true))

	fPushNotify := ui.NewField(0, 0, 0, 0, grp.Handle, ui.WithFieldOrientation(ui.FieldHorizontal))
	xui.ItemFill(fPushNotify)
	fPushNotify.Label("推送通知")
	ui.NewSwitch(0, 0, fPushNotify.Handle)

	ui.NewFieldSeparator(0, 0, 0, 0, grp.Handle, "或")

	fUpdates := ui.NewField(0, 0, 0, 0, grp.Handle, ui.WithFieldOrientation(ui.FieldHorizontal))
	xui.ItemFill(fUpdates)
	cbUpdates := ui.NewCheckbox(0, 0, 0, 0, "", fUpdates.Handle)
	xui.ItemWidth(cbUpdates, 24) // 只剩指示器, 标题/说明放进 FieldContent(同上游 checkbox 字段写法)
	updContent := fUpdates.Content()
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "接收产品更新", updContent.Handle).SetMedium())
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "每月一封, 随时可退订。", updContent.Handle).SetMuted())

	ui.NewFieldLegend(0, 0, 0, 0, "账户", set.Handle, ui.WithFieldLegendVariant(ui.FieldLegendLabel))
	grp2 := ui.NewFieldGroup(0, 0, 0, 0, set.Handle)
	xui.ItemFill(grp2)
	fLang := ui.NewField(0, 0, 0, 0, grp2.Handle, ui.WithFieldOrientation(ui.FieldHorizontal))
	xui.ItemFill(fLang)
	fLang.Label("界面语言")
	lang := ui.NewNativeComboBox(0, 0, 0, 0, fLang.Handle, []string{"简体中文", "English", "日本語"})
	xui.ItemWidth(lang, 160)
	lang.SetSelected(0)

	// ---- 文件对话框(系统原生, 不套主题) ----
	_, vfd := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "文件对话框(FileDialog)", vfd.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"原生打开 / 多选 / 保存 / 选择文件夹; 过滤器按「名称 + 通配符」生成, 取消返回空", vfd.Handle))

	fd := ui.NewFileDialog(hWindow,
		ui.WithFileDialogFilters(
			ui.FileFilter{Name: "图片", Patterns: "*.png;*.jpg;*.jpeg;*.gif;*.bmp"},
			ui.FileFilter{Name: "所有文件", Patterns: "*.*"}),
		ui.WithFileDialogDefaultName("demo.png"),
		ui.WithFileDialogDefaultExt("png"))

	fdState := ui.NewLabel(0, 0, 0, 24, "结果: (尚未选择)", vfd.Handle).SetMuted()
	xui.ItemFill(fdState)
	showResult := func(label string, paths []string) {
		switch len(paths) {
		case 0:
			fdState.SetText(label + ": 已取消")
		case 1:
			fdState.SetText(label + ": " + paths[0])
		default:
			fdState.SetText(fmt.Sprintf("%s: %d 个文件, 首个 %s", label, len(paths), paths[0]))
		}
		fdState.Redraw()
	}

	fdRow := xui.HStack(vfd.Handle, 32, 8)
	btnOpen := ui.NewButton(0, 0, 0, 0, "打开文件", fdRow.Handle, ui.WithVariant(ui.ButtonOutline))
	btnOpen.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if path := fd.Open(); path != "" {
			showResult("打开", []string{path})
		} else {
			showResult("打开", nil)
		}
		return 0
	})
	btnMulti := ui.NewButton(0, 0, 0, 0, "打开多个", fdRow.Handle, ui.WithVariant(ui.ButtonOutline))
	btnMulti.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		showResult("多选", fd.OpenMulti())
		return 0
	})
	btnSave := ui.NewButton(0, 0, 0, 0, "保存文件", fdRow.Handle, ui.WithVariant(ui.ButtonOutline))
	btnSave.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if path := fd.Save(); path != "" {
			showResult("保存", []string{path})
		} else {
			showResult("保存", nil)
		}
		return 0
	})
	btnDir := ui.NewButton(0, 0, 0, 0, "选择文件夹", fdRow.Handle, ui.WithVariant(ui.ButtonOutline))
	btnDir.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if dir := fd.PickDir(); dir != "" {
			showResult("目录", []string{dir})
		} else {
			showResult("目录", nil)
		}
		return 0
	})
}
