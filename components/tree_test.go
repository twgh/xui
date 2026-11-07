package ui

import "testing"

func TestCombineTreeCheck(t *testing.T) {
	cases := []struct {
		name   string
		states []int8
		want   int8
	}{
		{"没有子项", nil, treeCheckNone},
		{"全部未选", []int8{treeCheckNone, treeCheckNone}, treeCheckNone},
		{"全部勾选", []int8{treeCheckChecked, treeCheckChecked}, treeCheckChecked},
		{"部分勾选", []int8{treeCheckChecked, treeCheckNone}, treeCheckPartial},
		{"有子项半选", []int8{treeCheckPartial, treeCheckNone}, treeCheckPartial},
		{"混有半选与勾选", []int8{treeCheckChecked, treeCheckPartial}, treeCheckPartial},
		{"单个叶子未选", []int8{treeCheckNone}, treeCheckNone},
		{"单个叶子勾选", []int8{treeCheckChecked}, treeCheckChecked},
	}
	for _, c := range cases {
		if got := combineTreeCheck(c.states); got != c.want {
			t.Errorf("%s: combineTreeCheck(%v) = %d, want %d", c.name, c.states, got, c.want)
		}
	}
}

func TestTreeCheckHitGeometry(t *testing.T) {
	// 复选框左缘 = 展开按钮内缩(1) + 层级缩进 + chevron 槽(16) + 间距(4)
	for _, depth := range []int32{0, 1, 3} {
		left := treeCheckLeft(depth)
		if want := treeBtnInset + depth*treeIndent + treeChevronSlot + treeCheckGap; left != want {
			t.Fatalf("treeCheckLeft(%d) = %d, want %d", depth, left, want)
		}
		if !treeCheckHit(depth, left) || !treeCheckHit(depth, left+treeCheckSize-1) {
			t.Fatalf("depth %d: 复选框自身范围应命中", depth)
		}
		// 左侧不能吃掉原生展开按钮的点击热区(按钮右缘 = left - treeCheckGap)
		chevronRight := left - treeCheckGap
		if treeCheckHit(depth, chevronRight) {
			t.Fatalf("depth %d: 命中区不能盖住 chevron 热区", depth)
		}
		// 右侧不能吃掉文字区(文字从 left + size + gap 开始)
		if treeCheckHit(depth, left+treeCheckSize+treeCheckGap) {
			t.Fatalf("depth %d: 命中区不能盖住文字", depth)
		}
	}
}
