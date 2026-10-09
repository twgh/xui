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

// ButtonVariant 按钮变体, 对应 shadcn button.tsx 的 variant。
type ButtonVariant int

const (
	ButtonPrimary     ButtonVariant = iota // 实心主按钮 —— shadcn 的 `default`(用 primary 色)
	ButtonSecondary                        // 弱化填充
	ButtonOutline                          // 边框按钮
	ButtonGhost                            // 幽灵按钮, 悬停才有底色
	ButtonDestructive                      // 破坏性操作(软色调)
	ButtonLink                             // 链接样式: 主色文字, 悬停加下划线
)

// ButtonDefault 是 ButtonPrimary 的别名 —— shadcn 里这个变体就叫 `default`。
// 两个名字等价, 想跟着上游叫法时用它。
const ButtonDefault = ButtonPrimary

// ButtonSize 按钮尺寸预设, 对应 shadcn button.tsx 的 size(换算成逻辑像素):
// default=h-8 / xs=h-6 / sm=h-7 / lg=h-9; icon 系是方形(size-8 / size-6 / size-7 / size-9)。
type ButtonSize int

const (
	ButtonSizeXs      ButtonSize = iota // 高 24
	ButtonSizeSm                        // 高 28
	ButtonSizeDefault                   // 高 32
	ButtonSizeLg                        // 高 36
	ButtonSizeIcon                      // 32×32
	ButtonSizeIconXs                    // 24×24
	ButtonSizeIconSm                    // 28×28
	ButtonSizeIconLg                    // 36×36
)

// buttonSpec 尺寸规格, 数值为逻辑像素。
type buttonSpec struct {
	height    int32
	font      int32
	width     int32 // >0: 方形预设(图标按钮), 调用方 cx 传 <=0 时采用
	radiusCap int32 // >0: rounded-[min(--radius-md, cap)]; 0 = 圆角用 RadiusLg(rounded-lg)
}

var buttonSpecs = map[ButtonSize]buttonSpec{
	ButtonSizeXs:      {height: 24, font: 12, radiusCap: 10}, // rounded-[min(--radius-md,10px)]
	ButtonSizeSm:      {height: 28, font: 13, radiusCap: 12}, // rounded-[min(--radius-md,12px)]
	ButtonSizeDefault: {height: 32, font: 14},
	ButtonSizeLg:      {height: 36, font: 14},
	ButtonSizeIcon:    {height: 32, width: 32},
	ButtonSizeIconXs:  {height: 24, width: 24, radiusCap: 10},
	ButtonSizeIconSm:  {height: 28, width: 28, radiusCap: 12},
	ButtonSizeIconLg:  {height: 36, width: 36},
}

// radius 返回该尺寸的圆角: 预设没写 cap 就是 rounded-lg; 写了则是 min(--radius-md, cap)。
func (s buttonSpec) radius(t *xui.Theme) int32 {
	if s.radiusCap <= 0 {
		return t.RadiusLg()
	}
	if r := t.RadiusMd(); r < s.radiusCap {
		return r
	}
	return s.radiusCap
}

// Button 自绘按钮: 嵌入原生 widget.Button 获得全部交互行为,
// 隐藏原生外观后在绘制事件中按主题绘制壳体, 文字仍由原生绘制。
type Button struct {
	*widget.Button
	variant ButtonVariant
	size    ButtonSize
	icon    string // 非空时绘制图标替代文字
}

// ButtonOption 组件配置项, 构造时传入。
type ButtonOption func(*Button)

// WithVariant 设置按钮变体。
func WithVariant(v ButtonVariant) ButtonOption { return func(b *Button) { b.variant = v } }

// WithSize 设置按钮尺寸预设。
func WithSize(s ButtonSize) ButtonOption { return func(b *Button) { b.size = s } }

// WithIcon 让按钮绘制图标, 替代文字(要通常直接用 github.com/twgh/xui/icons 子包里
// 的常量, 如 icons.IconX / icons.IconCheck —— 常量按需链接, 不会把整张图标表带进二进制)。
// 图标按 size-4(16px)居中绘制, 配合 ButtonSizeSm 即 shadcn 的 icon-sm 图标按钮。
func WithIcon(path string) ButtonOption { return func(b *Button) { b.icon = path } }

// pressShift 是按下位移量(逻辑像素), 对应 shadcn 的 active:translate-y-px。
// 元素的创建高度比视觉高度多这么多, 多出的部分留作位移空间: 元素绘制被裁剪在
// 自身客户区内, 若按下时把底边推出客户区, 越界的 1px 会被切掉(看起来底部被削平),
// 预留后整体位移始终落在客户区内, 与原版"整块下沉"的观感一致。
const pressShift int32 = 1

// NewButton 创建自绘按钮。cy 传 0 时按尺寸预设自适应高度。
func NewButton(x, y, cx, cy int32, text string, hParent int, opts ...ButtonOption) *Button {
	b := &Button{size: ButtonSizeDefault}
	autoW, autoH := cx <= 0, cy <= 0
	sp := buttonSpecs[b.size]
	if autoH {
		cy = sp.height
	}
	if autoW && sp.width > 0 { // 图标类预设自带边长
		cx = sp.width
	}
	cy += pressShift // 底部预留按下位移空间
	b.Button = widget.NewButton(x, y, cx, cy, text, hParent)
	if b.Button == nil {
		return nil
	}
	for _, o := range opts {
		o(b)
	}
	// WithSize 可能改变预设, 自适应模式下以最终预设为准
	if autoH || autoW {
		sp = buttonSpecs[b.size]
		if autoH {
			b.SetHeight(sp.height + pressShift)
		}
		if autoW && sp.width > 0 {
			b.SetWidth(sp.width)
		}
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

// SetIcon 设置/替换按钮图标(path 同 WithIcon)。
// 传空串恢复文字绘制。用于图标按钮的状态切换(如 icons.IconEye / icons.IconEyeOff、
// icons.IconSun / icons.IconMoon)。
func (b *Button) SetIcon(path string) *Button {
	b.icon = path
	b.Redraw(false)
	return b
}

// resolve 按主题与状态解析壳体颜色, 与 shadcn button.tsx 的类名逐条对应。
// hv 是 hover 过渡值(0..1): 静止态与悬停态两套色按它插值(button.tsx 的
// transition-all, 时长/缓动见 xui.Hover)。
func (b *Button) resolve(t *xui.Theme, st xui.State, hv float64) (bg, border xui.Color) {
	if st.Disabled {
		// disabled:opacity-50 —— 原色整体半透明, 而不是换灰色
		switch b.variant {
		case ButtonPrimary:
			return t.Primary.WithAlpha(128), xui.None
		case ButtonSecondary:
			return t.Secondary.WithAlpha(128), xui.None
		case ButtonDestructive:
			return t.Destructive.WithAlpha(13), xui.None
		case ButtonOutline:
			return xui.None, t.Border.WithAlpha(128)
		default: // Ghost / Link: 本来就没有壳
			return xui.None, xui.None
		}
	}
	var bgHV, borderHV xui.Color
	switch b.variant {
	case ButtonPrimary: // bg-primary hover:bg-primary/80
		bg, bgHV = t.Primary, t.Primary.WithAlpha(204)
	case ButtonSecondary: // bg-secondary hover:bg-mix(fg 5%)
		bg, bgHV = t.Secondary, t.Secondary.Mix(t.Foreground, 0.05)
	case ButtonOutline: // border-border bg-background hover:bg-muted
		bg, border = t.Background, t.Border
		bgHV, borderHV = t.Muted, t.Border
	case ButtonGhost: // hover:bg-muted
		bg, border = xui.None, xui.None
		bgHV = t.Muted
	case ButtonDestructive: // bg-destructive/10 text-destructive hover:bg-destructive/20
		bg, bgHV = t.Destructive.WithAlpha(26), t.Destructive.WithAlpha(51)
	case ButtonLink: // 无底色/边框; 悬停只加下划线(在 onPaint 里手绘)
		bg, border = xui.None, xui.None
	}
	return xui.Fade(bg, bgHV, hv), xui.Fade(border, borderHV, hv)
}

// textColor 按主题与状态解析文字颜色。
func (b *Button) textColor(t *xui.Theme, st xui.State) xui.Color {
	var c xui.Color
	switch b.variant {
	case ButtonPrimary:
		c = t.PrimaryForeground
	case ButtonSecondary:
		c = t.SecondaryForeground
	case ButtonDestructive:
		c = t.Destructive
	case ButtonLink:
		c = t.Primary
	default: // Outline / Ghost
		c = t.Foreground
	}
	// disabled:opacity-50 对实心/边框按钮是"组级透明": 文字先以原色合成进组内、组再整体压到
	// 50%, 文字核心像素仍是原色(实测浏览器禁用实心钮的白字核心 = 255, 若给文字加 50% alpha
	// 只有 194, 明显偏暗)。但幽灵/链接没有壳体可淡, 组级透明只能落在文字上, 自己加 50% alpha。
	if st.Disabled && (b.variant == ButtonGhost || b.variant == ButtonLink) {
		c = c.WithAlpha(128)
	}
	return c
}

func (b *Button) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	t := xui.Current()
	st := xui.StateOf(hEle)

	// active:translate-y-px —— 按下时整体(壳体与文字一起)下沉 1 逻辑像素。
	// 正常态壳体画到预留边之上; 按下时整块下移, 底边恰好落在客户区底边(见 pressShift)。
	shell := p.Rc
	shell.Bottom -= pressShift
	textRc := shell
	if st.Pressed && !st.Disabled {
		shell.Top += pressShift
		shell.Bottom += pressShift
		textRc.Top += pressShift
		textRc.Bottom += pressShift
	}
	*pbHandled = true // 文字自己画, 才能与壳体一起下沉

	// hover 过渡: 悬停/按下共用一个色(按下时鼠标必然在按钮上, 观感与 shadcn 一致)
	hv := xui.Hover(hEle, (st.Hover || st.Pressed) && !st.Disabled)
	bg, border := b.resolve(t, st, hv)
	// focus-visible: 鼠标点击不显示焦点框, Tab 切换才显示
	p.Rc = shell
	radius := b.Spec().radius(t)
	if st.Focused && !st.Disabled && xui.FocusedByKeyboard() {
		p.FocusFrame(t.Ring, bg, radius)
	} else {
		p.FillRound(&shell, bg, radius)
		p.StrokeRound(&shell, border, radius, 1)
	}

	// 图标按钮: lucide 图标按 size-4(16px)居中绘制, 不画文字
	if b.icon != "" {
		// 18 而非 16: 我们的描边图标墨迹只覆盖路径范围(无 stroke 外扩), 16px 时
		// 视觉上比浏览器 size-4(16px) 小一圈, 实测取 18 后墨迹与参考一致
		const iconSize int32 = 18
		midX := (textRc.Left + textRc.Right) / 2
		midY := (textRc.Top + textRc.Bottom) / 2
		iconRc := xc.RECT{
			Left:   midX - iconSize/2,
			Top:    midY - iconSize/2,
			Right:  midX + iconSize/2,
			Bottom: midY + iconSize/2,
		}
		p.Icon(b.icon, &iconRc, b.textColor(t, st), 2)
		return 0
	}

	// 文字: 手动绘制(字体 = 主题字体 + 尺寸预设), 颜色随状态。
	// 字重用常规: shadcn 的 font-medium(500) 在中文下由系统回落成常规, 用粗体字形会过粗
	f := xui.FontHandle(xui.FontSpec{
		Family: t.FontDefault.Family,
		Size:   b.Spec().font,
	})
	if f != 0 {
		p.Draw.SetFont(f)
	}
	// DrawText 的 Vcenter 按字体行高(含降部空间)居中, 字形整体偏下(实测 14pt 下
	// 上留白 8.6px / 下留白 3.7px), 文字矩形上移 1px 回补; 上移 2px 会明显偏上。
	textRc.Top--
	textRc.Bottom--
	p.Text(b.GetText(), &textRc, b.textColor(t, st),
		xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)

	// link 变体: hover:underline + underline-offset-4。xcgui 没有下划线标志位, 手工画 1px 线:
	// 宽度取量测的文字宽, y = 基线下方 4px(基线 ≈ 行盒中心 + 0.3em, 见 xui.TextWidth 的坑)。
	if b.variant == ButtonLink && !st.Disabled && (st.Hover || st.Pressed) {
		if w := xui.TextWidth(b.GetText(), f); w > 0 {
			cy := (textRc.Top + textRc.Bottom) / 2
			y := cy + int32(float32(b.Spec().font)*0.3) + 4
			x0 := (textRc.Left+textRc.Right)/2 - w/2
			p.Line(x0, y, x0+w, y, b.textColor(t, st), 1)
		}
	}
	return 0
}
