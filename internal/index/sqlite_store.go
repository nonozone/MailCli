package index

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/nonozone/MailCli/pkg/schema"
)

//go:embed schema.sql
var schemaSQL string

// Store is the SQLite-backed message index.
// Use NewFileStore to create one.
type Store struct {
	path string
	mu   sync.Mutex
	db   *sql.DB
	once sync.Once
	err  error
}

// FileStore is kept as a type alias so all existing callers compile unchanged.
type FileStore = Store

// NewFileStore opens (or creates) the SQLite index at path.
// If path is empty the user-cache default is used.
// Paths ending in ".json" are silently remapped to ".db" so users can keep
// passing their existing --index flag values and get the new backend.
func NewFileStore(path string) *Store {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath()
	}
	if strings.HasSuffix(path, ".json") {
		path = strings.TrimSuffix(path, ".json") + ".db"
	}
	return &Store{path: path}
}

// DefaultPath returns the platform default index path.
func DefaultPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ".mailcli-index.db"
	}
	return filepath.Join(dir, "mailcli", "index.db")
}

// Path returns the resolved database file path.
func (s *Store) Path() string { return s.path }

// ── internal DB lifecycle ─────────────────────────────────────────────────────

func (s *Store) openDB() (*sql.DB, error) {
	s.once.Do(func() {
		if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
			s.err = err
			return
		}
		dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL", s.path)
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			s.err = err
			return
		}
		// Apply schema (idempotent – all statements use IF NOT EXISTS).
		if _, err := db.Exec(schemaSQL); err != nil {
			db.Close()
			s.err = fmt.Errorf("apply schema: %w", err)
			return
		}
		s.db = db
	})
	return s.db, s.err
}

// ── Has ───────────────────────────────────────────────────────────────────────

func (s *Store) Has(account, id string) (bool, error) {
	db, err := s.openDB()
	if err != nil {
		return false, err
	}
	var n int
	err = db.QueryRow(
		`SELECT COUNT(1) FROM messages WHERE account=? AND id=? LIMIT 1`,
		strings.TrimSpace(account), strings.TrimSpace(id),
	).Scan(&n)
	return n > 0, err
}

// BulkHas returns the set of IDs that already exist in the index for the
// given account. Used by sync to filter messages that need fetching.
func (s *Store) BulkHas(account string, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}
	db, err := s.openDB()
	if err != nil {
		return nil, err
	}
	account = strings.TrimSpace(account)

	// SQLite supports up to 999 bind params; chunk to be safe.
	const chunkSize = 500
	result := make(map[string]bool, len(ids))
	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)+1)
		args = append(args, account)
		for j, id := range chunk {
			placeholders[j] = "?"
			args = append(args, id)
		}
		q := "SELECT id FROM messages WHERE account=? AND id IN (" +
			strings.Join(placeholders, ",") + ")"
		rows, err := db.Query(q, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			result[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// BulkUpsert writes all messages inside a single SQLite transaction.
// This is dramatically faster than calling Upsert() in a loop for large
// batches because SQLite only calls fsync once at the end.
func (s *Store) BulkUpsert(msgs []IndexedMessage) error {
	if len(msgs) == 0 {
		return nil
	}
	db, err := s.openDB()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	// Prepare statements once for the whole batch.
	stmtDel, err := tx.Prepare(`DELETE FROM messages_fts WHERE account=? AND msg_id=?`)
	if err != nil {
		return err
	}
	defer stmtDel.Close()

	stmtUps, err := tx.Prepare(`
		INSERT INTO messages
		  (account, mailbox, id, thread_id, date, indexed_at,
		   from_addr, subject, category, labels, action_types,
		   has_codes, code_count, snippet, body_text, body_json)
		VALUES (?,?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?)
		ON CONFLICT(account, id) DO UPDATE SET
		  mailbox      = excluded.mailbox,
		  thread_id    = excluded.thread_id,
		  date         = excluded.date,
		  indexed_at   = excluded.indexed_at,
		  from_addr    = excluded.from_addr,
		  subject      = excluded.subject,
		  category     = excluded.category,
		  labels       = excluded.labels,
		  action_types = excluded.action_types,
		  has_codes    = excluded.has_codes,
		  code_count   = excluded.code_count,
		  snippet      = excluded.snippet,
		  body_text    = excluded.body_text,
		  body_json    = excluded.body_json`)
	if err != nil {
		return err
	}
	defer stmtUps.Close()

	stmtFTS, err := tx.Prepare(`
		INSERT INTO messages_fts(account, msg_id, subject, from_addr, body_text)
		VALUES (?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmtFTS.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for i := range msgs {
		msg := &msgs[i]
		msg.Account = strings.TrimSpace(msg.Account)
		msg.Mailbox = strings.TrimSpace(msg.Mailbox)
		msg.ID = strings.TrimSpace(msg.ID)
		if msg.IndexedAt == "" {
			msg.IndexedAt = now
		}
		if msg.ThreadID == "" {
			msg.ThreadID = deriveThreadID(*msg)
		}

		bodyJSON, err := json.Marshal(msg.Message)
		if err != nil {
			return fmt.Errorf("marshal message %s: %w", msg.ID, err)
		}
		labels, _ := json.Marshal(msg.Message.Labels)
		actionTypes := extractActionTypes(msg.Message.Actions)
		atJSON, _ := json.Marshal(actionTypes)

		date := strings.TrimSpace(msg.Message.Meta.Date)
		fromAddr := formatAddress(msg.Message.Meta.From)
		subject := strings.TrimSpace(msg.Message.Meta.Subject)
		category := strings.TrimSpace(msg.Message.Content.Category)
		snippet := strings.TrimSpace(msg.Message.Content.Snippet)
		bodyText := strings.Join([]string{
			snippet,
			strings.TrimSpace(msg.Message.Content.BodyMD),
			actionSearchText(msg.Message.Actions),
			codeSearchText(msg.Message.Codes),
		}, " ")
		bodyText = strings.TrimSpace(bodyText)
		hasCodes := 0
		if len(msg.Message.Codes) > 0 {
			hasCodes = 1
		}

		if _, err := stmtDel.Exec(msg.Account, msg.ID); err != nil {
			return fmt.Errorf("fts delete %s: %w", msg.ID, err)
		}
		if _, err := stmtUps.Exec(
			msg.Account, msg.Mailbox, msg.ID, msg.ThreadID, date, msg.IndexedAt,
			fromAddr, subject, category, string(labels), string(atJSON),
			hasCodes, len(msg.Message.Codes), snippet, bodyText, string(bodyJSON),
		); err != nil {
			return fmt.Errorf("upsert %s: %w", msg.ID, err)
		}
		if _, err := stmtFTS.Exec(msg.Account, msg.ID, subject, fromAddr, bodyText); err != nil {
			return fmt.Errorf("fts insert %s: %w", msg.ID, err)
		}
	}

	return tx.Commit()
}


// ── Upsert ────────────────────────────────────────────────────────────────────

func (s *Store) Upsert(msg IndexedMessage) error {
	db, err := s.openDB()
	if err != nil {
		return err
	}

	msg.Account = strings.TrimSpace(msg.Account)
	msg.Mailbox = strings.TrimSpace(msg.Mailbox)
	msg.ID = strings.TrimSpace(msg.ID)
	if msg.IndexedAt == "" {
		msg.IndexedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if msg.ThreadID == "" {
		msg.ThreadID = deriveThreadID(msg)
	}

	bodyJSON, err := json.Marshal(msg.Message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	labels, _ := json.Marshal(msg.Message.Labels)
	actionTypes := extractActionTypes(msg.Message.Actions)
	atJSON, _ := json.Marshal(actionTypes)

	date := strings.TrimSpace(msg.Message.Meta.Date)
	fromAddr := formatAddress(msg.Message.Meta.From)
	subject := strings.TrimSpace(msg.Message.Meta.Subject)
	category := strings.TrimSpace(msg.Message.Content.Category)
	snippet := strings.TrimSpace(msg.Message.Content.Snippet)
	// body_text for FTS: all searchable text fields concatenated.
	// This mirrors the legacy scoreMatch() weighting so structured signals
	// (actions, codes) are discoverable via FTS.
	bodyText := strings.Join([]string{
		snippet,
		strings.TrimSpace(msg.Message.Content.BodyMD),
		actionSearchText(msg.Message.Actions),
		codeSearchText(msg.Message.Codes),
	}, " ")
	bodyText = strings.TrimSpace(bodyText)
	hasCodes := 0
	if len(msg.Message.Codes) > 0 {
		hasCodes = 1
	}
	codeCount := len(msg.Message.Codes)

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	// Remove stale FTS entry if the row already exists.
	_, _ = tx.Exec(
		`DELETE FROM messages_fts WHERE account=? AND msg_id=?`,
		msg.Account, msg.ID,
	)

	// Upsert main table. ON CONFLICT preserves the rowid (unlike INSERT OR REPLACE).
	_, err = tx.Exec(`
		INSERT INTO messages
		  (account, mailbox, id, thread_id, date, indexed_at,
		   from_addr, subject, category, labels, action_types,
		   has_codes, code_count, snippet, body_text, body_json)
		VALUES (?,?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?)
		ON CONFLICT(account, id) DO UPDATE SET
		  mailbox      = excluded.mailbox,
		  thread_id    = excluded.thread_id,
		  date         = excluded.date,
		  indexed_at   = excluded.indexed_at,
		  from_addr    = excluded.from_addr,
		  subject      = excluded.subject,
		  category     = excluded.category,
		  labels       = excluded.labels,
		  action_types = excluded.action_types,
		  has_codes    = excluded.has_codes,
		  code_count   = excluded.code_count,
		  snippet      = excluded.snippet,
		  body_text    = excluded.body_text,
		  body_json    = excluded.body_json`,
		msg.Account, msg.Mailbox, msg.ID, msg.ThreadID, date, msg.IndexedAt,
		fromAddr, subject, category, string(labels), string(atJSON),
		hasCodes, codeCount, snippet, bodyText, string(bodyJSON),
	)
	if err != nil {
		return fmt.Errorf("upsert message: %w", err)
	}

	// Insert fresh FTS entry.
	_, err = tx.Exec(
		`INSERT INTO messages_fts(account, msg_id, subject, from_addr, body_text)
		 VALUES (?,?,?,?,?)`,
		msg.Account, msg.ID, subject, fromAddr, bodyText,
	)
	if err != nil {
		return fmt.Errorf("upsert fts: %w", err)
	}

	return tx.Commit()
}

// ── All ───────────────────────────────────────────────────────────────────────

// All returns all indexed messages unfiltered (used internally by Threads).
func (s *Store) All() ([]IndexedMessage, error) {
	return s.queryRows(`SELECT body_json, account, mailbox, id, thread_id, indexed_at
		FROM messages ORDER BY date DESC`, nil)
}

// allMessages is the internal alias used by threads.go.
func (s *Store) allMessages() ([]IndexedMessage, error) { return s.All() }

// ── Search ────────────────────────────────────────────────────────────────────

func (s *Store) Search(query SearchQuery) ([]SearchResult, error) {
	msgs, _, err := s.findMatchesSQL(query)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(query.Query))

	type scored struct {
		result SearchResult
		score  int
		date   string
	}
	items := make([]scored, 0, len(msgs))
	for _, item := range msgs {
		score := scoreMatch(item, needle)
		items = append(items, scored{
			result: SearchResult{
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
				Score:     score,
				IndexedAt: item.IndexedAt,
			},
			score: score,
			date:  effectiveMessageDate(item),
		})
	}

	// Re-sort by Go-layer score DESC, then date DESC.
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].date > items[j].date
	})

	results := make([]SearchResult, 0, len(items))
	for _, it := range items {
		results = append(results, it.result)
	}
	return results, nil
}

func (s *Store) SearchMessages(query SearchQuery) ([]IndexedMessage, error) {
	msgs, _, err := s.findMatchesSQL(query)
	return msgs, err
}

// findMatchesSQL retrieves matching messages and their Go-layer scores.
// FTS5 handles recall; scoreMatch() handles field-weighted re-ranking.
func (s *Store) findMatchesSQL(query SearchQuery) ([]IndexedMessage, []int, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, nil, err
	}

	needle := strings.TrimSpace(query.Query)
	account := strings.TrimSpace(query.Account)
	mailbox := strings.TrimSpace(query.Mailbox)
	threadID := normalizeMessageRef(query.ThreadID)
	since := strings.TrimSpace(query.Since)
	before := strings.TrimSpace(query.Before)

	// ── FTS path ──────────────────────────────────────────────────────────
	if needle != "" {
		return s.ftsSearch(db, needle, account, mailbox, threadID, since, before, query.Limit)
	}

	// ── Filter-only path (no text search) ─────────────────────────────────
	return s.filterSearch(db, account, mailbox, threadID, since, before, query.Limit)
}

func (s *Store) ftsSearch(db *sql.DB, needle, account, mailbox, threadID, since, before string, limit int) ([]IndexedMessage, []int, error) {
	ftsQ := toFTSQuery(needle)
	if ftsQ == "" {
		return nil, nil, nil
	}

	// We need to optionally filter the FTS results by account so we use a
	// subquery that first narrows by account in the FTS table (unindexed but
	// quick for small corpora), then joins the main table for the rest.
	args := []any{ftsQ}
	extra := ""
	if account != "" {
		extra += " AND m.account = ?"
		args = append(args, account)
	}
	if mailbox != "" {
		extra += " AND LOWER(m.mailbox) = LOWER(?)"
		args = append(args, mailbox)
	}
	if threadID != "" {
		extra += " AND m.thread_id = ?"
		args = append(args, threadID)
	}
	if since != "" {
		extra += " AND m.date >= ?"
		args = append(args, since)
	}
	if before != "" {
		extra += " AND m.date < ?"
		args = append(args, before)
	}
	limitSQL := ""
	if limit > 0 {
		limitSQL = fmt.Sprintf(" LIMIT %d", limit)
	}

	// bm25(table, w0=account_unindexed, w1=id_unindexed, w2=subject, w3=from_addr, w4=body_text)
	// Higher weights on subject and from so subject-heavy messages rank above body-only matches.
	bm25Weights := "0.0, 0.0, 10.0, 4.0, 1.0"
	q := fmt.Sprintf(`
		SELECT m.body_json, m.account, m.mailbox, m.id, m.thread_id, m.indexed_at,
		       CAST(-bm25(messages_fts, %s) * 100 AS INTEGER) AS score
		FROM messages_fts
		JOIN messages m ON m.account = messages_fts.account AND m.id = messages_fts.msg_id
		WHERE messages_fts MATCH ?%s
		ORDER BY bm25(messages_fts, %s), m.date DESC%s`, bm25Weights, extra, bm25Weights, limitSQL)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("fts query: %w", err)
	}
	defer rows.Close()
	return scanRowsWithScore(rows)
}

func (s *Store) filterSearch(db *sql.DB, account, mailbox, threadID, since, before string, limit int) ([]IndexedMessage, []int, error) {
	var conds []string
	var args []any
	if account != "" {
		conds = append(conds, "account = ?")
		args = append(args, account)
	}
	if mailbox != "" {
		conds = append(conds, "LOWER(mailbox) = LOWER(?)")
		args = append(args, mailbox)
	}
	if threadID != "" {
		conds = append(conds, "thread_id = ?")
		args = append(args, threadID)
	}
	if since != "" {
		conds = append(conds, "date >= ?")
		args = append(args, since)
	}
	if before != "" {
		conds = append(conds, "date < ?")
		args = append(args, before)
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	limitSQL := ""
	if limit > 0 {
		limitSQL = fmt.Sprintf(" LIMIT %d", limit)
	}

	q := fmt.Sprintf(`SELECT body_json, account, mailbox, id, thread_id, indexed_at
		FROM messages %s ORDER BY date DESC%s`, where, limitSQL)
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("filter query: %w", err)
	}
	defer rows.Close()

	msgs, _, err := scanRows(rows)
	if err != nil {
		return nil, nil, err
	}
	scores := make([]int, len(msgs))
	return msgs, scores, nil
}

// ── Row scanning ─────────────────────────────────────────────────────────────

func (s *Store) queryRows(q string, args []any) ([]IndexedMessage, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	msgs, _, err := scanRows(rows)
	return msgs, err
}

// scanRows reads rows that have columns: body_json, account, mailbox, id, thread_id, indexed_at
func scanRows(rows *sql.Rows) ([]IndexedMessage, []int, error) {
	var msgs []IndexedMessage
	for rows.Next() {
		var bodyJSON, account, mailbox, id, threadID, indexedAt string
		if err := rows.Scan(&bodyJSON, &account, &mailbox, &id, &threadID, &indexedAt); err != nil {
			return nil, nil, err
		}
		var msg schema.StandardMessage
		if err := json.Unmarshal([]byte(bodyJSON), &msg); err != nil {
			continue // skip corrupt entries
		}
		msgs = append(msgs, IndexedMessage{
			Account:   account,
			Mailbox:   mailbox,
			ID:        id,
			ThreadID:  threadID,
			IndexedAt: indexedAt,
			Message:   msg,
		})
	}
	return msgs, make([]int, len(msgs)), rows.Err()
}

// scanRowsWithScore reads rows that have an extra integer score column at the end.
func scanRowsWithScore(rows *sql.Rows) ([]IndexedMessage, []int, error) {
	var msgs []IndexedMessage
	var scores []int
	for rows.Next() {
		var bodyJSON, account, mailbox, id, threadID, indexedAt string
		var score int
		if err := rows.Scan(&bodyJSON, &account, &mailbox, &id, &threadID, &indexedAt, &score); err != nil {
			return nil, nil, err
		}
		var msg schema.StandardMessage
		if err := json.Unmarshal([]byte(bodyJSON), &msg); err != nil {
			continue
		}
		msgs = append(msgs, IndexedMessage{
			Account:   account,
			Mailbox:   mailbox,
			ID:        id,
			ThreadID:  threadID,
			IndexedAt: indexedAt,
			Message:   msg,
		})
		// FTS5 rank is negative (lower = more relevant); negate for positive score.
		if score < 0 {
			score = -score
		}
		scores = append(scores, score)
	}
	return msgs, scores, rows.Err()
}

// ── FTS query builder ─────────────────────────────────────────────────────────

// toFTSQuery converts a plain-text user query to an FTS5 MATCH expression.
// Each whitespace-separated word becomes a prefix term; all must match (AND).
func toFTSQuery(q string) string {
	words := strings.Fields(strings.TrimSpace(q))
	if len(words) == 0 {
		return ""
	}
	escaped := make([]string, 0, len(words))
	for _, w := range words {
		// Escape FTS5 special characters by quoting the term.
		safe := strings.ReplaceAll(w, "\"", "\"\"")
		escaped = append(escaped, `"`+safe+`"*`)
	}
	return strings.Join(escaped, " ")
}

// ── Helper functions (shared with store.go v1 helpers) ───────────────────────

func extractActionTypes(actions []schema.Action) []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range actions {
		if t := strings.TrimSpace(a.Type); t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}

func ensureThreadID(item IndexedMessage) IndexedMessage {
	if item.ThreadID == "" {
		item.ThreadID = deriveThreadID(item)
	}
	return item
}
