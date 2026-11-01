package ui

import (
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/wapi/wutil"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xcc"
)

// TrayIcon 是系统托盘图标 + 右键菜单的封装。
//
//	xui.NewTray(w.Handle, "app.ico", "我的应用", xui.WithTrayMenu(func(m *ui.Menu) {
//	    m.Item("显示主窗口", ui.MenuOnSelect(func() { w.Show(true) }))
//	    m.Separator()
//	    m.Item("退出", ui.MenuDestructive(), ui.MenuOnSelect(func() { w.CloseWindow() }))
//	}))
//
// 默认左键单击显示/还原绑定窗口; 右键单击弹出 WithTrayMenu 构建的菜单。
// 托盘图标在 NewTray 时立即显示, 用 Hide/Show 切换。
//
// ⚠️ xcgui 的托盘 API 是进程级全局状态(一次只有一个图标数据), 同一进程应只创建
// 一个 TrayIcon; 多个实例会互相覆盖图标与提示文本。
type TrayIcon struct {
	hWindow int
	win     *window.Window
	tray    *window.TrayIcon

	hIcon    uintptr
	ownsIcon bool
	tips     string
	visible  bool

	onLeftClick func()
	buildMenu   func(*Menu)

	taskbarMsg uint32
	destroyed  bool
}

// TrayOption 是 TrayIcon 的配置项。
type TrayOption func(*trayIconConfig)

type trayIconConfig struct {
	hIcon       uintptr
	onLeftClick func()
	buildMenu   func(*Menu)
}

// WithTrayIconHandle 用调用方自己的 HICON(例如资源里的图标)代替 iconPath。
// 该句柄归调用方所有, TrayIcon 销毁时不会 DestroyIcon。
func WithTrayIconHandle(hIcon uintptr) TrayOption {
	return func(c *trayIconConfig) { c.hIcon = hIcon }
}

// WithTrayMenu 在右键单击时用 build 构建菜单。菜单是 themed Menu, 每次右键重建,
// 勾选/单选状态由调用方持有。
func WithTrayMenu(build func(m *Menu)) TrayOption {
	return func(c *trayIconConfig) { c.buildMenu = build }
}

// WithTrayLeftClick 替换默认的左键单击行为(默认显示/还原绑定窗口)。
func WithTrayLeftClick(fn func()) TrayOption {
	return func(c *trayIconConfig) { c.onLeftClick = fn }
}

// NewTray 创建托盘图标并立即显示。iconPath 是 .ico 文件路径; 用
// WithTrayIconHandle 传 HICON 时可以传空串。两者都拿不到图标时返回 nil。
//
// tips 是托盘悬停提示(上限 127 字符)。hWindow 同时是托盘事件的承载体与默认
// 左键"显示窗口"的目标。
func NewTray(hWindow int, iconPath, tips string, opts ...TrayOption) *TrayIcon {
	cfg := trayIconConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	win := window.NewByHandle(hWindow)
	if win == nil {
		return nil
	}
	t := &TrayIcon{
		hWindow:     hWindow,
		win:         win,
		tips:        tips,
		onLeftClick: cfg.onLeftClick,
		buildMenu:   cfg.buildMenu,
	}
	if cfg.hIcon != 0 {
		t.hIcon = cfg.hIcon
	} else if iconPath != "" {
		h := wutil.HIcon(iconPath)
		if h == 0 {
			return nil
		}
		t.hIcon = h
		t.ownsIcon = true
	}
	if t.hIcon == 0 {
		return nil
	}

	// 托盘图标事件: wParam 是托盘唯一标识, lParam 是 WM_LBUTTONDOWN / WM_RBUTTONDOWN 等。
	win.AddEvent_TrayIcon(func(wParam, lParam uintptr, pbHandled *bool) int {
		if t.destroyed || t.tray == nil || int32(wParam) != t.tray.Id {
			return 0
		}
		switch xcc.WM_(lParam) {
		case xcc.WM_LBUTTONDOWN:
			*pbHandled = true
			if t.onLeftClick != nil {
				t.onLeftClick()
			} else {
				t.showHost()
			}
		case xcc.WM_RBUTTONDOWN:
			*pbHandled = true
			t.popupMenu()
		}
		return 0
	})

	// Explorer 重启后托盘图标会消失: 监听 TaskbarCreated 广播并重新添加。
	t.taskbarMsg = uint32(wapi.RegisterWindowMessageW("TaskbarCreated"))
	if t.taskbarMsg != 0 {
		win.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
			if message == wapi.WM_DESTROY {
				t.Destroy()
				return 0
			}
			if message == t.taskbarMsg && t.visible && !t.destroyed {
				// 图标已被 Explorer 丢弃, 先重置再重新设置并添加。
				if t.tray != nil {
					t.tray.Reset()
					t.tray.SetIcon(t.hIcon).SetTips(t.tips)
					t.tray.Show(true)
				}
			}
			return 0
		})
	} else {
		win.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
			if message == wapi.WM_DESTROY {
				t.Destroy()
			}
			return 0
		})
	}

	t.tray = window.NewTrayIcon(hWindow, t.hIcon, t.tips)
	if t.tray == nil || !t.tray.Show(true).IsOk() {
		if t.ownsIcon && t.hIcon != 0 {
			wapi.DestroyIcon(t.hIcon)
		}
		return nil
	}
	t.visible = true
	return t
}

// Show 显示托盘图标(隐藏后重新显示时会重新应用图标与提示文本)。
func (t *TrayIcon) Show() {
	if t == nil || t.destroyed || t.tray == nil {
		return
	}
	t.tray.SetIcon(t.hIcon).SetTips(t.tips)
	t.tray.Show(true)
	t.visible = true
}

// Hide 隐藏托盘图标(内部是 XTrayIcon_Del, 图标对象保留以便再次 Show)。
func (t *TrayIcon) Hide() {
	if t == nil || t.destroyed || t.tray == nil {
		return
	}
	t.tray.Show(false)
	t.visible = false
}

// SetIcon 换一个 .ico 文件图标; 成功后旧的自有图标会被销毁。
// 失败(路径无效)时保留原图标。
func (t *TrayIcon) SetIcon(iconPath string) *TrayIcon {
	if t == nil || t.destroyed || iconPath == "" {
		return t
	}
	h := wutil.HIcon(iconPath)
	if h == 0 {
		return t
	}
	t.replaceIcon(h, true)
	return t
}

// SetIconHandle 用调用方自己的 HICON 换图标(不会被 DestroyIcon)。
func (t *TrayIcon) SetIconHandle(hIcon uintptr) *TrayIcon {
	if t == nil || t.destroyed || hIcon == 0 {
		return t
	}
	t.replaceIcon(hIcon, false)
	return t
}

// replaceIcon 应用新图标; 旧的自有图标在新图标写进全局后即可释放。
func (t *TrayIcon) replaceIcon(hIcon uintptr, owns bool) {
	old, oldOwns := t.hIcon, t.ownsIcon
	t.hIcon, t.ownsIcon = hIcon, owns
	if t.tray != nil {
		t.tray.SetIcon(hIcon).SetTips(t.tips).Modify()
	}
	if oldOwns && old != 0 && old != hIcon {
		wapi.DestroyIcon(old)
	}
}

// SetTips 更新悬停提示文本(上限 127 字符)。
func (t *TrayIcon) SetTips(tips string) *TrayIcon {
	if t == nil || t.destroyed {
		return t
	}
	t.tips = tips
	if t.tray != nil {
		t.tray.SetTips(tips).Modify()
	}
	return t
}

// Balloon 弹出系统托盘气泡(原生 Shell 通知)。
// flags 用 xcc.TrayIcon_Flag_Icon_Info/Warning/Error/None, 可叠加
// xcc.TrayIcon_Flag_Nosound 静音。
func (t *TrayIcon) Balloon(title, text string, flags xcc.TrayIcon_Flag_) *TrayIcon {
	if t == nil || t.destroyed || t.tray == nil || !t.visible {
		return t
	}
	t.tray.SetPopupBalloon(title, text, 0, flags).Modify()
	return t
}

// Visible 报告托盘图标当前是否显示。
func (t *TrayIcon) Visible() bool {
	return t != nil && !t.destroyed && t.visible
}

// Destroy 从托盘移除图标并释放自有 HICON(外部句柄不释放)。可重复调用。
func (t *TrayIcon) Destroy() {
	if t == nil || t.destroyed {
		return
	}
	t.destroyed = true
	if t.tray != nil {
		t.tray.Show(false)
		t.tray = nil
	}
	if t.ownsIcon && t.hIcon != 0 {
		wapi.DestroyIcon(t.hIcon)
	}
	t.ownsIcon = false
	t.hIcon = 0
	t.visible = false
}

// showHost 是默认左键行为: 还原并前台化绑定窗口。
func (t *TrayIcon) showHost() {
	if t.destroyed {
		return
	}
	w := window.NewByHandle(t.hWindow)
	if w == nil {
		return
	}
	w.Show(true)
	// 无脑 SW_RESTORE 会把最大化窗口缩成普通大小; 最大化时用 SW_SHOWMAXIMIZED
	// 恢复(最小化前是最大化的, 走这里也能回到最大化)。
	if w.IsMaxWindow() {
		w.ShowWindow(xcc.SW_SHOWMAXIMIZED)
	} else {
		w.ShowWindow(xcc.SW_RESTORE)
	}
	wapi.SetForegroundWindow(w.GetHWND())
}

// popupMenu 用最新的状态重建菜单并在光标处弹出。
func (t *TrayIcon) popupMenu() {
	if t.destroyed || t.buildMenu == nil {
		return
	}
	m := NewMenu()
	t.buildMenu(m)
	m.PopupAtCursor(t.hWindow)
}
