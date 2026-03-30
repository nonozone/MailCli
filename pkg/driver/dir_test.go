package driver

import (
	"context"
	"errors"
	"os"
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

// ── Writer tests (Delete / Move / MarkRead) ───────────────────────────────────

const minimalEML = "From: sender@example.com\r\nTo: user@example.com\r\nSubject: Test\r\nMessage-ID: <writer-test@example.com>\r\nDate: Mon, 30 Mar 2026 12:00:00 +0000\r\n\r\nBody text."

func newTempDirDriver(t *testing.T) (*dirDriver, string) {
	t.Helper()
	dir := t.TempDir()
	drv, err := newDirDriver(config.AccountConfig{
		Name:   "test",
		Driver: "dir",
		Path:   dir,
	})
	if err != nil {
		t.Fatalf("expected temp dir driver: %v", err)
	}
	return drv.(*dirDriver), dir
}

func TestDirDriverDelete(t *testing.T) {
	drv, dir := newTempDirDriver(t)

	path := filepath.Join(dir, "msg.eml")
	if err := os.WriteFile(path, []byte(minimalEML), 0o644); err != nil {
		t.Fatal(err)
	}

	w, ok := Driver(drv).(Writer)
	if !ok {
		t.Fatal("dirDriver should implement Writer")
	}

	if err := w.Delete(context.Background(), "msg.eml"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, stat err: %v", err)
	}
}

func TestDirDriverDeleteReturnsNotFoundForMissing(t *testing.T) {
	drv, _ := newTempDirDriver(t)
	w := Driver(drv).(Writer)

	err := w.Delete(context.Background(), "nonexistent.eml")
	if !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("expected ErrMessageNotFound, got %v", err)
	}
}

func TestDirDriverMove(t *testing.T) {
	drv, dir := newTempDirDriver(t)

	path := filepath.Join(dir, "msg.eml")
	if err := os.WriteFile(path, []byte(minimalEML), 0o644); err != nil {
		t.Fatal(err)
	}

	w := Driver(drv).(Writer)
	if err := w.Move(context.Background(), "msg.eml", "Archive"); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected original to be removed")
	}
	destPath := filepath.Join(dir, "Archive", "msg.eml")
	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("expected file in dest: %v", err)
	}
}

func TestDirDriverMarkRead(t *testing.T) {
	drv, dir := newTempDirDriver(t)

	path := filepath.Join(dir, "msg.eml")
	if err := os.WriteFile(path, []byte(minimalEML), 0o644); err != nil {
		t.Fatal(err)
	}

	w := Driver(drv).(Writer)

	// Mark as read — file should be renamed to seen.msg.eml
	if err := w.MarkRead(context.Background(), "msg.eml", true); err != nil {
		t.Fatalf("MarkRead(true) failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "seen.msg.eml")); err != nil {
		t.Fatalf("expected seen.msg.eml: %v", err)
	}

	// Mark back as unread
	if err := w.MarkRead(context.Background(), "seen.msg.eml", false); err != nil {
		t.Fatalf("MarkRead(false) failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "unseen.msg.eml")); err != nil {
		t.Fatalf("expected unseen.msg.eml: %v", err)
	}
}

func TestDirDriverMarkReadByMessageID(t *testing.T) {
	drv, dir := newTempDirDriver(t)

	if err := os.WriteFile(filepath.Join(dir, "msg.eml"), []byte(minimalEML), 0o644); err != nil {
		t.Fatal(err)
	}

	w := Driver(drv).(Writer)
	if err := w.MarkRead(context.Background(), "<writer-test@example.com>", true); err != nil {
		t.Fatalf("MarkRead by message-id failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "seen.msg.eml")); err != nil {
		t.Fatalf("expected seen.msg.eml: %v", err)
	}
}
