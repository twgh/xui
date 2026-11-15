package ui

import "testing"

// 这些用例只覆盖不依赖元素句柄的纯状态逻辑(否决回调 / 集合计算 / 整组禁用标志)。
// 点击与键盘的真实路径要靠探针验证(原生 Check 按钮的勾选位回滚)。

func newTestAccordion(mode AccordionType, values ...string) (*Accordion, []*AccordionItem) {
	a := &Accordion{mode: mode}
	items := make([]*AccordionItem, 0, len(values))
	for _, v := range values {
		it := &AccordionItem{acc: a, value: v}
		items = append(items, it)
	}
	a.items = items
	return a, items
}

func TestAccordionUserOpenVeto(t *testing.T) {
	a, items := newTestAccordion(AccordionSingle, "a", "b")
	first, second := items[0], items[1]

	// 条目级否决: 状态不落地
	first.OnOpenChangeVeto(func(open bool) bool { return false })
	if first.userOpen(true) {
		t.Fatal("条目否决后 userOpen 应返回 false")
	}
	if first.Open() {
		t.Fatal("条目否决后不应展开")
	}

	// 根级否决: single 模式下展开 b 时 next 只有 b
	var next []string
	a.OnValueChangeVeto(func(vals []string) bool { next = vals; return false })
	if second.userOpen(true) {
		t.Fatal("根否决后 userOpen 应返回 false")
	}
	if len(next) != 1 || next[0] != "b" {
		t.Fatalf("single 模式的 next = %v, want [b]", next)
	}
	if second.Open() {
		t.Fatal("根否决后不应展开")
	}
}

func TestAccordionValuesWith(t *testing.T) {
	// multiple: 已有 a 展开, 再展开 b 时 next = [a b]
	a, items := newTestAccordion(AccordionMultiple, "a", "b", "c")
	items[0].open = true
	if got := a.valuesWith(items[1], true); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("multiple next = %v, want [a b]", got)
	}
	// 收起 a: next 为空
	if got := a.valuesWith(items[0], false); len(got) != 0 {
		t.Fatalf("收起后的 next = %v, want 空", got)
	}

	// single: 展开 b 会收起 a, next 只有 b
	s, sitems := newTestAccordion(AccordionSingle, "a", "b")
	sitems[0].open = true
	if got := s.valuesWith(sitems[1], true); len(got) != 1 || got[0] != "b" {
		t.Fatalf("single next = %v, want [b]", got)
	}
}

func TestAccordionProgrammaticSkipsVeto(t *testing.T) {
	// 程序化 SetOpen / SetValue 不经过否决(对齐上游: cancel 只拦用户交互)
	a, items := newTestAccordion(AccordionSingle, "a")
	a.OnValueChangeVeto(func([]string) bool { return false })
	items[0].OnOpenChangeVeto(func(bool) bool { return false })

	a.SetValue("a")
	if !items[0].Open() {
		t.Fatal("SetValue 不应被否决回调拦下")
	}
	items[0].SetOpen(false)
	if items[0].Open() {
		t.Fatal("SetOpen(false) 不应被否决回调拦下")
	}
}

func TestAccordionRootDisabledFlag(t *testing.T) {
	a, _ := newTestAccordion(AccordionSingle, "a")
	if a.Disabled() {
		t.Fatal("默认不应禁用")
	}
	a.SetDisabled(true)
	if !a.Disabled() {
		t.Fatal("SetDisabled(true) 后应为禁用")
	}
	a.SetDisabled(false)
	if a.Disabled() {
		t.Fatal("SetDisabled(false) 后应恢复")
	}
	var nilAcc *Accordion
	if nilAcc.Disabled() {
		t.Fatal("nil 接收者应返回 false")
	}
}
