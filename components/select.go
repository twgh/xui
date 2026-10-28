package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Select 独立下拉选择框, 对应 shadcn select.tsx(radix Select, item-aligned 定位):
//   - 触发区: 与 ComboBox 同一套视觉(border-input / focus-visible 环 / sm 尺寸 /
//     aria-invalid / 暗色 bg-input/30 悬停加深), 未选择时显示占位符;
//   - 弹层: 自建阴影窗(popoverPanel) + ScrollArea, 条目是自绘按钮 —— 所以条目可以
//     带图标、禁用态、分组标签与分隔线(原生 ComboBox 弹层做不到); item-aligned 定位:
//     弹层盖住触发区、选中项正对触发区, 宽度 = max(触发区宽, min-w-36);
//   - 键盘: ↑/↓/Home/End 移动高亮(焦点即高亮, bg-accent), Enter/空格 选中,
//     ESC 关闭并把焦点还给触发区; 超出可视高度自动滚;
//   - 与 ComboBox 的取舍: 弹层行为(定位/动画/关闭时机)全部自持, 不依赖原生弹窗。
type Select struct {
	*widget.Button
	groups        []SelectGroup
	flat          []*selectRow
	value         string
	placeholder   string
	size          SelectSize
	invalid       bool
	disabled      bool
	panel         *popoverPanel
	scroll        *ScrollArea
	rows          *widget.LayoutEle
	built         bool
	active        int32 // 键盘高亮条目在 flat 里的索引(-1 无)
	pendingScroll int32 // Open 落位后要应用的滚动位(item-aligned)
	changes       []func(item SelectItem)
}

type selectRowKind int32

const (
	selectRowItem selectRowKind = iota
	selectRowLabel
	selectRowSep
)

// selectRow 弹层里的一行: 条目(自绘按钮)/分组标签/分隔线。
type selectRow struct {
	kind selectRowKind
	item SelectItem
	btn  *widget.Button
	ele  *widget.Element
	top  int32 // 打开时重算: 相对滚动内容顶部的 y
	seq  int32 // 可聚焦条目的序号(键盘导航用), 非条目 = -1
}

// SelectOption 下拉框配置项。
type SelectOption func(*Select)

// WithSelectPlaceholder 设置未选择时的占位文本。
func WithSelectPlaceholder(s string) SelectOption {
	return func(s2 *Select) { s2.placeholder = s }
}

// WithSelectSize 设置触发区尺寸(sm = h-7 = 28)。
func WithSelectSize(s SelectSize) SelectOption {
	return func(s2 *Select) { s2.size = s }
}

// WithSelectItems 设置平铺条目(无分组)。
func WithSelectItems(items []SelectItem) SelectOption {
	return func(s2 *Select) { s2.groups = []SelectGroup{{Items: items}} }
}

// WithSelectGroups 设置分组条目(组间自动加分隔线)。
func WithSelectGroups(groups []SelectGroup) SelectOption {
	return func(s2 *Select) { s2.groups = groups }
}

// WithSelectValue 设置初始选中值(按 SelectItem.Value 匹配)。
func WithSelectValue(v string) SelectOption {
	return func(s2 *Select) { s2.value = v }
}

// WithSelectInvalid 设置校验失败态(aria-invalid)。
func WithSelectInvalid() SelectOption {
	return func(s2 *Select) { s2.invalid = true }
}

// WithSelectDisabled 禁用触发区。
func WithSelectDisabled() SelectOption {
	return func(s2 *Select) { s2.disabled = true }
}

// NewSelect 创建下拉选择框。cx/cy 传 0: cy 用尺寸档高度, cx 按内容量宽
// (放卡片里通常配 xui.ItemFill 让它铺满)。
func NewSelect(x, y, cx, cy int32, hParent int, opts ...SelectOption) *Select {
	s := &Select{size: SelectSizeDefault, active: -1}
	for _, o := range opts {
		o(s)
	}
	if cy <= 0 {
		if s.size == SelectSizeSm {
			cy = 28
		} else {
			cy = 32
		}
	}
	btn := widget.NewButton(x, y, cx, cy, "", hParent)
	if btn == nil {
		return nil
	}
	s.Button = btn
	if cx <= 0 {
		btn.SetWidth(s.naturalWidth())
	}
	btn.EnableBkTransparent(true)
	btn.SetTextColor(0) // 文字自绘
	btn.AddEvent_Paint(s.onPaint)
	btn.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if !s.IsEnable() {
			return 0
		}
		xui.NotePointerPress()
		// 弹层开着时点触发区: 弹层先因失焦关闭(更新了 lastClose), 这里不再重开 ——
		// 否则会出现"点两次才关掉"(同 datepicker)。
		if s.panel != nil && (s.panel.IsOpen() || s.panel.RecentlyClosed()) {
			return 0
		}
		s.Open()
		return 0
	})
	btn.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if s.panel != nil {
			s.panel.Destroy()
		}
		s.panel, s.scroll, s.rows = nil, nil, nil
		return 0
	})
	if s.disabled {
		s.SetDisabled(true)
	}
	xui.AttachWindow(btn.Handle)
	return s
}

// ===== 数据 =====

// SetItems 替换为平铺条目(重建弹层内容), 保持原 value(匹配不上就清空)。
func (s *Select) SetItems(items []SelectItem) *Select {
	return s.SetGroups([]SelectGroup{{Items: items}})
}

// SetGroups 替换分组条目。
func (s *Select) SetGroups(groups []SelectGroup) *Select {
	s.groups = groups
	s.built = false
	if s.panel != nil {
		s.panel.Destroy()
		s.panel, s.scroll, s.rows = nil, nil, nil
	}
	s.syncValue()
	s.Redraw(false)
	return s
}

// Value 返回当前选中的 SelectItem.Value, 未选中为空串。
func (s *Select) Value() string { return s.value }

// Selected 返回当前选中条目; 没有匹配(ok=false)时返回零值。
// 直接查**源数据**(groups)而不是弹层行(flat): flat 是懒构建的, 没弹过层之前是空的,
// 触发区在构建后立刻 SetValue 就会查空表 —— 表现为"设了值却还显示占位符"。
func (s *Select) Selected() (SelectItem, bool) {
	for _, g := range s.groups {
		for _, it := range g.Items {
			if it.Value == s.value {
				return it, true
			}
		}
	}
	return SelectItem{}, false
}

// setValue 按值选中条目(不触发 OnChange); 没有匹配项时取消选中。
func (s *Select) SetValue(v string) *Select {
	s.value = v
	s.syncValue()
	s.Redraw(false)
	return s
}

// OnChange 注册选中变化回调(用户选中时触发; SetValue 不触发)。
func (s *Select) OnChange(fn func(item SelectItem)) *Select {
	s.changes = append(s.changes, fn)
	return s
}

// SetInvalid 设置/清除校验失败态。
func (s *Select) SetInvalid(v bool) *Select {
	if s.invalid == v {
		return s
	}
	s.invalid = v
	s.Redraw(false)
	return s
}

// Invalid 返回当前是否处于校验失败态。
func (s *Select) Invalid() bool { return s.invalid }

// SetDisabled 禁用/启用触发区(开着时一并关闭弹层)。
func (s *Select) SetDisabled(v bool) *Select {
	s.disabled = v
	s.Enable(!v)
	if v && s.panel != nil {
		s.panel.Hide()
	}
	s.Redraw(false)
	return s
}

// Disabled 返回触发区是否禁用。
func (s *Select) Disabled() bool { return s.disabled }

// FitWidth 按当前选中项重新量宽(仅 cx 传 0 创建的触发区)。
func (s *Select) FitWidth() *Select {
	s.SetWidth(s.naturalWidth())
	s.Redraw(false)
	return s
}

func (s *Select) syncValue() {
	// 查源数据而不是 flat(flat 懒构建, 见 Selected 的说明)
	if s.value == "" {
		return
	}
	for _, g := range s.groups {
		for _, it := range g.Items {
			if it.Value == s.value {
				return
			}
		}
	}
	s.value = ""
}

// ===== 弹层 =====

// Open 打开弹层(内容按需构建/重建)。
func (s *Select) Open() *Select {
	if s == nil || s.disabled || (s.panel != nil && s.panel.IsOpen()) {
		return s
	}
	if !s.built {
		s.build()
	}
	if s.panel == nil {
		return s
	}
	x, y := s.itemAlignedPos()
	s.panel.ShowAt(s.Handle, x, y)
	// item-aligned: 内容超高时把选中项滚到触发区中心(radix 的滚动对位)
	if s.pendingScroll > 0 {
		s.scroll.ScrollPosYV(s.pendingScroll)
	}
	// 打开时把焦点/高亮放到选中条目(radix 内容获得焦点, 选中项 bg-accent)
	s.setActive(s.rowOfValue(s.value), s.pendingScroll <= 0)
	return s
}

// Close 关闭弹层。
func (s *Select) Close() *Select {
	if s.panel != nil {
		s.panel.Hide()
	}
	return s
}

// IsOpen 返回弹层是否打开。
func (s *Select) IsOpen() bool { return s != nil && s.panel != nil && s.panel.IsOpen() }

// build 构建弹层窗口 + 行(数据变化后 built=false, 下次 Open 重建)。
func (s *Select) build() {
	s.built = true
	s.flat = s.flatten()
	if s.panel != nil {
		s.panel.Destroy()
		s.panel, s.scroll, s.rows = nil, nil, nil
	}
	s.syncValue()

	cw, ch := s.popupSize()
	panel := newPopoverPanel(cw, ch, s.Handle)
	if panel == nil {
		return
	}
	s.panel = panel

	// 视口内缩 p-1(4): 条目高亮距卡片边缘 4px(同 Content 的 p-1 组内边距)
	s.scroll = NewScrollArea(selectPad, selectPad, cw-selectPad*2, ch-selectPad*2, panel.Content())
	if s.scroll == nil {
		return
	}
	xui.ItemSize(s.scroll, cw-selectPad*2, ch-selectPad*2)
	s.rows = s.scroll.ContentStack(0)

	// 键盘: ↑/↓/Home/End 移动高亮, Enter 选中; ESC 关闭由 popup.go 兜底
	panel.OnKeyDown = func(wParam uintptr) bool {
		switch wParam {
		case xcc.VK_Up:
			s.moveActive(-1)
			return true
		case xcc.VK_Down:
			s.moveActive(1)
			return true
		case xcc.VK_Home:
			s.setActive(s.firstEnabled(0, 1), true)
			return true
		case xcc.VK_End:
			s.setActive(s.firstEnabled(int32(len(s.flat))-1, -1), true)
			return true
		case xcc.VK_Enter:
			if s.active >= 0 && s.active < int32(len(s.flat)) {
				s.choose(s.flat[s.active])
			}
			return true
		}
		return false
	}
	panel.OnEsc = func() { s.SetFocus() }

	for i, r := range s.flat {
		r := r // go.mod 是 1.18: 循环变量旧语义, 闭包必须拷贝(HANDOFF 硬规则)
		switch r.kind {
		case selectRowItem:
			btn := widget.NewButton(0, 0, 0, 0, "", s.rows.Handle)
			if btn == nil {
				continue
			}
			r.btn = btn
			btn.EnableBkTransparent(true)
			btn.SetTextColor(0)
			btn.Enable(!r.item.Disabled)
			xui.ItemFill(btn)
			btn.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, selectItemH)
			btn.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
				s.paintItem(r, hEle, hDraw, pbHandled)
				return 0
			})
			idx := i
			btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
				s.choose(s.flat[idx])
				return 0
			})
		case selectRowLabel:
			ele := widget.NewElement(0, 0, 0, 0, s.rows.Handle)
			if ele == nil {
				continue
			}
			r.ele = ele
			ele.EnableBkTransparent(true)
			xui.ItemFill(ele)
			ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, selectLabelH)
			label := r.item.Text
			ele.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
				p := xui.BeginPaint(hEle, hDraw)
				if p == nil {
					return 0
				}
				*pbHandled = true
				t := xui.Current()
				p.FillRound(&p.Rc, t.Popover, 0) // 擦底(透明元素重绘不擦底)
				f := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 12})
				if f != 0 {
					p.Draw.SetFont(f)
				}
				// px-2: 组标签的左内距与条目内容(pl-1.5 + 图标列)对齐, 不能贴视口边框
				rc := p.Rc
				rc.Left += 6
				p.Text(label, &rc, t.MutedForeground, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
				return 0
			})
		case selectRowSep:
			ele := widget.NewElement(0, 0, 0, 0, s.rows.Handle)
			if ele == nil {
				continue
			}
			r.ele = ele
			ele.EnableBkTransparent(true)
			xui.ItemFill(ele)
			ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 1)
			// my-1: 上下各 8px 外边距(shadcn 分隔线 -mx-1 my-1; 负水平外边距没有
			// 对应物, 分隔线只到视口内缘, 差 4px 不可感知)
			xc.XWidget_LayoutItem_SetMargin(ele.Handle, 0, 8, 0, 8)
			ele.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
				p := xui.BeginPaint(hEle, hDraw)
				if p == nil {
					return 0
				}
				*pbHandled = true
				t := xui.Current()
				p.FillRound(&p.Rc, t.Border, 0)
				return 0
			})
		}
	}
	s.layoutRows()
}

// flatten 把分组数据摊平成行序列(组间加分隔线)。
func (s *Select) flatten() []*selectRow {
	var rows []*selectRow
	first := true
	for _, g := range s.groups {
		if !first {
			rows = append(rows, &selectRow{kind: selectRowSep})
		}
		first = false
		if g.Label != "" {
			rows = append(rows, &selectRow{kind: selectRowLabel, item: SelectItem{Text: g.Label}})
		}
		for _, it := range g.Items {
			rows = append(rows, &selectRow{kind: selectRowItem, item: it, seq: -1})
		}
	}
	seq := int32(0)
	for _, r := range rows {
		if r.kind == selectRowItem {
			r.seq = seq
			seq++
		}
	}
	return rows
}

// rowHeight 各类行的占位高度。
func selectRowHeight(k selectRowKind) int32 {
	switch k {
	case selectRowItem:
		return selectItemH
	case selectRowLabel:
		return selectLabelH
	default:
		return selectSepH
	}
}

// layoutRows 计算各行 top(滚动内容坐标系)与弹层内容总高。
func (s *Select) layoutRows() {
	y := int32(0)
	for _, r := range s.flat {
		r.top = y
		y += selectRowHeight(r.kind)
	}
}

// contentHeight 全部行的内容总高(不含 p-1)。
func (s *Select) contentHeight() int32 {
	h := int32(0)
	for _, r := range s.flat {
		h += selectRowHeight(r.kind)
	}
	return h
}

// popupSize 弹层卡片尺寸: 宽 = max(触发区宽, min-w-36), 高 = 内容高(有上限)。
func (s *Select) popupSize() (int32, int32) {
	cw := s.GetWidth()
	if cw < selectMinW {
		cw = selectMinW
	}
	ch := s.contentHeight() + selectPad*2
	if ch > selectMaxH {
		ch = selectMaxH
	}
	return cw, ch
}

// rowOfValue 返回匹配 value 的条目行索引; 没有则返回第一个可用条目(radix 无选中时
// item-aligned 把首项对到触发区)。
func (s *Select) rowOfValue(v string) int32 {
	first := int32(-1)
	for i, r := range s.flat {
		if r.kind != selectRowItem || r.item.Disabled {
			continue
		}
		if first < 0 {
			first = int32(i)
		}
		if r.item.Value == v {
			return int32(i)
		}
	}
	return first
}

// firstEnabled 从 from 起按 dir(+1/-1)找第一个可聚焦条目行。
func (s *Select) firstEnabled(from int32, dir int32) int32 {
	for i := from; i >= 0 && i < int32(len(s.flat)); i += dir {
		if r := s.flat[i]; r.kind == selectRowItem && !r.item.Disabled {
			return i
		}
	}
	return -1
}

// setActive 设置键盘高亮条目: 焦点落到它的按钮上(bg-accent 高亮随焦点走)。
func (s *Select) setActive(idx int32, reveal bool) {
	if idx < 0 || idx >= int32(len(s.flat)) {
		return
	}
	s.active = idx
	r := s.flat[idx]
	if r.btn != nil {
		r.btn.SetFocus()
	}
	if reveal {
		s.revealRow(idx)
	}
}

// moveActive 按 dir(+1/-1)移动高亮, 跳过禁用条目。
func (s *Select) moveActive(dir int32) {
	if len(s.flat) == 0 {
		return
	}
	i := s.active
	if i < 0 {
		i = 0
	}
	for {
		i += dir
		if i < 0 || i >= int32(len(s.flat)) {
			return
		}
		if r := s.flat[i]; r.kind == selectRowItem && !r.item.Disabled {
			s.setActive(i, true)
			return
		}
	}
}

// revealRow 把行滚进可视区(视口 = 弹层高 - 上下 p-1)。
func (s *Select) revealRow(idx int32) {
	if s.panel == nil || s.scroll == nil || idx < 0 || idx >= int32(len(s.flat)) {
		return
	}
	r := s.flat[idx]
	vh := s.panel.ch - selectPad*2
	pos := xc.XSView_GetViewPosV(s.scroll.Handle) // 当前滚动位(像素)
	if r.top < pos {
		s.scroll.ScrollPosYV(r.top)
	} else if r.top+selectItemH > pos+vh {
		s.scroll.ScrollPosYV(r.top + selectItemH - vh)
	}
}

// itemAlignedPos 计算 item-aligned 弹层位置(radix SelectContent position=默认):
// 选中项正对触发区(两种盒子高度不同, 按**中心**对齐, 同 alignComboPopup 的推导),
// 水平居中于触发区; 返回逻辑像素, 原点 = 宿主窗口矩形原点(同 popoverPanel.Show)。
// 内容超出弹层高时顺带算出"选中项对到触发区中心"的滚动位(pendingScroll),
// Open 在落位后应用。
func (s *Select) itemAlignedPos() (int32, int32) {
	var trig xc.RECT
	xc.XEle_GetWndClientRect(s.Handle, &trig)
	hWnd := xc.XWidget_GetHWINDOW(s.Handle)
	if hWnd <= 0 {
		return 0, 0
	}
	var wr xc.RECT
	xc.XWnd_GetRect(hWnd, &wr)

	cw, ch := s.popupSize()
	trigH := trig.Bottom - trig.Top
	trigW := trig.Right - trig.Left
	selRow := s.rowOfValue(s.value)
	selTop := int32(0)
	if selRow >= 0 {
		selTop = s.flat[selRow].top
	}
	// 弹层顶 = 触发区顶 - 选中项偏移: 选中项在弹层内容里的 y = pad + top - scroll,
	// 要它落在触发区中心 (trigH-条目高)/2 处。
	s.pendingScroll = selTop + selectPad - (trigH-selectItemH)/2
	content := selectPad*2 + s.contentHeight()
	if s.pendingScroll > content-ch {
		s.pendingScroll = content - ch
	}
	if s.pendingScroll < 0 {
		s.pendingScroll = 0
	}
	x := wr.Left + trig.Left + (trigW-cw)/2
	y := wr.Top + trig.Top - (selTop + selectPad - s.pendingScroll) + (trigH-selectItemH)/2
	return x, y
}

// choose 选中一行: 回写 value、关闭弹层、通知监听(即使重复选同一项, 同 radix)。
func (s *Select) choose(r *selectRow) {
	if r == nil || r.kind != selectRowItem || r.item.Disabled {
		return
	}
	s.value = r.item.Value
	s.Close()
	s.Redraw(false)
	for _, fn := range s.changes {
		fn(r.item)
	}
}

// ===== 触发区绘制 =====

// naturalWidth 量出 w-fit 触发区宽: pl-2.5 + 内容 + gap-1.5 + 箭头 + pr-2。
func (s *Select) naturalWidth() int32 {
	t := xui.Current()
	f := xui.FontHandle(t.FontDefault)
	w := int32(10 + 6 + 18 + 8) // pl-2.5 + gap-1.5 + chevron 16+2 + pr-2
	if it, ok := s.Selected(); ok {
		if it.Icon != "" {
			w += 16
			if it.Text != "" {
				w += 6
			}
		}
		w += xui.TextWidth(it.Text, f)
	} else if s.placeholder != "" {
		w += xui.TextWidth(s.placeholder, f)
	} else {
		w += 24 // 空触发区的最小可视宽
	}
	return w
}

// onPaint 触发区: 与 ComboBox.onPaint 同一套规则(border-input / focus-visible 环 /
// aria-invalid / 暗色 bg-input/30 悬停加深 / disabled)。
func (s *Select) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)

	bg, border := t.Card, t.Input
	if st.Disabled {
		bg = t.Input.WithAlpha(128) // disabled:bg-input/50(同 ComboBox)
		border = xui.None
	}
	radius := t.RadiusLg()
	if s.size == SelectSizeSm {
		if md := t.RadiusMd(); md < 10 {
			radius = md
		} else {
			radius = 10
		}
	}
	corners := xui.UniformCorners(radius)
	const inset = int32(3)
	switch {
	case s.invalid && !st.Disabled:
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
		// dark:hover:bg-input/50: 暗色触发区有常驻 bg-input/30 底, 悬停加深到 /50
		if t.IsDark() && !st.Disabled {
			a := 0.3 * xui.Hover(hEle, st.Hover && !st.Pressed)
			p.FillRoundCorners(&p.Rc, t.Input.ScaleAlpha(0.3+a), corners)
		}
		p.StrokeRoundCorners(&p.Rc, border, corners, 1)
	}

	// 文字: 选中 = 条目文字+前置图标; 未选 = 占位符(muted)
	fore := t.Foreground
	f := xui.FontHandle(t.FontDefault)
	if f != 0 {
		p.Draw.SetFont(f)
	}
	it, ok := s.Selected()
	if st.Disabled {
		fore = fore.Mix(t.Card, 0.5) // disabled:opacity-50 的文字预混
	}
	if !ok && st.Disabled {
		fore = t.MutedForeground.Mix(t.Card, 0.5)
	} else if !ok {
		fore = t.MutedForeground
	}
	x := p.Rc.Left + 10 // pl-2.5
	if ok && it.Icon != "" {
		rc := xc.RECT{Left: x, Top: (p.Rc.Top + p.Rc.Bottom - 16) / 2,
			Right: x + 16, Bottom: (p.Rc.Top + p.Rc.Bottom + 16) / 2}
		p.Icon(it.Icon, &rc, fore, 2)
		x += 16
		if it.Text != "" {
			x += 6
		}
	}
	text := s.placeholder
	if ok {
		text = it.Text
	}
	if text != "" {
		rc := xc.RECT{
			Left:   x,
			Top:    p.Rc.Top - 1, // Vcenter 字形偏下 1px(同 Button)
			Right:  p.Rc.Right - 30,
			Bottom: p.Rc.Bottom - 1,
		}
		p.Text(text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	}

	// 右侧 chevron-down: text-muted-foreground(同 ComboBox)
	icon := xc.RECT{
		Left:   p.Rc.Right - 26,
		Top:    p.Rc.Top + (p.H()-16)/2,
		Right:  p.Rc.Right - 10,
		Bottom: p.Rc.Top + (p.H()-16)/2 + 16,
	}
	p.Icon(icons.IconChevronDown, &icon, t.MutedForeground, 2)
	return 0
}

// ===== 弹层条目绘制 =====

// paintItem 自绘条目行: 高亮(悬停/焦点)rounded-md bg-accent, 选中右侧 check,
// 禁用 opacity-50。文字/图标颜色实时取令牌(主题切换由弹窗整窗重绘带动)。
func (s *Select) paintItem(r *selectRow, hEle int, hDraw int, pbHandled *bool) {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)

	p.FillRound(&p.Rc, t.Popover, 0) // 擦底(透明元素重绘不擦底)
	hv := xui.Hover(hEle, (st.Hover || st.Focused) && !r.item.Disabled)
	if hv > 0 {
		p.FillRound(&p.Rc, t.Accent.ScaleAlpha(hv), t.RadiusSm())
	}

	fore := t.PopoverForeground
	if r.item.Disabled {
		fore = fore.Mix(t.Popover, 0.5) // data-disabled:opacity-50
	}
	f := xui.FontHandle(t.FontDefault)
	if f != 0 {
		p.Draw.SetFont(f)
	}
	x := p.Rc.Left + 6 // pl-1.5
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
		rc := xc.RECT{
			Left:   x,
			Top:    p.Rc.Top - 1,
			Right:  p.Rc.Right - 24, // pr-8: 给 check 留位
			Bottom: p.Rc.Bottom - 1,
		}
		p.Text(r.item.Text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
	}
	if r.item.Value == s.value && !r.item.Disabled {
		// 选中项右侧 check(absolute right-2 size-4)
		rc := xc.RECT{
			Left:   p.Rc.Right - 24,
			Top:    (p.Rc.Top + p.Rc.Bottom - 16) / 2,
			Right:  p.Rc.Right - 8,
			Bottom: (p.Rc.Top + p.Rc.Bottom + 16) / 2,
		}
		p.Icon(icons.IconCheck, &rc, t.PopoverForeground, 2)
	}
}
