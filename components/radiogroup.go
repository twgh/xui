package ui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// radioGroupGap 是 RadioGroup 的默认条目间距(gap-2 = 8)。
const radioGroupGap int32 = 8

// RadioGroup 单选组: 竖向流容器 + 一组 Radio, 对应 shadcn radio-group.tsx。
//
// 与单个 Radio 的区别:
//   - 组值语义: Value / SetValue / OnChange 用字符串值(对应 Radix 的 value/onValueChange);
//   - 跨层互斥: 条目可以放在 Field 等子容器里(Attach), 不只依赖 xcgui 的"同父容器互斥";
//   - 键盘导航: ↑/↓/←/→ 在可用项之间移动并选中(循环), Home/End 到首/末项,
//     空格选中当前项 —— 对应 Radix 的 RovingFocusGroup(loop=true);
//   - 组级禁用/校验态: SetDisabled / SetInvalid 作用于所有条目(对应 disabled / aria-invalid)。
//
// 用法:
//
//	grp := ui.NewRadioGroup(card.Handle, ui.WithRadioGroupValue("all"))
//	ui.ItemFill(grp)
//	grp.AddItem("all", "所有消息")
//	grp.AddItem("mentions", "仅提及")
//	grp.OnChange(func(v string) { ... })
type RadioGroup struct {
	*widget.LayoutEle
	items       []*Radio
	spacing     int32
	disabled    bool
	invalid     bool
	initial     string
	disabledOwn map[*Radio]bool
	cb          func(value string)
	applying    bool
}

// RadioGroupOption 单选组配置项。
type RadioGroupOption func(*RadioGroup)

// WithRadioGroupValue 设置初始值(对应 Radix 的 defaultValue)。
func WithRadioGroupValue(v string) RadioGroupOption {
	return func(g *RadioGroup) { g.initial = v }
}

// WithRadioGroupDisabled 让整组初始为禁用态。
func WithRadioGroupDisabled() RadioGroupOption {
	return func(g *RadioGroup) { g.disabled = true }
}

// WithRadioGroupInvalid 让整组初始为校验失败态(每个条目画红边框 + 红环)。
func WithRadioGroupInvalid() RadioGroupOption {
	return func(g *RadioGroup) { g.invalid = true }
}

// WithRadioGroupSpacing 设置条目间距(逻辑像素; 默认 8 = gap-2)。
func WithRadioGroupSpacing(px int32) RadioGroupOption {
	return func(g *RadioGroup) {
		if px < 0 {
			px = 0
		}
		g.spacing = px
	}
}

// NewRadioGroup 创建单选组。组宽铺满、高自撑, 放进卡片内容区即可。
func NewRadioGroup(hParent int, opts ...RadioGroupOption) *RadioGroup {
	g := &RadioGroup{spacing: radioGroupGap}
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

// AddItem 在组内追加一个条目(直接子项)并返回它。value 是组值, text 为标签文字。
func (g *RadioGroup) AddItem(value, text string, opts ...RadioOption) *Radio {
	if g.invalid {
		opts = append(opts, WithRadioInvalid())
	}
	r := NewRadio(0, 0, 0, 0, text, g.Handle, opts...)
	if r == nil {
		return nil
	}
	xui.ItemFill(r)
	return g.Attach(r, value)
}

// Attach 把一个已创建的 Radio 纳入组语义。用于条目需要放在 Field 等子容器里的场景
// (Radio 的父级是子容器, 不再与组内其它条目同父, xcgui 的原生互斥不会生效)。
func (g *RadioGroup) Attach(r *Radio, value string) *Radio {
	if g == nil || r == nil {
		return r
	}
	r.value = value
	if g.invalid {
		r.SetInvalid(true)
	}
	if g.disabled {
		r.Enable(false)
	}
	r.addChangeListener(func(on bool) { g.onItemSelected(r, on) })
	r.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		return g.onItemKeyDown(r, wParam, pbHandled)
	})
	g.items = append(g.items, r)
	// 条目可能是在外部预先选中后接进来的; 保持组内唯一(同父项交给原生互斥)。
	if r.Selected() {
		g.applying = true
		g.syncSelection(r)
		g.applying = false
	}
	if g.initial != "" && value == g.initial {
		g.SetValue(g.initial)
		g.initial = ""
	}
	g.relayout()
	return r
}

// Value 返回当前选中项的值(没有选中时为空串)。
func (g *RadioGroup) Value() string {
	if g == nil {
		return ""
	}
	for _, it := range g.items {
		if it.Selected() {
			return it.value
		}
	}
	return ""
}

// SetValue 设置选中值, 不触发 OnChange。传空串清空全部。
//
// 选中目标只调用 SetSelected(true): 同父兄弟项由原生单选逻辑自动取消
// (见 onItemSelected 的说明), 逐个 SetSelected(false) 反而会把目标也取消。
func (g *RadioGroup) SetValue(value string) *RadioGroup {
	if g == nil {
		return g
	}
	var target *Radio
	if value != "" {
		for _, it := range g.items {
			if it.value == value {
				target = it
				break
			}
		}
	}
	g.applying = true
	if target != nil {
		target.SetSelected(true)
		g.syncSelection(target)
	} else {
		for _, it := range g.items {
			it.SetSelected(false)
		}
	}
	g.applying = false
	return g
}

// OnChange 注册选中值变化回调(用户点击/空格/方向键选中时触发)。
func (g *RadioGroup) OnChange(fn func(value string)) *RadioGroup {
	if g == nil {
		return g
	}
	g.cb = fn
	return g
}

// SetDisabled 禁用/启用整组。禁用时记下组内条目各自原有的禁用状态, 重新启用后恢复
// (单独禁用的条目不会被组级启用误开)。
func (g *RadioGroup) SetDisabled(v bool) *RadioGroup {
	if g == nil || g.disabled == v {
		return g
	}
	g.disabled = v
	if v {
		g.disabledOwn = make(map[*Radio]bool, len(g.items))
		for _, it := range g.items {
			g.disabledOwn[it] = !it.IsEnable()
			it.Enable(false)
			it.Redraw(false)
		}
		return g
	}
	for _, it := range g.items {
		it.Enable(!g.disabledOwn[it])
		it.Redraw(false)
	}
	g.disabledOwn = nil
	return g
}

// Disabled 返回整组是否禁用。
func (g *RadioGroup) Disabled() bool { return g != nil && g.disabled }

// SetInvalid 设置/清除整组的校验失败态(对每个条目生效)。
func (g *RadioGroup) SetInvalid(v bool) *RadioGroup {
	if g == nil || g.invalid == v {
		return g
	}
	g.invalid = v
	for _, it := range g.items {
		it.SetInvalid(v)
	}
	return g
}

// Invalid 返回整组是否处于校验失败态。
func (g *RadioGroup) Invalid() bool { return g != nil && g.invalid }

// SetSpacing 运行时修改条目间距(逻辑像素)。
func (g *RadioGroup) SetSpacing(px int32) *RadioGroup {
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

// onItemSelected 处理条目被用户选中: 取消其它条目并通知组回调。
// 取消选中(false)事件不触发组回调(Radix 的 onValueChange 只在值变化时触发)。
//
// 同父容器里的兄弟项**不要手动取消**: 原生单选的互斥逻辑在收到"取消 B"时会
// 连带把刚选中的 A 也取消(探针实测事件 A:true → A:false, 最终全不选)。
// 只处理跨父级的条目(例如放在 Field 子容器里的项), 那边原生互斥管不到。
func (g *RadioGroup) onItemSelected(item *Radio, on bool) {
	if g.applying || !on {
		return
	}
	g.syncSelection(item)
	if g.cb != nil {
		g.cb(item.value)
	}
}

// syncSelection 保证组内只有 keep 处于选中: 同父项已由原生互斥处理,
// 只把**其它父级**里仍然选中的条目取消。
func (g *RadioGroup) syncSelection(keep *Radio) {
	if keep == nil {
		return
	}
	keepParent := xc.XWidget_GetParentEle(keep.Handle)
	for _, it := range g.items {
		if it != keep && it.Selected() && xc.XWidget_GetParentEle(it.Handle) != keepParent {
			it.SetSelected(false)
		}
	}
}

// onItemKeyDown 处理组内键盘导航(条目自己的 KeyDown 只负责空格)。
func (g *RadioGroup) onItemKeyDown(item *Radio, wParam uintptr, pbHandled *bool) int {
	if g.disabled || item == nil || !item.IsEnable() {
		return 0
	}
	const (
		vkHome  = 0x24
		vkEnd   = 0x23
		vkLeft  = 0x25
		vkUp    = 0x26
		vkRight = 0x27
		vkDown  = 0x28
	)
	var next int
	switch wParam {
	case vkLeft, vkUp:
		next = g.nextEnabled(g.index(item), -1)
	case vkRight, vkDown:
		next = g.nextEnabled(g.index(item), 1)
	case vkHome:
		next = g.nextEnabled(-1, 1)
	case vkEnd:
		next = g.nextEnabled(len(g.items), -1)
	default:
		return 0
	}
	if next < 0 || next >= len(g.items) {
		return 0
	}
	*pbHandled = true
	xui.NoteKeyboard() // 方向键搬过去的焦点要画焦点框
	g.items[next].setSelectedNotify(true)
	focusRadio(g.items[next].Handle)
	return 0
}

func (g *RadioGroup) index(r *Radio) int {
	for i, it := range g.items {
		if it == r {
			return i
		}
	}
	return -1
}

func (g *RadioGroup) nextEnabled(from, dir int) int {
	enabled := make([]bool, len(g.items))
	for i, it := range g.items {
		enabled[i] = it.IsEnable()
	}
	return radioNextEnabled(enabled, from, dir)
}

func (g *RadioGroup) relayout() {
	if g == nil {
		return
	}
	if p := g.GetParentEle(); p != 0 {
		xc.XEle_AdjustLayout(p)
	} else {
		g.AdjustLayout()
	}
}

// radioNextEnabled 在 enabled 里从 from 出发按 dir 找下一个可用项(循环)。
// from 允许越界: -1 表示从第一项之前开始, len 表示从最后一项之后开始(反向)。
func radioNextEnabled(enabled []bool, from, dir int) int {
	n := len(enabled)
	if n == 0 || dir == 0 {
		return -1
	}
	idx := from
	if idx < 0 {
		idx = -1
	} else if idx >= n {
		idx = n
	}
	for i := 0; i < n; i++ {
		idx = ((idx+dir)%n + n) % n
		if enabled[idx] {
			return idx
		}
	}
	return -1
}

// focusRadio 把键盘焦点搬到条目上。必须推迟到消息循环: 在 KeyDown 回调里同步
// SetFocusEle 会被当前按键的默认焦点导航覆盖(实测落到下一个条目上, 同 Menubar)。
func focusRadio(hEle int) {
	if hEle == 0 {
		return
	}
	app.AfterFunc(0, func() {
		if hWindow := xc.XWidget_GetHWINDOW(hEle); hWindow != 0 {
			xc.XWnd_SetFocusEle(hWindow, hEle)
		}
	})
}
