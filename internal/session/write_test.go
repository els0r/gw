package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature/foo", "feature-foo"},
		{"fix/bar/baz", "fix-bar-baz"},
		{"no-slashes", "no-slashes"},
		{"main", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeBranch(tt.input); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteEntry(t *testing.T) {
	dir := t.TempDir()

	// Write a focus entry.
	if err := WriteEntry(dir, "feature/test", Focus, "started work"); err != nil {
		t.Fatal(err)
	}

	sessionDir := filepath.Join(dir, "feature-test")

	// Verify log file was created with correct format.
	logData, err := os.ReadFile(filepath.Join(sessionDir, "log"))
	if err != nil {
		t.Fatal(err)
	}
	logLine := string(logData)
	if !strings.Contains(logLine, "  focus: started work\n") {
		t.Errorf("log line format wrong: %q", logLine)
	}

	// Verify state file.
	stateData, err := os.ReadFile(filepath.Join(sessionDir, "focus"))
	if err != nil {
		t.Fatal(err)
	}
	if string(stateData) != "started work\n" {
		t.Errorf("focus state: got %q, want %q", string(stateData), "started work\n")
	}

	// Write a park entry — should append to same log.
	if err := WriteEntry(dir, "feature/test", Park, "left off here"); err != nil {
		t.Fatal(err)
	}

	logData, err = os.ReadFile(filepath.Join(sessionDir, "log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "  park:  left off here") {
		t.Errorf("park line format wrong: %q", lines[1])
	}

	// Verify park state file.
	stateData, err = os.ReadFile(filepath.Join(sessionDir, "park"))
	if err != nil {
		t.Fatal(err)
	}
	if string(stateData) != "left off here\n" {
		t.Errorf("park state: got %q", string(stateData))
	}
}
