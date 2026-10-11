package xui

import (
	"math"
	"strconv"
	"unicode"

	"github.com/twgh/xcgui/xc"
)

// 图标: 解析 lucide 的 path 数据并用 Painter 画出来。
//
// 数据格式是 24x24 viewBox 的 SVG path 指令(与 shadcn 的默认图标库 lucide 同源):
// 支持 M/L/H/V/C/S/Q/T/Z 及相对指令(曲线按 8 段离散); 圆弧 A/a 转为多段三次贝塞尔
// (每段圆心角 <= 90°, 见 arcToBeziers); `!` 前缀表示下一条子路径是填充(生成资源内部约定)。
//
// 为什么不用 xcgui 的 SVG 渲染(2026-10-07 复核实证, 见 DESIGN.md 第八节 17):
//  1. lucide 官方的紧凑 path 写法(如 `a.25.25 0 00.407.195`)xcgui 解析失败,
//     svg.NewByString 直接返回 nil, 句柄都拿不到;
//  2. 即便喂归一化后的 path(icons 包存的绝对坐标形式), 虽能解析, 渲染结果也是错的
//     (圆弧走样、fill="none" 的子路径被糊成填充块);
//  3. 图标需要逐绘制换色(SetUserStrokeColor 的活)与精确几何, 所以自己解析 path 几何。
//
// 出图有两条路(见 Painter.Icon):
//  1. D2D 直通绘制(d2dicon.go): 解析出的折线 → D2D 几何(描边子路径走 DrawGeometry + 圆角样式,
//     填充色点走 FillGeometry), 抗锯齿、几何精确, 是当前的主路径;
//  2. 折线描边(DrawLine)回落: 只在 GDI 渲染(app.New(false))或 D2D 资源创建失败时使用。
//
// 图标从哪来 —— 全在子包 github.com/twgh/xui/icons(数据由 scripts/gen_icons.py 生成):
//
//	p.Icon(icons.IconChevronDown, &rc, color, 2) // 常量: 编译期内联, 零体积成本(推荐)
//	p.Icon(icons.Path("star"), &rc, color, 2)    // 按名字取: 会把整张资源表链进二进制
//
// 每个图标都有一个 `icons.IconXxx` 常量(名字是 lucide 名的驼峰形式, 见子包文档),
// 只用常量的程序**一个字节都不多花**; `icons.Path` 是给"名字来自用户输入/配置"的场合
// 准备的, 代价是整张表(2130 个图标约 350KB)跟着进二进制。也可以直接贴 lucide 的
// path 数据(见 https://lucide.dev)。

// Icon 在 rc 内绘制一枚**描边图标**(等比铺满)。
//
// data 两种形态, **自动识别**(跳过前导空白看首字符是不是 `<`):
//
//   - **path 数据**: **24x24 网格**的描边型 path, 例如 icons.IconXxx / icons.Path("eye") / 自己拼的
//     path。子路径想按**填充**画时在它前面加 `!`(项目内部约定, 见 parseIconPath); 手写 path 不含它 = 纯描边。
//     非 24 网格的请先把坐标缩放过去(整段 SVG 则不用, 见下)。
//   - **整段 SVG**: 从图标站复制来的 `<svg viewBox="…"><path d="…"/></svg>`(heroicons / feather /
//     iconfont…)—— 自动提取 `<path>`、按 viewBox **归一到 24 网格**(任意网格)、按 fill/stroke 判定
//     填充子路径。归一化结果**按 SVG 原文缓存**(最多 512 条), 每帧只多一次首字符判断。
//
// strokeWidth 用 24 网格单位; 传 <= 0 时: SVG 用它自己的 stroke-width(里面也没有则 2),
// path 数据按 2(lucide 惯例)。想先校验 SVG 或自己缓存归一化结果, 用 ParseIconSVG。
//
// 出图优先走 D2D 几何管线(有抗锯齿, 见 d2dicon.go); 只有 GDI 渲染模式(app.New(false))或
// D2D 资源建不出来时才回落到折线描边。
func (p *Painter) Icon(data string, rc *xc.RECT, color Color, strokeWidth float32) {
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
	if p.d2dIcon(paths, rc, color, strokeWidth, func() []iconSub {
		return parseIconPath(paths)
	}) {
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
	for _, sub := range parseIconPath(paths) {
		for i := 1; i < len(sub.pts); i++ {
			ax, ay := ox+round32(sub.pts[i-1].x*s), oy+round32(sub.pts[i-1].y*s)
			bx, by := ox+round32(sub.pts[i].x*s), oy+round32(sub.pts[i].y*s)
			if ax == bx && ay == by {
				continue
			}
			p.Draw.DrawLine(ax, ay, bx, by)
		}
	}
}

// iconBox 校验并算出图标在 rc 内的居中正方形(逻辑像素)。
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
//
// 一个小数点就停: SVG 允许 ".25.25" 表示两个数(.25 与 .25), 贪婪吃到第二个 '.'
// 会让 ParseFloat 失败并把后面整段图形静默丢掉(lucide 的 album/settings 等都是这个写法,
// 当时表现为"复制来的图标只画出一半")。
func (sc *iconScanner) num() (float32, bool) {
	sc.skipSep()
	st := sc.pos
	if sc.pos < len(sc.s) && (sc.s[sc.pos] == '-' || sc.s[sc.pos] == '+') {
		sc.pos++
	}
	dot := false
	for sc.pos < len(sc.s) {
		c := sc.s[sc.pos]
		if c >= '0' && c <= '9' {
			sc.pos++
			continue
		}
		if c == '.' && !dot {
			dot = true
			sc.pos++
			continue
		}
		break
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

// flag 取一个 arc 标志位: SVG 规定 large-arc/sweep 是**单字符** 0/1,
// 所以 "00.407" 必须按 0 / 0 / .407 读, 不能按数字切(见 num 的注释)。
func (sc *iconScanner) flag() (float32, bool) {
	sc.skipSep()
	if sc.pos >= len(sc.s) || (sc.s[sc.pos] != '0' && sc.s[sc.pos] != '1') {
		return 0, false
	}
	c := sc.s[sc.pos]
	sc.pos++
	return float32(c - '0'), true
}

// iconSub 是一条子路径: 折线点(24 网格) + 是否"填充"。
//
// 描边是常态(round cap/join 的展开见 iconStrokeFigures); 少数 lucide 图标里有
// fill="currentColor" 的小图形(色点), 那些子路径要整体填实而不是描边, 由 `!` 前缀标记
// (生成资源用, 见 icons/lucide_gen.go; 手写 path 用不到)。
type iconSub struct {
	pts    []iconPt
	filled bool
}

// parseIconPath 把 SVG path 解析为若干条折线(24x24 网格坐标)。
//
// `!` 前缀表示"接下来的这条子路径是填充"(非 SVG 语法, 本项目生成资源内部约定):
// 手写/复制的 lucide path 不含它, 走纯描边。
func parseIconPath(d string) []iconSub {
	subs, _ := parseIconPathEx(d)
	return subs
}

// parseIconPathEx 是可诊断版本: complete 表示整条数据被解析到底。
// 遇到不认识的指令或不满足的参数会提前停下并返回 false(此前是静默截断, 表现成
// "图标只画出一半") —— 生成资源用测试断言它必须为 true。
func parseIconPathEx(d string) ([]iconSub, bool) {
	sc := &iconScanner{s: d}
	var (
		subs         []iconSub
		cur          []iconPt
		curFilled    bool
		x, y, sx, sy float32
		cmd          byte
		lastC, lastQ iconPt
		hasC, hasQ   bool
	)
	push := func(px, py float32) { cur = append(cur, iconPt{px, py}) }
	flush := func() {
		if cur == nil { // 没有正在写的子路径: 不要动填充标记(它属于接下来的那条)
			return
		}
		if len(cur) > 1 {
			subs = append(subs, iconSub{pts: cur, filled: curFilled})
		}
		cur = nil
		curFilled = false
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
		// `!` = 接下来的子路径是填充(生成资源内部约定, 不是 SVG 语法)
		sc.skipSep()
		if sc.pos >= len(sc.s) {
			break // 尾部空白: 正常结束(否则会掉进"重复上一指令"分支, 把干净数据误判成不完整)
		}
		if sc.s[sc.pos] == '!' {
			flush()
			curFilled = true
			sc.pos++
			continue
		}
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
			// 圆弧: 端点参数化转多段三次贝塞尔(每段圆心角 <= 90°)。
			// 参数依次是 rx ry rot largeArc sweep x y —— 中间两个标志位是单字符,
			// 必须用 flag 逐字符取("a.25.25 0 00.407.195" 这样的连写只能这么读)。
			var v [7]float32
			for k := 0; k < 7; k++ {
				var (
					f float32
					o bool
				)
				if k == 3 || k == 4 {
					f, o = sc.flag()
				} else {
					f, o = sc.num()
				}
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
		default: // 不认识的指令: 数据有问题, 如实报告(不要静默截断)
			flush()
			return subs, false
		}
		if !ok { // 参数不足/数字畸形
			flush()
			return subs, false
		}
	}
	flush()
	return subs, true
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
