# Parser Body Extraction Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve parser output quality by extracting the primary HTML content region before Markdown conversion, reducing navigation and footer noise in agent-facing text.

**Architecture:** Keep the parser pipeline unchanged from the outside. Tighten `cleanHTML` so it removes more non-content elements and prefers semantic content roots such as `main`, `article`, or `[role="main"]`, while preserving links, images, tables, and headings inside the chosen content subtree.

**Tech Stack:** Go, `goquery`, `html-to-markdown`, standard library testing

---

## Target File Layout

### Files to modify

- `pkg/parser/html.go`
- `pkg/parser/parser_test.go`

## Chunk 1: Body Extraction Tests

### Task 1: Add failing unit tests for noisy HTML cleanup

**Files:**
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write a failing test for semantic main-content extraction**

Build an HTML email fixture inline with `header`, `nav`, `main`, and `footer`. Verify the cleaned/converted Markdown keeps only the `main` content.

- [ ] **Step 2: Write a failing test for article fallback**

Verify `article` is treated as the primary content root when `main` is absent.

- [ ] **Step 3: Write a failing test for body fallback**

Verify the parser still works when no semantic root exists and falls back to the body.

- [ ] **Step 4: Run the focused parser test command**

Run:

```bash
go test ./pkg/parser -run 'TestCleanHTML' -v
```

Expected: FAIL because the parser does not yet prefer a primary content root.

## Chunk 2: Minimal Implementation

### Task 2: Implement primary-content extraction

**Files:**
- Modify: `pkg/parser/html.go`

- [ ] **Step 1: Expand the non-content removal list**

Remove `style`, `script`, `svg`, `textarea`, `noscript`, `template`, and common layout noise such as `nav`, `footer`, and `form`.

- [ ] **Step 2: Add a primary-root selector**

Prefer, in order:
- `main`
- `article`
- `[role="main"]`

Return the first non-empty match as the HTML fragment to convert.

- [ ] **Step 3: Keep existing attribute filtering**

Continue preserving only `href`, `src`, `alt`, and `title` on surviving nodes.

- [ ] **Step 4: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestCleanHTML|TestParsePrefersHTMLOverPlainText|TestParseCleansHTMLAndConvertsToMarkdown' -v
```

Expected: PASS.

## Chunk 3: Full Verification

### Task 3: Verify parser and repository behavior

**Files:**
- Test: `pkg/parser/parser_test.go`

- [ ] **Step 1: Run the full parser package**

Run:

```bash
go test ./pkg/parser -v
```

Expected: PASS.

- [ ] **Step 2: Run the full test suite**

Run:

```bash
GOPROXY=https://goproxy.cn,direct go test ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

Run:

```bash
git add pkg/parser/html.go pkg/parser/parser_test.go docs/superpowers/plans/2026-03-26-parser-body-extraction.md
git commit -m "feat: improve parser body extraction"
```

- [ ] **Step 4: Push**

Run:

```bash
git push origin main
```
