package composer

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/yourname/mailcli/pkg/schema"
)

func ComposeDraft(draft schema.DraftMessage) ([]byte, error) {
	headers := map[string]string{
		"Message-ID": fmt.Sprintf("<%d@mailcli.local>", time.Now().UnixNano()),
		"Date":       time.Now().UTC().Format(time.RFC1123Z),
		"Subject":    draft.Subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}

	if draft.From != nil {
		headers["From"] = formatAddress(*draft.From)
	}
	if len(draft.To) > 0 {
		headers["To"] = formatAddressList(draft.To)
	}
	if len(draft.Cc) > 0 {
		headers["Cc"] = formatAddressList(draft.Cc)
	}
	if len(draft.Bcc) > 0 {
		headers["Bcc"] = formatAddressList(draft.Bcc)
	}
	for k, v := range draft.Headers {
		headers[k] = v
	}

	body := strings.TrimSpace(firstBody(draft.BodyText, draft.BodyMD))
	return renderMessage(headers, body), nil
}

func ComposeReply(draft schema.ReplyDraft) ([]byte, error) {
	headers := map[string]string{
		"Message-ID": fmt.Sprintf("<%d@mailcli.local>", time.Now().UnixNano()),
		"Date":       time.Now().UTC().Format(time.RFC1123Z),
		"Subject":    ensureReplySubject(draft.Subject),
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}

	if draft.From != nil {
		headers["From"] = formatAddress(*draft.From)
	}
	if len(draft.To) > 0 {
		headers["To"] = formatAddressList(draft.To)
	}
	if len(draft.Cc) > 0 {
		headers["Cc"] = formatAddressList(draft.Cc)
	}
	if len(draft.Bcc) > 0 {
		headers["Bcc"] = formatAddressList(draft.Bcc)
	}
	if strings.TrimSpace(draft.ReplyToMessageID) != "" {
		headers["In-Reply-To"] = draft.ReplyToMessageID
	}
	if len(draft.References) > 0 {
		headers["References"] = strings.Join(draft.References, " ")
	}

	body := strings.TrimSpace(firstBody(draft.BodyText, draft.BodyMD))
	return renderMessage(headers, body), nil
}

func renderMessage(headers map[string]string, body string) []byte {
	var buf bytes.Buffer
	writeHeader(&buf, "From", headers["From"])
	writeHeader(&buf, "To", headers["To"])
	writeHeader(&buf, "Cc", headers["Cc"])
	writeHeader(&buf, "Bcc", headers["Bcc"])
	writeHeader(&buf, "Subject", headers["Subject"])
	writeHeader(&buf, "Message-ID", headers["Message-ID"])
	writeHeader(&buf, "In-Reply-To", headers["In-Reply-To"])
	writeHeader(&buf, "References", headers["References"])
	writeHeader(&buf, "Date", headers["Date"])
	writeHeader(&buf, "MIME-Version", headers["MIME-Version"])
	writeHeader(&buf, "Content-Type", headers["Content-Type"])
	for key, value := range headers {
		switch key {
		case "From", "To", "Cc", "Bcc", "Subject", "Message-ID", "In-Reply-To", "References", "Date", "MIME-Version", "Content-Type":
			continue
		}
		writeHeader(&buf, key, value)
	}
	buf.WriteString("\r\n")
	buf.WriteString(body)
	return buf.Bytes()
}

func writeHeader(buf *bytes.Buffer, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

func firstBody(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

func ensureReplySubject(subject string) string {
	trimmed := strings.TrimSpace(subject)
	if strings.HasPrefix(strings.ToLower(trimmed), "re:") {
		return trimmed
	}
	if trimmed == "" {
		return "Re:"
	}
	return "Re: " + trimmed
}

func formatAddressList(addrs []schema.Address) string {
	values := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		values = append(values, formatAddress(addr))
	}
	return strings.Join(values, ", ")
}

func formatAddress(addr schema.Address) string {
	if addr.Name == "" {
		return addr.Address
	}
	return fmt.Sprintf("%s <%s>", addr.Name, addr.Address)
}
