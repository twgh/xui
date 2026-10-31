package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Data Table 几何(逻辑像素), 对齐 shadcn data-table 文档的 tasks 示例:
// 工具栏 h-8 / 页脚 h-8 / 排序图标 size-4(这里取 14, 与 14px 表头文字同档)。
const (
	dataTableToolbarH int32 = 32
	dataTableFooterH  int32 = 32
	dataTableGap      int32 = 8
	dataTableSortIcon int32 = 14
	dataTableSortGap  int32 = 6
	// dataTableSelectW 是勾选列宽(w-8): 16 的复选框 + 单元格 p-2(8×2)。
	dataTableSelectW int32 = 32
	// dataTableCheckSize 是勾选框边长(shadcn checkbox 的 size-4)。
	dataTableCheckSize int32 = 16
	// dataTableToggleW 是工具栏"视图"按钮宽(图标 + "视图" 两字)。
	dataTableToggleW int32 = 88
)

// DataSort 排序方向。
type DataSort int

const (
	DataSortNone DataSort = iota // 不排序(保持原始顺序)
	DataSortAsc                  // 升序
	DataSortDesc                 // 降序
)

// DataColumn 数据表格的列定义。
type DataColumn struct {
	Title string
	// Width 列宽(逻辑像素)。<= 0 表示占满剩余宽度(同 TableColumn); 多个 <= 0 时只有
	// 第一个吃掉剩余宽度, 其余按 120 固定宽排。
	Width int32
	// Sortable 允许点击表头排序: 每次点击在 升序 → 降序 → 不排序 之间循环
	// (同 TanStack 的 enableSortingRemoval 默认语义)。
	Sortable bool
	// Compare 覆盖该列的排序比较(负数 = a 在前)。nil = 自动: 两侧都能解析成数字
	// (允许货币符号 / 千分位 / 百分号)时按数值, 否则按大小写不敏感的自然顺序
	// (字符串里的数字段按数值比较, 同 localeCompare 的 numeric: true)。
	Compare func(a, b string) int
	// Align 原生文本的水平对齐(默认左对齐)。金额 / 数字列用 DataAlignRight。
	Align DataAlign
	// Render 自定义单元格绘制(徽章 / 状态图标 / 行操作钮), 对应 tasks 示例里
	// `columns.tsx` 的 cell 渲染函数。设置后该列**不再向原生 List 写文本** ——
	// 原生 shape text 是独立子元素, 会叠在自绘内容上; 所以回调必须自己画文字。
	Render func(p *xui.Painter, cell DataCell)
	// Hidden 初始隐藏, 可在工具栏"视图"下拉里重新勾出(需要 WithDataTableColumnToggle)。
	Hidden bool
}

// DataCell 传给 DataColumn.Render 的单元格信息。
type DataCell struct {
	Row      int        // 原始数据行下标(不是当前页里的下标)
	Col      int        // 列下标(DataColumn 序列, 含隐藏列)
	Text     string     // 该单元格的原始文本(未经过任何格式化)
	Rc       xc.RECT    // 单元格矩形(逻辑像素, 相对表格元素; 内容按 p-2 自己内缩 8)
	Hover    bool       // 所在行是否处于悬停态
	Selected bool       // 所在行是否被勾选
	DTable   *DataTable // 反向引用(便于回调里改选择状态)
}

// DataTableOption 组件配置项。
type DataTableOption func(*DataTable)

// WithDataTableSearch 启用顶部关键字筛选框并指定占位符(默认已启用, 占位符 "筛选...")。
func WithDataTableSearch(placeholder string) DataTableOption {
	return func(d *DataTable) {
		d.searchOn = true
		if placeholder != "" {
			d.searchPlaceholder = placeholder
		}
	}
}

// WithDataTableNoSearch 去掉顶部筛选框(表格占满整个高度)。
func WithDataTableNoSearch() DataTableOption {
	return func(d *DataTable) { d.searchOn = false }
}

// WithDataTablePageSize 设置初始每页行数(默认 10)。
func WithDataTablePageSize(n int) DataTableOption {
	return func(d *DataTable) {
		if n > 0 {
			d.pageSize = n
		}
	}
}

// WithDataTablePageSizes 设置"每页行数"下拉的候选值(默认 10/20/30/40/50)。
func WithDataTablePageSizes(sizes ...int) DataTableOption {
	return func(d *DataTable) {
		out := make([]int, 0, len(sizes))
		for _, n := range sizes {
			if n > 0 {
				out = append(out, n)
			}
		}
		if len(out) > 0 {
			d.pageSizes = out
		}
	}
}

// WithDataTableFilterColumns 限定关键字筛选只匹配指定的列(下标); 不调用 = 匹配所有列。
func WithDataTableFilterColumns(cols ...int) DataTableOption {
	return func(d *DataTable) { d.filterCols = append([]int(nil), cols...) }
}

// WithDataTableSelection 启用行勾选: 表格最左侧多一列复选框(表头是全选 / 不确定态),
// 勾选状态按**原始数据行**保存 —— 翻页 / 排序 / 筛选都不会丢。
//
// 勾选列的点击只切勾选, 不再让原生 List 单选高亮(行点击与勾选解耦, 见 DESIGN 8.170)。
func WithDataTableSelection() DataTableOption {
	return func(d *DataTable) { d.selectOn = true }
}

// WithDataTableColumnToggle 在工具栏右侧加"视图"下拉(Menu.Check 列表), 逐列显隐。
// 列显隐会重建 List 列结构并重加当前页(见 Table.SetColumns / DESIGN 8.169)。
func WithDataTableColumnToggle() DataTableOption {
	return func(d *DataTable) { d.toggleOn = true }
}

// DataTable 数据表格: 在 Table 之上提供 **列排序 / 关键字筛选 / 分页**。
//
// 对齐 shadcn 的 Data Table 模式(TanStack Table 的 tasks 示例):
//   - 可排序列的表头点击后在 升序 → 降序 → 不排序 之间循环, 图标依次是
//     arrow-up-down(muted) / arrow-up / arrow-down;
//   - 顶部筛选框输入即过滤(不区分大小写, 子串匹配), 过滤后回到第一页;
//   - 页脚: 左侧总行数(筛选中显示"筛选出 x / y 行"), 右侧每页行数下拉 +
//     第 x / y 页 + 首页/上一页/下一页/末页按钮, 到端点自动禁用;
//   - 排序列与筛选条件变化都会回到第一页(autoResetPageIndex 语义)。
//
// 排序与分页都在 Go 侧完成, **不启用原生 List 的 SetSort** —— 原生的适配器排序
// 只认列数据本身的类型, 与单元格显示文本(格式化过的金额/日期)和自定义比较器
// 对不上, 而且没法做"第三次点击回到原始顺序"。
//
// 表格行高/外观沿用 Table; cy 为 0 时按"工具栏 + 每页行数 + 页脚"自动配高。
type DataTable struct {
	*widget.LayoutEle

	cols    []DataColumn
	rows    [][]string
	view    []int // 过滤 + 排序后的原始行下标
	visible []int // 当前显示的列(d.cols 下标); 列显隐改变的只是它

	sortCol int
	sortDir DataSort

	// 行勾选: selected 的键是**原始数据行**下标, 翻页 / 排序 / 筛选都不影响它。
	selectOn bool
	selected map[int]bool

	toggleOn bool
	viewBtn  *Button

	filter     string
	filterCols []int

	page      int // 0 起
	pageSize  int
	pageSizes []int

	searchOn          bool
	searchPlaceholder string

	search  *Input
	table   *Table
	toolbar *widget.LayoutEle
	footer  *widget.LayoutEle

	totalLabel   *Label
	perPageLabel *Label
	pageLabel    *Label
	sizeSel      *Select

	firstBtn *Button
	prevBtn  *Button
	nextBtn  *Button
	lastBtn  *Button

	suppressSearch bool

	onSort   func(col int, dir DataSort)
	onPage   func(page int)
	onFilter func(text string)
	onRow    func(row int)
	onCell   func(row, col int)
	onSel    func(rows []int)
}

// NewDataTable 创建数据表格。cols 末列 Width<=0 时占满剩余宽度。
// cy 传 0: 高度按默认每页行数配好; cy>0: 表格占满工具栏/页脚之间的剩余高度。
func NewDataTable(x, y, cx, cy int32, hParent int, cols []DataColumn, opts ...DataTableOption) *DataTable {
	if len(cols) == 0 {
		cols = []DataColumn{{Title: ""}}
	}
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	d := &DataTable{
		LayoutEle:         ele,
		cols:              append([]DataColumn(nil), cols...), // 列显隐会改 Hidden, 别改到调用方的切片
		sortCol:           -1,
		pageSize:          10,
		pageSizes:         []int{10, 20, 30, 40, 50},
		searchOn:          true,
		searchPlaceholder: "筛选...",
	}
	for _, o := range opts {
		o(d)
	}
	if d.pageSize <= 0 {
		d.pageSize = 10
	}
	if !dataHasInt(d.pageSizes, d.pageSize) {
		d.pageSizes = append([]int{d.pageSize}, d.pageSizes...)
	}
	d.selected = map[int]bool{}
	d.visible = d.initVisible()

	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(dataTableGap)
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)

	// cy=0: 高度按"工具栏 + 每页行数 + 页脚"配好。必须把根元素的初始高度也设上 ——
	// 根是 Auto 高时, 里面 Weight 高的表格不会撑出高度(整块塌成 0, 实测见 DESIGN 8.164)。
	if cy <= 0 {
		total := d.minBodyH() + dataTableFooterH + dataTableGap
		if d.searchOn || d.toggleOn {
			total += dataTableToolbarH + dataTableGap
		}
		ele.SetHeight(total)
	}

	d.buildToolbar()
	d.buildTable()
	d.buildFooter()

	d.apply(false)
	xui.AttachWindow(ele.Handle)
	return d
}

func (d *DataTable) buildToolbar() {
	if !d.searchOn && !d.toggleOn {
		return
	}
	d.toolbar = xui.HStack(d.Handle, dataTableToolbarH, 0)
	if d.searchOn {
		d.search = NewInput(0, 0, 0, dataTableToolbarH, d.toolbar.Handle,
			WithPlaceholder(d.searchPlaceholder),
			WithLeadingIcon(icons.IconSearch),
			WithClearButton())
		xui.ItemWeight(d.search, 1)
		d.search.OnChange(func(text string) {
			if d.suppressSearch {
				return
			}
			d.filter = text
			d.apply(true)
			if d.onFilter != nil {
				d.onFilter(text)
			}
		})
	}
	if d.toggleOn {
		// 对应 tasks 示例工具栏右侧的 View 下拉(DataTableViewOptions): 逐列显隐。
		d.viewBtn = NewButton(0, 0, 0, 0, "视图", d.toolbar.Handle,
			WithVariant(ButtonOutline), WithSize(ButtonSizeSm),
			WithButtonLeadingIcon(icons.IconSlidersHorizontal))
		xui.ItemSize(d.viewBtn, dataTableToggleW, 28)
		xui.ItemMargin(d.viewBtn, dataTableGap, 2, 0, 2)
		d.viewBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			d.showViewMenu(hEle)
			return 0
		})
	}
}

// initVisible 初始化可见列: 过滤掉 Hidden, 并保证至少留一列(全隐藏等于表格变空壳)。
func (d *DataTable) initVisible() []int {
	out := make([]int, 0, len(d.cols))
	for i, c := range d.cols {
		if !c.Hidden {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		out = append(out, 0)
		d.cols[0].Hidden = false
	}
	return out
}

// tableColumns 把可见列翻成 Table 的列定义(勾选列在前)。
func (d *DataTable) tableColumns() []TableColumn {
	tc := make([]TableColumn, 0, len(d.visible)+1)
	if d.selectOn {
		tc = append(tc, TableColumn{Title: "", Width: dataTableSelectW})
	}
	for _, ci := range d.visible {
		c := d.cols[ci]
		tc = append(tc, TableColumn{Title: c.Title, Width: c.Width, Align: c.Align})
	}
	return tc
}

// colAt 把 Table 的列下标翻成 d.cols 下标(勾选列返回 -1)。
func (d *DataTable) colAt(tcol int32) int {
	i := int(tcol)
	if d.selectOn {
		i--
	}
	if i < 0 || i >= len(d.visible) {
		return -1
	}
	return d.visible[i]
}

func (d *DataTable) buildTable() {
	d.table = NewTable(0, 0, 0, 0, d.Handle, d.tableColumns())
	xui.ItemFill(d.table)
	// 表格吃掉工具栏/页脚之间的剩余高度; 高度 Auto 的父级里 Weight 会退化成 0
	// (DESIGN 8.55), cy=0 的情况由构造函数的"根初始高度"兜底(见 8.164)。
	xui.ItemFillHeight(d.table)
	xui.ItemMinSize(d.table, 0, d.minBodyH())

	d.table.onHeaderExtra = d.paintHeaderExtra
	d.table.onCellExtra = d.paintCell
	d.table.AddEvent_List_Header_Click(func(hEle int, iItem int32, pbHandled *bool) int {
		if d.selectOn && iItem == 0 {
			d.togglePageSelection()
			*pbHandled = true
			return 0
		}
		if ci := d.colAt(iItem); ci >= 0 && d.cols[ci].Sortable {
			d.toggleSort(ci)
			*pbHandled = true
		}
		return 0
	})

	// 勾选模式下自己在按下时做命中测试: 原生 List 的单选高亮只表示"刚点过",
	// 与勾选无关, 吞掉按下消息不让它改选中行(见 DESIGN 8.170)。
	d.table.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if !d.selectOn || pPt == nil {
			return 0
		}
		var row, sub int32
		if !d.table.HitTestOffset(pPt, &row, &sub) {
			return 0
		}
		raw := d.rawRow(row)
		if raw < 0 {
			return 0
		}
		*pbHandled = true
		if int(sub) == 0 { // 勾选列
			d.toggleRowSelected(raw)
			return 0
		}
		if ci := d.colAt(sub); ci >= 0 && d.onCell != nil {
			d.onCell(raw, ci)
		}
		if d.onRow != nil {
			d.onRow(raw)
		}
		return 0
	})

	// 空状态 / 排序图标都画在表格元素上: List 的内容区由原生管理, 往里塞覆盖元素
	// 会被原生布局摆到奇怪的位置(见 DESIGN 8.162)。
	d.table.EnableEvent_XE_PAINT_END()
	d.table.AddEvent_Paint_End(d.paintEmpty)

	d.table.OnSelect(func(row int32) {
		if row < 0 || d.onRow == nil {
			return
		}
		idx := d.page*d.pageSize + int(row)
		if idx >= 0 && idx < len(d.view) {
			d.onRow(d.view[idx])
		}
	})
}

func (d *DataTable) buildFooter() {
	f := xui.HStack(d.Handle, dataTableFooterH, 8)
	d.footer = f

	d.totalLabel = NewLabel(0, 0, 0, 20, "", f.Handle).SetMuted()
	xui.ItemWeight(d.totalLabel, 1)

	d.perPageLabel = NewLabel(0, 0, 36, 20, "每页", f.Handle).SetMuted()
	xui.ItemWidth(d.perPageLabel, 36)

	d.sizeSel = NewSelect(0, 0, 72, 28, f.Handle,
		WithSelectItems(d.sizeItems()),
		WithSelectValue(strconv.Itoa(d.pageSize)),
		WithSelectSize(SelectSizeSm))
	xui.ItemSize(d.sizeSel, 72, 28)
	d.sizeSel.OnChange(func(it SelectItem) {
		if n, err := strconv.Atoi(it.Value); err == nil && n > 0 {
			d.SetPageSize(n)
		}
	})

	d.pageLabel = NewLabel(0, 0, 112, 20, "", f.Handle).SetMuted()
	xui.ItemWidth(d.pageLabel, 112)

	d.firstBtn = newDataTableNav(f.Handle, icons.IconChevronsLeft)
	d.prevBtn = newDataTableNav(f.Handle, icons.IconChevronLeft)
	d.nextBtn = newDataTableNav(f.Handle, icons.IconChevronRight)
	d.lastBtn = newDataTableNav(f.Handle, icons.IconChevronsRight)

	d.firstBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int { d.SetPage(0); return 0 })
	d.prevBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int { d.SetPage(d.page - 1); return 0 })
	d.nextBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int { d.SetPage(d.page + 1); return 0 })
	d.lastBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int { d.SetPage(d.PageCount() - 1); return 0 })
}

func newDataTableNav(parent int, icon string) *Button {
	b := NewButton(0, 0, 0, 0, "", parent,
		WithVariant(ButtonOutline), WithSize(ButtonSizeIcon), WithIcon(icon))
	xui.ItemSize(b, 32, 32)
	return b
}

// minBodyH 是表格体"(表头 + 每页行数)所需的最小高度。
func (d *DataTable) minBodyH() int32 {
	return tableHeaderH + tableInset*2 + int32(d.pageSize)*tableRowH
}

func (d *DataTable) sizeItems() []SelectItem {
	items := make([]SelectItem, 0, len(d.pageSizes))
	for _, n := range d.pageSizes {
		items = append(items, SelectItem{Value: strconv.Itoa(n), Text: strconv.Itoa(n)})
	}
	return items
}

// ===== 数据 API =====

// SetRows 替换全部数据(行 = 各列显示文本)。
func (d *DataTable) SetRows(rows [][]string) *DataTable {
	d.rows = rows
	d.apply(true)
	return d
}

// AddRow 追加一行。
func (d *DataTable) AddRow(cells ...string) *DataTable {
	d.rows = append(d.rows, cells)
	d.apply(false)
	return d
}

// Refresh 重新计算过滤/排序/分页并重绘(外部直接改过 rows 后调用)。
func (d *DataTable) Refresh() *DataTable {
	d.apply(false)
	return d
}

// SetFilter 设置关键字筛选(不区分大小写); 空串 = 不过滤。回到第一页。
func (d *DataTable) SetFilter(s string) *DataTable {
	d.filter = s
	if d.search != nil && d.search.Value() != s {
		d.suppressSearch = true
		d.search.SetValue(s)
		d.suppressSearch = false
		// 清除按钮的命中区是在 paint 里重建的: 程序化改值后不重绘的话,
		// 画面上出现了 ×, 点它却没有反应(parts 还是空的)。
		d.search.Redraw(false)
	}
	d.apply(true)
	if d.onFilter != nil {
		d.onFilter(s)
	}
	return d
}

// Filter 返回当前关键字。
func (d *DataTable) Filter() string { return d.filter }

// SetSort 程序化设置排序列(下标; -1 = 不排序)与方向。
func (d *DataTable) SetSort(col int, dir DataSort) *DataTable {
	if col < 0 || col >= len(d.cols) || dir == DataSortNone {
		col, dir = -1, DataSortNone
	}
	d.sortCol, d.sortDir = col, dir
	d.apply(true)
	if d.onSort != nil {
		d.onSort(d.sortCol, d.sortDir)
	}
	return d
}

// Sort 返回当前排序列(未排序为 -1)与方向。
func (d *DataTable) Sort() (int, DataSort) { return d.sortCol, d.sortDir }

// Page 返回当前页(0 起)。
func (d *DataTable) Page() int { return d.page }

// PageCount 返回总页数(至少 1)。
func (d *DataTable) PageCount() int { return dataPageCount(len(d.view), d.pageSize) }

// SetPage 跳到指定页(越界自动夹取), 触发 OnPageChange。
func (d *DataTable) SetPage(p int) *DataTable {
	d.page = p
	d.clampPage()
	d.render()
	if d.onPage != nil {
		d.onPage(d.page)
	}
	return d
}

// PageSize 返回每页行数。
func (d *DataTable) PageSize() int { return d.pageSize }

// SetPageSize 修改每页行数, 回到第一页。
func (d *DataTable) SetPageSize(n int) *DataTable {
	if n <= 0 || n == d.pageSize {
		return d
	}
	d.pageSize = n
	if !dataHasInt(d.pageSizes, n) {
		d.pageSizes = append([]int{n}, d.pageSizes...)
		if d.sizeSel != nil {
			d.sizeSel.SetItems(d.sizeItems())
		}
	}
	d.apply(true)
	if d.onPage != nil {
		d.onPage(d.page)
	}
	return d
}

// TotalRows 返回全部行数。
func (d *DataTable) TotalRows() int { return len(d.rows) }

// FilteredRows 返回过滤后的行数。
func (d *DataTable) FilteredRows() int { return len(d.view) }

// OnSortChange 注册排序变化回调(col = -1 表示取消排序)。
func (d *DataTable) OnSortChange(fn func(col int, dir DataSort)) *DataTable {
	d.onSort = fn
	return d
}

// OnPageChange 注册翻页回调(page 从 0 起)。
func (d *DataTable) OnPageChange(fn func(page int)) *DataTable {
	d.onPage = fn
	return d
}

// OnFilterChange 注册筛选变化回调(用户输入与程序化 SetFilter 都会触发)。
func (d *DataTable) OnFilterChange(fn func(text string)) *DataTable {
	d.onFilter = fn
	return d
}

// OnRowClick 注册行点击回调, row 是**原始数据**的行下标(不是当前页里的下标)。
func (d *DataTable) OnRowClick(fn func(row int)) *DataTable {
	d.onRow = fn
	return d
}

// OnCellClick 注册单元格点击回调(需要 WithDataTableSelection, 这一模式下原生 List
// 的单选被接管)。row 是原始行下标, col 是 d.cols 下标 —— 行操作列靠它区分。
func (d *DataTable) OnCellClick(fn func(row, col int)) *DataTable {
	d.onCell = fn
	return d
}

// ===== 行勾选 =====

// OnSelectionChange 注册勾选变化回调(rows 是升序的原始行下标快照)。
func (d *DataTable) OnSelectionChange(fn func(rows []int)) *DataTable {
	d.onSel = fn
	return d
}

// Selected 返回被勾选的原始行下标(升序)。
func (d *DataTable) Selected() []int {
	out := make([]int, 0, len(d.selected))
	for raw, on := range d.selected {
		if on {
			out = append(out, raw)
		}
	}
	sort.Ints(out)
	return out
}

// IsSelected 报告某原始行是否被勾选。
func (d *DataTable) IsSelected(row int) bool { return d.selected[row] }

// SelectedCount 返回被勾选且**在当前筛选结果里**的行数(页脚分母口径)。
func (d *DataTable) SelectedCount() int { return d.selectedInView() }

// SetSelected 用给定原始行下标覆盖勾选集合。
func (d *DataTable) SetSelected(rows ...int) *DataTable {
	d.selected = make(map[int]bool, len(rows))
	for _, r := range rows {
		if r >= 0 && r < len(d.rows) {
			d.selected[r] = true
		}
	}
	d.afterSelectionChange(true)
	return d
}

// ClearSelection 清空勾选。
func (d *DataTable) ClearSelection() *DataTable {
	d.selected = map[int]bool{}
	d.afterSelectionChange(true)
	return d
}

// ===== 列显隐 =====

// SetColumnVisible 显示 / 隐藏某列(d.cols 下标)。至少保留一列 —— 全隐藏时调用被忽略。
func (d *DataTable) SetColumnVisible(col int, on bool) *DataTable {
	if col < 0 || col >= len(d.cols) || d.cols[col].Hidden == !on {
		return d
	}
	if !on && len(d.visible) <= 1 {
		return d
	}
	d.cols[col].Hidden = !on
	d.visible = d.initVisible()
	d.rebuildColumns()
	return d
}

// ColumnVisible 返回某列当前是否显示。
func (d *DataTable) ColumnVisible(col int) bool {
	return col >= 0 && col < len(d.cols) && !d.cols[col].Hidden
}

// SetColumns 替换列定义(列名 / 宽度 / 排序 / 渲染函数全部重来), 勾选状态保留。
func (d *DataTable) SetColumns(cols []DataColumn) *DataTable {
	if len(cols) == 0 {
		return d
	}
	d.cols = append([]DataColumn(nil), cols...)
	d.visible = d.initVisible()
	if d.sortCol >= len(d.cols) {
		d.sortCol, d.sortDir = -1, DataSortNone
	}
	d.rebuildColumns()
	return d
}

// rebuildColumns 重建 List 列结构并重加当前页(列数变了必须重建适配器, 见 8.169)。
func (d *DataTable) rebuildColumns() {
	if d.table == nil {
		return
	}
	d.table.SetColumns(d.tableColumns())
	d.render()
}

// showViewMenu 弹出"视图"列显隐菜单(Menu.Check 支持多选)。
func (d *DataTable) showViewMenu(hEle int) {
	if d.viewBtn != nil {
		d.viewBtn.SetExpanded(true)
		d.viewBtn.Redraw(true) // 原生菜单阻塞, 不先重绘就看不到展开色
		defer d.viewBtn.SetExpanded(false)
	}
	m := NewMenu()
	m.Width(176)
	m.Label("显示列")
	for i := range d.cols {
		i := i
		m.Check(d.cols[i].Title, !d.cols[i].Hidden, MenuOnCheck(func(on bool) {
			d.SetColumnVisible(i, on)
		}))
	}
	m.PopupAtElement(hEle)
}

// ===== 内部: 勾选状态 =====

// pageSelection 统计当前页的(已勾选数, 行数) —— 表头全选的判断口径(TanStack 的
// getIsAllPageRowsSelected: 全选只管当前页, 不是全部筛选结果)。
func (d *DataTable) pageSelection() (sel, total int) {
	start, end := d.page*d.pageSize, d.page*d.pageSize+d.pageSize
	if start > len(d.view) {
		start = len(d.view)
	}
	if end > len(d.view) {
		end = len(d.view)
	}
	for i := start; i < end; i++ {
		total++
		if d.selected[d.view[i]] {
			sel++
		}
	}
	return sel, total
}

// selectedInView 统计勾选 ∩ 当前筛选结果(页脚分母用筛选后的行数)。
func (d *DataTable) selectedInView() int {
	n := 0
	for _, raw := range d.view {
		if d.selected[raw] {
			n++
		}
	}
	return n
}

func (d *DataTable) toggleRowSelected(raw int) {
	if d.selected == nil {
		d.selected = map[int]bool{}
	}
	if d.selected[raw] {
		delete(d.selected, raw)
	} else {
		d.selected[raw] = true
	}
	d.afterSelectionChange(true)
}

// togglePageSelection 表头点击: 整页勾满则清空整页, 否则把整页勾上。
func (d *DataTable) togglePageSelection() {
	sel, total := d.pageSelection()
	on := sel < total
	start, end := d.page*d.pageSize, d.page*d.pageSize+d.pageSize
	if end > len(d.view) {
		end = len(d.view)
	}
	if d.selected == nil {
		d.selected = map[int]bool{}
	}
	for i := start; i < end; i++ {
		if on {
			d.selected[d.view[i]] = true
		} else {
			delete(d.selected, d.view[i])
		}
	}
	d.afterSelectionChange(true)
}

func (d *DataTable) afterSelectionChange(redraw bool) {
	if d.table != nil && redraw {
		d.table.Redraw(false)
	}
	d.syncFooter()
	if d.onSel != nil {
		d.onSel(d.Selected())
	}
}

// ===== 内部: 视图 / 分页 / 绘制 =====

func (d *DataTable) apply(resetPage bool) {
	var cmp func(a, b string) int
	if d.sortCol >= 0 && d.sortCol < len(d.cols) {
		cmp = d.cols[d.sortCol].Compare
	}
	d.view = dataTableView(d.rows, d.filter, d.filterCols, d.sortCol, d.sortDir, cmp)
	if resetPage {
		d.page = 0
	}
	d.clampPage()
	d.render()
}

func (d *DataTable) clampPage() {
	pages := d.PageCount()
	if d.page >= pages {
		d.page = pages - 1
	}
	if d.page < 0 {
		d.page = 0
	}
}

// rawRow 把表格行下标(当前页内)翻成原始数据行下标; 越界返回 -1。
func (d *DataTable) rawRow(row int32) int {
	i := d.page*d.pageSize + int(row)
	if row < 0 || i < 0 || i >= len(d.view) {
		return -1
	}
	return d.view[i]
}

// cellText 取某原始行某列的文本(行比列短时为空)。
func (d *DataTable) cellText(raw, col int) string {
	if raw < 0 || raw >= len(d.rows) || col < 0 {
		return ""
	}
	row := d.rows[raw]
	if col >= len(row) {
		return ""
	}
	return row[col]
}

func (d *DataTable) render() {
	if d.table == nil {
		return
	}
	d.table.Clear()
	start, end := d.page*d.pageSize, d.page*d.pageSize+d.pageSize
	if start > len(d.view) {
		start = len(d.view)
	}
	if end > len(d.view) {
		end = len(d.view)
	}
	for i := start; i < end; i++ {
		raw := d.view[i]
		cells := make([]string, 0, len(d.visible)+1)
		if d.selectOn {
			cells = append(cells, "") // 勾选列: 全部自绘
		}
		for _, ci := range d.visible {
			// 自绘列不写原生文本: 原生 shape text 会在自绘内容上再叠一层(见 DESIGN 8.168)。
			if ci >= 0 && ci < len(d.cols) && d.cols[ci].Render != nil {
				cells = append(cells, "")
				continue
			}
			cells = append(cells, d.cellText(raw, ci))
		}
		d.table.AddRow(cells...)
	}
	d.table.Refresh()
	d.syncFooter()
}

func (d *DataTable) syncFooter() {
	pages := d.PageCount()
	if d.totalLabel != nil {
		text := fmt.Sprintf("共 %d 行", len(d.rows))
		if strings.TrimSpace(d.filter) != "" {
			text = fmt.Sprintf("筛选出 %d / %d 行", len(d.view), len(d.rows))
		}
		// 有勾选时优先显示"已选 x / y 行"(tasks 示例页脚的 "{n} of {m} row(s) selected"),
		// y 取筛选后的行数: 筛选把行藏起来了, 拿总数当分母对不上。
		if d.selectOn {
			if n := d.selectedInView(); n > 0 {
				text = fmt.Sprintf("已选 %d / %d 行", n, len(d.view))
			}
		}
		d.totalLabel.SetText(text)
		d.totalLabel.Redraw(false)
	}
	if d.pageLabel != nil {
		d.pageLabel.SetText(fmt.Sprintf("第 %d / %d 页", d.page+1, pages))
		d.pageLabel.Redraw(false)
	}
	atFirst, atLast := d.page <= 0, d.page >= pages-1
	for _, n := range []struct {
		b  *Button
		on bool
	}{
		{d.firstBtn, !atFirst}, {d.prevBtn, !atFirst},
		{d.nextBtn, !atLast}, {d.lastBtn, !atLast},
	} {
		if n.b == nil {
			continue
		}
		n.b.Enable(n.on)
		n.b.Redraw(false)
	}
	if d.sizeSel != nil {
		d.sizeSel.SetValue(strconv.Itoa(d.pageSize))
	}
}

// toggleSort 按 升序 → 降序 → 不排序 循环(同 TanStack 的默认排序移除语义)。
func (d *DataTable) toggleSort(col int) {
	d.sortCol, d.sortDir = nextDataSort(d.sortCol, d.sortDir, col)
	d.apply(true)
	if d.onSort != nil {
		d.onSort(d.sortCol, d.sortDir)
	}
}

func (d *DataTable) paintHeaderExtra(p *xui.Painter, rc xc.RECT, tcol int32) {
	if d.selectOn && tcol == 0 {
		// 表头全选: 整页勾满 = 选中, 部分勾选 = 不确定态(减号)
		sel, total := d.pageSelection()
		d.paintCheckBox(p, rc, total > 0 && sel >= total, sel > 0 && sel < total)
		return
	}
	ci := d.colAt(tcol)
	if ci < 0 || ci >= len(d.cols) || !d.cols[ci].Sortable {
		return
	}
	th := xui.Current()
	path, color := icons.IconArrowUpDown, th.MutedForeground
	if ci == d.sortCol {
		switch d.sortDir {
		case DataSortAsc:
			path, color = icons.IconArrowUp, th.Foreground
		case DataSortDesc:
			path, color = icons.IconArrowDown, th.Foreground
		}
	}
	font := xui.FontHandle(th.FontDefault)
	x := rc.Left + tableCellPadX + xui.TextWidth(d.cols[ci].Title, font) + dataTableSortGap
	if limit := rc.Right - tableCellPadX - dataTableSortIcon; x > limit {
		x = limit
	}
	top := rc.Top + (rc.Bottom-rc.Top-dataTableSortIcon)/2
	ir := xc.RECT{Left: x, Top: top, Right: x + dataTableSortIcon, Bottom: top + dataTableSortIcon}
	p.Icon(path, &ir, color, 2)
}

// paintCell 是 Table 的单元格扩展钩子: 勾选列的复选框 + 各列的自定义渲染。
func (d *DataTable) paintCell(p *xui.Painter, rc xc.RECT, row, tcol int32, hover bool) {
	raw := d.rawRow(row)
	if raw < 0 {
		return
	}
	selected := d.selectOn && d.selected[raw]
	if selected {
		// data-[state=selected]:bg-muted —— 勾选行整行铺弱化底色(逐格填 = 整行)
		p.FillRound(&rc, xui.Current().Muted, 0)
	}
	if d.selectOn && tcol == 0 {
		d.paintCheckBox(p, rc, selected, false)
		return
	}
	ci := d.colAt(tcol)
	if ci < 0 || ci >= len(d.cols) {
		return
	}
	if fn := d.cols[ci].Render; fn != nil {
		fn(p, DataCell{
			Row:      raw,
			Col:      ci,
			Text:     d.cellText(raw, ci),
			Rc:       rc,
			Hover:    hover,
			Selected: selected,
			DTable:   d,
		})
	}
}

// paintCheckBox 在矩形内居中画一个复选框(勾选 / 不确定态), 视觉对齐 Checkbox 组件:
// rounded-[4px] border-input, 选中 bg-primary + 白勾/白减号, 暗色未选中 bg-input/30。
func (d *DataTable) paintCheckBox(p *xui.Painter, rc xc.RECT, checked, indeterminate bool) {
	th := xui.Current()
	size := dataTableCheckSize
	left := rc.Left + (rc.Right-rc.Left-size)/2
	top := rc.Top + (rc.Bottom-rc.Top-size)/2
	box := xc.RECT{Left: left, Top: top, Right: left + size, Bottom: top + size}
	if !checked && !indeterminate {
		if th.IsDark() {
			p.FillRound(&box, th.Input.ScaleAlpha(0.3), 4)
		}
		p.StrokeRound(&box, th.Input, 4, 1)
		return
	}
	p.FillRound(&box, th.Primary, 4)
	ir := xc.RECT{Left: box.Left + 1, Top: box.Top + 1, Right: box.Right - 1, Bottom: box.Bottom - 1}
	path := icons.IconCheck
	if indeterminate {
		path = icons.IconMinus
	}
	p.Icon(path, &ir, th.PrimaryForeground, 2)
}

func (d *DataTable) paintEmpty(hEle, hDraw int, pbHandled *bool) int {
	if len(d.view) > 0 {
		return 0
	}
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	th := xui.Current()
	if f := xui.FontHandle(th.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	rc := p.Rc
	top := tableHeaderH + tableInset
	if d.table != nil {
		if hh := d.table.GetHeaderHeight(); hh > 0 {
			top = hh + tableInset
		}
	}
	rc.Top += top
	if rc.Top >= rc.Bottom {
		return 0
	}
	msg := "暂无数据"
	if strings.TrimSpace(d.filter) != "" {
		msg = "没有匹配的数据"
	}
	p.Text(msg, &rc, th.MutedForeground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	return 0
}

// ===== 纯逻辑(单测覆盖) =====

// nextDataSort 是表头点击的排序循环: 新列 → 升序; 同列升序 → 降序; 同列降序 → 取消。
func nextDataSort(curCol int, curDir DataSort, col int) (int, DataSort) {
	switch {
	case curCol != col:
		return col, DataSortAsc
	case curDir == DataSortAsc:
		return col, DataSortDesc
	case curDir == DataSortDesc:
		return -1, DataSortNone
	default:
		return col, DataSortAsc
	}
}

func dataPageCount(n, size int) int {
	if size <= 0 {
		size = 1
	}
	pages := (n + size - 1) / size
	if pages < 1 {
		pages = 1
	}
	return pages
}

// dataTableView 返回过滤 + 排序后的原始行下标。compare 为 nil 时用 dataCompare。
func dataTableView(rows [][]string, filter string, filterCols []int, sortCol int, dir DataSort,
	compare func(a, b string) int) []int {
	f := strings.ToLower(strings.TrimSpace(filter))
	view := make([]int, 0, len(rows))
	for i, row := range rows {
		if f != "" && !dataRowMatches(row, f, filterCols) {
			continue
		}
		view = append(view, i)
	}
	if sortCol < 0 || dir == DataSortNone {
		return view
	}
	value := func(row []string) string {
		if sortCol < len(row) {
			return row[sortCol]
		}
		return ""
	}
	sort.SliceStable(view, func(a, b int) bool {
		av, bv := value(rows[view[a]]), value(rows[view[b]])
		var c int
		if compare != nil {
			c = compare(av, bv)
		} else {
			c = dataCompare(av, bv)
		}
		if dir == DataSortDesc {
			return c > 0
		}
		return c < 0
	})
	return view
}

func dataRowMatches(row []string, filter string, cols []int) bool {
	for i, cell := range row {
		if len(cols) > 0 && !dataHasInt(cols, i) {
			continue
		}
		if strings.Contains(strings.ToLower(cell), filter) {
			return true
		}
	}
	return false
}

// dataCompare 默认列比较: 两侧都是数字时按数值, 否则按自然顺序。
// 空值排最后(升序) —— 与表格里"没填"的行通常沉底的观感一致。
func dataCompare(a, b string) int {
	af, aok := dataNumber(a)
	bf, bok := dataNumber(b)
	if aok && bok {
		switch {
		case af < bf:
			return -1
		case af > bf:
			return 1
		}
		return 0
	}
	as, bs := strings.TrimSpace(a), strings.TrimSpace(b)
	switch {
	case as == "" && bs != "":
		return 1
	case bs == "" && as != "":
		return -1
	}
	return dataNaturalCompare(strings.ToLower(as), strings.ToLower(bs))
}

// dataNumber 解析"带格式的数字": 允许货币符号 / 千分位逗号 / 百分号 / 空白,
// 例如 "¥1,250.00" → 1250。无法解析时返回 ok=false(按字符串比)。
func dataNumber(s string) (float64, bool) {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r == '.', r == '-', r == '+', r == 'e', r == 'E':
			sb.WriteRune(r)
		case r == ',', r == ' ', r == '\u00a0', r == '\u00a5', r == '$', r == '\u20ac',
			r == '\u00a3', r == '%':
			// 货币符号 / 千分位 / 百分号忽略
		default:
			return 0, false
		}
	}
	t := sb.String()
	if t == "" || t == "-" || t == "+" || t == "." {
		return 0, false
	}
	v, err := strconv.ParseFloat(t, 64)
	return v, err == nil
}

// dataNaturalCompare 大小写不敏感的自然顺序: 数字段按数值, 其余按字节。
func dataNaturalCompare(a, b string) int {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if dataIsDigit(a[i]) && dataIsDigit(b[j]) {
			si, sj := i, j
			for i < len(a) && dataIsDigit(a[i]) {
				i++
			}
			for j < len(b) && dataIsDigit(b[j]) {
				j++
			}
			na := strings.TrimLeft(a[si:i], "0")
			nb := strings.TrimLeft(b[sj:j], "0")
			if len(na) != len(nb) {
				if len(na) < len(nb) {
					return -1
				}
				return 1
			}
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
			continue
		}
		if a[i] != b[j] {
			if a[i] < b[j] {
				return -1
			}
			return 1
		}
		i++
		j++
	}
	switch {
	case i < len(a):
		return 1
	case j < len(b):
		return -1
	}
	return 0
}

func dataIsDigit(b byte) bool { return b >= '0' && b <= '9' }

func dataHasInt(list []int, v int) bool {
	for _, n := range list {
		if n == v {
			return true
		}
	}
	return false
}
