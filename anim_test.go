package xui

import (
	"math"
	"testing"
)

// TestEaseTransitionCurve 校验 hover 过渡的缓动曲线就是 tailwind / shadcn 的
// cubic-bezier(0.4, 0, 0.2, 1)。参考值由独立实现(直接按贝塞尔公式算, 见
// DESIGN.md 第八节 45)得出, 容差 1e-3。
func TestEaseTransitionCurve(t *testing.T) {
	cases := []struct{ x, y float64 }{
		{0, 0},
		{0.1, 0.0259},
		{0.25, 0.2366},
		{0.5, 0.7756},
		{0.75, 0.9594},
		{0.9, 0.9944},
		{1, 1},
	}
	for _, c := range cases {
		if got := easeTransition(c.x); math.Abs(got-c.y) > 1e-3 {
			t.Errorf("easeTransition(%v) = %v, 期望 %v", c.x, got, c.y)
		}
	}
	// 曲线单调递增(过渡值不能来回抖)
	prev := -1.0
	for i := 0; i <= 100; i++ {
		v := easeTransition(float64(i) / 100)
		if v < prev {
			t.Fatalf("缓动曲线在 x=%v 处回退: %v < %v", float64(i)/100, v, prev)
		}
		prev = v
	}
}

// TestFade 校验过渡取色: 两端原样返回; 全透明 → 不透明的淡入必须走预乘
// (中途不能透出黑灰), 否则 Ghost 按钮悬停时会先"变脏"。
func TestFade(t *testing.T) {
	white := RGBA(255, 255, 255, 255)
	black := RGBA(0, 0, 0, 255)
	muted := RGBA(38, 38, 38, 255)

	if got := Fade(white, black, 0); got != white {
		t.Errorf("t=0 应返回原色, 得到 %v", got)
	}
	if got := Fade(white, black, 1); got != black {
		t.Errorf("t=1 应返回目标色, 得到 %v", got)
	}
	// 不透明对不透明: 退化成普通线性插值(127.5 四舍五入成 128)
	mid := Fade(black, white, 0.5)
	if r, g, b, a := mid.RGBA(); r != 128 || g != 128 || b != 128 || a != 255 {
		t.Errorf("black→white @0.5 期望 #808080, 得到 %v", mid)
	}
	// None → Muted @0.5: 只应该变淡, RGB 必须仍是 muted 的 RGB(预乘插值)
	faded := Fade(None, muted, 0.5)
	r, g, b, a := faded.RGBA()
	if r != 38 || g != 38 || b != 38 {
		t.Errorf("None→Muted @0.5 的 RGB 应为 38/38/38(预乘), 得到 %d/%d/%d", r, g, b)
	}
	if a < 126 || a > 129 {
		t.Errorf("None→Muted @0.5 的 alpha 应约 128, 得到 %d", a)
	}
	// 同色不同透明度: RGB 不变, 只有 alpha 插值(实心按钮 hover:bg-primary/80 走这条)
	p1 := RGBA(220, 38, 38, 255)
	p2 := RGBA(220, 38, 38, 204)
	r, g, b, a = Fade(p1, p2, 0.5).RGBA()
	if r != 220 || g != 38 || b != 38 {
		t.Errorf("primary→primary/80 的 RGB 不应改变, 得到 %d/%d/%d", r, g, b)
	}
	if a < 228 || a > 231 {
		t.Errorf("primary→primary/80 @0.5 的 alpha 应约 229, 得到 %d", a)
	}
}
