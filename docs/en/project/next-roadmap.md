[中文](../../zh-CN/project/next-roadmap.md) | English

# Next Development Roadmap

This document defines the recommended work order after `v0.1.0-rc1`.

The goal is not to add random surface area.

The goal is to turn the current RC into a stable, contributor-friendly open-source project for agent developers.

The next phase also clarifies a product and implementation direction: **the core runtime and official examples should be Go-only**. MailCLI should ship as a binary that agents can call reliably and users can install without understanding or deploying another language runtime.

## Recently Completed On `main`

The original RC hardening pass already shipped several important pieces:

- CLI JSON snapshot coverage for `parse`, `get`, `search`, `threads`, and `thread`
- stronger HTML body extraction and noise filtering
- more aggressive but bounded tracked URL normalization
- clearer `sync` index-state output
- richer thread triage signals such as `code_count`, `action_count`, and `participant_count`
- a standard maintainer workflow for refreshing and checking local thread demo artifacts
- an RFC issue template for contract-sensitive changes
- reusable driver conformance harness in `pkg/driver/drivertest`
- parser contributor workflow docs
- more reliable local search and thread ranking semantics
- zero-network-first README onboarding and a minimal reply artifact that shows MailCLI deriving outbound defaults

For copy-ready GitHub issue drafts, see [GitHub Backlog Drafts](github-backlog.md).
For the realistic maintainer-led sequence, see [Internal Development Priority](internal-priority.md).
For the rationale behind adopting harness-style workflows and how they differ
from ordinary skills, see [Agent Harness Strategy](agent-harness-strategy.md).

## Priority Order

1. Tighten existing-mailbox setup and configuration as Go core behavior.
2. Make inbox/thread summaries, priority, and todo extraction structured Go command output.
3. Improve attachment, invoice, code, and action-link extraction in the Go parser/schema.
4. Make draft, confirmation, execution, and operation logging a Go CLI contract.
5. Do not prioritize dedicated Agent mailbox hosting or broad provider expansion yet.

## Maintainer Rules

- prefer stable machine-facing contracts over feature count
- keep official executable behavior in Go; do not introduce non-Go runtime prerequisites for installation or agent use
- keep AI provider integration language-neutral through JSON contracts, while keeping official examples and long-term maintenance Go-first
- keep provider-specific business logic out of shared parser and composer layers
- treat parser heuristics as product work, not cleanup work
- optimize for "easy to contribute to" as much as "useful to use"

## Next Go Mainline

These are the four main tracks after the current product direction decision:

1. **Existing mailbox setup and configuration: Go core.**
   Help users connect the Gmail, Outlook, QQ, 163, corporate mailbox, or local `.eml` data they already have, instead of requiring a dedicated Agent mailbox first.
   The first concrete slice is a Go-only config path: `config init`, `config doctor`, `config test`, and `config capabilities`.
2. **Inbox/thread summaries, priority, and todo extraction: Go provides structured data and commands; AI providers remain external.**
   Go owns stable retrieval, aggregation, fields, and lightweight signals. LLMs can interpret, summarize, and draft recommendations outside the core.
3. **Attachment, invoice, code, and action-link extraction: Go parser/schema.**
   These are high-value signals in real inboxes and should be part of the core parser quality bar.
4. **Draft, confirmation, execution, and operation logging: Go CLI contract.**
   Automation must be controllable. High-impact actions should produce auditable intents before execution and machine-readable results afterward.

## Milestone 1: v0.1 Hardening

Goal: make the current RC easier to trust, easier to document, and safer to build against.

### Done when

- README, release notes, and specs describe the same stable boundary
- CLI JSON outputs are snapshot-tested for core commands
- parser heuristics that are still unstable are clearly marked in docs
- the contributor path for contract changes is explicit

### Suggested GitHub issues

#### Issue: Align docs with actual RC capabilities

Status: baseline alignment is completed on `main`, but should be kept current as contracts evolve.

- Area: docs
- Problem: roadmap and status text can drift from actual command and schema support
- Scope:
  - reconcile README roadmap checkboxes with current implementation
  - verify release notes and specs point to the same stable contracts
  - add one maintainer-facing roadmap page for next tasks
- Deliverable: docs-only PR

#### Issue: Add JSON contract snapshot tests for CLI commands

Status: completed for the current core command set.

- Area: cmd, schema, tests
- Problem: agent integrations need stable output shapes
- Scope:
  - add golden or snapshot coverage for `parse`, `get`, `search`, `threads`, and `thread`
  - pin the fields that are intended to stay stable in `v0.1`
  - document which fields remain heuristic
- Deliverable: tests plus small spec/doc update

#### Issue: Add RFC template for contract-changing proposals

Status: completed.

- Area: docs, governance
- Problem: schema and CLI changes need a predictable review path
- Scope:
  - add a GitHub issue template for RFC-style proposals
  - list when contributors must open an RFC instead of a normal feature request
  - point driver and schema docs to that path
- Deliverable: `.github` template and docs update

## Milestone 2: Parser / Schema Quality

Goal: improve the one area that most directly affects agent usefulness.

### Done when

- HTML body extraction is more reliable on noisy templates
- redirect-heavy tracking links are cleaned more aggressively
- structured output for attachments, invoices, codes, and action links is more complete
- fixture coverage better represents real agent workflows
- parser regressions are easier to catch before release

### Suggested GitHub issues

#### Issue: Strengthen HTML body extraction and noise filtering

Status: baseline improvement completed; keep treating this as ongoing parser product work.

- Area: parser
- Problem: some templates still leak navigation, footer, or layout noise into `body_md`
- Scope:
  - improve main-content detection before HTML-to-Markdown conversion
  - keep useful structures such as links, headings, tables, and key images
  - add fixtures for newsletter, alert, and transactional layouts that currently degrade
- Deliverable: parser changes with golden tests

#### Issue: Improve URL normalization for agent-facing actions

Status: baseline improvement completed; follow-up work should focus on more fixtures, not broader rules by default.

- Area: parser
- Problem: action URLs can still contain provider tracking wrappers that reduce agent clarity
- Scope:
  - normalize common redirect patterns without breaking legitimate URLs
  - preserve original raw URL where needed for debugging
  - cover unsubscribe, invoice, reset, and attachment entry points
- Deliverable: parser changes, tests, and spec notes

#### Issue: Expand parser corpus for real-world edge cases

- Area: parser, tests
- Problem: the parser is only as strong as the fixture set behind it
- Scope:
  - add more anonymized fixtures for multilingual, thread-heavy, bounce, and layout-heavy mail
  - group fixtures by behavior, not provider brand
  - document what each fixture is meant to protect
- Deliverable: new fixtures and focused regression coverage

#### Issue: Promote inbound attachments and invoice entry points to first-class structured output

Status: baseline completed with inbound MIME attachment metadata, existing invoice/code/action extraction, and fixture-backed contract coverage.

- Area: parser, schema, cmd
- Problem: high-value inbox information often lives in attachments, invoice entry points, download links, or body URLs; agents should not rely on full-text guessing.
- Scope:
  - design a stable representation for inbound attachments and attachment entry points in `StandardMessage`
  - distinguish real MIME attachments, body download links, and invoice view/download actions
  - add fixtures and golden coverage for invoices, attachment notices, and multilingual codes
- Deliverable: schema/parser changes, CLI output, tests, and spec updates

## Milestone 3: Inbox / Thread Intelligence

Goal: help AI retrieve information from a user's existing mailbox faster and more accurately, not just run keyword search.

### Done when

- sync behavior is easier to reason about across reruns
- cache/index state is more visible to users and contributors
- thread metadata reduces unnecessary full-message loads
- inbox/thread summaries support common triage decisions such as priority, todos, and needs-reply

### Suggested GitHub issues

#### Issue: Surface local index and sync state more clearly

Status: baseline sync/index visibility improvements completed.

- Area: cmd, internal/index, docs
- Problem: agents and contributors need more visibility into what is cached locally
- Scope:
  - expose basic sync/index stats through CLI output
  - document what `sync` skips, overwrites, or refreshes
  - make local retrieval semantics easier to explain
- Deliverable: command output improvements plus docs

#### Issue: Enrich thread summaries for triage loops

Status: baseline thread-summary expansion and local ranking refinement completed.

- Area: internal/index, cmd, schema
- Problem: agents still need too many full thread loads for common triage decisions
- Scope:
  - evaluate compact additions such as latest timestamp clarity, participant summary, or action/code counts
  - keep output small enough for agent prompts
  - snapshot-test the chosen shape
- Deliverable: schema/output adjustment with docs

#### Issue: Add priority and todo extraction signals for inbox/thread triage

Status: baseline evidence/enrichment contract completed. Provider judgment quality remains evaluation work and is intentionally not represented by parser snapshots.

- Area: internal/index, cmd, schema
- Problem: the common user question is not just "find mail"; it is "what matters, what needs handling, and what should happen next."
- Scope:
  - design lightweight, explainable priority / needs_reply / todo-like signals
  - keep Go output as structured candidate signals without binding core to a specific LLM
  - add fixed output coverage to local fixtures and the thread demo
- Deliverable: Go command output, schema/spec updates, and snapshot tests

## Milestone 4: Safe Outbound Loop And Operation Logs

Goal: move AI automation from "execute the command directly" to "prepare intent, confirm execution, record result."

### Done when

- high-impact actions such as `send`, `reply`, `delete`, `move`, and `mark` can produce a dry-run or intent first
- confirmation uses a stable token or intent id so agents do not accidentally execute a different action
- execution results and failure reasons are written to machine-readable operation logs
- operation logging does not depend on provider-specific behavior

### Suggested GitHub issues

#### Issue: Add prepare / confirm flow for dangerous actions

- Area: cmd, schema, docs
- Problem: agents can draft send/delete/move actions, but direct execution amplifies mistakes.
- Scope:
  - design `prepare` output for sending and mailbox mutations
  - execute the same intent through an intent id or confirmation token
  - keep dry-run, prepare, and confirm output readable by agents
- Deliverable: Go CLI contract, schema, tests, and docs

#### Issue: Add local operation logs

- Area: cmd, internal, docs
- Problem: agent automation needs auditability: what ran, why it failed, and which message or thread it targeted.
- Scope:
  - record operation type, account, target ID, intent id, result, error code, and timestamp
  - provide `mailcli operations list/show` or equivalent query commands
  - avoid storing secret fields and full sensitive bodies
- Deliverable: Go storage/CLI, tests, and security notes

## Deferred: Provider Expansion And Dedicated Agent Mailboxes

Tencent Agent Mail shows that a dedicated Agent mailbox identity can be valuable, but MailCLI's current mainline is not hosted mailbox service. The next phase should first help users process their existing mailboxes with AI.

Therefore defer:

- OAuth-heavy auth flows in core
- hosted `@agent` mailbox identity
- broad provider expansion
- runtime plugin loading

Re-evaluate dedicated Agent mailbox modes or new providers only after the Go core setup, retrieval, extraction, and safe execution loop are stable.

## Recommended Milestones In GitHub

- `v0.1 hardening`
- `go-only core`
- `existing mailbox setup`
- `inbox intelligence`
- `parser actions and attachments`
- `safe outbound automation`

## Recommended Labels

- `parser`
- `driver`
- `composer`
- `cmd`
- `schema`
- `docs`
- `examples`
- `governance`
- `good first issue`
- `rfc`

## Do Not Prioritize Yet

- full terminal mail client UX
- OAuth-heavy auth flows in core
- hosted dedicated Agent mailbox service
- runtime plugin loading
- provider-specific business policy in shared layers
- trying to solve every mailbox vendor at once
- adding a second official runtime path beside Go

The strongest next move is to use Go to make the boundary for "AI safely processes a user's existing mailbox" sharper, not broader.
