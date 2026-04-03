package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var threadEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Show required environment variables",
	Run: func(cmd *cobra.Command, args []string) {
		_ = args
		fmt.Fprintln(cmd.OutOrStdout(), "No required auth environment variables.")
		fmt.Fprintln(cmd.OutOrStdout(), "Use `endgame auth login` for browser-based OAuth.")
		if timeout := os.Getenv("ENDGAME_TIMEOUT_SECONDS"); timeout != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "ENDGAME_TIMEOUT_SECONDS=%s\n", timeout)
		}
	},
}

func init() {
	threadCmd.AddCommand(threadEnvCmd)
}
