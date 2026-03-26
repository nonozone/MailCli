# Parser Attachment Actions Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract attachment-related actions from HTML links so agents can recognize obvious attachment entry points without a new inbound attachment schema.

**Architecture:** Keep `schema.Action` unchanged. Extend anchor classification with a conservative pair of action types, `view_attachment` and `download_attachment`, based on anchor text and href patterns only.

**Tech Stack:** Go, `goquery`, standard library testing

---

## Target File Layout

### Files to modify

- `pkg/parser/actions.go`
- `pkg/parser/parser_test.go`

## Chunk 1: Failing Tests

### Task 1: Add parser tests for attachment actions

**Files:**
- Modify: `pkg/parser/parser_test.go`

- [x] **Step 1: Write a failing test for `view_attachment`**

Verify anchor text like `View attachment` produces:
- `type: "view_attachment"`
- preserved label
- URL preserved

- [x] **Step 2: Write a failing test for `download_attachment`**

Verify anchor text like `Download attachment` or an href pattern containing `/download/` produces:
- `type: "download_attachment"`

- [x] **Step 3: Write a failing dedupe test**

Verify duplicate attachment links do not create duplicate actions of the same type and URL.

- [x] **Step 4: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestExtractActions' -v
```

Expected: FAIL because the parser currently does not classify attachment actions.

## Chunk 2: Minimal Implementation

### Task 2: Extend anchor classification

**Files:**
- Modify: `pkg/parser/actions.go`

- [x] **Step 1: Add conservative attachment action matching**

Support:
- `view_attachment`
- `download_attachment`

using a combination of anchor text and href patterns.

- [x] **Step 2: Preserve existing label behavior**

Keep the original anchor text as the action label when available.

- [x] **Step 3: Reuse existing dedupe behavior**

Deduplicate by action type and URL just like existing actions.

- [x] **Step 4: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestExtractActions' -v
```

Expected: PASS.

## Chunk 3: Verification

### Task 3: Verify parser behavior

**Files:**
- Test: `pkg/parser/parser_test.go`

- [x] **Step 1: Run full parser tests**

Run:

```bash
go test ./pkg/parser -v
```

Expected: PASS.

- [x] **Step 2: Run full repository tests**

Run:

```bash
GOPROXY=https://goproxy.cn,direct go test ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

Run:

```bash
git add pkg/parser/actions.go pkg/parser/parser_test.go docs/superpowers/plans/2026-03-26-parser-attachment-actions.md
git commit -m "feat: extract attachment actions"
```

- [ ] **Step 4: Push**

Run:

```bash
git push origin main
```
