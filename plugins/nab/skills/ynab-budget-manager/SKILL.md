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

## Finding nab

Before using any `nab` commands, determine which binary to use:

```bash
# Check if nab is in PATH (standard install)
which nab && NAB_CMD="nab"

# If not found, check for sideloaded binary in workspace (Claude Cowork)
if [ -z "${NAB_CMD:-}" ] && [ -x ./nab ]; then NAB_CMD="./nab"; fi

# If still not found, install it
if [ -z "${NAB_CMD:-}" ]; then
  echo "nab not found. Run the setup script:" >&2
  echo "  curl -fsSL https://raw.githubusercontent.com/kfriede/nab/main/scripts/cowork-setup.sh | bash" >&2
fi
```

Use `$NAB_CMD` (or `./nab` in Cowork, `nab` in Claude Code / terminal) for all commands below.

## Prerequisites

### Option A: Standard install (Claude Code / terminal)
```bash
brew install kfriede/tap/nab
# or: go install github.com/kfriede/nab@latest
```

### Option B: Claude Cowork (sideloaded binary)
Cowork runs in a sandboxed Linux VM — host binaries are not available. Run this on your **host machine** to prepare a workspace:
```bash
nab cowork setup --dir ~/Cowork
```
This downloads the correct Linux binary and shows the environment variables to configure in Cowork.

Alternatively, if nab isn't installed on the host, use the standalone script:
```bash
curl -fsSL https://raw.githubusercontent.com/kfriede/nab/main/scripts/cowork-setup.sh | bash
```

### Configure access
```bash
export NAB_TOKEN=<your-ynab-personal-access-token>
export NAB_BUDGET=last-used
```

You can create a personal access token at https://app.ynab.com/settings/developer

> **Cowork tip:** Set `NAB_TOKEN` and `NAB_BUDGET` in your Cowork environment settings (Settings → Environment Variables) so they persist across sessions.

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

## Troubleshooting: Network Access (Cowork)

nab requires outbound HTTPS access to `api.ynab.com`. Claude Cowork's sandbox blocks network access by default.

If you see connection errors (`EGRESS_BLOCKED`, timeout, connection refused):

1. The user must add `*.ynab.com` to the domain allowlist in Claude Desktop
2. Go to: **Claude Desktop → Settings → Capabilities → Domain allowlist**
3. Under "Additional allowed domains", add: `*.ynab.com`
4. Restart the Cowork session

Tell the user this if any nab command fails with a network error.
