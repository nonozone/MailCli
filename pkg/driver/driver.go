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
	ErrAuthFailed            = errors.New("auth failed")
)

type Driver interface {
	List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error)
	FetchRaw(ctx context.Context, id string) ([]byte, error)
	SendRaw(ctx context.Context, raw []byte) error
}
