# nab — YNAB CLI

This project includes `nab`, a CLI for managing You Need A Budget (YNAB).

Run `nab skills` for the full agent reference, or `nab schema` to discover commands.

## Quick Reference

```bash
nab <resource> <action> [flags]
```

**Always**: use `--fields` on reads, `--dry-run` before writes, `--json-input` for complex payloads.

**Never**: parse table output, omit `--yes` on destructive commands in non-interactive mode.

See [AGENTS.md](./AGENTS.md) for the complete specification.
