package driver

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	netmail "net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/schema"
)

type imapDriver struct {
	host        string
	port        int
	username    string
	password    string
	useTLS      bool
	mailbox     string
	smtp        smtpConfig
	connectFunc func() (imapSession, error)
}

type smtpConfig struct {
	host     string
	port     int
	username string
	password string
	useTLS   bool
}

// imapSession covers the full set of IMAP operations used by this driver.
type imapSession interface {
	Select(name string, readOnly bool) (*imap.MailboxStatus, error)
	Search(criteria *imap.SearchCriteria) ([]uint32, error)
	Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	UidStore(seqset *imap.SeqSet, item imap.StoreItem, value interface{}, ch chan *imap.Message) error
	UidCopy(seqset *imap.SeqSet, dest string) error
	Expunge(ch chan uint32) error
	Logout() error
}

type fetchTargetKind int

const (
	fetchTargetSequence fetchTargetKind = iota + 1
	fetchTargetUID
	fetchTargetMessageID
)

type fetchTarget struct {
	kind        fetchTargetKind
	value       uint32
	headerValue string
}

var smtpSendFunc = sendSMTP

func newIMAPDriver(account config.AccountConfig) (Driver, error) {
	if stringsTrim(account.Host) == "" || account.Port == 0 || stringsTrim(account.Username) == "" || stringsTrim(account.Password) == "" {
		return nil, fmt.Errorf("%w: imap account requires host, port, username, and password", ErrDriverConfigInvalid)
	}

	mailbox := account.Mailbox
	if stringsTrim(mailbox) == "" {
		mailbox = "INBOX"
	}

	driver := &imapDriver{
		host:     account.Host,
		port:     account.Port,
		username: account.Username,
		password: account.Password,
		useTLS:   account.TLS || account.Port == 993,
		mailbox:  mailbox,
		smtp: smtpConfig{
			host:     account.SMTPHost,
			port:     account.SMTPPort,
			username: firstNonEmptyTrim(account.SMTPUsername, account.Username),
			password: firstNonEmptyTrim(account.SMTPPassword, account.Password),
			useTLS:   account.SMTPTLS || account.SMTPPort == 465,
		},
	}
	driver.connectFunc = driver.connect
	return driver, nil
}

// ─── Driver interface ────────────────────────────────────────────────────────

func (d *imapDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	client, err := d.connectFunc()
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	mailboxName := d.mailbox
	if stringsTrim(query.Mailbox) != "" {
		mailboxName = query.Mailbox
	}

	mbox, err := client.Select(mailboxName, true)
	if err != nil {
		return nil, err
	}

	if mbox.Messages == 0 {
		return []schema.MessageMetaSummary{}, nil
	}

	// Use IMAP SEARCH when date filters are present.
	hasSince := stringsTrim(query.Since) != ""
	hasBefore := stringsTrim(query.Before) != ""
	if hasSince || hasBefore {
		return d.listWithDateSearch(ctx, client, query)
	}

	limit := query.Limit
	if limit < 0 {
		limit = 10
	}
	from := uint32(1)
	to := mbox.Messages
	if limit > 0 && mbox.Messages > uint32(limit-1) {
		from = mbox.Messages - uint32(limit-1)
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)
	items := []imap.FetchItem{imap.FetchEnvelope}
	bufferSize := 1
	if limit > 0 {
		bufferSize = limit
	}
	messages := make(chan *imap.Message, bufferSize)
	done := make(chan error, 1)

	go func() {
		done <- client.Fetch(seqset, items, messages)
	}()

	var results []schema.MessageMetaSummary
	for msg := range messages {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		summary := schema.MessageMetaSummary{
			From:    safeEnvelopeFrom(msg),
			Subject: safeEnvelopeSubject(msg),
			Date:    safeEnvelopeDate(msg),
		}
		if msg.Envelope != nil && stringsTrim(msg.Envelope.MessageId) != "" {
			summary.ID = msg.Envelope.MessageId
		} else {
			summary.ID = fmt.Sprintf("%d", msg.SeqNum)
		}
		results = append(results, summary)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	return results, nil
}

// listWithDateSearch uses IMAP SEARCH criteria for server-side date filtering.
func (d *imapDriver) listWithDateSearch(ctx context.Context, client imapSession, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	criteria := imap.NewSearchCriteria()
	if since := stringsTrim(query.Since); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			criteria.Since = t
		}
	}
	if before := stringsTrim(query.Before); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			criteria.Before = t
		}
	}

	seqNums, err := client.Search(criteria)
	if err != nil {
		return nil, err
	}
	if len(seqNums) == 0 {
		return []schema.MessageMetaSummary{}, nil
	}

	limit := query.Limit
	if limit > 0 && len(seqNums) > limit {
		seqNums = seqNums[len(seqNums)-limit:]
	}

	seqset := new(imap.SeqSet)
	for _, n := range seqNums {
		seqset.AddNum(n)
	}

	messages := make(chan *imap.Message, len(seqNums))
	done := make(chan error, 1)
	go func() {
		done <- client.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	var results []schema.MessageMetaSummary
	for msg := range messages {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		summary := schema.MessageMetaSummary{
			From:    safeEnvelopeFrom(msg),
			Subject: safeEnvelopeSubject(msg),
			Date:    safeEnvelopeDate(msg),
		}
		if msg.Envelope != nil && stringsTrim(msg.Envelope.MessageId) != "" {
			summary.ID = msg.Envelope.MessageId
		} else {
			summary.ID = fmt.Sprintf("%d", msg.SeqNum)
		}
		results = append(results, summary)
	}
	if err := <-done; err != nil {
		return nil, err
	}
	return results, nil
}

func (d *imapDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	client, err := d.connectFunc()
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	if _, err := client.Select(d.mailbox, true); err != nil {
		return nil, err
	}

	return d.fetchRawFromSession(ctx, client, id)
}

func (d *imapDriver) fetchRawFromSession(ctx context.Context, client imapSession, id string) ([]byte, error) {
	target, err := resolveFetchTarget(id)
	if err != nil {
		return nil, err
	}

	switch target.kind {
	case fetchTargetUID:
		return fetchMessageBody(ctx, client, true, target.value, id)
	case fetchTargetSequence:
		return fetchMessageBody(ctx, client, false, target.value, id)
	case fetchTargetMessageID:
		criteria := imap.NewSearchCriteria()
		criteria.Header.Add("Message-Id", target.headerValue)
		matches, err := client.Search(criteria)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("%w: %s", ErrMessageNotFound, id)
		}
		return fetchMessageBody(ctx, client, false, matches[0], id)
	default:
		return nil, fmt.Errorf("unsupported fetch target: %s", id)
	}
}

func (d *imapDriver) SendRaw(ctx context.Context, raw []byte) error {
	if stringsTrim(d.smtp.host) == "" || d.smtp.port == 0 || stringsTrim(d.smtp.username) == "" || stringsTrim(d.smtp.password) == "" {
		return fmt.Errorf("%w: smtp settings not configured for account", ErrTransportNotConfigured)
	}

	from, recipients, err := extractEnvelope(raw)
	if err != nil {
		return err
	}

	return smtpSendFunc(d.smtp, from, recipients, raw)
}

// ─── BulkFetcher interface ───────────────────────────────────────────────────

// FetchRawBulk fetches multiple messages over a single IMAP connection,
// avoiding O(n) TLS handshakes when syncing many messages.
func (d *imapDriver) FetchRawBulk(ctx context.Context, ids []string) ([]BulkMessage, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	client, err := d.connectFunc()
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	if _, err := client.Select(d.mailbox, true); err != nil {
		return nil, err
	}

	results := make([]BulkMessage, 0, len(ids))
	for _, id := range ids {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		raw, err := d.fetchRawFromSession(ctx, client, id)
		results = append(results, BulkMessage{ID: id, Raw: raw, Err: err})
	}
	return results, nil
}

// ─── Writer interface ────────────────────────────────────────────────────────

// Delete permanently removes a message by setting \Deleted and expunging.
func (d *imapDriver) Delete(ctx context.Context, id string) error {
	client, err := d.connectFunc()
	if err != nil {
		return err
	}
	defer client.Logout()

	if _, err := client.Select(d.mailbox, false); err != nil {
		return err
	}

	uid, err := d.resolveUID(ctx, client, id)
	if err != nil {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	flags := []interface{}{imap.DeletedFlag}
	if err := client.UidStore(seqset, imap.FormatFlagsOp(imap.AddFlags, true), flags, nil); err != nil {
		return err
	}
	return client.Expunge(nil)
}

// Move copies a message to destMailbox and deletes the original.
func (d *imapDriver) Move(ctx context.Context, id, destMailbox string) error {
	client, err := d.connectFunc()
	if err != nil {
		return err
	}
	defer client.Logout()

	if _, err := client.Select(d.mailbox, false); err != nil {
		return err
	}

	uid, err := d.resolveUID(ctx, client, id)
	if err != nil {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	if err := client.UidCopy(seqset, destMailbox); err != nil {
		return err
	}

	flags := []interface{}{imap.DeletedFlag}
	if err := client.UidStore(seqset, imap.FormatFlagsOp(imap.AddFlags, true), flags, nil); err != nil {
		return err
	}
	return client.Expunge(nil)
}

// MarkRead sets or clears the \Seen flag on a message.
func (d *imapDriver) MarkRead(ctx context.Context, id string, read bool) error {
	client, err := d.connectFunc()
	if err != nil {
		return err
	}
	defer client.Logout()

	if _, err := client.Select(d.mailbox, false); err != nil {
		return err
	}

	uid, err := d.resolveUID(ctx, client, id)
	if err != nil {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	flags := []interface{}{imap.SeenFlag}
	op := imap.FlagsOp(imap.AddFlags)
	if !read {
		op = imap.FlagsOp(imap.RemoveFlags)
	}
	return client.UidStore(seqset, imap.FormatFlagsOp(op, true), flags, nil)
}

// resolveUID returns the UID for any supported message ID format.
func (d *imapDriver) resolveUID(ctx context.Context, client imapSession, id string) (uint32, error) {
	target, err := resolveFetchTarget(id)
	if err != nil {
		return 0, err
	}

	switch target.kind {
	case fetchTargetUID:
		return target.value, nil

	case fetchTargetSequence:
		seqset := new(imap.SeqSet)
		seqset.AddNum(target.value)
		ch := make(chan *imap.Message, 1)
		done := make(chan error, 1)
		go func() { done <- client.Fetch(seqset, []imap.FetchItem{imap.FetchUid}, ch) }()
		var uid uint32
		for msg := range ch {
			if msg != nil {
				uid = msg.Uid
			}
		}
		if err := <-done; err != nil {
			return 0, err
		}
		if uid == 0 {
			return 0, fmt.Errorf("%w: %s", ErrMessageNotFound, id)
		}
		return uid, nil

	case fetchTargetMessageID:
		criteria := imap.NewSearchCriteria()
		criteria.Header.Add("Message-Id", target.headerValue)
		matches, err := client.Search(criteria)
		if err != nil {
			return 0, err
		}
		if len(matches) == 0 {
			return 0, fmt.Errorf("%w: %s", ErrMessageNotFound, id)
		}
		seqset := new(imap.SeqSet)
		seqset.AddNum(matches[0])
		ch := make(chan *imap.Message, 1)
		done := make(chan error, 1)
		go func() { done <- client.Fetch(seqset, []imap.FetchItem{imap.FetchUid}, ch) }()
		var uid uint32
		for msg := range ch {
			if msg != nil {
				uid = msg.Uid
			}
		}
		if err := <-done; err != nil {
			return 0, err
		}
		if uid == 0 {
			return 0, fmt.Errorf("%w: %s", ErrMessageNotFound, id)
		}
		return uid, nil

	default:
		return 0, fmt.Errorf("unsupported fetch target: %s", id)
	}
}

// ─── Connection ──────────────────────────────────────────────────────────────

func (d *imapDriver) connect() (imapSession, error) {
	addr := fmt.Sprintf("%s:%d", d.host, d.port)

	var (
		c   *imapclient.Client
		err error
	)

	if d.useTLS {
		c, err = imapclient.DialTLS(addr, &tls.Config{
			ServerName: d.host,
		})
	} else {
		c, err = imapclient.Dial(addr)
	}
	if err != nil {
		return nil, err
	}

	if err := c.Login(d.username, d.password); err != nil {
		_ = c.Logout()
		return nil, err
	}

	return c, nil
}

// ─── Fetch helpers ───────────────────────────────────────────────────────────

func resolveFetchTarget(id string) (fetchTarget, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return fetchTarget{}, fmt.Errorf("message id is required")
	}

	for _, prefix := range []string{"imap:uid:", "uid:"} {
		if strings.HasPrefix(strings.ToLower(trimmed), prefix) {
			value, err := parsePositiveUint32(trimmed[len(prefix):])
			if err != nil {
				return fetchTarget{}, fmt.Errorf("invalid uid: %s", id)
			}
			return fetchTarget{kind: fetchTargetUID, value: value}, nil
		}
	}

	if isDigits(trimmed) {
		value, err := parsePositiveUint32(trimmed)
		if err != nil {
			return fetchTarget{}, fmt.Errorf("invalid sequence number: %s", id)
		}
		return fetchTarget{kind: fetchTargetSequence, value: value}, nil
	}

	return fetchTarget{
		kind:        fetchTargetMessageID,
		headerValue: normalizeMessageID(trimmed),
	}, nil
}

func fetchMessageBody(ctx context.Context, client imapSession, useUID bool, value uint32, id string) ([]byte, error) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(value)

	section, err := imap.ParseBodySectionName("BODY.PEEK[]")
	if err != nil {
		return nil, err
	}

	items := []imap.FetchItem{imap.FetchItem("BODY.PEEK[]")}
	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		if useUID {
			done <- client.UidFetch(seqset, items, messages)
			return
		}
		done <- client.Fetch(seqset, items, messages)
	}()

	var raw []byte
	for msg := range messages {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if msg == nil {
			continue
		}

		body := msg.GetBody(section)
		if body == nil {
			continue
		}

		data, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		raw = append([]byte(nil), data...)
	}

	if err := <-done; err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("raw body not found for message: %s", id)
	}
	return raw, nil
}

// ─── Envelope helpers ────────────────────────────────────────────────────────

func safeEnvelopeSubject(msg *imap.Message) string {
	if msg == nil || msg.Envelope == nil {
		return ""
	}
	return msg.Envelope.Subject
}

func safeEnvelopeFrom(msg *imap.Message) string {
	if msg == nil || msg.Envelope == nil || len(msg.Envelope.From) == 0 || msg.Envelope.From[0] == nil {
		return ""
	}

	from := msg.Envelope.From[0]
	address := strings.Trim(strings.Join([]string{from.MailboxName, from.HostName}, "@"), "@")
	if from.PersonalName == "" {
		return address
	}
	if address == "" {
		return from.PersonalName
	}
	return fmt.Sprintf("%s <%s>", from.PersonalName, address)
}

func safeEnvelopeDate(msg *imap.Message) string {
	if msg == nil || msg.Envelope == nil || msg.Envelope.Date.IsZero() {
		return ""
	}
	return msg.Envelope.Date.UTC().Format("2006-01-02T15:04:05Z")
}

// ─── String utilities ────────────────────────────────────────────────────────

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeMessageID(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		return trimmed
	}
	return "<" + strings.Trim(trimmed, "<>") + ">"
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func parsePositiveUint32(value string) (uint32, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("invalid positive integer: %s", value)
	}
	return uint32(parsed), nil
}

// ─── SMTP helpers ────────────────────────────────────────────────────────────

func extractEnvelope(raw []byte) (string, []string, error) {
	msg, err := netmail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return "", nil, err
	}

	fromList, err := msg.Header.AddressList("From")
	if err != nil || len(fromList) == 0 {
		return "", nil, fmt.Errorf("missing From header")
	}

	var recipients []string
	for _, header := range []string{"To", "Cc", "Bcc"} {
		addrs, err := msg.Header.AddressList(header)
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			recipients = append(recipients, addr.Address)
		}
	}

	if len(recipients) == 0 {
		return "", nil, fmt.Errorf("missing recipients")
	}

	return fromList[0].Address, recipients, nil
}

func sendSMTP(cfg smtpConfig, from string, to []string, raw []byte) error {
	addr := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
	auth := smtp.PlainAuth("", cfg.username, cfg.password, cfg.host)

	if cfg.useTLS {
		return wrapAuthError(sendSMTPTLS(addr, cfg.host, auth, from, to, raw))
	}

	return wrapAuthError(smtp.SendMail(addr, auth, from, to, raw))
}

func sendSMTPTLS(addr, host string, auth smtp.Auth, from string, to []string, raw []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Quit()

	if err := client.Auth(auth); err != nil {
		return wrapAuthError(err)
	}
	if err := client.Mail(from); err != nil {
		return wrapAuthError(err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return wrapAuthError(err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func wrapAuthError(err error) error {
	if err == nil {
		return nil
	}
	if stringsContainsAuth(err.Error()) {
		return fmt.Errorf("%w: %v", ErrAuthFailed, err)
	}
	return err
}

func stringsContainsAuth(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "auth") ||
		strings.Contains(lower, "authentication") ||
		strings.Contains(lower, "credentials invalid") ||
		strings.Contains(lower, "535")
}
