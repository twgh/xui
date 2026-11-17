package ui

import (
	"math"
	"testing"
)

// 这些测试固定分栏的纯计算语义: 份额归一化、拖动/键盘的伸缩与夹取、像素取整。
// 元素与鼠标交互见 demo 的视觉验收, 这里不碰 xcgui。

func floatSum(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s
}

func nearEqual(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func assertSizes(t *testing.T, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("长度 = %d, 期望 %d (%v)", len(got), len(want), want)
	}
	for i := range want {
		if !nearEqual(got[i], want[i]) {
			t.Fatalf("第 %d 项 = %v, 期望 %v (全部 %v)", i, got[i], want[i], got)
		}
	}
	if !nearEqual(floatSum(got), 100) {
		t.Fatalf("总和 = %v, 期望 100 (%v)", floatSum(got), got)
	}
}

func TestNormalizeResizableSizes(t *testing.T) {
	// 都没指定尺寸 → 平分
	assertSizes(t,
		normalizeResizableSizes([]float64{0, 0}, nil, nil),
		[]float64{50, 50})

	// 一个指定、一个未指定 → 未指定的吃掉剩余(先补份额, 再归一)
	assertSizes(t,
		normalizeResizableSizes(resizableInitialSizes([]float64{30, 0}), nil, nil),
		[]float64{30, 70})

	// 显式值总和不是 100 → 等比缩放到 100
	assertSizes(t,
		normalizeResizableSizes([]float64{40, 40}, nil, nil),
		[]float64{50, 50})

	// min 顶大一项后, 差额从另一项扣回, 总和仍是 100
	assertSizes(t,
		normalizeResizableSizes([]float64{10, 90}, []float64{40, 0}, nil),
		[]float64{40, 60})

	// max 压小一项后, 差额补给另一项
	assertSizes(t,
		normalizeResizableSizes([]float64{40, 80}, []float64{0, 0}, []float64{0, 60}),
		[]float64{40, 60})

	// 退化输入(负数 / NaN)按 0 处理, 仍能归一
	assertSizes(t,
		normalizeResizableSizes([]float64{-5, math.NaN()}, nil, nil),
		[]float64{50, 50})
}

func TestApplyResizableDelta(t *testing.T) {
	// 向右拖 60: 左邻只能从右邻拿到 50(右邻保底 min 20)
	assertSizes(t,
		applyResizableDelta([]float64{30, 70}, 0, 60, []float64{20, 20}, nil),
		[]float64{80, 20})

	// 向左拖 60: 右邻最多涨到 max 80, 因此只移动 10
	assertSizes(t,
		applyResizableDelta([]float64{30, 70}, 0, -60, []float64{20, 20}, []float64{100, 80}),
		[]float64{20, 80})

	// 相邻面板已到 min 时继续挤压更远的面板
	assertSizes(t,
		applyResizableDelta([]float64{40, 10, 50}, 1, 30, []float64{0, 0, 20}, nil),
		[]float64{40, 40, 20})

	// 负数方向同样会跨到更左的面板
	assertSizes(t,
		applyResizableDelta([]float64{50, 10, 40}, 1, -30, []float64{20, 10, 0}, nil),
		[]float64{20, 10, 70})

	// 非法下标 / 零位移不动
	orig := []float64{50, 50}
	assertSizes(t, applyResizableDelta(orig, 1, 10, nil, nil), orig)
	assertSizes(t, applyResizableDelta(orig, -1, 10, nil, nil), orig)
	assertSizes(t, applyResizableDelta(orig, 0, 0, nil, nil), orig)
}

func TestResizablePixelWidths(t *testing.T) {
	// 三段均分 100: 取整不能丢像素, 总和必须正好 100
	got := resizablePixelWidths([]float64{100.0 / 3, 100.0 / 3, 100.0 / 3}, 100)
	if got[0]+got[1]+got[2] != 100 {
		t.Fatalf("宽度之和 = %d, 期望 100 (%v)", got[0]+got[1]+got[2], got)
	}
	for i, w := range got {
		if w < 33 || w > 34 {
			t.Fatalf("第 %d 段 = %d, 期望 33~34 (%v)", i, w, got)
		}
	}

	// 极端小尺寸: 每段至少 1px, 不出现 0 宽面板
	got = resizablePixelWidths([]float64{50, 50}, 1)
	if got[0] < 1 || got[1] < 1 {
		t.Fatalf("小尺寸下出现 0 宽: %v", got)
	}

	// 百分比与可用宽不同: 60/40 在 250 上应得 150/100
	got = resizablePixelWidths([]float64{60, 40}, 250)
	if got[0] != 150 || got[1] != 100 {
		t.Fatalf("60/40 @250 = %v, 期望 [150 100]", got)
	}
}

func TestPinResizableSize(t *testing.T) {
	// 折叠到 0: 另一侧拿走全部
	assertSizes(t,
		pinResizableSize([]float64{35, 65}, 0, 0, []float64{20, 0}, nil),
		[]float64{0, 100})

	// 折叠尺寸低于 min: 钉住值不能被 min 抬回去
	assertSizes(t,
		pinResizableSize([]float64{35, 65}, 0, 5, []float64{20, 0}, nil),
		[]float64{5, 95})

	// 其它面板的 max 会限制分摊, 剩下的差额给还能长的面板
	assertSizes(t,
		pinResizableSize([]float64{30, 70, 30}, 0, 0, nil, []float64{0, 50, 100}),
		[]float64{0, 50, 50})
}

func TestResizableCollapseExpand(t *testing.T) {
	g := &ResizableGroup{sizes: []float64{35, 65}, normalized: true}
	a := &ResizablePanel{group: g, min: 20, collapsible: true, collapsedSizeSet: true, collapsedSize: 0}
	b := &ResizablePanel{group: g}
	g.panels = []*ResizablePanel{a, b}

	g.collapsePanel(a)
	if !a.collapsed || !nearEqual(a.restore, 35) {
		t.Fatalf("折叠后 collapsed=%v restore=%v, 期望 true / 35", a.collapsed, a.restore)
	}
	assertSizes(t, g.sizes, []float64{0, 100})

	// 已折叠时重复折叠不覆盖恢复尺寸
	g.collapsePanel(a)
	if !nearEqual(a.restore, 35) {
		t.Fatalf("重复折叠把 restore 改成 %v, 期望仍是 35", a.restore)
	}

	g.expandPanel(a)
	if a.collapsed || a.restore != 0 {
		t.Fatalf("展开后 collapsed=%v restore=%v, 期望 false / 0", a.collapsed, a.restore)
	}
	assertSizes(t, g.sizes, []float64{35, 65})

	// 展开目标低于 min 时按 min 恢复
	a.restore = 5
	a.collapsed = true
	g.expandPanel(a)
	assertSizes(t, g.sizes, []float64{20, 80})
}

func TestResizableCollapsedSizeDefaultsToMin(t *testing.T) {
	p := &ResizablePanel{min: 25, collapsible: true}
	if !nearEqual(p.collapseSize(), 25) {
		t.Fatalf("未配置 collapsedSize 时 = %v, 期望跟随 min 25", p.collapseSize())
	}

	// 显式 0 是合法的折叠尺寸, 不能当成"未配置"回落到 min
	p = &ResizablePanel{min: 25, collapsible: true, collapsedSizeSet: true, collapsedSize: 0}
	if p.collapseSize() != 0 {
		t.Fatalf("显式 collapsedSize=0 时 = %v, 期望 0", p.collapseSize())
	}
}

func TestResizableToggleAdjacentCollapse(t *testing.T) {
	g := &ResizableGroup{sizes: []float64{40, 60}, normalized: true}
	a := &ResizablePanel{group: g, collapsible: false}
	b := &ResizablePanel{group: g, collapsible: true, collapsedSizeSet: true, collapsedSize: 0}
	g.panels = []*ResizablePanel{a, b}

	// 前一个不可折叠 → 展开/折叠后一个
	if !g.toggleAdjacentCollapse(0) {
		t.Fatal("后一个可折叠时应消费 Enter")
	}
	if !b.collapsed {
		t.Fatal("Enter 未折叠后一个面板")
	}
	if !g.toggleAdjacentCollapse(0) || b.collapsed {
		t.Fatal("再次 Enter 应展开后一个面板")
	}

	// 两侧都不可折叠 → 不消费
	a.collapsible, b.collapsible = false, false
	if g.toggleAdjacentCollapse(0) {
		t.Fatal("两侧都不可折叠时不应消费 Enter")
	}
}

func TestResizableEffectiveDisabled(t *testing.T) {
	g := &ResizableGroup{}
	a := &ResizablePanel{group: g}
	b := &ResizablePanel{group: g}
	g.panels = []*ResizablePanel{a, b}
	h := &ResizableHandle{group: g, index: 0}
	g.handles = []*ResizableHandle{h}

	if g.effectiveDisabled(h) {
		t.Fatal("默认不应禁用")
	}
	a.disabled = true
	if !g.effectiveDisabled(h) {
		t.Fatal("左侧面板禁用后分隔条应禁用")
	}
	a.disabled = false
	h.disabled = true
	if !g.effectiveDisabled(h) {
		t.Fatal("分隔条自身禁用应生效")
	}
	h.disabled = false
	g.disabled = true
	if !g.effectiveDisabled(h) {
		t.Fatal("分组禁用应叠加生效")
	}
}
