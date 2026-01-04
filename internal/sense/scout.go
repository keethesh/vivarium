// Package sense implements reconnaissance tools.
package sense

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"vivarium/internal/common"
)

// Asset represents a discovered asset on the target.
type Asset struct {
	URL         string
	Size        int64
	ContentType string
	StatusCode  int
	Error       string
}

// ScoutResult contains the results of scouting a target.
type ScoutResult struct {
	Target   string
	Assets   []Asset
	Duration time.Duration
}

// Scout performs asset discovery on targets.
type Scout struct {
	client  *http.Client
	visited map[string]bool
	mu      sync.Mutex
}

// NewScout creates a new Scout instance.
func NewScout() *Scout {
	return &Scout{
		client:  common.DefaultHTTPClient(),
		visited: make(map[string]bool),
	}
}

// Discover finds assets on the target and ranks them by size.
func (s *Scout) Discover(ctx context.Context, target string, depth int, concurrency int, verbose bool) (*ScoutResult, error) {
	start := time.Now()

	if concurrency <= 0 {
		concurrency = 20
	}
	if depth <= 0 {
		depth = 1
	}

	// Parse target URL
	parsedTarget, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}
	baseHost := parsedTarget.Host

	// Start with the target URL
	s.visited[target] = true
	assets := make([]Asset, 0)
	var assetsMu sync.Mutex

	// URLs to process at each depth level
	toProcess := []string{target}

	for d := 0; d < depth; d++ {
		if verbose {
			fmt.Printf("   Depth %d: Processing %d URLs...\n", d+1, len(toProcess))
		}

		var wg sync.WaitGroup
		jobs := make(chan string, len(toProcess))
		foundURLs := make(chan string, 1000)

		// Start workers
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for targetURL := range jobs {
					select {
					case <-ctx.Done():
						return
					default:
					}

					asset, links := s.analyzeURL(ctx, targetURL, baseHost, verbose)
					if asset != nil {
						assetsMu.Lock()
						assets = append(assets, *asset)
						assetsMu.Unlock()
					}

					// Send discovered links
					for _, link := range links {
						select {
						case foundURLs <- link:
						default:
							// Channel full, skip
						}
					}
				}
			}()
		}

		// Send jobs
		for _, u := range toProcess {
			select {
			case <-ctx.Done():
				break
			case jobs <- u:
			}
		}
		close(jobs)

		// Wait for workers and collect new URLs
		go func() {
			wg.Wait()
			close(foundURLs)
		}()

		// Collect new URLs for next depth
		newURLs := make([]string, 0)
		for link := range foundURLs {
			s.mu.Lock()
			if !s.visited[link] {
				s.visited[link] = true
				newURLs = append(newURLs, link)
			}
			s.mu.Unlock()
		}

		toProcess = newURLs
		if len(toProcess) == 0 {
			break
		}
	}

	// Sort assets by size (largest first)
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Size > assets[j].Size
	})

	return &ScoutResult{
		Target:   target,
		Assets:   assets,
		Duration: time.Since(start),
	}, nil
}

// analyzeURL fetches a URL and extracts information about it.
func (s *Scout) analyzeURL(ctx context.Context, targetURL, baseHost string, verbose bool) (*Asset, []string) {
	links := make([]string, 0)

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return &Asset{URL: targetURL, Error: err.Error()}, nil
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())
	req.Header.Set("Accept", "*/*")

	resp, err := s.client.Do(req)
	if err != nil {
		return &Asset{URL: targetURL, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	asset := &Asset{
		URL:         targetURL,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
	}

	// Get size from Content-Length header
	asset.Size = resp.ContentLength

	// If Content-Length not available, read the body
	if asset.Size < 0 {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // Limit to 10MB
		if err == nil {
			asset.Size = int64(len(body))

			// Extract links from HTML
			if strings.Contains(asset.ContentType, "text/html") {
				links = s.extractLinks(string(body), targetURL, baseHost)
			}
		}
	} else if strings.Contains(asset.ContentType, "text/html") {
		// Read body for link extraction
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
		if err == nil {
			links = s.extractLinks(string(body), targetURL, baseHost)
		}
	}

	if verbose && asset.Size > 0 {
		fmt.Printf("      Found: %s (%s, %s)\n", truncateURL(targetURL, 60), formatSize(asset.Size), asset.ContentType)
	}

	return asset, links
}

// extractLinks extracts links from HTML content.
func (s *Scout) extractLinks(html, baseURL, baseHost string) []string {
	links := make([]string, 0)

	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return links
	}

	// Regex patterns for links
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`href=["']([^"']+)["']`),
		regexp.MustCompile(`src=["']([^"']+)["']`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}

			link := match[1]

			// Skip data URLs, javascript, mailto, etc.
			if strings.HasPrefix(link, "data:") ||
				strings.HasPrefix(link, "javascript:") ||
				strings.HasPrefix(link, "mailto:") ||
				strings.HasPrefix(link, "#") {
				continue
			}

			// Resolve relative URLs
			parsed, err := url.Parse(link)
			if err != nil {
				continue
			}

			resolved := baseParsed.ResolveReference(parsed)

			// Only include same-host URLs
			if resolved.Host == baseHost {
				links = append(links, resolved.String())
			}
		}
	}

	return links
}

// formatSize formats bytes as human-readable size.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// truncateURL truncates a URL to a maximum length.
func truncateURL(u string, maxLen int) string {
	if len(u) <= maxLen {
		return u
	}
	return u[:maxLen-3] + "..."
}
