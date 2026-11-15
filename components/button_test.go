package ui

import "testing"

// Height 是固定高行留按压空间的依据(行高 = Height() + 1), 数值必须与预设一致。
func TestButtonSizeHeight(t *testing.T) {
	cases := []struct {
		size ButtonSize
		want int32
	}{
		{ButtonSizeXs, 24},
		{ButtonSizeSm, 28},
		{ButtonSizeDefault, 32},
		{ButtonSizeLg, 36},
		{ButtonSizeIcon, 32},
		{ButtonSizeIconXs, 24},
		{ButtonSizeIconSm, 28},
		{ButtonSizeIconLg, 36},
		{ButtonSize(99), 32}, // 未知值按 Default
	}
	for _, c := range cases {
		if got := c.size.Height(); got != c.want {
			t.Errorf("ButtonSize(%d).Height() = %d, want %d", c.size, got, c.want)
		}
	}
}