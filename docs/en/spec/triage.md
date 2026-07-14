[中文](../../zh-CN/spec/triage.md) | English

# Triage Evidence And Enrichment Spec

## Purpose

MailCLI separates facts it can derive deterministically from judgments that
require heuristics or an external AI provider.

This boundary prevents fields such as `priority` and `needs_reply` from looking
like parser facts. The Go core does not call an LLM and does not infer those
judgments itself.

## Commands

```bash
mailcli triage message message.eml
mailcli triage thread --index ~/.cache/mailcli/index.db "<thread-id>"
```

Both commands return a versioned `TriageRecord` with deterministic `evidence`.
The thread command loads the complete matching local thread; it does not apply
the default 50-message display limit used by `mailcli thread`.

Each record also includes an `evidence_id` SHA-256 value. External enrichment
must echo this value so a result generated before a thread changed cannot be
silently attached to newer evidence.

Use `--account` when the same thread id exists in multiple configured accounts.
MailCLI rejects a mixed-account triage record instead of silently combining it.

## Deterministic Evidence

Evidence includes:

- message count, ordered message ids, participants, categories, and labels
- latest date, last message id, and last sender
- action, code, attachment, auto-submitted, and error counts
- one compact `messages` fact per message with sender, recipients, subject,
  date, snippet, and per-message counts

The per-message list is intentional. A request can appear early in a thread and
remain unresolved after several later replies. Only looking at the latest
message is not enough to support a reliable `needs_reply` judgment.

Evidence is compact context, not a replacement for the full conversation. An
external provider deciding `needs_reply`, deadlines, or todos should also read
the complete `mailcli thread` result when body context matters.

## External Enrichment

An external provider may return a `TriageEnrichment` JSON document:

```json
{
  "version": "v1",
  "scope": "thread",
  "subject_id": "<root@example.com>",
  "evidence_id": "sha256:...",
  "source": {
    "kind": "external",
    "provider": "example-provider",
    "model": "example-model"
  },
  "generated_at": "2026-07-14T09:00:00Z",
  "summary": "A customer is waiting for a revised quote.",
  "priority": {
    "level": "high",
    "confidence": 0.9,
    "reasons": ["The requested deadline is tomorrow."]
  },
  "needs_reply": {
    "value": true,
    "confidence": 0.85,
    "reasons": ["The first message contains an unanswered request."]
  },
  "todos": [
    {
      "text": "Send the revised quote",
      "source_message_id": "msg-root",
      "confidence": 0.9
    }
  ]
}
```

Merge and validate the enrichment through the same command:

```bash
mailcli triage thread \
  --index ~/.cache/mailcli/index.db \
  --enrichment enrichment.json \
  "<root@example.com>"
```

`--enrichment -` reads enrichment JSON from stdin. For `triage message`, the
message and enrichment cannot both use stdin.

## Validation Rules

MailCLI rejects enrichment when:

- the contract version, scope, subject id, or evidence id does not match the current evidence
- source kind is not explicitly `external` or `heuristic`
- provider or RFC3339 `generated_at` metadata is missing
- priority is outside `low`, `normal`, `high`, or `urgent`
- confidence is outside 0 through 1
- priority or needs-reply assessments omit their reasons
- a todo or deadline does not identify a message id present in the evidence
- the JSON contains unknown fields or more than one top-level value

The merge does not persist enrichment in the local index. The caller owns the
result and may store it in its own workflow state.

## Testing And Evaluation

Snapshot tests protect the deterministic evidence shape and enrichment
validation contract. They do not prove that an AI provider makes good triage
judgments.

Provider evaluation should use a separately labeled thread set covering at
least:

- a request buried several messages earlier
- the latest message sent by the user versus by the other participant
- automated receipts and no-reply senders
- reopened conversations after an apparent resolution
- quoted requests that should not be treated as new requests
- explicit and ambiguous deadline language

Measure `priority`, `needs_reply`, deadlines, and todos separately. Do not treat
parser fixtures staying green as evidence that enrichment quality is good.

## Safety And Privacy

- Triage evidence does not contain attachment bytes.
- Attachment filenames remain untrusted metadata.
- MailCLI sends nothing to an external provider by itself.
- The caller chooses whether to provide compact evidence, full messages, or
  both to an external process.

The safe MCP surface exposes deterministic `mailcli_triage_message` and
`mailcli_triage_thread` tools. Enrichment generation remains outside the core.
