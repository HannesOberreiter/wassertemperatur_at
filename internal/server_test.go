package internal

import (
	"bytes"
	"testing"
)

func TestTemplatesRender(t *testing.T) {
	tpl := templates()
	data := PageData{Title: "Test", Description: "Test", States: []string{"Kärnten"}}
	for _, name := range []string{"index", "table", "detail"} {
		var buf bytes.Buffer
		if err := tpl.ExecuteTemplate(&buf, name, data); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
