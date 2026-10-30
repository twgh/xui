package ui

import (
	"math"
	"testing"

	"github.com/twgh/xcgui/xc"
)

// approxEq 比较浮点(刻度计算里允许 1e-9 级误差)。
func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestChartNiceTicks(t *testing.T) {
	cases := []struct {
		lo, hi float64
		count  int
		want   []float64
	}{
		{0, 100, 5, []float64{0, 25, 50, 75, 100}},
		{0, 1, 5, []float64{0, 0.25, 0.5, 0.75, 1}},
		{0, 9, 5, []float64{0, 2.5, 5, 7.5, 10}},
		// 范围要向上取整到整步长, 首尾可以超出输入(轴留白)
		{0, 173, 5, []float64{0, 50, 100, 150, 200}},
		// 负值范围: 两端都吸附到整步长(100/4 = 25)
		{-30, 70, 5, []float64{-50, -25, 0, 25, 50, 75}},
	}
	for _, c := range cases {
		got := chartNiceTicks(c.lo, c.hi, c.count)
		if len(got) != len(c.want) {
			t.Fatalf("chartNiceTicks(%v,%v,%d) = %v, want %v", c.lo, c.hi, c.count, got, c.want)
		}
		for i := range got {
			if !approxEq(got[i], c.want[i]) {
				t.Fatalf("chartNiceTicks(%v,%v,%d)[%d] = %v, want %v (all %v)",
					c.lo, c.hi, c.count, i, got[i], c.want[i], got)
			}
		}
	}
}

func TestChartNiceTicksDegenerate(t *testing.T) {
	// hi == lo 不能死循环/返回单点
	got := chartNiceTicks(5, 5, 5)
	if len(got) < 2 {
		t.Fatalf("退化范围应至少返回两个刻度, got %v", got)
	}
	// 极端跨度受 n<=60 兜底, 退化成均分
	got = chartNiceTicks(0, 1e9, 5)
	if len(got) < 2 {
		t.Fatalf("极大跨度应回退均分, got %v", got)
	}
}

func TestChartNiceNum(t *testing.T) {
	cases := map[float64]float64{
		0.9:    1,
		1.1:    2,
		2.4:    2.5,
		4.9:    5,
		6:      10,
		0.011:  0.02,
		250:    250,
		1234.0: 2000,
	}
	for in, want := range cases {
		if got := chartNiceNum(in); !approxEq(got, want) {
			t.Fatalf("chartNiceNum(%v) = %v, want %v", in, got, want)
		}
	}
	if got := chartNiceNum(0); got != 1 {
		t.Fatalf("chartNiceNum(0) = %v, want 1", got)
	}
}

func TestChartFormatNumber(t *testing.T) {
	cases := map[float64]string{
		0:          "0",
		1234:       "1,234",
		1234.5:     "1,234.5",
		1000000:    "1,000,000",
		-0.25:      "-0.25",
		12.345:     "12.35", // 最多两位小数
		math.NaN(): "0",
	}
	for in, want := range cases {
		if got := chartFormatNumber(in); got != want {
			t.Fatalf("chartFormatNumber(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestChartTickText(t *testing.T) {
	cases := []struct {
		v, step float64
		want    string
	}{
		{0, 100, "0"},
		{1500, 500, "1,500"},
		{0.5, 0.5, "0.5"},
		{1, 0.5, "1.0"},
		{2.5, 2.5, "2.5"},
	}
	for _, c := range cases {
		if got := chartTickText(c.v, c.step); got != c.want {
			t.Fatalf("chartTickText(%v,%v) = %q, want %q", c.v, c.step, got, c.want)
		}
	}
}

func TestChartDomainAuto(t *testing.T) {
	c := &Chart{
		labels: []string{"a", "b", "c"},
		series: []ChartSeries{
			{Label: "s1", Values: []float64{10, 20, 30}},
			{Label: "s2", Values: []float64{5, 25, 15}},
		},
	}
	lo, hi, ticks := c.domain()
	if lo != 0 {
		t.Fatalf("全正数据基线应为 0, got %v", lo)
	}
	if hi < 30 {
		t.Fatalf("上界应覆盖最大值 30, got %v", hi)
	}
	if len(ticks) < 2 || !approxEq(ticks[0], lo) || !approxEq(ticks[len(ticks)-1], hi) {
		t.Fatalf("刻度首尾应与值域一致, got %v (lo=%v hi=%v)", ticks, lo, hi)
	}

	// 堆叠: 值域应按每个分类的合计算
	c.stacked = true
	_, hi2, _ := c.domain()
	if hi2 < 50 {
		t.Fatalf("堆叠上界应覆盖 30+15=45 的合计, got %v", hi2)
	}

	// 图例隐藏一路后, 值域跟着收窄
	c.hidden = []bool{false, true}
	_, hi3, _ := c.domain()
	if hi3 >= hi2 {
		t.Fatalf("隐藏一路后上界应收窄, got %v (之前 %v)", hi3, hi2)
	}
}

func TestChartDomainExplicit(t *testing.T) {
	c := &Chart{
		labels: []string{"a", "b"},
		series: []ChartSeries{{Label: "s", Values: []float64{10, 20}}},
		yMin:   0,
		yMax:   400,
	}
	lo, hi, ticks := c.domain()
	if lo != 0 || hi != 400 || len(ticks) != 5 {
		t.Fatalf("显式值域应原样使用并均分 5 档, got lo=%v hi=%v ticks=%v", lo, hi, ticks)
	}
	if ticks[2] != 200 {
		t.Fatalf("均分中点应为 200, got %v", ticks[2])
	}
}

func TestChartPieSlices(t *testing.T) {
	c := &Chart{
		kind:   ChartPie,
		labels: []string{"a", "b", "c"},
		series: []ChartSeries{{Label: "v", Values: []float64{50, 25, 25}}},
	}
	cats, start, sweep := c.pieSlices()
	if len(cats) != 3 || len(start) != 3 || len(sweep) != 3 {
		t.Fatalf("扇区数量应为 3, got cats=%v start=%v sweep=%v", cats, start, sweep)
	}
	if !approxEq(sweep[0], 180) || !approxEq(sweep[1], 90) || !approxEq(sweep[2], 90) {
		t.Fatalf("扇区角度应为 180/90/90, got %v", sweep)
	}
	if !approxEq(start[1], 180) || !approxEq(start[2], 270) {
		t.Fatalf("扇区起点应首尾相接, got %v", start)
	}

	// 隐藏一块后按剩余重新分配(50 与 25 → 240° / 120°)
	c.hidden = []bool{false, true, false}
	_, _, sweep2 := c.pieSlices()
	if len(sweep2) != 2 || !approxEq(sweep2[0], 240) || !approxEq(sweep2[1], 120) {
		t.Fatalf("隐藏一块后应按剩余重算(240/120), got %v", sweep2)
	}

	// 全 0 / 负数 = 无扇区
	c2 := &Chart{
		kind:   ChartPie,
		labels: []string{"a", "b"},
		series: []ChartSeries{{Values: []float64{0, -5}}},
	}
	if cats, _, _ := c2.pieSlices(); len(cats) != 0 {
		t.Fatalf("全 0/负数不应生成扇区, got %v", cats)
	}
}

func TestChartSmooth(t *testing.T) {
	pts := []xc.POINTF{{X: 0, Y: 0}, {X: 10, Y: 10}, {X: 20, Y: 0}}
	out := chartSmooth(pts, 12)
	if want := (len(pts)-1)*12 + 1; len(out) != want {
		t.Fatalf("采样点数 = %d, want %d", len(out), want)
	}
	if out[0] != pts[0] || out[len(out)-1] != pts[len(pts)-1] {
		t.Fatalf("首尾必须落在原数据点上, got %v .. %v", out[0], out[len(out)-1])
	}
	// x 必须单调递增(采样不会回绕), 否则填充多边形会自交
	for i := 1; i < len(out); i++ {
		if out[i].X < out[i-1].X-1e-4 {
			t.Fatalf("平滑后 x 应单调递增, 第 %d 点 %v < %v", i, out[i].X, out[i-1].X)
		}
	}
	// 少于 3 点不采样
	two := []xc.POINTF{{X: 0, Y: 0}, {X: 10, Y: 10}}
	if got := chartSmooth(two, 12); len(got) != 2 {
		t.Fatalf("两点应原样返回, got %v", got)
	}
}

func TestChartArcPoints(t *testing.T) {
	// 饼图: 12 点方向 0°, 顺时针 90° 结束在 3 点方向
	pts := chartArcPoints(100, 100, 50, 0, 0, 90)
	if len(pts) < 3 {
		t.Fatalf("扇区至少 3 个顶点, got %d", len(pts))
	}
	if pts[0].X != 100 || pts[0].Y != 100 {
		t.Fatalf("饼图扇区应从圆心起, got %v", pts[0])
	}
	if math.Abs(float64(pts[1].X)-100) > 0.01 || math.Abs(float64(pts[1].Y)-50) > 0.01 {
		t.Fatalf("0° 应指向 12 点方向 (100,50), got %v", pts[1])
	}
	if math.Abs(float64(pts[len(pts)-1].X)-150) > 0.01 || math.Abs(float64(pts[len(pts)-1].Y)-100) > 0.01 {
		t.Fatalf("90° 应指向 3 点方向 (150,100), got %v", pts[len(pts)-1])
	}

	// 扇环: 外弧 + 内弧(内弧倒序闭合)
	ring := chartArcPoints(100, 100, 50, 25, 0, 360)
	if len(ring) < 20 {
		t.Fatalf("整圈扇环顶点过多/过少都不对, got %d", len(ring))
	}
	for _, p := range ring {
		d := math.Hypot(float64(p.X)-100, float64(p.Y)-100)
		if math.Abs(d-50) > 0.5 && math.Abs(d-25) > 0.5 {
			t.Fatalf("扇环顶点必须落在外/内半径上, got %v (d=%.2f)", p, d)
		}
	}
}

func TestChartClampRatio(t *testing.T) {
	if got := chartClampRatio(-1); got != 0 {
		t.Fatalf("负比例应夹到 0, got %v", got)
	}
	if got := chartClampRatio(2); got != 0.9 {
		t.Fatalf("过大比例应夹到 0.9, got %v", got)
	}
	if got := chartClampRatio(0.5); got != 0.5 {
		t.Fatalf("合法比例应原样返回, got %v", got)
	}
}

func TestChartSetDataPadsAndTrims(t *testing.T) {
	c := &Chart{kind: ChartBar}
	c.setData([]string{"a", "b", "c"}, []ChartSeries{
		{Label: "短", Values: []float64{1}},
		{Values: []float64{1, 2, 3, 4}},
	})
	if len(c.series[0].Values) != 3 || c.series[0].Values[1] != 0 {
		t.Fatalf("短的 Values 应补 0 到分类数, got %v", c.series[0].Values)
	}
	if len(c.series[1].Values) != 3 || c.series[1].Values[2] != 3 {
		t.Fatalf("长的 Values 应截断到分类数, got %v", c.series[1].Values)
	}
	if c.series[1].Label != "系列 2" {
		t.Fatalf("空 Label 应自动命名, got %q", c.series[1].Label)
	}
}
