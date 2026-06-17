//go:build linux

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// launchDetached starts a fresh clipd process fully detached from the current
// one (new session), so it keeps running after this CLI invocation exits.
func launchDetached() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
