[中文](../../zh-CN/project/github-backlog.md) | English

# GitHub Backlog Drafts

This page turns the post-`v0.1.0-rc1` roadmap into copy-ready GitHub issues.

Use it when you want to create milestones, labels, and first-wave issues without rethinking the wording each time.

## Suggested Setup

### Milestones

- `v0.1 hardening`
- `parser quality`
- `local memory`
- `contributor surface`
- `provider expansion`

### Labels

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

## Recommended First Wave

Create these first:

1. Align docs with actual RC capabilities
2. Add JSON contract snapshot tests for CLI commands
3. Strengthen HTML body extraction and noise filtering
4. Improve URL normalization for agent-facing actions
5. Surface local index and sync state more clearly
6. Enrich thread summaries for triage loops
7. Add a fake-driver test harness for extension contributors
8. Write a parser contributor guide

Status note:

- item 7 now has a baseline reusable harness on `main` in `pkg/driver/drivertest`
- item 8 is now covered by `docs/en/contributing/parser.md`
- keep those drafts only if follow-up expansion work is needed

## Issue Drafts

## Issue 1

Title: `Align docs with actual RC capabilities`

Type: Feature request

Milestone: `v0.1 hardening`

Labels: `docs`

Body:

```md
## Problem

README, release notes, and specs can drift from the actual state of the current RC. That creates confusion for contributors and agent integrators who are trying to treat the current command and schema surface as stable.

## Proposed change

Reconcile user-facing docs with the current `v0.1.0-rc1` implementation boundary.

## Area

- docs

## Contract impact

This issue should not change machine-facing contracts. It should clarify which existing commands, fields, and behaviors are currently intended to be stable, and which areas remain heuristic.

## Alternatives considered

- Leave docs loosely aligned and rely on release notes only.
- Wait until after more features land.

The narrower and better option is to align docs now, before more drift accumulates.
```

## Issue 2

Title: `Add JSON contract snapshot tests for CLI commands`

Type: Feature request

Milestone: `v0.1 hardening`

Labels: `cmd`, `schema`

Body:

```md
## Problem

Agent integrations depend on stable JSON output shapes, but the current CLI surface does not yet have explicit snapshot or golden coverage for the main machine-facing commands.

## Proposed change

Add snapshot or golden tests for the JSON output of the core CLI commands used by agent workflows.

## Area

- cmd
- schema

## Contract impact

This issue reinforces existing contracts rather than expanding them.

Commands to cover first:

- `mailcli parse`
- `mailcli get`
- `mailcli search`
- `mailcli threads`
- `mailcli thread`

The implementation should also document which fields are intended to be stable in `v0.1` and which fields remain heuristic.

## Alternatives considered

- Rely on package-level tests only.
- Snapshot only `parse` and defer the rest.

Those options leave too much room for accidental CLI contract drift.
```

## Issue 3

Title: `Strengthen HTML body extraction and noise filtering`

Type: Feature request

Milestone: `parser quality`

Labels: `parser`

Body:

```md
## Problem

Some email templates still leak layout noise, navigation blocks, footer clutter, or weak body selection into `body_md`, which reduces usefulness for agent prompts.

## Proposed change

Improve main-content detection and HTML cleanup before Markdown conversion.

## Area

- parser

## Contract impact

This issue should improve output quality without redefining the `StandardMessage` contract.

Target outcomes:

- better selection of the primary message body
- less template chrome in `body_md`
- preserve links, headings, tables, and meaningful images where helpful

## Alternatives considered

- Keep current heuristics and rely on more fixtures only.
- Add provider-specific cleanup rules in shared parser paths.

The first is too weak, and the second would damage the shared architecture.
```

## Issue 4

Title: `Improve URL normalization for agent-facing actions`

Type: Feature request

Milestone: `parser quality`

Labels: `parser`

Body:

```md
## Problem

Agent-facing actions can still expose tracking-heavy or redirect-heavy URLs, which makes actions harder to reason about and reduces prompt clarity.

## Proposed change

Normalize common redirect patterns for extracted action URLs while preserving enough raw data for debugging when needed.

## Area

- parser

## Contract impact

This issue improves action quality but should avoid unnecessary contract changes.

Focus on actions such as:

- unsubscribe
- invoice or payment links
- password reset links
- attachment entry links

## Alternatives considered

- Leave URLs untouched and expect external tools to normalize them.
- Rewrite aggressively without preserving raw source values.

The first is too weak for agent workflows, and the second risks hiding useful forensic context.
```

## Issue 5

Title: `Surface local index and sync state more clearly`

Type: Feature request

Milestone: `local memory`

Labels: `cmd`, `docs`

Body:

```md
## Problem

Users and contributors do not have enough visibility into what `mailcli sync` actually cached, skipped, refreshed, or left untouched in the local index.

## Proposed change

Expose basic sync and local-index state more clearly in CLI output and docs.

## Area

- cmd
- docs

## Contract impact

This issue may affect CLI output shape for sync-related commands, so any JSON changes should be handled carefully and documented.

Potential additions:

- counts for fetched, skipped, refreshed, and failed messages
- clearer explanation of incremental sync behavior
- clearer visibility into which account and mailbox were indexed

## Alternatives considered

- Keep current behavior and document it better only.
- Add verbose logging without structured output.

Docs alone are not enough, and unstructured logs are weak for agents.
```

## Issue 6

Title: `Enrich thread summaries for triage loops`

Type: Feature request

Milestone: `local memory`

Labels: `cmd`, `schema`

Body:

```md
## Problem

Agents still need to load too many full threads before they can decide whether a conversation is worth opening, replying to, or deprioritizing.

## Proposed change

Improve compact thread summaries with a few more triage-friendly signals while keeping output small enough for prompts.

## Area

- cmd
- schema

## Contract impact

This may affect `mailcli threads` and `mailcli search` output shape. If new fields are added, they should be additive and explicitly documented.

Useful candidate signals:

- clearer latest-message timestamp semantics
- participant summary
- action counts
- code presence or code counts

## Alternatives considered

- Keep current summaries minimal and require `mailcli thread` for almost everything.
- Add too many fields and make compact output prompt-heavy.

The right path is a small number of additive, triage-focused fields.
```

## Issue 7

Title: `Add a fake-driver test harness for extension contributors`

Type: Feature request

Milestone: `contributor surface`

Labels: `driver`, `docs`, `good first issue`

Body:

```md
## Problem

External contributors do not yet have a clear, reusable way to validate that a new driver satisfies the expected MailCLI behaviors.

## Proposed change

Add a fake-driver or conformance-style test harness that makes driver expectations explicit and reusable.

## Area

- driver
- docs

## Contract impact

This issue should not expand the driver contract. It should make the existing contract easier to test and understand.

The minimum bar should cover:

- config validation failures
- list behavior
- raw fetch behavior
- send behavior or explicit send-not-supported behavior

## Alternatives considered

- Rely on contributors to copy existing tests manually.
- Document expectations without adding reusable test helpers.

Both options keep contribution cost higher than necessary.
```

## Issue 8

Title: `Write a parser contributor guide`

Type: Feature request

Milestone: `contributor surface`

Labels: `docs`, `parser`, `good first issue`

Body:

```md
## Problem

Parser work is one of the highest-value contribution areas in MailCLI, but the path for making safe parser changes is still too implicit.

## Proposed change

Write a dedicated parser contributor guide that explains how parser work is expected to be structured and tested.

## Area

- docs
- parser

## Contract impact

This issue should not change contracts. It should reduce ambiguity for contributors touching parser behavior.

The guide should cover:

- where parser fixtures live
- how golden tests are organized
- what counts as acceptable heuristic behavior
- what kinds of provider-specific logic do not belong in shared parser code

## Alternatives considered

- Keep parser contribution guidance scattered across code and PR review comments.
- Add only inline code comments.

A dedicated guide creates a much clearer entry path for outside contributors.
```

## RFC Candidate

When one of the following comes up, use the RFC template instead of a normal feature request:

- changing `StandardMessage`
- changing `DraftMessage`
- changing `ReplyDraft`
- changing `SendResult`
- changing stable CLI JSON output
- changing `Driver` contract expectations

Reference:

- [RFC issue template](../../../.github/ISSUE_TEMPLATE/rfc_contract_change.md)
- [Next Development Roadmap](next-roadmap.md)
