[中文](../../zh-CN/examples/agent-decision-patterns.md) | English

# Agent Decision Patterns

This page shows three concrete decision examples built on top of the current MailCLI agent contracts.

It is meant for developers who want to understand:

- what an agent provider receives
- which stable decision it should return
- which message fields actually drive that decision

## Stable Decisions In Scope

Current stable decisions are defined in:

- [Agent Decision Spec](../spec/agent-decisions.md)

This page focuses on three common cases:

1. verification mail
2. bounce / delivery failure mail
3. unsubscribe / subscription mail

## Stored Example Artifacts

- [verification-input.json](../../../examples/artifacts/agent-decisions/verification-input.json)
- [verification-output.json](../../../examples/artifacts/agent-decisions/verification-output.json)
- [bounce-input.json](../../../examples/artifacts/agent-decisions/bounce-input.json)
- [bounce-output.json](../../../examples/artifacts/agent-decisions/bounce-output.json)
- [unsubscribe-input.json](../../../examples/artifacts/agent-decisions/unsubscribe-input.json)
- [unsubscribe-output.json](../../../examples/artifacts/agent-decisions/unsubscribe-output.json)

## Pattern 1: Verification Mail

Decision:

```json
{
  "decision": "capture_code",
  "summary": "Verification email with 1 code(s). First code expires in 600 seconds."
}
```

Why:

- the parsed message already exposes `codes`
- the current agent contract treats verification codes as high-priority artifacts
- no reply draft is needed to make this message useful

Key input fields:

- `message.codes`
- optional `expires_in_seconds`

## Pattern 2: Bounce / Delivery Failure

Decision:

```json
{
  "decision": "escalate_delivery_error",
  "summary": "535 Invalid login: 535 Authentication credentials invalid"
}
```

Why:

- the parsed message exposes structured `error_context`
- this is operational state, not ordinary inbox content
- the agent should surface it to a human or higher-level workflow

Key input fields:

- `message.error_context`
- `message.labels`
- `message.content.category`

## Pattern 3: Unsubscribe / Subscription Mail

Decision:

```json
{
  "decision": "review",
  "summary": "Subscription email with 2 unsubscribe action(s)."
}
```

Why:

- the current stable decision set does not include a dedicated `unsubscribe` decision
- the right action is still machine-visible through `message.actions`
- `review` keeps the contract small while still letting automation inspect the available unsubscribe URLs

Key input fields:

- `message.actions`
- `message.meta.list_unsubscribe`

## Practical Guidance

- Use `capture_code` when the value itself is the important artifact.
- Use `escalate_delivery_error` when the message is reporting a system failure.
- Keep `review` for cases where the next step depends on downstream policy, even if the parser already exposed useful actions.

## Related

- [Agent Provider Contract](../spec/agent-provider.md)
- [Agent Decision Spec](../spec/agent-decisions.md)
- [Examples Index](README.md)
