package ui

import (
	"strconv"
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Combobox 几何(逻辑像素), 对齐 shadcn 文档的 Combobox 示例(Popover + Command):
// 弹层内顶部搜索行 h-9(36), 条目 h-7(28, 同 CommandItem), 分组标签 24, 内容 p-1(4),
// 列表上限 300(上游 CommandList max-h-[300px])。
const (
	comboboxSearchH int32 = 36
	comboboxPad     int32 = 4
	comboboxMaxList int32 = 300
	comboboxMinW    int32 = 200
	comboboxEmptyH  int32 = 40
)

// Combobox 自动补全选择框 —— 对应 shadcn 文档的 **Combobox**(输入过滤 + 建议列表),
// 与 ui.ComboBox(shadcn select 的原生下拉)不是同一个东西:
//
//   - 触发区是 outline 按钮(同 Select), 显示当前值/占位符, 右侧 chevrons-up-down;
//   - 点开弹层 = 顶部搜索框(输入即过滤, 不区分大小写) + 建议列表 + 空状态;
//   - 条目支持前置图标 / 禁用 / 分组标签 / 左侧(多选)或右侧(单选)选中勾;
//   - 键盘: 搜索框里 ↑/↓ 循环移动高亮(跳过禁用)、Home/End 首末、Enter 选中、
//     ESC 关闭; 鼠标悬停即高亮, 点击选中;
//   - 单选选中即关闭; `WithComboboxMultiple` 多选时点击切换并保持打开,
//     触发区显示 "首个值 +N"。
//
// 弹层复用 popoverPanel(阴影窗 + 卡片 body + 失焦/ESC 关闭 + 工作区夹取),
// 数据模型与 Select 共用 SelectItem / SelectGroup。
type Combobox struct {
	*widget.Button
	groups            []SelectGroup
	rows              []*comboboxRow
	picked            map[string]bool
	multiple          bool
	placeholder       string
	searchPlaceholder string
	emptyText         string
	size              SelectSize
	invalid           bool
	disabled          bool

	panel  *popoverPanel
	scroll *ScrollArea
	stack  *widget.LayoutEle
	edit   *widget.Edit
	empty  *Label

	built   bool
	query   string
	active  int32 // 当前高亮条目在 rows 里的下标(-1 无)
	dying   bool
	lastTop bool // 弹层当前在触发区上方(过滤 resize 时保持同侧, 见 updateAnchorSide)

	changes []func(items []SelectItem)
}

// comboboxRowKind 弹层里的一行: 条目或分组标签。
type comboboxRowKind int32

const (
	comboboxRowItem comboboxRowKind = iota
	comboboxRowLabel
)

type comboboxRow struct {
	kind   comboboxRowKind
	item   SelectItem
	group  int32 // 所属分组下标(标签行即本组)
	ele    *widget.Element
	shown  bool
	active bool
}

// ComboboxOption 配置项。
type ComboboxOption func(*Combobox)

// WithComboboxItems 设置平铺条目(无分组)。
func WithComboboxItems(items []SelectItem) ComboboxOption {
	return func(c *Combobox) { c.groups = []SelectGroup{{Items: items}} }
}

// WithComboboxGroups 设置分组条目(组内自动加标签行)。
func WithComboboxGroups(groups []SelectGroup) ComboboxOption {
	return func(c *Combobox) { c.groups = groups }
}

// WithComboboxPlaceholder 设置未选择时的占位文本。
func WithComboboxPlaceholder(s string) ComboboxOption {
	return func(c *Combobox) { c.placeholder = s }
}

// WithComboboxSearchPlaceholder 设置弹层搜索框的占位文本。
func WithComboboxSearchPlaceholder(s string) ComboboxOption {
	return func(c *Combobox) { c.searchPlaceholder = s }
}

// WithComboboxEmptyText 设置无匹配结果时的空状态文字。
func WithComboboxEmptyText(s string) ComboboxOption {
	return func(c *Combobox) { c.emptyText = s }
}

// WithComboboxValue 设置初始选中值(单选)。多选用 WithComboboxValues。
func WithComboboxValue(v string) ComboboxOption {
	return func(c *Combobox) {
		c.picked[v] = true
	}
}

// WithComboboxValues 设置初始选中值(多选)。
func WithComboboxValues(vs ...string) ComboboxOption {
	return func(c *Combobox) {
		for _, v := range vs {
			c.picked[v] = true
		}
	}
}

// WithComboboxMultiple 多选模式: 点击条目切换选中且弹层保持打开。
func WithComboboxMultiple() ComboboxOption {
	return func(c *Combobox) { c.multiple = true }
}

// WithComboboxSize 设置触发区尺寸预设(sm = h-7 = 28)。
func WithComboboxSize(s SelectSize) ComboboxOption {
	return func(c *Combobox) { c.size = s }
}

// WithComboboxInvalid 设置校验失败态(aria-invalid)。
func WithComboboxInvalid() ComboboxOption {
	return func(c *Combobox) { c.invalid = true }
}

// WithComboboxDisabled 禁用触发区。
func WithComboboxDisabled() ComboboxOption {
	return func(c *Combobox) { c.disabled = true }
}

// NewCombobox 创建自动补全选择框。cx/cy 传 0: cy 用尺寸档高度, cx 按内容量宽
// (放卡片里通常配 xui.ItemFill 让它铺满)。
func NewCombobox(x, y, cx, cy int32, hParent int, opts ...ComboboxOption) *Combobox {
	c := &Combobox{
		picked:            map[string]bool{},
		active:            -1,
		placeholder:       "请选择...",
		searchPlaceholder: "搜索...",
		emptyText:         "没有找到结果。",
	}
	for _, o := range opts {
		o(c)
	}
	if cy <= 0 {
		if c.size == SelectSizeSm {
			cy = 28
		} else {
			cy = 32
		}
	}
	btn := widget.NewButton(x, y, cx, cy, "", hParent)
	if btn == nil {
		return nil
	}
	c.Button = btn
	if cx <= 0 {
		btn.SetWidth(c.naturalWidth())
	}
	btn.EnableBkTransparent(true)
	btn.SetTextColor(0) // 文字自绘
	btn.AddEvent_Paint(c.onPaint)
	btn.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if !c.IsEnable() {
			return 0
		}
		xui.NotePointerPress()
		// 弹层开着时点触发区: 弹层先因失焦关闭(更新了 lastClose), 这里不再重开 ——
		// 否则会出现"点两次才关掉"(同 Select / DatePicker)。
		if c.panel != nil && (c.panel.IsOpen() || c.panel.RecentlyClosed()) {
			return 0
		}
		c.Open()
		return 0
	})
	btn.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		c.dying = true
		if c.panel != nil {
			c.panel.Destroy()
		}
		c.panel, c.scroll, c.stack, c.edit, c.empty = nil, nil, nil, nil, nil
		return 0
	})
	if c.disabled {
		c.SetDisabled(true)
	}
	xui.AttachWindow(btn.Handle)
	return c
}

// ===== 数据 =====

// SetItems 替换为平铺条目(丢弃弹层, 下次打开重建)。
func (c *Combobox) SetItems(items []SelectItem) *Combobox {
	return c.SetGroups([]SelectGroup{{Items: items}})
}

// SetGroups 替换分组条目。
func (c *Combobox) SetGroups(groups []SelectGroup) *Combobox {
	if c == nil {
		return c
	}
	c.groups = groups
	c.built = false
	if c.panel != nil {
		c.panel.Destroy()
		c.panel, c.scroll, c.stack, c.edit, c.empty = nil, nil, nil, nil, nil
	}
	c.syncPicked()
	c.Redraw(false)
	return c
}

// Value 返回单选值(多选返回首个选中值), 未选中为空串。
func (c *Combobox) Value() string {
	if c == nil {
		return ""
	}
	for _, g := range c.groups {
		for _, it := range g.Items {
			if c.picked[it.Value] {
				return it.Value
			}
		}
	}
	return ""
}

// Values 返回全部选中值(按数据顺序)。
func (c *Combobox) Values() []string {
	if c == nil {
		return nil
	}
	var out []string
	for _, g := range c.groups {
		for _, it := range g.Items {
			if c.picked[it.Value] {
				out = append(out, it.Value)
			}
		}
	}
	return out
}

// Selected 返回当前选中的条目(单选语义, 多选返回第一个)。
func (c *Combobox) Selected() (SelectItem, bool) {
	v := c.Value()
	for _, g := range c.groups {
		for _, it := range g.Items {
			if it.Value == v {
				return it, true
			}
		}
	}
	return SelectItem{}, false
}

// SetValue 按值选中条目(单选语义: 先清空其它选中), 不触发 OnChange。
func (c *Combobox) SetValue(v string) *Combobox {
	if c == nil {
		return c
	}
	c.picked = map[string]bool{}
	if v != "" {
		c.picked[v] = true
	}
	c.syncPicked()
	c.Redraw(false)
	c.refreshRows()
	return c
}

// SetValues 设置多选值, 不触发 OnChange。
func (c *Combobox) SetValues(vs ...string) *Combobox {
	if c == nil {
		return c
	}
	c.picked = map[string]bool{}
	for _, v := range vs {
		c.picked[v] = true
	}
	c.syncPicked()
	c.Redraw(false)
	c.refreshRows()
	return c
}

// OnChange 注册选中变化回调(用户操作触发; SetValue/SetValues 不触发)。
func (c *Combobox) OnChange(fn func(items []SelectItem)) *Combobox {
	if c != nil {
		c.changes = append(c.changes, fn)
	}
	return c
}

// syncPicked 清掉数据里已经不存在的值。
func (c *Combobox) syncPicked() {
	exist := map[string]bool{}
	for _, g := range c.groups {
		for _, it := range g.Items {
			exist[it.Value] = true
		}
	}
	for v := range c.picked {
		if !exist[v] {
			delete(c.picked, v)
		}
	}
}

// SetInvalid 设置/清除校验失败态。
func (c *Combobox) SetInvalid(v bool) *Combobox {
	if c == nil || c.invalid == v {
		return c
	}
	c.invalid = v
	c.Redraw(false)
	return c
}

// Invalid 返回当前是否处于校验失败态。
func (c *Combobox) Invalid() bool { return c != nil && c.invalid }

// SetDisabled 禁用/启用触发区(开着时一并关闭弹层)。
func (c *Combobox) SetDisabled(v bool) *Combobox {
	if c == nil {
		return c
	}
	c.disabled = v
	c.Enable(!v)
	if v && c.panel != nil {
		c.panel.Hide()
	}
	c.Redraw(false)
	return c
}

// Disabled 返回触发区是否禁用。
func (c *Combobox) Disabled() bool { return c != nil && c.disabled }

// FitWidth 按当前选中项重新量宽(仅 cx 传 0 创建的触发区)。
func (c *Combobox) FitWidth() *Combobox {
	if c == nil {
		return c
	}
	c.SetWidth(c.naturalWidth())
	c.Redraw(false)
	return c
}

// ===== 弹层 =====

// Open 打开弹层(内容按需构建/重建), 焦点落到搜索框。
func (c *Combobox) Open() *Combobox {
	if c == nil || c.disabled || (c.panel != nil && c.panel.IsOpen()) {
		return c
	}
	if !c.built {
		c.build()
	}
	if c.panel == nil {
		return c
	}
	x, y := popupPosition(c.Handle, c.panel.cw, c.panel.ch, PopoverSideBottom, PopoverAlignStart, 4)
	c.panel.ShowAt(c.Handle, x, y)
	c.updateAnchorSide(y, c.panel.ch)
	// 每次打开都从"无搜索词"开始: 上一次的过滤结果不能带进来(否则搜索框是空的、
	// 列表却是过滤后的)。applyFilter 同时会把高亮落到第一条。
	c.query = ""
	if c.edit != nil {
		c.edit.SetText("")
		c.edit.SetFocus()
	}
	c.applyFilter()
	return c
}

// SetQuery 设置搜索词并立即过滤(程序化驱动, 等价于用户在搜索框里输入)。
func (c *Combobox) SetQuery(q string) *Combobox {
	if c == nil {
		return c
	}
	c.query = q
	if c.edit != nil {
		c.edit.SetText(q)
	}
	c.applyFilter()
	return c
}

// Query 返回当前搜索词。
func (c *Combobox) Query() string {
	if c == nil {
		return ""
	}
	return c.query
}

// Close 关闭弹层。
func (c *Combobox) Close() *Combobox {
	if c != nil && c.panel != nil {
		c.panel.Hide()
	}
	return c
}

// IsOpen 返回弹层是否打开。
func (c *Combobox) IsOpen() bool { return c != nil && c.panel != nil && c.panel.IsOpen() }

// build 构建弹层: 搜索行 + 列表 + 空状态。
func (c *Combobox) build() {
	c.built = true
	if c.panel != nil {
		c.panel.Destroy()
		c.panel, c.scroll, c.stack, c.edit, c.empty = nil, nil, nil, nil, nil
	}
	c.syncPicked()

	cw := c.GetWidth()
	if cw < comboboxMinW {
		cw = comboboxMinW
	}
	// 行还没建, 这里先给一个占位高度; 行建好后 layoutContent 会按实际内容重算。
	listH := comboboxSearchH
	ch := comboboxPad*2 + comboboxSearchH + listH

	panel := newPopoverPanel(cw, ch, c.Handle)
	if panel == nil {
		return
	}
	c.panel = panel

	// 内容宿主: 纵向(搜索行 + 列表), 四周 p-1(4)。内边距用元素 padding 表达;
	// 里面的子项**必须**锁 Fixed 宽(见 DESIGN 8.129): 元素 padding 不扣子项 Fill 宽,
	// 用 Fill 的话列表会按父级全宽解析、向右溢出, 高亮条直接压到弹层圆角上。
	root := widget.NewLayoutEle(0, 0, 0, 0, panel.Content())
	if root == nil {
		return
	}
	root.EnableHorizon(false)
	root.EnableLayout(true)
	root.EnableAutoWrap(false)
	root.EnableBkTransparent(true)
	root.EnableDrawBorder(false)
	root.SetSpace(0)
	root.SetPadding(comboboxPad, comboboxPad, comboboxPad, comboboxPad)
	root.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	root.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, ch)
	c.stack = root

	// ---- 搜索行 ----
	row := widget.NewLayoutEle(0, 0, 0, comboboxSearchH, root.Handle)
	if row != nil {
		row.EnableHorizon(true)
		row.EnableLayout(true)
		row.EnableAutoWrap(false)
		row.EnableBkTransparent(true)
		row.EnableDrawBorder(false)
		row.SetAlignV(xcc.Layout_Align_Center)
		row.SetSpace(6)
		row.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, cw-comboboxPad*2)
		row.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, comboboxSearchH)
		row.SetPadding(8, 0, 8, 0)

		icon := widget.NewElement(0, 0, 16, 16, row.Handle)
		if icon != nil {
			icon.EnableBkTransparent(true)
			icon.EnableDrawBorder(false)
			icon.EnableFocus(false)
			icon.EnableSwitchFocus(false)
			icon.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 16)
			icon.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 16)
			icon.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
				p := xui.BeginPaint(hEle, hDraw)
				if p == nil {
					return 0
				}
				*pbHandled = true
				p.Icon(icons.IconSearch, &p.Rc, xui.Current().MutedForeground, 2)
				return 0
			})
		}

		c.edit = widget.NewEdit(0, 0, 0, comboboxSearchH, row.Handle)
		if c.edit != nil {
			c.edit.EnableBkTransparent(true)
			c.edit.EnableDrawBorder(false).EnableDrawFocus(false)
			c.edit.EnableAutoSelAll(false)
			c.edit.SetBorderSize(0, 0, 0, 0)
			c.edit.SetTextAlign(xcc.Edit_TextAlign_Flag_Left | xcc.Edit_TextAlign_Flag_Center_V)
			if f := xui.FontHandle(xui.Current().FontDefault); f != 0 {
				c.edit.SetFont(f)
			}
			c.edit.SetDefaultText(c.searchPlaceholder)
			c.edit.SetDefaultTextColor(uint32(xui.Current().MutedForeground))
			c.edit.SetTextColor(uint32(xui.Current().Foreground))
			c.edit.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
			c.edit.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, comboboxSearchH)
			c.edit.AddEvent_Edit_Changed(func(hEle int, pbHandled *bool) int {
				c.query = c.edit.GetText()
				c.applyFilter()
				return 0
			})
			c.edit.AddEvent_KeyDown(c.onSearchKeyDown)
		}
	}

	// ---- 列表 ----
	c.scroll = NewScrollArea(0, 0, 0, listH, root.Handle)
	if c.scroll == nil {
		return
	}
	c.scroll.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, cw-comboboxPad*2)
	c.scroll.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, listH)
	stack := c.scroll.ContentStack(0)
	if stack == nil {
		return
	}

	c.empty = NewLabel(0, 0, 0, comboboxEmptyH, c.emptyText, stack.Handle)
	if c.empty != nil {
		c.empty.SetMuted().SetFontSize(14).SetAlignCenter()
		c.empty.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, cw-comboboxPad*2)
		c.empty.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, comboboxEmptyH)
		c.empty.Show(false)
	}

	// ---- 行(条目 / 分组标签) ----
	c.rows = c.rows[:0]
	for gi, g := range c.groups {
		gi := gi // go.mod 是 1.18: 循环变量旧语义, 闭包必须拷贝
		if g.Label != "" {
			r := &comboboxRow{kind: comboboxRowLabel, item: SelectItem{Text: g.Label}, group: int32(gi), shown: true}
			ele := widget.NewElement(0, 0, 0, 0, stack.Handle)
			if ele != nil {
				r.ele = ele
				ele.EnableBkTransparent(true)
				ele.EnableDrawBorder(false)
				ele.EnableFocus(false)
				ele.EnableSwitchFocus(false)
				xui.ItemFill(ele)
				ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, selectLabelH)
				ele.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
					p := xui.BeginPaint(hEle, hDraw)
					if p == nil {
						return 0
					}
					*pbHandled = true
					t := xui.Current()
					p.FillRound(&p.Rc, t.Popover, 0) // 擦底
					rc := p.Rc
					rc.Left += 6 // 与条目文字(pl-1.5)对齐
					rc.Top += 4  // py-1
					rc.Bottom -= 2
					if f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 12}); f != 0 {
						p.Draw.SetFont(f)
					}
					p.Text(r.item.Text, &rc, t.MutedForeground, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
					return 0
				})
				c.rows = append(c.rows, r)
			}
		}
		for _, it := range g.Items {
			it := it
			r := &comboboxRow{kind: comboboxRowItem, item: it, group: int32(gi), shown: true}
			ele := widget.NewElement(0, 0, 0, 0, stack.Handle)
			if ele != nil {
				r.ele = ele
				ele.EnableBkTransparent(true)
				ele.EnableDrawBorder(false)
				ele.EnableFocus(false)
				ele.EnableSwitchFocus(false)
				xui.ItemFill(ele)
				ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, selectItemH)
				ele.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
					c.paintRow(r, hEle, hDraw)
					return 0
				})
				ele.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
					c.hoverRow(r)
					return 0
				})
				ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
					c.hoverRow(r)
					return 0
				})
				ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
					c.chooseRow(r)
					return 0
				})
				c.rows = append(c.rows, r)
			}
		}
	}
	c.applyFilter()
	// 行齐了才知道真实内容高: 重算列表与弹层高度(此时 panel 未打开, 只 resize 不重定位)。
	c.layoutContent()
}

// layoutContent 按当前可见行重算列表高度与弹层高度(不要求弹层已打开)。
func (c *Combobox) layoutContent() {
	if c == nil || c.scroll == nil || c.panel == nil {
		return
	}
	listH := c.viewHeight()
	c.scroll.SetHeight(listH)
	c.scroll.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, listH)
	ch := comboboxPad*2 + comboboxSearchH + listH
	if c.stack != nil {
		c.stack.SetHeight(ch)
		c.stack.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, ch)
	}
	c.panel.resize(c.panel.cw, ch)
	if c.stack != nil {
		c.stack.AdjustLayout()
	}
	c.scroll.AdjustLayout()
}

// onSearchKeyDown 搜索框键盘: ↑/↓ 循环高亮(跳过禁用)、Home/End、Enter 选中、ESC 关闭。
func (c *Combobox) onSearchKeyDown(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
	switch int(wParam) {
	case xcc.VK_Down:
		c.moveActive(1)
		*pbHandled = true
	case xcc.VK_Up:
		c.moveActive(-1)
		*pbHandled = true
	case xcc.VK_Home:
		c.setActive(c.firstEnabled(0, 1), true)
		*pbHandled = true
	case xcc.VK_End:
		c.setActive(c.firstEnabled(int32(len(c.rows))-1, -1), true)
		*pbHandled = true
	case xcc.VK_Enter:
		if c.active >= 0 && int(c.active) < len(c.rows) {
			c.chooseRow(c.rows[c.active])
		}
		*pbHandled = true
	case xcc.VK_Esc:
		c.Close()
		c.SetFocus()
		*pbHandled = true
	}
	return 0
}

// firstEnabled 从 from 起按 dir(+1/-1)找第一个可高亮的条目行。
func (c *Combobox) firstEnabled(from int32, dir int32) int32 {
	for i := from; i >= 0 && i < int32(len(c.rows)); i += dir {
		r := c.rows[i]
		if r.kind == comboboxRowItem && r.shown && !r.item.Disabled {
			return i
		}
	}
	return -1
}

// moveActive 按 dir 移动高亮(在**可见条目**之间循环, 跳过禁用与标签行)。
func (c *Combobox) moveActive(dir int32) {
	n := len(c.rows)
	if n == 0 {
		return
	}
	idx := int(c.active)
	if idx < 0 || idx >= n {
		if dir >= 0 {
			c.setActive(c.firstEnabled(0, 1), true)
		} else {
			c.setActive(c.firstEnabled(int32(n)-1, -1), true)
		}
		return
	}
	for step := 0; step < n; step++ {
		idx = (idx + int(dir) + n) % n
		r := c.rows[idx]
		if r.kind == comboboxRowItem && r.shown && !r.item.Disabled {
			c.setActive(int32(idx), true)
			return
		}
	}
}

// setActive 设置高亮行并把它滚进视口。
func (c *Combobox) setActive(idx int32, reveal bool) {
	if idx < 0 || idx >= int32(len(c.rows)) {
		return
	}
	if c.active >= 0 && int(c.active) < len(c.rows) {
		c.rows[c.active].active = false
		if c.rows[c.active].ele != nil {
			c.rows[c.active].ele.Redraw(false)
		}
	}
	c.active = idx
	r := c.rows[idx]
	r.active = true
	if r.ele != nil {
		r.ele.Redraw(false)
	}
	if reveal {
		c.revealRow(idx)
	}
}

// revealRow 把高亮行滚进列表视口。
func (c *Combobox) revealRow(idx int32) {
	if c == nil || c.scroll == nil || idx < 0 || idx >= int32(len(c.rows)) {
		return
	}
	r := c.rows[idx]
	if r.ele == nil {
		return
	}
	var ir, cr xc.RECT
	xc.XEle_GetWndClientRect(r.ele.Handle, &ir)
	xc.XEle_GetWndClientRect(c.scroll.Content().Handle, &cr)
	top := ir.Top - cr.Top
	view := c.scroll.GetViewHeight()
	if view <= 0 {
		view = c.scroll.GetHeight()
	}
	if view <= 0 {
		return
	}
	pos := c.scroll.GetViewPosV()
	h := selectItemH
	if r.kind == comboboxRowLabel {
		h = selectLabelH
	}
	if top < pos {
		c.scroll.ScrollPosYV(top)
	} else if top+h > pos+view {
		c.scroll.ScrollPosYV(top + h - view)
	}
}

// hoverRow 鼠标悬停即高亮(shadcn 里是 focus:bg-accent, 指针进即高亮)。
func (c *Combobox) hoverRow(r *comboboxRow) {
	if c == nil || r == nil || r.kind != comboboxRowItem || !r.shown || r.item.Disabled {
		return
	}
	for i, it := range c.rows {
		if it == r {
			c.setActive(int32(i), false)
			return
		}
	}
}

// chooseRow 选中一行: 单选回写并关闭, 多选切换并保持打开。
func (c *Combobox) chooseRow(r *comboboxRow) {
	if c == nil || r == nil || r.kind != comboboxRowItem || !r.shown || r.item.Disabled {
		return
	}
	if c.multiple {
		if c.picked[r.item.Value] {
			delete(c.picked, r.item.Value)
		} else {
			c.picked[r.item.Value] = true
		}
		// 勾选状态画在行上(弹层里的行), 数据变了行要立刻重绘 —— 否则点击后
		// 勾选标记要等下一次无关重绘才出现
		c.refreshRows()
	} else {
		c.picked = map[string]bool{r.item.Value: true}
	}
	c.Redraw(false)
	if c.edit != nil {
		c.edit.Redraw(false)
	}
	c.notify()
	if !c.multiple {
		c.Close()
		c.SetFocus()
	}
}

// notify 触发 OnChange(按数据顺序回传全部选中条目)。
func (c *Combobox) notify() {
	if len(c.changes) == 0 {
		return
	}
	var items []SelectItem
	for _, g := range c.groups {
		for _, it := range g.Items {
			if c.picked[it.Value] {
				items = append(items, it)
			}
		}
	}
	for _, fn := range c.changes {
		fn(items)
	}
}

// refreshRows 数据外的状态变化(SetValue/SetValues/多选勾选)后刷新条目勾选。
// 用立即重绘: 行是弹层小窗里的独立元素, 立即重绘不影响宿主窗口的待重绘区域,
// 且勾选反馈必须跟上点击节奏。
func (c *Combobox) refreshRows() {
	for _, r := range c.rows {
		if r.ele != nil {
			r.ele.Redraw(true)
		}
	}
}

// updateAnchorSide 记录弹层当前在触发区的哪一侧(过滤 resize 时保持同侧)。
func (c *Combobox) updateAnchorSide(y, h int32) {
	if c == nil {
		return
	}
	var rc, wr xc.RECT
	xc.XEle_GetWndClientRect(c.Handle, &rc)
	hWnd := xc.XWidget_GetHWINDOW(c.Handle)
	if hWnd <= 0 {
		return
	}
	xc.XWnd_GetRect(hWnd, &wr)
	c.lastTop = y+h <= wr.Top+rc.Top+2
}

// applyFilter 按搜索词显示/隐藏行, 同步列表与弹层高度。
func (c *Combobox) applyFilter() {
	if c == nil || !c.built || c.dying {
		return
	}
	q := strings.ToLower(strings.TrimSpace(c.query))
	groupHas := make([]bool, len(c.groups))
	for _, r := range c.rows {
		if r.kind == comboboxRowLabel {
			continue
		}
		match := q == "" || strings.Contains(strings.ToLower(r.item.Text), q) ||
			strings.Contains(strings.ToLower(r.item.Value), q)
		r.shown = match
		if match {
			groupHas[r.group] = true
		}
		if r.ele != nil {
			r.ele.Show(match)
		}
	}
	for _, r := range c.rows {
		if r.kind != comboboxRowLabel {
			continue
		}
		show := groupHas[r.group]
		r.shown = show
		if r.ele != nil {
			r.ele.Show(show)
		}
	}
	visible := 0
	for _, r := range c.rows {
		if r.shown {
			visible++
		}
	}
	if c.empty != nil {
		c.empty.Show(visible == 0)
	}
	if c.stack != nil {
		c.stack.AdjustLayout()
	}
	c.resizePopup()
	c.active = -1
	c.setActive(c.firstEnabled(0, 1), false)
}

// viewHeight 列表可视高度: 可见行总高, 上限 comboboxMaxList。
func (c *Combobox) viewHeight() int32 {
	h := int32(0)
	for _, r := range c.rows {
		if r.kind == comboboxRowLabel {
			if r.shown {
				h += selectLabelH
			}
			continue
		}
		if r.shown {
			h += selectItemH
		}
	}
	if h == 0 {
		h = comboboxEmptyH
	}
	if h > comboboxMaxList {
		h = comboboxMaxList
	}
	return h
}

// resizePopup 过滤结果变化后同步列表高度与弹层高度/位置。
func (c *Combobox) resizePopup() {
	if c == nil || c.scroll == nil || c.panel == nil || !c.panel.IsOpen() {
		return
	}
	listH := c.viewHeight()
	c.scroll.SetHeight(listH)
	c.scroll.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, listH)
	ch := comboboxPad*2 + comboboxSearchH + listH
	if c.panel.ch == ch {
		if c.stack != nil {
			c.stack.AdjustLayout()
		}
		return
	}
	cw := c.panel.cw
	c.panel.resize(cw, ch)
	if c.stack != nil {
		c.stack.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, ch)
	}
	// 保持打开时的弹出方向: 触发器在屏幕底部、弹层翻到上方后, 过滤让高度变小,
	// 若仍按 sideBottom 重算, 缩小的弹层"放得下"就会跳到下方(实测很怪)。
	side := PopoverSideBottom
	if c.lastTop {
		side = PopoverSideTop
	}
	x, y := popupPosition(c.Handle, cw, ch, side, PopoverAlignStart, 4)
	c.updateAnchorSide(y, ch)
	c.panel.Relocate(c.Handle, x, y)
	if c.stack != nil {
		c.stack.AdjustLayout()
	}
	c.scroll.AdjustLayout()
}

// ===== 触发区绘制 =====

// naturalWidth 量出 w-fit 触发区宽: pl-2.5 + 内容 + gap-1.5 + 箭头 + pr-2。
func (c *Combobox) naturalWidth() int32 {
	t := xui.Current()
	f := xui.FontHandle(t.FontDefault)
	w := int32(10 + 6 + 18 + 8)
	if text := c.triggerText(); text != "" {
		w += xui.TextWidth(text, f)
	} else if c.placeholder != "" {
		w += xui.TextWidth(c.placeholder, f)
	}
	if w < 100 {
		w = 100
	}
	return w
}

// triggerText 触发区显示的文字: 多选 "首个值 +N", 单选选中项, 否则空(显示占位符)。
func (c *Combobox) triggerText() string {
	var picked []SelectItem
	for _, g := range c.groups {
		for _, it := range g.Items {
			if c.picked[it.Value] {
				picked = append(picked, it)
			}
		}
	}
	if len(picked) == 0 {
		return ""
	}
	if len(picked) == 1 {
		return picked[0].Text
	}
	return picked[0].Text + " +" + strconv.Itoa(len(picked)-1)
}

// onPaint 触发区: 与 Select/ComboBox 同一套规则(border-input / focus-visible 环 /
// aria-invalid / 暗色 bg-input/30 悬停加深 / disabled)。
func (c *Combobox) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)

	bg, border := t.Card, t.Input
	if st.Disabled {
		bg = t.Input.WithAlpha(128) // disabled:bg-input/50
		border = xui.None
	}
	radius := t.RadiusLg()
	if c.size == SelectSizeSm {
		if md := t.RadiusMd(); md < 10 {
			radius = md
		} else {
			radius = 10
		}
	}
	corners := xui.UniformCorners(radius)
	const inset = int32(3)
	switch {
	case c.invalid && !st.Disabled:
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

	fore := t.Foreground
	text := c.triggerText()
	if text == "" {
		fore = t.MutedForeground
		text = c.placeholder
	}
	if st.Disabled {
		fore = fore.Mix(t.Card, 0.5) // disabled:opacity-50 的文字预混
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	if text != "" {
		dy := textVDy(text)
		rc := xc.RECTF{
			Left:   float32(p.Rc.Left + 10), // pl-2.5
			Top:    float32(p.Rc.Top) - dy,
			Right:  float32(p.Rc.Right - 30),
			Bottom: float32(p.Rc.Bottom) - dy,
		}
		p.TextF(text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	}
	// 右侧 chevrons-up-down(combobox 用上下双箭头, 区别于 select 的单下箭头)
	icon := xc.RECT{
		Left:   p.Rc.Right - 26,
		Top:    p.Rc.Top + (p.H()-16)/2,
		Right:  p.Rc.Right - 10,
		Bottom: p.Rc.Top + (p.H()-16)/2 + 16,
	}
	p.Icon(icons.IconChevronsUpDown, &icon, t.MutedForeground, 2)
	return 0
}

// paintRow 自绘条目: 高亮 bg-accent、前置图标、选中勾(单选右侧 / 多选左侧)、禁用变淡。
func (c *Combobox) paintRow(r *comboboxRow, hEle int, hDraw int) {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return
	}
	t := xui.Current()
	p.FillRound(&p.Rc, t.Popover, 0) // 擦底(透明元素重绘不擦底)
	if r.active && !r.item.Disabled {
		p.FillRound(&p.Rc, t.Accent, t.RadiusSm())
	}
	fore := t.PopoverForeground
	if r.item.Disabled {
		fore = fore.Mix(t.Popover, 0.5)
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	x := p.Rc.Left + 6 // pl-1.5
	if c.multiple {
		// 多选: 左侧勾(选中才画), 占位固定 18
		if c.picked[r.item.Value] {
			rc := xc.RECT{Left: x, Top: (p.Rc.Top + p.Rc.Bottom - 16) / 2,
				Right: x + 16, Bottom: (p.Rc.Top + p.Rc.Bottom + 16) / 2}
			p.Icon(icons.IconCheck, &rc, fore, 2)
		}
		x += 18
	}
	if r.item.Icon != "" {
		rc := xc.RECT{Left: x, Top: (p.Rc.Top + p.Rc.Bottom - 16) / 2,
			Right: x + 16, Bottom: (p.Rc.Top + p.Rc.Bottom + 16) / 2}
		p.Icon(r.item.Icon, &rc, fore, 2)
		x += 16
		if r.item.Text != "" {
			x += 6
		}
	}
	if r.item.Text != "" {
		dy := textVDy(r.item.Text)
		rc := xc.RECTF{
			Left:   float32(x),
			Top:    float32(p.Rc.Top) - dy,
			Right:  float32(p.Rc.Right - 24),
			Bottom: float32(p.Rc.Bottom) - dy,
		}
		p.TextF(r.item.Text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	}
	if !c.multiple && c.picked[r.item.Value] && !r.item.Disabled {
		rc := xc.RECT{
			Left:   p.Rc.Right - 24,
			Top:    (p.Rc.Top + p.Rc.Bottom - 16) / 2,
			Right:  p.Rc.Right - 8,
			Bottom: (p.Rc.Top + p.Rc.Bottom + 16) / 2,
		}
		p.Icon(icons.IconCheck, &rc, t.PopoverForeground, 2)
	}
}
