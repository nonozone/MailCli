package driver

import (
	"context"

	"github.com/yourname/mailcli/pkg/schema"
)

type Driver interface {
	List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error)
	FetchRaw(ctx context.Context, id string) ([]byte, error)
	SendRaw(ctx context.Context, raw []byte) error
}
