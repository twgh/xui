package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

// ===== 页 23: 对话2(MessageScroller: 轮次锚定 / 贴底跟随 / 前置历史 / 跳转) =====

// chatScrollerDemo 演示面板的可变状态。
type chatScrollerDemo struct {
	ms   *ui.MessageScroller
	in   *ui.Input
	send *ui.Button

	state   *ui.Label
	anchor  *ui.Label
	visible *ui.Label

	seq     int // 新消息序号
	oldSeq  int // 前置历史序号
	stream  bool
	streamB *ui.Bubble
	streamT string
	streamI int
}

func buildChatScrollerPage(body *widget.LayoutEle) {
	d := &chatScrollerDemo{seq: 6}
	r := row(body.Handle, 0, gap)
	left := col(r.Handle, 5, gap)
	right := col(r.Handle, 4, gap)

	// ---- 左列: 可交互会话 ----
	_, v := autoCard(left.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "消息滚动(MessageScroller)", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"发送后新轮次顶到视口顶部; 回复逐字长高; 上滚接手、回底恢复跟随", v.Handle))

	d.ms = ui.NewMessageScroller(0, 0, 0, 0, v.Handle, ui.WithMessageScrollerAutoScroll(true))
	xui.ItemFill(d.ms)
	xui.ItemHeight(d.ms, 440)
	d.ms.SetViewPadding(16, 16, 16, 16)
	d.ms.ContentStack(24)

	seed := []struct{ user, reply string }{
		{"Can you check why the invoices stopped syncing?", "On it. I'll scan the last 24 hours of sync logs."},
		{"It started right after Friday's deploy.", "Found it: the retry job exits before the final batch. Replaying the failed queue now."},
		{"Any missing invoices?", "Two: INV-2418 and INV-2419. Both are queued for the next pass."},
		{"Perfect, thanks.", "All caught up. I'll keep watching for an hour."},
	}
	for i, s := range seed {
		addChatScrollerTurn(d.ms, fmt.Sprintf("u%d", i+1), s.user, s.reply)
	}

	rowIn := row(v.Handle, 0, 8)
	d.in = ui.NewInput(0, 0, 0, 0, rowIn.Handle, ui.WithPlaceholder("输入一条消息…"))
	xui.ItemWeight(d.in, 1)
	d.send = ui.NewButton(0, 0, 0, 0, "发送", rowIn.Handle)
	d.send.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		d.sendMessage()
		return 0
	})
	xui.ItemFill(hint(v.Handle, "发送时: 用户消息是轮次锚点(顶到视口), 回复作为同一轮在下方流式长高。"))

	// ---- 右列: 控制 ----
	_, vc := autoCard(right.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "滚动控制", vc.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"自动跟随 = 只在贴底时跟; 任意上滚立即放手, 回底重新跟随", vc.Handle))

	r1 := row(vc.Handle, 0, 8)
	b := ui.NewButton(0, 0, 0, 0, "滚到顶部", r1.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWeight(b, 1)
	b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int { d.ms.ScrollToStart(); return 0 })
	b = ui.NewButton(0, 0, 0, 0, "滚到底部", r1.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWeight(b, 1)
	b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int { d.ms.ScrollToEnd(); return 0 })

	r2 := row(vc.Handle, 0, 8)
	b = ui.NewButton(0, 0, 0, 0, "跳到第 2 轮", r2.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWeight(b, 1)
	b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if !d.ms.ScrollToMessage("u2") {
			d.note("u2 已被移除或容器未上屏")
		}
		return 0
	})
	b = ui.NewButton(0, 0, 0, 0, "载入更早历史", r2.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWeight(b, 1)
	b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		d.prependHistory()
		return 0
	})

	// ---- 右列: 实时状态 ----
	_, vs := autoCard(right.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "滚动状态", vs.Handle))
	d.state = ui.NewLabel(0, 0, 0, 20, "可上滚: 否 · 可下滚: 否", vs.Handle)
	xui.ItemFill(d.state)
	d.anchor = ui.NewLabel(0, 0, 0, 20, "当前轮次: -", vs.Handle)
	xui.ItemFill(d.anchor)
	d.visible = ui.NewLabel(0, 0, 0, 42, "可见: -", vs.Handle).SetWrap()
	xui.ItemFill(d.visible)
	xui.ItemFill(hint(vs.Handle, "状态来自 OnScrollStateChange / OnVisibilityChange; 当前轮次 = 视口顶部所在的锚点消息。"))

	// ---- 右列: 行为说明 ----
	_, vb := autoCard(right.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "行为", vb.Handle))
	for _, line := range []string{
		"新轮次: 第一个锚点顶到视口顶部, 上方保留上一条的 64px 尾巴",
		"流式回复: 轮次锚点保持不动, 回复在下方长高",
		"前置历史: 默认保持视口里的内容不跳动",
		"浮动按钮: 离底 16px 居中, 200ms 滑入 / 400ms 滑出, 点击平滑滚动",
	} {
		xui.ItemFill(hint(vb.Handle, "• "+line))
	}

	d.ms.OnScrollStateChange(func(start, end bool) {
		d.state.SetText(fmt.Sprintf("可上滚: %s · 可下滚: %s", yesNo(start), yesNo(end)))
	})
	d.ms.OnVisibilityChange(func(anchorID string, visible []string) {
		if anchorID == "" {
			d.anchor.SetText("当前轮次: -")
		} else {
			d.anchor.SetText("当前轮次: " + anchorID)
		}
		show := visible
		if len(show) > 5 {
			show = show[:5]
		}
		d.visible.SetText("可见: " + strings.Join(show, ", ") + suffixEllipsis(len(visible), 5))
	})
}

// addChatScrollerUser 追加一条用户消息(作为轮次锚点)。
func addChatScrollerUser(ms *ui.MessageScroller, id, text string) {
	it := ms.AddItem(id, true)
	m := ui.NewMessage(0, 0, 0, 0, it.Handle, ui.WithMessageAlign(ui.MessageAlignEnd))
	xui.ItemFill(m)
	ui.NewBubble(m.Content().Handle, text, ui.WithBubbleVariant(ui.BubbleTinted))
}

// addChatScrollerTurn 追加一轮(用户锚点 + 助手回复), 返回回复气泡。
func addChatScrollerTurn(ms *ui.MessageScroller, id, userText, reply string) *ui.Bubble {
	addChatScrollerUser(ms, id, userText)
	it := ms.AddItem("a"+strings.TrimPrefix(id, "u"), false)
	m := ui.NewMessage(0, 0, 0, 0, it.Handle)
	xui.ItemFill(m)
	ui.NewMessageMeta(m.Header().Handle, "Assistant")
	return ui.NewBubble(m.Content().Handle, reply, ui.WithBubbleVariant(ui.BubbleMuted))
}

func (d *chatScrollerDemo) sendMessage() {
	if d.stream {
		return
	}
	text := strings.TrimSpace(d.in.Value())
	if text == "" {
		d.note("先输入一条消息")
		return
	}
	d.in.SetValue("")
	d.seq++
	id := fmt.Sprintf("u%d", d.seq)
	addChatScrollerUser(d.ms, id, text)

	// 助手回复: 先建空气泡, 之后逐字填(内容变高会触发"轮次锚点保持"链路)。
	it := d.ms.AddItem("a"+fmt.Sprintf("%d", d.seq), false)
	m := ui.NewMessage(0, 0, 0, 0, it.Handle)
	xui.ItemFill(m)
	ui.NewMessageMeta(m.Header().Handle, "Assistant")
	d.streamB = ui.NewBubble(m.Content().Handle, "", ui.WithBubbleVariant(ui.BubbleMuted))
	d.streamT = "Got it. I'll check that and post the result here as soon as the run finishes."
	d.streamI = 0
	d.stream = true
	d.send.Enable(false)
	app.AfterFunc(80*time.Millisecond, d.streamTick)
}

func (d *chatScrollerDemo) streamTick() {
	if d.streamB == nil {
		return
	}
	d.streamI++
	d.streamB.SetText(d.streamT[:d.streamI])
	if d.streamI >= len(d.streamT) {
		d.stream = false
		d.send.Enable(true)
		return
	}
	app.AfterFunc(24*time.Millisecond, d.streamTick)
}

// prependHistory 在开头插入两条更早的消息; 视口里原来的内容应原地不动。
func (d *chatScrollerDemo) prependHistory() {
	d.oldSeq++
	id := fmt.Sprintf("h%d", d.oldSeq)
	addChatScrollerTurn(d.ms, id, "Earlier question #"+fmt.Sprint(d.oldSeq), "Earlier context loaded from history.")
	d.note("已载入更早历史(视口保持不动)")
}

func (d *chatScrollerDemo) note(s string) {
	d.visible.SetText(s)
}

func yesNo(v bool) string {
	if v {
		return "是"
	}
	return "否"
}

func suffixEllipsis(n, limit int) string {
	if n > limit {
		return " …"
	}
	return ""
}
