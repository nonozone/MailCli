package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
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

	if !strings.Contains(out.String(), `.eml"`) {
		t.Fatalf("expected dir-backed list output to contain a relative file id, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"subject": "`) {
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
	expectedFixtures := countFixtureEmails(t, root)

	loadConfigFunc = config.Load
	driverFactoryFunc = driver.NewFromAccount

	syncCmd := NewRootCmd()
	var syncOut bytes.Buffer
	syncCmd.SetOut(&syncOut)
	syncCmd.SetErr(&syncOut)
	syncCmd.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", expectedFixtures})
	if err := syncCmd.Execute(); err != nil {
		t.Fatalf("expected dir-backed sync to succeed: %v", err)
	}

	if !strings.Contains(syncOut.String(), `"indexed_count": `+expectedFixtures) {
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

func TestSyncCommandWithDirDriverTreatsZeroLimitAsUnbounded(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	root := mustFixtureMailRoot(t)
	configPath := writeTempFile(t, "config.yaml", "current_account: fixtures\naccounts:\n  - name: fixtures\n    driver: dir\n    path: "+root+"\n    mailbox: INBOX\n")
	indexPath := writeTempFile(t, "index.json", "{}\n")
	expectedFixtures := countFixtureEmails(t, root)

	loadConfigFunc = config.Load
	driverFactoryFunc = driver.NewFromAccount

	syncCmd := NewRootCmd()
	var syncOut bytes.Buffer
	syncCmd.SetOut(&syncOut)
	syncCmd.SetErr(&syncOut)
	syncCmd.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", "0"})
	if err := syncCmd.Execute(); err != nil {
		t.Fatalf("expected zero-limit dir-backed sync to succeed: %v", err)
	}

	if !strings.Contains(syncOut.String(), `"indexed_count": `+expectedFixtures) {
		t.Fatalf("expected zero-limit dir-backed sync to index all fixtures, got %s", syncOut.String())
	}
	if !strings.Contains(syncOut.String(), `"listed_count": `+expectedFixtures) {
		t.Fatalf("expected zero-limit dir-backed sync to list all fixtures, got %s", syncOut.String())
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

func countFixtureEmails(t *testing.T, root string) string {
	t.Helper()

	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".eml" {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected fixture email count to succeed: %v", err)
	}
	return strconv.Itoa(count)
}
