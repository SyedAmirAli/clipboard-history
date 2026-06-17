//go:build windows

package main

import (
	"os"
	"os/exec"
)

// launchDetached starts a fresh clipd process detached from the current one.
// On Windows, we just start the process without special session handling.
func launchDetached() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
