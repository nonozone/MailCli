package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/internal/config"
	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/schema"
)

// ─── stubWriterDriver ────────────────────────────────────────────────────────

type stubWriterDriver struct {
	items       []schema.MessageMetaSummary
	deletedIDs  []string
	movedIDs    []string
	movedDests  []string
	markedIDs   []string
	markedReads []bool
}

func (w *stubWriterDriver) List(_ context.Context, _ schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	return w.items, nil
}

func (w *stubWriterDriver) FetchRaw(_ context.Context, _ string) ([]byte, error) { return nil, nil }
func (w *stubWriterDriver) SendRaw(_ context.Context, _ []byte) error             { return nil }

func (w *stubWriterDriver) Delete(_ context.Context, id string) error {
	w.deletedIDs = append(w.deletedIDs, id)
	return nil
}

func (w *stubWriterDriver) Move(_ context.Context, id, dest string) error {
	w.movedIDs = append(w.movedIDs, id)
	w.movedDests = append(w.movedDests, dest)
	return nil
}

func (w *stubWriterDriver) MarkRead(_ context.Context, id string, read bool) error {
	w.markedIDs = append(w.markedIDs, id)
	w.markedReads = append(w.markedReads, read)
	return nil
}

// writeMutationFixtures writes a minimal config file and wires up the
// stubWriterDriver, returning a cleanup function.
func writeMutationFixtures(t *testing.T, w *stubWriterDriver) (configPath string) {
	t.Helper()
	dir := t.TempDir()
	configPath = filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configPath, []byte(
		"current_account: work\naccounts:\n  - name: work\n    driver: fake\n",
	), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	loadConfigFunc = config.Load
	driverFactoryFunc = func(_ config.AccountConfig) (driver.Driver, error) { return w, nil }
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})
	return configPath
}

// ─── delete ──────────────────────────────────────────────────────────────────

func TestDeleteCommandCallsWriterDelete(t *testing.T) {
	w := &stubWriterDriver{}
	configPath := writeMutationFixtures(t, w)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"delete", "abc@example.com", "--config", configPath})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete command failed: %v", err)
	}

	if len(w.deletedIDs) != 1 || w.deletedIDs[0] != "abc@example.com" {
		t.Fatalf("expected delete called with abc@example.com, got %v", w.deletedIDs)
	}

	var result schema.OperationResult
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON result: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got %+v", result)
	}
}

// ─── move ────────────────────────────────────────────────────────────────────

func TestMoveCommandCallsWriterMove(t *testing.T) {
	w := &stubWriterDriver{}
	configPath := writeMutationFixtures(t, w)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"move", "abc@example.com", "Archive", "--config", configPath})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("move command failed: %v", err)
	}

	if len(w.movedIDs) != 1 || w.movedIDs[0] != "abc@example.com" || w.movedDests[0] != "Archive" {
		t.Fatalf("expected move to Archive, got ids=%v dests=%v", w.movedIDs, w.movedDests)
	}

	var result schema.OperationResult
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON result: %v", err)
	}
	if result.Extra["dest_mailbox"] != "Archive" {
		t.Fatalf("expected dest_mailbox in result extra, got %+v", result)
	}
}

// ─── mark ────────────────────────────────────────────────────────────────────

func TestMarkCommandMarksRead(t *testing.T) {
	w := &stubWriterDriver{}
	configPath := writeMutationFixtures(t, w)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"mark", "abc@example.com", "--config", configPath})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mark command failed: %v", err)
	}

	if len(w.markedIDs) != 1 || w.markedIDs[0] != "abc@example.com" || !w.markedReads[0] {
		t.Fatalf("expected mark read=true, got ids=%v reads=%v", w.markedIDs, w.markedReads)
	}

	var result schema.OperationResult
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON result: %v", err)
	}
	if result.Extra["state"] != "read" {
		t.Fatalf("expected state=read, got %+v", result)
	}
}

func TestMarkCommandMarksUnread(t *testing.T) {
	w := &stubWriterDriver{}
	configPath := writeMutationFixtures(t, w)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"mark", "abc@example.com", "--config", configPath, "--unread"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mark --unread command failed: %v", err)
	}

	if len(w.markedReads) != 1 || w.markedReads[0] {
		t.Fatalf("expected mark read=false, got %v", w.markedReads)
	}

	var result schema.OperationResult
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON result: %v", err)
	}
	if result.Extra["state"] != "unread" {
		t.Fatalf("expected state=unread, got %+v", result)
	}
}

// ─── export ──────────────────────────────────────────────────────────────────

func TestExportCommandJSONL(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.json")
	store := mailindex.NewFileStore(indexPath)
	if err := store.Upsert(mailindex.IndexedMessage{
		Account: "work",
		Mailbox: "INBOX",
		ID:      "<export-1@example.com>",
		Message: schema.StandardMessage{
			Meta: schema.MessageMeta{Subject: "Export test"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"export", "--index", indexPath, "--format", "jsonl"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("export JSONL command failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 JSONL line, got %d: %s", len(lines), buf.String())
	}
	var item mailindex.IndexedMessage
	if err := json.Unmarshal([]byte(lines[0]), &item); err != nil {
		t.Fatalf("expected valid JSONL: %v", err)
	}
	if item.Message.Meta.Subject != "Export test" {
		t.Fatalf("expected correct subject, got %q", item.Message.Meta.Subject)
	}
}

func TestExportCommandCSV(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.json")
	store := mailindex.NewFileStore(indexPath)
	if err := store.Upsert(mailindex.IndexedMessage{
		Account: "work",
		Mailbox: "INBOX",
		ID:      "<export-csv@example.com>",
		Message: schema.StandardMessage{
			Meta: schema.MessageMeta{Subject: "CSV test"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"export", "--index", indexPath, "--format", "csv"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("export CSV command failed: %v", err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("expected valid CSV: %v", err)
	}
	// header row + 1 data row
	if len(records) != 2 {
		t.Fatalf("expected 2 CSV rows (header+data), got %d\n%s", len(records), buf.String())
	}
	subjectColIdx := 5 // account,mailbox,id,thread_id,from,subject
	if records[1][subjectColIdx] != "CSV test" {
		t.Fatalf("expected subject 'CSV test', got %q", records[1][subjectColIdx])
	}
}

func TestExportCommandJSONArray(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.json")
	store := mailindex.NewFileStore(indexPath)
	if err := store.Upsert(mailindex.IndexedMessage{
		Account: "work",
		Mailbox: "INBOX",
		ID:      "<export-json@example.com>",
		Message: schema.StandardMessage{
			Meta: schema.MessageMeta{Subject: "JSON array test"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"export", "--index", indexPath, "--format", "json"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("export JSON command failed: %v", err)
	}

	var items []mailindex.IndexedMessage
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &items); err != nil {
		t.Fatalf("expected valid JSON array: %v", err)
	}
	if len(items) != 1 || items[0].Message.Meta.Subject != "JSON array test" {
		t.Fatalf("unexpected export result: %+v", items)
	}
}
