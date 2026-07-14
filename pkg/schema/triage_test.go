package schema

import (
	"strings"
	"testing"
)

func TestValidateTriageEnrichment(t *testing.T) {
	valid := TriageEnrichment{
		Version:   "v1",
		Scope:     "thread",
		SubjectID: "<root@example.com>",
		Source: TriageEnrichmentSource{
			Kind:     "external",
			Provider: "test-provider",
			Model:    "test-v1",
		},
		GeneratedAt: "2026-07-14T09:00:00Z",
		Summary:     "A customer is waiting for a revised quote.",
		Priority: &TriagePriority{
			Level:      "high",
			Confidence: 0.9,
			Reasons:    []string{"The requested deadline is tomorrow."},
		},
		NeedsReply: &TriageNeedsReply{
			Value:      true,
			Confidence: 0.85,
			Reasons:    []string{"The first message contains an unanswered request."},
		},
		Todos: []TriageTodo{{
			Text:            "Send the revised quote",
			SourceMessageID: "msg-root",
			Confidence:      0.9,
		}},
	}

	if err := ValidateTriageEnrichment(valid); err != nil {
		t.Fatalf("expected valid enrichment: %v", err)
	}
}

func TestValidateTriageEnrichmentRejectsUntraceableAssessment(t *testing.T) {
	invalid := TriageEnrichment{
		Version:     "v1",
		Scope:       "thread",
		SubjectID:   "<root@example.com>",
		Source:      TriageEnrichmentSource{Kind: "external", Provider: "test-provider"},
		GeneratedAt: "2026-07-14T09:00:00Z",
		NeedsReply: &TriageNeedsReply{
			Value:      true,
			Confidence: 1.2,
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
