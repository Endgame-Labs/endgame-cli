package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Endgame",
	Long: `Authenticate with Endgame using browser-based OAuth.

Examples:
  endgame auth              # Choose browser or device login
  endgame auth login        # Same as above
  endgame auth status       # Check authentication status
  endgame auth logout       # Clear stored credentials`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args
		return auth.Login(authLoginOptions(cmd))
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Endgame",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args
		return auth.Login(authLoginOptions(cmd))
	},
}

var authMode string
var authCallbackPort int

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
	authCmd.PersistentFlags().StringVar(&authMode, "mode", "", "authentication mode: browser or device")
	authCmd.PersistentFlags().IntVar(&authCallbackPort, "callback-port", 0, "fixed localhost callback port to use for OAuth")
}

func authLoginOptions(cmd *cobra.Command) auth.LoginOptions {
	options := auth.LoginOptions{
		Mode:         "browser",
		OpenBrowser:  true,
		CallbackHost: "127.0.0.1",
		CallbackPort: authCallbackPort,
	}

	switch strings.TrimSpace(strings.ToLower(authMode)) {
	case "":
	case "browser":
		options.Mode = "browser"
	case "device":
		options.Mode = "device"
	default:
		fmt.Fprintf(os.Stderr, "invalid --mode %q; expected browser or device\n", authMode)
		os.Exit(2)
	}

	if shouldPromptForAuthMode(cmd) {
		options.Mode = promptForAuthMode()
	}

	return options
}

func shouldPromptForAuthMode(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if flag := cmd.Flags().Lookup("mode"); flag != nil && flag.Changed {
		return false
	}
	if !isInteractiveTerminal(os.Stdin) || !isInteractiveTerminal(os.Stdout) {
		return false
	}
	return true
}

func promptForAuthMode() string {
	reader := bufio.NewReader(os.Stdin)
	deviceSupported, err := auth.SupportsDeviceLogin()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Could not check device login support: %v\n", err)
		fmt.Fprintln(os.Stdout, "Falling back to browser login.")
		return "browser"
	}

	if !deviceSupported {
		fmt.Fprintln(os.Stdout, "This auth server currently supports browser login only.")
		return "browser"
	}

	for {
		fmt.Fprintln(os.Stdout, "Choose authentication method:")
		fmt.Fprintln(os.Stdout, "  1. Open a browser on this machine")
		fmt.Fprintln(os.Stdout, "  2. Use a device code")
		fmt.Fprint(os.Stdout, "Selection [1]: ")

		choice, err := reader.ReadString('\n')
		if err != nil {
			return "browser"
		}

		switch strings.TrimSpace(strings.ToLower(choice)) {
		case "", "1", "browser", "b":
			return "browser"
		case "2", "device", "d":
			return "device"
		default:
			fmt.Fprintln(os.Stdout, "Enter 1 for browser login or 2 for device code.")
		}
	}
}

func isInteractiveTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
