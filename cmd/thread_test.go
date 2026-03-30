package cmd

import (
	"bytes"
	"strings"
	"testing"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestThreadCommandReturnsFullMessagesForThread(t *testing.T) {
	indexPath := writeTempFile(t, "index.json", "{}\n")
	store := mailindex.NewFileStore(indexPath)

	for _, item := range []mailindex.IndexedMessage{
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "msg-root",
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
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "msg-reply",
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

	if !strings.Contains(out.String(), `"id": "msg-root"`) || !strings.Contains(out.String(), `"id": "msg-reply"`) {
		t.Fatalf("expected thread command to return both local messages, got %s", out.String())
	}
}

func TestThreadCommandExposesThreadIDForStoredMessages(t *testing.T) {
	indexPath := writeTempFile(t, "index.db", "")
	store := mailindex.NewFileStore(indexPath)

	for _, item := range []mailindex.IndexedMessage{
		{
			Account:   "demo",
			Mailbox:   "INBOX",
			ID:        "msg-root",
			IndexedAt: "2026-03-27T08:00:00Z",
			Message: schema.StandardMessage{
				ID: "msg-root",
				Meta: schema.MessageMeta{
					Subject:   "Project update",
					Date:      "2026-03-27T08:00:00Z",
					MessageID: "<root@example.com>",
				},
				Content: schema.Content{Snippet: "Initial update", BodyMD: "Initial update"},
			},
		},
		{
			Account:   "demo",
			Mailbox:   "INBOX",
			ID:        "msg-reply",
			IndexedAt: "2026-03-27T09:00:00Z",
			Message: schema.StandardMessage{
				ID: "msg-reply",
				Meta: schema.MessageMeta{
					Subject:    "Re: Project update",
					Date:       "2026-03-27T09:00:00Z",
					MessageID:  "<reply@example.com>",
					InReplyTo:  "<root@example.com>",
					References: []string{"<root@example.com>"},
				},
				Content: schema.Content{Snippet: "Looks good", BodyMD: "Looks good"},
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

	if !strings.Contains(out.String(), `"thread_id": "\u003croot@example.com\u003e"`) {
		t.Fatalf("expected thread output to derive thread id, got %s", out.String())
	}
}

func TestThreadCommandSupportsAccountFilter(t *testing.T) {
	indexPath := writeTempFile(t, "index.json", "{}\n")
	store := mailindex.NewFileStore(indexPath)

	for _, item := range []mailindex.IndexedMessage{
		{
			Account: "work",
			Mailbox: "INBOX",
			ID:      "work-root",
			Message: schema.StandardMessage{
				ID: "work-root",
				Meta: schema.MessageMeta{
					Subject:   "Project update",
					Date:      "2026-03-27T08:00:00Z",
					MessageID: "<root@example.com>",
				},
			},
		},
		{
			Account: "personal",
			Mailbox: "INBOX",
			ID:      "personal-root",
			Message: schema.StandardMessage{
				ID: "personal-root",
				Meta: schema.MessageMeta{
					Subject:   "Project update",
					Date:      "2026-03-27T09:00:00Z",
					MessageID: "<reply@example.com>",
					InReplyTo: "<root@example.com>",
					References: []string{
						"<root@example.com>",
					},
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
	cmd.SetArgs([]string{"thread", "--index", indexPath, "--account", "personal", "<root@example.com>"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected filtered thread command to succeed: %v", err)
	}

	if strings.Contains(out.String(), `"id": "work-root"`) {
		t.Fatalf("expected work message to be filtered out, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"id": "personal-root"`) {
		t.Fatalf("expected personal message to remain, got %s", out.String())
	}
}
