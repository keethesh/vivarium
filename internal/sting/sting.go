// Package sting implements DoS attack vectors.
package sting

import (
	"context"
	"time"
)

// AttackOpts contains options for an attack.
type AttackOpts struct {
	// Common options
	Rounds      int           // Number of requests/packets to send
	Concurrency int           // Number of concurrent workers
	Delay       time.Duration // Delay between operations
	Verbose     bool          // Enable verbose output

	// Tick-specific
	Sockets int // Number of sockets to open (Slowloris)

	// FlySwarm-specific
	Port       int // Target port for UDP
	PacketSize int // Size of UDP packets
}

// Result contains the results of an attack.
type Result struct {
	TotalRequests int           // Total requests/packets attempted
	Successful    int           // Successful requests/packets
	Failed        int           // Failed requests/packets
	Duration      time.Duration // Total attack duration
}

// Sting is the interface for all DoS attack types.
type Sting interface {
	// Attack executes the attack against the target.
	Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error)
	// Name returns the name of the attack.
	Name() string
	// Description returns a brief description.
	Description() string
}
