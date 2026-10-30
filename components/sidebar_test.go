package ui

import (
	"testing"
)

func TestSidebarGapWidth(t *testing.T) {
	// 纯几何: 不建窗口, 只验证收起/展开宽度与变体的换算。
	cases := []struct {
		name        string
		collapsible SidebarCollapsible
		variant     SidebarVariant
		progress    float64
		want        float64
	}{
		{"icon-expanded", SidebarCollapsibleIcon, SidebarVariantSidebar, 1, 256},
		{"icon-collapsed", SidebarCollapsibleIcon, SidebarVariantSidebar, 0, 48},
		{"icon-floating-collapsed", SidebarCollapsibleIcon, SidebarVariantFloating, 0, 66},
		{"icon-inset-collapsed", SidebarCollapsibleIcon, SidebarVariantInset, 0, 66},
		{"icon-half", SidebarCollapsibleIcon, SidebarVariantSidebar, 0.5, 152},
		{"offcanvas-expanded", SidebarCollapsibleOffcanvas, SidebarVariantSidebar, 1, 256},
		{"offcanvas-collapsed", SidebarCollapsibleOffcanvas, SidebarVariantSidebar, 0, 0},
		{"none", SidebarCollapsibleNone, SidebarVariantSidebar, 0, 256},
	}
	for _, c := range cases {
		s := &Sidebar{width: 256, iconWidth: 48, collapsible: c.collapsible, variant: c.variant, progress: c.progress}
		if got := s.gapWidth(); got != c.want {
			t.Fatalf("%s: gapWidth = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestSidebarItemHeight(t *testing.T) {
	cases := map[SidebarSize]int32{
		SidebarSizeDefault: 32,
		SidebarSizeSm:      28,
		SidebarSizeLg:      48,
	}
	for size, want := range cases {
		if got := sidebarItemHeight(size); got != want {
			t.Fatalf("sidebarItemHeight(%v) = %d, want %d", size, got, want)
		}
	}
}

func TestSidebarIconMode(t *testing.T) {
	cases := []struct {
		collapsible SidebarCollapsible
		open        bool
		want        bool
	}{
		{SidebarCollapsibleIcon, false, true},
		{SidebarCollapsibleIcon, true, false},
		{SidebarCollapsibleOffcanvas, false, false},
		{SidebarCollapsibleNone, false, false},
	}
	for _, c := range cases {
		s := &Sidebar{collapsible: c.collapsible, open: c.open}
		if got := s.iconMode(); got != c.want {
			t.Fatalf("collapsible=%v open=%v iconMode = %v, want %v", c.collapsible, c.open, got, c.want)
		}
	}
}
