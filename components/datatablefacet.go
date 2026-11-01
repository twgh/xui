package ui

import (
	"strconv"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// ===== 分面筛选的触发胶囊 =====
//
// shadcn 的 DataTableFacetedFilter 触发区是"虚线描边按钮 + 右侧并排的选中项 badge",
// 3 个以上收敛成 "3 selected"。Button 只会画一行居中文字, 所以这里自绘一个轻量元素:
// 圆角壳(无选中 = 描边, 有选中 = secondary 填充) + 前置 ⊕ 图标 + 列名 + 竖分隔 + 胶囊。

const (
	facetTriggerH int32 = 28 // 与工具栏里 ButtonSizeSm 同高
	facetChipH    int32 = 19
	facetChipPad  int32 = 6
	facetPadX     int32 = 6
	facetIconGap  int32 = 5
	facetGap      int32 = 4 // 胶囊之间 gap-1
)

// facetTrigger 分面筛选的触发元素(替代原来的 Button)。
type facetTrigger struct {
	*widget.Element

	title  string
	icon   string
	values []string // 选中的选项文字(已按选项顺序)
	open   bool     // 弹层是否展开(画成按下/展开色)
}

func newFacetTrigger(x, y, w, h int32, parent int, title, icon string) *facetTrigger {
	ele := widget.NewElement(x, y, w, h, parent)
	if ele == nil {
		return nil
	}
	f := &facetTrigger{Element: ele, title: title, icon: icon}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.AddEvent_Paint(f.onPaint)
	// 点击交给 Popover 自己注册的 LButtonUp 处理(见 NewPopover) —— 不要在这里再挂一个
	// 回调: 每个会话都新建一个 Popover 会给同一个元素反复挂 toggle, 点一次切多次(闪烁越开越多)。
	return f
}

// SetValues 更新选中项(空 = 未筛选)。宽度会跟着重算, 调用方负责重新排版。
func (f *facetTrigger) SetValues(vals []string) *facetTrigger {
	if f == nil {
		return f
	}
	f.values = vals
	f.Redraw(false)
	return f
}

// SetOpen 标记弹层展开状态(只影响配色)。
func (f *facetTrigger) SetOpen(open bool) *facetTrigger {
	if f == nil {
		return f
	}
	f.open = open
	f.Redraw(false)
	return f
}

// naturalWidth 按内容算宽度(含胶囊), 供工具栏设 ItemWidth。
func (f *facetTrigger) naturalWidth(t *xui.Theme) int32 {
	if f == nil {
		return 0
	}
	w := facetPadX*2 + 14 + facetIconGap + xui.TextWidth(f.title, f.fontHandle(t, false))
	if chips := f.chips(); len(chips) > 0 {
		w += 9 // 竖分隔: 左 4 + 1 + 右 4
		for _, c := range chips {
			w += xui.TextWidth(c, f.fontHandle(t, true)) + facetChipPad*2
		}
		w += int32(len(chips)-1) * facetGap
	}
	return w
}

// chips 返回要画的胶囊文字: 最多两个值, 再多就收敛成 "N 个已选"(对应 shadcn 的 3 selected)。
func (f *facetTrigger) chips() []string {
	if len(f.values) == 0 {
		return nil
	}
	if len(f.values) > 2 {
		return []string{strconv.Itoa(len(f.values)) + " 个已选"}
	}
	return f.values
}

func (f *facetTrigger) fontHandle(t *xui.Theme, small bool) int {
	if small {
		if h := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 12}); h != 0 {
			return h
		}
	}
	return xui.FontHandle(t.FontDefault)
}

func (f *facetTrigger) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)

	rc := p.Rc
	radius := t.RadiusMd()
	hv := xui.Hover(hEle, st.Hover || f.open || st.Pressed)
	active := len(f.values) > 0

	if active {
		bg := t.Secondary
		if st.Hover || f.open {
			bg = xui.Fade(t.Secondary, t.Accent, hv)
		}
		p.FillRound(&rc, bg, radius)
	} else {
		if st.Hover || f.open {
			p.FillRound(&rc, xui.Fade(t.Background, t.Accent, hv), radius)
		}
		p.StrokeRound(&rc, t.Border, radius, 1)
	}

	fg := t.Foreground
	muted := t.MutedForeground
	x := rc.Left + facetPadX
	mid := (rc.Top + rc.Bottom) / 2

	if f.icon != "" {
		ir := xc.RECT{Left: x, Top: mid - 7, Right: x + 14, Bottom: mid + 7}
		p.Icon(f.icon, &ir, muted, 2)
	}
	x += 14 + facetIconGap

	if fh := f.fontHandle(t, false); fh != 0 {
		p.Draw.SetFont(fh)
	}
	tw := xui.TextWidth(f.title, f.fontHandle(t, false))
	tr := xc.RECT{Left: x, Top: rc.Top, Right: x + tw, Bottom: rc.Bottom}
	p.Text(f.title, &tr, fg, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
	x += tw

	chips := f.chips()
	if len(chips) == 0 {
		return 0
	}

	x += 4
	sep := xc.RECT{Left: x, Top: mid - 8, Right: x + 1, Bottom: mid + 8}
	p.FillRound(&sep, t.Border, 0)
	x += 5

	chipFont := f.fontHandle(t, true)
	if chipFont != 0 {
		p.Draw.SetFont(chipFont)
	}
	for _, c := range chips {
		cw := xui.TextWidth(c, chipFont) + facetChipPad*2
		cr := xc.RECT{Left: x, Top: mid - facetChipH/2, Right: x + cw, Bottom: mid + facetChipH/2}
		p.FillRound(&cr, t.Background, 4)
		p.Text(c, &cr, t.SecondaryForeground,
			xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		x += cw + facetGap
	}
	return 0
}

// ===== 分面筛选弹层(Popover + Command: 搜索 / 多选不关闭 / 计数 / 清除) =====

// facetCounts 统计某列各选项在**当前其它筛选条件**下的行数(TanStack 的
// getFacetedUniqueValues 口径: 统计时排除该列自身的分面筛选, 否则随便勾一个,
// 其余选项的数量就全变 0 了)。
func (d *DataTable) facetCounts(ci int) map[string]int {
	others := make(map[int]map[string]bool, len(d.facet))
	for k, v := range d.facet {
		if k != ci {
			others[k] = v
		}
	}
	view := dataTableView(d.rows, d.filter, d.filterCols, others, -1, d.sortDir, nil)
	out := make(map[string]int, 8)
	for _, r := range view {
		if r < 0 || r >= len(d.rows) || ci < 0 || ci >= len(d.rows[r]) {
			continue
		}
		out[d.rows[r][ci]]++
	}
	return out
}

// facetLabels 返回某列已选选项的**显示文字**, 按选项声明顺序(触发胶囊与弹层都按这个顺序)。
func (d *DataTable) facetLabels(ci int) []string {
	if ci < 0 || ci >= len(d.cols) {
		return nil
	}
	set := d.facet[ci]
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for _, opt := range d.cols[ci].FilterOptions {
		if set[opt.value()] {
			out = append(out, opt.Label)
		}
	}
	return out
}

// facetPanel 一列分面筛选的常驻会话: 弹层/内容只建一次, 之后每次打开只刷新勾选与计数。
// **不能每次点击都 NewPopover** —— 每个 Popover 都会往触发元素上挂一个 toggle 回调与
// 一个 Destroy 订阅, 越开越多, 点一次切 N 次(表现为"越用越闪")。
type facetPanel struct {
	d     *DataTable
	ci    int
	btn   *facetTrigger
	pop   *Popover
	cmd   *Command
	items map[string]*CommandItem
}

// newFacetPanel 建一列的面板(在 buildToolbar 里一次性创建, 不阻塞)。
func (d *DataTable) newFacetPanel(ci int, btn *facetTrigger) *facetPanel {
	p := &facetPanel{d: d, ci: ci, btn: btn, items: map[string]*CommandItem{}}
	col := d.cols[ci]

	const panelW = facetPanelW
	p.pop = NewPopover(btn.Handle,
		WithPopoverWidth(panelW),
		WithPopoverPadding(0),
		WithPopoverAlign(PopoverAlignStart), // 左对齐, 与触发区起始边对齐(shadcn 的 align="start")
		WithPopoverOnOpenChange(func(open bool) {
			btn.SetOpen(open)
			if open {
				p.refresh()
				if p.cmd != nil {
					p.cmd.Focus()
				}
			}
		}),
		WithPopoverContent(func(content int) {
			// Command 的高度按内容精确给(输入行 + 选项分组 + 底部清除行):
			// Command 是固定高, 多给的高度会全落在列表底部; 少给则裁行并出滚动条。
			p.cmd = NewCommand(0, 0, panelW, p.cmdHeight(), content,
				WithCommandPlaceholder(col.Title),
				WithCommandOnEscape(func() { p.pop.Close() }),
			)
			xui.ItemSize(p.cmd, panelW, p.cmdHeight())
			g := p.cmd.AddGroup("")
			for _, opt := range col.FilterOptions {
				opt := opt
				v := opt.value()
				opts := []CommandItemOption{
					WithCommandItemCheckable(),
					WithCommandItemChecked(d.facetHas(ci, v)),
					WithCommandItemCount(0),
					WithCommandItemKeywords(v),
					WithCommandItemOnSelect(func(v string) { p.selectValue(v) }),
				}
				if opt.Icon != "" {
					opts = append(opts, WithCommandItemIcon(opt.Icon))
				}
				p.items[v] = g.AddItem(v, opt.Label, opts...)
			}
			// 清除筛选放进 Command 列表内部(分割线 + 居中条目): shadcn 的 Clear filters
			// 就是 CommandList 里的一个 CommandItem, 两边都靠分组内边距定位, 不会像
			// Popover 内容流那样多出 8px 子项间距, 也不需要给窗口高度补余量。
			// 点它只清选择、**不关闭面板**(shadcn 的 Popover 也只在点外部/ESC 时关闭)。
			p.cmd.addFooter("__clear__", "清除筛选", func(string) {
				d.clearFacet(ci)
				p.refresh()
			})
			p.layout() // 构建完就把高度/显隐定下来(首次 Open 的 measure 直接用这个高度)
		}))
	if p.pop == nil {
		return nil
	}
	return p
}

// facetPanelW 弹层宽度(对齐 shadcn 的 w-[200px] + 内边距)。
const facetPanelW int32 = 232

// cmdHeight Command 的精确高度(逻辑像素)。Command 是固定高, 高度必须等于
// 输入行 + 选项分组(上下各 4) + 底部清除行(有勾选时), 多一个像素都会变成列表底部的
// 空白, 少一个像素就裁行并出滚动条。
func (p *facetPanel) cmdHeight() int32 {
	n := int32(len(p.d.cols[p.ci].FilterOptions))
	h := commandInputH + n*commandItemH + commandGroupPad*2
	if len(p.d.facet[p.ci]) > 0 {
		// 分割线 1 + 底部组上下各 4 + 清除条目 32, 与 shadcn 的
		// CommandSeparator + CommandGroup(p-1) + CommandItem(h-8) 同几何。
		h += 1 + commandGroupPad*2 + commandItemH
	}
	return h
}

// layout 按"当前是否勾选"决定底部清除行在不在, 并把 Command 与弹层高度改成精确值。
// 未勾选时 shadcn 不显示清除行, 这一段要连布局项一起收掉(只 Show(false) 仍占位, 见 172)。
// 高度用 SetContentHeight 直接改窗口, 不走离屏测量(那样会闪)。
func (p *facetPanel) layout() {
	if p.pop == nil || p.cmd == nil {
		return
	}
	has := len(p.d.facet[p.ci]) > 0
	p.cmd.setFooterVisible(has)
	h := p.cmdHeight()
	p.cmd.SetHeight(h)
	xui.ItemHeight(p.cmd, h)
	p.cmd.AdjustLayout()
	// 内容区只有 Command 一个子项, 没有内容流间距, 弹层高度就是 Command 高度。
	p.pop.SetContentHeight(h)
}

// selectValue 处理一次勾选/取消: 面板不关闭, 表格与计数实时刷新(shadcn 的多选语义)。
func (p *facetPanel) selectValue(v string) {
	p.d.setFacet(p.ci, v, !p.d.facetHas(p.ci, v))
	p.refresh()
}

// refresh 把勾选态与各选项行数同步到面板与触发胶囊。
func (p *facetPanel) refresh() {
	counts := p.d.facetCounts(p.ci)
	for v, it := range p.items {
		it.SetChecked(p.d.facetHas(p.ci, v))
		it.SetCount(counts[v])
	}
	p.layout()
	p.d.updateToolbar()
}
