# Parser Chinese View-Online And Report-Abuse Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add bounded Chinese `view_online` and `report_abuse` extraction backed by a real fixture, golden output, and focused classifier guard tests.

**Architecture:** Keep the parser pipeline unchanged and extend only the `view_online` and `report_abuse` classifier branches. Protect the change with a fixture-backed test plus focused positive and negative classifier tests so the rule surface stays narrow and auditable.

**Tech Stack:** Go, standard library tests, existing parser golden-fixture workflow

---

## Chunk 1: Fixture And Classifier

### Task 1: Add the failing fixture-backed parser test

**Files:**
- Create: `testdata/emails/view_online_abuse_cn.eml`
- Create: `testdata/golden/view_online_abuse_cn.json`
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write the failing test**

Add a new fixture-backed parser test:

```go
func TestParseChineseViewOnlineAndReportAbuseEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/view_online_abuse_cn.eml", "../../testdata/golden/view_online_abuse_cn.json")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./pkg/parser -run TestParseChineseViewOnlineAndReportAbuseEmail -count=1
```

Expected: FAIL because the current parser output does not yet match the expected Chinese `view_online` and `report_abuse` behavior.

The fixture must use a neutral preview URL such as
`https://example.cn/campaign/preview/weekly` so the test cannot pass through the
existing English href patterns.

### Task 2: Add focused classifier tests and minimal implementation

**Files:**
- Modify: `pkg/parser/actions.go`
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Add classifier guard tests**

Add tests that prove:

- `在线查看` classifies as `view_online`
- `举报滥用` classifies as `report_abuse`
- `查看活动详情` does not classify as `view_online`
- `举报问题` does not classify as `report_abuse`
- `联系客服` does not classify as `report_abuse`

- [ ] **Step 2: Implement the minimal classifier change**

Add narrow raw-label phrase lists:

- `在线查看`
- `在网页中查看`
- `举报滥用`
- `举报垃圾邮件`

- [ ] **Step 3: Run targeted parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestParseChineseViewOnlineAndReportAbuseEmail|TestExtractActionsClassifiesViewOnlineLink|TestExtractActionsClassifiesReportAbuseLink|TestExtractActionsClassifiesChineseViewOnlineLink|TestExtractActionsClassifiesChineseReportAbuseLink|TestExtractActionsDoesNotClassifyChineseContentViewLinkAsViewOnline|TestExtractActionsDoesNotClassifyChineseIssueReportLinkAsReportAbuse|TestExtractActionsDoesNotClassifyChineseSupportLinkAsReportAbuse' -count=1
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
git add testdata/emails/view_online_abuse_cn.eml testdata/golden/view_online_abuse_cn.json pkg/parser/actions.go pkg/parser/parser_test.go examples/artifacts/local-thread-demo/sync.json examples/artifacts/local-thread-demo/agent-report.json docs/superpowers/specs/2026-03-28-parser-cn-viewonline-abuse-design.md docs/superpowers/plans/2026-03-28-parser-cn-viewonline-abuse.md
git commit -m "feat: add chinese view-online and abuse fixture coverage"
git push origin main
```
