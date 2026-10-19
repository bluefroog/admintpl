// Package cron 提供 cron 表达式解析与时间计算。
//
// 表达式格式与若依（Quartz）保持一致，共 6~7 段：秒 分 时 日 月 周 [年]
//
//	0/20 * * * * ?          每 20 秒执行一次
//	0 0 2 1 * ?             每月 1 日凌晨 2 点执行
//	0 15 10 ? * MON-FRI     周一到周五 10:15:00 执行
//	0 0 12 L * ?            每月最后一天 12:00:00 执行
//	0 0 12 15W * ?          每月最接近 15 号的周一到周五执行
//	0 0 12 ? * 6L           每月最后一个周五 12:00:00 执行
//	0 0 12 ? * 6#3          每月第三个周五 12:00:00 执行
//
// 与若依（Quartz）一致的约定：
//   - 日（日）与 周 两者都指定时取「或」关系（Quartz 语义），未使用的一侧建议写 ?；
//   - 周字段取值 1~7 依次表示周日~周六（与 Quartz 一致），同时兼容 0 表示周日；
//   - 不支持 Quartz 的时区/夏令时特殊处理，统一使用进程本地时区。
//
// 支持的特殊字符：* ? , - / L W #
package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 字段下标
const (
	fieldSecond     = iota // 秒 0-59
	fieldMinute            // 分 0-59
	fieldHour              // 时 0-23
	fieldDayOfMonth        // 日 1-31
	fieldMonth             // 月 1-12
	fieldDayOfWeek         // 周 1-7（1=周日）
	fieldYear              // 年（可选）
)

// fieldNames 字段中文名，仅用于错误提示
var fieldNames = []string{"秒", "分", "时", "日", "月", "周", "年"}

// 各字段取值范围
var fieldRanges = [][2]int{
	{0, 59}, {0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}, {1970, 2099},
}

// 月份/星期英文缩写
var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

var weekNames = map[string]int{
	"SUN": 1, "MON": 2, "TUE": 3, "WED": 4, "THU": 5, "FRI": 6, "SAT": 7,
}

// field 单个字段的解析结果
type field struct {
	raw     string
	bits    uint64       // 允许取值位图（周字段存 Go 的 0-6）
	years   map[int]bool // 年字段取值（1970~2099 超出 64 位，单独用集合保存）
	any     bool         // * 或 ?：不限制
	last    bool         // 日：L / L-n / LW；周：nL（当月最后一个该星期几）
	lastOff int          // L-n 中的 n
	nearest bool         // 日：nW / LW：最接近的周一到周五
	nth     int          // 周：n#k 中的 k
	weekday int          // 周：nL / n#k 中的 n（已归一化为 Go 的 0-6）
}

// Schedule cron 表达式
type Schedule struct {
	expr   string
	fields [7]*field
}

// String 返回原始表达式
func (s *Schedule) String() string {
	if s == nil {
		return ""
	}
	return s.expr
}

// Parse 解析 cron 表达式，支持 6 段（秒 分 时 日 月 周）与 7 段（含年）
func Parse(expr string) (*Schedule, error) {
	items := strings.Fields(expr)
	if len(items) != 6 && len(items) != 7 {
		return nil, fmt.Errorf("cron 表达式必须为 6~7 段（秒 分 时 日 月 周 [年]），当前 %d 段：%s", len(items), expr)
	}
	s := &Schedule{expr: strings.Join(items, " ")}
	for i := 0; i < len(items); i++ {
		f, err := parseField(i, items[i])
		if err != nil {
			return nil, err
		}
		s.fields[i] = f
	}
	if len(items) == 6 {
		// 未指定年时使用 * 占位（若依表达式通常不带年）
		s.fields[fieldYear] = &field{raw: "*", any: true}
	}
	// 与 Quartz 一致：日与周必须有一侧指定取值，不能同时写 ?
	if strings.TrimSpace(items[fieldDayOfMonth]) == "?" && strings.TrimSpace(items[fieldDayOfWeek]) == "?" {
		return nil, fmt.Errorf("cron 表达式的「日」与「周」不能同时为 ?：%s", expr)
	}
	return s, nil
}

// Validate 校验表达式是否合法
func Validate(expr string) error {
	_, err := Parse(expr)
	return err
}

// parseField 解析单个字段
func parseField(idx int, raw string) (*field, error) {
	fieldName := fieldNames[idx]
	f := &field{raw: raw}
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return nil, fmt.Errorf("%s字段不能为空：%s", fieldName, raw)
	}
	if value == "*" || value == "?" {
		if value == "?" && idx != fieldDayOfMonth && idx != fieldDayOfWeek {
			return nil, fmt.Errorf("%s字段不支持 ?，请使用 *：%s", fieldName, raw)
		}
		f.any = true
		return f, nil
	}
	if strings.Contains(value, "?") {
		return nil, fmt.Errorf("%s字段的 ? 只能单独使用：%s", fieldName, raw)
	}
	// L / # 表示「最后一个」「第几个」，与其它取值混用没有明确语义，直接拒绝
	if (idx == fieldDayOfMonth || idx == fieldDayOfWeek) && strings.Contains(value, ",") &&
		strings.ContainsAny(value, "LW#") {
		return nil, fmt.Errorf("%s字段的 L/W/# 不能与其它取值组合使用：%s", fieldName, raw)
	}
	for _, item := range strings.Split(value, ",") {
		if item == "" {
			return nil, fmt.Errorf("%s字段存在空取值：%s", fieldName, raw)
		}
		if err := f.parseItem(idx, item); err != nil {
			return nil, err
		}
	}
	if f.bits == 0 && !f.last && !f.nearest && f.nth == 0 && len(f.years) == 0 {
		return nil, fmt.Errorf("%s字段没有有效取值：%s", fieldName, raw)
	}
	return f, nil
}

// parseItem 解析字段中的一个取值项
func (f *field) parseItem(idx int, item string) error {
	fieldName := fieldNames[idx]
	min, max := fieldRanges[idx][0], fieldRanges[idx][1]

	// 特殊字符：仅「日」与「周」支持
	switch {
	case idx == fieldDayOfMonth && item == "L":
		f.last = true
		return nil
	case idx == fieldDayOfMonth && item == "LW":
		// 当月最后一个工作日
		f.last, f.nearest = true, true
		return nil
	case idx == fieldDayOfMonth && strings.HasPrefix(item, "L-"):
		n, err := strconv.Atoi(item[2:])
		if err != nil || n < 0 || n > 30 {
			return fmt.Errorf("日字段的 %s 取值不合法", item)
		}
		f.last, f.lastOff = true, n
		return nil
	case idx == fieldDayOfMonth && strings.HasSuffix(item, "W"):
		day, err := strconv.Atoi(strings.TrimSuffix(item, "W"))
		if err != nil || day < min || day > max {
			return fmt.Errorf("日字段的 %s 取值不合法", item)
		}
		f.bits |= 1 << uint(day)
		f.nearest = true
		return nil
	case idx == fieldDayOfWeek && item == "L":
		f.last, f.weekday = true, goWeekday(7)
		return nil
	case idx == fieldDayOfWeek && strings.HasSuffix(item, "L"):
		n, err := parseValue(fieldDayOfWeek, strings.TrimSuffix(item, "L"))
		if err != nil {
			return err
		}
		f.last, f.weekday = true, goWeekday(n)
		return nil
	case idx == fieldDayOfWeek && strings.Contains(item, "#"):
		parts := strings.Split(item, "#")
		if len(parts) != 2 {
			return fmt.Errorf("周字段的 %s 取值不合法", item)
		}
		n, err := parseValue(fieldDayOfWeek, parts[0])
		if err != nil {
			return err
		}
		k, err := strconv.Atoi(parts[1])
		if err != nil || k < 1 || k > 5 {
			return fmt.Errorf("周字段 # 后的序号只能是 1~5：%s", item)
		}
		f.nth, f.weekday = k, goWeekday(n)
		return nil
	case strings.ContainsAny(item, "LW#"):
		return fmt.Errorf("%s字段不支持 %s", fieldName, item)
	}

	// 普通取值：n、a-b、*/c、a/c、a-b/c
	step := 1
	base := item
	if i := strings.Index(item, "/"); i >= 0 {
		base = item[:i]
		stepValue, err := strconv.Atoi(item[i+1:])
		if err != nil || stepValue <= 0 {
			return fmt.Errorf("%s字段的步长不合法：%s", fieldName, item)
		}
		step = stepValue
	}
	var start, end int
	switch {
	case base == "*" || base == "?":
		start, end = min, max
	case strings.Contains(base, "-"):
		parts := strings.SplitN(base, "-", 2)
		var err error
		if start, err = parseValue(idx, parts[0]); err != nil {
			return err
		}
		if end, err = parseValue(idx, parts[1]); err != nil {
			return err
		}
		if start > end {
			return fmt.Errorf("%s字段的区间不合法：%s", fieldName, item)
		}
	default:
		value, err := parseValue(idx, base)
		if err != nil {
			return err
		}
		start = value
		// 起始值 + 步长 时按「起始值~最大值」展开（如 5/10）
		if strings.Contains(item, "/") {
			end = max
		} else {
			end = value
		}
	}
	for v := start; v <= end; v += step {
		if idx == fieldYear {
			if f.years == nil {
				f.years = make(map[int]bool)
			}
			f.years[v] = true
			continue
		}
		if idx == fieldDayOfWeek {
			// 周字段位图统一按 Go 的 0-6（0=周日）存放
			f.bits |= 1 << uint(goWeekday(v))
			continue
		}
		f.bits |= 1 << uint(v)
	}
	return nil
}

// parseValue 解析单个数值，支持月份/星期英文缩写
func parseValue(idx int, value string) (int, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return 0, fmt.Errorf("%s字段存在空取值", fieldNames[idx])
	}
	if idx == fieldMonth {
		if n, ok := monthNames[value]; ok {
			return n, nil
		}
	}
	if idx == fieldDayOfWeek {
		if n, ok := weekNames[value]; ok {
			return n, nil
		}
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s字段的取值不合法：%s", fieldNames[idx], value)
	}
	min, max := fieldRanges[idx][0], fieldRanges[idx][1]
	if n < min || n > max {
		return 0, fmt.Errorf("%s字段的取值必须在 %d~%d 之间：%s", fieldNames[idx], min, max, value)
	}
	return n, nil
}

// goWeekday 将 Quartz 的 1~7（1=周日 7=周六，另兼容 0=周日）转换成 Go 的 0~6（0=周日）
func goWeekday(quartzValue int) int {
	if quartzValue <= 1 {
		return 0
	}
	return quartzValue - 1
}

// Match 判断 time 是否命中表达式（精确到秒）
func (s *Schedule) Match(t time.Time) bool {
	if s == nil {
		return false
	}
	if !s.match(t, fieldSecond, t.Second()) || !s.match(t, fieldMinute, t.Minute()) || !s.match(t, fieldHour, t.Hour()) {
		return false
	}
	if !s.matchYear(t) {
		return false
	}
	if !s.matchMonth(t) {
		return false
	}
	return s.matchDay(t)
}

// match 通用位图匹配
func (s *Schedule) match(t time.Time, idx, value int) bool {
	f := s.fields[idx]
	if f == nil || f.any {
		return true
	}
	return f.bits&(1<<uint(value)) != 0
}

// matchYear 年字段匹配
func (s *Schedule) matchYear(t time.Time) bool {
	f := s.fields[fieldYear]
	if f == nil || f.any {
		return true
	}
	return f.years[t.Year()]
}

// matchMonth 月字段匹配
func (s *Schedule) matchMonth(t time.Time) bool {
	return s.match(t, fieldMonth, int(t.Month()))
}

// matchDay 日匹配：日与周都指定时取「或」关系（与 Quartz 一致）
func (s *Schedule) matchDay(t time.Time) bool {
	dom, dow := s.fields[fieldDayOfMonth], s.fields[fieldDayOfWeek]
	domMatch := dom.any || dom.matchDayOfMonth(t)
	dowMatch := dow.any || dow.matchDayOfWeek(t)
	switch {
	case dom.any && dow.any:
		return true
	case dom.any:
		return dowMatch
	case dow.any:
		return domMatch
	default:
		return domMatch || dowMatch
	}
}

// matchDayOfMonth 日字段匹配（含 L / L-n / LW / nW）
func (f *field) matchDayOfMonth(t time.Time) bool {
	year, month, day := t.Date()
	if f.last {
		target := lastDayOfMonth(year, month)
		if f.nearest { // LW：当月最后一个工作日
			return day == nearestWeekday(year, month, target)
		}
		return day == target-f.lastOff
	}
	if f.nearest {
		for d := 1; d <= 31; d++ {
			if f.bits&(1<<uint(d)) == 0 {
				continue
			}
			if d > lastDayOfMonth(year, month) {
				continue
			}
			if day == nearestWeekday(year, month, d) {
				return true
			}
		}
		return false
	}
	return f.bits&(1<<uint(day)) != 0
}

// matchDayOfWeek 周字段匹配（含 nL / n#k）
func (f *field) matchDayOfWeek(t time.Time) bool {
	weekday := int(t.Weekday())
	if f.nth > 0 { // n#k：当月第 k 个星期 n
		return weekday == f.weekday && (t.Day()-1)/7+1 == f.nth
	}
	if f.last { // nL：当月最后一个星期 n
		return weekday == f.weekday && t.AddDate(0, 0, 7).Month() != t.Month()
	}
	return f.bits&(1<<uint(weekday)) != 0
}

// lastDayOfMonth 当月最后一天
func lastDayOfMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

// nearestWeekday 返回与 day 最接近的周一到周五（跳过周末）
func nearestWeekday(year int, month time.Month, day int) int {
	target := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	switch target.Weekday() {
	case time.Saturday:
		if day == 1 { // 1 号是周六，顺延到周一
			return 3
		}
		return day - 1
	case time.Sunday:
		if day == lastDayOfMonth(year, month) { // 月末是周日，提前到周五
			return day - 2
		}
		return day + 1
	default:
		return day
	}
}

// Next 返回 strict 之后第一个命中时间；8 年内都找不到时返回零值
func (s *Schedule) Next(strict time.Time) time.Time {
	if s == nil {
		return time.Time{}
	}
	cursor := truncateToSecond(strict).Add(time.Second)
	limit := strict.AddDate(8, 0, 0)
	for cursor.Before(limit) {
		if !s.matchYear(cursor) {
			cursor = time.Date(cursor.Year()+1, 1, 1, 0, 0, 0, 0, cursor.Location())
			continue
		}
		if !s.matchMonth(cursor) {
			cursor = time.Date(cursor.Year(), cursor.Month(), 1, 0, 0, 0, 0, cursor.Location()).AddDate(0, 1, 0)
			continue
		}
		if !s.matchDay(cursor) {
			cursor = time.Date(cursor.Year(), cursor.Month(), cursor.Day(), 0, 0, 0, 0, cursor.Location()).AddDate(0, 0, 1)
			continue
		}
		if !s.match(cursor, fieldHour, cursor.Hour()) {
			cursor = time.Date(cursor.Year(), cursor.Month(), cursor.Day(), cursor.Hour(), 0, 0, 0, cursor.Location()).Add(time.Hour)
			continue
		}
		if !s.match(cursor, fieldMinute, cursor.Minute()) {
			cursor = time.Date(cursor.Year(), cursor.Month(), cursor.Day(), cursor.Hour(), cursor.Minute(), 0, 0, cursor.Location()).Add(time.Minute)
			continue
		}
		if !s.match(cursor, fieldSecond, cursor.Second()) {
			cursor = cursor.Add(time.Second)
			continue
		}
		return cursor
	}
	return time.Time{}
}

// Prev 返回 strict 之前最近一个命中时间；8 年内都找不到时返回零值
func (s *Schedule) Prev(strict time.Time) time.Time {
	if s == nil {
		return time.Time{}
	}
	cursor := truncateToSecond(strict).Add(-time.Second)
	limit := strict.AddDate(-8, 0, 0)
	for cursor.After(limit) {
		if !s.matchYear(cursor) {
			cursor = time.Date(cursor.Year(), 1, 1, 0, 0, 0, 0, cursor.Location()).Add(-time.Second)
			continue
		}
		if !s.matchMonth(cursor) {
			cursor = time.Date(cursor.Year(), cursor.Month(), 1, 0, 0, 0, 0, cursor.Location()).Add(-time.Second)
			continue
		}
		if !s.matchDay(cursor) {
			cursor = time.Date(cursor.Year(), cursor.Month(), cursor.Day(), 0, 0, 0, 0, cursor.Location()).Add(-time.Second)
			continue
		}
		if !s.match(cursor, fieldHour, cursor.Hour()) {
			cursor = time.Date(cursor.Year(), cursor.Month(), cursor.Day(), cursor.Hour(), 0, 0, 0, cursor.Location()).Add(-time.Second)
			continue
		}
		if !s.match(cursor, fieldMinute, cursor.Minute()) {
			cursor = time.Date(cursor.Year(), cursor.Month(), cursor.Day(), cursor.Hour(), cursor.Minute(), 0, 0, cursor.Location()).Add(-time.Second)
			continue
		}
		if !s.match(cursor, fieldSecond, cursor.Second()) {
			cursor = cursor.Add(-time.Second)
			continue
		}
		return cursor
	}
	return time.Time{}
}

// truncateToSecond 截断到秒（保持本地时区）
func truncateToSecond(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}
