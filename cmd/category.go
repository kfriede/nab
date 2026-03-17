package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(categoryCmd)
	categoryCmd.AddCommand(categoryListCmd)
	categoryCmd.AddCommand(categoryGetCmd)
}

var categoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage YNAB categories",
	Long: `List and view budget categories.

Examples:
  nab category list                              List all category groups and categories
  nab category get <category-id>                 Get category details
  nab category get <category-id> --month 2024-01 Get category for a specific month`,
}

var categoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all categories",
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
			CategoryGroups []map[string]any `json:"category_groups"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/categories", budgetID), &result); err != nil {
			return fmt.Errorf("listing categories: %w", err)
		}

		return printAPIResult(toAnySlice(result.CategoryGroups))
	},
}

var categoryGetCmd = &cobra.Command{
	Use:   "get <category-id>",
	Short: "Get category details",
	Long: `Get details of a specific category, optionally for a specific month.

Examples:
  nab category get <category-id>
  nab category get <category-id> --month 2024-01`,
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

		month, _ := cmd.Flags().GetString("month")

		var path string
		if month != "" {
			path = fmt.Sprintf("/budgets/%s/months/%s/categories/%s", budgetID, month, args[0])
		} else {
			path = fmt.Sprintf("/budgets/%s/categories/%s", budgetID, args[0])
		}

		var result struct {
			Category map[string]any `json:"category"`
		}
		if err := client.GetJSON(path, &result); err != nil {
			return fmt.Errorf("getting category: %w", err)
		}

		return printAPIResult(result.Category)
	},
}

func init() {
	categoryGetCmd.Flags().String("month", "", "Get category data for a specific month (YYYY-MM)")
}
