package ui

import (
	"image"
	"math"
	"os"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// AvatarSize 头像尺寸档, 对应 shadcn avatar.tsx 的 size(sm size-6 / default size-8 / lg size-10)。
type AvatarSize int

const (
	AvatarSizeDefault AvatarSize = iota // 32
	AvatarSizeSm                        // 24
	AvatarSizeLg                        // 40
)

func (s AvatarSize) px() int32 {
	switch s {
	case AvatarSizeSm:
		return 24
	case AvatarSizeLg:
		return 40
	default:
		return 32
	}
}

// badgeD 返回该尺寸档的徽标直径(上游 sm size-2 / default size-2.5 / lg size-3)。
func (s AvatarSize) badgeD() int32 {
	switch s {
	case AvatarSizeSm:
		return 8
	case AvatarSizeLg:
		return 12
	default:
		return 10
	}
}

// avatarGroupOverlap 是 AvatarGroup 的负间距(-space-x-2 = -8)。
const avatarGroupOverlap int32 = 8

// AvatarOption 头像配置项。
type AvatarOption func(*Avatar)

// WithAvatarSize 设置尺寸档(默认 32)。
func WithAvatarSize(s AvatarSize) AvatarOption { return func(a *Avatar) { a.size = s } }

// WithAvatarImage 从图片文件加载头像(支持 png/jpeg/gif), 失败时保留 fallback 文字。
func WithAvatarImage(path string) AvatarOption {
	return func(a *Avatar) { a.imagePath = path }
}

// WithAvatarImageData 直接使用已解码的图片作为头像(自动居中裁成圆形)。
// 需要在创建时传入图片时用它; 之后换图用 SetImage / SetImageData。
func WithAvatarImageData(img image.Image) AvatarOption {
	return func(a *Avatar) { a.imageData = img }
}

// WithAvatarRing 在头像外圈画 2px 背景色环(头像组里用它把相邻头像隔开)。
func WithAvatarRing() AvatarOption { return func(a *Avatar) { a.ring = true } }

// WithAvatarInteractive 让头像可交互(可聚焦 + 可 Tab 到, 键盘焦点画内侧 2px 环),
// 用作 Popover / Menu 的触发区(对应上游把 Avatar 放进 DropdownMenuTrigger 的示例)。
// 头像元素尺寸就是直径, 环只能画在圆内侧, 所以是"内描边"而不是外扩环带。
func WithAvatarInteractive() AvatarOption { return func(a *Avatar) { a.interactive = true } }

// WithAvatarBadge 在右下角显示状态徽标; icon 可为空(纯色圆点)或 lucide path。
func WithAvatarBadge(icon string) AvatarOption {
	return func(a *Avatar) { a.badge = true; a.badgeIcon = icon }
}

// WithAvatarBadgeColor 指定徽标底色(默认主色)。对应上游示例里直接写
// `className="bg-green-600 dark:bg-green-800"` 的用法; 图标色会自动按对比度挑。
func WithAvatarBadgeColor(c xui.Color) AvatarOption {
	return func(a *Avatar) { a.badgeColor = func(*xui.Theme) xui.Color { return c } }
}

// WithAvatarBadgeColorFunc 用主题函数给徽标上色(切主题时实时重算, 同其他组件的
// `SetColorFunc` 形态)。
func WithAvatarBadgeColorFunc(fn func(*xui.Theme) xui.Color) AvatarOption {
	return func(a *Avatar) { a.badgeColor = fn }
}

// WithAvatarBadgeIconColor 指定徽标图标色(默认按徽标底色的对比度自动挑)。
func WithAvatarBadgeIconColor(c xui.Color) AvatarOption {
	return func(a *Avatar) { a.badgeIconColor = func(*xui.Theme) xui.Color { return c } }
}

// WithAvatarBadgeSize 覆盖徽标直径(默认跟随尺寸档: sm 8 / default 10 / lg 12)。
// 头像直径被 cx/cy 放大时, 徽标仍按"档位比例 × 实际半径"等比缩放。
func WithAvatarBadgeSize(d int32) AvatarOption {
	return func(a *Avatar) {
		if d > 0 {
			a.badgeSize = d
		}
	}
}

// Avatar 圆形头像, 对应 shadcn avatar.tsx:
//   - 尺寸 sm 24 / default 32 / lg 40, rounded-full;
//   - 有图时居中裁剪成圆形; 无图/加载失败时显示 fallback 文字(bg-muted + muted-foreground);
//   - 右下角可加状态徽标(bg-primary + 2px 背景色环);
//   - 纯展示, 不接收焦点与鼠标。
//
// 图片用 Go 标准库解码后在 CPU 侧做居中裁剪 + 抗锯齿圆形蒙版, 再交给 xcgui 绘制;
// 圆形蒙版按 4 倍逻辑尺寸生成, 高 DPI 下缩放依然平滑。
type Avatar struct {
	*widget.Element
	size        AvatarSize
	fallback    string
	ring        bool
	badge       bool
	badgeIcon   string
	interactive bool

	badgeColor     func(*xui.Theme) xui.Color // 徽标底色(nil = Primary)
	badgeIconColor func(*xui.Theme) xui.Color // 徽标图标色(nil = 按对比度挑)
	badgeSize      int32                      // 徽标直径覆盖(0 = 跟随尺寸档)

	icon string // 有图/无图时的居中图标(替代 fallback 文字, 供计数头像放 Plus 用)

	imagePath string
	imageData image.Image
	imgHandle int
}

// NewAvatar 创建头像。cx 传 0 用尺寸档; 传其他值作为直径覆盖尺寸档。
func NewAvatar(x, y, cx, cy int32, hParent int, fallback string, opts ...AvatarOption) *Avatar {
	a := &Avatar{fallback: fallback}
	for _, o := range opts {
		o(a)
	}
	size := a.size.px()
	if cx > 0 {
		size = cx
	}
	if cy > 0 {
		size = cy
	}
	ele := widget.NewElement(x, y, size, size, hParent)
	if ele == nil {
		return nil
	}
	a.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	// 默认纯展示: 不接焦点也不进 Tab; WithAvatarInteractive 时才打开(当触发区用)
	ele.EnableFocus(a.interactive)
	ele.EnableSwitchFocus(a.interactive)
	ele.AddEvent_Paint(a.onPaint)
	ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		a.releaseImage()
		return 0
	})
	a.syncImage()
	xui.AttachWindow(ele.Handle)
	return a
}

// SetImage 换一张头像图片; 传空串 = 清空图片回到 fallback。
func (a *Avatar) SetImage(path string) *Avatar {
	if a == nil {
		return a
	}
	a.imagePath, a.imageData = path, nil
	a.syncImage()
	a.Redraw(false)
	return a
}

// SetImageData 用已解码图片换头像(center-crop 圆形); nil = 清空图片。
func (a *Avatar) SetImageData(img image.Image) *Avatar {
	if a == nil {
		return a
	}
	a.imagePath, a.imageData = "", img
	a.syncImage()
	a.Redraw(false)
	return a
}

// SetFallback 设置无图时显示的文字(通常是姓名缩写)。
func (a *Avatar) SetFallback(s string) *Avatar {
	if a == nil {
		return a
	}
	a.fallback = s
	a.Redraw(false)
	return a
}

// SetIcon 设置无图时居中显示的图标(lucide path / 整段 SVG), 传空串回到 fallback 文字。
// 对应上游"头像里放图标"的写法(如 AvatarGroupCount 里的 Plus)。
func (a *Avatar) SetIcon(path string) *Avatar {
	if a == nil {
		return a
	}
	a.icon = path
	a.Redraw(false)
	return a
}

// SetBadge 开关右下角徽标; icon 为空时画纯色圆点。
func (a *Avatar) SetBadge(on bool, icon string) *Avatar {
	if a == nil {
		return a
	}
	a.badge, a.badgeIcon = on, icon
	a.Redraw(false)
	return a
}

// SetBadgeColor 运行时改徽标底色; 传 nil 恢复主题主色。
func (a *Avatar) SetBadgeColor(c func(*xui.Theme) xui.Color) *Avatar {
	if a == nil {
		return a
	}
	a.badgeColor = c
	a.Redraw(false)
	return a
}

// SetBadgeIconColor 运行时改徽标图标色; 传 nil 恢复"按底色对比度自动挑"。
func (a *Avatar) SetBadgeIconColor(c func(*xui.Theme) xui.Color) *Avatar {
	if a == nil {
		return a
	}
	a.badgeIconColor = c
	a.Redraw(false)
	return a
}

// SetBadgeSize 运行时改徽标直径(<=0 恢复跟随尺寸档)。
func (a *Avatar) SetBadgeSize(d int32) *Avatar {
	if a == nil {
		return a
	}
	if d < 0 {
		d = 0
	}
	a.badgeSize = d
	a.Redraw(true)
	return a
}

// SetSize 运行时切尺寸档(sm / default / lg): 重设元素边长, 并按新直径重建圆形图片
// (蒙版是 CPU 侧按直径生成的, 不重建的话放大后会糊)。与构造期的 WithAvatarSize 等价。
func (a *Avatar) SetSize(s AvatarSize) *Avatar {
	if a == nil || a.size == s {
		return a
	}
	d := s.px()
	var old xc.RECT
	xc.XEle_GetWndClientRect(a.Handle, &old)
	a.size = s
	// 元素矩形与布局项一起改: 只改矩形的话父级下一次布局会按旧的布局项把尺寸压回去
	a.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, d)
	a.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, d)
	a.SetWidth(d)
	a.SetHeight(d)
	// 圆形图片按元素宽度生成, 直径变了必须重建(否则缩放糊边)
	a.releaseImage()
	a.syncImage()
	reflowAfterResize(a.Handle, old)
	a.Redraw(true)
	return a
}

// SetInteractive 运行时开关"可交互"(可聚焦 / 进 Tab / 画焦点环), 用作弹层触发区时打开。
func (a *Avatar) SetInteractive(on bool) *Avatar {
	if a == nil || a.interactive == on {
		return a
	}
	a.interactive = on
	xc.XEle_EnableFocus(a.Handle, on)
	xc.XEle_EnableSwitchFocus(a.Handle, on)
	// 可交互头像要给焦点环让出 2px, 圆半径变了 → 位图重建
	a.syncImage()
	a.Redraw(true)
	return a
}

// SetRing 开/关头像外圈的 2px 背景色环(叠放组用它把相邻头像隔开)。
func (a *Avatar) SetRing(on bool) *Avatar {
	if a == nil || a.ring == on {
		return a
	}
	a.ring = on
	// 环带宽度变了 → 圆半径变, 位图要重建(否则圆边会被环带压住一截)
	a.syncImage()
	a.Redraw(true)
	return a
}

// syncImage 重新生成圆形图片句柄(失败时回落 fallback)。
func (a *Avatar) syncImage() {
	a.releaseImage()
	var img image.Image
	if a.imageData != nil {
		img = a.imageData
	} else if a.imagePath != "" {
		f, err := os.Open(a.imagePath)
		if err == nil {
			img, _, err = image.Decode(f)
			f.Close()
			if err != nil {
				img = nil
			}
		}
	}
	if img == nil {
		return
	}
	a.imgHandle = avatarImageHandle(img, a.GetWidth(), avatarDPI(a.Handle), a.circleInset())
}

// circleInset 返回照片位图要让出的环带宽度(逻辑像素): 画色环或需要画焦点环时为 2。
//
// 位图按**整块元素**的物理尺寸生成, 圆半径再扣掉这条环带 —— 于是"位图尺寸 == 元素尺寸",
// 绘制时按元素矩形 1:1 铺满, 不存在框架二次缩放/裁切, 圆边永远是圆的。
func (a *Avatar) circleInset() int32 {
	if a == nil {
		return 0
	}
	if a.ring || a.interactive {
		return 2
	}
	return 0
}

func (a *Avatar) releaseImage() {
	if a.imgHandle != 0 {
		xc.XImage_Destroy(a.imgHandle)
		a.imgHandle = 0
	}
}

func (a *Avatar) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()

	rc := p.Rc
	elemW := rc.Right - rc.Left
	// 环带(色环 / 键盘焦点环)统一画在**整块元素**的圆里, 照片的位图自带 inset 透明边,
	// 让出这圈宽度。不要把照片画进内缩后的矩形: 位图的像素尺寸就是元素尺寸, 传一个更小的
	// 目标矩形时框架不保证按比例缩放, 实测照片会溢出到元素右下角、圆边被切成直角。
	focused := a.interactive && xui.StateOf(hEle).Focused
	if band := a.circleInset(); band > 0 && (a.ring || focused) {
		c := t.Background
		if focused {
			c = t.Ring
		}
		p.Draw.SetBrushColor(uint32(c))
		p.Draw.FillEllipse(&rc)
	}

	if a.imgHandle != 0 {
		p.Draw.XDraw_ImageEx(a.imgHandle, rc.Left, rc.Top, rc.Right-rc.Left, rc.Bottom-rc.Top)
	} else {
		if band := a.circleInset(); band > 0 {
			if a.ring || focused {
				rc = insetRect(rc, band)
			}
		}
		p.Draw.SetBrushColor(uint32(t.Muted))
		p.Draw.FillEllipse(&rc)
		if a.icon != "" {
			// 图标版(上游 AvatarGroupCount 里放 Plus 之类): 占直径的 55%, 居中
			d := (rc.Right - rc.Left) * 55 / 100
			if d < 8 {
				d = 8
			}
			ir := xc.RECT{
				Left:   rc.Left + (rc.Right-rc.Left-d)/2,
				Top:    rc.Top + (rc.Bottom-rc.Top-d)/2,
				Right:  rc.Left + (rc.Right-rc.Left-d)/2 + d,
				Bottom: rc.Top + (rc.Bottom-rc.Top-d)/2 + d,
			}
			p.Icon(a.icon, &ir, t.MutedForeground, 2)
		} else {
			size := int32(14)
			if a.size == AvatarSizeSm || elemW <= 28 {
				size = 12
			}
			spec := t.FontDefault
			spec.Size = size
			if f := xui.FontHandle(spec); f != 0 {
				p.Draw.SetFont(f)
			}
			p.Text(a.fallback, &rc, t.MutedForeground,
				xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		}
	}

	if a.badge {
		a.paintBadge(p, t)
	}
	return 0
}

func (a *Avatar) paintBadge(p *xui.Painter, t *xui.Theme) {
	base := a.size.px()
	d := a.size.badgeD()
	if a.badgeSize > 0 {
		d = a.badgeSize
	}
	if w := p.Rc.Right - p.Rc.Left; w > 0 && w != base {
		d = w * d / base
		if d < 6 {
			d = 6
		}
	}
	ring := int32(2)
	outer := d + ring*2
	// 徽标整体向内缩 circleInset: 头像的可见形状是圆, 元素矩形却是方的。徽标盒贴到元素右下角时,
	// 盒角落在圆外 → 白色的"环"会和卡片底色连成一片, 圆边被吃掉一块(看起来右下角是方的)。
	// 内缩后整个盒(含环)都落在圆内, 圆轮廓始终完整。
	band := a.circleInset()
	badge := xc.RECT{
		Left:   p.Rc.Right - outer - band,
		Top:    p.Rc.Bottom - outer - band,
		Right:  p.Rc.Right - band,
		Bottom: p.Rc.Bottom - band,
	}
	p.Draw.SetBrushColor(uint32(t.Background))
	p.Draw.FillEllipse(&badge)
	inner := insetRect(badge, ring)
	fill := t.Primary
	if a.badgeColor != nil {
		fill = a.badgeColor(t)
	}
	p.Draw.SetBrushColor(uint32(fill))
	p.Draw.FillEllipse(&inner)
	if a.badgeIcon != "" {
		ic := t.PrimaryForeground
		if a.badgeColor != nil {
			// 自定义底色时按亮度挑对比色(同 chart 对比色口径, 但局部实现:
			// avatar 单组件安装拿不到 chart.go)
			ic = avatarContrastText(t, fill)
		}
		if a.badgeIconColor != nil {
			ic = a.badgeIconColor(t)
		}
		ir := insetRect(inner, 1)
		p.Icon(a.badgeIcon, &ir, ic, 2)
	}
}

// avatarContrastText 在 Foreground / Background 之间挑一个与底色对比更高的文字色。
func avatarContrastText(t *xui.Theme, bg xui.Color) xui.Color {
	lb := avatarLum(bg)
	if avatarAbs(avatarLum(t.Foreground)-lb) >= avatarAbs(avatarLum(t.Background)-lb) {
		return t.Foreground
	}
	return t.Background
}

// avatarLum 返回颜色的 0..1 感知亮度(加权近似, 只用于挑对比色)。
func avatarLum(c xui.Color) float64 {
	r, g, b, _ := c.RGBA()
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
}

func avatarAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func insetRect(rc xc.RECT, n int32) xc.RECT {
	return xc.RECT{Left: rc.Left + n, Top: rc.Top + n, Right: rc.Right - n, Bottom: rc.Bottom - n}
}

// AvatarGroup 头像叠放组, 对应 shadcn avatar.tsx 的 AvatarGroup(-space-x-2)。
// 用横向布局 + 负间距实现重叠; 组内头像默认加 2px 背景色环。
type AvatarGroup struct {
	*widget.LayoutEle
	size AvatarSize
}

// AvatarGroupOption 头像组配置项。
type AvatarGroupOption func(*AvatarGroup)

// WithAvatarGroupSize 设置组内默认尺寸(默认 32 = default)。
func WithAvatarGroupSize(s AvatarSize) AvatarGroupOption {
	return func(g *AvatarGroup) { g.size = s }
}

// NewAvatarGroup 创建头像叠放组(宽高按内容自撑)。
func NewAvatarGroup(hParent int, opts ...AvatarGroupOption) *AvatarGroup {
	le := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if le == nil {
		return nil
	}
	g := &AvatarGroup{LayoutEle: le}
	for _, o := range opts {
		o(g)
	}
	le.EnableHorizon(true)
	le.EnableLayout(true)
	le.EnableAutoWrap(false)
	le.EnableBkTransparent(true)
	le.EnableDrawBorder(false)
	le.EnableFocus(false)
	le.SetSpace(-avatarGroupOverlap)
	le.SetAlignV(xcc.Layout_Align_Top)
	le.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	le.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return g
}

// Add 往组里加一个 fallback 头像, 默认带背景色环与组尺寸。
func (g *AvatarGroup) Add(fallback string, opts ...AvatarOption) *Avatar {
	if g == nil {
		return nil
	}
	base := []AvatarOption{WithAvatarSize(g.size), WithAvatarRing()}
	return NewAvatar(0, 0, 0, 0, g.Handle, fallback, append(base, opts...)...)
}

// AddCount 加一个 "+N" 计数头像。
func (g *AvatarGroup) AddCount(n int) *Avatar {
	return g.Add("+" + itoa(n))
}

// AddCountIcon 加一个只放图标的计数头像(对应上游 Avatar Group with Icon 示例)。
func (g *AvatarGroup) AddCountIcon(icon string) *Avatar {
	a := g.Add("")
	if a != nil {
		a.SetIcon(icon)
	}
	return a
}

// itoa 是 strconv.Itoa 的小包装, 避免组件文件为一个数字引入 strconv。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// avatarDPI 取元素所属窗口的 DPI(取不到按 96)。
//
// 每个头像元素自己生成一份圆形位图, 位图按**物理像素**尺寸生成: 框架把逻辑坐标映射到
// 设备像素绘制图片, 按物理尺寸生成时位图与目标 1:1, 抗锯齿的 1px 边缘才不会被再缩放一次
// (按 2×/4× 逻辑尺寸生成时, 下采样会把那一圈 AA 抹掉, 边缘变成硬台阶 —— 实测)。
func avatarDPI(hEle int) int32 {
	if hwnd := xc.XWidget_GetHWINDOW(hEle); hwnd != 0 {
		if dpi := xc.XWnd_GetDPI(hwnd); dpi > 0 {
			return dpi
		}
	}
	return 96
}

// avatarImageHandle 把任意图片居中裁剪成圆形 RGBA, 交给 xcgui 生成图片句柄。
//
// 位图边长 n = 逻辑直径 × DPI(物理像素), 与元素的物理尺寸一致: 绘制时按元素矩形铺满就是
// 1:1 映射, 既不会因下采样丢掉 1px 抗锯齿边, 也不会因缩放把圆边切平。inset 是环带宽度
// (逻辑像素), 圆半径扣掉它, 让出的透明边由调用方在下面铺环带色。
func avatarImageHandle(img image.Image, logical, dpi, inset int32) int {
	if img == nil {
		return 0
	}
	if dpi <= 0 {
		dpi = 96
	}
	n := int(float64(logical)*float64(dpi)/96 + 0.5)
	if n < 24 {
		n = 24
	}
	src := img.Bounds()
	sw, sh := float64(src.Dx()), float64(src.Dy())
	if sw <= 0 || sh <= 0 {
		return 0
	}
	// 居中裁成正方形, 再缩放到 n×n(双线性)
	side := math.Min(sw, sh)
	ox := float64(src.Min.X) + (sw-side)/2
	oy := float64(src.Min.Y) + (sh-side)/2
	scale := side / float64(n)
	cx, cy := float64(n)/2, float64(n)/2
	band := float64(inset) * float64(dpi) / 96 // 环带换算成物理像素
	radius := float64(n)/2 - band - 0.5
	if radius < 4 {
		radius = 4
	}

	data := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		sy := oy + (float64(y)+0.5)*scale - 0.5
		for x := 0; x < n; x++ {
			sx := ox + (float64(x)+0.5)*scale - 0.5
			r, g, b, a := bilinearNRGBA(img, sx, sy)
			// 抗锯齿圆形蒙版: 边缘 1px 内按覆盖率衰减
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			cov := radius - d + 0.5
			if cov <= 0 {
				continue
			}
			if cov > 1 {
				cov = 1
			}
			i := (y*n + x) * 4
			data[i] = r
			data[i+1] = g
			data[i+2] = b
			data[i+3] = byte(float64(a)*cov + 0.5)
		}
	}
	return xc.XImage_LoadFromDataRGBA(data, int32(n), int32(n))
}

// bilinearNRGBA 在源图上双线性采样, 返回非预乘 RGBA。
func bilinearNRGBA(img image.Image, sx, sy float64) (byte, byte, byte, byte) {
	b := img.Bounds()
	maxX, maxY := b.Max.X-1, b.Max.Y-1
	x0 := int(math.Floor(sx))
	y0 := int(math.Floor(sy))
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x0 > maxX {
		x0 = maxX
	}
	if y0 > maxY {
		y0 = maxY
	}
	x1, y1 := x0+1, y0+1
	if x1 > maxX {
		x1 = maxX
	}
	if y1 > maxY {
		y1 = maxY
	}
	fx := sx - float64(x0)
	if fx < 0 {
		fx = 0
	}
	if fx > 1 {
		fx = 1
	}
	fy := sy - float64(y0)
	if fy < 0 {
		fy = 0
	}
	if fy > 1 {
		fy = 1
	}
	r00, g00, b00, a00 := nrgba8(img.At(x0, y0))
	r10, g10, b10, a10 := nrgba8(img.At(x1, y0))
	r01, g01, b01, a01 := nrgba8(img.At(x0, y1))
	r11, g11, b11, a11 := nrgba8(img.At(x1, y1))
	mix := func(v00, v10, v01, v11 byte) byte {
		top := float64(v00)*(1-fx) + float64(v10)*fx
		bot := float64(v01)*(1-fx) + float64(v11)*fx
		return byte(top*(1-fy) + bot*fy + 0.5)
	}
	return mix(r00, r10, r01, r11), mix(g00, g10, g01, g11),
		mix(b00, b10, b01, b11), mix(a00, a10, a01, a11)
}

// nrgba8 取非预乘 8 位 RGBA; 用 NRGBA 模型转换, 避免 RGBA() 的预乘结果让透明边缘发黑。
func nrgba8(c interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) (byte, byte, byte, byte) {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return 0, 0, 0, 0
	}
	// RGBA() 是预乘值, 反预乘回直通 alpha
	return byte((r * 0xffff / a) >> 8), byte((g * 0xffff / a) >> 8),
		byte((b * 0xffff / a) >> 8), byte(a >> 8)
}
