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
		"budget.settings": {
			Resource: "budget", Action: "settings", Description: "Get budget settings",
			Method: "GET", Path: "/budgets/{budgetId}/settings",
			Example:    "nab budget settings",
			Parameters: []SchemaParam{budgetParam},
		},
		"account.list": {
			Resource: "account", Action: "list", Description: "List all accounts in the budget (supports delta)",
			Method: "GET", Path: "/budgets/{budgetId}/accounts",
			Example:    "nab account list --fields id,name,type,balance",
			Parameters: []SchemaParam{budgetParam, {Name: "last_knowledge_of_server", Type: "integer", In: "query"}},
			Flags:      []SchemaFlag{{Name: "last-knowledge", Type: "integer", Desc: "Delta request: server_knowledge value"}},
		},
		"account.get": {
			Resource: "account", Action: "get", Description: "Get account details",
			Method: "GET", Path: "/budgets/{budgetId}/accounts/{accountId}",
			Example:    "nab account get <account-id>",
			Parameters: []SchemaParam{budgetParam, {Name: "accountId", Type: "string", Required: true, In: "path"}},
		},
		"account.create": {
			Resource: "account", Action: "create", Description: "Create an account",
			Method: "POST", Path: "/budgets/{budgetId}/accounts", Mutating: true, DryRun: true,
			Example: `nab account create --json-input '{"name":"Checking","type":"checking","balance":0}'`,
			Flags:   []SchemaFlag{{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON account body"}},
		},
		"transaction.list": {
			Resource: "transaction", Action: "list", Description: "List transactions (supports filtering and delta)",
			Method: "GET", Path: "/budgets/{budgetId}/transactions",
			Example:    "nab transaction list --fields id,date,amount,payee_name,category_name",
			Parameters: []SchemaParam{budgetParam, {Name: "since_date", Type: "string", In: "query"}, {Name: "type", Type: "string", In: "query"}},
			Flags: []SchemaFlag{
				{Name: "since", Type: "string", Desc: "Only return transactions on or after this date (YYYY-MM-DD)"},
				{Name: "account", Type: "string", Desc: "Filter by account ID (sub-route)"},
				{Name: "category", Type: "string", Desc: "Filter by category ID (sub-route)"},
				{Name: "payee", Type: "string", Desc: "Filter by payee ID (sub-route)"},
				{Name: "month", Type: "string", Desc: "Filter by month YYYY-MM-DD (sub-route)"},
				{Name: "type", Type: "string", Desc: "Filter: uncategorized or unapproved"},
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
		"transaction.import": {
			Resource: "transaction", Action: "import", Description: "Import transactions from linked accounts",
			Method: "POST", Path: "/budgets/{budgetId}/transactions/import", Mutating: true, DryRun: true,
			Example: "nab transaction import",
		},
		"category.list": {
			Resource: "category", Action: "list", Description: "List all category groups and categories (supports delta)",
			Method: "GET", Path: "/budgets/{budgetId}/categories",
			Example:    "nab category list",
			Parameters: []SchemaParam{budgetParam, {Name: "last_knowledge_of_server", Type: "integer", In: "query"}},
			Flags:      []SchemaFlag{{Name: "last-knowledge", Type: "integer", Desc: "Delta request: server_knowledge value"}},
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
		"category.create": {
			Resource: "category", Action: "create", Description: "Create a category",
			Method: "POST", Path: "/budgets/{budgetId}/categories", Mutating: true, DryRun: true,
			Example: `nab category create --json-input '{"category_group_id":"...","name":"Dining Out"}'`,
			Flags:   []SchemaFlag{{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON category body"}},
		},
		"category.update": {
			Resource: "category", Action: "update", Description: "Update a category or assign budget for a month",
			Method: "PATCH", Path: "/budgets/{budgetId}/categories/{categoryId}", Mutating: true, DryRun: true,
			Example: `nab category update <id> --month 2024-01-01 --json-input '{"budgeted":500000}'`,
			Flags: []SchemaFlag{
				{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON category body"},
				{Name: "month", Type: "string", Desc: "Update budget amount for a specific month (YYYY-MM-DD)"},
			},
		},
		"category.group-create": {
			Resource: "category", Action: "group-create", Description: "Create a category group",
			Method: "POST", Path: "/budgets/{budgetId}/category_groups", Mutating: true, DryRun: true,
			Example: `nab category group-create --json-input '{"name":"Monthly Bills"}'`,
			Flags:   []SchemaFlag{{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON category group body"}},
		},
		"category.group-update": {
			Resource: "category", Action: "group-update", Description: "Update a category group",
			Method: "PATCH", Path: "/budgets/{budgetId}/category_groups/{categoryGroupId}", Mutating: true, DryRun: true,
			Example: `nab category group-update <id> --json-input '{"name":"New Name"}'`,
			Flags:   []SchemaFlag{{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON category group body"}},
		},
		"payee.list": {
			Resource: "payee", Action: "list", Description: "List all payees (supports delta)",
			Method: "GET", Path: "/budgets/{budgetId}/payees",
			Example:    "nab payee list --fields id,name",
			Parameters: []SchemaParam{budgetParam, {Name: "last_knowledge_of_server", Type: "integer", In: "query"}},
			Flags:      []SchemaFlag{{Name: "last-knowledge", Type: "integer", Desc: "Delta request: server_knowledge value"}},
		},
		"payee.get": {
			Resource: "payee", Action: "get", Description: "Get payee details",
			Method: "GET", Path: "/budgets/{budgetId}/payees/{payeeId}",
			Example:    "nab payee get <payee-id>",
			Parameters: []SchemaParam{budgetParam, {Name: "payeeId", Type: "string", Required: true, In: "path"}},
		},
		"payee.update": {
			Resource: "payee", Action: "update", Description: "Update a payee",
			Method: "PATCH", Path: "/budgets/{budgetId}/payees/{payeeId}", Mutating: true, DryRun: true,
			Example: `nab payee update <id> --json-input '{"name":"New Name"}'`,
			Flags:   []SchemaFlag{{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON payee body"}},
		},
		"payee-location.list": {
			Resource: "payee-location", Action: "list", Description: "List payee GPS locations",
			Method: "GET", Path: "/budgets/{budgetId}/payee_locations",
			Example: "nab payee-location list --payee <payee-id>",
			Flags:   []SchemaFlag{{Name: "payee", Type: "string", Desc: "Filter by payee ID"}},
		},
		"payee-location.get": {
			Resource: "payee-location", Action: "get", Description: "Get payee location details",
			Method: "GET", Path: "/budgets/{budgetId}/payee_locations/{payeeLocationId}",
			Example:    "nab payee-location get <location-id>",
			Parameters: []SchemaParam{budgetParam, {Name: "payeeLocationId", Type: "string", Required: true, In: "path"}},
		},
		"month.list": {
			Resource: "month", Action: "list", Description: "List all budget months (supports delta)",
			Method: "GET", Path: "/budgets/{budgetId}/months",
			Example:    "nab month list",
			Parameters: []SchemaParam{budgetParam, {Name: "last_knowledge_of_server", Type: "integer", In: "query"}},
			Flags:      []SchemaFlag{{Name: "last-knowledge", Type: "integer", Desc: "Delta request: server_knowledge value"}},
		},
		"month.get": {
			Resource: "month", Action: "get", Description: "Get budget month details",
			Method: "GET", Path: "/budgets/{budgetId}/months/{month}",
			Example:    "nab month get 2024-01-01",
			Parameters: []SchemaParam{budgetParam, {Name: "month", Type: "string", Required: true, In: "path"}},
		},
		"user.get": {
			Resource: "user", Action: "get", Description: "Get authenticated user info",
			Method: "GET", Path: "/user",
			Example: "nab user get",
		},
		"scheduled-transaction.list": {
			Resource: "scheduled-transaction", Action: "list", Description: "List scheduled transactions (supports delta)",
			Method: "GET", Path: "/budgets/{budgetId}/scheduled_transactions",
			Example:    "nab scheduled-transaction list",
			Parameters: []SchemaParam{budgetParam, {Name: "last_knowledge_of_server", Type: "integer", In: "query"}},
			Flags:      []SchemaFlag{{Name: "last-knowledge", Type: "integer", Desc: "Delta request: server_knowledge value"}},
		},
		"scheduled-transaction.get": {
			Resource: "scheduled-transaction", Action: "get", Description: "Get scheduled transaction details",
			Method: "GET", Path: "/budgets/{budgetId}/scheduled_transactions/{scheduledTransactionId}",
			Example:    "nab scheduled-transaction get <id>",
			Parameters: []SchemaParam{budgetParam, {Name: "scheduledTransactionId", Type: "string", Required: true, In: "path"}},
		},
		"scheduled-transaction.create": {
			Resource: "scheduled-transaction", Action: "create", Description: "Create a scheduled transaction",
			Method: "POST", Path: "/budgets/{budgetId}/scheduled_transactions", Mutating: true, DryRun: true,
			Example: `nab scheduled-transaction create --json-input '{"account_id":"...","date":"2024-02-01","amount":-50000,"frequency":"monthly"}'`,
			Flags:   []SchemaFlag{{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON scheduled transaction body"}},
		},
		"scheduled-transaction.update": {
			Resource: "scheduled-transaction", Action: "update", Description: "Update a scheduled transaction",
			Method: "PUT", Path: "/budgets/{budgetId}/scheduled_transactions/{scheduledTransactionId}", Mutating: true, DryRun: true,
			Example: `nab scheduled-transaction update <id> --json-input '{"amount":-60000}'`,
			Flags:   []SchemaFlag{{Name: "json-input", Type: "string", Required: true, Desc: "Full JSON scheduled transaction body"}},
		},
		"scheduled-transaction.delete": {
			Resource: "scheduled-transaction", Action: "delete", Description: "Delete a scheduled transaction",
			Method: "DELETE", Path: "/budgets/{budgetId}/scheduled_transactions/{scheduledTransactionId}", Mutating: true, DryRun: true,
			Example: "nab scheduled-transaction delete <id> --yes",
		},
		"money-movement.list": {
			Resource: "money-movement", Action: "list", Description: "List money movements",
			Method: "GET", Path: "/budgets/{budgetId}/money_movements",
			Example: "nab money-movement list --month 2024-01-01",
			Flags:   []SchemaFlag{{Name: "month", Type: "string", Desc: "Filter by month (YYYY-MM-DD)"}},
		},
		"money-movement.group-list": {
			Resource: "money-movement", Action: "group-list", Description: "List money movement groups",
			Method: "GET", Path: "/budgets/{budgetId}/money_movement_groups",
			Example: "nab money-movement group-list --month 2024-01-01",
			Flags:   []SchemaFlag{{Name: "month", Type: "string", Desc: "Filter by month (YYYY-MM-DD)"}},
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
