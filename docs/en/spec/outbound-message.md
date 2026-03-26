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

## Command Direction

Recommended future commands:

```bash
cat draft.json | mailcli send -
cat reply.json | mailcli reply -
```

This keeps the contract language-agnostic and works well for agents, shell scripts, Python, and Node.js.

## Current Status

The repository already contains a local MIME composer in `pkg/composer` for `DraftMessage` and `ReplyDraft`.

`mailcli send` is now wired for driver-backed transport when an account is configured.

`mailcli reply` is also wired for driver-backed transport when an account is configured.
