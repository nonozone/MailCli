[中文](../../zh-CN/contributing/drivers.md) | English

# Adding a Driver

This guide explains how contributors should add a new MailCLI driver.

## Before Writing Code

Read:

- [Driver Extension Spec](../spec/driver-extension.md)
- [Config Spec](../spec/config.md)
- `pkg/driver/driver.go`
- `pkg/driver/factory.go`

Open an issue first if your driver requires changes to:

- `Driver`
- `AccountConfig`
- `SearchQuery`
- `MessageMetaSummary`

## Minimum Implementation Steps

1. Add a new driver implementation under `pkg/driver/`
2. Make it satisfy the `Driver` interface
3. Wire it into `NewFromAccount`
4. Add tests for listing, raw fetch, and send behavior where applicable
5. Document required config fields and limitations

## Design Rules

- keep transport logic in the driver
- return raw message bytes from `FetchRaw`
- never return partially parsed message content from the driver
- do not duplicate parser logic in the driver
- do not hardcode provider-specific business interpretation into shared layers

## Testing Expectations

At minimum, add tests for:

- config validation failures
- list behavior
- fetch behavior
- send behavior or explicit send-not-supported behavior

## Factory Integration

Current drivers are registered in:

- `pkg/driver/factory.go`

If you add a new driver, make the dispatch explicit and readable.

## Documentation Expectations

If you add a driver, update:

- user-facing config guidance
- any relevant README examples
- driver-specific limitations

The goal is that another contributor can understand how your driver works without reading all of its internals.
