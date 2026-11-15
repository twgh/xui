package ui

import "testing"

// TestSelectDisplayText S8: 自定义格式化优先; 没有格式化时 Text 空回落 Value。
func TestSelectDisplayText(t *testing.T) {
	s := &Select{}
	if got := s.displayText(SelectItem{Value: "v1", Text: "一号"}); got != "一号" {
		t.Errorf("有 Text 时 = %q, want 一号", got)
	}
	if got := s.displayText(SelectItem{Value: "v1"}); got != "v1" {
		t.Errorf("Text 为空时 = %q, want v1", got)
	}

	WithSelectValueText(func(it SelectItem) string { return "¥" + it.Value })(s)
	if got := s.displayText(SelectItem{Value: "42", Text: "四十二"}); got != "¥42" {
		t.Errorf("自定义格式化 = %q, want ¥42", got)
	}
}

// TestSelectReadOnlyOption S4: 只读与禁用是两件事, 只读不置灰。
func TestSelectReadOnlyOption(t *testing.T) {
	s := &Select{}
	if s.ReadOnly() {
		t.Fatal("新建默认不是只读")
	}
	WithSelectReadOnly()(s)
	if !s.ReadOnly() {
		t.Fatal("WithSelectReadOnly 未生效")
	}
	if s.Disabled() {
		t.Fatal("只读不应同时变成禁用")
	}
	s.SetReadOnly(false)
	if s.ReadOnly() {
		t.Fatal("SetReadOnly(false) 未恢复")
	}
}
