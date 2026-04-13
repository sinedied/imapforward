package main

import (
	"context"
	"sync"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestForwarder_GetStatus_Initial(t *testing.T) {
	source := SourceConfig{
		Name:    "Test",
		Host:    "imap.test.com",
		Port:    993,
		Secure:  boolPtr(true),
		Auth:    Auth{User: "u", Pass: "p"},
		Folders: []string{"INBOX"},
	}

	fwd := NewForwarder(source, nil, nil)
	status := fwd.GetStatus()

	if status.Name != "Test" {
		t.Errorf("expected name 'Test', got %q", status.Name)
	}
	if status.Connected {
		t.Error("expected not connected initially")
	}
	if status.LastSync != "" {
		t.Error("expected empty lastSync initially")
	}
	if status.Error != "" {
		t.Error("expected empty error initially")
	}
}

func TestForwarder_StatusUpdates(t *testing.T) {
	source := SourceConfig{
		Name:    "Test",
		Host:    "imap.test.com",
		Port:    993,
		Secure:  boolPtr(true),
		Auth:    Auth{User: "u", Pass: "p"},
		Folders: []string{"INBOX"},
	}

	var mu sync.Mutex
	var lastStatus ForwarderStatus
	fwd := NewForwarder(source, nil, func(s ForwarderStatus) {
		mu.Lock()
		lastStatus = s
		mu.Unlock()
	})

	fwd.setConnected(true)
	fwd.notifyStatus()

	mu.Lock()
	if !lastStatus.Connected {
		t.Error("expected connected=true in status callback")
	}
	mu.Unlock()

	fwd.setError("connection lost")
	fwd.notifyStatus()

	mu.Lock()
	if lastStatus.Error != "connection lost" {
		t.Errorf("expected error 'connection lost', got %q", lastStatus.Error)
	}
	mu.Unlock()
}

func TestForwarder_Stop(t *testing.T) {
	source := SourceConfig{
		Name:    "Test",
		Host:    "imap.test.com",
		Port:    993,
		Secure:  boolPtr(true),
		Auth:    Auth{User: "u", Pass: "p"},
		Folders: []string{"INBOX"},
	}

	mockSender := &mockSender{}
	fwd := NewForwarder(source, mockSender, nil)

	fwd.setConnected(true)
	fwd.Stop()

	if fwd.GetStatus().Connected {
		t.Error("expected disconnected after stop")
	}
	if !mockSender.closed {
		t.Error("expected sender to be closed")
	}
}

func TestForwarder_RunCancellation(t *testing.T) {
	source := SourceConfig{
		Name:    "Test",
		Host:    "invalid.host.example",
		Port:    993,
		Secure:  boolPtr(true),
		Auth:    Auth{User: "u", Pass: "p"},
		Folders: []string{"INBOX"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	fwd := NewForwarder(source, &mockSender{}, nil)

	// Should return immediately due to cancelled context
	fwd.Run(ctx)
}

func TestSupportsIdle(t *testing.T) {
	caps := imap.CapSet{
		imap.CapIdle: {},
	}

	if !supportsIdle(caps) {
		t.Fatal("expected IDLE to be supported")
	}
}

func TestSupportsIdle_IMAP4rev2(t *testing.T) {
	caps := imap.CapSet{
		imap.CapIMAP4rev2: {},
	}

	if !supportsIdle(caps) {
		t.Fatal("expected IMAP4rev2 to imply IDLE support")
	}
}

func TestSupportsIdle_Unsupported(t *testing.T) {
	caps := imap.CapSet{
		imap.CapIMAP4rev1: {},
		imap.CapUIDPlus:   {},
	}

	if supportsIdle(caps) {
		t.Fatal("expected IDLE to be unsupported")
	}
}

func TestShouldFallbackFromIdleError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "bad unrecognized command",
			err:  context.DeadlineExceeded, // overwritten below would be awkward inline if no fmt
			want: true,
		},
		{
			name: "not supported",
			err:  nil,
			want: true,
		},
		{
			name: "other error",
			err:  nil,
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}

	tests[0].err = testError("imap: BAD Unrecognized command")
	tests[1].err = testError("imap: IDLE not supported")
	tests[2].err = testError("connection reset by peer")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldFallbackFromIdleError(tt.err); got != tt.want {
				t.Fatalf("shouldFallbackFromIdleError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

type mockSender struct {
	sent   [][]byte
	closed bool
	err    error
}

func (m *mockSender) Send(ctx context.Context, rawMessage []byte, targetFolder string) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, rawMessage)
	return nil
}

func (m *mockSender) Close() error {
	m.closed = true
	return nil
}

type testError string

func (e testError) Error() string {
	return string(e)
}
