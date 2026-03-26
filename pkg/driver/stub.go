package driver

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/schema"
)

type stubDriver struct {
	mailbox  string
	messages []stubMessage
	sent     [][]byte
}

type stubMessage struct {
	id        string
	messageID string
	summary   schema.MessageMetaSummary
	raw       []byte
}

func newStubDriver(account config.AccountConfig) (Driver, error) {
	mailbox := strings.TrimSpace(account.Mailbox)
	if mailbox == "" {
		mailbox = "INBOX"
	}

	return &stubDriver{
		mailbox:  mailbox,
		messages: defaultStubMessages(),
	}, nil
}

func (d *stubDriver) List(_ context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	if mailbox := strings.TrimSpace(query.Mailbox); mailbox != "" && !strings.EqualFold(mailbox, d.mailbox) {
		return []schema.MessageMetaSummary{}, nil
	}

	results := make([]schema.MessageMetaSummary, 0, len(d.messages))
	filter := strings.ToLower(strings.TrimSpace(query.Query))

	for _, message := range d.messages {
		if filter != "" {
			haystack := strings.ToLower(strings.Join([]string{
				message.summary.ID,
				message.summary.From,
				message.summary.Subject,
			}, "\n"))
			if !strings.Contains(haystack, filter) {
				continue
			}
		}

		results = append(results, message.summary)
	}

	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

func (d *stubDriver) FetchRaw(_ context.Context, id string) ([]byte, error) {
	target := strings.TrimSpace(id)
	if target == "" {
		return nil, fmt.Errorf("message id is required")
	}

	for _, message := range d.messages {
		if target == message.id || target == message.messageID {
			return append([]byte(nil), message.raw...), nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrMessageNotFound, id)
}

func (d *stubDriver) SendRaw(_ context.Context, raw []byte) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fmt.Errorf("raw message is required")
	}

	d.sent = append(d.sent, append([]byte(nil), raw...))
	return nil
}

func defaultStubMessages() []stubMessage {
	return []stubMessage{
		{
			id:        "stub:security-reset",
			messageID: "<stub-security-reset@example.com>",
			summary: schema.MessageMetaSummary{
				ID:      "stub:security-reset",
				From:    "Example Security <security@example.com>",
				Subject: "Reset code for your workspace",
				Date:    "2026-03-25T08:30:00Z",
			},
			raw: []byte(strings.Join([]string{
				"Message-ID: <stub-security-reset@example.com>",
				"From: Example Security <security@example.com>",
				"To: agent@example.com",
				"Subject: Reset code for your workspace",
				"Date: Wed, 25 Mar 2026 08:30:00 +0000",
				"MIME-Version: 1.0",
				"Content-Type: text/plain; charset=UTF-8",
				"",
				"Use code 482991 to finish signing in.",
				"",
			}, "\r\n")),
		},
		{
			id:        "stub:invoice",
			messageID: "<stub-invoice@example.com>",
			summary: schema.MessageMetaSummary{
				ID:      "stub:invoice",
				From:    "Example Billing <billing@example.com>",
				Subject: "Invoice INV-2026-031 ready",
				Date:    "2026-03-24T16:10:00Z",
			},
			raw: []byte(strings.Join([]string{
				"Message-ID: <stub-invoice@example.com>",
				"From: Example Billing <billing@example.com>",
				"To: agent@example.com",
				"Subject: Invoice INV-2026-031 ready",
				"Date: Tue, 24 Mar 2026 16:10:00 +0000",
				"MIME-Version: 1.0",
				"Content-Type: multipart/alternative; boundary=stub-alt",
				"",
				"--stub-alt",
				"Content-Type: text/plain; charset=UTF-8",
				"",
				"Your March invoice is ready.",
				"",
				"--stub-alt",
				"Content-Type: text/html; charset=UTF-8",
				"",
				"<html><body><p>Your <strong>March invoice</strong> is ready.</p></body></html>",
				"",
				"--stub-alt--",
				"",
			}, "\r\n")),
		},
	}
}
