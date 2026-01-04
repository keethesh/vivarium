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

// Bing implements SearchEngine for Bing.
type Bing struct {
	client *http.Client
}

// NewBing creates a new Bing engine.
func NewBing(client *http.Client) *Bing {
	if client == nil {
		client = common.DefaultHTTPClient()
	}
	return &Bing{client: client}
}

// Name returns the engine name.
func (b *Bing) Name() string {
	return "Bing"
}

// Search performs a dork search using Bing.
func (b *Bing) Search(ctx context.Context, dork string, maxResults int) ([]string, error) {
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", url.QueryEscape(dork), maxResults+5)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b.extractURLs(string(body)), nil
}

// extractURLs extracts result URLs from Bing HTML.
func (b *Bing) extractURLs(html string) []string {
	urls := make([]string, 0)

	// Bing results are typically in <li class="b_algo"><h2><a href="...">
	// We'll just regex for http links in standard anchors
	hrefPattern := regexp.MustCompile(`href="(https?://[^"]+)"`)
	matches := hrefPattern.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			u := match[1]
			// Filter out Bing's own links and common junk
			if !strings.Contains(u, "bing.com") && !strings.Contains(u, "microsoft.com") {
				urls = append(urls, u)
			}
		}
	}

	return urls
}
