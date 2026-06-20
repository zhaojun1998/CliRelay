package usage

import (
	"testing"
	"time"
)

func TestParseTimeWindowAtDateOnlyInclusiveEnd(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	win, ok := parseTimeWindowAt(loc, "2026-06-01", "2026-06-10")
	if !ok {
		t.Fatal("parseTimeWindowAt() ok = false, want true")
	}
	wantStart := time.Date(2026, 6, 1, 0, 0, 0, 0, loc).UTC()
	// A date-only end is inclusive of that whole day, so the exclusive upper
	// bound is the next day's midnight.
	wantEnd := time.Date(2026, 6, 11, 0, 0, 0, 0, loc).UTC()
	if !win.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", win.Start, wantStart)
	}
	if !win.End.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", win.End, wantEnd)
	}
}

func TestParseTimeWindowAtWithExplicitTime(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	win, ok := parseTimeWindowAt(loc, "2026-06-01T08:30", "2026-06-01T12:00")
	if !ok {
		t.Fatal("parseTimeWindowAt() ok = false, want true")
	}
	wantStart := time.Date(2026, 6, 1, 8, 30, 0, 0, loc).UTC()
	// An explicit time is taken literally; no rounding up to the next day.
	wantEnd := time.Date(2026, 6, 1, 12, 0, 0, 0, loc).UTC()
	if !win.Start.Equal(wantStart) || !win.End.Equal(wantEnd) {
		t.Errorf("window = [%v, %v], want [%v, %v]", win.Start, win.End, wantStart, wantEnd)
	}
}

func TestParseTimeWindowAtRejectsInvalid(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	cases := []struct {
		name, start, end string
	}{
		{"start after end", "2026-06-10", "2026-06-01"},
		{"start equals end", "2026-06-01T10:00", "2026-06-01T10:00"},
		{"missing start", "", "2026-06-10"},
		{"missing end", "2026-06-01", ""},
		{"garbage start", "not-a-date", "2026-06-10"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := parseTimeWindowAt(loc, tc.start, tc.end); ok {
				t.Errorf("parseTimeWindowAt(%q, %q) ok = true, want false", tc.start, tc.end)
			}
		})
	}
}

func TestSpanDaysAt(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	mk := func(s, e string) TimeWindow {
		win, ok := parseTimeWindowAt(loc, s, e)
		if !ok {
			t.Fatalf("parseTimeWindowAt(%q, %q) failed", s, e)
		}
		return win
	}
	now := time.Date(2026, 6, 19, 10, 0, 0, 0, loc)

	cases := []struct {
		name string
		win  TimeWindow
		want int
	}{
		{"ten day inclusive range", mk("2026-06-01", "2026-06-10"), 10},
		{"single day", mk("2026-06-01", "2026-06-01"), 1},
		{"same day with time", mk("2026-06-01T08:00", "2026-06-01T12:00"), 1},
		{"cross midnight short", mk("2026-06-01T23:00", "2026-06-02T01:00"), 2},
		{"open range falls back to now", TimeWindow{Start: time.Date(2026, 6, 13, 0, 0, 0, 0, loc).UTC()}, 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.win.spanDaysAt(now, loc); got != tc.want {
				t.Errorf("spanDaysAt() = %d, want %d", got, tc.want)
			}
		})
	}
}
