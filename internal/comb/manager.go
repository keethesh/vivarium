// Package comb manages worker lists (open redirect URLs).
package comb

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Manager handles worker list operations.
type Manager struct {
	workers []string
}

// NewManager creates a new comb manager.
func NewManager() *Manager {
	return &Manager{
		workers: make([]string, 0),
	}
}

// LoadFromFile loads workers from a file.
func (m *Manager) LoadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m.workers = append(m.workers, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	return nil
}

// SaveToFile saves workers to a file.
func (m *Manager) SaveToFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, worker := range m.workers {
		fmt.Fprintln(writer, worker)
	}
	return writer.Flush()
}

// Add adds a worker to the list.
func (m *Manager) Add(worker string) {
	m.workers = append(m.workers, worker)
}

// AddUnique adds a worker only if it doesn't already exist.
func (m *Manager) AddUnique(worker string) bool {
	for _, w := range m.workers {
		if w == worker {
			return false
		}
	}
	m.workers = append(m.workers, worker)
	return true
}

// Workers returns all workers.
func (m *Manager) Workers() []string {
	return m.workers
}

// Count returns the number of workers.
func (m *Manager) Count() int {
	return len(m.workers)
}

// Clear removes all workers.
func (m *Manager) Clear() {
	m.workers = make([]string, 0)
}

// Merge adds workers from another manager, avoiding duplicates.
func (m *Manager) Merge(other *Manager) int {
	added := 0
	for _, worker := range other.workers {
		if m.AddUnique(worker) {
			added++
		}
	}
	return added
}

// SetWorkers replaces the worker list.
func (m *Manager) SetWorkers(workers []string) {
	m.workers = workers
}
