package cmd

import (
	"fmt"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt <question>",
	Short: "Start a new Endgame thread prompt and print the operation ID",
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

		opID, err := client.Prompt(args[0])
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), opID)
		return nil
	},
}

func init() {
	threadsCmd.AddCommand(promptCmd)
}
