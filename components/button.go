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
	"math"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
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

// ButtonAlign 是按钮内容(行内图标 + 文字 + 尾随图标)的水平对齐方式,
// 对应 shadcn 示例里给 Button 加的 justify-center / justify-start / justify-between。
// 默认居中; 触发区这类整宽按钮常用 Between(文字靠左、尾随图标贴右)。
type ButtonAlign int

const (
	ButtonAlignCenter  ButtonAlign = iota // justify-center(默认)
	ButtonAlignStart                      // justify-start: 内容从左侧内边距开始
	ButtonAlignBetween                    // justify-between: 前置内容靠左, 尾随图标贴右
)

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
	padX      int32 // 左右内边距(px-2.5 / px-2)
	iconPadX  int32 // 有行内图标那一侧的内边距(has-data-[icon]:pl-2 / pl-1.5)
	gap       int32 // 图标与文字间距(gap-1.5 / gap-1)
}

var buttonSpecs = map[ButtonSize]buttonSpec{
	ButtonSizeXs:      {height: 24, font: 12, radiusCap: 10, padX: 8, iconPadX: 6, gap: 4},  // rounded-[min(--radius-md,10px)]
	ButtonSizeSm:      {height: 28, font: 13, radiusCap: 12, padX: 10, iconPadX: 6, gap: 4}, // rounded-[min(--radius-md,12px)]
	ButtonSizeDefault: {height: 32, font: 14, padX: 10, iconPadX: 8, gap: 6},
	ButtonSizeLg:      {height: 36, font: 14, padX: 10, iconPadX: 8, gap: 6},
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
	variant      ButtonVariant
	size         ButtonSize
	icon         string      // 非空时绘制图标替代文字
	leadingIcon  string      // 文字前的行内图标(data-icon=inline-start)
	trailingIcon string      // 文字后的行内图标(data-icon=inline-end)
	iconSize     int32       // 图标绘制边长(逻辑像素); 0 = 按尺寸预设
	align        ButtonAlign // 内容水平对齐(默认居中)
	invalid      bool        // aria-invalid: 表单校验失败态(红色边框 + 红色 ring)
	rounded      bool        // rounded-full(胶囊)
	expanded     bool        // aria-expanded: 弹层触发区展开态
	hasPopup     bool        // aria-haspopup: 按下不整体下沉(菜单/下拉触发区)
	loading      bool        // 加载态: 行内/纯图标位置绘制持续旋转的 loader
	autoWidth    bool        // cx 传 0 时为 true: 宽度按文字/图标量出
}

// ButtonOption 组件配置项, 构造时传入。
type ButtonOption func(*Button)

// WithVariant 设置按钮变体。
func WithVariant(v ButtonVariant) ButtonOption { return func(b *Button) { b.variant = v } }

// WithSize 设置按钮尺寸预设。
func WithSize(s ButtonSize) ButtonOption { return func(b *Button) { b.size = s } }

// WithButtonInvalid 设置校验失败态(对应 shadcn 的 aria-invalid):
// border-destructive + ring-3 destructive/20(暗色 /40)。与 focus-visible 同时成立时红色优先。
// 名字带 Button 前缀: 组件都在 package ui 里, Input/ComboBox 也有同名语义的选项。
func WithButtonInvalid() ButtonOption { return func(b *Button) { b.invalid = true } }

// WithIcon 让按钮绘制图标, 替代文字(要通常直接用 github.com/twgh/xui/icons 子包里
// 的常量, 如 icons.IconX / icons.IconCheck —— 常量按需链接, 不会把整张图标表带进二进制)。
// 图标默认按 18px 画布居中绘制(见 WithIconSize), 配合 ButtonSizeSm 即 shadcn 的 icon-sm 图标按钮。
func WithIcon(path string) ButtonOption { return func(b *Button) { b.icon = path } }

// WithIconSize 覆盖图标绘制边长(逻辑像素)。
// 默认值: 纯图标按钮 18; 行内图标按尺寸档取 14/16/18/18(svg size-3/3.5/4/4 的视觉补偿)。
// 不同 lucide 图标的墨迹占 24 网格的比例不同(maximize/square 是 3..21, x 是 6..18,
// minus 是 5..19), 同一画布下墨迹宽度能差 1.5 倍 —— 想让一排图标视觉齐平, 按墨迹
// 比例挑尺寸, 例如 minus/x 用 22 才能与 18 画布下的 maximize(墨迹 23px)齐平。
func WithIconSize(size int32) ButtonOption { return func(b *Button) { b.iconSize = size } }

// WithButtonLeadingIcon 在文字前绘制一枚行内图标(对应 shadcn 的 data-icon="inline-start")。
// 与 WithIcon 的"图标替代文字"不同, 这个保留文字; 图标与文字整体居中, 间距按尺寸档
// (4/6)取值, 图标侧内边距收窄(pl-1.5/pl-2)。
func WithButtonLeadingIcon(path string) ButtonOption {
	return func(b *Button) { b.leadingIcon = path }
}

// WithButtonTrailingIcon 在文字后绘制一枚行内图标(data-icon="inline-end"),
// 用于"下一步/展开"这类尾随语义(如 chevron-right / chevron-down)。
func WithButtonTrailingIcon(path string) ButtonOption {
	return func(b *Button) { b.trailingIcon = path }
}

// WithButtonAlign 设置内容水平对齐(默认居中)。整宽触发区用 Between 时,
// 文字靠左、尾随图标贴右, 等价 shadcn 示例里的 justify-between 类。
func WithButtonAlign(a ButtonAlign) ButtonOption { return func(b *Button) { b.align = a } }

// WithButtonRounded 让按钮使用胶囊圆角(rounded-full), 适用于方形图标按钮或短文本。
func WithButtonRounded() ButtonOption { return func(b *Button) { b.rounded = true } }

// WithButtonExpanded 设置触发区的展开态(aria-expanded): outline/ghost 变体换成
// bg-muted, secondary 保持 secondary 底; 打开菜单/下拉时配合 SetExpanded 使用。
func WithButtonExpanded() ButtonOption { return func(b *Button) { b.expanded = true } }

// WithButtonHasPopup 标记按钮是弹层触发区(aria-haspopup): 按下时不再整体下沉
// (shadcn 的 active:not-aria-[haspopup]:translate-y-px), 展开/收起时不"弹跳"。
func WithButtonHasPopup() ButtonOption { return func(b *Button) { b.hasPopup = true } }

// WithButtonLoading 显示加载态: 在行内图标位置(纯图标按钮则替换图标)绘制一个
// 持续旋转的 loader(1s/圈, 同 tailwind 的 animate-spin)。只负责视觉, 是否禁用
// 交互由调用方决定(对应 shadcn 示例里的 disabled + LoaderCircleIcon animate-spin)。
func WithButtonLoading() ButtonOption { return func(b *Button) { b.loading = true } }

// pressShift 是按下位移量(逻辑像素), 对应 shadcn 的 active:translate-y-px。
// 元素的创建高度比视觉高度多这么多, 多出的部分留作位移空间: 元素绘制被裁剪在
// 自身客户区内, 若按下时把底边推出客户区, 越界的 1px 会被切掉(看起来底部被削平),
// 预留后整体位移始终落在客户区内, 与原版"整块下沉"的观感一致。
// 实际渲染在 150% DPI 下会被取整成 1 或 2 物理像素; 文字/图标跟随壳体的实际
// 物理位移(见 onPaint 与 DESIGN 88), 不能让它们各自按逻辑量移动。
const pressShift int32 = 1

// NewButton 创建自绘按钮。cy 传 0 时按尺寸预设自适应高度;
// cx 传 0 时按文字/行内图标的量测宽度自适应(纯图标按钮用方形预设)。
func NewButton(x, y, cx, cy int32, text string, hParent int, opts ...ButtonOption) *Button {
	b := &Button{size: ButtonSizeDefault, autoWidth: cx <= 0}
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
	b.SetUserData(uiRoleButton)
	for _, o := range opts {
		o(b)
	}
	// WithSize 可能改变预设, 自适应模式下以最终预设为准
	if autoH || autoW {
		sp = buttonSpecs[b.size]
		if autoH {
			b.SetHeight(sp.height + pressShift)
		}
		if autoW {
			b.SetWidth(b.naturalWidth(xui.Current()))
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

// SetInvalid 设置/清除校验失败态(见 WithButtonInvalid)。
func (b *Button) SetInvalid(v bool) *Button {
	if b.invalid == v {
		return b
	}
	b.invalid = v
	b.Redraw(false)
	return b
}

// Invalid 返回当前是否处于校验失败态。
func (b *Button) Invalid() bool { return b.invalid }

// Spec 返回当前尺寸规格。
func (b *Button) Spec() buttonSpec { return buttonSpecs[b.size] }

// fontHandle 返回与绘制同字体的句柄(正文常规字重)。
func (b *Button) fontHandle(t *xui.Theme) int {
	return xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: b.Spec().font})
}

// inlineIconSize 返回行内图标的默认绘制边长。lucide 的 svg 是 size-3/3.5/4(12/14/16),
// 本项目的描边几何不含 stroke 外扩, 统一放大 2px 才与浏览器观感一致(同纯图标按钮的 18)。
func (b *Button) inlineIconSize() int32 {
	if b.iconSize > 0 {
		return b.iconSize
	}
	switch b.size {
	case ButtonSizeXs:
		return 14
	case ButtonSizeSm:
		return 16
	default:
		return 18
	}
}

// naturalWidth 量出自适应宽度: 内边距 + 行内图标/文字 + 间距(对应 shadcn 的
// px-2.5 / gap-1.5 与 has-data-[icon]:pl/pr-2)。纯图标按钮取方形预设。
func (b *Button) naturalWidth(t *xui.Theme) int32 {
	sp := b.Spec()
	if b.icon != "" {
		if sp.width > 0 {
			return sp.width
		}
		return sp.height
	}
	text := b.GetText()
	if text == "" && b.leadingIcon == "" && b.trailingIcon == "" {
		return 0
	}
	tw := xui.TextWidth(text, b.fontHandle(t))
	isz := b.inlineIconSize()
	lead, trail := b.leadingIcon, b.trailingIcon
	if b.loading {
		lead = icons.IconLoaderCircle
	}
	content := int32(0)
	add := func(w int32) {
		if content > 0 {
			content += sp.gap
		}
		content += w
	}
	padL, padR := sp.padX, sp.padX
	if lead != "" {
		add(isz)
		padL = sp.iconPadX
	}
	if text != "" {
		add(tw)
	}
	if trail != "" {
		add(isz)
		padR = sp.iconPadX
	}
	return padL + content + padR
}

// FitWidth 按当前文字/图标重新量宽(仅 cx 传 0 创建的按钮)。主题换了字体族后可手动调用。
func (b *Button) FitWidth() *Button {
	if !b.autoWidth {
		return b
	}
	b.SetWidth(b.naturalWidth(xui.Current()))
	b.Redraw(false)
	return b
}

// SetLeadingIcon 设置/替换文字前的行内图标(空串移除)。
func (b *Button) SetLeadingIcon(path string) *Button {
	b.leadingIcon = path
	return b.FitWidth()
}

// SetTrailingIcon 设置/替换文字后的行内图标(空串移除)。
func (b *Button) SetTrailingIcon(path string) *Button {
	b.trailingIcon = path
	return b.FitWidth()
}

// SetRounded 切换胶囊圆角(rounded-full)。
func (b *Button) SetRounded(v bool) *Button {
	b.rounded = v
	b.Redraw(false)
	return b
}

// SetAlign 设置内容水平对齐(见 WithButtonAlign)。
func (b *Button) SetAlign(a ButtonAlign) *Button {
	if b.align == a {
		return b
	}
	b.align = a
	b.Redraw(false)
	return b
}

// SetExpanded 设置/清除触发区展开态(见 WithButtonExpanded)。
func (b *Button) SetExpanded(v bool) *Button {
	if b.expanded == v {
		return b
	}
	b.expanded = v
	b.Redraw(false)
	return b
}

// Expanded 返回当前是否处于展开态。
func (b *Button) Expanded() bool { return b.expanded }

// SetHasPopup 设置是否为弹层触发区(见 WithButtonHasPopup)。
func (b *Button) SetHasPopup(v bool) *Button {
	b.hasPopup = v
	return b
}

// HasPopup 返回是否标记为弹层触发区。
func (b *Button) HasPopup() bool { return b.hasPopup }

// SetLoading 切换加载态(见 WithButtonLoading)。
func (b *Button) SetLoading(v bool) *Button {
	if b.loading == v {
		return b
	}
	b.loading = v
	if !v {
		xui.Spin(b.Handle, false) // 停掉运行时的旋转定时器(不等下一帧绘制)
	}
	if b.autoWidth {
		b.SetWidth(b.naturalWidth(xui.Current()))
	}
	b.Redraw(false)
	return b
}

// Loading 返回是否处于加载态。
func (b *Button) Loading() bool { return b.loading }

// SetIcon 设置/替换按钮图标(path 同 WithIcon)。
// 传空串恢复文字绘制。用于图标按钮的状态切换(如 icons.IconEye / icons.IconEyeOff、
// icons.IconSun / icons.IconMoon)。
func (b *Button) SetIcon(path string) *Button {
	b.icon = path
	if b.autoWidth {
		b.SetWidth(b.naturalWidth(xui.Current()))
	}
	b.Redraw(false)
	return b
}

// SetIconSize 运行时修改图标绘制边长(0 = 按尺寸预设: 纯图标 18, 行内 14/16/18)。
// 换图标时若两个图标的墨迹比例不同(如 maximize ↔ minimize), 可一并改尺寸保持观感一致。
func (b *Button) SetIconSize(size int32) *Button {
	b.iconSize = size
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
			a := byte(13) // bg-destructive/10 × 50%
			if t.IsDark() {
				a = 26 // dark:bg-destructive/20 × 50%
			}
			return t.Destructive.WithAlpha(a), xui.None
		case ButtonOutline:
			if t.IsDark() { // dark:bg-input/30 × 50%, dark:border-input × 50%
				return t.Input.ScaleAlpha(0.15), t.Input.ScaleAlpha(0.5)
			}
			return t.Background.Mix(t.Card, 0.5), t.Border.WithAlpha(128)
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
		if t.IsDark() { // dark:border-input dark:bg-input/30 dark:hover:bg-input/50
			bg, border = t.Input.ScaleAlpha(0.30), t.Input
			bgHV, borderHV = t.Input.ScaleAlpha(0.50), t.Input
		}
	case ButtonGhost: // hover:bg-muted
		bg, border = xui.None, xui.None
		bgHV = t.Muted
		if t.IsDark() { // dark:hover:bg-muted/50
			bgHV = t.Muted.ScaleAlpha(0.5)
		}
	case ButtonDestructive: // bg-destructive/10 text-destructive hover:bg-destructive/20
		bg, bgHV = t.Destructive.WithAlpha(26), t.Destructive.WithAlpha(51)
		if t.IsDark() { // dark:bg-destructive/20 dark:hover:bg-destructive/30
			bg, bgHV = t.Destructive.WithAlpha(51), t.Destructive.WithAlpha(77)
		}
	case ButtonLink: // 无底色/边框; 悬停只加下划线(在 onPaint 里手绘)
		bg, border = xui.None, xui.None
	}
	// aria-expanded: outline/ghost 换 bg-muted, secondary 保持 secondary 底(与悬停不同,
	// secondary 的 hover 是混 5% 前景)。展开时直接按全量终端色取。
	if b.expanded {
		switch b.variant {
		case ButtonOutline, ButtonGhost:
			bgHV, hv = t.Muted, 1
		case ButtonSecondary:
			bgHV, hv = t.Secondary, 1
		}
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
	// disabled:opacity-50 是**整元素**透明: 文字像素 = 50% 文字色 + 50% 宿主底色。
	// 用 Card 预混后绘制, 暗色主题下文字才会跟着变暗。不要直接给文字加 alpha ——
	// 文字叠在已经半透明的壳体上会二次透出按钮底色, 与浏览器"整组再合成"的结果不符。
	if st.Disabled {
		c = c.Mix(t.Card, 0.5)
	}
	return c
}

// corners 解析按钮四角半径:
//   - 组内: shadcn 的 in-data-[slot=button-group]:rounded-lg 把 xs/sm/icon-xs/icon-sm
//     的内缩圆角统一成 rounded-lg, 再按首/中/尾收起相邻角;
//   - rounded-full: 用壳体高度的一半盖过上述规则;
//   - 组外: 按尺寸预设(见 buttonSpec.radius)。
func (b *Button) corners(t *xui.Theme, h int32) xui.Corners {
	gp := buttonGroupPlacementOf(b.Handle)
	radius := b.Spec().radius(t)
	if gp.inGroup {
		radius = t.RadiusLg()
	}
	if b.rounded {
		radius = h / 2
	}
	return gp.corners(radius)
}

// elementDPIScale 返回元素所在窗口的物理/逻辑像素比(取不到按 1)。
func elementDPIScale(hEle int) float64 {
	hwnd := xc.XWidget_GetHWND(hEle)
	if hwnd == 0 {
		return 1
	}
	hw := xc.XC_hWindowFromHWnd(hwnd)
	if hw == 0 {
		return 1
	}
	dpi := xc.XWnd_GetDPI(hw)
	if dpi <= 0 {
		return 1
	}
	return float64(dpi) / 96
}

// shellPressShift 返回按下时壳体在**物理像素**上的实际位移。
//
// 1 逻辑像素在 150% DPI 下是 1.5 物理, 壳体矩形经 ConvRect 换算后会按所在
// 位置的相位取整成 1 或 2 物理像素。文字排版和图标盒各自取整的相位不同, 若让
// 它们各自按逻辑坐标平移, 就会出现"文字/图标/壳体没一起下沉"(见 DESIGN 88)。
// 这里直接量壳体的实际物理位移, 内容统一跟随这个整数, 不再猜 round(1×scale)。
func shellPressShift(p *xui.Painter, base, shifted xc.RECT) int32 {
	if p == nil || p.Draw == nil || p.Draw.Handle <= 0 {
		return pressShift
	}
	a, b := base, shifted
	xc.XDraw_ConvRect(p.Draw.Handle, &a)
	xc.XDraw_ConvRect(p.Draw.Handle, &b)
	if d := b.Top - a.Top; d > 0 {
		return d
	}
	return pressShift
}

func (b *Button) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	t := xui.Current()
	st := xui.StateOf(hEle)

	// active:translate-y-px —— 按下时整体下沉。壳体的实际物理位移可能被取整成
	// 1 或 2 物理像素(150% DPI), 所以内容不再各自按逻辑坐标移动, 而是跟随壳体的
	// **实际物理位移**; 否则文字排版和图标盒的取整相位不同, 会显得没有一起下沉(见 DESIGN 第八节 88)。
	// 弹层触发区(aria-haspopup)例外: 菜单弹出时按钮不跟着弹跳。
	// 正常态壳体画到预留边之上; 按下时整块下移, 底边恰好落在客户区底边(见 pressShift)。
	baseShell := p.Rc
	baseShell.Bottom -= pressShift
	shell := baseShell
	pressed := st.Pressed && !st.Disabled && !b.hasPopup
	contentDy := int32(0) // 按下时内容的物理像素位移(与壳体一致)
	if pressed {
		shell.Top += pressShift
		shell.Bottom += pressShift
		contentDy = shellPressShift(p, baseShell, shell)
	}
	textRc := baseShell // 文字/图标按未按下位置排版, 再统一加 contentDy
	scale := 1.0
	if pressed {
		scale = elementDPIScale(hEle)
	}
	textDy := float32(0)
	if contentDy != 0 && scale > 0 {
		textDy = float32(contentDy) / float32(scale)
	}
	*pbHandled = true // 文字自己画, 才能与壳体一起下沉

	// hover 过渡: 悬停/按下共用一个色(按下时鼠标必然在按钮上, 观感与 shadcn 一致)
	hv := xui.Hover(hEle, (st.Hover || st.Pressed) && !st.Disabled)
	bg, border := b.resolve(t, st, hv)
	p.Rc = shell
	corners := b.corners(t, shell.Bottom-shell.Top)
	// focus-visible: 鼠标点击不显示焦点框, Tab 切换才显示。
	// shadcn 语义 = 边框换色 + 3px ring 光环(focus-visible:border-ring +
	// ring-3 ring-ring/50); destructive 变体覆写为 border-destructive/40 +
	// ring-destructive/20(见 button.tsx 第 7/19 行)。
	// 光环画在元素矩形内(xcgui 裁剪到自身客户区, 画不出边界, 口径同 FocusFrame):
	// 整块 halo 带 → 带内先铺宿主底色 t.Card 再叠 bg → 内缘描 1px 实色边框。
	// 铺底色不能省: bg 可能是半透明色(destructive = destructive/10), 直接叠在
	// halo 上会变成 10%+20% ≈ 30% 红, 内芯比光环还深; 浏览器里 bg 与 ring 是
	// 各自独立合成在页面底色上的, 内芯(10%)要比光环(20%)浅。幽灵/链接没有壳体
	// (bg = None), 铺 Card 后即"透明底透出面板色"(见 DESIGN 第八节 61);
	// 若什么都不铺, 整个按钮会被 halo 色填满(实测踩坑)。
	// aria-invalid(表单校验失败)与 focus-visible 共用同一套"环带 + 内芯"画法:
	// 红环 **常驻**(不要求键盘焦点), 与 focus-visible 同时成立时红色优先 ——
	// shadcn 里 aria-invalid:* 变体排在 focus-visible 之后, 后写的 ring 色胜出。
	invalid := b.invalid && !st.Disabled
	if invalid || st.Focused && !st.Disabled && xui.FocusedByKeyboard() {
		halo, borderColor := t.Ring.WithAlpha(128), t.Ring
		switch {
		case invalid: // aria-invalid:border-destructive ring-destructive/20 (dark /40, border /50)
			halo, borderColor = t.Destructive.WithAlpha(51), t.Destructive
			if t.IsDark() {
				halo, borderColor = t.Destructive.WithAlpha(102), t.Destructive.WithAlpha(128)
			}
		case b.variant == ButtonDestructive:
			halo, borderColor = t.Destructive.WithAlpha(51), t.Destructive.WithAlpha(102)
			if t.IsDark() { // dark:ring-destructive/40
				halo = t.Destructive.WithAlpha(102)
			}
		}
		p.FillRoundCorners(&shell, halo, corners)
		const inset = int32(3)
		inner := xc.RECT{
			Left:   shell.Left + inset,
			Top:    shell.Top + inset,
			Right:  shell.Right - inset,
			Bottom: shell.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			ic := corners.Inset(inset)
			p.FillRoundCorners(&inner, t.Card, ic)
			if !bg.IsNone() {
				p.FillRoundCorners(&inner, bg, ic)
			}
			p.StrokeRoundCorners(&inner, borderColor, ic, 1)
		}
	} else {
		p.FillRoundCorners(&shell, bg, corners)
		p.StrokeRoundCorners(&shell, border, corners, 1)
	}

	fore := b.textColor(t, st)
	// 加载态相位: 运行时按墙钟算并驱动重绘, 组件只取相位画旋转图标。
	// 只在加载中调用, 普通按钮的绘制路径不碰动画状态的锁。
	phase := 0.0
	if b.loading {
		phase = xui.Spin(hEle, true)
	}
	// 纯图标按钮: 方形预设, 图标按中心绘制, 不画文字。
	// 默认 18 而非 16: 我们的描边图标墨迹只覆盖路径范围(无 stroke 外扩), 16px 时
	// 视觉上比浏览器 size-4(16px) 小一圈, 实测取 18 后墨迹与参考一致。
	// WithIconSize 可按图标墨迹比例覆盖(如 maximize 的 3..21 比 x 的 6..18 大 50%)。
	if b.icon != "" {
		iconSize := b.iconSize
		if iconSize <= 0 {
			iconSize = 18
		}
		midX := (textRc.Left + textRc.Right) / 2
		midY := (textRc.Top + textRc.Bottom) / 2
		iconRc := xc.RECT{
			Left:   midX - iconSize/2,
			Top:    midY - iconSize/2,
			Right:  midX + iconSize/2,
			Bottom: midY + iconSize/2,
		}
		path, angle := b.icon, 0.0
		if b.loading {
			path, angle = icons.IconLoaderCircle, phase*2*math.Pi
		}
		p.IconRotatedOffset(path, &iconRc, fore, 2, angle, 0, contentDy)
		return 0
	}

	// 文字: 手动绘制(字体 = 主题字体 + 尺寸预设), 颜色随状态。
	// 字重用常规: shadcn 的 font-medium(500) 在中文下由系统回落成常规, 用粗体字形会过粗
	f := b.fontHandle(t)
	if f != 0 {
		p.Draw.SetFont(f)
	}
	text := b.GetText()

	// 加载态占据行内前置图标的位置(有文字时是 spinner+文字, 无文字时是居中 spinner)。
	lead, trail := b.leadingIcon, b.trailingIcon
	if b.loading {
		lead = icons.IconLoaderCircle
	}

	// 无行内图标: 文字直接在整块壳体里居中(与旧行为一致)。
	if lead == "" && trail == "" {
		// DrawText 的 Vcenter 按字体行高(含降部空间)居中, 字形整体偏下(实测 14pt 下
		// 上留白 8.6px / 下留白 3.7px), 文字矩形上移 1px 回补; 上移 2px 会明显偏上。
		// TextF 用浮点矩形承载按下时的物理像素位移(见 DESIGN 第八节 88)。
		left, align := textRc.Left, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter
		if b.align != ButtonAlignCenter {
			// justify-start / justify-between 只有一个子项时, 文字都靠左(同 flex 语义)
			left += b.Spec().padX
			align = xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter
		}
		rcF := xc.RECTF{
			Left:   float32(left),
			Top:    float32(textRc.Top-1) + textDy,
			Right:  float32(textRc.Right),
			Bottom: float32(textRc.Bottom-1) + textDy,
		}
		p.TextF(text, &rcF, fore, align)
		// link 变体: hover:underline + underline-offset-4。xcgui 没有下划线标志位, 手工画 1px 线:
		// 宽度取量测的文字宽, y = 基线下方 4px(基线 ≈ 行盒中心 + 0.3em, 见 xui.TextWidth 的坑)。
		if b.variant == ButtonLink && !st.Disabled && (st.Hover || st.Pressed) {
			if w := xui.TextWidth(text, f); w > 0 {
				cy := float64(textRc.Top+textRc.Bottom)/2 - 1 + float64(textDy)
				y := int32(math.Round(cy + float64(b.Spec().font)*0.3 + 4))
				x0 := left
				if b.align == ButtonAlignCenter {
					x0 = (textRc.Left+textRc.Right)/2 - w/2
				}
				p.Line(x0, y, x0+w, y, fore, 1)
			}
		}
		return 0
	}

	// 行内图标 + 文字: 图标与文字作为一整块居中, 间距 gap, 对齐 shadcn 的 flex gap。
	isz := b.inlineIconSize()
	gap := b.Spec().gap
	sp := b.Spec()
	tw := xui.TextWidth(text, f)
	content := int32(0)
	if lead != "" {
		content += isz
	}
	if text != "" {
		if content > 0 {
			content += gap
		}
		content += tw
	}
	if trail != "" {
		if content > 0 {
			content += gap
		}
		content += isz
	}
	x := textRc.Left + sp.padX
	if lead != "" {
		x = textRc.Left + sp.iconPadX
	}
	if b.align == ButtonAlignCenter {
		x = (textRc.Left+textRc.Right)/2 - content/2
	}
	cy := (textRc.Top + textRc.Bottom) / 2
	drawIcon := func(path string, left int32, angle float64) {
		rc := xc.RECT{
			Left:   left,
			Top:    cy - isz/2,
			Right:  left + isz,
			Bottom: cy + isz/2,
		}
		p.IconRotatedOffset(path, &rc, fore, 2, angle, 0, contentDy)
	}
	leadAngle := 0.0
	if b.loading {
		leadAngle = phase * 2 * math.Pi
	}
	if lead != "" {
		drawIcon(lead, x, leadAngle)
		x += isz
		if text != "" || trail != "" {
			x += gap
		}
	}
	textX := int32(0)
	if text != "" {
		textX = x
		rcF := xc.RECTF{
			Left:   float32(x),
			Top:    float32(textRc.Top-1) + textDy,
			Right:  float32(x + tw + 2),
			Bottom: float32(textRc.Bottom-1) + textDy,
		}
		p.TextF(text, &rcF, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
		x += tw
		if trail != "" {
			x += gap
		}
	}
	if trail != "" {
		tx := x
		// justify-between: 有前置内容时尾随图标贴住右侧内边距(同 flex 的 space-between)
		if b.align == ButtonAlignBetween && (lead != "" || text != "") {
			tx = textRc.Right - sp.iconPadX - isz
		}
		drawIcon(trail, tx, 0)
	}
	if b.variant == ButtonLink && text != "" && !st.Disabled && (st.Hover || st.Pressed) && tw > 0 {
		y := int32(math.Round(float64(cy) - 1 + float64(textDy) + float64(b.Spec().font)*0.3 + 4))
		p.Line(textX, y, textX+tw, y, fore, 1)
	}
	return 0
}
