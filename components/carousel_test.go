package ui

import (
	"testing"

	"github.com/twgh/xcgui/xc"
)

func TestCarouselSlideOffset(t *testing.T) {
	const step = 840.0
	// 非循环: 第 i 页在 i×step 处, 拖到负偏移时保持负坐标(没有折回)。
	got := []float64{
		carouselSlideOffset(0, 0, step, 3, false),
		carouselSlideOffset(1, 0, step, 3, false),
		carouselSlideOffset(2, 0, step, 3, false),
	}
	want := []float64{0, 840, 1680}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("non-loop slide%d = %v, want %v", i, got[i], want[i])
		}
	}
	if v := carouselSlideOffset(2, -120, step, 3, false); v != 1800 {
		t.Fatalf("non-loop drag past start = %v, want 1800", v)
	}

	// 循环: index 0 时上一张(2)折到左边 -step, 下一页(1)在右边 step。
	got = []float64{
		carouselSlideOffset(0, 0, step, 3, true),
		carouselSlideOffset(1, 0, step, 3, true),
		carouselSlideOffset(2, 0, step, 3, true),
	}
	want = []float64{0, 840, -840}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("loop slide%d at 0 = %v, want %v", i, got[i], want[i])
		}
	}

	// 循环 index 2: 第一张折到右边 step, 上一张(1)在左边 -step。
	got = []float64{
		carouselSlideOffset(0, 2*step, step, 3, true),
		carouselSlideOffset(1, 2*step, step, 3, true),
		carouselSlideOffset(2, 2*step, step, 3, true),
	}
	want = []float64{840, -840, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("loop slide%d at 2 = %v, want %v", i, got[i], want[i])
		}
	}

	// 从末页往下翻的一半: 第一张从右侧进入, 仍在 (0, step) 之间。
	v := carouselSlideOffset(0, 2.5*step, step, 3, true)
	if v <= 0 || v >= step {
		t.Fatalf("loop wrap-in = %v, want 0<v<step", v)
	}
	// 拖过末页: 折回后仍是一组合法可见位置(没有蹦到负无限)。
	v = carouselSlideOffset(0, 3.25*step, step, 3, true)
	if v < -step || v > step {
		t.Fatalf("loop past end = %v, want within [-step, step]", v)
	}
}

func TestCarouselDotsOrigin(t *testing.T) {
	rc := xc.RECT{Left: 0, Right: 200}
	// 5 点 6px + 4 间隙 6px = 54, 居中后左缘 = 73。
	if got := carouselDotsOrigin(rc, 5, 6, 6); got != 73 {
		t.Fatalf("origin(5 dots) = %d, want 73", got)
	}
	// 单点: 只居中一个点。
	if got := carouselDotsOrigin(rc, 1, 6, 6); got != 97 {
		t.Fatalf("origin(1 dot) = %d, want 97", got)
	}
	// 非零左边距同样按整行居中。
	rc = xc.RECT{Left: 10, Right: 210}
	if got := carouselDotsOrigin(rc, 5, 6, 6); got != 83 {
		t.Fatalf("origin(offset rc) = %d, want 83", got)
	}
	if got := carouselDotsOrigin(rc, 0, 6, 6); got != 110 {
		t.Fatalf("origin(no dots) = %d, want center 110", got)
	}
}
