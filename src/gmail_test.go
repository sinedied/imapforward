package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestGmailSender(server *httptest.Server) *GmailAPISender {
	return &GmailAPISender{
		config: GmailAPIConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RefreshToken: "test-refresh-token",
		},
		userEmail:      "test@gmail.com",
		logger:         newLogger("test"),
		httpClient:     server.Client(),
		labelCache:     make(map[string]string),
		gmailImportURL: server.URL + "/import",
		gmailLabelsURL: server.URL + "/labels",
		tokenURL:       server.URL + "/token",
	}
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": "test-access-token",
		"expires_in":   3600,
	})
}

func TestGmailAPISender_Send_Success(t *testing.T) {
	var receivedAuth string
	var receivedContentType string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenHandler(w, r)
		case "/import":
			receivedAuth = r.Header.Get("Authorization")
			receivedContentType = r.Header.Get("Content-Type")
			receivedBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id": "msg123"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sender := newTestGmailSender(server)
	rawMsg := []byte("From: sender@example.com\r\nSubject: Test\r\n\r\nBody")
	if err := sender.Send(context.Background(), rawMsg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer test-access-token" {
		t.Errorf("expected Bearer auth, got %q", receivedAuth)
	}
	if !strings.HasPrefix(receivedContentType, "multipart/related") {
		t.Errorf("expected multipart/related, got %q", receivedContentType)
	}
	if !strings.Contains(string(receivedBody), `"INBOX"`) {
		t.Error("expected INBOX label in metadata")
	}
	if !strings.Contains(string(receivedBody), `"UNREAD"`) {
		t.Error("expected UNREAD label in metadata")
	}
	if !strings.Contains(string(receivedBody), string(rawMsg)) {
		t.Error("expected raw message in body")
	}
}

func TestGmailAPISender_Send_TokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer server.Close()

	sender := newTestGmailSender(server)
	if err := sender.Send(context.Background(), []byte("test"), ""); err == nil {
		t.Fatal("expected error for bad token")
	}
}

func TestGmailAPISender_Send_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenHandler(w, r)
		default:
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error": "forbidden"}`))
		}
	}))
	defer server.Close()

	sender := newTestGmailSender(server)
	if err := sender.Send(context.Background(), []byte("test"), ""); err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestGmailAPISender_TokenCaching(t *testing.T) {
	tokenCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			tokenHandler(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id": "msg"}`))
		}
	}))
	defer server.Close()

	sender := newTestGmailSender(server)
	for i := 0; i < 2; i++ {
		if err := sender.Send(context.Background(), []byte("test"), ""); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}

	if tokenCalls != 1 {
		t.Errorf("expected 1 token call (cached), got %d", tokenCalls)
	}
}

func TestGmailAPISender_Send_WithTargetFolder(t *testing.T) {
	var receivedBody []byte
	labelListCalls := 0
	labelCreateCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			tokenHandler(w, r)
		case r.URL.Path == "/labels" && r.Method == http.MethodGet:
			labelListCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"labels": []}`))
		case r.URL.Path == "/labels" && r.Method == http.MethodPost:
			labelCreateCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id": "Label_123", "name": "Import/Work"}`))
		case r.URL.Path == "/import":
			receivedBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id": "msg456"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sender := newTestGmailSender(server)

	// First send — should list + create label
	if err := sender.Send(context.Background(), []byte("test msg 1"), "Import/Work"); err != nil {
		t.Fatalf("send 1: %v", err)
	}
	if labelListCalls != 1 {
		t.Errorf("expected 1 label list call, got %d", labelListCalls)
	}
	if labelCreateCalls != 1 {
		t.Errorf("expected 1 label create call, got %d", labelCreateCalls)
	}
	if !strings.Contains(string(receivedBody), `"Label_123"`) {
		t.Error("expected label ID Label_123 in metadata")
	}
	if strings.Contains(string(receivedBody), `"INBOX"`) {
		t.Error("INBOX should NOT be in labels when targetFolder is set")
	}

	// Second send — label should be cached, no more list/create calls
	if err := sender.Send(context.Background(), []byte("test msg 2"), "Import/Work"); err != nil {
		t.Fatalf("send 2: %v", err)
	}
	if labelListCalls != 1 {
		t.Errorf("expected label list calls still 1 (cached), got %d", labelListCalls)
	}
	if labelCreateCalls != 1 {
		t.Errorf("expected label create calls still 1 (cached), got %d", labelCreateCalls)
	}
}

func TestGmailAPISender_Send_ExistingLabel(t *testing.T) {
	labelCreateCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			tokenHandler(w, r)
		case r.URL.Path == "/labels" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"labels": [{"id": "Label_existing", "name": "MyLabel"}]}`))
		case r.URL.Path == "/labels" && r.Method == http.MethodPost:
			labelCreateCalls++
		case r.URL.Path == "/import":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id": "msg"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sender := newTestGmailSender(server)
	if err := sender.Send(context.Background(), []byte("test"), "MyLabel"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if labelCreateCalls != 0 {
		t.Errorf("expected 0 label create calls for existing label, got %d", labelCreateCalls)
	}
}

func TestGmailAPISender_Close(t *testing.T) {
	s := NewGmailAPISender(GmailAPIConfig{}, "test@gmail.com")
	if err := s.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
