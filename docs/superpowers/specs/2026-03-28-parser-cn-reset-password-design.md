# Parser Chinese Reset Password Design

## Goal

Add one bounded parser-quality improvement for multilingual security mail:
recognize Chinese reset-password links from anchor labels even when the URL does
not expose the current English password-reset heuristics.

## Why This Slice

The parser already extracts `reset_password` from:

- English labels such as `Reset password`
- English URL patterns such as `/reset-password/...`

That leaves a gap for Chinese security mail where the meaningful signal is the
anchor text rather than the URL.

This slice stays intentionally small:

- one new password-reset fixture
- one fixture-backed parser test
- two positive classifier tests
- two negative classifier tests
- no schema changes

## Scope

In scope:

- add one Chinese password-reset fixture and golden output
- ensure the fixture uses a neutral URL that does not match the existing English password-reset href heuristics
- add bounded Chinese label detection for `reset_password`
- add focused positive and negative classifier tests
- refresh repository demo artifacts if fixture-count-based examples drift

Out of scope:

- code extraction changes
- verify-sign-in changes
- broader multilingual security semantics
- new action types

## Options Considered

### Option 1: Href-only reset-password classification

Keep `reset_password` tied to English URL structure.

This misses multilingual security mail where the href is generic.

### Option 2: Broad Chinese password vocabulary

Match many Chinese password and account-recovery phrases.

This increases false-positive risk for generic account settings or security pages.

### Option 3: Narrow reset-password phrases

Match only explicit password-reset phrases such as:

- `重置密码`
- `修改密码`

Recommendation: option 3.

## Design

Keep the action extraction pipeline unchanged and extend only the
`reset_password` classifier branch.

The classifier should continue to accept current English label and href behavior.
It should additionally accept a short Chinese phrase list in the raw anchor label.

The fixture must use a neutral URL like `https://example.cn/security/password/...`
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

- false positives from generic password or account settings labels
- fixture-count changes causing local demo artifact drift

Mitigation:

- only match explicit reset-password phrases
- include a generic account-password negative test
- include a security-settings negative test
- use the standard demo refresh/check workflow after verification

