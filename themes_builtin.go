package xui

import "github.com/twgh/xcgui/xc"

// 内置主题, 由 shadcn-ui nova 预设(neutral 色板, oklch 值已换算 hex)机械转译而来。
// 这些主题是拷贝模式的一部分: 用户 init 时会得到一份自己的 theme 副本随意修改。

// Light 亮色主题。
func Light() *Theme {
	return &Theme{
		Name:  "light",
		Background: Hex("#ffffff"),
		Foreground: Hex("#0a0a0a"),

		Card:        Hex("#ffffff"),
		CardForeground:    Hex("#0a0a0a"),
		Popover:     Hex("#ffffff"),
		PopoverForeground: Hex("#0a0a0a"),

		Primary:         Hex("#171717"),
		PrimaryForeground:     Hex("#fafafa"),
		Secondary:       Hex("#f5f5f5"),
		SecondaryForeground:   Hex("#171717"),
		Muted:           Hex("#f5f5f5"),
		MutedForeground:       Hex("#737373"),
		Accent:          Hex("#f5f5f5"),
		AccentForeground:      Hex("#171717"),
		Destructive:     Hex("#e7000b"),
		DestructiveForeground: Hex("#ffffff"),

		Border: Hex("#e5e5e5"),
		Input:  Hex("#e5e5e5"),
		Ring:   Hex("#a3a3a3"),

		// --chart-1..5 与 --sidebar-*(oklch, 无彩度灰阶)换算
		Chart1: Hex("#d4d4d4"),
		Chart2: Hex("#737373"),
		Chart3: Hex("#525252"),
		Chart4: Hex("#404040"),
		Chart5: Hex("#262626"),

		Sidebar:                  Hex("#fafafa"),
		SidebarForeground:        Hex("#0a0a0a"),
		SidebarPrimary:           Hex("#171717"),
		SidebarPrimaryForeground: Hex("#fafafa"),
		SidebarAccent:            Hex("#f5f5f5"),
		SidebarAccentForeground:  Hex("#171717"),
		SidebarBorder:            Hex("#e5e5e5"),
		SidebarRing:              Hex("#a1a1a1"),
		Radius:                   10,

		FontDefault: FontSpec{Family: "微软雅黑", Size: 14},
		FontMono:    FontSpec{Family: "Consolas", Size: 14},
	}
}

// Dark 暗色主题。Border/Input 是半透明白(与 shadcn dark 定义一致), 可自适应叠放的表面。
func Dark() *Theme {
	return &Theme{
		Name:  "dark",
		Background: Hex("#0a0a0a"),
		Foreground: Hex("#fafafa"),

		Card:        Hex("#171717"),
		CardForeground:    Hex("#fafafa"),
		Popover:     Hex("#171717"),
		PopoverForeground: Hex("#fafafa"),

		Primary:         Hex("#e5e5e5"),
		PrimaryForeground:     Hex("#171717"),
		Secondary:       Hex("#262626"),
		SecondaryForeground:   Hex("#fafafa"),
		Muted:           Hex("#262626"),
		MutedForeground:       Hex("#a3a3a3"),
		Accent:          Hex("#262626"),
		AccentForeground:      Hex("#fafafa"),
		Destructive:     Hex("#ff6467"),
		DestructiveForeground: Hex("#ffffff"),

		// 白色 10% / 15% 透明, 与 shadcn dark 的 border/input 定义一致
		Border: Color(xc.RGBA(255, 255, 255, 26)),
		Input:  Color(xc.RGBA(255, 255, 255, 38)),
		Ring:   Hex("#737373"),

		// chart 色板亮暗自带同一组值(shadcn dark 未覆写 --chart-*); sidebar 暗色自有一套
		Chart1: Hex("#d4d4d4"),
		Chart2: Hex("#737373"),
		Chart3: Hex("#525252"),
		Chart4: Hex("#404040"),
		Chart5: Hex("#262626"),

		Sidebar:                  Hex("#171717"),
		SidebarForeground:        Hex("#fafafa"),
		SidebarPrimary:           Hex("#1447e6"), // oklch(0.488 0.243 264.376) 唯一带彩度的 sidebar 令牌
		SidebarPrimaryForeground: Hex("#fafafa"),
		SidebarAccent:            Hex("#262626"),
		SidebarAccentForeground:  Hex("#fafafa"),
		SidebarBorder:            Color(xc.RGBA(255, 255, 255, 26)), // oklch(1 0 0 / 10%)
		SidebarRing:              Hex("#737373"),
		Radius:                   10,

		FontDefault: FontSpec{Family: "微软雅黑", Size: 14},
		FontMono:    FontSpec{Family: "Consolas", Size: 14},
	}
}
