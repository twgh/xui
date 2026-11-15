package ui

import (
	"math"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 文本域的几何(对应 shadcn textarea.tsx: rounded-lg border border-input px-2.5 py-2
// text-sm, min-h-16 且 field-sizing-content)。
const (
	textareaPadX int32 = 10 // px-2.5
	textareaPadY int32 = 8  // py-2: 与 Input 单行(7/5)、多行预设(6)都不同
	textareaMinH int32 = 64 // min-h-16
)

// Textarea 文本域: shadcn textarea.tsx 的对应实现。
//
// 它是 Input 的多行预设: 壳体/焦点环/校验环/禁用态/占位符全部复用 Input(原生 Edit 载体),
// 差异只有三点 ——
//   - 上下内边距 8(py-2, Input 多行默认 6);
//   - 默认最小高度 64(min-h-16), 创建参数 cy 当作最小高度;
//   - **随内容自动长高**(field-sizing-content): 用原生 Edit 的显示行数(含自动换行)算高,
//     内容永远装得下, 所以不挂滚动条。
//
// 自动长高会把高度写回布局项(Fixed)并逐级重排祖先, 所以要放在 Auto 高的竖向容器里
// (卡片内容区/竖向栈); 行高由原生 Edit 自己折行, 不要再用文字宽度量一遍。
// 需要固定高度时直接用 Input + WithMultiline(高度写在创建参数里, 溢出出滚动条)。
type Textarea struct {
	*Input
	// minH 是创建时给的最小高度(宽/高被拖拽后是新的最小高度), 内容再少也不缩到它以下。
	minH int32
	// rows > 0 表示最小高度按"显示 n 行"实时换算(WithTextareaRows); 用户拖过高度后
	// userMin 置位, 不再按行数覆盖 —— 否则松开鼠标高度又跳回 rows 行。
	rows    int32
	userMin bool
	// resize: 右下角拖拽抓手(WithTextareaResize)。dragging 期间用屏幕坐标增量改 minH。
	resize     bool
	gripHot    bool
	dragging   bool
	dragStartY int32
	dragStartH int32
	dragScale  float64
	// fitPending/fitting: 合并同轮的量高请求, 并挡住"改高度 → Size 事件 → 再量高"的重入。
	fitPending bool
	fitting    bool
}

// NewTextarea 创建文本域。cy 是最小高度(0 = 64, 即 min-h-16), 内容更高时自动长高。
// 其余选项与 Input 相同(WithPlaceholder / WithInputInvalid / WithTipSide 之类不适用);
// WithMultiline 不需要传。
func NewTextarea(x, y, cx, cy int32, hParent int, opts ...InputOption) *Textarea {
	if cy <= 0 {
		cy = textareaMinH
	}
	// 文本域与多行预设的区别在这里定: 内边距 8 必须在 NewInput 的 syncPadding 之前生效。
	opts = append([]InputOption{func(i *Input) {
		i.multiline = true
		i.padY = textareaPadY
	}}, opts...)
	in := NewInput(x, y, cx, cy, hParent, opts...)
	if in == nil {
		return nil
	}
	t := &Textarea{Input: in, minH: cy, rows: in.minRows, resize: in.resize}
	// 拖拽抓手: 事件要抢在 Input 自己的 LButtonDown(附加件命中)之前跑, 因此注册得更晚
	// —— AddEvent 系列按后注册先执行(hits 见 skill 说明), 命中抓手时置 pbHandled 挡掉后续。
	if t.resize {
		in.AddEvent_LButtonDown(t.onResizeDown)
		in.AddEvent_MouseMove(t.onResizeMove)
		in.AddEvent_LButtonUp(t.onResizeUp)
		in.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
			t.setGripHot(false)
			return 0
		})
	}
	// 自动长高后内容总装得下: 关掉原生滚动条。留着的话, 内容在"改高度的那一帧"会先溢出、
	// 滚动条闪现一下, 而且它还占掉一行宽度, 下一帧宽度变了又要重新折行。
	in.EnableAutoShowScrollBar(false)
	if h := in.GetScrollBarHObj(); h != nil {
		h.Show(false)
	}
	if v := in.GetScrollBarVObj(); v != nil {
		v.Show(false)
	}
	// 用户输入、程序化 SetText、宽度变化(窗口缩放/换主题改字体)都会改变行数或折行。
	in.AddEvent_Edit_Changed(func(int, *bool) int {
		t.scheduleFit()
		return 0
	})
	in.AddEvent_Edit_Set(func(int, *bool) int {
		t.scheduleFit()
		return 0
	})
	in.AddEvent_Size(func(int, xcc.AdjustLayout_, uint32, *bool) int {
		t.scheduleFit()
		return 0
	})
	// 换主题会换字体/字号: Input 的 applyTheme 先跑(注册在前), 这里重新钉住文本域的内边距,
	// 再按新行高量一次。
	xui.OnThemeChangeFor(in.Handle, func(*xui.Theme) {
		in.padY = textareaPadY
		in.syncPadding()
		t.scheduleFit()
	})
	t.scheduleFit()
	return t
}

// SetValue 设置文本并重新量高(程序化改文本不保证派发变更事件)。
func (t *Textarea) SetValue(s string) *Textarea {
	if t == nil {
		return t
	}
	t.Input.SetValue(s)
	t.scheduleFit()
	return t
}

// Refresh 重新量高。外部直接改过字体/字号后调用(主题切换已自动处理)。
func (t *Textarea) Refresh() *Textarea {
	if t == nil {
		return t
	}
	t.scheduleFit()
	return t
}

// scheduleFit 把量高推迟到下一轮 UI 消息: 在编辑事件/尺寸事件里直接改高度会重入布局,
// 走 AfterFunc 合并同轮触发(同 Alert 的做法)。
func (t *Textarea) scheduleFit() {
	if t == nil || t.fitPending {
		return
	}
	t.fitPending = true
	app.AfterFunc(0, func() {
		t.fitPending = false
		t.fit()
	})
}

// fit 按原生 Edit 的显示行数把元素高度写回。
//
// 为什么要读行数而不是按文本宽度量高: 自动换行是 Edit 自己折的, 同一段文字在不同宽度下
// 行数不同; GetRowCountEx 给的就是**含折行**的显示行数, 行高则由行坐标差反推
// (GetPoint 的 y = 上内边距 + 行高, 与文字内容/滚动位置无关)。
func (t *Textarea) fit() {
	if t == nil || t.fitting || !xc.XC_IsHXCGUI(t.Handle, xcc.XC_ELE) {
		return
	}
	// 隐藏页/首次布局前宽度是 0, 这时量出来的折行行数没有意义(全挤成一行), 等显示后
	// 的 Size 事件再量。
	if t.GetWidth() <= 0 {
		return
	}
	pt := t.GetPointEx(0, 0)
	lineH := pt.Y - textareaPadY
	if lineH <= 0 {
		return
	}
	rows := t.GetRowCountEx()
	if rows < 1 {
		rows = 1
	}
	// 最小高度: rows 行(WithTextareaRows)按当前行高实时换算, 与创建参数的 cy 二选一;
	// 用户拖过高度后以拖出来的值为准(userMin), 否则松手又会跳回 rows 行。
	minH := t.minH
	if t.rows > 0 && !t.userMin {
		minH = t.rows*lineH + textareaPadY*2
	}
	h := rows*lineH + textareaPadY*2
	if h < minH {
		h = minH
	}
	if h == t.GetHeight() || !xc.XC_IsHXCGUI(t.Handle, xcc.XC_ELE) {
		return
	}
	t.fitting = true
	t.SetHeight(h)
	t.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	t.fitting = false
	textareaReflow(t.Handle)
}

// SetRows 按"显示几行"设最小高度(对应原生 `<textarea rows>`); 传 0 表示回到创建参数给的
// 高度。用户拖过抓手后调用它会重新接管高度。
func (t *Textarea) SetRows(n int32) *Textarea {
	if t == nil {
		return t
	}
	if n < 0 {
		n = 0
	}
	t.rows = n
	t.userMin = false
	t.scheduleFit()
	return t
}

// ===== 右下角拖拽改高度(WithTextareaResize) =====

// textareaGrip 是右下角抓手的命中边长(逻辑像素)。
const textareaGrip int32 = 14

// sizeNWSECursor 是斜向调整光标(抓手上悬停时用)。
var sizeNWSECursorHandle uintptr

func sizeNWSECursor() uintptr {
	if sizeNWSECursorHandle == 0 {
		sizeNWSECursorHandle = wapi.LoadImageW(0, wapi.IDC_SIZENWSE, wapi.IMAGE_CURSOR, 0,
			0, wapi.LR_DEFAULTSIZE|wapi.LR_SHARED)
	}
	return sizeNWSECursorHandle
}

// inGrip 判断指针是否落在右下角抓手上。
func (t *Textarea) inGrip(pPt *xc.POINT) bool {
	if t == nil || !t.resize || pPt == nil {
		return false
	}
	w, h := t.GetWidth(), t.GetHeight()
	return pPt.X >= w-textareaGrip && pPt.Y >= h-textareaGrip
}

// setGripHot 切换抓手的悬停光标。离开时把元素光标还原为 0(框架默认箭头),
// Edit 的文本光标本来就是原生绘制, 不受影响。
func (t *Textarea) setGripHot(v bool) {
	if t == nil || t.gripHot == v {
		return
	}
	t.gripHot = v
	if v {
		t.SetCursor(sizeNWSECursor())
	} else {
		t.SetCursor(0)
	}
}

// onResizeDown 在抓手上按下: 捕获鼠标并记下起点(屏幕坐标 + 当前高)。
func (t *Textarea) onResizeDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if !t.inGrip(pPt) || t.Disabled() {
		return 0
	}
	t.dragging = true
	t.userMin = true
	t.dragStartH = t.GetHeight()
	t.dragScale = 1
	if hw := xc.XWidget_GetHWINDOW(hEle); hw != 0 {
		if dpi := xc.XWnd_GetDPI(hw); dpi > 0 {
			t.dragScale = float64(dpi) / 96
		}
	}
	if t.dragScale <= 0 {
		t.dragScale = 1
	}
	var pt wapi.POINT
	if wapi.GetCursorPos(&pt) {
		t.dragStartY = pt.Y
	}
	xc.XEle_SetCapture(hEle, true) // 拖出元素范围也能继续(同 Slider)
	if pbHandled != nil {
		*pbHandled = true
	}
	return 0
}

// onResizeMove 拖动中改最小高度; 未拖动时只管抓手光标。
//
// 用**屏幕坐标增量**换算, 而不是事件里的元素坐标: 元素本身在变高, 坐标原点跟着动,
// 直接相减会变成"追着自己跑"(同状态栏/卡片的抓手做法)。物理增量按窗口 DPI 折回逻辑像素。
func (t *Textarea) onResizeMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if !t.resize {
		return 0
	}
	if t.dragging {
		// 在窗口外松手收不到 LButtonUp: 用 MK_LBUTTON 收尾(同 Resizable / StatusBar)。
		if nFlags&1 == 0 {
			t.endResizeDrag(hEle)
			return 0
		}
		var pt wapi.POINT
		if !wapi.GetCursorPos(&pt) {
			return 0
		}
		h := t.dragStartH + int32(math.Round(float64(pt.Y-t.dragStartY)/t.dragScale))
		if h < textareaMinH {
			h = textareaMinH
		}
		if h != t.minH {
			t.minH = h // 自动长高仍然生效: 内容比它高时 fit 会继续长
			t.scheduleFit()
		}
		if pbHandled != nil {
			*pbHandled = true
		}
		return 0
	}
	t.setGripHot(t.inGrip(pPt))
	return 0
}

// onResizeUp 结束拖拽。
func (t *Textarea) onResizeUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if t.dragging {
		t.endResizeDrag(hEle)
	}
	t.setGripHot(t.inGrip(pPt))
	return 0
}

func (t *Textarea) endResizeDrag(hEle int) {
	if t == nil || !t.dragging {
		return
	}
	t.dragging = false
	xc.XEle_SetCapture(hEle, false)
	t.scheduleFit()
}

// textareaReflow 只把最近的 ScrollArea 框架(或窗口)重排一次。
//
// 不要在父链上逐级 AdjustLayout: 中间层的 Weight 布局会在量算趟里把比例框(以及它里面的
// 卡片/内容区)临时排成 0 宽 —— 实测文本域长高后卡片宽变成 0, 兄弟位置也没更新。
// 框架级重排会统一更新内容区高度、兄弟位置与滚动范围(同 aspectratio.go 的 adjustAncestors)。
func textareaReflow(hEle int) {
	for p := xc.XWidget_GetParentEle(hEle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(p)
			xc.XEle_Redraw(p, false)
			return
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			// p 是框架的内容区: 直接对内容区 AdjustLayout 会把子项宽度打成 0(实测),
			// 必须交给框架排。
			xc.XEle_AdjustLayout(pp)
			xc.XEle_Redraw(pp, false)
			return
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
			return
		}
	}
}
