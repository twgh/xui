package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Label 文本标签: 载体为原生 Button(不响应点击), 绘制时按主题着色。
// 原生 ShapeText 无事件且换肤不便, 用 Button 载体统一管理。
type Label struct {
	*widget.Button
	muted  bool   // 弱化文字(说明性文本)
	bold   bool   // 粗体(标题, 字号 +1)
	medium bool   // 中等字重(粗体字形但不加字号)
	size   int32  // 字号覆盖(逻辑像素; 0 = 主题默认)
	textC  uint32 // 上次下发给原生的字色(避免在 paint 里反复设置)
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

// SetWrap 允许文字在宽度内自动换行(默认 NoWrap, 超宽截断为省略号)。
// 换行需要调用方给足高度:**在构造参数里给**(如 NewLabel(0,0,0,48,…) 的多行说明),
// 高度不足时溢出行被裁剪; 布局里只声明宽度(见 xui/layout.go 的布局约定)。
// 对齐保持左对齐/垂直居中。
func (l *Label) SetWrap() *Label {
	l.SetTextAlign(l.GetTextAlign() &^ xcc.TextFormatFlag_NoWrap)
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
}

// syncTextColor 把当前应有的字色下发给原生 Button(字色只能通过 SetTextColor 同步)。
// 禁用态 = disabled:opacity-50 的组级透明, 原生文字不吃 alpha, 与 Input 禁用文字
// 同口径: 当前字色与卡片底色对半混。弱化文本禁用时在 MutedForeground 上混。
// 值没变就不重复调用 —— 它可能在 paint 回调里被调用, 要避免自激重绘。
func (l *Label) syncTextColor() {
	t := xui.Current()
	c := t.Foreground
	if l.muted {
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
