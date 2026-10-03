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
	muted bool // 弱化文字(说明性文本)
	bold  bool // 粗体(标题)
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
	l := &Label{Button: btn}
	// NoWrap + 省略号: 文字超出宽度时截断显示省略号。
	// 不加 NoWrap 时长文本会换行, 而 Label 只有单行高度, 换出的行被裁剪,
	// 视觉上呈现为笔画撕裂状的"乱码"。
	btn.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter |
		xcc.TextFormatFlag_NoWrap | xcc.TextTrimming_EllipsisCharacter)
	btn.EnableBkTransparent(true)
	xui.AttachWindow(btn.Handle)
	l.applyTheme(xui.Current())
	xui.OnThemeChange(func(t *xui.Theme) { l.applyTheme(t) })
	btn.AddEvent_Paint(l.onPaint)
	return l
}

// SetMuted 切换为弱化文字(说明性文本)。
func (l *Label) SetMuted() *Label {
	l.muted = true
	l.applyTheme(xui.Current())
	return l
}

// SetBold 使用主题字体的粗体(标题用)。
func (l *Label) SetBold() *Label {
	l.bold = true
	l.applyTheme(xui.Current())
	return l
}

func (l *Label) applyTheme(t *xui.Theme) {
	spec := t.FontDefault
	if l.bold {
		spec.Bold = true
		spec.Size += 1
	}
	if h := xui.FontHandle(spec); h != 0 {
		l.SetFont(h)
	}
	if l.muted {
		l.Button.SetTextColor(uint32(t.MutedFore))
	} else {
		l.Button.SetTextColor(uint32(t.Foreground))
	}
	l.Redraw(false)
}

func (l *Label) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	// 颜色与字体已在 applyTheme 中同步给原生; 绘制回调中改元素属性会干扰当前帧渲染
	return 0
}
