package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestParseDateTimeArg(t *testing.T) {
	now := time.Date(2026, 10, 1, 13, 45, 0, 0, time.Local)
	tests := map[string]string{
		"":                 "",
		"none":             "",
		"now":              "2026-10-01 13:45",
		"today":            "2026-10-01 00:00",
		"tomorrow":         "2026-10-02 00:00",
		"2026-10-15":       "2026-10-15 00:00",
		"2026-10-15 09:30": "2026-10-15 09:30",
		"2026-10-15T09:30": "2026-10-15 09:30",
	}
	for in, want := range tests {
		got, err := parseDateTimeArg(in, now)
		if err != nil || got != want {
			t.Errorf("parseDateTimeArg(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"15.10.2026", "2026-13-01", "soon"} {
		if _, err := parseDateTimeArg(bad, now); err == nil {
			t.Errorf("parseDateTimeArg(%q) succeeded, want error", bad)
		}
	}
}

func TestParseHours(t *testing.T) {
	tests := map[string]float64{
		"": 0, "none": 0, "1.5": 1.5, "2": 2, "1.5h": 1.5, "90m": 1.5, "1h30m": 1.5, "20m": 0.33,
	}
	for in, want := range tests {
		got, err := parseHours(in)
		if err != nil || got != want {
			t.Errorf("parseHours(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"-1", "abc", "1x"} {
		if _, err := parseHours(bad); err == nil {
			t.Errorf("parseHours(%q) succeeded, want error", bad)
		}
	}
}

func TestParseRecurrenceEvery(t *testing.T) {
	tests := []struct {
		in                string
		factor, timeframe int
	}{
		{"3d", 3, recurrenceTimeframeDays},
		{"2 months", 2, recurrenceTimeframeMonths},
		{"1y", 1, recurrenceTimeframeYears},
	}
	for _, tt := range tests {
		f, tf, err := parseRecurrenceEvery(tt.in)
		if err != nil || f != tt.factor || tf != tt.timeframe {
			t.Errorf("parseRecurrenceEvery(%q) = %d, %d, %v", tt.in, f, tf, err)
		}
		if got := recurrenceEveryLabel(f, tf); got[0] != tt.in[0] {
			t.Errorf("recurrenceEveryLabel(%d, %d) = %q", f, tf, got)
		}
	}
	for _, bad := range []string{"", "d", "0d", "3w", "x3d"} {
		if _, _, err := parseRecurrenceEvery(bad); err == nil {
			t.Errorf("parseRecurrenceEvery(%q) succeeded, want error", bad)
		}
	}
}

func TestParseColor(t *testing.T) {
	for in, want := range map[string]string{
		"Red": "red", "deep-orange": "deep_orange", "dark gray": "dark_grey", "gray": "grey",
	} {
		if got, err := parseColor(in); err != nil || got != want {
			t.Errorf("parseColor(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := parseColor("ultraviolet"); err == nil {
		t.Error("parseColor accepted an unknown color")
	}
}

func TestMatchName(t *testing.T) {
	names := map[string]string{"1": "Alice Smith", "2": "Bob Builder", "3": "Bobby Tables"}
	tests := []struct {
		in      string
		want    int
		wantErr string
	}{
		{"alice smith", 1, ""},
		{"builder", 2, ""},
		{"bob builder", 2, ""},
		{"bob", 0, "ambiguous"},
		{"carol", 0, "not found"},
	}
	for _, tt := range tests {
		got, err := matchName(tt.in, names, "user")
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("matchName(%q) err = %v, want %q", tt.in, err, tt.wantErr)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("matchName(%q) = %d, %v; want %d", tt.in, got, err, tt.want)
		}
	}
}

func TestSummarizeText(t *testing.T) {
	if got := summarizeText(""); got != "(empty)" {
		t.Errorf("got %q", got)
	}
	if got := summarizeText("line one\nline two"); got != `"line one ..." (17 chars)` {
		t.Errorf("got %q", got)
	}
}
