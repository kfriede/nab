<p align="center">
  <strong>nab</strong> — manage your YNAB budget from the terminal
</p>

<p align="center">
  <a href="#install">Install</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#commands">Commands</a> •
  <a href="#for-llm-agents">For LLM Agents</a>
</p>

---

**nab** is a CLI for [You Need A Budget (YNAB)](https://www.ynab.com/). It covers the full YNAB API — 33 operations across 10 resources — in a single static binary. Designed from the ground up for humans *and* LLM agents.

```bash
# See your budget at a glance
$ nab transaction list --fields date,payee_name,amount,category_name
DATE        PAYEE_NAME      AMOUNT   CATEGORY_NAME
──────────  ──────────────  ───────  ─────────────
2024-03-15  Grocery Store   -50000   Groceries
2024-03-14  Monthly Rent    -150000  Housing
2024-03-13  Paycheck        350000   Ready to Assign

# Pipe to an agent? Automatic JSON — no flags needed
$ nab budget list --fields id,name | jq '.[].name'
"My Budget"
"Joint Budget"
```

## Install

**Homebrew** (macOS and Linux):

```bash
brew install kfriede/tap/nab
```

**Pre-built binaries** (linux, macOS, Windows — amd64 + arm64):

Download from [GitHub Releases](https://github.com/kfriede/nab/releases/latest).

**Go install**:

```bash
go install github.com/kfriede/nab@latest
```

**From source**:

```bash
git clone https://github.com/kfriede/nab.git && cd nab && make build
```

## Quick Start

```bash
# 1. Authenticate (interactive — stores token in OS keyring)
nab login

# 2. List your budgets
nab budget list

# 3. See your accounts
nab account list --fields name,type,balance

# 4. List recent transactions
nab transaction list --fields date,payee_name,amount,category_name

# 5. Assign money to a category
nab category update <category-id> --month 2024-03-01 --json-input '{"budgeted":500000}'

# 6. Create a transaction
nab transaction create --json-input '{"account_id":"...","date":"2024-03-15","amount":-50000,"payee_name":"Grocery Store"}'

# 7. Manage scheduled transactions
nab scheduled-transaction list --fields date,payee_name,amount,frequency
nab scheduled-transaction create --json-input '{"account_id":"...","date":"2024-04-01","amount":-150000,"frequency":"monthly"}'

# 8. Import from linked accounts
nab transaction import
```

## Commands

### Budget & Accounts (6 operations)

| Resource | Actions |
|---|---|
| `budget` | `list` `get` `settings` |
| `account` | `list` `get` `create` |

### Transactions (11 operations)

| Resource | Actions |
|---|---|
| `transaction` | `list` `get` `create` `update` `delete` `import` |
| `scheduled-transaction` | `list` `get` `create` `update` `delete` |

### Categories & Payees (10 operations)

| Resource | Actions |
|---|---|
| `category` | `list` `get` `create` `update` `group-create` `group-update` |
| `payee` | `list` `get` `update` |
| `payee-location` | `list` `get` |

### Months & Money Movements (4 operations)

| Resource | Actions |
|---|---|
| `month` | `list` `get` |
| `money-movement` | `list` `group-list` |

### Utility

| Command | Description |
|---|---|
| `user get` | Verify authentication |
| `login` | Interactive auth (stores token in OS keyring) |
| `config show\|set\|path` | View and manage configuration |
| `schema [resource.action]` | Runtime command introspection (for agents) |
| `skills` | Agent-optimized usage instructions |
| `version` | Version info (supports `--json`) |
| `completion bash\|zsh\|fish` | Shell completions |

## Output

nab auto-detects what you need:

| Context | What you get |
|---|---|
| **Terminal (TTY)** | Colored, aligned tables |
| **Piped / scripted / agent** | JSON (automatic, no flags needed) |
| `--json` | Force JSON anywhere |
| `--csv` | CSV output |
| `--output ndjson` | One JSON object per line (streaming) |
| `--fields date,amount,payee_name` | Only the fields you ask for |
| `NAB_OUTPUT_FORMAT=json` | Set globally via env var |

**stdout** is always data. Logs, progress, and errors go to **stderr**.

## Safety

Every mutating command supports `--dry-run`:

```bash
$ nab transaction delete <id> --dry-run
Dry run — would delete transaction a69e9a69-8bd0-49b4-8f65-42345bf8e8ec

$ nab transaction delete <id>
Are you sure you want to delete transaction a69e9a69? (y/N):

$ nab transaction delete <id> --yes    # skip prompt (for scripts/agents)
✓ Deleted transaction a69e9a69-8bd0-49b4-8f65-42345bf8e8ec
```

## Amounts

YNAB represents all amounts as integers in **milliunits**: `1000` = $1.00. Negative values are outflows, positive values are inflows.

## For LLM Agents

nab is built to be the CLI that agents don't fight with. Here's why:

### No config needed — just env vars

```bash
export NAB_TOKEN=your-ynab-personal-access-token
export NAB_BUDGET=last-used
# That's it. Every command works now.
```

### Auto-JSON when piped

Agents never see table output. When stdout isn't a TTY, nab automatically outputs JSON:

```bash
# Agent runs this — gets JSON, not a table
nab transaction list --fields id,date,amount,payee_name
```

### Schema introspection

Agents discover commands at runtime instead of hallucinating flags:

```bash
$ nab schema transaction.create
{
  "resource": "transaction",
  "action": "create",
  "httpMethod": "POST",
  "apiPath": "/budgets/{budgetId}/transactions",
  "flags": [
    {"name": "json-input", "type": "string", "required": true, "description": "Full JSON transaction body (amounts in milliunits: 1000 = $1.00)"}
  ],
  "example": "nab transaction create --json-input '{\"account_id\":\"...\",\"date\":\"2024-01-15\",\"amount\":-50000,\"payee_name\":\"Grocery Store\"}'",
  "mutating": true,
  "supportsDryRun": true
}
```

### `--json-input` prevents flag hallucination

Instead of guessing flags, agents send the exact API payload:

```bash
nab transaction create --json-input '{"account_id":"...","date":"2024-01-15","amount":-50000,"payee_name":"Grocery Store"}'
```

### `--fields` minimizes token usage

```bash
# 4 fields instead of 20+ — saves tokens, faster parsing
nab transaction list --fields id,date,amount,payee_name
```

### Structured errors with guidance

When something fails, the error tells the agent exactly what to do next:

```json
{
  "code": "AUTH_ERROR",
  "message": "Token is invalid or expired",
  "guidance": "Run `nab login` to authenticate, or set the NAB_TOKEN environment variable."
}
```

### Safe by default

- `--dry-run` on **every** mutation — agents preview before executing
- `--yes` required for destructive actions in non-TTY (never hangs waiting for input)
- Input validation rejects malformed IDs with clear error messages

### Delta requests (efficient sync)

Most list commands support `--last-knowledge` for incremental updates — only fetch what changed:

```bash
# First request returns server_knowledge
nab transaction list
# Subsequent requests only return changes
nab transaction list --last-knowledge 1234
```

### Agent config files

nab ships config files that agents discover automatically:

| File | Agent | Purpose |
|---|---|---|
| [`AGENTS.md`](AGENTS.md) | All agents (cross-vendor standard) | Full usage spec, rules, patterns, boundaries |
| [`CLAUDE.md`](CLAUDE.md) | Claude Code / Claude Desktop | Points to AGENTS.md + quick reference |
| [`.github/copilot-instructions.md`](.github/copilot-instructions.md) | GitHub Copilot CLI | Project conventions and design principles |
| [`.claude-plugin/`](.claude-plugin/) | Claude Cowork marketplace | Installable plugin with skills |
| [`SKILLS.md`](SKILLS.md) | Any agent (via `nab skills`) | YAML frontmatter + usage patterns |

Agents that clone or work within this repo will automatically pick up the appropriate file. For agents using nab as an *external tool* (not within the repo), set the env vars and run `nab skills` to bootstrap.

### Claude Cowork Plugin

Install nab as a Claude Cowork plugin for natural language budget management:

```
/plugin install https://github.com/kfriede/nab
```

Then ask Claude things like:
- "List my recent transactions"
- "How much did I spend on groceries this month?"
- "Create a transaction for $50 at the grocery store"
- "Assign $500 to my rent category for next month"

## Configuration

```bash
# Interactive setup (stores token in OS keyring)
nab login

# Or configure via environment
export NAB_TOKEN=your-ynab-personal-access-token
export NAB_BUDGET=last-used

# Or config file (~/.config/nab/config.yaml)
nab config set budget last-used
nab config show

# Multiple accounts with named profiles
nab login --profile family
nab login --profile personal
nab budget list --profile family
```

**Precedence**: CLI flags > environment variables > config file.

## Contributing

```bash
git clone https://github.com/kfriede/nab.git
cd nab
make all        # lint + test + build
make test       # just tests
make lint       # just lint
```

## License

MIT
