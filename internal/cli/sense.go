package cli

import (
	"github.com/spf13/cobra"
)

// senseCmd represents the sense command group
var senseCmd = &cobra.Command{
	Use:   "sense",
	Short: "Reconnaissance tools to analyze targets",
	Long: `Sense commands provide reconnaissance capabilities.
Use these to analyze targets before engaging.

Available senses:
  scout - Find the largest assets on a target (best attack vectors)`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Sense subcommands will be added in Phase 5
}
