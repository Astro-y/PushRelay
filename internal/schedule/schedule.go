package schedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type Spec struct {
	Kind     string `json:"kind"`
	Interval int    `json:"interval,omitempty"`
	Time     string `json:"time,omitempty"`
	Weekdays []int  `json:"weekdays,omitempty"`
	Day      int    `json:"day,omitempty"`
	Month    int    `json:"month,omitempty"`
	Cron     string `json:"cron,omitempty"`
	RunAt    string `json:"run_at,omitempty"`
	StartAt  string `json:"start_at,omitempty"`
	EndAt    string `json:"end_at,omitempty"`
}

func Parse(raw json.RawMessage, zone string) (Spec, error) {
	var s Spec
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, err
	}
	if s.Interval == 0 {
		s.Interval = 1
	}
	if s.Interval < 1 {
		return s, errors.New("interval must be positive")
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return s, err
	}
	switch s.Kind {
	case "once":
		if _, err := time.Parse(time.RFC3339, s.RunAt); err != nil {
			return s, fmt.Errorf("run_at: %w", err)
		}
	case "daily", "weekly", "monthly", "yearly":
		if _, _, err := clock(s.Time); err != nil {
			return s, err
		}
	case "cron":
		if strings.Fields(s.Cron) == nil || len(strings.Fields(s.Cron)) != 5 {
			return s, errors.New("cron must contain exactly five fields")
		}
		if _, err := cron.ParseStandard(s.Cron); err != nil {
			return s, err
		}
	default:
		return s, fmt.Errorf("unsupported recurrence kind %q", s.Kind)
	}
	return s, nil
}

func Next(raw json.RawMessage, zone string, after time.Time) (*time.Time, error) {
	s, err := Parse(raw, zone)
	if err != nil {
		return nil, err
	}
	loc, _ := time.LoadLocation(zone)
	after = after.In(loc)
	var next time.Time
	switch s.Kind {
	case "once":
		next, _ = time.Parse(time.RFC3339, s.RunAt)
		next = next.In(loc)
		if !next.After(after) {
			return nil, nil
		}
	case "cron":
		schedule, _ := cron.ParseStandard(s.Cron)
		next = schedule.Next(after)
	case "daily":
		next = nextDaily(s, after, loc)
	case "weekly":
		next = nextWeekly(s, after, loc)
	case "monthly":
		next = nextMonthly(s, after, loc)
	case "yearly":
		next = nextYearly(s, after, loc)
	}
	if next.IsZero() {
		return nil, nil
	}
	if s.EndAt != "" {
		end, err := time.Parse(time.RFC3339, s.EndAt)
		if err == nil && next.After(end.In(loc)) {
			return nil, nil
		}
	}
	utc := next.UTC()
	return &utc, nil
}

func clock(raw string) (int, int, error) {
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, 0, errors.New("time must use HH:mm")
	}
	h, e1 := strconv.Atoi(parts[0])
	m, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, errors.New("invalid time")
	}
	return h, m, nil
}
func anchor(s Spec, loc *time.Location, now time.Time) time.Time {
	if s.StartAt != "" {
		if t, err := time.Parse(time.RFC3339, s.StartAt); err == nil {
			return t.In(loc)
		}
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
}
func candidate(loc *time.Location, y int, m time.Month, d, h, min int) (time.Time, bool) {
	t := time.Date(y, m, d, h, min, 0, 0, loc)
	lt := t.In(loc)
	return t, lt.Year() == y && lt.Month() == m && lt.Day() == d && lt.Hour() == h && lt.Minute() == min
}
func daysBetween(a, b time.Time) int {
	return int(time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC).Sub(time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)).Hours() / 24)
}

func nextDaily(s Spec, after time.Time, loc *time.Location) time.Time {
	h, m, _ := clock(s.Time)
	a := anchor(s, loc, after)
	for i := 0; i < 3660; i++ {
		d := time.Date(after.Year(), after.Month(), after.Day()+i, 0, 0, 0, 0, loc)
		if daysBetween(a, d) >= 0 && daysBetween(a, d)%s.Interval == 0 {
			if c, ok := candidate(loc, d.Year(), d.Month(), d.Day(), h, m); ok && c.After(after) {
				return c
			}
		}
	}
	return time.Time{}
}
func nextWeekly(s Spec, after time.Time, loc *time.Location) time.Time {
	h, m, _ := clock(s.Time)
	a := anchor(s, loc, after)
	allowed := map[int]bool{}
	for _, d := range s.Weekdays {
		if d >= 0 && d <= 6 {
			allowed[d] = true
		}
	}
	if len(allowed) == 0 {
		allowed[int(a.Weekday())] = true
	}
	for i := 0; i < 3660; i++ {
		d := time.Date(after.Year(), after.Month(), after.Day()+i, 0, 0, 0, 0, loc)
		weeks := daysBetween(a, d) / 7
		if daysBetween(a, d) >= 0 && weeks%s.Interval == 0 && allowed[int(d.Weekday())] {
			if c, ok := candidate(loc, d.Year(), d.Month(), d.Day(), h, m); ok && c.After(after) {
				return c
			}
		}
	}
	return time.Time{}
}
func nextMonthly(s Spec, after time.Time, loc *time.Location) time.Time {
	h, m, _ := clock(s.Time)
	a := anchor(s, loc, after)
	day := s.Day
	if day == 0 {
		day = a.Day()
	}
	for i := 0; i < 1200; i++ {
		monthStart := time.Date(after.Year(), after.Month()+time.Month(i), 1, 0, 0, 0, 0, loc)
		diff := (monthStart.Year()-a.Year())*12 + int(monthStart.Month()-a.Month())
		if diff >= 0 && diff%s.Interval == 0 {
			if c, ok := candidate(loc, monthStart.Year(), monthStart.Month(), day, h, m); ok && c.After(after) {
				return c
			}
		}
	}
	return time.Time{}
}
func nextYearly(s Spec, after time.Time, loc *time.Location) time.Time {
	h, m, _ := clock(s.Time)
	a := anchor(s, loc, after)
	month := s.Month
	if month == 0 {
		month = int(a.Month())
	}
	day := s.Day
	if day == 0 {
		day = a.Day()
	}
	for i := 0; i < 100; i++ {
		year := after.Year() + i
		if year >= a.Year() && (year-a.Year())%s.Interval == 0 {
			if c, ok := candidate(loc, year, time.Month(month), day, h, m); ok && c.After(after) {
				return c
			}
		}
	}
	return time.Time{}
}
