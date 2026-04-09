package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"

	"github.com/emersion/go-imap/v2"
)

// Sender is the interface for forwarding email messages to a target.
type Sender interface {
	Send(ctx context.Context, rawMessage []byte, targetFolder string) error
	Close() error
}

// IMAPSender forwards messages by appending them to a target IMAP mailbox.
type IMAPSender struct {
	target         TargetConfig
	logger         *Logger
	dial           IMAPDialFunc
	client         IMAPClient
	ensuredFolders map[string]bool
}

// NewIMAPSender creates a new IMAP append sender.
func NewIMAPSender(target TargetConfig, dial IMAPDialFunc) *IMAPSender {
	return &IMAPSender{
		target:         target,
		logger:         newLogger("imap-sender"),
		dial:           dial,
		ensuredFolders: make(map[string]bool),
	}
}

func (s *IMAPSender) Send(ctx context.Context, rawMessage []byte, targetFolder string) error {
	c, err := s.getClient()
	if err != nil {
		return fmt.Errorf("connect to target: %w", err)
	}

	folder := targetFolder
	if folder == "" {
		folder = s.target.Folder
	}

	if err := s.ensureFolder(c, folder); err != nil {
		return fmt.Errorf("ensure target folder %q: %w", folder, err)
	}

	appendCmd := c.Append(folder, int64(len(rawMessage)), nil)
	if _, err := appendCmd.Write(rawMessage); err != nil {
		return fmt.Errorf("write append data: %w", err)
	}
	if err := appendCmd.Close(); err != nil {
		return fmt.Errorf("close append: %w", err)
	}
	if _, err := appendCmd.Wait(); err != nil {
		return fmt.Errorf("append wait: %w", err)
	}

	return nil
}

func (s *IMAPSender) Close() error {
	if s.client != nil {
		err := s.client.Logout().Wait()
		s.client = nil
		return err
	}
	return nil
}

func (s *IMAPSender) getClient() (IMAPClient, error) {
	if s.client != nil {
		select {
		case <-s.client.Closed():
			s.logger.Warn("Target connection lost, reconnecting")
			s.client = nil
			s.ensuredFolders = make(map[string]bool)
		default:
			return s.client, nil
		}
	}

	c, err := s.dial(s.target.Host, s.target.Port, s.target.Secure)
	if err != nil {
		return nil, err
	}

	if err := c.Login(s.target.Auth.User, s.target.Auth.Pass).Wait(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("target login: %w", err)
	}

	s.client = c
	s.ensuredFolders = make(map[string]bool)
	s.logger.Info("Connected to target %s:%d", s.target.Host, s.target.Port)
	return c, nil
}

func (s *IMAPSender) ensureFolder(c IMAPClient, folder string) error {
	if folder == "INBOX" || s.ensuredFolders[folder] {
		return nil
	}

	if err := c.Create(folder, nil).Wait(); err != nil {
		// ALREADYEXISTS is expected — the folder may already exist
		// go-imap returns an *imap.Error with the ALREADYEXISTS response code
		if imapErr, ok := err.(*imap.Error); ok && imapErr.Code == imap.ResponseCodeAlreadyExists {
			s.logger.Debug("Target folder %q already exists", folder)
		} else {
			return err
		}
	} else {
		s.logger.Info("Created target folder: %s", folder)
	}

	s.ensuredFolders[folder] = true
	return nil
}

// SMTPSender forwards messages via SMTP with header preservation.
type SMTPSender struct {
	target TargetConfig
	logger *Logger
}

// NewSMTPSender creates a new SMTP forwarding sender.
func NewSMTPSender(target TargetConfig) *SMTPSender {
	return &SMTPSender{
		target: target,
		logger: newLogger("smtp-sender"),
	}
}

func (s *SMTPSender) Send(ctx context.Context, rawMessage []byte, targetFolder string) error {
	modified := ensureReplyTo(rawMessage, s.logger)

	addr := fmt.Sprintf("%s:%d", s.target.Host, s.target.Port)
	auth := smtp.PlainAuth("", s.target.Auth.User, s.target.Auth.Pass, s.target.Host)

	if s.target.Port == 465 {
		return s.sendImplicitTLS(addr, auth, modified)
	}

	return smtp.SendMail(addr, auth, s.target.Auth.User, []string{s.target.Auth.User}, modified)
}

func (s *SMTPSender) Close() error {
	return nil
}

func (s *SMTPSender) sendImplicitTLS(addr string, auth smtp.Auth, msg []byte) error {
	tlsConfig := &tls.Config{ServerName: s.target.Host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.target.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth: %w", err)
	}
	if err := client.Mail(s.target.Auth.User); err != nil {
		return fmt.Errorf("SMTP MAIL: %w", err)
	}
	if err := client.Rcpt(s.target.Auth.User); err != nil {
		return fmt.Errorf("SMTP RCPT: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		return fmt.Errorf("SMTP write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("SMTP data close: %w", err)
	}

	return client.Quit()
}

// ensureReplyTo adds a Reply-To header pointing to the original From address
// if one is not already present. This preserves reply/reply-all functionality
// when forwarding via SMTP.
func ensureReplyTo(rawMsg []byte, logger *Logger) []byte {
	msg, err := mail.ReadMessage(bytes.NewReader(rawMsg))
	if err != nil {
		if logger != nil {
			logger.Debug("Failed to parse message for Reply-To injection: %v", err)
		}
		return rawMsg
	}

	if msg.Header.Get("Reply-To") != "" {
		return rawMsg
	}

	from := msg.Header.Get("From")
	if from == "" {
		return rawMsg
	}

	nl := detectLineEnding(rawMsg)
	replyToHeader := "Reply-To: " + from + nl
	result := make([]byte, 0, len(replyToHeader)+len(rawMsg))
	result = append(result, []byte(replyToHeader)...)
	result = append(result, rawMsg...)
	return result
}

func detectLineEnding(data []byte) string {
	if idx := bytes.Index(data, []byte("\r\n")); idx >= 0 {
		return "\r\n"
	}
	return "\n"
}
