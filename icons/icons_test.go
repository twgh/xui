package icons

import "testing"

// TestConsts 守住两件靠生成器保证、但要有人看着的事:
//  1. 常量与名字表是**同一份数据**(表里存的就是常量本身), 且组件要用的那几个都在;
//  2. "数字挨数字"的名字不会撞成同一个常量 —— 直接引用常量名本身就是这条断言的锚点:
//     生成器的驼峰规则一改(或漏了分隔符), 这份文件就编译不过, 立刻会被发现。
func TestConsts(t *testing.T) {
	// 组件用的图标(ComboBox/菜单的箭头与勾选、Dialog 关闭、标题栏的 eye/moon/minus/x)
	// + 名字里"数字挨数字"的几对。注意那几对在 lucide 里是**别名关系**(数据完全相同),
	// 但常量必须是两个: arrow-down-0-1 → IconArrowDown0_1, arrow-down-01 → IconArrowDown01。
	cases := map[string]string{
		"chevron-down":   IconChevronDown,
		"chevron-up":     IconChevronUp,
		"chevron-left":   IconChevronLeft,
		"chevron-right":  IconChevronRight,
		"check":          IconCheck,
		"x":              IconX,
		"minus":          IconMinus,
		"eye":            IconEye,
		"eye-off":        IconEyeOff,
		"sun":            IconSun,
		"moon":           IconMoon,
		"arrow-down-0-1": IconArrowDown0_1,
		"arrow-down-01":  IconArrowDown01,
		"axis-3-d":       IconAxis3D,
		"axis-3d":        IconAxis3d,
		"grid-2-x-2":     IconGrid2X2,
		"grid-2x2":       IconGrid2x2,
	}
	for name, d := range cases {
		if d == "" {
			t.Errorf("%s: 常量为空", name)
			continue
		}
		if !Has(name) {
			t.Errorf("%s: 表里没有这个名字", name)
			continue
		}
		if got := Path(name); got != d {
			t.Errorf("%s: 按名字取到的数据与常量不一致(生成器接线错了?)", name)
		}
	}
	if n := Count(); n < 2000 {
		t.Errorf("图标只有 %d 个, 资源不完整", n)
	}
}
