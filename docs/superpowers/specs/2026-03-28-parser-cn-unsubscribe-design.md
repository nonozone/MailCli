# Parser Chinese Unsubscribe Design

## Goal

Add one bounded parser-quality improvement for multilingual agent workflows:
recognize Chinese unsubscribe links in HTML mail even when no `List-Unsubscribe`
header is present.

## Why This Slice

The current parser already extracts unsubscribe actions from:

- `List-Unsubscribe` headers
- English unsubscribe anchor text

That leaves a real gap for multilingual newsletter mail where the only machine-useful
action is a Chinese anchor such as `退订` or `取消订阅`.

This is a strong maintainer-led slice because:

- it improves agent usefulness directly
- it expands the corpus with a real failure mode
- it does not require any schema change
- it keeps the heuristic surface small and auditable

## Scope

In scope:

- add one Chinese newsletter fixture without a `List-Unsubscribe` header
- add a golden output proving that `unsubscribe` is extracted from anchor text
- add bounded Chinese unsubscribe keyword checks in parser action classification
- keep docs unchanged unless the final scope actually needs a status update

Out of scope:

- general multilingual action extraction
- locale-aware labels for default generated action names
- new action types
- model-driven action classification

## Recommended Approach

### Option 1: Fixture-only expansion

Add the fixture and accept current failure.

This increases visibility but does not improve the product.

### Option 2: Broad multilingual action vocabulary

Teach the parser many languages and action families at once.

This is too wide for the current phase and increases regression risk.

### Option 3: Bounded Chinese unsubscribe detection

Add one realistic fixture and a narrow set of Chinese unsubscribe phrases.

This delivers immediate value with the smallest rule surface.

Recommendation: option 3.

## Design

### Parser behavior

Keep the existing action pipeline:

1. normalize anchor label text
2. classify the action from label and URL
3. emit `schema.Action{Type: "unsubscribe"}`

Only extend the unsubscribe branch with a short set of Chinese phrases that are
unlikely to collide with unrelated actions:

- `退订`
- `取消订阅`
- `取消邮件订阅`

The implementation should continue to prefer explicit URL or anchor intent and
must not introduce fuzzy matching beyond simple substring checks.

### Test strategy

Use TDD:

1. add a fixture-backed parser test that fails under current behavior
2. verify the failure is a golden mismatch around missing unsubscribe classification
3. add focused positive and negative classifier tests
4. implement the smallest classification change
5. run the targeted parser tests
6. run full repository verification

### Docs

Docs are optional for this slice.

Only update maintainer-facing status text if the final change meaningfully expands
the public description of parser coverage.

## Risks

- overly broad Chinese keywords could create false positives
- fixture wording that is too synthetic could weaken its long-term value

Mitigation:

- keep the keyword set short
- use a realistic newsletter-like HTML mail body
- avoid classifying generic words like `取消`
