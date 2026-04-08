package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func TestResolveRange(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		today     time.Time
		wantFirst time.Time
		wantLast  time.Time
		wantOK    bool
	}{
		{
			name:      "today",
			input:     "today",
			today:     date(2026, 4, 1),
			wantFirst: date(2026, 4, 1),
			wantLast:  date(2026, 4, 2),
			wantOK:    true,
		},
		{
			name:      "yesterday",
			input:     "yesterday",
			today:     date(2026, 4, 1),
			wantFirst: date(2026, 3, 31),
			wantLast:  date(2026, 4, 1),
			wantOK:    true,
		},
		{
			name:      "this week on wednesday",
			input:     "this week",
			today:     date(2026, 4, 1), // Wednesday
			wantFirst: date(2026, 3, 30), // Monday
			wantLast:  date(2026, 4, 2),  // today + 1 day
			wantOK:    true,
		},
		{
			name:      "this week on monday",
			input:     "this week",
			today:     date(2026, 3, 30), // Monday
			wantFirst: date(2026, 3, 30),
			wantLast:  date(2026, 3, 31),
			wantOK:    true,
		},
		{
			name:      "this week on sunday",
			input:     "this week",
			today:     date(2026, 4, 5), // Sunday
			wantFirst: date(2026, 3, 30), // Monday of that week
			wantLast:  date(2026, 4, 6),
			wantOK:    true,
		},
		{
			name:      "last week on wednesday",
			input:     "last week",
			today:     date(2026, 4, 1), // Wednesday
			wantFirst: date(2026, 3, 23), // previous Monday
			wantLast:  date(2026, 3, 30), // this Monday
			wantOK:    true,
		},
		{
			name:      "last week on sunday",
			input:     "last week",
			today:     date(2026, 4, 5), // Sunday
			wantFirst: date(2026, 3, 23),
			wantLast:  date(2026, 3, 30),
			wantOK:    true,
		},
		// this year
		{
			name:      "this year",
			input:     "this year",
			today:     date(2026, 4, 8),
			wantFirst: date(2026, 1, 1),
			wantLast:  date(2026, 4, 9),
			wantOK:    true,
		},
		// last quarter
		{
			name:      "last quarter from Q2",
			input:     "last quarter",
			today:     date(2026, 4, 8), // Q2 → last = Q1 2026
			wantFirst: date(2026, 1, 1),
			wantLast:  date(2026, 4, 1),
			wantOK:    true,
		},
		{
			name:      "last quarter from Q1",
			input:     "last quarter",
			today:     date(2026, 2, 15), // Q1 → last = Q4 2025
			wantFirst: date(2025, 10, 1),
			wantLast:  date(2026, 1, 1),
			wantOK:    true,
		},
		// last N weeks
		{
			name:      "last 2 weeks on wednesday",
			input:     "last 2 weeks",
			today:     date(2026, 4, 1), // Wed; thisMonday = Mar 30
			wantFirst: date(2026, 3, 16),
			wantLast:  date(2026, 3, 30),
			wantOK:    true,
		},
		{
			name:      "last 3 weeks on sunday",
			input:     "last 3 weeks",
			today:     date(2026, 4, 5), // Sun; thisMonday = Mar 30
			wantFirst: date(2026, 3, 9),
			wantLast:  date(2026, 3, 30),
			wantOK:    true,
		},
		{
			name:      "last 1 weeks resolves like last week",
			input:     "last 1 weeks",
			today:     date(2026, 4, 1), // Wed; thisMonday = Mar 30
			wantFirst: date(2026, 3, 23),
			wantLast:  date(2026, 3, 30),
			wantOK:    true,
		},
		// last N quarters
		{
			name:      "last 2 quarters from Q2 2026",
			input:     "last 2 quarters",
			today:     date(2026, 4, 8), // Q2 → last = Apr 1; first = Oct 1 2025
			wantFirst: date(2025, 10, 1),
			wantLast:  date(2026, 4, 1),
			wantOK:    true,
		},
		{
			name:      "last 3 quarters from Q2 2026",
			input:     "last 3 quarters",
			today:     date(2026, 4, 8), // Q2 → last = Apr 1; first = Jul 1 2025
			wantFirst: date(2025, 7, 1),
			wantLast:  date(2026, 4, 1),
			wantOK:    true,
		},
		{
			name:      "last 1 quarters resolves like last quarter",
			input:     "last 1 quarters",
			today:     date(2026, 4, 8),
			wantFirst: date(2026, 1, 1),
			wantLast:  date(2026, 4, 1),
			wantOK:    true,
		},
		// this quarter
		{
			name:      "this quarter from Q2",
			input:     "this quarter",
			today:     date(2026, 4, 8), // Q2 starts Apr 1
			wantFirst: date(2026, 4, 1),
			wantLast:  date(2026, 4, 9),
			wantOK:    true,
		},
		{
			name:      "this quarter from Q1",
			input:     "this quarter",
			today:     date(2026, 2, 15), // Q1 starts Jan 1
			wantFirst: date(2026, 1, 1),
			wantLast:  date(2026, 2, 16),
			wantOK:    true,
		},
		// unknown
		{
			name:   "unknown range",
			input:  "next month",
			today:  date(2026, 4, 1),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			first, last, ok := resolveRange(tt.input, tt.today)
			require.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				return
			}
			require.True(t, first.Equal(tt.wantFirst), "first: got %v, want %v", first, tt.wantFirst)
			require.True(t, last.Equal(tt.wantLast), "last: got %v, want %v", last, tt.wantLast)
		})
	}
}
