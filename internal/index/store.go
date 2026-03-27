package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nonozone/MailCli/pkg/schema"
)

type IndexedMessage struct {
	Account   string                 `json:"account"`
	Mailbox   string                 `json:"mailbox,omitempty"`
	ID        string                 `json:"id"`
	IndexedAt string                 `json:"indexed_at,omitempty"`
	ThreadID  string                 `json:"thread_id,omitempty"`
	Message   schema.StandardMessage `json:"message"`
}

type SearchQuery struct {
	Query    string
	Account  string
	Mailbox  string
	ThreadID string
	Limit    int
}

type SearchResult struct {
	Account   string   `json:"account,omitempty"`
	Mailbox   string   `json:"mailbox,omitempty"`
	ID        string   `json:"id,omitempty"`
	ThreadID  string   `json:"thread_id,omitempty"`
	Subject   string   `json:"subject,omitempty"`
	From      string   `json:"from,omitempty"`
	Date      string   `json:"date,omitempty"`
	Snippet   string   `json:"snippet,omitempty"`
	Category  string   `json:"category,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Score     int      `json:"score"`
	IndexedAt string   `json:"indexed_at,omitempty"`
}

type FileStore struct {
	path string
}

type fileData struct {
	Version  int              `json:"version"`
	Messages []IndexedMessage `json:"messages"`
}

type searchMatch struct {
	item  IndexedMessage
	score int
}

func DefaultPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ".mailcli-index.json"
	}
	return filepath.Join(dir, "mailcli", "index.json")
}

func NewFileStore(path string) *FileStore {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath()
	}
	return &FileStore{path: path}
}

func (s *FileStore) Path() string {
	return s.path
}

func (s *FileStore) All() ([]IndexedMessage, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	return append([]IndexedMessage(nil), data.Messages...), nil
}

func (s *FileStore) Has(account, id string) (bool, error) {
	data, err := s.load()
	if err != nil {
		return false, err
	}

	account = strings.TrimSpace(account)
	id = strings.TrimSpace(id)
	for _, item := range data.Messages {
		if item.Account == account && item.ID == id {
			return true, nil
		}
	}

	return false, nil
}

func (s *FileStore) Upsert(message IndexedMessage) error {
	data, err := s.load()
	if err != nil {
		return err
	}

	message.Account = strings.TrimSpace(message.Account)
	message.Mailbox = strings.TrimSpace(message.Mailbox)
	message.ID = strings.TrimSpace(message.ID)
	if message.IndexedAt == "" {
		message.IndexedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if message.ThreadID == "" {
		message.ThreadID = deriveThreadID(message)
	}

	updated := false
	for i := range data.Messages {
		if data.Messages[i].Account == message.Account && data.Messages[i].ID == message.ID {
			data.Messages[i] = message
			updated = true
			break
		}
	}
	if !updated {
		data.Messages = append(data.Messages, message)
	}

	return s.save(data)
}

func (s *FileStore) Search(query SearchQuery) ([]SearchResult, error) {
	matches, err := s.findMatches(query)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(matches))
	for _, match := range matches {
		item := match.item
		results = append(results, SearchResult{
			Account:   item.Account,
			Mailbox:   item.Mailbox,
			ID:        item.ID,
			ThreadID:  ensureThreadID(item).ThreadID,
			Subject:   item.Message.Meta.Subject,
			From:      formatAddress(item.Message.Meta.From),
			Date:      item.Message.Meta.Date,
			Snippet:   item.Message.Content.Snippet,
			Category:  item.Message.Content.Category,
			Labels:    append([]string(nil), item.Message.Labels...),
			Score:     match.score,
			IndexedAt: item.IndexedAt,
		})
	}

	return results, nil
}

func (s *FileStore) SearchMessages(query SearchQuery) ([]IndexedMessage, error) {
	matches, err := s.findMatches(query)
	if err != nil {
		return nil, err
	}

	results := make([]IndexedMessage, 0, len(matches))
	for _, match := range matches {
		results = append(results, ensureThreadID(match.item))
	}

	return results, nil
}

func ensureThreadID(item IndexedMessage) IndexedMessage {
	if item.ThreadID == "" {
		item.ThreadID = deriveThreadID(item)
	}
	return item
}

func (s *FileStore) findMatches(query SearchQuery) ([]searchMatch, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(strings.TrimSpace(query.Query))
	account := strings.TrimSpace(query.Account)
	mailbox := strings.TrimSpace(query.Mailbox)
	threadID := normalizeMessageRef(query.ThreadID)
	results := make([]searchMatch, 0, len(data.Messages))

	for _, item := range data.Messages {
		if account != "" && item.Account != account {
			continue
		}
		if mailbox != "" && !strings.EqualFold(item.Mailbox, mailbox) {
			continue
		}
		if threadID != "" && deriveThreadID(item) != threadID {
			continue
		}
		score := scoreMatch(item, needle)
		if needle != "" && score == 0 {
			continue
		}
		results = append(results, searchMatch{item: item, score: score})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].item.IndexedAt > results[j].item.IndexedAt
	})

	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

func (s *FileStore) load() (fileData, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fileData{Version: 1, Messages: []IndexedMessage{}}, nil
		}
		return fileData{}, err
	}

	if len(strings.TrimSpace(string(raw))) == 0 {
		return fileData{Version: 1, Messages: []IndexedMessage{}}, nil
	}

	var data fileData
	if decodeErr := json.Unmarshal(raw, &data); decodeErr == nil {
		if data.Version == 0 {
			data.Version = 1
		}
		if data.Messages == nil {
			data.Messages = []IndexedMessage{}
		}
		return data, nil
	}

	var empty map[string]any
	if decodeErr := json.Unmarshal(raw, &empty); decodeErr == nil && len(empty) == 0 {
		return fileData{Version: 1, Messages: []IndexedMessage{}}, nil
	}

	return fileData{}, json.Unmarshal(raw, &data)
}

func (s *FileStore) save(data fileData) error {
	if data.Version == 0 {
		data.Version = 1
	}
	sort.SliceStable(data.Messages, func(i, j int) bool {
		return data.Messages[i].IndexedAt < data.Messages[j].IndexedAt
	})

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	return os.WriteFile(s.path, raw, 0o644)
}

func searchableText(item IndexedMessage) string {
	parts := []string{
		item.Account,
		item.Mailbox,
		item.ID,
		item.Message.Meta.Subject,
		item.Message.Content.Snippet,
		item.Message.Content.BodyMD,
		item.Message.Content.Category,
		strings.Join(item.Message.Labels, "\n"),
		formatAddress(item.Message.Meta.From),
	}
	return strings.ToLower(strings.Join(parts, "\n"))
}

func scoreMatch(item IndexedMessage, needle string) int {
	if needle == "" {
		return 1
	}

	score := 0
	score += weightedContains(item.Message.Meta.Subject, needle, 10)
	score += weightedContains(item.Message.Content.Snippet, needle, 6)
	score += weightedContains(item.Message.Content.BodyMD, needle, 3)
	score += weightedContains(formatAddress(item.Message.Meta.From), needle, 4)
	score += weightedContains(item.ID, needle, 2)
	score += weightedContains(item.Account, needle, 1)
	score += weightedContains(item.Mailbox, needle, 1)
	score += weightedContains(item.Message.Content.Category, needle, 2)
	score += weightedContains(strings.Join(item.Message.Labels, "\n"), needle, 2)
	return score
}

func weightedContains(value, needle string, weight int) int {
	if weight <= 0 {
		return 0
	}

	lower := strings.ToLower(value)
	if lower == "" || needle == "" {
		return 0
	}

	count := strings.Count(lower, needle)
	if count == 0 {
		return 0
	}

	return count * weight
}

func formatAddress(addr *schema.Address) string {
	if addr == nil {
		return ""
	}
	if addr.Name == "" {
		return addr.Address
	}
	if addr.Address == "" {
		return addr.Name
	}
	return addr.Name + " <" + addr.Address + ">"
}
