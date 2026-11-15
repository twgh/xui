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
	"github.com/twgh/xui"
)

// AlertDialog 警示对话框: 打断用户并要求明确选择(取消 / 继续)。
//
// 与 shadcn alert-dialog.tsx(radix-nova) 对齐:
//   - 没有右上角关闭按钮, 点遮罩不关闭(radix 的 onInteractOutside preventDefault),
//     ESC 关闭; 打开时焦点强制落在 Cancel(radix 的 onOpenAutoFocus);
//   - default: 头部左对齐(gap-1.5)、页脚右对齐一行; sm: 头部居中、页脚等宽双列;
//   - media = size-10 rounded-md bg-muted + size-6 图标, destructive 变体用
//     bg-destructive/10(text-destructive); sm 下 media 堆在标题上方(mb-2);
//   - Action 用调用方给的变体(primary / destructive), Cancel 默认 outline, 两者点击后都关闭;
//   - 与 Dialog 共用遮罩、backdrop-blur、焦点陷阱与 100ms 淡入淡出。
//
// 与上游的差异: zoom-in-95 不做(元素级缩放对自绘元素无效, 见 DESIGN 第八节 47);
// text-balance / text-pretty 不做(自绘文字没有该排版能力)。
type AlertDialog struct {
	d      *Dialog
	cancel *Button
	action *Button
}

// AlertDialogOption 警示对话框配置项。
type AlertDialogOption func(*AlertDialog)

// WithAlertDialogSize 设置尺寸(默认 AlertDialogSizeDefault)。
func WithAlertDialogSize(s AlertDialogSize) AlertDialogOption {
	return func(a *AlertDialog) { a.SetSize(s) }
}

// WithAlertDialogTitle 设置标题(等价于构造后调 SetTitle)。
func WithAlertDialogTitle(s string) AlertDialogOption {
	return func(a *AlertDialog) { a.SetTitle(s) }
}

// WithAlertDialogDescription 设置描述(等价于构造后调 SetDescription)。
func WithAlertDialogDescription(s string) AlertDialogOption {
	return func(a *AlertDialog) { a.SetDescription(s) }
}

// WithAlertDialogMedia 设置头部媒体块图标(path 数据或整段 SVG; 空串 = 无媒体块)。
func WithAlertDialogMedia(icon string) AlertDialogOption {
	return func(a *AlertDialog) { a.SetMedia(icon) }
}

// WithAlertDialogMediaImage 用图片作头部媒体块(对应 AlertDialogMedia 里放 `<img>`):
// 图片居中裁剪成圆角矩形, 与图标形态互斥(后设置的那个生效)。
func WithAlertDialogMediaImage(path string) AlertDialogOption {
	return func(a *AlertDialog) { a.SetMediaImage(path) }
}

// WithAlertDialogMediaTone 设置媒体块配色(默认 AlertDialogMediaMuted)。
func WithAlertDialogMediaTone(tone AlertDialogMediaTone) AlertDialogOption {
	return func(a *AlertDialog) { a.SetMediaTone(tone) }
}

// NewAlertDialog 创建警示对话框(默认隐藏)。hWnd 为宿主窗口句柄。
func NewAlertDialog(hWnd int, opts ...AlertDialogOption) *AlertDialog {
	d := NewDialog(hWnd, WithDialogCloseButton(false))
	if d == nil {
		return nil
	}
	d.alert = true
	d.dismissOutside = false
	a := &AlertDialog{d: d}
	for _, o := range opts {
		o(a)
	}
	a.applyAlignment()
	return a
}

// SetTitle 设置标题; 传空串隐藏标题。
func (a *AlertDialog) SetTitle(s string) *AlertDialog {
	if a != nil {
		a.d.SetTitle(s)
	}
	return a
}

// SetDescription 设置描述(text-sm, muted; 按宽度自动折行); 传空串隐藏描述。
func (a *AlertDialog) SetDescription(s string) *AlertDialog {
	if a != nil {
		a.d.SetDescription(s)
	}
	return a
}

// SetSize 切换尺寸(同时切换头部对齐与页脚排布)。
func (a *AlertDialog) SetSize(s AlertDialogSize) *AlertDialog {
	if a == nil {
		return a
	}
	a.d.alertSize = s
	a.applyAlignment()
	a.d.relayoutIfOpen()
	return a
}

// Size 返回当前尺寸。
func (a *AlertDialog) Size() AlertDialogSize {
	if a == nil {
		return AlertDialogSizeDefault
	}
	return a.d.alertSize
}

// SetMedia 设置头部媒体块图标(path 数据或整段 SVG; 空串 = 隐藏)。
// 图标与图片互斥 —— 这里会清掉 SetMediaImage 设的图片。
func (a *AlertDialog) SetMedia(icon string) *AlertDialog {
	if a == nil {
		return a
	}
	d := a.d
	a.ensureMedia()
	if d.media == nil {
		return a
	}
	if d.mediaImg != 0 {
		xc.XImage_Destroy(d.mediaImg)
		d.mediaImg = 0
	}
	d.mediaImgPath = ""
	d.mediaIcon = icon
	// 媒体块可能在构造函数末尾的整树隐藏之后创建, 显隐要跟对话框状态同步
	d.media.Show(icon != "")
	showEleTree(d.media.Handle, d.open && icon != "")
	d.media.Redraw(false)
	d.relayoutIfOpen()
	return a
}

// ensureMedia 按需创建媒体块元素(图标与图片两种形态共用同一个 40×40 容器)。
func (a *AlertDialog) ensureMedia() {
	if a == nil || a.d == nil {
		return
	}
	d := a.d
	if d.media != nil {
		return
	}
	e := widget.NewElement(0, 0, alertDlgMediaSize, alertDlgMediaSize, d.overlay.Handle)
	if e == nil {
		return
	}
	e.EnableBkTransparent(true)
	e.EnableDrawBorder(false)
	e.EnableFocus(false)
	e.EnableSwitchFocus(false)
	e.AddEvent_Paint(d.onMediaPaint)
	d.media = e
}

// SetMediaImage 用图片作头部媒体块(替代图标): 居中裁剪成圆角矩形; 传空串清空图片。
// 图片与图标互斥 —— 这里会清掉 SetMedia 设的图标。
func (a *AlertDialog) SetMediaImage(path string) *AlertDialog {
	if a == nil {
		return a
	}
	d := a.d
	a.ensureMedia()
	if d.media == nil {
		return a
	}
	if d.mediaImg != 0 {
		xc.XImage_Destroy(d.mediaImg)
		d.mediaImg = 0
	}
	if path != "" {
		d.mediaImg = alertDlgMediaHandle(path, alertDlgMediaSize, alertDlgDPI(d.media.Handle))
	}
	d.mediaIcon = ""
	d.mediaImgPath = path
	show := d.mediaImg != 0
	d.media.Show(show)
	showEleTree(d.media.Handle, d.open && show)
	d.media.Redraw(false)
	d.relayoutIfOpen()
	return a
}

// MediaImage 返回媒体块图片路径(没有图片时为空串)。
func (a *AlertDialog) MediaImage() string {
	if a == nil {
		return ""
	}
	return a.d.mediaImgPath
}

// Media 返回媒体块图标(空串 = 没有)。
func (a *AlertDialog) Media() string {
	if a == nil {
		return ""
	}
	return a.d.mediaIcon
}

// SetMediaTone 设置媒体块配色。
func (a *AlertDialog) SetMediaTone(tone AlertDialogMediaTone) *AlertDialog {
	if a == nil {
		return a
	}
	a.d.mediaTone = tone
	if a.d.media != nil {
		a.d.media.Redraw(false)
	}
	return a
}

// AddCancel 追加取消按钮(outline, 默认聚焦)。点击后先执行 onClick(可为 nil)再关闭;
// 对应 AlertDialogCancel。可以在 AddAction 前后调用, 打开时焦点总是 Cancel。
func (a *AlertDialog) AddCancel(text string, onClick func()) *AlertDialog {
	if a == nil {
		return a
	}
	btn := a.d.addFooterButton(text, ButtonOutline)
	if btn == nil {
		return a
	}
	btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if onClick != nil {
			onClick()
		}
		a.Close()
		return 0
	})
	a.cancel = btn
	a.d.initialFocus = btn.Handle
	a.d.relayoutIfOpen()
	return a
}

// AddAction 追加确认按钮(变体由调用方给, 通常是 ButtonPrimary / ButtonDestructive)。
// 点击后先执行 onClick(可为 nil)再关闭; 对应 AlertDialogAction。
func (a *AlertDialog) AddAction(text string, variant ButtonVariant, onClick func()) *AlertDialog {
	if a == nil {
		return a
	}
	btn := a.d.addFooterButton(text, variant)
	if btn == nil {
		return a
	}
	btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if onClick != nil {
			onClick()
		}
		a.Close()
		return 0
	})
	a.action = btn
	if a.d.initialFocus == 0 {
		a.d.initialFocus = btn.Handle
	}
	a.d.relayoutIfOpen()
	return a
}

// CancelButton 返回 Cancel 按钮(可在 AddCancel 后改变体/图标); 未添加时 nil。
func (a *AlertDialog) CancelButton() *Button {
	if a == nil {
		return nil
	}
	return a.cancel
}

// ActionButton 返回 Action 按钮(可在 AddAction 后改变体/图标); 未添加时 nil。
func (a *AlertDialog) ActionButton() *Button {
	if a == nil {
		return nil
	}
	return a.action
}

// BindTrigger 把控件绑成"点击即打开"(= AlertDialogTrigger)。
func (a *AlertDialog) BindTrigger(t clickBinder) *AlertDialog {
	if a == nil || t == nil {
		return a
	}
	t.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		a.Show()
		return 0
	})
	return a
}

// Content 返回内容容器句柄(可放额外内容; AlertDialog 通常只有头部 + 页脚)。
func (a *AlertDialog) Content() int {
	if a == nil {
		return 0
	}
	return a.d.Content()
}

// ContentWidth 内容容器宽度(面板宽 - 左右 padding)。
func (a *AlertDialog) ContentWidth() int32 {
	if a == nil {
		return 0
	}
	return a.d.ContentWidth()
}

// Show 打开警示对话框。
func (a *AlertDialog) Show() *AlertDialog {
	if a != nil {
		a.d.Show()
	}
	return a
}

// Hide 关闭(不触发 OnClose 回调)。
func (a *AlertDialog) Hide() *AlertDialog {
	if a != nil {
		a.d.Hide()
	}
	return a
}

// Close 关闭并触发关闭回调。
func (a *AlertDialog) Close() *AlertDialog {
	if a != nil {
		a.d.Close()
	}
	return a
}

// IsOpen 报告是否打开。
func (a *AlertDialog) IsOpen() bool {
	return a != nil && a.d.IsOpen()
}

// OnClose 注册关闭回调(ESC / Cancel / Action / Close 都会触发; 点遮罩不会)。
func (a *AlertDialog) OnClose(fn func()) *AlertDialog {
	if a != nil {
		a.d.OnClose(fn)
	}
	return a
}

// OnOpenChange 注册开合状态回调(Show / Hide / ESC / Cancel / Action / Close 都走),
// 语义同 Dialog.OnOpenChange —— 同一状态不重复通知。
func (a *AlertDialog) OnOpenChange(fn func(open bool)) *AlertDialog {
	if a != nil {
		a.d.OnOpenChange(fn)
	}
	return a
}

// SetOpen 受控入口: true 打开、false 关闭(false 不触发 OnClose, 同 Dialog)。
func (a *AlertDialog) SetOpen(open bool) *AlertDialog {
	if a != nil {
		a.d.SetOpen(open)
	}
	return a
}

// Refresh 重新量内容并重排(打开状态下改过内容后调用)。
func (a *AlertDialog) Refresh() *AlertDialog {
	if a != nil {
		a.d.Refresh()
	}
	return a
}

// alertDlgMediaHandle 把图片文件居中裁剪成**圆角矩形**位图, 交给 xcgui 生成图片句柄。
//
// 与 Avatar 的圆形位图同一套做法(按物理像素生成、1:1 铺满元素, 避免二次缩放把
// 1px 抗锯齿边切平), 只是蒙版换成圆角矩形: 媒体块是 size-10 rounded-md,
// 不裁剪的话方形图片会把四个圆角盖成直角。
func alertDlgMediaHandle(path string, logical, dpi int32) int {
	if path == "" {
		return 0
	}
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return 0
	}
	if dpi <= 0 {
		dpi = 96
	}
	n := int(float64(logical)*float64(dpi)/96 + 0.5)
	if n < 16 {
		n = 16
	}
	src := img.Bounds()
	sw, sh := float64(src.Dx()), float64(src.Dy())
	if sw <= 0 || sh <= 0 {
		return 0
	}
	side := math.Min(sw, sh)
	ox := float64(src.Min.X) + (sw-side)/2
	oy := float64(src.Min.Y) + (sh-side)/2
	scale := side / float64(n)
	// 圆角半径与面板同一档(RadiusMd), 按物理像素换算。
	radius := float64(xui.Current().RadiusMd()) * float64(dpi) / 96
	if radius > float64(n)/2 {
		radius = float64(n) / 2
	}
	half := float64(n)/2 - 0.5

	data := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		sy := oy + (float64(y)+0.5)*scale - 0.5
		for x := 0; x < n; x++ {
			sx := ox + (float64(x)+0.5)*scale - 0.5
			r, g, b, a := alertDlgSample(img, sx, sy)
			// 圆角矩形的有符号距离: 先按"内缩 radius 的矩形"求到边的距离, 再减半径。
			dx := math.Abs(float64(x)+0.5-half) - (half - radius)
			dy := math.Abs(float64(y)+0.5-half) - (half - radius)
			dx, dy = math.Max(dx, 0), math.Max(dy, 0)
			cov := radius - math.Hypot(dx, dy) + 0.5
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

// alertDlgDPI 取元素所属窗口的 DPI(取不到按 96)。位图按物理像素生成, 1:1 铺满元素。
func alertDlgDPI(hEle int) int32 {
	if hwnd := xc.XWidget_GetHWINDOW(hEle); hwnd != 0 {
		if dpi := xc.XWnd_GetDPI(hwnd); dpi > 0 {
			return dpi
		}
	}
	return 96
}

// alertDlgSample 在源图上做双线性采样, 返回非预乘 RGBA。
//
// 与 Avatar 的采样实现同法, 但**不能直接复用**(组件的源码是按闭包单独下发的,
// alert-dialog 的闭包里没有 avatar.go)。这里是它自己的副本。
func alertDlgSample(img image.Image, sx, sy float64) (byte, byte, byte, byte) {
	b := img.Bounds()
	maxX, maxY := b.Max.X-1, b.Max.Y-1
	clamp := func(x, lo, hi int) int {
		if x < lo {
			return lo
		}
		if x > hi {
			return hi
		}
		return x
	}
	x0 := clamp(int(math.Floor(sx)), b.Min.X, maxX)
	y0 := clamp(int(math.Floor(sy)), b.Min.Y, maxY)
	x1, y1 := clamp(x0+1, b.Min.X, maxX), clamp(y0+1, b.Min.Y, maxY)
	fx := sx - float64(x0)
	if fx < 0 {
		fx = 0
	} else if fx > 1 {
		fx = 1
	}
	fy := sy - float64(y0)
	if fy < 0 {
		fy = 0
	} else if fy > 1 {
		fy = 1
	}
	r00, g00, b00, a00 := alertDlgNRGBA(img.At(x0, y0))
	r10, g10, b10, a10 := alertDlgNRGBA(img.At(x1, y0))
	r01, g01, b01, a01 := alertDlgNRGBA(img.At(x0, y1))
	r11, g11, b11, a11 := alertDlgNRGBA(img.At(x1, y1))
	mix := func(v00, v10, v01, v11 byte) byte {
		top := float64(v00)*(1-fx) + float64(v10)*fx
		bot := float64(v01)*(1-fx) + float64(v11)*fx
		return byte(top*(1-fy) + bot*fy + 0.5)
	}
	return mix(r00, r10, r01, r11), mix(g00, g10, g01, g11),
		mix(b00, b10, b01, b11), mix(a00, a10, a01, a11)
}

// alertDlgNRGBA 取非预乘 8 位 RGBA(避免 RGBA() 的预乘结果让透明边缘发黑)。
func alertDlgNRGBA(c interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) (byte, byte, byte, byte) {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return 0, 0, 0, 0
	}
	return byte((r * 0xffff / a) >> 8), byte((g * 0xffff / a) >> 8),
		byte((b * 0xffff / a) >> 8), byte(a >> 8)
}

// applyAlignment 按尺寸切换标题/描述对齐: default 左对齐, sm 居中。
func (a *AlertDialog) applyAlignment() {
	if a == nil || a.d == nil {
		return
	}
	if a.d.alertSize == AlertDialogSizeSmall {
		a.d.title.SetAlignCenter()
		a.d.desc.SetAlignCenter()
	} else {
		a.d.title.SetAlignLeft()
		a.d.desc.SetAlignLeft()
	}
}
