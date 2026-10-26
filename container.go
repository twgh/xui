package xui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
)

// ===== 容器构造(流式布局骨架) =====
//
// LayoutEle 是"横向流 + 自动换行"的盒子, 手写一遍要记一串实测语义(见 DESIGN 第八节 29/41):
// EnableHorizon / SetSpace / SetAlignV / 自己那条轴的 Fill|Fixed, 还要关掉纵向溢出换列。
// 这三个构造把 app 里 99% 用到的三种骨架固定下来, **布局项尺寸也一并由构造写好**,
// 调用方只需要往里塞控件:
//
//	card.ContentStack(8)                  // 卡片内容区: 纵向流水(卡片是布局容器, 内容区随卡片缩放)
//	xui.HStack(v, 36, 8)                  // 一行: 宽度铺满, 高 36, 子项垂直居中
//	xui.VStackWeight(row, 1, 8)           // 行里的一列: 宽度等分, 高度占满
//
// 其它组合(比如"铺满父级的纵向区")用 widget.NewLayoutEle + xui.Item*(见 layout.go)自己拼。

// HStack 建一个横向行容器(作为 parent 布局的布局项): 宽度铺满, 高度固定 ht, 子项间距 space,
// 交叉轴(垂直)居中 —— 行内混合高度的控件靠它对齐。
//
// 行内的项按需要声明宽度: Fill 会独占整行, 等分剩余宽度用 xui.ItemWeight。
func HStack(parent int, ht, space int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, ht, parent)
	l.EnableHorizon(true)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	// 关掉"横向溢出换行": 行内总宽超了会把后面的项甩到下一行(y 超出行高), 整块消失
	// (实测, 见 DESIGN 第八节 41)。行宽不够时应该由调用方收窄子项, 而不是静默换行。
	l.EnableAutoWrap(false)
	l.SetSpace(space)
	l.SetAlignV(xcc.Layout_Align_Center)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, ht)
	return l
}

// VStackWeight 建一个纵向流容器, 宽度按权重等分**横向父布局**的剩余宽度(做等宽列用, 传 1 = 等分),
// 高度占满父级, 子项间距 space。
//
// 注意高度是 `Fill`(父级整高): 列里装的是"高度自撑(Auto)"的卡片时**别用它**, 要自己拼一个
// "Weight 宽 + Auto 高"的列 —— 否则卡片内容区里的整宽控件(原生 Edit/ComboBox/分隔线)会按
// 卡片宽排版、右边缘压到卡片边框上(实测, 见 DESIGN 第八节 59)。它适合列内是"吃剩余高度"的面板。
func VStackWeight(parent int, weight, space int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, parent)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.SetSpace(space)
	// 关掉"纵向溢出换列": LayoutEle 在主轴放不下下一项时会把它甩到下一列(x += 容器宽度, y = 0),
	// 表现为控件整块消失(实测, 见 DESIGN 第八节 41) —— 列容器一律关掉。
	l.EnableAutoWrap(false)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Weight, weight)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
	return l
}
