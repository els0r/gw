package render

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0m"},
		{-5 * time.Minute, "0m"},
		{1 * time.Minute, "1m"},
		{30 * time.Minute, "30m"},
		{59 * time.Minute, "59m"},
		{60 * time.Minute, "1h"},
		{90 * time.Minute, "1h30m"},
		{120 * time.Minute, "2h"},
		{145 * time.Minute, "2h25m"},
		// Rounding: 30 seconds rounds to nearest minute.
		{89*time.Minute + 30*time.Second, "1h30m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatDuration(tt.d); got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestContextSwitchColor(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{1, green},
		{2, green},  // below 3 → green
		{3, orange},
		{4, orange},
		{5, red},
		{10, red},
	}

	for _, tt := range tests {
		if got := contextSwitchColor(tt.n); got != tt.want {
			t.Errorf("contextSwitchColor(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestPluralize(t *testing.T) {
	if got := pluralize("switch", "switches", 1); got != "switch" {
		t.Errorf("got %q, want switch", got)
	}
	if got := pluralize("switch", "switches", 3); got != "switches" {
		t.Errorf("got %q, want switches", got)
	}
	if got := pluralize("switch", "switches", 0); got != "switches" {
		t.Errorf("got %q, want switches", got)
	}
}
