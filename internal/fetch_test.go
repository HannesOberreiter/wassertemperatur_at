package internal

import "testing"

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
