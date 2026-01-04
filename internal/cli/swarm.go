package cli

import (
	"github.com/spf13/cobra"
)

// swarmCmd represents the swarm command group
var swarmCmd = &cobra.Command{
	Use:   "swarm",
	Short: "Execute distributed DDoS attacks via open redirects",
	Long: `Swarm attacks mobilize the full colony.
These are distributed attacks that use open redirect vulnerabilities
to amplify and mask the origin of requests.

Requires a worker list (comb) of validated open redirect URLs.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Swarm subcommands will be added in Phase 4
}
