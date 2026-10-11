package xui

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/twgh/xcgui/xc"
)

// ==================== SVG 文本 → 图标 path ====================
//
// 调用方手里的图标数据往往是一整段 SVG(从 heroicons / feather / iconfont 复制来的
// `<svg viewBox="0 0 24 24"><path d="…"/></svg>`), 而本仓库的图标管线吃的是"24 网格的 path 数据"。
// 本文件负责那一步转换:
//
//	<svg 文本> --ParseIconSVG--> 归一化 path(绝对坐标、24 网格、填充子路径加 `!` 前缀) --> Painter.Icon
//
// 归一化结果与 icons 包里的数据同形式, 所以几何缓存、元素 alpha、D2D 直通/折线回落全都照旧生效。

// svgPathData 从 SVG 里提取到的一条 <path>。
type svgPathData struct {
	d      string
	filled bool
}

// ParseIconSVG 把一整段 SVG 文本转成图标 path 数据。
//
// 返回:
//   - paths: 归一化后的 path(绝对坐标、缩放到 **24x24 网格**、填充子路径带 `!` 前缀),
//     可直接喂给 Painter.Icon, 也可以自己缓存起来复用;
//   - strokeWidth: 建议线宽(**24 网格单位**): 取 SVG 里的 stroke-width(按 viewBox 折算), 没有则 2
//     (lucide 惯例)。想让调用方决定线宽就忽略它。
//
// 支持:
//   - 多个 <path> 合并(先出现的先画);
//   - <g> 嵌套并继承 fill/stroke/stroke-width;
//   - viewBox(任意网格, 按较长边缩放到 24 并保持比例; min-x/min-y 会平掉)或退化成 width/height;
//   - 注释、任意空白、`d` 里的紧凑写法(省略逗号/空格, 与官方 SVG 一致)。
//
// 不支持(碰到就忽略, 不报错):
//   - <circle>/<rect>/<line>/<polyline> 等形状元素 —— 先自己转成 path(或提出 issue);
//   - transform 属性、CSS 样式表(class 里的 fill/stroke)。
//
// 判定"填充还是描边": 该 path(含 <g> 继承)的 stroke 为空或 "none" 且 fill 非空且非 "none" → 填充;
// 其余(尤其 fill="none" stroke="currentColor")→ 描边 —— 图标集里后者是绝对主流。
func ParseIconSVG(svg string) (paths string, strokeWidth float32, err error) {
	items, scale, minX, minY, sw, err := svgCollect(svg)
	if err != nil {
		return "", 0, err
	}
	if len(items) == 0 {
		return "", 0, fmt.Errorf("xui: SVG 里没有可用的 <path d=…>")
	}

	// 逐条子路径重写成绝对坐标(顺带按 viewBox 缩放/平移), 填充子路径加 `!` 前缀。
	var b strings.Builder
	needSep := false // 子路径之间用一个空格分隔(不在每条子路径尾部留空格)
	for _, it := range items {
		subs, complete := parseIconPathEx(it.d)
		if !complete || len(subs) == 0 {
			continue // 单条 path 解析不动就跳过它, 不牵连整张图
		}
		for _, sub := range subs {
			if len(sub.pts) < 2 {
				continue
			}
			if needSep {
				b.WriteByte(' ')
			}
			needSep = true
			if it.filled {
				b.WriteByte('!') // 项目内部约定: 接下来的子路径按填充画
			}
			for i, pt := range sub.pts {
				if i == 0 {
					b.WriteByte('M')
				} else {
					b.WriteByte('L')
				}
				b.WriteString(svgNum(float64(pt.x)*scale - minX*scale))
				b.WriteByte(' ')
				b.WriteString(svgNum(float64(pt.y)*scale - minY*scale))
				if i != len(sub.pts)-1 {
					b.WriteByte(' ')
				}
			}
		}
	}
	if b.Len() == 0 {
		return "", 0, fmt.Errorf("xui: SVG 的 path 数据解析不出来")
	}
	if strokeWidth = sw; strokeWidth <= 0 {
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

// svgCollect 扫一遍 SVG, 收集所有 <path>, 并算出缩放(→24 网格)与建议线宽(→24 网格单位)。
func svgCollect(svg string) (items []svgPathData, scale, minX, minY float64, strokeWidth float32, err error) {
	scale, minX, minY = 1, 0, 0

	dec := xml.NewDecoder(strings.NewReader(svg))
	dec.Strict = false // SVG 里常有未声明的实体/宽松写法

	// <g> 的属性继承栈
	type inherited struct{ fill, stroke, sw string }
	stack := []inherited{{}}
	cur := func() inherited { return stack[len(stack)-1] }
	get := func(a []xml.Attr, name string) string {
		for _, at := range a {
			if strings.EqualFold(at.Name.Local, name) {
				return strings.TrimSpace(at.Value)
			}
		}
		return ""
	}
	hasViewBox, hasSize := false, false

	for {
		tok, e := dec.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, 1, 0, 0, 0, fmt.Errorf("xui: SVG 解析失败: %w", e)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch strings.ToLower(t.Name.Local) {
			case "svg":
				if vb := get(t.Attr, "viewBox"); vb != "" {
					if f := svgNums(vb); len(f) == 4 && f[2] > 0 && f[3] > 0 {
						scale = 24 / max64(f[2], f[3]) // 按较长边缩放, 保持比例
						minX, minY = f[0], f[1]
						hasViewBox = true
					}
				}
				if !hasViewBox && !hasSize {
					w := svgLen(get(t.Attr, "width"))
					h := svgLen(get(t.Attr, "height"))
					if w > 0 && h > 0 {
						scale = 24 / max64(w, h)
						hasSize = true
					}
				}
				// 线宽也可能写在根 <svg> 上
				if v := svgLen(get(t.Attr, "stroke-width")); v > 0 {
					strokeWidth = float32(v)
					stack[len(stack)-1].sw = get(t.Attr, "stroke-width")
				}
			case "g":
				n := cur()
				if v := get(t.Attr, "fill"); v != "" {
					n.fill = v
				}
				if v := get(t.Attr, "stroke"); v != "" {
					n.stroke = v
				}
				if v := get(t.Attr, "stroke-width"); v != "" {
					n.sw = v
				}
				stack = append(stack, n)
			case "path":
				d := get(t.Attr, "d")
				if d == "" {
					continue
				}
				n := cur()
				fill, stroke, sws := n.fill, n.stroke, n.sw
				if v := get(t.Attr, "fill"); v != "" {
					fill = v
				}
				if v := get(t.Attr, "stroke"); v != "" {
					stroke = v
				}
				if v := get(t.Attr, "stroke-width"); v != "" {
					sws = v
				}
				if strokeWidth <= 0 {
					if v := svgLen(sws); v > 0 {
						strokeWidth = float32(v)
					}
				}
				filled := fill != "" && !strings.EqualFold(fill, "none") &&
					(stroke == "" || strings.EqualFold(stroke, "none"))
				items = append(items, svgPathData{d: d, filled: filled})
			}
		case xml.EndElement:
			if strings.EqualFold(t.Name.Local, "g") && len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	// 线宽是"源网格单位", 折算到 24 网格
	if strokeWidth > 0 {
		strokeWidth = float32(float64(strokeWidth) * scale)
	}
	return items, scale, minX, minY, strokeWidth, nil
}

// svgNums 从 "0 0 24 24"(也容忍逗号)里取一串数字。
func svgNums(s string) []float64 {
	s = strings.ReplaceAll(s, ",", " ")
	fields := strings.Fields(s)
	out := make([]float64, 0, len(fields))
	for _, f := range fields {
		if v, err := strconv.ParseFloat(f, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

// svgLen 解析长度值("24" / "24px" / "1.5"), 拿不到就返回 0。
func svgLen(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "px")
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return 0
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ---------------- 绘制入口(带解析缓存) ----------------

type svgParsed struct {
	paths string
	sw    float32
	ok    bool
}

var (
	svgCacheMu sync.Mutex
	svgCache   = make(map[string]svgParsed)
)

// parseIconSVGCached 按 SVG 原文缓存归一化结果 —— 绘制回调每帧都会调 IconSVG, 不能每帧重新解析。
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

// IconSVG 用一整段 SVG 文本在 rc 内画图标(等价于 Icon(ParseIconSVG(svg)) + 缓存)。
//
//   - svg 可以是从图标站复制的完整 `<svg viewBox="…">…</svg>`, 也可以是裸的 `<path d="…"/>` 片段;
//   - strokeWidth <= 0 时用 SVG 里的 stroke-width(已折算到 24 网格), 都没有则 2;
//   - 解析失败就什么都不画(想先校验用 ParseIconSVG)。
//
// 性能: 归一化结果按 svg 原文缓存(最多 512 条), 每帧只多一次 map 查找; 几何本身照旧走图标几何缓存。
func (p *Painter) IconSVG(svg string, rc *xc.RECT, color Color, strokeWidth float32) {
	if p == nil {
		return
	}
	v := parseIconSVGCached(svg)
	if !v.ok || v.paths == "" {
		return
	}
	if strokeWidth <= 0 {
		strokeWidth = v.sw
	}
	p.Icon(v.paths, rc, color, strokeWidth)
}
