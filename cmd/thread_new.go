package cmd

import (
	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"github.com/spf13/cobra"
)

var newPrompt string

var threadNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new Endgame thread",
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt, err := resolvePrompt(newPrompt)
		if err != nil {
			return err
		}

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

		result, err := client.SubmitPrompt(prompt, "")
		if err != nil {
			return err
		}

		return printPromptResult(cmd, result)
	},
}

func init() {
	threadNewCmd.Flags().StringVar(&newPrompt, "prompt", "", "prompt text to send; if omitted, stdin is used")
	threadCmd.AddCommand(threadNewCmd)
}
