package parser

import "strings"

func normalizeText(input string) string {
	valid := strings.ToValidUTF8(input, "")
	return strings.Map(normalizeCommonFullWidthRune, valid)
}

func normalizeCommonFullWidthRune(r rune) rune {
	switch {
	case r >= '０' && r <= '９':
		return '0' + (r - '０')
	}

	switch r {
	case '　':
		return ' '
	case '－':
		return '-'
	default:
		return r
	}
}
