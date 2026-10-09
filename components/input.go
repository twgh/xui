package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

// Input 自绘单行输入框: 载体为原生 Edit, 光标/输入法/选择等行为全部原生,
// 本组件只绘制圆角壳体与焦点框, 并实时同步文字/占位符颜色。
//
// 与 shadcn input.tsx 对齐: h-8, border-input, 聚焦时 border-ring +
// ring-ring/50(文本输入控件点击聚焦也显示, 同浏览器 :focus-visible 行为),
// 禁用为 bg-input/50。
type Input struct {
	*widget.Edit

	placeholder string
	multiline   bool
}

// InputOption 组件配置项。
type InputOption func(*Input)

// WithPlaceholder 设置占位文本。
func WithPlaceholder(s string) InputOption { return func(i *Input) { i.placeholder = s } }

// WithMultiline 启用多行(文本域)模式。
func WithMultiline() InputOption { return func(i *Input) { i.multiline = true } }

// NewInput 创建输入框。cy 传 0 时使用默认高度 32。
func NewInput(x, y, cx, cy int32, hParent int, opts ...InputOption) *Input {
	if cy <= 0 {
		cy = 32
	}
	in := &Input{Edit: widget.NewEdit(x, y, cx, cy, hParent)}
	if in == nil {
		return nil
	}
	for _, o := range opts {
		o(in)
	}

	if in.multiline {
		in.EnableMultiLine(true).EnableAutoWrap(true).EnableAutoShowScrollBar(true)
	}

	// 灭掉原生外观
	in.EnableBkTransparent(true)
	in.EnableDrawBorder(false).EnableDrawFocus(false)
	// 文字内边距, 避免贴到圆角上
	in.SetBorderSize(10, 6, 10, 6)

	// 文字颜色需要实时同步给原生 API, 订阅主题切换
	in.applyTheme(xui.Current())
	xui.OnThemeChangeFor(in.Handle, func(t *xui.Theme) { in.applyTheme(t) })

	in.AddEvent_Paint(in.onPaint)
	xui.AttachWindow(in.Handle)
	in.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	return in
}

func (i *Input) applyTheme(t *xui.Theme) {
	i.SetTextColor(uint32(t.Foreground))
	if i.placeholder != "" {
		i.SetDefaultText(i.placeholder)
		i.SetDefaultTextColor(uint32(t.MutedForeground))
	}
	i.SetCaretColor(uint32(t.Primary))
	i.Redraw(false)
}

// Value 返回输入内容。
func (i *Input) Value() string { return i.GetText() }

// SetValue 设置输入内容。
func (i *Input) SetValue(s string) *Input {
	i.SetText(s)
	return i
}

// OnChange 注册内容变化回调。
func (i *Input) OnChange(fn func(text string)) *Input {
	i.AddEvent_Edit_Changed(func(hEle int, pbHandled *bool) int {
		fn(i.GetText())
		return 0
	})
	return i
}

func (i *Input) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	t := xui.Current()
	st := xui.StateOf(hEle)

	bg := t.Background
	border := t.Input
	if st.Disabled {
		// disabled:bg-input/50
		bg = t.Input.WithAlpha(128)
		border = xui.None
	}

	if st.Focused && !st.Disabled {
		// focus-visible:border-ring + ring-3 ring-ring/50
		p.FocusFrame(t.Ring, bg, t.RadiusLg())
	} else {
		p.FillRound(&p.Rc, bg, t.RadiusLg())
		p.StrokeRound(&p.Rc, border, t.RadiusLg(), 1)
	}
	return 0
}
