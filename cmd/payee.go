package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(payeeCmd)
	payeeCmd.AddCommand(payeeListCmd)
	payeeCmd.AddCommand(payeeGetCmd)
	payeeCmd.AddCommand(payeeUpdateCmd)

	payeeListCmd.Flags().Int("last-knowledge", 0, "Delta request: only fetch changes since this server_knowledge value")
	payeeUpdateCmd.Flags().String("json-input", "", "Full JSON payee body")
}

var payeeCmd = &cobra.Command{
	Use:   "payee",
	Short: "Manage YNAB payees",
	Long: `List, view, and update payees within a budget.

Examples:
  nab payee list                                    List all payees
  nab payee get <payee-id>                          Get payee details
  nab payee update <payee-id> --json-input '{...}'  Update a payee`,
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

		lastKnowledge, _ := cmd.Flags().GetInt("last-knowledge")
		path := fmt.Sprintf("/budgets/%s/payees", budgetID)
		if lastKnowledge > 0 {
			path = fmt.Sprintf("%s?last_knowledge_of_server=%d", path, lastKnowledge)
		}

		var result struct {
			Payees []map[string]any `json:"payees"`
		}
		if err := client.GetJSON(path, &result); err != nil {
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

var payeeUpdateCmd = &cobra.Command{
	Use:   "update <payee-id>",
	Short: "Update a payee",
	Long: `Update an existing payee.

Examples:
  nab payee update <id> --json-input '{"name":"New Payee Name"}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for payee update")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would update payee " + args[0] + ":")
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

		payload := map[string]any{"payee": body}
		respData, err := client.Patch(fmt.Sprintf("/budgets/%s/payees/%s", budgetID, args[0]), payload)
		if err != nil {
			return fmt.Errorf("updating payee: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Payee updated")
		return printAPIResult(result)
	},
}
