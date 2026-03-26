package index

import (
	"os"
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

func TestFileStoreReturnsErrorForInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("expected fixture write to succeed: %v", err)
	}

	store := NewFileStore(path)
	if _, err := store.All(); err == nil {
		t.Fatalf("expected invalid index json to return an error")
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
				ID: "msg-work",
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
				ID: "msg-personal",
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
