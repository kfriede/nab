# 0.2.6 candidate

Mutating requests (POST, PUT, PATCH, DELETE and other non-read methods) now make one attempt. `--mutation-attempts=1` and `NAB_MUTATION_ATTEMPTS=1` explicitly assert the ceiling; other or conflicting supplied values fail before command execution. There is no retry-enable override.

The mutation transport uses fresh HTTP/1 connections, no redirect following and non-rewindable request bodies to exclude client-internal replay paths. Reads retain their existing bounded application retry loop; 429 Retry-After accepts seconds or HTTP dates and caps sleep at 30 seconds.

Transport failure, redirects, 5xx, incomplete bodies and invalid mutation success/rejection envelopes return typed `MUTATION_OUTCOME_UNKNOWN` with nonzero CLI status and reconciliation guidance. An unrecognized result cannot prove that a write did not happen. The caller must GET the exact target and verify intended fields and unrelated-field preservation. Missing, stale or incomplete readback leaves the outcome unknown. Never automatically replay the command or entire batch. Dry-run and applicable user confirmation remain required for a genuinely new mutation.

Semantic endpoint validation and per-item batch acceptance remain the consuming workflow's responsibility. Transport success is not independent exact-target acceptance.

This file records a candidate, not an upstream release or installation receipt. Normal publication is a reviewed source change followed by a `v*` tag invoking `.github/workflows/release.yml` / GoReleaser. That release also writes `kfriede/homebrew-tap`; use only an explicitly approved publisher. The shared GitHub read consumer is not a publisher.
