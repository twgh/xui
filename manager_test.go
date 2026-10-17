package xui

import "testing"

// 回归: OnThemeChange 返回的反注册函数必须真的生效。
// 旧实现用 `&f == &fn` 比较函数地址来判断要删哪一条, 恒为假 → unsub 是空操作。
func TestOnThemeChangeUnsubscribe(t *testing.T) {
	defer SetTheme(Light())

	var a, b int
	unA := OnThemeChange(func(*Theme) { a++ })
	unB := OnThemeChange(func(*Theme) { b++ })
	defer unB()

	SetTheme(Dark())
	unA()
	SetTheme(Light())

	if a != 1 {
		t.Fatalf("反注册后仍收到回调: a=%d, want 1", a)
	}
	if b != 2 {
		t.Fatalf("反注册 A 误删了 B: b=%d, want 2", b)
	}
}

// 反注册函数应幂等: 重复调用不 panic, 也不影响其它订阅。
func TestOnThemeChangeUnsubIdempotent(t *testing.T) {
	defer SetTheme(Light())

	var n int
	un := OnThemeChange(func(*Theme) { n++ })
	un()
	un()
	SetTheme(Dark())

	if n != 0 {
		t.Fatalf("已反注册的订阅仍被调用: n=%d", n)
	}
}

// 内置主题必须把 --chart-1..5 / --sidebar-* 也填上: 这两组令牌组件库自己不用,
// 但用户应用从主题里取用 —— 漏填会静默变成黑色(机械转译新主题时最容易漏)。
func TestBuiltinThemeChartSidebarTokens(t *testing.T) {
	for _, th := range []*Theme{Light(), Dark()} {
		tokens := map[string]Color{
			"Chart1": th.Chart1, "Chart2": th.Chart2, "Chart3": th.Chart3,
			"Chart4": th.Chart4, "Chart5": th.Chart5,
			"Sidebar": th.Sidebar, "SidebarForeground": th.SidebarForeground,
			"SidebarPrimary": th.SidebarPrimary, "SidebarPrimaryForeground": th.SidebarPrimaryForeground,
			"SidebarAccent": th.SidebarAccent, "SidebarAccentForeground": th.SidebarAccentForeground,
			"SidebarBorder": th.SidebarBorder, "SidebarRing": th.SidebarRing,
		}
		for name, c := range tokens {
			if c.IsNone() {
				t.Fatalf("%s 主题的 %s 令牌为空", th.Name, name)
			}
		}
	}
}
