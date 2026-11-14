package ui

import "github.com/twgh/xcgui/wapi"

// UsePointerCursor 复刻 shadcn 的 `init --pointer` 开关: 打开后, 可交互控件
// (Button / Checkbox / Radio / Switch) 使用系统手型光标。
//
// Tailwind v4 起按钮默认是 cursor: default, 与桌面应用的观感一致, 所以默认关闭;
// 需要在创建控件**之前**调用 —— 已经创建的控件不会追溯改光标。
func UsePointerCursor(on bool) { usePointerCursor = on }

// PointerCursorEnabled 返回当前是否启用了手型光标。
func PointerCursorEnabled() bool { return usePointerCursor }

var usePointerCursor bool

var (
	pointerCursorHandle    uintptr
	notAllowedCursorHandle uintptr
)

// linkCursor 返回手型光标(链接按钮恒用, 与全局 UsePointerCursor 无关)。
func linkCursor() uintptr {
	if pointerCursorHandle == 0 {
		pointerCursorHandle = wapi.LoadImageW(0, wapi.IDC_HAND, wapi.IMAGE_CURSOR, 0, 0,
			wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)
	}
	return pointerCursorHandle
}

// controlCursor 返回控件当前的鼠标指针: 禁用时 not-allowed(checkbox.tsx 的
// disabled:cursor-not-allowed), 否则按全局开关取 hand, 关闭时返回 0(系统箭头)。
func controlCursor(disabled bool) uintptr {
	if disabled {
		if notAllowedCursorHandle == 0 {
			notAllowedCursorHandle = wapi.LoadImageW(0, wapi.IDC_NO, wapi.IMAGE_CURSOR, 0, 0,
				wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)
		}
		return notAllowedCursorHandle
	}
	if !usePointerCursor {
		return 0
	}
	return linkCursor()
}
