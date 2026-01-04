// Package cli implements the Vivarium command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"vivarium/internal/config"
)

const banner = `
██╗   ██╗██╗██╗   ██╗ █████╗ ██████╗ ██╗██╗   ██╗███╗   ███╗
██║   ██║██║██║   ██║██╔══██╗██╔══██╗██║██║   ██║████╗ ████║
██║   ██║██║██║   ██║███████║██████╔╝██║██║   ██║██╔████╔██║
╚██╗ ██╔╝██║╚██╗ ██╔╝██╔══██║██╔══██╗██║██║   ██║██║╚██╔╝██║
 ╚████╔╝ ██║ ╚████╔╝ ██║  ██║██║  ██║██║╚██████╔╝██║ ╚═╝ ██║
  ╚═══╝  ╚═╝  ╚═══╝  ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝ ╚═════╝ ╚═╝     ╚═╝
                                                     v0.6.0
    "The ecosystem is the weapon. Resistance is organic failure."
`

var (
	cfgFile string
	verbose bool
	cfg     *config.Config
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
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.vivarium/config.toml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(stingCmd)
	rootCmd.AddCommand(swarmCmd)
	rootCmd.AddCommand(senseCmd)
	rootCmd.AddCommand(combCmd)
	rootCmd.AddCommand(forageCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
}

func initConfig() {
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		cfg = &config.Config{}
	}

	// Merge verbose flag from config if not set via CLI
	if cfg.Verbose && !verbose {
		verbose = true
	}
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Vivarium",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Vivarium v0.6.0 - The Swarm Engine")
		fmt.Println("Built with Go 1.25")
	},
}

// configCmd shows config info
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show configuration information",
	Run: func(cmd *cobra.Command, args []string) {
		path := cfgFile
		if path == "" {
			path = config.DefaultConfigPath()
		}

		fmt.Println("📋 Configuration")
		fmt.Printf("   Path: %s\n", path)
		fmt.Printf("   Exists: %v\n", config.Exists(path))

		if cfg != nil {
			fmt.Printf("   Authorized: %v\n", cfg.Authorized)
			fmt.Printf("   Verbose: %v\n", cfg.Verbose)
		}
	},
}

// GetConfig returns the loaded configuration.
func GetConfig() *config.Config {
	return cfg
}

// GetConfigPath returns the config file path.
func GetConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return config.DefaultConfigPath()
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}
