package ui

import "testing"

func TestTabsWrapGrid(t *testing.T) {
	cases := []struct {
		name               string
		inner, n           int32
		rows, perRow, tabW int32
	}{
		{"13 tabs at 882 (min window)", 882, 13, 2, 7, 126},
		{"13 tabs at 1122 (default window)", 1122, 13, 1, 13, 86},
		{"7 tabs fit one row", 882, 7, 1, 7, 126},
		{"exactly 3 x min width", 216, 3, 1, 3, 72},
		{"narrower than one min tab", 100, 13, 13, 1, 100},
		{"empty", 0, 13, 0, 0, 0},
		{"no labels", 882, 0, 0, 0, 0},
	}
	for _, c := range cases {
		rows, perRow, tabW := tabsWrapGrid(c.inner, c.n)
		if rows != c.rows || perRow != c.perRow || tabW != c.tabW {
			t.Errorf("%s: got rows=%d perRow=%d tabW=%d, want %d/%d/%d",
				c.name, rows, perRow, tabW, c.rows, c.perRow, c.tabW)
		}
	}
}
