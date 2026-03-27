[中文](../../zh-CN/release/announcement-kit.md) | English

# MailCLI Announcement Kit

This page collects reusable public-facing copy for announcing MailCLI.

Use it for:

- GitHub release descriptions
- X / Twitter posts
- Telegram or Discord posts
- Hacker News / Reddit / forum intros
- personal blog or changelog intros

## Positioning

MailCLI is an open-source email interface for agents.

It is not trying to be a traditional mail client.

It gives AI systems a stable boundary for:

- reading email as structured JSON and clean Markdown
- working with local message and thread context
- producing `DraftMessage` and `ReplyDraft` instead of raw MIME
- delegating mailbox and transport details to drivers and CLI contracts

## Short GitHub Description

MailCLI is an AI-native email interface for agents. It turns raw MIME into structured context, supports local thread-aware retrieval, and lets agents produce reply/send drafts through stable machine-facing contracts.

## GitHub Release Draft

Title:

`MailCLI v0.1.0-rc1`

Body:

```md
MailCLI is an open-source email interface built for agents.

Instead of pushing raw MIME, bloated HTML, and provider-specific quirks into prompts, MailCLI gives AI systems a stable machine-facing boundary for reading, understanding, replying to, and sending email.

Highlights in the current RC:

- Parse raw email into `StandardMessage` JSON and Markdown
- Read inboxes through baseline IMAP support with `list` and `get`
- Sync recent messages into a local index
- Search local messages without re-fetching remote mail
- Inspect local thread summaries and full local threads
- Support thread-aware local triage with `thread_id`, labels, categories, actions, and code-related signals
- Compile outbound drafts and replies through `DraftMessage` and `ReplyDraft`
- Support external provider workflows for both single-message and thread-aware agent examples
- Ship Python, shell, template-provider, and optional OpenAI-provider examples

Stable contracts for integrators:

- `mailcli parse`
- `mailcli list`
- `mailcli get`
- `mailcli sync`
- `mailcli search`
- `mailcli threads`
- `mailcli reply`
- `mailcli send`
- `StandardMessage`
- `DraftMessage`
- `ReplyDraft`
- `SendResult`

Recommended docs:

- `README.md`
- `docs/en/agent-workflows.md`
- `docs/en/examples/README.md`
- `docs/en/spec/agent-provider.md`
```

## Short Announcement

MailCLI is now usable as an open-source email interface for agents.

It parses email into structured JSON, supports local thread-aware retrieval, and lets agents draft replies through stable CLI contracts instead of hand-written MIME.

Repo: `github.com/nonozone/MailCli`

## X / Twitter Version

MailCLI is an open-source email interface for agents.

It turns raw MIME into structured JSON + Markdown, supports local thread-aware retrieval, and lets agents produce reply/send drafts through stable CLI contracts.

Not a mail client for humans. A mail boundary for AI systems.

## Hacker News / Forum Intro

I’ve been building MailCLI, an open-source email interface for agents.

The goal is not to build another terminal mail client. The goal is to give AI systems a stable way to read, understand, reply to, and send email without pushing raw MIME and provider quirks into prompts.

Current scope includes:

- parsing raw email into structured JSON and Markdown
- baseline IMAP read path
- SMTP-backed send path
- local sync, search, and thread summaries
- thread-aware reply workflows
- external provider handoff for model-backed analysis

What I care about most is making the machine-facing contracts stable enough that agent developers can treat email like an interface, not a scraping problem.

## Key Talking Points

- “Email interface for agents” is the primary frame.
- “Not a traditional mail client” helps reduce the wrong expectations.
- “Stable machine-facing contracts” is the core value proposition.
- “Thread-aware local workflows” is one of the strongest differentiators in the current repo state.
- “External provider handoff” makes the project easy to pair with different model stacks without bloating core dependencies.

## Recommended Links

- [README](../../../README.md)
- [Agent Workflows](../agent-workflows.md)
- [Examples Index](../examples/README.md)
- [Agent Provider Contract](../spec/agent-provider.md)
