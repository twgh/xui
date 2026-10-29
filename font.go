package xui

import (
	"math"
	"sync"

	"github.com/twgh/xcgui/font"
	"github.com/twgh/xcgui/xcc"
)

// 字体缓存: 主题切换时旧字体仍被元素引用, 不主动销毁(进程生命周期内少量对象)。
type fontCache struct {
	mu    sync.Mutex
	cache map[FontSpec]int // FontSpec -> 炫彩字体句柄
}

var fonts = &fontCache{cache: map[FontSpec]int{}}

// FontHandle 返回主题字体对应的炫彩字体句柄, 相同规格全局复用。
// 组件在构造或主题回调中调用 SetFont(xui.FontHandle(t.FontDefault))。
//
// 单位换算: xcgui 字体的 size 单位是 pt(磅, 见 font.NewEX 文档), 渲染尺寸 =
// pt×DPI/72 物理像素; 主题令牌的 Size 是逻辑像素(= px×DPI/96 物理)。联立得
// pt = px×72/96(= ×0.75, 与 DPI 无关)。此前把 14 直接当 pt 传入, 144 DPI 下
// 渲染出 18.7 逻辑像素的文字(墨迹实测 26-27 物理像素), 全组件文字放大了 1.33×。
// pt 只能取整, 14px→11pt=14.67px(+0.67), 12/16px 等恰好取整。
func FontHandle(spec FontSpec) int {
	fonts.mu.Lock()
	defer fonts.mu.Unlock()
	if h, ok := fonts.cache[spec]; ok {
		return h
	}
	style := xcc.FontStyle_Regular
	if spec.Bold {
		style |= xcc.FontStyle_Bold
	}
	if spec.Italic {
		style |= xcc.FontStyle_Italic
	}
	// 不要传 FontStyle_Underline: 它在本框架的 D2D 文本路径下不渲染(实测,
	// 见 DESIGN 第八节 110)。需要下划线一律手绘 1px 线。
	sizePt := int32(math.Round(float64(spec.Size) * 72.0 / 96.0))
	var f *font.Font
	// xcgui 会把 9pt 及以下的**常规**字重画成宋体(不论字族, 10pt 起才正常,
	// 9pt 粗体也正常 —— 见 DESIGN 第八节 149)。从字体文件创建同一字族可以
	// 绕开它的字族解析, 字号保持不变; 解析不到文件时回退原路径。
	if sizePt < 10 && !spec.Bold && !spec.Italic {
		if path := fontFileForFamily(spec.Family); path != "" {
			f = font.NewByFile(path, sizePt, style)
		}
	}
	if f == nil {
		f = font.NewEX(spec.Family, sizePt, style)
	}
	if f == nil {
		return 0
	}
	f.EnableAutoDestroy(false) // 缓存持有, 永不销毁
	fonts.cache[spec] = f.Handle
	return f.Handle
}
