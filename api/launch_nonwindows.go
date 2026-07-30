//go:build !windows

package api

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// launchServer starts the server executable in a new detached process.
func launchServer(exePath, _ string) (string, error) {
	logDir := filepath.Join(filepath.Dir(exePath), "PST-Logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("create server log directory %q: %w", logDir, err)
	}
	logPath := filepath.Join(logDir, "server-"+time.Now().Format("20060102-150405")+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("open server log %q: %w", logPath, err)
	}
	args := serverLaunchArgs()
	var cmd *exec.Cmd
	if filepath.Ext(exePath) == ".sh" {
		cmd = exec.Command("sh", append([]string{exePath}, args...)...)
	} else {
		cmd = exec.Command(exePath, args...)
	}
	cmd.Dir = filepath.Dir(exePath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return logPath, fmt.Errorf("start %q: %w", exePath, err)
	}
	_ = logFile.Close()
	return logPath, nil
}
