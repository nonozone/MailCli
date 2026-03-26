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
