package cli

import (
	"github.com/spf13/cobra"
)

// combCmd represents the comb command group
var combCmd = &cobra.Command{
	Use:   "comb",
	Short: "Manage worker lists (open redirect URLs)",
	Long: `Comb commands manage the worker list - your army of open redirect URLs.
These URLs form the backbone of distributed swarm attacks.

Available commands:
  list     - Display workers in a comb file
  validate - Test and filter working redirect URLs
  merge    - Combine multiple comb files`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Comb subcommands will be added in Phase 3
}
