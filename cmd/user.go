package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userGetCmd)
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage YNAB user info",
	Long: `View authenticated user information.

Examples:
  nab user get                     Get current user info`,
}

var userGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get authenticated user info",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient()
		if err != nil {
			return err
		}

		var result struct {
			User map[string]any `json:"user"`
		}
		if err := client.GetJSON("/user", &result); err != nil {
			return err
		}

		return printAPIResult(result.User)
	},
}
