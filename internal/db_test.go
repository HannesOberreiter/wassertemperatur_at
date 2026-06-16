package internal

import (
	"database/sql"
	"testing"
	"time"
)

func TestWeeklyAverageChange(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	observations := []Observation{
		{MeasuredAt: now, Temperature: sql.NullFloat64{Float64: 22, Valid: true}},
		{MeasuredAt: now.AddDate(0, 0, -1), Temperature: sql.NullFloat64{Float64: 20, Valid: true}},
		{MeasuredAt: now.AddDate(0, 0, -8), Temperature: sql.NullFloat64{Float64: 10, Valid: true}},
		{MeasuredAt: now.AddDate(0, 0, -9), Temperature: sql.NullFloat64{Float64: 10, Valid: true}},
	}
	change := weeklyAverageChange(observations)
	if !change.Valid || change.Float64 < 109.9 || change.Float64 > 110.1 {
		t.Fatalf("want 110, got %#v", change)
	}
}

func TestSlugify(t *testing.T) {
	if got := slugify("Wörther See / Klagenfurt"); got != "woerther-see-klagenfurt" {
		t.Fatalf("unexpected slug %q", got)
	}
}
