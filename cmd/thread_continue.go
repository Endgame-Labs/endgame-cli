package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"github.com/spf13/cobra"
)

var continuePrompt string
var continueThreadID string

var threadContinueCmd = &cobra.Command{
	Use:   "continue --thread-id <thread-id>",
	Short: "Continue an existing Endgame thread",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args
		prompt, err := resolvePrompt(continuePrompt)
		if err != nil {
			return err
		}
		if continueThreadID == "" {
			return fmt.Errorf("--thread-id is required")
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

		result, err := client.SubmitPrompt(prompt, continueThreadID)
		if err != nil {
			return err
		}

		return printPromptResult(cmd, result)
	},
}

func init() {
	threadContinueCmd.Flags().StringVar(&continuePrompt, "prompt", "", "prompt text to send; if omitted, stdin is used")
	threadContinueCmd.Flags().StringVar(&continueThreadID, "thread-id", "", "thread ID to continue")
	threadCmd.AddCommand(threadContinueCmd)
}

func printPromptResult(cmd *cobra.Command, result *endgame.PromptResult) error {
	payload := map[string]string{
		"operation_id": result.OperationID,
	}
	if result.ThreadID != "" {
		payload["thread_id"] = result.ThreadID
	}
	if result.RawText != "" {
		payload["raw"] = result.RawText
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal prompt result: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}
