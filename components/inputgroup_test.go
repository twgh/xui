package ui

import "testing"

func TestInputGroupAddonWidth(t *testing.T) {
	cases := []struct {
		name                        string
		contentW, padL, padR, items int32
		want                        int32
	}{
		{"单个图标 + 内边距", 16, 0, 8, 1, 24},
		{"两个控件加一组间距", 24 + 24, 0, 8, 2, 64},
		{"前后都有内边距", 24, 8, 8, 1, 40},
		{"空内容兜底 1", 0, 0, 0, 0, 1},
	}
	for _, c := range cases {
		if got := inputGroupAddonWidth(c.contentW, c.padL, c.padR, c.items); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestInputGroupAddonPull(t *testing.T) {
	cases := []struct {
		hasBtn, hasKbd bool
		want           int32
	}{
		{false, false, 0},
		{true, false, inputGroupBtnPull},
		{false, true, inputGroupKbdPull},
		{true, true, inputGroupBtnPull}, // 按钮优先(外扩量更大)
	}
	for _, c := range cases {
		if got := inputGroupAddonPull(c.hasBtn, c.hasKbd); got != c.want {
			t.Errorf("pull(btn=%v, kbd=%v) = %d, want %d", c.hasBtn, c.hasKbd, got, c.want)
		}
	}
}
