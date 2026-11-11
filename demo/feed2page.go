package main

import (
	"fmt"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 17: 反馈2(提示条 / 通知 / 进度 / 空状态) =====

func buildFeed2Page(body *widget.LayoutEle, hWindow int) {
	// ---- 第一行: 提示条 ----
	rAlert := row(body.Handle, 0, gap)

	_, va := autoCard(xui.VStack(rAlert.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "提示条", va.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"图标 + 标题 + 描述 + 右上动作槽; 标题/描述随宽度折行, 高度按内容自撑", va.Handle))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, va.Handle,
		ui.WithAlertIcon(icons.IconCircleCheck),
		ui.WithAlertTitle("Success! Your changes have been saved"),
		ui.WithAlertDescription("This is an alert with icon, title and description.")))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, va.Handle,
		ui.WithAlertIcon(icons.IconInfo),
		ui.WithAlertTitle("This Alert has a title and an icon. No description.")))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, va.Handle,
		ui.WithAlertDescription("This one has a description only. No title. No icon.")))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, va.Handle,
		ui.WithAlertIcon(icons.IconCircleAlert),
		ui.WithAlertTitle("This is a very long alert title that demonstrates how the component handles extended text content and potentially wraps across multiple lines"),
		ui.WithAlertDescription("This is an equally long description that contains detailed information about the alert. It wraps at the alert width, and the alert grows with it instead of clipping.")))
	xui.ItemFill(desc(va.Handle, "标题与描述用当前宽度量高: 窗口拉宽/收窄时会重新折行。"))

	_, vb := autoCard(xui.VStack(rAlert.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "破坏性 / 动作", vb.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"destructive: 标题与图标用破坏性色, 描述 90%; 动作贴右上", vb.Handle))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, vb.Handle,
		ui.WithAlertVariant(ui.AlertDestructive),
		ui.WithAlertIcon(icons.IconCircleAlert),
		ui.WithAlertTitle("Unable to process your payment."),
		ui.WithAlertDescription("Please verify your billing information and try again.")))
	withAction := ui.NewAlert(0, 0, 0, 0, vb.Handle,
		ui.WithAlertIcon(icons.IconCircleAlert),
		ui.WithAlertTitle("The selected emails have been marked as spam."),
		ui.WithAlertDescription("This is a very long alert title that demonstrates how the component handles extended text content."))
	xui.ItemFill(withAction)
	ui.NewButton(0, 0, 0, 0, "Undo", withAction.Action().Handle,
		ui.WithSize(ui.ButtonSizeXs))
	withBadge := ui.NewAlert(0, 0, 0, 0, vb.Handle,
		ui.WithAlertIcon(icons.IconInfo),
		ui.WithAlertTitle("A new version is available."),
		ui.WithAlertDescription("Restart the app to apply the update."))
	xui.ItemFill(withBadge)
	badge := ui.NewBadge(0, 0, 0, 0, "New", withBadge.Action().Handle,
		ui.WithBadgeVariant(ui.BadgeSecondary))
	xui.ItemWidth(badge, 44)
	xui.ItemFill(desc(vb.Handle, "动作槽用 Action() 取; 可放 Button / Badge, 顶端对齐、内容自量宽。"))

	// ---- 第二行: 通知 + 进度 ----
	r2 := row(body.Handle, 0, gap)

	// 通知: 右下角堆叠, 4s 自动关闭, 悬停暂停
	_, v3 := autoCard(xui.VStack(r2.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "通知", v3.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"窗口内覆盖元素, 右下角堆叠; 4s 自动关闭, 悬停全部暂停; 关闭按钮默认隐藏; richColors 按类型上色", v3.Handle))
	toaster := ui.NewToaster(hWindow)
	rn1 := xui.HStack(v3.Handle, 36, 8)
	for _, it := range []struct {
		text    string
		variant ui.ToastVariant
	}{{"成功", ui.ToastSuccess}, {"错误", ui.ToastError}, {"警告", ui.ToastWarning}, {"信息", ui.ToastInfo}} {
		it := it
		b := ui.NewButton(0, 0, 0, 0, it.text, rn1.Handle, ui.WithVariant(ui.ButtonOutline))
		xui.ItemWidth(b, 78)
		b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			toaster.Show(ui.ToastOptions{
				Title:       it.text + "通知",
				Description: "这条通知会在 4 秒后自动关闭, 鼠标悬停可暂停倒计时。",
				Variant:     it.variant,
			})
			return 0
		})
	}
	rn2 := xui.HStack(v3.Handle, 36, 8)
	bAction := ui.NewButton(0, 0, 0, 0, "带动作", rn2.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(bAction, 78)
	bAction.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		toaster.Show(ui.ToastOptions{
			Title:       "已删除项目",
			Description: "「xui-demo」已移入回收站。",
			Action:      "撤销",
			CloseButton: true,
			OnAction:    func() { fmt.Println("toast: 撤销删除") },
			OnClose:     func() { fmt.Println("toast: 关闭") },
		})
		return 0
	})
	bLong := ui.NewButton(0, 0, 0, 0, "长文本", rn2.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(bLong, 78)
	bLong.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		toaster.Show(ui.ToastOptions{
			Title: "同步完成",
			Description: "本次同步共处理 1,284 个文件, 其中 37 个发生冲突已自动合并, " +
				"3 个需要手动处理。",
			Variant:     ui.ToastInfo,
			Duration:    -1,
			CloseButton: true,
		})
		return 0
	})
	bRich := ui.NewButton(0, 0, 0, 0, "richColors", rn2.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(bRich, 96)
	bRich.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		toaster.Show(ui.ToastOptions{
			Title:       "富色通知",
			Description: "richColors: 按类型给卡面/描边/文字上色。",
			Variant:     ui.ToastSuccess,
			RichColors:  true,
		})
		return 0
	})
	// Promise: 进行中显示转圈, 结束后同一条变成成功/失败态(sonner 的 toast.promise)
	rn3 := xui.HStack(v3.Handle, 36, 8)
	bPromise := ui.NewButton(0, 0, 0, 0, "Promise", rn3.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(bPromise, 96)
	bPromise.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		toaster.Promise(func() (string, error) {
			time.Sleep(1500 * time.Millisecond)
			return "已同步 128 个文件", nil
		}, ui.ToastPromise{
			Loading: ui.ToastOptions{Title: "正在同步…", Description: "完成后会自动切换成结果"},
			Success: func(result string) ui.ToastOptions {
				return ui.ToastOptions{Title: "同步完成", Description: result, Variant: ui.ToastSuccess}
			},
		})
		return 0
	})
	bPromiseFail := ui.NewButton(0, 0, 0, 0, "Promise 失败", rn3.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(bPromiseFail, 106)
	bPromiseFail.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		toaster.Promise(func() (string, error) {
			time.Sleep(1200 * time.Millisecond)
			return "", fmt.Errorf("网络超时, 请重试")
		}, ui.ToastPromise{
			Loading: ui.ToastOptions{Title: "正在保存…"},
		})
		return 0
	})
	bLoading := ui.NewButton(0, 0, 0, 0, "常驻加载", rn3.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(bLoading, 96)
	bLoading.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		toaster.Show(ui.ToastOptions{
			Title:       "正在导出…",
			Description: "Loading 默认常驻, 需要自己关闭或 Update。",
			Variant:     ui.ToastLoading,
			Duration:    -1,
			CloseButton: true,
			OnDismiss:   func() { fmt.Println("toast: 用户关掉了加载通知") },
		})
		return 0
	})
	xui.ItemFill(desc(v3.Handle,
		"ToastOptions.Duration 传负数即常驻; Action 点击后默认关闭; OnAutoClose / OnDismiss 区分超时与主动关闭; WithToastCloseButton() 让整组都带关闭钮。"))

	_, v4 := autoCard(xui.VStack(r2.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "进度", v4.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"h-1 圆角轨道 + 指示器(宽度按百分比); 支持 max / 不确定态 / 自定义指示器颜色", v4.Handle))
	for _, it := range []struct{ val, max float64 }{{24, 100}, {64, 100}, {150, 200}} {
		pg := ui.NewProgress(0, 0, 0, 0, v4.Handle, ui.WithProgressMax(it.max))
		xui.ItemFill(pg)
		pg.SetValue(it.val)
	}

	xui.ItemFill(hint(v4.Handle, "带标题与数值(ProgressField, 对应文档 Labels 示例)"))
	pf := ui.NewProgressField(0, 0, 0, 0, v4.Handle,
		ui.WithProgressFieldTitle("上传进度"), ui.WithProgressFieldValue(66))
	xui.ItemFill(pf)

	xui.ItemFill(hint(v4.Handle, "不确定态(短条往返) / 指示器跟随主题色"))
	pInd := ui.NewProgress(0, 0, 0, 0, v4.Handle, ui.WithProgressIndeterminate())
	xui.ItemFill(pInd)
	pColor := ui.NewProgress(0, 0, 0, 0, v4.Handle,
		ui.WithProgressIndicatorColorFunc(func(t *xui.Theme) xui.Color { return t.Chart2 }))
	xui.ItemFill(pColor)
	pColor.SetValue(70)

	// ---- 屏幕通知 + 系统托盘 ----
	rScreen := row(body.Handle, 0, gap)

	demoScreen = ui.NewScreenToaster(hWindow)
	_, vs := autoCard(xui.VStack(rScreen.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "屏幕通知", vs.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"独立置顶小窗, 固定在屏幕右下角; 应用最小化/隐藏也能看到; 不占任务栏、不抢焦点", vs.Handle))
	rs1 := xui.HStack(vs.Handle, 36, 8)
	for _, it := range []struct {
		text    string
		variant ui.ToastVariant
	}{{"成功", ui.ToastSuccess}, {"错误", ui.ToastError}, {"警告", ui.ToastWarning}, {"信息", ui.ToastInfo}} {
		it := it
		b := ui.NewButton(0, 0, 0, 0, it.text, rs1.Handle, ui.WithVariant(ui.ButtonOutline))
		xui.ItemWidth(b, 78)
		b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			demoScreen.Show(ui.ToastOptions{
				Title:       it.text + "通知",
				Description: "这条通知贴在屏幕右下角, 4 秒后自动关闭。",
				Variant:     it.variant,
			})
			return 0
		})
	}
	rs2 := xui.HStack(vs.Handle, 36, 8)
	bScreenAction := ui.NewButton(0, 0, 0, 0, "带动作", rs2.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(bScreenAction, 78)
	bScreenAction.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		demoScreen.Show(ui.ToastOptions{
			Title:       "已删除项目",
			Description: "「xui-demo」已移入回收站。",
			Action:      "撤销",
			CloseButton: true,
			OnAction:    func() { fmt.Println("screen toast: 撤销删除") },
		})
		return 0
	})
	bScreenLong := ui.NewButton(0, 0, 0, 0, "常驻长文本", rs2.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(bScreenLong, 96)
	bScreenLong.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		demoScreen.Show(ui.ToastOptions{
			Title: "同步完成",
			Description: "本次同步共处理 1,284 个文件, 其中 37 个发生冲突已自动合并, " +
				"3 个需要手动处理。",
			Variant:     ui.ToastInfo,
			Duration:    -1,
			CloseButton: true,
		})
		return 0
	})
	xui.ItemFill(desc(vs.Handle,
		"内容与窗口内 Toast 共用一套卡片绘制; 悬停暂停全部倒计时; Duration < 0 常驻。"))

	_, vt := autoCard(xui.VStack(rScreen.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "系统托盘", vt.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"托盘图标 + 右键 themed 菜单; 左键默认显示主窗口; Explorer 重启后自动重新添加", vt.Handle))
	demoTrayButton = ui.NewButton(0, 0, 0, 0, "启用托盘", vt.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(demoTrayButton, 104)
	demoTrayButton.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		toggleDemoTray()
		return 0
	})
	xui.ItemFill(hint(vt.Handle,
		"启用后右键系统托盘图标: 显示主窗口 / 发送屏幕通知 / 退出。关闭 demo 时托盘随之移除。"))

	// ---- 屏幕弹窗(常驻面板, 不是通知) ----
	demoPopupLines = 0
	demoPopup = ui.NewScreenPopup(hWindow,
		ui.WithScreenPopupWidth(340),
		ui.WithScreenPopupDuration(10*time.Second),
		ui.WithScreenPopupHeader("传输队列"),
		ui.WithScreenPopupOnOpenChange(func(open bool) {
			if open {
				syncDemoPopupStatus("已显示: 10 秒后自动关闭, 焦点仍在主窗口")
			} else {
				syncDemoPopupStatus("已关闭: 右上角 X / 超时 / 内容按钮都可收起")
			}
		}),
		ui.WithScreenPopupContent(func(c int) {
			xui.ItemFill(ui.NewLabel(0, 0, 0, 20,
				"任意 xui 内容; 本卡设了 10 秒自动关闭。", c).SetMuted().SetFontSize(13))
			pg := ui.NewProgress(0, 0, 0, 0, c, ui.WithProgressMax(100))
			xui.ItemFill(pg)
			pg.SetValue(66)
			rb := xui.HStack(c, 32, 8)
			add := ui.NewButton(0, 0, 0, 0, "加一行", rb.Handle,
				ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm))
			xui.ItemWidth(add, 84)
			add.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
				demoPopupLines++
				line := ui.NewLabel(0, 0, 0, 20, fmt.Sprintf("动态追加第 %d 行", demoPopupLines), c).
					SetMuted().SetFontSize(13)
				xui.ItemFill(line) // 不设布局项宽度时标签宽 0: 高度算进去了, 字却看不见
				demoPopup.FitContent()
				syncDemoPopupStatus("已追加一行: 面板向上长高, 底边不动")
				return 0
			})
			closeBtn := ui.NewButton(0, 0, 0, 0, "关闭", rb.Handle,
				ui.WithVariant(ui.ButtonPrimary), ui.WithSize(ui.ButtonSizeSm))
			xui.ItemWidth(closeBtn, 72)
			closeBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
				demoPopup.Hide()
				return 0
			})
		}))
	_, vp := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "屏幕弹窗", vp.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"右下角常驻面板: 默认不抢焦点、10 秒自动关;「可输入弹窗」演示 Focusable(true)。", vp.Handle))
	rp := xui.HStack(vp.Handle, 36, 8)
	demoPopupToggle = ui.NewButton(0, 0, 0, 0, "显示弹窗", rp.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(demoPopupToggle, 104)
	demoPopupToggle.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if demoPopup == nil {
			return 0
		}
		if demoPopup.IsOpen() {
			demoPopup.Hide()
		} else {
			if demoFocusPopup != nil {
				demoFocusPopup.Hide()
			}
			demoPopup.Show()
		}
		return 0
	})
	focusBtn := ui.NewButton(0, 0, 0, 0, "可输入弹窗", rp.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(focusBtn, 104)
	focusBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if demoFocusPopup == nil {
			demoFocusPopup = ui.NewScreenPopup(hWindow,
				ui.WithScreenPopupWidth(340),
				ui.WithScreenPopupFocusable(true),
				ui.WithScreenPopupHeader("可输入弹窗"),
				ui.WithScreenPopupOnOpenChange(func(open bool) {
					if open {
						syncDemoPopupStatus("已显示且没抢前台: 点输入框激活后打字, ESC 收起")
					} else {
						syncDemoPopupStatus("可输入弹窗已收起: 失焦不会自动关, ESC/X/按钮才会")
					}
				}),
				ui.WithScreenPopupContent(func(c int) {
					xui.ItemFill(ui.NewLabel(0, 0, 0, 20,
						"Focusable(true): 点击后激活可输入, 失焦不关。", c).
						SetMuted().SetFontSize(13))
					xui.ItemFill(ui.NewInput(0, 0, 0, 0, c,
						ui.WithPlaceholder("在这里输入文字, ESC 收起...")))
					closeBtn := ui.NewButton(0, 0, 0, 0, "收起", c,
						ui.WithVariant(ui.ButtonPrimary), ui.WithSize(ui.ButtonSizeSm))
					xui.ItemWidth(closeBtn, 72)
					closeBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
						demoFocusPopup.Hide()
						return 0
					})
				}))
		}
		if demoPopup != nil {
			demoPopup.Hide()
		}
		demoFocusPopup.Show()
		return 0
	})
	demoPopupStatus = ui.NewLabel(0, 0, 0, 20, "尚未显示", vp.Handle).SetMuted().SetFontSize(13)
	xui.ItemFill(demoPopupStatus)
	xui.ItemFill(desc(vp.Handle,
		"Header 在右上角放 X; Duration 到点自动关(0 = 常驻); FitContent 让窗口向上生长; "+
			"Focusable(true): 显示不抢前台, 点击后激活可输入, 失焦不自动关。"))

	// ---- 第三行: 空状态 ----
	rEmpty := row(body.Handle, 0, gap)

	_, ve := autoCard(xui.VStack(rEmpty.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "空状态(图标底板)", ve.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Empty: 图标块 + 标题 + 描述 + 动作区; p-6 / gap-6, 整块居中", ve.Handle))
	empty1 := ui.NewEmpty(0, 0, 0, 0, ve.Handle)
	xui.ItemFill(empty1)
	eh := empty1.Header()
	ui.NewEmptyMedia(eh.Handle, icons.IconFolderCode, ui.WithEmptyMediaVariant(ui.EmptyMediaIcon))
	ui.NewEmptyTitle(eh.Handle, "还没有项目")
	ui.NewEmptyDescription(eh.Handle, "你还没有创建任何项目, 从新建或导入开始。")
	emptyRow := xui.HStack(empty1.Content().Handle, 36, 8)
	xui.ItemWidth(emptyRow, 208) // 两枚 100 宽按钮 + 8 间距: 行宽按内容定, 才能被空状态居中
	createBtn := ui.NewButton(0, 0, 0, 0, "新建项目", emptyRow.Handle)
	xui.ItemWidth(createBtn, 100)
	importBtn := ui.NewButton(0, 0, 0, 0, "导入项目", emptyRow.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(importBtn, 100)

	_, ve2 := autoCard(xui.VStack(rEmpty.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "空状态(默认媒体 / 尾部链接)", ve2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"默认媒体无底板(24 图标); 自定义元素直接挂 Body, 随整块居中", ve2.Handle))
	empty2 := ui.NewEmpty(0, 0, 0, 0, ve2.Handle)
	xui.ItemFill(empty2)
	eh2 := empty2.Header()
	ui.NewEmptyMedia(eh2.Handle, icons.IconInbox)
	ui.NewEmptyTitle(eh2.Handle, "收件箱为空")
	ui.NewEmptyDescription(eh2.Handle, "新消息会出现在这里。")
	learn := ui.NewButton(0, 0, 0, 0, "了解通知设置", empty2.Body().Handle,
		ui.WithVariant(ui.ButtonLink), ui.WithSize(ui.ButtonSizeSm))
	tip.Bind(learn.Handle, "Empty 不画边框(上游默认也没有可见边框); 需要边框时放进 Card")
}

func toggleDemoTray() {
	if demoTray != nil && demoTray.Visible() {
		demoTray.Destroy()
		demoTray = nil
		if demoTrayButton != nil {
			demoTrayButton.SetText("启用托盘")
		}
		return
	}
	hIcon := demoAppIcon()
	if hIcon == 0 {
		fmt.Println("托盘图标资源 APP 加载失败")
		return
	}
	demoTray = ui.NewTray(demoWindow.Handle, "", "xui demo",
		ui.WithTrayIconHandle(hIcon), ui.WithTrayMenu(buildDemoTrayMenu))
	if demoTray == nil {
		fmt.Println("托盘图标创建失败")
		return
	}
	if demoTrayButton != nil {
		demoTrayButton.SetText("停用托盘")
	}
}

// syncDemoPopupStatus 同步「屏幕弹窗」卡的按钮文案与状态行(msg 为空时按开关状态给默认文案)。
func syncDemoPopupStatus(msg string) {
	if demoPopup == nil || demoPopupStatus == nil || demoPopupToggle == nil {
		return
	}
	if demoPopup.IsOpen() {
		demoPopupToggle.SetText("隐藏弹窗").Redraw(false)
		if msg == "" {
			msg = "已显示: 面板贴在屏幕右下角, 焦点仍在主窗口"
		}
	} else {
		demoPopupToggle.SetText("显示弹窗").Redraw(false)
		if msg == "" {
			msg = "已隐藏"
		}
	}
	demoPopupStatus.SetText(msg).Redraw(false)
}

func buildDemoTrayMenu(m *ui.Menu) {
	m.Label("xui demo")
	m.Item("显示主窗口", ui.MenuIcon(icons.IconAppWindow), ui.MenuOnSelect(showDemoWindow))
	m.Item("发送屏幕通知", ui.MenuIcon(icons.IconBell), ui.MenuOnSelect(func() {
		if demoScreen != nil {
			demoScreen.Success("来自托盘", "主窗口最小化/隐藏时, 屏幕右下角也能看到通知。")
		}
	}))
	m.Separator()
	m.Item("退出", ui.MenuDestructive(), ui.MenuOnSelect(quitDemo))
}

func showDemoWindow() {
	if demoWindow == nil {
		return
	}
	demoWindow.Show(true)
	if demoWindow.IsMaxWindow() {
		demoWindow.ShowWindow(xcc.SW_SHOWMAXIMIZED)
	} else {
		demoWindow.ShowWindow(xcc.SW_RESTORE)
	}
	wapi.SetForegroundWindow(demoWindow.GetHWND())
}

func quitDemo() {
	if demoScreen != nil {
		demoScreen.Clear()
	}
	if demoTray != nil {
		demoTray.Destroy()
		demoTray = nil
	}
	if demoWindow != nil {
		demoWindow.CloseWindow()
	}
	app.PostQuitMessage(0)
}
