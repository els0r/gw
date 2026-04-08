package cmd

import (
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

// resolveRange maps a named range to [first, last) boundaries.
func resolveRange(name string, today time.Time) (first, last time.Time, ok bool) {
	switch name {
	case "today":
		return today, today.Add(24 * time.Hour), true
	case "yesterday":
		y := today.AddDate(0, 0, -1)
		return y, today, true
	case "this week":
		wd := today.Weekday()
		if wd == time.Sunday {
			wd = 7
		}
		monday := today.AddDate(0, 0, -int(wd-time.Monday))
		return monday, today.Add(24 * time.Hour), true
	case "last week":
		wd := today.Weekday()
		if wd == time.Sunday {
			wd = 7
		}
		thisMonday := today.AddDate(0, 0, -int(wd-time.Monday))
		lastMonday := thisMonday.AddDate(0, 0, -7)
		return lastMonday, thisMonday, true
	case "this year":
		first = time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, today.Location())
		return first, today.Add(24 * time.Hour), true
	case "this quarter":
		first = quarterStart(today)
		return first, today.Add(24 * time.Hour), true
	case "last quarter":
		first, last = lastNQuartersRange(today, 1)
		return first, last, true
	default:
		var n int
		if _, err := fmt.Sscanf(name, "last %d weeks", &n); err == nil && n >= 1 {
			wd := today.Weekday()
			if wd == time.Sunday {
				wd = 7
			}
			thisMonday := today.AddDate(0, 0, -int(wd-time.Monday))
			return thisMonday.AddDate(0, 0, -7*n), thisMonday, true
		}
		if _, err := fmt.Sscanf(name, "last %d quarters", &n); err == nil && n >= 1 {
			first, last = lastNQuartersRange(today, n)
			return first, last, true
		}
		return time.Time{}, time.Time{}, false
	}
}

// quarterStart returns the first day of the calendar quarter containing t.
func quarterStart(t time.Time) time.Time {
	month := int(t.Month())
	qStartMonth := time.Month(((month-1)/3)*3 + 1)
	return time.Date(t.Year(), qStartMonth, 1, 0, 0, 0, 0, t.Location())
}

// lastNQuartersRange returns the [first, last) range covering the n complete
// calendar quarters immediately preceding the current quarter.
func lastNQuartersRange(today time.Time, n int) (first, last time.Time) {
	last = quarterStart(today)
	first = last.AddDate(0, -3*n, 0)
	return
}
