package xui

import "testing"

// NextPresenceKey 必须单调递增且不重复: 弹层(菜单/下拉框/对话框)共用宿主窗口的键空间,
// 撞键会让 Presence 走"目标不变"的幂等分支 —— 过渡不启动、apply 不执行, 新弹层停在
// alpha 下限(全透明但仍模态吞点击), 见 DESIGN 第八节 70。
func TestNextPresenceKeyUnique(t *testing.T) {
	const n = 200
	seen := make(map[int32]bool, n)
	prev := NextPresenceKey()
	for i := 0; i < n; i++ {
		k := NextPresenceKey()
		if k <= prev {
			t.Fatalf("键未递增: %d → %d", prev, k)
		}
		if seen[k] {
			t.Fatalf("键重复: %d", k)
		}
		seen[k] = true
		prev = k
	}
}
