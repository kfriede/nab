package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(schemaCmd)
}

// SchemaEntry describes a single command for agent introspection.
type SchemaEntry struct {
	Resource    string        `json:"resource"`
	Action      string        `json:"action"`
	Description string        `json:"description"`
	Method      string        `json:"httpMethod"`
	Path        string        `json:"apiPath"`
	Parameters  []SchemaParam `json:"parameters,omitempty"`
	Flags       []SchemaFlag  `json:"flags,omitempty"`
	Example     string        `json:"example"`
	Mutating    bool          `json:"mutating"`
	DryRun      bool          `json:"supportsDryRun"`
}

// SchemaParam describes a path/query parameter.
type SchemaParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	In       string `json:"in"` // path, query, config
}

// SchemaFlag describes a CLI flag.
type SchemaFlag struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Desc     string `json:"description"`
}

var schemaCmd = &cobra.Command{
	Use:   "schema [resource.action]",
	Short: "Runtime command schema for LLM agents",
	Long: `Returns a JSON schema for any command, including parameters, types,
required fields, and copy-pasteable examples.

This is the primary entry point for LLM agents discovering how to use nab.

Examples:
  nab schema                       List all available commands
  nab schema budget.list            Schema for budget list
  nab schema transaction.create     Schema for transaction create`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		registry := buildSchemaRegistry()

		if len(args) == 0 {
			// List all commands
			summary := make([]map[string]string, 0, len(registry))
			for key, entry := range registry {
				summary = append(summary, map[string]string{
					"command":     key,
					"description": entry.Description,
					"mutating":    fmt.Sprintf("%v", entry.Mutating),
				})
			}
			return printAPIResult(toAnySlice(mapSlice(summary)))
		}

		key := args[0]
		entry, ok := registry[key]
		if !ok {
			// Try to suggest
			suggestions := findSimilar(key, registry)
			msg := fmt.Sprintf("unknown command: %s", key)
			if len(suggestions) > 0 {
				msg += fmt.Sprintf("\n\nDid you mean one of these?\n  %s", strings.Join(suggestions, "\n  "))
			}
			return fmt.Errorf("%s", msg)
		}

		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(entry)
	},
}

func buildSchemaRegistry() map[string]SchemaEntry {
	budgetParam := SchemaParam{Name: "budgetId", Type: "string", Required: true, In: "config"}

	r := map[string]SchemaEntry{
		"budget.list": {
			Resource: "budget", Action: "list", Description: "List all budgets",
			Method: "GET", Path: "/budgets", Example: "nab budget list",
		},
		"budget.get": {
			Resource: "budget", Action: "get", Description: "Get budget details (supports delta via --last-knowledge)",
			Method: "GET", Path: "/budgets/{budgetId}", Example: "nab budget get --fields id,name,last_modified_on",
			Parameters: []SchemaParam{budgetParam, {Name: "last_knowledge_of_server", Type: "integer", In: "query"}},
		},
		"account.list": {
			Resource: "account", Action: "list", Description: "List all accounts in the budget",
			Method: "GET", Path: "/budgets/{budgetId}/accounts",
			Example:    "nab account list --fields id,name,type,balance",
			Parameters: []SchemaParam{budgetParam},
		},
		"account.get": {
			Resource: "account", Action: "get", Description: "Get account details",
			Method: "GET", Path: "/budgets/{budgetId}/accounts/{accountId}",
			Example:    "nab account get <account-id>",
			Parameters: []SchemaParam{budgetParam, {Name: "accountId", Type: "string", Required: true, In: "path"}},
		},
		"transaction.list": {
			Resource: "transaction", Action: "list", Description: "List transactions",
			Method: "GET", Path: "/budgets/{budgetId}/transactions",
			Example: "nab transaction list --fields id,date,amount,payee_name,category_name",
			Parameters: []SchemaParam{budgetParam, {Name: "since_date", Type: "string", In: "query"}},
			Flags: []SchemaFlag{
				{Name: "since", Type: "string", Desc: "Only return transactions on or after this date (YYYY-MM-DD)"},
				{Name: "account", Type: "string", Desc: "Filter by account ID"},
				{Name: "last-knowledge", Type: "integer", Desc: "Delta request: server_knowledge value"},
			},
		},
		"transaction.get": {
			Resource: "transaction", Action: "get", Description: "Get transaction details",
			Method: "GET", Path: "/budgets/{budgetId}/transactions/{transactionId}",
			Example:    "nab transaction get <transaction-id>",
			Parameters: []SchemaParam{budgetParam, {Name: "transactionId", Type: "string", Required: true, In: "path"}},
		},
		"transaction.create": {
			Resource: "transaction", Action: "create", Description: "Create a transaction",
			Method: "POST", Path: "/budgets/{budgetId}/transactions", Mutating: true, DryRun: true,
			Example: `nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store"}'`,
			Flags: []SchemaFlag{
				{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON transaction body (amounts in milliunits: 1000 = $1.00)"},
			},
		},
		"transaction.update": {
			Resource: "transaction", Action: "update", Description: "Update a transaction",
			Method: "PUT", Path: "/budgets/{budgetId}/transactions/{transactionId}", Mutating: true, DryRun: true,
			Example: `nab transaction update <id> --json-input '{"amount":-75000,"memo":"Updated"}'`,
			Flags:   []SchemaFlag{{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON transaction body"}},
		},
		"transaction.delete": {
			Resource: "transaction", Action: "delete", Description: "Delete a transaction",
			Method: "DELETE", Path: "/budgets/{budgetId}/transactions/{transactionId}", Mutating: true, DryRun: true,
			Example: "nab transaction delete <transaction-id> --yes",
		},
		"category.list": {
			Resource: "category", Action: "list", Description: "List all category groups and categories",
			Method: "GET", Path: "/budgets/{budgetId}/categories",
			Example:    "nab category list",
			Parameters: []SchemaParam{budgetParam},
		},
		"category.get": {
			Resource: "category", Action: "get", Description: "Get category details (optionally by month)",
			Method: "GET", Path: "/budgets/{budgetId}/categories/{categoryId}",
			Example: "nab category get <category-id> --month 2024-01",
			Parameters: []SchemaParam{
				budgetParam,
				{Name: "categoryId", Type: "string", Required: true, In: "path"},
			},
			Flags: []SchemaFlag{
				{Name: "month", Type: "string", Desc: "Get category data for a specific month (YYYY-MM)"},
			},
		},
		"payee.list": {
			Resource: "payee", Action: "list", Description: "List all payees",
			Method: "GET", Path: "/budgets/{budgetId}/payees",
			Example:    "nab payee list --fields id,name",
			Parameters: []SchemaParam{budgetParam},
		},
		"payee.get": {
			Resource: "payee", Action: "get", Description: "Get payee details",
			Method: "GET", Path: "/budgets/{budgetId}/payees/{payeeId}",
			Example:    "nab payee get <payee-id>",
			Parameters: []SchemaParam{budgetParam, {Name: "payeeId", Type: "string", Required: true, In: "path"}},
		},
		"month.list": {
			Resource: "month", Action: "list", Description: "List all budget months",
			Method: "GET", Path: "/budgets/{budgetId}/months",
			Example:    "nab month list",
			Parameters: []SchemaParam{budgetParam},
		},
		"month.get": {
			Resource: "month", Action: "get", Description: "Get budget month details",
			Method: "GET", Path: "/budgets/{budgetId}/months/{month}",
			Example:    "nab month get 2024-01-01",
			Parameters: []SchemaParam{budgetParam, {Name: "month", Type: "string", Required: true, In: "path"}},
		},
	}

	return r
}

func findSimilar(input string, registry map[string]SchemaEntry) []string {
	var suggestions []string
	inputLower := strings.ToLower(input)
	for key := range registry {
		if strings.Contains(key, inputLower) || strings.Contains(inputLower, strings.Split(key, ".")[0]) {
			suggestions = append(suggestions, key)
		}
	}
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}
	return suggestions
}

func mapSlice(maps []map[string]string) []map[string]any {
	result := make([]map[string]any, len(maps))
	for i, m := range maps {
		r := make(map[string]any, len(m))
		for k, v := range m {
			r[k] = v
		}
		result[i] = r
	}
	return result
}

// toAnySlice converts []map[string]any to []any for the printer.
func toAnySlice(maps []map[string]any) []any {
	result := make([]any, len(maps))
	for i, m := range maps {
		result[i] = m
	}
	return result
}
