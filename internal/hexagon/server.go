// Package hexagon provides the web GUI for Vivarium.
package hexagon

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"
)

//go:embed static/*
var staticFS embed.FS

// Server is the Hexagon web server.
type Server struct {
	port       int
	httpServer *http.Server
	hub        *Hub
	attacks    *AttackManager
	mu         sync.RWMutex
}

// NewServer creates a new Hexagon server.
func NewServer(port int) *Server {
	s := &Server{
		port:    port,
		hub:     NewHub(),
		attacks: NewAttackManager(),
	}
	return s
}

// Start starts the Hexagon server.
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.hub.Run()

	// Set up routes
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/sting/locust", s.handleLocust)
	mux.HandleFunc("/api/sting/tick", s.handleTick)
	mux.HandleFunc("/api/sting/flyswarm", s.handleFlySwarm)
	mux.HandleFunc("/api/attack/stop", s.handleStop)
	mux.HandleFunc("/api/ws", s.handleWebSocket)

	// Static files
	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("failed to get static files: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticContent)))

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🔷 Hexagon starting on http://localhost:%d", s.port)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server.
func (s *Server) Stop(ctx context.Context) error {
	s.attacks.StopAll()
	return s.httpServer.Shutdown(ctx)
}

// corsMiddleware adds CORS headers for development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// handleStatus returns server status.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := map[string]interface{}{
		"version":       "0.3.0",
		"status":        "ready",
		"activeAttacks": s.attacks.ActiveCount(),
		"uptime":        time.Since(s.attacks.startTime).String(),
	}
	writeJSON(w, http.StatusOK, status)
}

// handleConfig returns current configuration.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	config := map[string]interface{}{
		"authorized": true, // GUI always requires authorization
		"port":       s.port,
	}
	writeJSON(w, http.StatusOK, config)
}
