package cmd

import (
	"fmt"

	"github.com/Endgame-Labs/endgame-cli/pkg/buildinfo"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show build version information",
	Run: func(cmd *cobra.Command, args []string) {
		_ = args
		fmt.Fprintf(cmd.OutOrStdout(), "endgame %s\n", buildinfo.Summary())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
