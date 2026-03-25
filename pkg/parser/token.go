package parser

import (
	"strings"

	"github.com/yourname/mailcli/pkg/schema"
)

func estimateTokenUsage(input string) *schema.TokenUsage {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	words := len(strings.Fields(input))
	estimated := words
	if estimated == 0 {
		estimated = len([]rune(input)) / 4
	}
	if estimated == 0 {
		estimated = 1
	}

	return &schema.TokenUsage{
		EstimatedInputTokens: estimated,
	}
}
