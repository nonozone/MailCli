package operations

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nonozone/MailCli/pkg/schema"
)

var ErrNotFound = errors.New("operation not found")

type SendIntent struct {
	ID        string                  `json:"id"`
	Operation string                  `json:"operation"`
	Account   string                  `json:"account,omitempty"`
	MessageID string                  `json:"message_id,omitempty"`
	CreatedAt string                  `json:"created_at"`
	Summary   schema.OperationSummary `json:"summary"`
	Draft     schema.DraftMessage     `json:"draft"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath()
	}
	return &Store{path: path}
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".mailcli-operations.jsonl"
	}
	return filepath.Join(home, ".local", "state", "mailcli", "operations.jsonl")
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) SaveSendIntent(intent SendIntent) error {
	if strings.TrimSpace(intent.ID) == "" {
		return fmt.Errorf("intent id is required")
	}
	if strings.TrimSpace(intent.CreatedAt) == "" {
		intent.CreatedAt = Now()
	}
	if strings.TrimSpace(intent.Operation) == "" {
		intent.Operation = "send"
	}

	if err := os.MkdirAll(s.intentDir(), 0o700); err != nil {
		return fmt.Errorf("create intent directory: %w", err)
	}
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal intent: %w", err)
	}
	if err := os.WriteFile(s.intentPath(intent.ID), data, 0o600); err != nil {
		return fmt.Errorf("write intent: %w", err)
	}
	if err := os.Chmod(s.intentPath(intent.ID), 0o600); err != nil {
		return fmt.Errorf("set intent permissions: %w", err)
	}

	summary := intent.Summary
	return s.Append(schema.OperationLogEntry{
		ID:        NewID("op"),
		IntentID:  intent.ID,
		Operation: intent.Operation,
		Status:    "prepared",
		Account:   intent.Account,
		MessageID: intent.MessageID,
		CreatedAt: intent.CreatedAt,
		Summary:   &summary,
	})
}

func (s *Store) LoadSendIntent(id string) (SendIntent, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return SendIntent{}, ErrNotFound
	}
	data, err := os.ReadFile(s.intentPath(trimmed))
	if err != nil {
		if os.IsNotExist(err) {
			return SendIntent{}, fmt.Errorf("%w: %s", ErrNotFound, trimmed)
		}
		return SendIntent{}, fmt.Errorf("read intent: %w", err)
	}
	var intent SendIntent
	if err := json.Unmarshal(data, &intent); err != nil {
		return SendIntent{}, fmt.Errorf("decode intent: %w", err)
	}
	if intent.ID == "" {
		intent.ID = trimmed
	}
	return intent, nil
}

func (s *Store) Append(entry schema.OperationLogEntry) error {
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = NewID("op")
	}
	if strings.TrimSpace(entry.CreatedAt) == "" {
		entry.CreatedAt = Now()
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create operations directory: %w", err)
	}
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open operations log: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("write operations log: %w", err)
	}
	if err := os.Chmod(s.path, 0o600); err != nil {
		return fmt.Errorf("set operations log permissions: %w", err)
	}
	return nil
}

func (s *Store) List() ([]schema.OperationLogEntry, error) {
	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []schema.OperationLogEntry{}, nil
		}
		return nil, fmt.Errorf("open operations log: %w", err)
	}
	defer file.Close()

	var entries []schema.OperationLogEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry schema.OperationLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("decode operations log: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan operations log: %w", err)
	}
	return entries, nil
}

func (s *Store) Find(id string) (schema.OperationLogEntry, error) {
	target := strings.TrimSpace(id)
	if target == "" {
		return schema.OperationLogEntry{}, ErrNotFound
	}
	entries, err := s.List()
	if err != nil {
		return schema.OperationLogEntry{}, err
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].ID == target || entries[i].IntentID == target {
			return entries[i], nil
		}
	}
	return schema.OperationLogEntry{}, fmt.Errorf("%w: %s", ErrNotFound, target)
}

func (s *Store) SentEntryForIntent(intentID string) (schema.OperationLogEntry, error) {
	target := strings.TrimSpace(intentID)
	if target == "" {
		return schema.OperationLogEntry{}, ErrNotFound
	}
	entries, err := s.List()
	if err != nil {
		return schema.OperationLogEntry{}, err
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].IntentID == target && entries[i].Operation == "send" && entries[i].Status == "sent" {
			return entries[i], nil
		}
	}
	return schema.OperationLogEntry{}, fmt.Errorf("%w: %s", ErrNotFound, target)
}

func NewID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return strings.TrimSpace(prefix) + "_" + hex.EncodeToString(bytes[:])
	}
	return fmt.Sprintf("%s_%d", strings.TrimSpace(prefix), time.Now().UnixNano())
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func (s *Store) intentDir() string {
	return s.path + ".intents"
}

func (s *Store) intentPath(id string) string {
	return filepath.Join(s.intentDir(), strings.TrimSpace(id)+".json")
}
