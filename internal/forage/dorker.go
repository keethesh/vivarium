// Package forage implements search engine dorking for open redirect discovery.
package forage

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
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

// DorkResult contains results for a single dork from a single engine.
type DorkResult struct {
	Dork   string
	Engine string
	URLs   []string
	Error  string
}

// SearchResult contains overall search results.
type SearchResult struct {
	TotalDorks  int
	TotalURLs   int
	UniqueURLs  []string
	DorkResults []DorkResult
	Duration    time.Duration
}

// Dorker searches for open redirect URLs using multiple search engines.
type Dorker struct {
	client  *http.Client
	engines []SearchEngine
}

// NewDorker creates a new Dorker instance.
// If engines is empty, defaults to all available engines.
func NewDorker(engines []string) *Dorker {
	client := common.DefaultHTTPClient()

	// Create engine instances
	var selectedEngines []SearchEngine

	if len(engines) == 0 {
		// Default to all
		selectedEngines = []SearchEngine{
			NewDuckDuckGo(client),
			NewGoogle(client),
			NewBing(client),
			NewYahoo(client),
		}
	} else {
		for _, name := range engines {
			switch strings.ToLower(name) {
			case "duckduckgo", "ddg":
				selectedEngines = append(selectedEngines, NewDuckDuckGo(client))
			case "google":
				selectedEngines = append(selectedEngines, NewGoogle(client))
			case "bing":
				selectedEngines = append(selectedEngines, NewBing(client))
			case "yahoo":
				selectedEngines = append(selectedEngines, NewYahoo(client))
			}
		}
	}

	return &Dorker{
		client:  client,
		engines: selectedEngines,
	}
}

// Search performs dork searches using configured engines.
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
		DorkResults: make([]DorkResult, 0, len(dorks)*len(d.engines)),
	}

	seenURLs := make(map[string]bool)
	var mu sync.Mutex

	// Process dorks sequentially to stay polite, but engines in parallel for each dork
	for i, dork := range dorks {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		if verbose {
			fmt.Printf("   [%d/%d] Searching: %s\n", i+1, len(dorks), dork)
		}

		// Search concurrently across engines for this dork
		var wg sync.WaitGroup
		engineResults := make(chan DorkResult, len(d.engines))

		for _, engine := range d.engines {
			wg.Add(1)
			go func(e SearchEngine) {
				defer wg.Done()

				// Add small random delay to stagger requests
				time.Sleep(time.Duration(common.RandomInt(100, 500)) * time.Millisecond)

				urls, err := e.Search(ctx, dork, maxPerDork)

				// Identify and filter for open redirects locally
				validURLs := make([]string, 0)
				for _, u := range urls {
					if LooksLikeRedirect(u) {
						validURLs = append(validURLs, u)
					}
				}

				res := DorkResult{
					Dork:   dork,
					Engine: e.Name(),
					URLs:   validURLs,
				}
				if err != nil {
					res.Error = err.Error()
				}
				engineResults <- res

				if verbose && len(validURLs) > 0 {
					fmt.Printf("      %s found %d URLs\n", e.Name(), len(validURLs))
				}
				// if verbose && err != nil {
				// 	fmt.Printf("      %s error: %v\n", e.Name(), err)
				// }
			}(engine)
		}

		wg.Wait()
		close(engineResults)

		// Collect results
		for res := range engineResults {
			mu.Lock()
			result.DorkResults = append(result.DorkResults, res)
			for _, u := range res.URLs {
				if !seenURLs[u] {
					seenURLs[u] = true
					result.UniqueURLs = append(result.UniqueURLs, u)
					result.TotalURLs++
				}
			}
			mu.Unlock()
		}

		// Sleep between dorks
		time.Sleep(2 * time.Second)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// LooksLikeRedirect checks if a URL contains redirect-related parameters.
func LooksLikeRedirect(u string) bool {
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
