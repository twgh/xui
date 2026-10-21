package xui

import (
	"github.com/twgh/xcgui/drawx"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// Painter 封装绘制回调中的常用操作: 圆角壳、焦点环、边框、状态取色。
// 组件的 onPaint 一般以 BeginPaint 开头, 以它返回的 Painter 操作。
//
// 重要: 绘制坐标系是逻辑像素(DPI=96 基准)。XEle_GetClientRect 返回的
// 是未缩放的逻辑矩形, 框架在渲染时统一做逻辑→物理变换, 因此这里
// 不要对尺寸做任何 DPI 预缩放(实测 DPI=144 时缩放会导致双重放大,
// 指示器撑满元素客户区而被上下裁剪)。
type Painter struct {
	Draw  *drawx.Draw
	hEle  int
	Rc    xc.RECT // 元素客户区(逻辑坐标)
	Scale float64 // 物理/逻辑像素比(高 DPI 下 >1)
}

// 关于"自绘文字字号要不要按 DPI 反除": 曾经怀疑绘制路径会二次乘 DPI 而做过
// 字号反除, 实测反证——反除后文字明显偏小, 所以绘制路径的字号就是逻辑像素,
// 保持 b.Spec().font 原样传入即可。真正要小心的是量测口径:
// app.GetTextShowRect 在 144 DPI 下返回值约为实际绘制宽度的一半(见 dialog.go 页脚量宽)。
//
// 另外句柄有多种: XWidget_GetHWND 取到的是 Win32 HWND, 而 XWnd_* 系列要的是
// xcgui 窗口句柄(hWindow), 中间必须过 XC_hWindowFromHWnd; 把 HWND 直接喂给
// XWnd_GetDPI 会弹"输入句柄可能无效"错误框, 且窗口句柄也不能喂给 XWidget_GetHWND。

// BeginPaint 从元素绘制回调的 hDraw 构造 Painter, 并启用抗锯齿。
func BeginPaint(hEle, hDraw int) *Painter {
	d := drawx.NewByHandle(hDraw)
	if d == nil {
		return nil
	}
	d.EnableSmoothingMode(true)
	var rc xc.RECT
	xc.XEle_GetClientRect(hEle, &rc)
	return &Painter{Draw: d, hEle: hEle, Rc: rc}
}

// BeginPaintWindow 从窗口绘制回调的 hDraw 构造 Painter(填窗口客户区)。
func BeginPaintWindow(hWindow, hDraw int) *Painter {
	d := drawx.NewByHandle(hDraw)
	if d == nil {
		return nil
	}
	d.EnableSmoothingMode(true)
	var rc xc.RECT
	xc.XWnd_GetClientRect(hWindow, &rc)
	return &Painter{Draw: d, hEle: hWindow, Rc: rc}
}

// W, H 返回客户区宽高(逻辑像素)。
func (p *Painter) W() int32 { return p.Rc.Right - p.Rc.Left }

// H 是客户区高度。
func (p *Painter) H() int32 { return p.Rc.Bottom - p.Rc.Top }

// FillRound 填充圆角矩形。
func (p *Painter) FillRound(rc *xc.RECT, color Color, radius int32) {
	if color.IsNone() {
		return
	}
	p.Draw.SetBrushColor(uint32(color))
	p.Draw.FillRoundRect(rc, radius, radius)
}

// StrokeRound 绘制圆角矩形边框。注意线宽沿路径居中, 外扩约 lineWidth/2,
// 贴边的描边会被元素边界裁掉, 需要完整可见时请内缩。
func (p *Painter) StrokeRound(rc *xc.RECT, color Color, radius int32, lineWidth int32) {
	if color.IsNone() {
		return
	}
	p.Draw.SetLineWidth(lineWidth)
	p.Draw.SetBrushColor(uint32(color))
	p.Draw.DrawRoundRect(rc, radius, radius)
}

// Corners 是四角独立的圆角半径(逻辑像素), 用于按钮组这类"外角圆、内角方"的壳。
type Corners struct {
	TopLeft     int32
	TopRight    int32
	BottomRight int32
	BottomLeft  int32
}

// UniformCorners 返回四角相同的圆角。
func UniformCorners(radius int32) Corners {
	return Corners{radius, radius, radius, radius}
}

// Inset 把四角半径同时收缩 d(不小于 0), 供焦点环内芯与描边内缩使用。
func (c Corners) Inset(d int32) Corners {
	f := func(r int32) int32 {
		if r -= d; r < 0 {
			return 0
		}
		return r
	}
	return Corners{f(c.TopLeft), f(c.TopRight), f(c.BottomRight), f(c.BottomLeft)}
}

// FillRoundCorners 填充四角半径可不同的圆角矩形。
func (p *Painter) FillRoundCorners(rc *xc.RECT, color Color, c Corners) {
	if color.IsNone() {
		return
	}
	p.Draw.SetBrushColor(uint32(color))
	p.Draw.FillRoundRectEx(rc, c.TopLeft, c.TopRight, c.BottomRight, c.BottomLeft)
}

// StrokeRoundCorners 绘制四角半径可不同的圆角矩形边框(裁剪/线宽语义同 StrokeRound)。
func (p *Painter) StrokeRoundCorners(rc *xc.RECT, color Color, c Corners, lineWidth int32) {
	if color.IsNone() {
		return
	}
	p.Draw.SetLineWidth(lineWidth)
	p.Draw.SetBrushColor(uint32(color))
	p.Draw.DrawRoundRectEx(rc, c.TopLeft, c.TopRight, c.BottomRight, c.BottomLeft)
}

// FocusFrame 绘制 shadcn v4 风格的聚焦态(focus-visible:ring-3 ring-ring/50):
// 先把元素边缘 [0,3] 区域填 50% 透明焦点色, 调用方随后把正常填充画在
// 内缩 3px 的矩形上——焦点环完全替代边框。
//
// 为什么不用"边框 + 描边环"两层: 元素绘制被裁剪在自身客户区内, 外扩式
// 圆环不可见; 而内侧描边环与边框叠放会在对边产生不均匀的视觉宽度
// (环与边框重合的一侧显窄), 半透明环下还会透出边框黑线。
// 填充式单层结构没有描边居中/裁剪问题, 四边宽度天然一致。
func (p *Painter) FocusFrame(ring Color, fill Color, radius int32) {
	p.RingFrame(ring.WithAlpha(128), fill, radius) // ring-ring/50
}

// RingFrame 与 FocusFrame 同构, 但环带色**按传入值原样**使用(不再统一乘 50%)。
// shadcn 的 aria-invalid 环改成 destructive/20(暗色 /40), 与 focus-visible 的
// ring-ring/50 不是同一个透明度, 所以需要一个能自定义环带色的入口。
func (p *Painter) RingFrame(ring Color, fill Color, radius int32) {
	p.RingFrameCorners(ring, fill, UniformCorners(radius))
}

// FocusFrameCorners 是四角独立的 FocusFrame(按钮组的外角圆/内角方)。
func (p *Painter) FocusFrameCorners(ring Color, fill Color, c Corners) {
	p.RingFrameCorners(ring.WithAlpha(128), fill, c)
}

// RingFrameCorners 是四角独立的 RingFrame。
func (p *Painter) RingFrameCorners(ring Color, fill Color, c Corners) {
	if ring.IsNone() {
		return
	}
	p.FillRoundCorners(&p.Rc, ring, c)
	if fill.IsNone() {
		return
	}
	const inset = int32(3)
	inner := xc.RECT{
		Left:   p.Rc.Left + inset,
		Top:    p.Rc.Top + inset,
		Right:  p.Rc.Right - inset,
		Bottom: p.Rc.Bottom - inset,
	}
	if inner.Right-inner.Left < 4 || inner.Bottom-inner.Top < 4 {
		return
	}
	p.FillRoundCorners(&inner, fill, c.Inset(inset))
}

// Line 绘制一条线段。
func (p *Painter) Line(x1, y1, x2, y2 int32, color Color, lineWidth int32) {
	if color.IsNone() {
		return
	}
	p.Draw.SetLineWidth(lineWidth)
	p.Draw.SetBrushColor(uint32(color))
	p.Draw.DrawLine(x1, y1, x2, y2)
}

// Text 在指定矩形绘制文字, align 为 xcc.TextAlignFlag_* 组合。
// 颜色为 None 时跳过。
func (p *Painter) Text(str string, rc *xc.RECT, color Color, align xcc.TextFormatFlag_) {
	if color.IsNone() || str == "" {
		return
	}
	p.Draw.SetBrushColor(uint32(color))
	p.Draw.SetTextAlign(align)
	p.Draw.DrawText(str, rc)
}

// TextCenter 在元素中央绘制文字。
func (p *Painter) TextCenter(str string, color Color) {
	p.Text(str, &p.Rc, color, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
}

// TextF 是 Text 的浮点矩形版本, 用于高 DPI 下需要亚逻辑像素位移的场合
// (例如按钮按压时让文字与图标对齐到同一个物理像素, 见 DESIGN 第八节 88)。
func (p *Painter) TextF(str string, rc *xc.RECTF, color Color, align xcc.TextFormatFlag_) {
	if color.IsNone() || str == "" {
		return
	}
	p.Draw.SetBrushColor(uint32(color))
	p.Draw.SetTextAlign(align)
	p.Draw.DrawTextF(str, rc)
}

// State 是组件绘制的统一状态输入。
type State struct {
	Disabled bool
	Hover    bool
	Pressed  bool
	Focused  bool
}

// StateOf 从元素状态标志解析出组件关心的四态。
// 底层: xc.XEle_GetStateFlags 返回 xcc.CombinedState, 各标志位定义见 xcc/combinedstate.go。
func StateOf(hEle int) State {
	f := xc.XEle_GetStateFlags(hEle)
	return State{
		Disabled: f&xcc.Element_State_Flag_Disable != 0,
		Hover:    f&xcc.Element_State_Flag_Stay != 0,
		Pressed:  f&xcc.Element_State_Flag_Down != 0,
		Focused:  f&xcc.Element_State_Flag_Focus != 0,
	}
}
