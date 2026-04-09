package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Auth holds IMAP/SMTP authentication credentials.
type Auth struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

// SourceConfig is the configuration for a source IMAP account.
type SourceConfig struct {
	Name               string   `json:"name"`
	Host               string   `json:"host"`
	Port               int      `json:"port"`
	Secure             *bool    `json:"secure,omitempty"`
	Auth               Auth     `json:"auth"`
	Folders            []string `json:"folders,omitempty"`
	DeleteAfterForward bool     `json:"deleteAfterForward,omitempty"`
	TargetFolder       string   `json:"targetFolder,omitempty"`
}

// TargetConfig is the configuration for the target server (IMAP or SMTP).
type TargetConfig struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Secure *bool  `json:"secure,omitempty"`
	Auth   Auth   `json:"auth"`
	Folder string `json:"folder,omitempty"`
}

// HealthCheckConfig configures the HTTP health check endpoint.
type HealthCheckConfig struct {
	Port int `json:"port,omitempty"`
}

// GmailAPIConfig holds OAuth2 credentials for the Gmail API forwarding method.
type GmailAPIConfig struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RefreshToken string `json:"refreshToken"`
}

// Config is the top-level application configuration.
type Config struct {
	Target        TargetConfig      `json:"target"`
	ForwardMethod string            `json:"forwardMethod,omitempty"`
	GmailAPI      *GmailAPIConfig   `json:"gmailApi,omitempty"`
	Sources       []SourceConfig    `json:"sources"`
	HealthCheck   HealthCheckConfig `json:"healthCheck,omitempty"`
}

const defaultHealthCheckPort = 8080

func isImplicitTLS(port int) bool {
	return port == 465 || port == 993
}

func boolPtr(b bool) *bool {
	return &b
}

func validateConfig(c *Config) error {
	// Forward method
	if c.ForwardMethod == "" {
		c.ForwardMethod = "imap"
	}
	if c.ForwardMethod != "imap" && c.ForwardMethod != "smtp" && c.ForwardMethod != "gmail-api" {
		return fmt.Errorf("forwardMethod must be \"imap\", \"smtp\", or \"gmail-api\"")
	}

	// Target — gmail-api only needs auth.user
	if c.ForwardMethod == "gmail-api" {
		if c.Target.Auth.User == "" {
			return fmt.Errorf("target.auth.user must be a non-empty string")
		}
	} else {
		if c.Target.Host == "" {
			return fmt.Errorf("target.host must be a non-empty string")
		}
		if c.Target.Port <= 0 {
			return fmt.Errorf("target.port must be a positive integer")
		}
		if c.Target.Secure == nil {
			c.Target.Secure = boolPtr(isImplicitTLS(c.Target.Port))
		}
		if c.Target.Auth.User == "" {
			return fmt.Errorf("target.auth.user must be a non-empty string")
		}
		if c.Target.Auth.Pass == "" {
			return fmt.Errorf("target.auth.pass must be a non-empty string")
		}
	}
	if c.Target.Folder == "" {
		c.Target.Folder = "INBOX"
	}

	// Gmail API config
	if c.ForwardMethod == "gmail-api" {
		if c.GmailAPI == nil {
			return fmt.Errorf("gmailApi config is required when forwardMethod is \"gmail-api\"")
		}
		if c.GmailAPI.ClientID == "" {
			return fmt.Errorf("gmailApi.clientId must be a non-empty string")
		}
		if c.GmailAPI.ClientSecret == "" {
			return fmt.Errorf("gmailApi.clientSecret must be a non-empty string")
		}
		if c.GmailAPI.RefreshToken == "" {
			return fmt.Errorf("gmailApi.refreshToken must be a non-empty string")
		}
	}

	// Sources
	if len(c.Sources) == 0 {
		return fmt.Errorf("sources must be a non-empty array")
	}
	for i := range c.Sources {
		s := &c.Sources[i]
		if s.Name == "" {
			return fmt.Errorf("sources[%d].name must be a non-empty string", i)
		}
		if s.Host == "" {
			return fmt.Errorf("sources[%d].host must be a non-empty string", i)
		}
		if s.Port <= 0 {
			return fmt.Errorf("sources[%d].port must be a positive integer", i)
		}
		if s.Secure == nil {
			s.Secure = boolPtr(isImplicitTLS(s.Port))
		}
		if s.Auth.User == "" {
			return fmt.Errorf("sources[%d].auth.user must be a non-empty string", i)
		}
		if s.Auth.Pass == "" {
			return fmt.Errorf("sources[%d].auth.pass must be a non-empty string", i)
		}
		if len(s.Folders) == 0 {
			s.Folders = []string{"INBOX"}
		}
	}

	// Health check
	if c.HealthCheck.Port <= 0 {
		c.HealthCheck.Port = defaultHealthCheckPort
	}

	return nil
}

func loadConfig(path string) (*Config, error) {
	log := newLogger("config")
	log.Info("Loading configuration from %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	log.Info("Configuration loaded: %d source(s) configured", len(config.Sources))
	return &config, nil
}
