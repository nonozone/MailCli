[中文](../../zh-CN/spec/outbound-message.md) | English

# Outbound Message Spec

## Purpose

Outbound flows should not require AI agents to construct raw MIME directly.

Instead, agents should produce intent-level message drafts, and `mailcli` should compile those drafts into standards-compliant email with the correct headers, MIME structure, and provider transport behavior.

## Core Objects

### `DraftMessage`

Used for a new outgoing email.

```json
{
  "account": "work",
  "from": {
    "name": "Nono",
    "address": "support@nono.im"
  },
  "to": [
    { "address": "user@example.com" }
  ],
  "cc": [],
  "bcc": [],
  "subject": "Welcome",
  "body_md": "Hello, welcome to MailCLI.",
  "body_text": "Hello, welcome to MailCLI.",
  "headers": {
    "X-MailCLI-Agent": "assistant"
  },
  "attachments": []
}
```

### `ReplyDraft`

Used for replying to an existing message.

```json
{
  "account": "work",
  "reply_to_message_id": "<orig-123@example.com>",
  "to": [
    { "address": "user@example.com" }
  ],
  "subject": "Re: Your question",
  "body_md": "Thanks for your email.\n\nHere is the answer...",
  "attachments": []
}
```

Alternative reply target:

```json
{
  "account": "work",
  "reply_to_id": "imap:uid:12345",
  "body_md": "Thanks, got it."
}
```

When `reply_to_id` is used, `mailcli` may fetch the original message through the configured driver and derive:

- `reply_to_message_id`
- `references`
- default reply subject
- default reply recipient when `to` is omitted

### `SendResult`

Returned after a send or reply operation.

```json
{
  "ok": true,
  "message_id": "<sent-456@example.com>",
  "thread_id": "<orig-123@example.com>",
  "provider": "imap-smtp",
  "account": "work"
}
```

Failure example:

```json
{
  "ok": false,
  "error": {
    "code": "auth_failed",
    "message": "SMTP authentication failed"
  },
  "provider": "imap-smtp",
  "account": "work"
}
```

Current stable error codes:

- `auth_failed`
- `account_not_found`
- `account_not_selected`
- `message_not_found`
- `transport_not_configured`
- `invalid_draft`
- `transport_failed`

Implementations should prefer typed operational errors internally where practical, then map them into this stable public code set.

## Compiler Responsibilities

`mailcli` should compile `DraftMessage` and `ReplyDraft` into real email messages and automatically manage:

- `Message-ID`
- `Date`
- `From`
- `In-Reply-To`
- `References`
- `multipart/alternative`
- text and HTML body generation
- charset normalization
- content-transfer encoding
- attachment packaging

## v0.1 RC Support Matrix

Current baseline behavior:

- `body_text` only: emits a single `text/plain` message
- `body_md` plus optional `body_text`: emits `multipart/alternative`
- `attachments`: emits `multipart/mixed`, with the body as the first part
- `reply_to_message_id` and `references`: emitted into reply headers
- non-dry-run outbound commands may fill `from.address` from configured `smtp_username` or `username`
- `reply_to_id` may also derive a default reply recipient from the original message when `to` is omitted

Current limitations:

- HTML output is generated from Markdown with a deliberately simple renderer
- attachment support is path-based; inline attachments and remote fetch are out of scope
- this is a transport-safe baseline, not a full newsletter-grade MIME engine

Current Markdown rendering baseline now preserves:

- headings
- Markdown links as clickable HTML anchors
- unordered lists as `ul/li`
- ordered lists as `ol/li`
- blockquotes
- simple Markdown tables
- readable plain-text link fallbacks such as `Label: https://...`

Concrete JSON and MIME pairs are documented in:

- [Outbound Draft Patterns](../examples/outbound-draft-patterns.md)

## Command Direction

Recommended future commands:

```bash
cat draft.json | mailcli send -
cat reply.json | mailcli reply -
```

This keeps the contract language-agnostic and works well for agents, shell scripts, Python, and Node.js.

## Current Status

The repository already contains a local MIME composer in `pkg/composer` for `DraftMessage` and `ReplyDraft`.

That composer now supports:

- plain `text/plain` output
- `multipart/alternative` when Markdown content is present
- `multipart/mixed` when attachments are present

`mailcli send` is now wired for driver-backed transport when an account is configured.

`mailcli reply` is also wired for driver-backed transport when an account is configured.

For non-dry-run outbound commands, MailCLI now returns `SendResult` JSON on both success and operational failure, so agents can branch on `error.code` without scraping raw stderr text.
