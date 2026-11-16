package ui

import (
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// ToggleVariant 切换按钮变体, 对应 shadcn toggle.tsx 的 variant。
type ToggleVariant int

const (
	ToggleVariantDefault ToggleVariant = iota // 透明底, 悬停/选中 bg-muted
	ToggleVariantOutline                      // 1px border-input, 悬停/选中 bg-muted
)

// ToggleSize 切换按钮尺寸档, 对应 base-nova toggle.tsx 的 size(h-8 / h-7 / h-9)。
type ToggleSize int

const (
	ToggleSizeDefault ToggleSize = iota // h-8(32)
	ToggleSizeSm                        // h-7(28)
	ToggleSizeLg                        // h-9(36)
)

// toggleSpec 是尺寸规格(逻辑像素): 高度、左右内边距、图标侧内边距、最小宽度、
// 字号、图标尺寸与图标间距。
type toggleSpec struct {
	height   int32
	padX     int32 // px-2.5
	padIcon  int32 // has-data-[icon=inline-start]:pl-2 / sm:pl-1.5
	minWidth int32
	font     int32
	icon     int32 // size-4 / sm:size-3.5, 已含 2px 描边补偿(同 Button)
	gap      int32 // gap-1
}

var toggleSpecs = map[ToggleSize]toggleSpec{
	ToggleSizeDefault: {height: 32, padX: 10, padIcon: 8, minWidth: 32, font: 14, icon: 18, gap: 4},
	ToggleSizeSm:      {height: 28, padX: 10, padIcon: 6, minWidth: 28, font: 13, icon: 16, gap: 4},
	ToggleSizeLg:      {height: 36, padX: 10, padIcon: 8, minWidth: 36, font: 14, icon: 18, gap: 4},
}

const (
	// ToggleGroup 的 spacing=0(connected)时条目的 px-2 / icon 侧 px-1.5 覆盖尺寸档内距。
	toggleGroupConnectedPadX    int32 = 8
	toggleGroupConnectedPadIcon int32 = 6
)

// Toggle 双状态按钮: 载体为 Button_Type_Check 原生按钮(点击/空格/回车切换, 带键盘焦点),
// 壳体与内容全部自绘。
//
// 与 base-nova toggle.tsx 对齐:
//   - 未选中: 透明底 + 前景色; 悬停 bg-muted + text-foreground;
//   - 选中(data-[state=on] / aria-pressed): bg-muted + text-foreground;
//   - outline: border-input, 悬停/选中同样只换 bg-muted(shadow-xs 不画, 见映射表);
//   - focus-visible: border-ring + ring-3 ring-ring/50; aria-invalid: 红边框 + 红环(常驻);
//   - disabled: opacity-50。
type Toggle struct {
	*widget.Button
	variant   ToggleVariant
	size      ToggleSize
	icon      string
	invalid   bool
	autoWidth bool
	value     string // 所属 ToggleGroup 的值(AddItem 写入; 独立使用时为空)
	changes   []func(pressed bool)

	// 禁用分三层: 调用方(SetDisabled) / 组级(ToggleGroup.SetDisabled) / Field 级联。
	// 分开记是为了"整组启用后恢复各条目自己的禁用态"(同 Accordion 的整组禁用)。
	userDisabled  bool
	groupDisabled bool
	fieldDisabled bool

	// layoutHook 在尺寸相关字段(文字/图标/尺寸档)变化后调用, ToggleGroup 用它
	// 重算纵向布局的统一宽度。
	layoutHook func()
	// guard 是 ToggleGroup 的准入判定: 返回 false 表示本次切换被否决(组禁用 /
	// OnValueChangeVeto), 条目状态已在 guard 里回滚, 用户回调不应再收到这次变化。
	guard func(pressed bool) bool
}

// ToggleOption 切换按钮配置项。
type ToggleOption func(*Toggle)

// WithToggleVariant 设置变体。
func WithToggleVariant(v ToggleVariant) ToggleOption { return func(t *Toggle) { t.variant = v } }

// WithToggleSize 设置尺寸档(默认 32 高)。
func WithToggleSize(s ToggleSize) ToggleOption { return func(t *Toggle) { t.size = s } }

// WithTogglePressed 设置初始选中态。
func WithTogglePressed(v bool) ToggleOption { return func(t *Toggle) { t.SetPressed(v) } }

// WithToggleIcon 让切换按钮在文字前绘制一枚行内图标。
func WithToggleIcon(path string) ToggleOption { return func(t *Toggle) { t.icon = path } }

// WithToggleInvalid 设置校验失败态(对应 shadcn 的 aria-invalid)。
func WithToggleInvalid() ToggleOption { return func(t *Toggle) { t.invalid = true } }

// WithToggleDisabled 设置初始禁用态(控件本身禁用, 不受 Field / 组级影响)。
func WithToggleDisabled() ToggleOption { return func(t *Toggle) { t.userDisabled = true } }

// NewToggle 创建切换按钮。cy 传 0 用尺寸档高度; cx 传 0 按内容量宽
// (不小于尺寸档的 min-width; 位于 ToggleGroup 内时不设最小宽)。
func NewToggle(x, y, cx, cy int32, text string, hParent int, opts ...ToggleOption) *Toggle {
	btn := widget.NewButton(x, y, cx, cy, text, hParent)
	if btn == nil {
		return nil
	}
	t := &Toggle{Button: btn, autoWidth: cx <= 0}
	for _, o := range opts {
		o(t)
	}
	if cy <= 0 {
		t.SetHeight(toggleSpecs[t.size].height)
	}
	if t.autoWidth {
		t.SetWidth(t.naturalWidth(xui.Current()))
	}

	t.SetTypeEx(xcc.Button_Type_Check)
	t.EnableBkTransparent(true) // 灭掉原生外观, 交给 onPaint
	t.SetTextColor(0)           // 文字自绘, 颜色随状态
	t.AddEvent_Paint(t.onPaint)
	xui.AttachWindow(t.Handle)
	t.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	// 原生勾选不会自动失效重绘; 鼠标点击经 XE_BUTTON_CHECK 补重绘并通知监听者。
	t.AddEvent_Button_Check(func(hEle int, bCheck bool, pbHandled *bool) int {
		t.Redraw(false)
		t.notifyChange(bCheck)
		return 0
	})
	// 原生 Check 按钮不处理空格/回车(实测 KeyDown 能收到但状态不变);
	// Radix 的 Toggle 是 <button>, 键盘激活必须自己补。
	t.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		const vkReturn, vkSpace = 0x0D, 0x20
		if (wParam == vkReturn || wParam == vkSpace) && xc.XEle_IsEnable(hEle) {
			t.setPressedNotify(!t.IsCheck())
			*pbHandled = true
		}
		return 0
	})
	t.syncEnabled()
	return t
}

// setPressedNotify 供键盘路径使用: 写状态并触发与鼠标点击相同的回调。
func (t *Toggle) setPressedNotify(v bool) {
	t.SetCheck(v)
	t.Redraw(false)
	t.notifyChange(v)
}

// notifyChange 先过所属组的准入判定, 再触发用户监听。组的准入必须先于用户回调:
// 被否决的切换会在 guard 里回滚, 用户回调不该看到"亮了一下又灭"的中间值。
func (t *Toggle) notifyChange(pressed bool) {
	if t.guard != nil && !t.guard(pressed) {
		return
	}
	for _, fn := range t.changes {
		fn(pressed)
	}
}

// Pressed 返回是否处于选中态。
func (t *Toggle) Pressed() bool { return t.IsCheck() }

// FieldValue 实现 FieldValueProvider: 按下为 "1", 未按下为空串。
func (t *Toggle) FieldValue() string {
	if t == nil || !t.Pressed() {
		return ""
	}
	return "1"
}

// SetPressed 设置选中态, 不触发 OnChange。
func (t *Toggle) SetPressed(v bool) *Toggle {
	if t.IsCheck() == v {
		return t
	}
	t.SetCheck(v)
	t.Redraw(false)
	return t
}

// OnChange 注册选中态变化回调(鼠标点击或键盘切换时触发)。
func (t *Toggle) OnChange(fn func(pressed bool)) *Toggle {
	t.changes = append(t.changes, fn)
	return t
}

// SetText 修改文字; 自适应宽度时重新量宽。
func (t *Toggle) SetText(text string) *Toggle {
	t.Button.SetText(text)
	t.fitWidth()
	return t
}

// SetIcon 设置/替换行内图标(空串移除)。
func (t *Toggle) SetIcon(path string) *Toggle {
	t.icon = path
	t.fitWidth()
	return t
}

// SetSize 运行时切换尺寸档(高度、内距、字号与图标一起换), 自适应宽度会重新量宽。
func (t *Toggle) SetSize(s ToggleSize) *Toggle {
	if t.size == s {
		return t
	}
	t.size = s
	t.SetHeight(toggleSpecs[s].height)
	t.fitWidth()
	return t
}

// SetVariant 运行时切换变体(默认 / outline)。
func (t *Toggle) SetVariant(v ToggleVariant) *Toggle {
	if t.variant == v {
		return t
	}
	t.variant = v
	t.Redraw(false)
	return t
}

// fitWidth 按当前文字/图标重算自适应宽度并通知所属组重排, 最后重绘。
func (t *Toggle) fitWidth() {
	if t.autoWidth {
		t.SetWidth(t.naturalWidth(xui.Current()))
	}
	if t.layoutHook != nil {
		t.layoutHook()
	}
	t.Redraw(false)
}

// SetInvalid 设置/清除校验失败态(见 WithToggleInvalid)。
func (t *Toggle) SetInvalid(v bool) *Toggle {
	if t.invalid == v {
		return t
	}
	t.invalid = v
	t.Redraw(false)
	return t
}

// Invalid 返回当前是否处于校验失败态。
func (t *Toggle) Invalid() bool { return t.invalid }

// SetFieldState 实现 FieldControl: 由 Field 级联校验失败/禁用态。
func (t *Toggle) SetFieldState(invalid, disabled bool) {
	t.SetInvalid(invalid)
	if t.fieldDisabled != disabled {
		t.fieldDisabled = disabled
		t.syncEnabled()
	}
}

// FieldDisabled 实现 FieldControl。
func (t *Toggle) FieldDisabled() bool { return t == nil || !t.IsEnable() }

// SetDisabled 设置/清除控件自身的禁用态(与组级、Field 级联互不覆盖)。
func (t *Toggle) SetDisabled(v bool) *Toggle {
	if t == nil || t.userDisabled == v {
		return t
	}
	t.userDisabled = v
	t.syncEnabled()
	return t
}

// Disabled 报告当前是否禁用(自身 / 组级 / Field 任一生效)。
func (t *Toggle) Disabled() bool { return t == nil || !t.IsEnable() }

// setGroupDisabled 由 ToggleGroup 调用: 组级禁用与条目自身禁用分开记,
// 整组恢复时不会把"本来就禁用"的条目一起放出来。
func (t *Toggle) setGroupDisabled(v bool) {
	if t == nil || t.groupDisabled == v {
		return
	}
	t.groupDisabled = v
	t.syncEnabled()
}

// syncEnabled 把三层禁用状态合成到元素的可交互性与焦点链上。
func (t *Toggle) syncEnabled() {
	if t == nil || t.Button == nil {
		return
	}
	on := !(t.userDisabled || t.groupDisabled || t.fieldDisabled)
	t.Enable(on)
	t.EnableFocus(on)
	t.EnableSwitchFocus(on)
	t.Redraw(false)
}

// FitWidth 按当前文字/图标重新量宽(仅 cx 传 0 创建的切换按钮)。
func (t *Toggle) FitWidth() *Toggle {
	if !t.autoWidth {
		return t
	}
	t.SetWidth(t.naturalWidth(xui.Current()))
	t.Redraw(false)
	return t
}

// naturalWidth 量出自适应宽度。组内条目按组级排版量宽(spacing=0 时内距收成
// px-2 / icon 侧 px-1.5), 组外按尺寸档的 px-2.5 与 min-width(同 base-nova 的
// toggleVariants 与 ToggleGroupItem 的覆盖规则)。
func (t *Toggle) naturalWidth(th *xui.Theme) int32 {
	sp := toggleSpecs[t.size]
	padL, padR := sp.padX, sp.padX
	minW := sp.minWidth
	place := toggleGroupPlacementOf(t.Handle)
	if place.inGroup {
		minW = 0
		if place.compact {
			padL, padR = toggleGroupConnectedPadX, toggleGroupConnectedPadX
		}
	}
	if t.icon != "" {
		padL = sp.padIcon
		if place.inGroup && place.compact {
			padL = toggleGroupConnectedPadIcon
		}
	}
	w := padL + padR + t.contentWidth(th)
	if w < minW {
		w = minW
	}
	return w
}

// contentWidth 返回图标 + 间距 + 文字的宽度。
func (t *Toggle) contentWidth(th *xui.Theme) int32 {
	sp := toggleSpecs[t.size]
	f := t.fontHandle(th)
	w := xui.TextWidth(t.GetText(), f)
	if t.icon != "" {
		if w > 0 {
			w += sp.gap
		}
		w += sp.icon
	}
	return w
}

// fontHandle 返回尺寸档对应的字体句柄(text-sm / sm:text-[0.8rem])。
func (t *Toggle) fontHandle(th *xui.Theme) int {
	spec := toggleSpecs[t.size]
	return xui.FontHandle(xui.FontSpec{Family: th.FontDefault.Family, Size: spec.font})
}

// onPaint 绘制壳体与内容。坐标全部为逻辑像素(框架统一做 DPI 变换)。
func (t *Toggle) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true // 文字/图标自绘, 才能与壳体一起做焦点环内缩
	th := xui.Current()
	st := xui.StateOf(hEle)
	on := t.IsCheck()
	place := toggleGroupPlacementOf(hEle)
	corners := t.corners(th, place)

	// 先铺宿主底色(整元素): 透明元素重绘不擦底(见 DESIGN 第八节 28),
	// 不铺的话悬停/选中色会在失焦或取消选中后留在画布上。
	p.FillRound(&p.Rc, th.Card, 0)

	hv := xui.Hover(hEle, (st.Hover || st.Pressed) && !st.Disabled)
	bg, border, fore := t.resolve(th, st, on, hv)

	invalid := t.invalid && !st.Disabled
	focusRing := st.Focused && !st.Disabled && xui.FocusedByKeyboard()
	if invalid || focusRing {
		// focus-visible:ring-3 ring-ring/50; aria-invalid 用 destructive/20(dark /40)。
		// 与 Button 同款画法: 环带铺满元素, 内芯内缩 3px 后重新铺底色 + bg + 1px 边框。
		halo, borderColor := th.Ring.WithAlpha(128), th.Ring
		if invalid {
			halo, borderColor = th.Destructive.WithAlpha(51), th.Destructive
			if th.IsDark() {
				halo, borderColor = th.Destructive.WithAlpha(102), th.Destructive
			}
		}
		p.FillRoundCorners(&p.Rc, halo, corners)
		const inset = int32(3)
		inner := xc.RECT{
			Left:   p.Rc.Left + inset,
			Top:    p.Rc.Top + inset,
			Right:  p.Rc.Right - inset,
			Bottom: p.Rc.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			ic := corners.Inset(inset)
			p.FillRoundCorners(&inner, th.Card, ic)
			p.FillRoundCorners(&inner, bg, ic)
			p.StrokeRoundCorners(&inner, borderColor, ic, 1)
		}
	} else {
		p.FillRoundCorners(&p.Rc, bg, corners)
		p.StrokeRoundCorners(&p.Rc, border, corners, 1)
	}

	t.drawContent(p, th, fore)
	return 0
}

// resolve 解析壳体与文字颜色, 逐条对应 base-nova toggleVariants 的类名:
// 悬停与选中都是 bg-muted + text-foreground(旧 radix-nova 的 accent 已不再使用)。
func (t *Toggle) resolve(th *xui.Theme, st xui.State, on bool, hv float64) (bg, border, fore xui.Color) {
	bgHV := th.Muted
	switch t.variant {
	case ToggleVariantOutline:
		border = th.Input
	default:
		bg = xui.None
	}
	fore = th.Foreground
	// data-[state=on] 排在 hover 之后: 选中态始终用 muted, 悬停不再改底色。
	if on {
		bg, bgHV = th.Muted, th.Muted
	}
	bg = xui.Fade(bg, bgHV, hv)

	if st.Disabled {
		// disabled:opacity-50 是整元素半透明: 壳体加 alpha, 文字按宿主底色预混。
		bg = bg.ScaleAlpha(0.5)
		border = border.ScaleAlpha(0.5)
		fore = fore.Mix(th.Card, 0.5)
	}
	return bg, border, fore
}

// toggleRadius 返回尺寸档的圆角: 默认/大档 rounded-lg(= RadiusLg),
// 小档 rounded-[min(var(--radius-md),12px)] = RadiusMd。
func toggleRadius(th *xui.Theme, size ToggleSize) int32 {
	if size == ToggleSizeSm {
		return th.RadiusMd()
	}
	return th.RadiusLg()
}

// corners 解析四角半径: 组内 spacing=0 时只保留首项外侧角与末项外侧角
// (横向 = 左/右, 纵向 = 上/下), 其余情况四角都是尺寸档圆角。
func (t *Toggle) corners(th *xui.Theme, place toggleGroupPlacement) xui.Corners {
	r := toggleRadius(th, t.size)
	if !place.inGroup || !place.compact {
		return xui.UniformCorners(r)
	}
	var c xui.Corners
	if place.vertical {
		if place.first {
			c.TopLeft, c.TopRight = r, r
		}
		if place.last {
			c.BottomLeft, c.BottomRight = r, r
		}
		return c
	}
	if place.first {
		c.TopLeft, c.BottomLeft = r, r
	}
	if place.last {
		c.TopRight, c.BottomRight = r, r
	}
	return c
}

// drawContent 居中绘制 [图标 + gap + 文字]。
func (t *Toggle) drawContent(p *xui.Painter, th *xui.Theme, fore xui.Color) {
	sp := toggleSpecs[t.size]
	f := t.fontHandle(th)
	if f != 0 {
		p.Draw.SetFont(f)
	}
	cw := t.contentWidth(th)
	x := (p.Rc.Left + p.Rc.Right - cw) / 2
	midY := (p.Rc.Top + p.Rc.Bottom) / 2
	if t.icon != "" {
		rc := xc.RECT{
			Left:   x,
			Top:    midY - sp.icon/2,
			Right:  x + sp.icon,
			Bottom: midY + sp.icon/2,
		}
		p.Icon(t.icon, &rc, fore, 2)
		x += sp.icon
		if t.GetText() != "" {
			x += sp.gap
		}
	}
	if text := t.GetText(); text != "" {
		dy := textVDy(text)
		rc := xc.RECTF{
			Left:   float32(x),
			Top:    float32(p.Rc.Top) - dy,
			Right:  float32(x + xui.TextWidth(text, f)),
			Bottom: float32(p.Rc.Bottom) - dy,
		}
		p.TextF(text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	}
}

// ===== ToggleGroup =====

// ToggleGroup 把若干 Toggle 排成一组(对应 shadcn 的 ToggleGroup):
//   - 容器是 LayoutEle, 宽高按内容自撑, 子项间距由 spacing 决定(逻辑像素;
//     上游 spacing={2} = 8px, 默认同上游; connected = spacing 0);
//   - spacing=0 时首尾保留外圆角、相邻边收成直角(分段控件外观);
//   - type=single(默认)同组互斥, type=multiple 各自独立;
//   - orientation 横向(默认)/纵向, 纵向时条目拉伸到组内最宽的一项;
//   - 组级 variant/size 作用于所有条目, 条目自己的同名选项会被组级覆盖(同上游 context);
//   - 整组禁用忽略用户交互; OnValueChangeVeto 可否决组值变化(对应 Root 的 cancel)。
//
// 用法:
//
//	g := ui.NewToggleGroup(row.Handle, ui.WithToggleGroupVariant(ui.ToggleVariantOutline))
//	g.AddItem("bold", "Bold")
//	g.AddItem("italic", "Italic")
//	g.SetValue("bold")
type ToggleGroup struct {
	*widget.LayoutEle
	variant    ToggleVariant
	size       ToggleSize
	hasVariant bool
	hasSize    bool
	spacing    int32
	multiple   bool
	vertical   bool
	disabled   bool
	items      []*Toggle
	cb         func(values []string)
	veto       func(values []string) bool
	applying   bool
}

// ToggleGroupOption 切换按钮组配置项。
type ToggleGroupOption func(*ToggleGroup)

// WithToggleGroupVariant 设置组内条目的统一变体。
func WithToggleGroupVariant(v ToggleVariant) ToggleGroupOption {
	return func(g *ToggleGroup) { g.variant, g.hasVariant = v, true }
}

// WithToggleGroupSize 设置组内条目的统一尺寸档。
func WithToggleGroupSize(s ToggleSize) ToggleGroupOption {
	return func(g *ToggleGroup) { g.size, g.hasSize = s, true }
}

// WithToggleGroupSpacing 设置条目间距(逻辑像素; 默认 8 = 上游 spacing=2)。
func WithToggleGroupSpacing(px int32) ToggleGroupOption {
	return func(g *ToggleGroup) {
		if px < 0 {
			px = 0
		}
		g.spacing = px
	}
}

// WithToggleGroupConnected 让条目相邻无缝(spacing=0, 首尾保留外侧圆角) —— 分段控件外观。
func WithToggleGroupConnected() ToggleGroupOption {
	return func(g *ToggleGroup) { g.spacing = 0 }
}

// WithToggleGroupMultiple 改为多选(默认单选, 同组互斥且允许全部取消)。
func WithToggleGroupMultiple() ToggleGroupOption {
	return func(g *ToggleGroup) { g.multiple = true }
}

// WithToggleGroupVertical 纵向排列(对应 Base UI ToggleGroup 的 orientation="vertical")。
// 条目统一拉伸到组内最宽的一项。
func WithToggleGroupVertical() ToggleGroupOption {
	return func(g *ToggleGroup) { g.vertical = true }
}

// WithToggleGroupDisabled 整组禁用: 忽略用户交互(点击 / 空格回车), 程序化
// SetValue 仍然生效; 条目自身的禁用态独立记录, 整组启用后各自恢复。
func WithToggleGroupDisabled() ToggleGroupOption {
	return func(g *ToggleGroup) { g.disabled = true }
}

// NewToggleGroup 创建切换按钮组。组本身宽高自撑, 放进 HStack / 行列时无需指定尺寸。
func NewToggleGroup(hParent int, opts ...ToggleGroupOption) *ToggleGroup {
	g := &ToggleGroup{size: ToggleSizeDefault, spacing: 8}
	for _, o := range opts {
		o(g)
	}
	ele := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if ele == nil {
		return nil
	}
	g.LayoutEle = ele
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(g.spacing)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	g.applyOrientation()
	g.applyRole()
	return g
}

// applyOrientation 把横/纵排列写到布局上(横向 items-center, 纵向左对齐 + 条目等宽)。
func (g *ToggleGroup) applyOrientation() {
	if g == nil || g.LayoutEle == nil {
		return
	}
	g.EnableHorizon(!g.vertical)
	if g.vertical {
		g.SetAlignH(xcc.Layout_Align_Left)
		g.SetAlignV(xcc.Layout_Align_Top)
	} else {
		g.SetAlignV(xcc.Layout_Align_Center)
	}
}

// applyRole 把"是否无缝 + 方向"写到用户数据, 条目绘制时读取首尾圆角规则。
func (g *ToggleGroup) applyRole() {
	if g == nil {
		return
	}
	switch {
	case g.vertical && g.spacing == 0:
		g.SetUserData(uiRoleToggleGroupTightV)
	case g.vertical:
		g.SetUserData(uiRoleToggleGroupSpacedV)
	case g.spacing == 0:
		g.SetUserData(uiRoleToggleGroupTight)
	default:
		g.SetUserData(uiRoleToggleGroupSpaced)
	}
}

// fitVertical 在纵向模式下把组与条目统一成"组内最宽一项"的宽度
// (对应上游 data-vertical:items-stretch); 横向模式下不干预条目宽度。
func (g *ToggleGroup) fitVertical() {
	if g == nil || g.LayoutEle == nil || !g.vertical || len(g.items) == 0 {
		return
	}
	th := xui.Current()
	var maxW int32
	for _, it := range g.items {
		if w := it.naturalWidth(th); w > maxW {
			maxW = w
		}
	}
	g.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, maxW)
	for _, it := range g.items {
		it.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, maxW)
	}
}

// AddItem 追加一个条目并返回它。value 是组值(用于 Value/SetValue/OnChange),
// 可传空串; text 为空时配合 WithToggleIcon 得到纯图标条目。
func (g *ToggleGroup) AddItem(value, text string, opts ...ToggleOption) *Toggle {
	all := make([]ToggleOption, 0, len(opts)+2)
	all = append(all, opts...)
	// 组级设置覆盖条目自己的 variant/size(同上游 context.variant || variant)。
	if g.hasVariant {
		all = append(all, WithToggleVariant(g.variant))
	}
	if g.hasSize {
		all = append(all, WithToggleSize(g.size))
	}
	t := NewToggle(0, 0, 0, 0, text, g.Handle, all...)
	if t == nil {
		return nil
	}
	t.value = value
	t.layoutHook = func() {
		g.fitVertical()
		g.relayout()
	}
	g.items = append(g.items, t)
	t.guard = func(pressed bool) bool { return g.acceptItemCheck(t, pressed) }
	if g.disabled {
		t.setGroupDisabled(true)
	}
	g.fitVertical()
	g.relayout()
	return t
}

// Value 返回当前选中的值(按条目顺序)。单选组最多一个。
func (g *ToggleGroup) Value() []string {
	vals := make([]string, 0, len(g.items))
	for _, it := range g.items {
		if it.Pressed() {
			vals = append(vals, it.value)
		}
	}
	return vals
}

// FieldValue 实现 FieldValueProvider: 选中值用逗号连接, 未选为空串。
func (g *ToggleGroup) FieldValue() string {
	if g == nil {
		return ""
	}
	return strings.Join(g.Value(), ",")
}

// SetValue 设置选中值, 不触发 OnChange。单选组取第一个值, 传空清空全部。
func (g *ToggleGroup) SetValue(values ...string) *ToggleGroup {
	g.applying = true
	if g.multiple {
		for _, it := range g.items {
			it.SetPressed(containsString(values, it.value))
		}
	} else {
		want := ""
		if len(values) > 0 {
			want = values[0]
		}
		for _, it := range g.items {
			it.SetPressed(it.value == want)
		}
	}
	g.applying = false
	return g
}

// OnChange 注册选中值变化回调(用户点击/键盘切换时触发)。
func (g *ToggleGroup) OnChange(fn func(values []string)) *ToggleGroup {
	g.cb = fn
	return g
}

// OnValueChangeVeto 注册组值变化的否决回调(对应 Base UI ToggleGroup Root 的
// onValueChange 的 eventDetails.cancel())。参数是"将要变成的完整集合"(单选互斥
// 已算进去); 返回 false 时本次切换回滚, 条目状态恢复、OnChange 不触发。
// 程序化 SetValue 不经过否决。
func (g *ToggleGroup) OnValueChangeVeto(fn func(values []string) bool) *ToggleGroup {
	g.veto = fn
	return g
}

// SetDisabled 整组禁用/启用: 只忽略用户交互, 程序化 SetValue 仍然生效。
func (g *ToggleGroup) SetDisabled(v bool) *ToggleGroup {
	if g == nil || g.disabled == v {
		return g
	}
	g.disabled = v
	for _, it := range g.items {
		it.setGroupDisabled(v)
	}
	return g
}

// Disabled 报告整组是否禁用。
func (g *ToggleGroup) Disabled() bool { return g != nil && g.disabled }

// Vertical 报告是否纵向排列。
func (g *ToggleGroup) Vertical() bool { return g != nil && g.vertical }

// SetVertical 运行时切换横/纵排列。
func (g *ToggleGroup) SetVertical(v bool) *ToggleGroup {
	if g == nil || g.vertical == v {
		return g
	}
	g.vertical = v
	g.applyOrientation()
	g.applyRole()
	th := xui.Current()
	for _, it := range g.items {
		if v {
			it.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, it.naturalWidth(th))
		} else if it.autoWidth {
			it.SetWidth(it.naturalWidth(th))
			it.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
		}
		it.Redraw(false) // 内角/外角随方向改变
	}
	g.fitVertical()
	g.relayout()
	return g
}

// SetSpacing 运行时修改条目间距(逻辑像素)。
func (g *ToggleGroup) SetSpacing(px int32) *ToggleGroup {
	if px < 0 {
		px = 0
	}
	if g.spacing == px {
		return g
	}
	g.spacing = px
	g.SetSpace(px)
	g.applyRole()
	for _, it := range g.items {
		it.Redraw(false) // 内角/外角随 spacing 改变
	}
	g.fitVertical()
	g.relayout()
	return g
}

// acceptItemCheck 是条目切换的准入判定(在用户回调之前):
//   - 组禁用 → 回滚并拒绝;
//   - OnValueChangeVeto 返回 false → 回滚并拒绝;
//   - 通过后执行单选互斥并触发组回调。
func (g *ToggleGroup) acceptItemCheck(item *Toggle, on bool) bool {
	if g.applying {
		return true
	}
	next := g.valuesAfter(item, on)
	if g.disabled || (g.veto != nil && !g.veto(next)) {
		g.applying = true
		item.SetPressed(!on)
		g.applying = false
		return false
	}
	if on && !g.multiple {
		g.applying = true
		for _, it := range g.items {
			if it != item {
				it.SetPressed(false)
			}
		}
		g.applying = false
	}
	if g.cb != nil {
		g.cb(next)
	}
	return true
}

// valuesAfter 算出"本次切换完成后"的组值, 不改状态(否决回调要用)。
func (g *ToggleGroup) valuesAfter(item *Toggle, on bool) []string {
	if !g.multiple {
		if !on {
			return []string{}
		}
		return []string{item.value}
	}
	vals := make([]string, 0, len(g.items))
	for _, it := range g.items {
		pressed := it.Pressed()
		if it == item {
			pressed = on
		}
		if pressed {
			vals = append(vals, it.value)
		}
	}
	return vals
}

// relayout 让组的 Auto 宽高按新的子项重新量一遍。
func (g *ToggleGroup) relayout() {
	if p := g.GetParentEle(); p != 0 {
		xc.XEle_AdjustLayout(p)
	} else {
		g.AdjustLayout()
	}
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// toggleGroupPlacement 描述条目在无缝 ToggleGroup 里的位置。
type toggleGroupPlacement struct {
	inGroup  bool
	compact  bool // spacing=0: 首尾保留外圆角
	vertical bool // 纵向组: 首尾角落在上/下
	first    bool
	last     bool
}

func isToggleGroupRole(role int) bool {
	return role >= uiRoleToggleGroupTight && role <= uiRoleToggleGroupSpacedV
}

// toggleParentRole 返回直接父元素上的组标记(不是 ToggleGroup 时为 0)。
func toggleParentRole(hEle int) int {
	parent := xc.XWidget_GetParentEle(hEle)
	if parent == 0 {
		return 0
	}
	role := xc.XEle_GetUserData(parent)
	if !isToggleGroupRole(role) {
		return 0
	}
	return role
}

// toggleGroupPlacementOf 实时读取条目在组里的位置(不缓存, 增删条目后自动正确)。
func toggleGroupPlacementOf(hEle int) toggleGroupPlacement {
	parent := xc.XWidget_GetParentEle(hEle)
	if parent == 0 {
		return toggleGroupPlacement{}
	}
	role := xc.XEle_GetUserData(parent)
	if !isToggleGroupRole(role) {
		return toggleGroupPlacement{}
	}
	p := toggleGroupPlacement{
		inGroup:  true,
		compact:  role == uiRoleToggleGroupTight || role == uiRoleToggleGroupTightV,
		vertical: role == uiRoleToggleGroupTightV || role == uiRoleToggleGroupSpacedV,
	}
	n := int(xc.XEle_GetChildCount(parent))
	idx := -1
	for i := 0; i < n; i++ {
		if xc.XEle_GetChildByIndex(parent, int32(i)) == hEle {
			idx = i
			break
		}
	}
	if idx < 0 {
		return toggleGroupPlacement{}
	}
	p.first, p.last = idx == 0, idx == n-1
	return p
}
