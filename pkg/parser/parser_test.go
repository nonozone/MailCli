package parser

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/yourname/mailcli/pkg/schema"
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
