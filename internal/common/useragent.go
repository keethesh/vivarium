package common

import (
	_ "embed"
	"math/rand"
	"strings"
)

//go:embed user-agents.txt
var userAgentsData string

var userAgents []string

func init() {
	lines := strings.Split(userAgentsData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			userAgents = append(userAgents, line)
		}
	}
	// Fallback if no user agents loaded
	if len(userAgents) == 0 {
		userAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		}
	}
}

// RandomUserAgent returns a random user agent string.
func RandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// UserAgentCount returns the number of available user agents.
func UserAgentCount() int {
	return len(userAgents)
}
