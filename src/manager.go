package main

import (
	"context"
	"sync"
)

// Manager manages multiple forwarder goroutines.
type Manager struct {
	config     *Config
	forwarders []*Forwarder
	statuses   sync.Map
	logger     *Logger
}

// NewManager creates a new Manager from the given config.
func NewManager(config *Config) *Manager {
	return &Manager{
		config: config,
		logger: newLogger("manager"),
	}
}

// StartAll starts all forwarders and blocks until ctx is cancelled.
func (m *Manager) StartAll(ctx context.Context) {
	m.logger.Info("Starting %d forwarder(s)...", len(m.config.Sources))

	var wg sync.WaitGroup
	for i := range m.config.Sources {
		source := m.config.Sources[i]

		var sender Sender

		switch m.config.ForwardMethod {
		case "smtp":
			sender = NewSMTPSender(m.config.Target)
		case "gmail-api":
			sender = NewGmailAPISender(*m.config.GmailAPI, m.config.Target.Auth.User)
		default:
			sender = NewIMAPSender(m.config.Target, DefaultIMAPDial)
		}

		fwd := NewForwarder(source, sender, func(status ForwarderStatus) {
			m.statuses.Store(status.Name, status)
		})
		m.forwarders = append(m.forwarders, fwd)

		wg.Add(1)
		go func() {
			defer wg.Done()
			fwd.Run(ctx)
			fwd.Stop()
		}()
	}

	wg.Wait()
	m.logger.Info("All forwarders stopped")
}

// GetStatuses returns the status of all forwarders.
func (m *Manager) GetStatuses() []ForwarderStatus {
	statuses := make([]ForwarderStatus, len(m.forwarders))
	for i, fwd := range m.forwarders {
		statuses[i] = fwd.GetStatus()
	}
	return statuses
}

// GetOverallStatus returns "ok", "degraded", or "error".
func (m *Manager) GetOverallStatus() string {
	statuses := m.GetStatuses()
	connected := 0
	for _, s := range statuses {
		if s.Connected {
			connected++
		}
	}
	if connected == len(statuses) {
		return "ok"
	}
	if connected > 0 {
		return "degraded"
	}
	return "error"
}
