package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// ToggleVariant 切换按钮变体, 对应 shadcn toggle.tsx 的 variant。
type ToggleVariant int

const (
	ToggleVariantDefault ToggleVariant = iota // 透明底, 悬停 bg-muted, 选中 bg-accent
	ToggleVariantOutline                      // 1px border-input, 悬停/选中填充 accent
)

// ToggleSize 切换按钮尺寸档, 对应 shadcn toggle.tsx 的 size(h-9 / h-8 / h-10)。
type ToggleSize int

const (
	ToggleSizeDefault ToggleSize = iota // h-9(36)
	ToggleSizeSm                        // h-8(32)
	ToggleSizeLg                        // h-10(40)
)

// toggleSpec 是尺寸规格(逻辑像素): 高度、左右内边距与最小宽度。
type toggleSpec struct {
	height   int32
	padX     int32
	minWidth int32
}

var toggleSpecs = map[ToggleSize]toggleSpec{
	ToggleSizeDefault: {height: 36, padX: 8, minWidth: 36},  // h-9 min-w-9 px-2
	ToggleSizeSm:      {height: 32, padX: 6, minWidth: 32},  // h-8 min-w-8 px-1.5
	ToggleSizeLg:      {height: 40, padX: 10, minWidth: 40}, // h-10 min-w-10 px-2.5
}

const (
	toggleIconGap  int32 = 8  // gap-2
	toggleIconSize int32 = 18 // [&_svg]:size-4(16) + 2px 描边补偿, 同 Button
	// toggleGroupPadX 是组内条目的左右内边距: ToggleGroupItem 的 px-3 覆盖尺寸档的 px。
	toggleGroupPadX int32 = 12
)

// Toggle 双状态按钮: 载体为 Button_Type_Check 原生按钮(点击/空格/回车切换, 带键盘焦点),
// 壳体与内容全部自绘。
//
// 与 shadcn toggle.tsx(radix-nova) 对齐:
//   - 未选中: 透明底 + 前景色; 悬停 bg-muted + muted-foreground;
//   - 选中(data-state=on): bg-accent + accent-foreground;
//   - outline: border-input, 悬停/选中填充 accent(shadow-xs 不画, 见映射表);
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
}

// ToggleOption 切换按钮配置项。
type ToggleOption func(*Toggle)

// WithToggleVariant 设置变体。
func WithToggleVariant(v ToggleVariant) ToggleOption { return func(t *Toggle) { t.variant = v } }

// WithToggleSize 设置尺寸档(默认 36 高)。
func WithToggleSize(s ToggleSize) ToggleOption { return func(t *Toggle) { t.size = s } }

// WithTogglePressed 设置初始选中态。
func WithTogglePressed(v bool) ToggleOption { return func(t *Toggle) { t.SetPressed(v) } }

// WithToggleIcon 让切换按钮在文字前绘制一枚行内图标。
func WithToggleIcon(path string) ToggleOption { return func(t *Toggle) { t.icon = path } }

// WithToggleInvalid 设置校验失败态(对应 shadcn 的 aria-invalid)。
func WithToggleInvalid() ToggleOption { return func(t *Toggle) { t.invalid = true } }

// NewToggle 创建切换按钮。cy 传 0 用尺寸档高度; cx 传 0 按内容量宽
// (不小于尺寸档的 min-width; 位于 ToggleGroup 内时按 px-3、min-w-0 量宽)。
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
	return t
}

// setPressedNotify 供键盘路径使用: 写状态并触发与鼠标点击相同的回调。
func (t *Toggle) setPressedNotify(v bool) {
	t.SetCheck(v)
	t.Redraw(false)
	t.notifyChange(v)
}

// notifyChange 触发变更监听(独立 OnChange 与所属组共用)。
func (t *Toggle) notifyChange(pressed bool) {
	for _, fn := range t.changes {
		fn(pressed)
	}
}

// addChangeListener 注册内部监听(组用它做互斥与组回调)。
func (t *Toggle) addChangeListener(fn func(pressed bool)) {
	t.changes = append(t.changes, fn)
}

// Pressed 返回是否处于选中态。
func (t *Toggle) Pressed() bool { return t.IsCheck() }

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
	t.addChangeListener(fn)
	return t
}

// SetText 修改文字; 自适应宽度时重新量宽。
func (t *Toggle) SetText(text string) *Toggle {
	t.Button.SetText(text)
	if t.autoWidth {
		t.SetWidth(t.naturalWidth(xui.Current()))
	}
	t.Redraw(false)
	return t
}

// SetIcon 设置/替换行内图标(空串移除)。
func (t *Toggle) SetIcon(path string) *Toggle {
	t.icon = path
	if t.autoWidth {
		t.SetWidth(t.naturalWidth(xui.Current()))
	}
	t.Redraw(false)
	return t
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

// FitWidth 按当前文字/图标重新量宽(仅 cx 传 0 创建的切换按钮)。
func (t *Toggle) FitWidth() *Toggle {
	if !t.autoWidth {
		return t
	}
	t.SetWidth(t.naturalWidth(xui.Current()))
	t.Redraw(false)
	return t
}

// naturalWidth 量出自适应宽度。组内条目按 px-3 + min-w-0, 组外按尺寸档的
// px 与 min-width(同 toggleVariants 的 size 类与 ToggleGroupItem 的 px-3 覆盖)。
func (t *Toggle) naturalWidth(th *xui.Theme) int32 {
	sp := toggleSpecs[t.size]
	pad, minW := sp.padX, sp.minWidth
	if toggleParentRole(t.Handle) != 0 {
		pad, minW = toggleGroupPadX, 0
	}
	w := pad*2 + t.contentWidth(th)
	if w < minW {
		w = minW
	}
	return w
}

// contentWidth 返回图标 + 间距 + 文字的宽度。
func (t *Toggle) contentWidth(th *xui.Theme) int32 {
	f := t.fontHandle(th)
	w := xui.TextWidth(t.GetText(), f)
	if t.icon != "" {
		if w > 0 {
			w += toggleIconGap
		}
		w += toggleIconSize
	}
	return w
}

// fontHandle 返回 text-sm(14px) 对应的字体句柄。
func (t *Toggle) fontHandle(th *xui.Theme) int {
	return xui.FontHandle(xui.FontSpec{Family: th.FontDefault.Family, Size: 14})
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

// resolve 解析壳体与文字颜色, 逐条对应 toggleVariants 的类名。
func (t *Toggle) resolve(th *xui.Theme, st xui.State, on bool, hv float64) (bg, border, fore xui.Color) {
	var bgHV, foreHV xui.Color
	switch t.variant {
	case ToggleVariantOutline:
		border = th.Input
		bg, bgHV = xui.None, th.Accent
		fore, foreHV = th.Foreground, th.AccentForeground
	default:
		bg, bgHV = xui.None, th.Muted
		fore, foreHV = th.Foreground, th.MutedForeground
	}
	// data-[state=on] 排在 hover 之后: 选中态始终用 accent, 悬停不再改底色/字色。
	if on {
		bg, bgHV = th.Accent, th.Accent
		fore, foreHV = th.AccentForeground, th.AccentForeground
	}
	bg = xui.Fade(bg, bgHV, hv)
	fore = xui.Fade(fore, foreHV, hv)

	if st.Disabled {
		// disabled:opacity-50 是整元素半透明: 壳体加 alpha, 文字按宿主底色预混。
		bg = bg.ScaleAlpha(0.5)
		border = border.ScaleAlpha(0.5)
		fore = fore.Mix(th.Card, 0.5)
	}
	return bg, border, fore
}

// corners 解析四角半径: 组内 spacing=0 时只保留首项左外角与末项右外角
// (ToggleGroupItem 的 rounded-l-md / rounded-r-md), 其余情况都是 rounded-md。
func (t *Toggle) corners(th *xui.Theme, place toggleGroupPlacement) xui.Corners {
	r := th.RadiusMd()
	if !place.inGroup || !place.compact {
		return xui.UniformCorners(r)
	}
	var c xui.Corners
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
			Top:    midY - toggleIconSize/2,
			Right:  x + toggleIconSize,
			Bottom: midY + toggleIconSize/2,
		}
		p.Icon(t.icon, &rc, fore, 2)
		x += toggleIconSize
		if t.GetText() != "" {
			x += toggleIconGap
		}
	}
	if text := t.GetText(); text != "" {
		rc := xc.RECT{
			Left:   x,
			Top:    p.Rc.Top - 1, // Vcenter 字形偏下 1px(同 Button)
			Right:  x + xui.TextWidth(text, f),
			Bottom: p.Rc.Bottom - 1,
		}
		p.Text(text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	}
}

// ===== ToggleGroup =====

// ToggleGroup 把若干 Toggle 排成一组(对应 shadcn 的 ToggleGroup):
//   - 容器是 LayoutEle, 宽高按内容自撑, 子项间距由 spacing 决定(逻辑像素;
//     上游 spacing={2} = 8px);
//   - spacing=0 时首尾保留外圆角、相邻边收成直角(分段控件外观);
//   - type=single(默认)同组互斥, type=multiple 各自独立;
//   - 组级 variant/size 作用于所有条目, 条目自己的同名选项会被组级覆盖(同上游 context)。
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
	items      []*Toggle
	cb         func(values []string)
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

// WithToggleGroupSpacing 设置条目间距(逻辑像素; 默认 0 = 相邻无缝)。
func WithToggleGroupSpacing(px int32) ToggleGroupOption {
	return func(g *ToggleGroup) {
		if px < 0 {
			px = 0
		}
		g.spacing = px
	}
}

// WithToggleGroupMultiple 改为多选(默认单选, 同组互斥且允许全部取消)。
func WithToggleGroupMultiple() ToggleGroupOption {
	return func(g *ToggleGroup) { g.multiple = true }
}

// NewToggleGroup 创建切换按钮组。组本身宽高自撑, 放进 HStack / 行列时无需指定尺寸。
func NewToggleGroup(hParent int, opts ...ToggleGroupOption) *ToggleGroup {
	g := &ToggleGroup{size: ToggleSizeDefault}
	for _, o := range opts {
		o(g)
	}
	ele := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if ele == nil {
		return nil
	}
	g.LayoutEle = ele
	ele.EnableHorizon(true)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(g.spacing)
	ele.SetAlignV(xcc.Layout_Align_Top)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	g.applyRole()
	return g
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
	g.items = append(g.items, t)
	t.addChangeListener(func(pressed bool) {
		g.onItemCheck(t, pressed)
	})
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
	g.relayout()
	return g
}

// applyRole 把"是否无缝"写到用户数据, 条目绘制时读取首尾圆角规则。
func (g *ToggleGroup) applyRole() {
	if g.spacing == 0 {
		g.SetUserData(uiRoleToggleGroupTight)
	} else {
		g.SetUserData(uiRoleToggleGroupSpaced)
	}
}

// onItemCheck 处理条目切换: 单选组先取消其它条目, 再通知组回调。
func (g *ToggleGroup) onItemCheck(item *Toggle, on bool) {
	if g.applying {
		return
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
		g.cb(g.Value())
	}
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
	inGroup bool
	compact bool // spacing=0: 首尾保留外圆角
	first   bool
	last    bool
}

// toggleParentRole 返回直接父元素上的组标记(不是 ToggleGroup 时为 0)。
func toggleParentRole(hEle int) int {
	parent := xc.XWidget_GetParentEle(hEle)
	if parent == 0 {
		return 0
	}
	role := xc.XEle_GetUserData(parent)
	if role != uiRoleToggleGroupTight && role != uiRoleToggleGroupSpaced {
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
	if role != uiRoleToggleGroupTight && role != uiRoleToggleGroupSpaced {
		return toggleGroupPlacement{}
	}
	p := toggleGroupPlacement{inGroup: true, compact: role == uiRoleToggleGroupTight}
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
