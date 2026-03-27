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
	"regexp"
	"strings"
	"time"

	"github.com/nonozone/MailCli/pkg/schema"
)

var markdownLinkRe = regexp.MustCompile(`\[(.*?)\]\((https?://[^)\s]+)\)`)
var markdownOrderedListRe = regexp.MustCompile(`^\d+\.\s+`)
var markdownTableSeparatorRe = regexp.MustCompile(`^:?-{3,}:?$`)

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
	plainLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			plainLines = append(plainLines, "")
			continue
		}
		if cells := parseMarkdownTableRow(line); len(cells) > 0 {
			if isMarkdownTableSeparatorRow(line, len(cells)) {
				continue
			}
			line = strings.Join(cells, " | ")
		}
		line = strings.TrimLeft(line, "#")
		line = markdownLinkRe.ReplaceAllString(line, "$1: $2")
		plainLines = append(plainLines, strings.TrimSpace(replacer.Replace(line)))
	}

	return strings.TrimSpace(strings.Join(plainLines, "\n"))
}

func markdownToHTML(input string) string {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	var blocks []string
	listType := ""

	closeList := func() {
		if listType != "" {
			blocks = append(blocks, "</"+listType+">")
			listType = ""
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			closeList()
			continue
		}

		if isMarkdownTableStart(lines, i) {
			closeList()
			rendered, next := renderMarkdownTable(lines, i)
			blocks = append(blocks, rendered...)
			i = next
			continue
		}

		if strings.HasPrefix(trimmed, ">") {
			closeList()
			var quoteLines []string
			for ; i < len(lines); i++ {
				innerLine := strings.TrimSpace(lines[i])
				if !strings.HasPrefix(innerLine, ">") {
					i--
					break
				}
				content := strings.TrimSpace(strings.TrimPrefix(innerLine, ">"))
				if content == "" {
					continue
				}
				quoteLines = append(quoteLines, "<p>"+renderInlineHTML(content)+"</p>")
			}
			if len(quoteLines) > 0 {
				blocks = append(blocks, "<blockquote>")
				blocks = append(blocks, quoteLines...)
				blocks = append(blocks, "</blockquote>")
			}
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "### "):
			closeList()
			blocks = append(blocks, "<h3>"+renderInlineHTML(strings.TrimSpace(strings.TrimPrefix(trimmed, "### ")))+"</h3>")
		case strings.HasPrefix(trimmed, "## "):
			closeList()
			blocks = append(blocks, "<h2>"+renderInlineHTML(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))+"</h2>")
		case strings.HasPrefix(trimmed, "# "):
			closeList()
			blocks = append(blocks, "<h1>"+renderInlineHTML(strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")))+"</h1>")
		case strings.HasPrefix(trimmed, "- "):
			if listType != "ul" {
				closeList()
				blocks = append(blocks, "<ul>")
				listType = "ul"
			}
			blocks = append(blocks, "<li>"+renderInlineHTML(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))+"</li>")
		case markdownOrderedListRe.MatchString(trimmed):
			if listType != "ol" {
				closeList()
				blocks = append(blocks, "<ol>")
				listType = "ol"
			}
			blocks = append(blocks, "<li>"+renderInlineHTML(strings.TrimSpace(markdownOrderedListRe.ReplaceAllString(trimmed, "")))+"</li>")
		default:
			closeList()
			blocks = append(blocks, "<p>"+renderInlineHTML(trimmed)+"</p>")
		}
	}
	closeList()

	if len(blocks) == 0 {
		return ""
	}

	return strings.Join(blocks, "\n")
}

func isMarkdownTableStart(lines []string, index int) bool {
	if index+1 >= len(lines) {
		return false
	}
	header := parseMarkdownTableRow(lines[index])
	if len(header) < 2 {
		return false
	}
	return isMarkdownTableSeparatorRow(lines[index+1], len(header))
}

func renderMarkdownTable(lines []string, start int) ([]string, int) {
	header := parseMarkdownTableRow(lines[start])
	blocks := []string{"<table>", "<thead>", "<tr>"}
	for _, cell := range header {
		blocks = append(blocks, "<th>"+renderInlineHTML(cell)+"</th>")
	}
	blocks = append(blocks, "</tr>", "</thead>", "<tbody>")

	i := start + 2
	for ; i < len(lines); i++ {
		cells := parseMarkdownTableRow(lines[i])
		if len(cells) == 0 || len(cells) != len(header) || isMarkdownTableSeparatorRow(lines[i], len(cells)) {
			i--
			break
		}
		blocks = append(blocks, "<tr>")
		for _, cell := range cells {
			blocks = append(blocks, "<td>"+renderInlineHTML(cell)+"</td>")
		}
		blocks = append(blocks, "</tr>")
	}

	blocks = append(blocks, "</tbody>", "</table>")
	return blocks, i
}

func parseMarkdownTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") || strings.HasPrefix(trimmed, ">") {
		return nil
	}

	parts := strings.Split(trimmed, "|")
	if len(parts) < 3 {
		return nil
	}
	if strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:]
	}
	if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) < 2 {
		return nil
	}

	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func isMarkdownTableSeparatorRow(line string, columns int) bool {
	cells := parseMarkdownTableRow(line)
	if len(cells) != columns || len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if !markdownTableSeparatorRe.MatchString(strings.TrimSpace(cell)) {
			return false
		}
	}
	return true
}

func renderInlineHTML(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	indexes := markdownLinkRe.FindAllStringSubmatchIndex(input, -1)
	if len(indexes) == 0 {
		return html.EscapeString(input)
	}

	var buf strings.Builder
	last := 0
	for _, match := range indexes {
		if len(match) < 6 {
			continue
		}
		buf.WriteString(html.EscapeString(input[last:match[0]]))
		label := html.EscapeString(input[match[2]:match[3]])
		url := html.EscapeString(input[match[4]:match[5]])
		buf.WriteString(`<a href="`)
		buf.WriteString(url)
		buf.WriteString(`">`)
		buf.WriteString(label)
		buf.WriteString(`</a>`)
		last = match[1]
	}
	buf.WriteString(html.EscapeString(input[last:]))
	return buf.String()
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
