package ui

import (
	"testing"

	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

// 焦点/按键回调是纯注册, 无需起窗口; Blur 在元素还没建出来时也不能 panic。
func TestRatingFocusCallbacks(t *testing.T) {
	r := &Rating{}
	var got string
	r.OnFocus(func() { got += "f" }).
		OnBlur(func() { got += "b" }).
		OnKeyDown(func(vk uintptr) { got += "k" })
	r.focusCb()
	r.blurCb()
	r.keyCb(0x24)
	if got != "fbk" {
		t.Fatalf("callbacks = %q, want fbk", got)
	}
	if r.Blur() != r {
		t.Fatal("Blur should return the receiver")
	}
}

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

// 三档尺寸共用同一套几何函数: 星高/自然宽/星矩形/命中映射都按 starSize 走。
func TestRatingSizeGeometry(t *testing.T) {
	for _, c := range []struct {
		size RatingSize
		want int32
	}{{RatingMedium, 20}, {RatingSmall, 15}, {RatingLarge, 25}} {
		s := ratingStarSizeFor(c.size)
		if s != c.want {
			t.Fatalf("starSize(%v) = %d, want %d", c.size, s, c.want)
		}
		rc := xc.RECT{Left: 0, Top: 0, Right: 400, Bottom: s + 2*ratingFocusPad}
		area := ratingArea(rc, 5, s)
		if area.Right != ratingNaturalWidth(5, s) || area.Bottom-area.Top != s+2*ratingFocusPad {
			t.Fatalf("area = %v, want natural size for starSize %d", area, s)
		}
		for i := 0; i < 5; i++ {
			box := ratingStarRect(area, i, s)
			if box.Right-box.Left != s ||
				box.Left != area.Left+ratingFocusPad+int32(i)*(s+ratingStarGap) {
				t.Fatalf("starRect(%d, starSize %d) = %v", i, s, box)
			}
		}
		star0 := area.Left + ratingFocusPad
		// 奇数尺寸的"中线"要向上取整, 否则 15 的一半会被算进左半。
		for _, m := range []struct {
			x    int32
			want float64
		}{{star0, 0.5}, {star0 + (s+1)/2, 1}, {area.Right - 1, -1}} {
			if got := ratingValueAt(area, m.x, 5, true, s); got != m.want {
				t.Fatalf("valueAt(%d, starSize %d) = %v, want %v", m.x, s, got, m.want)
			}
		}
	}
	if got, want := ratingNaturalWidth(0, 0), ratingNaturalWidth(5, 20); got != want {
		t.Fatalf("natural width(0, 0) = %d, want default %d", got, want)
	}
}

func TestRatingNaturalWidth(t *testing.T) {
	s := ratingStarSizeFor(RatingMedium)
	if got, want := ratingNaturalWidth(5, s), 2*ratingFocusPad+5*s+4*ratingStarGap; got != want {
		t.Fatalf("natural width = %d, want %d", got, want)
	}
	if got, want := ratingNaturalWidth(0, s), ratingNaturalWidth(5, s); got != want {
		t.Fatalf("natural width(0) = %d, want %d", got, want)
	}
}

// 星星区从左边留 3px 开始: 每颗星的左半 = i+0.5(半星), 右半/星间空隙 = i+1,
// 星星区左右留白与元素被 Fill 拉宽后的空区都返回 -1(不参与预览/点击)。
func TestRatingValueAt(t *testing.T) {
	s := ratingStarSizeFor(RatingMedium)
	area := ratingArea(xc.RECT{Left: 0, Top: 0, Right: 400, Bottom: 26}, 5, s)
	if area.Right != ratingNaturalWidth(5, s) || area.Bottom-area.Top != s+2*ratingFocusPad {
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
		{star0 + s/2 - 1, true, 0.5},
		{star0 + s/2, true, 1}, // 中线归右半
		{star0 + s - 1, false, 1},
		{star0 + s + 1, true, 1}, // 星间空隙算前一颗星的右端
		{star0 + s + ratingStarGap, true, 1.5},
		{star0 + 4*(s+ratingStarGap) + 1, false, 5},
		{area.Right - 1, false, -1}, // 右侧留白
		{400, false, -1},            // 元素被拉宽后的空区
	}
	for _, c := range cases {
		if got := ratingValueAt(area, c.x, 5, c.half, s); got != c.want {
			t.Fatalf("valueAt(%d, half=%v) = %v, want %v", c.x, c.half, got, c.want)
		}
	}
}

// allowClear / 文案 / 悬停索引都是纯逻辑, 不需要起窗口。
func TestRatingPickAndLabels(t *testing.T) {
	r := &Rating{max: 5, value: 3, allowClear: true}
	if got := r.pickValue(3); got != 0 {
		t.Fatalf("pickValue(3) = %v, want 0(再次点击清除)", got)
	}
	if got := r.pickValue(4); got != 4 {
		t.Fatalf("pickValue(4) = %v, want 4", got)
	}
	r.SetAllowClear(false)
	if got := r.pickValue(3); got != 3 {
		t.Fatalf("pickValue(3) after SetAllowClear(false) = %v, want 3", got)
	}
	zero := &Rating{max: 5, value: 0, allowClear: true}
	if got := zero.pickValue(0.5); got != 0.5 {
		t.Fatalf("pickValue(0.5) from zero = %v, want 0.5", got)
	}

	labels := []string{"很差", "差", "一般", "好", "很好"}
	lr := &Rating{max: 5, labels: labels, starSize: 20, labelW: 30}
	for _, c := range []struct {
		v    float64
		want string
	}{{0, ""}, {0.5, "很差"}, {1, "很差"}, {1.5, "差"}, {3.5, "好"}, {5, "很好"}, {6, ""}} {
		if got := lr.labelFor(c.v); got != c.want {
			t.Fatalf("labelFor(%v) = %q, want %q", c.v, got, c.want)
		}
	}
	if got, want := lr.naturalWidth(), ratingNaturalWidth(5, 20)+ratingLabelGap+30; got != want {
		t.Fatalf("natural width with labels = %d, want %d", got, want)
	}
	for _, c := range []struct {
		hover float64
		want  int
	}{{-1, -1}, {0.5, 0}, {1, 0}, {2.5, 2}, {5, 4}} {
		hr := &Rating{max: 5, hover: c.hover}
		if got := hr.hoverIndex(); got != c.want {
			t.Fatalf("hoverIndex(%v) = %d, want %d", c.hover, got, c.want)
		}
	}
}

func TestRatingOptions(t *testing.T) {
	r := &Rating{size: RatingMedium, outlineSVG: "a", fullSVG: "b", halfSVG: "c"}
	WithRatingSize(RatingLarge)(r)
	WithRatingSize(RatingSize(99))(r) // 非法值不生效
	WithRatingIcons("outline", "", "half")(r)
	WithRatingAllowClear(false)(r)
	WithRatingKeyboard(false)(r)
	if r.size != RatingLarge || r.outlineSVG != "outline" || r.fullSVG != "b" || r.halfSVG != "half" {
		t.Fatalf("options = %+v", r)
	}
	if r.allowClear || r.keyboard {
		t.Fatalf("allowClear/keyboard should be off: %+v", r)
	}
	labels := []string{"A", "B"}
	WithRatingLabels(labels)(r)
	labels[0] = "changed"
	if r.labels[0] != "A" {
		t.Fatalf("labels must be copied, got %q", r.labels[0])
	}
}

// 实心星/半星与自定义图标的 SVG 必须能被运行时解析(否则运行时什么都不画, 且不会报错)。
func TestRatingStarSVG(t *testing.T) {
	const heartFull = `<svg viewBox="0 0 24 24"><path fill="currentColor" d="M2 9.5A5.5 5.5 0 0 1 11.591 5.824A.56 .56 0 0 0 12.409 5.824A5.49 5.49 0 0 1 22 9.5C22 11.79 20.5 13.5 19 15L13.508 20.313A2 2 0 0 1 10.508 20.332L5 15C3.5 13.5 2 11.8 2 9.5 Z"/></svg>`
	const heartHalf = `<svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 20.8 L10.508 20.332 L5 15 C3.5 13.5 2 11.8 2 9.5 A5.5 5.5 0 0 1 11.591 5.824 L12 5.82 Z"/></svg>`
	for name, svg := range map[string]string{
		"full":      ratingStarFullSVG,
		"half":      ratingStarHalfSVG,
		"heartFull": heartFull,
		"heartHalf": heartHalf,
	} {
		if paths, _, err := xui.ParseIconSVG(svg); err != nil || paths == "" {
			t.Fatalf("%s star svg: paths=%q err=%v", name, paths, err)
		}
	}
}
