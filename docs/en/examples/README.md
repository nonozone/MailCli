[中文](../../zh-CN/examples/README.md) | English

# Examples

This page is the entry point for runnable MailCLI examples.

Use it when you want to quickly decide which example matches your workflow.

Maintainers can verify the checked-in local thread demo artifacts with:

```bash
make demo-local-thread-check
```

## Start Here

- For the fastest zero-network setup, use `examples/config/fixtures-dir.yaml` with the built-in `dir` driver.
- [Agent Inbox Assistant](agent-inbox-assistant.md)
  Best for single-message parsing, classification, and reply dry-runs.
- [Agent Thread Assistant](agent-thread-assistant.md)
  Best for local index, thread selection, thread context, and reply workflows.
- [Local Thread Demo](local-thread-demo.md)
  Best for understanding the exact JSON and MIME boundaries of the local thread loop.
- [Agent Decision Patterns](agent-decision-patterns.md)
  Best for verification, bounce, and unsubscribe decision examples.
- [Outbound Draft Patterns](outbound-draft-patterns.md)
  Best for concrete `ReplyDraft` and `DraftMessage` handoff patterns.
- [OpenAI External Provider](openai-external-provider.md)
  Best for plugging an OpenAI-backed analyzer into either agent example.

## Python Scripts

### Agent Workflows

- `examples/python/agent_inbox_assistant.py`
  Minimal single-message agent boundary.
- `examples/python/agent_thread_assistant.py`
  Thread-aware local retrieval and reply boundary.

### Raw Utilities

- `examples/python/parse_email.py`
  Parse one local `.eml` file through `mailcli parse`.
- `examples/python/reply_dry_run.py`
  Compile one `ReplyDraft` JSON file through `mailcli reply --dry-run`.
- `examples/python/refresh_local_thread_demo.py`
  Regenerate the stored local-thread-demo artifacts from the current fixture corpus.

### Provider Adapters

- `examples/providers/template_external_provider.py`
  Minimal external provider template for both inbox and thread payloads.
- `examples/providers/openai_external_provider.py`
  Optional OpenAI-backed provider using the Responses API.

## Shell Scripts

- `examples/shell/parse_email.sh`
- `examples/shell/reply_dry_run.sh`

## Suggested Reading Order

1. Start with [Agent Inbox Assistant](agent-inbox-assistant.md) if you want the smallest possible agent loop.
2. Move to [Agent Thread Assistant](agent-thread-assistant.md) when you need local memory and thread context.
3. Read [Outbound Draft Patterns](outbound-draft-patterns.md) when you need stable reply and new-message JSON boundaries.
4. Add [OpenAI External Provider](openai-external-provider.md) when you want to replace rule-based analysis with a model-backed subprocess.

## Related

- [Agent Workflows](../agent-workflows.md)
- [Agent Provider Contract](../spec/agent-provider.md)
- [Outbound Message Spec](../spec/outbound-message.md)
