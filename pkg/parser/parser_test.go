package parser

import (
	"testing"

	"github.com/yourname/mailcli/pkg/schema"
)

func TestStandardMessageShape(t *testing.T) {
	var msg schema.StandardMessage
	raw := []byte("placeholder")

	_, err := Parse(raw)
	if err == nil {
		t.Fatalf("expected parser to fail until implemented")
	}

	_ = msg
}
