package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Access Endgame MCP tools",
	Long:  "Access Endgame MCP tools directly by raw v1 tool name.",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args
		return cmd.Help()
	},
}

type stubToolSpec struct {
	Use           string
	Aliases       []string
	Short         string
	BackendTool   string
	Hidden        bool
	RequiresFlags string
}

func newToolGroup(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			return cmd.Help()
		},
	}
}

func newStubToolCommand(spec stubToolSpec) *cobra.Command {
	var jsonArgs string

	cmd := &cobra.Command{
		Use:     spec.Use,
		Aliases: spec.Aliases,
		Short:   spec.Short,
		Hidden:  spec.Hidden,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			arguments, err := loadToolArguments(jsonArgs)
			if err != nil {
				return err
			}
			return executeMCPTool(cmd, spec.BackendTool, arguments)
		},
	}

	cmd.Flags().StringVar(&jsonArgs, "json", "", "JSON object of tool arguments; if omitted, piped stdin is used or an empty object is sent")
	if spec.RequiresFlags != "" {
		cmd.Long = fmt.Sprintf("Invokes MCP tool %q. Pass arguments via --json or piped stdin.\n\nSuggested shape: %s", spec.BackendTool, spec.RequiresFlags)
	}
	return cmd
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}

func loadToolArguments(jsonArgs string) (map[string]any, error) {
	if trimmed := strings.TrimSpace(jsonArgs); trimmed != "" {
		return decodeToolArguments(trimmed)
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return map[string]any{}, nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return map[string]any{}, nil
	}

	return decodeToolArguments(string(data))
}

func decodeToolArguments(input string) (map[string]any, error) {
	var arguments map[string]any
	if err := json.Unmarshal([]byte(input), &arguments); err != nil {
		return nil, fmt.Errorf("decode --json tool arguments: %w", err)
	}
	if arguments == nil {
		return nil, errors.New("tool arguments must be a JSON object")
	}
	return arguments, nil
}
