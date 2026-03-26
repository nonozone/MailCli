[中文](../../zh-CN/examples/agent-inbox-assistant.md) | English

# Agent Inbox Assistant

This example shows a minimal agent loop built on top of `mailcli`.

It is intentionally simple:

- no LLM SDK dependency
- no framework dependency
- all email I/O goes through `mailcli`

The example script is:

- `examples/python/agent_inbox_assistant.py`

## What It Demonstrates

1. Read a message through `mailcli parse` or `mailcli get`
2. Reason over structured `StandardMessage` output
3. Capture high-value artifacts such as `codes` and `actions`
4. Optionally generate a `ReplyDraft`
5. Compile that reply with `mailcli reply --dry-run`

## Local `.eml` Example

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/verification.eml
```

## Reply Dry-Run Example

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/plaintext.eml \
  --from-address support@nono.im \
  --reply-text "Thanks for your email."
```

This adds:

- `reply.draft`
- `reply.mime`

to the JSON output.

## Configured Inbox Example

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --config ~/.config/mailcli/config.yaml \
  --account work \
  --message-id "<message-id>"
```

## Notes

- The example uses rule-based analysis so the control flow is easy to inspect.
- Replace the `analyze_message` function with your own LLM or agent framework.
- Keep `mailcli` as the boundary for parsing, reply compilation, and protocol access.
