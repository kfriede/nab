# Copilot Instructions for nab

## Project Overview

**nab** is a command-line interface for You Need A Budget (YNAB). It wraps the YNAB API to provide a fast, scriptable CLI experience for budget management. The design philosophy follows [clig.dev](https://clig.dev) and takes inspiration from `gh`, `kubectl`, and `aws` — CLIs that feel like natural extensions of the terminal.

**A core differentiator of nab is first-class LLM/agent compatibility.** It is designed to be used seamlessly by AI coding agents (Copilot CLI, Claude Code, Gemini CLI, etc.) as well as human operators. Every design decision should consider both audiences: humans get beautiful, discoverable output; agents get structured, minimal, parseable output with runtime introspection and clear safety rails.

## CLI Name & Command Grammar

The binary/command name is `nab`. Commands follow a **resource action** (noun-verb) pattern with flags over positional arguments:

```
nab <resource> <action> [flags]
```

Examples:
- `nab budget list`
- `nab transaction list --since 2024-01-01`
- `nab account list --budget last-used`
- `nab category list --json`
- `nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000}'`

Resources are **singular nouns** (e.g., `nab transaction`, not `nab transactions`). Actions use clear, unambiguous verbs: `list`, `get`, `create`, `update`, `delete`.

For complex or nested input, support a **`--json-input` flag** that accepts a full JSON payload as the request body. Document that agents should prefer `--json-input` to avoid flag hallucination:
```
nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store"}'
```

## Key Concepts

- **Budget**: A YNAB budget containing accounts, categories, and transactions.
- **Account**: A financial account (checking, savings, credit card, etc.) within a budget.
- **Transaction**: A financial transaction (inflow or outflow) within an account.
- **Category**: Budget categories organized into category groups.
- **Payee**: A payee (merchant, person, etc.) associated with transactions.
- **Month**: A budget month summary with category details.
- **Milliunits**: YNAB represents all amounts as integers in milliunits (1000 = $1.00).

## Discoverability

- Running `nab` with no arguments prints a concise overview with grouped subcommands and usage examples.
- `--help` on every command shows **examples first**, then flags and descriptions.
- Typos should trigger "did you mean …?" suggestions.
- **`nab schema <resource>.<action>`** — Runtime introspection command that returns a JSON schema for any command.
- **`nab skills`** — Dumps concise, agent-optimized usage instructions.

## Output & Formatting

- **Default (TTY detected)**: Human-readable colored, aligned tables.
- **Non-TTY (piped/agent)**: Automatically switch to JSON output.
- **`--json`**: Full JSON output, explicitly requested.
- **`--csv`**: CSV output for spreadsheets and simple parsing.
- **`--output <format>`**: Alternative long-form flag accepting `table`, `json`, `csv`, `ndjson`.
- **`--fields <field1,field2,...>`**: Field mask to select only specific fields.
- **`--no-color`**: Force disable colors. Also respect the `NO_COLOR` environment variable.
- **`--quiet` / `-q`**: Suppress non-essential output.
- **`NAB_OUTPUT_FORMAT`** env var: Set to `json` globally for structured output.
- **stdout** is for primary data output only. Logs, progress, errors, and prompts go to **stderr**.

## Feedback & Safety

- Every mutating action prints a clear confirmation.
- Destructive actions (delete) **prompt for confirmation** unless `--yes` is passed.
- **`--dry-run`** is supported on every mutating command.
- Errors are human-readable and actionable. Agent-native error output includes `code`, `message`, and `guidance` fields.

## Configuration & Auth

Precedence (highest wins): **CLI flags → environment variables → project config → user config**.

- User config lives at `~/.config/nab/config.yaml` (XDG-compliant). Support `NAB_CONFIG` env var to override.
- Auth: `nab login` stores a YNAB personal access token in the OS keyring.
- Key environment variables: `NAB_TOKEN`, `NAB_BUDGET`, `NAB_PROFILE`, `NAB_NO_COLOR`, `NAB_DEBUG`.

## API Notes

- YNAB API base URL: `https://api.ynab.com/v1`
- Authentication: Bearer token in Authorization header.
- All responses are wrapped in `{"data": {...}}` — the client unwraps this.
- Error responses: `{"error": {"id": "...", "name": "...", "detail": "..."}}`.
- Amounts are in milliunits (1000 = $1.00, negative = outflow).
- Delta requests supported via `server_knowledge` parameter for efficient syncing.
- No pagination needed — the YNAB API returns all results.

## Architecture & Code Conventions

- **Three-layer separation**: CLI parsing layer → API client library → output/formatting layer.
- The API client is a standalone package (not tangled with CLI code).
- Tests for: API client functions, output formatting, argument parsing/validation.
- Detect TTY in the output layer to decide between rich and plain rendering.
- `--verbose` / `-v` for debug-level logging to stderr.
- `--debug` for full request/response body logging.
