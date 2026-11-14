package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// checkboxGroupGap 是 CheckboxGroup 的默认条目间距(gap-2 = 8)。
const checkboxGroupGap int32 = 8

// CheckboxGroup 复选组: 竖向流容器 + 一组 Checkbox, 对应 Base UI 的 Checkbox Group
// 与 shadcn checkbox 文档的 Group / Table 示例。
//
// 与散装 Checkbox 的区别:
//   - 组值语义: Values / SetValues / OnChange 用字符串值(对应 Base UI 的 group value);
//   - 父项联动: SetParent 绑定一个 Checkbox, 子项全选 → 父项选中、部分选中 → 父项
//     不确定(减号)、全不选 → 未选; 点父项全选/全不选(跳过单独禁用的子项);
//   - 跨层成员: 条目可以放在 Field 等子容器里(Attach), 不只依赖同父容器;
//   - 组级禁用/校验态: SetDisabled / SetInvalid 作用于所有条目与父项。
//
// 用法:
//
//	grp := ui.NewCheckboxGroup(card.Handle, ui.WithCheckboxGroupValues("disks"))
//	ui.ItemFill(grp)
//	all := ui.NewCheckbox(0, 0, 0, 0, "全选", card.Handle)
//	grp.SetParent(all)
//	grp.Add("disks", "硬盘")
//	grp.Add("external", "外置磁盘")
//	grp.OnChange(func(values []string) { ... })
type CheckboxGroup struct {
	*widget.LayoutEle
	items          []*Checkbox
	values         map[*Checkbox]string
	initial        map[string]bool
	spacing        int32
	disabled       bool
	invalid        bool
	focusRingProxy bool
	disabledOwn    map[*Checkbox]bool
	parent         *Checkbox
	applying       bool
	cb             func(values []string)
}

// CheckboxGroupOption 复选组配置项。
type CheckboxGroupOption func(*CheckboxGroup)

// WithCheckboxGroupValues 设置初始选中值(对应 Base UI 的默认 group value)。
func WithCheckboxGroupValues(values ...string) CheckboxGroupOption {
	return func(g *CheckboxGroup) {
		g.initial = make(map[string]bool, len(values))
		for _, v := range values {
			g.initial[v] = true
		}
	}
}

// WithCheckboxGroupDisabled 让整组初始为禁用态。
func WithCheckboxGroupDisabled() CheckboxGroupOption {
	return func(g *CheckboxGroup) { g.disabled = true }
}

// WithCheckboxGroupInvalid 让整组初始为校验失败态(每个条目与父项画红环)。
func WithCheckboxGroupInvalid() CheckboxGroupOption {
	return func(g *CheckboxGroup) { g.invalid = true }
}

// WithCheckboxGroupSpacing 设置条目间距(逻辑像素; 默认 8 = gap-2)。
func WithCheckboxGroupSpacing(px int32) CheckboxGroupOption {
	return func(g *CheckboxGroup) {
		if px < 0 {
			px = 0
		}
		g.spacing = px
	}
}

// NewCheckboxGroup 创建复选组。组宽铺满、高自撑, 放进卡片内容区即可。
func NewCheckboxGroup(hParent int, opts ...CheckboxGroupOption) *CheckboxGroup {
	g := &CheckboxGroup{
		spacing: checkboxGroupGap,
		values:  make(map[*Checkbox]string),
		initial: make(map[string]bool),
	}
	for _, o := range opts {
		o(g)
	}
	ele := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if ele == nil {
		return nil
	}
	g.LayoutEle = ele
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(g.spacing)
	ele.SetAlignH(xcc.Layout_Align_Left)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.AttachWindow(ele.Handle)
	return g
}

// Add 在组内追加一个条目(直接子项)并返回它。value 是组值, text 为标签文字。
func (g *CheckboxGroup) Add(value, text string, opts ...CheckboxOption) *Checkbox {
	if g.invalid {
		opts = append(opts, WithCheckboxInvalid())
	}
	if g.disabled {
		opts = append(opts, WithCheckboxDisabled())
	}
	if g.initial[value] {
		opts = append(opts, WithCheckboxChecked(true))
	}
	c := NewCheckbox(0, 0, 0, 0, text, g.Handle, opts...)
	if c == nil {
		return nil
	}
	xui.ItemFill(c)
	return g.Attach(c, value)
}

// Attach 把一个已创建的 Checkbox 纳入组语义。用于条目需要放在 Field 等子容器里的场景。
// 同一个 Checkbox 重复 Attach 为空操作。
func (g *CheckboxGroup) Attach(c *Checkbox, value string) *Checkbox {
	if g == nil || c == nil {
		return c
	}
	if _, ok := g.values[c]; ok {
		return c
	}
	g.items = append(g.items, c)
	g.values[c] = value
	if g.initial[value] {
		c.SetChecked(true)
		delete(g.initial, value)
	}
	if g.invalid {
		c.SetInvalid(true)
	}
	if g.disabled {
		c.SetDisabled(true)
	}
	if g.focusRingProxy {
		c.SetFocusRingProxy(true)
	}
	c.OnChange(func(checked bool) { g.onItemChange() })
	g.syncParent()
	g.relayout()
	return c
}

// SetParent 绑定父项复选框(全选/半选联动的入口)。只设置一次; 父项可以在组外任意位置
// (例如表格表头), 点击父项会把所有可用子项切成同一状态。
func (g *CheckboxGroup) SetParent(parent *Checkbox) *CheckboxGroup {
	if g == nil || parent == nil || g.parent == parent {
		return g
	}
	g.parent = parent
	if g.invalid {
		parent.SetInvalid(true)
	}
	if g.disabled {
		parent.SetDisabled(true)
	}
	if g.focusRingProxy {
		parent.SetFocusRingProxy(true)
	}
	parent.OnChange(func(checked bool) { g.onParentChange(checked) })
	g.syncParent()
	return g
}

// Parent 返回绑定的父项复选框(未设置时为 nil)。
func (g *CheckboxGroup) Parent() *Checkbox { return g.parent }

// Values 返回当前选中的组值(按条目加入顺序)。
func (g *CheckboxGroup) Values() []string {
	if g == nil {
		return nil
	}
	out := make([]string, 0, len(g.items))
	for _, it := range g.items {
		if it.Checked() {
			out = append(out, g.values[it])
		}
	}
	return out
}

// SetValues 设置选中值, 不触发 OnChange。传空 = 全部取消。
func (g *CheckboxGroup) SetValues(values ...string) *CheckboxGroup {
	if g == nil {
		return g
	}
	want := make(map[string]bool, len(values))
	for _, v := range values {
		want[v] = true
	}
	g.applying = true
	for _, it := range g.items {
		it.SetChecked(want[g.values[it]])
	}
	g.applying = false
	g.initial = make(map[string]bool)
	g.syncParent()
	return g
}

// OnChange 注册选中值变化回调(用户点击/空格切换子项或父项时触发)。
func (g *CheckboxGroup) OnChange(fn func(values []string)) *CheckboxGroup {
	if g == nil {
		return g
	}
	g.cb = fn
	return g
}

// SetDisabled 禁用/启用整组(含父项)。禁用时记下各控件原来的禁用状态, 重新启用后
// 恢复(单独禁用的条目不会被组级启用误开)。
func (g *CheckboxGroup) SetDisabled(v bool) *CheckboxGroup {
	if g == nil || g.disabled == v {
		return g
	}
	g.disabled = v
	if v {
		g.disabledOwn = make(map[*Checkbox]bool, len(g.items)+1)
		for _, it := range g.items {
			g.disabledOwn[it] = it.Disabled()
			it.SetDisabled(true)
		}
		if g.parent != nil {
			g.disabledOwn[g.parent] = g.parent.Disabled()
			g.parent.SetDisabled(true)
		}
		return g
	}
	for _, it := range g.items {
		it.SetDisabled(g.disabledOwn[it])
	}
	if g.parent != nil {
		g.parent.SetDisabled(g.disabledOwn[g.parent])
	}
	g.disabledOwn = nil
	g.syncParent()
	return g
}

// Disabled 返回整组是否禁用。
func (g *CheckboxGroup) Disabled() bool { return g != nil && g.disabled }

// SetInvalid 设置/清除整组的校验失败态(对每个条目与父项生效)。
func (g *CheckboxGroup) SetInvalid(v bool) *CheckboxGroup {
	if g == nil || g.invalid == v {
		return g
	}
	g.invalid = v
	for _, it := range g.items {
		it.SetInvalid(v)
	}
	if g.parent != nil {
		g.parent.SetInvalid(v)
	}
	return g
}

// Invalid 返回整组是否处于校验失败态。
func (g *CheckboxGroup) Invalid() bool { return g != nil && g.invalid }

// SetSpacing 运行时修改条目间距(逻辑像素)。
func (g *CheckboxGroup) SetSpacing(px int32) *CheckboxGroup {
	if g == nil {
		return g
	}
	if px < 0 {
		px = 0
	}
	if g.spacing == px {
		return g
	}
	g.spacing = px
	g.SetSpace(px)
	g.relayout()
	return g
}

// SetFocusRingProxy 让组内所有条目与父项的键盘焦点环由外层包裹元素代理。
func (g *CheckboxGroup) SetFocusRingProxy(v bool) *CheckboxGroup {
	if g == nil {
		return g
	}
	g.focusRingProxy = v
	for _, it := range g.items {
		it.SetFocusRingProxy(v)
	}
	if g.parent != nil {
		g.parent.SetFocusRingProxy(v)
	}
	return g
}

// SetFieldState 实现 FieldControl: 由 Field 级联校验失败/禁用态。
func (g *CheckboxGroup) SetFieldState(invalid, disabled bool) {
	g.SetInvalid(invalid)
	g.SetDisabled(disabled)
}

// FieldDisabled 实现 FieldControl。
func (g *CheckboxGroup) FieldDisabled() bool { return g.Disabled() }

func (g *CheckboxGroup) onItemChange() {
	if g.applying {
		return
	}
	g.syncParent()
	g.emit()
}

// onParentChange 处理父项被用户切换: 把所有可用子项切成同一状态, 再通知组回调。
func (g *CheckboxGroup) onParentChange(checked bool) {
	if g.applying || g.disabled {
		return
	}
	before := g.Values()
	g.applying = true
	for _, it := range g.items {
		if it.IsEnable() {
			it.SetChecked(checked)
		}
	}
	g.applying = false
	g.syncParent()
	if after := g.Values(); !checkboxValuesEqual(before, after) {
		g.emit()
	}
}

// syncParent 按子项状态更新父项: 全选 = 选中, 部分 = 不确定, 全不选 = 未选。
// 程序化写入不触发父项的 OnChange; 分组被整体禁用时也照样记录, 重新启用后可见。
func (g *CheckboxGroup) syncParent() {
	if g == nil || g.parent == nil {
		return
	}
	total, on := len(g.items), 0
	for _, it := range g.items {
		if it.Checked() {
			on++
		}
	}
	switch {
	case total == 0 || on == 0:
		g.parent.SetChecked(false)
	case on == total:
		g.parent.SetChecked(true)
	default:
		g.parent.SetIndeterminate(true)
	}
}

func (g *CheckboxGroup) emit() {
	if g.cb != nil {
		g.cb(g.Values())
	}
}

func (g *CheckboxGroup) relayout() {
	if g == nil {
		return
	}
	if p := g.GetParentEle(); p != 0 {
		xc.XEle_AdjustLayout(p)
	} else {
		g.AdjustLayout()
	}
}

func checkboxValuesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
