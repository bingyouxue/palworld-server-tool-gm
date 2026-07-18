package api

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

const detachedProcess = 0x00000008

// launchServer starts the selected server binary with the window behaviour
// requested by the user. Cmd mode must be launched through the Windows shell:
// redirecting its standard handles would leave the new console permanently blank.
func launchServer(exePath, mode string) (string, error) {
	if mode == "cmd" {
		cmd := exec.Command("cmd.exe", "/C", "start", "PalServer Cmd", "/D", filepath.Dir(exePath), exePath)
		cmd.Dir = filepath.Dir(exePath)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("start %q in cmd mode: %w; output=%s", exePath, err, string(output))
		}
		return "", nil
	}

	logDir := filepath.Join(filepath.Dir(exePath), "PST-Logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("create server log directory %q: %w", logDir, err)
	}
	logPath := filepath.Join(logDir, "server-"+time.Now().Format("20060102-150405")+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("open server log %q: %w", logPath, err)
	}

	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	creationFlags := uint32(syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: creationFlags}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return logPath, fmt.Errorf("start %q in %s mode: %w", exePath, mode, err)
	}
	_ = logFile.Close()
	return logPath, nil
}
