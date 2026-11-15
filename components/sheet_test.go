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

func TestSheetGeometryMatchesBase(t *testing.T) {
	// base-nova 的 sheet.tsx: 关闭钮 top-3 right-3 + icon-sm(size-7);
	// SheetHeader 是 gap-0.5(2), 不是旧版的 gap-1.5(6)。
	if sheetCloseOff != 12 || sheetCloseSize != 28 {
		t.Fatalf("关闭钮几何 = (%d,%d), want (12,28)", sheetCloseOff, sheetCloseSize)
	}
	if sheetHeadGap != 2 {
		t.Fatalf("SheetHeader 间距 = %d, want 2", sheetHeadGap)
	}
}

func TestSheetSlideStart(t *testing.T) {
	d := &Dialog{}
	d.panel = xc.RECT{Left: 100, Top: 50, Right: 484, Bottom: 810} // 384×760
	// 默认 = base-nova 的 translate-x-[2.5rem](40px) 小位移 + 淡入, 不再整块推入。
	cases := []struct {
		side   SheetSide
		dx, dy int32
	}{
		{SheetRight, sheetSlideDist, 0},
		{SheetLeft, -sheetSlideDist, 0},
		{SheetTop, 0, -sheetSlideDist},
		{SheetBottom, 0, sheetSlideDist},
	}
	for _, c := range cases {
		d.sheetSide = c.side
		dx, dy := d.sheetSlideStart()
		if dx != c.dx || dy != c.dy {
			t.Fatalf("side %d: got (%d,%d), want (%d,%d)", c.side, dx, dy, c.dx, c.dy)
		}
	}
	// WithSheetSlideDistance 显式给大位移时恢复"整块从屏幕外滑入"的旧观感。
	d.slideDist = 384
	d.sheetSide = SheetRight
	if dx, _ := d.sheetSlideStart(); dx != 384 {
		t.Fatalf("slideDist 覆盖: got %d, want 384", dx)
	}
}
