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

// Google implements SearchEngine for Google.
type Google struct {
	client *http.Client
}

// NewGoogle creates a new Google engine.
func NewGoogle(client *http.Client) *Google {
	if client == nil {
		client = common.DefaultHTTPClient()
	}
	return &Google{client: client}
}

// Name returns the engine name.
func (g *Google) Name() string {
	return "Google"
}

// Search performs a dork search using Google.
func (g *Google) Search(ctx context.Context, dork string, maxResults int) ([]string, error) {
	searchURL := fmt.Sprintf("https://www.google.com/search?q=%s&num=%d", url.QueryEscape(dork), maxResults+10)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	// Google helps prevent scraping if cookies are missing, but sometimes works.
	// We rotate User-Agents to help.

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (HTTP 429)")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return g.extractURLs(string(body)), nil
}

// extractURLs extracts result URLs from Google HTML.
func (g *Google) extractURLs(html string) []string {
	urls := make([]string, 0)

	// Google's result links usually look like /url?q=... or direct hrefs depending on client
	// Pattern: href="/url?q=https://example.com/..."
	urlPattern := regexp.MustCompile(`href="/url\?q=([^&"]+)`)
	matches := urlPattern.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			decodedURL, err := url.QueryUnescape(match[1])
			if err == nil && strings.HasPrefix(decodedURL, "http") && !strings.Contains(decodedURL, "google.com") {
				urls = append(urls, decodedURL)
			}
		}
	}

	return urls
}
