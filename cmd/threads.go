package cmd

import "github.com/spf13/cobra"

var threadsCmd = &cobra.Command{
	Use:   "threads",
	Short: "Interact with Endgame MCP thread tools",
}

func init() {
	rootCmd.AddCommand(threadsCmd)
}
