[中文](../../zh-CN/examples/openai-external-provider.md) | English

# OpenAI External Provider

This example shows how to plug an OpenAI-backed analyzer into the MailCLI external provider contract.

It is optional on purpose:

- the main repository does not depend on the OpenAI SDK
- the provider is a standalone subprocess example
- the contract still flows through stdin and stdout JSON

Example file:

- `examples/providers/openai_external_provider.py`

## Install

```bash
pip install openai
```

## Environment

```bash
export OPENAI_API_KEY=...
export OPENAI_MODEL=gpt-5-mini
```

`OPENAI_MODEL` is optional. The example defaults to `gpt-5-mini`.

## Run With The Agent Example

```bash
python3 examples/python/agent_inbox_assistant.py \
  --mailcli-bin ./mailcli \
  --email testdata/emails/plaintext.eml \
  --from-address support@nono.im \
  --agent-provider external \
  --provider-command python3 \
  --provider-arg examples/providers/openai_external_provider.py
```

The same provider can also be used with the thread-aware example:

```bash
python3 examples/python/agent_thread_assistant.py \
  --mailcli-bin ./mailcli \
  --index /tmp/mailcli-index.json \
  --skip-sync \
  --thread-id "<root@example.com>" \
  --from-address support@nono.im \
  --agent-provider external \
  --provider-command python3 \
  --provider-arg examples/providers/openai_external_provider.py
```

## What It Does

1. Reads the MailCLI provider payload from stdin
2. Calls the OpenAI Responses API
3. Requests structured output through JSON Schema
4. Returns a validated provider response on stdout

## Output Contract

The example returns the same shape required by the shared external provider contract:

```json
{
  "decision": "review",
  "summary": "Short analysis"
}
```

When a reply is appropriate, it can also return:

```json
{
  "decision": "draft_reply",
  "summary": "Short analysis",
  "reply_text": "Thanks for your email."
}
```

Related references:

- [Examples Index](README.md)
- [Agent Provider Contract](../spec/agent-provider.md)
- [Agent Decision Spec](../spec/agent-decisions.md)
- [Agent Inbox Example](agent-inbox-assistant.md)
- [Agent Thread Example](agent-thread-assistant.md)
