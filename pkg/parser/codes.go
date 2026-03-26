package parser

import (
	"regexp"
	"strings"

	"github.com/yourname/mailcli/pkg/schema"
)

var (
	securityPhraseRe  = regexp.MustCompile(`(?i)\b(verification code|security code|one[- ]time code|login code|sign[- ]in code|two[- ]factor code|2fa code)\b`)
	codeCandidateRe   = regexp.MustCompile(`\b(?:\d[\s-]?){4,8}\b`)
	digitsOnlyRe      = regexp.MustCompile(`^\d{4,8}$`)
	verificationLabel = "Verification code"
)

func extractCodes(inputs ...string) []schema.Code {
	combined := normalizeText(strings.Join(inputs, "\n"))
	if strings.TrimSpace(combined) == "" {
		return nil
	}

	lines := strings.Split(combined, "\n")
	seen := map[string]struct{}{}
	var codes []schema.Code

	for i := range lines {
		window := strings.TrimSpace(lines[i])
		if window == "" {
			continue
		}
		if i+1 < len(lines) {
			next := strings.TrimSpace(lines[i+1])
			if next != "" && digitsOnlyRe.MatchString(strings.ReplaceAll(next, " ", "")) {
				window += "\n" + next
			}
		}

		if !securityPhraseRe.MatchString(window) {
			continue
		}

		matches := codeCandidateRe.FindAllString(window, -1)
		for _, match := range matches {
			value := normalizeCodeValue(match)
			if !digitsOnlyRe.MatchString(value) {
				continue
			}
			key := "verification_code\x00" + value
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			codes = append(codes, schema.Code{
				Type:  "verification_code",
				Value: value,
				Label: verificationLabel,
			})
		}
	}

	return codes
}

func normalizeCodeValue(input string) string {
	input = strings.ReplaceAll(input, " ", "")
	input = strings.ReplaceAll(input, "-", "")
	return strings.TrimSpace(input)
}
