package api

import (
	"os/exec"
	"path/filepath"
	"syscall"
)

// launchDetached starts the server executable in a new detached process so
// that pst does not hold a reference to it. On Windows we use CREATE_NEW_PROCESS_GROUP
// + DETACHED_PROCESS so the child survives if pst exits.
func launchDetached(exePath string) error {
	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008, // DETACHED_PROCESS
	}
	return cmd.Start()
}
