---
name: ynab-budget-manager
description: "Manage YNAB budgets, accounts, transactions, and categories using the nab CLI"
usage: "Ask the agent to manage your YNAB budget, e.g. 'list my transactions', 'create a transaction', 'check my budget categories'"
arguments:
  - name: budget
    description: Budget ID or "last-used"
    type: string
examples:
  - input: "List my recent transactions"
    output: "Running `nab transaction list --fields id,date,amount,payee_name,category_name` to get your recent transactions."
  - input: "Create a grocery transaction for $50"
    output: "I'll use `nab transaction create --json-input '{...}'` with --dry-run first for your approval."
  - input: "Show me my budget categories for this month"
    output: "Fetching categories with `nab category list --fields id,name,budgeted,activity,balance`."
---

# YNAB Budget Manager

Manage your YNAB budget using the `nab` CLI. This skill enables the agent to list budgets, view accounts, manage transactions, check category balances, and review budget months — all through structured CLI commands with safety rails.

## Prerequisites

Install `nab` and ensure it's available in PATH:
```bash
brew install kfriede/tap/nab
# or: go install github.com/kfriede/nab@latest
```

Configure access:
```bash
export NAB_TOKEN=<your-ynab-personal-access-token>
export NAB_BUDGET=last-used
```

You can create a personal access token at https://app.ynab.com/settings/developer

> **Note:** This plugin requires local execution (e.g., Claude Code) with network access to `api.ynab.com`. It is not compatible with Claude Cowork's sandboxed VM, which restricts outbound network access.

## How to Use nab

Command pattern: `nab <resource> <action> [flags]`

### Discover commands
```bash
nab schema                      # list all available commands
nab schema transaction.create   # full schema for a specific command
nab skills                      # complete agent reference
```

### Read operations (always safe)
```bash
nab budget list --fields id,name,last_modified_on
nab account list --fields id,name,type,balance
nab transaction list --fields id,date,amount,payee_name,category_name
nab transaction list --since 2024-01-01
nab category list
nab payee list --fields id,name
nab month list
nab month get 2024-01-01
```

### Write operations (always dry-run first)
```bash
# Step 1: preview
nab transaction create --dry-run --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store","category_id":"..."}'
# Step 2: execute after user confirms
nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store","category_id":"..."}'
```

### Destructive operations (require --yes)
```bash
nab transaction delete <id> --dry-run    # preview
nab transaction delete <id> --yes        # execute
```

## Rules

- **ALWAYS** use `--fields` on list/get commands to limit output
- **ALWAYS** use `--dry-run` before any mutating command, show the preview, and ask for confirmation
- **ALWAYS** pass `--yes` for confirmed destructive actions
- **ALWAYS** use `--json-input` for create/update payloads
- **NEVER** parse table-formatted output — non-TTY mode auto-outputs JSON
- **NEVER** omit `--yes` on destructive commands (will hang in non-TTY)
- **NEVER** send unvalidated user input directly as IDs

## Amounts

All monetary amounts in YNAB are **milliunits** (integers):
- `1000` = $1.00
- `-50000` = -$50.00 (outflow)
- `150750` = $150.75 (inflow)

Always convert human-readable dollar amounts to milliunits before sending to the API.

## Available Resources

budget, account, transaction, category, payee, month

## Error Handling

Errors include structured JSON on stderr with a `guidance` field:
```json
{"code":"AUTH_ERROR","message":"Token is invalid","guidance":"Run `nab login` to store a new personal access token."}
```

Exit codes: 0=success, 1=general, 2=auth, 3=not found, 4=conflict
