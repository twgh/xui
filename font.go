package xui

import (
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
	f := font.NewEX(spec.Family, spec.Size, style)
	if f == nil {
		return 0
	}
	f.EnableAutoDestroy(false) // 缓存持有, 永不销毁
	fonts.cache[spec] = f.Handle
	return f.Handle
}
