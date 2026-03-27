package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nonozone/MailCli/internal/config"
	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestParseCommandJSONSnapshot(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"parse", "../testdata/emails/plaintext.eml"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected parse command to succeed: %v", err)
	}

	assertJSONSnapshot(t, "parse_plaintext.json", out.Bytes())
}

func TestGetCommandJSONSnapshot(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: local\naccounts:\n  - name: local\n    driver: fake\n")

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

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected get command to succeed: %v", err)
	}

	assertJSONSnapshot(t, "get_message.json", out.Bytes())
}

func TestSearchCommandJSONSnapshot(t *testing.T) {
	indexPath := writeTempFile(t, "index.json", "{}\n")
	store := mailindex.NewFileStore(indexPath)
	if err := store.Upsert(mailindex.IndexedMessage{
		Account:   "demo",
		Mailbox:   "INBOX",
		ID:        "stub:invoice",
		IndexedAt: "2026-03-27T10:00:00Z",
		Message: schema.StandardMessage{
			ID: "stub:invoice",
			Meta: schema.MessageMeta{
				From: &schema.Address{
					Name:    "Example Billing",
					Address: "billing@example.com",
				},
				Subject:   "Invoice INV-2026-031 ready",
				Date:      "2026-03-24T16:10:00Z",
				MessageID: "<stub-invoice@example.com>",
			},
			Content: schema.Content{
				Format:  "markdown",
				Snippet: "Your **March invoice** is ready.",
				BodyMD:  "Your **March invoice** is ready.",
			},
		},
	}); err != nil {
		t.Fatalf("expected indexed message write to succeed: %v", err)
	}

	searchCmd := NewRootCmd()
	var out bytes.Buffer
	searchCmd.SetOut(&out)
	searchCmd.SetErr(&out)
	searchCmd.SetArgs([]string{"search", "--index", indexPath, "invoice"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("expected search command to succeed: %v", err)
	}

	assertJSONSnapshot(t, "search_compact.json", out.Bytes())
}

func TestThreadsCommandJSONSnapshot(t *testing.T) {
	indexPath := writeTempFile(t, "index.json", "{}\n")
	store := mailindex.NewFileStore(indexPath)

	for _, item := range []mailindex.IndexedMessage{
		{
			Account:   "demo",
			Mailbox:   "INBOX",
			ID:        "msg-root",
			IndexedAt: "2026-03-27T08:05:00Z",
			Message: schema.StandardMessage{
				ID: "msg-root",
				Meta: schema.MessageMeta{
					Subject:   "Project update",
					Date:      "2026-03-27T08:00:00Z",
					MessageID: "<root@example.com>",
					From: &schema.Address{
						Name:    "Alice",
						Address: "alice@example.com",
					},
					To: []schema.Address{{Name: "Bob", Address: "bob@example.com"}},
				},
				Content: schema.Content{
					Category: "operations",
					Snippet:  "Initial update",
					BodyMD:   "Initial update",
				},
				Actions: []schema.Action{{Type: "view_online"}},
				Labels:  []string{"project"},
			},
		},
		{
			Account:   "demo",
			Mailbox:   "INBOX",
			ID:        "msg-reply",
			IndexedAt: "2026-03-27T09:05:00Z",
			Message: schema.StandardMessage{
				ID: "msg-reply",
				Meta: schema.MessageMeta{
					Subject:   "Re: Project update",
					Date:      "2026-03-27T09:00:00Z",
					MessageID: "<reply@example.com>",
					InReplyTo: "<root@example.com>",
					References: []string{
						"<root@example.com>",
					},
					From: &schema.Address{
						Name:    "Bob",
						Address: "bob@example.com",
					},
					To: []schema.Address{{Name: "Alice", Address: "alice@example.com"}},
				},
				Content: schema.Content{
					Category: "verification",
					Snippet:  "Looks good",
					BodyMD:   "Looks good",
				},
				Actions: []schema.Action{{Type: "verify_sign_in"}},
				Codes:   []schema.Code{{Type: "otp", Value: "123456"}},
				Labels:  []string{"security"},
			},
		},
	} {
		if err := store.Upsert(item); err != nil {
			t.Fatalf("expected upsert to succeed: %v", err)
		}
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"threads", "--index", indexPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected threads command to succeed: %v", err)
	}

	assertJSONSnapshot(t, "threads_summary.json", out.Bytes())
}

func TestThreadCommandJSONSnapshot(t *testing.T) {
	indexPath := writeTempFile(t, "index.json", "{}\n")
	store := mailindex.NewFileStore(indexPath)

	for _, item := range []mailindex.IndexedMessage{
		{
			Account:   "demo",
			Mailbox:   "INBOX",
			ID:        "msg-root",
			IndexedAt: "2026-03-27T08:05:00Z",
			Message: schema.StandardMessage{
				ID: "msg-root",
				Meta: schema.MessageMeta{
					Subject:   "Project update",
					Date:      "2026-03-27T08:00:00Z",
					MessageID: "<root@example.com>",
				},
				Content: schema.Content{
					Snippet: "Initial update",
					BodyMD:  "Initial update",
				},
			},
		},
		{
			Account:   "demo",
			Mailbox:   "INBOX",
			ID:        "msg-reply",
			IndexedAt: "2026-03-27T09:05:00Z",
			Message: schema.StandardMessage{
				ID: "msg-reply",
				Meta: schema.MessageMeta{
					Subject:   "Re: Project update",
					Date:      "2026-03-27T09:00:00Z",
					MessageID: "<reply@example.com>",
					InReplyTo: "<root@example.com>",
					References: []string{
						"<root@example.com>",
					},
				},
				Content: schema.Content{
					Snippet: "Looks good",
					BodyMD:  "Looks good",
				},
			},
		},
	} {
		if err := store.Upsert(item); err != nil {
			t.Fatalf("expected upsert to succeed: %v", err)
		}
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"thread", "--index", indexPath, "<root@example.com>"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected thread command to succeed: %v", err)
	}

	assertJSONSnapshot(t, "thread_messages.json", out.Bytes())
}

func assertJSONSnapshot(t *testing.T, fixture string, actual []byte) {
	t.Helper()

	expectedPath := filepath.Join("testdata", "snapshots", fixture)
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("expected snapshot fixture %s: %v", expectedPath, err)
	}

	if got, want := canonicalizeJSON(t, actual), canonicalizeJSON(t, expected); got != want {
		t.Fatalf("json snapshot mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", fixture, got, want)
	}
}

func canonicalizeJSON(t *testing.T, data []byte) string {
	t.Helper()

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("expected valid json: %v", err)
	}

	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("expected canonical json marshal to succeed: %v", err)
	}

	return string(out)
}
