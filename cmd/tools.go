package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Access Endgame data tools",
	Long:  "Grouped Endgame data tools for meetings, people, accounts, documents, research, memories, and more.",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args
		return cmd.Help()
	},
}

type stubToolSpec struct {
	Use           string
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
	cmd := &cobra.Command{
		Use:    spec.Use,
		Short:  spec.Short,
		Hidden: spec.Hidden,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			msg := fmt.Sprintf("not implemented: backend API does not support this command yet (%s)", spec.BackendTool)
			if spec.RequiresFlags != "" {
				msg = fmt.Sprintf("%s; expected interface: %s", msg, spec.RequiresFlags)
			}
			return fmt.Errorf(msg)
		},
	}

	return cmd
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}
