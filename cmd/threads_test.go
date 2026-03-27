package cmd

import (
	"bytes"
	"strings"
	"testing"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestThreadsCommandListsLocalThreadSummaries(t *testing.T) {
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

	if !strings.Contains(out.String(), `"thread_id": "\u003croot@example.com\u003e"`) {
		t.Fatalf("expected thread id in output, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"message_count": 2`) {
		t.Fatalf("expected message count in output, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"last_message_preview": "Looks good"`) {
		t.Fatalf("expected last message preview in output, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"has_codes": true`) {
		t.Fatalf("expected has_codes in output, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"code_count": 1`) {
		t.Fatalf("expected code_count in output, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"action_count": 2`) {
		t.Fatalf("expected action_count in output, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"participant_count": 2`) {
		t.Fatalf("expected participant_count in output, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"action_types": [`) {
		t.Fatalf("expected action_types in output, got %s", out.String())
	}
}

func TestThreadsCommandSupportsAccountFilter(t *testing.T) {
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
					Subject:   "Work thread",
					Date:      "2026-03-27T08:00:00Z",
					MessageID: "<work-root@example.com>",
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
					Subject:   "Personal thread",
					Date:      "2026-03-27T09:00:00Z",
					MessageID: "<personal-root@example.com>",
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
	cmd.SetArgs([]string{"threads", "--index", indexPath, "--account", "personal"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected filtered threads command to succeed: %v", err)
	}

	if strings.Contains(out.String(), `"thread_id": "\u003cwork-root@example.com\u003e"`) {
		t.Fatalf("expected work thread to be filtered out, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"thread_id": "\u003cpersonal-root@example.com\u003e"`) {
		t.Fatalf("expected personal thread to remain, got %s", out.String())
	}
}

func TestThreadsCommandSupportsTriageFilters(t *testing.T) {
	indexPath := writeTempFile(t, "index.json", "{}\n")
	store := mailindex.NewFileStore(indexPath)

	for _, item := range []mailindex.IndexedMessage{
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "verification-thread",
			Message: schema.StandardMessage{
				ID: "verification-thread",
				Meta: schema.MessageMeta{
					Subject:   "Verify sign in",
					Date:      "2026-03-27T09:00:00Z",
					MessageID: "<verification@example.com>",
				},
				Content: schema.Content{
					Category: "verification",
					Snippet:  "Verification code",
					BodyMD:   "Verification code",
				},
				Actions: []schema.Action{{Type: "verify_sign_in"}},
				Codes:   []schema.Code{{Type: "otp", Value: "123456"}},
			},
		},
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "invoice-thread",
			Message: schema.StandardMessage{
				ID: "invoice-thread",
				Meta: schema.MessageMeta{
					Subject:   "Invoice ready",
					Date:      "2026-03-27T10:00:00Z",
					MessageID: "<invoice@example.com>",
				},
				Content: schema.Content{
					Category: "finance",
					Snippet:  "Invoice reminder",
					BodyMD:   "Invoice reminder",
				},
				Actions: []schema.Action{{Type: "view_invoice"}},
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
	cmd.SetArgs([]string{"threads", "--index", indexPath, "--has-codes", "--category", "verification", "--action", "verify_sign_in"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected filtered threads command to succeed: %v", err)
	}

	if strings.Contains(out.String(), `"thread_id": "\u003cinvoice@example.com\u003e"`) {
		t.Fatalf("expected invoice thread to be filtered out, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"thread_id": "\u003cverification@example.com\u003e"`) {
		t.Fatalf("expected verification thread to remain, got %s", out.String())
	}
}
