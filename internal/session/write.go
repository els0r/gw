package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteEntry appends a log entry and updates the state file (focus or park).
func WriteEntry(sessionsDir, branch string, typ EntryType, note string) error {
	dirName := sanitizeBranch(branch)
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

	return nil
}

// sanitizeBranch converts branch names to directory-safe names.
// Matches gw.sh: ${1//\//-}
func sanitizeBranch(branch string) string {
	out := make([]byte, len(branch))
	for i := range branch {
		if branch[i] == '/' {
			out[i] = '-'
		} else {
			out[i] = branch[i]
		}
	}
	return string(out)
}
