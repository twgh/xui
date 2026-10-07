package ui

import (
	"image"
	"syscall"
	"unsafe"

	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// 对话框遮罩的背景模糊(shadcn dialog 的 backdrop-blur-xs)。
//
// xcgui 没有背景模糊, 元素 paint 里也画不出位图(DESIGN 第八节 17), 所以绕行:
// 打开对话框前先用 GDI 把窗口内容抓成位图 → Go 里做盒式模糊(≈高斯) → 装进
// "底层元素的背景图片填充"(元素透明度能跟着遮罩一起淡入, 自绘/形状都不行)。
// 模态打开期间背景本来就不会变, 静态快照足够。

// backdropBlurPx 模糊半径(逻辑像素) = backdrop-blur-xs。
const backdropBlurPx = 4

var (
	modUser32              = syscall.NewLazyDLL("user32.dll")
	modGdi32               = syscall.NewLazyDLL("gdi32.dll")
	procGetWindowDC        = modUser32.NewProc("GetWindowDC")
	procReleaseDC          = modUser32.NewProc("ReleaseDC")
	procPrintWindow        = modUser32.NewProc("PrintWindow")
	procCreateCompatibleDC = modGdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = modGdi32.NewProc("DeleteDC")
	procCreateDIBSection   = modGdi32.NewProc("CreateDIBSection")
	procDeleteObject       = modGdi32.NewProc("DeleteObject")
	procSelectObject       = modGdi32.NewProc("SelectObject")
	procBitBlt             = modGdi32.NewProc("BitBlt")
)

// bitmapinfoheader 与 Win32 的 BITMAPINFOHEADER 对齐(只用到自上而下 32bit BI_RGB)。
type bitmapinfoheader struct {
	Size          uint32
	Width         int32
	Height        int32 // 负数 = 自上而下的行序
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// newBackdrop 创建承载模糊背景的元素(窗口子元素, 创建即隐藏, z 序在 Show 时抬到遮罩之下)。
func newBackdrop(hWnd int) *widget.Element {
	e := widget.NewElement(0, 0, 10, 10, hWnd)
	if e == nil {
		return nil
	}
	// 与遮罩同样的"灭原生框"三件套(实测: 容器元素被点击会画蓝色焦点框)
	e.EnableFocus(false).EnableDrawBorder(false)
	e.Show(false)
	return e
}

// refreshBackdrop 抓取宿主窗口当前画面 → 模糊 → 缩放到逻辑尺寸 → 装进背景元素并显示。
// 任一步失败都保持隐藏(只损失模糊, 不影响对话框功能), recover 兜底同理。
func (d *Dialog) refreshBackdrop() {
	defer func() { _ = recover() }()
	if d.backdrop == nil {
		return
	}
	d.backdrop.Show(false)
	// 注意句柄级别: 抓屏要的是 Win32 HWND, 必须经窗口 wrapper 的 GetHWND()
	// (XWidget_GetHWND 吃的是元素句柄, 喂窗口句柄会弹"输入句柄可能无效"错误框, DESIGN 第八节 34)
	hwnd := d.win.GetHWND()
	var pr xc.RECT
	if !wapi.GetWindowRect(hwnd, &pr) {
		return
	}
	w, h := int(pr.Right-pr.Left), int(pr.Bottom-pr.Top)
	img := captureWindowRGBA(hwnd, w, h)
	if img == nil {
		return
	}
	dpi := xc.XWnd_GetDPI(d.hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	// 物理像素上做模糊(半径 = 逻辑 4px × DPI), 再缩到逻辑尺寸:
	// BkM 图片是按 1:1 逻辑像素居中绘制的(实测), 物理尺寸的图直接喂会错位且被裁
	boxBlurRGBA(img, int(float64(backdropBlurPx)*float64(dpi)/96))

	var wr xc.RECT
	xc.XWnd_GetRect(d.hWnd, &wr)
	lw, lh := int(wr.Right-wr.Left), int(wr.Bottom-wr.Top)
	if lw <= 0 || lh <= 0 {
		return
	}
	img = scaleRGBA(img, lw, lh)

	if d.backdropImg != 0 {
		xc.XImage_Release(d.backdropImg)
		d.backdropImg = 0
	}
	hImg := xc.XImage_LoadFromGoImage(img)
	if hImg == 0 {
		return
	}
	d.backdropImg = hImg
	// BkM 按状态选择注册: 注意元素的"离开"态是 Element_State_Flag_Leave(0x0010),
	// 不是窗口的 Window_State_Flag_Leave(0x0001) —— 用错状态注册永不渲染(实测踩坑);
	// 经元素自身 BkM 管理器以固定 id 写入(元素与窗口两种宿主环境实测都渲染)
	xc.XEle_ClearBkInfo(d.backdrop.Handle)
	hm := xc.XEle_GetBkManager(d.backdrop.Handle)
	xc.XBkM_AddImage(hm, xcc.Element_State_Flag_Leave, hImg, backdropBkImgID)
	setEleRect(d.backdrop.Handle, 0, 0, int32(lw), int32(lh))
	d.backdrop.Show(true)
}

// backdropBkImgID 模糊快照在元素 BkM 管理器里的固定对象 id。
const backdropBkImgID int32 = 1

// scaleRGBA 把 RGBA 图面积平均缩放到 dstW×dstH(缩小用盒滤波足够, 快照场景质量没问题)。
func scaleRGBA(img *image.RGBA, dstW, dstH int) *image.RGBA {
	sw, sh := img.Rect.Dx(), img.Rect.Dy()
	if sw == dstW && sh == dstH {
		return img
	}
	if sw <= 0 || sh <= 0 || dstW <= 0 || dstH <= 0 {
		return img
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for dy := 0; dy < dstH; dy++ {
		sy0 := dy * sh / dstH
		sy1 := (dy + 1) * sh / dstH
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for dx := 0; dx < dstW; dx++ {
			sx0 := dx * sw / dstW
			sx1 := (dx + 1) * sw / dstW
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var sum [4]int32
			for sy := sy0; sy < sy1; sy++ {
				row := sy * sw * 4
				for sx := sx0; sx < sx1; sx++ {
					i := row + sx*4
					sum[0] += int32(img.Pix[i])
					sum[1] += int32(img.Pix[i+1])
					sum[2] += int32(img.Pix[i+2])
					sum[3] += int32(img.Pix[i+3])
				}
			}
			n := int32((sx1 - sx0) * (sy1 - sy0))
			j := (dy*dstW + dx) * 4
			dst.Pix[j] = byte(sum[0] / n)
			dst.Pix[j+1] = byte(sum[1] / n)
			dst.Pix[j+2] = byte(sum[2] / n)
			dst.Pix[j+3] = byte(sum[3] / n)
		}
	}
	return dst
}

// captureWindowRGBA 用 PrintWindow(PW_RENDERFULLCONTENT) 把窗口整幅抓成 RGBA。
// 对 D2D 自绘窗口这是最稳的一条路; PrintWindow 失败时退回 BitBlt。
func captureWindowRGBA(hwnd uintptr, w, h int) *image.RGBA {
	if hwnd == 0 || w <= 0 || h <= 0 {
		return nil
	}
	hdcWnd, _, _ := procGetWindowDC.Call(hwnd)
	if hdcWnd == 0 {
		return nil
	}
	defer procReleaseDC.Call(hwnd, hdcWnd)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcWnd)
	if hdcMem == 0 {
		return nil
	}
	defer procDeleteDC.Call(hdcMem)

	bi := bitmapinfoheader{Size: 40, Width: int32(w), Height: int32(-h), Planes: 1, BitCount: 32}
	var bits unsafe.Pointer
	hbm, _, _ := procCreateDIBSection.Call(hdcMem, uintptr(unsafe.Pointer(&bi)), 0,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 || bits == nil {
		return nil
	}
	defer procDeleteObject.Call(hbm)
	old, _, _ := procSelectObject.Call(hdcMem, hbm)
	defer procSelectObject.Call(hdcMem, old)

	r, _, _ := procPrintWindow.Call(hwnd, hdcMem, 2) // PW_RENDERFULLCONTENT
	if r == 0 {
		procBitBlt.Call(hdcMem, 0, 0, uintptr(w), uintptr(h), hdcWnd, 0, 0, 0x00CC0020) // SRCCOPY
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	src := unsafe.Slice((*byte)(bits), w*h*4)
	n := w * h
	for i := 0; i < n; i++ { // DIB 是 BGRA, image.RGBA 要 RGBA
		img.Pix[i*4+0] = src[i*4+2]
		img.Pix[i*4+1] = src[i*4+1]
		img.Pix[i*4+2] = src[i*4+0]
		img.Pix[i*4+3] = 255
	}
	return img
}

// boxBlurRGBA 盒式模糊两轮(横向+纵向 ×2), 视觉上足够接近高斯, 半径为物理像素。
func boxBlurRGBA(img *image.RGBA, r int) {
	if r <= 0 || img == nil {
		return
	}
	w, h := img.Rect.Dx(), img.Rect.Dy()
	if w < 2 || h < 2 {
		return
	}
	tmp := image.NewRGBA(img.Rect)
	boxBlurPass(img.Pix, tmp.Pix, w, h, r, false)
	boxBlurPass(tmp.Pix, img.Pix, w, h, r, true)
	boxBlurPass(img.Pix, tmp.Pix, w, h, r, false)
	boxBlurPass(tmp.Pix, img.Pix, w, h, r, true)
}

// boxBlurPass 一趟盒式模糊: vertical=false 沿 x(src→dst), true 沿 y(src→dst)。滑动窗口求和。
func boxBlurPass(src, dst []byte, w, h, r int, vertical bool) {
	win := 2*r + 1
	clamp := func(v, lo, hi int) int {
		if v < lo {
			return lo
		} else if v > hi {
			return hi
		}
		return v
	}
	if !vertical {
		for y := 0; y < h; y++ {
			row := y * w * 4
			var sum [4]int
			for i := -r; i <= r; i++ {
				px := row + clamp(i, 0, w-1)*4
				sum[0] += int(src[px])
				sum[1] += int(src[px+1])
				sum[2] += int(src[px+2])
				sum[3] += int(src[px+3])
			}
			for x := 0; x < w; x++ {
				px := row + x*4
				dst[px] = byte(sum[0] / win)
				dst[px+1] = byte(sum[1] / win)
				dst[px+2] = byte(sum[2] / win)
				dst[px+3] = byte(sum[3] / win)
				out := row + clamp(x-r, 0, w-1)*4
				in := row + clamp(x+r+1, 0, w-1)*4
				sum[0] += int(src[in]) - int(src[out])
				sum[1] += int(src[in+1]) - int(src[out+1])
				sum[2] += int(src[in+2]) - int(src[out+2])
				sum[3] += int(src[in+3]) - int(src[out+3])
			}
		}
		return
	}
	for x := 0; x < w; x++ {
		col := x * 4
		var sum [4]int
		for i := -r; i <= r; i++ {
			px := col + clamp(i, 0, h-1)*w*4
			sum[0] += int(src[px])
			sum[1] += int(src[px+1])
			sum[2] += int(src[px+2])
			sum[3] += int(src[px+3])
		}
		for y := 0; y < h; y++ {
			px := col + y*w*4
			dst[px] = byte(sum[0] / win)
			dst[px+1] = byte(sum[1] / win)
			dst[px+2] = byte(sum[2] / win)
			dst[px+3] = byte(sum[3] / win)
			out := col + clamp(y-r, 0, h-1)*w*4
			in := col + clamp(y+r+1, 0, h-1)*w*4
			sum[0] += int(src[in]) - int(src[out])
			sum[1] += int(src[in+1]) - int(src[out+1])
			sum[2] += int(src[in+2]) - int(src[out+2])
			sum[3] += int(src[in+3]) - int(src[out+3])
		}
	}
}
