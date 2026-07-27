package docker

import (
	"fmt"
	"runtime"
	"strings"
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

	// Calls platform-specific implementation (system_windows.go or system_other.go)
	totalMB, usedMB := getPlatformMemory()

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
