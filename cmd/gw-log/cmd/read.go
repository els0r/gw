package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/els0r/gw/internal/activity"
	"github.com/els0r/gw/internal/render"
	"github.com/els0r/gw/internal/session"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read [range]",
		Short: "Show session log entries",
		Long: `Show session log entries filtered by date range.

Positional ranges: today, yesterday, "this week", "last week".
Explicit flags --first/--last override the positional range.`,
		Args: cobra.ArbitraryArgs,
		RunE: readEntrypoint,
	}

	registerReadFlags(cmd)
	return cmd
}

const (
	flagFirst = "first"
	flagLast  = "last"
	flagSort  = "sort"
)

func registerReadFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.String(flagFirst, "", "show logs from this date (YYYY-MM-DD)")
	flags.String(flagLast, "", "show logs until this date (YYYY-MM-DD)")
	flags.String(flagSort, "desc", "sort order: desc (most recent last) or asc (most recent first)")

	viper.BindPFlag(flagFirst, flags.Lookup(flagFirst))
	viper.BindPFlag(flagLast, flags.Lookup(flagLast))
	viper.BindPFlag(flagSort, flags.Lookup(flagSort))
}

func readEntrypoint(cmd *cobra.Command, args []string) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	first := today
	last := today.Add(24 * time.Hour)

	if len(args) > 0 {
		rangeStr := strings.ToLower(strings.Join(args, " "))
		rf, rl, ok := resolveRange(rangeStr, today)
		if !ok {
			return fmt.Errorf("unknown range %q: use today, yesterday, \"this week\", or \"last week\"", rangeStr)
		}
		first, last = rf, rl
	}

	if firstStr := viper.GetString(flagFirst); firstStr != "" {
		t, err := time.ParseInLocation(dateLayout, firstStr, now.Location())
		if err != nil {
			return fmt.Errorf("bad --first date: %w", err)
		}
		first = t
	}
	if lastStr := viper.GetString(flagLast); lastStr != "" {
		t, err := time.ParseInLocation(dateLayout, lastStr, now.Location())
		if err != nil {
			return fmt.Errorf("bad --last date: %w", err)
		}
		last = t.Add(24 * time.Hour)
	}

	order := session.SortDesc
	sortVal := viper.GetString(flagSort)
	switch sortVal {
	case "desc":
		order = session.SortDesc
	case "asc":
		order = session.SortAsc
	default:
		return fmt.Errorf("invalid --sort %q: use desc or asc", sortVal)
	}

	activities, err := session.ReadAllActivities(sessionsDir(), first, last, order)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	if len(activities) == 0 {
		fmt.Println("  no entries")
		return nil
	}

	ctx := cmd.Context()
	resolver := buildResolver(ctx, cmd)
	nameFunc := func(a session.Activity) string {
		return activity.DisplayName(ctx, a, resolver)
	}

	activities = session.MergeByName(activities, nameFunc, order)

	render.Activities(activities, nameFunc)
	return nil
}

func buildResolver(ctx context.Context, cmd *cobra.Command) activity.Resolver {
	apiKey := viper.GetString("early_api_key")
	apiSecret := viper.GetString("early_api_secret")
	if apiKey == "" || apiSecret == "" {
		return activity.Nop{}
	}

	stateDir, _ := cmd.Root().PersistentFlags().GetString(flagStateDir)
	token, err := activity.EarlyToken(ctx, stateDir, apiKey, apiSecret)
	if err != nil {
		// degrade gracefully — fall back to branch names
		fmt.Printf("  ⚠  EARLY auth failed: %v\n", err)
		return activity.Nop{}
	}

	return activity.NewEarlyResolver(token)
}
