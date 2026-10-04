package xui

import (
	"math"
	"sync"

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

// fontScale 记录元素的窗口 DPI 比例(元素句柄 → 比例)。
// 只在组件构造(句柄一定有效)时写入, 绘制路径里只查表 —— 绘制回调里调
// XWidget_GetHWND/XWnd_GetDPI 遇到非预期句柄会弹"输入句柄可能无效"错误框。
var fontScale = struct {
	mu sync.RWMutex
	m  map[int]float64
}{m: map[int]float64{}}

// RegisterFontScale 在组件构造时登记所属窗口的物理/逻辑像素比(96 DPI = 1)。
// 注意两级句柄转换: XWidget_GetHWND 给的是 Win32 HWND, 而 XWnd_* 要的是
// xcgui 窗口句柄(hWindow), 必须再过一次 XC_hWindowFromHWnd —— 直接把 HWND
// 传给 XWnd_GetDPI 会弹"输入句柄可能无效"错误框。
func RegisterFontScale(hEle int) {
	hwnd := xc.XWidget_GetHWND(hEle)
	if hwnd == 0 {
		return
	}
	hWindow := xc.XC_hWindowFromHWnd(hwnd)
	if hWindow == 0 {
		return
	}
	if dpi := xc.XWnd_GetDPI(hWindow); dpi > 0 {
		fontScale.mu.Lock()
		fontScale.m[int(hEle)] = float64(dpi) / 96.0
		fontScale.mu.Unlock()
	}
}

// FontPx 把"想画的逻辑字号"换算成绘制路径需要的字号。
// 绘制路径的文字会被框架再乘一次 DPI 缩放(自绘按钮实测 1.5x 偏大, 文字顶到按钮边),
// 这里按登记的比例反除; 未登记(或 96 DPI)时原样返回。
func FontPx(hEle int, logicalPX int32) int32 {
	fontScale.mu.RLock()
	s := fontScale.m[hEle]
	fontScale.mu.RUnlock()
	if s <= 1 {
		return logicalPX
	}
	return int32(math.Round(float64(logicalPX) / s))
}

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

// FocusFrame 绘制 shadcn v4 风格的聚焦态(focus-visible:ring-3 ring-ring/50):
// 先把元素边缘 [0,3] 区域填 50% 透明焦点色, 调用方随后把正常填充画在
// 内缩 3px 的矩形上——焦点环完全替代边框。
//
// 为什么不用"边框 + 描边环"两层: 元素绘制被裁剪在自身客户区内, 外扩式
// 圆环不可见; 而内侧描边环与边框叠放会在对边产生不均匀的视觉宽度
// (环与边框重合的一侧显窄), 半透明环下还会透出边框黑线。
// 填充式单层结构没有描边居中/裁剪问题, 四边宽度天然一致。
func (p *Painter) FocusFrame(ring Color, fill Color, radius int32) {
	if ring.IsNone() {
		return
	}
	p.FillRound(&p.Rc, ring.WithAlpha(128), radius) // ring-ring/50
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
	p.FillRound(&inner, fill, radius-inset)
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
