// Package icons 是 xui 的 lucide 图标资源: 把 lucide-static 的 SVG 预处理成
// **一条 path 数据**(24 网格), 直接喂给 xui.Painter.Icon 即可出图。
//
// 两种用法:
//
//	p.Icon(icons.IconChevronDown, &rc, xui.Current().Foreground, 2) // 常量: 推荐
//	p.Icon(icons.Path("star"), &rc, xui.Current().Foreground, 2)    // 按名字取
//
// 每个图标都有一个 `IconXxx` 常量(名字是 lucide kebab 名的驼峰形式: chevron-down →
// IconChevronDown、eye-off → IconEyeOff)。常量**按需链接**: 只用几个常量的程序会被
// 编译期内联到调用点, 实测体积增量 0 字节; 而 Path/Has/Names 引用整张表, 只要调用就会把
// 2130 条数据(约 350KB)带进二进制(实测 +426KB)。所以名字来自用户输入/配置时用 Path,
// 其余场合用常量。
//
// 为什么单独一个包: 资源与绘制运行时解耦(xui 运行时本身不依赖本包), 常量才能按需链接 ——
// 数据若以运行时常量表的形式放在 xui 里, 每个用户程序都得背它。数据由 scripts/gen_icons.py
// 生成, 版本见 Version —— 与 shadcn 参考实现的 lucide-react 保持同一 lucide 版本;
// 常量与名字表由生成器一起产出(表里存的就是常量本身), 不存在"手抄漂移"。
//
// 名字就是 lucide 的 kebab-case 名(如 "chevron-down"、"circle-alert"); SVG 目录里的
// 废弃别名(lucide 保留的旧名, 如 "alert-circle")也在其中, 一样有常量(IconAlertCircle)。
// 少数名字转驼峰会撞车(arrow-down-0-1 与 arrow-down-01、axis-3-d 与 axis-3d 都同时存在),
// 生成器在"数字段紧跟数字段"处补下划线区分(IconArrowDown0_1 / IconArrowDown01)。
package icons

import "sort"

// Path 按名字取图标的 path 数据, 名字不存在时返回 ""。
//
// 返回的数据是 24 网格下的完整 path(相对指令已绝对化, 多段子路径用空格分隔;
// `!` 前缀的子路径表示填充, 见 lucide_gen.go 头部说明)。空串会被 Painter.Icon 忽略,
// 所以名字拼错只会"不画", 不会崩 —— 需要提前校验用 Has。
func Path(name string) string {
	i := sort.Search(len(all), func(i int) bool { return all[i].name >= name })
	if i < len(all) && all[i].name == name {
		return all[i].d
	}
	return ""
}

// Has 报告是否存在该名字的图标。
func Has(name string) bool {
	i := sort.Search(len(all), func(i int) bool { return all[i].name >= name })
	return i < len(all) && all[i].name == name
}

// Names 返回全部图标名(字典序升序)。返回的是副本, 调用方可随意修改/排序。
func Names() []string {
	out := make([]string, len(all))
	for i := range all {
		out[i] = all[i].name
	}
	return out
}

// Count 是图标总数。
func Count() int { return len(all) }
