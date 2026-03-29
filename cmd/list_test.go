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
	items      []schema.MessageMetaSummary
	lastQuery  schema.SearchQuery
}

func (f fakeDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	f.lastQuery = query
	return f.items, nil
}

func (f fakeDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	return nil, nil
}

func (f fakeDriver) SendRaw(ctx context.Context, raw []byte) error {
	return nil
}

type captureListDriver struct {
	items     []schema.MessageMetaSummary
	lastQuery schema.SearchQuery
}

func (f *captureListDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	f.lastQuery = query
	return f.items, nil
}

func (f *captureListDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	return nil, nil
}

func (f *captureListDriver) SendRaw(ctx context.Context, raw []byte) error {
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

func TestListCommandPassesMailboxAndLimitToDriver(t *testing.T) {
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
	fake := &captureListDriver{
		items: []schema.MessageMetaSummary{
			{ID: "1", Subject: "Hello"},
		},
	}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--config", configPath, "--mailbox", "Archive", "--limit", "5"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("expected list command to succeed: %v", err)
	}

	if fake.lastQuery.Mailbox != "Archive" {
		t.Fatalf("expected mailbox to be propagated, got %q", fake.lastQuery.Mailbox)
	}
	if fake.lastQuery.Limit != 5 {
		t.Fatalf("expected limit to be propagated, got %d", fake.lastQuery.Limit)
	}
}

func TestListCommandPropagatesZeroLimitToDriver(t *testing.T) {
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
	fake := &captureListDriver{
		items: []schema.MessageMetaSummary{
			{ID: "1", Subject: "Hello"},
		},
	}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--config", configPath, "--limit", "0"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("expected zero-limit list to succeed: %v", err)
	}

	if fake.lastQuery.Limit != 0 {
		t.Fatalf("expected zero limit to be propagated unchanged, got %d", fake.lastQuery.Limit)
	}
}
