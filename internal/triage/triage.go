package triage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/schema"
)

func FromMessage(message schema.StandardMessage) schema.TriageRecord {
	id := firstNonEmpty(message.ID, message.Meta.MessageID)
	return buildRecord("message", id, "", "", []messageInput{{
		ID:      id,
		Message: message,
	}})
}

func FromThread(threadID string, items []mailindex.IndexedMessage) (schema.TriageRecord, error) {
	if len(items) == 0 {
		return schema.TriageRecord{}, fmt.Errorf("thread %q did not contain any messages", threadID)
	}

	ordered := append([]mailindex.IndexedMessage(nil), items...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := firstNonEmpty(ordered[i].Message.Meta.Date, ordered[i].IndexedAt)
		right := firstNonEmpty(ordered[j].Message.Meta.Date, ordered[j].IndexedAt)
		if left != right {
			return left < right
		}
		return ordered[i].ID < ordered[j].ID
	})
	account := ordered[0].Account
	mailbox := ordered[0].Mailbox
	for _, item := range ordered[1:] {
		if item.Account != account {
			return schema.TriageRecord{}, fmt.Errorf("thread %q spans multiple accounts; pass --account to select one", threadID)
		}
		if item.Mailbox != mailbox {
			mailbox = ""
		}
	}

	messages := make([]messageInput, 0, len(ordered))
	for _, item := range ordered {
		messages = append(messages, messageInput{
			ID:        firstNonEmpty(item.ID, item.Message.ID, item.Message.Meta.MessageID),
			IndexedAt: item.IndexedAt,
			Message:   item.Message,
		})
	}

	return buildRecord(
		"thread",
		strings.TrimSpace(threadID),
		account,
		mailbox,
		messages,
	), nil
}

func ApplyEnrichment(record *schema.TriageRecord, enrichment schema.TriageEnrichment) error {
	if record == nil {
		return fmt.Errorf("triage record is required")
	}
	if err := schema.ValidateTriageEnrichment(enrichment); err != nil {
		return err
	}
	if enrichment.Scope != record.Scope {
		return fmt.Errorf("enrichment scope %q does not match triage scope %q", enrichment.Scope, record.Scope)
	}
	if enrichment.SubjectID != record.SubjectID {
		return fmt.Errorf("enrichment subject_id %q does not match triage subject_id %q", enrichment.SubjectID, record.SubjectID)
	}
	if enrichment.EvidenceID != record.EvidenceID {
		return fmt.Errorf("enrichment evidence_id %q does not match current triage evidence_id %q", enrichment.EvidenceID, record.EvidenceID)
	}

	messageIDs := make(map[string]struct{}, len(record.Evidence.MessageIDs))
	for _, id := range record.Evidence.MessageIDs {
		messageIDs[id] = struct{}{}
	}
	for _, deadline := range enrichment.Deadlines {
		if _, ok := messageIDs[deadline.SourceMessageID]; !ok {
			return fmt.Errorf("deadline source_message_id %q is not present in triage evidence", deadline.SourceMessageID)
		}
	}
	for _, todo := range enrichment.Todos {
		if _, ok := messageIDs[todo.SourceMessageID]; !ok {
			return fmt.Errorf("todo source_message_id %q is not present in triage evidence", todo.SourceMessageID)
		}
	}

	record.Enrichment = &enrichment
	return nil
}

type messageInput struct {
	ID        string
	IndexedAt string
	Message   schema.StandardMessage
}

func buildRecord(scope, subjectID, account, mailbox string, messages []messageInput) schema.TriageRecord {
	evidence := schema.TriageEvidence{
		Source:       "deterministic",
		MessageCount: len(messages),
		Messages:     make([]schema.TriageMessageFact, 0, len(messages)),
	}
	participants := map[string]string{}
	categories := map[string]struct{}{}
	labels := map[string]struct{}{}
	actionTypes := map[string]struct{}{}

	for _, item := range messages {
		message := item.Message
		id := firstNonEmpty(item.ID, message.ID, message.Meta.MessageID)
		from := formatAddress(message.Meta.From)
		to := formatAddresses(message.Meta.To)
		date := firstNonEmpty(message.Meta.Date, item.IndexedAt)
		factActionTypes := uniqueActionTypes(message.Actions)
		fact := schema.TriageMessageFact{
			ID:              id,
			From:            from,
			To:              to,
			Subject:         strings.TrimSpace(message.Meta.Subject),
			Date:            date,
			Snippet:         strings.TrimSpace(message.Content.Snippet),
			AutoSubmitted:   message.Meta.AutoSubmitted,
			ActionCount:     len(message.Actions),
			ActionTypes:     factActionTypes,
			CodeCount:       len(message.Codes),
			AttachmentCount: len(message.Attachments),
			HasError:        message.ErrorContext != nil,
		}
		evidence.Messages = append(evidence.Messages, fact)
		evidence.MessageIDs = append(evidence.MessageIDs, id)
		evidence.LatestDate = date
		evidence.LastMessageID = id
		evidence.LastMessageFrom = from
		evidence.ActionCount += fact.ActionCount
		evidence.CodeCount += fact.CodeCount
		evidence.AttachmentCount += fact.AttachmentCount
		if fact.AutoSubmitted {
			evidence.AutoSubmittedCount++
		}
		if fact.HasError {
			evidence.ErrorCount++
		}

		addParticipant(participants, message.Meta.From)
		for i := range message.Meta.To {
			addParticipant(participants, &message.Meta.To[i])
		}
		addSetValue(categories, message.Content.Category)
		for _, label := range message.Labels {
			addSetValue(labels, label)
		}
		for _, actionType := range factActionTypes {
			addSetValue(actionTypes, actionType)
		}
	}

	evidence.Participants = sortedParticipantValues(participants)
	evidence.ParticipantCount = len(evidence.Participants)
	evidence.Categories = sortedSetValues(categories)
	evidence.Labels = sortedSetValues(labels)
	evidence.ActionTypes = sortedSetValues(actionTypes)
	evidence.HasActions = evidence.ActionCount > 0
	evidence.HasCodes = evidence.CodeCount > 0
	evidence.HasAttachments = evidence.AttachmentCount > 0

	record := schema.TriageRecord{
		Version:   schema.TriageContractVersion,
		Scope:     scope,
		SubjectID: subjectID,
		Account:   strings.TrimSpace(account),
		Mailbox:   strings.TrimSpace(mailbox),
		Evidence:  evidence,
	}
	record.EvidenceID = triageEvidenceID(evidence)
	return record
}

func triageEvidenceID(evidence schema.TriageEvidence) string {
	raw, _ := json.Marshal(evidence)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func uniqueActionTypes(actions []schema.Action) []string {
	values := map[string]struct{}{}
	for _, action := range actions {
		addSetValue(values, action.Type)
	}
	return sortedSetValues(values)
}

func addSetValue(values map[string]struct{}, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		values[trimmed] = struct{}{}
	}
}

func addParticipant(values map[string]string, address *schema.Address) {
	formatted := formatAddress(address)
	if formatted == "" {
		return
	}
	key := strings.ToLower(strings.TrimSpace(address.Address))
	if key == "" {
		key = "name:" + strings.ToLower(strings.TrimSpace(address.Name))
	}
	current := values[key]
	if current == "" || (!strings.Contains(current, "<") && strings.Contains(formatted, "<")) {
		values[key] = formatted
	}
}

func sortedParticipantValues(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedSetValues(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func formatAddresses(addresses []schema.Address) []string {
	if len(addresses) == 0 {
		return nil
	}
	out := make([]string, 0, len(addresses))
	for i := range addresses {
		if value := formatAddress(&addresses[i]); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func formatAddress(address *schema.Address) string {
	if address == nil {
		return ""
	}
	name := strings.TrimSpace(address.Name)
	email := strings.TrimSpace(address.Address)
	if name == "" {
		return email
	}
	if email == "" {
		return name
	}
	return name + " <" + email + ">"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
