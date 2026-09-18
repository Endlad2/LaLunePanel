// Package sysinfo reports host load, memory and uptime for the dashboard.
package sysinfo

import (
	"os"
	"strconv"
	"strings"
	"time"
)

var startTime = time.Now()

type Info struct {
	Load1     float64 `json:"load1"`
	Load5     float64 `json:"load5"`
	Load15    float64 `json:"load15"`
	MemTotal  uint64  `json:"mem_total_kb"`
	MemFree   uint64  `json:"mem_free_kb"`
	MemUsed   uint64  `json:"mem_used_kb"`
	CPUCores  int     `json:"cpu_cores"`
	Hostname  string  `json:"hostname"`
}

// Host reads current host metrics from /proc (Linux).
func Host() Info {
	info := Info{
		Hostname: hostname(),
		CPUCores: cpuCores(),
	}

	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			info.Load1, _ = strconv.ParseFloat(fields[0], 64)
			info.Load5, _ = strconv.ParseFloat(fields[1], 64)
			info.Load15, _ = strconv.ParseFloat(fields[2], 64)
		}
	}

	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			val, _ := strconv.ParseUint(fields[1], 10, 64)
			switch fields[0] {
			case "MemTotal:":
				info.MemTotal = val
			case "MemFree:":
				info.MemFree = val
			}
		}
		info.MemUsed = info.MemTotal - info.MemFree
	}

	return info
}

// UptimeSeconds returns how long the panel process has been running.
func UptimeSeconds() int64 {
	return int64(time.Since(startTime).Seconds())
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func cpuCores() int {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	n := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "processor") {
			n++
		}
	}
	return n
}
