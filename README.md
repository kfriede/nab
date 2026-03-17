# nab

A command-line interface for [You Need A Budget (YNAB)](https://www.ynab.com/) — built for humans and LLM agents.

nab wraps the YNAB API to provide a fast, scriptable CLI for budget management. Humans get beautiful colored tables; agents get structured JSON automatically.

## Installation

### Homebrew

```bash
brew install kfriede/tap/nab
```

### Go install

```bash
go install github.com/kfriede/nab@latest
```

### Binary releases

Download pre-built binaries from [GitHub Releases](https://github.com/kfriede/nab/releases).

## Quick Start

```bash
# 1. Authenticate with your YNAB personal access token
nab login

# 2. List your budgets
nab budget list

# 3. List recent transactions
nab transaction list --fields id,date,amount,payee_name
```

## Usage

```
nab <resource> <action> [flags]
```

Resources are singular nouns. Actions use clear verbs.

| Resource               | Actions                                                 |
| ---------------------- | ------------------------------------------------------- |
| `budget`               | `list`, `get`, `settings`                               |
| `account`              | `list`, `get`, `create`                                 |
| `transaction`          | `list`, `get`, `create`, `update`, `delete`, `import`   |
| `category`             | `list`, `get`, `create`, `update`, `group-create`, `group-update` |
| `payee`                | `list`, `get`, `update`                                 |
| `payee-location`       | `list`, `get`                                           |
| `month`                | `list`, `get`                                           |
| `scheduled-transaction`| `list`, `get`, `create`, `update`, `delete`             |
| `money-movement`       | `list`, `group-list`                                    |
| `user`                 | `get`                                                   |

Run `nab` with no arguments for a full command overview, or `nab <command> --help` for details.

## Output Formats

By default, nab renders colored tables in TTY mode and switches to JSON automatically in non-TTY (piped) mode.

```bash
# Explicit JSON
nab budget list --json

# CSV
nab transaction list --csv

# Select specific fields
nab transaction list --fields id,date,amount,payee_name

# Long-form output flag (table, json, csv, ndjson)
nab account list --output ndjson
```

You can also set `NAB_OUTPUT_FORMAT=json` to default to JSON globally.

## Agent / LLM Usage

nab is designed as a first-class tool for AI coding agents (Copilot CLI, Claude Code, Gemini CLI, etc.).

```bash
# Discover all commands and their schemas
nab schema

# Introspect a specific command
nab schema transaction.create

# Full agent-optimized reference
nab skills
```

**Key patterns for agents:**

- **Structured input** — use `--json-input` for create/update to avoid flag hallucination:
  ```bash
  nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store"}'
  ```
- **Dry-run first** — validate mutations before executing:
  ```bash
  nab transaction create --dry-run --json-input '{"account_id":"...","amount":-50000}'
  ```
- **Auto-JSON** — non-TTY output is JSON by default; no `--json` flag needed.
- **Structured errors** — errors go to stderr with `code`, `message`, and `guidance` fields.
- **Field selection** — use `--fields` to minimize output and save tokens.

## Configuration

```bash
# Store your YNAB personal access token in the OS keyring
nab login
```

### Environment Variables

| Variable           | Description                              |
| ------------------ | ---------------------------------------- |
| `NAB_TOKEN`        | YNAB personal access token               |
| `NAB_BUDGET`       | Default budget (`last-used` or a UUID)   |
| `NAB_PROFILE`      | Active configuration profile             |
| `NAB_OUTPUT_FORMAT` | Default output format (`json`, `csv`)   |
| `NAB_NO_COLOR`     | Disable colored output                   |
| `NAB_DEBUG`        | Enable debug logging                     |

### Config File

User config lives at `~/.config/nab/config.yaml` (XDG-compliant). Override with `NAB_CONFIG`.

Profiles are supported via `--profile` for managing multiple YNAB accounts.

**Precedence** (highest wins): CLI flags → environment variables → project config → user config.

## Safety

- **`--dry-run`** is supported on all mutating commands.
- **`--yes`** is required to skip confirmation on destructive operations (e.g., delete).
- TTY mode prompts for confirmation before destructive actions.

## Amounts

YNAB represents all amounts as integers in **milliunits**: `1000` = $1.00. Negative values are outflows, positive values are inflows.

## Development

Requires Go 1.25+.

```bash
make build    # build binary
make test     # run tests
make lint     # run linter (golangci-lint)
make all      # lint + test + build
make clean    # remove build artifacts
```

## License

[MIT](LICENSE) © Kevin Friedemann
