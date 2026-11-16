package xui

import (
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// 焦点来源跟踪, 实现 shadcn 的 :focus-visible 语义:
//   - 鼠标点击获得的焦点 → 不显示焦点框(按钮/复选框/单选);
//   - Tab 键移动的焦点 → 显示焦点框;
//   - 文本输入框例外: 获得焦点即显示(浏览器对文本输入控件始终匹配 focus-visible)。
var focusSrc = struct {
	mouse    bool         // 最近一次焦点由鼠标点击获得
	hooked   map[int]bool // 已挂 Tab 键钩子的窗口
	tabStops map[int]bool // 显式声明的自绘 Tab 站点(普通 Element)
}{hooked: map[int]bool{}, tabStops: map[int]bool{}}

const (
	vkTab   uintptr = 0x09
	vkShift int32   = 0x10
)

// NotePointerPress 由组件在鼠标按下时调用: 随后获得的焦点视为鼠标焦点。
func NotePointerPress() {
	focusSrc.mouse = true
}

// NoteKeyboard 由组件在键盘导航时调用(例如对话框内的 Tab 焦点循环在自行搬焦点前):
// 随后获得的焦点视为键盘焦点, 组件会绘制焦点框。
func NoteKeyboard() {
	focusSrc.mouse = false
}

// FocusedByKeyboard 报告当前焦点是否来自键盘 Tab。
// 组件绘制时据此决定是否绘制焦点框(文本输入类无需判断)。
func FocusedByKeyboard() bool {
	return !focusSrc.mouse
}

// HostWindowOf 返回元素所属的 **xcgui 窗口**句柄(找不到返回 0)。
//
// 不能一步 `XC_hWindowFromHWnd(XWidget_GetHWND(ele))`: 元素在 ScrollArea(LayoutFrame)、
// 弹层容器这类自带 HWND 的容器内部时, 取到的 HWND 不是 xcgui 窗口, 映射出来是个无效
// 小值(实测 7) —— 拿它去 `XWnd_*`、presence、主题重绘都会静默失败。沿 HWND 父链往上找
// 第一个"映射成功且确实是窗口"的句柄。
func HostWindowOf(hEle int) int {
	for hwnd := xc.XWidget_GetHWND(hEle); hwnd != 0; hwnd = wapi.GetParent(hwnd) {
		if h := xc.XC_hWindowFromHWnd(hwnd); h != 0 && xc.XC_IsHXCGUI(h, xcc.XC_WINDOW) {
			return h
		}
	}
	return 0
}

// FocusElement 把键盘焦点移到元素上, 元素没有焦点能力时返回 false。
//
// 不能直接用 `XWnd_SetFocusEle(XC_hWindowFromHWnd(XWidget_GetHWND(ele)), ele)`:
// 元素在 ScrollArea / 弹层容器里时那个 HWND 不是 xcgui 窗口(见 HostWindowOf), 这里沿父链
// 找到真正的宿主窗口再设焦点。
func FocusElement(hEle int) bool {
	if hEle == 0 {
		return false
	}
	hWindow := HostWindowOf(hEle)
	if hWindow == 0 {
		return false
	}
	return xc.XWnd_SetFocusEle(hWindow, hEle)
}

// AttachWindow 组件构造时调用, 为组件所在窗口挂接全局服务:
//   - 登记窗口随主题切换重绘;
//   - 挂 Tab 键钩子, Tab 后获得的焦点标记为键盘来源;
//   - tabStop 传 true 时把该元素登记为自绘 Tab 站点(普通 Element 没有原生控件类型,
//     跨层 Tab 遍历只收显式登记的; 纯布局/装饰元素不要传)。
//
// 每个窗口只挂一次, 幂等。
func AttachWindow(hEle int, tabStop ...bool) {
	if len(tabStop) > 0 && tabStop[0] {
		markTabStop(hEle)
	}
	hWindow := HostWindowOf(hEle)
	if hWindow == 0 {
		return
	}
	RegisterWindow(hWindow)
	if focusSrc.hooked[hWindow] {
		return
	}
	focusSrc.hooked[hWindow] = true
	w := window.NewByHandle(hWindow)
	if w == nil {
		return
	}
	// 任何承载 xui 组件的窗口(不只是 AppWindow: 屏幕通知小窗、对话框、弹层)
	// 都在上面登记过主题重绘, 必须在该窗口销毁时摘掉 —— 句柄失效后
	// SetTheme 里的 XWnd_Redraw 会弹 xcgui 模态错误框(不是静默无效)。
	w.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
		if message == wapi.WM_DESTROY {
			UnregisterWindow(hWindow)
			delete(focusSrc.hooked, hWindow)
		}
		return 0
	})
	w.AddEvent_KeyDown(func(hWindow int, wParam, lParam uintptr, pbHandled *bool) int {
		if wParam != vkTab {
			return 0
		}
		// VK_TAB: 之后的焦点来自键盘
		focusSrc.mouse = false
		// 代码/多行编辑框自己接收 Tab 输入时不抢(EnableKeyTab)
		if hFocus := xc.XWnd_GetFocusEle(hWindow); hFocus != 0 && xc.XEle_IsKeyTab(hFocus) {
			return 0
		}
		if moveFocusByTab(hWindow, wapi.GetAsyncKeyState(vkShift) < 0) {
			*pbHandled = true
		}
		return 0
	})
}

// moveFocusByTab 在窗口内**跨层**移动焦点。
//
// xcgui 原生的 Tab 遍历只在一个父容器内部循环: 套一层 LayoutEle / Element 之后,
// 外面的控件 Tab 不进去, 里面的控件也 Tab 不出来(见 DESIGN 第八节 218)。
// 页面里的组件几乎都嵌在卡片/流容器里, 所以这里在窗口级接管 Tab:
// 深度优先收集整棵元素树里的 Tab 站点, 再按当前焦点位置取下一个。
// 返回 false 表示没有可移动的目标, 交回框架默认处理。
func moveFocusByTab(hWindow int, back bool) bool {
	stops := tabStops(hWindow)
	if len(stops) == 0 {
		return false
	}
	cur := xc.XWnd_GetFocusEle(hWindow)
	idx := -1
	// 焦点可能落在复合控件的内部元素上, 沿父链反查所属的 Tab 站点
	for h := cur; h != 0; h = xc.XWidget_GetParentEle(h) {
		for i, s := range stops {
			if s == h {
				idx = i
				break
			}
		}
		if idx >= 0 {
			break
		}
	}
	// SetFocusEle 对复合控件未必一次成功, 逐个往后试(同对话框的焦点循环)
	for i := 0; i < len(stops); i++ {
		var next int
		switch {
		case idx >= 0 && back:
			next = (idx - 1 - i + 2*len(stops)) % len(stops)
		case idx >= 0:
			next = (idx + 1 + i) % len(stops)
		case back:
			next = len(stops) - 1 - i
		default:
			next = i
		}
		if xc.XWnd_SetFocusEle(hWindow, stops[next]) {
			// 搬焦点不会自动重绘, 焦点环要手动刷新新旧两个元素
			if cur != 0 && cur != stops[next] {
				xc.XEle_Redraw(cur, false)
			}
			xc.XEle_Redraw(stops[next], false)
			return true
		}
	}
	return false
}

// markTabStop 登记一个自绘 Tab 站点, 元素销毁时自动摘掉(句柄可能被复用)。
func markTabStop(hEle int) {
	if hEle == 0 || focusSrc.tabStops[hEle] {
		return
	}
	focusSrc.tabStops[hEle] = true
	if ele := widget.NewElementByHandle(hEle); ele != nil {
		ele.AddEvent_Destroy(func(h int, pbHandled *bool) int {
			delete(focusSrc.tabStops, h)
			return 0
		})
	}
}

// tabStops 收集窗口整棵元素树里的 Tab 站点(深度优先, 与创建顺序一致)。
func tabStops(hWindow int) []int {
	var out []int
	n := xc.XWnd_GetChildCount(hWindow)
	for i := int32(0); i < n; i++ {
		collectTabStops(xc.XWnd_GetChildByIndex(hWindow, i), &out)
	}
	return out
}

func collectTabStops(hEle int, out *[]int) {
	if hEle == 0 || !xc.XWidget_IsShow(hEle) || !xc.XEle_IsEnable(hEle) || !tabVisible(hEle) {
		return
	}
	if isTabStop(hEle) {
		*out = append(*out, hEle)
	}
	if !tabDescend(hEle) {
		return
	}
	n := xc.XEle_GetChildCount(hEle)
	for i := int32(0); i < n; i++ {
		collectTabStops(xc.XEle_GetChildByIndex(hEle, i), out)
	}
}

// isTabStop 判断元素是否是 Tab 站点:
//   - 原生交互控件按类型白名单(同对话框的焦点循环);
//   - 自绘组件把根元素(普通 Element)通过 AttachWindow(h, true) 显式登记。
//
// 纯布局容器(布局元素)与没有登记的普通 Element(装饰/文本)都不进链 ——
// xcgui 的普通 Element 默认可聚焦, 不能拿"可聚焦"当作 Tab 站点的判据。
func isTabStop(hEle int) bool {
	if !xc.XEle_IsEnableFocus(hEle) || !xc.XEle_IsSwitchFocus(hEle) {
		return false
	}
	switch xc.XObj_GetType(hEle) {
	case xcc.XC_BUTTON, xcc.XC_EDIT, xcc.XC_COMBOBOX, xcc.XC_SLIDERBAR,
		xcc.XC_EDITOR, xcc.XC_RICHEDIT, xcc.XC_LISTBOX, xcc.XC_TABBAR,
		xcc.XC_LIST, xcc.XC_LISTVIEW, xcc.XC_TREE, xcc.XC_MONTHCAL,
		xcc.XC_DATETIME:
		return true
	case xcc.XC_ELE:
		return focusSrc.tabStops[hEle]
	}
	return false
}

// tabDescend 报告是否要继续深入元素的子级。只进布局容器与自绘组件的根元素;
// 原生控件(列表行、编辑框内部滚动条等)的内部元素不是独立的 Tab 站点。
func tabDescend(hEle int) bool {
	switch xc.XObj_GetType(hEle) {
	case xcc.XC_ELE, xcc.XC_ELE_LAYOUT, xcc.XC_LAYOUT_FRAME, xcc.XC_SCROLLVIEW,
		xcc.XC_PANE, xcc.XC_PANE_CELL:
		return true
	}
	return false
}

// tabVisible 判断元素是否真的显示(自身与全部祖先都 Show, 窗口未最小化/隐藏)。
func tabVisible(hEle int) bool {
	if hwnd := xc.XWidget_GetHWND(hEle); hwnd != 0 {
		if !wapi.IsWindowVisible(uintptr(hwnd)) || wapi.IsIconic(uintptr(hwnd)) {
			return false
		}
	}
	for h := hEle; h != 0; h = xc.XWidget_GetParentEle(h) {
		if !xc.XWidget_IsShow(h) {
			return false
		}
	}
	return true
}
