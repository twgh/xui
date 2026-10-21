package ui

import (
	"testing"
	"time"
)

func testDate(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

func TestCalNormalizeRange(t *testing.T) {
	a, b := testDate("2026-01-20"), testDate("2026-02-09")
	if from, to := calNormalizeRange(a, b); !from.Equal(a) || !to.Equal(b) {
		t.Fatalf("forward range = %v..%v, want %v..%v", from, to, a, b)
	}
	if from, to := calNormalizeRange(b, a); !from.Equal(a) || !to.Equal(b) {
		t.Fatalf("reversed range = %v..%v, want %v..%v", from, to, a, b)
	}
	if from, to := calNormalizeRange(a, time.Time{}); !from.Equal(a) || !to.IsZero() {
		t.Fatalf("open range = %v..%v, want %v..zero", from, to, a)
	}
}

func TestCalRangeAfterClick(t *testing.T) {
	d1, d2, d0 := testDate("2026-01-10"), testDate("2026-01-20"), testDate("2026-01-01")

	from, to := calRangeAfterClick(time.Time{}, time.Time{}, d1)
	if !from.Equal(d1) || !to.IsZero() {
		t.Fatalf("first click = %v..%v, want %v..zero", from, to, d1)
	}

	from, to = calRangeAfterClick(from, to, d2)
	if !from.Equal(d1) || !to.Equal(d2) {
		t.Fatalf("second click = %v..%v, want %v..%v", from, to, d1, d2)
	}

	from, to = calRangeAfterClick(d1, time.Time{}, d0)
	if !from.Equal(d0) || !to.Equal(d1) {
		t.Fatalf("backward click = %v..%v, want %v..%v", from, to, d0, d1)
	}

	from, to = calRangeAfterClick(d1, d2, d0)
	if !from.Equal(d0) || !to.IsZero() {
		t.Fatalf("restart click = %v..%v, want %v..zero", from, to, d0)
	}
}

func TestCalWeekNumber(t *testing.T) {
	cases := []struct {
		day   time.Time
		start time.Weekday
		want  int
	}{
		{testDate("2025-12-29"), time.Monday, 1},  // 2026 第 1 个 ISO 周
		{testDate("2026-12-28"), time.Monday, 53}, // 2026 有 53 个 ISO 周
		{testDate("2026-01-04"), time.Sunday, 2},  // 周日开头的行含 1/5(周一, 第 2 周)
		{testDate("2026-10-19"), time.Monday, 43},
	}
	for _, c := range cases {
		if got := calWeekNumber(c.day, c.start); got != c.want {
			t.Errorf("calWeekNumber(%s, %v) = %d, want %d", c.day.Format("2006-01-02"), c.start, got, c.want)
		}
	}
}

func TestCalMetrics(t *testing.T) {
	if got := CalWidthFor(1); got != 278 {
		t.Errorf("CalWidthFor(1) = %d, want 278", got)
	}
	if got := CalWidthFor(2); got != 547 {
		t.Errorf("CalWidthFor(2) = %d, want 547", got)
	}
	if got := CalHeightFor(CalWidth()); got != 325 {
		t.Errorf("CalHeightFor(278) = %d, want 325", got)
	}
	if got := CalHeightForMonths(CalWidthFor(2), 2); got != 319 {
		t.Errorf("CalHeightForMonths(547, 2) = %d, want 319", got)
	}
	single := calMetricsFor(CalWidth(), 1, false)
	if single.cell != 36 {
		t.Errorf("single cell = %d, want 36", single.cell)
	}
	week := calMetricsFor(calDefaultWidth(1, true), 1, true)
	if week.weekW != week.cell || week.monthW != 8*week.cell {
		t.Errorf("week-number metrics = monthW %d / cell %d / weekW %d, want 8*cell / cell", week.monthW, week.cell, week.weekW)
	}
}

func TestCalDateAtMonth(t *testing.T) {
	c := &Calendar{
		view:      time.Date(2026, time.February, 1, 0, 0, 0, 0, time.Local),
		weekStart: time.Monday,
		months:    2,
	}
	// 2026-02-01 是周日, 周一开头时网格起点回退到 1/26
	if got := c.dateAtMonth(0, 0); !got.Equal(testDate("2026-01-26")) {
		t.Errorf("first cell = %s, want 2026-01-26", got.Format("2006-01-02"))
	}
	if got := c.dateAtMonth(0, 6); !got.Equal(testDate("2026-02-01")) {
		t.Errorf("cell 6 = %s, want 2026-02-01", got.Format("2006-01-02"))
	}
	if got := c.monthAt(1); !got.Equal(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.Local)) {
		t.Errorf("monthAt(1) = %s, want 2026-03", got.Format("2006-01"))
	}
}

func TestParseDateText(t *testing.T) {
	want := testDate("2026-10-22")
	for _, s := range []string{"2026-10-22", "2026/10/22", "2026.10.22", "2026年10月22日", " 2026年10月22日 "} {
		got, ok := ParseDateText(s)
		if !ok || !got.Equal(want) {
			t.Errorf("ParseDateText(%q) = %v, %v; want %v, true", s, got, ok, want)
		}
	}

	today := calDay(time.Now())
	cases := map[string]time.Time{
		"今天":  today,
		"明天":  today.AddDate(0, 0, 1),
		"后天":  today.AddDate(0, 0, 2),
		"大后天": today.AddDate(0, 0, 3),
		"3天后": today.AddDate(0, 0, 3),
		"2周后": today.AddDate(0, 0, 14),
	}
	for s, want := range cases {
		got, ok := ParseDateText(s)
		if !ok || !got.Equal(want) {
			t.Errorf("ParseDateText(%q) = %v, %v; want %v, true", s, got, ok, want)
		}
	}
	if _, ok := ParseDateText("not-a-date"); ok {
		t.Error("ParseDateText(not-a-date) should fail")
	}
}

func TestCalHeadKind(t *testing.T) {
	// headerHit 的编码是 块*2 + 0 年/1 月; 直接当 calDrop* 用会把"年"(0)当成
	// calDropNone(点年份无效)、把"月"(1)当成 calDropYear(点月份弹出年份)。
	if got := calHeadKind(0); got != calDropYear {
		t.Errorf("calHeadKind(0) = %d, want calDropYear(%d)", got, calDropYear)
	}
	if got := calHeadKind(1); got != calDropMonth {
		t.Errorf("calHeadKind(1) = %d, want calDropMonth(%d)", got, calDropMonth)
	}
	if got := calHeadKind(2); got != calDropYear {
		t.Errorf("calHeadKind(2) = %d, want calDropYear", got)
	}
	if got := calHeadKind(3); got != calDropMonth {
		t.Errorf("calHeadKind(3) = %d, want calDropMonth", got)
	}
}

func TestDatePickerText(t *testing.T) {
	d := &DatePicker{value: testDate("2026-10-22")}
	if got := d.Text(); got != "2026年10月22日" {
		t.Errorf("single text = %q", got)
	}
	d = &DatePicker{rangeMode: true, rangeFrom: testDate("2026-01-20"), rangeTo: testDate("2026-02-09")}
	if got := d.Text(); got != "2026年1月20日 - 2026年2月9日" {
		t.Errorf("range text = %q", got)
	}
	d.rangeTo = time.Time{}
	if got := d.Text(); got != "2026年1月20日" {
		t.Errorf("open range text = %q", got)
	}
	if got := (&DatePicker{}).Text(); got != "" {
		t.Errorf("empty text = %q", got)
	}
}
