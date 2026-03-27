[中文](../../zh-CN/project/next-roadmap.md) | English

# Next Development Roadmap

This document defines the recommended work order after `v0.1.0-rc1`.

The goal is not to add random surface area.

The goal is to turn the current RC into a stable, contributor-friendly open-source project for agent developers.

For copy-ready GitHub issue drafts, see [GitHub Backlog Drafts](github-backlog.md).

## Priority Order

1. Harden the current RC boundary.
2. Improve parser quality where agents feel pain today.
3. Make local memory and thread workflows more reliable.
4. Reduce contribution overhead for new driver and parser contributors.
5. Expand providers only after the shared contracts are clearer.

## Maintainer Rules

- prefer stable machine-facing contracts over feature count
- keep provider-specific business logic out of shared parser and composer layers
- treat parser heuristics as product work, not cleanup work
- optimize for "easy to contribute to" as much as "useful to use"

## Milestone 1: v0.1 Hardening

Goal: make the current RC easier to trust, easier to document, and safer to build against.

### Done when

- README, release notes, and specs describe the same stable boundary
- CLI JSON outputs are snapshot-tested for core commands
- parser heuristics that are still unstable are clearly marked in docs
- the contributor path for contract changes is explicit

### Suggested GitHub issues

#### Issue: Align docs with actual RC capabilities

- Area: docs
- Problem: roadmap and status text can drift from actual command and schema support
- Scope:
  - reconcile README roadmap checkboxes with current implementation
  - verify release notes and specs point to the same stable contracts
  - add one maintainer-facing roadmap page for next tasks
- Deliverable: docs-only PR

#### Issue: Add JSON contract snapshot tests for CLI commands

- Area: cmd, schema, tests
- Problem: agent integrations need stable output shapes
- Scope:
  - add golden or snapshot coverage for `parse`, `get`, `search`, `threads`, and `thread`
  - pin the fields that are intended to stay stable in `v0.1`
  - document which fields remain heuristic
- Deliverable: tests plus small spec/doc update

#### Issue: Add RFC template for contract-changing proposals

- Area: docs, governance
- Problem: schema and CLI changes need a predictable review path
- Scope:
  - add a GitHub issue template for RFC-style proposals
  - list when contributors must open an RFC instead of a normal feature request
  - point driver and schema docs to that path
- Deliverable: `.github` template and docs update

## Milestone 2: Parser Quality

Goal: improve the one area that most directly affects agent usefulness.

### Done when

- HTML body extraction is more reliable on noisy templates
- redirect-heavy tracking links are cleaned more aggressively
- fixture coverage better represents real agent workflows
- parser regressions are easier to catch before release

### Suggested GitHub issues

#### Issue: Strengthen HTML body extraction and noise filtering

- Area: parser
- Problem: some templates still leak navigation, footer, or layout noise into `body_md`
- Scope:
  - improve main-content detection before HTML-to-Markdown conversion
  - keep useful structures such as links, headings, tables, and key images
  - add fixtures for newsletter, alert, and transactional layouts that currently degrade
- Deliverable: parser changes with golden tests

#### Issue: Improve URL normalization for agent-facing actions

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

## Milestone 3: Local Memory And Threads

Goal: make local agent retrieval feel dependable instead of experimental.

### Done when

- sync behavior is easier to reason about across reruns
- cache/index state is more visible to users and contributors
- thread metadata reduces unnecessary full-message loads

### Suggested GitHub issues

#### Issue: Surface local index and sync state more clearly

- Area: cmd, internal/index, docs
- Problem: agents and contributors need more visibility into what is cached locally
- Scope:
  - expose basic sync/index stats through CLI output
  - document what `sync` skips, overwrites, or refreshes
  - make local retrieval semantics easier to explain
- Deliverable: command output improvements plus docs

#### Issue: Enrich thread summaries for triage loops

- Area: internal/index, cmd, schema
- Problem: agents still need too many full thread loads for common triage decisions
- Scope:
  - evaluate compact additions such as latest timestamp clarity, participant summary, or action/code counts
  - keep output small enough for agent prompts
  - snapshot-test the chosen shape
- Deliverable: schema/output adjustment with docs

## Milestone 4: Contributor Surface

Goal: lower the effort required for outside developers to make useful PRs.

### Done when

- adding a driver is easy to understand and test
- parser contributors can run targeted tests from documented entry points
- contract changes have a review process that does not depend on tribal knowledge

### Suggested GitHub issues

#### Issue: Add a fake-driver test harness for extension contributors

- Area: driver, tests, docs
- Problem: contributor drivers are harder to validate than they need to be
- Scope:
  - add reusable test helpers or fixtures for driver conformance
  - make list/fetch/send expectations explicit
  - document the minimum acceptance bar
- Deliverable: test harness plus contributor docs

#### Issue: Write a parser contributor guide

- Area: docs
- Problem: parser work is high-value, but the entry path is still implicit
- Scope:
  - explain fixture layout, golden tests, and parser design constraints
  - explain where heuristic behavior is acceptable and where it is not
  - link to the most relevant parser packages and tests
- Deliverable: new contributor doc

## Milestone 5: Provider Expansion

Goal: grow the ecosystem without destabilizing the shared model.

This milestone should start only after the previous milestones are in reasonable shape.

### Suggested GitHub issues

#### Issue: Add one more built-in provider with full docs and tests

- Area: driver, docs
- Problem: the ecosystem story gets stronger once there is more than one real integration path
- Scope:
  - implement one additional provider or provider style
  - keep transport isolated from parser and composer logic
  - document config, limits, and test strategy
- Deliverable: new driver, tests, docs

#### Issue: Define a driver compliance checklist

- Area: docs, tests
- Problem: community drivers need a shared quality bar
- Scope:
  - define required behaviors for list, fetch, send, and config validation
  - map those expectations to tests and contributor guidance
  - keep the checklist stable enough to reference in PR review
- Deliverable: spec or contributing doc update

## Recommended Milestones In GitHub

- `v0.1 hardening`
- `parser quality`
- `local memory`
- `contributor surface`
- `provider expansion`

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
- runtime plugin loading
- provider-specific business policy in shared layers
- trying to solve every mailbox vendor at once

The strongest next move is to make the current agent boundary sharper, not broader.
