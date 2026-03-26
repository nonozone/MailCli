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

type fakeDriver struct {
	items []schema.MessageMetaSummary
}

func (f fakeDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	return f.items, nil
}

func (f fakeDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	return nil, nil
}

func (f fakeDriver) SendRaw(ctx context.Context, raw []byte) error {
	return nil
}

func TestListCommandReadsConfiguredAccount(t *testing.T) {
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
		if account.Name != "local" {
			t.Fatalf("expected selected account to be local")
		}
		return fakeDriver{
			items: []schema.MessageMetaSummary{
				{ID: "1", Subject: "Hello"},
			},
		}, nil
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--config", configPath})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("expected list command to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "\"subject\": \"Hello\"") {
		t.Fatalf("expected json list output")
	}
}

func TestListCommandFailsForMissingAccount(t *testing.T) {
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
		t.Fatalf("driver factory should not be called when account resolution fails")
		return nil, nil
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--config", configPath, "--account", "missing"})

	err = cmd.Execute()
	if err == nil {
		t.Fatalf("expected list command to fail for missing account")
	}
}

func TestListCommandSupportsTableFormat(t *testing.T) {
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
		return fakeDriver{
			items: []schema.MessageMetaSummary{
				{ID: "1", Subject: "Hello"},
			},
		}, nil
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--config", configPath, "--format", "table"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("expected list table output to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "SUBJECT") {
		t.Fatalf("expected table output")
	}
}
