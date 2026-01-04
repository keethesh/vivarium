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

// Yahoo implements SearchEngine for Yahoo.
type Yahoo struct {
	client *http.Client
}

// NewYahoo creates a new Yahoo engine.
func NewYahoo(client *http.Client) *Yahoo {
	if client == nil {
		client = common.DefaultHTTPClient()
	}
	return &Yahoo{client: client}
}

// Name returns the engine name.
func (y *Yahoo) Name() string {
	return "Yahoo"
}

// Search performs a dork search using Yahoo.
func (y *Yahoo) Search(ctx context.Context, dork string, maxResults int) ([]string, error) {
	searchURL := fmt.Sprintf("https://search.yahoo.com/search?p=%s&n=%d", url.QueryEscape(dork), maxResults+5)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return y.extractURLs(string(body)), nil
}

// extractURLs extracts result URLs from Yahoo HTML.
func (y *Yahoo) extractURLs(html string) []string {
	urls := make([]string, 0)

	// Yahoo redirects search results through r.search.yahoo.com
	// But sometimes direct links exist. Regex for generic links first.
	// Yahoo's structure is messy, let's look for hrefs that aren't yahoo
	hrefPattern := regexp.MustCompile(`href="(https?://[^"]+)"`)
	matches := hrefPattern.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			u := match[1]
			// Yahoo uses intermediate redirects like https://r.search.yahoo.com/_ylt=.../RU=.../RK=...
			// If we decode it, we might find the target. But often the raw href is usable or encoded.

			// Simple filter for now: exclude yahoo domains
			if !strings.Contains(u, "yahoo.com") && !strings.Contains(u, "yimg.com") {
				urls = append(urls, u)
			}
		}
	}

	return urls
}
