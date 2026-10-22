package ui

import "testing"

func TestAspectRatioHeight(t *testing.T) {
	cases := []struct {
		name  string
		width int32
		ratio float64
		want  int32
	}{
		{"16:9", 320, 16.0 / 9.0, 180},
		{"1:1", 200, 1, 200},
		{"9:16", 180, 9.0 / 16.0, 320},
		{"round half up", 100, 3, 33},
		{"zero width", 0, 2, 0},
		{"tiny height clamps to 1", 10, 100, 1},
	}
	for _, c := range cases {
		if got := aspectRatioHeight(c.width, c.ratio); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestNormalizeAspectRatio(t *testing.T) {
	for _, bad := range []float64{0, -1, -0.5} {
		if got := normalizeAspectRatio(bad); got != 1 {
			t.Errorf("normalize(%v) = %v, want 1", bad, got)
		}
	}
	if got := normalizeAspectRatio(16.0 / 9.0); got != 16.0/9.0 {
		t.Errorf("valid ratio changed: %v", got)
	}
}
