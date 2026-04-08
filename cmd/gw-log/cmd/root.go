package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagStateDir    = "state-dir"
	flagSessionsDir = "sessions-dir"
)

func NewRootCmd() *cobra.Command {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gw-log: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	defaultStateDir := filepath.Join(home, ".gw")

	root := &cobra.Command{
		Use:           "gw-log",
		Short:         "Session log for gw (git worktree manager)",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			stateDir, _ := cmd.Flags().GetString(flagStateDir)

			viper.SetConfigName("config")
			viper.SetConfigType("yaml")
			viper.AddConfigPath(stateDir)
			viper.SetEnvPrefix("GW")
			viper.AutomaticEnv()
			viper.SetDefault("sessions_dir", filepath.Join(stateDir, "sessions"))

			if err := viper.ReadInConfig(); err != nil {
				var notFound viper.ConfigFileNotFoundError
				if !errors.As(err, &notFound) {
					return fmt.Errorf("reading config: %w", err)
				}
			}
			return nil
		},
	}

	flags := root.PersistentFlags()
	flags.String(flagStateDir, defaultStateDir, "gw state directory")
	flags.String(flagSessionsDir, "", "sessions directory (default: <state-dir>/sessions)")

	viper.BindPFlag("sessions_dir", flags.Lookup(flagSessionsDir))

	root.AddCommand(newReadCmd(), newWriteCmd())
	return root
}

func sessionsDir() string {
	return viper.GetString("sessions_dir")
}
