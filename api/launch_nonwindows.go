//go:build !windows

package api

import (
	"os/exec"
	"path/filepath"
	"syscall"
)

// launchDetached starts the server executable in a new detached process.
func launchDetached(exePath string) error {
	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	return cmd.Start()
}
