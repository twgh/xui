package main

import (
	"fmt"

	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 数据页: tasks 风格任务表(对齐 shadcn data-table 的 tasks 示例) =====

// 任务表的列下标, 与 buildDataPage 里的 DataColumn 顺序一致(勾选列由组件加在最左)。
const (
	taskColID = iota
	taskColTitle
	taskColStatus
	taskColPriority
	taskColAction
)

// demoTask 是任务表一行的原始数据。行文本只承担"排序 / 关键字筛选", 徽章 / 状态图标 /
// 行操作钮由 DataColumn.Render 按**原始行下标**回查结构体绘制。
type demoTask struct {
	ID       string
	Title    string
	Label    string // 标签徽章: 文档 / 缺陷 / 功能 / 增强(空 = 不画徽章)
	Status   string // 待办 / 进行中 / 已完成 / 已取消
	Priority string // 高 / 中 / 低
}

var demoTasks = []demoTask{
	{"TASK-8782", "无法压缩离线量化流程里的程序", "文档", "进行中", "高"},
	{"TASK-8781", "不量化源码就能生成开放源码 SSD 像素", "缺陷", "已完成", "中"},
	{"TASK-8780", "接上多协议 SSD 面板时会绕过接口", "增强", "待办", "低"},
	{"TASK-8779", "数字化主控时钟而不用合成", "功能", "已取消", "中"},
	{"TASK-8778", "把虚拟接口面板输入固态矩阵", "文档", "进行中", "高"},
	{"TASK-8777", "压缩开放架构程序时程序不再启动", "缺陷", "待办", "高"},
	{"TASK-8776", "量化跨平台协议而不接面板", "增强", "已完成", "低"},
	{"TASK-8775", "解析冗余服务时内存持续上涨", "缺陷", "进行中", "高"},
	{"TASK-8774", "同步无线带宽而不用编解码器", "功能", "待办", "中"},
	{"TASK-8773", "索引数字总线端口后界面卡顿", "缺陷", "已完成", "中"},
	{"TASK-8772", "覆写移动端显卡驱动协议", "增强", "已取消", "低"},
	{"TASK-8771", "旁路核心内核而不改固件", "文档", "待办", "低"},
	{"TASK-8770", "复制多字节程序时丢字符", "缺陷", "进行中", "高"},
	{"TASK-8769", "生成跨平台防火墙像素", "功能", "已完成", "中"},
	{"TASK-8768", "压缩开放源码矩阵而不开线程池", "文档", "待办", "低"},
	{"TASK-8767", "计算虚拟硬盘接口的带宽", "增强", "进行中", "中"},
	{"TASK-8766", "拖动窗口时标题栏闪烁", "缺陷", "已取消", "低"},
	{"TASK-8765", "同步主控面板后主题不生效", "缺陷", "待办", "高"},
	{"TASK-8764", "解析冗余主控协议并缓存结果", "功能", "已完成", "中"},
	{"TASK-8763", "量化数字时钟信号而不降采样", "增强", "进行中", "低"},
	{"TASK-8762", "接上辅助总线端口时会丢包", "缺陷", "已完成", "高"},
	{"TASK-8761", "把多协议 SSD 矩阵输入虚拟面板", "文档", "待办", "中"},
	{"TASK-8760", "覆写冗余服务而不断电", "功能", "已取消", "低"},
	{"TASK-8759", "生成无线驱动协议时超时", "缺陷", "进行中", "高"},
}

// taskIDCell 任务编号: text-xs muted-mono 观感(与 tasks 示例的 w-[80px] span 一致)。
func taskIDCell(p *xui.Painter, c ui.DataCell) {
	t := xui.Current()
	rc := c.Rc
	rc.Left += 8
	rc.Right -= 8
	p.Draw.SetFont(xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size - 2}))
	p.Text(c.Text, &rc, t.MutedForeground,
		xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
}

// taskTitleCell 标题: 左侧标签徽章(Badge outline) + 标题文本(超宽裁剪)。
func taskTitleCell(p *xui.Painter, c ui.DataCell) {
	t := xui.Current()
	task := demoTasks[c.Row]
	rc := c.Rc
	x := rc.Left + 8
	if task.Label != "" {
		small := xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size - 2})
		w := xui.TextWidth(task.Label, small) + 12 // px-1.5 ×2
		top := rc.Top + (rc.Bottom-rc.Top-18)/2
		br := xc.RECT{Left: x, Top: top, Right: x + w, Bottom: top + 18}
		p.StrokeRound(&br, t.Border, 6, 1)
		p.Draw.SetFont(small)
		p.Text(task.Label, &br, t.Foreground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		x = br.Right + 8
	}
	tr := rc
	tr.Left = x
	tr.Right -= 8
	if tr.Left >= tr.Right {
		return
	}
	p.Draw.SetFont(xui.FontHandle(t.FontDefault))
	p.Draw.SetClipRect(&tr)
	p.Text(task.Title, &tr, t.Foreground,
		xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
	p.Draw.ClearClip()
}

// taskStatusCell 状态: 图标(已完成 ✓ / 进行中 ⏱ / 已取消 ✕ / 待办 ○) + 文本。
func taskStatusCell(p *xui.Painter, c ui.DataCell) {
	t := xui.Current()
	icon := icons.IconCircleDashed
	switch c.Text {
	case "已完成":
		icon = icons.IconCircleCheck
	case "进行中":
		icon = icons.IconTimer
	case "已取消":
		icon = icons.IconCircleX
	}
	drawTaskIconText(p, c.Rc, icon, c.Text, t.MutedForeground, t.Foreground)
}

// taskPriorityCell 优先级: 高 ↑ / 中 → / 低 ↓。
func taskPriorityCell(p *xui.Painter, c ui.DataCell) {
	t := xui.Current()
	icon := icons.IconArrowDown
	switch c.Text {
	case "高":
		icon = icons.IconArrowUp
	case "中":
		icon = icons.IconArrowRight
	}
	drawTaskIconText(p, c.Rc, icon, c.Text, t.MutedForeground, t.Foreground)
}

// taskActionCell 行操作: 居中的省略号钮(点击整格弹出菜单, 见 OnCellClick)。
func taskActionCell(p *xui.Painter, c ui.DataCell) {
	t := xui.Current()
	size := int32(16)
	x := c.Rc.Left + (c.Rc.Right-c.Rc.Left-size)/2
	y := c.Rc.Top + (c.Rc.Bottom-c.Rc.Top-size)/2
	ir := xc.RECT{Left: x, Top: y, Right: x + size, Bottom: y + size}
	p.Icon(icons.IconEllipsis, &ir, t.MutedForeground, 2)
}

// drawTaskIconText 是"小图标 + 文本"单元格的公共排版(状态 / 优先级)。
func drawTaskIconText(p *xui.Painter, rc xc.RECT, icon, text string, iconColor, textColor xui.Color) {
	t := xui.Current()
	size := int32(14)
	x := rc.Left + 8
	top := rc.Top + (rc.Bottom-rc.Top-size)/2
	ir := xc.RECT{Left: x, Top: top, Right: x + size, Bottom: top + size}
	p.Icon(icon, &ir, iconColor, 2)
	tr := rc
	tr.Left = x + size + 6
	tr.Right -= 8
	p.Draw.SetFont(xui.FontHandle(t.FontDefault))
	p.Text(text, &tr, textColor,
		xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
}

// showTaskMenu 行操作菜单(对齐 tasks 示例的 DataTableRowActions)。
func showTaskMenu(hWindow int, task demoTask) {
	m := ui.NewMenu()
	m.Width(176)
	m.Label(task.ID)
	m.Item("编辑", ui.MenuIcon(icons.IconPencil), ui.MenuOnSelect(func() {
		fmt.Printf("编辑: %s\n", task.ID)
	}))
	m.Item("复制任务 ID", ui.MenuIcon(icons.IconCopy), ui.MenuOnSelect(func() {
		fmt.Printf("复制: %s\n", task.ID)
	}))
	m.Item("收藏", ui.MenuIcon(icons.IconStar), ui.MenuOnSelect(func() {
		fmt.Printf("收藏: %s\n", task.ID)
	}))
	m.Sub("标签", func(s *ui.Menu) {
		for _, label := range []string{"文档", "缺陷", "功能", "增强"} {
			label := label
			s.Radio(label, task.Label == label, ui.MenuGroup(1), ui.MenuOnCheck(func(on bool) {
				if on {
					fmt.Printf("标签: %s → %s\n", task.ID, label)
				}
			}))
		}
	}, ui.MenuIcon(icons.IconTag))
	m.Separator()
	m.Item("删除", ui.MenuDestructive(), ui.MenuIcon(icons.IconTrash), ui.MenuOnSelect(func() {
		fmt.Printf("删除: %s\n", task.ID)
	}))
	m.PopupAtCursor(hWindow)
}
