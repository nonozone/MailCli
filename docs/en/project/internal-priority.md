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

That means the next maintainer phase should move from boundary hardening toward corpus strength, contributor leverage, and shared quality bars.

## Working Assumption

For the next stage, treat MailCLI as:

- maintainer-led in core contracts
- maintainer-led in parser quality
- maintainer-led in local memory model
- community-assisted in docs, fixtures, examples, and smaller contributor-surface tasks

## The Next Maintainer Five

These are the next five tasks that should be treated as the main internal development sequence.

### 1. Expand the parser fixture corpus around real failure modes

Why first:

- parser quality is now bottlenecked more by corpus coverage than by missing infrastructure
- more fixtures will make the current heuristics safer to iterate on
- this is the fastest way to keep improving agent usefulness without redefining contracts

Why maintainers should own it:

- fixture choice defines the real parser quality bar
- the first wave of corpus curation should reflect maintainer taste and product priorities

## 2. Add a fake-driver or conformance harness for contributor drivers

Why second:

- driver contribution is still harder than it needs to be
- a shared acceptance bar will lower maintainer review cost
- this opens a safer path for outside provider contributions

Why maintainers should own it:

- it defines the minimum transport contract the ecosystem will inherit
- a weak harness would create noisy provider PRs later

## 3. Tighten local search semantics and ranking

Why third:

- local memory is now a real user-facing loop
- better ranking and filtering can reduce unnecessary full-message loads
- this improves agent usefulness without introducing new transport surface area

Why maintainers should own it:

- search semantics are close to the long-term memory model
- weak ranking choices would leak into prompts and examples quickly

## 4. Improve outbound ergonomics without changing the core contract

Why fourth:

- send and reply already work, but contributor and user ergonomics can still improve
- this keeps the project useful while larger provider work is deferred
- it strengthens the read/write loop for agents end to end

Why maintainers should own it:

- send/reply ergonomics are close to stable public contracts
- maintainers should decide where convenience ends and schema sprawl begins

## 5. Add one more built-in provider only after the previous four are in shape

Why fifth:

- ecosystem breadth matters, but only once the shared layers are easier to extend safely
- another provider will strengthen the architecture story if it does not destabilize the core
- it is easier to review once parser, local memory, and driver guidance are stronger

Why maintainers should own it:

- the second real provider will shape community expectations for extension style
- maintainers still need to set the baseline for transport isolation and docs quality

## Recommended Sequence

1. Expand fixture coverage for parser regressions.
2. Build driver conformance tooling.
3. Improve local search semantics.
4. Improve outbound ergonomics.
5. Add one more provider.

## What Can Wait

These are useful, but should not displace the next maintainer five:

- docs alignment cleanup
- parser contributor guide
- broad provider expansion
- broader community process work

They matter, but they are support work around the product core, not the main path to a stronger `v0.2`.

## Community-Suitable Parallel Work

While maintainers focus on the core five, the community can still help with:

- fixture collection and anonymization
- examples
- docs polish
- small parser regression reports
- contributor guides
- test-case additions that do not redefine contracts

## Rule Of Thumb

If a task changes:

- public JSON shape
- parser quality bar
- thread summary semantics
- local memory semantics

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
