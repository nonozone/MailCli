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
				Category: "operations",
				Snippet:  "Initial update",
				BodyMD:   "Initial update",
			},
			Actions: []schema.Action{
				{Type: "view_online"},
			},
			Labels: []string{"project"},
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
				Category: "verification",
				Snippet:  "Looks good",
				BodyMD:   "Looks good",
			},
			Actions: []schema.Action{
				{Type: "verify_sign_in"},
			},
			Codes: []schema.Code{
				{Type: "otp", Value: "123456"},
			},
			Labels: []string{"security"},
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
	if thread.LastMessageID != "msg-reply" {
		t.Fatalf("expected last message id from latest reply, got %q", thread.LastMessageID)
	}
	if thread.LastMessageFrom != "Bob <bob@example.com>" {
		t.Fatalf("expected last message sender from latest reply, got %q", thread.LastMessageFrom)
	}
	if thread.LastMessagePreview != "Looks good" {
		t.Fatalf("expected preview from latest reply, got %q", thread.LastMessagePreview)
	}
	if !containsString(thread.Categories, "operations") || !containsString(thread.Categories, "verification") {
		t.Fatalf("expected aggregated categories, got %#v", thread.Categories)
	}
	if !containsString(thread.ActionTypes, "view_online") || !containsString(thread.ActionTypes, "verify_sign_in") {
		t.Fatalf("expected aggregated action types, got %#v", thread.ActionTypes)
	}
	if !containsString(thread.Labels, "project") || !containsString(thread.Labels, "security") {
		t.Fatalf("expected aggregated labels, got %#v", thread.Labels)
	}
	if !thread.HasCodes {
		t.Fatalf("expected thread to expose has_codes")
	}
	if len(thread.MessageIDs) != 2 {
		t.Fatalf("expected both message ids in thread summary, got %d", len(thread.MessageIDs))
	}
	if len(thread.Participants) != 2 {
		t.Fatalf("expected deduplicated participants, got %d", len(thread.Participants))
	}
	if thread.ParticipantCount != 2 {
		t.Fatalf("expected participant count to expose deduplicated total, got %d", thread.ParticipantCount)
	}
	if thread.ActionCount != 2 {
		t.Fatalf("expected action count to expose unique aggregated total, got %d", thread.ActionCount)
	}
	if thread.CodeCount != 1 {
		t.Fatalf("expected code count to expose extracted code total, got %d", thread.CodeCount)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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

func TestFileStoreThreadsWithoutQueryPreferRecentActivityOverThreadSize(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	for _, item := range []IndexedMessage{
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "older-root",
			Message: schema.StandardMessage{
				ID: "older-root",
				Meta: schema.MessageMeta{
					Subject:   "Long running thread",
					Date:      "2026-03-20T08:00:00Z",
					MessageID: "<older-root@example.com>",
				},
				Content: schema.Content{
					Snippet: "First update",
				},
			},
		},
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "older-reply",
			Message: schema.StandardMessage{
				ID: "older-reply",
				Meta: schema.MessageMeta{
					Subject:   "Re: Long running thread",
					Date:      "2026-03-21T08:00:00Z",
					MessageID: "<older-reply@example.com>",
					InReplyTo: "<older-root@example.com>",
					References: []string{
						"<older-root@example.com>",
					},
				},
				Content: schema.Content{
					Snippet: "Second update",
				},
			},
		},
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "recent-root",
			Message: schema.StandardMessage{
				ID: "recent-root",
				Meta: schema.MessageMeta{
					Subject:   "Fresh thread",
					Date:      "2026-03-27T09:00:00Z",
					MessageID: "<recent-root@example.com>",
				},
				Content: schema.Content{
					Snippet: "Fresh update",
				},
			},
		},
	} {
		if err := store.Upsert(item); err != nil {
			t.Fatalf("expected upsert to succeed: %v", err)
		}
	}

	threads, err := store.Threads(ThreadQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected unfiltered thread list to succeed: %v", err)
	}
	if len(threads) != 2 {
		t.Fatalf("expected two threads, got %d", len(threads))
	}
	if threads[0].ThreadID != "<recent-root@example.com>" {
		t.Fatalf("expected most recent thread first, got %q", threads[0].ThreadID)
	}
	if threads[0].Score != 0 || threads[1].Score != 0 {
		t.Fatalf("expected queryless thread scores to stay neutral, got %d and %d", threads[0].Score, threads[1].Score)
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

func TestFileStoreThreadsSupportsTriageFilters(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	for _, item := range []IndexedMessage{
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

	verification, err := store.Threads(ThreadQuery{
		HasCodes: true,
		Category: "verification",
		Action:   "verify_sign_in",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("expected filtered threads to succeed: %v", err)
	}
	if len(verification) != 1 {
		t.Fatalf("expected one verification thread, got %d", len(verification))
	}
	if verification[0].ThreadID != "<verification@example.com>" {
		t.Fatalf("expected verification thread, got %q", verification[0].ThreadID)
	}
}
