[中文](../../zh-CN/contributing/parser.md) | English

# Parser Contributor Guide

This guide explains how to contribute to MailCLI's shared parser without weakening the machine-facing contract.

The parser is one of the highest-value parts of the project.

It is also one of the easiest places to cause subtle regressions.

## What The Parser Owns

The parser takes raw message bytes and returns a `StandardMessage`.

Its job is to:

- decode MIME structure and transfer encodings
- normalize charsets into UTF-8 text
- choose the best body representation for agent use
- clean noisy HTML into Markdown-friendly content
- extract machine-usable actions, codes, and bounce context
- estimate token usage conservatively

Its job is not to:

- talk to mailbox providers
- apply provider-specific business policy
- decide inbox workflow policy for a specific product
- hide transport failures that belong in drivers

## Important Files

Start here before changing parser behavior:

- `pkg/parser/parser.go`
- `pkg/parser/mime.go`
- `pkg/parser/charset.go`
- `pkg/parser/html.go`
- `pkg/parser/actions.go`
- `pkg/parser/codes.go`
- `pkg/parser/token.go`
- `pkg/parser/parser_test.go`

Fixture and golden data live in:

- `testdata/emails/`
- `testdata/golden/`

CLI JSON snapshots that protect downstream output shape live in:

- `cmd/json_snapshot_test.go`
- `cmd/testdata/snapshots/`

## Safe Workflow

Use this order unless there is a strong reason not to:

1. Add a focused failing test in `pkg/parser/parser_test.go`.
2. Add or update a fixture in `testdata/emails/` if the case is end-to-end.
3. Update the matching golden file in `testdata/golden/` only when the new output is intentional.
4. Run targeted parser tests first.
5. Run the full test suite before opening a PR.

Recommended commands:

```bash
go test ./pkg/parser -run TestParse
go test ./pkg/parser -run TestCleanHTML
go test ./...
```

If your parser change affects command JSON output, also run:

```bash
go test ./cmd
```

## Heuristic Boundaries

Parser work is allowed to be heuristic.

Parser work is not allowed to be vague.

Good parser heuristics:

- are motivated by a concrete fixture or failure case
- are conservative when evidence is weak
- keep useful structure like headings, links, tables, and key images
- come with regression coverage
- improve agent-facing clarity without inventing meaning

Bad parser heuristics:

- special-case one provider brand without a reusable pattern
- delete content because it "looks like marketing" without evidence
- rewrite URLs that are not obvious tracking wrappers
- classify actions from weak hints alone
- change output shape without updating docs and tests

## HTML Cleanup Rules

When working on `html.go`, optimize for these outcomes:

- keep the primary body content
- remove obvious chrome such as preheaders, footers, navigation, and preference links
- avoid throwing away short transactional bodies
- preserve valuable structure for Markdown conversion

Do not assume that footer-like words mean the whole container is safe to delete.

Prefer scoring and narrow removal over broad keyword deletion.

## Action And URL Rules

When working on `actions.go`:

- normalize only obvious redirect wrappers
- preserve legitimate destination URLs
- keep labels readable for agents
- deduplicate actions deterministically
- avoid turning generic product links into semantic actions without strong evidence

If a URL cleanup rule could destroy a legitimate app link, the rule is too aggressive.

## Fixture Guidance

Prefer fixtures organized by behavior, not by brand.

Good fixture categories:

- newsletter or promo
- transactional status update
- verification or sign-in
- invoice or payment
- bounce or delivery failure
- security reset
- attachment entry

Representative fixtures already in the repo:

- `mercury.eml`: large HTML-heavy newsletter or promo mail
- `verification.eml`: straightforward verification flow
- `reply_quoted_verification.eml`: verification content inside quoted thread noise
- `invoice.eml`: transactional billing or invoice flow
- `attachment_notice.eml`: message where the main user action is attachment access
- `bounce.eml` and `postfix_bounce.eml`: delivery failure and machine-generated bounce context
- `unsubscribe_mixed.eml`: mixed-content mail where unsubscribe must stay available but not dominate the body

When adding a fixture:

- anonymize addresses, ids, and tokens when possible
- keep the raw mail realistic enough to preserve the failure mode
- add the smallest fixture that still protects the behavior

## Multilingual Action Fixtures

MailCLI already carries multilingual action coverage, including Chinese transactional and preference-management cases.

Representative multilingual fixtures:

- `unsubscribe_cn.eml`
- `confirm_subscription_cn.eml`
- `invoice_cn.eml`
- `security_reset_cn.eml`
- `security_verify_cn.eml`
- `verification_cn_fullwidth.eml`
- `view_online_abuse_cn.eml`
- `attachment_notice_cn.eml`

When expanding multilingual coverage:

- prefer adding a real action pattern over adding generic translated copy
- cover the action label and the surrounding body context together
- include at least one negative case if a new keyword could overfire
- keep heuristics language-aware, but still pattern-based rather than sender-branded

If a new rule only works because one provider always uses one exact phrase, it probably belongs in a narrower test or should not land yet.

## Golden Update Rules

Golden files are contract protection, not a shortcut for approving changed output.

Before updating a golden:

- read the full diff, not only the changed lines
- confirm the new output is easier for agents to consume
- confirm token or structure changes are intentional
- check whether the change also affects `cmd/json_snapshot_test.go`

Avoid bundling unrelated parser cleanup into the same golden refresh. Small, explainable diffs are easier to review and safer to keep.

## Recommended Verification Loop

For parser-focused work, this is the preferred loop:

```bash
go test ./pkg/parser -run TestParse
go test ./pkg/parser -run TestExtractActions
go test ./pkg/parser -run TestExtractCodes
go test ./cmd
go test ./...
go build -o /tmp/mailcli ./cmd/mailcli
```

If your change affects local demo artifacts or thread-facing JSON output, also run:

```bash
make demo-local-thread-refresh
make demo-local-thread-check
```

The first command regenerates the checked-in demo artifacts.

The second command proves the refreshed artifacts still match repository expectations.

## Contract Changes

If your parser change affects:

- `StandardMessage`
- action or code shape
- command JSON output
- any documented stable field

open an RFC-style issue first instead of only sending a code PR.

See:

- [RFC Contract Change Template](../../../.github/ISSUE_TEMPLATE/rfc_contract_change.md)
- [Agent Decision Spec](../spec/agent-decisions.md)

## Review Standard

A parser contribution is stronger when:

- the behavior is easier for agents to consume
- the regression is clearly covered by tests
- the rule generalizes beyond one provider
- the output contract becomes clearer, not noisier

If you are unsure whether a behavior belongs in the parser, default to a smaller change and document the uncertainty in the PR.
