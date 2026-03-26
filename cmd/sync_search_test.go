package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestSyncAndSearchCommandsUseLocalIndex(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "current_account: demo\naccounts:\n  - name: demo\n    driver: stub\n")
	indexPath := writeTempFile(t, "index.json", "{}\n")

	root := NewRootCmd()
	var syncOut bytes.Buffer
	root.SetOut(&syncOut)
	root.SetErr(&syncOut)
	root.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", "2"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected sync command to succeed: %v", err)
	}

	if !strings.Contains(syncOut.String(), `"indexed_count": 2`) {
		t.Fatalf("expected sync result output, got %s", syncOut.String())
	}

	searchCmd := NewRootCmd()
	var searchOut bytes.Buffer
	searchCmd.SetOut(&searchOut)
	searchCmd.SetErr(&searchOut)
	searchCmd.SetArgs([]string{"search", "--index", indexPath, "invoice"})

	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("expected search command to succeed: %v", err)
	}

	if !strings.Contains(searchOut.String(), `"subject": "Invoice INV-2026-031 ready"`) {
		t.Fatalf("expected search output to include indexed invoice mail, got %s", searchOut.String())
	}
}

func TestSyncCommandSkipsExistingMessagesByDefault(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "current_account: demo\naccounts:\n  - name: demo\n    driver: stub\n")
	indexPath := writeTempFile(t, "index.json", "{}\n")

	first := NewRootCmd()
	var firstOut bytes.Buffer
	first.SetOut(&firstOut)
	first.SetErr(&firstOut)
	first.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", "2"})
	if err := first.Execute(); err != nil {
		t.Fatalf("expected first sync to succeed: %v", err)
	}

	second := NewRootCmd()
	var secondOut bytes.Buffer
	second.SetOut(&secondOut)
	second.SetErr(&secondOut)
	second.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", "2"})
	if err := second.Execute(); err != nil {
		t.Fatalf("expected second sync to succeed: %v", err)
	}

	if !strings.Contains(secondOut.String(), `"indexed_count": 0`) {
		t.Fatalf("expected second sync to skip reindexing, got %s", secondOut.String())
	}
	if !strings.Contains(secondOut.String(), `"skipped_count": 2`) {
		t.Fatalf("expected second sync to report skipped messages, got %s", secondOut.String())
	}
}

func TestSyncCommandRefreshesExistingMessagesWhenRequested(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "current_account: demo\naccounts:\n  - name: demo\n    driver: stub\n")
	indexPath := writeTempFile(t, "index.json", "{}\n")

	first := NewRootCmd()
	first.SetOut(&bytes.Buffer{})
	first.SetErr(&bytes.Buffer{})
	first.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", "2"})
	if err := first.Execute(); err != nil {
		t.Fatalf("expected first sync to succeed: %v", err)
	}

	refresh := NewRootCmd()
	var refreshOut bytes.Buffer
	refresh.SetOut(&refreshOut)
	refresh.SetErr(&refreshOut)
	refresh.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", "2", "--refresh"})
	if err := refresh.Execute(); err != nil {
		t.Fatalf("expected refresh sync to succeed: %v", err)
	}

	if !strings.Contains(refreshOut.String(), `"indexed_count": 2`) {
		t.Fatalf("expected refresh sync to reindex messages, got %s", refreshOut.String())
	}
	if !strings.Contains(refreshOut.String(), `"skipped_count": 0`) {
		t.Fatalf("expected refresh sync to avoid skip counts, got %s", refreshOut.String())
	}
}
