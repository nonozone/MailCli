package parser

import (
	"regexp"
	"strings"

	"github.com/nonozone/MailCli/pkg/schema"
)

var (
	securityPhraseRe  = regexp.MustCompile(`(?i)(\b(verification code|security code|one[- ]time code|login code|sign[- ]in code|two[- ]factor code|2fa code)\b|验证码|校验码|登录验证码|安全码|一次性验证码)`)
	codeCandidateRe   = regexp.MustCompile(`\b(?:\d[\s-]?){4,8}\b`)
	digitsOnlyRe      = regexp.MustCompile(`^\d{4,8}$`)
	expiresEnglishRe  = regexp.MustCompile(`(?i)\bexpires?\s+in\s+(\d+)\s+minutes?\b`)
	expiresChineseRe  = regexp.MustCompile(`(\d+)\s*分钟内有效`)
	verificationLabel = "Verification code"
)

func extractCodes(inputs ...string) []schema.Code {
	combined := normalizeText(strings.Join(inputs, "\n"))
	if strings.TrimSpace(combined) == "" {
		return nil
	}

	lines := strings.Split(combined, "\n")
	seen := map[string]int{}
	var codes []schema.Code

	for i := range lines {
		window := strings.TrimSpace(lines[i])
		if window == "" {
			continue
		}

		following := nextNonEmptyLines(lines, i+1, 2)
		if len(following) > 0 {
			window += "\n" + strings.Join(following, "\n")
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
			expiry := extractCodeExpirySeconds(window)
			if idx, ok := seen[key]; ok {
				if codes[idx].ExpiresInSeconds == 0 && expiry > 0 {
					codes[idx].ExpiresInSeconds = expiry
				}
				continue
			}
			seen[key] = len(codes)
			codes = append(codes, schema.Code{
				Type:             "verification_code",
				Value:            value,
				Label:            verificationLabel,
				ExpiresInSeconds: expiry,
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

func nextNonEmptyLines(lines []string, start, limit int) []string {
	var out []string
	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			out = append(out, line)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func extractCodeExpirySeconds(input string) int {
	if match := expiresEnglishRe.FindStringSubmatch(input); len(match) > 1 {
		return parseMinutesToSeconds(match[1])
	}
	if match := expiresChineseRe.FindStringSubmatch(input); len(match) > 1 {
		return parseMinutesToSeconds(match[1])
	}
	return 0
}

func parseMinutesToSeconds(value string) int {
	minutes := strings.TrimSpace(value)
	if minutes == "" {
		return 0
	}

	total := 0
	for _, r := range minutes {
		if r < '0' || r > '9' {
			return 0
		}
		total = total*10 + int(r-'0')
	}
	if total <= 0 {
		return 0
	}
	return total * 60
}
