[中文](../../zh-CN/spec/driver-extension.md) | English

# Driver Extension Spec

## Purpose

MailCLI keeps transport behind the `driver` package.

Drivers are responsible for talking to mailbox protocols or provider APIs. They are not responsible for MIME cleanup, semantic extraction, or outbound draft generation.

## Core Interface

Current interface:

```go
type Driver interface {
    List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error)
    FetchRaw(ctx context.Context, id string) ([]byte, error)
    SendRaw(ctx context.Context, raw []byte) error
}
```

## Responsibilities

A driver may:

- list mailbox messages
- fetch raw RFC 822 style bytes
- send raw outbound MIME bytes
- map provider transport failures into stable operational errors where practical

A driver must not:

- parse message bodies into `StandardMessage`
- clean HTML
- extract actions or codes
- generate MIME from `DraftMessage` or `ReplyDraft`
- add provider-specific business semantics into the parser layer

## Current Extension Model

For `v0.1 RC`, the extension model is **in-tree code-level extension**.

That means contributors add a new driver by:

1. implementing the `Driver` interface
2. wiring it into the driver factory
3. adding tests
4. documenting account config and limitations

Runtime-loaded plugins are not part of the current contract.

## Reference Implementations

Current in-tree drivers provide two different implementation shapes:

- `pkg/driver/imap.go`: real network transport with IMAP read and SMTP send
- `pkg/driver/stub.go`: deterministic local-only driver for development and extension examples

If you are contributing a new driver, start by reading `stub.go` for the smallest possible implementation, then compare `imap.go` for a stateful network-backed implementation.

## Stable Boundary

The stable driver boundary for now is:

- `pkg/driver.Driver`
- `internal/config.AccountConfig`
- `schema.SearchQuery`
- `schema.MessageMetaSummary`

This boundary may evolve before `v1.0`, but changes should be discussed first because they affect every provider implementation.

## Error Guidance

Drivers should prefer returning:

- `ErrMessageNotFound`
- `ErrTransportNotConfigured`
- `ErrDriverConfigInvalid`
- `ErrAuthFailed`

Other provider-specific errors may still bubble up, but stable typed errors are preferred when they improve caller behavior.

## Message Identity Guidance

Drivers should document which message identifiers they accept.

For example, the current IMAP driver supports:

- sequence numbers
- explicit UIDs
- `Message-ID` lookup

Provider-specific convenience forms are acceptable, but they should be documented and not silently change meaning.

## Config Guidance

Each driver should define:

- required account fields
- optional account fields
- transport assumptions
- sending limitations

If a driver requires unusual credentials or provider-specific setup, document that in the driver contributor guide rather than leaking it into unrelated parser or CLI code.
