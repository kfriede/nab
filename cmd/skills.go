package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(skillsCmd)
}

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Dump agent-optimized usage instructions",
	Long: `Prints concise, agent-optimized usage instructions for LLM agents.

This is designed to be read once and internalized by an agent to
reduce hallucinations and improve command accuracy.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), skillsText)
		return nil
	},
}

const skillsText = `# nab — Agent Skills

## Quick Reference
nab <resource> <action> [flags]
Resources: budget, account, transaction, category, payee, month

## Rules
- ALWAYS use --fields on list/get to limit output (saves tokens)
- ALWAYS use --dry-run before any mutating command, then confirm with user
- ALWAYS pass --yes on confirmed destructive actions (delete)
- Use --json-input for create/update payloads (avoids flag hallucination)
- Parse JSON from stdout; errors go to stderr as JSON with "guidance" field
- Non-TTY automatically outputs JSON — no need for --json in agent context
- Amounts are in milliunits: 1000 = $1.00, negative = outflow

## Invariants
- Resource names are singular nouns: budget, account, transaction, category, payee, month
- All timestamps are ISO 8601 / UTC
- All dates are YYYY-MM-DD
- IDs are UUIDs
- Amounts are integers in milliunits (1000 = $1.00)
- "last-used" is a valid budget ID alias (resolves to most recently accessed budget)

## Common Patterns

### List with field selection
nab budget list --fields id,name,last_modified_on
nab account list --fields id,name,type,balance
nab transaction list --fields id,date,amount,payee_name,category_name
nab category list --fields id,name,budgeted,activity,balance

### Create with JSON input (preferred for agents)
nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store","category_id":"..."}'

### Mutating with dry-run first
nab transaction create --dry-run --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000}'
# show preview to user, then:
nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000}'

### Destructive with confirmation
nab transaction delete <transaction-id> --dry-run
# show preview to user, then:
nab transaction delete <transaction-id> --yes

### Delta requests (efficient sync)
nab transaction list --last-knowledge 1234
# Response includes server_knowledge for next delta request

### Runtime introspection
nab schema                        # list all commands
nab schema transaction.create     # full schema for a specific command

## Error Handling
Errors include structured JSON on stderr:
{"code":"AUTH_ERROR","message":"Token expired","guidance":"Run 'nab login' to re-authenticate."}

Exit codes: 0=success, 1=general error, 2=auth error, 3=not found, 4=conflict/validation

## Configuration
NAB_TOKEN, NAB_BUDGET, NAB_OUTPUT_FORMAT=json, NAB_DEBUG=1
`
