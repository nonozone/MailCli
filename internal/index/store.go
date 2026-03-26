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
	Message   schema.StandardMessage `json:"message"`
}

type SearchQuery struct {
	Query   string
	Account string
	Mailbox string
	Limit   int
}

type SearchResult struct {
	Account   string   `json:"account,omitempty"`
	Mailbox   string   `json:"mailbox,omitempty"`
	ID        string   `json:"id,omitempty"`
	Subject   string   `json:"subject,omitempty"`
	From      string   `json:"from,omitempty"`
	Date      string   `json:"date,omitempty"`
	Snippet   string   `json:"snippet,omitempty"`
	Category  string   `json:"category,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	IndexedAt string   `json:"indexed_at,omitempty"`
}

type FileStore struct {
	path string
}

type fileData struct {
	Version  int              `json:"version"`
	Messages []IndexedMessage `json:"messages"`
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
	items, err := s.SearchMessages(query)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(items))
	for _, item := range items {
		results = append(results, SearchResult{
			Account:   item.Account,
			Mailbox:   item.Mailbox,
			ID:        item.ID,
			Subject:   item.Message.Meta.Subject,
			From:      formatAddress(item.Message.Meta.From),
			Date:      item.Message.Meta.Date,
			Snippet:   item.Message.Content.Snippet,
			Category:  item.Message.Content.Category,
			Labels:    append([]string(nil), item.Message.Labels...),
			IndexedAt: item.IndexedAt,
		})
	}

	return results, nil
}

func (s *FileStore) SearchMessages(query SearchQuery) ([]IndexedMessage, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(strings.TrimSpace(query.Query))
	account := strings.TrimSpace(query.Account)
	mailbox := strings.TrimSpace(query.Mailbox)
	results := make([]IndexedMessage, 0, len(data.Messages))

	for i := len(data.Messages) - 1; i >= 0; i-- {
		item := data.Messages[i]
		if account != "" && item.Account != account {
			continue
		}
		if mailbox != "" && !strings.EqualFold(item.Mailbox, mailbox) {
			continue
		}
		if needle != "" && !strings.Contains(searchableText(item), needle) {
			continue
		}
		results = append(results, item)
		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
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
