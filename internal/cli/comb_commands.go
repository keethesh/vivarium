package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"vivarium/internal/comb"
)

var (
	combInputFile   string
	combOutputFile  string
	combConcurrency int
	combTestTarget  string
)

// combCmd is defined in comb.go placeholder - we need to redefine it here
func init() {
	// Add subcommands to combCmd
	combCmd.AddCommand(combListCmd)
	combCmd.AddCommand(combValidateCmd)
	combCmd.AddCommand(combMergeCmd)
}

var combListCmd = &cobra.Command{
	Use:   "list",
	Short: "Display workers in a comb file",
	Long:  `List all workers (open redirect URLs) stored in a comb file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if combInputFile == "" {
			return fmt.Errorf("--file is required")
		}

		manager := comb.NewManager()
		if err := manager.LoadFromFile(combInputFile); err != nil {
			return fmt.Errorf("failed to load comb file: %w", err)
		}

		fmt.Printf("📋 Workers in %s:\n\n", combInputFile)
		for i, worker := range manager.Workers() {
			fmt.Printf("   %4d. %s\n", i+1, worker)
		}
		fmt.Printf("\n   Total: %d workers\n", manager.Count())

		return nil
	},
}

var combValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Test and filter working redirect URLs",
	Long: `Validate open redirect URLs by testing if they actually redirect.
Only URLs that successfully redirect to the test target are kept.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if combInputFile == "" {
			return fmt.Errorf("--input is required")
		}
		if combOutputFile == "" {
			return fmt.Errorf("--output is required")
		}
		if combTestTarget == "" {
			combTestTarget = "https://example.com"
		}

		// Load workers
		manager := comb.NewManager()
		if err := manager.LoadFromFile(combInputFile); err != nil {
			return fmt.Errorf("failed to load input file: %w", err)
		}

		fmt.Printf("🔍 Validating %d workers...\n", manager.Count())
		fmt.Printf("   Test target: %s\n", combTestTarget)
		fmt.Printf("   Concurrency: %d\n\n", combConcurrency)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Received interrupt, stopping validation...")
			cancel()
		}()

		validator := comb.NewValidator(combConcurrency, IsVerbose())
		results, stats := validator.ValidateWorkers(ctx, manager.Workers(), combTestTarget)

		// Save valid workers
		validWorkers := comb.ExtractValidWorkers(results)
		outputManager := comb.NewManager()
		outputManager.SetWorkers(validWorkers)

		if err := outputManager.SaveToFile(combOutputFile); err != nil {
			return fmt.Errorf("failed to save output file: %w", err)
		}

		fmt.Printf("\n📊 Validation Results:\n")
		fmt.Printf("   Total tested:  %d\n", stats.Total)
		fmt.Printf("   Valid:         %d (%.1f%%)\n", stats.Valid, float64(stats.Valid)/float64(stats.Total)*100)
		fmt.Printf("   Invalid:       %d\n", stats.Invalid)
		fmt.Printf("   Duration:      %s\n", stats.Duration.Round(100*1e6))
		fmt.Printf("   Saved to:      %s\n", combOutputFile)

		return nil
	},
}

var combMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Combine multiple comb files",
	Long:  `Merge multiple comb files into one, removing duplicates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("at least 2 input files required")
		}
		if combOutputFile == "" {
			return fmt.Errorf("--output is required")
		}

		merged := comb.NewManager()
		totalBefore := 0

		for _, inputFile := range args {
			manager := comb.NewManager()
			if err := manager.LoadFromFile(inputFile); err != nil {
				fmt.Printf("⚠️  Warning: failed to load %s: %v\n", inputFile, err)
				continue
			}
			totalBefore += manager.Count()
			added := merged.Merge(manager)
			fmt.Printf("   Loaded %d workers from %s (%d new)\n", manager.Count(), inputFile, added)
		}

		if err := merged.SaveToFile(combOutputFile); err != nil {
			return fmt.Errorf("failed to save output file: %w", err)
		}

		fmt.Printf("\n📊 Merge Results:\n")
		fmt.Printf("   Total loaded:     %d\n", totalBefore)
		fmt.Printf("   After dedup:      %d\n", merged.Count())
		fmt.Printf("   Duplicates found: %d\n", totalBefore-merged.Count())
		fmt.Printf("   Saved to:         %s\n", combOutputFile)

		return nil
	},
}

func init() {
	// List flags
	combListCmd.Flags().StringVarP(&combInputFile, "file", "f", "", "comb file to list (required)")

	// Validate flags
	combValidateCmd.Flags().StringVarP(&combInputFile, "input", "i", "", "input comb file (required)")
	combValidateCmd.Flags().StringVarP(&combOutputFile, "output", "o", "", "output file for valid workers (required)")
	combValidateCmd.Flags().IntVarP(&combConcurrency, "concurrency", "c", 50, "number of concurrent validators")
	combValidateCmd.Flags().StringVarP(&combTestTarget, "test-target", "t", "https://example.com", "URL to test redirects against")

	// Merge flags
	combMergeCmd.Flags().StringVarP(&combOutputFile, "output", "o", "", "output file for merged workers (required)")
}
