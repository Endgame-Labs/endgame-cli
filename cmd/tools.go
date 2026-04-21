package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"github.com/Endgame-Labs/endgame-cli/pkg/toolscache"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Access Endgame MCP tools",
	Long:  "Access Endgame MCP tools discovered from the cached live MCP tool registry.",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args
		return cmd.Help()
	},
}

type stubToolSpec struct {
	Use         string
	Aliases     []string
	Short       string
	BackendTool string
	Hidden      bool
	Long        string
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
	cmd.Long = spec.Long
	return cmd
}

func init() {
	rootCmd.AddCommand(toolsCmd)
	toolsCmd.AddCommand(newToolsSyncCommand())
	registerCachedToolCommands()
	toolsCmd.SetHelpFunc(renderToolsHelp)
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

func registerCachedToolCommands() {
	cache, err := toolscache.Load()
	if err != nil {
		return
	}

	tools := append([]endgame.Tool(nil), cache.Tools...)
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		toolsCmd.AddCommand(newDynamicToolCommand(tool))
	}
}

func newDynamicToolCommand(tool endgame.Tool) *cobra.Command {
	short := summarizeToolDescription(tool.Description)
	if short == "" {
		short = "Invoke MCP tool " + tool.Name
	}

	long := fmt.Sprintf("Invokes MCP tool %q. Pass arguments via --json or piped stdin.", tool.Name)
	if description := strings.TrimSpace(tool.Description); description != "" {
		long = fmt.Sprintf("MCP tool %q.\n\nDescription:\n%s\n\nInvocation:\n  endgame tools %s --json '{...}'", tool.Name, description, tool.Name)
	}
	if schema := strings.TrimSpace(string(tool.InputSchema)); schema != "" {
		var formatted any
		if err := json.Unmarshal(tool.InputSchema, &formatted); err == nil {
			if data, err := json.MarshalIndent(formatted, "", "  "); err == nil {
				long = long + "\n\nInput schema:\n" + string(data)
			} else {
				long = long + "\n\nInput schema:\n" + schema
			}
		} else {
			long = long + "\n\nInput schema:\n" + schema
		}
	}

	cmd := newStubToolCommand(stubToolSpec{
		Use:         tool.Name,
		Short:       short,
		BackendTool: tool.Name,
		Long:        long,
	})
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["dynamic_tool"] = "1"
	cmd.Annotations["tool_description"] = strings.TrimSpace(tool.Description)
	return cmd
}

func summarizeToolDescription(description string) string {
	text := strings.TrimSpace(description)
	if text == "" {
		return ""
	}
	text = strings.Join(strings.Fields(text), " ")

	for _, sep := range []string{". ", "! ", "? "} {
		if idx := strings.Index(text, sep); idx > 0 {
			text = text[:idx+1]
			break
		}
	}

	const maxLen = 96
	if utf8.RuneCountInString(text) <= maxLen {
		return text
	}

	runes := []rune(text)
	return strings.TrimSpace(string(runes[:maxLen-1])) + "..."
}

func renderToolsHelp(cmd *cobra.Command, args []string) {
	_ = args
	out := cmd.OutOrStdout()

	fmt.Fprintln(out, cmd.Long)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintf(out, "  %s [flags]\n", cmd.CommandPath())
	fmt.Fprintf(out, "  %s [command]\n", cmd.CommandPath())
	fmt.Fprintln(out)

	commands := cmd.Commands()
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name() < commands[j].Name()
	})

	fmt.Fprintln(out, "Available Commands:")
	for _, sub := range commands {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		fmt.Fprintf(out, "  %-25s %s\n", sub.Name(), sub.Short)
	}
	fmt.Fprintln(out)

	if flagUsages := strings.TrimSpace(cmd.Flags().FlagUsages()); flagUsages != "" {
		fmt.Fprintln(out, "Flags:")
		fmt.Fprintln(out, flagUsages)
	}
	fmt.Fprintf(out, "Use \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())

	hadDynamic := false
	for _, sub := range commands {
		if sub.Annotations == nil || sub.Annotations["dynamic_tool"] != "1" {
			continue
		}
		if !hadDynamic {
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Tool Documentation:")
			hadDynamic = true
		}
		fmt.Fprintf(out, "  %s\n", sub.Name())
		description := strings.TrimSpace(sub.Annotations["tool_description"])
		if description == "" {
			description = strings.TrimSpace(sub.Short)
		}
		for _, line := range strings.Split(description, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				fmt.Fprintln(out)
				continue
			}
			fmt.Fprintf(out, "    %s\n", line)
		}
		fmt.Fprintf(out, "    Usage: endgame tools %s --json '{...}'\n\n", sub.Name())
	}
}
