package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ResolveFunc maps an activity ID to a display name.
// Returns empty string when resolution is unavailable.
type ResolveFunc func(activityID string) string

// WriteEntry appends a log entry and updates the state file (focus or park).
// If activityID is non-empty and resolve returns a name, the sanitized name is
// used as the session directory. Otherwise falls back to the sanitized branch.
func WriteEntry(sessionsDir, branch string, typ EntryType, note, activityID string, resolve ResolveFunc) error {
	dirName := SanitizeName(branch)
	if activityID != "" && resolve != nil {
		if name := resolve(activityID); name != "" {
			dirName = SanitizeName(name)
		}
	}

	dir := filepath.Join(sessionsDir, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating session dir: %w", err)
	}

	ts := time.Now().Format(timeLayout)

	// format matches existing convention:
	//   "2026-03-31 14:03  focus: note"
	//   "2026-03-31 14:42  park:  note"
	var line string
	switch typ {
	case Focus:
		line = fmt.Sprintf("%s  focus: %s\n", ts, note)
	case Park:
		line = fmt.Sprintf("%s  park:  %s\n", ts, note)
	default:
		return fmt.Errorf("unknown entry type: %s", typ)
	}

	// append to log
	logPath := filepath.Join(dir, "log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening log: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("writing log: %w", err)
	}

	// update state file
	stateFile := filepath.Join(dir, string(typ))
	if err := os.WriteFile(stateFile, []byte(note+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	// persist activity ID when provided
	if activityID != "" {
		activityFile := filepath.Join(dir, "activity")
		if err := os.WriteFile(activityFile, []byte(activityID+"\n"), 0o644); err != nil {
			return fmt.Errorf("writing activity file: %w", err)
		}
	}

	return nil
}

// SanitizeName converts a string to a directory-safe session name.
// Only [a-z0-9_-] characters are kept. Slashes and spaces become hyphens,
// uppercase is lowered, everything else is dropped. Consecutive hyphens are
// collapsed and leading/trailing hyphens trimmed.
func SanitizeName(name string) string {
	out := make([]byte, 0, len(name))
	for i := range name {
		c := name[i]
		switch {
		case c == '/' || c == ' ':
			c = '-'
		case c >= 'A' && c <= 'Z':
			c = c + 32 // lowercase
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_':
			// keep as-is
		default:
			continue // drop everything else
		}
		// collapse consecutive hyphens
		if c == '-' && len(out) > 0 && out[len(out)-1] == '-' {
			continue
		}
		out = append(out, c)
	}
	// trim leading/trailing hyphens
	s := string(out)
	for len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == '-' {
		s = s[:len(s)-1]
	}
	return s
}
