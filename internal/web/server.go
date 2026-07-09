package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"

	"github.com/nonozone/MailCli/internal/config"
	mailindex "github.com/nonozone/MailCli/internal/index"
	opstore "github.com/nonozone/MailCli/internal/operations"
	"github.com/nonozone/MailCli/pkg/composer"
	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/parser"
	"github.com/nonozone/MailCli/pkg/schema"
	"github.com/nonozone/MailCli/web"
)

const sessionCookieName = "mailcli_local_session"

type Options struct {
	ConfigPath     string
	IndexPath      string
	OperationsPath string
	Token          string
	Version        string
}

type Server struct {
	options Options
}

type apiEnvelope struct {
	OK    bool      `json:"ok"`
	Data  any       `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	NextStep string `json:"next_step,omitempty"`
}

func NewServer(options Options) (*Server, error) {
	if strings.TrimSpace(options.Token) == "" {
		token, err := GenerateToken()
		if err != nil {
			return nil, err
		}
		options.Token = token
	}
	return &Server{options: options}, nil
}

func GenerateToken() (string, error) {
	var bytes [24]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate web token: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

func IsLocalHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	switch host {
	case "", "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	default:
		parsed := net.ParseIP(strings.Trim(host, "[]"))
		return parsed != nil && parsed.IsLoopback()
	}
}

func (s *Server) Token() string {
	return s.options.Token
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /session/start", s.handleSessionStart)
	mux.HandleFunc("GET /api/session", s.withToken(s.handleSession))
	mux.HandleFunc("GET /api/accounts", s.withToken(s.handleAccounts))
	mux.HandleFunc("POST /api/sync", s.withToken(s.handleSync))
	mux.HandleFunc("GET /api/messages", s.withToken(s.handleMessages))
	mux.HandleFunc("GET /api/messages/{account}/{id}", s.withToken(s.handleMessage))
	mux.HandleFunc("GET /api/threads", s.withToken(s.handleThreads))
	mux.HandleFunc("GET /api/threads/{thread_id}", s.withToken(s.handleThread))
	mux.HandleFunc("POST /api/send/prepare", s.withToken(s.handleSendPrepare))
	mux.HandleFunc("GET /api/operations", s.withToken(s.handleOperations))
	mux.HandleFunc("GET /api/operations/{id}", s.withToken(s.handleOperation))
	mux.HandleFunc("POST /api/operations/{id}/confirm", s.withToken(s.handleOperationConfirm))
	mux.HandleFunc("GET /", s.handleStatic)
	return mux
}

func (s *Server) withToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.validToken(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "A valid local session token is required.", "")
			return
		}
		next(w, r)
	}
}

func (s *Server) validToken(r *http.Request) bool {
	expected := strings.TrimSpace(s.options.Token)
	if expected == "" {
		return false
	}
	actual := strings.TrimSpace(r.Header.Get("X-MailCLI-Token"))
	if actual == "" {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			actual = strings.TrimSpace(cookie.Value)
		}
	}
	if actual == "" {
		actual = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if actual == "" {
		return false
	}
	return s.validRawToken(actual)
}

func (s *Server) validRawToken(actual string) bool {
	expected := strings.TrimSpace(s.options.Token)
	actual = strings.TrimSpace(actual)
	if expected == "" || actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func (s *Server) handleSessionStart(w http.ResponseWriter, r *http.Request) {
	if !s.validRawToken(r.URL.Query().Get("token")) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "A valid local session token is required.", "")
		return
	}
	s.setSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if token := r.URL.Query().Get("token"); s.validRawToken(token) {
		s.setSessionCookie(w)
		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return
	}
	staticHandler().ServeHTTP(w, r)
}

func (s *Server) setSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    s.options.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{
		"version":         s.options.Version,
		"config_path":     effectiveConfigPath(s.options.ConfigPath),
		"index_path":      mailindex.NewFileStore(s.options.IndexPath).Path(),
		"operations_path": opstore.NewStore(s.options.OperationsPath).Path(),
	})
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadRaw(effectiveConfigPath(s.options.ConfigPath))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_load_failed", err.Error(), "Check the MailCLI config path and permissions.")
		return
	}
	accounts := make([]publicAccount, 0, len(cfg.Accounts))
	for _, account := range cfg.Accounts {
		accounts = append(accounts, redactAccount(account))
	}
	writeOK(w, map[string]any{
		"current_account": cfg.CurrentAccount,
		"accounts":        accounts,
	})
}

type syncRequest struct {
	Account string `json:"account,omitempty"`
	Mailbox string `json:"mailbox,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Refresh bool   `json:"refresh,omitempty"`
	Since   string `json:"since,omitempty"`
	Before  string `json:"before,omitempty"`
}

type syncResponse struct {
	Account        string `json:"account,omitempty"`
	Mailbox        string `json:"mailbox,omitempty"`
	ListedCount    int    `json:"listed_count"`
	FetchedCount   int    `json:"fetched_count"`
	IndexedCount   int    `json:"indexed_count"`
	SkippedCount   int    `json:"skipped_count"`
	RefreshedCount int    `json:"refreshed_count"`
	ErrorCount     int    `json:"error_count,omitempty"`
	IndexPath      string `json:"index_path,omitempty"`
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	var req syncRequest
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), "")
			return
		}
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), "")
				return
			}
		}
	}
	if req.Limit < 0 {
		writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be >= 0", "")
		return
	}

	cfg, err := config.Load(effectiveConfigPath(s.options.ConfigPath))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_load_failed", err.Error(), "Check the MailCLI config path and permissions.")
		return
	}
	selectedAccount, err := cfg.ResolveAccount(strings.TrimSpace(req.Account))
	if err != nil {
		writeError(w, http.StatusBadRequest, "account_not_found", err.Error(), "Choose a configured account.")
		return
	}

	result, err := s.syncAccount(r, selectedAccount, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sync_failed", err.Error(), "")
		return
	}
	writeOK(w, result)
}

func (s *Server) syncAccount(r *http.Request, selectedAccount config.AccountConfig, req syncRequest) (syncResponse, error) {
	drv, err := driver.NewFromAccount(selectedAccount)
	if err != nil {
		return syncResponse{}, err
	}
	queryMailbox := strings.TrimSpace(req.Mailbox)
	if queryMailbox == "" {
		queryMailbox = selectedAccount.Mailbox
	}
	if strings.TrimSpace(queryMailbox) == "" {
		queryMailbox = "INBOX"
	}

	items, err := drv.List(r.Context(), schema.SearchQuery{
		Mailbox: queryMailbox,
		Limit:   req.Limit,
		Since:   req.Since,
		Before:  req.Before,
	})
	if err != nil {
		return syncResponse{}, err
	}

	store := mailindex.NewFileStore(s.options.IndexPath)
	idsToFetch := make([]string, 0, len(items))
	skippedCount := 0
	if req.Refresh {
		for _, item := range items {
			idsToFetch = append(idsToFetch, item.ID)
		}
	} else {
		ids := make([]string, len(items))
		for i, item := range items {
			ids[i] = item.ID
		}
		existing, err := store.BulkHas(selectedAccount.Name, ids)
		if err != nil {
			return syncResponse{}, err
		}
		for _, item := range items {
			if existing[item.ID] {
				skippedCount++
				continue
			}
			idsToFetch = append(idsToFetch, item.ID)
		}
	}

	type rawEntry struct {
		id  string
		raw []byte
		err error
	}
	rawEntries := make([]rawEntry, 0, len(idsToFetch))
	if bf, ok := drv.(driver.BulkFetcher); ok && len(idsToFetch) > 0 {
		bulk, err := bf.FetchRawBulk(r.Context(), idsToFetch)
		if err != nil {
			return syncResponse{}, err
		}
		for _, item := range bulk {
			rawEntries = append(rawEntries, rawEntry{id: item.ID, raw: item.Raw, err: item.Err})
		}
	} else {
		for _, id := range idsToFetch {
			raw, err := drv.FetchRaw(r.Context(), id)
			rawEntries = append(rawEntries, rawEntry{id: id, raw: raw, err: err})
		}
	}

	fetchedCount := 0
	errorCount := 0
	refreshedCount := 0
	indexed := make([]mailindex.IndexedMessage, 0, len(rawEntries))
	for _, entry := range rawEntries {
		if entry.err != nil {
			errorCount++
			continue
		}
		fetchedCount++
		msg, err := parser.Parse(entry.raw)
		if err != nil {
			errorCount++
			continue
		}
		indexed = append(indexed, mailindex.IndexedMessage{
			Account: selectedAccount.Name,
			Mailbox: queryMailbox,
			ID:      entry.id,
			Message: *msg,
		})
		if req.Refresh {
			refreshedCount++
		}
	}
	if err := store.BulkUpsert(indexed); err != nil {
		return syncResponse{}, err
	}

	return syncResponse{
		Account:        selectedAccount.Name,
		Mailbox:        queryMailbox,
		ListedCount:    len(items),
		FetchedCount:   fetchedCount,
		IndexedCount:   len(indexed),
		SkippedCount:   skippedCount,
		RefreshedCount: refreshedCount,
		ErrorCount:     errorCount,
		IndexPath:      store.Path(),
	}, nil
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	store := mailindex.NewFileStore(s.options.IndexPath)
	query := mailindex.SearchQuery{
		Query:   strings.TrimSpace(r.URL.Query().Get("q")),
		Account: strings.TrimSpace(r.URL.Query().Get("account")),
		Mailbox: strings.TrimSpace(r.URL.Query().Get("mailbox")),
		Limit:   intQuery(r, "limit", 50),
	}
	results, err := store.Search(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "messages_search_failed", err.Error(), "")
		return
	}
	writeOK(w, map[string]any{"messages": results})
}

func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	account := r.PathValue("account")
	id := r.PathValue("id")
	store := mailindex.NewFileStore(s.options.IndexPath)
	items, err := store.SearchMessages(mailindex.SearchQuery{Account: account, Limit: 0})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "message_lookup_failed", err.Error(), "")
		return
	}
	for _, item := range items {
		if item.ID == id {
			writeOK(w, map[string]any{"message": item})
			return
		}
	}
	writeError(w, http.StatusNotFound, "message_not_found", "Message was not found in the local index.", "Run sync and try again.")
}

func (s *Server) handleThreads(w http.ResponseWriter, r *http.Request) {
	store := mailindex.NewFileStore(s.options.IndexPath)
	results, err := store.Threads(mailindex.ThreadQuery{
		Query:   strings.TrimSpace(r.URL.Query().Get("q")),
		Account: strings.TrimSpace(r.URL.Query().Get("account")),
		Mailbox: strings.TrimSpace(r.URL.Query().Get("mailbox")),
		Limit:   intQuery(r, "limit", 50),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "threads_failed", err.Error(), "")
		return
	}
	writeOK(w, map[string]any{"threads": results})
}

func (s *Server) handleThread(w http.ResponseWriter, r *http.Request) {
	store := mailindex.NewFileStore(s.options.IndexPath)
	results, err := store.ThreadMessages(mailindex.ThreadMessageQuery{
		ThreadID: r.PathValue("thread_id"),
		Account:  strings.TrimSpace(r.URL.Query().Get("account")),
		Mailbox:  strings.TrimSpace(r.URL.Query().Get("mailbox")),
		Limit:    intQuery(r, "limit", 100),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "thread_failed", err.Error(), "")
		return
	}
	writeOK(w, map[string]any{"messages": results})
}

func (s *Server) handleOperations(w http.ResponseWriter, r *http.Request) {
	entries, err := opstore.NewStore(s.options.OperationsPath).List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "operations_failed", err.Error(), "")
		return
	}
	writeOK(w, map[string]any{"operations": entries})
}

func (s *Server) handleSendPrepare(w http.ResponseWriter, r *http.Request) {
	var draft schema.DraftMessage
	if err := json.NewDecoder(r.Body).Decode(&draft); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), "")
		return
	}
	selectedAccount, err := s.resolveAccount(draft.Account)
	if err != nil {
		writeError(w, http.StatusBadRequest, "account_not_found", err.Error(), "Choose a configured account.")
		return
	}
	applyDefaultFromAddress(selectedAccount, &draft.From)
	if err := validateEnvelopeAddressing(draft.From, draft.To, draft.Cc, draft.Bcc); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_draft", err.Error(), "")
		return
	}
	_, messageID, err := composer.ComposeDraft(draft)
	if err != nil {
		writeError(w, http.StatusBadRequest, "compose_failed", err.Error(), "")
		return
	}
	if draft.Headers == nil {
		draft.Headers = map[string]string{}
	}
	draft.Headers["Message-ID"] = messageID

	store := opstore.NewStore(s.options.OperationsPath)
	summary := summarizeDraft(draft)
	intent := opstore.SendIntent{
		ID:        opstore.NewID("intent"),
		Operation: "send",
		Account:   selectedAccount.Name,
		MessageID: messageID,
		CreatedAt: opstore.Now(),
		Summary:   summary,
		Draft:     draft,
	}
	if err := store.SaveSendIntent(intent); err != nil {
		writeError(w, http.StatusInternalServerError, "intent_save_failed", err.Error(), "")
		return
	}
	writeOK(w, schema.OperationIntentResult{
		Status:         "prepared",
		IntentID:       intent.ID,
		Operation:      intent.Operation,
		Account:        intent.Account,
		MessageID:      intent.MessageID,
		OperationsPath: store.Path(),
		ConfirmCommand: fmt.Sprintf("mailcli send confirm --operations %s %s", store.Path(), intent.ID),
		Summary:        summary,
	})
}

func (s *Server) handleOperation(w http.ResponseWriter, r *http.Request) {
	entry, err := opstore.NewStore(s.options.OperationsPath).Find(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "operation_not_found", err.Error(), "")
		return
	}
	writeOK(w, map[string]any{"operation": entry})
}

func (s *Server) handleOperationConfirm(w http.ResponseWriter, r *http.Request) {
	store := opstore.NewStore(s.options.OperationsPath)
	intent, err := store.LoadSendIntent(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "intent_not_found", err.Error(), "")
		return
	}
	if intent.Operation != "send" {
		writeError(w, http.StatusBadRequest, "unsupported_operation", "Only send intents can be confirmed through the Web API in this release.", "")
		return
	}
	selectedAccount, err := s.resolveAccount(intent.Account)
	if err != nil {
		writeError(w, http.StatusBadRequest, "account_not_found", err.Error(), "Choose a configured account.")
		return
	}
	if _, err := store.SentEntryForOperation(intent.ID, "send"); err == nil {
		writeError(w, http.StatusConflict, "intent_already_sent", "This intent has already been sent.", "")
		return
	} else if !errors.Is(err, opstore.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "operation_lookup_failed", err.Error(), "")
		return
	}
	if err := validateEnvelopeAddressing(intent.Draft.From, intent.Draft.To, intent.Draft.Cc, intent.Draft.Bcc); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_draft", err.Error(), "")
		return
	}
	mime, _, err := composer.ComposeDraft(intent.Draft)
	if err != nil {
		writeError(w, http.StatusBadRequest, "compose_failed", err.Error(), "")
		return
	}
	drv, err := driver.NewFromAccount(selectedAccount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "driver_init_failed", err.Error(), "")
		return
	}
	if err := drv.SendRaw(r.Context(), mime); err != nil {
		writeError(w, http.StatusInternalServerError, "send_failed", err.Error(), "")
		return
	}
	operationID := opstore.NewID("op")
	summary := intent.Summary
	if err := store.Append(schema.OperationLogEntry{
		ID:        operationID,
		IntentID:  intent.ID,
		Operation: "send",
		Status:    "sent",
		Account:   selectedAccount.Name,
		MessageID: intent.MessageID,
		CreatedAt: opstore.Now(),
		Summary:   &summary,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "operation_log_failed", err.Error(), "")
		return
	}
	writeOK(w, schema.SendResult{
		OK:          true,
		MessageID:   intent.MessageID,
		Provider:    selectedAccount.Driver,
		Account:     selectedAccount.Name,
		IntentID:    intent.ID,
		OperationID: operationID,
	})
}

func (s *Server) resolveAccount(name string) (config.AccountConfig, error) {
	cfg, err := config.Load(effectiveConfigPath(s.options.ConfigPath))
	if err != nil {
		return config.AccountConfig{}, err
	}
	return cfg.ResolveAccount(strings.TrimSpace(name))
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, apiEnvelope{OK: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, code, message, nextStep string) {
	writeJSON(w, status, apiEnvelope{
		OK: false,
		Error: &apiError{
			Code:     code,
			Message:  message,
			NextStep: nextStep,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload apiEnvelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func staticHandler() http.Handler {
	sub, err := fs.Sub(webassets.Files, ".")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(sub))
}

func effectiveConfigPath(path string) string {
	if strings.TrimSpace(path) != "" {
		return strings.TrimSpace(path)
	}
	return config.DefaultPath()
}

func intQuery(r *http.Request, name string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback
	}
	var value int
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil || value < 0 {
		return fallback
	}
	return value
}

type publicAccount struct {
	Name            string `json:"name"`
	Provider        string `json:"provider,omitempty"`
	Driver          string `json:"driver"`
	AuthMethod      string `json:"auth_method,omitempty"`
	Path            string `json:"path,omitempty"`
	Host            string `json:"host,omitempty"`
	Port            int    `json:"port,omitempty"`
	Username        string `json:"username,omitempty"`
	TLS             bool   `json:"tls,omitempty"`
	Mailbox         string `json:"mailbox,omitempty"`
	SMTPHost        string `json:"smtp_host,omitempty"`
	SMTPPort        int    `json:"smtp_port,omitempty"`
	SMTPUsername    string `json:"smtp_username,omitempty"`
	SMTPTLS         bool   `json:"smtp_tls,omitempty"`
	HasPassword     bool   `json:"has_password"`
	HasSMTPPassword bool   `json:"has_smtp_password"`
}

func redactAccount(account config.AccountConfig) publicAccount {
	return publicAccount{
		Name:            account.Name,
		Provider:        account.Provider,
		Driver:          account.Driver,
		AuthMethod:      account.AuthMethod,
		Path:            account.Path,
		Host:            account.Host,
		Port:            account.Port,
		Username:        account.Username,
		TLS:             account.TLS,
		Mailbox:         account.Mailbox,
		SMTPHost:        account.SMTPHost,
		SMTPPort:        account.SMTPPort,
		SMTPUsername:    account.SMTPUsername,
		SMTPTLS:         account.SMTPTLS,
		HasPassword:     strings.TrimSpace(account.Password) != "",
		HasSMTPPassword: strings.TrimSpace(account.SMTPPassword) != "",
	}
}

func applyDefaultFromAddress(account config.AccountConfig, current **schema.Address) {
	if current == nil || *current != nil {
		return
	}
	address := strings.TrimSpace(account.SMTPUsername)
	if address == "" {
		address = strings.TrimSpace(account.Username)
	}
	if address == "" {
		return
	}
	*current = &schema.Address{Address: address}
}

func validateEnvelopeAddressing(from *schema.Address, to, cc, bcc []schema.Address) error {
	if from == nil || strings.TrimSpace(from.Address) == "" {
		return fmt.Errorf("from address is required")
	}
	if !hasAddress(to) && !hasAddress(cc) && !hasAddress(bcc) {
		return fmt.Errorf("at least one recipient is required")
	}
	return nil
}

func hasAddress(addresses []schema.Address) bool {
	for _, address := range addresses {
		if strings.TrimSpace(address.Address) != "" {
			return true
		}
	}
	return false
}

func summarizeDraft(draft schema.DraftMessage) schema.OperationSummary {
	return schema.OperationSummary{
		Subject:         draft.Subject,
		From:            draft.From,
		To:              append([]schema.Address(nil), draft.To...),
		Cc:              append([]schema.Address(nil), draft.Cc...),
		BccCount:        len(draft.Bcc),
		AttachmentCount: len(draft.Attachments),
	}
}
