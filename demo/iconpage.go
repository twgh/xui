package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/twgh/xcgui/wapi/wutil"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 24: 图标(icons 资源子包) =====
// 网格是"一个格子一个自绘元素"(各自带 hover 态), 翻页只改格子里的名字 + 重绘,
// 不销毁元素(见 DESIGN 第八节 44)。格子默认只画图标: 悬浮出提示气泡显示名字,
// 点击把名字复制到剪贴板。

func buildIconPage(body *widget.LayoutEle) {
	_, v := autoCard(body.Handle, 14)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "图标", v.Handle))
	xui.ItemFill(ui.NewLabel(0, 0, 0, 40,
		"运行时自带 lucide 图标资源(lucide-static "+icons.Version+", 共 "+
			strconv.Itoa(icons.Count())+" 个)。\n用法: p.Icon(icons.Path(\"star\"), &rc, color, 2); "+
			"悬浮图标看名字, 点击复制名字。", v.Handle).SetMuted().SetWrap())
	// 内容区水平居中: 固定尺寸的图标网格才会居中(Fill 宽的子项不受影响)
	v.SetAlignH(xcc.Layout_Align_Center)

	// 搜索行: 输入框(定宽) + 结果计数 + 复制反馈(两份等宽, 各占剩余宽度一半)
	searchRow := xui.HStack(v.Handle, 32, 8)
	iconSearch = ui.NewInput(0, 0, 0, 0, searchRow.Handle,
		ui.WithPlaceholder("搜索图标名, 如 arrow / star / chevron"))
	xui.ItemWidth(iconSearch, 260)
	iconInfo = ui.NewLabel(0, 0, 0, 0, "", searchRow.Handle).SetMuted()
	xui.ItemWeight(iconInfo, 1)
	iconCopied = ui.NewLabel(0, 0, 0, 0, "", searchRow.Handle).SetMuted()
	xui.ItemWeight(iconCopied, 1)

	// 分页(shadcn pagination: 上一页/页码/省略号/下一页, 内容居中)
	iconPag = ui.NewPagination(0, 0, 0, 0, v.Handle,
		ui.WithPrevText("上一页"), ui.WithNextText("下一页"))
	xui.ItemFill(iconPag)
	iconPag.OnChange(func(page int) { showIconPage(page - 1) })

	// 网格容器尺寸在创建时就给足(格子是固定尺寸), 不需要声明布局尺寸
	grid := widget.NewElement(0, 0, int32(iconCols*iconTileW), int32(iconRows*iconTileH), v.Handle)
	grid.EnableBkTransparent(true).EnableDrawBorder(false).EnableFocus(false) // 纯容器: 不画底(暗色下会是一块白)
	iconTiles = iconTiles[:0]
	for r := 0; r < iconRows; r++ {
		for c := 0; c < iconCols; c++ {
			iconTiles = append(iconTiles,
				newIconTile(int32(c)*iconTileW, int32(r)*iconTileH, grid.Handle))
		}
	}

	iconSearch.OnChange(func(text string) {
		filterIcons(text)
		iconCopied.SetText("")
		iconCopied.Redraw(false)
		showIconPage(0)
	})
	filterIcons("")
	showIconPage(0)
}

// ===== 图标总览页的状态与绘制 =====

const (
	iconCols  = 9
	iconRows  = 4
	iconTileW = 77 // 696 / 9
	iconTileH = 60
)

var (
	iconTiles    []*iconTile // 网格里复用的格子(翻页只换名字)
	iconFiltered []string    // 当前筛选结果(图标名, 升序)
	iconPageIdx  int
	iconInfo     *ui.Label      // 结果计数 / 当前页
	iconCopied   *ui.Label      // 复制反馈
	iconSearch   *ui.Input      // 搜索框
	iconPag      *ui.Pagination // 分页组件
)

// iconTile 是网格里的一格: 只画图标(名字走悬浮提示), 悬浮 bg-accent, 点击复制名字。
type iconTile struct {
	ele   *widget.Element
	name  string
	hover bool
}

func newIconTile(x, y int32, parent int) *iconTile {
	ele := widget.NewElement(x, y, iconTileW, iconTileH, parent)
	if ele == nil {
		return nil
	}
	t := &iconTile{ele: ele}
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(t.onPaint)
	// 只在 hover 状态**变化**时重绘: MouseStay 是"鼠标在元素内移动"就会来一发的
	// (按下不判状态就会变成每个鼠标移动事件都触发一次重绘, 45 个格子扫一遍很难看)
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		if t.hover {
			return 0
		}
		t.hover = true
		ele.Redraw(false)
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		if !t.hover {
			return 0
		}
		t.hover = false
		ele.Redraw(false)
		return 0
	})
	// 点击把图标名复制到剪贴板(xcgui 自带的剪贴板封装)
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if t.name == "" {
			return 0
		}
		if err := wutil.SetClipboardText(t.name); err != nil {
			iconCopied.SetText("复制失败: " + err.Error())
			iconCopied.Redraw(false)
			return 0
		}
		iconCopied.SetText("已复制: " + t.name)
		iconCopied.Redraw(false)
		return 0
	})
	return t
}

// setName 换格子显示的图标(空串 = 本页没有这么多图标), 并同步悬浮提示的文字。
func (t *iconTile) setName(name string) {
	if t.name == name {
		return
	}
	t.name = name
	if tip != nil {
		// 悬浮气泡显示完整图标名(名字每次翻页/筛选都会变, Bind 可重复调用改文字)
		tip.Bind(t.ele.Handle, name)
	}
	t.ele.Redraw(false)
}

func (t *iconTile) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	*pbHandled = true
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	th := xui.Current()
	// 透明子元素重绘不会擦底: 先铺宿主底色(卡片), 否则 hover 离开后残留高亮块(见 DESIGN 第八节 28)
	p.FillRound(&p.Rc, th.Card, 0)
	if hv := xui.Hover(hEle, t.hover); hv > 0 {
		p.FillRound(&p.Rc, th.Accent.ScaleAlpha(hv), th.RadiusSm())
	}
	if t.name == "" {
		return 0 // 空格子(本页没有这么多图标)
	}
	// 只画图标(名字不画: 悬浮出提示气泡, 点击复制), 居中比无名字时大一号
	const iconSize = 28
	iconRc := xc.RECT{
		Left:   p.Rc.Left + (p.W()-iconSize)/2,
		Top:    p.Rc.Top + (p.H()-iconSize)/2,
		Right:  p.Rc.Left + (p.W()-iconSize)/2 + iconSize,
		Bottom: p.Rc.Top + (p.H()-iconSize)/2 + iconSize,
	}
	p.Icon(icons.Path(t.name), &iconRc, th.Foreground, 2)
	return 0
}

// filterIcons 按关键字过滤图标名(大小写不敏感, 空关键字 = 全部), 结果保持名字升序。
func filterIcons(q string) {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		iconFiltered = icons.Names()
		return
	}
	out := make([]string, 0, 64)
	for _, n := range icons.Names() {
		if strings.Contains(strings.ToLower(n), q) {
			out = append(out, n)
		}
	}
	iconFiltered = out
}

// showIconPage 翻到第 idx 页(越界夹紧), 名字取自筛选结果(按名字升序)。
func showIconPage(idx int) {
	if len(iconTiles) == 0 {
		return
	}
	names := iconFiltered
	pages := (len(names) + len(iconTiles) - 1) / len(iconTiles)
	if pages < 1 {
		pages = 1
	}
	if idx < 0 {
		idx = 0
	}
	if idx > pages-1 {
		idx = pages - 1
	}
	iconPageIdx = idx
	base := idx * len(iconTiles)
	for i, t := range iconTiles {
		if base+i < len(names) {
			t.setName(names[base+i])
		} else {
			t.setName("")
		}
	}
	if iconInfo != nil {
		if len(names) == icons.Count() {
			iconInfo.SetText(fmt.Sprintf("共 %d 个图标 · 第 %d/%d 页", len(names), idx+1, pages))
		} else {
			iconInfo.SetText(fmt.Sprintf("筛选出 %d/%d 个 · 第 %d/%d 页",
				len(names), icons.Count(), idx+1, pages))
		}
		// 原生按钮改文字不会自己失效, 得手动重绘(与 SetIcon 后 w.Redraw(false) 同理)
		iconInfo.Redraw(false)
	}
	if iconPag != nil {
		// 分页组件不做数据分页: 调用方把总数/每页数同步进去, 再同步当前页
		// (SetPage 不触发 OnChange, 不会回流到 showIconPage 造成递归)
		iconPag.SetTotal(len(names))
		iconPag.SetPageSize(len(iconTiles))
		iconPag.SetPage(idx + 1)
	}
}

// buildDemoTree 填充演示树(文件树形态, 与 shadcn App.tsx 对照卡同构), 返回默认选中项。
func buildDemoTree(t *ui.Tree) int32 {
	src := t.AddRoot("src")
	comp := t.AddChild(src, "components")
	button := t.AddChild(comp, "button.go")
	t.AddChild(comp, "table.go")
	treeItem := t.AddChild(comp, "tree.go")
	demo := t.AddChild(src, "demo")
	t.AddChild(demo, "main.go")
	t.AddChild(src, "go.mod")
	docs := t.AddRoot("docs")
	t.AddChild(docs, "DESIGN.md")
	t.AddChild(docs, "DESIGN-PITFALLS.md")
	t.AddChild(docs, "HANDOFF.md")
	t.AddChild(docs, "CHANGELOG.md")
	readme := t.AddRoot("README.md")
	// 一棵勾选了一半的树: 叶子勾一个 → components / src 自动半选; 根级叶子单独全选
	t.SetChecked(button, true)
	t.SetChecked(readme, true)
	t.Expand(comp, true)
	t.Expand(src, true)
	t.Select(treeItem)
	return treeItem
}

// demo 菜单状态(菜单是一次性对象, 状态由调用方持有, 每次弹出重建)
var (
	demoLineNo    = true
	demoThemeG1   = true  // 单选组: 浅色
	demoThemeG2   = false // 单选组: 深色
	demoBookmarks = true
	demoURLs      = false
	demoPerson    = int32(1) // 右键菜单单选组: 1 张伟 / 2 李娜
)

// showDemoMenu 弹出演示菜单, 覆盖全部条目类型。
func showDemoMenu(hEle int) {
	m := ui.NewMenu()
	// 明确指定弹层宽度; 不加 Width 时组件默认 = max(触发区宽, 128)。
	m.Width(224)
	m.Item("复制", ui.MenuShortcut("Ctrl+C"), ui.MenuOnSelect(func() {}))
	m.Item("粘贴", ui.MenuShortcut("Ctrl+V"))
	m.Separator()
	m.Label("外观")
	m.Check("显示行号", demoLineNo, ui.MenuOnCheck(func(on bool) { demoLineNo = on }))
	m.Item("更多外观设置…", ui.MenuInset()) // 文字左缩进, 与勾选条目的文字对齐(shadcn inset)
	m.Radio("浅色主题", demoThemeG1, ui.MenuGroup(1), ui.MenuOnCheck(func(on bool) {
		if on {
			demoThemeG1, demoThemeG2 = true, false
		}
	}))
	m.Radio("深色主题", demoThemeG2, ui.MenuGroup(1), ui.MenuOnCheck(func(on bool) {
		if on {
			demoThemeG1, demoThemeG2 = false, true
		}
	}))
	m.Sub("更多操作", func(s *ui.Menu) {
		s.Item("刷新", ui.MenuShortcut("F5"))
		s.Item("另存为", ui.MenuShortcut("Ctrl+Shift+S"))
	})
	m.Separator()
	m.Item("删除项目", ui.MenuDestructive())
	m.Item("灰掉的选项", ui.MenuDisabled())
	m.PopupAtElement(hEle)
}

// contextTrigger 建右键触发区: border-dashed 虚线框 + 居中两行提示
// (对应 shadcn ContextMenuDemo 的 "Right click here" 虚线区域)。
func contextTrigger(parent int) *widget.Element {
	var hovered, opened bool
	box := widget.NewElement(0, 0, 0, 150, parent)
	box.EnableBkTransparent(true)
	xui.ItemFill(box)
	xui.ItemHeight(box, 150)
	box.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		hovered = true
		box.Redraw(true) // 立即重绘一次, 让 xui.Hover 在 paint 里启动过渡(同 Badge/Slider)
		return 0
	})
	box.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		hovered = false
		box.Redraw(true)
		return 0
	})
	box.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		t := xui.Current()
		// 触发区给 hover / 展开两档反馈(对应按钮的 hover / aria-expanded):
		// 右键本身就是"点在一个区域上", 没有反馈的话菜单像凭空出现。
		hv := xui.Hover(hEle, hovered)
		ov := xui.HoverItem(hEle, 0, opened)
		// 透明元素重绘不擦底(见 DESIGN 28): 先铺宿主卡片底色, 否则悬停淡出会留旧像素。
		p.FillRound(&p.Rc, t.Card, t.RadiusMd())
		// hover = bg-accent(与 ghost 按钮同一档); 打开态继续满色, 靠边框变 Ring 区分。
		p.FillRound(&p.Rc, t.Accent.ScaleAlpha(hv+(1-hv)*ov), t.RadiusMd())
		rc := p.Rc
		// 边框线宽 1, 沿路径居中; 内缩 1px 让虚线完整落在客户区内(同 borderedBox)。
		rc.Left++
		rc.Top++
		rc.Right--
		rc.Bottom--
		d := p.Draw
		// 4px 线 + 3px 间隔的短划线(shadcn border-dashed 的观感;
		// xcgui 的 Dottedline 是 1px 点线, 在这个尺寸下几乎看不见)。
		d.SetLineWidth(1)
		mix := hv
		if ov > mix {
			mix = ov
		}
		d.SetBrushColor(uint32(xui.Fade(t.Border, t.Ring, mix)))
		for x := rc.Left; x < rc.Right; x += 7 {
			x2 := x + 4
			if x2 > rc.Right {
				x2 = rc.Right
			}
			d.DrawLine(x, rc.Top, x2, rc.Top)
			d.DrawLine(x, rc.Bottom, x2, rc.Bottom)
		}
		for y := rc.Top; y < rc.Bottom; y += 7 {
			y2 := y + 4
			if y2 > rc.Bottom {
				y2 = rc.Bottom
			}
			d.DrawLine(rc.Left, y, rc.Left, y2)
			d.DrawLine(rc.Right, y, rc.Right, y2)
		}

		mid := (rc.Top + rc.Bottom) / 2
		line1 := xc.RECT{Left: rc.Left, Top: mid - 22, Right: rc.Right, Bottom: mid}
		// 对齐必须在 SetFont 之后设置: XDraw_SetFont 会重置文本对齐状态,
		// 先设 align 再换字体会让这一行退回默认的左对齐(实测)。
		d.SetFont(xui.FontHandle(t.FontDefault))
		p.Text("在此处右键", &line1, t.Foreground,
			xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		d.SetFont(xui.FontHandle(xui.FontSpec{
			Family: t.FontDefault.Family,
			Size:   t.FontDefault.Size - 2, // text-xs
		}))
		line2 := xc.RECT{Left: rc.Left, Top: mid + 2, Right: rc.Right, Bottom: mid + 20}
		p.Text("ContextMenuTrigger · right click", &line2, t.MutedForeground,
			xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		return 0
	})
	cm := ui.NewContextMenu(box.Handle, showDemoContextMenu)
	cm.OnOpenChange(func(open bool) {
		opened = open
		box.Redraw(true) // 阻塞式弹层: 先把这个展开态画出来
	})
	return box
}

// showDemoContextMenu 右键菜单示例, 与 shadcn ContextMenuDemo 同构:
// 条目 + 快捷键 + 前置图标 + SubTrigger inset + 勾选 + 单选组 + 破坏性。
func showDemoContextMenu(m *ui.Menu) {
	m.Width(208) // w-52
	m.Item("后退", ui.MenuShortcut("Ctrl+["), ui.MenuIcon(icons.IconArrowLeft))
	m.Item("前进", ui.MenuShortcut("Ctrl+]"), ui.MenuIcon(icons.IconArrowRight), ui.MenuDisabled())
	m.Item("重新加载", ui.MenuShortcut("Ctrl+R"), ui.MenuIcon(icons.IconRotateCw))
	m.Sub("更多工具", func(s *ui.Menu) {
		s.Item("保存页面…")
		s.Item("创建快捷方式…")
		s.Item("重命名窗口…")
		s.Separator()
		s.Item("开发者工具")
		s.Separator()
		s.Item("删除", ui.MenuDestructive())
	}, ui.MenuInset(), ui.MenuIcon(icons.IconWrench))
	m.Separator()
	m.Check("显示书签", demoBookmarks, ui.MenuOnCheck(func(on bool) { demoBookmarks = on }))
	m.Check("显示完整 URL", demoURLs, ui.MenuOnCheck(func(on bool) { demoURLs = on }))
	m.Separator()
	// 分组标签与条目文字同列(不 inset): 本菜单的 Check/Radio 勾选在右侧、条目文字
	// 从 6 起, 再给 Label 加 inset 会把它单独推出 22px, 看起来像没对齐。
	m.Label("人员")
	m.Radio("张伟", demoPerson == 1, ui.MenuGroup(2), ui.MenuOnCheck(func(on bool) {
		if on {
			demoPerson = 1
		}
	}))
	m.Radio("李娜", demoPerson == 2, ui.MenuGroup(2), ui.MenuOnCheck(func(on bool) {
		if on {
			demoPerson = 2
		}
	}))
}

// buildDialogs 三个对话框: 表单型(编辑资料)、确认型(删除项目) 与可滚动内容。
func buildDialogs(hWindow int, editTrigger, delTrigger, scrollTrigger *ui.Button) {
	// 表单对话框: 标题 + 描述 + 两个带标签的输入框 + 页脚
	edit := ui.NewDialog(hWindow)
	edit.SetTitle("编辑资料")
	edit.SetDescription("修改你的公开资料, 完成后点击保存。")
	edit.BindTrigger(editTrigger)

	// 内容容器是普通元素, 子元素按显式坐标摆放(对话框宽度固定, 高度由对话框自动量出)。
	// 对齐 shadcn App.tsx: 两个 space-y-2 组(标签 14 + 8 + 输入框 32), 组间 gap-4
	cw := edit.ContentWidth()
	ui.NewLabel(0, -2, cw, 20, "显示名称", edit.Content())
	name := ui.NewInput(0, 22, cw, 0, edit.Content(), ui.WithPlaceholder("显示名称"))
	name.SetValue("twgh")
	ui.NewLabel(0, 68, cw, 20, "个人主页", edit.Content())
	ui.NewInput(0, 92, cw, 0, edit.Content(), ui.WithPlaceholder("https://"))

	edit.AddFooter("取消", ui.ButtonOutline, func() { edit.Close() })
	edit.AddFooter("保存", ui.ButtonPrimary, func() { edit.Close() })

	// 确认对话框: 只有标题 + 描述 + 页脚
	del := ui.NewDialog(hWindow)
	del.SetTitle("确认删除?")
	del.SetDescription("删除后无法恢复, 确定要删除该项目吗?")
	del.BindTrigger(delTrigger)
	del.AddFooter("取消", ui.ButtonOutline, func() { del.Close() })
	del.AddFooter("删除", ui.ButtonDestructive, func() { del.Close() })

	// 可滚动内容: 对应 shadcn 文档的 Scrollable Content / With Sticky Footer ——
	// 头部与页脚固定, 中间内容超过 70vh(实际再夹到面板可用高度)后出滚动条。
	scroll := ui.NewDialog(hWindow, ui.WithDialogScroll(0))
	scroll.SetTitle("可滚动内容")
	scroll.SetDescription("头部与页脚固定, 中间内容区超过可用高度后滚动。")
	scroll.BindTrigger(scrollTrigger)
	body := scroll.ScrollArea().ContentStack(12)
	for i := 0; i < 10; i++ {
		l := ui.NewLabel(0, 0, 0, 100, fmt.Sprintf(
			"%d. 这是一段用于演示滚动的长文本。对话框的面板高度由头部、内容与页脚共同决定, "+
				"内容超过 max-h-[70vh] 时保持头部和页脚不动, 只让中间区域滚动。", i+1),
			body.Handle).SetMuted().SetWrap()
		xui.ItemFill(l)
	}
	scroll.AddFooter("关闭", ui.ButtonOutline, func() { scroll.Close() })
}

// buildAlertDialogs 四个 AlertDialog 示例: 默认 / 小尺寸 / 带媒体块 / 破坏性。
func buildAlertDialogs(hWindow int, defaultTrigger, smallTrigger, mediaTrigger, destructiveTrigger *ui.Button) {
	basic := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogTitle("Are you absolutely sure?"),
		ui.WithAlertDialogDescription("This action cannot be undone. This will permanently delete your account and remove your data from our servers."))
	basic.AddCancel("Cancel", nil)
	basic.AddAction("Continue", ui.ButtonPrimary, nil)
	basic.BindTrigger(defaultTrigger)

	small := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogSize(ui.AlertDialogSizeSmall),
		ui.WithAlertDialogTitle("Allow accessory to connect?"),
		ui.WithAlertDialogDescription("Do you want to allow the USB accessory to connect to this device?"))
	small.AddCancel("Don't allow", nil)
	small.AddAction("Allow", ui.ButtonPrimary, nil)
	small.BindTrigger(smallTrigger)

	media := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogMedia(icons.IconBluetooth),
		ui.WithAlertDialogTitle("Are you absolutely sure?"),
		ui.WithAlertDialogDescription("This will permanently delete your account and remove your data from our servers."))
	media.AddCancel("Cancel", nil)
	media.AddAction("Continue", ui.ButtonPrimary, nil)
	media.BindTrigger(mediaTrigger)

	destructive := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogSize(ui.AlertDialogSizeSmall),
		ui.WithAlertDialogMedia(icons.IconTrash2),
		ui.WithAlertDialogMediaTone(ui.AlertDialogMediaDestructive),
		ui.WithAlertDialogTitle("Delete chat?"),
		ui.WithAlertDialogDescription("This will permanently delete this chat conversation. You can delete any memories saved during this chat in Settings."))
	destructive.AddCancel("Cancel", nil) // AlertDialogCancel 默认就是 outline(有边框)样式
	destructive.AddAction("Delete", ui.ButtonDestructive, nil)
	destructive.BindTrigger(destructiveTrigger)
}

// buildDialogVariants 无右上角关闭钮的对话框 + 嵌套模态示例(对话框里再开 AlertDialog)。
func buildDialogVariants(hWindow int, noCloseTrigger, nestedTrigger *ui.Button) {
	noClose := ui.NewDialog(hWindow, ui.WithDialogCloseButton(false))
	noClose.SetTitle("No Close Button")
	noClose.SetDescription("This dialog doesn't have a close button in the top-right corner.")
	noClose.BindTrigger(noCloseTrigger)
	noClose.AddFooter("Close", ui.ButtonOutline, func() { noClose.Close() })

	// 嵌套: 下层 Dialog 开着一层 AlertDialog。模态栈保证只有栈顶响应 ESC/Tab,
	// 关闭上层后焦点回到下层对话框。
	host := ui.NewDialog(hWindow)
	host.SetTitle("Alert Dialog Example")
	host.SetDescription("Click the button below to open an alert dialog.")
	host.BindTrigger(nestedTrigger)
	nested := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogSize(ui.AlertDialogSizeSmall),
		ui.WithAlertDialogTitle("Are you absolutely sure?"),
		ui.WithAlertDialogDescription("This action cannot be undone. This will permanently delete your account and remove your data from our servers."))
	nested.AddCancel("Cancel", nil)
	nested.AddAction("Continue", ui.ButtonPrimary, nil)
	openNested := ui.NewButton(0, 8, host.ContentWidth(), 0, "Open Alert Dialog", host.Content())
	openNested.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		nested.Show()
		return 0
	})
	host.AddFooter("Close", ui.ButtonOutline, func() { host.Close() })
}
