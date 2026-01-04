// Package main is the entry point for the Vivarium CLI.
package main

import (
	"os"

	"vivarium/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
