package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustTime(s string) time.Time {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    Entry
		wantErr bool
	}{
		{
			name: "focus entry",
			line: "2026-03-31 14:03  focus: tuned agentic setup",
			want: Entry{Time: mustTime("2026-03-31 14:03"), Type: Focus, Note: "tuned agentic setup"},
		},
		{
			name: "park entry with double space",
			line: "2026-03-31 14:42  park:  introduced hooks",
			want: Entry{Time: mustTime("2026-03-31 14:42"), Type: Park, Note: "introduced hooks"},
		},
		{
			name: "park entry with single space",
			line: "2026-03-31 14:42  park: single space note",
			want: Entry{Time: mustTime("2026-03-31 14:42"), Type: Park, Note: "single space note"},
		},
		{
			name:    "too short",
			line:    "2026-03-31 14:03",
			wantErr: true,
		},
		{
			name:    "bad timestamp",
			line:    "not-a-date 14:03  focus: something",
			wantErr: true,
		},
		{
			name:    "unknown type",
			line:    "2026-03-31 14:03  break: coffee time",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLine(tt.line)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Time != tt.want.Time {
				t.Errorf("time: got %v, want %v", got.Time, tt.want.Time)
			}
			if got.Type != tt.want.Type {
				t.Errorf("type: got %q, want %q", got.Type, tt.want.Type)
			}
			if got.Note != tt.want.Note {
				t.Errorf("note: got %q, want %q", got.Note, tt.want.Note)
			}
		})
	}
}

func TestBuildPairs(t *testing.T) {
	f1 := Entry{Time: mustTime("2026-03-31 10:00"), Type: Focus, Note: "first focus"}
	p1 := Entry{Time: mustTime("2026-03-31 10:30"), Type: Park, Note: "first park"}
	f2 := Entry{Time: mustTime("2026-03-31 11:00"), Type: Focus, Note: "second focus"}
	p2 := Entry{Time: mustTime("2026-03-31 11:45"), Type: Park, Note: "second park"}
	f3 := Entry{Time: mustTime("2026-03-31 12:00"), Type: Focus, Note: "unpaired focus"}

	tests := []struct {
		name      string
		entries   []Entry
		wantPairs int
		check     func(t *testing.T, pairs []FocusPair)
	}{
		{
			name:      "empty",
			entries:   nil,
			wantPairs: 0,
		},
		{
			name:      "single focus-park pair",
			entries:   []Entry{f1, p1},
			wantPairs: 1,
			check: func(t *testing.T, pairs []FocusPair) {
				if pairs[0].Park == nil {
					t.Fatal("expected park to be set")
				}
				if pairs[0].Duration() != 30*time.Minute {
					t.Errorf("duration: got %v, want 30m", pairs[0].Duration())
				}
			},
		},
		{
			name:      "two complete pairs",
			entries:   []Entry{f1, p1, f2, p2},
			wantPairs: 2,
		},
		{
			name:      "trailing unpaired focus",
			entries:   []Entry{f1, p1, f3},
			wantPairs: 2,
			check: func(t *testing.T, pairs []FocusPair) {
				if pairs[1].Park != nil {
					t.Fatal("second pair should have nil park")
				}
				if pairs[1].Duration() != 0 {
					t.Errorf("unpaired duration: got %v, want 0", pairs[1].Duration())
				}
			},
		},
		{
			name:      "consecutive focuses without park",
			entries:   []Entry{f1, f2, p2},
			wantPairs: 2,
			check: func(t *testing.T, pairs []FocusPair) {
				if pairs[0].Park != nil {
					t.Fatal("first pair should have nil park (interrupted)")
				}
				if pairs[1].Park == nil {
					t.Fatal("second pair should have park")
				}
			},
		},
		{
			name:      "orphan park is ignored",
			entries:   []Entry{p1},
			wantPairs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairs := BuildPairs(tt.entries)
			if len(pairs) != tt.wantPairs {
				t.Fatalf("got %d pairs, want %d", len(pairs), tt.wantPairs)
			}
			if tt.check != nil {
				tt.check(t, pairs)
			}
		})
	}
}

func TestActivityName(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"feature-firewall-hits-routing", "Firewall Hits Routing"},
		{"fix-null-pointer", "Null Pointer"},
		{"bug-login-crash", "Login Crash"},
		{"chore-deps-update", "Deps Update"},
		{"refactor-auth-flow", "Auth Flow"},
		{"hotfix-prod-outage", "Prod Outage"},
		{"admin-misc", "Admin Misc"},       // no prefix stripped
		{"staff", "Staff"},                  // single word
		{"plan-review", "Plan Review"},      // no matching prefix
		{"feature-single", "Single"},        // single word after prefix
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			a := Activity{ID: tt.id}
			if got := a.Name(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestActivityTotalDuration(t *testing.T) {
	f1 := Entry{Time: mustTime("2026-03-31 10:00"), Type: Focus}
	p1 := Entry{Time: mustTime("2026-03-31 10:30"), Type: Park}
	f2 := Entry{Time: mustTime("2026-03-31 11:00"), Type: Focus}
	p2 := Entry{Time: mustTime("2026-03-31 11:15"), Type: Park}
	f3 := Entry{Time: mustTime("2026-03-31 12:00"), Type: Focus}

	a := Activity{
		ID: "test",
		Pairs: []FocusPair{
			{Focus: f1, Park: &p1},  // 30m
			{Focus: f2, Park: &p2},  // 15m
			{Focus: f3, Park: nil},  // 0m (unpaired)
		},
	}

	if got := a.TotalDuration(); got != 45*time.Minute {
		t.Errorf("got %v, want 45m", got)
	}
	if got := a.ContextSwitches(); got != 3 {
		t.Errorf("got %d switches, want 3", got)
	}
}

func TestActivityLatestTime(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		a := Activity{Pairs: nil}
		if !a.LatestTime().IsZero() {
			t.Error("expected zero time for empty activity")
		}
	})

	t.Run("last pair has park", func(t *testing.T) {
		parkTime := mustTime("2026-03-31 11:30")
		a := Activity{Pairs: []FocusPair{
			{Focus: Entry{Time: mustTime("2026-03-31 11:00")}, Park: &Entry{Time: parkTime}},
		}}
		if got := a.LatestTime(); got != parkTime {
			t.Errorf("got %v, want %v", got, parkTime)
		}
	})

	t.Run("last pair unparked", func(t *testing.T) {
		focusTime := mustTime("2026-03-31 12:00")
		a := Activity{Pairs: []FocusPair{
			{Focus: Entry{Time: focusTime}, Park: nil},
		}}
		if got := a.LatestTime(); got != focusTime {
			t.Errorf("got %v, want %v", got, focusTime)
		}
	})
}

func TestReadAllActivities(t *testing.T) {
	dir := t.TempDir()

	// Create two activity dirs with logs.
	mkActivity := func(name, content string) {
		d := filepath.Join(dir, name)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "log"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mkActivity("alpha", ""+
		"2026-03-31 09:00  focus: alpha work\n"+
		"2026-03-31 09:30  park:  alpha done\n")

	mkActivity("beta", ""+
		"2026-03-31 14:00  focus: beta work\n"+
		"2026-03-31 14:45  park:  beta done\n")

	// Empty dir (should be skipped).
	os.MkdirAll(filepath.Join(dir, "empty"), 0o755)

	first := mustTime("2026-03-31 00:00")
	last := mustTime("2026-04-01 00:00")

	t.Run("desc order", func(t *testing.T) {
		activities, err := ReadAllActivities(dir, first, last, SortDesc)
		if err != nil {
			t.Fatal(err)
		}
		if len(activities) != 2 {
			t.Fatalf("got %d activities, want 2", len(activities))
		}
		// desc = most recent last
		if activities[0].ID != "alpha" {
			t.Errorf("first activity: got %q, want alpha", activities[0].ID)
		}
		if activities[1].ID != "beta" {
			t.Errorf("second activity: got %q, want beta", activities[1].ID)
		}
	})

	t.Run("asc order", func(t *testing.T) {
		activities, err := ReadAllActivities(dir, first, last, SortAsc)
		if err != nil {
			t.Fatal(err)
		}
		// asc = most recent first
		if activities[0].ID != "beta" {
			t.Errorf("first activity: got %q, want beta", activities[0].ID)
		}
	})

	t.Run("time filtering excludes entries", func(t *testing.T) {
		narrow := mustTime("2026-03-31 12:00")
		activities, err := ReadAllActivities(dir, narrow, last, SortDesc)
		if err != nil {
			t.Fatal(err)
		}
		if len(activities) != 1 {
			t.Fatalf("got %d activities, want 1 (only beta)", len(activities))
		}
		if activities[0].ID != "beta" {
			t.Errorf("got %q, want beta", activities[0].ID)
		}
	})
}

func TestReadAllActivitiesWithActivityID(t *testing.T) {
	dir := t.TempDir()

	mkActivity := func(name, logContent, activityID string) {
		t.Helper()
		d := filepath.Join(dir, name)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "log"), []byte(logContent), 0o644); err != nil {
			t.Fatal(err)
		}
		if activityID != "" {
			if err := os.WriteFile(filepath.Join(d, "activity"), []byte(activityID+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	mkActivity("alpha",
		"2026-03-31 09:00  focus: alpha work\n"+
			"2026-03-31 09:30  park:  alpha done\n",
		"early-id-1")

	mkActivity("beta",
		"2026-03-31 14:00  focus: beta work\n"+
			"2026-03-31 14:45  park:  beta done\n",
		"") // no activity ID

	first := mustTime("2026-03-31 00:00")
	last := mustTime("2026-04-01 00:00")

	activities, err := ReadAllActivities(dir, first, last, SortDesc)
	if err != nil {
		t.Fatal(err)
	}
	if len(activities) != 2 {
		t.Fatalf("got %d activities, want 2", len(activities))
	}

	// alpha has activity ID
	if activities[0].ActivityID != "early-id-1" {
		t.Errorf("alpha.ActivityID: got %q, want %q", activities[0].ActivityID, "early-id-1")
	}

	// beta has no activity ID
	if activities[1].ActivityID != "" {
		t.Errorf("beta.ActivityID: got %q, want empty", activities[1].ActivityID)
	}
}

func TestMergeByName(t *testing.T) {
	pair := func(focusTS, parkTS string) FocusPair {
		fp := FocusPair{Focus: Entry{Time: mustTime(focusTS), Type: Focus}}
		if parkTS != "" {
			p := Entry{Time: mustTime(parkTS), Type: Park}
			fp.Park = &p
		}
		return fp
	}

	t.Run("activities with same name are merged", func(t *testing.T) {
		activities := []Activity{
			{ID: "branch-a", ActivityID: "E1", Pairs: []FocusPair{pair("2026-04-08 14:00", "2026-04-08 15:00")}},
			{ID: "branch-b", ActivityID: "E1", Pairs: []FocusPair{pair("2026-04-08 09:00", "2026-04-08 10:00")}},
			{ID: "branch-c", Pairs: []FocusPair{pair("2026-04-08 12:00", "2026-04-08 13:00")}},
		}
		nameFunc := func(a Activity) string {
			if a.ActivityID == "E1" {
				return "Shared Task"
			}
			return a.Name()
		}

		merged := MergeByName(activities, nameFunc, SortDesc)

		if len(merged) != 2 {
			t.Fatalf("got %d activities, want 2", len(merged))
		}

		// find the merged one
		var shared *Activity
		for i := range merged {
			if nameFunc(merged[i]) == "Shared Task" {
				shared = &merged[i]
				break
			}
		}
		if shared == nil {
			t.Fatal("merged activity 'Shared Task' not found")
		}
		if len(shared.Pairs) != 2 {
			t.Fatalf("merged pairs: got %d, want 2", len(shared.Pairs))
		}
		// pairs should be sorted chronologically (09:00 before 14:00)
		if shared.Pairs[0].Focus.Time.After(shared.Pairs[1].Focus.Time) {
			t.Error("merged pairs not in chronological order")
		}
	})

	t.Run("no merge when names differ", func(t *testing.T) {
		activities := []Activity{
			{ID: "alpha", Pairs: []FocusPair{pair("2026-04-08 10:00", "2026-04-08 11:00")}},
			{ID: "beta", Pairs: []FocusPair{pair("2026-04-08 12:00", "2026-04-08 13:00")}},
		}
		nameFunc := func(a Activity) string { return a.Name() }

		merged := MergeByName(activities, nameFunc, SortDesc)
		if len(merged) != 2 {
			t.Fatalf("got %d activities, want 2", len(merged))
		}
	})

	t.Run("sort order is respected", func(t *testing.T) {
		activities := []Activity{
			{ID: "early", Pairs: []FocusPair{pair("2026-04-08 09:00", "2026-04-08 10:00")}},
			{ID: "late", Pairs: []FocusPair{pair("2026-04-08 16:00", "2026-04-08 17:00")}},
		}
		nameFunc := func(a Activity) string { return a.Name() }

		asc := MergeByName(activities, nameFunc, SortAsc)
		if asc[0].ID != "late" {
			t.Errorf("asc: first should be 'late', got %q", asc[0].ID)
		}

		desc := MergeByName(activities, nameFunc, SortDesc)
		if desc[0].ID != "early" {
			t.Errorf("desc: first should be 'early', got %q", desc[0].ID)
		}
	})
}
