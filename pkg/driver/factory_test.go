package driver

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/driver/drivertest"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestNewFromAccountBuildsIMAPDriver(t *testing.T) {
	account := config.AccountConfig{
		Name:     "work",
		Driver:   "imap",
		Host:     "imap.example.com",
		Port:     993,
		Username: "user@example.com",
		Password: "secret",
		TLS:      true,
		Mailbox:  "INBOX",
	}

	drv, err := NewFromAccount(account)
	if err != nil {
		t.Fatalf("expected imap account to build driver: %v", err)
	}

	if drv == nil {
		t.Fatalf("expected driver instance")
	}
}

func TestNewFromAccountRejectsIncompleteIMAPConfig(t *testing.T) {
	account := config.AccountConfig{
		Name:   "work",
		Driver: "imap",
		Host:   "imap.example.com",
	}

	_, err := NewFromAccount(account)
	if err == nil {
		t.Fatalf("expected incomplete imap config to fail")
	}
}

func TestNewFromAccountBuildsStubDriver(t *testing.T) {
	account := config.AccountConfig{
		Name:   "demo",
		Driver: "stub",
	}

	drv, err := NewFromAccount(account)
	if err != nil {
		t.Fatalf("expected stub account to build driver: %v", err)
	}

	stub, ok := drv.(*stubDriver)
	if !ok {
		t.Fatalf("expected stub driver, got %T", drv)
	}

	if len(stub.messages) == 0 {
		t.Fatalf("expected stub driver to preload messages")
	}
}

func TestStubDriverListFetchAndSend(t *testing.T) {
	outbound := []byte("From: sender@example.com\r\nTo: receiver@example.com\r\nSubject: Demo\r\n\r\nHello")

	drivertest.RunContractSuite(t, drivertest.Harness{
		NewDriver: func(t *testing.T) drivertest.Driver {
			t.Helper()

			drv, err := NewFromAccount(config.AccountConfig{
				Name:   "demo",
				Driver: "stub",
			})
			if err != nil {
				t.Fatalf("expected stub account to build driver: %v", err)
			}
			return drv
		},
		ListQuery:      schema.SearchQuery{Limit: 1},
		MissingFetchID: "missing-id",
		NotFoundError:  ErrMessageNotFound,
		SendRaw:        outbound,
		AssertList: func(t *testing.T, got []schema.MessageMetaSummary) {
			t.Helper()
			if len(got) != 1 {
				t.Fatalf("expected limited stub list result, got %d", len(got))
			}
			if got[0].ID == "" {
				t.Fatalf("expected stub list item id")
			}
		},
		AssertFetchRaw: func(t *testing.T, listed schema.MessageMetaSummary, raw []byte) {
			t.Helper()
			if !strings.Contains(string(raw), "Subject:") {
				t.Fatalf("expected raw stub message to look like RFC822 content")
			}
		},
		AssertAfterSend: func(t *testing.T, drv drivertest.Driver) {
			t.Helper()
			stub := drv.(*stubDriver)
			if got := len(stub.sent); got != 1 {
				t.Fatalf("expected stub driver to record sent payload, got %d", got)
			}
		},
	})
}

func TestStubDriverFetchRawReturnsNotFound(t *testing.T) {
	drv, err := NewFromAccount(config.AccountConfig{
		Name:   "demo",
		Driver: "stub",
	})
	if err != nil {
		t.Fatalf("expected stub account to build driver: %v", err)
	}

	_, err = drv.FetchRaw(context.Background(), "missing-id")
	if !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("expected missing stub message to return ErrMessageNotFound, got %v", err)
	}
}
