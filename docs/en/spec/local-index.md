[中文](../../zh-CN/spec/local-index.md) | English

# Local Index Spec

## Purpose

MailCLI provides a local file-backed index so agents can search recently synced mail without hitting the remote mailbox for every retrieval step.

This is the current baseline for local retrieval loops. It is intentionally simple and may later be replaced by a stronger storage backend.

## Commands

Current commands:

- `mailcli sync`
- `mailcli search`
- `mailcli threads`
- `mailcli thread`

Current search modes:

- compact search results by default
- full indexed message output with `mailcli search --full`

Current thread mode:

- conversation summaries with `mailcli threads`

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

Current compact search results also include a deterministic `score` field.

Ranking is currently weighted toward:

- subject matches
- snippet matches
- body markdown matches
- sender matches

Results are ordered by score first, then by recency of local indexing.

`mailcli search` also supports local filtering by:

- account
- mailbox
- thread id

When `--full` is used, the command returns full indexed records rather than the compact summary shape.

## Thread Semantics

`mailcli threads` groups local indexed messages into conversation summaries.

Current thread grouping is deterministic and based on:

- first available `references` entry
- otherwise `in_reply_to`
- otherwise `message_id`
- otherwise the local indexed message id

Thread summaries currently expose:

- `thread_id`
- subject
- latest date
- message count
- participants
- local message ids
- score

`mailcli thread <thread_id>` returns the full indexed messages for a selected local thread, ordered by message date.

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
