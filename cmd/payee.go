package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(payeeCmd)
	payeeCmd.AddCommand(payeeListCmd)
	payeeCmd.AddCommand(payeeGetCmd)
}

var payeeCmd = &cobra.Command{
	Use:   "payee",
	Short: "Manage YNAB payees",
	Long: `List and view payees within a budget.

Examples:
  nab payee list                   List all payees
  nab payee get <payee-id>         Get payee details`,
}

var payeeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all payees",
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
			Payees []map[string]any `json:"payees"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/payees", budgetID), &result); err != nil {
			return fmt.Errorf("listing payees: %w", err)
		}

		return printAPIResult(toAnySlice(result.Payees))
	},
}

var payeeGetCmd = &cobra.Command{
	Use:   "get <payee-id>",
	Short: "Get payee details",
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
			Payee map[string]any `json:"payee"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/payees/%s", budgetID, args[0]), &result); err != nil {
			return fmt.Errorf("getting payee: %w", err)
		}

		return printAPIResult(result.Payee)
	},
}
