package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Field 几何常量(对齐 shadcn field.tsx 的 radix-nova 样式, 单位逻辑像素):
//   - Field: gap-3(12); 标签/标题/说明都是 text-sm(14) / 行高 20;
//   - FieldSet: gap-6(24); FieldGroup: gap-7(28), 嵌套组常用 gap-4(16);
//   - FieldLegend: mb-3(12), legend 16 / label 变体 14;
//   - FieldSeparator: h-5(20), 线画在垂直中点, 文字两侧各留 8(bg 遮线);
//   - FieldError: text-sm destructive, 多条错误间距 4、左缩进 16(list ml-4)。
const (
	fieldGap      int32 = 12
	fieldSetGap   int32 = 24
	fieldGroupGap int32 = 28
	fieldTextH    int32 = 20
	fieldLegendMb int32 = 12
	fieldSepH     int32 = 20
	fieldSepPadX  int32 = 8
	fieldErrorGap int32 = 4
	fieldErrorMl  int32 = 16
	fieldContentW int32 = 240 // 横向 Field 的 Content 默认宽(嵌套布局只认 Fixed 宽)
)

// FieldOrientation 字段取向, 对应 shadcn field.tsx 的 orientation。
//
// 上游还有 responsive(按容器宽度在 @md 断点切换横竖), Go 侧没有容器查询,
// 需要响应式时由调用方按窗口宽度自行选 FieldVertical / FieldHorizontal。
type FieldOrientation int

const (
	FieldVertical   FieldOrientation = iota // 竖排: 标签在上、控件在下
	FieldHorizontal                         // 横排: 标签在左、控件在右, 垂直居中
)

// FieldOption 字段配置项。
type FieldOption func(*Field)

// WithFieldOrientation 设置字段取向(默认 FieldVertical)。
func WithFieldOrientation(o FieldOrientation) FieldOption {
	return func(f *Field) { f.orientation = o }
}

// WithFieldInvalid 初始即为校验失败态(标签/标题用 destructive 色)。
// 运行时用 SetInvalid 切换。
func WithFieldInvalid() FieldOption {
	return func(f *Field) { f.invalid = true }
}

// Field 表单字段容器: 把标签、控件、说明、错误按 12px 间距排成一组。
//
// 与 shadcn field.tsx 的 Field 对齐:
//   - 默认竖排, 子项满宽; FieldHorizontal 时横排、垂直居中;
//   - 标签/标题 font-medium; 说明 muted; 校验失败时标签/标题转 destructive
//     (说明仍保持 muted, 同上游 text-muted-foreground 覆盖继承色);
//   - 不强制子项宽度: 竖排请对控件/说明用 `xui.ItemFill`, 横排请用 `xui.ItemWidth`
//     或让控件按自身宽度排(Field.Label 在横排下自动吃剩余宽度)。
//
// 标签/说明通过 Field 的方法创建, 这样 SetInvalid 能统一改色:
//
//	f := ui.NewField(0, 0, 0, 0, stack.Handle)
//	xui.ItemFill(f)
//	f.Label("邮箱地址")
//	xui.ItemFill(ui.NewInput(0, 0, 0, 0, f.Handle, ui.WithPlaceholder("m@example.com")))
//	f.Description("我们只会用它发送登录链接。")
type Field struct {
	*widget.LayoutEle
	orientation FieldOrientation
	invalid     bool
	labels      []*Label
	content     *widget.LayoutEle
}

// NewField 创建字段容器。宽铺满、高自撑; cy 仅作初始高度。
func NewField(x, y, cx, cy int32, hParent int, opts ...FieldOption) *Field {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	f := &Field{LayoutEle: ele}
	for _, o := range opts {
		o(f)
	}
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(fieldGap)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	f.applyOrientation()
	xui.AttachWindow(ele.Handle)
	return f
}

func (f *Field) applyOrientation() {
	if f.orientation == FieldHorizontal {
		f.EnableHorizon(true)
		f.SetAlignV(xcc.Layout_Align_Center)
	} else {
		f.EnableHorizon(false)
		f.SetAlignH(xcc.Layout_Align_Left)
	}
}

// Orientation 返回当前取向。
func (f *Field) Orientation() FieldOrientation {
	if f == nil {
		return FieldVertical
	}
	return f.orientation
}

// Label 创建字段标签(text-sm, font-medium), 并纳入 SetInvalid 的统一改色。
// 竖排下满宽; 横排下按权重吃剩余宽度(对应上游的 flex-auto)。
func (f *Field) Label(text string) *Label {
	return f.newText(text)
}

// Title 创建选项标题(用于复选框/单选框行), 样式与 Label 相同。
// 上游 FieldTitle 与 FieldLabel 的区别在 DOM 语义(label/div), Go 侧没有对应物。
func (f *Field) Title(text string) *Label {
	return f.newText(text)
}

// Description 创建字段说明(text-sm, muted)。多条说明按 Field 的 gap 堆叠。
// 说明文字较长时用 SetWrap 并自行给足高度(Label 不会随折行自撑)。
func (f *Field) Description(text string) *Label {
	l := NewLabel(0, 0, 0, fieldTextH, text, f.Handle).SetMuted().SetWrap()
	f.setChildWidth(l)
	return l
}

// Content 返回字段内容列(竖向, gap-6): 用于"标题 + 多行说明"或复选框组这类复合内容。
// 竖排下满宽; 横排下默认固定 240 宽 —— xcgui 不把 Weight/Percent 宽传给嵌套布局的
// Fill 子项(Fixed 才可靠), 需要更宽/更窄用 xui.ItemWidth(c, w) 覆盖。
func (f *Field) Content() *widget.LayoutEle {
	if f == nil {
		return nil
	}
	if f.content == nil {
		c := widget.NewLayoutEle(0, 0, 0, 0, f.Handle)
		c.EnableHorizon(false)
		c.EnableLayout(true)
		c.EnableBkTransparent(true)
		c.EnableAutoWrap(false)
		c.SetSpace(6) // gap-1.5
		c.SetAlignH(xcc.Layout_Align_Left)
		if f.orientation == FieldHorizontal {
			// 横向 Content 默认固定宽 240, 需要更宽/更窄用 xui.ItemWidth 覆盖。
			// xcgui 不会把 Weight/Percent 宽度传给嵌套布局的 Fill 子项(实测: 子项宽度恒为 0,
			// 内容整块不显示), 只有 Fixed 宽才可靠; 竖排的 Fill 不受影响。
			c.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, fieldContentW)
		} else {
			c.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		}
		c.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
		f.content = c
	}
	return f.content
}

// SetInvalid 切换校验失败态(标签/标题转 destructive), 返回自身便于链式调用。
func (f *Field) SetInvalid(invalid bool) *Field {
	if f == nil || f.invalid == invalid {
		return f
	}
	f.invalid = invalid
	for _, l := range f.labels {
		f.applyLabelColor(l)
	}
	f.Redraw(false)
	return f
}

// Invalid 返回当前校验失败态。
func (f *Field) Invalid() bool {
	return f != nil && f.invalid
}

func (f *Field) newText(text string) *Label {
	l := NewLabel(0, 0, 0, fieldTextH, text, f.Handle).SetMedium()
	f.setChildWidth(l)
	f.labels = append(f.labels, l)
	f.applyLabelColor(l)
	return l
}

func (f *Field) setChildWidth(l *Label) {
	if f.orientation == FieldHorizontal {
		xui.ItemWeight(l, 1)
	} else {
		xui.ItemFill(l)
	}
}

func (f *Field) applyLabelColor(l *Label) {
	if f.invalid {
		l.SetColorFunc(func(t *xui.Theme) xui.Color { return t.Destructive })
	} else {
		l.SetColorFunc(nil)
	}
}

// FieldSet 字段集容器(fieldset): 竖向流, 子项间距 24(gap-6), 宽铺满、高自撑。
//
// 与 shadcn field.tsx 的 FieldSet 对齐; 内部放 FieldLegend + FieldGroup 组成一组表单。
// 上游"包含 checkbox-group 时收成 gap-3"依赖 DOM 选择器, Go 侧需要时直接 SetSpace(12)。
func NewFieldSet(x, y, cx, cy int32, hParent int) *widget.LayoutEle {
	return newFieldStack(hParent, fieldSetGap)
}

// FieldGroup 字段组容器: 竖向流, 子项间距 28(gap-7), 宽铺满、高自撑。
// 嵌套组需要更紧的间距时调用 SetSpace(16)(上游嵌套 gap-4)。
func NewFieldGroup(x, y, cx, cy int32, hParent int) *widget.LayoutEle {
	return newFieldStack(hParent, fieldGroupGap)
}

func newFieldStack(hParent int, gap int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if l == nil {
		return nil
	}
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(gap)
	l.SetAlignH(xcc.Layout_Align_Left)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.AttachWindow(l.Handle)
	return l
}

// FieldLegendVariant 分组标题变体, 对应 shadcn field.tsx 的 FieldLegend variant。
type FieldLegendVariant int

const (
	FieldLegendDefault FieldLegendVariant = iota // legend: text-base(16) / 行高 24
	FieldLegendLabel                             // label: text-sm(14) / 行高 20
)

// FieldLegendOption 分组标题配置项。
type FieldLegendOption func(*fieldLegendConfig)

type fieldLegendConfig struct {
	variant FieldLegendVariant
}

// WithFieldLegendVariant 设置分组标题变体(默认 FieldLegendDefault)。
func WithFieldLegendVariant(v FieldLegendVariant) FieldLegendOption {
	return func(c *fieldLegendConfig) { c.variant = v }
}

// NewFieldLegend 创建分组标题(text-base 或 text-sm, font-medium, 下边距 12)。
// 宽度铺满, 通常作为 FieldSet 的第一个子项。
func NewFieldLegend(x, y, cx, cy int32, text string, hParent int, opts ...FieldLegendOption) *Label {
	cfg := fieldLegendConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	size, h := int32(16), int32(24)
	if cfg.variant == FieldLegendLabel {
		size, h = 14, fieldTextH
	}
	l := NewLabel(x, y, cx, h, text, hParent).SetFontSize(size).SetMedium()
	xui.ItemFill(l)
	l.LayoutItem_SetMargin(0, 0, 0, fieldLegendMb)
	return l
}

// FieldSeparator 字段分隔条: 1px 横线居中, 可带"或"这类居中文字(文字两侧用背景色压线)。
//
// 与 shadcn field.tsx 的 FieldSeparator 对齐: h-5(20), 上游的 `-my-2` 负外边距
// 在 xcgui 布局里不生效, 这里靠 Field 的 gap 控制垂直间距。
type FieldSeparator struct {
	*widget.Element
	text string
}

// NewFieldSeparator 创建分隔条。text 传空串 = 只画线; 非空 = 线 + 居中文字。
func NewFieldSeparator(x, y, cx, cy int32, hParent int, text string) *FieldSeparator {
	ele := widget.NewElement(x, y, cx, fieldSepH, hParent)
	if ele == nil {
		return nil
	}
	s := &FieldSeparator{Element: ele, text: text}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.AddEvent_Paint(s.onPaint)
	xui.AttachWindow(ele.Handle)
	return s
}

func (s *FieldSeparator) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	mid := (p.Rc.Top + p.Rc.Bottom) / 2
	p.FillRound(&xc.RECT{Left: p.Rc.Left, Top: mid, Right: p.Rc.Right, Bottom: mid + 1}, t.Border, 0)
	if s.text == "" {
		return 0
	}
	f := xui.FontHandle(t.FontDefault)
	if f == 0 {
		return 0
	}
	tw := xui.TextWidth(s.text, f)
	if tw <= 0 {
		return 0
	}
	cx := (p.Rc.Left + p.Rc.Right) / 2
	rc := xc.RECT{
		Left:   cx - tw/2 - fieldSepPadX,
		Top:    p.Rc.Top,
		Right:  cx + tw/2 + fieldSepPadX,
		Bottom: p.Rc.Bottom,
	}
	// 用 Card 而不是 Background 压线: 组件按"放在 Card 上"设计(同 Badge/Checkbox),
	// 暗色主题下 background(#0a0a0a) 与 card(#171717) 不同, 用 Background 会在卡片上
	// 留下一块明显更暗的矩形(实测)。
	p.FillRound(&rc, t.Card, 0)
	p.Draw.SetFont(f)
	p.Text(s.text, &rc, t.MutedForeground,
		xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	return 0
}

// FieldError 字段错误文本区: 一条或多条错误, 默认隐藏, SetErrors 后显示。
//
// 与 shadcn field.tsx 的 FieldError 对齐: text-sm destructive; 多条错误去重、
// 左缩进 16 并用 "•" 作列表标记。上游按错误文案折行; 这里的单条错误保持单行
// (超出省略号), 需要多行时用返回的 Labels 自行加高。
type FieldError struct {
	*widget.LayoutEle
	labels []*Label
}

// NewFieldError 创建错误区。messages 为空时保持隐藏。
func NewFieldError(x, y, cx, cy int32, hParent int, messages ...string) *FieldError {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	e := &FieldError{LayoutEle: ele}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(fieldErrorGap)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.AttachWindow(ele.Handle)
	e.SetErrors(messages...)
	return e
}

// SetErrors 更新错误列表(自动去重、空列表隐藏整块), 返回自身便于链式调用。
func (e *FieldError) SetErrors(messages ...string) *FieldError {
	if e == nil {
		return e
	}
	seen := make(map[string]struct{}, len(messages))
	msgs := make([]string, 0, len(messages))
	for _, m := range messages {
		if m == "" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		msgs = append(msgs, m)
	}
	for len(e.labels) < len(msgs) {
		l := NewLabel(0, 0, 0, fieldTextH, "", e.Handle)
		l.SetColorFunc(func(t *xui.Theme) xui.Color { return t.Destructive })
		xui.ItemFill(l)
		e.labels = append(e.labels, l)
	}
	for i, l := range e.labels {
		if i >= len(msgs) {
			l.Show(false)
			continue
		}
		text := msgs[i]
		if len(msgs) > 1 {
			text = "•  " + text
			xui.ItemMarginLeft(l, fieldErrorMl)
		} else {
			xui.ItemMarginLeft(l, 0)
		}
		l.SetText(text)
		l.Show(true)
	}
	e.Show(len(msgs) > 0)
	// 隐藏时也要重排: FieldError 所在 Field/卡片是 Auto 高, 只 Show(false) 不会让
	// 祖先收回错误区腾出的高度(实测会留下与错误区等高的空洞), 必须逐级 AdjustLayout。
	adjustFieldAncestors(e.Handle)
	e.Redraw(false)
	return e
}

// Labels 返回错误文本 Label 列表(尚未创建的不存在), 供自定义字号/高度。
func (e *FieldError) Labels() []*Label {
	if e == nil {
		return nil
	}
	return e.labels
}

// adjustFieldAncestors 从元素开始逐级重排祖先, 让 Auto 高的 Field/卡片跟随
// 可见性变化重新量高(同 Alert 的量高重排链路, 见 DESIGN 第八节 90/97)。
func adjustFieldAncestors(hEle int) {
	xc.XEle_AdjustLayout(hEle)
	for p := xc.XWidget_GetParentEle(hEle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			// ScrollArea 的框架自己会重排内容区并重算滚动范围。
			xc.XEle_AdjustLayout(p)
			break
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			// p 是框架的内容区: 对内容区 AdjustLayout 会把宽度打成 0, 交给框架排。
			xc.XEle_AdjustLayout(pp)
			break
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_ELE) {
			xc.XEle_AdjustLayout(p)
			continue
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
		}
		break
	}
	// 整窗重绘: 元素被重排后旧位置的像素不会被自动擦除, 只重绘错误区会留下
	// "文字重影"(错误区隐藏后, 下面字段上移, 旧位置仍留着标题/说明, 实测)。
	// 与 Alert 的量高重排同口径(见 DESIGN 第八节 90), 必须在一轮布局都结束后做。
	if hwnd := xc.XWidget_GetHWND(hEle); hwnd != 0 {
		if hw := xc.XC_hWindowFromHWnd(hwnd); hw != 0 {
			xc.XWnd_Redraw(hw, false)
		}
	}
}
