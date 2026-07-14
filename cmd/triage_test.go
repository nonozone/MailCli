package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestTriageMessageCommandJSONSnapshot(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"triage", "message", "../testdata/emails/mime_attachment.eml"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected triage message to succeed: %v\n%s", err, out.String())
	}

	assertJSONSnapshot(t, "triage_message.json", out.Bytes())
}

func TestTriageThreadCommandPreservesAllMessageFacts(t *testing.T) {
	indexPath := writeTriageThreadFixture(t)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"triage", "thread", "--index", indexPath, "<root@example.com>"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected triage thread to succeed: %v\n%s", err, out.String())
	}

	assertJSONSnapshot(t, "triage_thread.json", out.Bytes())
}

func TestTriageMessageCommandMergesValidatedExternalEnrichment(t *testing.T) {
	enrichmentPath := writeTempFile(t, "enrichment.json", `{
  "version": "v1",
  "scope": "message",
  "subject_id": "<mime-attachment-123@example.com>",
  "source": {"kind": "external", "provider": "test-provider", "model": "test-v1"},
  "generated_at": "2026-07-14T09:00:00Z",
  "summary": "An invoice PDF needs review.",
  "priority": {"level": "high", "confidence": 0.9, "reasons": ["The invoice is attached."]},
  "needs_reply": {"value": false, "confidence": 0.8, "reasons": ["No reply request is present."]},
  "todos": [{"text": "Review the invoice", "source_message_id": "<mime-attachment-123@example.com>", "confidence": 0.9}]
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"triage", "message", "--enrichment", enrichmentPath, "../testdata/emails/mime_attachment.eml"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected enrichment merge to succeed: %v\n%s", err, out.String())
	}

	var result schema.TriageRecord
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("expected triage JSON: %v", err)
	}
	if result.Enrichment == nil || result.Enrichment.Priority == nil {
		t.Fatalf("expected validated enrichment, got %+v", result)
	}
	if result.Enrichment.NeedsReply == nil || result.Enrichment.NeedsReply.Value {
		t.Fatalf("expected explicit needs_reply=false, got %+v", result.Enrichment.NeedsReply)
	}
}

func TestTriageMessageCommandRejectsEnrichmentForDifferentSubject(t *testing.T) {
	enrichmentPath := writeTempFile(t, "enrichment.json", `{
  "version": "v1",
  "scope": "message",
  "subject_id": "<different@example.com>",
  "source": {"kind": "external", "provider": "test-provider"},
  "generated_at": "2026-07-14T09:00:00Z",
  "summary": "Wrong message."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"triage", "message", "--enrichment", enrichmentPath, "../testdata/emails/mime_attachment.eml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected mismatched enrichment to fail")
	}
	if !strings.Contains(err.Error(), "subject_id") {
		t.Fatalf("expected subject_id mismatch, got %v", err)
	}
}

func writeTriageThreadFixture(t *testing.T) string {
	t.Helper()
	indexPath := writeTempFile(t, "triage-index.json", "{}\n")
	store := mailindex.NewFileStore(indexPath)

	items := []mailindex.IndexedMessage{
		{
			Account: "demo", Mailbox: "INBOX", ID: "msg-root", IndexedAt: "2026-07-14T08:00:00Z",
			Message: schema.StandardMessage{
				ID: "msg-root",
				Meta: schema.MessageMeta{
					MessageID: "<root@example.com>", Subject: "Revised quote", Date: "2026-07-14T08:00:00Z",
					From: &schema.Address{Name: "Alice", Address: "alice@example.com"},
					To:   []schema.Address{{Name: "Support", Address: "support@example.com"}},
				},
				Content: schema.Content{Snippet: "Could you send the revised quote by Friday?", Category: "sales"},
				Actions: []schema.Action{{Type: "view_invoice", Label: "View quote"}},
			},
		},
		{
			Account: "demo", Mailbox: "INBOX", ID: "msg-ack", IndexedAt: "2026-07-14T09:00:00Z",
			Message: schema.StandardMessage{
				ID: "msg-ack",
				Meta: schema.MessageMeta{
					MessageID: "<ack@example.com>", InReplyTo: "<root@example.com>", References: []string{"<root@example.com>"},
					Subject: "Re: Revised quote", Date: "2026-07-14T09:00:00Z",
					From: &schema.Address{Name: "Support", Address: "support@example.com"},
					To:   []schema.Address{{Name: "Alice", Address: "alice@example.com"}},
				},
				Content: schema.Content{Snippet: "I will check with the team.", Category: "sales"},
			},
		},
		{
			Account: "demo", Mailbox: "INBOX", ID: "msg-detail", IndexedAt: "2026-07-14T10:00:00Z",
			Message: schema.StandardMessage{
				ID: "msg-detail",
				Meta: schema.MessageMeta{
					MessageID: "<detail@example.com>", InReplyTo: "<ack@example.com>", References: []string{"<root@example.com>", "<ack@example.com>"},
					Subject: "Re: Revised quote", Date: "2026-07-14T10:00:00Z",
					From: &schema.Address{Name: "Alice", Address: "alice@example.com"},
					To:   []schema.Address{{Name: "Support", Address: "support@example.com"}},
				},
				Content:     schema.Content{Snippet: "The requested quantities are in the attached sheet.", Category: "sales"},
				Attachments: []schema.InboundAttachment{{PartID: "2", Filename: "quantities.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", SizeBytes: 128}},
			},
		},
		{
			Account: "demo", Mailbox: "INBOX", ID: "msg-last", IndexedAt: "2026-07-14T11:00:00Z",
			Message: schema.StandardMessage{
				ID: "msg-last",
				Meta: schema.MessageMeta{
					MessageID: "<last@example.com>", InReplyTo: "<detail@example.com>", References: []string{"<root@example.com>", "<ack@example.com>", "<detail@example.com>"},
					Subject: "Re: Revised quote", Date: "2026-07-14T11:00:00Z",
					From: &schema.Address{Name: "Support", Address: "support@example.com"},
					To:   []schema.Address{{Name: "Alice", Address: "alice@example.com"}},
				},
				Content: schema.Content{Snippet: "I am still looking into it.", Category: "sales"},
			},
		},
	}

	for _, item := range items {
		if err := store.Upsert(item); err != nil {
			t.Fatalf("expected triage fixture upsert: %v", err)
		}
	}
	return indexPath
}
