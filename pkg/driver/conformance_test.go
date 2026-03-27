package driver

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/pkg/schema"
)

type driverContractHarness struct {
	newDriver        func(t *testing.T) Driver
	newMissingDriver func(t *testing.T) Driver
	newSendDriver    func(t *testing.T) Driver
	listQuery        schema.SearchQuery
	missingFetchID   string
	sendRaw          []byte
	expectedSendErr  error
	assertList       func(t *testing.T, got []schema.MessageMetaSummary)
	assertFetchRaw   func(t *testing.T, listed schema.MessageMetaSummary, raw []byte)
	assertAfterSend  func(t *testing.T, drv Driver)
}

func runDriverContractTests(t *testing.T, harness driverContractHarness) {
	t.Helper()

	t.Run("list", func(t *testing.T) {
		drv := harness.newDriver(t)
		query := harness.listQuery
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
		if harness.assertList != nil {
			harness.assertList(t, got)
		}
	})

	t.Run("fetch listed id", func(t *testing.T) {
		drv := harness.newDriver(t)
		query := harness.listQuery
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
		if harness.assertFetchRaw != nil {
			harness.assertFetchRaw(t, listed[0], raw)
		}
	})

	if strings.TrimSpace(harness.missingFetchID) != "" {
		t.Run("missing fetch id", func(t *testing.T) {
			drv := harness.newDriver(t)
			if harness.newMissingDriver != nil {
				drv = harness.newMissingDriver(t)
			}

			_, err := drv.FetchRaw(context.Background(), harness.missingFetchID)
			if !errors.Is(err, ErrMessageNotFound) {
				t.Fatalf("expected ErrMessageNotFound for missing id %q, got %v", harness.missingFetchID, err)
			}
		})
	}

	if len(harness.sendRaw) > 0 || harness.expectedSendErr != nil {
		t.Run("send raw", func(t *testing.T) {
			drv := harness.newDriver(t)
			if harness.newSendDriver != nil {
				drv = harness.newSendDriver(t)
			}
			err := drv.SendRaw(context.Background(), harness.sendRaw)

			switch {
			case harness.expectedSendErr != nil:
				if !errors.Is(err, harness.expectedSendErr) {
					t.Fatalf("expected send error %v, got %v", harness.expectedSendErr, err)
				}
			case err != nil:
				t.Fatalf("expected SendRaw to succeed: %v", err)
			}

			if harness.assertAfterSend != nil {
				harness.assertAfterSend(t, drv)
			}
		})
	}
}
