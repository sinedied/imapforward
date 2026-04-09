package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var version = "1.1.1"

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	authClientID := flag.String("auth-client-id", "", "Google OAuth2 client ID (for -auth flow)")
	authClientSecret := flag.String("auth-client-secret", "", "Google OAuth2 client secret (for -auth flow)")
	runAuth := flag.Bool("auth", false, "Run OAuth2 authorization flow to get a Gmail API refresh token")
	showVersion := flag.Bool("version", false, "Show version")
	showHelp := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if *runAuth {
		if *authClientID == "" || *authClientSecret == "" {
			fmt.Fprintln(os.Stderr, "Error: -auth-client-id and -auth-client-secret are required with -auth")
			os.Exit(1)
		}
		if err := RunOAuthFlow(*authClientID, *authClientSecret); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Log level: env var > flag > default
	lvl := *logLevel
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		lvl = env
	}
	setLogLevel(parseLogLevel(lvl))

	// Config path: env var > flag > default
	cfgPath := *configPath
	if env := os.Getenv("IMAPFORWARD_CONFIG"); env != "" {
		cfgPath = env
	}

	log := newLogger("cli")

	config, err := loadConfig(cfgPath)
	if err != nil {
		log.Error("Configuration error: %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(config)
	healthServer := StartHealthServer(manager, config.HealthCheck.Port)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		log.Info("Shutting down...")
		cancel()
		_ = healthServer.Close()
	}()

	log.Info("imapforward v%s starting (method: %s)", version, config.ForwardMethod)
	manager.StartAll(ctx)
	log.Info("Shutdown complete")
}

func printHelp() {
	fmt.Println(`imapforward - IMAP email forwarder

Usage: imapforward [options]

Options:
  -config <path>             Path to config file (default: config.json)
  -log-level <level>         Log level: debug, info, warn, error (default: info)
  -auth                      Run OAuth2 flow to obtain a Gmail API refresh token
  -auth-client-id <id>       Google OAuth2 client ID (required with -auth)
  -auth-client-secret <sec>  Google OAuth2 client secret (required with -auth)
  -version                   Show version
  -help                      Show this help`)
}
