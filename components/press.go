package ui

import (
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

// pressShift 是按下位移量(逻辑像素), 对应 shadcn 的 active:translate-y-px。
// 元素的创建高度比视觉高度多这么多, 多出的部分留作位移空间: 元素绘制被裁剪在
// 自身客户区内, 若按下时把底边推出客户区, 越界的 1px 会被切掉(看起来底部被削平),
// 预留后整体位移始终落在客户区内, 与原版"整块下沉"的观感一致。
// 实际渲染在 150% DPI 下会被取整成 1 或 2 物理像素; 文字/图标跟随壳体的实际
// 物理位移(见 onPaint 与 DESIGN 88), 不能让它们各自按逻辑量移动。
const pressShift int32 = 1

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
