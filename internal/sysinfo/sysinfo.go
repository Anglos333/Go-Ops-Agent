package sysinfo

import (
	"bytes"
	"fmt"
	stdnet "net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type Snapshot struct {
	CPUUsagePercent float64
	Load1           float64
	Load5           float64
	Load15          float64
	MemoryUsedMB    uint64
	MemoryTotalMB   uint64
	MemoryFreeMB    uint64
	DiskUsedGB      uint64
	DiskTotalGB     uint64
	DiskFreeGB      uint64
	PrimaryIP       string
	NetworkRxMB     uint64
	NetworkTxMB     uint64
	TopProcesses    []ProcessInfo
}

type ProcessInfo struct {
	PID    int32
	Name   string
	CPU    float64
	Memory uint64
}

func CollectSnapshot() (Snapshot, error) {
	result := Snapshot{}

	cpuPercent, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil {
		return result, err
	}
	if len(cpuPercent) > 0 {
		result.CPUUsagePercent = cpuPercent[0]
	}

	if avg, err := load.Avg(); err == nil {
		result.Load1 = avg.Load1
		result.Load5 = avg.Load5
		result.Load15 = avg.Load15
	}

	v, err := mem.VirtualMemory()
	if err != nil {
		return result, err
	}
	result.MemoryUsedMB = v.Used / 1024 / 1024
	result.MemoryTotalMB = v.Total / 1024 / 1024
	result.MemoryFreeMB = v.Available / 1024 / 1024

	if usage, err := disk.Usage("/"); err == nil {
		result.DiskUsedGB = usage.Used / 1024 / 1024 / 1024
		result.DiskTotalGB = usage.Total / 1024 / 1024 / 1024
		result.DiskFreeGB = usage.Free / 1024 / 1024 / 1024
	}

	result.PrimaryIP = detectPrimaryIP()
	if counters, err := psnet.IOCounters(false); err == nil && len(counters) > 0 {
		result.NetworkRxMB = counters[0].BytesRecv / 1024 / 1024
		result.NetworkTxMB = counters[0].BytesSent / 1024 / 1024
	}

	processes, err := process.Processes()
	if err != nil {
		return result, err
	}

	items := make([]ProcessInfo, 0, len(processes))
	for _, p := range processes {
		name, _ := p.Name()
		cpuValue, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		var rssMB uint64
		if memInfo != nil {
			rssMB = memInfo.RSS / 1024 / 1024
		}
		items = append(items, ProcessInfo{PID: p.Pid, Name: name, CPU: cpuValue, Memory: rssMB})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CPU == items[j].CPU {
			return items[i].Memory > items[j].Memory
		}
		return items[i].CPU > items[j].CPU
	})
	if len(items) > 5 {
		items = items[:5]
	}
	result.TopProcesses = items

	return result, nil
}

func (s Snapshot) Summary() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("CPU负载: %.2f%%\n", s.CPUUsagePercent))
	builder.WriteString(fmt.Sprintf("系统负载: load1=%.2f load5=%.2f load15=%.2f\n", s.Load1, s.Load5, s.Load15))
	builder.WriteString(fmt.Sprintf("内存: 已用 %d MB / 总计 %d MB / 可用 %d MB\n", s.MemoryUsedMB, s.MemoryTotalMB, s.MemoryFreeMB))
	builder.WriteString(fmt.Sprintf("磁盘(/): 已用 %d GB / 总计 %d GB / 可用 %d GB\n", s.DiskUsedGB, s.DiskTotalGB, s.DiskFreeGB))
	if strings.TrimSpace(s.PrimaryIP) != "" {
		builder.WriteString(fmt.Sprintf("主机地址: %s\n", s.PrimaryIP))
	}
	builder.WriteString(fmt.Sprintf("网络累计: 接收 %d MB / 发送 %d MB\n", s.NetworkRxMB, s.NetworkTxMB))
	builder.WriteString("Top 5 进程:\n")
	for _, p := range s.TopProcesses {
		builder.WriteString(fmt.Sprintf("- PID=%d Name=%s CPU=%.2f%% RSS=%dMB\n", p.PID, p.Name, p.CPU, p.Memory))
	}
	return strings.TrimSpace(builder.String())
}

func detectPrimaryIP() string {
	interfaces, err := stdnet.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if (iface.Flags&stdnet.FlagUp) == 0 || (iface.Flags&stdnet.FlagLoopback) != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*stdnet.IPNet)
			if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

func ReadRecentLogs(lines int) (string, error) {
	if runtime.GOOS != "linux" {
		return "当前运行环境不是 Linux，未采集系统日志。", nil
	}

	if data, err := tailFile("/var/log/syslog", lines); err == nil && strings.TrimSpace(data) != "" {
		return data, nil
	}

	cmd := exec.Command("journalctl", "-n", fmt.Sprintf("%d", lines), "--no-pager")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func FilterOOMLogs(logs string) string {
	if strings.TrimSpace(logs) == "" {
		return ""
	}
	keywords := []string{"oom", "out of memory", "killed process", "memory cgroup out of memory"}
	lines := strings.Split(logs, "\n")
	matched := make([]string, 0)
	for _, line := range lines {
		lower := strings.ToLower(line)
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				matched = append(matched, line)
				break
			}
		}
	}
	return strings.Join(matched, "\n")
}

func tailFile(path string, lines int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	parts := strings.Split(string(data), "\n")
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}
