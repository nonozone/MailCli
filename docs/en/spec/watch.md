[中文](../../zh-CN/spec/watch.md) | English

# Watch Spec

## Purpose

`mailcli watch` turns a mailbox into a streaming JSONL event feed.
It is the interface between real-time email delivery and AI agent workflows.

Rather than polling the CLI repeatedly, an agent starts one `watch` process and
subscribes to new messages as they arrive — using one line per event.

## Usage

```bash
mailcli watch [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | — | Config file path |
| `--account` | — | Account name override |
| `--mailbox` | account default | Mailbox to watch (repeatable for multi-mailbox) |
| `--poll` | `30s` | Polling interval when IMAP IDLE is unavailable |
| `--since` | — | Only emit events for messages on or after this RFC3339 timestamp |
| `--auto-sync` | `false` | Also write new messages into the local index (requires `--index`) |
| `--index` | — | Local index path (enables persistent watch state, required for `--auto-sync`) |
| `--heartbeat` | `0` (disabled) | Emit `heartbeat` events at this interval (e.g. `5m`) |

## Event Schema

Every line of stdout is a complete JSON object.

### `watching`

Emitted once per mailbox on startup.

```json
{
  "event": "watching",
  "account": "work",
  "mailbox": "INBOX",
  "ts": "2026-03-31T00:00:00Z"
}
```

### `new_message`

Emitted whenever a new message arrives. Includes the full parsed `StandardMessage`.

```json
{
  "event": "new_message",
  "account": "work",
  "mailbox": "INBOX",
  "id": "<msg-id@example.com>",
  "message": { "...": "StandardMessage" },
  "ts": "2026-03-31T00:01:00Z"
}
```

### `heartbeat`

Emitted at the configured `--heartbeat` interval. Useful for keepalive detection.

```json
{
  "event": "heartbeat",
  "account": "work",
  "mailbox": "INBOX",
  "ts": "2026-03-31T00:05:00Z"
}
```

### `error`

Emitted for non-fatal errors (fetch failures, parse failures, connection errors).
The stream continues after `error` events.

```json
{
  "event": "error",
  "account": "work",
  "mailbox": "INBOX",
  "id": "<msg-id@example.com>",
  "error": "fetch: dial tcp: connection refused",
  "ts": "2026-03-31T00:01:00Z"
}
```

## Transport Strategy

MailCLI selects the most efficient transport strategy per driver:

| Driver | Strategy |
|--------|----------|
| `imap` | **IMAP IDLE** (push) when the server supports it |
| `imap` (no IDLE) | Polling at `--poll` interval |
| `dir` | Polling at `--poll` interval |
| `stub` | Polling at `--poll` interval |

IMAP IDLE holds a persistent connection and receives server-pushed notifications.
This is far lower latency than polling and generates no unnecessary traffic.

## Persistent Watch State

When `--index` is provided, `mailcli watch` maintains a persistent **seen set**
inside the SQLite index database (`watch_seen` table).

```
watch_seen(account, mailbox, msg_id, seen_at)
```

**Purpose**: Prevents re-emitting messages that have already been processed,
even after the process restarts.

**Behavior**:

1. **Startup**: The seen set is loaded from SQLite. Any IDs already in the table
   are skipped silently.
2. **First run (empty seen set)**: The current mailbox contents are seeded into
   the seen set without emitting events. This prevents an old-mail flood on the
   first invocation.
3. **New message**: The message ID is persisted to `watch_seen` before the
   `new_message` event is emitted. If the process crashes immediately after
   writing the row but before emitting the event, the message will not be
   re-emitted — this is the conservative choice for agent workflows.

**Without `--index`**: The seen set is memory-only and is lost on restart.

## Combined with --auto-sync

When both `--index` and `--auto-sync` are provided, `mailcli watch`:

1. Receives a new message ID
2. Fetches the raw message
3. Parses it into `StandardMessage`
4. Writes it to the local SQLite index via `Upsert`
5. Records the message ID in `watch_seen`
6. Emits the `new_message` event to stdout

This gives the agent both a real-time event stream and a persistent, searchable
local index — without running a separate `mailcli sync` process.

```bash
# Monitor INBOX, write new messages to index, stream events to agent:
mailcli watch \
  --account work \
  --index ~/.config/mailcli/index.db \
  --auto-sync \
  | python3 my_agent.py
```

## Agent Integration Pattern

The simplest agent pattern:

```bash
mailcli watch --account work --index ~/.config/mailcli/index.db \
  | while IFS= read -r line; do
      echo "$line" | python3 handle_event.py
    done
```

Or pipe directly to a long-running agent process:

```bash
mailcli watch --account work --index ~/.config/mailcli/index.db \
  | python3 tools/agent_example.py
```

The agent reads from stdin line by line. Each line is a complete JSON event.
The agent can filter by `event` field and act only on `new_message`.

See [tools/agent_example.py](../../../tools/agent_example.py) for a full
reference implementation.

## Multi-Mailbox

Watch multiple mailboxes simultaneously:

```bash
mailcli watch \
  --account work \
  --mailbox INBOX \
  --mailbox "Sent" \
  --mailbox "Support"
```

Each mailbox runs in its own goroutine. Events from all mailboxes are
multiplexed into a single stdout stream. The `mailbox` field in each event
identifies the source.

## Graceful Shutdown

Send `SIGINT` (Ctrl+C) or `SIGTERM` to stop the watch process cleanly.
No partial events are emitted during shutdown.

## Non-Goals

- `watch` does not produce a reply or take action on messages — that is the
  agent's responsibility.
- `watch` does not guarantee exactly-once delivery within a single process
  run. IMAP IDLE may deliver the same UID twice during reconnection. The
  persistent seen set prevents re-emission across restarts, but not within
  a single session reconnect window.
- `watch` does not expose IMAP IDLE reconnection logic as a separate event type.
  The `error` + `reconnecting` event pattern is handled internally.
