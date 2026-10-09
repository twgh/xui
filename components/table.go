package ui

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 表格几何常量, 逐条对齐 shadcn table.tsx(radix-nova):
//
//	TableHead  = h-10 px-2            → 表头高 40, 水平内边距 8
//	TableCell = p-2 + text-sm(行高 20) → 数据行高 36, 单元格内边距 8
const (
	tableHeaderH  int32 = 40 // h-10
	tableRowH     int32 = 36 // p-2(8×2) + text-sm 行高 20
	tableCellPadX int32 = 8  // px-2 / p-2 的水平内边距
	// 原生 List 的内容区在左侧 / 顶部各留 2 逻辑像素内缩(探针实测): 列宽要减掉它,
	// 表头文本缩进也要补偿它, 否则最后一条底边越过右缘、表头文字与单元格文字错位。
	tableInset int32 = 2
	// 原生纵向滚动条占用的宽度(SetScrollBarSize(8) + SetSliderPadding(2)): 内容溢出
	// 时给自适应列让出来, 否则最后一列会被滚动条盖住。
	tableSBW int32 = 12
)

// TableColumn 表格列定义。
type TableColumn struct {
	Title string
	// Width 列宽(逻辑像素)。<= 0 表示占满剩余宽度 —— 只对最后一列有意义。
	Width int32
}

// Table 是 shadcn 风格的多列表格: 表头 + 数据行 + 行悬浮 / 选中。
//
// 载体是**原生 List**(表头、纵向滚动、行选中、项模板复用全部原生), 外观由自绘接管;
// 原生 List 的默认皮肤(灰表头、横竖网格线、系统滚动条、白底)全部被压掉。
//
// 与 shadcn table.tsx 对齐:
//   - 表头: 高 40, 文字 foreground, 只有一条 1px 底边(border);
//   - 数据行: 高 36, 底边 1px border, **最后一行不画**(tbody 的 [&_tr:last-child]:border-0);
//   - 行悬浮 hover:bg-muted/50, 选中 data-[state=selected]:bg-muted;
//   - 表体无外框、无圆角、无纵向网格线(shadcn 的 table 就是 w-full text-sm, 外框由调用方
//     决定: 放进 Card, 或自己套一层圆角描边容器)。
//
// 为什么不直接用原生 Table(Shape 系): 它是无滚动、无虚拟行的"单元格画布"(合并单元格、
// 逐格画线), 与 shadcn 的语义(可滚动的行列表)不符, 详见 DESIGN.md 第八节 40。
type Table struct {
	*widget.List

	cols    []TableColumn
	autoCol int32 // 自适应列的索引, -1 表示各列都是固定宽
	rowH    int32
	onBg    bool
	onSel   func(row int32)
	sbWide  bool // 当前列宽是否已为滚动条让位
}

// TableOption 组件配置项。
type TableOption func(*Table)

// WithTableRowHeight 覆盖数据行高(默认 36, 即 shadcn 的 p-2 + text-sm)。
func WithTableRowHeight(h int32) TableOption {
	return func(t *Table) {
		if h > 0 {
			t.rowH = h
		}
	}
}

// WithTableOnBackground 让表体底色跟随 Background 令牌(默认跟随 Card)。
// 组件默认按"放在 Card 上"设计(表体透明等价于 Card 色, 同 ComboBox); 直接把表格放在
// 窗口背景上时用本选项, 否则暗色下表格底色会比窗口底色亮一档。
func WithTableOnBackground() TableOption {
	return func(t *Table) { t.onBg = true }
}

// NewTable 创建表格。cy 传 0 时使用"表头 + 5 行"的默认高度。
func NewTable(x, y, cx, cy int32, hParent int, cols []TableColumn, opts ...TableOption) *Table {
	if len(cols) == 0 {
		cols = []TableColumn{{Title: ""}}
	}
	t := &Table{
		List:    widget.NewList(x, y, cx, cy, hParent),
		cols:    cols,
		autoCol: -1,
		rowH:    tableRowH,
	}
	if t.List == nil {
		return nil
	}
	for _, o := range opts {
		o(t)
	}
	if cy <= 0 {
		t.SetHeight(tableHeaderH + tableInset*2 + t.rowH*5)
	}
	if cols[len(cols)-1].Width <= 0 {
		t.autoCol = int32(len(cols) - 1)
	}

	// 数据适配器: 表头与列表项各一个(List 不挂适配器就是空壳)
	t.CreateAdapterHeader()
	t.CreateAdapter(int32(len(cols)))

	// 几何: 表头高 / 行高 / 行间距 0(原生默认行间距 1px, shadcn 的行是紧贴的)
	t.SetHeaderHeight(tableHeaderH)
	t.SetRowHeightDefault(t.rowH, t.rowH)
	t.SetRowSpace(0)

	// 原生皮肤全压掉: 行底与分割线交给自绘, 表体真透明(露出宿主底色)
	t.SetDrawRowBkFlags(xcc.List_DrawItemBk_Flag_Nothing)
	t.SetSplitLineColor(0)
	t.EnableBkTransparent(true)
	t.EnableDrawBorder(false)
	t.EnableDrawFocus(false)

	// 滚动条: 细条 + 内容不溢出时隐藏; 横向不需要(列宽由本组件算好)
	t.SetScrollBarSize(8)
	t.EnableAutoShowScrollBar(true)
	t.ShowSBarH(false)

	font := xui.FontHandle(xui.Current().FontDefault)
	if font != 0 {
		t.SetFont(font)
	}
	for i, c := range cols {
		t.AddColumnText(t.colWidth(int32(i), cx, false), fmt.Sprintf("name%d", i+1), c.Title)
	}

	// 表头元素也要透明(shadcn 的表头没有底色, 只有一条底边), 并灭掉原生焦点框
	if h := t.GetHeaderHELE(); h > 0 {
		if hd := widget.NewElementByHandle(h); hd != nil {
			hd.EnableBkTransparent(true)
			hd.EnableDrawBorder(false)
			hd.EnableDrawFocus(false)
		}
	}

	t.AddEvent_List_Temp_Create_End(t.onTempCreate)
	t.AddEvent_List_Header_Temp_Create_End(t.onHeaderTempCreate)
	t.AddEvent_List_DrawItem(t.onDrawItem)
	t.AddEvent_List_Header_DrawItem(t.onHeaderDrawItem)
	t.AddEvent_List_Select(func(hEle int, iItem int32, pbHandled *bool) int {
		if t.onSel != nil {
			t.onSel(iItem)
		}
		return 0
	})

	t.applyTheme(xui.Current())
	xui.OnThemeChange(func(th *xui.Theme) { t.applyTheme(th) })
	xui.AttachWindow(t.Handle)
	return t
}

// ===== 数据 API =====

// AddRow 添加一行, 返回行索引。cells 依次对应各列。
func (t *Table) AddRow(cells ...string) int32 {
	first := ""
	if len(cells) > 0 {
		first = cells[0]
	}
	idx := t.AddRowText(first)
	for i := 1; i < len(cells) && i < len(t.cols); i++ {
		t.SetItemText(idx, int32(i), cells[i])
	}
	t.relayout(false)
	return idx
}

// SetCell 更新某个单元格的文本(row/col 越界时忽略)。
func (t *Table) SetCell(row, col int32, text string) *Table {
	if row < 0 || row >= t.Rows() || col < 0 || col >= int32(len(t.cols)) {
		return t
	}
	t.SetItemText(row, col, text)
	t.RefreshRow(row)
	t.Redraw(false)
	return t
}

// SetRow 覆盖整行文本。
func (t *Table) SetRow(row int32, cells ...string) *Table {
	for i, s := range cells {
		t.SetCell(row, int32(i), s)
	}
	return t
}

// CellText 读取单元格文本。
func (t *Table) CellText(row, col int32) string { return t.GetItemText(row, col) }

// Rows 返回行数。
func (t *Table) Rows() int32 { return t.GetCount_AD() }

// Columns 返回列数。
func (t *Table) Columns() int32 { return int32(len(t.cols)) }

// Clear 清空所有行。
func (t *Table) Clear() *Table {
	t.DeleteRowAll()
	t.relayout(false)
	t.Redraw(false)
	return t
}

// Selected 返回选中行索引, 未选中返回 -1。
func (t *Table) Selected() int32 { return t.GetSelectRow() }

// Select 选中指定行(传 -1 取消选中)并滚动到可见。
func (t *Table) Select(row int32) *Table {
	if row < 0 {
		t.CancelSelectAll()
	} else {
		t.SetSelectRow(row)
		t.VisibleRow(row)
	}
	t.Redraw(false)
	return t
}

// OnSelect 注册行选中回调。
func (t *Table) OnSelect(fn func(row int32)) *Table {
	t.onSel = fn
	return t
}

// Refresh 重新加载全部行(批量改完数据后调用)。
func (t *Table) Refresh() *Table {
	t.relayout(false)
	t.RefreshData()
	t.Redraw(false)
	return t
}

// ===== 布局 / 主题 =====

// colWidth 计算第 i 列的宽度。scrolling 为 true 时给滚动条让出宽度。
func (t *Table) colWidth(i int32, cx int32, scrolling bool) int32 {
	if w := t.cols[i].Width; w > 0 {
		return w
	}
	var used int32
	for j, c := range t.cols {
		if int32(j) != i && c.Width > 0 {
			used += c.Width
		}
	}
	rest := cx - tableInset*2 - used
	if scrolling {
		rest -= tableSBW
	}
	if rest < 20 {
		rest = 20
	}
	return rest
}

// needScroll 报告当前行数是否装不下(需要纵向滚动条)。
func (t *Table) needScroll() bool {
	vis := (t.GetHeight() - tableHeaderH - tableInset*2) / t.rowH
	return t.Rows() > vis
}

// relayout 在"是否需要滚动条"翻转时重新分配自适应列宽 —— 滚动条出现后不让出宽度的话
// 最后一列会被盖住, 没滚动条时白留空隙同样难看。force 为 true 时(元素宽度变了)强制重算。
func (t *Table) relayout(force bool) {
	if t.autoCol < 0 {
		return
	}
	need := t.needScroll()
	if !force && t.sbWide == need {
		return
	}
	t.sbWide = need
	t.SetColumnWidth(t.autoCol, t.colWidth(t.autoCol, t.GetWidth(), need))
	t.Redraw(false)
}

// FitColumns 按元素**当前**宽度重算自适应列宽。
// 表格被布局元素拉伸(或所在窗口尺寸变化)后调用, 列宽才会跟上新的宽度。
func (t *Table) FitColumns() *Table {
	t.relayout(true)
	return t
}

// surface 返回表体底色令牌。
func (t *Table) surface(th *xui.Theme) xui.Color {
	if t.onBg {
		return th.Background
	}
	return th.Card
}

// applyTheme 只刷新"必须同步给原生对象"的颜色与字体: 形状文本(单元格 / 表头)的字色字体、
// 滚动条配色。行底、表头底、边框都在绘制回调里实时取令牌, 换肤零成本, 无需在此处理。
func (t *Table) applyTheme(th *xui.Theme) {
	if !xc.XC_IsHXCGUI(t.Handle, xcc.XC_ELE) {
		return // 元素已销毁(窗口关闭), 句柄失效
	}
	font := xui.FontHandle(th.FontDefault)
	color := uint32(th.Foreground)
	t.SetTextColor(color)
	if font != 0 {
		t.SetFont(font)
	}
	for i := int32(0); i < t.Columns(); i++ {
		if h := t.GetHeaderTemplateObject(i, 1); h != 0 {
			xc.XShapeText_SetTextColor(h, color)
			if font != 0 {
				xc.XShapeText_SetFont(h, font)
			}
			xc.XWidget_LayoutItem_SetMargin(h, tableCellPadX-tableInset, 0, 0, 0)
		}
	}
	start, end := int32(0), int32(-1)
	t.GetVisibleRowRange(&start, &end)
	if start < 0 {
		start = 0 // 没有可视行时原生可能返回 -1
	}
	for r := start; r <= end && r < t.Rows(); r++ {
		for c := int32(0); c < t.Columns(); c++ {
			h := t.GetTemplateObject(r, c, 1)
			if h == 0 {
				continue
			}
			xc.XShapeText_SetTextColor(h, color)
			if font != 0 {
				xc.XShapeText_SetFont(h, font)
			}
			xc.XWidget_LayoutItem_SetMargin(h, tableCellPadX, 0, 0, 0)
		}
	}
	t.styleScrollBars(th)
	t.relayout(false)
	t.Redraw(false)
}

// onTempCreate 单元格模板实例化 / 复用时同步文字样式。
// 复用(nFlag=2)也要设: 实例可能来自另一种状态, 或主题刚换过。
func (t *Table) onTempCreate(hEle int, pItem *xc.List_Item_, nFlag int32, pbHandled *bool) int {
	if pItem == nil {
		return 0
	}
	th := xui.Current()
	font := xui.FontHandle(th.FontDefault)
	for i := range t.cols {
		h := t.GetTemplateObject(pItem.Index, int32(i), 1)
		if h == 0 {
			continue
		}
		xc.XShapeText_SetTextColor(h, uint32(th.Foreground))
		if font != 0 {
			xc.XShapeText_SetFont(h, font)
		}
		xc.XWidget_LayoutItem_SetMargin(h, tableCellPadX, 0, 0, 0) // p-2
	}
	return 0
}

// onHeaderTempCreate 表头模板实例化时同步文字样式。
// 表头单元格的横坐标比数据单元格靠左 tableInset(原生 List 的怪癖), 用文本缩进补回来,
// 表头文字才能与列内文字左对齐。
func (t *Table) onHeaderTempCreate(hEle int, pItem *xc.List_Header_Item_, pbHandled *bool) int {
	th := xui.Current()
	font := xui.FontHandle(th.FontDefault)
	for i := range t.cols {
		h := t.GetHeaderTemplateObject(int32(i), 1)
		if h == 0 {
			continue
		}
		xc.XShapeText_SetTextColor(h, uint32(th.Foreground))
		if font != 0 {
			xc.XShapeText_SetFont(h, font)
		}
		xc.XWidget_LayoutItem_SetMargin(h, tableCellPadX-tableInset, 0, 0, 0)
	}
	return 0
}

// onDrawItem 自绘单元格: 行底色(悬浮 / 选中) + 行底边。
// 绘制回调按"单元格"触发(同一行各列的 NState 相同), 单元格矩形首尾相接, 因此逐格填充
// 拼出的就是整行底色, 逐格画线拼出的就是整行分割线。
func (t *Table) onDrawItem(hEle, hDraw int, pItem *xc.List_Item_, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil || pItem == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()
	rc := pItem.RcItem

	// 行底: 先铺一层不透明的宿主底色(透明元素重绘不擦底, 见 DESIGN 第八节 28),
	// 悬停色叠在它上面并按过渡值淡入淡出(行上的 transition-colors, 见 table.tsx);
	// 选中态立即到位(点击结果不该有延迟)。
	p.FillRound(&rc, t.surface(th), 0)
	if pItem.NState == xcc.List_Item_State_Select {
		p.FillRound(&rc, th.Muted, 0) // data-[state=selected]:bg-muted
	} else if hv := xui.HoverItem(hEle, pItem.Index, pItem.NState == xcc.List_Item_State_Stay); hv > 0 {
		p.FillRound(&rc, th.Muted.ScaleAlpha(0.5*hv), 0) // hover:bg-muted/50
	}
	// 行底边: 最后一行不画(tbody 的 [&_tr:last-child]:border-0)
	if pItem.Index < t.Rows()-1 {
		p.Line(rc.Left, rc.Bottom-1, rc.Right, rc.Bottom-1, th.Border, 1)
	}
	return 0
}

// onHeaderDrawItem 自绘表头单元格: 底色 + 底边(横向铺满).
func (t *Table) onHeaderDrawItem(hEle, hDraw int, pItem *xc.List_Header_Item_, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil || pItem == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()
	rc := pItem.RcItem
	p.FillRound(&rc, t.surface(th), 0)
	p.Line(rc.Left, rc.Bottom-1, rc.Right, rc.Bottom-1, th.Border, 1)
	return 0
}

// styleScrollBars 把纵向滚动条收成细条 + 主题色。
// 滚动条颜色只能通过背景管理器下发给原生对象, 因此换肤要重刷一遍(Clear 后重加,
// 不会累积超过炫彩的 20 个背景对象上限)。
func (t *Table) styleScrollBars(th *xui.Theme) {
	sb := t.GetScrollBarVObj()
	if sb == nil || sb.Handle <= 0 {
		return
	}
	sb.ShowButton(false) // 去掉上下箭头
	sb.SetSliderPadding(2)
	sb.EnableBkTransparent(true)
	sb.ClearBkInfo()
	sb.AddBkFill(xcc.Element_State_Flag_Leave, uint32(t.surface(th)))
	sb.AddBkFill(xcc.Element_State_Flag_Stay, uint32(t.surface(th)))
	if sl := sb.GetButtonSliderObj(); sl != nil && sl.Handle > 0 {
		sl.ClearBkInfo()
		sl.AddBkFill(xcc.Element_State_Flag_Leave, uint32(th.Border))
		sl.AddBkFill(xcc.Element_State_Flag_Stay, uint32(th.MutedForeground))
		sl.AddBkFill(xcc.Element_State_Flag_Down, uint32(th.MutedForeground))
	}
}
