package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().Bool("local", false, "Install to the current project instead of user-level")
}

var installCmd = &cobra.Command{
	Use:   "install <agent>",
	Short: "Install nab skill for an agent harness",
	Long: `Install nab configuration files for a specific AI agent harness.

Supported agents:
  claude-code    Claude Code / Claude Desktop (~/.claude/skills/nab/)
  copilot        GitHub Copilot CLI (~/.copilot/copilot-instructions.md)
  codex          OpenAI Codex CLI (~/.codex/AGENTS.md)
  gemini         Gemini CLI (~/.gemini/GEMINI.md)
  opencode       OpenCode (~/.config/opencode/agents/)

By default, installs to the user-level config directory (applies to all
projects). Use --local to install to the current project directory instead.

Examples:
  nab install claude-code              User-level Claude Code skill
  nab install copilot --local          Project-level Copilot instructions
  nab install codex                    User-level Codex AGENTS.md
  nab install gemini                   User-level Gemini GEMINI.md
  nab install opencode --local         Project-level OpenCode agent`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"claude-code", "copilot", "codex", "gemini", "opencode"},
	RunE:      runInstall,
}

type agentHarness struct {
	name     string
	userDir  func() string          // returns the user-level directory
	localDir func() string          // returns the project-level directory
	write    func(dir string) error // writes the config files
	files    string                 // description of what gets written
}

func runInstall(cmd *cobra.Command, args []string) error {
	agent := strings.ToLower(args[0])
	local, _ := cmd.Flags().GetBool("local")

	harness, ok := agentHarnesses()[agent]
	if !ok {
		return fmt.Errorf("unknown agent %q\n\nSupported agents: claude-code, copilot, codex, gemini, opencode", agent)
	}

	var dir string
	if local {
		dir = harness.localDir()
		printer.Status(fmt.Sprintf("Installing nab for %s (project-level)...", harness.name))
	} else {
		dir = harness.userDir()
		printer.Status(fmt.Sprintf("Installing nab for %s (user-level)...", harness.name))
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := harness.write(dir); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	printer.Success(fmt.Sprintf("Installed %s to %s", harness.files, dir))
	return nil
}

func agentHarnesses() map[string]agentHarness {
	home, _ := os.UserHomeDir()

	return map[string]agentHarness{
		"claude-code": {
			name:     "Claude Code",
			userDir:  func() string { return filepath.Join(home, ".claude", "skills", "nab") },
			localDir: func() string { return filepath.Join(".claude", "skills", "nab") },
			write:    writeClaudeCodeSkill,
			files:    "SKILL.md",
		},
		"copilot": {
			name:     "GitHub Copilot CLI",
			userDir:  func() string { return filepath.Join(home, ".copilot") },
			localDir: func() string { return ".github" },
			write:    writeCopilotInstructions,
			files:    "copilot-instructions.md",
		},
		"codex": {
			name:     "OpenAI Codex CLI",
			userDir:  func() string { return filepath.Join(home, ".codex") },
			localDir: func() string { return "." },
			write:    writeCodexAgentsMD,
			files:    "AGENTS.md",
		},
		"gemini": {
			name:     "Gemini CLI",
			userDir:  func() string { return filepath.Join(home, ".gemini") },
			localDir: func() string { return "." },
			write:    writeGeminiMD,
			files:    "GEMINI.md",
		},
		"opencode": {
			name:     "OpenCode",
			userDir:  func() string { return filepath.Join(home, ".config", "opencode", "agents") },
			localDir: func() string { return filepath.Join(".opencode", "agents") },
			write:    writeOpenCodeAgent,
			files:    "nab.md",
		},
	}
}

// nabInstructions returns the core agent instructions shared across all harnesses.
func nabInstructions() string {
	return `# nab — YNAB CLI for Humans and Agents

This project includes nab, a CLI for managing You Need A Budget (YNAB).
If nab is installed and available in PATH, use it for all YNAB operations.

## Discovering nab

` + "```bash" + `
# Check if available
which nab

# Learn all commands (preferred for agents)
nab schema

# Full skills reference
nab skills
` + "```" + `

## Using nab

Command pattern: ` + "`nab <resource> <action> [flags]`" + `

### Rules

- Use ` + "`--fields`" + ` on list/get commands to limit output and save tokens
- Use ` + "`--dry-run`" + ` before any mutating command, then confirm with the user
- Use ` + "`--json-input`" + ` for complex create/update payloads (avoids flag hallucination)
- Pass ` + "`--yes`" + ` on confirmed destructive actions (delete)
- Parse JSON from stdout; errors go to stderr with a ` + "`guidance`" + ` field
- Non-TTY mode automatically outputs JSON — no need for ` + "`--json`" + `
- Amounts are in milliunits: 1000 = $1.00, negative = outflow

### Common Patterns

` + "```bash" + `
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
` + "```" + `

### Environment Setup

` + "```bash" + `
export NAB_TOKEN=<your-ynab-personal-access-token>
export NAB_BUDGET=last-used
` + "```" + `

## Invariants

- Resource names are singular nouns: budget, account, transaction, category, payee, month
- All timestamps are ISO 8601 / UTC
- All dates are YYYY-MM-DD
- IDs are UUIDs
- Amounts are integers in milliunits (1000 = $1.00)

## Boundaries

- **Always do**: Use --dry-run before mutations, use --fields to minimize output
- **Ask the user**: Before any destructive action (delete)
- **Never do**: Bypass --yes on destructive commands without user confirmation
`
}

func writeClaudeCodeSkill(dir string) error {
	content := `---
name: nab
description: "Manage YNAB budgets, accounts, transactions, and categories using the nab CLI. Use when the user asks about budgets, transactions, spending, categories, or YNAB."
---

` + nabInstructions()

	return os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644)
}

func writeCopilotInstructions(dir string) error {
	return os.WriteFile(filepath.Join(dir, "copilot-instructions.md"), []byte(nabInstructions()), 0o644)
}

func writeCodexAgentsMD(dir string) error {
	path := filepath.Join(dir, "AGENTS.md")
	return writeOrAppend(path, nabInstructions())
}

func writeGeminiMD(dir string) error {
	path := filepath.Join(dir, "GEMINI.md")
	return writeOrAppend(path, nabInstructions())
}

func writeOpenCodeAgent(dir string) error {
	return os.WriteFile(filepath.Join(dir, "nab.md"), []byte(nabInstructions()), 0o644)
}

// writeOrAppend writes content to a file, or appends if the file exists
// and doesn't already contain nab instructions.
func writeOrAppend(path string, content string) error {
	if existing, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(existing), "nab") && strings.Contains(string(existing), "YNAB") {
			return fmt.Errorf("%s already contains nab instructions", path)
		}
		// Append to existing file
		content = string(existing) + "\n\n" + content
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
