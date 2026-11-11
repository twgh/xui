package xui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// ===== 容器构造(流式布局骨架) =====
//
// LayoutEle 是"横向流 + 自动换行"的盒子, 手写一遍要记一串实测语义(见 DESIGN 第八节 29/41):
// EnableHorizon / SetSpace / SetAlignV / 自己那条轴的 Fill|Fixed, 还要关掉纵向溢出换列。
// 这几个构造把 app 里 99% 用到的骨架固定下来, **布局项尺寸也一并由构造写好**,
// 调用方只需要往里塞控件:
//
//	card.ContentStack(8)                  // 卡片内容区: 纵向流水(卡片是布局容器, 内容区随卡片缩放)
//	xui.HStack(v, 36, 8)                  // 一行: 宽度铺满, 高 36, 子项垂直居中
//	xui.VStack(row, 1, 8)                 // 行里的一列: 宽度等分, 高度按内容自撑(卡片列)
//	xui.VStackWeight(row, 1, 8)           // 行里的一列: 宽度等分, 高度占满(表格/树这类面板)
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

// VStack 建一个纵向流容器: 宽度按权重等分横向父布局的剩余宽度, **高度按内容自撑(Auto)**,
// 子项间距 space —— 用于"一列高度自撑的卡片"(对比 VStackWeight 的整高面板)。
//
// 列里常有整宽(Fill)的嵌套控件; Weight 宽不会下发给嵌套布局的 Fill 子项(见 DESIGN 第八节 101),
// 卡片内容区改成元素 padding 后(见 129), InputGroup / Collapsible 这类子项会变 0 宽整块消失。
// 这里在列宽确定/变化时把 Fill 子项锁成 Fixed(同卡片头部的 syncTitleWidth), 调用方不用自己处理。
func VStack(parent int, weight, space int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, parent)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(space)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Weight, weight)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	// 宽度在 AfterFunc 里现读, 不能捕获 Size 事件那一刻的值 —— 缩放窗口时事件会先给中间宽,
	// 捕获值会让判断失效, 子项停在旧宽(表现为缩放后行尾按钮被卡片裁掉)。
	var lastW int32 = -1
	var syncPending bool
	synced := map[int]bool{}
	l.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		if syncPending {
			return 0
		}
		syncPending = true
		app.AfterFunc(0, func() {
			syncPending = false
			w := l.GetWidth()
			if w <= 0 || w == lastW {
				return
			}
			lastW = w
			next := make(map[int]bool, len(synced))
			for i := int32(0); i < xc.XEle_GetChildCount(l.Handle); i++ {
				child := xc.XEle_GetChildByIndex(l.Handle, i)
				var typ xcc.Layout_Size_
				var cw int32
				xc.XWidget_LayoutItem_GetWidth(child, &typ, &cw)
				// 第一次锁 Fill 子项后它会变成 Fixed, 后续尺寸变化按 synced 继续更新。
				if typ == xcc.Layout_Size_Fill || synced[child] {
					xc.XWidget_LayoutItem_SetWidth(child, xcc.Layout_Size_Fixed, w)
					next[child] = true
				}
			}
			synced = next
			l.AdjustLayout()
		})
		return 0
	})
	return l
}

// VStackWeight 建一个纵向流容器, 宽度按权重等分**横向父布局**的剩余宽度(做等宽列用, 传 1 = 等分),
// 高度占满父级, 子项间距 space。
//
// 注意高度是 `Fill`(父级整高): 列里装的是"高度自撑(Auto)"的卡片时**别用它**, 用 VStack ——
// 否则卡片内容区里的整宽控件(原生 Edit/ComboBox/分隔线)会按卡片宽排版、右边缘压到卡片边框上
// (实测, 见 DESIGN 第八节 59)。它适合列内是"吃剩余高度"的面板(表格/树)。
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
