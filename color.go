// Package xui 为 github.com/twgh/xcgui 提供一套 shadcn 风格的自绘控件运行时:
// 设计令牌(主题)管理、绘制助手与状态跟踪。
//
// 组件源码不由本包提供, 而是通过 xui CLI 拷贝到用户项目的 ui/ 目录下,
// 组件在绘制时通过 Current() 实时解析主题令牌, 因此切换主题只需 SetTheme + 重绘。
//
// 图标: 内置 lucide 资源在 github.com/twgh/xui/icons 子包(每图标一个 IconXxx 常量, 另有 Path(name));
// 图标数据统一交给 Painter.Icon —— path 数据与整段 SVG 都吃(自动识别); 想先校验 SVG 用 ParseIconSVG。
//
// 注意: 本模块的 go 指令是 1.18 —— 即使在 Go 1.22+ 工具链下, for range 的循环变量也按旧语义**共享**:
// 在循环里注册回调 / 起 goroutine 时必须先 `v := v` 拷一份, 否则所有回调都取到最后一个值。
package xui

import (
	"fmt"
	"strings"

	"github.com/twgh/xcgui/xc"
)

// Color 是炫彩界面库使用的 ABGR 颜色(底层 uint32, 由 xc.RGBA 生成)。
type Color uint32

// None 表示"不绘制"。组件解析主题时用它表达透明/缺省, 例如 Ghost 按钮的背景。
const None Color = 0

// RGBA 按炫彩颜色布局(r 低位)构造颜色。
func RGBA(r, g, b, a byte) Color {
	return Color(xc.RGBA(r, g, b, a))
}

// Hex 解析 Web/CSS 十六进制颜色("#fafafa"、"fafafa"、"#fff"),
// 透明度默认不透明, 失败时 panic(主题文件是静态数据, 尽早暴露拼写错误)。
func Hex(s string) Color {
	c, err := HexE(s)
	if err != nil {
		panic(fmt.Sprintf("xui: invalid hex color %q: %v", s, err))
	}
	return c
}

// HexE 是 Hex 的错误返回版本。
func HexE(s string) (Color, error) {
	s = strings.TrimPrefix(s, "#")
	var r, g, b byte
	switch len(s) {
	case 3:
		if _, err := fmt.Sscanf(s, "%1x%1x%1x", &r, &g, &b); err != nil {
			return 0, fmt.Errorf("invalid 3-digit hex: %w", err)
		}
		r, g, b = r*17, g*17, b*17 // "fff" -> 0xff
	case 6:
		var ri, gi, bi int
		if _, err := fmt.Sscanf(s, "%2x%2x%2x", &ri, &gi, &bi); err != nil {
			return 0, fmt.Errorf("invalid 6-digit hex: %w", err)
		}
		r, g, b = byte(ri), byte(gi), byte(bi)
	default:
		return 0, fmt.Errorf("hex color must be 3 or 6 digits, got %d", len(s))
	}
	return RGBA(r, g, b, 255), nil
}

// RGBA 分量, 注意炫彩布局是 r 在最低字节。
func (c Color) RGBA() (r, g, b, a byte) {
	return xc.ParseRGBA(uint32(c))
}

// WithAlpha 返回同色但指定透明度的副本。
// ScaleAlpha 在现有透明度上再乘一个系数, 对应 CSS 的"半透明色再加 /30"这类写法:
// 例如暗色令牌 input = 白 15%, `bg-input/30` 的最终 alpha = 15% × 30% = 4.5%。
// (用 WithAlpha 会直接覆盖成 30%, 实测会明显偏亮)
func (c Color) ScaleAlpha(frac float64) Color {
	_, _, _, a := c.RGBA()
	return c.WithAlpha(byte(float64(a) * frac))
}

func (c Color) WithAlpha(a byte) Color {
	r, g, b, _ := c.RGBA()
	return RGBA(r, g, b, a)
}

// Lighten 返回按比例提亮的颜色, amount 取值 0~1。
func (c Color) Lighten(amount float64) Color {
	r, g, b, a := c.RGBA()
	return RGBA(mix(r, 255, amount), mix(g, 255, amount), mix(b, 255, amount), a)
}

// Darken 返回按比例加暗的颜色, amount 取值 0~1。
func (c Color) Darken(amount float64) Color {
	r, g, b, a := c.RGBA()
	return RGBA(mix(r, 0, amount), mix(g, 0, amount), mix(b, 0, amount), a)
}

// IsNone 报告是否为 None(不绘制)。
func (c Color) IsNone() bool { return c == None }

// String 格式化为 "#rrggbb"。
func (c Color) String() string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// Mix 返回与 other 按 t(0~1)插值的颜色, t=0 即原色。
// 对应 CSS color-mix(in oklch, a, b t%) 的近似(线性 sRGB 插值)。
func (c Color) Mix(other Color, t float64) Color {
	r1, g1, b1, a1 := c.RGBA()
	r2, g2, b2, a2 := other.RGBA()
	return RGBA(
		byte(mix(r1, r2, t)),
		byte(mix(g1, g2, t)),
		byte(mix(b1, b2, t)),
		byte(mix(a1, a2, t)),
	)
}

func mix(v byte, target byte, amount float64) byte {
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}
	return byte(float64(v) + float64(int(target)-int(v))*amount)
}
