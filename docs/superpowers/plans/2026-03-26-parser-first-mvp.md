# Parser-First MVP Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first usable `MailCLI` MVP that parses `.eml` input from file or stdin into normalized JSON and Markdown for AI workflows.

**Architecture:** Start with a parser-first vertical slice. Define a stable schema, build a MIME/HTML parser with golden tests, then expose it through a minimal Cobra CLI. Add only the provider/config interfaces needed to preserve future architecture, without implementing IMAP list/send yet.

**Tech Stack:** Go, Cobra, `enmime`, `html-to-markdown`, `goquery`, `yaml.v3`, standard library testing

---

## Assumptions And Preconditions

- The current workspace is not yet a Git repository.
- The current workspace does not yet have a Go module.
- This plan uses parser-first scope only. It does not include IMAP list/send implementation.
- If the final GitHub module path is not known yet, initialize with `github.com/yourname/mailcli` and replace it later before public release.

## Target File Layout

### Files to create

- `go.mod`
- `go.sum`
- `cmd/mailcli/main.go`
- `cmd/root.go`
- `cmd/parse.go`
- `cmd/parse_test.go`
- `pkg/schema/message.go`
- `pkg/schema/query.go`
- `pkg/parser/parser.go`
- `pkg/parser/mime.go`
- `pkg/parser/html.go`
- `pkg/parser/actions.go`
- `pkg/parser/charset.go`
- `pkg/parser/token.go`
- `pkg/parser/parser_test.go`
- `pkg/driver/driver.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `testdata/emails/mercury.eml`
- `testdata/emails/bounce.eml`
- `testdata/emails/plaintext.eml`
- `testdata/golden/mercury.json`
- `testdata/golden/bounce.json`
- `testdata/golden/plaintext.json`
- `README.md`
- `README.zh-CN.md`

### Files to modify

- None. Repository starts empty.

## Chunk 1: Repository Bootstrap And Stable Schema

### Task 1: Bootstrap the repository

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `cmd/mailcli/main.go`

- [ ] **Step 1: Initialize Git**

Run:

```bash
git init
```

Expected: repository is initialized and `git status` works.

- [ ] **Step 2: Initialize the Go module**

Run:

```bash
go mod init github.com/yourname/mailcli
```

Expected: `go.mod` exists.

- [ ] **Step 3: Write the failing smoke test for CLI startup**

Add `cmd/parse_test.go` with a minimal executable smoke test that expects the root command to exist and return usage on no args.

```go
func TestRootCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected root command to execute without error, got %v", err)
	}
}
```

- [ ] **Step 4: Run the test to verify it fails**

Run:

```bash
go test ./cmd -run TestRootCommandExists -v
```

Expected: FAIL because `NewRootCmd` does not exist yet.

- [ ] **Step 5: Write the minimal CLI bootstrap**

Create `cmd/mailcli/main.go` and `cmd/root.go` with a `NewRootCmd()` constructor and a no-op root command that prints help.

```go
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailcli",
		Short: "AI-native email normalization toolkit",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	return cmd
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run:

```bash
go test ./cmd -run TestRootCommandExists -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add go.mod go.sum cmd/mailcli/main.go cmd/root.go cmd/parse_test.go
git commit -m "chore: bootstrap go module and root cli"
```

Expected: initial bootstrap commit created.

### Task 2: Define the stable message schema

**Files:**
- Create: `pkg/schema/message.go`
- Create: `pkg/schema/query.go`
- Test: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write the failing schema-shape test**

Create `pkg/parser/parser_test.go` with a schema-level test that unmarshals parser output into `schema.StandardMessage` and asserts required top-level fields exist.

```go
func TestStandardMessageShape(t *testing.T) {
	var msg schema.StandardMessage
	raw := []byte("placeholder")

	_, err := Parse(raw)
	if err == nil {
		t.Fatalf("expected parser to fail until implemented")
	}

	_ = msg
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test ./pkg/parser -run TestStandardMessageShape -v
```

Expected: FAIL because `Parse` and `schema.StandardMessage` do not exist yet.

- [ ] **Step 3: Write the minimal schema**

Create `pkg/schema/message.go` with:

- `StandardMessage`
- `MessageMeta`
- `Address`
- `Content`
- `Action`
- `ErrorContext`
- `TokenUsage`

Create `pkg/schema/query.go` with placeholder query types for future drivers:

- `SearchQuery`
- `MessageMetaSummary`

Required fields for `StandardMessage`:

```go
type StandardMessage struct {
	ID           string        `json:"id" yaml:"id"`
	Meta         MessageMeta   `json:"meta" yaml:"meta"`
	Content      Content       `json:"content" yaml:"content"`
	Actions      []Action      `json:"actions,omitempty" yaml:"actions,omitempty"`
	ErrorContext *ErrorContext `json:"error_context,omitempty" yaml:"error_context,omitempty"`
	Labels       []string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	TokenUsage   *TokenUsage   `json:"token_usage,omitempty" yaml:"token_usage,omitempty"`
}
```

- [ ] **Step 4: Add a parser stub**

Create `pkg/parser/parser.go` with a placeholder:

```go
var ErrNotImplemented = errors.New("parser not implemented")

func Parse(raw []byte) (*schema.StandardMessage, error) {
	return nil, ErrNotImplemented
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run:

```bash
go test ./pkg/parser -run TestStandardMessageShape -v
```

Expected: PASS because the test only checks that the types and stub exist.

- [ ] **Step 6: Commit**

Run:

```bash
git add pkg/schema/message.go pkg/schema/query.go pkg/parser/parser.go pkg/parser/parser_test.go
git commit -m "feat: define standard message schema"
```

## Chunk 2: Parser Engine With Golden Tests

### Task 3: Add representative fixtures and first failing parser test

**Files:**
- Create: `testdata/emails/mercury.eml`
- Create: `testdata/emails/bounce.eml`
- Create: `testdata/emails/plaintext.eml`
- Create: `testdata/golden/mercury.json`
- Create: `testdata/golden/bounce.json`
- Create: `testdata/golden/plaintext.json`
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Add sanitized fixture emails**

Place representative sample emails into:

- `testdata/emails/mercury.eml`
- `testdata/emails/bounce.eml`
- `testdata/emails/plaintext.eml`

Use real structure, but remove secrets and unstable identifiers if needed.

- [ ] **Step 2: Write the first failing golden test**

Extend `pkg/parser/parser_test.go`:

```go
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
```

- [ ] **Step 3: Run the test to verify it fails**

Run:

```bash
go test ./pkg/parser -run TestParseMercuryEmail -v
```

Expected: FAIL because parser behavior is not implemented.

- [ ] **Step 4: Create placeholder golden files**

Add minimal JSON placeholders for all three golden files so tests can load them.

- [ ] **Step 5: Commit**

Run:

```bash
git add testdata/emails testdata/golden pkg/parser/parser_test.go
git commit -m "test: add parser fixtures and golden placeholders"
```

### Task 4: Parse MIME structure and choose best content

**Files:**
- Create: `pkg/parser/mime.go`
- Modify: `pkg/parser/parser.go`
- Modify: `pkg/parser/parser_test.go`
- Test: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write the failing content-selection test**

Add a test that asserts HTML is preferred over plain text when both exist.

```go
func TestParsePrefersHTMLOverPlainText(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/emails/mercury.eml")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got.Content.BodyMD, "Mercury Insights") {
		t.Fatalf("expected markdown converted from HTML body")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test ./pkg/parser -run TestParsePrefersHTMLOverPlainText -v
```

Expected: FAIL because multipart traversal and HTML preference do not exist yet.

- [ ] **Step 3: Write minimal MIME parsing**

Use `enmime` to:

- read raw MIME
- extract headers
- detect multipart structure
- prefer HTML part when present
- fall back to text/plain

Implement these helpers in `pkg/parser/mime.go`:

- `readEnvelope(raw []byte) (*enmime.Envelope, error)`
- `selectBody(env *enmime.Envelope) (format string, body string)`
- `populateMeta(env *enmime.Envelope) schema.MessageMeta`

- [ ] **Step 4: Wire parser output**

Update `pkg/parser/parser.go` so `Parse` returns:

- message ID
- normalized headers
- content format
- initial snippet

- [ ] **Step 5: Run the targeted tests**

Run:

```bash
go test ./pkg/parser -run 'TestStandardMessageShape|TestParsePrefersHTMLOverPlainText|TestParseMercuryEmail' -v
```

Expected: the HTML preference test passes, while golden comparison may still fail until HTML cleanup is added.

- [ ] **Step 6: Commit**

Run:

```bash
git add pkg/parser/mime.go pkg/parser/parser.go pkg/parser/parser_test.go
git commit -m "feat: parse mime structure and select best body"
```

### Task 5: Clean HTML and convert it to Markdown

**Files:**
- Create: `pkg/parser/html.go`
- Modify: `pkg/parser/parser.go`
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: Write the failing HTML-to-Markdown test**

Add a test that verifies noisy tags are removed and meaningful links remain.

```go
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
	if !strings.Contains(got.Content.BodyMD, "https://") {
		t.Fatalf("expected links to survive markdown conversion")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test ./pkg/parser -run TestParseCleansHTMLAndConvertsToMarkdown -v
```

Expected: FAIL because HTML cleanup and Markdown conversion do not exist yet.

- [ ] **Step 3: Implement HTML cleanup**

In `pkg/parser/html.go`:

- parse HTML with `goquery`
- remove `style`, `script`, `svg`, and tracking-only nodes
- keep semantic structure such as links, tables, headings, paragraphs, and images

Add:

- `cleanHTML(input string) (string, error)`
- `htmlToMarkdown(input string) (string, error)`

- [ ] **Step 4: Wire cleaned Markdown into parser output**

If HTML exists:

- clean HTML
- convert to Markdown
- set `Content.Format = "markdown"`
- set `Content.BodyMD`

If only text exists:

- keep it as plain text converted into markdown-safe text

- [ ] **Step 5: Re-record golden outputs**

Update:

- `testdata/golden/mercury.json`
- `testdata/golden/plaintext.json`

with real parser output after cleanup.

- [ ] **Step 6: Run parser tests**

Run:

```bash
go test ./pkg/parser -run 'TestParseMercuryEmail|TestParseCleansHTMLAndConvertsToMarkdown' -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add pkg/parser/html.go pkg/parser/parser.go pkg/parser/parser_test.go testdata/golden/mercury.json testdata/golden/plaintext.json
git commit -m "feat: clean html and convert email content to markdown"
```

### Task 6: Extract actions, bounce context, charset normalization, and token estimates

**Files:**
- Create: `pkg/parser/actions.go`
- Create: `pkg/parser/charset.go`
- Create: `pkg/parser/token.go`
- Modify: `pkg/parser/parser.go`
- Modify: `pkg/parser/parser_test.go`
- Modify: `testdata/golden/bounce.json`

- [ ] **Step 1: Write the failing unsubscribe action test**

```go
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
```

- [ ] **Step 2: Write the failing bounce-context test**

```go
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
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./pkg/parser -run 'TestParseExtractsUnsubscribeAction|TestParseBounceEmailExtractsErrorContext' -v
```

Expected: FAIL because action extraction and bounce extraction do not exist yet.

- [ ] **Step 4: Implement action extraction**

In `pkg/parser/actions.go`:

- parse `List-Unsubscribe`
- extract unsubscribe URLs from headers and HTML links
- define stable action types such as `unsubscribe`

Add:

- `extractActions(meta schema.MessageMeta, html string) []schema.Action`

- [ ] **Step 5: Implement bounce extraction**

In `pkg/parser/parser.go` or a dedicated helper:

- detect DSN-style messages
- extract failed recipient
- extract status code
- extract diagnostic code
- map category labels such as `bounce` and `error`

- [ ] **Step 6: Implement charset normalization and token estimate**

In `pkg/parser/charset.go`:

- normalize common charsets to UTF-8
- handle quoted-printable and base64 through the MIME library plus charset conversion

In `pkg/parser/token.go`:

- implement a simple token estimate based on rune/word count heuristic
- populate `TokenUsage.EstimatedInputTokens`

- [ ] **Step 7: Re-record bounce golden output**

Update `testdata/golden/bounce.json` with real expected normalized output.

- [ ] **Step 8: Run full parser test suite**

Run:

```bash
go test ./pkg/parser -v
```

Expected: PASS.

- [ ] **Step 9: Commit**

Run:

```bash
git add pkg/parser/actions.go pkg/parser/charset.go pkg/parser/token.go pkg/parser/parser.go pkg/parser/parser_test.go testdata/golden/bounce.json
git commit -m "feat: extract actions bounce context and token estimates"
```

## Chunk 3: Minimal CLI Surface And Future-Proof Scaffolding

### Task 7: Expose parser through `mailcli parse`

**Files:**
- Create: `cmd/parse.go`
- Modify: `cmd/root.go`
- Modify: `cmd/parse_test.go`

- [ ] **Step 1: Write the failing file-input CLI test**

```go
func TestParseCommandReadsFile(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"parse", "../../testdata/emails/plaintext.eml"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected parse command to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "\"content\"") {
		t.Fatalf("expected json output")
	}
}
```

- [ ] **Step 2: Write the failing stdin CLI test**

```go
func TestParseCommandReadsStdin(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(loadFixture(t, "../../testdata/emails/plaintext.eml")))
	cmd.SetArgs([]string{"parse", "-"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected stdin parse to succeed: %v", err)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./cmd -run 'TestParseCommandReadsFile|TestParseCommandReadsStdin' -v
```

Expected: FAIL because `parse` command does not exist yet.

- [ ] **Step 4: Implement the parse command**

Create `cmd/parse.go`:

- accept a file path or `-` for stdin
- read input bytes
- call `parser.Parse`
- write JSON to stdout

Required command shape:

```go
Use:   "parse [file|-]",
Short: "Parse an email into normalized JSON",
Args:  cobra.ExactArgs(1),
```

- [ ] **Step 5: Wire `parse` into the root command**

Register the parse command from `cmd/root.go`.

- [ ] **Step 6: Run CLI tests**

Run:

```bash
go test ./cmd -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add cmd/root.go cmd/parse.go cmd/parse_test.go
git commit -m "feat: add parse command for file and stdin input"
```

### Task 8: Add minimal provider and config scaffolding

**Files:**
- Create: `pkg/driver/driver.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing config round-trip test**

```go
func TestConfigRoundTrip(t *testing.T) {
	cfg := Config{
		CurrentAccount: "local",
		Accounts: []AccountConfig{
			{Name: "local", Driver: "imap"},
		},
	}

	data, err := Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	if got.CurrentAccount != "local" {
		t.Fatalf("expected round-trip config")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test ./internal/config -run TestConfigRoundTrip -v
```

Expected: FAIL because config types and marshal helpers do not exist yet.

- [ ] **Step 3: Implement minimal config support**

In `internal/config/config.go` add:

- `Config`
- `AccountConfig`
- `Marshal`
- `Unmarshal`

Use `yaml.v3`.

- [ ] **Step 4: Add driver interface**

In `pkg/driver/driver.go` add:

```go
type Driver interface {
	List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error)
	FetchRaw(ctx context.Context, id string) ([]byte, error)
	SendRaw(ctx context.Context, raw []byte) error
}
```

This is scaffold only. Do not implement IMAP behavior in this plan.

- [ ] **Step 5: Run config tests**

Run:

```bash
go test ./internal/config -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/config/config.go internal/config/config_test.go pkg/driver/driver.go
git commit -m "feat: add config and driver scaffolding"
```

### Task 9: Add initial bilingual project entry docs

**Files:**
- Create: `README.md`
- Create: `README.zh-CN.md`

- [ ] **Step 1: Write the failing documentation check**

Add a lightweight check in CI later. For now, define the requirement manually:

- each README must link to the other language
- each README must describe parser-first MVP scope

- [ ] **Step 2: Create `README.md`**

Include:

- project positioning
- non-goals
- parser-first MVP usage
- architecture summary
- links to Chinese docs

- [ ] **Step 3: Create `README.zh-CN.md`**

Include:

- 中文项目定位
- 非目标
- parser-first MVP 用法
- 架构概要
- 指向英文文档的链接

- [ ] **Step 4: Manually verify cross-links**

Check that both files contain top-of-file language links.

- [ ] **Step 5: Commit**

Run:

```bash
git add README.md README.zh-CN.md
git commit -m "docs: add bilingual readme entrypoints"
```

## Final Verification

- [ ] Run parser tests:

```bash
go test ./pkg/parser -v
```

- [ ] Run CLI tests:

```bash
go test ./cmd -v
```

- [ ] Run config tests:

```bash
go test ./internal/config -v
```

- [ ] Run the full suite:

```bash
go test ./...
```

- [ ] Manually verify parser command on file input:

```bash
go run ./cmd/mailcli parse ./testdata/emails/plaintext.eml
```

Expected: normalized JSON printed to stdout.

- [ ] Manually verify parser command on stdin:

```bash
cat ./testdata/emails/plaintext.eml | go run ./cmd/mailcli parse -
```

Expected: normalized JSON printed to stdout.

## Out Of Scope For This Plan

- IMAP list command
- SMTP send command
- provider-specific implementations
- account switching behavior beyond config scaffold
- plugin loading system
- documentation site generation
- cloud integrations

## Execution Notes

- Keep parser behavior stable once golden files are introduced.
- Prefer adding new fixtures over overfitting parser logic to one sample.
- If a golden file changes, review whether the change reflects a real improvement or a schema regression.
- Do not add provider-specific fields into the core schema during this MVP.
