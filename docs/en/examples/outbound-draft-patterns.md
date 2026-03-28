[中文](../../zh-CN/examples/outbound-draft-patterns.md) | English

# Outbound Draft Patterns

This page shows the smallest stable outbound objects an agent should hand back to MailCLI after it has already made a decision.

The goal is not to teach MIME.

The goal is to show which JSON boundary to return for three common outbound cases:

1. short acknowledgement reply
2. richer support-style reply
3. brand new outbound message

## Stored Example Artifacts

- [ack-reply.draft.json](../../../examples/artifacts/outbound-patterns/ack-reply.draft.json)
- [ack-reply.mime.txt](../../../examples/artifacts/outbound-patterns/ack-reply.mime.txt)
- [minimal-reply.reply.json](../../../examples/artifacts/outbound-patterns/minimal-reply.reply.json)
- [minimal-reply.mime.txt](../../../examples/artifacts/outbound-patterns/minimal-reply.mime.txt)
- [support-followup.reply.json](../../../examples/artifacts/outbound-patterns/support-followup.reply.json)
- [support-followup.mime.txt](../../../examples/artifacts/outbound-patterns/support-followup.mime.txt)
- [release-update.draft.json](../../../examples/artifacts/outbound-patterns/release-update.draft.json)
- [release-update.mime.txt](../../../examples/artifacts/outbound-patterns/release-update.mime.txt)

The stored MIME files normalize generated values such as `Message-ID` and multipart boundaries so the examples stay readable and diff-friendly.

## Pattern 1: Short Acknowledgement Reply

Use this when the agent selected a local message or thread and only needs to send a safe confirmation.

Draft boundary:

```json
{
  "account": "fixtures",
  "from": { "address": "support@nono.im" },
  "to": [{ "address": "billing@example.com", "name": "Billing Team" }],
  "body_text": "Thanks, we received the invoice notification and queued it for processing.",
  "reply_to_id": "invoice.eml"
}
```

Compile command:

```bash
mailcli reply --config examples/config/fixtures-dir.yaml --account fixtures --dry-run examples/artifacts/outbound-patterns/ack-reply.draft.json
```

Why this shape matters:

- `reply_to_id` lets MailCLI re-fetch the original message and derive thread headers
- MailCLI can also derive a default reply recipient from the original message when `to` is omitted
- the agent does not need to construct `In-Reply-To` or `References`
- this is the best fit for zero-network demos and local retrieval loops

## Pattern 1B: Minimal Reply With Derived Addressing

Use this when the agent already chose the target message and you want the smallest possible reply boundary.

Draft boundary:

```json
{
  "account": "fixtures",
  "body_text": "Thanks, we received the invoice notification and queued it for processing.",
  "reply_to_id": "invoice.eml"
}
```

Compile command:

```bash
mailcli reply --config examples/config/fixtures-dir.yaml --account fixtures --dry-run examples/artifacts/outbound-patterns/minimal-reply.reply.json
```

Why this shape matters:

- MailCLI derives `from.address` from the configured account when the account already represents the sending identity
- MailCLI derives the default reply recipient from the original message
- this is the smallest useful handoff for agent-generated acknowledgement replies

## Pattern 2: Support Follow-Up Reply

Use this when an upstream system already knows the original `Message-ID` and the agent needs a more deliberate reply body.

Draft boundary:

```json
{
  "from": { "name": "Nono Support", "address": "support@nono.im" },
  "to": [{ "address": "sender@example.com", "name": "Example Sender" }],
  "subject": "Re: Plaintext message",
  "body_text": "Hi,\n\nWe checked your request and need one detail before we can proceed:\n\n- workspace name\n- expected delivery date\n\nYou can reply directly to this thread.\n\nOriginal request received and queued.",
  "body_md": "Hi,\n\nWe checked your request and need one detail before we can proceed:\n\n- workspace name\n- expected delivery date\n\nYou can reply directly to this thread.\n\n> Original request received and queued.",
  "reply_to_message_id": "<plain-123@example.com>",
  "references": ["<plain-123@example.com>"]
}
```

Compile command:

```bash
mailcli reply --dry-run examples/artifacts/outbound-patterns/support-followup.reply.json
```

Why this shape matters:

- `reply_to_message_id` keeps the draft portable even when MailCLI cannot re-fetch the original message
- explicit `references` preserves thread continuity across transports
- `body_md` keeps richer structure while `body_text` provides a safe plain-text fallback

## Pattern 3: New Outbound Message

Use this when the agent is initiating a new message rather than replying to an existing thread.

Draft boundary:

```json
{
  "from": { "name": "MailCLI Bot", "address": "updates@mailcli.dev" },
  "to": [{ "address": "team@example.com", "name": "MailCLI Contributors" }],
  "subject": "Weekly parser status",
  "body_text": "Weekly parser status\n\nWe shipped three improvements:\n1. Better thread examples\n2. Safer zero-network onboarding\n3. Cleaner outbound MIME docs\n\nArea / Status\nParser fixtures / Stable\nThread examples / Expanded\nDriver docs / Ready for review\n\nExamples index: https://github.com/nonozone/MailCli/tree/main/docs/en/examples/README.md",
  "body_md": "## Weekly parser status\n\nWe shipped three improvements:\n\n1. Better thread examples\n2. Safer zero-network onboarding\n3. Cleaner outbound MIME docs\n\n| Area | Status |\n| --- | --- |\n| Parser fixtures | Stable |\n| Thread examples | Expanded |\n| Driver docs | Ready for review |\n\nRead the [examples index](https://github.com/nonozone/MailCli/tree/main/docs/en/examples/README.md) for runnable flows."
}
```

Compile command:

```bash
mailcli send --dry-run examples/artifacts/outbound-patterns/release-update.draft.json
```

Why this shape matters:

- the agent stays at the intent layer and does not hand-author MIME
- MailCLI can emit `multipart/alternative` automatically
- this is the baseline pattern for status mail, summaries, and automation output

## Choosing Between Them

- Use `reply_to_id` when MailCLI already has local access to the original message.
- Use the minimal `reply_to_id` form when account config should provide sender identity and the original message should provide reply routing.
- Use `reply_to_message_id` plus `references` when thread metadata comes from elsewhere and must stay portable.
- Use `DraftMessage` when you are starting a new conversation and do not want reply headers at all.

## Related

- [Outbound Message Spec](../spec/outbound-message.md)
- [Agent Workflows](../agent-workflows.md)
- [Local Thread Demo](local-thread-demo.md)
- [Examples Index](README.md)
