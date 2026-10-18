package xui

import (
	"sync"
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
)

// ===== hover 过渡动画 =====
//
// 把"鼠标进出引起的颜色跳变"变成过渡: 默认 150ms + cubic-bezier(0.4, 0, 0.2, 1)
// —— 与 tailwind / shadcn 的 transition-*(transition-all / transition-colors)
// 默认值一致(见 shadcn/src/components/ui/*.tsx)。
//
// 驱动: 挂在**重绘目标**上的炫彩定时器(XM_TIMER, 见 animTimerID)。只有该目标上
// 还有没走完的过渡时才开着, 全部到位立刻自己关掉 —— 静止时零开销、不占 CPU。
// 实测(探针, 2026-10-06): 定时器请求 16ms、真实回调约 30ms/帧(请求 1ms/4ms 也一样,
// xcgui 内部有帧间隔), 且必须是**立即重绘**才吃得到这个帧率(走 Redraw(false) 的
// 延迟重绘会被合并, 掉到约 65ms/帧) → 150ms 过渡约 5 帧, 颜色插值的单帧跨度小,
// 观感足够平滑。
//
// 组件用法(绘制回调里, 每帧都要调用):
//
//	hv := xui.Hover(hEle, st.Hover)          // 元素级(原生 Stay 标志或自跟踪)
//	hv := xui.HoverItem(t.Handle, i, stay)   // 逐项(原生 List/Tree 的行/条目)
//	bg = xui.Fade(bgRest, bgHover, hv)       // 颜色插值
//	p.FillRound(&rc, t.Accent.ScaleAlpha(hv), r)  // 只是"叠加层"时直接压透明度
//
// 为什么不把过渡值存在组件字段里: 组件源码是拷贝分发给用户的, 过渡状态属于
// 运行时该管的脏活(定时器、事件、生命周期), 放运行时组件文件才能保持写薄。
//
// 中断语义与 CSS 相同: 目标反向时从**当前插值位置**继续, 时长仍是完整的
// 150ms(不是剩余时间)。

const (
	// animTimerID 是过渡动画占用的炫彩定时器 ID。炫彩定时器是 xcgui 自己的
	// 命名空间(XM_TIMER, 与 Win32 定时器无关), 用户若在同一元素/窗口上自用
	// 炫彩定时器, 避开这个 ID 即可。
	animTimerID uint32 = 0x7875
	// animInterval 是逐帧回调的请求间隔(毫秒)。
	animInterval uint32 = 16
	// defaultHoverDuration 对齐 tailwind 的 --default-transition-duration(150ms)。
	defaultHoverDuration = 150 * time.Millisecond
	// itemKeyBase 是逐项过渡的键基数: 键 0 留给元素级 hover, 行索引整体上抬。
	itemKeyBase int32 = 1
)

// hoverItem 是一个过渡值: 在 from → target 之间按缓动插值。
// hover 用它的 0↔1; 弹层/对话框进出场(presence)也复用它, 只是多挂 apply/done 回调。
type hoverItem struct {
	cur    float64
	from   float64
	target float64
	start  time.Time
	// settled 记录"已到位回调已发过", 避免被同 owner 上其它过渡拖着的定时器重复触发。
	settled bool
	// 可选覆盖: 条目自己的时长与缓动(0/nil 用全局 hover 时长与默认缓动)
	dur  time.Duration
	ease func(float64) float64
	// 可选回调(都在定时器回调里、解锁之后执行)
	apply func(v float64) // 每帧
	done  func(v float64) // 到位一次
}

// hoverOwner 是一个重绘目标(元素或窗口)上的全部过渡值。
type hoverOwner struct {
	hOwner  int
	running bool
	items   map[int32]*hoverItem
}

var hovers = struct {
	mu       sync.Mutex
	duration time.Duration
	// scale 是过渡时长的倍率(1 = 原速): 供调试 / 截图抓中间帧放大用, 0 = 立即到位。
	scale  float64
	owners map[int]*hoverOwner
}{duration: defaultHoverDuration, scale: 1, owners: map[int]*hoverOwner{}}

// SetAnimationScale 按倍率缩放**全部**过渡时长(hover 与弹层进出场都受影响):
// 1 = 原速, 0 = 立即到位(等于关掉全部过渡), 10 = 慢十倍。
//
// 除了"低配机器 / 无障碍关动画", 它主要用于调试: 把 100~150ms 的动画放慢后,
// 外部截图脚本才能稳定抓到中间帧做像素验收(见 DESIGN 第八节 47)。
// 必须在 UI 线程调用(与其它 UI 操作同一约束)。
func SetAnimationScale(f float64) {
	if f < 0 {
		f = 0
	}
	hovers.mu.Lock()
	hovers.scale = f
	hovers.mu.Unlock()
}

// AnimationScale 返回当前过渡时长倍率。
func AnimationScale() float64 {
	hovers.mu.Lock()
	defer hovers.mu.Unlock()
	return hovers.scale
}

// SetHoverDuration 覆盖 hover 过渡时长(默认 150ms)。
// 传 0 表示关闭过渡: 状态变化立即到位, 不启动定时器(低配机器 / 自动化对比可用)。
// 必须在 UI 线程调用(与其它 UI 操作同一约束)。
func SetHoverDuration(d time.Duration) {
	hovers.mu.Lock()
	hovers.duration = d
	hovers.mu.Unlock()
}

// HoverDuration 返回当前 hover 过渡时长(0 表示已关闭过渡)。
func HoverDuration() time.Duration {
	hovers.mu.Lock()
	defer hovers.mu.Unlock()
	return hovers.duration
}

// Hover 取 hOwner 上"元素级"hover 的过渡值(0..1), 并把目标设为 active。
// hOwner 既是过渡状态的归属(按句柄登记), 也是动画期间的重绘目标。
func Hover(hOwner int, active bool) float64 {
	return hoverValue(hOwner, 0, active)
}

// HoverItem 取 hOwner 上第 index 项的 hover 过渡值(0..1), 用于"逐项悬停"的
// 载体(原生 List / Tree / ListBox 的行与条目)。active 一般直接来自绘制回调里
// 观察到的项状态(如 List_Item_State_Stay)。
func HoverItem(hOwner int, index int32, active bool) float64 {
	return hoverValue(hOwner, index+itemKeyBase, active)
}

func hoverValue(hOwner int, key int32, active bool) float64 {
	target := 0.0
	if active {
		target = 1.0
	}
	return transValue(hOwner, key, target, 0, nil, nil, nil)
}

// transValue 是过渡内核: 把 (hOwner,key) 上的目标设为 target(0..1) 并返回当前值。
// dur>0 / ease!=nil 覆盖该条目的时长与缓动; apply/done 是可选回调(语义见 hoverTick)。
func transValue(hOwner int, key int32, target float64, dur time.Duration,
	ease func(float64) float64, apply, done func(float64)) float64 {
	now := time.Now()
	hovers.mu.Lock()
	defer hovers.mu.Unlock()

	o := hovers.owners[hOwner]
	if o == nil {
		if target == 0 {
			// 没活跃过又不活跃: 不登记任何状态(大表格里只 hover 过的行才有条目)
			return 0
		}
		o = newHoverOwner(hOwner)
		hovers.owners[hOwner] = o
	}
	it := o.items[key]
	if it == nil {
		if target == 0 {
			return 0
		}
		it = &hoverItem{}
		o.items[key] = it
	}
	it.dur, it.ease, it.apply, it.done = dur, ease, apply, done
	if it.target != target {
		// 从当前位置续跑(不是从 0/1 重来), 与 CSS 中断过渡的语义一致
		it.from = it.value(now, hovers.duration)
		it.target = target
		it.start = now
		it.settled = false
		o.ensureRunning()
	}
	return it.value(now, hovers.duration)
}

// value 返回当前值。注意: 它会把已算出的值缓存进 cur(纯优化, 不影响结果)。
func (it *hoverItem) value(now time.Time, dur time.Duration) float64 {
	if it.cur == it.target {
		return it.cur
	}
	if it.dur > 0 {
		dur = it.dur
	}
	dur = time.Duration(float64(dur) * hovers.scale)
	if dur <= 0 || it.start.IsZero() {
		return it.target // 关闭过渡 / 尚未启动
	}
	el := now.Sub(it.start)
	if el >= dur {
		it.cur = it.target
		return it.cur
	}
	x := float64(el) / float64(dur)
	if it.ease != nil {
		it.cur = it.from + (it.target-it.from)*it.ease(x)
		return it.cur
	}
	it.cur = it.from + (it.target-it.from)*easeTransition(x)
	return it.cur
}

// ensureRunning 确保驱动定时器已开(调用方持有 hovers.mu)。
func (o *hoverOwner) ensureRunning() {
	if o.running {
		return
	}
	o.running = true
	setHoverTimer(o.hOwner, true)
}

// newHoverOwner 登记一个重绘目标并挂上定时器事件(只挂一次, 走事件总线, 不占
// syscall 回调上限)。
func newHoverOwner(hOwner int) *hoverOwner {
	o := &hoverOwner{hOwner: hOwner, items: map[int32]*hoverItem{}}
	if xc.XC_IsHELE(hOwner) {
		if e := widget.NewElementByHandle(hOwner); e != nil {
			e.AddEvent_XC_Timer(func(hEle, nID int, pbHandled *bool) int {
				if nID == int(animTimerID) {
					hoverTick(hEle)
				}
				return 0
			})
			// 销毁后句柄可能被复用: 清掉登记, 避免把旧过渡值带给新元素
			e.AddEvent_Destroy_End(func(hEle int, pbHandled *bool) int {
				dropHoverOwner(hEle)
				return 0
			})
		}
	} else if w := window.NewByHandle(hOwner); w != nil {
		w.AddEvent_XC_Timer(func(hWindow int, nTimerID uint, pbHandled *bool) int {
			if nTimerID == uint(animTimerID) {
				hoverTick(hWindow)
			}
			return 0
		})
	}
	return o
}

// dropHoverOwner 忘掉某个句柄上的过渡状态。
func dropHoverOwner(hOwner int) {
	hovers.mu.Lock()
	o, ok := hovers.owners[hOwner]
	if ok {
		delete(hovers.owners, hOwner)
		if o.running {
			o.running = false
		}
	}
	hovers.mu.Unlock()
	if ok {
		// 定时器可能还挂着(条目被中途清掉时, hoverTick 不会走到关定时器的分支),
		// 不关的话每 16ms 空转一次回调。
		setHoverTimer(hOwner, false)
	}
}

// DropTransitions 丢弃 hOwner 上**全部**过渡条目(元素级与窗口级都适用)。
//
// 供"宿主生命周期结束 / 弹层被隐藏"的调用方清场: 淡入中途隐藏时, 未到位条目会留在
// 表里(未 settled), 它们的 apply 会随残留定时器继续执行, 与下一次进场的 apply 打架
// (2026-10-18 时间框弹层概率性停在半透明: 淡入中途 hide 后快速重开, fast 路径的
// 终态与旧条目的 apply 互相覆盖)。调用方负责把相关元素的 alpha 归位到期望的静止态。
func DropTransitions(hOwner int) {
	if hOwner != 0 {
		dropHoverOwner(hOwner)
	}
}

// hoverTick 推进一帧: 所有过渡值前进, 还差得远的继续重绘, 全部到位就关定时器。
func hoverTick(hOwner int) {
	now := time.Now()
	hovers.mu.Lock()
	o := hovers.owners[hOwner]
	if o == nil {
		hovers.mu.Unlock()
		return
	}
	dur := hovers.duration
	active := false
	type callback struct {
		fn func(float64)
		v  float64
	}
	var applies, dones []callback
	for k, it := range o.items {
		v := it.value(now, dur)
		// 到位条目(target=1 且已 settled)不再调 apply —— 它们虽然留在表里不回收
		// (presence 的键空间设计, 见 presence.go), 但每帧继续执行旧 apply 会和新
		// 过渡打架: 同一宿主窗口上第二次弹出弹层时, 旧条目的 apply(v=1.0) 把窗口
		// alpha 拉回全亮, 新条目的 apply(v) 又拉低, 交替 = 弹层闪烁(2026-10-18
		// 用户报障: 日期框第一次弹出不闪, 第二次起闪)。"到位后条目静默"本就是
		// presence 的设计意图, 这里补上实现; 反向过渡(settled 被重置)不受影响。
		if it.apply != nil && !it.settled {
			applies = append(applies, callback{it.apply, v})
		}
		if v != it.target {
			active = true
			continue
		}
		if !it.settled {
			it.settled = true
			if it.done != nil {
				dones = append(dones, callback{it.done, v})
			}
		}
		if it.target == 0 {
			// 淡出到位: 回收条目(大列表不会随滚动累积索引)
			delete(o.items, k)
		}
	}
	if !active {
		o.running = false
		setHoverTimer(hOwner, false)
	}
	hovers.mu.Unlock()

	// 回调与重绘一律在解锁之后执行(apply 里要碰 UI, 不能持锁)
	for _, c := range applies {
		c.fn(c.v)
	}
	if active {
		if xc.XC_IsHELE(hOwner) {
			xc.XEle_Redraw(hOwner, true)
		} else {
			// 立即重绘: 走延迟重绘会被合并掉(实测 30ms/帧 → 65ms/帧, 见 DESIGN 第八节 45)
			xc.XWnd_Redraw(hOwner, true)
		}
	}
	for _, c := range dones {
		c.fn(c.v)
	}
}

// setHoverTimer 开/关驱动定时器(hOwner 是元素句柄或窗口句柄)。
func setHoverTimer(hOwner int, on bool) {
	if xc.XC_IsHELE(hOwner) {
		if on {
			xc.XEle_SetXCTimer(hOwner, animTimerID, animInterval)
		} else {
			xc.XEle_KillXCTimer(hOwner, animTimerID)
		}
		return
	}
	if on {
		xc.XWnd_SetXCTimer(hOwner, animTimerID, animInterval)
	} else {
		xc.XWnd_KillXCTimer(hOwner, animTimerID)
	}
}

// Fade 按 t(0~1)在 a(静止态)与 b(悬停态)之间插值, 供 hover 过渡使用。
//
// 与 Mix 的差别是**在预乘 alpha 空间插值**(与浏览器 CSS transition 的做法一致):
// a=None(全透明, RGB 记作 0) → b=Muted 这类"淡入", 直接线性插值会先经历"半透明的
// 黑灰"(先变脏再变亮), 预乘后是干净的透明度渐变。两端素都是不透明时两者等价。
func Fade(a, b Color, t float64) Color {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	outA := float64(aa) + (float64(ba)-float64(aa))*t
	if outA <= 0 {
		return None
	}
	// 预乘分量线性插值后反预乘
	ch := func(av, bv byte) byte {
		pre := float64(av)*float64(aa)/255*(1-t) + float64(bv)*float64(ba)/255*t
		v := pre * 255 / outA
		if v < 0 {
			v = 0
		} else if v > 255 {
			v = 255
		}
		return byte(v + 0.5)
	}
	return RGBA(ch(ar, br), ch(ag, bg), ch(ab, bb), byte(outA+0.5))
}

// ===== 缓动 =====

// hoverEase 是 tailwind / shadcn 的默认缓动 cubic-bezier(0.4, 0, 0.2, 1)。
var hoverEase = newCubicBezier(0.4, 0.0, 0.2, 1.0)

// easeTransition 求该缓动曲线在进度 x 处的输出(0..1)。
func easeTransition(x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	return hoverEase.at(x)
}

// cubicBezier 是三次贝塞尔缓动曲线(端点固定为 (0,0)/(1,1), 只给两个控制点),
// 与 CSS transition-timing-function 的 cubic-bezier() 同构。
type cubicBezier struct {
	ax, bx, cx float64
	ay, by, cy float64
}

func newCubicBezier(x1, y1, x2, y2 float64) cubicBezier {
	cx := 3 * x1
	bx := 3*(x2-x1) - cx
	ax := 1 - cx - bx
	cy := 3 * y1
	by := 3*(y2-y1) - cy
	ay := 1 - cy - by
	return cubicBezier{ax: ax, bx: bx, cx: cx, ay: ay, by: by, cy: cy}
}

// at 求曲线在横坐标 x 处的纵坐标: 先解 x(t)=x 得参数 t(牛顿迭代, 与浏览器
// 的实现同法), 再算 y(t)。曲线的 x 分量单调, 迭代收敛很快。
func (b cubicBezier) at(x float64) float64 {
	t := x
	for i := 0; i < 8; i++ {
		dx := (3*b.ax*t+2*b.bx)*t + b.cx // x'(t)
		if dx == 0 {
			break
		}
		t -= ((((b.ax*t+b.bx)*t)+b.cx)*t - x) / dx
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
	}
	return ((b.ay*t+b.by)*t + b.cy) * t
}
