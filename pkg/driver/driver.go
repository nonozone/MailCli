package driver

import (
	"context"
	"errors"

	"github.com/nonozone/MailCli/pkg/schema"
)

var (
	ErrMessageNotFound        = errors.New("message not found")
	ErrTransportNotConfigured = errors.New("transport not configured")
	ErrDriverConfigInvalid    = errors.New("driver config invalid")
	ErrAuthFailed             = errors.New("auth failed")
)

// Driver is the core interface for reading and sending mail.
type Driver interface {
	List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error)
	FetchRaw(ctx context.Context, id string) ([]byte, error)
	SendRaw(ctx context.Context, raw []byte) error
}

// BulkMessage is a single result entry from a BulkFetcher operation.
// Err is non-nil when the individual message could not be retrieved.
type BulkMessage struct {
	ID  string
	Raw []byte
	Err error
}

// BulkFetcher is an optional interface that drivers may implement to fetch
// multiple messages over a single connection, avoiding repeated TLS handshakes.
// cmd/sync uses this when available.
type BulkFetcher interface {
	FetchRawBulk(ctx context.Context, ids []string) ([]BulkMessage, error)
}

// Writer is an optional interface for mailbox-mutation operations.
// Drivers that support it expose Delete, Move, and MarkRead.
type Writer interface {
	Delete(ctx context.Context, id string) error
	Move(ctx context.Context, id, destMailbox string) error
	MarkRead(ctx context.Context, id string, read bool) error
}
