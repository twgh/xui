package ui

import (
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

// Drawer 底部抽屉: 语义对齐 shadcn drawer.tsx(vaul):
//   - 默认从底部弹出, 面板 max-h-[80vh]、上方两个圆角、顶部 1px 边框;
//   - 顶部一根 100×8 的抓手(mt-4), 按住向下拖过面板高度 1/3 即关闭, 否则回弹;
//   - 头部/页脚与 Sheet 相同(p-4、标题 + 描述、页脚按钮纵向铺满);
//   - 遮罩 bg-black/50, 点遮罩 / ESC 关闭(vaul 默认行为)。
//
// 方向可经 WithDrawerDirection 改成 top/left/right(左右与 Sheet 一致; 抓手只在底部出现)。
// 与上游的差异: vaul 允许从内容任意位置起拖, 桌面上会与文本选择/按钮点击冲突,
// 这里只允许从抓手起拖; 拖拽不跟手做橡皮筋阻尼(offset 直接跟手, 松手后回弹)。
type Drawer struct {
	s *Sheet
}

// DrawerOption 是 Drawer 的构造配置项。
type DrawerOption func(*Drawer)

// WithDrawerDirection 设置弹出方向(默认 SheetBottom)。
func WithDrawerDirection(side SheetSide) DrawerOption {
	return func(dr *Drawer) {
		if dr != nil && dr.s != nil {
			dr.s.SetSide(side)
		}
	}
}

// WithDrawerTitle 设置标题(等价于构造后调 SetTitle)。
func WithDrawerTitle(text string) DrawerOption {
	return func(dr *Drawer) { dr.SetTitle(text) }
}

// WithDrawerDescription 设置描述(等价于构造后调 SetDescription)。
func WithDrawerDescription(text string) DrawerOption {
	return func(dr *Drawer) { dr.SetDescription(text) }
}

// NewDrawer 创建抽屉(默认隐藏)。hWnd 为宿主窗口句柄。
func NewDrawer(hWnd int, opts ...DrawerOption) *Drawer {
	s := NewSheet(hWnd, WithSheetSide(SheetBottom), WithSheetCloseButton(false))
	if s == nil {
		return nil
	}
	d := s.d
	d.drawer = true
	d.addSheetGrabber()
	if d.dragger != nil {
		d.dragger.AddEvent_LButtonDown(d.onGrabberDown)
		d.dragger.AddEvent_MouseMove(d.onGrabberMove)
		d.dragger.AddEvent_LButtonUp(d.onGrabberUp)
	}
	dr := &Drawer{s: s}
	for _, o := range opts {
		o(dr)
	}
	return dr
}

// SetTitle 设置标题; 传空串隐藏标题。
func (dr *Drawer) SetTitle(text string) *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.SetTitle(text)
	}
	return dr
}

// SetDescription 设置描述(text-sm muted); 传空串隐藏描述。
func (dr *Drawer) SetDescription(text string) *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.SetDescription(text)
	}
	return dr
}

// SetDirection 切换弹出方向(打开状态下调用会重新布局并回到完成态)。
func (dr *Drawer) SetDirection(side SheetSide) *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.SetSide(side)
	}
	return dr
}

// Direction 返回当前弹出方向。
func (dr *Drawer) Direction() SheetSide {
	if dr == nil || dr.s == nil {
		return SheetBottom
	}
	return dr.s.Side()
}

// Content 返回内容容器句柄(坐标相对内容区左上角)。
func (dr *Drawer) Content() int {
	if dr == nil || dr.s == nil {
		return 0
	}
	return dr.s.Content()
}

// ContentWidth 返回内容区宽度, 建子控件时作参考宽度。
func (dr *Drawer) ContentWidth() int32 {
	if dr == nil || dr.s == nil {
		return 0
	}
	return dr.s.ContentWidth()
}

// AddFooter 在底部追加一个铺满宽度的按钮(纵向排列, gap-2)。
func (dr *Drawer) AddFooter(text string, variant ButtonVariant, onClick func()) *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.AddFooter(text, variant, onClick)
	}
	return dr
}

// BindTrigger 把控件绑成"点击即打开"(= vaul 的 DrawerTrigger)。
func (dr *Drawer) BindTrigger(t clickBinder) *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.BindTrigger(t)
	}
	return dr
}

// BindClose 把任意可点击控件绑成"点击即关闭"(= vaul 的 DrawerClose)。
func (dr *Drawer) BindClose(t clickBinder) *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.BindClose(t)
	}
	return dr
}

// Show 打开抽屉(从所选方向滑入)。
func (dr *Drawer) Show() *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.Show()
	}
	return dr
}

// Hide 关闭抽屉(不触发 OnClose 回调)。
func (dr *Drawer) Hide() *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.Hide()
	}
	return dr
}

// Close 关闭抽屉并触发关闭回调。
func (dr *Drawer) Close() *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.Close()
	}
	return dr
}

// IsOpen 报告抽屉是否打开。
func (dr *Drawer) IsOpen() bool {
	return dr != nil && dr.s != nil && dr.s.IsOpen()
}

// OnClose 注册关闭回调(ESC / 点遮罩 / 拖拽关闭 / Close 都会触发)。
func (dr *Drawer) OnClose(fn func()) *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.OnClose(fn)
	}
	return dr
}

// Refresh 重新量内容并重排重绘。
func (dr *Drawer) Refresh() *Drawer {
	if dr != nil && dr.s != nil {
		dr.s.Refresh()
	}
	return dr
}

// ===== 拖拽关闭 =====

// dragScale 返回逻辑像素/物理像素的比值(与窗口 DPI 一致)。
func (d *Dialog) dragScale() float64 {
	dpi := xc.XWnd_GetDPI(d.hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	return float64(dpi) / 96
}

// cursorY 返回光标的屏幕 Y 坐标(物理像素)。
func cursorY() int32 {
	var pt wapi.POINT
	if !wapi.GetCursorPos(&pt) {
		return 0
	}
	return pt.Y
}

// onGrabberDown 开始拖拽: 记录起点与当前偏移, 捕获鼠标。
func (d *Dialog) onGrabberDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if d == nil || !d.open || d.sheetSide != SheetBottom {
		return 0
	}
	*pbHandled = true
	// ⚠️ 不要在这里 DropTransitions: 它会把主 presence 条目一起删掉, 之后拖拽关闭
	// 调 Presence(visible=false) 时目标 0 会被当成"从未活跃过"直接返回, done 永远
	// 不执行 —— 遮罩停在 alpha 0 但元素仍可见, 之后整页点击都被它吞掉(实测)。
	// 正在跑的滑入帧由 showSheet 的 apply 里 d.dragging 判断跳过。
	d.dragging = true
	d.dragStartY = cursorY()
	// 动画中途起拖时从当前偏移继续, 不回跳
	d.dragBaseY = d.slideY
	if d.dragger != nil {
		d.dragger.SetCapture(true)
	}
	return 0
}

// onGrabberMove 拖拽中: 面板跟随光标下移(限制在 0..面板高)。
func (d *Dialog) onGrabberMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if d == nil || !d.dragging {
		return 0
	}
	*pbHandled = true
	sc := d.dragScale()
	dy := int32(float64(cursorY()-d.dragStartY)/sc + 0.5)
	off := d.dragBaseY + dy
	if off < 0 {
		off = 0
	}
	ph := d.panel.Bottom - d.panel.Top
	// d.panel 已含当前偏移, 面板高不受影响
	if off > ph {
		off = ph
	}
	d.slideX = 0
	d.slideY = off
	d.applyPanelOffset(d.slideX, d.slideY)
	xc.XWnd_Redraw(d.hWnd, true)
	return 0
}

// onGrabberUp 松手: 超过 1/3 面板高就关闭, 否则回弹到打开位置。
func (d *Dialog) onGrabberUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if d == nil || !d.dragging {
		return 0
	}
	*pbHandled = true
	d.dragging = false
	if d.dragger != nil {
		d.dragger.SetCapture(false)
	}
	ph := d.panel.Bottom - d.panel.Top
	if ph <= 0 || d.slideY <= 2 {
		// 轻点/抖动留下 1~2px 偏移也要归位, 否则面板会永远矮一截
		d.slideX, d.slideY = 0, 0
		d.applyPanelOffset(0, 0)
		xc.XWnd_Redraw(d.hWnd, true)
		return 0
	}
	if d.slideY*3 > ph {
		// 拖拽关闭: 先把模态状态收掉(键盘/遮罩立即失效), 再从当前位置滑出。
		d.open = false
		d.popModal()
		xc.XWnd_SetFocusEle(d.hWnd, 0)
		restoreModalFocus(d.hWnd)
		d.animateSheetClose(d.slideX, d.slideY)
		for _, fn := range d.onClose {
			fn()
		}
		return 0
	}
	// 回弹: 用一个临时键做"当前偏移 → 0"的短过渡, 不打扰主 presence 键的终态。
	start := d.slideY
	key := xui.NextPresenceKey()
	dur := d.durClose
	if dur <= 0 {
		dur = sheetCloseDuration
	}
	xui.Presence(d.hWnd, key, true, dur, func(v float64) {
		if !d.open || d.dragging {
			return
		}
		d.slideY = int32(float64(start)*(1-v) + 0.5)
		d.applyPanelOffset(d.slideX, d.slideY)
	}, nil)
	return 0
}
