# Parser Chinese Confirm Subscription Design

## Goal

Add one bounded parser-quality improvement for multilingual newsletter flows:
recognize Chinese confirm-subscription links in HTML mail when the URL itself does
not expose an English `confirm-subscription` pattern.

## Why This Slice

The parser already extracts `confirm_subscription` from:

- English anchor text such as `Confirm subscription`
- English URL patterns such as `/confirm-subscription/...`

That leaves a gap for Chinese opt-in mail where the only reliable signal is anchor
text such as `确认订阅`.

This slice is intentionally small:

- one new fixture
- one fixture-backed parser test
- a neutral URL in the end-to-end fixture so the signal really comes from the Chinese label
- two positive classifier tests
- two negative classifier tests
- no schema changes

## Scope

In scope:

- add one Chinese confirm-subscription fixture and golden output
- ensure the fixture uses a neutral URL that does not match the existing English href heuristics
- add bounded Chinese confirm-subscription keyword detection
- add focused positive and negative classifier-level tests for the allowlist
- refresh repository demo artifacts if fixture-count-based examples drift

Out of scope:

- broader multilingual action extraction
- locale-aware generated labels
- provider-specific subscription logic

## Options Considered

### Option 1: URL-only classification

Keep `confirm_subscription` English-URL-driven only.

This misses real multilingual mail and does not improve the parser.

### Option 2: Broad Chinese confirmation vocabulary

Match many Chinese confirmation phrases.

This risks false positives such as order confirmation or generic account approval.

### Option 3: Narrow subscription-confirmation phrases

Match only explicit subscription-confirmation phrases such as:

- `确认订阅`
- `确认邮件订阅`

Recommendation: option 3.

## Design

Keep the current action extraction pipeline unchanged and extend only the
`confirm_subscription` classifier branch.

The classifier should continue to accept existing English label and URL patterns.
It should additionally accept a very short Chinese phrase list in the raw anchor
label, while avoiding generic confirmation language like `确认订单`.

The fixture must use a neutral URL like `https://example.cn/welcome?...` so the
end-to-end test proves the Chinese-label path rather than reusing English href
heuristics.

## Test Strategy

Use TDD:

1. add the fixture-backed parser test
2. verify the parser fails with a golden mismatch because the action is missing
3. add positive and negative classifier tests, including coverage for both allowed Chinese phrases
4. implement the smallest classification change
5. run targeted parser tests
6. run full repository verification and refresh demo artifacts if needed

## Risks

- false positives from generic Chinese confirmation text
- fixture-count changes causing local demo artifact drift

Mitigation:

- only match explicit subscription phrases
- include an order-confirmation negative test
- include an email-address confirmation negative test
- use the standard demo refresh/check workflow after verification
