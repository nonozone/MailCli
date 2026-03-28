# Parser Chinese Attachment Actions Design

## Goal

Add one bounded parser-quality improvement for multilingual document mail:
recognize Chinese attachment-entry links from anchor labels even when the URL does
not expose the current English attachment heuristics.

## Why This Slice

The parser already extracts:

- `view_attachment` from English labels such as `View attachment`
- `download_attachment` from English labels such as `Download attachment`

That still leaves a gap for Chinese document mail where the useful signal is the
anchor text rather than the URL.

This slice stays intentionally small:

- one new attachment fixture
- one fixture-backed parser test
- two positive classifier tests
- two negative classifier tests
- no schema changes

## Scope

In scope:

- add one Chinese attachment fixture and golden output
- ensure the fixture uses neutral URLs that do not match the existing English attachment heuristics
- add bounded Chinese label detection for `view_attachment`
- add bounded Chinese label detection for `download_attachment`
- add focused positive and negative classifier tests
- refresh repository demo artifacts if fixture-count-based examples drift

Out of scope:

- new attachment metadata extraction
- composer attachment packaging changes
- broader multilingual document semantics

## Design

Keep the action extraction pipeline unchanged and extend only the
`view_attachment` and `download_attachment` classifier branches.

The classifier should continue to accept current English label and href behavior.
It should additionally accept a short Chinese phrase list in the raw anchor label.

The fixture must use neutral URLs like `https://example.cn/files/view/...` and
`https://example.cn/files/fetch/...` so the end-to-end test proves the
Chinese-label path rather than English href patterns.

Use narrow phrases:

- `查看附件`
- `打开附件`
- `下载附件`
- `下载文件`

## Test Strategy

Use TDD:

1. add the fixture-backed parser test
2. verify the parser fails with a golden mismatch because actions are missing
3. add positive and negative classifier tests
4. implement the smallest classification change
5. run targeted parser tests
6. run full repository verification and refresh demo artifacts if needed

## Risks

- false positives from generic file center or document list labels
- fixture-count changes causing local demo artifact drift

Mitigation:

- only match explicit attachment phrases
- include a generic file center negative test
- include a document list negative test
- use the standard demo refresh/check workflow after verification

