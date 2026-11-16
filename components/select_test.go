package ui

import "testing"

func typeaheadRows(items ...SelectItem) []*selectRow {
	rows := make([]*selectRow, 0, len(items))
	for _, it := range items {
		rows = append(rows, &selectRow{kind: selectRowItem, item: it, seq: -1})
	}
	return rows
}

func TestSelectPrefixMatch(t *testing.T) {
	rows := typeaheadRows(
		SelectItem{Value: "apple", Text: "Apple"},
		SelectItem{Value: "apricot", Text: "Apricot"},
		SelectItem{Value: "banana", Text: "Banana", Disabled: true},
		SelectItem{Value: "blueberry", Text: "Blueberry"},
	)
	cases := []struct {
		name   string
		prefix string
		start  int32
		want   int32
	}{
		{"first match", "app", 0, 0},
		{"case insensitive", "AP", 0, 0},
		{"from active+1", "ap", 1, 1},
		{"wraps around", "bl", 2, 3},
		{"skips disabled", "ba", 0, -1},
		{"no match", "zz", 0, -1},
		{"empty prefix", "", 0, -1},
		{"negative start", "ban", -2, -1},
	}
	for _, c := range cases {
		if got := selectPrefixMatch(rows, c.prefix, c.start); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestSelectAllowRapidCycle(t *testing.T) {
	if !selectAllowRapidCycle(typeaheadRows(
		SelectItem{Text: "Apple"}, SelectItem{Text: "Banana"})) {
		t.Fatal("前两个字母都不同的列表应允许同字符连按循环")
	}
	if selectAllowRapidCycle(typeaheadRows(
		SelectItem{Text: "Apple"}, SelectItem{Text: "Llama"})) {
		t.Fatal("存在 llama 这类同首双字母条目时不能把连按当循环")
	}
	if !selectAllowRapidCycle(typeaheadRows(
		SelectItem{Text: "Llama", Disabled: true}, SelectItem{Text: "Apple"})) {
		t.Fatal("禁用条目不应阻止循环")
	}
}

func TestSelectTypeaheadChar(t *testing.T) {
	cases := []struct {
		vk   uintptr
		want byte
		ok   bool
	}{
		{'A', 'A', true},
		{'7', '7', true},
		{0x63, '3', true}, // VK_NUMPAD3
		{0x20, ' ', true},
		{0x28, 0, false}, // VK_DOWN
		{0x0D, 0, false}, // VK_RETURN
	}
	for _, c := range cases {
		got, ok := selectTypeaheadChar(c.vk)
		if got != c.want || ok != c.ok {
			t.Errorf("vk=%#x: got (%q,%v), want (%q,%v)", c.vk, got, ok, c.want, c.ok)
		}
	}
}
