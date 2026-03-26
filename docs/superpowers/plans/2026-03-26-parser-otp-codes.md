# Parser OTP Codes Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a small inbound `codes` schema and extract verification codes from email content for direct agent consumption.

**Architecture:** Keep existing parser flow intact. Add a narrow `schema.Code` type plus `StandardMessage.Codes`, extract codes from normalized text/Markdown with conservative heuristics, and verify the result through focused unit tests plus a parser fixture golden file.

**Tech Stack:** Go, standard library regexp testing, existing parser stack

---

**Follow-up enhancement:** After the initial `codes` field landed, parser extraction was tightened for real mail layouts by adding support for common Chinese verification phrases and codes placed on the next non-empty line after the security phrase.

## Chunk 1: Schema And Failing Tests

### Task 1: Add schema slot and failing tests

**Files:**
- Modify: `pkg/schema/message.go`
- Modify: `pkg/parser/parser_test.go`
- Create: `testdata/emails/verification.eml`
- Create: `testdata/golden/verification.json`

- [x] **Step 1: Add a `Code` schema type and `StandardMessage.Codes` field**
- [x] **Step 2: Write failing positive tests for verification code extraction**
- [x] **Step 3: Write failing negative tests for non-security numeric text**
- [x] **Step 4: Add a parser fixture test for a verification email**
- [x] **Step 5: Run focused parser tests and confirm they fail**

## Chunk 2: Minimal Extraction

### Task 2: Implement conservative code extraction

**Files:**
- Create: `pkg/parser/codes.go`
- Modify: `pkg/parser/parser.go`

- [x] **Step 1: Extract verification codes from normalized text**
- [x] **Step 2: Deduplicate by `type + value`**
- [x] **Step 3: Populate `StandardMessage.Codes` during parse**
- [x] **Step 4: Run focused parser tests and confirm they pass**

## Chunk 3: Docs And Verification

### Task 3: Document the new inbound field

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/en/agent-workflows.md`
- Modify: `docs/zh-CN/agent-workflows.md`

- [x] **Step 1: Update docs to mention inbound `codes`**
- [x] **Step 2: Run full repository tests**
- [ ] **Step 3: Commit**
- [ ] **Step 4: Push**
