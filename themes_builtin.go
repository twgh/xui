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
		CardFore:    Hex("#0a0a0a"),
		Popover:     Hex("#ffffff"),
		PopoverFore: Hex("#0a0a0a"),

		Primary:         Hex("#171717"),
		PrimaryFore:     Hex("#fafafa"),
		Secondary:       Hex("#f5f5f5"),
		SecondaryFore:   Hex("#171717"),
		Muted:           Hex("#f5f5f5"),
		MutedFore:       Hex("#737373"),
		Accent:          Hex("#f5f5f5"),
		AccentFore:      Hex("#171717"),
		Destructive:     Hex("#e7000b"),
		DestructiveFore: Hex("#ffffff"),

		Border: Hex("#e5e5e5"),
		Input:  Hex("#e5e5e5"),
		Ring:   Hex("#a3a3a3"),
		Radius: 10,

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
		CardFore:    Hex("#fafafa"),
		Popover:     Hex("#171717"),
		PopoverFore: Hex("#fafafa"),

		Primary:         Hex("#e5e5e5"),
		PrimaryFore:     Hex("#171717"),
		Secondary:       Hex("#262626"),
		SecondaryFore:   Hex("#fafafa"),
		Muted:           Hex("#262626"),
		MutedFore:       Hex("#a3a3a3"),
		Accent:          Hex("#262626"),
		AccentFore:      Hex("#fafafa"),
		Destructive:     Hex("#ff6467"),
		DestructiveFore: Hex("#ffffff"),

		// 白色 10% / 15% 透明, 与 shadcn dark 的 border/input 定义一致
		Border: Color(xc.RGBA(255, 255, 255, 26)),
		Input:  Color(xc.RGBA(255, 255, 255, 38)),
		Ring:   Hex("#737373"),
		Radius: 10,

		FontDefault: FontSpec{Family: "微软雅黑", Size: 14},
		FontMono:    FontSpec{Family: "Consolas", Size: 14},
	}
}
