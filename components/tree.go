package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 树几何常量。shadcn 没有官方 tree 组件(社区实现通行的语义), 令牌取值
// 对齐本项目已验收组件: 行高 h-8(32) 同侧栏菜单项, 圆角 rounded-md(6) 同菜单条目,
// 悬浮/选中 bg-accent 同 select/menu 条目, 每级缩进 16(size-4), chevron size-4。
const (
	treeRowH     int32 = 32 // h-8
	treeIndent   int32 = 16 // 每级缩进 size-4
	treeChevron  int32 = 16 // chevron 图标 size-4
	treeRowInset int32 = 4  // 行视觉矩形左右内缩(mx-1), 命中区域仍是整行
	treeRowPad   int32 = 4  // 行视觉矩形到缩进槽的起点(原生内容区 padding, 探针实测 ≈4)
	treeGap      int32 = 4  // chevron 与文字的间距(gap-1)

	// 自定义项模板: 行高 32(与 SetItemHeightDefault 一致), chevron 槽 16×16,
	// 文字间距 4。展开按钮只是行为载体 —— 其原生 "+/-" 图形在模板创建后用透明
	// 填充压掉(ClearBkInfo 会回落到系统默认内容, 必须再 AddBkFill 透明色盖住),
	// 视觉由 DrawItem 回调里的 chevron 接管。
	treeTemplateXML = `<?xml version="1.0" encoding="UTF-8"?>
<head />
<xc_template_tree>
	<layoutEleUI widthT="300" heightT="32" layout.horizon="true" layout.alignH="left" layout.alignV="center" layout.width="auto" layout.height="fill" layout.space="4">
		<layoutEleUI layout.alignV="center" layout.width="16" layout.height="16" itemID="0">
			<buttonUI itemID="1" transparent="true" rect="0,0,16,16" expandButton="true" buttonType="check" />
		</layoutEleUI>
		<shapeText itemID="2" layout.width="auto" layout.height="fill" content="{binding field=name1}" />
	</layoutEleUI>
</xc_template_tree>`
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

	onSel func(id int32)
	onBg  bool // 宿主底色跟随 Background(默认按"放在 Card 上"约定跟随 Card)
}

// TreeOption 组件配置项。
type TreeOption func(*Tree)

// WithTreeOnBackground 让滚动条等底色跟随 Background 令牌(默认跟随 Card)。
// 组件默认按"放在 Card 上"设计(同 Table/ComboBox); 直接放在窗口背景上时用本选项。
func WithTreeOnBackground() TreeOption {
	return func(t *Tree) { t.onBg = true }
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

	// 模板与默认行高都必须在适配器/数据之前设置(探针实测: 之后设置不生效)
	t.SetItemTemplateXMLFromString(treeTemplateXML)
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

	// 滚动条: 细条 + 内容不溢出时隐藏
	t.SetScrollBarSize(8)
	t.EnableAutoShowScrollBar(true)
	t.ShowSBarH(false)

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

	t.applyTheme(xui.Current())
	xui.OnThemeChange(func(th *xui.Theme) { t.applyTheme(th) })
	xui.AttachWindow(t.Handle)
	return t
}

// ===== 数据 API =====

// AddRoot 添加根节点, 返回项 ID。
func (t *Tree) AddRoot(text string) int32 {
	return t.InsertItemText(text, xcc.XC_ID_ROOT, xcc.XC_ID_LAST)
}

// AddChild 给 parent 添加子节点, 返回项 ID。
func (t *Tree) AddChild(parent int32, text string) int32 {
	return t.InsertItemText(text, parent, xcc.XC_ID_LAST)
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
	t.Tree.DeleteItem(id)
	t.Redraw(false)
	return t
}

// Clear 清空全部项。
func (t *Tree) Clear() *Tree {
	t.DeleteItemAll()
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
	switch pItem.NState {
	case xcc.Tree_Item_State_Stay:
		p.FillRound(&vis, th.Accent, 6) // hover:bg-accent
	case xcc.Tree_Item_State_Select:
		p.FillRound(&vis, th.Accent, 6) // data-selected:bg-accent
	}

	// 祖先层级缩进参考线: 每级槽位居中 1px(border 色)
	for i := int32(0); i < pItem.NDepth; i++ {
		gx := rc.Left + treeRowInset + treeRowPad + i*treeIndent + treeIndent/2
		p.Line(gx, vis.Top, gx, vis.Bottom, th.Border, 1)
	}

	// chevron: 有子项才画, 收起朝右/展开朝下; 叶节点留空槽位(文字全树对齐)
	if t.HasChildren(pItem.NID) {
		box := xc.RECT{
			Left:   rc.Left + treeRowInset + treeRowPad + pItem.NDepth*treeIndent,
			Top:    rc.Top + (pItem.NHeight-treeChevron)/2,
			Right:  rc.Left + treeRowInset + treeRowPad + pItem.NDepth*treeIndent + treeChevron,
			Bottom: rc.Top + (pItem.NHeight-treeChevron)/2 + treeChevron,
		}
		path := xui.IconChevronRight
		if pItem.BExpand {
			path = xui.IconChevronDown
		}
		p.Icon(path, &box, th.MutedFore, 2)
	}
	return 0
}

// HasChildren 返回项是否有子项(决定 chevron 的显隐)。
func (t *Tree) HasChildren(id int32) bool {
	return t.GetFirstChildItem(id) != xcc.XC_ID_ERROR
}

// styleScrollBars 把纵向滚动条收成细条 + 主题色(同 Table 做法)。
func (t *Tree) styleScrollBars(th *xui.Theme) {
	sb := t.GetScrollBarVObj()
	if sb == nil || sb.Handle <= 0 {
		return
	}
	sb.ShowButton(false)
	sb.SetSliderPadding(2)
	sb.EnableBkTransparent(true)
	sb.ClearBkInfo()
	sb.AddBkFill(xcc.Element_State_Flag_Leave, uint32(t.surface(th)))
	sb.AddBkFill(xcc.Element_State_Flag_Stay, uint32(t.surface(th)))
	if sl := sb.GetButtonSliderObj(); sl != nil && sl.Handle > 0 {
		sl.ClearBkInfo()
		sl.AddBkFill(xcc.Element_State_Flag_Leave, uint32(th.Border))
		sl.AddBkFill(xcc.Element_State_Flag_Stay, uint32(th.MutedFore))
		sl.AddBkFill(xcc.Element_State_Flag_Down, uint32(th.MutedFore))
	}
}
