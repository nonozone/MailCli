# Parser Chinese Attachment Actions Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add bounded Chinese attachment action extraction backed by a real fixture, golden output, and focused classifier guard tests.

**Architecture:** Keep the parser pipeline unchanged and extend only the `view_attachment` and `download_attachment` classifier branches. Protect the change with a fixture-backed test plus focused positive and negative classifier tests so the rule surface stays narrow and auditable.

**Tech Stack:** Go, standard library tests, existing parser golden-fixture workflow

---

## Chunk 1: Fixture And Classifier

### Task 1: Add the failing fixture-backed parser test

**Files:**
- Create: `testdata/emails/attachment_notice_cn.eml`
- Create: `testdata/golden/attachment_notice_cn.json`
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write the failing test**

Add a new fixture-backed parser test:

```go
func TestParseChineseAttachmentNoticeEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/attachment_notice_cn.eml", "../../testdata/golden/attachment_notice_cn.json")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./pkg/parser -run TestParseChineseAttachmentNoticeEmail -count=1
```

Expected: FAIL because the current parser output does not yet match the expected Chinese attachment action behavior.

The fixture must use neutral URLs such as `https://example.cn/files/view/...`
and `https://example.cn/files/fetch/...` so the test cannot pass through the
existing English href patterns.

### Task 2: Add focused classifier tests and minimal implementation

**Files:**
- Modify: `pkg/parser/actions.go`
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Add classifier guard tests**

Add tests that prove:

- `查看附件` classifies as `view_attachment`
- `下载附件` classifies as `download_attachment`
- `前往文件中心` does not classify as `download_attachment`
- `查看文档列表` does not classify as `view_attachment`

- [ ] **Step 2: Implement the minimal classifier change**

Add narrow raw-label phrase lists:

- `查看附件`
- `打开附件`
- `下载附件`
- `下载文件`

- [ ] **Step 3: Run targeted parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestParseChineseAttachmentNoticeEmail|TestExtractActionsClassifiesViewAttachmentLink|TestExtractActionsClassifiesDownloadAttachmentLink|TestExtractActionsClassifiesChineseViewAttachmentLink|TestExtractActionsClassifiesChineseDownloadAttachmentLink|TestExtractActionsDoesNotClassifyChineseFileCenterLinkAsAttachmentDownload|TestExtractActionsDoesNotClassifyChineseDocumentListLinkAsViewAttachment' -count=1
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
git add testdata/emails/attachment_notice_cn.eml testdata/golden/attachment_notice_cn.json pkg/parser/actions.go pkg/parser/parser_test.go examples/artifacts/local-thread-demo/sync.json examples/artifacts/local-thread-demo/agent-report.json docs/superpowers/specs/2026-03-28-parser-cn-attachment-actions-design.md docs/superpowers/plans/2026-03-28-parser-cn-attachment-actions.md
git commit -m "feat: add chinese attachment action fixture coverage"
git push origin main
```
