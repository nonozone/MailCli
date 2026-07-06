# MailCLI Specability Governance

## Purpose

Define the project-level Specability contract for MailCLI governance. MailCLI is
a Go CLI that lets AI agents and humans work with existing mailboxes through
stable commands, JSON schemas, local configuration, MCP integration, and
explicit confirmation boundaries for mailbox mutations.

This README is the harness-facing contract layer. The repository root README
remains the user-facing installation and usage document.

## Principles

- MUST: Keep the runtime product implementation in Go, with CLI entrypoints under `cmd/` and reusable logic under `pkg/` or `internal/`.
- MUST: Preserve stable machine-readable JSON contracts for agent workflows; add fields compatibly or document breaking changes before releases.
- MUST: Store mailbox credentials as environment-variable references or future secret-provider references, not as raw passwords, app passwords, authorization codes, or OAuth tokens in committed config examples.
- MUST: Keep read and setup surfaces available to agents without granting destructive mail actions by default; send, delete, move, and mark operations must stay behind explicit CLI capabilities, confirmation, dry-run, or operation-log boundaries.
- SHOULD: Prefer existing-mailbox onboarding through provider presets and normalized IMAP/SMTP account config rather than provider-specific core driver logic.
- SHOULD: Validate user-visible workflow changes with Go tests plus a CLI smoke test or demo check when the change affects command behavior.

## Boundaries

- Does NOT handle: end-user product documentation and install instructions (see: ../../README.md).
- Does NOT handle: localized end-user product documentation (see: ../../README.zh-CN.md).
- Does NOT handle: command-specific behavioral specs (see: ../en/spec/).
- Does NOT handle: localized command-specific behavioral specs (see: ../zh-CN/spec/).
- Does NOT handle: release workflow implementation (see: ../../.github/workflows/).

## Open Questions

- [ ] Should MailCLI add a repo-root structured README for full Specability project-map coverage without changing the product README? (open since: 2026-07)
- [ ] Should provider-specific OAuth and keyring integrations become core commands or pluggable extensions? (open since: 2026-07)
- [ ] Which mailbox mutation commands should be exposed through MCP after operation logs and confirmation contracts are hardened? (open since: 2026-07)
