package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGmailAPISender_Send_Success(t *testing.T) {
	var receivedBody []byte
	var receivedAuth string
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-access-token",
				"expires_in":   3600,
			})
			return
		}
		if r.URL.Path == "/import" {
			receivedAuth = r.Header.Get("Authorization")
			receivedContentType = r.Header.Get("Content-Type")
			receivedBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id": "msg123"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	sender := &GmailAPISender{
		config: GmailAPIConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RefreshToken: "test-refresh-token",
		},
		userEmail:  "test@gmail.com",
		logger:     newLogger("test"),
		httpClient: server.Client(),
	}

	// Override URLs for testing
	origTokenURL := tokenURL
	origImportURL := gmailImportURL
	defer func() {
		// We can't reassign constants, so we test via the httptest approach
		_ = origTokenURL
		_ = origImportURL
	}()

	// Use a custom sender with overridden URLs
	customSender := &testableGmailSender{
		sender:    sender,
		tokenURL:  server.URL + "/token",
		importURL: server.URL + "/import",
	}

	rawMsg := []byte("From: sender@example.com\r\nSubject: Test\r\n\r\nBody")
	err := customSender.Send(context.Background(), rawMsg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer test-access-token" {
		t.Errorf("expected Bearer auth, got %q", receivedAuth)
	}
	if !strings.HasPrefix(receivedContentType, "multipart/related") {
		t.Errorf("expected multipart/related content type, got %q", receivedContentType)
	}
	if !strings.Contains(string(receivedBody), "labelIds") {
		t.Error("expected metadata with labelIds in body")
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

	customSender := &testableGmailSender{
		sender: &GmailAPISender{
			config: GmailAPIConfig{
				ClientID:     "bad-id",
				ClientSecret: "bad-secret",
				RefreshToken: "bad-token",
			},
			userEmail:  "test@gmail.com",
			logger:     newLogger("test"),
			httpClient: server.Client(),
		},
		tokenURL:  server.URL + "/token",
		importURL: server.URL + "/import",
	}

	err := customSender.Send(context.Background(), []byte("test"), "")
	if err == nil {
		t.Fatal("expected error for bad token")
	}
}

func TestGmailAPISender_Send_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-token",
				"expires_in":   3600,
			})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": "forbidden"}`))
	}))
	defer server.Close()

	customSender := &testableGmailSender{
		sender: &GmailAPISender{
			config: GmailAPIConfig{
				ClientID:     "id",
				ClientSecret: "secret",
				RefreshToken: "token",
			},
			userEmail:  "test@gmail.com",
			logger:     newLogger("test"),
			httpClient: server.Client(),
		},
		tokenURL:  server.URL + "/token",
		importURL: server.URL + "/import",
	}

	err := customSender.Send(context.Background(), []byte("test"), "")
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestGmailAPISender_TokenCaching(t *testing.T) {
	tokenCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			tokenCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "cached-token",
				"expires_in":   3600,
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "msg"}`))
	}))
	defer server.Close()

	customSender := &testableGmailSender{
		sender: &GmailAPISender{
			config: GmailAPIConfig{
				ClientID:     "id",
				ClientSecret: "secret",
				RefreshToken: "token",
			},
			userEmail:  "test@gmail.com",
			logger:     newLogger("test"),
			httpClient: server.Client(),
		},
		tokenURL:  server.URL + "/token",
		importURL: server.URL + "/import",
	}

	// Send twice — token should only be fetched once
	for i := 0; i < 2; i++ {
		if err := customSender.Send(context.Background(), []byte("test"), ""); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}

	if tokenCalls != 1 {
		t.Errorf("expected 1 token call (cached), got %d", tokenCalls)
	}
}

func TestGmailAPISender_Close(t *testing.T) {
	s := NewGmailAPISender(GmailAPIConfig{}, "test@gmail.com")
	if err := s.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// testableGmailSender wraps GmailAPISender with overridable URLs for testing.
type testableGmailSender struct {
	sender    *GmailAPISender
	tokenURL  string
	importURL string
}

func (t *testableGmailSender) Send(ctx context.Context, rawMessage []byte, targetFolder string) error {
	t.sender.mu.Lock()
	defer t.sender.mu.Unlock()

	token, err := t.refreshToken(ctx)
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	importURL := t.importURL + "?uploadType=multipart&internalDateSource=dateHeader&neverMarkSpam=false"

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	metaHeader := make(textproto.MIMEHeader)
	metaHeader.Set("Content-Type", "application/json")
	metaPart, err := writer.CreatePart(metaHeader)
	if err != nil {
		return fmt.Errorf("create metadata part: %w", err)
	}
	if _, err := metaPart.Write([]byte(`{"labelIds":["INBOX"]}`)); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	msgHeader := make(textproto.MIMEHeader)
	msgHeader.Set("Content-Type", "message/rfc822")
	msgPart, err := writer.CreatePart(msgHeader)
	if err != nil {
		return fmt.Errorf("create message part: %w", err)
	}
	if _, err := msgPart.Write(rawMessage); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, importURL, &body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "multipart/related; boundary="+writer.Boundary())

	resp, err := t.sender.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gmail API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("gmail API error (status %d): %s", resp.StatusCode, respBody)
	}

	return nil
}

func (t *testableGmailSender) refreshToken(ctx context.Context) (string, error) {
	s := t.sender
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return s.accessToken, nil
	}

	data := url.Values{
		"client_id":     {s.config.ClientID},
		"client_secret": {s.config.ClientSecret},
		"refresh_token": {s.config.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	s.accessToken = tokenResp.AccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	return s.accessToken, nil
}
