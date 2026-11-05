package ui

import (
	"testing"

	"github.com/twgh/xcgui/xc"
)

func stepsItems(n int, desc bool) []StepItem {
	out := make([]StepItem, n)
	for i := range out {
		out[i].Title = "步骤"
		if desc {
			out[i].Description = "说明"
		}
	}
	return out
}

func TestStepsNaturalHeight(t *testing.T) {
	m := stepsMetricsFor(StepsMedium)
	withDesc := stepsItems(3, true)
	noDesc := stepsItems(3, false)
	if got, want := stepsNaturalHeight(withDesc, true, m), 2*m.focusPad+3*stepsRowH(withDesc, m)+2*m.rowGap; got != want {
		t.Fatalf("vertical with desc = %d, want %d", got, want)
	}
	if got, want := stepsNaturalHeight(noDesc, true, m), 2*m.focusPad+3*m.circle+2*m.rowGap; got != want {
		t.Fatalf("vertical no desc = %d, want %d", got, want)
	}
	if got, want := stepsNaturalHeight(withDesc, false, m), m.circle+m.hGap+m.titleH+m.descGap+m.descH+2*m.focusPad; got != want {
		t.Fatalf("horizontal with desc = %d, want %d", got, want)
	}
	if got, want := stepsNaturalHeight(noDesc, false, m), m.circle+m.hGap+m.titleH+2*m.focusPad; got != want {
		t.Fatalf("horizontal no desc = %d, want %d", got, want)
	}

	small := stepsMetricsFor(StepsSmall)
	if got, want := stepsNaturalHeight(noDesc, false, small), small.circle+small.hGap+small.titleH+2*small.focusPad; got != want {
		t.Fatalf("small horizontal no desc = %d, want %d", got, want)
	}
	if small.circle >= m.circle || small.titleH >= m.titleH {
		t.Fatalf("small metrics not smaller: circle %d/%d title %d/%d", small.circle, m.circle, small.titleH, m.titleH)
	}
}

// 横向: 列等宽、圆点居中、文字不越出本列; 圆点直径固定。
func TestStepsLayoutHorizontal(t *testing.T) {
	m := stepsMetricsFor(StepsMedium)
	origin := xc.RECT{Left: 0, Top: 0, Right: 400, Bottom: 76}
	boxes := stepsLayout(origin, stepsItems(4, true), false, m)
	if len(boxes) != 4 {
		t.Fatalf("boxes = %d, want 4", len(boxes))
	}
	for i, b := range boxes {
		wantL := int32(i * 100)
		wantR := int32((i + 1) * 100)
		if b.cell.Left != wantL || b.cell.Right != wantR {
			t.Fatalf("box %d cell x = [%d,%d], want [%d,%d]", i, b.cell.Left, b.cell.Right, wantL, wantR)
		}
		if got := b.circle.Right - b.circle.Left; got != m.circle {
			t.Fatalf("box %d circle width = %d, want %d", i, got, m.circle)
		}
		if b.circle.Left < b.cell.Left || b.circle.Right > b.cell.Right {
			t.Fatalf("box %d circle %v out of cell %v", i, b.circle, b.cell)
		}
		if b.title.Left < b.cell.Left || b.title.Right > b.cell.Right {
			t.Fatalf("box %d title %v out of cell %v", i, b.title, b.cell)
		}
		if b.desc.Top < b.title.Bottom {
			t.Fatalf("box %d desc overlaps title", i)
		}
	}
}

// 竖向: 元素高于自然高度时把余量均摊到行距, 最后一行仍贴底; 文字起点固定。
func TestStepsLayoutVertical(t *testing.T) {
	m := stepsMetricsFor(StepsMedium)
	items := stepsItems(3, true)
	rowH := stepsRowH(items, m)
	natural := stepsNaturalHeight(items, true, m)
	boxes := stepsLayout(xc.RECT{Left: 0, Top: 0, Right: 300, Bottom: natural}, items, true, m)
	if len(boxes) != 3 {
		t.Fatalf("boxes = %d, want 3", len(boxes))
	}
	if boxes[0].circle.Top != m.focusPad {
		t.Fatalf("first circle top = %d, want %d", boxes[0].circle.Top, m.focusPad)
	}
	if boxes[2].desc.Bottom+m.focusPad != natural {
		t.Fatalf("last row bottom + pad = %d, want %d", boxes[2].desc.Bottom+m.focusPad, natural)
	}
	if boxes[1].circle.Top-boxes[0].circle.Top != rowH+m.rowGap {
		t.Fatalf("row stride = %d, want %d", boxes[1].circle.Top-boxes[0].circle.Top, rowH+m.rowGap)
	}
	if boxes[0].title.Left != m.focusPad+m.textLeft {
		t.Fatalf("text left = %d, want %d", boxes[0].title.Left, m.focusPad+m.textLeft)
	}

	// 拉高后余量进行距: 首行位置不变, 末行贴底。
	tall := xc.RECT{Left: 0, Top: 0, Right: 300, Bottom: natural + 56}
	boxes = stepsLayout(tall, items, true, m)
	if boxes[0].circle.Top != m.focusPad {
		t.Fatalf("tall first circle top = %d, want %d", boxes[0].circle.Top, m.focusPad)
	}
	if boxes[2].desc.Bottom+m.focusPad != tall.Bottom {
		t.Fatalf("tall last bottom = %d, want %d", boxes[2].desc.Bottom+m.focusPad, tall.Bottom)
	}
}

// 命中测试: 默认只允许点回已完成步(< current), 当前/未达步一律 -1。
func TestStepsHitOnlyCompleted(t *testing.T) {
	m := stepsMetricsFor(StepsMedium)
	boxes := stepsLayout(xc.RECT{Left: 0, Top: 0, Right: 400, Bottom: 76}, stepsItems(4, true), false, m)
	back := func(current int) func(int) bool {
		return func(i int) bool { return i >= 0 && i < current }
	}
	cases := []struct {
		x, y    int32
		current int
		want    int
	}{
		{50, 30, 2, 0},   // 已完成
		{150, 30, 2, 1},  // 已完成
		{250, 30, 2, -1}, // 当前步不可点
		{350, 30, 2, -1}, // 未达步不可点
		{50, 30, 0, -1},  // 没有已完成步
		{150, 30, 1, -1}, // 只有第 0 步完成时, 第 1 步仍是当前
		{999, 30, 2, -1}, // 元素外
	}
	for _, c := range cases {
		if got := stepsHitIndex(boxes, c.x, c.y, back(c.current)); got != c.want {
			t.Fatalf("hit(%d,%d) current=%d = %d, want %d", c.x, c.y, c.current, got, c.want)
		}
	}
}

// 命中测试: 可点全部后, 任意非当前步都能命中(未达步也可)。
func TestStepsHitClickable(t *testing.T) {
	m := stepsMetricsFor(StepsMedium)
	boxes := stepsLayout(xc.RECT{Left: 0, Top: 0, Right: 400, Bottom: 76}, stepsItems(4, true), false, m)
	clickable := func(i int) bool { return i != 2 }
	if got := stepsHitIndex(boxes, 350, 30, clickable); got != 3 {
		t.Fatalf("hit ahead step = %d, want 3", got)
	}
	if got := stepsHitIndex(boxes, 250, 30, clickable); got != -1 {
		t.Fatalf("hit current step = %d, want -1", got)
	}
}

// testSteps 直接构造 Steps(不建元素)并套用配置项, 供不依赖窗口的逻辑测试使用。
func testSteps(items []StepItem, opts ...StepsOption) *Steps {
	s := &Steps{items: items, curStatus: StepStatusProcess}
	for _, o := range opts {
		o(s)
	}
	s.clampCurrent()
	return s
}

func TestStepsStatusAt(t *testing.T) {
	s := testSteps(stepsItems(4, false), WithStepsCurrent(2))
	cases := []struct {
		i    int
		want StepStatus
	}{
		{0, StepStatusFinish},
		{1, StepStatusFinish},
		{2, StepStatusProcess},
		{3, StepStatusWait},
	}
	for _, c := range cases {
		if got := s.statusAt(c.i); got != c.want {
			t.Fatalf("statusAt(%d) = %d, want %d", c.i, got, c.want)
		}
	}

	// 单步覆盖优先(错误步 / 手动完成)。
	s.items[1].Status = StepStatusError
	s.items[3].Status = StepStatusFinish
	if got := s.statusAt(1); got != StepStatusError {
		t.Fatalf("item error override = %d, want error", got)
	}
	if got := s.statusAt(3); got != StepStatusFinish {
		t.Fatalf("item finish override = %d, want finish", got)
	}

	// Steps 级默认状态只作用于当前步。
	s2 := testSteps(stepsItems(3, false), WithStepsCurrent(1), WithStepsStatus(StepStatusError))
	if got := s2.statusAt(0); got != StepStatusFinish {
		t.Fatalf("steps-level override leaked to finished step: %d", got)
	}
	if got := s2.statusAt(1); got != StepStatusError {
		t.Fatalf("steps status at current = %d, want error", got)
	}
	if got := s2.statusAt(2); got != StepStatusWait {
		t.Fatalf("steps status leaked to wait step: %d", got)
	}

	// 越界返回未达, 不 panic。
	if got := s.statusAt(-1); got != StepStatusWait {
		t.Fatalf("statusAt(-1) = %d, want wait", got)
	}
	if got := s.statusAt(99); got != StepStatusWait {
		t.Fatalf("statusAt(99) = %d, want wait", got)
	}
}

func TestStepsCanActivate(t *testing.T) {
	items := stepsItems(4, false)
	items[2].Disabled = true
	s := testSteps(items, WithStepsCurrent(3))
	// 默认: 只可点回已完成步; 当前步与禁用的已完成步不可点。
	want := []bool{true, true, false, false}
	for i, w := range want {
		if got := s.canActivate(i); got != w {
			t.Fatalf("back-only canActivate(%d) = %v, want %v", i, got, w)
		}
	}

	// 可点全部: 未达步也可, 但仍排除当前步与禁用步。
	all := testSteps(items, WithStepsCurrent(3), WithStepsClickable())
	for i, w := range want {
		if got := all.canActivate(i); got != w {
			t.Fatalf("clickable canActivate(%d) = %v, want %v", i, got, w)
		}
	}
	if got := all.canActivate(-1); got {
		t.Fatalf("canActivate(-1) = true, want false")
	}
	if got := all.canActivate(99); got {
		t.Fatalf("canActivate(99) = true, want false")
	}

	// 禁用只挡点击, 不改变显示状态。
	if got := all.statusAt(2); got != StepStatusFinish {
		t.Fatalf("disabled step status = %d, want finish", got)
	}
}
