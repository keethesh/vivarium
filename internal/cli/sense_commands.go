package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"vivarium/internal/common"
	"vivarium/internal/sense"
)

var (
	senseTarget      string
	senseDepth       int
	senseTop         int
	senseConcurrency int
	sensePorts       string
	senseTimeout     time.Duration
)

func init() {
	// Add subcommands to senseCmd
	senseCmd.AddCommand(senseScoutCmd)
	senseCmd.AddCommand(senseEyeCmd)
	senseCmd.AddCommand(senseAntennaCmd)

	// Scout flags
	senseScoutCmd.Flags().StringVarP(&senseTarget, "target", "t", "", "target URL to scout (required)")
	senseScoutCmd.Flags().IntVarP(&senseDepth, "depth", "d", 2, "crawl depth")
	senseScoutCmd.Flags().IntVarP(&senseTop, "top", "n", 10, "number of top assets to show")
	senseScoutCmd.Flags().IntVarP(&senseConcurrency, "concurrency", "c", 20, "number of concurrent requests")

	// Eye flags
	senseEyeCmd.Flags().StringVarP(&senseTarget, "target", "t", "", "target host to scan (required)")
	senseEyeCmd.Flags().StringVarP(&sensePorts, "ports", "p", "", "comma-separated ports (default: common ports)")
	senseEyeCmd.Flags().DurationVar(&senseTimeout, "timeout", time.Second, "connection timeout")

	// Antenna flags
	senseAntennaCmd.Flags().StringVarP(&senseTarget, "target", "t", "", "target URL to scan (required)")
}

var senseScoutCmd = &cobra.Command{
	Use:   "scout",
	Short: "Find the largest assets on a target",
	Long: `Scout analyzes a target to find the largest static assets.
Large assets (images, videos, downloads) make the best attack vectors
as they consume the most server resources per request.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
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

var senseEyeCmd = &cobra.Command{
	Use:   "eye",
	Short: "Compound Eye - Port scanner",
	Long: `Compound Eye scans the target for open ports.
It sees all open doors simultaneously, revealing potential entry points.
Default scans common ports (21, 22, 80, 443, 3306, 8080, etc).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if senseTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n👁️  COMPOUND EYE - Scanning ports...")
		fmt.Printf("   Target: %s\n", senseTarget)

		var ports []int
		if sensePorts != "" {
			parts := strings.Split(sensePorts, ",")
			for _, p := range parts {
				var port int
				fmt.Sscanf(strings.TrimSpace(p), "%d", &port)
				if port > 0 {
					ports = append(ports, port)
				}
			}
			fmt.Printf("   Ports: %d custom\n", len(ports))
		} else {
			fmt.Printf("   Ports: common list (%d)\n", len(sense.CommonPorts))
		}
		fmt.Println()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping scan...")
			cancel()
		}()

		eye := sense.NewCompoundEye(ports)
		openPorts, err := eye.Scan(ctx, senseTarget, senseTimeout)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		fmt.Printf("📊 Scan Results:\n")
		if len(openPorts) == 0 {
			fmt.Println("   No open ports found (or host is down/filtered)")
		} else {
			for _, p := range openPorts {
				fmt.Printf("   ✓ Port %d  (OPEN)\n", p)
			}
		}
		fmt.Println()

		return nil
	},
}

var senseAntennaCmd = &cobra.Command{
	Use:   "antenna",
	Short: "Antenna - WAF and technology detector",
	Long: `Antenna detects Wide Area Defenses (WAFs) and server technologies.
It senses the electromagnetic signature of the target's defense grid.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if senseTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n📡 ANTENNA - Detecting defenses...")
		fmt.Printf("   Target: %s\n", senseTarget)
		fmt.Println()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		antenna := sense.NewAntenna()
		info, err := antenna.Detect(ctx, senseTarget)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		fmt.Printf("📊 Detection Results:\n")
		if info.WAF != "" {
			fmt.Printf("   🛡️  WAF Detected: %s\n", info.WAF)
		} else {
			fmt.Printf("   🛡️  WAF: None detected (or unknown)\n")
		}

		if info.Server != "" {
			fmt.Printf("   💻 Server: %s\n", info.Server)
		}
		if info.PoweredBy != "" {
			fmt.Printf("   ⚡ Powered By: %s\n", info.PoweredBy)
		}

		if len(info.Cookies) > 0 {
			fmt.Printf("   cookies: %s\n", strings.Join(info.Cookies, ", "))
		}
		fmt.Println()

		return nil
	},
}
