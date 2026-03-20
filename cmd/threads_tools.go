package cmd

import (
	"fmt"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args
		apiKey, orgID, err := auth.LoadCredentials()
		if err != nil {
			return err
		}

		client, err := endgame.NewClient(apiKey, orgID)
		if err != nil {
			return err
		}
		if err := client.Initialize(); err != nil {
			return err
		}

		tools, err := client.ListTools()
		if err != nil {
			return err
		}

		for _, tool := range tools {
			if tool.Description == "" {
				fmt.Fprintln(cmd.OutOrStdout(), tool.Name)
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", tool.Name, tool.Description)
		}
		return nil
	},
}

func init() {
	threadsCmd.AddCommand(toolsCmd)
}
