# Parser Chinese Confirm Subscription Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add bounded Chinese confirm-subscription action extraction backed by a real parser fixture, golden output, and classifier guard tests.

**Architecture:** Keep the parser pipeline unchanged and extend only the `confirm_subscription` classifier branch. Protect the change with a fixture-backed test plus focused positive and negative classifier tests so the rule stays narrow and auditable.

**Tech Stack:** Go, standard library tests, existing parser golden-fixture workflow

---

## Chunk 1: Fixture And Classifier

### Task 1: Add the failing fixture-backed parser test

**Files:**
- Create: `testdata/emails/confirm_subscription_cn.eml`
- Create: `testdata/golden/confirm_subscription_cn.json`
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write the failing test**

Add a new fixture-backed parser test:

```go
func TestParseChineseConfirmSubscriptionEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/confirm_subscription_cn.eml", "../../testdata/golden/confirm_subscription_cn.json")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./pkg/parser -run TestParseChineseConfirmSubscriptionEmail -count=1
```

Expected: FAIL because the current parser output does not yet match the expected Chinese confirm-subscription behavior.

The fixture must use a neutral URL such as `https://example.cn/welcome?...` so
the test cannot pass through the existing English href patterns.

### Task 2: Add focused classifier tests and minimal implementation

**Files:**
- Modify: `pkg/parser/actions.go`
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Add classifier guard tests**

Add tests that prove:

- `确认订阅` classifies as `confirm_subscription`
- `确认邮件订阅` classifies as `confirm_subscription`
- `确认订单` does not classify as `confirm_subscription`
- `确认邮件地址` does not classify as `confirm_subscription`

- [ ] **Step 2: Implement the minimal classifier change**

Add a narrow raw-label phrase list:

- `确认订阅`
- `确认邮件订阅`

- [ ] **Step 3: Run targeted parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestParseChineseConfirmSubscriptionEmail|TestExtractActionsClassifiesConfirmSubscriptionLink|TestExtractActionsClassifiesConfirmSubscriptionFromHrefPattern|TestExtractActionsClassifiesChineseConfirmSubscriptionLink|TestExtractActionsClassifiesChineseEmailConfirmSubscriptionLink|TestExtractActionsDoesNotClassifyChineseOrderConfirmationAsSubscription|TestExtractActionsDoesNotClassifyChineseEmailAddressConfirmationAsSubscription' -count=1
```

Expected: PASS

## Chunk 2: Verification

### Task 3: Run repository verification

**Files:**
- Verify the parser change, fixture, and example artifacts together

- [ ] **Step 1: Run repository verification**

Run:

```bash
git diff --check
go test ./...
go build -o /tmp/mailcli ./cmd/mailcli
make demo-local-thread-check
```

Expected: all pass

- [ ] **Step 2: Refresh demo artifacts if fixture-count examples drift**

Run:

```bash
make demo-local-thread-refresh
make demo-local-thread-check
```

Expected: checked-in local thread demo artifacts are current again.

### Task 4: Commit

**Files:**
- Commit the fixture, parser change, tests, demo artifact refresh, and spec/plan docs together

- [ ] **Step 1: Commit**

```bash
git add testdata/emails/confirm_subscription_cn.eml testdata/golden/confirm_subscription_cn.json pkg/parser/actions.go pkg/parser/parser_test.go examples/artifacts/local-thread-demo/sync.json examples/artifacts/local-thread-demo/agent-report.json docs/superpowers/specs/2026-03-28-parser-cn-confirm-subscription-design.md docs/superpowers/plans/2026-03-28-parser-cn-confirm-subscription.md
git commit -m "feat: add chinese confirm-subscription fixture coverage"
git push origin main
```
