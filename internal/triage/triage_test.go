package triage

import (
	"strings"
	"testing"

	"github.com/nonozone/MailCli/pkg/schema"
)

func TestApplyEnrichmentRejectsUnknownSourceMessage(t *testing.T) {
	record := schema.TriageRecord{
		Version:   "v1",
		Scope:     "thread",
		SubjectID: "<root@example.com>",
		Evidence: schema.TriageEvidence{
			Source:     "deterministic",
			MessageIDs: []string{"msg-root"},
		},
	}
	enrichment := schema.TriageEnrichment{
		Version:     "v1",
		Scope:       "thread",
		SubjectID:   "<root@example.com>",
		Source:      schema.TriageEnrichmentSource{Kind: "external", Provider: "test-provider"},
		GeneratedAt: "2026-07-14T09:00:00Z",
		Todos: []schema.TriageTodo{{
			Text:            "Reply to the request",
			SourceMessageID: "msg-missing",
			Confidence:      0.9,
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
