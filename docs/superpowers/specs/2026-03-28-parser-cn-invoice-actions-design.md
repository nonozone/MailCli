# Parser Chinese Invoice Actions Design

## Goal

Add one bounded parser-quality improvement for multilingual billing mail:
recognize Chinese invoice-view and invoice-payment links from anchor labels even
when the URL does not expose the current English invoice heuristics.

## Why This Slice

The parser already extracts:

- `view_invoice` from English labels such as `View invoice`
- `pay_invoice` from English labels such as `Pay invoice`
- some invoice/payment cases from English-looking href structure

That still leaves a gap for Chinese billing mail where the machine-meaningful
signal is the anchor text rather than the URL.

This slice stays intentionally narrow:

- one new billing fixture
- one fixture-backed parser test
- two positive classifier tests
- two negative classifier tests
- no schema changes

## Scope

In scope:

- add one Chinese billing fixture and golden output
- ensure the fixture uses neutral URLs that do not match existing English invoice/pay heuristics
- add bounded Chinese label detection for `view_invoice`
- add bounded Chinese label detection for `pay_invoice`
- add focused positive and negative classifier tests
- refresh repository demo artifacts if fixture-count-based examples drift

Out of scope:

- broader multilingual payment semantics
- new billing action types
- amount extraction or invoice metadata extraction
- provider-specific billing logic

## Options Considered

### Option 1: Href-only heuristics

Keep invoice/payment classification tied to English URL structure.

This misses multilingual transactional mail where the href is generic.

### Option 2: Broad Chinese finance vocabulary

Match many billing and payment phrases.

This increases false-positive risk and makes the classifier harder to reason about.

### Option 3: Narrow invoice/payment labels

Match only explicit invoice/payment labels such as:

- `查看发票`
- `查看账单`
- `支付发票`
- `支付账单`

Recommendation: option 3.

## Design

Keep the action extraction pipeline unchanged and extend only the `view_invoice`
and `pay_invoice` classifier branches.

The classifier should continue to accept current English label and href behavior.
It should additionally accept a short Chinese phrase list in the raw anchor label.

The fixture must use neutral URLs like `https://example.cn/billing/document/...`
and `https://example.cn/billing/checkout/...` so the end-to-end test proves the
Chinese-label path rather than English href patterns.

## Test Strategy

Use TDD:

1. add the fixture-backed parser test
2. verify the parser fails with a golden mismatch because actions are missing
3. add positive and negative classifier tests
4. implement the smallest classification change
5. run targeted parser tests
6. run full repository verification and refresh demo artifacts if needed

## Risks

- false positives from generic Chinese payment or settings labels
- fixture-count changes causing local demo artifact drift

Mitigation:

- only match explicit invoice/payment phrases
- include a billing-settings negative test
- include a generic payment-center negative test
- use the standard demo refresh/check workflow after verification

