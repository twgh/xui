package main

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 22: 对话(消息 / 气泡 / 标记 / 轨迹 / 附件) =====

func buildChatPage(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)
	left := col(r.Handle, 5, gap)
	right := col(r.Handle, 4, gap)

	// ---- 左列: 一段完整会话 ----
	_, v := autoCard(left.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "会话", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Message / Bubble: 头像贴底、页眉页脚、左右对齐、气泡组、反应胶囊", v.Handle))

	grp := ui.NewMessageGroup(0, 0, 0, 0, v.Handle)
	xui.ItemFill(grp)

	// 自己(右对齐) + 页眉 + 页脚
	m := ui.NewMessage(0, 0, 0, 0, grp.Handle, ui.WithMessageAlign(ui.MessageAlignEnd))
	xui.ItemFill(m)
	ui.NewMessageMeta(m.Header().Handle, "You · now")
	ui.NewBubble(m.Content().Handle, "Deploying to prod real quick.")
	ft := m.Footer()
	ui.NewMessageMeta(ft.Handle, "Delivered")
	ui.NewButton(0, 0, 0, 0, "Undo", ft.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeXs))

	// 对方(左对齐、头像 + 页眉)
	m = ui.NewMessage(0, 0, 0, 0, grp.Handle)
	xui.ItemFill(m)
	ui.NewAvatar(0, 0, 0, 0, m.AvatarSlot().Handle, "CN")
	ui.NewMessageMeta(m.Header().Handle, "Casey · now")
	ui.NewBubble(m.Content().Handle, "It's 4:55 PM. On a Friday.", ui.WithBubbleVariant(ui.BubbleMuted))

	// 自己(tinted)
	m = ui.NewMessage(0, 0, 0, 0, grp.Handle, ui.WithMessageAlign(ui.MessageAlignEnd))
	xui.ItemFill(m)
	ui.NewBubble(m.Content().Handle, "It's a one-line change.", ui.WithBubbleVariant(ui.BubbleTinted))

	// 对方: 头像 + 页眉 + 气泡 + 反应胶囊 + 页脚
	m = ui.NewMessage(0, 0, 0, 0, grp.Handle)
	xui.ItemFill(m)
	ui.NewAvatar(0, 0, 0, 0, m.AvatarSlot().Handle, "O")
	ui.NewMessageMeta(m.Header().Handle, "Olivia · 1m ago")
	rb := ui.NewBubble(m.Content().Handle,
		"I checked the logs. The retry finished and the missing invoices are included now.",
		ui.WithBubbleVariant(ui.BubbleMuted))
	rb.AddReaction("Thanks", nil).AddReaction("Nice", nil)
	ui.NewMessageMeta(m.Footer().Handle, "已读 10:24")

	// 分隔标记 + 气泡组
	mk := ui.NewMarker(0, 0, 0, 0, grp.Handle, ui.WithMarkerVariant(ui.MarkerSeparator))
	xui.ItemFill(mk)
	mk.Content("今天 10:24")

	m = ui.NewMessage(0, 0, 0, 0, grp.Handle)
	xui.ItemFill(m)
	bg := ui.NewBubbleGroup(0, 0, 0, 0, m.Content().Handle)
	xui.ItemFill(bg)
	ui.NewBubble(bg.Handle, "It's always a one-line change. Make sure to run the tests this time.",
		ui.WithBubbleVariant(ui.BubbleMuted))
	ui.NewBubble(bg.Handle, "Alright, go for it.", ui.WithBubbleVariant(ui.BubbleMuted))

	// ghost 正文 + 轨迹(推理过程)
	m = ui.NewMessage(0, 0, 0, 0, grp.Handle)
	xui.ItemFill(m)
	m.SetMetaFlush(true)
	ui.NewMessageMeta(m.Header().Handle, "Eve")
	ui.NewBubble(m.Content().Handle,
		"I found the following error in the logs: no users were found for the selected period. "+
			"Can you check whether the users are being created?",
		ui.WithBubbleVariant(ui.BubbleGhost))
	tr := ui.NewTrace(0, 0, 0, 0, m.Content().Handle)
	xui.ItemFill(tr)
	ui.NewTraceLine(0, 0, 0, 0, tr.Handle,
		ui.WithTraceStatus(ui.TraceStatusRunning),
		ui.WithTraceIcon(icons.IconFileSearch),
		ui.WithTraceTitle("Scanning 52 files"))
	ui.NewTraceLine(0, 0, 0, 0, tr.Handle,
		ui.WithTraceStatus(ui.TraceStatusError),
		ui.WithTraceIcon(icons.IconX),
		ui.WithTraceTitle("Read failed: permission denied"),
		ui.WithTraceDescription("retry"))
	ui.NewTraceLine(0, 0, 0, 0, tr.Handle,
		ui.WithTraceStatus(ui.TraceStatusPending),
		ui.WithTraceIcon(icons.IconClock),
		ui.WithTraceTitle("Waiting for tool output"))
	tc := ui.NewTraceCollapsible(0, 0, 0, 0, tr.Handle,
		ui.WithTraceCollapsibleTrigger(
			ui.WithTraceIcon(icons.IconTerminal),
			ui.WithTraceTitle("Ran node --version")))
	tcBody := tc.ContentStack(4)
	lbl := ui.NewLabel(0, 0, 0, 20, "v24.18.0", tcBody.Handle)
	lbl.SetMuted().SetFontSize(12)
	xui.ItemFill(lbl)

	// ---- 右列: 气泡变体 ----
	_, vb := autoCard(right.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "气泡变体", vb.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"7 个变体: default / secondary / muted / tinted / outline / ghost / destructive", vb.Handle))
	for _, it := range []struct {
		name string
		v    ui.BubbleVariant
		text string
	}{
		{"default", ui.BubbleDefault, "Deploying to prod real quick."},
		{"secondary", ui.BubbleSecondary, "It's a one-line change."},
		{"muted", ui.BubbleMuted, "It's 4:55 PM. On a Friday."},
		{"tinted", ui.BubbleTinted, "Why is the dashboard showing zero users"},
		{"outline", ui.BubbleOutline, "可点击的 outline 气泡(悬停加深)"},
		{"ghost", ui.BubbleGhost, "Ghost 气泡没有底色、占满整行。"},
		{"destructive", ui.BubbleDestructive, "Something went wrong. Any idea?"},
	} {
		xui.ItemFill(hint(vb.Handle, it.name))
		if it.v == ui.BubbleOutline {
			ui.NewBubble(vb.Handle, it.text, ui.WithBubbleVariant(it.v),
				ui.WithBubbleOnClick(func() { fmt.Println("outline bubble clicked") }))
		} else {
			ui.NewBubble(vb.Handle, it.text, ui.WithBubbleVariant(it.v))
		}
	}

	// ---- 右列: 标记 ----
	_, vm := autoCard(right.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "标记", vm.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Marker: 状态行 / 底边框 / 居中分隔, running 前可放 Spinner", vm.Handle))
	mk = ui.NewMarker(0, 0, 0, 0, vm.Handle, ui.WithMarkerIcon(icons.IconCheck))
	xui.ItemFill(mk)
	mk.Content("Explored 4 files")
	mk = ui.NewMarker(0, 0, 0, 0, vm.Handle, ui.WithMarkerVariant(ui.MarkerBorder))
	xui.ItemFill(mk)
	mk.Content("System note: context window usage 42%")
	mk = ui.NewMarker(0, 0, 0, 0, vm.Handle, ui.WithMarkerVariant(ui.MarkerSeparator))
	xui.ItemFill(mk)
	mk.Content("earlier")
	mk = ui.NewMarker(0, 0, 0, 0, vm.Handle)
	xui.ItemFill(mk)
	ui.NewSpinner(mk.Handle, ui.WithSpinnerMuted())
	mk.Content("Checking the logs…")

	// ---- 右列: 轨迹 ----
	_, vt := autoCard(right.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "轨迹", vt.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Trace: running 微光 / error 红 / pending 60%, 可折叠段带 chevron 旋转", vt.Handle))
	tr = ui.NewTrace(0, 0, 0, 0, vt.Handle)
	xui.ItemFill(tr)
	ui.NewTraceLine(0, 0, 0, 0, tr.Handle,
		ui.WithTraceStatus(ui.TraceStatusRunning),
		ui.WithTraceIcon(icons.IconFileSearch),
		ui.WithTraceTitle("Scanning 52 files"))
	ui.NewTraceLine(0, 0, 0, 0, tr.Handle,
		ui.WithTraceIcon(icons.IconFileCode),
		ui.WithTraceTitle("Read components/button.go"),
		ui.WithTraceTrailingIcon(icons.IconChevronRight),
		ui.WithTraceOnClick(func() { fmt.Println("trace line clicked") }))
	tc = ui.NewTraceCollapsible(0, 0, 0, 0, tr.Handle,
		ui.WithTraceCollapsibleTrigger(
			ui.WithTraceIcon(icons.IconTerminal),
			ui.WithTraceTitle("Ran npm --version"),
			ui.WithTraceDescription("1.2s")))
	tcBody = tc.ContentStack(4)
	lbl = ui.NewLabel(0, 0, 0, 20, "10.9.4", tcBody.Handle)
	lbl.SetMuted().SetFontSize(12)
	xui.ItemFill(lbl)

	// ---- 右列: 附件 ----
	_, va := autoCard(right.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "附件", va.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Attachment: 5 状态 · 水平/垂直 · 3 档尺寸; idle 虚线、error 红、上传中标题微光", va.Handle))
	att := ui.NewAttachment(0, 0, 0, 0, va.Handle,
		ui.WithAttachmentIcon(icons.IconClock), ui.WithAttachmentState(ui.AttachmentIdle))
	xui.ItemFill(att)
	att.SetText("selected-file.pdf", "Ready to upload")
	att = ui.NewAttachment(0, 0, 0, 0, va.Handle,
		ui.WithAttachmentIcon(icons.IconFileText), ui.WithAttachmentState(ui.AttachmentUploading))
	xui.ItemFill(att)
	att.SetText("sales-dashboard.pdf", "PDF · 2.4 MB")
	att.Action(icons.IconX, nil)
	att = ui.NewAttachment(0, 0, 0, 0, va.Handle,
		ui.WithAttachmentIcon(icons.IconFileSpreadsheet), ui.WithAttachmentState(ui.AttachmentProcessing))
	xui.ItemFill(att)
	att.SetText("customer-import.csv", "CSV · 18 KB")
	att = ui.NewAttachment(0, 0, 0, 0, va.Handle,
		ui.WithAttachmentIcon(icons.IconFileCode), ui.WithAttachmentState(ui.AttachmentError))
	xui.ItemFill(att)
	att.SetText("message-renderer.tsx", "Upload failed")
	att.Action(icons.IconX, nil)
	att = ui.NewAttachment(0, 0, 0, 0, va.Handle,
		ui.WithAttachmentIcon(icons.IconFileArchive))
	xui.ItemFill(att)
	att.SetText("source-assets.zip", "ZIP · 4.2 MB")
	att.Action(icons.IconX, nil)

	xui.ItemFill(hint(va.Handle, "尺寸档(默认 / sm / xs)"))
	ag := ui.NewAttachmentGroup(0, 0, 0, 0, va.Handle)
	xui.ItemFill(ag)
	for i, sz := range []ui.AttachmentSize{ui.AttachmentSizeDefault, ui.AttachmentSizeSm, ui.AttachmentSizeXs} {
		a := ui.NewAttachment(0, 0, 0, 0, ag.Handle,
			ui.WithAttachmentIcon(icons.IconFile), ui.WithAttachmentSize(sz),
			ui.WithAttachmentWidth(int32(240-i*20)))
		a.SetText([]string{"report.pdf", "notes.txt", "meta.json"}[i],
			[]string{"PDF · 1.1 MB", "TXT · 3 KB", "JSON · 1 KB"}[i])
	}

	xui.ItemFill(hint(va.Handle, "垂直朝向(适合图片/文件网格)"))
	ag2 := ui.NewAttachmentGroup(0, 0, 0, 0, va.Handle)
	xui.ItemFill(ag2)
	for i, icon := range []string{icons.IconPresentation, icons.IconFileArchive, icons.IconFileText} {
		a := ui.NewAttachment(0, 0, 0, 0, ag2.Handle,
			ui.WithAttachmentIcon(icon),
			ui.WithAttachmentOrientation(ui.AttachmentVertical))
		a.SetText([]string{"quarterly-review.key", "assets.zip", "readme.md"}[i],
			[]string{"Keynote · 9 MB", "ZIP · 4.2 MB", "MD · 2 KB"}[i])
	}
}
