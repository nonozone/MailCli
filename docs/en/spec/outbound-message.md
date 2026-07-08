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

## Prepared Send Intents

For agent automation, a new outbound message should normally be prepared before
it is sent:

```bash
mailcli send prepare --config ~/.config/mailcli/config.yaml draft.json
mailcli send confirm --config ~/.config/mailcli/config.yaml <intent-id>
mailcli operations list
mailcli operations show <operation-id|intent-id>
```

`send prepare` validates the draft, composes enough MIME to allocate a stable
`Message-ID`, writes a send intent, and returns `OperationIntentResult` JSON. It
does not initialize the transport driver and does not send mail.

Example prepare result:

```json
{
  "status": "prepared",
  "intent_id": "intent_...",
  "operation": "send",
  "account": "work",
  "message_id": "<...>",
  "operations_path": "/home/user/.local/state/mailcli/operations.jsonl",
  "confirm_command": "mailcli send confirm --operations /home/user/.local/state/mailcli/operations.jsonl intent_...",
  "summary": {
    "subject": "Welcome",
    "to": [{"address": "user@example.com"}]
  }
}
```

`send confirm` reloads the stored intent and sends that exact draft. On success,
`SendResult` includes both `intent_id` and `operation_id`. On operational
failure, MailCLI still returns structured `SendResult` JSON with `error.code`
and appends a failed operation entry when the intent could be loaded. Confirm
failures include `intent_id` and the failed `operation_id` in the returned
`SendResult`, so agents can jump directly to `mailcli operations show`.
If an intent already has a successful `sent` operation entry, a later
`send confirm` is rejected with `error.code = "intent_already_sent"` before the
transport driver is called.

The operation log is JSONL. By default it lives at:

```text
~/.local/state/mailcli/operations.jsonl
```

Intent payloads are stored next to the log in:

```text
~/.local/state/mailcli/operations.jsonl.intents/
```

The log stores summaries only: subject, from, visible recipients, Bcc count,
attachment count, status, IDs, timestamps, and structured errors. It must not
store full `body_text`, full `body_md`, raw MIME, or configured secrets. Intent
payload files necessarily store the full draft so confirm can execute the same
prepared operation; they are written with owner-only permissions.

## Command Direction

Recommended commands:

```bash
cat draft.json | mailcli send -
cat reply.json | mailcli reply -
mailcli send prepare draft.json
mailcli send confirm <intent-id>
```

This keeps the contract language-agnostic and works well for agents, shell scripts, Go, Node.js, and other runtimes.

## Current Status

The repository already contains a local MIME composer in `pkg/composer` for `DraftMessage` and `ReplyDraft`.

That composer now supports:

- plain `text/plain` output
- `multipart/alternative` when Markdown content is present
- `multipart/mixed` when attachments are present

`mailcli send` is now wired for driver-backed transport when an account is configured.

`mailcli send prepare` and `mailcli send confirm` are wired for the first
operation-intent phase for new outbound messages.

`mailcli reply` is also wired for driver-backed transport when an account is configured.

For non-dry-run outbound commands, MailCLI now returns `SendResult` JSON on both success and operational failure, so agents can branch on `error.code` without scraping raw stderr text.
