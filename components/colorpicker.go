package ui

import (
	"math"
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// ColorPicker 颜色选择器: 触发区(色块 + #rrggbb + chevron) + 弹层(SV 方块 / 色相条 / 预设色 / Hex 输入)。
//
// 交互对齐 shadcn 生态里的 color-picker 写法, 但底层是本库自己的自绘元素:
//   - SV 方块: 横轴饱和度、纵轴亮度, 按下即取色并捕获鼠标, 可以拖出元素继续调;
//   - 色相条: 0~360° 六段渐变, 点/拖都改色相;
//   - 预设色: 两行圆点, 点一下跳到该颜色;
//   - Hex 输入: 输入满 6 位才应用, 免得中途每敲一个字符都闪一次。
//
// 颜色一律以 "#rrggbb" 字符串进出(与 xui.Hex 同格式); 触发区与弹层都从 xui.Current() 实时取令牌,
// 切主题时跟着变, 不缓存任何主题色。
const (
	colorPickerWidth   int32 = 232 // 触发区与弹层同宽
	colorPickerPad     int32 = 12
	colorPickerGap     int32 = 10
	colorPickerSVH     int32 = 132
	colorPickerHueH    int32 = 14 // 色相条本身的粗细
	colorPickerHueBox  int32 = 22 // 色相条元素高 = 条 + 上下各 4px 给圆形滑块
	colorPickerThumbR  int32 = 7  // SV 指示圈半径
	colorPickerPresetD int32 = 20
	colorPickerPresetG int32 = 8
	colorPickerPresetN int32 = 5 // 每行预设色个数
	// colorPickerPresetPad 是预设色行上下留白: 选中环画在圆点外 4px 处,
	// 不留这圈空白, 环的上下就被元素矩形裁掉(只剩左右两段弧)。
	colorPickerPresetPad int32 = 5
	colorPickerHexH    int32 = 32
)

// colorPickerDefaultPresets 是内置预设色: 黑/白 + 六种主色 + 两种辅助色(与 shadcn 常见色板一致)。
var colorPickerDefaultPresets = []string{
	"#000000", "#ffffff", "#ef4444", "#f97316", "#eab308",
	"#22c55e", "#06b6d4", "#3b82f6", "#8b5cf6", "#ec4899",
}

// ColorPickerOption 颜色选择器选项。
type ColorPickerOption func(c *ColorPicker)

// WithColorPickerValue 设置初始颜色("#rrggbb" 或 "#rgb")。
func WithColorPickerValue(hex string) ColorPickerOption {
	return func(c *ColorPicker) { c.value = normalizeHex(hex) }
}

// WithColorPickerOnChange 颜色变化回调(拖动时每次都回调, 松手不额外回调)。
func WithColorPickerOnChange(fn func(hex string)) ColorPickerOption {
	return func(c *ColorPicker) { c.onChange = fn }
}

// WithColorPickerWidth 触发区宽度(默认 232, 弹层同宽)。
func WithColorPickerWidth(w int32) ColorPickerOption {
	return func(c *ColorPicker) {
		if w > 0 {
			c.width = w
		}
	}
}

// WithColorPickerPlaceholder 未选颜色时的占位文本。
func WithColorPickerPlaceholder(s string) ColorPickerOption {
	return func(c *ColorPicker) { c.placeholder = s }
}

// WithColorPickerPresets 替换预设色列表(默认 10 个, 按每行 5 个排)。
func WithColorPickerPresets(list []string) ColorPickerOption {
	return func(c *ColorPicker) {
		ps := make([]string, 0, len(list))
		for _, s := range list {
			if v := normalizeHex(s); v != "" {
				ps = append(ps, v)
			}
		}
		if len(ps) > 0 {
			c.presets = ps
		}
	}
}

// ColorPicker 颜色选择器。
type ColorPicker struct {
	*widget.Button

	panel *popoverPanel
	built bool

	value       string
	lastSynced  string // 上次由 value 反推 HSV 时的值, 避免重复 Open 把拖动得到的精度冲掉
	h, s, v     float64 // 色相 0~360, 饱和度/亮度 0~1
	width       int32
	placeholder string
	presets     []string
	onChange    func(string)
	syncing     bool // 回写 Hex 输入框时置位, 避免 Edit_Changed 再把半成品解析成颜色

	svEle     *widget.Element
	hueEle    *widget.Element
	presetEle *widget.Element
	hexInput  *Input

	dragSV  bool
	dragHue bool
}

// NewColorPicker 创建颜色选择器。
//
//	cp := ui.NewColorPicker(0, 0, 0, 0, parent.Handle,
//		ui.WithColorPickerValue("#3b82f6"),
//		ui.WithColorPickerOnChange(func(hex string) { ... }))
func NewColorPicker(x, y, cx, cy int32, hParent int, opts ...ColorPickerOption) *ColorPicker {
	c := &ColorPicker{
		width:   colorPickerWidth,
		presets: append([]string(nil), colorPickerDefaultPresets...),
	}
	for _, o := range opts {
		o(c)
	}
	if c.placeholder == "" {
		c.placeholder = "选择颜色"
	}
	if cy <= 0 {
		cy = 32
	}
	btn := widget.NewButton(x, y, cx, cy, "", hParent)
	if btn == nil {
		return nil
	}
	c.Button = btn
	if cx <= 0 {
		btn.SetWidth(c.width)
	}
	btn.EnableBkTransparent(true)
	btn.SetTextColor(0) // 文字自绘
	btn.AddEvent_Paint(c.onPaintTrigger)
	// 用 BnClick 而不是 LButtonUp: 触发区是原生 Button, 点击语义由它自己判定(按下又抬在同一
	// 元素上); 键盘 Enter/Space 也会走这里。LButtonUp 在 Button 上收不到 —— 按钮内部已经把
	// 按下/抬起消化成 BnClick, 表现为"点了没反应"。
	btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if !c.IsEnable() {
			return 0
		}
		xui.NotePointerPress()
		// 弹层开着时点触发区: 弹层先因失焦关闭(记进了 lastClose), 这里不再重开(同 Select)
		if c.panel != nil && (c.panel.IsOpen() || c.panel.RecentlyClosed()) {
			return 0
		}
		c.Open()
		return 0
	})
	btn.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if c.panel != nil {
			c.panel.Destroy()
		}
		c.panel, c.svEle, c.hueEle, c.presetEle, c.hexInput = nil, nil, nil, nil, nil
		return 0
	})
	c.syncFromValue()
	xui.AttachWindow(btn.Handle)
	return c
}

// ===== 值 =====

// Value 返回当前颜色 "#rrggbb"; 从未选过时为空串。
func (c *ColorPicker) Value() string {
	if c == nil {
		return ""
	}
	return c.value
}

// SetValue 设置颜色(不触发 OnChange; 传空串表示清空)。
func (c *ColorPicker) SetValue(hex string) *ColorPicker {
	if c == nil {
		return c
	}
	v := normalizeHex(hex)
	if v == c.value {
		return c
	}
	c.value = v
	c.syncFromValue()
	c.redrawAll()
	return c
}

// OnChange 追加颜色变化回调。
func (c *ColorPicker) OnChange(fn func(hex string)) *ColorPicker {
	if c != nil && fn != nil {
		c.onChange = fn
	}
	return c
}

// Open 打开弹层。
func (c *ColorPicker) Open() *ColorPicker {
	if c == nil || !c.IsEnable() || (c.panel != nil && c.panel.IsOpen()) {
		return c
	}
	if !c.built {
		c.build()
	}
	if c.panel == nil {
		return c
	}
	// 首次打开(或清空后)拿主色当起点, 让面板一出来就有个可见的当前位置
	c.syncFromValue()
	c.syncHexInput()
	c.redrawAll()
	x, y := c.popupPos()
	c.panel.ShowAt(c.Handle, x, y)
	return c
}

// Close 关闭弹层。
func (c *ColorPicker) Close() *ColorPicker {
	if c != nil && c.panel != nil {
		c.panel.Hide()
	}
	return c
}

// Toggle 开关弹层。
func (c *ColorPicker) Toggle() *ColorPicker {
	if c.IsOpen() {
		return c.Close()
	}
	return c.Open()
}

// IsOpen 返回弹层是否打开。
func (c *ColorPicker) IsOpen() bool { return c != nil && c.panel != nil && c.panel.IsOpen() }

// normalizeHex 把用户给的颜色规整成 "#rrggbb"; 不认识返回空串。
func normalizeHex(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	c, err := xui.HexE(s)
	if err != nil {
		return ""
	}
	return c.String()
}

// syncFromValue 由 value 反推 HSV; 空值时用主题主色作为面板起点(不改 value)。
//
// 只在 value 真的变过时才反推: hex 只有 8 位精度, 反复反推会把拖动时得到的 h/s/v
// 磨成"四舍五入后的值" —— 拖到纯黑(亮度 0)时色相信息还会整个丢掉, 再打开面板色相条
// 会跳回红色。setHSV 自己会维护 lastSynced, 所以这里命中就直接返回。
func (c *ColorPicker) syncFromValue() {
	if c.value != "" && c.value == c.lastSynced {
		return
	}
	base := c.value
	if base == "" {
		base = xui.Current().Primary.String()
	}
	c.h, c.s, c.v = colorToHSV(xui.Hex(base))
	c.lastSynced = c.value
}

// ===== 弹层 =====

// popupSize 弹层卡片尺寸(含内边距, 不含阴影带)。
func (c *ColorPicker) popupSize() (int32, int32) {
	rows := int32((len(c.presets) + int(colorPickerPresetN) - 1) / int(colorPickerPresetN))
	presetH := presetBlockHeight(rows)
	h := colorPickerPad*2 + colorPickerSVH + colorPickerGap + colorPickerHueBox +
		colorPickerGap + presetH + colorPickerGap + colorPickerHexH
	return c.width, h
}

// presetBlockHeight 是预设色区块的高度(含选中环需要的上下留白)。
func presetBlockHeight(rows int32) int32 {
	if rows <= 0 {
		return 0
	}
	return rows*colorPickerPresetD + (rows-1)*colorPickerPresetG + colorPickerPresetPad*2
}

// popupPos 弹层左上角的屏幕逻辑坐标: 与触发区起始边对齐、优先下方, 放不下自动翻到上方。
func (c *ColorPicker) popupPos() (int32, int32) {
	cw, ch := c.popupSize()
	return popupPosition(c.Handle, cw, ch, PopoverSideBottom, PopoverAlignStart, 6)
}

// build 构建弹层(尺寸/预设变化后需要重建: 目前只有 Destroy 会清空, 内容固定)。
func (c *ColorPicker) build() {
	c.built = true
	if c.panel != nil {
		c.panel.Destroy()
		c.panel, c.svEle, c.hueEle, c.presetEle, c.hexInput = nil, nil, nil, nil, nil
	}
	cw, ch := c.popupSize()
	// layoutBody=true: 走布局容器, 内容区四边内边距由它给(见 8.129, 不能用子项外边距)
	panel := newPopoverPanelEx(cw, ch, c.Handle, true, true)
	if panel == nil {
		return
	}
	c.panel = panel
	panel.OnEsc = func() { c.SetFocus() }
	panel.OnHide = func() {
		c.dragSV, c.dragHue = false, false
	}
	content := panel.LayoutContent(colorPickerPad, colorPickerGap)
	if content == nil {
		return
	}

	// SV 方块
	sv := widget.NewElement(0, 0, 0, colorPickerSVH, content.Handle)
	if sv == nil {
		return
	}
	sv.EnableBkTransparent(true)
	sv.EnableDrawBorder(false)
	sv.EnableFocus(false)
	xui.ItemFill(sv)
	xui.ItemHeight(sv, colorPickerSVH)
	sv.AddEvent_Paint(c.onPaintSV)
	sv.AddEvent_LButtonDown(c.onSVDown)
	sv.AddEvent_MouseMove(c.onSVMove)
	sv.AddEvent_LButtonUp(c.onSVUp)
	sv.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		if !c.dragSV {
			sv.Redraw(true)
		}
		return 0
	})
	c.svEle = sv

	// 色相条: 元素比条本身高, 圆形滑块才不会被上下裁掉
	hue := widget.NewElement(0, 0, 0, colorPickerHueBox, content.Handle)
	if hue == nil {
		return
	}
	hue.EnableBkTransparent(true)
	hue.EnableDrawBorder(false)
	hue.EnableFocus(false)
	xui.ItemFill(hue)
	xui.ItemHeight(hue, colorPickerHueBox)
	hue.AddEvent_Paint(c.onPaintHue)
	hue.AddEvent_LButtonDown(c.onHueDown)
	hue.AddEvent_MouseMove(c.onHueMove)
	hue.AddEvent_LButtonUp(c.onHueUp)
	c.hueEle = hue

	// 预设色
	rows := int32((len(c.presets) + int(colorPickerPresetN) - 1) / int(colorPickerPresetN))
	presetH := presetBlockHeight(rows)
	ps := widget.NewElement(0, 0, 0, presetH, content.Handle)
	if ps == nil {
		return
	}
	ps.EnableBkTransparent(true)
	ps.EnableDrawBorder(false)
	ps.EnableFocus(false)
	xui.ItemFill(ps)
	xui.ItemHeight(ps, presetH)
	ps.AddEvent_Paint(c.onPaintPresets)
	ps.AddEvent_LButtonUp(c.onPresetUp)
	c.presetEle = ps

	// Hex 输入
	in := NewInput(0, 0, 0, 0, content.Handle, WithPlaceholder("rrggbb"), WithPrefixText("#"))
	if in == nil {
		return
	}
	xui.ItemFill(in)
	xui.ItemHeight(in, colorPickerHexH)
	in.OnChange(func(text string) {
		if c.syncing {
			return
		}
		v := normalizeHex(text)
		if v == "" || v == c.value {
			return
		}
		c.value = v
		c.syncFromValue()
		c.redrawAll()
		c.emitChange()
	})
	c.hexInput = in
	c.syncHexInput()
}

// syncHexInput 把当前颜色回写给输入框(带 syncing 保护, 不触发新一轮 OnChange)。
func (c *ColorPicker) syncHexInput() {
	if c.hexInput == nil {
		return
	}
	c.syncing = true
	c.hexInput.SetValue(strings.TrimPrefix(c.value, "#"))
	c.syncing = false
}

// redrawAll 重绘触发区与面板里的三个自绘区(面板窗口是独立的, 要单独重绘)。
func (c *ColorPicker) redrawAll() {
	if c.Button != nil {
		c.Redraw(false)
	}
	if !c.panelAlive() {
		return
	}
	if c.svEle != nil {
		c.svEle.Redraw(false)
	}
	if c.hueEle != nil {
		c.hueEle.Redraw(false)
	}
	if c.presetEle != nil {
		c.presetEle.Redraw(false)
	}
	if c.hexInput != nil && c.panel != nil {
		// 面板窗口重绘会把 Edit 的底色/文字色一起带走; 输入框自己也会 Redraw
		c.hexInput.Redraw(false)
	}
}

func (c *ColorPicker) panelAlive() bool {
	return c.panel != nil && c.panel.IsOpen()
}

// emitChange 触发颜色变化回调。
func (c *ColorPicker) emitChange() {
	if c.onChange != nil {
		c.onChange(c.value)
	}
}

// setHSV 应用新的 HSV(夹取到合法区间), 更新 value/输入框并回调。
func (c *ColorPicker) setHSV(h, s, v float64, fire bool) {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c.h, c.s, c.v = h, clamp01(s), clamp01(v)
	c.value = hsvToColor(c.h, c.s, c.v).String()
	c.lastSynced = c.value
	c.syncHexInput()
	c.redrawAll()
	if fire {
		c.emitChange()
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ===== 拖动 =====

func (c *ColorPicker) onSVDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if c.svEle == nil {
		return 0
	}
	*pbHandled = true
	xui.NotePointerPress()
	c.dragSV = true
	xc.XEle_SetCapture(hEle, true) // 拖出方块也能继续调(同 Slider)
	c.applySVPoint(pPt)
	return 0
}

func (c *ColorPicker) onSVMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if !c.dragSV {
		return 0
	}
	*pbHandled = true
	c.applySVPoint(pPt)
	return 0
}

func (c *ColorPicker) onSVUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if c.dragSV {
		c.dragSV = false
		xc.XEle_SetCapture(hEle, false)
	}
	return 0
}

// applySVPoint 把元素内坐标换算成(饱和度, 亮度)并应用。
func (c *ColorPicker) applySVPoint(pPt *xc.POINT) {
	w, h := c.svEle.GetWidth(), c.svEle.GetHeight()
	if w <= 0 || h <= 0 {
		return
	}
	s := float64(pPt.X) / float64(w)
	v := 1 - float64(pPt.Y)/float64(h)
	c.setHSV(c.h, s, v, true)
}

func (c *ColorPicker) onHueDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if c.hueEle == nil {
		return 0
	}
	*pbHandled = true
	xui.NotePointerPress()
	c.dragHue = true
	xc.XEle_SetCapture(hEle, true)
	c.applyHuePoint(pPt)
	return 0
}

func (c *ColorPicker) onHueMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if !c.dragHue {
		return 0
	}
	*pbHandled = true
	c.applyHuePoint(pPt)
	return 0
}

func (c *ColorPicker) onHueUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if c.dragHue {
		c.dragHue = false
		xc.XEle_SetCapture(hEle, false)
	}
	return 0
}

func (c *ColorPicker) applyHuePoint(pPt *xc.POINT) {
	w := c.hueEle.GetWidth()
	if w <= 0 {
		return
	}
	// 先把位置夹进条内: 指针拖出右边界时 x>w 会算出 h>360, setHSV 里的 Mod 会把它折回
	// 0 附近 —— 表现为"拖到最右端, 滑块突然从最左边继续拖"(左边同理折到最右)。
	x := clampI32(pPt.X, 0, w-1)
	c.setHSV(float64(x)/float64(w)*360, c.s, c.v, true)
}

// onPresetUp 点预设色: 命中哪个圆点就跳到哪个颜色。
func (c *ColorPicker) onPresetUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	idx := c.presetAt(pPt.X, pPt.Y)
	if idx < 0 {
		return 0
	}
	*pbHandled = true
	xui.NotePointerPress()
	v := c.presets[idx]
	c.value = v
	c.lastSynced = ""
	c.syncFromValue()
	c.syncHexInput()
	c.redrawAll()
	c.emitChange()
	return 0
}

// presetAt 命中测试: 每行 colorPickerPresetN 个, 单元格内居中的圆点才算命中。
func (c *ColorPicker) presetAt(x, y int32) int {
	if c.presetEle == nil || len(c.presets) == 0 {
		return -1
	}
	cellW := float64(c.presetEle.GetWidth()) / float64(colorPickerPresetN)
	if cellW <= 0 {
		return -1
	}
	col := int32(float64(x) / cellW)
	row := (y - colorPickerPresetPad) / (colorPickerPresetD + colorPickerPresetG)
	if col < 0 || col >= colorPickerPresetN || row < 0 {
		return -1
	}
	idx := int(row*colorPickerPresetN + col)
	if idx < 0 || idx >= len(c.presets) {
		return -1
	}
	// 圆点在单元格内居中: 只有落在圆内才算命中, 免得整块空白都能点
	// (圆心算法必须和绘制那边一致, 否则圆点看着在那儿、点上去却没反应)
	cx := int32(float64(col)*cellW + cellW/2 + 0.5)
	cy := colorPickerPresetPad + row*(colorPickerPresetD+colorPickerPresetG) + colorPickerPresetD/2
	dx, dy := x-cx, y-cy
	if dx*dx+dy*dy > (colorPickerPresetD/2+2)*(colorPickerPresetD/2+2) {
		return -1
	}
	return idx
}

// ===== 绘制 =====

// onPaintTrigger 触发区: 圆角底 + 16×16 色块 + hex 文本 + chevron。
func (c *ColorPicker) onPaintTrigger(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)

	bg, border := t.Card, t.Input
	if st.Disabled {
		bg, border = t.Input.WithAlpha(128), xui.None
	}
	corners := xui.UniformCorners(t.RadiusLg())
	switch {
	case st.Focused && !st.Disabled && xui.FocusedByKeyboard():
		p.FocusFrameCorners(t.Ring, bg, corners)
	default:
		p.FillRoundCorners(&p.Rc, bg, corners)
		if t.IsDark() && !st.Disabled {
			a := 0.3 * xui.Hover(hEle, st.Hover && !st.Pressed)
			p.FillRoundCorners(&p.Rc, t.Input.ScaleAlpha(0.3+a), corners)
		}
		p.StrokeRoundCorners(&p.Rc, border, corners, 1)
	}

	x := p.Rc.Left + 10 // pl-2.5
	// 色块: 未选过用 muted 底, 免得占位状态下画出一块黑
	sw := xc.RECT{
		Left:   x,
		Top:    p.Rc.Top + (p.H()-16)/2,
		Right:  x + 16,
		Bottom: p.Rc.Top + (p.H()-16)/2 + 16,
	}
	swatch := t.Muted
	if c.value != "" {
		swatch = xui.Hex(c.value)
	}
	p.FillRound(&sw, swatch, 4)
	p.StrokeRound(&sw, t.Border, 4, 1)
	x = sw.Right + 8

	fore := t.Foreground
	text := c.value
	if c.value == "" {
		fore, text = t.MutedForeground, c.placeholder
	}
	f := xui.FontHandle(t.FontDefault)
	if f != 0 {
		p.Draw.SetFont(f)
	}
	dy := textVDy(text)
	rc := xc.RECTF{
		Left:   float32(x),
		Top:    float32(p.Rc.Top) - dy,
		Right:  float32(p.Rc.Right - 30),
		Bottom: float32(p.Rc.Bottom) - dy,
	}
	p.TextF(text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)

	icon := xc.RECT{
		Left:   p.Rc.Right - 26,
		Top:    p.Rc.Top + (p.H()-16)/2,
		Right:  p.Rc.Right - 10,
		Bottom: p.Rc.Top + (p.H()-16)/2 + 16,
	}
	p.Icon(icons.IconChevronDown, &icon, t.MutedForeground, 2)
	return 0
}

// onPaintSV 画 SV 方块: 底色 = 纯色相, 再叠"左白→右透明"与"上透明→下黑"两层渐变。
//
// 用 alpha 渐变而不是"白→纯色""纯色→黑"两组实色: 后者在中段会出现灰调偏移,
// 与标准 HSV 方块的取色值对不上。
func (c *ColorPicker) onPaintSV(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	base := hsvToColor(c.h, 1, 1)
	rc := p.Rc
	p.FillRound(&rc, base, 0)
	p.Draw.GradientFill2(&rc, uint32(xui.RGBA(255, 255, 255, 255)),
		uint32(xui.RGBA(255, 255, 255, 0)), xcc.GRADIENT_FILL_RECT_H)
	p.Draw.GradientFill2(&rc, uint32(xui.RGBA(0, 0, 0, 0)),
		uint32(xui.RGBA(0, 0, 0, 255)), xcc.GRADIENT_FILL_RECT_V)

	t := xui.Current()
	p.StrokeRound(&rc, t.Border, 0, 1) // 1px 描边, 和输入框同样的分隔感

	// 当前位置: 白圈 + 内侧暗圈, 深浅背景上都看得见
	cx := rc.Left + int32(c.s*float64(rc.Right-rc.Left))
	cy := rc.Top + int32((1-c.v)*float64(rc.Bottom-rc.Top))
	cx = clampI32(cx, rc.Left+colorPickerThumbR, rc.Right-colorPickerThumbR)
	cy = clampI32(cy, rc.Top+colorPickerThumbR, rc.Bottom-colorPickerThumbR)
	p.Draw.SetLineWidth(2)
	p.Draw.SetBrushColor(uint32(xui.RGBA(0, 0, 0, 90)))
	p.Draw.DrawEllipse(&xc.RECT{Left: cx - colorPickerThumbR - 1, Top: cy - colorPickerThumbR - 1,
		Right: cx + colorPickerThumbR + 1, Bottom: cy + colorPickerThumbR + 1})
	p.Draw.SetBrushColor(uint32(xui.RGBA(255, 255, 255, 255)))
	p.Draw.DrawEllipse(&xc.RECT{Left: cx - colorPickerThumbR, Top: cy - colorPickerThumbR,
		Right: cx + colorPickerThumbR, Bottom: cy + colorPickerThumbR})
	return 0
}

// onPaintHue 画色相条: 六段 60° 渐变 + 当前色相位置的圆形滑块。
func (c *ColorPicker) onPaintHue(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	w := p.W()
	if w <= 0 {
		return 0
	}
	// 条比元素矮(元素上下多留的空给圆形滑块), 条在元素内垂直居中
	barRc := xc.RECT{
		Left:   p.Rc.Left,
		Top:    p.Rc.Top + (p.H()-colorPickerHueH)/2,
		Right:  p.Rc.Right,
		Bottom: p.Rc.Top + (p.H()+colorPickerHueH)/2,
	}
	const segs = 6
	for i := int32(0); i < segs; i++ {
		seg := xc.RECT{
			Left:   barRc.Left + w*i/segs,
			Top:    barRc.Top,
			Right:  barRc.Left + w*(i+1)/segs + 1, // +1 掩盖相邻段的抗锯齿缝
			Bottom: barRc.Bottom,
		}
		c1 := hsvToColor(float64(i)*360/segs, 1, 1)
		c2 := hsvToColor(float64(i+1)*360/segs, 1, 1)
		p.Draw.GradientFill2(&seg, uint32(c1), uint32(c2), xcc.GRADIENT_FILL_RECT_H)
	}
	t := xui.Current()
	p.StrokeRound(&barRc, t.Border, 0, 1)

	// 滑块: 白圆 + border-ring 1px 边 —— 和 Slider 的 thumb 同一套观感(白底是 shadcn 的
	// 字面量, 两个主题都一样), 圆心对齐当前色相。
	//
	// 两层实心圆而不是"胶囊 + 描边": 描边落在半像素上会让圆边毛糙, 胶囊形态还会出现
	// 左右黑边一会儿 1px 一会儿 2px 的假象。
	const thumbR, edgeR = int32(8), int32(7)
	cx := barRc.Left + int32(c.h/360*float64(w)+0.5)
	cx = clampI32(cx, barRc.Left+thumbR, barRc.Right-thumbR)
	cy := (barRc.Top + barRc.Bottom) / 2
	p.Draw.SetBrushColor(uint32(t.Ring))
	p.Draw.FillEllipse(&xc.RECT{Left: cx - thumbR, Top: cy - thumbR, Right: cx + thumbR, Bottom: cy + thumbR})
	p.Draw.SetBrushColor(uint32(xui.RGBA(255, 255, 255, 255)))
	p.Draw.FillEllipse(&xc.RECT{Left: cx - edgeR, Top: cy - edgeR, Right: cx + edgeR, Bottom: cy + edgeR})
	return 0
}

// onPaintPresets 画预设色圆点; 与当前颜色相同的那个加一圈 ring。
func (c *ColorPicker) onPaintPresets(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	if len(c.presets) == 0 {
		return 0
	}
	t := xui.Current()
	// 单元格宽度按浮点算再四舍五入: 宽度除不尽 5 时(208/5), 用整数截断会让每列的圆心
	// 累计左移零点几像素, 放大看就是"圆点没在环中心"。
	cellWF := float64(p.W()) / float64(colorPickerPresetN)
	if cellWF <= 0 {
		return 0
	}
	r := colorPickerPresetD / 2
	for i, hex := range c.presets {
		col := int32(i) % colorPickerPresetN
		row := int32(i) / colorPickerPresetN
		cx := p.Rc.Left + int32(float64(col)*cellWF+cellWF/2+0.5)
		cy := p.Rc.Top + colorPickerPresetPad + row*(colorPickerPresetD+colorPickerPresetG) + r
		if hex == c.value {
			// 选中环: 外圆 ring 色 + 内圆面板底色挖洞, 与圆点之间留 1px 缝。
			//
			// 缝不能去掉: 环色和选中的色块经常同色(比如选中蓝、主色也是蓝), 无缝时环和圆
			// 粘成一坨, 只剩圆点那圈 border 淡边, 看着像"双圈甜甜圈"。
			rr := r + 3
			p.Draw.SetBrushColor(uint32(t.Ring))
			p.Draw.FillEllipse(&xc.RECT{Left: cx - rr, Top: cy - rr, Right: cx + rr, Bottom: cy + rr})
			ri := r + 1
			p.Draw.SetBrushColor(uint32(t.Popover))
			p.Draw.FillEllipse(&xc.RECT{Left: cx - ri, Top: cy - ri, Right: cx + ri, Bottom: cy + ri})
		}
		// 圆点: 外圆 border 色 + 内圆纯色 = 1px 描边。
		// 白色预设色在浅色面板上没有描边就"消失", 但 1px 椭圆描边(StrokeRound)会毛糙,
		// 所以同样用两层实心圆来造边。
		p.Draw.SetBrushColor(uint32(t.Border))
		p.Draw.FillEllipse(&xc.RECT{Left: cx - r, Top: cy - r, Right: cx + r, Bottom: cy + r})
		rin := r - 1
		p.Draw.SetBrushColor(uint32(xui.Hex(hex)))
		p.Draw.FillEllipse(&xc.RECT{Left: cx - rin, Top: cy - rin, Right: cx + rin, Bottom: cy + rin})
	}
	return 0
}

// ===== HSV 工具 =====

// colorToHSV 把颜色转成(色相 0~360, 饱和度 0~1, 亮度 0~1)。
func colorToHSV(c xui.Color) (h, s, v float64) {
	r, g, b, _ := c.RGBA()
	rf, gf, bf := float64(r)/255, float64(g)/255, float64(b)/255
	mx := math.Max(rf, math.Max(gf, bf))
	mn := math.Min(rf, math.Min(gf, bf))
	d := mx - mn
	v = mx
	if mx > 0 {
		s = d / mx
	}
	if d == 0 {
		return 0, s, v
	}
	switch {
	case mx == rf:
		h = math.Mod((gf-bf)/d, 6)
	case mx == gf:
		h = (bf-rf)/d + 2
	default:
		h = (rf-gf)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, s, v
}

// hsvToColor 把 HSV 转成颜色。
func hsvToColor(h, s, v float64) xui.Color {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	ch := v * s
	x := ch * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - ch
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = ch, x, 0
	case h < 120:
		r, g, b = x, ch, 0
	case h < 180:
		r, g, b = 0, ch, x
	case h < 240:
		r, g, b = 0, x, ch
	case h < 300:
		r, g, b = x, 0, ch
	default:
		r, g, b = ch, 0, x
	}
	return xui.RGBA(byte8(r+m), byte8(g+m), byte8(b+m), 255)
}

// byte8 把 0~1 的分量转成 0~255(四舍五入 + 夹取, 防止浮点误差溢出成 0)。
func byte8(f float64) byte {
	n := int(f*255 + 0.5)
	if n < 0 {
		n = 0
	}
	if n > 255 {
		n = 255
	}
	return byte(n)
}

func clampI32(v, lo, hi int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
