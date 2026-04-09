package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

// HealthResponse is the JSON payload returned by the health endpoint.
type HealthResponse struct {
	Status  string            `json:"status"`
	Sources []ForwarderStatus `json:"sources"`
}

// StartHealthServer creates and starts the HTTP health check server.
func StartHealthServer(manager *Manager, port int) *http.Server {
	log := newLogger("health")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		status := manager.GetOverallStatus()
		statuses := manager.GetStatuses()

		statusCode := http.StatusOK
		if status == "error" {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:  status,
			Sources: statuses,
		})
	})

	// Catch-all for other routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	})

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("Failed to start health server: %v", err)
		return server
	}

	log.Info("Health check server listening on port %d", port)
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Error("Health server error: %v", err)
		}
	}()

	return server
}
