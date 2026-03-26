[中文](../../zh-CN/spec/agent-decisions.md) | English

# Agent Decision Spec

## Purpose

The agent example uses a small decision vocabulary so MailCLI-facing integrations can exchange stable, machine-usable intent instead of free-form status strings.

## Stable Decision Set

Current stable decisions:

- `review`
- `capture_code`
- `draft_reply`
- `escalate_delivery_error`

## Meaning

### `review`

The message should be inspected or classified further, but no immediate automated action is required.

### `capture_code`

The message contains a verification code or similar high-value code artifact that the agent should surface immediately.

### `draft_reply`

The provider wants the example workflow to prepare a reply draft and continue into `mailcli reply --dry-run`.

### `escalate_delivery_error`

The message represents a delivery failure or operational error that should be surfaced to a human or higher-level workflow.

## Why The Set Is Small

The goal is stability, not completeness.

A small vocabulary is easier for:

- example scripts
- external providers
- tests
- future SDK adapters

to implement consistently.

## Extension Guidance

If a new decision is needed:

1. add it to this spec
2. update the example runtime validation
3. add or update integration tests
4. document the new behavior in example docs

Do not introduce ad-hoc decision strings in provider implementations without updating the shared contract.
