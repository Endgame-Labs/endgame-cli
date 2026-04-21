package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "endgame",
	Short:         "CLI for the Endgame.io API",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd
		_ = args
		if timeoutValue := strings.TrimSpace(os.Getenv("ENDGAME_TIMEOUT_SECONDS")); timeoutValue != "" {
			timeoutSeconds, err := strconv.Atoi(timeoutValue)
			if err != nil || timeoutSeconds <= 0 {
				return fmt.Errorf("invalid ENDGAME_TIMEOUT_SECONDS: %q", timeoutValue)
			}
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		_ = args
		maybeStartAutoToolSync(cmd)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
