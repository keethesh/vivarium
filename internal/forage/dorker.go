// Package forage implements search engine dorking for open redirect discovery.
package forage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"vivarium/internal/common"
)

// DefaultDorks contains common open redirect dork patterns.
var DefaultDorks = []string{
	`inurl:"redirect?url="`,
	`inurl:"redirect_uri="`,
	`inurl:"redir?url="`,
	`inurl:"return?url="`,
	`inurl:"returnUrl="`,
	`inurl:"next="`,
	`inurl:"goto="`,
	`inurl:"dest="`,
	`inurl:"destination="`,
	`inurl:"out?url="`,
	`inurl:"checkout_url="`,
	`inurl:"continue="`,
	`inurl:"link?url="`,
	`inurl:"image_url="`,
	`inurl:"redirect_to="`,
}

// DorkResult contains results from a dork search.
type DorkResult struct {
	Dork  string
	URLs  []string
	Error string
}

// SearchResult contains overall search results.
type SearchResult struct {
	TotalDorks  int
	TotalURLs   int
	UniqueURLs  []string
	DorkResults []DorkResult
	Duration    time.Duration
}

// Dorker searches for open redirect URLs using search engines.
type Dorker struct {
	client *http.Client
}

// NewDorker creates a new Dorker instance.
func NewDorker() *Dorker {
	return &Dorker{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search performs dork searches using DuckDuckGo.
func (d *Dorker) Search(ctx context.Context, dorks []string, maxPerDork int, verbose bool) (*SearchResult, error) {
	start := time.Now()

	if len(dorks) == 0 {
		dorks = DefaultDorks
	}
	if maxPerDork <= 0 {
		maxPerDork = 20
	}

	result := &SearchResult{
		TotalDorks:  len(dorks),
		DorkResults: make([]DorkResult, 0, len(dorks)),
	}

	seenURLs := make(map[string]bool)

	for i, dork := range dorks {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		if verbose {
			fmt.Printf("   [%d/%d] Searching: %s\n", i+1, len(dorks), dork)
		}

		dorkResult := d.searchDork(ctx, dork, maxPerDork, verbose)
		result.DorkResults = append(result.DorkResults, dorkResult)

		// Collect unique URLs
		for _, u := range dorkResult.URLs {
			if !seenURLs[u] {
				seenURLs[u] = true
				result.UniqueURLs = append(result.UniqueURLs, u)
				result.TotalURLs++
			}
		}

		// Rate limit between searches
		time.Sleep(2 * time.Second)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// searchDork performs a single dork search using DuckDuckGo's HTML interface.
func (d *Dorker) searchDork(ctx context.Context, dork string, maxResults int, verbose bool) DorkResult {
	result := DorkResult{
		Dork: dork,
		URLs: make([]string, 0),
	}

	// DuckDuckGo HTML search
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(dork))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := d.client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Extract URLs from DuckDuckGo results
	urls := d.extractURLsFromDDG(string(body))

	// Filter and clean URLs
	for _, u := range urls {
		if len(result.URLs) >= maxResults {
			break
		}

		// Only include URLs that look like open redirects
		if d.looksLikeRedirect(u) {
			result.URLs = append(result.URLs, u)
			if verbose {
				fmt.Printf("      Found: %s\n", truncateString(u, 80))
			}
		}
	}

	return result
}

// extractURLsFromDDG extracts result URLs from DuckDuckGo HTML.
func (d *Dorker) extractURLsFromDDG(html string) []string {
	urls := make([]string, 0)

	// DuckDuckGo uses uddg parameter for actual URLs
	// Pattern: uddg=https%3A%2F%2F...
	uddgPattern := regexp.MustCompile(`uddg=([^&"]+)`)
	matches := uddgPattern.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			decodedURL, err := url.QueryUnescape(match[1])
			if err == nil && strings.HasPrefix(decodedURL, "http") {
				urls = append(urls, decodedURL)
			}
		}
	}

	// Also try to find direct href links
	hrefPattern := regexp.MustCompile(`href="(https?://[^"]+)"`)
	hrefMatches := hrefPattern.FindAllStringSubmatch(html, -1)

	for _, match := range hrefMatches {
		if len(match) >= 2 {
			u := match[1]
			// Skip DuckDuckGo internal links
			if !strings.Contains(u, "duckduckgo.com") {
				urls = append(urls, u)
			}
		}
	}

	return urls
}

// looksLikeRedirect checks if a URL contains redirect-related parameters.
func (d *Dorker) looksLikeRedirect(u string) bool {
	lower := strings.ToLower(u)
	redirectParams := []string{
		"redirect", "redir", "url=", "next=", "goto=", "dest=",
		"destination=", "return", "continue=", "link=", "out=",
		"checkout", "image_url=", "target=", "to=",
	}

	for _, param := range redirectParams {
		if strings.Contains(lower, param) {
			return true
		}
	}
	return false
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// DuckDuckGo API response structure (for future use)
type ddgAPIResponse struct {
	Results []struct {
		URL   string `json:"u"`
		Title string `json:"t"`
	} `json:"results"`
}

// parseJSONResponse parses DuckDuckGo API JSON response.
func parseJSONResponse(data []byte) ([]string, error) {
	var resp ddgAPIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		urls = append(urls, r.URL)
	}
	return urls, nil
}
