# Local Mail Center Web Panel Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first local-only `mailcli web` implementation for account status, unified indexed inbox, message/thread reading, operation queue, and static browser UI.

**Architecture:** Add a thin `internal/web` HTTP adapter over existing Go packages: config, driver, parser, local index, and operation log. Add `cmd/web.go` as the CLI entrypoint, bind to localhost only, generate a per-run token, and embed static HTML/CSS/JS assets into the Go binary.

**Tech Stack:** Go `net/http`, `httptest`, `embed`, existing MailCLI config/index/operations/driver/parser packages, static HTML/CSS/vanilla JS.

---

## Chunk 1: Local Web Server Skeleton

### Task 1: Server Options, Token Guard, and Static UI

**Files:**
- Create: `internal/web/server.go`
- Create: `internal/web/server_test.go`
- Create: `web/index.html`
- Create: `web/styles.css`
- Create: `web/app.js`

- [ ] Write failing tests for token rejection, token acceptance, `/api/session`, and static index rendering.
- [ ] Implement `internal/web.Server` with localhost-only config, token middleware, JSON envelopes, and embedded static assets.
- [ ] Verify with `go test ./internal/web`.

### Task 2: CLI Entry Point

**Files:**
- Create: `cmd/web.go`
- Create: `cmd/web_test.go`
- Modify: `cmd/root.go`

- [ ] Write failing tests that `mailcli web --host 0.0.0.0` is rejected and `mailcli web --no-open --port 0` prints a localhost URL.
- [ ] Add `newWebCmd()` and register it in `NewRootCmd`.
- [ ] Verify with `go test ./cmd -run Web`.

## Chunk 2: Local Mail Data APIs

### Task 3: Account Status and Secret Redaction

**Files:**
- Modify: `internal/web/server.go`
- Modify: `internal/web/server_test.go`

- [ ] Write failing tests for `/api/accounts` redacting password and SMTP password values.
- [ ] Implement account listing from raw config, with public account fields and no secret values.
- [ ] Verify with `go test ./internal/web -run Accounts`.

### Task 4: Sync, Messages, Threads, and Operations

**Files:**
- Modify: `internal/web/server.go`
- Modify: `internal/web/server_test.go`

- [ ] Write failing tests using the `dir` driver and fixture config for `/api/sync`, `/api/messages`, `/api/messages/{account}/{id}`, `/api/threads`, `/api/threads/{thread_id}`, and `/api/operations`.
- [ ] Implement sync through existing driver/parser/index packages.
- [ ] Implement local index read routes and operation log read routes.
- [ ] Verify with `go test ./internal/web`.

## Chunk 3: Browser Shell and Verification

### Task 5: Usable Local Mail Center UI

**Files:**
- Modify: `web/index.html`
- Modify: `web/styles.css`
- Modify: `web/app.js`

- [ ] Build a dense local-operator UI with accounts, sync, unified inbox, message/thread detail, and operations sections.
- [ ] Ensure browser rendering uses text-safe DOM writes and avoids raw HTML injection.
- [ ] Verify static assets render through `go test ./internal/web`.

### Task 6: Docs and Final Checks

**Files:**
- Modify: `README.zh-CN.md`
- Modify: `README.md`

- [ ] Document `mailcli web` as local-only and experimental.
- [ ] Run `go test ./...`.
- [ ] Run `go build -o /tmp/mailcli ./cmd/mailcli`.
- [ ] Smoke run `/tmp/mailcli web --config examples/config/fixtures-dir.yaml --index /tmp/mailcli-web.db --no-open`.
- [ ] Commit the implementation.
