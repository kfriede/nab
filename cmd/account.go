package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountGetCmd)
}

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage YNAB accounts",
	Long: `List and view accounts within a budget.

Examples:
  nab account list                 List all accounts
  nab account get <account-id>     Get account details`,
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts in the budget",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID, err := requireBudget()
		if err != nil {
			return err
		}

		var result struct {
			Accounts []map[string]any `json:"accounts"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/accounts", budgetID), &result); err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}

		return printAPIResult(toAnySlice(result.Accounts))
	},
}

var accountGetCmd = &cobra.Command{
	Use:   "get <account-id>",
	Short: "Get account details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID, err := requireBudget()
		if err != nil {
			return err
		}

		var result struct {
			Account map[string]any `json:"account"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/accounts/%s", budgetID, args[0]), &result); err != nil {
			return fmt.Errorf("getting account: %w", err)
		}

		return printAPIResult(result.Account)
	},
}
