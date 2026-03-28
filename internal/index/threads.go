package index

import (
	"sort"
	"strings"

	"github.com/nonozone/MailCli/pkg/schema"
)

type ThreadQuery struct {
	Query    string
	Account  string
	Mailbox  string
	Category string
	Action   string
	HasCodes bool
	Limit    int
}

type ThreadSummary struct {
	Account            string   `json:"account,omitempty"`
	Mailbox            string   `json:"mailbox,omitempty"`
	ThreadID           string   `json:"thread_id,omitempty"`
	Subject            string   `json:"subject,omitempty"`
	LatestDate         string   `json:"latest_date,omitempty"`
	LastMessageID      string   `json:"last_message_id,omitempty"`
	LastMessageFrom    string   `json:"last_message_from,omitempty"`
	LastMessagePreview string   `json:"last_message_preview,omitempty"`
	Categories         []string `json:"categories,omitempty"`
	ActionTypes        []string `json:"action_types,omitempty"`
	Labels             []string `json:"labels,omitempty"`
	HasCodes           bool     `json:"has_codes"`
	CodeCount          int      `json:"code_count"`
	ActionCount        int      `json:"action_count"`
	MessageCount       int      `json:"message_count"`
	ParticipantCount   int      `json:"participant_count"`
	Participants       []string `json:"participants,omitempty"`
	MessageIDs         []string `json:"message_ids,omitempty"`
	Score              int      `json:"score"`
}

type ThreadMessageQuery struct {
	ThreadID string
	Account  string
	Mailbox  string
	Limit    int
}

type threadAccumulator struct {
	summary       ThreadSummary
	latestIndexed string
	participants  map[string]struct{}
	categories    map[string]struct{}
	actionTypes   map[string]struct{}
	labels        map[string]struct{}
}

func (s *FileStore) Threads(query ThreadQuery) ([]ThreadSummary, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(strings.TrimSpace(query.Query))
	account := strings.TrimSpace(query.Account)
	mailbox := strings.TrimSpace(query.Mailbox)
	category := strings.TrimSpace(query.Category)
	action := strings.TrimSpace(query.Action)
	threads := map[string]*threadAccumulator{}

	for _, item := range data.Messages {
		if account != "" && item.Account != account {
			continue
		}
		if mailbox != "" && !strings.EqualFold(item.Mailbox, mailbox) {
			continue
		}

		threadID := deriveThreadID(item)
		acc, ok := threads[threadID]
		if !ok {
			acc = &threadAccumulator{
				summary: ThreadSummary{
					Account:  item.Account,
					Mailbox:  item.Mailbox,
					ThreadID: threadID,
				},
				participants: map[string]struct{}{},
				categories:   map[string]struct{}{},
				actionTypes:  map[string]struct{}{},
				labels:       map[string]struct{}{},
			}
			threads[threadID] = acc
		}

		acc.summary.MessageCount++
		acc.summary.MessageIDs = append(acc.summary.MessageIDs, item.ID)

		if item.Message.Meta.Date > acc.summary.LatestDate {
			acc.summary.LatestDate = item.Message.Meta.Date
			acc.summary.LastMessageID = item.ID
			acc.summary.LastMessageFrom = formatAddress(item.Message.Meta.From)
			acc.summary.LastMessagePreview = firstNonEmptyThreadValue(item.Message.Content.Snippet, item.Message.Content.BodyMD)
		} else if acc.summary.LastMessageID == "" {
			acc.summary.LastMessageID = item.ID
			acc.summary.LastMessageFrom = formatAddress(item.Message.Meta.From)
			acc.summary.LastMessagePreview = firstNonEmptyThreadValue(item.Message.Content.Snippet, item.Message.Content.BodyMD)
		}
		if item.IndexedAt > acc.latestIndexed {
			acc.latestIndexed = item.IndexedAt
		}

		if acc.summary.Subject == "" || normalizeMessageRef(item.Message.Meta.MessageID) == threadID {
			acc.summary.Subject = firstNonEmptyThreadValue(item.Message.Meta.Subject, acc.summary.Subject)
		}

		addThreadParticipant(acc, item.Message.Meta.From)
		for _, addr := range item.Message.Meta.To {
			addrCopy := addr
			addThreadParticipant(acc, &addrCopy)
		}
		addThreadCategory(acc, item.Message.Content.Category)
		for _, action := range item.Message.Actions {
			addThreadActionType(acc, action.Type)
		}
		for _, label := range item.Message.Labels {
			addThreadLabel(acc, label)
		}
		if len(item.Message.Codes) > 0 {
			acc.summary.HasCodes = true
			acc.summary.CodeCount += len(item.Message.Codes)
		}

		if needle != "" {
			acc.summary.Score += scoreMatch(item, needle)
		}
	}

	results := make([]ThreadSummary, 0, len(threads))
	for _, acc := range threads {
		if needle != "" && acc.summary.Score == 0 {
			continue
		}
		if query.HasCodes && !acc.summary.HasCodes {
			continue
		}
		if category != "" && !containsThreadValue(acc.summary.Categories, category) {
			continue
		}
		if action != "" && !containsThreadValue(acc.summary.ActionTypes, action) {
			continue
		}

		sort.Strings(acc.summary.Categories)
		sort.Strings(acc.summary.ActionTypes)
		sort.Strings(acc.summary.Labels)
		sort.Strings(acc.summary.Participants)
		sort.Strings(acc.summary.MessageIDs)
		acc.summary.ActionCount = len(acc.summary.ActionTypes)
		acc.summary.ParticipantCount = len(acc.summary.Participants)
		results = append(results, acc.summary)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if needle != "" && results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].LatestDate != results[j].LatestDate {
			return results[i].LatestDate > results[j].LatestDate
		}
		if results[i].ActionCount != results[j].ActionCount {
			return results[i].ActionCount > results[j].ActionCount
		}
		if results[i].CodeCount != results[j].CodeCount {
			return results[i].CodeCount > results[j].CodeCount
		}
		return results[i].ThreadID < results[j].ThreadID
	})

	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

func containsThreadValue(values []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *FileStore) ThreadMessages(query ThreadMessageQuery) ([]IndexedMessage, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}

	threadID := normalizeMessageRef(query.ThreadID)
	account := strings.TrimSpace(query.Account)
	mailbox := strings.TrimSpace(query.Mailbox)
	results := make([]IndexedMessage, 0, len(data.Messages))

	for _, item := range data.Messages {
		if threadID != "" && deriveThreadID(item) != threadID {
			continue
		}
		if account != "" && item.Account != account {
			continue
		}
		if mailbox != "" && !strings.EqualFold(item.Mailbox, mailbox) {
			continue
		}
		results = append(results, ensureThreadID(item))
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Message.Meta.Date != results[j].Message.Meta.Date {
			return results[i].Message.Meta.Date < results[j].Message.Meta.Date
		}
		return results[i].IndexedAt < results[j].IndexedAt
	})

	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

func deriveThreadID(item IndexedMessage) string {
	for _, ref := range item.Message.Meta.References {
		if normalized := normalizeMessageRef(ref); normalized != "" {
			return normalized
		}
	}
	if normalized := normalizeMessageRef(item.Message.Meta.InReplyTo); normalized != "" {
		return normalized
	}
	if normalized := normalizeMessageRef(item.Message.Meta.MessageID); normalized != "" {
		return normalized
	}
	if normalized := strings.TrimSpace(item.ID); normalized != "" {
		return normalized
	}
	return "unknown-thread"
}

func normalizeMessageRef(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		return trimmed
	}
	return "<" + strings.Trim(trimmed, "<>") + ">"
}

func addThreadParticipant(acc *threadAccumulator, addr *schema.Address) {
	if acc == nil || addr == nil {
		return
	}
	participant := formatAddress(addr)
	if participant == "" {
		return
	}
	if _, ok := acc.participants[participant]; ok {
		return
	}
	acc.participants[participant] = struct{}{}
	acc.summary.Participants = append(acc.summary.Participants, participant)
}

func addThreadCategory(acc *threadAccumulator, category string) {
	value := strings.TrimSpace(category)
	if acc == nil || value == "" {
		return
	}
	if _, ok := acc.categories[value]; ok {
		return
	}
	acc.categories[value] = struct{}{}
	acc.summary.Categories = append(acc.summary.Categories, value)
}

func addThreadActionType(acc *threadAccumulator, actionType string) {
	value := strings.TrimSpace(actionType)
	if acc == nil || value == "" {
		return
	}
	if _, ok := acc.actionTypes[value]; ok {
		return
	}
	acc.actionTypes[value] = struct{}{}
	acc.summary.ActionTypes = append(acc.summary.ActionTypes, value)
}

func addThreadLabel(acc *threadAccumulator, label string) {
	value := strings.TrimSpace(label)
	if acc == nil || value == "" {
		return
	}
	if _, ok := acc.labels[value]; ok {
		return
	}
	acc.labels[value] = struct{}{}
	acc.summary.Labels = append(acc.summary.Labels, value)
}

func firstNonEmptyThreadValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
