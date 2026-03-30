package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	mailindex "github.com/nonozone/MailCli/internal/index"
	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/parser"
	"github.com/nonozone/MailCli/pkg/schema"
)

// watchOutputEvent is the JSONL event schema emitted to stdout.
type watchOutputEvent struct {
	Event   string                  `json:"event"`
	Account string                  `json:"account,omitempty"`
	Mailbox string                  `json:"mailbox,omitempty"`
	ID      string                  `json:"id,omitempty"`
	Message *schema.StandardMessage `json:"message,omitempty"`
	Error   string                  `json:"error,omitempty"`
	TS      string                  `json:"ts"`
}

// watchMsg is an internal signal from a mailbox watcher goroutine.
type watchMsg struct {
	mailbox string
	id      string
	err     error
}

func newWatchCmd() *cobra.Command {
	var (
		configPath   string
		account      string
		mailboxes    []string
		pollInterval time.Duration
		autoSync     bool
		indexPath    string
		heartbeat    time.Duration
		since        string
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Monitor one or more mailboxes and stream new messages as JSONL events",
		Long: `Watch connects to the configured mail account and emits a JSON event to stdout
each time a new message arrives. Each line is a complete JSON object (JSONL).

Event types:
  watching     — emitted once per mailbox on startup
  new_message  — full parsed message when a new email arrives
  heartbeat    — periodic keepalive (see --heartbeat)
  error        — non-fatal connection or fetch error
  reconnecting — the driver is reconnecting after an error

IMAP accounts use IMAP IDLE (push) when available; all other drivers fall
back to polling via --poll.

Pipe the output to any AI agent or script:
  mailcli watch --account work | python3 ai_reply_agent.py`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Graceful shutdown on Ctrl+C / SIGTERM.
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			selectedAccount, err := resolveSelectedAccount(configPath, account, "")
			if err != nil {
				return err
			}

			drv, err := driverFactoryFunc(selectedAccount)
			if err != nil {
				return err
			}

			if len(mailboxes) == 0 {
				mb := strings.TrimSpace(selectedAccount.Mailbox)
				if mb == "" {
					mb = "INBOX"
				}
				mailboxes = []string{mb}
			}

			var store *mailindex.Store
			if strings.TrimSpace(indexPath) != "" {
				store = mailindex.NewFileStore(indexPath)
			}
			if autoSync && store == nil {
				return fmt.Errorf("--auto-sync requires --index")
			}

			combined := make(chan watchMsg, 128)
			var wg sync.WaitGroup
			for _, mb := range mailboxes {
				wg.Add(1)
				go func(mailbox string) {
					defer wg.Done()
					watchMailbox(ctx, drv, mailbox, pollInterval, since, store, selectedAccount.Name, combined)
				}(mb)
			}
			go func() {
				wg.Wait()
				close(combined)
			}()

			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetEscapeHTML(false)

			// Emit initial "watching" events.
			now := time.Now().UTC().Format(time.RFC3339)
			for _, mb := range mailboxes {
				_ = enc.Encode(watchOutputEvent{
					Event:   "watching",
					Account: selectedAccount.Name,
					Mailbox: mb,
					TS:      now,
				})
			}

			// Setup optional heartbeat ticker.
			var heartbeatC <-chan time.Time
			if heartbeat > 0 {
				t := time.NewTicker(heartbeat)
				defer t.Stop()
				heartbeatC = t.C
			}

			for {
				select {
				case <-ctx.Done():
					return nil

				case <-heartbeatC:
					for _, mb := range mailboxes {
						_ = enc.Encode(watchOutputEvent{
							Event:   "heartbeat",
							Account: selectedAccount.Name,
							Mailbox: mb,
							TS:      time.Now().UTC().Format(time.RFC3339),
						})
					}

				case msg, open := <-combined:
					if !open {
						return nil
					}
					ts := time.Now().UTC().Format(time.RFC3339)

					if msg.err != nil {
						_ = enc.Encode(watchOutputEvent{
							Event:   "error",
							Account: selectedAccount.Name,
							Mailbox: msg.mailbox,
							Error:   msg.err.Error(),
							TS:      ts,
						})
						continue
					}

					raw, err := drv.FetchRaw(ctx, msg.id)
					if err != nil {
						_ = enc.Encode(watchOutputEvent{
							Event:   "error",
							Account: selectedAccount.Name,
							Mailbox: msg.mailbox,
							ID:      msg.id,
							Error:   fmt.Sprintf("fetch: %v", err),
							TS:      ts,
						})
						continue
					}

					parsed, err := parser.Parse(raw)
					if err != nil {
						_ = enc.Encode(watchOutputEvent{
							Event:   "error",
							Account: selectedAccount.Name,
							Mailbox: msg.mailbox,
							ID:      msg.id,
							Error:   fmt.Sprintf("parse: %v", err),
							TS:      ts,
						})
						continue
					}

					if store != nil {
						_ = store.Upsert(mailindex.IndexedMessage{
							Account: selectedAccount.Name,
							Mailbox: msg.mailbox,
							ID:      msg.id,
							Message: *parsed,
						})
						// Mark as seen so restarts don't re-emit this message.
						_ = store.WatchMarkSeen(selectedAccount.Name, msg.mailbox, []string{msg.id})
					} else if autoSync {
						// autoSync without store: skip (store required check above handles this)
					}

					_ = enc.Encode(watchOutputEvent{
						Event:   "new_message",
						Account: selectedAccount.Name,
						Mailbox: msg.mailbox,
						ID:      msg.id,
						Message: parsed,
						TS:      ts,
					})
				}
			}
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&account, "account", "", "account name override")
	cmd.Flags().StringArrayVar(&mailboxes, "mailbox", nil, "mailbox to watch (repeatable, default: account mailbox)")
	cmd.Flags().DurationVar(&pollInterval, "poll", 30*time.Second, "polling interval when IMAP IDLE is unavailable")
	cmd.Flags().BoolVar(&autoSync, "auto-sync", false, "also write new messages into the local index")
	cmd.Flags().StringVar(&indexPath, "index", "", "local index path (required with --auto-sync)")
	cmd.Flags().DurationVar(&heartbeat, "heartbeat", 0, "emit heartbeat events at this interval (e.g. 5m), 0 = disabled")
	cmd.Flags().StringVar(&since, "since", "", "only emit events for messages on or after this RFC3339 timestamp (prevents old-mail flood on startup)")
	return cmd
}

// watchMailbox monitors a single mailbox, using IMAP IDLE if the driver
// supports it, otherwise polling at the given interval.
// since, if non-empty, suppresses events for messages older than that RFC3339 timestamp.
func watchMailbox(ctx context.Context, drv driver.Driver, mailbox string, poll time.Duration, since string, store *mailindex.Store, account string, out chan<- watchMsg) {
	if w, ok := drv.(driver.Watcher); ok {
		ch, err := w.Watch(ctx, mailbox)
		if err == nil {
			for ev := range ch {
				if ev.Err != nil {
					select {
					case out <- watchMsg{mailbox: mailbox, err: ev.Err}:
					case <-ctx.Done():
						return
					}
					continue
				}
				for _, id := range ev.NewIDs {
					select {
					case out <- watchMsg{mailbox: mailbox, id: id}:
					case <-ctx.Done():
						return
					}
				}
			}
			return
		}
	}
	// Polling fallback.
	pollWatch(ctx, drv, mailbox, poll, since, store, account, out)
}

// pollWatch emits IDs of messages that weren't present in the previous poll.
// since, if set to a valid RFC3339 string, suppresses events for messages
// whose Date header predates it — preventing an old-mail flood on startup.
// When store is non-nil, the seen set is loaded from and persisted to SQLite
// so it survives process restarts.
func pollWatch(ctx context.Context, drv driver.Driver, mailbox string, interval time.Duration, since string, store *mailindex.Store, account string, out chan<- watchMsg) {
	// Restore seen set from SQLite if available; otherwise start empty.
	var seen map[string]bool
	if store != nil {
		loaded, err := store.WatchSeen(account, mailbox)
		if err == nil {
			seen = loaded
		}
	}
	if seen == nil {
		seen = map[string]bool{}
	}
	sinceTime, _ := time.Parse(time.RFC3339, since)

	doCheck := func() {
		items, err := drv.List(ctx, schema.SearchQuery{Mailbox: mailbox, Limit: 50, Since: since})
		if err != nil {
			select {
			case out <- watchMsg{mailbox: mailbox, err: err}:
			case <-ctx.Done():
			}
			return
		}
		var newIDs []string
		for _, item := range items {
			// Client-side date guard when driver doesn't filter.
			if !sinceTime.IsZero() {
				if t, err := time.Parse(time.RFC3339, item.Date); err == nil && t.Before(sinceTime) {
					continue
				}
			}
			if !seen[item.ID] {
				seen[item.ID] = true
				newIDs = append(newIDs, item.ID)
			}
		}
		// Persist newly seen IDs before emitting events.
		if store != nil && len(newIDs) > 0 {
			_ = store.WatchMarkSeen(account, mailbox, newIDs)
		}
		for _, id := range newIDs {
			select {
			case out <- watchMsg{mailbox: mailbox, id: id}:
			case <-ctx.Done():
				return
			}
		}
	}

	// On first startup: seed seen from the live mailbox AND from persisted state
	// (already loaded above). Only seed live IDs if seen is empty (fresh start).
	if len(seen) == 0 {
		seedItems, _ := drv.List(ctx, schema.SearchQuery{Mailbox: mailbox, Limit: 50})
		var seedIDs []string
		for _, item := range seedItems {
			if !seen[item.ID] {
				seen[item.ID] = true
				seedIDs = append(seedIDs, item.ID)
			}
		}
		if store != nil && len(seedIDs) > 0 {
			_ = store.WatchMarkSeen(account, mailbox, seedIDs)
		}
	}

	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			doCheck()
		}
	}
}
