package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Popover 几何常量(逻辑像素), 对齐 shadcn popover.tsx:
// w-72(288) / p-4(16) / sideOffset=4。
const (
	popoverDefaultW int32 = 288
	popoverPad      int32 = 16
	popoverGap      int32 = 4
)

// PopoverOption 弹出层配置项。
type PopoverOption func(*Popover)

// WithPopoverContent 设置内容构建回调(content 是纵向流式容器, 宽铺满、高按内容自撑)。
func WithPopoverContent(build func(content int)) PopoverOption {
	return func(p *Popover) { p.build = build }
}

// WithPopoverWidth 覆盖卡片宽度(默认 288 = w-72)。
func WithPopoverWidth(w int32) PopoverOption {
	return func(p *Popover) {
		if w > 0 {
			p.width = w
		}
	}
}

// WithPopoverHeight 固定卡片高度(默认 0 = 按内容自适应)。
func WithPopoverHeight(h int32) PopoverOption {
	return func(p *Popover) {
		if h > 0 {
			p.height = h
		}
	}
}

// WithPopoverPadding 覆盖四周内边距(默认 16 = p-4)。
func WithPopoverPadding(pad int32) PopoverOption {
	return func(p *Popover) {
		if pad >= 0 {
			p.padding = pad
		}
	}
}

// WithPopoverGap 设置弹层与触发区的间距(默认 4 = sideOffset)。
func WithPopoverGap(gap int32) PopoverOption {
	return func(p *Popover) { p.gap = gap }
}

// WithPopoverSide 设置优先方位(默认下方; 放不下会自动翻到对侧)。
func WithPopoverSide(s PopoverSide) PopoverOption {
	return func(p *Popover) { p.side = s }
}

// WithPopoverAlign 设置交叉轴对齐(默认居中)。
func WithPopoverAlign(a PopoverAlign) PopoverOption {
	return func(p *Popover) { p.align = a }
}

// WithPopoverOnOpenChange 注册开关状态回调。
func WithPopoverOnOpenChange(fn func(open bool)) PopoverOption {
	return func(p *Popover) { p.changes = append(p.changes, fn) }
}

// Popover 点击触发的通用弹出卡片, 对应 shadcn popover.tsx。
//
// 与 shadcn 对齐:
//   - 内容 bg-popover + 1px ring + rounded-lg + shadow-md, 默认宽 288、内边距 16;
//   - 点击触发区开合; 点击外部(失焦)或 ESC 关闭, ESC 关闭后焦点回到触发区;
//   - 方位于下方优先, 空间不足自动翻到上方 / 左侧 / 右侧, 并夹在显示器工作区内;
//   - 内容高度按子项自然高度自适应(宽度固定), 也可用 WithPopoverHeight 固定。
//
// 与 shadcn 的差异: 进场只做淡入, 不做 zoom/slide —— 自建阴影窗的元素 alpha
// 是唯一稳定的动画通道(见 popup.go 的说明)。
//
// 用法:
//
//	pop := ui.NewPopover(btn.Handle, ui.WithPopoverContent(func(c int) {
//	    h := ui.NewPopoverHeader(c)
//	    ui.NewPopoverTitle(h.Handle, "标题")
//	    ui.NewPopoverDescription(h.Handle, "说明文字")
//	}))
type Popover struct {
	trigger int
	build   func(content int)

	width   int32
	height  int32
	padding int32
	gap     int32
	side    PopoverSide
	align   PopoverAlign

	panel    *popoverPanel
	content  *widget.LayoutEle
	built    bool
	measured bool

	changes []func(open bool)
}

// NewPopover 把弹出卡片绑定到任意元素(通常是一个 Button)。hTrigger 必须有效。
func NewPopover(hTrigger int, opts ...PopoverOption) *Popover {
	if hTrigger == 0 {
		return nil
	}
	p := &Popover{
		trigger: hTrigger,
		width:   popoverDefaultW,
		padding: popoverPad,
		gap:     popoverGap,
	}
	for _, o := range opts {
		o(p)
	}

	toggle := func() {
		if p.panel != nil && p.panel.RecentlyClosed() {
			return
		}
		if p.IsOpen() {
			p.Close()
			return
		}
		p.Open()
	}
	// 原生按钮同时支持鼠标点击与空格/回车; 普通元素只有鼠标事件可借。
	if xc.XObj_GetType(hTrigger) == xcc.XC_BUTTON {
		if btn := widget.NewButtonByHandle(hTrigger); btn != nil {
			btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
				toggle()
				return 0
			})
		}
	} else if ele := widget.NewElementByHandle(hTrigger); ele != nil {
		ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			xui.NotePointerPress()
			toggle()
			return 0
		})
	}
	if ele := widget.NewElementByHandle(hTrigger); ele != nil {
		ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
			if p.panel != nil {
				p.panel.Destroy()
			}
			p.panel, p.content = nil, nil
			return 0
		})
	}
	return p
}

// Content 返回内容容器句柄(首次调用时构建内容)。未设置构建回调时返回 0。
func (p *Popover) Content() int {
	if p == nil || !p.ensureBuilt() {
		return 0
	}
	return p.content.Handle
}

// Open 打开弹层。
func (p *Popover) Open() *Popover {
	if p == nil || !p.ensureBuilt() {
		return p
	}
	if p.panel.IsOpen() {
		return p
	}
	if !p.measured {
		p.measure()
	}
	x, y := p.placement()
	p.panel.ShowAt(p.trigger, x, y)
	p.notify(true)
	return p
}

// Close 关闭弹层。
func (p *Popover) Close() *Popover {
	if p != nil && p.panel != nil {
		p.panel.Hide()
	}
	return p
}

// SetOpen 按参数打开/关闭弹层。
func (p *Popover) SetOpen(open bool) *Popover {
	if open {
		return p.Open()
	}
	return p.Close()
}

// IsOpen 报告弹层是否打开。
func (p *Popover) IsOpen() bool { return p != nil && p.panel != nil && p.panel.IsOpen() }

// Refresh 重新量一次内容高度(内容子项增减后调用), 打开中时保持在原位置。
func (p *Popover) Refresh() *Popover {
	if p == nil || !p.built || p.panel == nil {
		return p
	}
	if p.panel.IsOpen() {
		// 打开中只重排内容并重绘, 不做离屏测量(测量会先把窗口挪到 -30000 再隐藏)
		p.content.AdjustLayout()
		if p.panel.alive() {
			p.panel.win.Redraw(true)
		}
		return p
	}
	p.measure()
	return p
}

// measure 量一次内容高度/应用固定高度; 首次 Open 与 Refresh 时调用。
func (p *Popover) measure() {
	if p.height > 0 {
		p.panel.resize(p.width, p.height)
		p.measured = true
		return
	}
	if h := p.panel.MeasureContent(p.padding); h <= 0 {
		// 空内容也保留 p-4 的上下留白, 与上游空卡片一致
		p.panel.resize(p.width, p.padding*2)
	}
	p.measured = true
}

// ensureBuilt 懒创建弹层与内容(尺寸只在第一次构建时量, 之后用 Refresh 更新)。
func (p *Popover) ensureBuilt() bool {
	if p.built {
		return p.panel != nil
	}
	p.built = true
	ch := p.height
	if ch <= 0 {
		ch = p.padding*2 + 1 // 临时高度; 内容量出来后 resize
	}
	panel := newPopoverPanelEx(p.width, ch, p.trigger, true, true)
	if panel == nil {
		return false
	}
	content := panel.LayoutContent(p.padding, 8)
	if content == nil {
		panel.Destroy()
		return false
	}
	p.panel, p.content = panel, content
	panel.OnEsc = func() {
		if t := widget.NewElementByHandle(p.trigger); t != nil {
			t.SetFocus()
		}
	}
	panel.OnHide = func() { p.notify(false) }
	if p.build != nil {
		p.build(content.Handle)
	}
	return true
}

func (p *Popover) notify(open bool) {
	for _, fn := range p.changes {
		fn(open)
	}
}

// placement 计算弹层左上角(屏幕逻辑像素, 与 popoverPanel.ShowAt 同口径), 并处理翻转与工作区夹取。
func (p *Popover) placement() (int32, int32) {
	return popupPosition(p.trigger, p.width, p.panel.ch, p.side, p.align, p.gap)
}

// PopoverHeader 弹层标题区: 纵向流, 间距 4(gap-1)。
func PopoverHeader(hParent int) *widget.LayoutEle {
	h := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if h == nil {
		return nil
	}
	h.EnableHorizon(false)
	h.EnableLayout(true)
	h.EnableBkTransparent(true)
	h.EnableAutoWrap(false)
	h.SetSpace(4)
	h.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	h.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return h
}

// PopoverTitle 弹层标题: font-medium, text-sm。
func PopoverTitle(hParent int, text string) *Label {
	l := NewLabel(0, 0, 0, 20, text, hParent).SetMedium().SetFontSize(14)
	xui.ItemFill(l)
	return l
}

// PopoverDescription 弹层说明: text-sm muted。
func PopoverDescription(hParent int, text string) *Label {
	l := NewLabel(0, 0, 0, 20, text, hParent).SetMuted().SetFontSize(14)
	xui.ItemFill(l)
	return l
}
