package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/driver"
)

func TestListCommandSupportsDirDriver(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	root := mustFixtureMailRoot(t)
	configPath := writeTempFile(t, "config.yaml", "current_account: fixtures\naccounts:\n  - name: fixtures\n    driver: dir\n    path: "+root+"\n    mailbox: INBOX\n")

	loadConfigFunc = config.Load
	driverFactoryFunc = driver.NewFromAccount

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--config", configPath, "--limit", "2"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected list command with dir driver to succeed: %v", err)
	}

	if !strings.Contains(out.String(), `"id": "postfix_bounce.eml"`) {
		t.Fatalf("expected dir-backed list output to contain a relative file id, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"subject": "Undelivered Mail Returned to Sender"`) {
		t.Fatalf("expected dir-backed list output to contain parsed metadata, got %s", out.String())
	}
}

func TestSyncAndSearchCommandsSupportDirDriver(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	root := mustFixtureMailRoot(t)
	configPath := writeTempFile(t, "config.yaml", "current_account: fixtures\naccounts:\n  - name: fixtures\n    driver: dir\n    path: "+root+"\n    mailbox: INBOX\n")
	indexPath := writeTempFile(t, "index.json", "{}\n")

	loadConfigFunc = config.Load
	driverFactoryFunc = driver.NewFromAccount

	syncCmd := NewRootCmd()
	var syncOut bytes.Buffer
	syncCmd.SetOut(&syncOut)
	syncCmd.SetErr(&syncOut)
	syncCmd.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", "20"})
	if err := syncCmd.Execute(); err != nil {
		t.Fatalf("expected dir-backed sync to succeed: %v", err)
	}

	if !strings.Contains(syncOut.String(), `"indexed_count": 11`) {
		t.Fatalf("expected dir-backed sync to index messages, got %s", syncOut.String())
	}

	searchCmd := NewRootCmd()
	var searchOut bytes.Buffer
	searchCmd.SetOut(&searchOut)
	searchCmd.SetErr(&searchOut)
	searchCmd.SetArgs([]string{"search", "--index", indexPath, "invoice"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("expected dir-backed search to succeed: %v", err)
	}

	if !strings.Contains(searchOut.String(), `"id": "invoice.eml"`) {
		t.Fatalf("expected invoice fixture in search output, got %s", searchOut.String())
	}
	if !strings.Contains(searchOut.String(), `"thread_id": "\u003cinvoice-123@example.com\u003e"`) {
		t.Fatalf("expected invoice search output to expose thread id, got %s", searchOut.String())
	}
}

func mustFixtureMailRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "testdata", "emails"))
	if err != nil {
		t.Fatalf("expected fixture root to resolve: %v", err)
	}
	return root
}
