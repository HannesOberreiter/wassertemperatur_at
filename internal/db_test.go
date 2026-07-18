package internal

import (
	"context"
	"database/sql"
	"path/filepath"
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

func TestListWatersIncludesLatestAndWeeklyChange(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "wasser.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now().UTC().Truncate(time.Second)
	readings := []Reading{
		{SourceKey: "test:1", Name: "Testsee", State: "Testland", Source: "Testquelle", MeasuredAt: now, Temperature: sql.NullFloat64{Float64: 22, Valid: true}, Depth: sql.NullFloat64{Float64: 2, Valid: true}},
		{SourceKey: "test:1", Name: "Testsee", State: "Testland", Source: "Testquelle", MeasuredAt: now.AddDate(0, 0, -1), Temperature: sql.NullFloat64{Float64: 20, Valid: true}},
		{SourceKey: "test:1", Name: "Testsee", State: "Testland", Source: "Testquelle", MeasuredAt: now.AddDate(0, 0, -8), Temperature: sql.NullFloat64{Float64: 10, Valid: true}},
		{SourceKey: "test:1", Name: "Testsee", State: "Testland", Source: "Testquelle", MeasuredAt: now.AddDate(0, 0, -9), Temperature: sql.NullFloat64{Float64: 10, Valid: true}},
	}
	if err := SaveReadings(context.Background(), db, readings); err != nil {
		t.Fatal(err)
	}

	waters, err := ListWaters(context.Background(), db, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(waters) != 1 {
		t.Fatalf("got %d waters, want 1", len(waters))
	}
	water := waters[0]
	if !water.MeasuredAt.Valid || !water.MeasuredAt.Time.Equal(now) || !water.Temperature.Valid || water.Temperature.Float64 != 22 {
		t.Fatalf("unexpected latest observation: %#v", water)
	}
	if !water.Depth.Valid || water.Depth.Float64 != 2 || !water.Recent {
		t.Fatalf("unexpected latest metadata: %#v", water)
	}
	if !water.Change.Valid || water.Change.Float64 < 109.9 || water.Change.Float64 > 110.1 {
		t.Fatalf("want 110%% change, got %#v", water.Change)
	}
}

func TestSlugify(t *testing.T) {
	if got := slugify("Wörther See / Klagenfurt"); got != "woerther-see-klagenfurt" {
		t.Fatalf("unexpected slug %q", got)
	}
}
