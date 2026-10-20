package xui

import (
	"math"

	"github.com/twgh/xcgui/svgp"
	"github.com/twgh/xcgui/xc"
)

// ==================== 图标 ====================
//
// 图标数据从哪来(全在子包 github.com/twgh/xui/icons, 由 scripts/gen_icons.py 生成):
//
//	p.Icon(icons.IconChevronDown, &rc, color, 2) // 常量: 编译期内联, 零体积成本(推荐)
//	p.Icon(icons.Path("star"), &rc, color, 2)    // 按名字取: 会把整张资源表链进二进制
//
// 每个图标都有一个 `icons.IconXxx` 常量(名字是 lucide 名的驼峰形式, 见子包文档),
// 只用常量的程序**一个字节都不多花**; `icons.Path` 是给"名字来自用户输入/配置"的场合
// 准备的, 代价是整张表(2130 个图标约 350KB)跟着进二进制。
//
// 手里是**整段 SVG**(heroicons / feather / iconfont 复制来的)也交给同一个 Icon ——
// 见 iconsvg.go 的自动识别。path 语法解析与 SVG 文档遍历在 **github.com/twgh/xcgui/svgp**
// (纯 Go, 不经过 xcgui.dll 的 svg 支持), 本文件只管"图标语义": 尺寸/位置、折线回落。
//
// 出图有两条路(见 Painter.Icon):
//  1. D2D 直通绘制(d2dicon.go): 解析出的折线 → D2D 几何(描边子路径走 DrawGeometry + 圆角样式,
//     填充色点走 FillGeometry), 抗锯齿、几何精确, 是当前的主路径;
//  2. 折线描边(DrawLine)回落: 只在 GDI 渲染(app.New(false))或 D2D 资源创建失败时使用。

// Icon 在 rc 内绘制一枚**描边图标**(等比铺满)。
//
// data 两种形态, **自动识别**(跳过前导空白看首字符是不是 `<`):
//
//   - **path 数据**: **24x24 网格**的描边型 path, 例如 icons.IconXxx / icons.Path("eye") / 自己拼的
//     path。子路径想按**填充**画时用 SubPath.Filled 标注(整段 SVG 会自动判定); 非 24 网格的请先把
//     坐标缩放过去(整段 SVG 则不用, 见下)。
//   - **整段 SVG**: 从图标站复制来的 `<svg viewBox="…"><path d="…"/></svg>`(heroicons / feather /
//     iconfont…)—— 自动提取 `<path>`、按 viewBox **归一到 24 网格**(任意网格)、按 fill/stroke 判定
//     填充子路径。归一化结果**按 SVG 原文缓存**(最多 512 条), 每帧只多一次首字符判断。
//
// strokeWidth 用 24 网格单位; 传 <= 0 时: SVG 用它自己的 stroke-width(里面也没有则 2),
// path 数据按 2(lucide 惯例)。想先校验 SVG 或自己缓存归一化结果, 用 ParseIconSVG。
//
// 出图优先走 D2D 几何管线(有抗锯齿, 见 d2dicon.go); 只有 GDI 渲染模式(app.New(false))或
// D2D 资源建不出来时才回落到折线描边。
//
// scale(可选, 默认 1)是以绘制方块中心为基准的缩放, 用于"按下变小"这类按压反馈 ——
// 缩放走渲染目标变换矩阵, 不会击穿几何缓存。折线回落路径忽略它(GDI 侧没有变换)。
func (p *Painter) Icon(data string, rc *xc.RECT, color Color, strokeWidth float32, scale ...float32) {
	sc := float32(1)
	if len(scale) > 0 && scale[0] > 0 {
		sc = scale[0]
	}
	paths := data
	if isIconSVG(data) {
		v := parseIconSVGCached(data)
		if !v.ok || v.paths == "" {
			return // 解析失败什么都不画(想先校验用 ParseIconSVG)
		}
		paths = v.paths
		if strokeWidth <= 0 {
			strokeWidth = v.sw
		}
	}
	if strokeWidth <= 0 {
		strokeWidth = 2
	}
	if p.d2dIcon(paths, rc, color, strokeWidth, func() []svgp.SubPath {
		return parseIconPaths(paths)
	}, sc) {
		return
	}
	ox, oy, size, ok := iconBox(paths, rc, color)
	if !ok {
		return
	}

	// 回落: 折线描边(旧实现。点坐标取整到逻辑像素 + 整数线宽, 小尺寸下不如几何精确,
	// 但 GDI 渲染模式(app.New(false))没有 D2D 目标, 得有这条兜底。
	// 填充子路径在这里只能画轮廓 —— GDI 侧没有多边形填充, 只能近似)
	s := float32(size) / 24
	p.Draw.SetLineWidth(iconLineWidth(strokeWidth, s))
	p.Draw.SetBrushColor(uint32(color))
	for _, sub := range parseIconPaths(paths) {
		n := len(sub.Points)
		for i := 1; i < n; i++ {
			ax, ay := ox+round32(sub.Points[i-1].X*s), oy+round32(sub.Points[i-1].Y*s)
			bx, by := ox+round32(sub.Points[i].X*s), oy+round32(sub.Points[i].Y*s)
			if ax == bx && ay == by {
				continue
			}
			p.Draw.DrawLine(ax, ay, bx, by)
		}
		if sub.Closed && n >= 2 { // Z 收尾: 补一条首尾连线(svgp 不往点集里塞重复点)
			ax, ay := ox+round32(sub.Points[n-1].X*s), oy+round32(sub.Points[n-1].Y*s)
			bx, by := ox+round32(sub.Points[0].X*s), oy+round32(sub.Points[0].Y*s)
			if ax != bx || ay != by {
				p.Draw.DrawLine(ax, ay, bx, by)
			}
		}
	}
}

// parseIconPaths 宽松解析(忽略错误, 用已解析出的部分) —— 与折线回落的旧行为一致。
func parseIconPaths(d string) []svgp.SubPath {
	subs, _ := svgp.ParsePath(d)
	return subs
}

// iconBox 计算图标在 rc 内的正方形区域(等比铺满, 居中)。
func iconBox(paths string, rc *xc.RECT, color Color) (ox, oy, size int32, ok bool) {
	w, h := rc.Right-rc.Left, rc.Bottom-rc.Top
	if paths == "" || w <= 0 || h <= 0 || color.IsNone() {
		return 0, 0, 0, false
	}
	size = w
	if h < size {
		size = h
	}
	return rc.Left + (w-size)/2, rc.Top + (h-size)/2, size, true
}

// iconLineWidth 是折线回落路径的线宽(物理像素): 取整会让 16px 图标(2*16/24=1.33)退化成 1px 细线。
func iconLineWidth(strokeWidth, s float32) int32 {
	lw := int32(math.Ceil(float64(strokeWidth * s)))
	if lw < 1 {
		lw = 1
	}
	return lw
}

func round32(v float32) int32 { return int32(v + 0.5) }
