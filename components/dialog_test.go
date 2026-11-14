package ui

import "testing"

func TestAlertDialogPanelWidth(t *testing.T) {
	cases := []struct {
		name  string
		size  AlertDialogSize
		bodyW int32
		want  int32
	}{
		{"default wide", AlertDialogSizeDefault, 1200, alertDlgDefaultW},
		{"default at sm breakpoint", AlertDialogSizeDefault, 640, alertDlgDefaultW},
		{"default below sm", AlertDialogSizeDefault, 639, alertDlgSmallW},
		{"small wide", AlertDialogSizeSmall, 1200, alertDlgSmallW},
		{"clamped to body", AlertDialogSizeDefault, 300, 300 - 2*dlgPad},
	}
	for _, c := range cases {
		if got := alertDialogPanelWidth(c.size, c.bodyW); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

// 关闭否决与开合通知都是纯回调逻辑(不碰元素), 直接测。
func TestDialogCanCloseVeto(t *testing.T) {
	d := &Dialog{}
	if !d.canClose() {
		t.Fatal("没有 OnClosing 时必须放行关闭")
	}
	d.OnClosing(func() bool { return true })
	if !d.canClose() {
		t.Fatal("全部回调放行时必须放行关闭")
	}
	d.OnClosing(func() bool { return false })
	if d.canClose() {
		t.Fatal("任一回调返回 false 时必须拦下关闭")
	}
}

func TestDialogOnOpenChange(t *testing.T) {
	d := &Dialog{}
	var got []bool
	d.OnOpenChange(func(open bool) { got = append(got, open) })
	d.OnOpenChange(func(open bool) { got = append(got, open) })
	d.notifyOpenChange(true)
	d.notifyOpenChange(false)
	want := []bool{true, true, false, false}
	if len(got) != len(want) {
		t.Fatalf("回调收到 %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("回调收到 %v, want %v", got, want)
		}
	}
}
