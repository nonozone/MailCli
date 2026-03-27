package index

import (
	"path/filepath"
	"testing"

	"github.com/nonozone/MailCli/pkg/schema"
)

func TestFileStoreThreadsGroupsRepliesIntoSingleThread(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	root := IndexedMessage{
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
				Snippet: "Initial update",
				BodyMD:  "Initial update",
			},
		},
	}

	reply := IndexedMessage{
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
				Snippet: "Looks good",
				BodyMD:  "Looks good",
			},
		},
	}

	for _, item := range []IndexedMessage{root, reply} {
		if err := store.Upsert(item); err != nil {
			t.Fatalf("expected upsert to succeed: %v", err)
		}
	}

	threads, err := store.Threads(ThreadQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected thread aggregation to succeed: %v", err)
	}
	if len(threads) != 1 {
		t.Fatalf("expected one thread, got %d", len(threads))
	}

	thread := threads[0]
	if thread.ThreadID != "<root@example.com>" {
		t.Fatalf("expected root thread id, got %q", thread.ThreadID)
	}
	if thread.MessageCount != 2 {
		t.Fatalf("expected two messages in thread, got %d", thread.MessageCount)
	}
	if thread.LatestDate != "2026-03-27T09:00:00Z" {
		t.Fatalf("expected latest date from reply, got %q", thread.LatestDate)
	}
	if len(thread.MessageIDs) != 2 {
		t.Fatalf("expected both message ids in thread summary, got %d", len(thread.MessageIDs))
	}
	if len(thread.Participants) != 2 {
		t.Fatalf("expected deduplicated participants, got %d", len(thread.Participants))
	}
}

func TestFileStoreThreadsSupportsQueryFilteringAndRanking(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	items := []IndexedMessage{
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "invoice-root",
			Message: schema.StandardMessage{
				ID: "invoice-root",
				Meta: schema.MessageMeta{
					Subject:   "Invoice follow-up",
					Date:      "2026-03-27T10:00:00Z",
					MessageID: "<invoice-root@example.com>",
				},
				Content: schema.Content{
					Snippet: "Invoice reminder",
					BodyMD:  "Invoice reminder",
				},
			},
		},
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "random-root",
			Message: schema.StandardMessage{
				ID: "random-root",
				Meta: schema.MessageMeta{
					Subject:   "Weekly note",
					Date:      "2026-03-27T11:00:00Z",
					MessageID: "<random-root@example.com>",
				},
				Content: schema.Content{
					Snippet: "No invoice here",
					BodyMD:  "General update",
				},
			},
		},
	}

	for _, item := range items {
		if err := store.Upsert(item); err != nil {
			t.Fatalf("expected upsert to succeed: %v", err)
		}
	}

	threads, err := store.Threads(ThreadQuery{Query: "invoice", Limit: 10})
	if err != nil {
		t.Fatalf("expected thread query to succeed: %v", err)
	}
	if len(threads) != 2 {
		t.Fatalf("expected both matching threads to remain, got %d", len(threads))
	}
	if threads[0].ThreadID != "<invoice-root@example.com>" {
		t.Fatalf("expected invoice-heavy thread first, got %q", threads[0].ThreadID)
	}
	if threads[0].Score < threads[1].Score {
		t.Fatalf("expected first thread to have >= score, got %d < %d", threads[0].Score, threads[1].Score)
	}
}

func TestFileStoreThreadMessagesReturnsFullMessagesForThread(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	for _, item := range []IndexedMessage{
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
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "other-thread",
			Message: schema.StandardMessage{
				ID: "other-thread",
				Meta: schema.MessageMeta{
					Subject:   "Invoice",
					Date:      "2026-03-27T10:00:00Z",
					MessageID: "<invoice@example.com>",
				},
				Content: schema.Content{
					Snippet: "Invoice ready",
					BodyMD:  "Invoice ready",
				},
			},
		},
	} {
		if err := store.Upsert(item); err != nil {
			t.Fatalf("expected upsert to succeed: %v", err)
		}
	}

	messages, err := store.ThreadMessages(ThreadMessageQuery{
		ThreadID: "<root@example.com>",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("expected thread messages lookup to succeed: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected two thread messages, got %d", len(messages))
	}
	if messages[0].ID != "msg-root" || messages[1].ID != "msg-reply" {
		t.Fatalf("expected messages ordered by date within thread, got %q then %q", messages[0].ID, messages[1].ID)
	}
}
