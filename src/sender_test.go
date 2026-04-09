package main

import (
	"testing"
)

func TestEnsureReplyTo_AddsReplyTo(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\n\r\nBody here")

	result := ensureReplyTo(raw, nil)

	expected := "Reply-To: sender@example.com\r\n"
	if string(result[:len(expected)]) != expected {
		t.Errorf("expected Reply-To header prepended, got: %q", string(result[:60]))
	}
}

func TestEnsureReplyTo_PreservesExisting(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nReply-To: other@example.com\r\nSubject: Test\r\n\r\nBody")

	result := ensureReplyTo(raw, nil)

	if string(result) != string(raw) {
		t.Error("expected message to be unchanged when Reply-To already exists")
	}
}

func TestEnsureReplyTo_NoFrom(t *testing.T) {
	raw := []byte("To: recipient@example.com\r\nSubject: Test\r\n\r\nBody")

	result := ensureReplyTo(raw, nil)

	if string(result) != string(raw) {
		t.Error("expected message unchanged when no From header")
	}
}

func TestEnsureReplyTo_InvalidMessage(t *testing.T) {
	raw := []byte("not a valid email")

	result := ensureReplyTo(raw, nil)

	if string(result) != string(raw) {
		t.Error("expected message unchanged for invalid input")
	}
}

func TestEnsureReplyTo_LFLineEndings(t *testing.T) {
	raw := []byte("From: sender@example.com\nTo: recipient@example.com\nSubject: Test\n\nBody")

	result := ensureReplyTo(raw, nil)

	expected := "Reply-To: sender@example.com\n"
	if string(result[:len(expected)]) != expected {
		t.Errorf("expected LF line ending in Reply-To, got: %q", string(result[:50]))
	}
}

func TestEnsureReplyTo_ComplexFrom(t *testing.T) {
	raw := []byte("From: \"John Doe\" <john@example.com>\r\nSubject: Test\r\n\r\nBody")

	result := ensureReplyTo(raw, nil)

	expected := "Reply-To: \"John Doe\" <john@example.com>\r\n"
	if string(result[:len(expected)]) != expected {
		t.Errorf("expected full From value in Reply-To, got: %q", string(result[:60]))
	}
}

func TestEnsureReplyTo_EmptyFrom(t *testing.T) {
	raw := []byte("From: \r\nTo: recipient@example.com\r\nSubject: Test\r\n\r\nBody")

	result := ensureReplyTo(raw, nil)

	if string(result) != string(raw) {
		t.Error("expected message unchanged with empty From")
	}
}

func TestDetectLineEnding_CRLF(t *testing.T) {
	data := []byte("Header: value\r\nOther: value\r\n\r\nBody")
	if nl := detectLineEnding(data); nl != "\r\n" {
		t.Errorf("expected \\r\\n, got %q", nl)
	}
}

func TestDetectLineEnding_LF(t *testing.T) {
	data := []byte("Header: value\nOther: value\n\nBody")
	if nl := detectLineEnding(data); nl != "\n" {
		t.Errorf("expected \\n, got %q", nl)
	}
}
