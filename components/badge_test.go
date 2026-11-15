package ui

import "testing"

// TestBadgeSidePads BA2: 有图标那一侧内距收窄到 6(has-data-[icon]:px-1.5), 另一侧仍是 8。
func TestBadgeSidePads(t *testing.T) {
	cases := []struct {
		name         string
		leading      string
		trailing     string
		loading      bool
		wantL, wantR int32
	}{
		{"纯文字", "", "", false, badgePadX, badgePadX},
		{"前置图标", "icon", "", false, badgeIconPadX, badgePadX},
		{"后置图标", "", "icon", false, badgePadX, badgeIconPadX},
		{"两侧图标", "a", "b", false, badgeIconPadX, badgeIconPadX},
		{"加载指示器算前置", "", "", true, badgeIconPadX, badgePadX},
	}
	for _, c := range cases {
		b := &Badge{leading: c.leading, trailing: c.trailing, loading: c.loading}
		if l, r := b.sidePads(); l != c.wantL || r != c.wantR {
			t.Errorf("%s: sidePads = (%d, %d), want (%d, %d)", c.name, l, r, c.wantL, c.wantR)
		}
	}
}

// TestBadgeLinkURL BA3: 链接地址默认为空, 非链接徽章不会打开任何东西。
func TestBadgeLinkURL(t *testing.T) {
	b := &Badge{}
	if got := b.LinkURL(); got != "" {
		t.Fatalf("新建 LinkURL = %q, want 空串", got)
	}
	b.linkURL = "https://example.com"
	if got := b.LinkURL(); got != "https://example.com" {
		t.Fatalf("LinkURL = %q", got)
	}
}
