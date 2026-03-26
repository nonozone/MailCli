# Parser Invoice Actions Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract invoice-related actions from HTML links so agents can recognize obvious billing entry points without adding a new invoice schema.

**Architecture:** Keep `schema.Action` unchanged. Extend anchor classification with a conservative pair of action types, `view_invoice` and `pay_invoice`, using visible label text and narrow href patterns only.

**Tech Stack:** Go, `goquery`, standard library testing

---

## Target File Layout

### Files to modify

- `pkg/parser/actions.go`
- `pkg/parser/parser_test.go`
- `README.md`
- `README.zh-CN.md`
- `docs/en/agent-workflows.md`
- `docs/zh-CN/agent-workflows.md`

## Chunk 1: Failing Tests

### Task 1: Add parser tests for invoice actions

**Files:**
- Modify: `pkg/parser/parser_test.go`

- [x] **Step 1: Write a failing test for `pay_invoice`**

Verify anchor text like `Pay invoice` produces:
- `type: "pay_invoice"`
- preserved label
- URL preserved

- [x] **Step 2: Write a failing test for `view_invoice`**

Verify an obvious invoice href like `/invoices/123` can produce:
- `type: "view_invoice"`

- [x] **Step 3: Write a failing dedupe test**

Verify repeated invoice links do not create duplicate actions of the same type and URL.

- [x] **Step 4: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestExtractActions(ClassifiesPayInvoiceLink|ClassifiesViewInvoiceFromHrefPattern|DeduplicatesInvoiceActions)' -v
```

Expected: FAIL because the parser currently does not classify invoice actions.

## Chunk 2: Minimal Implementation

### Task 2: Extend anchor classification

**Files:**
- Modify: `pkg/parser/actions.go`

- [x] **Step 1: Add conservative invoice action matching**

Support:
- `pay_invoice`
- `view_invoice`

using a combination of anchor text and href patterns.

- [x] **Step 2: Preserve existing label behavior**

Keep the original anchor text as the action label when available.

- [x] **Step 3: Reuse existing dedupe behavior**

Deduplicate by action type and URL just like existing actions.

- [x] **Step 4: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestExtractActions(ClassifiesPayInvoiceLink|ClassifiesViewInvoiceFromHrefPattern|DeduplicatesInvoiceActions)' -v
```

Expected: PASS.

## Chunk 3: Documentation And Verification

### Task 3: Align docs and verify repository state

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/en/agent-workflows.md`
- Modify: `docs/zh-CN/agent-workflows.md`

- [x] **Step 1: Update docs to mention invoice actions**

Reflect the expanded action vocabulary in the main project docs and agent workflow docs.

- [x] **Step 2: Run full repository tests**

Run:

```bash
GOPROXY=https://goproxy.cn,direct go test ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

Run:

```bash
git add pkg/parser/actions.go pkg/parser/parser_test.go README.md README.zh-CN.md docs/en/agent-workflows.md docs/zh-CN/agent-workflows.md docs/superpowers/plans/2026-03-26-parser-invoice-actions.md
git commit -m "feat: extract invoice actions"
```

- [ ] **Step 4: Push**

Run:

```bash
git push origin main
```
