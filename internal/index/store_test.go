package index

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/nonozone/MailCli/pkg/schema"
)

func TestFileStoreUpsertAndSearch(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	first := IndexedMessage{
		Account: "demo",
		Mailbox: "INBOX",
		ID:      "stub:security-reset",
		Message: schema.StandardMessage{
			ID: "stub:security-reset",
			Meta: schema.MessageMeta{
				Subject: "Reset code for your workspace",
				Date:    "2026-03-25T08:30:00Z",
				From: &schema.Address{
					Name:    "Example Security",
					Address: "security@example.com",
				},
			},
			Content: schema.Content{
				Snippet: "Use code 482991 to finish signing in.",
				BodyMD:  "Use code 482991 to finish signing in.",
			},
			Labels: []string{"security"},
		},
	}

	second := IndexedMessage{
		Account: "demo",
		Mailbox: "INBOX",
		ID:      "stub:invoice",
		Message: schema.StandardMessage{
			ID: "stub:invoice",
			Meta: schema.MessageMeta{
				Subject: "Invoice INV-2026-031 ready",
				Date:    "2026-03-24T16:10:00Z",
				From: &schema.Address{
					Name:    "Example Billing",
					Address: "billing@example.com",
				},
			},
			Content: schema.Content{
				Snippet: "Your March invoice is ready.",
				BodyMD:  "Your March invoice is ready.",
			},
			Labels: []string{"finance"},
		},
	}

	if err := store.Upsert(first); err != nil {
		t.Fatalf("expected first upsert to succeed: %v", err)
	}
	if err := store.Upsert(second); err != nil {
		t.Fatalf("expected second upsert to succeed: %v", err)
	}

	first.Message.Content.Snippet = "Updated sign-in guidance."
	if err := store.Upsert(first); err != nil {
		t.Fatalf("expected overwrite upsert to succeed: %v", err)
	}

	results, err := store.Search(SearchQuery{Query: "updated", Limit: 10})
	if err != nil {
		t.Fatalf("expected search to succeed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one updated result, got %d", len(results))
	}
	if results[0].ID != "stub:security-reset" {
		t.Fatalf("expected updated record to be returned, got %q", results[0].ID)
	}

	all, err := store.All()
	if err != nil {
		t.Fatalf("expected all to succeed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected deduplicated record count, got %d", len(all))
	}
}

func TestFileStoreJSONPathRemappedToDb(t *testing.T) {
	dir := t.TempDir()
	// Passing a .json path should be silently remapped to .db by NewFileStore.
	store := NewFileStore(filepath.Join(dir, "index.json"))
	if store.Path() != filepath.Join(dir, "index.db") {
		t.Fatalf("expected .json path to be remapped to .db, got %q", store.Path())
	}
	// Should work correctly despite unusual extension in the original arg.
	record := IndexedMessage{
		Account: "demo",
		ID:      "remap-test",
		Message: schema.StandardMessage{ID: "remap-test"},
	}
	if err := store.Upsert(record); err != nil {
		t.Fatalf("expected upsert after path remap to succeed: %v", err)
	}
	has, err := store.Has("demo", "remap-test")
	if err != nil || !has {
		t.Fatalf("expected remapped store to have upserted record: has=%v err=%v", has, err)
	}
}

func TestFileStoreHasUsesAccountAndID(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))
	record := IndexedMessage{
		Account: "demo",
		ID:      "stub:invoice",
		Message: schema.StandardMessage{ID: "stub:invoice"},
	}

	if err := store.Upsert(record); err != nil {
		t.Fatalf("expected upsert to succeed: %v", err)
	}

	has, err := store.Has("demo", "stub:invoice")
	if err != nil {
		t.Fatalf("expected has to succeed: %v", err)
	}
	if !has {
		t.Fatalf("expected stored message to be found")
	}

	missing, err := store.Has("demo", "missing")
	if err != nil {
		t.Fatalf("expected missing has check to succeed: %v", err)
	}
	if missing {
		t.Fatalf("expected missing message to be absent")
	}
}

func TestFileStoreSearchSupportsAccountAndMailboxFilters(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	for _, record := range []IndexedMessage{
		{
			Account: "work",
			Mailbox: "INBOX",
			ID:      "msg-work",
			Message: schema.StandardMessage{
				ID:   "msg-work",
				Meta: schema.MessageMeta{Subject: "Invoice from work"},
				Content: schema.Content{
					Snippet: "invoice work",
					BodyMD:  "invoice work",
				},
			},
		},
		{
			Account: "personal",
			Mailbox: "Archive",
			ID:      "msg-personal",
			Message: schema.StandardMessage{
				ID:   "msg-personal",
				Meta: schema.MessageMeta{Subject: "Invoice from personal"},
				Content: schema.Content{
					Snippet: "invoice personal",
					BodyMD:  "invoice personal",
				},
			},
		},
	} {
		if err := store.Upsert(record); err != nil {
			t.Fatalf("expected index write to succeed: %v", err)
		}
	}

	results, err := store.Search(SearchQuery{
		Query:   "invoice",
		Account: "personal",
		Mailbox: "Archive",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("expected filtered search to succeed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one filtered result, got %d", len(results))
	}
	if results[0].ID != "msg-personal" {
		t.Fatalf("expected personal record, got %q", results[0].ID)
	}
}

func TestFileStoreSearchRanksMoreRelevantResultsFirst(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	for _, record := range []IndexedMessage{
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "subject-hit",
			Message: schema.StandardMessage{
				ID: "subject-hit",
				Meta: schema.MessageMeta{
					Subject: "Invoice invoice for March",
				},
				Content: schema.Content{
					Snippet: "Billing reminder",
					BodyMD:  "March billing summary",
				},
			},
		},
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "body-hit",
			Message: schema.StandardMessage{
				ID: "body-hit",
				Meta: schema.MessageMeta{
					Subject: "Monthly update",
				},
				Content: schema.Content{
					Snippet: "General update",
					BodyMD:  "Invoice details are in the attached PDF.",
				},
			},
		},
	} {
		if err := store.Upsert(record); err != nil {
			t.Fatalf("expected index write to succeed: %v", err)
		}
	}

	results, err := store.Search(SearchQuery{Query: "invoice", Limit: 10})
	if err != nil {
		t.Fatalf("expected ranked search to succeed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two ranked results, got %d", len(results))
	}
	if results[0].ID != "subject-hit" {
		t.Fatalf("expected subject-heavy result first, got %q", results[0].ID)
	}
	if results[0].Score <= results[1].Score {
		t.Fatalf("expected first result to have higher score, got %d <= %d", results[0].Score, results[1].Score)
	}
}

func TestFileStoreSearchPrefersMessageDateOverIndexedAtWhenScoresTie(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	for _, record := range []IndexedMessage{
		{
			Account:   "demo",
			Mailbox:   "INBOX",
			ID:        "older-but-reindexed",
			IndexedAt: "2026-03-28T12:00:00Z",
			Message: schema.StandardMessage{
				ID: "older-but-reindexed",
				Meta: schema.MessageMeta{
					Subject: "Invoice ready",
					Date:    "2026-03-20T08:00:00Z",
				},
				Content: schema.Content{
					Snippet: "Invoice attached",
				},
			},
		},
		{
			Account:   "demo",
			Mailbox:   "INBOX",
			ID:        "recent-message",
			IndexedAt: "2026-03-27T12:00:00Z",
			Message: schema.StandardMessage{
				ID: "recent-message",
				Meta: schema.MessageMeta{
					Subject: "Invoice ready",
					Date:    "2026-03-27T08:00:00Z",
				},
				Content: schema.Content{
					Snippet: "Invoice attached",
				},
			},
		},
	} {
		if err := store.Upsert(record); err != nil {
			t.Fatalf("expected index write to succeed: %v", err)
		}
	}

	results, err := store.Search(SearchQuery{Query: "invoice", Limit: 10})
	if err != nil {
		t.Fatalf("expected tied-score search to succeed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two tied-score results, got %d", len(results))
	}
	if results[0].ID != "recent-message" {
		t.Fatalf("expected newer message date to rank first, got %q", results[0].ID)
	}
}

func TestFileStoreSearchSupportsThreadFilter(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	for _, record := range []IndexedMessage{
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
					Snippet: "initial",
					BodyMD:  "initial",
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
					Snippet: "reply",
					BodyMD:  "reply",
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
					Snippet: "invoice",
					BodyMD:  "invoice",
				},
			},
		},
	} {
		if err := store.Upsert(record); err != nil {
			t.Fatalf("expected index write to succeed: %v", err)
		}
	}

	results, err := store.Search(SearchQuery{
		ThreadID: "<root@example.com>",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("expected thread-filtered search to succeed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two messages in thread, got %d", len(results))
	}
	for _, result := range results {
		if result.ID == "other-thread" {
			t.Fatalf("expected other thread to be filtered out")
		}
	}
	if results[0].ThreadID != "<root@example.com>" {
		t.Fatalf("expected search results to expose thread id, got %q", results[0].ThreadID)
	}
}

func TestFileStoreSearchMatchesStructuredActionSignals(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	record := IndexedMessage{
		Account: "demo",
		Mailbox: "INBOX",
		ID:      "action-match",
		Message: schema.StandardMessage{
			ID: "action-match",
			Meta: schema.MessageMeta{
				Subject: "Security review",
			},
			Content: schema.Content{
				Snippet: "Review the latest account activity.",
				BodyMD:  "A new account activity was detected.",
			},
			Actions: []schema.Action{
				{Type: "verify_sign_in", Label: "Verify sign-in"},
			},
		},
	}

	if err := store.Upsert(record); err != nil {
		t.Fatalf("expected index write to succeed: %v", err)
	}

	results, err := store.Search(SearchQuery{Query: "verify sign-in", Limit: 10})
	if err != nil {
		t.Fatalf("expected structured action search to succeed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one structured action result, got %d", len(results))
	}
	if results[0].ID != "action-match" {
		t.Fatalf("expected action-backed record, got %q", results[0].ID)
	}
}

func TestFileStoreSearchMatchesStructuredCodeValues(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	record := IndexedMessage{
		Account: "demo",
		Mailbox: "INBOX",
		ID:      "code-match",
		Message: schema.StandardMessage{
			ID: "code-match",
			Meta: schema.MessageMeta{
				Subject: "Verification request",
			},
			Content: schema.Content{
				Snippet: "Use the code from your device.",
				BodyMD:  "Use the code from your device.",
			},
			Codes: []schema.Code{
				{Type: "verification_code", Value: "123456", Label: "Verification code"},
			},
		},
	}

	if err := store.Upsert(record); err != nil {
		t.Fatalf("expected index write to succeed: %v", err)
	}

	results, err := store.Search(SearchQuery{Query: "123456", Limit: 10})
	if err != nil {
		t.Fatalf("expected structured code search to succeed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one code-backed result, got %d", len(results))
	}
	if results[0].ID != "code-match" {
		t.Fatalf("expected code-backed record, got %q", results[0].ID)
	}
}

func TestFileStoreSearchRanksStructuredSignalsAboveBodyOnlyMentions(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.json"))

	for _, record := range []IndexedMessage{
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "structured-action",
			Message: schema.StandardMessage{
				ID: "structured-action",
				Meta: schema.MessageMeta{
					Subject: "Account review",
				},
				Content: schema.Content{
					Snippet: "Review the latest account activity.",
					BodyMD:  "A new account activity was detected.",
				},
				Actions: []schema.Action{
					{Type: "verify_sign_in", Label: "Verify sign-in"},
				},
			},
		},
		{
			Account: "demo",
			Mailbox: "INBOX",
			ID:      "body-only",
			Message: schema.StandardMessage{
				ID: "body-only",
				Meta: schema.MessageMeta{
					Subject: "General update",
				},
				Content: schema.Content{
					Snippet: "Please verify sign in details in the portal.",
					BodyMD:  "Please verify sign in details in the portal.",
				},
			},
		},
	} {
		if err := store.Upsert(record); err != nil {
			t.Fatalf("expected index write to succeed: %v", err)
		}
	}

	results, err := store.Search(SearchQuery{Query: "verify sign-in", Limit: 10})
	if err != nil {
		t.Fatalf("expected ranked structured search to succeed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two ranked results, got %d", len(results))
	}
	if results[0].ID != "structured-action" {
		t.Fatalf("expected structured action result first, got %q", results[0].ID)
	}
	if results[0].Score <= results[1].Score {
		t.Fatalf("expected structured action score to outrank body-only score, got %d <= %d", results[0].Score, results[1].Score)
	}
}

func TestBulkHasReturnsExistingIDs(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.db"))

	// Insert two messages.
	for _, id := range []string{"msg-a", "msg-b"} {
		if err := store.Upsert(IndexedMessage{
			Account: "demo", Mailbox: "INBOX", ID: id,
			Message: schema.StandardMessage{ID: id, Meta: schema.MessageMeta{Subject: "Test " + id}},
		}); err != nil {
			t.Fatalf("upsert %s: %v", id, err)
		}
	}

	existing, err := store.BulkHas("demo", []string{"msg-a", "msg-b", "msg-c"})
	if err != nil {
		t.Fatalf("BulkHas: %v", err)
	}
	if !existing["msg-a"] || !existing["msg-b"] {
		t.Fatalf("expected msg-a and msg-b to be known, got %v", existing)
	}
	if existing["msg-c"] {
		t.Fatalf("expected msg-c to be unknown, got %v", existing)
	}
}

func TestBulkUpsertWritesAllMessages(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "index.db"))

	batch := make([]IndexedMessage, 50)
	for i := range batch {
		id := fmt.Sprintf("bulk-%02d", i)
		batch[i] = IndexedMessage{
			Account: "demo", Mailbox: "INBOX", ID: id,
			Message: schema.StandardMessage{
				ID:      id,
				Meta:    schema.MessageMeta{Subject: "Bulk message " + id},
				Content: schema.Content{Snippet: "bulk content " + id},
			},
		}
	}

	if err := store.BulkUpsert(batch); err != nil {
		t.Fatalf("BulkUpsert: %v", err)
	}

	// Verify all 50 messages are searchable.
	results, err := store.Search(SearchQuery{Query: "bulk content", Limit: 100})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 50 {
		t.Fatalf("expected 50 results, got %d", len(results))
	}
}

func BenchmarkUpsertSequential(b *testing.B) {
	store := NewFileStore(filepath.Join(b.TempDir(), "index.db"))
	msgs := makeBenchMessages(b.N)
	b.ResetTimer()
	for i := range msgs {
		if err := store.Upsert(msgs[i]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBulkUpsert(b *testing.B) {
	store := NewFileStore(filepath.Join(b.TempDir(), "index.db"))
	msgs := makeBenchMessages(b.N)
	b.ResetTimer()
	if err := store.BulkUpsert(msgs); err != nil {
		b.Fatal(err)
	}
}

func makeBenchMessages(n int) []IndexedMessage {
	msgs := make([]IndexedMessage, n)
	for i := range msgs {
		id := fmt.Sprintf("bench-%06d", i)
		msgs[i] = IndexedMessage{
			Account: "bench", Mailbox: "INBOX", ID: id,
			Message: schema.StandardMessage{
				ID:      id,
				Meta:    schema.MessageMeta{Subject: "Benchmark message " + id},
				Content: schema.Content{Snippet: "bench content", BodyMD: "some body text for indexing"},
			},
		}
	}
	return msgs
}
