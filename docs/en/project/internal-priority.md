[中文](../../zh-CN/project/internal-priority.md) | English

# Internal Development Priority

This page defines the realistic next-step order if MailCLI is still mostly maintainer-driven.

The assumption is simple:

- community participation is welcome
- community participation is still limited
- core product direction still needs to be carried by the maintainers

So the goal is not to maximize issue count.

The goal is to pick the smallest set of work that moves MailCLI from "good RC demo" toward "something agents can reliably build against."

## Working Assumption

For the next stage, treat MailCLI as:

- maintainer-led in core contracts
- maintainer-led in parser quality
- maintainer-led in local memory model
- community-assisted in docs, fixtures, examples, and smaller contributor-surface tasks

## The Core Five

These are the five tasks that should be treated as the main internal development sequence.

### 1. Add JSON contract snapshot tests for CLI commands

Why first:

- this locks down the machine-facing boundary before more output drift accumulates
- parser and thread improvements are much safer once output shape is pinned
- this protects agent integrations from accidental breakage

Why maintainers should own it:

- it defines what `v0.1.x` and `v0.2` are really promising
- it is close to schema and command contract decisions

## 2. Strengthen HTML body extraction and noise filtering

Why second:

- parser quality is the clearest source of product value
- if `body_md` is weak, the rest of the workflow is much less compelling
- this is one of the strongest differentiators for an AI-native email interface

Why maintainers should own it:

- it sets the quality bar for shared parser behavior
- bad cleanup decisions can quietly damage a lot of downstream output

## 3. Improve URL normalization for agent-facing actions

Why third:

- action quality directly affects whether agents can use extracted links reliably
- cleaner action URLs reduce prompt noise and improve automation quality
- it complements body extraction and makes action extraction more trustworthy

Why maintainers should own it:

- URL cleanup is easy to overdo
- bad normalization can destroy legitimate targets or hide useful context

## 4. Surface local index and sync state more clearly

Why fourth:

- local memory is one of the most practical reasons to use MailCLI in agent workflows
- current sync behavior is usable, but clearer state reporting will make it more dependable
- once parser quality is stronger, local retrieval becomes much more valuable

Why maintainers should own it:

- this shapes how the local memory model is explained and consumed
- it may affect future CLI output and local workflow semantics

## 5. Enrich thread summaries for triage loops

Why fifth:

- thread-aware local workflows are one of the strongest parts of the current repo state
- compact triage signals can reduce unnecessary full-thread loads
- this moves MailCLI closer to "agent memory interface" territory instead of just parser-plus-send

Why maintainers should own it:

- this is close to the long-term thread and memory model
- additive fields are easy to add, but hard to remove once people build around them

## Recommended Sequence

1. Lock CLI JSON shape with snapshot tests.
2. Improve `body_md` quality.
3. Improve extracted action URL quality.
4. Make local sync/index state easier to understand.
5. Make thread summaries more useful for triage.

## What Can Wait

These are useful, but should not displace the core five:

- docs alignment cleanup
- fake-driver contributor harness
- parser contributor guide
- more provider expansion
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
