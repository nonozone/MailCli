package cmd

import (
	"bytes"
	"strings"
	"testing"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/schema"
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
	if !strings.Contains(syncOut.String(), `"listed_count": 2`) {
		t.Fatalf("expected sync output to expose listed count, got %s", syncOut.String())
	}
	if !strings.Contains(syncOut.String(), `"fetched_count": 2`) {
		t.Fatalf("expected sync output to expose fetched count, got %s", syncOut.String())
	}
	if !strings.Contains(syncOut.String(), `"refreshed_count": 0`) {
		t.Fatalf("expected sync output to expose refreshed count, got %s", syncOut.String())
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
	if !strings.Contains(secondOut.String(), `"listed_count": 2`) {
		t.Fatalf("expected second sync to expose listed count, got %s", secondOut.String())
	}
	if !strings.Contains(secondOut.String(), `"fetched_count": 0`) {
		t.Fatalf("expected second sync to expose zero fetched count, got %s", secondOut.String())
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
	if !strings.Contains(refreshOut.String(), `"fetched_count": 2`) {
		t.Fatalf("expected refresh sync to expose fetched count, got %s", refreshOut.String())
	}
	if !strings.Contains(refreshOut.String(), `"refreshed_count": 2`) {
		t.Fatalf("expected refresh sync to expose refreshed count, got %s", refreshOut.String())
	}
}

func TestSearchCommandSupportsFullResults(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "current_account: demo\naccounts:\n  - name: demo\n    driver: stub\n")
	indexPath := writeTempFile(t, "index.json", "{}\n")

	syncCmd := NewRootCmd()
	syncCmd.SetOut(&bytes.Buffer{})
	syncCmd.SetErr(&bytes.Buffer{})
	syncCmd.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", "2"})
	if err := syncCmd.Execute(); err != nil {
		t.Fatalf("expected sync command to succeed: %v", err)
	}

	searchCmd := NewRootCmd()
	var out bytes.Buffer
	searchCmd.SetOut(&out)
	searchCmd.SetErr(&out)
	searchCmd.SetArgs([]string{"search", "--index", indexPath, "--full", "reset"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("expected full search to succeed: %v", err)
	}

	if !strings.Contains(out.String(), `"message"`) || !strings.Contains(out.String(), `"body_md": "Use code 482991 to finish signing in."`) {
		t.Fatalf("expected full search output to include indexed message content, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"thread_id":`) {
		t.Fatalf("expected full search output to expose thread id, got %s", out.String())
	}
}

func TestSearchCommandFullResultsExposeThreadIDForLegacyIndex(t *testing.T) {
	indexPath := writeTempFile(t, "index.json", `{
  "version": 1,
  "messages": [
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-root",
      "indexed_at": "2026-03-27T08:00:00Z",
      "message": {
        "id": "msg-root",
        "meta": {
          "subject": "Project update",
          "date": "2026-03-27T08:00:00Z",
          "message_id": "<root@example.com>"
        },
        "content": {
          "snippet": "Initial update",
          "body_md": "Initial update"
        }
      }
    },
    {
      "account": "demo",
      "mailbox": "INBOX",
      "id": "msg-reply",
      "indexed_at": "2026-03-27T09:00:00Z",
      "message": {
        "id": "msg-reply",
        "meta": {
          "subject": "Re: Project update",
          "date": "2026-03-27T09:00:00Z",
          "message_id": "<reply@example.com>",
          "in_reply_to": "<root@example.com>",
          "references": [
            "<root@example.com>"
          ]
        },
        "content": {
          "snippet": "Looks good",
          "body_md": "Looks good"
        }
      }
    }
  ]
}`)

	searchCmd := NewRootCmd()
	var out bytes.Buffer
	searchCmd.SetOut(&out)
	searchCmd.SetErr(&out)
	searchCmd.SetArgs([]string{"search", "--index", indexPath, "--full", "project"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("expected full search on legacy index to succeed: %v", err)
	}

	if !strings.Contains(out.String(), `"thread_id": "\u003croot@example.com\u003e"`) {
		t.Fatalf("expected legacy full search output to derive thread id, got %s", out.String())
	}
}

func TestSearchCommandSupportsAccountAndMailboxFilters(t *testing.T) {
	indexPath := writeTempFile(t, "index.json", "{}\n")
	store := mailindex.NewFileStore(indexPath)

	if err := store.Upsert(mailindex.IndexedMessage{
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
	}); err != nil {
		t.Fatalf("expected work index write to succeed: %v", err)
	}

	if err := store.Upsert(mailindex.IndexedMessage{
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
	}); err != nil {
		t.Fatalf("expected personal index write to succeed: %v", err)
	}

	searchCmd := NewRootCmd()
	var out bytes.Buffer
	searchCmd.SetOut(&out)
	searchCmd.SetErr(&out)
	searchCmd.SetArgs([]string{"search", "--index", indexPath, "--account", "personal", "--mailbox", "Archive", "invoice"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("expected filtered search to succeed: %v", err)
	}

	if strings.Contains(out.String(), `"id": "msg-work"`) {
		t.Fatalf("expected work message to be filtered out, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"id": "msg-personal"`) {
		t.Fatalf("expected personal message to remain, got %s", out.String())
	}
}

func TestSearchCommandIncludesScoreInCompactResults(t *testing.T) {
	configPath := writeTempFile(t, "config.yaml", "current_account: demo\naccounts:\n  - name: demo\n    driver: stub\n")
	indexPath := writeTempFile(t, "index.json", "{}\n")

	syncCmd := NewRootCmd()
	syncCmd.SetOut(&bytes.Buffer{})
	syncCmd.SetErr(&bytes.Buffer{})
	syncCmd.SetArgs([]string{"sync", "--config", configPath, "--index", indexPath, "--limit", "2"})
	if err := syncCmd.Execute(); err != nil {
		t.Fatalf("expected sync command to succeed: %v", err)
	}

	searchCmd := NewRootCmd()
	var out bytes.Buffer
	searchCmd.SetOut(&out)
	searchCmd.SetErr(&out)
	searchCmd.SetArgs([]string{"search", "--index", indexPath, "invoice"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("expected search command to succeed: %v", err)
	}

	if !strings.Contains(out.String(), `"score":`) {
		t.Fatalf("expected compact search output to include score, got %s", out.String())
	}
}

func TestSearchCommandSupportsThreadFilter(t *testing.T) {
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
		if err := store.Upsert(item); err != nil {
			t.Fatalf("expected upsert to succeed: %v", err)
		}
	}

	searchCmd := NewRootCmd()
	var out bytes.Buffer
	searchCmd.SetOut(&out)
	searchCmd.SetErr(&out)
	searchCmd.SetArgs([]string{"search", "--index", indexPath, "--thread", "<root@example.com>", "update"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("expected thread-filtered search to succeed: %v", err)
	}

	if strings.Contains(out.String(), `"id": "other-thread"`) {
		t.Fatalf("expected other thread to be filtered out, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"id": "msg-root"`) {
		t.Fatalf("expected thread result to include root message, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"thread_id": "\u003croot@example.com\u003e"`) {
		t.Fatalf("expected search output to expose thread id, got %s", out.String())
	}
}

func TestSearchCommandMatchesStructuredActionSignals(t *testing.T) {
	indexPath := writeTempFile(t, "index.json", "{}\n")
	store := mailindex.NewFileStore(indexPath)

	if err := store.Upsert(mailindex.IndexedMessage{
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
	}); err != nil {
		t.Fatalf("expected action-backed index write to succeed: %v", err)
	}

	searchCmd := NewRootCmd()
	var out bytes.Buffer
	searchCmd.SetOut(&out)
	searchCmd.SetErr(&out)
	searchCmd.SetArgs([]string{"search", "--index", indexPath, "verify sign-in"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("expected structured action search command to succeed: %v", err)
	}

	if !strings.Contains(out.String(), `"id": "action-match"`) {
		t.Fatalf("expected structured action search result, got %s", out.String())
	}
}
