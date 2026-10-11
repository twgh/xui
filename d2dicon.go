package xui

import (
	"math"

	"github.com/twgh/xcgui/d2d"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// ==================== 图标抗锯齿: D2D 几何填充 ====================
//
// 背景: lucide 是描边图标, 老实现是"把 path 解析成折线 → XDraw_DrawLine 逐段描边"。
// 那条路在小尺寸下不干净: 折线端点被 round32 取整到逻辑像素(18px 图标上就是 ±1.5 物理像素)、
// 线条被 ceil 取整到整数线宽、8 段离散的多边形棱角, 加上重叠线段在 AA 下深浅不均。
//
// 方案: 把描边在 CPU 侧展开成**填充**多边形 —— 每条线段向两侧各偏移 w/2 得到一个四边形,
// 每个顶点补一个半径 w/2 的圆帽多边形(round linecap / round linejoin 的离散近似),
// 全部塞进同一条路径几何, 用 ID2D1RenderTarget::FillGeometry 一次填充出图。
// 几何是精确的(实测墨迹面积与理论值差 4%, 折线法是 +50%), 且合并成一条几何后
// 重叠区不会互相加深, 半透明色也能正确合成。
//
// COM 直通那一层(ABI 三规则 / vtable 索引 / 几何与画刷的创建释放 / 几何自检)已上移到
// **github.com/twgh/xcgui/d2d**, 本文件只留图标语义: 解析结果 → 轮廓 → 几何、缓存、元素 alpha。
//
// 坐标系: **必须先把元素客户区坐标用 XDraw_ConvRect 换算成画布对齐坐标**
// (炫彩只在刷新自己的 XDraw 命令时才施加元素变换/裁剪, 直通调用拿到的是父级坐标系),
// 换算结果是物理像素, 不要再乘 DPI —— 见 d2dIcon / DESIGN.md 第八节 43。
//
// 性能与坑(几何不能退化成"图元汤"、绕向必须统一等)见 docs/DESIGN.md 第八节 43。

// ---------------- 几何构造与缓存 ----------------

// d2dGeomKey 标识一条图标几何。坐标已"一次成型"烘焙成最终绘制坐标,
// 所以原点(ox/oy)必须进 key, 否则同一图标换个位置会画错地方。
type d2dGeomKey struct {
	paths string // 图标 path 原文(资源里的一条数据, 或调用方自备的 lucide path)
	ox    int32
	oy    int32
	size  int32
	sw    float32
}

// 几何缓存。绘制回调每帧都会新建 Painter(Painter 是帧内对象), 所以缓存必须放包级。
// 几何是**设备无关**资源, 归属 D2D 工厂而非渲染目标, 因此按工厂作废:
// 工厂变了(D2D 设备重建)整体 Release 重建。这样多个窗口(主窗/菜单/弹层各有各的 RT)
// 共用一份几何, 不会互相把缓存顶掉。
var (
	d2dGeomFactory *d2d.ID2D1Factory
	d2dGeoms       map[d2dGeomKey]*d2d.ID2D1PathGeometry
)

func d2dGeomCache(factory *d2d.ID2D1Factory) map[d2dGeomKey]*d2d.ID2D1PathGeometry {
	if factory != d2dGeomFactory {
		for _, g := range d2dGeoms {
			g.Release()
		}
		d2dGeoms = make(map[d2dGeomKey]*d2d.ID2D1PathGeometry)
		d2dGeomFactory = factory
	}
	if d2dGeoms == nil {
		d2dGeoms = make(map[d2dGeomKey]*d2d.ID2D1PathGeometry)
	}
	return d2dGeoms
}

// ---------------- 描边展开(CPU) ----------------

// iconStrokeFigures 把子路径展开成填充多边形: **每条描边子路径 → 一条闭合轮廓**
// (round cap/join), 填充子路径(filled, 即 lucide 的 fill="currentColor" 色点)原样入几何。
//
// 为什么必须是"一条轮廓"而不是"每段一个四边形 + 每个顶点一个圆帽":
// D2D 填充自重叠路径走 stencil-and-cover, 代价随**子路径数**近似平方增长 ——
// 实测 45 个图标一帧: 9 个子路径 4.1ms, 39 个 39ms, 199 个 1590ms。图元汤写法一个
// 图标就上百个子路径, 图标总览页(45 个图标同框)直接卡死。合并成一条轮廓后每图标
// 只剩几条子路径, 填充代价回到常数级。
//
// 轮廓构造(与 SVG 的 round linejoin/linecap 语义一致):
//   - 两侧各按 w/2 偏移; 顶点处**外侧**补圆弧(圆角连接), **内侧**取两条偏移线的交点(精确);
//   - 两端各补一个半圆帽。
//
// 所有轮廓的绕向天然一致(与折线自身绕向无关), 因此在 WINDING 模式下并集即笔画形状;
// 跨子路径的重叠(比如 X 的两笔)也会正确并起来。
//
// 坐标一次性成型: 先按 s=size/24 缩放到最终绘制坐标(含 ox/oy 偏移), 再按 w(=strokeWidth*s)展开。
func iconStrokeFigures(subs []iconSub, ox, oy int32, s, w float32) [][]float32 {
	r := w / 2
	if r <= 0 {
		return nil
	}
	figs := make([][]float32, 0, 8)
	for _, sub := range subs {
		if len(sub.pts) < 2 {
			continue
		}
		pts := make([]float32, len(sub.pts)*2)
		for i, p := range sub.pts {
			pts[i*2] = float32(ox) + p.x*s
			pts[i*2+1] = float32(oy) + p.y*s
		}
		if sub.filled { // 填充形状: 原样入几何(闭合由 EndFigure 负责)
			figs = append(figs, sameOrientation(pts))
			continue
		}
		pts = dedupePts(pts)
		if len(pts) < 4 {
			continue
		}
		if outline := strokeOutline(pts, r); len(outline) >= 6 {
			figs = append(figs, outline)
		}
	}
	return figs
}

// strokeOutline 把一条折线(最终坐标, 扁平 x,y)展开成一条闭合轮廓(round cap/join)。
func strokeOutline(pts []float32, r float32) []float32 {
	segs := len(pts)/2 - 1
	if segs < 1 {
		return nil
	}
	// 每段的单位方向与单位左法线(x,y)
	ux := make([]float32, segs)
	uy := make([]float32, segs)
	nx := make([]float32, segs)
	ny := make([]float32, segs)
	for i := 0; i < segs; i++ {
		vx := pts[i*2+2] - pts[i*2]
		vy := pts[i*2+3] - pts[i*2+1]
		l := float32(math.Hypot(float64(vx), float64(vy)))
		ux[i], uy[i] = vx/l, vy/l
		nx[i], ny[i] = -uy[i], ux[i] // 左法线
	}
	out := make([]float32, 0, segs*4+48)
	add := func(x, y float32) { out = append(out, x, y) }

	// ---- A 侧(左法线侧)正向 ----
	add(pts[0]+nx[0]*r, pts[1]+ny[0]*r)
	for i := 1; i < segs; i++ { // 顶点 i(连接段 i-1 与段 i)
		cx, cy := pts[i*2], pts[i*2+1]
		cross := ux[i-1]*uy[i] - uy[i-1]*ux[i]
		if cross < 0 { // A 侧在外: 补圆角弧(起点已在上一步给出)
			a0 := float32(math.Atan2(float64(ny[i-1]), float64(nx[i-1])))
			sweep := float32(math.Atan2(float64(nx[i-1]*ny[i]-ny[i-1]*nx[i]), float64(nx[i-1]*nx[i]+ny[i-1]*ny[i])))
			out = addArc(out, cx, cy, r, a0, sweep, true)
		} else if mx, my, ok := miterPoint(cx, cy, nx[i-1], ny[i-1], nx[i], ny[i], r); ok {
			add(mx, my) // A 侧在内: 取两条偏移线的交点(精确的内角)
		} else {
			add(cx+nx[i]*r, cy+ny[i]*r)
		}
	}
	add(pts[segs*2]+nx[segs-1]*r, pts[segs*2+1]+ny[segs-1]*r)

	// ---- 末端半圆帽: A 侧 → B 侧, 扫过 -π(经过前进方向的端点) ----
	a0 := float32(math.Atan2(float64(ny[segs-1]), float64(nx[segs-1])))
	out = addArc(out, pts[segs*2], pts[segs*2+1], r, a0, -math.Pi, true)

	// ---- B 侧反向 ----
	for i := segs - 1; i >= 1; i-- {
		cx, cy := pts[i*2], pts[i*2+1]
		cross := ux[i-1]*uy[i] - uy[i-1]*ux[i]
		if cross > 0 { // B 侧在外: 从 -N_i 绕到 -N_{i-1}
			a0 := float32(math.Atan2(float64(-ny[i]), float64(-nx[i])))
			sweep := float32(math.Atan2(float64(nx[i]*ny[i-1]-ny[i]*nx[i-1]), float64(nx[i]*nx[i-1]+ny[i]*ny[i-1])))
			out = addArc(out, cx, cy, r, a0, sweep, true)
		} else if mx, my, ok := miterPoint(cx, cy, -nx[i], -ny[i], -nx[i-1], -ny[i-1], r); ok {
			add(mx, my)
		} else {
			add(cx-nx[i-1]*r, cy-ny[i-1]*r)
		}
	}
	add(pts[0]-nx[0]*r, pts[1]-ny[0]*r)

	// ---- 起始半圆帽: B 侧 → A 侧, 同样扫过 -π(经过后退方向的端点) ----
	a0 = float32(math.Atan2(float64(-ny[0]), float64(-nx[0])))
	out = addArc(out, pts[0], pts[1], r, a0, -math.Pi, true)
	return out
}

// addArc 从角度 a0 起按 sweep 扫过(弧度), 每段不超过 30°; skipFirst 用于起点已给的情形。
func addArc(out []float32, cx, cy, r, a0, sweep float32, skipFirst bool) []float32 {
	k := int(math.Ceil(math.Abs(float64(sweep)) / (math.Pi / 6)))
	if k < 1 {
		k = 1
	}
	for i := 0; i <= k; i++ {
		if i == 0 && skipFirst {
			continue
		}
		a := float64(a0 + sweep*float32(i)/float32(k))
		out = append(out, cx+r*float32(math.Cos(a)), cy+r*float32(math.Sin(a)))
	}
	return out
}

// miterPoint 求两侧偏移线在角点的交点(单位法线 n1/n2, 长度 r)。
// 分母 1+n1·n2 在 180° 急折时趋 0, 此时返回 ok=false(调用方退化成直连)。
func miterPoint(cx, cy, n1x, n1y, n2x, n2y, r float32) (float32, float32, bool) {
	d := 1 + n1x*n2x + n1y*n2y
	if d < 0.05 {
		return 0, 0, false
	}
	return cx + r*(n1x+n2x)/d, cy + r*(n1y+n2y)/d, true
}

// dedupePts 去掉重合的相邻点(路径数据里常见重复端点), 否则方向向量无从定义。
func dedupePts(pts []float32) []float32 {
	out := make([]float32, 0, len(pts))
	for i := 0; i+1 < len(pts); i += 2 {
		if n := len(out); n >= 2 {
			dx, dy := pts[i]-out[n-2], pts[i+1]-out[n-1]
			if math.Hypot(float64(dx), float64(dy)) < 1e-4 {
				continue
			}
		}
		out = append(out, pts[i], pts[i+1])
	}
	return out
}

// sameOrientation 把多边形统一成"正向"(鞋带公式面积为正), 需要时反转点序。
//
// 为什么必须: 在 WINDING 填充模式下, 只有缠绕组数非 0 的区域才填充 —— 反向重叠的
// 多边形会让 winding 归零, 重叠区被挖空。段四边形与圆帽的天然绕向相反, 不统一
// 就会出现"梳齿"状描边。
func sameOrientation(pts []float32) []float32 {
	n := len(pts) / 2
	if n < 3 {
		return pts
	}
	var sum float64
	for i := 0; i < n-1; i++ {
		sum += float64(pts[i*2])*float64(pts[i*2+3]) - float64(pts[i*2+2])*float64(pts[i*2+1])
	}
	sum += float64(pts[(n-1)*2])*float64(pts[1]) - float64(pts[0])*float64(pts[(n-1)*2+1])
	if sum >= 0 {
		return pts
	}
	rev := make([]float32, len(pts))
	for i := 0; i < n; i++ {
		rev[i*2] = pts[(n-1-i)*2]
		rev[i*2+1] = pts[(n-1-i)*2+1]
	}
	return rev
}

// ---------------- Painter 出图口 ----------------

// d2dIcon 尝试用 D2D 填充管线把图标画在 rc(元素客户区坐标)内,
// 返回 false 表示不可用(GDI 渲染 / 资源创建失败 / 参数无效), 调用方回落折线。
func (p *Painter) d2dIcon(paths string, rc *xc.RECT, color Color, strokeWidth float32,
	build func() []iconSub) bool {

	if p == nil || p.Draw == nil || p.Draw.Handle <= 0 || rc == nil || color.IsNone() ||
		rc.Right <= rc.Left || rc.Bottom <= rc.Top {
		return false
	}
	// 关键: 直通 D2D 调用拿到的渲染目标还没有"当前元素"的变换与裁剪 ——
	// 炫彩是在刷新自己的绘制命令时才施加它们(XDraw_* 走的是那条队列)。
	// 所以必须用官方给的 XDraw_ConvRect 把"客户区坐标"换算成"画布对齐坐标"
	// (含元素偏移 + DPI 缩放), 否则直通绘制会落到父级坐标系里去。
	canvas := *rc
	xc.XDraw_ConvRect(p.Draw.Handle, &canvas)
	ox, oy, size, ok := iconBox(paths, &canvas, color)
	if !ok {
		return false
	}
	return p.d2dStroke(d2dGeomKey{paths: paths, ox: ox, oy: oy, size: size, sw: strokeWidth},
		color, ox, oy, size, strokeWidth, build)
}

// d2dStroke 用 D2D 的填充管线画一条"描边"。
// 返回 false 表示当前环境不可用(GDI 渲染 / 资源创建失败), 调用方回落折线渲染。
// build 只在几何缓存未命中时调用, 避免每帧重复解析 path。
func (p *Painter) d2dStroke(key d2dGeomKey, color Color, ox, oy, size int32, strokeWidth float32,
	build func() []iconSub) bool {

	if p == nil || p.Draw == nil || p.Draw.Handle <= 0 {
		return false
	}
	// app.New(false) 的 GDI 渲染没有 D2D 目标 → 构造函数拿到 0 返回 nil, 走折线回落。
	rt := d2d.NewRenderTargetByHandle(xc.XDraw_GetD2dRenderTarget(p.Draw.Handle))
	factory := d2d.NewFactoryByHandle(xc.XC_GetD2dFactory())
	if rt == nil || factory == nil {
		return false
	}

	geoms := d2dGeomCache(factory)
	geom, hit := geoms[key]
	if !hit {
		s := float32(size) / 24
		geom, _ = factory.NewGeometry(iconStrokeFigures(build(), ox, oy, s, strokeWidth*s))
		geoms[key] = geom // 失败(nil)也记下, 免得每帧重试
	}
	if geom == nil {
		return false
	}

	// 直通绘制绕过了炫彩的元素 alpha 合成(实测: Dialog 淡入时关闭图标不跟着淡),
	// 这里把元素当前 alpha 乘进颜色的 alpha; 窗口级 Painter 没有元素 alpha 概念, 跳过。
	// 画刷是设备相关资源, 由 FillGeometryColor 每帧即建即放(实测建刷代价可忽略,
	// 图标几何才是大头)。
	r, g, b, a := color.RGBA()
	if p.hEle != 0 && xc.XObj_GetType(p.hEle) != xcc.XC_WINDOW {
		if ea := xc.XEle_GetAlpha(p.hEle); ea != 255 {
			a = byte(uint32(a) * uint32(ea) / 255)
		}
	}
	return rt.FillGeometryColor(geom, d2d.NewColorF(r, g, b, a)) == nil
}
