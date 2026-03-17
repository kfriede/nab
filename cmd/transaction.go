package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/kfriede/nab/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(transactionCmd)
	transactionCmd.AddCommand(transactionListCmd)
	transactionCmd.AddCommand(transactionGetCmd)
	transactionCmd.AddCommand(transactionCreateCmd)
	transactionCmd.AddCommand(transactionUpdateCmd)
	transactionCmd.AddCommand(transactionDeleteCmd)
	transactionCmd.AddCommand(transactionImportCmd)
}

var transactionCmd = &cobra.Command{
	Use:   "transaction",
	Short: "Manage YNAB transactions",
	Long: `List, view, create, update, delete, and import transactions within a budget.

Examples:
  nab transaction list                            List recent transactions
  nab transaction list --since 2024-01-01         List since date
  nab transaction list --category <id>            List by category
  nab transaction list --payee <id>               List by payee
  nab transaction get <transaction-id>            Get transaction details
  nab transaction create --json-input '{...}'     Create a transaction
  nab transaction update <id> --json-input '{}'   Update a transaction
  nab transaction delete <transaction-id> --yes   Delete a transaction
  nab transaction import                          Import transactions from linked accounts`,
}

var transactionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List transactions",
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

		sinceDate, _ := cmd.Flags().GetString("since")
		accountID, _ := cmd.Flags().GetString("account")
		lastKnowledge, _ := cmd.Flags().GetInt("last-knowledge")
		categoryID, _ := cmd.Flags().GetString("category")
		payeeID, _ := cmd.Flags().GetString("payee")
		monthFilter, _ := cmd.Flags().GetString("month")
		txnType, _ := cmd.Flags().GetString("type")

		path := fmt.Sprintf("/budgets/%s/transactions", budgetID)
		if accountID != "" {
			path = fmt.Sprintf("/budgets/%s/accounts/%s/transactions", budgetID, accountID)
		} else if categoryID != "" {
			path = fmt.Sprintf("/budgets/%s/categories/%s/transactions", budgetID, categoryID)
		} else if payeeID != "" {
			path = fmt.Sprintf("/budgets/%s/payees/%s/transactions", budgetID, payeeID)
		} else if monthFilter != "" {
			path = fmt.Sprintf("/budgets/%s/months/%s/transactions", budgetID, monthFilter)
		}

		sep := "?"
		if sinceDate != "" {
			path += sep + "since_date=" + sinceDate
			sep = "&"
		}
		if txnType != "" {
			path += sep + "type=" + txnType
			sep = "&"
		}
		if lastKnowledge > 0 {
			path += fmt.Sprintf("%slast_knowledge_of_server=%d", sep, lastKnowledge)
		}

		var result struct {
			Transactions []map[string]any `json:"transactions"`
		}
		if err := client.GetJSON(path, &result); err != nil {
			return fmt.Errorf("listing transactions: %w", err)
		}

		return printAPIResult(toAnySlice(result.Transactions))
	},
}

var transactionGetCmd = &cobra.Command{
	Use:   "get <transaction-id>",
	Short: "Get transaction details",
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
			Transaction map[string]any `json:"transaction"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/transactions/%s", budgetID, args[0]), &result); err != nil {
			return fmt.Errorf("getting transaction: %w", err)
		}

		return printAPIResult(result.Transaction)
	},
}

var transactionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a transaction",
	Long: `Create a new transaction in the budget.

Examples:
  nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store","category_id":"..."}'

Note: Amounts are in milliunits (1000 = $1.00). Use negative for outflows.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for transaction create")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would create transaction:")
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

		payload := map[string]any{"transaction": body}
		respData, err := client.Post(fmt.Sprintf("/budgets/%s/transactions", budgetID), payload)
		if err != nil {
			return fmt.Errorf("creating transaction: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Transaction created")
		return printAPIResult(result)
	},
}

var transactionUpdateCmd = &cobra.Command{
	Use:   "update <transaction-id>",
	Short: "Update a transaction",
	Long: `Update an existing transaction.

Examples:
  nab transaction update <id> --json-input '{"amount":-75000,"memo":"Updated amount"}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for transaction update")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would update transaction " + args[0] + ":")
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

		payload := map[string]any{"transaction": body}
		respData, err := client.Put(fmt.Sprintf("/budgets/%s/transactions/%s", budgetID, args[0]), payload)
		if err != nil {
			return fmt.Errorf("updating transaction: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Transaction updated")
		return printAPIResult(result)
	},
}

var transactionDeleteCmd = &cobra.Command{
	Use:   "delete <transaction-id>",
	Short: "Delete a transaction",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagDryRun {
			printer.Status(fmt.Sprintf("Dry run — would delete transaction %s", args[0]))
			return nil
		}

		if !confirmAction(fmt.Sprintf("delete transaction %s", args[0])) {
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

		if err := client.Delete(fmt.Sprintf("/budgets/%s/transactions/%s", budgetID, args[0])); err != nil {
			return fmt.Errorf("deleting transaction: %w", err)
		}

		printer.Success(fmt.Sprintf("Deleted transaction %s", args[0]))
		return nil
	},
}

var transactionImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import transactions from linked accounts",
	Long: `Import transactions from linked accounts via file-based (OFX/QFX) import.

Examples:
  nab transaction import`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagDryRun {
			printer.Status("Dry run — would import transactions from linked accounts")
			return nil
		}

		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID, err := requireBudget()
		if err != nil {
			return err
		}

		respData, err := client.Post(fmt.Sprintf("/budgets/%s/transactions/import", budgetID), nil)
		if err != nil {
			return fmt.Errorf("importing transactions: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Transactions imported")
		return printAPIResult(result)
	},
}

func init() {
	transactionListCmd.Flags().String("since", "", "Only return transactions on or after this date (YYYY-MM-DD)")
	transactionListCmd.Flags().String("account", "", "Filter by account ID")
	transactionListCmd.Flags().Int("last-knowledge", 0, "Delta request: only fetch changes since this server_knowledge value")
	transactionListCmd.Flags().String("type", "", "Filter by type: uncategorized or unapproved")
	transactionListCmd.Flags().String("category", "", "Filter by category ID")
	transactionListCmd.Flags().String("payee", "", "Filter by payee ID")
	transactionListCmd.Flags().String("month", "", "Filter by month (uses month sub-route, YYYY-MM-DD)")

	transactionCreateCmd.Flags().String("json-input", "", "Full JSON transaction body")
	transactionUpdateCmd.Flags().String("json-input", "", "Full JSON transaction body")
}

func parseResponse(data []byte, target any) error {
	return json.Unmarshal(data, target)
}
