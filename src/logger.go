package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	currentLevel LogLevel = LevelInfo
	levelMu      sync.RWMutex
)

func parseLogLevel(s string) LogLevel {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func setLogLevel(level LogLevel) {
	levelMu.Lock()
	defer levelMu.Unlock()
	currentLevel = level
}

func getLogLevel() LogLevel {
	levelMu.RLock()
	defer levelMu.RUnlock()
	return currentLevel
}

// Logger provides structured logging with a context prefix.
type Logger struct {
	context string
}

func newLogger(context string) *Logger {
	return &Logger{context: context}
}

func (l *Logger) log(level LogLevel, levelStr, msg string) {
	if getLogLevel() > level {
		return
	}
	prefix := ""
	if l.context != "" {
		prefix = fmt.Sprintf("[%s] ", l.context)
	}
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	fmt.Fprintf(os.Stderr, "%s %-5s %s%s\n", ts, levelStr, prefix, msg)
}

func (l *Logger) Debug(format string, args ...any) {
	l.log(LevelDebug, "DEBUG", fmt.Sprintf(format, args...))
}

func (l *Logger) Info(format string, args ...any) {
	l.log(LevelInfo, "INFO", fmt.Sprintf(format, args...))
}

func (l *Logger) Warn(format string, args ...any) {
	l.log(LevelWarn, "WARN", fmt.Sprintf(format, args...))
}

func (l *Logger) Error(format string, args ...any) {
	l.log(LevelError, "ERROR", fmt.Sprintf(format, args...))
}
