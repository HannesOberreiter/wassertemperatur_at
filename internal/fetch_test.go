package internal

import (
	"testing"
	"time"
)

func TestParseCarinthiaDate(t *testing.T) {
	for _, value := range []string{"15.06.2026 13:00", "15.06.2026 13:00:00"} {
		parsed, err := parseCarinthiaDate(value)
		if err != nil {
			t.Fatalf("%s: %v", value, err)
		}
		if parsed.Year() != 2026 || parsed.Month() != 6 || parsed.Day() != 15 || parsed.Hour() != 13 {
			t.Fatalf("unexpected date %v", parsed)
		}
	}
}

func TestParseVOWIS(t *testing.T) {
	html := `<table><tbody>
<tr><td><a href=javascript:open("?hzbnr=200329&x=1")>Kennelbach, 200329</a> / Bregenzerach</td><td>18.07.26 13:05</td><td>7,03</td><td>90</td><td>21,5</td><td>MJNQ-MQ</td></tr>
<tr><td><a href=javascript:open("?hzbnr=200014&x=1")>Bangs, 200014</a> / Rhein</td><td>18.07.26 13:05</td><td>90,8</td><td>598</td><td></td><td>MJNQ-MQ</td></tr>
</tbody></table>`

	readings := parseVOWIS(html)
	if len(readings) != 1 {
		t.Fatalf("got %d readings, want 1", len(readings))
	}
	reading := readings[0]
	if reading.SourceKey != "vowis:200329" || reading.Name != "Bregenzerach (Kennelbach, 200329)" {
		t.Fatalf("unexpected station: %#v", reading)
	}
	if reading.State != "Vorarlberg" || reading.Source != "Wasser Online Vorarlberg" {
		t.Fatalf("unexpected source: %#v", reading)
	}
	if !reading.Temperature.Valid || reading.Temperature.Float64 != 21.5 {
		t.Fatalf("unexpected temperature: %#v", reading.Temperature)
	}
	wantTime := time.Date(2026, time.July, 18, 13, 5, 0, 0, time.UTC)
	if !reading.MeasuredAt.Equal(wantTime) {
		t.Fatalf("got time %v, want %v", reading.MeasuredAt, wantTime)
	}
}

func TestParseVOWISLake(t *testing.T) {
	html := `<table id="seemesswerte"><tbody><tr>
<td><span>Aktueller Wert</span><br><span class="fs-8">18.07.2026 12:20</span></td>
<td>315 cm</td><td>25,4 °C</td><td>24,9 °C</td>
</tr></tbody></table>`

	readings := parseVOWISLake(html)
	if len(readings) != 1 {
		t.Fatalf("got %d readings, want 1", len(readings))
	}
	reading := readings[0]
	if reading.SourceKey != "vowis:200337" || reading.Name != "Bodensee (Bregenz)" {
		t.Fatalf("unexpected station: %#v", reading)
	}
	if !reading.Temperature.Valid || reading.Temperature.Float64 != 25.4 {
		t.Fatalf("unexpected temperature: %#v", reading.Temperature)
	}
	wantTime := time.Date(2026, time.July, 18, 12, 20, 0, 0, time.UTC)
	if !reading.MeasuredAt.Equal(wantTime) {
		t.Fatalf("got time %v, want %v", reading.MeasuredAt, wantTime)
	}
}
