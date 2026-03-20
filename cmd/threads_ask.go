package cmd

import (
	"fmt"
	"time"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"github.com/spf13/cobra"
)

const (
	pollDelay    = 3 * time.Second
	defaultPolls = 40
)

var askCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Prompt Endgame and wait for the completed response",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, orgID, err := auth.LoadCredentials()
		if err != nil {
			return err
		}

		client, err := endgame.NewClient(apiKey, orgID)
		if err != nil {
			return err
		}
		if err := client.Initialize(); err != nil {
			return err
		}

		maxPolls, err := cmd.Flags().GetInt("max-polls")
		if err != nil {
			return err
		}
		delaySeconds, err := cmd.Flags().GetInt("poll-delay")
		if err != nil {
			return err
		}

		opID, err := client.Prompt(args[0])
		if err != nil {
			return err
		}

		answer, err := endgame.WaitForCompletion(client, opID, maxPolls, time.Duration(delaySeconds)*time.Second)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), answer)
		return nil
	},
}

func init() {
	askCmd.Flags().Int("max-polls", defaultPolls, "maximum number of followup polls before timing out")
	askCmd.Flags().Int("poll-delay", int(pollDelay/time.Second), "seconds to wait between followup polls")
	threadsCmd.AddCommand(askCmd)
}
