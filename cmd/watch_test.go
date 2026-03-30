package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/schema"
)

func newWatchTestStore(t *testing.T) *mailindex.Store {
	t.Helper()
	return mailindex.NewFileStore(filepath.Join(t.TempDir(), "watch.db"))
}

// ─── pollableDriver ──────────────────────────────────────────────────────────
// A fake driver that returns a fixed list — used to verify polling-based watch.

type pollableDriver struct {
	items []schema.MessageMetaSummary
	raws  map[string][]byte
}

func (p *pollableDriver) List(_ context.Context, _ schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	return p.items, nil
}

func (p *pollableDriver) FetchRaw(_ context.Context, id string) ([]byte, error) {
	if raw, ok := p.raws[id]; ok {
		return raw, nil
	}
	// Return a minimal valid EML so parser.Parse succeeds.
	return []byte("From: sender@example.com\r\nTo: me@example.com\r\nSubject: Watch test\r\nDate: Mon, 30 Mar 2026 12:00:00 +0000\r\nMessage-ID: " + id + "\r\n\r\nBody"), nil
}

func (p *pollableDriver) SendRaw(_ context.Context, _ []byte) error { return nil }

// ─── watcherDriver ───────────────────────────────────────────────────────────
// A fake driver that also implements driver.Watcher.

type watcherDriver struct {
	pollableDriver
	events []driver.WatchEvent
}

func (w *watcherDriver) Watch(_ context.Context, _ string) (<-chan driver.WatchEvent, error) {
	ch := make(chan driver.WatchEvent, len(w.events))
	for _, ev := range w.events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func writeWatchConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(
		"current_account: work\naccounts:\n  - name: work\n    driver: fake\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestWatchCommandEmitsWatchingEvent(t *testing.T) {
	cfgPath := writeWatchConfig(t)

	// A watcher driver that closes immediately (empty events).
	drv := &watcherDriver{}
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	loadConfigFunc = config.Load
	driverFactoryFunc = func(_ config.AccountConfig) (driver.Driver, error) { return drv, nil }
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"watch", "--config", cfgPath, "--mailbox", "INBOX", "--poll", "10s"})
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	// Run with timeout context — the command exits when the Watcher channel closes.
	_ = rootCmd.ExecuteContext(ctx)

	lines := nonEmptyLines(buf.String())
	if len(lines) == 0 {
		t.Fatal("expected at least one JSONL line")
	}
	var ev watchOutputEvent
	if err := json.Unmarshal([]byte(lines[0]), &ev); err != nil {
		t.Fatalf("expected valid JSON: %v\nraw: %s", err, lines[0])
	}
	if ev.Event != "watching" {
		t.Fatalf("expected first event to be 'watching', got %q", ev.Event)
	}
	if ev.Account != "work" {
		t.Fatalf("expected account 'work', got %q", ev.Account)
	}
}

func TestWatchCommandEmitsNewMessageViaWatcher(t *testing.T) {
	cfgPath := writeWatchConfig(t)

	msgID := "<watch-test@example.com>"
	drv := &watcherDriver{
		events: []driver.WatchEvent{
			{NewIDs: []string{msgID}},
		},
	}
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	loadConfigFunc = config.Load
	driverFactoryFunc = func(_ config.AccountConfig) (driver.Driver, error) { return drv, nil }
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"watch", "--config", cfgPath, "--mailbox", "INBOX"})
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	_ = rootCmd.ExecuteContext(ctx)

	newMsgEvents := 0
	for _, line := range nonEmptyLines(buf.String()) {
		var ev watchOutputEvent
		if err := json.Unmarshal([]byte(line), &ev); err == nil && ev.Event == "new_message" {
			newMsgEvents++
			if ev.ID != msgID {
				t.Fatalf("expected ID %q, got %q", msgID, ev.ID)
			}
			if ev.Message == nil {
				t.Fatal("expected Message to be populated in new_message event")
			}
		}
	}
	if newMsgEvents != 1 {
		t.Fatalf("expected 1 new_message event, got %d\noutput:\n%s", newMsgEvents, buf.String())
	}
}

func TestPollWatchDetectsNewMessages(t *testing.T) {
	msgID := "<poll-test@example.com>"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	drv := &capturePollDriver{
		laterItems: []schema.MessageMetaSummary{{ID: msgID, Subject: "Poll test"}},
	}

	out := make(chan watchMsg, 32)
	go pollWatch(ctx, drv, "INBOX", 100*time.Millisecond, "", nil, "", out)

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("expected new_message for %q but timed out", msgID)
		case msg := <-out:
			if msg.id == msgID {
				return // success
			}
		}
	}
}

// capturePollDriver returns firstItems on the first List call, laterItems after.
type capturePollDriver struct {
	calls      int
	firstItems []schema.MessageMetaSummary
	laterItems []schema.MessageMetaSummary
}

func (c *capturePollDriver) List(_ context.Context, _ schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	c.calls++
	if c.calls <= 1 {
		return c.firstItems, nil
	}
	return c.laterItems, nil
}

func (c *capturePollDriver) FetchRaw(_ context.Context, id string) ([]byte, error) {
	return []byte("From: sender@example.com\r\nTo: me@example.com\r\nSubject: Poll test\r\nDate: Mon, 30 Mar 2026 12:00:00 +0000\r\nMessage-ID: " + id + "\r\n\r\nBody"), nil
}

func (c *capturePollDriver) SendRaw(_ context.Context, _ []byte) error { return nil }

// ─── helpers ─────────────────────────────────────────────────────────────────

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

func TestPollWatchRestoresSeenStateFromSQLite(t *testing.T) {
	seenID := "<already-seen@example.com>"
	newID := "<brand-new@example.com>"

	// Both messages are visible in the mailbox.
	drv := &capturePollDriver{
		laterItems: []schema.MessageMetaSummary{
			{ID: seenID, Subject: "Old message"},
			{ID: newID, Subject: "New message"},
		},
	}

	// Pre-seed the SQLite watch state with seenID.
	store := newWatchTestStore(t)
	if err := store.WatchMarkSeen("testaccount", "INBOX", []string{seenID}); err != nil {
		t.Fatalf("WatchMarkSeen: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out := make(chan watchMsg, 32)
	go pollWatch(ctx, drv, "INBOX", 50*time.Millisecond, "", store, "testaccount", out)

	received := map[string]bool{}
	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			goto done
		case msg := <-out:
			if msg.id != "" {
				received[msg.id] = true
			}
		}
	}
done:
	if received[seenID] {
		t.Fatalf("expected already-seen ID %q NOT to be emitted, but it was", seenID)
	}
	if !received[newID] {
		t.Fatalf("expected new ID %q to be emitted, but it was not (received: %v)", newID, received)
	}
}

