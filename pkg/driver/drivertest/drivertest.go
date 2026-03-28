package drivertest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/pkg/schema"
)

// Driver is the minimum interface required by the shared conformance harness.
type Driver interface {
	List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error)
	FetchRaw(ctx context.Context, id string) ([]byte, error)
	SendRaw(ctx context.Context, raw []byte) error
}

// Harness defines the shared baseline contract that MailCLI drivers should satisfy.
type Harness struct {
	NewDriver        func(t *testing.T) Driver
	NewMissingDriver func(t *testing.T) Driver
	NewSendDriver    func(t *testing.T) Driver
	ListQuery        schema.SearchQuery
	MissingFetchID   string
	NotFoundError    error
	SendRaw          []byte
	ExpectedSendErr  error
	AssertList       func(t *testing.T, got []schema.MessageMetaSummary)
	AssertFetchRaw   func(t *testing.T, listed schema.MessageMetaSummary, raw []byte)
	AssertAfterSend  func(t *testing.T, drv Driver)
}

// RunContractSuite executes the minimum shared list/fetch/send contract checks for a driver.
func RunContractSuite(t *testing.T, harness Harness) {
	t.Helper()

	if harness.NewDriver == nil {
		t.Fatalf("drivertest Harness requires NewDriver")
	}

	t.Run("list", func(t *testing.T) {
		drv := harness.NewDriver(t)
		query := harness.ListQuery
		if query.Limit == 0 {
			query.Limit = 1
		}

		got, err := drv.List(context.Background(), query)
		if err != nil {
			t.Fatalf("expected List to succeed: %v", err)
		}
		if len(got) == 0 {
			t.Fatalf("expected List to return at least one message")
		}
		if harness.AssertList != nil {
			harness.AssertList(t, got)
		}
	})

	t.Run("fetch listed id", func(t *testing.T) {
		drv := harness.NewDriver(t)
		query := harness.ListQuery
		if query.Limit == 0 {
			query.Limit = 1
		}

		listed, err := drv.List(context.Background(), query)
		if err != nil {
			t.Fatalf("expected List to succeed before FetchRaw: %v", err)
		}
		if len(listed) == 0 || strings.TrimSpace(listed[0].ID) == "" {
			t.Fatalf("expected listed message with fetchable id, got %+v", listed)
		}

		raw, err := drv.FetchRaw(context.Background(), listed[0].ID)
		if err != nil {
			t.Fatalf("expected FetchRaw to succeed for listed id %q: %v", listed[0].ID, err)
		}
		if len(raw) == 0 {
			t.Fatalf("expected non-empty raw message bytes")
		}
		if harness.AssertFetchRaw != nil {
			harness.AssertFetchRaw(t, listed[0], raw)
		}
	})

	if strings.TrimSpace(harness.MissingFetchID) != "" {
		t.Run("missing fetch id", func(t *testing.T) {
			if harness.NotFoundError == nil {
				t.Fatalf("drivertest Harness requires NotFoundError when MissingFetchID is set")
			}

			drv := harness.NewDriver(t)
			if harness.NewMissingDriver != nil {
				drv = harness.NewMissingDriver(t)
			}

			_, err := drv.FetchRaw(context.Background(), harness.MissingFetchID)
			if !errors.Is(err, harness.NotFoundError) {
				t.Fatalf("expected ErrMessageNotFound for missing id %q, got %v", harness.MissingFetchID, err)
			}
		})
	}

	if len(harness.SendRaw) > 0 || harness.ExpectedSendErr != nil {
		t.Run("send raw", func(t *testing.T) {
			drv := harness.NewDriver(t)
			if harness.NewSendDriver != nil {
				drv = harness.NewSendDriver(t)
			}
			err := drv.SendRaw(context.Background(), harness.SendRaw)

			switch {
			case harness.ExpectedSendErr != nil:
				if !errors.Is(err, harness.ExpectedSendErr) {
					t.Fatalf("expected send error %v, got %v", harness.ExpectedSendErr, err)
				}
			case err != nil:
				t.Fatalf("expected SendRaw to succeed: %v", err)
			}

			if harness.AssertAfterSend != nil {
				harness.AssertAfterSend(t, drv)
			}
		})
	}
}
