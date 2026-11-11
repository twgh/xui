package main

import (
	"fmt"
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 20: 数据(DataTable / 树) =====
func buildDataPage(body *widget.LayoutEle) {
	r := rowStretch(body.Handle, gap) // 铺满型页面: 表格/树跟随窗口长高(内容区已 SetContentFill)

	// ---- 数据表格(整行占满剩余高度, 表格跟随窗口长高) ----
	_, v := stretchCard(colStretch(r.Handle, 3, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "数据表格", v.Handle))
	xui.ItemFill(ui.NewLabel(0, 0, 0, 40, "DataTable(tasks): 勾选 / 分面 / 列头菜单 / 视图; Shift 多列排序与跨页全选, 拖拽表头换列; "+
		"文本列双击编辑(Enter / 失焦提交, Esc 取消); 状态 / 优先级单击弹固定选项菜单。", v.Handle).SetMuted().SetAutoWrap())
	xui.ItemFillHeight(v) // 内容区占满卡片剩余高度
	dt := ui.NewDataTable(0, 0, 0, 0, v.Handle, []ui.DataColumn{
		{Title: "任务", Width: 92, Sortable: true, Render: taskIDCell},
		{Title: "标题", Sortable: true, Render: taskTitleCell}, // Width<=0: 占满剩余宽度
		{Title: "状态", Width: 104, Sortable: true, Render: taskStatusCell,
			EditOptions: []string{"待办", "进行中", "已完成", "已取消"},
			FilterOptions: []ui.DataFilterOption{ // 工具栏"状态"多选(列内 OR)
				{Label: "待办", Icon: icons.IconCircleDashed},
				{Label: "进行中", Icon: icons.IconTimer},
				{Label: "已完成", Icon: icons.IconCircleCheck},
				{Label: "已取消", Icon: icons.IconCircleX},
			}},
		{Title: "优先级", Width: 92, Sortable: true, Render: taskPriorityCell,
			EditOptions: []string{"高", "中", "低"},
			FilterOptions: []ui.DataFilterOption{
				{Label: "高", Icon: icons.IconArrowUp},
				{Label: "中", Icon: icons.IconArrowRight},
				{Label: "低", Icon: icons.IconArrowDown},
			}},
		{Title: "操作", Width: 52, NoHide: true, Render: taskActionCell, ReadOnly: true}, // 行操作列不进"隐藏", 也不可编辑
	},
		ui.WithDataTableSelection(),
		ui.WithDataTableColumnToggle(),
		ui.WithDataTableEditing(), // 双击单元格原地编辑(Enter / 失焦提交, Esc 取消)
		ui.WithDataTablePageSize(10),
		ui.WithDataTablePageSizes(10, 20, 30),
		// 关键字筛选只匹配"任务 / 标题 / 状态 / 优先级"这四列(操作列没有文字)
		ui.WithDataTableFilterColumns(taskColID, taskColTitle, taskColStatus, taskColPriority))
	xui.ItemFill(dt)
	xui.ItemFillHeight(dt) // 表格占满内容区里"标题/说明之外"的剩余高度
	rows := make([][]string, len(demoTasks))
	for i, task := range demoTasks {
		rows[i] = []string{task.ID, task.Title, task.Status, task.Priority, ""}
	}
	dt.SetRows(rows)
	dt.OnSortChange(func(col int, dir ui.DataSort) {
		fmt.Printf("排序: 列 %d, 方向 %d\n", col, dir)
	})
	dt.OnRowClick(func(row int) {
		fmt.Printf("点击数据行: %s\n", demoTasks[row].ID)
	})
	dt.OnSelectionChange(func(sel []int) {
		fmt.Printf("勾选: %v\n", sel)
	})
	dt.OnFacetChange(func(col int, values []string) {
		fmt.Printf("分面筛选: 列 %d = %v (剩 %d 行)\n", col, values, dt.FilteredRows())
	})
	dt.OnCellClick(func(row, col int) {
		if col == taskColAction { // 行操作: 在光标处弹官方 tasks 菜单
			showTaskMenu(xc.XWidget_GetHWINDOW(v.Handle), demoTasks[row])
		}
	})
	// 编辑回调返回 false = 拒绝本次修改(单元格保持旧值)。自绘列读的是 demoTasks,
	// 所以这里同步结构体, 徽章 / 状态图标才会跟着新值重画。
	dt.OnCellEdit(func(row, col int, old, next string) bool {
		next = strings.TrimSpace(next)
		switch col {
		case taskColID:
			if next == "" {
				return false
			}
			demoTasks[row].ID = next
		case taskColTitle:
			if next == "" { // 标题不允许清空
				return false
			}
			demoTasks[row].Title = next
		case taskColStatus:
			switch next { // 状态只认四个枚举值, 其它输入直接拒绝
			case "待办", "进行中", "已完成", "已取消":
				demoTasks[row].Status = next
			default:
				return false
			}
		case taskColPriority:
			switch next {
			case "高", "中", "低":
				demoTasks[row].Priority = next
			default:
				return false
			}
		default:
			return false
		}
		fmt.Printf("编辑: %s 列 %d: %q → %q\n", demoTasks[row].ID, col, old, next)
		return true
	})

	// ---- 树 ----
	_, v2 := stretchCard(colStretch(r.Handle, 2, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "树形视图", v2.Handle))
	xui.ItemFill(ui.NewLabel(0, 0, 0, 40, "原生 Tree 载体 + 全自绘皮肤: 缩进参考线 / 旋转 chevron / 行悬浮与选中 bg-accent。"+
		"勾选模式下父节点随后代显示全选 / 半选, 点父节点级联全部后代; 点复选框不改变选中行。", v2.Handle).SetMuted().SetAutoWrap())
	xui.ItemFillHeight(v2)
	demoTree = ui.NewTree(0, 0, 0, 0, v2.Handle, ui.WithTreeCheckable())
	xui.ItemFill(demoTree)
	xui.ItemFillHeight(demoTree)
	demoTreeSel = buildDemoTree(demoTree)
	demoTree.OnSelect(func(id int32) {
		fmt.Printf("选中项: %s\n", demoTree.ItemText(id))
	})
	demoTree.OnCheckChange(func(id int32, checked bool) {
		fmt.Printf("勾选: %s = %v (共 %d 项)\n", demoTree.ItemText(id), checked, len(demoTree.CheckedKeys()))
	})
}
