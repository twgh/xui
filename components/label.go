package ui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

const (
	// 原生 Button 载体的文字排版区比元素窄一圈; Alert / Dialog / Label.SetAutoWrap
	// 共用同一组量高常量。多行时再补一点高度, 否则最后一行会被裁掉并出现省略号(实测)。
	labelWrapInsetX int32 = 12
	labelWrapPadY   int32 = 4
)

// Label 文本标签: 载体为原生 Button(不响应点击), 绘制时按主题着色。
// 原生 ShapeText 无事件且换肤不便, 用 Button 载体统一管理。
type Label struct {
	*widget.Button
	muted       bool                       // 弱化文字(说明性文本)
	bold        bool                       // 粗体(标题, 字号 +1)
	medium      bool                       // 中等字重(粗体字形但不加字号)
	size        int32                      // 字号覆盖(逻辑像素; 0 = 主题默认)
	colorFn     func(*xui.Theme) xui.Color // 语义字色解析器(实时取令牌; nil = 默认规则)
	textC       uint32                     // 上次下发给原生的字色(避免在 paint 里反复设置)
	autoWrap    bool                       // SetAutoWrap: 高度随折行结果自撑
	wrapW       int32                      // 上次量高时的元素宽
	wrapH       int32                      // 上次写回的折行高度
	wrapPending bool                       // 已排过一轮 AfterFunc 量高
}

// NewLabel 创建文本标签。h 为行高(逻辑像素), 可传 0 使用默认 24。
func NewLabel(x, y, cx, h int32, text string, hParent int) *Label {
	if h <= 0 {
		h = 24
	}
	btn := widget.NewButton(x, y, cx, h, text, hParent)
	if btn == nil {
		return nil
	}
	// 字重用常规: shadcn 的 Label 虽是 font-medium(500), 但中文下 500 由系统回落到
	// 常规字重——用粗体字形会明显更粗(实测对比), 需要更重时用 SetMedium/SetBold
	l := &Label{Button: btn}
	// NoWrap + 省略号: 文字超出宽度时截断显示省略号。
	// 不加 NoWrap 时长文本会换行, 而 Label 只有单行高度, 换出的行被裁剪,
	// 视觉上呈现为笔画撕裂状的"乱码"。
	btn.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter |
		xcc.TextFormatFlag_NoWrap | xcc.TextTrimming_EllipsisCharacter)
	btn.EnableBkTransparent(true)
	// 原生按钮默认四边 border=1, 会把文字区上下各扣 1px; Label 本身没有边框,
	// 这 2px 在 20 高单行标签里会把 g/p 的下伸部裁平(见 DESIGN 第八节 109)。
	// 左右保留 1px 维持原有水平内缩, 只清掉上下。
	btn.SetBorderSize(1, 0, 1, 0)
	// 原生按钮的边框/焦点矩形也要灭掉(载体只借事件与文本绘制能力)
	btn.EnableDrawBorder(false).EnableFocus(false)
	// Label 不是交互控件: 不参与 Tab 焦点遍历(同 shadcn 的 <label>)。
	btn.EnableSwitchFocus(false)
	xui.AttachWindow(btn.Handle)
	l.applyTheme(xui.Current())
	xui.OnThemeChangeFor(btn.Handle, func(t *xui.Theme) { l.applyTheme(t) })
	btn.AddEvent_Paint(l.onPaint)
	return l
}

// SetMuted 切换为弱化文字(说明性文本)。
func (l *Label) SetMuted() *Label {
	l.muted = true
	l.applyTheme(xui.Current())
	return l
}

// SetBold 使用主题字体的粗体(标题用, 字号 +1, 更接近浏览器 font-semibold 观感)。
func (l *Label) SetBold() *Label {
	l.bold = true
	l.applyTheme(xui.Current())
	return l
}

// SetMedium 中等字重(= shadcn 的 font-medium/500): 用粗体字形但不动字号,
// 中文下比 Bold 轻一档, 比常规重一档, 用于 Label 与对话框描述这类文本。
func (l *Label) SetMedium() *Label {
	l.medium = true
	l.applyTheme(xui.Current())
	return l
}

// SetFontSize 覆盖字号(逻辑像素)。用于标题/说明这类与正文尺寸不同的文本
// (shadcn 的 dialog 标题是 text-base = 16, 描述是 text-sm = 14)。
func (l *Label) SetFontSize(px int32) *Label {
	l.size = px
	l.applyTheme(xui.Current())
	return l
}

// SetColorFunc 指定语义字色解析器: 每次同步字色(换主题/绘制)时按当前主题实时求值,
// 所以不要在 fn 里缓存颜色。传 nil 恢复默认规则(常规 = Foreground, SetMuted = MutedForeground)。
// 用于 Alert 这类文字颜色跟变体走、又不是简单的弱化/常规的场景。
func (l *Label) SetColorFunc(fn func(*xui.Theme) xui.Color) *Label {
	l.colorFn = fn
	l.applyTheme(xui.Current())
	return l
}

// WrappedHeight 量测当前文本在当前字体下、按 width 限宽折行后的高度(逻辑像素, 已含行距)。
//
// 换行文本的元素高度**不会**随内容自撑(Layout 的 Auto 只取元素现有高度), 需要多行时
// 用本方法量出高度再 SetHeight + 布局项 Fixed。绘制与量测共用同一字体与限宽, 别自己按行数补行距。
func (l *Label) WrappedHeight(width int32) int32 {
	if l == nil || width <= 0 {
		return 0
	}
	text := l.GetText()
	font := l.GetFont()
	if text == "" || font == 0 {
		return 0
	}
	var sz xc.SIZE
	app.GetTextShowRect(text, -1, font, xcc.TextAlignFlag_Left, width, &sz)
	return sz.CY
}

// SetWrap 允许文字在宽度内自动换行(默认 NoWrap, 超宽截断为省略号)。
// 换行需要调用方给足高度:可以在构造参数里给固定高度, 也可以在宽度确定后用
// WrappedHeight 量出折行高度再 SetHeight + 布局项 Fixed; 需要自动量高时用 SetAutoWrap。
// 高度不足时溢出行被裁剪; 布局里只声明宽度(见 xui/layout.go 的布局约定)。
// 对齐保持左对齐/垂直居中。
func (l *Label) SetWrap() *Label {
	l.SetTextAlign(l.GetTextAlign() &^ xcc.TextFormatFlag_NoWrap)
	return l
}

// SetAutoWrap 开启自动折行, 并让元素高度随折行结果自撑。
//
// 与 SetWrap 的区别: SetWrap 只允许换行, 高度仍由调用方给; SetAutoWrap 会在宽度
// 确定后量 WrappedHeight, 把元素高度与布局项改成实际折行高度, 并逐级通知祖先重排;
// 宽度 / 文本 / 字体变化时会重新量。适合卡片说明、长描述这类宽度随窗口变化的文字。
//
// Label 必须能拿到实际宽度(固定宽, 或 Fill 排在普通列里); 父级是 Weight 列且 Fill
// 子项宽度被解析成 0 时(见 DESIGN 第八节 101), 仍需调用方先把宽度锁成 Fixed。
func (l *Label) SetAutoWrap() *Label {
	if l == nil || l.autoWrap {
		return l
	}
	l.autoWrap = true
	l.SetWrap()
	l.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		l.scheduleAutoWrap()
		return 0
	})
	l.scheduleAutoWrap()
	return l
}

// SetText 更新文本。开启 SetAutoWrap 后会重新量高; 返回值保持原生 Button 的签名,
// 不影响现有链式调用。
func (l *Label) SetText(name string) *widget.Button {
	b := l.Button.SetText(name)
	if l.autoWrap {
		l.wrapW = -1
		l.scheduleAutoWrap()
	}
	return b
}

// scheduleAutoWrap 把量高/重排推迟到下一轮 UI 消息: 尺寸事件里直接改高度并
// AdjustLayout 会重入布局, 走 AfterFunc 合并同轮触发。
func (l *Label) scheduleAutoWrap() {
	if l == nil || !l.autoWrap || l.wrapPending {
		return
	}
	l.wrapPending = true
	app.AfterFunc(0, func() {
		l.wrapPending = false
		l.reflowAutoWrap()
	})
}

// reflowAutoWrap 按当前宽度量出折行高度并写回, 再逐级通知祖先重排。
func (l *Label) reflowAutoWrap() {
	if l == nil || !l.autoWrap || l.Handle == 0 || !xc.XC_IsHXCGUI(l.Handle, xcc.XC_ELE) {
		return
	}
	w := l.GetWidth()
	if w <= 0 || w == l.wrapW {
		return
	}
	l.wrapW = w
	// 单行量测值就是当前字体的行高(14px 文字为 20; 见 Alert/Dialog 的量高口径)。
	lineH := l.WrappedHeight(1 << 30)
	h := labelWrapHeight(l, w, lineH)
	if h <= 0 || h == l.wrapH {
		return
	}
	l.wrapH = h
	l.SetHeight(h)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	reflowAfterTextHeight(l.Handle)
}

// labelWrapHeight 量出 Label 在 width 下折行后的高度(逻辑像素):
// 单行直接用行高, 多行向上取整到整行并补 labelWrapPadY。
func labelWrapHeight(l *Label, width, lineH int32) int32 {
	if l == nil || width <= 0 || lineH <= 0 {
		return 0
	}
	measureW := width - labelWrapInsetX
	if measureW < lineH {
		measureW = width
	}
	return labelRoundWrapHeight(l.WrappedHeight(measureW), lineH)
}

// labelRoundWrapHeight 把 GetTextShowRect 的量测高度取整到整行并补足末行空间。
func labelRoundWrapHeight(h, lineH int32) int32 {
	if h <= 0 || lineH <= 0 {
		return 0
	}
	if h <= lineH {
		return lineH
	}
	return (h+lineH-1)/lineH*lineH + labelWrapPadY
}

// reflowAfterTextHeight 在折行高度变化后从元素自身开始逐级通知祖先重排:
// Auto 高的卡片/行只改子元素高度不会自己长高; 遇到 ScrollArea 的 LayoutFrame
// 交给框架排。
func reflowAfterTextHeight(hEle int) {
	if hEle == 0 || !xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
		return
	}
	xc.XEle_AdjustLayout(hEle)
	for p := xc.XWidget_GetParentEle(hEle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
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

// SetAlignCenter 水平居中对齐(AlertDialog 的 sm 尺寸头部用)。
func (l *Label) SetAlignCenter() *Label {
	f := l.GetTextAlign() &^ (xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Right)
	l.SetTextAlign(f | xcc.TextAlignFlag_Center)
	return l
}

// SetAlignLeft 水平左对齐(默认)。
func (l *Label) SetAlignLeft() *Label {
	f := l.GetTextAlign() &^ (xcc.TextAlignFlag_Center | xcc.TextAlignFlag_Right)
	l.SetTextAlign(f | xcc.TextAlignFlag_Left)
	return l
}

func (l *Label) applyTheme(t *xui.Theme) {
	spec := t.FontDefault
	if l.size > 0 {
		spec.Size = l.size
	}
	if l.bold {
		spec.Bold = true
		spec.Size += 1
	} else if l.medium && !l.muted {
		spec.Bold = true // 同字号下的粗体字形 = 中间字重(500)
	}
	if h := xui.FontHandle(spec); h != 0 {
		l.SetFont(h)
	}
	l.syncTextColor()
	l.Redraw(false)
	if l.autoWrap {
		// 字体/字号可能变了: 宽不变也要重新量行高。
		l.wrapW = -1
		l.scheduleAutoWrap()
	}
}

// syncTextColor 把当前应有的字色下发给原生 Button(字色只能通过 SetTextColor 同步)。
// 禁用态 = disabled:opacity-50 的组级透明, 原生文字不吃 alpha, 与 Input 禁用文字
// 同口径: 当前字色与卡片底色对半混。弱化文本禁用时在 MutedForeground 上混。
// 值没变就不重复调用 —— 它可能在 paint 回调里被调用, 要避免自激重绘。
func (l *Label) syncTextColor() {
	t := xui.Current()
	c := t.Foreground
	if l.colorFn != nil {
		c = l.colorFn(t)
	} else if l.muted {
		c = t.MutedForeground
	}
	if !l.IsEnable() {
		c = c.Mix(t.Card, 0.5)
	}
	if u := uint32(c); u != l.textC {
		l.textC = u
		l.Button.SetTextColor(u)
	}
}

func (l *Label) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	// 字体在 applyTheme 同步; 字色在这里补同步 —— Enable(false) 不触发主题回调,
	// 禁用/恢复后的首帧靠 paint 回调发现状态变化(同 Input.syncTextColor 的口径)。
	l.syncTextColor()
	return 0
}
