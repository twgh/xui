package ui

import (
	"strconv"
	"strings"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
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

// comboboxEmptyText 是过滤后一条都不剩时的空状态文案。
const comboboxEmptyText = "没有找到结果。"

// chips 几何(逻辑像素), 对齐 base-nova combobox:
// chip 高 21(h-[calc(--spacing(5.25))])、text-xs(12)、rounded-sm、px-1.5(6),
// 容器 gap-1(4) 且在有 chip 时左右内边距收到 4(has-data-[slot=combobox-chip]:px-1)。
// 上游是多行换行(min-h-8 + flex-wrap), 本库触发区高度固定, 放不下的折叠成 "+N"。
const (
	comboboxChipH    int32 = 21
	comboboxChipsPad int32 = 4 // 有 chip 时容器左右内边距(has-data-[slot=combobox-chip]:px-1)
	comboboxChipPad  int32 = 6
	comboboxChipGap  int32 = 4
	comboboxChipXSz  int32 = 12
	comboboxChipXHit int32 = 16
	comboboxChipTrk  int32 = 30 // 右侧箭头/清除钮的预留宽
	comboboxClearSz  int32 = 24
	// 输入框形态的裸 Edit 左右内缩(同触发区文字的 pl-2.5 / 右侧箭头预留 30)。
	comboboxInputPadL int32 = 10
	comboboxInputPadR int32 = 30
	// comboboxClearHoverKey 是清除钮的 hover 过渡键: chip 用下标, 这里取一个大偏移避免相撞。
	comboboxClearHoverKey int32 = 1000
)

// Combobox 自动补全选择框 —— 对应 shadcn 文档的 **Combobox**(输入过滤 + 建议列表),
// 与 ui.NativeComboBox(shadcn select 的原生下拉)不是同一个东西。触发形态:
//
//   - 默认 = 上游 Popup 示例: 触发区是 outline 按钮(同 Select), 显示当前值/占位符,
//     右侧 chevrons-up-down; 弹层 = 顶部搜索框(输入即过滤) + 建议列表 + 空状态;
//   - `WithComboboxInput` = 上游 Basic 示例: 触发区本身是可输入的原生输入框
//     (ComboboxInput 就在触发层), 边打字边过滤; 弹层里不再有搜索行, 只列条目。
//   - `WithComboboxEditable` = 可编辑组合框(本库扩展): 输入框只编辑值、不做搜索,
//     打字不过滤也不自动开层; 箭头/↓ 展开固定选项的全量列表(见下方段落)。
//
// 共通语义: 条目支持前置图标 / 禁用 / 分组标签 / 左侧(多选)或右侧(单选)选中勾;
// 键盘 ↑/↓ 循环移动高亮(跳过禁用)、Home/End 首末、Enter 选中、ESC 关闭; 鼠标悬停
// 即高亮, 点击选中; 单选选中即关闭; `WithComboboxMultiple` 多选时点击切换并保持打开,
// 触发区默认显示 "首个值 +N", `WithComboboxChips` 改为 chips 展示(每个 chip 带移除钮,
// 放不下的折叠成 "+N"; 搜索框为空时退格删末尾); `WithComboboxClear` 有选中值时在
// 触发区右侧显示清除钮(替代箭头), 点击清空; 过滤后默认自动高亮第一条匹配项,
// `WithComboboxAutoHighlight(false)` 关闭(同上游 base combobox 的默认);
// 条目显示与搜索用 Text, 为空时回落到 Value(对应上游 itemToStringLabel;
// Value 是程序侧值, 对应 itemToStringValue)。
//
// 输入框形态额外语义: 点输入框/↓ 打开列表并全选已选文字(直接打字是替换, 不会拼在
// 标签后面), 列表开着时输入框保持焦点(弹层不激活), 点条目照常选中; 把文字删空即
// 取消选中(上游 inputClear); ESC 关闭列表, 列表已关时 ESC 清值并清空文字;
// 关闭列表后输入框恢复显示已选标签(未提交的过滤词不保留)。
//
// `WithComboboxEditable`(本库扩展)把触发区换成**可编辑组合框**(仅单选): 输入框接受
// 任意文字, 打字不搜索、不过滤、也不自动开列表; 点右侧箭头或 ↓ 展开的是固定选项的
// 全量列表。文字在 Enter / 失焦 / 点触发区外时提交: 与某条目的 Text/Value 精确相等
// (不区分大小写)时选中该条目, 否则整段文字就是值(Text=Value=文字, OnChange 回传
// SelectItem{Value: 文字, Text: 文字}); 提交空文字 = 清空值; ESC 放弃未提交的文字。
//
// 弹层复用 popoverPanel(阴影窗 + 卡片 body + 失焦/ESC 关闭 + 工作区夹取),
// 数据模型与 Select 共用 SelectItem / SelectGroup。
type Combobox struct {
	*widget.Button
	groups            []SelectGroup
	rows              []*comboboxRow
	picked            map[string]bool
	multiple          bool
	chips             bool
	clear             bool
	autoHighlight     bool
	input             bool
	editable          bool
	placeholder       string
	searchPlaceholder string
	emptyText         string
	size              SelectSize
	invalid           bool
	disabled          bool

	inputEdit     *widget.Edit // 输入框形态的触发区裸 Edit(叠在 Button 上)
	suppressInput bool         // 程序化 SetText 期间屏蔽 Edit_Changed
	emptyPending  bool         // "删空即清值"的确认已排进下一轮消息
	hasCustom     bool         // 当前值是否来自可编辑输入(不在 groups 里)
	custom        SelectItem   // 可编辑输入提交的值(Text=Value=输入的文字)

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

	// 触发区 chips / 清除钮: 命中块在每次绘制时按当前尺寸重算, 悬停与点击共用。
	chipHits    []comboboxChipHit
	hoverChip   int32
	clearHit    xc.RECT
	hasClearHit bool
	hoverClear  bool
	hitsW       int32
	hitsH       int32

	changes []func(items []SelectItem)
}

// comboboxChipHit 是触发区里一个 chip 的矩形与移除钮命中块(value 为空 = 溢出 "+N" 标记)。
type comboboxChipHit struct {
	value string
	label string
	chip  xc.RECT
	x     xc.RECT // 移除钮命中块(hasX 为 false 时无效)
	hasX  bool
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

// WithComboboxChips 多选模式的触发区用 chips 展示已选项(默认 "首个值 +N")。
// 每个 chip 可点击右侧 × 单独移除, 搜索框为空时退格移除最后一个; 宽度不够的
// 折叠成 "+N"(上游是容器换行长高, 本库触发区高度固定)。
func WithComboboxChips() ComboboxOption {
	return func(c *Combobox) { c.chips = true }
}

// WithComboboxClear 有选中值时在触发区右侧显示清除钮(替代下拉箭头), 点击清空全部选中值
// (单选/多选都生效, 触发 OnChange)。没有选中值时仍显示箭头。
func WithComboboxClear() ComboboxOption {
	return func(c *Combobox) { c.clear = true }
}

// WithComboboxInput 把触发区换成可输入的原生输入框(上游 base combobox 的 Basic 形态:
// ComboboxInput 就是触发层, 边打字边过滤, 弹层里只列条目); 默认形态是"按钮触发 +
// 弹层搜索框"(上游 Popup 形态)。仅用于单选 —— 与 WithComboboxMultiple / WithComboboxChips
// 一起用时忽略本项, 多选仍在弹层搜索框里输入。
func WithComboboxInput() ComboboxOption {
	return func(c *Combobox) { c.input, c.editable = true, false }
}

// WithComboboxEditable 把触发区换成可编辑组合框(本库扩展, 仅单选): 输入框接受任意
// 文字, 打字不搜索、不过滤、也不自动打开列表; 点右侧箭头或 ↓ 展开固定选项的全量列表,
// 条目照常可点选。文字在 Enter / 失焦 / 点触发区外时提交 —— 与某条目的 Text/Value
// 精确相等(不区分大小写)时选中该条目, 否则整段文字作为自定义值(Value/Text 都是这段
// 文字, OnChange 回传 SelectItem{Value: 文字, Text: 文字}); 提交空文字 = 清空值;
// ESC 放弃未提交的文字(恢复原来的值)。
// 与 WithComboboxInput 互斥, 后写的那个生效 —— 前者是"边打字边过滤"的自动补全,
// 本项是"只编辑值、不搜索"的经典可编辑组合框。多选模式忽略本项。
func WithComboboxEditable() ComboboxOption {
	return func(c *Combobox) { c.input, c.editable = true, true }
}

// WithComboboxAutoHighlight 过滤后是否自动高亮第一条匹配项。默认 true(本库既有语义,
// 打完字直接 Enter 即选首条); 传 false 对齐上游 base combobox 的默认 —— 打字不移动高亮,
// 按 ↓ 或悬停才出现。
func WithComboboxAutoHighlight(on bool) ComboboxOption {
	return func(c *Combobox) { c.autoHighlight = on }
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
		autoHighlight:     true,
		placeholder:       "请选择...",
		searchPlaceholder: "搜索...",
		emptyText:         comboboxEmptyText,
		hoverChip:         -1,
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
		// 清除钮 / chip 的 × 优先于"打开弹层": 两者都在触发区内部, 命中后只改值不开层。
		if c.clearHitAt(pPt.X, pPt.Y) {
			c.clearPicked()
			return 0
		}
		if v, ok := c.chipRemoveAt(pPt.X, pPt.Y); ok {
			c.removePicked(v)
			return 0
		}
		// 弹层开着时点触发区: 弹层先因失焦关闭(更新了 lastClose), 这里不再重开 ——
		// 否则会出现"点两次才关掉"(同 Select / DatePicker)。输入框形态的弹层不抢焦点,
		// 点右侧箭头区收不到那个"先关"的信号, 直接在这里收起。
		if c.panel != nil && (c.panel.IsOpen() || c.panel.RecentlyClosed()) {
			if c.input && c.panel.IsOpen() {
				if c.editable {
					c.commitEditable() // 箭头收起列表时, 输入框里没提交的文字照常落值
				}
				c.Close()
			}
			return 0
		}
		c.Open()
		return 0
	})
	btn.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		c.hoverTrigger(pPt.X, pPt.Y)
		return 0
	})
	btn.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		if c.hoverChip != -1 || c.hoverClear {
			c.hoverChip, c.hoverClear = -1, false
			c.Redraw(false)
		}
		return 0
	})
	btn.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		c.dying = true
		if c.panel != nil {
			c.panel.Destroy()
		}
		c.panel, c.scroll, c.stack, c.edit, c.empty = nil, nil, nil, nil, nil
		c.inputEdit = nil
		return 0
	})
	if c.input && !c.multiple {
		c.setupInput()
	} else {
		c.input, c.editable = false, false // 输入框形态只对单选成立(多选在弹层搜索框里输入)
	}
	if c.disabled {
		c.SetDisabled(true)
	}
	xui.AttachWindow(btn.Handle)
	return c
}

// setupInput 在按钮触发区上叠一个透明原生 Edit(上游 Basic 形态: 输入框就是触发层)。
// Button 仍画边框/箭头/清除钮, Edit 只负责文字、光标与 IME; Button 不是布局容器,
// 所以子控件位置用 SetRectEx 跟随触发区尺寸(同 TagInput 的裸输入做法)。
func (c *Combobox) setupInput() {
	e := widget.NewEdit(0, 0, comboboxChipTrk+20, c.GetHeight(), c.Handle)
	if e == nil {
		c.input = false
		return
	}
	c.inputEdit = e
	e.EnableMultiLine(false)
	e.EnableBkTransparent(true)
	e.EnableDrawBorder(false).EnableDrawFocus(false)
	e.EnableAutoSelAll(false)
	e.EnableAutoShowScrollBar(false)
	if h := e.GetScrollBarHObj(); h != nil {
		h.Show(false)
	}
	if v := e.GetScrollBarVObj(); v != nil {
		v.Show(false)
	}
	e.SetBorderSize(0, 0, 0, 0)
	e.SetTextAlign(xcc.Edit_TextAlign_Flag_Left | xcc.Edit_TextAlign_Flag_Center_V)
	e.AddEvent_Edit_Changed(func(hEle int, pbHandled *bool) int {
		c.onInputChanged()
		return 0
	})
	e.AddEvent_KeyDown(c.onInputKeyDown)
	e.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		if !c.editable {
			// 自动补全: 点输入框即开列表, 并全选已选标签(直接打字是替换)
			c.Open()
			c.selectAllInput()
		}
		// 可编辑组合框: 点文字区只放光标/选中文字, 列表由右侧箭头或 ↓ 打开
		return 0
	})
	e.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		c.Redraw(false)
		return 0
	})
	e.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		// 输入框形态的弹层不激活(activate=false), 没有自己的失焦关闭信号 ——
		// 关层由触发区输入框失焦驱动: 点窗口内其它控件 / 切到别的应用都走这里。
		if c.editable {
			c.commitEditable() // 可编辑组合框: 输入完直接点别处也算提交
		}
		if c.IsOpen() {
			c.Close()
		}
		c.Redraw(false)
		return 0
	})
	e.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		c.Redraw(false) // 让按钮的暗色悬停底色跟上 Edit 的 hover
		return 0
	})
	e.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		c.Redraw(false)
		return 0
	})
	c.Button.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		c.syncInputRect()
		return 0
	})
	// 输入框形态的弹层不激活窗口(否则抢走输入焦点), 收不到"弹层失焦"信号;
	// 点宿主窗口里触发区以外的位置时手动收起(点弹层自身走不到这里, 它是独立窗口)。
	if hWnd := xc.XWidget_GetHWINDOW(c.Handle); hWnd > 0 {
		if w := window.NewByHandle(hWnd); w != nil {
			w.AddEvent_LButtonDown(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
				c.closeOnOutsidePress(hWindow, pPt)
				return 0
			})
		}
	}
	xui.OnThemeChangeFor(c.Button.Handle, func(t *xui.Theme) {
		c.applyInputTheme(t)
		c.Redraw(false)
	})
	c.applyInputTheme(xui.Current())
	c.syncInputRect()
	c.syncInputText()
	// 焦点属于输入框: 按钮自身不再进 Tab 顺序, 焦点环按 Edit 的焦点画
	c.Button.EnableFocus(false)
	c.Button.EnableSwitchFocus(false)
}

// syncInputRect 把裸 Edit 摆到触发区文字的位置(左 pl-2.5, 右侧给箭头/清除钮留 30)。
func (c *Combobox) syncInputRect() {
	if c == nil || c.inputEdit == nil {
		return
	}
	var rc xc.RECT
	xc.XEle_GetClientRect(c.Handle, &rc)
	w, h := rc.Right-rc.Left, rc.Bottom-rc.Top
	if w <= 0 || h <= 0 {
		return
	}
	left := rc.Left + comboboxInputPadL
	right := rc.Right - comboboxInputPadR
	if right-left < 24 {
		right = left + 24
	}
	c.inputEdit.SetRectEx(left, rc.Top, right-left, h, false, xcc.AdjustLayout_No)
}

// applyInputTheme 把主题字体/文字色/占位符色/插入符色下发给原生 Edit(原生 API 不吃令牌)。
func (c *Combobox) applyInputTheme(t *xui.Theme) {
	e := c.inputEdit
	if e == nil {
		return
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		e.SetFont(f)
	}
	fore, ph := t.Foreground, t.MutedForeground
	if c.disabled {
		fore, ph = fore.Mix(t.Card, 0.5), ph.Mix(t.Card, 0.5)
	}
	e.SetTextColor(uint32(fore))
	e.SetDefaultText(c.placeholder)
	e.SetDefaultTextColor(uint32(ph))
	e.SetCaretColor(uint32(t.Primary))
}

// syncInputText 关闭列表后把输入框恢复成"已选标签 / 空", 未提交的过滤词不保留。
func (c *Combobox) syncInputText() {
	if c == nil || c.inputEdit == nil {
		return
	}
	text := ""
	if !c.multiple {
		if it, ok := c.Selected(); ok {
			text = selectItemLabel(it)
		}
	}
	if c.inputEdit.GetText() == text {
		return
	}
	c.suppressInput = true
	c.inputEdit.SetText(text)
	// 原生 Edit 也不自动刷新(与其它元素一致): SetText 后补 Redraw, 聚焦态光标用 MoveEnd 归位
	c.inputEdit.MoveEnd()
	c.inputEdit.Redraw(false)
	c.suppressInput = false
}

// selectAllInput 全选输入框文字: 点开列表后直接打字是替换已选标签, 不会拼在标签后面。
// 推迟到下一轮消息 —— 鼠标按下的原生选区落在事件之后, 同步 SelectAll 会被覆盖。
func (c *Combobox) selectAllInput() {
	if c == nil || c.inputEdit == nil {
		return
	}
	app.AfterFunc(0, func() {
		if c.inputEdit != nil && xc.XC_IsHXCGUI(c.inputEdit.Handle, xcc.XC_ELE) {
			c.inputEdit.SelectAll()
		}
	})
}

// closeOnOutsidePress 宿主窗口按下鼠标时: 可编辑组合框把输入框里没提交的文字落值
// (点击不用落在可聚焦控件上, 否则 Edit 不失焦、文字永远不提交), 弹层开着时一并收起。
func (c *Combobox) closeOnOutsidePress(hWindow int, pt *xc.POINT) {
	if c == nil || pt == nil || c.inputEdit == nil {
		return
	}
	if !xc.XC_IsHXCGUI(c.Handle, xcc.XC_ELE) || xc.XWidget_GetHWINDOW(c.Handle) != hWindow {
		return
	}
	var rc xc.RECT
	xc.XEle_GetWndClientRect(c.Handle, &rc)
	if pt.X >= rc.Left && pt.X < rc.Right && pt.Y >= rc.Top && pt.Y < rc.Bottom {
		return
	}
	if c.editable {
		c.commitEditable() // 点触发区外视同"输入完了"
	}
	if c.IsOpen() {
		c.Close()
	}
}

// onInputChanged 自动补全形态: 边打字边过滤; 删空文字即取消选中(上游 inputClear);
// 列表没开就一起打开(弹层不激活, 焦点一直留在触发区输入框)。
// 可编辑组合框不做任何搜索: 打字只是改文字, 列表保持全量、也不自动打开。
func (c *Combobox) onInputChanged() {
	if c == nil || c.inputEdit == nil || c.suppressInput {
		return
	}
	if c.editable {
		return
	}
	c.query = c.inputEdit.GetText()
	if strings.TrimSpace(c.query) == "" && len(c.picked) > 0 {
		c.clearIfInputEmpty()
	}
	if !c.IsOpen() {
		c.showInputPopup()
	} else {
		c.applyFilter()
	}
}

// clearIfInputEmpty 处理"输入框删空 = 取消选中"(上游 inputClear)。判定推迟到下一轮
// 消息: 原生 Edit 替换整段选中文字时会先报一次空文本、再报新文本(实测), 立即清值
// 会把"打字替换已选标签"误判成用户删空; 下一轮再看仍是空, 才是真的删空。
func (c *Combobox) clearIfInputEmpty() {
	if c == nil || c.inputEdit == nil || c.emptyPending {
		return
	}
	c.emptyPending = true
	app.AfterFunc(0, func() {
		if c == nil {
			return
		}
		c.emptyPending = false
		if c.inputEdit == nil || c.suppressInput || !xc.XC_IsHXCGUI(c.inputEdit.Handle, xcc.XC_ELE) {
			return
		}
		if len(c.picked) == 0 || strings.TrimSpace(c.inputEdit.GetText()) != "" {
			return
		}
		c.clearPicked()
		c.applyFilter() // 选中值没了, 高亮回落到第一条(不再停在刚清掉的那项)
	})
}

// onInputKeyDown 输入框形态的键盘: 列表开着时 ↑/↓ 移动、Enter 选中、ESC 关闭;
// 列表没开时 ↓ 打开(可编辑组合框顺带点亮第一项), Enter 提交文字;
// 自动补全形态关闭态的 ESC 清值(上游 base combobox), 可编辑形态 ESC 放弃未提交的文字。
// Home/End 与退格都不抢 —— 上游输入框里它们是光标/删除操作; 自动补全的"删空文字 =
// 清值"由 onInputChanged 处理, 在这里拦退格反而会吃掉原生删除、让 Changed 不触发。
func (c *Combobox) onInputKeyDown(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
	switch int(wParam) {
	case xcc.VK_Down:
		if c.IsOpen() {
			c.moveActive(1)
		} else if c.editable {
			c.Open()
			c.setActive(c.firstEnabled(0, 1), true) // 打开不预亮(见 initialActive), ↓ 明确要选第一项
		} else {
			c.Open()
		}
		*pbHandled = true
	case xcc.VK_Up:
		if c.IsOpen() {
			c.moveActive(-1)
			*pbHandled = true
		}
	case xcc.VK_Enter:
		if c.IsOpen() {
			if c.active >= 0 && int(c.active) < len(c.rows) {
				c.chooseRow(c.rows[c.active])
			} else if c.editable {
				// 可编辑组合框打开时不预亮条目: 没有高亮就是把输入的文字提交为值
				c.commitEditable()
				c.Close()
			}
			*pbHandled = true
		} else if c.editable {
			c.commitEditable()
			*pbHandled = true
		}
	case xcc.VK_Esc:
		if c.IsOpen() {
			c.Close()
		} else if c.editable {
			c.syncInputText() // 放弃未提交的文字, 恢复原来的值
		} else if len(c.picked) > 0 {
			c.clearPicked()
		}
		*pbHandled = true
	}
	return 0
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
	c.invalidateTriggerHits()
	c.Redraw(false)
	c.syncInputText()
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
	if c.hasCustom && c.picked[c.custom.Value] {
		return c.custom.Value
	}
	return ""
}

// Values 返回全部选中值(按数据顺序)。
func (c *Combobox) Values() []string {
	if c == nil {
		return nil
	}
	var out []string
	for _, it := range c.pickedItems() {
		out = append(out, it.Value)
	}
	return out
}

// pickedItems 按数据顺序返回已选中条目(触发区文字 / chips / OnChange 共用);
// 可编辑组合框提交的自定义值不在数据里, 追加在最后。
func (c *Combobox) pickedItems() []SelectItem {
	if c == nil {
		return nil
	}
	var out []SelectItem
	for _, g := range c.groups {
		for _, it := range g.Items {
			if c.picked[it.Value] {
				out = append(out, it)
			}
		}
	}
	if c.hasCustom && c.picked[c.custom.Value] {
		out = append(out, c.custom)
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
	if c.hasCustom && c.custom.Value == v && v != "" {
		return c.custom, true
	}
	return SelectItem{}, false
}

// SetValue 按值选中条目(单选语义: 先清空其它选中), 不触发 OnChange。
// 可编辑模式下, 数据里没有的值会以自定义值保留(Text=Value=v)。
func (c *Combobox) SetValue(v string) *Combobox {
	if c == nil {
		return c
	}
	c.picked = map[string]bool{}
	c.hasCustom, c.custom = false, SelectItem{}
	if v != "" {
		c.picked[v] = true
		if c.editable && !c.hasItemValue(v) {
			c.hasCustom, c.custom = true, SelectItem{Value: v, Text: v}
		}
	}
	c.syncPicked()
	c.invalidateTriggerHits()
	c.Redraw(false)
	c.syncInputText()
	c.refreshRows()
	return c
}

// SetValues 设置多选值, 不触发 OnChange。
func (c *Combobox) SetValues(vs ...string) *Combobox {
	if c == nil {
		return c
	}
	c.picked = map[string]bool{}
	c.hasCustom, c.custom = false, SelectItem{}
	for _, v := range vs {
		c.picked[v] = true
	}
	c.syncPicked()
	c.invalidateTriggerHits()
	c.Redraw(false)
	c.syncInputText()
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

// syncPicked 清掉数据里已经不存在的值; 可编辑组合框提交的自定义值单独保留,
// 数据里出现同值条目时以数据条目为准(自定义值退场, 选中的 value 不变)。
func (c *Combobox) syncPicked() {
	exist := map[string]bool{}
	for _, g := range c.groups {
		for _, it := range g.Items {
			exist[it.Value] = true
		}
	}
	if c.hasCustom {
		// 未被选中或数据里已经有同值条目: 自定义值不再需要
		if !c.picked[c.custom.Value] || exist[c.custom.Value] {
			c.hasCustom, c.custom = false, SelectItem{}
		}
	}
	for v := range c.picked {
		if !exist[v] && !(c.hasCustom && v == c.custom.Value) {
			delete(c.picked, v)
		}
	}
}

// hasItemValue 报告数据里是否有该 value(精确匹配)。
func (c *Combobox) hasItemValue(v string) bool {
	for _, g := range c.groups {
		for _, it := range g.Items {
			if it.Value == v {
				return true
			}
		}
	}
	return false
}

// itemByText 按显示文字或程序值精确查找条目(不区分大小写), 可编辑组合框提交时用。
func (c *Combobox) itemByText(text string) (SelectItem, bool) {
	lt := strings.ToLower(text)
	for _, g := range c.groups {
		for _, it := range g.Items {
			if strings.ToLower(it.Value) == lt || strings.ToLower(selectItemLabel(it)) == lt {
				return it, true
			}
		}
	}
	return SelectItem{}, false
}

// setTypedValue 把一段文字落成选中值: 精确命中已有条目时选该条目(不覆盖为自定义值),
// 否则作为自定义值(Text=Value=文字)。返回是否真的改动了状态。
func (c *Combobox) setTypedValue(text string) bool {
	if c == nil || c.multiple {
		return false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if it, ok := c.itemByText(text); ok {
		if it.Disabled {
			return false
		}
		if len(c.picked) == 1 && c.picked[it.Value] && !c.hasCustom {
			return false
		}
		c.picked = map[string]bool{it.Value: true}
		c.hasCustom, c.custom = false, SelectItem{}
		return true
	}
	if c.hasCustom && c.custom.Value == text && c.picked[text] {
		return false
	}
	c.hasCustom = true
	c.custom = SelectItem{Value: text, Text: text}
	c.picked = map[string]bool{text: true}
	return true
}

// commitEditable 把可编辑组合框输入框里的文字提交为选中值(Enter / 失焦 / 点触发区外):
// 空文字 = 清空值; 否则见 setTypedValue。返回是否真的改动了选中值。
func (c *Combobox) commitEditable() bool {
	if c == nil || c.inputEdit == nil || !c.editable {
		return false
	}
	if strings.TrimSpace(c.inputEdit.GetText()) == "" {
		if len(c.picked) == 0 {
			c.syncInputText() // 只有空白: 把输入框恢复成空(去掉空格)
			return false
		}
		c.clearPicked()
		return true
	}
	if !c.setTypedValue(c.inputEdit.GetText()) {
		c.syncInputText() // 没改动(或精确命中禁用条目): 文字恢复成已选标签
		return false
	}
	c.invalidateTriggerHits()
	c.Redraw(false)
	c.refreshRows()
	c.syncInputText()
	c.notify()
	return true
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
	if c.inputEdit != nil {
		c.inputEdit.Enable(!v)
		// Enable(false) 只挡输入, 点上去 xcgui 仍会给它焦点并显示光标(同 TagInput 的坑)
		c.inputEdit.EnableFocus(!v)
		c.inputEdit.EnableSwitchFocus(!v)
		c.applyInputTheme(xui.Current())
	}
	if v && c.panel != nil {
		c.panel.Hide()
	}
	c.Redraw(false)
	return c
}

// Disabled 返回触发区是否禁用。
func (c *Combobox) Disabled() bool { return c != nil && c.disabled }

// SetFieldState 实现 FieldControl: 由 Field 级联校验失败/禁用态。
func (c *Combobox) SetFieldState(invalid, disabled bool) {
	c.SetInvalid(invalid)
	c.SetDisabled(disabled)
}

// FieldDisabled 实现 FieldControl。
func (c *Combobox) FieldDisabled() bool { return c == nil || c.Disabled() }

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

// Open 打开弹层(内容按需构建/重建)。按钮触发形态: 焦点落到弹层搜索框, 每次从空搜索词
// 开始; 输入框形态: 搜索词清空、已选标签保留在输入框里, 列表显示全量(上游点开即全量,
// 打字才开始过滤), 焦点留在触发区输入框。
func (c *Combobox) Open() *Combobox {
	if c == nil || c.disabled || (c.panel != nil && c.panel.IsOpen()) {
		return c
	}
	if c.input {
		c.query = ""
		return c.showInputPopup()
	}
	if !c.built {
		c.build()
	}
	if c.panel == nil {
		return c
	}
	// 失焦关闭走的是 popoverPanel.Hide(没经过 c.Close), 行元素一直留着;
	// 显示前先清一次高亮, 免得旧行在 applyFilter 之前先画出来。
	c.clearActive()
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

// showInputPopup 是输入框形态的打开路径: 不清输入框文字(那是已选标签或正在输入的内容),
// 只按当前 query 过滤; 弹层不抢焦点, 键盘继续由触发区输入框接收。
func (c *Combobox) showInputPopup() *Combobox {
	if c == nil || c.disabled || c.IsOpen() {
		return c
	}
	if !c.built {
		c.build()
	}
	if c.panel == nil {
		return c
	}
	c.clearActive()
	x, y := popupPosition(c.Handle, c.panel.cw, c.panel.ch, PopoverSideBottom, PopoverAlignStart, 4)
	c.panel.ShowAt(c.Handle, x, y)
	c.updateAnchorSide(y, c.panel.ch)
	c.applyFilter()
	return c
}

// SetQuery 设置搜索词并立即过滤(程序化驱动, 等价于用户在搜索框里输入)。
// 可编辑组合框没有搜索功能, 不适用(要设值用 SetValue)。
func (c *Combobox) SetQuery(q string) *Combobox {
	if c == nil || c.editable {
		return c
	}
	c.query = q
	if c.inputEdit != nil {
		c.suppressInput = true
		c.inputEdit.SetText(q)
		c.inputEdit.MoveEnd()
		c.inputEdit.Redraw(false)
		c.suppressInput = false
	}
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
		// 关闭时就把高亮清掉: 弹层只是 Hide, 行元素留着, 下次 ShowAt 到
		// applyFilter 之间若有重绘, 不会闪一帧上一次的高亮行。
		c.clearActive()
		c.panel.Hide()
	}
	if c != nil && c.input {
		c.syncInputText()
		c.query = "" // 关闭后输入框显示的是标签, 未提交的过滤词不再算"当前搜索词"
	}
	return c
}

// IsOpen 返回弹层是否打开。
func (c *Combobox) IsOpen() bool { return c != nil && c.panel != nil && c.panel.IsOpen() }

// build 构建弹层: 搜索行(可选) + 列表 + 空状态。
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
	ch := comboboxPad*2 + c.headerH() + listH

	// 输入框形态的弹层不能激活窗口: 一激活就把触发区输入框的焦点抢走,
	// 输入框失焦又会立刻把自己的弹层关掉(光标消失 + 列表闪一下)。
	panel := newPopoverPanelEx(cw, ch, c.Handle, false, !c.input)
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

	// ---- 搜索行(输入框形态没有: 输入就在触发区) ----
	var row *widget.LayoutEle
	if !c.input {
		row = widget.NewLayoutEle(0, 0, 0, comboboxSearchH, root.Handle)
	}
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
	ch := comboboxPad*2 + c.headerH() + listH
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
	case xcc.VK_Backspace:
		// 对齐 base combobox chips: 搜索词为空时退格移除最后一个已选项。
		if c.multiple && c.chips && strings.TrimSpace(c.query) == "" {
			if picked := c.pickedItems(); len(picked) > 0 {
				c.removePicked(picked[len(picked)-1].Value)
				*pbHandled = true
			}
		}
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

// rowOfPicked 返回当前选中值对应的可见可高亮行(多选取数据顺序里第一个已选中的),
// 没有选中值或选中值不在当前结果里返回 -1。
func (c *Combobox) rowOfPicked() int32 {
	for i, r := range c.rows {
		if r.kind != comboboxRowItem || !r.shown || r.item.Disabled {
			continue
		}
		if c.picked[r.item.Value] {
			return int32(i)
		}
	}
	return -1
}

// initialActive 决定打开/过滤后的高亮落点:
//   - 没有搜索词时优先落在当前选中项(同 Select 打开即定位选中项, 也对应 cmdk
//     用 value 驱动的 data-selected), 避免"已选 Nuxt.js、灰底却在 Next.js"这种
//     勾与高亮对不上的状态, 更不会让一次误按 Enter 把值改成第一项;
//   - 有搜索词且 autoHighlight 开着时落到第一条匹配项(输入即过滤, Enter 直接选中);
//     关掉(对齐上游 base combobox 默认)则不高亮, 由 ↓/Home/End 或悬停点亮。
//   - 可编辑组合框没有搜索词、文字也可能与已选值不同: 打开时不预亮任何条目,
//     把 Enter 留给"提交输入的文字"; 要选条目按 ↓ 或悬停。
func (c *Combobox) initialActive() int32 {
	if strings.TrimSpace(c.query) == "" {
		if c.editable {
			return -1
		}
		if i := c.rowOfPicked(); i >= 0 {
			return i
		}
		return c.firstEnabled(0, 1)
	}
	if !c.autoHighlight {
		return -1
	}
	return c.firstEnabled(0, 1)
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

// clearActive 清掉所有条目的高亮标记与键盘高亮下标。
// 只把 c.active 置 -1 不够: paintRow 看的是 r.active, 旧行的 true 会残留,
// 下一次 setActive 又点亮一行 —— 重开弹层或改搜索词时会同时出现两条 bg-accent。
func (c *Combobox) clearActive() {
	for _, r := range c.rows {
		if !r.active {
			continue
		}
		r.active = false
		if r.ele != nil {
			r.ele.Redraw(false)
		}
	}
	c.active = -1
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
		c.hasCustom, c.custom = false, SelectItem{}
	}
	c.hoverChip = -1
	c.invalidateTriggerHits()
	c.Redraw(false)
	if c.edit != nil {
		c.edit.Redraw(false)
	}
	c.notify()
	if !c.multiple {
		if c.input {
			c.syncInputText() // 输入框形态回填标签, 不等弹层淡出
		}
		c.Close()
		c.focusTrigger()
	}
}

// focusTrigger 把焦点还给触发区: 输入框形态给裸 Edit, 按钮形态给按钮自身。
func (c *Combobox) focusTrigger() {
	if c.input && c.inputEdit != nil {
		c.inputEdit.SetFocus()
		return
	}
	c.SetFocus()
}

// notify 触发 OnChange(按数据顺序回传全部选中条目)。
func (c *Combobox) notify() {
	if len(c.changes) == 0 {
		return
	}
	items := c.pickedItems()
	for _, fn := range c.changes {
		fn(items)
	}
}

// removePicked 移除一个选中值并通知(触发区 chips 的 × / 搜索框退格用)。
func (c *Combobox) removePicked(v string) {
	if c == nil || !c.picked[v] {
		return
	}
	delete(c.picked, v)
	if c.hasCustom && c.custom.Value == v {
		c.hasCustom, c.custom = false, SelectItem{}
	}
	c.hoverChip = -1
	c.invalidateTriggerHits()
	c.Redraw(false)
	c.syncInputText()
	c.refreshRows()
	c.notify()
}

// clearPicked 清空全部选中值并通知(清除钮用; SetValue/SetValues 仍不触发 OnChange)。
func (c *Combobox) clearPicked() {
	if c == nil || len(c.picked) == 0 {
		return
	}
	c.picked = map[string]bool{}
	c.hasCustom, c.custom = false, SelectItem{}
	c.hoverChip = -1
	c.invalidateTriggerHits()
	c.Redraw(false)
	c.syncInputText()
	c.refreshRows()
	c.notify()
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
		label := strings.ToLower(selectItemLabel(r.item))
		match := q == "" || strings.Contains(label, q) ||
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
	c.clearActive()
	c.setActive(c.initialActive(), false)
}

// headerH 是弹层顶部搜索行占的高度: 输入框形态没有搜索行(输入在触发区)。
func (c *Combobox) headerH() int32 {
	if c.input {
		return 0
	}
	return comboboxSearchH
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
	ch := comboboxPad*2 + c.headerH() + listH
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

// naturalWidth 量出 w-fit 触发区宽: pl-2.5 + 内容 + gap-1.5 + 箭头 + pr-2;
// chips 模式按 chip 宽累加(容器内距 4 + gap 4 + 右侧预留 30)。
func (c *Combobox) naturalWidth() int32 {
	t := xui.Current()
	w := int32(10 + 6 + 18 + 8)
	if picked := c.pickedItems(); c.multiple && c.chips && len(picked) > 0 {
		f := comboboxChipFont(t)
		w = comboboxChipsPad + comboboxChipTrk
		for i, it := range picked {
			if i > 0 {
				w += comboboxChipGap
			}
			w += comboboxChipWidth(selectItemLabel(it), f, true)
		}
	} else if text := c.triggerText(); text != "" {
		w += xui.TextWidth(text, xui.FontHandle(t.FontDefault))
	} else if c.placeholder != "" {
		w += xui.TextWidth(c.placeholder, xui.FontHandle(t.FontDefault))
	}
	if w < 100 {
		w = 100
	}
	return w
}

// triggerText 触发区显示的文字: 多选 "首个值 +N", 单选选中项, 否则空(显示占位符)。
func (c *Combobox) triggerText() string {
	picked := c.pickedItems()
	if len(picked) == 0 {
		return ""
	}
	if len(picked) == 1 {
		return selectItemLabel(picked[0])
	}
	return selectItemLabel(picked[0]) + " +" + strconv.Itoa(len(picked)-1)
}

// onPaint 触发区: 与 Select/NativeComboBox 同一套规则(border-input / focus-visible 环 /
// aria-invalid / 暗色 bg-input/30 悬停加深 / disabled)。
func (c *Combobox) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)
	if c.input && c.inputEdit != nil {
		// 输入框叠在按钮上层, 焦点/悬停/按下都发生在 Edit 上; 焦点环按 Edit 的焦点画
		// (shadcn 的 Input 是 :focus-visible, 文本输入框任何聚焦都算).
		es := xui.StateOf(c.inputEdit.Handle)
		st.Focused = es.Focused
		st.Hover = st.Hover || es.Hover
		st.Pressed = st.Pressed || es.Pressed
	}

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
	case st.Focused && !st.Disabled && (c.input || xui.FocusedByKeyboard()):
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
	picked := c.pickedItems()
	showChips := c.multiple && c.chips && len(picked) > 0
	if showChips {
		c.chipHits = c.layoutChips(t, p.Rc, picked)
		c.hitsW, c.hitsH = p.W(), p.H()
		c.paintChips(p, t, hEle, bg, fore, st.Disabled)
	} else if !c.input {
		// 输入框形态的文字/占位符由原生 Edit 画在上层, 这里再画就是双份(会发虚)。
		c.chipHits = nil
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
				Left:   float32(p.Rc.Left + comboboxInputPadL), // pl-2.5
				Top:    float32(p.Rc.Top) - dy,
				Right:  float32(p.Rc.Right - comboboxInputPadR),
				Bottom: float32(p.Rc.Bottom) - dy,
			}
			p.TextF(text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
		}
	}
	// 右侧: 有选中值且开了 WithComboboxClear 时画清除钮(上游 base combobox 的
	// ComboboxClear, 只在该有值时渲染), 并让位隐藏下拉箭头; 否则画 chevrons-up-down。
	if c.clear && len(picked) > 0 {
		rc := comboboxClearRect(p.Rc)
		c.clearHit, c.hasClearHit = rc, true
		hv := xui.HoverItem(hEle, comboboxClearHoverKey, !st.Disabled && c.hoverClear)
		if hv > 0 && !st.Disabled {
			hr := comboboxButtonRadius(t)
			hover := t.Muted
			if t.IsDark() {
				hover = t.Muted.ScaleAlpha(0.5) // dark:hover:bg-muted/50(ghost 按钮)
			}
			p.FillRound(&rc, hover.ScaleAlpha(hv), hr)
		}
		col := xui.Fade(t.MutedForeground, t.Foreground, hv)
		if st.Disabled {
			col = col.Mix(t.Card, 0.5)
		}
		glyph := comboboxCenterRect(rc, comboboxChipXSz)
		p.Icon(icons.IconX, &glyph, col, 2)
	} else {
		c.clearHit, c.hasClearHit = xc.RECT{}, false
		icon := xc.RECT{
			Left:   p.Rc.Right - 26,
			Top:    p.Rc.Top + (p.H()-16)/2,
			Right:  p.Rc.Right - 10,
			Bottom: p.Rc.Top + (p.H()-16)/2 + 16,
		}
		p.Icon(icons.IconChevronsUpDown, &icon, t.MutedForeground, 2)
	}
	return 0
}

// comboboxChipFont 是 chip 的字体(text-xs = 12; 上游 font-medium, xcgui 只有常规/粗体)。
func comboboxChipFont(t *xui.Theme) int {
	return xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 12})
}

// comboboxChipWidth 返回一个 chip 的宽度: 左右 px-1.5 + 文字 + (移除钮: gap 4 + 16 命中块 + 2 右距)。
func comboboxChipWidth(label string, f int, remove bool) int32 {
	w := comboboxChipPad*2 + xui.TextWidth(label, f)
	if remove {
		w += comboboxChipGap + comboboxChipXHit + 2
	}
	return w
}

// layoutChips 按触发区矩形排 chip; 放不下的折叠成 "+N"(value 为空, 无移除钮)。
// 布局是纯计算: 绘制时算一次存 chipHits 供绘制/命中复用, 命中块失效时按当前尺寸补算。
func (c *Combobox) layoutChips(t *xui.Theme, rc xc.RECT, picked []SelectItem) []comboboxChipHit {
	f := comboboxChipFont(t)
	h := comboboxChipH
	y := rc.Top + (rc.Bottom-rc.Top-h)/2
	right := rc.Right - comboboxChipTrk
	x := rc.Left + comboboxChipsPad
	hits := make([]comboboxChipHit, 0, len(picked))
	for i, it := range picked {
		label := selectItemLabel(it)
		cw := comboboxChipWidth(label, f, true)
		// 后面还有条目时要给 "+N" 标记留位置(按最坏位数预留, 免得差一位让标记压到箭头)。
		need := cw
		if rest := len(picked) - i; rest > 1 {
			need += comboboxChipGap + comboboxChipWidth("+"+strconv.Itoa(rest), f, false)
		}
		if x+need > right {
			break
		}
		hit := comboboxChipHit{
			value: it.Value, label: label, hasX: true,
			chip: xc.RECT{Left: x, Top: y, Right: x + cw, Bottom: y + h},
		}
		hit.x = xc.RECT{
			Left:   hit.chip.Right - 2 - comboboxChipXHit,
			Top:    hit.chip.Top + (h-comboboxChipXHit)/2,
			Right:  hit.chip.Right - 2,
			Bottom: hit.chip.Top + (h-comboboxChipXHit)/2 + comboboxChipXHit,
		}
		hits = append(hits, hit)
		x = hit.chip.Right + comboboxChipGap
	}
	if hidden := len(picked) - len(hits); hidden > 0 {
		label := "+" + strconv.Itoa(hidden)
		cw := comboboxChipWidth(label, f, false)
		if x+cw > right {
			cw = right - x
		}
		if cw > 0 {
			hits = append(hits, comboboxChipHit{
				label: label,
				chip:  xc.RECT{Left: x, Top: y, Right: x + cw, Bottom: y + h},
			})
		}
	}
	return hits
}

// paintChips 画触发区 chips: bg-muted + rounded-sm + text-xs; × 默认半透明, 悬停到全色。
func (c *Combobox) paintChips(p *xui.Painter, t *xui.Theme, hEle int, bg, fore xui.Color, disabled bool) {
	f := comboboxChipFont(t)
	if f != 0 {
		p.Draw.SetFont(f) // p.TextF 之外还要给原生绘制留回默认字体(见函数尾)
	}
	chipBg := t.Muted
	xBase := fore.ScaleAlpha(0.5) // × 的 opacity-50
	if disabled {
		chipBg = chipBg.Mix(bg, 0.5)
		fore = fore.Mix(bg, 0.5)
		xBase = xBase.Mix(bg, 0.5)
	}
	for i, hit := range c.chipHits {
		hv := xui.HoverItem(hEle, int32(i), !disabled && c.hoverChip == int32(i))
		p.FillRound(&hit.chip, chipBg, t.RadiusSm())
		tr := xc.RECT{
			Left:   hit.chip.Left + comboboxChipPad,
			Top:    hit.chip.Top,
			Right:  hit.chip.Right - comboboxChipPad,
			Bottom: hit.chip.Bottom,
		}
		if hit.hasX {
			tr.Right = hit.x.Left - 4
		}
		if hit.label != "" && tr.Right > tr.Left {
			dy := textVDy(hit.label)
			trf := xc.RECTF{
				Left:   float32(tr.Left),
				Top:    float32(tr.Top) - dy,
				Right:  float32(tr.Right),
				Bottom: float32(tr.Bottom) - dy,
			}
			p.TextF(hit.label, &trf, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
		}
		if hit.hasX {
			glyph := comboboxCenterRect(hit.x, comboboxChipXSz)
			p.Icon(icons.IconX, &glyph, xui.Fade(xBase, fore, hv), 2)
		}
	}
	// 画笔字体是接着这个 hDraw 用的, 不还原会把原生绘制也带成 12px(同 TagInput)。
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
}

// comboboxClearRect 是清除钮的命中块(24×24, 中心与箭头一致)。
func comboboxClearRect(rc xc.RECT) xc.RECT {
	h := comboboxClearSz
	y := rc.Top + (rc.Bottom-rc.Top-h)/2
	return xc.RECT{Left: rc.Right - 6 - h, Top: y, Right: rc.Right - 6, Bottom: y + h}
}

// comboboxButtonRadius 是 icon-xs 尺寸按钮的圆角: min(--radius-md, 10)。
func comboboxButtonRadius(t *xui.Theme) int32 {
	if md := t.RadiusMd(); md < 10 {
		return md
	}
	return 10
}

// comboboxCenterRect 返回 rc 中心处边长 sz 的正方形。
func comboboxCenterRect(rc xc.RECT, sz int32) xc.RECT {
	cx, cy := (rc.Left+rc.Right)/2, (rc.Top+rc.Bottom)/2
	return xc.RECT{Left: cx - sz/2, Top: cy - sz/2, Right: cx - sz/2 + sz, Bottom: cy - sz/2 + sz}
}

// ===== 触发区 chips / 清除钮命中 =====

// invalidateTriggerHits 使上次绘制缓存的命中块失效(选值变化后由下一次绘制/命中补算)。
func (c *Combobox) invalidateTriggerHits() {
	c.chipHits = nil
	c.clearHit, c.hasClearHit = xc.RECT{}, false
	c.hoverChip = -1
	c.hitsW, c.hitsH = 0, 0
}

// ensureTriggerHits 在命中块缺失/尺寸变化时按当前客户区重算。
func (c *Combobox) ensureTriggerHits() {
	if c == nil || c.Button == nil {
		return
	}
	var rc xc.RECT
	xc.XEle_GetClientRect(c.Handle, &rc)
	w, h := rc.Right-rc.Left, rc.Bottom-rc.Top
	if w <= 0 || h <= 0 {
		return
	}
	if (len(c.chipHits) > 0 || c.hasClearHit) && w == c.hitsW && h == c.hitsH {
		return
	}
	c.hitsW, c.hitsH = w, h
	t := xui.Current()
	if c.multiple && c.chips && len(c.picked) > 0 {
		c.chipHits = c.layoutChips(t, rc, c.pickedItems())
	} else {
		c.chipHits = nil
	}
	if c.clear && len(c.picked) > 0 {
		c.clearHit, c.hasClearHit = comboboxClearRect(rc), true
	} else {
		c.clearHit, c.hasClearHit = xc.RECT{}, false
	}
}

// chipRemoveAt 返回命中的 chip 移除钮对应的值(fallback 会先补算命中块)。
func (c *Combobox) chipRemoveAt(x, y int32) (string, bool) {
	if c == nil || !c.multiple || !c.chips {
		return "", false
	}
	c.ensureTriggerHits()
	for _, hit := range c.chipHits {
		if hit.hasX && inRect(hit.x, x, y) {
			return hit.value, true
		}
	}
	return "", false
}

// clearHitAt 返回是否命中清除钮。
func (c *Combobox) clearHitAt(x, y int32) bool {
	if c == nil || !c.clear || len(c.picked) == 0 {
		return false
	}
	c.ensureTriggerHits()
	return c.hasClearHit && inRect(c.clearHit, x, y)
}

// hoverTrigger 更新指针下的 chip × / 清除钮, 变化时重绘(HoverItem 会平滑过渡)。
func (c *Combobox) hoverTrigger(x, y int32) {
	if c == nil || !c.IsEnable() {
		return
	}
	idx := int32(-1)
	clear := c.clearHitAt(x, y)
	if !clear && c.multiple && c.chips && len(c.picked) > 0 {
		c.ensureTriggerHits()
		for i, hit := range c.chipHits {
			if hit.hasX && inRect(hit.x, x, y) {
				idx = int32(i)
				break
			}
		}
	}
	if idx != c.hoverChip || clear != c.hoverClear {
		c.hoverChip, c.hoverClear = idx, clear
		c.Redraw(false)
	}
}

func inRect(rc xc.RECT, x, y int32) bool {
	return x >= rc.Left && x < rc.Right && y >= rc.Top && y < rc.Bottom
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
	x := p.Rc.Left + 8 // px-2(与 Command 条目一致)
	if c.multiple {
		// 多选: 左侧勾(选中才画), 占位 16 + gap 8(上游 Check 的 mr-2), 未选中也占位。
		if c.picked[r.item.Value] {
			rc := xc.RECT{Left: x, Top: (p.Rc.Top + p.Rc.Bottom - 16) / 2,
				Right: x + 16, Bottom: (p.Rc.Top + p.Rc.Bottom + 16) / 2}
			p.Icon(icons.IconCheck, &rc, fore, 2)
		}
		x += 24
	}
	label := selectItemLabel(r.item)
	if r.item.Icon != "" {
		rc := xc.RECT{Left: x, Top: (p.Rc.Top + p.Rc.Bottom - 16) / 2,
			Right: x + 16, Bottom: (p.Rc.Top + p.Rc.Bottom + 16) / 2}
		p.Icon(r.item.Icon, &rc, fore, 2)
		x += 16
		if label != "" {
			x += 8 // gap-2
		}
	}
	if label != "" {
		dy := textVDy(label)
		rc := xc.RECTF{
			Left:   float32(x),
			Top:    float32(p.Rc.Top) - dy,
			Right:  float32(p.Rc.Right - 24),
			Bottom: float32(p.Rc.Bottom) - dy,
		}
		p.TextF(label, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
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
