package session

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SortOrder controls activity ordering.
type SortOrder string

const (
	SortDesc SortOrder = "desc" // most recent activity last (default)
	SortAsc  SortOrder = "asc"  // most recent activity first
)

// EntryType distinguishes focus from park log entries.
type EntryType string

const (
	Focus EntryType = "focus"
	Park  EntryType = "park"
)

const timeLayout = "2006-01-02 15:04"

// Entry is a single line in a session log.
type Entry struct {
	Time time.Time
	Type EntryType
	Note string
}

// FocusPair groups a focus entry with its optional park entry.
type FocusPair struct {
	Focus Entry
	Park  *Entry
}

// Duration returns the time between focus and park, or zero if unparked.
func (fp FocusPair) Duration() time.Duration {
	if fp.Park == nil {
		return 0
	}
	return fp.Park.Time.Sub(fp.Focus.Time)
}

// Activity holds all entries for one session directory.
type Activity struct {
	ID    string      // directory name (e.g. "feature-firewall-hits-routing")
	Pairs []FocusPair // focus/park pairs in chronological order
}

// Name derives a human-readable name from the activity ID.
// Strips common branch prefixes, replaces hyphens with spaces, title-cases.
func (a Activity) Name() string {
	name := a.ID
	for _, prefix := range []string{"feature-", "fix-", "bug-", "chore-", "refactor-", "hotfix-"} {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}
	words := strings.Split(name, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// TotalDuration sums durations of all pairs.
func (a Activity) TotalDuration() time.Duration {
	var total time.Duration
	for _, p := range a.Pairs {
		total += p.Duration()
	}
	return total
}

// ContextSwitches returns the number of focus entries (each focus = one switch).
func (a Activity) ContextSwitches() int {
	return len(a.Pairs)
}

// LatestTime returns the timestamp of the last entry in this activity.
func (a Activity) LatestTime() time.Time {
	if len(a.Pairs) == 0 {
		return time.Time{}
	}
	last := a.Pairs[len(a.Pairs)-1]
	if last.Park != nil {
		return last.Park.Time
	}
	return last.Focus.Time
}

// ParseLogFile reads a session log file and returns parsed entries.
func ParseLogFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		e, err := parseLine(line)
		if err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

// parseLine parses a single log line.
// Format: "2026-03-31 14:03  focus: tuned agentic setup"
// Format: "2026-03-31 14:42  park:  introduced hooks"
func parseLine(line string) (Entry, error) {
	// Minimum: "YYYY-MM-DD HH:MM  X: Y" = 16 + 2 + at least "park: x" = 25
	if len(line) < 25 {
		return Entry{}, fmt.Errorf("line too short")
	}

	ts, err := time.Parse(timeLayout, line[:16])
	if err != nil {
		return Entry{}, fmt.Errorf("bad timestamp: %w", err)
	}

	rest := line[18:] // skip "  " after timestamp
	var typ EntryType
	var note string

	switch {
	case strings.HasPrefix(rest, "focus: "):
		typ = Focus
		note = strings.TrimPrefix(rest, "focus: ")
	case strings.HasPrefix(rest, "park:  "):
		typ = Park
		note = strings.TrimPrefix(rest, "park:  ")
	case strings.HasPrefix(rest, "park: "):
		typ = Park
		note = strings.TrimPrefix(rest, "park: ")
	default:
		return Entry{}, fmt.Errorf("unknown entry type in: %s", rest)
	}

	return Entry{Time: ts, Type: typ, Note: note}, nil
}

// BuildPairs groups entries into focus/park pairs.
func BuildPairs(entries []Entry) []FocusPair {
	var pairs []FocusPair
	var current *FocusPair

	for i := range entries {
		e := entries[i]
		switch e.Type {
		case Focus:
			if current != nil {
				pairs = append(pairs, *current)
			}
			current = &FocusPair{Focus: e}
		case Park:
			if current != nil {
				current.Park = &e
				pairs = append(pairs, *current)
				current = nil
			}
		}
	}
	if current != nil {
		pairs = append(pairs, *current)
	}
	return pairs
}

// ReadAllActivities reads all session directories and returns activities
// filtered to entries within [first, last], sorted by the given order.
func ReadAllActivities(sessionsDir string, first, last time.Time, order SortOrder) ([]Activity, error) {
	dirEntries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("reading sessions dir: %w", err)
	}

	var activities []Activity
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		logPath := filepath.Join(sessionsDir, de.Name(), "log")
		entries, err := ParseLogFile(logPath)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", logPath, err)
		}
		if len(entries) == 0 {
			continue
		}

		// Filter entries to the time window.
		var filtered []Entry
		for _, e := range entries {
			if !e.Time.Before(first) && e.Time.Before(last) {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			continue
		}

		pairs := BuildPairs(filtered)
		if len(pairs) == 0 {
			continue
		}

		activities = append(activities, Activity{
			ID:    de.Name(),
			Pairs: pairs,
		})
	}

	sort.Slice(activities, func(i, j int) bool {
		if order == SortAsc {
			return activities[i].LatestTime().After(activities[j].LatestTime())
		}
		// desc: most recent last
		return activities[i].LatestTime().Before(activities[j].LatestTime())
	})

	return activities, nil
}
