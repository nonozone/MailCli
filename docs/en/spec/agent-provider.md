[中文](../../zh-CN/spec/agent-provider.md) | English

# Agent Provider Contract

## Purpose

The Python inbox and thread agent examples support a pluggable external provider mode.

This keeps the repository free of LLM SDK dependencies while still giving developers a stable handoff point for OpenAI, Claude, local models, or custom agent runtimes.

## Entry Point

Example invocation:

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/plaintext.eml \
  --from-address support@nono.im \
  --agent-provider external \
  --provider-command python3 \
  --provider-arg ./my_provider.py
```

The external provider is executed as a subprocess.

The same contract is also used by:

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

## Input Contract

The provider receives one JSON object on stdin.

Single-message example payload:

```json
{
  "source": {
    "mode": "email",
    "value": "testdata/emails/plaintext.eml"
  },
  "message": {
    "id": "<plain-123@example.com>",
    "meta": {},
    "content": {},
    "actions": [],
    "codes": []
  },
  "wants_reply": false
}
```

Fields:

- `source`: where the message came from
- `message`: parsed `StandardMessage`
- `wants_reply`: whether the caller explicitly requested a reply body

Thread-aware example payload:

```json
{
  "source": {
    "mode": "local_thread",
    "index": "/tmp/mailcli-index.json",
    "thread_id": "<root@example.com>"
  },
  "selection": {
    "thread_id": "<root@example.com>",
    "last_message_id": "imap:uid:123"
  },
  "thread_summaries": [],
  "thread_messages": [],
  "latest_message": {
    "id": "imap:uid:123",
    "message": {}
  },
  "wants_reply": false
}
```

Additional thread fields:

- `selection`: chosen local thread summary
- `thread_summaries`: ranked thread candidates returned by `mailcli threads`
- `thread_messages`: full local thread records returned by `mailcli thread`
- `latest_message`: the resolved latest local message that the example will treat as the reply target

## Output Contract

The provider must return a JSON object on stdout.

Minimum valid output:

```json
{
  "decision": "review",
  "summary": "Short analysis"
}
```

Reply-capable output:

```json
{
  "decision": "draft_reply",
  "summary": "Short analysis",
  "reply_text": "Thanks for your email."
}
```

## Required Fields

- `decision`: non-empty string

The value must be one of the stable decisions defined in:

- [Agent Decision Spec](agent-decisions.md)

## Optional Fields

- `summary`: short text summary
- `reply_text`: reply body text; if present it must be a string

## Runtime Validation

The example currently validates:

- the provider returned a JSON object
- `decision` exists and is a non-empty string
- `reply_text`, when present, is a string

If validation fails, the example exits with a contract error instead of producing partial output.

## Decision Vocabulary

The example now validates decisions against the shared stable decision set:

- [Agent Decision Spec](agent-decisions.md)
