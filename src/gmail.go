package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	defaultGmailImportURL = "https://gmail.googleapis.com/upload/gmail/v1/users/me/messages/import"
	defaultGmailLabelsURL = "https://gmail.googleapis.com/gmail/v1/users/me/labels"
	defaultTokenURL       = "https://oauth2.googleapis.com/token"
	authURL               = "https://accounts.google.com/o/oauth2/v2/auth"
	gmailScope            = "https://www.googleapis.com/auth/gmail.insert https://www.googleapis.com/auth/gmail.labels"
	oauthTimeout          = 5 * time.Minute
)

// GmailAPISender forwards messages via the Gmail API messages.import endpoint.
// This method preserves original headers AND runs spam/phishing filters.
type GmailAPISender struct {
	mu             sync.Mutex
	config         GmailAPIConfig
	userEmail      string
	logger         *Logger
	httpClient     *http.Client
	accessToken    string
	tokenExpiry    time.Time
	labelsLoaded   bool
	labelCache     map[string]string // label name → label ID
	gmailImportURL string
	gmailLabelsURL string
	tokenURL       string
}

// NewGmailAPISender creates a new Gmail API sender.
func NewGmailAPISender(config GmailAPIConfig, userEmail string) *GmailAPISender {
	return &GmailAPISender{
		config:         config,
		userEmail:      userEmail,
		logger:         newLogger("gmail-api"),
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		labelCache:     make(map[string]string),
		gmailImportURL: defaultGmailImportURL,
		gmailLabelsURL: defaultGmailLabelsURL,
		tokenURL:       defaultTokenURL,
	}
}

func (s *GmailAPISender) Send(ctx context.Context, rawMessage []byte, targetFolder string, targetLabels []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, err := s.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	importURL := s.gmailImportURL + "?uploadType=multipart&internalDateSource=dateHeader&neverMarkSpam=false"

	primaryFolder := targetFolder
	if primaryFolder == "" {
		primaryFolder = "INBOX"
	}
	extraLabels := normalizeTargetLabels(primaryFolder, targetLabels)

	// Build label list — UNREAD always, INBOX only when no custom target folder
	labelIDs := []string{"UNREAD"}
	if primaryFolder != "INBOX" {
		labelID, err := s.ensureLabel(ctx, token, primaryFolder)
		if err != nil {
			return fmt.Errorf("ensure label %q: %w", primaryFolder, err)
		}
		labelIDs = append(labelIDs, labelID)
	} else {
		labelIDs = append(labelIDs, "INBOX")
	}
	for _, label := range extraLabels {
		labelID, err := s.ensureLabel(ctx, token, label)
		if err != nil {
			return fmt.Errorf("ensure label %q: %w", label, err)
		}
		labelIDs = append(labelIDs, labelID)
	}
	metadata, err := json.Marshal(map[string]interface{}{"labelIds": labelIDs})
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	// Build multipart body: metadata (with labels) + raw RFC822 message
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Part 1: JSON metadata
	metaHeader := make(textproto.MIMEHeader)
	metaHeader.Set("Content-Type", "application/json")
	metaPart, err := writer.CreatePart(metaHeader)
	if err != nil {
		return fmt.Errorf("create metadata part: %w", err)
	}
	if _, err := metaPart.Write(metadata); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	// Part 2: raw email
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

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gmail API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("gmail API error (status %d): %s", resp.StatusCode, body)
	}

	return nil
}

func (s *GmailAPISender) Close() error {
	return nil
}

// ensureLabel looks up a Gmail label by name, creating it if it doesn't exist.
// Returns the label ID. Results are cached.
func (s *GmailAPISender) ensureLabel(ctx context.Context, token, labelName string) (string, error) {
	if id, ok := s.labelCache[labelName]; ok {
		return id, nil
	}

	if err := s.loadLabels(ctx, token); err != nil {
		return "", err
	}
	if id, ok := s.labelCache[labelName]; ok {
		return id, nil
	}

	if parent := parentLabel(labelName); parent != "" {
		if _, err := s.ensureLabel(ctx, token, parent); err != nil {
			return "", err
		}
	}

	// Label not found — create it
	s.logger.Info("Creating Gmail label: %s", labelName)
	createBody, err := json.Marshal(map[string]string{"name": labelName})
	if err != nil {
		return "", fmt.Errorf("marshal label create payload: %w", err)
	}
	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.gmailLabelsURL, bytes.NewReader(createBody))
	if err != nil {
		return "", fmt.Errorf("create label request: %w", err)
	}
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := s.httpClient.Do(createReq)
	if err != nil {
		return "", fmt.Errorf("create label: %w", err)
	}
	defer func() { _ = createResp.Body.Close() }()

	if createResp.StatusCode < 200 || createResp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(createResp.Body, 1024))
		return "", fmt.Errorf("create label failed (status %d): %s", createResp.StatusCode, body)
	}

	var created struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("decode created label: %w", err)
	}

	s.logger.Info("Created Gmail label: %s (ID: %s)", created.Name, created.ID)
	s.labelCache[labelName] = created.ID
	return created.ID, nil
}

func (s *GmailAPISender) loadLabels(ctx context.Context, token string) error {
	if s.labelsLoaded {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.gmailLabelsURL, nil)
	if err != nil {
		return fmt.Errorf("create list labels request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("list labels: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("list labels failed (status %d): %s", resp.StatusCode, body)
	}

	var listResp struct {
		Labels []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return fmt.Errorf("decode labels: %w", err)
	}

	for _, l := range listResp.Labels {
		s.labelCache[l.Name] = l.ID
	}

	s.labelsLoaded = true
	return nil
}

func parentLabel(label string) string {
	idx := strings.LastIndex(label, "/")
	if idx <= 0 {
		return ""
	}
	return label[:idx]
}

// getAccessToken returns a valid access token, refreshing if expired.
func (s *GmailAPISender) getAccessToken(ctx context.Context) (string, error) {
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return s.accessToken, nil
	}

	s.logger.Debug("Refreshing access token")

	data := url.Values{
		"client_id":     {s.config.ClientID},
		"client_secret": {s.config.ClientSecret},
		"refresh_token": {s.config.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
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
		return "", fmt.Errorf("decode token response: %w", err)
	}

	s.accessToken = tokenResp.AccessToken
	// Refresh 60s before expiry to avoid edge-case failures.
	// Clamp at zero so very short-lived tokens do not appear immediately expired.
	refreshIn := tokenResp.ExpiresIn - 60
	if refreshIn < 0 {
		refreshIn = 0
	}
	s.tokenExpiry = time.Now().Add(time.Duration(refreshIn) * time.Second)
	s.logger.Debug("Access token refreshed, expires in %ds", tokenResp.ExpiresIn)

	return s.accessToken, nil
}

// RunOAuthFlow runs an interactive OAuth2 authorization code flow to obtain
// a refresh token. It starts a local HTTP server, opens the browser for consent,
// and exchanges the authorization code for tokens.
func RunOAuthFlow(clientID, clientSecret string) error {
	log := newLogger("auth")

	// Generate random state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Start local server to receive the callback
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			_, _ = fmt.Fprint(w, "Authorization failed: invalid state parameter.\nYou can close this window.")
			errCh <- fmt.Errorf("invalid OAuth state — possible CSRF")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			_, _ = fmt.Fprintf(w, "Authorization failed: %s\nYou can close this window.", errMsg)
			errCh <- fmt.Errorf("authorization failed: %s", errMsg)
			return
		}
		_, _ = fmt.Fprint(w, "Authorization successful! You can close this window.")
		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	listener, err := listenOnAvailablePort()
	if err != nil {
		return fmt.Errorf("start local server: %w", err)
	}
	defer func() { _ = server.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Build authorization URL
	authParams := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {gmailScope},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}
	authorizationURL := authURL + "?" + authParams.Encode()

	fmt.Println("Opening browser for Google authorization...")
	fmt.Println()
	fmt.Println("If the browser doesn't open, visit this URL manually:")
	fmt.Println(authorizationURL)
	fmt.Println()

	_ = openBrowser(authorizationURL)

	// Wait for callback (with timeout)
	log.Info("Waiting for authorization callback on port %d...", port)
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(oauthTimeout):
		return fmt.Errorf("authorization timed out after %v — no callback received", oauthTimeout)
	}

	// Exchange code for tokens
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	}

	resp, err := http.PostForm(defaultTokenURL, data)
	if err != nil {
		return fmt.Errorf("token exchange: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, body)
	}

	var tokenResp struct {
		RefreshToken string `json:"refresh_token"`
		AccessToken  string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.RefreshToken == "" {
		return fmt.Errorf("no refresh token received (was the consent prompt shown?)")
	}

	fmt.Println()
	fmt.Println("Authorization successful!")
	fmt.Println()
	fmt.Println("Add this to your config.json:")
	fmt.Println()
	fmt.Printf("  \"gmailApi\": {\n")
	fmt.Printf("    \"clientId\": %q,\n", clientID)
	fmt.Printf("    \"clientSecret\": %q,\n", clientSecret)
	fmt.Printf("    \"refreshToken\": %q\n", tokenResp.RefreshToken)
	fmt.Printf("  }\n")

	return nil
}

func listenOnAvailablePort() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

func openBrowser(targetURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", targetURL).Start()
	case "linux":
		return exec.Command("xdg-open", targetURL).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
