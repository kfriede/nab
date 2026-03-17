---
tool: nab
version: 0.1.0
description: CLI for You Need A Budget (YNAB)
always:
  - use --fields on list/get commands to limit output
  - use --dry-run before any mutating command
  - use --json-input for create/update payloads
  - pass --yes for confirmed destructive actions
  - parse JSON output from stdout, errors from stderr
  - remember amounts are in milliunits (1000 = $1.00)
never:
  - rely on table-formatted output for parsing
  - omit --yes on destructive commands (will hang in non-TTY)
  - send unvalidated user input directly as IDs
  - assume dollar amounts — always use milliunits
invariants:
  - resource names are singular nouns (budget, account, transaction, category, payee, month)
  - all timestamps are ISO 8601 / UTC
  - all dates are YYYY-MM-DD
  - IDs are UUIDs unless otherwise noted
  - non-TTY mode automatically outputs JSON
  - amounts are integers in milliunits (1000 = $1.00, negative = outflow)
---

# nab Agent Skills

## Command Pattern
```
nab <resource> <action> [flags]
```

## Resources
| Resource | Actions | Description |
|---|---|---|
| budget | list, get, settings | YNAB budgets |
| account | list, get, create | Budget accounts (checking, savings, credit cards, etc.) |
| transaction | list, get, create, update, delete, import | Transactions |
| category | list, get, create, update, group-create, group-update | Budget categories and category groups |
| payee | list, get, update | Payees |
| payee-location | list, get | Payee GPS locations |
| month | list, get | Budget month summaries |
| scheduled-transaction | list, get, create, update, delete | Scheduled/recurring transactions |
| money-movement | list, group-list | Money movements and groups |
| user | get | Authenticated user info |

## Global Flags
| Flag | Short | Description |
|---|---|---|
| --json | | Force JSON output |
| --csv | | Force CSV output |
| --output | -o | table, json, csv, ndjson |
| --fields | | Comma-separated field mask |
| --quiet | -q | Suppress non-essential output |
| --budget | -b | Budget ID, name, or "last-used" |
| --profile | -p | Configuration profile |
| --yes | -y | Skip confirmation prompts |
| --dry-run | | Preview without executing |
| --verbose | -v | Verbose logging to stderr |
| --debug | | Full request/response logging |
| --no-color | | Disable colors |

## Common Workflows

### Discover available commands
```bash
nab schema
nab schema transaction.create
```

### List budgets
```bash
nab budget list --fields id,name,last_modified_on
```

### List transactions with filters
```bash
nab transaction list --fields id,date,amount,payee_name,category_name
nab transaction list --since 2024-01-01
nab transaction list --account <account-id>
```

### Create a transaction (agent-preferred pattern)
```bash
# Step 1: dry-run
nab transaction create --dry-run --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store","category_id":"..."}'
# Step 2: execute after user confirms
nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store","category_id":"..."}'
```

### Delete with safety
```bash
# Step 1: preview
nab transaction delete <id> --dry-run
# Step 2: execute
nab transaction delete <id> --yes
```

### Delta sync (efficient updates)
```bash
# First request returns server_knowledge
nab transaction list
# Subsequent requests only return changes
nab transaction list --last-knowledge 1234
```

## Amounts
All monetary amounts are in **milliunits** (integers):
- `1000` = $1.00
- `-50000` = -$50.00 (outflow)
- `150750` = $150.75 (inflow)

## Error Format (stderr)
```json
{
  "code": "AUTH_ERROR",
  "message": "Token is invalid or expired",
  "guidance": "Run `nab login` to store a new personal access token."
}
```

## Exit Codes
| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General error |
| 2 | Authentication/permission error |
| 3 | Resource not found |
| 4 | Conflict/validation error |
