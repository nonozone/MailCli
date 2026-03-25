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
	raw, err := os.ReadFile("../../testdata/emails/mercury.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	want, err := os.ReadFile("../../testdata/golden/mercury.json")
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
