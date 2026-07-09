package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	opstore "github.com/nonozone/MailCli/internal/operations"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestServerRejectsAPICallsWithoutToken(t *testing.T) {
	srv, err := NewServer(Options{Token: "secret"})
	if err != nil {
		t.Fatalf("expected server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status, got %d: %s", rec.Code, rec.Body.String())
	}
	var body apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON envelope: %v", err)
	}
	if body.OK || body.Error == nil || body.Error.Code != "unauthorized" {
		t.Fatalf("expected unauthorized envelope, got %#v", body)
	}
}

func TestServerAcceptsTokenFromQuery(t *testing.T) {
	srv, err := NewServer(Options{Token: "secret", Version: "test-version"})
	if err != nil {
		t.Fatalf("expected server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/session?token=secret", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok status, got %d: %s", rec.Code, rec.Body.String())
	}
	var body apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON envelope: %v", err)
	}
	if !body.OK {
		t.Fatalf("expected ok envelope, got %#v", body)
	}
	data, ok := body.Data.(map[string]any)
	if !ok || data["version"] != "test-version" {
		t.Fatalf("expected session version, got %#v", body.Data)
	}
}

func TestServerRendersStaticIndex(t *testing.T) {
	srv, err := NewServer(Options{Token: "secret"})
	if err != nil {
		t.Fatalf("expected server: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok status, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Mail Center") {
		t.Fatalf("expected Mail Center static shell, got %s", rec.Body.String())
	}
}

func TestAccountsRedactSecrets(t *testing.T) {
	configPath := writeTestConfig(t, `
current_account: work
accounts:
  - name: work
    provider: gmail
    driver: imap
    host: imap.gmail.com
    port: 993
    username: user@example.com
    password: super-secret
    smtp_host: smtp.gmail.com
    smtp_port: 465
    smtp_password: smtp-secret
`)
	srv, err := NewServer(Options{Token: "secret", ConfigPath: configPath})
	if err != nil {
		t.Fatalf("expected server: %v", err)
	}

	rec := request(t, srv, http.MethodGet, "/api/accounts?token=secret", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok status, got %d: %s", rec.Code, rec.Body.String())
	}
	text := rec.Body.String()
	if strings.Contains(text, "super-secret") || strings.Contains(text, "smtp-secret") {
		t.Fatalf("expected secrets to be redacted, got %s", text)
	}
	if !strings.Contains(text, `"has_password":true`) || !strings.Contains(text, `"has_smtp_password":true`) {
		t.Fatalf("expected secret presence flags, got %s", text)
	}
}

func TestSyncMessagesThreadsAndOperations(t *testing.T) {
	root := repoRoot(t)
	configPath := writeTestConfig(t, `
current_account: fixtures
accounts:
  - name: fixtures
    driver: dir
    path: `+filepath.ToSlash(filepath.Join(root, "testdata", "emails"))+`
    mailbox: INBOX
`)
	indexPath := filepath.Join(t.TempDir(), "index.db")
	operationsPath := filepath.Join(t.TempDir(), "operations.jsonl")
	store := opstore.NewStore(operationsPath)
	if err := store.Append(schema.OperationLogEntry{
		ID:        "op_test",
		IntentID:  "intent_test",
		Operation: "send",
		Status:    "prepared",
		Account:   "fixtures",
		CreatedAt: "2026-07-09T00:00:00Z",
	}); err != nil {
		t.Fatalf("append operation: %v", err)
	}

	srv, err := NewServer(Options{
		Token:          "secret",
		ConfigPath:     configPath,
		IndexPath:      indexPath,
		OperationsPath: operationsPath,
	})
	if err != nil {
		t.Fatalf("expected server: %v", err)
	}

	syncRec := request(t, srv, http.MethodPost, "/api/sync?token=secret", `{"account":"fixtures","limit":0}`)
	if syncRec.Code != http.StatusOK {
		t.Fatalf("expected sync ok, got %d: %s", syncRec.Code, syncRec.Body.String())
	}
	if !strings.Contains(syncRec.Body.String(), `"indexed_count"`) {
		t.Fatalf("expected sync counts, got %s", syncRec.Body.String())
	}

	messagesRec := request(t, srv, http.MethodGet, "/api/messages?token=secret&q=invoice", "")
	if messagesRec.Code != http.StatusOK {
		t.Fatalf("expected messages ok, got %d: %s", messagesRec.Code, messagesRec.Body.String())
	}
	if !strings.Contains(messagesRec.Body.String(), "invoice.eml") {
		t.Fatalf("expected indexed invoice message, got %s", messagesRec.Body.String())
	}

	messagePath := "/api/messages/fixtures/" + url.PathEscape("invoice.eml") + "?token=secret"
	messageRec := request(t, srv, http.MethodGet, messagePath, "")
	if messageRec.Code != http.StatusOK {
		t.Fatalf("expected message ok, got %d: %s", messageRec.Code, messageRec.Body.String())
	}
	if !strings.Contains(messageRec.Body.String(), "Your invoice is ready") {
		t.Fatalf("expected full message content, got %s", messageRec.Body.String())
	}

	threadsRec := request(t, srv, http.MethodGet, "/api/threads?token=secret&q=invoice", "")
	if threadsRec.Code != http.StatusOK {
		t.Fatalf("expected threads ok, got %d: %s", threadsRec.Code, threadsRec.Body.String())
	}
	if !strings.Contains(threadsRec.Body.String(), `"thread_id"`) {
		t.Fatalf("expected thread summaries, got %s", threadsRec.Body.String())
	}

	operationsRec := request(t, srv, http.MethodGet, "/api/operations?token=secret", "")
	if operationsRec.Code != http.StatusOK {
		t.Fatalf("expected operations ok, got %d: %s", operationsRec.Code, operationsRec.Body.String())
	}
	if !strings.Contains(operationsRec.Body.String(), "intent_test") {
		t.Fatalf("expected operation log entry, got %s", operationsRec.Body.String())
	}

	operationRec := request(t, srv, http.MethodGet, "/api/operations/intent_test?token=secret", "")
	if operationRec.Code != http.StatusOK {
		t.Fatalf("expected operation detail ok, got %d: %s", operationRec.Code, operationRec.Body.String())
	}
	if !strings.Contains(operationRec.Body.String(), `"id":"op_test"`) {
		t.Fatalf("expected operation detail, got %s", operationRec.Body.String())
	}
}

func TestSendPrepareAndConfirmUseOperationLog(t *testing.T) {
	configPath := writeTestConfig(t, `
current_account: stub
accounts:
  - name: stub
    driver: stub
    username: sender@example.com
`)
	operationsPath := filepath.Join(t.TempDir(), "operations.jsonl")
	srv, err := NewServer(Options{
		Token:          "secret",
		ConfigPath:     configPath,
		OperationsPath: operationsPath,
	})
	if err != nil {
		t.Fatalf("expected server: %v", err)
	}

	prepareRec := request(t, srv, http.MethodPost, "/api/send/prepare?token=secret", `{
		"account": "stub",
		"to": [{ "address": "recipient@example.com" }],
		"subject": "Hello from web",
		"body_text": "Prepared from local web panel."
	}`)
	if prepareRec.Code != http.StatusOK {
		t.Fatalf("expected prepare ok, got %d: %s", prepareRec.Code, prepareRec.Body.String())
	}
	var prepareBody apiEnvelope
	if err := json.Unmarshal(prepareRec.Body.Bytes(), &prepareBody); err != nil {
		t.Fatalf("decode prepare: %v", err)
	}
	data := prepareBody.Data.(map[string]any)
	intentID := data["intent_id"].(string)
	if intentID == "" {
		t.Fatalf("expected intent id, got %#v", data)
	}

	confirmRec := request(t, srv, http.MethodPost, "/api/operations/"+intentID+"/confirm?token=secret", "")
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("expected confirm ok, got %d: %s", confirmRec.Code, confirmRec.Body.String())
	}
	if !strings.Contains(confirmRec.Body.String(), `"ok":true`) {
		t.Fatalf("expected send result, got %s", confirmRec.Body.String())
	}

	operationsRec := request(t, srv, http.MethodGet, "/api/operations?token=secret", "")
	if !strings.Contains(operationsRec.Body.String(), `"status":"sent"`) {
		t.Fatalf("expected sent operation log entry, got %s", operationsRec.Body.String())
	}
}

func request(t *testing.T, srv *Server, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
