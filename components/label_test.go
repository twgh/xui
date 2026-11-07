package ui

import "testing"

func TestLabelRoundWrapHeight(t *testing.T) {
	cases := []struct {
		name  string
		h     int32
		lineH int32
		want  int32
	}{
		{"空文本", 0, 20, 0},
		{"单行", 20, 20, 20},
		{"两行补末行空间", 39, 20, 44},
		{"三行补末行空间", 59, 20, 64},
		{"小字号单行", 16, 16, 16},
		{"小字号两行", 32, 16, 36},
		{"大字号单行", 22, 24, 24},
		{"大字号两行", 43, 24, 52},
		{"非法行高", 20, 0, 0},
	}
	for _, c := range cases {
		if got := labelRoundWrapHeight(c.h, c.lineH); got != c.want {
			t.Errorf("%s: labelRoundWrapHeight(%d, %d) = %d, want %d", c.name, c.h, c.lineH, got, c.want)
		}
	}
}
