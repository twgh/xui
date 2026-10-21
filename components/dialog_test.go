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
