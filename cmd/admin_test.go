package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestWorkOSDirectoryReconcileCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantTarget string
		wantError  bool
	}{
		{
			name:       "one organization",
			args:       []string{"--organization-id", "org-123"},
			wantTarget: "org-123",
		},
		{
			name:       "all organizations",
			args:       []string{"--all"},
			wantTarget: "*",
		},
		{name: "missing scope", wantError: true},
		{
			name:      "conflicting scopes",
			args:      []string{"--all", "--organization-id", "org-123"},
			wantError: true,
		},
		{
			name:      "wildcard requires all flag",
			args:      []string{"--organization-id", "*"},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var toolName string
			var arguments map[string]any
			command := newWorkOSDirectoryReconcileCommand(
				func(_ *cobra.Command, receivedTool string, receivedArgs map[string]any) error {
					toolName = receivedTool
					arguments = receivedArgs
					return nil
				},
			)
			command.SetArgs(test.args)

			err := command.Execute()
			if test.wantError {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("execute command: %v", err)
			}
			if toolName != "queue_workos_directory_reconciliation" {
				t.Fatalf("unexpected tool name %q", toolName)
			}
			wantArguments := map[string]any{"organization_id": test.wantTarget}
			if !reflect.DeepEqual(arguments, wantArguments) {
				t.Fatalf("arguments = %#v, want %#v", arguments, wantArguments)
			}
		})
	}
}
