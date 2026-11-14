package ui

import (
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 组件占用的元素用户数据标记。xcgui 的元素没有 Go 侧类型信息, 按钮组只能靠它
// 识别"父级是不是按钮组 / 我是组里的哪种部件"。
//
// 本文件被 button / input / combobox / buttongroup 共同依赖(见 registry.json):
// 这些组件单独安装时组语义仍然可用, 不装 buttongroup 也不会缺符号。
const (
	uiRoleBase      = 0x78756900
	uiRoleButton    = uiRoleBase + 1
	uiRoleGroupH    = uiRoleBase + 2
	uiRoleGroupV    = uiRoleBase + 3
	uiRoleGroupText = uiRoleBase + 4
	uiRoleGroupSep  = uiRoleBase + 5
	// ToggleGroup 用独立标记: 组内条目要按 spacing 决定首尾外圆角, 但不能被
	// ButtonGroup 的通用首尾收角规则误伤(两者的边框折叠方式不同)。
	uiRoleToggleGroupTight  = uiRoleBase + 6 // spacing=0: 相邻边收直角
	uiRoleToggleGroupSpaced = uiRoleBase + 7 // spacing>0: 每项保留完整圆角
)

// buttonGroupPlacement 描述元素在按钮组里的位置。绘制时实时读取(不缓存),
// 主题/布局变化都不需要同步状态。
type buttonGroupPlacement struct {
	inGroup  bool
	first    bool
	last     bool
	vertical bool
}

// buttonGroupPlacementOf 查元素是否直接位于按钮组内, 以及它是不是首/尾项。
// 用父元素的用户数据标记识别按钮组, 再按子对象顺序定位自己。
func buttonGroupPlacementOf(hEle int) buttonGroupPlacement {
	parent := xc.XWidget_GetParentEle(hEle)
	if parent == 0 {
		return buttonGroupPlacement{}
	}
	role := xc.XEle_GetUserData(parent)
	if role != uiRoleGroupH && role != uiRoleGroupV {
		return buttonGroupPlacement{}
	}
	p := buttonGroupPlacement{inGroup: true, vertical: role == uiRoleGroupV}
	n := int(xc.XEle_GetChildCount(parent))
	idx := -1
	for i := 0; i < n; i++ {
		if xc.XEle_GetChildByIndex(parent, int32(i)) == hEle {
			idx = i
			break
		}
	}
	if idx < 0 {
		return buttonGroupPlacement{}
	}
	p.first, p.last = idx == 0, idx == n-1
	return p
}

// corners 按组内位置收角: 横向组只保留首项左外角与末项右外角, 纵向组只保留
// 首项上外角与末项下外角; 相邻边画成直角后严丝合缝(见 buttongroup.go)。
func (g buttonGroupPlacement) corners(radius int32) xui.Corners {
	if !g.inGroup {
		return xui.UniformCorners(radius)
	}
	var c xui.Corners
	if g.vertical {
		if g.first {
			c.TopLeft, c.TopRight = radius, radius
		}
		if g.last {
			c.BottomLeft, c.BottomRight = radius, radius
		}
	} else {
		if g.first {
			c.TopLeft, c.BottomLeft = radius, radius
		}
		if g.last {
			c.TopRight, c.BottomRight = radius, radius
		}
	}
	return c
}

// groupCorners 供输入框/下拉框这类"也能直接放进按钮组"的壳体使用:
// 组内按首/中/尾收角, 组外四角统一。
func groupCorners(hEle int, radius int32) xui.Corners {
	return buttonGroupPlacementOf(hEle).corners(radius)
}

// reflowAfterResize 在控件尺寸(尤其高度)变化后逐级通知祖先重排:
// Auto 高的卡片/行只改子元素尺寸不会自己长高, 遇到 ScrollArea 的框架交给框架排。
// 与 Label 的折行重排同口径(reflowAfterTextHeight), 放在 group.go 是因为
// button.go 依赖本文件、但不应额外依赖 label.go。
//
// oldRc 是尺寸变化前的元素矩形: 重排后据此判断矩形是否真的变了, 变了就整窗重绘
// (见 repaintResize)。
func reflowAfterResize(hEle int, oldRc xc.RECT) {
	if hEle == 0 || !xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
		return
	}
	xc.XEle_AdjustLayout(hEle)
	for p := xc.XWidget_GetParentEle(hEle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(p)
			break
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(pp)
			break
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_ELE) {
			xc.XEle_AdjustLayout(p)
			continue
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
		}
		break
	}
	repaintResize(hEle, oldRc)
}

// repaintResize 重绘尺寸变化影响的区域。
//
// 元素变小后, 新矩形之外的旧像素不会被元素自身的 Redraw 覆盖(自绘底色会留下大号尺寸的
// 残影, 见 DESIGN 第八节 200)。而且"尺寸变了"从不只是自己的事: 同一行/列里的兄弟会跟着
// 平移, 它们腾出的旧位置也不在新旧元素的矩形内(实测: 行内头像缩小时右侧按钮整体左移,
// 只重绘并集会在按钮原位置留下半截药丸)。所以尺寸真的变了就整窗重绘一次。
//
// 不要只重绘父元素: 布局壳(LayoutEle)通常是透明的, 自己什么也不画, 擦不掉旧像素;
// 真正需要刷新的是窗口表面。尺寸变化都是用户动作触发的(切换尺寸档 / 改文字 / 换图标),
// 不是每帧路径, 整窗重绘的代价可以接受。
func repaintResize(hEle int, oldRc xc.RECT) {
	repaintUnionWith(hEle, oldRc)
}

// repaintTextBounds 在改文字前后各取一次元素矩形, 矩形变了就补一次重绘。
//
// 原生控件(cx=0 的 Button / Label)在 `SetText` 时会按新文字自撑宽度: 文字变短, 元素当场
// 缩小(与第八节 200 的缩小残影同源, 触发源是文字而不是尺寸档)。矩形没变就不重绘 —— 定宽
// 布局槽里的标签改文字是高频操作, 每次都整窗重绘没有必要。
func repaintTextBounds(hEle int, change func()) {
	if hEle == 0 || change == nil {
		return
	}
	var oldRc xc.RECT
	xc.XEle_GetWndClientRect(hEle, &oldRc)
	change()
	repaintUnionWith(hEle, oldRc)
}

// repaintUnionWith 元素矩形变了就整窗重绘一次(见 repaintResize 的说明: 兄弟也会移动,
// 只刷新旧矩形并集擦不干净)。
func repaintUnionWith(hEle int, oldRc xc.RECT) {
	if hEle == 0 {
		return
	}
	var newRc xc.RECT
	xc.XEle_GetWndClientRect(hEle, &newRc)
	if oldRc.Left == newRc.Left && oldRc.Top == newRc.Top &&
		oldRc.Right == newRc.Right && oldRc.Bottom == newRc.Bottom {
		return
	}
	hwnd := xc.XWidget_GetHWND(hEle)
	if hwnd == 0 {
		return
	}
	hw := xc.XC_hWindowFromHWnd(hwnd)
	if hw == 0 {
		return
	}
	xc.XWnd_Redraw(hw, true)
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
