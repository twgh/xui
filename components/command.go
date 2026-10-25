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

// Command 几何常量(逻辑像素), 对齐 shadcn command.tsx(cmdb):
// 搜索行 h-9(36) + border-b; 条目 px-2 py-1.5 text-sm = 32; 分组标签 text-xs + py-1.5 = 24;
// 分组 p-1 = 4; 列表 max-h-[300px]; 空状态 py-6 + text-sm = 52。
const (
	commandInputH   int32 = 36
	commandItemH    int32 = 32
	commandHeadingH int32 = 24
	commandGroupPad int32 = 4
	commandListMaxH int32 = 300
	commandEmptyH   int32 = 52
)

// CommandOption 命令面板配置项。
type CommandOption func(*Command)

// WithCommandPlaceholder 设置搜索框占位文字(默认 "输入命令或搜索...")。
func WithCommandPlaceholder(s string) CommandOption {
	return func(c *Command) { c.placeholder = s }
}

// WithCommandEmptyText 设置无结果时的提示文字(默认 "没有找到结果。" )。
func WithCommandEmptyText(s string) CommandOption {
	return func(c *Command) { c.emptyText = s }
}

// WithCommandOnSelect 注册选中回调(回车或点击条目时触发)。
func WithCommandOnSelect(fn func(value string)) CommandOption {
	return func(c *Command) { c.selects = append(c.selects, fn) }
}

// WithCommandOnChange 注册搜索词变化回调。
func WithCommandOnChange(fn func(query string)) CommandOption {
	return func(c *Command) { c.changes = append(c.changes, fn) }
}

// WithCommandOnEscape 注册 ESC 回调(不设置时: 有搜索词就清空, 没有则忽略)。
func WithCommandOnEscape(fn func()) CommandOption {
	return func(c *Command) { c.escapes = append(c.escapes, fn) }
}

// Command 命令面板/搜索选择列表, 对应 shadcn command.tsx(cmdb 的基础形态):
//   - 顶部搜索行(搜索图标 + 原生 Edit, 下边框), 列表滚动区, 分组标签, 条目图标/快捷键;
//   - 输入即过滤(不区分大小写的包含匹配, 可配 keywords), 无结果时显示空状态;
//   - ↑/↓(循环, 跳过禁用)/Home/End 移动高亮, Enter 选中, ESC 交给回调;
//   - 鼠标悬停高亮、点击选中; 选中后不自动关闭(由外层 Dialog/Popover 决定)。
//
// 高度: cy 传 0 时用 36 + 300(上游 max-h-[300px]); 传值时列表吃剩余高度。
//
// CommandDialog 没有单独做: 用 ui.NewDialog + 把 Command 放进 Dialog.Content() 即可
// (Dialog 自带遮罩/ESC/焦点陷阱), 语义与 shadcn 的 CommandDialog 一致。
type Command struct {
	*widget.LayoutEle

	row     *widget.LayoutEle
	edit    *widget.Edit
	list    *ScrollArea
	content *widget.LayoutEle
	empty   *Label

	groups      []*CommandGroup
	seps        []*widget.Element
	items       []*CommandItem // 建立顺序
	visible     []*CommandItem // 当前过滤结果
	active      int32          // visible 下标(-1 无)
	query       string
	built       bool
	dying       bool
	placeholder string
	emptyText   string

	changes   []func(query string)
	selects   []func(value string)
	escapes   []func()
	textColor uint32
}

// NewCommand 创建命令面板。cx 传 0 = 宽度铺满父级; cy 传 0 = 36+300。
func NewCommand(x, y, cx, cy int32, hParent int, opts ...CommandOption) *Command {
	if cy <= 0 {
		cy = commandInputH + commandListMaxH
	}
	if cy < commandInputH {
		cy = commandInputH
	}
	c := &Command{active: -1, placeholder: "输入命令或搜索...", emptyText: "没有找到结果。"}
	for _, o := range opts {
		o(c)
	}
	root := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if root == nil {
		return nil
	}
	c.LayoutEle = root
	root.EnableHorizon(false)
	root.EnableLayout(true)
	root.EnableAutoWrap(false)
	root.EnableBkTransparent(true)
	root.EnableDrawBorder(false)
	root.EnableFocus(false)
	root.SetSpace(0)
	root.AddEvent_Paint(c.onRootPaint)
	if cx <= 0 {
		root.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	} else {
		root.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, cx)
	}
	root.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, cy)

	// ---- 搜索行 ----
	c.row = widget.NewLayoutEle(0, 0, 0, commandInputH, root.Handle)
	c.row.EnableHorizon(true)
	c.row.EnableLayout(true)
	c.row.EnableAutoWrap(false)
	c.row.EnableBkTransparent(true)
	c.row.EnableDrawBorder(false)
	c.row.EnableFocus(false)
	c.row.SetAlignV(xcc.Layout_Align_Center)
	c.row.SetSpace(8)
	c.row.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	c.row.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, commandInputH)
	c.row.AddEvent_Paint(c.onRowPaint)

	icon := widget.NewElement(0, 0, 16, 16, c.row.Handle)
	if icon != nil {
		icon.EnableBkTransparent(true)
		icon.EnableDrawBorder(false)
		icon.EnableFocus(false)
		icon.EnableSwitchFocus(false)
		icon.LayoutItem_SetMargin(12, 0, 0, 0)
		icon.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 16)
		icon.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 16)
		icon.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
			p := xui.BeginPaint(hEle, hDraw)
			if p == nil {
				return 0
			}
			*pbHandled = true
			t := xui.Current()
			p.Icon(icons.IconSearch, &p.Rc, t.MutedForeground.ScaleAlpha(0.5), 2)
			return 0
		})
	}

	c.edit = widget.NewEdit(0, 0, 0, commandInputH, c.row.Handle)
	if c.edit != nil {
		c.edit.EnableBkTransparent(true)
		c.edit.EnableDrawBorder(false).EnableDrawFocus(false)
		c.edit.EnableAutoSelAll(false)
		c.edit.SetBorderSize(0, 0, 0, 0)
		c.edit.SetTextAlign(xcc.Edit_TextAlign_Flag_Left | xcc.Edit_TextAlign_Flag_Center_V)
		c.edit.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
		c.edit.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, commandInputH)
		c.edit.LayoutItem_SetMargin(0, 0, 12, 0)
		c.edit.AddEvent_Edit_Changed(func(hEle int, pbHandled *bool) int {
			c.query = c.edit.GetText()
			c.applyFilter()
			for _, fn := range c.changes {
				fn(c.query)
			}
			return 0
		})
		c.edit.AddEvent_KeyDown(c.onKeyDown)
	}

	// ---- 列表 ----
	c.list = NewScrollArea(0, 0, 0, 0, root.Handle)
	if c.list == nil {
		root.Destroy()
		return nil
	}
	c.list.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	c.list.LayoutItem_SetHeight(xcc.Layout_Size_Weight, 1)
	c.content = c.list.ContentStack(0)

	c.empty = NewLabel(0, 0, 0, commandEmptyH, c.emptyText, c.content.Handle)
	if c.empty != nil {
		c.empty.SetMuted().SetFontSize(14).SetAlignCenter()
		xui.ItemFill(c.empty)
		c.empty.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, commandEmptyH)
		c.empty.Show(false)
	}

	c.built = true
	c.applyTheme(xui.Current())
	xui.OnThemeChangeFor(root.Handle, func(t *xui.Theme) { c.applyTheme(t) })
	root.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		c.dying = true
		return 0
	})
	xui.AttachWindow(root.Handle)
	return c
}

// AddGroup 添加一个分组; label 为空时只有条目, 不画分组标题。
func (c *Command) AddGroup(label string) *CommandGroup {
	if c == nil {
		return nil
	}
	g := &CommandGroup{cmd: c}
	le := widget.NewLayoutEle(0, 0, 0, 0, c.content.Handle)
	if le == nil {
		return nil
	}
	g.LayoutEle = le
	le.EnableHorizon(false)
	le.EnableLayout(true)
	le.EnableAutoWrap(false)
	le.EnableBkTransparent(true)
	le.EnableDrawBorder(false)
	le.EnableFocus(false)
	le.SetSpace(0)
	le.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	le.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	le.LayoutItem_SetMargin(commandGroupPad, commandGroupPad, commandGroupPad, commandGroupPad)

	if label != "" {
		h := NewLabel(0, 0, 0, commandHeadingH, label, le.Handle)
		if h != nil {
			h.SetMuted().SetMedium().SetFontSize(12)
			xui.ItemFill(h)
			h.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, commandHeadingH)
			h.LayoutItem_SetMargin(8, 0, 8, 0)
		}
	}
	c.groups = append(c.groups, g)
	c.syncEmpty()
	return g
}

// AddSeparator 添加一条分隔线(过滤时自动隐藏)。
func (c *Command) AddSeparator() *Command {
	if c == nil {
		return c
	}
	se := widget.NewElement(0, 0, 0, 9, c.content.Handle)
	if se == nil {
		return c
	}
	se.EnableBkTransparent(true)
	se.EnableDrawBorder(false)
	se.EnableFocus(false)
	se.EnableSwitchFocus(false)
	se.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	se.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 9)
	se.LayoutItem_SetMargin(4, 0, 4, 0)
	se.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		commandDeviceLine(p, p.Rc, uint32(xui.Current().Border))
		return 0
	})
	c.seps = append(c.seps, se)
	c.applyFilter()
	return c
}

// Focus 把焦点放到搜索框。
func (c *Command) Focus() *Command {
	if c != nil && c.edit != nil {
		c.edit.SetFocus()
	}
	return c
}

// Query 返回当前搜索词。
func (c *Command) Query() string {
	if c == nil || c.edit == nil {
		return ""
	}
	return c.edit.GetText()
}

// SetQuery 设置搜索词并立即过滤。
func (c *Command) SetQuery(q string) *Command {
	if c == nil || c.edit == nil {
		return c
	}
	c.edit.SetText(q)
	c.query = q
	c.applyFilter()
	return c
}

// Clear 清空搜索词。
func (c *Command) Clear() *Command { return c.SetQuery("") }

// SetPlaceholder 更新占位文字。
func (c *Command) SetPlaceholder(s string) *Command {
	if c == nil || c.edit == nil {
		return c
	}
	c.placeholder = s
	c.edit.SetDefaultText(s)
	c.edit.SetDefaultTextColor(uint32(xui.Current().MutedForeground))
	c.edit.Redraw(false)
	return c
}

func (c *Command) applyTheme(t *xui.Theme) {
	if c == nil || c.edit == nil || c.dying {
		return
	}
	c.edit.SetFont(xui.FontHandle(t.FontDefault))
	c.edit.SetDefaultText(c.placeholder)
	c.edit.SetDefaultTextColor(uint32(t.MutedForeground))
	c.edit.SetCaretColor(uint32(t.Primary))
	if tc := uint32(t.PopoverForeground); tc != c.textColor {
		c.textColor = tc
		c.edit.SetTextColor(tc)
	}
	c.LayoutEle.Redraw(false)
}

func (c *Command) onRootPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	p.FillRound(&p.Rc, t.Popover, t.RadiusMd())
	return 0
}

func (c *Command) onRowPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	line := xc.RECT{Left: p.Rc.Left, Top: p.Rc.Bottom - 1, Right: p.Rc.Right, Bottom: p.Rc.Bottom}
	commandDeviceLine(p, line, uint32(xui.Current().Border))
	return 0
}

// commandDeviceLine 在 rect 的中心画一条对齐整设备像素的水平线。
// 与 separator.go 同一套算法: 1 逻辑像素在 150% DPI 下是 1.5 物理像素,
// 不自己对齐会出现同一面板里的线一粗一细。
func commandDeviceLine(p *xui.Painter, rc xc.RECT, color uint32) {
	sc := p.Scale
	if sc <= 0 {
		sc = 1
	}
	c := math.Floor(float64(rc.Top+rc.Bottom) / 2 * sc)
	rf := xc.RECTF{
		Left:   float32(rc.Left),
		Top:    float32(c / sc),
		Right:  float32(rc.Right),
		Bottom: float32((c + 1) / sc),
	}
	p.Draw.FillRectColorF(&rf, color)
}

// CommandGroup 命令分组。
type CommandGroup struct {
	*widget.LayoutEle
	cmd   *Command
	items []*CommandItem
}

// CommandItemOption 条目配置项。
type CommandItemOption func(*CommandItem)

// WithCommandItemIcon 设置条目前置图标(lucide path)。
func WithCommandItemIcon(path string) CommandItemOption {
	return func(i *CommandItem) { i.icon = path }
}

// WithCommandItemShortcut 设置条目右侧快捷键文字。
func WithCommandItemShortcut(s string) CommandItemOption {
	return func(i *CommandItem) { i.shortcut = s }
}

// WithCommandItemKeywords 设置额外的搜索关键词(不显示)。
func WithCommandItemKeywords(s string) CommandItemOption {
	return func(i *CommandItem) { i.keywords = s }
}

// WithCommandItemDisabled 设置条目禁用(可见但不可选)。
func WithCommandItemDisabled() CommandItemOption {
	return func(i *CommandItem) { i.disabled = true }
}

// WithCommandItemOnSelect 设置条目单独的回调(在 Command 的 OnSelect 之前调用)。
func WithCommandItemOnSelect(fn func(value string)) CommandItemOption {
	return func(i *CommandItem) { i.onselect = fn }
}

// AddItem 往分组里添加一个条目。
func (g *CommandGroup) AddItem(value, text string, opts ...CommandItemOption) *CommandItem {
	if g == nil || g.cmd == nil {
		return nil
	}
	it := &CommandItem{cmd: g.cmd, value: value, text: text}
	for _, o := range opts {
		o(it)
	}
	ele := widget.NewElement(0, 0, 0, commandItemH, g.Handle)
	if ele == nil {
		return nil
	}
	it.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, commandItemH)
	ele.AddEvent_Paint(it.onPaint)
	ele.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		it.hover()
		return 0
	})
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		it.hover()
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if it.disabled {
			return 0
		}
		it.hover()
		it.selectSelf()
		return 0
	})
	g.items = append(g.items, it)
	g.cmd.items = append(g.cmd.items, it)
	g.cmd.applyFilter()
	return it
}

// CommandItem 命令条目。
type CommandItem struct {
	*widget.Element
	cmd      *Command
	value    string
	text     string
	keywords string
	icon     string
	shortcut string
	disabled bool
	match    bool
	active   bool
	onselect func(value string)
}

// Value 返回条目的程序侧值。
func (i *CommandItem) Value() string { return i.value }

// Text 返回条目显示文字。
func (i *CommandItem) Text() string { return i.text }

// SetDisabled 设置禁用状态。
func (i *CommandItem) SetDisabled(v bool) *CommandItem {
	if i == nil {
		return i
	}
	i.disabled = v
	if v && i.active {
		i.setActive(false)
	}
	i.Redraw(false)
	return i
}

func (i *CommandItem) hover() {
	if i == nil || i.disabled {
		return
	}
	for idx, it := range i.cmd.visible {
		if it == i {
			i.cmd.setActive(int32(idx), false)
			return
		}
	}
}

func (i *CommandItem) selectSelf() {
	if i == nil || i.disabled {
		return
	}
	if i.onselect != nil {
		i.onselect(i.value)
	}
	for _, fn := range i.cmd.selects {
		fn(i.value)
	}
	if i.cmd.edit != nil {
		i.cmd.edit.SetFocus()
	}
}

func (i *CommandItem) setActive(on bool) {
	if i.active == on {
		return
	}
	i.active = on
	i.Redraw(false)
}

func (i *CommandItem) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	fg := t.PopoverForeground
	muted := t.MutedForeground
	if i.active && !i.disabled {
		p.FillRound(&p.Rc, t.Accent, t.RadiusSm())
		fg = t.AccentForeground
	}
	alpha := byte(255)
	if i.disabled {
		alpha = 128
	}
	fg = fg.WithAlpha(alpha)

	x := p.Rc.Left + 8
	if i.icon != "" {
		ir := xc.RECT{Left: x, Top: p.Rc.Top + 8, Right: x + 16, Bottom: p.Rc.Top + 24}
		ic := muted
		if i.active && !i.disabled {
			ic = t.AccentForeground
		}
		p.Icon(i.icon, &ir, ic.WithAlpha(alpha), 2)
		x += 24 // 16 图标 + 8 gap-2
	}
	right := p.Rc.Right - 8
	if i.shortcut != "" {
		sf := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 12})
		if sf != 0 {
			p.Draw.SetFont(sf)
		}
		sr := xc.RECT{Left: right - 80, Top: p.Rc.Top, Right: right, Bottom: p.Rc.Bottom}
		p.Text(i.shortcut, &sr, muted.WithAlpha(alpha),
			xcc.TextAlignFlag_Right|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		right -= 88
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	tr := xc.RECT{Left: x, Top: p.Rc.Top, Right: right, Bottom: p.Rc.Bottom}
	p.Text(i.text, &tr, fg,
		xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap|
			xcc.TextTrimming_EllipsisCharacter)
	return 0
}

// onKeyDown 接管搜索框的方向键/回车/ESC(cmdb 的键盘语义)。
func (c *Command) onKeyDown(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
	switch int(wParam) {
	case xcc.VK_Down:
		c.move(1)
		*pbHandled = true
	case xcc.VK_Up:
		c.move(-1)
		*pbHandled = true
	case xcc.VK_Home:
		c.moveToEdge(false)
		*pbHandled = true
	case xcc.VK_End:
		c.moveToEdge(true)
		*pbHandled = true
	case xcc.VK_PageDown:
		c.moveBy(5)
		*pbHandled = true
	case xcc.VK_PageUp:
		c.moveBy(-5)
		*pbHandled = true
	case 0x0D: // VK_RETURN
		c.selectActive()
		*pbHandled = true
	case xcc.VK_Esc:
		if len(c.escapes) > 0 {
			for _, fn := range c.escapes {
				fn()
			}
		} else if c.Query() != "" {
			c.Clear()
		}
		*pbHandled = true
	}
	return 0
}

func (c *Command) move(delta int) {
	if len(c.visible) == 0 {
		return
	}
	n := len(c.visible)
	if c.active < 0 || int(c.active) >= n {
		// 没有高亮时: ↓ 从第一个可选条目开始, ↑ 从最后一个开始
		if delta >= 0 {
			for i := 0; i < n; i++ {
				if !c.visible[i].disabled {
					c.setActive(int32(i), true)
					return
				}
			}
		} else {
			for i := n - 1; i >= 0; i-- {
				if !c.visible[i].disabled {
					c.setActive(int32(i), true)
					return
				}
			}
		}
		return
	}
	idx := int(c.active)
	for step := 0; step < n; step++ {
		idx = (idx + delta + n) % n
		if !c.visible[idx].disabled {
			c.setActive(int32(idx), true)
			return
		}
	}
}

func (c *Command) moveBy(delta int) {
	for k := 0; k < absInt(delta); k++ {
		c.move(signInt(delta))
	}
}

func (c *Command) moveToEdge(last bool) {
	if len(c.visible) == 0 {
		return
	}
	if !last {
		for idx, it := range c.visible {
			if !it.disabled {
				c.setActive(int32(idx), true)
				return
			}
		}
		return
	}
	for idx := len(c.visible) - 1; idx >= 0; idx-- {
		if !c.visible[idx].disabled {
			c.setActive(int32(idx), true)
			return
		}
	}
}

func (c *Command) selectActive() {
	if c.active < 0 || int(c.active) >= len(c.visible) {
		return
	}
	c.visible[c.active].selectSelf()
}

func (c *Command) setActive(idx int32, scroll bool) {
	if idx == c.active {
		if scroll && idx >= 0 && int(idx) < len(c.visible) {
			c.scrollTo(c.visible[idx])
		}
		return
	}
	if c.active >= 0 && int(c.active) < len(c.visible) {
		c.visible[c.active].setActive(false)
	}
	c.active = idx
	if idx >= 0 && int(idx) < len(c.visible) {
		c.visible[idx].setActive(true)
		if scroll {
			c.scrollTo(c.visible[idx])
		}
	}
}

// scrollTo 把条目滚进列表视口(内容坐标 = 条目屏幕矩形 - 内容区屏幕矩形)。
func (c *Command) scrollTo(it *CommandItem) {
	if c == nil || c.list == nil || it == nil || !it.IsShow() {
		return
	}
	var ir, cr xc.RECT
	xc.XEle_GetWndClientRect(it.Handle, &ir)
	xc.XEle_GetWndClientRect(c.content.Handle, &cr)
	top := ir.Top - cr.Top
	view := c.list.GetViewHeight()
	if view <= 0 {
		view = c.list.GetHeight()
	}
	if view <= 0 {
		return
	}
	pos := c.list.GetViewPosV()
	if top < pos {
		c.list.ScrollPosYV(top)
	} else if top+commandItemH > pos+view {
		c.list.ScrollPosYV(top + commandItemH - view)
	}
}

// applyFilter 按当前搜索词显示/隐藏条目与分组, 并重建 visible 列表。
func (c *Command) applyFilter() {
	if c == nil || !c.built || c.dying {
		return
	}
	q := strings.ToLower(strings.TrimSpace(c.query))
	c.visible = c.visible[:0]
	for _, g := range c.groups {
		any := false
		for _, it := range g.items {
			match := q == "" || strings.Contains(strings.ToLower(it.text), q) ||
				strings.Contains(strings.ToLower(it.keywords), q)
			it.match = match
			if it.match {
				any = true
				c.visible = append(c.visible, it)
			}
			it.Show(match)
			if !match {
				it.setActive(false)
			}
		}
		g.Show(any)
		if any {
			g.AdjustLayout()
		}
	}
	for _, se := range c.seps {
		se.Show(q == "" && len(c.visible) > 0)
	}
	if c.empty != nil {
		c.empty.Show(len(c.visible) == 0)
	}
	c.content.AdjustLayout()
	c.list.AdjustLayout()

	c.active = -1
	first := int32(-1)
	for idx, it := range c.visible {
		if !it.disabled {
			first = int32(idx)
			break
		}
	}
	if first >= 0 {
		c.setActive(first, false)
	}
	c.LayoutEle.Redraw(false)
}

func (c *Command) syncEmpty() {
	if c == nil || c.empty == nil {
		return
	}
	c.empty.Show(len(c.items) == 0)
	c.content.AdjustLayout()
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func signInt(v int) int {
	if v < 0 {
		return -1
	}
	return 1
}
