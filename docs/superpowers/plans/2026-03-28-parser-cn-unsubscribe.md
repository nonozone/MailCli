# Parser Chinese Unsubscribe Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add bounded Chinese unsubscribe action extraction backed by a real parser fixture, golden output, and focused classifier guard tests.

**Architecture:** Keep the current parser pipeline intact and extend only the action-classification branch for unsubscribe detection. Protect the change with a fixture-backed failing test plus focused positive and negative classifier tests so the heuristic surface stays narrow and explicit.

**Tech Stack:** Go, standard library tests, existing parser golden-fixture workflow

---

## Chunk 1: Parser Fixture And Heuristic

### Task 1: Add the failing fixture-backed parser test

**Files:**
- Create: `testdata/emails/unsubscribe_cn.eml`
- Create: `testdata/golden/unsubscribe_cn.json`
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write the failing test**

Add a new fixture-backed parser test:

```go
func TestParseChineseUnsubscribeEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/unsubscribe_cn.eml", "../../testdata/golden/unsubscribe_cn.json")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./pkg/parser -run TestParseChineseUnsubscribeEmail -count=1
```

Expected: FAIL because the current parser output does not yet match the expected Chinese unsubscribe fixture behavior.

### Task 2: Implement the minimal parser change

**Files:**
- Modify: `pkg/parser/actions.go`
- Test: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write minimal implementation**

Add a helper or bounded branch so unsubscribe classification also matches these Chinese phrases:

- `退订`
- `取消订阅`
- `取消邮件订阅`

- [ ] **Step 2: Add focused positive and negative classifier tests**

Add tests that prove:

- `取消订阅` classifies as `unsubscribe` even on a neutral preferences URL
- `查看邮件偏好设置` does not classify as `unsubscribe`

- [ ] **Step 3: Run targeted parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestParseChineseUnsubscribeEmail|TestParseMixedUnsubscribeEmail|TestParseExtractsMailtoUnsubscribeAction|TestExtractActionsClassifiesChineseUnsubscribeLink|TestExtractActionsDoesNotClassifyGenericChinesePreferenceLinkAsUnsubscribe' -count=1
```

Expected: PASS

## Chunk 2: Verification

### Task 3: Run repository verification

**Files:**
- Verify the parser change, fixture, and tests together

- [ ] **Step 1: Run repository verification**

Run:

```bash
git diff --check
go test ./...
go build -o /tmp/mailcli ./cmd/mailcli
```

Expected: all pass

### Task 4: Commit

**Files:**
- Commit the fixture, parser change, tests, and optional doc updates together

- [ ] **Step 1: Commit**

```bash
git add testdata/emails/unsubscribe_cn.eml testdata/golden/unsubscribe_cn.json pkg/parser/actions.go pkg/parser/parser_test.go
git commit -m "feat: add chinese unsubscribe action detection"
git push origin main
```
