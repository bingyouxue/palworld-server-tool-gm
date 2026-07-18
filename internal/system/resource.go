package system

// ResourceSnapshot describes the live PalServer process tree and host CPU state.
type ResourceSnapshot struct {
	CPUPercent      *float64
	CPUTotalPercent *float64
	CPUPerCore      []float64
	MemoryBytes     uint64
	MemoryTotal     uint64
	CPUCores        int
	ProcessCount    int
	UptimeSeconds   int64
}
