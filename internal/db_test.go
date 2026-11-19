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

func TestOpenDBLimitsConnections(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "wasser.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got := db.Stats().MaxOpenConnections; got != maxOpenConns {
		t.Fatalf("got %d max open connections, want %d", got, maxOpenConns)
	}
}

func TestDailySummariesPagesByDay(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "wasser.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	day := func(d, h int) time.Time { return time.Date(2026, 7, d, h, 0, 0, 0, time.UTC) }
	reading := func(at time.Time, temperature float64) Reading {
		return Reading{SourceKey: "test:1", Name: "Testsee", State: "Testland", Source: "Testquelle", MeasuredAt: at, Temperature: sql.NullFloat64{Float64: temperature, Valid: true}}
	}
	readings := []Reading{
		reading(day(1, 8), 10), reading(day(1, 12), 14),
		reading(day(2, 8), 11), reading(day(2, 12), 13), reading(day(2, 16), 18),
		reading(day(3, 8), 20),
	}
	if err := SaveReadings(context.Background(), db, readings); err != nil {
		t.Fatal(err)
	}
	var waterID int64
	if err := db.QueryRow(`select id from waters where source_key = 'test:1'`).Scan(&waterID); err != nil {
		t.Fatal(err)
	}

	first, err := DailySummaries(context.Background(), db, waterID, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []DailySummary{
		{Day: time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), Avg: 20, Median: 20, High: 20, Low: 20, Count: 1},
		{Day: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC), Avg: 14, Median: 13, High: 18, Low: 11, Count: 3},
	}
	if len(first) != len(want) {
		t.Fatalf("got %d summaries, want %d: %#v", len(first), len(want), first)
	}
	for i := range want {
		if first[i] != want[i] {
			t.Fatalf("summary %d: got %#v, want %#v", i, first[i], want[i])
		}
	}

	second, err := DailySummaries(context.Background(), db, waterID, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	wantLast := DailySummary{Day: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Avg: 12, Median: 12, High: 14, Low: 10, Count: 2}
	if len(second) != 1 || second[0] != wantLast {
		t.Fatalf("got %#v, want [%#v]", second, wantLast)
	}

	beyond, err := DailySummaries(context.Background(), db, waterID, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(beyond) != 0 {
		t.Fatalf("got %#v, want no summaries", beyond)
	}
}

func TestSlugify(t *testing.T) {
	if got := slugify("Wörther See / Klagenfurt"); got != "woerther-see-klagenfurt" {
		t.Fatalf("unexpected slug %q", got)
	}
}
