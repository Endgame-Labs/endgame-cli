package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	workOSDirectoryReconcileGoal    = "Initialize or repair the WorkOS directory replica from the Endgame CLI."
	workOSDirectoryReconcileJourney = "An Endgame administrator invoked the dedicated WorkOS directory reconciliation command."
)

type mcpToolExecutor func(*cobra.Command, string, map[string]any) error

func newAdminCommand(execute mcpToolExecutor) *cobra.Command {
	admin := &cobra.Command{
		Use:   "admin",
		Short: "Run Endgame administrative operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			return cmd.Help()
		},
	}
	workosDirectory := &cobra.Command{
		Use:   "workos-directory",
		Short: "Manage the WorkOS directory replica",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			return cmd.Help()
		},
	}
	workosDirectory.AddCommand(newWorkOSDirectoryReconcileCommand(execute))
	admin.AddCommand(workosDirectory)
	return admin
}

func newWorkOSDirectoryReconcileCommand(execute mcpToolExecutor) *cobra.Command {
	var organizationID string
	var allOrganizations bool
	command := &cobra.Command{
		Use:   "reconcile (--organization-id <id> | --all)",
		Short: "Queue WorkOS directory reconciliation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args
			organizationID = strings.TrimSpace(organizationID)
			if allOrganizations == (organizationID != "") {
				return errors.New("set exactly one of --organization-id or --all")
			}
			if organizationID == "*" {
				return errors.New("use --all instead of --organization-id '*'")
			}
			target := organizationID
			task := fmt.Sprintf(
				"Queue WorkOS directory reconciliation for Cerebro organization %s.",
				target,
			)
			if allOrganizations {
				target = "*"
				task = "Queue WorkOS directory reconciliation for every mapped organization."
			}
			return execute(
				cmd,
				"queue_workos_directory_reconciliation",
				map[string]any{
					"organization_id": target,
					"goal":            workOSDirectoryReconcileGoal,
					"task":            task,
					"journey":         workOSDirectoryReconcileJourney,
				},
			)
		},
	}
	command.Flags().StringVar(
		&organizationID,
		"organization-id",
		"",
		"Cerebro organization ID to reconcile",
	)
	command.Flags().BoolVar(
		&allOrganizations,
		"all",
		false,
		"Reconcile every mapped organization",
	)
	return command
}

func init() {
	rootCmd.AddCommand(newAdminCommand(executeMCPTool))
}
