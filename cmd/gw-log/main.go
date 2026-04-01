package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/els0r/gw/internal/render"
	"github.com/els0r/gw/internal/session"
)

const dateLayout = "2006-01-02"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	stateDir := os.Getenv("GW_STATE_DIR")
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fatal("cannot determine home directory: %v", err)
		}
		stateDir = filepath.Join(home, ".gw")
	}
	sessionsDir := filepath.Join(stateDir, "sessions")

	switch os.Args[1] {
	case "write":
		cmdWrite(sessionsDir, os.Args[2:])
	case "read":
		cmdRead(sessionsDir, os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func cmdWrite(sessionsDir string, args []string) {
	fs := flag.NewFlagSet("write", flag.ExitOnError)
	typ := fs.String("type", "", "entry type: focus or park")
	branch := fs.String("branch", "", "branch / activity ID")
	note := fs.String("note", "", "log note")
	fs.Parse(args)

	if *typ == "" || *branch == "" || *note == "" {
		fmt.Fprintln(os.Stderr, "usage: gw-log write --type=focus|park --branch=BRANCH --note=\"...\"")
		os.Exit(1)
	}

	var entryType session.EntryType
	switch *typ {
	case "focus":
		entryType = session.Focus
	case "park":
		entryType = session.Park
	default:
		fatal("invalid type %q: must be focus or park", *typ)
	}

	if err := session.WriteEntry(sessionsDir, *branch, entryType, *note); err != nil {
		fatal("write failed: %v", err)
	}
}

func cmdRead(sessionsDir string, args []string) {
	flagArgs, posArgs := splitArgs(args)

	fs := flag.NewFlagSet("read", flag.ExitOnError)
	firstStr := fs.String("first", "", "show logs from this date (YYYY-MM-DD)")
	lastStr := fs.String("last", "", "show logs until this date (YYYY-MM-DD)")
	sortStr := fs.String("sort", "desc", "sort order: desc (most recent last) or asc (most recent first)")
	fs.Parse(flagArgs)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	first := today
	last := today.Add(24 * time.Hour)

	// positional range argument (e.g. "today", "yesterday", "this week", "last week")
	if len(posArgs) > 0 {
		rangeStr := strings.ToLower(strings.Join(posArgs, " "))
		rf, rl, ok := resolveRange(rangeStr, today)
		if !ok {
			fatal("unknown range %q: use today, yesterday, \"this week\", or \"last week\"", rangeStr)
		}
		first, last = rf, rl
	}

	// explicit flags override the range
	if *firstStr != "" {
		t, err := time.ParseInLocation(dateLayout, *firstStr, now.Location())
		if err != nil {
			fatal("bad --first date: %v", err)
		}
		first = t
	}
	if *lastStr != "" {
		t, err := time.ParseInLocation(dateLayout, *lastStr, now.Location())
		if err != nil {
			fatal("bad --last date: %v", err)
		}
		// include the entire last day
		last = t.Add(24 * time.Hour)
	}

	order := session.SortDesc
	switch *sortStr {
	case "desc":
		order = session.SortDesc
	case "asc":
		order = session.SortAsc
	default:
		fatal("invalid --sort %q: use desc or asc", *sortStr)
	}

	activities, err := session.ReadAllActivities(sessionsDir, first, last, order)
	if err != nil {
		fatal("read failed: %v", err)
	}

	if len(activities) == 0 {
		fmt.Println("  no entries")
		return
	}

	render.Activities(activities)
}

// resolveRange maps a named range to [first, last) boundaries.
func resolveRange(name string, today time.Time) (first, last time.Time, ok bool) {
	switch name {
	case "today":
		return today, today.Add(24 * time.Hour), true
	case "yesterday":
		y := today.AddDate(0, 0, -1)
		return y, today, true
	case "this week":
		// week starts on Monday
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
	default:
		return time.Time{}, time.Time{}, false
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gw-log <read|write>")
	fmt.Fprintln(os.Stderr, "  write --type=focus|park --branch=BRANCH --note=\"...\"")
	fmt.Fprintln(os.Stderr, "  read  [today|yesterday|\"this week\"|\"last week\"] [--first=YYYY-MM-DD] [--last=YYYY-MM-DD]")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gw-log: "+format+"\n", args...)
	os.Exit(1)
}

// splitArgs partitions args into flag args and positional args,
// allowing flags to appear in any position relative to positional words.
func splitArgs(args []string) (flagArgs, posArgs []string) {
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			posArgs = append(posArgs, args[i])
			continue
		}
		flagArgs = append(flagArgs, args[i])
		// flag without "=" needs the next arg as its value
		if strings.Contains(args[i], "=") || i+1 >= len(args) {
			continue
		}
		i++
		flagArgs = append(flagArgs, args[i])
	}
	return
}
