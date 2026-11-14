package ui

import (
	"testing"

	"github.com/twgh/xui"
)

// 头像的徽标/图标配置项是纯字段逻辑(元素与绘制在 demo 里视觉验收), 这里测选项语义。

func TestAvatarBadgeOptions(t *testing.T) {
	a := &Avatar{}
	if a.badge {
		t.Fatal("新建头像默认没有徽标")
	}
	WithAvatarBadge("check")(a)
	if !a.badge || a.badgeIcon != "check" {
		t.Fatalf("WithAvatarBadge 未生效: %+v", a)
	}
	if a.badgeColor != nil || a.badgeIconColor != nil {
		t.Fatal("默认应跟随主题色")
	}
	WithAvatarBadgeColor(xui.Hex("#16a34a"))(a)
	if a.badgeColor == nil {
		t.Fatal("WithAvatarBadgeColor 未设置解析器")
	}
	WithAvatarBadgeSize(14)(a)
	if a.badgeSize != 14 {
		t.Fatalf("badgeSize = %d, want 14", a.badgeSize)
	}
	WithAvatarBadgeSize(0)(a) // <=0 忽略
	if a.badgeSize != 14 {
		t.Fatalf("badgeSize 被 0 覆盖成 %d", a.badgeSize)
	}
}

func TestAvatarIconOption(t *testing.T) {
	a := &Avatar{}
	if a.icon != "" {
		t.Fatal("新建头像默认没有图标")
	}
	a.icon = "plus"
	if a.icon != "plus" {
		t.Fatal("icon 字段不可写")
	}
}
