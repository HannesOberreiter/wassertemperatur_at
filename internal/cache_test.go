package internal

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func newCacheTestServer(t *testing.T, ttl time.Duration) *Server {
	t.Helper()
	db, err := OpenDB(filepath.Join(t.TempDir(), "wasser.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Now().UTC().Truncate(time.Second)
	readings := []Reading{
		{SourceKey: "test:1", Name: "Wörthersee", State: "Kärnten", Source: "Testquelle", MeasuredAt: now, Temperature: sql.NullFloat64{Float64: 20, Valid: true}},
		{SourceKey: "test:2", Name: "Attersee", State: "Oberösterreich", Source: "Testquelle", MeasuredAt: now, Temperature: sql.NullFloat64{Float64: 18, Valid: true}},
		{SourceKey: "test:3", Name: "Badesee", State: "Kärnten", Source: agesSource, MeasuredAt: now, Temperature: sql.NullFloat64{Float64: 21, Valid: true}},
	}
	if err := SaveReadings(context.Background(), db, readings); err != nil {
		t.Fatal(err)
	}
	return &Server{db: db, templates: templates(), lists: newListCache(ttl)}
}

func waterNames(waters []Water) []string {
	names := make([]string, 0, len(waters))
	for _, water := range waters {
		names = append(names, water.Name)
	}
	return names
}

func TestPageDataFiltersCachedList(t *testing.T) {
	s := newCacheTestServer(t, time.Hour)
	tests := []struct {
		name    string
		url     string
		quality bool
		want    []string
	}{
		{"all temperature waters", "/", false, []string{"Attersee", "Wörthersee"}},
		{"search ignores case of umlauts", "/?suche=W%C3%96RTHER", false, []string{"Wörthersee"}},
		{"state filter", "/?bundesland=Ober%C3%B6sterreich", false, []string{"Attersee"}},
		{"search and state without match", "/?suche=atter&bundesland=K%C3%A4rnten", false, []string{}},
		{"quality page has only AGES", "/wasserqualitaet", true, []string{"Badesee"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := s.pageData(httptest.NewRequest("GET", tt.url, nil), tt.quality)
			if err != nil {
				t.Fatal(err)
			}
			got := waterNames(data.Waters)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
	data, err := s.pageData(httptest.NewRequest("GET", "/", nil), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.States) != 2 || data.States[0] != "Kärnten" || data.States[1] != "Oberösterreich" {
		t.Fatalf("unexpected states %v", data.States)
	}
}

func TestPageDataServesListFromCache(t *testing.T) {
	s := newCacheTestServer(t, time.Hour)
	if _, err := s.pageData(httptest.NewRequest("GET", "/", nil), false); err != nil {
		t.Fatal(err)
	}
	// A closed database proves that the second request does not query SQLite.
	_ = s.db.Close()
	data, err := s.pageData(httptest.NewRequest("GET", "/?suche=atter", nil), false)
	if err != nil {
		t.Fatalf("cached request queried the database: %v", err)
	}
	if got := waterNames(data.Waters); len(got) != 1 || got[0] != "Attersee" {
		t.Fatalf("got %v, want [Attersee]", got)
	}
}

func TestPageDataReloadsExpiredList(t *testing.T) {
	s := newCacheTestServer(t, 0)
	if _, err := s.pageData(httptest.NewRequest("GET", "/", nil), false); err != nil {
		t.Fatal(err)
	}
	_ = s.db.Close()
	if _, err := s.pageData(httptest.NewRequest("GET", "/", nil), false); err == nil {
		t.Fatal("expired list was not loaded again")
	}
}
