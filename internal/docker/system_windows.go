//go:build windows

package docker

import (
	"syscall"
	"unsafe"
)

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

type FILETIME struct {
	dwLowDateTime  uint32
	dwHighDateTime uint32
}

var (
	lastIdleTime   uint64
	lastKernelTime uint64
	lastUserTime   uint64
)

func fileTimeToUint64(ft FILETIME) uint64 {
	return (uint64(ft.dwHighDateTime) << 32) | uint64(ft.dwLowDateTime)
}

func getPlatformMemory() (float64, float64) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")

	var memStatus MEMORYSTATUSEX
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

	ret, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret != 0 {
		totalMB := MathRound(float64(memStatus.ullTotalPhys)/1024/1024, 0)
		availMB := float64(memStatus.ullAvailPhys) / 1024 / 1024
		usedMB := MathRound(totalMB-availMB, 0)
		return totalMB, usedMB
	}

	return 4096.0, 512.0
}

func getPlatformCPULoad(numCPUs int) float64 {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getSystemTimes := kernel32.NewProc("GetSystemTimes")

	var idleTime, kernelTime, userTime FILETIME
	ret, _, _ := getSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if ret == 0 {
		return 0.0
	}

	idle := fileTimeToUint64(idleTime)
	kernel := fileTimeToUint64(kernelTime)
	user := fileTimeToUint64(userTime)

	if lastKernelTime == 0 {
		lastIdleTime = idle
		lastKernelTime = kernel
		lastUserTime = user
		return 0.0
	}

	usrDelta := user - lastUserTime
	kerDelta := kernel - lastKernelTime
	idlDelta := idle - lastIdleTime

	lastIdleTime = idle
	lastKernelTime = kernel
	lastUserTime = user

	totalDelta := usrDelta + kerDelta
	if totalDelta == 0 {
		return 0.0
	}

	activeDelta := totalDelta - idlDelta
	// Multiplied by numCPUs so 2 cores at 100% = 200%!
	cpuPct := (float64(activeDelta) / float64(totalDelta)) * 100.0 * float64(numCPUs)
	if cpuPct < 0 {
		cpuPct = 0
	}
	return MathRound(cpuPct, 1)
}
