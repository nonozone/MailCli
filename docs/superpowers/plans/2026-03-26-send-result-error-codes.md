# Send Result Error Codes Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Return stable machine-readable `SendResult.error.code` values for `mailcli send` and `mailcli reply` failures.

**Architecture:** Keep the `Driver` interface unchanged and avoid broad typed-error refactors for now. Add a small command-layer error mapper that converts common config, lookup, auth, and transport failures into stable `SendError.code` strings, then emit `SendResult{ok:false}` JSON instead of raw CLI errors for non-dry-run outbound operations.

**Tech Stack:** Go, Cobra, standard library testing

**Status:** Implemented on 2026-03-26 in the current repository state. The checklist below is preserved as the execution record for the feature.

---

## Target File Layout

### Files to modify

- `cmd/send.go`
- `cmd/send_test.go`
- `docs/en/spec/outbound-message.md`
- `docs/zh-CN/spec/outbound-message.md`

## Chunk 1: Failing Tests For Structured Failures

### Task 1: Add failing tests for `send`

**Files:**
- Modify: `cmd/send_test.go`

- [ ] **Step 1: Write a failing test for auth failure mapping**

Verify `mailcli send` returns JSON with:
- `ok: false`
- `error.code: "auth_failed"`

when the configured driver returns an SMTP auth-style error.

- [ ] **Step 2: Write a failing test for account resolution failure mapping**

Verify `mailcli send` returns JSON with:
- `ok: false`
- `error.code: "account_not_found"`

when the requested account does not exist.

- [ ] **Step 3: Run the focused command tests**

Run:

```bash
go test ./cmd -run 'TestSendCommand' -v
```

Expected: FAIL because failures still bubble out as raw errors.

### Task 2: Add failing tests for `reply`

**Files:**
- Modify: `cmd/send_test.go`

- [ ] **Step 1: Write a failing test for message lookup failure mapping**

Verify `mailcli reply` returns JSON with:
- `ok: false`
- `error.code: "message_not_found"`

when `reply_to_id` fetch cannot find the referenced message.

- [ ] **Step 2: Write a failing test for transport-not-configured mapping**

Verify `mailcli reply` or `mailcli send` returns JSON with:
- `ok: false`
- `error.code: "transport_not_configured"`

when SMTP settings are missing for the selected account.

- [ ] **Step 3: Run the focused command tests**

Run:

```bash
go test ./cmd -run 'TestReplyCommand|TestSendCommand' -v
```

Expected: FAIL because failures are not yet converted into `SendResult`.

## Chunk 2: Minimal Implementation

### Task 3: Implement command-layer error mapping

**Files:**
- Modify: `cmd/send.go`

- [ ] **Step 1: Add a helper that builds failure `SendResult` values**

Include:
- `ok: false`
- best-effort `provider`
- best-effort `account`
- stable `error.code`
- original error text in `error.message`

- [ ] **Step 2: Add a small error classifier**

Map at least:
- `auth_failed`
- `account_not_found`
- `account_not_selected`
- `message_not_found`
- `transport_not_configured`
- `invalid_draft`
- `transport_failed` as fallback

- [ ] **Step 3: Use the helper in non-dry-run send/reply paths**

Convert resolved failures into JSON results instead of returning raw errors.

- [ ] **Step 4: Keep dry-run behavior unchanged**

Dry-run should still behave like a developer-facing validation path.

- [ ] **Step 5: Run focused command tests**

Run:

```bash
go test ./cmd -run 'TestSendCommand|TestReplyCommand' -v
```

Expected: PASS.

## Chunk 3: Verification And Docs

### Task 4: Verify and document stable result codes

**Files:**
- Modify: `docs/en/spec/outbound-message.md`
- Modify: `docs/zh-CN/spec/outbound-message.md`

- [ ] **Step 1: Document the current behavior**

Clarify that non-dry-run `send` and `reply` return `SendResult` JSON on both success and operational failure.

- [ ] **Step 2: Run full test suite**

Run:

```bash
GOPROXY=https://goproxy.cn,direct go test ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

Run:

```bash
git add cmd/send.go cmd/send_test.go docs/en/spec/outbound-message.md docs/zh-CN/spec/outbound-message.md docs/superpowers/plans/2026-03-26-send-result-error-codes.md
git commit -m "feat: add stable send result error codes"
```

- [ ] **Step 4: Push**

Run:

```bash
git push origin main
```
