package ui

import (
	"testing"

	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

func TestRatingSnap(t *testing.T) {
	intRating := &Rating{max: 5}
	for _, c := range []struct {
		in, want float64
	}{{-1, 0}, {0.4, 0}, {0.6, 1}, {2.5, 3}, {4.4, 4}, {5.6, 5}} {
		if got := intRating.snap(c.in); got != c.want {
			t.Fatalf("int snap(%v) = %v, want %v", c.in, got, c.want)
		}
	}

	halfRating := &Rating{max: 5, half: true}
	for _, c := range []struct {
		in, want float64
	}{{-1, 0}, {0.2, 0}, {0.3, 0.5}, {2.4, 2.5}, {2.6, 2.5}, {2.8, 3}, {5.6, 5}} {
		if got := halfRating.snap(c.in); got != c.want {
			t.Fatalf("half snap(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRatingNaturalWidth(t *testing.T) {
	if got, want := ratingNaturalWidth(5), 2*ratingFocusPad+5*ratingStarSize+4*ratingStarGap; got != want {
		t.Fatalf("natural width = %d, want %d", got, want)
	}
	if got, want := ratingNaturalWidth(0), ratingNaturalWidth(5); got != want {
		t.Fatalf("natural width(0) = %d, want %d", got, want)
	}
}

// 星星区从左边留 3px 开始: 每颗星的左半 = i+0.5(半星), 右半/星间空隙 = i+1,
// 星星区左右留白与元素被 Fill 拉宽后的空区都返回 -1(不参与预览/点击)。
func TestRatingValueAt(t *testing.T) {
	area := ratingArea(xc.RECT{Left: 0, Top: 0, Right: 400, Bottom: 26}, 5)
	if area.Right != ratingNaturalWidth(5) || area.Bottom-area.Top != ratingStarSize+2*ratingFocusPad {
		t.Fatalf("area = %v, want natural size", area)
	}
	star0 := area.Left + ratingFocusPad
	cases := []struct {
		x    int32
		half bool
		want float64
	}{
		{0, false, -1},     // 左侧留白
		{star0, false, 1},  // 第一颗星左端
		{star0, true, 0.5}, // 半星模式左半
		{star0 + ratingStarSize/2 - 1, true, 0.5},
		{star0 + ratingStarSize/2, true, 1}, // 中线归右半
		{star0 + ratingStarSize - 1, false, 1},
		{star0 + ratingStarSize + 1, true, 1}, // 星间空隙算前一颗星的右端
		{star0 + ratingStarSize + ratingStarGap, true, 1.5},
		{star0 + 4*(ratingStarSize+ratingStarGap) + 1, false, 5},
		{area.Right - 1, false, -1}, // 右侧留白
		{400, false, -1},            // 元素被拉宽后的空区
	}
	for _, c := range cases {
		if got := ratingValueAt(area, c.x, 5, c.half); got != c.want {
			t.Fatalf("valueAt(%d, half=%v) = %v, want %v", c.x, c.half, got, c.want)
		}
	}
}

// 实心星/半星的 SVG 必须能被运行时解析(否则运行时什么都不画, 且不会报错)。
func TestRatingStarSVG(t *testing.T) {
	for name, svg := range map[string]string{
		"full": ratingStarFullSVG,
		"half": ratingStarHalfSVG,
	} {
		if paths, _, err := xui.ParseIconSVG(svg); err != nil || paths == "" {
			t.Fatalf("%s star svg: paths=%q err=%v", name, paths, err)
		}
	}
}
