package sysinfo

import (
	"strings"
	"testing"
)

func TestSnapshotSummaryIncludesExtendedMetrics(t *testing.T) {
	snapshot := Snapshot{
		CPUUsagePercent: 91.2,
		Load1:           2.3,
		Load5:           1.8,
		Load15:          1.2,
		MemoryUsedMB:    7000,
		MemoryTotalMB:   8192,
		MemoryFreeMB:    512,
		DiskUsedGB:      88,
		DiskTotalGB:     100,
		DiskFreeGB:      12,
		PrimaryIP:       "192.168.1.10",
		NetworkRxMB:     2048,
		NetworkTxMB:     1024,
		TopProcesses:    []ProcessInfo{{PID: 1234, Name: "java", CPU: 80.5, Memory: 2048}},
	}

	summary := snapshot.Summary()
	checks := []string{"系统负载:", "磁盘(/):", "主机地址:", "网络累计:", "PID=1234 Name=java"}
	for _, check := range checks {
		if !strings.Contains(summary, check) {
			t.Fatalf("expected summary to contain %q, got %q", check, summary)
		}
	}
}
