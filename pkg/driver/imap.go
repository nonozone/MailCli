package driver

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/yourname/mailcli/internal/config"
	"github.com/yourname/mailcli/pkg/schema"
)

type imapDriver struct {
	host     string
	port     int
	username string
	password string
	useTLS   bool
	mailbox  string
}

func newIMAPDriver(account config.AccountConfig) (Driver, error) {
	if stringsTrim(account.Host) == "" || account.Port == 0 || stringsTrim(account.Username) == "" || stringsTrim(account.Password) == "" {
		return nil, fmt.Errorf("imap account requires host, port, username, and password")
	}

	mailbox := account.Mailbox
	if stringsTrim(mailbox) == "" {
		mailbox = "INBOX"
	}

	return &imapDriver{
		host:     account.Host,
		port:     account.Port,
		username: account.Username,
		password: account.Password,
		useTLS:   account.TLS || account.Port == 993,
		mailbox:  mailbox,
	}, nil
}

func (d *imapDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	client, err := d.connect()
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
	return nil, fmt.Errorf("imap fetch raw not implemented")
}

func (d *imapDriver) SendRaw(ctx context.Context, raw []byte) error {
	return fmt.Errorf("imap send raw not implemented")
}

func (d *imapDriver) connect() (*imapclient.Client, error) {
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

func safeEnvelopeSubject(msg *imap.Message) string {
	if msg == nil || msg.Envelope == nil {
		return ""
	}
	return msg.Envelope.Subject
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}
