//go:build windows

package system

import (
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modpsapi                 = windows.NewLazySystemDLL("psapi.dll")
	procGetProcessMemoryInfo = modpsapi.NewProc("GetProcessMemoryInfo")
)

type processMemoryCounters struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

// cpuState holds the previous measurement for delta calculation.
var (
	cpuMu      sync.Mutex
	prevKernel int64
	prevUser   int64
	prevWall   int64
	cpuPct     float64
	cpuInited  bool
)

func filetime2int64(ft windows.Filetime) int64 {
	return int64(ft.HighDateTime)<<32 | int64(ft.LowDateTime)
}

// GetProcCPUPercent returns the CPU usage % of the current process
// normalised to a single core (0–100). First call always returns 0.
func GetProcCPUPercent() float64 {
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false,
		uint32(os.Getpid()),
	)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(handle)

	var creation, exit, kernel, user windows.Filetime
	err = windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user)
	if err != nil {
		return 0
	}

	k := filetime2int64(kernel)
	u := filetime2int64(user)
	now := time.Now().UnixNano() / 100 // 100ns units

	cpuMu.Lock()
	defer cpuMu.Unlock()

	if !cpuInited {
		prevKernel = k
		prevUser = u
		prevWall = now
		cpuInited = true
		return 0
	}

	wallDelta := now - prevWall
	if wallDelta <= 0 {
		return cpuPct
	}
	procDelta := (k - prevKernel) + (u - prevUser)
	pct := float64(procDelta) / float64(wallDelta) * 100.0 / float64(runtime.NumCPU())

	prevKernel = k
	prevUser = u
	prevWall = now
	cpuPct = pct
	return pct
}

// GetProcMemoryBytes returns the Working Set Size of the current process in bytes.
func GetProcMemoryBytes() uint64 {
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false,
		uint32(os.Getpid()),
	)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(handle)

	var mc processMemoryCounters
	mc.cb = uint32(unsafe.Sizeof(mc))
	r, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&mc)),
		uintptr(mc.cb),
	)
	if r == 0 {
		return 0
	}
	return uint64(mc.WorkingSetSize)
}

// GetTotalMemoryBytes returns total physical memory in bytes.
func GetTotalMemoryBytes() uint64 {
	type memStatusEx struct {
		dwLength                uint32
		dwMemoryLoad            uint32
		ullTotalPhys            uint64
		ullAvailPhys            uint64
		ullTotalPageFile        uint64
		ullAvailPageFile        uint64
		ullTotalVirtual         uint64
		ullAvailVirtual         uint64
		ullAvailExtendedVirtual uint64
	}
	procGlobalMemStatus := windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
	var ms memStatusEx
	ms.dwLength = uint32(unsafe.Sizeof(ms))
	procGlobalMemStatus.Call(uintptr(unsafe.Pointer(&ms)))
	return ms.ullTotalPhys
}
