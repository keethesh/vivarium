package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"vivarium/internal/forage"
)

var (
	forageOutput     string
	forageDorksFile  string
	forageMaxPerDork int
	forageValidate   bool
	forageEngines    string
)

func init() {
	forageCmd.Run = nil // Clear the placeholder Run
	forageCmd.RunE = runForage

	forageCmd.Flags().StringVarP(&forageOutput, "output", "o", "", "output file for discovered URLs")
	forageCmd.Flags().StringVarP(&forageDorksFile, "dorks", "d", "", "file with custom dork patterns (optional)")
	forageCmd.Flags().IntVarP(&forageMaxPerDork, "max", "m", 20, "maximum results per dork")
	forageCmd.Flags().BoolVarP(&forageValidate, "validate", "V", false, "validate discovered URLs (slower)")
	forageCmd.Flags().StringVarP(&forageEngines, "engines", "e", "all", "comma-separated engines (google, bing, yahoo, ddg)")
}

func runForage(cmd *cobra.Command, args []string) error {
	fmt.Println("\n🐝 FORAGING - Scouting the web for open redirects...")

	// Load custom dorks if provided
	var dorks []string
	if forageDorksFile != "" {
		file, err := os.Open(forageDorksFile)
		if err != nil {
			return fmt.Errorf("failed to open dorks file: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				dorks = append(dorks, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read dorks file: %w", err)
		}
		fmt.Printf("   Loaded %d custom dorks from %s\n", len(dorks), forageDorksFile)
	} else {
		dorks = forage.DefaultDorks
		fmt.Printf("   Using %d default dorks\n", len(dorks))
	}

	var engines []string
	if forageEngines != "" && forageEngines != "all" {
		engines = strings.Split(forageEngines, ",")
	}
	if len(engines) > 0 {
		fmt.Printf("   Engines: %s\n", strings.Join(engines, ", "))
	} else {
		fmt.Printf("   Engines: all (Google, Bing, Yahoo, DuckDuckGo)\n")
	}

	fmt.Printf("   Max results per dork: %d\n\n", forageMaxPerDork)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Received interrupt, stopping foraging...")
		cancel()
	}()

	dorker := forage.NewDorker(engines)
	result, err := dorker.Search(ctx, dorks, forageMaxPerDork, IsVerbose())
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("foraging failed: %w", err)
	}

	fmt.Printf("\n📊 Foraging Results:\n")
	fmt.Printf("   Dorks searched:  %d\n", result.TotalDorks)
	fmt.Printf("   URLs found:      %d\n", result.TotalURLs)
	fmt.Printf("   Duration:        %s\n", result.Duration.Round(100*1e6))

	// Show dork breakdown if verbose
	if IsVerbose() {
		fmt.Println("\n   Results by Dork:")
		for _, dr := range result.DorkResults {
			if dr.Error != "" {
				fmt.Printf("     [%s] ✗ %s - Error: %s\n", dr.Engine, truncateDork(dr.Dork, 40), dr.Error)
			} else {
				count := len(dr.URLs)
				if count > 0 {
					fmt.Printf("     [%s] ✓ %s - %d URLs\n", dr.Engine, truncateDork(dr.Dork, 40), count)
				}
			}
		}
	}

	// Save to file if output specified
	if forageOutput != "" && len(result.UniqueURLs) > 0 {
		file, err := os.Create(forageOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		for _, u := range result.UniqueURLs {
			fmt.Fprintln(file, u)
		}
		fmt.Printf("\n   Saved %d URLs to %s\n", len(result.UniqueURLs), forageOutput)
	}

	// Show sample URLs
	if len(result.UniqueURLs) > 0 {
		fmt.Println("\n   🔗 Sample discovered URLs:")
		for i, u := range result.UniqueURLs {
			if i >= 5 {
				fmt.Printf("   ... and %d more\n", len(result.UniqueURLs)-5)
				break
			}
			fmt.Printf("      %s\n", truncateURL(u, 80))
		}
	}

	fmt.Println("\n   💡 Tip: Use 'vivarium comb validate' to test which URLs actually work as open redirects.")

	return nil
}

func truncateDork(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func truncateURL(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
