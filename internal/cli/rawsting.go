package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"vivarium/internal/common"
	"vivarium/internal/rawsting"
)

var (
	rawTarget      string
	rawRounds      int
	rawConcurrency int
	rawPort        int
)

func init() {
	rootCmd.AddCommand(rawstingCmd)

	// Wasp flags
	waspCmd.Flags().StringVarP(&rawTarget, "target", "t", "", "target IP address (required)")
	waspCmd.Flags().IntVarP(&rawPort, "port", "p", 80, "target port")
	waspCmd.Flags().IntVarP(&rawRounds, "rounds", "r", 10000, "number of SYN packets to send")
	waspCmd.Flags().IntVarP(&rawConcurrency, "concurrency", "c", 100, "number of concurrent workers")

	// Mantis flags
	mantisCmd.Flags().StringVarP(&rawTarget, "target", "t", "", "target IP address (required)")
	mantisCmd.Flags().IntVarP(&rawPort, "port", "p", 80, "target port")
	mantisCmd.Flags().IntVarP(&rawRounds, "rounds", "r", 10000, "number of RST packets to send")
	mantisCmd.Flags().IntVarP(&rawConcurrency, "concurrency", "c", 100, "number of concurrent workers")

	rawstingCmd.AddCommand(waspCmd)
	rawstingCmd.AddCommand(mantisCmd)
}

// rawstingCmd represents the rawsting command group
var rawstingCmd = &cobra.Command{
	Use:   "rawsting",
	Short: "Execute raw socket DoS attacks (requires admin)",
	Long: `Raw Sting attacks use low-level network techniques.
These require administrator/root privileges.

⚠️  WARNING: These attacks operate at the TCP/IP layer and may require
elevated permissions. Run as Administrator (Windows) or root (Linux/Mac).

Available raw stings:
  wasp    - SYN flood (quick, painful TCP SYN jabs)
  mantis  - RST injection (severs connections with precision)`,
}

// waspCmd represents the wasp SYN flood attack
var waspCmd = &cobra.Command{
	Use:   "wasp",
	Short: "SYN flood attack",
	Long: `Wasp delivers quick, painful TCP SYN jabs.
Floods the target with half-open connections to exhaust resources.

⚠️  May require administrator/root privileges.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if rawTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n🐝 WASP - Quick, painful SYN jabs...")
		fmt.Printf("   Target: %s:%d\n", rawTarget, rawPort)
		fmt.Printf("   Rounds: %d\n", rawRounds)
		fmt.Printf("   Concurrency: %d\n", rawConcurrency)
		fmt.Println()
		fmt.Println("   ⚠️  Note: True SYN flood requires raw sockets.")
		fmt.Println("   This uses rapid connection attempts to simulate behavior.")
		fmt.Println()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping attack...")
			cancel()
		}()

		opts := rawsting.AttackOpts{
			Rounds:      rawRounds,
			Concurrency: rawConcurrency,
			Port:        rawPort,
			Verbose:     IsVerbose(),
		}

		wasp := rawsting.NewWasp()
		result, err := wasp.Attack(ctx, rawTarget, opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Total packets:  %d\n", result.TotalPackets)
		fmt.Printf("   Successful:     %d\n", result.Successful)
		fmt.Printf("   Failed:         %d\n", result.Failed)
		fmt.Printf("   Duration:       %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Packets/sec:    %.2f\n", float64(result.TotalPackets)/result.Duration.Seconds())

		return nil
	},
}

// mantisCmd represents the mantis RST injection attack
var mantisCmd = &cobra.Command{
	Use:   "mantis",
	Short: "RST injection attack",
	Long: `Mantis severs connections with precision RST cuts.
Forces TCP RST packets to disrupt established connections.

⚠️  May require administrator/root privileges.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if rawTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n🦗 MANTIS - Precision RST cuts...")
		fmt.Printf("   Target: %s:%d\n", rawTarget, rawPort)
		fmt.Printf("   Rounds: %d\n", rawRounds)
		fmt.Printf("   Concurrency: %d\n", rawConcurrency)
		fmt.Println()
		fmt.Println("   ⚠️  Note: Uses SO_LINGER=0 to force RST on close.")
		fmt.Println()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping attack...")
			cancel()
		}()

		opts := rawsting.AttackOpts{
			Rounds:      rawRounds,
			Concurrency: rawConcurrency,
			Port:        rawPort,
			Verbose:     IsVerbose(),
		}

		mantis := rawsting.NewMantis()
		result, err := mantis.Attack(ctx, rawTarget, opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Total packets:  %d\n", result.TotalPackets)
		fmt.Printf("   Successful:     %d\n", result.Successful)
		fmt.Printf("   Failed:         %d\n", result.Failed)
		fmt.Printf("   Duration:       %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Packets/sec:    %.2f\n", float64(result.TotalPackets)/result.Duration.Seconds())

		return nil
	},
}
