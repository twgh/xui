package ui

import "testing"

func TestMenubarNextRow(t *testing.T) {
	v := &menubarView{rows: []mbRow{
		{it: &menuItem{kind: menuKindNormal}},
		{it: &menuItem{kind: menuKindSeparator}},
		{it: &menuItem{kind: menuKindNormal, disabled: true}},
		{it: &menuItem{kind: menuKindCheck, checked: true}},
		{it: &menuItem{kind: menuKindLabel, disabled: true}},
	}}
	cases := []struct {
		from, dir, want int
	}{
		{-1, 1, 0}, // 从头开始: 第一项
		{0, 1, 3},  // 跳过分隔线/禁用项
		{3, 1, 0},  // 循环
		{0, -1, 3}, // 反向循环
		{3, -1, 0}, // 反向跳过禁用/标签
		{1, 1, 3},  // 从分隔线继续向后
	}
	for _, c := range cases {
		if got := v.nextRow(c.from, c.dir); got != c.want {
			t.Fatalf("nextRow(%d, %d) = %d, want %d", c.from, c.dir, got, c.want)
		}
	}
}

func TestMenubarRowAt(t *testing.T) {
	v := &menubarView{rows: []mbRow{
		{it: &menuItem{kind: menuKindNormal}, y: 0, h: 32},
		{it: &menuItem{kind: menuKindSeparator}, y: 32, h: 9},
		{it: &menuItem{kind: menuKindNormal, disabled: true}, y: 41, h: 32},
		{it: &menuItem{kind: menuKindSub}, y: 73, h: 32},
	}}
	cases := []struct {
		y, want int32
	}{
		{0, 0},
		{31, 0},
		{32, -1}, // 分隔线不可交互
		{45, -1}, // 禁用项不可交互
		{75, 3},  // 子菜单触发项可交互
		{110, -1},
	}
	for _, c := range cases {
		if got := v.rowAt(c.y); int32(got) != c.want {
			t.Fatalf("rowAt(%d) = %d, want %d", c.y, got, c.want)
		}
	}
}
