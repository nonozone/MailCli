[中文](../../zh-CN/contributing/drivers.md) | English

# Adding a Driver

This guide explains how contributors should add a new MailCLI driver.

## Before Writing Code

Read:

- [Driver Extension Spec](../spec/driver-extension.md)
- [Driver Compliance Checklist](../spec/driver-compliance.md)
- [Config Spec](../spec/config.md)
- `pkg/driver/driver.go`
- `pkg/driver/drivertest/drivertest.go`
- `pkg/driver/factory.go`
- `pkg/driver/stub.go`
- `pkg/driver/dir.go`

Open an issue first if your driver requires changes to:

- `Driver`
- `AccountConfig`
- `SearchQuery`
- `MessageMetaSummary`

If the change would affect a shared contract, open the `RFC contract change` issue template instead of a normal feature request.

## Minimum Implementation Steps

1. Add a new driver implementation under `pkg/driver/`
2. Make it satisfy the `Driver` interface
3. Wire it into `NewFromAccount`
4. Add tests for listing, raw fetch, and send behavior where applicable
5. Document required config fields and limitations

Prefer wiring the new driver into the shared contract suite in `pkg/driver/drivertest` first, then add provider-specific edge-case tests around it.

## Shared Conformance Harness

MailCLI now exposes a reusable driver contract helper in:

- `pkg/driver/drivertest`

Use it to prove the shared baseline before writing provider-specific edge-case tests.

Minimal example:

```go
func TestExampleDriverContractSuite(t *testing.T) {
	drivertest.RunContractSuite(t, drivertest.Harness{
		NewDriver: func(t *testing.T) drivertest.Driver {
			t.Helper()
			return newExampleDriverForTest(t)
		},
		MissingFetchID: "missing-id",
		NotFoundError:  driver.ErrMessageNotFound,
		SendRaw:        []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Demo\r\n\r\nHello"),
	})
}
```

This helper is intentionally small.

It checks the shared bar only:

- `List` returns at least one fetchable id
- `FetchRaw` succeeds for a listed id
- missing ids map to a stable not-found error when configured
- `SendRaw` succeeds or returns the expected stable operational error

It does not replace provider-specific tests for auth mapping, mailbox semantics, pagination, or identifier forms.

## Design Rules

- keep transport logic in the driver
- return raw message bytes from `FetchRaw`
- never return partially parsed message content from the driver
- do not duplicate parser logic in the driver
- do not hardcode provider-specific business interpretation into shared layers
- prefer starting from the `stub` driver shape before adding network concerns

## Testing Expectations

At minimum, add tests for:

- config validation failures
- list behavior
- fetch behavior
- send behavior or explicit send-not-supported behavior
- shared contract coverage through `pkg/driver/drivertest`

## Factory Integration

Current drivers are registered in:

- `pkg/driver/factory.go`

If you add a new driver, make the dispatch explicit and readable.

## Documentation Expectations

If you add a driver, update:

- user-facing config guidance
- any relevant README examples
- driver-specific limitations
- the compliance checklist if the shared acceptance bar changes

The goal is that another contributor can understand how your driver works without reading all of its internals.
