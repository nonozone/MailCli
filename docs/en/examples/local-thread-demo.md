[中文](../../zh-CN/examples/local-thread-demo.md) | English

# Local Thread Demo

This page shows the full agent round trip for a local thread workflow without reading the Python example source.

It uses:

- `examples/config/fixtures-dir.yaml`
- the built-in `dir` driver
- the repository fixture corpus under `testdata/emails`

The goal is simple:

1. sync local state
2. rank candidate threads
3. load the selected thread
4. let the agent choose a reply action
5. compile a `ReplyDraft` into MIME

## Why This Demo Exists

The normal example pages explain how to run commands.

This page explains what the machine-facing inputs and outputs actually look like across the full loop.

## Command Sequence

```bash
mailcli sync --config examples/config/fixtures-dir.yaml --account fixtures --index /tmp/mailcli-fixtures-index.json --limit 20
mailcli threads --index /tmp/mailcli-fixtures-index.json invoice
mailcli thread --index /tmp/mailcli-fixtures-index.json "<invoice-123@example.com>"
mailcli reply --config examples/config/fixtures-dir.yaml --account fixtures --dry-run examples/artifacts/local-thread-demo/reply.draft.json
```

## Stored Example Artifacts

- [sync.json](../../../examples/artifacts/local-thread-demo/sync.json)
- [threads.json](../../../examples/artifacts/local-thread-demo/threads.json)
- [thread.json](../../../examples/artifacts/local-thread-demo/thread.json)
- [reply.draft.json](../../../examples/artifacts/local-thread-demo/reply.draft.json)
- [reply.mime.txt](../../../examples/artifacts/local-thread-demo/reply.mime.txt)
- [agent-report.json](../../../examples/artifacts/local-thread-demo/agent-report.json)

## Step 1: Sync

The first machine-facing output is the sync result:

```json
{
  "account": "fixtures",
  "mailbox": "INBOX",
  "listed_count": 13,
  "fetched_count": 13,
  "indexed_count": 13,
  "skipped_count": 0,
  "refreshed_count": 0,
  "index_path": "/tmp/mailcli-fixtures-index.json"
}
```

What the agent learns here:

- which account and mailbox were indexed
- whether local state is fresh enough to trust
- whether a second pass is needed

## Step 2: Rank Threads

Next the agent looks at compact thread summaries:

```json
{
  "thread_id": "<invoice-123@example.com>",
  "subject": "Your April invoice is ready",
  "last_message_id": "invoice.eml",
  "action_types": ["pay_invoice", "view_invoice"],
  "score": 85
}
```

Why this matters:

- the agent can choose a relevant conversation without loading the full message body first
- `action_types` and `score` reduce prompt-side triage work

## Step 3: Load The Thread

Then the agent loads the chosen thread:

```json
{
  "id": "invoice.eml",
  "thread_id": "<invoice-123@example.com>",
  "message": {
    "meta": {
      "from": { "name": "Billing Team", "address": "billing@example.com" },
      "subject": "Your April invoice is ready",
      "message_id": "<invoice-123@example.com>"
    },
    "content": {
      "body_md": "## Your invoice is ready\n\nInvoice #123 is now available in your billing portal.\n\n[View invoice](https://billing.example.com/invoices/123)\n\n[Pay invoice](https://billing.example.com/pay/123)"
    },
    "actions": [
      { "type": "view_invoice", "url": "https://billing.example.com/invoices/123" },
      { "type": "pay_invoice", "url": "https://billing.example.com/pay/123" }
    ]
  }
}
```

What the agent learns here:

- who sent the message
- what the normalized body looks like
- which reply or follow-up actions are available

## Step 4: Agent Decision And Reply Draft

At this point the agent can produce a stable `ReplyDraft` boundary:

```json
{
  "from": { "address": "support@nono.im" },
  "to": [{ "address": "billing@example.com", "name": "Billing Team" }],
  "body_text": "Thanks, we have received the invoice notification.",
  "account": "fixtures",
  "reply_to_id": "invoice.eml"
}
```

Why `reply_to_id` matters:

- the agent does not need to reconstruct `In-Reply-To`
- the agent does not need to rebuild `References`
- MailCLI can re-fetch the original message and derive thread headers itself

## Step 5: Compiled MIME

`mailcli reply --dry-run` turns that draft into transport-ready MIME:

```text
From: support@nono.im
To: Billing Team <billing@example.com>
Subject: Re: Your April invoice is ready
Message-ID: <generated@mailcli.local>
In-Reply-To: <invoice-123@example.com>
References: <invoice-123@example.com>
Date: Fri, 27 Mar 2026 15:11:13 +0000
MIME-Version: 1.0
Content-Type: text/plain; charset=UTF-8

Thanks, we have received the invoice notification.
```

The exact `Message-ID` and `Date` are generated at runtime.

The stable point is that the agent never had to author raw MIME itself.

## Full Report

If you want to inspect the whole loop in one object, see:

- [agent-report.json](../../../examples/artifacts/local-thread-demo/agent-report.json)

That file mirrors the output of `examples/python/agent_thread_assistant.py` for this local fixture flow.

## Related

- [Agent Thread Assistant](agent-thread-assistant.md)
- [Examples Index](README.md)
- [Agent Workflows](../agent-workflows.md)
