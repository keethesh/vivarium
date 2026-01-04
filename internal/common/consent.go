// Package common provides shared utilities for Vivarium.
package common

import (
	"errors"
	"fmt"
)

// ErrNoPermission is returned when the user hasn't confirmed authorization.
var ErrNoPermission = errors.New("permission not granted")

// RequirePermission checks if the user has confirmed they have authorization
// to test the target. This is an ethical safeguard.
func RequirePermission(hasPermission bool) error {
	if !hasPermission {
		fmt.Println(`
⚠️  AUTHORIZATION REQUIRED

You must confirm that you have explicit permission to test the target system.

This tool is for:
  • Educational purposes
  • Authorized penetration testing
  • Stress testing systems you own or have written permission to test

To proceed, add the --i-have-permission flag:

  vivarium sting locust --target <url> --i-have-permission

Using this tool without authorization may be illegal in your jurisdiction.
`)
		return ErrNoPermission
	}
	return nil
}
