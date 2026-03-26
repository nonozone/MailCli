# Parser Action Classification Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract a richer set of agent-usable actions from HTML email links without changing the public schema shape.

**Architecture:** Keep `schema.Action` unchanged. Extend parser action extraction so HTML anchors are classified by visible label text and href patterns, producing a small stable set of action types in addition to `unsubscribe`.

**Tech Stack:** Go, `goquery`, standard library testing

**Status:** Implemented on 2026-03-26 in the current repository state. The checklist below is preserved as the execution record for the feature.

---

## Target File Layout

### Files to modify

- `pkg/parser/actions.go`
- `pkg/parser/parser_test.go`

## Chunk 1: Failing Tests

### Task 1: Add parser tests for richer action extraction

**Files:**
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write a failing test for `view_online` extraction**

Use HTML with anchor text like `View online` and verify the parser emits an action with:
- `type: "view_online"`
- preserved label
- cleaned URL

- [ ] **Step 2: Write a failing test for `confirm_subscription` extraction**

Use HTML with anchor text like `Confirm subscription` and verify the parser emits a `confirm_subscription` action.

- [ ] **Step 3: Write a failing test for `report_abuse` extraction**

Use HTML with anchor text like `Report abuse` or a `mailto:` abuse link and verify the parser emits a `report_abuse` action.

- [ ] **Step 4: Write a failing dedupe test**

Verify repeated HTML links do not create duplicate actions when the same URL and action type appear more than once.

- [ ] **Step 5: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestExtractActions' -v
```

Expected: FAIL because the parser currently only extracts `unsubscribe`.

## Chunk 2: Minimal Implementation

### Task 2: Add anchor-based classification

**Files:**
- Modify: `pkg/parser/actions.go`

- [ ] **Step 1: Parse HTML anchors directly**

Inspect:
- anchor text
- `href`

instead of only regex-scanning raw HTML strings.

- [ ] **Step 2: Classify a minimal action set**

Support:
- `unsubscribe`
- `view_online`
- `confirm_subscription`
- `report_abuse`

using conservative label and href matching.

- [ ] **Step 3: Preserve existing unsubscribe header support**

Keep `List-Unsubscribe` extraction unchanged except for existing URL cleaning.

- [ ] **Step 4: Deduplicate by action type plus URL**

Avoid duplicate actions when the same action appears multiple times.

- [ ] **Step 5: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestExtractActions|TestParseExtractsUnsubscribeAction' -v
```

Expected: PASS.

## Chunk 3: Verification

### Task 3: Verify parser behavior

**Files:**
- Test: `pkg/parser/parser_test.go`

- [ ] **Step 1: Run full parser tests**

Run:

```bash
go test ./pkg/parser -v
```

Expected: PASS.

- [ ] **Step 2: Run full repository tests**

Run:

```bash
GOPROXY=https://goproxy.cn,direct go test ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

Run:

```bash
git add pkg/parser/actions.go pkg/parser/parser_test.go docs/superpowers/plans/2026-03-26-parser-action-classification.md
git commit -m "feat: classify parser actions"
```

- [ ] **Step 4: Push**

Run:

```bash
git push origin main
```
