package ui

import "testing"

func TestTimelineSideAt(t *testing.T) {
	cases := []struct {
		name string
		o    TimelineOrientation
		m    TimelineMode
		idx  int
		want timelineSide
	}{
		{"vertical start", TimelineVertical, TimelineStart, 0, timelineRight},
		{"vertical end", TimelineVertical, TimelineEnd, 0, timelineLeft},
		{"vertical alternate first", TimelineVertical, TimelineAlternate, 0, timelineLeft},
		{"vertical alternate second", TimelineVertical, TimelineAlternate, 1, timelineRight},
		{"horizontal start", TimelineHorizontal, TimelineStart, 0, timelineBelow},
		{"horizontal end", TimelineHorizontal, TimelineEnd, 0, timelineAbove},
		{"horizontal alternate first", TimelineHorizontal, TimelineAlternate, 0, timelineAbove},
		{"horizontal alternate second", TimelineHorizontal, TimelineAlternate, 1, timelineBelow},
	}
	for _, c := range cases {
		if got := timelineSideAt(c.o, c.m, c.idx); got != c.want {
			t.Errorf("%s: got side %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTimelineVerticalPadding(t *testing.T) {
	const w int32 = 400
	const gap int32 = 24

	if l, tp, r, b := timelineVerticalPadding(w, TimelineStart, timelineRight, false, gap); l != timelineNodeCol || tp != 0 || r != 0 || b != gap {
		t.Fatalf("start: got (%d,%d,%d,%d)", l, tp, r, b)
	}
	if l, _, r, b := timelineVerticalPadding(w, TimelineEnd, timelineLeft, true, gap); l != 0 || r != timelineNodeCol || b != 0 {
		t.Fatalf("end/last: got (l=%d, r=%d, b=%d)", l, r, b)
	}

	// alternate: 轴线在正中, 内容边缘距轴线 axisGap; 左右两条内容宽度相等。
	ll, _, lr, _ := timelineVerticalPadding(w, TimelineAlternate, timelineLeft, false, gap)
	rl, _, rr, _ := timelineVerticalPadding(w, TimelineAlternate, timelineRight, false, gap)
	if ll != 0 || rr != 0 || lr != w/2+timelineAxisGap || rl != w/2+timelineAxisGap {
		t.Fatalf("alternate padding: left=(%d,%d), right=(%d,%d)", ll, lr, rl, rr)
	}
	if w-ll-lr != w-rl-rr {
		t.Fatalf("alternate content widths differ: %d vs %d", w-ll-lr, w-rl-rr)
	}
}

func TestTimelineHorizontalPadding(t *testing.T) {
	const maxAbove int32 = 80
	const maxBelow int32 = 40
	const gap = timelineAxisGap

	cases := []struct {
		mode       TimelineMode
		side       timelineSide
		contentH   int32
		wantTop    int32
		wantBottom int32
	}{
		{TimelineStart, timelineBelow, 40, 32, 0},
		{TimelineStart, timelineBelow, 16, 32, 24},
		{TimelineEnd, timelineAbove, 80, 0, 32},
		{TimelineEnd, timelineAbove, 40, 40, 32},
		{TimelineAlternate, timelineAbove, 80, 0, 80},
		{TimelineAlternate, timelineAbove, 40, 40, 80},
		{TimelineAlternate, timelineBelow, 40, 120, 0},
		{TimelineAlternate, timelineBelow, 16, 120, 24},
	}
	for i, c := range cases {
		axisY, total := timelineHorizontalGeometry(maxAbove, maxBelow, c.mode)
		top, bottom := timelineHorizontalPadding(c.contentH, maxAbove, maxBelow, c.mode, c.side)
		if top != c.wantTop || bottom != c.wantBottom {
			t.Errorf("case %d: got (%d,%d), want (%d,%d)", i, top, bottom, c.wantTop, c.wantBottom)
		}
		if top+c.contentH+bottom != total {
			t.Errorf("case %d: sum %d, want total %d", i, top+c.contentH+bottom, total)
		}
		switch c.side {
		case timelineAbove:
			if top+c.contentH != axisY-gap {
				t.Errorf("case %d: content bottom %d, want %d", i, top+c.contentH, axisY-gap)
			}
		case timelineBelow:
			if top != axisY+gap {
				t.Errorf("case %d: content top %d, want %d", i, top, axisY+gap)
			}
		}
	}
}
