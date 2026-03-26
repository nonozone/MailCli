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

type imapSession interface {
	Select(name string, readOnly bool) (*imap.MailboxStatus, error)
	Search(criteria *imap.SearchCriteria) ([]uint32, error)
	Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
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

	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > uint32(limit-1) {
		from = mbox.Messages - uint32(limit-1)
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)
	items := []imap.FetchItem{imap.FetchEnvelope}
	messages := make(chan *imap.Message, limit)
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
			Subject: safeEnvelopeSubject(msg),
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

func safeEnvelopeSubject(msg *imap.Message) string {
	if msg == nil || msg.Envelope == nil {
		return ""
	}
	return msg.Envelope.Subject
}

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
		return sendSMTPTLS(addr, cfg.host, auth, from, to, raw)
	}

	return smtp.SendMail(addr, auth, from, to, raw)
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
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
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
