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
	// Width 列宽(逻辑像素)。<= 0 表示占满剩余宽度 —— 只对最后一列有意义(同 TableColumn)。
	Width int32
	// Sortable 允许点击表头排序: 每次点击在 升序 → 降序 → 不排序 之间循环
	// (同 TanStack 的 enableSortingRemoval 默认语义)。
	Sortable bool
	// Compare 覆盖该列的排序比较(负数 = a 在前)。nil = 自动: 两侧都能解析成数字
	// (允许货币符号 / 千分位 / 百分号)时按数值, 否则按大小写不敏感的自然顺序
	// (字符串里的数字段按数值比较, 同 localeCompare 的 numeric: true)。
	Compare func(a, b string) int
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

	cols []DataColumn
	rows [][]string
	view []int // 过滤 + 排序后的原始行下标

	sortCol int
	sortDir DataSort

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
		cols:              cols,
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
		if d.searchOn {
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
	if !d.searchOn {
		return
	}
	d.toolbar = xui.HStack(d.Handle, dataTableToolbarH, 0)
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

func (d *DataTable) buildTable() {
	tc := make([]TableColumn, len(d.cols))
	for i, c := range d.cols {
		tc[i] = TableColumn{Title: c.Title, Width: c.Width}
	}
	d.table = NewTable(0, 0, 0, 0, d.Handle, tc)
	xui.ItemFill(d.table)
	// 表格吃掉工具栏/页脚之间的剩余高度; 高度 Auto 的父级里 Weight 会退化成 0
	// (DESIGN 8.55), cy=0 的情况由构造函数的"根初始高度"兜底(见 8.164)。
	xui.ItemFillHeight(d.table)
	xui.ItemMinSize(d.table, 0, d.minBodyH())

	d.table.onHeaderExtra = d.paintHeaderExtra
	d.table.AddEvent_List_Header_Click(func(hEle int, iItem int32, pbHandled *bool) int {
		if iItem >= 0 && int(iItem) < len(d.cols) && d.cols[iItem].Sortable {
			d.toggleSort(int(iItem))
			*pbHandled = true
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
		d.table.AddRow(d.rows[d.view[i]]...)
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

func (d *DataTable) paintHeaderExtra(p *xui.Painter, rc xc.RECT, col int32) {
	if col < 0 || int(col) >= len(d.cols) || !d.cols[col].Sortable {
		return
	}
	th := xui.Current()
	path, color := icons.IconArrowUpDown, th.MutedForeground
	if int(col) == d.sortCol {
		switch d.sortDir {
		case DataSortAsc:
			path, color = icons.IconArrowUp, th.Foreground
		case DataSortDesc:
			path, color = icons.IconArrowDown, th.Foreground
		}
	}
	font := xui.FontHandle(th.FontDefault)
	x := rc.Left + tableCellPadX + xui.TextWidth(d.cols[col].Title, font) + dataTableSortGap
	if limit := rc.Right - tableCellPadX - dataTableSortIcon; x > limit {
		x = limit
	}
	top := rc.Top + (rc.Bottom-rc.Top-dataTableSortIcon)/2
	ir := xc.RECT{Left: x, Top: top, Right: x + dataTableSortIcon, Bottom: top + dataTableSortIcon}
	p.Icon(path, &ir, color, 2)
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
