package ui

import "testing"

// 状态栏的纯计算部分(尺寸读数、尺寸下限夹取、选项)不依赖 GUI, 直接测。
func TestStatusBarSizeTextAndClamp(t *testing.T) {
	if got := statusBarSizeText(1200, 860); got != "1200 × 860" {
		t.Errorf("尺寸读数 = %q", got)
	}
	if got := statusBarClamp(120, statusBarMinWinW); got != statusBarMinWinW {
		t.Errorf("缩到下限以下应夹住: %d", got)
	}
	if got := statusBarClamp(1440, statusBarMinWinW); got != 1440 {
		t.Errorf("下限以上不该被动: %d", got)
	}
}

func TestStatusBarOptions(t *testing.T) {
	b := &StatusBar{}
	for _, o := range []StatusBarOption{
		WithStatusBarHeight(36),
		WithStatusBarGrip(false),
		WithStatusBarSizeReadout(),
		WithStatusBarMinWindowSize(640, 480),
	} {
		o(b)
	}
	if b.height != 36 || b.gripEnabled || !b.showSize || b.minW != 640 || b.minH != 480 {
		t.Fatalf("选项没落到配置上: %+v", b)
	}
	// 非法高度与非法下限要被忽略, 不能用 0 把布局项锁成不可见
	WithStatusBarHeight(2)(b)
	WithStatusBarMinWindowSize(0, -1)(b)
	if b.height != 36 || b.minW != 640 || b.minH != 480 {
		t.Fatalf("非法值不该覆盖已有配置: %+v", b)
	}
	ext := NewTooltip()
	WithStatusBarTooltip(ext)(b)
	if b.tip != ext {
		t.Error("外部提示管理器没被注入")
	}
}

func TestStatusBarItemOptions(t *testing.T) {
	clicked := false
	i := &StatusBarItem{}
	for _, o := range []StatusBarItemOption{
		WithStatusBarItemIcon("M1 1L2 2"),
		WithStatusBarItemTooltip("编码"),
		WithStatusBarItemOnClick(func() { clicked = true }),
		WithStatusBarItemDisabled(),
		WithStatusBarItemActive(),
		WithStatusBarItemWidth(96),
	} {
		o(i)
	}
	if i.icon == "" || i.tipText != "编码" || i.onClick == nil || !i.disabled || !i.active || i.fixedW != 96 {
		t.Fatalf("条目选项没落到配置上: %+v", i)
	}
	i.onClick()
	if !clicked {
		t.Error("条目点击回调没接上")
	}
	// 禁用态不响应点击, 但回调仍保留
	if got := (&StatusBarItem{disabled: true}).Disabled(); !got {
		t.Error("Disabled() 读值不对")
	}
	if got := (&StatusBarItem{}).Text(); got != "" {
		t.Errorf("空条目的 Text() = %q", got)
	}
}
