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
	withDesc := stepsItems(3, true)
	noDesc := stepsItems(3, false)
	if got, want := stepsNaturalHeight(withDesc, true), 2*stepsFocusPad+3*stepsRowH(withDesc)+2*stepsRowGap; got != want {
		t.Fatalf("vertical with desc = %d, want %d", got, want)
	}
	if got, want := stepsNaturalHeight(noDesc, true), 2*stepsFocusPad+3*stepsCircle+2*stepsRowGap; got != want {
		t.Fatalf("vertical no desc = %d, want %d", got, want)
	}
	if got, want := stepsNaturalHeight(withDesc, false), stepsCircle+stepsHGap+stepsTitleH+stepsDescGap+stepsDescH+2*stepsFocusPad; got != want {
		t.Fatalf("horizontal with desc = %d, want %d", got, want)
	}
	if got, want := stepsNaturalHeight(noDesc, false), stepsCircle+stepsHGap+stepsTitleH+2*stepsFocusPad; got != want {
		t.Fatalf("horizontal no desc = %d, want %d", got, want)
	}
}

// 横向: 列等宽、圆点居中、文字不越出本列; 圆点直径固定。
func TestStepsLayoutHorizontal(t *testing.T) {
	origin := xc.RECT{Left: 0, Top: 0, Right: 400, Bottom: 76}
	boxes := stepsLayout(origin, stepsItems(4, true), false)
	if len(boxes) != 4 {
		t.Fatalf("boxes = %d, want 4", len(boxes))
	}
	for i, b := range boxes {
		wantL := int32(i * 100)
		wantR := int32((i + 1) * 100)
		if b.cell.Left != wantL || b.cell.Right != wantR {
			t.Fatalf("box %d cell x = [%d,%d], want [%d,%d]", i, b.cell.Left, b.cell.Right, wantL, wantR)
		}
		if got := b.circle.Right - b.circle.Left; got != stepsCircle {
			t.Fatalf("box %d circle width = %d, want %d", i, got, stepsCircle)
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
	items := stepsItems(3, true)
	rowH := stepsRowH(items)
	natural := stepsNaturalHeight(items, true)
	boxes := stepsLayout(xc.RECT{Left: 0, Top: 0, Right: 300, Bottom: natural}, items, true)
	if len(boxes) != 3 {
		t.Fatalf("boxes = %d, want 3", len(boxes))
	}
	if boxes[0].circle.Top != stepsFocusPad {
		t.Fatalf("first circle top = %d, want %d", boxes[0].circle.Top, stepsFocusPad)
	}
	if boxes[2].desc.Bottom+stepsFocusPad != natural {
		t.Fatalf("last row bottom + pad = %d, want %d", boxes[2].desc.Bottom+stepsFocusPad, natural)
	}
	if boxes[1].circle.Top-boxes[0].circle.Top != rowH+stepsRowGap {
		t.Fatalf("row stride = %d, want %d", boxes[1].circle.Top-boxes[0].circle.Top, rowH+stepsRowGap)
	}
	if boxes[0].title.Left != stepsFocusPad+stepsTextLeft {
		t.Fatalf("text left = %d, want %d", boxes[0].title.Left, stepsFocusPad+stepsTextLeft)
	}

	// 拉高后余量进行距: 首行位置不变, 末行贴底。
	tall := xc.RECT{Left: 0, Top: 0, Right: 300, Bottom: natural + 56}
	boxes = stepsLayout(tall, items, true)
	if boxes[0].circle.Top != stepsFocusPad {
		t.Fatalf("tall first circle top = %d, want %d", boxes[0].circle.Top, stepsFocusPad)
	}
	if boxes[2].desc.Bottom+stepsFocusPad != tall.Bottom {
		t.Fatalf("tall last bottom = %d, want %d", boxes[2].desc.Bottom+stepsFocusPad, tall.Bottom)
	}
}

// 命中测试: 只有已完成步(< current)可点, 当前与未达步一律 -1。
func TestStepsHitOnlyCompleted(t *testing.T) {
	boxes := stepsLayout(xc.RECT{Left: 0, Top: 0, Right: 400, Bottom: 76}, stepsItems(4, true), false)
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
		if got := stepsHitIndex(boxes, c.x, c.y, c.current); got != c.want {
			t.Fatalf("hit(%d,%d) current=%d = %d, want %d", c.x, c.y, c.current, got, c.want)
		}
	}
}
