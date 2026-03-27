package composer

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/pkg/schema"
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

func TestComposeDraftUsesMultipartAlternativeWhenMarkdownBodyIsPresent(t *testing.T) {
	raw, err := ComposeDraft(schema.DraftMessage{
		From: &schema.Address{
			Address: "support@nono.im",
		},
		To: []schema.Address{
			{Address: "user@example.com"},
		},
		Subject:  "Welcome",
		BodyText: "Hello from MailCLI.",
		BodyMD:   "## Hello from MailCLI",
	})
	if err != nil {
		t.Fatalf("expected draft compose to succeed: %v", err)
	}

	mime := string(raw)
	for _, token := range []string{
		"Content-Type: multipart/alternative;",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Type: text/html; charset=UTF-8",
		"Hello from MailCLI.",
		"Hello from MailCLI",
	} {
		if !strings.Contains(mime, token) {
			t.Fatalf("expected multipart alternative MIME to contain %q", token)
		}
	}
}

func TestComposeReplyPackagesAttachments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	content := "attachment body"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	raw, err := ComposeReply(schema.ReplyDraft{
		From: &schema.Address{
			Address: "support@nono.im",
		},
		To: []schema.Address{
			{Address: "user@example.com"},
		},
		Subject:          "Question",
		BodyText:         "Thanks for the email.",
		ReplyToMessageID: "<orig-123@example.com>",
		Attachments: []schema.Attachment{
			{
				Name:        "note.txt",
				ContentType: "text/plain",
				Path:        path,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected reply compose to succeed: %v", err)
	}

	mime := string(raw)
	for _, token := range []string{
		"Content-Type: multipart/mixed;",
		"Content-Disposition: attachment; filename=\"note.txt\"",
		"Content-Transfer-Encoding: base64",
		base64.StdEncoding.EncodeToString([]byte(content)),
	} {
		if !strings.Contains(mime, token) {
			t.Fatalf("expected attachment MIME to contain %q", token)
		}
	}
}

func TestComposeDraftRendersMarkdownLinksIntoHTMLAndPlainText(t *testing.T) {
	raw, err := ComposeDraft(schema.DraftMessage{
		From: &schema.Address{
			Address: "support@nono.im",
		},
		To: []schema.Address{
			{Address: "user@example.com"},
		},
		Subject: "Welcome",
		BodyMD:  "## Access your workspace\n\n[Open dashboard](https://example.com/dashboard)",
	})
	if err != nil {
		t.Fatalf("expected draft compose to succeed: %v", err)
	}

	mime := string(raw)
	for _, token := range []string{
		"Content-Type: multipart/alternative;",
		"Open dashboard: https://example.com/dashboard",
		`<a href="https://example.com/dashboard">Open dashboard</a>`,
	} {
		if !strings.Contains(mime, token) {
			t.Fatalf("expected markdown link rendering to contain %q", token)
		}
	}
}

func TestComposeDraftRendersMarkdownBulletListsIntoHTML(t *testing.T) {
	raw, err := ComposeDraft(schema.DraftMessage{
		From: &schema.Address{
			Address: "support@nono.im",
		},
		To: []schema.Address{
			{Address: "user@example.com"},
		},
		Subject: "Checklist",
		BodyMD:  "## Next steps\n\n- Confirm billing email\n- Verify sign-in settings",
	})
	if err != nil {
		t.Fatalf("expected draft compose to succeed: %v", err)
	}

	mime := string(raw)
	for _, token := range []string{
		"<ul>",
		"<li>Confirm billing email</li>",
		"<li>Verify sign-in settings</li>",
	} {
		if !strings.Contains(mime, token) {
			t.Fatalf("expected markdown list rendering to contain %q", token)
		}
	}
}
