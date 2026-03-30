package driver

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
)

// Watch monitors mailbox using IMAP IDLE, reconnecting automatically on error.
func (d *imapDriver) Watch(ctx context.Context, mailbox string) (<-chan WatchEvent, error) {
	ch := make(chan WatchEvent, 32)
	go func() {
		defer close(ch)
		backoff := 5 * time.Second
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := d.idleOnce(ctx, mailbox, ch); err != nil {
				select {
				case ch <- WatchEvent{Err: fmt.Errorf("watch: %w", err)}:
				case <-ctx.Done():
					return
				}
				select {
				case <-time.After(backoff):
					if backoff < 5*time.Minute {
						backoff *= 2
					}
				case <-ctx.Done():
					return
				}
				continue
			}
			backoff = 5 * time.Second
		}
	}()
	return ch, nil
}

// idleOnce opens one IMAP IDLE session and handles incoming updates until ctx
// is cancelled or the connection drops.
func (d *imapDriver) idleOnce(ctx context.Context, mailbox string, ch chan<- WatchEvent) error {
	sess, err := d.connectFunc()
	if err != nil {
		return err
	}
	defer sess.Logout()

	// We need the real *imapclient.Client for IDLE.
	realClient, ok := sess.(*imapclient.Client)
	if !ok {
		// In test contexts the session may be a fake; treat as unsupported.
		<-ctx.Done()
		return nil
	}

	mbox, err := realClient.Select(mailbox, true)
	if err != nil {
		return err
	}
	knownCount := mbox.Messages

	// Set up the updates channel before entering IDLE.
	updates := make(chan imapclient.Update, 32)
	realClient.Updates = updates

	stop := make(chan struct{})
	var stopOnce sync.Once
	doStop := func() { stopOnce.Do(func() { close(stop) }) }

	idleErr := make(chan error, 1)
	go func() {
		idleErr <- realClient.Idle(stop, nil)
	}()

	defer doStop()

	for {
		select {
		case <-ctx.Done():
			doStop()
			<-idleErr
			return nil

		case err := <-idleErr:
			return err

		case update, ok := <-updates:
			if !ok {
				return nil
			}
			mu, ok := update.(*imapclient.MailboxUpdate)
			if !ok || mu.Mailbox == nil || mu.Mailbox.Messages <= knownCount {
				continue
			}

			// New messages arrived — exit IDLE to fetch their IDs.
			doStop()
			if err := <-idleErr; err != nil {
				return err
			}

			from := knownCount + 1
			newCount := mu.Mailbox.Messages
			knownCount = newCount

			seqset := new(imap.SeqSet)
			seqset.AddRange(from, newCount)
			msgs := make(chan *imap.Message, int(newCount-from+1))
			fetchDone := make(chan error, 1)
			go func() {
				fetchDone <- realClient.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, msgs)
			}()

			var newIDs []string
			for msg := range msgs {
				if msg == nil {
					continue
				}
				if msg.Envelope != nil && msg.Envelope.MessageId != "" {
					newIDs = append(newIDs, msg.Envelope.MessageId)
				} else {
					newIDs = append(newIDs, fmt.Sprintf("%d", msg.SeqNum))
				}
			}
			if err := <-fetchDone; err != nil {
				return err
			}

			if len(newIDs) > 0 {
				select {
				case ch <- WatchEvent{NewIDs: newIDs}:
				case <-ctx.Done():
					return nil
				}
			}

			// Re-enter IDLE.
			stopOnce = sync.Once{}
			stop = make(chan struct{})
			idleErr = make(chan error, 1)
			go func() {
				idleErr <- realClient.Idle(stop, nil)
			}()
		}
	}
}
