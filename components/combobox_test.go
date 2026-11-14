package ui

import "testing"

// Combobox 的弹层只 Hide 不销毁, 行元素与 r.active 会跨开合保留。
// paintRow 看的是 r.active, 只重置 c.active 下标不够: 上一次高亮的行仍是 true,
// 下一次 applyFilter 又点亮第一行, 弹层里就会同时出现两条 bg-accent。
// 用不带元素的纯状态机覆盖: applyFilter 每次都必须收敛到恰好一条高亮。

func comboboxTestRows(groups []SelectGroup) []*comboboxRow {
	var rows []*comboboxRow
	for gi, g := range groups {
		for _, it := range g.Items {
			rows = append(rows, &comboboxRow{
				kind: comboboxRowItem, item: it, group: int32(gi), shown: true,
			})
		}
	}
	return rows
}

func assertOnlyRowActive(t *testing.T, c *Combobox, want int32) {
	t.Helper()
	if c.active != want {
		t.Fatalf("c.active = %d, want %d", c.active, want)
	}
	for i, r := range c.rows {
		if r.active != (int32(i) == want) {
			t.Fatalf("row %d active = %v, want only row %d active", i, r.active, want)
		}
	}
}

func TestComboboxApplyFilterClearsStaleActive(t *testing.T) {
	groups := []SelectGroup{{Items: []SelectItem{
		{Value: "a", Text: "Next.js"},
		{Value: "b", Text: "SvelteKit"},
		{Value: "c", Text: "Nuxt.js"},
	}}}
	c := &Combobox{
		groups: groups, rows: comboboxTestRows(groups),
		picked: map[string]bool{}, built: true, active: -1,
	}

	c.applyFilter() // 第一次打开: 高亮第一条
	assertOnlyRowActive(t, c, 0)

	c.setActive(1, false) // 悬停/选中第二条
	assertOnlyRowActive(t, c, 1)

	c.applyFilter() // 关闭后再次打开: 旧实现会留下第二条的高亮
	assertOnlyRowActive(t, c, 0)
}

func TestComboboxApplyFilterClearsHiddenActive(t *testing.T) {
	groups := []SelectGroup{{Items: []SelectItem{
		{Value: "a", Text: "Next.js"},
		{Value: "b", Text: "SvelteKit"},
		{Value: "c", Text: "Nuxt.js"},
	}}}
	c := &Combobox{
		groups: groups, rows: comboboxTestRows(groups), autoHighlight: true,
		picked: map[string]bool{}, built: true, active: -1,
	}

	c.setActive(2, false) // 高亮第三条
	c.query = "svelte"    // 过滤后第三条被隐藏
	c.applyFilter()
	assertOnlyRowActive(t, c, 1)

	c.query = "" // 清空搜索词, 第三条重新出现时不能带着旧高亮
	c.applyFilter()
	assertOnlyRowActive(t, c, 0)
}

// 打开/清空搜索词时高亮落在当前选中项(同 Select 打开即定位选中项),
// 输入搜索词后落到第一条匹配项(autoHighlight 开), 选中项被过滤掉则回落到第一条。
func TestComboboxInitialActiveFollowsPicked(t *testing.T) {
	groups := []SelectGroup{{Items: []SelectItem{
		{Value: "a", Text: "Next.js"},
		{Value: "b", Text: "SvelteKit"},
		{Value: "c", Text: "Nuxt.js"},
	}}}
	newC := func() *Combobox {
		return &Combobox{
			groups: groups, rows: comboboxTestRows(groups), autoHighlight: true,
			picked: map[string]bool{"c": true}, built: true, active: -1,
		}
	}

	c := newC()
	c.applyFilter() // 已选 Nuxt.js, 打开时高亮第三条而不是第一条
	assertOnlyRowActive(t, c, 2)

	c.query = "svelte" // 输入即过滤: Enter 应选中第一条匹配项
	c.applyFilter()
	assertOnlyRowActive(t, c, 1)

	c.query = "" // 清空搜索词: 回到选中项
	c.applyFilter()
	assertOnlyRowActive(t, c, 2)

	c.query = "next" // 选中项不在结果里: 回落到第一条匹配项
	c.applyFilter()
	assertOnlyRowActive(t, c, 0)

	c.query = "zzz" // 无匹配: 一条高亮都不该有
	c.applyFilter()
	assertOnlyRowActive(t, c, -1)
}

// autoHighlight 关闭(对齐上游 base combobox 默认): 输入过滤后不自动点亮任何行,
// 由 ↓/Home/End 或悬停点亮; 清空搜索词仍回到选中项。
func TestComboboxAutoHighlightOff(t *testing.T) {
	groups := []SelectGroup{{Items: []SelectItem{
		{Value: "a", Text: "Next.js"},
		{Value: "b", Text: "SvelteKit"},
		{Value: "c", Text: "Nuxt.js"},
	}}}
	c := &Combobox{
		groups: groups, rows: comboboxTestRows(groups),
		picked: map[string]bool{"c": true}, built: true, active: -1,
	}

	c.applyFilter() // 打开仍定位选中项(与 autoHighlight 无关)
	assertOnlyRowActive(t, c, 2)

	c.query = "svelte"
	c.applyFilter()
	assertOnlyRowActive(t, c, -1)

	c.moveActive(1) // ↓ 才落到第一条匹配项
	assertOnlyRowActive(t, c, 1)

	c.query = "next"
	c.applyFilter() // 重新过滤: 高亮再次清空
	assertOnlyRowActive(t, c, -1)

	c.query = ""
	c.applyFilter()
	assertOnlyRowActive(t, c, 2)
}

func TestComboboxInitialActiveMultiple(t *testing.T) {
	groups := []SelectGroup{{Items: []SelectItem{
		{Value: "a", Text: "Next.js"},
		{Value: "b", Text: "SvelteKit"},
		{Value: "c", Text: "Nuxt.js"},
	}}}
	c := &Combobox{
		groups: groups, rows: comboboxTestRows(groups), multiple: true,
		picked: map[string]bool{"b": true, "c": true}, built: true, active: -1,
	}
	c.applyFilter() // 多选取数据顺序里第一个已选中的行
	assertOnlyRowActive(t, c, 1)
}
