package parser

import (
	"bytes"
	"net/mail"
	"sort"
	"strings"

	"github.com/jhillyerd/enmime"
	"github.com/nonozone/MailCli/pkg/schema"
)

func readEnvelope(raw []byte) (*enmime.Envelope, error) {
	return enmime.ReadEnvelope(bytes.NewReader(raw))
}

func selectBody(env *enmime.Envelope) (format string, body string, html string) {
	if strings.TrimSpace(env.HTML) != "" {
		return "html", env.HTML, env.HTML
	}

	return "text", env.Text, ""
}

func populateMeta(env *enmime.Envelope) schema.MessageMeta {
	meta := schema.MessageMeta{
		Subject:         env.GetHeader("Subject"),
		MessageID:       env.GetHeader("Message-ID"),
		InReplyTo:       env.GetHeader("In-Reply-To"),
		ListUnsubscribe: splitHeaderLinks(env.GetHeader("List-Unsubscribe")),
		AutoSubmitted:   strings.TrimSpace(env.GetHeader("Auto-Submitted")) != "",
	}

	if date, err := env.Date(); err == nil {
		meta.Date = date.UTC().Format("2006-01-02T15:04:05Z")
	}

	if refs := parseReferences(env.GetHeader("References")); len(refs) > 0 {
		meta.References = refs
	}

	if addrs, err := env.AddressList("From"); err == nil && len(addrs) > 0 {
		meta.From = convertAddress(addrs[0])
	}

	if addrs, err := env.AddressList("To"); err == nil {
		meta.To = convertAddresses(addrs)
	}

	return meta
}

func convertAddress(addr *mail.Address) *schema.Address {
	if addr == nil {
		return nil
	}

	return &schema.Address{
		Name:    addr.Name,
		Address: addr.Address,
	}
}

func convertAddresses(addrs []*mail.Address) []schema.Address {
	out := make([]schema.Address, 0, len(addrs))
	for _, addr := range addrs {
		if addr == nil {
			continue
		}
		out = append(out, schema.Address{
			Name:    addr.Name,
			Address: addr.Address,
		})
	}
	return out
}

func parseReferences(header string) []string {
	fields := strings.Fields(header)
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func extractReportAbuseTargets(env *enmime.Envelope) []string {
	if env == nil {
		return nil
	}

	var targets []string
	for _, header := range []string{"X-Report-Abuse-To", "Report-Abuse", "X-Complaints-To"} {
		for _, value := range splitHeaderLinks(env.GetHeader(header)) {
			if strings.TrimSpace(value) != "" {
				targets = append(targets, value)
			}
		}
	}

	return targets
}

func extractInboundAttachments(env *enmime.Envelope) []schema.InboundAttachment {
	if env == nil {
		return nil
	}

	parts := make([]*enmime.Part, 0, len(env.Attachments)+len(env.Inlines)+len(env.OtherParts))
	seen := map[*enmime.Part]struct{}{}
	for _, group := range [][]*enmime.Part{env.Attachments, env.Inlines, env.OtherParts} {
		for _, part := range group {
			if part == nil {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			parts = append(parts, part)
		}
	}

	sort.SliceStable(parts, func(i, j int) bool {
		return parts[i].PartID < parts[j].PartID
	})

	attachments := make([]schema.InboundAttachment, 0, len(parts))
	for _, part := range parts {
		disposition := strings.ToLower(strings.TrimSpace(part.Disposition))
		contentID := strings.Trim(strings.TrimSpace(part.ContentID), "<>")
		attachments = append(attachments, schema.InboundAttachment{
			PartID:      strings.TrimSpace(part.PartID),
			Filename:    strings.TrimSpace(part.FileName),
			ContentType: strings.ToLower(strings.TrimSpace(part.ContentType)),
			SizeBytes:   int64(len(part.Content)),
			Disposition: disposition,
			ContentID:   contentID,
			Inline:      disposition == "inline" || (disposition == "" && contentID != ""),
		})
	}

	return attachments
}
