package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourname/mailcli/internal/config"
	"github.com/yourname/mailcli/pkg/driver"
	"github.com/yourname/mailcli/pkg/schema"
)

type fakeSendDriver struct {
	lastRaw  []byte
	sendErr  error
	fetchErr error
}

func (f *fakeSendDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	return nil, nil
}

func (f *fakeSendDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	return nil, f.fetchErr
}

func (f *fakeSendDriver) SendRaw(ctx context.Context, raw []byte) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.lastRaw = append([]byte(nil), raw...)
	return nil
}

func TestSendCommandDryRunPrintsMIME(t *testing.T) {
	draftPath := writeTempFile(t, "draft.json", `{
  "from": {"name": "Nono", "address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--dry-run", draftPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected send dry-run to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "Subject: Welcome") {
		t.Fatalf("expected MIME output")
	}
}

func TestReplyCommandDryRunPrintsThreadHeaders(t *testing.T) {
	replyPath := writeTempFile(t, "reply.json", `{
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Re: Question",
  "body_text": "Thanks for the email.",
  "reply_to_message_id": "<orig-123@example.com>",
  "references": ["<older-1@example.com>", "<orig-123@example.com>"]
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--dry-run", replyPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected reply dry-run to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "In-Reply-To: <orig-123@example.com>") {
		t.Fatalf("expected reply MIME output")
	}
}

func TestSendCommandUsesConfiguredDriver(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected send command to succeed: %v", err)
	}

	if len(fake.lastRaw) == 0 {
		t.Fatalf("expected send command to pass MIME to driver")
	}
	if !strings.Contains(out.String(), "\"ok\": true") {
		t.Fatalf("expected send result output")
	}
}

func TestSendCommandReturnsStructuredAuthFailure(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{sendErr: assertErr("535 Authentication credentials invalid")}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "auth_failed"`) {
		t.Fatalf("expected auth_failed send result, got %s", out.String())
	}
}

func TestSendCommandReturnsStructuredAccountNotFound(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		t.Fatalf("driver factory should not be called when account resolution fails")
		return nil, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "missing",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "account_not_found"`) {
		t.Fatalf("expected account_not_found send result, got %s", out.String())
	}
}

func TestReplyCommandUsesConfiguredDriver(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	replyPath := writeTempFile(t, "reply.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Question",
  "body_text": "Thanks for the email.",
  "reply_to_message_id": "<orig-123@example.com>",
  "references": ["<orig-123@example.com>"]
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--config", configPath, replyPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected reply command to succeed: %v", err)
	}

	if len(fake.lastRaw) == 0 || !strings.Contains(string(fake.lastRaw), "In-Reply-To: <orig-123@example.com>") {
		t.Fatalf("expected reply command to send MIME with thread headers")
	}
	if !strings.Contains(out.String(), "\"ok\": true") {
		t.Fatalf("expected send result output")
	}
}

func TestReplyCommandReturnsStructuredMessageNotFound(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{fetchErr: assertErr("message not found: imap:uid:123")}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	replyPath := writeTempFile(t, "reply.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "body_text": "Thanks for the email.",
  "reply_to_id": "imap:uid:123"
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--config", configPath, replyPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "message_not_found"`) {
		t.Fatalf("expected message_not_found reply result, got %s", out.String())
	}
}

func TestSendCommandReturnsStructuredTransportNotConfigured(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{sendErr: assertErr("smtp settings not configured for account")}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "transport_not_configured"`) {
		t.Fatalf("expected transport_not_configured send result, got %s", out.String())
	}
}

func TestReplyCommandResolvesReplyToIDViaFetch(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return &struct {
			*fakeSendDriver
		}{fake}, nil
	}

	replyPath := writeTempFile(t, "reply.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "body_text": "Thanks for the email.",
  "reply_to_id": "imap:uid:123"
}`)

	drv := &replyResolveDriver{
		fakeSendDriver: fake,
		raw:            []byte("From: sender@example.com\r\nTo: support@nono.im\r\nSubject: Original subject\r\nMessage-ID: <orig-123@example.com>\r\nReferences: <older-1@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nOriginal body"),
	}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return drv, nil
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--config", configPath, replyPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected reply command to succeed: %v", err)
	}

	raw := string(drv.lastRaw)
	if !strings.Contains(raw, "In-Reply-To: <orig-123@example.com>") {
		t.Fatalf("expected reply_to_id to resolve original message id")
	}
	if !strings.Contains(raw, "References: <older-1@example.com> <orig-123@example.com>") {
		t.Fatalf("expected references to include original thread")
	}
	if !strings.Contains(raw, "Subject: Re: Original subject") {
		t.Fatalf("expected subject to derive from original message")
	}
}

func TestSendCommandDryRunInvalidJSONStillReturnsRawError(t *testing.T) {
	draftPath := writeTempFile(t, "draft.json", `{`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--dry-run", draftPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected dry-run invalid json to fail")
	}
	if errors.Is(err, errSendFailure) {
		t.Fatalf("expected dry-run invalid json to keep raw error path")
	}
	if strings.Contains(out.String(), `"ok": false`) {
		t.Fatalf("expected no structured SendResult on dry-run parse failure")
	}
}

type replyResolveDriver struct {
	*fakeSendDriver
	raw []byte
}

func (d *replyResolveDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	return nil, nil
}

func (d *replyResolveDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	return d.raw, nil
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertErr(message string) error {
	return fakeErr(message)
}

type fakeErr string

func (e fakeErr) Error() string {
	return string(e)
}
