package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
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
	// dataTableSortRankW 是次排序键序号的宽度(多列排序时画在箭头右侧)。
	dataTableSortRankW int32 = 9
	// dataTableSelectW 是勾选列宽(w-8): 16 的复选框 + 单元格 p-2(8×2)。
	dataTableSelectW int32 = 32
	// dataTableCheckSize 是勾选框边长(shadcn checkbox 的 size-4)。
	dataTableCheckSize int32 = 16
	// dataTableToggleW 是工具栏"视图"按钮宽(图标 + "视图" 两字)。
	dataTableToggleW int32 = 88
	// dataTableSearchW 是工具栏关键字筛选框的固定宽(对齐 tasks 的 w-[150px] lg:w-[250px])。
	// 不能再用 Weight 吃满整行: 那样分面胶囊一变宽, 输入框就变窄、把分面按钮整个往左挤。
	dataTableSearchW int32 = 200
	// dataTableSearchMinW 是窗口很窄时的下限: 固定宽加上分面胶囊 / 重置 / 视图会超出
	// 卡片宽度, 视图会被挤出右缘。空间不足时输入框收到这里, 保证视图仍贴右可见。
	dataTableSearchMinW int32 = 100
)

// DataSort 排序方向。
type DataSort int

const (
	DataSortNone DataSort = iota // 不排序(保持原始顺序)
	DataSortAsc                  // 升序
	DataSortDesc                 // 降序
)

// DataSortKey 是一个排序键: 列下标(DataColumn 序列) + 方向。
// 多个键按优先级排列 —— 第一个是主排序, 前面的键相等时才比较后面的; 同一列只保留一个键。
type DataSortKey struct {
	Col int
	Dir DataSort
}

// DataColumn 数据表格的列定义。
type DataColumn struct {
	Title string
	// Width 列宽(逻辑像素)。<= 0 表示占满剩余宽度(同 TableColumn); 多个 <= 0 时只有
	// 第一个吃掉剩余宽度, 其余按 120 固定宽排。
	Width int32
	// Sortable 允许通过列头菜单排序(升序 / 降序, 再点当前项 = 取消, 同 TanStack 的
	// enableSortingRemoval); Shift+点击表头则在 升序 → 降序 → 取消 之间循环,
	// 并把该列追加为多列排序的次键(箭头后的数字是次序)。
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
	// NoHide 为 true 时列头菜单里不提供"隐藏"(对应 tasks 的 canHide: false, 行操作列)。
	NoHide bool
	// FilterOptions 该列的分面筛选项: 非空时工具栏多出一个按钮(tasks 的
	// DataTableFacetedFilter)。列内多选是 OR、列间是 AND, 并与关键字同时生效。
	FilterOptions []DataFilterOption
	// EditOptions 非空时该列只能用这些固定值编辑: **单击**弹出选择菜单(当前值带勾),
	// 不再出现文本输入框 —— 状态 / 优先级 / 角色这类枚举列用它。空 = 普通文本编辑;
	// ReadOnly 优先(设了 ReadOnly 就只走普通点击)。
	EditOptions []string
	// EditNumeric 为 true 时该列按数值输入: 输入过程中过滤掉非数字字符(只留数字、
	// 最多一个小数点与开头的负号), 提交时再校验一次(空 / 只有符号不落盘)。
	// 金额 / 数量列用; 与 EditOptions 互斥(固定选项列优先)。
	EditNumeric bool
	// ReadOnly 禁止编辑该列(见 WithDataTableEditing; 文本列双击、固定选项列单击)。
	// 自绘列也能编辑 —— 编辑期间该格的自绘内容会让位给输入框; 行操作列这类
	// "画按钮"的列应该设成只读。
	ReadOnly bool
}

// DataFilterOption 是一个分面筛选项: 与单元格**原始文本**相等即命中。
type DataFilterOption struct {
	Label string // 菜单里显示的文字
	Value string // 比较用的值(空 = 用 Label)
	Icon  string // 可选的 lucide path, 画在条目前的图标
}

func (o DataFilterOption) value() string {
	if o.Value != "" {
		return o.Value
	}
	return o.Label
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
// 输入框定宽 200(窗口窄到放不下时收窄到 100), 不再吃掉整行剩余宽度。
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

// WithDataTablePageSize 设置初始每页行数(默认 10, 推荐 20~50)。
// 表格体只装当前页; 排序/筛选仍按全部数据在 Go 侧计算。虚拟滚动模式忽略本选项。
func WithDataTablePageSize(n int) DataTableOption {
	return func(d *DataTable) {
		if n > 0 {
			d.pageSize = n
		}
	}
}

// WithDataTableVirtualScroll 在构造时切换为虚拟滚动模式: 一次展示全部筛选结果,
// 用滚动代替翻页。
//
// 默认仍是分页。虚拟模式适合 10 万行以上的连续浏览: 原生 List 只给可见行填文本,
// 行数据不复制进原生适配器; 页脚只保留总行数 / 勾选统计, 分页控件隐藏, SetPage /
// PageCount / OnPageChange 不再生效。排序 / 筛选 / 勾选 / 编辑与分页模式语义一致。
func WithDataTableVirtualScroll() DataTableOption {
	return func(d *DataTable) { d.virtual = true }
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

// WithDataTableEditing 开启单元格编辑: 文本列**双击**在原地弹出输入框, Enter / 失焦提交,
// Esc 取消; 配了 DataColumn.EditOptions 的固定选项列则**单击**弹选择菜单, 选中即提交;
// DataColumn.EditNumeric 的数值列在输入时过滤非数字字符。值没变不写回, 排序与筛选会
// 按新值重算。
//
// 不可编辑的格子: 勾选列与 DataColumn.ReadOnly 列(自绘列可以编辑, 编辑期间该格
// 暂停自绘)。写入前回调 OnCellEdit, 返回 false 即拒绝(单元格保持旧值)。
func WithDataTableEditing() DataTableOption {
	return func(d *DataTable) { d.editEnabled = true }
}

// DataTable 数据表格: 在 Table 之上提供 **列排序 / 分面筛选 / 关键字筛选 / 分页**。
//
// 对齐 shadcn 的 Data Table 模式(TanStack Table 的 tasks 示例):
//   - 列头点击弹菜单(升序 / 降序 / 隐藏): 升序降序是勾选项, 再点当前项 = 取消排序,
//     图标依次是 arrow-up-down(muted) / arrow-up / arrow-down;
//   - 列头菜单仍是单列排序; Shift+点击可排序表头直接循环该列(升 → 降 → 取消)并
//     追加为多列排序的次键, 次键箭头后有次序数字; 可交互表头有悬停底色,
//     拖拽表头列可交换显示顺序(隐藏列的位置保留);
//   - 顶部筛选框输入即过滤(不区分大小写, 子串匹配), 过滤后回到第一页; 配了
//     DataColumn.FilterOptions 的列还会在工具栏多一个多选按钮(列内 OR、列间 AND);
//   - 关键字或分面筛选生效时工具栏出现"重置";
//   - 勾选: 普通点表头复选框 = 整页全选, Shift+点击 = 整个筛选结果跨页全选;
//   - 页脚: 左侧总行数(筛选中显示"筛选出 x / y 行"), 右侧每页行数下拉 +
//     第 x / y 页 + 首页/上一页/下一页/末页按钮, 到端点自动禁用;
//   - 排序列与筛选条件变化都会回到第一页(autoResetPageIndex 语义)。
//
// 默认是分页模式; WithDataTableVirtualScroll 切换为虚拟滚动(一次展示全部筛选结果,
// 页脚只留总行数 / 勾选统计, 分页 API 不生效)。两种模式共用上面的排序 / 筛选 /
// 勾选 / 列显隐 / 编辑逻辑, 差别只在行映射与页脚。
//
// 排序与分页都在 Go 侧完成, **不启用原生 List 的 SetSort** —— 原生的适配器排序
// 只认列数据本身的类型, 与单元格显示文本(格式化过的金额/日期)和自定义比较器
// 对不上, 而且没法做"第三次点击回到原始顺序"。
//
// 表格行高/外观沿用 Table; cy 为 0 时按"工具栏 + 每页行数(虚拟滚动为 10 行) + 页脚"自动配高。
type DataTable struct {
	*widget.LayoutEle

	cols    []DataColumn
	rows    [][]string
	view    []int // 过滤 + 排序后的原始行下标
	visible []int // 当前显示的列(d.cols 下标); 列显隐改变的只是它
	order   []int // 全部列(d.cols 下标)的显示顺序; 列拖拽只改它, 隐藏列的位置也保留

	sortKeys []DataSortKey

	// 行勾选: selected 的键是**原始数据行**下标, 翻页 / 排序 / 筛选都不影响它。
	selectOn bool
	selected map[int]bool

	toggleOn bool
	viewBtn  *Button

	// 分面筛选: facet 的键是 d.cols 下标, 值是选中的 Value 集合(列内 OR)。
	facet       map[int]map[string]bool
	facetBtns   map[int]*facetTrigger
	facetPanels map[int]*facetPanel
	resetBtn    *Button

	filter     string
	filterCols []int

	// virtual 为 true = 虚拟滚动模式: 不分页, 全部筛选结果放在同一张可滚动表里。
	// 默认 false(分页); 两种模式共用同一套排序/筛选/勾选/编辑逻辑, 差别只在行映射与页脚。
	virtual bool

	page      int // 0 起
	pageSize  int
	pageSizes []int

	searchOn          bool
	searchPlaceholder string
	searchSyncing     bool

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

	onSort     func(col int, dir DataSort)
	onSortKeys func(keys []DataSortKey)
	onPage     func(page int)
	onFilter   func(text string)
	onRow      func(row int)
	onCell     func(row, col int)
	onSel      func(rows []int)
	onFacet    func(col int, values []string)

	// 单元格编辑(见 WithDataTableEditing): 输入框在第一次双击时惰性创建,
	// 父是表格所在窗口 —— 绝对定位, 不参与页面/表格的流式布局。
	editEnabled   bool
	editing       bool
	editor        *Input
	editRaw       int   // 正在编辑的原始数据行
	editCol       int   // 正在编辑的列(DataColumn 下标)
	editRowT      int32 // 原生页内行号(编辑期间清空/恢复该格文本用)
	editSub       int32 // 原生列号(含勾选列偏移)
	editorSyncing bool  // 程序化写编辑框文本时抑制数值过滤, 防止递归/把原值洗掉
	onEdit        func(row, col int, old, next string) bool
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
	d.facet = map[int]map[string]bool{}
	d.order = make([]int, len(d.cols))
	for i := range d.order {
		d.order[i] = i
	}
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
		total := d.initialBodyH() + dataTableFooterH + dataTableGap
		if d.toolbarOn() {
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

// hasFacets 报告是否存在分面筛选列(某列配了 FilterOptions)。
func (d *DataTable) hasFacets() bool {
	for _, c := range d.cols {
		if len(c.FilterOptions) > 0 {
			return true
		}
	}
	return false
}

// toolbarOn 报告工具栏是否出现(筛选框 / 分面筛选 / 视图下拉 任一开启)。
func (d *DataTable) toolbarOn() bool {
	return d.searchOn || d.toggleOn || d.hasFacets()
}

func (d *DataTable) buildToolbar() {
	if !d.toolbarOn() {
		return
	}
	d.toolbar = xui.HStack(d.Handle, dataTableToolbarH, 0)
	// 窗口变窄时按可用宽度收窄定宽搜索框(见 dataTableSearchMinW); 工具栏自身宽度
	// 只由父级决定, 改子项宽度不会反过来触发这个事件, 不会循环。
	d.toolbar.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		if d.search == nil || d.searchSyncing {
			return 0
		}
		d.searchSyncing = true
		old := d.search.GetWidth()
		d.syncSearchWidth(xui.Current())
		if d.search.GetWidth() != old {
			xc.XEle_AdjustLayout(d.toolbar.Handle)
			d.toolbar.Redraw(false)
		}
		d.searchSyncing = false
		return 0
	})
	// 关键字筛选框固定宽放最左(tasks 的 DataTableToolbar): 它定宽, 后面的分面触发区
	// 左缘才不会随勾选变化 —— 触发区自己只向右长, 弹层锚点也就不动。输入框若像原来那样
	// 用 Weight 吃满剩余宽, 分面胶囊一变宽它就会变窄, 把触发区整个推向左。
	if d.searchOn {
		d.search = NewInput(0, 0, 0, dataTableToolbarH, d.toolbar.Handle,
			WithPlaceholder(d.searchPlaceholder),
			WithLeadingIcon(icons.IconSearch),
			WithClearButton())
		xui.ItemWidth(d.search, dataTableSearchW)
		xui.ItemMargin(d.search, 0, 0, dataTableGap, 0)
		d.search.OnChange(func(text string) {
			if d.suppressSearch {
				return
			}
			d.filter = text
			d.apply(true)
			d.updateToolbar()
			if d.onFilter != nil {
				d.onFilter(text)
			}
		})
	}
	// 分面筛选触发区(tasks 的 DataTableFacetedFilter): 紧跟在定宽输入框右侧。
	// 自绘胶囊的选中项直接显示在列名右侧(Button 画不出分隔线 + 胶囊, 见 datatablefacet.go)。
	d.facetBtns = map[int]*facetTrigger{}
	d.facetPanels = map[int]*facetPanel{}
	for ci := range d.cols {
		if len(d.cols[ci].FilterOptions) == 0 {
			continue
		}
		ci := ci
		b := newFacetTrigger(0, 0, 0, facetTriggerH, d.toolbar.Handle,
			d.cols[ci].Title, icons.IconCirclePlus)
		// 只留右间距: 第一个触发区的左间隙由输入框的右外边距提供, 相邻分面之间 gap 8。
		xui.ItemMargin(b, 0, 2, dataTableGap, 2)
		d.facetBtns[ci] = b
		// 面板随触发区一次建好(内容懒构建): 点击由 Popover 自己的 toggle 处理,
		// 每次都新建 Popover 会重复挂事件, 越开越闪(见新建处的注释)。
		d.facetPanels[ci] = d.newFacetPanel(ci, b)
	}
	// 重置(tasks 的 Reset + X): 关键字或分面筛选任一生效才出现。左间隙由前一项
	// (输入框或最后一个分面触发区)的右外边距提供, 所以自身左边距为 0。
	d.resetBtn = NewButton(0, 0, 0, 0, "重置", d.toolbar.Handle,
		WithVariant(ButtonGhost), WithSize(ButtonSizeSm),
		WithButtonTrailingIcon(icons.IconX))
	xui.ItemSize(d.resetBtn, 0, 28) // 默认收起(固定宽 0 + 隐藏)
	xui.ItemMargin(d.resetBtn, 0, 2, 0, 2)
	d.resetBtn.Show(false)
	d.resetBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		d.ResetFilters()
		return 0
	})
	d.updateToolbar()
	if d.toggleOn {
		// 用 Weight 撑开左侧内容, 把"视图"钉在工具栏右缘(分面变宽 / 重置按钮出现都不移动它)。
		spacer := widget.NewElement(0, 0, 0, 0, d.toolbar.Handle)
		if spacer != nil {
			spacer.EnableBkTransparent(true)
			spacer.EnableDrawBorder(false)
			spacer.EnableFocus(false)
			spacer.EnableSwitchFocus(false)
			xui.ItemWeight(spacer, 1)
		}
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

// initVisible 初始化可见列(按 order 的显示顺序过滤掉 Hidden), 并保证至少留一列
// (全隐藏等于表格变空壳)。
func (d *DataTable) initVisible() []int {
	if len(d.order) != len(d.cols) { // 列定义被整体替换(SetColumns): 顺序重来
		d.order = make([]int, len(d.cols))
		for i := range d.order {
			d.order[i] = i
		}
	}
	out := make([]int, 0, len(d.cols))
	for _, i := range d.order {
		if i < 0 || i >= len(d.cols) || d.cols[i].Hidden {
			continue
		}
		out = append(out, i)
	}
	if len(out) == 0 {
		out = append(out, 0)
		d.cols[0].Hidden = false
	}
	return out
}

// orderedCols 按当前显示顺序返回全部列(d.cols 下标, 含隐藏列): 列拖拽之后,
// "视图"菜单也应按用户看到的顺序排。order 长度不对(尚未初始化)时退回定义顺序。
func (d *DataTable) orderedCols() []int {
	if len(d.order) == len(d.cols) {
		return d.order
	}
	out := make([]int, len(d.cols))
	for i := range out {
		out[i] = i
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
	var opts []TableOption
	if d.virtual {
		opts = append(opts, withTableVirtual())
	}
	d.table = NewTable(0, 0, 0, 0, d.Handle, d.tableColumns(), opts...)
	xui.ItemFill(d.table)
	// 表格吃掉工具栏/页脚之间的剩余高度; 高度 Auto 的父级里 Weight 会退化成 0
	// (DESIGN 8.55), cy=0 的情况由构造函数的"根初始高度"兜底(见 8.164)。
	xui.ItemFillHeight(d.table)
	xui.ItemMinSize(d.table, 0, d.minBodyH())

	d.table.onHeaderExtra = d.paintHeaderExtra
	d.table.onCellExtra = d.paintCell
	// 只有能排序 / 能隐藏的列表头才画悬停底色、才可拖拽换位(操作列 NoHide 不动)。
	d.table.onHeaderActive = func(tcol int32) bool {
		ci := d.colAt(tcol)
		return ci >= 0 && (d.cols[ci].Sortable || d.canHide(ci))
	}
	d.table.onColumnsMoved = d.moveColumn
	d.table.AddEvent_List_Header_Click(func(hEle int, iItem int32, pbHandled *bool) int {
		// 拖拽换列后的那次松开可能还会补发点击: 吞掉, 别弹出列头菜单。
		if d.table.dragSwallowClick {
			d.table.dragSwallowClick = false
			*pbHandled = true
			return 0
		}
		if d.selectOn && iItem == 0 {
			if dataShiftDown() {
				d.toggleFilteredSelection() // 跨页全选(只针对当前筛选结果)
			} else {
				d.togglePageSelection()
			}
			*pbHandled = true
			return 0
		}
		ci := d.colAt(iItem)
		if ci < 0 || (!d.cols[ci].Sortable && !d.canHide(ci)) {
			return 0 // 既不能排序也不能隐藏: 表头当普通文字用
		}
		// Shift+点击可排序表头 = 直接多列排序(升 → 降 → 取消), 不弹菜单;
		// 普通点击仍走列头菜单(升序 / 降序 / 隐藏)。
		if dataShiftDown() && d.cols[ci].Sortable {
			d.cycleSortKey(ci)
			*pbHandled = true
			return 0
		}
		// 菜单按**被点的列**定位(showHeaderMenu 内部用绘制缓存里的表头矩形做锚点),
		// 不用 PopupAtCursor —— 否则按住鼠标拖到别处再松开, 菜单会跟着跑到松开的位置。
		// 不能改到 LButtonDown 里弹: 菜单一弹出就被紧随其后的松开消息关掉(实测)。
		d.showHeaderMenu(ci, iItem)
		*pbHandled = true
		return 0
	})

	// 单击命中: 勾选列切勾选; 固定选项列弹选择菜单; 其余走行/单元格点击。
	// 原生 List 的单选高亮只表示"刚点过", 勾选模式与固定选项列都要吞掉按下消息,
	// 不让它改选中行(见 DESIGN 8.170)。
	d.table.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if pPt == nil {
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
		if d.selectOn && sub == 0 { // 勾选列
			*pbHandled = true
			d.toggleRowSelected(raw)
			return 0
		}
		ci := d.colAt(sub)
		// 固定选项列: 单击弹菜单(等左键松开后再弹, 避开这次点击的松开), 不再触发
		// 行/单元格点击 —— 这一下就是"打开下拉"。
		if d.editEnabled && ci >= 0 && !d.cols[ci].ReadOnly && len(d.cols[ci].EditOptions) > 0 {
			d.commitCellEdit(true) // 上一格文本编辑先落盘
			if rc, ok := d.cellRectFor(row, sub); ok {
				*pbHandled = true
				d.openEditOptionsMenuAfterRelease(raw, ci, d.cols[ci].EditOptions, rc)
			}
			return 0
		}
		if !d.selectOn {
			return 0
		}
		*pbHandled = true
		if ci >= 0 && d.onCell != nil {
			d.onCell(raw, ci)
		}
		if d.onRow != nil {
			d.onRow(raw)
		}
		return 0
	})

	// 双击勾选框 = 两次点击: List 把第二次按下并进 LButtonDBClick(不再单独发
	// LButtonDown), 这里补一次切换并吞掉消息, 否则只响应第一次(同 Tree 勾选框)。
	d.table.AddEvent_LButtonDBClick(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if !d.selectOn || pPt == nil {
			return 0
		}
		var row, sub int32
		if !d.table.HitTestOffset(pPt, &row, &sub) || sub != 0 {
			return 0
		}
		raw := d.rawRow(row)
		if raw < 0 {
			return 0
		}
		*pbHandled = true // 别让这次双击落到单元格编辑 / 原生默认双击
		d.toggleRowSelected(raw)
		return 0
	})

	// 空状态 / 排序图标都画在表格元素上: List 的内容区由原生管理, 往里塞覆盖元素
	// 会被原生布局摆到奇怪的位置(见 DESIGN 8.162)。
	d.table.EnableEvent_XE_PAINT_END()
	d.table.AddEvent_Paint_End(d.paintEmpty)

	d.table.OnSelect(func(row int32) {
		raw := d.rawRow(row)
		if raw < 0 || d.onRow == nil {
			return
		}
		d.onRow(raw)
	})

	if d.editEnabled {
		// 文本列双击才进编辑: 单击仍走 OnRowClick / OnCellClick(与 shadcn 的"点行选行"
		// 不冲突); 固定选项列的单击由 LButtonDown 接管为选择菜单。
		d.table.AddEvent_LButtonDBClick(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			if d.beginCellEdit(pPt) {
				*pbHandled = true
			}
			return 0
		})
		// 滚动(滚轮 / 拖滚动条 / 键盘)会把行带走, 而编辑框是绝对定位的窗口子元素
		// (不跟滚动): 先落盘再滚。挂 ScrollView 事件而不是只挂 MouseWheel。
		d.table.AddEvent_ScrollView_Scroll_V(func(hEle int, pos int32, pbHandled *bool) int {
			d.commitCellEdit(true)
			return 0
		})
		// 编辑器挂在窗口上, 只销毁表格不会带走它: 表格先销毁时要自己收尾。
		d.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
			d.releaseEditor()
			return 0
		})
	}
}

func (d *DataTable) buildFooter() {
	f := xui.HStack(d.Handle, dataTableFooterH, 8)
	d.footer = f

	d.totalLabel = NewLabel(0, 0, 0, 20, "", f.Handle).SetMuted()
	xui.ItemWeight(d.totalLabel, 1)
	if d.virtual {
		// 虚拟滚动没有页: 页脚只留总行数 / 勾选统计。
		return
	}

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

// 虚拟滚动没有"每页行数": 布局最小高度只保证表头 + 3 行, cy=0 的初始高度
// 仍按 10 行配(和分页默认观感一致), 但不把 pageSize 当成最小高度 —— 否则
// 高度小于 pageSize 行的容器会把页脚挤出可视区。
const (
	dataTableVirtualMinRows     int32 = 3
	dataTableVirtualInitialRows int32 = 10
)

func dataTableBodyH(rows int32) int32 {
	return tableHeaderH + tableInset*2 + rows*tableRowH
}

// minBodyH 是表格体的布局最小高度(分页 = 每页行数, 虚拟滚动 = 3 行)。
func (d *DataTable) minBodyH() int32 {
	if d.virtual {
		return dataTableBodyH(dataTableVirtualMinRows)
	}
	return dataTableBodyH(int32(d.pageSize))
}

// initialBodyH 是 cy=0 时的初始表格体高度。
func (d *DataTable) initialBodyH() int32 {
	if d.virtual {
		return dataTableBodyH(dataTableVirtualInitialRows)
	}
	return d.minBodyH()
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
	d.updateToolbar() // "重置"按钮按筛选状态出现/收起(关键字筛选也要刷新)
	if d.onFilter != nil {
		d.onFilter(s)
	}
	return d
}

// Filter 返回当前关键字。
func (d *DataTable) Filter() string { return d.filter }

// SetSort 程序化设置排序列(下标; -1 = 不排序)与方向。等价于只传一个键的 SetSortKeys。
func (d *DataTable) SetSort(col int, dir DataSort) *DataTable {
	return d.SetSortKeys(DataSortKey{Col: col, Dir: dir})
}

// SetSortKeys 设置多列排序键(按优先级从主到次)。非法列 / DataSortNone / 重复列会被忽略;
// 不传 = 取消排序。点击表头列头菜单仍是单列排序, 多列排序的交互是 Shift+点击表头。
func (d *DataTable) SetSortKeys(keys ...DataSortKey) *DataTable {
	out := make([]DataSortKey, 0, len(keys))
	seen := make(map[int]bool, len(keys))
	for _, k := range keys {
		if k.Col < 0 || k.Col >= len(d.cols) || k.Dir == DataSortNone || seen[k.Col] {
			continue
		}
		seen[k.Col] = true
		out = append(out, k)
	}
	d.sortKeys = out
	d.apply(true)
	d.notifySort()
	return d
}

// Sort 返回主排序列(未排序为 -1)与方向。
func (d *DataTable) Sort() (int, DataSort) {
	if len(d.sortKeys) == 0 {
		return -1, DataSortNone
	}
	return d.sortKeys[0].Col, d.sortKeys[0].Dir
}

// SortKeys 返回当前全部排序键(主 → 次)。
func (d *DataTable) SortKeys() []DataSortKey {
	return append([]DataSortKey(nil), d.sortKeys...)
}

// OnSortKeysChange 注册排序键变化回调(keys 是主 → 次的快照, 空 = 不排序)。
// 与 OnSortChange 同时注册时两个都会触发; OnSortChange 只给主键, 兼容单列用法。
func (d *DataTable) OnSortKeysChange(fn func(keys []DataSortKey)) *DataTable {
	d.onSortKeys = fn
	return d
}

func (d *DataTable) notifySort() {
	col, dir := -1, DataSortNone
	if len(d.sortKeys) > 0 {
		col, dir = d.sortKeys[0].Col, d.sortKeys[0].Dir
	}
	if d.onSort != nil {
		d.onSort(col, dir)
	}
	if d.onSortKeys != nil {
		d.onSortKeys(d.SortKeys())
	}
}

// sortKeyIndex 返回该列的排序键次序(0 = 主键), -1 = 不在排序键里。
func (d *DataTable) sortKeyIndex(col int) int {
	for i, k := range d.sortKeys {
		if k.Col == col {
			return i
		}
	}
	return -1
}

func (d *DataTable) hasSortKey(col int, dir DataSort) bool {
	i := d.sortKeyIndex(col)
	return i >= 0 && d.sortKeys[i].Dir == dir
}

// cycleSortKey 是 Shift+点击可排序表头的行为: 未排序 → 升序(追加为次键),
// 升序 → 降序, 降序 → 移除该键。
func (d *DataTable) cycleSortKey(col int) {
	i := d.sortKeyIndex(col)
	switch {
	case i < 0:
		d.sortKeys = append(d.sortKeys, DataSortKey{Col: col, Dir: DataSortAsc})
	case d.sortKeys[i].Dir == DataSortAsc:
		d.sortKeys[i].Dir = DataSortDesc
	default:
		d.sortKeys = append(d.sortKeys[:i], d.sortKeys[i+1:]...)
	}
	d.apply(true)
	d.notifySort()
}

// dataShiftDown 报告 Shift 是否按下(高位置位 = 当前按住; GetAsyncKeyState 返回 int16,
// 所以用 <0 判断, 不用和 0x8000 做按位与)。
func dataShiftDown() bool {
	return wapi.GetAsyncKeyState(xcc.VK_Shift) < 0
}

// Virtual 报告当前是否为虚拟滚动模式。
func (d *DataTable) Virtual() bool { return d.virtual }

// Page 返回当前页(0 起)。虚拟滚动模式恒为 0。
func (d *DataTable) Page() int { return d.page }

// PageCount 返回总页数(至少 1)。虚拟滚动模式恒为 1(没有页的概念)。
func (d *DataTable) PageCount() int {
	if d.virtual {
		return 1
	}
	return dataPageCount(len(d.view), d.pageSize)
}

// SetPage 跳到指定页(越界自动夹取), 触发 OnPageChange。虚拟滚动模式下无操作。
func (d *DataTable) SetPage(p int) *DataTable {
	if d.virtual {
		return d
	}
	d.page = p
	d.clampPage()
	d.render()
	if d.onPage != nil {
		d.onPage(d.page)
	}
	return d
}

// PageSize 返回每页行数。虚拟滚动模式下该值不生效(仅保留配置, 初始高度按 10 行配)。
func (d *DataTable) PageSize() int { return d.pageSize }

// SetPageSize 修改每页行数, 回到第一页。虚拟滚动模式下无操作。
func (d *DataTable) SetPageSize(n int) *DataTable {
	if d.virtual || n <= 0 || n == d.pageSize {
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

// SelectAllFiltered 勾选当前筛选结果中的全部行(跨页; 已被筛掉、但之前勾过的行保持勾选)。
// 对应表头的 Shift+点击全选: 普通点击只切当前页(TanStack 的 page rows 口径)。
func (d *DataTable) SelectAllFiltered() *DataTable {
	if d.selected == nil {
		d.selected = map[int]bool{}
	}
	for _, raw := range d.view {
		d.selected[raw] = true
	}
	d.afterSelectionChange(true)
	return d
}

// AllFilteredSelected 报告当前筛选结果是否已全部勾选(结果为空时返回 false)。
func (d *DataTable) AllFilteredSelected() bool {
	return len(d.view) > 0 && d.selectedInView() >= len(d.view)
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

// moveColumn 把拖拽的表头列插到目标边界(Table 列下标, 含勾选列偏移)。
// 只改 order(全部列的显示顺序): 隐藏列的相对位置也保留, 列显隐 / 排序 / 分面的
// d.cols 下标语义完全不变。
func (d *DataTable) moveColumn(from, to int32) {
	off := 0
	if d.selectOn {
		off = 1
	}
	f, toVis := int(from)-off, int(to)-off
	if len(d.visible) <= 1 || f < 0 || f >= len(d.visible) {
		return
	}
	moved := d.visible[f]
	rem := make([]int, 0, len(d.visible)-1)
	rem = append(rem, d.visible[:f]...)
	rem = append(rem, d.visible[f+1:]...)
	if toVis > f {
		toVis-- // 源列移除后, 目标边界左移一格
	}
	if toVis < 0 {
		toVis = 0
	}
	if toVis > len(rem) {
		toVis = len(rem)
	}
	if toVis == f {
		return // 落在原位(在本列左/右边界松开都算原位)
	}
	pos := -1
	for i, ci := range d.order {
		if ci == moved {
			pos = i
			break
		}
	}
	if pos < 0 {
		return
	}
	d.order = append(d.order[:pos], d.order[pos+1:]...)
	at := len(d.order)
	if toVis < len(rem) {
		for i, ci := range d.order {
			if ci == rem[toVis] {
				at = i
				break
			}
		}
	} else if len(rem) > 0 {
		for i, ci := range d.order {
			if ci == rem[len(rem)-1] {
				at = i + 1
				break
			}
		}
	}
	order := make([]int, 0, len(d.order)+1)
	order = append(order, d.order[:at]...)
	order = append(order, moved)
	order = append(order, d.order[at:]...)
	d.order = order
	d.visible = d.initVisible()
	d.rebuildColumns()
}

// SetColumns 替换列定义(列名 / 宽度 / 排序 / 渲染函数全部重来), 勾选状态保留。
func (d *DataTable) SetColumns(cols []DataColumn) *DataTable {
	if len(cols) == 0 {
		return d
	}
	d.cols = append([]DataColumn(nil), cols...)
	d.order = nil // 新列定义 = 默认显示顺序
	d.visible = d.initVisible()
	kept := make([]DataSortKey, 0, len(d.sortKeys))
	for _, k := range d.sortKeys {
		if k.Col >= 0 && k.Col < len(d.cols) {
			kept = append(kept, k)
		}
	}
	d.sortKeys = kept
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
	for _, i := range d.orderedCols() {
		i := i
		m.Check(d.cols[i].Title, !d.cols[i].Hidden, MenuOnCheck(func(on bool) {
			d.SetColumnVisible(i, on)
		}))
	}
	m.PopupAtElement(hEle)
	// 原生菜单是模态的: 打开期间的失效区域会被它关闭时的"还原屏幕"盖回去, 所以关闭后
	// 必须再立即重绘一次, 否则工具栏/表格停在菜单打开前的像素(表现为残留的半截边框)。
	d.updateToolbar()
	d.Redraw(true)
}

// ===== 分面筛选 =====

// showFacetMenu 弹出某列的分面筛选菜单(Menu.Check, 列内多选)。
// setFacet 勾选/取消一个分面值(value 是该列的单元格原始文本)。
func (d *DataTable) setFacet(col int, value string, on bool) {
	if d.facet == nil {
		d.facet = map[int]map[string]bool{}
	}
	set := d.facet[col]
	if on {
		if set == nil {
			set = map[string]bool{}
			d.facet[col] = set
		}
		set[value] = true
	} else if set != nil {
		delete(set, value)
		if len(set) == 0 {
			delete(d.facet, col)
		}
	}
	d.afterFacetChange(col)
}

// clearFacet 清掉某列的全部分面选择。
func (d *DataTable) clearFacet(col int) {
	if _, ok := d.facet[col]; !ok {
		return
	}
	delete(d.facet, col)
	d.afterFacetChange(col)
}

func (d *DataTable) afterFacetChange(col int) {
	d.apply(true)
	d.updateToolbar()
	if d.onFacet != nil {
		d.onFacet(col, d.FacetValues(col))
	}
}

func (d *DataTable) facetHas(col int, value string) bool {
	return d.facet[col][value]
}

// facetActive 报告是否有任一分面筛选生效。
func (d *DataTable) facetActive() bool {
	for _, set := range d.facet {
		if len(set) > 0 {
			return true
		}
	}
	return false
}

// filtersActive 报告关键字或分面筛选是否有生效的(决定页脚文案与"重置"是否出现)。
func (d *DataTable) filtersActive() bool {
	return strings.TrimSpace(d.filter) != "" || d.facetActive()
}

// updateToolbar 按筛选状态刷新工具栏: 分面按钮显示"列名 + 已选数", "重置"按需出现。
func (d *DataTable) updateToolbar() {
	if d.toolbar == nil {
		return
	}
	th := xui.Current()
	for ci, b := range d.facetBtns {
		// 选中项按选项顺序排(不是点击顺序), 与 shadcn 的 facets 渲染一致
		b.SetValues(d.facetLabels(ci))
		w := b.naturalWidth(th)
		b.SetWidth(w)
		xui.ItemWidth(b, w)
		b.Redraw(false)
	}
	if d.resetBtn != nil {
		on := d.filtersActive()
		if on {
			w := d.resetBtn.naturalWidth(th)
			d.resetBtn.Show(true)
			d.resetBtn.SetWidth(w)
			xui.ItemWidth(d.resetBtn, w)
			xui.ItemMargin(d.resetBtn, 0, 2, 0, 2)
		} else {
			d.resetBtn.Show(false)
			xui.ItemWidth(d.resetBtn, 0)
			xui.ItemMargin(d.resetBtn, 0, 2, 0, 2)
		}
	}
	d.syncSearchWidth(th)
	xc.XEle_AdjustLayout(d.toolbar.Handle)
	// AdjustLayout 只重排, 不擦旧像素: 分面按钮文字变宽(状态 → 状态 1)、重置钮出现/消失,
	// 或窄窗下搜索框跟着收窄时, 旧位置的边框都会残留(实测是半截横线)。这里整块重绘一次
	// (工具栏 + 组件根, 根负责把卡片底色铺回去)。
	d.toolbar.Redraw(false)
	d.Redraw(false)
}

// toolbarUsedWidth 汇总工具栏里除搜索框与 Weight 撑开项之外的固定宽(含外边距),
// 用于按可用宽度给搜索框定宽(见 syncSearchWidth)。
func (d *DataTable) toolbarUsedWidth(th *xui.Theme) int32 {
	used := dataTableGap // 输入框的右外边距(它自己贴工具栏左缘, 左边距为 0)
	for ci := range d.facetBtns {
		used += d.facetBtns[ci].naturalWidth(th) + dataTableGap
	}
	if d.resetBtn != nil && d.resetBtn.IsShow() {
		used += d.resetBtn.naturalWidth(th)
	}
	if d.viewBtn != nil {
		used += dataTableToggleW + dataTableGap
	}
	return used
}

// syncSearchWidth 把搜索框宽定为 dataTableSearchW; 工具栏窄到放不下时收到
// dataTableSearchMinW, 让"视图"仍然贴在右缘。正常窗口下宽恒定, 分面胶囊变宽只
// 会向右长(触发区在最左, 不再被输入框反挤)。
func (d *DataTable) syncSearchWidth(th *xui.Theme) {
	if d.search == nil {
		return
	}
	w := dataTableSearchW
	if tw := d.toolbar.GetWidth(); tw > 0 {
		if avail := tw - d.toolbarUsedWidth(th); avail < w {
			w = avail
		}
	}
	if w < dataTableSearchMinW {
		w = dataTableSearchMinW
	}
	xui.ItemWidth(d.search, w)
	if d.search.GetWidth() != w {
		d.search.SetWidth(w)
	}
}

// ResetFilters 清空关键字与全部分面筛选(工具栏"重置"按钮的行为)。
func (d *DataTable) ResetFilters() *DataTable {
	d.facet = map[int]map[string]bool{}
	hadFilter := d.filter != ""
	d.filter = ""
	if d.search != nil && d.search.Value() != "" {
		d.suppressSearch = true
		d.search.SetValue("")
		d.suppressSearch = false
		d.search.Redraw(false) // 清除钮的命中区是 paint 时重建的(见 DESIGN 8.163)
	}
	d.apply(true)
	d.updateToolbar()
	if hadFilter && d.onFilter != nil {
		d.onFilter("")
	}
	if d.onFacet != nil {
		d.onFacet(-1, nil)
	}
	return d
}

// FacetValues 返回某列当前选中的分面值(升序)。
func (d *DataTable) FacetValues(col int) []string {
	set := d.facet[col]
	out := make([]string, 0, len(set))
	for v, on := range set {
		if on {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// SetFacet 用给定值覆盖某列的分面筛选(空列表 = 清除), 回到第一页。
func (d *DataTable) SetFacet(col int, values ...string) *DataTable {
	if col < 0 || col >= len(d.cols) {
		return d
	}
	if len(values) == 0 {
		d.clearFacet(col)
		return d
	}
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true // 空串也是合法值: 用来筛"这一列没填"的行
	}
	if d.facet == nil {
		d.facet = map[int]map[string]bool{}
	}
	d.facet[col] = set
	d.afterFacetChange(col)
	return d
}

// OnFacetChange 注册分面筛选变化回调(col = -1 表示整体重置, values 是升序的值快照)。
func (d *DataTable) OnFacetChange(fn func(col int, values []string)) *DataTable {
	d.onFacet = fn
	return d
}

// ===== 内部: 勾选状态 =====

// pageSelection 统计当前页的(已勾选数, 行数) —— 表头全选的判断口径(TanStack 的
// getIsAllPageRowsSelected: 全选只管当前页, 不是全部筛选结果)。
// 虚拟滚动没有页,"当前页"就是全部筛选结果。
func (d *DataTable) pageSelection() (sel, total int) {
	if d.virtual {
		return d.selectedInView(), len(d.view)
	}
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
// 虚拟滚动没有页, 等价于 Shift+点击(全部筛选结果)。
func (d *DataTable) togglePageSelection() {
	if d.virtual {
		d.toggleFilteredSelection()
		return
	}
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

// toggleFilteredSelection 是表头勾选框的 Shift+点击: 跨页勾满 / 清空当前筛选结果。
func (d *DataTable) toggleFilteredSelection() {
	if len(d.view) == 0 {
		return
	}
	on := !d.AllFilteredSelected()
	if d.selected == nil {
		d.selected = map[int]bool{}
	}
	for _, raw := range d.view {
		if on {
			d.selected[raw] = true
		} else {
			delete(d.selected, raw)
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

// compareFor 返回某列的比较函数: 优先用 DataColumn.Compare, 否则走 dataCompare。
func (d *DataTable) compareFor(col int, a, b string) int {
	if col >= 0 && col < len(d.cols) && d.cols[col].Compare != nil {
		return d.cols[col].Compare(a, b)
	}
	return dataCompare(a, b)
}

func (d *DataTable) apply(resetPage bool) {
	d.view = dataTableViewKeys(d.rows, d.filter, d.filterCols, d.facet, d.sortKeys, d.compareFor)
	if d.virtual {
		d.page = 0
		d.render()
		if resetPage {
			d.table.ScrollPosV(0) // 没有页: autoResetPageIndex 的等价语义是回到顶部
		}
		return
	}
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

// rawRow 把表格行下标翻成原始数据行下标; 越界返回 -1。
// 分页模式: row 是当前页内下标; 虚拟滚动模式: row 就是筛选结果下标。
func (d *DataTable) rawRow(row int32) int {
	i := int(row)
	if !d.virtual {
		i += d.page * d.pageSize
	}
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

// virtualCellText 是 Table 虚表的可见行文本 provider。tcol 是**表格列**下标
// (勾选列在前), 需要先翻回 DataColumn; 自绘列与勾选列写空串, 编辑中的格子也写空串
// 让输入框底下不叠字。
func (d *DataTable) virtualCellText(row, tcol int32) string {
	raw := d.rawRow(row)
	if raw < 0 {
		return ""
	}
	if d.editing && raw == d.editRaw && tcol == d.editSub {
		return ""
	}
	ci := d.colAt(tcol)
	if ci < 0 || d.cols[ci].Render != nil {
		return ""
	}
	return d.cellText(raw, ci)
}

// ===== 单元格编辑 =====

// OnCellEdit 注册单元格编辑回调(需要 WithDataTableEditing)。回调在写回数据**之前**触发:
// 返回 false 表示拒绝本次修改, 单元格保持旧值。row 是原始数据行下标, col 是列下标
// (DataColumn 序列, 含隐藏列)。
func (d *DataTable) OnCellEdit(fn func(row, col int, old, next string) bool) *DataTable {
	d.onEdit = fn
	return d
}

// Editing 返回当前正在编辑的单元格(原始数据行 + 列下标); ok=false 表示当前没有编辑。
func (d *DataTable) Editing() (row, col int, ok bool) {
	if !d.editing {
		return 0, 0, false
	}
	return d.editRaw, d.editCol, true
}

// rowTopInView 返回第 row 行在 List 客户区里的顶边(逻辑像素)。top 是滚动为 0 时
// 第一行顶边(表头高 + 原生内缩); 滚动条可以停在非整行位置, 必须减真实滚动偏移。
func rowTopInView(t *Table, row, top, rh int32) int32 {
	return top + row*rh - t.GetViewPosV()
}

// cellRectFor 返回 row / sub 的单元格矩形(相对 List 客户区)。行顶按真实滚动偏移算;
// 首/末行被视口裁掉一部分时先滚到整行完整可见。表头矩形还没绘制过时 ok=false。
func (d *DataTable) cellRectFor(row, sub int32) (xc.RECT, bool) {
	if d.table == nil {
		return xc.RECT{}, false
	}
	hdr, ok := d.table.HeaderCellRect(sub)
	if !ok {
		return xc.RECT{}, false
	}
	top := xc.XList_GetHeaderHeight(d.table.Handle) + tableInset
	rh := d.table.RowHeight()
	if rh <= 0 {
		return xc.RECT{}, false
	}
	cellTop := rowTopInView(d.table, row, top, rh)
	if cellTop < top || cellTop+rh > d.table.GetHeight() {
		d.table.VisibleRow(row)
		cellTop = rowTopInView(d.table, row, top, rh)
	}
	rc := xc.RECT{
		Left:   hdr.Left + tableInset,
		Top:    cellTop + 1,
		Right:  hdr.Right - tableInset,
		Bottom: cellTop + rh - 1,
	}
	if rc.Right-rc.Left < 24 || rc.Bottom-rc.Top < 16 {
		return xc.RECT{}, false
	}
	return rc, true
}

// beginCellEdit 在 pt 命中的单元格上弹出编辑框。pt 是表格元素客户区坐标(逻辑像素)。
func (d *DataTable) beginCellEdit(pt *xc.POINT) bool {
	if !d.editEnabled || d.table == nil || pt == nil {
		return false
	}
	var row, sub int32
	if !d.table.HitTestOffset(pt, &row, &sub) {
		return false
	}
	if d.selectOn && sub == 0 { // 勾选列
		return false
	}
	ci := d.colAt(sub)
	if ci < 0 || d.cols[ci].ReadOnly {
		return false
	}
	raw := d.rawRow(row)
	if raw < 0 {
		return false
	}
	if len(d.cols[ci].EditOptions) > 0 {
		// 固定选项列由单击触发选择菜单; 双击不要再开文本编辑。
		return false
	}
	cell, ok := d.cellRectFor(row, sub)
	if !ok {
		return false
	}

	// 双击另一格: 先把上一处落盘再开新的。
	d.commitCellEdit(true)

	ed := d.ensureEditor()
	if ed == nil {
		return false
	}
	var or xc.RECT
	xc.XEle_GetWndClientRect(d.table.Handle, &or)
	cell.Left += or.Left
	cell.Right += or.Left
	cell.Top += or.Top
	cell.Bottom += or.Top
	ed.SetRect(&cell, true, xcc.AdjustLayout_No)

	d.editing = true
	d.editRaw, d.editCol = raw, ci
	d.editRowT, d.editSub = row, sub
	old := d.cellText(raw, ci)
	// 原生文本会从编辑框四周/下层透出来: 编辑期间先清掉这一格(提交时按原值恢复,
	// 值真变了的话随后 render 会重建)。虚表没有适配器可写, 让 provider 返回空串后
	// 重刷可见行; 分页模式仍走原生 SetCell。
	if d.virtual {
		d.table.refreshVirtual()
	} else {
		d.table.SetCell(row, sub, "")
		d.table.Redraw(false)
	}
	d.editorSyncing = true
	ed.SetValue(old) // 原值照原样显示, 不在这里做数值过滤(避免把带格式的旧值洗掉)
	d.editorSyncing = false
	ed.SelectAll()
	ed.Show(true)
	ed.SetFocus()
	return true
}

// commitCellEdit 结束当前编辑。accept=false 或值未变化时不写回数据。
func (d *DataTable) commitCellEdit(accept bool) {
	if !d.editing || d.editor == nil {
		return
	}
	d.editing = false
	next := d.editor.Value()
	raw, ci := d.editRaw, d.editCol
	old := d.cellText(raw, ci)

	d.editor.Show(false)
	// 被清空的那格交给 render 重建 —— 自绘列平时原生文本就是空的(render 时写 ""),
	// 手工把数据值塞回去会跟自绘内容叠在一起。
	d.render()

	if !accept || next == old {
		return
	}
	if ci >= 0 && ci < len(d.cols) && d.cols[ci].EditNumeric && !dataNumericOK(next) {
		return
	}
	if d.onEdit != nil && !d.onEdit(raw, ci, old, next) {
		return
	}
	d.setCellValue(raw, ci, next)
}

// dataNumericInput 过滤数值输入: 只留数字、最多一个小数点与开头的负号。
// 用户输入 "abc12.3.4x" 得到 "12.34"; 负号只有首字符有效。
func dataNumericInput(s string) string {
	out := make([]rune, 0, len(s))
	dot := false
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
			out = append(out, r)
		case r == '-' && i == 0:
			out = append(out, r)
		case r == '.' && !dot:
			dot = true
			out = append(out, r)
		}
	}
	return string(out)
}

// dataNumericOK 校验提交值: 至少一位数字, 只允许数字 / 一个小数点 / 开头负号。
func dataNumericOK(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	dot, digit := false, false
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digit = true
		case r == '-' && i == 0:
		case r == '.' && !dot:
			dot = true
		default:
			return false
		}
	}
	return digit
}

// openEditOptionsMenuAfterRelease 等鼠标左键松开后再弹固定选项菜单。
// 单击/双击的按下都会紧跟一个松开消息; 菜单若立即弹出且向上翻到光标下, 这条松开会
// 直接选中第一项。轮询 16ms 等键抬起, 抬起后再弹就不会被同一次点击"点掉"。
func (d *DataTable) openEditOptionsMenuAfterRelease(raw, col int, options []string, rc xc.RECT) {
	if d.table == nil {
		return
	}
	if wapi.GetAsyncKeyState(0x01) < 0 { // VK_LBUTTON: 高位置位 = 仍按住
		app.AfterFunc(16*time.Millisecond, func() {
			d.openEditOptionsMenuAfterRelease(raw, col, options, rc)
		})
		return
	}
	// 再等一小段: 物理键已抬起时, WM_LBUTTONUP 可能还在消息队列里, 让这条松开
	// 先被表格处理完, 菜单才不会被它点到。
	app.AfterFunc(10*time.Millisecond, func() {
		d.showEditOptionsMenu(raw, col, options, rc)
	})
}

// showEditOptionsMenu 为固定选项列弹出选择菜单(当前值带勾), 选中即落盘。
// 菜单锚定在单元格下方, 不跟鼠标走(同列头菜单, 见 DESIGN 174)。
func (d *DataTable) showEditOptionsMenu(raw, col int, options []string, rc xc.RECT) {
	if d.table == nil || !xc.XC_IsHELE(d.table.Handle) {
		return
	}
	current := d.cellText(raw, col)
	m := NewMenu()
	m.Width(160)
	for _, opt := range options {
		opt := opt
		m.Check(opt, opt == current, MenuOnCheck(func(on bool) {
			if on {
				d.applyFixedCellEdit(raw, col, opt)
			}
		}))
	}
	m.PopupAtElementRect(d.table.Handle, rc)
}

// applyFixedCellEdit 提交固定选项的编辑结果(走与文本编辑相同的 OnCellEdit 门控)。
func (d *DataTable) applyFixedCellEdit(raw, col int, next string) {
	old := d.cellText(raw, col)
	if next == old {
		return
	}
	if d.onEdit != nil && !d.onEdit(raw, col, old, next) {
		return
	}
	d.setCellValue(raw, col, next)
}

// setCellValue 写回一个单元格并重算视图(排序 / 筛选 / 当前页都会跟着更新)。
func (d *DataTable) setCellValue(raw, col int, v string) {
	if raw < 0 || raw >= len(d.rows) || col < 0 {
		return
	}
	row := d.rows[raw]
	for len(row) <= col {
		row = append(row, "")
	}
	row[col] = v
	d.rows[raw] = row
	d.apply(false)
}

// ensureEditor 惰性创建内联编辑框(父是表格所在窗口: 绝对定位, 不参与任何布局)。
func (d *DataTable) ensureEditor() *Input {
	if d.editor != nil {
		return d.editor
	}
	hwnd := xc.XWidget_GetHWINDOW(d.Handle)
	if hwnd <= 0 {
		return nil
	}
	ed := NewInput(0, 0, 120, 30, hwnd)
	if ed == nil {
		return nil
	}
	ed.Show(false)
	// 数值列输入过滤: 非数字字符直接丢弃; 程序化写值(加载原值 / 回写过滤结果)
	// 用 editorSyncing 挡掉, 避免递归, 也不把带格式的旧值洗掉。
	ed.OnChange(func(text string) {
		if d.editorSyncing || !d.editing {
			return
		}
		if d.editCol < 0 || d.editCol >= len(d.cols) || !d.cols[d.editCol].EditNumeric {
			return
		}
		clean := dataNumericInput(text)
		if clean == text {
			return
		}
		d.editorSyncing = true
		ed.SetValue(clean)
		ed.MoveEnd()
		d.editorSyncing = false
	})
	ed.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
		const (
			vkReturn = 0x0D
			vkEscape = 0x1B
		)
		switch wParam {
		case vkReturn:
			d.commitCellEdit(true)
			*pbHandled = true
		case vkEscape:
			d.commitCellEdit(false)
			*pbHandled = true
		}
		return 0
	})
	ed.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		d.commitCellEdit(true) // 点到别处(表格 / 工具栏 / 窗口外)即落盘
		return 0
	})
	d.editor = ed
	return ed
}

// releaseEditor 销毁编辑框(表格自己先销毁时调用)。
func (d *DataTable) releaseEditor() {
	d.editing = false
	ed := d.editor
	d.editor = nil
	if ed == nil {
		return
	}
	// 编辑器挂在窗口上: 窗口退出销毁时会把它先行销毁, 此时表格的 Destroy 事件
	// 里再 Destroy 就是"句柄可能无效"。只有表格先走、窗口还在的场景才需要手动销毁。
	if xc.XC_IsHELE(ed.Handle) {
		ed.Destroy()
	}
}

// render 按模式刷新表格体:
//   - 分页(默认): 只把当前页灌进原生 List, 原生内存锁在页大小;
//   - 虚拟滚动: 只下发总行数 + 可见行文本 provider, 行数据不进原生适配器。
//
// 两条路径的滚动绘制成本相同(原生 List 本来就只画可见行), 差别在大数据时的构建/内存。
func (d *DataTable) render() {
	if d.table == nil {
		return
	}
	if d.virtual {
		// 行下标在排序/筛选后指向别的原始行, 清掉原生单选高亮(勾选状态与它无关)。
		d.table.CancelSelectAll()
		d.table.setVirtualRows(int32(len(d.view)), d.virtualCellText)
		d.syncFooter()
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
		if d.filtersActive() {
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
	if d.virtual {
		return // 没有页: 页脚只保留总行数 / 勾选统计
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

// canHide 报告该列能否隐藏: 显式 NoHide 的列不行, 最后一列也不能(表格会变空壳)。
func (d *DataTable) canHide(ci int) bool {
	if ci < 0 || ci >= len(d.cols) || d.cols[ci].NoHide {
		return false
	}
	return len(d.visible) > 1
}

// showHeaderMenu 弹出列头菜单(tasks 的 DataTableColumnHeader): 升序 / 降序 / 隐藏。
// 升序降序是勾选项 —— 再点一次当前项 = 取消排序, 保留"升→降→取消"的能力;
// 隐藏走 SetColumnVisible(重建列结构, 见 8.169)。
func (d *DataTable) showHeaderMenu(ci int, tcol int32) {
	col := d.cols[ci]
	m := NewMenu()
	m.Width(160)
	if col.Sortable {
		m.Check("升序", d.hasSortKey(ci, DataSortAsc),
			MenuIcon(icons.IconArrowUp), MenuOnCheck(func(on bool) {
				d.applyHeaderSort(ci, DataSortAsc, on)
			}))
		m.Check("降序", d.hasSortKey(ci, DataSortDesc),
			MenuIcon(icons.IconArrowDown), MenuOnCheck(func(on bool) {
				d.applyHeaderSort(ci, DataSortDesc, on)
			}))
	}
	if d.canHide(ci) {
		if col.Sortable {
			m.Separator()
		}
		m.Item("隐藏", MenuIcon(icons.IconEyeOff), MenuOnSelect(func() {
			d.SetColumnVisible(ci, false)
		}))
	}
	// 表头点击回调不带坐标: 用绘制时缓存的表头单元格矩形做锚点, 弹层固定落在该列下方 ——
	// 用 PopupAtCursor 的话, 按住鼠标拖到别处再松开, 菜单会跑到松开的位置。
	if d.table != nil {
		if rc, ok := d.table.HeaderCellRect(tcol); ok {
			m.PopupAtElementRect(d.table.Handle, rc)
		} else {
			m.PopupAtCursor(xc.XWidget_GetHWINDOW(d.Handle))
		}
		// 排序图标 / 隐藏列都要等菜单关闭后再立即重绘(关闭会还原屏幕, 见 showFacetMenu)。
		d.table.Redraw(true)
		d.updateToolbar()
		return
	}
	m.PopupAtCursor(xc.XWidget_GetHWINDOW(d.Handle))
}

// applyHeaderSort 处理列头菜单的勾选: on=false 表示点了已勾选的那一项 = 去掉这个键。
// 列头菜单是单列排序入口 —— 勾选另一列的方向会换成只按那一列排; 多列排序用
// Shift+点击表头(见 cycleSortKey)。
func (d *DataTable) applyHeaderSort(ci int, dir DataSort, on bool) {
	if on {
		d.sortKeys = []DataSortKey{{Col: ci, Dir: dir}}
	} else if i := d.sortKeyIndex(ci); i >= 0 && d.sortKeys[i].Dir == dir {
		d.sortKeys = append(d.sortKeys[:i], d.sortKeys[i+1:]...)
	}
	d.apply(true)
	d.notifySort()
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
	rank := d.sortKeyIndex(ci)
	path, color := icons.IconArrowUpDown, th.MutedForeground
	if rank >= 0 {
		switch d.sortKeys[rank].Dir {
		case DataSortAsc:
			path, color = icons.IconArrowUp, th.Foreground
		case DataSortDesc:
			path, color = icons.IconArrowDown, th.Foreground
		}
	}
	font := xui.FontHandle(th.FontDefault)
	rankW := int32(0)
	if rank > 0 {
		rankW = dataTableSortRankW
	}
	x := rc.Left + tableCellPadX + xui.TextWidth(d.cols[ci].Title, font) + dataTableSortGap
	if limit := rc.Right - tableCellPadX - dataTableSortIcon - rankW; x > limit {
		x = limit
	}
	top := rc.Top + (rc.Bottom-rc.Top-dataTableSortIcon)/2
	ir := xc.RECT{Left: x, Top: top, Right: x + dataTableSortIcon, Bottom: top + dataTableSortIcon}
	p.Icon(path, &ir, color, 2)
	if rank > 0 {
		// 多列排序的次序(第 2、第 3 键): 箭头后的数字, 小一号加粗(9pt 常规会被画成宋体)。
		if nf := xui.FontHandle(xui.FontSpec{Family: th.FontDefault.Family, Size: 10, Bold: true}); nf != 0 {
			p.Draw.SetFont(nf)
			nr := xc.RECT{Left: x + dataTableSortIcon + 1, Top: top, Right: x + dataTableSortIcon + 1 + rankW, Bottom: top + dataTableSortIcon}
			p.Text(strconv.Itoa(rank+1), &nr, th.MutedForeground, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
			p.Draw.SetFont(font)
		}
	}
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
	// 正在这一格上编辑: 自绘内容让位给输入框(否则徽章/图标会从输入框底下透出来)。
	if d.editing && row == d.editRowT && tcol == d.editSub {
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

// dataTableView 返回过滤(分面 + 关键字) + 单列排序后的原始行下标。
// compare 为 nil 时用 dataCompare; 多列排序走 dataTableViewKeys。
func dataTableView(rows [][]string, filter string, filterCols []int, facets map[int]map[string]bool,
	sortCol int, dir DataSort, compare func(a, b string) int) []int {
	var keys []DataSortKey
	if sortCol >= 0 && dir != DataSortNone {
		keys = []DataSortKey{{Col: sortCol, Dir: dir}}
	}
	cmp := func(col int, a, b string) int {
		if compare != nil {
			return compare(a, b)
		}
		return dataCompare(a, b)
	}
	return dataTableViewKeys(rows, filter, filterCols, facets, keys, cmp)
}

// dataTableViewKeys 按多个排序键(主 → 次)排序: 前一个键相等才比较后面的键;
// 全部键相等时保持原始顺序(稳定排序)。compare 按列取比较函数(可为 nil)。
func dataTableViewKeys(rows [][]string, filter string, filterCols []int, facets map[int]map[string]bool,
	keys []DataSortKey, compare func(col int, a, b string) int) []int {
	f := strings.ToLower(strings.TrimSpace(filter))
	view := make([]int, 0, len(rows))
	for i, row := range rows {
		if !dataRowMatchesFacets(row, facets) {
			continue
		}
		if f != "" && !dataRowMatches(row, f, filterCols) {
			continue
		}
		view = append(view, i)
	}
	var valid []DataSortKey
	for _, k := range keys {
		if k.Col >= 0 && k.Dir != DataSortNone {
			valid = append(valid, k)
		}
	}
	if len(valid) == 0 {
		return view
	}
	value := func(row []string, col int) string {
		if col < len(row) {
			return row[col]
		}
		return ""
	}
	sort.SliceStable(view, func(a, b int) bool {
		for _, k := range valid {
			av, bv := value(rows[view[a]], k.Col), value(rows[view[b]], k.Col)
			var c int
			if compare != nil {
				c = compare(k.Col, av, bv)
			} else {
				c = dataCompare(av, bv)
			}
			if c == 0 {
				continue
			}
			if k.Dir == DataSortDesc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
	return view
}

// dataRowMatchesFacets 判断行是否命中全部分面筛选: **列内多选是 OR, 列间是 AND**,
// 与单元格原始文本做**相等**比较(tasks 里的状态/优先级筛选就是这个口径)。
func dataRowMatchesFacets(row []string, facets map[int]map[string]bool) bool {
	for col, set := range facets {
		if len(set) == 0 {
			continue
		}
		cell := ""
		if col >= 0 && col < len(row) {
			cell = row[col]
		}
		if !set[cell] {
			return false
		}
	}
	return true
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
