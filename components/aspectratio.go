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
// aspect-ratio, 组件在尺寸事件里按新宽度算高写回元素, 再逐级通知祖先重排
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
	pending  bool
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

// onSize 尺寸变化时预约一次同步。尺寸事件里直接改高度会重入布局,
// 且高度写回后还会再触发一次尺寸事件 —— 用 lastW/lastH 闸门避免自激。
func (a *AspectRatio) onSize(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
	if a == nil {
		return 0
	}
	w, h := a.GetWidth(), a.GetHeight()
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
	if a == nil || a.pending {
		return
	}
	a.pending = true
	app.AfterFunc(0, func() {
		a.pending = false
		a.syncSize()
	})
}

func (a *AspectRatio) syncSize() {
	if a == nil || a.LayoutEle == nil || !xc.XC_IsHXCGUI(a.Handle, xcc.XC_ELE) {
		return
	}
	w := a.GetWidth()
	if w <= 0 {
		return
	}
	a.lastW = w
	h := a.GetHeight()
	heightChanged := false
	if a.autoH {
		h = aspectRatioHeight(w, a.ratio)
		if h != a.GetHeight() {
			a.SetHeight(h)
			heightChanged = true
		}
	}
	a.lastH = h
	if h <= 0 {
		return
	}
	// 内容区不能用 Fill 参加布局: 父级是 Weight 宽时, xcgui 的量算趟会让 Fill 子项
	// 按祖先宽度算(实测比比例框宽 5~15px, 9:16 那块首帧高度还是 0)。这里把内容区
	// 锁成比例框的实际尺寸, 它内部的 Fill 子项才有正确的参照。
	if w != a.contentW || h != a.contentH {
		a.contentW, a.contentH = w, h
		a.content.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
		a.content.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
		a.content.SetWidth(w)
		a.content.SetHeight(h)
		a.content.AdjustLayout()
		a.content.Redraw(false)
	}
	if heightChanged {
		a.adjustAncestors()
		a.Redraw(false)
	}
}

// adjustAncestors 从自己开始逐级对祖先调用 AdjustLayout。只通知直接父级时,
// Auto 高的卡片/行仍按旧高度排版, ScrollArea 也不会重算滚动范围(同第八节 90/93)。
func (a *AspectRatio) adjustAncestors() {
	if a == nil || a.LayoutEle == nil {
		return
	}
	xc.XEle_AdjustLayout(a.Handle)
	for p := xc.XWidget_GetParentEle(a.Handle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			// ScrollArea 的框架自己会重排内容区并重算滚动范围。
			xc.XEle_AdjustLayout(p)
			break
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			// p 是框架的内容区: 直接对内容区 AdjustLayout 会把子项宽度打成 0(实测),
			// 必须交给框架排。
			xc.XEle_AdjustLayout(pp)
			break
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_ELE) {
			xc.XEle_AdjustLayout(p)
			continue
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
		}
		break
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
