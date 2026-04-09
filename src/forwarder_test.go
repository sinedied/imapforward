package main

import (
	"context"
	"sync"
	"testing"
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

	fwd := NewForwarder(source, nil, nil, nil)
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
	fwd := NewForwarder(source, nil, nil, func(s ForwarderStatus) {
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
	fwd := NewForwarder(source, mockSender, nil, nil)

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

	fwd := NewForwarder(source, &mockSender{}, DefaultIMAPDial, nil)

	// Should return immediately due to cancelled context
	fwd.Run(ctx)
}

type mockSender struct {
	sent   [][]byte
	closed bool
	err    error
}

func (m *mockSender) Send(ctx context.Context, rawMessage []byte) error {
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
