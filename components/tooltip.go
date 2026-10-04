package ui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 提示气泡的几何常量(对应 shadcn tooltip.tsx: rounded-md px-3 py-1.5 text-xs + size-2.5 旋转方块箭头)
const (
	tipPadX int32 = 12 // px-3
	tipPadY int32 = 6  // py-1.5
	// tipArrow 既是箭头露出的三角形高度, 也是气泡与目标元素之间的间距 ——
	// 箭头(45° 菱形)恰好把这段间距填满, 尖端顶到元素边缘, 所以两者必须相等。
	// 菱形对角线的另一半被气泡盖住, 因此露出的三角形底宽 = 2*tipArrow。
	tipArrow  int32 = 10
	tipMaxW   int32 = 280
	tipRadius int32 = 6 // rounded-md
)

// Tooltip 提示气泡: 原生控件没有提示部件, 这里自建一个无边框置顶小窗承载。
//
// 与 shadcn tooltip.tsx(radix-nova) 对齐:
//   - 内容: bg-foreground rounded-md px-3 py-1.5 text-xs text-background, 贴边一侧加箭头;
//   - 触发: 鼠标进入元素即显示(nova 预设 delayDuration = 0), 离开即隐藏;
//   - 位置: 默认在元素上方水平居中, 上方放不下时翻到下方。
//
// 窗口为异型透明窗口(Window_Transparent_Shaped): 只有气泡与箭头不透明, 其余区域
// 真透明, 覆盖在别的元素上时不会遮挡下层内容; 再配合 WS_EX_NOACTIVATE|WS_EX_TRANSPARENT
// 实现不进任务栏、不抢焦点、鼠标穿透(气泡可能压在鼠标位置, 穿透后目标元素不会因为
// "鼠标离开"而把提示关掉)。
type Tooltip struct {
	win   *window.Window
	ele   *widget.Element
	tips  map[int]string
	bound map[int]bool
	text  string
	below bool // 当前显示在元素下方(箭头朝上)
}

// NewTooltip 创建提示管理器(窗口懒创建, 首次显示时才建)。
func NewTooltip() *Tooltip {
	return &Tooltip{tips: map[int]string{}, bound: map[int]bool{}}
}

// Bind 把提示绑定到元素: 鼠标进入显示, 离开隐藏。同一个 Tooltip 可绑定任意多个元素。
func (t *Tooltip) Bind(hEle int, text string) *Tooltip {
	t.tips[hEle] = text
	if t.bound[hEle] {
		return t
	}
	tooltipReg[hEle] = t
	w := widget.NewElementByHandle(hEle)
	if w == nil {
		return t
	}
	t.bound[hEle] = true
	w.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		t.show(hEle)
		return 0
	})
	w.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		t.show(hEle)
		return 0
	})
	w.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		t.Hide()
		return 0
	})
	return t
}

// Unbind 解除绑定。
func (t *Tooltip) Unbind(hEle int) {
	delete(t.tips, hEle)
	delete(t.bound, hEle)
	delete(tooltipReg, hEle)
}

// HideTooltip 隐藏绑定在指定元素上的提示气泡(未绑定则无操作)。
//
// 为什么要主动隐藏: 元素自身弹层(如下拉列表)打开后是另一个窗口接管鼠标, 触发元素
// 收不到 MouseLeave, 提示会一直挂着(实测 ComboBox 弹层)。这类组件应在弹层打开时
// 调用 HideTooltip(hEle)。
var tooltipReg = map[int]*Tooltip{}

func HideTooltip(hEle int) {
	if tt, ok := tooltipReg[hEle]; ok {
		tt.Hide()
	}
}

// Hide 隐藏提示。
func (t *Tooltip) Hide() {
	if t.win != nil {
		t.win.ShowWindow(xcc.SW_HIDE)
	}
}

func (t *Tooltip) show(hEle int) {
	text := t.tips[hEle]
	if text == "" {
		return
	}
	t.ensure()
	t.text = text

	// 量文字尺寸: 必须用"绘制时同一个字体与限宽", 否则气泡会比文字宽一圈
	// (此前这里量的是 14px 主题字体, 而实际绘制用 12px)
	var sz xc.SIZE
	app.GetTextShowRect(text, -1, xui.FontHandle(xui.FontSpec{
		Family: xui.Current().FontDefault.Family,
		Size:   xui.Current().FontDefault.Size - 2, // text-xs
	}), 0, tipMaxW-tipPadX*2, &sz)
	bw := sz.CX + tipPadX*2
	if bw > tipMaxW {
		bw = tipMaxW
	}
	bh := sz.CY + tipPadY*2
	ww, wh := bw, bh+tipArrow

	// 元素在窗口客户区中的矩形 -> 屏幕逻辑坐标。
	// 注意: XEle_GetWndClientRect 的原点就是"窗口 rect 原点"(含阴影边距), 不再叠加
	// XWnd_GetClientRect 的客户区内缩 —— 叠加过一次, 气泡会整体下移阴影宽度(实测 8px,
	// 箭头尖端扎进目标元素里)。
	var rc, wr xc.RECT
	xc.XEle_GetWndClientRect(hEle, &rc)
	hWnd := windowOf(hEle)
	xc.XWnd_GetRect(hWnd, &wr)
	sx, sy := wr.Left+rc.Left, wr.Top+rc.Top
	ex, ey := rc.Right-rc.Left, rc.Bottom-rc.Top

	// 窗口整高 = 气泡 + 箭头, 箭头恰好填满这段间距, 所以窗口边缘要贴住元素边缘
	// (不再额外加间距, 否则箭头尖端会悬在半空, 看起来"提示框飘得太高")。
	x := sx + ex/2 - ww/2
	y := sy - wh
	t.below = false
	if y < 0 { // 上方放不下 -> 翻到下方
		y = sy + ey
		t.below = true
	}
	if sw := screenWidth(hWnd); x+ww > sw {
		x = sw - ww
	}
	if x < 0 {
		x = 0
	}

	xc.XEle_SetRect(t.ele.Handle, &xc.RECT{Left: 0, Top: 0, Right: ww, Bottom: wh}, false, xcc.AdjustLayout_No)
	// 注意: XWnd_SetWindowPos 直接透传给 Win32, 用的是物理像素(框架内部坐标是逻辑像素,
	// 窗口创建时由框架按 DPI 缩放), 所以这里要自己乘 DPI 比例, 否则窗口会跑到 1/scale 处。
	dpi := xc.XWnd_GetDPI(hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	scale := float64(dpi) / 96
	t.win.SetWindowPos(xcc.HWND_TOPMOST,
		int32(float64(x)*scale), int32(float64(y)*scale),
		int32(float64(ww)*scale), int32(float64(wh)*scale),
		xcc.SWP_NOACTIVATE|xcc.SWP_SHOWWINDOW)
	wapi.ShowWindow(t.win.GetHWND(), 4) // SW_SHOWNOACTIVATE
	t.ele.Redraw(false)
	t.win.Redraw(true)
}

func (t *Tooltip) ensure() {
	if t.win != nil {
		return
	}
	w := window.New(0, 0, 10, 10, "", 0, xcc.Window_Style_Nothing)
	w.SetBorderSize(0, 0, 0, 0)
	// 真透明: 异型窗口带透明通道, 窗口形状 = 自身绘制内容中不透明像素的并集。
	// 本组件的气泡与箭头全部由 paint 回调自绘(元素已 EnableBkTransparent), 因此
	// 气泡圆角之外、箭头两侧天然是全透明的, 压在别的元素上也不会露出方形底。
	w.SetTransparentType(xcc.Window_Transparent_Shaped)
	w.SetTransparentAlpha(255)
	hwnd := w.GetHWND()
	ex := wapi.GetWindowLongPtrW(hwnd, wapi.GWL_EXSTYLE)
	wapi.SetWindowLongPtrW(hwnd, wapi.GWL_EXSTYLE,
		ex|int(xcc.WS_EX_TOOLWINDOW)|int(xcc.WS_EX_NOACTIVATE)|int(xcc.WS_EX_TRANSPARENT))
	w.SetWindowPos(xcc.HWND_TOPMOST, 0, 0, 0, 0,
		xcc.SWP_NOMOVE|xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE)

	ele := widget.NewElement(0, 0, 10, 10, w.Handle)
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(t.onPaint)
	t.win, t.ele = w, ele
}

func (t *Tooltip) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()

	// 气泡矩形: 窗口 = 气泡 + 箭头, 箭头在贴元素的那一侧
	bubble := p.Rc
	bubble.Bottom = p.Rc.Top + p.H() - tipArrow
	if t.below {
		bubble.Top = p.Rc.Top + tipArrow
	}

	// 箭头: 45° 旋转的方块(菱形), 中心压在气泡边线上; 先画箭头再画气泡, 气泡盖住内侧一半,
	// 露出的三角形高 tipArrow(底宽 2*tipArrow), 尖端正好落在窗口边缘 = 目标元素边缘上。
	cx := (p.Rc.Left + p.Rc.Right) / 2
	cy := bubble.Bottom - 1
	if t.below {
		cy = bubble.Top + 1
	}
	half := tipArrow
	p.Draw.SetBrushColor(uint32(th.Foreground))
	p.Draw.FillPolygon([]xc.POINT{
		{X: cx, Y: cy - half},
		{X: cx + half, Y: cy},
		{X: cx, Y: cy + half},
		{X: cx - half, Y: cy},
	})

	// 气泡: bg-foreground, rounded-md
	p.FillRound(&bubble, th.Foreground, tipRadius)
	textRc := xc.RECT{
		Left:   bubble.Left + tipPadX,
		Top:    bubble.Top + tipPadY,
		Right:  bubble.Right - tipPadX,
		Bottom: bubble.Bottom - tipPadY,
	}
	if f := xui.FontHandle(xui.FontSpec{
		Family: th.FontDefault.Family,
		Size:   th.FontDefault.Size - 2, // text-xs
	}); f != 0 {
		p.Draw.SetFont(f)
	}
	p.Text(t.text, &textRc, th.Background, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	return 0
}

// windowOf 返回元素所属窗口句柄。
func windowOf(hEle int) int {
	return xc.XC_hWindowFromHWnd(xc.XWidget_GetHWND(hEle))
}

// screenWidth 返回窗口所在显示器的逻辑宽度(物理宽度按 DPI 折算), 用于把提示夹在屏幕内。
func screenWidth(hWindow int) int32 {
	dpi := xc.XWnd_GetDPI(hWindow)
	if dpi <= 0 {
		dpi = 96
	}
	return int32(float64(wapi.GetSystemMetrics(wapi.SM_CXVIRTUALSCREEN)) * 96 / float64(dpi))
}
