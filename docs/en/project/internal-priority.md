[中文](../../zh-CN/project/internal-priority.md) | English

# Internal Development Priority

This page defines the realistic next-step order if MailCLI is still mostly maintainer-driven.

The assumption is simple:

- community participation is welcome
- community participation is still limited
- core product direction still needs to be carried by the maintainers

So the goal is not to maximize issue count.

The goal is to pick the smallest set of work that moves MailCLI from "good RC demo" toward "something agents can reliably build against."

## Completed Baseline Sequence

The original maintainer-led "core five" hardening sequence is now complete on `main`:

1. CLI JSON snapshot tests
2. stronger HTML body extraction
3. cleaner tracked URL normalization
4. clearer local sync/index state
5. richer thread summary triage signals

The follow-up maintenance loop for the stored local thread demo artifacts is also now wired into repository-level commands and CI checks.

That means the next maintainer phase should move from boundary hardening toward a Go-only core, existing-mailbox UX, inbox intelligence, and safe automation loops.

## Working Assumption

For the next stage, treat MailCLI as:

- official executable behavior belongs in Go core
- maintainers own core contracts, parser quality, and the local memory model
- AI providers stay outside core through language-neutral JSON contracts
- Go examples are the official path; do not maintain a second official runtime path
- docs, fixtures, examples, and smaller contributor-surface tasks can be community-assisted

## The Next Maintainer Four

These are the next four tasks that should be treated as the main internal development sequence. The fifth track, dedicated Agent mailboxes or broad provider expansion, is deferred for now.

### 1. Existing mailbox setup and configuration: Go core

Why first:

- most users already have Gmail, Outlook, QQ, 163, or corporate mailboxes; they will not start with a dedicated Agent mailbox
- the real need is helping AI safely understand and process existing mailboxes
- install, configuration, connection tests, and account-capability output should come from the Go binary, without another language runtime
- the first implementation slice should make `config init`, `config doctor`, `config test`, and `config capabilities` cover setup, static diagnostics, live checks, and machine-readable capability discovery

Why maintainers should own it:

- this defines MailCLI's first experience for normal users
- config contracts and account-capability output affect every later agent workflow

## 2. Inbox/thread summaries, priority, and todo extraction

Why second:

- users want to know what matters, what needs a reply, and what should happen next
- local thread/search is already usable; the next step is explainable triage signals
- Go should provide structured candidate data, while LLM providers remain external for interpretation and recommendations

Why maintainers should own it:

- summaries and priority signals affect public JSON shape and prompt usage
- weak local memory semantics will leak into examples and user workflows quickly

## 3. Attachment, invoice, code, and action-link extraction

Why third:

- these are the most common and automation-worthy signals in ordinary inboxes
- the existing action/code baseline is already in place; follow-up work should cover inbound attachments and invoice entry points
- this improves AI usefulness for existing mailboxes more directly than adding a new provider

Why maintainers should own it:

- parser/schema field design defines the product quality bar
- fixture choice and golden output should reflect maintainer judgment about real mail scenarios

## 4. Draft, confirmation, execution, and operation logging

Why fourth:

- automation must be controllable, especially for high-impact actions such as send, delete, move, and mark
- `send`, `reply`, and mailbox mutations already work; the next step is prepare / confirm / log
- this moves agents from "can call commands" to "can execute within an auditable boundary"

Why maintainers should own it:

- it is close to the public CLI contract and the user trust boundary
- confirmation tokens, intent ids, and operation log fields need to be stable from the start

## Recommended Sequence

1. Tighten existing-mailbox setup, configuration, account capabilities, and Go-first examples.
2. Improve inbox/thread summaries, priority, and todo extraction.
3. Improve attachment, invoice, code, and action-link extraction.
4. Add the draft / confirm / execute / operation-log loop.

## What Can Wait

These are useful, but should not displace the next maintainer four:

- docs alignment cleanup
- parser contributor guide
- broad provider expansion
- broader community process work
- dedicated Agent mailbox or hosted mailbox identity
- maintaining a second official runtime path beside Go

They matter, but they are support work around the product core, not the main path to a stronger `v0.2`.

## Community-Suitable Parallel Work

While maintainers focus on the core four, the community can still help with:

- fixture collection and anonymization
- examples
- docs polish
- small parser regression reports
- contributor guides
- test-case additions that do not redefine contracts
- unofficial language examples, as long as they do not become installation or usage prerequisites

## Rule Of Thumb

If a task changes:

- public JSON shape
- parser quality bar
- thread summary semantics
- local memory semantics
- Go CLI execution / confirmation / audit contracts

it should be maintainer-driven first.

If a task improves:

- docs
- examples
- fixtures
- contributor onboarding

it is a good place to welcome outside participation earlier.

## Related

- [Next Development Roadmap](next-roadmap.md)
- [GitHub Backlog Drafts](github-backlog.md)
- [Parser Contributor Guide](../contributing/parser.md)
