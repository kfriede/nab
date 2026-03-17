package cmd

import (
	"fmt"

	"github.com/kfriede/nab/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(scheduledTransactionCmd)
	scheduledTransactionCmd.AddCommand(scheduledTransactionListCmd)
	scheduledTransactionCmd.AddCommand(scheduledTransactionGetCmd)
	scheduledTransactionCmd.AddCommand(scheduledTransactionCreateCmd)
	scheduledTransactionCmd.AddCommand(scheduledTransactionUpdateCmd)
	scheduledTransactionCmd.AddCommand(scheduledTransactionDeleteCmd)
}

var scheduledTransactionCmd = &cobra.Command{
	Use:     "scheduled-transaction",
	Aliases: []string{"st"},
	Short:   "Manage scheduled transactions",
	Long: `List, view, create, update, and delete scheduled transactions.

Examples:
  nab scheduled-transaction list
  nab scheduled-transaction get <id>
  nab scheduled-transaction create --json-input '{"account_id":"...","date":"2024-02-01","amount":-50000,"frequency":"monthly"}'
  nab scheduled-transaction update <id> --json-input '{"amount":-60000}'
  nab scheduled-transaction delete <id> --yes`,
}

var scheduledTransactionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scheduled transactions",
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

		path := fmt.Sprintf("/budgets/%s/scheduled_transactions", budgetID)
		lastKnowledge, _ := cmd.Flags().GetInt("last-knowledge")
		if lastKnowledge > 0 {
			path = fmt.Sprintf("%s?last_knowledge_of_server=%d", path, lastKnowledge)
		}

		var result struct {
			ScheduledTransactions []map[string]any `json:"scheduled_transactions"`
		}
		if err := client.GetJSON(path, &result); err != nil {
			return fmt.Errorf("listing scheduled transactions: %w", err)
		}

		return printAPIResult(toAnySlice(result.ScheduledTransactions))
	},
}

var scheduledTransactionGetCmd = &cobra.Command{
	Use:   "get <scheduled-transaction-id>",
	Short: "Get scheduled transaction details",
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
			ScheduledTransaction map[string]any `json:"scheduled_transaction"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/scheduled_transactions/%s", budgetID, args[0]), &result); err != nil {
			return fmt.Errorf("getting scheduled transaction: %w", err)
		}

		return printAPIResult(result.ScheduledTransaction)
	},
}

var scheduledTransactionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a scheduled transaction",
	Long: `Create a new scheduled transaction.

Examples:
  nab scheduled-transaction create --json-input '{"account_id":"...","date":"2024-02-01","amount":-50000,"payee_name":"Rent","frequency":"monthly"}'

Note: Amounts are in milliunits (1000 = $1.00). Use negative for outflows.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for scheduled-transaction create")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would create scheduled transaction:")
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

		payload := map[string]any{"scheduled_transaction": body}
		respData, err := client.Post(fmt.Sprintf("/budgets/%s/scheduled_transactions", budgetID), payload)
		if err != nil {
			return fmt.Errorf("creating scheduled transaction: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Scheduled transaction created")
		return printAPIResult(result)
	},
}

var scheduledTransactionUpdateCmd = &cobra.Command{
	Use:   "update <scheduled-transaction-id>",
	Short: "Update a scheduled transaction",
	Long: `Update an existing scheduled transaction.

Examples:
  nab scheduled-transaction update <id> --json-input '{"amount":-60000,"memo":"Updated"}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for scheduled-transaction update")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would update scheduled transaction " + args[0] + ":")
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

		payload := map[string]any{"scheduled_transaction": body}
		respData, err := client.Put(fmt.Sprintf("/budgets/%s/scheduled_transactions/%s", budgetID, args[0]), payload)
		if err != nil {
			return fmt.Errorf("updating scheduled transaction: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Scheduled transaction updated")
		return printAPIResult(result)
	},
}

var scheduledTransactionDeleteCmd = &cobra.Command{
	Use:   "delete <scheduled-transaction-id>",
	Short: "Delete a scheduled transaction",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagDryRun {
			printer.Status(fmt.Sprintf("Dry run — would delete scheduled transaction %s", args[0]))
			return nil
		}

		if !confirmAction(fmt.Sprintf("delete scheduled transaction %s", args[0])) {
			printer.PrintError(output.NewError(output.ErrCodeGeneral, "Cancelled", ""))
			return fmt.Errorf("cancelled")
		}

		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID, err := requireBudget()
		if err != nil {
			return err
		}

		if err := client.Delete(fmt.Sprintf("/budgets/%s/scheduled_transactions/%s", budgetID, args[0])); err != nil {
			return fmt.Errorf("deleting scheduled transaction: %w", err)
		}

		printer.Success(fmt.Sprintf("Deleted scheduled transaction %s", args[0]))
		return nil
	},
}

func init() {
	scheduledTransactionListCmd.Flags().Int("last-knowledge", 0, "Delta request: only fetch changes since this server_knowledge value")

	scheduledTransactionCreateCmd.Flags().String("json-input", "", "Full JSON scheduled transaction body")
	scheduledTransactionUpdateCmd.Flags().String("json-input", "", "Full JSON scheduled transaction body")
}
