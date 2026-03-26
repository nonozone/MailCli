# IMAP FetchRaw Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement real IMAP raw-message fetching so `mailcli get` and `mailcli reply` can resolve messages from configured IMAP accounts.

**Architecture:** Keep the public driver interface unchanged. Add a small internal IMAP session abstraction so `imapDriver.FetchRaw` can be tested without a live server. Resolve CLI ids into one of three fetch targets: explicit UID, sequence number fallback, or `Message-Id` header search, then fetch `BODY.PEEK[]` bytes and return them unchanged.

**Tech Stack:** Go, `go-imap`, standard library testing

**Status:** Implemented on 2026-03-26 in the current repository state. The checklist below is preserved as the execution record for the feature.

---

## Target File Layout

### Files to modify

- `pkg/driver/imap.go`

### Files to create

- `pkg/driver/imap_fetch_test.go`

## Chunk 1: Testable IMAP Fetch Path

### Task 1: Add failing tests for fetch target resolution

**Files:**
- Create: `pkg/driver/imap_fetch_test.go`

- [x] **Step 1: Write a failing test for explicit UID ids**

Add a test that verifies `imap:uid:42` resolves to a UID fetch target.

- [x] **Step 1.1: Write a failing test for the short UID alias**

Add a test that verifies `uid:42` resolves to the same UID fetch target.

- [x] **Step 2: Write a failing test for numeric ids**

Add a test that verifies `42` resolves to a sequence-number fetch target.

- [x] **Step 3: Write a failing test for message-id ids**

Add a test that verifies `abc@example.com` resolves to a normalized `Message-Id` header search target using angle brackets.

- [x] **Step 4: Run the focused test command**

Run:

```bash
go test ./pkg/driver -run 'TestResolveFetchTarget' -v
```

Expected: FAIL because fetch target resolution does not exist yet.

### Task 2: Add failing tests for raw-message retrieval

**Files:**
- Create: `pkg/driver/imap_fetch_test.go`
- Modify: `pkg/driver/imap.go`

- [x] **Step 1: Write a failing test for message-id search plus body fetch**

Use a fake IMAP session that records `Select`, `Search`, and `Fetch` calls. Verify `FetchRaw`:
- selects the configured mailbox in readonly mode
- searches by `Message-Id`
- fetches `BODY.PEEK[]`
- returns the exact raw bytes

- [x] **Step 2: Write a failing test for explicit UID fetch**

Verify `FetchRaw` uses `UidFetch` when the input id is an explicit UID.

- [x] **Step 3: Write a failing test for missing search results**

Verify `FetchRaw` returns a clear error when no message matches the requested id.

- [x] **Step 4: Write a failing test for an empty fetch response**

Verify `FetchRaw` returns a clear error when IMAP fetch completes but returns no message or no body literal.

- [x] **Step 5: Run the focused test command**

Run:

```bash
go test ./pkg/driver -run 'TestIMAPDriverFetchRaw' -v
```

Expected: FAIL because `FetchRaw` is still not implemented.

## Chunk 2: Minimal Implementation

### Task 3: Implement IMAP FetchRaw with internal seams

**Files:**
- Modify: `pkg/driver/imap.go`
- Test: `pkg/driver/imap_fetch_test.go`

- [x] **Step 1: Add a small internal IMAP session interface**

Wrap only the methods needed by `List` and `FetchRaw`: `Select`, `Search`, `Fetch`, `UidFetch`, and `Logout`.

- [x] **Step 2: Add fetch target parsing helpers**

Implement helpers that:
- treat `imap:uid:<n>` and `uid:<n>` as explicit UIDs
- treat all-digit ids as sequence numbers
- treat everything else as a `Message-Id` search target and normalize angle brackets

- [x] **Step 3: Implement fetch execution**

Fetch `BODY.PEEK[]`, read the returned literal fully, and return the raw bytes. If the id resolved through search, use the first match.

- [x] **Step 4: Run focused driver tests**

Run:

```bash
go test ./pkg/driver -run 'TestResolveFetchTarget|TestIMAPDriverFetchRaw|TestIMAPDriverSendRaw|TestNewFromAccount' -v
```

Expected: PASS.

- [x] **Step 5: Refactor only if needed**

Keep list/send behavior unchanged while reusing the same session abstraction.

- [x] **Step 6: Add list regression coverage**

Add a driver-level test proving `List` still selects the mailbox in readonly mode and returns envelope-derived ids and subjects after the session seam is introduced.

## Chunk 3: Integration Verification

### Task 4: Verify command-level behavior still holds

**Files:**
- Test: `cmd/get_test.go`
- Test: `cmd/send_test.go`

- [x] **Step 1: Run focused command tests**

Run:

```bash
go test ./cmd -run 'TestGetCommandParsesFetchedMessage|TestReplyCommandResolvesReplyToIDViaFetch|TestSendCommandUsesConfiguredDriver|TestReplyCommandUsesConfiguredDriver' -v
```

Expected: PASS.

- [x] **Step 2: Run full test suite**

Run:

```bash
GOPROXY=https://goproxy.cn,direct go test ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

Run:

```bash
git add pkg/driver/imap.go pkg/driver/imap_fetch_test.go docs/superpowers/plans/2026-03-26-imap-fetch-raw.md
git commit -m "feat: implement imap raw fetch"
```

- [ ] **Step 4: Push**

Run:

```bash
git push origin main
```
