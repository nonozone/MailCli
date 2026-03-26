package driver

import (
	"testing"

	"github.com/nonozone/MailCli/internal/config"
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
