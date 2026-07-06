package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAccountAddGmailWritesProviderPresetWithSecretReferences(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"account", "add",
		"--config", configPath,
		"--provider", "gmail",
		"--email", "user@gmail.com",
		"--account", "personal",
		"--password-env", "MAILCLI_GMAIL_APP_PASSWORD",
		"--format", "json",
		"--force",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected account add to succeed: %v\n%s", err, out.String())
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to be written: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"current_account: personal",
		"name: personal",
		"driver: imap",
		"host: imap.gmail.com",
		"port: 993",
		"username: user@gmail.com",
		"password: ${MAILCLI_GMAIL_APP_PASSWORD}",
		"smtp_host: smtp.gmail.com",
		"smtp_port: 465",
		"smtp_username: user@gmail.com",
		"smtp_password: ${MAILCLI_GMAIL_APP_PASSWORD}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected config to contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "app-password") || strings.Contains(out.String(), "app-password") {
		t.Fatalf("account add must not write or print raw secrets")
	}

	var result accountAddResult
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON result: %v\n%s", err, out.String())
	}
	if result.Status != "configured" || result.Provider != "gmail" || result.AuthMethod != "app_password" {
		t.Fatalf("unexpected account add result: %+v", result)
	}
	if result.NextSteps[0] == "" {
		t.Fatalf("expected human next steps in result")
	}
}

func TestAccountAddAppendsAccountWithoutOverwritingExistingConfig(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", `
current_account: work
accounts:
  - name: work
    driver: dir
    path: ./fixtures
`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"account", "add",
		"--config", configPath,
		"--provider", "163",
		"--email", "user@163.com",
		"--password-env", "MAILCLI_163_AUTH_CODE",
		"--format", "json",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected account add to append account: %v\n%s", err, out.String())
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to be written: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"current_account: user_163_com",
		"name: work",
		"name: user_163_com",
		"host: imap.163.com",
		"smtp_host: smtp.163.com",
		"password: ${MAILCLI_163_AUTH_CODE}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected appended config to contain %q, got:\n%s", want, text)
		}
	}
}

func TestAccountAddInteractiveQQDoesNotLeakSecret(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader(strings.Join([]string{
		"qq",
		"user@qq.com",
		"MAILCLI_QQ_AUTH_CODE",
		"y",
	}, "\n") + "\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"account", "add", "--config", configPath, "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected interactive account add to succeed: %v\n%s", err, out.String())
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to be written: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: user_qq_com",
		"host: imap.qq.com",
		"smtp_host: smtp.qq.com",
		"password: ${MAILCLI_QQ_AUTH_CODE}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected QQ config to contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(out.String(), "授权码内容") || strings.Contains(text, "授权码内容") {
		t.Fatalf("interactive account add must not request or store raw authorization codes")
	}
	if !strings.Contains(out.String(), "Set MAILCLI_QQ_AUTH_CODE") {
		t.Fatalf("expected next-step guidance for env secret, got:\n%s", out.String())
	}
}

func TestAccountAddGenericIMAPRequiresExplicitHost(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"account", "add",
		"--config", configPath,
		"--provider", "generic-imap",
		"--email", "user@example.com",
		"--password-env", "MAILCLI_GENERIC_PASSWORD",
		"--format", "json",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected generic-imap without host to fail")
	}
	if !strings.Contains(err.Error(), "--host is required") {
		t.Fatalf("expected host error, got %v", err)
	}
}

func TestAccountAddGenericIMAPRequiresSMTPHostWhenSMTPPortProvided(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"account", "add",
		"--config", configPath,
		"--provider", "generic-imap",
		"--email", "user@example.com",
		"--password-env", "MAILCLI_GENERIC_PASSWORD",
		"--host", "imap.example.com",
		"--smtp-port", "465",
		"--format", "json",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected generic-imap with partial SMTP options to fail")
	}
	if !strings.Contains(err.Error(), "--smtp-host is required") {
		t.Fatalf("expected smtp host error, got %v", err)
	}
}

func TestAccountAddGenericIMAPUsesExplicitHosts(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"account", "add",
		"--config", configPath,
		"--provider", "generic-imap",
		"--email", "user@example.com",
		"--password-env", "MAILCLI_GENERIC_PASSWORD",
		"--host", "imap.example.com",
		"--port", "993",
		"--smtp-host", "smtp.example.com",
		"--smtp-port", "465",
		"--format", "json",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected generic-imap with explicit hosts to succeed: %v\n%s", err, out.String())
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to be written: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"provider: generic-imap",
		"host: imap.example.com",
		"smtp_host: smtp.example.com",
		"smtp_port: 465",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected generic config to contain %q, got:\n%s", want, text)
		}
	}
}

func TestAccountAddOutlookReadFirstDoesNotWritePartialSMTP(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"account", "add",
		"--config", configPath,
		"--provider", "outlook",
		"--email", "user@outlook.com",
		"--password-env", "MAILCLI_OUTLOOK_PASSWORD",
		"--format", "json",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected outlook account add to succeed: %v\n%s", err, out.String())
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to be written: %v", err)
	}
	text := string(raw)
	for _, forbidden := range []string{"smtp_host:", "smtp_port:", "smtp_username:", "smtp_password:"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("expected outlook read-first config to omit %q, got:\n%s", forbidden, text)
		}
	}

	var result accountAddResult
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON result: %v\n%s", err, out.String())
	}
	if result.Outbound {
		t.Fatalf("expected outlook preset to be read-first, got %+v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected OAuth warning for outlook preset")
	}
}
