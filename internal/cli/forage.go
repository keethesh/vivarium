package cli

import (
	"github.com/spf13/cobra"
)

// forageCmd represents the forage command
var forageCmd = &cobra.Command{
	Use:   "forage",
	Short: "Discover new open redirect URLs via dorking",
	Long: `Forage scouts the web for vulnerable hosts.
Uses search engine dorking to find open redirect vulnerabilities
that can be added to your worker comb.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Forage implementation will be added in Phase 6
}
