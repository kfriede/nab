package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(budgetCmd)
	budgetCmd.AddCommand(budgetListCmd)
	budgetCmd.AddCommand(budgetGetCmd)
}

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage YNAB budgets",
	Long: `List and view YNAB budgets.

Examples:
  nab budget list                  List all budgets
  nab budget get                   Get current budget details
  nab budget get <budget-id>       Get specific budget details`,
}

var budgetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all budgets",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient()
		if err != nil {
			return err
		}

		var result struct {
			Budgets []map[string]any `json:"budgets"`
		}
		if err := client.GetJSON("/budgets", &result); err != nil {
			return fmt.Errorf("listing budgets: %w", err)
		}

		return printAPIResult(toAnySlice(result.Budgets))
	},
}

var budgetGetCmd = &cobra.Command{
	Use:   "get [budget-id]",
	Short: "Get budget details",
	Long: `Get details of a specific budget. Uses the configured default budget if no ID is provided.

Supports delta requests via the --last-knowledge flag to fetch only changes since a known server state.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID := ""
		if len(args) > 0 {
			budgetID = args[0]
		} else {
			budgetID, err = requireBudget()
			if err != nil {
				return err
			}
		}

		path := fmt.Sprintf("/budgets/%s", budgetID)
		lastKnowledge, _ := cmd.Flags().GetInt("last-knowledge")
		if lastKnowledge > 0 {
			path = fmt.Sprintf("%s?last_knowledge_of_server=%d", path, lastKnowledge)
		}

		var result map[string]any
		if err := client.GetJSON(path, &result); err != nil {
			return fmt.Errorf("getting budget: %w", err)
		}

		return printAPIResult(result)
	},
}

func init() {
	budgetGetCmd.Flags().Int("last-knowledge", 0, "Delta request: only fetch changes since this server_knowledge value")
}
