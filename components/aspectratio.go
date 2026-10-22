package ui

import (
	"math"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// AspectRatio 固定宽高比的容器(对应 shadcn / radix 的 AspectRatio)。
//
// 宽度由父布局或布局项决定, **高度 = 宽度 / ratio**: xcgui 的布局没有 CSS 的
// aspect-ratio, 组件在尺寸事件里排队, 下一轮按新宽度算高写回元素并统一重排
// (Auto 高的卡片/行/滚动范围才会跟着长, 见 DESIGN 第八节 94)。
//
// 内容放进 Content(): 宽高铺满比例框(图片占位、图标网格等)。
//
// 构造时 cy > 0 表示调用方自己管高度(同 CSS 里宽高都确定时比例不生效);
// cy 传 0 才按宽度联动。高度被布局项声明成 Fill/Weight 时比例同样不生效。
type AspectRatio struct {
	*widget.LayoutEle
	ratio    float64
	autoH    bool
	queued   bool
	lastW    int32
	lastH    int32
	contentW int32
	contentH int32
	content  *widget.LayoutEle
}

// NewAspectRatio 创建比例容器。ratio = 宽 / 高(16:9 传 16.0/9.0);
// cx 传 0 时宽度由布局项决定, cy 传 0 时高度随宽度推导。
func NewAspectRatio(x, y, cx, cy int32, ratio float64, hParent int) *AspectRatio {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	a := &AspectRatio{
		LayoutEle: ele,
		ratio:     normalizeAspectRatio(ratio),
		autoH:     cy <= 0,
		lastW:     -1,
		lastH:     -1,
		contentW:  -1,
		contentH:  -1,
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false)
	ele.EnableBkTransparent(true)

	// 内容区: 铺满比例框。AspectRatio 不在自己身上挂子项 —— 布局会竖向堆叠,
	// 内容放这里才有"整个比例框都是内容"的语义。
	c := widget.NewLayoutEle(0, 0, 0, 0, ele.Handle)
	c.EnableHorizon(false)
	c.EnableLayout(true)
	c.EnableAutoWrap(false)
	c.EnableBkTransparent(true)
	c.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	c.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
	a.content = c

	if a.autoH {
		if cx > 0 {
			ele.SetHeight(aspectRatioHeight(cx, a.ratio))
		}
	}
	// 高度不联动(cy > 0)时也要跟随元素尺寸同步内容区, 所以两条路径都挂尺寸事件。
	ele.AddEvent_Size(a.onSize)
	a.scheduleSync()
	xui.AttachWindow(ele.Handle)
	return a
}

// Content 返回铺满比例框的内容容器(竖向流式)。
func (a *AspectRatio) Content() *widget.LayoutEle {
	if a == nil {
		return nil
	}
	return a.content
}

// ContentStack 设置内容区子项间距并返回它(与 Card.ContentStack 同名同义)。
func (a *AspectRatio) ContentStack(space int32) *widget.LayoutEle {
	if a == nil || a.content == nil {
		return nil
	}
	a.content.SetSpace(space)
	return a.content
}

// Ratio 返回当前宽高比。
func (a *AspectRatio) Ratio() float64 {
	if a == nil {
		return 0
	}
	return a.ratio
}

// SetRatio 改宽高比并立即按当前宽度重排(非正数忽略)。
func (a *AspectRatio) SetRatio(ratio float64) *AspectRatio {
	if a == nil || ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return a
	}
	a.ratio = ratio
	if a.autoH {
		a.syncSize()
	}
	return a
}

// 同一帧内可能有多个比例框同时拿到宽度(例如首次显示一个页面)。全局排队后
// 先统一写高度, 再统一重排/重绘: 每块各自同步会触发多次布局, 屏幕上是
// "只有前两块 → 前两块变高 → 竖块出现"的逐帧展开(实测)。
var (
	arQueue  []*AspectRatio
	arQueued bool
)

// onSize 尺寸变化时排队一次同步。高度写回会再触发尺寸事件, 由 queued 去重;
// 真正写高度放在下一轮消息里, 避免在布局趟里改尺寸造成重入/震荡。
func (a *AspectRatio) onSize(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
	if a == nil {
		return 0
	}
	w, h := a.GetWidth(), a.GetHeight()
	// 布局过渡期会把比例框临时排成 0 宽(ScrollArea 框架重排/滚动条显隐那一帧)。
	// 这种事件不能回写, 否则 335 ↔ 0 会无限自激; 保留上一次有效宽度, 等真正的
	// 布局恢复宽度后不会再触发同步。
	if w <= 0 {
		return 0
	}
	if a.autoH {
		if w == a.lastW {
			return 0
		}
	} else if w == a.lastW && h == a.lastH {
		return 0
	}
	a.scheduleSync()
	return 0
}

func (a *AspectRatio) scheduleSync() {
	if a == nil || a.queued {
		return
	}
	a.queued = true
	arQueue = append(arQueue, a)
	if arQueued {
		return
	}
	arQueued = true
	app.AfterFunc(0, flushAspectRatios)
}

// flushAspectRatios 一轮内先写完全部高度, 再统一重排/重绘, 避免每块比例框
// 各自触发一次布局造成的逐帧展开。布局过程中新产生的尺寸变化由新的批次处理。
func flushAspectRatios() {
	batch := arQueue
	arQueue = nil
	arQueued = false
	if len(batch) == 0 {
		return
	}
	var active []*AspectRatio
	for _, a := range batch {
		a.queued = false
		if a.applySize() {
			active = append(active, a)
		}
	}
	// 尺寸没变就什么都不做: 布局过渡期会额外触发一轮尺寸事件, 每轮都重排/重绘
	// 会让 ScrollArea 的 0 宽过渡和恢复互相驱动, 变成无限循环。
	if len(active) == 0 {
		return
	}
	var windows []int
	for _, a := range active {
		a.content.AdjustLayout()
		a.content.Redraw(false)
		a.adjustAncestors()
		windows = append(windows, a.windowHandle())
	}
	for _, hw := range windows {
		if hw != 0 {
			xc.XWnd_Redraw(hw, false)
		}
	}
}

// applySize 按当前宽度写回高度与内容区尺寸; 只改尺寸, 不重排不重绘。
func (a *AspectRatio) applySize() bool {
	if a == nil || a.LayoutEle == nil || !xc.XC_IsHXCGUI(a.Handle, xcc.XC_ELE) {
		return false
	}
	w := a.GetWidth()
	if w <= 0 {
		return false
	}
	a.lastW = w
	h := a.GetHeight()
	changed := false
	if a.autoH {
		h = aspectRatioHeight(w, a.ratio)
		if h != a.GetHeight() {
			a.SetHeight(h)
			changed = true
		}
	}
	if h <= 0 {
		return changed
	}
	a.lastH = h
	// 内容区不能用 Fill 参加布局: 父级是 Weight 宽时, xcgui 的量算趟会让 Fill 子项
	// 按祖先宽度算(实测比比例框宽 5~15px, 9:16 那块首帧高度还是 0)。这里把内容区
	// 锁成比例框的实际尺寸, 它内部的 Fill 子项才有正确的参照。
	if w != a.contentW || h != a.contentH {
		a.contentW, a.contentH = w, h
		a.content.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
		a.content.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
		a.content.SetWidth(w)
		a.content.SetHeight(h)
		changed = true
	}
	return changed
}

// syncSize 用于 SetRatio 这类不在尺寸事件里的即时变更。
func (a *AspectRatio) syncSize() {
	if a == nil || a.LayoutEle == nil || !xc.XC_IsHXCGUI(a.Handle, xcc.XC_ELE) {
		return
	}
	if a.GetWidth() <= 0 {
		return
	}
	if !a.applySize() {
		return
	}
	a.content.AdjustLayout()
	a.content.Redraw(false)
	a.adjustAncestors()
	if hw := a.windowHandle(); hw != 0 {
		xc.XWnd_Redraw(hw, false)
	}
	a.Redraw(false)
}

// windowHandle 返回宿主窗口的 xcgui 句柄(0 = 尚未附加窗口)。
func (a *AspectRatio) windowHandle() int {
	if a == nil || a.LayoutEle == nil {
		return 0
	}
	hwnd := xc.XWidget_GetHWND(a.Handle)
	if hwnd == 0 {
		return 0
	}
	return xc.XC_hWindowFromHWnd(hwnd)
}

// adjustAncestors 只把最近的 ScrollArea 框架(或窗口)重排一次。
// 不要在父链上逐级 AdjustLayout / 重新声明 Auto: 中间层的 Weight 布局会在
// 量算趟里把比例框临时排成 0 宽, 再被下一次布局恢复, 尺寸事件因此无限自激
// (实测: 335 ↔ 0 反复)。框架级重排会统一更新内容区高度、兄弟位置与滚动范围。
func (a *AspectRatio) adjustAncestors() {
	if a == nil || a.LayoutEle == nil {
		return
	}
	for p := xc.XWidget_GetParentEle(a.Handle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(p)
			return
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			// p 是框架的内容区: 直接对内容区 AdjustLayout 会把子项宽度打成 0(实测),
			// 必须交给框架排。
			xc.XEle_AdjustLayout(pp)
			return
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
			return
		}
	}
}

// normalizeAspectRatio 把非法比例收成 1:1(构造参数不能直接让组件消失)。
func normalizeAspectRatio(ratio float64) float64 {
	if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return 1
	}
	return ratio
}

// aspectRatioHeight 按宽算高, 至少 1 像素(极宽比例下别返回 0)。
func aspectRatioHeight(width int32, ratio float64) int32 {
	if width <= 0 || ratio <= 0 {
		return 0
	}
	h := int32(math.Round(float64(width) / ratio))
	if h < 1 {
		h = 1
	}
	return h
}
