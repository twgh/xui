package ui

import (
	"github.com/twgh/xcgui/xc"
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
