package cmd

import (
	"fmt"
	"os"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Endgame",
	Long: `Authenticate with Endgame using browser-based OAuth.

Examples:
  endgame auth              # Browser-based login
  endgame auth login        # Same as above
  endgame auth status       # Check authentication status
  endgame auth logout       # Clear stored credentials`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd
		_ = args
		return auth.Login()
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Endgame",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd
		_ = args
		return auth.Login()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args
		status, err := auth.Verify()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Authenticated")
		if status.Email != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\n", status.Email)
		}
		if status.OrgName != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Org: %s (%s)\n", status.OrgName, status.OrgID)
		} else if status.OrgID != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Org: %s\n", status.OrgID)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Source: config file")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Endgame",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd
		_ = args
		if err := auth.Logout(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "Successfully logged out")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authentication target",
	RunE: func(cmd *cobra.Command, args []string) error {
		return statusCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(authCmd, whoamiCmd)
	authCmd.AddCommand(loginCmd, statusCmd, logoutCmd)
}
