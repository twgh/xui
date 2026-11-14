package ui

import "testing"

// 卡片间距的取值规则是纯字段逻辑(元素创建/下发放到 demo 视觉验收), 这里测选项语义。

func TestCardSpacingOption(t *testing.T) {
	// 未设置: default = 16, sm = 12
	def := &Card{}
	if got := WithCardSize(CardSizeDefault); got == nil {
		t.Fatal("WithCardSize 返回空选项")
	}
	WithCardSize(CardSizeSm)(def)
	if def.Size() != CardSizeSm {
		t.Fatalf("Size = %v, want CardSizeSm", def.Size())
	}
	if def.TitleSize() != 14 {
		t.Fatalf("sm 标题字号 = %d, want 14", def.TitleSize())
	}

	// WithCardSpacing 记下显式值, 由 NewCard 换算成 Padding(构造需要窗口, 这里只验选项本身)
	c := &Card{}
	WithCardSpacing(24)(c)
	if !c.spacingSet || c.spacing != 24 {
		t.Fatalf("spacing = %d (set=%v), want 24", c.spacing, c.spacingSet)
	}
	if got := c.Spacing(); got != 0 {
		t.Fatalf("未构造时 Spacing 读字段 = %d, want 0", got)
	}
	c.Padding = 32
	if got := c.Spacing(); got != 32 {
		t.Fatalf("Spacing 应回读 Padding, got %d", got)
	}
}

func TestCardFooterPaddingOption(t *testing.T) {
	f := &CardFooter{}
	WithCardFooterPadding(12)(f)
	if f.Padding != 12 {
		t.Fatalf("页脚内边距 = %d, want 12", f.Padding)
	}
}
