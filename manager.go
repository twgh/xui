package xui

import (
	"sync"

	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// 主题管理器: 持有当前主题, 组件绘制时无锁读取。
//
// 约束(基于 xcgui 的事件模型):
//   - UI 事件回调一定在 UI 线程, SetTheme 也应仅在 UI 线程调用;
//   - 已注册窗口切换主题后需要重绘, 由本包统一触发 XWnd_Redraw。
type themeManager struct {
	mu      sync.RWMutex
	current *Theme
	windows map[int]struct{} // 已注册需要随主题切换重绘的窗口句柄
	onChg   []func(*Theme)
}

var mgr = &themeManager{
	current: Light(),
	windows: map[int]struct{}{},
}

// SetTheme 切换当前主题并重绘所有已注册窗口。
// 必须在 UI 线程调用(与任何 UI 操作的线程约束一致)。
func SetTheme(t *Theme) {
	if t == nil {
		return
	}
	mgr.mu.Lock()
	mgr.current = t
	hw := make([]int, 0, len(mgr.windows))
	for h := range mgr.windows {
		hw = append(hw, h)
	}
	cbs := make([]func(*Theme), len(mgr.onChg))
	copy(cbs, mgr.onChg)
	mgr.mu.Unlock()

	for _, fn := range cbs {
		fn(t)
	}
	for _, h := range hw {
		xc.XWnd_Redraw(h)
	}
}

// Current 返回当前主题。组件在绘制回调中调用, 无需加锁:
// 事件回调与 SetTheme 都在 UI 线程, 不会并发竞争。
// 若在非 UI 线程需要读取, 使用 CurrentSafe。
func Current() *Theme { return mgr.current }

// CurrentSafe 是 Current 的加锁版本, 供非 UI 线程读取。
func CurrentSafe() *Theme {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.current
}

// RegisterWindow 登记一个窗口, 主题切换时自动重绘它。
// 窗口关闭后应调用 UnregisterWindow, 句柄失效会导致 XWnd_Redraw 静默无效(无害但应避免)。
func RegisterWindow(hWindow int) {
	mgr.mu.Lock()
	mgr.windows[hWindow] = struct{}{}
	mgr.mu.Unlock()
}

// UnregisterWindow 取消登记窗口。
func UnregisterWindow(hWindow int) {
	mgr.mu.Lock()
	delete(mgr.windows, hWindow)
	mgr.mu.Unlock()
}

// OnThemeChange 订阅主题切换。需要把颜色同步给原生 API 的组件
// (如 Edit.SetTextColor)用它; 只在绘制回调里取色的组件无需订阅。
// 返回取消订阅函数。
func OnThemeChange(fn func(*Theme)) (unsub func()) {
	mgr.mu.Lock()
	mgr.onChg = append(mgr.onChg, fn)
	mgr.mu.Unlock()
	return func() {
		mgr.mu.Lock()
		for i, f := range mgr.onChg {
			if &f == &fn {
				mgr.onChg = append(mgr.onChg[:i], mgr.onChg[i+1:]...)
				break
			}
		}
		mgr.mu.Unlock()
	}
}

// BindWindowBackground 把窗口底色绑定到主题 Background 令牌,
// 主题切换时自动更新。必须用 Leave 状态注册填充——窗口绘制时处于
// Window_State_Flag_Leave, 用 Nothing(0) 状态注册的填充不会被选中;
// 且必须先 Clear 清掉默认的白色填充。
// 必须在 UI 线程调用。
func BindWindowBackground(hWindow int) {
	apply := func(t *Theme) {
		bkm := xc.XWnd_GetBkManager(hWindow)
		if bkm == 0 {
			return
		}
		xc.XBkM_Clear(bkm)
		xc.XBkM_AddFill(bkm, xcc.CombinedState_Window_State_Flag_Leave, uint32(t.Background), 0)
	}
	apply(mgr.current)
	OnThemeChange(apply)
}
