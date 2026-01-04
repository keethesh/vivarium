// Package common provides shared utilities for Vivarium.
package common

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"vivarium/internal/config"
)

// ErrNotAuthorized is returned when the user hasn't configured authorization.
var ErrNotAuthorized = errors.New("not authorized")

const firstLaunchWarning = `
╔══════════════════════════════════════════════════════════════════════════════╗
║                            ⚠️  FIRST LAUNCH WARNING                          ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  VIVARIUM is a powerful network stress testing toolkit.                      ║
║                                                                              ║
║  This tool is ONLY for:                                                      ║
║    • Educational purposes                                                    ║
║    • Authorized penetration testing                                          ║
║    • Stress testing systems you OWN or have WRITTEN PERMISSION to test       ║
║                                                                              ║
║  Using this tool without authorization may be ILLEGAL in your jurisdiction.  ║
║                                                                              ║
║  You are solely responsible for ensuring you have proper authorization.      ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝

A configuration file will be created at:
  %s

To enable attack features, edit the config file and set:
  authorized = true

`

const notAuthorizedMessage = `
⚠️  NOT AUTHORIZED

Attack features are disabled. To enable them:

1. Edit your config file at:
   %s

2. Set authorized = true

This confirms you understand this tool is for authorized testing only.
`

// CheckAuthorization checks if the user has authorized use of the tool.
// On first launch, it shows a warning and creates a config file.
func CheckAuthorization(cfgPath string) error {
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	// Check if config exists
	if !config.Exists(cfgPath) {
		// First launch - show warning
		fmt.Printf(firstLaunchWarning, cfgPath)

		// Prompt user
		fmt.Print("Press Enter to create the config file and continue...")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')

		// Create default config
		if err := config.CreateDefault(cfgPath); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}

		fmt.Printf("\n✓ Config created at: %s\n", cfgPath)
		fmt.Println("  Edit the file and set 'authorized = true' to enable attack features.\n")

		return ErrNotAuthorized
	}

	// Load config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.Authorized {
		fmt.Printf(notAuthorizedMessage, cfgPath)
		return ErrNotAuthorized
	}

	return nil
}

// RequireAuthorization checks authorization and returns an error if not authorized.
func RequireAuthorization(cfgPath string) error {
	return CheckAuthorization(cfgPath)
}

// IsAuthorized returns true if the user has authorized use.
func IsAuthorized(cfgPath string) bool {
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return false
	}
	return cfg.Authorized
}

// PromptConfirmation asks the user to confirm an action.
func PromptConfirmation(message string) bool {
	fmt.Printf("%s [y/N]: ", message)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
