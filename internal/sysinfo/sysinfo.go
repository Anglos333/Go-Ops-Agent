package sysinfo

type Snapshot struct {
	CPUUsagePercent float64
	MemoryUsedMB    uint64
	MemoryTotalMB   uint64
	TopProcesses    []ProcessInfo
}

type ProcessInfo struct {
	PID    int32
	Name   string
	CPU    float64
	Memory uint64
}
