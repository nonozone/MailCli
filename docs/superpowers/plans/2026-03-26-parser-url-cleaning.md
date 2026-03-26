# Parser URL Cleaning Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clean common redirect-style tracking links so parser output exposes more meaningful target URLs to agents.

**Architecture:** Keep the parser structure intact. Add a small URL normalization helper that recognizes safe, explicit target parameters in redirect-style links, then use it in action extraction and HTML attribute cleanup so Markdown output and extracted actions both prefer the cleaned target URL.

**Tech Stack:** Go, `goquery`, standard library testing

**Status:** Implemented on 2026-03-26 in the current repository state. The checklist below is preserved as the execution record for the feature.

---

## Target File Layout

### Files to modify

- `pkg/parser/actions.go`
- `pkg/parser/html.go`
- `pkg/parser/parser_test.go`

## Chunk 1: Failing Tests

### Task 1: Add parser tests for URL cleaning

**Files:**
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write a failing test for redirect-style HTML link cleaning**

Use HTML containing a tracked URL like:

```text
https://tracker.example.com/click?redirect=https%3A%2F%2Fapp.example.com%2Freport
```

Verify the Markdown output prefers `https://app.example.com/report`.

- [ ] **Step 2: Write a failing test for unsubscribe action cleaning**

Verify unsubscribe actions use the cleaned target URL when the HTML link or header contains a redirect-style wrapper.

- [ ] **Step 3: Write a failing test for non-redirect URLs**

Verify ordinary URLs remain unchanged.

- [ ] **Step 4: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestParseCleansTrackedURLs|TestExtractActions' -v
```

Expected: FAIL because URLs are currently passed through unchanged.

## Chunk 2: Minimal Implementation

### Task 2: Add a small URL cleaning helper

**Files:**
- Modify: `pkg/parser/actions.go`
- Modify: `pkg/parser/html.go`

- [ ] **Step 1: Implement redirect-parameter extraction**

Support explicit query parameters:
- `url`
- `target`
- `redirect`
- `redirect_url`
- `redirect_uri`
- `dest`
- `destination`

Only rewrite when the extracted value is an absolute `http` or `https` URL.

- [ ] **Step 2: Use the helper in action extraction**

Normalize:
- `List-Unsubscribe` URLs
- HTML-extracted unsubscribe URLs

- [ ] **Step 3: Use the helper in HTML attribute filtering**

When preserving `href` and `src`, replace redirect-style wrappers with the cleaned target URL if one is available.

- [ ] **Step 4: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestParseCleansTrackedURLs|TestExtractActions|TestParseCleansHTMLAndConvertsToMarkdown' -v
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
git add pkg/parser/actions.go pkg/parser/html.go pkg/parser/parser_test.go docs/superpowers/plans/2026-03-26-parser-url-cleaning.md
git commit -m "feat: clean tracked parser urls"
```

- [ ] **Step 4: Push**

Run:

```bash
git push origin main
```
