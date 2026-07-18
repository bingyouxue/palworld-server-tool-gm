//go:build !windows

package system

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	cpuMu     sync.Mutex
	prevIdle  uint64
	prevTotal uint64
	cpuPct    float64
)

// GetProcCPUPercent returns an approximation of CPU usage for the current
// process on Linux by reading /proc/self/stat.
func GetProcCPUPercent() float64 {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0
	}
	utime, _ := strconv.ParseUint(fields[13], 10, 64)
	stime, _ := strconv.ParseUint(fields[14], 10, 64)
	proc := utime + stime

	total := procTotalCPU()
	cpuMu.Lock()
	defer cpuMu.Unlock()
	if prevTotal == 0 {
		prevTotal = total
		prevIdle = proc
		return 0
	}
	totalDelta := total - prevTotal
	procDelta := proc - prevIdle
	prevTotal = total
	prevIdle = proc
	if totalDelta == 0 {
		return cpuPct
	}
	cpuPct = float64(procDelta) / float64(totalDelta) * 100.0 / float64(runtime.NumCPU())
	return cpuPct
}

func procTotalCPU() uint64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return uint64(time.Now().UnixNano())
	}
	line := strings.SplitN(string(data), "\n", 2)[0]
	fields := strings.Fields(line)
	var total uint64
	for _, f := range fields[1:] {
		v, _ := strconv.ParseUint(f, 10, 64)
		total += v
	}
	return total
}

// GetProcMemoryBytes returns the RSS (resident set size) of the current
// process in bytes by reading /proc/self/status.
func GetProcMemoryBytes() uint64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseUint(fields[1], 10, 64)
				return kb * 1024
			}
		}
	}
	return 0
}

// GetTotalMemoryBytes returns total physical memory in bytes by reading
// /proc/meminfo.
func GetTotalMemoryBytes() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseUint(fields[1], 10, 64)
				return kb * 1024
			}
		}
	}
	return 0
}
