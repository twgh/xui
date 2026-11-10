package ui

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 树几何常量。shadcn 没有官方 tree 组件(社区实现通行的语义), 令牌取值
// 对齐本项目已验收组件: 行高 h-8(32) 同侧栏菜单项, 圆角 rounded-md(6) 同菜单条目,
// 悬浮/选中 bg-accent 同 select/menu 条目, 每级缩进 16(size-4), chevron size-4。
const (
	treeRowH     int32 = 32 // h-8
	treeIndent   int32 = 16 // 每级缩进 size-4
	treeRowInset int32 = 4  // 行视觉矩形左右内缩(mx-1), 命中区域仍是整行
	treeChevronW int32 = 6  // chevron 字形横向跨度(收起朝右为 W×H; lucide 比例的 1.25 倍)
	treeChevronH int32 = 10 // chevron 字形纵向跨度(展开朝下旋转 90°, 宽高互换为 H×W)
	treeBtnInset int32 = 1  // 原生展开按钮左缘相对 RcItem.Left 的偏移(布局边框实测 ≈1.3)

	// 勾选模式: chevron 槽(16)后再留 4 间距, 复选框在模板里只占位(16), 由 DrawItem 自绘。
	treeCheckSize   int32 = 16 // size-4
	treeCheckGap    int32 = 4  // 模板 layout.space
	treeChevronSlot int32 = 16

	// treeWakeTimerID 是"首次绘制后唤醒"占用的炫彩定时器 ID(与运行时动画定时器错开)。
	// 用途: 原生 Tree 的项模板是在**布局排版**(元素处于 show 状态时)那一步实例化的 —— 树建在
	// 隐藏的祖先容器里(多页签页面/对话框)、且尺寸只由创建参数给定时, 全程没有任何一次"可见时的
	// 排版"落在它身上, 首次显示就只有行底/缩进线/chevron(那些是我们自绘的), **全树文字空白**;
	// 绘制本身不会补上模板实例(实测, 见 DESIGN 第八节 51)。首次绘制之后补一次 RefreshData 即可
	// 拿到正确文字。延时 1ms 只是"下一帧", 让刷新发生在绘制回调之外(xcgui 定时器由消息循环驱动,
	// 不会重入绘制)。
	treeWakeTimerID uint32 = 0x7874

	// 自定义项模板: 行高 32(与 SetItemHeightDefault 一致), chevron 槽 16×16, 文字间距 4。
	// 展开按钮只是行为载体(点击热区)——其原生 "+/-" 图形在模板创建后用透明填充压掉
	// (ClearBkInfo 会回落到系统默认内容, 必须再 AddBkFill 透明色盖住), 视觉由
	// DrawItem 回调里居中于按钮矩形的折线 chevron 接管。
	treeTemplateXML = `<?xml version="1.0" encoding="UTF-8"?>
<head />
<xc_template_tree>
	<layoutEleUI widthT="300" heightT="32" layout.horizon="true" layout.alignH="left" layout.alignV="center" layout.width="auto" layout.height="fill" layout.space="4">
		<layoutEleUI layout.alignV="center" layout.width="16" layout.height="16" itemID="0">
			<buttonUI itemID="1" transparent="true" rect="0,0,16,16" expandButton="true" buttonType="check" />
		</layoutEleUI>
		%s
		<shapeText itemID="2" layout.width="auto" layout.height="fill" content="{binding field=name1}" />
	</layoutEleUI>
</xc_template_tree>`

	// 勾选模式的复选框占位(自绘在 DrawItem 里, 模板只要把文字推到它右边)。
	treeCheckSlotXML = `<layoutEleUI layout.alignV="center" layout.width="16" layout.height="16" itemID="3" />`
)

// Tree 是 shadcn 风格的树形视图: 缩进参考线 + 旋转 chevron + 行悬浮/选中。
//
// 载体是**原生 Tree**(展开/收起、选中、滚动、点击 chevron 展开全部原生行为),
// 外观由自绘接管: 原生项背景、连接线、分割线、展开按钮 "+/-" 图形全部压掉。
//
// 与 shadcn 语义的对应(社区通行实现, 无官方 tree):
//   - 行高 32, 视觉矩形左右内缩 4, 圆角 6;
//   - 行悬浮 bg-accent, 选中 bg-accent(同 select/menu 条目);
//   - 每级缩进 16, 祖先层级画 1px 缩进参考线(border 色);
//   - chevron size-4 muted-foreground, 收起朝右 / 展开朝下(即 rotate-90 的两态),
//     叶节点不画但保留槽位, 文字全树对齐;
//   - 条目文字 text-sm foreground。
//
// 项 ID 是树内部的句柄(非行号), 由 AddRoot/AddChild 返回, 供 Select/Expand 等使用。
type Tree struct {
	*widget.Tree

	onSel     func(id int32)
	onCheck   func(id int32, checked bool)
	checkable bool
	check     map[int32]int8 // 项 ID -> treeCheck*(父节点由直接子项聚合, 见 combineTreeCheck)
	onBg      bool           // 宿主底色跟随 Background(默认按"放在 Card 上"约定跟随 Card)
	woken     bool           // 是否已排过"首次绘制后唤醒"(见 treeWakeTimerID)
}

const (
	treeCheckNone int8 = iota
	treeCheckChecked
	treeCheckPartial // 父节点半选: 部分后代勾选
)

// TreeOption 组件配置项。
type TreeOption func(*Tree)

// WithTreeOnBackground 让滚动条等底色跟随 Background 令牌(默认跟随 Card)。
// 组件默认按"放在 Card 上"设计(同 Table/NativeComboBox); 直接放在窗口背景上时用本选项。
func WithTreeOnBackground() TreeOption {
	return func(t *Tree) { t.onBg = true }
}

// WithTreeCheckable 开启勾选模式: 每项 chevron 后多一个复选框, 父节点随后代
// 自动落到全选 / 半选 / 未选。模板必须在建树前确定, 所以只能作为构造选项。
func WithTreeCheckable() TreeOption {
	return func(t *Tree) { t.checkable = true }
}

// NewTree 创建树形视图。cy 传 0 时使用 5 行的默认高度。
func NewTree(x, y, cx, cy int32, hParent int, opts ...TreeOption) *Tree {
	t := &Tree{Tree: widget.NewTree(x, y, cx, cy, hParent)}
	if t.Tree == nil {
		return nil
	}
	for _, o := range opts {
		o(t)
	}
	if cy <= 0 {
		t.SetHeight(treeRowH*5 + 2)
	}
	t.check = map[int32]int8{}

	// 模板与默认行高都必须在适配器/数据之前设置(探针实测: 之后设置不生效)
	checkSlot := ""
	if t.checkable {
		checkSlot = treeCheckSlotXML
	}
	t.SetItemTemplateXMLFromString(fmt.Sprintf(treeTemplateXML, checkSlot))
	t.SetItemHeightDefault(treeRowH, treeRowH)
	t.CreateAdapter()

	// 原生皮肤全压掉: 项背景/连接线/分割线交给自绘, 树体真透明(露出宿主底色)
	t.SetDrawItemBkFlags(xcc.List_DrawItemBk_Flag_Nothing)
	t.EnableConnectLine(false, false)
	t.SetSplitLineColor(0)
	t.EnableBkTransparent(true)
	t.EnableDrawBorder(false)
	t.EnableDrawFocus(false)
	t.SetRowSpace(0)
	t.SetIndentation(treeIndent)

	// 滚动条: 细条(shadcn 规格, 见 scrollbar.go) + 内容不溢出时隐藏
	t.SetScrollBarSize(scrollBarW)
	t.EnableAutoShowScrollBar(true)
	t.ShowSBarH(false)
	showScrollBarOnHover(t.Handle, t.GetScrollBarVObj())

	if font := xui.FontHandle(xui.Current().FontDefault); font != 0 {
		t.SetFont(font)
	}

	t.AddEvent_Tree_Temp_Create_End(t.onTempCreate)
	t.AddEvent_Tree_DrawItem(t.onDrawItem)
	t.AddEvent_Tree_Select(func(hEle int, nItemID int32, pbHandled *bool) int {
		if t.onSel != nil {
			t.onSel(nItemID)
		}
		return 0
	})
	if t.checkable {
		// 复选框命中自己做: 吞掉按下消息, 原生单选不会跟着变(同 DataTable 勾选行, 见 DESIGN 8.170)
		t.AddEvent_LButtonDown(t.onCheckDown)
		// 双击落在复选框上也要吞: 原生 Tree 会把双击展开/收起; 双击按两次点击处理,
		// 第二次继续切勾选(见 onCheckDblClick)
		t.AddEvent_LButtonDBClick(t.onCheckDblClick)
	}

	// 首次绘制后补一次"唤醒"刷新(见 treeWakeTimerID 注释)
	t.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		if !t.woken {
			t.woken = true
			xc.XEle_SetXCTimer(hEle, treeWakeTimerID, 1)
		}
		return 0
	})
	t.AddEvent_XC_Timer(func(hEle, nID int, pbHandled *bool) int {
		if nID == int(treeWakeTimerID) {
			xc.XEle_KillXCTimer(hEle, treeWakeTimerID)
			t.RefreshData()
			t.Redraw(false)
		}
		return 0
	})

	t.applyTheme(xui.Current())
	xui.OnThemeChangeFor(t.Handle, func(th *xui.Theme) { t.applyTheme(th) })
	xui.AttachWindow(t.Handle)
	return t
}

// ===== 数据 API =====

// AddRoot 添加根节点, 返回项 ID。
func (t *Tree) AddRoot(text string) int32 {
	id := t.InsertItemText(text, xcc.XC_ID_ROOT, xcc.XC_ID_LAST)
	if t.checkable {
		t.check[id] = treeCheckNone
	}
	return id
}

// AddChild 给 parent 添加子节点, 返回项 ID。
func (t *Tree) AddChild(parent int32, text string) int32 {
	id := t.InsertItemText(text, parent, xcc.XC_ID_LAST)
	if t.checkable {
		t.check[id] = treeCheckNone
		t.refreshCheckAncestors(parent)
	}
	return id
}

// ItemText 读取项文本。
func (t *Tree) ItemText(id int32) string { return t.GetItemText(id, 0) }

// SetItemText 更新项文本。
func (t *Tree) SetItemText(id int32, text string) *Tree {
	t.Tree.SetItemText(id, 0, text)
	t.RefreshItem(id)
	t.Redraw(false)
	return t
}

// DeleteItem 删除项(含子项)。
func (t *Tree) DeleteItem(id int32) *Tree {
	parent := xcc.XC_ID_ERROR
	if t.checkable {
		parent = t.GetParentItem(id)
		t.dropCheckSubtree(id)
	}
	t.Tree.DeleteItem(id)
	if t.checkable {
		t.refreshCheckAncestors(parent)
	}
	t.Redraw(false)
	return t
}

// Clear 清空全部项。
func (t *Tree) Clear() *Tree {
	t.DeleteItemAll()
	if t.checkable {
		t.check = map[int32]int8{}
	}
	t.Redraw(false)
	return t
}

// Expand 展开/收起项。
func (t *Tree) Expand(id int32, on bool) *Tree {
	t.ExpandItem(id, on)
	t.Redraw(false)
	return t
}

// ExpandAll 展开(或收起)某项及其全部后代。
func (t *Tree) ExpandAll(id int32, on bool) *Tree {
	t.ExpandAllChildItem(id, on)
	t.Redraw(false)
	return t
}

// IsExpand 返回项是否展开。
func (t *Tree) IsExpand(id int32) bool { return t.Tree.IsExpand(id) }

// Selected 返回选中项 ID, 未选中返回 XC_ID_ERROR。
func (t *Tree) Selected() int32 { return t.GetSelectItem() }

// Select 选中指定项并滚动到可见。
func (t *Tree) Select(id int32) *Tree {
	t.SetSelectItem(id)
	t.VisibleItem(id)
	t.Redraw(false)
	return t
}

// OnSelect 注册项选中回调(参数为项 ID)。
func (t *Tree) OnSelect(fn func(id int32)) *Tree {
	t.onSel = fn
	return t
}

// ===== 勾选(WithTreeCheckable) =====

// Checked 报告项是否被勾选(半选的父节点返回 false)。
func (t *Tree) Checked(id int32) bool {
	return t.checkable && t.check[id] == treeCheckChecked
}

// SetChecked 勾选 / 取消勾选某项: 父节点会级联到全部后代, 祖先随后代自动重算。
// 状态真的改变时触发 OnCheckChange。
func (t *Tree) SetChecked(id int32, on bool) *Tree {
	if !t.checkable || id == xcc.XC_ID_ERROR {
		return t
	}
	target := treeCheckNone
	if on {
		target = treeCheckChecked
	}
	if t.setCheckSubtree(id, target) {
		t.refreshCheckAncestors(t.GetParentItem(id))
		t.Redraw(false)
		if t.onCheck != nil {
			t.onCheck(id, on)
		}
	}
	return t
}

// CheckedKeys 返回全部勾选项的 ID 快照(树的前序; 全选的父节点与后代都包含)。
func (t *Tree) CheckedKeys() []int32 {
	if !t.checkable {
		return nil
	}
	out := make([]int32, 0, len(t.check))
	var walk func(id int32)
	walk = func(id int32) {
		if t.check[id] == treeCheckChecked {
			out = append(out, id)
		}
		for c := t.GetFirstChildItem(id); c != xcc.XC_ID_ERROR; c = t.GetNextSiblingItem(c) {
			walk(c)
		}
	}
	for r := t.GetFirstChildItem(xcc.XC_ID_ROOT); r != xcc.XC_ID_ERROR; r = t.GetNextSiblingItem(r) {
		walk(r)
	}
	return out
}

// OnCheckChange 注册勾选变化回调: id 是被操作项, checked 是它变化后的状态。
// 父节点级联导致的后代变化不逐条回调, 需要完整集合请调 CheckedKeys()。
func (t *Tree) OnCheckChange(fn func(id int32, checked bool)) *Tree {
	t.onCheck = fn
	return t
}

// checkHitPoint 返回客户区坐标命中的复选框项 ID; 不在复选框上时 ok=false。
func (t *Tree) checkHitPoint(pPt *xc.POINT) (int32, bool) {
	if !t.checkable || pPt == nil {
		return xcc.XC_ID_ERROR, false
	}
	id := t.HitTestOffset(pPt)
	if id == xcc.XC_ID_ERROR || !treeCheckHit(t.itemDepth(id), pPt.X) {
		return id, false
	}
	return id, true
}

// onCheckDown 命中复选框时切勾选并吞掉按下消息, 原生单选保持不变。
func (t *Tree) onCheckDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	id, ok := t.checkHitPoint(pPt)
	if !ok {
		return 0
	}
	*pbHandled = true
	t.toggleChecked(id)
	return 0
}

// onCheckDblClick 处理复选框上的双击: 双击 = 两次点击, 第二次也要切勾选;
// 同时吞掉消息, 否则原生 Tree 会把这次双击当成展开/收起。
func (t *Tree) onCheckDblClick(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	id, ok := t.checkHitPoint(pPt)
	if !ok {
		return 0
	}
	*pbHandled = true
	t.toggleChecked(id)
	return 0
}

// toggleChecked 按当前状态取反(半选父节点视作未全选, 点一下落到全选)。
func (t *Tree) toggleChecked(id int32) {
	t.SetChecked(id, t.check[id] != treeCheckChecked)
}

// itemDepth 沿父链算层级(点复选框时只需要横向位置, 竖向整行都可命中)。
func (t *Tree) itemDepth(id int32) int32 {
	d := int32(0)
	for p := t.GetParentItem(id); p != xcc.XC_ID_ERROR && p != xcc.XC_ID_ROOT; p = t.GetParentItem(p) {
		d++
	}
	return d
}

// treeCheckLeft 返回复选框相对项矩形左缘的横坐标: 原生展开按钮槽(16)+间距(4)。
func treeCheckLeft(depth int32) int32 {
	return treeBtnInset + depth*treeIndent + treeChevronSlot + treeCheckGap
}

// treeCheckHit 判断客户区横坐标 x 是否落在该层级的复选框上。
// 左右各放 2px 容差: RcItem.Left 实测 1px 内缩, 不放容差时右半边可能点不中。
func treeCheckHit(depth, x int32) bool {
	left := treeCheckLeft(depth)
	return x >= left-2 && x < left+treeCheckSize+2
}

// setCheckSubtree 把某项与全部后代设成 target, 返回是否有状态变化。
func (t *Tree) setCheckSubtree(id int32, target int8) bool {
	changed := t.check[id] != target
	t.check[id] = target
	for c := t.GetFirstChildItem(id); c != xcc.XC_ID_ERROR; c = t.GetNextSiblingItem(c) {
		if t.setCheckSubtree(c, target) {
			changed = true
		}
	}
	return changed
}

// refreshCheckAncestors 从 id 开始沿父链重算聚合状态(父节点 = 直接子项状态的汇总)。
func (t *Tree) refreshCheckAncestors(id int32) {
	for p := id; p != xcc.XC_ID_ERROR && p != xcc.XC_ID_ROOT; p = t.GetParentItem(p) {
		t.check[p] = t.combineCheckChildren(p)
	}
}

// combineCheckChildren 汇总某项直接子项的勾选状态。
func (t *Tree) combineCheckChildren(id int32) int8 {
	var states []int8
	for c := t.GetFirstChildItem(id); c != xcc.XC_ID_ERROR; c = t.GetNextSiblingItem(c) {
		states = append(states, t.check[c])
	}
	return combineTreeCheck(states)
}

// combineTreeCheck 把一组直接子项状态聚合成父项状态: 全勾 = 全选, 有勾/有半选 = 半选。
// 空组(没有子项)视为未选 —— 叶子节点的状态由 setCheckSubtree 直接维护。
func combineTreeCheck(states []int8) int8 {
	if len(states) == 0 {
		return treeCheckNone
	}
	checked, partial := 0, 0
	for _, st := range states {
		switch st {
		case treeCheckChecked:
			checked++
		case treeCheckPartial:
			partial++
		}
	}
	if partial > 0 || (checked > 0 && checked < len(states)) {
		return treeCheckPartial
	}
	if checked == len(states) {
		return treeCheckChecked
	}
	return treeCheckNone
}

// dropCheckSubtree 在删除项之前摘掉它和全部后代的状态。
func (t *Tree) dropCheckSubtree(id int32) {
	for c := t.GetFirstChildItem(id); c != xcc.XC_ID_ERROR; c = t.GetNextSiblingItem(c) {
		t.dropCheckSubtree(c)
	}
	delete(t.check, id)
}

// ===== 主题 / 绘制 =====

// surface 返回滚动条轨道底色令牌(同 Table: 默认跟随 Card)。
func (t *Tree) surface(th *xui.Theme) xui.Color {
	if t.onBg {
		return th.Background
	}
	return th.Card
}

// applyTheme 只刷新"必须同步给原生对象"的颜色与字体: 项模板文字 + 滚动条。
// 行底、参考线、chevron 都在绘制回调里实时取令牌。文字颜色挂在模板实例上,
// 无法遍历设置(项是 NID 寻址), 用 RefreshData 重走一遍模板创建事件统一同步。
func (t *Tree) applyTheme(th *xui.Theme) {
	if !xc.XC_IsHXCGUI(t.Handle, xcc.XC_ELE) {
		return // 元素已销毁(窗口关闭), 句柄失效
	}
	if font := xui.FontHandle(th.FontDefault); font != 0 {
		t.SetFont(font)
	}
	t.styleScrollBars(th)
	t.RefreshData()
	t.Redraw(false)
}

// onTempCreate 项模板实例化/复用时: 压掉展开按钮的原生 "+/-" 图形 + 同步文字样式。
// 复用(nFlag=2)也要重设 —— 实例可能来自旧主题。
func (t *Tree) onTempCreate(hEle int, pItem *xc.Tree_Item_, nFlag int32, pbHandled *bool) int {
	if pItem == nil {
		return 0
	}
	// 展开按钮只保留行为: 清背景(回落系统默认)后用透明填充盖住, "+/-" 图形即消失
	if hbtn := t.GetTemplateObject(pItem.NID, 1); hbtn > 0 {
		if b := widget.NewButtonByHandle(hbtn); b != nil {
			b.ClearBkInfo()
			tr := uint32(xc.RGBA(0, 0, 0, 0))
			b.AddBkFill(xcc.Element_State_Flag_Leave, tr)
			b.AddBkFill(xcc.Element_State_Flag_Stay, tr)
			b.AddBkFill(xcc.Element_State_Flag_Down, tr)
			b.AddBkFill(xcc.Element_State_Flag_Disable, tr)
		}
	}
	th := xui.Current()
	if h := t.GetTemplateObject(pItem.NID, 2); h > 0 {
		xc.XShapeText_SetTextColor(h, uint32(th.Foreground))
		if font := xui.FontHandle(th.FontDefault); font != 0 {
			xc.XShapeText_SetFont(h, font)
		}
	}
	return 0
}

// onDrawItem 自绘项: 行底色(悬浮/选中) + 祖先缩进参考线 + chevron。
// 绘制在模板子元素之前, 文字由模板的 ShapeText 自己画(透明填充的展开按钮不遮挡)。
func (t *Tree) onDrawItem(hEle, hDraw int, pItem *xc.Tree_Item_, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil || pItem == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()
	rc := pItem.RcItem

	// 行视觉矩形: 左右各内缩 4, 高度整行
	vis := xc.RECT{
		Left:   rc.Left + treeRowInset,
		Top:    rc.Top,
		Right:  rc.Right - treeRowInset,
		Bottom: rc.Bottom,
	}
	// 行底: 选中立即到位, 悬停从宿主底色淡入 bg-accent(行本身透出 Card, 见 surface)
	if pItem.NState == xcc.Tree_Item_State_Select {
		p.FillRound(&vis, th.Accent, 6) // data-selected:bg-accent
	} else if hv := xui.HoverItem(hEle, pItem.NID, pItem.NState == xcc.Tree_Item_State_Stay); hv > 0 {
		p.FillRound(&vis, xui.Fade(t.surface(th), th.Accent, hv), 6) // hover:bg-accent
	}

	// 祖先层级缩进参考线: 穿过各级展开按钮的中心(1px, border 色)
	for i := int32(0); i < pItem.NDepth; i++ {
		gx := rc.Left + treeBtnInset + i*treeIndent + treeIndent/2
		p.Line(gx, vis.Top, gx, vis.Bottom, th.Border, 1)
	}

	// chevron: 有子项才画, 收起朝右/展开朝下(lucide 6:12 比例的两段折线);
	// 中心对准原生展开按钮矩形 —— 按钮是点击热区, 图标偏出热区会点不到(实测踩坑);
	// 叶节点不画但保留槽位(文字全树对齐)
	if t.HasChildren(pItem.NID) {
		cx := rc.Left + treeBtnInset + pItem.NDepth*treeIndent + treeIndent/2
		cy := rc.Top + pItem.NHeight/2
		w2, h2 := treeChevronW/2, treeChevronH/2
		if pItem.BExpand {
			// ˅ = 收起字形顺时针转 90°, 宽高互换(宽 H、高 W), 否则会画成细长的竖箭头
			p.Line(cx-h2, cy-w2, cx, cy+w2, th.MutedForeground, 2) // ˅ 左斜边
			p.Line(cx, cy+w2, cx+h2, cy-w2, th.MutedForeground, 2) // ˅ 右斜边
		} else {
			p.Line(cx-w2, cy-h2, cx+w2, cy, th.MutedForeground, 2) // > 上斜边
			p.Line(cx+w2, cy, cx-w2, cy+h2, th.MutedForeground, 2) // > 下斜边
		}
	}

	// 复选框: 模板里留了 16px 槽位, 这里画; 状态由父/子聚合维护, 绘制时只查表
	if t.checkable {
		box := xc.RECT{
			Left:   rc.Left + treeCheckLeft(pItem.NDepth),
			Top:    rc.Top + (rc.Bottom-rc.Top-treeCheckSize)/2,
			Right:  rc.Left + treeCheckLeft(pItem.NDepth) + treeCheckSize,
			Bottom: rc.Top + (rc.Bottom-rc.Top-treeCheckSize)/2 + treeCheckSize,
		}
		switch t.check[pItem.NID] {
		case treeCheckChecked, treeCheckPartial:
			p.FillRound(&box, th.Primary, 4)
			ir := xc.RECT{Left: box.Left + 1, Top: box.Top + 1, Right: box.Right - 1, Bottom: box.Bottom - 1}
			path := icons.IconCheck
			if t.check[pItem.NID] == treeCheckPartial {
				path = icons.IconMinus
			}
			p.Icon(path, &ir, th.PrimaryForeground, 2)
		default:
			if th.IsDark() {
				p.FillRound(&box, th.Input.ScaleAlpha(0.3), 4)
			}
			p.StrokeRound(&box, th.Input, 4, 1)
		}
	}
	return 0
}

// HasChildren 返回项是否有子项(决定 chevron 的显隐)。
func (t *Tree) HasChildren(id int32) bool {
	return t.GetFirstChildItem(id) != xcc.XC_ID_ERROR
}

// styleScrollBars 把纵向滚动条收成细条 + 主题色(同 Table 做法, 几何/圆角/配色见 scrollbar.go)。
func (t *Tree) styleScrollBars(th *xui.Theme) {
	styleScrollBar(t.GetScrollBarVObj(), uint32(t.surface(th)), embedScrollBarSkin(th))
}
