//go:build !windows

package docker

import (
	"bufio"
	"os"
	"strconv"
	"strings"
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
