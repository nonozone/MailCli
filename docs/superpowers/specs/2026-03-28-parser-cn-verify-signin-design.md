# Parser Chinese Verify Sign-In Design

## Goal

Add one bounded parser-quality improvement for multilingual security mail:
recognize Chinese verify-sign-in links from anchor labels even when the URL does
not expose the current English login-verification heuristics.

## Why This Slice

The parser already extracts `verify_sign_in` from:

- English labels such as `Verify sign-in`
- English URL patterns such as `/verify-sign-in/...`

That leaves a gap for Chinese security mail where the machine-meaningful signal
is the anchor text rather than the URL.

This slice stays intentionally small:

- one new security fixture
- one fixture-backed parser test
- two positive classifier tests
- two negative classifier tests
- no schema changes

## Scope

In scope:

- add one Chinese security fixture and golden output
- ensure the fixture uses a neutral URL that does not match the existing English verify-sign-in href heuristics
- add bounded Chinese label detection for `verify_sign_in`
- add focused positive and negative classifier tests
- refresh repository demo artifacts if fixture-count-based examples drift

Out of scope:

- code extraction changes
- password-reset changes
- broader multilingual security semantics
- new action types

## Options Considered

### Option 1: Href-only verification classification

Keep `verify_sign_in` tied to English URL structure.

This misses multilingual security mail where the href is generic.

### Option 2: Broad Chinese login vocabulary

Match many Chinese login and security phrases.

This increases false-positive risk for generic login or account links.

### Option 3: Narrow sign-in verification phrases

Match only explicit verification phrases such as:

- `验证登录`
- `确认登录`

Recommendation: option 3.

## Design

Keep the action extraction pipeline unchanged and extend only the
`verify_sign_in` classifier branch.

The classifier should continue to accept current English label and href behavior.
It should additionally accept a short Chinese phrase list in the raw anchor label.

The fixture must use a neutral URL like `https://example.cn/security/session/...`
so the end-to-end test proves the Chinese-label path rather than English href
patterns.

## Test Strategy

Use TDD:

1. add the fixture-backed parser test
2. verify the parser fails with a golden mismatch because the action is missing
3. add positive and negative classifier tests
4. implement the smallest classification change
5. run targeted parser tests
6. run full repository verification and refresh demo artifacts if needed

## Risks

- false positives from generic login or account-security labels
- fixture-count changes causing local demo artifact drift

Mitigation:

- only match explicit verification phrases
- include a generic login negative test
- include an account-security negative test
- use the standard demo refresh/check workflow after verification

