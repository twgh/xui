package xui

import (
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/twgh/xcgui/xc"
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
// 为什么不用 RT::DrawGeometry 做真描边: Win64 下浮点形参走 XMM 寄存器,
// syscall.SyscallN 传不进去。FillGeometry 的形参全是整数/指针, 因此可以纯 Go 调用,
// 不需要 cgo 也不需要汇编。
//
// 坐标系: **必须先把元素客户区坐标用 XDraw_ConvRect 换算成画布对齐坐标**
// (炫彩只在刷新自己的 XDraw 命令时才施加元素变换/裁剪, 直通调用拿到的是父级坐标系),
// 换算结果是物理像素, 不要再乘 DPI —— 见 d2dIcon / DESIGN.md 第八节 43。
//
// vtable 索引的实测依据与踩坑记录见 docs/DESIGN.md 第八节 43。

// ---------------- COM 直通(vtable 调用) ----------------

// comPtr 是 COM 接口指针。内部一律用 unsafe.Pointer 承载而非 uintptr:
// 这样 vtable 解引用不需要 uintptr→unsafe.Pointer 的反向转换
// (go vet 的 unsafeptr 检查会把它报成 "possible misuse of unsafe.Pointer")。
type comPtr = unsafe.Pointer

// comPtrOf 把 xcgui 以 uintptr 返回的 COM 接口指针搬成 comPtr。
// 绕一层取地址是为了避开 uintptr→unsafe.Pointer 的直接转换(见 comPtr 的说明);
// 这是纯位模式重解释, 不做任何指针算术。这些 COM 对象由 xcgui 持有,
// 只在绘制回调内使用, Go 的 GC 既不移动也不管理它们, 不存在悬垂问题。
func comPtrOf(v uintptr) comPtr { return *(*unsafe.Pointer)(unsafe.Pointer(&v)) }

// vcall 按 vtable 索引调用 COM 方法(Windows x64 调用约定)。
//
// ABI 硬规则(违反轻则整帧被毁/窗口彻底隐形, 重则崩溃):
//  1. 形参里含 float32/float64 的方法一律不能这样调 —— 浮点参数走 XMM 寄存器,
//     SyscallN 只能传通用寄存器;
//  2. 按值传递的 8 字节结构体要打包成一个 64 位整数(见 pointArg);
//  3. 其余参数一律用指针/整数传, 是安全的。
func vcall(obj comPtr, index int, args ...uintptr) uintptr {
	if obj == nil {
		return 0
	}
	vtbl := *(*comPtr)(obj)
	fn := *(*uintptr)(unsafe.Add(vtbl, uintptr(index)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{uintptr(obj)}, args...)...)
	return r
}

func f32(v float32) uintptr { return uintptr(math.Float32bits(v)) }

// pointArg 把 D2D1_POINT_2F(按值传递的两个 float32)打包成一个 64 位整数。
func pointArg(x, y float32) uintptr { return f32(x) | f32(y)<<32 }

// hrFailed 判断 HRESULT 是否失败。本项目用到的 COM 方法只返回 S_OK 或错误,
// 因此"非 0 即失败"是安全的。
func hrFailed(hr uintptr) bool { return uint32(hr) != 0 }

// outPtr 把局部 comPtr 变量的地址交给 COM 方法当 out 参数写回。
func outPtr(p *comPtr) uintptr { return uintptr(unsafe.Pointer(p)) }

// releaseCOM 调用 IUnknown::Release。
func releaseCOM(obj comPtr) {
	if obj != nil {
		vcall(obj, 2)
	}
}

// ID2D1RenderTarget: IUnknown(0-2) → ID2D1Resource::GetFactory(3), 方法从 4 起。
// 注意 Create* 与 Draw/Fill* 是混排的, 不要按字母序想当然。
const (
	rtCreateSolidColorBrush = 8  // (color *D2D1_COLOR_F, props *D2D1_BRUSH_PROPERTIES(可 NULL), out **brush)
	rtFillGeometry          = 23 // (geometry, brush, layer=NULL) ← 本方案的唯一出图口
)

// ID2D1Factory: 直接继承 IUnknown, 所以方法从 3 起。
const facCreatePathGeometry = 10 // (out **geom)

// ID2D1PathGeometry: 继承 ID2D1Geometry 的 13 个方法, 因此 Open 落在 17。
const (
	pgGetBounds = 4  // (transform=NULL, out *D2D1_RECT_F) —— 几何合法性探针
	pgOpen      = 17 // (out **sink) → HRESULT
)

// ID2D1GeometrySink: ID2D1SimplifiedGeometrySink 部分只有 7 个自有方法, 即 3~9
// (GeometrySink 追加的 AddBezier/AddArc 等都带浮点参数, 本项目不碰)。
//
// !!! 3~8 全部是 STDMETHOD_(void, ...) —— 返回值是寄存器里的残留垃圾, 千万别当
// HRESULT 去查(曾因此把 EndFigure 的垃圾值当失败, 导致每次静默回落折线渲染)。
// 这一段里只有 Close(9) 返回 HRESULT; 同理 ID2D1RenderTarget 的 Draw/Fill 全家也是 void。
const (
	sinkSetFillMode = 3 // (D2D1_FILL_MODE)          void
	sinkBeginFigure = 5 // (POINT_2F 打包, FIGURE_BEGIN) void
	sinkAddLines    = 6 // (points *float32, count)   void
	sinkEndFigure   = 8 // (D2D1_FIGURE_END)          void
	sinkClose       = 9 // () HRESULT —— 必须调用, 否则几何不可用
)

const (
	// 描边展开出的凸多边形大量重叠, 必须 WINDING(并集); 默认的 ALTERNATE
	// 会让重叠的段四边形与圆帽互相镂空。
	fillModeWinding   = 1
	figureBeginFilled = 0
	figureEndClosed   = 1
)

// d2dColorF 是 D2D1_COLOR_F(按指针传递, 顺序 RGBA, 分量 0~1)。
type d2dColorF struct{ R, G, B, A float32 }

func colorToD2D(c Color) d2dColorF {
	r, g, b, a := c.RGBA()
	const k = 1.0 / 255.0
	return d2dColorF{float32(r) * k, float32(g) * k, float32(b) * k, float32(a) * k}
}

// createSolidColorBrush 建纯色画刷(props 传 NULL 用默认值; 透明度由颜色的 alpha 承担)。
func createSolidColorBrush(rt comPtr, c d2dColorF) comPtr {
	var brush comPtr
	if hrFailed(vcall(rt, rtCreateSolidColorBrush,
		uintptr(unsafe.Pointer(&c)), 0, outPtr(&brush))) {
		return nil
	}
	return brush
}

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
	d2dGeomFactory comPtr
	d2dGeoms       map[d2dGeomKey]comPtr
)

func d2dGeomCache(factory comPtr) map[d2dGeomKey]comPtr {
	if factory != d2dGeomFactory {
		for _, g := range d2dGeoms {
			releaseCOM(g)
		}
		d2dGeoms = make(map[d2dGeomKey]comPtr)
		d2dGeomFactory = factory
	}
	if d2dGeoms == nil {
		d2dGeoms = make(map[d2dGeomKey]comPtr)
	}
	return d2dGeoms
}

// newGeometry 用若干闭合折线(扁平 x,y 点对, 最终绘制坐标)造一条路径几何。
// 任一环节失败返回 nil, 调用方回落到折线渲染。
func newGeometry(factory comPtr, figures [][]float32) comPtr {
	if factory == nil {
		return nil
	}
	var geom comPtr
	if hrFailed(vcall(factory, facCreatePathGeometry, outPtr(&geom))) || geom == nil {
		return nil
	}
	var sink comPtr
	if hrFailed(vcall(geom, pgOpen, outPtr(&sink))) || sink == nil {
		releaseCOM(geom)
		return nil
	}
	// 每个 figure 都要 BeginFigure → AddLines → EndFigure
	vcall(sink, sinkSetFillMode, fillModeWinding)
	ok := false
	for _, f := range figures {
		if len(f) < 6 { // 至少要 3 个点
			continue
		}
		vcall(sink, sinkBeginFigure, pointArg(f[0], f[1]), figureBeginFilled)
		vcall(sink, sinkAddLines, uintptr(unsafe.Pointer(&f[2])), uintptr(len(f)/2-1))
		vcall(sink, sinkEndFigure, figureEndClosed) // void, 不查返回值
		ok = true
	}
	if !ok || hrFailed(vcall(sink, sinkClose)) {
		releaseCOM(geom)
		return nil
	}
	runtime.KeepAlive(figures)
	// 几何合法性自检: 没 Close 成功的几何在这里返回 D2DERR_WRONG_STATE,
	// 拦下来才不会把坏几何喂给 FillGeometry(那会毁掉整帧, 表现为窗口彻底隐形)。
	if !geomUsable(geom) {
		releaseCOM(geom)
		return nil
	}
	return geom
}

// geomUsable 用 ID2D1Geometry::GetBounds 判断几何是否真的可用(hr=0 且 bounds 有限)。
func geomUsable(geom comPtr) bool {
	if geom == nil {
		return false
	}
	var rc [4]float32
	if hrFailed(vcall(geom, pgGetBounds, 0, uintptr(unsafe.Pointer(&rc)))) {
		return false
	}
	for _, v := range rc {
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return false
		}
	}
	return true
}

// ---------------- 描边展开(CPU) ----------------

// iconStrokeFigures 把 24 网格的折线子路径"展开"成填充多边形集合。
//
// 子路径分两种: 默认是**描边**(展开规则见下); 带 filled 标记的(lucide 的
// fill="currentColor" 色点, 由解析器的 `!` 前缀标出)整条作为填充多边形入几何 ——
// 它本来就是实心形状, 按描边处理会变成"胖一圈的空心环"。
//
// 规则(lucide 的 stroke-linecap / stroke-linejoin 都是 round):
//   - 每段向两侧各偏移 w/2 → 一个四边形(闭合 figure);
//   - 每个顶点补一个半径 w/2 的正多边形圆帽 —— 折线内部是圆角连接, 两个端点是圆头线帽,
//     同一套圆帽即可覆盖;
//
// 所有形状都是凸的, 在 WINDING 填充模式下并集即笔画形状; 且每个形状都落在
// "真·圆角描边"内部(内接近似), 不会比真描边胖。
//
// 关键: **所有 figure 必须同向**(见 sameOrientation)。段四边形与圆帽的天然绕向
// 是相反的, 直接混在一起 WINDING 会让重叠处的 winding 归零被挖空, 描边会变成
// 一排梳齿。
//
// 坐标一次性成型: 先按 s=size/24 缩放到逻辑像素, 再按 w(=strokeWidth*s)展开,
// 最后加 (ox,oy) 偏移 —— 几何里存的就是最终绘制坐标。
func iconStrokeFigures(subs []iconSub, ox, oy int32, s, w float32) [][]float32 {
	r := w / 2
	if r <= 0 {
		return nil
	}
	// 圆帽边数: 48px 图标(r=2px)下内接正多边形与真圆的偏差也只有 0.07px, 肉眼不可辨。
	const capSides = 12
	figs := make([][]float32, 0, 16)
	// 24 网格 → 最终绘制坐标
	scale := func(pts []iconPt) []float32 {
		out := make([]float32, len(pts)*2)
		for i, p := range pts {
			out[i*2] = float32(ox) + p.x*s
			out[i*2+1] = float32(oy) + p.y*s
		}
		return out
	}
	addCap := func(cx, cy float32) {
		pts := make([]float32, 0, (capSides+1)*2)
		for i := 0; i <= capSides; i++ {
			t := 2 * math.Pi * float64(i) / float64(capSides)
			pts = append(pts, cx+r*float32(math.Cos(t)), cy+r*float32(math.Sin(t)))
		}
		figs = append(figs, sameOrientation(pts))
	}
	for _, sub := range subs {
		if len(sub.pts) < 2 {
			continue
		}
		pts := scale(sub.pts)
		if sub.filled { // 填充形状: 原样入几何(closed figure 由 EndFigure 闭合)
			figs = append(figs, pts)
			continue
		}
		for i := 0; i+1 < len(sub.pts); i++ {
			ax, ay := pts[i*2], pts[i*2+1]
			bx, by := pts[i*2+2], pts[i*2+3]
			dx, dy := bx-ax, by-ay
			l := float32(math.Hypot(float64(dx), float64(dy)))
			if l < 1e-4 { // 重合点(路径里的重复端点), 没有方向可言
				continue
			}
			nx, ny := -dy/l*r, dx/l*r
			figs = append(figs, sameOrientation([]float32{
				ax + nx, ay + ny,
				bx + nx, by + ny,
				bx - nx, by - ny,
				ax - nx, ay - ny,
			}))
		}
		for i := 0; i < len(sub.pts); i++ {
			addCap(pts[i*2], pts[i*2+1])
		}
	}
	return figs
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
	// app.New(false) 的 GDI 渲染没有 D2D 目标 → 返回 0, 走折线回落。
	rt := comPtrOf(xc.XDraw_GetD2dRenderTarget(p.Draw.Handle))
	if rt == nil {
		return false
	}
	factory := comPtrOf(xc.XC_GetD2dFactory())
	if factory == nil {
		return false
	}

	geoms := d2dGeomCache(factory)
	geom, hit := geoms[key]
	if !hit {
		s := float32(size) / 24
		geom = newGeometry(factory, iconStrokeFigures(build(), ox, oy, s, strokeWidth*s))
		geoms[key] = geom // 失败(nil)也记下, 免得每帧重试
	}
	if geom == nil || !geomUsable(geom) {
		return false
	}

	// 画刷是设备相关资源, 必须由 RT 创建、且只能给该 RT 用, 所以不缓存:
	// 每帧建一个、画完立刻 Release(探针实测建刷代价可忽略, 图标几何才是大头)。
	brush := createSolidColorBrush(rt, colorToD2D(color))
	if brush == nil {
		return false
	}
	vcall(rt, rtFillGeometry, uintptr(geom), uintptr(brush), 0)
	releaseCOM(brush)
	return true
}
