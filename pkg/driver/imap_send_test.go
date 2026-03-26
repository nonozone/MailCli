package driver

import (
	"context"
	"testing"

	"github.com/yourname/mailcli/internal/config"
)

func TestIMAPDriverSendRawUsesSMTPSettings(t *testing.T) {
	restore := smtpSendFunc
	t.Cleanup(func() {
		smtpSendFunc = restore
	})

	account := config.AccountConfig{
		Name:         "work",
		Driver:       "imap",
		Host:         "imap.example.com",
		Port:         993,
		Username:     "imap-user@example.com",
		Password:     "imap-secret",
		TLS:          true,
		Mailbox:      "INBOX",
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "smtp-user@example.com",
		SMTPPassword: "smtp-secret",
	}

	drv, err := newIMAPDriver(account)
	if err != nil {
		t.Fatalf("expected imap driver to build: %v", err)
	}

	var called bool
	smtpSendFunc = func(cfg smtpConfig, from string, to []string, raw []byte) error {
		called = true
		if cfg.host != "smtp.example.com" || cfg.port != 587 {
			t.Fatalf("expected smtp settings from config")
		}
		if from != "support@nono.im" {
			t.Fatalf("expected from address from message")
		}
		if len(to) != 1 || to[0] != "user@example.com" {
			t.Fatalf("expected recipient extracted from message")
		}
		return nil
	}

	raw := []byte("From: support@nono.im\r\nTo: user@example.com\r\nSubject: Hello\r\n\r\nBody")
	if err := drv.SendRaw(context.Background(), raw); err != nil {
		t.Fatalf("expected send raw to succeed: %v", err)
	}
	if !called {
		t.Fatalf("expected smtp sender to be called")
	}
}

func TestIMAPDriverSendRawFailsWithoutSMTPSettings(t *testing.T) {
	account := config.AccountConfig{
		Name:     "work",
		Driver:   "imap",
		Host:     "imap.example.com",
		Port:     993,
		Username: "imap-user@example.com",
		Password: "imap-secret",
		TLS:      true,
	}

	drv, err := newIMAPDriver(account)
	if err != nil {
		t.Fatalf("expected imap driver to build: %v", err)
	}

	raw := []byte("From: support@nono.im\r\nTo: user@example.com\r\nSubject: Hello\r\n\r\nBody")
	if err := drv.SendRaw(context.Background(), raw); err == nil {
		t.Fatalf("expected send raw to fail without smtp settings")
	}
}
