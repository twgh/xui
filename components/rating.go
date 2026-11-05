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
// shadcn 无官方 rating, 语义按社区通行做法 + Ant Design Rate 定死(见 DESIGN 第四节末):
//   - 星数默认 5, 可配置; 整星或半星(0.5 步进); 三档尺寸;
//   - 悬停预览, 点击提交; 再次点击当前值清零(allowClear, 默认开);
//   - 只读仅展示, 禁用 50% 透明且不响应; 键盘可用 WithRatingKeyboard 关闭;
//   - 整块一个 Tab 站点, ←/↓ 减一档、→/↑ 加一档、Home/End 两端,
//     键盘改变值时立即触发 OnChange(同 slider)。
const (
	ratingStarGap    int32   = 4   // 星星间距(gap-1)
	ratingFocusPad   int32   = 3   // 焦点环带宽, 元素四周留出的呼吸量
	ratingLabelGap   int32   = 8   // 星星区与右侧文案的间距
	ratingHoverScale float32 = 1.1 // 悬停那颗星的放大倍率(AntD starHoverScale)
)

// RatingSize 尺寸档(默认 RatingMedium)。
//
// 星边长取 AntD 的 controlHeight×0.625 口径: small 24→15, medium 32→20, large 40→25。
// 只影响几何与字号留白, 不改变值语义; 自然宽高按它算, 所以要在构造时指定。
type RatingSize int

const (
	RatingMedium RatingSize = iota // 20(默认)
	RatingSmall                    // 15
	RatingLarge                    // 25
)

func ratingStarSizeFor(s RatingSize) int32 {
	switch s {
	case RatingSmall:
		return 15
	case RatingLarge:
		return 25
	default:
		return 20
	}
}

// 实心星与实心半星的 path(整段 SVG 交给 xui.Painter.Icon, fill 非 none 即按填充绘制)。
// 星形关于 x=12 左右对称, 半星沿中轴线闭合即可; 底下的描边星仍用 lucide 的原图,
// 两者同色叠画, 曲线/折线的细微差异看不见。
// 自定义图标(WithRatingIcons)三态必须成套给, half 也要是沿 x=12 闭合的左半 ——
// Painter 没有裁剪 API, 半星只能靠几何本身表达。
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

	allowClear bool       // 再次点击当前值归 0(默认开)
	keyboard   bool       // 方向键/Home/End 是否接管(默认开)
	size       RatingSize // 尺寸档
	starSize   int32      // 由 size 解析出的星边长
	labels     []string   // 文案展现: 按 ceil(value)-1 取文案
	labelW     int32      // 最长文案宽(构造时量一次, 决定自然宽)
	outlineSVG string     // 空态描边图标
	fullSVG    string     // 实心图标
	halfSVG    string     // 左半实心图标

	hover   float64 // 悬停预览值(-1 = 指针不在星星上)
	cleaned float64 // 刚被"再次点击清除"的值: 指针还停原位时不恢复预览
	keyID   int
	cb      func(value float64)
	hoverCb func(value float64)
	focusCb func()
	blurCb  func()
	keyCb   func(vk uintptr)
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

// WithRatingAllowClear 设置"再次点击当前值归 0"(默认 true, 同 AntD allowClear)。
// 关闭后点击已选中的星不再有任何变化, 只能靠键盘 Home 或 SetValue(0) 清零。
func WithRatingAllowClear(v bool) RatingOption {
	return func(r *Rating) { r.allowClear = v }
}

// WithRatingKeyboard 设置方向键/Home/End 是否接管(默认 true)。
// 关掉后方向键交回 xcgui 的焦点切换, 组件仍可点击、仍是一个 Tab 站点。
func WithRatingKeyboard(v bool) RatingOption {
	return func(r *Rating) { r.keyboard = v }
}

// WithRatingSize 设置尺寸档(默认 RatingMedium)。自然宽高按它算, 运行中不可改。
func WithRatingSize(s RatingSize) RatingOption {
	return func(r *Rating) {
		if s == RatingSmall || s == RatingMedium || s == RatingLarge {
			r.size = s
		}
	}
}

// WithRatingLabels 设置文案展现: 星星区右侧画 ceil(当前值或悬停预览)-1 对应的文案。
// 数组长度可小于星数(越界不画); 值为 0 不画文案。文案宽度计入元素自然宽,
// 所以必须在构造时给, 之后不提供 setter(元素矩形不会自己变宽)。
func WithRatingLabels(labels []string) RatingOption {
	return func(r *Rating) { r.labels = append([]string(nil), labels...) }
}

// WithRatingIcons 替换三态图标(空态描边 / 实心 / 左半实心)。
// 口径同 StepItem.Icon 与 Painter.Icon: icons.IconXxx 常量、path 数据或整段 SVG。
// 空字符串表示保留该态的默认星形; 三态应成套给, 否则半星形状对不上。
func WithRatingIcons(outline, full, half string) RatingOption {
	return func(r *Rating) {
		if outline != "" {
			r.outlineSVG = outline
		}
		if full != "" {
			r.fullSVG = full
		}
		if half != "" {
			r.halfSVG = half
		}
	}
}

// NewRating 创建星级评分。cx/cy 传 0 时按内容取宽高
// (宽 = 星数×星边长 + 间距 + 两侧焦点环呼吸量 [+ 文案区], 高 = 星边长 + 呼吸量)。
func NewRating(x, y, cx, cy int32, hParent int, opts ...RatingOption) *Rating {
	r := &Rating{
		max: 5, hover: -1, cleaned: -1,
		allowClear: true, keyboard: true,
		size:       RatingMedium,
		outlineSVG: icons.IconStar,
		fullSVG:    ratingStarFullSVG,
		halfSVG:    ratingStarHalfSVG,
	}
	for _, o := range opts {
		o(r)
	}
	r.starSize = ratingStarSizeFor(r.size)
	r.value = r.snap(r.value)
	r.labelW = ratingLongestLabel(r.labels)
	if cx <= 0 {
		cx = r.naturalWidth()
	}
	if cy <= 0 {
		cy = r.starSize + 2*ratingFocusPad
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
		r.clearHover()
		return 0
	})
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		r.press(pPt.X)
		return 0
	})
	ele.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		ele.Redraw(false)
		if r.focusCb != nil {
			r.focusCb()
		}
		return 0
	})
	ele.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		ele.Redraw(false)
		r.clearHover()
		if r.blurCb != nil {
			r.blurCb()
		}
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
		r.clearHover()
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
		r.clearHover()
	}
	r.syncFocusable()
	r.Redraw(false)
	return r
}

// SetAllowClear 设置"再次点击当前值归 0"。
func (r *Rating) SetAllowClear(v bool) *Rating {
	if r != nil {
		r.allowClear = v
		if !v {
			r.cleaned = -1
		}
	}
	return r
}

// SetKeyboard 设置方向键/Home/End 是否接管。
func (r *Rating) SetKeyboard(v bool) *Rating {
	if r != nil {
		r.keyboard = v
	}
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

// OnHoverChange 注册悬停预览变化的回调。value 是预览值; -1 表示指针离开星星区
// (含"再次点击清除"后预览被压住)。只读/禁用时不触发。
func (r *Rating) OnHoverChange(fn func(value float64)) *Rating {
	if r == nil {
		return r
	}
	r.hoverCb = fn
	return r
}

// OnFocus 注册获得焦点时的回调(同 AntD onFocus)。
func (r *Rating) OnFocus(fn func()) *Rating {
	if r == nil {
		return r
	}
	r.focusCb = fn
	return r
}

// OnBlur 注册失去焦点时的回调(同 AntD onBlur)。
func (r *Rating) OnBlur(fn func()) *Rating {
	if r == nil {
		return r
	}
	r.blurCb = fn
	return r
}

// OnKeyDown 注册按键回调(仅当本组件是焦点元素时)。vk 是虚拟键码(VK_*);
// keyboard=false 时也会转发, 方便表单自己处理方向键。
func (r *Rating) OnKeyDown(fn func(vk uintptr)) *Rating {
	if r == nil {
		return r
	}
	r.keyCb = fn
	return r
}

// Blur 移除焦点(同 AntD blur())。只在焦点确实在本组件上时动手, 避免误清别的控件。
// xcgui 没有元素级 KillFocus API, 用宿主窗口的 SetFocusEle(0) 把焦点交还窗口。
func (r *Rating) Blur() *Rating {
	if r == nil || r.Element == nil {
		return r
	}
	if hWindow := xui.HostWindowOf(r.Handle); hWindow != 0 &&
		xc.XWnd_GetFocusEle(hWindow) == r.Handle {
		xc.XWnd_SetFocusEle(hWindow, 0)
	}
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

// pickValue 返回点击命中的新值: allowClear 时再次点当前值归 0。
// 纯逻辑, 便于单测; 命中值已经过 snap。
func (r *Rating) pickValue(v float64) float64 {
	if r.allowClear && v == r.value {
		return 0
	}
	return v
}

// labelFor 按 ceil(value)-1 取文案, 值 <=0 / 越界 / 未配文案时返回空。
func (r *Rating) labelFor(v float64) string {
	if v <= 0 || len(r.labels) == 0 {
		return ""
	}
	i := int(math.Ceil(v)) - 1
	if i < 0 || i >= len(r.labels) {
		return ""
	}
	return r.labels[i]
}

// hoverIndex 返回悬停预览落在第几颗星(-1 = 无预览), 用于逐项 hover 缩放。
func (r *Rating) hoverIndex() int {
	if r.hover <= 0 {
		return -1
	}
	i := int(math.Ceil(r.hover)) - 1
	if i >= r.max {
		return r.max - 1
	}
	return i
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
	return ratingValueAt(ratingArea(rc, r.max, r.starSize), x, r.max, r.half, r.starSize)
}

// setHover 更新悬停预览(值变化才重绘并通知)。
// 刚清除的值不要立刻恢复预览: 指针没动, MouseMove 还会送来同一个值
// (AntD 的 cleanedValue 语义), 否则画面会显示"已清零但星星仍亮着"。
func (r *Rating) setHover(v float64) {
	if r.disabled || r.readonly {
		return
	}
	if r.cleaned >= 0 {
		if v == r.cleaned {
			v = -1
		} else {
			r.cleaned = -1
		}
	}
	if r.hover == v {
		return
	}
	r.hover = v
	r.Redraw(true)
	r.notifyHover()
}

// clearHover 清掉预览与清除压制(鼠标离开 / 只读 / 禁用 / 键盘改值)。
// 不走 setHover: 只读与禁用时 setHover 会直接返回, 悬停态却必须清干净。
func (r *Rating) clearHover() {
	r.cleaned = -1
	if r.hover != -1 {
		r.hover = -1
		r.notifyHover()
	}
	// 必须重绘一帧: 逐项 hover 过渡的回退由 paint 里的 HoverItem(active=false)
	// 驱动, 少这一帧就永远停在"放大"的终态(鼠标离开元素后不会再有 MouseMove)。
	// 状态本来就是 -1 也要重绘, 过渡值可能还停在 1。
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
	nv := r.snap(v)
	next := r.pickValue(nv)
	if next == 0 && nv == r.value && r.allowClear {
		r.cleaned = nv
		r.setHover(v) // 映射成 -1: 清掉预览并通知 OnHoverChange
		r.value = 0
		r.Redraw(true)
		r.notify()
		return
	}
	r.cleaned = -1
	r.setHover(v)
	if next != r.value {
		r.value = next
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

// notifyHover 触发 OnHoverChange。
func (r *Rating) notifyHover() {
	if r.hoverCb != nil {
		r.hoverCb(r.hover)
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
	if r.keyCb != nil {
		r.keyCb(wParam) // 观察型回调: 键盘导航关掉时也要能收到按键
	}
	if r.disabled || r.readonly || !r.keyboard {
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
	r.clearHover()     // 键盘改值时清掉鼠标预览, 否则读数被悬停值盖住
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

// ratingLongestLabel 量最长文案宽(与绘制同字体), 决定元素自然宽。
// 只量宽度不缓存颜色: paint 里的颜色一律实时取主题令牌。
func ratingLongestLabel(labels []string) int32 {
	if len(labels) == 0 {
		return 0
	}
	f := xui.FontHandle(xui.Current().FontDefault)
	var w int32
	for _, s := range labels {
		if n := xui.TextWidth(s, f); n > w {
			w = n
		}
	}
	return w
}

// naturalWidth 返回含文案区的自然宽度。
func (r *Rating) naturalWidth() int32 {
	w := ratingNaturalWidth(r.max, r.starSize)
	if r.labelW > 0 {
		w += ratingLabelGap + r.labelW
	}
	return w
}

// ratingNaturalWidth 返回星星区+焦点环呼吸量的自然宽度。
func ratingNaturalWidth(max int, starSize int32) int32 {
	if max <= 0 {
		max = 5
	}
	if starSize <= 0 {
		starSize = ratingStarSizeFor(RatingMedium)
	}
	return 2*ratingFocusPad + int32(max)*starSize + int32(max-1)*ratingStarGap
}

// ratingArea 返回星星在客户区里的自然区域(宽 = 自然宽度, 高 = 星高 + 呼吸量, 垂直居中)。
// 元素被布局拉宽时星星仍靠左, 焦点环也只围住这块区域。
func ratingArea(rc xc.RECT, max int, starSize int32) xc.RECT {
	w := ratingNaturalWidth(max, starSize)
	h := starSize + 2*ratingFocusPad
	if availW := rc.Right - rc.Left; w > availW {
		w = availW
	}
	if availH := rc.Bottom - rc.Top; h > availH {
		h = availH
	}
	top := rc.Top + (rc.Bottom-rc.Top-h)/2
	return xc.RECT{Left: rc.Left, Top: top, Right: rc.Left + w, Bottom: top + h}
}

// ratingLabelRect 返回星星区右侧的文案矩形。
func ratingLabelRect(area xc.RECT, labelW int32) xc.RECT {
	return xc.RECT{
		Left:   area.Right + ratingLabelGap,
		Top:    area.Top,
		Right:  area.Right + ratingLabelGap + labelW,
		Bottom: area.Bottom,
	}
}

// ratingStarRect 返回第 i 颗星的绘制矩形(area 为 ratingArea 的结果)。
func ratingStarRect(area xc.RECT, i int, starSize int32) xc.RECT {
	h := starSize
	if availH := area.Bottom - area.Top; h > availH {
		h = availH
	}
	x := area.Left + ratingFocusPad + int32(i)*(starSize+ratingStarGap)
	y := area.Top + (area.Bottom-area.Top-h)/2
	return xc.RECT{Left: x, Top: y, Right: x + starSize, Bottom: y + h}
}

// ratingValueAt 把客户区 x 映射成评分: 第 i 颗星左半 = i+0.5(半星模式)或 i+1,
// 右半 = i+1; 落在星星区外(左右留白/元素被拉宽后的空区)返回 -1, 不参与预览与点击。
func ratingValueAt(area xc.RECT, x int32, max int, half bool, starSize int32) float64 {
	if max <= 0 || starSize <= 0 {
		return -1
	}
	left := area.Left + ratingFocusPad
	right := left + int32(max)*starSize + int32(max-1)*ratingStarGap
	if x < left || x >= right {
		return -1
	}
	stride := starSize + ratingStarGap
	rel := x - left
	i := int(rel / stride)
	if i >= max {
		return float64(max)
	}
	off := rel - int32(i)*stride
	if off > starSize {
		off = starSize // 星与星之间的空隙算作前一颗星的右端
	}
	if half && off*2 < starSize {
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

	// 只铺星星区(+ 文案区): 元素常被 Fill 拉宽, 整块铺底会在非 Card 容器上留一条色带。
	area := ratingArea(p.Rc, r.max, r.starSize)
	if area.Right <= area.Left || area.Bottom <= area.Top {
		return 0
	}
	p.FillRound(&area, t.Card, 0)
	var lr xc.RECT
	hasLabel := r.labelW > 0
	if hasLabel {
		lr = ratingLabelRect(area, r.labelW)
		if lr.Right > p.Rc.Right {
			lr.Right = p.Rc.Right
		}
		hasLabel = lr.Right > lr.Left
		if hasLabel {
			// 透明元素重绘不擦底, 上一帧的文案会留残影。
			p.FillRound(&lr, t.Card, 0)
		}
	}

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
	hoverIdx := r.hoverIndex()
	if disabled {
		hoverIdx = -1
	}

	for i := 0; i < r.max; i++ {
		box := ratingStarRect(area, i, r.starSize)
		// 逐项过渡走运行时, 不在组件里起定时器/缓存过渡值; 缩放不影响命中测试矩形。
		hv := xui.HoverItem(r.Handle, int32(i), !disabled && !r.readonly && i == hoverIdx)
		sc := 1 + (ratingHoverScale-1)*float32(hv)
		full := display >= float64(i+1)
		half := r.half && !full && display >= float64(i)+0.5
		edge := outline
		if full || half {
			edge = fill
		}
		p.Icon(r.outlineSVG, &box, edge.ScaleAlpha(alpha), 2, sc)
		switch {
		case full:
			p.Icon(r.fullSVG, &box, fill.ScaleAlpha(alpha), 2, sc)
		case half:
			p.Icon(r.halfSVG, &box, fill.ScaleAlpha(alpha), 2, sc)
		}
	}

	if hasLabel {
		if label := r.labelFor(display); label != "" {
			p.Draw.SetFont(xui.FontHandle(t.FontDefault))
			p.Text(label, &lr, t.MutedForeground.ScaleAlpha(alpha),
				xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
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
