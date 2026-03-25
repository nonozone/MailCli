package parser

import "strings"

func normalizeText(input string) string {
	return strings.ToValidUTF8(input, "")
}
