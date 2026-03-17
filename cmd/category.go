package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(categoryCmd)
	categoryCmd.AddCommand(categoryListCmd)
	categoryCmd.AddCommand(categoryGetCmd)
	categoryCmd.AddCommand(categoryCreateCmd)
	categoryCmd.AddCommand(categoryUpdateCmd)
	categoryCmd.AddCommand(categoryGroupCreateCmd)
	categoryCmd.AddCommand(categoryGroupUpdateCmd)

	categoryListCmd.Flags().Int("last-knowledge", 0, "Delta request: only fetch changes since this server_knowledge value")
	categoryCreateCmd.Flags().String("json-input", "", "Full JSON category body")
	categoryUpdateCmd.Flags().String("json-input", "", "Full JSON category body")
	categoryUpdateCmd.Flags().String("month", "", "Update budget amount for a specific month (YYYY-MM-DD)")
	categoryGroupCreateCmd.Flags().String("json-input", "", "Full JSON category group body")
	categoryGroupUpdateCmd.Flags().String("json-input", "", "Full JSON category group body")
}

var categoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage YNAB categories",
	Long: `List, view, create, and update budget categories and category groups.

Examples:
  nab category list                              List all category groups and categories
  nab category get <category-id>                 Get category details
  nab category get <category-id> --month 2024-01 Get category for a specific month
  nab category create --json-input '{...}'       Create a category
  nab category update <id> --json-input '{...}'  Update a category
  nab category group-create --json-input '{...}' Create a category group
  nab category group-update <id> --json-input .. Update a category group`,
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

		lastKnowledge, _ := cmd.Flags().GetInt("last-knowledge")
		path := fmt.Sprintf("/budgets/%s/categories", budgetID)
		if lastKnowledge > 0 {
			path = fmt.Sprintf("%s?last_knowledge_of_server=%d", path, lastKnowledge)
		}

		var result struct {
			CategoryGroups []map[string]any `json:"category_groups"`
		}
		if err := client.GetJSON(path, &result); err != nil {
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

var categoryCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a category",
	Long: `Create a new category in the budget.

Examples:
  nab category create --json-input '{"category_group_id":"...","name":"Dining Out"}'`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for category create")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would create category:")
			return printAPIResult(body)
		}

		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID, err := requireBudget()
		if err != nil {
			return err
		}

		payload := map[string]any{"category": body}
		respData, err := client.Post(fmt.Sprintf("/budgets/%s/categories", budgetID), payload)
		if err != nil {
			return fmt.Errorf("creating category: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Category created")
		return printAPIResult(result)
	},
}

var categoryUpdateCmd = &cobra.Command{
	Use:   "update <category-id>",
	Short: "Update a category",
	Long: `Update an existing category, or update budget amount for a specific month.

Examples:
  nab category update <id> --json-input '{"name":"New Name"}'
  nab category update <id> --month 2024-01-01 --json-input '{"budgeted":500000}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for category update")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would update category " + args[0] + ":")
			return printAPIResult(body)
		}

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

		payload := map[string]any{"category": body}
		respData, err := client.Patch(path, payload)
		if err != nil {
			return fmt.Errorf("updating category: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Category updated")
		return printAPIResult(result)
	},
}

var categoryGroupCreateCmd = &cobra.Command{
	Use:   "group-create",
	Short: "Create a category group",
	Long: `Create a new category group in the budget.

Examples:
  nab category group-create --json-input '{"name":"Monthly Bills"}'`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for category group-create")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would create category group:")
			return printAPIResult(body)
		}

		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID, err := requireBudget()
		if err != nil {
			return err
		}

		payload := map[string]any{"category_group": body}
		respData, err := client.Post(fmt.Sprintf("/budgets/%s/category_groups", budgetID), payload)
		if err != nil {
			return fmt.Errorf("creating category group: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Category group created")
		return printAPIResult(result)
	},
}

var categoryGroupUpdateCmd = &cobra.Command{
	Use:   "group-update <category-group-id>",
	Short: "Update a category group",
	Long: `Update an existing category group.

Examples:
  nab category group-update <id> --json-input '{"name":"New Group Name"}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for category group-update")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would update category group " + args[0] + ":")
			return printAPIResult(body)
		}

		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID, err := requireBudget()
		if err != nil {
			return err
		}

		payload := map[string]any{"category_group": body}
		respData, err := client.Patch(fmt.Sprintf("/budgets/%s/category_groups/%s", budgetID, args[0]), payload)
		if err != nil {
			return fmt.Errorf("updating category group: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Category group updated")
		return printAPIResult(result)
	},
}
