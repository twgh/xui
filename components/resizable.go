package ui

import (
	"math"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Resizable 的几何与交互常量(逻辑像素 / 百分比), 对齐 shadcn resizable.tsx(radix-nova)
// 与它底下的 react-resizable-panels v4:
//   - 分隔条可见线 1px(bg-border), 热区 4px(上游 after:w-1);
//   - 抓手 w-1 h-6 = 4×24, 纵向分组时旋转成 24×4, rounded-lg 在 4px 宽度上收敛为胶囊;
//   - 方向键每次 5 个百分点, Home/End 直接推到相邻面板的边界(总位移 ±100),
//     双击分隔条把有显式默认值的相邻面板复位。
const (
	resizableHandleSize int32   = 4
	resizableGripLong   int32   = 24
	resizableGripShort  int32   = 4
	resizableKeyStep    float64 = 5
	resizableKeyJump    float64 = 100
)

// ===== 选项 =====

// ResizableGroupOption 分组配置项。
type ResizableGroupOption func(*ResizableGroup)

// WithResizableVertical 纵向分组: 面板上下堆叠, 分隔条是 1px 横线, 方向键改为 ↑/↓。
func WithResizableVertical() ResizableGroupOption {
	return func(g *ResizableGroup) { g.vertical = true }
}

// WithResizableDisabled 初始禁用调整(拖拽/键盘/双击都不响应)。
func WithResizableDisabled() ResizableGroupOption {
	return func(g *ResizableGroup) { g.disabled = true }
}

// WithResizableOnResize 布局变化(拖动/键盘/双击/SetSizes)时回调百分比。
func WithResizableOnResize(fn func(sizes []float64)) ResizableGroupOption {
	return func(g *ResizableGroup) {
		if fn != nil {
			g.callbacks = append(g.callbacks, fn)
		}
	}
}

// ResizablePanelOption 面板配置项。
type ResizablePanelOption func(*ResizablePanel)

// WithResizableSize 初始尺寸(百分比 0~100), 同时作为双击分隔条时的复位目标。
// 未指定的面板平分剩余份额。
func WithResizableSize(percent float64) ResizablePanelOption {
	return func(p *ResizablePanel) {
		if percent > 0 && percent <= 100 {
			p.size = percent
			p.def = percent
		}
	}
}

// WithResizableMin 最小尺寸(百分比), 拖动/键盘/窗口缩放都不会把它压得更小。
func WithResizableMin(percent float64) ResizablePanelOption {
	return func(p *ResizablePanel) {
		if percent >= 0 && percent <= 100 {
			p.min = percent
		}
	}
}

// WithResizableMax 最大尺寸(百分比, 默认 100)。
func WithResizableMax(percent float64) ResizablePanelOption {
	return func(p *ResizablePanel) {
		if percent > 0 && percent <= 100 {
			p.max = percent
		}
	}
}

// ResizableHandleOption 分隔条配置项。
type ResizableHandleOption func(*ResizableHandle)

// WithResizableGrip 在分隔条中间画一根抓手(上游的 withHandle): 4×24 的圆角短条。
func WithResizableGrip() ResizableHandleOption {
	return func(h *ResizableHandle) { h.grip = true }
}

// ===== 组件 =====

// ResizableGroup 可调整分栏容器, 对应 shadcn resizable.tsx 的 ResizablePanelGroup。
//
// 尺寸语义: 面板按**百分比**分配主轴空间(手柄占固定 4px), 分组尺寸变化时保持相对比例
// (上游默认的 preserve-relative-size)。因此分组必须有一个确定的主轴尺寸:
// 构造时给 cx(横向)/ cy(纵向), 或把布局项声明成 Fixed / Fill(父级确定尺寸)。
//
// 用法:
//
//	g := ui.NewResizableGroup(0, 0, 0, 240, parent)
//	xui.ItemFill(g)
//	xui.ItemHeight(g, 240)
//	left := g.AddPanel(ui.WithResizableSize(30), ui.WithResizableMin(20))
//	ui.NewLabel(0, 0, 0, 0, "左", left.Handle)
//	g.AddHandle(ui.WithResizableGrip())
//	right := g.AddPanel(ui.WithResizableSize(70))
//	...
//
// 与上游的差异: 未实现 collapsible / collapsedSize(Enter 折叠)、preserve-pixel-size
// 与 F6 在分隔条之间跳焦点; 面板/分隔条也没有单独的 disabled 开关(只有分组级)。
type ResizableGroup struct {
	*widget.LayoutEle

	vertical bool
	disabled bool

	panels  []*ResizablePanel
	handles []*ResizableHandle

	sizes      []float64
	normalized bool

	hWindow   int
	keyID     int
	queued    bool
	callbacks []func([]float64)

	dragging  *ResizableHandle
	dragIndex int
	dragStart int32
	dragBase  []float64
}

// NewResizableGroup 创建分组。构造尺寸与普通容器一致, 布局项尺寸由调用方声明;
// 分组尺寸变化时面板按百分比重排, 所以主轴尺寸必须确定。
func NewResizableGroup(x, y, cx, cy int32, hParent int, opts ...ResizableGroupOption) *ResizableGroup {
	g := &ResizableGroup{}
	for _, o := range opts {
		o(g)
	}
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	ele.EnableHorizon(!g.vertical)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false)
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.SetSpace(0)
	g.LayoutEle = ele

	ele.AddEvent_Size(g.onSize)
	ele.AddEvent_MouseMove(g.onMouseMove)
	ele.AddEvent_LButtonUp(g.onMouseUp)

	xui.AttachWindow(ele.Handle)
	g.hWindow = xui.HostWindowOf(ele.Handle)
	g.bindKeys()
	return g
}

// AddPanel 追加一个面板。面板本身是竖向流容器(可改 EnableHorizon), 内容直接放进去;
// 顺序必须与实际布局一致: AddPanel -> AddHandle -> AddPanel ...
func (g *ResizableGroup) AddPanel(opts ...ResizablePanelOption) *ResizablePanel {
	if g == nil {
		return nil
	}
	p := &ResizablePanel{group: g, max: 100}
	for _, o := range opts {
		o(p)
	}
	var ele *widget.LayoutEle
	if g.vertical {
		ele = widget.NewLayoutEle(0, 0, 0, 1, g.Handle)
	} else {
		ele = widget.NewLayoutEle(0, 0, 1, 0, g.Handle)
	}
	if ele == nil {
		return nil
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false)
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	p.LayoutEle = ele
	if g.vertical {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 1)
	} else {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 1)
		ele.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
	}
	p.px = -1

	// 已经分配过份额后再加面板: 给它一份, 其余按比例收缩, 避免新面板拿到 0。
	if g.normalized && len(g.sizes) > 0 {
		n := len(g.panels) + 1
		share := p.size
		if share <= 0 {
			share = 100 / float64(n)
		}
		scale := (100 - share) / 100
		next := make([]float64, 0, n)
		for _, s := range g.sizes {
			next = append(next, s*scale)
		}
		next = append(next, share)
		g.sizes = normalizeResizableSizes(next, append(g.mins(), p.min), append(g.maxes(), p.max))
	}
	g.panels = append(g.panels, p)
	// 分组已经显示过时, 新面板要按当前百分比补上像素尺寸(否则停在创建时的 1px)。
	g.scheduleSync()
	return p
}

// AddHandle 追加一个分隔条, 它调整**前一个面板与后一个面板** —— 因此必须在两个面板
// 之间调用, 且后面要再 AddPanel 才有右邻。
func (g *ResizableGroup) AddHandle(opts ...ResizableHandleOption) *ResizableHandle {
	if g == nil || len(g.panels) == 0 {
		return nil
	}
	h := &ResizableHandle{group: g, index: len(g.panels) - 1, lastSig: -1}
	for _, o := range opts {
		o(h)
	}
	var ele *widget.Element
	if g.vertical {
		ele = widget.NewElement(0, 0, 0, resizableHandleSize, g.Handle)
	} else {
		ele = widget.NewElement(0, 0, resizableHandleSize, 0, g.Handle)
	}
	if ele == nil {
		return nil
	}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	// 一个 Tab 站点: 只有键盘能打开焦点环(鼠标点击不改外观, 同 focus-visible)。
	ele.EnableFocus(true)
	ele.EnableSwitchFocus(true)
	xc.XEle_EnableDrawFocus(ele.Handle, false)
	if g.vertical {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, resizableHandleSize)
	} else {
		ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, resizableHandleSize)
		ele.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
	}
	ele.SetCursor(resizableCursor(g.vertical))
	if g.disabled {
		ele.SetCursor(resizableCursorArrow())
		ele.EnableFocus(false)
		ele.EnableSwitchFocus(false)
	}
	h.Element = ele

	ele.AddEvent_Paint(h.onPaint)
	ele.AddEvent_LButtonDown(h.onDown)
	ele.AddEvent_LButtonDBClick(h.onDoubleClick)
	ele.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		ele.Redraw(false)
		return 0
	})
	ele.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		ele.Redraw(false)
		return 0
	})
	xui.AttachWindow(ele.Handle)

	g.handles = append(g.handles, h)
	g.scheduleSync()
	return h
}

// Sizes 返回各面板当前的百分比。
func (g *ResizableGroup) Sizes() []float64 {
	if g == nil {
		return nil
	}
	return append([]float64(nil), g.ensureSizes()...)
}

// SetSizes 按百分比设置各面板尺寸(长度必须等于面板数); 超出 min/max 的会被夹取,
// 总和自动归一到 100。
func (g *ResizableGroup) SetSizes(sizes ...float64) *ResizableGroup {
	if g == nil || len(sizes) != len(g.panels) || len(sizes) == 0 {
		return g
	}
	g.normalized = true
	g.sizes = normalizeResizableSizes(sizes, g.mins(), g.maxes())
	g.syncLayout()
	g.notify()
	return g
}

// OnResize 追加布局变化回调(与 WithResizableOnResize 等价)。
func (g *ResizableGroup) OnResize(fn func(sizes []float64)) *ResizableGroup {
	if g != nil && fn != nil {
		g.callbacks = append(g.callbacks, fn)
	}
	return g
}

// SetDisabled 开/关调整能力(拖拽、键盘、双击一起关)。禁用时分隔条改回普通箭头光标。
func (g *ResizableGroup) SetDisabled(v bool) *ResizableGroup {
	if g == nil || g.disabled == v {
		return g
	}
	g.disabled = v
	for _, h := range g.handles {
		if v {
			h.SetCursor(resizableCursorArrow())
			h.EnableFocus(false)
			h.EnableSwitchFocus(false)
		} else {
			h.SetCursor(resizableCursor(g.vertical))
			h.EnableFocus(true)
			h.EnableSwitchFocus(true)
		}
		h.Redraw(false)
	}
	return g
}

// Disabled 报告是否禁用调整。
func (g *ResizableGroup) Disabled() bool { return g != nil && g.disabled }

// ResizablePanel 是分组里的一个面板。本身是流式容器, 内容直接加在它上面;
// 尺寸由分组按百分比算出像素后锁成 Fixed, 不要自己再声明主轴尺寸。
type ResizablePanel struct {
	*widget.LayoutEle
	group *ResizableGroup
	size  float64 // 初始百分比(0 = 未指定)
	def   float64 // 双击复位的目标(只有 WithResizableSize 才会设)
	min   float64
	max   float64
	px    int32 // 当前像素尺寸(驱动布局变化检测)
}

// ContentStack 设置面板内子项间距并返回面板自身(与 Card.ContentStack 同名同义)。
func (p *ResizablePanel) ContentStack(space int32) *widget.LayoutEle {
	if p == nil || p.LayoutEle == nil {
		return nil
	}
	p.SetSpace(space)
	return p.LayoutEle
}

// ResizableHandle 是分隔条。默认画 1px 主题色线; WithResizableGrip 在中间加一根
// 4×24 的抓手段。可聚焦, 键盘 ←/→(纵向 ↑/↓) 每次 5%, Home/End 到相邻边界。
type ResizableHandle struct {
	*widget.Element
	group   *ResizableGroup
	grip    bool
	index   int // 调整 panels[index] 与 panels[index+1]
	lastSig int // 上一次绘制的状态(聚焦/收起/抓手), 变化时整块擦底
}

// ===== 尺寸计算 =====

// ensureSizes 首次计算默认份额: 有显式尺寸的面板按给定值, 其余平分剩余;
// 随后夹进 min/max 并归一到 100。
func (g *ResizableGroup) ensureSizes() []float64 {
	n := len(g.panels)
	if n == 0 {
		return nil
	}
	if g.normalized && len(g.sizes) == n {
		return g.sizes
	}
	sizes := make([]float64, n)
	for i, p := range g.panels {
		if p.size > 0 {
			sizes[i] = p.size
		}
	}
	g.sizes = normalizeResizableSizes(resizableInitialSizes(sizes), g.mins(), g.maxes())
	g.normalized = true
	return g.sizes
}

// resizableInitialSizes 给未指定(<=0)的面板平分剩余份额; 显式值已占满时未指定项按
// 100/n 参与, 交给归一化统一缩放。
func resizableInitialSizes(sizes []float64) []float64 {
	out := append([]float64(nil), sizes...)
	explicit, unset := 0.0, 0
	for _, s := range out {
		if s > 0 {
			explicit += s
		} else {
			unset++
		}
	}
	if unset == 0 {
		return out
	}
	rem := 100 - explicit
	if rem <= 0 {
		rem = 100
	}
	each := rem / float64(unset)
	for i := range out {
		if out[i] <= 0 {
			out[i] = each
		}
	}
	return out
}

func (g *ResizableGroup) mins() []float64 {
	out := make([]float64, len(g.panels))
	for i, p := range g.panels {
		out[i] = p.min
	}
	return out
}

func (g *ResizableGroup) maxes() []float64 {
	out := make([]float64, len(g.panels))
	for i, p := range g.panels {
		m := p.max
		if m <= 0 || m > 100 {
			m = 100
		}
		out[i] = m
	}
	return out
}

// normalizeResizableSizes 把一组百分比夹进 [min,max] 并归一到总和 100。
//
// 先等比缩放, 再按"还有余量的面板"分摊夹取造成的偏差(最多几轮): 例如 min 把某一项
// 顶大后, 多出来的部分从还有空间的面板上按均摊扣回, 不会让总和超过 100。
func normalizeResizableSizes(sizes, mins, maxes []float64) []float64 {
	n := len(sizes)
	out := make([]float64, n)
	if n == 0 {
		return out
	}
	lo := func(i int) float64 {
		if mins != nil && mins[i] > 0 {
			return math.Min(mins[i], 100)
		}
		return 0
	}
	hi := func(i int) float64 {
		h := 100.0
		if maxes != nil && maxes[i] > 0 && maxes[i] < 100 {
			h = maxes[i]
		}
		if h < lo(i) {
			h = lo(i)
		}
		return h
	}
	sum := 0.0
	for i := range out {
		v := sizes[i]
		if math.IsNaN(v) || v < 0 {
			v = 0
		}
		if v < lo(i) {
			v = lo(i)
		}
		if v > hi(i) {
			v = hi(i)
		}
		out[i] = v
		sum += v
	}
	if sum <= 1e-9 {
		each := 100 / float64(n)
		for i := range out {
			out[i] = each
			if out[i] < lo(i) {
				out[i] = lo(i)
			}
			if out[i] > hi(i) {
				out[i] = hi(i)
			}
		}
	} else {
		scale := 100 / sum
		for i := range out {
			out[i] *= scale
		}
		// 等比缩小可能把某一项压到 min 以下(或推到 max 以上), 先夹回去,
		// 偏差交给下面的"按余量分摊"修正。
		for i := range out {
			if out[i] < lo(i) {
				out[i] = lo(i)
			}
			if out[i] > hi(i) {
				out[i] = hi(i)
			}
		}
	}
	for pass := 0; pass < 8; pass++ {
		sum = 0
		for _, v := range out {
			sum += v
		}
		diff := 100 - sum
		if math.Abs(diff) < 1e-6 {
			break
		}
		var free []int
		for i := range out {
			if diff > 0 && out[i] < hi(i)-1e-9 {
				free = append(free, i)
			}
			if diff < 0 && out[i] > lo(i)+1e-9 {
				free = append(free, i)
			}
		}
		if len(free) == 0 {
			break
		}
		share := diff / float64(len(free))
		for _, i := range free {
			out[i] += share
			if out[i] < lo(i) {
				out[i] = lo(i)
			}
			if out[i] > hi(i) {
				out[i] = hi(i)
			}
		}
	}
	return out
}

// applyResizableDelta 把 index 处的分隔条移动 delta 个百分点: 正数让左侧面板变大、
// 右侧变小, 负数反之。空间不够时依次吞掉更远的面板(每块都保留 min), 增长端受 max 限制。
// 与上游一致: 手柄只影响它两侧的方向, 但相邻面板到边界后会继续挤压更远的同侧面板。
func applyResizableDelta(sizes []float64, index int, delta float64, mins, maxes []float64) []float64 {
	out := append([]float64(nil), sizes...)
	n := len(out)
	if n < 2 || index < 0 || index+1 >= n || math.Abs(delta) < 1e-9 {
		return out
	}
	lo := func(i int) float64 {
		if mins != nil && mins[i] > 0 {
			return math.Min(mins[i], 100)
		}
		return 0
	}
	hi := func(i int) float64 {
		h := 100.0
		if maxes != nil && maxes[i] > 0 && maxes[i] < 100 {
			h = maxes[i]
		}
		if h < lo(i) {
			h = lo(i)
		}
		return h
	}
	if delta > 0 {
		need := math.Min(delta, hi(index)-out[index])
		if need <= 0 {
			return out
		}
		taken := 0.0
		for j := index + 1; j < n && taken < need-1e-9; j++ {
			take := math.Min(need-taken, out[j]-lo(j))
			if take > 0 {
				out[j] -= take
				taken += take
			}
		}
		out[index] += taken
	} else {
		need := math.Min(-delta, hi(index+1)-out[index+1])
		if need <= 0 {
			return out
		}
		taken := 0.0
		for j := index; j >= 0 && taken < need-1e-9; j-- {
			take := math.Min(need-taken, out[j]-lo(j))
			if take > 0 {
				out[j] -= take
				taken += take
			}
		}
		out[index+1] += taken
	}
	return out
}

// resizablePixelWidths 把百分比换算成整数像素: 按累计边界取整, 各段之和恰好等于可用宽,
// 不会因为逐段 round 留下 1px 缝隙或溢出。
func resizablePixelWidths(sizes []float64, avail int32) []int32 {
	n := len(sizes)
	out := make([]int32, n)
	if n == 0 {
		return out
	}
	if avail < int32(n) {
		avail = int32(n)
	}
	acc := 0.0
	prev := int32(0)
	for i, s := range sizes {
		acc += s
		edge := int32(math.Round(acc / 100 * float64(avail)))
		if edge < prev+1 {
			edge = prev + 1
		}
		if edge > avail {
			edge = avail
		}
		out[i] = edge - prev
		prev = edge
	}
	return out
}

// ===== 布局 =====

func (g *ResizableGroup) mainSize() int32 {
	if g.vertical {
		return g.GetHeight()
	}
	return g.GetWidth()
}

func (g *ResizableGroup) availSize(main int32) int32 {
	avail := main - resizableHandleSize*int32(len(g.handles))
	if min := int32(len(g.panels)); avail < min {
		avail = min
	}
	return avail
}

// applyPixels 按当前百分比写面板/分隔条的像素尺寸; 返回是否有面板尺寸变化。
func (g *ResizableGroup) applyPixels(main int32) bool {
	sizes := g.ensureSizes()
	if len(sizes) == 0 {
		return false
	}
	widths := resizablePixelWidths(sizes, g.availSize(main))
	changed := false
	for i, p := range g.panels {
		px := widths[i]
		if p.px != px {
			p.px = px
			changed = true
		}
		if g.vertical {
			p.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, px)
			p.SetHeight(px)
		} else {
			p.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, px)
			p.SetWidth(px)
		}
	}
	for _, h := range g.handles {
		if g.vertical {
			h.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, resizableHandleSize)
			h.SetHeight(resizableHandleSize)
		} else {
			h.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, resizableHandleSize)
			h.SetWidth(resizableHandleSize)
		}
	}
	return changed
}

func (g *ResizableGroup) syncLayout() {
	if g == nil || g.LayoutEle == nil || !xc.XC_IsHXCGUI(g.Handle, xcc.XC_ELE) {
		return
	}
	main := g.mainSize()
	if main <= 0 {
		return
	}
	if g.applyPixels(main) {
		g.LayoutEle.AdjustLayout()
		g.LayoutEle.Redraw(false)
	}
}

func (g *ResizableGroup) onSize(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
	g.scheduleSync()
	return 0
}

// scheduleSync 把尺寸写回推到下一轮消息: 在布局趟里改子项尺寸会重入布局(同 AspectRatio,
// 见 DESIGN 第八节 94), 首次显示/窗口缩放时还会出现 0 宽过渡。
func (g *ResizableGroup) scheduleSync() {
	if g == nil || g.queued {
		return
	}
	g.queued = true
	app.AfterFunc(0, func() {
		g.queued = false
		g.syncLayout()
	})
}

func (g *ResizableGroup) notify() {
	if g == nil || len(g.callbacks) == 0 {
		return
	}
	for _, fn := range g.callbacks {
		fn(append([]float64(nil), g.sizes...))
	}
}

// ===== 拖拽 =====

func (h *ResizableHandle) onDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	g := h.group
	if g == nil || g.disabled {
		return 0
	}
	xui.NotePointerPress()
	if g.hWindow != 0 {
		xc.XWnd_SetFocusEle(g.hWindow, hEle)
	}
	g.beginDrag(h, pPt)
	return 0
}

func (g *ResizableGroup) beginDrag(h *ResizableHandle, pPt *xc.POINT) {
	sizes := g.ensureSizes()
	if h.index+1 >= len(sizes) {
		return
	}
	g.dragBase = append([]float64(nil), sizes...)
	g.dragIndex = h.index
	g.dragging = h

	// 手柄会跟着布局一起移动, 不能拿它自己的坐标做基准 —— 换算到分组客户区(拖动期间不动)。
	var hr, gr xc.RECT
	xc.XEle_GetWndClientRect(h.Handle, &hr)
	xc.XEle_GetWndClientRect(g.Handle, &gr)
	if g.vertical {
		g.dragStart = hr.Top + pPt.Y - gr.Top
	} else {
		g.dragStart = hr.Left + pPt.X - gr.Left
	}
	// 捕获到分组: 指针移出手柄/分组也能继续跟手, 且后续事件只送到分组(见 DESIGN 140)。
	xc.XEle_SetCapture(g.Handle, true)
}

func (g *ResizableGroup) onMouseMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if g.dragging == nil {
		return 0
	}
	if nFlags&1 == 0 { // 在窗口外松手时收不到 UP, 用 MK_LBUTTON 收尾(同 Carousel)
		g.endDrag()
		return 0
	}
	main := g.mainSize()
	avail := g.availSize(main)
	if avail <= 0 {
		return 0
	}
	var pos int32
	if g.vertical {
		pos = pPt.Y
	} else {
		pos = pPt.X
	}
	delta := float64(pos-g.dragStart) / float64(avail) * 100
	g.sizes = applyResizableDelta(g.dragBase, g.dragIndex, delta, g.mins(), g.maxes())
	if g.applyPixels(main) {
		g.LayoutEle.AdjustLayout()
		g.LayoutEle.Redraw(false)
	}
	g.notify()
	return 0
}

func (g *ResizableGroup) onMouseUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	g.endDrag()
	return 0
}

func (g *ResizableGroup) endDrag() {
	if g.dragging == nil {
		return
	}
	g.dragging = nil
	xc.XEle_SetCapture(g.Handle, false)
}

// ===== 双击复位 =====

func (h *ResizableHandle) onDoubleClick(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	g := h.group
	if g == nil || g.disabled {
		return 0
	}
	g.resetPair(h.index)
	return 0
}

// resetPair 对应上游双击: 把相邻面板里**有显式默认值**的第一个恢复成默认尺寸,
// 差额交给另一个相邻面板(再由约束夹取)。都没有显式默认值时不动作。
func (g *ResizableGroup) resetPair(index int) {
	sizes := g.ensureSizes()
	if index < 0 || index+1 >= len(sizes) {
		return
	}
	target, other := -1, -1
	if g.panels[index].def > 0 {
		target, other = index, index+1
	} else if g.panels[index+1].def > 0 {
		target, other = index+1, index
	}
	if target < 0 {
		return
	}
	next := append([]float64(nil), sizes...)
	want := g.panels[target].def
	if want < g.panels[target].min {
		want = g.panels[target].min
	}
	if want > g.panels[target].max && g.panels[target].max > 0 {
		want = g.panels[target].max
	}
	next[other] += sizes[target] - want
	next[target] = want
	g.sizes = normalizeResizableSizes(next, g.mins(), g.maxes())
	g.syncLayout()
	g.notify()
}

// ===== 键盘 =====

// bindKeys 在**窗口级**处理键盘: 手柄开了 EnableSwitchFocus(Tab 才能进入)后, 框架会先把
// 方向键当"切换焦点"执行, 元素级 KeyDown 里焦点已经跑掉(同 Tabs/Pagination, 见 DESIGN 152)。
func (g *ResizableGroup) bindKeys() {
	if g.hWindow == 0 {
		return
	}
	w := window.NewByHandle(g.hWindow)
	if w == nil {
		return
	}
	g.keyID = w.AddEvent_KeyDown(func(hWin int, wParam, lParam uintptr, pbHandled *bool) int {
		focus := xc.XWnd_GetFocusEle(hWin)
		for _, h := range g.handles {
			if h.Handle != focus {
				continue
			}
			if g.handleKey(h, wParam) {
				*pbHandled = true
			}
			return 0
		}
		return 0
	})
	// 回调挂在窗口上, 分组先销毁时不会自动摘掉 —— 按 ID 反注册(同 Pagination)。
	g.LayoutEle.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if g.keyID > 0 {
			if xc.XC_IsHXCGUI(g.hWindow, xcc.XC_WINDOW) {
				w.RemoveEvent(xcc.WM_KEYDOWN, g.keyID)
			}
			g.keyID = 0
		}
		return 0
	})
}

// handleKey 返回是否消费了按键。步骤对齐 react-resizable-panels v4: 方向键 5 个百分点、
// Home/End ±100 直接到边界; 只有分组轴向上的方向键生效。
func (g *ResizableGroup) handleKey(h *ResizableHandle, wParam uintptr) bool {
	if g == nil || g.disabled {
		return false
	}
	var delta float64
	switch wParam {
	case 0x25: // VK_LEFT
		if g.vertical {
			return false
		}
		delta = -resizableKeyStep
	case 0x27: // VK_RIGHT
		if g.vertical {
			return false
		}
		delta = resizableKeyStep
	case 0x26: // VK_UP
		if !g.vertical {
			return false
		}
		delta = -resizableKeyStep
	case 0x28: // VK_DOWN
		if !g.vertical {
			return false
		}
		delta = resizableKeyStep
	case 0x24: // VK_HOME
		delta = -resizableKeyJump
	case 0x23: // VK_END
		delta = resizableKeyJump
	default:
		return false
	}
	sizes := g.ensureSizes()
	if h.index+1 >= len(sizes) {
		return false
	}
	g.sizes = applyResizableDelta(sizes, h.index, delta, g.mins(), g.maxes())
	g.syncLayout()
	g.notify()
	return true
}

// ===== 绘制 =====

// onPaint 画 1px 分隔线(与 Separator 同款"中心设备像素"对齐, 见 DESIGN 第八节 112 末)、
// 可选抓手, 以及键盘焦点高亮。
//
// 几条实测结论:
//   - 抓手沿厚度轴的设备宽度取**奇数**: 4 逻辑像素在 150% 下是 6 设备像素, 偶数宽以
//     1 像素的线为中心永远一边多 1px(实测"还是偏左"), 5 设备像素才严格对称;
//   - 焦点态画**一条更粗的单线**(ring 色), 不用"环带 + 细线"或内缩描边 —— 4px 厚的
//     手柄上双色叠画会被看成两条平行线;
//   - 一侧面板收到 0 时, 分隔线画到手柄**外缘**(与分组边缘/外层容器边框重合)并隐藏
//     抓手: 线若停在手柄中心, 会距外框 2px, 看起来像"两条线"(见 DESIGN 第八节 157)。
func (h *ResizableHandle) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	focus := xui.StateOf(hEle).Focused && xui.FocusedByKeyboard() && (h.group == nil || !h.group.disabled)
	vertical := h.group != nil && h.group.vertical
	leadCollapsed, tailCollapsed := false, false
	if h.group != nil {
		leadCollapsed, tailCollapsed = h.group.collapsedSides(h.index)
	}

	// 透明元素重绘不擦底: 聚焦/收起/抓手显隐任一变化时整块擦一次(见 DESIGN 28),
	// 否则旧位置的细线、焦点粗线或抓手会留在画布上。
	sig := 0
	if focus {
		sig |= 1
	}
	if leadCollapsed {
		sig |= 2
	}
	if tailCollapsed {
		sig |= 4
	}
	if h.lastSig >= 0 && sig != h.lastSig {
		p.FillRound(&p.Rc, t.Card, 0)
	}
	h.lastSig = sig

	sc := p.Scale
	if sc <= 0 {
		sc = 1
	}

	// 分隔线的设备位置: 正常在 4px 手柄中心; 一侧收起时贴到手柄外缘。
	thickDev := int32(1)
	color := uint32(t.Border)
	if focus {
		thickDev = int32(math.Round(3 * sc))
		if thickDev < 3 {
			thickDev = 3
		}
		if thickDev%2 == 0 {
			thickDev++
		}
		color = uint32(t.Ring)
	}
	var centerDev float64
	var startDev, endDev float64
	var line xc.RECTF
	if vertical {
		centerDev = math.Floor(float64(p.Rc.Top+p.Rc.Bottom)/2*sc) + 0.5
		edgeDev := math.Round(float64(p.Rc.Top) * sc)
		if tailCollapsed {
			edgeDev = math.Round(float64(p.Rc.Bottom) * sc)
		}
		switch {
		case leadCollapsed:
			startDev, endDev = edgeDev, edgeDev+float64(thickDev)
		case tailCollapsed:
			startDev, endDev = edgeDev-float64(thickDev), edgeDev
		default:
			startDev, endDev = centerDev-float64(thickDev)/2, centerDev+float64(thickDev)/2
		}
		line = xc.RECTF{
			Left:   float32(p.Rc.Left),
			Top:    float32(startDev / sc),
			Right:  float32(p.Rc.Right),
			Bottom: float32(endDev / sc),
		}
	} else {
		centerDev = math.Floor(float64(p.Rc.Left+p.Rc.Right)/2*sc) + 0.5
		edgeDev := math.Round(float64(p.Rc.Left) * sc)
		if tailCollapsed {
			edgeDev = math.Round(float64(p.Rc.Right) * sc)
		}
		switch {
		case leadCollapsed:
			startDev, endDev = edgeDev, edgeDev+float64(thickDev)
		case tailCollapsed:
			startDev, endDev = edgeDev-float64(thickDev), edgeDev
		default:
			startDev, endDev = centerDev-float64(thickDev)/2, centerDev+float64(thickDev)/2
		}
		line = xc.RECTF{
			Left:   float32(startDev / sc),
			Top:    float32(p.Rc.Top),
			Right:  float32(endDev / sc),
			Bottom: float32(p.Rc.Bottom),
		}
	}
	p.Draw.FillRectColorF(&line, color)

	// 收起时抓手会有一半在分组外(被裁掉), 直接不画。
	if h.grip && !leadCollapsed && !tailCollapsed {
		// 厚度轴的设备宽度取奇数, 与线共用 centerDev; 长轴按设备像素取整后居中。
		shortDev := int32(math.Round(float64(resizableGripShort) * sc))
		if shortDev < 1 {
			shortDev = 1
		}
		if shortDev%2 == 0 {
			shortDev--
		}
		halfShort := float64(shortDev) / 2
		longDev := int32(math.Round(float64(resizableGripLong) * sc))
		if longDev < 1 {
			longDev = 1
		}
		var longCenterDev float64
		if vertical {
			longCenterDev = math.Floor(float64(p.Rc.Left+p.Rc.Right)/2*sc) + 0.5
		} else {
			longCenterDev = math.Floor(float64(p.Rc.Top+p.Rc.Bottom)/2*sc) + 0.5
		}
		longFrom := math.Round(longCenterDev - float64(longDev)/2)
		var rc xc.RECTF
		if vertical {
			rc = xc.RECTF{
				Left:   float32(longFrom / sc),
				Top:    float32((centerDev - halfShort) / sc),
				Right:  float32((longFrom + float64(longDev)) / sc),
				Bottom: float32((centerDev + halfShort) / sc),
			}
		} else {
			rc = xc.RECTF{
				Left:   float32((centerDev - halfShort) / sc),
				Top:    float32(longFrom / sc),
				Right:  float32((centerDev + halfShort) / sc),
				Bottom: float32((longFrom + float64(longDev)) / sc),
			}
		}
		p.Draw.SetBrushColor(uint32(t.Border))
		p.Draw.FillRoundRectF(&rc, float32(float64(shortDev)/2/sc), float32(float64(shortDev)/2/sc))
	}
	return 0
}

// collapsedSides 报告手柄两侧面板是否已经收到 0(像素上只剩 1px 的布局下限)。
func (g *ResizableGroup) collapsedSides(index int) (lead, tail bool) {
	if g == nil || index < 0 || index+1 >= len(g.panels) {
		return false, false
	}
	main := g.mainSize()
	if main <= 0 {
		return false, false
	}
	avail := float64(g.availSize(main))
	sizes := g.ensureSizes()
	if len(sizes) != len(g.panels) {
		return false, false
	}
	return sizes[index]/100*avail <= 1.5, sizes[index+1]/100*avail <= 1.5
}

// ===== 光标 =====

func resizableCursor(vertical bool) uintptr {
	id := wapi.IDC_SIZEWE
	if vertical {
		id = wapi.IDC_SIZENS
	}
	return wapi.LoadImageW(0, uintptr(id), wapi.IMAGE_CURSOR, 0, 0, wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)
}

func resizableCursorArrow() uintptr {
	return wapi.LoadImageW(0, uintptr(wapi.IDC_ARROW), wapi.IMAGE_CURSOR, 0, 0, wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)
}
