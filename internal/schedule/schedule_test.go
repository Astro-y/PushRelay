package schedule

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMonthlySkipsInvalidDay(t *testing.T) {
	after := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	next, err := Next(json.RawMessage(`{"kind":"monthly","interval":1,"time":"09:30","day":31}`), "UTC", after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 3, 31, 9, 30, 0, 0, time.UTC)
	if next == nil || !next.Equal(want) {
		t.Fatalf("got %v want %v", next, want)
	}
}
func TestYearlyLeapDay(t *testing.T) {
	after := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	next, err := Next(json.RawMessage(`{"kind":"yearly","interval":1,"time":"08:00","month":2,"day":29}`), "UTC", after)
	if err != nil {
		t.Fatal(err)
	}
	if next == nil || next.Year() != 2028 || next.Month() != 2 || next.Day() != 29 {
		t.Fatal(next)
	}
}
func TestCronRequiresFiveFields(t *testing.T) {
	if _, err := Parse(json.RawMessage(`{"kind":"cron","cron":"0 0 9 * * *"}`), "UTC"); err == nil {
		t.Fatal("expected six field cron to fail")
	}
}

func TestDailyIntervalUsesStableAnchor(t *testing.T) {
	raw := json.RawMessage(`{"kind":"daily","interval":2,"time":"09:00","start_at":"2026-01-01T00:00:00Z"}`)
	next, err := Next(raw, "UTC", time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 1, 3, 9, 0, 0, 0, time.UTC)
	if next == nil || !next.Equal(want) {
		t.Fatalf("got %v want %v", next, want)
	}
}
func TestDSTGapIsSkipped(t *testing.T) {
	after := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	next, err := Next(json.RawMessage(`{"kind":"daily","time":"02:30"}`), "Europe/Berlin", after)
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Berlin")
	local := next.In(loc)
	if local.Day() != 30 || local.Hour() != 2 || local.Minute() != 30 {
		t.Fatal(local)
	}
}
