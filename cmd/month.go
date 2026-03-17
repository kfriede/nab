package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(monthCmd)
	monthCmd.AddCommand(monthListCmd)
	monthCmd.AddCommand(monthGetCmd)
}

var monthCmd = &cobra.Command{
	Use:   "month",
	Short: "Manage YNAB budget months",
	Long: `List and view budget months.

Examples:
  nab month list                   List all budget months
  nab month get 2024-01            Get budget month details`,
}

var monthListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all budget months",
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
			Months []map[string]any `json:"months"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/months", budgetID), &result); err != nil {
			return fmt.Errorf("listing months: %w", err)
		}

		return printAPIResult(toAnySlice(result.Months))
	},
}

var monthGetCmd = &cobra.Command{
	Use:   "get <month>",
	Short: "Get budget month details",
	Long: `Get details of a specific budget month.

The month should be in YYYY-MM-DD format (first day of the month).

Examples:
  nab month get 2024-01-01`,
	Args: cobra.ExactArgs(1),
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
			Month map[string]any `json:"month"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/months/%s", budgetID, args[0]), &result); err != nil {
			return fmt.Errorf("getting month: %w", err)
		}

		return printAPIResult(result.Month)
	},
}
