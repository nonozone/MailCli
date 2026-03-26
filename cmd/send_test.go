package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourname/mailcli/internal/config"
	"github.com/yourname/mailcli/pkg/driver"
	"github.com/yourname/mailcli/pkg/schema"
)

type fakeSendDriver struct {
	lastRaw []byte
}

func (f *fakeSendDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	return nil, nil
}

func (f *fakeSendDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	return nil, nil
}

func (f *fakeSendDriver) SendRaw(ctx context.Context, raw []byte) error {
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

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
