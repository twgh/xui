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
	facetTriggerH int32 = 28 // h-7 附近, 与工具栏里 ButtonSizeSm 同高
	facetChipH    int32 = 19
	facetChipPad  int32 = 6
	facetPadX     int32 = 6 // px-2 略宽一点, 胶囊贴着边不好看
	facetIconGap  int32 = 5
	facetGap      int32 = 4 // 胶囊之间 gap-1
)

// facetTrigger 分面筛选的触发元素(替代原来的 Button)。
type facetTrigger struct {
	*widget.Element

	title   string
	icon    string
	values  []string // 选中的选项文字(已按选项顺序)
	open    bool     // 弹层是否展开(画成按下/展开色)
	onClick func()
}

func newFacetTrigger(x, y, w, h int32, parent int, title, icon string, onClick func()) *facetTrigger {
	ele := widget.NewElement(x, y, w, h, parent)
	if ele == nil {
		return nil
	}
	f := &facetTrigger{Element: ele, title: title, icon: icon, onClick: onClick}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.AddEvent_Paint(f.onPaint)
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if f.onClick != nil {
			f.onClick()
		}
		return 0
	})
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

// naturalWidth 按内容算宽度(含胶囊): 与 Button.naturalWidth 同口径, 供工具栏设 ItemWidth。
func (f *facetTrigger) naturalWidth(t *xui.Theme) int32 {
	if f == nil {
		return 0
	}
	w := facetPadX*2 + 14 + facetIconGap + xui.TextWidth(f.title, f.fontHandle(t, false))
	if len(f.values) > 0 {
		w += 9 // 竖分隔左右各 4 + 1
		for _, c := range f.chips() {
			w += xui.TextWidth(c, f.fontHandle(t, true)) + facetChipPad*2
		}
		w += int32(len(f.chips())-1) * facetGap
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
		// 有选中 = secondary(与 Button.ButtonSecondary 同色), 悬停再压一档
		bg := t.Secondary
		if st.Hover || f.open {
			bg = xui.Fade(t.Secondary, t.Accent, hv)
		}
		p.FillRound(&rc, bg, radius)
	} else {
		// 无选中 = 描边空态(近似 border-dashed 的观感)
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

	// 列名
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

	// 竖分隔(对应 shadcn 的 Separator, 高 h-4)
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

// ===== 分面筛选弹层(Command 承载搜索 + 多选 + 计数) =====

const facetClearValue = "\x00clear"

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

// facetPanel 一次弹层的会话状态: 关掉就丢(计数与勾选都从 d 现算)。
type facetPanel struct {
	d     *DataTable
	ci    int
	pop   *Popover
	cmd   *Command
	items map[string]*CommandItem
}

// showFacetPanel 弹出某列的分面筛选面板。与旧版(Menu.Check)的差别: 面板里带搜索、
// 点一项不关闭、右侧显示该选项当前行数、底部有"清除筛选"(对应 shadcn 的
// DataTableFacetedFilter = Popover + Command)。
func (d *DataTable) showFacetPanel(ci int) {
	if ci < 0 || ci >= len(d.cols) {
		return
	}
	col := d.cols[ci]
	btn := d.facetBtns[ci]
	if btn == nil {
		return
	}
	// 再点一次 = 收起
	if d.facetPanel != nil {
		d.facetPanel.close()
		return
	}

	const panelW int32 = 232 // 对齐 shadcn 的 w-[200px] + 内边距
	p := &facetPanel{d: d, ci: ci, items: map[string]*CommandItem{}}

	var pop *Popover
	pop = NewPopover(btn.Handle,
		WithPopoverWidth(panelW),
		WithPopoverPadding(0),
		WithPopoverOnOpenChange(func(open bool) {
			btn.SetOpen(open)
			if !open && d.facetPanel == p {
				d.facetPanel = nil
			}
		}),
		WithPopoverContent(func(content int) {
			rowH := commandInputH + int32(len(col.FilterOptions))*commandItemH + commandGroupPad*2
			p.cmd = NewCommand(0, 0, panelW, rowH, content,
				WithCommandPlaceholder(col.Title),
				WithCommandOnEscape(func() { pop.Close() }),
				WithCommandOnSelect(func(v string) { p.selectValue(v) }),
			)
			xui.ItemWidth(p.cmd, panelW)
			g := p.cmd.AddGroup("")
			for _, opt := range col.FilterOptions {
				opt := opt
				v := opt.value()
				opts := []CommandItemOption{
					WithCommandItemChecked(d.facetHas(ci, v)),
					WithCommandItemCount(0),
					WithCommandItemKeywords(v),
				}
				if opt.Icon != "" {
					opts = append(opts, WithCommandItemIcon(opt.Icon))
				}
				p.items[v] = g.AddItem(v, opt.Label, opts...)
			}
			clear := NewButton(0, 0, 0, 0, "清除筛选", content,
				WithVariant(ButtonGhost), WithSize(ButtonSizeSm))
			xui.ItemFill(clear)
			clear.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
				d.clearFacet(ci)
				p.refresh()
				pop.Close()
				return 0
			})
		}))
	if pop == nil {
		return
	}
	p.pop = pop
	d.facetPanel = p
	btn.SetOpen(true)
	pop.Open()
	p.refresh()
	if p.cmd != nil {
		p.cmd.Focus()
	}
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

// selectValue 处理一次勾选/取消: 面板不关闭, 表格与计数实时刷新(shadcn 的多选语义)。
func (p *facetPanel) selectValue(v string) {
	if v == facetClearValue {
		return
	}
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
	if p.d.facetPanel == p && p.ci < len(p.d.facetBtns) {
		p.d.updateToolbar()
	}
}

func (p *facetPanel) close() {
	if p.pop != nil {
		p.pop.Close()
	}
	if p.d.facetPanel == p {
		p.d.facetPanel = nil
	}
}
