package driver

import (
	"context"
	"fmt"
	"io/fs"
	netmail "net/mail"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/schema"
)

type dirDriver struct {
	root    string
	mailbox string
}

type dirMessage struct {
	id        string
	messageID string
	path      string
	date      time.Time
	summary   schema.MessageMetaSummary
}

func newDirDriver(account config.AccountConfig) (Driver, error) {
	root := strings.TrimSpace(account.Path)
	if root == "" {
		return nil, fmt.Errorf("%w: dir account requires path", ErrDriverConfigInvalid)
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve dir path: %v", ErrDriverConfigInvalid, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: dir path: %v", ErrDriverConfigInvalid, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: dir path must be a directory", ErrDriverConfigInvalid)
	}

	mailbox := strings.TrimSpace(account.Mailbox)
	if mailbox == "" {
		mailbox = "INBOX"
	}

	return &dirDriver{
		root:    absRoot,
		mailbox: mailbox,
	}, nil
}

func (d *dirDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	if mailbox := strings.TrimSpace(query.Mailbox); mailbox != "" && !strings.EqualFold(mailbox, d.mailbox) {
		return []schema.MessageMetaSummary{}, nil
	}

	messages, err := d.loadMessages(ctx)
	if err != nil {
		return nil, err
	}

	filter := strings.ToLower(strings.TrimSpace(query.Query))
	results := make([]schema.MessageMetaSummary, 0, len(messages))
	for _, message := range messages {
		if filter != "" {
			haystack := strings.ToLower(strings.Join([]string{
				message.id,
				message.messageID,
				message.summary.From,
				message.summary.Subject,
			}, "\n"))
			if !strings.Contains(haystack, filter) {
				continue
			}
		}
		results = append(results, message.summary)
	}

	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

func (d *dirDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil, fmt.Errorf("message id is required")
	}

	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		messages, err := d.loadMessages(ctx)
		if err != nil {
			return nil, err
		}
		for _, message := range messages {
			if message.messageID == trimmed {
				return os.ReadFile(message.path)
			}
		}
		return nil, fmt.Errorf("%w: %s", ErrMessageNotFound, id)
	}

	path, err := d.resolveMessagePath(trimmed)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrMessageNotFound, id)
		}
		return nil, err
	}
	return raw, nil
}

func (d *dirDriver) SendRaw(_ context.Context, _ []byte) error {
	return fmt.Errorf("%w: dir driver does not support sending", ErrTransportNotConfigured)
}

func (d *dirDriver) loadMessages(ctx context.Context) ([]dirMessage, error) {
	var messages []dirMessage

	err := filepath.WalkDir(d.root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".eml") {
			return nil
		}

		message, err := d.readMessage(path)
		if err != nil {
			return err
		}
		messages = append(messages, message)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(messages, func(i, j int) bool {
		if messages[i].date.Equal(messages[j].date) {
			return messages[i].id < messages[j].id
		}
		if messages[i].date.IsZero() {
			return false
		}
		if messages[j].date.IsZero() {
			return true
		}
		return messages[i].date.After(messages[j].date)
	})

	return messages, nil
}

func (d *dirDriver) readMessage(path string) (dirMessage, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return dirMessage{}, err
	}

	rel, err := filepath.Rel(d.root, path)
	if err != nil {
		return dirMessage{}, err
	}
	rel = filepath.ToSlash(rel)

	reader := strings.NewReader(string(raw))
	msg, err := netmail.ReadMessage(reader)
	if err != nil {
		return dirMessage{}, err
	}

	summary := schema.MessageMetaSummary{
		ID:      rel,
		From:    strings.TrimSpace(msg.Header.Get("From")),
		Subject: strings.TrimSpace(msg.Header.Get("Subject")),
		Date:    strings.TrimSpace(msg.Header.Get("Date")),
	}

	parsedDate, err := msg.Header.Date()
	if err == nil {
		summary.Date = parsedDate.UTC().Format(time.RFC3339)
	}

	return dirMessage{
		id:        rel,
		messageID: strings.TrimSpace(msg.Header.Get("Message-Id")),
		path:      path,
		date:      parsedDate.UTC(),
		summary:   summary,
	}, nil
}

func (d *dirDriver) resolveMessagePath(id string) (string, error) {
	clean := filepath.Clean(id)
	if clean == "." || clean == "" || filepath.IsAbs(clean) {
		return "", fmt.Errorf("%w: %s", ErrMessageNotFound, id)
	}

	path := filepath.Join(d.root, clean)
	rel, err := filepath.Rel(d.root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s", ErrMessageNotFound, id)
	}

	return path, nil
}

// ── Writer interface ──────────────────────────────────────────────────────────

// Delete removes the matching .eml file from the directory.
func (d *dirDriver) Delete(ctx context.Context, id string) error {
	path, err := d.locatePath(ctx, id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrMessageNotFound, id)
		}
		return err
	}
	return nil
}

// Move copies the .eml file to <root>/<dest>/ and removes the original.
// dest is treated as a subdirectory name relative to the driver root.
func (d *dirDriver) Move(ctx context.Context, id, dest string) error {
	src, err := d.locatePath(ctx, id)
	if err != nil {
		return err
	}

	destDir := filepath.Join(d.root, filepath.Clean(dest))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	destPath := filepath.Join(destDir, filepath.Base(src))
	if err := os.WriteFile(destPath, raw, 0o644); err != nil {
		return fmt.Errorf("write to dest: %w", err)
	}

	return os.Remove(src)
}

// MarkRead renames the file with a "seen." or "unseen." prefix to signal
// read status — a lightweight convention for directory-based stores.
func (d *dirDriver) MarkRead(ctx context.Context, id string, read bool) error {
	path, err := d.locatePath(ctx, id)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// Strip any existing seen/unseen prefix.
	base = strings.TrimPrefix(base, "seen.")
	base = strings.TrimPrefix(base, "unseen.")

	prefix := "unseen."
	if read {
		prefix = "seen."
	}

	newPath := filepath.Join(dir, prefix+base)
	if path == newPath {
		return nil // already in the desired state
	}
	return os.Rename(path, newPath)
}

// locatePath resolves an ID (Message-ID or relative path) to an absolute path.
func (d *dirDriver) locatePath(ctx context.Context, id string) (string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", fmt.Errorf("message id is required")
	}

	// Message-ID format: search by header.
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		msgs, err := d.loadMessages(ctx)
		if err != nil {
			return "", err
		}
		for _, m := range msgs {
			if m.messageID == trimmed {
				return m.path, nil
			}
		}
		return "", fmt.Errorf("%w: %s", ErrMessageNotFound, id)
	}

	return d.resolveMessagePath(trimmed)
}
