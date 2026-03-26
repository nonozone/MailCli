[中文](../../zh-CN/spec/local-index.md) | English

# Local Index Spec

## Purpose

MailCLI provides a local file-backed index so agents can search recently synced mail without hitting the remote mailbox for every retrieval step.

This is the current baseline for local retrieval loops. It is intentionally simple and may later be replaced by a stronger storage backend.

## Commands

Current commands:

- `mailcli sync`
- `mailcli search`

Current search modes:

- compact search results by default
- full indexed message output with `mailcli search --full`

## Storage Model

Current implementation uses a JSON file on local disk.

Default path:

```text
$CACHE_DIR/mailcli/index.json
```

If the platform cache directory is unavailable, MailCLI falls back to:

```text
.mailcli-index.json
```

## Indexed Shape

Each indexed record currently stores:

- account name
- mailbox name
- driver-facing message id
- indexed timestamp
- parsed `StandardMessage`

This keeps the retrieval boundary simple:

- transport remains in drivers
- structure remains in `StandardMessage`
- search remains local

## Search Semantics

Current local search is case-insensitive substring matching over a combined text surface including:

- account
- mailbox
- message id
- subject
- snippet
- body markdown
- category
- labels
- normalized sender

This is not full-text search yet. It is a deterministic baseline intended for agent workflows and future storage evolution.

`mailcli search` also supports local filtering by:

- account
- mailbox

When `--full` is used, the command returns full indexed records rather than the compact summary shape.

## Sync Semantics

Current `mailcli sync` behavior is incremental by default:

- list recent messages from the selected account
- skip any message whose `(account, id)` pair already exists in the local index
- fetch, parse, and upsert only missing messages

Use `--refresh` to force a re-fetch and overwrite existing indexed records.

## Non-Goals For This Stage

- no SQLite requirement yet
- no FTS requirement yet
- no incremental remote change tracking yet
- no partial-sync conflict handling yet

## Why This Exists

This layer gives MailCLI a practical agent retrieval loop today:

1. sync recent mail once
2. search locally many times
3. fetch full messages again only when needed
