package ui

import "testing"

// 步进器的值域/步进都是纯计算, 不需要 GUI, 直接测。
func TestDecimalsOf(t *testing.T) {
	cases := []struct {
		step float64
		want int
	}{
		{1, 0},
		{10, 0},
		{0.1, 1},
		{0.25, 2},
		{0.001, 3},
		{1e-9, 6}, // 超过 6 位按 6 位截断
	}
	for _, c := range cases {
		if got := decimalsOf(c.step); got != c.want {
			t.Errorf("decimalsOf(%v) = %d, want %d", c.step, got, c.want)
		}
	}
}

func TestInputStepSnapAndFormat(t *testing.T) {
	s := &inputStep{min: 0, max: 100, step: 10, decimals: decimalsOf(10)}

	// 吸附: 33 → 30, 36 → 40; 夹取: -5 → 0, 500 → 100
	for _, c := range []struct{ in, want float64 }{
		{33, 30}, {36, 40}, {-5, 0}, {500, 100}, {0, 0}, {100, 100},
	} {
		if got := s.snap(c.in); got != c.want {
			t.Errorf("snap(%v) = %v, want %v", c.in, got, c.want)
		}
	}
	if got := s.format(30); got != "30" {
		t.Errorf("format(30) = %q, want \"30\"", got)
	}

	// 小数步长不能写出 1.2000000000000002 这种尾巴
	f := &inputStep{min: -10, max: 10, step: 0.1, decimals: decimalsOf(0.1)}
	if got := f.format(f.snap(1.23)); got != "1.2" {
		t.Errorf("0.1 步长吸附后 = %q, want \"1.2\"", got)
	}
	if got := f.format(f.snap(-0.7000000001)); got != "-0.7" {
		t.Errorf("0.1 步长负数吸附后 = %q, want \"-0.7\"", got)
	}
}
