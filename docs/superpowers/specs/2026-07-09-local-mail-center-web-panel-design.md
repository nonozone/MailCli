# Local Mail Center Web Panel v1 Design

## 1. Summary

MailCLI should add a local-only temporary Web control panel for human mailbox
setup and day-to-day mail handling.

The first version is not an AI relay system and not a hosted mail service. It is
a local Mail Center that lets a user connect existing mailboxes, sync them into
one local index, read/search/thread messages, prepare replies or sends, and
confirm operations through the browser.

AI-assisted relay and automatic reply matching should come after this panel
exists, using the same indexed messages, operation intents, and confirmation
queue.

## 2. Product Decision

Add a command:

```bash
mailcli web
```

It starts a temporary HTTP server bound to `127.0.0.1` by default, generates a
per-run session token, and opens a local browser URL such as:

```text
http://127.0.0.1:49231/?token=<session-token>
```

The server process is temporary. The user's MailCLI config, local message index,
and operation log remain persistent through the existing local paths.

## 3. Requirements

### Explicit Requirements

- The Web panel is for local machine use only.
- Users can see and manage mailbox configuration from the panel.
- Users can connect multiple existing mailboxes.
- Users can see a unified inbox across configured accounts.
- Users can read messages, inspect threads, search mail, and manage replies.
- AI automatic matching should be a second step, not part of the first delivery.

### Inferred Requirements

- The panel must reuse the Go core instead of adding another user-facing runtime.
- The UI should use the same account, sync, index, search, thread, composer, and
  operation-log contracts already used by the CLI.
- Browser actions that change mailbox state must go through the same
  prepare/confirm/log model as agent workflows.
- The panel should be useful to humans while still producing machine-readable
  state for future agents.

### Constraints

- Default bind address is `127.0.0.1`; no LAN or public binding in v1.
- No hosted backend, no user account system, no cloud state.
- No Node/Python runtime requirement for users.
- No direct Web exposure of send/delete/move/mark actions without an operation
  intent and a user confirmation step.
- No AI provider inside the Web panel core in v1.

### Non-goals

- Do not build a traditional full-featured webmail replacement.
- Do not implement a Tencent-style hosted agent mailbox.
- Do not implement Cloudflare/Resend relay receiving in v1.
- Do not let AI auto-forward, auto-delete, or auto-send mail in v1.
- Do not introduce provider-specific business logic into parser or composer
  layers.

## 4. Current Baseline

The repo already has the needed backend building blocks:

- Account setup and diagnostics: `account add`, `config init`, `config show`,
  `config doctor`, `config test`, `config capabilities`
- Reading and indexing: `list`, `get`, `sync`, `search`, `threads`, `thread`
- Local index: SQLite-backed message storage with `account`, `mailbox`, message
  metadata, full `StandardMessage` JSON, and FTS search
- Outbound safety: `send prepare|confirm`, `reply prepare|confirm`
- Audit trail: `operations list|show` and JSONL operation log
- Agent integration: `agent doctor`, `agent configure`, `mcp serve`

Gaps that matter for the Web panel:

- No local HTTP server or browser UI exists today.
- `delete`, `move`, and `mark` currently execute directly; they should not be
  exposed as Web actions until they have prepare/confirm/log coverage.
- Attachments are parser-facing today; a first-class attachment list/download
  flow is not yet designed.

## 5. Approach Options

| Option | Description | Fit | Cost | Risk | Score |
| --- | --- | ---: | ---: | ---: | ---: |
| A | Go `net/http` server plus embedded static HTML/CSS/vanilla JS | 5 | 2 | 2 | 9 |
| B | Go API plus embedded React/Vite SPA | 4 | 4 | 3 | 5 |
| C | Separate desktop app or Electron/Tauri shell | 3 | 5 | 4 | 2 |

Scoring favors fit with the Go-only product direction, installation simplicity,
and low operational risk.

Recommended option: **A**.

Use a Go local HTTP server with embedded static assets and small browser-side
JavaScript that calls JSON endpoints. This keeps the installed product as one
binary and avoids making Node, Python, Electron, or a hosted service part of the
user path.

## 6. Proposed Architecture

### New Command

```bash
mailcli web \
  --config ~/.config/mailcli/config.yaml \
  --index ~/.local/state/mailcli/index.db \
  --operations ~/.local/state/mailcli/operations.jsonl \
  --host 127.0.0.1 \
  --port 0
```

Suggested flags:

- `--config`: existing MailCLI config path
- `--index`: local SQLite index path
- `--operations`: operation log path
- `--host`: defaults to `127.0.0.1`; v1 should reject non-localhost values unless
  a future explicit unsafe flag exists
- `--port`: defaults to `0` for random available port
- `--no-open`: do not open the browser automatically

### New Packages

- `cmd/web.go`
  Defines the CLI entrypoint and flags.
- `internal/web`
  Owns the local HTTP server, session token validation, route registration,
  request parsing, response envelopes, and adapter calls into existing MailCLI
  services.
- `web/`
  Embedded browser assets: HTML, CSS, and small JavaScript modules.

The Web layer should be an adapter. It should not duplicate mailbox protocol,
parser, composer, index, or operation-log behavior.

### Local Security Model

- Bind to `127.0.0.1` by default.
- Generate a random session token at startup.
- Require the token for every stateful API request.
- Set no permissive CORS headers in v1.
- Never render secret values.
- Never write raw passwords or app passwords into config files.
- Never expose write tools without the operation-intent confirmation step.

## 7. Web Panel Views

### Overview

Show local status:

- MailCLI version
- active config path
- active index path
- active operations log path
- detected accounts
- last sync summary
- agent integration status

### Accounts

Support:

- list configured accounts
- add account from provider presets: Gmail, Outlook/Microsoft 365, QQ, 163,
  generic IMAP
- show provider-specific setup guidance
- run static config diagnostics
- run connection test
- show account capabilities

Secrets should be represented as environment variable references. If a provider
requires an app password or authorization code, the UI should explain the next
step rather than storing the secret directly.

### Unified Inbox

Support:

- sync one account or all accounts
- display indexed messages across accounts
- filter by account, mailbox, category, action type, has codes, unread status if
  available, and date
- search using the existing local index
- show compact rows: account, mailbox, sender, subject, date, snippet, category,
  extracted actions/codes

### Message Reader

Support:

- read one indexed message
- display normalized body Markdown
- show sender, recipients, subject, date, Message-ID, In-Reply-To, References
- show extracted actions and codes
- show thread navigation
- show raw JSON in an optional inspector for debugging and agent developers

### Thread View

Support:

- list thread summaries
- open a thread
- show chronological messages
- use existing thread IDs and local index data

### Compose, Reply, and Forward

Support:

- compose a new draft
- reply to an existing message
- prepare a basic forward-style draft from an existing message when the user
  chooses to forward it
- prepare send/reply operation
- show generated summary, recipients, subject, body preview, and confirm command
- confirm the operation from the UI only after the user explicitly clicks confirm
- write results to the existing operation log

MailCLI does not yet have a native `ForwardDraft` contract. In v1, forwarding
should be represented as a normal `send prepare` draft that includes explicit
source-message context in the body. Native MIME forwarding, original attachment
carry-forward, and a first-class `forward prepare|confirm` command need a
separate design before being exposed as stable contracts.

### Operation Queue

Support:

- list prepared operations
- show operation detail
- confirm pending send/reply operations
- show sent/failed outcomes

Future mailbox mutations (`delete`, `move`, `mark`) should appear here only
after they have the same prepare/confirm/log contract.

## 8. API Shape

Use local JSON endpoints. Suggested first pass:

```text
GET  /api/session
GET  /api/accounts
POST /api/accounts
POST /api/accounts/{account}/test
GET  /api/accounts/{account}/capabilities

POST /api/sync
GET  /api/messages
GET  /api/messages/{account}/{id}
GET  /api/threads
GET  /api/threads/{thread_id}

POST /api/send/prepare
POST /api/reply/prepare
GET  /api/operations
GET  /api/operations/{id}
POST /api/operations/{intent_id}/confirm
```

Responses should use stable envelopes:

```json
{
  "ok": true,
  "data": {}
}
```

Errors should include stable machine-readable codes:

```json
{
  "ok": false,
  "error": {
    "code": "config_env_unset",
    "message": "MAILCLI_GMAIL_APP_PASSWORD is not set",
    "next_step": "Set the environment variable, then run connection test again."
  }
}
```

The Web API is local and private in v1, but it should still be stable enough for
browser code and future agent workflows.

## 9. Phase 2: AI Relay and Reply Matching

AI relay should be built after v1, not inside v1.

Phase 2 can add:

- relay candidate detection
- thread/context matching with evidence
- generated forward/reply drafts
- explanation of why a message matched a relay target
- confidence and risk signals
- user-confirmed execution through the same operation queue

The AI layer should not directly mutate mailboxes. It should produce candidate
actions that a user can inspect and confirm in the Web panel.

## 10. Acceptance Criteria

1. `mailcli web` starts a local server bound to `127.0.0.1` and prints the local
   URL.
2. The command can choose a random port when `--port 0` is used.
3. A per-run token is required for API requests that read private mailbox data
   or perform state changes.
4. The Web panel lists configured accounts without exposing secret values.
5. The Web panel can run config diagnostics and connection tests.
6. The Web panel can trigger sync into the local index.
7. The Web panel can show a unified indexed inbox across accounts.
8. The Web panel can open a message and a thread using existing index data.
9. The Web panel can prepare and confirm send/reply operations through existing
   operation-log contracts.
10. Delete/move/mark are not exposed as direct Web actions until they have
    prepare/confirm/log support.
11. The installed user path remains one Go binary with no Node/Python runtime
    dependency.
12. Tests cover server startup, token enforcement, account listing, sync route
    wiring, message route wiring, operation confirmation, and secret redaction.

## 11. Testing Plan

- Unit tests for route handlers using `httptest`.
- Unit tests for token rejection and accepted token paths.
- Unit tests for response envelopes and stable error codes.
- Tests proving account responses redact secret values.
- Tests using the existing `dir` driver and fixture config for zero-network
  inbox, message, and thread views.
- Tests for send/reply prepare-confirm route wiring using existing fake drivers.
- Snapshot or golden tests for key JSON API responses.
- Manual smoke test:

```bash
go build -o /tmp/mailcli ./cmd/mailcli
/tmp/mailcli web --config examples/config/fixtures-dir.yaml --index /tmp/mailcli-web.db --no-open
```

## 12. Implementation Slices

1. Add `mailcli web` skeleton, localhost binding, random port, token, and static
   embedded shell.
2. Add account/status APIs and a basic Accounts view.
3. Add sync/search/messages/thread APIs and unified inbox views using the local
   index.
4. Add message reader and thread view.
5. Add send/reply prepare-confirm routes and operation queue.
6. Add docs, smoke commands, tests, and release notes.

## 13. Open Decisions

- Exact default index path for Web mode should align with the existing MailCLI
  state path convention before implementation.
- Whether the first UI should include mailbox mutations depends on completing
  prepare/confirm/log for `delete`, `move`, and `mark`.
- Whether native forwarding belongs in the first implementation depends on a
  separate `ForwardDraft` / `forward prepare|confirm` contract. The Web panel
  can still offer basic forward-style drafts through `send prepare`.
- Attachment download needs a separate design if the UI is expected to save
  remote provider attachments, not just display parsed attachment-like actions.

## 14. Highest-risk Assumptions

- The current index APIs may not yet expose all pagination/filtering needed for
  a pleasant inbox UI.
- Existing CLI command functions may need service-level extraction so the Web
  layer can reuse behavior without shelling out to itself.
- Localhost token protection is necessary but not sufficient if future versions
  add LAN or remote access; v1 must keep that out of scope.

## 15. Done Signal for Implementation

An implementation of this spec should report:

- changed packages and files
- exact `go test` commands run
- manual `mailcli web` smoke result
- confirmation that no secrets are rendered
- confirmation that all state-changing browser actions use operation intents
- any deviations from this design and why they were necessary
