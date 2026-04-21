package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"github.com/spf13/cobra"
)

func executeMCPTool(cmd *cobra.Command, toolName string, arguments map[string]any) error {
	client, err := auth.NewClient()
	if err != nil {
		return err
	}
	if err := client.Initialize(); err != nil {
		return err
	}

	result, err := client.CallTool(toolName, arguments)
	if err != nil {
		return err
	}

	return printToolCallResponse(cmd, result)
}

func printToolCallResponse(cmd *cobra.Command, result *endgame.ToolCallResponse) error {
	for _, item := range result.Content {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}

		var payload any
		if err := json.Unmarshal([]byte(text), &payload); err == nil {
			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return fmt.Errorf("format tool response: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			continue
		}

		fmt.Fprintln(cmd.OutOrStdout(), text)
	}

	return nil
}
