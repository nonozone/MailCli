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
