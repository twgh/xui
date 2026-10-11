package xui

import (
	"math"

	"github.com/twgh/xcgui/d2d"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// ==================== 图标抗锯齿: D2D 几何绘制 ====================
//
// 背景: lucide 是描边图标, 老实现是"把 path 解析成折线 → XDraw_DrawLine 逐段描边"。
// 那条路在小尺寸下不干净: 折线端点被 round32 取整到逻辑像素(18px 图标上就是 ±1.5 物理像素)、
// 线条被 ceil 取整到整数线宽、8 段离散的多边形棱角, 加上重叠线段在 AA 下深浅不均。
//
// 方案(2026-10-11 换成快路线): 描边**交给 Direct2D** —— 把 lucide 折线建成路径几何(子路径保持**开口**),
// 用 ID2D1RenderTarget::DrawGeometry(线宽 + 圆角连接/圆头样式)一次描边出图; 少数 fill 色点用
// FillGeometry 填充。
//
// 早先的做法是"CPU 侧把描边展开成填充多边形再 FillGeometry"(strokeOutline/addArc/miterPoint 那套,
// 约 300 行), 实测同样画 2 万次: **DrawGeometry 约 20ms、填充已展开的环约 80ms(4 倍)**; 而且省掉
// 展开代码、圆角/圆帽由 D2D 精确保证(SVG 的 round 语义, 见 d2d.RoundStrokeStyleProperties)。
// 展开法唯一的优势是"几何与线宽绑死", 换来的是 4 倍绘制成本 + 改线宽就得重建几何。
//
// COM 直通那一层(ABI 三规则 / vtable 索引 / 几何与画刷的创建释放 / 几何自检)已上移到
// **github.com/twgh/xcgui/d2d**, 本文件只留图标语义: 解析结果 → 几何、缓存、元素 alpha。
//
// 坐标系: **必须先把元素客户区坐标用 XDraw_ConvRect 换算成画布对齐坐标**
// (炫彩只在刷新自己的 XDraw 命令时才施加元素变换/裁剪, 直通调用拿到的是父级坐标系),
// 换算结果是物理像素, 不要再乘 DPI —— 见 d2dIcon / DESIGN.md 第八节 43。
//
// 性能与坑(图元汤会让填充退化成平方级代价、填充形状绕向必须统一等)见 DESIGN 第八节 43。

// ---------------- 几何构造与缓存 ----------------

// d2dGeomKey 标识一个图标的几何。
//
// **位置(ox/oy)与线宽都不在 key 里**: 几何以自身原点为基准构造、不含线宽(线宽绘制时传给
// DrawGeometry), 位置在绘制时用渲染目标的变换矩阵平移过去(见 d2dStroke)。于是窗口 resize(位置全变)
// 或改线宽都不会触发几何重建 —— 早先把位置烘进几何时, resize 后每个图标都是新 key, 全部重建。
type d2dGeomKey struct {
	paths string // 图标 path 原文(资源里的一条数据, 或调用方自备的 lucide path)
	size  int32  // 图标边长(物理像素)
}

// d2dGeomPair 一个图标的两条几何: 描边折线(开口, DrawGeometry 用)与填充形状(闭合, FillGeometry 用)。
// 任一可为 nil —— 大多数图标只有描边, 少数(如 X 中心的点)还有填充。
type d2dGeomPair struct {
	stroke *d2d.ID2D1PathGeometry
	fill   *d2d.ID2D1PathGeometry
}

func (g *d2dGeomPair) release() {
	if g == nil {
		return
	}
	if g.stroke != nil {
		g.stroke.Release()
	}
	if g.fill != nil {
		g.fill.Release()
	}
}

// 几何缓存。绘制回调每帧都会新建 Painter(Painter 是帧内对象), 所以缓存必须放包级。
// 几何与描边样式都是**设备无关**资源, 归属 D2D 工厂而非渲染目标, 因此按工厂作废:
// 工厂变了(D2D 设备重建)整体 Release 重建。这样多个窗口(主窗/菜单/弹层各有各的 RT)
// 共用一份, 不会互相把缓存顶掉。
var (
	d2dGeomFactory *d2d.ID2D1Factory
	d2dGeoms       map[d2dGeomKey]*d2dGeomPair
	d2dStrokeStyle *d2d.ID2D1StrokeStyle
)

// d2dGeomCache 取几何缓存与描边样式, 工厂变了就先整体作废重建。
func d2dGeomCache(factory *d2d.ID2D1Factory) (map[d2dGeomKey]*d2dGeomPair, *d2d.ID2D1StrokeStyle) {
	if factory != d2dGeomFactory {
		for _, g := range d2dGeoms {
			g.release()
		}
		d2dGeoms = make(map[d2dGeomKey]*d2dGeomPair)
		if d2dStrokeStyle != nil {
			d2dStrokeStyle.Release()
			d2dStrokeStyle = nil
		}
		d2dGeomFactory = factory
	}
	if d2dGeoms == nil {
		d2dGeoms = make(map[d2dGeomKey]*d2dGeomPair)
	}
	if d2dStrokeStyle == nil {
		// SVG 语义: stroke-linecap/linejoin = round(lucide 图标全带这两个属性)
		props := d2d.RoundStrokeStyleProperties()
		if s, err := factory.CreateStrokeStyle(&props); err == nil {
			d2dStrokeStyle = s
		}
		// 建失败也不致命: DrawGeometry 传 nil 样式 = 默认(平头 + 斜接), 只是角不圆。
	}
	return d2dGeoms, d2dStrokeStyle
}

// ---------------- 几何构造 ----------------

// iconFigures 把子路径按 s=size/24 缩放到"自身原点"的最终绘制坐标(位置由绘制时的变换施加),
// 并按语义分成两组:
//   - stroke: 描边折线, **保持开口** —— 闭合会多画一段收尾线(三角形的描边会变成回形针);
//   - fill: 填充形状, 闭合且统一绕向。
func iconFigures(subs []iconSub, s float32) (stroke, fill []d2d.GeometryFigure) {
	for _, sub := range subs {
		if len(sub.pts) < 2 {
			continue
		}
		pts := make([]float32, len(sub.pts)*2)
		for i, p := range sub.pts {
			pts[i*2] = p.x * s
			pts[i*2+1] = p.y * s
		}
		if sub.filled {
			fill = append(fill, d2d.GeometryFigure{Points: sameOrientation(pts), Closed: true})
			continue
		}
		if pts = dedupePts(pts); len(pts) >= 4 {
			stroke = append(stroke, d2d.GeometryFigure{Points: pts, Closed: false})
		}
	}
	return stroke, fill
}

// dedupePts 去掉重合的相邻点(路径数据里常见重复端点), 免得描边在零长线段处出毛刺。
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
// 多边形会让 winding 归零, 重叠区被挖空(描边子路径交给 D2D 自己处理, 这条只对填充形状重要)。
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

// d2dIcon 尝试用 D2D 几何绘制管线把图标画在 rc(元素客户区坐标)内,
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
	return p.d2dStroke(d2dGeomKey{paths: paths, size: size},
		color, ox, oy, size, strokeWidth, build)
}

// d2dStroke 给图标出图: 描边折线走 DrawGeometry(线宽 + 圆角样式), 填充形状走 FillGeometry。
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

	geoms, style := d2dGeomCache(factory)
	pair, hit := geoms[key]
	if !hit {
		s := float32(size) / 24
		strokeFigs, fillFigs := iconFigures(build(), s)
		pair = &d2dGeomPair{}
		if len(strokeFigs) > 0 {
			pair.stroke, _ = factory.NewGeometryFigures(strokeFigs)
		}
		if len(fillFigs) > 0 {
			pair.fill, _ = factory.NewGeometryFigures(fillFigs)
		}
		geoms[key] = pair // 失败(nil)也记下, 免得每帧重试
	}
	if pair.stroke == nil && pair.fill == nil {
		return false
	}

	// 位置不烘进几何(见 d2dGeomKey): 用渲染目标的变换矩阵把几何平移到目标位置。
	// **必须成对恢复** —— 实测(探针 .verify/probexform2)炫彩自己的绘制(XDraw_*)在绘制时用的是
	// 当前变换, 残留的矩阵会把页面内容一起平移; 成对设置/恢复后不受影响。
	if ox != 0 || oy != 0 {
		prev, err := rt.GetTransform()
		if err != nil {
			return false
		}
		tm := d2d.TranslationMatrix(float32(ox), float32(oy))
		rt.SetTransform(&tm)
		defer rt.SetTransform(&prev)
	}

	// 直通绘制绕过了炫彩的元素 alpha 合成(实测: Dialog 淡入时关闭图标不跟着淡),
	// 这里把元素当前 alpha 乘进颜色的 alpha; 窗口级 Painter 没有元素 alpha 概念, 跳过。
	// 画刷是设备相关资源, 由 *GeometryColor 每帧即建即放(实测建刷代价可忽略)。
	r, g, b, a := color.RGBA()
	if p.hEle != 0 && xc.XObj_GetType(p.hEle) != xcc.XC_WINDOW {
		if ea := xc.XEle_GetAlpha(p.hEle); ea != 255 {
			a = byte(uint32(a) * uint32(ea) / 255)
		}
	}
	col := d2d.NewColorF(r, g, b, a)

	ok := true
	if pair.stroke != nil {
		// 线宽: lucide 网格单位(24 见方) → 物理像素
		if err := rt.DrawGeometryColor(pair.stroke, col, strokeWidth*float32(size)/24, style); err != nil {
			ok = false
		}
	}
	if pair.fill != nil {
		if err := rt.FillGeometryColor(pair.fill, col); err != nil {
			ok = false
		}
	}
	return ok
}
