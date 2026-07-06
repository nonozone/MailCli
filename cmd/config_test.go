package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/pkg/schema"
)

func TestConfigInitWritesIMAPConfigWithEnvironmentSecretReferences(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "nested", "config.yaml")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"config", "init",
		"--config", configPath,
		"--account", "work",
		"--driver", "imap",
		"--host", "imap.example.com",
		"--port", "993",
		"--username", "user@example.com",
		"--password-env", "MAILCLI_IMAP_PASSWORD",
		"--smtp-host", "smtp.example.com",
		"--smtp-port", "587",
		"--smtp-username", "user@example.com",
		"--smtp-password-env", "MAILCLI_SMTP_PASSWORD",
		"--mailbox", "INBOX",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config init to succeed: %v\n%s", err, out.String())
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to be written: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"current_account: work",
		"name: work",
		"driver: imap",
		"host: imap.example.com",
		"password: ${MAILCLI_IMAP_PASSWORD}",
		"smtp_password: ${MAILCLI_SMTP_PASSWORD}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected config to contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "super-secret") || strings.Contains(out.String(), "super-secret") {
		t.Fatalf("config init must not print or write raw secrets")
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("expected config file stat to succeed: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected config file permissions 0600, got %o", got)
	}

	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON init result: %v\n%s", err, out.String())
	}
	if result["status"] != "created" || result["account"] != "work" || result["driver"] != "imap" {
		t.Fatalf("unexpected init result: %#v", result)
	}
}

func TestConfigInitRefusesToOverwriteExistingConfigWithoutForce(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "current_account: existing\n")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"config", "init",
		"--config", configPath,
		"--account", "work",
		"--driver", "dir",
		"--path", "./mail",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected config init to refuse overwriting an existing file")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected overwrite error to mention --force, got %v", err)
	}
	raw, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != "current_account: existing\n" {
		t.Fatalf("expected existing config to remain unchanged, got %q", string(raw))
	}
}

func TestConfigShowPrintsProviderMetadataAndRedactsSecrets(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", `
current_account: work
accounts:
  - name: work
    provider: gmail
    driver: imap
    auth_method: app_password
    host: imap.gmail.com
    port: 993
    username: user@gmail.com
    password: super-secret
    tls: true
    mailbox: INBOX
    smtp_host: smtp.gmail.com
    smtp_port: 465
    smtp_username: user@gmail.com
    smtp_password: smtp-secret
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "show", "--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config show to succeed: %v\n%s", err, out.String())
	}
	text := out.String()
	for _, want := range []string{
		"provider: gmail",
		"auth:     app_password",
		"host:     imap.gmail.com:993",
		"host:     smtp.gmail.com:465",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected config show to contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "super-secret") || strings.Contains(text, "smtp-secret") {
		t.Fatalf("config show must not print configured secrets: %s", text)
	}
}

func TestConfigDoctorReportsCompleteDirAccountOK(t *testing.T) {
	fixtureDir := t.TempDir()
	configPath := writeTempFile(t, "config.yaml", `
current_account: fixtures
accounts:
  - name: fixtures
    driver: dir
    path: `+fixtureDir+`
    mailbox: INBOX
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "doctor", "--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config doctor to succeed for complete dir config: %v\n%s", err, out.String())
	}

	var got schema.ConfigDiagnostics
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("expected JSON doctor result: %v\n%s", err, out.String())
	}
	if got.Status != "ok" {
		t.Fatalf("expected ok status, got %+v", got)
	}
	if len(got.Accounts) != 1 || got.Accounts[0].Status != "ok" {
		t.Fatalf("expected one ok account, got %+v", got.Accounts)
	}
	if !got.Accounts[0].Capabilities.Capabilities.LocalIndex {
		t.Fatalf("expected doctor to include account capabilities: %+v", got.Accounts[0].Capabilities)
	}
}

func TestConfigDoctorReportsIncompleteIMAPConfigWithoutLeakingSecrets(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", `
current_account: work
accounts:
  - name: work
    driver: imap
    host: imap.example.com
    port: 993
    username: user@example.com
    password: super-secret
    smtp_host: smtp.example.com
    smtp_password: smtp-secret
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "doctor", "--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config doctor to return diagnostics without failing command: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "super-secret") {
		t.Fatalf("config doctor must not leak configured secrets: %s", out.String())
	}
	if strings.Contains(out.String(), "smtp-secret") {
		t.Fatalf("config doctor must not leak smtp secrets: %s", out.String())
	}

	var got schema.ConfigDiagnostics
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("expected JSON doctor result: %v\n%s", err, out.String())
	}
	if got.Status != "warning" {
		t.Fatalf("expected warning status for incomplete smtp config, got %+v", got)
	}
	if len(got.Problems) == 0 {
		t.Fatalf("expected flattened diagnostic problems")
	}
	if !hasDiagnosticCode(got.Problems, "smtp_port_missing") {
		t.Fatalf("expected smtp missing diagnostics, got %+v", got.Problems)
	}
}

func TestConfigDoctorReportsUnsetSecretEnvironmentReferences(t *testing.T) {
	t.Setenv("MAILCLI_IMAP_PASSWORD", "")
	t.Setenv("MAILCLI_SMTP_PASSWORD", "")
	configPath := writeTempFile(t, "config.yaml", `
current_account: work
accounts:
  - name: work
    driver: imap
    host: imap.example.com
    port: 993
    username: user@example.com
    password: ${MAILCLI_IMAP_PASSWORD}
    smtp_host: smtp.example.com
    smtp_port: 587
    smtp_password: ${MAILCLI_SMTP_PASSWORD}
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "doctor", "--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config doctor to return diagnostics without failing command: %v\n%s", err, out.String())
	}

	var got schema.ConfigDiagnostics
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("expected JSON doctor result: %v\n%s", err, out.String())
	}
	if got.Status != "error" {
		t.Fatalf("expected error status for unset inbound env reference, got %+v", got)
	}
	if !hasDiagnosticCode(got.Problems, "imap_password_env_unset") {
		t.Fatalf("expected imap env unset diagnostic, got %+v", got.Problems)
	}
	if !hasDiagnosticCode(got.Problems, "smtp_password_env_unset") {
		t.Fatalf("expected smtp env unset diagnostic, got %+v", got.Problems)
	}
}

func TestConfigDoctorReportsEnvironmentBackedIMAPConfigOKWithoutLeakingSecrets(t *testing.T) {
	t.Setenv("MAILCLI_IMAP_PASSWORD", "imap-secret")
	t.Setenv("MAILCLI_SMTP_PASSWORD", "smtp-secret")
	configPath := writeTempFile(t, "config.yaml", `
current_account: work
accounts:
  - name: work
    driver: imap
    host: imap.example.com
    port: 993
    username: user@example.com
    password: ${MAILCLI_IMAP_PASSWORD}
    smtp_host: smtp.example.com
    smtp_port: 587
    smtp_password: ${MAILCLI_SMTP_PASSWORD}
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "doctor", "--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config doctor to succeed for env-backed config: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "imap-secret") || strings.Contains(out.String(), "smtp-secret") {
		t.Fatalf("config doctor must not leak expanded secrets: %s", out.String())
	}

	var got schema.ConfigDiagnostics
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("expected JSON doctor result: %v\n%s", err, out.String())
	}
	if got.Status != "ok" {
		t.Fatalf("expected ok status for env-backed config, got %+v", got)
	}
	if len(got.Accounts) != 1 || !got.Accounts[0].Capabilities.Configuration.InboundConfigured || !got.Accounts[0].Capabilities.Configuration.OutboundConfigured {
		t.Fatalf("expected env-backed config to report inbound and outbound capabilities: %+v", got.Accounts)
	}
}

func TestConfigDoctorReportsMissingCurrentAccount(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", `
current_account: missing
accounts:
  - name: work
    driver: dir
    path: ./testdata/emails
    mailbox: INBOX
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "doctor", "--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config doctor to return diagnostics without failing command: %v\n%s", err, out.String())
	}

	var got schema.ConfigDiagnostics
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("expected JSON doctor result: %v\n%s", err, out.String())
	}
	if got.Status != "error" {
		t.Fatalf("expected error status for missing current account, got %+v", got)
	}
	if !hasDiagnosticCode(got.Problems, "current_account_not_found") {
		t.Fatalf("expected current account diagnostic, got %+v", got.Problems)
	}
}

func hasDiagnosticCode(items []schema.ConfigDiagnostic, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func TestConfigCapabilitiesReportsIMAPAccountCapabilities(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", `
current_account: work
accounts:
  - name: work
    driver: imap
    host: imap.example.com
    port: 993
    username: user@example.com
    password: super-secret
    tls: true
    mailbox: INBOX
    smtp_host: smtp.example.com
    smtp_port: 587
    smtp_username: user@example.com
    smtp_password: smtp-secret
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "capabilities", "--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config capabilities to succeed: %v", err)
	}

	if strings.Contains(out.String(), "super-secret") || strings.Contains(out.String(), "smtp-secret") {
		t.Fatalf("capabilities output must not include configured secrets: %s", out.String())
	}

	var got schema.AccountCapabilities
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("expected JSON capabilities output: %v", err)
	}

	if got.Account != "work" || got.Driver != "imap" || got.Mailbox != "INBOX" {
		t.Fatalf("unexpected account identity: %+v", got)
	}
	if !got.Capabilities.List || !got.Capabilities.FetchRaw || !got.Capabilities.Watch || !got.Capabilities.Send {
		t.Fatalf("expected imap account to support list, fetch, watch, send: %+v", got.Capabilities)
	}
	if !got.Capabilities.Delete || !got.Capabilities.Move || !got.Capabilities.MarkRead {
		t.Fatalf("expected imap account to support mailbox mutations: %+v", got.Capabilities)
	}
	if !got.Capabilities.LocalIndex || !got.Configuration.InboundConfigured || !got.Configuration.OutboundConfigured {
		t.Fatalf("expected configured imap account to be indexable with inbound/outbound config: %+v", got)
	}
}

func TestConfigCapabilitiesReportsDirAccountWithoutOutboundOrWatch(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", `
current_account: fixtures
accounts:
  - name: fixtures
    driver: dir
    path: ./testdata/emails
    mailbox: INBOX
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "capabilities", "--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config capabilities to succeed: %v", err)
	}

	var got schema.AccountCapabilities
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("expected JSON capabilities output: %v", err)
	}

	if got.Account != "fixtures" || got.Driver != "dir" {
		t.Fatalf("unexpected account identity: %+v", got)
	}
	if !got.Capabilities.List || !got.Capabilities.FetchRaw || !got.Capabilities.LocalIndex {
		t.Fatalf("expected dir account to support read and local index capabilities: %+v", got.Capabilities)
	}
	if got.Capabilities.Send || got.Capabilities.Watch {
		t.Fatalf("expected dir account to omit send and watch capabilities: %+v", got.Capabilities)
	}
	if !got.Capabilities.Delete || !got.Capabilities.Move || !got.Capabilities.MarkRead {
		t.Fatalf("expected dir account to report mailbox mutation support: %+v", got.Capabilities)
	}
	if !got.Configuration.InboundConfigured || got.Configuration.OutboundConfigured {
		t.Fatalf("expected dir account to be inbound-only: %+v", got.Configuration)
	}
}

func TestConfigCapabilitiesRequiresCompleteIMAPInboundConfigForReadActions(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", `
current_account: work
accounts:
  - name: work
    driver: imap
    username: user@example.com
    mailbox: INBOX
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"config", "capabilities", "--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected config capabilities to succeed: %v", err)
	}

	var got schema.AccountCapabilities
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("expected JSON capabilities output: %v", err)
	}

	if got.Configuration.InboundConfigured {
		t.Fatalf("expected incomplete imap account to report inbound_configured=false: %+v", got.Configuration)
	}
	if got.Capabilities.List || got.Capabilities.FetchRaw || got.Capabilities.Watch || got.Capabilities.Delete {
		t.Fatalf("expected incomplete imap inbound config to disable read and mutation capabilities: %+v", got.Capabilities)
	}
}
