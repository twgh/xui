package xui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/twgh/xcgui/svgp"
)

// ==================== SVG 文本 → 图标 path ====================
//
// 调用方手里的图标数据往往是一整段 SVG(从 heroicons / feather / iconfont 复制来的
// `<svg viewBox="0 0 24 24"><path d="…"/></svg>`), 而本仓库的图标管线吃的是"24 网格的 path 数据"。
//
// SVG 的解析(元素遍历 / path 语法 / viewBox / fill-stroke 判定)在 **github.com/twgh/xcgui/svgp**
// (纯 Go, 不经过 xcgui.dll 的 svg 支持); 本文件只做两件 xui 特有的事:
//
//	1. 归一到 **24 网格**(图标尺寸约定);
//	2. 把"填充子路径"编码成 path 字符串里的 `!` 前缀(内部约定, 见 parseIconPath)。

// ParseIconSVG 把一整段 SVG 文本转成图标 path 数据。
//
// 返回:
//   - paths: 归一化后的 path(绝对坐标、缩放到 **24x24 网格**、填充子路径带 `!` 前缀、闭合子路径带 `Z`),
//     可直接喂给 Painter.Icon, 也可以自己缓存起来复用;
//   - strokeWidth: 建议线宽(**24 网格单位**): 取 SVG 里的 stroke-width(按 viewBox 折算), 没有则 2
//     (lucide 惯例)。想让调用方决定线宽就忽略它。
//
// 支持哪些元素/属性见 svgp 包文档(认 svg/g/path, 继承 fill/stroke/stroke-width; 忽略形状元素与
// transform 属性)。
func ParseIconSVG(svg string) (paths string, strokeWidth float32, err error) {
	doc, err := svgp.ParseDoc(svg)
	if err != nil {
		return "", 0, err
	}
	subs := svgp.ScaleTo(doc.Paths, doc.ViewBox, 24) // 任意网格 → 24

	var b strings.Builder
	needSep := false // 子路径之间用一个空格分隔(不在每条子路径尾部留空格)
	for _, sub := range subs {
		if len(sub.Points) < 2 {
			continue
		}
		if needSep {
			b.WriteByte(' ')
		}
		needSep = true
		if sub.Filled {
			b.WriteByte('!') // 项目内部约定: 接下来的子路径按填充画
		}
		for i, pt := range sub.Points {
			if i == 0 {
				b.WriteByte('M')
			} else {
				b.WriteByte('L')
			}
			b.WriteString(svgNum(float64(pt.X)))
			b.WriteByte(' ')
			b.WriteString(svgNum(float64(pt.Y)))
			if i != len(sub.Points)-1 {
				b.WriteByte(' ')
			}
		}
		if sub.Closed {
			b.WriteString(" Z") // 闭合语义要带过去(几何那边靠它决定 EndFigure)
		}
	}
	if b.Len() == 0 {
		return "", 0, fmt.Errorf("xui: SVG 里没有可用的子路径")
	}
	if strokeWidth = float32(doc.StrokeWidth * svgp.ScaleFactor(doc.ViewBox, 24)); strokeWidth <= 0 {
		strokeWidth = 2 // lucide 惯例
	}
	return b.String(), strokeWidth, nil
}

// svgNum 输出坐标: 定点两位小数, 去掉多余的 0(24 网格下 0.01 的精度远超像素需求)。
func svgNum(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	if s == "-0" {
		s = "0"
	}
	return s
}

// ---------------- 绘制入口用: 格式识别 + 解析缓存 ----------------

// isIconSVG 判断 data 是"整段 SVG"还是"path 数据": 跳过前导空白后看首字符是不是 `<`。
//
// path 数据以 M/m/L/!/数字开头, 不会与它冲突; 绘制回调每帧都会调一次, 成本可忽略
// (Painter.Icon 用它决定要不要走 SVG 归一化)。
func isIconSVG(data string) bool {
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case ' ', '\t', '\n', '\r':
			continue
		case '<':
			return true
		}
		return false
	}
	return false
}

type svgParsed struct {
	paths string
	sw    float32
	ok    bool
}

var (
	svgCacheMu sync.Mutex
	svgCache   = make(map[string]svgParsed)
)

// parseIconSVGCached 按 SVG 原文缓存归一化结果 —— 绘制回调每帧都会调 Icon, 不能每帧重新解析。
func parseIconSVGCached(svg string) svgParsed {
	svgCacheMu.Lock()
	v, hit := svgCache[svg]
	svgCacheMu.Unlock()
	if hit {
		return v
	}
	paths, sw, err := ParseIconSVG(svg)
	v = svgParsed{paths: paths, sw: sw, ok: err == nil}
	svgCacheMu.Lock()
	if len(svgCache) >= 512 { // 图标种类有限, 满了整体丢弃重来, 不做 LRU
		svgCache = make(map[string]svgParsed)
	}
	svgCache[svg] = v
	svgCacheMu.Unlock()
	return v
}
