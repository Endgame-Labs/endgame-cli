package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/spf13/cobra"
)

const impersonateOrgHeader = "X-Endgame-Act-As-Org-Id"

var impersonateOrgID string

var rootCmd = &cobra.Command{
	Use:           "endgame",
	Short:         "CLI for the Endgame.io API",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd
		_ = args
		if timeoutValue := strings.TrimSpace(os.Getenv("ENDGAME_TIMEOUT_SECONDS")); timeoutValue != "" {
			timeoutSeconds, err := strconv.Atoi(timeoutValue)
			if err != nil || timeoutSeconds <= 0 {
				return fmt.Errorf("invalid ENDGAME_TIMEOUT_SECONDS: %q", timeoutValue)
			}
		}
		if err := configureMCPHeaders(); err != nil {
			return err
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		_ = args
		maybeStartAutoToolSync(cmd)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&impersonateOrgID, "impersonate-org", "", "Endgame org ID for multi-instance admins to send as X-Endgame-Act-As-Org-Id on every MCP request")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func configureMCPHeaders() error {
	orgID := strings.TrimSpace(impersonateOrgID)
	if orgID == "" {
		auth.SetConfiguredMCPHeaders(nil)
		return nil
	}
	auth.SetConfiguredMCPHeaders(map[string]string{
		impersonateOrgHeader: orgID,
	})
	return nil
}
