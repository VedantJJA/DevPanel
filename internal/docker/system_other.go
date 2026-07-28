//go:build !windows

package docker

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

var (
	lastStatTotal uint64
	lastStatIdle  uint64
)

func getPlatformMemory() (float64, float64) {
	// Try reading Linux /proc/meminfo (Linux / Oracle Cloud VM)
	if f, err := os.Open("/proc/meminfo"); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		var memTotalKB, memAvailableKB float64

		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				key := fields[0]
				val, _ := strconv.ParseFloat(fields[1], 64)
				if key == "MemTotal:" {
					memTotalKB = val
				} else if key == "MemAvailable:" {
					memAvailableKB = val
				}
			}
		}

		if memTotalKB > 0 {
			totalMB := MathRound(memTotalKB/1024, 0)
			usedKB := memTotalKB - memAvailableKB
			usedMB := 0.0
			if usedKB > 0 {
				usedMB = MathRound(usedKB/1024, 0)
			}
			return totalMB, usedMB
		}
	}

	// Fallback for other Unix environments if /proc/meminfo is unreadable
	return 6144.0, 1024.0
}

func getPlatformCPULoad(numCPUs int) float64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0.0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[0] == "cpu" {
			var user, nice, sys, idle, iowait, irq, softirq, steal uint64
			user, _ = strconv.ParseUint(fields[1], 10, 64)
			nice, _ = strconv.ParseUint(fields[2], 10, 64)
			sys, _ = strconv.ParseUint(fields[3], 10, 64)
			idle, _ = strconv.ParseUint(fields[4], 10, 64)
			if len(fields) > 5 {
				iowait, _ = strconv.ParseUint(fields[5], 10, 64)
			}
			if len(fields) > 6 {
				irq, _ = strconv.ParseUint(fields[6], 10, 64)
			}
			if len(fields) > 7 {
				softirq, _ = strconv.ParseUint(fields[7], 10, 64)
			}
			if len(fields) > 8 {
				steal, _ = strconv.ParseUint(fields[8], 10, 64)
			}

			total := user + nice + sys + idle + iowait + irq + softirq + steal
			idleTotal := idle + iowait

			if lastStatTotal == 0 {
				lastStatTotal = total
				lastStatIdle = idleTotal
				return 0.0
			}

			totalDelta := total - lastStatTotal
			idleDelta := idleTotal - lastStatIdle

			lastStatTotal = total
			lastStatIdle = idleTotal

			if totalDelta == 0 {
				return 0.0
			}

			activeDelta := totalDelta - idleDelta
			// Multiplied by numCPUs so 2 cores at 100% = 200%!
			cpuPct := (float64(activeDelta) / float64(totalDelta)) * 100.0 * float64(numCPUs)
			if cpuPct < 0 {
				cpuPct = 0
			}
			return MathRound(cpuPct, 1)
		}
	}
	return 0.0
}
