package cli

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"vivarium/internal/common"
	"vivarium/internal/smoke"
)

func init() {
	rootCmd.AddCommand(smokeCmd)
	smokeCmd.AddCommand(smokeTestCmd)
	smokeCmd.AddCommand(smokeIPCmd)
	smokeCmd.AddCommand(smokeRotateCmd)

	// Test flags
	smokeTestCmd.Flags().StringP("proxy", "p", "socks5://127.0.0.1:9050", "SOCKS5 proxy URL")

	// IP flags
	smokeIPCmd.Flags().StringP("proxy", "p", "socks5://127.0.0.1:9050", "SOCKS5 proxy URL")
}

var smokeCmd = &cobra.Command{
	Use:   "smoke",
	Short: "Anonymity layer for proxying traffic through Tor/SOCKS5",
	Long: `Smoke provides anonymity by routing traffic through proxies.
Like beekeepers use smoke to calm and obscure, this layer hides your origin.

The default Tor proxy is socks5://127.0.0.1:9050
Make sure Tor is running before using smoke features.

For attacks, use the global --proxy flag:
  vivarium --proxy socks5://127.0.0.1:9050 sting locust -t http://target.com

Available commands:
  test   - Test proxy connectivity
  ip     - Show your external IP through proxy
  rotate - Request a new Tor circuit`,
}

var smokeTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test proxy connection",
	Long:  `Tests that the proxy is working correctly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyURL, _ := cmd.Flags().GetString("proxy")

		fmt.Println("\n💨 SMOKE - Testing proxy connection...")
		fmt.Printf("   Proxy: %s\n", proxyURL)
		fmt.Println()

		config := &smoke.Config{
			Enabled:  true,
			ProxyURL: proxyURL,
			Timeout:  10 * time.Second,
		}

		smoker, err := smoke.NewSmoker(config)
		if err != nil {
			return fmt.Errorf("failed to create smoker: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Print("   Testing connection... ")
		if err := smoker.TestConnection(ctx); err != nil {
			fmt.Println("❌ FAILED")
			return fmt.Errorf("proxy test failed: %w", err)
		}
		fmt.Println("✅ SUCCESS")

		fmt.Print("   Getting external IP... ")
		ip, err := smoker.GetExternalIP(ctx)
		if err != nil {
			fmt.Println("❌ FAILED")
			return fmt.Errorf("failed to get IP: %w", err)
		}
		fmt.Printf("✅ %s\n", ip)

		fmt.Println("\n   Proxy is working correctly!")
		return nil
	},
}

var smokeIPCmd = &cobra.Command{
	Use:   "ip",
	Short: "Show your external IP through the proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyURL, _ := cmd.Flags().GetString("proxy")

		config := &smoke.Config{
			Enabled:  true,
			ProxyURL: proxyURL,
			Timeout:  10 * time.Second,
		}

		smoker, err := smoke.NewSmoker(config)
		if err != nil {
			return fmt.Errorf("failed to create smoker: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get IP through proxy
		proxyIP, err := smoker.GetExternalIP(ctx)
		if err != nil {
			return fmt.Errorf("failed to get proxy IP: %w", err)
		}

		// Get IP directly
		directClient := &http.Client{Timeout: 10 * time.Second}
		resp, err := directClient.Get("https://api.ipify.org")
		directIP := "unknown"
		if err == nil {
			buf := make([]byte, 64)
			n, _ := resp.Body.Read(buf)
			directIP = string(buf[:n])
			resp.Body.Close()
		}

		fmt.Println("\n💨 SMOKE - IP Check")
		fmt.Printf("   Direct IP:  %s\n", directIP)
		fmt.Printf("   Proxy IP:   %s\n", proxyIP)

		if directIP != proxyIP {
			fmt.Println("\n   ✅ Your IP is masked!")
		} else {
			fmt.Println("\n   ⚠️  IPs match - proxy may not be working")
		}

		return nil
	},
}

var smokeRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Request a new Tor circuit",
	Long:  `Sends a signal to Tor to use a new exit node.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("\n💨 SMOKE - Rotating Tor circuit...")

		config := smoke.DefaultTorConfig()
		smoker, err := smoke.NewSmoker(config)
		if err != nil {
			return fmt.Errorf("failed to create smoker: %w", err)
		}

		if err := smoker.RotateCircuit(); err != nil {
			fmt.Println("   ⚠️  Could not rotate circuit (is Tor control port open?)")
			fmt.Println("   Make sure Tor is running with ControlPort 9051")
			return nil
		}

		fmt.Println("   ✅ New circuit requested!")
		fmt.Println("   Note: It may take a few seconds for the new circuit to be ready.")

		return nil
	},
}

// SetupProxy configures global proxy from the --proxy flag.
// This should be called early in command execution.
func SetupProxy(proxyURL string) error {
	if proxyURL == "" {
		return nil
	}

	fmt.Printf("💨 Using proxy: %s\n", proxyURL)
	return common.SetGlobalProxy(proxyURL)
}
