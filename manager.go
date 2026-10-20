package xui

import (
	"sync"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// themeSub 是一次主题切换订阅。Go 的函数值不可比较(只能与 nil 比),
// 所以用自增 id 做反注册的句柄, 不能用函数地址或 == 判断。
type themeSub struct {
	id int
	fn func(*Theme)
}

// 主题管理器: 持有当前主题, 组件绘制时无锁读取。
//
// 约束(基于 xcgui 的事件模型):
//   - UI 事件回调一定在 UI 线程, SetTheme 也应仅在 UI 线程调用;
//   - 已注册窗口切换主题后需要重绘, 由本包统一触发 XWnd_Redraw。
type themeManager struct {
	mu      sync.RWMutex
	current *Theme
	windows map[int]struct{} // RegisterWindow: 主题切换时需重绘的窗口
	winBg   map[int]struct{} // BindWindowBackground: 主题切换时需重刷底色的窗口
	onChg   []themeSub
	nextID  int
}

var mgr = &themeManager{
	current: Light(),
	windows: map[int]struct{}{},
	winBg:   map[int]struct{}{},
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
	bg := make([]int, 0, len(mgr.winBg))
	for h := range mgr.winBg {
		bg = append(bg, h)
	}
	cbs := make([]func(*Theme), len(mgr.onChg))
	for i, s := range mgr.onChg {
		cbs[i] = s.fn
	}
	mgr.mu.Unlock()

	// 快照后不加锁执行: 回调里可能再订阅/反注册或调 SetTheme。
	for _, h := range bg {
		applyWindowBackground(h, t)
	}
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

// UnregisterWindow 取消登记窗口: 同时清掉它注册的重绘与底色绑定,
// 避免窗口销毁后残留失效句柄(主题切换时对失效句柄调 XWnd_* 会弹模态错误框)。
func UnregisterWindow(hWindow int) {
	mgr.mu.Lock()
	delete(mgr.windows, hWindow)
	delete(mgr.winBg, hWindow)
	mgr.mu.Unlock()
}

// OnThemeChange 订阅主题切换。需要把颜色同步给原生 API 的组件
// (如 Edit.SetTextColor)用它; 只在绘制回调里取色的组件无需订阅。
// 返回反注册函数(幂等, 重复调用无副作用)。
func OnThemeChange(fn func(*Theme)) (unsub func()) {
	if fn == nil {
		return func() {}
	}
	mgr.mu.Lock()
	mgr.nextID++
	id := mgr.nextID
	mgr.onChg = append(mgr.onChg, themeSub{id: id, fn: fn})
	mgr.mu.Unlock()

	return func() {
		mgr.mu.Lock()
		defer mgr.mu.Unlock()
		for i, s := range mgr.onChg {
			if s.id == id {
				mgr.onChg = append(mgr.onChg[:i], mgr.onChg[i+1:]...)
				return
			}
		}
	}
}

// BindWindowBackground 把窗口底色绑定到主题 Background 令牌,
// 主题切换时自动更新。必须用 Leave 状态注册填充——窗口绘制时处于
// Window_State_Flag_Leave, 用 Nothing(0) 状态注册的填充不会被选中;
// 且必须先 Clear 清掉默认的白色填充。
// 必须在 UI 线程调用。窗口关闭时调 UnregisterWindow 一并解绑。
func BindWindowBackground(hWindow int) {
	mgr.mu.Lock()
	mgr.winBg[hWindow] = struct{}{}
	cur := mgr.current
	mgr.mu.Unlock()
	applyWindowBackground(hWindow, cur)
}

// OnThemeChangeFor 订阅主题切换, 并在 hEle 销毁时**自动反注册**。
// 组件构造函数用它替代裸 OnThemeChange —— 元素/窗口反复创建时, 裸订阅会在全局
// 订阅表里无限累积, 句柄被复用后还可能把回调刷到无关元素上。
// 实测: 元素显式 Destroy、随父窗口 CloseWindow 连带销毁、
// 以及嵌套在子容器里的元素, 都会触发 XE_DESTROY, 所以这条绑定是可靠的。
func OnThemeChangeFor(hEle int, fn func(*Theme)) {
	if fn == nil {
		return
	}
	unsub := OnThemeChange(fn)
	if hEle == 0 {
		return
	}
	e := widget.NewElementByHandle(hEle)
	if e == nil {
		return
	}
	// 走 AddEvent* 事件总线(不是 XEle_RegEventC, 那条路有 2000 回调上限)。
	e.AddEvent_Destroy(func(int, *bool) int {
		unsub()
		return 0
	})
}

// applyWindowBackground 把窗口底色重刷为主题 Background 令牌。
func applyWindowBackground(hWindow int, t *Theme) {
	bkm := xc.XWnd_GetBkManager(hWindow)
	if bkm == 0 {
		return
	}
	xc.XBkM_Clear(bkm)
	xc.XBkM_AddFill(bkm, xcc.CombinedState_Window_State_Flag_Leave, uint32(t.Background), 0)
}
