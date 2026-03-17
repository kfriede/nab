package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(moneyMovementCmd)
	moneyMovementCmd.AddCommand(moneyMovementListCmd)
	moneyMovementCmd.AddCommand(moneyMovementGroupListCmd)
}

var moneyMovementCmd = &cobra.Command{
	Use:   "money-movement",
	Short: "View money movements",
	Long: `List money movements and money movement groups within a budget.

Examples:
  nab money-movement list                        List all money movements
  nab money-movement list --month 2024-01-01     List money movements for a month
  nab money-movement group-list                  List all money movement groups
  nab money-movement group-list --month 2024-01-01`,
}

var moneyMovementListCmd = &cobra.Command{
	Use:   "list",
	Short: "List money movements",
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

		month, _ := cmd.Flags().GetString("month")

		var path string
		if month != "" {
			path = fmt.Sprintf("/budgets/%s/months/%s/money_movements", budgetID, month)
		} else {
			path = fmt.Sprintf("/budgets/%s/money_movements", budgetID)
		}

		var result struct {
			MoneyMovements []map[string]any `json:"money_movements"`
		}
		if err := client.GetJSON(path, &result); err != nil {
			return fmt.Errorf("listing money movements: %w", err)
		}

		return printAPIResult(toAnySlice(result.MoneyMovements))
	},
}

var moneyMovementGroupListCmd = &cobra.Command{
	Use:   "group-list",
	Short: "List money movement groups",
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

		month, _ := cmd.Flags().GetString("month")

		var path string
		if month != "" {
			path = fmt.Sprintf("/budgets/%s/months/%s/money_movement_groups", budgetID, month)
		} else {
			path = fmt.Sprintf("/budgets/%s/money_movement_groups", budgetID)
		}

		var result struct {
			MoneyMovementGroups []map[string]any `json:"money_movement_groups"`
		}
		if err := client.GetJSON(path, &result); err != nil {
			return fmt.Errorf("listing money movement groups: %w", err)
		}

		return printAPIResult(toAnySlice(result.MoneyMovementGroups))
	},
}

func init() {
	moneyMovementListCmd.Flags().String("month", "", "Filter by month (YYYY-MM-DD)")
	moneyMovementGroupListCmd.Flags().String("month", "", "Filter by month (YYYY-MM-DD)")
}
