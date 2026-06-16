package internal

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	db        *sql.DB
	templates *template.Template
}

type PageData struct {
	Title         string
	Description   string
	Waters        []Water
	Water         Water
	Observations  []Observation
	Summaries     []DailySummary
	Search        string
	State         string
	States        []string
	Now           time.Time
	Partial       bool
	QualityPage   bool
	QualityDetail bool
	Page          int
	PrevPage      int
	NextPage      int
}

func NewServer(db *sql.DB) http.Handler {
	server := &Server{db: db, templates: templates()}
	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	mux.HandleFunc("/", server.index)
	mux.HandleFunc("/table", server.table)
	mux.HandleFunc("/wasserqualitaet", server.quality)
	mux.HandleFunc("/qualitaet-tabelle", server.qualityTable)
	mux.HandleFunc("/gewaesser/", server.detail)
	mux.HandleFunc("/impressum", server.imprint)
	return mux
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	data, err := s.pageData(r, false)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		slog.Error("Data could not be loaded", "error", err)
		http.Error(w, "Daten konnten nicht geladen werden", http.StatusInternalServerError)
		return
	}
	data.Title = "Wassertemperaturen Österreich: Seen, Flüsse und Badegewässer"
	data.Description = "Aktuelle Wassertemperaturen österreichischer Seen, Flüsse und Badegewässer mit Verlauf und Tagesänderung."
	s.render(w, "index", data)
}

func (s *Server) quality(w http.ResponseWriter, r *http.Request) {
	data, err := s.pageData(r, true)
	if err != nil {
		http.Error(w, "Daten konnten nicht geladen werden", http.StatusInternalServerError)
		return
	}
	data.Title = "Wasserqualität österreichischer Badegewässer"
	data.Description = "Aktuelle Wasserqualität österreichischer Badegewässer mit Sichttiefe und Bewertung."
	data.QualityPage = true
	s.render(w, "quality", data)
}

func (s *Server) table(w http.ResponseWriter, r *http.Request) {
	data, err := s.pageData(r, false)
	if err != nil {
		http.Error(w, "Daten konnten nicht geladen werden", http.StatusInternalServerError)
		return
	}
	data.Partial = true
	s.render(w, "table", data)
}

func (s *Server) qualityTable(w http.ResponseWriter, r *http.Request) {
	data, err := s.pageData(r, true)
	if err != nil {
		http.Error(w, "Daten konnten nicht geladen werden", http.StatusInternalServerError)
		return
	}
	data.Partial = true
	data.QualityPage = true
	s.render(w, "qualityTable", data)
}

func (s *Server) detail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/gewaesser/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	water, err := GetWater(r.Context(), s.db, parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	page := pageNumber(r)
	data := PageData{
		Title:       "Wassertemperatur " + water.Name,
		Description: "Historische Wassertemperaturen für " + water.Name + " in " + water.State + ".",
		Water:       water,
		Now:         time.Now(),
		Page:        page,
		PrevPage:    page - 1,
		NextPage:    page + 1,
	}
	if water.Source == "AGES Badegewässerdatenbank" {
		observations, err := LatestObservations(r.Context(), s.db, water.ID, 30)
		if err != nil {
			http.Error(w, "Verlauf konnte nicht geladen werden", http.StatusInternalServerError)
			return
		}
		data.Observations = observations
		data.QualityDetail = true
	} else {
		summaries, err := DailySummaries(r.Context(), s.db, water.ID, 30, (page-1)*30)
		if err != nil {
			http.Error(w, "Verlauf konnte nicht geladen werden", http.StatusInternalServerError)
			return
		}
		data.Summaries = summaries
	}
	s.render(w, "detail", data)
}

func (s *Server) imprint(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "imprint", PageData{Title: "Impressum", Description: "Impressum von wassertemperatur.at"})
}

func (s *Server) pageData(r *http.Request, quality bool) (PageData, error) {
	search := r.URL.Query().Get("suche")
	state := r.URL.Query().Get("bundesland")
	var waters []Water
	var err error
	if quality {
		waters, err = ListQualityWaters(r.Context(), s.db, search, state)
	} else {
		waters, err = ListWaters(r.Context(), s.db, search, state)
	}
	if err != nil {
		return PageData{}, err
	}
	source := "!AGES Badegewässerdatenbank"
	if quality {
		source = "AGES Badegewässerdatenbank"
	}
	availableStates, err := ListStates(r.Context(), s.db, source)
	if err != nil {
		return PageData{}, err
	}
	return PageData{Waters: waters, Search: search, State: state, States: availableStates, Now: time.Now(), QualityPage: quality}, nil
}

func (s *Server) render(w http.ResponseWriter, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Templatefehler", http.StatusInternalServerError)
	}
}

func pageNumber(r *http.Request) int {
	page, err := strconv.Atoi(r.URL.Query().Get("seite"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func templates() *template.Template {
	funcs := template.FuncMap{
		"date": func(t sql.NullTime) string {
			if !t.Valid {
				return ""
			}
			return t.Time.Format("02.01.2006 15:04")
		},
		"day":       func(t time.Time) string { return t.Format("02.01.2006") },
		"plainTemp": func(v float64) string { return fmtPlainFloat(v, " °C") },
		"temp":      func(v sql.NullFloat64) string { return fmtFloat(v, " °C") },
		"depth":     func(v sql.NullFloat64) string { return fmtFloat(v, " m") },
		"quality": func(v sql.NullInt64) string {
			if !v.Valid {
				return ""
			}
			return strconv.FormatInt(v.Int64, 10)
		},
		"param": func(op string, v sql.NullInt64) string {
			if !v.Valid {
				return ""
			}
			return op + strconv.FormatInt(v.Int64, 10)
		},
		"change": func(v sql.NullFloat64) string {
			if !v.Valid {
				return ""
			}
			icon := "➡️"
			if v.Float64 > 0 {
				icon = "🔺"
			} else if v.Float64 < 0 {
				icon = "🔻"
			}
			return icon + " " + fmtFloat(v, " %")
		},
		"detailURL": func(w Water) string { return "/gewaesser/" + strconv.FormatInt(w.ID, 10) + "/" + w.Slug },
	}
	return template.Must(template.New("pages").Funcs(funcs).Parse(pageTemplates))
}
