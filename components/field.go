package ui

import (
	"strings"

	"github.com/twgh/xcgui/app"
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

	// fieldResponsiveBreakpoint 是响应式取向的切换宽度, 取上游 `@md/field-group`
	// 容器查询的 28rem(= 448px)。按字段自身宽度判定: 字段在组里满宽, 与上游按
	// FieldGroup 量宽等价。
	fieldResponsiveBreakpoint int32 = 448

	// fieldRequiredMessage 是必填字段的默认错误文案(可用 WithFieldRequiredMessage /
	// SetRequiredMessage 覆盖)。取上游 Input#Required 示例的提示语。
	fieldRequiredMessage = "This field is required."
)

// FieldOrientation 字段取向, 对应 shadcn field.tsx 的 orientation。
type FieldOrientation int

const (
	FieldVertical   FieldOrientation = iota // 竖排: 标签在上、控件在下
	FieldHorizontal                         // 横排: 标签在左、控件在右, 垂直居中
	// FieldResponsive 响应式: 宽度 >= 448px(上游 @md 容器查询)时横排, 不足时竖排;
	// 窗口/卡片尺寸变化跨过断点时自动切换。运行时也可用 SetOrientation 手动切。
	FieldResponsive
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

// WithFieldDisabled 初始即禁用整个字段: 标签/说明转 50%, 已 Attach 的控件同步禁用。
func WithFieldDisabled() FieldOption {
	return func(f *Field) { f.disabled = true }
}

// WithFieldFocusProxy 声明键盘焦点环由外层包裹元素(选择卡片 / 自绘 FieldLabel)代理:
// Field 会把该标记传给实现了 SetFocusRingProxy 的控件, 控件不再画自己的环。
func WithFieldFocusProxy() FieldOption {
	return func(f *Field) { f.labelProxy = true }
}

// WithFieldRequired 标记字段为必填: 标签后追加破坏性色 "*"(同上游 Input#Required),
// Validate 时值为空即失败并写入错误区。
func WithFieldRequired() FieldOption {
	return func(f *Field) { f.required = true }
}

// WithFieldRequiredMessage 覆盖必填失败时的错误文案(默认 "This field is required.")。
// 通常与 WithFieldRequired 一起用。
func WithFieldRequiredMessage(msg string) FieldOption {
	return func(f *Field) { f.requiredMsg = msg }
}

// WithFieldValidator 追加字段级校验器: 收到当前值, 返回 error 时把 Error() 文案写进
// 错误区。校验器总会收到当前值(必填为空时值是空串), 需要"可空"的规则自行放行。
func WithFieldValidator(v FieldValidator) FieldOption {
	return func(f *Field) {
		if v != nil {
			f.validators = append(f.validators, v)
		}
	}
}

// FieldControl 是可被 Field 级联校验失败态/禁用态的控件。
// 内置控件均已实现(Checkbox / Radio / RadioGroup / Switch / Input / Textarea /
// NativeComboBox / NativeSelect / Select / Combobox / Slider / InputGroup / InputOTP /
// TagInput / Toggle / Button / DatePicker / TimePicker); 自定义控件实现这两个方法
// 也能用 Field.Attach 纳入级联。
type FieldControl interface {
	// SetFieldState 由 Field 调用: 同时写入校验失败态与禁用态。
	SetFieldState(invalid, disabled bool)
	// FieldDisabled 报告控件当前是否禁用, Field 重新启用时据此恢复
	// "本来就单独禁用"的控件, 而不是一律打开。
	FieldDisabled() bool
}

// fieldFocusProxy 是可选能力: 控件支持把键盘焦点环交给外层代理。
type fieldFocusProxy interface {
	SetFocusRingProxy(bool)
}

// FieldValueProvider 是可选能力: 控件向 Field 报告当前值的字符串形式, 供必填校验与
// FieldValidator 读取; 空串表示未填写。内置的可取值控件均已实现(Input / Textarea /
// Select / Combobox / NativeComboBox / NativeSelect / RadioGroup / Radio / Checkbox /
// Switch / CheckboxGroup / InputGroup / InputOTP / TagInput / DatePicker / TimePicker /
// Toggle / ToggleGroup / Slider); 自定义控件实现本接口即可参与 Field 校验。
type FieldValueProvider interface {
	FieldValue() string
}

// FieldValidator 校验字段当前值: 返回 nil 通过; 返回 error 时把 Error() 文案写进错误区。
type FieldValidator func(value string) error

// Field 表单字段容器: 把标签、控件、说明、错误按 12px 间距排成一组。
//
// 与 shadcn field.tsx 的 Field 对齐:
//   - 默认竖排, 子项满宽; FieldHorizontal 时横排、垂直居中;
//   - 标签/标题 font-medium; 说明 muted; 校验失败时标签/标题转 destructive
//     (说明仍保持 muted, 同上游 text-muted-foreground 覆盖继承色);
//   - 禁用时标签/说明整体 50%, 并用 Attach 把状态级联给控件(对应 data-disabled);
//   - 不强制子项宽度: 竖排请对控件/说明用 `xui.ItemFill`, 横排请用 `xui.ItemWidth`
//     或让控件按自身宽度排(Field.Label 在横排下自动吃剩余宽度)。
//
// 标签/说明通过 Field 的方法创建, 这样 SetInvalid / SetDisabled 能统一改色:
//
//	f := ui.NewField(0, 0, 0, 0, stack.Handle)
//	xui.ItemFill(f)
//	f.Label("邮箱地址")
//	input := ui.NewInput(0, 0, 0, 0, f.Handle, ui.WithPlaceholder("m@example.com"))
//	xui.ItemFill(input)
//	f.Attach(input) // 校验/禁用状态与标签联动
//	f.Description("我们只会用它发送登录链接。")
type Field struct {
	*widget.LayoutEle
	orientation FieldOrientation
	// horizontal 是当前**实际**取向: FieldResponsive 按宽度解析后的结果,
	// 标签/内容列的布局项宽度也按它设置。
	horizontal    bool
	resizePending bool
	invalid       bool
	disabled      bool
	labelProxy    bool
	required      bool
	requiredMsg   string
	validators    []FieldValidator
	lastErrs      []string
	labels        []*Label
	texts         []*Label
	content       *widget.LayoutEle
	controls      []FieldControl
	errors        []*FieldError
	origDisabled  map[FieldControl]bool
	// labelRows / stars 把标签行与必填星号关联到主标签: 标签行承载布局宽度,
	// 主标签只按文字自身宽度排版, 星号跟在文字后面(同上游 label + destructive span)。
	labelRows map[*Label]*widget.LayoutEle
	stars     map[*Label]*Label
	// fillControls 记录"切横排时被收成固定宽"的直属控件, 切回竖排要恢复 Fill。
	fillControls map[int]bool
}

// fieldReg 记录"字段句柄 → Field", 供 FieldError 沿父链找回所属字段。
// 与 tooltipReg 同一种用法: 只在 UI 线程读写, 元素销毁时摘除。
var fieldReg = map[int]*Field{}

// findFieldOf 沿父链找最近的字段容器(没有则返回 nil)。
func findFieldOf(hEle int) *Field {
	for h := hEle; h != 0; h = xc.XWidget_GetParentEle(h) {
		if f, ok := fieldReg[h]; ok {
			return f
		}
	}
	return nil
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
	fieldReg[ele.Handle] = f
	ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		delete(fieldReg, hEle)
		return 0
	})
	// 响应式取向要跟着字段宽走: 尺寸事件里只做判定/推迟, AfterFunc 里再改布局,
	// 避免重入布局。非响应式字段在这里直接返回。
	ele.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		f.scheduleResponsive()
		return 0
	})
	return f
}

// applyOrientation 按配置取向初始化实际取向(创建期调用一次)。
func (f *Field) applyOrientation() {
	horizontal := f.orientation == FieldHorizontal
	if f.orientation == FieldResponsive {
		horizontal = f.GetWidth() >= fieldResponsiveBreakpoint
	}
	f.setHorizontal(horizontal)
}

// setHorizontal 应用实际取向, 并同步已创建子项的布局项宽度:
// 竖排标签满宽、横排标签吃剩余宽度; Content 横排固定 240(嵌套 Fill 会塌成 0,
// 见 DESIGN 第八节 101)、竖排满宽。
func (f *Field) setHorizontal(horizontal bool) {
	f.horizontal = horizontal
	if horizontal {
		f.EnableHorizon(true)
		f.SetAlignV(xcc.Layout_Align_Center)
	} else {
		f.EnableHorizon(false)
		f.SetAlignH(xcc.Layout_Align_Left)
	}
	for _, l := range f.texts {
		f.setChildWidth(l)
	}
	if f.content != nil {
		if horizontal {
			f.content.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, fieldContentW)
		} else {
			f.content.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		}
	}
	if horizontal {
		f.narrowFillControls()
	} else {
		f.restoreFillControls()
	}
}

// narrowFillControls 在切到横排时把直属控件(占满宽、高度不是 Auto)收成内容列同宽:
// 竖排时它们是 Fill(满宽), 横排下 Fill 会独占整行、把标签挤到下一行(见 DESIGN 8.55),
// 对应上游 responsive 的 `[&>*]:w-auto`。错误区/自定义容器是 Auto 高, 保持原样;
// Field 自己创建的标签与 Content 另行处理。
func (f *Field) narrowFillControls() {
	skip := make(map[int]struct{}, len(f.texts)+1)
	for _, l := range f.texts {
		if l != nil {
			skip[l.Handle] = struct{}{}
		}
	}
	if f.content != nil {
		skip[f.content.Handle] = struct{}{}
	}
	if f.fillControls == nil {
		f.fillControls = map[int]bool{}
	}
	n := xc.XEle_GetChildCount(f.Handle)
	for i := int32(0); i < n; i++ {
		h := xc.XEle_GetChildByIndex(f.Handle, i)
		if h == 0 || f.fillControls[h] {
			continue
		}
		if _, ok := skip[h]; ok {
			continue
		}
		var wt xcc.Layout_Size_
		var w int32
		xc.XWidget_LayoutItem_GetWidth(h, &wt, &w)
		if wt != xcc.Layout_Size_Fill {
			continue
		}
		var ht xcc.Layout_Size_
		var hh int32
		xc.XWidget_LayoutItem_GetHeight(h, &ht, &hh)
		if ht == xcc.Layout_Size_Auto {
			continue // 错误区/自定义容器: Auto 高, 宽度由自己管
		}
		if e := widget.NewElementByHandle(h); e != nil {
			xui.ItemWidth(e, fieldContentW)
			f.fillControls[h] = true
		}
	}
}

// restoreFillControls 切回竖排时把 narrowFillControls 收窄过的控件恢复满宽。
// 期间被调用方改成别的固定宽(不是 240)的不动, 视为调用方自己的覆盖。
func (f *Field) restoreFillControls() {
	for h := range f.fillControls {
		if h == 0 || !xc.XC_IsHXCGUI(h, xcc.XC_ELE) {
			continue
		}
		var wt xcc.Layout_Size_
		var w int32
		xc.XWidget_LayoutItem_GetWidth(h, &wt, &w)
		if wt == xcc.Layout_Size_Fixed && w == fieldContentW {
			if e := widget.NewElementByHandle(h); e != nil {
				xui.ItemFill(e)
			}
		}
	}
	f.fillControls = nil
}

// scheduleResponsive 在字段宽度变化时按断点解析 FieldResponsive 的实际取向。
func (f *Field) scheduleResponsive() {
	if f == nil || f.orientation != FieldResponsive || f.resizePending {
		return
	}
	if (f.GetWidth() >= fieldResponsiveBreakpoint) == f.horizontal {
		return
	}
	f.resizePending = true
	app.AfterFunc(0, func() {
		f.resizePending = false
		if f.Handle == 0 || !xc.XC_IsHXCGUI(f.Handle, xcc.XC_ELE) || f.orientation != FieldResponsive {
			return
		}
		horizontal := f.GetWidth() >= fieldResponsiveBreakpoint
		if horizontal == f.horizontal {
			return
		}
		f.setHorizontal(horizontal)
		// 取向变化会改变字段高度: 逐级通知 Auto 高的祖先重排并整窗重绘擦旧像素。
		adjustFieldAncestors(f.Handle)
	})
}

// Orientation 返回配置的取向(FieldResponsive 时返回 FieldResponsive, 实际横竖见 Horizontal)。
func (f *Field) Orientation() FieldOrientation {
	if f == nil {
		return FieldVertical
	}
	return f.orientation
}

// Horizontal 报告字段当前的实际取向(FieldResponsive 按宽度解析后的结果)。
func (f *Field) Horizontal() bool {
	return f != nil && f.horizontal
}

// SetOrientation 运行时切换取向。FieldResponsive 会按当前宽度立即解析一次,
// 之后随字段宽度变化自动切换。切换会同步已创建标签/内容列的布局项宽度并重排。
func (f *Field) SetOrientation(o FieldOrientation) *Field {
	if f == nil || f.orientation == o {
		return f
	}
	f.orientation = o
	switch o {
	case FieldHorizontal:
		f.setHorizontal(true)
	case FieldResponsive:
		f.setHorizontal(f.GetWidth() >= fieldResponsiveBreakpoint)
	default:
		f.setHorizontal(false)
	}
	adjustFieldAncestors(f.Handle)
	return f
}

// Label 创建字段标签(text-sm, font-medium), 并纳入 SetInvalid 的统一改色。
// 竖排下标签行满宽; 横排下按权重吃剩余宽度(对应上游的 flex-auto)。返回的标签自身
// 保持文字宽度(必填星号紧跟其后), 布局宽度由字段的标签行承担, 一般不要再 ItemFill。
func (f *Field) Label(text string) *Label {
	return f.newText(text)
}

// Title 创建选项标题(用于复选框/单选框行), 样式与 Label 相同。
// 上游 FieldTitle 与 FieldLabel 的区别在 DOM 语义(label/div), Go 侧没有对应物。
func (f *Field) Title(text string) *Label {
	return f.newText(text)
}

// Description 创建字段说明(text-sm, muted)。多条说明按 Field 的 gap 堆叠。
// 说明文字较长时对返回的 Label 调用 SetAutoWrap(), 高度会随折行自撑。
func (f *Field) Description(text string) *Label {
	l := NewLabel(0, 0, 0, fieldTextH, text, f.Handle).SetMuted().SetWrap()
	f.setChildWidth(l)
	f.addText(l)
	return l
}

// Attach 把控件纳入字段状态联动: SetInvalid / SetDisabled 会级联到它。
// 支持 FieldControl 的内置控件(见接口注释); 重复 Attach 同一个控件只记一次。
func (f *Field) Attach(controls ...FieldControl) *Field {
	if f == nil {
		return f
	}
	for _, c := range controls {
		if c == nil || f.hasControl(c) {
			continue
		}
		f.controls = append(f.controls, c)
		if f.disabled {
			if f.origDisabled == nil {
				f.origDisabled = make(map[FieldControl]bool, len(f.controls))
			}
			f.origDisabled[c] = c.FieldDisabled()
			c.SetFieldState(f.invalid, true)
		} else {
			c.SetFieldState(f.invalid, c.FieldDisabled())
		}
		if f.labelProxy {
			if fp, ok := c.(fieldFocusProxy); ok {
				fp.SetFocusRingProxy(true)
			}
		}
	}
	return f
}

func (f *Field) hasControl(c FieldControl) bool {
	for _, it := range f.controls {
		if it == c {
			return true
		}
	}
	return false
}

// SetDisabled 禁用/启用整个字段: 标签/说明转 50%, 已 Attach 的控件同步禁用。
// 重新启用时保留控件自己原本的禁用状态(单独禁用的不会被字段级启用误开)。
func (f *Field) SetDisabled(v bool) *Field {
	if f == nil || f.disabled == v {
		return f
	}
	f.disabled = v
	if v {
		f.origDisabled = make(map[FieldControl]bool, len(f.controls))
		for _, c := range f.controls {
			f.origDisabled[c] = c.FieldDisabled()
			c.SetFieldState(f.invalid, true)
		}
	} else {
		for _, c := range f.controls {
			c.SetFieldState(f.invalid, f.origDisabled[c])
		}
		f.origDisabled = nil
	}
	f.applyTextEnabled()
	f.Redraw(false)
	return f
}

// Disabled 返回字段是否禁用。
func (f *Field) Disabled() bool { return f != nil && f.disabled }

// SetFocusRingProxy 声明键盘焦点环由外层包裹元素代理(见 WithFieldFocusProxy)。
// 只影响实现了 SetFocusRingProxy 的控件(Checkbox / Radio / Group / Switch)。
func (f *Field) SetFocusRingProxy(v bool) *Field {
	if f == nil || f.labelProxy == v {
		return f
	}
	f.labelProxy = v
	for _, c := range f.controls {
		if fp, ok := c.(fieldFocusProxy); ok {
			fp.SetFocusRingProxy(v)
		}
	}
	return f
}

// FocusRingProxy 返回是否由外层代理键盘焦点环。
func (f *Field) FocusRingProxy() bool { return f != nil && f.labelProxy }

// SetRequired 运行时切换必填标记: 标签后的 "*" 同步显隐, 后续 Validate 按必填判定。
func (f *Field) SetRequired(v bool) *Field {
	if f == nil || f.required == v {
		return f
	}
	f.required = v
	for _, l := range f.labels {
		if s := f.stars[l]; s != nil {
			s.Show(v)
			s.Enable(!f.disabled)
		}
	}
	if len(f.labels) > 0 {
		adjustFieldAncestors(f.Handle)
	}
	return f
}

// Required 返回字段是否必填。
func (f *Field) Required() bool { return f != nil && f.required }

// SetRequiredMessage 覆盖必填失败时的错误文案。
func (f *Field) SetRequiredMessage(msg string) *Field {
	if f != nil {
		f.requiredMsg = msg
	}
	return f
}

// AddValidator 追加字段级校验器(见 WithFieldValidator), 返回自身便于链式调用。
func (f *Field) AddValidator(v FieldValidator) *Field {
	if f != nil && v != nil {
		f.validators = append(f.validators, v)
	}
	return f
}

// Value 返回第一个可报告值的已接控件的当前值(FieldValueProvider), 没有时为空串。
func (f *Field) Value() string {
	if f == nil {
		return ""
	}
	for _, c := range f.controls {
		if p, ok := c.(FieldValueProvider); ok {
			return p.FieldValue()
		}
	}
	return ""
}

// Validate 校验必填与全部 FieldValidator: 失败文案写进已 AttachError 的错误区并置
// invalid, 通过时清空错误并恢复配色; 返回是否通过(禁用字段跳过校验, 恒为 true)。
// 多条失败会一次收集(必填 + 各校验器), 由错误区按列表展示。
func (f *Field) Validate() bool {
	if f == nil {
		return true
	}
	if f.disabled {
		return true
	}
	value := f.Value()
	msgs := fieldValidationMessages(value, f.required, f.requiredMsg, f.validators)
	f.lastErrs = msgs
	f.SetErrors(msgs...)
	return len(msgs) == 0
}

// fieldValidationMessages 收集必填与校验器的错误文案(按顺序, 供 Validate 与测试复用)。
// 值为空白视为未填写; 校验器即使值为空也会调用, 需要可空规则的校验器自行放行。
func fieldValidationMessages(value string, required bool, requiredMsg string, validators []FieldValidator) []string {
	var msgs []string
	if required && strings.TrimSpace(value) == "" {
		if requiredMsg == "" {
			requiredMsg = fieldRequiredMessage
		}
		msgs = append(msgs, requiredMsg)
	}
	for _, v := range validators {
		if v == nil {
			continue
		}
		if err := v(value); err != nil && err.Error() != "" {
			msgs = append(msgs, err.Error())
		}
	}
	return msgs
}

// ValidationErrors 返回最近一次 Validate 产生的错误文案(未接入错误区也能取到)。
func (f *Field) ValidationErrors() []string {
	if f == nil {
		return nil
	}
	return append([]string(nil), f.lastErrs...)
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
		if f.horizontal {
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

// SetInvalid 切换校验失败态(标签/标题转 destructive), 并级联到已 Attach 的控件。
func (f *Field) SetInvalid(invalid bool) *Field {
	if f == nil || f.invalid == invalid {
		return f
	}
	f.invalid = invalid
	for _, l := range f.labels {
		f.applyLabelColor(l)
	}
	for _, c := range f.controls {
		disabled := f.disabled || c.FieldDisabled()
		c.SetFieldState(invalid, disabled)
	}
	f.Redraw(false)
	return f
}

// Invalid 返回当前校验失败态。
func (f *Field) Invalid() bool {
	return f != nil && f.invalid
}

// AttachError 把错误文本区接进字段(对应上游 FieldError 与 data-invalid 同源):
// 之后 Field.SetErrors 会同时更新这些区域, 反过来它们自己 SetErrors 也会把字段切到
// 校验失败态。错误区可以放在字段直属下, 也可以放进 Content() 里(沿父链能找到字段)。
func (f *Field) AttachError(errs ...*FieldError) *Field {
	if f == nil {
		return f
	}
	for _, e := range errs {
		if e == nil || f.hasError(e) {
			continue
		}
		f.errors = append(f.errors, e)
		if e.HasErrors() {
			f.SetInvalid(true)
		}
	}
	return f
}

func (f *Field) hasError(e *FieldError) bool {
	for _, it := range f.errors {
		if it == e {
			return true
		}
	}
	return false
}

// SetErrors 设置字段错误: 文案同步到已 AttachError 的错误区, 并按"有没有错误"切换
// 校验失败态(空 = 清除错误、标签与已接控件恢复正常配色)。多条错误由错误区负责去重
// 与列表化。
func (f *Field) SetErrors(messages ...string) *Field {
	if f == nil {
		return f
	}
	n := 0
	for _, m := range messages {
		if m != "" {
			n++
		}
	}
	for _, e := range f.errors {
		e.SetErrors(messages...)
	}
	f.SetInvalid(n > 0)
	return f
}

// Errors 返回已接的错误文本区(未 AttachError 时为空)。
func (f *Field) Errors() []*FieldError {
	if f == nil {
		return nil
	}
	return f.errors
}

func (f *Field) newText(text string) *Label {
	// 标签放在横向行里: 主标签按文字宽度 + 紧跟破坏性色 "*"(必填时显示)。
	// 行本身承担布局宽度(竖排 Fill / 横排 Weight), 主标签保持自身宽度, 星号才能
	// 贴在文字后面而不是被推到行尾。
	row := widget.NewLayoutEle(0, 0, 0, 0, f.Handle)
	if row != nil {
		row.EnableHorizon(true)
		row.EnableLayout(true)
		row.EnableBkTransparent(true)
		row.EnableAutoWrap(false)
		row.SetSpace(4)
		row.SetAlignV(xcc.Layout_Align_Center)
		row.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	}
	parent := f.Handle
	if row != nil {
		parent = row.Handle
	}
	l := NewLabel(0, 0, 0, fieldTextH, text, parent).SetMedium()
	// 原生 Button 在布局父级里的默认宽是 0(实测), 必须显式 Auto 才会按文字自撑:
	// 行内两个标签都保持自身宽, 星号才能紧跟文字。
	l.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	if row != nil {
		if f.labelRows == nil {
			f.labelRows = map[*Label]*widget.LayoutEle{}
		}
		f.labelRows[l] = row
		s := NewLabel(0, 0, 0, fieldTextH, "*", row.Handle)
		s.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
		s.SetColorFunc(func(t *xui.Theme) xui.Color { return t.Destructive })
		s.Show(f.required)
		if f.stars == nil {
			f.stars = map[*Label]*Label{}
		}
		f.stars[l] = s
	}
	f.setChildWidth(l)
	f.labels = append(f.labels, l)
	f.applyLabelColor(l)
	f.addText(l)
	return l
}

// addText 记录 Field 自己创建的文本, 供 SetDisabled 统一调暗。
func (f *Field) addText(l *Label) {
	f.texts = append(f.texts, l)
	l.Enable(!f.disabled)
	if s := f.stars[l]; s != nil {
		s.Enable(!f.disabled)
	}
}

// applyTextEnabled 把字段禁用态同步到标签/标题/说明/必填星号。
// Label 的禁用是"字色与卡片底色预混 50%", 与 disabled:opacity-50 的观感一致。
func (f *Field) applyTextEnabled() {
	for _, l := range f.texts {
		l.Enable(!f.disabled)
		l.Redraw(false)
	}
	for _, s := range f.stars {
		s.Enable(!f.disabled)
		s.Redraw(false)
	}
}

func (f *Field) setChildWidth(l *Label) {
	if l == nil {
		return
	}
	var item xui.Item = l
	if row := f.labelRows[l]; row != nil {
		item = row
	}
	if f.horizontal {
		xui.ItemWeight(item, 1)
	} else {
		xui.ItemFill(item)
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
//
// 除布局外还提供组级校验入口: Fields 按视觉顺序列出组内全部字段(含嵌套容器里的),
// Validate 一次校验并让每个字段各自展示错误, Submit 校验通过后再执行提交回调。
type FieldGroup struct {
	*widget.LayoutEle
}

// NewFieldGroup 创建字段组。
func NewFieldGroup(x, y, cx, cy int32, hParent int) *FieldGroup {
	l := newFieldStack(hParent, fieldGroupGap)
	if l == nil {
		return nil
	}
	return &FieldGroup{LayoutEle: l}
}

// Fields 按视觉顺序(深度优先)返回组内全部字段, 包含放在 Content / 普通容器里的字段;
// 嵌套 FieldGroup 的字段也包含在内。
func (g *FieldGroup) Fields() []*Field {
	if g == nil {
		return nil
	}
	var out []*Field
	var walk func(hEle int)
	walk = func(hEle int) {
		if hEle == 0 {
			return
		}
		if f, ok := fieldReg[hEle]; ok {
			out = append(out, f)
		}
		n := int(xc.XEle_GetChildCount(hEle))
		for i := 0; i < n; i++ {
			walk(xc.XEle_GetChildByIndex(hEle, int32(i)))
		}
	}
	walk(g.Handle)
	return out
}

// Validate 校验组内全部字段, 返回是否全部通过。每个字段都会得到自己的错误文案,
// 不会在第一个失败处停下(提交时一次看到所有问题)。
func (g *FieldGroup) Validate() bool {
	ok := true
	for _, f := range g.Fields() {
		if !f.Validate() {
			ok = false
		}
	}
	return ok
}

// Submit 校验通过后执行 fn(对应表单提交动作), 返回校验是否通过;
// 校验失败时不执行 fn, 各字段的错误区已更新。
func (g *FieldGroup) Submit(fn func()) bool {
	if !g.Validate() {
		return false
	}
	if fn != nil {
		fn()
	}
	return true
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
// 左缩进 16 并用 "•" 作列表标记。默认单条错误保持单行(超出省略号); 需要多行时
// 对 Labels() 返回的 Label 调用 SetAutoWrap(), 高度会随折行自撑。
type FieldError struct {
	*widget.LayoutEle
	labels []*Label
	msgs   []string // 去重后的当前错误文案(空 = 隐藏)
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
	e.msgs = msgs
	e.Show(len(msgs) > 0)
	// 隐藏时也要重排: FieldError 所在 Field/卡片是 Auto 高, 只 Show(false) 不会让
	// 祖先收回错误区腾出的高度(实测会留下与错误区等高的空洞), 必须逐级 AdjustLayout。
	adjustFieldAncestors(e.Handle)
	e.Redraw(false)
	// 所在的字段跟着切校验态(上游 FieldError 与 data-invalid 同源, 不需要调用方各写一遍)
	if f := findFieldOf(e.Handle); f != nil {
		f.SetInvalid(len(msgs) > 0)
	}
	return e
}

// HasErrors 报告当前是否挂着错误文案。
func (e *FieldError) HasErrors() bool {
	return e != nil && len(e.msgs) > 0
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
