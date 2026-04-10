package cmd

import (
	"fmt"

	"github.com/els0r/gw/internal/activity"
	"github.com/els0r/gw/internal/session"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newWriteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "write",
		Short: "Write a session log entry",
		RunE:  writeEntrypoint,
	}

	registerWriteFlags(cmd)
	return cmd
}

const (
	flagType     = "type"
	flagBranch   = "branch"
	flagNote     = "note"
	flagActivity = "activity"
)

func registerWriteFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.String(flagType, "", "entry type: focus or park")
	flags.String(flagBranch, "", "branch / activity ID")
	flags.String(flagNote, "", "log note")
	flags.String(flagActivity, "", "external activity ID (e.g. EARLY)")

	viper.BindPFlag(flagType, flags.Lookup(flagType))
	viper.BindPFlag(flagBranch, flags.Lookup(flagBranch))
	viper.BindPFlag(flagNote, flags.Lookup(flagNote))
	viper.BindPFlag(flagActivity, flags.Lookup(flagActivity))

	cmd.MarkFlagRequired(flagType)
	cmd.MarkFlagRequired(flagBranch)
	cmd.MarkFlagRequired(flagNote)
}

func writeEntrypoint(cmd *cobra.Command, args []string) error {
	typ := viper.GetString(flagType)

	var entryType session.EntryType
	switch typ {
	case "focus":
		entryType = session.Focus
	case "park":
		entryType = session.Park
	default:
		return fmt.Errorf("invalid --type %q: must be focus or park", typ)
	}

	branch := viper.GetString(flagBranch)
	note := viper.GetString(flagNote)
	activityID := viper.GetString(flagActivity)

	ctx := cmd.Context()
	resolver := buildResolver(ctx, cmd)
	resolve := func(id string) string {
		return activity.ResolveName(ctx, id, resolver)
	}

	if err := session.WriteEntry(sessionsDir(), branch, entryType, note, activityID, resolve); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	return nil
}
