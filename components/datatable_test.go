package ui

import (
	"reflect"
	"testing"
)

func TestDataTableViewFilter(t *testing.T) {
	rows := [][]string{
		{"张三", "已支付", "¥1,250.00"},
		{"李四", "待支付", "¥300.00"},
		{"Alice", "已退款", "¥99.00"},
	}
	cases := []struct {
		name   string
		filter string
		cols   []int
		want   []int
	}{
		{"空筛选保留全部", "", nil, []int{0, 1, 2}},
		{"中文子串", "张", nil, []int{0}},
		{"金额子串", "300", nil, []int{1}},
		{"大小写不敏感", "alice", nil, []int{2}},
		{"空白只 trim", "  ", nil, []int{0, 1, 2}},
		{"限定列", "支付", []int{1}, []int{0, 1}},
		{"限定列后不命中其它列", "张三", []int{1}, []int{}},
		{"无匹配", "zzz", nil, []int{}},
	}
	for _, c := range cases {
		got := dataTableView(rows, c.filter, c.cols, -1, DataSortNone, nil)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDataTableViewSort(t *testing.T) {
	rows := [][]string{
		{"张三", "¥1,250.00", "2026-10-03"},
		{"李四", "¥300.00", "2026-10-01"},
		{"王五", "¥99.00", "2026-10-02"},
	}
	asc := dataTableView(rows, "", nil, 1, DataSortAsc, nil)
	if want := []int{2, 1, 0}; !reflect.DeepEqual(asc, want) {
		t.Fatalf("金额升序: got %v, want %v", asc, want)
	}
	desc := dataTableView(rows, "", nil, 1, DataSortDesc, nil)
	if want := []int{0, 1, 2}; !reflect.DeepEqual(desc, want) {
		t.Fatalf("金额降序: got %v, want %v", desc, want)
	}
	dateAsc := dataTableView(rows, "", nil, 2, DataSortAsc, nil)
	if want := []int{1, 2, 0}; !reflect.DeepEqual(dateAsc, want) {
		t.Fatalf("日期升序: got %v, want %v", dateAsc, want)
	}
	none := dataTableView(rows, "", nil, -1, DataSortNone, nil)
	if want := []int{0, 1, 2}; !reflect.DeepEqual(none, want) {
		t.Fatalf("不排序保持原始顺序: got %v, want %v", none, want)
	}
}

func TestDataTableViewSortStableAndCustom(t *testing.T) {
	rows := [][]string{
		{"b", "2"},
		{"a", "2"},
		{"c", "1"},
	}
	got := dataTableView(rows, "", nil, 1, DataSortAsc, nil)
	if want := []int{2, 0, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("同值保持原始顺序(稳定排序): got %v, want %v", got, want)
	}
	// 自定义比较: 按首字母倒序
	cmp := func(a, b string) int {
		switch {
		case a > b:
			return -1
		case a < b:
			return 1
		}
		return 0
	}
	got = dataTableView(rows, "", nil, 0, DataSortAsc, cmp)
	if want := []int{2, 0, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("自定义比较: got %v, want %v", got, want)
	}
}

func TestDataCompareNumericAndNatural(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"¥1,250.00", "¥300.00", 1},
		{"99", "1250", -1},
		{"12%", "9%", 1},
		{"item2", "item10", -1},
		{"item10", "item2", 1},
		{"apple", "Banana", -1},
		{"", "a", 1}, // 空值沉底(升序)
		{"a", "", -1},
		{"2026-10-01", "2026-10-02", -1},
	}
	for _, c := range cases {
		if got := dataCompare(c.a, c.b); got != c.want {
			t.Errorf("dataCompare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestNextDataSortCycle(t *testing.T) {
	col, dir := -1, DataSortNone
	col, dir = nextDataSort(col, dir, 2)
	if col != 2 || dir != DataSortAsc {
		t.Fatalf("首次点击 → 升序, got col=%d dir=%v", col, dir)
	}
	col, dir = nextDataSort(col, dir, 2)
	if col != 2 || dir != DataSortDesc {
		t.Fatalf("第二次点击 → 降序, got col=%d dir=%v", col, dir)
	}
	col, dir = nextDataSort(col, dir, 2)
	if col != -1 || dir != DataSortNone {
		t.Fatalf("第三次点击 → 取消排序, got col=%d dir=%v", col, dir)
	}
	col, dir = nextDataSort(col, dir, 1)
	if col != 1 || dir != DataSortAsc {
		t.Fatalf("换列 → 升序, got col=%d dir=%v", col, dir)
	}
}

func TestDataPageCount(t *testing.T) {
	cases := []struct {
		n, size, want int
	}{
		{0, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{12, 5, 3},
		{5, 0, 5},
	}
	for _, c := range cases {
		if got := dataPageCount(c.n, c.size); got != c.want {
			t.Errorf("dataPageCount(%d, %d) = %d, want %d", c.n, c.size, got, c.want)
		}
	}
}
