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

When adding a fixture:

- anonymize addresses, ids, and tokens when possible
- keep the raw mail realistic enough to preserve the failure mode
- add the smallest fixture that still protects the behavior

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
