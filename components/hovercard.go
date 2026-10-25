package ui

import (
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// HoverCard 几何与延时(逻辑像素 / 时间), 对齐 shadcn hover-card.tsx:
// w-64(256) / p-4(16) / sideOffset=4; radix 默认 openDelay=700ms、closeDelay=300ms。
const (
	hoverCardDefaultW int32 = 256
	hoverCardPad      int32 = 16
	hoverCardGap      int32 = 4
)

const (
	hoverCardOpenDelay  = 700 * time.Millisecond
	hoverCardCloseDelay = 300 * time.Millisecond
)

// HoverCardOption 悬停卡片配置项。
type HoverCardOption func(*HoverCard)

// WithHoverCardContent 设置内容构建回调(content 是纵向流式容器, 宽铺满、高自撑)。
func WithHoverCardContent(build func(content int)) HoverCardOption {
	return func(h *HoverCard) { h.build = build }
}

// WithHoverCardWidth 覆盖卡片宽度(默认 256 = w-64)。
func WithHoverCardWidth(w int32) HoverCardOption {
	return func(h *HoverCard) {
		if w > 0 {
			h.width = w
		}
	}
}

// WithHoverCardPadding 覆盖四周内边距(默认 16 = p-4)。
func WithHoverCardPadding(pad int32) HoverCardOption {
	return func(h *HoverCard) {
		if pad >= 0 {
			h.padding = pad
		}
	}
}

// WithHoverCardSide 设置优先方位(默认下方, 放不下自动翻转)。
func WithHoverCardSide(s PopoverSide) HoverCardOption {
	return func(h *HoverCard) { h.side = s }
}

// WithHoverCardAlign 设置交叉轴对齐(默认居中)。
func WithHoverCardAlign(a PopoverAlign) HoverCardOption {
	return func(h *HoverCard) { h.align = a }
}

// WithHoverCardDelay 覆盖悬停打开/离开关闭的延时。
func WithHoverCardDelay(open, close time.Duration) HoverCardOption {
	return func(h *HoverCard) {
		if open >= 0 {
			h.openDelay = open
		}
		if close >= 0 {
			h.closeDelay = close
		}
	}
}

// WithHoverCardOnOpenChange 注册开关状态回调。
func WithHoverCardOnOpenChange(fn func(open bool)) HoverCardOption {
	return func(h *HoverCard) { h.changes = append(h.changes, fn) }
}

// HoverCard 悬停卡片, 对应 shadcn hover-card.tsx: 鼠标在触发区停留 700ms 后弹出,
// 移出触发区/卡片 300ms 后关闭; 卡片本身可以交互, 打开期间不抢焦点。
//
// 与 shadcn 的差异: 进场只淡入, 不做 zoom/slide; 不做键盘焦点打开(鼠标悬停语义)。
type HoverCard struct {
	trigger int
	build   func(content int)

	width   int32
	padding int32
	gap     int32
	side    PopoverSide
	align   PopoverAlign

	openDelay  time.Duration
	closeDelay time.Duration

	panel    *popoverPanel
	content  *widget.LayoutEle
	built    bool
	measured bool

	timer *time.Timer
	gen   int

	hostWatched bool
	dying       bool
	changes     []func(open bool)
}

// NewHoverCard 把悬停卡片绑定到任意元素。
func NewHoverCard(hTrigger int, opts ...HoverCardOption) *HoverCard {
	if hTrigger == 0 {
		return nil
	}
	h := &HoverCard{
		trigger:    hTrigger,
		width:      hoverCardDefaultW,
		padding:    hoverCardPad,
		gap:        hoverCardGap,
		openDelay:  hoverCardOpenDelay,
		closeDelay: hoverCardCloseDelay,
	}
	for _, o := range opts {
		o(h)
	}

	ele := widget.NewElementByHandle(hTrigger)
	if ele == nil {
		return nil
	}
	h.watchHost()
	ele.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		h.scheduleOpen()
		return 0
	})
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		h.scheduleOpen()
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		// 还没打开时是"悬停途中离开": 取消待打开计时, 否则鼠标早就走了卡片还会弹出来
		if h.IsOpen() {
			h.scheduleClose()
		} else {
			h.cancelTimer()
		}
		return 0
	})
	ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		h.cancelTimer()
		if h.panel != nil {
			h.panel.Destroy()
		}
		h.panel, h.content = nil, nil
		return 0
	})
	return h
}

// Content 返回内容容器句柄(首次调用时构建)。
func (h *HoverCard) Content() int {
	if h == nil || !h.ensureBuilt() {
		return 0
	}
	return h.content.Handle
}

// Open 立即打开(不受悬停延时限制, 例如键盘聚焦时)。
func (h *HoverCard) Open() *HoverCard {
	if h == nil || !h.ensureBuilt() {
		return h
	}
	if h.panel.IsOpen() {
		return h
	}
	if !h.measured {
		h.measure()
	}
	x, y := popupPosition(h.trigger, h.width, h.panel.ch, h.side, h.align, h.gap)
	h.panel.ShowAt(h.trigger, x, y)
	h.notify(true)
	return h
}

// Close 立即关闭。
func (h *HoverCard) Close() *HoverCard {
	if h != nil {
		h.cancelTimer()
		if h.panel != nil {
			h.panel.Hide()
		}
	}
	return h
}

// SetOpen 按参数打开/关闭。
func (h *HoverCard) SetOpen(open bool) *HoverCard {
	if open {
		return h.Open()
	}
	return h.Close()
}

// IsOpen 报告卡片是否打开。
func (h *HoverCard) IsOpen() bool { return h != nil && h.panel != nil && h.panel.IsOpen() }

// Refresh 重新量一次内容高度(内容子项变化后调用)。
func (h *HoverCard) Refresh() *HoverCard {
	if h == nil || !h.built || h.panel == nil {
		return h
	}
	if h.panel.IsOpen() {
		// 打开中只重排内容: 离屏量高会先把窗口挪到 -30000 再隐藏
		h.content.AdjustLayout()
		if h.panel.alive() {
			h.panel.win.Redraw(true)
		}
		return h
	}
	h.measure()
	return h
}

func (h *HoverCard) measure() {
	if hh := h.panel.MeasureContent(h.padding); hh <= 0 {
		h.panel.resize(h.width, h.padding*2)
	}
	h.measured = true
}

func (h *HoverCard) ensureBuilt() bool {
	if h.built {
		return h.panel != nil
	}
	h.built = true
	panel := newPopoverPanelEx(h.width, h.padding*2+1, h.trigger, true, false)
	if panel == nil {
		return false
	}
	content := panel.LayoutContent(h.padding, 8)
	if content == nil {
		panel.Destroy()
		return false
	}
	h.panel, h.content = panel, content
	panel.OnHide = func() { h.notify(false) }
	if h.build != nil {
		h.build(content.Handle)
	}
	// 卡片里任何一个子元素被划出都会触发一次关闭检查; 指针仍在卡片/触发区上时
	// 检查会续期, 不会误关。
	h.bindLeave(content.Handle)
	h.bindLeave(panel.body.Handle)
	return true
}

// bindLeave 给内容区整棵子树挂 MouseLeave(保留已有事件, allowAddingMultiple 默认 true)。
func (h *HoverCard) bindLeave(hEle int) {
	if hEle == 0 || !xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
		return
	}
	if ele := widget.NewElementByHandle(hEle); ele != nil {
		ele.AddEvent_MouseLeave(func(int, int, *bool) int {
			h.scheduleClose()
			return 0
		})
	}
	n := xc.XEle_GetChildCount(hEle)
	for i := int32(0); i < n; i++ {
		h.bindLeave(xc.XEle_GetChildByIndex(hEle, i))
	}
}

func (h *HoverCard) notify(open bool) {
	for _, fn := range h.changes {
		fn(open)
	}
}

func (h *HoverCard) cancelTimer() {
	h.gen++
	if h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
}

// scheduleOpen 在 openDelay 后打开; 重复的 MouseMove 不会重置已到点的计时
// (悬停一直移动也算停留), 只在还没有计时/已打开时直接返回。
func (h *HoverCard) scheduleOpen() {
	if h.IsOpen() || h.timer != nil || h.dying {
		return
	}
	gen := h.gen
	h.timer = app.AfterFunc(h.openDelay, func() {
		if gen != h.gen {
			return
		}
		h.timer = nil
		if h.dying || !h.pointerOver() {
			return
		}
		h.Open()
	})
}

func (h *HoverCard) scheduleClose() {
	if !h.IsOpen() || h.dying {
		return
	}
	h.cancelTimer()
	gen := h.gen
	h.timer = app.AfterFunc(h.closeDelay, func() {
		if gen != h.gen {
			return
		}
		h.timer = nil
		if h.dying {
			return
		}
		h.closeIfOutside()
	})
}

// closeIfOutside 检查指针是否还在触发区/卡片上; 仍在则过一小段时间再看,
// 直到真正移出为止(鼠标从子元素移到卡片留白时不会再有 MouseLeave 事件)。
func (h *HoverCard) closeIfOutside() {
	if !h.IsOpen() {
		return
	}
	if h.pointerOver() {
		gen := h.gen
		h.timer = app.AfterFunc(120*time.Millisecond, func() {
			if gen != h.gen {
				return
			}
			h.timer = nil
			h.closeIfOutside()
		})
		return
	}
	h.panel.Hide()
}

func (h *HoverCard) pointerOver() bool {
	var pt wapi.POINT
	if !wapi.GetCursorPos(&pt) {
		return false
	}
	if pointInElement(h.trigger, pt) {
		return true
	}
	if h.panel != nil && h.panel.alive() {
		var wr wapi.RECT
		if wapi.GetWindowRect(h.panel.win.GetHWND(), &wr) {
			return int64(pt.X) >= int64(wr.Left) && int64(pt.X) < int64(wr.Right) &&
				int64(pt.Y) >= int64(wr.Top) && int64(pt.Y) < int64(wr.Bottom)
		}
	}
	return false
}

// pointInElement 报告物理屏幕坐标是否落在元素矩形内。
func pointInElement(hEle int, pt wapi.POINT) bool {
	hWnd := xc.XWidget_GetHWINDOW(hEle)
	if hWnd <= 0 {
		return false
	}
	var rc, wr xc.RECT
	xc.XEle_GetWndClientRect(hEle, &rc)
	xc.XWnd_GetRect(hWnd, &wr)
	dpi := xc.XWnd_GetDPI(hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96
	l := int32(float64(wr.Left+rc.Left)*sc + 0.5)
	t := int32(float64(wr.Top+rc.Top)*sc + 0.5)
	r := int32(float64(wr.Left+rc.Right)*sc + 0.5)
	b := int32(float64(wr.Top+rc.Bottom)*sc + 0.5)
	return int64(pt.X) >= int64(l) && int64(pt.X) < int64(r) &&
		int64(pt.Y) >= int64(t) && int64(pt.Y) < int64(b)
}

// watchHost 宿主窗口失焦/销毁时收起卡片并丢弃计时器(卡片是非激活窗, 不会自己失焦)。
func (h *HoverCard) watchHost() {
	hWnd := xc.XWidget_GetHWINDOW(h.trigger)
	if hWnd <= 0 || h.hostWatched {
		return
	}
	h.hostWatched = true
	if w := window.NewByHandle(hWnd); w != nil {
		w.AddEvent_KillFocus(func(hWindow int, pbHandled *bool) int {
			h.cancelTimer()
			if h.panel != nil {
				h.panel.Hide()
			}
			return 0
		})
		w.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
			if message == wapi.WM_DESTROY {
				h.dying = true
				h.cancelTimer()
				if h.panel != nil {
					h.panel.Destroy()
					h.panel, h.content = nil, nil
				}
			}
			return 0
		})
	}
}
