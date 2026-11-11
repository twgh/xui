package main

import (
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

// ===== 页 8: 表单4(日期与时间 / 月历 / 范围与预设) =====

func buildForm4Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)
	colL := xui.VStack(r.Handle, 1, gap)
	colR := xui.VStack(r.Handle, 1, gap)

	// ---- 左列 1: 日期与时间 ----
	_, v5 := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "日期与时间", v5.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "触发区自绘(outline 按钮), 弹层是自建阴影窗: 月历全自绘, 时间框两列滚动列表", v5.Handle))

	xui.ItemFill(hint(v5.Handle, "日期框(点击弹出月历卡片)"))
	dp := ui.NewDatePicker(0, 0, 0, 0, v5.Handle, ui.WithDatePickerPlaceholder("选择日期"))
	xui.ItemFill(dp)
	dpVal := ui.NewLabel(0, 0, 0, 0, "", v5.Handle).SetMuted()
	dpVal.SetText("未选择")
	xui.ItemFill(dpVal)
	dp.OnChange(func(t time.Time) {
		dpVal.SetText("选择: " + t.Format("2006年1月2日")).Redraw()
	})

	xui.ItemFill(hint(v5.Handle, "时间框(两列列表: 时 / 分)"))
	// 直接挂在卡片内容区, 不再套 HStack: 隐藏页首次显示时, 嵌套水平容器的
	// 元素矩形会停在旧位置, 时间框的命中区与画面错位(需滚一下滚轮才恢复)。
	tp := ui.NewTimePicker(0, 0, 0, 0, v5.Handle, ui.WithTimePickerValue(9, 30))
	xui.ItemFill(tp)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "时间框为两列滚动列表, 选择即生效", v5.Handle).SetMuted())

	// ---- 右列 1: 月历卡片 ----
	_, v6 := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "月历卡片", v6.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"标题年月可下拉切换; 周末禁用(灰显不可点); 页脚 = 内置「今天」+ 自定义「清除」", v6.Handle))
	// 默认尺寸已是"格子 36 / 字号 16"的比例; 元素再放大时内部等比缩放
	var cal *ui.Calendar
	cal = ui.NewCalendar(0, 0, 0, 0, v6.Handle,
		ui.WithCalendarSelected(time.Now()),
		ui.WithCalendarDisabled(ui.CalendarDisableWeekend),
		ui.WithCalendarFooterToday(),
		ui.WithCalendarFooter(ui.CalendarFooterItem{Text: "清除", OnClick: func() {
			cal.SetSelected(time.Time{})
		}}))
	cal.OnSelect(func(t time.Time) { tip.Bind(cal.Handle, t.Format("2006年1月2日")) })

	// ---- 第二行: 月历 / 日期框的 shadcn 文档能力 ----
	r2 := row(body.Handle, 0, gap)
	colL2 := xui.VStack(r2.Handle, 1, gap)
	colR2 := xui.VStack(r2.Handle, 1, gap)

	// ---- 左列 1: 日期范围框 ----
	_, v7 := autoCard(colL2.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "日期范围框", v7.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"两次点击落区间; 弹层并排两个月份 + 预设按钮; 起点未落终点时保持打开", v7.Handle))
	xui.ItemFill(hint(v7.Handle, "范围选择(预设: 今天 / 未来 7 天 / 未来 30 天)"))
	now := time.Now()
	rf := time.Date(now.Year(), time.January, 20, 0, 0, 0, 0, time.Local)
	rt := rf.AddDate(0, 0, 20)
	dr := ui.NewDatePicker(0, 0, 0, 0, v7.Handle,
		ui.WithDatePickerRangeValue(rf, rt),
		ui.WithDatePickerYearRange(2000, now.Year()+1),
		ui.WithDatePickerPresets(
			ui.CalPreset("今天", 0, 0),
			ui.CalPreset("未来 7 天", 0, 7),
			ui.CalPreset("未来 30 天", 0, 30)))
	xui.ItemFill(dr)
	drVal := ui.NewLabel(0, 0, 0, 0, "范围: "+dr.Text(), v7.Handle).SetMuted()
	xui.ItemFill(drVal)
	dr.OnRangeChange(func(from, to time.Time) {
		if to.IsZero() {
			drVal.SetText("起点: " + from.Format("2006年1月2日") + " · 再选终点").Redraw()
			return
		}
		drVal.SetText("范围: " + from.Format("2006年1月2日") + " - " + to.Format("2006年1月2日")).Redraw()
	})

	// ---- 左列 2: 可输入日期框 ----
	_, v8 := autoCard(colL2.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "可输入日期框", v8.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"InputGroup 形态: 键入 2026-10-22 / 2026年10月22日 / 明天 / 3天后, Enter 或失焦回写; 右侧日历图标打开弹层", v8.Handle))
	xui.ItemFill(hint(v8.Handle, "出生日期(年份范围 1900 - 今天, 弹层带年月下拉)"))
	dp2 := ui.NewDatePicker(0, 0, 0, 0, v8.Handle,
		ui.WithDatePickerEditable(),
		ui.WithDatePickerYearRange(1900, now.Year()),
		ui.WithDatePickerPlaceholder("1990-01-01 或 明天"),
		ui.WithDatePickerValue(time.Date(1990, time.January, 1, 0, 0, 0, 0, time.Local)))
	xui.ItemFill(dp2)
	dp2Val := ui.NewLabel(0, 0, 0, 0, "输入值: "+dp2.Text(), v8.Handle).SetMuted()
	xui.ItemFill(dp2Val)
	dp2.OnChange(func(t time.Time) {
		if t.IsZero() {
			dp2Val.SetText("输入值: (空)").Redraw()
			return
		}
		dp2Val.SetText("输入值: " + t.Format("2006年1月2日")).Redraw()
	})

	// ---- 右列 1: 范围月历(宽度跟随列, 高度按宽度自动配平) ----
	_, v9 := autoCard(colR2.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "范围月历", v9.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"两个并排月份 + 范围选择; 今天之后的日期禁用(CalendarDisableAfter)", v9.Handle))
	calRange := ui.NewCalendar(0, 0, 0, 0, v9.Handle,
		ui.WithCalendarMonths(2),
		ui.WithCalendarRange(),
		ui.WithCalendarSelectedRange(rf, rt),
		ui.WithCalendarDisabled(ui.CalendarDisableAfter(now)))
	xui.ItemFill(calRange)
	rangeInfo := ui.NewLabel(0, 0, 0, 0, "范围: "+dateRangeText(rf, rt), v9.Handle).SetMuted()
	xui.ItemFill(rangeInfo)
	calRange.OnRangeSelect(func(from, to time.Time) {
		rangeInfo.SetText("范围: " + dateRangeText(from, to)).Redraw()
	})

	// ---- 右列 2: 周数 / 预订日期 / 预设 ----
	_, v10 := autoCard(colR2.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "周数与预订日期", v10.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"ISO 周数列(周一开头); 预订日期删除线 + 禁用不可点; 页脚预设按钮", v10.Handle))
	booked := make([]time.Time, 0, 5)
	for i := 0; i < 5; i++ {
		booked = append(booked, now.AddDate(0, 0, 3+i))
	}
	calWeek := ui.NewCalendar(0, 0, 0, 0, v10.Handle,
		ui.WithCalendarWeekNumbers(),
		ui.WithCalendarSelected(now),
		ui.WithCalendarDisabled(ui.CalendarDisableDates(booked...)),
		ui.WithCalendarStrike(ui.CalendarStrikeDates(booked...)),
		ui.WithCalendarPresets(
			ui.CalPreset("今天", 0, 0),
			ui.CalPreset("明天", 1, 0),
			ui.CalPreset("一周后", 7, 0)))
	// 周数日历保持默认宽度: 被卡片拉满会把格子/字号一起放大。
	xui.ItemWidth(calWeek, ui.CalWidthForWeekNumbers(1))
}

// dateRangeText 是 demo 里展示范围用的短文本。
func dateRangeText(from, to time.Time) string {
	if from.IsZero() {
		return "未选择"
	}
	if to.IsZero() {
		return from.Format("2006年1月2日") + " - 再选终点"
	}
	return from.Format("2006年1月2日") + " - " + to.Format("2006年1月2日")
}
