package ui

import (
	"testing"

	"github.com/twgh/xui"
)

// Progress 的纯数值逻辑(百分比 / max 换算), 绘制与动画由探针程序验证。

func TestProgressPercent(t *testing.T) {
	p := &Progress{value: 33, max: 100}
	if got := p.Percent(); got != 33 {
		t.Fatalf("Percent = %v, want 33", got)
	}
	p2 := &Progress{value: 1, max: 4}
	if got := p2.Percent(); got != 25 {
		t.Fatalf("Percent(1/4) = %v, want 25", got)
	}
	p3 := &Progress{value: 3, max: 4, indeterminate: true}
	if got := p3.Percent(); got != 0 {
		t.Fatalf("indeterminate Percent = %v, want 0", got)
	}
	p4 := &Progress{value: 5, max: 0}
	if got := p4.Percent(); got != 0 {
		t.Fatalf("zero max Percent = %v, want 0", got)
	}
}

func TestProgressIndeterminate(t *testing.T) {
	p := &Progress{max: 100}
	if p.Indeterminate() {
		t.Fatal("new progress must not be indeterminate")
	}
}

func TestProgressMin(t *testing.T) {
	// 负值域: 0 落在中间, 百分比按 (v-min)/(max-min) 算
	p := &Progress{value: 0, min: -50, max: 50}
	if got := p.Percent(); got != 50 {
		t.Fatalf("Percent(-50..50, 0) = %v, want 50", got)
	}
	if got := p.Min(); got != -50 {
		t.Fatalf("Min = %v, want -50", got)
	}
	p.value = -50
	if got := p.Percent(); got != 0 {
		t.Fatalf("下限百分比 = %v, want 0", got)
	}
	p.value = 50
	if got := p.Percent(); got != 100 {
		t.Fatalf("上限百分比 = %v, want 100", got)
	}
	// 值域塌成 0 时不除零
	z := &Progress{value: 5, min: 5, max: 5}
	if got := z.Percent(); got != 0 {
		t.Fatalf("空值域 Percent = %v, want 0", got)
	}
}

func TestProgressStyleOptions(t *testing.T) {
	p := &Progress{}
	WithProgressRadius(3)(p)
	if p.radius != 3 {
		t.Fatalf("radius = %d, want 3", p.radius)
	}
	WithProgressTrackColor(xui.Hex("#e5e7eb"))(p)
	if p.track == nil {
		t.Fatal("TrackColor 未设置解析器")
	}
	WithProgressRadius(0)(p) // <=0 不覆盖
	if p.radius != 3 {
		t.Fatalf("radius 被 0 覆盖成 %d", p.radius)
	}
}
