package ui

import (
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Sheet 侧边面板的几何常量(对齐 shadcn sheet.tsx, 单位逻辑像素)。
const (
	sheetPad        int32 = 16  // p-4 / SheetHeader/SheetFooter 的内边距
	sheetGap        int32 = 16  // gap-4: 头部/内容/页脚之间
	sheetHeadGap    int32 = 6   // SheetHeader 的 gap-1.5
	sheetTitleH     int32 = 24  // text-base 默认行高(1.5rem)
	sheetDescH      int32 = 20  // text-sm 默认行高(1.25rem)
	sheetCloseSize  int32 = 28  // 关闭按钮点击区(幽灵图标按钮)
	sheetCloseOff   int32 = 16  // top-4 right-4
	sheetMaxW       int32 = 384 // sm:max-w-sm(24rem)
	sheetBtnH       int32 = 32  // 页脚按钮 h-8
	sheetFooterGap  int32 = 8   // SheetFooter 的 gap-2
	sheetGrabberW   int32 = 100 // Drawer 抓手 w-[100px]
	sheetGrabberH   int32 = 8   // h-2
	sheetGrabberTop int32 = 8   // 抓手热区距面板顶(视觉胶囊在 16 处, 对齐 mt-4)
	sheetGrabberBox int32 = 24  // 抓手热区高(比胶囊高, 便于鼠标抓住)
)

// Sheet 的进出场时长。上游 sheet.tsx 写的是进场 duration-500 / 退场 duration-300;
// 桌面端没有移动端手势的缓冲需求, 500ms 会显得拖沓, 取 300/200 —— 观感仍是
// "从侧边滑入/滑出", 交互节奏更快。可用 WithSheetDuration 覆盖。
const (
	sheetOpenDuration  = 300 * time.Millisecond
	sheetCloseDuration = 200 * time.Millisecond
)

// SheetSide 是面板贴哪一边, 对应 shadcn sheet.tsx 的 side。
type SheetSide int

const (
	SheetRight  SheetSide = iota // 右侧(默认)
	SheetLeft                    // 左侧
	SheetTop                     // 顶部
	SheetBottom                  // 底部
)

// Sheet 从窗口边缘滑出的面板(遮罩 + 侧边内容), 语义对齐 shadcn sheet.tsx:
//   - 遮罩 bg-black/50, 点遮罩 / ESC / 右上角 X 关闭; Tab 焦点陷阱与 Dialog 相同;
//   - 左/右面板占满内容区高, 宽 min(3/4, sm:max-w-sm=384); 上/下面板宽度铺满, 高按内容自撑;
//   - 面板 bg-background, 贴边侧画 1px border(border-l / border-r / border-b / border-t);
//   - 头部 p-4、标题 text-base 中等字重 + 描述 text-sm muted(gap-1.5);
//   - 页脚在底部, 按钮纵向排列、等宽铺满(gap-2, p-4);
//   - 进场从贴边侧滑入 + 遮罩淡入, 退场反向滑出。
//
// 用法:
//
//	s := ui.NewSheet(w.Handle, ui.WithSheetSide(ui.SheetRight))
//	s.SetTitle("编辑资料")
//	s.SetDescription("修改你的公开资料")
//	// 内容建在 s.Content() 里(坐标相对内容区):
//	ui.NewLabel(0, 0, 0, 0, "显示名称", s.Content())
//	s.AddFooter("保存更改", ui.ButtonPrimary, func() { ... })
//	s.BindTrigger(btn).Show()
//
// 与上游的差异: 面板投影(shadow-lg)不做(xcgui 没有元素级圆角投影, 窗口自带阴影);
// 上/下面板内容超过面板高时裁剪而不是自动滚动(需要滚动时在 Content() 里放 ScrollArea)。
type Sheet struct {
	d *Dialog
}

// SheetOption 是 Sheet 的构造配置项。
type SheetOption func(*Sheet)

// WithSheetSide 设置面板方向(默认 SheetRight)。
func WithSheetSide(side SheetSide) SheetOption {
	return func(s *Sheet) { s.SetSide(side) }
}

// WithSheetWidth 覆盖左右面板的宽度(默认 min(内容区宽 × 3/4, 384))。
func WithSheetWidth(w int32) SheetOption {
	return func(s *Sheet) {
		if s != nil && s.d != nil {
			s.d.sheetW = w
			s.d.relayoutIfOpen()
		}
	}
}

// WithSheetCloseButton 是否显示右上角关闭按钮(默认显示)。
func WithSheetCloseButton(b bool) SheetOption {
	return func(s *Sheet) {
		if s == nil || s.d == nil {
			return
		}
		s.d.showClose = b
		if s.d.closeBtn != nil {
			s.d.closeBtn.Show(b)
			showEleTree(s.d.closeBtn.Handle, b && s.d.open)
		}
		s.d.relayoutIfOpen()
	}
}

// WithSheetDuration 覆盖进场/退场时长(0 = 用默认 300ms / 200ms)。
func WithSheetDuration(open, close time.Duration) SheetOption {
	return func(s *Sheet) {
		if s == nil || s.d == nil {
			return
		}
		if open > 0 {
			s.d.durOpen = open
		}
		if close > 0 {
			s.d.durClose = close
		}
	}
}

// NewSheet 创建侧边面板(默认隐藏)。hWnd 为宿主窗口句柄。
func NewSheet(hWnd int, opts ...SheetOption) *Sheet {
	d := NewDialog(hWnd)
	if d == nil {
		return nil
	}
	d.sheet = true
	d.sheetSide = SheetRight
	d.durOpen = sheetOpenDuration
	d.durClose = sheetCloseDuration
	d.sheetW = 0
	s := &Sheet{d: d}
	for _, o := range opts {
		o(s)
	}
	return s
}

// SetSide 切换贴边方向(打开状态下调用会重新布局并回到完成态)。
func (s *Sheet) SetSide(side SheetSide) *Sheet {
	if s == nil || s.d == nil {
		return s
	}
	s.d.sheetSide = side
	s.d.slideX, s.d.slideY = 0, 0
	s.d.appliedDX, s.d.appliedDY = 0, 0
	s.d.relayoutIfOpen()
	return s
}

// Side 返回当前贴边方向。
func (s *Sheet) Side() SheetSide {
	if s == nil || s.d == nil {
		return SheetRight
	}
	return s.d.sheetSide
}

// SetTitle 设置标题; 传空串隐藏标题。
func (s *Sheet) SetTitle(text string) *Sheet {
	if s != nil {
		s.d.SetTitle(text)
	}
	return s
}

// SetDescription 设置描述(text-sm muted, 按宽度折行); 传空串隐藏描述。
func (s *Sheet) SetDescription(text string) *Sheet {
	if s != nil {
		s.d.SetDescription(text)
	}
	return s
}

// Content 返回内容容器句柄: 子控件建在它里面(坐标相对内容区左上角)。
// 左/右面板的内容区高度是"面板剩余空间", 上/下面板按内容自撑。
func (s *Sheet) Content() int {
	if s == nil {
		return 0
	}
	return s.d.Content()
}

// ContentWidth 返回内容区宽度(面板宽 - 左右 p-4), 建子控件时作参考宽度。
func (s *Sheet) ContentWidth() int32 {
	if s == nil || s.d == nil {
		return 0
	}
	pw := s.d.sheetPanelW
	if pw <= 0 {
		pw = s.d.sheetW
	}
	if pw <= 0 {
		pw = sheetMaxW
	}
	w := pw - 2*sheetPad
	if w < 0 {
		w = 0
	}
	return w
}

// AddFooter 在底部追加一个铺满宽度的按钮(纵向排列, gap-2)。onClick 为 nil 时只画不绑事件。
func (s *Sheet) AddFooter(text string, variant ButtonVariant, onClick func()) *Sheet {
	if s == nil {
		return s
	}
	s.d.AddFooter(text, variant, onClick)
	return s
}

// BindTrigger 把控件绑成"点击即打开面板"(= shadcn 的 SheetTrigger)。
func (s *Sheet) BindTrigger(t clickBinder) *Sheet {
	if s != nil {
		s.d.BindTrigger(t)
	}
	return s
}

// BindClose 把任意可点击控件绑成"点击即关闭"(= shadcn 的 SheetClose)。
func (s *Sheet) BindClose(t clickBinder) *Sheet {
	if s != nil {
		s.d.BindClose(t)
	}
	return s
}

// Show 打开面板(从贴边侧滑入)。
func (s *Sheet) Show() *Sheet {
	if s != nil {
		s.d.Show()
	}
	return s
}

// Hide 关闭面板(不触发 OnClose 回调)。
func (s *Sheet) Hide() *Sheet {
	if s != nil {
		s.d.Hide()
	}
	return s
}

// Close 关闭面板并触发关闭回调。
func (s *Sheet) Close() *Sheet {
	if s != nil {
		s.d.Close()
	}
	return s
}

// IsOpen 报告面板是否打开。
func (s *Sheet) IsOpen() bool {
	return s != nil && s.d.IsOpen()
}

// OnClose 注册关闭回调(ESC / 点遮罩 / 关闭按钮 / Close 都会触发)。
func (s *Sheet) OnClose(fn func()) *Sheet {
	if s != nil {
		s.d.OnClose(fn)
	}
	return s
}

// Refresh 重新量内容并重排重绘(打开状态下改过内容后调用)。
func (s *Sheet) Refresh() *Sheet {
	if s != nil {
		s.d.Refresh()
	}
	return s
}

// ===== Dialog 中与 Sheet 共用的模态实现(本文件实现, dialog.go 只负责分支) =====

// sheetBodyRect 返回遮罩应覆盖的 Body 区(窗口阴影带之内)。与 Dialog 的
// 居中面板同一口径: 遮罩盖 Body 而不是整窗, 否则会把窗口自身投影再压暗。
func (d *Dialog) sheetBodyRect() (mx, my, ow, oh int32) {
	var wr xc.RECT
	xc.XWnd_GetRect(d.hWnd, &wr)
	ow, oh = wr.Right-wr.Left, wr.Bottom-wr.Top
	if ow <= 0 || oh <= 0 {
		return 0, 0, 0, 0
	}
	var bodyRc xc.RECT
	xc.XWnd_GetBodyRect(d.hWnd, &bodyRc)
	if bodyRc.Right > bodyRc.Left && bodyRc.Bottom > bodyRc.Top &&
		(bodyRc.Left > 0 || bodyRc.Top > 0 || bodyRc.Right < ow || bodyRc.Bottom < oh) {
		return bodyRc.Left, bodyRc.Top, bodyRc.Right - bodyRc.Left, bodyRc.Bottom - bodyRc.Top
	}
	return 0, 0, ow, oh
}

// layoutSheet 计算面板与全部子元素的最终位置, 再把当前滑入偏移量叠上去。
// 所有坐标都是遮罩局部坐标(遮罩贴在 Body 上, 原点不再是窗口 (0,0))。
func (d *Dialog) layoutSheet() {
	mx, my, ow, oh := d.sheetBodyRect()
	if ow <= 0 || oh <= 0 {
		return
	}
	xc.XEle_SetRect(d.overlay.Handle, &xc.RECT{Left: mx, Top: my, Right: mx + ow, Bottom: my + oh},
		false, xcc.AdjustLayout_No)

	pw := sheetPanelWidth(ow, d.sheetW)
	d.sheetPanelW = pw
	innerW := pw - 2*sheetPad
	if innerW < 1 {
		innerW = 1
	}

	textW := innerW
	if d.closeBtn != nil && d.showClose {
		textW -= sheetCloseSize + 8
		if textW < 1 {
			textW = 1
		}
	}

	titleH := d.textHeight(d.title, d.titleText, textW, sheetTitleH)
	descH := d.textHeight(d.desc, d.descText, textW, sheetDescH)
	headerH := int32(0)
	if titleH > 0 {
		headerH = titleH
	}
	if descH > 0 {
		if headerH > 0 {
			headerH += sheetHeadGap
		}
		headerH += descH
	}

	headerTop := int32(sheetPad)
	if d.drawer && d.sheetSide == SheetBottom {
		headerTop = sheetGrabberTop + sheetGrabberBox + sheetPad
	}
	contentTop := headerTop + headerH
	if headerH > 0 {
		contentTop += sheetGap
	}

	footerH := int32(0)
	if len(d.footers) > 0 {
		footerH = sheetPad*2 + int32(len(d.footers))*sheetBtnH +
			int32(len(d.footers)-1)*sheetFooterGap
	}

	sideH := d.sheetSide == SheetTop || d.sheetSide == SheetBottom
	var ph int32
	if sideH {
		contentH := int32(0)
		if d.content != nil {
			xc.XEle_AdjustLayout(d.content.Handle)
			contentH = childrenBottom(d.content.Handle)
		}
		body := headerH
		if contentH > 0 {
			if body > 0 {
				body += sheetGap
			}
			body += contentH
		}
		if footerH > 0 {
			if body > 0 {
				body += sheetGap
			}
			body += footerH
		}
		ph = headerTop + body + sheetPad
		maxH := oh
		if d.drawer { // Drawer 的 max-h-[80vh]
			maxH = oh * 4 / 5
		}
		if ph > maxH {
			ph = maxH
		}
	} else {
		ph = oh
	}

	var px, py int32
	switch d.sheetSide {
	case SheetLeft:
		px, py = 0, 0
	case SheetTop:
		px, py = 0, 0
	case SheetBottom:
		px, py = 0, oh-ph
	default: // SheetRight
		px, py = ow-pw, 0
	}
	panel := xc.RECT{Left: px, Top: py, Right: px + pw, Bottom: py + ph}
	d.panel = panel
	d.footerRc = xc.RECT{}

	// 内容区高度: 左右面板吃掉"头部与页脚之间"的剩余空间; 上下面板在量高时已按内容撑开。
	contentH := int32(0)
	if !sideH {
		contentTop = py + headerTop + headerH
		if headerH > 0 {
			contentTop += sheetGap
		}
		bottom := panel.Bottom - sheetPad
		if footerH > 0 {
			bottom = panel.Bottom - footerH - sheetGap
		}
		if bottom > contentTop {
			contentH = bottom - contentTop
		}
	} else {
		contentTop = py + contentTop
		if d.content != nil {
			xc.XEle_AdjustLayout(d.content.Handle)
			contentH = childrenBottom(d.content.Handle)
		}
		if footerH > 0 {
			maxContent := panel.Bottom - footerH - sheetGap - contentTop
			if maxContent < 0 {
				maxContent = 0
			}
			if contentH > maxContent {
				contentH = maxContent
			}
		} else {
			maxContent := panel.Bottom - sheetPad - contentTop
			if maxContent < 0 {
				maxContent = 0
			}
			if contentH > maxContent {
				contentH = maxContent
			}
		}
	}

	// 关闭按钮: 面板右上角, 内缩 16(top-4 right-4)。
	if d.closeBtn != nil {
		showEleTree(d.closeBtn.Handle, d.showClose && d.open)
		if d.showClose {
			ch := d.closeBtn.GetHeight()
			setEleRect(d.closeBtn.Handle, panel.Right-sheetCloseOff-sheetCloseSize,
				panel.Top+sheetCloseOff, sheetCloseSize, ch)
		}
	}
	// 抓手: 只有底部 Drawer 显示(对齐 vaul 的 group-data-[direction=bottom] 规则)。
	if d.dragger != nil {
		show := d.drawer && d.sheetSide == SheetBottom
		showEleTree(d.dragger.Handle, show && d.open)
		if show {
			setEleRect(d.dragger.Handle, panel.Left+(pw-sheetGrabberW)/2,
				panel.Top+sheetGrabberTop, sheetGrabberW, sheetGrabberBox)
		}
	}

	textX := panel.Left + sheetPad
	y := panel.Top + headerTop
	if titleH > 0 {
		setEleRect(d.title.Handle, textX, y-dlgTextOver, textW, titleH+2*dlgTextOver)
		d.title.SetText(d.titleText)
		y += titleH
	}
	if descH > 0 {
		if titleH > 0 {
			y += sheetHeadGap
		}
		setEleRect(d.desc.Handle, textX, y-dlgTextOver, textW, descH+2*dlgTextOver)
		d.desc.SetText(d.descText)
		y += descH
	}
	if d.content != nil {
		showContent := contentH > 0
		showEleTree(d.content.Handle, showContent && d.open)
		if showContent {
			setEleRect(d.content.Handle, panel.Left+sheetPad, contentTop, innerW, contentH)
		}
	}
	if footerH > 0 {
		by := panel.Bottom - footerH + sheetPad
		for _, b := range d.footers {
			setEleRect(b.Handle, panel.Left+sheetPad, by, innerW, sheetBtnH)
			by += sheetBtnH + sheetFooterGap
		}
	}

	// 缩放/换页时布局会重跑: 先把之前叠加的偏移清掉, 再按当前动画进度平移一次。
	d.appliedDX, d.appliedDY = 0, 0
	d.applyPanelOffset(d.slideX, d.slideY)
}

// sheetPanelWidth 按内容区宽算左右面板宽: 默认 min(3/4, sm:max-w-sm=384),
// want > 0 时用调用方给的值并夹到内容区宽以内。
func sheetPanelWidth(bodyW, want int32) int32 {
	pw := want
	if pw <= 0 {
		pw = bodyW * 3 / 4
		if pw > sheetMaxW {
			pw = sheetMaxW
		}
	}
	if pw > bodyW {
		pw = bodyW
	}
	if pw < 1 {
		pw = 1
	}
	return pw
}

// applyPanelOffset 整体平移面板与它的直接子元素(子元素坐标随父元素移动)。
// 只处理面板第一层子元素: 内容容器里的控件相对内容容器定位, 会跟着它走。
func (d *Dialog) applyPanelOffset(dx, dy int32) {
	ddx, ddy := dx-d.appliedDX, dy-d.appliedDY
	if ddx == 0 && ddy == 0 {
		return
	}
	d.panel.Left += ddx
	d.panel.Top += ddy
	d.panel.Right += ddx
	d.panel.Bottom += ddy
	if d.footerRc.Bottom > d.footerRc.Top {
		d.footerRc.Left += ddx
		d.footerRc.Top += ddy
		d.footerRc.Right += ddx
		d.footerRc.Bottom += ddy
	}
	shift := func(h int) {
		if h == 0 || !xc.XWidget_IsShow(h) {
			return
		}
		var rc xc.RECT
		xc.XEle_GetRect(h, &rc)
		rc.Left += ddx
		rc.Top += ddy
		rc.Right += ddx
		rc.Bottom += ddy
		xc.XEle_SetRect(h, &rc, false, xcc.AdjustLayout_No)
	}
	if d.closeBtn != nil {
		shift(d.closeBtn.Handle)
	}
	shift(d.title.Handle)
	shift(d.desc.Handle)
	if d.content != nil {
		shift(d.content.Handle)
	}
	if d.dragger != nil {
		shift(d.dragger.Handle)
	}
	for _, b := range d.footers {
		shift(b.Handle)
	}
	d.appliedDX, d.appliedDY = dx, dy
}

// sheetSlideStart 返回面板"完全贴在屏幕外"时的偏移量(像素)。
func (d *Dialog) sheetSlideStart() (int32, int32) {
	w := d.panel.Right - d.panel.Left
	h := d.panel.Bottom - d.panel.Top
	switch d.sheetSide {
	case SheetLeft:
		return -w, 0
	case SheetTop:
		return 0, -h
	case SheetBottom:
		return 0, h
	default:
		return w, 0
	}
}

// paintSheet 画遮罩与面板: bg-black/50, 面板 bg-background + 贴边侧 1px 边框。
func (d *Dialog) paintSheet(p *xui.Painter) {
	t := xui.Current()
	p.FillRound(&p.Rc, xui.RGBA(0, 0, 0, 128), 0)

	panel := d.panel
	if panel.Right <= panel.Left || panel.Bottom <= panel.Top {
		return
	}
	radius := int32(0)
	bottomRound := d.drawer && (d.sheetSide == SheetTop || d.sheetSide == SheetBottom)
	if bottomRound {
		radius = t.RadiusLg()
	}
	p.FillRound(&panel, t.Background, radius)
	switch d.sheetSide {
	case SheetLeft:
		p.Line(panel.Right-1, panel.Top, panel.Right-1, panel.Bottom, t.Border, 1)
	case SheetTop:
		p.Line(panel.Left, panel.Bottom-1, panel.Right, panel.Bottom-1, t.Border, 1)
	case SheetBottom:
		// 圆角端点内缩, 不让直线压到圆角外的遮罩上
		p.Line(panel.Left+radius, panel.Top, panel.Right-radius, panel.Top, t.Border, 1)
	default:
		p.Line(panel.Left, panel.Top, panel.Left, panel.Bottom, t.Border, 1)
	}
}

// showSheet 打开面板: 先按最终位置排版, 再整体推到屏幕外, 由 Presence 逐帧滑入。
func (d *Dialog) showSheet() *Dialog {
	d.open = true
	d.pushModal()
	d.slideX, d.slideY = 0, 0
	d.appliedDX, d.appliedDY = 0, 0
	d.materialize()

	dx, dy := d.sheetSlideStart()
	d.slideX, d.slideY = dx, dy
	d.applyPanelOffset(dx, dy)
	xc.XEle_SetAlpha(d.overlay.Handle, 0)
	d.overlay.Redraw(false)
	xc.XWnd_Redraw(d.hWnd, true)

	d.focusIdx = 0
	fs := d.focusables()
	focus := d.defaultFocus(fs)
	if focus != 0 {
		d.setCaretWidthOf(focus, 0)
		xc.XWnd_SetFocusEle(d.hWnd, focus)
		d.lastFocus = focus
	}
	dur := d.durOpen
	if dur <= 0 {
		dur = sheetOpenDuration
	}
	xui.Presence(d.hWnd, d.fadeKey, true, dur, func(v float64) {
		if !d.open || d.dragging {
			return
		}
		xc.XEle_SetAlpha(d.overlay.Handle, byte(v*255+0.5))
		d.slideX = int32(float64(dx)*(1-v) + 0.5)
		d.slideY = int32(float64(dy)*(1-v) + 0.5)
		d.applyPanelOffset(d.slideX, d.slideY)
	}, func(float64) {
		if focus != 0 {
			d.setCaretWidthOf(focus, 1)
			if xc.XWnd_GetFocusEle(d.hWnd) == focus {
				xc.XWnd_SetFocusEle(d.hWnd, 0)
				xc.XWnd_SetFocusEle(d.hWnd, focus)
			}
		}
	})
	return d
}

// hideSheet 关闭面板: 从当前位置滑出, 到位后隐藏整组。
func (d *Dialog) hideSheet() *Dialog {
	d.open = false
	d.popModal()
	d.suspended = false
	xc.XWnd_SetFocusEle(d.hWnd, 0)
	restoreModalFocus(d.hWnd)
	d.animateSheetClose(d.slideX, d.slideY)
	return d
}

// animateSheetClose 从 (startX,startY) 滑到屏幕外并淡出遮罩, 到位后隐藏。
func (d *Dialog) animateSheetClose(startX, startY int32) {
	dx, dy := d.sheetSlideStart()
	dur := d.durClose
	if dur <= 0 {
		dur = sheetCloseDuration
	}
	xui.Presence(d.hWnd, d.fadeKey, false, dur, func(v float64) {
		xc.XEle_SetAlpha(d.overlay.Handle, byte(v*255+0.5))
		d.slideX = startX + int32(float64(dx-startX)*(1-v)+0.5)
		d.slideY = startY + int32(float64(dy-startY)*(1-v)+0.5)
		d.applyPanelOffset(d.slideX, d.slideY)
	}, func(float64) {
		d.finishSheetHide()
	})
}

// finishSheetHide 隐藏面板子树并把偏移/透明度复位(下次 Show 从这里重新开始)。
func (d *Dialog) finishSheetHide() {
	showEleTree(d.overlay.Handle, false)
	d.slideX, d.slideY = 0, 0
	d.appliedDX, d.appliedDY = 0, 0
	xc.XEle_SetAlpha(d.overlay.Handle, 255)
	xc.XWnd_Redraw(d.hWnd, true)
}

// addSheetGrabber 给 Drawer 创建顶部抓手(100×8 圆角胶囊, 热区 100×24)。
func (d *Dialog) addSheetGrabber() {
	if d == nil || d.dragger != nil {
		return
	}
	e := widget.NewElement(0, 0, sheetGrabberW, sheetGrabberBox, d.overlay.Handle)
	if e == nil {
		return
	}
	e.EnableBkTransparent(true)
	e.EnableDrawBorder(false)
	e.EnableFocus(false)
	e.EnableSwitchFocus(false)
	e.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		rc := p.Rc
		rc.Top = (rc.Top + rc.Bottom - sheetGrabberH) / 2
		rc.Bottom = rc.Top + sheetGrabberH
		p.FillRound(&rc, xui.Current().Muted, sheetGrabberH/2)
		return 0
	})
	d.dragger = e
	showEleTree(e.Handle, false)
}
