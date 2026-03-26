package composer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nonozone/MailCli/pkg/schema"
)

func ComposeDraft(draft schema.DraftMessage) ([]byte, error) {
	headers := map[string]string{
		"Message-ID":   fmt.Sprintf("<%d@mailcli.local>", time.Now().UnixNano()),
		"Date":         time.Now().UTC().Format(time.RFC1123Z),
		"Subject":      draft.Subject,
		"MIME-Version": "1.0",
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

	contentType, body, err := composeMessageBody(draft.BodyText, draft.BodyMD, draft.Attachments)
	if err != nil {
		return nil, err
	}
	headers["Content-Type"] = contentType
	return renderMessage(headers, body), nil
}

func ComposeReply(draft schema.ReplyDraft) ([]byte, error) {
	headers := map[string]string{
		"Message-ID":   fmt.Sprintf("<%d@mailcli.local>", time.Now().UnixNano()),
		"Date":         time.Now().UTC().Format(time.RFC1123Z),
		"Subject":      ensureReplySubject(draft.Subject),
		"MIME-Version": "1.0",
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

	contentType, body, err := composeMessageBody(draft.BodyText, draft.BodyMD, draft.Attachments)
	if err != nil {
		return nil, err
	}
	headers["Content-Type"] = contentType
	return renderMessage(headers, body), nil
}

func renderMessage(headers map[string]string, body []byte) []byte {
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
	buf.Write(body)
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

func composeMessageBody(bodyText, bodyMD string, attachments []schema.Attachment) (string, []byte, error) {
	bodyContentType, body, err := composeBodyPart(bodyText, bodyMD)
	if err != nil {
		return "", nil, err
	}

	if len(attachments) == 0 {
		return bodyContentType, body, nil
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writeMultipartPart(writer, bodyContentType, body, map[string]string{
		"Content-Transfer-Encoding": "8bit",
	}); err != nil {
		return "", nil, err
	}

	for _, attachment := range attachments {
		if err := writeAttachmentPart(writer, attachment); err != nil {
			return "", nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return "", nil, err
	}

	return fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary()), buf.Bytes(), nil
}

func composeBodyPart(bodyText, bodyMD string) (string, []byte, error) {
	if strings.TrimSpace(bodyMD) == "" {
		return "text/plain; charset=UTF-8", []byte(strings.TrimSpace(firstBody(bodyText, bodyMD))), nil
	}

	plain := strings.TrimSpace(firstBody(bodyText, markdownToPlain(bodyMD)))
	htmlBody := strings.TrimSpace(markdownToHTML(bodyMD))

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writeMultipartPart(writer, "text/plain; charset=UTF-8", []byte(plain), map[string]string{
		"Content-Transfer-Encoding": "8bit",
	}); err != nil {
		return "", nil, err
	}
	if err := writeMultipartPart(writer, "text/html; charset=UTF-8", []byte(htmlBody), map[string]string{
		"Content-Transfer-Encoding": "8bit",
	}); err != nil {
		return "", nil, err
	}

	if err := writer.Close(); err != nil {
		return "", nil, err
	}

	return fmt.Sprintf("multipart/alternative; boundary=%s", writer.Boundary()), buf.Bytes(), nil
}

func writeMultipartPart(writer *multipart.Writer, contentType string, body []byte, extraHeaders map[string]string) error {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType)
	for key, value := range extraHeaders {
		header.Set(key, value)
	}

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	_, err = part.Write(body)
	return err
}

func writeAttachmentPart(writer *multipart.Writer, attachment schema.Attachment) error {
	path := strings.TrimSpace(attachment.Path)
	if path == "" {
		return fmt.Errorf("attachment path is required")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	name := strings.TrimSpace(attachment.Name)
	if name == "" {
		name = filepath.Base(path)
	}
	if name == "." || name == "" {
		return fmt.Errorf("attachment name is required")
	}

	contentType := strings.TrimSpace(attachment.ContentType)
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(name))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	header := textproto.MIMEHeader{}
	header.Set("Content-Type", fmt.Sprintf("%s; name=%q", contentType, name))
	header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	header.Set("Content-Transfer-Encoding", "base64")

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	_, err = part.Write([]byte(wrapBase64(base64.StdEncoding.EncodeToString(content))))
	return err
}

func markdownToPlain(input string) string {
	replacer := strings.NewReplacer(
		"**", "",
		"__", "",
		"`", "",
	)

	lines := strings.Split(strings.TrimSpace(input), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#")
		lines[i] = strings.TrimSpace(replacer.Replace(line))
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func markdownToHTML(input string) string {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	var blocks []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "### "):
			blocks = append(blocks, "<h3>"+html.EscapeString(strings.TrimSpace(strings.TrimPrefix(trimmed, "### ")))+"</h3>")
		case strings.HasPrefix(trimmed, "## "):
			blocks = append(blocks, "<h2>"+html.EscapeString(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))+"</h2>")
		case strings.HasPrefix(trimmed, "# "):
			blocks = append(blocks, "<h1>"+html.EscapeString(strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")))+"</h1>")
		default:
			blocks = append(blocks, "<p>"+html.EscapeString(trimmed)+"</p>")
		}
	}

	if len(blocks) == 0 {
		return ""
	}

	return strings.Join(blocks, "\n")
}

func wrapBase64(input string) string {
	if input == "" {
		return ""
	}

	const lineLength = 76
	var out []string
	for len(input) > lineLength {
		out = append(out, input[:lineLength])
		input = input[lineLength:]
	}
	out = append(out, input)
	return strings.Join(out, "\r\n")
}
