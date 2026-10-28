package ui

import (
	"math"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// SeparatorOrientation 分隔线取向, 对应 shadcn separator.tsx 的 orientation。
type SeparatorOrientation int

const (
	SeparatorHorizontal SeparatorOrientation = iota // 水平: 高 1px, 横向铺开
	SeparatorVertical                               // 垂直: 宽 1px, 纵向铺开
)

// Separator 分隔线: 载体为原生 Element, 画一条 1 逻辑像素的主题色线。
// 与 shadcn separator.tsx 对齐: bg-border; 水平 `h-px w-full`, 垂直 `w-px self-stretch`。
type Separator struct {
	*widget.Element
	vertical bool
}

// SeparatorOption 分隔线配置项。
type SeparatorOption func(*Separator)

// WithSeparatorVertical 改为垂直分隔线。此时 NewSeparator 的 cx 表示**长度(高度)**。
func WithSeparatorVertical() SeparatorOption {
	return func(s *Separator) { s.vertical = true }
}

// WithSeparatorOrientation 显式指定取向。
func WithSeparatorOrientation(o SeparatorOrientation) SeparatorOption {
	return func(s *Separator) { s.vertical = o == SeparatorVertical }
}

// NewSeparator 创建分隔线。cx 是长度(水平 = 宽, 垂直 = 高), 线粗固定 1 逻辑像素。
func NewSeparator(x, y, cx int32, hParent int, opts ...SeparatorOption) *Separator {
	s := &Separator{}
	for _, o := range opts {
		o(s)
	}
	var ele *widget.Element
	if s.vertical {
		ele = widget.NewElement(x, y, 1, cx, hParent)
	} else {
		ele = widget.NewElement(x, y, cx, 1, hParent)
	}
	if ele == nil {
		return nil
	}
	s.Element = ele
	// EnableBkTransparent 只清背景, 元素的原生 1px 边框还在 —— 1px 高的元素上它就是
	// 那条"分隔线"本身, 而框架对它按相位取整, 同一张卡片里的两条线会一条 1 行一条
	// 2 行(实测)。必须关掉, 让下面的自绘全权负责。
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.AddEvent_Paint(s.onPaint)
	xui.AttachWindow(ele.Handle)
	return s
}

func (s *Separator) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	// 直接填充整个元素矩形(元素本身就是 1px 粗的细条): 用画线的话线宽沿路径居中,
	// 贴在元素边缘的那一半会被元素裁剪掉 —— 实测 150% DPI 下只剩 1 个物理像素、
	// 比 Border 令牌的观感淡一半(见 DESIGN 第八节 10)。
	//
	// ⚠️ 但"1 逻辑像素"在 150% DPI 下是 1.5 物理像素, 交给 D2D 会按元素 y 的相位
	// 有时落 1 行、有时落 2 行 —— 同一张卡片里的两条分隔线一粗一细(实测截图:
	// 一条只有 1 行 #E5E5E5, 另一条占 2 行)。这里自己对齐到**整设备像素**:
	// 取元素矩形中心所在的设备行/列, 永远只填 1 个物理像素, 全 DPI 一致。
	sc := p.Scale
	if sc <= 0 {
		sc = 1
	}
	if s.vertical {
		c := math.Floor(float64(p.Rc.Left+p.Rc.Right) / 2 * sc)
		rc := xc.RECTF{
			Left:   float32(c / sc),
			Top:    float32(p.Rc.Top),
			Right:  float32((c + 1) / sc),
			Bottom: float32(p.Rc.Bottom),
		}
		p.Draw.FillRectColorF(&rc, uint32(t.Border))
		return 0
	}
	c := math.Floor(float64(p.Rc.Top+p.Rc.Bottom) / 2 * sc)
	rc := xc.RECTF{
		Left:   float32(p.Rc.Left),
		Top:    float32(c / sc),
		Right:  float32(p.Rc.Right),
		Bottom: float32((c + 1) / sc),
	}
	p.Draw.FillRectColorF(&rc, uint32(t.Border))
	return 0
}

// BadgeVariant 徽章变体, 对应 shadcn badge.tsx 的 variant。
type BadgeVariant int

const (
	BadgeDefault     BadgeVariant = iota // 实心主色 —— shadcn 的 `default`
	BadgeSecondary                       // 弱化填充
	BadgeOutline                         // 边框
	BadgeDestructive                     // 破坏性(软色调)
	BadgeGhost                           // 无底无框, 悬停才出底色
	BadgeLink                            // 主色文字, 悬停加下划线
)

// Badge 几何(逻辑像素), 对齐 badge.tsx 的
// `h-5 gap-1 rounded-4xl border px-2 py-0.5 text-xs [&>svg]:size-3`。
const (
	badgePadX   int32 = 8
	badgeGap    int32 = 4
	badgeIconSz int32 = 12
)

// Badge 自绘徽章: 载体为原生 Element + 手绘文字。
// 与 shadcn badge.tsx 对齐: h-5(20), rounded-4xl(胶囊), text-xs(主题字号 -2);
// destructive 为软色调(bg-destructive/10 text-destructive)。
//
// 内容排布支持行内图标与加载指示器(对应文档里的 data-icon="inline-start/inline-end"
// 与"带加载指示器"示例), 三者(前置图标/文字/后置图标)作为一个整体在徽章里居中:
//
//	ui.NewBadge(0, 0, 0, 0, "Verified", parent, ui.WithBadgeLeadingIcon(icons.IconBadgeCheck))
//	ui.NewBadge(0, 0, 0, 0, "Generating", parent, ui.WithBadgeLoading())
//
// 关于悬停: shadcn 的 badge 悬停样式写作 `[a]:hover:*`(只有包在链接里才生效)。
// Go 侧没有"链接"概念, 这里**恒按悬停生效**, 行为与 Button 一致(见 DESIGN.md 第四节的差异清单)。
type Badge struct {
	*widget.Element
	variant  BadgeVariant
	text     string
	hover    bool
	leading  string // 前置图标(24 网格 path / SVG 原文)
	trailing string // 后置图标
	iconSize int32
	loading  bool      // 前置位画加载指示器(替换前置图标)
	onClick  func()    // 点击回调(对应把徽章渲染成链接/按钮)
	custom   bool      // 使用自定义配色(对应 className 覆盖颜色)
	bg       xui.Color // 自定义底色
	fore     xui.Color // 自定义前景色
	border   xui.Color // 自定义描边
	autoW    bool      // cx 传 0 时按内容量宽, 文字变化后自动重量
}

// NewBadge 创建徽章。cy 可传 0 使用默认高度 20; cx 传 0 时按内容量宽。
func NewBadge(x, y, cx, cy int32, text string, hParent int, opts ...func(*Badge)) *Badge {
	if cy <= 0 {
		cy = 20
	}
	b := &Badge{variant: BadgeDefault, text: text, iconSize: badgeIconSz}
	for _, o := range opts {
		o(b)
	}
	if cx <= 0 {
		cx = b.naturalWidth(xui.Current())
		b.autoW = true
	}
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	b.Element = ele
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(b.onPaint)
	// 悬停: ghost / link / outline 的变体语义要靠它体现
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		b.hover = true
		ele.Redraw(true)
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		b.hover = false
		ele.Redraw(true)
		return 0
	})
	// 点击: 链接形态的徽章(Go 侧没有 <a>, 用回调表达)
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if b.onClick != nil {
			xui.NotePointerPress()
			b.onClick()
		}
		return 0
	})
	xui.AttachWindow(ele.Handle)
	return b
}

// WithBadgeVariant 构造时设置徽章变体。
func WithBadgeVariant(v BadgeVariant) func(*Badge) {
	return func(b *Badge) { b.variant = v }
}

// WithBadgeLeadingIcon 在文字前放一枚图标(size-3 = 12, 对应 data-icon=inline-start)。
func WithBadgeLeadingIcon(path string) func(*Badge) {
	return func(b *Badge) { b.leading = path }
}

// WithBadgeTrailingIcon 在文字后放一枚图标(对应 data-icon=inline-end)。
func WithBadgeTrailingIcon(path string) func(*Badge) {
	return func(b *Badge) { b.trailing = path }
}

// WithBadgeLoading 在文字前放一个持续旋转的加载指示器(替代前置图标)。
// 视觉与文档的 Spinner 示例一致; 只负责视觉, 不改变可点击性。
func WithBadgeLoading() func(*Badge) {
	return func(b *Badge) { b.loading = true }
}

// WithBadgeOnClick 让徽章可点击(对应把 Badge 渲染成链接)。
func WithBadgeOnClick(fn func()) func(*Badge) {
	return func(b *Badge) { b.onClick = fn }
}

// WithBadgeColors 使用自定义配色(对应文档"自定义颜色"的 className 覆盖):
// bg 为底色、fore 为文字/图标色、border 为描边色(传 xui.None 表示无描边)。
func WithBadgeColors(bg, fore, border xui.Color) func(*Badge) {
	return func(b *Badge) {
		b.custom, b.bg, b.fore, b.border = true, bg, fore, border
	}
}

// WithVariant 设置徽章变体。
func (b *Badge) WithVariant(v BadgeVariant) *Badge {
	b.variant = v
	b.Redraw(false)
	return b
}

// SetText 更新徽章文字(cx 传 0 创建的徽章会重新量宽)。
func (b *Badge) SetText(text string) *Badge {
	if b == nil || b.text == text {
		return b
	}
	b.text = text
	if b.autoW {
		w := b.naturalWidth(xui.Current())
		b.SetWidth(w)
		b.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	}
	b.Redraw(false)
	return b
}

// SetLoading 打开/关闭前置加载指示器。
func (b *Badge) SetLoading(v bool) *Badge {
	if b == nil || b.loading == v {
		return b
	}
	b.loading = v
	if !v {
		xui.Spin(b.Handle, false) // 停掉运行时旋转定时器(不等下一帧绘制)
	}
	b.Redraw(false)
	return b
}

// Loading 返回是否处于加载态。
func (b *Badge) Loading() bool { return b != nil && b.loading }

// SetColors 运行时改自定义配色(语义同 WithBadgeColors)。
func (b *Badge) SetColors(bg, fore, border xui.Color) *Badge {
	if b == nil {
		return b
	}
	b.custom, b.bg, b.fore, b.border = true, bg, fore, border
	b.Redraw(false)
	return b
}

// OnClick 注册点击回调(与 WithBadgeOnClick 等价, 便于链式)。
func (b *Badge) OnClick(fn func()) *Badge {
	if b != nil {
		b.onClick = fn
	}
	return b
}

// naturalWidth 按内容量宽: px-2 + 图标 + gap + 文字 + 后置图标。
func (b *Badge) naturalWidth(t *xui.Theme) int32 {
	w := badgePadX*2 + xui.TextWidth(b.text, badgeFont(t))
	if b.leading != "" || b.loading {
		w += b.iconSize + badgeGap
	}
	if b.trailing != "" {
		w += b.iconSize + badgeGap
	}
	return w
}

// badgeFont 徽章字体: text-xs = 主题字号 - 2(与 tooltip / kbd 同一口径)。
func badgeFont(t *xui.Theme) int {
	return xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: t.FontDefault.Size - 2})
}

// resolve 按变体与悬停进度解析底色/边框/文字色, 逐条对应 badge.tsx 的类名。
func (b *Badge) resolve(t *xui.Theme, hv float64) (bg, border, fore xui.Color) {
	if b.custom {
		return b.bg, b.border, b.fore
	}
	var bgHV, foreHV xui.Color
	switch b.variant {
	case BadgeSecondary: // bg-secondary text-secondary-foreground; hover:bg-secondary/80
		bg, fore = t.Secondary, t.SecondaryForeground
		// hover 色预混成不透明(over Card): 不铺方底时, 半透明色会叠在上一帧实色上
		bgHV = t.Card.Mix(t.Secondary, 0.8)
	case BadgeOutline: // border-border text-foreground; hover:bg-muted hover:text-muted-foreground
		border, fore = t.Border, t.Foreground
		bgHV, foreHV = t.Muted, t.MutedForeground
	case BadgeDestructive: // bg-destructive/10 text-destructive; hover:bg-destructive/20
		bg, fore = t.Card.Mix(t.Destructive, 0.1), t.Destructive
		bgHV = t.Card.Mix(t.Destructive, 0.2)
	case BadgeGhost: // hover:bg-muted hover:text-muted-foreground (dark:hover:bg-muted/50)
		fore = t.Foreground
		bgHV, foreHV = t.Muted, t.MutedForeground
		if t.IsDark() {
			bgHV = t.Card.Mix(t.Muted, 0.5)
		}
	case BadgeLink: // text-primary hover:underline(下划线在 onPaint 画)
		fore = t.Primary
	default: // bg-primary text-primary-foreground; hover:bg-primary/80
		bg, fore = t.Primary, t.PrimaryForeground
		bgHV = t.Card.Mix(t.Primary, 0.8)
	}
	bg = xui.Fade(bg, bgHV, hv)
	if !foreHV.IsNone() {
		fore = xui.Fade(fore, foreHV, hv)
	}
	return bg, border, fore
}

func (b *Badge) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	hv := xui.Hover(hEle, b.hover)
	bg, border, fore := b.resolve(t, hv)
	radius := t.Pill(p.H())
	// 需要擦底的变体才铺宿主底色: link / ghost 静止态 bg = None, 透明元素重绘
	// 不擦底(见 DESIGN 第八节 28), 不铺的话上一帧的悬停痕迹会残留。
	// default / secondary 的静止色本身不透明, 整块圆角都会被覆盖 —— 再铺一层
	// Card 方底会在非 Card 背景(如 ItemMuted)上露出一圈深色矩形, 所以跳过。
	switch b.variant {
	case BadgeDefault, BadgeSecondary, BadgeDestructive:
		// 静止/悬停色都是不透明预混色, 整块胶囊每帧被覆盖, 不需要擦底。
		if b.custom {
			p.FillRound(&p.Rc, t.Card, radius)
		}
	default:
		// outline / ghost / link 的静止态没有底色, 用圆角底擦掉上一帧悬停痕迹;
		// 方底会在 muted 底上露出四角, 必须用与胶囊相同的圆角。
		p.FillRound(&p.Rc, t.Card, radius)
	}
	p.FillRound(&p.Rc, bg, radius)
	p.StrokeRound(&p.Rc, border, radius, 1)

	fontSize := t.FontDefault.Size - 2
	f := badgeFont(t)
	if f != 0 {
		p.Draw.SetFont(f)
	}
	// 内容整体居中: [前置图标/加载] 文字 [后置图标]
	tw := xui.TextWidth(b.text, f)
	leadSz := int32(0)
	if b.leading != "" || b.loading {
		leadSz = b.iconSize
	}
	trailSz := int32(0)
	if b.trailing != "" {
		trailSz = b.iconSize
	}
	total := tw
	if leadSz > 0 {
		total += leadSz + badgeGap
	}
	if trailSz > 0 {
		total += trailSz + badgeGap
	}
	x := (p.Rc.Left+p.Rc.Right)/2 - total/2
	cy := (p.Rc.Top + p.Rc.Bottom) / 2
	iconRect := func(x int32, sz int32) xc.RECT {
		return xc.RECT{Left: x, Top: cy - sz/2, Right: x + sz, Bottom: cy + sz/2}
	}
	if leadSz > 0 {
		rc := iconRect(x, leadSz)
		if b.loading {
			// 加载指示器: 运行时按墙钟算相位, 组件只取相位画旋转图标(同 Button)
			p.IconRotated(icons.IconLoaderCircle, &rc, fore, 2, xui.Spin(hEle, true)*2*math.Pi)
		} else {
			p.Icon(b.leading, &rc, fore, 2)
		}
		x += leadSz + badgeGap
	}
	textX := x
	if tw > 0 {
		rc := xc.RECT{Left: textX, Top: p.Rc.Top, Right: textX + tw, Bottom: p.Rc.Bottom}
		p.Text(b.text, &rc, fore, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
		x += tw + badgeGap
	}
	if trailSz > 0 {
		rc := iconRect(x, trailSz)
		p.Icon(b.trailing, &rc, fore, 2)
	}

	// link 变体: hover:underline + underline-offset-4(手绘, 写法同 Button link)。
	// 透明度随过渡值缩放 —— 过渡最后一帧(值恰好到位)不再触发重绘, 若按全色画,
	// 淡出倒数第二帧(v≈0.02)的全色下划线会永久留在画布上(实测残留)。
	if b.variant == BadgeLink && !b.custom && hv > 0 {
		if tw > 0 {
			y := cy + int32(float32(fontSize)*0.3) + 4
			p.Line(textX, y, textX+tw, y, fore.ScaleAlpha(hv), 1)
		}
	}
	return 0
}
