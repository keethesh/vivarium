package hexagon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"vivarium/internal/sting"
)

// AttackRequest represents an incoming attack request.
type AttackRequest struct {
	Target      string `json:"target"`
	Rounds      int    `json:"rounds"`
	Concurrency int    `json:"concurrency"`
	Sockets     int    `json:"sockets"`
	Port        int    `json:"port"`
	PacketSize  int    `json:"packetSize"`
	Delay       string `json:"delay"`
}

// AttackManager manages running attacks.
type AttackManager struct {
	active    map[string]context.CancelFunc
	mu        sync.RWMutex
	startTime time.Time
}

// NewAttackManager creates a new attack manager.
func NewAttackManager() *AttackManager {
	return &AttackManager{
		active:    make(map[string]context.CancelFunc),
		startTime: time.Now(),
	}
}

// ActiveCount returns the number of active attacks.
func (m *AttackManager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.active)
}

// Add registers an active attack.
func (m *AttackManager) Add(id string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active[id] = cancel
}

// Remove removes an attack from active list.
func (m *AttackManager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, id)
}

// Stop stops a specific attack.
func (m *AttackManager) Stop(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel, ok := m.active[id]; ok {
		cancel()
		delete(m.active, id)
		return true
	}
	return false
}

// StopAll stops all running attacks.
func (m *AttackManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, cancel := range m.active {
		cancel()
		delete(m.active, id)
	}
}

// handleLocust handles Locust attack requests.
func (s *Server) handleLocust(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req AttackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	if req.Rounds <= 0 {
		req.Rounds = 1000
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 100
	}

	attackID := fmt.Sprintf("locust-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	s.attacks.Add(attackID, cancel)

	// Send initial response
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "started",
		"attackId": attackID,
		"type":     "locust",
		"target":   req.Target,
	})

	// Run attack in background
	go func() {
		defer s.attacks.Remove(attackID)

		s.hub.Broadcast(Message{
			Type: "log",
			Data: map[string]interface{}{
				"message":  fmt.Sprintf("🦗 LOCUST - Devouring %s...", req.Target),
				"attackId": attackID,
			},
		})

		opts := sting.AttackOpts{
			Rounds:      req.Rounds,
			Concurrency: req.Concurrency,
			Verbose:     true,
		}

		locust := sting.NewLocust()

		// Create a progress callback
		progressChan := make(chan sting.Progress, 100)
		go func() {
			for p := range progressChan {
				s.hub.Broadcast(Message{
					Type: "progress",
					Data: map[string]interface{}{
						"attackId":   attackID,
						"total":      p.Total,
						"completed":  p.Completed,
						"successful": p.Successful,
						"failed":     p.Failed,
						"rps":        p.RPS,
					},
				})
			}
		}()

		result, err := locust.AttackWithProgress(ctx, req.Target, opts, progressChan)
		close(progressChan)

		if err != nil {
			s.hub.Broadcast(Message{
				Type: "error",
				Data: map[string]interface{}{
					"attackId": attackID,
					"error":    err.Error(),
				},
			})
			return
		}

		s.hub.Broadcast(Message{
			Type: "complete",
			Data: map[string]interface{}{
				"attackId":      attackID,
				"totalRequests": result.TotalRequests,
				"successful":    result.Successful,
				"failed":        result.Failed,
				"duration":      result.Duration.String(),
				"rps":           float64(result.TotalRequests) / result.Duration.Seconds(),
			},
		})
	}()
}

// handleTick handles Tick (Slowloris) attack requests.
func (s *Server) handleTick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req AttackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	if req.Sockets <= 0 {
		req.Sockets = 200
	}

	delay := 15 * time.Second
	if req.Delay != "" {
		if d, err := time.ParseDuration(req.Delay); err == nil {
			delay = d
		}
	}

	attackID := fmt.Sprintf("tick-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	s.attacks.Add(attackID, cancel)

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "started",
		"attackId": attackID,
		"type":     "tick",
		"target":   req.Target,
	})

	go func() {
		defer s.attacks.Remove(attackID)

		s.hub.Broadcast(Message{
			Type: "log",
			Data: map[string]interface{}{
				"message":  fmt.Sprintf("🕷️ TICK - Latching onto %s...", req.Target),
				"attackId": attackID,
			},
		})

		opts := sting.AttackOpts{
			Sockets: req.Sockets,
			Delay:   delay,
			Verbose: true,
		}

		tick := sting.NewTick()
		result, err := tick.Attack(ctx, req.Target, opts)

		if err != nil {
			s.hub.Broadcast(Message{
				Type: "error",
				Data: map[string]interface{}{
					"attackId": attackID,
					"error":    err.Error(),
				},
			})
			return
		}

		s.hub.Broadcast(Message{
			Type: "complete",
			Data: map[string]interface{}{
				"attackId":    attackID,
				"connections": result.TotalRequests,
				"alive":       result.Successful,
				"dropped":     result.Failed,
				"duration":    result.Duration.String(),
			},
		})
	}()
}

// handleFlySwarm handles FlySwarm (UDP flood) attack requests.
func (s *Server) handleFlySwarm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req AttackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	if req.Rounds <= 0 {
		req.Rounds = 10000
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 100
	}
	if req.Port <= 0 {
		req.Port = 80
	}
	if req.PacketSize <= 0 {
		req.PacketSize = 1024
	}

	attackID := fmt.Sprintf("flyswarm-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	s.attacks.Add(attackID, cancel)

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "started",
		"attackId": attackID,
		"type":     "flyswarm",
		"target":   req.Target,
	})

	go func() {
		defer s.attacks.Remove(attackID)

		s.hub.Broadcast(Message{
			Type: "log",
			Data: map[string]interface{}{
				"message":  fmt.Sprintf("🪰 FLYSWARM - Bombarding %s:%d...", req.Target, req.Port),
				"attackId": attackID,
			},
		})

		opts := sting.AttackOpts{
			Rounds:      req.Rounds,
			Concurrency: req.Concurrency,
			Port:        req.Port,
			PacketSize:  req.PacketSize,
			Verbose:     true,
		}

		flyswarm := sting.NewFlySwarm()
		result, err := flyswarm.Attack(ctx, req.Target, opts)

		if err != nil {
			s.hub.Broadcast(Message{
				Type: "error",
				Data: map[string]interface{}{
					"attackId": attackID,
					"error":    err.Error(),
				},
			})
			return
		}

		s.hub.Broadcast(Message{
			Type: "complete",
			Data: map[string]interface{}{
				"attackId":    attackID,
				"packetsSent": result.TotalRequests,
				"successful":  result.Successful,
				"failed":      result.Failed,
				"duration":    result.Duration.String(),
				"pps":         float64(result.TotalRequests) / result.Duration.Seconds(),
			},
		})
	}()
}

// handleStop stops a running attack.
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		AttackID string `json:"attackId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Stop all if no specific ID
		s.attacks.StopAll()
		s.hub.Broadcast(Message{
			Type: "log",
			Data: map[string]interface{}{
				"message": "🛑 All attacks stopped",
			},
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "all stopped"})
		return
	}

	if s.attacks.Stop(req.AttackID) {
		s.hub.Broadcast(Message{
			Type: "log",
			Data: map[string]interface{}{
				"message":  fmt.Sprintf("🛑 Attack %s stopped", req.AttackID),
				"attackId": req.AttackID,
			},
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "attackId": req.AttackID})
	} else {
		writeError(w, http.StatusNotFound, "attack not found")
	}
}
