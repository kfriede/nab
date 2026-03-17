# nab — YNAB CLI for Humans and Agents

This project includes `nab`, a CLI for managing You Need A Budget (YNAB).
If `nab` is installed and available in PATH, use it for all YNAB operations.

## Discovering nab

```bash
# Check if available
which nab

# Learn all commands (preferred for agents)
nab schema

# Full skills reference
nab skills
```

## Using nab

Command pattern: `nab <resource> <action> [flags]`

### Rules

- Use `--fields` on list/get commands to limit output and save tokens
- Use `--dry-run` before any mutating command, then confirm with the user
- Use `--json-input` for complex create/update payloads (avoids flag hallucination)
- Pass `--yes` on confirmed destructive actions (delete)
- Parse JSON from stdout; errors go to stderr with a `guidance` field
- Non-TTY mode automatically outputs JSON — no need for `--json`
- Amounts are in milliunits: 1000 = $1.00, negative = outflow

### Common Patterns

```bash
# List with field selection
nab budget list --fields id,name,last_modified_on

# List transactions with date filter
nab transaction list --since 2024-01-01 --fields id,date,amount,payee_name

# Introspect a command before using it
nab schema transaction.create

# Create with JSON input
nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store"}'

# Safe mutation: dry-run first, then execute
nab transaction create --dry-run --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000}'
nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000}'

# Delta requests (efficient sync)
nab transaction list --last-knowledge 1234
```

### Environment Setup

```bash
export NAB_TOKEN=<your-ynab-personal-access-token>
export NAB_BUDGET=last-used
```

## Invariants

- Resource names are singular nouns: budget, account, transaction, category, payee, month
- All timestamps are ISO 8601 / UTC
- All dates are YYYY-MM-DD
- IDs are UUIDs
- Amounts are integers in milliunits (1000 = $1.00)
- `--budget last-used` resolves to the most recently accessed budget

## Boundaries

- **Always do**: Use `--dry-run` before mutations, use `--fields` to minimize output
- **Ask the user**: Before any destructive action (delete)
- **Never do**: Bypass `--yes` on destructive commands without user confirmation, send unvalidated input as IDs
