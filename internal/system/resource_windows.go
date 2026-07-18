//go:build windows

package system

import (
	"errors"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	resourceMu       sync.Mutex
	resourcePrevProc uint64
	resourcePrevWall time.Time
	resourcePrevCore []processorPerformance
)

type processorPerformance struct {
	IdleTime       int64
	KernelTime     int64
	UserTime       int64
	DpcTime        int64
	InterruptTime  int64
	InterruptCount uint32
	_              uint32
}

var (
	modntdll                     = windows.NewLazySystemDLL("ntdll.dll")
	procNtQuerySystemInformation = modntdll.NewProc("NtQuerySystemInformation")
)

func palProcessName(name string) bool {
	n := strings.ToLower(name)
	return n == "palserver.exe" || n == "palserver-win64-shipping.exe" || n == "palserver-win64-shipping-cmd.exe"
}

func listPalProcessIDs() ([]uint32, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err = windows.Process32First(snapshot, &entry); err != nil {
		return nil, err
	}
	var result []uint32
	for {
		if palProcessName(windows.UTF16ToString(entry.ExeFile[:])) {
			result = append(result, entry.ProcessID)
		}
		if err = windows.Process32Next(snapshot, &entry); err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				break
			}
			return nil, err
		}
	}
	return result, nil
}

func queryCorePerformance() ([]processorPerformance, error) {
	cores := runtime.NumCPU()
	values := make([]processorPerformance, cores)
	if cores == 0 {
		return values, nil
	}
	status, _, _ := procNtQuerySystemInformation.Call(
		8,
		uintptr(unsafe.Pointer(&values[0])),
		uintptr(len(values))*unsafe.Sizeof(values[0]),
		0,
	)
	if int32(status) < 0 {
		return nil, errors.New("NtQuerySystemInformation(SystemProcessorPerformanceInformation) failed")
	}
	return values, nil
}

func processResource(pid uint32) (cpu uint64, memory uint64, created int64, ok bool) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return 0, 0, 0, false
	}
	defer windows.CloseHandle(handle)
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return 0, 0, 0, false
	}
	var mc processMemoryCounters
	mc.cb = uint32(unsafe.Sizeof(mc))
	r, _, _ := procGetProcessMemoryInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&mc)), uintptr(mc.cb))
	if r == 0 {
		return 0, 0, 0, false
	}
	return uint64(filetime2int64(kernel) + filetime2int64(user)), uint64(mc.WorkingSetSize), filetime2int64(creation), true
}

// GetPalServerResourceSnapshot aggregates every PalServer image on the host.
func GetPalServerResourceSnapshot() (*ResourceSnapshot, error) {
	pids, err := listPalProcessIDs()
	if err != nil {
		return nil, err
	}
	if len(pids) == 0 {
		return nil, nil
	}
	var procTime, memory uint64
	var earliest int64
	count := 0
	for _, pid := range pids {
		cpu, mem, created, ok := processResource(pid)
		if !ok {
			continue
		}
		procTime += cpu
		memory += mem
		count++
		if earliest == 0 || created < earliest {
			earliest = created
		}
	}
	if count == 0 {
		return nil, nil
	}
	cores, err := queryCorePerformance()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	result := &ResourceSnapshot{
		CPUPerCore:   make([]float64, len(cores)),
		MemoryBytes:  memory,
		MemoryTotal:  GetTotalMemoryBytes(),
		CPUCores:     len(cores),
		ProcessCount: count,
	}
	if earliest > 0 {
		createdAt := time.Unix(0, (earliest-116444736000000000)*100)
		result.UptimeSeconds = int64(time.Since(createdAt).Seconds())
	}
	resourceMu.Lock()
	defer resourceMu.Unlock()
	if !resourcePrevWall.IsZero() && procTime >= resourcePrevProc {
		wall100ns := now.Sub(resourcePrevWall).Nanoseconds() / 100
		if wall100ns > 0 {
			single := float64(procTime-resourcePrevProc) / float64(wall100ns) * 100
			total := single / float64(maxInt(len(cores), 1))
			result.CPUPercent = &single
			result.CPUTotalPercent = &total
		}
	}
	if len(resourcePrevCore) == len(cores) {
		for i, cur := range cores {
			prev := resourcePrevCore[i]
			total := (cur.KernelTime - prev.KernelTime) + (cur.UserTime - prev.UserTime)
			idle := cur.IdleTime - prev.IdleTime
			if total > 0 {
				pct := float64(total-idle) / float64(total) * 100
				if pct < 0 {
					pct = 0
				}
				if pct > 100 {
					pct = 100
				}
				result.CPUPerCore[i] = pct
			}
		}
	}
	resourcePrevProc = procTime
	resourcePrevWall = now
	resourcePrevCore = append(resourcePrevCore[:0], cores...)
	return result, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
