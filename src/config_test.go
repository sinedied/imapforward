package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateConfig_ValidMinimal(t *testing.T) {
	c := &Config{
		Target: TargetConfig{
			Host: "imap.gmail.com",
			Port: 993,
			Auth: Auth{User: "user@gmail.com", Pass: "pass"},
		},
		Sources: []SourceConfig{
			{Name: "Work", Host: "imap.work.com", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		},
	}
	if err := validateConfig(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Target.Folder != "INBOX" {
		t.Errorf("expected default folder INBOX, got %q", c.Target.Folder)
	}
	if c.ForwardMethod != "imap" {
		t.Errorf("expected default forwardMethod imap, got %q", c.ForwardMethod)
	}
	if c.HealthCheck.Port != 8080 {
		t.Errorf("expected default healthCheck port 8080, got %d", c.HealthCheck.Port)
	}
}

func TestValidateConfig_DefaultSecure993(t *testing.T) {
	c := &Config{
		Target: TargetConfig{
			Host: "imap.gmail.com",
			Port: 993,
			Auth: Auth{User: "u", Pass: "p"},
		},
		Sources: []SourceConfig{
			{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if c.Target.Secure == nil || !*c.Target.Secure {
		t.Error("expected target secure=true for port 993")
	}
	if c.Sources[0].Secure == nil || !*c.Sources[0].Secure {
		t.Error("expected source secure=true for port 993")
	}
}

func TestValidateConfig_DefaultSecure465(t *testing.T) {
	c := &Config{
		Target: TargetConfig{
			Host: "imap.gmail.com",
			Port: 465,
			Auth: Auth{User: "u", Pass: "p"},
		},
		Sources: []SourceConfig{
			{Name: "S", Host: "h", Port: 465, Auth: Auth{User: "u", Pass: "p"}},
		},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if c.Target.Secure == nil || !*c.Target.Secure {
		t.Error("expected target secure=true for port 465")
	}
}

func TestValidateConfig_DefaultSecure143(t *testing.T) {
	c := &Config{
		Target: TargetConfig{
			Host: "imap.gmail.com",
			Port: 143,
			Auth: Auth{User: "u", Pass: "p"},
		},
		Sources: []SourceConfig{
			{Name: "S", Host: "h", Port: 143, Auth: Auth{User: "u", Pass: "p"}},
		},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if c.Target.Secure == nil || *c.Target.Secure {
		t.Error("expected target secure=false for port 143")
	}
}

func TestValidateConfig_ExplicitSecure(t *testing.T) {
	f := false
	c := &Config{
		Target: TargetConfig{
			Host: "h", Port: 993, Secure: &f,
			Auth: Auth{User: "u", Pass: "p"},
		},
		Sources: []SourceConfig{
			{Name: "S", Host: "h", Port: 993, Secure: &f, Auth: Auth{User: "u", Pass: "p"}},
		},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if *c.Target.Secure != false {
		t.Error("explicit secure=false should be preserved")
	}
}

func TestValidateConfig_DefaultFolders(t *testing.T) {
	c := &Config{
		Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if len(c.Sources[0].Folders) != 1 || c.Sources[0].Folders[0] != "INBOX" {
		t.Errorf("expected default folders [INBOX], got %v", c.Sources[0].Folders)
	}
}

func TestValidateConfig_CustomFolders(t *testing.T) {
	c := &Config{
		Target: TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{
			{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}, Folders: []string{"INBOX", "Sent"}},
		},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if len(c.Sources[0].Folders) != 2 {
		t.Errorf("expected 2 folders, got %d", len(c.Sources[0].Folders))
	}
}

func TestValidateConfig_MultipleSources(t *testing.T) {
	c := &Config{
		Target: TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{
			{Name: "A", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
			{Name: "B", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if len(c.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(c.Sources))
	}
}

func TestValidateConfig_CustomHealthPort(t *testing.T) {
	c := &Config{
		Target:      TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources:     []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
		HealthCheck: HealthCheckConfig{Port: 9090},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if c.HealthCheck.Port != 9090 {
		t.Errorf("expected port 9090, got %d", c.HealthCheck.Port)
	}
}

func TestValidateConfig_GmailAPIMethod(t *testing.T) {
	c := &Config{
		Target:        TargetConfig{Auth: Auth{User: "u@gmail.com"}},
		ForwardMethod: "gmail-api",
		GmailAPI:      &GmailAPIConfig{ClientID: "cid", ClientSecret: "cs", RefreshToken: "rt"},
		Sources:       []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if c.ForwardMethod != "gmail-api" {
		t.Errorf("expected gmail-api, got %q", c.ForwardMethod)
	}
}

func TestValidateConfig_GmailAPIMissingConfig(t *testing.T) {
	c := &Config{
		Target:        TargetConfig{Auth: Auth{User: "u@gmail.com"}},
		ForwardMethod: "gmail-api",
		Sources:       []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
	}
	err := validateConfig(c)
	if err == nil {
		t.Fatal("expected error for missing gmailApi config")
	}
}

func TestValidateConfig_GmailAPIMissingFields(t *testing.T) {
	tests := []struct {
		name   string
		config GmailAPIConfig
		want   string
	}{
		{"missing clientId", GmailAPIConfig{ClientSecret: "cs", RefreshToken: "rt"}, "gmailApi.clientId"},
		{"missing clientSecret", GmailAPIConfig{ClientID: "cid", RefreshToken: "rt"}, "gmailApi.clientSecret"},
		{"missing refreshToken", GmailAPIConfig{ClientID: "cid", ClientSecret: "cs"}, "gmailApi.refreshToken"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Target:        TargetConfig{Auth: Auth{User: "u@gmail.com"}},
				ForwardMethod: "gmail-api",
				GmailAPI:      &tt.config,
				Sources:       []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
			}
			err := validateConfig(c)
			if err == nil {
				t.Fatal("expected error")
			}
			if !contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidateConfig_TargetFolder(t *testing.T) {
	c := &Config{
		Target: TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{
			{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}, TargetFolder: "Import/Work"},
		},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if c.Sources[0].TargetFolder != "Import/Work" {
		t.Errorf("expected targetFolder 'Import/Work', got %q", c.Sources[0].TargetFolder)
	}
}

func TestValidateConfig_TargetFolderEmpty(t *testing.T) {
	c := &Config{
		Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if c.Sources[0].TargetFolder != "" {
		t.Errorf("expected empty targetFolder, got %q", c.Sources[0].TargetFolder)
	}
}

func TestValidateConfig_SMTPMethod(t *testing.T) {
	c := &Config{
		Target:        TargetConfig{Host: "smtp.gmail.com", Port: 587, Auth: Auth{User: "u", Pass: "p"}},
		ForwardMethod: "smtp",
		Sources:       []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if c.ForwardMethod != "smtp" {
		t.Errorf("expected forwardMethod smtp, got %q", c.ForwardMethod)
	}
}

func TestValidateConfig_Errors(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "missing target host",
			cfg: Config{
				Target:  TargetConfig{Port: 993, Auth: Auth{User: "u", Pass: "p"}},
				Sources: []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
			},
			want: "target.host must be a non-empty string",
		},
		{
			name: "missing target port",
			cfg: Config{
				Target:  TargetConfig{Host: "h", Auth: Auth{User: "u", Pass: "p"}},
				Sources: []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
			},
			want: "target.port must be a positive integer",
		},
		{
			name: "missing target auth user",
			cfg: Config{
				Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{Pass: "p"}},
				Sources: []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
			},
			want: "target.auth.user must be a non-empty string",
		},
		{
			name: "missing target auth pass",
			cfg: Config{
				Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u"}},
				Sources: []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
			},
			want: "target.auth.pass must be a non-empty string",
		},
		{
			name: "empty sources",
			cfg: Config{
				Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
				Sources: []SourceConfig{},
			},
			want: "sources must be a non-empty array",
		},
		{
			name: "missing source name",
			cfg: Config{
				Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
				Sources: []SourceConfig{{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
			},
			want: "sources[0].name must be a non-empty string",
		},
		{
			name: "missing source host",
			cfg: Config{
				Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
				Sources: []SourceConfig{{Name: "S", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
			},
			want: "sources[0].host must be a non-empty string",
		},
		{
			name: "missing source port",
			cfg: Config{
				Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
				Sources: []SourceConfig{{Name: "S", Host: "h", Auth: Auth{User: "u", Pass: "p"}}},
			},
			want: "sources[0].port must be a positive integer",
		},
		{
			name: "missing source auth",
			cfg: Config{
				Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
				Sources: []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{Pass: "p"}}},
			},
			want: "sources[0].auth.user must be a non-empty string",
		},
		{
			name: "invalid forward method",
			cfg: Config{
				Target:        TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
				Sources:       []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
				ForwardMethod: "invalid",
			},
			want: `forwardMethod must be "imap", "smtp", or "gmail-api"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.want {
				t.Errorf("expected error %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	content := `{
		"target": {
			"host": "imap.gmail.com",
			"port": 993,
			"auth": {"user": "u@gmail.com", "pass": "p"}
		},
		"sources": [
			{"name": "Work", "host": "imap.work.com", "port": 993, "auth": {"user": "u", "pass": "p"}}
		]
	}`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Target.Host != "imap.gmail.com" {
		t.Errorf("unexpected host: %s", cfg.Target.Host)
	}
	if len(cfg.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(cfg.Sources))
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := loadConfig("/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadConfig_ValidationError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"target": {}, "sources": []}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadConfig(path)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadConfig_SMTPConfig(t *testing.T) {
	content := `{
		"target": {
			"host": "smtp.gmail.com",
			"port": 587,
			"auth": {"user": "u@gmail.com", "pass": "p"}
		},
		"forwardMethod": "smtp",
		"sources": [
			{"name": "Work", "host": "imap.work.com", "port": 993, "auth": {"user": "u", "pass": "p"}}
		]
	}`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ForwardMethod != "smtp" {
		t.Errorf("expected forwardMethod smtp, got %q", cfg.ForwardMethod)
	}
	if cfg.Target.Host != "smtp.gmail.com" {
		t.Errorf("expected target host smtp.gmail.com, got %q", cfg.Target.Host)
	}
}

func TestDeleteAfterForward_Default(t *testing.T) {
	c := &Config{
		Target:  TargetConfig{Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}},
		Sources: []SourceConfig{{Name: "S", Host: "h", Port: 993, Auth: Auth{User: "u", Pass: "p"}}},
	}
	if err := validateConfig(c); err != nil {
		t.Fatal(err)
	}
	if c.Sources[0].DeleteAfterForward {
		t.Error("expected deleteAfterForward default to be false")
	}
}
