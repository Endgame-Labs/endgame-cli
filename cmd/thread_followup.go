package cmd

import (
	"fmt"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"github.com/spf13/cobra"
)

var threadFollowupCmd = &cobra.Command{
	Use:   "followup <operation-id>",
	Short: "Fetch the latest status for an existing operation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		status, answer, err := client.Followup(args[0])
		if err != nil {
			return err
		}

		if answer == "" {
			fmt.Fprintln(cmd.OutOrStdout(), status)
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "status: %s\n%s\n", status, answer)
		return nil
	},
}

func init() {
	threadCmd.AddCommand(threadFollowupCmd)
}
