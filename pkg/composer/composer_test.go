package composer

import (
	"strings"
	"testing"

	"github.com/yourname/mailcli/pkg/schema"
)

func TestComposeDraftIncludesCoreHeaders(t *testing.T) {
	raw, err := ComposeDraft(schema.DraftMessage{
		From: &schema.Address{
			Name:    "Nono",
			Address: "support@nono.im",
		},
		To: []schema.Address{
			{Address: "user@example.com"},
		},
		Subject:  "Welcome",
		BodyText: "Hello from MailCLI.",
	})
	if err != nil {
		t.Fatalf("expected draft compose to succeed: %v", err)
	}

	mime := string(raw)
	for _, token := range []string{
		"From: Nono <support@nono.im>",
		"To: user@example.com",
		"Subject: Welcome",
		"Message-ID:",
		"Date:",
		"Content-Type: text/plain; charset=UTF-8",
		"Hello from MailCLI.",
	} {
		if !strings.Contains(mime, token) {
			t.Fatalf("expected MIME to contain %q", token)
		}
	}
}

func TestComposeReplyAddsThreadHeaders(t *testing.T) {
	raw, err := ComposeReply(schema.ReplyDraft{
		From: &schema.Address{
			Address: "support@nono.im",
		},
		To: []schema.Address{
			{Address: "user@example.com"},
		},
		Subject:          "Re: Question",
		BodyText:         "Thanks for the email.",
		ReplyToMessageID: "<orig-123@example.com>",
		References:       []string{"<older-1@example.com>", "<orig-123@example.com>"},
	})
	if err != nil {
		t.Fatalf("expected reply compose to succeed: %v", err)
	}

	mime := string(raw)
	for _, token := range []string{
		"In-Reply-To: <orig-123@example.com>",
		"References: <older-1@example.com> <orig-123@example.com>",
		"Subject: Re: Question",
		"Thanks for the email.",
	} {
		if !strings.Contains(mime, token) {
			t.Fatalf("expected MIME to contain %q", token)
		}
	}
}
