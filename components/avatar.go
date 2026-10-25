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

// WithAvatarBadge 在右下角显示状态徽标; icon 可为空(纯色圆点)或 lucide path。
func WithAvatarBadge(icon string) AvatarOption {
	return func(a *Avatar) { a.badge = true; a.badgeIcon = icon }
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
	size      AvatarSize
	fallback  string
	ring      bool
	badge     bool
	badgeIcon string

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
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
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

// SetBadge 开关右下角徽标; icon 为空时画纯色圆点。
func (a *Avatar) SetBadge(on bool, icon string) *Avatar {
	if a == nil {
		return a
	}
	a.badge, a.badgeIcon = on, icon
	a.Redraw(false)
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
	a.imgHandle = avatarImageHandle(img, a.GetWidth())
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
	if a.ring {
		// ring-2 ring-background: 画在头像内侧, 避免子元素被父矩形裁掉
		p.Draw.SetBrushColor(uint32(t.Background))
		p.Draw.FillEllipse(&rc)
		rc = insetRect(rc, 2)
	}

	if a.imgHandle != 0 {
		p.Draw.XDraw_ImageEx(a.imgHandle, rc.Left, rc.Top, rc.Right-rc.Left, rc.Bottom-rc.Top)
	} else {
		p.Draw.SetBrushColor(uint32(t.Muted))
		p.Draw.FillEllipse(&rc)
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

	if a.badge {
		a.paintBadge(p, t)
	}
	return 0
}

func (a *Avatar) paintBadge(p *xui.Painter, t *xui.Theme) {
	d := a.size.badgeD()
	if w := p.Rc.Right - p.Rc.Left; w > 0 && w != a.size.px() {
		d = w * d / a.size.px()
		if d < 6 {
			d = 6
		}
	}
	ring := int32(2)
	outer := d + ring*2
	badge := xc.RECT{
		Left:   p.Rc.Right - outer,
		Top:    p.Rc.Bottom - outer,
		Right:  p.Rc.Right,
		Bottom: p.Rc.Bottom,
	}
	p.Draw.SetBrushColor(uint32(t.Background))
	p.Draw.FillEllipse(&badge)
	inner := insetRect(badge, ring)
	p.Draw.SetBrushColor(uint32(t.Primary))
	p.Draw.FillEllipse(&inner)
	if a.badgeIcon != "" {
		ir := insetRect(inner, 1)
		p.Icon(a.badgeIcon, &ir, t.PrimaryForeground, 2)
	}
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

// avatarImageHandle 把任意图片居中裁剪成圆形 RGBA, 交给 xcgui 生成图片句柄。
// n 取 4 倍逻辑尺寸(至少 96), 保证高 DPI 下边缘平滑。
func avatarImageHandle(img image.Image, logical int32) int {
	if img == nil {
		return 0
	}
	n := int(logical) * 4
	if n < 96 {
		n = 96
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
	radius := float64(n)/2 - 0.5

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
