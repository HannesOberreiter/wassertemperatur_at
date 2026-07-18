package internal

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

const (
	urlAGES        = "https://www.ages.at/typo3temp/badegewaesser_db.json"
	urlOOE         = "https://data.ooe.gv.at/files/hydro/HDOOE_Export_WT.zrxp"
	urlKtnLakes    = "https://hydrographie.ktn.gv.at/DE/repos/evoscripts/hydrografischer/getSeeWassertemperatur.es"
	urlKtnRivers   = "https://hydrographie.ktn.gv.at/DE/repos/evoscripts/hydrografischer/getFluesseWassertemperatur.es"
	urlAusseerland = "https://www.steiermark.com/de/Ausseerland-Salzkammergut/Region/Sommerfrische/Seen-im-Ausseerland/Wassertemperaturen"
	urlVOWIS       = "https://vowis.vorarlberg.at/stationsInfo/tbl_Abflussstationen.aspx"
)

func StartCron(db *sql.DB, interval time.Duration) {
	if interval <= 0 {
		slog.Info("Cron job disabled", "interval", interval)
		return
	}
	slog.Info("Cron job scheduled", "interval", interval, "first_run_in", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for scheduledAt := range ticker.C {
		startedAt := time.Now()
		slog.Info("Cron job started", "scheduled_at", scheduledAt.Format(time.RFC3339))
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		err := UpdateTemperatures(ctx, db)
		cancel()
		if err != nil {
			slog.Warn("Cron job failed", "error", err, "duration", time.Since(startedAt))
			continue
		}
		slog.Info("Cron job completed", "duration", time.Since(startedAt))
	}
}

// UpdateTemperatures fetches every source independently; one broken source must not block the rest.
func UpdateTemperatures(ctx context.Context, db *sql.DB) error {
	var readings []Reading
	fetchers := []func(context.Context) ([]Reading, error){fetchAGES, fetchOOE, fetchCarinthia, fetchAusseerland, fetchVOWIS}
	for _, fetcher := range fetchers {
		part, err := fetcher(ctx)
		if err != nil {
			slog.Warn("Source failed", "error", err)
			continue
		}
		readings = append(readings, part...)
	}
	if err := SaveReadings(ctx, db, readings); err != nil {
		return err
	}
	slog.Info("Data fetch completed", "readings", len(readings))
	return nil
}

type agesResponse struct {
	States []struct {
		Name   string `json:"BUNDESLAND"`
		Waters []struct {
			ID       string `json:"BADEGEWAESSERID"`
			Name     string `json:"BADEGEWAESSERNAME"`
			Readings []struct {
				Date          string  `json:"D"`
				EnterococciOp string  `json:"O_E"`
				Enterococci   int64   `json:"E"`
				EColiOp       string  `json:"O_EC"`
				EColi         int64   `json:"E_C"`
				Temperature   float64 `json:"W"`
				Depth         float64 `json:"S"`
				Quality       int64   `json:"A"`
			} `json:"MESSWERTE"`
		} `json:"BADEGEWAESSER"`
	} `json:"BUNDESLAENDER"`
}

func fetchAGES(ctx context.Context) ([]Reading, error) {
	body, err := get(ctx, urlAGES)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var response agesResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, err
	}

	var readings []Reading
	for _, state := range response.States {
		for _, water := range state.Waters {
			for _, value := range water.Readings {
				date, err := time.Parse("02.01.2006", value.Date)
				if err != nil || value.Temperature == 0 {
					continue
				}
				readings = append(readings, Reading{
					SourceKey:     "ages:" + water.ID,
					Name:          water.Name,
					State:         state.Name,
					Source:        "AGES Badegewässerdatenbank",
					MeasuredAt:    date,
					Temperature:   sql.NullFloat64{Float64: value.Temperature, Valid: true},
					Depth:         sql.NullFloat64{Float64: value.Depth, Valid: value.Depth > 0},
					Quality:       sql.NullInt64{Int64: value.Quality, Valid: value.Quality > 0},
					Enterococci:   sql.NullInt64{Int64: value.Enterococci, Valid: value.Enterococci > 0},
					EColi:         sql.NullInt64{Int64: value.EColi, Valid: value.EColi > 0},
					EnterococciOp: value.EnterococciOp,
					EColiOp:       value.EColiOp,
				})
			}
		}
	}
	return readings, nil
}

func fetchOOE(ctx context.Context) ([]Reading, error) {
	body, err := get(ctx, urlOOE)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	decoded, err := charmap.ISO8859_1.NewDecoder().Bytes(content)
	if err != nil {
		return nil, err
	}
	return parseZRXP(string(decoded)), nil
}

var metaPattern = regexp.MustCompile(`SNAME(.*?)\|.*?SWATER(.*?)\|`)

func parseZRXP(content string) []Reading {
	var readings []Reading
	name := ""
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#SANR") {
			match := metaPattern.FindStringSubmatch(line)
			if len(match) == 3 {
				name = strings.TrimSpace(match[2] + ", " + match[1])
			}
			continue
		}
		if strings.HasPrefix(line, "#") || name == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		date, err := time.Parse("20060102150405", fields[0])
		if err != nil {
			continue
		}
		temperature, err := strconv.ParseFloat(strings.ReplaceAll(fields[1], ",", "."), 64)
		if err != nil || temperature < -50 || temperature > 50 {
			continue
		}
		readings = append(readings, Reading{
			SourceKey:   "ooe:" + name,
			Name:        name,
			State:       "Oberösterreich",
			Source:      "Hydrografischer Dienst Oberösterreich",
			MeasuredAt:  date,
			Temperature: sql.NullFloat64{Float64: temperature, Valid: true},
		})
	}
	return readings
}

type carinthiaResponse struct {
	Data []struct {
		Date    string `json:"datum"`
		Water   string `json:"gewasser"`
		Station string `json:"station"`
		Metric  string `json:"metrics2"`
		Level   string `json:"level"`
	} `json:"data"`
}

func fetchCarinthia(ctx context.Context) ([]Reading, error) {
	var readings []Reading
	for _, source := range []string{urlKtnLakes, urlKtnRivers} {
		body, err := get(ctx, source)
		if err != nil {
			return readings, err
		}
		var response carinthiaResponse
		err = json.NewDecoder(body).Decode(&response)
		_ = body.Close()
		if err != nil {
			return readings, err
		}
		for _, item := range response.Data {
			date, err := parseCarinthiaDate(item.Date)
			if err != nil {
				continue
			}
			raw := item.Metric
			if raw == "" {
				raw = item.Level
			}
			temperature, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", "."), 64)
			if err != nil {
				continue
			}
			name := item.Water + " (" + item.Station + ")"
			readings = append(readings, Reading{
				SourceKey:   "ktn:" + name,
				Name:        name,
				State:       "Kärnten",
				Source:      "Hydrographie Kärnten",
				MeasuredAt:  date,
				Temperature: sql.NullFloat64{Float64: temperature, Valid: true},
			})
		}
	}
	return readings, nil
}

var (
	datePattern                  = regexp.MustCompile(`\((\d{2}\.\d{2}\.\d{4})\)`)
	rowPattern                   = regexp.MustCompile(`(?is)<tr[^>]*>\s*<td[^>]*>(.*?)</td>.*?<td[^>]*>(.*?)</td>\s*</tr>`)
	vowisRowPattern              = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	vowisCellPattern             = regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	vowisStationPattern          = regexp.MustCompile(`(?is)hzbnr=(\d+).*?>([^<]+)</a>\s*/\s*([^<]+)`)
	ausseerlandTeaserPattern     = regexp.MustCompile(`(?is)<article class="infra-event-teaser.*?</article>`)
	ausseerlandLinkPattern       = regexp.MustCompile(`(?is)<a href="([^"]+)"[^>]*>\s*([^<]+?)\s*</a>`)
	ausseerlandDatePattern       = regexp.MustCompile(`Aktualisiert:\s*(\d{2}\.\d{2}\.\d{4})`)
	ausseerlandDetailTempPattern = regexp.MustCompile(`(?is)icon-info-item__value">\s*([0-9,.]+)\s*</span>\s*<span[^>]*>°C</span>.*?Wassertemperatur`)
	tagPattern                   = regexp.MustCompile(`(?is)<[^>]+>`)
)

// fetchAusseerland is best-effort HTML scraping; the source currently exposes no stable API.
func fetchAusseerland(ctx context.Context) ([]Reading, error) {
	body, err := get(ctx, urlAusseerland)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return parseAusseerland(ctx, string(content)), nil
}

func parseAusseerland(ctx context.Context, html string) []Reading {
	var readings []Reading
	for _, article := range ausseerlandTeaserPattern.FindAllString(html, -1) {
		if !strings.Contains(article, "icon-water-temperature") {
			continue
		}
		link := ausseerlandLinkPattern.FindStringSubmatch(article)
		if len(link) != 3 {
			continue
		}
		name := cleanHTML(link[2])
		date, temperature := fetchAusseerlandDetail(ctx, link[1])
		if name == "" || !temperature.Valid {
			continue
		}
		readings = append(readings, Reading{
			SourceKey:   "ausseerland:" + name,
			Name:        name,
			State:       "Steiermark",
			Source:      "Ausseerland Salzkammergut",
			MeasuredAt:  date,
			Temperature: temperature,
		})
	}
	return readings
}

func fetchAusseerlandDetail(ctx context.Context, path string) (time.Time, sql.NullFloat64) {
	date := time.Now()
	if !strings.HasPrefix(path, "http") {
		path = "https://www.steiermark.com" + path
	}
	body, err := get(ctx, path)
	if err != nil {
		return date, sql.NullFloat64{}
	}
	defer body.Close()
	content, err := io.ReadAll(body)
	if err != nil {
		return date, sql.NullFloat64{}
	}
	html := string(content)
	if match := ausseerlandDatePattern.FindStringSubmatch(html); len(match) == 2 {
		if parsed, err := time.Parse("02.01.2006", match[1]); err == nil {
			date = parsed
		}
	}
	if match := ausseerlandDetailTempPattern.FindStringSubmatch(html); len(match) == 2 {
		if temperature, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", "."), 64); err == nil {
			return date, sql.NullFloat64{Float64: temperature, Valid: true}
		}
	}
	return date, sql.NullFloat64{}
}

func fetchVOWIS(ctx context.Context) ([]Reading, error) {
	body, err := get(ctx, urlVOWIS)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return parseVOWIS(string(content)), nil
}

func parseVOWIS(html string) []Reading {
	var readings []Reading
	for _, row := range vowisRowPattern.FindAllStringSubmatch(html, -1) {
		cells := vowisCellPattern.FindAllStringSubmatch(row[1], -1)
		if len(cells) != 6 {
			continue
		}
		station := vowisStationPattern.FindStringSubmatch(cells[0][1])
		if len(station) != 4 {
			continue
		}
		measuredAt, err := time.Parse("02.01.06 15:04", cleanHTML(cells[1][1]))
		if err != nil {
			continue
		}
		temperature, err := strconv.ParseFloat(strings.ReplaceAll(cleanHTML(cells[4][1]), ",", "."), 64)
		if err != nil || temperature < -50 || temperature > 50 {
			continue
		}
		stationName := cleanHTML(station[2])
		waterName := cleanHTML(station[3])
		readings = append(readings, Reading{
			SourceKey:   "vowis:" + station[1],
			Name:        waterName + " (" + stationName + ")",
			State:       "Vorarlberg",
			Source:      "Wasser Online Vorarlberg",
			MeasuredAt:  measuredAt,
			Temperature: sql.NullFloat64{Float64: temperature, Valid: true},
		})
	}
	return readings
}

func cleanHTML(value string) string {
	value = tagPattern.ReplaceAllString(value, "")
	value = strings.ReplaceAll(value, "&nbsp;", " ")
	value = strings.ReplaceAll(value, "&amp;", "&")
	value = strings.ReplaceAll(value, "&lt;", "<")
	value = strings.ReplaceAll(value, "&gt;", ">")
	return strings.TrimSpace(value)
}

// parseCarinthiaDate accepts both Kärnten timestamp formats seen in production.
func parseCarinthiaDate(value string) (time.Time, error) {
	for _, layout := range []string{"02.01.2006 15:04", "02.01.2006 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, &time.ParseError{Layout: "02.01.2006 15:04", Value: value}
}

func get(ctx context.Context, url string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "wassertemperatur.at/1.0")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		_ = response.Body.Close()
		return nil, &httpError{status: response.Status}
	}
	return response.Body, nil
}

type httpError struct{ status string }

func (e *httpError) Error() string { return e.status }
