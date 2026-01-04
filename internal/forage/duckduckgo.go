package forage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"vivarium/internal/common"
)

// DuckDuckGo implements SearchEngine for DuckDuckGo.
type DuckDuckGo struct {
	client *http.Client
}

// NewDuckDuckGo creates a new DuckDuckGo engine.
func NewDuckDuckGo(client *http.Client) *DuckDuckGo {
	if client == nil {
		client = common.DefaultHTTPClient()
	}
	return &DuckDuckGo{client: client}
}

// Name returns the engine name.
func (d *DuckDuckGo) Name() string {
	return "DuckDuckGo"
}

// Search performs a dork search using DuckDuckGo HTML interface.
func (d *DuckDuckGo) Search(ctx context.Context, dork string, maxResults int) ([]string, error) {
	// DuckDuckGo HTML search
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(dork))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return d.extractURLs(string(body)), nil
}

// extractURLs extracts result URLs from DuckDuckGo HTML.
func (d *DuckDuckGo) extractURLs(html string) []string {
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
