// Package cli implements the Vivarium command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const banner = `
██╗   ██╗██╗██╗   ██╗ █████╗ ██████╗ ██╗██╗   ██╗███╗   ███╗
██║   ██║██║██║   ██║██╔══██╗██╔══██╗██║██║   ██║████╗ ████║
██║   ██║██║██║   ██║███████║██████╔╝██║██║   ██║██╔████╔██║
╚██╗ ██╔╝██║╚██╗ ██╔╝██╔══██║██╔══██╗██║██║   ██║██║╚██╔╝██║
 ╚████╔╝ ██║ ╚████╔╝ ██║  ██║██║  ██║██║╚██████╔╝██║ ╚═╝ ██║
  ╚═══╝  ╚═╝  ╚═══╝  ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝ ╚═════╝ ╚═╝     ╚═╝
                                                     v0.1.0
    "The ecosystem is the weapon. Resistance is organic failure."
`

var (
	cfgFile       string
	verbose       bool
	hasPermission bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "vivarium",
	Short: "A hive-mind network stress testing toolkit",
	Long: banner + `
VIVARIUM is a modern, high-performance network stress testing toolkit
built with a biological "Hive" architecture. It treats distributed
computing as a living, breathing swarm.

WARNING: This tool is for educational purposes and authorized testing only.
You must have explicit permission to test any target system.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip permission check for help commands
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" {
			return nil
		}
		// Also skip for parent commands (sting, swarm, etc. when showing help)
		if !cmd.HasParent() || cmd.Parent().Name() == "vivarium" {
			// This is a top-level subcommand, check if it's being run with subcommands
			if len(args) == 0 && cmd.Name() != "vivarium" {
				return nil // Let it show help
			}
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./vivarium.toml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&hasPermission, "i-have-permission", false, "confirm you have authorization to test the target")

	// Add subcommands
	rootCmd.AddCommand(stingCmd)
	rootCmd.AddCommand(swarmCmd)
	rootCmd.AddCommand(senseCmd)
	rootCmd.AddCommand(combCmd)
	rootCmd.AddCommand(forageCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if cfgFile != "" {
		// TODO: Load config from file
		if verbose {
			fmt.Fprintf(os.Stderr, "Using config file: %s\n", cfgFile)
		}
	}
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Vivarium",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Vivarium v0.1.0 - The Swarm Engine")
		fmt.Println("Built with Go 1.25")
	},
}

// HasPermission returns whether the user has confirmed authorization
func HasPermission() bool {
	return hasPermission
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}
