package driver

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/driver/drivertest"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestNewFromAccountBuildsDirDriver(t *testing.T) {
	account := config.AccountConfig{
		Name:   "fixtures",
		Driver: "dir",
		Path:   filepath.Join("..", "..", "testdata", "emails"),
	}

	drv, err := NewFromAccount(account)
	if err != nil {
		t.Fatalf("expected dir account to build driver: %v", err)
	}

	if _, ok := drv.(*dirDriver); !ok {
		t.Fatalf("expected dir driver, got %T", drv)
	}
}

func TestNewFromAccountRejectsMissingDirPath(t *testing.T) {
	_, err := NewFromAccount(config.AccountConfig{
		Name:   "fixtures",
		Driver: "dir",
	})
	if err == nil {
		t.Fatalf("expected dir driver without path to fail")
	}
}

func TestDirDriverListFetchAndSendNotConfigured(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "emails")

	drivertest.RunContractSuite(t, drivertest.Harness{
		NewDriver: func(t *testing.T) drivertest.Driver {
			t.Helper()

			drv, err := NewFromAccount(config.AccountConfig{
				Name:   "fixtures",
				Driver: "dir",
				Path:   root,
			})
			if err != nil {
				t.Fatalf("expected dir account to build driver: %v", err)
			}
			return drv
		},
		ListQuery:       schema.SearchQuery{Query: "April invoice", Limit: 1},
		MissingFetchID:  "missing.eml",
		NotFoundError:   ErrMessageNotFound,
		SendRaw:         []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Test\r\n\r\nHello"),
		ExpectedSendErr: ErrTransportNotConfigured,
		AssertList: func(t *testing.T, got []schema.MessageMetaSummary) {
			t.Helper()
			if len(got) != 1 {
				t.Fatalf("expected one filtered dir result, got %d", len(got))
			}
			if got[0].ID != "invoice.eml" {
				t.Fatalf("expected relative file id, got %q", got[0].ID)
			}
			if got[0].Subject != "Your April invoice is ready" {
				t.Fatalf("expected parsed invoice subject, got %q", got[0].Subject)
			}
		},
		AssertFetchRaw: func(t *testing.T, listed schema.MessageMetaSummary, raw []byte) {
			t.Helper()
			if listed.ID != "invoice.eml" {
				t.Fatalf("expected listed id to be invoice.eml, got %q", listed.ID)
			}
			if !strings.Contains(string(raw), "Message-ID: <invoice-123@example.com>") {
				t.Fatalf("expected raw invoice message bytes, got %q", string(raw))
			}
		},
	})
}

func TestDirDriverFetchRawSupportsMessageID(t *testing.T) {
	drv, err := NewFromAccount(config.AccountConfig{
		Name:   "fixtures",
		Driver: "dir",
		Path:   filepath.Join("..", "..", "testdata", "emails"),
	})
	if err != nil {
		t.Fatalf("expected dir account to build driver: %v", err)
	}

	raw, err := drv.FetchRaw(context.Background(), "<plain-123@example.com>")
	if err != nil {
		t.Fatalf("expected fetch by message id to succeed: %v", err)
	}

	if !strings.Contains(string(raw), "Subject: Plaintext message") {
		t.Fatalf("expected plaintext raw content, got %q", string(raw))
	}
}

func TestDirDriverFetchRawRejectsPathTraversal(t *testing.T) {
	drv, err := NewFromAccount(config.AccountConfig{
		Name:   "fixtures",
		Driver: "dir",
		Path:   filepath.Join("..", "..", "testdata", "emails"),
	})
	if err != nil {
		t.Fatalf("expected dir account to build driver: %v", err)
	}

	_, err = drv.FetchRaw(context.Background(), "../secret.eml")
	if !errors.Is(err, ErrMessageNotFound) && err == nil {
		t.Fatalf("expected path traversal to be rejected, got %v", err)
	}
}
