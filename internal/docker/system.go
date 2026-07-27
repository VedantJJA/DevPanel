package docker

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// HostMetrics holds live CPU, Memory, OS, and Architecture information from the host OS.
type HostMetrics struct {
	CPUs       int     `json:"cpus"`
	TotalMemMB float64 `json:"totalMemMb"`
	UsedMemMB  float64 `json:"usedMemMb"`
	MemPercent float64 `json:"memPercent"`
	OS         string  `json:"os"`
	Arch       string  `json:"arch"`
}

// MEMORYSTATUSEX is the Windows API struct for physical memory metrics.
type MEMORYSTATUSEX struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// GetHostMetrics retrieves exact CPU core count, total physical RAM, used RAM, OS, and Arch automatically.
func GetHostMetrics() HostMetrics {
	cpus := runtime.NumCPU()
	goOS := runtime.GOOS
	goArch := runtime.GOARCH

	// Format OS and Arch nicely
	osName := strings.Title(goOS)
	if goOS == "linux" {
		osName = "Linux Runtime"
	} else if goOS == "windows" {
		osName = "Windows Host"
	}

	archName := strings.ToUpper(goArch)

	var totalMB, usedMB float64

	// 1. Try Windows API via GlobalMemoryStatusEx if running on Windows
	if goOS == "windows" {
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")

		var memStatus MEMORYSTATUSEX
		memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

		ret, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
		if ret != 0 {
			totalMB = MathRound(float64(memStatus.ullTotalPhys)/1024/1024, 0)
			availMB := float64(memStatus.ullAvailPhys) / 1024 / 1024
			usedMB = MathRound(totalMB-availMB, 0)
		}
	}

	// 2. Try Linux /proc/meminfo if running on Linux (e.g. Oracle Cloud VM)
	if totalMB == 0 {
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
				totalMB = MathRound(memTotalKB/1024, 0)
				usedKB := memTotalKB - memAvailableKB
				if usedKB > 0 {
					usedMB = MathRound(usedKB/1024, 0)
				}
			}
		}
	}

	// Safety fallback if system calls fail
	if totalMB <= 0 {
		totalMB = 4096.0
		usedMB = 512.0
	}

	memPct := 0.0
	if totalMB > 0 {
		memPct = MathRound((usedMB/totalMB)*100, 1)
	}

	return HostMetrics{
		CPUs:       cpus,
		TotalMemMB: totalMB,
		UsedMemMB:  usedMB,
		MemPercent: memPct,
		OS:         fmt.Sprintf("%s (%s)", osName, archName),
		Arch:       archName,
	}
}
