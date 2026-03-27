[中文](../../zh-CN/examples/agent-thread-assistant.md) | English

# Agent Thread Assistant

This example shows a fuller agent workflow built on top of `mailcli` local indexing and thread navigation.

It is meant for developers who want to model the common agent loop:

1. sync recent inbox state
2. choose a relevant thread
3. inspect the full local thread
4. draft a reply through `ReplyDraft`
5. compile the reply with `mailcli reply --dry-run`

The example script is:

- `examples/python/agent_thread_assistant.py`

## What It Demonstrates

1. Running `mailcli sync` to refresh a local index
2. Using `mailcli threads` to rank thread candidates
3. Using `mailcli thread` to load the selected local conversation
4. Choosing the latest local message as the reply target
5. Generating a `ReplyDraft` instead of raw MIME
6. Compiling the draft with `mailcli reply --dry-run`

## Inbox-Backed Example

```bash
python3 examples/python/agent_thread_assistant.py \
  --mailcli-bin ./mailcli \
  --config ~/.config/mailcli/config.yaml \
  --account work \
  --index /tmp/mailcli-index.json \
  --query invoice \
  --from-address support@nono.im \
  --reply-text "Thanks for your email."
```

## Existing Local Index Example

```bash
python3 examples/python/agent_thread_assistant.py \
  --mailcli-bin ./mailcli \
  --index /tmp/mailcli-index.json \
  --skip-sync \
  --thread-id "<root@example.com>" \
  --from-address support@nono.im \
  --reply-text "Thanks for your email."
```

## External Provider Mode

You can also delegate the thread analysis step to your own script or agent runtime:

```bash
python3 examples/python/agent_thread_assistant.py \
  --mailcli-bin ./mailcli \
  --index /tmp/mailcli-index.json \
  --skip-sync \
  --thread-id "<root@example.com>" \
  --from-address support@nono.im \
  --agent-provider external \
  --provider-command python3 \
  --provider-arg ./my_provider.py
```

The external provider receives a JSON payload that includes:

- `selection`
- `thread_summaries`
- `thread_messages`
- `latest_message`
- `wants_reply`

If it returns:

```json
{
  "decision": "draft_reply",
  "summary": "Short analysis",
  "reply_text": "Thanks for your email."
}
```

the example will compile a reply dry-run automatically.

## Reply Draft Strategy

When `--config` is present, the example prefers `reply_to_id` using the latest local message id. That lets `mailcli reply` re-fetch the original raw message and derive:

- `In-Reply-To`
- `References`
- default reply subject

When `--skip-sync` is used without `--config`, the example falls back to a local-only draft using:

- `reply_to_message_id`
- local `references`
- local subject

This keeps the example usable even when developers only have an existing local index snapshot.

## Output Shape

The script prints one JSON report that includes:

- `sync`
- `selection`
- `thread_summaries`
- `thread_messages`
- `latest_message`
- `analysis`
- optional `reply`

## Why This Example Exists

The minimal inbox example is useful for single-message analysis.

This thread example is for the more realistic agent case where the model needs local memory, thread-level context, and a stable reply boundary.

Related docs:

- [Agent Workflows](../agent-workflows.md)
- [Outbound Message Spec](../spec/outbound-message.md)
- [Agent Provider Contract](../spec/agent-provider.md)
- [Agent Inbox Assistant](agent-inbox-assistant.md)
