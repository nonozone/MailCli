package parser

import (
	"bytes"
	"net/mail"
	"strings"

	"github.com/jhillyerd/enmime"
	"github.com/yourname/mailcli/pkg/schema"
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
