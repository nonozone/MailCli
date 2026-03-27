package index

import (
	"sort"
	"strings"

	"github.com/nonozone/MailCli/pkg/schema"
)

type ThreadQuery struct {
	Query   string
	Account string
	Mailbox string
	Limit   int
}

type ThreadSummary struct {
	Account      string   `json:"account,omitempty"`
	Mailbox      string   `json:"mailbox,omitempty"`
	ThreadID     string   `json:"thread_id,omitempty"`
	Subject      string   `json:"subject,omitempty"`
	LatestDate   string   `json:"latest_date,omitempty"`
	MessageCount int      `json:"message_count"`
	Participants []string `json:"participants,omitempty"`
	MessageIDs   []string `json:"message_ids,omitempty"`
	Score        int      `json:"score"`
}

type threadAccumulator struct {
	summary       ThreadSummary
	latestIndexed string
	participants  map[string]struct{}
}

func (s *FileStore) Threads(query ThreadQuery) ([]ThreadSummary, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(strings.TrimSpace(query.Query))
	account := strings.TrimSpace(query.Account)
	mailbox := strings.TrimSpace(query.Mailbox)
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
			}
			threads[threadID] = acc
		}

		acc.summary.MessageCount++
		acc.summary.MessageIDs = append(acc.summary.MessageIDs, item.ID)

		if item.Message.Meta.Date > acc.summary.LatestDate {
			acc.summary.LatestDate = item.Message.Meta.Date
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

		acc.summary.Score += scoreMatch(item, needle)
	}

	results := make([]ThreadSummary, 0, len(threads))
	for _, acc := range threads {
		if needle != "" && acc.summary.Score == 0 {
			continue
		}

		sort.Strings(acc.summary.Participants)
		sort.Strings(acc.summary.MessageIDs)
		results = append(results, acc.summary)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].LatestDate != results[j].LatestDate {
			return results[i].LatestDate > results[j].LatestDate
		}
		return results[i].ThreadID < results[j].ThreadID
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

func firstNonEmptyThreadValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
