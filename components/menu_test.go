package ui

import "testing"

func TestMenuOptions(t *testing.T) {
	m := NewMenu()
	m.Item("复制", MenuShortcut("Ctrl+C"), MenuIcon("copy"), MenuInset())
	m.Label("人员", MenuInset())
	m.Sub("更多工具", func(s *Menu) {
		s.Item("刷新")
	}, MenuInset(), MenuIcon("wrench"))

	copyItem := m.byID[1]
	if copyItem.shortcut != "Ctrl+C" || copyItem.icon != "copy" || !copyItem.inset {
		t.Errorf("item opts not applied: %+v", copyItem)
	}
	label := m.byID[2]
	if label.kind != menuKindLabel || !label.disabled || !label.inset {
		t.Errorf("label opts: %+v", label)
	}
	sub := m.byID[3]
	if sub.kind != menuKindSub || !sub.inset || sub.icon != "wrench" {
		t.Errorf("sub opts: %+v", sub)
	}
	if child := m.byID[4]; child == nil || child.parent != sub.id {
		t.Errorf("sub child parent = %v, want %d", child, sub.id)
	}
}

func TestMenuDispatch(t *testing.T) {
	var gotCheck bool
	var selected bool
	m := NewMenu()
	m.Check("显示行号", true, MenuOnCheck(func(on bool) { gotCheck = on }))
	m.Radio("张伟", true, MenuGroup(1))
	m.Radio("李娜", false, MenuGroup(1))
	m.Item("删除", MenuOnSelect(func() { selected = true }))

	m.dispatch(1)
	if gotCheck || m.byID[1].checked {
		t.Errorf("check should toggle false: checked=%v cb=%v", m.byID[1].checked, gotCheck)
	}

	m.dispatch(3)
	if !m.byID[3].checked || m.byID[2].checked {
		t.Errorf("radio group not exclusive: zhang=%v li=%v", m.byID[2].checked, m.byID[3].checked)
	}

	m.dispatch(4)
	if !selected {
		t.Error("normal item onSelect not called")
	}
}
