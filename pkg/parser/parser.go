package parser

import (
	"errors"
	"regexp"
	"strings"

	"github.com/nonozone/MailCli/pkg/schema"
)

var ErrNotImplemented = errors.New("parser not implemented")

func Parse(raw []byte) (*schema.StandardMessage, error) {
	if len(raw) == 0 {
		return nil, ErrNotImplemented
	}

	env, err := readEnvelope(raw)
	if err != nil {
		return nil, err
	}

	meta := populateMeta(env)
	bodyFormat, body, htmlBody := selectBody(env)

	var bodyMD string
	if bodyFormat == "html" {
		cleaned, err := cleanHTML(normalizeText(body))
		if err != nil {
			return nil, err
		}
		bodyMD, err = htmlToMarkdown(cleaned)
		if err != nil {
			return nil, err
		}
	} else {
		bodyMD = strings.TrimSpace(normalizeText(body))
	}

	content := schema.Content{
		Format:  "markdown",
		BodyMD:  strings.TrimSpace(bodyMD),
		Snippet: makeSnippet(bodyMD, 160),
	}

	msg := &schema.StandardMessage{
		ID:         firstNonEmpty(meta.MessageID, meta.InReplyTo),
		Meta:       meta,
		Content:    content,
		Actions:    extractActions(meta, htmlBody, extractReportAbuseTargets(env)...),
		Codes:      extractCodes(meta.Subject, env.Text, bodyMD),
		TokenUsage: estimateTokenUsage(bodyMD),
	}
	if msg.ID == "" {
		msg.ID = "unknown"
	}

	if bounce := extractBounceContext(strings.Join([]string{meta.Subject, env.Text, bodyMD}, "\n")); bounce != nil {
		msg.ErrorContext = bounce
		msg.Labels = []string{"bounce", "error"}
		msg.Content.Category = "system_error"
	}

	return msg, nil
}

var (
	statusCodeRegex   = regexp.MustCompile(`\b([245]\d{2})\b`)
	failedRecipientRe = regexp.MustCompile(`(?im)^Failed recipient:\s*(.+)$`)
	diagnosticCodeRe  = regexp.MustCompile(`(?im)^.*?([245]\d{2}.*)$`)
	originalMessageRe = regexp.MustCompile(`(?im)^Original-Message-ID:\s*(.+)$`)
	originalSubjectRe = regexp.MustCompile(`(?im)^Original-Subject:\s*(.+)$`)
)

func extractBounceContext(input string) *schema.ErrorContext {
	lower := strings.ToLower(input)
	if !strings.Contains(lower, "delivery status notification") &&
		!strings.Contains(lower, "邮件未送达") &&
		!strings.Contains(lower, "authentication credentials invalid") {
		return nil
	}

	ctx := &schema.ErrorContext{}

	if match := failedRecipientRe.FindStringSubmatch(input); len(match) > 1 {
		ctx.FailedRecipient = strings.TrimSpace(match[1])
	}
	if match := statusCodeRegex.FindStringSubmatch(input); len(match) > 1 {
		ctx.StatusCode = strings.TrimSpace(match[1])
	}
	if match := diagnosticCodeRe.FindStringSubmatch(input); len(match) > 1 {
		ctx.DiagnosticCode = strings.TrimSpace(match[1])
	}
	if match := originalMessageRe.FindStringSubmatch(input); len(match) > 1 {
		ctx.OriginalMessageID = strings.TrimSpace(match[1])
	}
	if match := originalSubjectRe.FindStringSubmatch(input); len(match) > 1 {
		ctx.OriginalSubject = strings.TrimSpace(match[1])
	}

	if ctx.FailedRecipient == "" && ctx.StatusCode == "" && ctx.DiagnosticCode == "" {
		return nil
	}

	return ctx
}

func makeSnippet(input string, limit int) string {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return ""
	}

	snippet := strings.Join(fields, " ")
	runes := []rune(snippet)
	if len(runes) <= limit {
		return snippet
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
