package driver

import "context"

// WatchEvent is fired when new messages arrive in the monitored mailbox.
// Err is non-nil for non-fatal watch errors; the loop continues regardless.
type WatchEvent struct {
	NewIDs []string // Message-IDs (or sequence numbers) of newly arrived messages
	Err    error
}

// Watcher is an optional interface for drivers that support real-time
// mailbox monitoring. The IMAP driver implements this via IMAP IDLE.
// cmd/watch falls back to polling via List() when this is not available.
type Watcher interface {
	// Watch monitors mailbox and sends events to the returned channel until
	// ctx is cancelled. The channel is closed when monitoring stops.
	Watch(ctx context.Context, mailbox string) (<-chan WatchEvent, error)
}
