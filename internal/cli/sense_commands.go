package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"vivarium/internal/common"
	"vivarium/internal/sense"
)

var (
	senseTarget      string
	senseDepth       int
	senseTop         int
	senseConcurrency int
)

func init() {
	// Add subcommands to senseCmd
	senseCmd.AddCommand(senseScoutCmd)
}

var senseScoutCmd = &cobra.Command{
	Use:   "scout",
	Short: "Find the largest assets on a target",
	Long: `Scout analyzes a target to find the largest static assets.
Large assets (images, videos, downloads) make the best attack vectors
as they consume the most server resources per request.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequirePermission(HasPermission()); err != nil {
			return err
		}
		if senseTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n🔭 SCOUT - Finding the juiciest targets...")
		fmt.Printf("   Target: %s\n", senseTarget)
		fmt.Printf("   Depth: %d\n", senseDepth)
		fmt.Printf("   Concurrency: %d\n\n", senseConcurrency)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping reconnaissance...")
			cancel()
		}()

		scout := sense.NewScout()
		result, err := scout.Discover(ctx, senseTarget, senseDepth, senseConcurrency, IsVerbose())
		if err != nil {
			return fmt.Errorf("scout failed: %w", err)
		}

		fmt.Printf("\n📊 Scout Results:\n")
		fmt.Printf("   Assets found: %d\n", len(result.Assets))
		fmt.Printf("   Duration: %s\n\n", result.Duration.Round(100*1e6))

		// Show top assets
		fmt.Printf("   🎯 Top %d Largest Assets:\n", min(senseTop, len(result.Assets)))
		fmt.Println("   " + "─────────────────────────────────────────────────────────────────")
		fmt.Printf("   %-8s %-15s %s\n", "SIZE", "TYPE", "URL")
		fmt.Println("   " + "─────────────────────────────────────────────────────────────────")

		for i, asset := range result.Assets {
			if i >= senseTop {
				break
			}
			if asset.Size > 0 {
				contentType := asset.ContentType
				if len(contentType) > 15 {
					contentType = contentType[:12] + "..."
				}
				fmt.Printf("   %-8s %-15s %s\n", formatSize(asset.Size), contentType, asset.URL)
			}
		}

		fmt.Println("   " + "─────────────────────────────────────────────────────────────────")
		fmt.Println("\n   💡 Tip: Use these URLs as targets for your sting attacks to maximize resource consumption.")

		return nil
	},
}

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

func init() {
	senseScoutCmd.Flags().StringVarP(&senseTarget, "target", "t", "", "target URL to scout (required)")
	senseScoutCmd.Flags().IntVarP(&senseDepth, "depth", "d", 2, "crawl depth")
	senseScoutCmd.Flags().IntVarP(&senseTop, "top", "n", 10, "number of top assets to show")
	senseScoutCmd.Flags().IntVarP(&senseConcurrency, "concurrency", "c", 20, "number of concurrent requests")
}
