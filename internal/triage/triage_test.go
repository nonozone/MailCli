package triage

import (
	"strings"
	"testing"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestApplyEnrichmentRejectsUnknownSourceMessage(t *testing.T) {
	todoConfidence := 0.9
	record := schema.TriageRecord{
		Version:    "v1",
		Scope:      "thread",
		SubjectID:  "<root@example.com>",
		EvidenceID: "sha256:test-evidence",
		Evidence: schema.TriageEvidence{
			Source:     "deterministic",
			MessageIDs: []string{"msg-root"},
		},
	}
	enrichment := schema.TriageEnrichment{
		Version:     "v1",
		Scope:       "thread",
		SubjectID:   "<root@example.com>",
		EvidenceID:  "sha256:test-evidence",
		Source:      schema.TriageEnrichmentSource{Kind: "external", Provider: "test-provider"},
		GeneratedAt: "2026-07-14T09:00:00Z",
		Todos: []schema.TriageTodo{{
			Text:            "Reply to the request",
			SourceMessageID: "msg-missing",
			Confidence:      &todoConfidence,
		}},
	}

	err := ApplyEnrichment(&record, enrichment)
	if err == nil {
		t.Fatal("expected unknown source message to fail")
	}
	if !strings.Contains(err.Error(), "not present in triage evidence") {
		t.Fatalf("expected traceability error, got %v", err)
	}
}

func TestApplyEnrichmentRejectsStaleEvidence(t *testing.T) {
	record := schema.TriageRecord{
		Version:    "v1",
		Scope:      "thread",
		SubjectID:  "<root@example.com>",
		EvidenceID: "sha256:current",
	}
	enrichment := schema.TriageEnrichment{
		Version:     "v1",
		Scope:       "thread",
		SubjectID:   "<root@example.com>",
		EvidenceID:  "sha256:stale",
		Source:      schema.TriageEnrichmentSource{Kind: "external", Provider: "test-provider"},
		GeneratedAt: "2026-07-14T09:00:00Z",
		Summary:     "This result was generated from old evidence.",
	}

	err := ApplyEnrichment(&record, enrichment)
	if err == nil {
		t.Fatal("expected stale enrichment to fail")
	}
	if !strings.Contains(err.Error(), "does not match current triage evidence_id") {
		t.Fatalf("expected stale evidence error, got %v", err)
	}
}

func TestFromThreadDeduplicatesParticipantsByAddress(t *testing.T) {
	record, err := FromThread("<root@example.com>", []mailindex.IndexedMessage{
		{
			Account: "demo",
			ID:      "msg-1",
			Message: schema.StandardMessage{
				Meta: schema.MessageMeta{
					MessageID: "<root@example.com>",
					Date:      "2026-07-14T08:00:00Z",
					From:      &schema.Address{Address: "alice@example.com"},
				},
			},
		},
		{
			Account: "demo",
			ID:      "msg-2",
			Message: schema.StandardMessage{
				Meta: schema.MessageMeta{
					MessageID: "<reply@example.com>",
					InReplyTo: "<root@example.com>",
					Date:      "2026-07-14T09:00:00Z",
					From:      &schema.Address{Name: "Alice", Address: "ALICE@example.com"},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Evidence.ParticipantCount != 1 {
		t.Fatalf("expected one participant identity, got %+v", record.Evidence.Participants)
	}
	if record.Evidence.Participants[0] != "Alice <ALICE@example.com>" {
		t.Fatalf("expected richer display name to win, got %+v", record.Evidence.Participants)
	}
}
