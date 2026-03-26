package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	cfg := Config{
		CurrentAccount: "local",
		Accounts: []AccountConfig{
			{Name: "local", Driver: "imap"},
		},
	}

	data, err := Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	if got.CurrentAccount != "local" {
		t.Fatalf("expected round-trip config")
	}
}

func TestLoadReadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(path, []byte("current_account: local\naccounts:\n  - name: local\n    driver: imap\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if cfg.CurrentAccount != "local" {
		t.Fatalf("expected current account to be loaded")
	}
}

func TestResolveAccountUsesOverrideBeforeCurrent(t *testing.T) {
	cfg := Config{
		CurrentAccount: "default",
		Accounts: []AccountConfig{
			{Name: "default", Driver: "imap"},
			{Name: "work", Driver: "imap"},
		},
	}

	account, err := cfg.ResolveAccount("work")
	if err != nil {
		t.Fatalf("expected resolve account to succeed: %v", err)
	}

	if account.Name != "work" {
		t.Fatalf("expected override account")
	}
}

func TestResolveAccountFailsForMissingName(t *testing.T) {
	cfg := Config{
		CurrentAccount: "default",
		Accounts: []AccountConfig{
			{Name: "default", Driver: "imap"},
		},
	}

	_, err := cfg.ResolveAccount("missing")
	if err == nil {
		t.Fatalf("expected missing account to fail")
	}
}

func TestUnmarshalExpandsEnvironmentVariablesForSecretFields(t *testing.T) {
	t.Setenv("MAILCLI_IMAP_PASSWORD", "imap-secret")
	t.Setenv("MAILCLI_SMTP_PASSWORD", "smtp-secret")

	cfg, err := Unmarshal([]byte(`
current_account: work
accounts:
  - name: work
    driver: imap
    password: ${MAILCLI_IMAP_PASSWORD}
    smtp_password: ${MAILCLI_SMTP_PASSWORD}
`))
	if err != nil {
		t.Fatalf("expected unmarshal to succeed: %v", err)
	}

	account, err := cfg.ResolveAccount("work")
	if err != nil {
		t.Fatalf("expected account to resolve: %v", err)
	}

	if account.Password != "imap-secret" {
		t.Fatalf("expected imap password expansion, got %q", account.Password)
	}
	if account.SMTPPassword != "smtp-secret" {
		t.Fatalf("expected smtp password expansion, got %q", account.SMTPPassword)
	}
}

func TestUnmarshalDoesNotExpandEnvironmentVariablesForNonSecretFields(t *testing.T) {
	t.Setenv("MAILCLI_ACCOUNT_NAME", "expanded-name")

	cfg, err := Unmarshal([]byte(`
current_account: work
accounts:
  - name: ${MAILCLI_ACCOUNT_NAME}
    driver: imap
`))
	if err != nil {
		t.Fatalf("expected unmarshal to succeed: %v", err)
	}

	if cfg.Accounts[0].Name != "${MAILCLI_ACCOUNT_NAME}" {
		t.Fatalf("expected non-secret fields to remain literal, got %q", cfg.Accounts[0].Name)
	}
}
