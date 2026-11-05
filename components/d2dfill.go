package ui

import (
	"github.com/twgh/xcgui/wapi/d2d"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

// fillPolygonSmooth 用 D2D 路径几何填充闭合多边形(边缘由 Direct2D 抗锯齿),
// 返回 false 表示当前是 GDI 渲染或资源创建失败, 调用方回落 FillPolygonF。
//
// 为什么不用 XDraw_FillPolygonF: 它在 D2D 渲染下是无抗锯齿的硬填充, 饼图/环形图的
// 扇区与 Steps 圆点上的进度弧都会出现一圈硬台阶(见 DESIGN 第八节 160; 加密采样
// 治不了, 问题在光栅化不在几何)。hEle 用来乘元素 alpha —— 直通绘制绕过了框架的
// 元素 alpha 合成, 否则元素淡入淡出时填充仍是全不透明。
//
// 单独成文件是为了让 chart / steps 两个组件的 registry 闭包都能带上它而互不依赖。
func fillPolygonSmooth(p *xui.Painter, hEle int, col xui.Color, pts []xc.POINTF) bool {
	if p == nil || p.Draw == nil || p.Draw.Handle <= 0 || len(pts) < 3 {
		return false
	}
	rt := d2d.NewRenderTargetByHandle(xc.XDraw_GetD2dRenderTarget(p.Draw.Handle))
	factory := d2d.NewFactoryByHandle(xc.XC_GetD2dFactory())
	if rt == nil || factory == nil {
		return false
	}
	// 直通 D2D 的渲染目标上没有当前元素变换(炫彩是刷新自己的绘制命令时才施加),
	// 必须用 XDraw_ConvRect 把客户区矩形换算成画布坐标(含元素偏移 + DPI 缩放),
	// 再按这个比值设置矩阵。否则高 DPI 下图形会按"逻辑像素当物理像素"画出来 ——
	// 整体缩小 1/1.5 并偏向左上。
	w0 := float64(p.Rc.Right - p.Rc.Left)
	h0 := float64(p.Rc.Bottom - p.Rc.Top)
	if w0 <= 0 || h0 <= 0 {
		return false
	}
	canvas := p.Rc
	xc.XDraw_ConvRect(p.Draw.Handle, &canvas)
	sx := float64(canvas.Right-canvas.Left) / w0
	sy := float64(canvas.Bottom-canvas.Top) / h0
	if sx <= 0 || sy <= 0 { // 换算失败(canvas 与客户区不像同一块区域)就别硬画
		return false
	}
	prev, err := rt.GetTransform()
	if err != nil {
		return false
	}
	tm := d2d.D2D1_MATRIX_3X2_F{
		M11: float32(sx),
		M22: float32(sy),
		DX:  float32(float64(canvas.Left) - float64(p.Rc.Left)*sx),
		DY:  float32(float64(canvas.Top) - float64(p.Rc.Top)*sy),
	}
	rt.SetTransform(&tm)
	defer rt.SetTransform(&prev) // 必须成对恢复: 残留矩阵会把后续绘制一起平移/缩放

	flat := make([]float32, 0, len(pts)*2)
	for _, q := range pts {
		flat = append(flat, q.X, q.Y)
	}
	g, err := factory.NewGeometryFigures([]d2d.GeometryFigure{{Points: flat, Closed: true}})
	if err != nil || g == nil {
		return false
	}
	defer g.Release()

	r, gg, b, a := col.RGBA()
	if hEle > 0 {
		if ea := xc.XEle_GetAlpha(hEle); ea != 255 {
			a = byte(uint32(a) * uint32(ea) / 255)
		}
	}
	return rt.FillGeometryColor(g.Base(), d2d.NewColorF(r, gg, b, a)) == nil
}
