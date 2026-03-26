[中文](../../zh-CN/spec/agent-provider.md) | English

# Agent Provider Contract

## Purpose

The Python inbox agent example supports a pluggable external provider mode.

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

## Input Contract

The provider receives one JSON object on stdin:

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
