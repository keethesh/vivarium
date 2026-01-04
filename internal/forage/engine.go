package forage

import (
	"context"
)

// SearchEngine represents a search engine dork implementation.
type SearchEngine interface {
	// Name returns the engine name (e.g., "DuckDuckGo", "Google").
	Name() string

	// Search performs a dork search and returns results.
	Search(ctx context.Context, dork string, maxResults int) ([]string, error)
}
