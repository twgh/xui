package ui

import (
	"testing"

	"github.com/twgh/xui"
)

func TestAlertTextHeight(t *testing.T) {
	cases := []struct {
		in, want int32
	}{
		{0, 20}, {20, 20}, {39, 44}, {59, 64}, {79, 84},
	}
	for _, c := range cases {
		if got := alertTextHeight(c.in); got != c.want {
			t.Errorf("alertTextHeight(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestAlertColors(t *testing.T) {
	th := xui.Light()
	def := &Alert{variant: AlertDefault}
	if got := def.foreground(th); got != th.CardForeground {
		t.Errorf("default foreground = %v, want CardForeground %v", got, th.CardForeground)
	}
	if got := def.descriptionColor(th); got != th.MutedForeground {
		t.Errorf("default description = %v, want MutedForeground %v", got, th.MutedForeground)
	}
	dest := &Alert{variant: AlertDestructive}
	if got := dest.foreground(th); got != th.Destructive {
		t.Errorf("destructive foreground = %v, want Destructive %v", got, th.Destructive)
	}
	if got := dest.descriptionColor(th); got != th.Destructive.ScaleAlpha(0.9) {
		t.Errorf("destructive description = %v, want Destructive/90", got)
	}
}

// TestAlertCustomColors AL1: 自定义配色盖住变体取色(两种变体都一样)。
func TestAlertCustomColors(t *testing.T) {
	th := xui.Light()
	bg, border := xui.Hex("#112233"), xui.Hex("#445566")
	fore, desc := xui.Hex("#778899"), xui.Hex("#aabbcc")

	a := &Alert{variant: AlertDefault}
	WithAlertColors(bg, border, fore, desc)(a)
	if !a.custom {
		t.Fatal("WithAlertColors 未标记自定义配色")
	}
	for _, v := range []AlertVariant{AlertDefault, AlertDestructive} {
		a.variant = v
		if got := a.foreground(th); got != fore {
			t.Errorf("variant %d foreground = %v, want 自定义 %v", v, got, fore)
		}
		if got := a.descriptionColor(th); got != desc {
			t.Errorf("variant %d description = %v, want 自定义 %v", v, got, desc)
		}
	}
}
