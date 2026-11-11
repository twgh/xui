package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 21: 数据2(虚拟滚动 DataTable) =====

// 数据2 表的列下标, 与 buildDataPage2 的 DataColumn 顺序一致(勾选列由组件加在最左)。
const (
	vTaskColID = iota
	vTaskColTitle
	vTaskColStatus
	vTaskColPriority
	vTaskColAmount
	vTaskColAction
)

const dataVirtualRows = 100000

func buildDataPage2(body *widget.LayoutEle) {
	r := rowStretch(body.Handle, gap)
	_, v := stretchCard(colStretch(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "数据表格 · 虚拟滚动", v.Handle))
	xui.ItemFill(ui.NewLabel(0, 0, 0, 40,
		"DataTable(虚拟滚动): "+strconv.Itoa(dataVirtualRows)+" 行数据全部留在 Go 侧, 原生 List 只给可见行填文本; "+
			"勾选 / 文本列双击编辑, 状态 / 优先级单击选择, 金额只允许数值 / 排序 / 筛选 / 列显隐与「数据」页的分页模式一致, 但页脚不翻页, 只显示总行数与勾选统计。"+
			"拖动右侧滚动条或滚轮浏览(分页模式见「数据」页)。10 万数据占用约 21MB 内存。", v.Handle).SetMuted().SetAutoWrap())
	xui.ItemFillHeight(v)

	cols := []ui.DataColumn{
		{Title: "任务", Width: 100, Sortable: true, Render: taskIDCell, ReadOnly: true},
		{Title: "标题", Sortable: true},
		{Title: "状态", Width: 110, Sortable: true, Render: taskStatusCell,
			EditOptions: []string{"待办", "进行中", "已完成", "已取消"},
			FilterOptions: []ui.DataFilterOption{
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
		{Title: "金额", Width: 130, Sortable: true, Align: ui.DataAlignRight, EditNumeric: true},
		{Title: "操作", Width: 52, NoHide: true, Render: taskActionCell, ReadOnly: true},
	}
	dt := ui.NewDataTable(0, 0, 0, 0, v.Handle, cols,
		ui.WithDataTableVirtualScroll(),
		ui.WithDataTableSelection(),
		ui.WithDataTableColumnToggle(),
		ui.WithDataTableEditing(),
		ui.WithDataTableFilterColumns(vTaskColID, vTaskColTitle, vTaskColStatus, vTaskColPriority, vTaskColAmount))
	xui.ItemFill(dt)
	xui.ItemFillHeight(dt)

	var rows [][]string
	demoData2Load = func() {
		rows = buildVirtualTaskRows(dataVirtualRows)
		dt.SetRows(rows)
		dt.AdjustLayout()
	}
	dt.OnRowClick(func(row int) {
		fmt.Printf("虚拟表 行点击: %s\n", rows[row][vTaskColID])
	})
	dt.OnSelectionChange(func(sel []int) {
		fmt.Printf("虚拟表 勾选: %d 行\n", len(sel))
	})
	dt.OnFilterChange(func(text string) {
		fmt.Printf("虚拟表 筛选: %q → %d 行\n", text, dt.FilteredRows())
	})
	dt.OnFacetChange(func(col int, values []string) {
		fmt.Printf("虚拟表 分面: 列 %d = %v → %d 行\n", col, values, dt.FilteredRows())
	})
	dt.OnCellClick(func(row, col int) {
		if col == vTaskColAction {
			showVirtualTaskMenu(xc.XWidget_GetHWINDOW(v.Handle), rows[row])
		}
	})
	dt.OnCellEdit(func(row, col int, old, next string) bool {
		next = strings.TrimSpace(next)
		switch col {
		case vTaskColTitle:
			if next == "" {
				return false
			}
		case vTaskColStatus:
			switch next {
			case "待办", "进行中", "已完成", "已取消":
			default:
				return false
			}
		case vTaskColPriority:
			switch next {
			case "高", "中", "低":
			default:
				return false
			}
		case vTaskColAmount:
			if next == "" {
				return false
			}
		default:
			return false
		}
		rows[row][col] = next
		fmt.Printf("虚拟表 编辑: %s 列 %d: %q → %q\n", rows[row][vTaskColID], col, old, next)
		return true
	})
}

// buildVirtualTaskRows 生成虚拟滚动演示数据(不依赖 demoTasks, 渲染回调只读单元格文本)。
func buildVirtualTaskRows(n int) [][]string {
	statuses := []string{"待办", "进行中", "已完成", "已取消"}
	priorities := []string{"高", "中", "低"}
	actions := []string{"同步", "压缩", "解析", "索引", "备份", "迁移", "校验", "清理"}
	modules := []string{"云端同步", "离线缓存", "任务队列", "数据管道", "权限服务", "报表引擎"}
	rows := make([][]string, n)
	for i := 0; i < n; i++ {
		id := "TASK-" + strconv.Itoa(900000-i)
		title := actions[i%len(actions)] + modules[(i/len(actions))%len(modules)] + " #" + strconv.Itoa(i+1)
		rows[i] = []string{
			id,
			title,
			statuses[i%len(statuses)],
			priorities[i%3],
			strconv.Itoa((i*37)%1000000) + ".00",
			"",
		}
	}
	return rows
}

func showVirtualTaskMenu(hWindow int, row []string) {
	m := ui.NewMenu()
	m.Width(176)
	m.Label(row[vTaskColID])
	m.Item("查看", ui.MenuIcon(icons.IconEye), ui.MenuOnSelect(func() {
		fmt.Printf("虚拟表 查看: %s\n", row[vTaskColID])
	}))
	m.Item("复制任务 ID", ui.MenuIcon(icons.IconCopy), ui.MenuOnSelect(func() {
		fmt.Printf("虚拟表 复制: %s\n", row[vTaskColID])
	}))
	m.Separator()
	m.Item("删除", ui.MenuDestructive(), ui.MenuIcon(icons.IconTrash), ui.MenuOnSelect(func() {
		fmt.Printf("虚拟表 删除: %s\n", row[vTaskColID])
	}))
	m.PopupAtCursor(hWindow)
}
