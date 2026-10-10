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
	textColor   uint32 // 上次下发给原生 Edit 的字色(避免在 paint 里反复设置)
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
	i.syncTextColor()
	if i.placeholder != "" {
		i.SetDefaultText(i.placeholder)
		i.SetDefaultTextColor(uint32(t.MutedForeground))
	}
	i.SetCaretColor(uint32(t.Primary))
	i.Redraw(false)
}

// syncTextColor 把当前应有的字色下发给原生 Edit(字色只能通过 SetTextColor 同步)。
// 禁用态整体 opacity-50: 文字色 = 前景色与卡片底色对半混(原生文本不吃 alpha, 只能预混)。
// 值没变就不重复调用 —— 它可能在 paint 回调里被调用, 要避免自激重绘。
func (i *Input) syncTextColor() {
	t := xui.Current()
	c := uint32(t.Foreground)
	if !i.IsEnable() {
		c = uint32(t.Foreground.Mix(t.Card, 0.5))
	}
	if c == i.textColor {
		return
	}
	i.textColor = c
	i.SetTextColor(c)
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
	i.syncTextColor() // 禁用态在 paint 里发现时补一次(Enable(false) 不会走主题回调)

	bg := t.Background
	border := t.Input
	if st.Disabled {
		// shadcn: disabled:bg-input/50(暗色另有 dark:disabled:bg-input/80) + 组级 opacity-50;
		// 用 ScaleAlpha 逐层乘, 不要用 WithAlpha 覆盖(暗色 input 自带 15% alpha, 覆盖成 50%
		// 会画出一块发亮的灰饼)
		a := 0.5
		if t.IsDark() {
			a = 0.8
		}
		bg = t.Input.ScaleAlpha(a).ScaleAlpha(0.5)
		border = t.Input.ScaleAlpha(0.5)
	}

	if st.Focused && !st.Disabled {
		// focus-visible:border-ring + ring-3 ring-ring/50: FocusFrame 画 3px 半透明
		// ring 带(填充式, 会盖掉原来的 border-input), 实色的 border-ring 边框
		// 再描在 ring 带内缘 —— 与浏览器一致: 聚焦后边框从 input 色换成 ring 色。
		p.FocusFrame(t.Ring, bg, t.RadiusLg())
		const inset = int32(3)
		inner := xc.RECT{
			Left:   p.Rc.Left + inset,
			Top:    p.Rc.Top + inset,
			Right:  p.Rc.Right - inset,
			Bottom: p.Rc.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.StrokeRound(&inner, t.Ring, t.RadiusLg()-inset, 1)
		}
	} else {
		p.FillRound(&p.Rc, bg, t.RadiusLg())
		p.StrokeRound(&p.Rc, border, t.RadiusLg(), 1)
	}
	return 0
}
