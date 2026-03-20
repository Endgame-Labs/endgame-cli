package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Endgame",
	Long: `Authenticate with Endgame using an API key and org ID.

Examples:
  endgame auth              # Interactive authentication
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
		orgID, err := auth.Verify()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		source := "config file"
		if strings.TrimSpace(os.Getenv("ENDGAME_API_KEY")) != "" && strings.TrimSpace(os.Getenv("ENDGAME_ORG_ID")) != "" {
			source = "environment"
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Authenticated")
		fmt.Fprintf(cmd.OutOrStdout(), "Org: %s\n", orgID)
		fmt.Fprintf(cmd.OutOrStdout(), "Source: %s\n", source)
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
