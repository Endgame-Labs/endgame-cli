package cmd

import "github.com/spf13/cobra"

var threadCmd = &cobra.Command{
	Use:   "thread",
	Short: "Create and continue Endgame threads",
}

func init() {
	rootCmd.AddCommand(threadCmd)
}
