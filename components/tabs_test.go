package ui

import "testing"

func TestTabsWrapGrid(t *testing.T) {
	cases := []struct {
		name               string
		inner, n, gap      int32
		rows, perRow, tabW int32
	}{
		{"13 tabs at 882 (min window)", 882, 13, 0, 2, 7, 126},
		{"13 tabs at 1122 (default window)", 1122, 13, 0, 1, 13, 86},
		{"7 tabs fit one row", 882, 7, 0, 1, 7, 126},
		{"exactly 3 x min width", 216, 3, 0, 1, 3, 72},
		{"narrower than one min tab", 100, 13, 0, 13, 1, 100},
		{"empty", 0, 13, 0, 0, 0, 0},
		{"no labels", 882, 0, 0, 0, 0, 0},
		// line 变体 gap-1: 每行标签之间留 4, 等分宽度要扣掉
		{"gap 4, 7 tabs one row", 882, 7, 4, 1, 7, 122},
	}
	for _, c := range cases {
		rows, perRow, tabW := tabsWrapGrid(c.inner, c.n, c.gap)
		if rows != c.rows || perRow != c.perRow || tabW != c.tabW {
			t.Errorf("%s: got rows=%d perRow=%d tabW=%d, want %d/%d/%d",
				c.name, rows, perRow, tabW, c.rows, c.perRow, c.tabW)
		}
	}
}

// TestTabsSeek 固定键盘导航的两条语义: 跳过禁用标签、到端点不循环。
func TestTabsSeek(t *testing.T) {
	// 3 号禁用: 0 1 2 [3] 4
	enabled := func(i int32) bool { return i != 3 }

	cases := []struct {
		name string
		from int32
		step int32
		want int32
	}{
		{"right skips disabled", 3, 1, 4},
		{"left skips disabled", 3, -1, 2},
		{"right at end stays (no loop)", 5, 1, -1},
		{"left at start stays (no loop)", -1, -1, -1},
		{"home from middle", 0, 1, 0},
		{"end from middle", 4, -1, 4},
	}
	for _, c := range cases {
		if got := tabsSeek(5, c.from, c.step, enabled); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
	if got := tabsSeek(0, 0, 1, enabled); got != -1 {
		t.Errorf("empty: got %d, want -1", got)
	}
	if got := tabsSeek(5, 0, 0, enabled); got != -1 {
		t.Errorf("zero step: got %d, want -1", got)
	}
}
