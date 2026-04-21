package cmd

import (
	"fmt"
	"time"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/toolscache"
	"github.com/spf13/cobra"
)

func newToolsSyncCommand() *cobra.Command {
	var quiet bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Refresh cached MCP tool metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args

			count, err := auth.SyncToolsCache()
			if err != nil {
				return err
			}
			if quiet {
				return nil
			}

			cachePath, err := toolscache.GetPath()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Synced %d tool(s) at %s\n", count, time.Now().UTC().Format(time.RFC3339))
			fmt.Fprintf(cmd.OutOrStdout(), "Cache: %s\n", cachePath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress success output")
	return cmd
}
