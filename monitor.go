package xui

import (
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/xc"
)

// MonitorWorkArea 返回窗口所在显示器的工作区(物理像素); 取不到时 ok=false。
//
// 用工作区而不是整屏: 工作区已经扣掉任务栏 / 停靠栏。弹层夹在整屏高度里时, 只要
// 触发区靠近屏幕底部(例如页面已滚到底), 卡片下半截就会被任务栏盖住 —— 任务栏永远
// 在最上层, 弹层是 topmost 也盖不过它。
//
// ⚠️ hWnd 必须是 **Win32 HWND**(xc.XWnd_GetHWND 转出来的), 不能喂 xcgui 的窗口句柄:
// MonitorFromWindow 对无效句柄返回 0, 调用方会静默退回"整屏"兜底分支, 任务栏又会盖上来。
//
// "上方放不下翻到下方"、贴边夹紧这类判断都必须按它算, 不能拿 (0,0) 当屏幕边缘:
// 显示器原点不一定是 (0,0)(显示器摆在主屏正上方/右侧时坐标是负的或偏移的),
// 拿 0 当边界会把明明放得下的弹层/提示翻走。
func MonitorWorkArea(hWnd uintptr) (xc.RECT, bool) {
	hMon := wapi.MonitorFromWindow(hWnd, wapi.MONITOR_DEFAULTTONEAREST)
	if hMon == 0 { // 句柄无效时退回桌面窗口: 至少仍是"工作区", 不会漏掉任务栏
		hMon = wapi.MonitorFromWindow(wapi.GetDesktopWindow(), wapi.MONITOR_DEFAULTTONEAREST)
	}
	if hMon == 0 {
		return xc.RECT{}, false
	}
	var mi wapi.MONITORINFO
	if !wapi.GetMonitorInfo(hMon, &mi) {
		return xc.RECT{}, false
	}
	return xc.RECT{
		Left:   mi.RcWork.Left,
		Top:    mi.RcWork.Top,
		Right:  mi.RcWork.Right,
		Bottom: mi.RcWork.Bottom,
	}, true
}
