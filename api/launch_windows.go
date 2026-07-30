package api

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const detachedProcess = 0x00000008

var (
	modShell32      = syscall.NewLazyDLL("shell32.dll")
	procShellExecEx = modShell32.NewProc("ShellExecuteExW")
)

// shellExecuteInfo mirrors the SHELLEXECUTEINFOW structure used by ShellExecuteExW.
// Only the fields we need are populated; the rest are zeroed.
type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    uintptr
	dwHotKey     uint32
	hIcon        uintptr
	hProcess     uintptr
}

const (
	seeMaskNoCloseProcess = 0x00000040
	swShowDefault         = 10 // SW_SHOWDEFAULT
)

// shellLaunch opens exePath exactly as if the user double-clicked it, giving
// the process its own visible console with all output attached.
func shellLaunch(exePath string, args []string) error {
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(exePath)
	dir, _ := syscall.UTF16PtrFromString(filepath.Dir(exePath))

	sei := shellExecuteInfo{
		fMask:       seeMaskNoCloseProcess,
		lpVerb:      verb,
		lpFile:      file,
		lpDirectory: dir,
		nShow:       swShowDefault,
	}
	if len(args) > 0 {
		params, err := syscall.UTF16PtrFromString(strings.Join(args, " "))
		if err != nil {
			return fmt.Errorf("encode launch arguments %v: %w", args, err)
		}
		sei.lpParameters = params
	}
	sei.cbSize = uint32(unsafe.Sizeof(sei))

	r, _, err := procShellExecEx.Call(uintptr(unsafe.Pointer(&sei)))
	if r == 0 {
		return fmt.Errorf("ShellExecuteExW failed: %w", err)
	}
	return nil
}

// launchServer starts the selected server binary with the window behaviour
// requested by the user.
func launchServer(exePath, mode string) (string, error) {
	args := serverLaunchArgs()
	if mode == "cmd" {
		// Use ShellExecuteExW so Windows launches the exe exactly as if the
		// user double-clicked it: the process gets its own console window and
		// all stdout/stderr output is attached to that window from the start.
		// exec.Command with CREATE_NEW_CONSOLE does not wire up the standard
		// handles correctly, leaving the window blank.
		if err := shellLaunch(exePath, args); err != nil {
			return "", fmt.Errorf("start %q in cmd mode: %w", exePath, err)
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

	cmd := exec.Command(exePath, args...)
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
