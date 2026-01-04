package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"

	"vivarium/internal/common"
	"vivarium/internal/hexagon"
)

var (
	hexagonPort      int
	hexagonNoBrowser bool
)

func init() {
	rootCmd.AddCommand(hexagonCmd)

	hexagonCmd.Flags().IntVarP(&hexagonPort, "port", "p", 8666, "port to run the web server on")
	hexagonCmd.Flags().BoolVar(&hexagonNoBrowser, "no-browser", false, "don't auto-open browser")
}

var hexagonCmd = &cobra.Command{
	Use:   "hexagon",
	Short: "Start the Hexagon web GUI",
	Long: `Hexagon is the visual command center for Vivarium.
It provides a web-based dashboard for launching attacks,
monitoring progress in real-time, and managing workers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := common.RequireAuthorization(GetConfigPath()); err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("🔷 HEXAGON - Visual Command Center")
		fmt.Printf("   Starting server on http://localhost:%d\n", hexagonPort)
		fmt.Println()

		server := hexagon.NewServer(hexagonPort)

		// Handle graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\n🛑 Shutting down Hexagon...")
			cancel()
			server.Stop(ctx)
		}()

		// Open browser unless disabled
		if !hexagonNoBrowser {
			go openBrowser(fmt.Sprintf("http://localhost:%d", hexagonPort))
		}

		return server.Start()
	},
}

// openBrowser opens the default browser to the given URL.
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, etc.
		cmd = exec.Command("xdg-open", url)
	}

	cmd.Start()
}
