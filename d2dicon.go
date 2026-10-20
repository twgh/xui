package xui

import (
	"github.com/twgh/xcgui/svgp"
	"github.com/twgh/xcgui/wapi/d2d"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// ==================== 图标抗锯齿: D2D 几何绘制 ====================
//
// 背景: lucide 是描边图标, 老实现是"把 path 解析成折线 → XDraw_DrawLine 逐段描边"。
// 那条路在小尺寸下不干净: 折线端点被 round32 取整到逻辑像素(18px 图标上就是 ±1.5 物理像素)、
// 线条被 ceil 取整到整数线宽、8 段离散的多边形棱角, 加上重叠线段在 AA 下深浅不均。
//
// 方案: 描边**交给 Direct2D** —— path 折线(子路径保持各自的开口/闭合)建成几何, 描边子路径走
// DrawGeometry(线宽 + 圆角连接/圆头样式), 少数 fill 色点走 FillGeometry。
// 早先的做法是"CPU 侧把描边展开成填充多边形再 FillGeometry"(约 300 行), 实测同样画 2 万次:
// **DrawGeometry 约 20ms、填充已展开的环约 80ms(4 倍)**; 而且圆角/圆帽由 D2D 精确保证
// (SVG 的 round 语义, 见 d2d.RoundStrokeStyleProperties)。
//
// 几何构造(path 语法 / 分组 / 绕向统一)在 **github.com/twgh/xcgui/svgp**; 本文件只管缓存与
// 绘制编排(位置用变换、元素 alpha 乘进颜色)。
//
// 坐标系: **必须先把元素客户区坐标用 XDraw_ConvRect 换算成画布对齐坐标**
// (炫彩只在刷新自己的 XDraw 命令时才施加元素变换/裁剪, 直通调用拿到的是父级坐标系),
// 换算结果是物理像素, 不要再乘 DPI —— 见 d2dIcon / DESIGN.md 第八节 43。

// d2dGeomKey 标识一个图标的几何。
//
// **位置(ox/oy)与线宽都不在 key 里**: 几何以自身原点为基准构造、不含线宽(线宽绘制时传给
// DrawGeometry), 位置在绘制时用渲染目标的变换矩阵平移过去。于是窗口 resize(位置全变)
// 或改线宽都不会触发几何重建 —— 早先把位置烘进几何时, resize 后每个图标都是新 key, 全部重建。
type d2dGeomKey struct {
	paths string // 图标 path 原文(资源里的一条数据, 或调用方自备的 lucide path)
	size  int32  // 图标边长(物理像素)
}

// 图标几何缓存与描边样式。绘制回调每帧都会新建 Painter(Painter 是帧内对象), 所以缓存必须放包级。
//
// 几何与样式都是**设备无关**资源(归属 D2D 工厂), 因此按工厂作废: 工厂变了(D2D 设备重建)整体
// Release 重建。这样多个窗口(主窗/菜单/弹层各有各的 RT)共用一份, 不会互相把缓存顶掉。
var (
	d2dGeomFactory *d2d.ID2D1Factory
	d2dGeoms       map[d2dGeomKey]svgp.Geometries // key → 一组几何(描边 + 填充)
	d2dStrokeStyle *d2d.ID2D1StrokeStyle
)

// d2dResources 取描边样式(必要时按工厂重建), 并让几何缓存跟着作废。
func d2dResources(factory *d2d.ID2D1Factory) *d2d.ID2D1StrokeStyle {
	if factory != d2dGeomFactory {
		for _, g := range d2dGeoms {
			g.Release()
		}
		d2dGeoms = make(map[d2dGeomKey]svgp.Geometries)
		if d2dStrokeStyle != nil {
			d2dStrokeStyle.Release()
			d2dStrokeStyle = nil
		}
		d2dGeomFactory = factory
	}
	if d2dGeoms == nil {
		d2dGeoms = make(map[d2dGeomKey]svgp.Geometries)
	}
	if d2dStrokeStyle == nil {
		// SVG 语义: stroke-linecap/linejoin = round(lucide 图标全带这两个属性)
		props := d2d.RoundStrokeStyleProperties()
		if s, err := factory.CreateStrokeStyle(&props); err == nil {
			d2dStrokeStyle = s
		}
		// 建失败也不致命: DrawGeometry 传 nil 样式 = 默认(平头 + 斜接), 只是角不圆。
	}
	return d2dStrokeStyle
}

// ---------------- Painter 出图口 ----------------

// d2dIcon 尝试用 D2D 几何绘制管线把图标画在 rc(元素客户区坐标)内,
// 返回 false 表示不可用(GDI 渲染 / 资源创建失败 / 参数无效), 调用方回落折线。
func (p *Painter) d2dIcon(paths string, rc *xc.RECT, color Color, strokeWidth float32,
	build func() []svgp.SubPath, scale float32) bool {

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
		color, ox, oy, size, strokeWidth, build, scale)
}

// d2dStroke 给图标出图: 描边子路径走 DrawGeometry(线宽 + 圆角样式), 填充子路径走 FillGeometry。
// scale 是**以绘制方块中心为基准**的缩放(1 = 原大), 供"按下变小"这类按压反馈用 ——
// 缩放走渲染目标变换矩阵而不是重建几何, 否则每帧尺寸都变会击穿几何缓存。
// 返回 false 表示当前环境不可用(GDI 渲染 / 资源创建失败), 调用方回落折线渲染。
// build 只在几何缓存未命中时调用, 避免每帧重复解析 path。
func (p *Painter) d2dStroke(key d2dGeomKey, color Color, ox, oy, size int32, strokeWidth float32,
	build func() []svgp.SubPath, scale float32) bool {

	if p == nil || p.Draw == nil || p.Draw.Handle <= 0 {
		return false
	}
	// app.New(false) 的 GDI 渲染没有 D2D 目标 → 构造函数拿到 0 返回 nil, 走折线回落。
	rt := d2d.NewRenderTargetByHandle(xc.XDraw_GetD2dRenderTarget(p.Draw.Handle))
	factory := d2d.NewFactoryByHandle(xc.XC_GetD2dFactory())
	if rt == nil || factory == nil {
		return false
	}

	style := d2dResources(factory)
	geoms, hit := d2dGeoms[key]
	if !hit {
		s := float32(size) / 24
		if g, err := svgp.NewGeometries(factory, build(), s); err == nil {
			geoms = g
		}
		d2dGeoms[key] = geoms // 失败也记下(零值), 免得每帧重试
	}
	if geoms.Empty() {
		return false
	}

	// 位置/缩放不烘进几何(见 d2dGeomKey): 用渲染目标的变换矩阵把几何摆到目标位置。
	// **必须成对恢复** —— 实测炫彩自己的绘制(XDraw_*)在绘制时用的是
	// 当前变换, 残留的矩阵会把页面内容一起平移; 成对设置/恢复后不受影响。
	// 缩放以方块中心为基准: x' = x·scale + ox + size/2·(1-scale)。
	if ox != 0 || oy != 0 || scale != 1 {
		prev, err := rt.GetTransform()
		if err != nil {
			return false
		}
		tm := d2d.D2D1_MATRIX_3X2_F{
			M11: scale,
			M22: scale,
			DX:  float32(ox) + float32(size)/2*(1-scale),
			DY:  float32(oy) + float32(size)/2*(1-scale),
		}
		rt.SetTransform(&tm)
		defer rt.SetTransform(&prev)
	}

	// 直通绘制绕过了炫彩的元素 alpha 合成(实测: Dialog 淡入时关闭图标不跟着淡),
	// 这里把元素当前 alpha 乘进颜色的 alpha; 窗口级 Painter 没有元素 alpha 概念, 跳过。
	// 画刷是设备相关资源, 由 svgp.Geometries.Draw 每帧即建即放(实测建刷代价可忽略)。
	r, g, b, a := color.RGBA()
	if p.hEle != 0 && xc.XObj_GetType(p.hEle) != xcc.XC_WINDOW {
		if ea := xc.XEle_GetAlpha(p.hEle); ea != 255 {
			a = byte(uint32(a) * uint32(ea) / 255)
		}
	}
	// 线宽: lucide 网格单位(24 见方) → 物理像素
	return geoms.Draw(rt, d2d.NewColorF(r, g, b, a), strokeWidth*float32(size)/24, style) == nil
}
