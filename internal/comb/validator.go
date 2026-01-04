package comb

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"vivarium/internal/common"
)

// ValidationResult contains the result of validating workers.
type ValidationResult struct {
	Total    int
	Valid    int
	Invalid  int
	Duration time.Duration
}

// ValidatedWorker represents a worker with its validation status.
type ValidatedWorker struct {
	URL        string
	Valid      bool
	Error      string
	RedirectTo string
}

// Validator validates open redirect URLs.
type Validator struct {
	client      *http.Client
	concurrency int
	verbose     bool
}

// NewValidator creates a new validator.
func NewValidator(concurrency int, verbose bool) *Validator {
	if concurrency <= 0 {
		concurrency = 50
	}

	// Custom client that doesn't follow redirects
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Validator{
		client:      client,
		concurrency: concurrency,
		verbose:     verbose,
	}
}

// ValidateWorkers validates a list of workers concurrently.
func (v *Validator) ValidateWorkers(ctx context.Context, workers []string, testTarget string) ([]ValidatedWorker, *ValidationResult) {
	start := time.Now()

	var (
		valid   atomic.Int64
		invalid atomic.Int64
	)

	results := make([]ValidatedWorker, len(workers))
	jobs := make(chan int, len(workers))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < v.concurrency; i++ {
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
					worker := workers[idx]
					vw := v.validateWorker(ctx, worker, testTarget)
					results[idx] = vw

					if vw.Valid {
						valid.Add(1)
					} else {
						invalid.Add(1)
					}

					if v.verbose {
						current := valid.Load() + invalid.Load()
						if vw.Valid {
							fmt.Printf("\r   Progress: %d/%d - ✓ Valid: %s", current, len(workers), worker)
						} else {
							fmt.Printf("\r   Progress: %d/%d - ✗ Invalid: %s", current, len(workers), worker)
						}
					}
				}
			}
		}()
	}

	// Send jobs
	for i := range workers {
		select {
		case <-ctx.Done():
			break
		case jobs <- i:
		}
	}
	close(jobs)

	wg.Wait()

	if v.verbose {
		fmt.Println()
	}

	return results, &ValidationResult{
		Total:    len(workers),
		Valid:    int(valid.Load()),
		Invalid:  int(invalid.Load()),
		Duration: time.Since(start),
	}
}

// validateWorker validates a single worker URL.
func (v *Validator) validateWorker(ctx context.Context, workerURL, testTarget string) ValidatedWorker {
	result := ValidatedWorker{
		URL:   workerURL,
		Valid: false,
	}

	// Construct the redirect URL
	redirectURL, err := buildRedirectURL(workerURL, testTarget)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, "HEAD", redirectURL, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())

	resp, err := v.client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	// Check if it's a redirect
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location != "" {
			result.RedirectTo = location
			// Check if it redirects to our test target
			if strings.Contains(location, testTarget) || location == testTarget {
				result.Valid = true
			} else {
				result.Error = fmt.Sprintf("redirects to %s, not target", location)
			}
		} else {
			result.Error = "no Location header in redirect"
		}
	} else if resp.StatusCode == 200 {
		// Some open redirects return 200 with a meta refresh or JS redirect
		// For simplicity, we'll mark these as needing manual verification
		result.Error = "returned 200 (may need manual verification)"
	} else {
		result.Error = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
	}

	return result
}

// buildRedirectURL constructs a URL that should trigger an open redirect.
func buildRedirectURL(workerURL, target string) (string, error) {
	parsed, err := url.Parse(workerURL)
	if err != nil {
		return "", err
	}

	// Check if the URL already has a query parameter that looks like a redirect
	// Common patterns: url=, redirect=, redir=, next=, dest=, destination=, return=, returnUrl=
	q := parsed.Query()

	// Try to find an existing redirect parameter
	redirectParams := []string{"url", "redirect", "redir", "next", "dest", "destination", "return", "returnUrl", "returnurl", "goto", "link", "target", "to"}

	foundParam := ""
	for _, param := range redirectParams {
		if q.Has(param) {
			foundParam = param
			break
		}
	}

	if foundParam != "" {
		// Replace the existing parameter value with our target
		q.Set(foundParam, target)
	} else {
		// If no redirect param found, try adding url= at the end
		q.Set("url", target)
	}

	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

// ExtractValidWorkers returns only the valid workers from a validation result.
func ExtractValidWorkers(results []ValidatedWorker) []string {
	valid := make([]string, 0)
	for _, r := range results {
		if r.Valid {
			valid = append(valid, r.URL)
		}
	}
	return valid
}
