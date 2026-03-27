[中文](../../zh-CN/spec/driver-compliance.md) | English

# Driver Compliance Checklist

This page defines the minimum acceptance bar for MailCLI drivers.

It is intentionally narrower than full provider-specific correctness.

The goal is to make new drivers reviewable without relying on tribal knowledge.

## Contract Surface

A driver is expected to preserve the shared boundary defined by:

- `pkg/driver.Driver`
- `internal/config.AccountConfig`
- `schema.SearchQuery`
- `schema.MessageMetaSummary`

If a proposed driver requires changing that boundary, open an RFC issue first.

## Required Behaviors

Every driver should satisfy these baseline behaviors:

1. `List` returns stable, fetchable message ids.
2. `FetchRaw` returns raw RFC 822 style bytes, not partially parsed content.
3. `FetchRaw` returns `ErrMessageNotFound` for missing messages when practical.
4. `SendRaw` either succeeds or returns a stable operational error.
5. Driver code keeps transport concerns inside the driver layer.

## Required Error Shapes

Prefer these stable errors where they apply:

- `ErrMessageNotFound`
- `ErrTransportNotConfigured`
- `ErrDriverConfigInvalid`
- `ErrAuthFailed`

Provider-specific errors may still bubble up, but stable typed errors should be used when they improve caller behavior.

## Required Tests

At minimum, a new driver should include:

- config validation coverage
- list behavior coverage
- fetch behavior coverage
- send behavior coverage, or explicit send-not-supported coverage
- the shared contract suite in `pkg/driver/conformance_test.go`

The shared contract suite is the minimum bar, not the complete test plan.

## Recommended Additional Tests

Add provider-specific tests for:

- mailbox selection semantics
- supported message identifier forms
- auth failure mapping
- transport misconfiguration
- pagination or recency assumptions
- send envelope extraction edge cases

## Review Questions

When reviewing a driver PR, check:

- can another contributor understand how ids are supposed to work?
- does `FetchRaw` really return raw bytes?
- are missing messages and auth failures mapped clearly?
- is provider-specific parsing logic leaking into shared layers?
- are config requirements and limitations documented?

## References

- [Driver Extension Spec](driver-extension.md)
- [Adding a Driver](../contributing/drivers.md)
- `pkg/driver/conformance_test.go`
