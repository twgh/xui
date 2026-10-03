package xui

import (
	"sync"

	"github.com/twgh/xcgui/widget"
)

// HoverTracker 为没有原生 hover 状态反馈的元素(纯 Element 载体的组件)
// 跟踪鼠标停留状态。基于 AddEvent_MouseStay/MouseLeave 实现。
//
// 用法: 组件构造时 tracker := xui.NewHoverTracker(btn.Element),
// 绘制时读 tracker.Hover()。回调复用(EventBus), 不占 2000 上限。
type HoverTracker struct {
	mu    sync.RWMutex
	hover bool
}

// NewHoverTracker 开始跟踪 ele 的鼠标进出。
func NewHoverTracker(ele *widget.Element) *HoverTracker {
	t := &HoverTracker{}
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		t.set(true)
		ele.Redraw(false)
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		t.set(false)
		ele.Redraw(false)
		return 0
	})
	return t
}

func (t *HoverTracker) set(v bool) {
	t.mu.Lock()
	t.hover = v
	t.mu.Unlock()
}

// Hover 返回当前是否鼠标停留。
func (t *HoverTracker) Hover() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.hover
}
