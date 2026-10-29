package ui

import (
	"math"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 提示气泡的几何常量(对应 shadcn tooltip.tsx: rounded-md px-3 py-1.5 text-xs + size-2.5 旋转方块箭头)
const (
	tipPadX int32 = 12 // px-3
	tipPadY int32 = 6  // py-1.5
	// tipArrow 既是箭头露出的三角形高度, 也是气泡与目标元素之间的间距 ——
	// 箭头(45° 菱形)恰好把这段间距填满, 尖端顶到元素边缘, 所以两者必须相等
	// (sideOffset = 0 时)。菱形对角线的另一半被气泡盖住, 因此露出的三角形底宽 = 2*tipArrow。
	//
	// 取值 = shadcn 的 size-2.5(10) 旋转 45° 后的**露出高度** 10/√2 ≈ 7: 之前用 10 时,
	// 底宽 20 的三角形在单行气泡(高 28)里占了 71%, 看起来是一个糊满整条边的大三角。
	tipArrow  int32 = 7
	tipMaxW   int32 = 280 // max-w-xs
	tipRadius int32 = 6   // rounded-md
	// 内容里带快捷键(Kbd)时: gap-1.5(6) 与 has-data-[slot=kbd]:pr-1.5(右内边距 6)
	tipKbdGap int32 = 6
	tipKbdH   int32 = 16
)

// TipSide 是气泡相对触发元素的方向, 对应 radix TooltipContent 的 side。
type TipSide int

const (
	TipAuto TipSide = iota // 上方优先, 上方放不下翻到下方(默认)
	TipTop
	TipBottom
	TipLeft
	TipRight
)

// TipAlign 是气泡在侧向轴上的对齐方式, 对应 radix TooltipContent 的 align。
type TipAlign int

const (
	TipAlignCenter TipAlign = iota // 与元素居中对齐(默认)
	TipAlignStart                  // 气泡起点贴元素起点
	TipAlignEnd                    // 气泡终点贴元素终点
)

// tipConfig 是单个绑定的显示配置(每个元素可以有自己的方向/延迟/快捷键)。
type tipConfig struct {
	side     TipSide
	align    TipAlign
	offset   int32 // 气泡与元素之间的额外间距(sideOffset)
	delay    int32 // 显示延迟(ms); -1 = 跟随 Tooltip.SetDelay
	shortcut string
	maxWidth int32
}

// TipOption 是单条工具提示的配置项。
type TipOption func(*tipConfig)

// WithTipSide 指定气泡方向(默认 TipAuto: 上方优先, 放不下翻下方)。
func WithTipSide(s TipSide) TipOption { return func(c *tipConfig) { c.side = s } }

// WithTipAlign 指定气泡在侧向轴上的对齐方式(默认 TipAlignCenter)。
func WithTipAlign(a TipAlign) TipOption { return func(c *tipConfig) { c.align = a } }

// WithTipOffset 在元素与气泡之间再加一段间距(对应 radix 的 sideOffset, 默认 0)。
// 默认 0 时箭头正好填满间距、尖端顶住元素边缘。
func WithTipOffset(px int32) TipOption { return func(c *tipConfig) { c.offset = px } }

// WithTipDelay 覆盖该元素的显示延迟(ms); 0 = 立即显示。
func WithTipDelay(ms int32) TipOption { return func(c *tipConfig) { c.delay = ms } }

// WithTipShortcut 在内容右侧显示一个快捷键块(对应文档里 tooltip 内容中嵌 Kbd 的示例)。
func WithTipShortcut(s string) TipOption { return func(c *tipConfig) { c.shortcut = s } }

// WithTipMaxWidth 覆盖气泡最大宽度(默认 280, 即 max-w-xs); 超宽时文字折行。
func WithTipMaxWidth(px int32) TipOption { return func(c *tipConfig) { c.maxWidth = px } }

// Tooltip 提示气泡: 原生控件没有提示部件, 这里自建一个无边框置顶小窗承载。
//
// 与 shadcn tooltip.tsx(radix-nova) 对齐:
//   - 内容: bg-foreground rounded-md px-3 py-1.5 text-xs text-background, 贴边一侧加箭头;
//     内容右侧可挂一个快捷键块(Kbd), 此时右内边距收到 6;
//   - 触发: 鼠标进入元素即显示(nova 预设 delayDuration = 0), 离开即隐藏;
//     键盘焦点进入也显示(文档定义: "获得键盘焦点或鼠标悬停时显示")。
//   - 位置: 默认在元素上方水平居中, 上方放不下时翻到下方; 可用 WithTipSide 指定
//     上/下/左/右四方向, WithTipAlign 指定对齐, WithTipOffset 加间距;
//     贴元素那一侧的箭头始终指向触发元素中心(超出圆角时夹回气泡内)。
//
// 窗口为异型透明窗口(Window_Transparent_Shaped): 只有气泡与箭头不透明, 其余区域
// 真透明, 覆盖在别的元素上时不会遮挡下层内容; 再配合 WS_EX_NOACTIVATE|WS_EX_TRANSPARENT
// 实现不进任务栏、不抢焦点、鼠标穿透(气泡可能压在鼠标位置, 穿透后目标元素不会因为
// "鼠标离开"而把提示关掉)。
type Tooltip struct {
	win      *window.Window
	ele      *widget.Element
	tips     map[int]string
	cfg      map[int]tipConfig
	bound    map[int]bool
	hostOf   map[int]int  // 元素 -> 宿主 xcgui 窗口(窗口级命中测试用)
	winWatch map[int]bool // 已挂"窗口级鼠标移动"回调的窗口
	hosts    map[int]bool // 已挂"宿主销毁 → 销毁气泡窗"清理的宿主窗口
	dying    map[int]bool // 正在销毁的宿主: 拒绝迟到的鼠标事件再显示气泡
	text     string
	side     TipSide // 当前实际显示方向(翻转后的结果)
	arrow    int32   // 箭头在贴元素那条边上的中心(窗口内坐标: 上下时是 x, 左右时是 y)

	shown bool // 气泡当前是否处于显示(或正在淡出)状态
	hWnd  int  // 触发元素所在的宿主窗口(进出场过渡挂它上面, 见下)
	// shownEle 是当前气泡"描述"的那个元素: 密集元素之间快速划过时, 旧元素的
	// MouseLeave 可能晚于新元素的 MouseStay 到达(实测: 相邻图标格之间移动),
	// 此时按元素过滤掉这条迟到的 leave, 否则刚显示的提示会被立刻关掉。
	shownEle int

	// 延迟: dashDelay 对应 radix Provider 的 delayDuration, skip 对应 skipDelayDuration
	// (移出后短时间内移到下一个元素不再等延迟)。
	dashDelay int32
	skip      int32
	lastHide  time.Time
	pending   *time.Timer // 延迟显示的计时器(有它就表示正在排队, 再次悬停不重复起表)
	// fadeKey 是进出场过渡的键: 必须**每个实例一个**(见 NextPresenceKey)。用固定常量会让
	// 同一宿主窗口上的多个 Tooltip 抢同一条过渡 —— 后显示的那个复用已到位的键, 过渡不再跑,
	// 气泡就停在元素 alpha 的初始值(半透明), 看起来"气泡是灰的"。
	fadeKey int32
	// fadeGen 是过渡代际: 每次 Show/Hide 自增。上一次进场的每帧回调捕获的是旧的 px/py,
	// 若它在切换目标后还在跑, 会把窗口**拉回旧位置**(实测: 200ms 内接着 Show 另一个
	// 元素的提示, 新位置被旧帧覆盖, 气泡停在上一个元素旁边)。过期代际的回调直接返回。
	fadeGen int
}

// NewTooltip 创建提示管理器(窗口懒创建, 首次显示时才建)。
func NewTooltip() *Tooltip {
	return &Tooltip{
		tips:     map[int]string{},
		cfg:      map[int]tipConfig{},
		bound:    map[int]bool{},
		hostOf:   map[int]int{},
		winWatch: map[int]bool{},
		hosts:    map[int]bool{},
		dying:    map[int]bool{},
		skip:     300,
		fadeKey:  xui.NextPresenceKey(),
	}
}

// SetDelay 设置默认显示延迟(ms, 默认 0 = 悬停即显示, 与 shadcn nova 预设一致)。
func (t *Tooltip) SetDelay(ms int32) *Tooltip {
	if t == nil {
		return t
	}
	t.dashDelay = ms
	return t
}

// SetSkipDelay 设置"移出后多久内移到下一个元素可跳过延迟"(ms, 默认 300;
// 对应 radix TooltipProvider 的 skipDelayDuration; 设为 0 表示每次都等延迟)。
func (t *Tooltip) SetSkipDelay(ms int32) *Tooltip {
	if t == nil {
		return t
	}
	t.skip = ms
	return t
}

// Bind 把提示绑定到元素: 鼠标进入显示, 离开隐藏。同一个 Tooltip 可绑定任意多个元素。
func (t *Tooltip) Bind(hEle int, text string) *Tooltip {
	return t.BindTip(hEle, text)
}

// BindTip 是带配置的 Bind: 每个元素可单独指定方向/对齐/间距/延迟/快捷键。
func (t *Tooltip) BindTip(hEle int, text string, opts ...TipOption) *Tooltip {
	if t == nil {
		return t
	}
	cfg := tipConfig{delay: -1}
	for _, o := range opts {
		o(&cfg)
	}
	t.tips[hEle] = text
	t.cfg[hEle] = cfg
	if t.bound[hEle] {
		return t
	}
	tooltipReg[hEle] = t
	w := widget.NewElementByHandle(hEle)
	if w == nil {
		return t
	}
	t.bound[hEle] = true
	// 窗口级兜底(见 hitTest 的说明): ScrollArea(LayoutFrame) 里的元素收不到元素级
	// MouseMove/MouseStay, 只靠下面这些回调, 页面里所有提示都不会弹。
	if hWnd := windowOf(hEle); hWnd != 0 {
		t.hostOf[hEle] = hWnd
		t.watchWindow(hWnd)
	}
	// 元素销毁时清掉登记, 否则句柄被复用后会把提示挂到别的元素上。
	w.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		delete(t.tips, hEle)
		delete(t.cfg, hEle)
		delete(t.bound, hEle)
		delete(t.hostOf, hEle)
		delete(tooltipReg, hEle)
		if t.shownEle == hEle {
			t.Hide()
		}
		return 0
	})
	w.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		t.request(hEle)
		return 0
	})
	w.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		t.request(hEle)
		return 0
	})
	w.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		// 只隐藏"当前气泡描述的元素"的离开: 鼠标在密集元素之间快速划过时, 旧元素的
		// MouseLeave 可能晚于新元素的 MouseStay/MouseMove 到达(实测: 相邻图标格),
		// 那条迟到的 leave 会把刚进场的气泡立刻关掉, 表现成"提示不显示"。
		if hEle == t.shownEle {
			t.Hide()
			return 0
		}
		t.cancelPending()
		return 0
	})
	// 键盘焦点进入也显示(文档语义: 焦点或悬停), 鼠标点击产生的焦点不算 ——
	// 否则点完按钮气泡一直挂着, 直到点别处才消失。
	w.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		if xui.FocusedByKeyboard() {
			t.request(hEle)
		}
		return 0
	})
	w.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		if hEle == t.shownEle {
			t.Hide()
		} else {
			t.cancelPending()
		}
		return 0
	})
	return t
}

// Unbind 解除绑定。
func (t *Tooltip) Unbind(hEle int) {
	delete(t.tips, hEle)
	delete(t.cfg, hEle)
	delete(t.bound, hEle)
	delete(t.hostOf, hEle)
	delete(tooltipReg, hEle)
	if t.shownEle == hEle {
		t.Hide()
	}
}

// watchWindow 给宿主窗口挂一次窗口级鼠标移动回调(每个窗口只挂一次)。
func (t *Tooltip) watchWindow(hWnd int) {
	if hWnd == 0 || t.winWatch[hWnd] {
		return
	}
	w := window.NewByHandle(hWnd)
	if w == nil {
		return
	}
	t.winWatch[hWnd] = true
	w.AddEvent_MouseMove(func(hWin int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		t.hitTest(hWnd, pPt)
		return 0
	})
}

// hitTest 是窗口级命中测试: 光标在窗口里移动时, 自己判断它压在哪个已绑定元素上。
//
// 为什么需要它: **ScrollArea(LayoutFrame) 内部的元素收不到元素级 MouseMove/MouseStay**
// (实测: 同一窗口的窗口级回调收到 2 次移动, 滚动区里的按钮计数器仍是 0), 而桌面端
// 页面内容几乎全在滚动区里 —— 只靠元素事件会出现"整个页面没有一个 tooltip 会弹"。
// 窗口级回调给的是客户区坐标, 与 XEle_GetWndClientRect 同一坐标系, 直接做矩形包含即可。
func (t *Tooltip) hitTest(hWnd int, pt *xc.POINT) {
	if t == nil || pt == nil {
		return
	}
	for hEle := range t.tips {
		if t.tips[hEle] == "" || t.hostOf[hEle] != hWnd {
			continue
		}
		if !xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
			continue
		}
		var rc xc.RECT
		xc.XEle_GetWndClientRect(hEle, &rc)
		if pt.X < rc.Left || pt.X >= rc.Right || pt.Y < rc.Top || pt.Y >= rc.Bottom {
			continue
		}
		// 矩形命中了还要确认真的可见: 分页容器里**隐藏页面的元素**矩形仍然保留在原位,
		// 不判可见性会命中最底层页面里同位置的元素, 弹出完全不相干的文字(实测: 悬停
		// 提示气泡卡的按钮, 弹出来的是图标页某个图标的名称)。
		if !widgetVisible(hEle) {
			continue
		}
		t.request(hEle)
		return
	}
	// 光标不在任何绑定元素上: 该关的就关(拖动/滚动时也能及时消失)
	if t.shownEle != 0 || t.pending != nil {
		t.Hide()
	}
}

// HideTooltip 隐藏绑定在指定元素上的提示气泡(未绑定则无操作)。
//
// 为什么要主动隐藏: 元素自身弹层(如下拉列表)打开后是另一个窗口接管鼠标, 触发元素
// 收不到 MouseLeave, 提示会一直挂着(实测 ComboBox 弹层)。这类组件应在弹层打开时
// 调用 HideTooltip(hEle)。
var tooltipReg = map[int]*Tooltip{}

func HideTooltip(hEle int) {
	if tt, ok := tooltipReg[hEle]; ok {
		tt.Hide()
	}
}

// IsOpen 报告气泡当前是否显示在该元素上。
func (t *Tooltip) IsOpen(hEle int) bool {
	return t != nil && t.shown && t.shownEle == hEle
}

// Show 程序化显示某个已绑定元素的提示(受控用法; 立即显示, 不受延迟影响)。
func (t *Tooltip) Show(hEle int) *Tooltip {
	if t == nil {
		return t
	}
	if t.tips[hEle] == "" {
		return t
	}
	t.cancelPending()
	t.showNow(hEle)
	return t
}

// Hide 隐藏提示: 退场淡出(150ms)后再真正隐藏窗口。
func (t *Tooltip) Hide() {
	t.cancelPending()
	if t.win == nil || !t.shown {
		return
	}
	t.shown = false
	t.lastHide = time.Now()
	// 退场: fade-out-0(shadcn 的 tooltip 退场没有位移), 到位后隐藏窗口。
	// 过渡挂在**触发元素所在的宿主窗口**上: 气泡自己的异型置顶小窗做重绘目标时,
	// xcgui 的定时器实测会掉到约 100ms/帧(见 DESIGN 第八节 47), 宿主窗口这条
	// 驱动链与 hover 相同(约 16~30ms/帧), 因此把驱动放在它上面。
	win := t.win
	t.fadeGen++
	gen := t.fadeGen
	xui.Presence(t.hWnd, t.fadeKey, false, xui.TooltipDuration,
		func(v float64) {
			if t.fadeGen != gen {
				return
			}
			t.fade(v)
		}, func(v float64) {
			// 宿主窗口销毁时气泡窗已经被清理掉(见 watchHost), 此时不能再碰旧句柄;
			// 另起新窗后旧过渡也不该把新窗藏掉。
			if t.win == win && t.fadeGen == gen {
				win.ShowWindow(xcc.SW_HIDE)
			}
		})
}

// cancelPending 取消尚未到点的延迟显示。
func (t *Tooltip) cancelPending() {
	if t.pending != nil {
		t.pending.Stop()
		t.pending = nil
	}
}

// request 是"该显示了"的统一入口: 按配置与 skipDelay 决定立即显示还是排队。
func (t *Tooltip) request(hEle int) {
	if t == nil || t.dying[windowOf(hEle)] {
		return
	}
	if t.shown && t.shownEle == hEle {
		return // 已经显示(在目标之间横移只需挪位置)
	}
	if t.tips[hEle] == "" {
		return
	}
	delay := t.dashDelay
	if c, ok := t.cfg[hEle]; ok && c.delay >= 0 {
		delay = c.delay
	}
	// skipDelay: 刚关掉一个气泡, 紧接着移到下一个元素时立即显示(radix 默认 300ms)。
	if t.skip > 0 && !t.lastHide.IsZero() && time.Since(t.lastHide) < time.Duration(t.skip)*time.Millisecond {
		delay = 0
	}
	if delay <= 0 {
		t.cancelPending()
		t.showNow(hEle)
		return
	}
	if t.pending != nil {
		return // 已在排队, 不要重复起表(悬停时 MouseMove 会连续触发)
	}
	t.pending = app.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
		t.pending = nil
		t.showNow(hEle)
	})
}

// fade 是气泡进出场的每帧回调: 按进度设置气泡的不透明度并立即重绘。
//
// 淡的是**元素** alpha(XEle_SetAlpha), 不是整窗 alpha(XWnd_SetTransparentAlpha):
// 异型透明窗(Window_Transparent_Shaped)的整窗透明度**只认第一次调用**, 之后再设什么
// 值、再怎么重绘(本窗 / 元素 / 宿主窗)都不再变化 —— 实测现象是"窗口位置在动、
// layered alpha 恒为首次那个值", 于是气泡整块几乎是全透明的, 屏幕上根本看不到它。
// 元素 alpha 是框架在合成阶段乘上去的, 每帧都生效(对话框整组淡入同理, 见 DESIGN 第八节 47/57)。
func (t *Tooltip) fade(v float64) {
	if t.win == nil || t.ele == nil {
		return
	}
	xui.SetElementsAlpha(v, t.ele.Handle)
	t.win.Redraw(true) // 立即重绘: 元素 alpha 要画一遍才落地
}

// measure 量气泡的内容尺寸与窗口尺寸(逻辑像素)。
// 文字按绘制时同一字体与限宽量, 否则气泡会比文字宽一圈(此前量的是 14px 主题字体,
// 而实际绘制用 12px)。返回气泡宽高、窗口宽高、右内边距与快捷键块宽。
func (t *Tooltip) measure(text string, cfg tipConfig) (bw, bh, tipRight, kbdW int32) {
	maxW := cfg.maxWidth
	if maxW <= 0 {
		maxW = tipMaxW
	}
	f := xui.FontHandle(xui.FontSpec{
		Family: xui.Current().FontDefault.Family,
		Size:   xui.Current().FontDefault.Size - 2, // text-xs
	})
	tipRight = tipPadX
	if cfg.shortcut != "" {
		tipRight = 6 // has-data-[slot=kbd]:pr-1.5
		if f != 0 {
			kbdW = xui.TextWidth(cfg.shortcut, f) + 8 // px-1
		}
	}
	maxTextW := maxW - tipPadX - tipRight
	if kbdW > 0 {
		maxTextW -= tipKbdGap + kbdW
	}
	if maxTextW < 40 {
		maxTextW = 40
	}
	var sz xc.SIZE
	app.GetTextShowRect(text, -1, f, 0, maxTextW, &sz)
	bw = sz.CX + tipPadX + tipRight
	if kbdW > 0 {
		bw += tipKbdGap + kbdW
	}
	if bw > maxW {
		bw = maxW
	}
	bh = sz.CY + tipPadY*2
	if kbdW > 0 && bh < tipKbdH+tipPadY*2 {
		bh = tipKbdH + tipPadY*2
	}
	return bw, bh, tipRight, kbdW
}

// showNow 立即显示(或平移)气泡到目标元素旁。
func (t *Tooltip) showNow(hEle int) {
	text := t.tips[hEle]
	if text == "" {
		return
	}
	hWnd := windowOf(hEle)
	// 宿主正在销毁(或已失效)时不要再显示: 宿主销毁期间仍可能收到迟到的
	// MouseMove/MouseStay, 再走一遍显示流程会碰到正在销毁的窗口/元素句柄。
	if hWnd == 0 || t.dying[hWnd] || !xc.XC_IsHXCGUI(hWnd, xcc.XC_WINDOW) {
		return
	}
	t.watchHost(hWnd)
	if t.hostOf[hEle] != hWnd { // 绑定时窗口还没就绪(元素刚创建)就在这里补挂
		t.hostOf[hEle] = hWnd
		t.watchWindow(hWnd)
	}
	t.ensure()
	cfg, ok := t.cfg[hEle]
	if !ok {
		cfg.delay = -1
	}
	t.text = text
	t.shownEle = hEle

	bw, bh, _, _ := t.measure(text, cfg)

	// 元素在窗口客户区中的矩形 -> 屏幕逻辑坐标。
	// 注意: XEle_GetWndClientRect 的原点就是"窗口 rect 原点"(含阴影边距), 不再叠加
	// XWnd_GetClientRect 的客户区内缩 —— 叠加过一次, 气泡会整体下移阴影宽度(实测 8px,
	// 箭头尖端扎进目标元素里)。
	var rc, wr xc.RECT
	xc.XEle_GetWndClientRect(hEle, &rc)
	xc.XWnd_GetRect(hWnd, &wr)
	sx, sy := wr.Left+rc.Left, wr.Top+rc.Top
	ex, ey := rc.Right-rc.Left, rc.Bottom-rc.Top

	// 工作区(逻辑像素): 翻转与夹紧都按**窗口所在显示器的工作区**算, 不能拿 0 当屏幕顶 ——
	// 副屏原点非 0(显示器摆在主屏正上方时 y 是负的)时 "y < 0" 恒成立, 上方明明有空间
	// 也会把气泡翻到下方。工作区还顺带扣掉了任务栏。
	dpi := xc.XWnd_GetDPI(hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	scale := float64(dpi) / 96

	// 窗口尺寸 = 气泡 + 贴元素那一侧的箭头(箭头恰好填满这段间距, 所以窗口边缘要贴住
	// 元素边缘 —— 不再额外加间距, 否则箭头尖端会悬在半空, 看起来"提示框飘得太高")。
	ww, wh := bw, bh
	side := cfg.side
	if side == TipAuto {
		side = TipTop
	}
	if side == TipTop || side == TipBottom {
		wh = bh + tipArrow
	} else {
		ww = bw + tipArrow
	}

	// 计算某方向下的落位: 气泡贴元素那一侧的边缘 = 元素边缘 + offset。
	// 落位公式只在 placeX/placeY 里写一份 —— 翻转预判与最终落位必须完全一致。
	place := func(s TipSide) (int32, int32) {
		return placeX(s, sx, ex, sy, ey, bw, bh, ww, wh, cfg),
			placeY(s, sx, sy, ex, ey, bw, bh, ww, wh, cfg)
	}
	flip := func(s TipSide) TipSide {
		switch s {
		case TipTop:
			return TipBottom
		case TipBottom:
			return TipTop
		case TipLeft:
			return TipRight
		case TipRight:
			return TipLeft
		}
		return s
	}

	x, y := place(side)
	if wa, ok := xui.MonitorWorkArea(xc.XWnd_GetHWND(hWnd)); ok {
		waL := int32(math.Round(float64(wa.Left) / scale))
		waT := int32(math.Round(float64(wa.Top) / scale))
		waR := int32(math.Round(float64(wa.Right) / scale))
		waB := int32(math.Round(float64(wa.Bottom) / scale))
		// 该方向放不下时: 先试着翻到对侧(radix avoidCollisions 语义), 还对侧也放不下就夹进工作区。
		if !tipFits(side, x, y, ww, wh, waL, waT, waR, waB) {
			if fs := flip(side); tipFits(fs, placeX(fs, sx, ex, sy, ey, bw, bh, ww, wh, cfg), placeY(fs, sx, sy, ex, ey, bw, bh, ww, wh, cfg), ww, wh, waL, waT, waR, waB) {
				side = fs
				x, y = place(side)
			}
		}
		if x+ww > waR {
			x = waR - ww
		}
		if x < waL {
			x = waL
		}
		if y+wh > waB {
			y = waB - wh
		}
		if y < waT {
			y = waT
		}
	} else { // 取不到显示器信息: 退回以虚拟屏原点为界的旧行为
		if side == TipTop && y < 0 && sy+ey+wh <= int32(float64(wapi.GetSystemMetrics(wapi.SM_CYVIRTUALSCREEN))) {
			side = TipBottom
			x, y = place(side)
		}
		if sw := screenWidth(hWnd); x+ww > sw {
			x = sw - ww
		}
		if x < 0 {
			x = 0
		}
	}
	t.side = side

	// 箭头中心: 一般指向触发元素中心(在贴元素那条边的方向上), 再夹进气泡里。
	//
	// 例外 —— **气泡在这一维比元素还短时, 取气泡中心**: 单行气泡高 28、按钮高 33 时,
	// "元素中心"必然落在气泡中心上方约 2.5px(气泡底边还要跟元素底边对齐, 见 align),
	// 三角形看着就是歪的(用户反馈"三角偏上")。而对齐语义本身已经保证气泡就在元素旁边,
	// 此时居中在视觉上才是"尖角对着元素"。元素比气泡长(长文案气泡)时仍按元素中心投影,
	// 否则 align=start/end 下箭头会跑到元素外面去。
	if side == TipTop || side == TipBottom {
		if bw <= ex {
			t.arrow = bw / 2
		} else {
			t.arrow = clampArrow(sx+ex/2-x, bw)
		}
	} else {
		if bh <= ey {
			t.arrow = bh / 2
		} else {
			t.arrow = clampArrow(sy+ey/2-y, bh)
		}
	}

	xc.XEle_SetRect(t.ele.Handle, &xc.RECT{Left: 0, Top: 0, Right: ww, Bottom: wh}, false, xcc.AdjustLayout_No)
	// 注意: XWnd_SetWindowPos 直接透传给 Win32, 用的是物理像素(框架内部坐标是逻辑像素,
	// 窗口创建时由框架按 DPI 缩放), 所以这里要自己乘 DPI 比例, 否则窗口会跑到 1/scale 处。
	px := int32(float64(x) * scale)
	py := int32(float64(y) * scale)
	fresh := !t.shown
	t.fadeGen++ // 作废上一次的进场/退场帧(见 fadeGen 的说明)
	gen := t.fadeGen
	if fresh {
		// 进场前把气泡元素压成全透明(淡的是元素 alpha, 见 fade 的说明): 首帧就不会
		// 带着上一次的 255 闪一下。元素 alpha 允许为 0 —— 只有**窗口** alpha 不能为 0
		// (0 会让 xcgui 停止合成该窗口, 见 SetWindowAlpha)。
		xui.SetElementsAlpha(0, t.ele.Handle)
	} else {
		// 切换目标(不重播进场): 上一次进场可能被打断在半透明, 这里直接钉到不透明。
		xui.SetElementsAlpha(1, t.ele.Handle)
	}
	t.win.SetWindowPos(xcc.HWND_TOPMOST, px, py,
		int32(float64(ww)*scale), int32(float64(wh)*scale),
		xcc.SWP_NOACTIVATE|xcc.SWP_SHOWWINDOW)
	wapi.ShowWindow(t.win.GetHWND(), 4) // SW_SHOWNOACTIVATE
	t.ele.Redraw(false)
	t.win.Redraw(true)

	if !fresh {
		// 已经显示着(只是在目标之间横移): 只挪位置, 不重播进场动画
		return
	}
	// 整窗 alpha 固定为"不透明": 它就是异型窗**第一次**落地的那个透明度, 之后再改也
	// 不生效(见 fade 的说明), 所以必须在窗口已经显示出来之后把它钉在 255。
	xui.SetWindowAlpha(t.win.Handle, 1)
	t.shown = true
	t.hWnd = hWnd
	// 进场(shadcn tooltip: fade-in-0 + zoom-in-95 + slide-in-from-<side>-2, 150ms):
	// 气泡从更靠近元素的位置(8px)滑到最终位置, 方向跟着 side 走。
	dx, dy := int32(0), int32(0)
	switch side {
	case TipTop:
		dy = int32(8 * scale)
	case TipBottom:
		dy = -int32(8 * scale)
	case TipLeft:
		dx = int32(8 * scale)
	case TipRight:
		dx = -int32(8 * scale)
	}
	xui.Presence(t.hWnd, t.fadeKey, true, xui.TooltipDuration, func(v float64) {
		if t.fadeGen != gen {
			return
		}
		ox := px + int32(float64(dx)*(1-v)+0.5)
		oy := py + int32(float64(dy)*(1-v)+0.5)
		t.win.SetWindowPos(xcc.HWND_TOPMOST, ox, oy, 0, 0,
			xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE|xcc.SWP_SHOWWINDOW)
		t.fade(v)
	}, nil)
}

// placeX / placeY 是 place 的纯函数版本, 供"翻转后是否放得下"的预判使用。
func placeX(s TipSide, sx, ex, sy, ey, bw, bh, ww, wh int32, cfg tipConfig) int32 {
	switch s {
	case TipTop, TipBottom:
		x := sx + ex/2 - bw/2
		switch cfg.align {
		case TipAlignStart:
			x = sx
		case TipAlignEnd:
			x = sx + ex - bw
		}
		return x
	}
	if s == TipLeft {
		return sx - ww - cfg.offset
	}
	return sx + ex + cfg.offset
}

func placeY(s TipSide, sx, sy, ex, ey, bw, bh, ww, wh int32, cfg tipConfig) int32 {
	switch s {
	case TipTop:
		return sy - wh - cfg.offset
	case TipBottom:
		return sy + ey + cfg.offset
	}
	y := sy + ey/2 - bh/2
	switch cfg.align {
	case TipAlignStart:
		y = sy
	case TipAlignEnd:
		y = sy + ey - bh
	}
	return y
}

// tipFits 报告气泡矩形是否完整落在工作区内。
func tipFits(_ TipSide, x, y, ww, wh, waL, waT, waR, waB int32) bool {
	return x >= waL && y >= waT && x+ww <= waR && y+wh <= waB
}

// WatchHost 在气泡首次用于某个宿主窗口时, 挂上"宿主销毁 → 清理气泡窗"的回调。
//
// 气泡是独立小窗, 平时只是 ShowWindow(SW_HIDE); 而消息循环的退出
// 条件是**所有用户创建的窗口都销毁**(不是都不可见) —— 不清理的话主窗口关掉后
// 进程还挂在 Run() 里(见 DESIGN 第八节 77)。
func (t *Tooltip) watchHost(hWnd int) {
	if hWnd == 0 || t.hosts[hWnd] {
		return
	}
	w := window.NewByHandle(hWnd)
	if w == nil {
		return
	}
	t.hosts[hWnd] = true
	w.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
		if message == wapi.WM_DESTROY {
			t.onHostDestroy(hWindow)
		}
		return 0
	})
}

// onHostDestroy 丢弃该宿主上的过渡并销毁气泡窗(气泡最后显示在其他宿主上时保留,
// 等那个宿主销毁再清)。
func (t *Tooltip) onHostDestroy(hWnd int) {
	delete(t.hosts, hWnd)
	delete(t.winWatch, hWnd)
	t.dying[hWnd] = true
	defer delete(t.dying, hWnd)
	xui.DropTransitions(hWnd)
	if t.hWnd != hWnd {
		return
	}
	t.destroyWin()
}

// destroyWin 销毁气泡窗并清空显示状态, 下次 show 由 ensure 重建。
// 状态要在 DestroyWindow 之前清: 销毁过程可能同步派发迟到的鼠标事件,
// 那时再用旧的 win/ele 句柄会触发 xcgui 的"句柄无效"错误弹窗。
func (t *Tooltip) destroyWin() {
	w := t.win
	t.win, t.ele = nil, nil
	t.shown = false
	t.shownEle = 0
	t.hWnd = 0
	t.text = ""
	t.cancelPending()
	if w != nil {
		w.DestroyWindow()
	}
}

func (t *Tooltip) ensure() {
	if t.win != nil {
		return
	}
	w := window.NewEx(
		xcc.WS_EX_TOOLWINDOW|xcc.WS_EX_NOACTIVATE|xcc.WS_EX_TRANSPARENT,
		0, "", 0, 0, 10, 10, "", 0, xcc.Window_Style_Nothing)
	w.SetBorderSize(0, 0, 0, 0)
	// 真透明: 异型窗口带透明通道, 窗口形状 = 自身绘制内容中不透明像素的并集。
	// 本组件的气泡与箭头全部由 paint 回调自绘(元素已 EnableBkTransparent), 因此
	// 气泡圆角之外、箭头两侧天然是全透明的, 压在别的元素上也不会露出方形底。
	w.SetTransparentType(xcc.Window_Transparent_Shaped)
	w.SetTransparentAlpha(255)
	w.SetWindowPos(xcc.HWND_TOPMOST, 0, 0, 0, 0,
		xcc.SWP_NOMOVE|xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE)

	ele := widget.NewElement(0, 0, 10, 10, w.Handle)
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(t.onPaint)
	// 气泡是独立置顶小窗, 不在主题管理器的"已注册窗口"清单里, 宿主窗口重绘也不会带动它
	// 的内容 —— 不订阅的话, 切主题时**正挂在屏幕上的**气泡会一直保持旧配色, 直到鼠标移开
	// 重新进场(实测)。按元素订阅, 元素随窗口销毁时自动反注册。
	xui.OnThemeChangeFor(ele.Handle, func(*xui.Theme) {
		if !t.shown {
			return
		}
		// 走一遍 show 而不是单纯重绘: 主题可能换字体/字号, 气泡尺寸得按新度量重算;
		// 目标元素已被 Unbind 时退化成纯重绘(配色照样要跟上)。
		if t.tips[t.shownEle] == "" {
			t.win.Redraw(true)
			return
		}
		t.showNow(t.shownEle)
	})
	t.win, t.ele = w, ele
}

func (t *Tooltip) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()

	// 气泡矩形: 窗口 = 气泡 + 箭头, 箭头在贴元素的那一侧, 只收缩贴箭头的那条边
	// (此前 Bottom 无条件收缩, 翻到下方时气泡矮了 tipArrow, 文字被垂直裁掉)。
	bubble := p.Rc
	switch t.side {
	case TipTop:
		bubble.Bottom = p.Rc.Top + p.H() - tipArrow
	case TipBottom:
		bubble.Top = p.Rc.Top + tipArrow
	case TipLeft:
		bubble.Right = p.Rc.Left + p.W() - tipArrow
	case TipRight:
		bubble.Left = p.Rc.Left + tipArrow
	}

	// 箭头: 只画露在气泡外的三角形(底宽 2*tipArrow), 底边与气泡边线严格重合。
	// 不要改回"画完整菱形 + 气泡盖住内侧一半": XEle_SetAlpha 是**逐图元**混合而不是
	// 整层合成 —— 淡入/淡出到半透明时, 被气泡盖住的那半个菱形会二次叠色, 看起来像
	// "倒三角上方浮出一个正三角"(见 DESIGN 第八节 89)。箭头与气泡的填充不能重叠。
	// 尖端留 1px 在窗口内(窗口边缘 = 目标元素边缘), 避免被元素矩形裁掉。
	half := tipArrow
	ax, ay := t.arrow, t.arrow
	if ax == 0 { // 老路径/极端情况兜底: 箭头居中
		ax = p.W() / 2
	}
	if ay == 0 {
		ay = p.H() / 2
	}
	p.Draw.SetBrushColor(uint32(th.Foreground))
	switch t.side {
	case TipTop:
		base := bubble.Bottom
		p.Draw.FillPolygon([]xc.POINT{
			{X: ax - half, Y: base},
			{X: ax + half, Y: base},
			{X: ax, Y: p.Rc.Bottom - 1},
		})
	case TipBottom:
		base := bubble.Top
		p.Draw.FillPolygon([]xc.POINT{
			{X: ax - half, Y: base},
			{X: ax + half, Y: base},
			{X: ax, Y: p.Rc.Top + 1},
		})
	case TipLeft:
		base := bubble.Right
		p.Draw.FillPolygon([]xc.POINT{
			{X: base, Y: ay - half},
			{X: base, Y: ay + half},
			{X: p.Rc.Right - 1, Y: ay},
		})
	default: // TipRight
		base := bubble.Left
		p.Draw.FillPolygon([]xc.POINT{
			{X: base, Y: ay - half},
			{X: base, Y: ay + half},
			{X: p.Rc.Left + 1, Y: ay},
		})
	}

	// 气泡: bg-foreground, rounded-md
	p.FillRound(&bubble, th.Foreground, tipRadius)

	cfg := t.cfg[t.shownEle]
	_, _, tipRight, kbdW := t.measure(t.text, cfg)
	textRc := xc.RECT{
		Left:   bubble.Left + tipPadX,
		Top:    bubble.Top + tipPadY,
		Right:  bubble.Right - tipRight,
		Bottom: bubble.Bottom - tipPadY,
	}
	f := xui.FontHandle(xui.FontSpec{
		Family: th.FontDefault.Family,
		Size:   th.FontDefault.Size - 2, // text-xs
	})
	if f != 0 {
		p.Draw.SetFont(f)
	}
	if kbdW > 0 {
		textRc.Right -= tipKbdGap + kbdW
	}
	p.Text(t.text, &textRc, th.Background, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)

	// 快捷键块: 气泡底色是 foreground, 用"反色 20%"的小圆角块 + 反色文字(同 Kbd 在
	// 深色气泡上的观感), 垂直与文字行居中。
	if kbdW > 0 {
		kh := tipKbdH
		ky := (bubble.Top + bubble.Bottom) / 2
		krc := xc.RECT{
			Left:   bubble.Right - tipRight - kbdW,
			Top:    ky - kh/2,
			Right:  bubble.Right - tipRight,
			Bottom: ky + kh/2,
		}
		p.FillRound(&krc, th.Background.WithAlpha(38), th.RadiusSm())
		kf := f
		if kf != 0 {
			p.Draw.SetFont(kf)
		}
		p.Text(cfg.shortcut, &krc, th.Background, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	}
	return 0
}

// clampArrow 只保证三角形完整落在气泡里(不外溢), 不再把箭头让开圆角。
//
// 早先下界写成 `radius+half`(6+7=13), 单行气泡(高 28)的合法区间上界才 15 —— 元素中心在
// 11.5 时会被硬抬到 13, 加上气泡(28)比按钮(33)矮、align=end 又把底边对齐, 箭头看起来
// 就比气泡中心高 3px 左右("三角偏上")。而圆角处让三角压过去最多只差 0.2px(半径 6 的
// 弧在 y=4.5 处内凹 6-sqrt(36-1.5²)), 肉眼不可见, 远不如"指不准"明显。
//
// 气泡比三角形还窄时(span < 2*half)无解, 取正中。
func clampArrow(v, span int32) int32 {
	lo, hi := tipArrow, span-tipArrow
	if hi < lo {
		return span / 2
	}
	if v < lo {
		v = lo
	}
	if v > hi {
		v = hi
	}
	return v
}

// widgetVisible 沿父元素链判断元素是否整体可见(隐藏页面里的元素 XWidget_IsShow 仍为 true,
// 必须看祖先)。
func widgetVisible(hEle int) bool {
	for h := hEle; h != 0; h = xc.XWidget_GetParentEle(h) {
		if !xc.XWidget_IsShow(h) {
			return false
		}
	}
	return true
}

// windowOf 返回元素所属的 **xcgui 窗口**句柄。
//
// 不能直接 `XC_hWindowFromHWnd(XWidget_GetHWND(ele))`: 元素在 ScrollArea(LayoutFrame)、
// 弹层容器这类**自带 HWND 的容器**内部时, 取到的 HWND 是那个子窗口, 它不是 xcgui 窗口,
// 映射出来是个无效小值(实测 7) —— 用它去 `XWnd_GetRect`/presence 会静默失败, 表现为
// "滚动区里的按钮悬停没反应、气泡永远不出现"。所以沿 HWND 父链一直往上找, 直到找到
// 真正的 xcgui 窗口。
func windowOf(hEle int) int {
	return xui.HostWindowOf(hEle)
}

// screenWidth 返回窗口所在显示器的逻辑宽度(物理宽度按 DPI 折算), 用于把提示夹在屏幕内。
func screenWidth(hWindow int) int32 {
	dpi := xc.XWnd_GetDPI(hWindow)
	if dpi <= 0 {
		dpi = 96
	}
	return int32(float64(wapi.GetSystemMetrics(wapi.SM_CXVIRTUALSCREEN)) * 96 / float64(dpi))
}
