package ui

import (
	"math"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 评分几何常量(逻辑像素)。
//
// shadcn 无官方 rating, 语义按社区通行做法定死(见 DESIGN 第四节末):
//   - 星数默认 5, 可配置; 整星或半星(0.5 步进);
//   - 悬停预览, 点击提交; 只读仅展示, 禁用 50% 透明且不响应;
//   - 整块一个 Tab 站点, ←/↓ 减一档、→/↑ 加一档、Home/End 两端,
//     键盘改变值时立即触发 OnChange(同 slider)。
const (
	ratingStarSize int32 = 20 // 单颗星边长(size-5)
	ratingStarGap  int32 = 4  // 星星间距(gap-1)
	ratingFocusPad int32 = 3  // 焦点环带宽, 元素四周留出的呼吸量
)

// 实心星与实心半星的 path(整段 SVG 交给 xui.Painter.Icon, fill 非 none 即按填充绘制)。
// 星形关于 x=12 左右对称, 半星沿中轴线闭合即可; 底下的描边星仍用 lucide 的原图,
// 两者同色叠画, 曲线/折线的细微差异看不见。
const (
	ratingStarFullSVG = `<svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 2 L15.53 7.15 L21.7 8.85 L17.71 13.85 L18 20.25 L12 18 L6 20.25 L6.29 13.85 L2.3 8.85 L8.47 7.15 Z"/></svg>`
	ratingStarHalfSVG = `<svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 2 L8.47 7.15 L2.3 8.85 L6.29 13.85 L6 20.25 L12 18 Z"/></svg>`
)

// Rating 星级评分(纯自绘: 一个元素承载全部星星, 鼠标/键盘驱动)。
//
// 为什么整块自绘: 星星的悬停预览是"指针在第几颗星"的连续状态, 拼 N 个按钮
// 就得为每颗星维护 hover/焦点并广播预览值, 还会让 Tab 在 N 颗星上各停一次。
// 单元素 + 内部命中测试与 slider 同路线。
type Rating struct {
	*widget.Element

	value    float64
	max      int
	half     bool
	readonly bool
	disabled bool

	hover float64 // 悬停预览值(-1 = 指针不在星星上)
	keyID int
	cb    func(value float64)
}

// RatingOption 评分配置项。
type RatingOption func(*Rating)

// WithRatingValue 设置初始值(越界夹紧并吸附到整星/半星)。
func WithRatingValue(v float64) RatingOption {
	return func(r *Rating) { r.value = v }
}

// WithRatingMax 设置星星数量(默认 5, <=0 按 5)。
func WithRatingMax(n int) RatingOption {
	return func(r *Rating) {
		if n > 0 {
			r.max = n
		}
	}
}

// WithRatingHalf 允许半星(0.5 步进); 默认只能选整星。
func WithRatingHalf() RatingOption {
	return func(r *Rating) { r.half = true }
}

// WithRatingReadonly 只读: 只展示当前值, 不响应鼠标/键盘, 也不能聚焦。
func WithRatingReadonly() RatingOption {
	return func(r *Rating) { r.readonly = true }
}

// WithRatingDisabled 初始即禁用(不可交互/聚焦, 整体 opacity-50)。
func WithRatingDisabled() RatingOption {
	return func(r *Rating) { r.disabled = true }
}

// NewRating 创建星级评分。cx/cy 传 0 时按内容取宽高
// (宽 = 星数×20 + 间距 + 两侧焦点环呼吸量, 高 = 20 + 呼吸量)。
func NewRating(x, y, cx, cy int32, hParent int, opts ...RatingOption) *Rating {
	r := &Rating{max: 5, hover: -1}
	for _, o := range opts {
		o(r)
	}
	r.value = r.snap(r.value)
	if cx <= 0 {
		cx = ratingNaturalWidth(r.max)
	}
	if cy <= 0 {
		cy = ratingStarSize + 2*ratingFocusPad
	}
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	r.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(true)
	ele.EnableSwitchFocus(true)
	xc.XEle_EnableDrawFocus(ele.Handle, false)
	if r.disabled {
		ele.Enable(false)
	}
	r.syncFocusable()

	ele.AddEvent_Paint(r.onPaint)
	ele.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		r.setHover(r.hoverValueAt(pPt.X))
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		r.setHover(-1)
		return 0
	})
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		r.press(pPt.X)
		return 0
	})
	ele.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		ele.Redraw(false)
		return 0
	})
	ele.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		ele.Redraw(false)
		return 0
	})
	xui.AttachWindow(ele.Handle)
	r.installKeys()
	return r
}

// Value 返回当前值。
func (r *Rating) Value() float64 { return r.value }

// SetValue 设置当前值(越界夹紧并吸附到整星/半星)。**不触发** OnChange ——
// OnChange 只响应用户点击/键盘。
func (r *Rating) SetValue(v float64) *Rating {
	if r == nil {
		return r
	}
	if nv := r.snap(v); nv != r.value {
		r.value = nv
		r.Redraw(false)
	}
	return r
}

// SetReadonly 设置只读(只展示值, 不可交互/聚焦)。
func (r *Rating) SetReadonly(v bool) *Rating {
	if r == nil || r.readonly == v {
		return r
	}
	r.readonly = v
	if v {
		r.hover = -1
	}
	r.syncFocusable()
	r.Redraw(false)
	return r
}

// SetDisabled 设置禁用态(不可交互/聚焦, 整体 opacity-50)。
func (r *Rating) SetDisabled(v bool) *Rating {
	if r == nil || r.disabled == v {
		return r
	}
	r.disabled = v
	r.Enable(!v)
	if v {
		r.hover = -1
	}
	r.syncFocusable()
	r.Redraw(false)
	return r
}

// OnChange 注册用户改变评分后的回调(value 已吸附到整星/半星)。
func (r *Rating) OnChange(fn func(value float64)) *Rating {
	if r == nil {
		return r
	}
	r.cb = fn
	return r
}

// snap 把值夹到 [0,max] 并吸附到 0.5(半星)或 1(整星)。
func (r *Rating) snap(v float64) float64 {
	if v < 0 {
		v = 0
	}
	if v > float64(r.max) {
		v = float64(r.max)
	}
	if r.half {
		return math.Round(v*2) / 2
	}
	return math.Round(v)
}

// syncFocusable 按 readonly/disabled 同步 Tab 可达性。
func (r *Rating) syncFocusable() {
	ok := !r.disabled && !r.readonly
	if r.Element != nil {
		r.EnableFocus(ok)
		r.EnableSwitchFocus(ok)
	}
}

// hoverValueAt 把客户区 x 映射成悬停预览值; 指针不在星星上返回 -1。
func (r *Rating) hoverValueAt(x int32) float64 {
	var rc xc.RECT
	xc.XEle_GetClientRect(r.Handle, &rc)
	return ratingValueAt(ratingArea(rc, r.max), x, r.max, r.half)
}

// setHover 更新悬停预览(值变化才重绘)。
func (r *Rating) setHover(v float64) {
	if r.disabled || r.readonly || r.hover == v {
		return
	}
	r.hover = v
	r.Redraw(true)
}

// press 响应鼠标点击: 提交命中值(只读/禁用忽略)。
func (r *Rating) press(x int32) {
	if r.disabled || r.readonly {
		return
	}
	xui.NotePointerPress() // 鼠标操作标记焦点来源, 保持 focus-visible 语义
	r.SetFocus()
	v := r.hoverValueAt(x)
	if v < 0 {
		return
	}
	r.setHover(v)
	if nv := r.snap(v); nv != r.value {
		r.value = nv
		r.Redraw(true)
		r.notify()
	}
}

// notify 触发 OnChange。
func (r *Rating) notify() {
	if r.cb != nil {
		r.cb(r.value)
	}
}

// installKeys 挂键盘导航。必须挂窗口级: 元素开了 EnableSwitchFocus(Tab 才能进入)后,
// xcgui 会把方向键当成"切换焦点"并在元素 KeyDown 之前执行(见 DESIGN 第八节 152)。
func (r *Rating) installKeys() {
	hWindow := xui.HostWindowOf(r.Handle)
	if hWindow == 0 {
		return
	}
	w := window.NewByHandle(hWindow)
	if w == nil {
		return
	}
	r.keyID = w.AddEvent_KeyDown(func(hWin int, wParam, lParam uintptr, pbHandled *bool) int {
		if xc.XWnd_GetFocusEle(hWin) != r.Handle {
			return 0
		}
		return r.onKeyDown(wParam, pbHandled)
	})
	r.Element.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if r.keyID > 0 {
			if xc.XC_IsHXCGUI(hWindow, xcc.XC_WINDOW) {
				w.RemoveEvent(xcc.WM_KEYDOWN, r.keyID)
			}
			r.keyID = 0
		}
		return 0
	})
}

// onKeyDown 键盘交互: ←/↓ 减一档、→/↑ 加一档、Home/End 到两端, 到端点不循环。
func (r *Rating) onKeyDown(wParam uintptr, pbHandled *bool) int {
	if r.disabled || r.readonly {
		return 0
	}
	step := 1.0
	if r.half {
		step = 0.5
	}
	v := r.value
	switch wParam {
	case 0x25, 0x28: // VK_LEFT / VK_DOWN
		v -= step
	case 0x27, 0x26: // VK_RIGHT / VK_UP
		v += step
	case 0x24: // VK_HOME
		v = 0
	case 0x23: // VK_END
		v = float64(r.max)
	default:
		return 0
	}
	xui.NoteKeyboard() // 键盘操作标记焦点来源, 焦点环才会出现
	r.hover = -1       // 键盘改值时清掉鼠标预览, 否则读数被悬停值盖住
	if nv := r.snap(v); nv != r.value {
		r.value = nv
		r.Redraw(false)
		r.notify()
	} else {
		r.Redraw(false)
	}
	*pbHandled = true
	return 0
}

// ratingNaturalWidth 返回星星区+焦点环呼吸量的自然宽度。
func ratingNaturalWidth(max int) int32 {
	if max <= 0 {
		max = 5
	}
	return 2*ratingFocusPad + int32(max)*ratingStarSize + int32(max-1)*ratingStarGap
}

// ratingArea 返回星星在客户区里的自然区域(宽 = 自然宽度, 高 = 星高 + 呼吸量, 垂直居中)。
// 元素被布局拉宽时星星仍靠左, 焦点环也只围住这块区域。
func ratingArea(rc xc.RECT, max int) xc.RECT {
	w := ratingNaturalWidth(max)
	h := ratingStarSize + 2*ratingFocusPad
	if availW := rc.Right - rc.Left; w > availW {
		w = availW
	}
	if availH := rc.Bottom - rc.Top; h > availH {
		h = availH
	}
	top := rc.Top + (rc.Bottom-rc.Top-h)/2
	return xc.RECT{Left: rc.Left, Top: top, Right: rc.Left + w, Bottom: top + h}
}

// ratingStarRect 返回第 i 颗星的绘制矩形(area 为 ratingArea 的结果)。
func ratingStarRect(area xc.RECT, i int) xc.RECT {
	h := ratingStarSize
	if availH := area.Bottom - area.Top; h > availH {
		h = availH
	}
	x := area.Left + ratingFocusPad + int32(i)*(ratingStarSize+ratingStarGap)
	y := area.Top + (area.Bottom-area.Top-h)/2
	return xc.RECT{Left: x, Top: y, Right: x + ratingStarSize, Bottom: y + h}
}

// ratingValueAt 把客户区 x 映射成评分: 第 i 颗星左半 = i+0.5(半星模式)或 i+1,
// 右半 = i+1; 落在星星区外(左右留白/元素被拉宽后的空区)返回 -1, 不参与预览与点击。
func ratingValueAt(area xc.RECT, x int32, max int, half bool) float64 {
	if max <= 0 {
		return -1
	}
	left := area.Left + ratingFocusPad
	right := left + int32(max)*ratingStarSize + int32(max-1)*ratingStarGap
	if x < left || x >= right {
		return -1
	}
	stride := ratingStarSize + ratingStarGap
	rel := x - left
	i := int(rel / stride)
	if i >= max {
		return float64(max)
	}
	off := rel - int32(i)*stride
	if off > ratingStarSize {
		off = ratingStarSize // 星与星之间的空隙算作前一颗星的右端
	}
	if half && off*2 < ratingStarSize {
		return float64(i) + 0.5
	}
	return float64(i + 1)
}

func (r *Rating) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)

	disabled := r.disabled || st.Disabled
	alpha := 1.0
	if disabled {
		alpha = 0.5 // data-disabled:opacity-50
	}

	// 只铺星星区: 元素常被 Fill 拉宽, 整块铺底会在非 Card 容器上留一条色带。
	area := ratingArea(p.Rc, r.max)
	if area.Right <= area.Left || area.Bottom <= area.Top {
		return 0
	}
	p.FillRound(&area, t.Card, 0)

	// focus-visible: 只在键盘获得的焦点上画环(鼠标点击不画, 同 shadcn 的 :focus-visible)。
	// 环带限制在星星区, 元素被拉宽时不会围出一大条空环。
	if st.Focused && !disabled && !r.readonly && xui.FocusedByKeyboard() {
		ratingFocusFrame(p, area, t.Ring, t.Card, t.RadiusMd())
	}

	display := r.value
	if r.hover >= 0 && !disabled && !r.readonly {
		display = r.hover
	}
	outline := t.MutedForeground.WithAlpha(128)
	fill := t.Primary

	for i := 0; i < r.max; i++ {
		box := ratingStarRect(area, i)
		full := display >= float64(i+1)
		half := r.half && !full && display >= float64(i)+0.5
		edge := outline
		if full || half {
			edge = fill
		}
		p.Icon(icons.IconStar, &box, edge.ScaleAlpha(alpha), 2)
		switch {
		case full:
			p.Icon(ratingStarFullSVG, &box, fill.ScaleAlpha(alpha), 2)
		case half:
			p.Icon(ratingStarHalfSVG, &box, fill.ScaleAlpha(alpha), 2)
		}
	}
	return 0
}

// ratingFocusFrame 在 area 上画 focus-visible 环(外圈 ring 色 + 内芯宿主底色)。
// 不用 p.FocusFrame: 它恒用元素整矩形, 元素被 Fill 拉宽时环会围住整行。
func ratingFocusFrame(p *xui.Painter, area xc.RECT, ring, hole xui.Color, radius int32) {
	p.FillRound(&area, ring.WithAlpha(128), radius)
	inner := xc.RECT{
		Left:   area.Left + ratingFocusPad,
		Top:    area.Top + ratingFocusPad,
		Right:  area.Right - ratingFocusPad,
		Bottom: area.Bottom - ratingFocusPad,
	}
	if inner.Right-inner.Left < 4 || inner.Bottom-inner.Top < 4 {
		return
	}
	if radius -= ratingFocusPad; radius < 0 {
		radius = 0
	}
	p.FillRound(&inner, hole, radius)
}
