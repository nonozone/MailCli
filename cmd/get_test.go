package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/schema"
)

type fakeFetchDriver struct {
	raw []byte
}

func (f fakeFetchDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	return nil, nil
}

func (f fakeFetchDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	return f.raw, nil
}

func (f fakeFetchDriver) SendRaw(ctx context.Context, raw []byte) error {
	return nil
}

func TestGetCommandParsesFetchedMessage(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configPath, []byte("current_account: local\naccounts:\n  - name: local\n    driver: fake\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loadConfigFunc = config.Load
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fakeFetchDriver{
			raw: []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Hello\r\nMessage-ID: <id-1@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nHello from fetched email."),
		}, nil
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"get", "--config", configPath, "id-1"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("expected get command to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "\"subject\": \"Hello\"") {
		t.Fatalf("expected parsed message output")
	}
}
