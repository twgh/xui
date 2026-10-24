package ui

import (
	"math"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 加载指示器默认 16(size-4), 与 shadcn spinner.tsx 的 `size-4 animate-spin` 一致。
const spinnerDefaultSize int32 = 16

// SpinnerOption 加载指示器配置项。
type SpinnerOption func(*Spinner)

// WithSpinnerSize 覆盖边长(逻辑像素), 默认 16。
func WithSpinnerSize(size int32) SpinnerOption {
	return func(s *Spinner) {
		if size > 0 {
			s.size = size
		}
	}
}

// WithSpinnerMuted 使用弱化色(muted-foreground)。
func WithSpinnerMuted() SpinnerOption {
	return func(s *Spinner) { s.muted = true }
}

// WithSpinnerColorFunc 指定语义色解析器(每次绘制实时取主题令牌)。
// 用于在 primary 按钮、tooltip 这类不同底色上跟随前景色。
func WithSpinnerColorFunc(fn func(*xui.Theme) xui.Color) SpinnerOption {
	return func(s *Spinner) { s.colorFn = fn }
}

// Spinner 加载指示器: 一个持续旋转的 loader 图标, 1s/圈(animate-spin)。
//
// 载体是透明 Element, 绘制 Loader2 图标并用运行时 xui.Spin 取相位 ——
// 不要自己起定时器。颜色在 paint 回调里按当前主题实时求值。
//
// 作为布局项使用时它已有固定宽高(见 WithSpinnerSize), 直接塞进行列即可:
//
//	row := xui.HStack(parent, 24, 8)
//	ui.NewSpinner(row.Handle, ui.WithSpinnerSize(16))
//	ui.NewLabel(0, 0, 0, 20, "加载中", row.Handle).SetFontSize(14)
//
// 需要"按钮内加载"时用 Button 的 WithButtonLoading(它自带同款旋转),
// 本组件用于按钮外的独立加载态。
type Spinner struct {
	*widget.Element
	size     int32
	muted    bool
	colorFn  func(*xui.Theme) xui.Color
	spinning bool
}

// NewSpinner 创建加载指示器并挂到 hParent。默认 16×16, 可用 WithSpinnerSize 调整。
func NewSpinner(hParent int, opts ...SpinnerOption) *Spinner {
	s := &Spinner{size: spinnerDefaultSize, spinning: true}
	for _, o := range opts {
		o(s)
	}
	ele := widget.NewElement(0, 0, s.size, s.size, hParent)
	if ele == nil {
		return nil
	}
	s.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, s.size)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, s.size)
	ele.AddEvent_Paint(s.onPaint)
	xui.AttachWindow(ele.Handle)
	return s
}

// SetSpinning 打开/关闭旋转。关闭后保留图标当前姿态, 不占动画定时器。
func (s *Spinner) SetSpinning(spinning bool) *Spinner {
	if s == nil || s.spinning == spinning {
		return s
	}
	s.spinning = spinning
	s.Redraw(false)
	return s
}

// Spinning 报告当前是否在旋转。
func (s *Spinner) Spinning() bool {
	return s != nil && s.spinning
}

func (s *Spinner) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	color := t.Foreground
	if s.colorFn != nil {
		color = s.colorFn(t)
	} else if s.muted {
		color = t.MutedForeground
	}
	phase := xui.Spin(hEle, s.spinning)
	p.IconRotated(icons.IconLoader2, &p.Rc, color, 2, phase*2*math.Pi)
	return 0
}
