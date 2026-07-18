//go:build !windows

package system

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type linuxCoreTime struct{ idle, total uint64 }

var (
	palResourceMu sync.Mutex
	palPrevProc   uint64
	palPrevWall   time.Time
	palPrevCores  []linuxCoreTime
)

func linuxPalProcesses() ([]string, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		if err != nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(string(comm)))
		if strings.Contains(name, "palserver") || strings.Contains(name, "palserver-linux-shipping") {
			paths = append(paths, filepath.Join("/proc", entry.Name()))
		}
	}
	return paths, nil
}

func linuxProcessResource(path string) (ticks, rss uint64, started uint64, ok bool) {
	stat, err := os.ReadFile(filepath.Join(path, "stat"))
	if err != nil {
		return 0, 0, 0, false
	}
	line := string(stat)
	closeParen := strings.LastIndex(line, ")")
	if closeParen < 0 {
		return 0, 0, 0, false
	}
	fields := strings.Fields(line[closeParen+2:])
	if len(fields) < 22 {
		return 0, 0, 0, false
	}
	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)
	started, _ = strconv.ParseUint(fields[19], 10, 64)
	status, _ := os.ReadFile(filepath.Join(path, "status"))
	for _, l := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(l, "VmRSS:") {
			parts := strings.Fields(l)
			if len(parts) >= 2 {
				kb, _ := strconv.ParseUint(parts[1], 10, 64)
				rss = kb * 1024
			}
			break
		}
	}
	return utime + stime, rss, started, true
}

func linuxCoreTimes() ([]linuxCoreTime, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil, err
	}
	var result []linuxCoreTime
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") || fields[0] == "cpu" {
			continue
		}
		if _, err := strconv.Atoi(strings.TrimPrefix(fields[0], "cpu")); err != nil {
			continue
		}
		var vals []uint64
		for _, f := range fields[1:] {
			v, _ := strconv.ParseUint(f, 10, 64)
			vals = append(vals, v)
		}
		var total uint64
		for _, v := range vals {
			total += v
		}
		idle := vals[3]
		if len(vals) > 4 {
			idle += vals[4]
		}
		result = append(result, linuxCoreTime{idle: idle, total: total})
	}
	return result, nil
}

// GetPalServerResourceSnapshot aggregates every PalServer image on the host.
func GetPalServerResourceSnapshot() (*ResourceSnapshot, error) {
	paths, err := linuxPalProcesses()
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	var ticks, memory, earliest uint64
	count := 0
	for _, path := range paths {
		cpu, mem, started, ok := linuxProcessResource(path)
		if !ok {
			continue
		}
		ticks += cpu
		memory += mem
		count++
		if earliest == 0 || started < earliest {
			earliest = started
		}
	}
	if count == 0 {
		return nil, nil
	}
	cores, err := linuxCoreTimes()
	if err != nil {
		return nil, err
	}
	result := &ResourceSnapshot{CPUPerCore: make([]float64, len(cores)), MemoryBytes: memory, MemoryTotal: GetTotalMemoryBytes(), CPUCores: runtime.NumCPU(), ProcessCount: count}
	if earliest > 0 {
		result.UptimeSeconds = int64(float64(procUptimeTicks()-earliest) / 100)
	}
	now := time.Now()
	palResourceMu.Lock()
	defer palResourceMu.Unlock()
	if !palPrevWall.IsZero() && ticks >= palPrevProc {
		seconds := now.Sub(palPrevWall).Seconds()
		if seconds > 0 {
			single := float64(ticks-palPrevProc) / 100 / seconds * 100
			total := single / float64(maxLinux(runtime.NumCPU(), 1))
			result.CPUPercent = &single
			result.CPUTotalPercent = &total
		}
	}
	if len(palPrevCores) == len(cores) {
		for i, cur := range cores {
			prev := palPrevCores[i]
			total := cur.total - prev.total
			idle := cur.idle - prev.idle
			if total > 0 {
				result.CPUPerCore[i] = float64(total-idle) / float64(total) * 100
			}
		}
	}
	palPrevProc = ticks
	palPrevWall = now
	palPrevCores = append(palPrevCores[:0], cores...)
	return result, nil
}

func procUptimeTicks() uint64 {
	data, _ := os.ReadFile("/proc/uptime")
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return 0
	}
	sec, _ := strconv.ParseFloat(parts[0], 64)
	return uint64(sec * 100)
}
func maxLinux(a, b int) int {
	if a > b {
		return a
	}
	return b
}
