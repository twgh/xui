package ui

import "testing"

// kinds 把条目序列压成可读的类型串, 便于断言。
func pagKinds(items []pagItem) string {
	out := make([]rune, 0, len(items))
	for _, it := range items {
		switch it.kind {
		case pagPrev:
			out = append(out, 'p')
		case pagPage:
			out = append(out, 'n')
		case pagEllipsis:
			out = append(out, '.')
		case pagNext:
			out = append(out, 'q')
		}
	}
	return string(out)
}

func TestPaginationSequenceModes(t *testing.T) {
	full := &Pagination{total: 120, pageSize: 10, page: 5}
	if got, want := pagKinds(full.sequence()), "pn.nnn.nq"; got != want {
		t.Fatalf("full mode middle = %q, want %q", got, want)
	}

	// 简洁: 只有页码, 没有上一页/下一页
	simple := &Pagination{total: 120, pageSize: 10, page: 5, mode: pagModeSimple}
	if got, want := pagKinds(simple.sequence()), "n.nnn.n"; got != want {
		t.Fatalf("simple mode = %q, want %q", got, want)
	}

	// 仅导航: 只有上一页/下一页
	nav := &Pagination{total: 120, pageSize: 10, page: 5, mode: pagModeNav}
	if got, want := pagKinds(nav.sequence()), "pq"; got != want {
		t.Fatalf("nav mode = %q, want %q", got, want)
	}

	// 页数不超过折叠阈值时全是页码
	short := &Pagination{total: 30, pageSize: 10, page: 2}
	if got, want := pagKinds(short.sequence()), "pnnnq"; got != want {
		t.Fatalf("short full mode = %q, want %q", got, want)
	}
}

// 折叠的槽位数必须恒定(翻页跨段时整行宽度不变, 点击位置才稳定)。
func TestPaginationSlotCountStable(t *testing.T) {
	g := &Pagination{total: 10000, pageSize: 10}
	for page := 1; page <= g.PageCount(); page++ {
		g.page = page
		if got, want := len(g.sequence()), pagMaxLinks+2; got != want {
			t.Fatalf("page %d slots = %d, want %d", page, got, want)
		}
	}
	// 简洁形态同样恒定(只是去掉了两端导航)
	g.mode = pagModeSimple
	for page := 1; page <= g.PageCount(); page++ {
		g.page = page
		if got, want := len(g.sequence()), pagMaxLinks; got != want {
			t.Fatalf("simple page %d slots = %d, want %d", page, got, want)
		}
	}
}

// clickable 决定键盘能否停在该条目上: 省略号与到边界的导航不可点。
func TestPaginationClickable(t *testing.T) {
	g := &Pagination{total: 120, pageSize: 10, page: 1}
	g.items = g.sequence() // [p][1][2][3][4][5][.][12][q]
	if g.clickable(0) {
		t.Fatal("prev must be disabled on the first page")
	}
	if !g.clickable(len(g.items) - 1) {
		t.Fatal("next must be enabled on the first page")
	}
	for i, it := range g.items {
		if it.kind == pagEllipsis && g.clickable(i) {
			t.Fatalf("ellipsis at %d must not be clickable", i)
		}
	}

	g.page = g.PageCount()
	g.items = g.sequence()
	if g.clickable(len(g.items) - 1) {
		t.Fatal("next must be disabled on the last page")
	}
	if !g.clickable(0) {
		t.Fatal("prev must be enabled on the last page")
	}
}

// 键盘左右移动必须跳过省略号与到边界的导航项(用纯函数 pagSeek 验证)。
func TestPaginationFocusSeek(t *testing.T) {
	g := &Pagination{total: 120, pageSize: 10, page: 5}
	g.items = g.sequence() // [p][1][.][4][5][6][.][12][q]
	ok := func(i int) bool { return g.clickable(i) }

	var visits []int
	for i := 1; i < len(g.items); {
		visits = append(visits, i)
		next := pagSeek(len(g.items), i+1, 1, ok)
		if next < 0 {
			break
		}
		i = next
	}
	want := []int{1, 3, 4, 5, 7, 8}
	if len(visits) != len(want) {
		t.Fatalf("right visits = %v, want %v", visits, want)
	}
	for i := range want {
		if visits[i] != want[i] {
			t.Fatalf("right visits = %v, want %v", visits, want)
		}
	}

	// 反向从最后一个可点项开始, 一路减到第一个页码(跳过省略号)。
	if got := pagSeek(len(g.items), 8, -1, ok); got != 8 {
		t.Fatalf("seek from last = %d, want 8", got)
	}
	if got := pagSeek(len(g.items), 6, -1, ok); got != 5 {
		t.Fatalf("seek left from ellipsis = %d, want 5", got)
	}

	// 第一页时"上一页"不可点: 从 0 起反向找不到任何可点条目。
	g.page = 1
	g.items = g.sequence()
	ok = func(i int) bool { return g.clickable(i) }
	if got := pagSeek(len(g.items), 0, -1, ok); got != -1 {
		t.Fatalf("seek left on first page = %d, want -1", got)
	}
	if got := pagSeek(len(g.items), 9, 1, ok); got != -1 {
		t.Fatalf("seek past end = %d, want -1", got)
	}
}

// 页码变化后焦点下标要是不可点(省略号/越界), normalizeFocus 会把它收敛掉。
func TestPaginationFocusReset(t *testing.T) {
	g := &Pagination{total: 120, pageSize: 10, page: 1}
	g.items = g.sequence()
	g.focus = 0 // 第一页时"上一页"不可点
	g.normalizeFocus()
	if g.focus != -1 {
		t.Fatalf("focus on disabled item = %d, want -1", g.focus)
	}

	// 可点条目与越界下标的行为
	g.focus = 1 // 页码 1, 可点
	g.normalizeFocus()
	if g.focus != 1 {
		t.Fatalf("focus on page item = %d, want 1", g.focus)
	}
	g.focus = 99
	g.normalizeFocus()
	if g.focus != -1 {
		t.Fatalf("focus out of range = %d, want -1", g.focus)
	}
}
