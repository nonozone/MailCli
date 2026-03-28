package parser

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/pkg/schema"
)

func TestStandardMessageShape(t *testing.T) {
	var msg schema.StandardMessage
	raw := []byte("placeholder")

	_, err := Parse(raw)
	if err == nil {
		t.Fatalf("expected parser to fail until implemented")
	}

	_ = msg
}

func TestParseMercuryEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/mercury.eml", "../../testdata/golden/mercury.json")
}

func TestParseBounceEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/bounce.eml", "../../testdata/golden/bounce.json")
}

func TestParsePlaintextEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/plaintext.eml", "../../testdata/golden/plaintext.json")
}

func TestParseVerificationEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/verification.eml", "../../testdata/golden/verification.json")
}

func TestParseChineseFullWidthVerificationEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/verification_cn_fullwidth.eml", "../../testdata/golden/verification_cn_fullwidth.json")
}

func TestParseInvoiceEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/invoice.eml", "../../testdata/golden/invoice.json")
}

func TestParseChineseInvoiceEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/invoice_cn.eml", "../../testdata/golden/invoice_cn.json")
}

func TestParseSecurityResetEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/security_reset.eml", "../../testdata/golden/security_reset.json")
}

func TestParseChineseResetPasswordEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/security_reset_cn.eml", "../../testdata/golden/security_reset_cn.json")
}

func TestParseSecurityResetSafeLinksEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/security_reset_safelinks.eml", "../../testdata/golden/security_reset_safelinks.json")
}

func TestParseChineseVerifySignInEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/security_verify_cn.eml", "../../testdata/golden/security_verify_cn.json")
}

func TestParseAttachmentNoticeEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/attachment_notice.eml", "../../testdata/golden/attachment_notice.json")
}

func TestParseMixedUnsubscribeEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/unsubscribe_mixed.eml", "../../testdata/golden/unsubscribe_mixed.json")
}

func TestParseChineseUnsubscribeEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/unsubscribe_cn.eml", "../../testdata/golden/unsubscribe_cn.json")
}

func TestParseChineseConfirmSubscriptionEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/confirm_subscription_cn.eml", "../../testdata/golden/confirm_subscription_cn.json")
}

func TestParseRelatedInlineImageEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/related_inline_image.eml", "../../testdata/golden/related_inline_image.json")
}

func TestParseReplyQuotedVerificationEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/reply_quoted_verification.eml", "../../testdata/golden/reply_quoted_verification.json")
}

func TestParsePostfixBounceEmail(t *testing.T) {
	assertFixtureMatchesGolden(t, "../../testdata/emails/postfix_bounce.eml", "../../testdata/golden/postfix_bounce.json")
}

func assertFixtureMatchesGolden(t *testing.T, fixturePath, goldenPath string) {
	t.Helper()

	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}

	assertJSONMatchesGolden(t, got, want)
}

func TestParsePrefersHTMLOverPlainText(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/emails/mercury.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got.Content.BodyMD, "Real-time financial clarity") {
		t.Fatalf("expected markdown converted from html body")
	}
}

func TestParseCleansHTMLAndConvertsToMarkdown(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/emails/mercury.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(got.Content.BodyMD, "<style") {
		t.Fatalf("expected style tags to be removed")
	}

	if !strings.Contains(got.Content.BodyMD, "https://app.mercury.com/insights") {
		t.Fatalf("expected links to survive markdown conversion")
	}
}

func TestParseExtractsUnsubscribeAction(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/emails/mercury.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Actions) == 0 {
		t.Fatalf("expected at least one action")
	}
}

func TestParseExtractsMailtoUnsubscribeAction(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/emails/unsubscribe_mixed.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Meta.ListUnsubscribe) != 2 {
		t.Fatalf("expected two normalized unsubscribe header entries, got %+v", got.Meta.ListUnsubscribe)
	}
	if got.Meta.ListUnsubscribe[0] != "mailto:leave@example.com?subject=unsubscribe" {
		t.Fatalf("expected mailto unsubscribe header to be preserved, got %q", got.Meta.ListUnsubscribe[0])
	}
	if got.Meta.ListUnsubscribe[1] != "https://example.com/unsubscribe" {
		t.Fatalf("expected https unsubscribe header to be normalized, got %q", got.Meta.ListUnsubscribe[1])
	}

	foundMailto := false
	foundHTTPS := false
	for _, action := range got.Actions {
		if action.Type != "unsubscribe" {
			continue
		}
		switch action.URL {
		case "mailto:leave@example.com?subject=unsubscribe":
			foundMailto = true
		case "https://example.com/unsubscribe":
			foundHTTPS = true
		}
	}
	if !foundMailto || !foundHTTPS {
		t.Fatalf("expected both mailto and https unsubscribe actions, got %+v", got.Actions)
	}
}

func TestParseBounceEmailExtractsErrorContext(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/emails/bounce.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if got.ErrorContext == nil || got.ErrorContext.StatusCode == "" {
		t.Fatalf("expected bounce status code to be extracted")
	}
}

func TestParsePostfixBounceExtractsErrorContext(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/emails/postfix_bounce.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if got.ErrorContext == nil {
		t.Fatalf("expected postfix-style bounce context to be extracted")
	}
	if got.ErrorContext.FailedRecipient != "missing@example.com" {
		t.Fatalf("expected failed recipient to be extracted, got %+v", got.ErrorContext)
	}
	if got.ErrorContext.StatusCode != "550" {
		t.Fatalf("expected smtp status code to be extracted, got %+v", got.ErrorContext)
	}
	if got.Content.Category != "system_error" {
		t.Fatalf("expected bounce content category, got %q", got.Content.Category)
	}
	if len(got.Labels) == 0 {
		t.Fatalf("expected bounce labels, got %+v", got.Labels)
	}
}

func TestParseExtractsVerificationCode(t *testing.T) {
	raw := []byte("From: Security Team <security@example.com>\r\nTo: user@example.com\r\nSubject: Your verification code\r\nMessage-ID: <verify-inline@example.com>\r\nDate: Thu, 26 Mar 2026 12:00:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nYour verification code is 123 456.\r\nUse this one-time code to sign in.\r\nThis code expires in 10 minutes.")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Codes) != 1 {
		t.Fatalf("expected one extracted code, got %+v", got.Codes)
	}
	if got.Codes[0].Type != "verification_code" {
		t.Fatalf("expected verification_code, got %q", got.Codes[0].Type)
	}
	if got.Codes[0].Value != "123456" {
		t.Fatalf("expected normalized code value, got %q", got.Codes[0].Value)
	}
	if got.Codes[0].Label != "Verification code" {
		t.Fatalf("expected verification label, got %q", got.Codes[0].Label)
	}
	if got.Codes[0].ExpiresInSeconds != 600 {
		t.Fatalf("expected expiry in seconds, got %d", got.Codes[0].ExpiresInSeconds)
	}
}

func TestParseDoesNotExtractOrderNumberAsVerificationCode(t *testing.T) {
	raw := []byte("From: Store <store@example.com>\r\nTo: user@example.com\r\nSubject: Order received\r\nMessage-ID: <order-inline@example.com>\r\nDate: Thu, 26 Mar 2026 12:10:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nYour order number is 123456.\r\nWe are processing it now.")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Codes) != 0 {
		t.Fatalf("expected no verification codes, got %+v", got.Codes)
	}
}

func TestParseExtractsVerificationCodeFromNextNonEmptyLine(t *testing.T) {
	raw := []byte("From: Security Team <security@example.com>\r\nTo: user@example.com\r\nSubject: Your verification code\r\nMessage-ID: <verify-next-line@example.com>\r\nDate: Thu, 26 Mar 2026 12:20:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nYour verification code is:\r\n\r\n654 321\r\n\r\nEnter it to continue signing in.")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Codes) != 1 {
		t.Fatalf("expected one extracted code, got %+v", got.Codes)
	}
	if got.Codes[0].Value != "654321" {
		t.Fatalf("expected normalized next-line code value, got %q", got.Codes[0].Value)
	}
}

func TestParseExtractsChineseVerificationCode(t *testing.T) {
	raw := []byte("From: 安全中心 <security@example.com>\r\nTo: user@example.com\r\nSubject: 登录验证码\r\nMessage-ID: <verify-cn@example.com>\r\nDate: Thu, 26 Mar 2026 12:30:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n您的验证码是 246810，5 分钟内有效。")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Codes) != 1 {
		t.Fatalf("expected one extracted code from chinese verification mail, got %+v", got.Codes)
	}
	if got.Codes[0].Value != "246810" {
		t.Fatalf("expected chinese code value, got %q", got.Codes[0].Value)
	}
	if got.Codes[0].ExpiresInSeconds != 300 {
		t.Fatalf("expected chinese expiry in seconds, got %d", got.Codes[0].ExpiresInSeconds)
	}
}

func TestParseExtractsChineseVerificationCodeWithFullWidthDigits(t *testing.T) {
	raw := []byte("From: 安全中心 <security@example.com>\r\nTo: user@example.com\r\nSubject: 登录验证码\r\nMessage-ID: <verify-cn-fullwidth@example.com>\r\nDate: Thu, 26 Mar 2026 12:35:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n您的登录验证码：１２３ ４５６，请在 5 分钟内有效期内使用。")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Codes) != 1 {
		t.Fatalf("expected one extracted code from chinese full-width verification mail, got %+v", got.Codes)
	}
	if got.Codes[0].Value != "123456" {
		t.Fatalf("expected normalized full-width code value, got %q", got.Codes[0].Value)
	}
	if got.Codes[0].ExpiresInSeconds != 300 {
		t.Fatalf("expected chinese full-width expiry in seconds, got %d", got.Codes[0].ExpiresInSeconds)
	}
}

func TestParseIgnoresQuotedVerificationCodeInReply(t *testing.T) {
	raw := []byte("From: user@example.com\r\nTo: support@example.com\r\nSubject: Re: Your verification code\r\nMessage-ID: <reply-quoted-code@example.com>\r\nDate: Thu, 26 Mar 2026 12:40:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nThanks, this is resolved.\r\n\r\nOn Wed, 25 Mar 2026 at 10:00, Security Team <security@example.com> wrote:\r\n> Your verification code is 123 456.\r\n> Use this one-time code to sign in.\r\n> This code expires in 10 minutes.")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Codes) != 0 {
		t.Fatalf("expected quoted verification code to be ignored, got %+v", got.Codes)
	}
}

func TestCleanHTMLPrefersMainContentRoot(t *testing.T) {
	input := `<html><body>
<header><p>Account navigation</p></header>
<nav><a href="https://example.com/home">Home</a></nav>
<main>
  <h1>Primary update</h1>
  <p>Main body for the agent.</p>
</main>
<footer><p>Footer links</p></footer>
</body></html>`

	cleaned, err := cleanHTML(input)
	if err != nil {
		t.Fatal(err)
	}

	markdown, err := htmlToMarkdown(cleaned)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(markdown, "Primary update") || !strings.Contains(markdown, "Main body for the agent.") {
		t.Fatalf("expected main content to survive cleanup")
	}
	if strings.Contains(markdown, "Account navigation") || strings.Contains(markdown, "Footer links") || strings.Contains(markdown, "Home") {
		t.Fatalf("expected layout noise to be removed, got %q", markdown)
	}
}

func TestCleanHTMLFallsBackToArticleRoot(t *testing.T) {
	input := `<html><body>
<div>Top banner</div>
<article>
  <h2>Article body</h2>
  <p>Useful content.</p>
</article>
<footer>Unrelated footer</footer>
</body></html>`

	cleaned, err := cleanHTML(input)
	if err != nil {
		t.Fatal(err)
	}

	markdown, err := htmlToMarkdown(cleaned)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(markdown, "Article body") || !strings.Contains(markdown, "Useful content.") {
		t.Fatalf("expected article content to survive cleanup")
	}
	if strings.Contains(markdown, "Unrelated footer") {
		t.Fatalf("expected footer noise to be removed")
	}
}

func TestCleanHTMLFallsBackToRoleMainRoot(t *testing.T) {
	input := `<html><body>
<div>Top banner</div>
<section role="main">
  <h2>Role main body</h2>
  <p>Useful role-based content.</p>
</section>
<footer>Unrelated footer</footer>
</body></html>`

	cleaned, err := cleanHTML(input)
	if err != nil {
		t.Fatal(err)
	}

	markdown, err := htmlToMarkdown(cleaned)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(markdown, "Role main body") || !strings.Contains(markdown, "Useful role-based content.") {
		t.Fatalf("expected role=main content to survive cleanup")
	}
	if strings.Contains(markdown, "Unrelated footer") {
		t.Fatalf("expected footer noise to be removed")
	}
}

func TestCleanHTMLPreservesHeaderInsideContentRoot(t *testing.T) {
	input := `<html><body>
<article>
  <header><h1>Article heading</h1></header>
  <p>Important content.</p>
</article>
</body></html>`

	cleaned, err := cleanHTML(input)
	if err != nil {
		t.Fatal(err)
	}

	markdown, err := htmlToMarkdown(cleaned)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(markdown, "Article heading") || !strings.Contains(markdown, "Important content.") {
		t.Fatalf("expected header inside content root to be preserved, got %q", markdown)
	}
}

func TestCleanHTMLFallsBackToBodyWhenNoSemanticRootExists(t *testing.T) {
	input := `<html><body>
<div>
  <h2>Body fallback</h2>
  <p>Useful content without semantic wrappers.</p>
</div>
<form><button>Dismiss</button></form>
</body></html>`

	cleaned, err := cleanHTML(input)
	if err != nil {
		t.Fatal(err)
	}

	markdown, err := htmlToMarkdown(cleaned)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(markdown, "Body fallback") || !strings.Contains(markdown, "Useful content without semantic wrappers.") {
		t.Fatalf("expected body fallback content to survive cleanup")
	}
	if strings.Contains(markdown, "Dismiss") {
		t.Fatalf("expected form noise to be removed")
	}
}

func TestCleanHTMLBodyFallbackRemovesTopLevelHeaderNoise(t *testing.T) {
	input := `<html><body>
<header><p>Marketing banner</p></header>
<div>
  <h2>Body fallback</h2>
  <p>Useful content without semantic wrappers.</p>
</div>
</body></html>`

	cleaned, err := cleanHTML(input)
	if err != nil {
		t.Fatal(err)
	}

	markdown, err := htmlToMarkdown(cleaned)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(markdown, "Body fallback") || !strings.Contains(markdown, "Useful content without semantic wrappers.") {
		t.Fatalf("expected body fallback content to survive cleanup")
	}
	if strings.Contains(markdown, "Marketing banner") {
		t.Fatalf("expected top-level header noise to be removed")
	}
}

func TestCleanHTMLPrefersDenseContentContainerWithoutSemanticRoot(t *testing.T) {
	input := `<html><body>
<div class="preheader">Preview text and account links</div>
<table><tr><td><a href="https://example.com/home">Home</a></td></tr></table>
<div class="shell">
  <div class="hero">Top campaign banner</div>
  <div class="content">
    <h1>Quarterly update</h1>
    <p>The main report is ready for review.</p>
    <p><a href="https://example.com/report">Open report</a></p>
  </div>
</div>
<div class="footer">
  <p>Manage preferences</p>
  <p><a href="https://example.com/unsubscribe">Unsubscribe</a></p>
</div>
</body></html>`

	cleaned, err := cleanHTML(input)
	if err != nil {
		t.Fatal(err)
	}

	markdown, err := htmlToMarkdown(cleaned)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(markdown, "Quarterly update") || !strings.Contains(markdown, "The main report is ready for review.") {
		t.Fatalf("expected dense content container to be selected, got %q", markdown)
	}
	if strings.Contains(markdown, "Preview text") || strings.Contains(markdown, "Manage preferences") || strings.Contains(markdown, "Unsubscribe") {
		t.Fatalf("expected surrounding email chrome to be removed, got %q", markdown)
	}
}

func TestCleanHTMLKeepsPrimaryContentInsidePreferencesContainer(t *testing.T) {
	input := `<html><body>
<div id="preferences">
  <h1>Email preferences updated</h1>
  <p>Your notification settings were saved successfully.</p>
</div>
<div><a href="https://example.com/unsubscribe">Unsubscribe</a></div>
</body></html>`

	cleaned, err := cleanHTML(input)
	if err != nil {
		t.Fatal(err)
	}

	markdown, err := htmlToMarkdown(cleaned)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(markdown, "Email preferences updated") || !strings.Contains(markdown, "Your notification settings were saved successfully.") {
		t.Fatalf("expected legitimate preferences content to survive cleanup, got %q", markdown)
	}
}

func TestCleanHTMLDoesNotPreferLinkHeavyHeaderOverShortTransactionalContent(t *testing.T) {
	input := `<html><body>
<table><tr>
  <td><a href="https://example.com/1">One</a></td>
  <td><a href="https://example.com/2">Two</a></td>
  <td><a href="https://example.com/3">Three</a></td>
  <td><a href="https://example.com/4">Four</a></td>
  <td><a href="https://example.com/5">Five</a></td>
  <td><a href="https://example.com/6">Six</a></td>
</tr></table>
<div>
  <p>Your code is 123456.</p>
</div>
</body></html>`

	cleaned, err := cleanHTML(input)
	if err != nil {
		t.Fatal(err)
	}

	markdown, err := htmlToMarkdown(cleaned)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(markdown, "Your code is 123456.") {
		t.Fatalf("expected transactional body to win over link-heavy header, got %q", markdown)
	}
	if strings.Contains(markdown, "One") && strings.Contains(markdown, "Six") {
		t.Fatalf("expected link-heavy header chrome to stay out of primary body, got %q", markdown)
	}
}

func TestParseOmitsUnsubscribeFooterFromBodyMarkdown(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/emails/mercury.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(got.Content.BodyMD, "Unsubscribe") {
		t.Fatalf("expected unsubscribe footer to be omitted from body markdown, got %q", got.Content.BodyMD)
	}
	if strings.Contains(got.Content.BodyMD, "https://email.mercury.com/unsubscribe/token123") {
		t.Fatalf("expected unsubscribe url to stay out of body markdown, got %q", got.Content.BodyMD)
	}
}

func TestParseCleansTrackedURLsInMarkdown(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Tracked link\r\nMessage-ID: <tracked-1@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<html><body><p><a href=\"https://tracker.example.com/click?redirect=https%3A%2F%2Fapp.example.com%2Freport\">Open report</a></p></body></html>")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got.Content.BodyMD, "https://app.example.com/report") {
		t.Fatalf("expected markdown to use cleaned target url, got %q", got.Content.BodyMD)
	}
	if strings.Contains(got.Content.BodyMD, "tracker.example.com") {
		t.Fatalf("expected markdown to avoid tracked wrapper url, got %q", got.Content.BodyMD)
	}
}

func TestExtractActionsCleansTrackedUnsubscribeURLs(t *testing.T) {
	meta := schema.MessageMeta{
		ListUnsubscribe: []string{
			"https://tracker.example.com/click?redirect=https%3A%2F%2Fexample.com%2Funsubscribe",
		},
	}

	actions := extractActions(meta, `<a href="https://tracker.example.com/click?redirect=https%3A%2F%2Fexample.com%2Funsubscribe">Unsubscribe</a>`)
	if len(actions) == 0 {
		t.Fatalf("expected unsubscribe action")
	}
	if actions[0].URL != "https://example.com/unsubscribe" {
		t.Fatalf("expected cleaned unsubscribe url, got %q", actions[0].URL)
	}
}

func TestParseCleansNestedTrackedURLsInMarkdown(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Nested tracked link\r\nMessage-ID: <tracked-nested@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<html><body><p><a href=\"https://tracker.example.com/click?redirect=https%3A%2F%2Flinks.example.com%2Fout%3Furl%3Dhttps%253A%252F%252Fapp.example.com%252Freports%252Fquarterly\">Open report</a></p></body></html>")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got.Content.BodyMD, "https://app.example.com/reports/quarterly") {
		t.Fatalf("expected markdown to resolve nested tracked target url, got %q", got.Content.BodyMD)
	}
	if strings.Contains(got.Content.BodyMD, "tracker.example.com") || strings.Contains(got.Content.BodyMD, "links.example.com/out") {
		t.Fatalf("expected markdown to avoid nested tracking wrappers, got %q", got.Content.BodyMD)
	}
}

func TestParseCleansProofpointTrackedURLsInMarkdown(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Proofpoint tracked link\r\nMessage-ID: <tracked-proofpoint@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<html><body><p><a href=\"https://urldefense.proofpoint.com/v2/url?u=https%3A%2F%2Fapp.example.com%2Freports%2Fweekly\">Open report</a></p></body></html>")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got.Content.BodyMD, "https://app.example.com/reports/weekly") {
		t.Fatalf("expected markdown to resolve proofpoint target url, got %q", got.Content.BodyMD)
	}
	if strings.Contains(got.Content.BodyMD, "urldefense.proofpoint.com") {
		t.Fatalf("expected markdown to avoid proofpoint wrapper url, got %q", got.Content.BodyMD)
	}
}

func TestParseCleansBarracudaTrackedURLsInMarkdown(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Barracuda tracked link\r\nMessage-ID: <tracked-barracuda@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<html><body><p><a href=\"https://linkprotect.cudasvc.com/url?a=https%3A%2F%2Fapp.example.com%2Freports%2Fmonthly\">Open report</a></p></body></html>")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got.Content.BodyMD, "https://app.example.com/reports/monthly") {
		t.Fatalf("expected markdown to resolve barracuda target url, got %q", got.Content.BodyMD)
	}
	if strings.Contains(got.Content.BodyMD, "linkprotect.cudasvc.com") {
		t.Fatalf("expected markdown to avoid barracuda wrapper url, got %q", got.Content.BodyMD)
	}
}

func TestExtractActionsCleansPathEmbeddedTrackedURLs(t *testing.T) {
	meta := schema.MessageMeta{
		ListUnsubscribe: []string{
			"https://tracker.example.com/click/https%3A%2F%2Fexample.com%2Funsubscribe",
		},
	}

	actions := extractActions(meta, `<a href="https://tracker.example.com/click/https%3A%2F%2Fexample.com%2Funsubscribe">Unsubscribe</a>`)
	if len(actions) == 0 {
		t.Fatalf("expected unsubscribe action")
	}
	if actions[0].URL != "https://example.com/unsubscribe" {
		t.Fatalf("expected path-embedded tracked url to be cleaned, got %q", actions[0].URL)
	}
}

func TestExtractActionsCleansProofpointTrackedURLs(t *testing.T) {
	meta := schema.MessageMeta{
		ListUnsubscribe: []string{
			"https://urldefense.proofpoint.com/v2/url?u=https%3A%2F%2Fexample.com%2Funsubscribe",
		},
	}

	actions := extractActions(meta, `<a href="https://urldefense.proofpoint.com/v2/url?u=https%3A%2F%2Fexample.com%2Funsubscribe">Unsubscribe</a>`)
	if len(actions) == 0 {
		t.Fatalf("expected unsubscribe action")
	}
	if actions[0].URL != "https://example.com/unsubscribe" {
		t.Fatalf("expected proofpoint tracked url to be cleaned, got %q", actions[0].URL)
	}
}

func TestExtractActionsCleansBarracudaTrackedURLs(t *testing.T) {
	meta := schema.MessageMeta{
		ListUnsubscribe: []string{
			"https://linkprotect.cudasvc.com/url?a=https%3A%2F%2Fexample.com%2Funsubscribe",
		},
	}

	actions := extractActions(meta, `<a href="https://linkprotect.cudasvc.com/url?a=https%3A%2F%2Fexample.com%2Funsubscribe">Unsubscribe</a>`)
	if len(actions) == 0 {
		t.Fatalf("expected unsubscribe action")
	}
	if actions[0].URL != "https://example.com/unsubscribe" {
		t.Fatalf("expected barracuda tracked url to be cleaned, got %q", actions[0].URL)
	}
}

func TestExtractActionsKeepsOrdinaryURLsUnchanged(t *testing.T) {
	meta := schema.MessageMeta{
		ListUnsubscribe: []string{
			"https://example.com/unsubscribe",
		},
	}

	actions := extractActions(meta, "")
	if len(actions) == 0 {
		t.Fatalf("expected unsubscribe action")
	}
	if actions[0].URL != "https://example.com/unsubscribe" {
		t.Fatalf("expected ordinary unsubscribe url to stay unchanged, got %q", actions[0].URL)
	}
}

func TestParseKeepsNonWrapperTargetURLsUnchanged(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Target link\r\nMessage-ID: <tracked-2@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<html><body><p><a href=\"https://example.com/login?target=https%3A%2F%2Fexample.com%2Fapp\">Continue</a></p></body></html>")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got.Content.BodyMD, "https://example.com/login?target=https%3A%2F%2Fexample.com%2Fapp") {
		t.Fatalf("expected non-wrapper target url to remain unchanged, got %q", got.Content.BodyMD)
	}
}

func TestExtractActionsKeepsNonWrapperRedirectParamURLsUnchanged(t *testing.T) {
	meta := schema.MessageMeta{
		ListUnsubscribe: []string{
			"https://example.com/preferences?redirect_uri=https%3A%2F%2Fexample.com%2Funsubscribe",
		},
	}

	actions := extractActions(meta, "")
	if len(actions) == 0 {
		t.Fatalf("expected unsubscribe action")
	}
	if actions[0].URL != "https://example.com/preferences?redirect_uri=https%3A%2F%2Fexample.com%2Funsubscribe" {
		t.Fatalf("expected non-wrapper redirect-param url to stay unchanged, got %q", actions[0].URL)
	}
}

func TestExtractActionsClassifiesViewOnlineLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.com/view">View online</a>`)
	action := findAction(actions, "view_online")
	if action == nil {
		t.Fatalf("expected view_online action, got %+v", actions)
	}
	if action.URL != "https://example.com/view" {
		t.Fatalf("expected view_online url, got %q", action.URL)
	}
	if action.Label != "View online" {
		t.Fatalf("expected preserved label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesConfirmSubscriptionLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.com/confirm">Confirm subscription</a>`)
	action := findAction(actions, "confirm_subscription")
	if action == nil {
		t.Fatalf("expected confirm_subscription action, got %+v", actions)
	}
	if action.URL != "https://example.com/confirm" {
		t.Fatalf("expected confirm_subscription url, got %q", action.URL)
	}
	if action.Label != "Confirm subscription" {
		t.Fatalf("expected preserved label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesReportAbuseLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="mailto:abuse@example.com">Report abuse</a>`)
	action := findAction(actions, "report_abuse")
	if action == nil {
		t.Fatalf("expected report_abuse action, got %+v", actions)
	}
	if action.URL != "mailto:abuse@example.com" {
		t.Fatalf("expected report_abuse url, got %q", action.URL)
	}
	if action.Label != "Report abuse" {
		t.Fatalf("expected preserved label, got %q", action.Label)
	}
}

func TestExtractActionsDeduplicatesHTMLActions(t *testing.T) {
	html := `<a href="https://example.com/view">View online</a><a href="https://example.com/view">View online</a>`
	actions := extractActions(schema.MessageMeta{}, html)
	count := 0
	for _, action := range actions {
		if action.Type == "view_online" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one deduplicated view_online action, got %d", count)
	}
}

func TestExtractActionsClassifiesViewOnlineFromHrefPattern(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.com/view-online/abc">Read in browser</a>`)
	action := findAction(actions, "view_online")
	if action == nil {
		t.Fatalf("expected href-driven view_online action, got %+v", actions)
	}
}

func TestExtractActionsClassifiesConfirmSubscriptionFromHrefPattern(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.com/confirm-subscription/token">Click here</a>`)
	action := findAction(actions, "confirm_subscription")
	if action == nil {
		t.Fatalf("expected href-driven confirm_subscription action, got %+v", actions)
	}
}

func TestExtractActionsClassifiesChineseConfirmSubscriptionLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/welcome?token=42">确认订阅</a>`)
	action := findAction(actions, "confirm_subscription")
	if action == nil {
		t.Fatalf("expected chinese confirm_subscription action, got %+v", actions)
	}
	if action.URL != "https://example.cn/welcome?token=42" {
		t.Fatalf("expected confirm_subscription url, got %q", action.URL)
	}
	if action.Label != "确认订阅" {
		t.Fatalf("expected preserved confirm_subscription label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesChineseEmailConfirmSubscriptionLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/welcome?token=43">确认邮件订阅</a>`)
	action := findAction(actions, "confirm_subscription")
	if action == nil {
		t.Fatalf("expected chinese email confirm_subscription action, got %+v", actions)
	}
	if action.URL != "https://example.cn/welcome?token=43" {
		t.Fatalf("expected confirm_subscription url, got %q", action.URL)
	}
	if action.Label != "确认邮件订阅" {
		t.Fatalf("expected preserved confirm_subscription label, got %q", action.Label)
	}
}

func TestExtractActionsDoesNotClassifyChineseOrderConfirmationAsSubscription(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/orders/42">确认订单</a>`)
	if action := findAction(actions, "confirm_subscription"); action != nil {
		t.Fatalf("expected chinese order confirmation to avoid confirm_subscription classification, got %+v", actions)
	}
}

func TestExtractActionsDoesNotClassifyChineseEmailAddressConfirmationAsSubscription(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/profile/email">确认邮件地址</a>`)
	if action := findAction(actions, "confirm_subscription"); action != nil {
		t.Fatalf("expected chinese email address confirmation to avoid confirm_subscription classification, got %+v", actions)
	}
}

func TestExtractActionsClassifiesViewAttachmentLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.com/messages/123/attachment/456">View attachment</a>`)
	action := findAction(actions, "view_attachment")
	if action == nil {
		t.Fatalf("expected view_attachment action, got %+v", actions)
	}
	if action.URL != "https://example.com/messages/123/attachment/456" {
		t.Fatalf("expected view_attachment url, got %q", action.URL)
	}
	if action.Label != "View attachment" {
		t.Fatalf("expected preserved label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesDownloadAttachmentLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.com/download/attachment/456">Download file</a>`)
	action := findAction(actions, "download_attachment")
	if action == nil {
		t.Fatalf("expected download_attachment action, got %+v", actions)
	}
	if action.URL != "https://example.com/download/attachment/456" {
		t.Fatalf("expected download_attachment url, got %q", action.URL)
	}
	if action.Label != "Download file" {
		t.Fatalf("expected preserved label, got %q", action.Label)
	}
}

func TestExtractActionsDeduplicatesAttachmentActions(t *testing.T) {
	html := `<a href="https://example.com/messages/123/attachment/456">View attachment</a><a href="https://example.com/messages/123/attachment/456">View attachment</a><a href="https://example.com/download/attachment/456">Download attachment</a><a href="https://example.com/download/attachment/456">Download attachment</a>`
	actions := extractActions(schema.MessageMeta{}, html)

	viewCount := 0
	downloadCount := 0
	for _, action := range actions {
		switch action.Type {
		case "view_attachment":
			viewCount++
		case "download_attachment":
			downloadCount++
		}
	}
	if viewCount != 1 {
		t.Fatalf("expected one deduplicated view_attachment action, got %d", viewCount)
	}
	if downloadCount != 1 {
		t.Fatalf("expected one deduplicated download_attachment action, got %d", downloadCount)
	}
}

func TestExtractActionsClassifiesPayInvoiceLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://billing.example.com/invoices/123/pay">Pay invoice</a>`)
	action := findAction(actions, "pay_invoice")
	if action == nil {
		t.Fatalf("expected pay_invoice action, got %+v", actions)
	}
	if action.URL != "https://billing.example.com/invoices/123/pay" {
		t.Fatalf("expected pay_invoice url, got %q", action.URL)
	}
	if action.Label != "Pay invoice" {
		t.Fatalf("expected preserved label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesViewInvoiceLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://billing.example.com/invoices/123">View invoice</a>`)
	action := findAction(actions, "view_invoice")
	if action == nil {
		t.Fatalf("expected view_invoice action, got %+v", actions)
	}
	if action.URL != "https://billing.example.com/invoices/123" {
		t.Fatalf("expected view_invoice url, got %q", action.URL)
	}
	if action.Label != "View invoice" {
		t.Fatalf("expected preserved label, got %q", action.Label)
	}
}

func TestExtractActionsDeduplicatesInvoiceActions(t *testing.T) {
	html := `<a href="https://billing.example.com/invoices/123/pay">Pay invoice</a><a href="https://billing.example.com/invoices/123/pay">Pay invoice</a><a href="https://billing.example.com/invoices/123">View invoice</a><a href="https://billing.example.com/invoices/123">View invoice</a>`
	actions := extractActions(schema.MessageMeta{}, html)

	payCount := 0
	viewCount := 0
	for _, action := range actions {
		switch action.Type {
		case "pay_invoice":
			payCount++
		case "view_invoice":
			viewCount++
		}
	}
	if payCount != 1 {
		t.Fatalf("expected one deduplicated pay_invoice action, got %d", payCount)
	}
	if viewCount != 1 {
		t.Fatalf("expected one deduplicated view_invoice action, got %d", viewCount)
	}
}

func TestExtractActionsDoesNotClassifyGenericPayNowLinkAsInvoice(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://checkout.example.com/pay">Pay now</a>`)
	if action := findAction(actions, "pay_invoice"); action != nil {
		t.Fatalf("expected generic pay now link to avoid pay_invoice classification, got %+v", actions)
	}
}

func TestExtractActionsDoesNotClassifyGenericInvoiceSettingsLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://billing.example.com/invoice/settings">Open</a>`)
	if action := findAction(actions, "view_invoice"); action != nil {
		t.Fatalf("expected generic invoice settings link to avoid view_invoice classification, got %+v", actions)
	}
}

func TestExtractActionsClassifiesChineseViewInvoiceLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/billing/document/123">查看发票</a>`)
	action := findAction(actions, "view_invoice")
	if action == nil {
		t.Fatalf("expected chinese view_invoice action, got %+v", actions)
	}
	if action.URL != "https://example.cn/billing/document/123" {
		t.Fatalf("expected view_invoice url, got %q", action.URL)
	}
	if action.Label != "查看发票" {
		t.Fatalf("expected preserved view_invoice label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesChinesePayInvoiceLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/billing/checkout/123">支付账单</a>`)
	action := findAction(actions, "pay_invoice")
	if action == nil {
		t.Fatalf("expected chinese pay_invoice action, got %+v", actions)
	}
	if action.URL != "https://example.cn/billing/checkout/123" {
		t.Fatalf("expected pay_invoice url, got %q", action.URL)
	}
	if action.Label != "支付账单" {
		t.Fatalf("expected preserved pay_invoice label, got %q", action.Label)
	}
}

func TestExtractActionsDoesNotClassifyChineseBillingSettingsLinkAsInvoice(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/billing/settings">查看账单设置</a>`)
	if action := findAction(actions, "view_invoice"); action != nil {
		t.Fatalf("expected chinese billing settings link to avoid view_invoice classification, got %+v", actions)
	}
}

func TestExtractActionsDoesNotClassifyChinesePaymentCenterLinkAsInvoicePayment(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/payments">前往支付中心</a>`)
	if action := findAction(actions, "pay_invoice"); action != nil {
		t.Fatalf("expected chinese payment center link to avoid pay_invoice classification, got %+v", actions)
	}
}

func TestExtractActionsClassifiesChineseUnsubscribeLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/preferences/email?user=42">取消订阅</a>`)
	action := findAction(actions, "unsubscribe")
	if action == nil {
		t.Fatalf("expected chinese unsubscribe link to classify, got %+v", actions)
	}
	if action.URL != "https://example.cn/preferences/email?user=42" {
		t.Fatalf("expected unsubscribe url, got %q", action.URL)
	}
	if action.Label != "取消订阅" {
		t.Fatalf("expected preserved unsubscribe label, got %q", action.Label)
	}
}

func TestExtractActionsDoesNotClassifyGenericChinesePreferenceLinkAsUnsubscribe(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/preferences/email?user=42">查看邮件偏好设置</a>`)
	if action := findAction(actions, "unsubscribe"); action != nil {
		t.Fatalf("expected generic chinese preference link to avoid unsubscribe classification, got %+v", actions)
	}
}

func TestExtractActionsPrefersDownloadAttachmentOverViewInvoice(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://billing.example.com/invoices/123.pdf">Download invoice</a>`)
	if action := findAction(actions, "view_invoice"); action != nil {
		t.Fatalf("expected download invoice link to avoid view_invoice classification, got %+v", actions)
	}
	action := findAction(actions, "download_attachment")
	if action == nil {
		t.Fatalf("expected download invoice link to classify as download_attachment, got %+v", actions)
	}
}

func TestExtractActionsClassifiesResetPasswordLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://accounts.example.com/reset-password/token-123">Reset password</a>`)
	action := findAction(actions, "reset_password")
	if action == nil {
		t.Fatalf("expected reset_password action, got %+v", actions)
	}
	if action.URL != "https://accounts.example.com/reset-password/token-123" {
		t.Fatalf("expected reset_password url, got %q", action.URL)
	}
	if action.Label != "Reset password" {
		t.Fatalf("expected preserved label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesChineseResetPasswordLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/security/password/abc123">重置密码</a>`)
	action := findAction(actions, "reset_password")
	if action == nil {
		t.Fatalf("expected chinese reset_password action, got %+v", actions)
	}
	if action.URL != "https://example.cn/security/password/abc123" {
		t.Fatalf("expected reset_password url, got %q", action.URL)
	}
	if action.Label != "重置密码" {
		t.Fatalf("expected preserved reset_password label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesChineseChangePasswordLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/security/password/xyz789">修改密码</a>`)
	action := findAction(actions, "reset_password")
	if action == nil {
		t.Fatalf("expected chinese change password action, got %+v", actions)
	}
	if action.URL != "https://example.cn/security/password/xyz789" {
		t.Fatalf("expected reset_password url, got %q", action.URL)
	}
	if action.Label != "修改密码" {
		t.Fatalf("expected preserved reset_password label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesVerifySignInLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://accounts.example.com/verify-sign-in/session-456">Verify sign-in</a>`)
	action := findAction(actions, "verify_sign_in")
	if action == nil {
		t.Fatalf("expected verify_sign_in action, got %+v", actions)
	}
	if action.URL != "https://accounts.example.com/verify-sign-in/session-456" {
		t.Fatalf("expected verify_sign_in url, got %q", action.URL)
	}
	if action.Label != "Verify sign-in" {
		t.Fatalf("expected preserved label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesChineseVerifySignInLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/security/session/abc123">验证登录</a>`)
	action := findAction(actions, "verify_sign_in")
	if action == nil {
		t.Fatalf("expected chinese verify_sign_in action, got %+v", actions)
	}
	if action.URL != "https://example.cn/security/session/abc123" {
		t.Fatalf("expected verify_sign_in url, got %q", action.URL)
	}
	if action.Label != "验证登录" {
		t.Fatalf("expected preserved verify_sign_in label, got %q", action.Label)
	}
}

func TestExtractActionsClassifiesChineseConfirmSignInLink(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/security/session/xyz789">确认登录</a>`)
	action := findAction(actions, "verify_sign_in")
	if action == nil {
		t.Fatalf("expected chinese confirm verify_sign_in action, got %+v", actions)
	}
	if action.URL != "https://example.cn/security/session/xyz789" {
		t.Fatalf("expected verify_sign_in url, got %q", action.URL)
	}
	if action.Label != "确认登录" {
		t.Fatalf("expected preserved verify_sign_in label, got %q", action.Label)
	}
}

func TestExtractActionsDoesNotClassifyGenericSignInLinkAsVerifySignIn(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://accounts.example.com/sign-in">Sign in</a>`)
	if action := findAction(actions, "verify_sign_in"); action != nil {
		t.Fatalf("expected generic sign in link to avoid verify_sign_in classification, got %+v", actions)
	}
}

func TestExtractActionsDoesNotClassifyChineseLoginLinkAsVerifySignIn(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/login">登录账户</a>`)
	if action := findAction(actions, "verify_sign_in"); action != nil {
		t.Fatalf("expected chinese login link to avoid verify_sign_in classification, got %+v", actions)
	}
}

func TestExtractActionsDoesNotClassifyChineseAccountSecurityLinkAsVerifySignIn(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/security/settings">查看账号安全设置</a>`)
	if action := findAction(actions, "verify_sign_in"); action != nil {
		t.Fatalf("expected chinese account security link to avoid verify_sign_in classification, got %+v", actions)
	}
}

func TestExtractActionsDoesNotClassifyGenericAccountResetLinkAsResetPassword(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://accounts.example.com/reset-preferences">Reset account settings</a>`)
	if action := findAction(actions, "reset_password"); action != nil {
		t.Fatalf("expected generic reset link to avoid reset_password classification, got %+v", actions)
	}
}

func TestExtractActionsDoesNotClassifyChineseAccountPasswordLinkAsResetPassword(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/account/password">查看账户密码</a>`)
	if action := findAction(actions, "reset_password"); action != nil {
		t.Fatalf("expected chinese account password link to avoid reset_password classification, got %+v", actions)
	}
}

func TestExtractActionsDoesNotClassifyChineseSecuritySettingsLinkAsResetPassword(t *testing.T) {
	actions := extractActions(schema.MessageMeta{}, `<a href="https://example.cn/security/settings">查看安全设置</a>`)
	if action := findAction(actions, "reset_password"); action != nil {
		t.Fatalf("expected chinese security settings link to avoid reset_password classification, got %+v", actions)
	}
}

func TestParseExtractsReportAbuseActionFromHeaders(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Abuse header\r\nMessage-ID: <abuse-1@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nX-Report-Abuse-To: abuse@example.com\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nHello")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	action := findAction(got.Actions, "report_abuse")
	if action == nil {
		t.Fatalf("expected report_abuse action from headers, got %+v", got.Actions)
	}
	if action.URL != "mailto:abuse@example.com" {
		t.Fatalf("expected mailto abuse action, got %q", action.URL)
	}
}

func TestParseDeduplicatesReportAbuseAcrossHeadersAndHTML(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Abuse header\r\nMessage-ID: <abuse-2@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nX-Report-Abuse-To: abuse@example.com\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<html><body><a href=\"mailto:abuse@example.com\">Report abuse</a></body></html>")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for _, action := range got.Actions {
		if action.Type == "report_abuse" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one deduplicated report_abuse action, got %+v", got.Actions)
	}
}

func TestParseKeepsImageSrcWrapperURLUnchanged(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Image link\r\nMessage-ID: <tracked-3@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<html><body><img alt=\"Chart\" src=\"https://tracker.example.com/click?redirect=https%3A%2F%2Fcdn.example.com%2Fchart.png\" /></body></html>")

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got.Content.BodyMD, "https://tracker.example.com/click?redirect=https%3A%2F%2Fcdn.example.com%2Fchart.png") {
		t.Fatalf("expected image src to remain unchanged, got %q", got.Content.BodyMD)
	}
}

func TestParseReplacesCIDImageReferencesWithAltText(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/emails/related_inline_image.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(got.Content.BodyMD, "cid:hero-image") {
		t.Fatalf("expected cid image reference to be removed from markdown, got %q", got.Content.BodyMD)
	}
	if !strings.Contains(got.Content.BodyMD, "Security code illustration") {
		t.Fatalf("expected inline image alt text to survive cleanup, got %q", got.Content.BodyMD)
	}
}

func findAction(actions []schema.Action, actionType string) *schema.Action {
	for i := range actions {
		if actions[i].Type == actionType {
			return &actions[i]
		}
	}
	return nil
}

func assertJSONMatchesGolden(t *testing.T, got any, want []byte) {
	t.Helper()

	gotBytes, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}

	var gotJSON any
	if err := json.Unmarshal(gotBytes, &gotJSON); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}

	var wantJSON any
	if err := json.Unmarshal(want, &wantJSON); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}

	gotCanonical, err := json.Marshal(gotJSON)
	if err != nil {
		t.Fatalf("canonical got: %v", err)
	}

	wantCanonical, err := json.Marshal(wantJSON)
	if err != nil {
		t.Fatalf("canonical want: %v", err)
	}

	if string(gotCanonical) != string(wantCanonical) {
		t.Fatalf("golden mismatch\nwant: %s\ngot:  %s", string(wantCanonical), string(gotCanonical))
	}
}
