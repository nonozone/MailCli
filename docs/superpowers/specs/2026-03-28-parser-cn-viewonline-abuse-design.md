# Parser Chinese View-Online And Report-Abuse Design

## Goal

Add one bounded parser-quality improvement for multilingual notification mail:
recognize Chinese `view_online` and `report_abuse` links from anchor labels even
when the URL does not expose the current English heuristics.

## Why This Slice

The parser already extracts:

- `view_online` from English labels such as `View online`
- `report_abuse` from English labels such as `Report abuse`
- `report_abuse` from some abuse-oriented header targets

That still leaves a gap for Chinese notification mail where the useful signal is
the anchor text rather than the URL or mailbox name.

This slice stays intentionally small:

- one new notification fixture
- one fixture-backed parser test
- two positive classifier tests
- three negative classifier tests
- no schema changes

## Scope

In scope:

- add one Chinese notification fixture and golden output
- ensure the fixture uses a neutral `view_online` URL that does not match the current English href heuristics
- add bounded Chinese label detection for `view_online`
- add bounded Chinese label detection for `report_abuse`
- add focused positive and negative classifier tests
- refresh repository demo artifacts if fixture-count-based examples drift

Out of scope:

- broader multilingual moderation semantics
- provider-specific abuse handling
- changes to header-based abuse extraction

## Design

Keep the action extraction pipeline unchanged and extend only the
`view_online` and `report_abuse` classifier branches.

The classifier should continue to accept current English label and href behavior.
It should additionally accept a short Chinese phrase list in the raw anchor label.

Use narrow phrases:

- `在线查看`
- `在网页中查看`
- `举报滥用`
- `举报垃圾邮件`

The fixture must use a neutral preview URL like
`https://example.cn/campaign/preview/weekly` so the end-to-end test proves the
Chinese-label path rather than English href patterns.

## Test Strategy

Use TDD:

1. add the fixture-backed parser test
2. verify the parser fails with a golden mismatch because actions are missing
3. add positive and negative classifier tests
4. implement the smallest classification change
5. run targeted parser tests
6. run full repository verification and refresh demo artifacts if needed

## Risks

- false positives from generic Chinese “view” or “report” labels
- fixture-count changes causing local demo artifact drift

Mitigation:

- only match explicit online-view and abuse phrases
- include a generic content-view negative test
- include a generic issue-report negative test
- keep the view-online phrases exact-match based

