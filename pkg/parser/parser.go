package parser

import (
	"errors"

	"github.com/yourname/mailcli/pkg/schema"
)

var ErrNotImplemented = errors.New("parser not implemented")

func Parse(raw []byte) (*schema.StandardMessage, error) {
	return nil, ErrNotImplemented
}
