package ui

import (
	"math"
	"testing"
)

// 这些测试只覆盖不依赖元素句柄的纯数值逻辑(值域换算 / 步进吸附 / 滑块间距)。
// 交互(拖拽、键盘)要靠探针程序验证。

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestSliderGeoFrac(t *testing.T) {
	s := &Slider{min: 0, max: 100, values: []float64{25}}
	if got := s.geoFrac(25); !almost(got, 0.25) {
		t.Fatalf("geoFrac(25) = %v, want 0.25", got)
	}
	if got := s.valueOfGeo(0.25); !almost(got, 25) {
		t.Fatalf("valueOfGeo(0.25) = %v, want 25", got)
	}
	s.inverted = true
	if got := s.geoFrac(25); !almost(got, 0.75) {
		t.Fatalf("inverted geoFrac(25) = %v, want 0.75", got)
	}
	if got := s.valueOfGeo(0.75); !almost(got, 25) {
		t.Fatalf("inverted valueOfGeo(0.75) = %v, want 25", got)
	}
	// 值域不是 0 起: 中间值仍映射到 0.5
	s2 := &Slider{min: 10, max: 20, values: []float64{15}}
	if got := s2.geoFrac(15); !almost(got, 0.5) {
		t.Fatalf("geoFrac(15) in [10,20] = %v, want 0.5", got)
	}
}

func TestSliderStepSnap(t *testing.T) {
	s := &Slider{min: 0, max: 100, step: 5, values: []float64{0}}
	s.setValueAt(0, 42)
	if !almost(s.values[0], 40) {
		t.Fatalf("step snap: got %v, want 40", s.values[0])
	}
	s.setValueAt(0, 101)
	if !almost(s.values[0], 100) {
		t.Fatalf("clamp max: got %v, want 100", s.values[0])
	}
	s.setValueAt(0, -3)
	if !almost(s.values[0], 0) {
		t.Fatalf("clamp min: got %v, want 0", s.values[0])
	}
}

func TestSliderValuesStayOrdered(t *testing.T) {
	// 双滑块: 左滑块不能越过右滑块(minStepsBetweenThumbs = 10)
	s := &Slider{min: 0, max: 100, step: 1, minSteps: 10, values: []float64{20, 60}}
	if !s.setValueAt(0, 55) {
		t.Fatal("setValueAt should change value")
	}
	if !almost(s.values[0], 50) || !almost(s.values[1], 60) {
		t.Fatalf("left thumb = %v / right = %v, want 50 / 60", s.values[0], s.values[1])
	}
	// 右滑块同理(试图压到左滑块上时停在 left+gap, 值不变)
	s.setValueAt(1, 10)
	if !almost(s.values[1], 60) {
		t.Fatalf("right thumb = %v, want 60 (left+gap)", s.values[1])
	}
}

func TestSliderNormalize(t *testing.T) {
	s := &Slider{min: 0, max: 100, values: []float64{80, 20, 150}}
	s.normalize()
	if len(s.values) != 3 {
		t.Fatalf("len = %d, want 3", len(s.values))
	}
	if !almost(s.values[0], 20) || !almost(s.values[1], 80) || !almost(s.values[2], 100) {
		t.Fatalf("normalize = %v, want [20 80 100]", s.values)
	}
}

func TestSliderMinGap(t *testing.T) {
	// 连续模式(step=0): 一格 = 值域千分之一
	s := &Slider{min: 0, max: 100}
	if got := s.unitStep(); !almost(got, 0.1) {
		t.Fatalf("continuous unit step = %v, want 0.1", got)
	}
	s.minSteps = 5
	if got := s.minGap(); !almost(got, 0.5) {
		t.Fatalf("min gap = %v, want 0.5", got)
	}
	// 有步进时按 step 计
	s2 := &Slider{min: 0, max: 100, step: 2, minSteps: 3}
	if got := s2.minGap(); !almost(got, 6) {
		t.Fatalf("step min gap = %v, want 6", got)
	}
}

func TestSliderEdgeAlignment(t *testing.T) {
	// 水平 200 宽: edge 时滑块**边缘**贴轨道端点(轨道端点 = 元素内缩 sliderHalo)
	start, length := sliderAxisSpan(200, false, true)
	if want := sliderHalo + sliderThumb/2; start != want {
		t.Fatalf("edge start = %d, want %d", start, want)
	}
	if want := 200 - 2*sliderHalo - sliderThumb; length != want {
		t.Fatalf("edge length = %d, want %d", length, want)
	}
	if got := start - sliderThumb/2; got != sliderHalo {
		t.Fatalf("f=0 时滑块左缘 = %d, want %d", got, sliderHalo)
	}
	if got := start + length + sliderThumb/2; got != 200-sliderHalo {
		t.Fatalf("f=1 时滑块右缘 = %d, want %d", got, 200-sliderHalo)
	}

	// center(旧语义): 中心贴轨道端点
	cs, cl := sliderAxisSpan(200, false, false)
	if cs != sliderHalo || cl != 200-2*sliderHalo {
		t.Fatalf("center span = (%d,%d), want (%d,%d)", cs, cl, sliderHalo, 200-2*sliderHalo)
	}

	// 垂直: 轴从底部起算, edge 同样内缩半个滑块
	vs, vl := sliderAxisSpan(200, true, true)
	if want := 200 - sliderHalo - sliderThumb/2; vs != want {
		t.Fatalf("vertical edge start = %d, want %d", vs, want)
	}
	if want := 200 - 2*sliderHalo - sliderThumb; vl != want {
		t.Fatalf("vertical edge length = %d, want %d", vl, want)
	}

	// 退化尺寸不产生负长度
	if _, l := sliderAxisSpan(4, false, true); l != 0 {
		t.Fatalf("过短元素 length = %d, want 0", l)
	}
}

func TestSliderLargeStep(t *testing.T) {
	// 未显式设置: 大步进 = step×10(旧行为); 连续模式为值域的一半
	s := &Slider{min: 0, max: 100, step: 2}
	if got := s.normalStep(); !almost(got, 2) {
		t.Fatalf("normalStep = %v, want 2", got)
	}
	if got := s.largeStepValue(); !almost(got, 20) {
		t.Fatalf("largeStepValue = %v, want 20", got)
	}
	c := &Slider{min: 0, max: 100}
	if got := c.largeStepValue(); !almost(got, 50) {
		t.Fatalf("连续模式 largeStepValue = %v, want 50", got)
	}
	// 显式 largeStep 优先
	e := &Slider{min: 0, max: 100, step: 1, largeStep: 7}
	if got := e.largeStepValue(); !almost(got, 7) {
		t.Fatalf("显式 largeStepValue = %v, want 7", got)
	}
}

func TestSliderValuesAccessors(t *testing.T) {
	s := &Slider{min: 0, max: 100, values: []float64{10, 90}}
	if got := s.Value(); !almost(got, 10) {
		t.Fatalf("Value = %v, want 10", got)
	}
	if got := s.ValueAt(1); !almost(got, 90) {
		t.Fatalf("ValueAt(1) = %v, want 90", got)
	}
	if got := s.ValueAt(5); !almost(got, 0) {
		t.Fatalf("ValueAt(out of range) = %v, want min", got)
	}
	vals := s.Values()
	vals[0] = 999
	if !almost(s.values[0], 10) {
		t.Fatal("Values() must return a copy")
	}
}
