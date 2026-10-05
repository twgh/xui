package xui

import (
	"math"
	"strconv"
	"unicode"

	"github.com/twgh/xcgui/xc"
)

// 图标 path 数据(24x24 viewBox, lucide 原始数据, 与 shadcn 的默认图标库同源)。
// 其它图标可从 https://lucide.dev 复制 path 数据交给 Painter.Icon。
//
// 为什么不用 SVG 渲染: 炫彩在元素自绘回调里画图片/SVG 不生效(实测 XDraw_DrawSvgEx /
// XDraw_Image / XDraw_ImageEx 都画不出东西, 见 DESIGN.md 第八节), 所以这里用 drawx 的
// 线段图元把 path 复刻成折线。
// 支持 M/L/H/V/C/S/Q/T/Z 及相对指令(曲线按 8 段离散); 圆弧 A/a 不支持,
// 需要圆弧的图标用专门实现(见 Painter.IconSearch)。
const (
	IconChevronDown  = "m6 9 6 6 6-6"         // lucide ChevronDown
	IconChevronUp    = "m18 15-6-6-6 6"       // lucide ChevronUp
	IconChevronLeft  = "m15 18-6-6 6-6"       // lucide ChevronLeft
	IconChevronRight = "m9 18 6-6-6-6"        // lucide ChevronRight
	IconCheck        = "M20 6 9 17l-5-5"      // lucide Check
	IconX            = "M18 6 6 18M6 6l12 12" // lucide X
	IconMinus        = "M5 12h14"             // lucide Minus
)

// Icon 在 rc 内按 lucide 几何绘制一枚描边图标(等比铺满)。
// strokeWidth 用 24 网格单位, lucide 默认为 2。
func (p *Painter) Icon(paths string, rc *xc.RECT, color Color, strokeWidth float32) {
	w, h := rc.Right-rc.Left, rc.Bottom-rc.Top
	if paths == "" || w <= 0 || h <= 0 || color.IsNone() {
		return
	}
	size := w
	if h < size {
		size = h
	}
	ox := rc.Left + (w-size)/2 // 图标在 rc 内居中
	oy := rc.Top + (h-size)/2
	s := float32(size) / 24

	lineW := int32(math.Ceil(float64(strokeWidth * s))) // 线宽是物理像素: 取整会让 16px 图标(2*16/24=1.33)退化成 1px 细线
	if lineW < 1 {
		lineW = 1
	}
	p.Draw.SetLineWidth(lineW)
	p.Draw.SetBrushColor(uint32(color))

	for _, sub := range parseIconPath(paths) {
		for i := 1; i < len(sub); i++ {
			ax, ay := ox+round32(sub[i-1].x*s), oy+round32(sub[i-1].y*s)
			bx, by := ox+round32(sub[i].x*s), oy+round32(sub[i].y*s)
			if ax == bx && ay == by {
				continue
			}
			p.Draw.DrawLine(ax, ay, bx, by)
		}
	}
}

// IconSearch 绘制搜索图标(lucide Search: 半径 8 的圆 + 右下斜手柄)。
// 它用圆弧指令描述, 因此不走 parseIconPath, 直接用椭圆图元实现。
func (p *Painter) IconSearch(rc *xc.RECT, color Color, strokeWidth float32) {
	w, h := rc.Right-rc.Left, rc.Bottom-rc.Top
	if w <= 0 || h <= 0 || color.IsNone() {
		return
	}
	size := w
	if h < size {
		size = h
	}
	ox := rc.Left + (w-size)/2
	oy := rc.Top + (h-size)/2
	s := float32(size) / 24

	lineW := int32(math.Ceil(float64(strokeWidth * s))) // 线宽是物理像素: 取整会让 16px 图标(2*16/24=1.33)退化成 1px 细线
	if lineW < 1 {
		lineW = 1
	}
	p.Draw.SetLineWidth(lineW)
	p.Draw.SetBrushColor(uint32(color))

	r := round32(8 * s)
	cx, cy := ox+round32(11*s), oy+round32(11*s)
	circle := xc.RECT{Left: cx - r, Top: cy - r, Right: cx + r, Bottom: cy + r}
	p.Draw.DrawEllipse(&circle)
	p.Draw.DrawLine(ox+round32(21*s), oy+round32(21*s), ox+round32(16.7*s), oy+round32(16.7*s))
}

func round32(v float32) int32 { return int32(v + 0.5) }

type iconPt struct{ x, y float32 }

// iconScanner 是 path 字符串的游标扫描器。
type iconScanner struct {
	s   string
	pos int
}

func (sc *iconScanner) skipSep() {
	for sc.pos < len(sc.s) {
		switch sc.s[sc.pos] {
		case ' ', ',', '\n', '\r', '\t':
			sc.pos++
		default:
			return
		}
	}
}

// cmd 取下一个命令字母(没有则返回 false, 表示沿用上一个命令)。
func (sc *iconScanner) cmd() (byte, bool) {
	sc.skipSep()
	if sc.pos >= len(sc.s) {
		return 0, false
	}
	c := sc.s[sc.pos]
	if unicode.IsLetter(rune(c)) {
		sc.pos++
		return c, true
	}
	return 0, false
}

// num 取下一个数字(支持 "-4.3-4.3" 这类连写)。
func (sc *iconScanner) num() (float32, bool) {
	sc.skipSep()
	st := sc.pos
	if sc.pos < len(sc.s) && (sc.s[sc.pos] == '-' || sc.s[sc.pos] == '+') {
		sc.pos++
	}
	for sc.pos < len(sc.s) && (sc.s[sc.pos] >= '0' && sc.s[sc.pos] <= '9' || sc.s[sc.pos] == '.') {
		sc.pos++
	}
	if sc.pos == st {
		return 0, false
	}
	v, err := strconv.ParseFloat(sc.s[st:sc.pos], 32)
	if err != nil {
		return 0, false
	}
	return float32(v), true
}

// parseIconPath 把 SVG path 解析为若干条折线(24x24 网格坐标)。
func parseIconPath(d string) [][]iconPt {
	sc := &iconScanner{s: d}
	var (
		subs         [][]iconPt
		cur          []iconPt
		x, y, sx, sy float32
		cmd          byte
		lastC, lastQ iconPt
		hasC, hasQ   bool
	)
	push := func(px, py float32) { cur = append(cur, iconPt{px, py}) }
	flush := func() {
		if len(cur) > 1 {
			subs = append(subs, cur)
		}
		cur = nil
	}
	// 三次贝塞尔离散为折线
	bezier := func(x1, y1, x2, y2, x3, y3, x4, y4 float32) {
		const n = 8
		for k := 1; k <= n; k++ {
			t := float32(k) / n
			u := 1 - t
			push(
				u*u*u*x1+3*u*u*t*x2+3*u*t*t*x3+t*t*t*x4,
				u*u*u*y1+3*u*u*t*y2+3*u*t*t*y3+t*t*t*y4,
			)
		}
	}

	for sc.pos < len(sc.s) {
		if c, ok := sc.cmd(); ok {
			cmd = c
		} else if cmd == 0 {
			break
		} else if cmd == 'M' { // 隐式重复: M 后的多组坐标按 L 处理
			cmd = 'L'
		} else if cmd == 'm' {
			cmd = 'l'
		}
		rel := unicode.IsLower(rune(cmd))
		ok := true

		switch unicode.ToUpper(rune(cmd)) {
		case 'M':
			var ax, ay float32
			if ax, ok = sc.num(); ok {
				ay, ok = sc.num()
			}
			if !ok {
				break
			}
			flush()
			if rel {
				x, y = x+ax, y+ay
			} else {
				x, y = ax, ay
			}
			sx, sy = x, y
			push(x, y)
			hasC, hasQ = false, false
		case 'L':
			var ax, ay float32
			if ax, ok = sc.num(); ok {
				ay, ok = sc.num()
			}
			if !ok {
				break
			}
			if rel {
				x, y = x+ax, y+ay
			} else {
				x, y = ax, ay
			}
			push(x, y)
			hasC, hasQ = false, false
		case 'H':
			ax, o := sc.num()
			if !o {
				ok = false
				break
			}
			if rel {
				x += ax
			} else {
				x = ax
			}
			push(x, y)
		case 'V':
			ay, o := sc.num()
			if !o {
				ok = false
				break
			}
			if rel {
				y += ay
			} else {
				y = ay
			}
			push(x, y)
		case 'Z':
			push(sx, sy)
			flush()
			x, y = sx, sy
		case 'C', 'S', 'Q', 'T':
			ok = iconCurve(cmd, rel, sc, &x, &y, &lastC, &lastQ, &hasC, &hasQ, push, bezier)
		case 'A':
			// 圆弧不支持: 读出 7 个参数后直接跳到终点
			var v [7]float32
			for k := 0; k < 7; k++ {
				f, o := sc.num()
				if !o {
					ok = false
					break
				}
				v[k] = f
			}
			if ok {
				if rel {
					x, y = x+v[5], y+v[6]
				} else {
					x, y = v[5], v[6]
				}
				push(x, y)
			}
			hasC, hasQ = false, false
		default:
			return subs
		}
		if !ok {
			break
		}
	}
	flush()
	return subs
}

// iconCurve 处理三次/二次贝塞尔相关命令(C/S/Q/T), 返回是否解析成功。
func iconCurve(cmd byte, rel bool, sc *iconScanner,
	x, y *float32, lastC, lastQ *iconPt, hasC, hasQ *bool,
	push func(float32, float32), bezier func(float32, float32, float32, float32, float32, float32, float32, float32)) bool {

	// 参数个数: C=6, S/Q=4, T=2
	upper := unicode.ToUpper(rune(cmd))
	n := 2
	switch upper {
	case 'C':
		n = 6
	case 'S', 'Q':
		n = 4
	}
	var v [6]float32
	for k := 0; k < n; k++ {
		f, ok := sc.num()
		if !ok {
			return false
		}
		v[k] = f
	}
	ex, ey := v[n-2], v[n-1]
	if rel {
		ex, ey = *x+ex, *y+ey
	}

	switch upper {
	case 'C':
		c1x, c1y, c2x, c2y := v[0], v[1], v[2], v[3]
		if rel {
			c1x, c1y, c2x, c2y = *x+c1x, *y+c1y, *x+c2x, *y+c2y
		}
		bezier(*x, *y, c1x, c1y, c2x, c2y, ex, ey)
		*lastC, *hasC, *hasQ = iconPt{c2x, c2y}, true, false
	case 'S':
		c2x, c2y := v[0], v[1]
		c1x, c1y := *x, *y
		if rel {
			c2x, c2y = *x+c2x, *y+c2y
		}
		if *hasC {
			c1x, c1y = 2**x-lastC.x, 2**y-lastC.y
		}
		bezier(*x, *y, c1x, c1y, c2x, c2y, ex, ey)
		*lastC, *hasC, *hasQ = iconPt{c2x, c2y}, true, false
	case 'Q', 'T':
		var c1x, c1y float32
		if upper == 'Q' {
			c1x, c1y = v[0], v[1]
			if rel {
				c1x, c1y = *x+c1x, *y+c1y
			}
		} else {
			c1x, c1y = *x, *y
			if *hasQ {
				c1x, c1y = 2**x-lastQ.x, 2**y-lastQ.y
			}
		}
		// 二次转三次
		bezier(*x, *y,
			*x+2*(c1x-*x)/3, *y+2*(c1y-*y)/3,
			ex+2*(c1x-ex)/3, ey+2*(c1y-ey)/3,
			ex, ey)
		*lastQ, *hasQ, *hasC = iconPt{c1x, c1y}, true, false
	}
	*x, *y = ex, ey
	return true
}
