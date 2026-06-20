package usage

import (
	"strings"
	"time"
)

func cutoffStartUTCAt(now time.Time, days int) time.Time {
	if days < 1 {
		days = 7
	}
	loc := getUsageLocation()
	now = now.In(loc)
	todayStartLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return todayStartLocal.AddDate(0, 0, -(days - 1)).UTC()
}

// CutoffStartUTC returns the start-of-day cutoff for the given number of days
// in the project-configured timezone, converted to UTC. Exported so that
// dashboard and other callers can reuse the same time-range semantics.
func CutoffStartUTC(days int) time.Time {
	return cutoffStartUTCAt(time.Now(), days)
}

func localDayKeyAt(t time.Time) string {
	loc := getUsageLocation()
	return t.In(loc).Format("2006-01-02")
}

// LocalDayKeyAt returns the YYYY-MM-DD day key in the project-configured timezone.
func LocalDayKeyAt(t time.Time) string {
	return localDayKeyAt(t)
}

func cutoffDayKey(days int) string {
	return localDayKeyAt(CutoffStartUTC(days))
}

// TimeWindow represents a UTC time range [Start, End). A zero End means an open
// range (upper bound = now), preserving the existing "last N days up to now"
// semantics so the days-based path stays byte-for-byte compatible.
type TimeWindow struct {
	Start time.Time // UTC, inclusive lower bound
	End   time.Time // UTC, exclusive upper bound; zero = open to now
}

// WindowFromDays builds a "last N days up to now" window by reusing
// CutoffStartUTC. End is left zero (open), matching the legacy behaviour.
func WindowFromDays(days int) TimeWindow {
	return TimeWindow{Start: CutoffStartUTC(days)}
}

// ParseTimeWindow parses local datetime strings from the frontend
// ("2006-01-02" or "2006-01-02T15:04") into a UTC window using the project
// timezone. Both start and end must be present and start must be before end,
// otherwise ok=false so callers fall back to the days-based range.
func ParseTimeWindow(startStr, endStr string) (TimeWindow, bool) {
	return parseTimeWindowAt(getUsageLocation(), startStr, endStr)
}

func parseTimeWindowAt(loc *time.Location, startStr, endStr string) (TimeWindow, bool) {
	if loc == nil {
		loc = time.Local
	}
	startStr = strings.TrimSpace(startStr)
	endStr = strings.TrimSpace(endStr)
	if startStr == "" || endStr == "" {
		return TimeWindow{}, false
	}
	start, ok := parseLocalBoundary(startStr, loc, false)
	if !ok {
		return TimeWindow{}, false
	}
	end, ok := parseLocalBoundary(endStr, loc, true)
	if !ok {
		return TimeWindow{}, false
	}
	if !start.Before(end) {
		return TimeWindow{}, false
	}
	return TimeWindow{Start: start.UTC(), End: end.UTC()}, true
}

// parseLocalBoundary parses a single boundary, trying "2006-01-02T15:04" first
// then "2006-01-02". For a date-only end boundary it returns the next day's
// midnight (the exclusive upper bound that makes the end date inclusive).
func parseLocalBoundary(s string, loc *time.Location, isEnd bool) (time.Time, bool) {
	if t, err := time.ParseInLocation("2006-01-02T15:04", s, loc); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		if isEnd {
			return t.AddDate(0, 0, 1), true
		}
		return t, true
	}
	return time.Time{}, false
}

// SpanDays returns the number of local calendar days the window covers, used to
// switch aggregation granularity (<=1 day -> hourly, >1 day -> daily).
func (w TimeWindow) SpanDays() int {
	return w.spanDaysAt(time.Now(), getUsageLocation())
}

func (w TimeWindow) spanDaysAt(now time.Time, loc *time.Location) int {
	if loc == nil {
		loc = time.Local
	}
	end := w.End
	if end.IsZero() {
		end = now
	}
	startLocal := w.Start.In(loc)
	endLocal := end.In(loc)
	startDay := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, loc)
	endDay := time.Date(endLocal.Year(), endLocal.Month(), endLocal.Day(), 0, 0, 0, 0, loc)
	// Exclusive upper bound: an end exactly on midnight does not count that day;
	// otherwise round up to the day it lands in.
	if endLocal.After(endDay) {
		endDay = endDay.AddDate(0, 0, 1)
	}
	days := int(endDay.Sub(startDay).Hours()/24 + 0.5)
	if days < 1 {
		days = 1
	}
	return days
}

// boundsForQuery returns the lower/upper RFC3339 bounds for SQL queries. The
// upper bound is "" when End is zero (open range).
func (w TimeWindow) boundsForQuery() (string, string) {
	start := w.Start.UTC().Format(time.RFC3339)
	if w.End.IsZero() {
		return start, ""
	}
	return start, w.End.UTC().Format(time.RFC3339)
}
