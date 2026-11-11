package main

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

// ===== 页 19: 图表(Chart) =====
// 每张卡片一张图: 柱状(分组/堆叠) / 折线 / 面积(堆叠) / 环形 / 饼图。
// 图表高度是显式常量(240): 自绘几何按元素矩形算, 高度必须确定, 不能靠内容自撑。
func buildChartPage(body *widget.LayoutEle) {
	months := []string{"1月", "2月", "3月", "4月", "5月", "6月"}
	desktop := ui.ChartSeries{Label: "桌面端", Values: []float64{186, 305, 237, 273, 209, 214}}
	mobile := ui.ChartSeries{Label: "移动端", Values: []float64{80, 200, 120, 190, 130, 140}}

	// ---- 第一行: 分组柱 + 堆叠柱 ----
	r := row(body.Handle, 0, gap)
	_, v := autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "柱状图", v.Handle))
	xui.ItemFill(desc(v.Handle, "分组柱 + 柱顶数值, 数值轴自动取整齐刻度。"))
	bar := ui.NewChart(0, 0, 0, 240, v.Handle,
		ui.WithChartData(months, desktop, mobile),
		ui.WithChartBarLabels())
	xui.ItemFill(bar)

	_, v = autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "堆叠柱状图", v.Handle))
	xui.ItemFill(desc(v.Handle, "WithChartStacked: 同类多路数据叠成一根柱。"))
	stacked := ui.NewChart(0, 0, 0, 240, v.Handle,
		ui.WithChartData(months, desktop, mobile),
		ui.WithChartStacked())
	xui.ItemFill(stacked)

	// ---- 第二行: 平滑折线 + 堆叠面积 ----
	r = row(body.Handle, 0, gap)
	_, v = autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "折线图", v.Handle))
	xui.ItemFill(desc(v.Handle, "WithChartCurve + WithChartDots: 平滑曲线与数据点。"))
	line := ui.NewChart(0, 0, 0, 240, v.Handle,
		ui.WithChartKind(ui.ChartLine),
		ui.WithChartData(months, desktop, mobile),
		ui.WithChartCurve(),
		ui.WithChartDots())
	xui.ItemFill(line)

	_, v = autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "面积图", v.Handle))
	xui.ItemFill(desc(v.Handle, "堆叠面积: 每层填在两条曲线之间。"))
	area := ui.NewChart(0, 0, 0, 240, v.Handle,
		ui.WithChartKind(ui.ChartArea),
		ui.WithChartData(months, desktop, mobile),
		ui.WithChartStacked())
	xui.ItemFill(area)

	// ---- 第三行: 环形(中心文字 + 点击回调) + 饼图 ----
	r = row(body.Handle, 0, gap)
	_, v = autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "环形图", v.Handle))
	xui.ItemFill(desc(v.Handle, "中心两行文字 + 图例, 点击扇区输出回调。"))
	status := ui.NewLabel(0, 0, 0, 24, "点击任一扇区查看回调", v.Handle).SetMuted()
	donutLabels := []string{"桌面端", "移动端", "平板", "其它"}
	donut := ui.NewChart(0, 0, 0, 240, v.Handle,
		ui.WithChartKind(ui.ChartDonut),
		ui.WithChartData(donutLabels,
			ui.ChartSeries{Label: "占比", Values: []float64{48, 32, 14, 6}}),
		ui.WithChartCenter("1,284", "总访问"),
		ui.WithChartInnerRatio(0.62))
	xui.ItemFill(donut)
	donut.OnPointClick(func(cat, _ int, value float64) {
		name := donutLabels[cat]
		status.SetText(fmt.Sprintf("点击: %s %.0f%%", name, value))
		status.Redraw(false)
	})
	xui.ItemFill(status)

	_, v = autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "饼图", v.Handle))
	xui.ItemFill(desc(v.Handle, "点图例临时隐藏/恢复一块扇区。"))
	pie := ui.NewChart(0, 0, 0, 240, v.Handle,
		ui.WithChartKind(ui.ChartPie),
		ui.WithChartData([]string{"已完成", "进行中", "未开始", "已取消"},
			ui.ChartSeries{Label: "任务", Values: []float64{42, 26, 22, 10}}),
		ui.WithChartYFormat(func(f float64) string { return fmt.Sprintf("%.0f 个", f) }))
	xui.ItemFill(pie)

	// ---- 第四行: 横向柱 + 径向 ----
	r = row(body.Handle, 0, gap)
	_, v = autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "横向柱状图", v.Handle))
	xui.ItemFill(desc(v.Handle, "WithChartHorizontal: 分类沿 y、数值沿 x, 数值贴在柱端。"))
	hbar := ui.NewChart(0, 0, 0, 240, v.Handle,
		ui.WithChartHorizontal(),
		ui.WithChartData(months, desktop, mobile),
		ui.WithChartBarLabels())
	xui.ItemFill(hbar)

	_, v = autoCard(xui.VStack(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "径向图", v.Handle))
	xui.ItemFill(desc(v.Handle, "ChartRadial: 每分类一层同心弧, 弧长按相对最大值。"))
	radial := ui.NewChart(0, 0, 0, 240, v.Handle,
		ui.WithChartKind(ui.ChartRadial),
		ui.WithChartData([]string{"桌面端", "移动端", "平板", "其它"},
			ui.ChartSeries{Label: "访问", Values: []float64{420, 260, 130, 70}}),
		ui.WithChartCenter("880", "总访问"))
	xui.ItemFill(radial)

	// ---- 第五行: 雷达(整行, 维度多时更舒展) ----
	r = row(body.Handle, 0, gap)
	_, v = autoCard(r.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "雷达图", v.Handle))
	xui.ItemFill(desc(v.Handle, "ChartRadar: 每分类一根等角轴, 每路数据一个闭合多边形。"))
	radar := ui.NewChart(0, 0, 0, 260, v.Handle,
		ui.WithChartKind(ui.ChartRadar),
		ui.WithChartData([]string{"性能", "易用", "功能", "稳定", "文档", "生态"}, desktop, mobile),
		ui.WithChartDots())
	xui.ItemFill(radar)
}
