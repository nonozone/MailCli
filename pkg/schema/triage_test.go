package schema

import (
	"strings"
	"testing"
)

func TestValidateTriageEnrichment(t *testing.T) {
	priorityConfidence := 0.9
	needsReply := true
	needsReplyConfidence := 0.85
	todoConfidence := 0.9
	valid := TriageEnrichment{
		Version:    "v1",
		Scope:      "thread",
		SubjectID:  "<root@example.com>",
		EvidenceID: "sha256:test-evidence",
		Source: TriageEnrichmentSource{
			Kind:     "external",
			Provider: "test-provider",
			Model:    "test-v1",
		},
		GeneratedAt: "2026-07-14T09:00:00Z",
		Summary:     "A customer is waiting for a revised quote.",
		Priority: &TriagePriority{
			Level:      "high",
			Confidence: &priorityConfidence,
			Reasons:    []string{"The requested deadline is tomorrow."},
		},
		NeedsReply: &TriageNeedsReply{
			Value:      &needsReply,
			Confidence: &needsReplyConfidence,
			Reasons:    []string{"The first message contains an unanswered request."},
		},
		Todos: []TriageTodo{{
			Text:            "Send the revised quote",
			SourceMessageID: "msg-root",
			Confidence:      &todoConfidence,
		}},
	}

	if err := ValidateTriageEnrichment(valid); err != nil {
		t.Fatalf("expected valid enrichment: %v", err)
	}
}

func TestValidateTriageEnrichmentRejectsUntraceableAssessment(t *testing.T) {
	needsReply := true
	invalidConfidence := 1.2
	invalid := TriageEnrichment{
		Version:     "v1",
		Scope:       "thread",
		SubjectID:   "<root@example.com>",
		EvidenceID:  "sha256:test-evidence",
		Source:      TriageEnrichmentSource{Kind: "external", Provider: "test-provider"},
		GeneratedAt: "2026-07-14T09:00:00Z",
		NeedsReply: &TriageNeedsReply{
			Value:      &needsReply,
			Confidence: &invalidConfidence,
		},
	}

	err := ValidateTriageEnrichment(invalid)
	if err == nil {
		t.Fatal("expected invalid enrichment to fail")
	}
	for _, want := range []string{"confidence", "reasons"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to mention %q, got %v", want, err)
		}
	}
}

func TestValidateTriageEnrichmentRequiresExplicitNeedsReplyValues(t *testing.T) {
	invalid := TriageEnrichment{
		Version:     "v1",
		Scope:       "message",
		SubjectID:   "msg-1",
		EvidenceID:  "sha256:test-evidence",
		Source:      TriageEnrichmentSource{Kind: "external", Provider: "test-provider"},
		GeneratedAt: "2026-07-14T09:00:00Z",
		NeedsReply:  &TriageNeedsReply{Reasons: []string{"No request was found."}},
	}

	err := ValidateTriageEnrichment(invalid)
	if err == nil {
		t.Fatal("expected missing needs_reply fields to fail")
	}
	for _, want := range []string{"needs_reply.value is required", "needs_reply.confidence is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to mention %q, got %v", want, err)
		}
	}
}
