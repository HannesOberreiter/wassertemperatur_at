package internal

import (
	"bytes"
	"strings"
	"testing"
)

func TestTemplatesRender(t *testing.T) {
	tpl := templates()
	data := PageData{Title: "Test", Description: "Test", States: []string{"Kärnten"}}
	for _, name := range []string{"index", "table", "quality", "qualityTable", "detail"} {
		var buf bytes.Buffer
		if err := tpl.ExecuteTemplate(&buf, name, data); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestTemplatesShowSource(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     PageData
		want     string
	}{
		{"table", "table", PageData{Waters: []Water{{Name: "See", Source: "Hydrographie Kärnten"}}}, "Quelle: Hydrographie Kärnten"},
		{"quality table", "qualityTable", PageData{QualityPage: true}, "Quelle: AGES Badegewässerdatenbank"},
		{"detail", "detail", PageData{Water: Water{Name: "See", Source: "Hydrographie Kärnten"}}, "Quelle: Hydrographie Kärnten"},
		{"quality detail", "detail", PageData{QualityDetail: true, Water: Water{Name: "See", Source: agesSource}}, "Quelle: AGES Badegewässerdatenbank"},
	}
	tpl := templates()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tpl.ExecuteTemplate(&buf, tt.template, tt.data); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Fatalf("output misses %q", tt.want)
			}
		})
	}
}

func TestTemplatesHideEmptySource(t *testing.T) {
	var buf bytes.Buffer
	data := PageData{Water: Water{Name: "See"}}
	if err := templates().ExecuteTemplate(&buf, "detail", data); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "Quelle:") {
		t.Fatal("detail page shows empty source")
	}
}
