package main

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestHealthServer_OK(t *testing.T) {
	cfg := &Config{
		Target: TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{
			{Name: "Test", Host: "h", Port: 993, Secure: boolPtr(true), Auth: Auth{User: "u", Pass: "p"}, Folders: []string{"INBOX"}},
		},
	}

	manager := NewManager(cfg)
	fwd := NewForwarder(cfg.Sources[0], &mockSender{}, nil)
	fwd.setConnected(true)
	manager.forwarders = append(manager.forwarders, fwd)

	server := StartHealthServer(manager, 0)
	defer func() { _ = server.Close() }()

	resp, err := http.Get("http://" + server.Addr + "/health")
	if err != nil {
		t.Fatalf("failed to get health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var hr HealthResponse
	if err := json.Unmarshal(body, &hr); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if hr.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", hr.Status)
	}
	if len(hr.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(hr.Sources))
	}
	if !hr.Sources[0].Connected {
		t.Error("expected source to be connected")
	}
}

func TestHealthServer_Error(t *testing.T) {
	cfg := &Config{
		Target: TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{
			{Name: "Test", Host: "h", Port: 993, Secure: boolPtr(true), Auth: Auth{User: "u", Pass: "p"}, Folders: []string{"INBOX"}},
		},
	}

	manager := NewManager(cfg)
	fwd := NewForwarder(cfg.Sources[0], &mockSender{}, nil)
	manager.forwarders = append(manager.forwarders, fwd)

	server := StartHealthServer(manager, 0)
	defer func() { _ = server.Close() }()

	resp, err := http.Get("http://" + server.Addr + "/health")
	if err != nil {
		t.Fatalf("failed to get health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 503 {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var hr HealthResponse
	if err := json.Unmarshal(body, &hr); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if hr.Status != "error" {
		t.Errorf("expected status 'error', got %q", hr.Status)
	}
}

func TestHealthServer_NotFound(t *testing.T) {
	cfg := &Config{
		Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{{Name: "S", Host: "h", Port: 993, Secure: boolPtr(true), Auth: Auth{User: "u", Pass: "p"}, Folders: []string{"INBOX"}}},
	}

	manager := NewManager(cfg)
	server := StartHealthServer(manager, 0)
	defer func() { _ = server.Close() }()

	resp, err := http.Get("http://" + server.Addr + "/unknown")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestManager_GetOverallStatus(t *testing.T) {
	cfg := &Config{
		Target: TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{
			{Name: "A", Host: "h", Port: 993, Secure: boolPtr(true), Auth: Auth{User: "u", Pass: "p"}, Folders: []string{"INBOX"}},
			{Name: "B", Host: "h", Port: 993, Secure: boolPtr(true), Auth: Auth{User: "u", Pass: "p"}, Folders: []string{"INBOX"}},
		},
	}

	manager := NewManager(cfg)
	fwdA := NewForwarder(cfg.Sources[0], &mockSender{}, nil)
	fwdB := NewForwarder(cfg.Sources[1], &mockSender{}, nil)
	manager.forwarders = []*Forwarder{fwdA, fwdB}

	// Both disconnected → error
	if s := manager.GetOverallStatus(); s != "error" {
		t.Errorf("expected 'error', got %q", s)
	}

	// One connected → degraded
	fwdA.setConnected(true)
	if s := manager.GetOverallStatus(); s != "degraded" {
		t.Errorf("expected 'degraded', got %q", s)
	}

	// Both connected → ok
	fwdB.setConnected(true)
	if s := manager.GetOverallStatus(); s != "ok" {
		t.Errorf("expected 'ok', got %q", s)
	}
}
