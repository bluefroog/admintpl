package cron

import (
	"testing"
	"time"
)

// parseLocal 解析本地时间，便于断言
func parseLocal(t *testing.T, value string) time.Time {
	t.Helper()
	result, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
	if err != nil {
		t.Fatalf("解析时间失败：%v", err)
	}
	return result
}

// TestParseValid 合法表达式解析
func TestParseValid(t *testing.T) {
	expressions := []string{
		"0/20 * * * * ?",
		"0 0 2 1 * ?",
		"0 15 10 ? * MON-FRI",
		"0 0 12 ? * 1",
		"0 0 12 L * ?",
		"0 0 12 LW * ?",
		"0 0 12 15W * ?",
		"0 0 12 ? * 6L",
		"0 0 12 ? * 6#3",
		"0 0 0 1 1 ? 2025",
		"*/5 * * * * *",
		"0 0-30/5 * * * ?",
		"0 0 0 1,15 * ?",
		"0 0 0 ? * SAT,SUN",
		"59 59 23 31 12 ?",
	}
	for _, expr := range expressions {
		if _, err := Parse(expr); err != nil {
			t.Errorf("表达式 %q 应当合法，但报错：%v", expr, err)
		}
	}
}

// TestParseInvalid 非法表达式
func TestParseInvalid(t *testing.T) {
	expressions := []string{
		"",
		"* * * * *",         // 只有 5 段
		"* * * * * ? extra", // 8 段
		"60 * * * * ?",      // 秒越界
		"* 60 * * * ?",      // 分越界
		"* * 24 * * ?",      // 时越界
		"* * * 32 * ?",      // 日越界
		"* * * * 13 ?",      // 月越界
		"* * * * * 8",       // 周越界
		"0 0 0 ? * ?",       // 日与周同时为 ?
		"* * ? * * *",       // ? 用在时字段
		"0 0 0 15-10 * ?",   // 区间倒置
		"0 0 0 * * 1/0",     // 步长为 0
		"0 0 0 * * 1#6",     // # 序号越界
		"0 0 0 * * XL",      // 非法周取值
		"0 0 0 * * 1L,2",    // L 与其它取值混用
		"0 0 0 * * 1#2,3",   // # 与其它取值混用
		"0 0 0 * * 1-2-3",   // 多个短横线
		"0 0 0 1 1 ? 1960",  // 年越界
		"bad * * * * ?",     // 非法秒
		"0 0 0 * * MON#2#3", // 多个 #
	}
	for _, expr := range expressions {
		if _, err := Parse(expr); err == nil {
			t.Errorf("表达式 %q 应当非法，但解析通过", expr)
		}
	}
}

// TestMatchSecondInterval 秒级间隔
func TestMatchSecondInterval(t *testing.T) {
	s, err := Parse("0/20 * * * * ?")
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	cases := map[string]bool{
		"2025-03-18 10:00:00": true,
		"2025-03-18 10:00:20": true,
		"2025-03-18 10:00:40": true,
		"2025-03-18 10:00:41": false,
		"2025-03-18 10:00:59": false,
	}
	for value, expect := range cases {
		if got := s.Match(parseLocal(t, value)); got != expect {
			t.Errorf("%s 匹配结果应为 %v，实际 %v", value, expect, got)
		}
	}
}

// TestMatchMonthDayAndWeek 月/日/周匹配（含 Quartz 的或语义）
func TestMatchMonthDayAndWeek(t *testing.T) {
	// 每月 1 日 02:00:00
	monthly, _ := Parse("0 0 2 1 * ?")
	if !monthly.Match(parseLocal(t, "2025-03-01 02:00:00")) {
		t.Error("每月 1 日 02:00:00 应当命中")
	}
	if monthly.Match(parseLocal(t, "2025-03-02 02:00:00")) {
		t.Error("每月 2 日不应命中")
	}

	// 周一到周五 10:15:00（周字段与日字段都指定时取或关系，这里日用 ?）
	workday, _ := Parse("0 15 10 ? * MON-FRI")
	for _, value := range []string{"2025-03-17 10:15:00", "2025-03-21 10:15:00"} { // 周一、周五
		if !workday.Match(parseLocal(t, value)) {
			t.Errorf("%s 应当命中周一~周五规则", value)
		}
	}
	if workday.Match(parseLocal(t, "2025-03-22 10:15:00")) { // 周六
		t.Error("周六不应命中周一~周五规则")
	}

	// 周字段 1 表示周日（与 Quartz 一致）
	sunday, _ := Parse("0 0 12 ? * 1")
	if !sunday.Match(parseLocal(t, "2025-03-16 12:00:00")) { // 周日
		t.Error("周字段 1 应表示周日")
	}
	if sunday.Match(parseLocal(t, "2025-03-17 12:00:00")) { // 周一
		t.Error("周字段 1 不应命中周一")
	}

	// 日与周同时指定 -> 或关系
	either, _ := Parse("0 0 0 10 * 1")
	if !either.Match(parseLocal(t, "2025-03-10 00:00:00")) { // 10 号（周一）
		t.Error("或关系：命中日期")
	}
	if !either.Match(parseLocal(t, "2025-03-16 00:00:00")) { // 周日
		t.Error("或关系：命中星期")
	}
	if either.Match(parseLocal(t, "2025-03-11 00:00:00")) { // 11 号周二
		t.Error("或关系：日期与星期都不命中时不应匹配")
	}
}

// TestMatchLastAndNearest L / LW / nW / nL / n#k
func TestMatchLastAndNearest(t *testing.T) {
	lastDay, _ := Parse("0 0 12 L * ?")
	if !lastDay.Match(parseLocal(t, "2025-02-28 12:00:00")) {
		t.Error("L 应当命中 2 月最后一天")
	}
	if lastDay.Match(parseLocal(t, "2025-02-27 12:00:00")) {
		t.Error("L 不应命中 2 月 27 日")
	}

	lastDayOffset, _ := Parse("0 0 12 L-2 * ?")
	if !lastDayOffset.Match(parseLocal(t, "2025-02-26 12:00:00")) {
		t.Error("L-2 应当命中 2 月 26 日")
	}

	// 2025-03-15 是周六 -> 最接近的工作日为 3-14（周五）
	nearest, _ := Parse("0 0 12 15W * ?")
	if !nearest.Match(parseLocal(t, "2025-03-14 12:00:00")) {
		t.Error("15W 在 3 月应当命中 14 日（周五）")
	}
	if nearest.Match(parseLocal(t, "2025-03-15 12:00:00")) {
		t.Error("15W 不应命中周六本身")
	}

	lastWorkday, _ := Parse("0 0 12 LW * ?")
	// 2025-03-31 是周一（工作日），也是当月最后一天
	if !lastWorkday.Match(parseLocal(t, "2025-03-31 12:00:00")) {
		t.Error("LW 应当命中 3 月 31 日")
	}

	lastFriday, _ := Parse("0 0 12 ? * 6L")
	if !lastFriday.Match(parseLocal(t, "2025-03-28 12:00:00")) { // 3 月最后一个周五
		t.Error("6L 应当命中 2025-03-28")
	}
	if lastFriday.Match(parseLocal(t, "2025-03-21 12:00:00")) {
		t.Error("6L 不应命中 3 月 21 日")
	}

	thirdFriday, _ := Parse("0 0 12 ? * 6#3")
	if !thirdFriday.Match(parseLocal(t, "2025-03-21 12:00:00")) { // 3 月第三个周五
		t.Error("6#3 应当命中 2025-03-21")
	}
	if thirdFriday.Match(parseLocal(t, "2025-03-28 12:00:00")) {
		t.Error("6#3 不应命中第四个周五")
	}
}

// TestNextAndPrev Next/Prev 计算
func TestNextAndPrev(t *testing.T) {
	s, _ := Parse("0 0 2 1 * ?")
	next := s.Next(parseLocal(t, "2025-03-18 10:00:00"))
	if want := parseLocal(t, "2025-04-01 02:00:00"); !next.Equal(want) {
		t.Errorf("Next 应为 %v，实际 %v", want, next)
	}

	prev := s.Prev(parseLocal(t, "2025-03-18 10:00:00"))
	if want := parseLocal(t, "2025-03-01 02:00:00"); !prev.Equal(want) {
		t.Errorf("Prev 应为 %v，实际 %v", want, prev)
	}

	// 跨年计算
	s2, _ := Parse("0 0 0 1 1 ?")
	if got, want := s2.Next(parseLocal(t, "2025-06-01 00:00:00")), parseLocal(t, "2026-01-01 00:00:00"); !got.Equal(want) {
		t.Errorf("跨年 Next 应为 %v，实际 %v", want, got)
	}

	// 秒级表达式：Next 与 Match 结果一致
	s3, _ := Parse("*/15 * * * * *")
	cursor := parseLocal(t, "2025-03-18 10:00:01")
	for i := 0; i < 5; i++ {
		cursor = s3.Next(cursor)
		if !s3.Match(cursor) {
			t.Fatalf("Next 返回的时间应当命中表达式：%v", cursor)
		}
		if got := cursor.Second(); got != 0 && got != 15 && got != 30 && got != 45 {
			t.Fatalf("秒值应在 15 的整数倍，实际 %d", got)
		}
	}

	// 年字段
	yearly, _ := Parse("0 0 0 1 1 ? 2030")
	if got, want := yearly.Next(parseLocal(t, "2025-01-01 00:00:00")), parseLocal(t, "2030-01-01 00:00:00"); !got.Equal(want) {
		t.Errorf("年字段 Next 应为 %v，实际 %v", want, got)
	}
	if !yearly.Next(parseLocal(t, "2031-01-01 00:00:00")).IsZero() {
		t.Error("超出年字段范围时 Next 应返回零值")
	}
}

// TestMatchSecondFieldRanges 取值范围与步长
func TestMatchSecondFieldRanges(t *testing.T) {
	s, err := Parse("0 0-30/5 * * * ?")
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	for _, minute := range []int{0, 5, 10, 30} {
		value := time.Date(2025, 3, 18, 9, minute, 0, 0, time.Local)
		if !s.Match(value) {
			t.Errorf("分钟 %d 应当命中 0-30/5", minute)
		}
	}
	for _, minute := range []int{31, 35, 59} {
		value := time.Date(2025, 3, 18, 9, minute, 0, 0, time.Local)
		if s.Match(value) {
			t.Errorf("分钟 %d 不应命中 0-30/5", minute)
		}
	}

	// 起始值 + 步长：10/20 -> 10,30,50
	step, _ := Parse("0 10/20 * * * ?")
	for _, minute := range []int{10, 30, 50} {
		if !step.Match(time.Date(2025, 3, 18, 9, minute, 0, 0, time.Local)) {
			t.Errorf("分钟 %d 应当命中 10/20", minute)
		}
	}
	if step.Match(time.Date(2025, 3, 18, 9, 20, 0, 0, time.Local)) {
		t.Error("分钟 20 不应命中 10/20")
	}
}

// TestValidate 校验入口
func TestValidate(t *testing.T) {
	if err := Validate("0 0/5 * * * ?"); err != nil {
		t.Errorf("合法表达式校验失败：%v", err)
	}
	if err := Validate("not a cron"); err == nil {
		t.Error("非法表达式校验应当失败")
	}
}

// TestString 表达式原样返回
func TestString(t *testing.T) {
	s, _ := Parse("  0 0/5 * * * ?  ")
	if got, want := s.String(), "0 0/5 * * * ?"; got != want {
		t.Errorf("String 应为 %q，实际 %q", want, got)
	}
	var empty *Schedule
	if empty.String() != "" {
		t.Error("nil Schedule 的 String 应为空串")
	}
}
