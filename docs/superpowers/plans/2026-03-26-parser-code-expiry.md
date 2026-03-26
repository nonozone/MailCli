# Parser Code Expiry Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend inbound verification-code extraction with machine-usable expiry metadata.

**Architecture:** Keep the existing `codes` schema and add one narrow optional field, `expires_in_seconds`. Extract relative expiry durations from nearby text using conservative English and Chinese patterns so agents can combine the value with `meta.date` if they need an absolute deadline.

**Tech Stack:** Go, standard library regexp testing, existing parser stack

---

## Chunk 1: Schema And Failing Tests

### Task 1: Add schema slot and failing tests

**Files:**
- Modify: `pkg/schema/message.go`
- Modify: `pkg/parser/parser_test.go`
- Modify: `testdata/golden/verification.json`

- [x] **Step 1: Add an optional `expires_in_seconds` field to `schema.Code`**
- [x] **Step 2: Write a failing test for English expiry extraction**
- [x] **Step 3: Write a failing test for Chinese expiry extraction**
- [x] **Step 4: Update the verification golden fixture to expect expiry**
- [x] **Step 5: Run focused parser tests and confirm they fail**

## Chunk 2: Minimal Extraction

### Task 2: Implement conservative expiry parsing

**Files:**
- Modify: `pkg/parser/codes.go`

- [x] **Step 1: Extract nearby relative expiry durations from verification-code text**
- [x] **Step 2: Support minutes for English and Chinese patterns**
- [x] **Step 3: Attach expiry metadata to extracted codes**
- [x] **Step 4: Run focused parser tests and confirm they pass**

## Chunk 3: Docs And Verification

### Task 3: Document expiry metadata

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/en/agent-workflows.md`
- Modify: `docs/zh-CN/agent-workflows.md`

- [ ] **Step 1: Update docs to mention `expires_in_seconds`**
- [ ] **Step 2: Run full repository tests**
- [ ] **Step 3: Commit**
- [ ] **Step 4: Push**
