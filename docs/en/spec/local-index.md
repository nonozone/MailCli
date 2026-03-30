[中文](../../zh-CN/spec/local-index.md) | English

# Local Index Spec

## Purpose

MailCLI provides a local SQLite index so agents can search recently synced mail
without hitting the remote mailbox for every retrieval step.

The index supports:

- fast full-text search via FTS5
- thread grouping and filtering
- persistent watch deduplication across process restarts

## Commands

- `mailcli sync` — fetch messages and write to the index
- `mailcli search` — full-text search the index
- `mailcli threads` — view conversation summaries
- `mailcli thread` — read a specific thread's messages
- `mailcli export` — export the index as JSON, JSONL, or CSV
- `mailcli watch --index` — stream new messages with a persistent seen set

## Storage Model

The index is a SQLite database file.

Default path:

```text
$CACHE_DIR/mailcli/index.db
```

If the platform cache directory is unavailable, MailCLI falls back to:

```text
.mailcli-index.db
```

### Backward Compatibility

Paths ending in `.json` are automatically remapped to `.db`. Existing
`--index` flag values continue to work without modification.

```bash
# Both of these open the same database:
mailcli sync --index /tmp/mailcli.json
mailcli sync --index /tmp/mailcli.db
```

### Schema

Three tables are maintained:

**`messages`** — one row per `(account, message-id)`:

- `account`, `mailbox`, `id`, `thread_id`, `date`, `indexed_at`
- `from_addr`, `subject`, `category`, `labels`, `action_types`
- `has_codes`, `code_count`, `snippet`, `body_text`, `body_json`

**`messages_fts`** — FTS5 virtual table:

- Columns: `subject`, `from_addr`, `body_text`
- Tokenizer: `unicode61 remove_diacritics 2`
- Synchronized with `messages` inside each write transaction

**`watch_seen`** — persistent watch deduplication:

- `(account, mailbox, msg_id, seen_at)`
- Used by `mailcli watch` to prevent re-emitting messages after restarts

### WAL Mode

The database runs in WAL (Write-Ahead Logging) mode. This allows concurrent
reads during writes, which is important for agents that search the index while
a separate `watch --auto-sync` process is writing to it.

## Indexed Shape

Each indexed record stores:

- account name
- mailbox name
- driver-facing message id
- `thread_id` (derived from References/In-Reply-To/Message-ID)
- indexed timestamp
- parsed `StandardMessage` (full JSON in `body_json`)
- pre-extracted search fields (subject, from, snippet, body text)

## Search Semantics

### Full-Text Search (FTS5)

`mailcli search <query>` uses FTS5 for recall:

- Each word in the query becomes a prefix term
- All terms must match (implicit AND)
- FTS5 handles unicode, diacritics, and prefix expansion automatically

After FTS5 recall, results are re-ranked by a Go-layer `scoreMatch()` function
that applies field weights:

| Field | Weight |
|-------|--------|
| Subject match | highest |
| Snippet match | high |
| Body match | medium |
| Sender match | medium |
| Action/code text | medium |

Results are ordered by score first, then by message date (most recent first).

### Filter-Only Search

When no query text is provided, `mailcli search` filters by:

- `--account`
- `--mailbox`
- `--thread`
- `--since` / `--before`

Results are ordered by date descending.

### Compact vs. Full Results

- Default: compact `SearchResult` (account, mailbox, id, thread_id, subject,
  from, date, snippet, category, labels, score)
- `--full`: full `IndexedMessage` including `StandardMessage` JSON

## Thread Semantics

`mailcli threads` groups local indexed messages into conversation summaries.

Thread grouping is deterministic:

1. First available `references` entry
2. Otherwise `in_reply_to`
3. Otherwise `message_id`
4. Otherwise the local indexed message id

Thread summaries expose:

- `thread_id`, subject, latest date, latest sender, latest preview
- Aggregated: categories, action types, labels
- `has_codes`, `code_count`, `action_count`, `participant_count`, participants
- Local message IDs, message count, score

Filtering with `mailcli threads`:

```bash
mailcli threads --has-codes
mailcli threads --category verification
mailcli threads --action verify_sign_in
```

`mailcli thread <thread_id>` returns the full indexed messages for one thread,
ordered by message date.

## Sync Semantics

`mailcli sync` behavior is incremental by default:

1. List recent messages from the configured account
2. **`BulkHas`**: check which IDs already exist in a single SQL query
3. Fetch only missing messages (or all if `--refresh`)
4. Parse all fetched messages
5. **`BulkUpsert`**: write all messages in a single SQLite transaction

Using a single transaction for the batch means only one `fsync` call regardless
of batch size. Performance scales much better than per-message commits for large
mailboxes.

`mailcli sync` output:

- `listed_count`, `fetched_count`, `indexed_count`, `skipped_count`
- `refreshed_count` (when `--refresh` is used)
- `index_path`

## Watch State Semantics

When `mailcli watch` is run with `--index`, the `watch_seen` table tracks which
messages have already been emitted as events.

See [Watch Spec](watch.md) for full details.

## Why This Exists

This layer gives MailCLI a practical agent retrieval loop:

1. `mailcli sync` — populate the index once (or continuously via `watch --auto-sync`)
2. `mailcli search` — search locally many times without remote fetches
3. `mailcli thread` — load full conversation context on demand
4. `mailcli watch` — receive new messages in real time with deduplication
