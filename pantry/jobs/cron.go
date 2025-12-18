package jobs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronExpr represents a parsed cron expression.
type CronExpr struct {
	minute     []int // 0-59
	hour       []int // 0-23
	dayOfMonth []int // 1-31
	month      []int // 1-12
	dayOfWeek  []int // 0-6 (Sunday = 0)
	raw        string
}

// ParseCron parses a cron expression.
// Supported formats:
//   - Standard cron: "minute hour day-of-month month day-of-week"
//   - With seconds: "second minute hour day-of-month month day-of-week"
//   - Predefined: @yearly, @annually, @monthly, @weekly, @daily, @midnight, @hourly
//
// Field values:
//   - * (any value)
//   - */n (every n)
//   - n (specific value)
//   - n-m (range)
//   - n,m,o (list)
//   - n-m/s (range with step)
//
// Examples:
//
//	"0 0 * * *"     - Every day at midnight
//	"*/15 * * * *"  - Every 15 minutes
//	"0 9 * * 1-5"   - At 9:00 AM, Monday through Friday
//	"0 0 1 * *"     - At midnight on the 1st of every month
//	"@daily"        - Every day at midnight
func ParseCron(expr string) (*CronExpr, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("cron: empty expression")
	}

	// Handle predefined expressions
	if strings.HasPrefix(expr, "@") {
		return parsePredefined(expr)
	}

	fields := strings.Fields(expr)
	if len(fields) < 5 || len(fields) > 6 {
		return nil, fmt.Errorf("cron: expected 5 or 6 fields, got %d", len(fields))
	}

	// Handle 6-field format (with seconds) by ignoring seconds
	offset := 0
	if len(fields) == 6 {
		offset = 1
	}

	c := &CronExpr{raw: expr}
	var err error

	// Parse minute (0-59)
	c.minute, err = parseField(fields[offset+0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("cron: minute field: %w", err)
	}

	// Parse hour (0-23)
	c.hour, err = parseField(fields[offset+1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("cron: hour field: %w", err)
	}

	// Parse day of month (1-31)
	c.dayOfMonth, err = parseField(fields[offset+2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("cron: day-of-month field: %w", err)
	}

	// Parse month (1-12)
	c.month, err = parseMonthField(fields[offset+3])
	if err != nil {
		return nil, fmt.Errorf("cron: month field: %w", err)
	}

	// Parse day of week (0-6)
	c.dayOfWeek, err = parseDayOfWeekField(fields[offset+4])
	if err != nil {
		return nil, fmt.Errorf("cron: day-of-week field: %w", err)
	}

	return c, nil
}

// parsePredefined parses predefined cron expressions.
func parsePredefined(expr string) (*CronExpr, error) {
	switch strings.ToLower(expr) {
	case "@yearly", "@annually":
		return &CronExpr{
			minute:     []int{0},
			hour:       []int{0},
			dayOfMonth: []int{1},
			month:      []int{1},
			dayOfWeek:  allValues(0, 6),
			raw:        expr,
		}, nil
	case "@monthly":
		return &CronExpr{
			minute:     []int{0},
			hour:       []int{0},
			dayOfMonth: []int{1},
			month:      allValues(1, 12),
			dayOfWeek:  allValues(0, 6),
			raw:        expr,
		}, nil
	case "@weekly":
		return &CronExpr{
			minute:     []int{0},
			hour:       []int{0},
			dayOfMonth: allValues(1, 31),
			month:      allValues(1, 12),
			dayOfWeek:  []int{0}, // Sunday
			raw:        expr,
		}, nil
	case "@daily", "@midnight":
		return &CronExpr{
			minute:     []int{0},
			hour:       []int{0},
			dayOfMonth: allValues(1, 31),
			month:      allValues(1, 12),
			dayOfWeek:  allValues(0, 6),
			raw:        expr,
		}, nil
	case "@hourly":
		return &CronExpr{
			minute:     []int{0},
			hour:       allValues(0, 23),
			dayOfMonth: allValues(1, 31),
			month:      allValues(1, 12),
			dayOfWeek:  allValues(0, 6),
			raw:        expr,
		}, nil
	default:
		// Check for @every format
		if strings.HasPrefix(expr, "@every ") {
			return nil, fmt.Errorf("cron: @every is not supported, use interval-based scheduling instead")
		}
		return nil, fmt.Errorf("cron: unknown predefined expression: %s", expr)
	}
}

// parseField parses a single cron field.
func parseField(field string, min, max int) ([]int, error) {
	if field == "*" {
		return allValues(min, max), nil
	}

	var result []int

	// Handle comma-separated values
	parts := strings.Split(field, ",")
	for _, part := range parts {
		values, err := parseFieldPart(part, min, max)
		if err != nil {
			return nil, err
		}
		result = append(result, values...)
	}

	// Remove duplicates and sort
	result = uniqueSorted(result)

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid values")
	}

	return result, nil
}

// parseFieldPart parses a single part of a cron field.
func parseFieldPart(part string, min, max int) ([]int, error) {
	// Handle step values
	var step int = 1
	if idx := strings.Index(part, "/"); idx != -1 {
		stepStr := part[idx+1:]
		var err error
		step, err = strconv.Atoi(stepStr)
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step: %s", stepStr)
		}
		part = part[:idx]
	}

	// Handle wildcard with step
	if part == "*" {
		return stepValues(min, max, step), nil
	}

	// Handle range
	if idx := strings.Index(part, "-"); idx != -1 {
		startStr := part[:idx]
		endStr := part[idx+1:]

		start, err := strconv.Atoi(startStr)
		if err != nil {
			return nil, fmt.Errorf("invalid range start: %s", startStr)
		}

		end, err := strconv.Atoi(endStr)
		if err != nil {
			return nil, fmt.Errorf("invalid range end: %s", endStr)
		}

		if start < min || end > max || start > end {
			return nil, fmt.Errorf("range %d-%d out of bounds (%d-%d)", start, end, min, max)
		}

		return stepValues(start, end, step), nil
	}

	// Handle single value
	val, err := strconv.Atoi(part)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %s", part)
	}

	if val < min || val > max {
		return nil, fmt.Errorf("value %d out of bounds (%d-%d)", val, min, max)
	}

	return []int{val}, nil
}

// parseMonthField parses a month field, supporting names.
func parseMonthField(field string) ([]int, error) {
	// Replace month names with numbers
	monthNames := map[string]string{
		"jan": "1", "feb": "2", "mar": "3", "apr": "4",
		"may": "5", "jun": "6", "jul": "7", "aug": "8",
		"sep": "9", "oct": "10", "nov": "11", "dec": "12",
	}

	lower := strings.ToLower(field)
	for name, num := range monthNames {
		lower = strings.ReplaceAll(lower, name, num)
	}

	return parseField(lower, 1, 12)
}

// parseDayOfWeekField parses a day-of-week field, supporting names.
func parseDayOfWeekField(field string) ([]int, error) {
	// Replace day names with numbers
	dayNames := map[string]string{
		"sun": "0", "mon": "1", "tue": "2", "wed": "3",
		"thu": "4", "fri": "5", "sat": "6",
	}

	lower := strings.ToLower(field)
	for name, num := range dayNames {
		lower = strings.ReplaceAll(lower, name, num)
	}

	// Handle 7 as Sunday (compatibility)
	lower = strings.ReplaceAll(lower, "7", "0")

	return parseField(lower, 0, 6)
}

// allValues returns all values in a range.
func allValues(min, max int) []int {
	result := make([]int, max-min+1)
	for i := range result {
		result[i] = min + i
	}
	return result
}

// stepValues returns values in a range with a step.
func stepValues(min, max, step int) []int {
	var result []int
	for i := min; i <= max; i += step {
		result = append(result, i)
	}
	return result
}

// uniqueSorted removes duplicates and sorts values.
func uniqueSorted(values []int) []int {
	if len(values) == 0 {
		return values
	}

	seen := make(map[int]bool)
	var result []int
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}

	// Simple bubble sort (values are small)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// Next returns the next time this cron expression will fire after the given time.
func (c *CronExpr) Next(after time.Time) time.Time {
	// Start from the next minute
	t := after.Truncate(time.Minute).Add(time.Minute)

	// Search for up to 5 years
	maxTime := after.AddDate(5, 0, 0)

	for t.Before(maxTime) {
		// Check month
		if !contains(c.month, int(t.Month())) {
			// Advance to next month
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}

		// Check day of month and day of week
		dayMatches := contains(c.dayOfMonth, t.Day())
		dowMatches := contains(c.dayOfWeek, int(t.Weekday()))

		// Both day-of-month and day-of-week must match (standard cron behavior)
		// unless one of them is * (all values)
		domIsAll := len(c.dayOfMonth) == 31
		dowIsAll := len(c.dayOfWeek) == 7

		var dayOk bool
		if domIsAll && dowIsAll {
			dayOk = true
		} else if domIsAll {
			dayOk = dowMatches
		} else if dowIsAll {
			dayOk = dayMatches
		} else {
			// If both are restricted, either can match (OR behavior)
			dayOk = dayMatches || dowMatches
		}

		if !dayOk {
			// Advance to next day
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}

		// Check hour
		if !contains(c.hour, t.Hour()) {
			// Find next matching hour
			nextHour := -1
			for _, h := range c.hour {
				if h > t.Hour() {
					nextHour = h
					break
				}
			}
			if nextHour == -1 {
				// No matching hour today, try tomorrow
				t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
				continue
			}
			t = time.Date(t.Year(), t.Month(), t.Day(), nextHour, 0, 0, 0, t.Location())
		}

		// Check minute
		if !contains(c.minute, t.Minute()) {
			// Find next matching minute
			nextMinute := -1
			for _, m := range c.minute {
				if m > t.Minute() {
					nextMinute = m
					break
				}
			}
			if nextMinute == -1 {
				// No matching minute this hour, try next hour
				t = t.Add(time.Hour).Truncate(time.Hour)
				continue
			}
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), nextMinute, 0, 0, t.Location())
		}

		// All conditions met
		return t
	}

	// No match found within 5 years
	return time.Time{}
}

// String returns the original cron expression.
func (c *CronExpr) String() string {
	return c.raw
}

// contains checks if a slice contains a value.
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// MustParseCron parses a cron expression and panics on error.
func MustParseCron(expr string) *CronExpr {
	c, err := ParseCron(expr)
	if err != nil {
		panic(err)
	}
	return c
}

// CronJob represents a job scheduled by a cron expression.
type CronJob struct {
	// Name identifies this job.
	Name string

	// Cron is the cron expression (e.g., "0 0 * * *").
	Cron string

	// Handler is the function to execute.
	Handler func(ctx context.Context) error

	// Timeout for each execution. Default: 5 minutes.
	Timeout time.Duration

	// Location is the timezone for the cron schedule.
	// Default: time.Local
	Location *time.Location

	// parsed is the parsed cron expression.
	parsed *CronExpr
}

// NextRun returns the next time this job should run after the given time.
func (j *CronJob) NextRun(after time.Time) time.Time {
	if j.parsed == nil {
		return time.Time{}
	}
	if j.Location != nil {
		after = after.In(j.Location)
	}
	return j.parsed.Next(after)
}

// cronEntry represents an internal entry for a cron job.
type cronEntry struct {
	job     *CronJob
	nextRun time.Time
	stopCh  chan struct{}
}
