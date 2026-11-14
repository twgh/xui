package ui

import "testing"

// 只读/否决的判定是纯字段逻辑(不需要 GUI), 直接测。

func TestSwitchReadOnlyOption(t *testing.T) {
	sw := &Switch{}
	if sw.ReadOnly() {
		t.Fatal("新建开关默认不是只读")
	}
	WithSwitchReadOnly()(sw)
	if !sw.ReadOnly() {
		t.Fatal("WithSwitchReadOnly 未生效")
	}
}

func TestSwitchAllowChange(t *testing.T) {
	sw := &Switch{}
	if !sw.allowChange(true) || !sw.allowChange(false) {
		t.Fatal("没有 OnCheckedChange 时两个方向都放行")
	}
	sw.OnCheckedChange(func(checked bool) bool { return checked }) // 只允许"选中", 拒绝"取消"
	if !sw.allowChange(true) {
		t.Fatal("勾选方向应放行")
	}
	if sw.allowChange(false) {
		t.Fatal("取消方向应被否决")
	}
}
