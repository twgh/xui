package ui

import "testing"

func TestRadioNextEnabled(t *testing.T) {
	enabled := []bool{true, false, true, true, false}
	cases := []struct {
		name    string
		enabled []bool
		from    int
		dir     int
		want    int
	}{
		{"next skips disabled", enabled, 0, 1, 2},
		{"next reaches last", enabled, 2, 1, 3},
		{"next wraps", enabled, 3, 1, 0},
		{"prev wraps", enabled, 0, -1, 3},
		{"home", enabled, -1, 1, 0},
		{"end", enabled, len(enabled), -1, 3},
		{"only current enabled", []bool{false, true, false}, 1, 1, 1},
		{"all disabled", []bool{false, false}, 0, 1, -1},
		{"empty", nil, 0, 1, -1},
		{"zero dir", enabled, 0, 0, -1},
	}
	for _, c := range cases {
		if got := radioNextEnabled(c.enabled, c.from, c.dir); got != c.want {
			t.Errorf("%s: radioNextEnabled(%v, %d, %d) = %d, want %d",
				c.name, c.enabled, c.from, c.dir, got, c.want)
		}
	}
}
