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
	// 原生纵向滚动条出现时给自适应列让出的宽度: 滚动条元素 scrollBarW(10) + 内容区
	// 左右固有内缩(2×2)的保守值, 否则最后一列会被滚动条盖住(见 DESIGN 第八节 66)。
	tableSBW int32 = 14
)

// DataAlign 单元格文字的水平对齐(只影响原生文本; 自绘单元格自己决定排版)。
type DataAlign int

const (
	DataAlignLeft   DataAlign = iota // 默认, px-2 左对齐
	DataAlignCenter                  // 居中
	DataAlignRight                   // 右对齐(金额 / 数字列)
)

// TableColumn 表格列定义。
type TableColumn struct {
	Title string
	// Width 列宽(逻辑像素)。<= 0 表示占满剩余宽度: 只有**第一个** <= 0 的列吃剩余宽度
	// (中间的标题列也能自适应), 其余 <= 0 的列按 120 固定宽排。
	Width int32
	// Align 原生文本的对齐方式(默认左对齐)。
	Align DataAlign
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
	sbWide  bool  // 当前列宽是否已为滚动条让位
	lastW   int32 // 上次算自适应列宽时元素的宽度, 用于判断"宽度真的变了"(见 relayout)
	// onHeaderExtra 在表头单元格底色/底边之后追加绘制(DataTable 用它画排序图标)。
	// 必须在 onHeaderDrawItem 里调用, **不能**让调用方另注册 AddEvent_List_Header_DrawItem ——
	// AddEvent 后注册的回调先执行, 组件自己的回调后铺不透明底色, 会把追加内容盖掉。
	onHeaderExtra func(p *xui.Painter, rc xc.RECT, col int32)
	// onCellExtra 在单元格底色之后、行底边之前追加绘制(DataTable 用它画勾选列 /
	// 富单元格 / 行操作钮)。参数 col 是**当前可见列**的下标(见 8.167)。
	// 同样必须走组件内部回调: 另注册 AddEvent_List_DrawItem 会被本组件的行底色盖掉。
	// 回调里要自带文字/图标 —— 富单元格的原生文本由调用方清空(见 8.168)。
	onCellExtra func(p *xui.Painter, rc xc.RECT, row, col int32, hover bool)
	// hdrRects 缓存各表头单元格的矩形(元素客户区坐标), 由 onHeaderDrawItem 填入 ——
	// 表头点击回调不带坐标, 要按点击的列定位弹层(菜单不能跟着鼠标松开处跑)就靠它。
	hdrRects []xc.RECT
	// onHeaderActive 报告该表头列是否可交互(可排序 / 可隐藏)。只有这些列才画
	// 悬停 / 按下底色, 也才允许拖拽换位; 普通文字表头保持无底色。nil = 全不可交互。
	onHeaderActive func(col int32) bool
	// onColumnsMoved 在用户拖拽表头列后回调。from 是 Table 列下标, to 是插入边界
	// (0..列数); 由调用方决定怎么落地(DataTable 会重排显示顺序)。nil = 不启用拖拽。
	onColumnsMoved func(from, to int32)

	headerEle   *widget.Element // 表头元素: 鼠标事件挂在它身上(原生表头悬停不会自己重绘)
	headerHover int32           // 当前悬停的可交互表头列, -1 = 无
	headerDown  int32           // 当前按下的可交互表头列, -1 = 无

	dragFrom  int32 // 拖拽起始表头列, -1 = 未拖拽
	dragTo    int32 // 拖拽插入边界(Table 列下标 0..列数)
	dragDownX int32
	dragMoved bool
	// dragSwallowClick 在拖拽结束的那次松开上置位: 若原生仍补发一次表头点击,
	// 调用方据此吞掉菜单(下一次按下会清掉, 不会误伤后续单击)。
	dragSwallowClick bool
}

// HeaderCellRect 返回表头第 col 列的单元格矩形(元素客户区坐标)。只在表头画过之后有效;
// 没有任何表头绘制过时 ok=false。
func (t *Table) HeaderCellRect(col int32) (rc xc.RECT, ok bool) {
	if col < 0 || int(col) >= len(t.hdrRects) {
		return rc, false
	}
	r := t.hdrRects[col]
	if r.Right <= r.Left || r.Bottom <= r.Top {
		return rc, false
	}
	return r, true
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
		List:        widget.NewList(x, y, cx, cy, hParent),
		cols:        append([]TableColumn(nil), cols...), // 归一化 Width 时不要改到调用方的切片
		autoCol:     -1,
		rowH:        tableRowH,
		lastW:       -1,
		headerHover: -1,
		headerDown:  -1,
		dragFrom:    -1,
		dragTo:      -1,
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
	t.pickAutoCol()

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

	// 滚动条: 细条(shadcn 规格, 见 scrollbar.go) + 内容不溢出时隐藏; 横向不需要(列宽由本组件算好)
	t.SetScrollBarSize(scrollBarW)
	t.EnableAutoShowScrollBar(true)
	t.ShowSBarH(false)
	showScrollBarOnHover(t.Handle, t.GetScrollBarVObj())

	font := xui.FontHandle(xui.Current().FontDefault)
	if font != 0 {
		t.SetFont(font)
	}
	for i, c := range cols {
		t.AddColumnText(t.colWidth(int32(i), cx, false), fmt.Sprintf("name%d", i+1), c.Title)
	}

	// 表头元素也要透明(shadcn 的表头没有底色, 只有一条底边), 并灭掉原生焦点框。
	// 原生表头悬停 / 按下只改项状态、**不会重绘**(见 DESIGN 8.176), 所以悬停底色与
	// 列拖拽都要自己订阅表头元素的鼠标事件, 在事件里改状态并主动重绘表头。
	if h := t.GetHeaderHELE(); h > 0 {
		if hd := widget.NewElementByHandle(h); hd != nil {
			hd.EnableBkTransparent(true)
			hd.EnableDrawBorder(false)
			hd.EnableDrawFocus(false)
			t.headerEle = hd
			hd.AddEvent_MouseMove(t.onHeaderMouseMove)
			hd.AddEvent_MouseLeave(t.onHeaderMouseLeave)
			hd.AddEvent_LButtonDown(t.onHeaderMouseDown)
			hd.AddEvent_LButtonUp(t.onHeaderMouseUp)
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
	xui.OnThemeChangeFor(t.Handle, func(th *xui.Theme) { t.applyTheme(th) })

	// 自适应列宽依赖"元素当前宽度", 而宽度是布局算出来的: 创建时(窗口 Show 之前)拿到的是 0,
	// 算出的列宽会退化成最小值(表头与单元格文本被裁成 "...")。订阅元素自身的尺寸事件,
	// 一旦拿到真实宽度就重算 —— 组件自愈, 调用方不必记着手动调 FitColumns。
	t.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		t.relayout(true)
		return 0
	})

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

// tableDefaultColW 是"多余的自适应列"的兜底宽度: 只有**第一个** Width<=0 的列
// 吃掉剩余宽度(同 shadcn 里只有一个 `w-full` 的列), 其余 Width<=0 的列按固定宽排,
// 否则每列都会算出同一份剩余宽度、把表格撑出可视区。
const tableDefaultColW int32 = 120

// pickAutoCol 选出占满剩余宽度的列(第一个 Width<=0 的), 其余 Width<=0 的列归一成固定宽。
func (t *Table) pickAutoCol() {
	t.autoCol = -1
	for i, c := range t.cols {
		if c.Width > 0 {
			continue
		}
		if t.autoCol < 0 {
			t.autoCol = int32(i)
			continue
		}
		t.cols[i].Width = tableDefaultColW
	}
}

// SetColumns 重建列结构(列显隐 / 动态列)。行数据会被清空 —— 调用方随后要重新
// AddRow(DataTable 的 render 就是这么做的)。
//
// 不能只 DeleteColumnAll + AddColumnText: 原生 List 的**行数据适配器**仍按旧列数
// 工作, 新加的第 4 列写进去会被丢掉(探针实测 rows 不变、GetItemText(0,3) 恒为空)。
// 必须 DeleteColumnAll 之后重新 CreateAdapter(新列数)(见 DESIGN 8.169)。
func (t *Table) SetColumns(cols []TableColumn) *Table {
	if len(cols) == 0 {
		cols = []TableColumn{{Title: ""}}
	}
	t.cols = append([]TableColumn(nil), cols...)
	t.pickAutoCol()
	t.hdrRects = nil // 列结构变了, 旧的表头矩形作废(下次绘制重新填)
	t.headerHover, t.headerDown = -1, -1
	t.resetHeaderDrag()
	t.dragSwallowClick = false
	t.DeleteRowAll()
	t.DeleteColumnAll()
	w := t.GetWidth()
	for i, c := range cols {
		t.AddColumnText(t.colWidth(int32(i), w, t.sbWide), fmt.Sprintf("name%d", i+1), c.Title)
	}
	t.CreateAdapter(int32(len(cols)))
	t.lastW = -1
	t.applyTheme(xui.Current()) // 重设表头/单元格字体、边距与对齐
	t.relayout(true)
	t.Redraw(false)
	return t
}

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

// relayout 在"是否需要滚动条"翻转、或元素宽度真的变了时重新分配自适应列宽 —— 滚动条出现后
// 不让出宽度的话最后一列会被盖住, 没滚动条时白留空隙同样难看。force 为 true 时跳过"不变就返回"
// 的短路。宽度没变的重复调用会被 lastW 挡掉, 因此元素尺寸事件里调用不会来回震荡。
func (t *Table) relayout(force bool) {
	if t.autoCol < 0 {
		return
	}
	w := t.GetWidth()
	need := t.needScroll()
	if !force && t.sbWide == need && t.lastW == w {
		return
	}
	t.sbWide, t.lastW = need, w
	t.SetColumnWidth(t.autoCol, t.colWidth(t.autoCol, w, need))
	t.Redraw(false)
}

// FitColumns 按元素**当前**宽度重算自适应列宽。
// 元素尺寸变化已由组件内部(尺寸事件)自动处理, 一般不需要手动调用; 保留它是为了在
// 特殊时序下(例如批量改完数据后立刻量宽度)能强制重算一次。
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
			t.styleCell(r, c, th)
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
	for i := range t.cols {
		t.styleCell(pItem.Index, int32(i), th)
	}
	return 0
}

// styleCell 把主题字体/字色/对齐下发给某一格的形状文本。
// 对齐按列(TableColumn.Align)设置: 右/居中对齐时对侧也留 px-2 内边距,
// 否则贴到单元格边缘(与表头 px-2 的观感不一致)。
func (t *Table) styleCell(row, col int32, th *xui.Theme) {
	h := t.GetTemplateObject(row, col, 1)
	if h == 0 {
		return
	}
	font := xui.FontHandle(th.FontDefault)
	xc.XShapeText_SetTextColor(h, uint32(th.Foreground))
	if font != 0 {
		xc.XShapeText_SetFont(h, font)
	}
	align := xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter
	padLeft, padRight := tableCellPadX, int32(0)
	if col >= 0 && int(col) < len(t.cols) {
		switch t.cols[col].Align {
		case DataAlignRight:
			align = xcc.TextAlignFlag_Right | xcc.TextAlignFlag_Vcenter
			padRight = tableCellPadX
		case DataAlignCenter:
			align = xcc.TextAlignFlag_Center | xcc.TextAlignFlag_Vcenter
			padRight = tableCellPadX
		}
	}
	xc.XShapeText_SetTextAlign(h, align)
	xc.XWidget_LayoutItem_SetMargin(h, padLeft, 0, padRight, 0) // p-2
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
	hover := pItem.NState == xcc.List_Item_State_Stay
	if pItem.NState == xcc.List_Item_State_Select {
		p.FillRound(&rc, th.Muted, 0) // data-[state=selected]:bg-muted
	} else if hv := xui.HoverItem(hEle, pItem.Index, hover); hv > 0 {
		p.FillRound(&rc, th.Muted.ScaleAlpha(0.5*hv), 0) // hover:bg-muted/50
	}
	// 富单元格: 在底色之上、行底边之下(底边不能被单元格内容盖住)。
	if t.onCellExtra != nil {
		t.onCellExtra(p, rc, pItem.Index, pItem.ISubItem, hover)
	}
	// 行底边: 最后一行不画(tbody 的 [&_tr:last-child]:border-0)
	if pItem.Index < t.Rows()-1 {
		p.Line(rc.Left, rc.Bottom-1, rc.Right, rc.Bottom-1, th.Border, 1)
	}
	return 0
}

// ===== 表头交互(悬停底色 / 列拖拽) =====

// tableHeaderDragMin 是列拖拽的起判距离: 小于它按单击处理(打开列头菜单)。
const tableHeaderDragMin int32 = 4

// headerColAt 返回表头内 x 坐标命中的列, -1 = 落在列缝或表头之外。
func (t *Table) headerColAt(x int32) int32 {
	for i, rc := range t.hdrRects {
		if x >= rc.Left && x < rc.Right {
			return int32(i)
		}
	}
	return -1
}

// headerDropIndex 把表头内 x 坐标翻成插入边界(0..列数): 越过某列中线 = 插到它右侧。
func (t *Table) headerDropIndex(x int32) int32 {
	for i, rc := range t.hdrRects {
		if rc.Right <= rc.Left {
			continue
		}
		if x < (rc.Left+rc.Right)/2 {
			return int32(i)
		}
	}
	return int32(len(t.cols))
}

// headerActive 报告该表头列是否可交互(悬停底色 / 按下 / 拖拽共用这一口径)。
func (t *Table) headerActive(col int32) bool {
	return col >= 0 && t.onHeaderActive != nil && t.onHeaderActive(col)
}

func (t *Table) redrawHeader() {
	if t.headerEle != nil {
		t.headerEle.Redraw(false)
	}
}

func (t *Table) resetHeaderDrag() {
	t.dragFrom, t.dragTo, t.dragDownX, t.dragMoved = -1, -1, 0, false
}

func (t *Table) onHeaderMouseMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pPt == nil {
		return 0
	}
	if t.dragFrom >= 0 {
		// 松手发生在表头之外时收不到 Up: 回到表头时左键已经不在了, 顺手收掉拖拽状态。
		if nFlags&0x0001 == 0 {
			t.resetHeaderDrag()
			t.headerDown = -1
			t.redrawHeader()
			return 0
		}
		if !t.dragMoved {
			dx := pPt.X - t.dragDownX
			if dx < 0 {
				dx = -dx
			}
			if dx < tableHeaderDragMin {
				return 0
			}
			t.dragMoved = true
			// 在起拖时就置位: 原生的表头点击可能先于 Up 处理器派发, 等 Up 再置位就晚了。
			t.dragSwallowClick = true
			t.headerHover = -1
		}
		if to := t.headerDropIndex(pPt.X); to != t.dragTo {
			t.dragTo = to
			t.redrawHeader()
		}
		return 0
	}
	col := t.headerColAt(pPt.X)
	if !t.headerActive(col) {
		col = -1
	}
	if col != t.headerHover {
		t.headerHover = col
		t.redrawHeader()
	}
	return 0
}

func (t *Table) onHeaderMouseLeave(hEle, hEleStay int, pbHandled *bool) int {
	if t.headerHover != -1 || t.headerDown != -1 {
		t.headerHover, t.headerDown = -1, -1
		t.redrawHeader()
	}
	return 0
}

func (t *Table) onHeaderMouseDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pPt == nil {
		return 0
	}
	t.dragSwallowClick = false
	col := t.headerColAt(pPt.X)
	if !t.headerActive(col) {
		if t.headerDown != -1 {
			t.headerDown = -1
			t.redrawHeader()
		}
		return 0
	}
	t.headerDown, t.headerHover = col, col
	if t.onColumnsMoved != nil {
		t.dragFrom, t.dragTo, t.dragDownX, t.dragMoved = col, col, pPt.X, false
	}
	t.redrawHeader()
	return 0
}

func (t *Table) onHeaderMouseUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	t.headerDown = -1
	if t.dragFrom >= 0 && t.dragMoved {
		from, to := t.dragFrom, t.dragTo
		// 拖拽结束后原生可能仍补发一次表头点击: 标记吞掉它(下一次按下会清掉)。
		t.dragSwallowClick = true
		t.resetHeaderDrag()
		if t.onColumnsMoved != nil && to != from && to != from+1 {
			t.onColumnsMoved(from, to)
		}
	} else {
		t.resetHeaderDrag()
	}
	t.redrawHeader()
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
	if idx := int(pItem.Index); idx >= 0 {
		if idx >= len(t.hdrRects) {
			t.hdrRects = append(t.hdrRects, make([]xc.RECT, idx+1-len(t.hdrRects))...)
		}
		t.hdrRects[idx] = rc
	}
	p.FillRound(&rc, t.surface(th), 0)
	// 可交互表头的悬停 / 按下底色: 原生表头不重绘, 悬停值由表头元素上的鼠标事件驱动
	// (见 8.176); 底色只铺在单元格内侧, 与 shadcn 的表头按钮观感一致。
	if t.headerActive(pItem.Index) {
		active := t.headerHover == pItem.Index || t.headerDown == pItem.Index
		// 过渡必须挂在**表头元素**上: 表头绘制回调的 hEle 是 List 本体, 与行悬停
		// 共用动画键(owner + 索引) —— 悬停第 N 个表头会把第 N 行点亮(实测)。
		hoverOwner := hEle
		if t.headerEle != nil {
			hoverOwner = t.headerEle.Handle
		}
		hv := xui.HoverItem(hoverOwner, pItem.Index, active)
		alpha := 0.5 * hv
		if t.headerDown == pItem.Index {
			alpha = 1 // 按下立即到位, 不等过渡
		}
		if alpha > 0 {
			bg := rc
			bg.Left += 2
			bg.Right -= 2
			bg.Top += 4
			bg.Bottom -= 4
			if bg.Right > bg.Left && bg.Bottom > bg.Top {
				p.FillRound(&bg, th.Muted.ScaleAlpha(alpha), 6)
			}
		}
	}
	p.Line(rc.Left, rc.Bottom-1, rc.Right, rc.Bottom-1, th.Border, 1)
	if t.onHeaderExtra != nil {
		t.onHeaderExtra(p, rc, pItem.Index)
	}
	// 列拖拽插入位置: 边界处画一条主色竖线(最后一列的右边界单独处理)。
	if t.dragMoved {
		x := int32(-1)
		if pItem.Index == t.dragTo {
			x = rc.Left + 1
		} else if t.dragTo >= int32(len(t.cols)) && pItem.Index == int32(len(t.cols))-1 {
			x = rc.Right - 1
		}
		if x >= 0 {
			p.Line(x, rc.Top+4, x, rc.Bottom-4, th.Primary, 2)
		}
	}
	return 0
}

// styleScrollBars 把纵向滚动条收成细条 + 主题色(几何/圆角/配色见 scrollbar.go: 10px 轨道 +
// 8px 圆头滑块; 嵌入式皮肤 = 前景色淡化 + 悬停加深)。滚动条颜色只能通过背景管理器下发
// 给原生对象, 因此换肤要重刷一遍(Clear 后重加, 不会累积超过炫彩的 20 个背景对象上限)。
func (t *Table) styleScrollBars(th *xui.Theme) {
	styleScrollBar(t.GetScrollBarVObj(), uint32(t.surface(th)), embedScrollBarSkin(th))
}
