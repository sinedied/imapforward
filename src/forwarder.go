package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

const (
	forwardedFlag      = "$Forwarded"
	reconnectBaseDelay = 1 * time.Second
	reconnectMaxDelay  = 60 * time.Second
)

// ForwarderStatus describes the current state of a forwarder.
type ForwarderStatus struct {
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	LastSync  string `json:"lastSync,omitempty"`
	Error     string `json:"error,omitempty"`
}

// IMAPClient is the subset of imapclient.Client used by the forwarder.
type IMAPClient interface {
	Login(username, password string) *imapclient.Command
	Logout() *imapclient.Command
	Close() error
	Closed() <-chan struct{}
	Select(mailbox string, options *imap.SelectOptions) *imapclient.SelectCommand
	UIDSearch(criteria *imap.SearchCriteria, options *imap.SearchOptions) *imapclient.SearchCommand
	Fetch(numSet imap.NumSet, options *imap.FetchOptions) *imapclient.FetchCommand
	Store(numSet imap.NumSet, store *imap.StoreFlags, options *imap.StoreOptions) *imapclient.FetchCommand
	Idle() (*imapclient.IdleCommand, error)
	List(ref string, pattern string, options *imap.ListOptions) *imapclient.ListCommand
	Append(mailbox string, size int64, options *imap.AppendOptions) *imapclient.AppendCommand
	Expunge() *imapclient.ExpungeCommand
}

// IMAPDialFunc creates a new IMAP client connection.
type IMAPDialFunc func(host string, port int, secure *bool) (IMAPClient, error)

// DefaultIMAPDial connects to an IMAP server using the appropriate TLS mode.
func DefaultIMAPDial(host string, port int, secure *bool) (IMAPClient, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	if secure != nil && *secure {
		return imapclient.DialTLS(addr, nil)
	}
	return imapclient.DialStartTLS(addr, nil)
}

// Forwarder monitors a source IMAP account and forwards new messages.
type Forwarder struct {
	source SourceConfig
	sender Sender
	dial   IMAPDialFunc
	logger *Logger

	mu        sync.Mutex
	connected bool
	lastSync  string
	lastErr   string

	onStatusChange func(ForwarderStatus)
}

// NewForwarder creates a new Forwarder for the given source.
func NewForwarder(
	source SourceConfig,
	sender Sender,
	dial IMAPDialFunc,
	onStatusChange func(ForwarderStatus),
) *Forwarder {
	return &Forwarder{
		source:         source,
		sender:         sender,
		dial:           dial,
		logger:         newLogger(source.Name),
		onStatusChange: onStatusChange,
	}
}

// GetStatus returns the current status of this forwarder.
func (f *Forwarder) GetStatus() ForwarderStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return ForwarderStatus{
		Name:      f.source.Name,
		Connected: f.connected,
		LastSync:  f.lastSync,
		Error:     f.lastErr,
	}
}

func (f *Forwarder) setConnected(v bool) {
	f.mu.Lock()
	f.connected = v
	f.mu.Unlock()
}

func (f *Forwarder) setError(err string) {
	f.mu.Lock()
	f.lastErr = err
	f.mu.Unlock()
}

func (f *Forwarder) setLastSync() {
	f.mu.Lock()
	f.lastSync = time.Now().UTC().Format(time.RFC3339)
	f.mu.Unlock()
}

// Run starts the forwarding loop. It blocks until ctx is cancelled.
func (f *Forwarder) Run(ctx context.Context) {
	delay := reconnectBaseDelay
	for {
		err := f.runOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			f.setError(err.Error())
			f.setConnected(false)
			f.notifyStatus()
			f.logger.Error("Connection error: %v", err)
		}

		f.logger.Info("Reconnecting in %v...", delay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		delay = min(delay*2, reconnectMaxDelay)
	}
}

// Stop cleans up sender resources.
func (f *Forwarder) Stop() {
	if f.sender != nil {
		if err := f.sender.Close(); err != nil {
			f.logger.Warn("Error closing sender: %v", err)
		}
	}
	f.setConnected(false)
	f.notifyStatus()
	f.logger.Info("Stopped")
}

func (f *Forwarder) notifyStatus() {
	if f.onStatusChange != nil {
		f.onStatusChange(f.GetStatus())
	}
}

// runOnce opens one IMAP connection per folder and monitors them concurrently.
// IMAP only allows one selected mailbox per connection, so each folder needs
// its own connection for concurrent IDLE monitoring.
func (f *Forwarder) runOnce(ctx context.Context) error {
	if len(f.source.Folders) == 1 {
		return f.monitorFolder(ctx, f.source.Folders[0], true)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(f.source.Folders))

	for i, folder := range f.source.Folders {
		wg.Add(1)
		go func(folder string, first bool) {
			defer wg.Done()
			if err := f.monitorFolder(ctx, folder, first); err != nil && ctx.Err() == nil {
				errCh <- err
			}
		}(folder, i == 0)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}
	return nil
}

func (f *Forwarder) connectSource(folder string) (*imapclient.Client, chan struct{}, error) {
	newMail := make(chan struct{}, 1)
	opts := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					select {
					case newMail <- struct{}{}:
					default:
					}
				}
			},
		},
	}

	addr := fmt.Sprintf("%s:%d", f.source.Host, f.source.Port)
	var client *imapclient.Client
	var err error
	if f.source.Secure != nil && *f.source.Secure {
		client, err = imapclient.DialTLS(addr, opts)
	} else {
		client, err = imapclient.DialStartTLS(addr, opts)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	if err := client.Login(f.source.Auth.User, f.source.Auth.Pass).Wait(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("login: %w", err)
	}

	return client, newMail, nil
}

// monitorFolder connects, selects a folder, forwards existing messages, and
// watches for new ones via IDLE. listFolders is only done on the first folder.
func (f *Forwarder) monitorFolder(ctx context.Context, folder string, listFolders bool) error {
	client, newMail, err := f.connectSource(folder)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	f.logger.Info("Connected to %s:%d for folder %s", f.source.Host, f.source.Port, folder)
	f.setConnected(true)
	f.setError("")
	f.notifyStatus()

	if listFolders {
		f.listAvailableFolders(client)
	}

	// Select the folder
	if _, err := client.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("select %q: %w", folder, err)
	}

	// Forward any existing unforwarded messages
	if err := f.forwardNewMessages(ctx, client); err != nil {
		f.logger.Error("Error forwarding messages in %q: %v", folder, err)
	}

	// Watch for new messages via IDLE
	return f.watchFolder(ctx, client, folder, newMail)
}

func (f *Forwarder) listAvailableFolders(client *imapclient.Client) {
	listCmd := client.List("", "*", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		f.logger.Warn("Failed to list folders: %v", err)
		return
	}
	names := make([]string, len(mailboxes))
	for i, m := range mailboxes {
		names[i] = m.Mailbox
	}
	f.logger.Info("Available folders: %v", names)
}

func (f *Forwarder) watchFolder(ctx context.Context, client *imapclient.Client, folder string, newMail <-chan struct{}) error {
	f.logger.Info("Watching for new messages in %s...", folder)

	for {
		if ctx.Err() != nil {
			return nil
		}

		idleCmd, err := client.Idle()
		if err != nil {
			return fmt.Errorf("idle: %w", err)
		}

		// Wait for context cancellation, connection close, or new mail notification
		select {
		case <-ctx.Done():
			_ = idleCmd.Close()
			_ = idleCmd.Wait()
			return nil
		case <-client.Closed():
			_ = idleCmd.Close()
			_ = idleCmd.Wait()
			return fmt.Errorf("connection closed during IDLE")
		case <-newMail:
			if err := idleCmd.Close(); err != nil {
				return fmt.Errorf("idle close: %w", err)
			}
			if err := idleCmd.Wait(); err != nil {
				return fmt.Errorf("idle wait: %w", err)
			}
		}

		// Drain any queued notifications
		select {
		case <-newMail:
		default:
		}

		f.logger.Info("New message(s) detected in %s", folder)
		if err := f.forwardNewMessages(ctx, client); err != nil {
			f.logger.Error("Error forwarding messages in %q: %v", folder, err)
		}
	}
}

func (f *Forwarder) forwardNewMessages(ctx context.Context, client *imapclient.Client) error {
	criteria := &imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen, imap.Flag(forwardedFlag)},
	}

	searchData, err := client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	uids := searchData.AllUIDs()
	if len(uids) == 0 {
		return nil
	}

	f.logger.Info("Found %d new message(s) to forward", len(uids))

	for _, uid := range uids {
		if ctx.Err() != nil {
			return nil
		}
		if err := f.forwardMessage(ctx, client, uid); err != nil {
			f.logger.Error("Message UID %d: failed to forward: %v", uid, err)
		}
	}

	return nil
}

func (f *Forwarder) forwardMessage(ctx context.Context, client *imapclient.Client, uid imap.UID) error {
	fetchOpts := &imap.FetchOptions{
		UID:   true,
		Flags: true,
		BodySection: []*imap.FetchItemBodySection{
			{Peek: true}, // BODY.PEEK[] — full message without marking as \Seen
		},
	}

	uidSet := imap.UIDSetNum(uid)
	fetchCmd := client.Fetch(uidSet, fetchOpts)
	defer func() { _ = fetchCmd.Close() }()

	msg := fetchCmd.Next()
	if msg == nil {
		return fmt.Errorf("no message data returned")
	}

	buf, err := msg.Collect()
	if err != nil {
		return fmt.Errorf("collect message: %w", err)
	}

	// Extract the raw message body. Try the exact section first, then fall
	// back to the first available body section (server may strip the Peek flag
	// from the response tag).
	var rawMessage []byte
	if data := buf.FindBodySection(&imap.FetchItemBodySection{Peek: true}); data != nil {
		rawMessage = data
	} else if len(buf.BodySection) > 0 {
		rawMessage = buf.BodySection[0].Bytes
	}
	if rawMessage == nil {
		return fmt.Errorf("no body section in response")
	}

	if err := f.sender.Send(ctx, rawMessage); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	f.logger.Info("Message UID %d: forwarded successfully", uid)

	// Mark as forwarded on source
	storeCmd := client.Store(uidSet, &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.Flag(forwardedFlag)},
	}, nil)
	if err := storeCmd.Close(); err != nil {
		f.logger.Warn("Message UID %d: failed to set forwarded flag: %v", uid, err)
	}

	// Optionally delete after forwarding
	if f.source.DeleteAfterForward {
		delCmd := client.Store(uidSet, &imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Silent: true,
			Flags:  []imap.Flag{imap.FlagDeleted},
		}, nil)
		if err := delCmd.Close(); err != nil {
			f.logger.Warn("Message UID %d: failed to mark deleted: %v", uid, err)
		}
		expungeCmd := client.Expunge()
		if err := expungeCmd.Close(); err != nil {
			f.logger.Warn("Message UID %d: expunge error: %v", uid, err)
		}
		f.logger.Info("Message UID %d: deleted from source", uid)
	}

	f.setLastSync()
	f.notifyStatus()
	return nil
}
