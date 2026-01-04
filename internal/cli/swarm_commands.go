package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"vivarium/internal/comb"
	"vivarium/internal/common"
	"vivarium/internal/swarm"
)

var (
	swarmTarget      string
	swarmCombFile    string
	swarmRounds      int
	swarmConcurrency int
	swarmDelay       time.Duration
)

func init() {
	// Add subcommands to swarmCmd
	swarmCmd.AddCommand(swarmLocustCmd)
	swarmCmd.AddCommand(swarmPollenCmd)
	swarmCmd.AddCommand(swarmFireflyCmd)
	swarmCmd.AddCommand(swarmMoltCmd)

	// Swarm locust flags
	swarmLocustCmd.Flags().StringVarP(&swarmTarget, "target", "t", "", "target URL (required)")
	swarmLocustCmd.Flags().StringVarP(&swarmCombFile, "comb", "c", "", "comb file with worker URLs (required)")
	swarmLocustCmd.Flags().IntVarP(&swarmRounds, "rounds", "r", 100, "number of requests per worker")
	swarmLocustCmd.Flags().IntVarP(&swarmConcurrency, "concurrency", "n", 100, "number of concurrent goroutines")
	swarmLocustCmd.Flags().DurationVarP(&swarmDelay, "delay", "d", 0, "delay between requests")

	// Swarm pollen flags
	swarmPollenCmd.Flags().StringVarP(&swarmTarget, "target", "t", "", "target URL (required)")
	swarmPollenCmd.Flags().StringVarP(&swarmCombFile, "comb", "c", "", "comb file with worker URLs (required)")
	swarmPollenCmd.Flags().IntVarP(&swarmRounds, "rounds", "r", 100, "number of requests per worker")
	swarmPollenCmd.Flags().IntVarP(&swarmConcurrency, "concurrency", "n", 100, "number of concurrent goroutines")
	swarmPollenCmd.Flags().DurationVarP(&swarmDelay, "delay", "d", 0, "delay between requests")

	// Swarm firefly flags
	swarmFireflyCmd.Flags().StringVarP(&swarmTarget, "target", "t", "", "target URL (required)")
	swarmFireflyCmd.Flags().StringVarP(&swarmCombFile, "comb", "c", "", "comb file with worker URLs (required)")
	swarmFireflyCmd.Flags().IntVarP(&swarmRounds, "rounds", "r", 100, "number of requests per worker")
	swarmFireflyCmd.Flags().IntVarP(&swarmConcurrency, "concurrency", "n", 100, "number of concurrent goroutines")
	swarmFireflyCmd.Flags().DurationVarP(&swarmDelay, "delay", "d", 0, "delay between requests")

	// Swarm molt flags
	swarmMoltCmd.Flags().StringVarP(&swarmTarget, "target", "t", "", "target URL (required)")
	swarmMoltCmd.Flags().StringVarP(&swarmCombFile, "comb", "c", "", "comb file with worker URLs (required)")
	swarmMoltCmd.Flags().IntVarP(&swarmRounds, "rounds", "r", 100, "number of requests per worker")
	swarmMoltCmd.Flags().IntVarP(&swarmConcurrency, "concurrency", "n", 100, "number of concurrent goroutines")
	swarmMoltCmd.Flags().DurationVarP(&swarmDelay, "delay", "d", 0, "delay between requests")
}

// runSwarmAttack is a helper that runs a swarm attack with the given attack type
func runSwarmAttack(attackType swarm.AttackType, emoji, name string) error {
	if err := common.RequireAuthorization(GetConfigPath()); err != nil {
		return err
	}
	if swarmTarget == "" {
		return fmt.Errorf("--target is required")
	}
	if swarmCombFile == "" {
		return fmt.Errorf("--comb is required")
	}

	// Load workers from comb file
	manager := comb.NewManager()
	if err := manager.LoadFromFile(swarmCombFile); err != nil {
		return fmt.Errorf("failed to load comb file: %w", err)
	}

	if manager.Count() == 0 {
		return fmt.Errorf("comb file is empty")
	}

	fmt.Printf("\n%s SWARM %s - Mobilizing the colony...\n", emoji, name)
	fmt.Printf("   Target: %s\n", swarmTarget)
	fmt.Printf("   Workers: %d from %s\n", manager.Count(), swarmCombFile)
	fmt.Printf("   Rounds per worker: %d\n", swarmRounds)
	fmt.Printf("   Concurrency: %d\n", swarmConcurrency)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Received interrupt, recalling the swarm...")
		cancel()
	}()

	coordinator := swarm.NewCoordinator(manager.Workers())
	opts := swarm.AttackOpts{
		Rounds:      swarmRounds,
		Concurrency: swarmConcurrency,
		Delay:       swarmDelay,
		Verbose:     IsVerbose(),
		AttackType:  attackType,
	}

	result, err := coordinator.Attack(ctx, swarmTarget, opts)
	if err != nil {
		return fmt.Errorf("swarm attack failed: %w", err)
	}

	fmt.Println("\n📊 Swarm Attack Results:")
	fmt.Printf("   Attack type:     %s\n", name)
	fmt.Printf("   Workers used:    %d\n", result.WorkersUsed)
	fmt.Printf("   Total requests:  %d\n", result.TotalRequests)
	fmt.Printf("   Successful:      %d (%.1f%%)\n", result.Successful, float64(result.Successful)/float64(result.TotalRequests)*100)
	fmt.Printf("   Failed:          %d\n", result.Failed)
	fmt.Printf("   Duration:        %s\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("   Requests/sec:    %.2f\n", float64(result.TotalRequests)/result.Duration.Seconds())

	if IsVerbose() && len(result.WorkerStats) > 0 {
		fmt.Println("\n   Top Workers:")
		count := 0
		for _, stats := range result.WorkerStats {
			if count >= 5 {
				break
			}
			if stats.Successful > 0 {
				fmt.Printf("     ✓ %s - %d successful\n", stats.URL, stats.Successful)
				count++
			}
		}
	}

	return nil
}

var swarmLocustCmd = &cobra.Command{
	Use:   "locust",
	Short: "Distributed HTTP GET flood via open redirects",
	Long: `Swarm Locust uses open redirect URLs to send GET requests to the target.
Each worker redirects traffic to the target, masking the origin.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSwarmAttack(swarm.AttackLocust, "🦗", "LOCUST")
	},
}

var swarmPollenCmd = &cobra.Command{
	Use:   "pollen",
	Short: "Distributed Pollen Burst via open redirects",
	Long: `Swarm Pollen uses open redirect URLs to scatter oversized URI requests.
Each worker redirects with long path payloads to stress target buffers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSwarmAttack(swarm.AttackPollen, "🌸", "POLLEN")
	},
}

var swarmFireflyCmd = &cobra.Command{
	Use:   "firefly",
	Short: "Distributed Firefly via open redirects",
	Long: `Swarm Firefly uses open redirect URLs to send varied unusual headers.
Each worker redirects with different header patterns.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSwarmAttack(swarm.AttackFirefly, "🪲", "FIREFLY")
	},
}

var swarmMoltCmd = &cobra.Command{
	Use:   "molt",
	Short: "Distributed Molt via open redirects",
	Long: `Swarm Molt uses open redirect URLs to stress with varied HTTP methods.
Each worker redirects using rotating GET/POST/PUT/DELETE methods.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSwarmAttack(swarm.AttackMolt, "🦎", "MOLT")
	},
}
