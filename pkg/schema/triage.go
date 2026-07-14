package schema

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const TriageContractVersion = "v1"

// TriageRecord keeps deterministic evidence separate from optional heuristic
// or external enrichment. Core MailCLI commands only create Evidence.
type TriageRecord struct {
	Version    string            `json:"version" yaml:"version"`
	Scope      string            `json:"scope" yaml:"scope"`
	SubjectID  string            `json:"subject_id" yaml:"subject_id"`
	Account    string            `json:"account,omitempty" yaml:"account,omitempty"`
	Mailbox    string            `json:"mailbox,omitempty" yaml:"mailbox,omitempty"`
	Evidence   TriageEvidence    `json:"evidence" yaml:"evidence"`
	Enrichment *TriageEnrichment `json:"enrichment,omitempty" yaml:"enrichment,omitempty"`
}

type TriageEvidence struct {
	Source             string              `json:"source" yaml:"source"`
	MessageCount       int                 `json:"message_count" yaml:"message_count"`
	MessageIDs         []string            `json:"message_ids,omitempty" yaml:"message_ids,omitempty"`
	ParticipantCount   int                 `json:"participant_count" yaml:"participant_count"`
	Participants       []string            `json:"participants,omitempty" yaml:"participants,omitempty"`
	Categories         []string            `json:"categories,omitempty" yaml:"categories,omitempty"`
	Labels             []string            `json:"labels,omitempty" yaml:"labels,omitempty"`
	LatestDate         string              `json:"latest_date,omitempty" yaml:"latest_date,omitempty"`
	LastMessageID      string              `json:"last_message_id,omitempty" yaml:"last_message_id,omitempty"`
	LastMessageFrom    string              `json:"last_message_from,omitempty" yaml:"last_message_from,omitempty"`
	HasActions         bool                `json:"has_actions" yaml:"has_actions"`
	ActionCount        int                 `json:"action_count" yaml:"action_count"`
	ActionTypes        []string            `json:"action_types,omitempty" yaml:"action_types,omitempty"`
	HasCodes           bool                `json:"has_codes" yaml:"has_codes"`
	CodeCount          int                 `json:"code_count" yaml:"code_count"`
	HasAttachments     bool                `json:"has_attachments" yaml:"has_attachments"`
	AttachmentCount    int                 `json:"attachment_count" yaml:"attachment_count"`
	AutoSubmittedCount int                 `json:"auto_submitted_count" yaml:"auto_submitted_count"`
	ErrorCount         int                 `json:"error_count" yaml:"error_count"`
	Messages           []TriageMessageFact `json:"messages,omitempty" yaml:"messages,omitempty"`
}

// TriageMessageFact is compact deterministic context. It intentionally keeps
// one entry per message so an earlier request is not hidden by the latest
// reply. Callers that need full bodies should also load mailcli thread output.
type TriageMessageFact struct {
	ID              string   `json:"id" yaml:"id"`
	From            string   `json:"from,omitempty" yaml:"from,omitempty"`
	To              []string `json:"to,omitempty" yaml:"to,omitempty"`
	Subject         string   `json:"subject,omitempty" yaml:"subject,omitempty"`
	Date            string   `json:"date,omitempty" yaml:"date,omitempty"`
	Snippet         string   `json:"snippet,omitempty" yaml:"snippet,omitempty"`
	AutoSubmitted   bool     `json:"auto_submitted" yaml:"auto_submitted"`
	ActionCount     int      `json:"action_count" yaml:"action_count"`
	ActionTypes     []string `json:"action_types,omitempty" yaml:"action_types,omitempty"`
	CodeCount       int      `json:"code_count" yaml:"code_count"`
	AttachmentCount int      `json:"attachment_count" yaml:"attachment_count"`
	HasError        bool     `json:"has_error" yaml:"has_error"`
}

type TriageEnrichment struct {
	Version     string                 `json:"version" yaml:"version"`
	Scope       string                 `json:"scope" yaml:"scope"`
	SubjectID   string                 `json:"subject_id" yaml:"subject_id"`
	Source      TriageEnrichmentSource `json:"source" yaml:"source"`
	GeneratedAt string                 `json:"generated_at" yaml:"generated_at"`
	Summary     string                 `json:"summary,omitempty" yaml:"summary,omitempty"`
	Priority    *TriagePriority        `json:"priority,omitempty" yaml:"priority,omitempty"`
	NeedsReply  *TriageNeedsReply      `json:"needs_reply,omitempty" yaml:"needs_reply,omitempty"`
	Deadlines   []TriageDeadline       `json:"deadlines,omitempty" yaml:"deadlines,omitempty"`
	Todos       []TriageTodo           `json:"todos,omitempty" yaml:"todos,omitempty"`
}

type TriageEnrichmentSource struct {
	Kind     string `json:"kind" yaml:"kind"`
	Provider string `json:"provider" yaml:"provider"`
	Model    string `json:"model,omitempty" yaml:"model,omitempty"`
}

type TriagePriority struct {
	Level      string   `json:"level" yaml:"level"`
	Confidence float64  `json:"confidence" yaml:"confidence"`
	Reasons    []string `json:"reasons" yaml:"reasons"`
}

type TriageNeedsReply struct {
	Value      bool     `json:"value" yaml:"value"`
	Confidence float64  `json:"confidence" yaml:"confidence"`
	Reasons    []string `json:"reasons" yaml:"reasons"`
}

type TriageDeadline struct {
	Text            string  `json:"text" yaml:"text"`
	At              string  `json:"at,omitempty" yaml:"at,omitempty"`
	SourceMessageID string  `json:"source_message_id" yaml:"source_message_id"`
	Confidence      float64 `json:"confidence" yaml:"confidence"`
}

type TriageTodo struct {
	Text            string  `json:"text" yaml:"text"`
	DueAt           string  `json:"due_at,omitempty" yaml:"due_at,omitempty"`
	SourceMessageID string  `json:"source_message_id" yaml:"source_message_id"`
	Confidence      float64 `json:"confidence" yaml:"confidence"`
}

func ValidateTriageEnrichment(value TriageEnrichment) error {
	var problems []string
	if value.Version != TriageContractVersion {
		problems = append(problems, fmt.Sprintf("version must be %q", TriageContractVersion))
	}
	if value.Scope != "message" && value.Scope != "thread" {
		problems = append(problems, "scope must be message or thread")
	}
	if strings.TrimSpace(value.SubjectID) == "" {
		problems = append(problems, "subject_id is required")
	}
	if value.Source.Kind != "external" && value.Source.Kind != "heuristic" {
		problems = append(problems, "source.kind must be external or heuristic")
	}
	if strings.TrimSpace(value.Source.Provider) == "" {
		problems = append(problems, "source.provider is required")
	}
	if err := validateRFC3339("generated_at", value.GeneratedAt, true); err != nil {
		problems = append(problems, err.Error())
	}
	if strings.TrimSpace(value.Summary) == "" && value.Priority == nil && value.NeedsReply == nil && len(value.Deadlines) == 0 && len(value.Todos) == 0 {
		problems = append(problems, "at least one enrichment field is required")
	}

	if value.Priority != nil {
		switch value.Priority.Level {
		case "low", "normal", "high", "urgent":
		default:
			problems = append(problems, "priority.level must be low, normal, high, or urgent")
		}
		validateAssessment("priority", value.Priority.Confidence, value.Priority.Reasons, &problems)
	}
	if value.NeedsReply != nil {
		validateAssessment("needs_reply", value.NeedsReply.Confidence, value.NeedsReply.Reasons, &problems)
	}
	for i, deadline := range value.Deadlines {
		prefix := fmt.Sprintf("deadlines[%d]", i)
		if strings.TrimSpace(deadline.Text) == "" {
			problems = append(problems, prefix+".text is required")
		}
		if strings.TrimSpace(deadline.SourceMessageID) == "" {
			problems = append(problems, prefix+".source_message_id is required")
		}
		if err := validateConfidence(prefix+".confidence", deadline.Confidence); err != nil {
			problems = append(problems, err.Error())
		}
		if deadline.At != "" {
			if err := validateRFC3339(prefix+".at", deadline.At, false); err != nil {
				problems = append(problems, err.Error())
			}
		}
	}
	for i, todo := range value.Todos {
		prefix := fmt.Sprintf("todos[%d]", i)
		if strings.TrimSpace(todo.Text) == "" {
			problems = append(problems, prefix+".text is required")
		}
		if strings.TrimSpace(todo.SourceMessageID) == "" {
			problems = append(problems, prefix+".source_message_id is required")
		}
		if err := validateConfidence(prefix+".confidence", todo.Confidence); err != nil {
			problems = append(problems, err.Error())
		}
		if todo.DueAt != "" {
			if err := validateRFC3339(prefix+".due_at", todo.DueAt, false); err != nil {
				problems = append(problems, err.Error())
			}
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid triage enrichment: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateAssessment(name string, confidence float64, reasons []string, problems *[]string) {
	if err := validateConfidence(name+".confidence", confidence); err != nil {
		*problems = append(*problems, err.Error())
	}
	if !hasNonEmptyString(reasons) {
		*problems = append(*problems, name+".reasons must include at least one reason")
	}
}

func validateConfidence(name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
		return fmt.Errorf("%s must be between 0 and 1", name)
	}
	return nil
}

func validateRFC3339(name, value string, required bool) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if required {
			return fmt.Errorf("%s is required", name)
		}
		return nil
	}
	if _, err := time.Parse(time.RFC3339, trimmed); err != nil {
		return fmt.Errorf("%s must be RFC3339", name)
	}
	return nil
}

func hasNonEmptyString(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
