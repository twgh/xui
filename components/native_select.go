package ui

import "github.com/twgh/xcgui/xc"

// NativeSelect 原生下拉选择框, 对应 shadcn native-select.tsx。
//
// 上游把浏览器原生 <select> 包上 shadcn 外壳: 触发区样式与 Select 一致
// (border-input、focus-visible 环、aria-invalid、sm 尺寸), 弹层完全是原生的;
// value 语义按 <option value> / onChange(e.target.value) 走字符串。
//
// xui 侧承载"原生弹层"的就是 ComboBox(触发区自绘 + 原生弹窗, 见 combobox.go),
// 所以本组件在其上做 value 字符串封装 —— 视觉/行为与 NewComboBox 完全一致,
// 用法更贴近 shadcn 的受控 select。要富条目(图标/禁用/分组)用 Select。

// NativeSelectOption 原生下拉框配置项。
type NativeSelectOption func(o *nativeSelectOpts)

type nativeSelectOpts struct {
	placeholder string
	size        ComboBoxSize
	sizeSet     bool
	invalid     bool
	value       string
}

// WithNativeSelectPlaceholder 设置未选择时的占位文本。
func WithNativeSelectPlaceholder(s string) NativeSelectOption {
	return func(o *nativeSelectOpts) { o.placeholder = s }
}

// WithNativeSelectSize 设置触发区尺寸(sm = h-7 = 28, 同 native-select 的 data-size)。
func WithNativeSelectSize(s ComboBoxSize) NativeSelectOption {
	return func(o *nativeSelectOpts) { o.size, o.sizeSet = s, true }
}

// WithNativeSelectInvalid 设置校验失败态(aria-invalid: 红边框 + 红环)。
func WithNativeSelectInvalid() NativeSelectOption {
	return func(o *nativeSelectOpts) { o.invalid = true }
}

// WithNativeSelectValue 设置初始选中值(按文本匹配 option, 同 defaultValue)。
func WithNativeSelectValue(v string) NativeSelectOption {
	return func(o *nativeSelectOpts) { o.value = v }
}

// NativeSelect 原生下拉框(ComboBox 的 value 语义封装)。
type NativeSelect struct {
	*ComboBox
}

// NewNativeSelect 创建原生下拉框。values 即 <option> 列表; 语义对齐:
//
//	ns := ui.NewNativeSelect(0, 0, 0, 0, parent, []string{"开发环境", "生产环境"})
//	ns.SetValue("生产环境")
//	ns.OnValueChange(func(v string) { ... })
func NewNativeSelect(x, y, cx, cy int32, hParent int, values []string, opts ...NativeSelectOption) *NativeSelect {
	var o nativeSelectOpts
	for _, fn := range opts {
		fn(&o)
	}
	copts := make([]ComboBoxOption, 0, 3)
	if o.placeholder != "" {
		copts = append(copts, WithComboPlaceholder(o.placeholder))
	}
	if o.sizeSet {
		copts = append(copts, WithComboSize(o.size))
	}
	if o.invalid {
		copts = append(copts, WithComboInvalid())
	}
	c := NewComboBox(x, y, cx, cy, hParent, values, copts...)
	if c == nil {
		return nil
	}
	n := &NativeSelect{ComboBox: c}
	if o.value != "" {
		n.SetValue(o.value)
	}
	return n
}

// Value 返回当前选中文本(同 e.target.value), 未选中返回空串。
func (n *NativeSelect) Value() string {
	if n == nil || n.ComboBox == nil {
		return ""
	}
	i := n.Selected()
	if i < 0 {
		return ""
	}
	return n.ItemText(i)
}

// SetValue 按文本选中 option; 没有匹配项时取消选中(同 value 不存在的 <select>)。
func (n *NativeSelect) SetValue(v string) *NativeSelect {
	if n == nil || n.ComboBox == nil {
		return n
	}
	for i := int32(0); i < xc.XComboBox_GetCount(n.Handle); i++ {
		if n.ItemText(i) == v {
			n.SetSelected(i)
			return n
		}
	}
	n.SetSelected(-1)
	return n
}

// OnValueChange 注册选中值变化回调(同 onChange(e.target.value); 未选中时值为空串)。
func (n *NativeSelect) OnValueChange(fn func(value string)) *NativeSelect {
	if n == nil || n.ComboBox == nil || fn == nil {
		return n
	}
	n.OnChange(func(index int32) {
		v := ""
		if index >= 0 {
			v = n.ItemText(index)
		}
		fn(v)
	})
	return n
}
