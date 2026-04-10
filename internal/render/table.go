package render

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/els0r/gw/internal/session"
)

// ansi color codes
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	italic = "\033[3m"
	dim    = "\033[2m"

	green  = "\033[32m"
	orange = "\033[33m"
	red    = "\033[31m"
)

// NameFunc resolves a display name for an activity.
type NameFunc func(a session.Activity) string

// Activities renders all activities to stdout in the table format.
func Activities(activities []session.Activity, nameFunc NameFunc) {
	for i, a := range activities {
		if i > 0 {
			fmt.Println()
		}
		activity(a, nameFunc)
	}
}

func activity(a session.Activity, nameFunc NameFunc) {
	name := nameFunc(a)
	dur := a.TotalDuration()
	switches := a.ContextSwitches()

	// header: activity name (italic)
	fmt.Printf("  %s%s%s\n", italic, name, reset)

	// duration + context switches
	switchColor := contextSwitchColor(switches)
	fmt.Printf("  %s%s%s%s    %s%s%d%s context %s\n",
		dim, bold, formatDuration(dur), reset,
		switchColor, bold, switches, reset,
		pluralize("switch", "switches", switches))

	fmt.Println()

	// focus/park pairs
	var lastDate string
	for _, p := range a.Pairs {
		// focus line: timestamp ○ note
		date := p.Focus.Time.Format("2006-01-02")
		timeOnly := p.Focus.Time.Format("15:04")

		var ts string
		if date != lastDate {
			ts = bold + date + " " + timeOnly + reset
			lastDate = date
		} else {
			ts = strings.Repeat(" ", len(date)+1) + timeOnly
		}
		fmt.Printf("  %s  %s○ %s%s\n", ts, dim, p.Focus.Note, reset)

		// park line: +Xm └── note
		if p.Park != nil {
			d := p.Duration()
			dStr := formatDuration(d)
			// right-align duration under the timestamp area
			pad := strings.Repeat(" ", 18-2-len(dStr))
			fmt.Printf("  %s%s%s  %s└── %s%s\n", pad, dim, dStr, reset, p.Park.Note, reset)
		}
		fmt.Println()
	}
}

func contextSwitchColor(n int) string {
	switch {
	case n == 1:
		return green
	case n >= 3 && n < 5:
		return orange
	case n >= 5:
		return red
	default:
		return green
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}
	totalMinutes := int(math.Round(d.Minutes()))
	if totalMinutes < 60 {
		return fmt.Sprintf("%dm", totalMinutes)
	}
	h := totalMinutes / 60
	m := totalMinutes % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}

func pluralize(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}
