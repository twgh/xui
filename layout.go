package xui

import (
	"github.com/twgh/xcgui/objectbase"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// ===== 布局项声明(LayoutEle 排版用) =====
//
// xcgui 的布局项每条轴都有一个尺寸类型(Fixed / Fill / Auto / Weight / Percent / Disable),
// 默认值是 **Layout_Size_Disable —— "这条轴布局不接管, 用元素自己的尺寸"**(2026-10-08 探针
// 实测: 完全不声明时元素按自身尺寸参与排版, 位置/居中/堆叠全部正确)。
//
// 而组件构造函数本来就把设计稿尺寸建在了元素上(Label 24 / Checkbox·Radio 24 / Badge 20 /
// Input 32 / Pagination 33 / Separator 1 / Progress 4 / Button 视觉高 + 1px 按压预留…),
// 所以"固定尺寸的元素"根本不需要声明尺寸: 声明只是把元素尺寸抄了一遍, 抄错就是 bug ——
// 标题栏按钮曾写 `itemFixed(btn, capBtnW, capBtnH)`, Fixed 把元素高从 29 压成 28,
// 壳体(`p.Rc` 去掉 1px 预留)因此矮 1px, 且与别处同款按钮不一致(见 DESIGN 第八节 29)。
//
// 约定: **只声明你要改变的那条轴**。
//
//	xui.ItemFill(lbl)                  // 宽铺满, 高用元素自身(Label 默认 24)
//	xui.ItemWidth(search, 240)         // 宽锁 240, 高用元素自身(Input 默认 32)
//	xui.ItemWeight(info, 1)            // 宽按权重, 高用元素自身
//	xui.ItemSize(box, 300, 200)        // 宽高都锁; 某一轴不想锁就传 xui.ItemKeep
//	xui.ItemMarginTop(lbl, 8)          // 布局项外边距(与 CSS margin 同义, 各方向独立)
//	// 元素尺寸本身就不对时(如多行说明文字要 48 高), 改**创建参数**:
//	// ui.NewLabel(0, 0, 0, 48, text, parent) + xui.ItemFill(desc)
//
// 什么时候还要写常量: 设计稿尺寸本身就是契约(按钮 96 宽、下拉框 180 宽、表格 180 高)。
// 这类常量的来源必须是设计稿, 而不是"元素当前多大"。
//
// 需要覆盖的那条轴走 Fill/Auto/Percent 等语义时, 直接用 xcgui 的 LayoutItem_SetWidth/Height;
// 项级 `LayoutItem_SetAlign` 实测无效(交叉轴对齐只能用布局级 SetAlignH/V), 故不在这里包装。

// ItemKeep 是 ItemSize 里表示"这条轴不覆盖"的哨兵(用元素自身尺寸)。
//
// 用 -1 而不是 0: 0 是**合法尺寸**(0 宽/高的元素会被排成不可见), 用户可能真要设 0。
const ItemKeep int32 = -1

// Item 是"能当布局项用"的元素: 所有 widget 都提升自 objectbase.Widget, 自带 LayoutItem_* 包装。
type Item interface {
	LayoutItem_SetWidth(nType xcc.Layout_Size_, nWidth int32) *objectbase.Widget
	LayoutItem_SetHeight(nType xcc.Layout_Size_, nHeight int32) *objectbase.Widget
	LayoutItem_GetMargin(pMargin *xc.RECT) *objectbase.Widget
	LayoutItem_SetMargin(left, top, right, bottom int32) *objectbase.Widget
	LayoutItem_SetMinSize(width, height int32) *objectbase.Widget
	LayoutItem_SetPosition(left, top, right, bottom int32) *objectbase.Widget
	LayoutItem_EnableFloat(bFloat ...bool) *objectbase.Widget
	LayoutItem_EnableWrap(bWrap ...bool) *objectbase.Widget
	LayoutItem_EnableSwap(bEnable ...bool) *objectbase.Widget
}

// ===== 尺寸 =====

// ItemWidth 锁定布局项宽度为固定像素, 高度不覆盖(用元素自身尺寸)。
//
// 用于设计稿常量宽: 等宽按钮行(`xui.ItemWidth(btn, 96)`)、定宽下拉框/搜索框、定宽表格。
// 常量要来自设计稿, 不要写成"元素当前多宽"。
func ItemWidth(item Item, w int32) {
	item.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
}

// ItemHeight 锁定布局项高度为固定像素, 宽度不覆盖(用元素自身尺寸)。
func ItemHeight(item Item, h int32) {
	item.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
}

// ItemSize 锁定布局项的宽/高为固定像素; 某一轴传 ItemKeep 表示该轴不覆盖(用元素自身尺寸)。
//
//	xui.ItemSize(box, 300, 200)           // 两条轴都锁
//	xui.ItemSize(box, 300, xui.ItemKeep)  // 只锁宽(等价于 ItemWidth)
//
// 只锁一条轴时用 ItemWidth / ItemHeight 更清楚; ItemSize 是"两条轴一起传"的场合(值可能是
// 算出来的, 其中一条按条件才锁)。
func ItemSize(item Item, w, h int32) {
	if w != ItemKeep {
		item.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	}
	if h != ItemKeep {
		item.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	}
}

// ItemFill 让布局项宽度占满剩余空间(主轴 Fill), 高度不覆盖(用元素自身尺寸)。
//
// 注意 Fill 在主轴上会**独占一整行/列**, 兄弟项被挤到下一行; 行内等分剩余空间要用 ItemWeight。
func ItemFill(item Item) {
	item.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
}

// ItemWeight 让布局项宽度按权重分配剩余空间, 高度不覆盖(用元素自身尺寸)。
func ItemWeight(item Item, weight int32) {
	item.LayoutItem_SetWidth(xcc.Layout_Size_Weight, weight)
}

// ItemFillHeight 让布局项高度占满父级的**剩余空间**(纵向按权重分配), 宽度不覆盖。
//
// 注意与宽度方向的 ItemFill 语义不同(实测, 见 DESIGN 第八节 55):
//   - 宽度 `Fill` = "铺满父级整宽", 竖向链里这样用没问题;
//   - 高度若也用 `Fill`, 拿到的是**父级整高**(减自身边距), 不是剩余空间 ——
//     前面还有兄弟项(标题/说明)时就会溢出, 所以这里用 `Weight 1`。
//   - `Weight` 只在**父级高度确定**时才算得出剩余空间(窗口直属 / Fixed / Weight 父级都行);
//     父级是 Fill 或 Auto 时会被算成 0 —— 卡片高度是 Auto 时不要用本函数。
func ItemFillHeight(item Item) {
	item.LayoutItem_SetHeight(xcc.Layout_Size_Weight, 1)
}

// ItemMinSize 限制布局项在**缩放类尺寸**(Auto/Fill/Weight/Percent)下的最小宽高, 0 = 不限制。
//
// Fixed 轴不受它影响(固定值就是固定值); 典型用法: Fill 铺满但有下限。
func ItemMinSize(item Item, w, h int32) {
	item.LayoutItem_SetMinSize(w, h)
}

// ===== 外边距 =====

// ItemMargin 设置布局项四边外边距(与 CSS margin 同义)。
func ItemMargin(item Item, left, top, right, bottom int32) {
	item.LayoutItem_SetMargin(left, top, right, bottom)
}

// margin 是四边外边距的读改写的公共实现: 只替换要改的那一边, 其余三边保持原值
// (所以 ItemMarginTop 之后还能再 ItemMarginBottom, 两者不会互相清掉)。
func margin(item Item, set func(*xc.RECT)) {
	var rc xc.RECT
	item.LayoutItem_GetMargin(&rc)
	set(&rc)
	item.LayoutItem_SetMargin(rc.Left, rc.Top, rc.Right, rc.Bottom)
}

// ItemMarginTop 设置布局项上边距, 其余三边保持不变。
func ItemMarginTop(item Item, m int32) {
	margin(item, func(rc *xc.RECT) { rc.Top = m })
}

// ItemMarginBottom 设置布局项下边距, 其余三边保持不变。
func ItemMarginBottom(item Item, m int32) {
	margin(item, func(rc *xc.RECT) { rc.Bottom = m })
}

// ItemMarginLeft 设置布局项左边距, 其余三边保持不变。
func ItemMarginLeft(item Item, m int32) {
	margin(item, func(rc *xc.RECT) { rc.Left = m })
}

// ItemMarginRight 设置布局项右边距, 其余三边保持不变。
func ItemMarginRight(item Item, m int32) {
	margin(item, func(rc *xc.RECT) { rc.Right = m })
}

// ===== 其它(xcgui 文档语义, 本项目尚未实测, 用之前先确认) =====

// ItemPosition 设置布局项位置偏移(四边)。
//
// ⚠️ 语义取自 xcgui 文档 "窗口组件_布局项_置位置", 本项目未实测 —— 用前先写探针确认相对谁偏移。
func ItemPosition(item Item, left, top, right, bottom int32) {
	item.LayoutItem_SetPosition(left, top, right, bottom)
}

// ItemFloat 让布局项"浮动": 在布局方向上**向反方向对齐**(行布局里靠尾, 即 CSS 的 margin-left:auto 效果)。
//
// ⚠️ 语义取自 xcgui 文档 "启用浮动, 向反方向对齐", 本项目未实测(demo 里靠 Weight 撑开实现同行靠尾)。
func ItemFloat(item Item) {
	item.LayoutItem_EnableFloat()
}

// ItemWrap 让布局项强制换行(从该项开始折到下一行/列)。
//
// ⚠️ 语义取自 xcgui 文档 "启用换行, 强制换行", 本项目未实测。
func ItemWrap(item Item) {
	item.LayoutItem_EnableWrap()
}

// ItemSwap 让布局项在水平/垂直布局切换时交换宽高(及最小宽高)。
//
// ⚠️ 语义取自 xcgui 文档 "根据水平垂直布局变换, 交换属性", 本项目未实测。
func ItemSwap(item Item) {
	item.LayoutItem_EnableSwap()
}
