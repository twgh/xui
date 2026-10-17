package xui

import (
	"sync/atomic"
	"time"

	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// ===== 弹层 / 对话框的进出场过渡(presence) =====
//
// 与 hover 过渡共用同一套驱动器(见 anim.go): 同一个目标上的多个过渡共用一个定时器,
// 全部到位自动关掉。区别在于 presence 面向"出现/消失":
//
//   - 值仍是 0..1 的进度, 但由 apply 回调每帧落到具体表现上(元素透明度 / 窗口位置 / 窗口 alpha);
//   - 到位时回调 done(每个条目只调一次), 退场到位后条目被回收;
//   - 键空间与 hover 隔离(presence 的键内部映射到负数)。
//
// 时长与缓动对齐 shadcn / tailwind:
//
//   - dialog / dropdown-menu / select 都写的是 duration-100 → PopupDuration(100ms);
//   - tooltip 没写时长 → tw-animate-css 的 animate-in 默认 150ms;
//   - 缓动取 CSS 默认的 `ease` = cubic-bezier(0.25,0.1,0.25,1)(animate-in 是 keyframes,
//     用的是 animation 的默认缓动, 与 transition-* 的 tailwind 默认值
//     cubic-bezier(0.4,0,0.2,1) 不是同一条)。
const (
	// PopupDuration 弹层 / 对话框进出场时长(shadcn duration-100)。
	PopupDuration = 100 * time.Millisecond
	// TooltipDuration 提示气泡进出场时长(tw-animate-css 的 animate-in 默认 150ms)。
	TooltipDuration = 150 * time.Millisecond
)

// presenceKeyBase 把 presence 的键移进负数空间, 与 hover 的键(0 与 行索引+base)隔离。
const presenceKeyBase int32 = -100000

// presenceKeySeq 是全局递增的 presence 键分配器(见 NextPresenceKey)。
var presenceKeySeq int32

// NextPresenceKey 分配一个全局唯一的 presence 键。
//
// 键在"同一个 hOwner"内必须互不相同: Presence 是幂等的 —— 目标值没变时它不会重启过渡,
// 也不再调用 apply(过渡由定时器逐帧驱动, 到位后条目就静默了)。若两个不同的弹层复用同一个
// 键, 第二个弹层的新窗口会**停留在初始状态**(例如 alpha 压在下限 = 完全看不见, 但窗口已经
// 存在并吞掉点击), 表现为"点了没反应 / 第二次才生效"。用本函数取键即可避免手工计数器撞车
// (2026-10-17 实测: menu 与 combobox 各自的计数器都从 1 开始, 先开下拉再开菜单 → 菜单全透明)。
func NextPresenceKey() int32 {
	return atomic.AddInt32(&presenceKeySeq, 1)
}

// easePopup 是弹层进出场的缓动(CSS `ease`)。
func easePopup(x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	return popupEase.at(x)
}

var popupEase = newCubicBezier(0.25, 0.1, 0.25, 1.0)

// Presence 驱动 (hOwner, key) 上的一个进出场过渡并返回当前进度(0..1)。
//
//   - visible = true: 进度 0→1(进场); false: 1→0(退场)。
//   - apply 每帧被调用(UI 线程内、定时器回调里), 用它把进度落到元素透明度 / 窗口位置上;
//   - done 到位时回调一次(退场到位后条目自动回收, 不需要自己清状态);
//   - 幂等: 目标不变时只返回当前进度, 可以像 Hover 一样在事件里无脑调用;
//   - **键在同一 hOwner 内必须唯一**(每个逻辑弹层一个键): 已完成的条目会一直留在表里
//     (target=1 不回收), 复用键不会重启过渡、也不会再调 apply, 新窗口就停在自己设的初始
//     状态上。键用 NextPresenceKey() 分配, 不要各组件自己起计数器(见该函数的说明);
//   - apply 在调用时**立即以当前进度执行一次**, 之后每帧再由定时器驱动 —— 这样即使键撞上
//     一条已到位的过渡(过渡不会跑), 目标状态也一定会落地(不会出现全透明窗口);
//   - hOwner 既是过渡值的归属, 也是动画期间的重绘目标(元素或窗口句柄), 每次 tick
//     都会立即重绘它 —— 改动落在**别的窗口**上时也一样(实测: 弹层窗口的透明度要靠
//     宿主窗口的重绘带动合成, 见 DESIGN 第八节 47)。
//
// dur 传 0 时用 PopupDuration; 过渡被 SetHoverDuration(0) 全局关掉时立即到位。
func Presence(hOwner int, key int32, visible bool, dur time.Duration,
	apply, done func(v float64)) float64 {
	if dur <= 0 {
		dur = PopupDuration
	}
	target := 0.0
	if visible {
		target = 1.0
	}
	v := transValue(hOwner, presenceKeyBase-key, target, dur, easePopup, apply, done)
	if apply != nil {
		apply(v)
	}
	return v
}

// SetElementsAlpha 按进度 v(0..1)设置一组元素的透明度, 供对话框 / 弹层做整组淡入淡出。
//
// 元素级透明度是框架在合成阶段乘上去的, 对自绘内容同样有效(实测),
// 因此这是"让一组自绘元素一起淡入"的最省事办法; 注意它不影响窗口本身的不透明度。
//
// ⚠️ 元素透明度**不会级联到子元素**(XEle_SetAlpha 只作用于元素自身的绘制输出,
// 子元素是独立元素、有自己的 alpha, 实测: 对话框只设 overlay 时, 遮罩和面板底淡了,
// 标题/输入框等子元素仍全不透明, 前几帧像"文字浮在弹窗上")。整组进出请用
// SetSubtreeAlpha。
func SetElementsAlpha(v float64, hEles ...int) {
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	a := byte(v*255 + 0.5)
	for _, h := range hEles {
		if h != 0 {
			xc.XEle_SetAlpha(h, a)
		}
	}
}

// SetSubtreeAlpha 按进度 v(0..1)设置一棵元素子树(含根)的透明度。
// 元素透明度不级联(见 SetElementsAlpha 的说明), "整组一起淡"必须显式逐个设置。
func SetSubtreeAlpha(v float64, hRoots ...int) {
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	a := byte(v*255 + 0.5)
	for _, h := range hRoots {
		if h != 0 {
			setAlphaTree(h, a)
		}
	}
}

func setAlphaTree(hEle int, a byte) {
	xc.XEle_SetAlpha(hEle, a)
	n := xc.XEle_GetChildCount(hEle)
	for i := int32(0); i < n; i++ {
		if h := xc.XEle_GetChildByIndex(hEle, i); h != 0 {
			setAlphaTree(h, a)
		}
	}
}

// alphaFloor 是窗口透明度的下限(1/255)。
//
// 实测(2026-10-06): alpha 精确等于 0 会让 xcgui **停止合成该窗口** —— 之后无论把
// 透明度调回多少、怎么重绘, 画面都冻结在停止前的那一帧。表现: 提示气泡退场淡出到
// 0 再进场, 气泡永远停在约 10% 透明度(调试日志显示 apply 已跑到 v=1.0, 像素不动)。
// 因此所有调用统一抬到底, "消失"由调用方在退场 done 里自己隐藏窗口。
const alphaFloor = 0.004

// AlphaFloor 返回窗口透明度的下限(1/255, 见 SetWindowAlpha 的说明)。
// 进场前把弹层窗口压到这个值可消除首帧闪现, 又不会触发 alpha=0 的合成停摆。
func AlphaFloor() float64 { return alphaFloor }

// SetWindowAlpha 按进度 v(0..1)设置窗口整体透明度。
//
// 只对创建时带透明类型的窗口有效(异型 / 阴影 / 圆角); 普通窗口
// XWnd_SetTransparentAlpha 无效(实测: 只设置 alpha 不改变窗口外观, 也不会自动加上
// WS_EX_LAYERED), 这类窗口只能拿位置做动画。
//
// 注意: 透明度在窗口**绘制时**才落地 —— 改完不重绘该窗口, 画面会一直停在
// 上一次绘制时的透明度上(实测踩过)。用 FadeWindowAlpha / SlideWindowAlpha 就带重绘。
func SetWindowAlpha(hWindow int, v float64) {
	if v < alphaFloor {
		v = alphaFloor
	} else if v > 1 {
		v = 1
	}
	xc.XWnd_SetTransparentAlpha(hWindow, byte(v*255+0.5))
}

// FadeWindowAlpha 返回一个给 Presence 用的 apply 回调: 按进度设置窗口透明度,
// 并立即重绘该窗口(透明度要绘制才可见)。
func FadeWindowAlpha(hWindow int) func(v float64) {
	return func(v float64) {
		SetWindowAlpha(hWindow, v)
		xc.XWnd_Redraw(hWindow, true)
	}
}

// SlideWindowAlpha 返回一个给 Presence 用的 apply 回调: 按进度把窗口从
// (x+dx, y+dy) 滑到 (x, y), 同时把窗口透明度设为 alpha×v(见 SetWindowAlpha 的限制)。
//
// 坐标是**物理像素**(XWnd_SetWindowPos 直通 Win32, 与 Tooltip 的定位一致)。
func SlideWindowAlpha(hWindow int, x, y, dx, dy int32, alpha byte) func(v float64) {
	return func(v float64) {
		ox := x + int32(float64(dx)*(1-v)+0.5)
		oy := y + int32(float64(dy)*(1-v)+0.5)
		SetWindowAlpha(hWindow, v*float64(alpha)/255)
		xc.XWnd_SetWindowPos(hWindow, xcc.HWND_TOPMOST, ox, oy, 0, 0,
			xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE|xcc.SWP_SHOWWINDOW)
		xc.XWnd_Redraw(hWindow, true)
	}
}
