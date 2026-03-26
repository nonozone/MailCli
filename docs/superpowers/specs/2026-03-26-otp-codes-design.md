# OTP Codes Design

## Goal

Add a small, agent-usable inbound schema for one-time passcodes and verification codes extracted from email bodies.

## Why This Needs A Schema Change

Current inbound structures do not have a good place for non-link security artifacts:

- `actions` are URL-oriented
- `labels` are too weak for machine use
- `content.body_md` is readable but forces every agent to re-parse the same code patterns

Verification codes are high-value data for agents. They should be exposed directly in a stable machine-readable field.

## Options Considered

### Option 1: Reuse `Action`

Example idea:

```json
{
  "type": "copy_code",
  "label": "Verification code",
  "url": "123456"
}
```

Why not:

- `url` would stop meaning URL
- action semantics become inconsistent
- downstream callers would need special cases

### Option 2: Add Generic `entities`

Example idea:

```json
{
  "entities": [
    { "kind": "verification_code", "value": "123456" }
  ]
}
```

Why not now:

- too broad for current scope
- introduces a long-lived generic abstraction without enough examples
- increases public schema complexity too early

### Option 3: Add Narrow `codes` Field

Example idea:

```json
{
  "codes": [
    {
      "type": "verification_code",
      "value": "123456",
      "label": "Verification code"
    }
  ]
}
```

Why this is the recommendation:

- explicit and easy for agents to consume
- minimal public schema change
- leaves room for future code-like artifacts without inventing a broad entity model
- easy to test and document

## Recommended Design

Add a new top-level field to `schema.StandardMessage`:

```go
Codes []Code `json:"codes,omitempty" yaml:"codes,omitempty"`
```

With:

```go
type Code struct {
    Type  string `json:"type" yaml:"type"`
    Value string `json:"value" yaml:"value"`
    Label string `json:"label,omitempty" yaml:"label,omitempty"`
}
```

## Extraction Rules

The first supported code type is:

- `verification_code`

The parser should extract only when both are true:

1. The surrounding text strongly suggests verification or login confirmation
2. The candidate code looks like a real OTP

Initial detection should be conservative:

- prefer 4-8 digit codes
- allow grouped forms like `123 456`
- only accept codes near phrases such as:
  - `verification code`
  - `security code`
  - `one-time code`
  - `login code`
  - `sign-in code`
  - `two-factor code`
  - `2fa code`

Initial exclusions:

- generic order numbers
- invoice numbers
- years and dates
- arbitrary long numbers without a security phrase nearby

## Parser Boundaries

- extraction should operate on normalized text, not raw MIME
- output should be deduplicated by `type + value`
- codes should not replace actions; both can coexist
- bounce/error extraction remains separate

## Testing Strategy

Add:

- a direct helper test for positive extraction
- direct helper tests for negative extraction
- a full parser fixture covering a realistic verification email
- golden JSON asserting `codes`

## Documentation Impact

Update:

- `README.md`
- `README.zh-CN.md`
- `docs/en/agent-workflows.md`
- `docs/zh-CN/agent-workflows.md`

to describe `codes` as part of agent-usable inbound output.
