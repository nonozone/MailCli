package driver

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/driver/drivertest"
	"github.com/nonozone/MailCli/pkg/schema"
)

func TestResolveFetchTargetUID(t *testing.T) {
	target, err := resolveFetchTarget("imap:uid:42")
	if err != nil {
		t.Fatalf("expected uid target to resolve: %v", err)
	}

	if target.kind != fetchTargetUID {
		t.Fatalf("expected uid target kind")
	}
	if target.value != 42 {
		t.Fatalf("expected uid value 42, got %d", target.value)
	}
}

func TestResolveFetchTargetUIDAlias(t *testing.T) {
	target, err := resolveFetchTarget("uid:42")
	if err != nil {
		t.Fatalf("expected uid alias target to resolve: %v", err)
	}

	if target.kind != fetchTargetUID || target.value != 42 {
		t.Fatalf("expected uid alias to resolve to uid=42")
	}
}

func TestResolveFetchTargetSequenceNumber(t *testing.T) {
	target, err := resolveFetchTarget("42")
	if err != nil {
		t.Fatalf("expected numeric target to resolve: %v", err)
	}

	if target.kind != fetchTargetSequence {
		t.Fatalf("expected sequence target kind")
	}
	if target.value != 42 {
		t.Fatalf("expected sequence value 42, got %d", target.value)
	}
}

func TestResolveFetchTargetMessageID(t *testing.T) {
	target, err := resolveFetchTarget("abc@example.com")
	if err != nil {
		t.Fatalf("expected message-id target to resolve: %v", err)
	}

	if target.kind != fetchTargetMessageID {
		t.Fatalf("expected message-id target kind")
	}
	if target.headerValue != "<abc@example.com>" {
		t.Fatalf("expected normalized message-id, got %q", target.headerValue)
	}
}

func TestIMAPDriverFetchRawSearchesMessageIDAndReturnsBody(t *testing.T) {
	session := &fakeIMAPSession{
		searchResults: []uint32{7},
		raw:           []byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Hello\r\n\r\nBody"),
	}
	drv := newTestIMAPDriver(session)

	raw, err := drv.FetchRaw(context.Background(), "hello@example.com")
	if err != nil {
		t.Fatalf("expected fetch raw to succeed: %v", err)
	}

	if session.selectedMailbox != "INBOX" || !session.selectReadOnly {
		t.Fatalf("expected FetchRaw to select INBOX in readonly mode")
	}
	if session.searchCriteria == nil {
		t.Fatalf("expected message-id search to run")
	}
	if got := session.searchCriteria.Header.Get("Message-Id"); got != "<hello@example.com>" {
		t.Fatalf("expected Message-Id search header, got %q", got)
	}
	if session.fetchSeqSet == nil || !session.fetchSeqSet.Contains(7) {
		t.Fatalf("expected fetch to use searched sequence number")
	}
	if len(session.fetchItems) != 1 || session.fetchItems[0] != imap.FetchItem("BODY.PEEK[]") {
		t.Fatalf("expected BODY.PEEK[] fetch, got %v", session.fetchItems)
	}
	if !bytes.Equal(raw, session.raw) {
		t.Fatalf("expected exact raw bytes to be returned")
	}
}

func TestIMAPDriverFetchRawUsesUIDFetchForExplicitUID(t *testing.T) {
	session := &fakeIMAPSession{
		raw: []byte("raw message"),
	}
	drv := newTestIMAPDriver(session)

	raw, err := drv.FetchRaw(context.Background(), "uid:42")
	if err != nil {
		t.Fatalf("expected uid fetch raw to succeed: %v", err)
	}

	if session.uidFetchSeqSet == nil || !session.uidFetchSeqSet.Contains(42) {
		t.Fatalf("expected uid fetch for explicit uid target")
	}
	if session.searchCriteria != nil {
		t.Fatalf("expected explicit uid fetch not to run header search")
	}
	if !bytes.Equal(raw, session.raw) {
		t.Fatalf("expected exact raw bytes from uid fetch")
	}
}

func TestIMAPDriverFetchRawUsesSequenceFetchForNumericID(t *testing.T) {
	session := &fakeIMAPSession{
		raw: []byte("raw message"),
	}
	drv := newTestIMAPDriver(session)

	raw, err := drv.FetchRaw(context.Background(), "42")
	if err != nil {
		t.Fatalf("expected sequence fetch raw to succeed: %v", err)
	}

	if session.fetchSeqSet == nil || !session.fetchSeqSet.Contains(42) {
		t.Fatalf("expected sequence fetch for numeric target")
	}
	if session.searchCriteria != nil || session.uidFetchSeqSet != nil {
		t.Fatalf("expected numeric target to skip search and uid fetch")
	}
	if !bytes.Equal(raw, session.raw) {
		t.Fatalf("expected exact raw bytes from sequence fetch")
	}
}

func TestIMAPDriverFetchRawReturnsNotFoundForMissingSearchResult(t *testing.T) {
	session := &fakeIMAPSession{}
	drv := newTestIMAPDriver(session)

	_, err := drv.FetchRaw(context.Background(), "missing@example.com")
	if err == nil {
		t.Fatalf("expected missing search result to fail")
	}
	if !strings.Contains(err.Error(), "missing@example.com") {
		t.Fatalf("expected missing id in error, got %v", err)
	}
}

func TestIMAPDriverFetchRawFailsWhenFetchReturnsNoBody(t *testing.T) {
	session := &fakeIMAPSession{
		searchResults: []uint32{7},
		skipMessage:   true,
	}
	drv := newTestIMAPDriver(session)

	_, err := drv.FetchRaw(context.Background(), "empty@example.com")
	if err == nil {
		t.Fatalf("expected empty fetch response to fail")
	}
	if !strings.Contains(err.Error(), "empty@example.com") {
		t.Fatalf("expected requested id in error, got %v", err)
	}
}

func TestIMAPDriverListUsesSessionFetch(t *testing.T) {
	session := &fakeIMAPSession{
		mailboxStatus: &imap.MailboxStatus{Name: "INBOX", Messages: 3},
		messages: []*imap.Message{
			{
				SeqNum: 3,
				Envelope: &imap.Envelope{
					Subject:   "Hello",
					MessageId: "<id-1@example.com>",
					Date:      time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
					From: []*imap.Address{
						{
							PersonalName: "Example Sender",
							MailboxName:  "sender",
							HostName:     "example.com",
						},
					},
				},
			},
		},
	}
	drv := newTestIMAPDriver(session)

	items, err := drv.List(context.Background(), schema.SearchQuery{})
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}

	if session.selectedMailbox != "INBOX" || !session.selectReadOnly {
		t.Fatalf("expected list to select INBOX in readonly mode")
	}
	if session.fetchSeqSet == nil || !session.fetchSeqSet.Contains(3) {
		t.Fatalf("expected list fetch to target latest message")
	}
	if len(items) != 1 || items[0].ID != "<id-1@example.com>" || items[0].Subject != "Hello" {
		t.Fatalf("expected list to use fetched envelope metadata, got %+v", items)
	}
	if items[0].From != "Example Sender <sender@example.com>" {
		t.Fatalf("expected list to include sender summary, got %+v", items[0])
	}
	if items[0].Date != "2026-03-26T12:00:00Z" {
		t.Fatalf("expected list to include normalized date, got %+v", items[0])
	}
}

func TestIMAPDriverContractSuite(t *testing.T) {
	restore := smtpSendFunc
	t.Cleanup(func() {
		smtpSendFunc = restore
	})

	drivertest.RunContractSuite(t, drivertest.Harness{
		NewDriver: func(t *testing.T) drivertest.Driver {
			t.Helper()

			section, err := imap.ParseBodySectionName("BODY[]")
			if err != nil {
				t.Fatalf("expected body section to parse: %v", err)
			}

			session := &fakeIMAPSession{
				mailboxStatus: &imap.MailboxStatus{Name: "INBOX", Messages: 1},
				searchResults: []uint32{1},
				messages: []*imap.Message{
					{
						SeqNum: 1,
						Envelope: &imap.Envelope{
							Subject:   "Contract message",
							MessageId: "<contract-1@example.com>",
							Date:      time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
							From: []*imap.Address{
								{
									PersonalName: "Contract Sender",
									MailboxName:  "sender",
									HostName:     "example.com",
								},
							},
						},
						Body: map[*imap.BodySectionName]imap.Literal{
							section: bytes.NewReader([]byte("From: sender@example.com\r\nTo: user@example.com\r\nSubject: Contract message\r\n\r\nBody")),
						},
					},
				},
			}

			drv := newTestIMAPDriver(session)
			drv.smtp = smtpConfig{
				host:     "smtp.example.com",
				port:     587,
				username: "user@example.com",
				password: "secret",
			}
			smtpSendFunc = func(cfg smtpConfig, from string, to []string, raw []byte) error {
				return nil
			}
			return drv
		},
		NewMissingDriver: func(t *testing.T) drivertest.Driver {
			t.Helper()
			session := &fakeIMAPSession{}
			return newTestIMAPDriver(session)
		},
		ListQuery:      schema.SearchQuery{Limit: 1},
		MissingFetchID: "missing@example.com",
		NotFoundError:  ErrMessageNotFound,
		SendRaw:        []byte("From: support@example.com\r\nTo: user@example.com\r\nSubject: Contract send\r\n\r\nHello"),
		AssertList: func(t *testing.T, got []schema.MessageMetaSummary) {
			t.Helper()
			if len(got) != 1 {
				t.Fatalf("expected one listed message, got %d", len(got))
			}
			if got[0].ID != "<contract-1@example.com>" {
				t.Fatalf("expected message-id backed list id, got %+v", got[0])
			}
		},
		AssertFetchRaw: func(t *testing.T, listed schema.MessageMetaSummary, raw []byte) {
			t.Helper()
			if !bytes.Contains(raw, []byte("Subject: Contract message")) {
				t.Fatalf("expected fetched raw message content, got %q", raw)
			}
		},
	})
}

func newTestIMAPDriver(session imapSession) *imapDriver {
	account := config.AccountConfig{
		Name:     "work",
		Driver:   "imap",
		Host:     "imap.example.com",
		Port:     993,
		Username: "user@example.com",
		Password: "secret",
		TLS:      true,
		Mailbox:  "INBOX",
	}

	drv, err := newIMAPDriver(account)
	if err != nil {
		panic(err)
	}

	imapDrv := drv.(*imapDriver)
	imapDrv.connectFunc = func() (imapSession, error) {
		return session, nil
	}
	return imapDrv
}

type fakeIMAPSession struct {
	selectedMailbox string
	selectReadOnly  bool
	searchCriteria  *imap.SearchCriteria
	fetchSeqSet     *imap.SeqSet
	uidFetchSeqSet  *imap.SeqSet
	fetchItems      []imap.FetchItem
	uidFetchItems   []imap.FetchItem
	searchResults   []uint32
	raw             []byte
	skipMessage     bool
	mailboxStatus   *imap.MailboxStatus
	messages        []*imap.Message
}

func (f *fakeIMAPSession) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	f.selectedMailbox = name
	f.selectReadOnly = readOnly
	if f.mailboxStatus != nil {
		return f.mailboxStatus, nil
	}
	return &imap.MailboxStatus{Name: name}, nil
}

func (f *fakeIMAPSession) Search(criteria *imap.SearchCriteria) ([]uint32, error) {
	f.searchCriteria = criteria
	return append([]uint32(nil), f.searchResults...), nil
}

func (f *fakeIMAPSession) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	f.fetchSeqSet = seqset
	f.fetchItems = append([]imap.FetchItem(nil), items...)
	return f.sendMessage(ch, 0)
}

func (f *fakeIMAPSession) UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	f.uidFetchSeqSet = seqset
	f.uidFetchItems = append([]imap.FetchItem(nil), items...)
	return f.sendMessage(ch, 42)
}

func (f *fakeIMAPSession) Logout() error {
	return nil
}

func (f *fakeIMAPSession) sendMessage(ch chan *imap.Message, uid uint32) error {
	defer close(ch)

	if len(f.messages) > 0 {
		for _, msg := range f.messages {
			ch <- msg
		}
		return nil
	}
	if f.skipMessage {
		return nil
	}

	section, err := imap.ParseBodySectionName("BODY[]")
	if err != nil {
		return err
	}

	ch <- &imap.Message{
		Uid: uid,
		Body: map[*imap.BodySectionName]imap.Literal{
			section: bytes.NewReader(f.raw),
		},
	}
	return nil
}
