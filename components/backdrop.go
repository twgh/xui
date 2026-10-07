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
// xcgui 没有背景模糊, 所以绕行:
// 打开对话框前先用 GDI 把窗口内容抓成位图 → Go 里做盒式模糊(≈高斯) → 缩放到
// 逻辑尺寸 → 装进"底层元素的背景图片填充"(元素透明度能跟着遮罩一起淡入)。
// 模态打开期间背景本来就不会变, 静态快照足够。
// (历史注: 当年选 BkM 路线是因为误以为元素 paint 画不出位图, 2026-10-07 复核
// paint 里 XDraw_Image/ImageEx 能画位图 —— 见 DESIGN 第八节 17 修正; BkM 路线
// 已验证且淡入一致, 保留。)

// backdropBlurPx 盒式模糊半径(单位 = 半分辨率像素, 见 backdropBlurDown) = backdrop-blur-xs。
// 双趟盒式(r=2)在半分辨率上的等效 σ≈2.0, 放大回逻辑尺寸后 ≈4.0 —— 与原先"整分辨率双趟
// r=4"(σ≈3.65)基本一致, 两者都是 CSS blur(4px) 的近似。
const backdropBlurPx = 2

// backdropBlurDown 模糊前的降采样倍数(边长)。模糊是全屏弹窗里最贵的一环: 1707×1019 整
// 分辨率双趟实测 65ms, 降到 1/2 边长(面积 1/4)只要 16ms; 而降采样 + 放大对"再模糊 4px"
// 的观感几乎没影响(总 σ 与原来同为 ~4 逻辑像素)。见 DESIGN 第八节 47。
const backdropBlurDown = 2

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

// refreshBackdrop 抓取宿主窗口当前画面 → 裁阴影带 → 半分辨率模糊 → 放大回逻辑尺寸 → 装进背景元素。
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
	dpi := xc.XWnd_GetDPI(d.hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	// 抓屏要 DIB 的**物理像素**尺寸(PrintWindow 按 1:1 物理像素画整窗并裁剪, 不会缩放 ——
	// 见 captureWindowScaled), 并在同一次遍历里直接降采样到 1/backdropBlurDown 逻辑尺寸:
	// 模糊与放大都在这个低分辨率上做, 物理尺寸的整图中间结果一次都不产生。
	physW, physH := int(pr.Right-pr.Left), int(pr.Bottom-pr.Top)
	lwFull := physW * 96 / int(dpi) // 窗口逻辑尺寸(与 Body 矩形比较用)
	lhFull := physH * 96 / int(dpi)
	cw := (lwFull + backdropBlurDown - 1) / backdropBlurDown
	ch := (lhFull + backdropBlurDown - 1) / backdropBlurDown
	if cw <= 0 || ch <= 0 {
		return
	}
	img := captureWindowScaled(hwnd, physW, physH, cw, ch)
	if img == nil {
		return
	}
	// 阴影窗口(Window_Transparent_Shadow)的 Win32 矩形含四周阴影边带 —— PrintWindow
	// 把它抓成纯黑(alpha 丢失), 模糊后会往画面里晕出一圈黑边(实测)。裁掉只留 Body 区;
	// 非阴影窗口 Body = 整窗, 仅当 Body 严格小于窗口(存在阴影带)时才裁。
	// Body 是逻辑坐标、快照是 1/backdropBlurDown 逻辑分辨率, 所以两个矩形都要除以倍数。
	var body xc.RECT
	xc.XWnd_GetBodyRect(d.hWnd, &body)
	x0, y0 := int(body.Left)/backdropBlurDown, int(body.Top)/backdropBlurDown
	x1, y1 := int(body.Right)/backdropBlurDown, int(body.Bottom)/backdropBlurDown
	if body.Right > body.Left && body.Bottom > body.Top &&
		x1 <= cw && y1 <= ch && (x0 > 0 || y0 > 0 || x1 < cw || y1 < ch) {
		img = cropRGBA(img, x0, y0, x1, y1)
	}
	bx, by, lw, lh := d.backdropPlacement()
	if lw <= 0 || lh <= 0 {
		return
	}
	// 快照已经是 1/backdropBlurDown 逻辑分辨率, 直接在上面模糊(半径也用 backdropBlurPx);
	// 最后放大回**精确**逻辑尺寸 —— BkM 图片按 1:1 逻辑像素居中绘制(实测), 尺寸必须严丝合缝。
	boxBlurRGBA(img, backdropBlurPx)
	img = upscaleRGBA(img, int(lw), int(lh))

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
	setEleRect(d.backdrop.Handle, bx, by, lw, lh)
	d.backdrop.Show(true)
}

// backdropPlacement 返回模糊层元素的位置与尺寸(逻辑像素)。
// 关键: 子元素坐标系的原点是 Win32 整窗左上角(**含阴影带**), XWnd_GetBodyRect 给出的
// (12,12) 偏移正是 Body 在这个坐标系里的位置 —— 快照裁掉阴影带后必须画在 Body 位置,
// 仍画在 (0,0) 会让内容整体偏移一个阴影带宽(实测主窗标题被削掉一半);
// 非阴影窗口 Body=整窗, 铺满即可。
func (d *Dialog) backdropPlacement() (x, y, w, h int32) {
	var wr xc.RECT
	xc.XWnd_GetRect(d.hWnd, &wr)
	w, h = wr.Right - wr.Left, wr.Bottom - wr.Top
	if w <= 0 || h <= 0 {
		return 0, 0, 0, 0
	}
	var body xc.RECT
	xc.XWnd_GetBodyRect(d.hWnd, &body)
	if body.Right > body.Left && body.Bottom > body.Top &&
		(body.Left > 0 || body.Top > 0 || body.Right < w || body.Bottom < h) {
		x, y = body.Left, body.Top
		w, h = body.Right - body.Left, body.Bottom - body.Top
	}
	return
}

// backdropBkImgID 模糊快照在元素 BkM 管理器里的固定对象 id。
const backdropBkImgID int32 = 1

// cropRGBA 裁出 [x0,y0)-(x1,y1) 的**独立紧凑副本**(越界或空区域返回原图)。
//
// 必须拷贝成独立图, 不能用 SubImage: 子图与原图共享像素数组、行距仍是原图宽度, 而模糊/
// 放大都按"行距 = 宽×4"索引, 拿子图去模糊会把整幅图斜切错位成斜粗条纹(实测踩坑)。
func cropRGBA(src *image.RGBA, x0, y0, x1, y1 int) *image.RGBA {
	if src == nil {
		return nil
	}
	w, h := src.Rect.Dx(), src.Rect.Dy()
	if x0 < 0 || y0 < 0 || x1 > w || y1 > h || x1 <= x0 || y1 <= y0 {
		return src
	}
	out := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
	for y := y0; y < y1; y++ {
		copy(out.Pix[(y-y0)*(x1-x0)*4:(y-y0+1)*(x1-x0)*4], src.Pix[(y*w+x0)*4:(y*w+x1)*4])
	}
	return out
}

// upscaleRGBA 把 RGBA 图放大到 dstW×dstH: 每个目标像素取源图上对应邻域(放大倍率 ≈2 时
// 正好是 2×2)的平均值 —— 只用加法与移位, 不引入浮点, 也不会像最近邻那样留方块。
// 用途: 把"在 1/backdropBlurDown 分辨率上做的模糊"还原成元素的**精确**逻辑尺寸。
func upscaleRGBA(src *image.RGBA, dstW, dstH int) *image.RGBA {
	if src == nil {
		return nil
	}
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	if sw <= 0 || sh <= 0 || dstW <= 0 || dstH <= 0 || (sw == dstW && sh == dstH) {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	// 列映射预计算一次(源邻域的左右两条列), 内层循环只剩加法
	xs0 := make([]int32, dstW)
	xs1 := make([]int32, dstW)
	for dx := 0; dx < dstW; dx++ {
		x0 := dx * sw / dstW
		x1 := x0 + 1
		if x1 > sw-1 {
			x1 = sw - 1
		}
		xs0[dx], xs1[dx] = int32(x0)*4, int32(x1)*4
	}
	for dy := 0; dy < dstH; dy++ {
		y0 := dy * sh / dstH
		y1 := y0 + 1
		if y1 > sh-1 {
			y1 = sh - 1
		}
		r0 := y0 * sw * 4
		r1 := y1 * sw * 4
		drow := dy * dstW * 4
		for dx := 0; dx < dstW; dx++ {
			a := r0 + int(xs0[dx])
			b := r0 + int(xs1[dx])
			c := r1 + int(xs0[dx])
			d := r1 + int(xs1[dx])
			j := drow + dx*4
			dst.Pix[j+0] = byte((int(src.Pix[a]) + int(src.Pix[b]) + int(src.Pix[c]) + int(src.Pix[d])) >> 2)
			dst.Pix[j+1] = byte((int(src.Pix[a+1]) + int(src.Pix[b+1]) + int(src.Pix[c+1]) + int(src.Pix[d+1])) >> 2)
			dst.Pix[j+2] = byte((int(src.Pix[a+2]) + int(src.Pix[b+2]) + int(src.Pix[c+2]) + int(src.Pix[d+2])) >> 2)
			dst.Pix[j+3] = byte((int(src.Pix[a+3]) + int(src.Pix[b+3]) + int(src.Pix[c+3]) + int(src.Pix[d+3])) >> 2)
		}
	}
	return dst
}

// captureWindowScaled 用 PrintWindow(PW_RENDERFULLCONTENT) 按**物理像素**抓整窗, 并在
// 同一次遍历里把 BGRA 转成 RGBA、盒式降采样到 dstW×dstH。
//
// 两个硬事实(均实测, 见 DESIGN 第八节 47 条 6):
//   - DIB **必须给足窗口的物理像素尺寸**: PrintWindow 是按 1:1 物理像素把整窗画进目标 DC
//     并**裁剪**到 DIB 尺寸, **不会缩放填满**(1200×885 的窗口抓进 800×590 的 DIB, 逐点等于
//     基准图左上 800×590 的物理像素, 均值差 0.0)。按逻辑尺寸要图 = 只剩左上角一块。
//   - 想要小图只能自己降采样, 但这一趟本来就要读遍全部源像素(全屏物理 2560×1528 = 3.9M),
//     顺手把通道交换与盒式平均做掉, 省一整遍全尺寸遍历(实测只多花 ~2ms)。
//
// 失败(含 PrintWindow 返回 0)时返回 nil 让调用方放弃模糊 —— 不要退 BitBlt: 对 D2D 自绘
// 窗口, 从窗口 DC BitBlt 拿到的是**纯黑**(实测整幅 mean=0.0/255), 糊上去就是一块黑板。
func captureWindowScaled(hwnd uintptr, physW, physH, dstW, dstH int) *image.RGBA {
	if hwnd == 0 || physW <= 0 || physH <= 0 || dstW <= 0 || dstH <= 0 {
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

	bi := bitmapinfoheader{Size: 40, Width: int32(physW), Height: int32(-physH), Planes: 1, BitCount: 32}
	var bits unsafe.Pointer
	hbm, _, _ := procCreateDIBSection.Call(hdcMem, uintptr(unsafe.Pointer(&bi)), 0,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 || bits == nil {
		return nil
	}
	defer procDeleteObject.Call(hbm)
	old, _, _ := procSelectObject.Call(hdcMem, hbm)
	defer procSelectObject.Call(hdcMem, old)

	if r, _, _ := procPrintWindow.Call(hwnd, hdcMem, 2); r == 0 { // PW_RENDERFULLCONTENT
		return nil
	}

	src := unsafe.Slice((*byte)(bits), physW*physH*4)
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for dy := 0; dy < dstH; dy++ {
		sy0, sy1 := dy*physH/dstH, (dy+1)*physH/dstH
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		row := dy * dstW * 4
		for dx := 0; dx < dstW; dx++ {
			sx0, sx1 := dx*physW/dstW, (dx+1)*physW/dstW
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var sum [3]int
			for sy := sy0; sy < sy1; sy++ {
				base := sy * physW * 4
				for sx := sx0; sx < sx1; sx++ {
					i := base + sx*4
					sum[0] += int(src[i+2]) // DIB 是 BGRA → RGBA
					sum[1] += int(src[i+1])
					sum[2] += int(src[i])
				}
			}
			n := (sx1 - sx0) * (sy1 - sy0)
			j := row + dx*4
			dst.Pix[j+0] = byte(sum[0] / n)
			dst.Pix[j+1] = byte(sum[1] / n)
			dst.Pix[j+2] = byte(sum[2] / n)
			dst.Pix[j+3] = 255
		}
	}
	return dst
}

// boxBlurRGBA 盒式模糊两轮(横向+纵向 ×2), 视觉上足够接近高斯。
// 半径单位是**传入图像自身的像素** —— 快照管线传进来的是 1/backdropBlurDown 逻辑分辨率,
// 所以半径用 backdropBlurPx(见常量注释), 不要再乘 DPI。
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
