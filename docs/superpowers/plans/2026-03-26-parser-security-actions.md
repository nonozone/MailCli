# Parser Security Actions Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract security-related actions from HTML links so agents can recognize obvious account-recovery and sign-in verification entry points without changing the public schema.

**Architecture:** Keep `schema.Action` unchanged. Extend anchor classification with `reset_password` and `verify_sign_in`, using explicit label text and narrow href patterns only.

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

### Task 1: Add parser tests for security actions

**Files:**
- Modify: `pkg/parser/parser_test.go`

- [x] **Step 1: Write a failing test for `reset_password`**

Verify anchor text like `Reset password` produces:
- `type: "reset_password"`
- preserved label
- URL preserved

- [x] **Step 2: Write a failing test for `verify_sign_in`**

Verify anchor text like `Verify sign-in` produces:
- `type: "verify_sign_in"`

- [x] **Step 3: Write negative tests**

Verify generic links such as:
- `Sign in`
- `Reset account settings`

do not get misclassified as security actions.

- [x] **Step 4: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestExtractActions(ClassifiesResetPasswordLink|ClassifiesVerifySignInLink|DoesNotClassifyGenericSignInLinkAsVerifySignIn|DoesNotClassifyGenericAccountResetLinkAsResetPassword)' -v
```

Expected: FAIL because the parser currently does not classify these security actions.

## Chunk 2: Minimal Implementation

### Task 2: Extend anchor classification

**Files:**
- Modify: `pkg/parser/actions.go`

- [x] **Step 1: Add conservative security action matching**

Support:
- `reset_password`
- `verify_sign_in`

using explicit labels and narrow href patterns.

- [x] **Step 2: Preserve existing label behavior**

Keep the original anchor text as the action label when available.

- [x] **Step 3: Run focused parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestExtractActions(ClassifiesResetPasswordLink|ClassifiesVerifySignInLink|DoesNotClassifyGenericSignInLinkAsVerifySignIn|DoesNotClassifyGenericAccountResetLinkAsResetPassword)' -v
```

Expected: PASS.

## Chunk 3: Documentation And Verification

### Task 3: Align docs and verify repository state

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/en/agent-workflows.md`
- Modify: `docs/zh-CN/agent-workflows.md`

- [x] **Step 1: Update docs to mention security actions**

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
git add pkg/parser/actions.go pkg/parser/parser_test.go README.md README.zh-CN.md docs/en/agent-workflows.md docs/zh-CN/agent-workflows.md docs/superpowers/plans/2026-03-26-parser-security-actions.md
git commit -m "feat: extract security actions"
```

- [ ] **Step 4: Push**

Run:

```bash
git push origin main
```
