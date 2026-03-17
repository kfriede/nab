package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountGetCmd)
	accountCmd.AddCommand(accountCreateCmd)

	accountListCmd.Flags().Int("last-knowledge", 0, "Delta request: only fetch changes since this server_knowledge value")
	accountCreateCmd.Flags().String("json-input", "", "Full JSON account body")
}

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage YNAB accounts",
	Long: `List, view, and create accounts within a budget.

Examples:
  nab account list                 List all accounts
  nab account get <account-id>     Get account details
  nab account create --json-input '{"name":"Checking","type":"checking","balance":0}'`,
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

		lastKnowledge, _ := cmd.Flags().GetInt("last-knowledge")
		path := fmt.Sprintf("/budgets/%s/accounts", budgetID)
		if lastKnowledge > 0 {
			path = fmt.Sprintf("%s?last_knowledge_of_server=%d", path, lastKnowledge)
		}

		var result struct {
			Accounts []map[string]any `json:"accounts"`
		}
		if err := client.GetJSON(path, &result); err != nil {
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

var accountCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an account",
	Long: `Create a new account in the budget.

Examples:
  nab account create --json-input '{"name":"Checking","type":"checking","balance":0}'`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonInput, _ := cmd.Flags().GetString("json-input")
		if jsonInput == "" {
			return fmt.Errorf("--json-input is required for account create")
		}

		body, err := parseJSONInput(jsonInput)
		if err != nil {
			return err
		}

		if flagDryRun {
			printer.Status("Dry run — would create account:")
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

		payload := map[string]any{"account": body}
		respData, err := client.Post(fmt.Sprintf("/budgets/%s/accounts", budgetID), payload)
		if err != nil {
			return fmt.Errorf("creating account: %w", err)
		}

		var result map[string]any
		if err := parseResponse(respData, &result); err != nil {
			return err
		}

		printer.Success("Account created")
		return printAPIResult(result)
	},
}
