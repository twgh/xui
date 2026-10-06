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
// 支持 M/L/H/V/C/S/Q/T/Z 及相对指令(曲线按 8 段离散); 圆弧 A/a 转为
// 多段三次贝塞尔(每段圆心角 <= 90°, 见 arcToBeziers)。
const (
	IconChevronDown  = "m6 9 6 6 6-6"         // lucide ChevronDown
	IconChevronUp    = "m18 15-6-6-6 6"       // lucide ChevronUp
	IconChevronLeft  = "m15 18-6-6 6-6"       // lucide ChevronLeft
	IconChevronRight = "m9 18 6-6-6-6"        // lucide ChevronRight
	IconCheck        = "M20 6 9 17l-5-5"      // lucide Check
	IconX            = "M18 6 6 18M6 6l12 12" // lucide X
	IconMinus        = "M5 12h14"             // lucide Minus

	IconEye    = "M2.062 12.348a1 1 0 0 1 0-.639 10.365 10.365 0 0 1 19.876 0 1 1 0 0 1 0 .639 10.365 10.365 0 0 1-19.876 0M12 9a3 3 0 1 0 0 6a3 3 0 1 0 0-6"                                                                                               // lucide Eye
	IconEyeOff = "M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49M14.084 14.158a3 3 0 0 1-4.242-4.242M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143M2 2l20 20" // lucide EyeOff
	IconSun    = "M12 8a4 4 0 1 0 0 8a4 4 0 1 0 0-8M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"                                                                                    // lucide Sun
	IconMoon   = "M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"                                                                                                                                                                                                       // lucide Moon
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
			// 圆弧: 端点参数化转多段三次贝塞尔(每段圆心角 <= 90°)
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
				ex, ey := v[5], v[6]
				if rel {
					ex, ey = x+ex, y+ey
				}
				arcToBeziers(x, y, v[0], v[1], v[2], v[3] != 0, v[4] != 0, ex, ey, push, bezier)
				x, y = ex, ey
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

// arcToBeziers 把 SVG 圆弧指令(A, 端点参数化)转换为若干段三次贝塞尔折线点,
// 端点/圆心换算与角度拆分见 W3C SVG 1.1 附录 F.6.5/F.6.6; 每段圆心角不超过 90°,
// 用 4/3·tan(θ/4) 的切向量长度做三次近似, 离散精度与整圆椭圆图元肉眼无差。
func arcToBeziers(x1, y1, rx, ry, phiDeg float32, largeArc, sweep bool, x2, y2 float32,
	push func(float32, float32),
	bezier func(x1, y1, x2, y2, x3, y3, x4, y4 float32)) {

	if x1 == x2 && y1 == y2 {
		return
	}
	rx, ry = abs32(rx), abs32(ry)
	if rx == 0 || ry == 0 { // 半径为 0 退化为直线
		push(x2, y2)
		return
	}
	phi := float64(phiDeg) * math.Pi / 180
	cosP, sinP := float32(math.Cos(phi)), float32(math.Sin(phi))

	// 端点换到以椭圆中点为原点、沿椭圆主轴的坐标系
	dx, dy := (x1-x2)/2, (y1-y2)/2
	x1p := cosP*dx + sinP*dy
	y1p := -sinP*dx + cosP*dy

	// 半径过小则按端点距离等比放大
	if lam := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry); lam > 1 {
		s := float32(math.Sqrt(float64(lam)))
		rx, ry = rx*s, ry*s
	}

	// 圆心(换算回屏幕坐标系)。两个候选圆心对称于弦, 符号按 SVG 规范 F.6.5:
	// large-arc 与 sweep 不相同时取 +√q, 相同时取 -√q(取错会绕远弧, 图标乱成一团)。
	sign := float32(-1)
	if largeArc != sweep {
		sign = 1
	}
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	// 半径按端点距离放大后 num 理论上为 0, 浮点误差下可能是极小负数, 必须钳到 0
	var co float32
	if q := num / den; q > 0 {
		co = sign * float32(math.Sqrt(float64(q)))
	}
	cxp := co * rx * y1p / ry
	cyp := -co * ry * x1p / rx
	cx := cosP*cxp - sinP*cyp + (x1+x2)/2
	cy := sinP*cxp + cosP*cyp + (y1+y2)/2

	ang := func(ux, uy, vx, vy float32) float32 {
		c := (ux*vx + uy*vy) / float32(math.Sqrt(float64((ux*ux+uy*uy)*(vx*vx+vy*vy))))
		if c > 1 {
			c = 1
		} else if c < -1 {
			c = -1
		}
		a := float32(math.Acos(float64(c)))
		if ux*vy-uy*vx < 0 {
			a = -a
		}
		return a
	}
	ux, uy := (x1p-cxp)/rx, (y1p-cyp)/ry
	vx, vy := (-x1p-cxp)/rx, (-y1p-cyp)/ry
	th1 := ang(1, 0, ux, uy)
	dth := ang(ux, uy, vx, vy)
	if !sweep && dth > 0 {
		dth -= 2 * math.Pi
	} else if sweep && dth < 0 {
		dth += 2 * math.Pi
	}

	// 拆成不超过 90° 的段逐段转贝塞尔; 点函数 E(a) 是椭圆上的点, D(a) 是切向导数
	n := int(math.Ceil(math.Abs(float64(dth))/(math.Pi/2))) + 1
	seg := dth / float32(n)
	k := float32(4.0 / 3.0 * math.Tan(float64(seg)/4))
	e := func(a float32) (float32, float32) {
		ca, sa := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
		return cx + rx*cosP*ca - ry*sinP*sa, cy + rx*sinP*ca + ry*cosP*sa
	}
	d := func(a float32) (float32, float32) {
		sa, ca := float32(math.Sin(float64(a))), float32(math.Cos(float64(a)))
		return -rx*cosP*sa - ry*sinP*ca, -rx*sinP*sa + ry*cosP*ca
	}
	for i := 0; i < n; i++ {
		a1, a2 := th1+seg*float32(i), th1+seg*float32(i+1)
		p1x, p1y := e(a1)
		p2x, p2y := e(a2)
		d1x, d1y := d(a1)
		d2x, d2y := d(a2)
		bezier(p1x, p1y, p1x+k*d1x, p1y+k*d1y, p2x-k*d2x, p2y-k*d2y, p2x, p2y)
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
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
