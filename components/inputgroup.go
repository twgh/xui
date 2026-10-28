package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Input Group 几何(逻辑像素), 对齐 input-group.tsx(radix-nova):
//   - 容器 h-9(36) / rounded-lg / border-input / shadow-xs;
//   - 内层输入框是 input.tsx 的 h-8(32), 在 36 高的容器里上下各留 2;
//   - addon: inline 侧 pl-2/pr-2(8), block 侧 px-2.5(10) + pt-2/pb-2(8), gap-2(8);
//   - 内层输入框与 addon 相邻的一侧内边距 6(pl-1.5/pr-1.5), 另一侧保持 12(px-3);
//   - addon 内按钮默认 ghost + xs(h-6 = 24), 与 addon 边缘抵消 5(has-[>button]:-ml-[0.3rem])。
const (
	inputGroupH         int32 = 36
	inputGroupInputH    int32 = 32
	inputGroupPadX      int32 = 12
	inputGroupInlineX   int32 = 8
	inputGroupBlockX    int32 = 10
	inputGroupBlockY    int32 = 8
	inputGroupGap       int32 = 8
	inputGroupBtnPull   int32 = 5 // addon 首/尾按钮的外扩量(负外边距)
	inputGroupTextH     int32 = 24
	inputGroupIconSz    int32 = 16
	inputGroupTextareaH int32 = 64
)

// InputGroupAlign 附加件位置, 对应 input-group.tsx 的 align。
type InputGroupAlign int

const (
	// InputGroupInlineStart 输入框起始侧(左), 默认。
	InputGroupInlineStart InputGroupAlign = iota
	// InputGroupInlineEnd 输入框末尾侧(右)。
	InputGroupInlineEnd
	// InputGroupBlockStart 输入框上方(整宽)。会使 Input Group 变成纵向三段式。
	InputGroupBlockStart
	// InputGroupBlockEnd 输入框下方(整宽)。
	InputGroupBlockEnd
)

// InputGroupOption Input Group 配置项。
type InputGroupOption func(*InputGroup)

// WithInputGroupInvalid 设置校验失败态(对应内层控件的 aria-invalid):
// 容器统一画红边框 + ring-3 destructive/20(暗色 /40)。
func WithInputGroupInvalid() InputGroupOption {
	return func(g *InputGroup) { g.invalid = true }
}

// WithInputGroupDisabled 禁用整组(输入框、addon 内按钮、addon 文字全部降到 50%)。
func WithInputGroupDisabled() InputGroupOption {
	return func(g *InputGroup) { g.disabled = true }
}

// InputGroup 输入组: 输入框与附加内容共用一层边框与焦点环。
//
// 对齐 shadcn input-group.tsx(radix-nova)。骨架分三行(纵向):
//
//	[ block-start addon ]   (整宽, 如"Full Name"标题)
//	[ inline-start addon | 输入框 | inline-end addon ]
//	[ block-end addon ]     (整宽, 如字数 / 发送按钮)
//
// 这里**不能**直接把内层输入框当普通 Input 用: Input 自己会画壳体与焦点环,
// 组里再画一层就成了双框。所以内层是"裸输入"(WithInputBare), 边框/环/禁用底色
// 全部由本容器在 Paint 里统一绘制; 输入框与 addon 相邻的一侧把文字内边距收到 6,
// 另一侧保持 12 —— 与上游 `has-[>[data-align=inline-start]]:[&>input]:pl-1.5` 一致。
type InputGroup struct {
	*widget.LayoutEle
	top, inner, bottom *widget.LayoutEle
	input              *Input
	addons             []*InputGroupAddon
	buttons            []*Button
	invalid            bool
	disabled           bool
	block              bool // 有 block addon(外层变 Auto 高)
	textarea           bool
}

// NewInputGroup 创建输入组。cy 传 0 使用默认高度 36(h-9);
// 宽度通常配 xui.ItemFill 让它铺满, cx 传 0 时需要调用方声明布局项宽度。
func NewInputGroup(x, y, cx, cy int32, hParent int, opts ...InputGroupOption) *InputGroup {
	g := &InputGroup{}
	for _, o := range opts {
		o(g)
	}
	if cy <= 0 {
		cy = inputGroupH
	}
	outer := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if outer == nil {
		return nil
	}
	g.LayoutEle = outer
	outer.EnableHorizon(false)
	outer.EnableLayout(true)
	outer.EnableBkTransparent(true)
	outer.EnableAutoWrap(false)
	outer.SetSpace(0)
	outer.SetAlignH(xcc.Layout_Align_Left)
	outer.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	outer.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, cy)
	outer.AddEvent_Paint(g.onPaint)

	// block-start / block-end 先建好并隐藏: 布局会跳过 Show(false) 的子项,
	// 所以"三段式"与"单行"共用同一套骨架, 后面加 addon 时只是 Show(true)。
	g.top = g.newRow(outer.Handle)
	g.top.SetPadding(inputGroupBlockX, inputGroupBlockY, inputGroupBlockX, 0)
	g.top.SetAlignH(xcc.Layout_Align_Left)
	g.top.SetAlignV(xcc.Layout_Align_Center)
	g.top.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	g.top.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	g.top.Show(false)

	g.inner = g.newRow(outer.Handle)
	g.inner.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	g.inner.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, inputGroupInputH)
	g.inner.SetAlignV(xcc.Layout_Align_Center)
	xc.XWidget_LayoutItem_SetMargin(g.inner.Handle, 0, 2, 0, 2)

	g.bottom = g.newRow(outer.Handle)
	g.bottom.SetPadding(inputGroupBlockX, 0, inputGroupBlockX, inputGroupBlockY)
	g.bottom.SetAlignH(xcc.Layout_Align_Left)
	g.bottom.SetAlignV(xcc.Layout_Align_Center)
	g.bottom.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	g.bottom.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	g.bottom.Show(false)

	if g.disabled {
		g.SetDisabled(true)
	}
	xui.AttachWindow(outer.Handle)
	return g
}

// newRow 建一个横向流容器(子项垂直居中、不换行)。top 必须排在内层输入行之**前**,
// 所以它直接挂在 group 上(创建顺序 top → inner → bottom 决定纵向顺序)。
func (g *InputGroup) newRow(hParent int) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if l == nil {
		return nil
	}
	l.EnableHorizon(true)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(inputGroupGap)
	l.SetAlignV(xcc.Layout_Align_Center)
	return l
}

// Input 创建(或返回)组内输入框。内层是裸输入: 不画壳体与焦点环, 由本容器统一绘制。
// 选项与 ui.NewInput 相同(占位符/校验/附加件等), 但 WithMultiline 请改用 Textarea。
func (g *InputGroup) Input(opts ...InputOption) *Input {
	if g == nil || g.inner == nil {
		return nil
	}
	if g.input != nil {
		return g.input
	}
	all := make([]InputOption, 0, len(opts)+1)
	all = append(all, WithInputBare())
	all = append(all, opts...)
	in := NewInput(0, 0, 0, 0, g.inner.Handle, all...)
	if in == nil {
		return nil
	}
	in.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	in.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, inputGroupInputH)
	g.input = in
	// 聚焦/失焦要让容器重画焦点环(内层是裸的, 环只有容器画)。这两件事只能用
	// SetFocus/KillFocus 事件驱动 —— 容器的 Paint 只在自身重绘时才读状态。
	in.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		g.Redraw(true)
		return 0
	})
	in.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		g.Redraw(true)
		return 0
	})
	g.syncInsets()
	return in
}

// Textarea 创建组内文本域(多行裸输入)。cy 传 0 使用 64(min-h-16);
// 有文本域时组高按内容自撑(与上游 has-[>textarea]:h-auto 一致)。
func (g *InputGroup) Textarea(cy int32, opts ...InputOption) *Input {
	if g == nil || g.inner == nil {
		return nil
	}
	if g.input != nil {
		return g.input
	}
	if cy <= 0 {
		cy = inputGroupTextareaH
	}
	all := make([]InputOption, 0, len(opts)+2)
	all = append(all, WithInputBare(), WithMultiline())
	all = append(all, opts...)
	in := NewInput(0, 0, 0, 0, g.inner.Handle, all...)
	if in == nil {
		return nil
	}
	in.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	in.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, cy)
	g.input, g.textarea = in, true
	in.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		g.Redraw(true)
		return 0
	})
	in.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		g.Redraw(true)
		return 0
	})
	// 文本域形态: 外层高度交给内容, 内行高度就是文本域高
	xc.XWidget_LayoutItem_SetMargin(g.inner.Handle, 0, 0, 0, 0)
	g.inner.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	g.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	g.syncInsets()
	return in
}

// Addon 添加一个附加件。align 决定它排在输入框的哪一侧:
// inline-* 与输入框同排, block-* 独占一行(整宽)。
func (g *InputGroup) Addon(align InputGroupAlign, build ...func(*InputGroupAddon)) *InputGroupAddon {
	if g == nil || g.inner == nil {
		return nil
	}
	parent := g.inner.Handle
	switch align {
	case InputGroupBlockStart:
		parent = g.top.Handle
	case InputGroupBlockEnd:
		parent = g.bottom.Handle
	}
	a := &InputGroupAddon{group: g, align: align}
	l := g.newRow(parent)
	if l == nil {
		return nil
	}
	a.LayoutEle = l
	// 竖向位置: inline 侧填满输入行(32), block 侧按内容。
	// 宽度不用 Auto —— 嵌套 LayoutEle 的 Auto 宽在横向主轴里会算成 0(addon 整个消失),
	// 所以由 a.grow()/applyWidth() 按内容累加并锁 Fixed 宽(见 DESIGN 147)。
	if align == InputGroupInlineStart || align == InputGroupInlineEnd {
		l.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
		if align == InputGroupInlineStart {
			a.padL = inputGroupInlineX
		} else {
			a.padR = inputGroupInlineX
		}
		l.SetPadding(a.padL, 0, a.padR, 0)
		a.applyWidth()
		if align == InputGroupInlineStart {
			// 前置 addon 必须排在输入框**左边**(上游是 order-first)。布局按子元素 z 序排,
			// 而调用方通常先 Input 再 Addon, 后建的自然落在右边 —— 用 SetZOrder(0) 挪到首位
			// (实测: xcgui 的子项索引 = z 序 = 布局顺序; InsertChild 对已布局的子项会卡死)。
			xc.XEle_SetZOrder(l.Handle, 0)
			g.AdjustLayout()
		}
	} else {
		l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	}
	g.addons = append(g.addons, a)

	// block addon 出现后: 显示对应行, 外层改 Auto 高
	if align == InputGroupBlockStart {
		g.top.Show(true)
		g.markBlock()
	} else if align == InputGroupBlockEnd {
		g.bottom.Show(true)
		g.markBlock()
	}
	g.syncInsets()
	for _, fn := range build {
		fn(a)
	}
	if g.disabled {
		g.SetDisabled(true)
	}
	return a
}

// markBlock 切成"三段式"布局: 外层高度交给内容, 内行高度固定 32(不再上下留 2)。
func (g *InputGroup) markBlock() {
	if g.block {
		return
	}
	g.block = true
	g.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xc.XWidget_LayoutItem_SetMargin(g.inner.Handle, 0, 0, 0, 0)
	if !g.textarea {
		g.inner.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, inputGroupInputH)
	}
	g.AdjustLayout()
}

// syncInsets 按 inline addon 的有无设置内层输入框的左右文字内边距:
// 与 addon 相邻的一侧 6(pl-1.5/pr-1.5), 另一侧 12(px-3)。
func (g *InputGroup) syncInsets() {
	if g.input == nil {
		return
	}
	l, r := inputGroupPadX, inputGroupPadX
	for _, a := range g.addons {
		switch a.align {
		case InputGroupInlineStart:
			l = 6
		case InputGroupInlineEnd:
			r = 6
		}
	}
	g.input.setBareInsets(l, r)
}

// SetInvalid 设置/清除整组校验失败态。
func (g *InputGroup) SetInvalid(v bool) *InputGroup {
	if g == nil || g.invalid == v {
		return g
	}
	g.invalid = v
	g.Redraw(true)
	return g
}

// Invalid 返回当前是否处于校验失败态。
func (g *InputGroup) Invalid() bool { return g != nil && g.invalid }

// SetDisabled 禁用/启用整组。
func (g *InputGroup) SetDisabled(v bool) *InputGroup {
	if g == nil {
		return g
	}
	g.disabled = v
	if g.input != nil {
		g.input.Enable(!v)
	}
	for _, b := range g.buttons {
		b.Enable(!v)
	}
	// addon 内容整体 50%(radix: group-data-[disabled=true]/input-group:opacity-50)。
	// 逐个 SetSubtreeAlpha 而不是 Show(false): 禁用时内容仍在, 只是变淡。
	for _, a := range g.addons {
		if a == nil || a.LayoutEle == nil {
			continue
		}
		xui.SetSubtreeAlpha(0.5, a.LayoutEle.Handle)
	}
	if !v {
		for _, a := range g.addons {
			if a != nil && a.LayoutEle != nil {
				xui.SetSubtreeAlpha(1, a.LayoutEle.Handle)
			}
		}
	}
	g.Redraw(true)
	return g
}

// Disabled 返回整组是否禁用。
func (g *InputGroup) Disabled() bool { return g != nil && g.disabled }

// Focus 把焦点放到组内输入框。
func (g *InputGroup) Focus() *InputGroup {
	if g != nil && g.input != nil {
		g.input.SetFocus()
	}
	return g
}

// focused 判断组内输入框是否持有焦点(内层是裸输入, 环由容器按这个状态画)。
func (g *InputGroup) focused() bool {
	return g != nil && g.input != nil && xui.StateOf(g.input.Handle).Focused
}

func (g *InputGroup) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	// 背板: 亮色 bg-transparent(等价卡片/页面底色), 暗色 bg-input/30
	bg, border := t.Background, t.Input
	if t.IsDark() {
		bg = t.Background.Mix(t.Input, 0.3)
	}
	if g.disabled {
		// disabled:bg-input/50(暗色 /80) + 组级 opacity-50, 与 Input 同款叠法
		a := 0.5
		if t.IsDark() {
			a = 0.8
		}
		bg = t.Input.ScaleAlpha(a).ScaleAlpha(0.5)
		border = t.Input.ScaleAlpha(0.5)
	}
	corners := groupCorners(hEle, t.RadiusLg())
	const inset = int32(3)
	switch {
	case g.invalid && !g.disabled:
		halo, borderColor := t.Destructive.WithAlpha(51), t.Destructive
		if t.IsDark() {
			halo, borderColor = t.Destructive.WithAlpha(102), t.Destructive.WithAlpha(128)
		}
		p.RingFrameCorners(halo, bg, corners)
		inner := xc.RECT{
			Left:   p.Rc.Left + inset,
			Top:    p.Rc.Top + inset,
			Right:  p.Rc.Right - inset,
			Bottom: p.Rc.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.StrokeRoundCorners(&inner, borderColor, corners.Inset(inset), 1)
		}
	case g.focused() && !g.disabled:
		p.FocusFrameCorners(t.Ring, bg, corners)
		inner := xc.RECT{
			Left:   p.Rc.Left + inset,
			Top:    p.Rc.Top + inset,
			Right:  p.Rc.Right - inset,
			Bottom: p.Rc.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.StrokeRoundCorners(&inner, t.Ring, corners.Inset(inset), 1)
		}
	default:
		p.FillRoundCorners(&p.Rc, bg, corners)
		p.StrokeRoundCorners(&p.Rc, border, corners, 1)
	}
	return 0
}

// ===== 附加件 =====

// InputGroupAddon 组内附加件: 一行横向容器(间距 8), 可放文字/图标/按钮/Kbd/Spinner,
// 也可以直接拿 Content() 放任意控件(自己声明的控件请同时声明布局项宽度)。
type InputGroupAddon struct {
	*widget.LayoutEle
	group    *InputGroup
	align    InputGroupAlign
	padL     int32
	padR     int32
	contentW int32 // 已追加内容的宽度合计
	items    int32 // 已追加内容个数(用来算间距)
}

// grow 记一笔内容宽度并重算 addon 宽度。
func (a *InputGroupAddon) grow(w int32) {
	a.contentW += w
	a.items++
	a.applyWidth()
}

// applyWidth 把 addon 锁成"内容宽 + 左右内边距 + 间距"的 Fixed 宽。
// 不能用 Auto: 横向主轴上嵌套 LayoutEle 的 Auto 宽实测算成 0, 整个 addon 会消失。
func (a *InputGroupAddon) applyWidth() {
	if a == nil || a.LayoutEle == nil || a.items == 0 {
		return
	}
	if a.align == InputGroupBlockStart || a.align == InputGroupBlockEnd {
		return // block 侧是整宽(Fill), 宽度交给布局
	}
	w := a.contentW + a.padL + a.padR
	if a.items > 1 {
		w += (a.items - 1) * inputGroupGap
	}
	if w < 1 {
		w = 1
	}
	a.SetWidth(w)
	a.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	// 宽度是布局项声明, 必须让父级重排才生效(自己 AdjustLayout 只重排内部)。
	if a.group != nil {
		a.group.AdjustLayout()
	}
	a.AdjustLayout()
}

// Content 返回附加件容器句柄(自由放置原生控件用)。
func (a *InputGroupAddon) Content() int {
	if a == nil || a.LayoutEle == nil {
		return 0
	}
	return a.LayoutEle.Handle
}

// Text 追加一段附加文字(text-sm, muted-foreground)。
func (a *InputGroupAddon) Text(s string) *InputGroupAddon {
	if a == nil || s == "" {
		return a
	}
	t := xui.Current()
	w := inputTextWidth(t, s, t.FontDefault.Size)
	e := widget.NewElement(0, 0, w+1, inputGroupTextH, a.Handle)
	if e == nil {
		return a
	}
	e.EnableBkTransparent(true)
	e.EnableDrawBorder(false)
	e.EnableFocus(false)
	e.EnableSwitchFocus(false)
	e.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w+1)
	e.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, inputGroupTextH)
	draw := func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		tt := xui.Current()
		if f := xui.FontHandle(tt.FontDefault); f != 0 {
			p.Draw.SetFont(f)
		}
		p.Text(s, &p.Rc, tt.MutedForeground, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
		return 0
	}
	e.AddEvent_Paint(draw)
	xui.AttachWindow(e.Handle)
	a.grow(w + 1)
	return a
}

// Icon 追加一枚图标(size-4 = 16, muted-foreground)。path 可用 icons.IconXxx 常量。
func (a *InputGroupAddon) Icon(path string) *InputGroupAddon {
	if a == nil || path == "" {
		return a
	}
	e := widget.NewElement(0, 0, inputGroupIconSz, inputGroupIconSz, a.Handle)
	if e == nil {
		return a
	}
	e.EnableBkTransparent(true)
	e.EnableDrawBorder(false)
	e.EnableFocus(false)
	e.EnableSwitchFocus(false)
	e.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, inputGroupIconSz)
	e.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, inputGroupIconSz)
	e.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		p.Icon(path, &p.Rc, xui.Current().MutedForeground, 2)
		return 0
	})
	xui.AttachWindow(e.Handle)
	a.grow(inputGroupIconSz)
	return a
}

// Button 追加一个按钮(默认 ghost + xs = 24 高, 对应 InputGroupButton 的默认值)。
// 组内首个/末个按钮会自动外扩 5px, 让它的视觉内边距贴住容器边缘。
func (a *InputGroupAddon) Button(text string, opts ...ButtonOption) *Button {
	if a == nil {
		return nil
	}
	all := []ButtonOption{WithVariant(ButtonGhost), WithSize(ButtonSizeXs)}
	all = append(all, opts...)
	b := NewButton(0, 0, 0, 0, text, a.Handle, all...)
	if b == nil {
		return nil
	}
	b.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 24)
	a.pullButton(b)
	a.grow(b.GetWidth() - inputGroupBtnPull)
	a.group.buttons = append(a.group.buttons, b)
	return b
}

// IconButton 追加一个纯图标按钮(默认 ghost + icon-xs = 24×24)。
func (a *InputGroupAddon) IconButton(icon string, opts ...ButtonOption) *Button {
	if a == nil {
		return nil
	}
	all := []ButtonOption{WithVariant(ButtonGhost), WithSize(ButtonSizeIconXs), WithIcon(icon)}
	all = append(all, opts...)
	b := NewButton(0, 0, 0, 0, "", a.Handle, all...)
	if b == nil {
		return nil
	}
	b.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 24)
	a.pullButton(b)
	a.grow(b.GetWidth() - inputGroupBtnPull)
	a.group.buttons = append(a.group.buttons, b)
	return b
}

// pullButton 让 addon 内的首个/末个按钮向容器边缘外扩(has-[>button]:ml-[-0.3rem])。
func (a *InputGroupAddon) pullButton(b *Button) {
	if a == nil || b == nil {
		return
	}
	switch a.align {
	case InputGroupInlineStart, InputGroupBlockStart:
		xc.XWidget_LayoutItem_SetMargin(b.Handle, -inputGroupBtnPull, 0, 0, 0)
	case InputGroupInlineEnd, InputGroupBlockEnd:
		xc.XWidget_LayoutItem_SetMargin(b.Handle, 0, 0, -inputGroupBtnPull, 0)
	}
}

// Kbd 追加一个键帽(如 "⌘K"), 与 input-group 文档的 Kbd 示例一致。
// 返回附加件以便继续链式追加; 需要改键帽文字时用 NewKbd(a.Handle, ...) 自己建一个。
func (a *InputGroupAddon) Kbd(text string) *InputGroupAddon {
	if a == nil {
		return a
	}
	if k := NewKbd(a.Handle, text); k != nil {
		a.grow(k.GetWidth())
	}
	return a
}

// Spinner 追加一个加载指示器(16, muted), 用于 "Saving..." 这类进行中状态。
func (a *InputGroupAddon) Spinner() *Spinner {
	if a == nil {
		return nil
	}
	sp := NewSpinner(a.Handle, WithSpinnerSize(16), WithSpinnerMuted())
	if sp != nil {
		a.grow(16)
	}
	return sp
}

// Search 是"搜索组"的便捷写法: 前置搜索图标 + 占位符。
// 等价于 Addon(InputGroupInlineStart).Icon(icons.IconSearch) + Input(WithPlaceholder(...))。
func (g *InputGroup) Search(placeholder string, opts ...InputOption) *Input {
	if g == nil {
		return nil
	}
	g.Addon(InputGroupInlineStart).Icon(icons.IconSearch)
	all := make([]InputOption, 0, len(opts)+1)
	all = append(all, WithPlaceholder(placeholder))
	all = append(all, opts...)
	return g.Input(all...)
}
