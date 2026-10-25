package ui

import (
	"testing"

	"github.com/twgh/xcgui/xc"
)

func TestSheetPanelWidth(t *testing.T) {
	cases := []struct {
		body, want, got int32
	}{
		{936, 0, 384},   // 宽屏: 3/4 远超 max-w-sm, 收在 384
		{400, 0, 300},   // 窄窗: 3/4 = 300, 不到 384
		{400, 500, 400}, // 显式宽超过内容区: 夹到内容区宽
		{400, 200, 200},
		{0, 0, 1}, // 退化宽至少 1, 避免零宽面板
	}
	for _, c := range cases {
		if got := sheetPanelWidth(c.body, c.want); got != c.got {
			t.Fatalf("sheetPanelWidth(%d, %d) = %d, want %d", c.body, c.want, got, c.got)
		}
	}
}

func TestSheetSlideStart(t *testing.T) {
	d := &Dialog{}
	d.panel = xc.RECT{Left: 100, Top: 50, Right: 484, Bottom: 810} // 384×760
	cases := []struct {
		side   SheetSide
		dx, dy int32
	}{
		{SheetRight, 384, 0},
		{SheetLeft, -384, 0},
		{SheetTop, 0, -760},
		{SheetBottom, 0, 760},
	}
	for _, c := range cases {
		d.sheetSide = c.side
		dx, dy := d.sheetSlideStart()
		if dx != c.dx || dy != c.dy {
			t.Fatalf("side %d: got (%d,%d), want (%d,%d)", c.side, dx, dy, c.dx, c.dy)
		}
	}
}
