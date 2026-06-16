package internal

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Water struct {
	ID          int64
	Slug        string
	Name        string
	State       string
	Source      string
	Temperature sql.NullFloat64
	MeasuredAt  sql.NullTime
	Depth       sql.NullFloat64
	Quality     sql.NullInt64
	Change      sql.NullFloat64
	Recent      bool
}

type Observation struct {
	MeasuredAt    time.Time
	Temperature   sql.NullFloat64
	Depth         sql.NullFloat64
	Quality       sql.NullInt64
	Enterococci   sql.NullInt64
	EColi         sql.NullInt64
	EnterococciOp string
	EColiOp       string
}

type DailySummary struct {
	Day    time.Time
	Avg    float64
	Median float64
	High   float64
	Low    float64
	Count  int
}

type Reading struct {
	SourceKey     string
	Name          string
	State         string
	Source        string
	MeasuredAt    time.Time
	Temperature   sql.NullFloat64
	Depth         sql.NullFloat64
	Quality       sql.NullInt64
	Enterococci   sql.NullInt64
	EColi         sql.NullInt64
	EnterococciOp string
	EColiOp       string
}

func OpenDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, stmt := range migrations {
		_, _ = db.Exec(stmt)
	}
	return db, nil
}

func SaveReadings(ctx context.Context, db *sql.DB, readings []Reading) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, reading := range readings {
		if !reading.Temperature.Valid || reading.MeasuredAt.IsZero() || reading.SourceKey == "" {
			continue
		}
		slug := slugify(reading.Name)
		_, err := tx.ExecContext(ctx, `insert into waters(source_key, slug, name, state, source) values(?, ?, ?, ?, ?)
			on conflict(source_key) do update set slug=excluded.slug, name=excluded.name, state=excluded.state, source=excluded.source`,
			reading.SourceKey, slug, reading.Name, reading.State, reading.Source)
		if err != nil {
			return err
		}

		var waterID int64
		if err := tx.QueryRowContext(ctx, `select id from waters where source_key = ?`, reading.SourceKey).Scan(&waterID); err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `insert into observations(water_id, measured_at, temperature, depth, quality, enterococci, ecoli, enterococci_op, ecoli_op) values(?, ?, ?, ?, ?, ?, ?, ?, ?)
			on conflict(water_id, measured_at, temperature) do update set depth=excluded.depth, quality=excluded.quality, enterococci=excluded.enterococci, ecoli=excluded.ecoli, enterococci_op=excluded.enterococci_op, ecoli_op=excluded.ecoli_op`,
			waterID, reading.MeasuredAt.Format(time.RFC3339), reading.Temperature, reading.Depth, reading.Quality, reading.Enterococci, reading.EColi, reading.EnterococciOp, reading.EColiOp)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func ListWaters(ctx context.Context, db *sql.DB, search, state string) ([]Water, error) {
	return listWaters(ctx, db, search, state, "!AGES Badegewässerdatenbank")
}

func ListQualityWaters(ctx context.Context, db *sql.DB, search, state string) ([]Water, error) {
	return listWaters(ctx, db, search, state, "AGES Badegewässerdatenbank")
}

// ListStates keeps filters honest: only states with stored data are shown.
// source may start with "!" to exclude that source.
func ListStates(ctx context.Context, db *sql.DB, source string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `select distinct state from waters where (? = '' or (substr(?, 1, 1) = '!' and source != substr(?, 2)) or source = ?) order by state`, source, source, source, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var states []string
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

func listWaters(ctx context.Context, db *sql.DB, search, state, source string) ([]Water, error) {
	rows, err := db.QueryContext(ctx, `select id, slug, name, state, source from waters
		where (? = '' or lower(name) like '%' || lower(?) || '%') and (? = '' or state = ?) and (? = '' or (substr(?, 1, 1) = '!' and source != substr(?, 2)) or source = ?)
		order by name`, search, search, state, state, source, source, source, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var waters []Water
	for rows.Next() {
		var water Water
		if err := rows.Scan(&water.ID, &water.Slug, &water.Name, &water.State, &water.Source); err != nil {
			return nil, err
		}
		latest, err := LatestObservations(ctx, db, water.ID, 2)
		if err != nil {
			return nil, err
		}
		if len(latest) > 0 {
			water.MeasuredAt = sql.NullTime{Time: latest[0].MeasuredAt, Valid: true}
			water.Temperature = latest[0].Temperature
			water.Depth = latest[0].Depth
			water.Quality = latest[0].Quality
			water.Recent = time.Since(latest[0].MeasuredAt) < 14*24*time.Hour
		}
		observations, err := LatestObservations(ctx, db, water.ID, 10000)
		if err != nil {
			return nil, err
		}
		water.Change = weeklyAverageChange(observations)
		waters = append(waters, water)
	}
	return waters, rows.Err()
}

func GetWater(ctx context.Context, db *sql.DB, id string) (Water, error) {
	var water Water
	err := db.QueryRowContext(ctx, `select id, slug, name, state, source from waters where id = ?`, id).Scan(&water.ID, &water.Slug, &water.Name, &water.State, &water.Source)
	return water, err
}

func DailySummaries(ctx context.Context, db *sql.DB, waterID int64, limit, offset int) ([]DailySummary, error) {
	observations, err := LatestObservations(ctx, db, waterID, 10000)
	if err != nil {
		return nil, err
	}
	byDay := map[string][]float64{}
	for _, observation := range observations {
		if observation.Temperature.Valid {
			key := observation.MeasuredAt.Format("2006-01-02")
			byDay[key] = append(byDay[key], observation.Temperature.Float64)
		}
	}
	keys := make([]string, 0, len(byDay))
	for key := range byDay {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	if offset > len(keys) {
		return nil, nil
	}
	end := offset + limit
	if end > len(keys) {
		end = len(keys)
	}

	summaries := make([]DailySummary, 0, end-offset)
	for _, key := range keys[offset:end] {
		values := byDay[key]
		sort.Float64s(values)
		var sum float64
		for _, value := range values {
			sum += value
		}
		median := values[len(values)/2]
		if len(values)%2 == 0 {
			median = (values[len(values)/2-1] + values[len(values)/2]) / 2
		}
		day, _ := time.Parse("2006-01-02", key)
		summaries = append(summaries, DailySummary{Day: day, Avg: sum / float64(len(values)), Median: median, Low: values[0], High: values[len(values)-1], Count: len(values)})
	}
	return summaries, nil
}

func LatestObservations(ctx context.Context, db *sql.DB, waterID int64, limit int) ([]Observation, error) {
	rows, err := db.QueryContext(ctx, `select measured_at, temperature, depth, quality, enterococci, ecoli, coalesce(enterococci_op, ''), coalesce(ecoli_op, '') from observations where water_id = ? order by measured_at desc limit ?`, waterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []Observation
	for rows.Next() {
		var raw string
		var observation Observation
		if err := rows.Scan(&raw, &observation.Temperature, &observation.Depth, &observation.Quality, &observation.Enterococci, &observation.EColi, &observation.EnterococciOp, &observation.EColiOp); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, err
		}
		observation.MeasuredAt = parsed
		observations = append(observations, observation)
	}
	return observations, rows.Err()
}

// weeklyAverageChange compares the latest 7-day average with the 7 days before it.
func weeklyAverageChange(observations []Observation) sql.NullFloat64 {
	if len(observations) == 0 {
		return sql.NullFloat64{}
	}
	latest := observations[0].MeasuredAt
	currentStart := latest.AddDate(0, 0, -7)
	previousStart := latest.AddDate(0, 0, -14)
	var currentSum, previousSum float64
	var currentCount, previousCount int
	for _, observation := range observations {
		if !observation.Temperature.Valid {
			continue
		}
		switch {
		case observation.MeasuredAt.After(currentStart) || observation.MeasuredAt.Equal(currentStart):
			currentSum += observation.Temperature.Float64
			currentCount++
		case observation.MeasuredAt.After(previousStart) || observation.MeasuredAt.Equal(previousStart):
			previousSum += observation.Temperature.Float64
			previousCount++
		}
	}
	if currentCount == 0 || previousCount == 0 {
		return sql.NullFloat64{}
	}
	previousAvg := previousSum / float64(previousCount)
	if previousAvg == 0 {
		return sql.NullFloat64{}
	}
	currentAvg := currentSum / float64(currentCount)
	return sql.NullFloat64{Float64: (currentAvg - previousAvg) / previousAvg * 100, Valid: true}
}

func slugify(value string) string {
	replacer := strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "Ä", "ae", "Ö", "oe", "Ü", "ue")
	value = strings.ToLower(replacer.Replace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			builder.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func fmtPlainFloat(value float64, suffix string) string {
	return fmt.Sprintf("%.1f%s", value, suffix)
}

func fmtFloat(value sql.NullFloat64, suffix string) string {
	if !value.Valid {
		return ""
	}
	return fmt.Sprintf("%.1f%s", value.Float64, suffix)
}

const schema = `
create table if not exists waters (
	id integer primary key autoincrement,
	source_key text not null unique,
	slug text not null,
	name text not null,
	state text not null,
	source text not null
);
create table if not exists observations (
	id integer primary key autoincrement,
	water_id integer not null references waters(id) on delete cascade,
	measured_at text not null,
	temperature real,
	depth real,
	quality integer,
	unique(water_id, measured_at, temperature)
);
create index if not exists observations_water_date on observations(water_id, measured_at desc);
`

var migrations = []string{
	`alter table observations add column enterococci integer`,
	`alter table observations add column ecoli integer`,
	`alter table observations add column enterococci_op text`,
	`alter table observations add column ecoli_op text`,
}
