// Package ui 提供 shadcn 风格的 xcgui 自绘组件。
//
// 这些源码由 xui CLI 拷贝到用户项目, 归用户所有, 可随意修改。
// 组件在绘制回调中实时读取 xui.Current() 主题, 修改主题后调用
// xui.SetTheme 即可全局换肤, 组件自身不缓存任何颜色。
//
// 交互语义与 shadcn-ui 源码(button.tsx 等)逐条对齐:
//   - hover:bg-primary/80 等透明度悬停;
//   - focus-visible 键盘焦点框(鼠标点击不显示, 文本输入类例外);
//   - disabled:opacity-50 整体半透明;
//   - destructive 为软色调(bg-destructive/10 + 红字)。
package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// ButtonVariant 按钮变体, 对应 shadcn button 的 variant。
type ButtonVariant int

const (
	ButtonPrimary     ButtonVariant = iota // 实心主按钮
	ButtonSecondary                        // 弱化填充
	ButtonOutline                          // 边框按钮
	ButtonGhost                            // 幽灵按钮, 悬停才有底色
	ButtonDestructive                      // 破坏性操作(软色调)
)

// ButtonSize 按钮尺寸预设, 对应 shadcn h-7/h-8/h-9。
type ButtonSize int

const (
	ButtonSizeSm      ButtonSize = iota // 高 28
	ButtonSizeDefault                   // 高 32
	ButtonSizeLg                        // 高 36
)

// buttonSpec 尺寸规格, 数值为逻辑像素。
type buttonSpec struct {
	height int32
	font   int32
}

var buttonSpecs = map[ButtonSize]buttonSpec{
	ButtonSizeSm:      {height: 28, font: 13},
	ButtonSizeDefault: {height: 32, font: 14},
	ButtonSizeLg:      {height: 36, font: 14},
}

// Button 自绘按钮: 嵌入原生 widget.Button 获得全部交互行为,
// 隐藏原生外观后在绘制事件中按主题绘制壳体, 文字仍由原生绘制。
type Button struct {
	*widget.Button
	variant ButtonVariant
	size    ButtonSize
}

// ButtonOption 组件配置项, 构造时传入。
type ButtonOption func(*Button)

// WithVariant 设置按钮变体。
func WithVariant(v ButtonVariant) ButtonOption { return func(b *Button) { b.variant = v } }

// WithSize 设置按钮尺寸预设。
func WithSize(s ButtonSize) ButtonOption { return func(b *Button) { b.size = s } }

// NewButton 创建自绘按钮。cy 传 0 时按尺寸预设自适应高度。
func NewButton(x, y, cx, cy int32, text string, hParent int, opts ...ButtonOption) *Button {
	b := &Button{size: ButtonSizeDefault}
	auto := cy <= 0
	if auto {
		cy = buttonSpecs[b.size].height
	}
	b.Button = widget.NewButton(x, y, cx, cy, text, hParent)
	if b.Button == nil {
		return nil
	}
	for _, o := range opts {
		o(b)
	}
	// WithSize 可能改变预设, 自适应模式下以最终预设为准
	if auto {
		b.SetHeight(buttonSpecs[b.size].height)
	}

	// 灭掉原生外观, 交给 onPaint
	b.EnableBkTransparent(true)
	b.SetTextColor(0) // 文字颜色绘制时按主题实时设置
	b.SetTextAlign(xcc.TextAlignFlag_Center | xcc.TextAlignFlag_Vcenter)
	b.AddEvent_Paint(b.onPaint)
	xui.AttachWindow(b.Handle)
	b.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	return b
}

// SetVariant 切换按钮变体。
func (b *Button) SetVariant(v ButtonVariant) *Button {
	b.variant = v
	b.Redraw(false)
	return b
}

// Spec 返回当前尺寸规格。
func (b *Button) Spec() buttonSpec { return buttonSpecs[b.size] }

// resolve 按主题与状态解析壳体颜色, 与 shadcn button.tsx 的类名逐条对应。
func (b *Button) resolve(t *xui.Theme, st xui.State) (bg, border xui.Color) {
	if st.Disabled {
		// disabled:opacity-50 —— 原色整体半透明, 而不是换灰色
		switch b.variant {
		case ButtonPrimary:
			return t.Primary.WithAlpha(128), xui.None
		case ButtonSecondary:
			return t.Secondary.WithAlpha(128), xui.None
		case ButtonDestructive:
			return t.Destructive.WithAlpha(13), xui.None
		default:
			return xui.None, t.Border.WithAlpha(128)
		}
	}
	switch b.variant {
	case ButtonPrimary: // bg-primary hover:bg-primary/80
		bg = t.Primary
		if st.Pressed || st.Hover {
			bg = t.Primary.WithAlpha(204)
		}
	case ButtonSecondary: // bg-secondary hover:bg-mix(fg 5%)
		bg = t.Secondary
		if st.Pressed || st.Hover {
			bg = t.Secondary.Mix(t.Foreground, 0.05)
		}
	case ButtonOutline: // border-border bg-background hover:bg-muted
		bg = t.Background
		border = t.Border
		if st.Hover || st.Pressed {
			bg = t.Muted
		}
	case ButtonGhost: // hover:bg-muted
		border = xui.None
		if st.Hover || st.Pressed {
			bg = t.Muted
		}
	case ButtonDestructive: // bg-destructive/10 text-destructive hover:bg-destructive/20
		bg = t.Destructive.WithAlpha(26)
		if st.Hover || st.Pressed {
			bg = t.Destructive.WithAlpha(51)
		}
	}
	return bg, border
}

// textColor 按主题与状态解析文字颜色。
func (b *Button) textColor(t *xui.Theme, st xui.State) xui.Color {
	if st.Disabled {
		switch b.variant {
		case ButtonPrimary, ButtonSecondary:
			return t.PrimaryFore.WithAlpha(128)
		case ButtonDestructive:
			return t.Destructive.WithAlpha(128)
		default:
			return t.Foreground.WithAlpha(128)
		}
	}
	switch b.variant {
	case ButtonPrimary:
		return t.PrimaryFore
	case ButtonSecondary:
		return t.SecondaryFore
	case ButtonDestructive:
		return t.Destructive
	default: // Outline / Ghost
		return t.Foreground
	}
}

func (b *Button) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	t := xui.Current()
	st := xui.StateOf(hEle)

	// active:translate-y-px —— 按下时整体(含文字)下沉 1 逻辑像素
	if st.Pressed && !st.Disabled {
		p.Rc.Top++
		p.Rc.Bottom++
	}
	*pbHandled = true // 文字自己画, 才能与壳体一起下沉

	bg, border := b.resolve(t, st)
	// focus-visible: 鼠标点击不显示焦点框, Tab 切换才显示
	if st.Focused && !st.Disabled && xui.FocusedByKeyboard() {
		p.FocusFrame(t.Ring, bg, t.RadiusLg())
	} else {
		p.FillRound(&p.Rc, bg, t.RadiusLg())
		p.StrokeRound(&p.Rc, border, t.RadiusLg(), 1)
	}

	// 文字: 手动绘制(字体 = 主题字体 + 尺寸预设), 颜色随状态
	if f := xui.FontHandle(xui.FontSpec{
		Family: t.FontDefault.Family,
		Size:   b.Spec().font,
	}); f != 0 {
		p.Draw.SetFont(f)
	}
	p.TextCenter(b.GetText(), b.textColor(t, st))
	return 0
}
