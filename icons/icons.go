// Package icons 是 xui 的 lucide 图标资源: 把 lucide-static 的 SVG 预处理成
// **一条 path 数据**(24 网格), 直接喂给 xui.Painter.Icon 即可出图。
//
//	p := xui.BeginPaint(hEle, hDraw)
//	p.Icon(icons.Path("eye"), &rc, xui.Current().Foreground, 2)
//
// 为什么单独一个包: 数据有 ~2000 个图标几百 KB, 只有 import 它的程序才会把数据链进
// 二进制; xui 运行时本身不依赖本包(资源与绘制解耦)。数据由 scripts/gen_icons.py 生成,
// 版本见 Version —— 与 shadcn 参考实现的 lucide-react 保持同一 lucide 版本。
//
// 名字就是 lucide 的 kebab-case 名(如 "chevron-down"、"circle-alert"); SVG 目录里的
// 废弃别名(lucide 保留的旧名, 如 "alert-circle")也在其中。
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
