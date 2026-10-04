package xui

import (
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
)

// 焦点来源跟踪, 实现 shadcn 的 :focus-visible 语义:
//   - 鼠标点击获得的焦点 → 不显示焦点框(按钮/复选框/单选);
//   - Tab 键移动的焦点 → 显示焦点框;
//   - 文本输入框例外: 获得焦点即显示(浏览器对文本输入控件始终匹配 focus-visible)。
var focusSrc = struct {
	mouse  bool         // 最近一次焦点由鼠标点击获得
	hooked map[int]bool // 已挂 Tab 键钩子的窗口
}{hooked: map[int]bool{}}

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

// AttachWindow 组件构造时调用, 为组件所在窗口挂接全局服务:
//   - 登记窗口随主题切换重绘;
//   - 挂 Tab 键钩子, Tab 后获得的焦点标记为键盘来源。
//
// 每个窗口只挂一次, 幂等。
func AttachWindow(hEle int) {
	hWnd := xc.XWidget_GetHWND(hEle)
	if hWnd == 0 {
		return
	}
	hWindow := xc.XC_hWindowFromHWnd(hWnd)
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
	w.AddEvent_KeyDown(func(hWindow int, wParam, lParam uintptr, pbHandled *bool) int {
		if wParam == 0x09 { // VK_TAB: 之后的焦点来自键盘
			focusSrc.mouse = false
		}
		return 0
	})
}
