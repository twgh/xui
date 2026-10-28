package ui

import "testing"

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
