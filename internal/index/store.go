// Package index provides the local message index backed by SQLite + FTS5.
// The public API surface (FileStore, IndexedMessage, SearchQuery, SearchResult)
// is unchanged from v1 — all callers continue to compile without modification.
package index

import (
	"strings"

	"github.com/nonozone/MailCli/pkg/schema"
)

// ── Scoring helpers (used by threads.go) ─────────────────────────────────────

var searchNormalizer = strings.NewReplacer(
	"_", " ", "-", " ", "/", " ", ":", " ", ".", " ",
	",", " ", ";", " ", "(", " ", ")", " ", "<", " ", ">", " ",
	"\n", " ", "\r", " ", "\t", " ",
)

func scoreMatch(item IndexedMessage, needle string) int {
	if needle == "" {
		return 0
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
	score += weightedContains(actionSearchText(item.Message.Actions), needle, 7)
	score += weightedContains(codeSearchText(item.Message.Codes), needle, 7)
	return score
}

func weightedContains(value, needle string, weight int) int {
	if weight <= 0 || value == "" || needle == "" {
		return 0
	}
	lower := strings.ToLower(value)
	count := strings.Count(lower, needle)
	normalizedValue := normalizeSearchValue(value)
	normalizedNeedle := normalizeSearchValue(needle)
	if normalizedValue != "" && normalizedNeedle != "" {
		if nc := strings.Count(normalizedValue, normalizedNeedle); nc > count {
			count = nc
		}
	}
	if count == 0 {
		return 0
	}
	return count * weight
}

func normalizeSearchValue(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return ""
	}
	return strings.Join(strings.Fields(searchNormalizer.Replace(lower)), " ")
}

func actionSearchText(actions []schema.Action) string {
	if len(actions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(actions)*2)
	for _, a := range actions {
		parts = append(parts, a.Type, a.Label)
	}
	return strings.Join(parts, "\n")
}

func codeSearchText(codes []schema.Code) string {
	if len(codes) == 0 {
		return ""
	}
	parts := make([]string, 0, len(codes)*3)
	for _, c := range codes {
		parts = append(parts, c.Type, c.Label, c.Value)
	}
	return strings.Join(parts, "\n")
}

// ── Public types ─────────────────────────────────────────────────────────────

// IndexedMessage is one entry in the local index.
type IndexedMessage struct {
	Account   string                 `json:"account"`
	Mailbox   string                 `json:"mailbox,omitempty"`
	ID        string                 `json:"id"`
	IndexedAt string                 `json:"indexed_at,omitempty"`
	ThreadID  string                 `json:"thread_id,omitempty"`
	Message   schema.StandardMessage `json:"message"`
}

// SearchQuery describes a filter + optional full-text search.
type SearchQuery struct {
	Query    string
	Account  string
	Mailbox  string
	ThreadID string
	Limit    int
	// Since and Before are optional RFC3339 timestamps.
	Since  string
	Before string
}

// SearchResult is the compact return type for Search().
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

// ── Shared helper functions ───────────────────────────────────────────────────
// These are used by both sqlite_store.go and threads.go.

func effectiveMessageDate(item IndexedMessage) string {
	if strings.TrimSpace(item.Message.Meta.Date) != "" {
		return item.Message.Meta.Date
	}
	return item.IndexedAt
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
